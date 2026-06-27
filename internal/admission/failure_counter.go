// Package admission — FailureCounter tracks per-source HMAC failure rates and
// emits E-ADM-017 admission alerts when the sliding-window threshold is crossed.
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-001 through AC-010.
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

// FailureCounter tracks per-source HMAC failure timestamps in a sliding window
// and emits E-ADM-017 exactly once per threshold crossing.
//
// Hysteresis: after an alert fires, it is suppressed until all in-window entries
// that were present at the time of the alert have been trimmed away — i.e., the
// oldest entry in the trimmed window is newer than the alert fire timestamp.
// Only at that point does the alert re-arm, allowing a fresh crossing to fire again.
//
// All exported methods are safe for concurrent use.
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-001 through AC-010.
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
func NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger, opts ...FailureCounterOption) *FailureCounter {
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
//  2. Checks hysteresis: if the oldest remaining entry is newer than the last
//     alert fire time (i.e., all "pre-fire" entries have been trimmed away),
//     re-arms the alert. Also re-arms when the window is completely empty.
//  3. Appends now().
//  4. If post-append count >= threshold AND not yet fired since re-arm: emits
//     E-ADM-017 via logger (fire-once-per-crossing, AC-004).
//
// Traces to BC-2.05.005 PC-3; S-W3.05 AC-002 through AC-010.
func (c *FailureCounter) RecordHMACFailure(srcAddr string) {
	c.mu.Lock()
	defer c.mu.Unlock()

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

	// Step 2: Hysteresis re-arm check.
	// Re-arm when the window has drained completely, OR when all surviving entries
	// post-trim are newer than the last fire time (the "old window" entries that
	// were present when the alert fired have all expired).
	lastFire := c.firedAt[srcAddr]
	if !lastFire.IsZero() {
		if len(keep) == 0 || keep[0].After(lastFire) {
			// Safe to re-arm: no entries from the previous alert window remain.
			delete(c.firedAt, srcAddr)
			lastFire = time.Time{} // re-arm: treat as not-yet-fired
		}
	}

	// Step 3: Append current timestamp.
	keep = append(keep, now)
	c.counts[srcAddr] = keep

	// Step 4: Emit E-ADM-017 on threshold crossing, exactly once per crossing.
	if len(keep) >= c.threshold && lastFire.IsZero() {
		c.firedAt[srcAddr] = now
		c.logger.Log(fmt.Sprintf(
			"E-ADM-017 HMAC failure rate alert: ≥%d failures in %.0fs from src %s",
			c.threshold,
			c.windowDuration.Seconds(),
			srcAddr,
		))
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
