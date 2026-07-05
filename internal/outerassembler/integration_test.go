package outerassembler_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"io"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/outerassembler"
	"github.com/arcavenae/switchboard/internal/routing"
)

// composedFixture wires up an admitted node + router + forwarding entries so a
// wire frame emitted by outerassembler.Assemble is verifiable end-to-end.
//
// The setup mirrors admittedRouterSetup in routing_test.go: derive a node keypair
// from a deterministic seed, complete admission.GenerateChallenge/AdmitNode so
// IsAdmitted returns true, derive srcAddr the same way DeriveNodeAddress does
// (SHA-256(svtnID || pubkey)[:8]), derive the HMAC key via hmac.DeriveKey, and
// register a forwarding entry for BOTH the source (so RouteFrame can look up
// the auth key) and a destination (so SVTNRoute succeeds).
type composedFixture struct {
	env    outerassembler.Envelope
	router *routing.Router
}

func newComposedFixture(t *testing.T) composedFixture {
	t.Helper()

	var svtnID [16]byte
	copy(svtnID[:], "s-bl-oa-svtn0001")

	// Deterministic node keypair.
	var nodeSeed [32]byte
	nodeSeed[0] = 0xA1
	copy(nodeSeed[1:], "s-bl-oa-node-seed-filler-xxxxxxx")
	nodePub, nodePriv, err := ed25519.GenerateKey(bytes.NewReader(nodeSeed[:]))
	if err != nil {
		t.Fatalf("generate node keypair: %v", err)
	}

	// Deterministic router keypair (needed for GenerateChallenge signature).
	var routerSeed [32]byte
	routerSeed[0] = 0xB2
	copy(routerSeed[1:], "s-bl-oa-rtr--seed-filler-xxxxxxx")
	_, routerPriv, err := ed25519.GenerateKey(bytes.NewReader(routerSeed[:]))
	if err != nil {
		t.Fatalf("generate router keypair: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Derive srcAddr = SHA-256(svtnID || pubkey)[:8].
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var srcAddr [8]byte
	copy(srcAddr[:], sum[:8])

	// Complete challenge-response so IsAdmitted(svtnID, srcAddr) returns true.
	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	// Per-(node, SVTN) auth key — matches admission.RegisterKey derivation.
	authKey := hmac.DeriveKey([]byte(nodePub), svtnID)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)

	// Destination forwarding entry so SVTNRoute succeeds.
	var dstAddr [8]byte
	copy(dstAddr[:], "oa-dst01")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xAA})

	return composedFixture{
		env: outerassembler.Envelope{
			SVTNID:       svtnID,
			SrcAddr:      srcAddr,
			DstAddr:      dstAddr,
			FrameAuthKey: authKey,
		},
		router: r,
	}
}

// AC-001 / F-003 — Composed wire-format round-trip.
//
// outerassembler.Assemble emits wire bytes that netingress.ReadFrame parses
// back into a header + payload, and routing.RouteFrame accepts the frame
// (HMAC verifies, admitted-set check passes, SVTN forwarding succeeds).
//
// This is the concrete closure of wave-adv F-003: pre-S-BL.OA the composed
// path was tested only in fragments (assembler-side HMAC recompute, routing-
// side HMAC verify, netingress-side truncation). Nothing exercised the three
// packages in the same test at real payload bytes; a mis-wired HMAC message
// shape between sender and verifier would have escaped every unit test.
func TestIntegration_Composed_RoutingRouteFrameVerifies(t *testing.T) {
	t.Parallel()

	fix := newComposedFixture(t)

	tests := []struct {
		name       string
		cf         halfchannel.ChannelFrame
		sackBitmap [8]byte
	}{
		{
			name: "data_frame_no_flags",
			cf: halfchannel.ChannelFrame{
				ChanID:    1,
				ChanSeq:   1,
				FrameType: frame.FrameTypeData,
				Flags:     0,
				Payload:   []byte("hello, verifier"),
			},
		},
		{
			name: "empty_tick_zero_length",
			cf: halfchannel.ChannelFrame{
				ChanID:    1,
				ChanSeq:   2,
				FrameType: frame.FrameTypeEmptyTick,
				Flags:     0,
				Payload:   nil,
			},
		},
		{
			name: "data_frame_with_sack_bitmap",
			cf: halfchannel.ChannelFrame{
				ChanID:    2,
				ChanSeq:   1,
				FrameType: frame.FrameTypeData,
				Flags:     outerassembler.FlagSACKPresent,
				Payload:   []byte("with sack window"),
			},
			sackBitmap: [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 1. Assemble the wire frame.
			wire, err := outerassembler.Assemble(tc.cf, tc.sackBitmap, fix.env)
			if err != nil {
				t.Fatalf("Assemble: %v", err)
			}

			// 2. Feed the wire bytes through netingress.ReadFrame — the
			//    same code path Serve/ServeConn drive on every accepted
			//    connection. If PayloadLen were wrong, this would return
			//    io.ErrUnexpectedEOF because the reader would run out.
			hdr, payload, err := netingress.ReadFrame(bytes.NewReader(wire))
			if err != nil {
				t.Fatalf("netingress.ReadFrame: %v", err)
			}

			// The header fields must reflect the envelope + channel-frame
			// discriminator (BC-2.01.002 PC-2).
			if hdr.SVTNID != fix.env.SVTNID {
				t.Errorf("SVTNID = % x, want % x", hdr.SVTNID, fix.env.SVTNID)
			}
			if hdr.SrcAddr != fix.env.SrcAddr {
				t.Errorf("SrcAddr = % x, want % x", hdr.SrcAddr, fix.env.SrcAddr)
			}
			if hdr.DstAddr != fix.env.DstAddr {
				t.Errorf("DstAddr = % x, want % x", hdr.DstAddr, fix.env.DstAddr)
			}
			if hdr.FrameType != tc.cf.FrameType {
				t.Errorf("FrameType = 0x%02x, want 0x%02x", hdr.FrameType, tc.cf.FrameType)
			}

			// 3. Route: this exercises verifyFrameHMAC against the same
			//    zeroed-tag + header + payload message the assembler used
			//    to compute the tag. If either side's message shape drifts,
			//    this returns ErrHMACVerificationFailed.
			if err := routing.RouteFrame(hdr, payload, fix.router); err != nil {
				t.Fatalf("routing.RouteFrame: %v (want nil — HMAC verify + admitted + SVTNRoute all succeed)", err)
			}
		})
	}
}

// AC — Adversarial dual 1 (F-003 negative case).
//
// Flipping any bit in the wire frame after Assemble must cause
// routing.RouteFrame to return ErrHMACVerificationFailed. This is the wire-
// forgery detection contract: the assembler's tag is the ground truth; any
// on-wire mutation is caught by the verifier. Traces to BC-2.05.008 PC-2.
func TestIntegration_FlippedBit_FailsHMAC(t *testing.T) {
	t.Parallel()

	fix := newComposedFixture(t)
	cf := halfchannel.ChannelFrame{
		ChanID:    1,
		ChanSeq:   1,
		FrameType: frame.FrameTypeData,
		Payload:   []byte("this-payload-will-be-corrupted"),
	}

	// Assemble the pristine frame once — we mutate copies so we can flip
	// bits in multiple wire regions independently.
	pristine, err := outerassembler.Assemble(cf, [8]byte{}, fix.env)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Regions of the wire frame: outer header (bytes 0..43), channel header
	// (bytes 44..55; no SACK for cf.Flags=0), and payload (bytes 56..).
	// A bit flip anywhere covered by the MAC must fail.
	regions := []struct {
		name   string
		offset int
	}{
		{"outer_header_svtn_byte", 4},                          // hdr.SVTNID[0]
		{"outer_header_src_addr_byte", 20},                     // hdr.SrcAddr[0]
		{"outer_header_dst_addr_byte", 28},                     // hdr.DstAddr[0]
		{"channel_header_chan_id_byte", frame.OuterHeaderSize}, // chan_id[0]
		{"channel_header_flags_byte", frame.OuterHeaderSize + 8},
		{"payload_first_byte", frame.OuterHeaderSize + outerassembler.ChannelHeaderFixedSize},
		{"payload_middle_byte", frame.OuterHeaderSize + outerassembler.ChannelHeaderFixedSize + len(cf.Payload)/2},
	}

	for _, region := range regions {
		region := region
		t.Run(region.name, func(t *testing.T) {
			t.Parallel()

			wire := make([]byte, len(pristine))
			copy(wire, pristine)
			// Flip bit 0 of the target byte — a single-bit corruption is
			// the strongest adversarial signal (largest surface for a
			// weak MAC to leak on).
			wire[region.offset] ^= 0x01

			hdr, payload, err := netingress.ReadFrame(bytes.NewReader(wire))
			if err != nil {
				t.Fatalf("netingress.ReadFrame after bit flip at %d: %v (want nil — flip is inside the MAC message, not the framing)",
					region.offset, err)
			}
			err = routing.RouteFrame(hdr, payload, fix.router)
			if !errors.Is(err, routing.ErrHMACVerificationFailed) {
				t.Errorf("routing.RouteFrame after bit flip at %d = %v, want errors.Is(err, routing.ErrHMACVerificationFailed)",
					region.offset, err)
			}
		})
	}
}

// AC — Adversarial dual 2 (F-003 truncation case).
//
// A truncated wire frame must fail parse at netingress.ReadFrame with
// io.ErrUnexpectedEOF. This closes the CWE-400 bounded-read half of F-003:
// a well-authored sender never emits a truncated frame, but the receiver
// must fail closed on one regardless.
func TestIntegration_Truncation_FailsParse(t *testing.T) {
	t.Parallel()

	fix := newComposedFixture(t)
	cf := halfchannel.ChannelFrame{
		ChanID:    1,
		ChanSeq:   1,
		FrameType: frame.FrameTypeData,
		Payload:   []byte("full-frame-truncated-below"),
	}
	wire, err := outerassembler.Assemble(cf, [8]byte{}, fix.env)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	tests := []struct {
		name  string
		bytes []byte
	}{
		{
			// Header truncated mid-way — ReadFrame's io.ReadFull on the
			// header buffer sees a short read.
			name:  "outer_header_truncated_at_20",
			bytes: wire[:20],
		},
		{
			// Header complete (44 bytes), payload region truncated to zero.
			// PayloadLen claims 12+len(payload), reader has 0 payload bytes.
			name:  "payload_truncated_to_zero",
			bytes: wire[:frame.OuterHeaderSize],
		},
		{
			// Header + partial channel header (10 of 12 needed for the
			// channel-header-only region).
			name:  "channel_header_truncated_10_of_12",
			bytes: wire[:frame.OuterHeaderSize+10],
		},
		{
			// Header + channel header complete, payload cut in half.
			name:  "payload_truncated_at_half",
			bytes: wire[:frame.OuterHeaderSize+outerassembler.ChannelHeaderFixedSize+len(cf.Payload)/2],
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := netingress.ReadFrame(bytes.NewReader(tc.bytes))
			if err == nil {
				t.Fatalf("netingress.ReadFrame(%d bytes) returned nil error, want io.ErrUnexpectedEOF", len(tc.bytes))
			}
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Errorf("err = %v, want errors.Is(err, io.ErrUnexpectedEOF)", err)
			}
		})
	}
}
