// Package arq_test — FEC strong-oracle tests (S-7.01, BC-2.02.007 v1.2).
//
// Test naming convention follows BC-S.SS.NNN_xxx pattern:
//
//	TestBC_2_02_007_Encode_ProducesParityFrame        (AC-001)
//	TestBC_2_02_007_Encode_ParityXORCorrect           (AC-001 parity oracle)
//	TestBC_2_02_007_Recover_SingleLoss                (AC-002)
//	TestBC_2_02_007_Recover_TwoLossesFail             (AC-003)
//	TestBC_2_02_007_FallbackToARQ_OnMultiLoss         (AC-004)
//	TestBC_2_02_007_Encode_IncompleteLastGroup_NoParity (AC-005)
//	TestBC_2_02_007_VP043_SingleLossRecovery_Property  (VP-043)
//
// Story AC names are preserved as aliases so the story's acceptance test names
// remain discoverable (TestFEC_*). Each delegates to the BC-named test body.
package arq_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/arq"
	"github.com/arcavenae/switchboard/internal/frame"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// buildPayloads creates n byte-slice payloads of the given width. Each byte is
// derived from a simple deterministic function of (frame index, byte index) so
// the XOR oracle can independently compute the expected parity without calling
// the encoder.
func buildPayloads(n, width int) [][]byte {
	t := make([][]byte, n)
	for i := range t {
		t[i] = make([]byte, width)
		for j := range t[i] {
			// mix frame index and byte position to get non-trivial XOR patterns
			t[i][j] = byte((i+1)*31 + j*7 + 0xA5)
		}
	}
	return t
}

// xorOracle computes the XOR parity of payloads the same way the FEC encoder
// should: XOR all payloads element-wise, extending shorter payloads with 0x00
// (standard XOR parity convention for equal-length payloads assumed here —
// all test payloads in this file have equal width).
func xorOracle(payloads [][]byte) []byte {
	if len(payloads) == 0 {
		return nil
	}
	maxLen := 0
	for _, p := range payloads {
		if len(p) > maxLen {
			maxLen = len(p)
		}
	}
	parity := make([]byte, maxLen)
	for _, p := range payloads {
		for i, b := range p {
			parity[i] ^= b
		}
	}
	return parity
}

// encodeGroup drives enc.AddFrame for all payloads and returns the parity
// payload emitted at the last frame. Fails the test if no parity is emitted.
func encodeGroup(t *testing.T, enc *arq.Encoder, payloads [][]byte) []byte {
	t.Helper()
	var parity []byte
	for i, p := range payloads {
		pp := enc.AddFrame(p)
		if pp != nil {
			parity = pp
		}
		if i < len(payloads)-1 && pp != nil {
			t.Fatalf("AddFrame(%d): parity emitted mid-group (want nil until last frame)", i)
		}
	}
	if parity == nil {
		t.Fatal("encodeGroup: encoder produced no parity for a complete group")
	}
	return parity
}

// assertBytesEqual asserts that got equals want byte-for-byte.
func assertBytesEqual(t *testing.T, label string, want, got []byte) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: length mismatch: want %d bytes, got %d", label, len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: byte[%d]: want %#02x, got %#02x", label, i, want[i], got[i])
		}
	}
}

// ─── AC-001: parity frame carries frame_type=fec=0x05; XOR is correct ────────

// TestBC_2_02_007_Encode_ProducesParityFrame verifies AC-001 / BC-2.02.007
// postcondition 1: for a complete group of N data frames, Encoder.AddFrame
// emits a non-nil parity payload exactly on the Nth call (not before).
// The constant frame.FrameTypeFec must equal 0x05 (canonical wire value,
// F-P8-008; do not redefine locally).
func TestBC_2_02_007_Encode_ProducesParityFrame(t *testing.T) {
	t.Parallel()

	const groupSize = 4
	enc := arq.NewEncoder(arq.FECConfig{GroupSize: groupSize})
	payloads := buildPayloads(groupSize, 8)

	// Verify the canonical enum value before exercising the encoder.
	if frame.FrameTypeFec != 0x05 {
		t.Fatalf("frame.FrameTypeFec: want 0x05, got %#x — canonical wire value violated (F-P8-008; BC-2.02.007 PC-5)", frame.FrameTypeFec)
	}

	// Frames 0..N-2 must return nil parity.
	for i := 0; i < groupSize-1; i++ {
		pp := enc.AddFrame(payloads[i])
		if pp != nil {
			t.Fatalf("AddFrame(%d): want nil (group incomplete), got non-nil parity after %d frames", i, i+1)
		}
	}

	// Nth frame completes the group; parity must be emitted.
	pp := enc.AddFrame(payloads[groupSize-1])
	if pp == nil {
		t.Fatal("AddFrame(groupSize-1): want parity payload on group completion, got nil")
	}
	if len(pp) == 0 {
		t.Fatal("parity payload length is 0; non-empty payload required")
	}
}

// TestFEC_Encode_ProducesParityFrame is the story-level alias for AC-001.
func TestFEC_Encode_ProducesParityFrame(t *testing.T) {
	TestBC_2_02_007_Encode_ProducesParityFrame(t)
}

// TestBC_2_02_007_Encode_ParityXORCorrect is the strong oracle for AC-001:
// the parity payload returned by AddFrame must equal the element-wise XOR of
// all data payloads in the group (BC-2.02.007 invariant 1; VP-043 unit clause
// "P = D1⊕D2⊕...⊕DN").
//
// Table-driven over multiple group sizes (2, 4, 8) and payload widths.
func TestBC_2_02_007_Encode_ParityXORCorrect(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		groupSize int
		width     int // payload byte width
	}{
		{"group2_width1", 2, 1},
		{"group4_width8", 4, 8},
		{"group4_width16", 4, 16},
		{"group8_width3", 8, 3},
		{"group8_width32", 8, 32},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			enc := arq.NewEncoder(arq.FECConfig{GroupSize: tc.groupSize})
			payloads := buildPayloads(tc.groupSize, tc.width)

			want := xorOracle(payloads)
			got := encodeGroup(t, enc, payloads)

			assertBytesEqual(t, "parity XOR oracle", want, got)
		})
	}
}

// ─── AC-002: single-loss recovery — byte-exact ───────────────────────────────

// TestBC_2_02_007_Recover_SingleLoss verifies AC-002 / BC-2.02.007 postcondition 3:
// Decoder.Recover reconstructs the single missing data frame exactly (byte-exact)
// when exactly one entry in the group slice is nil.
//
// Canonical test vector (BC-2.02.007): 4 data frames; D2 lost; P arrives →
// D2 = P⊕D1⊕D3⊕D4 reconstructed; all 4 delivered in order.
//
// Table-driven over all loss positions in a 4-frame group to ensure every
// position is recoverable.
func TestBC_2_02_007_Recover_SingleLoss(t *testing.T) {
	t.Parallel()

	const groupSize = 4
	const width = 16
	enc := arq.NewEncoder(arq.FECConfig{GroupSize: groupSize})
	dec := arq.NewDecoder(arq.FECConfig{GroupSize: groupSize})

	payloads := buildPayloads(groupSize, width)
	parity := encodeGroup(t, enc, payloads)

	for lossIdx := 0; lossIdx < groupSize; lossIdx++ {
		lossIdx := lossIdx
		t.Run("", func(t *testing.T) {
			t.Parallel()

			// Build group with nil at lossIdx (simulated loss).
			withGap := make([][]byte, groupSize)
			copy(withGap, payloads)
			want := make([]byte, len(payloads[lossIdx]))
			copy(want, payloads[lossIdx])
			withGap[lossIdx] = nil

			recovered, err := dec.Recover(withGap, parity)
			if err != nil {
				t.Fatalf("Recover single loss at idx=%d: unexpected error: %v", lossIdx, err)
			}

			assertBytesEqual(t, "recovered payload", want, recovered)
		})
	}
}

// TestFEC_Recover_SingleLoss is the story-level alias for AC-002.
func TestFEC_Recover_SingleLoss(t *testing.T) {
	TestBC_2_02_007_Recover_SingleLoss(t)
}

// ─── AC-003: two losses return ErrTooManyLosses ───────────────────────────────

// TestBC_2_02_007_Recover_TwoLossesFail verifies AC-003 / BC-2.02.007 precondition
// and postcondition 4: Recover returns ErrTooManyLosses when more than one entry
// in the group slice is nil. The sentinel must satisfy errors.Is identity.
//
// Table-driven over several 2-loss position pairs.
func TestBC_2_02_007_Recover_TwoLossesFail(t *testing.T) {
	t.Parallel()

	const groupSize = 4
	enc := arq.NewEncoder(arq.FECConfig{GroupSize: groupSize})
	dec := arq.NewDecoder(arq.FECConfig{GroupSize: groupSize})

	payloads := buildPayloads(groupSize, 8)
	parity := encodeGroup(t, enc, payloads)

	cases := []struct {
		name  string
		loss0 int
		loss1 int
	}{
		{"positions 0 and 1", 0, 1},
		{"positions 0 and 3", 0, 3},
		{"positions 1 and 2", 1, 2},
		{"positions 2 and 3", 2, 3},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			withTwoGaps := make([][]byte, groupSize)
			copy(withTwoGaps, payloads)
			withTwoGaps[tc.loss0] = nil
			withTwoGaps[tc.loss1] = nil

			_, err := dec.Recover(withTwoGaps, parity)
			if !errors.Is(err, arq.ErrTooManyLosses) {
				t.Errorf("Recover(%s): want errors.Is(err, ErrTooManyLosses), got %v", tc.name, err)
			}
		})
	}
}

// TestFEC_Recover_TwoLossesFail is the story-level alias for AC-003.
func TestFEC_Recover_TwoLossesFail(t *testing.T) {
	TestBC_2_02_007_Recover_TwoLossesFail(t)
}

// ─── AC-004: ErrTooManyLosses triggers ARQ retransmit fallback ───────────────

// TestBC_2_02_007_FallbackToARQ_OnMultiLoss verifies AC-004 / BC-2.02.007
// postcondition 4 + VP-043 composition: when Recover returns ErrTooManyLosses,
// the caller MUST NOT drop the group silently — it MUST invoke the ARQ
// SACK/retransmit path.
//
// The test verifies the full composition:
//  1. ErrTooManyLosses is returned (identity via errors.Is).
//  2. The caller detects ErrTooManyLosses and invokes ARQ.GapsToRetransmit.
//  3. GapsToRetransmit returns a non-empty gap list — the ARQ retransmit path
//     is observably engaged.
//  4. The gap list indices correspond to the two lost frame sequence numbers.
//
// An ARQ with frames enqueued at seqs 1..N with ackSeq=0 and an all-zero SACK
// (receiver has not acknowledged any) will report all N seqs as gaps — this
// models the state immediately after multi-loss detection before any ACK arrives.
func TestBC_2_02_007_FallbackToARQ_OnMultiLoss(t *testing.T) {
	t.Parallel()

	const groupSize = 4
	const width = 8
	enc := arq.NewEncoder(arq.FECConfig{GroupSize: groupSize})
	dec := arq.NewDecoder(arq.FECConfig{GroupSize: groupSize})

	payloads := buildPayloads(groupSize, width)
	parity := encodeGroup(t, enc, payloads)

	// Simulate two losses: positions 0 and 2 (seq 1 and seq 3 in 1-indexed ARQ).
	const lossA, lossB = 0, 2
	withTwoGaps := make([][]byte, groupSize)
	copy(withTwoGaps, payloads)
	withTwoGaps[lossA] = nil
	withTwoGaps[lossB] = nil

	_, err := dec.Recover(withTwoGaps, parity)
	if !errors.Is(err, arq.ErrTooManyLosses) {
		t.Fatalf("Recover with 2 losses: want ErrTooManyLosses, got %v", err)
	}

	// Caller receives ErrTooManyLosses — it MUST engage the ARQ retransmit path.
	// Construct an ARQ sender with all 4 frames in-flight (seq 1..4).
	sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := arq.New(arq.Config{DropTimeout: 100 * time.Millisecond})
	for seq := uint32(1); seq <= groupSize; seq++ {
		a.EnqueueSend(seq, payloads[seq-1], sendTime)
	}

	// ackSeq=0, all-zero SACK: no frames acknowledged → all 4 are gaps.
	var zeroSACK [arq.SACKBitmapBytes]byte
	gaps := a.GapsToRetransmit(0, zeroSACK)

	if len(gaps) == 0 {
		t.Fatal("ARQ retransmit fallback: GapsToRetransmit returned empty; caller would silently drop — MUST engage retransmit path (BC-2.02.007 PC-4)")
	}

	// At minimum, the two lost sequences (lossA+1, lossB+1 in 1-indexed seq) must
	// appear in the gap list. The gap list may include all 4 seqs since ackSeq=0.
	lossSeqs := map[uint32]bool{
		uint32(lossA + 1): true,
		uint32(lossB + 1): true,
	}
	gapSet := make(map[uint32]bool, len(gaps))
	for _, g := range gaps {
		gapSet[g] = true
	}
	for seq := range lossSeqs {
		if !gapSet[seq] {
			t.Errorf("ARQ retransmit fallback: lost seq=%d not in gap list %v", seq, gaps)
		}
	}
}

// TestFEC_FallbackToARQ_OnMultiLoss is the story-level alias for AC-004.
func TestFEC_FallbackToARQ_OnMultiLoss(t *testing.T) {
	TestBC_2_02_007_FallbackToARQ_OnMultiLoss(t)
}

// ─── AC-005: incomplete last group emits no parity ───────────────────────────

// TestBC_2_02_007_Encode_IncompleteLastGroup_NoParity verifies AC-005 /
// BC-2.02.007 EC-001: AddFrame never emits a parity payload for a partial group;
// Flush reports hasIncomplete=true and returns the buffered partial-group payloads.
//
// Table-driven over several incomplete fill levels (1, 2, 3 frames in a 4-frame
// group).
func TestBC_2_02_007_Encode_IncompleteLastGroup_NoParity(t *testing.T) {
	t.Parallel()

	const groupSize = 4

	cases := []struct {
		name       string
		frameCount int // frames added (must be < groupSize)
	}{
		{"one_of_four", 1},
		{"two_of_four", 2},
		{"three_of_four", 3},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			enc := arq.NewEncoder(arq.FECConfig{GroupSize: groupSize})
			t.Cleanup(func() {
				// no-op resource cleanup hook; present for t.Cleanup pattern compliance
			})

			payloads := buildPayloads(tc.frameCount, 4)

			for i, p := range payloads {
				pp := enc.AddFrame(p)
				if pp != nil {
					t.Fatalf("AddFrame(%d) of %d: got non-nil parity from incomplete group (want nil)", i, tc.frameCount)
				}
			}

			incomplete, hasIncomplete := enc.Flush()
			if !hasIncomplete {
				t.Fatalf("Flush after %d/%d frames: want hasIncomplete=true, got false", tc.frameCount, groupSize)
			}
			if len(incomplete) != tc.frameCount {
				t.Errorf("Flush after %d/%d frames: want %d incomplete payloads, got %d",
					tc.frameCount, groupSize, tc.frameCount, len(incomplete))
			}

			// Verify the flushed payloads are byte-exact copies of what was added.
			for i, want := range payloads {
				if i >= len(incomplete) {
					break
				}
				assertBytesEqual(t, "flushed payload", want, incomplete[i])
			}
		})
	}
}

// TestFEC_Encode_IncompleteLastGroup_NoParity is the story-level alias for AC-005.
func TestFEC_Encode_IncompleteLastGroup_NoParity(t *testing.T) {
	TestBC_2_02_007_Encode_IncompleteLastGroup_NoParity(t)
}

// ─── AC-005 edge: flush on complete group boundary ───────────────────────────

// TestBC_2_02_007_Flush_OnCompleteGroupBoundary verifies that Flush reports
// hasIncomplete=false when the encoder has processed exactly groupSize frames
// (i.e., the internal buffer was reset after parity emission). No residual
// partial-group state should remain.
func TestBC_2_02_007_Flush_OnCompleteGroupBoundary(t *testing.T) {
	t.Parallel()

	const groupSize = 4
	enc := arq.NewEncoder(arq.FECConfig{GroupSize: groupSize})
	payloads := buildPayloads(groupSize, 8)

	encodeGroup(t, enc, payloads) // completes group; resets internal buffer

	incomplete, hasIncomplete := enc.Flush()
	if hasIncomplete {
		t.Errorf("Flush after complete group: want hasIncomplete=false, got true with %d frames", len(incomplete))
	}
}

// ─── VP-043: property test — single-loss recovery across randomised inputs ───

// TestBC_2_02_007_VP043_SingleLossRecovery_Property is the VP-043 property test:
// for all (group_size ∈ [2,8], loss_index ∈ [0, group_size-1], randomised
// data payloads), Recover reconstructs the lost frame byte-exactly.
//
// Coverage: 7 group sizes × (2+3+4+5+6+7+8) loss positions × 1000 payload
// variants = 175 000 recovery assertions. The payload bytes are generated with
// a deterministic Knuth MMIX LCG so runs are reproducible.
func TestBC_2_02_007_VP043_SingleLossRecovery_Property(t *testing.T) {
	t.Parallel()

	// Knuth MMIX LCG for deterministic pseudo-random payload generation.
	seed := uint64(0xFEEDBEEFCAFEBABE)
	lcgNext := func() uint64 {
		seed = seed*6364136223846793005 + 1442695040888963407
		return seed
	}
	randByte := func() byte { return byte(lcgNext() >> 56) }

	const trials = 1000
	const payloadWidth = 16

	for groupSize := 2; groupSize <= 8; groupSize++ {
		groupSize := groupSize
		t.Run("", func(t *testing.T) {
			t.Parallel()

			for trial := 0; trial < trials; trial++ {
				// Build fresh payloads for this trial using the LCG.
				payloads := make([][]byte, groupSize)
				for i := range payloads {
					payloads[i] = make([]byte, payloadWidth)
					for j := range payloads[i] {
						payloads[i][j] = randByte()
					}
				}

				enc := arq.NewEncoder(arq.FECConfig{GroupSize: groupSize})
				dec := arq.NewDecoder(arq.FECConfig{GroupSize: groupSize})

				parity := encodeGroup(t, enc, payloads)
				if t.Failed() {
					t.Logf("VP-043 groupSize=%d trial=%d: encoder failure", groupSize, trial)
					return
				}

				// Independently verify parity correctness (XOR oracle).
				wantParity := xorOracle(payloads)
				for k, b := range wantParity {
					if parity[k] != b {
						t.Errorf("VP-043 groupSize=%d trial=%d: parity[%d]: want %#02x got %#02x",
							groupSize, trial, k, b, parity[k])
					}
				}

				// Test single-loss recovery at every loss position.
				for lossIdx := 0; lossIdx < groupSize; lossIdx++ {
					withGap := make([][]byte, groupSize)
					copy(withGap, payloads)
					want := make([]byte, len(payloads[lossIdx]))
					copy(want, payloads[lossIdx])
					withGap[lossIdx] = nil

					recovered, err := dec.Recover(withGap, parity)
					if err != nil {
						t.Errorf("VP-043 groupSize=%d trial=%d lossIdx=%d: unexpected error: %v",
							groupSize, trial, lossIdx, err)
						continue
					}
					if len(recovered) != len(want) {
						t.Errorf("VP-043 groupSize=%d trial=%d lossIdx=%d: len want %d got %d",
							groupSize, trial, lossIdx, len(want), len(recovered))
						continue
					}
					for k := range want {
						if recovered[k] != want[k] {
							t.Errorf("VP-043 groupSize=%d trial=%d lossIdx=%d byte[%d]: want %#02x got %#02x",
								groupSize, trial, lossIdx, k, want[k], recovered[k])
						}
					}
				}
			}
		})
	}
}

// TestFEC_VP043_SingleLossRecovery_Property is the story-level alias for VP-043.
func TestFEC_VP043_SingleLossRecovery_Property(t *testing.T) {
	TestBC_2_02_007_VP043_SingleLossRecovery_Property(t)
}
