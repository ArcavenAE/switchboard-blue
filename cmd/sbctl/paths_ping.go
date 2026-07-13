// paths_ping.go implements the `sbctl paths ping` subcommand.
//
// `sbctl paths ping --router=<addr>` is a one-shot, on-demand reachability +
// latency probe of an arbitrarily-dialed target — distinct from `paths list`/
// `router status`, which report historical EWMA-smoothed metrics accumulated
// by a PathTracker over time (Decision 1 / BC-2.06.004).
//
// Wire verb: paths.ping. Request args {} (empty — the daemon dialed via
// --router=<addr> IS the probe target by construction). Response {"pong": true}.
// sbctl synthesizes {"router": "<addr>", "rtt_ms": <float64>} client-side,
// measured from dial-start to response-decode-complete — not on the wire.
//
// Authority: Tier-1 operator-key auth only — same bar as paths.list/
// router.metrics/router.status; no additional Tier-2 role gate.
//
// Purity classification (ARCH-09): effectful-boundary — network I/O to daemon socket.
//
// STUB — S-BL.CLI-SURFACE-COMPLETION (Red Gate, BC-5.38.001). Not yet
// implemented; body panics unconditionally so no test can accidentally pass
// before Task 1's Green step.
package main

import "context"

// runPathsPing implements `sbctl paths ping --router=<addr>`.
//
// AC-001..AC-004 / BC-2.06.004 PC-1..PC-4, EC-001..EC-003, Invariant 1, Invariant 2.
//
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 1 (Green step) implements the
// dial/measure/report logic. Red Gate: body panics unconditionally.
func runPathsPing(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	panic("not implemented: S-BL.CLI-SURFACE-COMPLETION runPathsPing")
}
