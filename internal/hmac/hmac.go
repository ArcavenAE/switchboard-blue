// Package hmac provides HMAC-SHA256 frame authentication and HKDF-SHA256 key
// derivation for the Switchboard wire protocol (BC-2.05.005, ARCH-04 §HMAC keying).
//
// All three functions are pure-core: deterministic, no I/O, no side effects.
// The HMAC tag is always 8 bytes (64-bit truncation of the 32-byte HMAC-SHA256 output).
//
// Import constraints (ARCH-08 §boundary-violation-rules): this package MUST NOT
// import any other internal/ package. Only stdlib and golang.org/x/crypto are permitted.
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

// VerifyHMAC authenticates frameBytes against tag using key.
//
// Per BC-2.05.005 postconditions 2–4: returns true if and only if the tag
// matches the first TagSize bytes of HMAC-SHA256(key, frameBytes). Returns
// false for any single-bit flip in frameBytes (AC-003), for a wrong key
// (AC-004), and for a tag slice shorter than TagSize bytes (EC-003) — without
// panicking.
func VerifyHMAC(key []byte, frameBytes []byte, tag [TagSize]byte) bool {
	expected := ComputeHMAC(key, frameBytes)
	// hmac.Equal is constant-time over the full slice length, preventing
	// timing side-channels on tag comparison (BC-2.05.005 postcondition 3).
	return ghmac.Equal(expected[:], tag[:])
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
// HKDF is implemented inline per RFC 5869 using stdlib crypto/hmac + crypto/sha256
// (golang.org/x/crypto/hkdf is not vendored; pure-core stdlib-only discipline).
// For L=32 bytes only one expand iteration is needed (T(1) == OKM).
func DeriveKey(nodeAdmissionPubkey []byte, svtnID [16]byte) [KeySize]byte {
	// HKDF-Extract: PRK = HMAC-SHA256(salt=svtnID, IKM=nodeAdmissionPubkey)
	extractor := ghmac.New(sha256.New, svtnID[:])
	extractor.Write(nodeAdmissionPubkey)
	prk := extractor.Sum(nil)

	// HKDF-Expand: T(1) = HMAC-SHA256(PRK, info || 0x01)
	// For L=32, T(1) is the entire OKM (one iteration suffices).
	expander := ghmac.New(sha256.New, prk)
	expander.Write([]byte(HKDFInfo))
	expander.Write([]byte{0x01})
	okm := expander.Sum(nil)

	var key [KeySize]byte
	copy(key[:], okm[:KeySize])
	return key
}
