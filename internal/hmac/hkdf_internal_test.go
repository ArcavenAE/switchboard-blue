// Package hmac internal tests — access unexported hkdfSHA256 helper.
//
// This file is an "internal" test file in the canonical Go pattern
// (package hmac rather than package hmac_test). It exists to KAT the
// unexported HKDF helper directly against RFC 5869 §A.1 vectors per
// story spec rev 2 / ARCH-04 v1.1 MANDATORY requirement.
package hmac

import (
	"bytes"
	"testing"
)

// TestDeriveKey_RFC5869_KAT verifies the inline hkdfSHA256 implementation
// against RFC 5869 §A.1 Test Case 1 (SHA-256, IKM = 22 × 0x0b, salt =
// 0x00..0x0c, info = 0xf0..0xf9, L = 42). This pins the algorithm against
// the canonical externally-validated ground truth per spec rev 2 +
// ARCH-04 v1.1 + drbothen/vsdd-factory#260 family resolution.
//
// Closes adversary pass-2 F-001 (HIGH): replaces the self-circular
// TestDeriveKey_RFC5869_DeterministicAnchor with a real KAT against
// RFC 5869 §A.1 ground truth.
func TestDeriveKey_RFC5869_KAT(t *testing.T) {
	t.Parallel()
	// RFC 5869 §A.1 Test Case 1 inputs
	ikm := bytes.Repeat([]byte{0x0b}, 22)
	salt := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}
	info := []byte{0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9}
	const L = 42

	// Expected OKM per RFC 5869 §A.1 (42 bytes).
	expected := []byte{
		0x3c, 0xb2, 0x5f, 0x25, 0xfa, 0xac, 0xd5, 0x7a,
		0x90, 0x43, 0x4f, 0x64, 0xd0, 0x36, 0x2f, 0x2a,
		0x2d, 0x2d, 0x0a, 0x90, 0xcf, 0x1a, 0x5a, 0x4c,
		0x5d, 0xb0, 0x2d, 0x56, 0xec, 0xc4, 0xc5, 0xbf,
		0x34, 0x00, 0x72, 0x08, 0xd5, 0xb8, 0x87, 0x18,
		0x58, 0x65,
	}

	got := hkdfSHA256(ikm, salt, info, L)
	if !bytes.Equal(got, expected) {
		t.Errorf("hkdfSHA256 RFC 5869 §A.1 mismatch:\n  got  = %x\n  want = %x", got, expected)
	}
}
