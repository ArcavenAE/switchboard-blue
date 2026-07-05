// Daemon-side RPC handler logic for paths.list, router.metrics, and router.status
// (BC-2.06.003 v1.15 PC-1, PC-2, PC-3).
//
// Purity classification (ARCH-09): effectful — reads PathTracker state via
// PathSnapshot. I/O ownership stays in internal/mgmt; these functions are the
// pure-logic layer invoked by the registered handler closures.
//
// Package DAG: internal/metrics imports internal/paths for PathSnapshot.
// internal/mgmt imports internal/metrics for the handler functions.
// internal/mgmt MUST NOT import internal/paths directly (ARCH-12).

package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/arcavenae/switchboard/internal/paths"
)

// ErrDecodeArgs is the sentinel for E-RPC-002 (malformed or undecodable args).
// Tests inspect this via errors.Is to avoid string-matching on error messages
// (go.md error-handling rule 3).
var ErrDecodeArgs = errors.New("E-RPC-002: decode args")

// ErrInvalidParams is the sentinel for E-RPC-003 (required parameter missing or
// invalid after successful decode). Distinct from ErrDecodeArgs (E-RPC-002) which
// covers malformed JSON — ErrInvalidParams covers structurally valid JSON that
// fails semantic validation (e.g. a required field is empty).
// Tests inspect this via errors.Is (F-P10L1-05).
var ErrInvalidParams = errors.New("E-RPC-003: invalid params")

// PathsListSource is the read interface for fetching all active path snapshots.
// Implemented by whatever state store owns the PathTracker map in the daemon.
// Injected into the handler closure by internal/mgmt/register_metrics.go.
//
// Snapshot() must return a consistent copy per go.md rule 12 (no internal pointer leak).
type PathsListSource interface {
	// AllSnapshots returns path_id → PathSnapshot for all tracked paths.
	AllSnapshots() map[string]paths.PathSnapshot
}

// RouterMetricsSource is the read interface for per-SVTN forwarding counters.
// Injected into the handler closure by internal/mgmt/register_metrics.go.
type RouterMetricsSource interface {
	// SVTNMetrics returns the RouterMetricsResponse for the given SVTN ID, or
	// an error (E-RPC-011) when the SVTN is not found.
	SVTNMetrics(svtnID string) (RouterMetricsResponse, error)
}

// PathsList is the handler logic for the "paths.list" RPC.
// Reads all PathSnapshots via src, maps them to PathEntry values, and returns
// a PathsListResponse.
//
// ctx is the authenticated handler context supplied by mgmt.Server.
// args is the raw JSON args (unused for paths.list; may be nil or empty object).
//
// BC-2.06.003 v1.15 PC-1; AC-001.
func PathsList(_ context.Context, _ json.RawMessage, src PathsListSource) (PathsListResponse, error) {
	snaps := src.AllSnapshots()
	entries := make([]PathEntry, 0, len(snaps))
	for pathID, snap := range snaps {
		entries = append(entries, PathEntryFromSnapshot(pathID, snap))
	}
	if len(entries) == 0 {
		return PathsListResponse{Paths: entries, Message: "no active paths"}, nil
	}
	return PathsListResponse{Paths: entries}, nil
}

// RouterMetrics is the handler logic for the "router.metrics" RPC.
// Reads per-SVTN forwarding counters via src and returns a RouterMetricsResponse.
// Returns an E-RPC-* error if the svtn_id field is missing or empty, or if args
// cannot be decoded. Returns an E-RPC-011 error when the SVTN is not found.
//
// BC-2.06.003 v1.15 PC-2; AC-004; Fix 6 (svtn_id required).
func RouterMetrics(_ context.Context, args json.RawMessage, src RouterMetricsSource) (RouterMetricsResponse, error) {
	var req struct {
		// svtn_id is the canonical wire key sent by sbctl (cmd/sbctl/router_metrics.go).
		// F-P1L1-001: changed from json:"svtn" to json:"svtn_id" to match sbctl wire format.
		SVTN string `json:"svtn_id"`
	}
	if len(args) > 0 {
		// E-RPC-002: JSON is malformed or structurally undecodable.
		if err := json.Unmarshal(args, &req); err != nil {
			return RouterMetricsResponse{}, fmt.Errorf("router.metrics: %w: %w", ErrDecodeArgs, err)
		}
	}
	// svtn_id is required — an empty or missing value cannot identify a router.
	// This rejects callers that omit the field entirely (Fix 6).
	// E-RPC-003 (not E-RPC-002): the JSON decoded successfully but a required
	// semantic parameter is absent (F-P10L1-05).
	if req.SVTN == "" {
		return RouterMetricsResponse{}, fmt.Errorf("router.metrics: %w: svtn_id string field required", ErrInvalidParams)
	}
	return src.SVTNMetrics(req.SVTN)
}

// RouterStatus is the handler logic for the "router.status" RPC alias.
// Structurally identical to paths.list but with an additional "quality" summary
// field derived from status + rtt_p99_ms (BC-2.06.003 v1.15 PC-3, EC-007).
//
// When rtt_p99_ms is "pending" (SampleCount < 10), quality MUST be "pending"
// regardless of liveness state (S502-DEFER-3 degraded+pending precedence ruling;
// BC-2.06.003 v1.14 EC-007; AC-005a).
//
// BC-2.06.003 v1.15 PC-3; AC-005, AC-005a.
func RouterStatus(ctx context.Context, args json.RawMessage, src PathsListSource) (RouterStatusResponse, error) {
	pathsResp, err := PathsList(ctx, args, src)
	if err != nil {
		return RouterStatusResponse{}, err
	}

	// Derive overall quality from the worst quality across all paths.
	// Precedence: pending > red > yellow > green (pending dominates; indeterminate wins).
	// When any path is pending, the summary is pending regardless of other paths.
	quality := overallQuality(pathsResp.Paths)

	return RouterStatusResponse{
		Paths:   pathsResp.Paths,
		Message: pathsResp.Message,
		Quality: quality,
	}, nil
}

// PathEntryFromSnapshot converts a PathSnapshot to a PathEntry.
// Derives PathEntry.Status from PathSnapshot.Degraded and PathSnapshot.Active:
//   - Degraded=true → "degraded" (EWMA RTT > 200ms sustained)
//   - Active=false → "degraded" (liveness failure maps to "degraded" in Wave 6;
//     "failed" is reserved for S-BL.PATH-FAILED-STATUS per Ruling-4 and
//     BC-2.06.003 v1.14 PC-1 — implementations MUST NOT emit "failed" until
//     that story lands)
//   - otherwise → "active"
//
// RouterAddr is read from snap.RouterAddr — the sole source of truth per
// S-BL.ROUTER-ADDR / RULING-W6TB-B §3 (immutability invariant). Callers MUST
// populate snap.RouterAddr; a diverging out-of-band parameter would risk
// PathEntry.RouterAddr disagreeing with the snapshot it was derived from.
//
// BC-2.06.003 v1.15 PC-1; BC-2.06.001; AC-002; AC-003.
func PathEntryFromSnapshot(pathID string, snap paths.PathSnapshot) PathEntry {
	status := "active"
	if snap.Degraded || !snap.Active {
		status = "degraded"
	}
	// Defensive invariant: only "active" and "degraded" are valid in this wave.
	// "failed" is reserved for S-BL.PATH-FAILED-STATUS (Wave-7); any regression
	// that reintroduces it before that story lands is caught here (F-P10L1-07).
	if status != "active" && status != "degraded" {
		panic("BUG: PathEntryFromSnapshot: invalid status " + status + " — only active/degraded are valid until S-BL.PATH-FAILED-STATUS")
	}
	// Derive RTTValue Kind from SampleCount per BC-2.06.003 v1.14 PC-1:
	// FloatKind when SampleCount ≥ 10 (p99 is meaningful), PendingKind otherwise.
	// Use constructors to enforce the Kind/SampleCount invariant by construction (F-P10L1-08).
	var rttP99 RTTValue
	if snap.SampleCount >= 10 {
		rttP99 = NewRTTValueFloat(snap.P99RTTMs, snap.SampleCount)
	} else {
		rttP99 = NewRTTValuePending(snap.SampleCount)
	}
	return PathEntry{
		PathID:     pathID,
		RouterAddr: snap.RouterAddr,
		RTTMs:      snap.EWMARTTMs,
		RTTP99Ms:   rttP99,
		LossPct:    snap.LossPct,
		Status:     status,
	}
}

// QualityFromEntry derives the quality string for a single PathEntry.
// Returns "pending" when RTTP99Ms.Kind == PendingKind (BC-2.06.003 EC-006, EC-007).
// Returns green/yellow/red derived from rtt_p99_ms and status otherwise.
//
// Uses Kind (not SampleCount) for the pending check: SampleCount on RTTValue
// cannot be preserved through the wire format (bare float64 or "pending" string —
// BC-2.06.003 v1.14 PC-1), so Kind is the authoritative discriminator (H-2 Pass-8).
//
// The degraded+pending precedence ruling (S502-DEFER-3, BC-2.06.003 v1.14 EC-007):
// when status=="degraded" AND Kind==PendingKind, quality MUST still be "pending".
// status and quality are orthogonal fields.
//
// BC-2.06.003 v1.15 PC-3; AC-005, AC-005a.
func QualityFromEntry(entry PathEntry) string {
	// EC-007, EC-006, F-M3: pending p99 always yields pending quality.
	// This holds regardless of status value (status and quality are orthogonal per EC-007).
	if entry.RTTP99Ms.Kind == PendingKind {
		return "pending"
	}
	return Classify(entry.RTTP99Ms.Value, entry.LossPct).String()
}

// overallQuality derives the worst-case quality across all entries for the
// router.status summary field. Precedence: pending > red > yellow > green.
// An empty path list returns "pending" (indeterminate — no data).
//
// empty-paths → quality:"pending" is ratified by BC-2.06.003 EC-008 (v1.14)
// (F-P10L1-02).
func overallQuality(entries []PathEntry) string {
	if len(entries) == 0 {
		// empty-paths → pending is ratified by BC-2.06.003 EC-008 (v1.14) (F-P10L1-02).
		return "pending"
	}
	worst := "green"
	for _, e := range entries {
		q := QualityFromEntry(e)
		switch {
		case q == "pending":
			return "pending" // pending is immediately dominant
		case q == "red" && worst != "pending":
			worst = "red"
		case q == "yellow" && worst == "green":
			worst = "yellow"
		}
	}
	return worst
}
