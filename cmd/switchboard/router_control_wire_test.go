// router_control_wire_test.go — server-side (cmd/switchboard) tests for
// wireRouterControlHandlers: router.reload/router.drain registration,
// router-mode exclusivity, the sighupCh/drainRequestCh bridging, and the
// AC-011 PC-3 defense-in-depth guard.
//
// BC/AC coverage map:
//
//	TestRouterReload_BridgesToSighupCh_CodePathIdentical              → AC-011 PC-1, PC-2 (integration)
//	TestRouterReload_NoConfigLoaded_ECFG004                           → AC-011 PC-3 (unit)
//	TestRouterDrain_BridgesToShutdownSequence_ViaDrainRequestCh       → AC-012 PC-1, PC-2
//	TestRouterDrain_ConnectionSeveredAfterAccepted_NotAnError         → AC-012 PC-3
//	TestWireRouterControlHandlers_RegisterBeforeServe                 → AC-013 (registration half)
//	TestWireRouterControlHandlers_RouterModeExclusive_OtherModesERPC010 → AC-013 PC-2
//	TestRouterReloadDrain_TierOneAuthOnly_FireAndForgetAcceptedTrue   → AC-014 PC-1, PC-2
//
// Subprocess-isolation note: routerReloadRPCHandler/routerDrainRPCHandler
// are implemented (no longer Red Gate stubs). mgmt.Server.Serve dispatches
// each RPC on its own connection-handling goroutine (internal/mgmt/mgmt.go)
// with no per-connection recover() — an unrecovered handler panic would
// terminate the WHOLE process, not just the calling test, because
// recover() only catches panics on the same goroutine that panicked, and
// the panicking goroutine would belong to mgmt.Server, not to any
// *testing.T's tRunner. Any test that performs a REAL RPC round trip
// against one of these handlers therefore still runs inside a subprocess
// (TestSubprocessRouterControlScenario, invoked via
// runRouterControlScenario) — retained as a regression defense so a future
// handler panic is contained to the child process's own exit code, the
// same isolation runProductionMain already provides for cmd/sbctl
// (production_exit_code_test.go). Tests that don't need to actually invoke
// a handler body (registration-ordering checks, mode-exclusivity against a
// server that never registers these handlers at all) stay in-process — no
// panic risk exists there.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/mgmt"
)

// startRouterControlWireServer builds a bare mgmt.Server, registers
// wireRouterControlHandlers on it (before Serve starts — register-before-
// serve invariant, F-P2L1-001), and starts Serve. Returns the socket path and
// the daemon private key — the sole authorized caller in bootstrap mode
// (nil OperatorKeySet), mirroring admin_handlers_e2e_test.go's startE2EServer.
//
// Registers the daemon key into testDaemonKeys so sendAdminRPC/sendAdminRPCAsKey
// (admin_handlers_e2e_test.go) can authenticate against this server directly.
func startRouterControlWireServer(t *testing.T, configPath string, sighupCh chan os.Signal, drainRequestCh chan struct{}) (socketPath string, daemonPriv ed25519.PrivateKey) {
	t.Helper()

	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("startRouterControlWireServer: generate daemon keypair: %v", err)
	}

	dir, err := os.MkdirTemp("", "sw-rcw-*")
	if err != nil {
		t.Fatalf("startRouterControlWireServer: MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath = fmt.Sprintf("%s/m.sock", dir)
	if len(socketPath) > 104 {
		t.Fatalf("startRouterControlWireServer: socket path %q length %d exceeds 104-byte limit", socketPath, len(socketPath))
	}

	ln, err := listenUnixMgmt(socketPath)
	if err != nil {
		t.Fatalf("startRouterControlWireServer: listen: %v", err)
	}

	ops := mgmt.NewOperatorKeySet(nil)
	srv := mgmt.NewServer(ln, daemonPriv, ops, nil, "dev",
		mgmt.WithHandshakeTimeout(2*time.Second),
		mgmt.WithRPCIdleTimeout(5*time.Second),
	)

	// Register BEFORE Serve — same ordering runRouter uses at Decision 4's
	// registration point (F-P2L1-001). If registration happened after Serve
	// started, mgmt.Register would return a non-nil error here and this
	// helper would fail the test immediately, rather than silently succeeding
	// with an unregistered server.
	if err := wireRouterControlHandlers(srv, configPath, sighupCh, drainRequestCh); err != nil {
		t.Fatalf("startRouterControlWireServer: wireRouterControlHandlers: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(ctx)
	}()

	testDaemonKeysMu.Lock()
	testDaemonKeys[socketPath] = daemonPriv
	testDaemonKeysMu.Unlock()

	t.Cleanup(func() {
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(shutCtx)
		shutCancel()
		<-done
		testDaemonKeysMu.Lock()
		delete(testDaemonKeys, socketPath)
		testDaemonKeysMu.Unlock()
	})

	return socketPath, daemonPriv
}

// ─── Subprocess isolation for real-RPC-dispatch scenarios ───────────────────

// TestSubprocessRouterControlScenario is the re-exec landing point for every
// integration-level test in this file that dispatches a real RPC against a
// live, in-process mgmt.Server. router.reload/router.drain handlers are
// implemented (no longer Red Gate stubs) — subprocess isolation is retained
// as a regression defense; see the file header for why. In the parent test
// process (env var absent), the hook skips immediately.
func TestSubprocessRouterControlScenario(t *testing.T) {
	scenario := os.Getenv("SW_TEST_ROUTER_CONTROL_SCENARIO")
	if scenario == "" {
		t.Skip("subprocess hook — skip in parent process")
	}

	switch scenario {
	case "reload_bridges_sighup":
		sighupCh := make(chan os.Signal, 1)
		drainRequestCh := make(chan struct{}, 1)
		socketPath, daemonPriv := startRouterControlWireServer(t, "/tmp/does-not-need-to-exist.yaml", sighupCh, drainRequestCh)

		resp := sendAdminRPC(t, socketPath, daemonPriv, "router.reload", nil)

		// PC-1: the handler must synthesize a signal onto sighupCh — the same
		// channel signal.Notify's OS-signal path already consumes.
		select {
		case sig := <-sighupCh:
			if sig != syscall.SIGHUP {
				t.Fatalf("AC-011 PC-1: signal synthesized onto sighupCh = %v; want syscall.SIGHUP", sig)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("AC-011 PC-1: router.reload handler did not synthesize a signal onto sighupCh within 2s")
		}

		// PC-2 / AC-014 PC-2: response contract — {"accepted": true}, no error.
		if errObj, _ := resp["error"].(map[string]any); errObj != nil {
			t.Fatalf("AC-011 PC-2: router.reload returned an error response: %+v", errObj)
		}
		data, _ := resp["data"].(map[string]any)
		if accepted, _ := data["accepted"].(bool); !accepted {
			t.Fatalf("AC-011 PC-2 / AC-014 PC-2: response data = %+v; want accepted:true", data)
		}

	case "drain_bridges_drainrequestch":
		sighupCh := make(chan os.Signal, 1)
		drainRequestCh := make(chan struct{}, 1)
		socketPath, daemonPriv := startRouterControlWireServer(t, "", sighupCh, drainRequestCh)

		resp := sendAdminRPC(t, socketPath, daemonPriv, "router.drain", nil)

		select {
		case <-drainRequestCh:
			// PC-1: the handler must send on drainRequestCh.
		case <-time.After(2 * time.Second):
			t.Fatal("AC-012 PC-1: router.drain handler did not send on drainRequestCh within 2s")
		}

		if errObj, _ := resp["error"].(map[string]any); errObj != nil {
			t.Fatalf("AC-012 PC-2 / AC-014 PC-2: router.drain returned an error response: %+v", errObj)
		}
		data, _ := resp["data"].(map[string]any)
		if accepted, _ := data["accepted"].(bool); !accepted {
			t.Fatalf("AC-012 PC-2 / AC-014 PC-2: response data = %+v; want accepted:true", data)
		}

	case "drain_conn_severed_not_error":
		sighupCh := make(chan os.Signal, 1)
		drainRequestCh := make(chan struct{}, 1)
		socketPath, daemonPriv := startRouterControlWireServer(t, "", sighupCh, drainRequestCh)

		resp := sendAdminRPC(t, socketPath, daemonPriv, "router.drain", nil)
		if errObj, _ := resp["error"].(map[string]any); errObj != nil {
			t.Fatalf("AC-012 PC-3: router.drain returned an error response: %+v", errObj)
		}
		data, _ := resp["data"].(map[string]any)
		if accepted, _ := data["accepted"].(bool); !accepted {
			t.Fatalf("AC-012 PC-3: response data = %+v; want accepted:true before the connection is severed", data)
		}

		select {
		case <-drainRequestCh:
		case <-time.After(2 * time.Second):
			t.Fatal("AC-012 PC-3: first router.drain dispatch did not signal drainRequestCh within 2s")
		}

		// sendAdminRPC already closed its connection immediately after
		// reading the response above — the "severed connection" scenario has
		// already occurred by this point. Prove it left no corrupted server
		// state by dispatching a second, independent RPC.
		resp2 := sendAdminRPC(t, socketPath, daemonPriv, "router.drain", nil)
		if errObj, _ := resp2["error"].(map[string]any); errObj != nil {
			t.Fatalf("AC-012 PC-3: server did not recover cleanly after a severed connection — second dispatch returned an error: %+v", errObj)
		}
		select {
		case <-drainRequestCh:
		case <-time.After(2 * time.Second):
			t.Fatal("AC-012 PC-3: second router.drain dispatch did not signal drainRequestCh — server may have been corrupted by the earlier severed connection")
		}

	case "tier_one_auth_reload":
		runTierOneAuthScenario(t, "router.reload")
	case "tier_one_auth_drain":
		runTierOneAuthScenario(t, "router.drain")

	default:
		t.Fatalf("TestSubprocessRouterControlScenario: unknown scenario %q", scenario)
	}
}

// runTierOneAuthScenario is the shared body for the tier_one_auth_reload and
// tier_one_auth_drain subprocess scenarios (AC-014 PC-1, PC-2).
func runTierOneAuthScenario(t *testing.T, cmd string) {
	t.Helper()

	sighupCh := make(chan os.Signal, 1)
	drainRequestCh := make(chan struct{}, 1)
	// AC-014's premise is a production-reachable router: the AC-011 PC-3
	// defense-in-depth chain (runRouter's cfg==nil guard + main.go's
	// "router" case) guarantees configPath != "" for every router instance
	// that reaches wireRouterControlHandlers registration — configPath=="",
	// used elsewhere in this file only to drive TestRouterReload_
	// NoConfigLoaded_ECFG004's own guard test, would collide with that
	// test's E-CFG-004 assertion on the same handler for the reload
	// subcase. Mirrors the reload_bridges_sighup scenario's non-empty path.
	socketPath, daemonPriv := startRouterControlWireServer(t, "/tmp/does-not-need-to-exist.yaml", sighupCh, drainRequestCh)

	resp := sendAdminRPC(t, socketPath, daemonPriv, cmd, nil)
	if errObj, _ := resp["error"].(map[string]any); errObj != nil {
		t.Fatalf("AC-014 PC-1: %s with Tier-1 auth only must succeed; got error: %+v", cmd, errObj)
	}
	data, _ := resp["data"].(map[string]any)
	if len(data) != 1 {
		t.Fatalf("AC-014 PC-2: %s response data = %+v; want exactly one field (accepted)", cmd, data)
	}
	if accepted, _ := data["accepted"].(bool); !accepted {
		t.Fatalf("AC-014 PC-2: %s response data = %+v; want accepted:true (fire-and-forget)", cmd, data)
	}
}

// runRouterControlScenario runs TestSubprocessRouterControlScenario in a
// child process with SW_TEST_ROUTER_CONTROL_SCENARIO=scenario, returning the
// exit code and captured stdout/stderr. Exit code 0 means every assertion in
// the scenario's t.Fatalf/t.Errorf calls passed and no panic occurred;
// anything else (assertion failure via go test's own nonzero exit, or a
// crash via an unrecovered panic) is a Red Gate failure signal.
func runRouterControlScenario(t *testing.T, scenario string) (exitCode int, stdout, stderr string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestSubprocessRouterControlScenario$")
	cmd.Env = append(os.Environ(), "SW_TEST_ROUTER_CONTROL_SCENARIO="+scenario)

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr == nil {
		return 0, stdout, stderr
	}
	exitErr, ok := runErr.(*exec.ExitError)
	if !ok {
		t.Fatalf("runRouterControlScenario(%s): subprocess execution failed with non-exit error: %v", scenario, runErr)
	}
	return exitErr.ExitCode(), stdout, stderr
}

// ─── AC-011: router.reload bridges into the shipped SIGHUP-reload path ──────

// TestRouterReload_BridgesToSighupCh_CodePathIdentical verifies that the
// router.reload RPC handler synthesizes a signal onto the (bidirectional)
// sighupCh — select { case sighupCh <- syscall.SIGHUP: default: } — and that
// the RPC responds {"accepted": true} with no error, matching the shared
// AC-014 wire contract.
//
// Runs via subprocess isolation — see file header.
//
// AC-011 PC-1, PC-2 / BC-2.09.001 v1.2 PC-1.
func TestRouterReload_BridgesToSighupCh_CodePathIdentical(t *testing.T) {
	t.Parallel()

	exitCode, stdout, stderr := runRouterControlScenario(t, "reload_bridges_sighup")
	if exitCode != 0 {
		t.Errorf("AC-011 PC-1/PC-2: subprocess scenario failed (exit %d) — router.reload does not yet bridge to sighupCh per spec\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}
}

// TestRouterReload_NoConfigLoaded_ECFG004 is the AC-011 PC-3 defense-in-depth
// guard test. It calls routerReloadRPCHandler directly with configPath=""
// (Ruling 4 Addendum v1.1's invocation-pattern note: no live daemon
// required) and asserts the exact E-CFG-004 literal from error-taxonomy.md
// v4.9 Variant 3, and that no signal is synthesized onto sighupCh when the
// guard fires.
//
// The handler call happens on this test's own goroutine (no server, no
// background dispatch), so a handler panic here — unlike the RPC-dispatch
// tests above — CAN be safely recovered without any cross-goroutine risk.
// The recover is a safety net only, retained as a regression defense: if
// the handler were to panic, err is set to a message that does not match
// wantText below, so the test still fails as an honest assertion failure
// instead of a process crash.
//
// Test level: unit.
//
// AC-011 PC-3 / Ruling 4 Addendum v1.1, FO(c) DISCHARGED.
func TestRouterReload_NoConfigLoaded_ECFG004(t *testing.T) {
	t.Parallel()

	sighupCh := make(chan os.Signal, 1)
	handler := routerReloadRPCHandler("", sighupCh)

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("handler panicked instead of returning an error: %v", r)
			}
		}()
		_, err = handler(context.Background(), json.RawMessage("{}"))
	}()

	if err == nil {
		t.Fatal("AC-011 PC-3: routerReloadRPCHandler with configPath=\"\" must return a non-nil error")
	}

	const wantText = "E-CFG-004: reload not applicable: daemon started without --config"
	if err.Error() != wantText {
		t.Errorf("AC-011 PC-3: error text = %q; want exact literal %q (error-taxonomy.md v4.9 Variant 3)", err.Error(), wantText)
	}

	// The guard fires BEFORE any signal synthesis — configPath=="" is
	// unreachable via any real daemon startup path, so no reload should ever
	// be signaled on this branch.
	select {
	case sig := <-sighupCh:
		t.Errorf("AC-011 PC-3: handler must not synthesize onto sighupCh when configPath==\"\"; got signal %v", sig)
	default:
	}
}

// ─── AC-012: router.drain bridges into the shipped shutdown sequence ────────

// TestRouterDrain_BridgesToShutdownSequence_ViaDrainRequestCh verifies that
// the router.drain RPC handler sends on drainRequestCh — select {
// case drainRequestCh <- struct{}{}: default: } — and that the RPC responds
// {"accepted": true} with no error.
//
// Reaching the select loop's third arm and the shared shutdown: label is
// exercised at the full-runRouter level by
// TestRunRouter_DrainRequestChThirdSelectArm_ReachesShutdown_SameExitParityAsSIGTERM
// (mgmt_wire_test.go, AC-013) — this test covers only the handler-level
// wiring (Decision 4's "genuinely new channel" bridging). Runs via subprocess
// isolation — see file header.
//
// AC-012 PC-1, PC-2 / BC-2.09.002 v1.3 Trigger/PC-1.
func TestRouterDrain_BridgesToShutdownSequence_ViaDrainRequestCh(t *testing.T) {
	t.Parallel()

	exitCode, stdout, stderr := runRouterControlScenario(t, "drain_bridges_drainrequestch")
	if exitCode != 0 {
		t.Errorf("AC-012 PC-1: subprocess scenario failed (exit %d) — router.drain does not yet send on drainRequestCh per spec\nstdout: %s\nstderr: %s",
			exitCode, stdout, stderr)
	}
}

// TestRouterDrain_ConnectionSeveredAfterAccepted_NotAnError verifies AC-012
// PC-3: a connection severed following (or even without) the {"accepted":
// true} response is an expected outcome, not a protocol error. Proven at the
// server level by confirming the server continues serving subsequent callers
// cleanly after a caller's connection is torn down immediately post-response
// (sendAdminRPC closes its connection right after reading the response) — if
// the severed connection had corrupted server state, the second independent
// dispatch below would fail or hang. Runs via subprocess isolation — see
// file header.
//
// AC-012 PC-3 / BC-2.09.002 PC-3 best-effort-delivery framing, extended to
// the triggering RPC itself.
func TestRouterDrain_ConnectionSeveredAfterAccepted_NotAnError(t *testing.T) {
	t.Parallel()

	exitCode, stdout, stderr := runRouterControlScenario(t, "drain_conn_severed_not_error")
	if exitCode != 0 {
		t.Errorf("AC-012 PC-3: subprocess scenario failed (exit %d)\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}
}

// ─── AC-013: registration — router-mode-exclusive, register-before-serve ────

// TestWireRouterControlHandlers_RegisterBeforeServe verifies that
// wireRouterControlHandlers succeeds when called before Serve starts, and
// that a second call issued AFTER Serve has started is rejected — the same
// register-before-serve invariant (F-P2L1-001) every other handler
// registration in this codebase obeys (mgmt.Register itself, exercised
// directly by RegisterMetricsHandlers). This proves
// wireRouterControlHandlers participates in the invariant rather than
// bypassing mgmt.Register, without ever invoking a registered handler body —
// no RPC dispatch occurs, so no panic risk exists here.
//
// AC-013 PC-1.
func TestWireRouterControlHandlers_RegisterBeforeServe(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp("", "sw-rbs-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath := fmt.Sprintf("%s/m.sock", dir)

	ln, err := listenUnixMgmt(socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	_, daemonPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate daemon keypair: %v", err)
	}
	ops := mgmt.NewOperatorKeySet(nil)
	srv := mgmt.NewServer(ln, daemonPriv, ops, nil, "dev")

	sighupCh := make(chan os.Signal, 1)
	drainRequestCh := make(chan struct{}, 1)

	// Before Serve starts: registration must succeed cleanly.
	if err := wireRouterControlHandlers(srv, "", sighupCh, drainRequestCh); err != nil {
		t.Fatalf("AC-013 PC-1: wireRouterControlHandlers before Serve: unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.Serve(ctx) }()

	// Poll wireRouterControlHandlers until it observes serving==true.
	// listenUnixMgmt (above) already creates the socket file the instant
	// net.Listen returns — well before Serve is ever called — so the
	// socket file's existence is NOT a valid "Serve has started" signal
	// here. Server.serving.Store(true) is Serve's very first statement
	// (internal/mgmt/mgmt.go), so a short bounded poll converges as soon as
	// the goroutine above is scheduled. Each pre-serving call in the loop
	// succeeds and appends unused duplicate handler entries — harmless,
	// since this server instance is never dispatched against.
	deadline := time.Now().Add(2 * time.Second)
	var postServeErr error
	for time.Now().Before(deadline) {
		postServeErr = wireRouterControlHandlers(srv, "", sighupCh, drainRequestCh)
		if postServeErr != nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if postServeErr == nil {
		t.Error("AC-013 PC-1: wireRouterControlHandlers called after Serve has started must return a non-nil error")
	}
}

// TestWireRouterControlHandlers_RouterModeExclusive_OtherModesERPC010
// verifies that router.reload and router.drain are unreachable on a daemon
// that never calls wireRouterControlHandlers — mirroring runAccess,
// runConsole, and runControl, none of which call it (Decision 4). Both
// commands must return E-RPC-010 (unknown command). startE2EServer(t, nil)
// registers no handlers at all, so this test never invokes a panicking
// closure — no subprocess isolation needed.
//
// AC-013 PC-2.
func TestWireRouterControlHandlers_RouterModeExclusive_OtherModesERPC010(t *testing.T) {
	es := startE2EServer(t, nil)

	for _, cmd := range []string{"router.reload", "router.drain"} {
		t.Run(cmd, func(t *testing.T) {
			resp := sendAdminRPC(t, es.socketPath, es.daemonPriv, cmd, nil)
			errObj, _ := resp["error"].(map[string]any)
			if errObj == nil {
				t.Fatalf("AC-013 PC-2: %s on a non-router-mode daemon must return an error; got success: %+v", cmd, resp)
			}
			code, _ := errObj["code"].(string)
			if code != "E-RPC-010" {
				t.Errorf("AC-013 PC-2: %s error code = %q; want \"E-RPC-010\" (unknown command)", cmd, code)
			}
		})
	}
}

// ─── AC-014: router.reload/router.drain wire contract ───────────────────────

// TestRouterReloadDrain_TierOneAuthOnly_FireAndForgetAcceptedTrue verifies
// that both router.reload and router.drain require only Tier-1 operator-key
// authentication (no additional Tier-2 gate — router mode has no
// SVTNManager/RoleControl concept) and respond fire-and-forget {"accepted":
// true} with empty request args. Runs via subprocess isolation — see file
// header.
//
// AC-014 PC-1, PC-2.
func TestRouterReloadDrain_TierOneAuthOnly_FireAndForgetAcceptedTrue(t *testing.T) {
	cases := []struct {
		cmd      string
		scenario string
	}{
		{"router.reload", "tier_one_auth_reload"},
		{"router.drain", "tier_one_auth_drain"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.cmd, func(t *testing.T) {
			t.Parallel()

			exitCode, stdout, stderr := runRouterControlScenario(t, tc.scenario)
			if exitCode != 0 {
				t.Errorf("AC-014 PC-1/PC-2: %s subprocess scenario failed (exit %d)\nstdout: %s\nstderr: %s",
					tc.cmd, exitCode, stdout, stderr)
			}
		})
	}
}
