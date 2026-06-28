package replay_test

import (
	"errors"
	"math/rand"
	"sort"
	"testing"

	"github.com/arcavenae/switchboard/internal/replay"
)

// newCollector returns a DeliverFunc and a pointer to the slice it appends to.
// t.Helper() is intentionally omitted here — it is only useful inside test
// helper functions called from test bodies, not in constructor helpers.
func newCollector() (replay.DeliverFunc, *[]replay.Frame) {
	var got []replay.Frame
	deliver := func(f replay.Frame) {
		got = append(got, f)
	}
	return deliver, &got
}

// mustNew constructs a Replay or calls t.Fatal if New panics.
func mustNew(t *testing.T, windowSize uint32, deliver replay.DeliverFunc) *replay.Replay {
	t.Helper()
	var r *replay.Replay
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("New panicked unexpectedly: %v", p)
			}
		}()
		r = replay.New(windowSize, deliver)
	}()
	return r
}

// assertDelivered checks that delivered contains exactly the frames with the
// given sequence numbers, in order.
func assertDelivered(t *testing.T, got []replay.Frame, wantSeqs []uint32) {
	t.Helper()
	if len(got) != len(wantSeqs) {
		t.Fatalf("deliver count: got %d, want %d (seqs delivered: %v, want: %v)",
			len(got), len(wantSeqs), seqsOf(got), wantSeqs)
	}
	for i, f := range got {
		if f.Seq != wantSeqs[i] {
			t.Errorf("delivery[%d]: got seq %d, want seq %d", i, f.Seq, wantSeqs[i])
		}
	}
}

func seqsOf(frames []replay.Frame) []uint32 {
	out := make([]uint32, len(frames))
	for i, f := range frames {
		out[i] = f.Seq
	}
	return out
}

// ---------------------------------------------------------------------------
// AC-001 / VP-022: no duplicate delivery
// BC-2.02.004 postcondition 2: "each keystroke is applied exactly once"
// ---------------------------------------------------------------------------

// TestReplay_NoDuplicateDelivery verifies BC-2.02.004 postcondition 2.
// Second delivery of the same seq MUST return ErrAlreadyDelivered.
// Exercises VP-022 (no double delivery).
func TestReplay_NoDuplicateDelivery(t *testing.T) {
	t.Parallel()

	deliver, got := newCollector()
	r := mustNew(t, 5, deliver)

	// First delivery — should succeed and call deliver.
	if err := r.OnUpstream(replay.Frame{Seq: 1, Payload: []byte("a")}); err != nil {
		t.Fatalf("first delivery of seq=1: unexpected error %v", err)
	}
	assertDelivered(t, *got, []uint32{1})

	// Second delivery — same seq — must return ErrAlreadyDelivered.
	err := r.OnUpstream(replay.Frame{Seq: 1, Payload: []byte("a")})
	if !errors.Is(err, replay.ErrAlreadyDelivered) {
		t.Fatalf("duplicate seq=1: got %v, want ErrAlreadyDelivered", err)
	}

	// deliver must not have been called again.
	assertDelivered(t, *got, []uint32{1})
}

// TestReplay_NoDuplicateDelivery_MultipleSeqs exercises VP-022 across several
// sequence numbers, each re-sent once.
func TestReplay_NoDuplicateDelivery_MultipleSeqs(t *testing.T) {
	t.Parallel()

	deliver, got := newCollector()
	r := mustNew(t, 10, deliver)

	// Deliver seqs 1–5 in order, then replay each.
	for seq := uint32(1); seq <= 5; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("first delivery seq=%d: %v", seq, err)
		}
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5})

	for seq := uint32(1); seq <= 5; seq++ {
		if !errors.Is(r.OnUpstream(replay.Frame{Seq: seq}), replay.ErrAlreadyDelivered) {
			t.Errorf("re-delivery of seq=%d should return ErrAlreadyDelivered", seq)
		}
	}
	// Still exactly 5 deliveries — no extras.
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5})
}

// ---------------------------------------------------------------------------
// AC-002 / VP-023: in-order delivery
// BC-2.02.004 postcondition 2: "in sequence order"
// ---------------------------------------------------------------------------

// TestReplay_InOrderDelivery verifies BC-2.02.004 postcondition 2.
// seq N+1 arriving before N must be buffered and delivered after N arrives,
// in order. Exercises VP-023 (in-order delivery).
func TestReplay_InOrderDelivery(t *testing.T) {
	t.Parallel()

	deliver, got := newCollector()
	r := mustNew(t, 5, deliver)

	// seq=2 arrives first — must be buffered, nothing delivered yet.
	if err := r.OnUpstream(replay.Frame{Seq: 2, Payload: []byte("b")}); err != nil {
		t.Fatalf("seq=2 ahead of seq=1: unexpected error %v", err)
	}
	assertDelivered(t, *got, []uint32{}) // nothing yet

	// seq=1 arrives — fills the gap; both 1 and 2 must be delivered in order.
	if err := r.OnUpstream(replay.Frame{Seq: 1, Payload: []byte("a")}); err != nil {
		t.Fatalf("seq=1: unexpected error %v", err)
	}
	assertDelivered(t, *got, []uint32{1, 2})
}

// TestReplay_InOrderDelivery_LongerGap exercises VP-023 with a run of
// out-of-order arrivals that are all buffered and then flushed in order.
func TestReplay_InOrderDelivery_LongerGap(t *testing.T) {
	t.Parallel()

	deliver, got := newCollector()
	r := mustNew(t, 10, deliver)

	// Deliver seq 1 first to establish the window.
	if err := r.OnUpstream(replay.Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1: %v", err)
	}

	// Now deliver seq 5, 4, 3 — all buffered.
	for _, seq := range []uint32{5, 4, 3} {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: unexpected error %v", seq, err)
		}
	}
	// Only seq=1 delivered so far.
	assertDelivered(t, *got, []uint32{1})

	// seq=2 arrives — drains 2, 3, 4, 5 in order.
	if err := r.OnUpstream(replay.Frame{Seq: 2}); err != nil {
		t.Fatalf("seq=2: %v", err)
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5})
}

// TestReplay_InOrderDelivery_TableDriven covers a range of arrival permutations.
func TestReplay_InOrderDelivery_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		arrivals []uint32 // order in which frames arrive
		wantSeqs []uint32 // expected delivery order
	}{
		{
			name:     "strict order",
			arrivals: []uint32{1, 2, 3, 4, 5},
			wantSeqs: []uint32{1, 2, 3, 4, 5},
		},
		{
			name:     "reverse order",
			arrivals: []uint32{5, 4, 3, 2, 1},
			wantSeqs: []uint32{1, 2, 3, 4, 5},
		},
		{
			name:     "interleaved",
			arrivals: []uint32{1, 3, 2, 5, 4},
			wantSeqs: []uint32{1, 2, 3, 4, 5},
		},
		{
			name:     "single",
			arrivals: []uint32{1},
			wantSeqs: []uint32{1},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deliver, got := newCollector()
			r := mustNew(t, 10, deliver)
			for _, seq := range tc.arrivals {
				if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
					t.Fatalf("seq=%d: unexpected error %v", seq, err)
				}
			}
			assertDelivered(t, *got, tc.wantSeqs)
		})
	}
}

// ---------------------------------------------------------------------------
// AC-003 / BC-2.02.004 invariant 2: window boundary
// ---------------------------------------------------------------------------

// TestReplay_WindowBoundary verifies BC-2.02.004 invariant 2.
// Frames older than the window (seq < nextSeq - windowSize) are discarded
// without error. EC-001: seq exactly at boundary evicts oldest entry.
func TestReplay_WindowBoundary(t *testing.T) {
	t.Parallel()

	const windowSize = 5
	deliver, got := newCollector()
	r := mustNew(t, windowSize, deliver)

	// Deliver seqs 1–5 in order, filling the window.
	for seq := uint32(1); seq <= windowSize; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5})

	// seq=6 accepted — window now covers 2–6 (seq=1 evicted).
	if err := r.OnUpstream(replay.Frame{Seq: 6}); err != nil {
		t.Fatalf("seq=6: unexpected error %v", err)
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5, 6})

	// seq=1 is now outside the window — must be silently discarded (nil error).
	if err := r.OnUpstream(replay.Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1 outside window: expected nil error, got %v", err)
	}
	// No new delivery.
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5, 6})
}

// TestReplay_WindowBoundary_ExactBoundarySeq is EC-001: the frame whose seq is
// exactly (nextSeq - windowSize) — the oldest slot — is evicted when the next
// frame advances the window; the new frame is accepted.
func TestReplay_WindowBoundary_ExactBoundarySeq(t *testing.T) {
	t.Parallel()

	const windowSize = 3
	deliver, got := newCollector()
	r := mustNew(t, windowSize, deliver)

	// Fill window with seqs 1, 2, 3.
	for seq := uint32(1); seq <= windowSize; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
	}

	// seq=4 advances the window; seq=1 is now the oldest evicted entry.
	if err := r.OnUpstream(replay.Frame{Seq: 4}); err != nil {
		t.Fatalf("seq=4: %v", err)
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4})

	// seq=1 is outside window — discarded without error.
	if err := r.OnUpstream(replay.Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1 post-eviction: expected nil, got %v", err)
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4}) // no change

	// seq=2 is still inside the window (window covers 2–4) — duplicate.
	if !errors.Is(r.OnUpstream(replay.Frame{Seq: 2}), replay.ErrAlreadyDelivered) {
		t.Error("seq=2 inside window: expected ErrAlreadyDelivered")
	}
}

// ---------------------------------------------------------------------------
// EC-002: all N frames in window re-sent → all N deduplicated
// ---------------------------------------------------------------------------

// TestReplay_EC002_AllWindowFramesResent verifies EC-002: re-sending every
// frame currently in the window returns ErrAlreadyDelivered for all of them
// and produces no additional deliveries.
func TestReplay_EC002_AllWindowFramesResent(t *testing.T) {
	t.Parallel()

	const windowSize = 5
	deliver, got := newCollector()
	r := mustNew(t, windowSize, deliver)

	// Deliver seqs 1–5 in order.
	for seq := uint32(1); seq <= windowSize; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5})

	// Re-send all five — each must be deduplicated.
	for seq := uint32(1); seq <= windowSize; seq++ {
		if !errors.Is(r.OnUpstream(replay.Frame{Seq: seq}), replay.ErrAlreadyDelivered) {
			t.Errorf("re-delivery seq=%d: want ErrAlreadyDelivered", seq)
		}
	}

	// Delivery count unchanged.
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5})
}

// ---------------------------------------------------------------------------
// EC-003: gap in sequence — buffered, delivered in order when gap filled
// ---------------------------------------------------------------------------

// TestReplay_EC003_GapBufferedThenFilled verifies EC-003: frames N+1 through
// N+K arrive before N; they are buffered and delivered in order once N arrives.
func TestReplay_EC003_GapBufferedThenFilled(t *testing.T) {
	t.Parallel()

	deliver, got := newCollector()
	r := mustNew(t, 10, deliver)

	// seq=1 delivered normally.
	if err := r.OnUpstream(replay.Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1: %v", err)
	}

	// Gap: seq=3,4,5 arrive before seq=2.
	for _, seq := range []uint32{3, 4, 5} {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d (buffered): unexpected error %v", seq, err)
		}
	}
	// Only seq=1 delivered; 3–5 are buffered.
	assertDelivered(t, *got, []uint32{1})

	// seq=2 fills the gap; 2, 3, 4, 5 delivered in order.
	if err := r.OnUpstream(replay.Frame{Seq: 2}); err != nil {
		t.Fatalf("seq=2 (gap filler): %v", err)
	}
	assertDelivered(t, *got, []uint32{1, 2, 3, 4, 5})
}

// ---------------------------------------------------------------------------
// VP-022 property test: no double delivery under random sequence permutations
// ---------------------------------------------------------------------------

// TestReplay_VP022_NoDoubleDelivery_Property exercises VP-022 with 1000+
// randomised delivery scenarios. Each seq must appear in the delivery log at
// most once regardless of arrival order or replay.
func TestReplay_VP022_NoDoubleDelivery_Property(t *testing.T) {
	t.Parallel()

	const (
		iterations = 1000
		maxSeqs    = 20
		windowSize = 10
	)

	rng := rand.New(rand.NewSource(42)) //nolint:gosec // deterministic test seed

	for i := 0; i < iterations; i++ {
		n := rng.Intn(maxSeqs) + 1 // 1..maxSeqs
		// Build a set of unique seqs in [1,n], then shuffle, then duplicate some.
		seqs := make([]uint32, n)
		for j := range seqs {
			seqs[j] = uint32(j + 1)
		}
		rng.Shuffle(len(seqs), func(a, b int) { seqs[a], seqs[b] = seqs[b], seqs[a] })

		// Append duplicates of a random subset.
		dupeCount := rng.Intn(n + 1)
		for d := 0; d < dupeCount; d++ {
			seqs = append(seqs, uint32(rng.Intn(n)+1))
		}

		deliver, got := newCollector()
		r := mustNew(t, windowSize, deliver)

		for _, seq := range seqs {
			err := r.OnUpstream(replay.Frame{Seq: seq})
			if err != nil && !errors.Is(err, replay.ErrAlreadyDelivered) {
				t.Fatalf("iter %d: unexpected error for seq=%d: %v", i, seq, err)
			}
		}

		// Each seq may appear in *got at most once.
		seen := make(map[uint32]int)
		for _, f := range *got {
			seen[f.Seq]++
			if seen[f.Seq] > 1 {
				t.Errorf("iter %d: seq=%d delivered %d times (VP-022 violation)",
					i, f.Seq, seen[f.Seq])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// VP-023 property test: in-order delivery under random permutations
// ---------------------------------------------------------------------------

// TestReplay_VP023_InOrderDelivery_Property exercises VP-023 with 1000+
// randomised permutations. Delivered frames must always be in strictly
// ascending sequence-number order.
func TestReplay_VP023_InOrderDelivery_Property(t *testing.T) {
	t.Parallel()

	const (
		iterations = 1000
		maxSeqs    = 20
		windowSize = 10
	)

	rng := rand.New(rand.NewSource(137)) //nolint:gosec // deterministic test seed

	for i := 0; i < iterations; i++ {
		n := rng.Intn(maxSeqs) + 1
		seqs := make([]uint32, n)
		for j := range seqs {
			seqs[j] = uint32(j + 1)
		}
		rng.Shuffle(len(seqs), func(a, b int) { seqs[a], seqs[b] = seqs[b], seqs[a] })

		deliver, got := newCollector()
		r := mustNew(t, windowSize, deliver)

		for _, seq := range seqs {
			err := r.OnUpstream(replay.Frame{Seq: seq})
			if err != nil && !errors.Is(err, replay.ErrAlreadyDelivered) {
				t.Fatalf("iter %d: unexpected error seq=%d: %v", i, seq, err)
			}
		}

		// Verify strictly ascending delivery order.
		frames := *got
		for j := 1; j < len(frames); j++ {
			if frames[j].Seq <= frames[j-1].Seq {
				t.Errorf("iter %d: delivery[%d].Seq=%d <= delivery[%d].Seq=%d (VP-023 violation)",
					i, j, frames[j].Seq, j-1, frames[j-1].Seq)
			}
		}
	}
}

// TestReplay_VP023_SortedDelivery_Canonical verifies VP-023 against the
// canonical test vector from BC-2.02.004: seq=10 'a' lost, recovered from
// replay window in seq=11.
func TestReplay_VP023_SortedDelivery_Canonical(t *testing.T) {
	t.Parallel()

	// Canonical BC vector: seq 6–9 delivered first, then seq=11 carries replay
	// window including seq=10. Receiver must deliver 6,7,8,9,10,11 in order.
	deliver, got := newCollector()
	r := mustNew(t, 10, deliver)

	// Deliver in-order seqs 1–9 first to establish the state.
	for seq := uint32(1); seq <= 9; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
	}

	// seq=10 is "lost" — skip it.
	// seq=11 arrives and carries replay of seq=10 in a prior call.
	// Simulate recovery: deliver seq=10 (from replay window) then seq=11.
	if err := r.OnUpstream(replay.Frame{Seq: 10, Payload: []byte("a")}); err != nil {
		t.Fatalf("seq=10 (recovered): %v", err)
	}
	if err := r.OnUpstream(replay.Frame{Seq: 11}); err != nil {
		t.Fatalf("seq=11: %v", err)
	}

	wantSeqs := make([]uint32, 11)
	for i := range wantSeqs {
		wantSeqs[i] = uint32(i + 1)
	}
	assertDelivered(t, *got, wantSeqs)
}

// ---------------------------------------------------------------------------
// WindowSize / NextSeq accessors
// ---------------------------------------------------------------------------

// TestReplay_WindowSize verifies that WindowSize returns the configured value.
func TestReplay_WindowSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		windowSize uint32
	}{
		{1},
		{5},
		{100},
	}
	for _, tc := range tests {
		tc := tc
		t.Run("", func(t *testing.T) {
			t.Parallel()
			deliver, _ := newCollector()
			r := mustNew(t, tc.windowSize, deliver)
			if got := r.WindowSize(); got != tc.windowSize {
				t.Errorf("WindowSize(): got %d, want %d", got, tc.windowSize)
			}
		})
	}
}

// TestReplay_NextSeq verifies that NextSeq advances after each delivered frame.
func TestReplay_NextSeq(t *testing.T) {
	t.Parallel()

	deliver, _ := newCollector()
	r := mustNew(t, 5, deliver)

	if got := r.NextSeq(); got != 1 {
		t.Fatalf("initial NextSeq: got %d, want 1", got)
	}

	for seq := uint32(1); seq <= 3; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
		if got := r.NextSeq(); got != seq+1 {
			t.Errorf("NextSeq after delivering seq=%d: got %d, want %d", seq, got, seq+1)
		}
	}
}

// ---------------------------------------------------------------------------
// New precondition panics
// ---------------------------------------------------------------------------

// TestReplay_New_PanicsOnZeroWindowSize verifies New panics on windowSize=0.
func TestReplay_New_PanicsOnZeroWindowSize(t *testing.T) {
	t.Parallel()

	deliver, _ := newCollector()
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		replay.New(0, deliver)
	}()
	if !panicked {
		t.Error("New(0, deliver): expected panic, got none")
	}
}

// TestReplay_New_PanicsOnNilDeliver verifies New panics on nil deliver.
func TestReplay_New_PanicsOnNilDeliver(t *testing.T) {
	t.Parallel()

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		replay.New(5, nil)
	}()
	if !panicked {
		t.Error("New(5, nil): expected panic, got none")
	}
}

// ---------------------------------------------------------------------------
// VP-023 monotonic invariant: replay window contents monotonically increasing
// BC-2.02.004 Verification Properties table, row 3
// ---------------------------------------------------------------------------

// TestReplay_BC_2_02_004_invariant_window_monotonic_seqs verifies that the
// delivered sequence numbers are always a monotonically increasing run — no
// gaps in what was actually sent in-order.
func TestReplay_BC_2_02_004_invariant_window_monotonic_seqs(t *testing.T) {
	t.Parallel()

	const windowSize = 5

	// Build 1000 random permutations of seqs 1..15 and verify the delivered
	// prefix is always a sorted, contiguous prefix starting at 1.
	rng := rand.New(rand.NewSource(999)) //nolint:gosec // deterministic

	for i := 0; i < 1000; i++ {
		n := rng.Intn(15) + 1
		seqs := make([]uint32, n)
		for j := range seqs {
			seqs[j] = uint32(j + 1)
		}
		rng.Shuffle(len(seqs), func(a, b int) { seqs[a], seqs[b] = seqs[b], seqs[a] })

		deliver, got := newCollector()
		r := mustNew(t, windowSize, deliver)

		for _, seq := range seqs {
			_ = r.OnUpstream(replay.Frame{Seq: seq})
		}

		frames := *got
		sorted := sort.SliceIsSorted(frames, func(a, b int) bool {
			return frames[a].Seq < frames[b].Seq
		})
		if !sorted {
			t.Errorf("iter %d: delivery not sorted: %v", i, seqsOf(frames))
			break
		}

		// Verify contiguous from 1: no internal gaps in what was delivered.
		for j, f := range frames {
			if f.Seq != uint32(j+1) {
				t.Errorf("iter %d: delivery[%d]=%d, want %d (gap in delivered set)",
					i, j, f.Seq, uint32(j+1))
				break
			}
		}
	}
}

// ---------------------------------------------------------------------------
// AC-004 / VP-042: Keystroke-to-echo latency benchmark ≤ p99 100ms
// ---------------------------------------------------------------------------

// BenchmarkReplay_KeystrokeLatency is the VP-042 benchmark gate.
// It measures the single-path OnUpstream() round-trip — New + one frame
// delivery per operation. The benchmark is the Red Gate for the 100ms p99
// constraint; once implemented the sub-microsecond call must be well under
// the 100ms ceiling.
func BenchmarkReplay_KeystrokeLatency(b *testing.B) {
	deliver := func(f replay.Frame) {}
	r := replay.New(5, deliver)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Re-create per iteration to keep seq fresh without allocation
		// overhead dominating. The reconstruction is cheap for a pure state
		// machine; the benchmark measures the delivery path, not setup.
		deliver2 := func(f replay.Frame) {}
		r2 := replay.New(5, deliver2)
		if err := r2.OnUpstream(replay.Frame{Seq: 1, Payload: []byte("k")}); err != nil {
			b.Fatalf("OnUpstream: %v", err)
		}
		_ = r
	}
}

// BenchmarkReplay_KeystrokeLatency_Sequential measures the steady-state
// OnUpstream() cost for sequential in-order frames (no reorder buffer churn).
// VP-042 requires p99 ≤ 100ms; this benchmark is expected to complete each
// iteration in sub-microsecond time.
func BenchmarkReplay_KeystrokeLatency_Sequential(b *testing.B) {
	deliver := func(f replay.Frame) {}
	r := replay.New(100, deliver)
	// Pre-warm: deliver seqs 1..100.
	for seq := uint32(1); seq <= 100; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			b.Fatalf("pre-warm seq=%d: %v", seq, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		seq := uint32(101 + i)
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			b.Fatalf("seq=%d: %v", seq, err)
		}
	}
}
