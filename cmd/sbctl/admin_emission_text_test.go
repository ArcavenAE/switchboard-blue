// admin_emission_text_test.go — RED tests for Phase 5 Pass 4 remediation.
//
// Covers F-A-003 / F-B-001 (HIGH): E-CFG-012 emission text mismatch.
//
// Canonical (taxonomy v4.4):
//
//	"E-CFG-012: --yes cannot be combined with --confirm; pick one"
//
// Current code at admin.go:306:
//
//	"E-CFG-012: --yes cannot be combined with --confirm; provide one or the other"
//
// These tests MUST FAIL until admin.go:306 is updated to say "pick one".
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestNewInBurst19_ECFG012_PickOne verifies that when --yes and --confirm are
// both supplied, the error message says "pick one" (canonical taxonomy v4.4).
//
// MUST FAIL with current code which says "provide one or the other".
func TestNewInBurst19_ECFG012_PickOne(t *testing.T) {
	t.Parallel()

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	// Both --yes=true and --confirm="SVTN-abcd1234" supplied → E-CFG-012.
	err := runDestroyConfirmGate("SVTN-abcd1234", true, sio)
	if err == nil {
		t.Fatal("runDestroyConfirmGate(confirm+yes): expected E-CFG-012 error, got nil")
	}

	msg := err.Error()

	// Must contain E-CFG-012 stamp.
	if !strings.Contains(msg, "E-CFG-012") {
		t.Errorf("E-CFG-012: stamp missing; got: %q", msg)
	}

	// Must say "pick one" — FAILS with current "provide one or the other".
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
// MUST FAIL with current code.
func TestNewInBurst19_ECFG012_PickOne_CanonicalExact(t *testing.T) {
	t.Parallel()

	sio := sbctlIO{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	err := runDestroyConfirmGate("SVTN-abcd1234", true, sio)
	if err == nil {
		t.Fatal("expected E-CFG-012 error, got nil")
	}

	// The canonical message must match exactly up to the trailing suffix.
	canonical := "E-CFG-012: --yes cannot be combined with --confirm; pick one"
	if !strings.Contains(err.Error(), canonical) {
		t.Errorf("E-CFG-012 canonical message: must contain %q; got: %q", canonical, err.Error())
	}
}

// TestNewInBurst19_ECFG012_PickOne_ViaRunAdminSvtnDestroy exercises E-CFG-012
// through the full runAdminSvtnDestroy path (not just runDestroyConfirmGate).
// This ensures the canonical error text propagates through the CLI dispatch layer.
//
// Since runAdminSvtnDestroy dials out, we use the --name flag but rely on the
// confirm gate returning before any dial attempt (E-CFG-012 fires synchronously).
//
// MUST FAIL with current code.
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

	msg := err.Error()
	if !strings.Contains(msg, "E-CFG-012") {
		t.Errorf("E-CFG-012 via runAdminSvtnDestroy: stamp missing; got: %q", msg)
	}
	if !strings.Contains(msg, "pick one") {
		t.Errorf("E-CFG-012 via runAdminSvtnDestroy: must contain \"pick one\"; got: %q", msg)
	}
	if strings.Contains(msg, "provide one or the other") {
		t.Errorf("E-CFG-012 via runAdminSvtnDestroy: stale phrase \"provide one or the other\" present; got: %q", msg)
	}
}
