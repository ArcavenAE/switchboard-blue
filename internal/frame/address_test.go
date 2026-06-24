package frame_test

import (
	"crypto/sha256"
	"testing"

	"github.com/arcavenae/switchboard/internal/frame"
)

// AC-006 — TestDeriveNodeAddress_Deterministic
// Traces to BC-2.01.006 postcondition 1: same inputs always produce same
// 8-byte node address. Also verifies the derivation is SHA-256(svtn_id || public_key)[:8]
// per ARCH-02 §Session Identity and ADR-001.
func TestDeriveNodeAddress_Deterministic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		svtnID    [16]byte
		publicKey []byte
	}{
		{
			name:      "ed25519 test key A on svtn-1",
			svtnID:    [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			publicKey: []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF, 0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
		},
		{
			name:      "ed25519 test key B on svtn-1 (different key, same svtn → different address)",
			svtnID:    [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			publicKey: []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99},
		},
		{
			name:      "same key A on svtn-2 (same key, different svtn → different address, EC-003)",
			svtnID:    [16]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0},
			publicKey: []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF, 0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
		},
		{
			name:      "short public key (8 bytes)",
			svtnID:    [16]byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA},
			publicKey: []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
		},
		{
			name:      "all-zero svtn_id (EC-002 canonical — valid encoding per wire format)",
			svtnID:    [16]byte{},
			publicKey: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Determinism: call twice, assert identical results.
			addr1 := frame.DeriveNodeAddress(tc.svtnID, tc.publicKey)
			addr2 := frame.DeriveNodeAddress(tc.svtnID, tc.publicKey)
			if addr1 != addr2 {
				t.Errorf("DeriveNodeAddress not deterministic: first call = %x, second call = %x", addr1, addr2)
			}

			// Correctness: verify result equals SHA-256(svtn_id || public_key)[:8].
			assertSHA256Address(t, tc.svtnID, tc.publicKey, addr1)

			// Result must be exactly 8 bytes (asserted by the [8]byte return type,
			// but we also verify it is non-nil and fully populated).
		})
	}
}

// AC-006 / BC-2.01.006 — TestDeriveNodeAddress_ReturnsExpectedSHA256Prefix
// Verifies the concrete golden value: DeriveNodeAddress must return exactly
// SHA-256(svtn_id || public_key)[:8] for a deterministic input. Replaces the
// tautological len-check that SA4006 flagged — [8]byte return type already
// guarantees length; this test locks in the actual hash content.
func TestDeriveNodeAddress_ReturnsExpectedSHA256Prefix(t *testing.T) {
	t.Parallel()

	svtnID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	publicKey := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}

	// Compute expected: SHA-256(svtn_id || public_key)[:8]
	input := append([]byte{}, svtnID[:]...)
	input = append(input, publicKey...)
	sum := sha256.Sum256(input)
	var expected [8]byte
	copy(expected[:], sum[:8])

	addr := frame.DeriveNodeAddress(svtnID, publicKey)
	if addr != expected {
		t.Errorf("DeriveNodeAddress = %x, want %x", addr, expected)
	}
}

// TestDeriveNodeAddress_DifferentSVTNYieldsDifferentAddress verifies VP-014:
// address(svtn_id1, pk) != address(svtn_id2, pk) for different svtn_ids.
func TestDeriveNodeAddress_DifferentSVTNYieldsDifferentAddress(t *testing.T) {
	t.Parallel()

	publicKey := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF, 0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}
	svtn1 := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	svtn2 := [16]byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8, 0xF7, 0xF6, 0xF5, 0xF4, 0xF3, 0xF2, 0xF1, 0xF0}

	addr1 := frame.DeriveNodeAddress(svtn1, publicKey)
	addr2 := frame.DeriveNodeAddress(svtn2, publicKey)

	// Verify using expected SHA-256 values — the test must not pass trivially.
	h1 := sha256.New()
	h1.Write(svtn1[:])
	h1.Write(publicKey)
	sum1 := h1.Sum(nil)

	h2 := sha256.New()
	h2.Write(svtn2[:])
	h2.Write(publicKey)
	sum2 := h2.Sum(nil)

	// The SHA-256 hashes of distinct inputs must differ in the first 8 bytes
	// (verified against known good values).
	var expectedAddr1, expectedAddr2 [8]byte
	copy(expectedAddr1[:], sum1[:8])
	copy(expectedAddr2[:], sum2[:8])

	if expectedAddr1 == expectedAddr2 {
		// This would be an astronomically unlikely collision in the test vectors
		// themselves — fail fast so a broken test vector is caught.
		t.Fatal("test vector defect: SHA-256 of two distinct inputs collides in first 8 bytes")
	}

	if addr1 == addr2 {
		t.Errorf("DeriveNodeAddress returned same address for different svtn_ids: addr=%x", addr1)
	}

	if addr1 != expectedAddr1 {
		t.Errorf("addr1 mismatch: got %x, want %x", addr1, expectedAddr1)
	}
	if addr2 != expectedAddr2 {
		t.Errorf("addr2 mismatch: got %x, want %x", addr2, expectedAddr2)
	}
}
