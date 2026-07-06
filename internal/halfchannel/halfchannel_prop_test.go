// Package halfchannel_test: property-based tests for VP-051 (HalfChannel
// independence) and VP-053 (K empty-tick frames sequence).
//
// VP-051 (proof_method: proptest, gopter v0.2.9+):
// Given two HalfChannels A and B with distinct (chanID, direction) tuples, B's
// sequence counter is unaffected by A's tick production and vice versa.
//
// VP-053 (proof_method: proptest, gopter v0.2.9+):
// K consecutive Tick() calls with no queued payload emit K frames with
// contiguous ChanSeq values (startSeq+1, startSeq+2, …, startSeq+K), zero
// payload, and FrameType=EMPTY_TICK.
//
// Both tests use the real HalfChannel API (halfchannel.New / Tick / Seq) per
// the v1.2 revision of VP-051 that removed references to the phantom
// halfchannel.Config / NewFakeClock APIs. HalfChannel is pure-core (ARCH-09):
// Tick() is a pure state transition; no fake clock is needed.
//
// BC-2.01.001 / BC-2.01.002 / BC-2.01.003 / VP-051 / VP-053
package halfchannel_test

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/halfchannel"
)

// TestProp_VP051_HalfChannelIndependence discharges VP-051 as a gopter
// proptest. For arbitrary distinct (chanID, direction, interval) tuples and
// arbitrary tick counts N and M, producing N frames from A must leave
// B.Seq()==0, and subsequently producing M frames from B must leave A.Seq()
// unchanged at N.
//
// Generator ranges:
//
//	chanID_A, chanID_B ∈ [1, 65535]      (distinct via gen.SuchThat)
//	intervals          ∈ [5ms, 50ms]     (ADR-008 canonical bounds)
//	N, M               ∈ [1, 100]        (bounded per VP-051.md skeleton)
//
// Mutation-kill: introduce shared global state — e.g., replace `h.seq++` with
// a package-level counter — and A's ticks would advance B's Seq(). Property
// fails.
//
// VP-051 / BC-2.01.003 postcondition 4 / EC-001 / EC-002
func TestProp_VP051_HalfChannelIndependence(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 200

	properties := gopter.NewProperties(params)

	// Generator for a distinct pair of ChanIDs so (chanID_A, dir_A) ≠ (chanID_B, dir_B).
	// Directions are always distinct (Upstream / Downstream); the ChanID
	// distinctness guarantees the tuple is not just direction-distinguished.
	genDistinctChanIDs := gopter.CombineGens(
		gen.UInt32Range(1, 32767),
		gen.UInt32Range(32768, 65535),
	).Map(func(vals []interface{}) [2]uint32 {
		return [2]uint32{vals[0].(uint32), vals[1].(uint32)}
	})

	genIntervalMs := gen.IntRange(5, 50)
	genTickCount := gen.IntRange(1, 100)

	properties.Property(
		"N ticks from A leave B.Seq()==0; M subsequent ticks from B leave A.Seq()==N",
		prop.ForAll(
			func(chanIDs [2]uint32, intervalMsA, intervalMsB, n, m int) bool {
				intervalA := time.Duration(intervalMsA) * time.Millisecond
				intervalB := time.Duration(intervalMsB) * time.Millisecond

				hcA := halfchannel.New(chanIDs[0], halfchannel.Upstream, intervalA)
				hcB := halfchannel.New(chanIDs[1], halfchannel.Downstream, intervalB)

				// Sanity: B starts at Seq()==0.
				if hcB.Seq() != 0 {
					return false
				}
				aSeqStart := hcA.Seq()

				// Produce N frames from A; B must remain untouched.
				for i := 0; i < n; i++ {
					hcA.Tick()
				}
				if hcB.Seq() != 0 {
					return false
				}

				aSeqAfterA := hcA.Seq()
				// A must have advanced by exactly N (post-increment semantics).
				if aSeqAfterA != aSeqStart+uint32(n) {
					return false
				}

				// Produce M frames from B; A must remain unchanged.
				for i := 0; i < m; i++ {
					hcB.Tick()
				}
				if hcA.Seq() != aSeqAfterA {
					return false
				}
				// B must have advanced by exactly M.
				if hcB.Seq() != uint32(m) {
					return false
				}
				return true
			},
			genDistinctChanIDs,
			genIntervalMs, genIntervalMs,
			genTickCount, genTickCount,
		),
	)

	properties.TestingRun(t)
}

// TestProp_VP053_EmptyTickSequence discharges VP-053 as a gopter proptest.
// For arbitrary K in [1, 100], K consecutive Tick() calls with no enqueued
// payload emit K frames whose ChanSeq values are contiguous (startSeq+1..K)
// and whose payload is zero-length with FrameType=EMPTY_TICK.
//
// Post-increment semantics per BC-2.01.001 canonical vector "sequence 1..10":
// Seq() before any tick is 0. First Tick() advances seq to 1 and returns
// ChanSeq=1. So frames[i].ChanSeq == startSeq + uint32(i+1).
//
// Mutation-kill: remove the seq increment inside Tick() (or increment only
// on the first iteration) and the contiguity check fails immediately.
//
// VP-053 / BC-2.01.002 postconditions 1–3 / DI-008
func TestProp_VP053_EmptyTickSequence(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 200

	properties := gopter.NewProperties(params)

	genK := gen.IntRange(1, 100)

	properties.Property(
		"K empty ticks emit K frames with contiguous ChanSeq, zero payload, EMPTY_TICK type",
		prop.ForAll(
			func(k int) bool {
				hc := halfchannel.New(1, halfchannel.Upstream, 10*time.Millisecond)

				startSeq := hc.Seq()
				frames := make([]halfchannel.ChannelFrame, 0, k)

				for i := 0; i < k; i++ {
					// No Enqueue call: Tick() with empty pending queue emits a
					// zero-payload frame (BC-2.01.002).
					frames = append(frames, hc.Tick())
				}

				if len(frames) != k {
					return false
				}
				for i, f := range frames {
					if f.ChanSeq != startSeq+uint32(i+1) {
						return false
					}
					if len(f.Payload) != 0 {
						return false
					}
					if f.FrameType != frame.FrameTypeEmptyTick {
						return false
					}
					if f.ChanID != 1 {
						return false
					}
				}
				// Post-condition: HalfChannel Seq() advanced by exactly K.
				if hc.Seq() != startSeq+uint32(k) {
					return false
				}
				return true
			},
			genK,
		),
	)

	properties.TestingRun(t)
}
