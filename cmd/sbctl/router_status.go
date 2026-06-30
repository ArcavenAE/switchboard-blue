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
	panic("todo: AC-003 — parse --target flag override, call same query as paths list, append quality field")
}

// qualityFromPathEntry derives the green/yellow/red quality indicator from a
// PathEntry's status and rtt_p99_ms fields.
//
// AC-003 / BC-2.06.003 PC-3; green/yellow/red thresholds from metrics package (BC-2.06.001).
func qualityFromPathEntry(entry PathEntry) string {
	panic("todo: AC-003 — derive quality indicator from path entry status and p99")
}
