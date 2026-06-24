package frame

import "crypto/sha256"

// DeriveNodeAddress computes the 8-byte node address from a SVTN identifier
// and an Ed25519 public key using SHA-256(svtnID || publicKey), then
// truncating to 8 bytes. Deterministic: identical inputs always yield the
// same address (BC-2.01.006 postcondition 1).
//
// Uses crypto/sha256 per ARCH-INDEX changelog resolution F-007 (not Blake3).
func DeriveNodeAddress(svtnID [16]byte, publicKey []byte) [8]byte {
	h := sha256.New()
	h.Write(svtnID[:])
	h.Write(publicKey)
	sum := h.Sum(nil)
	var addr [8]byte
	copy(addr[:], sum[:8])
	return addr
}
