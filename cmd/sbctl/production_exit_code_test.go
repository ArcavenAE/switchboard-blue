// Package main — production exit-code and sessions sub-verb validation tests.
//
// RED GATE: these tests exercise PRODUCTION main() behavior via the
// TestSubprocessMain_* re-exec pattern established in main_test.go.
// They MUST FAIL against the current code:
//
//   - F-P5P6-A-001 (HIGH): main.go:82-84 collapses all errors to os.Exit(1).
//     Usage errors (E-CFG-012, E-CFG-013, missing required flags, unknown
//     subcommands) must exit 2, not 1.
//   - F-P5P6-A-006 (LOW): main.go:34-37 prints usage and exits 0 when no
//     subcommand is given.  Must exit 2.
//   - F-P5P6-A-003 (MED): main.go:49-50 dispatches ALL `sbctl sessions <x>`
//     to sessions.list, silently misdispatching attach/detach/status/bogus.
//     Must route non-list verbs to a clear error (exit 2).
//
// Spec authority: interface-definitions.md v1.18 §133 (confirm-gate summary),
// §174 (exit-code table).
//
// These tests do NOT require a live daemon.  Every case verified here fires
// a usage-error path before connectAndRun is reached.
package main

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// ── Subprocess hooks ──────────────────────────────────────────────────────────

// TestSubprocessMain_ProductionExitCode is the single re-exec landing point
// used by all F-P5P6-A-001 and F-P5P6-A-006 subprocess cases.
//
// The env var SBCTL_TEST_PROD_ARGS encodes the full argument list to pass to
// main() as space-separated tokens (simple tokenisation — no quoting needed
// for the flag forms tested here).  The function resets flag.CommandLine and
// os.Args, then calls main() which will call os.Exit internally.
//
// In the parent test process (env var absent), t.Skip fires immediately.
func TestSubprocessMain_ProductionExitCode(t *testing.T) {
	argsEnv := os.Getenv("SBCTL_TEST_PROD_ARGS")
	if argsEnv == "" {
		t.Skip("subprocess hook — skip in parent process")
	}

	// Reset flag state so main()'s flag.Parse() works cleanly.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Reconstruct os.Args: binary name + tokens from env.
	tokens := strings.Fields(argsEnv)
	os.Args = append([]string{os.Args[0]}, tokens...)

	main()
	// main() always calls os.Exit — reaching here is a test setup bug.
	t.Fatal("main() returned without calling os.Exit — unexpected")
}

// TestSubprocessMain_NoArgsExitTwo is the re-exec landing point for
// TestProductionMain_NoArgs_ExitTwo (F-P5P6-A-006).
//
// When SBCTL_TEST_PROD_NOARGS_P6=1 is set, it resets flag.CommandLine and
// os.Args to just the binary name (no subcommand), then calls main().
// In the parent process (env var absent), t.Skip fires immediately.
//
// Named with the P6 suffix to avoid collision with the existing
// TestSubprocessMain_NoArgs hook in main_test.go, which was written for the
// old AC-012 exit-0 contract and exercises different expected behavior.
func TestSubprocessMain_NoArgsExitTwo(t *testing.T) {
	if os.Getenv("SBCTL_TEST_PROD_NOARGS_P6") != "1" {
		t.Skip("subprocess hook — skip in parent process")
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{os.Args[0]}

	main()
	t.Fatal("main() returned without calling os.Exit — unexpected")
}

// TestSubprocessMain_SessionsSubVerb is the re-exec landing point for
// F-P5P6-A-003 sessions sub-verb tests.
//
// Behaviour mirrors TestSubprocessMain_ProductionExitCode: resets flag state,
// reconstructs os.Args from SBCTL_TEST_SESSIONS_ARGS, calls main().
func TestSubprocessMain_SessionsSubVerb(t *testing.T) {
	argsEnv := os.Getenv("SBCTL_TEST_SESSIONS_ARGS")
	if argsEnv == "" {
		t.Skip("subprocess hook — skip in parent process")
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	tokens := strings.Fields(argsEnv)
	os.Args = append([]string{os.Args[0]}, tokens...)

	main()
	t.Fatal("main() returned without calling os.Exit — unexpected")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// runProductionMain runs a subprocess landing in TestSubprocessMain_ProductionExitCode
// with os.Args set to args (space-joined into SBCTL_TEST_PROD_ARGS).
// Returns the exit code, stdout, and stderr.
func runProductionMain(t *testing.T, args ...string) (exitCode int, stdout, stderr string) {
	t.Helper()
	return runProductionHook(t, "TestSubprocessMain_ProductionExitCode",
		"SBCTL_TEST_PROD_ARGS", strings.Join(args, " "))
}

// runSessionsMain runs a subprocess landing in TestSubprocessMain_SessionsSubVerb.
func runSessionsMain(t *testing.T, args ...string) (exitCode int, stdout, stderr string) {
	t.Helper()
	return runProductionHook(t, "TestSubprocessMain_SessionsSubVerb",
		"SBCTL_TEST_SESSIONS_ARGS", strings.Join(args, " "))
}

// runProductionHook is the shared exec helper for both hooks.
func runProductionHook(t *testing.T, testName, envKey, envVal string) (exitCode int, stdout, stderr string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), envKey+"="+envVal)

	// Explicitly attach /dev/null as stdin so stdinIsTTY() returns false
	// in the subprocess (non-interactive session guard for E-CFG-013).
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open /dev/null: %v", err)
	}
	t.Cleanup(func() { _ = devNull.Close() })
	cmd.Stdin = devNull

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
		t.Fatalf("subprocess execution failed with non-exit error: %v", runErr)
	}
	return exitErr.ExitCode(), stdout, stderr
}

// ── F-P5P6-A-001 (HIGH): production main() usage-error exit codes ─────────────

// TestProductionMain_UsageErrors_ExitTwo verifies that PRODUCTION main()
// exits 2 — not 1 — for all usage-error cases identified across
// Phase 5 Pass 6 (F-P5P6-A-001) and Phase 5 Pass 7 adversarial findings
// F-P5P7-A-001 (console tree), F-P5P7-A-002 (router metrics --svtn),
// and F-P5P7-A-003 (router status --target).
//
// RED (Pass 6 cases 1-6): current main.go:82-84 collapses ALL errors from
// the dispatch switch to os.Exit(1), so every case below will observe exit
// code 1 instead of 2.
//
// RED (Pass 7 cases 7-12): console.go and router_metrics.go / router_status.go
// return plain fmt.Errorf for usage errors — the errors.As discriminator in
// main() cannot detect them as usageError, so they exit 1 instead of 2.
// Every usage path in cases 7-12 fires BEFORE connectAndRun is reached
// (daemon-free: no network I/O is attempted).
//
// Spec authority: interface-definitions.md v1.19 §174 (exit-code table),
// §86-88 (console --session required on attach/switch), §78-81 (router flags).
// Findings: F-P5P6-A-001, F-P5P7-A-001 (OBS-P5P7-A-001 site inventory),
// F-P5P7-A-002 (router metrics --svtn), F-P5P7-A-003 (router status --target).
func TestProductionMain_UsageErrors_ExitTwo(t *testing.T) {
	t.Parallel()

	keyPath := testdataKeyPath(t)
	// A target that nothing is listening on; for usage-error paths
	// connectAndRun is never reached, so the target doesn't matter.
	// We still supply a plausible address so flag.Parse doesn't error
	// on --target before we hit the usage-error branch.
	target := "127.0.0.1:19984"

	cases := []struct {
		name             string
		args             []string // full argument list passed to main() after the binary name
		wantExitCode     int
		wantStderrSubstr string // must appear in stderr
		// findingID is the canonical finding tag for failure messages.
		// F-P5P8-B-001: cases 1-6 trace to F-P5P6-A-001; cases 7-12 trace to
		// F-P5P7-A-001/002/003.  A shared hardcoded tag misattributes failures.
		findingID string
	}{
		{
			// Case 1: E-CFG-012 — --yes + --confirm combined on destroy.
			// The confirm gate fires BEFORE connectAndRun; no daemon needed.
			// Spec §133: "Combining --yes with --confirm is a usage error
			// (E-CFG-012, exit 2)."
			name: "destroy_yes_plus_confirm_E-CFG-012",
			args: []string{
				"--target", target, "--key", keyPath,
				"admin", "svtn", "destroy",
				"--name", "foo",
				"--yes",
				"--confirm", "SVTN-aabbccdd",
			},
			wantExitCode:     2,
			wantStderrSubstr: "E-CFG-012",
			findingID:        "F-P5P6-A-001",
		},
		{
			// Case 2: E-CFG-013 — non-TTY session, no --confirm, no --yes on destroy.
			// stdinIsTTY() returns false (stdin=/dev/null in subprocess).
			// Spec §133: "In a non-interactive session (no TTY) where neither
			// --confirm nor --yes is supplied, the command exits with E-CFG-013
			// (exit 2)."
			name: "destroy_non_tty_no_confirm_E-CFG-013",
			args: []string{
				"--target", target, "--key", keyPath,
				"admin", "svtn", "destroy",
				"--name", "foo",
			},
			wantExitCode:     2,
			wantStderrSubstr: "E-CFG-013",
			findingID:        "F-P5P6-A-001",
		},
		{
			// Case 3: missing required --name flag on destroy.
			// runAdminSvtnDestroy returns error before connectAndRun.
			name: "destroy_missing_name",
			args: []string{
				"--target", target, "--key", keyPath,
				"admin", "svtn", "destroy",
			},
			wantExitCode:     2,
			wantStderrSubstr: "--name",
			findingID:        "F-P5P6-A-001",
		},
		{
			// Case 4: unknown admin subcommand.
			// runAdmin default arm returns error.
			name: "admin_unknown_subcommand",
			args: []string{
				"--target", target, "--key", keyPath,
				"admin", "bogus",
			},
			wantExitCode:     2,
			wantStderrSubstr: "bogus",
			findingID:        "F-P5P6-A-001",
		},
		{
			// Case 5: admin key register — missing required --key flag.
			// runAdminKeyRegister returns error before connectAndRun.
			name: "key_register_missing_key",
			args: []string{
				"--target", target, "--key", keyPath,
				"admin", "key", "register",
				"--svtn", "my-svtn",
			},
			wantExitCode:     2,
			wantStderrSubstr: "--key",
			findingID:        "F-P5P6-A-001",
		},
		{
			// Case 6: unknown top-level subcommand.
			// Post-#65 the default arm returns usageErrf; exit 2 is mapped by
			// the errors.As discriminator in main(), not by a direct os.Exit(2).
			// This case verifies the discriminator path stays correct when the
			// dispatch logic is refactored.
			name: "top_level_unknown_subcommand",
			args: []string{
				"--target", target, "--key", keyPath,
				"bogus",
			},
			wantExitCode:     2,
			wantStderrSubstr: "bogus",
			findingID:        "F-P5P6-A-001",
		},
		// ── Cases 7-12: F-P5P7-A-001/002/003 — console/router usage errors ──
		// console.go and router_*.go currently return plain fmt.Errorf for all
		// of these usage errors.  The errors.As discriminator in main() cannot
		// match them as *usageError, so each exits 1 (operational) instead of 2
		// (usage).  All six paths fire before connectAndRun — daemon-free.
		{
			// Case 7: bare `sbctl console` (no sub-verb).
			// F-P5P7-A-001 site 1; interface-definitions.md v1.19 §86.
			// runConsole() returns fmt.Errorf — must become usageErrf.
			// Stderr must name the available sub-verbs (attach/detach/switch).
			name: "console_bare_no_subverb",
			args: []string{
				"--target", target, "--key", keyPath,
				"console",
			},
			wantExitCode:     2,
			wantStderrSubstr: "attach",
			findingID:        "F-P5P7-A-001",
		},
		{
			// Case 8: `sbctl console bogus` — unknown sub-verb.
			// F-P5P7-A-001 site 2; interface-definitions.md v1.19 §86.
			// runConsole() default arm returns fmt.Errorf — must become usageErrf.
			// Stderr must contain the unknown verb name.
			name: "console_unknown_subverb",
			args: []string{
				"--target", target, "--key", keyPath,
				"console", "bogus",
			},
			wantExitCode:     2,
			wantStderrSubstr: "bogus",
			findingID:        "F-P5P7-A-001",
		},
		{
			// Case 9: `sbctl console attach` without --session.
			// F-P5P7-A-001 sites 3-4; interface-definitions.md v1.19 §87.
			// runConsoleAttach() returns fmt.Errorf — must become usageErrf.
			// Usage validation fires before connectAndRun; daemon-free.
			name: "console_attach_missing_session",
			args: []string{
				"--target", target, "--key", keyPath,
				"console", "attach",
			},
			wantExitCode:     2,
			wantStderrSubstr: "--session",
			findingID:        "F-P5P7-A-001",
		},
		{
			// Case 10: `sbctl console switch` without --session.
			// F-P5P7-A-001 sites 5-6; interface-definitions.md v1.19 §88.
			// runConsoleSwitch() returns fmt.Errorf — must become usageErrf.
			// Usage validation fires before connectAndRun; daemon-free.
			name: "console_switch_missing_session",
			args: []string{
				"--target", target, "--key", keyPath,
				"console", "switch",
			},
			wantExitCode:     2,
			wantStderrSubstr: "--session",
			findingID:        "F-P5P7-A-001",
		},
		{
			// Case 11: `sbctl router metrics` without --svtn.
			// F-P5P7-A-002; interface-definitions.md v1.19 §78-79.
			// runRouterMetrics() calls writeError then returns plain fmt.Errorf —
			// must return usageErrf so main() maps it to exit 2.
			// Usage validation fires before connectAndRun; daemon-free.
			// Stderr must contain "--svtn" or the E-CFG-010 code.
			name: "router_metrics_missing_svtn",
			args: []string{
				"--target", target, "--key", keyPath,
				"router", "metrics",
			},
			wantExitCode:     2,
			wantStderrSubstr: "E-CFG-010",
			findingID:        "F-P5P7-A-002",
		},
		{
			// Case 12: `sbctl router status --target` with no trailing value.
			// F-P5P7-A-003 site 1; interface-definitions.md v1.19 §80-81.
			// runRouterStatus() returns plain fmt.Errorf — must become usageErrf.
			// Usage validation fires before the net.Dial call; daemon-free.
			// Stderr must contain "--target" or E-CFG-010.
			name: "router_status_target_missing_value",
			args: []string{
				"--target", target, "--key", keyPath,
				"router", "status", "--target",
			},
			wantExitCode:     2,
			wantStderrSubstr: "E-CFG-010",
			findingID:        "F-P5P7-A-003",
		},
		{
			// Case 13: `sbctl router status --target=` (empty value after equals).
			// DRIFT-P5P7-O1; interface-definitions.md v1.19 §80-81.
			// The manual --target scanner in router_status.go handles the
			// `--target <value>` form (case 12) via a slice bounds check; the
			// `--target=<value>` form is handled via strings.TrimPrefix which
			// yields the empty string when the operator writes `--target=`.
			// SPEC-3 covers this at the binary level; this Go-level case pins
			// the exact scanner path so future refactors of the manual scan
			// don't regress it.  Daemon-free (usage validation fires pre-dial).
			name: "router_status_target_empty_after_equals",
			args: []string{
				"--target", target, "--key", keyPath,
				"router", "status", "--target=",
			},
			wantExitCode:     2,
			wantStderrSubstr: "E-CFG-010",
			findingID:        "DRIFT-P5P7-O1",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exitCode, stdout, stderr := runProductionMain(t, tc.args...)

			if exitCode != tc.wantExitCode {
				t.Errorf("%s: expected exit code %d, got %d\nstdout: %q\nstderr: %q",
					tc.findingID, tc.wantExitCode, exitCode, stdout, stderr)
			}
			if tc.wantStderrSubstr != "" && !strings.Contains(stderr, tc.wantStderrSubstr) {
				t.Errorf("%s: expected stderr to contain %q; got: %q",
					tc.findingID, tc.wantStderrSubstr, stderr)
			}
		})
	}
}

// ── F-P5P6-A-006 (LOW): bare `sbctl` must exit 2 ─────────────────────────────

// TestProductionMain_NoArgs_ExitTwo verifies that invoking sbctl with no
// subcommand arguments exits 2 and lists available subcommands on stderr.
//
// RED: current main.go:34-37 calls os.Exit(0) on the no-args path.
// Per interface-definitions.md v1.18 §174, "no args" is a usage error (exit 2).
//
// Spec authority: interface-definitions.md v1.18 §174.
// Finding: F-P5P6-A-006.
//
// Historical note (DRIFT-P5P9, reconciled in Burst 23): the test
// TestSbctl_NoSubcommand_ExitsZero that is referenced above no longer exists by
// that name — it was renamed TestSbctl_NoSubcommand_ExitsTwoAfterP6 in Burst 23
// when the no-args exit-code behavior was corrected from 0 to 2 per §174.
// TestSbctl_NoSubcommand_ExitsTwoAfterP6 is now a green guard for exit 2.
func TestProductionMain_NoArgs_ExitTwo(t *testing.T) {
	t.Parallel()

	// Use the dedicated no-args hook (TestSubprocessMain_NoArgsExitTwo) so that
	// the subprocess env is unambiguously set and the hook does not skip.
	// Named with the P6 suffix to avoid collision with the existing
	// TestSubprocessMain_NoArgs hook (main_test.go, AC-012 exit-0 contract).
	cmd := exec.Command(os.Args[0], "-test.run=TestSubprocessMain_NoArgsExitTwo$")
	cmd.Env = append(os.Environ(), "SBCTL_TEST_PROD_NOARGS_P6=1")

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open /dev/null: %v", err)
	}
	t.Cleanup(func() { _ = devNull.Close() })
	cmd.Stdin = devNull

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout := outBuf.String()
	stderr := errBuf.String()

	exitCode := 0
	if runErr != nil {
		exitErr, ok := runErr.(*exec.ExitError)
		if !ok {
			t.Fatalf("subprocess execution failed with non-exit error: %v", runErr)
		}
		exitCode = exitErr.ExitCode()
	}

	if exitCode != 2 {
		t.Errorf("F-P5P6-A-006: expected exit code 2 for no-args invocation, got %d\nstdout: %q\nstderr: %q",
			exitCode, stdout, stderr)
	}

	// Usage output must enumerate at least two concrete verb names so the operator
	// knows what to type next.  Requiring two specific verbs guards against a
	// single-word match being satisfied by incidental output.
	// OBS-P5P10-B-001: dropped the || "subcommand" disjunct (too permissive) and
	// require both "sessions" AND "admin" to appear.
	combined := stdout + stderr
	if !strings.Contains(combined, "sessions") {
		t.Errorf("F-P5P6-A-006: expected usage output to contain verb \"sessions\"; got stdout=%q stderr=%q",
			stdout, stderr)
	}
	if !strings.Contains(combined, "admin") {
		t.Errorf("F-P5P6-A-006: expected usage output to contain verb \"admin\"; got stdout=%q stderr=%q",
			stdout, stderr)
	}
}

// ── F-P5P6-A-003 (MED): sessions sub-verb validation ─────────────────────────

// TestProductionMain_Sessions_SubVerbValidation verifies that `sbctl sessions`
// dispatches correctly based on the sub-verb:
//
//   - `sessions list`   → connects to daemon (may exit 1 on connection refused)
//   - `sessions status` → connects to daemon (may exit 1 on connection refused)
//     — added by S-BL.CONSOLE-OBS; sessions.status RPC is now implemented.
//   - `sessions attach` → exits 2 with a clear not-implemented error naming
//     the verb and citing backlog deferral
//   - `sessions detach` → exits 2 (same shape as attach)
//   - `sessions bogus`  → exits 2 with unknown-sub-verb error naming "bogus"
//   - `sessions` alone  → exits 2 (missing sub-verb)
//
// RED: current main.go:49-50 routes ALL `sbctl sessions <x>` to
// sessions.list regardless of the sub-verb, silently misdispatching
// attach/detach/status/bogus.  Every case except "list" will currently
// attempt a daemon connection (and exit 1 on connection refused) instead
// of exiting 2 with a usage error.
//
// Spec authority: interface-definitions.md v1.18 §71-73;
// S-BL.CONSOLE-OBS story (`status` promotion from stub to real dispatch).
// Finding: F-P5P6-A-003.
func TestProductionMain_Sessions_SubVerbValidation(t *testing.T) {
	t.Parallel()

	keyPath := testdataKeyPath(t)
	// Nothing listens here — for sessions.list/status the exit will be 1
	// (E-NET-001), for usage-error paths it should be 2 without touching
	// the network.
	target := "127.0.0.1:19983"

	cases := []struct {
		name             string
		subVerb          string // the word after "sessions"
		wantExitCode     int
		wantStderrSubstr string // must appear in stderr (case-insensitive checked below)
	}{
		{
			// attach → not-implemented, exit 2, stderr names the verb
			name:             "attach_not_implemented",
			subVerb:          "attach",
			wantExitCode:     2,
			wantStderrSubstr: "attach",
		},
		{
			// detach → not-implemented, exit 2, stderr names the verb
			name:             "detach_not_implemented",
			subVerb:          "detach",
			wantExitCode:     2,
			wantStderrSubstr: "detach",
		},
		{
			// status → dispatches to sessions.status RPC (S-BL.CONSOLE-OBS).
			// With no daemon listening it exits 1 (E-NET-001), not 2.
			name:             "status_dispatches_to_daemon",
			subVerb:          "status",
			wantExitCode:     1,
			wantStderrSubstr: "E-NET-001",
		},
		{
			// bogus → unknown sub-verb, exit 2, stderr names "bogus"
			name:             "bogus_unknown_subverb",
			subVerb:          "bogus",
			wantExitCode:     2,
			wantStderrSubstr: "bogus",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			args := []string{
				"--target", target, "--key", keyPath,
				"sessions", tc.subVerb,
			}
			exitCode, stdout, stderr := runSessionsMain(t, args...)

			if exitCode != tc.wantExitCode {
				t.Errorf("F-P5P6-A-003: sessions %q: expected exit code %d, got %d\nstdout: %q\nstderr: %q",
					tc.subVerb, tc.wantExitCode, exitCode, stdout, stderr)
			}
			if tc.wantStderrSubstr != "" &&
				!strings.Contains(strings.ToLower(stderr), strings.ToLower(tc.wantStderrSubstr)) {
				t.Errorf("F-P5P6-A-003: sessions %q: expected stderr to contain %q; got: %q",
					tc.subVerb, tc.wantStderrSubstr, stderr)
			}
		})
	}

	// Bare `sbctl sessions` (no sub-verb) defaults to sessions.list.  With no
	// daemon listening it exits 1 (E-NET-001), not 2.  This guards the subVerb
	// default in runSessions() — if the default were removed or inverted the
	// test would see exit 2 (usage-error) instead of exit 1 (connection error).
	t.Run("bare_sessions_defaults_to_list", func(t *testing.T) {
		t.Parallel()

		args := []string{"--target", target, "--key", keyPath, "sessions"}
		exitCode, _, stderr := runSessionsMain(t, args...)
		// exit 1 means runSessions dispatched to sessions.list and got a network error
		// (no daemon at the dummy target).  exit 2 means it hit a usage-error branch
		// before the network call — the default-to-list fallback is broken.
		if exitCode != 1 {
			t.Errorf("F-P5P6-A-003: bare 'sbctl sessions' should default to sessions.list (exit 1 on E-NET-001), got exit %d", exitCode)
		}
		// OBS-P5P8-B-001: guard that the sessions.list network-error path emits
		// E-NET-001 on stderr.  If this fails the dispatch reached sessions.list
		// but the error-formatting path changed.
		if !strings.Contains(stderr, "E-NET-001") {
			t.Errorf("OBS-P5P8-B-001: bare 'sbctl sessions' expected stderr to contain E-NET-001; got: %q", stderr)
		}
	})
}
