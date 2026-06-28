// Package replay in-package tests for wraparound and bounded-state invariants.
// These tests require access to unexported fields (nextSeq, pending, seen) and
// therefore live in the internal package rather than the external replay_test package.
package replay

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// F-001 RED TEST — uint32 sequence wraparound silently discards in-window frame
// BC-2.02.004 invariant 2 (in-order recovery) / VP-023
// ---------------------------------------------------------------------------

// TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered is the RED Gate test for
// finding F-001 (pass-2 adversarial review, S-4.02).
//
// The bug: in OnUpstream, the window-upper-bound check
//
//	seq < r.nextSeq + r.windowSize   (replay.go:125)
//
// uses plain uint32 addition. When nextSeq is within windowSize of
// math.MaxUint32, the addition overflows to a small number. A legitimately
// in-window future frame (e.g. nextSeq = MaxUint32-1, windowSize = 5,
// incoming seq = MaxUint32 = nextSeq+1) satisfies the in-window definition
// conceptually, but the overflow makes `seq < small_number` false, so the
// frame is dropped instead of buffered — breaking VP-023 (in-order delivery)
// at the wrap boundary.
//
// Test scenario (windowSize=5):
//  1. Seed nextSeq = math.MaxUint32 - 1 directly (in-package access).
//  2. Send seq = MaxUint32 (= nextSeq + 1, legitimately in-window).
//     Correct impl: buffered. Buggy impl: silently discarded.
//  3. Send seq = MaxUint32 - 1 (the filling frame, == nextSeq).
//     Correct impl: delivers MaxUint32-1 then drains MaxUint32.
//     Buggy impl: delivers only MaxUint32-1 (MaxUint32 was dropped).
//
// Note: wrapping past seq=0 is not exercised here because seq=0 is the
// documented discard sentinel; the wrap scenario is kept within [1, MaxUint32].
//
// This test MUST FAIL against the current implementation (Red Gate).
// It exercises VP-023 and BC-2.02.004 invariant 2.
func TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered(t *testing.T) {
	t.Parallel()

	const windowSize = 5

	var delivered []Frame
	deliver := func(f Frame) {
		delivered = append(delivered, f)
	}

	r := New(windowSize, deliver)

	// Seed nextSeq near the uint32 boundary using direct field access.
	// This mirrors the technique in internal/halfchannel/wraparound_test.go.
	// nextSeq = MaxUint32 - 1: the next expected frame is MaxUint32-1.
	r.nextSeq = math.MaxUint32 - 1

	// Step 1: send the in-window future frame (seq = MaxUint32 = nextSeq + 1).
	// With windowSize=5 the in-window upper bound is nextSeq+windowSize-1 =
	// MaxUint32+3 which overflows. The correct semantic bound is nextSeq+4.
	// Conceptually MaxUint32 is within [nextSeq+1, nextSeq+4], so it must be
	// buffered, not discarded.
	futureSeq := uint32(math.MaxUint32) // = nextSeq + 1
	if err := r.OnUpstream(Frame{Seq: futureSeq, Payload: []byte("future")}); err != nil {
		t.Fatalf("in-window future frame (seq=%d): unexpected error %v", futureSeq, err)
	}
	// Nothing delivered yet — futureSeq was not nextSeq, must be buffered.
	if len(delivered) != 0 {
		t.Fatalf("after buffering future frame: expected 0 deliveries, got %d (seqs: %v)",
			len(delivered), seqsFromFrames(delivered))
	}

	// Step 2: send the filling frame (seq = nextSeq = MaxUint32 - 1).
	// This delivers MaxUint32-1 in order, then drains the pending MaxUint32.
	fillerSeq := uint32(math.MaxUint32 - 1) // == r.nextSeq at this point
	if err := r.OnUpstream(Frame{Seq: fillerSeq, Payload: []byte("filler")}); err != nil {
		t.Fatalf("filling frame (seq=%d): unexpected error %v", fillerSeq, err)
	}

	// Assert: both frames delivered in order — filler first, then future.
	if len(delivered) != 2 {
		t.Fatalf("after filling gap: expected 2 deliveries, got %d (seqs: %v)",
			len(delivered), seqsFromFrames(delivered))
	}
	if delivered[0].Seq != fillerSeq {
		t.Errorf("delivery[0]: got seq %d, want %d (filler)", delivered[0].Seq, fillerSeq)
	}
	if delivered[1].Seq != futureSeq {
		t.Errorf("delivery[1]: got seq %d, want %d (future/buffered)", delivered[1].Seq, futureSeq)
	}
}

// seqsFromFrames extracts sequence numbers from a frame slice for diagnostic output.
func seqsFromFrames(frames []Frame) []uint32 {
	out := make([]uint32, len(frames))
	for i, f := range frames {
		out[i] = f.Seq
	}
	return out
}

// ---------------------------------------------------------------------------
// F-005 GUARD TEST — bounded internal state under sustained far-future traffic
// BC-2.02.004 invariant 3 / PC5 (DoS resistance)
// ---------------------------------------------------------------------------

// TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap is a regression guard
// for BC-2.02.004 invariant 3 and PC5 (DoS resistance / bounded memory).
//
// This test PASSES against the current implementation — it pins the existing
// correct bounding behaviour as a regression guard. If a future refactor
// accidentally lets the pending or seen maps grow unboundedly, this test catches
// it before it ships.
//
// Scenario: seq=1 is delivered (nextSeq→2); seq=2 never arrives (permanent gap).
// A continuous stream of mixed traffic then arrives:
//   - In-window future frames (nextSeq+1 … nextSeq+windowSize-1) — must buffer
//   - Far-future frames (nextSeq+windowSize … nextSeq+windowSize*2) — must discard
//
// At every point the invariants are:
//
//	len(pending) <= windowSize - 1   (at most windowSize-1 buffered future frames)
//	len(seen)    <= windowSize       (seen set is bounded by eviction)
//
// We verify these bounds hold after every single OnUpstream call.
func TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap(t *testing.T) {
	t.Parallel()

	const windowSize = uint32(5)

	var delivered []Frame
	deliver := func(f Frame) {
		delivered = append(delivered, f)
	}

	r := New(windowSize, deliver)

	// Deliver seq=1 to establish state. nextSeq is now 2.
	if err := r.OnUpstream(Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1: %v", err)
	}

	checkBounds := func(label string) {
		t.Helper()
		pendingLen := len(r.pending)
		seenLen := len(r.seen)
		// pending holds at most windowSize-1 buffered future frames
		// (the slot at nextSeq itself is not pending — it triggers immediate delivery).
		if uint32(pendingLen) > windowSize-1 {
			t.Errorf("%s: len(pending)=%d exceeds windowSize-1=%d (BC-2.02.004 invariant 3 / PC5 violation)",
				label, pendingLen, windowSize-1)
		}
		// seen holds at most windowSize entries (eviction keeps it bounded).
		if uint32(seenLen) > windowSize {
			t.Errorf("%s: len(seen)=%d exceeds windowSize=%d (seen set unbounded, eviction broken)",
				label, seenLen, windowSize)
		}
	}

	// seq=2 never arrives; drive a sustained stream of in-window and far-future frames.
	// We rotate through the same range repeatedly to simulate a long-running stream.
	const rounds = 100
	for round := 0; round < rounds; round++ {
		nextSeq := r.nextSeq // should remain 2 throughout (gap never filled)

		// In-window frames: [nextSeq+1 .. nextSeq+windowSize-1]
		for offset := uint32(1); offset < windowSize; offset++ {
			seq := nextSeq + offset
			if seq == 0 {
				// seq=0 is the discard sentinel — skip it.
				continue
			}
			if err := r.OnUpstream(Frame{Seq: seq}); err != nil {
				t.Fatalf("round %d in-window seq=%d: unexpected error %v", round, seq, err)
			}
			checkBounds("after in-window seq=" + uint32ToStr(seq))
		}

		// Far-future frames: [nextSeq+windowSize .. nextSeq+windowSize*2]
		for offset := windowSize; offset <= windowSize*2; offset++ {
			seq := nextSeq + offset
			if seq == 0 {
				continue
			}
			if err := r.OnUpstream(Frame{Seq: seq}); err != nil {
				t.Fatalf("round %d far-future seq=%d: unexpected error %v", round, seq, err)
			}
			checkBounds("after far-future seq=" + uint32ToStr(seq))
		}
	}

	// Verify nextSeq is still 2 — the gap was never filled.
	if r.nextSeq != 2 {
		t.Errorf("nextSeq: got %d, want 2 (gap at seq=2 should remain unfilled)", r.nextSeq)
	}

	// No frames beyond seq=1 should have been delivered (gap blocks everything).
	if len(delivered) != 1 || delivered[0].Seq != 1 {
		t.Errorf("delivered: got %v, want [seq=1] only", seqsFromFrames(delivered))
	}
}

// uint32ToStr is a minimal uint32→string converter used only for diagnostic
// labels inside this test file. Avoids importing strconv/fmt at package level
// for a test-only helper.
func uint32ToStr(n uint32) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
