// router_reload.go implements the `sbctl router reload` subcommand.
//
// `sbctl router reload --router=<addr>` dispatches the router.reload RPC via
// the existing connectAndRun pattern (same dial+auth+dispatch shape `router
// metrics` and `paths list` already use) — bridging into the shipped
// SIGHUP-reload path (S-7.04-FU-SIGHUP-RELOAD) without duplicating any reload
// logic (Decision 4 / AC-011, AC-015).
//
// Wire verb: router.reload. Request args {}. Response {"accepted": true} —
// fire-and-forget, matching raw-signal UX parity.
//
// Authority: Tier-1 operator-key auth only.
//
// Purity classification (ARCH-09): effectful-boundary — network I/O to daemon socket.
package main

import "context"

// runRouterReload implements `sbctl router reload --router=<addr>`.
//
// AC-015 / BC-2.09.001 v1.2 PC-1 (RPC-trigger note) — same anchor as AC-011.
// router.reload takes no flags of its own (wire args are always {}); args is
// part of the run* dispatch signature contract (main.go) but unused here.
func runRouterReload(ctx context.Context, target, keyPath string, useJSON bool, _ []string, sio sbctlIO) error {
	return connectAndRun(ctx, target, keyPath, useJSON, "router.reload", map[string]string{}, sio)
}
