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
//
// STUB — S-BL.CLI-SURFACE-COMPLETION (Red Gate, BC-5.38.001). Not yet
// implemented; body panics unconditionally so no test can accidentally pass
// before Task 4's Green step.
package main

import "context"

// runRouterDrain implements `sbctl router drain --router=<addr>`.
//
// AC-016 / BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note) — same anchor as AC-012.
//
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 4 (Green step) implements the
// connectAndRun dispatch to router.drain, tolerating connection-reset as a
// non-error outcome. Red Gate: body panics unconditionally.
func runRouterDrain(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	panic("not implemented: S-BL.CLI-SURFACE-COMPLETION runRouterDrain")
}
