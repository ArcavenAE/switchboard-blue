package metrics_test

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/metrics"
)

// TestQualityIndicator_ThresholdClassification verifies that Update returns the
// correct quality level for inputs at and around the NFR-001 thresholds.
//
// AC-001 traces to BC-2.06.001 postcondition 1.
func TestQualityIndicator_ThresholdClassification(t *testing.T) {
	t.Parallel()

	// n is a local copy of the constant so that loop-bound-never-changes
	// static analysis (SA4008) does not fire on the test loops below.
	n := metrics.HysteresisCount

	tests := []struct {
		name     string
		rttMs    float64
		lossPct  float64
		wantBase metrics.Quality // quality expected after a stable run of n updates
	}{
		{
			name:     "green: RTT at boundary, zero loss",
			rttMs:    metrics.GreenRTTMs,
			lossPct:  0,
			wantBase: metrics.Green,
		},
		{
			name:     "green: well inside budget",
			rttMs:    15,
			lossPct:  0,
			wantBase: metrics.Green,
		},
		{
			name:     "yellow: RTT just above green threshold",
			rttMs:    101,
			lossPct:  0,
			wantBase: metrics.Yellow,
		},
		{
			name:     "yellow: RTT at yellow boundary",
			rttMs:    metrics.YellowRTTMs,
			lossPct:  0,
			wantBase: metrics.Yellow,
		},
		{
			name:     "yellow: loss in yellow range",
			rttMs:    50,
			lossPct:  10,
			wantBase: metrics.Yellow,
		},
		{
			name:     "yellow: loss at yellow boundary",
			rttMs:    50,
			lossPct:  metrics.YellowLossPct,
			wantBase: metrics.Yellow,
		},
		{
			name:     "red: RTT above yellow threshold",
			rttMs:    501,
			lossPct:  0,
			wantBase: metrics.Red,
		},
		{
			name:     "red: loss above yellow threshold",
			rttMs:    50,
			lossPct:  21,
			wantBase: metrics.Red,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			qi := metrics.NewQualityIndicator()

			// Drive n updates to allow hysteresis to settle on a
			// non-initial level (downgrade is immediate; this loop ensures a
			// degraded indicator can stabilise from the Green zero value).
			for i := 0; i < n; i++ {
				qi.Update(tc.rttMs, tc.lossPct)
			}

			got := qi.Current()
			if got != tc.wantBase {
				t.Errorf("after %d updates at RTT=%.0f loss=%.0f: got %s, want %s",
					n, tc.rttMs, tc.lossPct, got, tc.wantBase)
			}
		})
	}
}

// TestQualityIndicator_HysteresisUpgrade verifies that a single good measurement
// after sustained degradation does not upgrade the indicator; HysteresisCount
// consecutive good measurements are required.
//
// AC-002 traces to BC-2.06.001 invariant 1.
func TestQualityIndicator_HysteresisUpgrade(t *testing.T) {
	t.Parallel()

	qi := metrics.NewQualityIndicator()

	// Drive to Red by sustained bad measurements.
	for i := 0; i < metrics.HysteresisCount; i++ {
		qi.Update(600, 25) // clearly red
	}
	if qi.Current() != metrics.Red {
		t.Fatalf("expected Red after %d bad updates, got %s", metrics.HysteresisCount, qi.Current())
	}

	// One good measurement must NOT upgrade.
	qi.Update(50, 1)
	if qi.Current() != metrics.Red {
		t.Errorf("single good measurement must not upgrade: got %s, want %s", qi.Current(), metrics.Red)
	}

	// Two more (total = HysteresisCount-1) must still not upgrade.
	for i := 0; i < metrics.HysteresisCount-2; i++ {
		qi.Update(50, 1)
	}
	if qi.Current() == metrics.Green {
		t.Errorf("upgrade should not happen before %d consecutive good measurements", metrics.HysteresisCount)
	}

	// The final measurement completing the hysteresis window should allow upgrade.
	qi.Update(50, 1) // HysteresisCount-th consecutive green measurement
	if qi.Current() == metrics.Red {
		t.Errorf("after %d consecutive good measurements, indicator should have upgraded from Red", metrics.HysteresisCount)
	}
}

// TestQualityIndicator_MissingFrameDowngrade verifies that OnMissingFrame
// triggers a one-step downgrade after HysteresisCount consecutive missing frames.
//
// AC-003 traces to BC-2.06.002 postcondition 1.
func TestQualityIndicator_MissingFrameDowngrade(t *testing.T) {
	t.Parallel()

	// n is a local copy so that SA4008 (loop-bound-never-changes) does not fire.
	n := metrics.HysteresisCount

	qi := metrics.NewQualityIndicator()

	// Establish Green baseline.
	for i := 0; i < n; i++ {
		qi.Update(50, 1) // green conditions
	}
	if qi.Current() != metrics.Green {
		t.Fatalf("expected Green baseline, got %s", qi.Current())
	}

	// Fewer than n missing frames must not downgrade.
	for i := 0; i < n-1; i++ {
		qi.OnMissingFrame()
	}
	if qi.Current() != metrics.Green {
		t.Errorf("fewer than %d missing frames must not downgrade: got %s", n, qi.Current())
	}

	// The n-th missing frame should trigger one-step downgrade.
	qi.OnMissingFrame()
	if qi.Current() != metrics.Yellow {
		t.Errorf("after %d consecutive missing frames: got %s, want %s",
			n, qi.Current(), metrics.Yellow)
	}
}

// TestQualityIndicator_DegradationOnlyGoesDown verifies that downgrade never
// skips a level (green→yellow→red, not green→red) and that upgrade requires
// the full hysteresis window.
//
// AC-004 traces to BC-2.06.002 postcondition 2.
func TestQualityIndicator_DegradationOnlyGoesDown(t *testing.T) {
	t.Parallel()

	// n is a local copy so that SA4008 (loop-bound-never-changes) does not fire.
	n := metrics.HysteresisCount

	qi := metrics.NewQualityIndicator()

	// Establish Green.
	for i := 0; i < n; i++ {
		qi.Update(50, 1)
	}
	if qi.Current() != metrics.Green {
		t.Fatalf("expected Green baseline, got %s", qi.Current())
	}

	// Immediately drive to extreme-Red conditions; downgrade must pass through Yellow first.
	qi.Update(9999, 99) // extreme red
	afterOne := qi.Current()
	if afterOne == metrics.Red {
		t.Errorf("first extreme-red update must not skip Yellow: got Red immediately")
	}

	// Continue until Red is reached; every intermediate step must be Yellow.
	for i := 0; i < metrics.HysteresisCount+5; i++ {
		qi.Update(9999, 99)
		level := qi.Current()
		if level != metrics.Yellow && level != metrics.Red {
			t.Errorf("unexpected level during degradation: %s", level)
		}
	}

	if qi.Current() != metrics.Red {
		t.Errorf("expected Red after sustained extreme conditions, got %s", qi.Current())
	}
}
