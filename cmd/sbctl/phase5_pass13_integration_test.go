//go:build integration

// Package main — Phase 5 Pass 13 RED test for F-P5P13-B-001 (LOW).
//
// RED test (must FAIL against current code):
//
//   - F-P5P13-B-001 (LOW): the e2e stub command name for the "control" mode is
//     "admin.key.list" (e2e_helpers_test.go:191 + e2e_test.go:314) but the production
//     BuildAdminHandlers handler registers "admin.key.list-keys"
//     (admin_handlers.go:131).  This string divergence masks integration failures:
//     any e2e test exercising the control-mode command dispatches "admin.key.list"
//     which the production handler will never match.
//
//     The guard test asserts modeSpecificCommand("control") == "admin.key.list-keys".
//
// This test uses the integration build tag because modeSpecificCommand is defined
// in e2e_test.go which is also gated behind //go:build integration.
//
// Spec authority: admin_handlers.go:131 (production handler name per BuildAdminHandlers).
// Finding: F-P5P13-B-001 (LOW).
package main

import (
	"testing"
)

// TestControlModeCommand_MatchesProductionHandlerName is the discriminating oracle
// for F-P5P13-B-001: the e2e stub command name for "control" mode must equal the
// production handler name registered by BuildAdminHandlers.
//
// Current state (develop tip):
//   - modeSpecificCommand("control") returns "admin.key.list"
//   - production BuildAdminHandlers registers "admin.key.list-keys" (admin_handlers.go:131)
//
// The two-character suffix divergence ("-keys" absent in the stub) means every
// e2e test that calls modeSpecificCommand("control") dispatches a command the
// production handler will never match.  The test must fail now and pass once the
// stub is corrected to "admin.key.list-keys".
//
// RED (F-P5P13-B-001): modeSpecificCommand("control") currently returns
// "admin.key.list" != "admin.key.list-keys".
// This test MUST FAIL at develop tip.
//
// Spec authority: admin_handlers.go:131 (production handler name).
// Finding: F-P5P13-B-001 (LOW).
func TestControlModeCommand_MatchesProductionHandlerName(t *testing.T) {
	t.Parallel()

	const wantCommand = "admin.key.list-keys"
	got := modeSpecificCommand("control")

	// RED: MUST FAIL at develop tip — stub says "admin.key.list".
	if got != wantCommand {
		t.Errorf("F-P5P13-B-001: modeSpecificCommand(\"control\") = %q; "+
			"want %q (production handler name per admin_handlers.go:131); "+
			"e2e tests exercising control mode dispatch nothing that the production "+
			"handler will match — the stub name diverges from production by %q",
			got, wantCommand, got)
	}
}
