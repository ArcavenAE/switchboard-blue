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
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 1 (Green step) implements the
// empty-args-in / {"pong": true}-out logic with zero PathTracker interaction
// (AC-004 postcondition 3). Red Gate: body panics unconditionally so no test
// can accidentally pass before the Green step.
func pingHandler() func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		panic("not implemented: S-BL.CLI-SURFACE-COMPLETION pingHandler")
	}
}
