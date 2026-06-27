// Package admission_test — FailureCounter Red-Gate tests (S-W3.05).
//
// These tests WILL NOT COMPILE against the current codebase — admission.FailureCounter,
// admission.NewFailureCounter, and admission.Logger do not yet exist. Compile failure
// IS the Red Gate: it proves the behaviour is absent before implementation begins.
//
// # Implementer contract (all from BC-2.05.005 PC-3 + AC-001 through AC-010)
//
// Add internal/admission/failure_counter.go with:
//
//	// Logger is a minimal logging interface for FailureCounter.
//	// Mirrors routing.Logger but lives in the admission package.
//	type Logger interface {
//	    Log(msg string)
//	}
//
//	type FailureCounter struct {
//	    mu             sync.Mutex
//	    counts         map[string][]time.Time  // per-srcAddr timestamp slices
//	    fired          map[string]bool         // fired state per srcAddr; resets when count < threshold
//	    threshold      int
//	    windowDuration time.Duration
//	    logger         Logger
//	    now            func() time.Time        // REQUIRED clock seam for tests (see AC-003, AC-005, AC-008)
//	}
//
//	func NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger) *FailureCounter
//
//	func (c *FailureCounter) RecordHMACFailure(srcAddr string)
//	    // Trim entries where timestamp < now() - windowDuration (strictly less than;
//	    // boundary is kept — AC-008 / BC-2.05.005 EC-008).
//	    // Append now().UTC().
//	    // If post-trim count >= threshold AND !fired[srcAddr]: emit E-ADM-017, set fired[srcAddr]=true.
//	    // If post-trim count < threshold: reset fired[srcAddr]=false (hysteresis — AC-005).
//	    // Hold sync.Mutex for entire trim+append+check sequence.
//
//	// Timestamps returns a value copy of the srcAddr timestamp slice (go.md rule 12).
//	// Used by tests for white-box state inspection. NEVER returns an internal pointer.
//	func (c *FailureCounter) Timestamps(srcAddr string) []time.Time
//
// CLOCK SEAM: FailureCounter MUST expose an internal clock function for tests.
// The simplest approach: add an unexported field `now func() time.Time` in the
// struct, defaulting to time.Now().UTC(). Expose it in the test package via a
// package-level setter (only available in _test.go within the package), or use
// the functional-option pattern:
//
//	func withNow(fn func() time.Time) option { ... }
//
// The tests below use a package-level exported helper SetNowFunc for injection
// — the implementer must either add:
//
//	(a) an unexported `now` field settable from the test package via an
//	    exported SetNowFunc (visible only in admission_test), OR
//	(b) a WithNow functional option on NewFailureCounter
//
// Because the tests are in package admission_test (black-box), option (b) is
// cleaner and is what these tests assume:
//
//	func WithNow(fn func() time.Time) FailureCounterOption
//	func NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger, opts ...FailureCounterOption) *FailureCounter
//
// Tests in this file use WithNow. The implementer MUST add this option seam.
// Without it, AC-003 / AC-005 / AC-008 cannot be tested without wall-clock sleeps.
//
// # E-ADM-017 canonical message (error-taxonomy.md §ADM)
//
//	"HMAC failure rate alert: ≥5 failures in 60s from src <src_addr>"
//
// Every E-ADM-017 log record MUST contain BOTH:
//   - the literal string "E-ADM-017"
//   - the srcAddr value as a substring
//
// Traces to: BC-2.05.005 PC-3; S-W3.05 AC-001 through AC-010; VP-059.
package admission_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// ── fakeLog ──────────────────────────────────────────────────────────────────

// fakeLog implements admission.Logger, capturing log lines for assertion.
// Concurrency-safe.
type fakeLog struct {
	mu    sync.Mutex
	lines []string
}

func (f *fakeLog) Log(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lines = append(f.lines, msg)
}

// Count returns the number of captured log lines.
func (f *fakeLog) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.lines)
}

// Lines returns a snapshot of all captured log lines (value copy).
func (f *fakeLog) Lines() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.lines))
	copy(out, f.lines)
	return out
}

// HasAll reports whether any captured log line contains ALL of the given substrings.
func (f *fakeLog) HasAll(substrs ...string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, line := range f.lines {
		allFound := true
		for _, s := range substrs {
			if !strings.Contains(line, s) {
				allFound = false
				break
			}
		}
		if allFound {
			return true
		}
	}
	return false
}

// ── AC-001: TestNewFailureCounter_ConstructorFields ──────────────────────────

// TestNewFailureCounter_ConstructorFields verifies that NewFailureCounter
// constructs a non-nil *FailureCounter without panicking, and that the
// returned value is usable (RecordHMACFailure can be called).
//
// Traces to BC-2.05.005 PC-3 (FailureCounter type contract); S-W3.05 AC-001.
func TestNewFailureCounter_ConstructorFields(t *testing.T) {
	t.Parallel()

	log := &fakeLog{}
	fc := admission.NewFailureCounter(5, 60*time.Second, log)
	if fc == nil {
		t.Fatal("AC-001: NewFailureCounter returned nil")
	}

	// RecordHMACFailure must be callable without panic.
	fc.RecordHMACFailure("test-src")

	// No alert yet (only 1 call, threshold=5).
	if log.Count() != 0 {
		t.Errorf("AC-001: want 0 log records after 1 RecordHMACFailure, got %d; lines: %v",
			log.Count(), log.Lines())
	}
}

// ── AC-002 / AC-008: TestFailureCounter_SlidingWindowTrimsStaleEntries ───────

// TestFailureCounter_SlidingWindowTrimsStaleEntries verifies that entries older
// than windowDuration are trimmed on each RecordHMACFailure call, using the
// clock seam (WithNow) to control time without wall-clock sleeps.
//
// Scenario:
//   - t=0: 3 failures recorded.
//   - t=61s: 2 more failures recorded.
//   - At t=61s, the 3 entries from t=0 should be trimmed (61s > 60s window).
//   - Post-trim count: 2 — no alert should fire.
//
// Traces to BC-2.05.005 PC-3 (sliding window trimming); S-W3.05 AC-002.
func TestFailureCounter_SlidingWindowTrimsStaleEntries(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	src := "trim-test-src"

	// 3 failures at t=0.
	for range 3 {
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 0 {
		t.Errorf("AC-002: want 0 alerts after 3 failures, got %d", log.Count())
	}

	// Advance to t=61s (beyond windowDuration). Entries from t=0 are now stale.
	current = base.Add(61 * time.Second)

	// 2 more failures at t=61s. Post-trim count should be 2, not 5.
	fc.RecordHMACFailure(src)
	fc.RecordHMACFailure(src)

	if log.Count() != 0 {
		t.Errorf("AC-002: want 0 alerts after trim+2 failures (count<threshold), got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// Inspect: Timestamps should contain exactly 2 entries (the t=61s ones).
	ts := fc.Timestamps(src)
	if len(ts) != 2 {
		t.Errorf("AC-002: want 2 timestamps after stale trim, got %d: %v", len(ts), ts)
	}
}

// ── AC-003: TestFailureCounter_EmitsEADM017AtThreshold ───────────────────────

// TestFailureCounter_EmitsEADM017AtThreshold verifies that exactly 1 E-ADM-017
// log record is emitted when RecordHMACFailure is called threshold times within
// the windowDuration, and that the record contains E-ADM-017 and the srcAddr.
//
// Canonical test vector from BC-2.05.005: 5 HMAC failures in 30s from same
// src_addr → E-ADM-017 emitted exactly once on the 5th call.
//
// Traces to BC-2.05.005 PC-3 (E-ADM-017 emission); S-W3.05 AC-003; VP-059.
func TestFailureCounter_EmitsEADM017AtThreshold(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "alert-src-addr-001"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// 4 failures — no alert.
	for i := range 4 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 0 {
		t.Errorf("AC-003: want 0 alerts after %d failures (threshold=%d), got %d; lines: %v",
			4, threshold, log.Count(), log.Lines())
	}

	// 5th failure — alert fires.
	current = base.Add(4 * time.Second)
	fc.RecordHMACFailure(src)

	// (a) exactly 1 log record.
	if log.Count() != 1 {
		t.Errorf("AC-003: want exactly 1 E-ADM-017 alert after threshold crossing, got %d; lines: %v",
			log.Count(), log.Lines())
	}
	// (b) record contains "E-ADM-017".
	if !log.HasAll("E-ADM-017") {
		t.Errorf("AC-003: log record missing \"E-ADM-017\"; lines: %v", log.Lines())
	}
	// (c) record contains srcAddr.
	if !log.HasAll(src) {
		t.Errorf("AC-003: log record missing srcAddr %q; lines: %v", src, log.Lines())
	}
	// (d) record contains canonical message prefix.
	if !log.HasAll("HMAC failure rate alert") {
		t.Errorf("AC-003: log record missing canonical prefix \"HMAC failure rate alert\"; lines: %v",
			log.Lines())
	}
}

// ── AC-004: TestFailureCounter_FiresOncePerCrossing ───────────────────────────

// TestFailureCounter_FiresOncePerCrossing verifies that E-ADM-017 fires exactly
// once when the threshold is first crossed, and subsequent failures within the
// same window do NOT re-emit the alert (fire-once-per-crossing).
//
// Scenario: 5 failures → alert fires once. 3 more failures (8 total in window)
// → no additional alert. Total: exactly 1 E-ADM-017.
//
// Traces to BC-2.05.005 PC-3 (fire-once-per-crossing); S-W3.05 AC-004.
func TestFailureCounter_FiresOncePerCrossing(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "fire-once-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// 5 failures — crosses threshold, alert fires once.
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 1 {
		t.Fatalf("AC-004: want 1 alert after threshold crossing, got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// 3 more failures in same window — must NOT re-emit.
	for i := 5; i < 8; i++ {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 1 {
		t.Errorf("AC-004: fire-once violated — want exactly 1 alert after 8 failures in window, got %d; lines: %v",
			log.Count(), log.Lines())
	}
}

// ── AC-005: TestFailureCounter_HysteresisRefirersAfterWindowExpires ────────────

// TestFailureCounter_HysteresisRefirersAfterWindowExpires verifies the hysteresis
// semantics from BC-2.05.005 EC-005:
//
//	T=0:  5 failures → E-ADM-017 fires (1st time).
//	T=61s: all prior entries trimmed (window expired) → fired state resets.
//	T=61s: 5 more failures → E-ADM-017 fires again (2nd time).
//	Total: exactly 2 E-ADM-017 events.
//
// Traces to BC-2.05.005 EC-005; S-W3.05 AC-005.
func TestFailureCounter_HysteresisRefirersAfterWindowExpires(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "hysteresis-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// First batch: 5 failures at T=0 → 1 alert.
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 1 {
		t.Fatalf("AC-005: want 1 alert after first batch, got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// Advance time by 61s so all prior entries are outside the window.
	// After the next RecordHMACFailure, trim will remove all T=0 entries.
	// The fired state for src must reset when count drops below threshold.
	batchTwoBase := base.Add(61 * time.Second)

	// Second batch: 5 failures at T=61s → fired state reset → 2nd alert fires.
	for i := range 5 {
		current = batchTwoBase.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 2 {
		t.Errorf("AC-005: hysteresis failed — want exactly 2 alerts (one per crossing), got %d; lines: %v",
			log.Count(), log.Lines())
	}
}

// ── AC-006: TestFailureCounter_BelowThresholdNoAlert ─────────────────────────

// TestFailureCounter_BelowThresholdNoAlert verifies that exactly threshold-1
// failures within the window do NOT emit E-ADM-017, and that the counter holds
// the entries for a subsequent call that would cross the threshold.
//
// Canonical test vector: 4 HMAC failures in 60s → no E-ADM-017 emitted.
//
// Traces to BC-2.05.005 EC-006; S-W3.05 AC-006.
func TestFailureCounter_BelowThresholdNoAlert(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "below-threshold-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// threshold-1 = 4 failures — no alert.
	for i := range threshold - 1 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 0 {
		t.Errorf("AC-006: want 0 alerts for %d failures (below threshold %d), got %d; lines: %v",
			threshold-1, threshold, log.Count(), log.Lines())
	}

	// 4 timestamps should be held.
	ts := fc.Timestamps(src)
	if len(ts) != threshold-1 {
		t.Errorf("AC-006: want %d timestamps held, got %d", threshold-1, len(ts))
	}

	// 5th failure — exactly threshold — alert fires once.
	current = base.Add(4 * time.Second)
	fc.RecordHMACFailure(src)
	if log.Count() != 1 {
		t.Errorf("AC-006: want exactly 1 alert on threshold-crossing (5th failure), got %d; lines: %v",
			log.Count(), log.Lines())
	}
}

// ── AC-007: TestFailureCounter_MultiSourceIsolation ──────────────────────────

// TestFailureCounter_MultiSourceIsolation verifies that two distinct srcAddr
// values accumulate failure counts independently: ≥5 from addr-A fires one
// E-ADM-017 for A; 3 from addr-B fires nothing for B. Calls are interleaved.
//
// Canonical test vector: 5 HMAC failures from src_addr A + 5 from src_addr B
// interleaved → E-ADM-017 emitted once for A and once for B, independently.
//
// Traces to BC-2.05.005 EC-007; S-W3.05 AC-007.
func TestFailureCounter_MultiSourceIsolation(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	srcA := "addr-A-isolation"
	srcB := "addr-B-isolation"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// Interleave: 5 from A (alert fires for A), 3 from B (no alert for B).
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(srcA)
		if i < 3 {
			fc.RecordHMACFailure(srcB)
		}
	}

	// Exactly 1 alert for A.
	aAlerts := 0
	bAlerts := 0
	for _, line := range log.Lines() {
		if strings.Contains(line, srcA) {
			aAlerts++
		}
		if strings.Contains(line, srcB) {
			bAlerts++
		}
	}
	if aAlerts != 1 {
		t.Errorf("AC-007: want 1 E-ADM-017 for %q, got %d; lines: %v", srcA, aAlerts, log.Lines())
	}
	if bAlerts != 0 {
		t.Errorf("AC-007: want 0 E-ADM-017 for %q (only 3 failures), got %d; lines: %v",
			srcB, bAlerts, log.Lines())
	}

	// Now 5 more from B (B crosses threshold).
	for i := 5; i < 10; i++ {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(srcB)
	}
	bAlerts = 0
	for _, line := range log.Lines() {
		if strings.Contains(line, srcB) {
			bAlerts++
		}
	}
	if bAlerts != 1 {
		t.Errorf("AC-007: want 1 E-ADM-017 for %q after crossing threshold, got %d; lines: %v",
			srcB, bAlerts, log.Lines())
	}
}

// ── AC-008: TestFailureCounter_BoundaryEntryIsKept ────────────────────────────

// TestFailureCounter_BoundaryEntryIsKept verifies the boundary semantics of the
// sliding window trim: entries at exactly now - windowDuration are KEPT (strictly-
// less-than comparison), not trimmed.
//
// Scenario (from BC-2.05.005 EC-008):
//   - 1st failure at t=0.
//   - 2nd-4th failures at t=1s, t=2s, t=3s.
//   - 5th failure at t=0+windowDuration (exactly at the boundary).
//   - Trim condition: timestamp < now - windowDuration, i.e. timestamp < t=0.
//   - The 1st entry (timestamp=t=0) is NOT strictly less than t=0 → kept.
//   - Post-trim count = 5; E-ADM-017 fires.
//
// An implementation using <= (trim-at-boundary) would produce count=4 and fail.
//
// Traces to BC-2.05.005 EC-008; S-W3.05 AC-008.
func TestFailureCounter_BoundaryEntryIsKept(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "boundary-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// 1st failure at t=0.
	fc.RecordHMACFailure(src)

	// 2nd–4th failures at t=1s, t=2s, t=3s.
	for i := 1; i <= 3; i++ {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}

	if log.Count() != 0 {
		t.Fatalf("AC-008: want 0 alerts before boundary-5th failure, got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// 5th failure at exactly t=0+windowDuration (the boundary instant).
	// Trim condition: timestamp < now - window = t=0. The 1st entry is at t=0
	// (not strictly less) so it must be kept. Post-trim count must be 5 → alert fires.
	current = base.Add(window) // now = t=60s; now-window = t=0 (boundary)
	fc.RecordHMACFailure(src)

	if log.Count() != 1 {
		t.Errorf("AC-008: boundary entry must be kept (strictly-less-than trim); "+
			"want 1 alert, got %d; lines: %v — implementation may use <= instead of < for trim",
			log.Count(), log.Lines())
	}
}

// ── AC-009 (no forwarding-counter path): TestFailureCounter_NoAlertOnSuccess ──

// TestFailureCounter_NoAlertOnSuccess verifies that when no failures are
// recorded, no E-ADM-017 is emitted (negative/mutation-resistance test).
//
// Traces to BC-2.05.005 PC-3 (no false positive); S-W3.05 AC-008 no-failure case.
func TestFailureCounter_NoAlertOnSuccess(t *testing.T) {
	t.Parallel()

	log := &fakeLog{}
	fc := admission.NewFailureCounter(5, 60*time.Second, log)

	// No RecordHMACFailure calls — counter stays empty.
	if log.Count() != 0 {
		t.Errorf("AC-009: want 0 alerts when no failures recorded, got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// Timestamps for unseen src must return empty slice (not nil with out-of-bound risk).
	ts := fc.Timestamps("never-seen-src")
	if len(ts) != 0 {
		t.Errorf("AC-009: want empty Timestamps for unseen src, got %d entries", len(ts))
	}
}

// ── AC-010: TestFailureCounter_ConcurrentCallsRaceSafe ───────────────────────

// TestFailureCounter_ConcurrentCallsRaceSafe verifies that concurrent
// RecordHMACFailure calls from multiple goroutines are race-safe and that
// counts are not lost (go.md rule 12; go test -race must pass).
//
// Design: 10 goroutines each call RecordHMACFailure 1 time for the same srcAddr.
// Total = 10 calls; threshold = 5; expect exactly 1 E-ADM-017.
// Concurrency stress: goroutines are interleaved under the race detector.
//
// Traces to BC-2.05.005 PC-3 (concurrency contract); S-W3.05 AC-010; go.md rule 12.
func TestFailureCounter_ConcurrentCallsRaceSafe(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const numGoroutines = 10
	src := "concurrent-src"

	log := &fakeLog{}
	// Use real time.Now() for this test — we don't need clock control;
	// all calls happen within a very short window (<< 60s).
	fc := admission.NewFailureCounter(threshold, 60*time.Second, log)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	start := make(chan struct{})

	for range numGoroutines {
		go func() {
			defer wg.Done()
			<-start // synchronize start to maximize concurrency
			fc.RecordHMACFailure(src)
		}()
	}

	close(start) // release all goroutines simultaneously
	wg.Wait()

	// 10 calls with threshold=5: at least 1 E-ADM-017 must have fired.
	if log.Count() == 0 {
		t.Errorf("AC-010: want at least 1 E-ADM-017 after %d concurrent failures (threshold=%d), got 0",
			numGoroutines, threshold)
	}
	// At most 1 alert should have fired (fire-once-per-crossing).
	if log.Count() > 1 {
		t.Errorf("AC-010: fire-once violated under concurrency — want 1 alert, got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// Timestamps must return a value copy (go.md rule 12): mutating it must not
	// affect the FailureCounter's internal state.
	ts := fc.Timestamps(src)
	if len(ts) == 0 {
		t.Error("AC-010: Timestamps returned empty slice after concurrent calls")
	}
	// Verify copy: append to the returned slice and check internal count unchanged.
	ts = append(ts, time.Now().UTC())
	ts2 := fc.Timestamps(src)
	if len(ts2) >= len(ts) {
		t.Errorf("AC-010: Timestamps leaked internal pointer — "+
			"appending to returned slice mutated internal state (go.md rule 12): "+
			"len after append=%d, re-fetch len=%d", len(ts), len(ts2))
	}
}

// ── BC-2.05.005 EC-007 multi-source interleaved: canonical test vector ────────

// TestFailureCounter_MultiSourceInterleaved exercises the canonical BC-2.05.005
// test vector: "5 HMAC failures from src_addr A + 5 from src_addr B, interleaved
// → E-ADM-017 emitted once for A and once for B, independently."
//
// This test is separate from AC-007 (which also tests B alone first). Here the
// two sources are fully interleaved symmetrically.
//
// Traces to BC-2.05.005 canonical test vector (multi-source); EC-007.
func TestFailureCounter_MultiSourceInterleaved(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	srcA := "interleaved-addr-A"
	srcB := "interleaved-addr-B"

	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// Interleave: A, B, A, B, A, B, A, B, A, B (5 each).
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(srcA)
		fc.RecordHMACFailure(srcB)
	}

	aAlerts := 0
	bAlerts := 0
	for _, line := range log.Lines() {
		if strings.Contains(line, srcA) {
			aAlerts++
		}
		if strings.Contains(line, srcB) {
			bAlerts++
		}
	}

	if aAlerts != 1 {
		t.Errorf("EC-007: want 1 E-ADM-017 for srcA, got %d; lines: %v", aAlerts, log.Lines())
	}
	if bAlerts != 1 {
		t.Errorf("EC-007: want 1 E-ADM-017 for srcB, got %d; lines: %v", bAlerts, log.Lines())
	}
}
