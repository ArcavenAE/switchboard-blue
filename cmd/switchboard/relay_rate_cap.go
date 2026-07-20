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

// relayRateCap enforces a ~1/sec per-(SVTNID, NodeAddr) rate cap on relay
// dispatch (AC-018, SEC-DW-09). Multiple per-SVTN listener goroutines (Task 6d)
// will share one instance, so all methods are concurrency-safe via a single
// sync.Mutex that guards both the timestamp map and the suppression counter
// together — the two fields must be updated atomically.
//
// The now field is injectable for deterministic test control, mirroring the
// tokenBucket.now pattern in internal/discovery/discovery_wire.go.
type relayRateCap struct {
	mu        sync.Mutex
	last      map[relayRateKey]time.Time
	suppCount uint64
	interval  time.Duration
	now       func() time.Time
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
