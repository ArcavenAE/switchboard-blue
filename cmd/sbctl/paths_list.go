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

	"github.com/arcavenae/switchboard/internal/metrics"
)

// PathEntry is one row in the `sbctl paths list` JSON output.
// rtt_p99_ms is encoded as either a bare float64 or the JSON string "pending"
// (BC-2.06.003 v1.15 PC-1, EC-003) via metrics.RTTValue's Marshal/Unmarshal.
//
// The typed metrics.RTTValue replaces the pre-#54 `any` field. This closes
// S-BL.SBCTL-DRIFT by sharing the client- and daemon-side wire discipline
// through a single type. metrics.RTTValue.UnmarshalJSON also rejects JSON
// null (F-P2L1-004), where the old `any` silently accepted it as a nil
// interface — the rejection is spec-correct.
type PathEntry struct {
	PathID     string           `json:"path_id"`
	RouterAddr string           `json:"router_addr"`
	RTTMs      float64          `json:"rtt_ms"`
	P99RTTMs   metrics.RTTValue `json:"rtt_p99_ms"`
	LossPct    float64          `json:"loss_pct"`
	Status     string           `json:"status"`
}

// runPathsList implements `sbctl paths list`.
// Dispatches the paths.list RPC to the daemon and formats the response.
//
// AC-001 / BC-2.06.003 PC-1
func runPathsList(ctx context.Context, target, keyPath string, useJSON bool, sio sbctlIO) error {
	return connectAndRun(ctx, target, keyPath, useJSON, "paths.list", nil, sio)
}
