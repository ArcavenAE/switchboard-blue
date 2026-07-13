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
	"fmt"
	"os"
	"syscall"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// wireRouterControlHandlers registers the router.reload and router.drain RPC
// handlers on srv. Router-mode-exclusive: only runRouter calls this, at the
// same phase as wireMetricsHandlers, before serveMgmtServer starts the Serve
// goroutine (register-before-serve invariant, F-P2L1-001).
//
// configPath, sighupCh, and drainRequestCh are threaded through to the
// handler bodies — configPath lets router.reload's handler check
// configPath == "" synchronously for AC-011 PC-3's defense-in-depth guard;
// sighupCh is the channel router.reload synthesizes a signal onto;
// drainRequestCh is the channel router.drain sends on.
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
// AC-011 PC-3 defense-in-depth guard: configPath == "" is unreachable via any
// real daemon startup path (runRouter's entry guard plus main.go's "router"
// case together guarantee configPath != "" for every router instance that
// reaches registration) — nonetheless checked synchronously, before any
// signal synthesis, returning the bare E-CFG-004 literal per error-taxonomy.md
// v4.9 Variant 3 if that invariant is ever violated.
//
// PC-1/PC-2: otherwise synthesizes syscall.SIGHUP onto sighupCh — the same
// coalescing semantics signal.Notify itself uses (a reload already pending
// silently drops the second request) — and responds fire-and-forget
// {"accepted": true}, matching the shared AC-014 wire contract.
//
// AC-011 / BC-2.09.001 v1.2 PC-1 (RPC-trigger note).
func routerReloadRPCHandler(configPath string, sighupCh chan os.Signal) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		if configPath == "" {
			return nil, fmt.Errorf("E-CFG-004: reload not applicable: daemon started without --config")
		}
		select {
		case sighupCh <- syscall.SIGHUP:
		default:
		}
		return struct {
			Accepted bool `json:"accepted"`
		}{Accepted: true}, nil
	}
}

// routerDrainRPCHandler returns the router.drain RPC handler.
//
// Sends on drainRequestCh — select{...default:} coalescing so an
// already-in-flight drain request is a no-op — and responds fire-and-forget
// {"accepted": true}. The RPC connection is expected to be severed shortly
// after as the daemon proceeds through the shutdown sequence (AC-012 PC-3);
// that is handled entirely at the transport/select-loop level, not here.
//
// AC-012 / BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note).
func routerDrainRPCHandler(drainRequestCh chan struct{}) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, _ json.RawMessage) (any, error) {
		select {
		case drainRequestCh <- struct{}{}:
		default:
		}
		return struct {
			Accepted bool `json:"accepted"`
		}{Accepted: true}, nil
	}
}
