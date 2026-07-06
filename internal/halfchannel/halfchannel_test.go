package halfchannel_test

import (
	"bytes"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/halfchannel"
)

// VP-016 and AC-001 ("exactly one frame per Tick call") are enforced
// structurally by func (h *HalfChannel) Tick() ChannelFrame — a singular
// return value cannot return zero or more than one frame. No runtime test
// asserts this; the type system does. TestHalfChannelTick_ChanIDPropagation
// (the test formerly named TestHalfChannelTick_OneFramePerCall) verifies
// the ChanID propagation aspect of BC-2.01.001 — not the cardinality.
//
// VP-017 ("sequence increments by exactly 1 per tick") is NOT structurally
// enforced — it requires runtime verification.
// TestProperty_VP017_SequenceIncrementsByOne exercises this invariant by
// asserting f2.ChanSeq - f1.ChanSeq == 1 across consecutive ticks.

// -----------------------------------------------------------------------------
// AC-001 / BC-2.01.001 postcondition 1
// -----------------------------------------------------------------------------

// TestHalfChannelTick_ChanIDPropagation verifies that Tick propagates the
// channel's ChanID into every returned ChannelFrame regardless of direction or
// payload state.
func TestHalfChannelTick_ChanIDPropagation(t *testing.T) {
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
// AC-002 / BC-2.01.002 postcondition 1–2
// -----------------------------------------------------------------------------

// TestHalfChannelTick_EmptyFrameIsValid verifies that a tick with no queued
// payload produces a frame with zero-length Payload, FrameTypeEmptyTick, and
// correct ChanID and ChanSeq, for both directions.
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
			// F-002: empty ticks must carry FrameTypeEmptyTick (BC-2.01.002 postcondition 2).
			if frame.FrameType != halfchannel.FrameTypeEmptyTick {
				t.Errorf("frame.FrameType = 0x%02x, want FrameTypeEmptyTick (0x%02x)",
					frame.FrameType, halfchannel.FrameTypeEmptyTick)
			}
			if frame.Flags != 0 {
				t.Errorf("frame.Flags = %#x, want 0 (BC-2.01.002 PC3 — flags=0 for empty-tick frames)", frame.Flags)
			}
		})
	}
}

// TestHalfChannelTick_DataFrameType verifies that when a non-nil payload is
// enqueued and Tick is called, the returned frame has FrameTypeData and
// non-empty Payload (BC-2.01.002 postcondition 2, AC-002).
func TestHalfChannelTick_DataFrameType(t *testing.T) {
	t.Parallel()

	hc := halfchannel.New(20, halfchannel.Upstream, 10*time.Millisecond)

	payload := []byte("hello switchboard")
	if err := hc.Enqueue(payload); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	frame := hc.Tick()

	if frame.FrameType != halfchannel.FrameTypeData {
		t.Errorf("frame.FrameType = 0x%02x, want FrameTypeData (0x%02x)",
			frame.FrameType, halfchannel.FrameTypeData)
	}
	if len(frame.Payload) == 0 {
		t.Error("frame.Payload is empty, want non-empty for data frame")
	}
	if !bytes.Equal(frame.Payload, payload) {
		t.Errorf("frame.Payload = %q, want %q", frame.Payload, payload)
	}
	if frame.Flags != 0 {
		t.Errorf("frame.Flags = %#x, want 0 (BC-2.01.002 PC3 — flags=0 for MVP data frames; FEC/ARQ/SACK bits land in later stories)", frame.Flags)
	}
}

// TestHalfChannelTick_PayloadZeroCopy pins the documented zero-copy contract.
// Per halfchannel.go Enqueue godoc: "The payload is not copied; the caller
// must not mutate it after passing it to Enqueue." This test verifies that
// frame.Payload shares its backing array with the original slice.
func TestHalfChannelTick_PayloadZeroCopy(t *testing.T) {
	t.Parallel()
	hc := halfchannel.New(0x100, halfchannel.Upstream, 10*time.Millisecond)
	p := []byte("zero-copy test payload")
	if err := hc.Enqueue(p); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	frame := hc.Tick()
	if len(frame.Payload) == 0 {
		t.Fatal("frame.Payload empty after Enqueue+Tick")
	}
	// Zero-copy contract: frame.Payload must share the backing array with p.
	// Per halfchannel.go Enqueue godoc: "The payload is not copied; the
	// caller must not mutate it after passing it to Enqueue."
	if &frame.Payload[0] != &p[0] {
		t.Errorf("frame.Payload[0] addr=%p, p[0] addr=%p — zero-copy contract violated (defensive copy?)",
			&frame.Payload[0], &p[0])
	}
}

// -----------------------------------------------------------------------------
// AC-003 / BC-2.01.003 postcondition 1
// -----------------------------------------------------------------------------

// TestHalfChannelIndependentSequences verifies that ticking one channel does not
// advance another channel's sequence counter. Both channels share the same chanID
// but differ in direction — pinning BC-2.01.003 PC1's claim that sequence spaces
// are independent per instance (not per chanID).
func TestHalfChannelIndependentSequences(t *testing.T) {
	t.Parallel()

	const chanID uint32 = 0x100
	up := halfchannel.New(chanID, halfchannel.Upstream, 10*time.Millisecond)
	down := halfchannel.New(chanID, halfchannel.Downstream, 10*time.Millisecond)

	// Tick the upstream channel 5 times.
	for range 5 {
		up.Tick()
	}
	if got := up.Seq(); got != 5 {
		t.Errorf("upstream Seq after 5 ticks = %d, want 5", got)
	}
	if got := down.Seq(); got != 0 {
		t.Errorf("downstream Seq after upstream ticks = %d, want 0 (sequence spaces independent)", got)
	}

	// Tick the downstream channel 3 times.
	for range 3 {
		down.Tick()
	}
	if got := up.Seq(); got != 5 {
		t.Errorf("upstream Seq after downstream ticks = %d, want 5 (unchanged)", got)
	}
	if got := down.Seq(); got != 3 {
		t.Errorf("downstream Seq after 3 ticks = %d, want 3", got)
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
// AC-005 / VP-041 — Benchmark (S-BL.BENCH)
// -----------------------------------------------------------------------------

// BenchmarkHalfChannelTickJitter measures the end-to-end inter-tick interval
// including the scheduled work (sleep accuracy + tick execution time). It
// reports p99 jitter as the custom metric jitter_p99_ms and enforces the
// NFR-009 / VP-041 gate (≤ 2ms p99) via b.Errorf.
//
// Phase stratification (AC-005 / S-1.02 rev 1.2 / VP-041 v1.2):
//   - Phase 3: recorded jitter_p99_ms observationally only (no gate).
//   - Phase 6 (S-BL.BENCH): adds b.Errorf gate. This benchmark is
//     DIAGNOSTIC, not a required CI check (ADR-007). Run it on stable
//     hardware to verify the SLO; developer laptops may legitimately
//     exceed 2ms under load. Reported results constitute proof evidence.
//
// The benchmark uses b.N so -benchtime controls the sample count:
//
//	-benchtime=1000x  →  ~1000 ticks (~10s wall-clock at 10ms/tick)
//	-benchtime=2s     →  ~200 ticks (whatever fits in 2s)
//
// The benchmark drives its own cadence via time.Sleep — ARCH-09 is not
// violated because the BENCHMARK (effectful glue) is what schedules ticks,
// not HalfChannel itself. HalfChannel.Tick() remains pure-core.
func BenchmarkHalfChannelTickJitter(b *testing.B) {
	const (
		interval  = 10 * time.Millisecond
		maxJitter = 2 * time.Millisecond // NFR-009 / VP-041
	)
	hc := halfchannel.New(0, halfchannel.Upstream, interval)

	deviations := make([]time.Duration, b.N)
	b.ResetTimer()
	prev := time.Now().UTC()

	for i := 0; i < b.N; i++ {
		// Sleep until the next tick boundary relative to prev.
		target := prev.Add(interval)
		if d := time.Until(target); d > 0 {
			time.Sleep(d)
		}
		_ = hc.Tick()
		now := time.Now().UTC()
		actual := now.Sub(prev)
		if actual >= interval {
			deviations[i] = actual - interval
		} else {
			deviations[i] = interval - actual
		}
		// Update prev to actual tick time so the next interval is measured
		// relative to the real previous tick, not the nominal schedule.
		prev = now
	}
	b.StopTimer()

	if b.N == 0 {
		return
	}
	sort.Slice(deviations, func(i, j int) bool { return deviations[i] < deviations[j] })
	p99 := deviations[int(float64(b.N)*0.99)]
	b.ReportMetric(float64(p99)/float64(time.Millisecond), "jitter_p99_ms")

	// VP-041 gate (S-BL.BENCH AC-001): enforce ≤ 2ms p99 on stable CI hardware.
	// Gate activates only at ≥ 100 samples (p99 is statistically meaningful).
	// With b.N < 100 (e.g. the Go framework's initial 1-iteration probe) the
	// gate is skipped — a single-tick measurement cannot produce a valid p99.
	// On developer laptops the gate may legitimately fail under load — it is
	// diagnostic, not a required CI check (ADR-007). Evidence: jitter_p99_ms above.
	if b.N >= 100 && p99 > maxJitter {
		b.Errorf("tick p99 jitter %v exceeds NFR-009 limit %v (VP-041)", p99, maxJitter)
	}
}

// -----------------------------------------------------------------------------
// AC-006 / BC-2.01.002 invariant 1 + VP-053
// -----------------------------------------------------------------------------

// TestHalfChannelEmptyTickSequence verifies that K consecutive empty ticks
// produce K frames with contiguous sequence numbers (no gaps, no duplicates),
// empty payloads, and FrameTypeEmptyTick on every frame.
func TestHalfChannelEmptyTickSequence(t *testing.T) {
	t.Parallel()

	const K = 20

	hc := halfchannel.New(7, halfchannel.Downstream, 10*time.Millisecond)

	seqs := make([]uint32, K)
	for i := range K {
		f := hc.Tick()
		seqs[i] = f.ChanSeq

		// F-003: every empty-tick frame must have zero-length payload.
		if len(f.Payload) != 0 {
			t.Errorf("tick %d: payload len = %d, want 0", i+1, len(f.Payload))
		}
		// F-003: every empty-tick frame must carry FrameTypeEmptyTick.
		if f.FrameType != halfchannel.FrameTypeEmptyTick {
			t.Errorf("tick %d: FrameType = 0x%02x, want FrameTypeEmptyTick (0x%02x)",
				i+1, f.FrameType, halfchannel.FrameTypeEmptyTick)
		}
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

// TestHalfChannelEnqueue_NilRejected verifies that Enqueue returns ErrEmptyPayload
// when passed a nil payload (BC-2.01.002 precondition 4).
func TestHalfChannelEnqueue_NilRejected(t *testing.T) {
	t.Parallel()

	hc := halfchannel.New(5, halfchannel.Upstream, 10*time.Millisecond)

	err := hc.Enqueue(nil)
	if err == nil {
		t.Fatal("Enqueue(nil) returned nil error")
	}
	if !errors.Is(err, halfchannel.ErrEmptyPayload) {
		t.Errorf("Enqueue(nil) error = %v, want errors.Is ErrEmptyPayload", err)
	}
}

// TestHalfChannelEnqueue_EmptySliceRejected verifies that Enqueue returns
// ErrEmptyPayload when passed a zero-length byte slice (BC-2.01.002
// precondition 4 — "nil or zero-length").
func TestHalfChannelEnqueue_EmptySliceRejected(t *testing.T) {
	t.Parallel()

	hc := halfchannel.New(5, halfchannel.Upstream, 10*time.Millisecond)

	err := hc.Enqueue([]byte{})
	if err == nil {
		t.Fatal("Enqueue([]byte{}) returned nil error")
	}
	if !errors.Is(err, halfchannel.ErrEmptyPayload) {
		t.Errorf("Enqueue([]byte{}) error = %v, want errors.Is ErrEmptyPayload", err)
	}
}

// -----------------------------------------------------------------------------
// Edge case EC-002 — sequence wraparound
// -----------------------------------------------------------------------------

// TestHalfChannelSequenceWraparound: EC-002 wraparound is now covered by the
// internal-package test in wraparound_test.go, which seeds hc.seq
// directly. A public-API-only test cannot reach MaxUint32 in reasonable time.

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

// TestProperty_VP017_SequenceIncrementsByOne asserts the BC-2.01.001 PC5
// invariant: each Tick increments ChanSeq by exactly 1. Uses uint32
// arithmetic — the modular subtraction `f.ChanSeq - prev.ChanSeq` is
// wraparound-safe by construction, but this test does NOT exercise
// wraparound (10,000 iterations from seq=0 is far below math.MaxUint32).
// EC-002 wraparound is covered by wraparound_test.go.
func TestProperty_VP017_SequenceIncrementsByOne(t *testing.T) {
	t.Parallel()

	const iterations = 10_000

	hc := halfchannel.New(0xAAAA, halfchannel.Upstream, 10*time.Millisecond)
	prev := hc.Tick()
	for i := 1; i < iterations; i++ {
		f := hc.Tick()
		if delta := f.ChanSeq - prev.ChanSeq; delta != 1 {
			t.Fatalf("iter %d: f.ChanSeq=%d, prev.ChanSeq=%d, delta=%d, want 1",
				i, f.ChanSeq, prev.ChanSeq, delta)
		}
		prev = f
	}
}

// -----------------------------------------------------------------------------
// F-004 follow-through / Direction() accessor
// -----------------------------------------------------------------------------

// TestHalfChannelDirection verifies that Direction() returns the direction
// configured at construction for both Upstream and Downstream.
func TestHalfChannelDirection(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		dir  halfchannel.Direction
	}{
		{"upstream", halfchannel.Upstream},
		{"downstream", halfchannel.Downstream},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			hc := halfchannel.New(0x100, tc.dir, 10*time.Millisecond)
			if got := hc.Direction(); got != tc.dir {
				t.Errorf("Direction() = %v, want %v", got, tc.dir)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// F-005 / TickInterval() accessor
// -----------------------------------------------------------------------------

// TestHalfChannelTickInterval verifies that TickInterval() returns the exact
// duration passed to New.
func TestHalfChannelTickInterval(t *testing.T) {
	t.Parallel()

	const want = 17 * time.Millisecond
	hc := halfchannel.New(0x100, halfchannel.Upstream, want)
	if got := hc.TickInterval(); got != want {
		t.Errorf("TickInterval() = %v, want %v", got, want)
	}
}

// -----------------------------------------------------------------------------
// F-009 follow-through / ADR-008 constant regression
// -----------------------------------------------------------------------------

// TestTickIntervalConstants pins MinTickInterval and MaxTickInterval to the
// ADR-008 documented values. Any change must come with a matching ADR-008
// revision.
func TestTickIntervalConstants(t *testing.T) {
	t.Parallel()
	if halfchannel.MinTickInterval != 5*time.Millisecond {
		t.Errorf("MinTickInterval = %v, want 5ms", halfchannel.MinTickInterval)
	}
	if halfchannel.MaxTickInterval != 50*time.Millisecond {
		t.Errorf("MaxTickInterval = %v, want 50ms", halfchannel.MaxTickInterval)
	}
}

// TestProperty_VP051_Independence verifies that two channels with different
// ChanIDs maintain independent sequence spaces and clocks across interleaved
// ticks. VP-051: two channels maintain independent sequence spaces and clocks
// across interleaved ticks.
func TestProperty_VP051_Independence(t *testing.T) {
	t.Parallel()

	const iterations = 10_000

	a := halfchannel.New(3000, halfchannel.Upstream, 10*time.Millisecond)
	b := halfchannel.New(3001, halfchannel.Downstream, 20*time.Millisecond)

	// Track expected counts locally — avoids calling Seq() before a Tick()
	// would advance state unexpectedly.
	var ticksA, ticksB uint32

	for i := range iterations {
		// Alternate ticking a and b.
		if i%2 == 0 {
			a.Tick()
			ticksA++
			// B must still equal ticksB.
			if got := b.Seq(); got != ticksB {
				t.Fatalf("VP-051 violated at iteration %d: ticking A changed B.Seq to %d, want %d",
					i+1, got, ticksB)
			}
		} else {
			b.Tick()
			ticksB++
			// A must still equal ticksA.
			if got := a.Seq(); got != ticksA {
				t.Fatalf("VP-051 violated at iteration %d: ticking B changed A.Seq to %d, want %d",
					i+1, got, ticksA)
			}
		}
	}
}

// -----------------------------------------------------------------------------
// F-001 / BC-2.01.002 PC5 — MTU validation (Red Gate)
// -----------------------------------------------------------------------------

// TestMaxPayloadSizeConstant pins the BC-2.01.002 v1.4 PC5 conservative bound:
// uint16 max (65535) minus the worst-case channel header size (20 bytes when
// SACK_present=1). Yields 65515 bytes. Using the conservative (SACK=1) bound
// means the constant is safe for both SACK and non-SACK frames; no caller needs
// to choose between two constants at enqueue time.
//
// Red Gate: references halfchannel.MaxPayloadSize which does not exist until
// the implementer's commit lands.
func TestMaxPayloadSizeConstant(t *testing.T) {
	t.Parallel()
	// BC-2.01.002 PC5 conservative MaxPayloadSize: uint16 max (65535) minus
	// the worst-case channel header size (20 bytes when SACK_present=1).
	const want = 65535 - 20
	if halfchannel.MaxPayloadSize != want {
		t.Errorf("MaxPayloadSize = %d, want %d (BC-2.01.002 PC5 conservative: uint16_max - 20-byte SACK channel header)",
			halfchannel.MaxPayloadSize, want)
	}
}

// TestHalfChannelEnqueue_RejectsOversizedPayload asserts that Enqueue
// rejects payloads larger than MaxPayloadSize with ErrPayloadTooLarge
// (BC-2.01.002 PC5, F-001).
//
// Red Gate: references halfchannel.MaxPayloadSize and halfchannel.ErrPayloadTooLarge
// which do not exist until the implementer's commit lands.
func TestHalfChannelEnqueue_RejectsOversizedPayload(t *testing.T) {
	t.Parallel()
	hc := halfchannel.New(0x100, halfchannel.Upstream, 10*time.Millisecond)
	payload := make([]byte, halfchannel.MaxPayloadSize+1)
	err := hc.Enqueue(payload)
	if err == nil {
		t.Fatal("Enqueue with len(payload)=MaxPayloadSize+1 returned nil error, want ErrPayloadTooLarge")
	}
	if !errors.Is(err, halfchannel.ErrPayloadTooLarge) {
		t.Errorf("err = %v, want errors.Is(err, halfchannel.ErrPayloadTooLarge)", err)
	}
}

// TestHalfChannelEnqueue_AcceptsMaxSizePayload asserts that Enqueue accepts
// a payload of exactly MaxPayloadSize bytes (boundary test, BC-2.01.002 PC5).
//
// Red Gate: references halfchannel.MaxPayloadSize which does not exist until
// the implementer's commit lands.
func TestHalfChannelEnqueue_AcceptsMaxSizePayload(t *testing.T) {
	t.Parallel()
	hc := halfchannel.New(0x100, halfchannel.Upstream, 10*time.Millisecond)
	payload := make([]byte, halfchannel.MaxPayloadSize)
	if err := hc.Enqueue(payload); err != nil {
		t.Fatalf("Enqueue with len(payload)=MaxPayloadSize unexpected error: %v", err)
	}
	cf := hc.Tick()
	if len(cf.Payload) != halfchannel.MaxPayloadSize {
		t.Errorf("cf.Payload length = %d, want %d", len(cf.Payload), halfchannel.MaxPayloadSize)
	}
}

// TestProperty_VP018_EmptyFrameEmitsForNoPayload verifies VP-018: when Tick
// is called with no queued payload, the returned frame's Payload field has
// zero length and FrameType is FrameTypeEmptyTick. Property-style: table-
// driven over varied (chanID, direction) seeds to surface state-dependent
// regressions.
func TestProperty_VP018_EmptyFrameEmitsForNoPayload(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		chanID    uint32
		direction halfchannel.Direction
	}{
		{"upstream-low-id", 0x1, halfchannel.Upstream},
		{"upstream-high-id", 0xFFFFFFFF, halfchannel.Upstream},
		{"downstream-low-id", 0x1, halfchannel.Downstream},
		{"downstream-high-id", 0xFFFFFFFF, halfchannel.Downstream},
		{"upstream-zero-id", 0x0, halfchannel.Upstream},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			hc := halfchannel.New(tc.chanID, tc.direction, 10*time.Millisecond)
			// 100 consecutive ticks with no Enqueue — every frame must be empty-tick.
			for i := 0; i < 100; i++ {
				f := hc.Tick()
				if len(f.Payload) != 0 {
					t.Fatalf("iter %d: len(f.Payload) = %d, want 0", i, len(f.Payload))
				}
				if f.FrameType != halfchannel.FrameTypeEmptyTick {
					t.Fatalf("iter %d: f.FrameType = %#x, want FrameTypeEmptyTick (%#x)", i, f.FrameType, halfchannel.FrameTypeEmptyTick)
				}
				if f.ChanID != tc.chanID {
					t.Fatalf("iter %d: f.ChanID = %#x, want %#x", i, f.ChanID, tc.chanID)
				}
			}
		})
	}
}
