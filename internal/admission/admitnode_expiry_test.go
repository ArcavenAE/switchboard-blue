// admitnode_expiry_test.go — AC-013 tests for AdmitNode expiry check.
//
// BC-2.05.001 Postcondition 6 / Invariant 5 (O-1 ruling §15):
// AdmitNode MUST return ErrKeyExpired when the key has a non-zero expiry
// timestamp and time.Now().UTC() is after that timestamp.
//
// These tests are RED GATE tests. TestAdmitNode_ExpiredKey_ReturnsErrKeyExpired
// MUST FAIL against the current unmodified AdmitNode (which does not check
// expiry). TestAdmitNode_FutureExpiry_Succeeds and
// TestAdmitNode_NoExpiry_Succeeds pass only once the expiry check is added.
//
// Traces to BC-2.05.001 PC-6, Invariant 5; S-BL.NODE-IDENTIFY-WIRE AC-013;
// rulings §15 (O-1 ruling, human-ratified 2026-07-18).
package admission_test

import (
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// ── AC-013: TestAdmitNode_ExpiredKey_ReturnsErrKeyExpired ──────────────────────

// TestAdmitNode_ExpiredKey_ReturnsErrKeyExpired verifies that AdmitNode returns
// ErrKeyExpired when the key's expiry is non-zero and the expiry timestamp is
// in the past.
//
// This is the O-1 RED GATE test — it MUST FAIL against the current AdmitNode
// because AdmitNode does not yet check expiry (unlike ReAuthenticate, which
// already does). The implementation task (Task 16 in S-BL.NODE-IDENTIFY-WIRE)
// adds the expiry check mirroring ReAuthenticate's pattern.
//
// Traces to BC-2.05.001 PC-6 (AdmitNode returns ErrKeyExpired for past-expiry
// key); BC-2.05.001 Invariant 5 (symmetric expiry across AdmitNode and
// ReAuthenticate); S-BL.NODE-IDENTIFY-WIRE AC-013.
func TestAdmitNode_ExpiredKey_ReturnsErrKeyExpired(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xE1)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Set expiry to 1 second in the past.
	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	pastExpiry := time.Now().UTC().Add(-time.Second)
	if err := ks.SetKeyExpiry(svtnID, nodeAddr, pastExpiry); err != nil {
		t.Fatalf("SetKeyExpiry: %v", err)
	}

	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
	if !errors.Is(err, admission.ErrKeyExpired) {
		t.Errorf("AdmitNode with past-expiry key: want ErrKeyExpired (E-ADM-015), got %v", err)
	}
}

// ── AC-013: TestAdmitNode_FutureExpiry_Succeeds ────────────────────────────────

// TestAdmitNode_FutureExpiry_Succeeds verifies that AdmitNode does NOT return
// ErrKeyExpired when the key has a non-zero expiry timestamp that is in the
// future (expiry has not yet been reached).
//
// Traces to BC-2.05.001 PC-6 (guard does not fire for future expiry);
// S-BL.NODE-IDENTIFY-WIRE AC-013.
func TestAdmitNode_FutureExpiry_Succeeds(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xE2)

	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	// Set expiry to 1 hour in the future — key is not yet expired.
	nodeAddr := nodeAddrForTest(svtnID, nodePub)
	futureExpiry := time.Now().UTC().Add(time.Hour)
	if err := ks.SetKeyExpiry(svtnID, nodeAddr, futureExpiry); err != nil {
		t.Fatalf("SetKeyExpiry: %v", err)
	}

	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
	if err != nil {
		t.Errorf("AdmitNode with future-expiry key: want nil, got %v", err)
	}
}

// ── AC-013: TestAdmitNode_NoExpiry_Succeeds ────────────────────────────────────

// TestAdmitNode_NoExpiry_Succeeds verifies that AdmitNode does NOT return
// ErrKeyExpired when the key has a zero expiry (no expiry set). The expiry
// guard must not fire on the zero-value time.Time.
//
// Traces to BC-2.05.001 PC-6 (guard does not fire when expiry is zero);
// BC-2.05.001 Invariant 5; S-BL.NODE-IDENTIFY-WIRE AC-013.
func TestAdmitNode_NoExpiry_Succeeds(t *testing.T) {
	t.Parallel()

	_, routerPriv := mustGenEd25519(t)
	nodePub, nodePriv := mustGenEd25519(t)
	svtnID := mustSVTN(0xE3)

	ks := admission.NewAdmittedKeySet()
	// RegisterKey without SetKeyExpiry — expiry remains zero.
	ks.RegisterKey(svtnID, nodePub, admission.RoleAccess)

	ch := mustGenerateChallenge(t, routerPriv)
	sig := ed25519.Sign(nodePriv, ch.Nonce[:])
	resp := admission.ChallengeResponse{NonceSig: sig}

	err := admission.AdmitNode(ch, resp, nodePub, svtnID, ks)
	if err != nil {
		t.Errorf("AdmitNode with no-expiry key: want nil, got %v", err)
	}
}
