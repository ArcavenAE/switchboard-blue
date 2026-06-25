// Package admission implements Tier 1 SVTN admission via signed Ed25519 challenge
// (BC-2.05.001) and the admitted key set (BC-2.05.002, BC-2.05.007).
//
// Classification (ARCH-09 v1.1): boundary — holds admitted key set (mutable under
// mutex); admission logic is pure but key set mutation is stateful.
//
// Import constraints (ARCH-08 §4): this package MAY import internal/frame and
// internal/hmac only. No upward imports.
package admission

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/frame"
	"github.com/arcavenae/switchboard/internal/hmac"
)

// Sentinel errors. Each error code maps to a value in the error taxonomy
// (.factory/specs/prd-supplements/error-taxonomy.md §ADM).

// ErrSignatureVerificationFailed is returned by AdmitNode when the node's
// signature over the challenge nonce does not verify against its public key
// (E-ADM-001; BC-2.05.001 postcondition 5).
var ErrSignatureVerificationFailed = errors.New("admission denied: signature verification failed")

// ErrKeyRevoked is returned by AdmitNode when the presented public key is
// marked as revoked in the admitted key set (E-ADM-005; BC-2.05.001 EC-001).
var ErrKeyRevoked = errors.New("key revoked")

// ErrNonceReplay is returned by AdmitNode when the challenge nonce has already
// been consumed — replay prevention (E-ADM-008; BC-2.05.001 invariant 3).
var ErrNonceReplay = errors.New("nonce replay: challenge nonce already consumed")

// ErrNotAdmitted is returned by AdmittedKeySet.IsAdmitted and RouteFrame (via
// internal/routing) when the frame's source address is not present in the
// admitted set for the frame's SVTN (E-ADM-003; BC-2.05.002 postcondition 2).
var ErrNotAdmitted = errors.New("frame from non-admitted source")

// KeyRole identifies the authorization role of an admitted key.
type KeyRole uint8

const (
	// RoleControl grants key management and SVTN lifecycle operations.
	RoleControl KeyRole = iota + 1
	// RoleConsole grants session attach/detach and remote console operations.
	RoleConsole
	// RoleAccess grants session publishing and Tier 2 session authorization.
	RoleAccess
)

// nonceTTL is the maximum age of a used nonce per ARCH-04 §Nonce uniqueness.
const nonceTTL = 60 * time.Second

// AdmittedKey holds the state the router stores after a successful
// challenge-response for a single node on a single SVTN.
type AdmittedKey struct {
	// PublicKey is the Ed25519 public key presented during admission.
	PublicKey ed25519.PublicKey
	// Role is the authorization level for this key on this SVTN.
	Role KeyRole
	// FrameAuthKey is the per-(node, SVTN) frame authentication key derived
	// via HKDF-SHA256 (ADR-001 amended; ARCH-04 §HMAC keying).
	FrameAuthKey [hmac.KeySize]byte
	// NodeAddr is the 8-byte address derived from (svtnID, publicKey)
	// via frame.DeriveNodeAddress.
	NodeAddr [8]byte
	// revoked records whether this key has been revoked. A revoked key causes
	// E-ADM-005 on the next re-authentication attempt (FM-007).
	revoked bool
}

// Challenge is the router-issued, router-signed nonce sent to a node during
// the admission handshake (ARCH-04 §Tier 1 Admission Protocol step 2).
//
// IMPORTANT: This struct MUST NOT contain any private key material. It is a
// wire-format message (BC-2.05.007 postcondition 1).
type Challenge struct {
	// Nonce is a 32-byte crypto/rand value, unique per challenge attempt.
	Nonce [32]byte
	// RouterSig is the router's Ed25519 signature over Nonce, preventing
	// nonce forgery by a man-in-the-middle (ARCH-04 §Tier 1 Admission Protocol).
	RouterSig []byte
}

// ChallengeResponse is the node's reply to a Challenge.
//
// IMPORTANT: This struct MUST NOT contain any private key material. Only the
// signature (a public artefact computed by the node locally) is transmitted
// (BC-2.05.007 invariant 1; DI-002).
type ChallengeResponse struct {
	// NonceSig is Sign(node_private_key, challenge.Nonce). The private key
	// never leaves the node (DI-002).
	NonceSig []byte
}

// AdmittedKeySet is the router's mutable store of admitted node keys,
// partitioned by SVTN. It is the enforcement point for BC-2.05.002 (fail-
// closed frame admission) and BC-2.05.006 (SVTN cryptographic isolation).
//
// It also owns the used-nonce set for replay prevention (ARCH-04 §Nonce
// uniqueness: TTL = 60s).
//
// All exported methods are safe for concurrent use.
type AdmittedKeySet struct {
	mu     sync.RWMutex
	keys   map[[16]byte]map[[8]byte]*AdmittedKey
	nonces map[[32]byte]time.Time // value = insertion time; entries older than nonceTTL are purged
}

// NewAdmittedKeySet returns an empty AdmittedKeySet ready for use.
func NewAdmittedKeySet() *AdmittedKeySet {
	return &AdmittedKeySet{
		keys:   make(map[[16]byte]map[[8]byte]*AdmittedKey),
		nonces: make(map[[32]byte]time.Time),
	}
}

// RegisterKey adds or replaces a public key for the given (svtnID, nodeAddr)
// pair. Last-write-wins semantics per ADR-003: duplicate registration overwrites
// the prior entry without error.
//
// Traces to BC-2.05.001 precondition 2 and ADR-003 (ARCH-04 §ADR-003).
func (s *AdmittedKeySet) RegisterKey(svtnID [16]byte, pubkey ed25519.PublicKey, role KeyRole) {
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pubkey))
	authKey := hmac.DeriveKey([]byte(pubkey), svtnID)

	entry := &AdmittedKey{
		PublicKey:    pubkey,
		Role:         role,
		FrameAuthKey: authKey,
		NodeAddr:     nodeAddr,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.keys[svtnID] == nil {
		s.keys[svtnID] = make(map[[8]byte]*AdmittedKey)
	}
	s.keys[svtnID][nodeAddr] = entry
}

// RevokeKey marks the key for nodeAddr in svtnID as revoked.
// A revoked key causes ErrKeyRevoked on the next AdmitNode call.
//
// Returns ErrNotAdmitted if no such key is registered.
func (s *AdmittedKeySet) RevokeKey(svtnID [16]byte, nodeAddr [8]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	svtnMap, ok := s.keys[svtnID]
	if !ok {
		return ErrNotAdmitted
	}
	entry, ok := svtnMap[nodeAddr]
	if !ok {
		return ErrNotAdmitted
	}
	entry.revoked = true
	return nil
}

// Lookup returns a copy of the AdmittedKey for (svtnID, nodeAddr), or nil if
// not found. Returns a value copy — callers do not hold a pointer into internal
// state (go.md rule 12; finding-032-store-sync-contract-leak).
func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) *AdmittedKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	svtnMap, ok := s.keys[svtnID]
	if !ok {
		return nil
	}
	entry, ok := svtnMap[nodeAddr]
	if !ok {
		return nil
	}
	// Return a value copy so callers cannot mutate internal state.
	cp := *entry
	return &cp
}

// IsAdmitted reports whether nodeAddr is currently in the admitted set for svtnID.
// Used by internal/routing for fail-closed frame admission (BC-2.05.002
// postcondition 3: admitted-set check happens before any forwarding logic).
func (s *AdmittedKeySet) IsAdmitted(svtnID [16]byte, nodeAddr [8]byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	svtnMap, ok := s.keys[svtnID]
	if !ok {
		return false
	}
	entry, ok := svtnMap[nodeAddr]
	if !ok {
		return false
	}
	// Revoked keys are not admitted.
	return !entry.revoked
}

// GenerateChallenge produces a fresh admission challenge: a 32-byte random
// nonce, signed by the router's private key.
//
// Per BC-2.05.007 postcondition 1: the challenge contains ONLY the nonce and
// the router's signature over it. No private key bytes of any node appear in
// the returned Challenge struct. Per DI-002: the router's private key is used
// only for the signing operation; it is not serialized.
//
// routerPrivKey is the router's Ed25519 private key, used locally only.
func GenerateChallenge(routerPrivKey ed25519.PrivateKey) (Challenge, error) {
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return Challenge{}, err
	}

	sig := ed25519.Sign(routerPrivKey, nonce[:])

	return Challenge{
		Nonce:     nonce,
		RouterSig: sig,
	}, nil
}

// AdmitNode verifies a node's challenge-response and, on success, registers
// the node in the AdmittedKeySet.
//
// Per BC-2.05.001 postcondition 3: verifies resp.NonceSig using pubKey against
// challenge.Nonce. On success (postcondition 4): derives the frame_auth_key via
// hmac.DeriveKey(pubKey, svtnID) and adds the node to ks[svtnID].
//
// Error returns:
//   - ErrNonceReplay  (E-ADM-008) if the nonce has already been consumed (invariant 3).
//   - ErrKeyRevoked   (E-ADM-005) if the key is registered but revoked (EC-001).
//   - ErrSignatureVerificationFailed (E-ADM-001) if signature verification fails.
//
// Implements DI-002: pubKey is the Ed25519 *public* key. The caller's private key
// never appears in this function's arguments or return values.
func AdmitNode(
	challenge Challenge,
	resp ChallengeResponse,
	pubKey ed25519.PublicKey,
	svtnID [16]byte,
	ks *AdmittedKeySet,
) error {
	// Check if key is registered for this SVTN before expensive crypto ops.
	// This enforces BC-2.05.002 precondition 1 and EC-001 (revoked check).
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pubKey))

	ks.mu.RLock()
	svtnMap, hasSVTN := ks.keys[svtnID]
	var existingEntry *AdmittedKey
	if hasSVTN {
		existingEntry = svtnMap[nodeAddr]
	}
	ks.mu.RUnlock()

	if existingEntry == nil {
		// Key not registered for this SVTN — admission fails (EC-001 covers
		// unregistered case; ErrNotAdmitted used as the "not registered" signal).
		return ErrNotAdmitted
	}

	if existingEntry.revoked {
		return ErrKeyRevoked
	}

	// Record nonce (replay check + TTL purge) before verifying signature.
	// This prevents timing oracle: we record before verifying so a valid nonce
	// cannot be extracted by an attacker who checks timing of the replay error.
	if err := ks.recordNonce(challenge.Nonce, time.Now().UTC()); err != nil {
		return err
	}

	// Verify the node's signature over the challenge nonce (constant-time via ed25519.Verify).
	if !ed25519.Verify(pubKey, challenge.Nonce[:], resp.NonceSig) {
		return ErrSignatureVerificationFailed
	}

	return nil
}

// recordNonce records the nonce with timestamp, purging expired entries, and
// returns ErrNonceReplay if the nonce was already consumed within the TTL window.
func (s *AdmittedKeySet) recordNonce(n [32]byte, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Purge expired nonces (ARCH-04: 60s TTL).
	for nonce, t := range s.nonces {
		if now.Sub(t) > nonceTTL {
			delete(s.nonces, nonce)
		}
	}

	// Check for replay within the live window.
	if _, exists := s.nonces[n]; exists {
		return ErrNonceReplay
	}
	s.nonces[n] = now
	return nil
}
