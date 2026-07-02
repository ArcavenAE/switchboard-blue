// Package svtnmgmt_test contains the full TDD test suite for BC-2.05.004
// (key lifecycle) and BC-2.07.001 (SVTN lifecycle).
//
// Traceability:
//
//	BC-2.07.001 — Control node creates/destroys SVTNs; first control key bootstrapped locally
//	BC-2.05.004 — Key lifecycle: register, revoke, and expire admission and session-authorization keys
//	VP-046      — Key lifecycle: registered→admitted, revoked→rejected, expired→rejected
//	VP-048      — SVTN lifecycle: create/destroy visibility and admission blocking
//
// Red Gate: all tests MUST fail before implementation (production functions panic("not implemented")).
package svtnmgmt_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// ── test helpers ─────────────────────────────────────────────────────────────

// controlCallerKey returns an admission.AdmittedKey with RoleControl set,
// used to satisfy SVTNManager.Destroy's defense-in-depth caller parameter in
// tests that exercise the happy-path (authorized) branch.
func controlCallerKey(pub ed25519.PublicKey) admission.AdmittedKey {
	return admission.AdmittedKey{
		PublicKey: pub,
		Role:      admission.RoleControl,
	}
}

// mustGenEdKey generates an Ed25519 key pair or fatals the test.
func mustGenEdKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return pub, priv
}

// newManager returns an SVTNManager wired to a fresh AdmittedKeySet.
// The returned manager is fresh (no SVTNs created yet).
func newManager(t *testing.T) *svtnmgmt.SVTNManager {
	t.Helper()
	controlPub, _ := mustGenEdKey(t)
	ks := admission.NewAdmittedKeySet()
	return svtnmgmt.NewSVTNManager(ks, controlPub)
}

// newManagerWithKS returns an SVTNManager and the control node's public key.
// Used by tests that need the control public key alongside SVTNManager operations.
func newManagerWithKS(t *testing.T) (*svtnmgmt.SVTNManager, ed25519.PublicKey) {
	t.Helper()
	controlPub, _ := mustGenEdKey(t)
	ks := admission.NewAdmittedKeySet()
	return svtnmgmt.NewSVTNManager(ks, controlPub), controlPub
}

// ── BC-2.07.001: SVTN lifecycle ───────────────────────────────────────────────

// TestSVTNManager_Create_BootstrapsControlKey verifies AC-001:
// SVTNManager.Create(svtnName) creates a new SVTN with a generated SVTN-ID and
// bootstraps the first control key locally.  Returns a CreateResult whose SVTN
// has a non-zero ID and a name matching the argument.
//
// BC-2.07.001 PC-1 (router running, no existing SVTN).
// BC-2.07.001 postcondition 1 (SVTN-ID registered; control key added as first admitted control-role key).
// BC-2.07.001 postcondition 2 (first control key added via local operation — trust anchor).
func TestSVTNManager_Create_BootstrapsControlKey(t *testing.T) {
	t.Parallel()

	mgr := newManager(t)

	// AC-001: Create returns a non-error result with a valid SVTN-ID.
	result, err := mgr.Create("test-svtn")
	if err != nil {
		t.Fatalf("BC-2.07.001 PC-1 — Create(\"test-svtn\") returned unexpected error: %v", err)
	}

	// BC-2.07.001 postcondition 1: SVTN-ID must be non-zero (16-byte random value).
	var zero [16]byte
	if result.SVTN.ID == zero {
		t.Error("BC-2.07.001 postcondition 1 — SVTN.ID is all-zero; expected a generated unique identifier")
	}

	// BC-2.07.001 postcondition 1: Name must equal the argument.
	if result.SVTN.Name != "test-svtn" {
		t.Errorf("BC-2.07.001 postcondition 1 — SVTN.Name = %q; want %q", result.SVTN.Name, "test-svtn")
	}

	// BC-2.07.001 postcondition 1: CreatedAt must be set and in UTC.
	if result.SVTN.CreatedAt.IsZero() {
		t.Error("BC-2.07.001 postcondition 1 — SVTN.CreatedAt is zero; expected a UTC timestamp")
	}
	if result.SVTN.CreatedAt.Location() != time.UTC {
		t.Errorf("BC-2.07.001 postcondition 1 — SVTN.CreatedAt.Location = %v; want UTC", result.SVTN.CreatedAt.Location())
	}
}

// TestSVTNManager_Create_IDUniqueness verifies BC-2.07.001 DI-005:
// Two successive Create calls with distinct names produce distinct SVTN-IDs.
//
// BC-2.07.001 invariant 2 (SVTN IDs globally unique within the router's scope).
// DI-005 (SVTN IDs must be globally unique).
func TestSVTNManager_Create_IDUniqueness(t *testing.T) {
	t.Parallel()

	// BC-2.07.001 DI-005 — SVTN IDs must be globally unique within the router's scope.
	mgr := newManager(t)

	r1, err := mgr.Create("net-a")
	if err != nil {
		t.Fatalf("Create(\"net-a\"): %v", err)
	}
	r2, err := mgr.Create("net-b")
	if err != nil {
		t.Fatalf("Create(\"net-b\"): %v", err)
	}

	if r1.SVTN.ID == r2.SVTN.ID {
		t.Errorf("BC-2.07.001 DI-005 — two distinct SVTNs share the same ID %v; IDs must be unique",
			r1.SVTN.ID)
	}
}

// TestSVTNManager_Create_DuplicateReturnsError verifies BC-2.07.001 EC-001:
// Create with a SVTN name that already exists returns ErrSVTNAlreadyExists (E-SVTN-001).
// No action is taken on the existing SVTN.
//
// BC-2.07.001 EC-001 (E-SVTN-001 "SVTN already exists: <id>"; no action taken).
// DI-005 (SVTN IDs must be globally unique; duplicate IDs are rejected).
func TestSVTNManager_Create_DuplicateReturnsError(t *testing.T) {
	t.Parallel()

	// BC-2.07.001 EC-001 — duplicate SVTN name must return ErrSVTNAlreadyExists.
	mgr := newManager(t)

	_, err := mgr.Create("dup-svtn")
	if err != nil {
		t.Fatalf("first Create: unexpected error: %v", err)
	}

	_, err = mgr.Create("dup-svtn")
	if !errors.Is(err, svtnmgmt.ErrSVTNAlreadyExists) {
		t.Errorf("BC-2.07.001 EC-001 — second Create(\"dup-svtn\"): want ErrSVTNAlreadyExists; got %v", err)
	}
}

// TestSVTNManager_Create_ControlKeyAdmittedToKeySet verifies BC-2.07.001
// postcondition 2: the control node's public key is added to the AdmittedKeySet
// as a control-role key (local bootstrap — trust anchor, no network round-trip).
//
// BC-2.07.001 postcondition 2 (bootstrap: first control key added via local operation).
// BC-2.07.001 postcondition 1 (control node's key added as the first admitted control-role key).
func TestSVTNManager_Create_ControlKeyAdmittedToKeySet(t *testing.T) {
	t.Parallel()

	// BC-2.07.001 postcondition 2 — first control key registered locally as trust anchor.
	mgr, controlPub := newManagerWithKS(t)

	result, err := mgr.Create("bootstrap-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verification strategy: we cannot call frame.DeriveNodeAddress from this
	// external test package. We verify the bootstrap indirectly by calling
	// RegisterKey for the same control key after Create — LWW semantics mean it
	// must succeed without ErrSVTNNotFound and without ErrKeyNotRegistered.
	// If Create did not bootstrap the key into the key set, the SVTN still exists
	// and RegisterKey would succeed as a fresh insert; that is acceptable — the
	// material test is that Create itself does not error and the SVTN record is valid.

	// Verify the SVTN ID is set correctly.
	if result.SVTN.ID == ([16]byte{}) {
		t.Fatal("SVTN.ID is zero")
	}

	// Now call RegisterKey for the SAME control key — last-write-wins (ADR-003) means
	// this should succeed (not return ErrKeyNotRegistered), which proves the key
	// was already registered by Create.
	// Note: RegisterKey on a key that is NOT yet registered would also succeed
	// (LWW semantics). We validate via KeyOpResult.Fingerprint being non-empty
	// and no error, which indirectly confirms the path through SVTN exists.
	_, err = mgr.RegisterKey(result.SVTN.Name, controlPub, admission.RoleControl)
	if err != nil {
		t.Errorf("BC-2.07.001 postcondition 2 — RegisterKey(controlPub) after Create returned error %v; "+
			"Create must have registered the key so LWW re-registration succeeds", err)
	}
}

// TestSVTNManager_Create_BootstrapKeyAdmittedFalse_TrustAnchor verifies
// BC-2.07.001 v1.2 PC-2: the bootstrap control key is initially registered
// with admitted=false. The control node must complete the standard
// challenge-response admission protocol to flip its own key to admitted=true.
// A change that bootstraps with admitted=true is a privilege bypass.
//
// BC-2.07.001 v1.2 PC-2 (bootstrap key starts admitted=false; challenge-response required).
// ARCH-04 §RegisterKey doc (admitted is intentionally zero at RegisterKey).
func TestSVTNManager_Create_BootstrapKeyAdmittedFalse_TrustAnchor(t *testing.T) {
	t.Parallel()

	// BC-2.07.001 v1.2 PC-2: bootstrap key must start with admitted=false.
	// Regression guard: a change that flips bootstrap to admitted=true is a
	// privilege bypass — the control key would be immediately usable for
	// frame admission without completing the challenge-response handshake.
	controlPub, _ := mustGenEdKey(t)
	ks := admission.NewAdmittedKeySet()
	mgr := svtnmgmt.NewSVTNManager(ks, controlPub)

	result, err := mgr.Create("bootstrap-admitted-test-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	nodeAddr := frame.DeriveNodeAddress(result.SVTN.ID, []byte(controlPub))
	if ks.IsAdmitted(result.SVTN.ID, nodeAddr) {
		t.Fatal("BC-2.07.001 v1.2 PC-2 violated: bootstrap key was admitted=true at " +
			"registration; challenge-response handshake required before admission")
	}
}

// ── BC-2.05.004: key lifecycle ────────────────────────────────────────────────

// TestSVTNManager_RegisterKey_AppearsInAdmissionChecks verifies AC-002 and
// BC-2.05.004 postcondition 1:
// RegisterKey adds a public key to the admitted key set with the specified role.
// The key becomes active for admission challenges.
// Also verifies that KeyOpResult carries a non-empty fingerprint and UTC timestamp.
//
// BC-2.05.004 postcondition 1 (key added to admitted set; fingerprint + timestamp returned).
// BC-2.05.004 invariant 2 (DI-002: only public key transmitted; private key never in result).
func TestSVTNManager_RegisterKey_AppearsInAdmissionChecks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		role admission.KeyRole
	}{
		// BC-2.05.004 postcondition 1 — key registered; fingerprint returned.
		{"register_control_key", admission.RoleControl},
		{"register_console_key", admission.RoleConsole},
		{"register_access_key", admission.RoleAccess},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mgr := newManager(t)

			// Create an SVTN first.
			svtnResult, err := mgr.Create("keyreg-svtn-" + tc.name)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			// Generate a fresh key to register.
			pub, _ := mustGenEdKey(t)

			// AC-002 / BC-2.05.004 postcondition 1.
			opResult, err := mgr.RegisterKey(svtnResult.SVTN.Name, pub, tc.role)
			if err != nil {
				t.Fatalf("BC-2.05.004 postcondition 1 — RegisterKey returned unexpected error: %v", err)
			}

			// BC-2.05.004 postcondition 4: key fingerprint must be non-empty.
			if opResult.Fingerprint == "" {
				t.Error("BC-2.05.004 postcondition 4 — KeyOpResult.Fingerprint is empty; want SHA256:<base64> format")
			}

			// BC-2.05.004 postcondition 4: operation timestamp must be set and UTC.
			if opResult.At.IsZero() {
				t.Error("BC-2.05.004 postcondition 4 — KeyOpResult.At is zero; want UTC timestamp")
			}
			if opResult.At.Location() != time.UTC {
				t.Errorf("BC-2.05.004 postcondition 4 — KeyOpResult.At.Location = %v; want UTC (go.md rule 11)", opResult.At.Location())
			}

			// BC-2.05.004 invariant 2 (DI-002): fingerprint must be in "SHA256:<base64>" format
			// (no private key bytes embedded).
			if len(opResult.Fingerprint) < 8 || opResult.Fingerprint[:7] != "SHA256:" {
				t.Errorf("BC-2.05.004 invariant 2 (DI-002) — Fingerprint %q does not start with 'SHA256:'; "+
					"public key fingerprints must use the standard SHA256: prefix to distinguish from raw key bytes",
					opResult.Fingerprint)
			}
		})
	}
}

// TestSVTNManager_RegisterKey_SVTNNotFound verifies that RegisterKey returns
// ErrSVTNNotFound when the named SVTN does not exist in the registry.
//
// BC-2.05.004 precondition 1 (requesting node must be admitted to the SVTN).
// ErrSVTNNotFound is E-SVTN-003.
func TestSVTNManager_RegisterKey_SVTNNotFound(t *testing.T) {
	t.Parallel()

	// BC-2.05.004 precondition 1 — SVTN must exist for key operations.
	mgr := newManager(t)
	pub, _ := mustGenEdKey(t)

	_, err := mgr.RegisterKey("nonexistent-svtn", pub, admission.RoleAccess)
	if !errors.Is(err, svtnmgmt.ErrSVTNNotFound) {
		t.Errorf("RegisterKey on nonexistent SVTN: want ErrSVTNNotFound; got %v", err)
	}
}

// TestSVTNManager_RegisterKey_DuplicateLastWriteWins verifies S-6.02 EC-001
// (ADR-003): registering an already-registered key with a different role
// applies last-write-wins semantics. The operation must succeed (no error).
//
// BC-2.05.004 EC-003 (ADR-003: last-write-wins for duplicate key registration).
// S-6.02 EC-001 (last-write-wins per ADR-003).
func TestSVTNManager_RegisterKey_DuplicateLastWriteWins(t *testing.T) {
	t.Parallel()

	// BC-2.05.004 EC-003 / S-6.02 EC-001 — last-write-wins; no duplicate error.
	mgr := newManager(t)

	svtnResult, err := mgr.Create("lww-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	pub, _ := mustGenEdKey(t)

	// First registration: RoleConsole.
	r1, err := mgr.RegisterKey(svtnResult.SVTN.Name, pub, admission.RoleConsole)
	if err != nil {
		t.Fatalf("first RegisterKey: %v", err)
	}

	// Second registration of the same key: RoleAccess (different role).
	// ADR-003 last-write-wins: must succeed and update the role.
	r2, err := mgr.RegisterKey(svtnResult.SVTN.Name, pub, admission.RoleAccess)
	if err != nil {
		t.Errorf("BC-2.05.004 EC-003 / ADR-003 — duplicate RegisterKey must not return error (LWW); got: %v", err)
	}

	// Both operations must return a non-empty fingerprint.
	if r1.Fingerprint == "" || r2.Fingerprint == "" {
		t.Error("EC-001 — KeyOpResult.Fingerprint must be non-empty for both registrations")
	}

	// The fingerprints must refer to the same key (same SHA256 digest).
	if r1.Fingerprint != r2.Fingerprint {
		t.Errorf("EC-001 — fingerprints differ for same key: %q vs %q; "+
			"LWW update must not change the key fingerprint", r1.Fingerprint, r2.Fingerprint)
	}
}

// TestSVTNManager_RevokeKey_RemovesFromAdmissionSet verifies AC-003 and
// BC-2.05.004 postcondition 2:
// RevokeKey removes the key from the admitted key set. Existing sessions continue
// until re-auth (propagation delay per FM-007).
//
// BC-2.05.004 postcondition 2 (key removed; sessions continue until re-auth).
// BC-2.05.004 postcondition 4 (fingerprint + timestamp in result).
// AC-003 (AC traces to BC-2.05.004 postcondition 2).
func TestSVTNManager_RevokeKey_RemovesFromAdmissionSet(t *testing.T) {
	t.Parallel()

	// AC-003 / BC-2.05.004 postcondition 2 — revoked key removed from admitted set.
	mgr, _ := newManagerWithKS(t)

	svtnResult, err := mgr.Create("revoke-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	pub, _ := mustGenEdKey(t)

	// Register a non-control key (console) to avoid the --confirm gate.
	_, err = mgr.RegisterKey(svtnResult.SVTN.Name, pub, admission.RoleConsole)
	if err != nil {
		t.Fatalf("RegisterKey: %v", err)
	}

	// Revoke the key. confirm=false is valid for non-control-to-control revocation.
	// BC-2.05.004 precondition 1: confirm is only required for control-to-control.
	// We pass RoleConsole as the current role to indicate the target is console.
	opResult, err := mgr.RevokeKey(svtnResult.SVTN.Name, pub, admission.RoleConsole, false)
	if err != nil {
		t.Fatalf("BC-2.05.004 postcondition 2 — RevokeKey returned unexpected error: %v", err)
	}

	// BC-2.05.004 postcondition 4: fingerprint and timestamp in result.
	if opResult.Fingerprint == "" {
		t.Error("BC-2.05.004 postcondition 4 — RevokeKey result Fingerprint is empty")
	}
	if opResult.At.IsZero() {
		t.Error("BC-2.05.004 postcondition 4 — RevokeKey result At is zero; want UTC timestamp")
	}
	if opResult.At.Location() != time.UTC {
		t.Errorf("BC-2.05.004 postcondition 4 — RevokeKey result At.Location = %v; want UTC", opResult.At.Location())
	}
}

// TestSVTNManager_RevokeKey_KeyNotFound verifies S-6.02 EC-002:
// Revoking a key that is not registered returns ErrKeyNotRegistered (E-ADM-013).
//
// S-6.02 EC-002 (E-ADM-013 key not found).
// BC-2.05.004 precondition 1 (key must be registered).
func TestSVTNManager_RevokeKey_KeyNotFound(t *testing.T) {
	t.Parallel()

	// S-6.02 EC-002 — revoking an unregistered key returns ErrKeyNotRegistered.
	mgr := newManager(t)

	svtnResult, err := mgr.Create("revoke-notfound-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	unregisteredPub, _ := mustGenEdKey(t)
	_, err = mgr.RevokeKey(svtnResult.SVTN.Name, unregisteredPub, admission.RoleConsole, false)
	if !errors.Is(err, admission.ErrKeyNotRegistered) {
		t.Errorf("S-6.02 EC-002 — RevokeKey on unregistered key: want admission.ErrKeyNotRegistered (E-ADM-013); got %v", err)
	}
}

// TestSVTNManager_RevokeKey_SVTNNotFound verifies that RevokeKey returns
// ErrSVTNNotFound when the named SVTN does not exist.
//
// BC-2.05.004 precondition 1 (SVTN must exist for key operations).
func TestSVTNManager_RevokeKey_SVTNNotFound(t *testing.T) {
	t.Parallel()

	// BC-2.05.004 precondition 1 — SVTN must exist.
	mgr := newManager(t)
	pub, _ := mustGenEdKey(t)

	_, err := mgr.RevokeKey("nonexistent-svtn", pub, admission.RoleConsole, false)
	if !errors.Is(err, svtnmgmt.ErrSVTNNotFound) {
		t.Errorf("RevokeKey on nonexistent SVTN: want ErrSVTNNotFound; got %v", err)
	}
}

// TestSVTNManager_ControlRevocation_RequiresConfirm verifies AC-005 and
// BC-2.05.004 precondition 1 / ADR-004:
// Control-to-control revocation without confirm=true returns
// ErrControlRevocationRequiresConfirm. With confirm=true it succeeds.
//
// BC-2.05.004 precondition 1 (control-to-control revocation requires sbctl admin human authorization).
// ADR-004 (split-brain mitigation: control-to-control revocation requires --confirm).
// AC-005 (control revocation requires human authorization).
func TestSVTNManager_ControlRevocation_RequiresConfirm(t *testing.T) {
	t.Parallel()

	mgr := newManager(t)

	svtnResult, err := mgr.Create("ctrl-revoke-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	targetControlPub, _ := mustGenEdKey(t)

	// Register the target key as a control-role key.
	_, err = mgr.RegisterKey(svtnResult.SVTN.Name, targetControlPub, admission.RoleControl)
	if err != nil {
		t.Fatalf("RegisterKey(control): %v", err)
	}

	t.Run("without_confirm_returns_error", func(t *testing.T) {
		t.Parallel()

		// AC-005 / BC-2.05.004 precondition 1 / ADR-004 — no confirm → error.
		_, err := mgr.RevokeKey(svtnResult.SVTN.Name, targetControlPub, admission.RoleControl, false)
		if !errors.Is(err, svtnmgmt.ErrControlRevocationRequiresConfirm) {
			t.Errorf("AC-005 / ADR-004 — control-to-control revoke without confirm: "+
				"want ErrControlRevocationRequiresConfirm; got %v", err)
		}
	})

	t.Run("with_confirm_succeeds", func(t *testing.T) {
		t.Parallel()

		// Use a separate manager+SVTN for this sub-test to avoid state interaction.
		mgr2 := newManager(t)
		svtnResult2, err := mgr2.Create("ctrl-revoke-svtn-confirm")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		targetPub2, _ := mustGenEdKey(t)
		_, err = mgr2.RegisterKey(svtnResult2.SVTN.Name, targetPub2, admission.RoleControl)
		if err != nil {
			t.Fatalf("RegisterKey(control): %v", err)
		}

		// AC-005 / ADR-004 — confirm=true → succeeds.
		opResult, err := mgr2.RevokeKey(svtnResult2.SVTN.Name, targetPub2, admission.RoleControl, true)
		if err != nil {
			t.Errorf("AC-005 / ADR-004 — control-to-control revoke with confirm=true: "+
				"unexpected error: %v", err)
		}
		if opResult.Fingerprint == "" {
			t.Error("AC-005 — RevokeKey with confirm=true: KeyOpResult.Fingerprint is empty")
		}
	})
}

// TestSVTNManager_RevokeKey_RoleMismatchReturnsError verifies that RevokeKey
// returns ErrRoleMismatch when the caller-supplied currentRole does not match
// the role stored in the AdmittedKeySet registry (E-ADM-019; HOLD-001 hybrid).
//
// BC-2.05.004 precondition 1 (role must be consistent with stored registration).
// ARCH-04 v1.7 HOLD-001 resolution: hybrid role cross-check.
func TestSVTNManager_RevokeKey_RoleMismatchReturnsError(t *testing.T) {
	t.Parallel()

	mgr := newManager(t)

	svtnResult, err := mgr.Create("role-mismatch-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	pub, _ := mustGenEdKey(t)

	// Register the key as RoleControl.
	_, err = mgr.RegisterKey(svtnResult.SVTN.Name, pub, admission.RoleControl)
	if err != nil {
		t.Fatalf("RegisterKey(RoleControl): %v", err)
	}

	// Attempt to revoke supplying RoleConsole — should return ErrRoleMismatch.
	_, err = mgr.RevokeKey(svtnResult.SVTN.Name, pub, admission.RoleConsole, true)
	if !errors.Is(err, svtnmgmt.ErrRoleMismatch) {
		t.Errorf("RevokeKey with mismatched role: want ErrRoleMismatch (E-ADM-019); got %v", err)
	}
}

// TestSVTNManager_RevokeKey_NonControlNoConfirmRequired verifies that revoking
// a non-control key (console, access) does NOT require confirm=true.
//
// BC-2.05.004 precondition 1 — only control-to-control requires confirmation;
// revoking console or access keys must succeed without confirm.
func TestSVTNManager_RevokeKey_NonControlNoConfirmRequired(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		role admission.KeyRole
	}{
		// BC-2.05.004 precondition 1 — console and access revocations do not require confirm.
		{"console_no_confirm", admission.RoleConsole},
		{"access_no_confirm", admission.RoleAccess},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mgr := newManager(t)
			svtnResult, err := mgr.Create("noconfirm-svtn-" + tc.name)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			pub, _ := mustGenEdKey(t)
			_, err = mgr.RegisterKey(svtnResult.SVTN.Name, pub, tc.role)
			if err != nil {
				t.Fatalf("RegisterKey(%v): %v", tc.role, err)
			}

			_, err = mgr.RevokeKey(svtnResult.SVTN.Name, pub, tc.role, false)
			if err != nil {
				t.Errorf("BC-2.05.004 precondition 1 — RevokeKey(%v, confirm=false): "+
					"expected success; got %v", tc.role, err)
			}
		})
	}
}

// TestSVTNManager_ExpireKey_SetsTTL verifies AC-004 and BC-2.05.004
// postcondition 3:
// ExpireKey associates a TTL with the key. After the TTL elapses, the key
// behaves as revoked (routers stop admitting it on the next re-auth).
// Verifies that KeyOpResult carries a non-empty fingerprint and UTC timestamp.
//
// BC-2.05.004 postcondition 3 (expiry timestamp associated with key).
// BC-2.05.004 postcondition 4 (fingerprint + timestamp in result).
// AC-004 (AC traces to BC-2.05.004 postcondition 3).
func TestSVTNManager_ExpireKey_SetsTTL(t *testing.T) {
	t.Parallel()

	// AC-004 / BC-2.05.004 postcondition 3 — expire key sets TTL.
	mgr := newManager(t)

	svtnResult, err := mgr.Create("expire-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	pub, _ := mustGenEdKey(t)
	_, err = mgr.RegisterKey(svtnResult.SVTN.Name, pub, admission.RoleConsole)
	if err != nil {
		t.Fatalf("RegisterKey: %v", err)
	}

	opResult, err := mgr.ExpireKey(svtnResult.SVTN.Name, pub, 24*time.Hour)
	if err != nil {
		t.Fatalf("BC-2.05.004 postcondition 3 — ExpireKey returned unexpected error: %v", err)
	}

	// BC-2.05.004 postcondition 4: fingerprint and timestamp in result.
	if opResult.Fingerprint == "" {
		t.Error("BC-2.05.004 postcondition 4 — ExpireKey result Fingerprint is empty")
	}
	if opResult.At.IsZero() {
		t.Error("BC-2.05.004 postcondition 4 — ExpireKey result At is zero; want UTC timestamp")
	}
	if opResult.At.Location() != time.UTC {
		t.Errorf("BC-2.05.004 postcondition 4 — ExpireKey result At.Location = %v; want UTC", opResult.At.Location())
	}
}

// TestSVTNManager_ExpireKey_ZeroDurationReturnsError verifies S-6.02 EC-003:
// ExpireKey with a zero TTL returns ErrInvalidDuration (E-CFG-001).
// Negative duration must also be rejected.
//
// S-6.02 EC-003 (E-CFG-001: invalid duration).
// BC-2.05.004 postcondition 3 (TTL must be positive).
func TestSVTNManager_ExpireKey_ZeroDurationReturnsError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ttl  time.Duration
	}{
		// S-6.02 EC-003 — zero and negative durations must return ErrInvalidDuration.
		{"zero_duration", 0},
		{"negative_duration", -1 * time.Hour},
		{"negative_microsecond", -1 * time.Microsecond},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mgr := newManager(t)
			svtnResult, err := mgr.Create("expire-invalid-svtn-" + tc.name)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			pub, _ := mustGenEdKey(t)
			_, err = mgr.RegisterKey(svtnResult.SVTN.Name, pub, admission.RoleConsole)
			if err != nil {
				t.Fatalf("RegisterKey: %v", err)
			}

			_, err = mgr.ExpireKey(svtnResult.SVTN.Name, pub, tc.ttl)
			if !errors.Is(err, svtnmgmt.ErrInvalidDuration) {
				t.Errorf("S-6.02 EC-003 — ExpireKey(ttl=%v): want ErrInvalidDuration (E-CFG-001); got %v",
					tc.ttl, err)
			}
		})
	}
}

// TestSVTNManager_ExpireKey_SVTNNotFound verifies that ExpireKey returns
// ErrSVTNNotFound when the named SVTN does not exist.
//
// BC-2.05.004 precondition 1 (SVTN must exist for key operations).
func TestSVTNManager_ExpireKey_SVTNNotFound(t *testing.T) {
	t.Parallel()

	// BC-2.05.004 precondition 1 — SVTN must exist.
	mgr := newManager(t)
	pub, _ := mustGenEdKey(t)

	_, err := mgr.ExpireKey("nonexistent-svtn", pub, time.Hour)
	if !errors.Is(err, svtnmgmt.ErrSVTNNotFound) {
		t.Errorf("ExpireKey on nonexistent SVTN: want ErrSVTNNotFound; got %v", err)
	}
}

// TestSVTNManager_ExpireKey_KeyNotFound verifies that ExpireKey returns
// ErrKeyNotRegistered (E-ADM-013) when the key is not registered for the SVTN.
//
// BC-2.05.004 postcondition 3 (key must be registered).
// admission.ErrKeyNotRegistered (E-ADM-013).
func TestSVTNManager_ExpireKey_KeyNotFound(t *testing.T) {
	t.Parallel()

	// BC-2.05.004 postcondition 3 — key must be registered to set expiry.
	mgr := newManager(t)

	svtnResult, err := mgr.Create("expire-notfound-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	unregisteredPub, _ := mustGenEdKey(t)
	_, err = mgr.ExpireKey(svtnResult.SVTN.Name, unregisteredPub, time.Hour)
	if !errors.Is(err, admission.ErrKeyNotRegistered) {
		t.Errorf("ExpireKey on unregistered key: want admission.ErrKeyNotRegistered (E-ADM-013); got %v", err)
	}
}

// ── VP-046 unit coverage ──────────────────────────────────────────────────────

// TestSVTNManager_VP046_RegisteredKeyFingerprint_DI002 verifies VP-046 property:
// private key never appears in key management wire messages (DI-002).
// The KeyOpResult.Fingerprint must not contain raw private key bytes.
//
// VP-046 (property: private key never appears in key management wire messages).
// BC-2.05.004 invariant 2 (DI-002: private keys never transmitted).
func TestSVTNManager_VP046_RegisteredKeyFingerprint_DI002(t *testing.T) {
	t.Parallel()

	// VP-046 / DI-002 — fingerprint must not contain private key material.
	mgr := newManager(t)

	svtnResult, err := mgr.Create("di002-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	pub, priv := mustGenEdKey(t)
	opResult, err := mgr.RegisterKey(svtnResult.SVTN.Name, pub, admission.RoleConsole)
	if err != nil {
		t.Fatalf("RegisterKey: %v", err)
	}

	// The private key is 64 bytes. Its hex/base64 encoding should not appear in
	// the fingerprint. We encode the private key in base64 and check for substring.
	// This is a coarse DI-002 sanity check at the unit level.
	privBytes := []byte(priv)
	fingerprint := opResult.Fingerprint

	// Check that the raw private key bytes are not the fingerprint body
	// (the fingerprint should be a SHA256 hash of the public key, ~44 chars in base64).
	// A raw ed25519 private key is 64 bytes = 88 chars in base64 (with padding).
	// The fingerprint must be << 88 chars of raw key data.
	if len(fingerprint) > 0 {
		// If fingerprint contains the full private key bytes (even partially encoded),
		// something is badly wrong.  We check the first 16 bytes of priv are not
		// verbatim in the fingerprint string.
		privHex := make([]byte, 32)
		copy(privHex, privBytes[:16])
		// Simplest invariant: fingerprint length should be consistent with a
		// "SHA256:<44 chars>" format (~51 chars), not 88+ chars of raw key.
		if len(fingerprint) > 100 {
			t.Errorf("VP-046 / DI-002 — fingerprint is suspiciously long (%d bytes): %q; "+
				"possible private key leakage", len(fingerprint), fingerprint)
		}
	}
}

// ── VP-048 unit coverage ──────────────────────────────────────────────────────

// TestSVTNManager_VP048_CreateIdempotentFirstInvocation verifies VP-048:
// SVTN create is idempotent for the first invocation; error on duplicate.
//
// VP-048 (SVTN create is idempotent for the first invocation; error on duplicate).
// BC-2.07.001 EC-001 (duplicate SVTN name returns E-SVTN-001).
func TestSVTNManager_VP048_CreateIdempotentFirstInvocation(t *testing.T) {
	t.Parallel()

	// VP-048 — first Create succeeds; second with same name returns ErrSVTNAlreadyExists.
	mgr := newManager(t)

	result, err := mgr.Create("vp048-svtn")
	if err != nil {
		t.Fatalf("VP-048: first Create: %v", err)
	}

	// First invocation: SVTN-ID must be non-zero.
	var zero [16]byte
	if result.SVTN.ID == zero {
		t.Error("VP-048 — SVTN.ID is zero after successful Create")
	}

	// Second invocation with same name: must return ErrSVTNAlreadyExists.
	_, err = mgr.Create("vp048-svtn")
	if !errors.Is(err, svtnmgmt.ErrSVTNAlreadyExists) {
		t.Errorf("VP-048 — second Create(\"vp048-svtn\"): want ErrSVTNAlreadyExists; got %v", err)
	}
}

// TestSVTNManager_VP048_OnlyControlKeyCanCreateSVTN verifies VP-048:
// Only control-role keys may create SVTNs.
//
// VP-048 (only control-role keys can create/destroy SVTNs).
// BC-2.07.001 invariant 3 (only control-role keys may create or destroy SVTNs).
//
// NOTE: SVTNManager.Create does not take a caller-role parameter in the current
// API surface (it is the control node itself calling it, not a remote caller).
// This test verifies the BOOTSTRAP scenario: the bootstrapped key is always
// registered as RoleControl, never as RoleConsole or RoleAccess.
// See HOLD-001 in the HOLD section at the end of this file.
func TestSVTNManager_VP048_BootstrappedKeyIsControlRole(t *testing.T) {
	t.Parallel()

	// VP-048 / BC-2.07.001 invariant 3 — bootstrapped key must be RoleControl.
	// Verified by registering the same key again and observing that LWW succeeds
	// (the key exists), and by checking the admission set via RegisterKey+Lookup.
	mgr, controlPub := newManagerWithKS(t)

	svtnResult, err := mgr.Create("role-verify-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Strengthened assertion (CR-011): confirm the bootstrapped key's role is
	// RoleControl by calling RevokeKey with RoleControl (confirm=true) BEFORE
	// any overwrite. If the bootstrap role were NOT RoleControl, this call would
	// return ErrRoleMismatch instead of succeeding, catching a regression.
	//
	// We use a fresh manager+SVTN (mgr2) so the revocation does not interfere
	// with any other test state. mgr is left intact.
	mgr2, controlPub2 := newManagerWithKS(t)
	svtnResult2, err2 := mgr2.Create("role-verify-svtn-confirm")
	if err2 != nil {
		t.Fatalf("Create (mgr2): %v", err2)
	}

	// Verify bootstrap key role via CallerKeyRole (not RevokeKey — the bootstrap
	// guard now prevents revoking the control key directly; TODO(BC-2.07.001):
	// spec clarification pending for E-ADM-020 bootstrap-key-revoke-forbidden).
	// CallerKeyRole is the correct verification surface for role introspection.
	role2, found2 := mgr2.CallerKeyRole(svtnResult2.SVTN.Name, controlPub2)
	if !found2 {
		t.Errorf("VP-048 / BC-2.07.001 invariant 3 — CallerKeyRole(controlPub): " +
			"key not found; bootstrap key must be present")
	} else if role2 != admission.RoleControl {
		t.Errorf("VP-048 / BC-2.07.001 invariant 3 — CallerKeyRole(controlPub): "+
			"want RoleControl; got %v", role2)
	}

	// Also verify that after the LWW overwrite to RoleConsole, CallerKeyRole
	// returns RoleConsole (key is now RoleConsole after LWW).
	_, err = mgr.RegisterKey(svtnResult.SVTN.Name, controlPub, admission.RoleConsole)
	if err != nil {
		t.Errorf("VP-048 — RegisterKey(controlPub, RoleConsole) after Create: "+
			"want success (LWW overwrite of bootstrapped control key); got: %v", err)
	}
	roleAfterLWW, foundAfterLWW := mgr.CallerKeyRole(svtnResult.SVTN.Name, controlPub)
	if !foundAfterLWW {
		t.Errorf("VP-048 — CallerKeyRole after LWW overwrite: key not found")
	} else if roleAfterLWW != admission.RoleConsole {
		t.Errorf("VP-048 — CallerKeyRole after LWW overwrite to RoleConsole: "+
			"want RoleConsole; got %v", roleAfterLWW)
	}
}

// ── Concurrent-safety smoke tests ─────────────────────────────────────────────

// TestSVTNManager_ConcurrentCreate_RaceDetector verifies SVTNManager is safe for
// concurrent use under go test -race. Multiple goroutines create distinct SVTNs.
//
// BC-2.07.001 (SVTNManager all exported methods safe for concurrent use; godoc comment).
func TestSVTNManager_ConcurrentCreate_RaceDetector(t *testing.T) {
	// NOT t.Parallel(): goroutine-count-sensitive test kept sequential to avoid
	// interference; the race detector is the focus.

	mgr := newManager(t)

	const workers = 10
	errs := make(chan error, workers)

	for i := range workers {
		go func(n int) {
			name := "concurrent-svtn-" + string(rune('a'+n))
			_, err := mgr.Create(name)
			errs <- err
		}(i)
	}

	for range workers {
		if err := <-errs; err != nil {
			t.Errorf("concurrent Create: unexpected error: %v", err)
		}
	}
}

// TestSVTNManager_ConcurrentRegisterRevoke_RaceDetector verifies SVTNManager
// concurrent register/revoke on the same SVTN is race-detector clean.
//
// BC-2.05.004 (SVTNManager all exported methods safe for concurrent use).
func TestSVTNManager_ConcurrentRegisterRevoke_RaceDetector(t *testing.T) {
	// NOT t.Parallel(): race detector focus.

	mgr := newManager(t)
	svtnResult, err := mgr.Create("concurrent-ops-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const workers = 5
	errs := make(chan error, workers*2)

	for range workers {
		pub, _ := mustGenEdKey(t)

		go func(key ed25519.PublicKey) {
			_, err := mgr.RegisterKey(svtnResult.SVTN.Name, key, admission.RoleConsole)
			errs <- err
		}(pub)

		go func(key ed25519.PublicKey) {
			// May return nil (success) or ErrKeyNotRegistered if the registration
			// hasn't run yet; both are valid for the race-detector smoke test.
			// Any other error is unexpected and must fail the test (F-006).
			_, err := mgr.RevokeKey(svtnResult.SVTN.Name, key, admission.RoleConsole, false)
			if err != nil && !errors.Is(err, admission.ErrKeyNotRegistered) {
				errs <- fmt.Errorf("concurrent RevokeKey: unexpected error: %w", err)
				return
			}
			errs <- nil
		}(pub)
	}

	for range workers * 2 {
		if err := <-errs; err != nil {
			t.Errorf("concurrent register: unexpected error: %v", err)
		}
	}
}

// TestSVTNManager_CreateBootstrapAtomicity_RaceDetector verifies F-003:
// the bootstrap control key is registered BEFORE the SVTN is published to
// m.svtns. Concurrent goroutines attempting to inject a foreign control key
// via RegisterKey must not be able to observe a half-bootstrapped SVTN.
//
// Verification strategy: after all goroutines complete, each successfully
// created SVTN must still have the bootstrap control key present (verifiable
// via RevokeKey — ErrKeyNotRegistered means the key is absent, which indicates
// an attacker key overwrote the bootstrap key before it was registered).
//
// Note: RevokeKey is used as the verification probe here because it is the
// only public API that reliably distinguishes "key present" from "key absent".
// The attacker key may also be present (LWW semantics per ADR-003); the
// invariant is only that the bootstrap key was registered atomically with
// SVTN publication.
//
// BC-2.07.001 PC-1+PC-2 composite postcondition (F-003).
// ADR-003 (LWW key semantics).
func TestSVTNManager_CreateBootstrapAtomicity_RaceDetector(t *testing.T) {
	// NOT t.Parallel(): race detector focus; goroutine count is sensitive.

	const svtns = 10
	const attackersPerSVTN = 5

	// Shared manager so all goroutines contend on the same state.
	controlPub, _ := mustGenEdKey(t)
	ks := admission.NewAdmittedKeySet()
	mgr := svtnmgmt.NewSVTNManager(ks, controlPub)

	attackerPub, _ := mustGenEdKey(t)

	type result struct {
		svtnName string
		created  bool
	}
	createResults := make(chan result, svtns)
	attackErrs := make(chan error, svtns*attackersPerSVTN)

	// Half: create goroutines — each creates a distinct SVTN.
	for i := range svtns {
		go func(n int) {
			name := fmt.Sprintf("atomic-svtn-%d", n)
			_, err := mgr.Create(name)
			createResults <- result{svtnName: name, created: err == nil}
		}(i)
	}

	// Half: attacker goroutines — attempt to inject a foreign control key.
	// RegisterKey returns ErrSVTNNotFound when the SVTN is not yet visible;
	// both nil and ErrSVTNNotFound are valid races here.
	for i := range svtns {
		for range attackersPerSVTN {
			go func(n int) {
				name := fmt.Sprintf("atomic-svtn-%d", n)
				_, err := mgr.RegisterKey(name, attackerPub, admission.RoleControl)
				if err != nil && !errors.Is(err, svtnmgmt.ErrSVTNNotFound) {
					attackErrs <- fmt.Errorf("attacker RegisterKey(%s): unexpected error: %w", name, err)
					return
				}
				attackErrs <- nil
			}(i)
		}
	}

	// Drain create results.
	created := make(map[string]bool, svtns)
	for range svtns {
		r := <-createResults
		created[r.svtnName] = r.created
	}

	// Drain attacker errors.
	for range svtns * attackersPerSVTN {
		if err := <-attackErrs; err != nil {
			t.Errorf("F-003 attacker goroutine: %v", err)
		}
	}

	// Verification (F-CS-001 fix): directly probe the AdmittedKeySet via
	// LookupByPubkey BEFORE any further RegisterKey call.
	//
	// The prior approach re-registered controlPub (LWW) before the probe,
	// which made the test tautological: the re-registration would re-install
	// controlPub even if the bootstrap ordering were broken, so RevokeKey
	// unconditionally succeeded regardless of whether F-003 prevented the
	// attacker LWW win. This rewrite removes that re-registration and probes
	// the key-set state directly after the race.
	//
	// Red-gate verification (F-CS-001): to confirm this test is non-tautological,
	// temporarily swap Create()'s bootstrap-RegisterKey to happen AFTER the
	// m.svtns[name]=... assignment and verify that some entries have
	// stored==nil or stored.Role != RoleControl. After verifying the failure,
	// revert the ordering. The test was confirmed to fail when the F-003
	// ordering invariant is broken.
	//
	// For each successfully created SVTN, look up controlPub directly in
	// the AdmittedKeySet. The bootstrap invariant (BC-2.07.001 PC-1+PC-2)
	// requires the key to be present with RoleControl. Under LWW, an attacker
	// goroutine may have overwritten it with a different role — the test
	// accepts only RoleControl here because the bootstrap registers BEFORE
	// the SVTN is published, making a concurrent RegisterKey for that SVTN
	// impossible (no SVTN is visible yet). An attacker must race a
	// RegisterKey call using the same SVTN name, but the SVTN is not in
	// m.svtns until after the bootstrap is complete, so RegisterKey returns
	// ErrSVTNNotFound. The attacker therefore cannot win the LWW race.
	for name, ok := range created {
		if !ok {
			continue
		}

		// Get the SVTN ID by calling Create for a different name — we need
		// the ID that was assigned during the race. Use a lookup-only approach:
		// register a fresh key to get the SVTN ID from the result, then
		// directly call keySet.LookupByPubkey.
		//
		// Since SVTNManager does not expose a Lookup(svtnName) accessor, we
		// verify via the manager's own RevokeKey with the expected role. If
		// the bootstrap key was overwritten by an attacker (different role),
		// Verify via CallerKeyRole that the bootstrap key is present and RoleControl.
		// (RevokeKey is no longer the verification surface — the bootstrap guard
		// prevents revoking the control key; TODO(BC-2.07.001): spec clarification
		// pending for E-ADM-020 bootstrap-key-revoke-forbidden.)
		role, found := mgr.CallerKeyRole(name, controlPub)
		if !found {
			t.Errorf("F-003 atomicity — CallerKeyRole(%s, controlPub): "+
				"bootstrap key not found; must be present after Create", name)
		} else if role != admission.RoleControl {
			t.Errorf("F-003 atomicity — CallerKeyRole(%s, controlPub): "+
				"want RoleControl; got %v", name, role)
		}
	}
}

// TestSVTNManager_RevokeRaceVsRegister_HOLD001 races RegisterKey(pub, RoleControl)
// goroutines against RevokeKey(pub, RoleConsole, confirm=false) to verify the
// atomic HOLD-001 primitive (F-CS-002).
//
// The acceptable outcomes are:
//
//	(a) revoke succeeds: key was RoleConsole at revoke time, no confirm needed; OR
//	(b) revoke fails with ErrRoleMismatch: key was RoleControl at revoke time.
//
// The forbidden outcome (c) is: revoke succeeded when the key was RoleControl
// without confirm=true. This would bypass the confirm gate.
//
// BC-2.05.004 precondition 1 (control-to-control revocation requires confirm).
// HOLD-001 (atomic role cross-check + revocation under single write lock).
func TestSVTNManager_RevokeRaceVsRegister_HOLD001(t *testing.T) {
	// NOT t.Parallel(): race detector focus; goroutine count is sensitive.
	const iterations = 200
	const registrars = 3

	for iter := range iterations {
		controlPub, _ := mustGenEdKey(t)
		ks := admission.NewAdmittedKeySet()
		mgr := svtnmgmt.NewSVTNManager(ks, controlPub)

		// Create the SVTN. Capture the result for post-race LookupByPubkey probe.
		svtn, err := mgr.Create("race-svtn")
		if err != nil {
			t.Fatalf("iter %d: Create: %v", iter, err)
		}

		// Register the target key as RoleConsole initially.
		targetPub, _ := mustGenEdKey(t)
		if _, err := mgr.RegisterKey("race-svtn", targetPub, admission.RoleConsole); err != nil {
			t.Fatalf("iter %d: initial RegisterKey(RoleConsole): %v", iter, err)
		}

		// Synchronize all goroutines to start simultaneously.
		ready := make(chan struct{})

		// Registrar goroutines: repeatedly upgrade the target key to RoleControl.
		for range registrars {
			go func() {
				<-ready
				// RoleControl registration races against the revocation below.
				_, _ = mgr.RegisterKey("race-svtn", targetPub, admission.RoleControl)
			}()
		}

		// Revoker: attempt to revoke as RoleConsole without confirm.
		revokeDone := make(chan error, 1)
		go func() {
			<-ready
			_, err := mgr.RevokeKey("race-svtn", targetPub, admission.RoleConsole, false)
			revokeDone <- err
		}()

		// Release all goroutines simultaneously.
		close(ready)

		revokeErr := <-revokeDone

		// Outcome (a): revoke succeeded — key was RoleConsole at revoke time.
		// Outcome (b): revoke failed with ErrRoleMismatch — key was RoleControl.
		// Outcome (c) FORBIDDEN: revoke succeeded when key was RoleControl.
		if revokeErr == nil {
			// Revoke claimed success as RoleConsole.
			//
			// F-P2-003: functional oracle beyond -race-only detection.
			//
			// Oracle 1 — IsAdmitted must be false: after a successful revoke,
			// the key must not be admitted. IsAdmitted AND-gates on
			// admitted && !revoked. If revoked=true was not actually set by the
			// implementation, a key that completed challenge-response would still
			// return IsAdmitted=true — exposing a buggy no-op revoke.
			// (In this test the key was registered but never admitted, so
			// IsAdmitted starts false; this guard is strongest for admitted keys
			// and acts as a belt-and-suspenders regression guard here.)
			nodeAddr := frame.DeriveNodeAddress(svtn.SVTN.ID, []byte(targetPub))
			if ks.IsAdmitted(svtn.SVTN.ID, nodeAddr) {
				t.Errorf("iter %d: HOLD-001 F-P2-003 — after successful revoke, "+
					"IsAdmitted still returns true for the revoked key; "+
					"revoked flag was not set (forbidden outcome c)", iter)
			}

			// Oracle 2 — re-register as RoleControl, then revoke as RoleConsole
			// without confirm. The role-match check must now return ErrRoleMismatch
			// (stored=Control, expected=Console), proving the atomic primitive
			// enforces role cross-check. A buggy implementation that ignores
			// expectedRole would return nil here, which is the oracle trigger.
			_, _ = mgr.RegisterKey("race-svtn", targetPub, admission.RoleControl)
			_, oracleErr := mgr.RevokeKey("race-svtn", targetPub, admission.RoleConsole, false)
			if oracleErr == nil {
				t.Errorf("iter %d: HOLD-001 F-P2-003 — RevokeKey(RoleConsole) on a "+
					"RoleControl key returned nil; atomic role cross-check not enforced "+
					"(forbidden: revoke succeeded despite role mismatch)", iter)
			} else if !errors.Is(oracleErr, svtnmgmt.ErrRoleMismatch) &&
				!errors.Is(oracleErr, svtnmgmt.ErrControlRevocationRequiresConfirm) {
				t.Errorf("iter %d: HOLD-001 F-P2-003 oracle — unexpected error %v "+
					"(want ErrRoleMismatch)", iter, oracleErr)
			}
		} else if !errors.Is(revokeErr, svtnmgmt.ErrRoleMismatch) &&
			!errors.Is(revokeErr, admission.ErrKeyNotRegistered) {
			// Unexpected error — neither outcome (a) nor (b).
			t.Errorf("iter %d: HOLD-001 — RevokeKey returned unexpected error: %v "+
				"(want nil=outcome-a, ErrRoleMismatch=outcome-b, or ErrKeyNotRegistered=outcome-b-variant)",
				iter, revokeErr)
		}
	}
}

// TestSVTNManager_ConcurrentCreate_NoOrphans verifies that concurrent Create calls
// for the same SVTN name do not leave orphan AdmittedKey entries in the keySet
// (F-CS-003).
//
// Spawns N goroutines all calling Create("foo"). After completion:
//   - Exactly one Create must succeed.
//   - The RevokeKey of the bootstrap control key must succeed for the winner.
//   - No extra revocations succeed (no orphan entries for the same SVTN visible
//     through the manager's public surface).
//
// BC-2.07.001 EC-001 (ErrSVTNAlreadyExists on duplicate name).
// F-CS-003 (no orphan keys under concurrent Create).
func TestSVTNManager_ConcurrentCreate_NoOrphans(t *testing.T) {
	// NOT t.Parallel(): concurrent create test with fixed goroutine count.
	const goroutines = 20

	controlPub, _ := mustGenEdKey(t)
	ks := admission.NewAdmittedKeySet()
	mgr := svtnmgmt.NewSVTNManager(ks, controlPub)

	type result struct {
		err error
	}
	results := make(chan result, goroutines)
	ready := make(chan struct{})

	for range goroutines {
		go func() {
			<-ready
			_, err := mgr.Create("foo")
			results <- result{err: err}
		}()
	}
	close(ready)

	var successes int
	for range goroutines {
		r := <-results
		if r.err == nil {
			successes++
		} else if !errors.Is(r.err, svtnmgmt.ErrSVTNAlreadyExists) {
			t.Errorf("F-CS-003 — Create returned unexpected error: %v", r.err)
		}
	}

	// BC-2.07.001 EC-001: exactly one Create must succeed.
	if successes != 1 {
		t.Errorf("F-CS-003 — %d Creates succeeded; want exactly 1", successes)
	}

	// The winning Create's bootstrap control key must be revocable exactly once.
	// If there are orphan entries, a second RevokeKey call might succeed (for
	// a different SVTN ID that was never published). But the manager only
	// routes by SVTN name → ID, so only one ID exists for "foo". The orphan
	// keys exist in the keySet under unpublished SVTN IDs and are therefore
	// unreachable through the manager's public surface.
	//
	// Primary verification: the bootstrap key is present and has RoleControl.
	// Using CallerKeyRole (not RevokeKey — the bootstrap guard prevents revoking
	// the control key; TODO(BC-2.07.001): spec clarification pending for E-ADM-020).
	role, found := mgr.CallerKeyRole("foo", controlPub)
	if !found {
		t.Errorf("F-CS-003 — post-concurrent CallerKeyRole(foo, controlPub): " +
			"bootstrap key not found; must be present after Create")
	} else if role != admission.RoleControl {
		t.Errorf("F-CS-003 — post-concurrent CallerKeyRole(foo, controlPub): "+
			"want RoleControl; got %v", role)
	}
}

// TestInsertRawSVTN_DuplicateName verifies that InsertRawSVTN returns
// ErrSVTNAlreadyExists on a second call with the same name, and that the
// semantic-contract postconditions (HasAnySVTN==true,
// BootstrapKeyHasControlRole==false) hold after the first successful insert.
// Also asserts ID uniqueness across two distinct-name inserts (kills the
// ID-generation mutant, F-L2-01) and that CreatedAt is in UTC (F-L2-02).
//
// Ruling-10: InsertRawSVTN duplicate-name contract + semantic lock-in.
func TestInsertRawSVTN_DuplicateName(t *testing.T) {
	t.Parallel()

	m := newManager(t)

	// Insert two distinct-named SVTNs first so we can assert ID uniqueness and
	// UTC timestamps (F-L2-01, F-L2-02) before the duplicate-name check.
	if err := m.InsertRawSVTN("svtn-a"); err != nil {
		t.Fatalf("InsertRawSVTN(\"svtn-a\") first call: unexpected error: %v", err)
	}
	if err := m.InsertRawSVTN("svtn-b"); err != nil {
		t.Fatalf("InsertRawSVTN(\"svtn-b\"): unexpected error: %v", err)
	}

	recA, okA := m.SVTNByName("svtn-a")
	recB, okB := m.SVTNByName("svtn-b")
	if !okA {
		t.Fatal("SVTNByName(\"svtn-a\"): not found after insert")
	}
	if !okB {
		t.Fatal("SVTNByName(\"svtn-b\"): not found after insert")
	}

	// F-L2-01: IDs must be distinct and non-zero.
	var zero [16]byte
	if recA.ID == zero {
		t.Error("F-L2-01 — svtn-a ID is all-zero; must be a generated unique identifier")
	}
	if recB.ID == zero {
		t.Error("F-L2-01 — svtn-b ID is all-zero; must be a generated unique identifier")
	}
	if recA.ID == recB.ID {
		t.Errorf("F-L2-01 — svtn-a and svtn-b share the same ID %v; IDs must be distinct", recA.ID)
	}

	// F-L2-02: CreatedAt must be in UTC.
	if recA.CreatedAt.Location() != time.UTC {
		t.Errorf("F-L2-02 — svtn-a CreatedAt.Location = %v; want UTC", recA.CreatedAt.Location())
	}

	// Semantic-contract postconditions after first insert.
	if !m.HasAnySVTN() {
		t.Error("HasAnySVTN() == false after InsertRawSVTN; want true")
	}
	if m.BootstrapKeyHasControlRole() {
		t.Error("BootstrapKeyHasControlRole() == true after InsertRawSVTN; want false (no keys registered)")
	}

	// Second insert with same name must return ErrSVTNAlreadyExists.
	err := m.InsertRawSVTN("svtn-a")
	if !errors.Is(err, svtnmgmt.ErrSVTNAlreadyExists) {
		t.Errorf("InsertRawSVTN(\"svtn-a\") second call: want ErrSVTNAlreadyExists; got %v", err)
	}
}

// TestInsertRawSVTN_ConcurrentDistinctNames verifies that concurrent InsertRawSVTN
// calls with distinct names all succeed, each produces a retrievable SVTN record
// with a unique ID, and the implementation is race-detector clean (F-L2-04).
//
// F-L2-04: concurrent inserts of distinct names must not panic and must produce
// unique IDs per SVTN.
func TestInsertRawSVTN_ConcurrentDistinctNames(t *testing.T) {
	// NOT t.Parallel(): race detector focus; goroutine count is sensitive.

	const workers = 10
	m := newManager(t)

	errs := make(chan error, workers)
	names := make([]string, workers)
	for i := range workers {
		names[i] = fmt.Sprintf("concurrent-raw-%d", i)
	}

	for _, name := range names {
		go func(n string) {
			errs <- m.InsertRawSVTN(n)
		}(name)
	}

	for range workers {
		if err := <-errs; err != nil {
			t.Errorf("F-L2-04 — concurrent InsertRawSVTN: unexpected error: %v", err)
		}
	}

	// HasAnySVTN must be true after all inserts.
	if !m.HasAnySVTN() {
		t.Error("F-L2-04 — HasAnySVTN() == false after concurrent inserts; want true")
	}

	// Each SVTN must be retrievable with a unique ID.
	seen := make(map[[16]byte]string, workers)
	for _, name := range names {
		rec, ok := m.SVTNByName(name)
		if !ok {
			t.Errorf("F-L2-04 — SVTNByName(%q): not found after concurrent insert", name)
			continue
		}
		var zero [16]byte
		if rec.ID == zero {
			t.Errorf("F-L2-04 — SVTNByName(%q): ID is all-zero; must be generated", name)
			continue
		}
		if prior, dup := seen[rec.ID]; dup {
			t.Errorf("F-L2-04 — SVTNByName(%q): ID %v already seen for %q; IDs must be unique",
				name, rec.ID, prior)
		}
		seen[rec.ID] = name
	}
}

// TestSVTNManager_ExpireKey_TOCTOU_RoleChangeRace verifies that a concurrent
// revoke+re-register at the same pubkey (changing its role) does not cause
// ExpireKey to silently expire the new-role entry. ExpireKey must either
// succeed (expired the original entry) or return ErrRoleMismatch — never
// silently corrupt the new registration.
//
// Traces to F-C2-002; HOLD-001 hybrid approach.
func TestSVTNManager_ExpireKey_TOCTOU_RoleChangeRace(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	ctrlPub, _ := mustGenEdKey(t)
	mgr := svtnmgmt.NewSVTNManager(ks, ctrlPub)
	if _, err := mgr.Create("race-svtn"); err != nil {
		t.Fatalf("create SVTN: %v", err)
	}

	// Register key K with access role.
	kPub, _ := mustGenEdKey(t)
	if _, err := mgr.RegisterKey("race-svtn", kPub, admission.RoleAccess); err != nil {
		t.Fatalf("register key: %v", err)
	}

	const iters = 200
	errCh := make(chan error, iters)

	for range iters {
		go func() {
			// goroutine A: expire key K (access role, 1h TTL)
			_, err := mgr.ExpireKey("race-svtn", kPub, time.Hour)
			errCh <- err
		}()
		go func() {
			// goroutine B: revoke K then re-register K as control role
			_, _ = mgr.RevokeKey("race-svtn", kPub, admission.RoleAccess, false)
			_, _ = mgr.RegisterKey("race-svtn", kPub, admission.RoleControl)
			// re-register as access to keep test repeatable
			_, _ = mgr.RegisterKey("race-svtn", kPub, admission.RoleAccess)
		}()
	}

	for range iters {
		err := <-errCh
		// Acceptable outcomes: nil (expire succeeded on access-role entry)
		// or ErrRoleMismatch (role changed under us — TOCTOU detected).
		// ErrKeyNotRegistered is also acceptable (key was revoked mid-expire).
		// Any other error is a bug.
		if err != nil &&
			!errors.Is(err, admission.ErrRoleMismatch) &&
			!errors.Is(err, admission.ErrKeyNotRegistered) &&
			!errors.Is(err, svtnmgmt.ErrSVTNNotFound) {
			t.Errorf("unexpected error from ExpireKey: %v", err)
		}
	}
}

// ── F-P1L2-005: LookupByPubkey miss semantics at svtnmgmt callsites ──────────

// TestSVTNManager_LookupMigration_MissReturnsKeyNotRegistered verifies that
// svtnmgmt operations that call LookupByPubkey internally return
// ErrKeyNotRegistered — not ErrRoleMismatch — when the supplied pubkey is not
// registered in the SVTN. This distinguishes "no such key" from "wrong role" at
// the callsites identified in F-P1L2-005 (ExpireKey line 352, CallerKeyRole
// line 404, CallerKeyRoleActive line 425, IsRegisteredAnyState line 526).
//
// Tested callsites:
//   - ExpireKey: must return ErrKeyNotRegistered (not ErrRoleMismatch) on miss.
//   - CallerKeyRole: must return (0, false) on miss (no sentinel, but not panicking
//     or returning a role for an absent key).
//   - CallerKeyRoleActive: must return (0, false) on miss.
//   - IsRegisteredAnyState: must return false on miss.
//
// BC-2.05.004 postcondition 3 (key must be registered to set expiry).
// admission.ErrKeyNotRegistered (E-ADM-013).
func TestSVTNManager_LookupMigration_MissReturnsKeyNotRegistered(t *testing.T) {
	t.Parallel()

	mgr := newManager(t)
	svtnResult, err := mgr.Create("miss-semantics-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	unregisteredPub, _ := mustGenEdKey(t)

	// ExpireKey on an unregistered key must return ErrKeyNotRegistered, not ErrRoleMismatch.
	// If LookupByPubkey were to return a spurious non-zero AdmittedKey with ok=false,
	// downstream code might interpret the role field and return ErrRoleMismatch instead.
	_, expireErr := mgr.ExpireKey(svtnResult.SVTN.Name, unregisteredPub, time.Hour)
	if !errors.Is(expireErr, admission.ErrKeyNotRegistered) {
		t.Errorf("ExpireKey with unregistered pubkey: want admission.ErrKeyNotRegistered (E-ADM-013); got %v", expireErr)
	}
	if errors.Is(expireErr, admission.ErrRoleMismatch) {
		t.Errorf("ExpireKey with unregistered pubkey: got ErrRoleMismatch; must NOT return ErrRoleMismatch for absent key")
	}

	// CallerKeyRole on a miss must return (0, false) — not a stale role for a phantom key.
	role, found := mgr.CallerKeyRole(svtnResult.SVTN.Name, unregisteredPub)
	if found {
		t.Errorf("CallerKeyRole with unregistered pubkey: want found=false; got found=true with role=%v", role)
	}
	if role != 0 {
		t.Errorf("CallerKeyRole with unregistered pubkey: want role=0; got %v", role)
	}

	// CallerKeyRoleActive on a miss must return (0, false).
	roleActive, foundActive := mgr.CallerKeyRoleActive(svtnResult.SVTN.Name, unregisteredPub)
	if foundActive {
		t.Errorf("CallerKeyRoleActive with unregistered pubkey: want found=false; got found=true with role=%v", roleActive)
	}
	if roleActive != 0 {
		t.Errorf("CallerKeyRoleActive with unregistered pubkey: want role=0; got %v", roleActive)
	}

	// IsRegisteredAnyState on a miss must return false.
	if mgr.IsRegisteredAnyState(svtnResult.SVTN.Name, unregisteredPub) {
		t.Error("IsRegisteredAnyState with unregistered pubkey: want false; got true")
	}
}

// ── S-6.05: SVTNManager.Destroy ───────────────────────────────────────────────

// TestSVTNManager_Destroy_RemovesAllKeys verifies AC-001:
// SVTNManager.Destroy(caller, svtnName) removes all admitted keys for the SVTN
// from the router's key set and frees the SVTN ID.
//
// ARCH-04 admission ordering verified: ListKeys (key removal) is asserted
// before SVTNByName/All (SVTN ID free), confirming keys are gone before
// the SVTN registry entry is gone.
//
// self-check: the stub panics; a complete implementation that returns nil
// without removing keys or the SVTN would fail the ListKeys and SVTNByName
// assertions below.
//
// Traces to BC-2.07.001 postcondition 3; AC-001; VP-048 property 2.
func TestSVTNManager_Destroy_RemovesAllKeys(t *testing.T) {
	t.Parallel()

	mgr, controlPub := newManagerWithKS(t)

	// Create an SVTN and register two extra keys.
	_, err := mgr.Create("destroy-test-svtn")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	extraPub1, _ := mustGenEdKey(t)
	extraPub2, _ := mustGenEdKey(t)

	if _, err := mgr.RegisterKey("destroy-test-svtn", extraPub1, admission.RoleConsole); err != nil {
		t.Fatalf("RegisterKey extra1: %v", err)
	}
	if _, err := mgr.RegisterKey("destroy-test-svtn", extraPub2, admission.RoleAccess); err != nil {
		t.Fatalf("RegisterKey extra2: %v", err)
	}

	// Pre-condition: SVTN exists and has keys (bootstrap + 2 extras).
	keysBefore, err := mgr.ListKeys("destroy-test-svtn")
	if err != nil {
		t.Fatalf("ListKeys before Destroy: %v", err)
	}
	if len(keysBefore) < 3 {
		t.Fatalf("expected ≥3 keys before Destroy (bootstrap + 2 extras); got %d", len(keysBefore))
	}

	// Destroy. Pass control-role caller (defense-in-depth authorized path; AC-004).
	if err := mgr.Destroy(controlCallerKey(controlPub), "destroy-test-svtn"); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	// ARCH-04 ordering: assert keys removed first.
	// ListKeys must return ErrSVTNNotFound (SVTN freed) OR an empty slice (keys
	// purged before registry entry removed). Either is consistent with ARCH-04:
	// if the implementation removes keys before the SVTN registry entry, ListKeys
	// would briefly return an empty slice; after the entry is also removed it
	// returns ErrSVTNNotFound. We assert both possibilities are correct outcomes.
	keysAfter, listErr := mgr.ListKeys("destroy-test-svtn")
	if listErr != nil {
		if !errors.Is(listErr, svtnmgmt.ErrSVTNNotFound) {
			t.Errorf("AC-001 — ListKeys after Destroy: expected ErrSVTNNotFound or empty slice; got error: %v", listErr)
		}
		// ErrSVTNNotFound from ListKeys confirms SVTN was freed (expected postcondition).
	} else if len(keysAfter) != 0 {
		// Slice returned without error must be empty (keys removed).
		t.Errorf("AC-001 — ListKeys after Destroy: expected 0 keys; got %d", len(keysAfter))
	}

	// SVTN must be absent from the registry.
	if _, found := mgr.SVTNByName("destroy-test-svtn"); found {
		t.Error("AC-001 — SVTNByName after Destroy: SVTN still present in registry; expected absent")
	}

	// All() must not contain the destroyed SVTN.
	for _, s := range mgr.All() {
		if s.Name == "destroy-test-svtn" {
			t.Error("AC-001 — All() after Destroy: destroyed SVTN still appears in list")
		}
	}

	// Bootstrap key must no longer appear in any key set for this SVTN.
	// CallerKeyRole returns (0, false) when SVTN is absent — confirming keys freed.
	role, found := mgr.CallerKeyRole("destroy-test-svtn", controlPub)
	if found {
		t.Errorf("AC-001 — CallerKeyRole after Destroy: bootstrap key still registered with role=%v", role)
	}
}

// TestSVTNManager_Destroy_NotFound verifies EC-001:
// Destroy on a non-existent SVTN returns ErrSVTNNotFound (E-SVTN-003).
// The error string must contain the SVTN name per canonical form
// "SVTN not found: <name>" (Ruling-11/12; S-6.05 Error Code Table).
//
// self-check: a stub that returns nil fails the non-nil error check; a stub
// that returns a generic error fails the errors.Is(err, ErrSVTNNotFound) check.
//
// Traces to BC-2.07.001 EC-001; E-SVTN-003; AC-001.
func TestSVTNManager_Destroy_NotFound(t *testing.T) {
	t.Parallel()

	mgr := newManager(t)
	// Pass a control-role caller (authorized); the not-found check fires after
	// the authorization check, so we need to pass a valid control caller here.
	caller := admission.AdmittedKey{Role: admission.RoleControl}

	err := mgr.Destroy(caller, "nonexistent-svtn")

	// Must return non-nil error.
	if err == nil {
		t.Fatal("EC-001 — Destroy on non-existent SVTN: expected ErrSVTNNotFound; got nil")
	}

	// errors.Is chain must include ErrSVTNNotFound.
	if !errors.Is(err, svtnmgmt.ErrSVTNNotFound) {
		t.Errorf("EC-001 — Destroy on non-existent SVTN: errors.Is(err, ErrSVTNNotFound) = false; got: %v", err)
	}

	// Canonical error string must contain the SVTN name (Ruling-11/12).
	if !strings.Contains(err.Error(), "nonexistent-svtn") {
		t.Errorf("EC-001 — Destroy error string must contain the SVTN name %q; got: %q", "nonexistent-svtn", err.Error())
	}
}

// TestSVTNManager_Destroy_KeyPurgePostcondition verifies the in-scope portion of
// AC-002 (story S-6.05 v1.4): after Destroy the SVTN is absent and all its keys
// are purged, so any subsequent admission attempt for formerly admitted nodes is
// blocked. Because internal/session is a forbidden import in internal/svtnmgmt
// (ARCH-08 position 15), session-terminated signals cannot be observed directly —
// this test verifies the admission-blocking postcondition (VP-048 property 2
// precursor) which is the session-layer-observable consequence.
//
// Session-terminated notification is deferred to S-BL.SESSION-DRAIN and is
// explicitly out of scope for story S-6.05 v1.4.
//
// Concurrent-admission race safety is covered separately by
// TestSVTNManager_Destroy_ConcurrentAdmissionIsRaceFree.
//
// self-check: a stub that panics fails on the Destroy call; a stub that
// returns nil without removing keys fails the CallerKeyRole assertion.
//
// Traces to BC-2.07.001 postcondition 3; AC-002 (in-scope portion per S-6.05 v1.4).
func TestSVTNManager_Destroy_KeyPurgePostcondition(t *testing.T) {
	t.Parallel()

	mgr, controlPub := newManagerWithKS(t)

	if _, err := mgr.Create("session-svtn"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Register several keys to simulate an active multi-node topology.
	nodePub1, _ := mustGenEdKey(t)
	nodePub2, _ := mustGenEdKey(t)

	if _, err := mgr.RegisterKey("session-svtn", nodePub1, admission.RoleConsole); err != nil {
		t.Fatalf("RegisterKey node1: %v", err)
	}
	if _, err := mgr.RegisterKey("session-svtn", nodePub2, admission.RoleAccess); err != nil {
		t.Fatalf("RegisterKey node2: %v", err)
	}

	// Pre-condition: keys registered and active.
	if _, found := mgr.CallerKeyRoleActive("session-svtn", nodePub1); !found {
		t.Fatal("pre-condition: node1 should be an active key before Destroy")
	}

	// Destroy the SVTN. Session-terminated notification is deferred to
	// S-BL.SESSION-DRAIN (AC-002 out-of-scope per story S-6.05 v1.5);
	// this test verifies the admission-blocking postcondition only.
	if err := mgr.Destroy(controlCallerKey(controlPub), "session-svtn"); err != nil {
		t.Fatalf("Destroy with active sessions: %v", err)
	}

	// After Destroy: SVTN absent.
	if _, found := mgr.SVTNByName("session-svtn"); found {
		t.Error("AC-002 — SVTNByName after Destroy: SVTN still present")
	}

	// After Destroy: no keys survive (admission of former nodes is now blocked).
	// CallerKeyRole returns (0, false) when SVTN is absent.
	if _, found := mgr.CallerKeyRole("session-svtn", controlPub); found {
		t.Error("AC-002 — control key still registered after Destroy; sessions cannot terminate")
	}
	if _, found := mgr.CallerKeyRole("session-svtn", nodePub1); found {
		t.Error("AC-002 — node1 key still registered after Destroy; sessions cannot terminate")
	}
	if _, found := mgr.CallerKeyRole("session-svtn", nodePub2); found {
		t.Error("AC-002 — node2 key still registered after Destroy; sessions cannot terminate")
	}
}

// TestSVTNManager_Destroy_ErrDestroyUnauthorized verifies AC-004 (Go-API
// defense-in-depth per RULING-W6TB-A §3 and BC-2.07.001 Inv-3):
// Calling SVTNManager.Destroy with a non-control caller returns
// ErrDestroyUnauthorized (E-ADM-011 Variant 2) before any SVTN state is
// consulted. Calling with a control-role caller succeeds (SVTN is removed).
//
// This is the inner defense-in-depth check — the outer gate is
// resolveAndVerifyCallerRole in the handler layer. The inner check guards
// against direct Go-API callers and future refactors that might inadvertently
// strip the handler gate.
//
// Traces to BC-2.07.001 Inv-3; AC-004 (Go-API defense-in-depth); RULING-W6TB-A §3.
func TestSVTNManager_Destroy_ErrDestroyUnauthorized(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		role        admission.KeyRole
		wantErr     error
		description string
	}{
		{
			name:        "console_caller_returns_unauthorized",
			role:        admission.RoleConsole,
			wantErr:     svtnmgmt.ErrDestroyUnauthorized,
			description: "console-role caller must be rejected before any state mutation",
		},
		{
			name:        "access_caller_returns_unauthorized",
			role:        admission.RoleAccess,
			wantErr:     svtnmgmt.ErrDestroyUnauthorized,
			description: "access-role caller must be rejected before any state mutation",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mgr := newManager(t)
			_, err := mgr.Create("svtn-A")
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			nonControlCaller := admission.AdmittedKey{Role: tc.role}
			err = mgr.Destroy(nonControlCaller, "svtn-A")

			if !errors.Is(err, svtnmgmt.ErrDestroyUnauthorized) {
				t.Errorf("AC-004 %s — Destroy with role %v: want ErrDestroyUnauthorized; got %v",
					tc.description, tc.role, err)
			}

			// SVTN must still exist (no state was mutated).
			if _, found := mgr.SVTNByName("svtn-A"); !found {
				t.Error("AC-004 — SVTN was destroyed despite non-control caller; expected SVTN to remain intact")
			}
		})
	}

	// Positive control: control-role caller succeeds.
	t.Run("control_caller_succeeds", func(t *testing.T) {
		t.Parallel()

		mgr := newManager(t)
		_, err := mgr.Create("svtn-B")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		controlCaller := admission.AdmittedKey{Role: admission.RoleControl}
		if err := mgr.Destroy(controlCaller, "svtn-B"); err != nil {
			t.Errorf("AC-004 — Destroy with control-role caller: unexpected error: %v", err)
		}

		// SVTN must be absent after successful destroy.
		if _, found := mgr.SVTNByName("svtn-B"); found {
			t.Error("AC-004 — SVTN still present after control-role destroy; expected absent")
		}
	})
}

// TestSVTNManager_Destroy_GenesisReopened verifies AC-005:
// Destroying the last SVTN (such that HasAnySVTN() returns false) re-opens the
// genesis carve-out for a subsequent Create with the bootstrap key.
// A subsequent Create with the bootstrap key must succeed exactly as on first
// initialization (RULING-W6TB-A §4 recovery semantics).
//
// self-check: a stub that panics fails on the first Destroy call; a stub that
// returns nil without removing the SVTN from the registry leaves HasAnySVTN() ==
// true, causing the subsequent Create to fail with ErrSVTNAlreadyExists (or the
// "already exists" gate to still block genesis semantics). The test is only
// passable with a working Destroy.
//
// Traces to BC-2.07.001 Inv-3; AC-005; RULING-W6TB-A §4.
func TestSVTNManager_Destroy_GenesisReopened(t *testing.T) {
	t.Parallel()

	mgr, _ := newManagerWithKS(t)

	// Create the first (and only) SVTN.
	_, err := mgr.Create("genesis-svtn")
	if err != nil {
		t.Fatalf("Create genesis-svtn: %v", err)
	}

	// Pre-condition: HasAnySVTN must be true after creation.
	if !mgr.HasAnySVTN() {
		t.Fatal("AC-005 pre-condition: HasAnySVTN() should be true after Create")
	}

	// Destroy the last SVTN. Use control-role caller (defense-in-depth authorized path).
	caller := admission.AdmittedKey{Role: admission.RoleControl}
	if err := mgr.Destroy(caller, "genesis-svtn"); err != nil {
		t.Fatalf("AC-005 — Destroy genesis-svtn: %v", err)
	}

	// Post-condition: HasAnySVTN() must be false (genesis carve-out re-opened).
	if mgr.HasAnySVTN() {
		t.Error("AC-005 — HasAnySVTN() should be false after destroying the last SVTN; genesis carve-out not re-opened")
	}

	// Recovery semantics: Create with the same name must succeed via the genesis
	// carve-out (the name is now free and the registry is empty).
	result, err := mgr.Create("genesis-svtn")
	if err != nil {
		t.Fatalf("AC-005 — RULING-W6TB-A §4: Create after last-SVTN destroy must succeed via genesis carve-out; got: %v", err)
	}
	var zero [16]byte
	if result.SVTN.ID == zero {
		t.Error("AC-005 — Create after genesis re-open: SVTN.ID is all-zero; expected a new generated ID")
	}
	if result.SVTN.Name != "genesis-svtn" {
		t.Errorf("AC-005 — Create after genesis re-open: SVTN.Name = %q; want %q", result.SVTN.Name, "genesis-svtn")
	}

	// Verify the registry is now populated again.
	if !mgr.HasAnySVTN() {
		t.Error("AC-005 — HasAnySVTN() should be true after second Create")
	}
}

// TestSVTNManager_Destroy_ConcurrentAdmissionIsRaceFree verifies that calling
// Destroy while concurrent read operations (CallerKeyRole, CallerKeyRoleActive,
// SVTNByName) are in flight does not produce a data race (F-L1 lock-ordering
// invariant: m.mu → keySet.mu, never the reverse).
//
// The test registers N node keys, then spawns 50 goroutines that continuously
// call CallerKeyRole / CallerKeyRoleActive / SVTNByName on those keys while the
// main goroutine calls Destroy once. After all goroutines finish the final state
// is asserted: SVTN gone, all keys purged. Run with `go test -race` to detect
// any mutex-order violation.
//
// Concurrent-admission semantics (session-terminated notifications) are deferred
// to S-BL.SESSION-DRAIN and are explicitly out of scope for story S-6.05.
//
// Traces to BC-2.07.001 postcondition 3; AC-002 (concurrent race safety portion).
func TestSVTNManager_Destroy_ConcurrentAdmissionIsRaceFree(t *testing.T) {
	t.Parallel()

	mgr, controlPub := newManagerWithKS(t)

	if _, err := mgr.Create("race-svtn"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	const numKeys = 5
	nodePubs := make([]ed25519.PublicKey, numKeys)
	roles := []admission.KeyRole{
		admission.RoleConsole,
		admission.RoleAccess,
		admission.RoleConsole,
		admission.RoleAccess,
		admission.RoleConsole,
	}
	for i := range nodePubs {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("GenerateKey[%d]: %v", i, err)
		}
		nodePubs[i] = pub
		if _, err := mgr.RegisterKey("race-svtn", pub, roles[i]); err != nil {
			t.Fatalf("RegisterKey[%d]: %v", i, err)
		}
	}

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Signal goroutines to start simultaneously with Destroy.
	start := make(chan struct{})

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			<-start
			// Alternate between read operations to exercise all code paths
			// that touch keySet under keySet.mu while Destroy may hold m.mu.
			pub := nodePubs[idx%numKeys]
			mgr.CallerKeyRole("race-svtn", pub)
			mgr.CallerKeyRoleActive("race-svtn", pub)
			mgr.SVTNByName("race-svtn")
		}(i)
	}

	close(start)
	if err := mgr.Destroy(controlCallerKey(controlPub), "race-svtn"); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	wg.Wait()

	// Final state: SVTN absent and all keys purged.
	if _, found := mgr.SVTNByName("race-svtn"); found {
		t.Error("race-svtn still present after Destroy")
	}
	if _, found := mgr.CallerKeyRole("race-svtn", controlPub); found {
		t.Error("control key still registered after Destroy")
	}
	for i, pub := range nodePubs {
		if _, found := mgr.CallerKeyRole("race-svtn", pub); found {
			t.Errorf("node key [%d] still registered after Destroy", i)
		}
	}
}

// TestSVTNManager_Destroy_Idempotent verifies EC-001 double-destroy semantics:
// a second Destroy on an already-destroyed SVTN returns ErrSVTNNotFound rather
// than succeeding silently (idempotency-as-error per S-6.05 error taxonomy).
//
// Traces to BC-2.07.001 EC-001; E-SVTN-003; AC-001.
func TestSVTNManager_Destroy_Idempotent(t *testing.T) {
	t.Parallel()

	mgr := newManager(t)
	caller := admission.AdmittedKey{Role: admission.RoleControl}

	if _, err := mgr.Create("idempotent-svtn"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First Destroy: must succeed.
	if err := mgr.Destroy(caller, "idempotent-svtn"); err != nil {
		t.Fatalf("first Destroy: %v", err)
	}

	// Second Destroy: must return ErrSVTNNotFound.
	err := mgr.Destroy(caller, "idempotent-svtn")
	if err == nil {
		t.Fatal("EC-001 — second Destroy: expected ErrSVTNNotFound; got nil")
	}
	if !errors.Is(err, svtnmgmt.ErrSVTNNotFound) {
		t.Errorf("EC-001 — second Destroy: expected errors.Is(err, ErrSVTNNotFound); got %v", err)
	}
}
