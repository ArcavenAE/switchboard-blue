// router_metrics.go implements the `sbctl router metrics --svtn=<id>` canonical subcommand.
//
// `sbctl router metrics --svtn=<id>` returns per-SVTN forwarding metrics from the router daemon:
// frame count, HMAC failure count, drop cache hit count, and per-path frame distribution
// (BC-2.06.003 PC-2).
//
// JSON output format (--json flag):
//
//	{"frame_count":<n>,"hmac_fail_count":<n>,"drop_cache_hits":<n>,"path_distribution":{<path_id>:<frame_count>}}
//
// Human-readable output: labelled table.
//
// When the daemon is unreachable, returns E-NET-001 exit code 1 per BC-2.07.003.
//
// Purity classification (ARCH-09): effectful-boundary — network I/O to daemon socket.
package main

import (
	"context"
)

// RouterMetrics is the JSON response schema for `sbctl router metrics --svtn=<id>`.
// BC-2.06.003 PC-2.
type RouterMetrics struct {
	FrameCount       uint64            `json:"frame_count"`
	HMACFailCount    uint64            `json:"hmac_fail_count"`
	DropCacheHits    uint64            `json:"drop_cache_hits"`
	PathDistribution map[string]uint64 `json:"path_distribution"`
}

// runRouterMetrics implements `sbctl router metrics --svtn=<id>`.
// Parses the --svtn flag and dispatches the router.metrics RPC.
//
// AC-002 / BC-2.06.003 PC-2
func runRouterMetrics(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	panic("todo: AC-002 — parse --svtn flag, dispatch router.metrics RPC, format output")
}
