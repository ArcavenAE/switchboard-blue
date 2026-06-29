package metrics_test

// Unit tests for internal/metrics (QualityIndicator).
//
// Coverage map:
//
//	TestQualityIndicator_ThresholdClassification — AC-001, BC-2.06.001 PC-2/PC-3/PC-4
//	TestQualityIndicator_ThresholdBoundary         — AC-001, BC-2.06.001 PC-2/PC-3/PC-4 (exact boundary values)
//	TestQualityIndicator_HysteresisUpgrade         — AC-002, BC-2.06.001 invariant 3
//	TestQualityIndicator_RedToGreenViaSixMeasurements — AC-002, BC-2.06.001 invariant 3 (CR-001: full recovery path)
//	TestQualityIndicator_SingleGoodMeasurementNoUpgrade — AC-002, EC-001 (story edge-case catalog)
//	TestQualityIndicator_HysteresisResetOnBadMeasurement — AC-002, BC-2.06.001 invariant 3 (streak resets)
//	TestQualityIndicator_MissingFrameDowngradeGreenToYellow — AC-003, BC-2.06.002 PC-2, VP-052
//	TestQualityIndicator_MissingFrameDowngradeYellowToRed   — AC-003, BC-2.06.002 PC-2, VP-052
//	TestQualityIndicator_MissingFrameSubthresholdNoDowngrade — AC-003, BC-2.06.002 PC-1 (N-1 misses insufficient)
//	TestQualityIndicator_MissingFrameCounterResetOnGoodUpdate — BC-2.06.002 PC-4
//	TestQualityIndicator_MissingFrameCounterResetByYellowUpdate — BC-2.06.002 PC-4 (CR-002: Yellow-range also resets)
//	TestQualityIndicator_DegradationNeverSkipsLevel — AC-004, BC-2.06.002 PC-2, VP-027
//	TestQualityIndicator_RecoveryNeverSkipsLevel    — AC-004, VP-027
//	TestQualityIndicator_DowngradeIsImmediate       — AC-004, BC-2.06.001 (no hysteresis on downgrade)
//	TestQualityIndicator_String                     — sanity / human-readable labels
//	TestQualityIndicator_ConcurrentUpdates          — race-detector coverage (sync.Mutex)

import (
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/metrics"
)

// assertLevel is a test helper that calls t.Errorf when got != want.
func assertLevel(t *testing.T, got, want metrics.Quality, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %s, want %s", msg, got, want)
	}
}

// driveToRed drives qi to Red by feeding HysteresisCount+1 extreme-bad measurements.
// downgrade is immediate, so a single Red-range call is actually sufficient, but we
// drive several so the test is not sensitive to that implementation detail.
func driveToRed(t *testing.T, qi *metrics.QualityIndicator) {
	t.Helper()
	// Downgrade is immediate per AC-004 / BC-2.06.001; but to be robust against
	// any implementation that computes current from a rolling window we drive
	// HysteresisCount red-range measurements.
	for i := 0; i < metrics.HysteresisCount; i++ {
		qi.Update(600, 25) // clearly Red: RTT > 500 AND loss > 20
	}
	if qi.Current() != metrics.Red {
		t.Fatalf("driveToRed: expected Red after %d bad updates, got %s",
			metrics.HysteresisCount, qi.Current())
	}
}

// driveToYellow drives qi (starting from Green) to Yellow by feeding
// HysteresisCount yellow-range measurements (downgrade is immediate).
func driveToYellow(t *testing.T, qi *metrics.QualityIndicator) {
	t.Helper()
	for i := 0; i < metrics.HysteresisCount; i++ {
		qi.Update(200, 0) // RTT in (100,500] — Yellow
	}
	if qi.Current() != metrics.Yellow {
		t.Fatalf("driveToYellow: expected Yellow after %d updates, got %s",
			metrics.HysteresisCount, qi.Current())
	}
}

// driveGreenBaseline establishes a stable Green state with HysteresisCount updates.
// From Green zero-value, this confirms the indicator starts as expected.
func driveGreenBaseline(t *testing.T, qi *metrics.QualityIndicator) {
	t.Helper()
	for i := 0; i < metrics.HysteresisCount; i++ {
		qi.Update(50, 1) // comfortably Green
	}
	if qi.Current() != metrics.Green {
		t.Fatalf("driveGreenBaseline: expected Green, got %s", qi.Current())
	}
}

// -----------------------------------------------------------------------
// AC-001 / BC-2.06.001 PC-2, PC-3, PC-4
// -----------------------------------------------------------------------

// TestQualityIndicator_ThresholdClassification verifies the three-band
// classification after the indicator has stabilised (n consecutive measurements
// of the same band, where downgrade is immediate and upgrade needs hysteresis).
//
// BC-2.06.001 PC-2 — Green: RTT ≤ 100ms AND loss ≤ 5%
// BC-2.06.001 PC-3 — Yellow: RTT ≤ 500ms AND loss ≤ 20% (and not green)
// BC-2.06.001 PC-4 — Red: RTT > 500ms OR loss > 20%
// AC-001
func TestQualityIndicator_ThresholdClassification(t *testing.T) {
	t.Parallel()

	// n: local copy avoids SA4008 (loop-bound-never-changes) from static analysis.
	n := metrics.HysteresisCount

	tests := []struct {
		name    string
		rttMs   float64
		lossPct float64
		want    metrics.Quality
	}{
		// Green region — BC-2.06.001 PC-2
		{"green: well inside budget", 15, 0, metrics.Green},
		{"green: RTT at boundary 100ms", metrics.GreenRTTMs, 0, metrics.Green},
		{"green: loss at boundary 5%", 50, metrics.GreenLossPct, metrics.Green},
		{"green: both at boundary", metrics.GreenRTTMs, metrics.GreenLossPct, metrics.Green},

		// Yellow region — BC-2.06.001 PC-3
		{"yellow: RTT just above green (101ms)", 101, 0, metrics.Yellow},
		{"yellow: RTT at yellow boundary (500ms)", metrics.YellowRTTMs, 0, metrics.Yellow},
		{"yellow: loss just above green (6%)", 50, 6, metrics.Yellow},
		{"yellow: loss at yellow boundary (20%)", 50, metrics.YellowLossPct, metrics.Yellow},
		{"yellow: RTT in middle of yellow range", 300, 0, metrics.Yellow},

		// Red region — BC-2.06.001 PC-4
		{"red: RTT just above yellow (501ms)", 501, 0, metrics.Red},
		{"red: loss just above yellow (21%)", 50, 21, metrics.Red},
		{"red: both above yellow", 600, 25, metrics.Red},
		{"red: extreme RTT", 9999, 0, metrics.Red},
		{"red: extreme loss", 50, 100, metrics.Red},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			qi := metrics.NewQualityIndicator()

			// Feed n measurements so the indicator can settle.
			// Downgrade is immediate; for Green the zero value starts there.
			for i := 0; i < n; i++ {
				qi.Update(tc.rttMs, tc.lossPct)
			}

			got := qi.Current()
			if got != tc.want {
				t.Errorf("RTT=%.0f loss=%.0f after %d updates: got %s, want %s",
					tc.rttMs, tc.lossPct, n, got, tc.want)
			}
		})
	}
}

// TestQualityIndicator_ThresholdBoundary exercises the exact boundary values
// using canonical test vectors from BC-2.06.001.
//
// BC-2.06.001 canonical test vectors
// AC-001
func TestQualityIndicator_ThresholdBoundary(t *testing.T) {
	t.Parallel()

	n := metrics.HysteresisCount

	// Canonical test vectors from BC-2.06.001 (table at end of contract).
	vectors := []struct {
		name    string
		rttMs   float64
		lossPct float64
		want    metrics.Quality
	}{
		// "10 probes: RTT=15ms, loss=0% → Green"
		{"canonical: RTT=15ms loss=0%", 15, 0, metrics.Green},
		// "10 probes: RTT=150ms, loss=3% → Yellow (RTT in yellow range)"
		{"canonical: RTT=150ms loss=3%", 150, 3, metrics.Yellow},
	}

	for _, tv := range vectors {
		t.Run(tv.name, func(t *testing.T) {
			t.Parallel()
			qi := metrics.NewQualityIndicator()
			for i := 0; i < n; i++ {
				qi.Update(tv.rttMs, tv.lossPct)
			}
			assertLevel(t, qi.Current(), tv.want, tv.name)
		})
	}
}

// -----------------------------------------------------------------------
// AC-002 / BC-2.06.001 invariant 3 — hysteresis on upgrade
// -----------------------------------------------------------------------

// TestQualityIndicator_HysteresisUpgrade verifies that upgrade from Red to Yellow
// and from Yellow to Green each require exactly HysteresisCount consecutive
// measurements in the target band.
//
// BC-2.06.001 invariant 3 — "3-consecutive-measurement hysteresis"
// AC-002
func TestQualityIndicator_HysteresisUpgrade(t *testing.T) {
	t.Parallel()

	n := metrics.HysteresisCount

	t.Run("Red requires n consecutive Yellow-range measurements to upgrade", func(t *testing.T) {
		t.Parallel()
		qi := metrics.NewQualityIndicator()
		driveToRed(t, qi)

		// Fewer than n consecutive Yellow-range measurements must not upgrade.
		for i := 0; i < n-1; i++ {
			qi.Update(200, 0) // yellow range
			if qi.Current() == metrics.Yellow {
				t.Errorf("upgrade to Yellow after only %d good measurement(s); need %d", i+1, n)
			}
		}

		// The n-th consecutive Yellow-range measurement must upgrade.
		qi.Update(200, 0)
		assertLevel(t, qi.Current(), metrics.Yellow, "after n consecutive yellow-range measurements")
	})

	t.Run("Yellow requires n consecutive Green-range measurements to upgrade", func(t *testing.T) {
		t.Parallel()
		qi := metrics.NewQualityIndicator()
		driveToYellow(t, qi)

		// Fewer than n consecutive Green-range measurements must not upgrade.
		for i := 0; i < n-1; i++ {
			qi.Update(50, 1) // green range
			if qi.Current() == metrics.Green {
				t.Errorf("upgrade to Green after only %d good measurement(s); need %d", i+1, n)
			}
		}

		// The n-th consecutive Green-range measurement must upgrade.
		qi.Update(50, 1)
		assertLevel(t, qi.Current(), metrics.Green, "after n consecutive green-range measurements")
	})
}

// TestQualityIndicator_RedToGreenViaSixMeasurements documents the full Red→Green
// recovery path using exactly 6 consecutive Green-range measurements (2×HysteresisCount).
//
// The indicator starts Red. Each group of HysteresisCount consecutive Green-range
// measurements upgrades exactly one level: calls 1–3 bring it to Yellow; calls 4–6
// bring it to Green. Intermediate calls 4 and 5 must still be Yellow because the
// streak resets when the level changes — the new window counts toward Yellow→Green.
//
// This test intentionally documents that the streak-resets-on-level-change
// behaviour is required: the 3-call window is relative to the current level, not
// accumulated globally.
//
// BC-2.06.001 invariant 3 — upgrade requires HysteresisCount consecutive measurements
// AC-002
func TestQualityIndicator_RedToGreenViaSixMeasurements(t *testing.T) {
	t.Parallel()
	qi := metrics.NewQualityIndicator()
	driveToRed(t, qi)

	// Calls 1–2: streak building toward Yellow (n-1 = 2 calls, must stay Red).
	qi.Update(50, 1)
	assertLevel(t, qi.Current(), metrics.Red, "after 1 green-range measurement from Red")
	qi.Update(50, 1)
	assertLevel(t, qi.Current(), metrics.Red, "after 2 green-range measurements from Red")

	// Call 3: completes the first HysteresisCount window → Red upgrades to Yellow.
	qi.Update(50, 1)
	assertLevel(t, qi.Current(), metrics.Yellow, "after 3 green-range measurements from Red: must reach Yellow")

	// Calls 4–5: streak building toward Green (new window; must stay Yellow).
	qi.Update(50, 1)
	assertLevel(t, qi.Current(), metrics.Yellow, "after 4 green-range measurements: still Yellow (streak reset at level change)")
	qi.Update(50, 1)
	assertLevel(t, qi.Current(), metrics.Yellow, "after 5 green-range measurements: still Yellow")

	// Call 6: completes the second HysteresisCount window → Yellow upgrades to Green.
	qi.Update(50, 1)
	assertLevel(t, qi.Current(), metrics.Green, "after 6 green-range measurements: must reach Green")
}

// TestQualityIndicator_SingleGoodMeasurementNoUpgrade verifies EC-001 from the
// story edge-case catalog: a single good measurement after Red does not upgrade.
//
// Story EC-001: "Single measurement at Green threshold after Red state → stays Red"
// BC-2.06.001 invariant 3
// AC-002
func TestQualityIndicator_SingleGoodMeasurementNoUpgrade(t *testing.T) {
	t.Parallel()
	qi := metrics.NewQualityIndicator()
	driveToRed(t, qi)

	qi.Update(metrics.GreenRTTMs, metrics.GreenLossPct) // boundary Green
	if qi.Current() != metrics.Red {
		t.Errorf("single green-boundary measurement must not upgrade from Red: got %s", qi.Current())
	}
}

// TestQualityIndicator_HysteresisResetOnBadMeasurement verifies that a bad
// measurement interrupting an upgrade streak resets the consecutive counter so
// that the full HysteresisCount is required again.
//
// BC-2.06.001 invariant 3 (streak must be consecutive)
// AC-002
func TestQualityIndicator_HysteresisResetOnBadMeasurement(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveToRed(t, qi)

	// Feed n-1 green measurements (streak in progress).
	for i := 0; i < n-1; i++ {
		qi.Update(50, 1) // green range
	}

	// Interrupt the streak with one bad measurement.
	qi.Update(200, 0) // yellow range — breaks the Green streak

	// Now feed n-1 more green measurements; we still must not be at Green
	// because the streak was reset.
	for i := 0; i < n-1; i++ {
		qi.Update(50, 1)
	}
	if qi.Current() == metrics.Green {
		t.Errorf("streak interrupted at %d measurements; must not reach Green until %d *consecutive* good measurements",
			n-1, n)
	}

	// Complete the n-th consecutive Green measurement after reset.
	// Trace: driveToRed→Red; 2×Green (streak=2, candidate=Green); Yellow (candidate
	// switches to Yellow, streak=1, still Red); 2×Green (candidate switches back to
	// Green, streak=2, still Red); final Green (streak=3 ≥ n) → upgrades one level
	// to Yellow. Current must be exactly Yellow at this point.
	qi.Update(50, 1)
	assertLevel(t, qi.Current(), metrics.Yellow,
		"after interrupted streak + fresh n consecutive green measurements from Red, must reach Yellow (one level at a time)")
}

// -----------------------------------------------------------------------
// AC-003 / BC-2.06.002 PC-2 — OnMissingFrame downgrade
// -----------------------------------------------------------------------

// TestQualityIndicator_MissingFrameDowngradeGreenToYellow verifies the canonical
// test vector: 3 consecutive missing frames degrade Green to Yellow.
//
// BC-2.06.002 PC-2 — "After N consecutive gap events (N=3), degrade one level"
// BC-2.06.002 canonical test vector: "3 consecutive frames missing → degrade one level"
// VP-052 — N=3 consecutive gaps trigger indicator downgrade
// AC-003
func TestQualityIndicator_MissingFrameDowngradeGreenToYellow(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveGreenBaseline(t, qi)

	// Fewer than n consecutive missing frames must not downgrade.
	for i := 0; i < n-1; i++ {
		qi.OnMissingFrame()
		if qi.Current() != metrics.Green {
			t.Errorf("after %d missing frame(s) (below threshold %d): got %s, want Green",
				i+1, n, qi.Current())
		}
	}

	// The n-th consecutive missing frame must downgrade Green → Yellow.
	qi.OnMissingFrame()
	assertLevel(t, qi.Current(), metrics.Yellow, "after 3 consecutive missing frames from Green")
}

// TestQualityIndicator_MissingFrameDowngradeYellowToRed verifies that 3 more
// consecutive missing frames from Yellow downgrade to Red (one step at a time).
//
// BC-2.06.002 PC-2 — one-level downgrade per N consecutive gaps
// VP-052 — Yellow state: N=3 gaps → Red
// AC-003
func TestQualityIndicator_MissingFrameDowngradeYellowToRed(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveGreenBaseline(t, qi)

	// Downgrade Green → Yellow via n missing frames.
	for i := 0; i < n; i++ {
		qi.OnMissingFrame()
	}
	if qi.Current() != metrics.Yellow {
		t.Fatalf("prerequisite: expected Yellow after %d missing frames, got %s", n, qi.Current())
	}

	// Now downgrade Yellow → Red via n more consecutive missing frames.
	// Fewer than n must not downgrade.
	for i := 0; i < n-1; i++ {
		qi.OnMissingFrame()
		if qi.Current() != metrics.Yellow {
			t.Errorf("after %d more missing frame(s): got %s, want Yellow", i+1, qi.Current())
		}
	}

	// The n-th consecutive missing frame must downgrade Yellow → Red.
	qi.OnMissingFrame()
	assertLevel(t, qi.Current(), metrics.Red, "after 3 more consecutive missing frames from Yellow")
}

// TestQualityIndicator_MissingFrameSubthresholdNoDowngrade verifies story
// EC-003: first degradation signal — indicator stays Green until 3 consecutive
// missing frames.
//
// Story EC-003: "Missing frame during Green state → stays Green until 3 consecutive"
// BC-2.06.002 PC-1 — gap event recorded
// AC-003
func TestQualityIndicator_MissingFrameSubthresholdNoDowngrade(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()

	// Establish Green.
	for i := 0; i < n; i++ {
		qi.Update(50, 1)
	}

	// Feed n-1 missing frames; indicator must stay Green.
	for i := 0; i < n-1; i++ {
		qi.OnMissingFrame()
	}
	assertLevel(t, qi.Current(), metrics.Green,
		"n-1 missing frames must not downgrade from Green")
}

// TestQualityIndicator_MissingFrameCounterResetOnGoodUpdate verifies that a
// successful Update resets the missing-frame counter, so the next downgrade
// requires a fresh streak of n consecutive missing frames.
//
// BC-2.06.002 PC-4 — "When frames resume, gap count decreases"
// BC-2.06.002 canonical test vector: "3 frames missing then 3 frames received → recovers"
func TestQualityIndicator_MissingFrameCounterResetOnGoodUpdate(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveGreenBaseline(t, qi)

	// Feed n-1 missing frames (streak in progress but below threshold).
	for i := 0; i < n-1; i++ {
		qi.OnMissingFrame()
	}

	// A good Update must reset the missing-frame counter.
	qi.Update(50, 1) // green measurement
	if qi.Current() != metrics.Green {
		t.Fatalf("after good update: expected Green, got %s", qi.Current())
	}

	// Feed n-1 more missing frames — must still not downgrade (streak was reset).
	for i := 0; i < n-1; i++ {
		qi.OnMissingFrame()
	}
	assertLevel(t, qi.Current(), metrics.Green,
		"n-1 missing frames after streak reset must not downgrade")
}

// TestQualityIndicator_MissingFrameCounterResetByYellowUpdate verifies that a
// Yellow-range Update also resets the missing-frame counter, not just Green-range
// Updates. Any received frame — regardless of measured quality level — breaks the
// consecutive-gap streak (BC-2.06.002 PC-4).
//
// Sequence: 2 missing frames, then Update(200,0) (Yellow-range, resets counter and
// downgrades Green→Yellow immediately), then 1 more missing frame. The single
// post-reset gap must not trigger a further downgrade to Red.
//
// BC-2.06.002 PC-4 — any Update resets the missing-frame counter
func TestQualityIndicator_MissingFrameCounterResetByYellowUpdate(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveGreenBaseline(t, qi)

	// Accumulate n-1 missing frames (below downgrade threshold).
	for i := 0; i < n-1; i++ {
		qi.OnMissingFrame()
	}
	assertLevel(t, qi.Current(), metrics.Green,
		"n-1 missing frames must not downgrade")

	// A Yellow-range Update resets missingFrameCount=0 and downgrades Green→Yellow immediately.
	qi.Update(200, 0)
	assertLevel(t, qi.Current(), metrics.Yellow,
		"Yellow-range update from Green must downgrade immediately to Yellow")

	// One more missing frame after the reset: count=1, below threshold.
	// Must NOT downgrade further to Red.
	qi.OnMissingFrame()
	assertLevel(t, qi.Current(), metrics.Yellow,
		"single missing frame after counter reset must not trigger another downgrade")
}

// -----------------------------------------------------------------------
// AC-004 / BC-2.06.002 PC-2, VP-027 — level-by-level transitions
// -----------------------------------------------------------------------

// TestQualityIndicator_DegradationNeverSkipsLevel verifies the deterministic
// downgrade path Green → Yellow → Red.  Every intermediate state must be
// observed; Red must never be the first level after Green.
//
// BC-2.06.002 PC-2 — "degrades one level (green→yellow or yellow→red)"
// VP-027 — "no single-step red→green transition; transitions only monotonically downward"
// AC-004
func TestQualityIndicator_DegradationNeverSkipsLevel(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveGreenBaseline(t, qi)

	// Feed Red-range measurements one at a time; record every observed level.
	// We need to see Green, then Yellow, then Red — never a Green→Red skip.
	levelSequence := []metrics.Quality{qi.Current()}
	// Drive enough updates to fully traverse the state machine.
	// Worst case: HysteresisCount updates to exit Green then HysteresisCount
	// more to exit Yellow.  We overshoot deliberately.
	for i := 0; i < 2*n+5; i++ {
		qi.Update(600, 25) // Red-range
		levelSequence = append(levelSequence, qi.Current())
	}

	// Validate the observed sequence:
	// 1. First level must be Green (from our driveGreenBaseline).
	if levelSequence[0] != metrics.Green {
		t.Fatalf("sequence[0]: expected Green, got %s", levelSequence[0])
	}

	// 2. Transitions may only go Green→Yellow or Yellow→Red (never skip).
	for i := 1; i < len(levelSequence); i++ {
		prev := levelSequence[i-1]
		cur := levelSequence[i]
		if prev == metrics.Green && cur == metrics.Red {
			t.Errorf("position %d: illegal skip transition Green→Red", i)
		}
		// Level may only stay the same or go down; it must never go up during
		// a sustained Red-range workload.
		if int(cur) < int(prev) {
			// cur < prev means cur is numerically smaller = "better" quality.
			// Green=0, Yellow=1, Red=2: smaller = better.
			t.Errorf("position %d: indicator improved from %s to %s during sustained degradation workload",
				i, prev, cur)
		}
	}

	// 3. Final state must be Red.
	if levelSequence[len(levelSequence)-1] != metrics.Red {
		t.Errorf("after %d red-range measurements, final state must be Red; got %s",
			2*n+5, levelSequence[len(levelSequence)-1])
	}
}

// TestQualityIndicator_RecoveryNeverSkipsLevel verifies the recovery path
// Red → Yellow → Green.  Recovery requires full hysteresis windows at each step.
//
// VP-027 — "no single-step red→green transition without intermediate yellow"
// BC-2.06.001 invariant 3 — hysteresis on upgrade
// AC-004
func TestQualityIndicator_RecoveryNeverSkipsLevel(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveToRed(t, qi)

	// Feed n Yellow-range measurements; must reach Yellow, not Green.
	for i := 0; i < n; i++ {
		qi.Update(200, 0) // Yellow range
	}
	if qi.Current() == metrics.Green {
		t.Errorf("upgrade from Red skipped Yellow: went directly to Green after %d yellow-range measurements", n)
	}
	assertLevel(t, qi.Current(), metrics.Yellow, "after n yellow-range measurements from Red")

	// Now feed n Green-range measurements; must reach Green.
	for i := 0; i < n; i++ {
		qi.Update(50, 1) // Green range
	}
	assertLevel(t, qi.Current(), metrics.Green, "after n green-range measurements from Yellow")
}

// TestQualityIndicator_DowngradeIsImmediate verifies that downgrade (upgrade
// in the "worse" direction) is immediate (no hysteresis on downgrade).
// A single Red-range measurement from Green must produce at worst Yellow on
// the same update call; the indicator never skips downgrade levels but
// should respond without delay.
//
// BC-2.06.001 (downgrade is immediate per invariant 3 + AC-004 description)
// AC-004
func TestQualityIndicator_DowngradeIsImmediate(t *testing.T) {
	t.Parallel()
	qi := metrics.NewQualityIndicator()
	driveGreenBaseline(t, qi)

	// Single Yellow-range measurement: must downgrade to Yellow immediately (no hysteresis on downgrade).
	qi.Update(200, 0)
	assertLevel(t, qi.Current(), metrics.Yellow,
		"single yellow-range measurement from Green must downgrade immediately to Yellow")
}

// TestQualityIndicator_MissingFrameRecoveryAfterDowngrade verifies the
// BC-2.06.002 canonical test vector:
// "3 frames missing then 3 frames received → indicator degrades then recovers".
//
// BC-2.06.002 canonical test vector (row 2)
// BC-2.06.002 PC-4 — recovery after M=3 consecutive good frames
func TestQualityIndicator_MissingFrameRecoveryAfterDowngrade(t *testing.T) {
	t.Parallel()
	n := metrics.HysteresisCount
	qi := metrics.NewQualityIndicator()
	driveGreenBaseline(t, qi)

	// Downgrade Green → Yellow via 3 missing frames.
	for i := 0; i < n; i++ {
		qi.OnMissingFrame()
	}
	assertLevel(t, qi.Current(), metrics.Yellow, "after n missing frames: must be Yellow")

	// Recover via n consecutive good frames (Update resets the missing-frame counter
	// and counts toward hysteresis upgrade).
	for i := 0; i < n; i++ {
		qi.Update(50, 1) // green-range measurement
	}
	assertLevel(t, qi.Current(), metrics.Green, "after n consecutive good frames: must recover to Green")
}

// -----------------------------------------------------------------------
// String method — sanity
// -----------------------------------------------------------------------

// TestQualityIndicator_String verifies the human-readable labels for all
// Quality constant values.
func TestQualityIndicator_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		q    metrics.Quality
		want string
	}{
		{metrics.Green, "green"},
		{metrics.Yellow, "yellow"},
		{metrics.Red, "red"},
	}

	for _, tc := range tests {
		if got := tc.q.String(); got != tc.want {
			t.Errorf("Quality(%d).String() = %q, want %q", int(tc.q), got, tc.want)
		}
	}
}

// -----------------------------------------------------------------------
// Concurrent access — race-detector coverage
// -----------------------------------------------------------------------

// TestQualityIndicator_ConcurrentUpdates exercises Update, OnMissingFrame, and
// Current concurrently to expose data races under -race.
// This is not a functional assertion; the race detector is the oracle.
func TestQualityIndicator_ConcurrentUpdates(t *testing.T) {
	t.Parallel()

	qi := metrics.NewQualityIndicator()
	const goroutines = 10
	const callsEach = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < callsEach; j++ {
				qi.Update(float64(50+j), float64(j%10))
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < callsEach; j++ {
				qi.OnMissingFrame()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < callsEach; j++ {
				_ = qi.Current()
			}
		}()
	}

	wg.Wait()
	// No assertion: -race is the correctness oracle.
	_ = qi.Current()
}
