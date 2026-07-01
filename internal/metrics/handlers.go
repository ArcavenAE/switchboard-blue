// handlers.go implements the daemon-side RPC handler logic for paths.list,
// router.metrics, and router.status (BC-2.06.003 v1.8 PC-1, PC-2, PC-3).
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
func PathsList(ctx context.Context, args json.RawMessage, src PathsListSource) (PathsListResponse, error) {
	panic("TODO: S-W5.04 PathsList not yet implemented")
}

// RouterMetrics is the handler logic for the "router.metrics" RPC.
// Reads per-SVTN forwarding counters via src and returns a RouterMetricsResponse.
// Returns an error (E-RPC-011) when the requested SVTN is not found.
//
// BC-2.06.003 v1.8 PC-2; AC-004.
func RouterMetrics(ctx context.Context, args json.RawMessage, src RouterMetricsSource) (RouterMetricsResponse, error) {
	panic("TODO: S-W5.04 RouterMetrics not yet implemented")
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
	panic("TODO: S-W5.04 RouterStatus not yet implemented")
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
	panic("TODO: S-W5.04 PathEntryFromSnapshot not yet implemented")
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
	panic("TODO: S-W5.04 QualityFromEntry not yet implemented")
}
