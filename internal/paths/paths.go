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

// DegradedRTTThresholdMS is the EWMA RTT threshold in milliseconds above which
// a path is considered degraded (BC-2.02.003 postcondition 5; ARCH-03 §Degraded-Path Flag Design).
// The comparison is exclusive: degraded = ewmaRTTMS > DegradedRTTThresholdMS.
const DegradedRTTThresholdMS = 200.0

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

	// degraded is true when the EWMA RTT has converged above DegradedRTTThresholdMS
	// (BC-2.02.003 postcondition 5). Written only under mu.
	degraded bool

	// firstProbe is true until the first successful probe has been received.
	// On first arrival the RTT estimate is replaced outright (TCP RFC 6298
	// style) rather than EWMA-blended, so the conservative initial value does
	// not poison the EWMA for many probe intervals (BC-2.02.003 EC-003).
	firstProbe bool

	// hist is the fixed-bucket RTT histogram (ARCH-03 v1.6 §p99 RTT Accumulator).
	// Guarded by mu. Wired in S-5.02.
	hist rttHistogram
}

// NewPathTracker constructs a PathTracker with a conservative initial RTT
// (initialRTTMS) and zero loss. The EWMA smoothing factor alpha must satisfy
// 0 < alpha ≤ 1; a typical value is 0.125 (equivalent to a window of ~8 probes).
//
// BC-2.02.003 precondition 3: metrics are initialized with a high-RTT default
// on first connection.
func NewPathTracker(initialRTTMS float64, alpha float64) *PathTracker {
	if alpha <= 0 || alpha > 1 {
		panic("paths: NewPathTracker alpha must satisfy 0 < alpha <= 1")
	}
	return &PathTracker{
		ewmaAlpha:   alpha,
		ewmaRTTMS:   initialRTTMS,
		ewmaLossPct: 0,
		active:      true,
		firstProbe:  true,
	}
}

// resetRTT sets the RTT outright from a measured sample, clearing loss and
// miss counters and marking the path active. Called on first-probe arrival and
// on reactivation after consecutive misses (BC-2.02.003 EC-003 / postcondition 6
// v1.2). Caller must hold t.mu.
func (t *PathTracker) resetRTT(arrivalRTTMS float64) {
	t.ewmaRTTMS = arrivalRTTMS
	t.ewmaLossPct = 0
	t.consecutiveMisses = 0
	t.firstProbe = false
	t.active = true
	t.updateDegraded(arrivalRTTMS) // BC-2.02.003 PC-5: set degraded from reactivating RTT
	t.hist.record(arrivalRTTMS)    // S-5.02: feed histogram on first-probe/reactivation
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
		//
		// Reactivation (BC-2.02.003 postcondition 6 v1.2): if the path was
		// deactivated by consecutive misses, the first successful probe restores
		// it to the active set. First-probe semantics apply: RTT is reset outright
		// rather than EWMA-blended (shared invariant with the initial first-probe
		// path below — both use resetRTT to prevent divergence).
		if !t.active || t.firstProbe {
			t.resetRTT(arrivalRTTMS)
			return
		}
		t.ewmaRTTMS = t.ewmaAlpha*arrivalRTTMS + (1-t.ewmaAlpha)*t.ewmaRTTMS
		t.ewmaLossPct = (1 - t.ewmaAlpha) * t.ewmaLossPct
		t.consecutiveMisses = 0
		t.updateDegraded(t.ewmaRTTMS) // BC-2.02.003 PC-5: evaluate threshold after EWMA update
		t.hist.record(arrivalRTTMS)   // S-5.02: feed histogram on successful probe
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

// updateDegraded evaluates rttMS against DegradedRTTThresholdMS and updates
// t.degraded accordingly. Caller must hold t.mu.
//
// The comparison is exclusive: degraded = rttMS > DegradedRTTThresholdMS.
// Called at the OnProbe success path (after EWMA update) and at resetRTT
// (first-probe and reactivation paths) so the flag always reflects the
// current EWMA RTT (BC-2.02.003 postcondition 5; ARCH-03 §Degraded-Path Flag Design).
func (t *PathTracker) updateDegraded(rttMS float64) {
	t.degraded = rttMS > DegradedRTTThresholdMS
}

// IsDegraded reports whether the path's EWMA RTT has converged above
// DegradedRTTThresholdMS (BC-2.02.003 postcondition 5; ARCH-03 §Degraded-Path Flag Design).
// The flag is exclusive: IsDegraded returns true only when ewmaRTTMS > DegradedRTTThresholdMS.
func (t *PathTracker) IsDegraded() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.degraded
}

// rttHistogramBuckets defines the right edge (exclusive) of each bucket in milliseconds.
// 16 buckets covering 0–2000 ms (ARCH-03 v1.6 §p99 RTT Accumulator).
// Canonical layout: buckets 0–3 are 25ms wide; buckets 4–5 are 50ms wide;
// coarser buckets above 200ms; last bucket is ∞.
var rttHistogramBuckets = [16]float64{
	25, 50, 75, 100, 150, 200, 300, 400, 500, 750, 1000, 1250, 1500, 1750, 2000, 1e18,
}

// rttHistogram is a fixed-bucket latency histogram for per-path RTT samples.
// It lives in PathTracker and is guarded by PathTracker.mu.
// 16 buckets cover 0–2000+ ms (ARCH-03 v1.6 §p99 RTT Accumulator).
//
// Zero value is ready to use.
type rttHistogram struct {
	counts [16]uint64
	total  uint64
}

// bucketFor returns the bucket index for an RTT sample (arrivalRTTMS >= 0).
// Caller must hold the PathTracker mu.
//
// Iterates rttHistogramBuckets and returns the index of the first bucket whose
// right edge (exclusive) exceeds arrivalRTTMS. The last bucket catches anything
// beyond 2000ms via its infinity sentinel (1e18).
func bucketFor(arrivalRTTMS float64) int {
	for i, edge := range rttHistogramBuckets {
		if arrivalRTTMS < edge {
			return i
		}
	}
	return len(rttHistogramBuckets) - 1
}

// record adds one RTT sample to the histogram.
// Caller must hold the PathTracker mu.
func (h *rttHistogram) record(arrivalRTTMS float64) {
	h.counts[bucketFor(arrivalRTTMS)]++
	h.total++
}

// p99 returns the p99 RTT estimate in milliseconds.
// Returns 0 when total < 10 (caller should surface as "pending").
// Approximation: p99() <= true_p99 + max_bucket_width (ARCH-03 p99 RTT Accumulator).
// Caller must hold the PathTracker mu.
//
// Traverses h.counts from the lowest bucket upward, accumulating sample counts
// until the running sum reaches or exceeds 99% of total samples. The upper edge
// of that bucket is returned as the p99 estimate.
func (h *rttHistogram) p99() float64 {
	if h.total < 10 {
		return 0
	}
	// threshold is the minimum cumulative count needed to claim p99.
	// Integer arithmetic: ceil(99 * total / 100).
	threshold := (99*h.total + 99) / 100
	var cumulative uint64
	for i, count := range h.counts {
		cumulative += count
		if cumulative >= threshold {
			return rttHistogramBuckets[i]
		}
	}
	return rttHistogramBuckets[len(rttHistogramBuckets)-1]
}

// PathSnapshot is a consistent point-in-time copy of all PathTracker metrics.
// It is a value type (go.md rule 12: never return internal pointers from a locked accessor).
type PathSnapshot struct {
	// EWMARTTMs is the current EWMA-smoothed RTT in milliseconds.
	EWMARTTMs float64
	// LossPct is the current EWMA-smoothed loss percentage (0–100).
	LossPct float64
	// Active reports whether the path is in the active set.
	Active bool
	// Degraded reports whether the path's EWMA RTT exceeds DegradedRTTThresholdMS.
	Degraded bool
	// P99RTTMs is the p99 RTT in milliseconds. 0 when SampleCount < 10 (surface as "pending").
	// Wired from rttHistogram.p99() in S-5.02.
	P99RTTMs float64
	// SampleCount is the total number of RTT samples recorded in the histogram.
	// Used by callers to determine whether P99RTTMs is valid (≥10) or pending (<10).
	SampleCount uint64
}

// Snapshot returns a consistent copy of all PathTracker metrics under a single
// lock acquisition. The returned PathSnapshot is fully decoupled from internal
// state (go.md rule 12).
func (t *PathTracker) Snapshot() PathSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return PathSnapshot{
		EWMARTTMs:   t.ewmaRTTMS,
		LossPct:     t.ewmaLossPct,
		Active:      t.active,
		Degraded:    t.degraded,
		P99RTTMs:    t.hist.p99(), // S-5.02: wired from histogram; 0 when SampleCount < 10
		SampleCount: t.hist.total, // S-5.02: callers check ≥10 before trusting P99RTTMs
	}
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
		// Guard against nil Tracker: skip the entry rather than panic on the
		// routing hot path (CWE-476 / F-001). A nil Tracker has no metrics and
		// cannot be ranked; treat it as inactive.
		if rp.Tracker == nil {
			continue
		}
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
