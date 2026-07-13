// router_drain.go implements the `sbctl router drain` subcommand.
//
// `sbctl router drain --router=<addr>` dispatches the router.drain RPC via
// the existing connectAndRun pattern (same dial+auth+dispatch shape `router
// metrics` and `paths list` already use) — bridging into the shipped
// drain/shutdown sequence (S-7.04-FU-DRAIN-WIRE) without duplicating any
// drain logic (Decision 4 / AC-012, AC-016).
//
// Wire verb: router.drain. Request args {}. Response {"accepted": true} —
// fire-and-forget. Because drain triggers the full shutdown sequence, a
// "connection reset" observed following (or even without) the response is an
// expected outcome, not a protocol error (AC-012 PC-3; BC-2.09.002 PC-3
// best-effort-delivery framing extended to the triggering RPC itself).
//
// Authority: Tier-1 operator-key auth only.
//
// Purity classification (ARCH-09): effectful-boundary — network I/O to daemon socket.
package main

import "context"

// runRouterDrain implements `sbctl router drain --router=<addr>`.
//
// AC-016 / BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note) — same anchor as AC-012.
// A connection reset following (or without) the {"accepted": true} response
// surfaces through connectAndRun as an ordinary E-RPC-001 dispatch error
// (never E-ADM-010/E-CFG-*) — the expected shape of a severed connection
// after the daemon begins its shutdown sequence (AC-012 PC-3).
// router.drain takes no flags of its own (wire args are always {}); args is
// part of the run* dispatch signature contract (main.go) but unused here.
func runRouterDrain(ctx context.Context, target, keyPath string, useJSON bool, _ []string, sio sbctlIO) error {
	return connectAndRun(ctx, target, keyPath, useJSON, "router.drain", map[string]string{}, sio)
}
