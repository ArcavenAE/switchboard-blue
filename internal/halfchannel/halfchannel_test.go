package halfchannel_test

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/halfchannel"
)

// tickOnce calls hc.Tick() and returns the elapsed wall-clock duration.
// The _ = t0 consume satisfies staticcheck SA4006 during Red Gate: once the
// stub panics on Tick(), time.Since(t0) is unreachable. The blank assignment
// makes t0 read before the potential panic without affecting the measurement.
func tickOnce(hc *halfchannel.HalfChannel) time.Duration {
	t0 := time.Now().UTC()
	_ = t0 // consumed here to guard SA4006 during Red Gate; also used in return below
	hc.Tick()
	return time.Since(t0)
}

// -----------------------------------------------------------------------------
// AC-001 / BC-2.01.001 postcondition 1
// -----------------------------------------------------------------------------

// TestHalfChannelTick_OneFramePerCall verifies that Tick produces exactly one
// ChannelFrame per call regardless of whether payload is queued.
func TestHalfChannelTick_OneFramePerCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		chanID    uint32
		direction halfchannel.Direction
		enqueue   [][]byte // payloads to enqueue before tick
	}{
		{
			name:      "upstream no payload",
			chanID:    1,
			direction: halfchannel.Upstream,
			enqueue:   nil,
		},
		{
			name:      "downstream no payload",
			chanID:    2,
			direction: halfchannel.Downstream,
			enqueue:   nil,
		},
		{
			name:      "upstream with payload",
			chanID:    3,
			direction: halfchannel.Upstream,
			enqueue:   [][]byte{[]byte("hello")},
		},
		{
			name:      "downstream with payload",
			chanID:    4,
			direction: halfchannel.Downstream,
			enqueue:   [][]byte{[]byte("world")},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hc := halfchannel.New(tc.chanID, tc.direction, 10*time.Millisecond)

			for _, p := range tc.enqueue {
				if err := hc.Enqueue(p); err != nil {
					t.Fatalf("Enqueue failed: %v", err)
				}
			}

			// Tick must return exactly one frame — the call must not block or
			// return multiple frames via variadic/slice return (it returns one
			// ChannelFrame value by design).
			frame := hc.Tick()

			// ChanID must be set to the channel's own ID.
			if frame.ChanID != tc.chanID {
				t.Errorf("frame.ChanID = %d, want %d", frame.ChanID, tc.chanID)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// AC-002 / BC-2.01.002 postcondition 1
// -----------------------------------------------------------------------------

// TestHalfChannelTick_EmptyFrameIsValid verifies that a tick with no queued
// payload produces a frame with zero-length Payload and the channel ID set,
// for both directions.
func TestHalfChannelTick_EmptyFrameIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		chanID    uint32
		direction halfchannel.Direction
	}{
		{"upstream", 10, halfchannel.Upstream},
		{"downstream", 11, halfchannel.Downstream},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hc := halfchannel.New(tc.chanID, tc.direction, 10*time.Millisecond)
			frame := hc.Tick()

			if len(frame.Payload) != 0 {
				t.Errorf("empty tick payload len = %d, want 0", len(frame.Payload))
			}
			if frame.ChanID != tc.chanID {
				t.Errorf("frame.ChanID = %d, want %d", frame.ChanID, tc.chanID)
			}
			// Frame must be structurally usable: ChanSeq must be 1 (first tick).
			if frame.ChanSeq != 1 {
				t.Errorf("frame.ChanSeq = %d, want 1 on first tick", frame.ChanSeq)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// AC-003 / BC-2.01.003 postcondition 1
// -----------------------------------------------------------------------------

// TestHalfChannelIndependentSequences verifies that ticking channel A does not
// advance channel B's sequence counter.
func TestHalfChannelIndependentSequences(t *testing.T) {
	t.Parallel()

	a := halfchannel.New(100, halfchannel.Upstream, 10*time.Millisecond)
	b := halfchannel.New(200, halfchannel.Downstream, 10*time.Millisecond)

	// Tick A three times.
	for range 3 {
		a.Tick()
	}

	// B must still be at seq 0 (never ticked).
	if got := b.Seq(); got != 0 {
		t.Errorf("B.Seq() = %d after ticking A, want 0", got)
	}

	// Tick B once; A's counter must not change.
	// A was ticked exactly 3 times; its seq must still be 3 after B ticks.
	b.Tick()
	if got := a.Seq(); got != 3 {
		t.Errorf("A.Seq() = %d after ticking B, want 3 (unchanged from 3 ticks of A)", got)
	}
}

// -----------------------------------------------------------------------------
// AC-004 / BC-2.01.003 postcondition 2
// -----------------------------------------------------------------------------

// TestHalfChannelSequenceIncrement verifies that after N ticks Seq() == N and
// that each individual tick increments by exactly 1.
func TestHalfChannelSequenceIncrement(t *testing.T) {
	t.Parallel()

	const N = 50

	hc := halfchannel.New(42, halfchannel.Upstream, 10*time.Millisecond)

	if hc.Seq() != 0 {
		t.Fatalf("initial Seq() = %d, want 0", hc.Seq())
	}

	for i := range N {
		hc.Tick()
		// After i+1 ticks, Seq() must equal i+1.
		want := uint32(i + 1)
		if got := hc.Seq(); got != want {
			t.Errorf("after tick %d: Seq() = %d, want %d", i+1, got, want)
		}
	}

	if got := hc.Seq(); got != N {
		t.Errorf("after %d ticks: Seq() = %d, want %d", N, got, N)
	}
}

// -----------------------------------------------------------------------------
// AC-005 / VP-041 — Benchmark
// -----------------------------------------------------------------------------

// BenchmarkHalfChannelTickJitter measures per-tick call latency over 1000
// iterations and reports p99 jitter. VP-041 gate (≤ 2ms p99) is enforced in
// the formal-verification phase; this benchmark records the metric only.
func BenchmarkHalfChannelTickJitter(b *testing.B) {
	const samples = 1000

	hc := halfchannel.New(99, halfchannel.Upstream, 10*time.Millisecond)

	b.ResetTimer()

	for range b.N {
		latencies := make([]time.Duration, samples)

		for i := range samples {
			latencies[i] = tickOnce(hc)
		}

		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		p99idx := int(math.Ceil(float64(samples)*0.99)) - 1
		if p99idx >= samples {
			p99idx = samples - 1
		}
		p99ms := float64(latencies[p99idx]) / float64(time.Millisecond)
		b.ReportMetric(p99ms, "jitter_p99_ms")
	}
}

// -----------------------------------------------------------------------------
// AC-006 / BC-2.01.002 invariant 1 + VP-053
// -----------------------------------------------------------------------------

// TestHalfChannelEmptyTickSequence verifies that K consecutive empty ticks
// produce K frames with contiguous sequence numbers (no gaps, no duplicates).
func TestHalfChannelEmptyTickSequence(t *testing.T) {
	t.Parallel()

	const K = 20

	hc := halfchannel.New(7, halfchannel.Downstream, 10*time.Millisecond)

	seqs := make([]uint32, K)
	for i := range K {
		f := hc.Tick()
		seqs[i] = f.ChanSeq
	}

	// Verify contiguous: seqs[i] == seqs[i-1] + 1 and seqs[0] == 1.
	if seqs[0] != 1 {
		t.Errorf("first empty-tick frame ChanSeq = %d, want 1", seqs[0])
	}
	for i := 1; i < K; i++ {
		if seqs[i] != seqs[i-1]+1 {
			t.Errorf("gap or dup at tick %d: seq[%d]=%d seq[%d]=%d",
				i+1, i-1, seqs[i-1], i, seqs[i])
		}
	}
}

// -----------------------------------------------------------------------------
// Edge case: BC-2.01.002 precondition — Enqueue rejects nil payload
// -----------------------------------------------------------------------------

// TestHalfChannel_EnqueueNilPayload verifies that Enqueue returns an error
// when passed a nil payload (BC-2.01.002 precondition).
func TestHalfChannel_EnqueueNilPayload(t *testing.T) {
	t.Parallel()

	hc := halfchannel.New(5, halfchannel.Upstream, 10*time.Millisecond)

	err := hc.Enqueue(nil)
	if err == nil {
		t.Error("Enqueue(nil) returned nil error, want non-nil error")
	}
}

// -----------------------------------------------------------------------------
// Edge case EC-002 — sequence wraparound
// -----------------------------------------------------------------------------

// TestHalfChannelSequenceWraparound drives sequence from math.MaxUint32-1 to
// verify the counter wraps to 0 without overflow panic.
//
// Seeding via the stub's unexported field is not possible without API changes.
// The public API has no constructor that accepts an initial seq. Looping to
// MaxUint32 ticks is infeasible in test time. This test is skipped pending
// VP-016 property-test harness that can inject initial state or a constructor
// variant (to be added by the implementer as a test-only option).
func TestHalfChannelSequenceWraparound(t *testing.T) {
	t.Skip("EC-002: wraparound covered once VP-016 harness or test constructor variant is available — see story S-1.02 edge-case notes")
}

// -----------------------------------------------------------------------------
// Edge case EC-003 — multiple payloads queued, single tick
// -----------------------------------------------------------------------------

// TestHalfChannelTick_MultiplePayloadsQueuedOneTick verifies that when two
// payloads are enqueued before a single tick, only the first payload is emitted
// and the second remains for the next tick.
func TestHalfChannelTick_MultiplePayloadsQueuedOneTick(t *testing.T) {
	t.Parallel()

	hc := halfchannel.New(8, halfchannel.Upstream, 10*time.Millisecond)

	first := []byte("payload-one")
	second := []byte("payload-two")

	if err := hc.Enqueue(first); err != nil {
		t.Fatalf("Enqueue(first) failed: %v", err)
	}
	if err := hc.Enqueue(second); err != nil {
		t.Fatalf("Enqueue(second) failed: %v", err)
	}

	// First tick: must emit first payload.
	f1 := hc.Tick()
	if string(f1.Payload) != string(first) {
		t.Errorf("tick 1 payload = %q, want %q", f1.Payload, first)
	}

	// Second tick: must emit second payload.
	f2 := hc.Tick()
	if string(f2.Payload) != string(second) {
		t.Errorf("tick 2 payload = %q, want %q", f2.Payload, second)
	}

	// Third tick: queue is empty — payload must be empty.
	f3 := hc.Tick()
	if len(f3.Payload) != 0 {
		t.Errorf("tick 3 payload len = %d, want 0 (queue exhausted)", len(f3.Payload))
	}
}

// -----------------------------------------------------------------------------
// Property tests
// -----------------------------------------------------------------------------

// TestProperty_VP016_SequenceStrictlyMonotonic verifies that across 10k ticks,
// the sequence number increments by exactly 1 on every call.
// VP-016: for all tick sequences, seq increments monotonically.
func TestProperty_VP016_SequenceStrictlyMonotonic(t *testing.T) {
	t.Parallel()

	const iterations = 10_000

	hc := halfchannel.New(1000, halfchannel.Upstream, 10*time.Millisecond)

	for i := range iterations {
		hc.Tick()
		// After i+1 ticks from seq=0, Seq() must equal i+1.
		wantSeq := uint32(i + 1)
		if got := hc.Seq(); got != wantSeq {
			t.Fatalf("VP-016 violated at iteration %d: Seq() = %d, want %d",
				i+1, got, wantSeq)
		}
	}
}

// TestProperty_VP017_SingleFramePerTick verifies that every call to Tick
// returns exactly one frame (no batching). The invariant is structural: the
// return type is a single ChannelFrame value, and its ChanSeq matches the
// post-tick Seq().
// VP-017: invariant — every Tick returns exactly one frame.
func TestProperty_VP017_SingleFramePerTick(t *testing.T) {
	t.Parallel()

	const iterations = 10_000

	hc := halfchannel.New(2000, halfchannel.Downstream, 10*time.Millisecond)

	for i := range iterations {
		f := hc.Tick()
		// The frame's own ChanSeq must equal the current Seq() (post-increment).
		wantSeq := hc.Seq()
		if f.ChanSeq != wantSeq {
			t.Fatalf("VP-017 violated at iteration %d: frame.ChanSeq=%d but Seq()=%d",
				i+1, f.ChanSeq, wantSeq)
		}
	}
}

// TestProperty_VP018_Independence verifies that two channels with different
// ChanIDs maintain independent sequence state across interleaved ticks.
// VP-018: two channels maintain independent sequence spaces.
func TestProperty_VP018_Independence(t *testing.T) {
	t.Parallel()

	const iterations = 10_000

	a := halfchannel.New(3000, halfchannel.Upstream, 10*time.Millisecond)
	b := halfchannel.New(3001, halfchannel.Downstream, 20*time.Millisecond)

	// Track expected counts locally — avoids calling Seq() before a Tick()
	// that would panic during Red Gate (which would cause SA4006 on the capture).
	var ticksA, ticksB uint32

	for i := range iterations {
		// Alternate ticking a and b.
		if i%2 == 0 {
			a.Tick()
			ticksA++
			// B must still equal ticksB.
			if got := b.Seq(); got != ticksB {
				t.Fatalf("VP-018 violated at iteration %d: ticking A changed B.Seq to %d, want %d",
					i+1, got, ticksB)
			}
		} else {
			b.Tick()
			ticksB++
			// A must still equal ticksA.
			if got := a.Seq(); got != ticksA {
				t.Fatalf("VP-018 violated at iteration %d: ticking B changed A.Seq to %d, want %d",
					i+1, got, ticksA)
			}
		}
	}
}
