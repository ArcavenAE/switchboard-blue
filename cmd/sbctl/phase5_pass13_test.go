// Package main — Phase 5 Pass 13 RED test for F-P5P13-A-002 (MED).
//
// RED test (must FAIL against current code):
//
//   - F-P5P13-A-002 (MED): `sbctl admin list-keys` without the required --svtn flag
//     produces an error that is missing the E-CFG-001 token.
//     Spec §111 v1.24 anchor cmd/sbctl/admin.go:168 promises E-CFG-001 for missing
//     required flags on the admin surface.  The current implementation emits
//     "admin list-keys: --svtn <id> is required" with no code token.
//
//     Target behavior (adjudicated): KEEP exit 2 / usageError class (client-side
//     flag-value validation is a §174 usage error, same class as E-CFG-012/013
//     which live at exit 2), ADD the token — message becomes:
//     "E-CFG-001: admin list-keys: --svtn is required"
//
// Spec authority: interface-definitions.md v1.24 §111 (E-CFG-001 missing-svtn).
// Finding: F-P5P13-A-002 (MED).
package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestListKeysMissingSvtnFlag_CarriesECFG001Token verifies that
// `sbctl admin list-keys` without the required --svtn flag produces an error
// whose message carries the "E-CFG-001" token and classifies as a usageError
// (exit-2 class per interface-definitions.md §174).
//
// This mirrors the shape of TestKeyExpire_ZeroNegativeDuration_CarriesECFG001Token
// (phase5_pass10_test.go) which established the pattern for adding E-CFG-001
// tokens to existing usageErrf calls on the admin-key surface.
//
// Current behavior (admin.go:168):
//
//	return usageErrf("admin list-keys: --svtn <id> is required")
//
// Required behavior after fix:
//
//	return usageErrf("E-CFG-001: admin list-keys: --svtn is required")
//
// RED (F-P5P13-A-002): the current message has no "E-CFG-001" token.
// This test MUST FAIL at develop tip.
//
// Spec authority: interface-definitions.md v1.24 §111.
// Finding: F-P5P13-A-002 (MED).
func TestListKeysMissingSvtnFlag_CarriesECFG001Token(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 127.0.0.1:1 is unreachable — any bypass of client-side validation
	// produces a network error, caught by the guard assertion below.
	err := runAdmin(ctx, "127.0.0.1:1", testdataKeyPath(t), false, []string{
		"list-keys",
		// --svtn is intentionally omitted.
	}, defaultIO())

	if err == nil {
		t.Fatal("F-P5P13-A-002: expected non-nil error for missing --svtn flag; got nil")
	}

	errStr := err.Error()

	// Guard: if a network error fires, the client-side validation did not run.
	if strings.Contains(errStr, "E-NET-001") || strings.Contains(errStr, "connection refused") {
		t.Fatalf("F-P5P13-A-002: got network error %q — "+
			"client-side --svtn validation must reject before any connection attempt",
			errStr)
	}

	// PRIMARY RED assertion: the error must carry the E-CFG-001 token.
	// This is the assertion that is RED against develop tip today.
	if !strings.Contains(errStr, "E-CFG-001") {
		t.Errorf("F-P5P13-A-002: error %q missing E-CFG-001 token; "+
			"spec §111 requires E-CFG-001 for missing required flags on the admin surface",
			errStr)
	}

	// SECONDARY assertion: the error must mention --svtn so the operator
	// knows which flag is missing.
	if !strings.Contains(errStr, "--svtn") {
		t.Errorf("F-P5P13-A-002: error %q must mention '--svtn'; "+
			"want field-specific message naming the missing flag",
			errStr)
	}

	// TERTIARY assertion: the error must be a usageError (exit-2 class).
	// §174 classifies missing-required-flag validation as a usage error (exit 2).
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Errorf("F-P5P13-A-002: error %q must be a usageError (exit-2 class); "+
			"got error of type %T — the E-CFG-001 token must be added to the existing "+
			"usageErrf call at admin.go:168, not converted to a different error type",
			errStr, err)
	}
}
