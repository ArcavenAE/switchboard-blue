// admin_handlers_emission_text_test.go — Green regression guards for Phase 5 Pass 4 remediation.
//
// Covers:
//   - F-A-004 (HIGH): E-ADM-018 has parenthetical suffix "(revoking control key from SVTN %q)"
//     — must be stripped.  Canonical: "E-ADM-018: control-to-control revocation requires
//     explicit confirmation: use --confirm to proceed: <wrapped sentinel>"
//   - F-A-007 (MED): E-ADM-013 missing "no key with" in message.
//     Canonical: "E-ADM-013: key not found: no key with fingerprint <fp> registered in SVTN <name>"
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

// TestNewInBurst19_EADM018_NoParenthetical verifies that E-ADM-018 does NOT
// include the parenthetical suffix "(revoking control key from SVTN %q)".
//
// Canonical emission:
//
//	"E-ADM-018: control-to-control revocation requires explicit confirmation: use --confirm to proceed: <wrapped sentinel>"
//
// Green regression guard for F-A-004 — parenthetical suffix stripped from E-ADM-018.
func TestNewInBurst19_EADM018_NoParenthetical(t *testing.T) {
	t.Parallel()

	svtnName := "test-svtn"
	err := mapAdminError(svtnmgmt.ErrControlRevocationRequiresConfirm, svtnName, nil, "control")
	if err == nil {
		t.Fatal("mapAdminError(ErrControlRevocationRequiresConfirm): expected non-nil error")
	}

	// Must have E-ADM-018 as the exact prefix of the error message (taxonomy v4.6).
	assertErrorPrefix(t, err, "E-ADM-018: ")

	msg := err.Error()

	// Must NOT contain the parenthetical.
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
// Green regression guard for F-A-004 — end-to-end parenthetical absence through handler.
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

	// Must have E-ADM-018 as the exact prefix of the error message (taxonomy v4.6).
	assertErrorPrefix(t, gotErr, "E-ADM-018: ")

	msg := gotErr.Error()
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
// Green regression guard for F-A-007 — "no key with" phrase present in E-ADM-013.
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

	// Must have E-ADM-013 as the exact prefix of the error message (taxonomy v4.6).
	assertErrorPrefix(t, mappedErr, "E-ADM-013: ")

	msg := mappedErr.Error()

	// Must contain "no key with".
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
// Green regression guard for F-A-007 — full canonical E-ADM-013 format.
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

			// Must have E-ADM-013 as the exact prefix (taxonomy v4.6).
			assertErrorPrefix(t, mappedErr, "E-ADM-013: ")

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
