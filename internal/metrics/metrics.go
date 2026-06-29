// Package metrics implements the session quality indicator (green/yellow/red)
// derived from measured path RTT and packet loss (BC-2.06.001, BC-2.06.002).
//
// Classification thresholds (NFR-001; ARCH-INDEX F-008):
//
//	Green:  RTT p99 ≤ 100 ms AND loss ≤ 5 %
//	Yellow: RTT p99 ≤ 500 ms AND loss ≤ 20 %  (and not green)
//	Red:    RTT p99 > 500 ms OR loss > 20 %
//
// Hysteresis constant (ARCH-INDEX F-021): 3 consecutive measurements.
package metrics

import "sync"

// Quality represents the three-level session quality signal.
type Quality int

const (
	// Green means the path is within latency and loss budgets (BC-2.06.001 postcondition 2).
	Green Quality = iota
	// Yellow means the path is degraded but functional (BC-2.06.001 postcondition 3).
	Yellow
	// Red means the path is significantly degraded or unavailable (BC-2.06.001 postcondition 4).
	Red
)

// String returns the human-readable label for a Quality value.
func (q Quality) String() string {
	switch q {
	case Green:
		return "green"
	case Yellow:
		return "yellow"
	case Red:
		return "red"
	default:
		return "unknown"
	}
}

// Thresholds encode the NFR-001 quality classification boundaries.
// All constants are inclusive on the green/yellow side (≤).
const (
	// GreenRTTMs is the maximum RTT (inclusive) for a Green classification.
	GreenRTTMs float64 = 100.0
	// YellowRTTMs is the maximum RTT (inclusive) for a Yellow classification.
	YellowRTTMs float64 = 500.0
	// GreenLossPct is the maximum packet-loss percentage (inclusive) for Green.
	GreenLossPct float64 = 5.0
	// YellowLossPct is the maximum packet-loss percentage (inclusive) for Yellow.
	YellowLossPct float64 = 20.0
	// HysteresisCount is the number of consecutive measurements required to
	// upgrade the quality indicator (ARCH-INDEX F-021).
	HysteresisCount = 3
)

// QualityIndicator computes and maintains the green/yellow/red session quality
// signal with hysteretic upgrade logic (BC-2.06.001 invariant 3, BC-2.06.002).
//
// Zero value is not usable; construct via NewQualityIndicator.
// All exported methods are safe for concurrent use.
type QualityIndicator struct {
	mu sync.Mutex

	// current is the current quality level displayed to the operator.
	current Quality

	// consecutiveCount tracks how many consecutive measurements have produced
	// the same candidate level. Used to enforce hysteresis on upgrade.
	consecutiveCount int

	// candidate is the quality level computed from the most recent measurement,
	// before hysteresis filtering is applied.
	candidate Quality

	// missingFrameCount counts consecutive missing-frame events (BC-2.06.002).
	missingFrameCount int
}

// NewQualityIndicator returns a QualityIndicator initialised to Green.
// The caller supplies no configuration because thresholds are fixed per NFR-001.
func NewQualityIndicator() *QualityIndicator {
	return &QualityIndicator{
		current:   Green,
		candidate: Green,
	}
}

// Current returns the current quality level.
func (qi *QualityIndicator) Current() Quality {
	qi.mu.Lock()
	defer qi.mu.Unlock()
	return qi.current
}

// Update classifies (rttMs, lossPct) against the NFR-001 thresholds and applies
// hysteresis to the quality indicator.
//
// Downgrade (green→yellow or yellow→red) is immediate.
// Upgrade (yellow→green or red→yellow) requires HysteresisCount consecutive
// measurements at the new level (BC-2.06.001 invariant 3; AC-002).
//
// BC-2.06.001 postconditions 2–4; AC-001, AC-002, AC-004.
func (qi *QualityIndicator) Update(rttMs float64, lossPct float64) {
	qi.mu.Lock()
	defer qi.mu.Unlock()
	// Fields and helpers referenced here so the compiler keeps them across stub commits.
	_ = qi.consecutiveCount
	_ = qi.candidate
	_ = classify(rttMs, lossPct)
	panic("not implemented: BC-2.06.001")
}

// OnMissingFrame records one consecutive missing-frame event. After
// HysteresisCount consecutive missing frames the quality indicator degrades
// one level (green→yellow or yellow→red, never skips; BC-2.06.002 postcondition 2;
// AC-003, AC-004).
//
// Receiving a frame resets the missing-frame counter; call Update to record a
// successful measurement.
func (qi *QualityIndicator) OnMissingFrame() {
	qi.mu.Lock()
	defer qi.mu.Unlock()
	// Field referenced here so the compiler keeps it across stub commits.
	_ = qi.missingFrameCount
	panic("not implemented: BC-2.06.002")
}

// classify returns the raw Quality level for (rttMs, lossPct) without applying
// hysteresis. Used internally by Update.
//
// BC-2.06.001 postconditions 2–4 (thresholds NFR-001; ARCH-INDEX F-008).
func classify(rttMs float64, lossPct float64) Quality {
	panic("not implemented: BC-2.06.001")
}
