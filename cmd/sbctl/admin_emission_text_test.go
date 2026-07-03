// admin_emission_text_test.go — Green regression guards for Phase 5 Pass 4 remediation.
//
// Covers F-A-003 / F-B-001 (HIGH): E-CFG-012 emission text mismatch.
//
// Canonical (taxonomy v4.6):
//
//	"E-CFG-012: --yes cannot be combined with --confirm; pick one"
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// assertErrorPrefix verifies that err is non-nil and that err.Error() has the
// given prefix.  If err is nil or the prefix does not match, a test failure is
// recorded via t.Errorf.
//
// Using HasPrefix rather than Contains ensures the error code appears at the
// start of the message — a code embedded mid-message would pass a Contains
// check but violate the canonical emission requirement (taxonomy v4.6).
func assertErrorPrefix(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected non-nil error with prefix %q; got nil", want)
		return
	}
	got := err.Error()
	if !strings.HasPrefix(got, want) {
		t.Errorf("error %q does not have prefix %q", got, want)
	}
}

// TestNewInBurst19_ECFG012_PickOne verifies that when --yes and --confirm are
// both supplied, the error message says "pick one" (canonical taxonomy v4.6).
//
// Green regression guard for F-A-003/F-B-001 — "pick one" phrase in E-CFG-012.
func TestNewInBurst19_ECFG012_PickOne(t *testing.T) {
	t.Parallel()

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	// Both --yes=true and --confirm="SVTN-abcd1234" supplied → E-CFG-012.
	err := runDestroyConfirmGate("admin svtn destroy", "SVTN-abcd1234", true, "--name", sio)
	if err == nil {
		t.Fatal("runDestroyConfirmGate(confirm+yes): expected E-CFG-012 error, got nil")
	}

	// Must have E-CFG-012 as the exact prefix of the error message (taxonomy v4.6).
	assertErrorPrefix(t, err, "E-CFG-012: ")

	msg := err.Error()

	// Must say "pick one".
	if !strings.Contains(msg, "pick one") {
		t.Errorf("E-CFG-012: must contain \"pick one\"; got: %q", msg)
	}

	// Must NOT say the stale phrase.
	if strings.Contains(msg, "provide one or the other") {
		t.Errorf("E-CFG-012: must NOT contain stale phrase \"provide one or the other\"; got: %q", msg)
	}
}

// TestNewInBurst19_ECFG012_PickOne_CanonicalExact verifies the exact canonical
// substring (excluding the %w suffix) is present.
//
// Green regression guard for F-A-003/F-B-001 — exact canonical E-CFG-012 substring.
func TestNewInBurst19_ECFG012_PickOne_CanonicalExact(t *testing.T) {
	t.Parallel()

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	err := runDestroyConfirmGate("admin svtn destroy", "SVTN-abcd1234", true, "--name", sio)
	if err == nil {
		t.Fatal("expected E-CFG-012 error, got nil")
	}

	// The canonical message must match exactly up to the trailing suffix.
	canonical := "E-CFG-012: --yes cannot be combined with --confirm; pick one"
	if !strings.Contains(err.Error(), canonical) {
		t.Errorf("E-CFG-012 canonical message: must contain %q; got: %q", canonical, err.Error())
	}
}

// TestNewInBurst19_ECFG013_NonInteractiveSession_CanonicalMessage verifies that
// runDestroyConfirmGate with a non-TTY seam and no --confirm value emits an error
// whose prefix is exactly "E-CFG-013: " (canonical taxonomy code for non-interactive
// session errors; Fix F-11A-4).
//
// Uses stdinIsTTY seam directly — no full CLI dispatch needed because the gate
// function is the unit under test.
//
// NOTE: NOT parallel — mutates package-level seam stdinIsTTY.
//
// Traces to Fix F-11A-4; ADR-004; interface-definitions.md v1.17 §129/§130.
func TestNewInBurst19_ECFG013_NonInteractiveSession_CanonicalMessage(t *testing.T) {
	// NOT parallel: mutates package-level seam stdinIsTTY.
	origIsTTY := stdinIsTTY
	stdinIsTTY = func() bool { return false }
	t.Cleanup(func() { stdinIsTTY = origIsTTY })

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	// confirmVal = "" (no --confirm), yes = false → Path 3 (non-TTY + no --confirm).
	err := runDestroyConfirmGate("admin svtn destroy", "", false, "--name", sio)

	// Must have E-CFG-013 as the exact prefix (canonical taxonomy v4.6).
	assertErrorPrefix(t, err, "E-CFG-013: ")

	// Must contain the full canonical body substring (taxonomy v4.6).
	const canonicalE013 = "E-CFG-013: non-interactive session: --confirm is required for scripted use; use --confirm=<svtn-short-id> or --yes"
	if !strings.Contains(err.Error(), canonicalE013) {
		t.Errorf("E-CFG-013 body mismatch:\n got: %q\nwant substring: %q", err.Error(), canonicalE013)
	}
}

// TestNewInBurst19_YesWarning_TargetFlag_Destroy verifies that the Path 4 --yes
// warning for admin svtn destroy mentions "--name" (the target flag for that
// subcommand), not "--svtn".
//
// Fix F-14A-1: runDestroyConfirmGate now accepts a targetFlag argument so each
// caller emits a contextually correct warning.  Destroy calls with "--name";
// key register calls with "--svtn".
//
// NOTE: NOT parallel — mutates package-level seam stdinIsTTY.
func TestNewInBurst19_YesWarning_TargetFlag_Destroy(t *testing.T) {
	// NOT parallel: mutates package-level seam stdinIsTTY.
	origIsTTY := stdinIsTTY
	stdinIsTTY = func() bool { return false }
	t.Cleanup(func() { stdinIsTTY = origIsTTY })

	var errBuf bytes.Buffer
	sio := sbctlIO{out: &bytes.Buffer{}, err: &errBuf}

	// Path 4: yes=true, confirmVal="" → emit warning, return nil.
	err := runDestroyConfirmGate("admin svtn destroy", "", true, "--name", sio)
	if err != nil {
		t.Fatalf("Path 4 (--yes, destroy): expected nil error; got %v", err)
	}

	warning := errBuf.String()
	if !strings.Contains(warning, "--name") {
		t.Errorf("Path 4 destroy warning must mention \"--name\"; got: %q", warning)
	}
	if strings.Contains(warning, "--svtn") {
		t.Errorf("Path 4 destroy warning must NOT mention \"--svtn\" (wrong target flag); got: %q", warning)
	}
}

// TestNewInBurst19_YesWarning_TargetFlag_KeyRegister verifies that the Path 4
// --yes warning for admin key register mentions "--svtn" (the target flag for
// that subcommand), not "--name".
//
// Fix F-14A-1: symmetric companion to TestNewInBurst19_YesWarning_TargetFlag_Destroy.
//
// NOTE: NOT parallel — mutates package-level seam stdinIsTTY.
func TestNewInBurst19_YesWarning_TargetFlag_KeyRegister(t *testing.T) {
	// NOT parallel: mutates package-level seam stdinIsTTY.
	origIsTTY := stdinIsTTY
	stdinIsTTY = func() bool { return false }
	t.Cleanup(func() { stdinIsTTY = origIsTTY })

	var errBuf bytes.Buffer
	sio := sbctlIO{out: &bytes.Buffer{}, err: &errBuf}

	// Path 4: yes=true, confirmVal="" → emit warning, return nil.
	err := runDestroyConfirmGate("admin key register", "", true, "--svtn", sio)
	if err != nil {
		t.Fatalf("Path 4 (--yes, key register): expected nil error; got %v", err)
	}

	warning := errBuf.String()
	if !strings.Contains(warning, "--svtn") {
		t.Errorf("Path 4 key register warning must mention \"--svtn\"; got: %q", warning)
	}
	if strings.Contains(warning, "--name") {
		t.Errorf("Path 4 key register warning must NOT mention \"--name\" (wrong target flag); got: %q", warning)
	}
}

// TestNewInBurst19_ECFG012_PickOne_ViaRunAdminSvtnDestroy exercises E-CFG-012
// through the full runAdminSvtnDestroy path (not just runDestroyConfirmGate).
// This ensures the canonical error text propagates through the CLI dispatch layer.
//
// Since runAdminSvtnDestroy dials out, we use the --name flag but rely on the
// confirm gate returning before any dial attempt (E-CFG-012 fires synchronously).
//
// Green regression guard for F-A-003/F-B-001 — E-CFG-012 propagation through dispatch.
func TestNewInBurst19_ECFG012_PickOne_ViaRunAdminSvtnDestroy(t *testing.T) {
	t.Parallel()

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	// Pass both --yes and --confirm to trigger the gate.
	// Also pass --name to pass the required-arg check.
	// target="" — runAdminSvtnDestroy won't dial because E-CFG-012 fires first.
	err := runAdminSvtnDestroy(
		context.Background(),
		"",    // target — not reached
		"",    // keyPath — not reached
		false, // useJSON
		[]string{"--name", "test-svtn", "--yes", "--confirm", "SVTN-abcd1234"},
		sio,
	)
	if err == nil {
		t.Fatal("runAdminSvtnDestroy(--yes + --confirm): expected E-CFG-012, got nil")
	}

	// Must have E-CFG-012 as the exact prefix (taxonomy v4.6).
	assertErrorPrefix(t, err, "E-CFG-012: ")

	msg := err.Error()
	if !strings.Contains(msg, "pick one") {
		t.Errorf("E-CFG-012 via runAdminSvtnDestroy: must contain \"pick one\"; got: %q", msg)
	}
	if strings.Contains(msg, "provide one or the other") {
		t.Errorf("E-CFG-012 via runAdminSvtnDestroy: stale phrase \"provide one or the other\" present; got: %q", msg)
	}
}
