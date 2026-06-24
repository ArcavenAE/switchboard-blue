package hmac_test

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/hmac"
)

// FuzzVerifyHMAC_SingleBitFlip is the VP-005 fuzz target verifying AC-003:
// VerifyHMAC returns false for any single-bit flip in the frame payload.
//
// Red Gate: will panic with "not implemented: S-2.01 ComputeHMAC" on first call.
func FuzzVerifyHMAC_SingleBitFlip(f *testing.F) {
	// Seed corpus: a small valid frame.
	f.Add([]byte("test-frame-bytes-for-fuzz"))

	f.Fuzz(func(t *testing.T, frameBytes []byte) {
		if len(frameBytes) == 0 {
			return
		}
		key := make([]byte, 32)
		tag := hmac.ComputeHMAC(key, frameBytes)

		// Flip one bit in a copy of the frame and verify it no longer passes.
		flipped := make([]byte, len(frameBytes))
		copy(flipped, frameBytes)
		flipped[0] ^= 0x01

		if hmac.VerifyHMAC(key, flipped, tag) {
			t.Errorf("VerifyHMAC: expected false after single-bit flip, got true for frame len=%d", len(frameBytes))
		}
	})
}
