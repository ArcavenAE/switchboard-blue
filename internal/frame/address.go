package frame

// DeriveNodeAddress computes the 8-byte node address from a SVTN identifier
// and an Ed25519 public key using SHA-256(svtnID || publicKey), then
// truncating to 8 bytes. Deterministic: identical inputs always yield the
// same address (BC-2.01.006 postcondition 1).
//
// S-1.01 stub — implementation is Step 8 of the delivery tasks.
// Note: the implementation MUST use crypto/sha256 (not Blake3) per
// ARCH-INDEX changelog resolution F-007.
func DeriveNodeAddress(svtnID [16]byte, publicKey []byte) [8]byte {
	panic("not implemented: S-1.01 DeriveNodeAddress")
}
