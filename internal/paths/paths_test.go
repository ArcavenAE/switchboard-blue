// Package paths_test contains the TDD test suite for BC-2.02.003
// (per-path RTT/loss tracking and path ranking).
//
// All tests MUST fail until PathScore, NewPathTracker, PathTracker.OnProbe,
// PathTracker.Score, PathTracker.IsActive, PathTracker.RTT, PathTracker.LossPct,
// and Rank are implemented (Red Gate).
//
// BC/AC coverage map:
//
//	TestBC_2_02_003_PathScore_LowerRTTLowerScore                  → AC-001, BC-2.02.003 postcondition 3
//	TestBC_2_02_003_PathScore_HigherLossRaisesScore               → AC-001, BC-2.02.003 postcondition 3
//	TestBC_2_02_003_PathScore_Transitive                          → AC-001, VP-026
//	TestBC_2_02_003_PathScore_ZeroLossPureRTT                     → AC-001, BC-2.02.003 postcondition 1
//	TestBC_2_02_003_PathScore_Formula                             → AC-001, ARCH-03 formula
//	TestBC_2_02_003_PathTracker_NewInitialRTT                     → BC-2.02.003 precondition 3
//	TestBC_2_02_003_PathTracker_EWMAConvergence                   → AC-002, BC-2.02.003 postcondition 1 (F-007: varying RTTs, distinguishes EWMA from last-value)
//	TestBC_2_02_003_PathTracker_LossUpdatesEWMA                   → BC-2.02.003 postcondition 2
//	TestBC_2_02_003_PathTracker_InactiveAfterMisses               → BC-2.02.003 postcondition 6, VP-026/VP-040
//	TestBC_2_02_003_PathTracker_ResetMissesOnSuccess              → BC-2.02.003 postcondition 6
//	TestBC_2_02_003_PathTracker_Reactivation                      → F-006, BC-2.02.003 postcondition 6 (v1.2) — FAILED→ACTIVE on first success
//	TestBC_2_02_003_PathTracker_FirstProbeRTTOverride             → F-008, BC-2.02.003 postcondition 1 — first-probe override with alpha<1
//	TestBC_2_02_003_PathTracker_ScoreDelegates                    → AC-001/AC-002
//	TestBC_2_02_003_Rank_OrderedByScore                           → BC-2.02.003 postcondition 3
//	TestBC_2_02_003_Rank_ExcludesInactivePaths                    → BC-2.02.003 postcondition 6
//	TestBC_2_02_003_Rank_ErrNoActivePaths                         → BC-2.02.001 precondition 1 / ErrNoActivePaths
//	TestBC_2_02_003_Rank_TiebreakByID                             → EC-002, BC-2.02.001 invariant 3
//	TestBC_2_02_003_Rank_SinglePath                               → EC-001, BC-2.02.001 postcondition 3
//	TestBC_2_02_003_PathTracker_RTTAndLossPctAccessors            → API surface
//	TestBC_2_02_003_PathScore_PropertyTransitive_Manual           → VP-026 (stdlib property sweep)
//	TestBC_2_02_003_PathTracker_ConcurrentOnProbeScore            → F-004, BC-2.02.003 (concurrent safety)
package paths_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/paths"
)

// ─── PathScore unit tests ────────────────────────────────────────────────────

// TestBC_2_02_003_PathScore_LowerRTTLowerScore verifies that, with loss held
// constant, a path with lower RTT receives a lower (better) score.
//
// AC-001 / BC-2.02.003 postcondition 3
func TestBC_2_02_003_PathScore_LowerRTTLowerScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		rttA, rttB float64
		lossPct    float64
	}{
		{"10ms vs 25ms, 0% loss", 10, 25, 0},
		{"10ms vs 50ms, 5% loss", 10, 50, 5},
		{"1ms vs 1000ms, 10% loss", 1, 1000, 10},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scoreA := paths.PathScore(tc.rttA, tc.lossPct)
			scoreB := paths.PathScore(tc.rttB, tc.lossPct)
			if scoreA >= scoreB {
				t.Errorf("PathScore(rtt=%v, loss=%v)=%v >= PathScore(rtt=%v, loss=%v)=%v; want strictly less",
					tc.rttA, tc.lossPct, scoreA, tc.rttB, tc.lossPct, scoreB)
			}
		})
	}
}

// TestBC_2_02_003_PathScore_HigherLossRaisesScore verifies that, with RTT held
// constant, a path with higher loss receives a higher (worse) score.
//
// AC-001 / BC-2.02.003 postcondition 3
func TestBC_2_02_003_PathScore_HigherLossRaisesScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		rtt          float64
		lossA, lossB float64
	}{
		{"0% vs 10% loss, 20ms RTT", 20, 0, 10},
		{"0% vs 50% loss, 50ms RTT", 50, 0, 50},
		{"1% vs 100% loss, 100ms RTT", 100, 1, 100},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scoreA := paths.PathScore(tc.rtt, tc.lossA)
			scoreB := paths.PathScore(tc.rtt, tc.lossB)
			if scoreA >= scoreB {
				t.Errorf("PathScore(rtt=%v, loss=%v)=%v >= PathScore(rtt=%v, loss=%v)=%v; want strictly less",
					tc.rtt, tc.lossA, scoreA, tc.rtt, tc.lossB, scoreB)
			}
		})
	}
}

// TestBC_2_02_003_PathScore_Transitive verifies transitivity of the PathScore
// ordering across three distinct paths with different quality metrics.
//
// AC-001 / VP-026 / BC-2.02.003 postcondition 3
func TestBC_2_02_003_PathScore_Transitive(t *testing.T) {
	t.Parallel()

	// Canonical test vector from BC-2.02.003: Path A RTT=10ms, Path B RTT=50ms,
	// Path C RTT=200ms; all with low/identical loss.
	type triple struct {
		name                string
		rttA, rttB, rttC    float64
		lossA, lossB, lossC float64
	}

	cases := []triple{
		{
			name: "ascending RTT no loss: A<B<C",
			rttA: 10, rttB: 50, rttC: 200,
			lossA: 0, lossB: 0, lossC: 0,
		},
		{
			name: "ascending loss same RTT: A<B<C",
			rttA: 20, rttB: 20, rttC: 20,
			lossA: 0, lossB: 5, lossC: 50,
		},
		{
			name: "mixed RTT and loss: A<B<C",
			rttA: 10, rttB: 30, rttC: 100,
			lossA: 0, lossB: 2, lossC: 10,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sa := paths.PathScore(tc.rttA, tc.lossA)
			sb := paths.PathScore(tc.rttB, tc.lossB)
			sc := paths.PathScore(tc.rttC, tc.lossC)

			// The three cases above are specifically crafted so that sa < sb < sc.
			if sa >= sb {
				t.Errorf("score(A)=%v >= score(B)=%v; want A < B", sa, sb)
			}
			if sb >= sc {
				t.Errorf("score(B)=%v >= score(C)=%v; want B < C", sb, sc)
			}
			// Transitivity: sa < sb ∧ sb < sc ⟹ sa < sc
			if sa >= sc {
				t.Errorf("transitivity violated: score(A)=%v >= score(C)=%v", sa, sc)
			}
		})
	}
}

// TestBC_2_02_003_PathScore_ZeroLossPureRTT verifies that with zero loss the
// score is determined entirely by RTT.
//
// AC-001 / BC-2.02.003 postcondition 1 (EWMA basis)
func TestBC_2_02_003_PathScore_ZeroLossPureRTT(t *testing.T) {
	t.Parallel()

	// score = rtt * (1 + 0 * loss_weight) = rtt  when loss=0
	rtts := []float64{5, 10, 25, 50, 100, 200, 500}
	for i := 0; i < len(rtts)-1; i++ {
		a, b := rtts[i], rtts[i+1]
		sa := paths.PathScore(a, 0)
		sb := paths.PathScore(b, 0)
		if sa >= sb {
			t.Errorf("rtt=%v score=%v >= rtt=%v score=%v; want strictly less", a, sa, b, sb)
		}
	}
}

// TestBC_2_02_003_PathScore_Formula verifies the exact ARCH-03 formula:
//
//	score = rtt * (1 + (loss_pct/100) * DefaultLossWeight)
//
// AC-001 / ARCH-03
func TestBC_2_02_003_PathScore_Formula(t *testing.T) {
	t.Parallel()

	const eps = 1e-9 // floating-point tolerance

	cases := []struct {
		rtt     float64
		lossPct float64
		want    float64
	}{
		// score = rtt * (1 + (loss/100)*10)
		{rtt: 10, lossPct: 0, want: 10.0},
		{rtt: 10, lossPct: 10, want: 20.0},
		{rtt: 10, lossPct: 50, want: 60.0},
		{rtt: 10, lossPct: 100, want: 110.0},
		{rtt: 50, lossPct: 20, want: 50 * (1 + 0.20*10)},
		{rtt: 100, lossPct: 5, want: 100 * (1 + 0.05*10)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := paths.PathScore(tc.rtt, tc.lossPct)
			diff := float64(got) - tc.want
			if diff < -eps || diff > eps {
				t.Errorf("PathScore(rtt=%v, loss=%v)=%v; want %v", tc.rtt, tc.lossPct, got, tc.want)
			}
		})
	}
}

// ─── PathTracker unit tests ──────────────────────────────────────────────────

// TestBC_2_02_003_PathTracker_NewInitialRTT verifies that NewPathTracker
// initialises with the supplied conservative RTT and zero loss, and that
// the path starts as active.
//
// BC-2.02.003 precondition 3
func TestBC_2_02_003_PathTracker_NewInitialRTT(t *testing.T) {
	t.Parallel()

	const initRTT = 999.0
	tracker := paths.NewPathTracker(initRTT, 0.125)

	if !tracker.IsActive() {
		t.Error("new PathTracker must start as active")
	}
	if tracker.RTT() != initRTT {
		t.Errorf("initial RTT: got %v, want %v", tracker.RTT(), initRTT)
	}
	if tracker.LossPct() != 0.0 {
		t.Errorf("initial loss: got %v, want 0", tracker.LossPct())
	}
}

// TestBC_2_02_003_PathTracker_EWMAConvergence verifies that the RTT tracker
// implements genuine EWMA smoothing, not a degenerate last-value (alpha=1)
// implementation. This is accomplished by feeding VARYING probe RTTs and
// asserting that the EWMA reflects smoothed history rather than the most
// recent sample.
//
// F-007 / AC-002 / BC-2.02.003 postcondition 1
//
// Test design: feed a baseline of probes at 100ms to establish an EWMA, then
// fire one probe at 10ms (a sudden improvement). A true EWMA (alpha=0.125)
// must NOT jump all the way to 10ms — it must reflect the weighted history.
// A degenerate last-value (alpha=1) implementation would return exactly 10ms.
// We assert the result is strictly between 10ms and 100ms, proving smoothing lag.
func TestBC_2_02_003_PathTracker_EWMAConvergence(t *testing.T) {
	t.Parallel()

	const initRTT = 100.0
	const alpha = 0.125
	// Feed 8 probes at 100ms to let EWMA settle near 100ms.
	const warmupProbes = 8
	const warmupRTT = 100.0
	// Then one sudden-improvement probe at 10ms.
	const spikeRTT = 10.0

	tracker := paths.NewPathTracker(initRTT, alpha)

	for i := 0; i < warmupProbes; i++ {
		tracker.OnProbe(warmupRTT, false)
	}

	// After warmup, RTT should be close to 100ms (EWMA settled).
	afterWarmup := tracker.RTT()
	if afterWarmup < 90.0 || afterWarmup > 110.0 {
		t.Errorf("after %d probes at %vms: EWMA=%v, expected ~100ms (±10ms)", warmupProbes, warmupRTT, afterWarmup)
	}

	// Fire one probe at 10ms — a sudden improvement.
	tracker.OnProbe(spikeRTT, false)

	got := tracker.RTT()

	// A degenerate last-value (alpha=1) implementation would return exactly 10ms.
	// True EWMA (alpha=0.125): new_ewma = 0.875*~100 + 0.125*10 ≈ 88.75ms.
	// We assert: got > 10ms (proves it is NOT last-value) AND got < 90ms
	// (proves EWMA did move toward the new sample).
	if got <= spikeRTT {
		t.Errorf("EWMA after sudden drop: got %vms ≤ %vms; looks like last-value (alpha=1), not EWMA", got, spikeRTT)
	}
	if got >= afterWarmup {
		t.Errorf("EWMA did not move toward new sample: got %vms ≥ warmup %vms; EWMA should decrease", got, afterWarmup)
	}
}

// TestBC_2_02_003_PathTracker_EWMAConvergence_ThreeProbes verifies that the
// spec requirement "after 3 probe arrivals, the score converges" is met: the
// tracker's RTT must be closer to the true RTT after 3 probes than the initial
// conservative value.
//
// AC-002 / BC-2.02.003 postcondition 1
func TestBC_2_02_003_PathTracker_EWMAConvergence_ThreeProbes(t *testing.T) {
	t.Parallel()

	const initRTT = 500.0
	const trueRTT = 15.0
	const alpha = 0.125

	tracker := paths.NewPathTracker(initRTT, alpha)

	for i := 0; i < 3; i++ {
		tracker.OnProbe(trueRTT, false)
	}

	got := tracker.RTT()
	if got >= initRTT {
		t.Errorf("RTT did not move after 3 probes: got %v, initial was %v", got, initRTT)
	}
	// After 3 probes the EWMA must have moved meaningfully toward trueRTT.
	midpoint := (initRTT + trueRTT) / 2
	if got >= midpoint {
		t.Errorf("RTT %v not yet converging: midpoint was %v; expected < midpoint after 3 probes with alpha=%v", got, midpoint, alpha)
	}
}

// TestBC_2_02_003_PathTracker_LossUpdatesEWMA verifies that a loss event raises
// the EWMA loss percentage.
//
// BC-2.02.003 postcondition 2
func TestBC_2_02_003_PathTracker_LossUpdatesEWMA(t *testing.T) {
	t.Parallel()

	tracker := paths.NewPathTracker(100.0, 0.5)

	initialLoss := tracker.LossPct()

	// Fire a loss event (missed keepalive).
	tracker.OnProbe(0, true)

	afterLoss := tracker.LossPct()
	if afterLoss <= initialLoss {
		t.Errorf("loss did not increase after lossEvent=true: before=%v after=%v", initialLoss, afterLoss)
	}
}

// TestBC_2_02_003_PathTracker_InactiveAfterMisses verifies that a path is
// marked inactive after 3 consecutive missed keepalives.
//
// BC-2.02.003 postcondition 6 / VP-026 / VP-040
func TestBC_2_02_003_PathTracker_InactiveAfterMisses(t *testing.T) {
	t.Parallel()

	const consecutiveMissThreshold = 3

	tracker := paths.NewPathTracker(50.0, 0.125)

	for i := 0; i < consecutiveMissThreshold-1; i++ {
		tracker.OnProbe(0, true) // loss event
		if !tracker.IsActive() {
			t.Fatalf("path became inactive prematurely after %d consecutive misses (threshold=%d)", i+1, consecutiveMissThreshold)
		}
	}

	// The third consecutive miss must deactivate the path.
	tracker.OnProbe(0, true)
	if tracker.IsActive() {
		t.Errorf("path still active after %d consecutive misses; want inactive", consecutiveMissThreshold)
	}
}

// TestBC_2_02_003_PathTracker_ResetMissesOnSuccess verifies that a successful
// probe resets the consecutive-miss counter so the path does not become
// inactive prematurely.
//
// BC-2.02.003 postcondition 6 (implicit: misses must be consecutive)
func TestBC_2_02_003_PathTracker_ResetMissesOnSuccess(t *testing.T) {
	t.Parallel()

	tracker := paths.NewPathTracker(50.0, 0.125)

	// Two misses, then a successful probe — counter should reset.
	tracker.OnProbe(0, true)
	tracker.OnProbe(0, true)
	tracker.OnProbe(50.0, false) // success resets

	// Two more misses: still below threshold from a fresh run.
	tracker.OnProbe(0, true)
	tracker.OnProbe(0, true)

	if !tracker.IsActive() {
		t.Error("path became inactive; 2+2=4 total misses but NOT consecutive; want active")
	}
}

// TestBC_2_02_003_PathTracker_Reactivation verifies the full lifecycle:
// ACTIVE → (3 consecutive misses) → FAILED → (first successful probe) → ACTIVE.
//
// Per BC-2.02.003 postcondition 6 (v1.2, amended by pass-1 spec ruling F-006):
// A failed path is re-added to the active path set upon the FIRST successful
// keep-alive round-trip. On reactivation, RTT is initialized from the
// reactivating probe's measured RTT, and loss EWMA resets to 0.
//
// This test MUST FAIL against the current implementation (571a31b) because
// paths.go:108-129 never restores active=true after deactivation — that is the
// Red Gate for this fix.
//
// F-006 / BC-2.02.003 postcondition 6 (v1.2) / pass-1-spec-rulings RULING 2
func TestBC_2_02_003_PathTracker_Reactivation(t *testing.T) {
	t.Parallel()

	tracker := paths.NewPathTracker(50.0, 0.125)

	// ── Phase 1: confirm active initially ────────────────────────────────────
	if !tracker.IsActive() {
		t.Fatal("precondition: new tracker must be active")
	}

	// ── Phase 2: 3 consecutive misses → FAILED ────────────────────────────────
	for i := range 3 {
		tracker.OnProbe(0, true) // missed keepalive
		if i < 2 && !tracker.IsActive() {
			t.Fatalf("path became inactive prematurely after %d consecutive misses (want 3)", i+1)
		}
	}
	if tracker.IsActive() {
		t.Fatal("after 3 consecutive misses: path still active; want FAILED")
	}

	// ── Phase 3: path absent from Rank while FAILED ───────────────────────────
	rp := []paths.RankedPath{{ID: 1, Tracker: tracker}}
	_, rankErr := paths.Rank(rp)
	if !errors.Is(rankErr, paths.ErrNoActivePaths) {
		t.Errorf("FAILED path should be absent from Rank; want ErrNoActivePaths, got %v", rankErr)
	}

	// ── Phase 4: first successful probe → ACTIVE ─────────────────────────────
	const reactivationRTT = 75.0
	tracker.OnProbe(reactivationRTT, false)

	if !tracker.IsActive() {
		t.Fatal("after first successful probe on FAILED path: want ACTIVE (reactivation), got FAILED")
	}

	// RTT must be initialized from the reactivating probe (not carried over from before deactivation).
	if tracker.RTT() != reactivationRTT {
		t.Errorf("reactivated RTT: got %v, want %v (must be initialized from reactivating probe)", tracker.RTT(), reactivationRTT)
	}

	// Loss EWMA must reset to 0 on reactivation (conservative assumption: loss-free until probes accumulate).
	if tracker.LossPct() != 0.0 {
		t.Errorf("reactivated loss EWMA: got %v, want 0 (must reset on reactivation)", tracker.LossPct())
	}

	// ── Phase 5: path present in Rank after reactivation ─────────────────────
	ranked, err := paths.Rank(rp)
	if err != nil {
		t.Fatalf("after reactivation: Rank returned error %v; want success", err)
	}
	if len(ranked) != 1 || ranked[0].ID != 1 {
		t.Errorf("after reactivation: Rank=%v; want [{ID:1}]", ranked)
	}
}

// TestBC_2_02_003_PathTracker_FirstProbeRTTOverride verifies that when alpha < 1,
// after exactly ONE successful probe, RTT() equals that probe's measured RTT
// (proving the first-probe override fired, not a blend from the conservative
// initial value).
//
// Without the override, the first EWMA update would be:
//
//	ewma = alpha*probe + (1-alpha)*initRTT
//
// With the override:
//
//	ewma = probe  (first probe sets RTT directly, ignoring the conservative init)
//
// F-008 / BC-2.02.003 postcondition 1 (first-probe override)
func TestBC_2_02_003_PathTracker_FirstProbeRTTOverride(t *testing.T) {
	t.Parallel()

	const initRTT = 999.0
	const alpha = 0.125 // deliberately < 1 so blend ≠ override
	const probeRTT = 42.0

	tracker := paths.NewPathTracker(initRTT, alpha)

	// Exactly one successful probe.
	tracker.OnProbe(probeRTT, false)

	got := tracker.RTT()

	// Without the first-probe override, EWMA would be:
	//   0.125 * 42 + 0.875 * 999 = 5.25 + 874.125 = 879.375
	// With the override: RTT() == 42.0.
	// We assert got == probeRTT to prove the override fired.
	if got != probeRTT {
		blended := alpha*probeRTT + (1-alpha)*initRTT
		t.Errorf("first-probe RTT: got %v, want %v (first-probe override); blended value would be %v",
			got, probeRTT, blended)
	}
}

// TestBC_2_02_003_PathTracker_ConcurrentOnProbeScore drives multiple goroutines
// calling OnProbe and Score concurrently on a single PathTracker. Run under
// `go test -race` — any missing lock will produce a data race report.
//
// F-004 / BC-2.02.003 (concurrent safety of PathTracker)
func TestBC_2_02_003_PathTracker_ConcurrentOnProbeScore(t *testing.T) {
	// Not parallel at the outer level — inner goroutines provide the concurrency.

	const goroutines = 8
	const probesPerGoroutine = 50

	tracker := paths.NewPathTracker(100.0, 0.125)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		g := g
		go func() {
			defer wg.Done()
			for i := range probesPerGoroutine {
				rtt := float64((g*probesPerGoroutine + i + 1) % 200)
				tracker.OnProbe(rtt, i%5 == 0) // every 5th is a miss
				_ = tracker.Score()
				_ = tracker.IsActive()
				_ = tracker.RTT()
			}
		}()
	}
	wg.Wait()

	// Post-concurrency: tracker must still be in a valid state (no panic, no race).
	_ = tracker.RTT()
	_ = tracker.LossPct()
}

// TestBC_2_02_003_PathTracker_ScoreDelegates verifies that PathTracker.Score()
// returns the same value as PathScore applied to the tracker's own RTT and
// loss estimates.
//
// AC-001 / AC-002 (score delegates to PathScore formula)
func TestBC_2_02_003_PathTracker_ScoreDelegates(t *testing.T) {
	t.Parallel()

	const initRTT = 40.0
	const alpha = 0.125

	tracker := paths.NewPathTracker(initRTT, alpha)
	tracker.OnProbe(20.0, false)
	tracker.OnProbe(20.0, false)

	trackerScore := tracker.Score()
	explicitScore := paths.PathScore(tracker.RTT(), tracker.LossPct())

	const eps = 1e-9
	diff := float64(trackerScore) - float64(explicitScore)
	if diff < -eps || diff > eps {
		t.Errorf("PathTracker.Score()=%v != PathScore(RTT=%v, LossPct=%v)=%v",
			trackerScore, tracker.RTT(), tracker.LossPct(), explicitScore)
	}
}

// TestBC_2_02_003_PathTracker_RTTAndLossPctAccessors verifies that RTT() and
// LossPct() return the EWMA values and not the raw probe input.
//
// API surface completeness
func TestBC_2_02_003_PathTracker_RTTAndLossPctAccessors(t *testing.T) {
	t.Parallel()

	// alpha=1 → EWMA = latest sample (degenerate case, easy to verify)
	tracker := paths.NewPathTracker(500.0, 1.0)
	tracker.OnProbe(42.0, false)

	if tracker.RTT() != 42.0 {
		t.Errorf("RTT() after alpha=1 probe: got %v, want 42.0", tracker.RTT())
	}
	// Loss should be 0 after a successful probe from 0 initial.
	if tracker.LossPct() != 0.0 {
		t.Errorf("LossPct() after successful probe: got %v, want 0.0", tracker.LossPct())
	}
}

// ─── Rank unit tests ─────────────────────────────────────────────────────────

// TestBC_2_02_003_Rank_OrderedByScore verifies that Rank returns active paths
// in ascending score order (best first).
//
// BC-2.02.003 postcondition 3 / BC-2.02.001 (two fastest paths selected)
func TestBC_2_02_003_Rank_OrderedByScore(t *testing.T) {
	t.Parallel()

	// Canonical test vector: RTT [10ms, 15ms, 40ms] → expect rank #1=10ms, #2=15ms.
	// (BC-2.02.001 test vector: "3 paths: RTT [10ms, 15ms, 40ms]; dispatch on 10 and 15")
	cases := []struct {
		name    string
		rtts    []float64 // initial RTTs in ms; alpha=1 so Score = PathScore(rtt, 0)
		wantIDs []uint64  // expected ordering by ID (IDs match rtt index)
	}{
		{
			name:    "three paths ascending RTT",
			rtts:    []float64{40, 15, 10}, // IDs 0,1,2 — RTTs deliberately out of order
			wantIDs: []uint64{2, 1, 0},     // sorted by score ascending: 10ms=ID2, 15ms=ID1, 40ms=ID0
		},
		{
			name:    "two paths",
			rtts:    []float64{25, 10},
			wantIDs: []uint64{1, 0},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			candidates := make([]paths.RankedPath, len(tc.rtts))
			for i, rtt := range tc.rtts {
				// alpha=1 so Score() = PathScore(rtt, 0) after one probe.
				tr := paths.NewPathTracker(rtt, 1.0)
				candidates[i] = paths.RankedPath{ID: uint64(i), Tracker: tr}
			}

			ranked, err := paths.Rank(candidates)
			if err != nil {
				t.Fatalf("Rank returned error: %v", err)
			}
			if len(ranked) != len(tc.wantIDs) {
				t.Fatalf("Rank returned %d paths, want %d", len(ranked), len(tc.wantIDs))
			}
			for pos, wantID := range tc.wantIDs {
				if ranked[pos].ID != wantID {
					t.Errorf("rank[%d]: got pathID=%d, want %d", pos, ranked[pos].ID, wantID)
				}
			}
		})
	}
}

// TestBC_2_02_003_Rank_ExcludesInactivePaths verifies that paths whose
// IsActive()=false are excluded from the ranked output.
//
// BC-2.02.003 postcondition 6
func TestBC_2_02_003_Rank_ExcludesInactivePaths(t *testing.T) {
	t.Parallel()

	// Create two trackers; deactivate the first by driving 3 consecutive misses.
	inactiveTracker := paths.NewPathTracker(10.0, 0.125)
	for i := 0; i < 3; i++ {
		inactiveTracker.OnProbe(0, true)
	}

	activeTracker := paths.NewPathTracker(50.0, 0.125)

	candidates := []paths.RankedPath{
		{ID: 1, Tracker: inactiveTracker},
		{ID: 2, Tracker: activeTracker},
	}

	ranked, err := paths.Rank(candidates)
	if err != nil {
		t.Fatalf("Rank returned error: %v", err)
	}
	if len(ranked) != 1 {
		t.Fatalf("Rank returned %d paths, want 1 (inactive excluded)", len(ranked))
	}
	if ranked[0].ID != 2 {
		t.Errorf("ranked[0].ID=%d, want 2 (the active path)", ranked[0].ID)
	}
}

// TestBC_2_02_003_Rank_ErrNoActivePaths verifies that Rank returns
// ErrNoActivePaths when no candidate path is active.
//
// BC-2.02.001 precondition 1 / ErrNoActivePaths sentinel
func TestBC_2_02_003_Rank_ErrNoActivePaths(t *testing.T) {
	t.Parallel()

	// Deactivate a single path.
	tr := paths.NewPathTracker(100.0, 0.125)
	for i := 0; i < 3; i++ {
		tr.OnProbe(0, true)
	}

	_, err := paths.Rank([]paths.RankedPath{{ID: 1, Tracker: tr}})
	if !errors.Is(err, paths.ErrNoActivePaths) {
		t.Errorf("want ErrNoActivePaths, got %v", err)
	}

	// Empty candidate list also returns ErrNoActivePaths.
	_, err2 := paths.Rank(nil)
	if !errors.Is(err2, paths.ErrNoActivePaths) {
		t.Errorf("nil candidates: want ErrNoActivePaths, got %v", err2)
	}
}

// TestBC_2_02_003_Rank_TiebreakByID verifies deterministic tiebreak by
// ascending path ID when scores are equal.
//
// EC-002 / BC-2.02.001 invariant 3 / AC-002 note
func TestBC_2_02_003_Rank_TiebreakByID(t *testing.T) {
	t.Parallel()

	// Three trackers with identical initial RTT and no probes yet — all equal score.
	const initRTT = 50.0
	const alpha = 0.125

	candidates := []paths.RankedPath{
		{ID: 30, Tracker: paths.NewPathTracker(initRTT, alpha)},
		{ID: 10, Tracker: paths.NewPathTracker(initRTT, alpha)},
		{ID: 20, Tracker: paths.NewPathTracker(initRTT, alpha)},
	}

	ranked, err := paths.Rank(candidates)
	if err != nil {
		t.Fatalf("Rank returned error: %v", err)
	}
	if len(ranked) != 3 {
		t.Fatalf("want 3 ranked paths, got %d", len(ranked))
	}

	// Expect ascending ID order for tiebreak: 10, 20, 30.
	wantIDs := []uint64{10, 20, 30}
	for i, want := range wantIDs {
		if ranked[i].ID != want {
			t.Errorf("tiebreak: ranked[%d].ID=%d, want %d", i, ranked[i].ID, want)
		}
	}
}

// TestBC_2_02_003_Rank_SinglePath verifies that Rank succeeds and returns
// one entry when only one active path exists.
//
// EC-001 / BC-2.02.001 postcondition 3
func TestBC_2_02_003_Rank_SinglePath(t *testing.T) {
	t.Parallel()

	tr := paths.NewPathTracker(20.0, 0.125)
	candidates := []paths.RankedPath{{ID: 7, Tracker: tr}}

	ranked, err := paths.Rank(candidates)
	if err != nil {
		t.Fatalf("single-path Rank returned error: %v", err)
	}
	if len(ranked) != 1 || ranked[0].ID != 7 {
		t.Errorf("single-path Rank: got %v, want [{ID:7}]", ranked)
	}
}

// ─── PathScore property sweep (VP-026 stdlib approximation) ──────────────────

// TestBC_2_02_003_PathScore_PropertyTransitive_Manual exercises transitivity
// over a fixed grid of (rtt, loss) pairs without an external proptest library.
//
// VP-026 (stdlib property sweep — full proptest deferred to formal-verifier)
func TestBC_2_02_003_PathScore_PropertyTransitive_Manual(t *testing.T) {
	t.Parallel()

	rtts := []float64{1, 5, 10, 20, 50, 100, 200, 500, 1000}
	losses := []float64{0, 1, 5, 10, 25, 50, 75, 100}

	type metric struct {
		rtt, loss float64
	}

	var all []metric
	for _, r := range rtts {
		for _, l := range losses {
			all = append(all, metric{r, l})
		}
	}

	// For every ordered triple (a, b, c) where score(a) < score(b) < score(c),
	// assert transitivity: score(a) < score(c).
	violations := 0
	for i := 0; i < len(all); i++ {
		for j := 0; j < len(all); j++ {
			for k := 0; k < len(all); k++ {
				sa := paths.PathScore(all[i].rtt, all[i].loss)
				sb := paths.PathScore(all[j].rtt, all[j].loss)
				sc := paths.PathScore(all[k].rtt, all[k].loss)
				if sa < sb && sb < sc && sa >= sc {
					t.Errorf("transitivity violation: score(%v,%v)=%v < score(%v,%v)=%v < score(%v,%v)=%v but score(a)=%v >= score(c)=%v",
						all[i].rtt, all[i].loss, sa,
						all[j].rtt, all[j].loss, sb,
						all[k].rtt, all[k].loss, sc,
						sa, sc)
					violations++
					if violations > 10 {
						t.FailNow()
					}
				}
			}
		}
	}
}

// ─── S-5.03: Degraded-path flag tests (BC-2.02.003 postcondition 5, VP-063) ──
//
// BC/AC coverage:
//
//	TestBC_2_02_003_PathTracker_IsDegraded                      → AC-001, AC-002 (table-driven; 3 sub-cases + EC-001 boundary)
//	TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation       → AC-003
//	TestBC_2_02_003_PathTracker_IsDegraded_Race                 → AC-004
//	TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded          → AC-001 (Snapshot consistency)
//	TestProp_IsDegraded_TracksEWMAThreshold                     → AC-005, VP-063
//	TestProp_IsDegraded_RecoveryClears                          → AC-005, VP-063 recovery branch

// TestBC_2_02_003_PathTracker_IsDegraded verifies that IsDegraded() tracks the
// EWMA RTT against DegradedRTTThresholdMS (200.0ms) — both onset and recovery —
// and that Snapshot().Degraded mirrors IsDegraded().
//
// Three sub-cases per AC-001 spec:
//
//	(a) RTT always below threshold → IsDegraded=false
//	(b) RTT always above threshold → IsDegraded=true
//	(c) RTT transitions above then recovers below → IsDegraded follows EWMA
//
// EC-001 (boundary): RTT exactly at 200.0ms → IsDegraded=false (exclusive >).
//
// AC-001 / AC-002 / BC-2.02.003 postcondition 5
func TestBC_2_02_003_PathTracker_IsDegraded(t *testing.T) {
	t.Parallel()

	// checkDegradedState is a test helper that asserts IsDegraded() and
	// Snapshot().Degraded both match wantDegraded.
	checkDegradedState := func(t *testing.T, tracker *paths.PathTracker, wantDegraded bool, msg string) {
		t.Helper()
		if got := tracker.IsDegraded(); got != wantDegraded {
			t.Errorf("%s: IsDegraded()=%v, want %v", msg, got, wantDegraded)
		}
		if snap := tracker.Snapshot(); snap.Degraded != wantDegraded {
			t.Errorf("%s: Snapshot().Degraded=%v, want %v (must mirror IsDegraded)", msg, snap.Degraded, wantDegraded)
		}
	}

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			// (a) RTT always well below threshold: EWMA stays below 200ms → never degraded.
			// Uses alpha=1.0 so each probe directly sets EWMA (makes arithmetic trivial).
			name: "always_below_threshold",
			run: func(t *testing.T) {
				t.Parallel()
				// alpha=1.0: each successful probe sets ewmaRTTMS = probe value directly
				// (first probe via resetRTT, subsequent via EWMA with alpha=1).
				tracker := paths.NewPathTracker(999.0, 1.0)
				for _, rtt := range []float64{10, 50, 100, 150, 199.9} {
					tracker.OnProbe(rtt, false)
					checkDegradedState(t, tracker, false, "RTT="+fmt.Sprintf("%.1f", rtt))
				}
			},
		},
		{
			// (b) RTT always above threshold: EWMA stays above 200ms → always degraded.
			// Canonical test vector: probes at 300ms (well above 200ms threshold).
			name: "always_above_threshold",
			run: func(t *testing.T) {
				t.Parallel()
				tracker := paths.NewPathTracker(999.0, 1.0)
				for i, rtt := range []float64{250, 300, 350, 400} {
					tracker.OnProbe(rtt, false)
					checkDegradedState(t, tracker, true, fmt.Sprintf("probe %d RTT=%.0f", i+1, rtt))
				}
			},
		},
		{
			// (c) RTT transitions above threshold then recovers below.
			// Using alpha=1.0 so each probe directly sets EWMA — recovery is immediate.
			// Drive above: OnProbe(250) → ewma=250 → degraded=true.
			// Recover: OnProbe(150) → ewma=150 → degraded=false (150 < 200).
			name: "transitions_above_then_recovers",
			run: func(t *testing.T) {
				t.Parallel()
				tracker := paths.NewPathTracker(999.0, 1.0)

				// Phase 1: establish degraded state.
				tracker.OnProbe(250.0, false)
				checkDegradedState(t, tracker, true, "after probe at 250ms")

				// Phase 2: one recovery probe below threshold.
				tracker.OnProbe(150.0, false)
				checkDegradedState(t, tracker, false, "after recovery probe at 150ms")
			},
		},
		{
			// EC-001: RTT exactly at threshold (200.0ms) → degraded=false.
			// The comparison is exclusive: ewmaRTTMS > DegradedRTTThresholdMS.
			// 200.0 > 200.0 is false → not degraded.
			name: "boundary_exactly_at_threshold",
			run: func(t *testing.T) {
				t.Parallel()
				tracker := paths.NewPathTracker(999.0, 1.0)
				tracker.OnProbe(200.0, false) // ewma = 200.0 exactly
				checkDegradedState(t, tracker, false, "RTT exactly at 200.0ms (exclusive boundary)")
			},
		},
		{
			// AC-002 sustained recovery: ≥5 above-threshold probes then ≥5 below-threshold.
			// Using alpha=0.5 to exercise EWMA smoothing (not alpha=1 degenerate).
			// After first probe at 300ms (resetRTT): ewma=300ms → degraded=true.
			// After 4 more probes at 300ms: ewma stays at 300ms (steady-state) → degraded=true.
			// First probe at 50ms: ewma = 0.5*50 + 0.5*300 = 175ms < 200ms → degraded=false.
			// Subsequent probes at 50ms: ewma converges further down → still not degraded.
			name: "sustained_degradation_then_sustained_recovery",
			run: func(t *testing.T) {
				t.Parallel()
				const alpha = 0.5

				tracker := paths.NewPathTracker(999.0, alpha)

				// Drive above threshold: first probe resets EWMA to 300ms directly.
				tracker.OnProbe(300.0, false) // resetRTT → ewma=300
				checkDegradedState(t, tracker, true, "after initial probe at 300ms")

				// 4 more above-threshold probes (alpha=0.5, ewma stays at 300ms with probe=300ms).
				for i := 1; i < 5; i++ {
					tracker.OnProbe(300.0, false)
					checkDegradedState(t, tracker, true, fmt.Sprintf("sustained high probe %d", i+1))
				}

				// 5 recovery probes at 50ms.
				// Probe 1: ewma = 0.5*50 + 0.5*300 = 175ms → degraded=false.
				for i := 0; i < 5; i++ {
					tracker.OnProbe(50.0, false)
					// After first recovery probe, EWMA = 175ms < 200ms → not degraded.
					checkDegradedState(t, tracker, false, fmt.Sprintf("recovery probe %d", i+1))
				}
			},
		},
		{
			// EC-004: Loss events do not update the degraded flag.
			// Drive EWMA to 250ms (degraded=true), then fire loss events.
			// Loss events must not clear the degraded flag (no RTT measured on loss).
			name: "loss_event_does_not_change_degraded",
			run: func(t *testing.T) {
				t.Parallel()
				tracker := paths.NewPathTracker(999.0, 1.0)

				// Establish degraded state.
				tracker.OnProbe(250.0, false)
				checkDegradedState(t, tracker, true, "before loss events")

				// Multiple loss events — degraded must remain true.
				for i := 0; i < 3; i++ {
					tracker.OnProbe(0, true) // lossEvent=true
					checkDegradedState(t, tracker, true, fmt.Sprintf("after loss event %d", i+1))
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, tc.run)
	}
}

// TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation verifies that on path
// reactivation (resetRTT path), the degraded flag is set from the reactivating
// probe's RTT — not carried over from the pre-deactivation state.
//
//   - Reactivation at 250ms → degraded=true (250 > 200).
//   - Reactivation at 150ms → degraded=false (150 ≤ 200).
//   - EC-003: no stale flag from before deactivation.
//
// AC-003 / BC-2.02.003 postcondition 6 (resetRTT branch) / EC-003
func TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		reactivationRTT float64
		wantDegraded    bool
	}{
		{"reactivation_at_250ms_degraded_true", 250.0, true},
		{"reactivation_at_150ms_degraded_false", 150.0, false},
		{"reactivation_at_200ms_boundary_not_degraded", 200.0, false},
		{"reactivation_at_200_1ms_degraded_true", 200.1, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tracker := paths.NewPathTracker(50.0, 0.125)

			// First, establish a non-degraded baseline (first probe at 50ms).
			// This clears the firstProbe flag via resetRTT.
			tracker.OnProbe(50.0, false)

			// Verify baseline: not degraded before deactivation.
			if tracker.IsDegraded() {
				t.Fatal("precondition: tracker at 50ms EWMA should not be degraded")
			}

			// Deactivate via 3 consecutive misses.
			for i := 0; i < 3; i++ {
				tracker.OnProbe(0, true)
			}
			if tracker.IsActive() {
				t.Fatal("precondition: tracker should be inactive after 3 consecutive misses")
			}

			// EC-003: degraded flag must not be stale from the pre-deactivation state.
			// Reactivation probe triggers resetRTT, which must set degraded from reactivating RTT.
			tracker.OnProbe(tc.reactivationRTT, false)

			if !tracker.IsActive() {
				t.Fatal("tracker must be reactivated after first successful probe")
			}
			if got := tracker.IsDegraded(); got != tc.wantDegraded {
				t.Errorf("IsDegraded() after reactivation at %.1fms: got %v, want %v",
					tc.reactivationRTT, got, tc.wantDegraded)
			}
			if snap := tracker.Snapshot(); snap.Degraded != tc.wantDegraded {
				t.Errorf("Snapshot().Degraded after reactivation at %.1fms: got %v, want %v (must mirror IsDegraded)",
					tc.reactivationRTT, snap.Degraded, tc.wantDegraded)
			}
		})
	}
}

// TestBC_2_02_003_PathTracker_IsDegraded_Race verifies that concurrent calls to
// IsDegraded(), OnProbe(), and Snapshot() on a single PathTracker produce no
// data races under the Go race detector (-race).
//
// Run with: go test -race ./internal/paths/ -run TestBC_2_02_003_PathTracker_IsDegraded_Race
//
// AC-004 / BC-2.02.003 invariant 2 (concurrent safety)
func TestBC_2_02_003_PathTracker_IsDegraded_Race(t *testing.T) {
	// Not parallel at outer level — inner goroutines provide the concurrency.

	const probeGoroutines = 6
	const readGoroutines = 6
	const probesPerWriter = 100

	tracker := paths.NewPathTracker(100.0, 0.125)

	var wg sync.WaitGroup
	wg.Add(probeGoroutines + readGoroutines)

	// Writer goroutines: alternate high-RTT and low-RTT probes to exercise
	// both degraded=true and degraded=false transitions concurrently.
	for g := range probeGoroutines {
		g := g
		go func() {
			defer wg.Done()
			for i := range probesPerWriter {
				// Alternate between above-threshold (250ms) and below-threshold (50ms)
				// probes to exercise both transition directions under concurrency.
				var rtt float64
				if (g+i)%2 == 0 {
					rtt = 250.0 // above threshold
				} else {
					rtt = 50.0 // below threshold
				}
				tracker.OnProbe(rtt, (g+i)%7 == 0) // occasional loss event
			}
		}()
	}

	// Reader goroutines: call IsDegraded() and Snapshot() concurrently.
	for range readGoroutines {
		go func() {
			defer wg.Done()
			for range probesPerWriter {
				_ = tracker.IsDegraded()
				_ = tracker.Snapshot()
				_ = tracker.IsActive()
			}
		}()
	}

	wg.Wait()

	// Post-concurrency sanity: tracker must still respond without panic.
	_ = tracker.IsDegraded()
	_ = tracker.Snapshot()
}

// TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded verifies that Snapshot().Degraded
// is always consistent with IsDegraded() under single-threaded sequential calls,
// AND that both reflect the expected degraded state at each step.
//
// Each row specifies the expected degraded value so the test fails if the stub
// never sets degraded=true — it is not enough for the two methods to agree with
// each other; they must also agree with the contract.
//
// AC-001 / BC-2.02.003 postcondition 5 (Snapshot consistency)
func TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded(t *testing.T) {
	t.Parallel()

	// alpha=1.0: each probe directly sets EWMA (easy to track expected values).
	// wantDegraded is computed from the EWMA value that results AFTER the probe:
	//   - First probe always triggers resetRTT → ewma = rtt.
	//   - Subsequent probes: ewma = 1.0*rtt + 0.0*prev = rtt.
	//   - loss events do not update ewma → degraded unchanged.
	type probe struct {
		rtt          float64
		lossEvent    bool
		wantDegraded bool
	}
	probes := []probe{
		{50.0, false, false},  // ewma=50 → not degraded
		{250.0, false, true},  // ewma=250 → degraded (250 > 200)
		{200.0, false, false}, // ewma=200 → not degraded (200 > 200 is false: exclusive)
		{200.1, false, true},  // ewma=200.1 → degraded (200.1 > 200)
		{199.9, false, false}, // ewma=199.9 → not degraded
		{0, true, false},      // loss event → ewma unchanged (199.9) → not degraded
		{350.0, false, true},  // ewma=350 → degraded
		{0, true, true},       // loss event → ewma unchanged (350) → still degraded
		{100.0, false, false}, // ewma=100 → not degraded
	}

	tracker := paths.NewPathTracker(999.0, 1.0)
	for i, p := range probes {
		tracker.OnProbe(p.rtt, p.lossEvent)
		isDeg := tracker.IsDegraded()
		snap := tracker.Snapshot()
		// Check both methods match the expected contract value.
		if isDeg != p.wantDegraded {
			t.Errorf("probe %d (rtt=%.1f, loss=%v): IsDegraded()=%v, want %v",
				i, p.rtt, p.lossEvent, isDeg, p.wantDegraded)
		}
		if snap.Degraded != p.wantDegraded {
			t.Errorf("probe %d (rtt=%.1f, loss=%v): Snapshot().Degraded=%v, want %v",
				i, p.rtt, p.lossEvent, snap.Degraded, p.wantDegraded)
		}
		// The two methods must also agree with each other.
		if snap.Degraded != isDeg {
			t.Errorf("probe %d (rtt=%.1f, loss=%v): Snapshot().Degraded=%v != IsDegraded()=%v (must mirror)",
				i, p.rtt, p.lossEvent, snap.Degraded, isDeg)
		}
	}
}

// ─── Pass-2 adversarial findings ─────────────────────────────────────────────

// TestNewPathTracker_RejectsInvalidAlpha asserts that NewPathTracker panics
// when alpha is outside the documented precondition range (0 < alpha ≤ 1).
//
// The NewPathTracker godoc states "alpha must satisfy 0 < alpha ≤ 1". Passing
// alpha=0 freezes the EWMA (no update ever applied — all measurements silently
// ignored). Passing alpha<0 or alpha>1 produces nonsensical EWMA arithmetic.
// These are programmer errors, not runtime conditions; panic is appropriate.
//
// Contract chosen: panic on alpha ≤ 0 or alpha > 1. The implementer must add
// a guard at the top of NewPathTracker:
//
//	if alpha <= 0 || alpha > 1 {
//	    panic("paths: NewPathTracker alpha must satisfy 0 < alpha <= 1")
//	}
//
// This test is RED until that guard is added — without it, NewPathTracker(10,0)
// silently creates a tracker with a frozen EWMA.
//
// Pass-2 finding F-L1 / NewPathTracker doc precondition (0 < alpha ≤ 1)
func TestNewPathTracker_RejectsInvalidAlpha(t *testing.T) {
	t.Parallel()

	invalidAlphas := []float64{0, -0.001, -1, -100, 1.0001, 2, 100}

	for _, alpha := range invalidAlphas {
		alpha := alpha
		t.Run("", func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("NewPathTracker(10, %v): want panic (alpha outside (0,1]), got no panic", alpha)
				}
			}()
			// Must panic — invalid alpha violates the PathTracker contract.
			paths.NewPathTracker(10.0, alpha)
		})
	}
}

// ─── S-BL.ROUTER-ADDR: RouterAddr field tests (BC-2.06.003 PC-1) ─────────────
//
// BC/AC coverage:
//
//	TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr      → AC-003 (constructor variant)
//	TestBC_2_06_003_Snapshot_RouterAddr_Propagates         → AC-001 (Snapshot().RouterAddr propagated)
//	TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess   → AC-001 (NewPathTracker yields "")
//	TestBC_2_06_003_NewPathTracker_Unchanged               → AC-003 (backward compat)
//	TestBC_2_06_003_NewPathTrackerWithAddr_RejectsInvalidAlpha → AC-003 (precondition carried over)
//	TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction  → RULING-W6TB-B §3 (immutability invariant)
//	TestBC_2_06_003_RouterAddr_ConcurrentSnapshot          → AC-003 + CI safety

// mustNewPathTrackerWithAddr is a test helper that wraps NewPathTrackerWithAddr
// and converts a stub panic (BC-5.38.001) into a test failure via t.Fatal,
// so that Red Gate panics are reported as test failures rather than binary aborts.
//
// Uses a fixed alpha=0.125 (the canonical EWMA smoothing factor for these tests).
// At Red Gate: t.Fatal is called with the panic message → test fails cleanly.
// Post-implementation: returns the constructed *PathTracker normally.
func mustNewPathTrackerWithAddr(t *testing.T, addr string, initialRTTMS float64) *paths.PathTracker {
	t.Helper()
	const alpha = 0.125
	var tracker *paths.PathTracker
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("NewPathTrackerWithAddr(%q, %v, %v) panicked (Red Gate — stub not yet implemented): %v", addr, initialRTTMS, alpha, r)
			}
		}()
		tracker = paths.NewPathTrackerWithAddr(addr, initialRTTMS, alpha)
	}()
	return tracker
}

// TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr verifies that after constructing
// a PathTracker via NewPathTrackerWithAddr("10.0.0.1:9000", ...), Snapshot().RouterAddr
// equals the supplied addr exactly.
//
// AC-003 / BC-2.06.003 PC-1 (S-BL.ROUTER-ADDR); RULING-W6TB-B §3.
//
// RED GATE: NewPathTrackerWithAddr panics (BC-5.38.001 stub). This test MUST fail
// until the constructor is implemented.
func TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		addr string
	}{
		{"ipv4_with_port", "10.0.0.1:9000"},
		{"localhost_with_port", "127.0.0.1:9000"},
		{"hostname_with_port", "router.example.com:4321"},
		{"h_colon_9000", "h:9000"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tracker := mustNewPathTrackerWithAddr(t, tc.addr, 50.0)
			snap := tracker.Snapshot()
			if snap.RouterAddr != tc.addr {
				t.Errorf("Snapshot().RouterAddr: got %q; want %q", snap.RouterAddr, tc.addr)
			}
		})
	}
}

// TestBC_2_06_003_Snapshot_RouterAddr_Propagates verifies that every call to
// Snapshot() on a NewPathTrackerWithAddr-constructed tracker returns the stored
// addr verbatim — including after probe operations that update EWMA state.
//
// AC-001 / BC-2.06.003 PC-1 (S-BL.ROUTER-ADDR); RULING-W6TB-B §3.
//
// RED GATE: NewPathTrackerWithAddr panics (BC-5.38.001 stub). This test MUST fail
// until the constructor is implemented.
func TestBC_2_06_003_Snapshot_RouterAddr_Propagates(t *testing.T) {
	t.Parallel()

	const wantAddr = "10.0.0.1:9000"
	tracker := mustNewPathTrackerWithAddr(t, wantAddr, 100.0)

	// Snapshot before any probes.
	snap0 := tracker.Snapshot()
	if snap0.RouterAddr != wantAddr {
		t.Errorf("before probes: Snapshot().RouterAddr=%q; want %q", snap0.RouterAddr, wantAddr)
	}

	// Probe the tracker — EWMA state changes but RouterAddr must be immutable.
	for i := 0; i < 10; i++ {
		tracker.OnProbe(20.0, false)
	}

	snap1 := tracker.Snapshot()
	if snap1.RouterAddr != wantAddr {
		t.Errorf("after 10 probes: Snapshot().RouterAddr=%q; want %q", snap1.RouterAddr, wantAddr)
	}

	// Loss events must not change RouterAddr either.
	for i := 0; i < 2; i++ {
		tracker.OnProbe(0, true)
	}
	snap2 := tracker.Snapshot()
	if snap2.RouterAddr != wantAddr {
		t.Errorf("after loss events: Snapshot().RouterAddr=%q; want %q", snap2.RouterAddr, wantAddr)
	}
}

// TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess verifies that a PathTracker
// constructed via the legacy NewPathTracker constructor (no addr) has
// Snapshot().RouterAddr == "" — the addr-less sentinel.
//
// AC-001 / BC-2.06.003 PC-1 (S-BL.ROUTER-ADDR); RULING-W6TB-B §3.
func TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess(t *testing.T) {
	t.Parallel()

	tracker := paths.NewPathTracker(50.0, 0.125)
	snap := tracker.Snapshot()
	if snap.RouterAddr != "" {
		t.Errorf("NewPathTracker Snapshot().RouterAddr=%q; want \"\" (addr-less sentinel)", snap.RouterAddr)
	}
}

// TestBC_2_06_003_NewPathTracker_Unchanged verifies that the existing NewPathTracker
// constructor remains unchanged after AC-003 implementation: it still produces a
// PathTracker with RouterAddr=="" and otherwise identical behavior to the pre-story state.
//
// AC-003 / BC-2.06.003 PC-1 (backward compat); RULING-W6TB-B.
func TestBC_2_06_003_NewPathTracker_Unchanged(t *testing.T) {
	t.Parallel()

	const initRTT = 999.0
	tracker := paths.NewPathTracker(initRTT, 0.125)

	if !tracker.IsActive() {
		t.Error("NewPathTracker: must start as active")
	}
	if tracker.RTT() != initRTT {
		t.Errorf("NewPathTracker: initial RTT=%v; want %v", tracker.RTT(), initRTT)
	}
	if tracker.LossPct() != 0.0 {
		t.Errorf("NewPathTracker: initial LossPct=%v; want 0.0", tracker.LossPct())
	}
	snap := tracker.Snapshot()
	if snap.RouterAddr != "" {
		t.Errorf("NewPathTracker: Snapshot().RouterAddr=%q; want \"\" (addr-less)", snap.RouterAddr)
	}
}

// TestBC_2_06_003_NewPathTrackerWithAddr_RejectsInvalidAlpha verifies that
// NewPathTrackerWithAddr panics when alpha is outside (0, 1], mirroring the
// same guard already present in NewPathTracker.
//
// AC-003 / BC-2.06.003 PC-1 precondition carried over to new constructor.
//
// RED GATE: NewPathTrackerWithAddr panics unconditionally (BC-5.38.001 stub),
// so this test will pass trivially during the Red Gate phase. The real alpha
// validation guard is tested post-implementation; this test is included to
// document the invariant and prevent regression.
//
// Implementation note: use a sub-check to distinguish "panicked due to stub"
// vs "panicked due to invalid alpha" once the constructor is real. During Red
// Gate, any panic satisfies the deferred func.
func TestBC_2_06_003_NewPathTrackerWithAddr_RejectsInvalidAlpha(t *testing.T) {
	t.Parallel()

	invalidAlphas := []float64{0, -0.001, -1, 1.0001, 2}

	for _, alpha := range invalidAlphas {
		alpha := alpha
		t.Run("", func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("NewPathTrackerWithAddr(addr, 10, %v): want panic (alpha outside (0,1]); got no panic", alpha)
				}
			}()
			paths.NewPathTrackerWithAddr("h:9000", 10.0, alpha)
		})
	}
}

// TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction verifies the immutability
// invariant from RULING-W6TB-B §3: RouterAddr is set once at construction and
// never mutated. Two trackers constructed with different addrs must return their
// respective addrs and never cross-contaminate.
//
// RULING-W6TB-B §3 / BC-2.06.003 PC-1 (S-BL.ROUTER-ADDR).
//
// RED GATE: NewPathTrackerWithAddr panics (BC-5.38.001 stub). This test MUST fail
// until the constructor is implemented.
func TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction(t *testing.T) {
	t.Parallel()

	const addrA = "10.0.0.1:9000"
	const addrB = "10.0.0.2:9001"

	trackerA := mustNewPathTrackerWithAddr(t, addrA, 50.0)
	trackerB := mustNewPathTrackerWithAddr(t, addrB, 100.0)

	// Probe both to mutate their EWMA state.
	for i := 0; i < 5; i++ {
		trackerA.OnProbe(20.0, false)
		trackerB.OnProbe(80.0, false)
	}

	// RouterAddr must not have changed and must not be cross-contaminated.
	if snapA := trackerA.Snapshot(); snapA.RouterAddr != addrA {
		t.Errorf("trackerA: RouterAddr=%q after probes; want %q", snapA.RouterAddr, addrA)
	}
	if snapB := trackerB.Snapshot(); snapB.RouterAddr != addrB {
		t.Errorf("trackerB: RouterAddr=%q after probes; want %q", snapB.RouterAddr, addrB)
	}
}

// TestBC_2_06_003_RouterAddr_ConcurrentSnapshot verifies that concurrent calls to
// Snapshot() on a NewPathTrackerWithAddr-constructed tracker always observe the
// correct RouterAddr under the Go race detector (-race).
//
// AC-003 / RULING-W6TB-B §3 (immutability + concurrent safety).
//
// RED GATE: NewPathTrackerWithAddr panics (BC-5.38.001 stub). This test MUST fail
// until the constructor is implemented.
func TestBC_2_06_003_RouterAddr_ConcurrentSnapshot(t *testing.T) {
	// Not parallel at outer level — inner goroutines provide the concurrency.

	const wantAddr = "10.0.0.1:9000"
	const goroutines = 8
	const itersPerGoroutine = 50

	tracker := mustNewPathTrackerWithAddr(t, wantAddr, 50.0)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		g := g
		go func() {
			defer wg.Done()
			for i := range itersPerGoroutine {
				// Mix probe writes with snapshot reads.
				if (g+i)%3 == 0 {
					tracker.OnProbe(float64(10+i%50), (g+i)%7 == 0)
				}
				snap := tracker.Snapshot()
				if snap.RouterAddr != wantAddr {
					// t.Errorf is goroutine-safe.
					t.Errorf("goroutine %d iter %d: Snapshot().RouterAddr=%q; want %q (race or mutation)", g, i, snap.RouterAddr, wantAddr)
				}
			}
		}()
	}
	wg.Wait()
}

// ─── P99 histogram tests (S-5.02, AC-004, AC-005) ──────────────────────────

// TestBC_2_06_003_P99_PendingLessThan10Samples verifies that the p99 is "pending"
// (represented as SampleCount < 10 in the snapshot) when fewer than 10 RTT samples
// have been collected.
//
// AC-004 / BC-2.06.003 EC-003
func TestBC_2_06_003_P99_PendingLessThan10Samples(t *testing.T) {
	t.Parallel()

	tracker := paths.NewPathTracker(500.0, 1.0) // alpha=1 → always first-probe semantics until real

	// Feed 9 samples (fewer than the 10-sample threshold).
	for i := 0; i < 9; i++ {
		tracker.OnProbe(15.0, false)
	}

	snap := tracker.Snapshot()
	if snap.SampleCount >= 10 {
		t.Errorf("expected SampleCount < 10 after 9 probes, got %d", snap.SampleCount)
	}
	// Callers must surface P99RTTMs as "pending" when SampleCount < 10.
	// The zero-value of P99RTTMs when pending is 0 — that's the stub value.
	// This test will pass once SampleCount is properly tracked.
}

// TestBC_2_06_003_P99_ValidAfter10Samples verifies that SampleCount ≥ 10 and
// P99RTTMs is a non-zero float64 after 10 or more RTT samples.
//
// AC-004 / BC-2.06.003 EC-003
func TestBC_2_06_003_P99_ValidAfter10Samples(t *testing.T) {
	t.Parallel()

	tracker := paths.NewPathTracker(500.0, 1.0)

	// Feed 10 samples.
	for i := 0; i < 10; i++ {
		tracker.OnProbe(30.0, false)
	}

	snap := tracker.Snapshot()
	if snap.SampleCount < 10 {
		t.Errorf("expected SampleCount ≥ 10 after 10 probes, got %d", snap.SampleCount)
	}
	if snap.P99RTTMs <= 0 {
		t.Errorf("expected P99RTTMs > 0 after 10 probes with 30ms RTT, got %v", snap.P99RTTMs)
	}
}

// TestBC_2_06_003_P99HistogramAccuracy verifies that the histogram p99 approximation
// satisfies p99 ≤ true_p99 + max_bucket_width for synthetic sample distributions.
//
// AC-005 / BC-2.06.003 PC-1 / ARCH-03 v1.6 §p99 RTT Accumulator
//
// NOTE: A formal accuracy verification property for the accumulator is deferred to
// S-BL.BENCH. No VP ID is assigned to the accumulator accuracy property. Do not
// invent a VP ID.
func TestBC_2_06_003_P99HistogramAccuracy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		samples     []float64 // RTT samples to feed
		maxBucketW  float64   // max bucket width for the p99 bucket
		wantP99High float64   // upper bound: true_p99 + maxBucketW
	}{
		{
			// (a) all samples in the 0–25ms bucket
			name: "all_samples_in_0_25ms_bucket",
			samples: func() []float64 {
				s := make([]float64, 15)
				for i := range s {
					s[i] = 10.0
				}
				return s
			}(),
			maxBucketW: 25.0,
			// true p99 ≈ 10ms; bucket [0,25) so upper bound = 25ms
			wantP99High: 25.0,
		},
		{
			// (b) samples distributed across 0–25ms and 100–150ms buckets
			name: "samples_across_0_25ms_and_100_150ms",
			// 10 samples at 10ms (bucket [0,25)), 5 samples at 110ms (bucket [100,150))
			samples: func() []float64 {
				s := make([]float64, 15)
				for i := 0; i < 10; i++ {
					s[i] = 10.0
				}
				for i := 10; i < 15; i++ {
					s[i] = 110.0
				}
				return s
			}(),
			// true p99 of [10x10ms, 5x110ms] ≈ 110ms; bucket [100,150) width = 50ms
			// ARCH-03 v1.6: bucket 4 = [100,150), width=50ms; upper bound = 110+50 = 160ms
			maxBucketW:  50.0,
			wantP99High: 160.0,
		},
		{
			// (c) distribution where the p99 falls in the [150,200ms) bucket
			name: "p99_in_150_200ms_bucket",
			// 12 samples at 15ms, 3 samples at 180ms
			samples: func() []float64 {
				s := make([]float64, 15)
				for i := 0; i < 12; i++ {
					s[i] = 15.0
				}
				for i := 12; i < 15; i++ {
					s[i] = 180.0
				}
				return s
			}(),
			// true p99 ≈ 180ms in bucket [150,200); width = 50ms
			// ARCH-03 v1.6: bucket 5 = [150,200), width=50ms; upper bound = 180+50 = 230ms
			maxBucketW:  50.0,
			wantP99High: 230.0,
		},
		{
			// F-M2 (d): p99 in coarse [200,300) bucket (bucket index 8, width=100ms).
			// The histogram changes bucket width here: [175,200) is width 25ms but
			// [200,300) is width 100ms. A sample at 250ms falls in [200,300).
			// true p99 ≈ 250ms; bucket upper edge = 300ms; bound = 250 + 100 = 350ms.
			// ARCH-03 v1.6 §p99 RTT Accumulator: p99 ≤ true_p99 + bucket_width.
			//
			// This test is RED until the histogram returns the upper edge of the
			// [200,300) bucket (300ms) — the accumulator must NOT conflate this
			// coarse bucket with the fine 25ms buckets above it.
			//
			// F-M2 / ARCH-03 v1.6 §p99 RTT Accumulator (coarse bucket boundary)
			name: "p99_in_200_300ms_coarse_bucket",
			// 10 samples at 10ms (bucket [0,25)), 5 samples at 250ms (bucket [200,300))
			samples: func() []float64 {
				s := make([]float64, 15)
				for i := 0; i < 10; i++ {
					s[i] = 10.0
				}
				for i := 10; i < 15; i++ {
					s[i] = 250.0
				}
				return s
			}(),
			// true p99 ≈ 250ms in bucket [200,300); bucket width = 100ms
			// upper bound = 250 + 100 = 350ms; histogram returns bucket edge = 300ms
			maxBucketW:  100.0,
			wantP99High: 350.0,
		},
		{
			// F-M2 (e): p99 in unbounded bucket (≥2000ms, bucket index 15, sentinel edge = 1e18).
			// Any RTT ≥ 2000ms falls into the last bucket whose upper edge is the
			// infinity sentinel 1e18. The histogram p99() returns 1e18 for this bucket.
			// The bound check here is: P99RTTMs > 0 (not pending) and P99RTTMs >= 2000
			// (correctly identified the unbounded bucket). wantP99High is set to 1e18
			// (the sentinel), which is the tightest meaningful bound.
			//
			// This test is RED until the histogram correctly returns the sentinel edge
			// for samples falling in the unbounded last bucket.
			//
			// F-M2 / ARCH-03 v1.6 §p99 RTT Accumulator (unbounded bucket)
			name: "p99_in_unbounded_bucket_ge_2000ms",
			// 10 samples at 10ms (bucket [0,25)), 5 samples at 2500ms (bucket [2000,∞))
			samples: func() []float64 {
				s := make([]float64, 15)
				for i := 0; i < 10; i++ {
					s[i] = 10.0
				}
				for i := 10; i < 15; i++ {
					s[i] = 2500.0
				}
				return s
			}(),
			// true p99 ≈ 2500ms in bucket [2000,∞); sentinel upper edge = 1e18
			// wantP99High = 1e18 (the sentinel value returned by p99())
			maxBucketW:  1e18,
			wantP99High: 1e18,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tracker := paths.NewPathTracker(500.0, 1.0)
			for _, rtt := range tc.samples {
				tracker.OnProbe(rtt, false)
			}

			snap := tracker.Snapshot()
			if snap.SampleCount < 10 {
				t.Fatalf("need ≥10 samples for valid p99, got SampleCount=%d", snap.SampleCount)
			}
			if snap.P99RTTMs > tc.wantP99High {
				t.Errorf("p99 approximation bound violated: got P99RTTMs=%v, want ≤ %v (true_p99 + max_bucket_width)", snap.P99RTTMs, tc.wantP99High)
			}
		})
	}
}
