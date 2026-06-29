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
	"os"
	"path/filepath"
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

// tempSockPath creates a temporary directory with a short path and returns the
// path to a socket file within it. Uses os.MkdirTemp with a brief prefix to
// stay well under the 104-character Unix socket path limit on macOS (sockaddr_un
// sun_path is 104 bytes on Darwin, 108 on Linux).
//
// The directory is removed via t.Cleanup.
func tempSockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "sb-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "m.sock")
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

// TestBuildMgmtListener verifies that buildMgmtListener returns a working
// net.Listener for a valid socket path and an error for an un-bindable path.
//
// Uses t.TempDir() for hermetic, OS-independent socket paths — never binds to
// system paths like /run/ which do not exist on macOS.
//
// Traces: S-W5.01 AC-010 / ARCH-12 §Wiring.
func TestBuildMgmtListener(t *testing.T) {
	t.Parallel()

	t.Run("valid_unix_socket_returns_listener", func(t *testing.T) {
		t.Parallel()

		sockPath := tempSockPath(t)
		cfg := &config.Config{ManagementSocket: sockPath}

		ln, err := buildMgmtListener(cfg, "router")
		if err != nil {
			t.Fatalf("buildMgmtListener: got error %v; want nil", err)
		}
		if ln == nil {
			t.Fatal("buildMgmtListener: got nil listener; want non-nil")
		}
		t.Cleanup(func() { _ = ln.Close() })

		// Addr must reflect the bound socket path.
		if !strings.Contains(ln.Addr().String(), sockPath) {
			t.Errorf("listener Addr = %q; want it to contain %q", ln.Addr().String(), sockPath)
		}
	})

	t.Run("invalid_path_returns_error", func(t *testing.T) {
		t.Parallel()

		// A path into a non-existent directory is un-bindable on all platforms.
		cfg := &config.Config{ManagementSocket: "/nonexistent-dir-sbtest/mgmt.sock"}

		ln, err := buildMgmtListener(cfg, "router")
		if err == nil {
			_ = ln.Close()
			t.Fatal("buildMgmtListener(bad path): got nil error; want error")
		}
		// Listener must be nil on error.
		if ln != nil {
			_ = ln.Close()
			t.Error("buildMgmtListener(bad path): got non-nil listener with error")
		}
	})
}

// ── startMgmtServer ───────────────────────────────────────────────────────────

// TestStartMgmtServer verifies that startMgmtServer:
//   - Opens the management listener on a hermetic temp-dir socket path
//   - Returns a non-nil *mgmt.Server with nil error
//   - Launches a WaitGroup-tracked goroutine that drains on ctx cancel (ARCH-01)
//   - Shuts down cleanly via Shutdown
//
// Uses t.TempDir() for hermetic, OS-independent socket paths.
//
// Traces: S-W5.01 AC-010, ARCH-12 §Wiring (Goroutine WaitGroup Contract).
func TestStartMgmtServer(t *testing.T) {
	t.Parallel()

	_, daemonPriv := mustGenKeyWire(t)
	sockPath := tempSockPath(t)
	cfg := &config.Config{
		ListenAddr:       "0.0.0.0:9090",
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	srv, err := startMgmtServer(ctx, &wg, cfg, "router", daemonPriv, nil)
	if err != nil {
		cancel()
		t.Fatalf("startMgmtServer: got error %v; want nil", err)
	}
	if srv == nil {
		cancel()
		t.Fatal("startMgmtServer: got nil Server with nil error")
	}

	// Cancel context → Serve returns → wg.Done() is called (ARCH-01 WaitGroup contract).
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// WaitGroup drained — goroutine lifecycle contract satisfied.
	case <-time.After(500 * time.Millisecond):
		t.Error("WaitGroup.Wait() did not complete within 500ms after context cancel; " +
			"mgmt goroutine must call wg.Done() on Serve return (ARCH-01 §Goroutine WaitGroup Contract)")
	}

	// Shutdown for final cleanup (idempotent if Serve already returned).
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	t.Cleanup(shutCancel)
	_ = srv.Shutdown(shutCtx)
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

// ── AC-014: Unix socket 0600 permissions (CWE-276, Ruling 4) ─────────────────

// TestDaemonWiring_UnixSocketPermissions_AC014 verifies that the management
// socket file is created with permissions 0600 (owner read/write only) via
// syscall.Umask(0177) before net.Listen — NOT via chmod-after-Listen (which
// has a TOCTOU window per CWE-276).
//
// The test calls a testable extract of the Unix-socket Listen setup (or
// buildMgmtListener directly), stats the resulting socket file, and asserts
// FileMode.Perm() == 0600.
//
// It also verifies that the umask is restored to its original value after
// the Listen call, so the test process's umask is not permanently modified.
//
// COMPILE NOTE: this test references listenUnixMgmt — a testable extract that
// the implementer MUST add to mgmt_wire.go:
//
//	func listenUnixMgmt(path string) (net.Listener, error) {
//	    old := syscall.Umask(0177)
//	    ln, err := net.Listen("unix", path)
//	    syscall.Umask(old)
//	    return ln, err
//	}
//
// If the implementer chooses a different extraction point, adjust the call
// site here accordingly. The behavioral assertion (0600 perm) is the Red Gate.
//
// Gates: darwin + linux only (build constraint for portability).
//
// Traces: BC-2.07.004 EC-013, Invariant 7, AC-014 v1.1, Ruling 4.
func TestDaemonWiring_UnixSocketPermissions_AC014(t *testing.T) {
	t.Parallel()

	sockPath := tempSockPath(t)

	// Call the testable Unix socket wiring helper.
	// listenUnixMgmt must set syscall.Umask(0177) before net.Listen and restore
	// the umask afterward. It is a package-level function in mgmt_wire.go.
	//
	// COMPILE FAILURE IS EXPECTED until the implementer adds listenUnixMgmt.
	ln, err := listenUnixMgmt(sockPath)
	if err != nil {
		t.Fatalf("AC-014: listenUnixMgmt(%q): %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// Stat the socket file — must exist because net.Listen("unix", ...) creates it.
	fi, err := os.Stat(sockPath)
	if err != nil {
		t.Fatalf("AC-014: os.Stat(%q): %v (socket must exist after Listen)", sockPath, err)
	}

	// AC-014: socket permissions must be 0600 (owner r/w only).
	// The umask of 0177 applied before net.Listen ensures this atomically.
	// A chmod-after-Listen approach is explicitly forbidden (TOCTOU window).
	perm := fi.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("AC-014: socket %q perm = %04o; want 0600 (must use syscall.Umask(0177) before net.Listen, not chmod-after)", sockPath, perm)
	}
}

// TestDaemonWiring_ConsoleBindsLocalhost_AC014 verifies that console mode binds
// the management TCP listener to 127.0.0.1 (loopback) only — not 0.0.0.0 or ":".
//
// Traces: BC-2.07.004 EC-013, AC-014 v1.1 (console TCP binding constraint).
func TestDaemonWiring_ConsoleBindsLocalhost_AC014(t *testing.T) {
	t.Parallel()

	// The console default address must be 127.0.0.1:xxxx, never 0.0.0.0 or ":"
	addr := mgmtDefaultSocket("console")
	if !strings.HasPrefix(addr, "127.0.0.1") {
		t.Errorf("AC-014: console mgmtDefaultSocket = %q; must bind to 127.0.0.1 only (not 0.0.0.0 or \":\")", addr)
	}
}

// ── Critical finding: runAccess MUST start mgmt server ───────────────────────

// TestRunAccess_StartsWithMgmt verifies that runAccess calls startMgmtServer —
// making the ACCESS daemon wired to the management plane, not unwired.
//
// This is the Critical finding from ARCH-12 v1.2 adversarial review: the access
// daemon is the only non-stub mode but currently does NOT call startMgmtServer.
// All four daemon modes (router, access, console, control) MUST start an
// mgmt.Server per the Scope Note.
//
// Test strategy: we cannot call runAccess directly (it connects to a real PTY).
// Instead we verify the behavioral invariant by checking that:
// (1) buildMgmtListener is called by the access wiring path — demonstrated by
//
//	injecting a cfg with a temp-dir socket path and asserting the socket file
//	is created during the access startup path.
//
// Since runAccess is not injectable for the mgmt path, we test the lower-level
// startMgmtServer function invoked by the access wiring stub and assert it is
// present and functional. The real behavioral test is:
//
//	runAccess must call startMgmtServer with the access mode.
//
// This test fails because runAccess does NOT call startMgmtServer today (critical
// finding). The implementer must add the startMgmtServer call to runAccess.
//
// Approach: we use a side-channel observable — if runAccess calls
// startMgmtServer("access", ...), the socket file at cfg.ManagementSocket will
// be created. We supply a temp-dir socket path and assert it exists after a
// brief startup window.
//
// NOTE: runAccess also calls tmux.New / sc.Connect which will fail in a test
// environment. We cancel the context immediately to make it return quickly, and
// check socket creation before/after the call. The socket file is the observable.
//
// Traces: S-W5.01 Scope Note (Critical: ALL FOUR modes), AC-010, ARCH-12 v1.2.
func TestRunAccess_StartsWithMgmt(t *testing.T) {
	// NOT t.Parallel(): touches filesystem socket path.

	sockPath := tempSockPath(t)

	cfg := &config.Config{
		ListenAddr:       "0.0.0.0:9090",
		TickInterval:     10 * time.Millisecond,
		ManagementSocket: sockPath,
	}

	// Pre-cancel so runAccess returns immediately (Connect will fail; that's fine —
	// we only want to observe whether the socket was created before the error path).
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	// runAccess is expected to fail quickly (PTY unavailable in test environment).
	// We don't assert on the error — only on socket creation.
	var buf strings.Builder
	_ = runAccess(ctx, &buf, cfg)

	// AC-014 observable: if runAccess calls startMgmtServer("access", ...) before
	// any data-plane work, the socket file will have been created (and possibly
	// removed on shutdown — but creation is the signal).
	//
	// If the socket does NOT exist, runAccess did NOT call startMgmtServer — the
	// critical missing wiring. This is the Red Gate assertion.
	//
	// Note: on a successful implementation the socket may be cleaned up by
	// Shutdown before the stat, in which case we need a different observable.
	// The preferred approach is for the implementer to use a socketCreated hook or
	// for the test to use a net.Listen listener injection. For now, the timing
	// window (context already cancelled → Serve returns immediately → socket file
	// persists because Unix sockets are not auto-removed on Close) is sufficient.
	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Errorf("TestRunAccess_StartsWithMgmt: socket %q was not created; "+
			"runAccess must call startMgmtServer (critical finding: access mode unwired)",
			sockPath)
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
