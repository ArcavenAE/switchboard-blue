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

// ErrControlRevocationRequiresConfirm is returned when a control-to-control
// key revocation is attempted without the --confirm flag (E-ADM-004;
// BC-2.05.004 invariant 1; ADR-004; AC-005).
var ErrControlRevocationRequiresConfirm = errors.New(
	"control-to-control revocation requires --confirm flag (ADR-004)",
)

// ErrRoleMismatch is returned by SVTNManager.RevokeKey when the caller-supplied
// currentRole does not match the role stored in the AdmittedKeySet registry
// (E-ADM-014). This prevents the confirm gate from being bypassed by supplying
// a lower role for a control key.
var ErrRoleMismatch = errors.New("revoke: supplied role does not match registered role")

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
// Traces to BC-2.07.001 postcondition 1.
func (m *SVTNManager) Create(svtnName string) (CreateResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.svtns[svtnName]; exists {
		return CreateResult{}, ErrSVTNAlreadyExists
	}

	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return CreateResult{}, fmt.Errorf("generate SVTN ID: %w", err)
	}

	svtn := SVTN{
		ID:        id,
		Name:      svtnName,
		CreatedAt: time.Now().UTC(),
	}
	m.svtns[svtnName] = svtn

	// BC-2.07.001 postcondition 2: bootstrap the control node's public key as
	// the first admitted control-role key (local operation — trust anchor).
	m.keySet.RegisterKey(id, m.controlPubKey, admission.RoleControl)

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
// returns ErrControlRevocationRequiresConfirm (BC-2.05.004 invariant 1;
// ADR-004; AC-005).
//
// Returns ErrSVTNNotFound if svtnName does not exist.
// Returns admission.ErrKeyNotRegistered (E-ADM-013) if the key is not
// registered (S-6.02 EC-002).
//
// Traces to BC-2.05.004 postcondition 2.
func (m *SVTNManager) RevokeKey(
	svtnName string,
	pubkey ed25519.PublicKey,
	currentRole admission.KeyRole,
	confirm bool,
) (KeyOpResult, error) {
	// Step 1: validate SVTN exists.
	m.mu.RLock()
	svtn, exists := m.svtns[svtnName]
	m.mu.RUnlock()

	if !exists {
		return KeyOpResult{}, ErrSVTNNotFound
	}

	svtnID := svtn.ID

	// Step 2: look up the key in the AdmittedKeySet (HOLD-001 hybrid: cross-check
	// stored role against caller-supplied currentRole before the confirm gate).
	// LookupByPubkey derives the node address internally (ARCH-04 v1.8).
	stored := m.keySet.LookupByPubkey(svtnID, pubkey)
	if stored == nil {
		return KeyOpResult{}, admission.ErrKeyNotRegistered
	}

	// Step 3: cross-check role (ARCH-04 v1.7 HOLD-001 resolution: hybrid approach).
	// The caller supplies currentRole; the manager verifies it matches stored.Role.
	// This prevents bypassing the confirm gate by supplying a lower role.
	if stored.Role != currentRole {
		return KeyOpResult{}, ErrRoleMismatch
	}

	// Step 4: control-to-control revocation requires confirm=true (ADR-004;
	// BC-2.05.004 invariant 1; AC-005).
	if currentRole == admission.RoleControl && !confirm {
		return KeyOpResult{}, ErrControlRevocationRequiresConfirm
	}

	// Step 5: revoke the key using the node address from the stored entry.
	if err := m.keySet.RevokeKey(svtnID, stored.NodeAddr); err != nil {
		return KeyOpResult{}, err
	}

	// Step 6: return result with fingerprint and UTC timestamp.
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

	// LookupByPubkey derives the node address internally (ARCH-04 v1.8).
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
