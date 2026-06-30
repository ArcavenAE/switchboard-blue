// Package paths_test: property-based tests for VP-063 (S-5.03) and S-5.02
// histogram invariants.
//
// VP-063: PathTracker.IsDegraded() is true iff EWMA-smoothed RTT >
// DegradedRTTThresholdMS (200.0ms). Recovery below the threshold clears the
// flag. No hysteresis. No additional delay.
//
// S-5.02 histogram properties (no VP ID assigned; formal accuracy VP deferred
// to S-BL.BENCH per ARCH-03 v1.6 §VP note):
//
//	TestProp_P99_SampleCountMonotone   — SampleCount only increases; never decreases
//	TestProp_P99_BucketBoundaryIntegrity — P99RTTMs ≤ bucket upper-bound for all single-bucket distributions
//
// These tests require gopter v0.2.9+ (added to go.mod by S-5.03 test-writer).
// Run with: go test ./internal/paths/ -run TestProp_
//
// AC-004, AC-005 / BC-2.06.003 EC-003, PC-1 / ARCH-03 v1.6 §p99 RTT Accumulator
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

// ─── S-5.02 histogram property tests ────────────────────────────────────────
//
// No VP ID is assigned to these properties; the formal accuracy VP for the
// histogram accumulator is deferred to S-BL.BENCH (per ARCH-03 v1.6 §VP note).
// Do NOT add a VP ID here.

// TestProp_P99_SampleCountMonotone verifies that SampleCount in PathSnapshot
// only ever increases across successive OnProbe calls — it never decreases.
//
// This is a structural invariant of the rttHistogram total counter:
// record() increments h.total by 1 on every call; nothing decrements it.
// An increment of exactly 1 per successful probe is required (no batching).
//
// Property: for any sequence of ≥1 RTT samples, SampleCount after k probes =
// SampleCount after k-1 probes + 1.
//
// AC-004 / BC-2.06.003 EC-003 / ARCH-03 v1.6 §p99 RTT Accumulator
func TestProp_P99_SampleCountMonotone(t *testing.T) {
	// 1000 samples satisfies the ≥1000 random cases requirement.
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 1000

	properties := gopter.NewProperties(params)

	properties.Property("SampleCount increments by exactly 1 per successful probe", prop.ForAll(
		func(rtts []float64) bool {
			if len(rtts) == 0 {
				return true
			}
			tracker := paths.NewPathTracker(500.0, 0.125)
			var prevCount uint64
			for i, rtt := range rtts {
				tracker.OnProbe(rtt, false)
				snap := tracker.Snapshot()
				if snap.SampleCount < prevCount {
					// SampleCount decreased — structural invariant violated.
					_ = i
					return false
				}
				// Each successful probe must increment by exactly 1.
				// Probe 0 resets via resetRTT which also calls record(), so
				// after probe 0 SampleCount must be 1.
				expectedCount := uint64(i + 1)
				if snap.SampleCount != expectedCount {
					return false
				}
				prevCount = snap.SampleCount
			}
			return true
		},
		// Non-empty slices of RTT values in [0, 2000] ms.
		gen.SliceOf(gen.Float64Range(0, 2000)).
			SuchThat(func(v interface{}) bool { return len(v.([]float64)) > 0 }),
	))

	properties.TestingRun(t)
}

// bucketUpperBound returns the upper bound (right edge, exclusive) for the
// histogram bucket that contains arrivalRTTMS. The bucket boundaries mirror
// rttHistogramBuckets in paths.go exactly (ARCH-03 v1.6 §p99 RTT Accumulator):
//
//	Buckets: [0,25), [25,50), [50,75), [75,100), [100,125), [125,150),
//	         [150,175), [175,200), [200,300), [300,400), [400,500),
//	         [500,700), [700,1000), [1000,1400), [1400,2000), [2000,∞)
//
// This is a test-local mirror — it is intentionally separate from the
// implementation so that the property test can verify the implementation rather
// than tautologically agree with it.
func bucketUpperBound(rttMS float64) float64 {
	edges := [16]float64{
		25, 50, 75, 100, 125, 150, 175, 200, 300, 400, 500, 700, 1000, 1400, 2000, 1e18,
	}
	for _, e := range edges {
		if rttMS < e {
			return e
		}
	}
	return 1e18
}

// TestProp_P99_BucketBoundaryIntegrity verifies the approximation bound for
// single-bucket distributions: when all ≥10 samples fall within the same
// histogram bucket, P99RTTMs must be ≤ that bucket's upper bound.
//
// This exercises the ARCH-03 guarantee:
//
//	p99() ≤ true_p99 + max_bucket_width
//
// For a single-bucket distribution the true_p99 is the sample value itself,
// and the approximation error is at most one bucket width, so the upper bound
// equals the bucket's right edge.
//
// Property: for any RTT value r and count ≥10, if all samples fall in the
// same bucket as r, then P99RTTMs ≤ bucketUpperBound(r).
//
// NOTE: no VP ID. Formal accuracy VP is deferred to S-BL.BENCH per ARCH-03 v1.6.
//
// AC-005 / BC-2.06.003 PC-1 / ARCH-03 v1.6 §p99 RTT Accumulator
func TestProp_P99_BucketBoundaryIntegrity(t *testing.T) {
	// 1000 samples satisfies the ≥1000 random cases requirement.
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 1000

	properties := gopter.NewProperties(params)

	properties.Property("P99RTTMs ≤ bucket upper bound for single-bucket distributions", prop.ForAll(
		func(rttMS float64, extraCount uint8) bool {
			// Use at least 10 samples (the pending threshold) plus up to 200 more.
			count := int(extraCount) + 10

			tracker := paths.NewPathTracker(500.0, 0.125)
			for i := 0; i < count; i++ {
				tracker.OnProbe(rttMS, false)
			}

			snap := tracker.Snapshot()
			if snap.SampleCount < 10 {
				// Should never happen with count ≥ 10 — guard against stub bugs.
				return false
			}

			upperBound := bucketUpperBound(rttMS)
			// P99RTTMs must be positive (non-zero) when SampleCount ≥ 10 and rttMS > 0.
			// A stub that returns 0 always would vacuously pass the ≤ upper-bound check
			// but would violate this assertion.
			if rttMS > 0 && snap.P99RTTMs <= 0 {
				return false // p99 must be a positive value when samples are non-zero
			}
			// P99RTTMs must be ≤ the bucket's upper bound.
			// If the histogram correctly routes samples, the p99 bucket is the
			// same bucket as rttMS, so the reported value cannot exceed upperBound.
			return snap.P99RTTMs <= upperBound
		},
		// RTT values across the full histogram range [0, 1999] ms.
		gen.Float64Range(0, 1999),
		// Extra samples beyond the 10-sample threshold.
		gen.UInt8Range(0, 200),
	))

	properties.TestingRun(t)
}
