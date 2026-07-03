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
// exits 2 — not 1 — for the six usage-error cases identified in the
// Phase 5 Pass 6 adversarial review finding F-P5P6-A-001.
//
// RED: current main.go:82-84 collapses ALL errors from the dispatch switch
// to os.Exit(1), so every case below will observe exit code 1 instead of 2.
//
// Spec authority: interface-definitions.md v1.18 §133, §174.
// Finding: F-P5P6-A-001.
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
		},
		{
			// Case 6: unknown top-level subcommand.
			// main.go default arm already calls os.Exit(2) — this case
			// verifies it stays correct when the error-propagation path is
			// refactored.  It is included so the full exit-2 contract is
			// visible in one test.
			name: "top_level_unknown_subcommand",
			args: []string{
				"--target", target, "--key", keyPath,
				"bogus",
			},
			wantExitCode:     2,
			wantStderrSubstr: "bogus",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exitCode, stdout, stderr := runProductionMain(t, tc.args...)

			if exitCode != tc.wantExitCode {
				t.Errorf("F-P5P6-A-001: expected exit code %d, got %d\nstdout: %q\nstderr: %q",
					tc.wantExitCode, exitCode, stdout, stderr)
			}
			if tc.wantStderrSubstr != "" && !strings.Contains(stderr, tc.wantStderrSubstr) {
				t.Errorf("F-P5P6-A-001: expected stderr to contain %q; got: %q",
					tc.wantStderrSubstr, stderr)
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
// NOTE: this is a NEW requirement. The existing TestSbctl_NoSubcommand_ExitsZero
// (main_test.go) was written against the old AC-012 spec that required exit 0.
// The Phase 5 Pass 6 adversarial review found that exit 0 violates §174.
// The implementer must reconcile TestSbctl_NoSubcommand_ExitsZero once the
// no-args behavior is corrected.
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

	// Usage output must enumerate available subcommands so the operator
	// knows what to type next.
	combined := stdout + stderr
	hasSubcmds := strings.Contains(combined, "sessions") ||
		strings.Contains(combined, "admin") ||
		strings.Contains(combined, "paths") ||
		strings.Contains(combined, "subcommand")
	if !hasSubcmds {
		t.Errorf("F-P5P6-A-006: expected usage output to enumerate subcommands; got stdout=%q stderr=%q",
			stdout, stderr)
	}
}

// ── F-P5P6-A-003 (MED): sessions sub-verb validation ─────────────────────────

// TestProductionMain_Sessions_SubVerbValidation verifies that `sbctl sessions`
// dispatches correctly based on the sub-verb:
//
//   - `sessions list`   → connects to daemon (may exit 1 on connection refused)
//   - `sessions attach` → exits 2 with a clear not-implemented error naming
//     the verb and citing backlog deferral
//   - `sessions status` → exits 2 (same shape as attach)
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
// Spec authority: interface-definitions.md v1.18 §71-73.
// Finding: F-P5P6-A-003.
func TestProductionMain_Sessions_SubVerbValidation(t *testing.T) {
	t.Parallel()

	keyPath := testdataKeyPath(t)
	// Nothing listens here — for sessions.list the exit will be 1 (E-NET-001),
	// for usage-error paths it should be 2 without touching the network.
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
			// status → not-implemented, exit 2, stderr names the verb
			name:             "status_not_implemented",
			subVerb:          "status",
			wantExitCode:     2,
			wantStderrSubstr: "status",
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
		exitCode, _, _ := runSessionsMain(t, args...)
		// exit 1 means runSessions dispatched to sessions.list and got a network error
		// (no daemon at the dummy target).  exit 2 means it hit a usage-error branch
		// before the network call — the default-to-list fallback is broken.
		if exitCode != 1 {
			t.Errorf("F-P5P6-A-003: bare 'sbctl sessions' should default to sessions.list (exit 1 on E-NET-001), got exit %d", exitCode)
		}
	})
}
