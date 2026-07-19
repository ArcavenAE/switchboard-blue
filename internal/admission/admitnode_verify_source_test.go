// admitnode_verify_source_test.go — F-1 discriminating security test for
// AdmitNode's signature-verification source.
//
// BC-2.05.001 PC-3 states: "The router verifies the signature using the
// STORED public key, not the frame-supplied pubKey parameter."
//
// AdmitNode currently verifies against the frame-supplied `pubKey`:
//
//	if !ed25519.Verify(pubKey, challenge.Nonce[:], resp.NonceSig) {
//
// The correct implementation must verify against `liveEntry.PublicKey` (the
// stored key re-fetched under the write lock). This file provides:
//
//  1. TestAdmitNode_VerifiesAgainstStoredKey_NotFramePubkey — the PRIMARY
//     discriminating test.  Simulates a node-address collision impersonation
//     attempt.  MUST FAIL against the current (buggy) code and PASS only
//     after the fix.
//
//  2. TestAdmitNode_StoredKeyMatches_Admits — positive companion.  Same
//     setup but with no mismatch (stored == frame pubkey, correct signature).
//     Guards against the fix over-rejecting legitimate admissions.
//
// Package: admission (white-box; accesses unexported keys map and
// AdmittedKey struct directly so we can install an entry at an explicit
// address without relying on DeriveNodeAddress).
package admission

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
)

// ── TestAdmitNode_VerifiesAgainstStoredKey_NotFramePubkey ─────────────────────

// TestAdmitNode_VerifiesAgainstStoredKey_NotFramePubkey is the PRIMARY
// discriminating test for BC-2.05.001 PC-3.
//
// It models a node-address collision impersonation attack where an attacker's
// DeriveNodeAddress matches the address of a registered victim entry:
//
//  1. Two keypairs: victim (vPub, vPriv) and attacker (aPub, aPriv).
//  2. attackerAddr = DeriveNodeAddress(svtnID, aPub).
//  3. At attackerAddr in the keyset, store an entry whose PublicKey is vPub
//     (the victim's key) — simulating a collision where the registered node
//     is the victim but the attacker derived the same address.
//  4. Attacker signs the nonce with aPriv and calls AdmitNode with aPub.
//     The DeriveNodeAddress lookup lands on the installed entry.
//  5. Buggy code verifies aPub's signature against aPub → returns nil
//     (impersonation succeeds).  Fixed code verifies against the stored
//     vPub → returns ErrSignatureVerificationFailed (impersonation rejected).
//
// This test MUST FAIL against admission.go line ~525
// `ed25519.Verify(pubKey, …)` (current buggy code) and PASS only after
// the fix changes that to `ed25519.Verify(liveEntry.PublicKey, …)`.
//
// Discriminating rationale: The test uses errors.Is(err, ErrSignatureVerificationFailed).
// Buggy code: aPriv signs correctly for aPub → Verify(aPub, …) = true → err=nil.
// Fixed code: Verify(vPub, …) = false (aPriv sig doesn't verify under vPub) → err=ErrSignatureVerificationFailed.
//
// NOT t.Parallel(): directly mutates the keyset's internal map under the mutex.
// A sequential execution is the safest contract for a white-box setup.
//
// Traces to BC-2.05.001 PC-3 ("verifies using the stored public key"); F-1.
func TestAdmitNode_VerifiesAgainstStoredKey_NotFramePubkey(t *testing.T) {
	var svtnID [16]byte
	svtnID[0] = 0xF1
	svtnID[1] = 0xA0 // non-zero, distinct byte

	// Generate two independent keypairs.
	vPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey (victim): %v", err)
	}
	aPub, aPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey (attacker): %v", err)
	}
	// Router keypair for GenerateChallenge.
	_, routerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey (router): %v", err)
	}

	// Derive the address the attacker's DeriveNodeAddress call would produce.
	attackerAddr := frame.DeriveNodeAddress(svtnID, []byte(aPub))

	// Construct the AdmittedKeySet and install the victim's PublicKey at
	// attackerAddr — directly via the internal map, bypassing RegisterKey
	// (which would call DeriveNodeAddress(svtnID, vPub) giving a different
	// address than attackerAddr).
	ks := NewAdmittedKeySet()
	ks.mu.Lock()
	if ks.keys[svtnID] == nil {
		ks.keys[svtnID] = make(map[[8]byte]*AdmittedKey)
	}
	ks.keys[svtnID][attackerAddr] = &AdmittedKey{
		// Store the VICTIM's public key at the attacker's derived address.
		// This simulates the collision: the attacker's DeriveNodeAddress matches
		// an existing registry entry, but that entry belongs to the victim.
		PublicKey:    append(ed25519.PublicKey(nil), vPub...),
		Role:         RoleAccess,
		FrameAuthKey: hmac.DeriveKey([]byte(vPub), svtnID),
		NodeAddr:     attackerAddr,
		// revoked=false, admitted=false, expiry=zero — entry is valid pre-handshake.
	}
	ks.mu.Unlock()

	// Generate a fresh challenge from the router.
	ch, err := GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}

	// Attacker signs the nonce with THEIR OWN private key.
	attackerSig := ed25519.Sign(aPriv, ch.Nonce[:])
	resp := ChallengeResponse{NonceSig: attackerSig}

	// AdmitNode is called with the attacker's aPub.  DeriveNodeAddress(svtnID, aPub)
	// resolves to attackerAddr, landing on the installed entry (stored key = vPub).
	//
	// EXPECTED (fixed):   err = ErrSignatureVerificationFailed
	//   Rationale: Verify(vPub, nonce, attackerSig) = false — attacker's sig
	//   was made with aPriv, which does NOT verify under vPub.
	//
	// ACTUAL (buggy now): err = nil
	//   Rationale: Verify(aPub, nonce, attackerSig) = true — the frame-supplied
	//   key is the attacker's own key; the signature trivially verifies.
	//   This is the impersonation vector.
	err = AdmitNode(ch, resp, aPub, svtnID, ks)
	if !errors.Is(err, ErrSignatureVerificationFailed) {
		t.Errorf(
			"AdmitNode_VerifiesAgainstStoredKey: "+
				"want ErrSignatureVerificationFailed (BC-2.05.001 PC-3: stored key must be used); "+
				"got %v\n"+
				"  If this returns nil, the current code verifies against the frame-supplied\n"+
				"  pubKey (aPub) instead of the stored key (vPub), allowing the attacker's\n"+
				"  signature to trivially pass — the impersonation attack succeeds.",
			err,
		)
	}
}

// ── TestAdmitNode_StoredKeyMatches_Admits ──────────────────────────────────────

// TestAdmitNode_StoredKeyMatches_Admits is the positive companion to the
// discriminating test above.  It verifies that when the stored PublicKey equals
// the frame-supplied pubKey (no mismatch), and the caller signs correctly, AdmitNode
// returns nil — i.e. the fix does not over-reject legitimate admissions.
//
// Setup mirrors TestAdmitNode_VerifiesAgainstStoredKey_NotFramePubkey exactly,
// except the stored entry's PublicKey is aPub (not vPub).  The attacker signs
// with aPriv, so Verify(aPub, nonce, sig) = true.
//
// If the fix incorrectly rejects this case, this test fails — acting as a
// regression guard for the correction.
//
// NOT t.Parallel(): directly mutates the keyset's internal map.
//
// Traces to BC-2.05.001 PC-3 (successful verification path); F-1 companion.
func TestAdmitNode_StoredKeyMatches_Admits(t *testing.T) {
	var svtnID [16]byte
	svtnID[0] = 0xF1
	svtnID[1] = 0xA1 // distinct from the discriminating test's SVTN

	aPub, aPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey (attacker): %v", err)
	}
	_, routerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey (router): %v", err)
	}

	attackerAddr := frame.DeriveNodeAddress(svtnID, []byte(aPub))

	// Install the ATTACKER's PublicKey at attackerAddr — no mismatch.
	ks := NewAdmittedKeySet()
	ks.mu.Lock()
	if ks.keys[svtnID] == nil {
		ks.keys[svtnID] = make(map[[8]byte]*AdmittedKey)
	}
	ks.keys[svtnID][attackerAddr] = &AdmittedKey{
		PublicKey:    append(ed25519.PublicKey(nil), aPub...),
		Role:         RoleAccess,
		FrameAuthKey: hmac.DeriveKey([]byte(aPub), svtnID),
		NodeAddr:     attackerAddr,
	}
	ks.mu.Unlock()

	ch, err := GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}

	// Sign with the matching private key — legitimate admission.
	sig := ed25519.Sign(aPriv, ch.Nonce[:])
	resp := ChallengeResponse{NonceSig: sig}

	err = AdmitNode(ch, resp, aPub, svtnID, ks)
	if err != nil {
		t.Errorf(
			"AdmitNode_StoredKeyMatches_Admits: "+
				"want nil (legitimate admission); got %v\n"+
				"  The fix must not over-reject valid admissions where stored key == frame pubKey.",
			err,
		)
	}
}
