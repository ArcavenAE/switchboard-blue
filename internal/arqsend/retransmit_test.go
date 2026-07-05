// Package arqsend_test exercises the boundary-layer retransmit-SEND path.
//
// S-BL.ARQ-TX wires the pure-core ARQ state machine (internal/arq) to
// wire bytes through internal/outerassembler, closing BC-2.02.005 PC-3.
//
// Test naming follows the story convention:
//
//	TestRetransmit_HappyPath_ProducesWireForNewSeq       (AC-002)
//	TestRetransmit_UnknownOldSeqReturnsError             (AC-002 negative)
//	TestRetransmit_DispatchErrorLeavesARQStateIntact     (AC-005)
//	TestRetransmit_NewSeqDiffersFromOldSeq_BC205_PC5     (BC-2.02.005 PC-5)
package arqsend_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/arq"
	"github.com/arcavenae/switchboard/internal/arqsend"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// envForTest constructs a deterministic Envelope for tests. FrameAuthKey is a
// fixed 32-byte value so wire-format tests can precompute HMAC expectations.
func envForTest() outerassembler.Envelope {
	var env outerassembler.Envelope
	// SVTNID: 0x11..0x11 (16 bytes)
	for i := range env.SVTNID {
		env.SVTNID[i] = 0x11
	}
	// SrcAddr: 0xAA..0xAA (8 bytes)
	for i := range env.SrcAddr {
		env.SrcAddr[i] = 0xAA
	}
	// DstAddr: 0xBB..0xBB (8 bytes)
	for i := range env.DstAddr {
		env.DstAddr[i] = 0xBB
	}
	// FrameAuthKey: 0x33..0x33 (32 bytes)
	for i := range env.FrameAuthKey {
		env.FrameAuthKey[i] = 0x33
	}
	return env
}

// TestRetransmit_HappyPath_ProducesWireForNewSeq exercises AC-002: given an
// ARQ state with an in-flight payload at oldSeq, arqsend.Retransmit fetches
// the original content via arq.PayloadForInFlight, composes wire bytes via
// outerassembler.Assemble carrying newSeq in ChanSeq, calls the injected
// Dispatch callback with those bytes, then EnqueueSends the payload under
// newSeq and RemoveInFlights oldSeq.
func TestRetransmit_HappyPath_ProducesWireForNewSeq(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	orig := []byte("retransmit-me")
	a.EnqueueSend(3, orig, now)

	var dispatched [][]byte
	dispatch := func(wire []byte) error {
		// Copy so subsequent slice reuse in the sender can't rewrite our record.
		buf := make([]byte, len(wire))
		copy(buf, wire)
		dispatched = append(dispatched, buf)
		return nil
	}

	sender := arqsend.New(a, envForTest(), arqsend.WithChanID(0xC0FFEE01))
	if err := sender.Retransmit(3, 42, now, dispatch); err != nil {
		t.Fatalf("Retransmit(3→42): unexpected error: %v", err)
	}

	if len(dispatched) != 1 {
		t.Fatalf("Retransmit dispatched %d frames, want 1", len(dispatched))
	}

	// Parse the outer header off the wire and assert the ChanSeq inside the
	// channel header equals newSeq (BC-2.02.005 PC-5).
	wire := dispatched[0]
	if len(wire) < frame.OuterHeaderSize+outerassembler.ChannelHeaderFixedSize {
		t.Fatalf("wire too short: %d bytes", len(wire))
	}
	hdr, err := frame.ParseOuterHeader(wire[:frame.OuterHeaderSize])
	if err != nil {
		t.Fatalf("parse outer header: %v", err)
	}
	if hdr.SVTNID != envForTest().SVTNID {
		t.Errorf("SVTNID mismatch: got %x, want %x", hdr.SVTNID, envForTest().SVTNID)
	}

	chdr, err := outerassembler.DecodeChannelHeader(wire[frame.OuterHeaderSize:])
	if err != nil {
		t.Fatalf("decode channel header: %v", err)
	}
	if chdr.ChanSeq != 42 {
		t.Errorf("channel ChanSeq: got %d, want 42 (new seq)", chdr.ChanSeq)
	}
	if chdr.ChanID != 0xC0FFEE01 {
		t.Errorf("channel ChanID: got %x, want 0xC0FFEE01", chdr.ChanID)
	}

	// ARQ state — oldSeq released, newSeq now in-flight.
	if a.InFlightContains(3) {
		t.Errorf("oldSeq 3 still in flight after retransmit; expected release")
	}
	if !a.InFlightContains(42) {
		t.Errorf("newSeq 42 not in flight; expected EnqueueSend under new seq")
	}
	if got := a.PayloadForInFlight(42); string(got) != string(orig) {
		t.Errorf("newSeq 42 payload: got %q, want %q", got, orig)
	}
}

// TestRetransmit_UnknownOldSeqReturnsError exercises AC-002 negative path:
// asking for a seq that is not in flight returns ErrSequenceNotInFlight and
// makes no side-effects (dispatch not called, no new EnqueueSend).
func TestRetransmit_UnknownOldSeqReturnsError(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	called := false
	dispatch := func(_ []byte) error {
		called = true
		return nil
	}

	sender := arqsend.New(a, envForTest())
	err := sender.Retransmit(99, 100, now, dispatch)
	if !errors.Is(err, arqsend.ErrSequenceNotInFlight) {
		t.Fatalf("Retransmit(unknown): want ErrSequenceNotInFlight, got %v", err)
	}
	if called {
		t.Errorf("dispatch was called for an unknown oldSeq")
	}
	if a.InFlightContains(100) {
		t.Errorf("newSeq 100 unexpectedly enqueued after error return")
	}
}

// TestRetransmit_DispatchErrorLeavesARQStateIntact exercises AC-005: if the
// dispatch callback returns an error, the sender surfaces the error without
// mutating ARQ state — oldSeq remains in flight, newSeq is NOT enqueued, no
// silent loss.
//
// This is the load-bearing "no orphan state" property: a failed retransmit
// must be re-tryable on the next GapsToRetransmit pass.
func TestRetransmit_DispatchErrorLeavesARQStateIntact(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(7, []byte("payload"), now)

	dispatchErr := errors.New("simulated wire failure")
	dispatch := func(_ []byte) error { return dispatchErr }

	sender := arqsend.New(a, envForTest())
	err := sender.Retransmit(7, 55, now, dispatch)
	if !errors.Is(err, dispatchErr) {
		t.Fatalf("Retransmit with failing dispatch: want error chain to include dispatchErr, got %v", err)
	}
	if !a.InFlightContains(7) {
		t.Errorf("oldSeq 7 released despite dispatch failure — orphan state")
	}
	if a.InFlightContains(55) {
		t.Errorf("newSeq 55 enqueued despite dispatch failure — orphan state")
	}
}

// TestRetransmit_NewSeqDiffersFromOldSeq_BC205_PC5 is the explicit assertion
// of BC-2.02.005 PC-5 (QUIC retransmit model): the retransmit carries the
// original content but a NEW frame sequence number, never the old one.
func TestRetransmit_NewSeqDiffersFromOldSeq_BC205_PC5(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(10, []byte("data"), now)

	var seenSeq uint32
	dispatch := func(wire []byte) error {
		chdr, err := outerassembler.DecodeChannelHeader(wire[frame.OuterHeaderSize:])
		if err != nil {
			return err
		}
		seenSeq = chdr.ChanSeq
		return nil
	}

	sender := arqsend.New(a, envForTest())
	if err := sender.Retransmit(10, 200, now, dispatch); err != nil {
		t.Fatalf("Retransmit(10→200): %v", err)
	}
	if seenSeq != 200 {
		t.Errorf("on-wire ChanSeq: got %d, want 200 (BC-2.02.005 PC-5: NEW seq)", seenSeq)
	}
	if seenSeq == 10 {
		t.Fatalf("on-wire ChanSeq equals old seq — BC-2.02.005 PC-5 violated")
	}
}

// TestRetransmit_HMACVerifiableAgainstEnvelopeKey composes with
// outerassembler's HMAC contract (routing.verifyFrameHMAC-shaped): a wire
// frame emitted by Retransmit must be HMAC-verifiable by the same
// FrameAuthKey. This asserts arqsend consumes Assemble correctly rather
// than short-circuiting the MAC path.
func TestRetransmit_HMACVerifiableAgainstEnvelopeKey(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a.EnqueueSend(1, []byte("verify-me"), now)

	env := envForTest()
	var wire []byte
	dispatch := func(w []byte) error {
		wire = append(wire, w...)
		return nil
	}
	sender := arqsend.New(a, env)
	if err := sender.Retransmit(1, 2, now, dispatch); err != nil {
		t.Fatalf("Retransmit: %v", err)
	}

	// Re-verify via the same shape routing.verifyFrameHMAC uses.
	hdr, err := frame.ParseOuterHeader(wire[:frame.OuterHeaderSize])
	if err != nil {
		t.Fatalf("parse outer header: %v", err)
	}
	wireTag := hdr.HMACTag
	hdrForMAC := hdr
	hdrForMAC.HMACTag = [8]byte{}
	encoded := frame.EncodeOuterHeader(hdrForMAC)
	msg := make([]byte, len(encoded)+len(wire)-frame.OuterHeaderSize)
	copy(msg, encoded[:])
	copy(msg[len(encoded):], wire[frame.OuterHeaderSize:])
	if !hmac.VerifyHMAC(env.FrameAuthKey[:], msg, wireTag) {
		t.Fatalf("HMAC verify failed against Envelope.FrameAuthKey; arqsend did not compose Assemble correctly")
	}
}
