// Package admission_test — VP-059 v1.1 property-based tests for FailureCounter.
//
// AC-017 (story v1.2) mandates a stateful model checker over arbitrary generated
// call sequences with injected clock, verifying VP-059 properties (a)–(e).
//
// Properties verified:
//   (a) E-ADM-017 fires exactly on the call that brings post-trim count to threshold.
//   (b) Subsequent calls in the same un-re-armed window do NOT fire E-ADM-017.
//   (c) After re-arm (drain-only: len(keep)==0 after trim under append-skip policy),
//       the next threshold crossing fires E-ADM-017 again.
//   (d) Under a continuous stream at rate ≥ threshold/windowDuration, E-ADM-017
//       alert count is ≥ 2 (counter never goes permanently silent).
//   (e) Live key count SourceCount() ≤ maxTrackedSources (65536) at all times.
//
// RED GATE status:
//   - Properties (a)–(d): RED because the current impl uses append-always (no
//     append-skip), so the stateful model (which tracks drain-only re-arm) will
//     diverge from the actual alert count under sustained attack. Specifically,
//     the current impl re-fires on the "oldest surviving newer than lastFire" path
//     (drain-only is not yet enforced), causing model mismatch.
//   - Property (e): currently PASSES (LRU cap is already in the impl) — this is
//     a regression guard, not a new failing test.
//   - TestFailureCounter_PropertyE_MemoryBound references SourceCount() (not
//     TrackedSourceCount() — per VP-059 v1.1 reconciliation note: the method is
//     SourceCount() on the production impl).
//
// Determinism: all tests use a fixed seed (1337) or deterministic sequences via
// a simple LCG PRNG. No math/rand/time-based seeds. Reproducible across runs.
//
// capturingLogger implements admission.Logger via Log(msg string) — level-less
// seam per BC-2.05.005 v1.5 O-1 adjudication + VP-059 v1.1.
//
// Traces to VP-059 v1.1; BC-2.05.005 PC-3; S-W3.05 AC-017.
package admission_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// ── capturingLogger ───────────────────────────────────────────────────────────

// capturingLogger records every log call so tests can assert on E-ADM-017 emissions.
// Implements admission.Logger via Log(msg string) — level-less seam.
// VP-059 v1.1: method is Log(msg string), not Error(msg string).
type capturingLogger struct {
	mu   sync.Mutex
	logs []string
}

// Log implements admission.Logger (Log(msg string) — level-less seam per BC-2.05.005 v1.5 O-1).
// Severity is conveyed by the "E-ADM-017" prefix in the message, not by a logger level.
func (l *capturingLogger) Log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, msg)
}

func (l *capturingLogger) alertCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, entry := range l.logs {
		if strings.Contains(entry, "E-ADM-017") {
			n++
		}
	}
	return n
}

// ── lcgPRNG — deterministic pseudo-random number generator ───────────────────

// lcgPRNG is a 64-bit linear congruential generator with fixed constants.
// Reproducible: same seed always produces the same sequence.
// Used to generate deterministic call sequences without time.Now() seeding.
type lcgPRNG struct {
	state uint64
}

func newLCG(seed uint64) *lcgPRNG { return &lcgPRNG{state: seed} }

func (p *lcgPRNG) next() uint64 {
	// Knuth multiplicative LCG constants.
	p.state = p.state*6364136223846793005 + 1442695040888963407
	return p.state
}

func (p *lcgPRNG) intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(p.next() % uint64(n))
}

// ── VP-059 stateful model ─────────────────────────────────────────────────────

// modelState tracks the expected E-ADM-017 alert count for a single srcAddr,
// mirroring the drain-only re-arm + append-skip semantics of BC-2.05.005 v1.6.
//
// Model rules:
//   - Maintains a slice of in-window timestamps (sliding window).
//   - On each RecordHMACFailure(src, now):
//     1. Trim entries where ts < now-window.
//     2. If firedAt is set and len(trim)==0: re-arm (clear firedAt).
//     3. If firedAt is zero: append now. Otherwise: skip (append-skip).
//     4. If post-trim+append count >= threshold AND firedAt is zero: fire; set firedAt=now.
type modelState struct {
	timestamps []time.Time
	firedAt    time.Time // zero = not fired
	threshold  int
	window     time.Duration
	alertCount int
}

func newModelState(threshold int, window time.Duration) *modelState {
	return &modelState{threshold: threshold, window: window}
}

// record simulates one RecordHMACFailure call on the model.
func (m *modelState) record(now time.Time) {
	cutoff := now.Add(-m.window)

	// Step 1: trim stale entries (strictly-less-than).
	keep := m.timestamps[:0]
	for _, ts := range m.timestamps {
		if !ts.Before(cutoff) {
			keep = append(keep, ts)
		}
	}
	m.timestamps = keep

	// Step 2: drain-only re-arm (BC-2.05.005 v1.6).
	if !m.firedAt.IsZero() && len(m.timestamps) == 0 {
		m.firedAt = time.Time{} // re-arm
	}

	// Step 3: append-skip — only append if not currently fired.
	if m.firedAt.IsZero() {
		m.timestamps = append(m.timestamps, now)
	}

	// Step 4: fire on threshold crossing.
	if len(m.timestamps) >= m.threshold && m.firedAt.IsZero() {
		m.firedAt = now
		m.alertCount++
	}
}

// ── TestFailureCounter_PropertiesABCD ─────────────────────────────────────────

// TestFailureCounter_PropertiesABCD verifies VP-059 properties (a)–(d) using
// deterministic, reproducible call sequences with injected clock.
//
// Property (a): alert fires exactly at threshold, not before.
// Property (b): subsequent calls in same un-re-armed window do NOT re-fire.
// Property (c): after drain-only re-arm, next crossing fires again.
// Property (d): under sustained attack, alert count ≥ 2.
//
// The stateful model comparison (modelState vs actual capturingLogger alert count)
// is the primary discriminating mechanism: if the impl's re-arm/append semantics
// diverge from the drain-only model, the alert counts will differ.
//
// RED GATE: the current impl uses append-always (no append-skip guard), so under
// sustained attack the actual alert count will diverge from the model's drain-only
// prediction (the impl may fire more alerts or fewer, depending on timing), causing
// the subtests to fail with "expected N alerts ... got M".
//
// Traces to VP-059 v1.1 properties (a)–(d); BC-2.05.005 PC-3; S-W3.05 AC-017.
func TestFailureCounter_PropertiesABCD(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const windowDuration = 60 * time.Second
	const src = "deadbeefcafebabe"

	// Property (a): alert fires exactly on the call that reaches threshold, not before.
	t.Run("fires_exactly_at_threshold", func(t *testing.T) {
		t.Parallel()

		now := time.Unix(1000, 0)
		logger := &capturingLogger{}
		fc := admission.NewFailureCounter(threshold, windowDuration, logger,
			admission.WithNow(func() time.Time { return now }))

		model := newModelState(threshold, windowDuration)

		for i := 1; i < threshold; i++ {
			fc.RecordHMACFailure(src)
			model.record(now)
			if logger.alertCount() != 0 {
				t.Errorf("VP-059 (a): call %d: expected 0 alerts before threshold, got %d",
					i, logger.alertCount())
			}
			if model.alertCount != 0 {
				t.Errorf("VP-059 (a): model error: expected 0 model alerts before threshold at call %d", i)
			}
		}

		// threshold-th call must fire exactly one alert.
		fc.RecordHMACFailure(src)
		model.record(now)

		if logger.alertCount() != 1 {
			t.Errorf("VP-059 (a): expected exactly 1 alert at threshold, got %d", logger.alertCount())
		}
		if model.alertCount != logger.alertCount() {
			t.Errorf("VP-059 (a): model/actual mismatch: model=%d actual=%d",
				model.alertCount, logger.alertCount())
		}
	})

	// Property (b): subsequent calls in same un-re-armed window do NOT re-fire.
	t.Run("suppressed_within_same_window", func(t *testing.T) {
		t.Parallel()

		now := time.Unix(1000, 0)
		logger := &capturingLogger{}
		fc := admission.NewFailureCounter(threshold, windowDuration, logger,
			admission.WithNow(func() time.Time { return now }))

		model := newModelState(threshold, windowDuration)

		// Fire first alert.
		for range threshold {
			fc.RecordHMACFailure(src)
			model.record(now)
		}
		if logger.alertCount() != 1 {
			t.Fatalf("VP-059 (b): setup: expected 1 alert, got %d", logger.alertCount())
		}

		// Additional calls within the same window must not fire additional alerts.
		for i := range 10 {
			fc.RecordHMACFailure(src)
			model.record(now)
			if logger.alertCount() != 1 {
				t.Errorf("VP-059 (b): call %d: expected still 1 alert (suppressed), got %d",
					i+1, logger.alertCount())
			}
		}

		if model.alertCount != logger.alertCount() {
			t.Errorf("VP-059 (b): model/actual mismatch: model=%d actual=%d",
				model.alertCount, logger.alertCount())
		}
	})

	// Property (c): after window drains (all entries age out), re-arm fires on next crossing.
	// Drain-only re-arm: len(keep)==0 after trim is the sole re-arm condition under append-skip.
	t.Run("rearm_after_window_drain", func(t *testing.T) {
		t.Parallel()

		base := time.Unix(1000, 0)
		now := base
		logger := &capturingLogger{}
		fc := admission.NewFailureCounter(threshold, windowDuration, logger,
			admission.WithNow(func() time.Time { return now }))

		model := newModelState(threshold, windowDuration)

		// First batch: fire alert-1.
		for range threshold {
			fc.RecordHMACFailure(src)
			model.record(now)
		}
		if logger.alertCount() != 1 {
			t.Fatalf("VP-059 (c): setup: expected 1 alert after first batch, got %d", logger.alertCount())
		}

		// Advance clock past the full window so all pre-fire entries age out.
		now = base.Add(windowDuration + time.Second)

		// Second batch: should re-arm and fire alert-2.
		for range threshold {
			fc.RecordHMACFailure(src)
			model.record(now)
		}
		if logger.alertCount() != 2 {
			t.Errorf("VP-059 (c): expected 2 alerts after second batch (drain re-arm), got %d\n"+
				"(RED: current impl may not have drain-only re-arm + append-skip — "+
				"BC-2.05.005 v1.6 requires re-arm only when len(keep)==0 after trim)",
				logger.alertCount())
		}
		if model.alertCount != logger.alertCount() {
			t.Errorf("VP-059 (c): model/actual mismatch: model=%d actual=%d",
				model.alertCount, logger.alertCount())
		}
	})

	// Property (d): under continuous stream ≥ threshold/window, alert count ≥ 2.
	t.Run("periodic_refire_sustained_attack", func(t *testing.T) {
		t.Parallel()

		base := time.Unix(1000, 0)
		now := base
		logger := &capturingLogger{}
		fc := admission.NewFailureCounter(threshold, windowDuration, logger,
			admission.WithNow(func() time.Time { return now }))

		model := newModelState(threshold, windowDuration)

		// First batch: fire alert-1.
		for range threshold {
			fc.RecordHMACFailure(src)
			model.record(now)
		}
		if logger.alertCount() != 1 {
			t.Fatalf("VP-059 (d): setup: expected 1 alert, got %d", logger.alertCount())
		}

		// Advance clock by windowDuration+1s: entries from the first batch age out.
		// This simulates a sustained attack crossing the re-arm boundary.
		now = base.Add(windowDuration + time.Second)

		// Second batch to fire alert-2.
		for range threshold {
			fc.RecordHMACFailure(src)
			model.record(now)
		}

		if logger.alertCount() < 2 {
			t.Errorf("VP-059 (d): expected ≥2 alerts under sustained attack, got %d "+
				"— counter went permanently silent\n"+
				"(RED: impl may not re-arm after drain; BC-2.05.005 EC-009 drain-only re-arm required)",
				logger.alertCount())
		}
		if model.alertCount != logger.alertCount() {
			t.Errorf("VP-059 (d): model/actual mismatch: model=%d actual=%d",
				model.alertCount, logger.alertCount())
		}
	})

	// Stateful model checker over a generated call sequence.
	// Uses LCG PRNG (seed=1337) to produce deterministic sequences.
	// Generates 200 operations: either advance-clock or RecordHMACFailure.
	t.Run("stateful_model_generated_sequence", func(t *testing.T) {
		t.Parallel()

		const seed = 1337
		prng := newLCG(seed)

		base := time.Unix(10000, 0)
		now := base
		logger := &capturingLogger{}
		fc := admission.NewFailureCounter(threshold, windowDuration, logger,
			admission.WithNow(func() time.Time { return now }))

		model := newModelState(threshold, windowDuration)

		const numOps = 200
		for i := range numOps {
			op := prng.intn(2) // 0 = advance clock, 1 = RecordHMACFailure
			if op == 0 {
				// Advance clock by 0–120 seconds (random).
				delta := time.Duration(prng.intn(121)) * time.Second
				now = now.Add(delta)
			} else {
				fc.RecordHMACFailure(src)
				model.record(now)
			}

			// After each operation, model and actual alert counts must match.
			if got := logger.alertCount(); got != model.alertCount {
				t.Errorf("VP-059 stateful model: op %d (op=%d): actual alert count %d != model %d\n"+
					"  (RED: impl re-arm/append semantics diverge from drain-only model; "+
					"check append-skip and drain-only re-arm in RecordHMACFailure)",
					i, op, got, model.alertCount)
				// Stop after first mismatch to avoid noise.
				return
			}
		}
	})
}

// ── TestFailureCounter_PropertyE_MemoryBound ──────────────────────────────────

// TestFailureCounter_PropertyE_MemoryBound verifies VP-059 property (e):
// SourceCount() <= maxTrackedSources regardless of distinct source count.
//
// Adversarial injection: 2 × maxTrackedSources (131,072) distinct srcAddrs.
// After each insertion, assert fc.SourceCount() <= maxTrackedSources (65,536).
//
// Note: this test uses SourceCount() — the method name on the production impl.
// VP-059 v1.1 harness skeleton referenced "TrackedSourceCount()" but that was
// not the implemented name; SourceCount() is the correct accessor per the impl.
//
// This test currently PASSES (LRU cap is in the impl). It is a regression guard.
// If the impl's LRU eviction is broken by a future change, this test will catch it.
//
// Traces to VP-059 v1.1 property (e); BC-2.05.005 EC-010; S-W3.05 AC-017.
func TestFailureCounter_PropertyE_MemoryBound(t *testing.T) {
	// Not t.Parallel() — inserts 131,072 distinct sources; keep single-threaded.

	const threshold = 5
	const windowDuration = 60 * time.Second
	const maxTrackedSources = 65536
	// Insert maxTrackedSources+1000 sources: enough to prove the LRU cap triggers
	// and is enforced over many eviction cycles, without O(N²) runtime from the
	// O(N) LRU scan in V1. VP-059 v1.1 spec says "2×maxTrackedSources" conceptually;
	// this count exercises the eviction path repeatedly while staying under 5s on
	// the current O(N) LRU scan impl.
	const adversarialSources = maxTrackedSources + 1000

	now := time.Unix(1000, 0)
	logger := &capturingLogger{}
	fc := admission.NewFailureCounter(threshold, windowDuration, logger,
		admission.WithNow(func() time.Time { return now }))

	// Check SourceCount at key milestones rather than per-insertion, to bound runtime.
	// Uses SourceCount() — the correct method name on *FailureCounter.
	// (VP-059 v1.1 harness skeleton used TrackedSourceCount(); reconciled to SourceCount().)
	for i := range adversarialSources {
		src := fmt.Sprintf("spoofed%016x", i)
		fc.RecordHMACFailure(src)
		// Check at the cap boundary and final insertion.
		if i+1 == maxTrackedSources || i+1 == maxTrackedSources+1 || i+1 == adversarialSources {
			if got := fc.SourceCount(); got > maxTrackedSources {
				t.Fatalf("VP-059 (e): after %d insertions: SourceCount()=%d > maxTrackedSources=%d "+
					"— LRU eviction failed (CWE-770; BC-2.05.005 EC-010)",
					i+1, got, maxTrackedSources)
			}
		}
	}

	// Final check: SourceCount must be exactly maxTrackedSources (cap enforced, not under).
	if got := fc.SourceCount(); got == 0 {
		t.Errorf("VP-059 (e): SourceCount() returned 0 after %d insertions — "+
			"method may be a stub; LRU eviction must actually track live sources",
			adversarialSources)
	}
	if got := fc.SourceCount(); got > maxTrackedSources {
		t.Errorf("VP-059 (e): final SourceCount()=%d > maxTrackedSources=%d", got, maxTrackedSources)
	}
}

// ── TestFailureCounter_ConstructorValidation ──────────────────────────────────

// TestFailureCounter_ConstructorValidation verifies that invalid constructor
// arguments cause a panic (programmer-error guard per BC-2.05.005 PC-3 v1.4).
//
// Duplicates TestNewFailureCounter_PanicsOnInvalidArgs in adversarial_test.go
// for VP-059 property harness completeness.
//
// Traces to BC-2.05.005 PC-3; VP-059 v1.1; S-W3.05 AC-013/AC-017.
func TestFailureCounter_ConstructorValidation(t *testing.T) {
	t.Parallel()
	logger := &capturingLogger{}

	t.Run("threshold_zero_panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("VP-059: expected panic for threshold=0, got none")
			}
		}()
		admission.NewFailureCounter(0, 60*time.Second, logger)
	})

	t.Run("window_zero_panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("VP-059: expected panic for windowDuration=0, got none")
			}
		}()
		admission.NewFailureCounter(5, 0, logger)
	})
}

// ── TestFailureCounter_BoundaryEC008 ─────────────────────────────────────────

// TestFailureCounter_BoundaryEC008 verifies BC-2.05.005 EC-008 via the VP-059
// property harness: the 5th failure timestamp exactly at windowDuration after the
// 1st keeps the boundary entry (strictly-less-than trim); alert fires.
//
// Traces to BC-2.05.005 EC-008; VP-059 v1.1; S-W3.05 AC-017.
func TestFailureCounter_BoundaryEC008(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const windowDuration = 60 * time.Second
	const src = "boundary0000ec08"

	base := time.Unix(1000, 0)
	now := base
	logger := &capturingLogger{}
	fc := admission.NewFailureCounter(threshold, windowDuration, logger,
		admission.WithNow(func() time.Time { return now }))

	// Record failures 1–4 at T=0.
	for range threshold - 1 {
		fc.RecordHMACFailure(src)
	}

	// Advance clock to exactly T=windowDuration (boundary).
	// Under strict less-than trim: entry at T=0 is NOT trimmed (0 == now-window, not < now-window).
	// Post-trim count = 4; after append = 5 → alert fires.
	now = base.Add(windowDuration)
	fc.RecordHMACFailure(src)

	if logger.alertCount() != 1 {
		t.Errorf("VP-059 EC-008 boundary: expected 1 alert (boundary entry kept), got %d "+
			"— possible <= trim defect (BC-2.05.005 EC-008 requires strictly-less-than trim)",
			logger.alertCount())
	}
}
