// Package admission_test — Red Gate tests for S-BL.LOOKUP signature migration.
//
// These tests assert that Lookup and LookupByPubkey return (AdmittedKey, bool)
// per go.md rule 12 (locked-accessor convention) and the DRIFT-F005-LOOKUP-CONVENTION
// tech-debt ruling. They will fail to compile until the migration lands, enforcing
// the Red Gate (BC-5.38.001).
package admission_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
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
// when the (svtnID, nodeAddr) tuple is registered.
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

	// Derive the node address the same way RegisterKey does — via frame.DeriveNodeAddress.
	// We look up by pubkey to avoid importing internal/frame directly.
	// Use LookupByPubkey to resolve nodeAddr, then verify Lookup via that addr.
	// The compile-time assertion is: Lookup must return two values (AdmittedKey, bool).
	nodeAddr := deriveAddrForTest(t, svtnID, pub)
	key, ok := ks.Lookup(svtnID, nodeAddr) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if !ok {
		t.Fatal("Lookup: want ok=true for registered key; got false")
	}
	if len(key.PublicKey) == 0 {
		t.Error("Lookup: returned AdmittedKey has empty PublicKey")
	}
}

// TestLookup_ReturnsBoolFalseOnMiss asserts that Lookup returns (AdmittedKey{}, false)
// when the (svtnID, nodeAddr) tuple is not registered.
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
}

// ── LookupByPubkey ────────────────────────────────────────────────────────────

// TestLookupByPubkey_ReturnsBoolTrueOnHit asserts that LookupByPubkey returns
// (AdmittedKey, true) when the public key is registered for the given SVTN.
func TestLookupByPubkey_ReturnsBoolTrueOnHit(t *testing.T) {
	t.Parallel()

	ks := admission.NewAdmittedKeySet()
	svtnID := svtnLookupID(0x03)
	pub := mustGenPub(t)

	ks.RegisterKey(svtnID, pub, admission.RoleControl)

	key, ok := ks.LookupByPubkey(svtnID, pub) // S-BL.LOOKUP: compile fails until (AdmittedKey, bool)
	if !ok {
		t.Fatal("LookupByPubkey: want ok=true for registered key; got false")
	}
	if len(key.PublicKey) == 0 {
		t.Error("LookupByPubkey: returned AdmittedKey has empty PublicKey")
	}
}

// TestLookupByPubkey_ReturnsBoolFalseOnMiss asserts that LookupByPubkey returns
// (AdmittedKey{}, false) when the public key is not registered for the given SVTN.
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
}

// ── compile-time helpers ──────────────────────────────────────────────────────

// deriveAddrForTest derives the node address via the package's own lookup path
// rather than importing internal/frame. We call LookupByPubkey (which internally
// derives the address) and, once the migration lands, use the returned value's
// NodeAddr field to feed back into Lookup.
//
// Before migration, LookupByPubkey still returns *AdmittedKey so this helper
// itself will also fail to compile — reinforcing the Red Gate.
func deriveAddrForTest(t *testing.T, svtnID [16]byte, pub ed25519.PublicKey) [8]byte {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	ks.RegisterKey(svtnID, pub, admission.RoleAccess)
	got, ok := ks.LookupByPubkey(svtnID, pub) // compile fails until (AdmittedKey, bool)
	if !ok {
		t.Fatal("deriveAddrForTest: LookupByPubkey returned ok=false after RegisterKey")
	}
	return got.NodeAddr
}
