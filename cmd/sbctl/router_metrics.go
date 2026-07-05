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
	"strings"
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
func runRouterMetrics(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	// Parse --svtn=<id> from args.
	var svtnID string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--svtn=") {
			svtnID = strings.TrimPrefix(arg, "--svtn=")
		}
	}
	if svtnID == "" {
		_ = writeError(useJSON, "E-CFG-010", "router metrics: --svtn=<id> is required", sio)
		// Wrap usageErrf so main() maps to exit 2 via *usageError chain
		// AND skips re-print via *reportedError chain (#89 single-print).
		return reported(usageErrf("router metrics: --svtn flag is required"))
	}
	return connectAndRun(ctx, target, keyPath, useJSON, "router.metrics", map[string]string{"svtn_id": svtnID}, sio)
}
