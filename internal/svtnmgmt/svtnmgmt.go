// Package svtnmgmt implements SVTN lifecycle management and key management
// operations for the control node via the management plane.
//
// Purity classification (ARCH-09): boundary — manages the SVTN registry and
// delegates key-set mutations to internal/admission. All exported methods are
// safe for concurrent use.
//
// Import constraints (ARCH-08 §6.6 position 15): this package MUST import only
// internal/admission and internal/config from the internal tree. No data-plane
// packages (routing, multipath, arq, replay, paths, halfchannel, session, tmux,
// discovery) are permitted.
package svtnmgmt

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// Sentinel errors for SVTN lifecycle operations.
// Error codes map to the error taxonomy at
// .factory/specs/prd-supplements/error-taxonomy.md.

// ErrSVTNAlreadyExists is returned by SVTNManager.Create when a SVTN with the
// given name already exists in the registry (E-SVTN-001; BC-2.07.001 EC-001).
var ErrSVTNAlreadyExists = errors.New("SVTN already exists")

// ErrSVTNNotFound is returned by operations that require the SVTN to exist
// when it is not present in the registry (E-SVTN-003).
var ErrSVTNNotFound = errors.New("SVTN not found")

// ErrInvalidDuration is returned when an expire duration of zero or negative
// is provided (E-CFG-001; S-6.02 EC-003).
var ErrInvalidDuration = errors.New("invalid duration: must be positive")

// ErrControlRevocationRequiresConfirm is re-exported from admission for
// callers that import only svtnmgmt. The underlying sentinel is
// admission.ErrControlRevocationRequiresConfirm (E-ADM-018); use
// errors.Is(err, admission.ErrControlRevocationRequiresConfirm) or
// errors.Is(err, svtnmgmt.ErrControlRevocationRequiresConfirm) — both are
// the same value.
// ADR-004; BC-2.05.004 precondition 1; AC-005; ARCH-04 Addendum H2.
var ErrControlRevocationRequiresConfirm = admission.ErrControlRevocationRequiresConfirm

// ErrRoleMismatch is re-exported from admission for callers that import only
// svtnmgmt. The underlying sentinel is admission.ErrRoleMismatch (E-ADM-019);
// use errors.Is(err, admission.ErrRoleMismatch) or errors.Is(err, svtnmgmt.ErrRoleMismatch)
// — both are the same value.
var ErrRoleMismatch = admission.ErrRoleMismatch

// ErrBootstrapKeyRevokeForbidden is returned when an attempt is made to revoke
// the daemon bootstrap control key. The bootstrap key is the trust anchor for
// the SVTN and must never be removed (E-ADM-020; bootstrap revocability invariant).
var ErrBootstrapKeyRevokeForbidden = errors.New("bootstrap control key cannot be revoked")

// SVTN is a record of a single Software-Defined Virtual Topology Network
// created and owned by this control node.
//
// Fields are value types; the SVTNManager returns value copies from all
// accessors (go.md rule 12; finding-032-store-sync-contract-leak).
type SVTN struct {
	// ID is the 16-byte globally-unique SVTN identifier, derived from
	// crypto/rand at creation time (BC-2.07.001 postcondition 1; DI-005).
	ID [16]byte
	// Name is the human-readable label provided by the operator at creation
	// time (sbctl admin create --name).
	Name string
	// CreatedAt is the UTC timestamp when the SVTN was created (go.md rule 11).
	CreatedAt time.Time
}

// CreateResult is the response from SVTNManager.Create, containing the newly
// assigned SVTN ID (BC-2.07.001 postcondition 1).
type CreateResult struct {
	// SVTN is the newly created SVTN record.
	SVTN SVTN
}

// KeyOpResult is the response from key lifecycle operations (register, revoke,
// expire). It carries the key fingerprint and operation timestamp for
// confirmation (BC-2.05.004 postcondition 4).
type KeyOpResult struct {
	// Fingerprint is the SHA-256 fingerprint of the key in the standard
	// "SHA256:<base64>" format (DI-002 — no private material).
	Fingerprint string
	// At is the UTC timestamp of the operation (go.md rule 11).
	At time.Time
}

// SVTNManager manages the SVTN registry and delegates key-set mutations to
// the shared AdmittedKeySet. It is the boundary object for BC-2.07.001
// (SVTN lifecycle) and BC-2.05.004 (key lifecycle).
//
// Construct via NewSVTNManager. Never copy after first use.
type SVTNManager struct {
	mu     sync.RWMutex
	svtns  map[string]SVTN // keyed by SVTN name for uniqueness check (DI-005)
	keySet *admission.AdmittedKeySet
	// controlPubKey is the control node's own Ed25519 public key, bootstrapped
	// locally as the first admitted key when a new SVTN is created
	// (BC-2.07.001 postcondition 1 + 2).
	controlPubKey ed25519.PublicKey
}

// NewSVTNManager constructs a SVTNManager with the given AdmittedKeySet and
// control node public key. The control node's public key is registered as a
// control-role key when a new SVTN is created (BC-2.07.001 postcondition 2).
//
// No init() functions — all dependencies injected (go.md rule 10).
func NewSVTNManager(ks *admission.AdmittedKeySet, controlPubKey ed25519.PublicKey) *SVTNManager {
	// Deep-clone the public key so SVTNManager's copy is independent of the caller's slice.
	cloned := append(ed25519.PublicKey(nil), controlPubKey...)
	return &SVTNManager{
		svtns:         make(map[string]SVTN),
		keySet:        ks,
		controlPubKey: cloned,
	}
}

// keyFingerprint computes the "SHA256:<base64>" fingerprint of an Ed25519
// public key (DI-002; BC-2.05.004 invariant 2; VP-046).
// Only the public key bytes are hashed — no private material.
func keyFingerprint(pubkey ed25519.PublicKey) string {
	digest := sha256.Sum256(pubkey)
	return "SHA256:" + base64.StdEncoding.EncodeToString(digest[:])
}

// Create creates a new SVTN with a generated SVTN-ID and bootstraps the first
// control key locally. The control node's public key is added to the
// AdmittedKeySet as a control-role key without a network admission round-trip
// (the local bootstrap is the trust anchor per BC-2.07.001 postcondition 2).
//
// Returns ErrSVTNAlreadyExists (E-SVTN-001) if a SVTN with svtnName already
// exists (BC-2.07.001 EC-001; DI-005).
//
// Duplicate-name check (F-CS-003): the existence check is performed BEFORE
// the bootstrap RegisterKey call. This prevents orphan AdmittedKey entries
// under concurrent Create calls for the same name: if the SVTN already exists
// we return ErrSVTNAlreadyExists immediately without touching keySet.
//
// Ordering invariant (BC-2.07.001 PC-1+PC-2 composite postcondition): after
// the existence check confirms the name is available, the bootstrap control
// key is registered in the AdmittedKeySet BEFORE the SVTN is published to
// m.svtns. This prevents a concurrent caller from observing a half-bootstrapped
// SVTN and injecting a foreign control key via last-write-wins (ADR-003). The
// key is bound to the un-published SVTN ID and is therefore inert until the
// SVTN appears in m.svtns.
//
// Traces to BC-2.07.001 postcondition 1.
func (m *SVTNManager) Create(svtnName string) (CreateResult, error) {
	// Generate SVTN ID before acquiring the lock to keep the critical section
	// short and avoid holding the mutex during a syscall (CR-005).
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return CreateResult{}, fmt.Errorf("generate SVTN ID: %w", err)
	}

	// Duplicate-name check BEFORE bootstrap RegisterKey (F-CS-003).
	// Acquire the lock, check for duplicate name, and release. If the name is
	// taken, return early without touching keySet — no orphan keys produced.
	m.mu.Lock()
	_, exists := m.svtns[svtnName]
	m.mu.Unlock()
	if exists {
		return CreateResult{}, ErrSVTNAlreadyExists
	}

	// Bootstrap the control key BEFORE publishing the SVTN (F-003 ordering
	// invariant). The key is bound to the un-published ID, so it is inert until
	// the SVTN appears in m.svtns. AdmittedKeySet owns its own mutex — no nested
	// lock ordering concern. The window between the existence check above and the
	// publish below is closed by the re-check in the publish lock (see below).
	m.keySet.RegisterKey(id, m.controlPubKey, admission.RoleControl)

	// Publish the SVTN under the lock. Re-check for duplicate name in case a
	// concurrent Create won the race between our existence check and this point.
	// If a race was lost, the bootstrap key above is orphaned (keyed by a
	// never-published SVTN ID) — harmless, no SVTN is ever exposed for that ID.
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.svtns[svtnName]; exists {
		return CreateResult{}, ErrSVTNAlreadyExists
	}

	svtn := SVTN{
		ID:        id,
		Name:      svtnName,
		CreatedAt: time.Now().UTC(),
	}
	m.svtns[svtnName] = svtn
	return CreateResult{SVTN: svtn}, nil
}

// RegisterKey registers a public key for the named SVTN with the given role.
// The key is added to the AdmittedKeySet immediately and becomes active for
// admission challenges (BC-2.05.004 postcondition 1).
//
// Last-write-wins semantics per ADR-003: registering an already-registered
// key updates its role (EC-001 in S-6.02 edge cases).
//
// Returns ErrSVTNNotFound if svtnName does not exist.
//
// Traces to BC-2.05.004 postcondition 1.
func (m *SVTNManager) RegisterKey(
	svtnName string,
	pubkey ed25519.PublicKey,
	role admission.KeyRole,
) (KeyOpResult, error) {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()

	if !exists {
		return KeyOpResult{}, ErrSVTNNotFound
	}

	m.keySet.RegisterKey(svtn.ID, pubkey, role)

	return KeyOpResult{
		Fingerprint: keyFingerprint(pubkey),
		At:          time.Now().UTC(),
	}, nil
}

// RevokeKey removes the public key from the admission set for the named SVTN.
// Existing sessions using the key continue until the next re-authentication
// challenge (propagation delay per FM-007; BC-2.05.004 postcondition 2).
//
// Control-to-control revocation requires confirm=true; passing confirm=false
// returns ErrControlRevocationRequiresConfirm (E-ADM-018; BC-2.05.004
// precondition 1; ADR-004; AC-005).
//
// Returns ErrSVTNNotFound if svtnName does not exist.
// Returns admission.ErrKeyNotRegistered (E-ADM-013) if the key is not
// registered (S-6.02 EC-002).
// Returns admission.ErrRoleMismatch (E-ADM-019) if currentRole does not match
// the stored role — prevents bypassing the confirm gate by supplying a lower role.
//
// Traces to BC-2.05.004 postcondition 2; HOLD-001; ARCH-04 v1.10.
func (m *SVTNManager) RevokeKey(
	svtnName string,
	pubkey ed25519.PublicKey,
	currentRole admission.KeyRole,
	confirm bool,
) (KeyOpResult, error) {
	// Guard: never allow the bootstrap control key to be revoked.
	// The bootstrap key is the trust anchor for the SVTN — removing it would
	// leave the SVTN without a control authority (E-ADM-020; bootstrap revocability
	// invariant). Constant-time comparison prevents timing oracle (Inv-5).
	if subtle.ConstantTimeCompare([]byte(pubkey), []byte(m.controlPubKey)) == 1 {
		return KeyOpResult{}, ErrBootstrapKeyRevokeForbidden
	}

	// Step 1: validate SVTN exists.
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()

	if !exists {
		return KeyOpResult{}, ErrSVTNNotFound
	}

	svtnID := svtn.ID

	// Step 2: atomic role cross-check + confirm-gate + revocation under a single
	// write lock (HOLD-001; ARCH-04 v1.10 Addendum H2). RevokeKeyIfRoleMatches
	// atomically:
	//   (a) looks up the key by pubkey
	//   (b) compares stored role to currentRole; returns ErrRoleMismatch (E-ADM-019)
	//       if stored role differs — prevents confirm-gate bypass via lower-role claim
	//   (c) enforces the confirm gate on the STORED role (existingRole), not the
	//       caller-supplied currentRole — uses admission.ErrControlRevocationRequiresConfirm
	//       (E-ADM-018) without revoking the key if confirm=false
	//   (d) marks the key revoked on success
	//
	// The confirm gate fires inside the atomic primitive (ARCH-04 H2 ordering:
	// atomic-FIRST, confirm-gate-SECOND). No intermediate state is observable:
	// if ErrControlRevocationRequiresConfirm is returned, the key is unchanged.
	//
	// ADR-004; BC-2.05.004 precondition 1; AC-005.
	_, err := m.keySet.RevokeKeyIfRoleMatches(svtnID, pubkey, currentRole, confirm)
	if err != nil {
		// ErrKeyNotRegistered (E-ADM-013), ErrRoleMismatch (E-ADM-019), or
		// ErrControlRevocationRequiresConfirm (E-ADM-018) returned as-is;
		// callers inspect via errors.Is.
		return KeyOpResult{}, err
	}

	// Step 4: return result with fingerprint and UTC timestamp.
	return KeyOpResult{
		Fingerprint: keyFingerprint(pubkey),
		At:          time.Now().UTC(),
	}, nil
}

// ExpireKey sets a TTL on the key for the named SVTN. After the TTL elapses,
// the key behaves as revoked — routers stop admitting it on the next
// re-authentication challenge (BC-2.05.004 postcondition 3).
//
// Returns ErrInvalidDuration (E-CFG-001) if ttl is zero or negative (S-6.02
// EC-003).
// Returns ErrSVTNNotFound if svtnName does not exist.
// Returns admission.ErrKeyNotRegistered (E-ADM-013) if the key is not
// registered.
//
// Traces to BC-2.05.004 postcondition 3.
func (m *SVTNManager) ExpireKey(
	svtnName string,
	pubkey ed25519.PublicKey,
	ttl time.Duration,
) (KeyOpResult, error) {
	if ttl <= 0 {
		return KeyOpResult{}, ErrInvalidDuration
	}

	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()

	if !exists {
		return KeyOpResult{}, ErrSVTNNotFound
	}

	svtnID := svtn.ID

	// LookupByPubkey derives the node address internally (ARCH-04 v1.10).
	lookedUp := m.keySet.LookupByPubkey(svtnID, pubkey)
	if lookedUp == nil {
		return KeyOpResult{}, admission.ErrKeyNotRegistered
	}

	expiry := time.Now().UTC().Add(ttl)
	if err := m.keySet.SetKeyExpiry(svtnID, lookedUp.NodeAddr, expiry); err != nil {
		// SetKeyExpiry returns ErrKeyNotRegistered if key not present (E-ADM-013).
		return KeyOpResult{}, err
	}

	return KeyOpResult{
		Fingerprint: keyFingerprint(pubkey),
		At:          time.Now().UTC(),
	}, nil
}

// KeySummary is an element in the list returned by ListKeys.
// It carries only the public portion of the key entry — no private material
// is included (DI-002; BC-2.05.004 PC-4).
type KeySummary struct {
	// Fingerprint is the SHA-256 fingerprint of the key in "SHA256:<base64>" form.
	Fingerprint string
	// Role is the authorization role of this key on the SVTN.
	Role admission.KeyRole
	// Expiry is the optional expiry time (zero means no expiry is set).
	Expiry time.Time
}

// CallerKeyRole returns the KeyRole of pubkey in the named SVTN, and true.
// Returns (0, false) if svtnName does not exist or the key is not registered.
// Used by admin handlers to resolve the authenticated caller's role server-side
// without trusting the client-supplied caller_role field (F-001b / BC-2.07.001 Inv-3).
// Safe for concurrent use.
func (m *SVTNManager) CallerKeyRole(svtnName string, pubkey ed25519.PublicKey) (admission.KeyRole, bool) {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()
	if !exists {
		return 0, false
	}
	entry := m.keySet.LookupByPubkey(svtn.ID, pubkey)
	if entry == nil {
		return 0, false
	}
	return entry.Role, true
}

// ListKeys returns a snapshot of all registered keys for the named SVTN.
// Returns ErrSVTNNotFound if svtnName does not exist.
// An empty slice (not nil) is returned when no keys are registered (EC-003).
//
// Traces to BC-2.05.004 postcondition 1 (key remains admitted until revoked
// or expired; listing reflects current state).
func (m *SVTNManager) ListKeys(svtnName string) ([]KeySummary, error) {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrSVTNNotFound
	}

	entries := m.keySet.ListBySVTN(svtn.ID)
	out := make([]KeySummary, 0, len(entries))
	for _, e := range entries {
		out = append(out, KeySummary{
			Fingerprint: keyFingerprint(e.PublicKey),
			Role:        e.Role,
			Expiry:      e.KeyExpiry(),
		})
	}
	return out, nil
}
