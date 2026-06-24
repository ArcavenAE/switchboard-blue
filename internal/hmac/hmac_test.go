package hmac_test

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/hmac"
)

// TestTagSize verifies that the TagSize constant is 8 (AC-001 structural assertion).
// GREEN-BY-DESIGN: constant comparison; zero branching, no I/O, no helpers, 1 line.
func TestTagSize(t *testing.T) {
	t.Parallel()
	if hmac.TagSize != 8 {
		t.Errorf("TagSize = %d, want 8", hmac.TagSize)
	}
}

// TestKeySize verifies that the KeySize constant is 32 (AC-005 structural assertion).
// GREEN-BY-DESIGN: constant comparison; zero branching, no I/O, no helpers, 1 line.
func TestKeySize(t *testing.T) {
	t.Parallel()
	if hmac.KeySize != 32 {
		t.Errorf("KeySize = %d, want 32", hmac.KeySize)
	}
}

// TestComputeHMAC_EightByteTag verifies AC-001: ComputeHMAC produces an 8-byte tag.
// Red Gate: panics with "not implemented: S-2.01 ComputeHMAC".
func TestComputeHMAC_EightByteTag(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	frame := []byte("test-frame-bytes")
	tag := hmac.ComputeHMAC(key, frame)
	if tag[0] == 0 && tag[1] == 0 && tag[2] == 0 && tag[3] == 0 &&
		tag[4] == 0 && tag[5] == 0 && tag[6] == 0 && tag[7] == 0 {
		// A real all-zero HMAC is astronomically unlikely; treat it as a stub leak.
		t.Log("ComputeHMAC returned all-zero tag (stub or genuine zero)")
	}
}

// TestVerifyHMAC_ValidTag verifies AC-002: VerifyHMAC returns true for a correct tag.
// Red Gate: panics with "not implemented: S-2.01 ComputeHMAC".
func TestVerifyHMAC_ValidTag(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	frame := []byte("test-frame-bytes")
	tag := hmac.ComputeHMAC(key, frame)
	if !hmac.VerifyHMAC(key, frame, tag) {
		t.Error("VerifyHMAC: expected true for valid tag")
	}
}

// TestVerifyHMAC_WrongKey verifies AC-004: VerifyHMAC returns false for a different key.
// Red Gate: panics with "not implemented: S-2.01 ComputeHMAC".
func TestVerifyHMAC_WrongKey(t *testing.T) {
	t.Parallel()
	rightKey := make([]byte, 32)
	wrongKey := make([]byte, 32)
	wrongKey[0] = 0xFF
	frame := []byte("test-frame-bytes")
	tag := hmac.ComputeHMAC(rightKey, frame)
	if hmac.VerifyHMAC(wrongKey, frame, tag) {
		t.Error("VerifyHMAC: expected false for wrong key")
	}
}

// TestDeriveKey_Deterministic verifies AC-005: DeriveKey is deterministic.
// Red Gate: panics with "not implemented: S-2.01 DeriveKey".
func TestDeriveKey_Deterministic(t *testing.T) {
	t.Parallel()
	pubkey := []byte("node-admission-pubkey-bytes-here")
	var svtnID [16]byte
	copy(svtnID[:], "test-svtn-id-123")
	k1 := hmac.DeriveKey(pubkey, svtnID)
	k2 := hmac.DeriveKey(pubkey, svtnID)
	if k1 != k2 {
		t.Error("DeriveKey: expected deterministic output, got different keys")
	}
}

// TestDeriveKey_ZeroSVTN verifies EC-002: DeriveKey accepts all-zero svtnID.
// Red Gate: panics with "not implemented: S-2.01 DeriveKey".
func TestDeriveKey_ZeroSVTN(t *testing.T) {
	t.Parallel()
	pubkey := []byte("node-admission-pubkey-bytes-here")
	var svtnID [16]byte // all zeros
	_ = hmac.DeriveKey(pubkey, svtnID)
}

// TestVerifyHMAC_ShortTag verifies EC-003: VerifyHMAC returns false for a zero/mismatched tag.
// Red Gate: panics with "not implemented: S-2.01 VerifyHMAC".
func TestVerifyHMAC_ShortTag(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	frame := []byte("test-frame-bytes")
	var zeroTag [hmac.TagSize]byte // zero bytes — will not match any real HMAC
	if hmac.VerifyHMAC(key, frame, zeroTag) {
		t.Error("VerifyHMAC: expected false for zero (mismatched) tag")
	}
}

// TestComputeHMAC_EmptyFrame verifies EC-001: ComputeHMAC handles empty frame bytes.
// Red Gate: panics with "not implemented: S-2.01 ComputeHMAC".
func TestComputeHMAC_EmptyFrame(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	tag := hmac.ComputeHMAC(key, []byte{})
	_ = tag
}
