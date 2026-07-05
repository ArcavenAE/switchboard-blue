// Package arqsend_test — end-to-end integration exercising
// gap-detection → arqsend.Retransmit → wire bytes → routing.RouteFrame.
//
// This test is the load-bearing closure of S-BL.ARQ-TX drift lineage
// (S403-H1-DEFER, Wave 4 audit): pre-story the retransmit-SEND path was
// implemented in fragments (arq.OnAck returns [][]byte; nothing turned
// those payloads into HMAC-authenticated wire frames verifiable by the
// receive-side routing/admission stack). This test drives the composed
// path at real payload bytes:
//
//  1. Set up an admitted node + router forwarding entries (mirrors
//     outerassembler's composedFixture pattern).
//  2. Populate the ARQ in-flight queue with three payloads (seqs 1,2,3).
//  3. Simulate the receiver ACKing seq 0 with a SACK bitmap that marks
//     seq 2 as received — leaving seqs 1 and 3 as gaps.
//  4. Walk arq.GapsToRetransmit; for each gap seq, arqsend.Retransmit
//     into a dispatch that captures the wire bytes.
//  5. Feed each wire frame through netingress.ReadFrame → routing.RouteFrame.
//     Both must verify: HMAC ok, admitted ok, SVTNRoute ok.
//  6. Assert ARQ state is consistent: old gap seqs released, new seqs
//     enqueued under their new numbers, non-gap seq (2) still in flight
//     from its original enqueue (the receiver reported it via SACK but
//     the sender's OnAck is out of scope here; GapsToRetransmit is the
//     boundary we exercise).
package arqsend_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/arq"
	"github.com/arcavenae/switchboard/internal/arqsend"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/netingress"
	"github.com/arcavenae/switchboard/internal/outerassembler"
	"github.com/arcavenae/switchboard/internal/routing"
)

// composedFixture wires an admitted node + router so retransmitted wire
// frames verify end-to-end.
type composedFixture struct {
	env    outerassembler.Envelope
	router *routing.Router
}

func newComposedFixture(t *testing.T) composedFixture {
	t.Helper()

	var svtnID [16]byte
	copy(svtnID[:], "s-bl-arqtx-svtn0")

	var nodeSeed [32]byte
	nodeSeed[0] = 0xC3
	copy(nodeSeed[1:], "s-bl-arqtx-node-seed-filler-xxx")
	nodePub, nodePriv, err := ed25519.GenerateKey(bytes.NewReader(nodeSeed[:]))
	if err != nil {
		t.Fatalf("generate node keypair: %v", err)
	}

	var routerSeed [32]byte
	routerSeed[0] = 0xD4
	copy(routerSeed[1:], "s-bl-arqtx-rtr--seed-filler-xxx")
	_, routerPriv, err := ed25519.GenerateKey(bytes.NewReader(routerSeed[:]))
	if err != nil {
		t.Fatalf("generate router keypair: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	h := sha256.New()
	h.Write(svtnID[:])
	h.Write([]byte(nodePub))
	sum := h.Sum(nil)
	var srcAddr [8]byte
	copy(srcAddr[:], sum[:8])

	challenge, err := admission.GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	nonceSig := ed25519.Sign(nodePriv, challenge.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: nonceSig}
	if err := admission.AdmitNode(challenge, resp, nodePub, svtnID, ks); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	authKey := hmac.DeriveKey([]byte(nodePub), svtnID)

	r := routing.NewRouter(ks)
	r.RegisterForwardingEntry(svtnID, srcAddr, authKey)

	var dstAddr [8]byte
	copy(dstAddr[:], "arqtxdst")
	r.RegisterForwardingEntry(svtnID, dstAddr, [hmac.KeySize]byte{0xAB})

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

// TestIntegration_GapWalkToRoutedRetransmit is the load-bearing end-to-end
// integration for S-BL.ARQ-TX: gap-detection → arqsend.Retransmit →
// netingress.ReadFrame → routing.RouteFrame all succeed for every gap.
//
// Sender-side setup: three frames in flight (seqs 1,2,3), all with distinct
// payloads. Receiver-side signal: cumulative ACK of seq 0 with a SACK
// bitmap that marks seq 2 as received — leaving seqs 1 and 3 as gaps.
//
// Sender processes the gap list via arqsend.Retransmit with monotonically
// increasing new seq numbers (100, 101). Each retransmit's wire bytes are
// captured, parsed, and routed; all must verify. Post-condition: old
// gap seqs 1 and 3 are released, new seqs 100 and 101 are in flight
// under their new numbers with the original payloads intact (BC-2.02.005
// PC-5).
func TestIntegration_GapWalkToRoutedRetransmit(t *testing.T) {
	t.Parallel()

	fix := newComposedFixture(t)

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	payloads := map[uint32][]byte{
		1: []byte("payload-one-original-content"),
		2: []byte("payload-two-original-content"),
		3: []byte("payload-three-original-content"),
	}
	for seq, p := range payloads {
		a.EnqueueSend(seq, p, now)
	}

	// Receiver ACKed seq 0 with SACK marking seq 2 received.
	// Bitmap: bit 0 (MSB of byte 0) covers ackSeq+1=1; bit 1 covers seq 2; bit 2 covers seq 3.
	// Set bit for seq 2 only: bit 1 (MSB-1 of byte 0) = 0x40.
	var sackBitmap [arq.SACKBitmapBytes]byte
	sackBitmap[0] = 0x40

	gaps := a.GapsToRetransmit(0, sackBitmap)
	if len(gaps) != 2 || gaps[0] != 1 || gaps[1] != 3 {
		t.Fatalf("GapsToRetransmit(0, sack{seq2}) = %v, want [1 3]", gaps)
	}

	sender := arqsend.New(a, fix.env)

	// Retransmit each gap under a fresh, monotonically increasing new seq.
	// This mirrors the caller pattern: the sender's transmit-side seq
	// counter advances; retransmits consume from that counter.
	newSeqBase := uint32(100)
	type routed struct {
		oldSeq uint32
		newSeq uint32
		wire   []byte
	}
	var routedFrames []routed

	for i, oldSeq := range gaps {
		newSeq := newSeqBase + uint32(i)
		var captured []byte
		dispatch := func(wire []byte) error {
			// Defensive copy — the wire slice may be reused across dispatches.
			captured = make([]byte, len(wire))
			copy(captured, wire)
			return nil
		}
		if err := sender.Retransmit(oldSeq, newSeq, now, dispatch); err != nil {
			t.Fatalf("Retransmit(oldSeq=%d newSeq=%d): %v", oldSeq, newSeq, err)
		}
		if captured == nil {
			t.Fatalf("Retransmit(oldSeq=%d newSeq=%d) did not call dispatch", oldSeq, newSeq)
		}
		routedFrames = append(routedFrames, routed{oldSeq: oldSeq, newSeq: newSeq, wire: captured})
	}

	// Feed each retransmitted wire frame through the receive-side stack.
	for _, rf := range routedFrames {
		hdr, payload, err := netingress.ReadFrame(bytes.NewReader(rf.wire))
		if err != nil {
			t.Fatalf("netingress.ReadFrame(oldSeq=%d newSeq=%d wire): %v",
				rf.oldSeq, rf.newSeq, err)
		}
		if hdr.SVTNID != fix.env.SVTNID {
			t.Errorf("hdr.SVTNID mismatch on newSeq=%d", rf.newSeq)
		}
		if hdr.SrcAddr != fix.env.SrcAddr {
			t.Errorf("hdr.SrcAddr mismatch on newSeq=%d", rf.newSeq)
		}
		if err := routing.RouteFrame(hdr, payload, fix.router); err != nil {
			t.Fatalf("routing.RouteFrame(oldSeq=%d newSeq=%d): %v — want nil (HMAC verify + admitted + SVTNRoute)",
				rf.oldSeq, rf.newSeq, err)
		}
	}

	// ARQ state postconditions:
	//   - old gap seqs (1, 3) released
	//   - non-gap seq (2) still in flight (SACK-received; sender-side release
	//     of SACKed frames is OnAck's job, not GapsToRetransmit's)
	//   - new seqs (100, 101) now in flight, payloads byte-equal originals
	for _, oldSeq := range gaps {
		if a.InFlightContains(oldSeq) {
			t.Errorf("oldSeq %d still in flight after retransmit — expected release", oldSeq)
		}
	}
	if !a.InFlightContains(2) {
		t.Errorf("seq 2 (non-gap) unexpectedly released")
	}
	for i, oldSeq := range gaps {
		newSeq := newSeqBase + uint32(i)
		if !a.InFlightContains(newSeq) {
			t.Errorf("newSeq %d not in flight after retransmit", newSeq)
			continue
		}
		got := a.PayloadForInFlight(newSeq)
		want := payloads[oldSeq]
		if !bytes.Equal(got, want) {
			t.Errorf("newSeq %d payload = %q, want %q (from oldSeq %d)",
				newSeq, got, want, oldSeq)
		}
	}
}

// TestIntegration_MultipleGapsGetMonotonicNewSeqs asserts that when the
// caller walks GapsToRetransmit with monotonically increasing new seqs,
// each retransmitted frame carries the assigned seq on the wire (never
// the old seq). Explicit assertion of BC-2.02.005 PC-5 across a batch.
func TestIntegration_MultipleGapsGetMonotonicNewSeqs(t *testing.T) {
	t.Parallel()

	fix := newComposedFixture(t)

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Four in-flight frames, all gaps (ackSeq=0, empty SACK).
	oldSeqs := []uint32{10, 20, 30, 40}
	for _, s := range oldSeqs {
		a.EnqueueSend(s, []byte("payload"), now)
	}

	gaps := a.GapsToRetransmit(0, [arq.SACKBitmapBytes]byte{})
	if len(gaps) != len(oldSeqs) {
		t.Fatalf("GapsToRetransmit: got %d gaps, want %d", len(gaps), len(oldSeqs))
	}

	sender := arqsend.New(a, fix.env)
	newSeqStart := uint32(500)

	seenSeqsOnWire := make([]uint32, 0, len(gaps))
	for i, oldSeq := range gaps {
		newSeq := newSeqStart + uint32(i)
		dispatch := func(wire []byte) error {
			chdr, err := outerassembler.DecodeChannelHeader(wire[44:])
			if err != nil {
				return err
			}
			seenSeqsOnWire = append(seenSeqsOnWire, chdr.ChanSeq)
			return nil
		}
		if err := sender.Retransmit(oldSeq, newSeq, now, dispatch); err != nil {
			t.Fatalf("Retransmit(%d→%d): %v", oldSeq, newSeq, err)
		}
	}

	// Each wire seq must equal its assigned new seq — strictly monotonic,
	// never equal to the old seq (BC-2.02.005 PC-5 across a batch).
	for i, seenSeq := range seenSeqsOnWire {
		wantNew := newSeqStart + uint32(i)
		if seenSeq != wantNew {
			t.Errorf("wire[%d] ChanSeq = %d, want %d (BC-2.02.005 PC-5)", i, seenSeq, wantNew)
		}
		// Belt-and-suspenders: never equal the corresponding old seq.
		if seenSeq == oldSeqs[i] {
			t.Errorf("wire[%d] ChanSeq equals old seq %d — BC-2.02.005 PC-5 violated", i, oldSeqs[i])
		}
	}
}
