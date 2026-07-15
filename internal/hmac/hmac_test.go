package hmac_test

import (
	"bytes"
	crypto_hmac "crypto/hmac"
	"crypto/sha256"
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
func TestComputeHMAC_EightByteTag(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	frame := []byte("test-frame-bytes")
	tag := hmac.ComputeHMAC(key, frame)
	if tag[0] == 0 && tag[1] == 0 && tag[2] == 0 && tag[3] == 0 &&
		tag[4] == 0 && tag[5] == 0 && tag[6] == 0 && tag[7] == 0 {
		// A real all-zero HMAC is astronomically unlikely; treat it as a stub leak.
		t.Fatalf("ComputeHMAC returned all-zero tag (stub or genuine zero)")
	}
}

// TestComputeHMAC_KnownAnswerVector pins ComputeHMAC against RFC 4231 §4.2 test case 2:
// key = "Jefe" (4 bytes), data = "what do ya want for nothing?"
// Expected HMAC-SHA256 full: 5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843
// First 8 bytes (TagSize): 5bdcc146bf60754e
//
// Exercises VP-004 (compute/verify consistency) at the ground-truth level: if truncation
// or algorithm deviates, this test fails before consistency tests even run.
func TestComputeHMAC_KnownAnswerVector(t *testing.T) {
	t.Parallel()
	key := []byte("Jefe")
	data := []byte("what do ya want for nothing?")
	expected := [hmac.TagSize]byte{0x5b, 0xdc, 0xc1, 0x46, 0xbf, 0x60, 0x75, 0x4e}
	got := hmac.ComputeHMAC(key, data)
	if got != expected {
		t.Errorf("ComputeHMAC RFC 4231 §4.2 vector mismatch: got %x, want %x", got, expected)
	}
}

// TestVerifyHMAC_ValidTag verifies AC-002: VerifyHMAC returns true for a correct tag.
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
func TestDeriveKey_ZeroSVTN(t *testing.T) {
	t.Parallel()
	pubkey := []byte("node-admission-pubkey-bytes-here")
	var svtnID [16]byte // all zeros
	_ = hmac.DeriveKey(pubkey, svtnID)
}

// TestVerifyHMAC_ZeroTagRejected verifies EC-003: VerifyHMAC returns false for a
// zero/mismatched tag. (Previously TestVerifyHMAC_ShortTag — renamed because a
// "short" tag is impossible given the [TagSize]byte signature; "zero tag" is the
// actual edge case being exercised.)
func TestVerifyHMAC_ZeroTagRejected(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)
	frame := []byte("test-frame-bytes")
	var zeroTag [hmac.TagSize]byte // zero bytes — will not match any real HMAC
	if hmac.VerifyHMAC(key, frame, zeroTag) {
		t.Error("VerifyHMAC: expected false for zero (mismatched) tag")
	}
}

// TestComputeHMAC_EmptyFrame verifies EC-001: ComputeHMAC handles empty frame bytes
// and produces the correct HMAC-SHA256 truncation.
// Expected: HMAC-SHA256(zero-32-byte-key, "") first 8 bytes = b613679a0814d9ec
func TestComputeHMAC_EmptyFrame(t *testing.T) {
	t.Parallel()
	key := make([]byte, 32)

	// Compute expected value using crypto/hmac directly to pin the result.
	h := crypto_hmac.New(sha256.New, key)
	// Write nothing — empty frame.
	var expected [hmac.TagSize]byte
	copy(expected[:], h.Sum(nil)[:hmac.TagSize])

	got := hmac.ComputeHMAC(key, []byte{})
	if got != expected {
		t.Errorf("ComputeHMAC empty-frame: got %x, want %x", got, expected)
	}
}

// TestPropComputeVerifyConsistency verifies VP-004: for any (key, frame_bytes) pair,
// ComputeHMAC produces a tag that VerifyHMAC accepts with the same key and rejects
// with a different key.
//
// VP-006 (wrong-key rejection) is covered in the cross-check sub-step of each case
// below — each case asserts both same-key acceptance and different-key rejection.
// The property is exercised across varied input sizes and key patterns; a random
// property engine is not needed because HMAC-SHA256 is deterministic and the
// correctness property holds by construction for all lengths, not just a sampled
// subset.
//
// Covers: VP-004 (consistency), VP-006 (wrong-key rejection).
func TestPropComputeVerifyConsistency(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		keyLen   int
		frameLen int
	}{
		{"32B key, 0B frame", 32, 0},
		{"32B key, 1B frame", 32, 1},
		{"32B key, 44B frame (outer-header sized)", 32, 44},
		{"32B key, 1024B frame", 32, 1024},
		{"32B key, 65515B frame (MaxPayloadSize)", 32, 65515},
		{"16B key, 1024B frame (short key edge)", 16, 1024},
		{"1B key, 1B frame (minimum key)", 1, 1},
		{"64B key, 512B frame (long key)", 64, 512},
		{"32B key, 3B frame (sub-block)", 32, 3},
		{"32B key, 64B frame (SHA256 block boundary)", 32, 64},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key := bytes.Repeat([]byte{0xAB}, tc.keyLen)
			frame := bytes.Repeat([]byte{0xCD}, tc.frameLen)

			tag := hmac.ComputeHMAC(key, frame)

			// VP-004: same key must accept.
			if !hmac.VerifyHMAC(key, frame, tag) {
				t.Errorf("VP-004: VerifyHMAC rejected its own ComputeHMAC tag: keyLen=%d frameLen=%d", tc.keyLen, tc.frameLen)
			}

			// VP-006: different key must reject.
			otherKey := bytes.Repeat([]byte{0xEF}, tc.keyLen)
			if hmac.VerifyHMAC(otherKey, frame, tag) {
				t.Errorf("VP-006: VerifyHMAC accepted tag with wrong key: keyLen=%d frameLen=%d", tc.keyLen, tc.frameLen)
			}
		})
	}
}

// TestDeriveKey_DistinctPubkeysProduceDistinctKeys asserts the per-(node, SVTN)
// forge-resistance invariant from ARCH-04 lines 175-180: different
// nodeAdmissionPubkey inputs MUST yield different derived keys (otherwise a
// constant-return implementation would pass TestDeriveKey_Deterministic).
func TestDeriveKey_DistinctPubkeysProduceDistinctKeys(t *testing.T) {
	t.Parallel()
	svtnID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	pubkeyA := bytes.Repeat([]byte{0xAA}, 32)
	pubkeyB := bytes.Repeat([]byte{0xBB}, 32)
	keyA := hmac.DeriveKey(pubkeyA, svtnID)
	keyB := hmac.DeriveKey(pubkeyB, svtnID)
	if keyA == keyB {
		t.Error("DeriveKey produced identical output for distinct pubkeys (forge-resistance violated)")
	}
}

// TestDeriveKey_DistinctSVTNsProduceDistinctKeys verifies the same forge-resistance
// invariant with the SVTN salt axis: same pubkey, different svtnID MUST yield different
// derived keys (salt mixing required by RFC 5869 §3.1).
func TestDeriveKey_DistinctSVTNsProduceDistinctKeys(t *testing.T) {
	t.Parallel()
	pubkey := bytes.Repeat([]byte{0xCD}, 32)
	svtnA := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	svtnB := [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	keyA := hmac.DeriveKey(pubkey, svtnA)
	keyB := hmac.DeriveKey(pubkey, svtnB)
	if keyA == keyB {
		t.Error("DeriveKey produced identical output for distinct SVTNs (salt mixing violated)")
	}
}

// ---------------------------------------------------------------------------
// S-BL.DISCOVERY-WIRE AC-004 — DeriveDiscoveryKey domain separation
// ---------------------------------------------------------------------------

// redGateGuard recovers from a not-yet-implemented stub's panic and fails the
// test cleanly (Red Gate discipline, BC-5.38.001) instead of crashing the
// whole test binary. Once the relevant Task's Green step lands, the panic
// disappears and this guard becomes a silent no-op — the assertions after
// `defer redGateGuard(t)` then run for real, with no test-file change
// required.
func redGateGuard(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("red gate: stub not yet implemented: %v", r)
	}
}

// TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey verifies AC-004
// postconditions 1 and 2: for the same (nodeAdmissionPubkey, svtnID) pair,
// DeriveDiscoveryKey's output differs from DeriveKey's output — the two
// derived keys are cryptographically independent (SEC-DW-06).
func TestDeriveDiscoveryKey_DomainSeparatedFromFrameAuthKey(t *testing.T) {
	t.Parallel()
	defer redGateGuard(t)

	pubkey := bytes.Repeat([]byte{0x42}, 32)
	var svtnID [16]byte
	copy(svtnID[:], "discovery-svtn-1")

	frameKey := hmac.DeriveKey(pubkey, svtnID)
	discoveryKey := hmac.DeriveDiscoveryKey(pubkey, svtnID)

	if frameKey == discoveryKey {
		t.Error("DeriveDiscoveryKey produced the same output as DeriveKey for the same (pubkey, svtnID) — domain separation (SEC-DW-06) violated")
	}

	// Determinism: DeriveDiscoveryKey must be a pure function of its inputs,
	// same as DeriveKey (AC-004 postcondition 1 implies determinism —
	// mirrors TestDeriveKey_Deterministic's oracle for the new function).
	again := hmac.DeriveDiscoveryKey(pubkey, svtnID)
	if discoveryKey != again {
		t.Error("DeriveDiscoveryKey: expected deterministic output, got different keys across calls")
	}
}
