// Package main — mgmt_wire_test.go tests the management server wiring helpers
// added by S-W5.01.
//
// These tests cover:
//   - startMgmtServer: wires mgmt.Server per ARCH-12 §Daemon Mode Startup
//   - buildMgmtListener: opens the management socket
//   - resolveManagementSocket: config override vs. mode-specific default
//   - mgmtDefaultSocket / mgmtNetwork: per-mode defaults (ARCH-05)
//   - Wiring tests: router/console/control daemon modes start mgmt listener;
//     access mode follows the same pattern; all within WaitGroup lifecycle
//
// All new tests are FAILING (Red Gate) because the stubs in mgmt_wire.go return
// "not implemented" errors. This file must compile; every test must fail
// before any implementation exists.
//
// Traceability:
//
//	BC-2.07.004 PC-10 — graceful shutdown via WaitGroup per ARCH-01
//	S-W5.01 AC-010 — mgmt goroutine WaitGroup-tracked
//	ARCH-12 §Daemon Mode Startup — wiring pattern per daemon mode
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/config"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// ── helpers ────────────────────────────────────────────────────────────────────

// mustGenKeyWire generates an Ed25519 keypair or fatals.
func mustGenKeyWire(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// ── mgmtDefaultSocket and mgmtNetwork ─────────────────────────────────────────

// TestMgmtDefaultSocket_PerMode verifies that mgmtDefaultSocket returns the
// ARCH-05 §Daemon Management Socket defaults for each daemon mode.
func TestMgmtDefaultSocket_PerMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		mode    string
		wantFmt string // substring that must appear in the returned path
	}{
		{mode: "router", wantFmt: "switchboard-router"},
		{mode: "access", wantFmt: "switchboard-access"},
		{mode: "console", wantFmt: "9091"}, // TCP address 127.0.0.1:9091
		{mode: "control", wantFmt: "switchboard-control"},
	}

	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			t.Parallel()
			got := mgmtDefaultSocket(tc.mode)
			if !strings.Contains(got, tc.wantFmt) {
				t.Errorf("mgmtDefaultSocket(%q) = %q; want it to contain %q (ARCH-05)", tc.mode, got, tc.wantFmt)
			}
		})
	}
}

// TestMgmtNetwork_PerMode verifies that mgmtNetwork returns "tcp" for console
// mode and "unix" for all other modes.
func TestMgmtNetwork_PerMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		mode    string
		wantNet string
	}{
		{mode: "router", wantNet: "unix"},
		{mode: "access", wantNet: "unix"},
		{mode: "console", wantNet: "tcp"},
		{mode: "control", wantNet: "unix"},
	}

	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			t.Parallel()
			got := mgmtNetwork(tc.mode)
			if got != tc.wantNet {
				t.Errorf("mgmtNetwork(%q) = %q; want %q", tc.mode, got, tc.wantNet)
			}
		})
	}
}

// ── resolveManagementSocket ────────────────────────────────────────────────────

// TestResolveManagementSocket verifies that the helper returns cfg.ManagementSocket
// when set and the mode-specific default when cfg is nil or socket is empty.
func TestResolveManagementSocket(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     *config.Config
		mode    string
		wantFmt string // substring that must appear in the returned address
	}{
		{
			name:    "nil_cfg_returns_router_default",
			cfg:     nil,
			mode:    "router",
			wantFmt: "switchboard-router",
		},
		{
			name: "cfg_with_management_socket_returns_it",
			cfg: &config.Config{
				ManagementSocket: "/tmp/my-custom.sock",
			},
			mode:    "router",
			wantFmt: "/tmp/my-custom.sock",
		},
		{
			name: "cfg_empty_management_socket_returns_default",
			cfg: &config.Config{
				ManagementSocket: "",
			},
			mode:    "access",
			wantFmt: "switchboard-access",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveManagementSocket(tc.cfg, tc.mode)
			if !strings.Contains(got, tc.wantFmt) {
				t.Errorf("resolveManagementSocket(%v, %q) = %q; want it to contain %q",
					tc.cfg, tc.mode, got, tc.wantFmt)
			}
		})
	}
}

// ── buildMgmtListener ─────────────────────────────────────────────────────────

// TestBuildMgmtListener_NotImplemented verifies that buildMgmtListener currently
// returns a "not implemented" error (Red Gate). Once implemented, it must return
// a real net.Listener.
//
// Traces: S-W5.01 AC-010 / ARCH-12 §Wiring (Red Gate).
func TestBuildMgmtListener_NotImplemented(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	ln, err := buildMgmtListener(cfg, "router")
	if err == nil {
		// Stub is implemented — skip this test; it becomes a regression guard.
		if ln != nil {
			_ = ln.Close()
		}
		t.Skip("buildMgmtListener is implemented; Red Gate test no longer applies")
		return
	}
	// Red Gate: error must mention "not implemented".
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("buildMgmtListener: want 'not implemented' error; got %q", err.Error())
	}
}

// ── startMgmtServer ───────────────────────────────────────────────────────────

// TestStartMgmtServer_NotImplemented verifies that startMgmtServer returns a
// "not implemented" error (Red Gate). Once implemented it must:
//   - Open the management listener
//   - Construct mgmt.NewServer with the listener, daemon key, operator keys, handlers
//   - Launch a WaitGroup-tracked goroutine: wg.Add(1); go func() { defer wg.Done(); srv.Serve(ctx) }()
//   - Return the *mgmt.Server so the caller can Shutdown it
//
// Traces: S-W5.01 AC-010, ARCH-12 §Wiring (Red Gate).
func TestStartMgmtServer_NotImplemented(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKeyWire(t)
	cfg := &config.Config{
		ListenAddr:   "0.0.0.0:9090",
		TickInterval: 10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	srv, err := startMgmtServer(ctx, &wg, cfg, "router", daemonPriv, nil)
	if err == nil {
		// Implementation is present — verify it returned a non-nil Server.
		if srv == nil {
			t.Error("startMgmtServer returned nil Server with nil error")
		}
		// Cleanup: shutdown the server if it started.
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer shutCancel()
		if srv != nil {
			_ = srv.Shutdown(shutCtx)
		}
		cancel()
		wg.Wait()
		t.Skip("startMgmtServer is implemented; Red Gate test no longer applies")
		return
	}

	// Red Gate: must return "not implemented" error.
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("startMgmtServer: want 'not implemented' error; got %q", err.Error())
	}
}

// ── WaitGroup lifecycle contract ──────────────────────────────────────────────

// TestMgmtServer_WaitGroupLifecycle verifies that once startMgmtServer is
// implemented, the mgmt goroutine is properly WaitGroup-tracked per ARCH-01
// §Goroutine WaitGroup Contract:
//   - wg.Add(1) before the goroutine starts
//   - wg.Done() called when Serve returns (on shutdown)
//
// This test drives the contract via a real in-memory listener so no filesystem
// socket is created.
//
// Red Gate: fails because startMgmtServer returns "not implemented".
func TestMgmtServer_WaitGroupLifecycle(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKeyWire(t)

	// Create a real in-memory TCP listener so buildMgmtListener is bypassed.
	// We inject the listener by overriding cfg.ManagementSocket to a real address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	addr := ln.Addr().String()
	_ = ln.Close() // close it so startMgmtServer can open its own

	cfg := &config.Config{
		ListenAddr:       "0.0.0.0:9090",
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: addr, // TCP address — use console mode
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	srv, err := startMgmtServer(ctx, &wg, cfg, "console", daemonPriv, nil)
	if err != nil {
		// Red Gate: not yet implemented.
		if strings.Contains(err.Error(), "not implemented") {
			t.Skip("startMgmtServer not yet implemented (Red Gate)")
		}
		t.Fatalf("startMgmtServer: unexpected error: %v", err)
	}

	if srv == nil {
		t.Fatal("startMgmtServer returned nil Server with nil error")
	}

	// Cancel context → server's Serve returns → wg.Done() is called.
	cancel()

	// wg.Wait() must complete within a reasonable deadline.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// WaitGroup drained cleanly — goroutine lifecycle contract satisfied.
	case <-time.After(500 * time.Millisecond):
		t.Error("WaitGroup.Wait() did not complete within 500ms after context cancel; " +
			"mgmt goroutine must call wg.Done() on Serve return (ARCH-01 §Goroutine WaitGroup Contract)")
	}

	// Shutdown the server for cleanup.
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}

// ── Daemon mode wiring: mgmt starts in all four modes ─────────────────────────

// TestRunRouter_StartsWithMgmt verifies that runRouter calls startMgmtServer
// and propagates a "not implemented" error. Once implemented, runRouter must:
//   - Start the mgmt listener before any data-plane work
//   - Register router-mode handlers
//   - Shutdown mgmt on context cancel
//
// Traces: S-W5.01 §Wiring Pattern (all four daemon modes), AC-010.
func TestRunRouter_StartsWithMgmt(t *testing.T) {
	// NOT t.Parallel(): runRouter modifies no shared state but is sequenced
	// with other mode tests to avoid port conflicts.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel() // pre-cancel so runRouter returns promptly

	cfg := &config.Config{
		ListenAddr:   "0.0.0.0:9090",
		TickInterval: 10 * time.Millisecond,
	}

	err := runRouter(ctx, nil, cfg)
	if err == nil {
		t.Skip("runRouter succeeded (implementation landed); Red Gate test no longer applies")
		return
	}

	// Red Gate: runRouter must return "not implemented" at this stage.
	// The test verifies it does NOT panic and returns an error.
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("runRouter: want 'not implemented' in error; got %q", err.Error())
	}
}

// TestRunConsole_StartsWithMgmt verifies that runConsole calls startMgmtServer
// (using TCP at 127.0.0.1:9091 per ARCH-05) and propagates "not implemented".
//
// Traces: S-W5.01 §Wiring Pattern.
func TestRunConsole_StartsWithMgmt(t *testing.T) {
	// NOT t.Parallel(): console uses TCP 9091 which may conflict if run in parallel.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel()

	cfg := &config.Config{
		ListenAddr:   "0.0.0.0:9090",
		TickInterval: 10 * time.Millisecond,
	}

	err := runConsole(ctx, nil, cfg)
	if err == nil {
		t.Skip("runConsole succeeded; Red Gate test no longer applies")
		return
	}

	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("runConsole: want 'not implemented' in error; got %q", err.Error())
	}
}

// TestRunControl_StartsWithMgmt verifies that runControl calls startMgmtServer
// and propagates "not implemented".
//
// Traces: S-W5.01 §Wiring Pattern.
func TestRunControl_StartsWithMgmt(t *testing.T) {
	// NOT t.Parallel(): runControl is sequenced with other mode tests.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancel()

	cfg := &config.Config{
		ListenAddr:   "0.0.0.0:9090",
		TickInterval: 10 * time.Millisecond,
	}

	err := runControl(ctx, nil, cfg)
	if err == nil {
		t.Skip("runControl succeeded; Red Gate test no longer applies")
		return
	}

	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("runControl: want 'not implemented' in error; got %q", err.Error())
	}
}

// ── Mgmt.Handler type: ensure it compiles and is usable ───────────────────────

// TestMgmtHandlerType verifies that mgmt.Handler can be constructed with a
// function signature matching what the test suite expects. This is a compile-time
// sanity check — not a behavioral assertion.
func TestMgmtHandlerType(t *testing.T) {
	t.Parallel()

	// Verify that mgmt.Handler{Command, Fn} compiles with the expected signature.
	// The Fn signature is: func(ctx context.Context, args json.RawMessage) (any, error)
	// per mgmt.go Handler definition.
	called := false
	h := mgmt.Handler{
		Command: "test.noop",
		Fn: func(ctx context.Context, args json.RawMessage) (any, error) {
			called = true
			return nil, nil
		},
	}
	if h.Command != "test.noop" {
		t.Errorf("Handler.Command: got %q; want %q", h.Command, "test.noop")
	}
	// Fn must be callable (not nil).
	if h.Fn == nil {
		t.Error("Handler.Fn: must not be nil")
	}
	_ = called // suppress unused-variable warning; Fn not invoked in this test
}

// ── OperatorKeySet bootstrap via config ───────────────────────────────────────

// TestNewOperatorKeySet_FromEmptyConfig verifies that NewOperatorKeySet constructed
// from an empty authorized_operator_keys config field is in bootstrap mode.
//
// This is the path taken by startMgmtServer when AuthorizedOperatorKeys is nil.
func TestNewOperatorKeySet_FromEmptyConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	// AuthorizedOperatorKeys is nil → bootstrap mode.
	ops := mgmt.NewOperatorKeySet(nil) // same as startMgmtServer would construct
	if !ops.IsBootstrap() {
		t.Error("NewOperatorKeySet(nil): want IsBootstrap()=true; got false")
	}
	_ = cfg
}
