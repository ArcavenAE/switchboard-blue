// Package multipath_test — hit-counter tests for BC-2.02.009 postcondition 2.
//
// # Build tag: dropcache_hits
//
// These tests reference DropCache.Hits() int64, which does NOT exist yet.
// They are gated behind the `dropcache_hits` build tag so that the regular
// `go test ./...` run compiles cleanly while the implementer adds Hits().
//
// # Implementer contract (required for this file to compile without the tag)
//
// Add the following to DropCache in internal/multipath/multipath.go:
//
//  1. A hits counter field (increment under the existing mutex, or atomically):
//     hits int64  // cumulative cache hits (AddIfAbsent or Add on already-present key)
//
//  2. Increment in AddIfAbsent when the key is already present (the "cache hit" path):
//     // already present → hit; move to front and return false
//     c.lru.MoveToFront(elem)
//     c.hits++
//     return false
//
//  3. Increment in Add when the key is already present (re-add of a known key = hit):
//     // already present → move to front; count as a hit
//     c.lru.MoveToFront(elem)
//     c.hits++
//     return
//
//  4. Add a Hits() int64 accessor:
//     func (c *DropCache) Hits() int64 {
//         c.mu.Lock()
//         defer c.mu.Unlock()
//         return c.hits
//     }
//
// Once Hits() is added the implementer MUST remove the `//go:build dropcache_hits`
// tag from this file so these tests run as part of the regular suite.
//
// # Red Gate status
//
// This file does NOT compile without the build tag because Hits() does not exist.
// That is the correct Red state. Run with the tag to see compilation fail before
// Hits() is implemented:
//
//	go test -tags dropcache_hits ./internal/multipath/...
//
// After Hits() is implemented (and the tag removed), all tests in this file MUST
// pass under `go test -race ./internal/multipath/...`.

package multipath_test

import (
	"sync"
	"testing"

	"github.com/arcavenae/switchboard/internal/multipath"
)

// TestBC_2_02_009_DropCache_HitCounterIncremented verifies that DropCache.Hits()
// starts at 0, does NOT increment on a first-arrival miss, and increments by
// exactly 1 on each subsequent cache hit (suppressed duplicate) via both the
// AddIfAbsent and Add paths.
//
// Uses canonical test vectors from BC-2.02.009:
//   - "Frame with checksum 0xABCD arrives; cache empty → forwarded"   (miss, Hits()==0)
//   - "Same frame (checksum 0xABCD) arrives again → dropped; hit counter incremented"
//
// Pass-2 ruling F-H2 (FIX-IN-S4.01) / BC-2.02.009 postcondition 2 / canonical test vector 2
func TestBC_2_02_009_DropCache_HitCounterIncremented(t *testing.T) {
	t.Parallel()

	dc := multipath.NewDropCache(10)

	// --- miss path via Add ---

	// First Add: key not present → miss, no hit recorded.
	dc.Add(0xABCD, 1)
	if got := dc.Hits(); got != 0 {
		t.Errorf("after Add (miss): Hits()=%d, want 0", got)
	}

	// --- hit path via AddIfAbsent ---

	// AddIfAbsent on already-present key → cache hit; counter must increment.
	if first := dc.AddIfAbsent(0xABCD, 1); first {
		t.Error("AddIfAbsent on present key: want false (duplicate), got true (first-arrival)")
	}
	if got := dc.Hits(); got != 1 {
		t.Errorf("after first AddIfAbsent hit: Hits()=%d, want 1", got)
	}

	// Second AddIfAbsent on the same key → second hit.
	if first := dc.AddIfAbsent(0xABCD, 1); first {
		t.Error("second AddIfAbsent on same key: want false (duplicate), got true")
	}
	if got := dc.Hits(); got != 2 {
		t.Errorf("after second AddIfAbsent hit: Hits()=%d, want 2", got)
	}

	// AddIfAbsent on a distinct key (first-arrival / miss) → counter must NOT change.
	if first := dc.AddIfAbsent(0xEEEE, 1); !first {
		t.Error("AddIfAbsent on absent key 0xEEEE: want true (first-arrival), got false")
	}
	if got := dc.Hits(); got != 2 {
		t.Errorf("after AddIfAbsent miss (new key): Hits()=%d, want 2 (unchanged)", got)
	}

	// --- hit path via Add (re-add of already-present key) ---

	// Re-adding 0xABCD via Add → the key is already present → this is a hit.
	dc.Add(0xABCD, 1)
	if got := dc.Hits(); got != 3 {
		t.Errorf("after Add on already-present key (hit): Hits()=%d, want 3", got)
	}

	// Add on the new key 0xEEEE (already present from AddIfAbsent above) → hit.
	dc.Add(0xEEEE, 1)
	if got := dc.Hits(); got != 4 {
		t.Errorf("after Add on already-present key 0xEEEE: Hits()=%d, want 4", got)
	}
}

// TestBC_2_02_009_DropCache_HitCounterConcurrent verifies that DropCache.Hits()
// is race-safe under concurrent access. N goroutines each Add the shared key
// once (first Add is a miss; subsequent Adds on the same key are hits), then
// repeatedly call AddIfAbsent (all return false = hit). After all goroutines
// complete, Hits() must equal the total number of suppressed duplicates.
//
// Run under `go test -race` — a missing or incorrectly-positioned counter
// increment will produce a data race report.
//
// Pass-2 ruling F-H2 (FIX-IN-S4.01) / BC-2.02.009 postcondition 2 concurrent-safety
func TestBC_2_02_009_DropCache_HitCounterConcurrent(t *testing.T) {
	// Not parallel at outer level — inner goroutines provide the concurrency.

	const goroutines = 8
	const hitsPerGoroutine = 20 // each goroutine calls AddIfAbsent this many times on the shared key

	dc := multipath.NewDropCache(100)

	// Seed the shared key so every subsequent AddIfAbsent is a hit.
	const sharedChecksum = uint32(0x1234)
	const sharedIface = uint64(1)
	dc.Add(sharedChecksum, sharedIface)

	// Verify seeding did not count as a hit.
	if got := dc.Hits(); got != 0 {
		t.Fatalf("before concurrent phase: Hits()=%d, want 0 (seed Add was a miss)", got)
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range hitsPerGoroutine {
				// Key is already present → every call is a cache hit.
				if first := dc.AddIfAbsent(sharedChecksum, sharedIface); first {
					// This must never happen — the key was seeded before goroutines started.
					t.Errorf("AddIfAbsent on seeded key returned true (first-arrival); expected false (hit)")
				}
			}
		}()
	}

	wg.Wait()

	// Each goroutine performed hitsPerGoroutine hits.
	want := int64(goroutines * hitsPerGoroutine)
	if got := dc.Hits(); got != want {
		t.Errorf("concurrent Hits(): got %d, want %d (goroutines=%d × hitsPerGoroutine=%d)",
			got, want, goroutines, hitsPerGoroutine)
	}
}
