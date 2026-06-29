package metrics_test

// Property-based tests for QualityIndicator.
//
// Coverage map:
//
//	TestProp_BC_2_06_001_NoSkipTransitionDuringDegradation — VP-027, BC-2.06.001 invariant 3
//	TestProp_BC_2_06_001_NoRedToGreenSkipDuringRecovery    — VP-027
//	TestProp_BC_2_06_001_QualityIsAlwaysValidEnum          — VP-027 (value-set invariant)
//	TestProp_BC_2_06_002_MissingFrameNeverSkipsLevel       — VP-052, BC-2.06.002 PC-2

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/arcavenae/switchboard/internal/metrics"
)

// gopterParams returns property test configuration.
// 1000 samples satisfies the "≥1000 random cases" requirement from the
// test-writer operating procedure.
func gopterParams() *gopter.TestParameters {
	p := gopter.DefaultTestParameters()
	p.MinSuccessfulTests = 1000
	return p
}

// degradationStep is a single (rttMs, lossPct) update.
type degradationStep struct {
	rttMs   float64
	lossPct float64
}

// genDegradationStep generates a single degradation step as a pair of uint32 values
// (rtt increment, loss increment), returning a degradationStep.
// The approach: generate 16 pairs of increments as flat int slices and reconstruct.
func genMonotoneRisingSteps() gopter.Gen {
	// Generate a slice of 32 uint32 values: [rttInc0, lossInc0, rttInc1, lossInc1, ...].
	// Each pair (i*2, i*2+1) is the increment for step i.
	//
	// CR-004 bias fix: start in the Yellow range (300 ms) and use a minimum RTT
	// increment of 20 ms. Starting at 300 ms means the Red threshold (> 500 ms)
	// is crossed after at most ceil(200/20) = 10 steps — well within 16.
	// Using gen.UInt32Range(20, 60) keeps the increments meaningful and gives a
	// near-certain probability (>99%) of reaching Red within the 16-step budget.
	return gen.SliceOfN(32, gen.UInt32Range(20, 60)).Map(func(incs []uint32) []degradationStep {
		steps := make([]degradationStep, 16)
		rttAcc := float64(300) // start in Yellow range so Red is reachable within 16 steps
		var lossAcc float64
		for i := 0; i < 16; i++ {
			rttAcc += float64(incs[i*2])
			lossAcc += float64(incs[i*2+1] % 6) // 0-5 loss increment
			if lossAcc > 100 {
				lossAcc = 100
			}
			steps[i] = degradationStep{rttMs: rttAcc, lossPct: lossAcc}
		}
		return steps
	})
}

// genMonotoneRecoverySteps generates steps starting at Red-range, monotonically
// decreasing toward Green-range.
func genMonotoneRecoverySteps() gopter.Gen {
	// CR-005 bias fix: use gen.UInt32Range(25, 50) so each step decrements RTT by
	// at least 25 ms. Starting at 800 ms, 16 steps at minimum decrement brings RTT
	// down to 800 - 25*16 = 400 ms (Yellow), and at maximum (50 ms/step) down to
	// 800 - 50*16 = 0 ms (Green). This makes Green-range reachable in practice
	// while keeping the monotone-recovery invariant exercised across the full range.
	return gen.SliceOfN(32, gen.UInt32Range(25, 50)).Map(func(decs []uint32) []degradationStep {
		steps := make([]degradationStep, 16)
		rttAcc := float64(800)
		lossAcc := float64(25)
		for i := 0; i < 16; i++ {
			rttAcc -= float64(decs[i*2])
			lossAcc -= float64(decs[i*2+1] % 6)
			if rttAcc < 0 {
				rttAcc = 0
			}
			if lossAcc < 0 {
				lossAcc = 0
			}
			steps[i] = degradationStep{rttMs: rttAcc, lossPct: lossAcc}
		}
		return steps
	})
}

// TestProp_BC_2_06_001_NoSkipTransitionDuringDegradation verifies property VP-027:
// under a monotone-degradation workload, the quality indicator never transitions
// directly from Green to Red (it must pass through Yellow).
//
// VP-027 — "quality transitions follow only: green → yellow → red"
// BC-2.06.001 invariant 3
func TestProp_BC_2_06_001_NoSkipTransitionDuringDegradation(t *testing.T) {
	// VP-027 — BC-2.06.001 invariant 3 — no Green→Red skip
	properties := gopter.NewProperties(gopterParams())

	properties.Property("no Green→Red skip during monotone degradation", prop.ForAll(
		func(steps []degradationStep) bool {
			if len(steps) == 0 {
				return true
			}
			qi := metrics.NewQualityIndicator()
			prev := qi.Current()

			for _, s := range steps {
				qi.Update(s.rttMs, s.lossPct)
				cur := qi.Current()
				if prev == metrics.Green && cur == metrics.Red {
					return false // illegal skip
				}
				prev = cur
			}
			return true
		},
		genMonotoneRisingSteps(),
	))

	properties.TestingRun(t)
}

// TestProp_BC_2_06_001_NoRedToGreenSkipDuringRecovery verifies the recovery
// direction of VP-027: Red must not jump directly to Green without passing
// through Yellow.
//
// VP-027 — "no single-step red→green transition without an intermediate yellow"
func TestProp_BC_2_06_001_NoRedToGreenSkipDuringRecovery(t *testing.T) {
	// VP-027 — no Red→Green skip
	properties := gopter.NewProperties(gopterParams())

	properties.Property("no Red→Green skip during monotone recovery", prop.ForAll(
		func(steps []degradationStep) bool {
			if len(steps) == 0 {
				return true
			}
			qi := metrics.NewQualityIndicator()
			// Pre-drive to Red so recovery transitions are exercised.
			for i := 0; i < metrics.HysteresisCount; i++ {
				qi.Update(600, 25)
			}
			prev := qi.Current()

			for _, s := range steps {
				qi.Update(s.rttMs, s.lossPct)
				cur := qi.Current()
				if prev == metrics.Red && cur == metrics.Green {
					return false // illegal skip
				}
				prev = cur
			}
			return true
		},
		genMonotoneRecoverySteps(),
	))

	properties.TestingRun(t)
}

// TestProp_BC_2_06_001_QualityIsAlwaysValidEnum verifies that every call to
// Current() returns one of the three defined Quality values (no invalid state).
//
// VP-027 — "Quality indicator is always one of: green, yellow, red"
// BC-2.06.001 (Quality is a closed enum)
func TestProp_BC_2_06_001_QualityIsAlwaysValidEnum(t *testing.T) {
	// VP-027 — Quality value is always in {Green, Yellow, Red}
	properties := gopter.NewProperties(gopterParams())

	properties.Property("Current() always returns a valid Quality enum value", prop.ForAll(
		func(steps []degradationStep) bool {
			qi := metrics.NewQualityIndicator()
			for _, s := range steps {
				qi.Update(s.rttMs, s.lossPct)
				cur := qi.Current()
				if cur != metrics.Green && cur != metrics.Yellow && cur != metrics.Red {
					return false
				}
			}
			return true
		},
		genMonotoneRisingSteps(),
	))

	properties.TestingRun(t)
}

// TestProp_BC_2_06_002_MissingFrameNeverSkipsLevel verifies the missing-frame
// degradation path does not skip levels across arbitrary interleavings of
// OnMissingFrame and Update calls.
//
// BC-2.06.002 PC-2 — "degrades one level (green→yellow or yellow→red)"
// VP-052 — N=3 consecutive gaps trigger indicator downgrade
func TestProp_BC_2_06_002_MissingFrameNeverSkipsLevel(t *testing.T) {
	// BC-2.06.002 PC-2 — missing-frame downgrade is always one level at a time
	properties := gopter.NewProperties(gopterParams())

	// Generate a sequence of booleans: true = OnMissingFrame, false = good Update.
	properties.Property("OnMissingFrame never skips a degradation level", prop.ForAll(
		func(boolSeq []bool) bool {
			qi := metrics.NewQualityIndicator()
			prev := qi.Current()

			for _, isMissing := range boolSeq {
				if isMissing {
					qi.OnMissingFrame()
				} else {
					qi.Update(50, 1) // green-range to reset missing-frame counter
				}
				cur := qi.Current()
				// A single step must never jump more than one level in the downgrade direction.
				if int(cur)-int(prev) > 1 {
					return false // skipped a level
				}
				prev = cur
			}
			return true
		},
		gen.SliceOfN(30, gen.Bool()),
	))

	properties.TestingRun(t)
}
