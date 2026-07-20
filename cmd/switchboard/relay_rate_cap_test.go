// relay_rate_cap_test.go — RED-first (Step 4.5) map-bounding tests for
// relayRateCap.last (SEC-DW-10, map-bounding-ruling.md Decision 1).
//
// All three tests MUST FAIL before the implementation lands — the bounding
// constant (maxRelayRateCapEntries) and the prune-by-age sweep inside allow()
// do NOT yet exist. Tests use white-box access to c.last (same package: main)
// and a test-local cap constant to avoid referencing the not-yet-existent
// production symbol, so the package compiles and failures manifest as
// assertion failures, not compile errors.
//
// Ruling: .factory/decisions/S-BL.DISCOVERY-WIRE-map-bounding-ruling.md v1.0
// Spec:   S-BL.DISCOVERY-WIRE.md v2.21, SEC-DW-10
package main

import (
	"testing"
	"time"
)

// testRelayRateCapMax is the cap value the ruling mandates
// (maxRelayRateCapEntries = 65536, Decision 1). Written as a test-local
// literal so the package compiles before the production constant exists;
// the implementer's const must match this value.
const testRelayRateCapMax = 65536

// newFakeClock returns a func() time.Time whose returned value is controlled
// by the caller via the returned *time.Time pointer.
func newFakeClock(initial time.Time) (func() time.Time, *time.Time) {
	ts := initial
	return func() time.Time { return ts }, &ts
}

// makeRelayRateKey returns a distinct relayRateKey for the given index,
// spread across svtnID and nodeAddr fields to ensure uniqueness.
func makeRelayRateKey(i int) relayRateKey {
	var k relayRateKey
	k.svtnID[0] = byte(i)
	k.svtnID[1] = byte(i >> 8)
	k.svtnID[2] = byte(i >> 16)
	k.nodeAddr[0] = byte(i >> 24)
	return k
}

// TestRelayRateCap_MapBounded_AfterStaleEntries verifies that allow() bounds
// len(c.last) to ≤ testRelayRateCapMax after the prune-by-age sweep is
// triggered (map size > maxRelayRateCapEntries/2, all entries stale).
//
// RED-first anti-vacuity: today's allow() has no prune sweep — after
// testRelayRateCapMax+1 distinct inserts the map will contain exactly
// testRelayRateCapMax+1 entries, causing the ≤ assertion to FAIL.
//
// White-box pre-loading: we call allow() for each key (not direct map
// manipulation) so the assertion exercises the real production code path.
// At testRelayRateCapMax+1 calls this is ~65537 allow() invocations — cheap
// because each is a single map write with no I/O.
func TestRelayRateCap_MapBounded_AfterStaleEntries(t *testing.T) {
	t.Parallel()

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clk, ts := newFakeClock(base)

	c := newRelayRateCap()
	c.now = clk

	// Insert testRelayRateCapMax+1 distinct keys at time T=base.
	// Each allow() sees no prior entry → returns true, records base.
	for i := 0; i <= testRelayRateCapMax; i++ {
		k := makeRelayRateKey(i)
		c.allow(k.svtnID, k.nodeAddr)
	}

	// Advance clock to T+2s — all entries are now older than c.interval (1s),
	// so every entry in c.last is stale per the ruling's prune semantics.
	*ts = base.Add(2 * time.Second)

	// Call allow() for one more distinct key to trigger the prune sweep.
	extra := makeRelayRateKey(testRelayRateCapMax + 1)
	c.allow(extra.svtnID, extra.nodeAddr)

	c.mu.Lock()
	size := len(c.last)
	c.mu.Unlock()

	// After the prune sweep the map must be bounded.
	if size > testRelayRateCapMax {
		t.Errorf("len(c.last) = %d after stale prune, want ≤ %d (map is unbounded — prune not yet implemented)",
			size, testRelayRateCapMax)
	}
	// Anti-vacuity: the map must not be completely empty (the triggering call's
	// entry must have been written).
	if size == 0 {
		t.Error("len(c.last) = 0 after prune+insert — the triggering call's entry was not written")
	}
}

// TestRelayRateCap_StalePrunedKey_ReAllowed verifies that pruning is
// semantically lossless: a key evicted as stale is treated as cold on its
// next allow() call, returning true exactly as if the entry had never existed.
//
// RED-first: without the prune sweep, after clock advance the key's stale
// entry remains — but that path actually returns true (VP-B: stale entry
// is overwritten). This test would PASS vacuously on that basis… UNLESS
// we verify that the prune sweep correctly removes the entry AND the
// subsequent allow() returns true. We verify the map actually ran the prune
// by checking that the pruned key's entry was removed before the re-allow,
// then re-added. Without the prune code the map never removes the entry, so
// we add an assertion that the entry was absent before the re-allow call
// (which requires the prune to have run). We achieve this by pre-loading the
// map to > testRelayRateCapMax/2 entries so the prune threshold is crossed,
// then checking the specific key is absent after prune before re-allowing.
func TestRelayRateCap_StalePrunedKey_ReAllowed(t *testing.T) {
	t.Parallel()

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clk, ts := newFakeClock(base)

	c := newRelayRateCap()
	c.now = clk

	// Insert key K at time T.
	keyK := makeRelayRateKey(0)
	gotFirst := c.allow(keyK.svtnID, keyK.nodeAddr)
	if !gotFirst {
		t.Fatal("first allow(K): want true, got false (test setup failed)")
	}

	// Advance clock past interval so K's entry is stale.
	*ts = base.Add(2 * time.Second)

	// Pre-load enough entries (> testRelayRateCapMax/2) so the prune threshold
	// is crossed, forcing the sweep to run and evict K's now-stale entry.
	// White-box: direct map pre-loading to avoid 32769 allow() calls for setup.
	// These entries are written with the STALE timestamp (base) so they too
	// will be pruned — only K matters for the behavioral assertion.
	c.mu.Lock()
	for i := 1; i <= testRelayRateCapMax/2+1; i++ {
		k := makeRelayRateKey(i)
		c.last[k] = base // stale timestamp: will be pruned alongside K
	}
	c.mu.Unlock()

	// Trigger the sweep by calling allow() for a new distinct key.
	trigger := makeRelayRateKey(testRelayRateCapMax + 100)
	c.allow(trigger.svtnID, trigger.nodeAddr)

	// After the prune, K's entry must be gone (it was stale).
	c.mu.Lock()
	_, kStillPresent := c.last[relayRateKey{svtnID: keyK.svtnID, nodeAddr: keyK.nodeAddr}]
	c.mu.Unlock()
	if kStillPresent {
		t.Error("key K still present after stale prune sweep — prune not yet implemented")
	}

	// Re-allow K: must return true (cold-start semantics, AC-018 / VP-B).
	gotReAllow := c.allow(keyK.svtnID, keyK.nodeAddr)
	if !gotReAllow {
		t.Error("allow(K) after prune: want true (cold-start after eviction), got false")
	}
}

// TestRelayRateCap_ActiveKeys_NotPruned verifies that an active key — one
// whose last-allowed timestamp is within c.interval — is NOT pruned even
// when the prune sweep runs, and that it remains rate-capped (a second
// allow() within the interval returns false).
//
// RED-first: without the prune sweep the active key is trivially not pruned
// (nothing is pruned), so this test PASSES vacuously today. To make it
// discriminating we assert that the map size after the sweep is bounded
// (≤ testRelayRateCapMax), which will FAIL today because no prune runs and
// the map grows to > testRelayRateCapMax/2+1 entries after pre-loading.
func TestRelayRateCap_ActiveKeys_NotPruned(t *testing.T) {
	t.Parallel()

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clk, ts := newFakeClock(base)

	c := newRelayRateCap()
	c.now = clk

	// Insert active key K at time T.
	keyK := makeRelayRateKey(0)
	if got := c.allow(keyK.svtnID, keyK.nodeAddr); !got {
		t.Fatal("first allow(K): want true (test setup)")
	}

	// Advance clock by 500ms — within the 1s interval, so K is still active.
	*ts = base.Add(500 * time.Millisecond)

	// Pre-load > testRelayRateCapMax/2 stale entries (written at base, now
	// older than interval at T+500ms+something… actually 500ms < interval=1s,
	// so entries at base are NOT yet stale relative to T+500ms).
	// We need entries that ARE stale. Advance clock further for the pre-loading
	// trick: write stale entries with a timestamp 2s in the past relative to
	// the sweep's clock. We achieve this by writing them with base-2s timestamp.
	staleTS := base.Add(-2 * time.Second)
	c.mu.Lock()
	for i := 1; i <= testRelayRateCapMax/2+1; i++ {
		k := makeRelayRateKey(i)
		c.last[k] = staleTS // will be pruned (older than interval at T+500ms)
	}
	c.mu.Unlock()

	// Call allow(K) again at T+500ms: within the interval, should return false.
	gotWithin := c.allow(keyK.svtnID, keyK.nodeAddr)
	if gotWithin {
		t.Error("allow(K) at T+500ms (within interval): want false (still rate-capped), got true")
	}

	// The prune sweep should have fired (map size was > testRelayRateCapMax/2).
	// After the sweep the stale entries should be gone; K must still be present
	// (it was active — its timestamp base is only 500ms old, within interval).
	c.mu.Lock()
	_, kPresent := c.last[relayRateKey{svtnID: keyK.svtnID, nodeAddr: keyK.nodeAddr}]
	size := len(c.last)
	c.mu.Unlock()

	if !kPresent {
		t.Error("active key K was pruned — prune must NOT remove keys within the interval")
	}
	// Discriminating assertion: after the prune sweep runs (triggered because
	// len(c.last) > testRelayRateCapMax/2 = 32768), all stale entries must be
	// deleted. Only K (with fresh timestamp base) survives. So post-sweep size
	// must be ≤ 2 (K + the trigger key).
	// Without the prune, size = testRelayRateCapMax/2+2 = 32770 → FAILS this
	// assertion, making the test discriminating.
	const maxPostSweepSize = 2 // K (fresh) + trigger key
	if size > maxPostSweepSize {
		t.Errorf("len(c.last) = %d after prune sweep, want ≤ %d (stale entries must be removed; prune not yet implemented)",
			size, maxPostSweepSize)
	}
}
