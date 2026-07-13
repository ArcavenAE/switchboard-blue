// Package main — Phase 5 Pass 8 RED tests (A-findings) + B-finding fixes.
//
// RED tests (must FAIL against current code):
//   - F-P5P8-A-001 (HIGH): runDestroyConfirmGate hardcodes "admin svtn destroy:" in its
//     invalid-confirm error prefix, but is also called from runAdminKeyRegister.
//     `sbctl admin key register --confirm=bogus` emits "admin svtn destroy:" — wrong.
//   - F-P5P8-A-006 (MED): `sbctl paths ping` emits "usage: sbctl paths list" instead
//     of a router-style message naming the typed verb (exit 2 + stderr names "ping").
//
// B-finding fixes (GREEN — suite must still pass after these changes):
//   - F-P5P8-B-001 (MED): production_exit_code_test.go cases 7-12 share a single
//     hardcoded "F-P5P6-A-001" finding tag; cases 7-12 trace to F-P5P7-A-001/002/003.
//   - OBS-P5P8-B-001: bare_sessions_defaults_to_list should assert wantStderrContains
//     "E-NET-001" to guard that the default-to-list path emits the correct error code.
//
// Spec authority: interface-definitions.md v1.19 §105/§125/§127/§174.
// Findings: F-P5P8-A-001, F-P5P8-A-006, F-P5P8-B-001.
package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ── F-P5P8-A-001 (HIGH): confirm-gate error prefix on key register ─────────

// TestConfirmGatePrefix_KeyRegister_MustNotSayAdminSVTNDestroy verifies that
// `sbctl admin key register --confirm=<invalid>` emits an error prefixed with
// "admin key register:" — NOT "admin svtn destroy:".
//
// RED: runDestroyConfirmGate (admin.go:372-374) hardcodes "admin svtn destroy:"
// in its invalid-confirm error message.  When called from runAdminKeyRegister
// the wrong command name leaks into the operator-visible error.
//
// Two-sided test: the positive guard (destroy path still names itself correctly)
// is in TestConfirmGatePrefix_SVTNDestroy_StillCorrect below.
//
// Spec authority: interface-definitions.md v1.19 §105/§125.
// Finding: F-P5P8-A-001.
func TestConfirmGatePrefix_KeyRegister_MustNotSayAdminSVTNDestroy(t *testing.T) {
	// NOT parallel: mutates package-level seams stdinIsTTY / stdinReader.
	origIsTTY := stdinIsTTY
	origReader := stdinReader
	stdinIsTTY = func() bool { return false }
	stdinReader = strings.NewReader("")
	t.Cleanup(func() {
		stdinIsTTY = origIsTTY
		stdinReader = origReader
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Supply an invalid --confirm value (bad shape) so Path 1 of runDestroyConfirmGate
	// fires the invalid-confirm error.  The key and svtn flags are present so the error
	// cannot be attributed to missing required flags.
	err := runAdmin(ctx, "127.0.0.1:19995", testdataKeyPath(t), false, []string{
		"key", "register",
		"--svtn", "my-svtn",
		"--key", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI test-key",
		"--role", "console",
		"--confirm", "not-a-valid-svtn-id",
	}, defaultIO())

	if err == nil {
		t.Fatal("F-P5P8-A-001: expected error for invalid --confirm on key register; got nil")
	}

	errStr := err.Error()

	// PRIMARY assertion (RED): the error must NOT say "admin svtn destroy:"
	// when triggered from key register.
	if strings.Contains(errStr, "admin svtn destroy:") {
		t.Errorf("F-P5P8-A-001: key register confirm error leaks 'admin svtn destroy:' prefix; got: %q\n"+
			"want: prefix 'admin key register:'", errStr)
	}

	// SECONDARY assertion: the error MUST name the correct command context.
	// "admin key register" is the expected prefix per spec §105.
	if !strings.Contains(errStr, "admin key register") {
		t.Errorf("F-P5P8-A-001: key register confirm error must contain 'admin key register'; got: %q", errStr)
	}
}

// TestConfirmGatePrefix_SVTNDestroy_StillCorrect is the positive guard for
// F-P5P8-A-001: the svtn destroy path must still emit "admin svtn destroy:" in
// its invalid-confirm error after the fix is applied.
//
// GREEN: this must pass both before and after the fix.  If it regresses after
// the implementer's change, the fix over-corrected.
//
// Spec authority: interface-definitions.md v1.19 §127.
// Finding: F-P5P8-A-001 (positive guard).
func TestConfirmGatePrefix_SVTNDestroy_StillCorrect(t *testing.T) {
	// NOT parallel: mutates package-level seams stdinIsTTY / stdinReader.
	origIsTTY := stdinIsTTY
	origReader := stdinReader
	stdinIsTTY = func() bool { return false }
	stdinReader = strings.NewReader("")
	t.Cleanup(func() {
		stdinIsTTY = origIsTTY
		stdinReader = origReader
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runAdmin(ctx, "127.0.0.1:19996", testdataKeyPath(t), false, []string{
		"svtn", "destroy",
		"--name", "some-svtn",
		"--confirm", "not-a-valid-svtn-id",
	}, defaultIO())

	if err == nil {
		t.Fatal("F-P5P8-A-001 (guard): expected error for invalid --confirm on svtn destroy; got nil")
	}

	errStr := err.Error()

	// The destroy path must still say "admin svtn destroy:" after the fix.
	if !strings.Contains(errStr, "admin svtn destroy:") {
		t.Errorf("F-P5P8-A-001 (guard): svtn destroy confirm error must contain 'admin svtn destroy:'; got: %q", errStr)
	}
}

// ── F-P5P8-A-006 (MED): paths unknown-verb error names the typed verb ────────

// TestPathsUnknownVerb_ErrorNamesTypedVerb verifies that `sbctl paths ping`
// (an unknown sub-verb) exits with a usageError and the error message names the
// unknown verb "ping" — not just a generic "usage: sbctl paths list" template.
//
// RED: main.go:73-74 unconditionally emits "usage: sbctl paths list" for any
// paths sub-verb that is not "list".  The typed verb "ping" is not reflected in
// the error message.
//
// The test drives runMain via subprocess (runProductionMain) so that os.Exit is
// captured and we can assert exit 2 + stderr content.
//
// Spec authority: interface-definitions.md v1.19 §174 (exit-code table).
// Finding: F-P5P8-A-006.
func TestPathsUnknownVerb_ErrorNamesTypedVerb(t *testing.T) {
	t.Parallel()

	keyPath := testdataKeyPath(t)
	target := "127.0.0.1:19997"

	cases := []struct {
		name             string
		verb             string
		wantExitCode     int
		wantStderrSubstr string // typed verb must appear in stderr
	}{
		{
			// S-BL.CLI-SURFACE-COMPLETION AC-001 makes "ping" a real `paths`
			// sub-verb (BC-2.06.004) — it is no longer unknown, so this case
			// uses "trace", a verb that remains genuinely unrecognized.
			name:             "paths_unknown_verb_trace",
			verb:             "trace",
			wantExitCode:     2,
			wantStderrSubstr: "trace",
		},
		{
			name:             "paths_unknown_verb_status",
			verb:             "status",
			wantExitCode:     2,
			wantStderrSubstr: "status",
		},
		{
			name:             "paths_unknown_verb_foo",
			verb:             "foo",
			wantExitCode:     2,
			wantStderrSubstr: "foo",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exitCode, _, stderr := runProductionMain(t,
				"--target", target, "--key", keyPath,
				"paths", tc.verb,
			)

			if exitCode != tc.wantExitCode {
				t.Errorf("F-P5P8-A-006: paths %q: expected exit code %d, got %d\nstderr: %q",
					tc.verb, tc.wantExitCode, exitCode, stderr)
			}
			// PRIMARY RED assertion: the typed verb must appear in the error output.
			if !strings.Contains(stderr, tc.wantStderrSubstr) {
				t.Errorf("F-P5P8-A-006: paths %q: expected stderr to contain %q (the typed verb); got: %q",
					tc.verb, tc.wantStderrSubstr, stderr)
			}
		})
	}
}
