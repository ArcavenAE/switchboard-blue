// svtn_test.go — tests for the top-level `sbctl svtn` subcommand tree:
// `svtn status --name=<svtn-name>` (Decision 2), `svtn destroy` migration
// shim (Decision 3), and `svtn` top-level case-arm dispatch (Decision 2/3
// dispatch structure).
//
// BC/AC coverage map:
//
//	TestSvtnStatus_CLIDispatch_BareTopLevel_NameFlag  → AC-008, BC-2.07.001 v1.14 PC-4
//	TestSvtnDestroy_TopLevelShim_UsageErrorRedirect_Exit2 → AC-009 PC-1
//	TestSvtnDestroy_TopLevelShim_NoRPCDispatch        → AC-009 PC-2, PC-3, PC-4
//	TestSvtn_UnknownSubVerb_UsageErrorExit2           → AC-010 PC-3
//
// runSvtn/runSvtnStatus/runSvtnDestroyShim's Red Gate stub bodies all panic
// unconditionally (svtn.go), regardless of arguments — calling any of them
// directly in-process would crash the whole cmd/sbctl test binary (an
// unrecovered panic terminates the process; testing's per-test recover only
// guards the goroutine running t.Run, not sibling tests) and take every
// unrelated test in this package down with it. Every test here therefore
// dispatches through the real compiled main() via the runProductionMain
// subprocess helper (production_exit_code_test.go), matching this repo's
// established pattern for exercising panic-risk dispatch paths
// (main_test.go's TestSubprocessMain_* hooks; see also paths_ping_test.go).
package main

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"
)

// startSvtnStatusCannedDaemon starts a stub daemon that performs the ADR-012
// handshake and then captures both the RPC command name and the raw request
// args onto the supplied channels before responding with responseData.
//
// Mirrors startCannedDaemonAssertCmd (router_status_test.go) but additionally
// exposes the request args — needed to verify AC-008 PC-1's exact wire
// contract ({"name": "<svtn-name>"}), which command-only assertion cannot see.
func startSvtnStatusCannedDaemon(t *testing.T, sockPath string, responseData json.RawMessage, gotCmdCh chan<- string, gotArgsCh chan<- json.RawMessage) net.Listener {
	t.Helper()

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("startSvtnStatusCannedDaemon: listen on %s: %v", sockPath, err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				_ = c.SetDeadline(time.Now().Add(10 * time.Second))

				nonce := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
				challenge := map[string]string{
					"type":       "challenge",
					"nonce":      nonce,
					"daemon_sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				}
				if err := json.NewEncoder(c).Encode(challenge); err != nil {
					return
				}
				var authResp map[string]string
				if err := json.NewDecoder(c).Decode(&authResp); err != nil {
					return
				}
				authOK := map[string]string{"type": "auth_ok", "daemon_version": "test-stub"}
				if err := json.NewEncoder(c).Encode(authOK); err != nil {
					return
				}

				var req map[string]json.RawMessage
				if err := json.NewDecoder(c).Decode(&req); err != nil {
					return
				}
				var reqID, cmd string
				if idRaw, ok := req["id"]; ok {
					_ = json.Unmarshal(idRaw, &reqID)
				}
				if cmdRaw, ok := req["command"]; ok {
					_ = json.Unmarshal(cmdRaw, &cmd)
				}
				if gotCmdCh != nil {
					select {
					case gotCmdCh <- cmd:
					default:
					}
				}
				if gotArgsCh != nil {
					argsRaw, hasArgs := req["args"]
					if !hasArgs {
						argsRaw = json.RawMessage("null")
					}
					select {
					case gotArgsCh <- argsRaw:
					default:
					}
				}

				rpcResp := map[string]interface{}{
					"type": "response",
					"id":   reqID,
					"ok":   true,
					"data": responseData,
				}
				_ = json.NewEncoder(c).Encode(rpcResp)
			}(conn)
		}
	}()
	return ln
}

// ─── AC-008: sbctl svtn status CLI dispatch ──────────────────────────────────

// TestSvtnStatus_CLIDispatch_BareTopLevel_NameFlag verifies that `sbctl svtn
// status --name=<svtn-name>` dispatches directly to admin.svtn.status with
// wire args {"name": "<svtn-name>"} (bare top-level — not routed through
// `sbctl admin` framing, dialed via the top-level --target flag, same shape
// as `paths list`/`router status`), and that a missing --name flag is a
// client-side E-CFG-001 usage error (exit 2) via usageErrf, per AC-008 PC-3.
//
// AC-008 / BC-2.07.001 PC-4 (CLI dispatch note).
func TestSvtnStatus_CLIDispatch_BareTopLevel_NameFlag(t *testing.T) {
	t.Run("happy_path_dispatches_admin_svtn_status_with_name_arg", func(t *testing.T) {
		sockPath, cleanup := stubDaemonSocket(t)
		defer cleanup()

		cannedStatus := json.RawMessage(`{"svtn_id":"deadbeef","name":"mynet","created_at":"2026-07-12T00:00:00Z","key_counts":{"control":1,"console":0,"access":2}}`)
		gotCmdCh := make(chan string, 1)
		gotArgsCh := make(chan json.RawMessage, 1)
		_ = startSvtnStatusCannedDaemon(t, sockPath, cannedStatus, gotCmdCh, gotArgsCh)

		exitCode, stdout, stderr := runProductionMain(t,
			"--target", sockPath, "--key", testdataKeyPath(t),
			"svtn", "status", "--name=mynet",
		)
		if exitCode != 0 {
			t.Fatalf("AC-008: expected exit code 0, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
		}

		select {
		case gotCmd := <-gotCmdCh:
			if gotCmd != "admin.svtn.status" {
				t.Errorf("AC-008: sbctl sent RPC command %q; want %q", gotCmd, "admin.svtn.status")
			}
		default:
			t.Error("AC-008: no RPC command received by canned daemon — channel empty")
		}

		select {
		case gotArgs := <-gotArgsCh:
			var argsMap map[string]json.RawMessage
			if parseErr := json.Unmarshal(gotArgs, &argsMap); parseErr != nil {
				t.Fatalf("AC-008: request args are not a JSON object: %v (raw: %s)", parseErr, gotArgs)
			}
			var name string
			if nameRaw, ok := argsMap["name"]; ok {
				_ = json.Unmarshal(nameRaw, &name)
			} else {
				t.Fatal("AC-008 / BC-2.07.001 PC-4: request args missing \"name\" field")
			}
			if name != "mynet" {
				t.Errorf("AC-008: request args name = %q; want %q", name, "mynet")
			}
			// Wire contract per interface-definitions.md §420: args is exactly
			// {"name": "<svtn-name>"} — no other keys.
			if len(argsMap) != 1 {
				t.Errorf("AC-008: request args has %d keys; want exactly 1 (\"name\"); got: %v", len(argsMap), mapKeys(argsMap))
			}
		default:
			t.Error("AC-008: no request args received by canned daemon — channel empty")
		}

		var env struct {
			OK bool `json:"ok"`
		}
		if parseErr := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); parseErr != nil {
			t.Fatalf("AC-008: stdout is not a valid JSON envelope: %v\nraw: %q", parseErr, stdout)
		}
		if !env.OK {
			t.Fatal("AC-008: envelope ok must be true")
		}
	})

	t.Run("missing_name_flag_usage_error_exit2", func(t *testing.T) {
		t.Parallel()

		// target points at a socket that does not exist — if flag validation
		// did not fire before dial, the resulting error would be E-NET-001.
		target := "/nonexistent/should-not-be-dialed-" + t.Name() + ".sock"

		exitCode, stdout, stderr := runProductionMain(t,
			"--target", target, "--key", testdataKeyPath(t),
			"svtn", "status",
		)
		if exitCode != 2 {
			t.Fatalf("AC-008 PC-3: expected exit code 2 for missing --name, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
		}
		if !strings.Contains(stderr, "--name") {
			t.Errorf("AC-008 PC-3: expected stderr to reference the missing --name flag; got: %q", stderr)
		}
		if !strings.Contains(strings.ToLower(stderr), "required") {
			t.Errorf("AC-008 PC-3: expected stderr to say --name is required; got: %q", stderr)
		}
		// AC-008 PC-3 / Ruling 2 Addendum: "Missing --name → E-CFG-001
		// (client-side), exit 2" — the token itself is spec'd, not just the
		// prose, per the sibling precedents the addendum cites
		// (usageErrf("E-CFG-001: admin list-keys: --svtn is required")).
		// Without this check, a bare code-less error message (the
		// runAdminSvtnDestroy anti-pattern the addendum explicitly flags)
		// would pass.
		if !strings.Contains(stderr, "E-CFG-001") {
			t.Errorf("AC-008 PC-3 / Ruling 2 Addendum: expected stderr to contain the \"E-CFG-001\" token (error-taxonomy.md v4.9 client-side variant); got: %q", stderr)
		}
		// Flag validation must fire before any dial attempt (client-side
		// E-CFG-001 pattern, error-taxonomy.md).
		if strings.Contains(stderr, "E-NET-001") {
			t.Errorf("AC-008 PC-3: missing --name must be caught before dialing (no E-NET-001); got: %q", stderr)
		}
	})
}

// ─── AC-009: sbctl svtn destroy top-level migration shim ────────────────────

// TestSvtnDestroy_TopLevelShim_UsageErrorRedirect_Exit2 verifies that `sbctl
// svtn destroy` always exits 2 with the exact redirect text naming the
// canonical `sbctl admin svtn destroy` form.
//
// AC-009 PC-1 / Decision 3.
func TestSvtnDestroy_TopLevelShim_UsageErrorRedirect_Exit2(t *testing.T) {
	t.Parallel()

	const wantText = "svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'"

	target := "/nonexistent/should-not-be-dialed-" + t.Name() + ".sock"
	exitCode, stdout, stderr := runProductionMain(t,
		"--target", target, "--key", testdataKeyPath(t),
		"svtn", "destroy",
	)
	if exitCode != 2 {
		t.Fatalf("AC-009 PC-1: expected exit code 2, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
	}
	if !strings.Contains(stderr, wantText) {
		t.Errorf("AC-009 PC-1: stderr = %q; want it to contain exact redirect text %q", stderr, wantText)
	}
}

// TestSvtnDestroy_TopLevelShim_NoRPCDispatch verifies that `sbctl svtn
// destroy` (any arguments) never dispatches admin.svtn.destroy, never parses
// --id/--name, and never invokes the confirm gate — the top-level shim
// always returns the identical redirect text regardless of what flags are
// supplied.
//
// target is a nonexistent socket; if the shim attempted to dial (i.e.
// mis-routed into an RPC path), the error would surface E-NET-001 instead of
// the exact redirect text.
//
// AC-009 PC-2, PC-3, PC-4 / Decision 3.
func TestSvtnDestroy_TopLevelShim_NoRPCDispatch(t *testing.T) {
	const wantText = "svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'"

	cases := []struct {
		name string
		args []string
	}{
		{name: "bare_destroy_no_args", args: []string{"destroy"}},
		{name: "destroy_with_name_flag", args: []string{"destroy", "--name=mynet"}},
		{name: "destroy_with_id_flag", args: []string{"destroy", "--id=deadbeef"}},
		{name: "destroy_with_confirm_and_yes", args: []string{"destroy", "--name=mynet", "--confirm=deadbeef", "--yes"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			target := "/nonexistent/should-not-be-dialed-" + t.Name() + ".sock"
			fullArgs := append([]string{"--target", target, "--key", testdataKeyPath(t), "svtn"}, tc.args...)
			exitCode, stdout, stderr := runProductionMain(t, fullArgs...)

			if exitCode != 2 {
				t.Errorf("AC-009: expected exit code 2, got %d\nstdout: %q\nstderr: %q", exitCode, stdout, stderr)
			}
			if !strings.Contains(stderr, wantText) {
				t.Errorf("AC-009: stderr = %q; want it to contain exact redirect text %q (proves --id/--name/--confirm/--yes are never parsed — PC-2)", stderr, wantText)
			}
			if strings.Contains(stderr, "E-NET-001") {
				t.Errorf("AC-009: stderr must not mention E-NET-001 — no dial/RPC dispatch may occur (PC-3, PC-4); got: %q", stderr)
			}
		})
	}
}

// ─── AC-010: sbctl svtn top-level case arm dispatch ──────────────────────────

// TestSvtn_UnknownSubVerb_UsageErrorExit2 verifies that an unknown sub-verb
// under `svtn` (including the bare `sbctl svtn` invocation with no sub-verb
// at all) returns a usage error, exit 2 — the same shape as the existing
// paths/router case arms' default arms, each of which names its own case-arm
// in the error text ("paths: unknown sub-verb...", "router: unknown
// subcommand...").
//
// AC-010 PC-3 / Decision 2 + Decision 3 dispatch structure.
func TestSvtn_UnknownSubVerb_UsageErrorExit2(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "bare_svtn_no_subverb", args: []string{}},
		{name: "svtn_list_unknown_subverb", args: []string{"list"}},
		{name: "svtn_bogus_unknown_subverb", args: []string{"bogus"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			target := "/nonexistent/should-not-be-dialed-" + t.Name() + ".sock"
			fullArgs := append([]string{"--target", target, "--key", testdataKeyPath(t), "svtn"}, tc.args...)
			exitCode, stdout, stderr := runProductionMain(t, fullArgs...)

			if exitCode != 2 {
				t.Errorf("AC-010 PC-3: svtn %v: expected exit code 2, got %d\nstdout: %q\nstderr: %q", tc.args, exitCode, stdout, stderr)
			}
			// A Go panic also exits 2 (runSvtn's Red Gate stub panics
			// unconditionally, and its trace text happens to contain the
			// substring "svtn" via the function name and file path) — the
			// bare exit-code and substring checks above cannot tell a clean
			// usage error from a crash that merely mentions "svtn" in its
			// stack trace. This explicit check is what actually makes the
			// test fail against the stub and pass only once runSvtn returns
			// a real usageError instead of panicking.
			if strings.Contains(stderr, "panic:") {
				t.Errorf("AC-010 PC-3: svtn %v: stderr must be a clean usage error, not a panic trace; got: %q", tc.args, stderr)
			}
			// Same case-arm-naming convention as "paths: unknown sub-verb..."
			// and "router: unknown subcommand...".
			if !strings.Contains(stderr, "svtn") {
				t.Errorf("AC-010 PC-3: svtn %v: stderr must name the \"svtn\" case arm (matches paths/router convention); got: %q", tc.args, stderr)
			}
		})
	}
}
