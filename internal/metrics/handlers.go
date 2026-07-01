// Daemon-side RPC handler logic for paths.list, router.metrics, and router.status
// (BC-2.06.003 v1.8 PC-1, PC-2, PC-3).
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
	"fmt"

	"github.com/arcavenae/switchboard/internal/paths"
)

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
// BC-2.06.003 v1.8 PC-1; AC-001.
func PathsList(_ context.Context, _ json.RawMessage, src PathsListSource) (PathsListResponse, error) {
	snaps := src.AllSnapshots()
	entries := make([]PathEntry, 0, len(snaps))
	for pathID, snap := range snaps {
		// router_addr: "" — interim per BC-2.06.003 v1.9; PathSnapshot enrichment
		// tracked in S-BL.ROUTER-ADDR (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER).
		entries = append(entries, PathEntryFromSnapshot(pathID, "", snap))
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
// BC-2.06.003 v1.9 PC-2; AC-004; Fix 6 (svtn_id required).
func RouterMetrics(_ context.Context, args json.RawMessage, src RouterMetricsSource) (RouterMetricsResponse, error) {
	var req struct {
		// svtn_id is the canonical wire key sent by sbctl (cmd/sbctl/router_metrics.go).
		// F-P1L1-001: changed from json:"svtn" to json:"svtn_id" to match sbctl wire format.
		SVTN string `json:"svtn_id"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &req); err != nil {
			return RouterMetricsResponse{}, fmt.Errorf("E-RPC-002: decode args: %w", err)
		}
	}
	// svtn_id is required — an empty or missing value cannot identify a router.
	// This rejects callers that omit the field entirely (Fix 6).
	if req.SVTN == "" {
		return RouterMetricsResponse{}, fmt.Errorf("E-RPC-002: router.metrics requires svtn_id string field")
	}
	return src.SVTNMetrics(req.SVTN)
}

// RouterStatus is the handler logic for the "router.status" RPC alias.
// Structurally identical to paths.list but with an additional "quality" summary
// field derived from status + rtt_p99_ms (BC-2.06.003 v1.8 PC-3, EC-007).
//
// When rtt_p99_ms is "pending" (SampleCount < 10), quality MUST be "pending"
// regardless of liveness state (S502-DEFER-3 failed+pending precedence ruling;
// BC-2.06.003 v1.8 EC-007; AC-005a).
//
// BC-2.06.003 v1.8 PC-3; AC-005, AC-005a.
func RouterStatus(ctx context.Context, args json.RawMessage, src PathsListSource) (RouterStatusResponse, error) {
	pathsResp, err := PathsList(ctx, args, src)
	if err != nil {
		return RouterStatusResponse{}, err
	}

	// Derive overall quality from the worst quality across all paths.
	// pending < green < yellow < red in severity for the summary field.
	// When any path is pending, the summary is pending (indeterminate).
	quality := overallQuality(pathsResp.Paths)

	return RouterStatusResponse{
		Paths:   pathsResp.Paths,
		Message: pathsResp.Message,
		Quality: quality,
	}, nil
}

// RouterStatusResponse is the response envelope for the router.status RPC.
// Structurally identical to PathsListResponse plus a quality summary field
// (BC-2.06.003 PC-3).
type RouterStatusResponse struct {
	// Paths is the per-path listing (same schema as PathsListResponse.Paths).
	Paths []PathEntry `json:"paths"`
	// Message is a human-readable note; omitted when empty.
	Message string `json:"message,omitempty"`
	// Quality is the overall path quality summary: "green", "yellow", "red", or "pending".
	// "pending" when any path has SampleCount < 10 (BC-2.06.003 PC-3, EC-006, EC-007).
	Quality string `json:"quality"`
}

// PathEntryFromSnapshot converts a PathSnapshot to a PathEntry.
// Derives PathEntry.Status from PathSnapshot.Degraded and PathSnapshot.Active:
//   - Active=false → "failed" (≥3 consecutive missed keepalives)
//   - Degraded=true → "degraded" (EWMA RTT > 200ms sustained)
//   - otherwise → "active"
//
// BC-2.06.003 PC-1; BC-2.06.001; AC-003.
func PathEntryFromSnapshot(pathID, routerAddr string, snap paths.PathSnapshot) PathEntry {
	status := "active"
	if !snap.Active {
		status = "failed"
	} else if snap.Degraded {
		status = "degraded"
	}
	// Derive RTTValue Kind from SampleCount per BC-2.06.003 v1.9 PC-1:
	// FloatKind when SampleCount ≥ 10 (p99 is meaningful), PendingKind otherwise.
	var rttP99 RTTValue
	if snap.SampleCount >= 10 {
		rttP99 = RTTValue{Kind: FloatKind, Value: snap.P99RTTMs, SampleCount: snap.SampleCount}
	} else {
		rttP99 = RTTValue{Kind: PendingKind, SampleCount: snap.SampleCount}
	}
	return PathEntry{
		PathID:     pathID,
		RouterAddr: routerAddr,
		RTTMs:      snap.EWMARTTMs,
		RTTP99Ms:   rttP99,
		LossPct:    snap.LossPct,
		Status:     status,
	}
}

// QualityFromEntry derives the quality string for a single PathEntry.
// Returns "pending" when RTTP99Ms.SampleCount < 10 (BC-2.06.003 EC-006, EC-007).
// Returns green/yellow/red derived from rtt_p99_ms and status otherwise.
//
// The failed+pending precedence ruling (S502-DEFER-3, BC-2.06.003 v1.8 EC-007):
// when status=="failed" AND SampleCount < 10, quality MUST still be "pending".
// status and quality are orthogonal fields.
//
// BC-2.06.003 PC-3; AC-005, AC-005a.
func QualityFromEntry(entry PathEntry) string {
	// EC-007, EC-006, F-M3: pending p99 always yields pending quality.
	// This holds even when status=="failed" (S502-DEFER-3).
	if entry.RTTP99Ms.SampleCount < 10 {
		return "pending"
	}
	return Classify(entry.RTTP99Ms.Value, entry.LossPct).String()
}

// overallQuality derives the worst-case quality across all entries for the
// router.status summary field. Precedence: pending > red > yellow > green.
// An empty path list returns "pending" (indeterminate — no data).
func overallQuality(entries []PathEntry) string {
	if len(entries) == 0 {
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
