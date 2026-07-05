package outerassembler_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// testEnvelope returns a deterministic Envelope + auth key for tests.
func testEnvelope(t *testing.T) (outerassembler.Envelope, [hmac.KeySize]byte) {
	t.Helper()
	var (
		svtn [16]byte
		src  [8]byte
		dst  [8]byte
		key  [hmac.KeySize]byte
	)
	for i := 0; i < 16; i++ {
		svtn[i] = byte(0x10 | i)
	}
	for i := 0; i < 8; i++ {
		src[i] = byte(0x20 | i)
		dst[i] = byte(0x30 | i)
	}
	for i := 0; i < hmac.KeySize; i++ {
		key[i] = byte(0x40 | (i & 0x3F))
	}
	env := outerassembler.Envelope{
		SVTNID:       svtn,
		SrcAddr:      src,
		DstAddr:      dst,
		FrameAuthKey: key,
	}
	return env, key
}

// AC-001 / F-003 — Composed wire-format round-trip.
//
// A DATA ChannelFrame from halfchannel.Tick is composed via Assemble into
// wire bytes, then those bytes are parsed by the same code path a router
// runs (frame.ParseOuterHeader), and the HMAC computed here must verify
// against the routing.verifyFrameHMAC contract — asserted separately by
// TestAssemble_Composed_RoutingRouteFrameVerifies in the integration test.
//
// This unit test asserts the byte-level shape: outer_header_size (44) +
// channel_header_size (12 or 20) + len(payload) == len(wire_bytes) and
// hdr.PayloadLen == channel_header_size + len(payload).
func TestAssemble_WireShape_MatchesPayloadLen(t *testing.T) {
	t.Parallel()

	env, _ := testEnvelope(t)

	tests := []struct {
		name       string
		cf         halfchannel.ChannelFrame
		sackBitmap [8]byte
		wantChdr   int
	}{
		{
			name: "data_no_flags_12byte_chanhdr",
			cf: halfchannel.ChannelFrame{
				ChanID:    42,
				ChanSeq:   1,
				FrameType: frame.FrameTypeData,
				Flags:     0,
				Payload:   []byte("hello switchboard"),
			},
			wantChdr: 12,
		},
		{
			name: "data_sack_present_20byte_chanhdr",
			cf: halfchannel.ChannelFrame{
				ChanID:    99,
				ChanSeq:   7,
				FrameType: frame.FrameTypeData,
				Flags:     outerassembler.FlagSACKPresent,
				Payload:   []byte{0xde, 0xad, 0xbe, 0xef},
			},
			sackBitmap: [8]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			wantChdr:   20,
		},
		{
			name: "empty_tick_zero_length_payload",
			cf: halfchannel.ChannelFrame{
				ChanID:    1,
				ChanSeq:   2,
				FrameType: frame.FrameTypeEmptyTick,
				Flags:     0,
				Payload:   nil,
			},
			wantChdr: 12,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wire, err := outerassembler.Assemble(tc.cf, tc.sackBitmap, env)
			if err != nil {
				t.Fatalf("Assemble: %v", err)
			}

			wantTotal := frame.OuterHeaderSize + tc.wantChdr + len(tc.cf.Payload)
			if len(wire) != wantTotal {
				t.Errorf("len(wire) = %d, want %d (44 outer + %d chanhdr + %d payload)",
					len(wire), wantTotal, tc.wantChdr, len(tc.cf.Payload))
			}

			// PayloadLen field reflects channel_header_size + len(payload) —
			// BC-2.01.004 invariant 3.
			gotLen := binary.BigEndian.Uint16(wire[2:4])
			wantLen := uint16(tc.wantChdr + len(tc.cf.Payload))
			if gotLen != wantLen {
				t.Errorf("PayloadLen = %d, want %d", gotLen, wantLen)
			}
		})
	}
}

// AC — Assemble sets OuterHeader.frame_type from ChannelFrame.FrameType
// (BC-2.01.002 PC-2: EMPTY_TICK / DATA discriminator passthrough).
func TestAssemble_FrameTypePassthrough(t *testing.T) {
	t.Parallel()

	env, _ := testEnvelope(t)

	tests := []struct {
		name string
		ft   frame.FrameType
	}{
		{"data", frame.FrameTypeData},
		{"empty_tick", frame.FrameTypeEmptyTick},
		{"ctl", frame.FrameTypeCtl},
		{"arq", frame.FrameTypeArq},
		{"fec", frame.FrameTypeFec},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cf := halfchannel.ChannelFrame{
				ChanID:    1,
				ChanSeq:   1,
				FrameType: tc.ft,
				Payload:   []byte("x"),
			}
			// EMPTY_TICK canonically has nil payload; test the assembler is
			// happy to encode either — the effectful layer chooses the shape.
			if tc.ft == frame.FrameTypeEmptyTick {
				cf.Payload = nil
			}

			wire, err := outerassembler.Assemble(cf, [8]byte{}, env)
			if err != nil {
				t.Fatalf("Assemble(%s): %v", tc.name, err)
			}
			if wire[1] != byte(tc.ft) {
				t.Errorf("wire[1] frame_type = 0x%02x, want 0x%02x", wire[1], byte(tc.ft))
			}
		})
	}
}

// AC — Assemble encodes outer-header fields from the Envelope.
func TestAssemble_OuterHeaderFieldsFromEnvelope(t *testing.T) {
	t.Parallel()

	env, _ := testEnvelope(t)
	cf := halfchannel.ChannelFrame{
		ChanID:    7,
		ChanSeq:   11,
		FrameType: frame.FrameTypeData,
		Payload:   []byte("abc"),
	}

	wire, err := outerassembler.Assemble(cf, [8]byte{}, env)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// wire[0]: version byte per BC-2.01.004 postcondition 2.
	if wire[0] != frame.VersionByte {
		t.Errorf("wire[0] version = 0x%02x, want 0x%02x", wire[0], frame.VersionByte)
	}
	// wire[4..20]: svtn_id
	if !bytes.Equal(wire[4:20], env.SVTNID[:]) {
		t.Errorf("wire[4..20] svtn_id = % x, want % x", wire[4:20], env.SVTNID[:])
	}
	// wire[20..28]: src_addr
	if !bytes.Equal(wire[20:28], env.SrcAddr[:]) {
		t.Errorf("wire[20..28] src_addr = % x, want % x", wire[20:28], env.SrcAddr[:])
	}
	// wire[28..36]: dst_addr
	if !bytes.Equal(wire[28:36], env.DstAddr[:]) {
		t.Errorf("wire[28..36] dst_addr = % x, want % x", wire[28:36], env.DstAddr[:])
	}
}

// AC — Channel-header bytes at wire offsets 44..(44+chanhdr_size) decode
// back to the source ChannelFrame's ChanID/ChanSeq/Flags.
func TestAssemble_ChannelHeaderAtOffset44(t *testing.T) {
	t.Parallel()

	env, _ := testEnvelope(t)
	sackBitmap := [8]byte{0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10, 0x11}

	cf := halfchannel.ChannelFrame{
		ChanID:    0xdeadbeef,
		ChanSeq:   0x12345678,
		FrameType: frame.FrameTypeData,
		Flags:     outerassembler.FlagARQReq | outerassembler.FlagSACKPresent,
		Payload:   []byte("ping"),
	}

	wire, err := outerassembler.Assemble(cf, sackBitmap, env)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	chdr, n, err := outerassembler.DecodeChannelHeaderN(wire[frame.OuterHeaderSize:])
	if err != nil {
		t.Fatalf("DecodeChannelHeaderN: %v", err)
	}
	if n != 20 {
		t.Errorf("channel header size = %d, want 20 (ARQ|SACK sets SACK_present bit)", n)
	}
	if chdr.ChanID != cf.ChanID {
		t.Errorf("chan_id = %#x, want %#x", chdr.ChanID, cf.ChanID)
	}
	if chdr.ChanSeq != cf.ChanSeq {
		t.Errorf("chan_seq = %#x, want %#x", chdr.ChanSeq, cf.ChanSeq)
	}
	if chdr.Flags != cf.Flags {
		t.Errorf("flags = 0x%02x, want 0x%02x", chdr.Flags, cf.Flags)
	}
	if chdr.SACKBitmap != sackBitmap {
		t.Errorf("sack_bitmap = % x, want % x", chdr.SACKBitmap, sackBitmap)
	}

	// Payload sits directly after the channel header.
	payloadStart := frame.OuterHeaderSize + n
	if !bytes.Equal(wire[payloadStart:], cf.Payload) {
		t.Errorf("payload bytes = %q, want %q", wire[payloadStart:], cf.Payload)
	}
}

// AC — Defensive MaxPayloadSize re-check: Assemble returns
// ErrPayloadTooLarge when app_payload_len + channel_header_size > uint16
// max, even if the caller passed a payload halfchannel would have rejected.
//
// The check is defensive because internal/halfchannel already enforces this
// on Enqueue (BC-2.01.002 PC5). But callers may build ChannelFrame structs
// directly (tests, control frames not routed through Enqueue), so the
// assembler MUST NOT rely on upstream enforcement for a wire-format
// truncation invariant.
func TestAssemble_RejectsPayloadThatWouldTruncatePayloadLen(t *testing.T) {
	t.Parallel()

	env, _ := testEnvelope(t)

	// 65523 bytes + 12-byte channel header = 65535 → the largest legal
	// SACK-absent frame. Try one over that.
	//
	// SACK-absent channel header is 12 bytes; MaxPayloadSize for SACK=0
	// is 65535 − 12 = 65523. Boundary tests:
	//   65523 (max valid), 65524 (over), 65535 (way over)
	tests := []struct {
		name       string
		flags      byte
		payloadLen int
		wantErr    error
	}{
		{"exact_max_sack_absent", 0, 65535 - outerassembler.ChannelHeaderFixedSize, nil},
		{"one_over_sack_absent", 0, 65535 - outerassembler.ChannelHeaderFixedSize + 1, outerassembler.ErrPayloadTooLarge},
		{"exact_max_sack_present", outerassembler.FlagSACKPresent, 65535 - outerassembler.ChannelHeaderWithSACKSize, nil},
		{"one_over_sack_present", outerassembler.FlagSACKPresent, 65535 - outerassembler.ChannelHeaderWithSACKSize + 1, outerassembler.ErrPayloadTooLarge},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cf := halfchannel.ChannelFrame{
				ChanID:    1,
				ChanSeq:   1,
				FrameType: frame.FrameTypeData,
				Flags:     tc.flags,
				Payload:   make([]byte, tc.payloadLen),
			}
			_, err := outerassembler.Assemble(cf, [8]byte{}, env)
			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("Assemble(%d bytes) = %v, want nil", tc.payloadLen, err)
				}
			} else {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("Assemble(%d bytes) = %v, want errors.Is(err, %v)", tc.payloadLen, err, tc.wantErr)
				}
			}
		})
	}
}

// AC — The HMAC tag in the outer header verifies against a MAC computed
// over (zeroed-tag outer_header || channel_header || payload) with the
// FrameAuthKey. This is the routing.verifyFrameHMAC contract in
// internal/routing/routing.go.
//
// This test asserts the mechanical shape locally (without importing
// routing). TestIntegration_Composed_RoutingRouteFrameVerifies
// exercises the actual routing.RouteFrame verifier end-to-end.
func TestAssemble_HMACTagMatchesLocalRecompute(t *testing.T) {
	t.Parallel()

	env, key := testEnvelope(t)

	cf := halfchannel.ChannelFrame{
		ChanID:    5,
		ChanSeq:   9,
		FrameType: frame.FrameTypeData,
		Flags:     0,
		Payload:   []byte("hmac-check"),
	}

	wire, err := outerassembler.Assemble(cf, [8]byte{}, env)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	// Extract the wire tag from bytes 36..44.
	var wireTag [hmac.TagSize]byte
	copy(wireTag[:], wire[36:44])

	// Rebuild: zero the tag in the outer-header portion and concatenate
	// with the channel header + payload.
	msg := make([]byte, len(wire))
	copy(msg, wire)
	for i := 36; i < 44; i++ {
		msg[i] = 0
	}

	expected := hmac.ComputeHMAC(key[:], msg)
	if expected != wireTag {
		t.Errorf("wire HMAC tag = % x, expected % x", wireTag, expected)
	}
}
