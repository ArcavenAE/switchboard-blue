// Package hmac provides HMAC-SHA256 frame authentication and HKDF-SHA256 key
// derivation for the Switchboard wire protocol (BC-2.05.005, ARCH-04 §HMAC keying).
//
// All three functions are pure-core: deterministic, no I/O, no side effects.
// The HMAC tag is always 8 bytes (64-bit truncation of the 32-byte HMAC-SHA256 output).
//
// Import constraints (ARCH-08 §boundary-violation-rules): this package MUST NOT
// import any other internal/ package. Only stdlib is permitted; no internal/
// imports and no external dependencies.
package hmac

import (
	ghmac "crypto/hmac"
	"crypto/sha256"
)

// TagSize is the byte length of the truncated HMAC-SHA256 tag written into the
// outer header hmac_tag field (ADR-001; ARCH-02 §HMAC tag).
const TagSize = 8

// KeySize is the byte length of the derived frame_auth_key produced by DeriveKey
// (ADR-001 amended: HKDF-SHA256 with length=32).
const KeySize = 32

// HKDFInfo is the fixed info string used in HKDF-SHA256 key derivation
// per ADR-001 (amended 2026-06-23).
const HKDFInfo = "switchboard-frame-auth"

// ComputeHMAC computes the 8-byte truncated HMAC-SHA256 tag for a frame.
//
// Per BC-2.05.005 postcondition 1 and ADR-001: the tag is the first TagSize
// bytes of the full 32-byte HMAC-SHA256 output, computed over frameBytes using
// key as the HMAC key. When called during frame construction, the outer header's
// hmac_tag bytes must be zeroed before passing the full frame as frameBytes.
//
// Returns a fixed-size [TagSize]byte array. Empty frameBytes is valid (EC-001).
func ComputeHMAC(key []byte, frameBytes []byte) [TagSize]byte {
	mac := ghmac.New(sha256.New, key)
	mac.Write(frameBytes)
	full := mac.Sum(nil)
	var tag [TagSize]byte
	copy(tag[:], full[:TagSize])
	return tag
}

// VerifyHMAC checks whether tag is the HMAC-SHA256 of frameBytes under key,
// truncated to the first TagSize (8) bytes.
//
// The comparison is constant-time (uses crypto/hmac.Equal) to prevent timing
// oracles (BC-2.05.005 postcondition 3). Returns true on exact match; false
// otherwise — including any single-bit perturbation of frameBytes (AC-003),
// tag, or key (AC-004), as verified by VP-005's fuzz harness.
//
// The [TagSize]byte signature enforces correct tag length at compile time;
// wrong-length tags are a type error, not a runtime case.
//
// VerifyHMAC is a pure function (no I/O, no side effects) suitable for use
// in router fast-path frame validation.
func VerifyHMAC(key []byte, frameBytes []byte, tag [TagSize]byte) bool {
	expected := ComputeHMAC(key, frameBytes)
	// hmac.Equal is constant-time over the full slice length, preventing
	// timing side-channels on tag comparison (BC-2.05.005 postcondition 3).
	return ghmac.Equal(expected[:], tag[:])
}

// hkdfSHA256 implements HKDF (RFC 5869) over SHA-256.
//
// Extract: PRK = HMAC-SHA256(salt, IKM)
// Expand: OKM = T(1) || T(2) || ... || T(N), where
//
//	T(0) = empty,
//	T(i) = HMAC-SHA256(PRK, T(i-1) || info || byte(i))
//
// Output: OKM truncated to length bytes.
//
// Supports L up to 255 * HMAC-SHA256 output size = 8160 bytes per RFC 5869 §2.3.
// Callers pass a non-nil salt (the empty case is the caller's responsibility:
// RFC 5869 §2.2 says salt is OPTIONAL and substitutes a zero-byte string of
// HMAC-SHA256 output length when nil — we do NOT auto-substitute here; callers
// must pass the salt they want).
func hkdfSHA256(ikm, salt, info []byte, length int) []byte {
	// Extract: PRK = HMAC-SHA256(salt, IKM)
	extractMAC := ghmac.New(sha256.New, salt)
	extractMAC.Write(ikm)
	prk := extractMAC.Sum(nil) // 32 bytes

	// Expand: build T(1), T(2), ..., until we have at least `length` bytes.
	// Each iteration appends one 32-byte block; for L=32 only T(1) is needed,
	// for L=42 (RFC 5869 §A.1) T(1)+T(2) are needed.
	var okm []byte
	var prev []byte
	for i := byte(1); len(okm) < length; i++ {
		expandMAC := ghmac.New(sha256.New, prk)
		expandMAC.Write(prev)
		expandMAC.Write(info)
		expandMAC.Write([]byte{i})
		prev = expandMAC.Sum(nil)
		okm = append(okm, prev...)
	}
	return okm[:length]
}

// DeriveKey derives a per-(node, SVTN) frame_auth_key via HKDF-SHA256.
//
// Per BC-2.05.005 precondition 2 and ADR-001 (amended 2026-06-23):
// nodeAdmissionPubkey is the IKM (input keying material); svtnID is encoded
// as a 16-byte salt; info is the constant HKDFInfo ("switchboard-frame-auth");
// output length is KeySize (32 bytes). The function is deterministic: the same
// inputs always produce the same output (AC-005). svtnID of all-zeros is
// accepted (EC-002).
//
// HKDF is implemented via hkdfSHA256 per RFC 5869 using stdlib crypto/hmac +
// crypto/sha256 (pure-core stdlib-only discipline).
func DeriveKey(nodeAdmissionPubkey []byte, svtnID [16]byte) [KeySize]byte {
	okm := hkdfSHA256(nodeAdmissionPubkey, svtnID[:], []byte(HKDFInfo), KeySize)
	var out [KeySize]byte
	copy(out[:], okm)
	return out
}
