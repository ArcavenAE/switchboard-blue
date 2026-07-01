// Package metrics — response types for daemon-side paths.list, router.metrics,
// and router.status RPC handlers (BC-2.06.003 v1.8).
//
// All types in this file are pure data + serialization. They perform no I/O.
// Purity classification (ARCH-09): pure-core.
package metrics

import "encoding/json"

// RTTValue represents the rtt_p99_ms field in PathEntry.
// It serializes as a float64 when SampleCount ≥ 10 and as the JSON string
// "pending" when SampleCount < 10 (BC-2.06.003 v1.8 PC-1, EC-003).
//
// Zero value is pending (SampleCount == 0, ValueMs == 0).
type RTTValue struct {
	// ValueMs is the p99 RTT in milliseconds. Only meaningful when SampleCount ≥ 10.
	ValueMs float64
	// SampleCount is the total histogram sample count. Mirrors PathSnapshot.SampleCount.
	// When SampleCount < 10, MarshalJSON emits "pending" instead of ValueMs.
	SampleCount uint64
}

// MarshalJSON implements json.Marshaler for the RTTValue union type.
// Emits a float64 when SampleCount ≥ 10; emits the string "pending" otherwise.
// BC-2.06.003 v1.8 PC-1 (pending sentinel), EC-003.
func (r RTTValue) MarshalJSON() ([]byte, error) {
	panic("TODO: S-W5.04 RTTValue.MarshalJSON not yet implemented")
}

// PathEntry is a single path in the paths.list or router.status response.
// Field names match the BC-2.06.003 PC-1 JSON schema exactly.
type PathEntry struct {
	// PathID is the opaque path identifier (string).
	PathID string `json:"path_id"`
	// RouterAddr is the remote router address (host:port).
	RouterAddr string `json:"router_addr"`
	// RTTMs is the most-recent EWMA RTT sample in milliseconds (float64).
	RTTMs float64 `json:"rtt_ms"`
	// RTTP99Ms is the p99 RTT union value: float64 or "pending" string.
	// Implements json.Marshaler via RTTValue (BC-2.06.003 PC-1, EC-003).
	RTTP99Ms RTTValue `json:"rtt_p99_ms"`
	// LossPct is the packet loss rate as a percentage (float64, 0.0–100.0).
	LossPct float64 `json:"loss_pct"`
	// Status is one of: "active", "degraded", "failed" (BC-2.06.003 PC-1).
	// Derived from PathSnapshot.Degraded and PathSnapshot.Active (BC-2.06.001).
	Status string `json:"status"`
}

// PathsListResponse is the response envelope for the paths.list RPC.
// BC-2.06.003 PC-1 canonical form; EC-001 empty-path case.
type PathsListResponse struct {
	// Paths is the list of active paths. May be empty.
	Paths []PathEntry `json:"paths"`
	// Message is a human-readable note. Set to "no active paths" when Paths is empty (EC-001).
	// Omitted from JSON when empty.
	Message string `json:"message,omitempty"`
}

// RouterMetricsResponse is the response envelope for the router.metrics RPC.
// BC-2.06.003 PC-2 canonical form.
type RouterMetricsResponse struct {
	// FrameCount is the total number of frames forwarded for the SVTN.
	FrameCount uint64 `json:"frame_count"`
	// HMACFailCount is the number of frames rejected due to HMAC verification failure.
	HMACFailCount uint64 `json:"hmac_fail_count"`
	// DropCacheHits is the number of frames rejected by the replay drop-cache.
	DropCacheHits uint64 `json:"drop_cache_hits"`
	// PathDistribution maps path_id to per-path frame count for the SVTN.
	PathDistribution map[string]uint64 `json:"path_distribution"`
}

// Ensure RTTValue implements json.Marshaler at compile time.
var _ json.Marshaler = RTTValue{}
