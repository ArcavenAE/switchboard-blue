// Package admission — FailureCounter tracks per-source HMAC failure rates and
// emits E-ADM-017 admission alerts when the sliding-window threshold is crossed.
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-001 through AC-015.
package admission

import (
	"fmt"
	"sync"
	"time"
)

// Logger is a minimal logging interface for FailureCounter.
// Mirrors routing.Logger but lives in the admission package so the
// admission package does not import routing (ARCH-08 §4 no upward imports).
type Logger interface {
	// Log records a single log line.
	Log(msg string)
}

// FailureCounterOption is a functional option for NewFailureCounter.
type FailureCounterOption func(*FailureCounter)

// WithNow injects a clock function into FailureCounter, replacing the default
// time.Now().UTC(). Required for deterministic testing without wall-clock sleeps.
//
// Traces to S-W3.05 AC-003, AC-005, AC-008 (clock-seam requirement).
func WithNow(fn func() time.Time) FailureCounterOption {
	return func(c *FailureCounter) {
		c.now = fn
	}
}

// maxTrackedSources is the hard upper bound on the number of distinct source
// addresses tracked simultaneously. When a new source would exceed this limit,
// the LRU source (oldest most-recent failure timestamp) is evicted from both
// counts and firedAt before the new source is inserted.
//
// Prevents unbounded map growth (CWE-770). O(N) LRU scan is acceptable for V1.
// Traces to BC-2.05.005 EC-010; S-W3.05 AC-011.
const maxTrackedSources = 65536

// FailureCounter tracks per-source HMAC failure timestamps in a sliding window
// and emits E-ADM-017 exactly once per threshold crossing.
//
// Drain-only re-arm (BC-2.05.005 v1.6): after an alert fires, timestamps are no
// longer appended (append-skip). Re-arm triggers only when the sliding window
// fully empties (len==0 after trim). This guarantees the slice is bounded at
// threshold entries while an alert is active, and prevents spurious re-fires.
//
// All exported methods are safe for concurrent use.
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-001 through AC-017.
type FailureCounter struct {
	mu             sync.Mutex
	counts         map[string][]time.Time // per-srcAddr timestamp slices
	firedAt        map[string]time.Time   // time of last alert per srcAddr; zero = never fired / re-armed
	threshold      int
	windowDuration time.Duration
	logger         Logger
	now            func() time.Time // clock seam; defaults to time.Now().UTC()
}

// NewFailureCounter constructs a FailureCounter with the given threshold and
// sliding window duration. logger must not be nil. Optional FailureCounterOption
// values (e.g. WithNow) are applied after construction.
//
// NewFailureCounter panics if threshold < 1 or windowDuration <= 0. These are
// programmer-error guards: a zero/negative threshold or non-positive window is
// always a caller bug, not a runtime condition.
//
// Traces to BC-2.05.005 PC-3 v1.4 (constructor-arg validation); S-W3.05 AC-013.
func NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger, opts ...FailureCounterOption) *FailureCounter {
	if threshold < 1 {
		panic("admission: NewFailureCounter: threshold must be >= 1")
	}
	if windowDuration <= 0 {
		panic("admission: NewFailureCounter: windowDuration must be > 0")
	}
	if logger == nil {
		panic("admission: NewFailureCounter: logger must not be nil")
	}
	c := &FailureCounter{
		counts:         make(map[string][]time.Time),
		firedAt:        make(map[string]time.Time),
		threshold:      threshold,
		windowDuration: windowDuration,
		logger:         logger,
		now:            func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// RecordHMACFailure records a single HMAC verification failure from srcAddr.
//
// Under the mutex it:
//  1. Trims entries where timestamp < now()-windowDuration (strictly less-than;
//     boundary entries are kept — BC-2.05.005 EC-008, AC-008).
//  2. If post-trim count is zero and the key existed, deletes counts[srcAddr] and
//     firedAt[srcAddr] (dead-key eviction — prevents unbounded map growth from
//     inactive sources; AC-012).
//  3. Drain-only re-arm (BC-2.05.005 v1.6): if firedAt is set and len(keep)==0,
//     clear firedAt[srcAddr]. The "oldest surviving entry is newer than firedAt"
//     path is removed — it is dead code under append-skip (Step 5).
//  4. Evicts the LRU source before inserting a new srcAddr key if
//     len(counts) == maxTrackedSources (CWE-770; AC-011).
//  5. Append-skip: appends now() only when lastFire.IsZero() (not currently fired).
//     The slice is bounded at threshold entries while an alert is active (EC-011).
//  6. If post-append count >= threshold AND not yet fired since re-arm: captures
//     the alert message under the lock, then logs after unlock to avoid holding
//     the lock during I/O.
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-002 through AC-017.
func (c *FailureCounter) RecordHMACFailure(srcAddr string) {
	c.mu.Lock()

	now := c.now()
	cutoff := now.Add(-c.windowDuration)

	// Step 1: Trim stale entries — strictly-less-than comparison keeps boundary
	// entries (BC-2.05.005 EC-008; AC-008).
	existing := c.counts[srcAddr]
	keep := existing[:0]
	for _, ts := range existing {
		if !ts.Before(cutoff) { // keep if ts >= cutoff
			keep = append(keep, ts)
		}
	}

	// Step 2: Dead-key eviction — if the window drained fully for this source,
	// delete it from both maps so len(counts) reflects only live sources and a
	// future re-arm starts clean (AC-012).
	if len(keep) == 0 && existing != nil {
		delete(c.counts, srcAddr)
		delete(c.firedAt, srcAddr)
		keep = nil
	}

	// Step 3: Drain-only re-arm (BC-2.05.005 v1.6).
	// Re-arm when the window has drained completely (len(keep)==0 after trim).
	// Under append-skip (Step 5), no new timestamps are added while firedAt is set,
	// so the "oldest surviving entry is newer than firedAt" path is dead code and
	// is removed to match the reconciled spec exactly.
	lastFire := c.firedAt[srcAddr]
	if !lastFire.IsZero() {
		if len(keep) == 0 {
			// All pre-fire entries have aged out — safe to re-arm.
			delete(c.firedAt, srcAddr)
			lastFire = time.Time{} // re-arm: treat as not-yet-fired
		}
	}

	// Step 4: LRU source cap — before inserting a brand-new key, evict the
	// source whose most-recent failure is oldest if we are at capacity (AC-011).
	_, exists := c.counts[srcAddr]
	if !exists && len(c.counts) >= maxTrackedSources {
		c.evictLRU()
	}

	// Step 5: Append-skip per-source slice bound (BC-2.05.005 EC-011; AC-016).
	// Only append a new timestamp when not currently fired. While firedAt is set,
	// pre-fire entries cannot age out (append-skip keeps the slice from growing),
	// so the slice is bounded at threshold entries during an active alert.
	// Re-arm (Step 3) clears lastFire to zero, so the first call after drain
	// does append (the re-arm and append are both visible in the same call).
	if lastFire.IsZero() {
		keep = append(keep, now)
	}
	c.counts[srcAddr] = keep

	// Step 6: Emit E-ADM-017 on threshold crossing, exactly once per crossing.
	// Capture the message under lock; log after unlock to avoid holding the
	// mutex during logger I/O.
	var alertMsg string
	if len(keep) >= c.threshold && lastFire.IsZero() {
		c.firedAt[srcAddr] = now
		alertMsg = fmt.Sprintf(
			"E-ADM-017 HMAC failure rate alert: ≥%d failures in %.0fs from src %s",
			c.threshold,
			c.windowDuration.Seconds(),
			srcAddr,
		)
	}

	c.mu.Unlock()

	if alertMsg != "" {
		c.logger.Log(alertMsg)
	}
}

// evictLRU removes the source with the oldest most-recent failure timestamp
// from both counts and firedAt. Must be called with c.mu already held.
// O(N) scan is acceptable for V1 per product-owner adjudication (AC-011).
// Traces to BC-2.05.005 EC-010; S-W3.05 AC-011; CWE-770.
func (c *FailureCounter) evictLRU() {
	var lruKey string
	var lruTime time.Time
	for k, ts := range c.counts {
		if len(ts) == 0 {
			// Empty slice — evict immediately (treat as infinitely old).
			lruKey = k
			break
		}
		last := ts[len(ts)-1]
		if lruKey == "" || last.Before(lruTime) {
			lruKey = k
			lruTime = last
		}
	}
	if lruKey != "" {
		delete(c.counts, lruKey)
		delete(c.firedAt, lruKey)
	}
}

// Timestamps returns a value copy of the current in-window timestamp slice for
// srcAddr. Returns an empty (non-nil) slice if srcAddr has no recorded failures.
//
// This is a white-box inspection helper for tests. The returned slice is
// independent of internal state — mutations to it do not affect the counter
// (go.md rule 12).
func (c *FailureCounter) Timestamps(srcAddr string) []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	src := c.counts[srcAddr]
	out := make([]time.Time, len(src))
	copy(out, src)
	return out
}

// SourceCount returns the number of distinct source addresses currently tracked
// in the sliding window (i.e., sources with at least one non-evicted entry in
// counts). Returns an int copy — safe for concurrent use (go.md rule 12).
//
// Traces to BC-2.05.005 EC-010; S-W3.05 AC-011.
func (c *FailureCounter) SourceCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.counts)
}
