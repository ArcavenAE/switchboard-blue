// Package admission: re-authentication handler for session continuity across
// IP address change (BC-2.01.007).
//
// When a node's source IP changes, the node re-authenticates using its
// existing SVTN admission keypair. The router verifies the signed challenge
// and updates its routing entry with the new source IP. The session channel ID
// and cryptographic node address remain unchanged (VP-036).
//
// Import constraints (ARCH-08 §4): this file may import only stdlib,
// internal/frame, and internal/hmac. No upward imports.
package admission

import (
	"errors"
	"net/netip"
	"time"
)

// ErrKeyExpired is returned by ReAuthenticate when the node's key has an
// expiry timestamp set and the current time is past that expiry. This is
// the re-authentication enforcement point for key expiry (E-ADM-005 family;
// BC-2.01.007 EC-001; ARCH-04 §Key Lifecycle).
//
// NOTE: The error-taxonomy maps E-ADM-002 to HMAC verification failure
// and E-ADM-005 to "key revoked". Key expiry at re-auth time is described in
// ARCH-04 §Key Lifecycle as "E-ADM-005 key expired". Story EC-001 cites
// E-ADM-002 for "key expired" — this appears to be a story-level
// cross-reference ambiguity. This sentinel is defined here as the
// re-authentication expiry sentinel; tests assert errors.Is(err, ErrKeyExpired).
// The orchestrator should route this ambiguity to the PO for error-taxonomy
// reconciliation before the implementer phase.
var ErrKeyExpired = errors.New("admission: key expired")

// ReAuthRequest carries the parameters for a re-authentication attempt from
// a new source IP address (BC-2.01.007 preconditions 1–4).
//
// IMPORTANT: This struct MUST NOT contain private key material. Only the
// public challenge-response (a signature) is transmitted (DI-002;
// BC-2.05.007 invariant 1).
type ReAuthRequest struct {
	// SVTNID is the SVTN the node is re-authenticating into.
	SVTNID [16]byte
	// NodeAddr is the node's cryptographic address, derived from
	// (svtnID, publicKey) via frame.DeriveNodeAddress. This is the stable
	// identity that persists across IP changes (BC-2.01.007 invariant 3;
	// VP-036).
	NodeAddr [8]byte
	// NewSourceAddr is the node's new source IP address after the IP change.
	// The router uses this to update its routing entry (BC-2.01.007 PC3).
	NewSourceAddr netip.Addr
	// Challenge is the router-issued challenge for this re-auth attempt.
	Challenge Challenge
	// Response is the node's signed response to the challenge.
	Response ChallengeResponse
}

// sourceAddrStore is the per-(SVTN, nodeAddr) source IP index maintained by
// ReAuthenticate. It records the most-recently accepted source address per
// node, enabling old-path eviction (BC-2.01.007 EC-002) and concurrent
// re-auth LWW (ADR-003; BC-2.01.007 EC-003; story EC-003).
//
// Design: separate from AdmittedKey to avoid modifying the existing struct
// and to keep the source-IP concern isolated to the re-auth path. The
// admitted key set remains the authority on cryptographic admission; the
// source addr store is the authority on current network path.
type sourceAddrStore struct {
	// addrs maps svtnID → nodeAddr → current source IP.
	// Protected by AdmittedKeySet.mu (callers hold the write lock when
	// mutating and either lock when reading).
	addrs map[[16]byte]map[[8]byte]netip.Addr
}

// newSourceAddrStore initialises an empty store.
func newSourceAddrStore() *sourceAddrStore {
	return &sourceAddrStore{
		addrs: make(map[[16]byte]map[[8]byte]netip.Addr),
	}
}

// ReAuthState is associated with an AdmittedKeySet and tracks per-node source
// addresses and key expiry timestamps for the re-authentication path. Callers
// construct one per AdmittedKeySet instance and pass it to ReAuthenticate.
//
// All exported methods are safe for concurrent use.
type ReAuthState struct {
	store    *sourceAddrStore
	expiries map[[16]byte]map[[8]byte]time.Time // svtnID → nodeAddr → expiry (zero = no expiry)
}

// NewReAuthState returns a ReAuthState ready for use with an AdmittedKeySet.
func NewReAuthState() *ReAuthState {
	return &ReAuthState{
		store:    newSourceAddrStore(),
		expiries: make(map[[16]byte]map[[8]byte]time.Time),
	}
}

// SetKeyExpiry records an expiry time for (svtnID, nodeAddr). ReAuthenticate
// will return ErrKeyExpired if the current time is past this value (EC-001;
// ARCH-04 §Key Lifecycle: "Expiry check is at re-authentication time").
//
// A zero Time clears any previously set expiry.
//
// Note for implementer: a future refactor may move expiry into AdmittedKey
// directly (add an Expiry time.Time field to the struct in admission.go);
// that is a clean-cut: SetKeyExpiry would then set entry.Expiry under the
// write lock rather than using ReAuthState.expiries. This stub keeps the
// scope contained to the S-1.03 files.
//
// TODO(S-1.03): implement per ARCH-04 §Key Lifecycle; BC-2.01.007 EC-001
func (s *AdmittedKeySet) SetKeyExpiry(svtnID [16]byte, nodeAddr [8]byte, expiry time.Time) error {
	panic("not implemented") //nolint:gocritic // stub: Red Gate per BC-5.38.001
}

// ReAuthenticate verifies a node's re-authentication request, updates the
// source IP routing entry on success, and evicts the old path (BC-2.01.007
// postconditions 1–4; EC-002; ADR-003 LWW for EC-003).
//
// Preconditions (BC-2.01.007):
//  1. The node has an active session (admitted=true in ks).
//  2. The node's SVTN admission keypair is unchanged (Pre3).
//  3. The node presents a valid signed challenge response.
//
// Postconditions on success:
//  1. The node remains admitted; its cryptographic node address is unchanged (PC1, VP-036).
//  2. The routing entry is updated to reflect the new source IP (PC3).
//  3. The previous source IP entry is evicted (EC-002).
//
// Error returns:
//   - ErrNotAdmitted      (E-ADM-003) if the node is not currently admitted.
//   - ErrKeyRevoked       (E-ADM-005) if the key has been revoked.
//   - ErrKeyExpired                   if the key's expiry has passed (EC-001; ARCH-04 §Key Lifecycle).
//   - ErrSignatureVerificationFailed  (E-ADM-001) if the challenge signature is invalid (AC-002).
//   - ErrNonceReplay      (E-ADM-008) if the challenge nonce was already consumed.
//
// TODO(S-1.03): implement per BC-2.01.007 PC1–PC4; ARCH-04 §Tier 1 Admission Protocol
func ReAuthenticate(req ReAuthRequest, ks *AdmittedKeySet, rs *ReAuthState) error {
	panic("not implemented") //nolint:gocritic // stub: Red Gate per BC-5.38.001
}

// CurrentSourceAddr returns the most-recently accepted source IP for
// (svtnID, nodeAddr), or the zero netip.Addr if no re-authentication has
// occurred yet for this node.
//
// TODO(S-1.03): implement per BC-2.01.007 PC3
func (rs *ReAuthState) CurrentSourceAddr(svtnID [16]byte, nodeAddr [8]byte) netip.Addr {
	panic("not implemented") //nolint:gocritic // stub: Red Gate per BC-5.38.001
}
