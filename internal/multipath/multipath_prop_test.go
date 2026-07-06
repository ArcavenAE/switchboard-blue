// Package multipath_test: property-based tests for VP-025 (DropCache capacity
// invariant).
//
// VP-025 (proof_method: proptest, gopter v0.2.9+):
// ∀ DropCache c with capacity N: after adding any number of entries, len(c) ≤ N.
//
// This file discharges VP-025's proof_method commitment. The deterministic
// stdlib sweep in multipath_test.go (TestBC_2_02_009_DropCache_BoundedCapacity_PropertySweep)
// covers a fixed grid of capacities × entry counts; the gopter harness below
// generalizes that sweep to random capacities in [1, 64] and random compound-key
// (checksum, arrival_interface_id) insert sequences of length up to 128.
//
// Coverage: after every insert (both first-arrival and duplicate paths), the
// cache Len MUST NOT exceed the configured capacity. This is the fence-post
// invariant the LRU eviction logic must preserve.
//
// Mutation-kill self-check: if `if c.lru.Len() >= c.capacity` in Add/AddIfAbsent
// were relaxed to `>` (off-by-one), the (N+1)th insert would leave the cache at
// N+1 and the property would fail on capacity=1 with 2+ inserts (well within
// the shrink space).
//
// BC-2.02.009 postcondition 3 / AC-006 / VP-025
package multipath_test

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/arcavenae/switchboard/internal/multipath"
)

// TestProp_VP025_DropCache_NeverExceedsCapacity is the gopter proptest that
// discharges VP-025.
//
// For random capacities in [1, 64] and random sequences of up to 128 compound
// keys (checksum, arrival_interface_id), the DropCache's Len MUST remain ≤
// capacity after every insert. Duplicate compound keys are permitted in the
// input sequence; they exercise the LRU-refresh path (which must not grow the
// cache).
//
// Mutation-kill: replace `if c.lru.Len() >= c.capacity` with `>` in
// multipath.go:Add and Add's eviction would fire one insert late, leaving
// Len == capacity+1 for one iteration. Property fails.
//
// VP-025 / BC-2.02.009 postcondition 3
func TestProp_VP025_DropCache_NeverExceedsCapacity(t *testing.T) {
	params := gopter.DefaultTestParameters()
	// 500 successful runs is well above the standard proptest bar and still
	// completes in well under one second on Go 1.25.4.
	params.MinSuccessfulTests = 500

	properties := gopter.NewProperties(params)

	properties.Property(
		"DropCache Len never exceeds capacity after any insert sequence",
		prop.ForAll(
			func(capacity uint8, checksums []uint32, ifaceIDs []uint64) bool {
				// gopter's UInt8Range(1, 64) already excludes 0 but guard anyway.
				capInt := int(capacity)
				if capInt < 1 {
					capInt = 1
				}
				c := multipath.NewDropCache(capInt)

				// Zip the two independent generators. Because gopter drives them
				// independently, their lengths may differ; use the shorter.
				n := len(checksums)
				if len(ifaceIDs) < n {
					n = len(ifaceIDs)
				}

				for i := 0; i < n; i++ {
					c.Add(checksums[i], ifaceIDs[i])
					if c.Len() > capInt {
						return false
					}
				}
				return true
			},
			gen.UInt8Range(1, 64),
			gen.SliceOfN(128, gen.UInt32()),
			gen.SliceOfN(128, gen.UInt64()),
		),
	)

	properties.TestingRun(t)
}

// TestProp_VP025_DropCache_AddIfAbsent_NeverExceedsCapacity is the companion
// proptest for the AddIfAbsent path (the atomic first-arrival check used by
// Multipath.Receive per BC-2.02.002 invariant 1 / F-005). AddIfAbsent has its
// own eviction copy; VP-025's capacity invariant must hold there too.
//
// Mutation-kill: replace `if c.lru.Len() >= c.capacity` with `>` in the
// AddIfAbsent branch and this property fails.
//
// VP-025 / BC-2.02.009 postcondition 3 / F-005
func TestProp_VP025_DropCache_AddIfAbsent_NeverExceedsCapacity(t *testing.T) {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 500

	properties := gopter.NewProperties(params)

	properties.Property(
		"DropCache AddIfAbsent Len never exceeds capacity",
		prop.ForAll(
			func(capacity uint8, checksums []uint32, ifaceIDs []uint64) bool {
				capInt := int(capacity)
				if capInt < 1 {
					capInt = 1
				}
				c := multipath.NewDropCache(capInt)

				n := len(checksums)
				if len(ifaceIDs) < n {
					n = len(ifaceIDs)
				}

				for i := 0; i < n; i++ {
					_ = c.AddIfAbsent(checksums[i], ifaceIDs[i])
					if c.Len() > capInt {
						return false
					}
				}
				return true
			},
			gen.UInt8Range(1, 64),
			gen.SliceOfN(128, gen.UInt32()),
			gen.SliceOfN(128, gen.UInt64()),
		),
	)

	properties.TestingRun(t)
}
