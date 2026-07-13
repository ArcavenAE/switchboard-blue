// Registration of the paths.ping RPC handler on the mgmt.Server
// (S-BL.CLI-SURFACE-COMPLETION Decision 1; ARCH-12).
//
// paths.ping is a one-shot reachability probe distinct from the paths.list/
// router.metrics/router.status trio registered by RegisterMetricsHandlers —
// it takes empty request args and returns {"pong": true}, with zero
// PathTracker interaction (AC-004). Kept in its own file so wireMetricsHandlers
// can call it as a discrete registration step alongside RegisterMetricsHandlers,
// per the Architecture Mapping.
//
// Purity classification (ARCH-09): boundary — wires an effectful handler into
// the management server; no business logic lives here.

package mgmt

import (
	"context"
	"encoding/json"
)

// RegisterPingHandler registers the "paths.ping" RPC handler on s.
//
// Called once during daemon startup (cmd/switchboard, via wireMetricsHandlers),
// before Server.Serve. Returns an error if called after Serve has started
// (register-before-serve invariant; F-P2L1-001), mirroring RegisterMetricsHandlers.
//
// AC-004 / BC-2.06.004 Invariant 1, Trigger.
func RegisterPingHandler(s *Server) error {
	return s.Register(Handler{Command: "paths.ping", Fn: pingHandler()})
}

// pingHandler returns a mgmt.Handler.Fn for the "paths.ping" command.
//
// Empty request args in, {"pong": true} response data out — zero PathTracker
// interaction (AC-004 postcondition 3). paths.ping is a one-shot reachability
// probe; the daemon dialed via --router=<addr> IS the probe target by
// construction, so the handler has nothing to look up.
func pingHandler() func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		return struct {
			Pong bool `json:"pong"`
		}{Pong: true}, nil
	}
}
