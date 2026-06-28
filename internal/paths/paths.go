// Package paths implements per-path RTT/loss quality tracking for the
// Switchboard routing engine. Paths are ranked by a composite EWMA score
// derived from measured round-trip time and packet-loss fraction so that
// the two fastest paths can be selected for duplicate-and-race forwarding
// (BC-2.02.001, BC-2.02.003).
//
// This package is pure-core: it performs no I/O and holds no network state.
// All side-effects (probing, timers) are owned by the caller.
package paths

import (
	"errors"
	"sort"
	"sync"
)

// DefaultLossWeight is the coefficient applied to the loss fraction in the
// composite path score formula (ARCH-03: loss_weight = 10).
const DefaultLossWeight = 10.0

// consecutiveMissThreshold is the number of consecutive missed keepalives
// required to mark a path inactive (BC-2.02.003 postcondition 6).
const consecutiveMissThreshold = 3

// ErrNoActivePaths is returned by Rank when the tracker has no paths in the
// active set (all paths have been removed due to consecutive missed keepalives).
var ErrNoActivePaths = errors.New("paths: no active paths")

// Score is the composite quality score for a single path, computed from its
// current EWMA RTT and loss estimates.
// Lower score = better path (ARCH-03 ranking formula).
type Score float64

// PathScore computes the composite quality score for a path given its current
// EWMA RTT (milliseconds) and loss percentage (0–100).
//
// Formula (ARCH-03):
//
//	score = rtt_ewma_ms * (1 + loss_ewma_fraction * loss_weight)
//
// where loss_weight = DefaultLossWeight (10) and loss_ewma_fraction = loss_pct/100.
//
// Lower score is better. Ranking by PathScore is deterministic and transitive
// (BC-2.02.003 postcondition 3, AC-001).
func PathScore(rttMS float64, lossPct float64) Score {
	return Score(rttMS * (1 + (lossPct/100)*DefaultLossWeight))
}

// PathTracker maintains the EWMA RTT and loss estimate for a single path.
// It is safe for concurrent use.
//
// Zero value is not usable; construct via NewPathTracker.
type PathTracker struct {
	mu sync.Mutex

	// ewmaAlpha is the smoothing factor for the EWMA update (0 < alpha ≤ 1).
	ewmaAlpha float64

	// ewmaRTTMS is the current EWMA-smoothed RTT in milliseconds.
	ewmaRTTMS float64

	// ewmaLossPct is the current EWMA-smoothed loss percentage (0–100).
	ewmaLossPct float64

	// consecutiveMisses counts consecutive missed keepalives for this path.
	consecutiveMisses int

	// active reports whether this path is in the active set.
	active bool

	// firstProbe is true until the first successful probe has been received.
	// On first arrival the RTT estimate is replaced outright (TCP RFC 6298
	// style) rather than EWMA-blended, so the conservative initial value does
	// not poison the EWMA for many probe intervals (BC-2.02.003 EC-003).
	firstProbe bool
}

// NewPathTracker constructs a PathTracker with a conservative initial RTT
// (initialRTTMS) and zero loss. The EWMA smoothing factor alpha must satisfy
// 0 < alpha ≤ 1; a typical value is 0.125 (equivalent to a window of ~8 probes).
//
// BC-2.02.003 precondition 3: metrics are initialized with a high-RTT default
// on first connection.
func NewPathTracker(initialRTTMS float64, alpha float64) *PathTracker {
	return &PathTracker{
		ewmaAlpha:   alpha,
		ewmaRTTMS:   initialRTTMS,
		ewmaLossPct: 0,
		active:      true,
		firstProbe:  true,
	}
}

// OnProbe updates the EWMA RTT and loss estimate for the path based on a
// single keepalive probe result. arrivalRTTMS is the measured round-trip time
// in milliseconds; lossEvent is true when the probe response was not received
// (the expected keepalive was missed).
//
// After 3 probe arrivals (lossEvent=false) the score converges toward the true
// measured RTT (BC-2.02.003 postcondition 1, AC-002).
//
// Consecutive missed keepalives are tracked; once the miss count reaches
// consecutiveMissThreshold the path is marked inactive (BC-2.02.003 postcondition 6).
func (t *PathTracker) OnProbe(arrivalRTTMS float64, lossEvent bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if lossEvent {
		// EWMA update for loss: treat a miss as 100% loss sample.
		t.ewmaLossPct = t.ewmaAlpha*100 + (1-t.ewmaAlpha)*t.ewmaLossPct
		t.consecutiveMisses++
		if t.consecutiveMisses >= consecutiveMissThreshold {
			t.active = false
		}
	} else {
		// Successful probe: update RTT EWMA, zero out loss sample, reset misses.
		// On first successful probe, replace the conservative initial RTT outright
		// (RFC 6298 style) so the high-RTT default does not slow convergence for
		// many probe intervals (BC-2.02.003 EC-003: "after first measured RTT,
		// path ranked appropriately").
		if t.firstProbe {
			t.ewmaRTTMS = arrivalRTTMS
			t.firstProbe = false
		} else {
			t.ewmaRTTMS = t.ewmaAlpha*arrivalRTTMS + (1-t.ewmaAlpha)*t.ewmaRTTMS
		}
		t.ewmaLossPct = (1 - t.ewmaAlpha) * t.ewmaLossPct
		t.consecutiveMisses = 0
	}
}

// Score returns the current composite quality score for this path.
// Delegates to PathScore using the tracker's current EWMA estimates.
func (t *PathTracker) Score() Score {
	t.mu.Lock()
	defer t.mu.Unlock()
	return PathScore(t.ewmaRTTMS, t.ewmaLossPct)
}

// IsActive reports whether the path is still in the active set (i.e., has not
// accumulated consecutiveMissThreshold consecutive missed keepalives).
func (t *PathTracker) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.active
}

// RTT returns the current EWMA RTT estimate in milliseconds.
func (t *PathTracker) RTT() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ewmaRTTMS
}

// LossPct returns the current EWMA loss percentage estimate (0–100).
func (t *PathTracker) LossPct() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ewmaLossPct
}

// RankedPath associates a caller-supplied path identifier with its current
// quality score for use by Rank.
type RankedPath struct {
	// ID is an opaque caller-supplied identifier for the path (e.g. interface
	// index or peer address hash). Used as tiebreak key when scores are equal
	// (AC-002 / EC-002).
	ID uint64
	// Tracker is the PathTracker whose current score is evaluated.
	Tracker *PathTracker
}

// Rank returns the active paths from candidates ordered by ascending score
// (best first). Paths whose IsActive() returns false are excluded.
// If no paths are active, ErrNoActivePaths is returned.
//
// Ties in score are broken by ascending RankedPath.ID for deterministic
// ordering (EC-002).
//
// The returned slice is a fresh allocation; mutations do not affect candidates.
func Rank(candidates []RankedPath) ([]RankedPath, error) {
	// Snapshot active paths and their scores.
	type scoredPath struct {
		rp    RankedPath
		score Score
	}

	active := make([]scoredPath, 0, len(candidates))
	for _, rp := range candidates {
		if rp.Tracker.IsActive() {
			active = append(active, scoredPath{rp: rp, score: rp.Tracker.Score()})
		}
	}

	if len(active) == 0 {
		return nil, ErrNoActivePaths
	}

	sort.Slice(active, func(i, j int) bool {
		if active[i].score != active[j].score {
			return active[i].score < active[j].score
		}
		// Tiebreak: ascending ID for determinism (EC-002).
		return active[i].rp.ID < active[j].rp.ID
	})

	result := make([]RankedPath, len(active))
	for i, sp := range active {
		result[i] = sp.rp
	}
	return result, nil
}
