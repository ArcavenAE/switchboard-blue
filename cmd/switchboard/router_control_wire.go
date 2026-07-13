// router_control_wire.go — registers the router-mode-exclusive RPC control
// handlers (router.reload, router.drain) on the management server
// (S-BL.CLI-SURFACE-COMPLETION Decision 4).
//
// wireRouterControlHandlers is called from runRouter only — runAccess,
// runConsole, and runControl never call it, since router.reload/router.drain
// are meaningless on those modes (no sighupCh/drain-coordinator concept).
// Both handlers bridge into the already-shipped SIGHUP-reload
// (S-7.04-FU-SIGHUP-RELOAD) and DRAIN/shutdown (S-7.04-FU-DRAIN-WIRE) code
// paths via the channels threaded in from runRouter — no reload/drain logic
// is duplicated here.
//
// Purity classification (ARCH-09): boundary — wires effectful handlers into
// the management server; no business logic lives here.
package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// wireRouterControlHandlers registers the router.reload and router.drain RPC
// handlers on srv. Router-mode-exclusive: only runRouter calls this, at the
// same phase as wireMetricsHandlers, before serveMgmtServer starts the Serve
// goroutine (register-before-serve invariant, F-P2L1-001).
//
// configPath, sighupCh, and drainRequestCh are threaded through for the
// eventual handler bodies — configPath lets router.reload's handler check
// configPath == "" synchronously for AC-011 PC-3's defense-in-depth guard;
// sighupCh is the channel router.reload synthesizes a signal onto;
// drainRequestCh is the channel router.drain sends on. This stub's handler
// bodies do not yet consume them (Task 4 Green step).
//
// AC-013 / Decision 4 (registration point).
func wireRouterControlHandlers(srv *mgmt.Server, configPath string, sighupCh chan os.Signal, drainRequestCh chan struct{}) error {
	if err := srv.Register(mgmt.Handler{Command: "router.reload", Fn: routerReloadRPCHandler(configPath, sighupCh)}); err != nil {
		return err
	}
	return srv.Register(mgmt.Handler{Command: "router.drain", Fn: routerDrainRPCHandler(drainRequestCh)})
}

// routerReloadRPCHandler returns the router.reload RPC handler.
//
// AC-011 / BC-2.09.001 v1.2 PC-1 (RPC-trigger note).
//
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 4 (Green step) implements the
// sighupCh synthesis + AC-011 PC-3 defense-in-depth guard. Red Gate: the
// returned closure panics unconditionally so no test can accidentally pass
// before the Green step.
func routerReloadRPCHandler(configPath string, sighupCh chan os.Signal) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		_, _ = configPath, sighupCh
		panic("not implemented: S-BL.CLI-SURFACE-COMPLETION routerReloadRPCHandler")
	}
}

// routerDrainRPCHandler returns the router.drain RPC handler.
//
// AC-012 / BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note).
//
// STUB — S-BL.CLI-SURFACE-COMPLETION Task 4 (Green step) implements the
// drainRequestCh send. Red Gate: the returned closure panics unconditionally
// so no test can accidentally pass before the Green step.
func routerDrainRPCHandler(drainRequestCh chan struct{}) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		_ = drainRequestCh
		panic("not implemented: S-BL.CLI-SURFACE-COMPLETION routerDrainRPCHandler")
	}
}
