package replay_test

import (
	"errors"
	"math/rand"
	"sort"
	"testing"
	"time"

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
// canonical test vector from BC-2.02.004 row 2: seq=10 'a' is lost in
// transit; seq=11 arrives first (out-of-order gap at 10). The receiver must
// buffer seq=11, NOT deliver it, until seq=10 arrives from the replay window
// carried in a later frame. Delivery must be 10 then 11, in order.
func TestReplay_VP023_SortedDelivery_Canonical(t *testing.T) {
	t.Parallel()

	deliver, got := newCollector()
	r := mustNew(t, 10, deliver)

	// Deliver in-order seqs 1–9 to establish state; nextSeq is now 10.
	for seq := uint32(1); seq <= 9; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
	}
	assertDelivered(t, *got, func() []uint32 {
		s := make([]uint32, 9)
		for i := range s {
			s[i] = uint32(i + 1)
		}
		return s
	}())

	// seq=10 is "lost" — skip it entirely.
	// seq=11 arrives out-of-order (gap at 10): must be buffered, NOT delivered.
	if err := r.OnUpstream(replay.Frame{Seq: 11}); err != nil {
		t.Fatalf("seq=11 (out-of-order): %v", err)
	}
	// Still only 1..9 delivered — seq=11 is buffered pending seq=10.
	assertDelivered(t, *got, func() []uint32 {
		s := make([]uint32, 9)
		for i := range s {
			s[i] = uint32(i + 1)
		}
		return s
	}())

	// seq=10 arrives (recovered from replay window in a later frame).
	// Now the gap is filled: 10 then 11 must drain in order.
	if err := r.OnUpstream(replay.Frame{Seq: 10, Payload: []byte("a")}); err != nil {
		t.Fatalf("seq=10 (recovered from replay window): %v", err)
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
// AC-004 / VP-042: Keystroke-to-echo latency gate ≤ p99 100ms
// ---------------------------------------------------------------------------

// TestReplay_VP042_KeystrokeLatencyP99 is the VP-042 CI gate.
// It measures the steady-state OnUpstream→deliver round-trip for 10 000
// in-order keystrokes and FAILS if p99 > 100ms. The Replay instance is
// constructed once outside the measured region; the loop exercises only the
// delivery path. VP-042 requires the p99 ≤ 100ms; the implementation is
// expected to be well under 1µs per call.
func TestReplay_VP042_KeystrokeLatencyP99(t *testing.T) {
	t.Parallel()

	const (
		iterations = 10_000
		p99limit   = 100 * time.Millisecond
		p99index   = iterations * 99 / 100 // index of the 99th-percentile sample
	)

	deliver := func(_ replay.Frame) {}
	r := replay.New(100, deliver)

	// Pre-warm: establish a rolling window so steady-state is measured.
	for seq := uint32(1); seq <= 100; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("pre-warm seq=%d: %v", seq, err)
		}
	}

	samples := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		seq := uint32(101 + i)
		start := time.Now()
		if err := r.OnUpstream(replay.Frame{Seq: seq, Payload: []byte("k")}); err != nil {
			t.Fatalf("iteration %d seq=%d: %v", i, seq, err)
		}
		samples[i] = time.Since(start)
	}

	// Sort to find p99.
	sort.Slice(samples, func(a, b int) bool { return samples[a] < samples[b] })
	p99 := samples[p99index]

	if p99 > p99limit {
		t.Fatalf("VP-042: p99 latency %v exceeds 100ms gate (p99 index %d of %d)",
			p99, p99index, iterations)
	}
}

// BenchmarkReplay_KeystrokeLatency_Sequential measures the steady-state
// OnUpstream() cost for sequential in-order frames (no reorder buffer churn).
// VP-042 requires p99 ≤ 100ms; this benchmark is expected to complete each
// iteration in sub-microsecond time.
func BenchmarkReplay_KeystrokeLatency_Sequential(b *testing.B) {
	deliver := func(_ replay.Frame) {}
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

// ---------------------------------------------------------------------------
// F-001 / BC-2.02.004 invariant 3: bounded pending buffer
// ---------------------------------------------------------------------------

// TestReplay_BoundedPendingBuffer verifies BC-2.02.004 invariant 3 (PC5):
// frames with seq >= nextSeq + windowSize must be discarded, not buffered.
// A never-filled gap (seq=2 never arrives) combined with a stream of far-future
// frames must not cause the replay buffer to accumulate unbounded state.
//
// The behavioral proof: after filling the gap (seq=2) and advancing nextSeq
// all the way to seq=100, the far-future frames that were sent while nextSeq=2
// (i.e., seq=100..window+1+2 which equals seq=7...) must NOT be delivered when
// nextSeq reaches them — they must have been discarded at arrival time.
//
// Concretely: send seq=1 (nextSeq→2), then send seq=50 (far future, >= 2+5=7).
// Then deliver seq=2..49 in order. When nextSeq reaches 50, if seq=50 was
// buffered (current impl) it WILL be auto-delivered as nextSeq drains through
// pending. If seq=50 was discarded (correct impl) it must NOT be delivered,
// and delivering seq=50 again after the gap returns nil (not ErrAlreadyDelivered).
//
// This test is expected to FAIL against the current unbounded implementation
// (Red Gate for the implementer).
func TestReplay_BoundedPendingBuffer(t *testing.T) {
	t.Parallel()

	const windowSize = 5

	deliver, got := newCollector()
	r := mustNew(t, windowSize, deliver)

	// Deliver seq=1 so nextSeq advances to 2.
	if err := r.OnUpstream(replay.Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1: %v", err)
	}
	assertDelivered(t, *got, []uint32{1})

	// nextSeq=2, windowSize=5: in-window upper bound is nextSeq+windowSize-1 = 6.
	// seq=50 is far outside the window (50 >= 2+5 = 7) — must be discarded.
	if err := r.OnUpstream(replay.Frame{Seq: 50, Payload: []byte("far")}); err != nil {
		t.Fatalf("far-future seq=50: expected nil (silent discard), got %v", err)
	}
	// Still only seq=1 delivered.
	assertDelivered(t, *got, []uint32{1})

	// Now deliver seq=2..49 in order — this fills the gap and advances
	// nextSeq step by step. When nextSeq reaches 50, the pending map will be
	// checked for seq=50. A correct (bounded) impl will NOT have seq=50 in
	// pending (it was discarded). A buggy (unbounded) impl WILL have it and
	// will auto-deliver it.
	for seq := uint32(2); seq <= 49; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
	}

	// At this point nextSeq should be 50 (we've delivered 1..49).
	// The far-future frame (seq=50) must NOT have been auto-delivered via pending.
	// Correct: 49 deliveries (seq 1..49). Buggy: 50 deliveries (seq 1..50).
	wantSeqs := make([]uint32, 49)
	for i := range wantSeqs {
		wantSeqs[i] = uint32(i + 1)
	}
	assertDelivered(t, *got, wantSeqs) // FAILS on current impl: seq=50 auto-drains

	// Confirm nextSeq is 50 (not 51, which would be evidence seq=50 was drained).
	if ns := r.NextSeq(); ns != 50 {
		t.Errorf("NextSeq: got %d, want 50", ns)
	}

	// Now explicitly deliver seq=50 — it was discarded so this must succeed
	// (not ErrAlreadyDelivered) and deliver seq=50 for the first time.
	if err := r.OnUpstream(replay.Frame{Seq: 50}); err != nil {
		t.Fatalf("seq=50 explicit delivery: got %v, want nil", err)
	}
	wantSeqs = append(wantSeqs, 50)
	assertDelivered(t, *got, wantSeqs)
}

// ---------------------------------------------------------------------------
// F-004: evicted-seq redelivery returns nil (no double delivery)
// ---------------------------------------------------------------------------

// TestReplay_EvictedSeqRedeliveryReturnsNil verifies that a seq which has been
// delivered AND evicted from the window (slid out of the seen set) returns nil
// when re-sent, and is NOT delivered again (PC2 holds).
//
// Contrast with TestReplay_NoDuplicateDelivery: that test verifies
// ErrAlreadyDelivered for in-window duplicates. This test verifies the evicted
// case returns nil instead.
func TestReplay_EvictedSeqRedeliveryReturnsNil(t *testing.T) {
	t.Parallel()

	const windowSize = 3
	deliver, got := newCollector()
	r := mustNew(t, windowSize, deliver)

	// Deliver seqs 1..windowSize+1 so seq=1 is evicted from the seen window.
	for seq := uint32(1); seq <= windowSize+1; seq++ {
		if err := r.OnUpstream(replay.Frame{Seq: seq}); err != nil {
			t.Fatalf("seq=%d: %v", seq, err)
		}
	}
	// Delivered 1,2,3,4. Window now covers 2..4 (seq=1 evicted).
	assertDelivered(t, *got, []uint32{1, 2, 3, 4})

	deliveredBefore := len(*got)

	// Re-send seq=1 — it was delivered but is now outside the window.
	// Must return nil (silent discard), NOT ErrAlreadyDelivered, AND must
	// not invoke deliver again.
	err := r.OnUpstream(replay.Frame{Seq: 1})
	if err != nil {
		t.Fatalf("evicted seq=1 redelivery: got %v, want nil", err)
	}
	if len(*got) != deliveredBefore {
		t.Errorf("evicted seq=1 redelivery caused extra delivery: got %d calls, want %d",
			len(*got), deliveredBefore)
	}
}

// TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered is the complement of
// TestReplay_EvictedSeqRedeliveryReturnsNil: an in-window duplicate must still
// return ErrAlreadyDelivered.
func TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered(t *testing.T) {
	t.Parallel()

	const windowSize = 5
	deliver, _ := newCollector()
	r := mustNew(t, windowSize, deliver)

	if err := r.OnUpstream(replay.Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1: %v", err)
	}

	// seq=1 is still inside the window — must return ErrAlreadyDelivered.
	if !errors.Is(r.OnUpstream(replay.Frame{Seq: 1}), replay.ErrAlreadyDelivered) {
		t.Error("in-window duplicate seq=1: want ErrAlreadyDelivered")
	}
}

// ---------------------------------------------------------------------------
// F-005: seq=0 (unset/invalid) is discarded
// ---------------------------------------------------------------------------

// TestReplay_SeqZeroDiscarded verifies that a Frame{Seq:0} returns nil and is
// not delivered. Seq=0 is the unset/invalid sentinel per the monotonically-
// increasing invariant in BC-2.02.004.
func TestReplay_SeqZeroDiscarded(t *testing.T) {
	t.Parallel()

	deliver, got := newCollector()
	r := mustNew(t, 5, deliver)

	// Seq=0 must be silently discarded (nil return, no delivery).
	if err := r.OnUpstream(replay.Frame{Seq: 0, Payload: []byte("x")}); err != nil {
		t.Fatalf("seq=0: expected nil, got %v", err)
	}
	assertDelivered(t, *got, []uint32{}) // nothing delivered

	// Normal delivery after seq=0 must still work.
	if err := r.OnUpstream(replay.Frame{Seq: 1}); err != nil {
		t.Fatalf("seq=1 after seq=0: %v", err)
	}
	assertDelivered(t, *got, []uint32{1})
}
