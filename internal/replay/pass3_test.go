// Package replay in-package tests for pass-3 adversarial findings (S-4.02).
// These tests access unexported fields (nextSeq, pending, seen) and therefore
// live in the internal package rather than the external replay_test package.
package replay

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// F-001 RED TEST — lower-bound "too-old" guard is NOT wrap-safe
// BC-2.02.004 invariant 2 (in-order recovery)
// ---------------------------------------------------------------------------

// TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow is the RED Gate
// test for finding F-001, pass-3 adversarial review (S-4.02).
//
// The bug: replay.go line 112 uses a plain integer comparison
//
//	r.nextSeq > r.windowSize && seq < r.nextSeq-r.windowSize
//
// to guard against "too-old" frames. This check executes BEFORE the wrap-safe
// upper-bound distance check on line 133 and returns early (discards the frame)
// without evaluating whether the frame is actually in the forward window.
//
// Near the uint32 wraparound, an in-window FUTURE frame whose seq value has
// wrapped below nextSeq in the integer sense is wrongly classified as "too old"
// and silently discarded — breaking BC-2.02.004 invariant 2 (in-order recovery).
//
// Concrete scenario (windowSize=5):
//
//	nextSeq = MaxUint32 - 2
//	In-window future frame: seq = 1
//	  True forward distance: (1 - (MaxUint32-2)) mod 2^32
//	                        = 1 - MaxUint32 + 2 mod 2^32
//	                        = 4  (inside (0, windowSize))  ← MUST be buffered
//	  Bug: nextSeq > windowSize  → true
//	       seq < nextSeq-windowSize → 1 < MaxUint32-7 → true  ← wrongly discards
//
// Assertion strategy: after sending seq=1 with nextSeq seeded to MaxUint32-2,
// inspect r.pending[1] directly. A correct implementation buffers the frame
// (r.pending[1] is present). The buggy line-112 guard discards it before the
// wrap-safe distance check, so r.pending[1] is absent.
//
// Note on delivery path: completing in-order delivery of seq=1 from pending
// would require nextSeq to advance through MaxUint32 and then 0 (the discard
// sentinel), which is a separate design consideration. The Red assertion here
// is purely that seq=1 is BUFFERED, not that it is auto-delivered — buffering
// is the necessary precondition for eventual delivery and is sufficient to
// distinguish correct from buggy behaviour at this boundary.
//
// This test MUST FAIL against the current implementation (Red Gate):
// line 112 discards seq=1 before line 133 can buffer it.
func TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow(t *testing.T) {
	t.Parallel()

	const windowSize = uint32(5)

	var delivered []Frame
	deliver := func(f Frame) {
		delivered = append(delivered, f)
	}

	r := New(windowSize, deliver)

	// Seed nextSeq near the uint32 boundary using direct field access.
	// nextSeq = MaxUint32 - 2: the next expected in-order frame is MaxUint32-2.
	r.nextSeq = math.MaxUint32 - 2

	// Send the in-window future frame: seq=1.
	//   True forward distance: (1 - (MaxUint32-2)) mod 2^32 = 4 ∈ (0, 5).
	//   Correct: buffered in r.pending[1].
	//   Buggy:   discarded by line-112 guard (1 < MaxUint32-7).
	futureSeq := uint32(1)
	if err := r.OnUpstream(Frame{Seq: futureSeq, Payload: []byte("wrap-future")}); err != nil {
		t.Fatalf("in-window future frame seq=%d: unexpected error %v", futureSeq, err)
	}

	// Nothing must be delivered yet — seq=1 is ahead of nextSeq=MaxUint32-2.
	if len(delivered) != 0 {
		t.Fatalf("after sending future frame seq=1: expected 0 deliveries, got %d (seqs: %v)",
			len(delivered), seqsFromFrames(delivered))
	}

	// Critical Red assertion: the frame must be held in r.pending[1].
	// If line 112 discards it, r.pending[1] will be absent.
	if _, ok := r.pending[futureSeq]; !ok {
		t.Errorf("BC-2.02.004 invariant 2 violation: in-window post-wrap frame seq=%d "+
			"was discarded by the too-old guard instead of being buffered "+
			"(r.pending[%d] absent; line-112 lower-bound check is not wrap-safe)",
			futureSeq, futureSeq)
	}
}

// ---------------------------------------------------------------------------
// F-003 COVERAGE TEST — advancing-window bounded seen-set eviction
// BC-2.02.004 invariant 3 / PC5 (bounded memory)
// ---------------------------------------------------------------------------

// TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow verifies that
// len(r.seen) and len(r.pending) stay within their declared bounds as nextSeq
// advances far past windowSize, exercising the evictOldSeen delete path at
// replay.go line ≈180.
//
// The existing bounded-state test (TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap
// in wraparound_test.go) pins nextSeq=2 for its entire run — evictOldSeen is
// never called because nextSeq never exceeds windowSize. This test covers the
// complementary path: a long in-order stream that forces nextSeq to advance
// thousands of steps past windowSize, repeatedly triggering eviction.
//
// Invariants asserted after EVERY OnUpstream call:
//
//	len(r.seen)    <= windowSize      (BC-2.02.004 invariant 3 / PC5)
//	len(r.pending) <= windowSize - 1  (at most windowSize-1 bufferable futures)
//
// This test PASSES against the current implementation — it is a regression
// guard. If a future refactor accidentally breaks eviction under a sliding
// window (e.g. by removing the delete call in evictOldSeen), this test will
// catch it.
func TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow(t *testing.T) {
	t.Parallel()

	const (
		windowSize = uint32(8)
		// Deliver enough frames to advance nextSeq well past windowSize many
		// times over, exercising every eviction step.
		totalFrames = uint32(5000)
	)

	var delivered []Frame
	deliver := func(f Frame) {
		delivered = append(delivered, f)
	}

	r := New(windowSize, deliver)

	checkBounds := func(label string) {
		t.Helper()
		seenLen := uint32(len(r.seen))
		pendingLen := uint32(len(r.pending))
		if seenLen > windowSize {
			t.Errorf("%s: len(seen)=%d exceeds windowSize=%d "+
				"(BC-2.02.004 invariant 3 / PC5: seen set unbounded, eviction broken)",
				label, seenLen, windowSize)
		}
		if pendingLen > windowSize-1 {
			t.Errorf("%s: len(pending)=%d exceeds windowSize-1=%d "+
				"(BC-2.02.004 invariant 3 / PC5: pending map unbounded)",
				label, pendingLen, windowSize-1)
		}
	}

	// Deliver a long contiguous in-order stream seq=1..totalFrames.
	// Each call advances nextSeq by 1, triggering evictOldSeen on every step
	// once nextSeq > windowSize (which happens after the first windowSize frames).
	for seq := uint32(1); seq <= totalFrames; seq++ {
		if err := r.OnUpstream(Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: unexpected error %v", seq, err)
		}
		checkBounds(uint32ToStr(seq))
	}

	// Sanity: every frame was delivered in order.
	if uint32(len(delivered)) != totalFrames {
		t.Errorf("delivery count: got %d, want %d", len(delivered), totalFrames)
	}
	for i, f := range delivered {
		if f.Seq != uint32(i+1) {
			t.Errorf("delivery[%d]: got seq=%d, want %d", i, f.Seq, uint32(i+1))
			break
		}
	}
}
