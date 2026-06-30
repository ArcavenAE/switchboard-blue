// paths_list.go implements the `sbctl paths list` canonical subcommand.
//
// `sbctl paths list` returns per-path RTT, p99 RTT, loss, and quality metrics
// for all active paths on the target node (BC-2.06.003 PC-1).
//
// JSON output format (--json flag):
//
//	[{"path_id":"<id>","router_addr":"<host:port>","rtt_ms":<f64>,"rtt_p99_ms":<f64|"pending">,"loss_pct":<f64>,"status":"active|degraded|failed"}]
//
// Human-readable output: tab-separated table.
//
// When the daemon is unreachable, returns E-NET-001 exit code 1 per BC-2.07.003.
// Empty path list: {"paths":[],"message":"no active paths"} exit code 0 (EC-001).
//
// Purity classification (ARCH-09): effectful-boundary — network I/O to daemon socket.
package main

import (
	"context"
)

// PathEntry is one row in the `sbctl paths list` JSON output.
// rtt_p99_ms is encoded as either float64 or the string "pending"
// (BC-2.06.003 EC-003) — the JSON marshaling handles the union type.
type PathEntry struct {
	PathID     string  `json:"path_id"`
	RouterAddr string  `json:"router_addr"`
	RTTMs      float64 `json:"rtt_ms"`
	// P99RTTMs is either a float64 or the string "pending"; represented as any
	// so the JSON encoder can emit the correct type (BC-2.06.003 EC-003).
	P99RTTMs any     `json:"rtt_p99_ms"`
	LossPct  float64 `json:"loss_pct"`
	Status   string  `json:"status"`
}

// runPathsList implements `sbctl paths list`.
// Dispatches the paths.list RPC to the daemon and formats the response.
//
// AC-001 / BC-2.06.003 PC-1
func runPathsList(ctx context.Context, target, keyPath string, useJSON bool) error {
	panic("todo: AC-001 — dispatch paths.list RPC and format per-path output")
}

// formatPathsTable formats a slice of PathEntry values as a human-readable
// tab-separated table for non-JSON output mode.
//
// AC-001 / BC-2.06.003 PC-1
func formatPathsTable(entries []PathEntry) string {
	panic("todo: AC-001 — format paths as human-readable table")
}
