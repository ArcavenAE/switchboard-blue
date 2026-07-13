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
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// runPathsPing implements `sbctl paths ping --router=<addr>`.
//
// AC-001..AC-004 / BC-2.06.004 PC-1..PC-4, EC-001..EC-003, Invariant 1, Invariant 2.
//
// Output shape follows the house useJSON convention (interface-definitions.md
// §214; same as paths list/router status/router reload): default mode prints
// the bare {"router":...,"rtt_ms":...} object; --json wraps it in the
// {"ok":true,"data":{...}} envelope.
func runPathsPing(ctx context.Context, target, keyPath string, useJSON bool, args []string, sio sbctlIO) error {
	// --router=<addr> overrides --target (BC-2.06.004 PC-1) — the daemon
	// dialed via --router IS the probe target by construction.
	for i, arg := range args {
		if arg == "--router" {
			if i+1 >= len(args) {
				return usageErrf("E-CFG-001: paths ping: --router requires a value")
			}
			target = args[i+1]
		} else if strings.HasPrefix(arg, "--router=") {
			target = strings.TrimPrefix(arg, "--router=")
		}
	}

	privKey, err := loadEd25519Key(keyPath, os.UserHomeDir)
	if err != nil {
		_ = writeError(useJSON, "E-CFG-010", err.Error(), sio)
		return reported(err)
	}

	// rtt_ms spans dial-start to response-decode-complete, measured
	// client-side (BC-2.06.004 PC-1).
	start := time.Now()

	var conn net.Conn
	if len(target) > 0 && target[0] == '/' {
		conn, err = (&net.Dialer{}).DialContext(ctx, "unix", target)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", target)
	}
	if err != nil {
		msg := fmt.Sprintf("daemon unreachable: %s: %s", target, err)
		return writeError(useJSON, "E-NET-001", msg, sio)
	}
	defer func() { _ = conn.Close() }()

	if err = Authenticate(ctx, conn, privKey); err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			msg := fmt.Sprintf("daemon unreachable: %s: connection timed out", target)
			return writeError(useJSON, "E-NET-001", msg, sio)
		}
		_ = writeError(useJSON, "E-ADM-010", "authentication failed", sio)
		return reported(err)
	}

	// paths.ping performs zero PathTracker interaction (AC-004 postcondition
	// 3): empty request args, {"pong": true} response (BC-2.06.004 PC-1).
	if _, err = dispatch(ctx, conn, "paths.ping", map[string]string{}); err != nil {
		_ = writeError(useJSON, "E-RPC-001", err.Error(), sio)
		return reported(err)
	}

	rttMs := float64(time.Since(start).Microseconds()) / 1000.0

	data, err := json.Marshal(struct {
		Router string  `json:"router"`
		RTTMs  float64 `json:"rtt_ms"`
	}{Router: target, RTTMs: rttMs})
	if err != nil {
		_, _ = fmt.Fprintf(sio.err, "marshal error: %s\n", err)
		return internal(fmt.Errorf("marshal ping result: %w", err))
	}

	return writeSuccess(useJSON, data, sio)
}
