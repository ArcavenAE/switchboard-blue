// router_status.go implements `sbctl router status --target <router>` as a
// convenience alias for `sbctl paths list`.
//
// This is a CLI dispatch shim only. There is exactly one code path in the
// underlying query layer serving both commands — no divergent implementations
// (BC-2.06.003 PC-3 + EC-005; F-P8-002 ruling).
//
// The `quality` column (green/yellow/red) is added by this formatter on top of
// the PathEntry output. The underlying data source is identical to `sbctl paths list`.
//
// Flags:
//
//	--target <router>   overrides the default daemon address
//
// Output format: identical to `sbctl paths list` plus a `quality` field
// (green/yellow/red derived from status + rtt_p99_ms).
//
// AC-003 / BC-2.06.003 PC-3
//
// Purity classification (ARCH-09): effectful-boundary — network I/O to daemon socket.
package main

import (
	"context"
	"strings"
)

// PathEntryWithQuality extends PathEntry with the quality field for the
// `sbctl router status` alias output.
//
// AC-003 / BC-2.06.003 PC-3
type PathEntryWithQuality struct {
	PathEntry
	Quality string `json:"quality"`
}

// runRouterStatus implements `sbctl router status --target <router>`.
// This is a CLI shim that calls the same underlying query path as runPathsList.
// It appends a `quality` field (green/yellow/red) derived from status + rtt_p99_ms.
//
// The --target flag in args overrides the caller-supplied target address, matching
// the alias semantics in BC-2.06.003 PC-3: `--target <router>` is equivalent to
// `sbctl --target <router> paths list`.
//
// AC-003 / BC-2.06.003 PC-3
func runRouterStatus(ctx context.Context, target, keyPath string, useJSON bool, args []string) error {
	// Parse --target override from args (alias semantics: --target overrides daemon address).
	for i, arg := range args {
		if arg == "--target" && i+1 < len(args) {
			target = args[i+1]
		} else if strings.HasPrefix(arg, "--target=") {
			target = strings.TrimPrefix(arg, "--target=")
		}
	}
	// Shim: delegate to the same underlying RPC as runPathsList (single code path,
	// no divergent implementation in internal/metrics per BC-2.06.003 PC-3).
	return connectAndRun(ctx, target, keyPath, useJSON, "paths.list", nil)
}
