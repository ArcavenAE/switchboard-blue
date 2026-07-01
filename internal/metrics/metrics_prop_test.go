package metrics_test

// Property-based tests for QualityIndicator.
//
// Coverage map:
//
//	TestProp_BC_2_06_001_NoSkipTransitionDuringDegradation — VP-027, BC-2.06.001 invariant 3
//	TestProp_BC_2_06_001_NoRedToGreenSkipDuringRecovery    — VP-027
//	TestProp_BC_2_06_001_QualityIsAlwaysValidEnum          — VP-027 (value-set invariant)
//	TestProp_BC_2_06_001_GreenToRedSingleStep              — VP-027, F-C1 (Green→Red jump tested)
//	TestProp_BC_2_06_002_MissingFrameNeverSkipsLevel       — VP-052, BC-2.06.002 PC-2
//	TestQualityIndicator_OnMissingFrame_PropertyMonotone   — VP-074, BC-2.06.002

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

// shrinkDegradationSteps shrinks a []degradationStep by progressively
// removing steps from the tail. This minimises the failing counter-example
// to the shortest sequence that still triggers the property failure.
func shrinkDegradationSteps(v interface{}) gopter.Shrink {
	steps, ok := v.([]degradationStep)
	if !ok || len(steps) == 0 {
		return gopter.NoShrink
	}
	i := len(steps) - 1
	return func() (interface{}, bool) {
		if i <= 0 {
			return nil, false
		}
		shrunk := make([]degradationStep, i)
		copy(shrunk, steps[:i])
		i--
		return shrunk, true
	}
}

// genMonotoneRisingSteps generates a monotone-degradation sequence starting
// in the Green band and increasing until Red is reached within the 16-step
// budget.
//
// Step bounds (40–100ms/step) are chosen so the Green→Yellow boundary
// (100ms) and Yellow→Red boundary (500ms) are both crossed in a typical
// run, making the Green→Red skip guard actually reachable rather than
// structurally dead code after step 1.
//
// Worst-case steps to Red: ceil((500 − 10) / 40) = 13 — within the budget.
func genMonotoneRisingSteps() gopter.Gen {
	// Generate a slice of 32 uint32 values: [rttInc0, lossInc0, rttInc1, lossInc1, ...].
	// Each pair (i*2, i*2+1) is the increment for step i.
	return gen.SliceOfN(32, gen.UInt32Range(40, 100)).
		Map(func(incs []uint32) []degradationStep {
			steps := make([]degradationStep, 16)
			rttAcc := float64(10 + incs[0]%50) // start in Green band: 10–59ms
			var lossAcc float64
			for i := 0; i < 16; i++ {
				rttAcc += float64(incs[i*2])
				lossAcc += float64(incs[i*2+1] % 6) // 0–5 loss increment
				if lossAcc > 100 {
					lossAcc = 100
				}
				steps[i] = degradationStep{rttMs: rttAcc, lossPct: lossAcc}
			}
			return steps
		}).
		WithShrinker(shrinkDegradationSteps)
}

// genMonotoneRecoverySteps generates steps starting at Red-range, monotonically
// decreasing toward Green-range.
func genMonotoneRecoverySteps() gopter.Gen {
	// CR-005 bias fix: use gen.UInt32Range(25, 50) so each step decrements RTT by
	// at least 25 ms. Starting at 800 ms, 16 steps at minimum decrement brings RTT
	// down to 800 - 25*16 = 400 ms (Yellow), and at maximum (50 ms/step) down to
	// 800 - 50*16 = 0 ms (Green). This makes Green-range reachable in practice
	// while keeping the monotone-recovery invariant exercised across the full range.
	return gen.SliceOfN(32, gen.UInt32Range(25, 50)).
		Map(func(decs []uint32) []degradationStep {
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
		}).
		WithShrinker(shrinkDegradationSteps)
}

// greenToRedPair is a pair of steps: one firmly in Green and one firmly in Red,
// used by genGreenToRedJump to exercise the single-step Green→Red transition path.
type greenToRedPair struct {
	green degradationStep
	red   degradationStep
}

// genGreenToRedJump generates exactly one Green-band step followed by one
// Red-band step, to verify that the QualityIndicator correctly transitions
// Green→Yellow→Red (not Green→Red directly) when a single measurement jumps
// all the way into the Red band.
//
// F-C1: this generator makes the Green→Red case directly testable rather than
// relying on it being incidentally reachable from the rising-steps generator.
func genGreenToRedJump() gopter.Gen {
	// Green: RTT ≤ 100ms AND loss ≤ 5%.
	// Red:   RTT > 500ms OR loss > 20%.
	greenRTT := gen.Float64Range(10, metrics.GreenRTTMs)
	greenLoss := gen.Float64Range(0, metrics.GreenLossPct)
	redRTT := gen.Float64Range(metrics.YellowRTTMs+1, 2000)
	redLoss := gen.Float64Range(0, 5) // loss stays low so RTT alone drives Red
	return gopter.CombineGens(greenRTT, greenLoss, redRTT, redLoss).
		Map(func(vals []interface{}) greenToRedPair {
			return greenToRedPair{
				green: degradationStep{rttMs: vals[0].(float64), lossPct: vals[1].(float64)},
				red:   degradationStep{rttMs: vals[2].(float64), lossPct: vals[3].(float64)},
			}
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

// TestProp_BC_2_06_001_GreenToRedSingleStep verifies that a single Update call
// that jumps from a Green-band measurement to a Red-band measurement does NOT
// produce a Green→Red transition — the indicator must pass through Yellow first.
//
// F-C1: this exercises the path that genMonotoneRisingSteps cannot reliably
// reach because its bounded increments make Green→Red unreachable in one step.
// VP-027 — BC-2.06.001 invariant 3 (no skip under any input, not just monotone).
func TestProp_BC_2_06_001_GreenToRedSingleStep(t *testing.T) {
	// VP-027 — BC-2.06.001 invariant 3 — no Green→Red skip on a single jump input
	properties := gopter.NewProperties(gopterParams())

	properties.Property("single Green→Red jump does not skip Yellow", prop.ForAll(
		func(pair greenToRedPair) bool {
			qi := metrics.NewQualityIndicator()

			// Stabilise in Green with HysteresisCount good measurements.
			for i := 0; i < metrics.HysteresisCount; i++ {
				qi.Update(pair.green.rttMs, pair.green.lossPct)
			}
			// Single Red-range measurement.
			qi.Update(pair.red.rttMs, pair.red.lossPct)
			cur := qi.Current()

			// After one Red-range call the indicator must be Yellow (immediate downgrade)
			// or still Green (if the implementation batches) — but NEVER Red, because
			// a Green→Red skip violates BC-2.06.001 invariant 3.
			return cur != metrics.Red
		},
		genGreenToRedJump(),
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

// TestQualityIndicator_OnMissingFrame_PropertyMonotone verifies two invariants
// from BC-2.06.002 under arbitrary missing-frame sequences:
//
//  1. Each individual OnMissingFrame() call never degrades the level by more
//     than one step (BC-2.06.002 PC-2: "degrades one level per threshold crossing").
//  2. Once HysteresisCount consecutive missing frames have been observed,
//     the OR-form threshold fires: the level degrades exactly one step at the
//     threshold crossing call, and sub-threshold calls leave the level unchanged.
//
// VP-074 — missing-frame threshold fires at N=HysteresisCount and degrades
// exactly one level per threshold crossing.
// BC-2.06.002 OR-form: threshold fires after N consecutive missing frames.
func TestQualityIndicator_OnMissingFrame_PropertyMonotone(t *testing.T) {
	// VP-074 — BC-2.06.002 OR-form threshold
	properties := gopter.NewProperties(gopterParams())

	properties.Property("each OnMissingFrame call degrades at most one level", prop.ForAll(
		func(calls []uint8) bool {
			// calls[i] encodes the i-th action: 0 = good Update, 1..255 = that many
			// OnMissingFrame() calls in a row.  After each burst we inspect per-call.
			//
			// Invariants verified:
			//   a) no individual OnMissingFrame() call moves the level by > 1.
			//   b) the level never increases via OnMissingFrame (monotone downward only).
			//   c) a run of < HysteresisCount consecutive OnMissingFrame() calls
			//      (without an interleaved good Update) does not change the level.
			qi := metrics.NewQualityIndicator()
			consecutiveMisses := 0

			for _, v := range calls {
				if v == 0 {
					// Good measurement — resets the missing-frame counter.
					qi.Update(50, 1)
					consecutiveMisses = 0
					continue
				}

				// Fire v individual OnMissingFrame() calls, checking invariants per call.
				for i := 0; i < int(v); i++ {
					before := qi.Current()
					qi.OnMissingFrame()
					after := qi.Current()
					consecutiveMisses++

					// Invariant a: no single call degrades by more than one level.
					if int(after)-int(before) > 1 {
						return false
					}

					// Invariant b: level never improves via OnMissingFrame.
					if after < before {
						return false
					}

					// Invariant c: level only changes at exact multiples of HysteresisCount.
					if consecutiveMisses%metrics.HysteresisCount != 0 && after != before {
						return false
					}

					// At the threshold crossing: level must have advanced by exactly 1
					// (unless already at Red, where it stays).
					if consecutiveMisses%metrics.HysteresisCount == 0 {
						if before < metrics.Red && int(after) != int(before)+1 {
							return false
						}
						if before == metrics.Red && after != metrics.Red {
							return false
						}
					}
				}
			}
			return true
		},
		// Each element is 0 (good update) or 1..6 (burst of missing frames).
		gen.SliceOfN(30, gen.UInt8Range(0, 6)),
	))

	properties.TestingRun(t)
}
