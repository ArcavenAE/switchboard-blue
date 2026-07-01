// Package admission_test — Red Gate tests for S-BL.LOOKUP signature migration.
//
// These tests assert that Lookup and LookupByPubkey return (AdmittedKey, bool)
// per go.md rule 12 (locked-accessor convention) and the DRIFT-F005-LOOKUP-CONVENTION
// tech-debt ruling. They will fail to compile until the migration lands, enforcing
// the Red Gate (BC-5.38.001).
package admission_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/frame"
)

// mustGenPub generates a fresh Ed25519 public key for lookup convention tests.
func mustGenPub(t *testing.T) ed25519.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	return pub
}

// svtnID returns a deterministic SVTN ID for lookup convention tests.
func svtnLookupID(b byte) [16]byte {
	var id [16]byte
	id[0] = b
	return id
}

// ── Lookup ────────────────────────────────────────────────────────────────────

// TestLookup_ReturnsBoolTrueOnHit asserts that Lookup returns (AdmittedKey, true)
// when the (svtnID, nodeAddr) tuple is registered, and that the returned value
// fields are byte-equal to the registered values.
//
// Compile failure before migration: Lookup currently returns *AdmittedKey (single
// value), so `key, ok := ks.Lookup(...)` will not compile until the signature is
// migrated to (AdmittedKey, bool). This is the Red Gate assertion for S-BL.LOOKUP.
func TestLookup_ReturnsBoolTrueOnHit(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x01)
	pub := mustGenPub(t)

	ks.RegisterKey(svtnID, pub, admission.RoleAccess)

	// Derive the node address using the same function production code uses.
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))
	key, ok := ks.Lookup(svtnID, nodeAddr) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if !ok {
		t.Fatal("Lookup: want ok=true for registered key; got false")
	}
	// F-P1L2-001: assert byte-equal PublicKey and exact NodeAddr — not just non-emptiness.
	// A bug returning arbitrary non-zero bytes would still pass a len!=0 check.
	if !bytes.Equal(key.PublicKey, []byte(pub)) {
		t.Errorf("Lookup: PublicKey mismatch: got %x, want %x", key.PublicKey, []byte(pub))
	}
	if key.NodeAddr != nodeAddr {
		t.Errorf("Lookup: NodeAddr mismatch: got %x, want %x", key.NodeAddr, nodeAddr)
	}
}

// TestLookup_ReturnsBoolFalseOnMiss asserts that Lookup returns (AdmittedKey{}, false)
// when the (svtnID, nodeAddr) tuple is not registered. The returned AdmittedKey must
// be the zero value — a non-zero AdmittedKey paired with false is a contract violation.
func TestLookup_ReturnsBoolFalseOnMiss(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x02)

	var absent [8]byte
	absent[0] = 0xFF

	key, ok := ks.Lookup(svtnID, absent) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if ok {
		t.Fatalf("Lookup: want ok=false for unregistered nodeAddr; got true with key=%+v", key)
	}
	// F-P1L2-002: on miss the returned AdmittedKey must be zero-valued.
	if len(key.PublicKey) != 0 || key.NodeAddr != ([8]byte{}) {
		t.Errorf("Lookup miss: returned non-zero AdmittedKey: %+v; want zero value", key)
	}
}

// ── LookupByPubkey ────────────────────────────────────────────────────────────

// TestLookupByPubkey_ReturnsBoolTrueOnHit asserts that LookupByPubkey returns
// (AdmittedKey, true) when the public key is registered for the given SVTN, and
// that the returned value fields are byte-equal to the registered values.
func TestLookupByPubkey_ReturnsBoolTrueOnHit(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x03)
	pub := mustGenPub(t)

	ks.RegisterKey(svtnID, pub, admission.RoleControl)

	expectedNodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pub))
	key, ok := ks.LookupByPubkey(svtnID, pub) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if !ok {
		t.Fatal("LookupByPubkey: want ok=true for registered key; got false")
	}
	// F-P1L2-001: assert byte-equal PublicKey and exact NodeAddr — not just non-emptiness.
	if !bytes.Equal(key.PublicKey, []byte(pub)) {
		t.Errorf("LookupByPubkey: PublicKey mismatch: got %x, want %x", key.PublicKey, []byte(pub))
	}
	if key.NodeAddr != expectedNodeAddr {
		t.Errorf("LookupByPubkey: NodeAddr mismatch: got %x, want %x", key.NodeAddr, expectedNodeAddr)
	}
}

// TestLookupByPubkey_ReturnsBoolFalseOnMiss asserts that LookupByPubkey returns
// (AdmittedKey{}, false) when the public key is not registered for the given SVTN.
// The returned AdmittedKey must be the zero value.
func TestLookupByPubkey_ReturnsBoolFalseOnMiss(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x04)
	pub := mustGenPub(t)

	// Do NOT register — key is absent.
	key, ok := ks.LookupByPubkey(svtnID, pub) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if ok {
		t.Fatalf("LookupByPubkey: want ok=false for unregistered key; got true with key=%+v", key)
	}
	// F-P1L2-002: on miss the returned AdmittedKey must be zero-valued.
	if len(key.PublicKey) != 0 || key.NodeAddr != ([8]byte{}) {
		t.Errorf("LookupByPubkey miss: returned non-zero AdmittedKey: %+v; want zero value", key)
	}
}
