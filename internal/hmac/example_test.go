package hmac_test

import (
	"fmt"

	"github.com/arcavenae/switchboard/internal/hmac"
)

// ExampleComputeHMAC demonstrates computing an 8-byte HMAC-SHA256 tag.
// In production, the key is derived per-(node, SVTN) via DeriveKey.
// Here a fixed key is used so the // Output: block is deterministic.
func ExampleComputeHMAC() {
	key := []byte("deterministic-example-key-32-byt") // exactly 32 bytes
	frame := []byte("example-frame-bytes")
	tag := hmac.ComputeHMAC(key, frame)
	fmt.Printf("tag (8 bytes): %x\n", tag)

	// Output:
	// tag (8 bytes): 4cf51cf4dfabcb89
}

// ExampleVerifyHMAC demonstrates constant-time tag verification.
// A valid tag verifies true; a single-bit-flipped tag verifies false.
func ExampleVerifyHMAC() {
	key := []byte("deterministic-example-key-32-byt") // exactly 32 bytes
	frame := []byte("example-frame-bytes")
	tag := hmac.ComputeHMAC(key, frame)

	// Valid tag verifies true.
	fmt.Println("valid tag:", hmac.VerifyHMAC(key, frame, tag))

	// Tampered tag (flip one bit) verifies false.
	tag[0] ^= 0x01
	fmt.Println("tampered tag:", hmac.VerifyHMAC(key, frame, tag))

	// Output:
	// valid tag: true
	// tampered tag: false
}
