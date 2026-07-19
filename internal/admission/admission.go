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
	"fmt"
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

// ErrKeyNotRegistered is returned when a key-lifecycle operation (e.g.,
// RevokeKey) is called against a (SVTN, node) tuple that has no registered
// key. Distinct from ErrNotAdmitted (frame-routing sentinel, E-ADM-003) so
// callers can distinguish "no such key" from "frame from non-admitted source"
// via errors.Is. Maps to E-ADM-013 per error-taxonomy.md.
var ErrKeyNotRegistered = errors.New("admission: key not registered for (SVTN, node)")

// ErrRoleMismatch is the sentinel error for role-mismatch (E-ADM-019).
// RevokeKeyIfRoleMatches and SetKeyExpiryIfRoleMatches return a *RoleMismatchError
// that wraps this sentinel — callers may use errors.Is(err, ErrRoleMismatch) or
// errors.As(err, &*RoleMismatchError) depending on whether they need the detail.
var ErrRoleMismatch = errors.New("admission: role mismatch: stored role differs from expected role")

// RoleMismatchError is returned by RevokeKeyIfRoleMatches and
// SetKeyExpiryIfRoleMatches when the caller-supplied role does not match the
// role stored in the registry (E-ADM-019). It carries both the claimed role
// (what the caller expected) and the registered role (what the registry holds)
// so that wire-level error formatters can produce the full canonical message
// without a second lookup.
//
// Unwrap returns ErrRoleMismatch so errors.Is(err, ErrRoleMismatch) continues
// to work for callers that do not need the role detail.
type RoleMismatchError struct {
	// ClaimedRole is the role the caller supplied (expectedRole argument).
	ClaimedRole KeyRole
	// RegisteredRole is the role stored in the registry at match-time.
	RegisteredRole KeyRole
}

// Error implements the error interface.
func (e *RoleMismatchError) Error() string {
	return fmt.Sprintf(
		"admission: role mismatch: claimed role %s does not match registered role %s",
		e.ClaimedRole, e.RegisteredRole,
	)
}

// Unwrap returns ErrRoleMismatch so errors.Is(err, ErrRoleMismatch) works.
func (e *RoleMismatchError) Unwrap() error { return ErrRoleMismatch }

// ErrControlRevocationRequiresConfirm is returned by RevokeKeyIfRoleMatches
// when the stored key has RoleControl and confirmControlRevocation is false
// (E-ADM-018). The check is performed inside the write lock — after the
// role-match check passes but before revoked is set — so no revocation occurs
// on this error path (ARCH-04 Addendum H2; ADR-004).
var ErrControlRevocationRequiresConfirm = errors.New(
	"control-to-control revocation requires explicit confirmation",
)

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

// ErrUnknownKeyRole is returned by KeyRoleFromString when the supplied string
// does not match any known role (E-CFG-001).
var ErrUnknownKeyRole = errors.New("E-CFG-001: unknown key role")

// KeyRoleFromString converts a canonical role string ("control", "console",
// "access") to a KeyRole. Returns ErrUnknownKeyRole for any unrecognised value,
// preventing callers from accidentally mapping unknowns to the RoleControl zero
// value (SEC-001; PR #34 security review).
func KeyRoleFromString(s string) (KeyRole, error) {
	switch s {
	case "control":
		return RoleControl, nil
	case "console":
		return RoleConsole, nil
	case "access":
		return RoleAccess, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnknownKeyRole, s)
	}
}

// String returns the canonical wire string for the role ("control", "console",
// "access"). Unknown values return "unknown(<n>)" to aid diagnostics.
func (r KeyRole) String() string {
	switch r {
	case RoleControl:
		return "control"
	case RoleConsole:
		return "console"
	case RoleAccess:
		return "access"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(r))
	}
}

// nonceTTL is the maximum age of a used nonce per ARCH-04 §Nonce uniqueness.
const nonceTTL = 60 * time.Second

// noncePurgePeriod is the minimum interval between O(N) nonce-map sweeps (M-2).
const noncePurgePeriod = time.Second

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
	// admitted records whether this key has completed the challenge-response
	// handshake (BC-2.05.001 PC4). False at RegisterKey; true only after a
	// successful AdmitNode call. IsAdmitted AND-gates on this field.
	admitted bool
	// expiry is the optional key expiry time. A zero value means no expiry.
	// ReAuthenticate checks this at re-auth time; if now > expiry the node is
	// not re-admitted (E-ADM-015; ARCH-04 §Key Lifecycle; BC-2.01.007 EC-005).
	// Unexported to prevent direct mutation; use SetKeyExpiry and KeyExpiry().
	expiry time.Time
}

// KeyExpiry returns the key's expiry time. A zero Time means no expiry is set.
// Use AdmittedKeySet.SetKeyExpiry to set or clear the expiry (E-ADM-015).
func (k AdmittedKey) KeyExpiry() time.Time { return k.expiry }

// IsRevoked reports whether this key has been revoked.
// A revoked key causes ErrKeyRevoked on the next re-authentication attempt (FM-007).
// Used by svtnmgmt.CallerKeyRoleActive to deny revoked keys at handler authority
// resolution time (F-P4L1-003 / BC-2.05.004 PC-1).
func (k AdmittedKey) IsRevoked() bool { return k.revoked }

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
	mu        sync.RWMutex
	keys      map[[16]byte]map[[8]byte]*AdmittedKey
	nonces    map[[32]byte]time.Time // value = insertion time; entries older than nonceTTL are purged
	lastPurge time.Time              // tracks last O(N) nonce-map sweep for lazy-purge gate (M-2)
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
// The resulting entry has admitted=false — the node must complete the
// challenge-response handshake via AdmitNode before IsAdmitted returns true
// (BC-2.05.001 PC4; H-2).
//
// Traces to BC-2.05.001 precondition 2 and ADR-003 (ARCH-04 §ADR-003).
func (s *AdmittedKeySet) RegisterKey(svtnID [16]byte, pubkey ed25519.PublicKey, role KeyRole) {
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pubkey))
	authKey := hmac.DeriveKey([]byte(pubkey), svtnID)

	// admitted is intentionally zero (false) — the node must complete the
	// challenge-response handshake before IsAdmitted returns true.
	//
	// Deep-clone the caller's pubkey slice so that a subsequent mutation of the
	// caller's backing array cannot corrupt the stored entry (go.md rule 12; M-3;
	// F-P3L2-001 RegisterKey caller-alias fence).
	entry := &AdmittedKey{
		PublicKey:    append(ed25519.PublicKey(nil), pubkey...),
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
// Returns ErrKeyNotRegistered (E-ADM-013) if no such (SVTN, node) tuple is
// registered. Use errors.Is to distinguish this from ErrNotAdmitted
// (E-ADM-003), which is the frame-routing sentinel.
func (s *AdmittedKeySet) RevokeKey(svtnID [16]byte, nodeAddr [8]byte) error {
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
	entry.revoked = true
	return nil
}

// RevokeKeyIfRoleMatches atomically checks that the key's stored role matches
// expectedRole, enforces the control-revocation confirm gate if needed, then
// marks the key revoked — all under a single write lock critical section
// (HOLD-001; ARCH-04 v1.13 Addendum H2).
//
// The confirm gate is evaluated INSIDE the write lock, after the role-match
// check passes but BEFORE setting revoked=true. This means no revocation
// occurs when ErrControlRevocationRequiresConfirm is returned (ARCH-04 H2
// step ordering: atomic-FIRST, confirm-gate-SECOND, no intermediate state).
//
// This prevents the time-of-check/time-of-use race in the non-atomic
// Lookup → cross-check → RevokeKey sequence: a concurrent RegisterKey with a
// different role cannot slip between the check and the revoke.
//
// Returns:
//   - (existingRole, nil)                              on success (key revoked)
//   - (0, ErrKeyNotRegistered)                         if no (SVTN, node) tuple exists (E-ADM-013)
//   - (existingRole, ErrRoleMismatch)                  if stored role != expectedRole (E-ADM-019)
//   - (existingRole, ErrControlRevocationRequiresConfirm) if key is RoleControl and confirmControlRevocation=false (E-ADM-018)
//
// Traces to HOLD-001 hybrid approach; BC-2.05.004 precondition 1; ADR-004;
// ARCH-04 Addendum H2 lines 548-563.
func (s *AdmittedKeySet) RevokeKeyIfRoleMatches(
	svtnID [16]byte,
	pubkey ed25519.PublicKey,
	expectedRole KeyRole,
	confirmControlRevocation bool,
) (existingRole KeyRole, err error) {
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pubkey))

	s.mu.Lock()
	defer s.mu.Unlock()

	svtnMap, ok := s.keys[svtnID]
	if !ok {
		return 0, ErrKeyNotRegistered
	}
	entry, ok := svtnMap[nodeAddr]
	if !ok {
		return 0, ErrKeyNotRegistered
	}

	// Capture the role before any mutation so callers can inspect it on error.
	existingRole = entry.Role

	// Atomic cross-check: if stored role differs from expected, reject.
	// This prevents a concurrent RegisterKey (LWW) from changing the role
	// between the caller's check and this revocation.
	if entry.Role != expectedRole {
		return existingRole, &RoleMismatchError{
			ClaimedRole:    expectedRole,
			RegisteredRole: existingRole,
		}
	}

	// Confirm gate (ARCH-04 H2 step ordering): evaluated AFTER role-match,
	// BEFORE setting revoked. No revocation occurs on this error path —
	// the key is unchanged if ErrControlRevocationRequiresConfirm is returned.
	// Uses existingRole (== expectedRole at this point per the role-match check
	// above) rather than a second read of entry.Role to make the intent clear.
	if existingRole == RoleControl && !confirmControlRevocation {
		return existingRole, ErrControlRevocationRequiresConfirm
	}

	entry.revoked = true
	return existingRole, nil
}

// Lookup returns a copy of the AdmittedKey for (svtnID, nodeAddr) and true,
// or the zero AdmittedKey and false if not found. Returns a value copy —
// callers do not hold a pointer into internal state (go.md rule 12;
// finding-032-store-sync-contract-leak).
//
// PublicKey is deep-cloned so the returned copy's backing array is independent
// of the live map entry (M-3).
func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	svtnMap, ok := s.keys[svtnID]
	if !ok {
		return AdmittedKey{}, false
	}
	entry, ok := svtnMap[nodeAddr]
	if !ok {
		return AdmittedKey{}, false
	}
	// Return a value copy so callers cannot mutate internal state.
	cp := *entry
	// Deep-clone PublicKey: ed25519.PublicKey is []byte; shallow copy shares
	// the backing array. Callers must not alias live state (go.md rule 12; M-3).
	cp.PublicKey = append(ed25519.PublicKey(nil), entry.PublicKey...)
	return cp, true
}

// LookupByPubkey looks up an admitted key by SVTN ID and Ed25519 public key,
// deriving the node address internally. Returns the zero AdmittedKey and false
// if the key is not registered or if svtnID does not exist.
//
// Convenience wrapper over Lookup for callers that hold the raw public key
// rather than the derived node address (ARCH-04 v1.8, ARCH-08 §6.6 position 15).
// Thread-safety and deep-clone guarantees are inherited from Lookup.
func (s *AdmittedKeySet) LookupByPubkey(svtnID [16]byte, pubkey ed25519.PublicKey) (AdmittedKey, bool) {
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pubkey))
	return s.Lookup(svtnID, nodeAddr)
}

// IsAdmitted reports whether nodeAddr has completed the challenge-response
// handshake for svtnID and has not been revoked.
//
// AND-gates on both admitted and !revoked (BC-2.05.001 PC4; H-2):
// a registered-but-not-handshaked node returns false.
//
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
	// Both conditions required: handshake completed AND key not revoked.
	return entry.admitted && !entry.revoked
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

// AdmitNode verifies a node's challenge-response and, on success, marks the
// node as admitted in the AdmittedKeySet (BC-2.05.001 PC4).
//
// Per BC-2.05.001 postcondition 3: verifies resp.NonceSig using pubKey against
// challenge.Nonce. On success (postcondition 4): records the nonce (replay
// prevention) and sets entry.admitted = true atomically under the write lock.
//
// Error returns:
//   - ErrNotAdmitted  (E-ADM-003) if the key is not registered for this SVTN.
//   - ErrKeyRevoked   (E-ADM-005) if the key is registered but revoked (EC-001).
//   - ErrNonceReplay  (E-ADM-008) if the nonce has already been consumed (invariant 3).
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
	nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pubKey))

	// Step 1: snapshot the entry under RLock (H-1).
	// Reading existingEntry.revoked OUTSIDE the lock is a Go memory-model
	// violation; snapshot the value while the lock is held instead.
	ks.mu.RLock()
	svtnMap, hasSVTN := ks.keys[svtnID]
	var snap AdmittedKey
	var found bool
	if hasSVTN {
		if e := svtnMap[nodeAddr]; e != nil {
			snap = *e // value copy inside RLock — revoked bool safely captured
			found = true
		}
	}
	ks.mu.RUnlock()

	if !found {
		// Key not registered for this SVTN (ErrNotAdmitted; E-ADM-003).
		return ErrNotAdmitted
	}

	if snap.revoked {
		return ErrKeyRevoked
	}

	// Step 2 (expiry): check expiry under the RLock snapshot before acquiring the write lock.
	// Mirrors ReAuthenticate reauth.go:195-197. The snapshot was taken under RLock so
	// snap.expiry is a safe read (no torn access). A concurrent SetKeyExpiry racing between
	// RUnlock and Lock is caught by the re-check under the write lock below (O-1; BC-2.05.001
	// PC-6 / Invariant 5; S-BL.NODE-IDENTIFY-WIRE rulings §15).
	now := time.Now().UTC()
	if !snap.expiry.IsZero() && now.After(snap.expiry) {
		return ErrKeyExpired
	}

	// Step 3: acquire write lock once — record nonce AND set admitted=true atomically (H-2).
	// Re-checking revoked under the write lock defends against a concurrent RevokeKey
	// call that raced between our RUnlock (step 1) and this Lock.
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Re-fetch entry; it may have been replaced by a concurrent RegisterKey (ADR-003 LWW).
	liveEntry := ks.keys[svtnID][nodeAddr]
	if liveEntry == nil || liveEntry.revoked {
		return ErrKeyRevoked
	}
	// Re-check expiry under write lock in case SetKeyExpiry raced (mirrors reauth.go:219-222).
	if !liveEntry.expiry.IsZero() && now.After(liveEntry.expiry) {
		return ErrKeyExpired
	}

	// Inline nonce record (replaces recordNonce call — avoids double-lock and
	// keeps nonce + admitted mutation in a single critical section).
	if err := ks.recordNonceUnlocked(challenge.Nonce, now); err != nil {
		return err
	}

	// Verify signature against the stored public key (BC-2.05.001 PC-3): the router
	// MUST use the key from the admitted keyset, not the pubkey supplied in the frame.
	// The frame pubkey was used only to derive the nodeAddr lookup (line ~464); it is
	// untrusted input and must not be the verification authority.  Using liveEntry.PublicKey
	// (the stored key) closes the impersonation vector where an attacker presents a valid
	// nodeAddr with a different key and forges a signature with their own private key.
	// Symmetry: reauth.go verifies against snap.PublicKey (the stored key) for the same reason.
	//
	// The nonce is already consumed above; sig failure returns error and the consumed
	// nonce prevents replay of the same challenge.
	if !ed25519.Verify(liveEntry.PublicKey, challenge.Nonce[:], resp.NonceSig) {
		return ErrSignatureVerificationFailed
	}

	// Mark admitted on success (BC-2.05.001 PC4; H-2).
	liveEntry.admitted = true
	return nil
}

// RemoveSVTN deletes all registered keys for svtnID from the key set.
// After this call, ListBySVTN(svtnID) returns an empty slice and
// LookupByPubkey / IsAdmitted return (zero, false) for all former entries.
//
// This is the bulk-removal primitive used by SVTNManager.Destroy (S-6.05;
// BC-2.07.001 postcondition 3; ARCH-04 admission ordering invariant: key
// removal precedes SVTN ID free).
//
// Safe for concurrent use.
func (s *AdmittedKeySet) RemoveSVTN(svtnID [16]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, svtnID)
}

// ListBySVTN returns a snapshot of all admitted key entries for the given SVTN.
// Returns an empty (non-nil) slice when no keys are registered for svtnID.
// Each returned AdmittedKey is a deep copy — PublicKey backing array is independent
// of internal state (go.md rule 12; finding-032-store-sync-contract-leak).
func (s *AdmittedKeySet) ListBySVTN(svtnID [16]byte) []AdmittedKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	svtnMap := s.keys[svtnID]
	out := make([]AdmittedKey, 0, len(svtnMap))
	for _, entry := range svtnMap {
		cp := *entry
		// Deep-clone PublicKey: []byte shallow copy shares backing array (M-3).
		cp.PublicKey = append(ed25519.PublicKey(nil), entry.PublicKey...)
		out = append(out, cp)
	}
	return out
}

// AllSVTNEntries returns a snapshot of all admitted key entries grouped by SVTN ID.
// The map and all slice values are deep copies — independent of internal state.
// Used by marshalSnapshot to serialise the full keyset (S-BL.ADMISSION-SYNC-WIRE;
// BC-2.05.010 PC-4; cmd/switchboard admission_sync_snapshot.go).
func (s *AdmittedKeySet) AllSVTNEntries() map[[16]byte][]AdmittedKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[[16]byte][]AdmittedKey, len(s.keys))
	for svtnID, svtnMap := range s.keys {
		entries := make([]AdmittedKey, 0, len(svtnMap))
		for _, entry := range svtnMap {
			cp := *entry
			// Deep-clone PublicKey: []byte shallow copy shares backing array (M-3).
			cp.PublicKey = append(ed25519.PublicKey(nil), entry.PublicKey...)
			entries = append(entries, cp)
		}
		out[svtnID] = entries
	}
	return out
}

// recordNonceUnlocked records the nonce with timestamp and returns ErrNonceReplay
// if already consumed within the TTL window. Performs a lazy O(N) purge sweep
// gated by elapsed time or map size (M-2).
//
// MUST be called with ks.mu held for writing. Does not acquire the lock itself.
func (s *AdmittedKeySet) recordNonceUnlocked(n [32]byte, now time.Time) error {
	// Lazy purge: sweep only when the map has grown large OR enough time has
	// passed since the last sweep. Amortised O(1) per call at steady state (M-2).
	if now.Sub(s.lastPurge) > noncePurgePeriod || len(s.nonces) > 10000 {
		for nonce, t := range s.nonces {
			if now.Sub(t) > nonceTTL {
				delete(s.nonces, nonce)
			}
		}
		s.lastPurge = now
	}

	// Check for replay within the live window.
	if _, exists := s.nonces[n]; exists {
		return ErrNonceReplay
	}
	s.nonces[n] = now
	return nil
}
