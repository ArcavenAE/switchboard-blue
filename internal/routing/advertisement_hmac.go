package routing

import (
	"github.com/arcavenae/switchboard/internal/hmac"
)

// AdvertisementHMACTagSize is the byte length of the HMAC tag prepended to
// every encoded advertisement payload (AC-005; BC-2.03.001 PC-5).
const AdvertisementHMACTagSize = hmac.TagSize

// ComputeAdvertisementHMAC computes the 8-byte HMAC-SHA256 tag for a
// discovery advertisement message using key as the HMAC key.
//
// Exposed so that internal/discovery can authenticate advertisement frames
// via the routing package HMAC surface without importing internal/hmac
// directly (ARCH-08 §6.5 position 14: discovery→routing legal;
// discovery→hmac forbidden).
func ComputeAdvertisementHMAC(key []byte, msg []byte) [hmac.TagSize]byte {
	return hmac.ComputeHMAC(key, msg)
}

// VerifyAdvertisementHMAC checks whether tag is the correct HMAC-SHA256 tag
// for msg under key. The comparison is constant-time (delegates to
// hmac.VerifyHMAC which uses crypto/hmac.Equal).
//
// Returns true only on an exact match — fail-closed (AC-005).
func VerifyAdvertisementHMAC(key []byte, msg []byte, tag [hmac.TagSize]byte) bool {
	return hmac.VerifyHMAC(key, msg, tag)
}
