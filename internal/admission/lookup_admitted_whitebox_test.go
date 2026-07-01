// Package admission — white-box tests for the admitted bit on Lookup return value.
//
// These tests are in package admission (not admission_test) to access the
// unexported admitted field on AdmittedKey directly, verifying that Lookup
// reflects the internal admitted state without requiring the IsAdmitted
// observable (which AND-gates on the store, not the returned value).
//
// F-L2-A1: TestLookup_Hit_AdmittedFieldTrackedAfterAdmit.
package admission

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
)

// TestLookup_Hit_AdmittedFieldTrackedAfterAdmit asserts that the admitted field
// on the AdmittedKey returned by Lookup reflects the internal admitted state:
// false before AdmitNode, true after.
//
// The external IsAdmitted observable AND-gates on the store state, not the
// returned value — this white-box test verifies the field directly so a
// regression where Lookup returns a zeroed copy (admitted=false even after
// AdmitNode) is caught at the source, not just via observable side-effects.
//
// F-L2-A1: admitted-field tracking in Lookup return value.
func TestLookup_Hit_AdmittedFieldTrackedAfterAdmit(t *testing.T) {
	t.Parallel()

	var svtnID [16]byte
	svtnID[0] = 0xAB

	// Generate a key pair so we can produce a valid challenge-response.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}

	// Generate a router key pair for GenerateChallenge.
	_, routerPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey (router): %v", err)
	}

	s := NewAdmittedKeySet()
	s.RegisterKey(svtnID, pub, RoleAccess)

	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))

	// Pre-admit: Lookup return value must have admitted=false.
	preKey, ok := s.Lookup(svtnID, nodeAddr)
	if !ok {
		t.Fatal("pre-admit Lookup: want ok=true; got false")
	}
	if preKey.admitted {
		t.Errorf("pre-admit: key.admitted got true; want false (RegisterKey does not admit)")
	}

	// Issue a challenge and have the node sign the nonce.
	ch, err := GenerateChallenge(routerPriv)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	sig := ed25519.Sign(priv, ch.Nonce[:])
	resp := ChallengeResponse{NonceSig: sig}

	if err := AdmitNode(ch, resp, pub, svtnID, s); err != nil {
		t.Fatalf("AdmitNode: %v", err)
	}

	// Post-admit: Lookup return value must have admitted=true.
	postKey, ok := s.Lookup(svtnID, nodeAddr)
	if !ok {
		t.Fatal("post-admit Lookup: want ok=true; got false")
	}
	if !postKey.admitted {
		t.Errorf("post-admit: key.admitted got false; want true after successful AdmitNode")
	}
}
