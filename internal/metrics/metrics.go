// Package metrics implements the session quality indicator (green/yellow/red)
// derived from measured path RTT and packet loss (BC-2.06.001 v1.3, BC-2.06.002).
//
// Classification thresholds (NFR-001; ARCH-INDEX F-008):
//
//	Green:  RTT p99 ≤ 100 ms AND loss ≤ 5 %
//	Yellow: RTT p99 in (100 ms, 500 ms] OR loss in (5 %, 20 %]   (and not Red)
//	Red:    RTT p99 > 500 ms OR loss > 20 %
//
// Red takes precedence over Yellow: when inputs simultaneously satisfy both
// PC-3 (Yellow) and PC-4 (Red) — e.g. RTT=600 ms and loss=10 % — the
// indicator is Red. Red is evaluated first in classify() (BC-2.06.001 v1.3 PC-4).
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

	// A received frame resets the missing-frame counter (BC-2.06.002 PC-4).
	qi.missingFrameCount = 0

	newLevel := classify(rttMs, lossPct)

	if newLevel > qi.current {
		// Downgrade is immediate but steps one level at a time (AC-004).
		// A single Red-range measurement on a Green indicator must go to Yellow,
		// not skip directly to Red.
		qi.current++
		qi.candidate = qi.current
		qi.consecutiveCount = 1
		return
	}

	if newLevel < qi.current {
		// Potential upgrade: track consecutive streak toward target level.
		if newLevel == qi.candidate {
			qi.consecutiveCount++
		} else {
			// New candidate level (e.g. jumped from Red directly feeding Green-range
			// measurements — candidate switches, streak resets).
			qi.candidate = newLevel
			qi.consecutiveCount = 1
		}

		if qi.consecutiveCount >= HysteresisCount {
			// Only upgrade one level at a time (BC-2.06.002 PC-2 / AC-004).
			nextLevel := qi.current - 1
			if nextLevel < Green {
				nextLevel = Green
			}
			qi.current = nextLevel
			// If we've reached the candidate level, keep tracking;
			// otherwise reset so the next window can continue.
			if qi.current == qi.candidate {
				// Keep the streak at threshold so the next upgrade step can
				// proceed without waiting for a fresh HysteresisCount window
				// (e.g. Red→Yellow and then immediately toward Green if the
				// candidate is already Green; BC-2.06.001 v1.3 invariant 3).
				qi.consecutiveCount = HysteresisCount
			} else {
				qi.consecutiveCount = 0
			}
		}
		return
	}

	// newLevel == qi.current: same level, keep streak going (or reset candidate).
	if newLevel == qi.candidate {
		qi.consecutiveCount++
	} else {
		qi.candidate = newLevel
		qi.consecutiveCount = 1
	}
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

	qi.missingFrameCount++
	if qi.missingFrameCount >= HysteresisCount {
		qi.missingFrameCount = 0
		// Degrade one level — never skip (BC-2.06.002 PC-2; AC-004).
		if qi.current < Red {
			qi.current++
			qi.candidate = qi.current
			qi.consecutiveCount = 0
		}
	}
}

// classify returns the raw Quality level for (rttMs, lossPct) without applying
// hysteresis. Used internally by Update.
//
// Band predicates use OR-form (BC-2.06.001 v1.3 PC-3, PC-4):
//   - Yellow fires when RTT or loss exceeds green thresholds (but neither exceeds red).
//   - Red fires when RTT > YellowRTTMs OR loss > YellowLossPct.
//
// Red-over-Yellow precedence (BC-2.06.001 v1.3 PC-4): Red is the fall-through
// default; the Yellow branch only fires when BOTH dimensions are within yellow
// bounds, so any single red-range value bypasses Yellow and returns Red directly.
func classify(rttMs float64, lossPct float64) Quality {
	// PC-2: both dimensions within green bounds → Green.
	if rttMs <= GreenRTTMs && lossPct <= GreenLossPct {
		return Green
	}
	// PC-3/PC-4: Yellow only when both dimensions are within yellow bounds.
	// If either exceeds yellow thresholds, fall through to Red (OR-form precedence).
	if rttMs <= YellowRTTMs && lossPct <= YellowLossPct {
		return Yellow
	}
	// PC-4: RTT > 500 ms OR loss > 20 % (BC-2.06.001 v1.3).
	return Red
}
