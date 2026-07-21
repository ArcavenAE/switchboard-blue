// relay_rate_cap.go implements the per-(SVTNID, NodeAddr) relay dispatch rate
// cap (SEC-DW-09). It lives in its own file because it is stateful; keeping it
// separate from discovery_relay_wire.go (pure assembly + stateless dispatch)
// maintains the single-responsibility boundary introduced in Task 6.

package main

import (
	"sync"
	"time"
)

// relayRateKey is the composite map key for the per-(SVTNID, NodeAddr) relay
// dispatch rate cap. Mirrors the lastSeenKey pattern in
// internal/discovery/discovery_wire.go — same composite (SVTN, NodeAddr)
// identity, different purpose (relay rate limiting rather than replay discard).
type relayRateKey struct {
	svtnID   [16]byte
	nodeAddr [8]byte
}

// maxRelayRateCapEntries is the upper bound on the relayRateCap.last map size
// (SEC-DW-10, map-bounding-ruling.md Decision 1). When len(c.last) exceeds
// half this value, allow() runs an amortized prune sweep that evicts all
// entries whose stored timestamp is older than c.interval (stale by the same
// criterion the rate-cap decision logic itself uses).
const maxRelayRateCapEntries = 65536

// relayRateCap enforces a ~1/sec per-(SVTNID, NodeAddr) rate cap on relay
// dispatch (AC-018, SEC-DW-09). Multiple per-SVTN listener goroutines (Task 6d)
// will share one instance, so all methods are concurrency-safe via a single
// sync.Mutex that guards both the timestamp map and the suppression counter
// together — the two fields must be updated atomically.
//
// The now field is injectable for deterministic test control, mirroring the
// tokenBucket.now pattern in internal/discovery/discovery_wire.go.
//
// The maxEntries field overrides the production cap for tests that need a
// small cap to avoid iterating 65536 times under the race detector. Production
// callers use newRelayRateCap() which leaves maxEntries at zero, meaning
// maxRelayRateCapEntries is used. Tests use newRelayRateCapWithMax(n) to set
// a small cap.
type relayRateCap struct {
	mu         sync.Mutex
	last       map[relayRateKey]time.Time
	suppCount  uint64
	interval   time.Duration
	now        func() time.Time
	maxEntries int // 0 means use maxRelayRateCapEntries
}

// newRelayRateCap returns a relay rate cap with a 1-second interval and
// time.Now as the default clock source. The now field may be replaced
// post-construction to inject a deterministic clock for testing.
func newRelayRateCap() *relayRateCap {
	return &relayRateCap{
		last:     make(map[relayRateKey]time.Time),
		interval: time.Second,
		now:      time.Now,
	}
}

// newRelayRateCapWithMax returns a relay rate cap with a custom maxEntries cap.
// Intended for tests that need a small cap to avoid 65536-iteration loops under
// the race detector. Production code uses newRelayRateCap().
func newRelayRateCapWithMax(maxEntries int) *relayRateCap {
	return &relayRateCap{
		last:       make(map[relayRateKey]time.Time),
		interval:   time.Second,
		now:        time.Now,
		maxEntries: maxEntries,
	}
}

// capLimit returns the effective maxEntries for this cap instance.
func (c *relayRateCap) capLimit() int {
	if c.maxEntries > 0 {
		return c.maxEntries
	}
	return maxRelayRateCapEntries
}

// allow reports whether a relay dispatch for (svtnID, nodeAddr) should proceed.
//
//   - First call for a key (no prior entry): returns true, records now() as
//     the key's last-allowed timestamp.
//   - Subsequent call within the 1s window (now()-last < interval): returns
//     false (silent drop per SEC-DW-09 postcondition 2), increments the
//     suppression counter, and does NOT update the timestamp.
//   - Call at or after the interval boundary (now()-last >= interval): returns
//     true, updates the timestamp to now(). The >= comparison is intentional —
//     an arrival at exactly the 1s mark is allowed, not dropped, which matches
//     AC-018's "~1/sec" framing (a strict > would silently shift the window).
//
// The suppression counter is non-gating: nothing inside allow (or relayDispatch)
// consults it to alter the drop decision. It exists solely as an observable
// diagnostic (AC-018 postcondition 3, SEC-DW-09).
func (c *relayRateCap) allow(svtnID [16]byte, nodeAddr [8]byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := relayRateKey{svtnID: svtnID, nodeAddr: nodeAddr}
	t := c.now()

	// Amortized prune-by-age sweep (SEC-DW-10, map-bounding-ruling.md Decision 1):
	// when the map exceeds half the cap, delete every entry whose stored timestamp
	// is stale (older than c.interval). Uses the same staleness criterion as the
	// allow/deny decision below for consistency. O(N) but triggered at most once
	// per cap/2 insertions, so amortised cost is O(1) per call.
	if len(c.last) > c.capLimit()/2 {
		for k, storedTS := range c.last {
			if t.Sub(storedTS) >= c.interval {
				delete(c.last, k)
			}
		}
	}

	if last, seen := c.last[key]; seen && t.Sub(last) < c.interval {
		// Within the cap window: silent drop. Increment the non-gating counter
		// so operators can observe suppression volume without any gate side-effect.
		c.suppCount++
		return false
	}

	// Either first call for this key or the interval has elapsed — record the
	// timestamp and allow the relay dispatch.
	c.last[key] = t
	return true
}

// suppressed returns the total count of allow()==false results since the cap
// was created. Non-gating (SEC-DW-09 postcondition 3): the returned value is
// observable for diagnostics but is never consulted by the dispatch path.
func (c *relayRateCap) suppressed() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.suppCount
}
