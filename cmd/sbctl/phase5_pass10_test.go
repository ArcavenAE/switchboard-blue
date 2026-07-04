// Package main — Phase 5 Pass 10 RED test for expire E-CFG-001 token on zero/negative duration.
//
// RED test (must FAIL against current code):
//
//   - F-P5P10-A-002 (MED): runAdminKeyExpire (admin.go:554-556) returns a usageError
//     for zero/negative --after duration without the E-CFG-001 token prefix.
//     Spec §110 promises E-CFG-001 for zero/negative durations.  Every other
//     taxonomy emission that fires from client-side flag-value validation carries its
//     code token; this one does not.
//
//     Target behavior (adjudicated): KEEP exit 2 / usageError class (client-side
//     flag-value validation is a §174 usage error, same class as E-CFG-012/013
//     which live at exit 2), ADD the token — message becomes:
//     "E-CFG-001: admin key expire: --after duration must be positive, got %q"
//
// Spec authority: interface-definitions.md v1.21 §110 (E-CFG-001: zero/negative duration).
// Finding: F-P5P10-A-002.
package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestKeyExpire_ZeroNegativeDuration_CarriesECFG001Token is a table-driven
// RED test verifying that `--after 0s` and `--after -1h` each produce an error
// containing the "E-CFG-001" token.
//
// RED (F-P5P10-A-002): admin.go:555 emits
// "admin key expire: --after duration must be positive, got %q" — the E-CFG-001
// token is absent.  After the fix the message must lead with "E-CFG-001:".
//
// The test also verifies that the error is classified as a usageError (the
// interface-definitions.md §174 exit-2 class), because §110 places zero/negative
// duration validation in the same entry as E-CFG-001, and every other E-CFG-001
// emission on the admin-key surface returns usageError (exit 2).
//
// Direct-function oracle: uses 127.0.0.1:1 (unreachable) so any regression that
// skips the validation and dials produces E-NET-001, which the network-guard
// assertion catches.
//
// Spec authority: interface-definitions.md v1.21 §110.
// Finding: F-P5P10-A-002.
func TestKeyExpire_ZeroNegativeDuration_CarriesECFG001Token(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		afterFlag string
	}{
		{
			name:      "zero_duration_0s",
			afterFlag: "0s",
		},
		{
			name:      "negative_duration_minus_1h",
			afterFlag: "-1h",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// 127.0.0.1:1 is unreachable — any bypass of client-side validation
			// produces a network error, caught by the guard assertion below.
			err := runAdmin(ctx, "127.0.0.1:1", testdataKeyPath(t), false, []string{
				"key", "expire",
				"--key", "ssh-ed25519 AAAA...",
				"--svtn", "test-svtn",
				"--after", tc.afterFlag,
			}, defaultIO())

			if err == nil {
				t.Fatalf("F-P5P10-A-002 [%s]: expected non-nil error; got nil", tc.name)
			}

			errStr := err.Error()

			// Guard: network errors prove the client-side check did not fire.
			if strings.Contains(errStr, "E-NET-001") || strings.Contains(errStr, "connection refused") {
				t.Fatalf("F-P5P10-A-002 [%s]: got network error %q — "+
					"client-side validation must reject before any connection attempt",
					tc.name, errStr)
			}

			// PRIMARY RED assertion: the error must carry the E-CFG-001 token.
			// This is the assertion that is RED against the current code.
			if !strings.Contains(errStr, "E-CFG-001") {
				t.Errorf("F-P5P10-A-002 [%s]: error %q missing E-CFG-001 token; "+
					"want 'E-CFG-001: admin key expire: --after duration must be positive, got %q'",
					tc.name, errStr, tc.afterFlag)
			}

			// SECONDARY assertion: the error must still mention the flag so the
			// operator knows which flag triggered the validation.
			if !strings.Contains(errStr, "--after") && !strings.Contains(errStr, "duration") {
				t.Errorf("F-P5P10-A-002 [%s]: error %q must mention '--after' or 'duration'; "+
					"want field-specific message", tc.name, errStr)
			}

			// TERTIARY assertion: the error must be a usageError (exit-2 class).
			// §174 classifies flag-value validation as a usage error; zero/negative
			// duration is client-side flag-value validation.
			var ue *usageError
			if !errors.As(err, &ue) {
				t.Errorf("F-P5P10-A-002 [%s]: error %q must be a usageError (exit-2 class); "+
					"got error of type %T — the E-CFG-001 token must be added to the existing "+
					"usageErrf call, not converted to a different error type",
					tc.name, errStr, err)
			}
		})
	}
}
