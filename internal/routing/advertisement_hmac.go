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

// DiscoveryAuthKeyFor returns the admitted-node discovery_auth_key for
// (svtnID, nodeAddr), derived on demand from AdmittedKeySet.Lookup's
// PublicKey field via hmac.DeriveDiscoveryKey — never cached as a new
// AdmittedKey field (S-BL.DISCOVERY-WIRE Decision 1 / Ruling 1 Implementation
// Constraint 3).
//
// Returns (key, true) when the (svtnID, nodeAddr) pair is admitted; (zero,
// false) otherwise — a thin, read-only wrapper adding no new mutable state
// to Router (AC-004 postcondition 3).
func (r *Router) DiscoveryAuthKeyFor(svtnID [16]byte, nodeAddr [8]byte) ([hmac.KeySize]byte, bool) {
	ak, ok := r.admittedKeySet.Lookup(svtnID, nodeAddr)
	if !ok {
		return [hmac.KeySize]byte{}, false
	}
	return hmac.DeriveDiscoveryKey([]byte(ak.PublicKey), svtnID), true
}

// DeriveDiscoveryKey is the sender-side symmetric wrapper over
// hmac.DeriveDiscoveryKey: it lets an access node compute its own
// discovery_auth_key locally (both inputs — own public key, own SVTN ID —
// are locally known) without querying the router, and without
// internal/discovery importing internal/hmac directly (AC-004
// postcondition 4; ARCH-08 §6.5 position 14: discovery→routing is legal,
// discovery→hmac is forbidden).
func DeriveDiscoveryKey(pubkey []byte, svtnID [16]byte) [hmac.KeySize]byte {
	return hmac.DeriveDiscoveryKey(pubkey, svtnID)
}
