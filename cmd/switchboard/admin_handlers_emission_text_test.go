// admin_handlers_emission_text_test.go — RED tests for Phase 5 Pass 4 remediation.
//
// Covers:
//   - F-A-004 (HIGH): E-ADM-018 has parenthetical suffix "(revoking control key from SVTN %q)"
//     — must be stripped.  Canonical: "E-ADM-018: control-to-control revocation requires
//     explicit confirmation: use --confirm to proceed: <wrapped sentinel>"
//   - F-A-007 (MED): E-ADM-013 missing "no key with" in message.
//     Canonical: "E-ADM-013: key not found: no key with fingerprint <fp> registered in SVTN <name>"
//
// These tests MUST FAIL until admin_handlers.go is updated.
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// TestNewInBurst19_EADM018_NoParenthetical verifies that E-ADM-018 does NOT
// include the parenthetical suffix "(revoking control key from SVTN %q)".
//
// Canonical emission:
//
//	"E-ADM-018: control-to-control revocation requires explicit confirmation: use --confirm to proceed: <wrapped sentinel>"
//
// MUST FAIL with current code at admin_handlers.go:413 which appends:
//
//	"(revoking control key from SVTN %q): %w"
func TestNewInBurst19_EADM018_NoParenthetical(t *testing.T) {
	t.Parallel()

	svtnName := "test-svtn"
	err := mapAdminError(svtnmgmt.ErrControlRevocationRequiresConfirm, svtnName, nil, "control")
	if err == nil {
		t.Fatal("mapAdminError(ErrControlRevocationRequiresConfirm): expected non-nil error")
	}

	msg := err.Error()

	// Must contain E-ADM-018 stamp.
	if !strings.Contains(msg, "E-ADM-018") {
		t.Errorf("E-ADM-018: stamp missing; got: %q", msg)
	}

	// Must NOT contain the parenthetical.
	// FAILS with current code which appends "(revoking control key from SVTN "test-svtn")".
	if strings.Contains(msg, "(revoking control key") {
		t.Errorf("E-ADM-018: must NOT contain parenthetical suffix \"(revoking control key...\"; got: %q", msg)
	}

	// Must NOT embed the SVTN name in the message body.
	// The canonical emission has no SVTN interpolation.
	if strings.Contains(msg, svtnName) {
		t.Errorf("E-ADM-018: must NOT embed SVTN name %q in message body; got: %q", svtnName, msg)
	}

	// The sentinel must be wrapped (errors.Is still works).
	if !errors.Is(err, svtnmgmt.ErrControlRevocationRequiresConfirm) {
		t.Errorf("E-ADM-018: errors.Is(err, ErrControlRevocationRequiresConfirm) must be true; got %v", err)
	}

	// Must contain the canonical body text.
	want := "control-to-control revocation requires explicit confirmation: use --confirm to proceed"
	if !strings.Contains(msg, want) {
		t.Errorf("E-ADM-018: expected body %q in error; got: %q", want, msg)
	}
}

// TestNewInBurst19_EADM018_NoParenthetical_ThroughHandler exercises E-ADM-018
// through the full revoke handler path to confirm the parenthetical is absent
// end-to-end (not just in mapAdminError in isolation).
//
// MUST FAIL with current code.
func TestNewInBurst19_EADM018_NoParenthetical_ThroughHandler(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var revokeFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.revoke" {
			revokeFn = h.Fn
			break
		}
	}
	if revokeFn == nil {
		t.Fatal("admin.key.revoke handler not registered")
	}

	// Register a second control key so we can attempt control-to-control revocation.
	targetPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate target key: %v", err)
	}
	if _, err := m.RegisterKey("test-svtn", targetPub, admission.RoleControl); err != nil {
		t.Fatalf("register second control key: %v", err)
	}

	// Encode the target key as base64 (current accepted format).
	targetEncoded := base64.StdEncoding.EncodeToString(targetPub)

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)

	// Revoke control key WITHOUT confirm=true — must get E-ADM-018.
	rpcArgs := adminKeyRevokeArgs{
		SVTNName:  "test-svtn",
		PublicKey: targetEncoded,
		Role:      "control",
		Confirm:   false,
	}
	rawArgs, err := json.Marshal(rpcArgs)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, gotErr := revokeFn(ctx, json.RawMessage(rawArgs))
	if gotErr == nil {
		t.Fatal("expected E-ADM-018 for control-to-control without confirm; got nil")
	}

	msg := gotErr.Error()
	if !strings.Contains(msg, "E-ADM-018") {
		t.Errorf("E-ADM-018 through handler: stamp missing; got: %q", msg)
	}
	// FAILS: current code includes the parenthetical.
	if strings.Contains(msg, "(revoking control key") {
		t.Errorf("E-ADM-018 through handler: must NOT contain parenthetical; got: %q", msg)
	}
}

// TestNewInBurst19_EADM013_NoKeyWith verifies that E-ADM-013 contains
// "no key with" in the message body.
//
// Canonical:
//
//	"E-ADM-013: key not found: no key with fingerprint <fp> registered in SVTN <name>"
//
// Current code at admin_handlers.go:602:
//
//	"E-ADM-013: key not found: fingerprint %s not registered in SVTN %s"
//	(missing "no key with")
//
// MUST FAIL with current code.
func TestNewInBurst19_EADM013_NoKeyWith(t *testing.T) {
	t.Parallel()

	targetPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	svtnName := "test-svtn"

	mappedErr := mapAdminError(admission.ErrKeyNotRegistered, svtnName, targetPub, "console")
	if mappedErr == nil {
		t.Fatal("mapAdminError(ErrKeyNotRegistered): expected non-nil error")
	}

	msg := mappedErr.Error()

	// Must contain E-ADM-013 stamp.
	if !strings.Contains(msg, "E-ADM-013") {
		t.Errorf("E-ADM-013: stamp missing; got: %q", msg)
	}

	// Must contain "no key with" — FAILS with current code.
	if !strings.Contains(msg, "no key with") {
		t.Errorf("E-ADM-013: must contain \"no key with\"; got: %q", msg)
	}

	// Must still contain "fingerprint" and "registered in SVTN".
	if !strings.Contains(msg, "fingerprint") {
		t.Errorf("E-ADM-013: expected \"fingerprint\" in error; got: %q", msg)
	}
	if !strings.Contains(msg, "registered in SVTN") {
		t.Errorf("E-ADM-013: expected \"registered in SVTN\" in error; got: %q", msg)
	}

	// Sentinel must still be wrapped.
	if !errors.Is(mappedErr, admission.ErrKeyNotRegistered) {
		t.Errorf("E-ADM-013: errors.Is(err, ErrKeyNotRegistered) must be true; got %v", mappedErr)
	}
}

// TestNewInBurst19_EADM013_CanonicalMessageFormat verifies the full canonical
// E-ADM-013 format including "no key with".
//
// MUST FAIL with current code.
func TestNewInBurst19_EADM013_CanonicalMessageFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		svtnName string
	}{
		{name: "simple_svtn", svtnName: "my-svtn"},
		{name: "prod_svtn", svtnName: "production"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pub, _, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("generate key: %v", err)
			}

			mappedErr := mapAdminError(admission.ErrKeyNotRegistered, tc.svtnName, pub, "console")
			if mappedErr == nil {
				t.Fatal("expected non-nil error")
			}

			msg := mappedErr.Error()

			// Full canonical check: "no key with fingerprint <fp> registered in SVTN <name>"
			// FAILS because current code says "fingerprint %s not registered" (missing "no key with").
			wantSubstring := "no key with fingerprint"
			if !strings.Contains(msg, wantSubstring) {
				t.Errorf("E-ADM-013 canonical format: must contain %q; got: %q", wantSubstring, msg)
			}

			wantSVTN := "registered in SVTN " + tc.svtnName
			if !strings.Contains(msg, wantSVTN) {
				t.Errorf("E-ADM-013 canonical format: must contain %q; got: %q", wantSVTN, msg)
			}
		})
	}
}
