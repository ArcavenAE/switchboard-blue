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
	"crypto/ed25519"
	"errors"
	"net/netip"
	"sync"
	"time"
)

// ErrKeyExpired is returned by ReAuthenticate when the node's key has an
// expiry timestamp set and the current time is past that expiry. This is
// the re-authentication enforcement point for key expiry (E-ADM-015;
// BC-2.01.007 EC-005; ARCH-04 §Key Lifecycle).
var ErrKeyExpired = errors.New("admission: key expired")

// ReAuthRequest carries the parameters for a re-authentication attempt from
// a new source IP address (BC-2.01.007 preconditions 1–4).
//
// IMPORTANT: This struct MUST NOT contain private key material. Only the
// public challenge-response (a signature) is transmitted (DI-002).
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

// ReAuthState is associated with an AdmittedKeySet and tracks per-node source
// addresses for the re-authentication path. Callers construct one per
// AdmittedKeySet instance and pass it to ReAuthenticate.
//
// All exported methods are safe for concurrent use.
type ReAuthState struct {
	mu sync.RWMutex
	// TODO(phase-6): no eviction path; map grows monotonically with admitted nodes.
	// Eviction hook needed when RevokeKey or expiry cleanup runs. Track in STATE.md drift register.
	// VP-036 backlog item.
	addrs map[[16]byte]map[[8]byte]netip.Addr // svtnID → nodeAddr → current source IP
}

// NewReAuthState returns a ReAuthState ready for use with an AdmittedKeySet.
func NewReAuthState() *ReAuthState {
	return &ReAuthState{
		addrs: make(map[[16]byte]map[[8]byte]netip.Addr),
	}
}

// setSourceAddr updates the source address for (svtnID, nodeAddr) under the
// write lock. Old path is implicitly evicted by the map overwrite (BC-2.01.007
// EC-006; ADR-003 LWW).
func (rs *ReAuthState) setSourceAddr(svtnID [16]byte, nodeAddr [8]byte, addr netip.Addr) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.addrs[svtnID] == nil {
		rs.addrs[svtnID] = make(map[[8]byte]netip.Addr)
	}
	rs.addrs[svtnID][nodeAddr] = addr
}

// SetKeyExpiry records an expiry time for (svtnID, nodeAddr) in the
// AdmittedKeySet entry. ReAuthenticate will return ErrKeyExpired if the
// current time is past this value (E-ADM-015; EC-001; ARCH-04 §Key Lifecycle:
// "Expiry check is at re-authentication time").
//
// A zero Time clears any previously set expiry.
//
// Returns ErrKeyNotRegistered (E-ADM-013) if no key is registered for the
// given (svtnID, nodeAddr).
func (s *AdmittedKeySet) SetKeyExpiry(svtnID [16]byte, nodeAddr [8]byte, expiry time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	svtnMap, ok := s.keys[svtnID]
	if !ok {
		return ErrKeyNotRegistered
	}
	entry, ok := svtnMap[nodeAddr]
	if !ok {
		return ErrKeyNotRegistered
	}
	entry.expiry = expiry
	return nil
}

// ReAuthenticate verifies a node's re-authentication request, updates the
// source IP routing entry on success, and evicts the old path (BC-2.01.007
// postconditions 1–4; EC-006; ADR-003 LWW).
//
// Preconditions (BC-2.01.007):
//  1. The node has an active session (admitted=true in ks).
//  2. The node's SVTN admission keypair is unchanged (Pre3).
//  3. The node presents a valid signed challenge response.
//
// Precondition: the caller has verified challenge.RouterSig against the router's public key
// before invoking ReAuthenticate (same caller-trust model as AdmitNode; see BC-2.01.007 Pre-2).
//
// Postconditions on success:
//  1. The node remains admitted; its cryptographic node address is unchanged (PC4 / Inv3, VP-036).
//  2. The routing entry is updated to reflect the new source IP (PC3).
//  3. The previous source IP entry is evicted (EC-006).
//
// Error returns:
//   - ErrNotAdmitted      (E-ADM-003) if the node is not currently admitted.
//   - ErrKeyRevoked       (E-ADM-005) if the key has been revoked.
//   - ErrKeyExpired       (E-ADM-015) if the key's expiry has passed (EC-001; ARCH-04 §Key Lifecycle).
//   - ErrSignatureVerificationFailed  (E-ADM-001) if the challenge signature is invalid (AC-002).
//   - ErrNonceReplay      (E-ADM-008) if the challenge nonce was already consumed.
func ReAuthenticate(req ReAuthRequest, ks *AdmittedKeySet, rs *ReAuthState) error {
	// Step 1: snapshot the entry under RLock.
	ks.mu.RLock()
	svtnMap, hasSVTN := ks.keys[req.SVTNID]
	var snap AdmittedKey
	var found bool
	if hasSVTN {
		if e := svtnMap[req.NodeAddr]; e != nil {
			snap = *e
			found = true
		}
	}
	ks.mu.RUnlock()

	if !found || !snap.admitted {
		return ErrNotAdmitted
	}
	if snap.revoked {
		return ErrKeyRevoked
	}

	// Step 2: check expiry before acquiring the write lock (pure, no side effects).
	now := time.Now().UTC()
	if !snap.expiry.IsZero() && now.After(snap.expiry) {
		return ErrKeyExpired
	}

	// Step 3: acquire write lock — record nonce, then verify signature.
	// Nonce is recorded BEFORE signature verification (BC-2.05.001 invariant 3:
	// "challenge nonce must be uniquely consumed"). This mirrors the AdmitNode
	// path in admission.go (recordNonceUnlocked at line ~337, ed25519.Verify at
	// line ~344): a failed-signature attempt still burns the nonce, preventing
	// same-nonce probe attacks on the re-authentication path.
	// Explicit lock/unlock (no defer) so we can release ks.mu before updating
	// rs (which has its own lock). Holding two locks simultaneously risks deadlock
	// if another goroutine acquires them in the opposite order.
	ks.mu.Lock()
	liveEntry := ks.keys[req.SVTNID][req.NodeAddr]
	if liveEntry == nil || !liveEntry.admitted {
		ks.mu.Unlock()
		return ErrNotAdmitted
	}
	if liveEntry.revoked {
		ks.mu.Unlock()
		return ErrKeyRevoked
	}
	// Re-check expiry under write lock in case SetKeyExpiry raced.
	if !liveEntry.expiry.IsZero() && now.After(liveEntry.expiry) {
		ks.mu.Unlock()
		return ErrKeyExpired
	}
	// Record nonce first (replay prevention, E-ADM-008; BC-2.05.001 invariant 3).
	if err := ks.recordNonceUnlocked(req.Challenge.Nonce, now); err != nil {
		ks.mu.Unlock()
		return err
	}
	// Verify signature after nonce is consumed — symmetric with AdmitNode.
	if !ed25519.Verify(snap.PublicKey, req.Challenge.Nonce[:], req.Response.NonceSig) {
		ks.mu.Unlock()
		return ErrSignatureVerificationFailed
	}
	ks.mu.Unlock()

	// Step 4: update source address (rs has its own lock; ks.mu already released).
	// Old path is evicted by overwrite (BC-2.01.007 EC-006; ADR-003 LWW).
	rs.setSourceAddr(req.SVTNID, req.NodeAddr, req.NewSourceAddr)
	return nil
}

// CurrentSourceAddr returns the most-recently accepted source IP for
// (svtnID, nodeAddr), or the zero netip.Addr if no re-authentication has
// occurred yet for this node (BC-2.01.007 PC3).
func (rs *ReAuthState) CurrentSourceAddr(svtnID [16]byte, nodeAddr [8]byte) netip.Addr {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	svtnMap, ok := rs.addrs[svtnID]
	if !ok {
		return netip.Addr{}
	}
	return svtnMap[nodeAddr]
}
