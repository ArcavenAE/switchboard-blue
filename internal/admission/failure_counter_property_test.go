// Package admission_test — VP-059 v1.1 property-based tests for FailureCounter.
//
// AC-017 (story v1.2) mandates a stateful model checker over arbitrary generated
// call sequences with injected clock, verifying VP-059 properties (a)–(e).
//
// Properties verified:
//
//	(a) E-ADM-017 fires exactly on the call that brings post-trim count to threshold.
//	(b) Subsequent calls in the same un-re-armed window do NOT fire E-ADM-017.
//	(c) After re-arm (drain-only: len(keep)==0 after trim under append-skip policy),
//	    the next threshold crossing fires E-ADM-017 again.
//	(d) Under a continuous stream at rate ≥ threshold/windowDuration, E-ADM-017
//	    alert count is ≥ 2 (counter never goes permanently silent).
//	(e) Live key count SourceCount() ≤ maxTrackedSources (65536) at all times.
//
// VP-059 specifies that properties (a)–(e) hold for ANY valid (threshold, window).
// Tests are therefore parameterized over multiple configurations:
//
//	{threshold:5, window:60s}   — production default
//	{threshold:3, window:30s}   — lower threshold, shorter window
//	{threshold:10, window:120s} — higher threshold, longer window
//
// Determinism: all tests use seeds derived from a fixed base (1337). No wall-clock
// or global-rand seeding. Fully reproducible across runs and goroutine schedules.
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

// ── propConfig — parameterized test configuration ────────────────────────────

// propConfig holds one (threshold, window) pair for VP-059 property tests.
// VP-059 properties (a)–(e) must hold for any valid configuration.
type propConfig struct {
	threshold int
	window    time.Duration
	name      string
}

// propConfigs is the set of configurations exercised by all property tests.
// Three distinct configs ensure threshold is genuinely varied across call sites
// (satisfying unparam) and broaden VP-059 coverage beyond the production default.
var propConfigs = []propConfig{
	{threshold: 5, window: 60 * time.Second, name: "threshold5_window60s"},
	{threshold: 3, window: 30 * time.Second, name: "threshold3_window30s"},
	{threshold: 10, window: 120 * time.Second, name: "threshold10_window120s"},
}

// ── VP-059 stateful model ─────────────────────────────────────────────────────

// modelState tracks the expected E-ADM-017 alert count for a single srcAddr,
// mirroring the drain-only re-arm + append-skip semantics of BC-2.05.005 v1.6.
//
// Model rules:
//  1. Trim entries where ts < now-window.
//  2. If firedAt is set and len(trim)==0: re-arm (clear firedAt).
//  3. If firedAt is zero: append now. Otherwise: skip (append-skip).
//  4. If post-trim+append count >= threshold AND firedAt is zero: fire; set firedAt=now.
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
// deterministic, reproducible call sequences with injected clock, across all
// propConfigs (threshold-invariance per VP-059 v1.1).
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
// Traces to VP-059 v1.1 properties (a)–(d); BC-2.05.005 PC-3; S-W3.05 AC-017.
func TestFailureCounter_PropertiesABCD(t *testing.T) {
	t.Parallel()

	const src = "deadbeefcafebabe"

	for _, cfg := range propConfigs {
		cfg := cfg // capture loop var

		// Property (a): alert fires exactly on the call that reaches threshold, not before.
		t.Run(fmt.Sprintf("fires_exactly_at_threshold/%s", cfg.name), func(t *testing.T) {
			t.Parallel()

			now := time.Unix(1000, 0)
			logger := &capturingLogger{}
			fc := admission.NewFailureCounter(cfg.threshold, cfg.window, logger,
				admission.WithNow(func() time.Time { return now }))

			model := newModelState(cfg.threshold, cfg.window)

			for i := 1; i < cfg.threshold; i++ {
				fc.RecordHMACFailure(src)
				model.record(now)
				if logger.alertCount() != 0 {
					t.Errorf("VP-059 (a) %s: call %d: expected 0 alerts before threshold, got %d",
						cfg.name, i, logger.alertCount())
				}
			}

			// threshold-th call must fire exactly one alert.
			fc.RecordHMACFailure(src)
			model.record(now)

			if logger.alertCount() != 1 {
				t.Errorf("VP-059 (a) %s: expected exactly 1 alert at threshold=%d, got %d",
					cfg.name, cfg.threshold, logger.alertCount())
			}
			if model.alertCount != logger.alertCount() {
				t.Errorf("VP-059 (a) %s: model/actual mismatch: model=%d actual=%d",
					cfg.name, model.alertCount, logger.alertCount())
			}
		})

		// Property (b): subsequent calls in same un-re-armed window do NOT re-fire.
		t.Run(fmt.Sprintf("suppressed_within_same_window/%s", cfg.name), func(t *testing.T) {
			t.Parallel()

			now := time.Unix(1000, 0)
			logger := &capturingLogger{}
			fc := admission.NewFailureCounter(cfg.threshold, cfg.window, logger,
				admission.WithNow(func() time.Time { return now }))

			model := newModelState(cfg.threshold, cfg.window)

			for range cfg.threshold {
				fc.RecordHMACFailure(src)
				model.record(now)
			}
			if logger.alertCount() != 1 {
				t.Fatalf("VP-059 (b) %s: setup: expected 1 alert, got %d", cfg.name, logger.alertCount())
			}

			for i := range 10 {
				fc.RecordHMACFailure(src)
				model.record(now)
				if logger.alertCount() != 1 {
					t.Errorf("VP-059 (b) %s: call %d: expected still 1 alert (suppressed), got %d",
						cfg.name, i+1, logger.alertCount())
				}
			}

			if model.alertCount != logger.alertCount() {
				t.Errorf("VP-059 (b) %s: model/actual mismatch: model=%d actual=%d",
					cfg.name, model.alertCount, logger.alertCount())
			}
		})

		// Property (c): after window drains (drain-only re-arm), next crossing fires again.
		t.Run(fmt.Sprintf("rearm_after_window_drain/%s", cfg.name), func(t *testing.T) {
			t.Parallel()

			base := time.Unix(1000, 0)
			now := base
			logger := &capturingLogger{}
			fc := admission.NewFailureCounter(cfg.threshold, cfg.window, logger,
				admission.WithNow(func() time.Time { return now }))

			model := newModelState(cfg.threshold, cfg.window)

			for range cfg.threshold {
				fc.RecordHMACFailure(src)
				model.record(now)
			}
			if logger.alertCount() != 1 {
				t.Fatalf("VP-059 (c) %s: setup: expected 1 alert after first batch, got %d",
					cfg.name, logger.alertCount())
			}

			// Advance clock past the full window so all pre-fire entries age out.
			now = base.Add(cfg.window + time.Second)

			for range cfg.threshold {
				fc.RecordHMACFailure(src)
				model.record(now)
			}
			if logger.alertCount() != 2 {
				t.Errorf("VP-059 (c) %s: expected 2 alerts after drain re-arm, got %d "+
					"(BC-2.05.005 v1.6 requires re-arm only when len(keep)==0 after trim)",
					cfg.name, logger.alertCount())
			}
			if model.alertCount != logger.alertCount() {
				t.Errorf("VP-059 (c) %s: model/actual mismatch: model=%d actual=%d",
					cfg.name, model.alertCount, logger.alertCount())
			}
		})

		// Property (d): under continuous stream ≥ threshold/window, alert count ≥ 2.
		t.Run(fmt.Sprintf("periodic_refire_sustained_attack/%s", cfg.name), func(t *testing.T) {
			t.Parallel()

			base := time.Unix(1000, 0)
			now := base
			logger := &capturingLogger{}
			fc := admission.NewFailureCounter(cfg.threshold, cfg.window, logger,
				admission.WithNow(func() time.Time { return now }))

			model := newModelState(cfg.threshold, cfg.window)

			for range cfg.threshold {
				fc.RecordHMACFailure(src)
				model.record(now)
			}
			if logger.alertCount() != 1 {
				t.Fatalf("VP-059 (d) %s: setup: expected 1 alert, got %d", cfg.name, logger.alertCount())
			}

			now = base.Add(cfg.window + time.Second)

			for range cfg.threshold {
				fc.RecordHMACFailure(src)
				model.record(now)
			}

			if logger.alertCount() < 2 {
				t.Errorf("VP-059 (d) %s: expected ≥2 alerts under sustained attack, got %d "+
					"— counter went permanently silent (BC-2.05.005 EC-009)",
					cfg.name, logger.alertCount())
			}
			if model.alertCount != logger.alertCount() {
				t.Errorf("VP-059 (d) %s: model/actual mismatch: model=%d actual=%d",
					cfg.name, model.alertCount, logger.alertCount())
			}
		})

		// Stateful model checker over a generated call sequence.
		// Seed is derived per-config: 1337 + index, so each config exercises a
		// distinct operation sequence while remaining fully deterministic.
		t.Run(fmt.Sprintf("stateful_model_generated_sequence/%s", cfg.name), func(t *testing.T) {
			t.Parallel()

			const baseSeed = uint64(1337)
			var configIndex uint64
			for i, c := range propConfigs {
				if c.name == cfg.name {
					configIndex = uint64(i)
					break
				}
			}
			prng := newLCG(baseSeed + configIndex)

			base := time.Unix(10000, 0)
			now := base
			logger := &capturingLogger{}
			fc := admission.NewFailureCounter(cfg.threshold, cfg.window, logger,
				admission.WithNow(func() time.Time { return now }))

			model := newModelState(cfg.threshold, cfg.window)

			// maxDelta: clock advances up to 2× the window to exercise both
			// within-window and full-drain scenarios deterministically.
			maxDeltaSec := int(cfg.window.Seconds()) * 2

			const numOps = 200
			for i := range numOps {
				op := prng.intn(2) // 0 = advance clock, 1 = RecordHMACFailure
				if op == 0 {
					delta := time.Duration(prng.intn(maxDeltaSec+1)) * time.Second
					now = now.Add(delta)
				} else {
					fc.RecordHMACFailure(src)
					model.record(now)
				}

				if got := logger.alertCount(); got != model.alertCount {
					t.Errorf("VP-059 stateful model %s: op %d (op=%d): actual=%d != model=%d\n"+
						"  impl re-arm/append semantics diverge from drain-only model",
						cfg.name, i, op, got, model.alertCount)
					return
				}
			}
		})
	}
}

// ── TestFailureCounter_PropertyE_MemoryBound ──────────────────────────────────

// TestFailureCounter_PropertyE_MemoryBound verifies VP-059 property (e):
// SourceCount() <= maxTrackedSources regardless of distinct source count.
//
// Adversarial injection: maxTrackedSources+1000 distinct srcAddrs. Checks at the
// cap boundary and final insertion; per-call checking is avoided to keep runtime
// bounded given the O(N) LRU scan in V1.
//
// Note: uses SourceCount() — the correct accessor per the production impl.
// VP-059 v1.1 harness skeleton referenced "TrackedSourceCount()"; reconciled.
//
// Traces to VP-059 v1.1 property (e); BC-2.05.005 EC-010; S-W3.05 AC-017.
func TestFailureCounter_PropertyE_MemoryBound(t *testing.T) {
	// Not t.Parallel() — inserts 66,536 distinct sources; keep single-threaded.

	const threshold = 5
	const windowDuration = 60 * time.Second
	const maxTrackedSources = 65536
	// Insert maxTrackedSources+1000 sources: exercises the LRU eviction path
	// repeatedly while staying well under 5s given the O(N) LRU scan impl.
	const adversarialSources = maxTrackedSources + 1000

	now := time.Unix(1000, 0)
	logger := &capturingLogger{}
	fc := admission.NewFailureCounter(threshold, windowDuration, logger,
		admission.WithNow(func() time.Time { return now }))

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
// property harness: the threshold-th failure at exactly windowDuration after the
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

	// Record failures 1–(threshold-1) at T=0.
	for range threshold - 1 {
		fc.RecordHMACFailure(src)
	}

	// Advance clock to exactly T=windowDuration (boundary).
	// Under strict less-than trim: entry at T=0 is NOT trimmed (not strictly < T=0).
	// Post-trim count = threshold-1; after append = threshold → alert fires.
	now = base.Add(windowDuration)
	fc.RecordHMACFailure(src)

	if logger.alertCount() != 1 {
		t.Errorf("VP-059 EC-008 boundary: expected 1 alert (boundary entry kept), got %d "+
			"— possible <= trim defect (BC-2.05.005 EC-008 requires strictly-less-than trim)",
			logger.alertCount())
	}
}
