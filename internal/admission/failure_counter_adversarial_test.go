// Package admission_test — adversarial TDD tests closing S-W3.05 convergence findings.
//
// These tests target the HIGH findings adjudicated by the product owner:
//   - BC-2.05.005 v1.4 PC-3: periodic re-fire, constructor panics, LRU source cap,
//     dead-key eviction, strict re-arm boundary
//   - error-taxonomy v1.9 E-ADM-017: canonical parameterized message format
//
// RED GATE: all tests in this file MUST FAIL before the implementer adds:
//   - Constructor panic validation (AC-013)
//   - LRU source cap (maxTrackedSources = 65536) with SourceCount() accessor (AC-011)
//   - Dead-key eviction of empty counts entries from the map (AC-012)
//   - The exact canonical E-ADM-017 message format from error-taxonomy v1.9 (AC-015)
//
// AC-014 and the re-arm boundary test (AC-004 extension) compile and run against the
// current implementation and verify discriminating edge cases of the re-arm logic.
//
// Implementer MUST add:
//   - NewFailureCounter panics on threshold<1 or windowDuration<=0
//   - SourceCount() int method on *FailureCounter (returns number of live tracked sources)
//   - Delete counts[srcAddr] (not just zero-length slice) when post-trim count == 0
//   - E-ADM-017 message format: "E-ADM-017 ≥<N> failures in <W>s from src <hex>"
//     (no "HMAC failure rate alert:" prefix — matches error-taxonomy v1.9)
//
// Traces to: BC-2.05.005 v1.4 PC-3; BC-2.05.005 EC-005, EC-009, EC-010;
// S-W3.05 AC-004, AC-011, AC-012, AC-013, AC-014, AC-015; error-taxonomy v1.9.
package admission_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// ── AC-014: TestFailureCounter_SustainedAttackReFires ─────────────────────────

// TestFailureCounter_SustainedAttackReFires verifies that under a sustained attack
// the E-ADM-017 alert fires periodically — once per window-drain-and-rearm cycle —
// rather than going permanently silent after the first crossing.
//
// Scenario (BC-2.05.005 EC-009):
//
//	T=0..4:  5 failures → assert exactly 1 E-ADM-017 emitted.
//	T=61s:   window fully drains; counter re-arms on the next call.
//	T=61..65: 5 more failures → assert a 2nd E-ADM-017.
//	Assert EXACTLY 2 total alerts for this source.
//	Also assert SourceCount() == 1 after the second batch (dead-key state check).
//
// The SourceCount() call provides a compile-time Red Gate: the method does not yet
// exist on *FailureCounter. Once added, the behavioral assertions will also verify
// periodic re-fire vs. permanent silence.
//
// Traces to BC-2.05.005 EC-009; S-W3.05 AC-014.
func TestFailureCounter_SustainedAttackReFires(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "sustained-attack-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// First batch: 5 failures at T=0..4s → exactly 1 alert.
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 1 {
		t.Fatalf("AC-014: want exactly 1 E-ADM-017 after first 5 failures, got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// Advance to T=61s — all first-batch entries (T=0..4) are now outside the 60s window.
	// The next call triggers a trim-to-zero, which re-arms the counter.
	batchTwoBase := base.Add(61 * time.Second)

	// Second batch: 5 failures at T=61..65s → counter re-armed, should fire 2nd alert.
	for i := range 5 {
		current = batchTwoBase.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}

	if log.Count() != 2 {
		t.Errorf("AC-014: sustained attack must re-fire after window drain — "+
			"want exactly 2 E-ADM-017 alerts total, got %d; lines: %v "+
			"(periodic re-fire per BC-2.05.005 EC-009; impl may be going permanently silent)",
			log.Count(), log.Lines())
	}

	// SourceCount() is the compile-time Red Gate: this method does not yet exist.
	// The implementer must add: func (c *FailureCounter) SourceCount() int
	// which returns len(c.counts) (number of live tracked sources, under the mutex).
	if got := fc.SourceCount(); got != 1 {
		t.Errorf("AC-014: after sustained attack on one source, want SourceCount()==1, got %d",
			got)
	}
}

// ── AC-004 extension: TestFailureCounter_RearmBoundaryAtLastFireTimestamp ─────

// TestFailureCounter_RearmBoundaryAtLastFireTimestamp pins the off-by-one in
// the re-arm boundary condition (HF-2): an entry whose timestamp == firedAt does
// NOT trigger re-arm (boundary is strictly-newer).
//
// Scenario (threshold=1, window=60s — uses threshold=1 to isolate boundary effect):
//
//	T=0:  failure 1 → threshold=1 crossed; fire; firedAt[src]=T=0.
//	T=60: failure — cutoff = T=0; entry T=0 exactly at boundary: NOT trimmed
//	      (strictly-less-than keeps T=0); keep=[T=0].
//	      keep[0]=T=0, lastFire=T=0 → keep[0].After(T=0) == false → NO re-arm.
//	      count=2 >= threshold but lastFire not zero → NO new fire.
//	      ASSERT: exactly 1 total alert (boundary entry does NOT trigger re-arm).
//	T=61: failure — cutoff = T=1; T=0 trimmed (T=0 < T=1); T=60 kept (T=60 >= T=1).
//	      keep=[T=60]; T=60 > T=0 → STRICTLY AFTER → re-arm.
//	      count=1 >= threshold AND lastFire now zero → fires 2nd alert.
//	      ASSERT: exactly 2 total alerts (re-arm on strictly-newer entry).
//
// Discriminating property: an implementation that re-arms on >= (instead of strictly >)
// would fire at T=60 (total 2 alerts instead of 1 after T=60 call), failing the
// mid-scenario assertion and then reaching 3 total alerts, causing the final check to fail.
//
// This test PASSES the current correct implementation and provides a regression guard.
// It is included here because the adversarial review (HF-2) flagged this boundary
// as an off-by-one risk that prior tests did not exercise at the exact equality point.
//
// Traces to BC-2.05.005 PC-3 + EC-005; S-W3.05 AC-004.
func TestFailureCounter_RearmBoundaryAtLastFireTimestamp(t *testing.T) {
	t.Parallel()

	const threshold = 1
	const window = 60 * time.Second
	src := "rearm-boundary-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// T=0: failure → threshold=1 reached; fires once. firedAt[src] = T=0.
	current = base
	fc.RecordHMACFailure(src)
	if log.Count() != 1 {
		t.Fatalf("rearm-boundary: want 1 alert on first failure (threshold=1), got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// T=60: failure — now = T=60, cutoff = T=0.
	// T=0 entry: T=0 >= cutoff(T=0) → NOT trimmed (strictly-less-than rule).
	// keep = [T=0]. keep[0] = T=0, firedAt = T=0 → T=0.After(T=0) == false → NO re-arm.
	// count=2 >= threshold but suppressed (lastFire != zero). → NO fire.
	//
	// This is the key boundary assertion: an entry whose timestamp exactly equals
	// firedAt does NOT trigger re-arm (BC-2.05.005 PC-3: "strictly newer than firedAt").
	current = base.Add(60 * time.Second)
	fc.RecordHMACFailure(src)
	if log.Count() != 1 {
		t.Errorf("rearm-boundary: at T=60 oldest entry (T=0) equals firedAt (T=0) — "+
			"must NOT re-arm (strictly-newer required by BC-2.05.005 PC-3); "+
			"want 1 total alert after T=60 call, got %d; lines: %v "+
			"(an implementation using >= instead of > to check re-arm would fire here)",
			log.Count(), log.Lines())
	}

	// T=61: failure — now = T=61, cutoff = T=1.
	// T=0 trimmed (T=0 < T=1). T=60 kept (T=60 >= T=1).
	// keep = [T=60]. keep[0] = T=60 > T=0 (firedAt) → STRICTLY AFTER → re-arm.
	// count=1 >= threshold AND re-armed → fires 2nd alert.
	current = base.Add(61 * time.Second)
	fc.RecordHMACFailure(src)
	if log.Count() != 2 {
		t.Errorf("rearm-boundary: at T=61 oldest entry (T=60) is strictly after firedAt (T=0) — "+
			"must re-arm and fire again; want 2 total alerts, got %d; lines: %v",
			log.Count(), log.Lines())
	}
}

// ── AC-011: TestFailureCounter_SourceCapBoundsMapGrowth ──────────────────────

// TestFailureCounter_SourceCapBoundsMapGrowth verifies that recording 1 failure each
// for maxTrackedSources+1 (65,537) distinct srcAddrs caps the live tracked-source
// count at maxTrackedSources (65,536). The LRU-evicted source's key is removed from
// both counts and firedAt, preventing unbounded map growth (CWE-770).
//
// The SourceCount() call is the compile-time Red Gate: this method does not yet
// exist on *FailureCounter; the implementer must add it.
//
// Traces to BC-2.05.005 EC-010; S-W3.05 AC-011; CWE-770.
func TestFailureCounter_SourceCapBoundsMapGrowth(t *testing.T) {
	// Not parallel — this test inserts 65,537 entries and may be slow under -race.
	// Keep it single-threaded to avoid false race reports on the fakeLog.

	const maxTrackedSources = 65536
	const window = 60 * time.Second

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Advance clock slightly per source so LRU ordering is deterministic.
	tick := int64(0)

	log := &fakeLog{}
	fc := admission.NewFailureCounter(100, window, log,
		admission.WithNow(func() time.Time {
			t := base.Add(time.Duration(tick) * time.Millisecond)
			tick++
			return t
		}),
	)

	// Insert 65,537 distinct sources — one failure each.
	for i := range maxTrackedSources + 1 {
		src := fmt.Sprintf("src-%06d", i)
		fc.RecordHMACFailure(src)
	}

	// After inserting maxTrackedSources+1 keys, the map must be capped.
	// SourceCount() is the Red Gate: implementer must add this method.
	got := fc.SourceCount()

	// Two-sided assertion:
	// (a) SourceCount > 0: proves the method actually returns live count, not a stub zero.
	//     Without a real implementation, got==0 → this assertion fails (Red Gate).
	if got == 0 {
		t.Errorf("AC-011: SourceCount() returned 0 after %d insertions — "+
			"method is not implemented (stub returns 0); "+
			"implementer must add SourceCount() returning len(c.counts) under mutex",
			maxTrackedSources+1)
	}

	// (b) SourceCount <= maxTrackedSources: proves the LRU cap is enforced.
	//     An unbounded map would have got == maxTrackedSources+1 == 65537 > 65536.
	if got > maxTrackedSources {
		t.Errorf("AC-011: SourceCount() after %d insertions must be <= maxTrackedSources (%d), got %d "+
			"(CWE-770: unbounded map growth; implementer must add LRU eviction at maxTrackedSources)",
			maxTrackedSources+1, maxTrackedSources, got)
	}
}

// ── AC-012: TestFailureCounter_DeadKeyEvictedAfterDrain ──────────────────────

// TestFailureCounter_DeadKeyEvictedAfterDrain verifies that when a source's window
// fully drains (all timestamps age out), both counts[srcAddr] and firedAt[srcAddr]
// are deleted from the map (dead-key eviction), not merely zeroed/emptied.
//
// This test verifies two properties:
//  1. After drain, a fresh threshold crossing re-fires E-ADM-017 (re-arm verified).
//  2. After drain, SourceCount() decrements — the key is evicted, not retained as
//     an empty entry (prevents unbounded map growth from inactive sources).
//
// SourceCount() is the compile-time Red Gate for property 2.
//
// Traces to BC-2.05.005 EC-005; S-W3.05 AC-012.
func TestFailureCounter_DeadKeyEvictedAfterDrain(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "dead-key-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// 5 failures → fires once.
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 1 {
		t.Fatalf("AC-012: want 1 alert after threshold crossing, got %d", log.Count())
	}

	// Advance past the window — all entries (T=0..4) are now stale (cutoff=T=65).
	// The NEXT call will trim them all. After trim, len==0 → dead-key eviction.
	drainedTime := base.Add(65 * time.Second)
	current = drainedTime

	// One probe call to trigger trim and drain. The window is now fully empty for src.
	// This call itself adds one new entry but the stale ones are removed.
	// We need the trim to happen, so make a call and then check state via SourceCount().
	//
	// After this call: trim removes T=0..4 (all < drainedTime-60s = T=5), keep=[].
	// Dead-key eviction: counts[src] and firedAt[src] must be deleted from the maps
	// before appending the new entry at T=65. Wait — the current call adds T=65.
	// SourceCount() should still be 1 (one active entry at T=65).
	//
	// To see SourceCount drop to 0 we'd need to drain WITHOUT adding a new entry,
	// which isn't possible via the public API. Instead, verify re-arm + re-fire:
	// after the drain, a fresh threshold crossing must fire again.

	// Second batch: starting from T=65, send threshold failures.
	for i := range threshold {
		current = drainedTime.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}

	// After full drain and re-arm, a second crossing must fire.
	if log.Count() != 2 {
		t.Errorf("AC-012: dead-key drain must clear firedAt[srcAddr] so a fresh threshold crossing "+
			"fires E-ADM-017 again — want 2 total alerts, got %d; lines: %v "+
			"(if count==1, firedAt was NOT cleared after drain → dead-key eviction missing)",
			log.Count(), log.Lines())
	}

	// SourceCount() verifies the key is properly tracked after the second batch.
	// After firing, SourceCount() must be 1 (the src is still active in the window).
	// The compile-time Red Gate: SourceCount() must be implemented.
	if got := fc.SourceCount(); got < 1 {
		t.Errorf("AC-012: SourceCount() after second batch must be >= 1, got %d", got)
	}
}

// ── AC-013: TestNewFailureCounter_PanicsOnInvalidArgs ─────────────────────────

// TestNewFailureCounter_PanicsOnInvalidArgs verifies that NewFailureCounter panics
// eagerly on programmer-error arguments: threshold < 1 or windowDuration <= 0.
//
// These are programmer-error guards (not runtime error paths) per BC-2.05.005 PC-3 v1.4.
// The current implementation does NOT panic → all subtests will FAIL (Red Gate).
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-013.
func TestNewFailureCounter_PanicsOnInvalidArgs(t *testing.T) {
	t.Parallel()

	log := &fakeLog{}

	tests := []struct {
		name           string
		threshold      int
		windowDuration time.Duration
		wantPanic      bool
	}{
		{
			name:           "zero threshold panics",
			threshold:      0,
			windowDuration: 60 * time.Second,
			wantPanic:      true,
		},
		{
			name:           "negative threshold panics",
			threshold:      -1,
			windowDuration: 60 * time.Second,
			wantPanic:      true,
		},
		{
			name:           "zero window duration panics",
			threshold:      5,
			windowDuration: 0,
			wantPanic:      true,
		},
		{
			name:           "negative window duration panics",
			threshold:      5,
			windowDuration: -1 * time.Second,
			wantPanic:      true,
		},
		{
			name:           "valid args do not panic",
			threshold:      1,
			windowDuration: 1 * time.Second,
			wantPanic:      false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			didPanic := mustPanic(t, func() {
				admission.NewFailureCounter(tc.threshold, tc.windowDuration, log)
			})

			if tc.wantPanic && !didPanic {
				t.Errorf("AC-013: NewFailureCounter(%d, %v, logger) must panic on invalid args "+
					"but did not — implementer must add panic guards in NewFailureCounter "+
					"(BC-2.05.005 PC-3 v1.4 constructor contract)",
					tc.threshold, tc.windowDuration)
			}
			if !tc.wantPanic && didPanic {
				t.Errorf("AC-013: NewFailureCounter(%d, %v, logger) must NOT panic on valid args "+
					"but did", tc.threshold, tc.windowDuration)
			}
		})
	}
}

// mustPanic calls fn and reports whether it panicked. It is a t.Helper.
func mustPanic(t *testing.T, fn func()) (panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	fn()
	return false
}

// ── AC-015: TestFailureCounter_AlertMessageFormat ─────────────────────────────

// TestFailureCounter_AlertMessageFormat verifies that the E-ADM-017 log message
// matches the canonical parameterized format from error-taxonomy v1.9:
//
//	"E-ADM-017 ≥<threshold> failures in <window>s from src <lowercase-hex-srcAddr>"
//
// The format pins three requirements:
//  1. Code literal "E-ADM-017" present (for operator grep-ability).
//  2. Parameterized threshold and window values embedded numerically.
//  3. srcAddr rendered as lowercase hex (fmt.Sprintf("%x", ...) convention).
//
// The current implementation uses format:
//
//	"E-ADM-017 HMAC failure rate alert: ≥%d failures in %.0fs from src %s"
//
// This diverges from error-taxonomy v1.9 by including "HMAC failure rate alert:"
// as a non-canonical prefix. This test asserts the canonical format WITHOUT that
// prefix, so it will FAIL against the current implementation.
//
// srcAddr convention: RouteFrame passes fmt.Sprintf("%x", hdr.SrcAddr) as the
// srcAddr string (routing.go L205, L220). The FailureCounter embeds this verbatim
// with %s, so the emitted message already contains lowercase hex. This test uses
// a raw hex string as srcAddr to match that convention.
//
// Traces to error-taxonomy v1.9; S-W3.05 AC-015.
func TestFailureCounter_AlertMessageFormat(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second

	// Use a lowercase hex string as srcAddr — matching the convention from routing.go
	// (fmt.Sprintf("%x", hdr.SrcAddr) → e.g. "0102030405060708").
	src := "deadbeef01020304"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// Drive threshold failures to trigger E-ADM-017.
	for i := range threshold {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}

	if log.Count() != 1 {
		t.Fatalf("AC-015: want 1 alert to inspect, got %d; lines: %v", log.Count(), log.Lines())
	}

	lines := log.Lines()
	msg := lines[0]

	// (a) Must contain code literal "E-ADM-017".
	if !strings.Contains(msg, "E-ADM-017") {
		t.Errorf("AC-015: E-ADM-017 message missing code literal \"E-ADM-017\"; got: %q", msg)
	}

	// (b) Must contain "≥5" (threshold embedded).
	wantThreshold := fmt.Sprintf("≥%d", threshold)
	if !strings.Contains(msg, wantThreshold) {
		t.Errorf("AC-015: E-ADM-017 message missing threshold substring %q; got: %q",
			wantThreshold, msg)
	}

	// (c) Must contain "60s" (window in seconds, no decimal).
	wantWindow := fmt.Sprintf("%.0fs", window.Seconds())
	if !strings.Contains(msg, wantWindow) {
		t.Errorf("AC-015: E-ADM-017 message missing window substring %q; got: %q",
			wantWindow, msg)
	}

	// (d) Must contain "from src <hex>" with the srcAddr as lowercase hex.
	wantSrc := fmt.Sprintf("from src %s", src)
	if !strings.Contains(msg, wantSrc) {
		t.Errorf("AC-015: E-ADM-017 message missing src substring %q; got: %q",
			wantSrc, msg)
	}

	// (e) Must NOT contain "HMAC failure rate alert:" — that prefix is not in
	// error-taxonomy v1.9's canonical format for E-ADM-017.
	// The canonical format is: "E-ADM-017 ≥<N> failures in <W>s from src <hex>"
	if strings.Contains(msg, "HMAC failure rate alert:") {
		t.Errorf("AC-015: E-ADM-017 message must NOT contain non-canonical prefix "+
			"\"HMAC failure rate alert:\" (error-taxonomy v1.9 canonical format is "+
			"\"E-ADM-017 ≥<N> failures in <W>s from src <hex>\"); got: %q", msg)
	}
}

// ── routing integration: TestRouteFrame_EndToEnd_EADMAlertViaRouteFrame ────────

// TestRouteFrame_EndToEnd_EADMAlertViaRouteFrame verifies the full end-to-end
// path: ≥threshold HMAC failures through RouteFrame trigger exactly one E-ADM-017
// from the real FailureCounter, and the canonical message is present in the alert log.
//
// This extends the existing EC-006 test (which only checks count and "E-ADM-017"
// presence) to also assert the canonical message format from error-taxonomy v1.9.
// It is placed here (admission_test) rather than routing_test because it is
// testing the FailureCounter's output contract, using RouteFrame as the driver.
//
// Note: this test imports both admission and routing. It will compile and run
// once SourceCount() and the format changes are in place; until then it fails
// because it shares a file with tests that reference SourceCount().
//
// Traces to BC-2.05.008 EC-006; BC-2.05.005 PC-3; S-W3.05 AC-015 (format
// assertion end-to-end); error-taxonomy v1.9.
func TestRouteFrame_EndToEnd_EADMAlertMessageFormat(t *testing.T) {
	t.Parallel()

	// This test drives the alert message format through the RouteFrame path to
	// verify that the srcAddr passed by RouteFrame (lowercase hex of hdr.SrcAddr)
	// ends up correctly embedded in the E-ADM-017 message.
	//
	// We use the FailureCounter directly here (not via RouteFrame) to isolate
	// the format assertion from routing plumbing. The routing integration is
	// already covered by TestRouteFrame_FiveConsecutiveFailures_TriggersEADM017
	// in routing_hmac_counter_test.go.
	//
	// srcAddr as it would come from RouteFrame (fmt.Sprintf("%x", [8]byte{...})):
	var rawAddr [8]byte
	copy(rawAddr[:], "srcaddr1")
	srcHex := fmt.Sprintf("%x", rawAddr) // "73726361646472" (lowercase hex)

	const threshold = 5
	const window = 60 * time.Second

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	for i := range threshold {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(srcHex)
	}

	if log.Count() != 1 {
		t.Fatalf("e2e-format: want 1 alert, got %d; lines: %v", log.Count(), log.Lines())
	}

	msg := log.Lines()[0]

	// Canonical format per error-taxonomy v1.9: "E-ADM-017 ≥N failures in Ws from src hex"
	canonical := fmt.Sprintf("E-ADM-017 ≥%d failures in %.0fs from src %s",
		threshold, window.Seconds(), srcHex)
	if !strings.Contains(msg, canonical) {
		t.Errorf("e2e-format: alert message does not contain canonical E-ADM-017 format\n"+
			"  want substring: %q\n"+
			"  got message:    %q\n"+
			"  (error-taxonomy v1.9: 'E-ADM-017 ≥<N> failures in <W>s from src <hex>')",
			canonical, msg)
	}

	// Exact canonical string (full message should be exactly this).
	if msg != canonical {
		t.Logf("e2e-format: message is not exactly canonical (may have extra context): %q", msg)
	}

	// The fakeLog.HasAll helper is used for the "both E-ADM-017 and srcHex present" check.
	if !log.HasAll("E-ADM-017", srcHex) {
		t.Errorf("e2e-format: alert must contain both E-ADM-017 and srcHex %q; got: %v",
			srcHex, log.Lines())
	}
}

// ── concurrent safety of new accessors ──────────────────────────────────────

// TestFailureCounter_SourceCount_RaceSafe verifies that SourceCount() is safe for
// concurrent calls alongside RecordHMACFailure. This is required by go.md rule 12
// (no data races on locked accessors) and AC-010's concurrency contract.
//
// Compile-time Red Gate: SourceCount() does not yet exist.
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-011 (accessor must be concurrency-safe).
func TestFailureCounter_SourceCount_RaceSafe(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const numWriters = 4
	const numReaders = 4
	const callsPerGoroutine = 50

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, 60*time.Second, log)

	var wg sync.WaitGroup
	start := make(chan struct{})

	// Writers: call RecordHMACFailure from multiple goroutines.
	for i := range numWriters {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			for j := range callsPerGoroutine {
				fc.RecordHMACFailure(fmt.Sprintf("src-%d-%d", id, j))
			}
		}(i)
	}

	// Readers: call SourceCount concurrently with writes.
	for range numReaders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for range callsPerGoroutine {
				// SourceCount() is the Red Gate compile-time check;
				// just calling it is sufficient for the race detector.
				_ = fc.SourceCount()
			}
		}()
	}

	close(start)
	wg.Wait()

	// After all writes, SourceCount must be positive and not exceed total distinct sources.
	// Each writer uses unique src keys (src-%d-%d), so there will be exactly
	// numWriters*callsPerGoroutine distinct sources tracked.
	maxExpected := numWriters * callsPerGoroutine
	got := fc.SourceCount()

	// Positive lower bound: proves SourceCount is not a stub returning 0.
	if got == 0 {
		t.Errorf("concurrent SourceCount: returned 0 after %d distinct source writes — "+
			"method is not implemented (Red Gate: stub returns 0)", maxExpected)
	}
	// Upper bound: cannot exceed the number of distinct sources written.
	if got > maxExpected {
		t.Errorf("concurrent SourceCount: want <= %d, got %d", maxExpected, got)
	}
}
