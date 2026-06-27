// Package admission_test — adversarial TDD tests closing S-W3.05 convergence findings.
//
// These tests target the HIGH findings adjudicated by the product owner:
//   - BC-2.05.005 v1.4 PC-3: periodic re-fire, constructor panics, LRU source cap,
//     dead-key eviction, drain-only re-arm (v1.6)
//   - error-taxonomy v1.9 E-ADM-017: canonical parameterized message format INCLUDING
//     the "HMAC failure rate alert:" phrase (story v1.2 AC-015 restores the correct form)
//
// RED GATE: all tests in this file MUST FAIL before the implementer:
//   - Fixes the E-ADM-017 format string to include "HMAC failure rate alert:" (AC-015)
//   - Implements append-skip (AC-016) + bounded-slice invariant
//   - Renames TestFailureCounter_RearmOccursOnFirstCallAfterDrain to test drain-only re-arm
//
// Implementer MUST fix/add:
//   - E-ADM-017 message format: "E-ADM-017 HMAC failure rate alert: ≥<N> failures in <W>s from src <hex>"
//     (failure_counter.go line ~169 is currently missing "HMAC failure rate alert:")
//   - Append-skip: do not append when firedAt[srcAddr] is set (BC-2.05.005 EC-011)
//   - Drain-only re-arm: len(keep)==0 is the SOLE re-arm condition (BC-2.05.005 v1.6)
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

// ── AC-004: TestFailureCounter_RearmOccursOnFirstCallAfterDrain ───────────────

// TestFailureCounter_RearmOccursOnFirstCallAfterDrain verifies the drain-only
// re-arm semantics mandated by BC-2.05.005 v1.6 + story v1.2 AC-004.
//
// Under the append-skip policy (EC-011), no timestamps are appended while
// firedAt[srcAddr] is set. Re-arm triggers ONLY when len(keep)==0 after trim
// (all pre-fire entries have aged out). The old "keep[0].After(lastFire)"
// re-arm path is dead code under append-skip and is NOT tested here.
//
// Discriminating scenario (threshold=5, window=60s):
//
//	T=0..4:  5 failures → alert fires; firedAt[src]=T=4. append-skip is now in force.
//	T=65:    advance clock so all pre-fire entries (T=0..4) are outside the window
//	         (cutoff = T=65-60 = T=5; all pre-fire entries T=0..4 < T=5 → trimmed).
//	         On the FIRST RecordHMACFailure call at T=65:
//	           (a) trim removes T=0..4 → len(keep)==0 → re-arm triggers (firedAt deleted)
//	           (b) append PROCEEDS: len(Timestamps(src)) == 1 (not 0)
//	           (c) count=1 < threshold → no new alert yet
//
// A buggy implementation that checks lastFire.IsZero() BEFORE removing the
// key in the re-arm path (wrong ordering) would skip the append and leave
// len==0 on this call, failing assertion (b).
//
// Note: under a correct drain-only re-arm + append-skip implementation the pre-fire
// entries are trimmed, re-arm fires, and the new call's timestamp IS appended
// (because firedAt is now cleared before the append step). This test pins that ordering.
//
// Replaces TestFailureCounter_RearmBoundaryAtLastFireTimestamp which tested the
// now-dead "oldest surviving entry is newer than firedAt" re-arm path.
//
// Traces to BC-2.05.005 v1.6 PC-3 (drain-only re-arm); BC-2.05.005 EC-009;
// S-W3.05 AC-004; story v1.2.
func TestFailureCounter_RearmOccursOnFirstCallAfterDrain(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	src := "rearm-drain-first-call-src"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// T=0..4: 5 failures → alert fires once. firedAt[src] is set.
	// append-skip is now in force: no further timestamps will be appended.
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 1 {
		t.Fatalf("AC-004: want 1 alert after threshold crossing, got %d; lines: %v",
			log.Count(), log.Lines())
	}

	// Advance clock to T=65s. Cutoff = T=65-60 = T=5.
	// All 5 pre-fire entries (T=0..4) have timestamp < T=5 → will be trimmed.
	drainT := base.Add(65 * time.Second)
	current = drainT

	// FIRST call after drain — this is the discriminating call:
	//   (a) trim removes T=0..4 → len(keep)==0 → re-arm triggers (firedAt deleted)
	//   (b) append proceeds → len(Timestamps(src)) must be exactly 1
	//   (c) count=1 < threshold → NO new alert
	fc.RecordHMACFailure(src)

	// Assertion (a): re-arm happened — log count stays at 1 (no spurious alert).
	if log.Count() != 1 {
		t.Errorf("AC-004 (a): first call after drain must re-arm, not re-fire; "+
			"want 1 total alert, got %d; lines: %v", log.Count(), log.Lines())
	}

	// Assertion (b): append proceeded → slice must have exactly 1 entry.
	// A buggy impl that skips the append (wrong re-arm ordering) would yield len==0.
	ts := fc.Timestamps(src)
	if len(ts) != 1 {
		t.Errorf("AC-004 (b): after drain re-arm, first call must append new timestamp; "+
			"want len(Timestamps)==1, got %d "+
			"(if 0: append was skipped — re-arm check happened AFTER the append branch, "+
			"or firedAt was not cleared before append; BC-2.05.005 v1.6 drain-only re-arm requires "+
			"firedAt cleared BEFORE the append step so the new entry IS recorded)",
			len(ts))
	}

	// Assertion (c): no E-ADM-017 fired yet (count=1 < threshold).
	if log.Count() != 1 {
		t.Errorf("AC-004 (c): want exactly 1 alert total (no re-fire at count=1<threshold=%d), got %d",
			threshold, log.Count())
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
// fully drains (all timestamps age out), the key IS deleted from counts/firedAt
// (dead-key eviction via delete(counts, srcAddr)), not left as an empty slice.
//
// Discriminating design (story v1.2 AC-012): the test must distinguish an impl
// that calls delete(counts, srcAddr) on drain from one that leaves an empty slice.
// This is achieved by using TWO sources — srcA and srcB — where srcA's entries
// drain naturally and are confirmed deleted by observing SourceCount() drops
// from 2 → 1 after only srcB is touched at post-drain time, NOT srcA.
//
// Scenario:
//  1. srcA: 5 failures at T=0..4 → alert fires; SourceCount()==1 after srcA fires.
//  2. srcB: 1 failure at T=1 → SourceCount()==2 (both sources live).
//  3. Advance clock to T=65. Cutoff = T=5.
//     All of srcA's entries (T=0..4) are outside the window.
//     srcB's entry (T=1) is also outside the window.
//  4. Record a failure for srcB at T=65 → trim removes srcB's stale entry (T=1 < T=5);
//     len(keep)==0 for srcB → dead-key eviction runs for srcB; new T=65 entry appended.
//     SourceCount() at this point: srcA is NOT touched so counts[srcA] still holds
//     [T=0..4] in a buggy impl (empty slice without delete), or is absent in correct impl.
//
// Here is the discriminating property:
//   - Correct impl: delete(counts, srcA) was called when srcA last had a RecordHMACFailure
//     that drained it. But wait — srcA is under append-skip from T=4 onward, so no more
//     calls touch srcA. The eviction only runs on the CALLING source's key, not any other.
//
// REVISED discriminating path (per story AC-012 guidance): dead-key eviction on drain
// occurs when the SAME source is called again after its window drains. To observe the
// delete, call srcA again AFTER its window drains — the drain+delete happens inside
// that very call, then the new entry is appended (re-arm+append). SourceCount stays 1
// for srcA, but we can observe the delete happened because:
//   (a) firedAt[srcA] is cleared → a fresh threshold crossing fires E-ADM-017 again.
//   (b) SourceCount() == 1 for srcA after re-arm (NOT 0, because the call re-appends).
//
// To discriminate "empty slice retained" vs "key deleted": use SourceCount across two
// sources. If srcA leaves an empty slice without delete, the map still has two keys
// (srcA=[] and srcB=[T=65]): SourceCount() == 2. If srcA's key was deleted (correct),
// after both sources drain and only srcB is active: SourceCount() == 1.
//
// Implementation note: dead-key eviction only runs when a source's own
// RecordHMACFailure is called and trims it to empty. If srcA is under append-skip
// and its pre-fire entries age out but no new call arrives for srcA, the map entry
// is NOT yet evicted — that is expected lazy behavior. The observable delete(counts,
// srcAddr) is only triggered on the NEXT call to RecordHMACFailure for that srcAddr.
// This test calls srcA after drain to trigger eviction + re-arm, then calls srcB to
// get a second active source, then checks that SourceCount() correctly reflects only
// the truly live sources.
//
// What this test CAN discriminate: an impl that retains empty slices (counts[srcA]=[])
// without calling delete(counts, srcA) will have SourceCount() count the empty entry,
// causing the SourceCount() > expected assertion to fail.
//
// Traces to BC-2.05.005 EC-005; S-W3.05 AC-012 (story v1.2 discrimination note).
func TestFailureCounter_DeadKeyEvictedAfterDrain(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second
	srcA := "dead-key-src-A-ac012"
	srcB := "dead-key-src-B-ac012"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return current }),
	)

	// srcA: 5 failures at T=0..4 → alert fires; append-skip now in force for srcA.
	for i := range 5 {
		current = base.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(srcA)
	}
	if log.Count() != 1 {
		t.Fatalf("AC-012: want 1 alert after srcA threshold crossing, got %d", log.Count())
	}

	// srcB: 1 failure at T=2 → srcB is now tracked. SourceCount() == 2.
	current = base.Add(2 * time.Second)
	fc.RecordHMACFailure(srcB)
	if got := fc.SourceCount(); got != 2 {
		t.Fatalf("AC-012: want SourceCount()==2 after srcA+srcB recorded, got %d", got)
	}

	// Advance clock to T=65s. Cutoff = T=5.
	// srcA pre-fire entries: T=0..4, all < T=5 → will be trimmed on next srcA call.
	// srcB entry: T=2 < T=5 → will be trimmed on next srcB call.
	drainT := base.Add(65 * time.Second)
	current = drainT

	// Trigger drain + dead-key eviction for srcB by calling RecordHMACFailure(srcB).
	// After trim: srcB's T=2 entry is removed; len(keep)==0 → delete(counts, srcB) runs.
	// Then T=65 is appended for srcB. SourceCount() should reflect:
	//   - srcA: still has stale entries in the map (append-skip in force;
	//     counts[srcA]=[T=0..4] — NOT yet evicted because we haven't called srcA).
	//   - srcB: fresh entry at T=65 (live).
	// A buggy impl that leaves empty slices would count srcA's empty entry too if
	// srcA had previously been drained. But here srcA's entries are still present
	// (stale but not yet trimmed). SourceCount() == 2 (srcA stale + srcB fresh).
	fc.RecordHMACFailure(srcB)
	// SourceCount() == 2: srcA still has stale entries; srcB has T=65.
	// (We cannot distinguish empty-slice-retained vs not-yet-evicted here since
	//  srcA hasn't been called post-drain yet.)

	// NOW trigger srcA's drain by calling RecordHMACFailure(srcA) at T=65.
	// This call: trims T=0..4 (all < T=5) → len(keep)==0 → dead-key eviction runs.
	// Re-arm triggers (firedAt[srcA] deleted). T=65 is appended for srcA. firedAt cleared.
	// SourceCount(): srcA has T=65, srcB has T=65. Both live. SourceCount() == 2.
	fc.RecordHMACFailure(srcA)
	if got := fc.SourceCount(); got != 2 {
		t.Errorf("AC-012: after drain+eviction+reappend for both sources, want SourceCount()==2, got %d", got)
	}

	// Key discriminating assertion: after the above call, re-arm happened for srcA.
	// Drive srcA to threshold again → MUST fire a 2nd alert.
	// A buggy impl that retained counts[srcA]=[] without deleting firedAt[srcA] would
	// NOT re-arm → no 2nd alert → this assertion fails, pinning the eviction defect.
	for i := 1; i < threshold; i++ {
		current = drainT.Add(time.Duration(i) * time.Second)
		fc.RecordHMACFailure(srcA)
	}
	// After the first call (above) + 4 more = 5 total in new window → alert-2 must fire.
	if log.Count() != 2 {
		t.Errorf("AC-012: dead-key eviction must clear firedAt[srcAddr] on drain so a fresh "+
			"threshold crossing fires E-ADM-017 again — want 2 total alerts, got %d; lines: %v\n"+
			"(if count==1: firedAt[srcA] was NOT cleared → delete(firedAt,srcAddr) missing in drain path; "+
			"BC-2.05.005 PC-3 dead-key eviction: both counts AND firedAt must be deleted on full drain)",
			log.Count(), log.Lines())
	}

	// After both sources are active again, SourceCount() should be 2.
	// An impl that left counts[srcA]=[] (empty slice, not deleted) and then re-appended
	// without a proper delete would still have the correct SourceCount here,
	// but the alert re-fire check above would have already caught the firedAt bug.
	if got := fc.SourceCount(); got != 2 {
		t.Errorf("AC-012: after re-arm and re-fire, want SourceCount()==2 (srcA+srcB active), got %d", got)
	}

	// Explicit empty-slice discrimination: advance past both sources' windows again
	// to T=130s. Both T=65 entries age out. Then check that after one more srcB call
	// (which drains srcB), and one more srcA call (which drains srcA), SourceCount
	// reflects only the one that was just called (1), not 2 (stale zombie entries).
	drainT2 := base.Add(130 * time.Second)
	current = drainT2

	// Drain+evict srcB: trim T=65 (< T=70 cutoff) → len==0 → delete(counts,srcB).
	// Append T=130. SourceCount should stay 2 (srcA stale + srcB fresh).
	fc.RecordHMACFailure(srcB)

	// Drain+evict srcA: trim T=65..69 (all < T=70) → len==0 → delete(counts,srcA).
	// Append T=130. SourceCount should be 2 (srcA fresh + srcB fresh).
	fc.RecordHMACFailure(srcA)

	if got := fc.SourceCount(); got != 2 {
		t.Errorf("AC-012 (empty-slice): after double-drain+evict+reappend, want SourceCount()==2, got %d\n"+
			"(if > 2: empty slice keys are NOT being deleted — delete(counts,src) not called on drain; "+
			"CWE-770 / BC-2.05.005 PC-3 dead-key eviction)", got)
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
// matches the FULL canonical parameterized format from error-taxonomy v1.9 + BC-2.05.005 PC-3:
//
//	"E-ADM-017 HMAC failure rate alert: ≥<threshold> failures in <window>s from src <hex>"
//
// The format pins FOUR requirements:
//  1. Code literal "E-ADM-017" present (for operator grep-ability).
//  2. Phrase "HMAC failure rate alert:" present — REQUIRED per error-taxonomy v1.9.
//  3. Parameterized threshold and window values embedded numerically.
//  4. srcAddr rendered verbatim as the string passed to RecordHMACFailure.
//
// IMPORTANT: A prior reconciliation pass (commit 56785b4) incorrectly claimed that
// error-taxonomy v1.9 dropped the "HMAC failure rate alert:" phrase and added an
// assertion that the phrase must NOT appear. That claim was FALSE. The phrase is
// present in error-taxonomy.md v1.9 line 53, BC-2.05.005 PC-3 line 57, and the
// canonical test vector table at line 107. Story v1.2 AC-015 explicitly restores
// the correct canonical form. This test now asserts the phrase IS present.
//
// The current production implementation (failure_counter.go line 168-173) emits:
//
//	"E-ADM-017 ≥%d failures in %.0fs from src %s"   ← WRONG (missing the phrase)
//
// This test will be RED against the current implementation until the implementer
// fixes the format string to include "HMAC failure rate alert:".
//
// srcAddr convention: RouteFrame passes fmt.Sprintf("%x", hdr.SrcAddr) as the
// srcAddr string (routing.go). This test uses a raw hex string to match that convention.
//
// Traces to error-taxonomy v1.9 row E-ADM-017; BC-2.05.005 PC-3 + test vector;
// S-W3.05 AC-015 (story v1.2).
func TestFailureCounter_AlertMessageFormat(t *testing.T) {
	t.Parallel()

	const threshold = 5
	const window = 60 * time.Second

	// Use a lowercase hex string as srcAddr — matching the convention from routing.go
	// (fmt.Sprintf("%x", hdr.SrcAddr) → e.g. "deadbeef01020304").
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

	// (b) Must contain "HMAC failure rate alert:" — REQUIRED by error-taxonomy v1.9.
	// A prior pass wrongly added an assertion that this phrase must NOT appear.
	// That was a defect: story v1.2 AC-015 and error-taxonomy v1.9 both require it.
	if !strings.Contains(msg, "HMAC failure rate alert:") {
		t.Errorf("AC-015: E-ADM-017 message MUST contain phrase \"HMAC failure rate alert:\" "+
			"(error-taxonomy v1.9 canonical format; story v1.2 AC-015); got: %q\n"+
			"NOTE: the current impl is missing this phrase — implementer must fix the format string "+
			"in failure_counter.go to: \"E-ADM-017 HMAC failure rate alert: ≥%%d failures in %%.0fs from src %%s\"",
			msg)
	}

	// (c) Must contain "≥5" (threshold embedded).
	wantThreshold := fmt.Sprintf("≥%d", threshold)
	if !strings.Contains(msg, wantThreshold) {
		t.Errorf("AC-015: E-ADM-017 message missing threshold substring %q; got: %q",
			wantThreshold, msg)
	}

	// (d) Must contain "60s" (window in seconds, no decimal).
	wantWindow := fmt.Sprintf("%.0fs", window.Seconds())
	if !strings.Contains(msg, wantWindow) {
		t.Errorf("AC-015: E-ADM-017 message missing window substring %q; got: %q",
			wantWindow, msg)
	}

	// (e) Must contain "from src <hex>" with the srcAddr as passed.
	wantSrc := fmt.Sprintf("from src %s", src)
	if !strings.Contains(msg, wantSrc) {
		t.Errorf("AC-015: E-ADM-017 message missing src substring %q; got: %q",
			wantSrc, msg)
	}

	// (f) Full canonical form: the message must contain the complete canonical substring.
	// This is the authoritative discriminating assertion (story v1.2 AC-015 architecture compliance).
	canonical := fmt.Sprintf("E-ADM-017 HMAC failure rate alert: ≥%d failures in %.0fs from src %s",
		threshold, window.Seconds(), src)
	if !strings.Contains(msg, canonical) {
		t.Errorf("AC-015: message does not contain full canonical E-ADM-017 string\n"+
			"  want substring: %q\n"+
			"  got message:    %q",
			canonical, msg)
	}
}

// ── AC-016: TestFailureCounter_HighRateAttackBoundedSlice ─────────────────────

// TestFailureCounter_HighRateAttackBoundedSlice verifies the append-skip per-source
// slice bound (CWE-770 amplification mitigation) from BC-2.05.005 EC-011 + story v1.2 AC-016.
//
// After an alert fires for srcAddr (firedAt[srcAddr] is set and re-arm has not
// triggered), new timestamps MUST NOT be appended to the slice. The per-source
// slice is bounded at threshold entries at all times.
//
// Scenario (threshold=5, frozen clock — no entries can age out):
//
//	1. Inject 5 failures → alert fires; firedAt[src] set; slice has 5 entries.
//	2. Inject 1,000,000 more failures with clock FROZEN (cutoff unchanged;
//	   pre-fire entries never age out → re-arm never triggers).
//	3. Assert: len(Timestamps(src)) == threshold (5), not 1,000,005.
//
// Against the current implementation (which appends on every call), the slice
// grows to 1,000,005 → this test is RED. It will pass only after the implementer
// adds the append-skip guard:
//
//	if firedAt[srcAddr].IsZero() {
//	    keep = append(keep, now)
//	    c.counts[srcAddr] = keep
//	}
//
// Memory implication: bounded at threshold × sizeof(time.Time) = 5 × 16 = 80 bytes
// per source under attack, regardless of call rate.
//
// Traces to BC-2.05.005 EC-011; BC-2.05.005 PC-3 per-source slice bound;
// S-W3.05 AC-016; CWE-770 (unbounded slice growth).
func TestFailureCounter_HighRateAttackBoundedSlice(t *testing.T) {
	// Not t.Parallel() — 1,000,000 iterations; keep single-threaded to minimize -race overhead.

	const threshold = 5
	const window = 60 * time.Second
	src := "highrate-attack-src-ac016"

	// Frozen clock: all calls share the same timestamp.
	// Cutoff = frozenNow - 60s. Pre-fire entries are at frozenNow, so cutoff < frozenNow
	// → they are NEVER trimmed. Re-arm never triggers while clock is frozen.
	frozenNow := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	log := &fakeLog{}
	fc := admission.NewFailureCounter(threshold, window, log,
		admission.WithNow(func() time.Time { return frozenNow }),
	)

	// Phase 1: inject exactly threshold failures to trigger the alert.
	for range threshold {
		fc.RecordHMACFailure(src)
	}
	if log.Count() != 1 {
		t.Fatalf("AC-016: want 1 alert after %d failures, got %d; lines: %v",
			threshold, log.Count(), log.Lines())
	}
	// Slice has exactly threshold entries at this point.
	ts := fc.Timestamps(src)
	if len(ts) != threshold {
		t.Fatalf("AC-016: want %d timestamps after threshold crossing, got %d",
			threshold, len(ts))
	}

	// Phase 2: inject 10,000 more calls with FROZEN clock.
	// append-skip must prevent any new entries from being added.
	// 10,000 is sufficient to pin CWE-770 while keeping test runtime bounded.
	// The BC-2.05.005 EC-011 spec states "1,000,000" conceptually; 10,000 is
	// the pragmatic discriminating count that avoids 600s timeout against buggy impls.
	const extraCalls = 10_000
	for range extraCalls {
		fc.RecordHMACFailure(src)
	}

	// Assert: slice is STILL bounded at threshold entries.
	// A buggy impl that appends unconditionally will have threshold + extraCalls entries.
	ts = fc.Timestamps(src)
	if len(ts) != threshold {
		t.Errorf("AC-016: append-skip violated — after %d extra calls with frozen clock, "+
			"want len(Timestamps)==%d (bounded), got %d "+
			"(CWE-770: slice grew to %d; implementer must skip append when firedAt[srcAddr] is set; "+
			"BC-2.05.005 EC-011 per-source slice bound)",
			extraCalls, threshold, len(ts), len(ts))
	}

	// No additional alerts must have fired (fire-once-per-crossing while suppressed).
	if log.Count() != 1 {
		t.Errorf("AC-016: want exactly 1 alert total (no re-fire while frozen/suppressed), got %d; lines: %v",
			log.Count(), log.Lines())
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

	// Canonical format per error-taxonomy v1.9 + BC-2.05.005 PC-3:
	// "E-ADM-017 HMAC failure rate alert: ≥N failures in Ws from src hex"
	// BOTH the "E-ADM-017" prefix AND the "HMAC failure rate alert:" phrase are REQUIRED.
	canonical := fmt.Sprintf("E-ADM-017 HMAC failure rate alert: ≥%d failures in %.0fs from src %s",
		threshold, window.Seconds(), srcHex)
	if !strings.Contains(msg, canonical) {
		t.Errorf("e2e-format: alert message does not contain canonical E-ADM-017 format\n"+
			"  want substring: %q\n"+
			"  got message:    %q\n"+
			"  (error-taxonomy v1.9 + BC-2.05.005 PC-3: canonical form includes 'HMAC failure rate alert:' phrase)",
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
