// single_print_test.go — issue #89 red gate.
//
// Every writeError call site currently double-prints on stderr: the call site
// emits the taxonomy envelope (--json) or single taxonomy line (plain) via
// writeError, then returns the error to main(), whose final handler prints it
// again with fmt.Fprintf(os.Stderr, "%s\n", err). In --json mode this means
// the whole-stderr stream is not a single JSON document (envelope line +
// plain-text line), breaking spec-runner's `jq -e` predicate for S-6.05
// json-envelope-integrity + BC-2.06.003 AC-006 (S-5.02).
//
// This file asserts the single-print contract at the subprocess re-exec level
// — the only test level that observes main()'s final printer, since unit
// tests through injected sio never see os.Stderr direct writes.
//
// Assertion shape (per case):
//
//	--json usage:   exit 2 · stdout empty · whole stderr is one JSON envelope
//	                with ok=false and error.code == "E-CFG-010"
//	--json network: exit 1 · stdout empty · whole stderr is one JSON envelope
//	                with ok=false and error.code == "E-NET-001"
//	plain usage:    exit 2 · stdout empty · stderr is exactly one line
//	                containing "E-CFG-010"
//	plain network:  exit 1 · stdout empty · stderr is exactly one line
//	                containing "E-NET-001"
//
// Spec authority: BC-2.06.003 AC-006 (S-5.02), S-6.05 json-envelope-integrity.
// Issue: ArcavenAE/switchboard-blue#89.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestProductionMain_ErrorPath_SinglePrint verifies the single-print contract
// on every error path: exactly one envelope on stderr in --json mode, exactly
// one taxonomy line in plain mode. RED against current main.go:120 which
// unconditionally re-prints every non-nil error.
//
// Cases use two error taxonomies known to fire before daemon contact:
//   - E-CFG-010 via `router status --target` (missing value) — usage error,
//     exits 2. writeError site is router_status.go:128.
//   - E-NET-001 via `paths list --target=127.0.0.1:<unreachable>` —
//     operational error, exits 1. writeError site is client.go:366.
//
// Both paths run daemon-free: --target requires a value fires before the
// dial branch; paths list fails dial against a non-listening port.
func TestProductionMain_ErrorPath_SinglePrint(t *testing.T) {
	t.Parallel()

	keyPath := testdataKeyPath(t)
	// Non-listening address: dial fails immediately with connection refused.
	const unreachable = "127.0.0.1:19982"

	cases := []struct {
		name     string
		args     []string
		wantExit int
		wantJSON bool   // if true, whole stderr must parse as a JSON envelope
		wantCode string // taxonomy code that must appear in the envelope / plain line
	}{
		{
			name: "json_usage_E-CFG-010",
			args: []string{
				"--json", "--target", unreachable, "--key", keyPath,
				"router", "status", "--target",
			},
			wantExit: 2,
			wantJSON: true,
			wantCode: "E-CFG-010",
		},
		{
			name: "json_network_E-NET-001",
			args: []string{
				"--json", "--target", unreachable, "--key", keyPath,
				"paths", "list",
			},
			wantExit: 1,
			wantJSON: true,
			wantCode: "E-NET-001",
		},
		{
			name: "plain_usage_E-CFG-010",
			args: []string{
				"--target", unreachable, "--key", keyPath,
				"router", "status", "--target",
			},
			wantExit: 2,
			wantJSON: false,
			wantCode: "E-CFG-010",
		},
		{
			name: "plain_network_E-NET-001",
			args: []string{
				"--target", unreachable, "--key", keyPath,
				"paths", "list",
			},
			wantExit: 1,
			wantJSON: false,
			wantCode: "E-NET-001",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exitCode, stdout, stderr := runProductionMain(t, tc.args...)

			if exitCode != tc.wantExit {
				t.Errorf("#89: exit=%d want=%d\nstdout: %q\nstderr: %q",
					exitCode, tc.wantExit, stdout, stderr)
			}

			// Success streams never leak to stdout on the error path
			// (BC-2.07.003 PC-3 already asserted elsewhere; kept here as a
			// nearby guard so a regression can't hide the single-print
			// break behind a stdout leak).
			if stdout != "" {
				t.Errorf("#89: expected empty stdout on error; got: %q", stdout)
			}

			if tc.wantJSON {
				// Whole-stderr stream must parse as one JSON document.
				// The current double-print emits envelope line + plain
				// text line — json.Unmarshal rejects trailing non-whitespace
				// after the first value with "invalid character 'E' after
				// top-level value" or similar. That's the S-6.05 break.
				var env struct {
					OK    bool `json:"ok"`
					Error struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"error"`
					Data json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal([]byte(stderr), &env); err != nil {
					t.Errorf("#89 S-6.05 json-envelope-integrity: whole stderr must parse as one JSON envelope; got parse error %v\nstderr: %q",
						err, stderr)
					return
				}
				if env.OK {
					t.Errorf("#89: envelope.ok must be false on error; got true\nstderr: %q", stderr)
				}
				if env.Error.Code != tc.wantCode {
					t.Errorf("#89: envelope.error.code=%q want=%q\nstderr: %q",
						env.Error.Code, tc.wantCode, stderr)
				}
			} else {
				// Plain mode: exactly one line on stderr.
				// One newline = one line under the writeError contract
				// (fmt.Fprintf(sio.err, "%s %s\n", code, message)).
				// Current bug: two newlines (writeError line + main's
				// re-print line).
				newlines := bytes.Count([]byte(stderr), []byte("\n"))
				if newlines != 1 {
					t.Errorf("#89: plain stderr must be exactly one line (one newline); got %d newlines\nstderr: %q",
						newlines, stderr)
				}
				if !strings.Contains(stderr, tc.wantCode) {
					t.Errorf("#89: stderr must contain taxonomy code %q; got: %q",
						tc.wantCode, stderr)
				}
			}
		})
	}
}
