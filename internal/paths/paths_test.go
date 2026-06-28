// Package paths_test contains the TDD test suite for BC-2.02.003
// (per-path RTT/loss tracking and path ranking).
//
// All tests MUST fail until PathScore, NewPathTracker, PathTracker.OnProbe,
// PathTracker.Score, PathTracker.IsActive, PathTracker.RTT, PathTracker.LossPct,
// and Rank are implemented (Red Gate).
//
// BC/AC coverage map:
//
//	TestBC_2_02_003_PathScore_LowerRTTLowerScore          → AC-001, BC-2.02.003 postcondition 3
//	TestBC_2_02_003_PathScore_HigherLossRaisesScore        → AC-001, BC-2.02.003 postcondition 3
//	TestBC_2_02_003_PathScore_Transitive                   → AC-001, VP-026
//	TestBC_2_02_003_PathScore_ZeroLossPureRTT              → AC-001, BC-2.02.003 postcondition 1
//	TestBC_2_02_003_PathScore_Formula                      → AC-001, ARCH-03 formula
//	TestBC_2_02_003_PathTracker_NewInitialRTT              → BC-2.02.003 precondition 3
//	TestBC_2_02_003_PathTracker_EWMAConvergence            → AC-002, BC-2.02.003 postcondition 1
//	TestBC_2_02_003_PathTracker_LossUpdatesEWMA            → BC-2.02.003 postcondition 2
//	TestBC_2_02_003_PathTracker_InactiveAfterMisses        → BC-2.02.003 postcondition 6, VP-026/VP-040
//	TestBC_2_02_003_PathTracker_ResetMissesOnSuccess       → BC-2.02.003 postcondition 6
//	TestBC_2_02_003_PathTracker_ScoreDelegates             → AC-001/AC-002
//	TestBC_2_02_003_Rank_OrderedByScore                    → BC-2.02.003 postcondition 3
//	TestBC_2_02_003_Rank_ExcludesInactivePaths             → BC-2.02.003 postcondition 6
//	TestBC_2_02_003_Rank_ErrNoActivePaths                  → BC-2.02.001 precondition 1 / ErrNoActivePaths
//	TestBC_2_02_003_Rank_TiebreakByID                      → EC-002, BC-2.02.001 invariant 3
//	TestBC_2_02_003_Rank_SinglePath                        → EC-001, BC-2.02.001 postcondition 3
//	TestBC_2_02_003_PathTracker_RTTAndLossPctAccessors     → API surface
//	TestBC_2_02_003_PathScore_PropertyTransitive_Manual    → VP-026 (stdlib property sweep)
package paths_test

import (
	"errors"
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

// TestBC_2_02_003_PathTracker_EWMAConvergence verifies that after 3+ successful
// probe arrivals the EWMA RTT converges toward the measured RTT.
//
// AC-002 / BC-2.02.003 postcondition 1
func TestBC_2_02_003_PathTracker_EWMAConvergence(t *testing.T) {
	t.Parallel()

	// Start with a high conservative RTT; feed 10 probes at 20ms.
	// After 10 probes with alpha=0.5, EWMA should be within 5ms of 20ms.
	const initRTT = 1000.0
	const targetRTT = 20.0
	const alpha = 0.5
	const probes = 10

	tracker := paths.NewPathTracker(initRTT, alpha)

	for i := 0; i < probes; i++ {
		tracker.OnProbe(targetRTT, false)
	}

	got := tracker.RTT()
	// After 10 probes at alpha=0.5 from 1000ms initial, convergence is rapid.
	// Exact bound: 1000 * (1-0.5)^10 + 20*(1-(1-0.5)^10) ≈ 20.97ms
	if got < 15.0 || got > 25.0 {
		t.Errorf("EWMA after %d probes (alpha=%v, target=%vms): got %vms, want ~%vms (within ±5ms)", probes, alpha, targetRTT, got, targetRTT)
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
