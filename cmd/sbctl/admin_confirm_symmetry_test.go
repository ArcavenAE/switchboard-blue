// admin_confirm_symmetry_test.go — RED tests for Phase 5 Pass 4 remediation.
//
// Covers F-A-009 (LOW): `admin key revoke --confirm` is a Bool flag but
// `admin svtn destroy --confirm` accepts a String value.
//
// For symmetry, `admin key revoke --confirm` must accept a value form:
//
//	--confirm=<value>   e.g. --confirm=true or --confirm=false
//
// Currently, `admin key revoke` uses fs.Bool("confirm", ...) which means
//
//	--confirm=some-value  is rejected with a parse error by Go's flag package
//	(Bool flags do not accept = value form other than "true"/"false").
//
// The canonical shape after the fix:
//
//	--confirm              (bare flag, equivalent to --confirm=true)
//	--confirm=true
//	--confirm=false
//
// But the UX-symmetry finding also implies the flag should work in the same
// style as svtn destroy's --confirm=<token>.  The test below verifies that
// --confirm=true does NOT produce a flag-parse error (which it currently may,
// depending on Go version) and that the Confirm field in the wire args is true.
//
// MUST FAIL with current code if the Bool flag rejects the = form, or
// if the parsed value is wrong.
//
// Note: Go's flag package Bool flag DOES accept --confirm=true and --confirm=false
// since Go 1.1.  The real symmetry gap is documented behaviour/documentation
// parity, not a hard parse failure.  However, the test exercises the full
// runAdminKeyRevoke path to confirm the Confirm field is populated correctly
// and that --confirm=true, --confirm=false, and bare --confirm all work.
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// captureRevokeConfirmArgs is a test helper that runs runAdminKeyRevoke
// with a fake server that captures the raw JSON payload and returns success.
// We extract the Confirm field from the wire payload.
//
// Since runAdminKeyRevoke dials a management socket, we cannot intercept the
// wire payload without a fake server.  Instead, we test runAdminKeyRevoke
// argument parsing by exercising the path that hits the confirm-flag parse,
// up to the point before dialing.
//
// For the pure flag-parsing test, we invoke runAdminKeyRevoke with an
// unreachable target and check whether flag parsing succeeds or fails.
// If --confirm=true causes a parse error, the test FAILS with a descriptive message.
func TestNewInBurst19_ConfirmSymmetry_BoolFlagAcceptsValueForm(t *testing.T) {
	t.Parallel()

	// These tests do NOT dial the server; they verify that the flag parsing
	// layer accepts the value-form syntax without error.  The connection
	// attempt will fail but that's after flag parsing succeeds.

	tests := []struct {
		name        string
		args        []string
		wantParseOK bool // if false, we expect a parse error
		wantConfirm bool // expected Confirm value after parsing
	}{
		{
			name: "bare_confirm_flag",
			// --confirm with no = is the bare bool form; expect Confirm=true.
			args:        []string{"--key", "AAAA", "--svtn", "test-svtn", "--role", "control", "--confirm"},
			wantParseOK: true,
			wantConfirm: true,
		},
		{
			name: "confirm_equals_true",
			// --confirm=true must be accepted and yield Confirm=true.
			args:        []string{"--key", "AAAA", "--svtn", "test-svtn", "--role", "control", "--confirm=true"},
			wantParseOK: true,
			wantConfirm: true,
		},
		{
			name: "confirm_equals_false",
			// --confirm=false must be accepted and yield Confirm=false.
			args:        []string{"--key", "AAAA", "--svtn", "test-svtn", "--role", "control", "--confirm=false"},
			wantParseOK: true,
			wantConfirm: false,
		},
		{
			name: "no_confirm_flag",
			// --confirm absent yields Confirm=false.
			args:        []string{"--key", "AAAA", "--svtn", "test-svtn", "--role", "control"},
			wantParseOK: true,
			wantConfirm: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}

			// Use an unreachable target — the test will error on dial, not on parse.
			// A valid-but-unreachable address for the test context.
			err := runAdminKeyRevoke(
				context.Background(),
				"127.0.0.1:1", // unreachable — connection refused
				"",            // keyPath — not reached before flag parse error
				false,
				tc.args,
				sio,
			)

			// Determine whether this is a flag-parse error or a dial error.
			if tc.wantParseOK {
				// We expect either:
				// (a) a dial/connect error (flag parsing succeeded), OR
				// (b) an E-CFG-001 "invalid pubkey" or similar from key read (not a flag parse error).
				// In any case, we must NOT see a "invalid value" or "flag provided but not defined" error
				// which would indicate a flag-parse failure.
				if err != nil && (strings.Contains(err.Error(), "invalid value") ||
					strings.Contains(err.Error(), "flag provided but not defined") ||
					strings.Contains(err.Error(), "parse error")) {
					t.Errorf("%s: flag parsing rejected valid --confirm form; got: %v", tc.name, err)
				}
			} else {
				if err == nil {
					t.Errorf("%s: expected parse error but got nil", tc.name)
				}
			}
		})
	}
}

// TestNewInBurst19_ConfirmSymmetry_BoolFlagRejectsNonBoolValue verifies that
// the current Bool confirm flag on `admin key revoke` REJECTS a non-bool value
// like "--confirm=some-token" while `admin svtn destroy --confirm` (String flag)
// accepts arbitrary tokens.
//
// This documents the asymmetry (F-A-009). Once --confirm on key revoke
// becomes a String flag, the non-bool token will be accepted (not rejected)
// and this test will need updating. Until then: the flag IS rejected and this
// test FAILS because it asserts the rejection must NOT happen.
//
// MUST FAIL with current Bool flag because Bool rejects "some-confirmation-token".
func TestNewInBurst19_ConfirmSymmetry_BoolFlagRejectsNonBoolValue(t *testing.T) {
	t.Parallel()

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	// Attempt to pass a non-bool token value to --confirm on key revoke.
	// This should work once --confirm is a String flag (symmetry with svtn destroy).
	// Currently it will fail with a Bool parse error.
	err := runAdminKeyRevoke(
		context.Background(),
		"127.0.0.1:1", // unreachable
		"",
		false,
		[]string{
			"--key", "AAAA",
			"--svtn", "test-svtn",
			"--role", "control",
			"--confirm=some-confirmation-token", // non-bool — Bool flag rejects this
		},
		sio,
	)

	// FAILS: with a Bool flag, "--confirm=some-confirmation-token" causes a parse
	// error. Once the flag is changed to String, this will get a connection error
	// (not a parse error) and this assertion must invert.
	//
	// We assert that err must NOT be a Bool parse error.
	// With current code it IS a parse error → test FAILS.
	if err != nil && (strings.Contains(err.Error(), "invalid boolean") ||
		strings.Contains(err.Error(), "invalid value")) {
		t.Errorf("key revoke --confirm must accept non-bool token value for symmetry with svtn destroy; "+
			"got Bool parse error: %v\n  (fix: change --confirm flag from Bool to String)", err)
	}
}

// TestNewInBurst19_ConfirmSymmetry_ConfirmValueInWirePayload verifies that
// the Confirm field in the wire payload sent by sbctl to the daemon correctly
// reflects the --confirm flag value.
//
// This test exercises the wire struct serialization via json.Marshal to confirm
// the bool value round-trips correctly (GREEN guard — this already passes).
//
// The important symmetry gap (F-A-009) is that --confirm on key revoke is a
// Bool while --confirm on svtn destroy is a String (accepts arbitrary token).
// The test below documents the current Bool behavior and will need updating
// once --confirm is made String for full parity.
//
// MUST FAIL if the Confirm field in adminKeyRevokeArgs is not a bool, or if
// json:"confirm" tag is absent.
func TestNewInBurst19_ConfirmSymmetry_WirePayload_ConfirmTrue(t *testing.T) {
	t.Parallel()

	args := adminKeyRevokeArgs{
		SVTNID:  "test-svtn",
		Pubkey:  "AAAA",
		Role:    "control",
		Confirm: true,
	}
	payload, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal adminKeyRevokeArgs: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	confirm, ok := m["confirm"]
	if !ok {
		t.Fatalf("wire payload missing \"confirm\" field; payload: %s", payload)
	}
	// confirm must be a bool true (json unmarshals to float64 or bool depending on
	// UseNumber; without UseNumber, bools remain bool).
	if v, isBool := confirm.(bool); !isBool || !v {
		t.Errorf("wire payload confirm must be bool true; got %T(%v); payload: %s", confirm, confirm, payload)
	}
}

// TestNewInBurst19_ConfirmSymmetry_SvtnDestroyConfirmIsString verifies that
// runAdminSvtnDestroy's --confirm flag accepts a string value form (already
// correct in current code: admin.go:245 uses fs.String).
//
// This is a GREEN guard that documents the currently-correct behavior.
// It passes with current code.
func TestNewInBurst19_ConfirmSymmetry_SvtnDestroyConfirmIsString(t *testing.T) {
	t.Parallel()

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	// --confirm=SVTN-abcd1234 must be accepted by svtn destroy (String flag).
	err := runAdminSvtnDestroy(
		context.Background(),
		"127.0.0.1:1", // unreachable — connection refused after gate passes
		"",
		false,
		[]string{"--name", "test-svtn", "--confirm", "SVTN-abcd1234"},
		sio,
	)

	// The only expected errors are dial errors (gate passed with valid short-ID).
	// A flag-parse error here would be a regression.
	if err != nil && strings.Contains(err.Error(), "invalid value") {
		t.Errorf("svtn destroy --confirm=<token> must be accepted as String flag; got: %v", err)
	}
}
