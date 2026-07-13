// router_control_test.go — client-side (cmd/sbctl) tests for `sbctl router
// reload` and `sbctl router drain`: the shared connection-error codes
// (E-NET-001, E-ADM-010) re-homed here per F-CS-SP6-001, plus the CLI
// dispatch happy path and sub-verb transition pin for both verbs.
//
// BC/AC coverage map:
//
//	TestRouterReloadDrain_Unreachable_ENET001                        → AC-014 PC-3
//	TestRouterReloadDrain_AuthFailure_EADM010                        → AC-014 PC-3
//	TestRouterReload_CLIDispatch_HappyPath_AcceptedTrue               → AC-015 PC-1, PC-2
//	TestRouterReload_SubVerbTransition_KnownDispatchesUnknownStillExit2 → AC-015 PC-3
//	TestRouterDrain_CLIDispatch_HappyPath_AcceptedTrueOrConnReset     → AC-016 PC-1, PC-2
//	TestRouterDrain_SubVerbTransition_KnownDispatchesUnknownStillExit2  → AC-016 PC-3
//
// Package main (internal test file) for access to runProductionMain and the
// canned-daemon helpers. runRouterReload/runRouterDrain's Red Gate stub
// bodies both panic unconditionally (router_reload.go, router_drain.go) —
// calling either directly in-process would crash the whole cmd/sbctl test
// binary. Every test here therefore drives the real production main() via
// the runProductionMain subprocess helper (production_exit_code_test.go),
// same mechanism used elsewhere in this package (see paths_ping_test.go,
// svtn_test.go) for exercising panic-risk dispatch paths.
package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ─── AC-014 PC-3: shared connection-error codes (re-homed, F-CS-SP6-001) ────

// TestRouterReloadDrain_Unreachable_ENET001 verifies that both `sbctl router
// reload` and `sbctl router drain` report E-NET-001 "daemon unreachable:
// <address>" when the target daemon is unreachable before connection — the
// client-observed half of AC-014's shared connection-error contract.
//
// AC-014 PC-3.
func TestRouterReloadDrain_Unreachable_ENET001(t *testing.T) {
	for _, subVerb := range []string{"reload", "drain"} {
		subVerb := subVerb
		t.Run(subVerb, func(t *testing.T) {
			t.Parallel()

			target := "/nonexistent/path/to/daemon-" + t.Name() + ".sock"
			exitCode, stdout, stderr := runProductionMain(t,
				"--target", target, "--key", testdataKeyPath(t),
				"router", subVerb,
			)
			if exitCode != 1 {
				t.Errorf("AC-014 PC-3: router %s against unreachable daemon: expected exit code 1, got %d\nstdout: %q\nstderr: %q", subVerb, exitCode, stdout, stderr)
			}
			if !strings.Contains(stderr, "E-NET-001") {
				t.Errorf("AC-014 PC-3: router %s stderr must contain \"E-NET-001\"; got: %q", subVerb, stderr)
			}
		})
	}
}

// TestRouterReloadDrain_AuthFailure_EADM010 verifies that both `sbctl router
// reload` and `sbctl router drain` report E-ADM-010 when the connection
// succeeds but Tier-1 authentication fails.
//
// AC-014 PC-3.
func TestRouterReloadDrain_AuthFailure_EADM010(t *testing.T) {
	for _, subVerb := range []string{"reload", "drain"} {
		subVerb := subVerb
		t.Run(subVerb, func(t *testing.T) {
			sockPath, cleanup := stubDaemonSocket(t)
			defer cleanup()
			startAuthFailDaemon(t, sockPath)

			exitCode, stdout, stderr := runProductionMain(t,
				"--target", sockPath, "--key", testdataKeyPath(t),
				"router", subVerb,
			)
			if exitCode != 1 {
				t.Errorf("AC-014 PC-3: router %s against auth-failing daemon: expected exit code 1, got %d\nstdout: %q\nstderr: %q", subVerb, exitCode, stdout, stderr)
			}
			if !strings.Contains(stderr, "E-ADM-010") {
				t.Errorf("AC-014 PC-3: router %s stderr must contain \"E-ADM-010\"; got: %q", subVerb, stderr)
			}
		})
	}
}

// ─── AC-015: sbctl router reload CLI dispatch ────────────────────────────────

// TestRouterReload_CLIDispatch_HappyPath_AcceptedTrue verifies that `sbctl
// router reload --target=<addr>` dispatches router.reload via the existing
// connectAndRun pattern and prints the {"accepted": true} response with exit
// code 0.
//
// AC-015 PC-1, PC-2 / BC-2.09.001 v1.2 PC-1.
func TestRouterReload_CLIDispatch_HappyPath_AcceptedTrue(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	gotCmdCh := make(chan string, 1)
	_ = startCannedDaemonAssertCmd(t, sockPath, json.RawMessage(`{"accepted":true}`), "router.reload", gotCmdCh)

	exitCode, stdout, stderr := runProductionMain(t,
		"--target", sockPath, "--key", testdataKeyPath(t), "--json",
		"router", "reload",
	)

	if exitCode != 0 {
		t.Fatalf("AC-015 PC-1/PC-2: sbctl router reload exit code = %d; want 0\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
	}

	select {
	case gotCmd := <-gotCmdCh:
		if gotCmd != "router.reload" {
			t.Errorf("AC-015 PC-1: dispatched RPC command %q; want %q", gotCmd, "router.reload")
		}
	default:
		t.Error("AC-015 PC-1: no RPC command received by canned daemon — channel empty")
	}

	if !strings.Contains(stdout, `"accepted":true`) && !strings.Contains(stdout, `"accepted": true`) {
		t.Errorf("AC-015 PC-2: stdout must contain the accepted:true response; got: %q", stdout)
	}
}

// TestRouterReload_SubVerbTransition_KnownDispatchesUnknownStillExit2 asserts
// both sides of the router case arm's sub-verb boundary in one run: `sbctl
// router reload` now dispatches via real RPC (exit 0), while `sbctl router
// bogus` — a still-genuinely-unknown sub-verb — continues to exit 2 via the
// unchanged default arm.
//
// AC-015 PC-3.
func TestRouterReload_SubVerbTransition_KnownDispatchesUnknownStillExit2(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()
	_ = startCannedDaemonAssertCmd(t, sockPath, json.RawMessage(`{"accepted":true}`), "router.reload", nil)

	// Known sub-verb: reload now dispatches for real.
	exitCode, stdout, stderr := runProductionMain(t,
		"--target", sockPath, "--key", testdataKeyPath(t), "--json",
		"router", "reload",
	)
	if exitCode != 0 {
		t.Errorf("AC-015 PC-3: sbctl router reload exit code = %d; want 0 (known sub-verb now dispatches)\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout, "accepted") {
		t.Errorf("AC-015 PC-3: sbctl router reload stdout must contain the accepted response; got: %q", stdout)
	}

	// Unknown sub-verb: bogus still falls through to the default arm, exit 2.
	exitCode2, _, stderr2 := runProductionMain(t,
		"--target", sockPath, "--key", testdataKeyPath(t),
		"router", "bogus",
	)
	if exitCode2 != 2 {
		t.Errorf("AC-015 PC-3: sbctl router bogus exit code = %d; want 2 (default arm unchanged)\nstderr: %q", exitCode2, stderr2)
	}
	if !strings.Contains(stderr2, "bogus") {
		t.Errorf("AC-015 PC-3: sbctl router bogus stderr must name the typed unknown verb; got: %q", stderr2)
	}
}

// ─── AC-016: sbctl router drain CLI dispatch ─────────────────────────────────

// TestRouterDrain_CLIDispatch_HappyPath_AcceptedTrueOrConnReset verifies that
// `sbctl router drain --target=<addr>` dispatches router.drain via the
// existing connectAndRun pattern and prints the {"accepted": true} response
// with exit code 0 — OR, per AC-012 PC-3 / BC-2.09.002 PC-3's best-effort-
// delivery framing, tolerates a connection-reset outcome as an expected
// non-error (since drain triggers full daemon shutdown).
//
// AC-016 PC-1, PC-2 / BC-2.09.002 v1.3 Trigger/PC-1.
func TestRouterDrain_CLIDispatch_HappyPath_AcceptedTrueOrConnReset(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()

	gotCmdCh := make(chan string, 1)
	_ = startCannedDaemonAssertCmd(t, sockPath, json.RawMessage(`{"accepted":true}`), "router.drain", gotCmdCh)

	exitCode, stdout, stderr := runProductionMain(t,
		"--target", sockPath, "--key", testdataKeyPath(t), "--json",
		"router", "drain",
	)

	// AC-012 PC-3: connection-reset following (or without) the response is an
	// expected outcome, not a protocol error — tolerate either exit 0 with the
	// accepted body, or a non-zero exit whose message describes a connection
	// reset/closed condition (not an auth or usage failure).
	acceptedInStdout := strings.Contains(stdout, `"accepted":true`) || strings.Contains(stdout, `"accepted": true`)
	if exitCode == 0 {
		if !acceptedInStdout {
			t.Errorf("AC-016 PC-2: exit 0 but stdout missing accepted:true response; got: %q", stdout)
		}
	} else {
		// Non-zero exit is only tolerated if it reads as a connection-teardown
		// artifact, not an auth/usage failure that would indicate real breakage.
		// A panic (runRouterDrain's Red Gate stub panics unconditionally) also
		// exits non-zero and contains neither "E-ADM-010" nor "E-CFG" — without
		// this explicit check a crash would silently fall through the
		// tolerance branch below and be misread as a legitimate connection
		// reset.
		if strings.Contains(stderr, "panic:") {
			t.Errorf("AC-016 PC-2: sbctl router drain crashed (panic), not a tolerated connection-reset; exit=%d stderr=%q", exitCode, stderr)
		} else if strings.Contains(stderr, "E-ADM-010") || strings.Contains(stderr, "E-CFG") {
			t.Errorf("AC-016 PC-2: sbctl router drain failed with an auth/usage error, not a tolerated connection-reset; exit=%d stderr=%q", exitCode, stderr)
		}
	}

	select {
	case gotCmd := <-gotCmdCh:
		if gotCmd != "router.drain" {
			t.Errorf("AC-016 PC-1: dispatched RPC command %q; want %q", gotCmd, "router.drain")
		}
	default:
		t.Error("AC-016 PC-1: no RPC command received by canned daemon — channel empty")
	}
}

// TestRouterDrain_SubVerbTransition_KnownDispatchesUnknownStillExit2 asserts
// both sides of the router case arm's sub-verb boundary in one run: `sbctl
// router drain` now dispatches via real RPC (exit 0, or tolerated connection
// reset per postcondition 2), while `sbctl router bogus` continues to exit 2
// via the unchanged default arm.
//
// AC-016 PC-3.
func TestRouterDrain_SubVerbTransition_KnownDispatchesUnknownStillExit2(t *testing.T) {
	sockPath, cleanup := stubDaemonSocket(t)
	defer cleanup()
	_ = startCannedDaemonAssertCmd(t, sockPath, json.RawMessage(`{"accepted":true}`), "router.drain", nil)

	// Known sub-verb: drain now dispatches for real.
	exitCode, stdout, stderr := runProductionMain(t,
		"--target", sockPath, "--key", testdataKeyPath(t), "--json",
		"router", "drain",
	)
	if exitCode == 0 {
		if !strings.Contains(stdout, "accepted") {
			t.Errorf("AC-016 PC-3: sbctl router drain exit 0 but stdout missing accepted response; got: %q", stdout)
		}
	} else if strings.Contains(stderr, "panic:") {
		// A panic (runRouterDrain's Red Gate stub panics unconditionally)
		// contains neither "E-ADM-010" nor "E-CFG" — without this explicit
		// check a crash would silently pass as a tolerated connection reset.
		t.Errorf("AC-016 PC-3: sbctl router drain crashed (panic), not a tolerated connection-reset; exit=%d stderr=%q", exitCode, stderr)
	} else if strings.Contains(stderr, "E-ADM-010") || strings.Contains(stderr, "E-CFG") {
		t.Errorf("AC-016 PC-3: sbctl router drain must dispatch (exit 0) or tolerate connection-reset, not fail auth/usage; exit=%d stderr=%q", exitCode, stderr)
	}

	// Unknown sub-verb: bogus still falls through to the default arm, exit 2.
	exitCode2, _, stderr2 := runProductionMain(t,
		"--target", sockPath, "--key", testdataKeyPath(t),
		"router", "bogus",
	)
	if exitCode2 != 2 {
		t.Errorf("AC-016 PC-3: sbctl router bogus exit code = %d; want 2 (default arm unchanged)\nstderr: %q", exitCode2, stderr2)
	}
	if !strings.Contains(stderr2, "bogus") {
		t.Errorf("AC-016 PC-3: sbctl router bogus stderr must name the typed unknown verb; got: %q", stderr2)
	}
}
