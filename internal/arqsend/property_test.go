// Property tests for Phase 6 formal hardening.
//
// The unit tests in retransmit_test.go and integration_test.go cover
// scenario-by-scenario acceptance criteria. This file adds *property* tests:
// invariants that must hold across many inputs, chained retransmits, and
// error-injection sequences.
//
// The three properties asserted:
//
//   - No-orphan-state on Dispatch error: after any number of failing
//     Retransmit calls, oldSeq remains InFlight and newSeq is NOT InFlight
//     (BC-2.02.005 no-orphan-state).
//   - Gap-walk termination: iterating GapsToRetransmit and calling
//     Retransmit for each gap must terminate in a bounded number of steps
//     (each successful Retransmit removes the old seq from in-flight, so
//     the gap set for a given ack shrinks monotonically).
//   - Sequence monotonicity: chained retransmits ratchet the ChanSeq stamped
//     in the dispatched wire header — never returning to an older seq.
//   - Byte-exact payload preservation: after a QUIC-model transition
//     (oldSeq→newSeq), PayloadForInFlight(newSeq) byte-equals the original
//     payload passed to the first EnqueueSend.
package arqsend_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/arq"
	"github.com/arcavenae/switchboard/internal/arqsend"
	"github.com/arcavenae/switchboard/internal/outerassembler"
)

// TestProperty_NoOrphanStateOnDispatchError_UnderRepeatedFailure asserts:
// no matter how many times Dispatch fails, ARQ state is untouched — oldSeq
// remains in-flight, newSeq never appears in-flight, payload is unchanged.
// The retransmit is re-tryable (BC-2.02.005 no-orphan-state postcondition).
func TestProperty_NoOrphanStateOnDispatchError_UnderRepeatedFailure(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	const oldSeq uint32 = 10
	originalPayload := []byte("payload-under-test")
	a.EnqueueSend(oldSeq, originalPayload, now)

	sentinel := errors.New("dispatch failed")
	failing := func(_ []byte) error { return sentinel }

	sender := arqsend.New(a, envForTest())

	// Attempt many retransmits with increasing newSeqs; each fails.
	for i := 0; i < 100; i++ {
		newSeq := uint32(1000 + i)
		err := sender.Retransmit(oldSeq, newSeq, now, failing)
		if !errors.Is(err, sentinel) {
			t.Fatalf("iteration %d: err = %v, want wrap of sentinel", i, err)
		}

		if !a.InFlightContains(oldSeq) {
			t.Fatalf("iteration %d: oldSeq=%d was removed on failed dispatch (orphan-state violation)", i, oldSeq)
		}
		if a.InFlightContains(newSeq) {
			t.Fatalf("iteration %d: newSeq=%d became in-flight on failed dispatch (orphan-state violation)", i, newSeq)
		}
		// Payload for oldSeq must be byte-exact identical.
		got := a.PayloadForInFlight(oldSeq)
		if string(got) != string(originalPayload) {
			t.Fatalf("iteration %d: payload mutated after failed dispatch: got %q want %q", i, string(got), string(originalPayload))
		}
	}
}

// TestProperty_UnknownOldSeqIsIdempotent asserts that repeated Retransmit
// calls with an oldSeq that is not in-flight always return
// ErrSequenceNotInFlight and never mutate ARQ state — regardless of how
// many other seqs are in flight.
func TestProperty_UnknownOldSeqIsIdempotent(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Populate with a few in-flight seqs to make sure the sender searches
	// the right set.
	for _, seq := range []uint32{1, 2, 3, 5, 8, 13, 21} {
		a.EnqueueSend(seq, []byte(fmt.Sprintf("seq-%d", seq)), now)
	}
	sender := arqsend.New(a, envForTest())

	dispatchCalled := false
	dispatch := func(_ []byte) error {
		dispatchCalled = true
		return nil
	}

	for _, unknownSeq := range []uint32{0, 4, 99, 1000, 0xFFFFFFFF} {
		err := sender.Retransmit(unknownSeq, unknownSeq+1_000_000, now, dispatch)
		if !errors.Is(err, arqsend.ErrSequenceNotInFlight) {
			t.Fatalf("unknownSeq=%d: err = %v, want ErrSequenceNotInFlight", unknownSeq, err)
		}
	}
	if dispatchCalled {
		t.Fatalf("dispatch was called for an unknown oldSeq — the sender must reject before assembling wire bytes")
	}

	// All original in-flight seqs still present.
	for _, seq := range []uint32{1, 2, 3, 5, 8, 13, 21} {
		if !a.InFlightContains(seq) {
			t.Fatalf("seq=%d disappeared after unknown-seq retransmits (state corruption)", seq)
		}
	}
}

// TestProperty_SequenceMonotonicityAcrossChainedRetransmits asserts that
// chaining retransmits — where each retransmit's newSeq becomes the oldSeq
// of the next call — preserves byte-exact payload identity through
// arbitrary depth, and each dispatched wire frame's ChanSeq matches the
// newSeq passed in (BC-2.02.005 PC-5).
func TestProperty_SequenceMonotonicityAcrossChainedRetransmits(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	originalPayload := []byte("chain-me-many-times")
	a.EnqueueSend(1, originalPayload, now)

	var dispatchedSeqs []uint32
	dispatch := func(wire []byte) error {
		// Extract the channel-header ChanSeq (at offset 44+4..44+8 big-endian).
		if len(wire) < 44+8 {
			t.Fatalf("wire too short: %d bytes", len(wire))
		}
		chanSeq := uint32(wire[48])<<24 | uint32(wire[49])<<16 | uint32(wire[50])<<8 | uint32(wire[51])
		dispatchedSeqs = append(dispatchedSeqs, chanSeq)
		return nil
	}

	sender := arqsend.New(a, envForTest(), arqsend.WithChanID(0x00000001))

	// Chain: 1 → 100 → 200 → 300 → 400 → 500.
	chain := []uint32{1, 100, 200, 300, 400, 500}
	for i := 0; i+1 < len(chain); i++ {
		oldSeq, newSeq := chain[i], chain[i+1]
		if err := sender.Retransmit(oldSeq, newSeq, now, dispatch); err != nil {
			t.Fatalf("chain step %d→%d: %v", oldSeq, newSeq, err)
		}
	}

	// Each dispatched wire frame's ChanSeq must match the corresponding
	// newSeq in monotonic order.
	if len(dispatchedSeqs) != len(chain)-1 {
		t.Fatalf("dispatched %d frames, want %d", len(dispatchedSeqs), len(chain)-1)
	}
	for i, want := range chain[1:] {
		if dispatchedSeqs[i] != want {
			t.Fatalf("dispatched[%d].ChanSeq = %d, want %d (monotonic chain)", i, dispatchedSeqs[i], want)
		}
	}

	// After the full chain, only the terminal seq is in flight.
	terminal := chain[len(chain)-1]
	for _, seq := range chain[:len(chain)-1] {
		if a.InFlightContains(seq) {
			t.Fatalf("intermediate seq=%d still in flight after chain complete", seq)
		}
	}
	if !a.InFlightContains(terminal) {
		t.Fatalf("terminal seq=%d not in flight after chain complete", terminal)
	}

	// Terminal seq's payload byte-equals the original.
	got := a.PayloadForInFlight(terminal)
	if string(got) != string(originalPayload) {
		t.Fatalf("payload mutated across chain: got %q want %q", string(got), string(originalPayload))
	}
}

// TestProperty_GapWalkTerminatesInBoundedSteps asserts the termination
// property of walking GapsToRetransmit: for each *initially* observed gap,
// after one successful Retransmit that gap is gone, and repeat calls with
// the same oldSeq return ErrSequenceNotInFlight (so a caller re-walking the
// gap list cannot loop over the same seq).
//
// Note: Retransmit's QUIC-model transition (EnqueueSend(newSeq) then
// RemoveInFlight(oldSeq)) means the *set* of gaps under a fixed (ackSeq,
// sackBitmap) is not monotonically shrinking — a new gap may be introduced
// at newSeq. Termination is over the *initial* gap set: each of those
// individual seqs is retired in one step.
func TestProperty_GapWalkTerminatesInBoundedSteps(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Enqueue seqs 1..20.
	for seq := uint32(1); seq <= 20; seq++ {
		a.EnqueueSend(seq, []byte(fmt.Sprintf("payload-%d", seq)), now)
	}

	// ackSeq=0, empty SACK — every enqueued seq is a gap.
	var sackBitmap [arq.SACKBitmapBytes]byte
	ackSeq := uint32(0)

	initialGaps := a.GapsToRetransmit(ackSeq, sackBitmap)
	if len(initialGaps) == 0 {
		t.Skip("initial gap count was zero; test needs seed data producing gaps")
	}

	sender := arqsend.New(a, envForTest())
	dispatch := func(_ []byte) error { return nil }

	// For each initial gap, one Retransmit must succeed AND a subsequent
	// Retransmit with the same oldSeq must return ErrSequenceNotInFlight.
	// This is the termination proof: no oldSeq can be retransmitted twice.
	newSeqCounter := uint32(10_000)
	for _, oldSeq := range initialGaps {
		newSeqCounter++
		if err := sender.Retransmit(oldSeq, newSeqCounter, now, dispatch); err != nil {
			t.Fatalf("first Retransmit oldSeq=%d: %v", oldSeq, err)
		}
		if a.InFlightContains(oldSeq) {
			t.Fatalf("oldSeq=%d still in flight after successful retransmit", oldSeq)
		}
		// Second call must reject — the gap is retired.
		newSeqCounter++
		err := sender.Retransmit(oldSeq, newSeqCounter, now, dispatch)
		if !errors.Is(err, arqsend.ErrSequenceNotInFlight) {
			t.Fatalf("second Retransmit oldSeq=%d: err=%v, want ErrSequenceNotInFlight (gap-walk termination violated)", oldSeq, err)
		}
	}

	// Every initial gap is now retired — GapsToRetransmit with the same
	// (ackSeq, sackBitmap) may return new seqs (the newSeqCounter values)
	// but must not contain any of the initial ones.
	remaining := a.GapsToRetransmit(ackSeq, sackBitmap)
	initialSet := make(map[uint32]bool, len(initialGaps))
	for _, s := range initialGaps {
		initialSet[s] = true
	}
	for _, s := range remaining {
		if initialSet[s] {
			t.Fatalf("initial gap seq=%d still appears in GapsToRetransmit after retransmit — retirement violated", s)
		}
	}
}

// TestProperty_DispatchErrorTaxonomy asserts that dispatch errors are
// wrapped consistently so callers can errors.Is / errors.Unwrap them, and
// that no dispatch error corrupts the outerassembler.Envelope or ARQ handle
// held by the Retransmitter.
func TestProperty_DispatchErrorTaxonomy(t *testing.T) {
	t.Parallel()

	a := arq.New(arq.Config{DropTimeout: time.Second})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	const oldSeq uint32 = 7
	a.EnqueueSend(oldSeq, []byte("data"), now)

	sender := arqsend.New(a, envForTest())

	errs := []error{
		errors.New("net: connection reset"),
		errors.New("multipath: no available path"),
		fmt.Errorf("wrapped: %w", errors.New("inner")),
	}
	for i, sentinel := range errs {
		s := sentinel
		err := sender.Retransmit(oldSeq, 100+uint32(i), now, func(_ []byte) error { return s })
		if !errors.Is(err, s) {
			t.Fatalf("iteration %d: errors.Is(err, sentinel) = false; err=%v", i, err)
		}
		// State preserved after every failure.
		if !a.InFlightContains(oldSeq) {
			t.Fatalf("iteration %d: oldSeq disappeared after failed dispatch", i)
		}
	}

	// After all failures, a successful dispatch still works — the
	// Retransmitter is not sticky-failed.
	successCalled := false
	dispatch := func(_ []byte) error {
		successCalled = true
		return nil
	}
	if err := sender.Retransmit(oldSeq, 9999, now, dispatch); err != nil {
		t.Fatalf("post-error success dispatch: %v", err)
	}
	if !successCalled {
		t.Fatalf("success dispatch was never called")
	}
	if a.InFlightContains(oldSeq) {
		t.Fatalf("oldSeq still in flight after successful dispatch")
	}
	if !a.InFlightContains(9999) {
		t.Fatalf("newSeq=9999 not in flight after successful dispatch")
	}
}

// TestProperty_ChanSeqNeverEqualsOldSeq_BC205_PC5 asserts that the ChanSeq
// stamped in the wire header always equals newSeq — never oldSeq — even
// when oldSeq and newSeq differ only by 1 (the "off-by-one" edge). This is
// BC-2.02.005 PC-5 in property form.
func TestProperty_ChanSeqNeverEqualsOldSeq_BC205_PC5(t *testing.T) {
	t.Parallel()

	// Try many (oldSeq, newSeq) pairs including the adjacent-integer case
	// which is the classic swap/mis-stamp bug.
	cases := []struct{ oldSeq, newSeq uint32 }{
		{1, 2},
		{100, 101},
		{0xFFFFFFFE, 0xFFFFFFFF},
		{0xDEADBEEF, 0xCAFEBABE},
		{42, 4200},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%d_to_%d", tc.oldSeq, tc.newSeq), func(t *testing.T) {
			t.Parallel()
			a := arq.New(arq.Config{DropTimeout: time.Second})
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			a.EnqueueSend(tc.oldSeq, []byte("pc5-check"), now)

			var got uint32
			dispatch := func(wire []byte) error {
				// ChanSeq at offset 44+4..44+8.
				got = uint32(wire[48])<<24 | uint32(wire[49])<<16 | uint32(wire[50])<<8 | uint32(wire[51])
				return nil
			}
			sender := arqsend.New(a, envForTest())
			if err := sender.Retransmit(tc.oldSeq, tc.newSeq, now, dispatch); err != nil {
				t.Fatalf("Retransmit: %v", err)
			}
			if got != tc.newSeq {
				t.Fatalf("wire ChanSeq = %d, want newSeq=%d (BC-2.02.005 PC-5 violation)", got, tc.newSeq)
			}
			if got == tc.oldSeq {
				t.Fatalf("wire ChanSeq = oldSeq=%d (retransmit is stamped with WRONG seq)", tc.oldSeq)
			}
		})
	}
	// Referenced only so the import-check does not complain if we ever
	// need to touch the assembler; the ChanSeq assertion above hard-decodes
	// the offset per the wire layout, which is the point.
	_ = outerassembler.ChannelHeaderFixedSize
}
