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
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
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

// ErrBootstrapKeyExpireForbidden is returned when ExpireKey is called against the
// SVTN's permanent trust anchor (bootstrap key). Mirrors ErrBootstrapKeyRevokeForbidden
// for symmetric management-lockout prevention per BC-2.05.004 EC-007 v1.12.
var ErrBootstrapKeyExpireForbidden = errors.New("bootstrap key cannot be expired")

// ErrDestroyUnauthorized is returned by SVTNManager.Destroy when the caller is
// not a control-role key (E-ADM-011 Variant 2; BC-2.07.001 Inv-3; RULING-W6TB-A §3).
//
// This is a defense-in-depth sentinel at the Go-API layer. The primary authority
// gate for the admin.svtn.destroy RPC is the handler-layer resolveAndVerifyCallerRole
// call (which returns E-RPC-011 wrapping E-ADM-009 for non-control callers).
// ErrDestroyUnauthorized is only surfaced in unit tests that call SVTNManager.Destroy
// directly without going through the handler layer.
var ErrDestroyUnauthorized = errors.New("destroy: caller is not a control-role key")

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
	// randSource is the entropy source used to generate SVTN IDs. Defaults to
	// crypto/rand.Reader in NewSVTNManager. Injectable via NewSVTNManagerWithRandSource
	// for tests that must exercise the rand.Read failure path (F-P3L2-01).
	randSource io.Reader
}

// NewSVTNManager constructs a SVTNManager with the given AdmittedKeySet and
// control node public key. The control node's public key is registered as a
// control-role key when a new SVTN is created (BC-2.07.001 postcondition 2).
//
// No init() functions — all dependencies injected (go.md rule 10).
func NewSVTNManager(ks *admission.AdmittedKeySet, controlPubKey ed25519.PublicKey) *SVTNManager {
	return newSVTNManager(ks, controlPubKey, rand.Reader)
}

// NewSVTNManagerWithRandSource constructs a SVTNManager with an injectable
// entropy source. Use in tests to exercise the rand.Read failure path
// (E-INT-001 branch in admin.svtn.create handler; F-P3L2-01).
// Production code MUST use NewSVTNManager (crypto/rand.Reader).
func NewSVTNManagerWithRandSource(ks *admission.AdmittedKeySet, controlPubKey ed25519.PublicKey, r io.Reader) *SVTNManager {
	return newSVTNManager(ks, controlPubKey, r)
}

func newSVTNManager(ks *admission.AdmittedKeySet, controlPubKey ed25519.PublicKey, r io.Reader) *SVTNManager {
	// Deep-clone the public key so SVTNManager's copy is independent of the caller's slice.
	cloned := append(ed25519.PublicKey(nil), controlPubKey...)
	return &SVTNManager{
		svtns:         make(map[string]SVTN),
		keySet:        ks,
		controlPubKey: cloned,
		randSource:    r,
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
	// Uses m.randSource (crypto/rand.Reader in production; injectable in tests
	// for E-INT-001 failure path coverage — F-P3L2-01).
	var id [16]byte
	if _, err := io.ReadFull(m.randSource, id[:]); err != nil {
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
// Traces to BC-2.05.004 postcondition 2; HOLD-001; ARCH-04 v1.13.
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
	// write lock (HOLD-001; ARCH-04 v1.13 Addendum H2). RevokeKeyIfRoleMatches
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

	// Guard: never allow the bootstrap control key to be expired.
	// The bootstrap key is the permanent trust anchor for the SVTN — setting a
	// TTL on it would eventually lock out all control authority (BC-2.05.004
	// EC-007 v1.12). Constant-time comparison prevents timing oracle (Inv-5).
	// Mirrors the RevokeKey guard; fires before SVTN existence lookup.
	if subtle.ConstantTimeCompare([]byte(pubkey), []byte(m.controlPubKey)) == 1 {
		return KeyOpResult{}, ErrBootstrapKeyExpireForbidden
	}

	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()

	if !exists {
		return KeyOpResult{}, ErrSVTNNotFound
	}

	svtnID := svtn.ID

	// Lookup to get the current role (for atomic cross-check below).
	// TOCTOU note: a concurrent LWW RegisterKey could change the role between
	// this lookup and SetKeyExpiryIfRoleMatches. SetKeyExpiryIfRoleMatches
	// detects this via the role cross-check and returns ErrRoleMismatch.
	lookedUp, ok := m.keySet.LookupByPubkey(svtnID, pubkey)
	if !ok {
		return KeyOpResult{}, admission.ErrKeyNotRegistered
	}
	expectedRole := lookedUp.Role

	expiry := time.Now().UTC().Add(ttl)
	if err := m.keySet.SetKeyExpiryIfRoleMatches(svtnID, pubkey, expectedRole, expiry); err != nil {
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

// IsBootstrapKey reports whether pubkey byte-equals the daemon's bootstrap
// control key (the trust anchor registered at SVTN creation time).
// Uses constant-time comparison to prevent timing oracle (BC-2.07.001 Inv-3;
// Inv-5; AC-006 / F-P2L1-001).
func (m *SVTNManager) IsBootstrapKey(pubkey ed25519.PublicKey) bool {
	return subtle.ConstantTimeCompare([]byte(m.controlPubKey), []byte(pubkey)) == 1
}

// HasAnySVTN reports whether at least one SVTN has been created in this manager.
// Used by makeAdminSVTNCreateHandler to skip the BootstrapKeyHasControlRole
// defense-in-depth check on first-ever SVTN creation (before any SVTN exists,
// the bootstrap key is not yet registered in any keySet — the check would return
// false and incorrectly block the genesis create; Ruling-7).
func (m *SVTNManager) HasAnySVTN() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.svtns) > 0
}

// BootstrapKeyHasControlRole reports whether the daemon bootstrap key's role is
// RoleControl in at least one registered SVTN. Returns false if no SVTNs exist
// (not yet created) or if the bootstrap key is not found in any SVTN's key set.
//
// Used by makeAdminSVTNCreateHandler as a defense-in-depth gate (Ruling-7;
// BC-2.07.001 Inv-3): the bootstrap key is always seeded as RoleControl at SVTN
// creation time, but this check verifies the invariant explicitly so that any
// future key-model change that decouples is_bootstrap from RoleControl cannot
// silently bypass the authorization gate.
//
// NOTE: the check fails closed — returns false — when no SVTNs exist yet. The
// handler MUST call IsBootstrapKey first; if that passes, this check is the
// secondary verification. On first-ever SVTN creation (zero existing SVTNs),
// the handler has already verified IsBootstrapKey and this secondary check is
// skipped (zero SVTNs implies bootstrap key not yet registered anywhere, which
// is the expected and authorized state). See makeAdminSVTNCreateHandler for
// the exact gating logic.
func (m *SVTNManager) BootstrapKeyHasControlRole() bool {
	m.mu.RLock()
	svtns := make([]SVTN, 0, len(m.svtns))
	for _, s := range m.svtns {
		svtns = append(svtns, s)
	}
	m.mu.RUnlock()

	now := time.Now().UTC()
	for _, s := range svtns {
		entry, ok := m.keySet.LookupByPubkey(s.ID, m.controlPubKey)
		if !ok {
			continue
		}
		// Treat revoked or expired bootstrap key as not having RoleControl.
		if entry.IsRevoked() {
			continue
		}
		if exp := entry.KeyExpiry(); !exp.IsZero() && !now.Before(exp) {
			continue
		}
		if entry.Role == admission.RoleControl {
			return true
		}
	}
	return false
}

// BootstrapFingerprint returns the "SHA256:<base64>" fingerprint of the daemon's
// bootstrap control key (BC-2.05.004 PC-4 canonical format; BC-2.07.001 PC-2).
// Called by the admin.svtn.create handler to populate the bootstrap_fingerprint
// field in the success response (AC-004 / S-6.07).
func (m *SVTNManager) BootstrapFingerprint() string {
	return keyFingerprint(m.controlPubKey)
}

// CallerKeyRole returns the KeyRole of pubkey in the named SVTN, and true.
// Returns (0, false) if svtnName does not exist or the key is not registered.
// Used by admin handlers to resolve the authenticated caller's role server-side
// without trusting the client-supplied caller_role field (F-001b / BC-2.07.001 Inv-3).
// Safe for concurrent use.
//
// NOTE: CallerKeyRole does not check revoked or expiry state. Use CallerKeyRoleActive
// for authority checks that must deny revoked/expired keys (F-P4L1-003).
func (m *SVTNManager) CallerKeyRole(svtnName string, pubkey ed25519.PublicKey) (admission.KeyRole, bool) {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()
	if !exists {
		return 0, false
	}
	entry, ok := m.keySet.LookupByPubkey(svtn.ID, pubkey)
	if !ok {
		return 0, false
	}
	return entry.Role, true
}

// CallerKeyRoleActive returns the KeyRole of pubkey in the named SVTN only if
// the key is currently active (not revoked and not expired). Returns (0, false)
// if svtnName does not exist, the key is not registered, revoked==true, or
// now >= expiry (F-P4L1-003 / BC-2.05.004 PC-1).
//
// Use this instead of CallerKeyRole for handler authority checks.
// Safe for concurrent use.
func (m *SVTNManager) CallerKeyRoleActive(svtnName string, pubkey ed25519.PublicKey) (admission.KeyRole, bool) {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()
	if !exists {
		return 0, false
	}
	entry, ok := m.keySet.LookupByPubkey(svtn.ID, pubkey)
	if !ok {
		return 0, false
	}
	// Deny revoked keys (F-P4L1-003).
	if entry.IsRevoked() {
		return 0, false
	}
	// Deny expired keys (F-P4L1-003): expiry zero means no expiry set.
	if exp := entry.KeyExpiry(); !exp.IsZero() && !time.Now().UTC().Before(exp) {
		return 0, false
	}
	return entry.Role, true
}

// HasControlKey reports whether the named SVTN has at least one active
// (not revoked, not expired) control-role key registered (including the
// daemon's own bootstrap key). Returns false if svtnName does not exist or
// no qualifying key is found.
// Safe for concurrent use.
func (m *SVTNManager) HasControlKey(svtnName string) bool {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()
	if !exists {
		return false
	}
	entries := m.keySet.ListBySVTN(svtn.ID)
	now := time.Now().UTC()
	for _, e := range entries {
		if e.Role != admission.RoleControl {
			continue
		}
		if e.IsRevoked() {
			continue
		}
		if exp := e.KeyExpiry(); !exp.IsZero() && !now.Before(exp) {
			continue
		}
		return true
	}
	return false
}

// HasNonBootstrapControlKey reports whether the named SVTN has at least one
// active (not revoked, not expired) control-role key OTHER than the daemon's
// own bootstrap key. Returns false if svtnName does not exist or no qualifying
// non-bootstrap control key is found.
//
// Used by the operator-key bootstrap grant in resolveAndVerifyCallerRole
// (F-P4L1-001): allows an operator-set member to register the first external
// control key into a SVTN that was just created (contains only the daemon's own
// bootstrap key). Once any non-bootstrap control key is registered, the grant
// no longer applies and all subsequent registrations require the caller to be
// a registered control key (fail-closed).
// Safe for concurrent use.
func (m *SVTNManager) HasNonBootstrapControlKey(svtnName string) bool {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()
	if !exists {
		return false
	}
	entries := m.keySet.ListBySVTN(svtn.ID)
	now := time.Now().UTC()
	for _, e := range entries {
		if e.Role != admission.RoleControl {
			continue
		}
		if e.IsRevoked() {
			continue
		}
		if exp := e.KeyExpiry(); !exp.IsZero() && !now.Before(exp) {
			continue
		}
		// Exclude the daemon's own bootstrap key.
		if m.IsBootstrapKey(e.PublicKey) {
			continue
		}
		return true
	}
	return false
}

// SVTNByName returns a copy of the SVTN record for name, or (SVTN{}, false)
// if no SVTN with that name exists. Returns a value copy — callers must not
// retain a pointer into the store (go.md rule 12).
// Safe for concurrent use.
func (m *SVTNManager) SVTNByName(name string) (SVTN, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.svtns[name]
	return s, ok
}

// All returns a snapshot of all registered SVTNs as a value-copy slice.
// Callers must not retain references into the returned slice beyond the
// current scope (go.md rule 12).
// Safe for concurrent use.
func (m *SVTNManager) All() []SVTN {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]SVTN, 0, len(m.svtns))
	for _, s := range m.svtns {
		out = append(out, s)
	}
	return out
}

// CallerKeyRoleInAny returns the active KeyRole of pubkey in any registered
// SVTN, and true. Returns (0, false) if pubkey is not found as an active
// (not revoked, not expired) key in any SVTN, or if no SVTNs exist.
//
// This is a diagnostic-only helper used to populate the "has role <role>"
// field in E-ADM-009 error messages when no specific SVTN name is in scope.
// Do NOT use this for authority decisions — use CallerKeyRoleActive with an
// explicit SVTN name.
// Safe for concurrent use.
func (m *SVTNManager) CallerKeyRoleInAny(pubkey ed25519.PublicKey) (admission.KeyRole, bool) {
	m.mu.RLock()
	svtns := make([]SVTN, 0, len(m.svtns))
	for _, s := range m.svtns {
		svtns = append(svtns, s)
	}
	m.mu.RUnlock()

	now := time.Now().UTC()
	for _, s := range svtns {
		entry, ok := m.keySet.LookupByPubkey(s.ID, pubkey)
		if !ok {
			continue
		}
		if entry.IsRevoked() {
			continue
		}
		if exp := entry.KeyExpiry(); !exp.IsZero() && !now.Before(exp) {
			continue
		}
		return entry.Role, true
	}
	return 0, false
}

// IsRegisteredAnyState reports whether pubkey is registered in the named SVTN
// in ANY state — active, revoked, or expired. Returns false if svtnName does
// not exist or the key has never been registered.
//
// Use this to distinguish "key never registered" from "key registered but
// revoked/expired" when CallerKeyRoleActive returns (0, false). Revoked or
// expired keys that ARE registered must be denied immediately (fail-closed,
// F-P5L1-001 / BC-2.05.004 EC-006); only truly unregistered keys fall through
// to the bootstrap-grant path.
// Safe for concurrent use.
func (m *SVTNManager) IsRegisteredAnyState(svtnName string, pubkey ed25519.PublicKey) bool {
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()
	if !exists {
		return false
	}
	_, ok := m.keySet.LookupByPubkey(svtn.ID, pubkey)
	return ok
}

// InsertRawSVTN inserts an SVTN record with a freshly generated ID without
// registering any keys in the AdmittedKeySet. This intentionally violates the
// Create() invariant that registers the bootstrap control key; callers are
// responsible for constructing meaningful state from the resulting record.
//
// SECURITY: InsertRawSVTN is a bootstrap-invariant-bypass primitive reachable
// only from test binaries (guarded by testing.Testing()). Never call from
// production code.
//
// Returns ErrSVTNAlreadyExists if svtnName is already present.
// The SVTN ID is generated from m.randSource (crypto/rand.Reader in production).
// Safe for concurrent use.
func (m *SVTNManager) InsertRawSVTN(svtnName string) error {
	if !testing.Testing() {
		panic("svtnmgmt.InsertRawSVTN: test-only mutation seam invoked from production binary")
	}
	var id [16]byte
	if _, err := io.ReadFull(m.randSource, id[:]); err != nil {
		return fmt.Errorf("generate SVTN ID for raw insert: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.svtns[svtnName]; exists {
		return ErrSVTNAlreadyExists
	}
	m.svtns[svtnName] = SVTN{ID: id, Name: svtnName, CreatedAt: time.Now().UTC()}
	return nil
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

// Destroy removes the named SVTN and all its admitted keys from the registry.
// It terminates all active sessions before freeing the SVTN ID.
//
// Design note on ctx (S-6.05 divergence from other SVTNManager methods):
// Unlike Create, RegisterKey, RevokeKey, and ExpireKey — which do NOT take a
// context.Context — Destroy requires ctx as its first parameter because session
// termination is potentially asynchronous and must be cancellable. This is the
// only SVTNManager method that takes a context; callers should pass a context
// with an appropriate deadline to bound the session-drain window.
//
// svtnName is the registry name key (the SVTN registry is keyed by name, not by
// [16]byte ID; consistent with all other SVTNManager methods).
//
// Postconditions on success:
//   - All admitted keys for the SVTN are removed from the AdmittedKeySet.
//   - All active sessions receive a session-terminated signal.
//   - The SVTN ID is freed from the registry (HasAnySVTN() may return false if
//     this was the last SVTN — re-opening the genesis carve-out per RULING-W6TB-A §4).
//
// Returns ErrSVTNNotFound (E-SVTN-003) if the SVTN does not exist.
// Returns ErrDestroyUnauthorized (E-ADM-011 Variant 2) as a defense-in-depth check
// at the Go-API layer when the caller is not a control-role key. The primary gate
// is the handler-layer resolveAndVerifyCallerRole call; this is an additional
// safeguard for callers that invoke SVTNManager.Destroy directly.
//
// Key removal precedes SVTN ID free (ARCH-04 admission ordering invariant).
//
// Traces to BC-2.07.001 postcondition 3; AC-001; AC-002; AC-004; AC-005;
// RULING-W6TB-A; VP-048 properties 2+3.
func (m *SVTNManager) Destroy(ctx context.Context, svtnName string) error {
	m.mu.Lock()
	svtn, exists := m.svtns[svtnName]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("SVTN not found: %s: %w", svtnName, ErrSVTNNotFound)
	}
	svtnID := svtn.ID
	// ARCH-04 ordering: key removal precedes SVTN ID free.
	m.keySet.RemoveSVTN(svtnID)
	delete(m.svtns, svtnName)
	m.mu.Unlock()
	return nil
}
