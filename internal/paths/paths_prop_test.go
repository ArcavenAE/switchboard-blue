// Package paths_test: property-based tests for VP-063.
//
// VP-063: PathTracker.IsDegraded() is true iff EWMA-smoothed RTT >
// DegradedRTTThresholdMS (200.0ms). Recovery below the threshold clears the
// flag. No hysteresis. No additional delay.
//
// These tests require gopter v0.2.9+ (added to go.mod by S-5.03 test-writer).
// Run with: go test ./internal/paths/ -run TestProp_
//
// AC-005 / BC-2.02.003 postcondition 5 / VP-063
package paths_test

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/arcavenae/switchboard/internal/paths"
)

// ewmaFromSamples computes the EWMA RTT that PathTracker.OnProbe will have
// converged to after feeding samples to a fresh PathTracker with the given
// initialRTT and alpha. It mirrors paths.go's EWMA arithmetic exactly:
//   - Probe 1: resetRTT → ewma = samples[0]  (first-probe override)
//   - Probe k>1: ewma = alpha*samples[k-1] + (1-alpha)*ewma
//
// The function returns the final EWMA value, which is what IsDegraded() must
// compare against DegradedRTTThresholdMS.
func ewmaFromSamples(initialRTT float64, alpha float64, samples []float64) float64 {
	ewma := initialRTT
	for i, rtt := range samples {
		if i == 0 {
			// First-probe override: resetRTT sets ewmaRTTMS = arrivalRTTMS directly.
			ewma = rtt
		} else {
			ewma = alpha*rtt + (1-alpha)*ewma
		}
	}
	return ewma
}

// genRTTSamples generates a non-empty slice of float64 RTT values in [0, 500] ms.
// This is the input space for VP-063 property tests.
func genRTTSamples() gopter.Gen {
	return gen.SliceOf(gen.Float64Range(0, 500)).
		SuchThat(func(v interface{}) bool {
			return len(v.([]float64)) > 0
		})
}

// TestProp_IsDegraded_TracksEWMAThreshold is the primary VP-063 property test.
//
// Property: for any non-empty RTT sample sequence fed to a fresh PathTracker
// (alpha=0.2, initialRTT=999ms conservative), IsDegraded() must equal
// (ewma > DegradedRTTThresholdMS) where ewma is computed by ewmaFromSamples.
//
// This exercises both onset (EWMA crosses above 200ms) and the base case where
// EWMA never reaches the threshold.
//
// AC-005 / VP-063 (primary property)
func TestProp_IsDegraded_TracksEWMAThreshold(t *testing.T) {
	// alpha used inside PathTracker for this test. Must match ewmaFromSamples.
	const alpha = 0.2
	const initialRTT = 999.0 // conservative starting value

	properties := gopter.NewProperties(nil)

	properties.Property("IsDegraded tracks EWMA vs DegradedRTTThresholdMS", prop.ForAll(
		func(samples []float64) bool {
			tracker := paths.NewPathTracker(initialRTT, alpha)

			// Feed all samples to the tracker.
			for _, rtt := range samples {
				tracker.OnProbe(rtt, false)
			}

			// Compute expected EWMA using the same arithmetic as paths.go.
			expectedEWMA := ewmaFromSamples(initialRTT, alpha, samples)
			expectedDegraded := expectedEWMA > paths.DegradedRTTThresholdMS

			return tracker.IsDegraded() == expectedDegraded
		},
		genRTTSamples(),
	))

	properties.TestingRun(t)
}

// TestProp_IsDegraded_RecoveryClears verifies that after a sustained high-RTT
// phase drives the EWMA well above the threshold, a sufficient number of
// low-RTT probes always drives IsDegraded() to false.
//
// Property: for any highRTT in [200.1, 400.0] and recoveryCount in [1, 50],
// 20 probes at highRTT+100 MUST produce IsDegraded()==true (drive phase),
// and (recoveryCount+20) probes at 10ms MUST produce IsDegraded()==false
// (recovery phase).
//
// Both halves of the property must hold. The drive-phase check ensures the
// test is not vacuously satisfied by a stub that always returns false.
//
// The +20 padding ensures enough probes for convergence even with small alpha.
//
// AC-005 / VP-063 (recovery branch)
func TestProp_IsDegraded_RecoveryClears(t *testing.T) {
	const alpha = 0.2
	const initialRTT = 999.0

	properties := gopter.NewProperties(nil)

	properties.Property("IsDegraded sets during drive phase and clears after recovery", prop.ForAll(
		func(highRTT float64, recoveryCount uint8) bool {
			if recoveryCount == 0 {
				recoveryCount = 1
			}
			tracker := paths.NewPathTracker(initialRTT, alpha)

			// Drive phase: 20 probes at highRTT+100ms (well above 200ms threshold).
			// After these probes the EWMA must have converged above 200ms → degraded=true.
			for i := 0; i < 20; i++ {
				tracker.OnProbe(highRTT+100.0, false)
			}

			// The drive phase must have set the flag. If this is false, the stub
			// (updateDegraded is a no-op) causes immediate failure here.
			if !tracker.IsDegraded() {
				return false // drive phase did not set degraded=true
			}

			// Recovery phase: feed low-RTT samples (well below threshold: 10ms).
			const lowRTT = 10.0
			for i := 0; i < int(recoveryCount)+20; i++ {
				tracker.OnProbe(lowRTT, false)
			}

			// After sufficient recovery probes, degraded must be false.
			return !tracker.IsDegraded()
		},
		gen.Float64Range(200.1, 400.0),
		gen.UInt8Range(1, 50),
	))

	properties.TestingRun(t)
}
