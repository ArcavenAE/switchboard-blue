package hmac_test

import (
	"testing"

	"github.com/arcavenae/switchboard/internal/hmac"
)

// FuzzVerifyHMAC_SingleBitFlip covers AC-003: VerifyHMAC returns false for any
// single-bit flip in the frame payload (not the tag). This is the original VP-005
// fuzz target; retained per user decision (2026-06-24) for dual coverage alongside
// FuzzVerifyHMAC_TagBitFlip.
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

// FuzzVerifyHMAC_TagBitFlip verifies VP-005's canonical property: a single-bit flip
// in the tag at ANY of the 64 bit positions (8 bytes × 8 bits) causes VerifyHMAC to
// return false. This is the tag-forgery-resistance semantic — flipping one bit of a
// valid tag must never produce another valid tag.
//
// tag is [TagSize]byte (array type). Each `flipped := tag` assignment copies all 8
// bytes — no shared backing array — so per-iteration mutation is isolated.
func FuzzVerifyHMAC_TagBitFlip(f *testing.F) {
	// Seed corpus: deterministic (key, frame) pairs covering common shapes.
	f.Add([]byte("deterministic-test-key-32-bytes!"), []byte("seed-frame-payload"))
	f.Add([]byte{0xFF, 0x00, 0xAB, 0xCD}, []byte("short-key-seed"))
	f.Add([]byte("test-admission-key-32-bytes-long!"), []byte{})

	f.Fuzz(func(t *testing.T, key, frame []byte) {
		if len(key) == 0 {
			return
		}
		tag := hmac.ComputeHMAC(key, frame)

		// Verify baseline: the correct tag must pass.
		if !hmac.VerifyHMAC(key, frame, tag) {
			t.Fatalf("VP-005 baseline: VerifyHMAC rejected its own ComputeHMAC tag")
		}

		// Flip every bit position across the 8-byte tag; all flipped tags must reject.
		// tag is [TagSize]byte — assignment copies the array, so each iteration
		// starts from the original valid tag with no aliasing.
		for byteIdx := 0; byteIdx < hmac.TagSize; byteIdx++ {
			for bitIdx := 0; bitIdx < 8; bitIdx++ {
				flipped := tag                        // array copy — no shared backing
				flipped[byteIdx] ^= 1 << uint(bitIdx) // mutates the copy only
				if hmac.VerifyHMAC(key, frame, flipped) {
					t.Errorf(
						"VP-005: VerifyHMAC accepted flipped tag at byte=%d bit=%d key=%x frame=%x",
						byteIdx, bitIdx, key, frame,
					)
				}
			}
		}
	})
}
