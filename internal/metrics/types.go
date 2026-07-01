// Package metrics — response types for daemon-side paths.list, router.metrics,
// and router.status RPC handlers (BC-2.06.003 v1.13).
//
// All types in this file are pure data + serialization. They perform no I/O.
// Purity classification (ARCH-09): pure-core.
package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// RTTKind discriminates the two states of an RTTValue.
// Using a Kind enum instead of a sentinel SampleCount value avoids the
// float64(0)-vs-nil ambiguity that sentinel-based decode cannot resolve
// (F-P2L1-004).
type RTTKind uint8

const (
	// PendingKind indicates the p99 RTT is not yet determined (SampleCount < 10).
	// MarshalJSON emits the JSON string "pending".
	PendingKind RTTKind = iota
	// FloatKind indicates the p99 RTT is a valid measured value (SampleCount ≥ 10).
	// MarshalJSON emits a JSON float64 number.
	FloatKind
)

// RTTValue represents the rtt_p99_ms field in PathEntry.
// It serializes as a float64 when Kind == FloatKind and as the JSON string
// "pending" when Kind == PendingKind (BC-2.06.003 v1.13 PC-1, EC-003).
//
// Zero value is PendingKind with Value == 0.
//
// F-P2L1-004: Kind-based discrimination replaces the sentinel SampleCount
// approach, which could not distinguish float64(0) from nil on UnmarshalJSON.
type RTTValue struct {
	// Kind discriminates pending (PendingKind) from valid float (FloatKind).
	Kind RTTKind
	// Value is the p99 RTT in milliseconds. Only meaningful when Kind == FloatKind.
	Value float64
	// SampleCount is the total histogram sample count. Mirrors PathSnapshot.SampleCount.
	// Kept for callers (e.g. QualityFromEntry) that derive quality from sample count.
	// When SampleCount < 10 this field matches Kind == PendingKind.
	SampleCount uint64
}

// MarshalJSON implements json.Marshaler for the RTTValue union type.
// Emits a float64 when Kind == FloatKind; emits the string "pending" otherwise.
// BC-2.06.003 v1.13 PC-1 (pending sentinel), EC-003.
func (r RTTValue) MarshalJSON() ([]byte, error) {
	switch r.Kind {
	case FloatKind:
		return json.Marshal(r.Value)
	default: // PendingKind (zero value)
		return []byte(`"pending"`), nil
	}
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
	// Status is one of: "active", "degraded" (BC-2.06.003 v1.13 PC-1).
	// "failed" is reserved for S-BL.PATH-FAILED-STATUS (Wave-7) and MUST NOT be emitted.
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

// UnmarshalJSON implements json.Unmarshaler for the RTTValue union type.
// Peeks at the first non-whitespace byte to discriminate:
//   - '"' → must be the string "pending" → Kind=PendingKind, Value=0, SampleCount=0.
//   - digit or '-' → JSON float64 number → Kind=FloatKind, Value=v, SampleCount=10.
//
// This Kind-based approach can correctly distinguish float64(0) (a valid green-path
// measurement) from the "pending" string, which the old sentinel approach could not
// (F-P2L1-004). SampleCount is set to 10 for FloatKind so that callers relying on
// the SampleCount-based pending check continue to work.
func (r *RTTValue) UnmarshalJSON(data []byte) error {
	// Find first non-whitespace byte.
	var first byte
	for _, b := range data {
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
			first = b
			break
		}
	}

	if first == '"' {
		// String variant: must be exactly "pending".
		if string(data) != `"pending"` {
			return fmt.Errorf("rtt_p99_ms: expected \"pending\" string or float64; got %s", data)
		}
		r.Kind = PendingKind
		r.Value = 0
		r.SampleCount = 0
		return nil
	}

	// Numeric variant: decode as float64.
	var v float64
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("rtt_p99_ms: expected float64 or \"pending\": %w", err)
	}
	if err := validateRTTFloat(v); err != nil {
		return err
	}
	r.Kind = FloatKind
	r.Value = v
	r.SampleCount = 10 // signal ≥10 so SampleCount-based callers treat this as valid
	return nil
}

// validateRTTFloat rejects NaN, ±Inf, and negative values.
// Package-private; the guard is applied inside UnmarshalJSON but factored out
// so tests can exercise the numeric-validation path directly via the helper,
// independent of the JSON string-token branch (F-P5L2-02).
func validateRTTFloat(v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return errors.New("rtt: NaN or Inf not permitted")
	}
	if v < 0 {
		return errors.New("rtt: negative not permitted")
	}
	return nil
}

// Ensure RTTValue implements json.Marshaler and json.Unmarshaler at compile time.
var (
	_ json.Marshaler   = RTTValue{}
	_ json.Unmarshaler = &RTTValue{}
)
