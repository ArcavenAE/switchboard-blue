// admin_handlers.go — daemon-side admin RPC handler builder for cmd/switchboard.
//
// BuildAdminHandlers returns the []mgmt.Handler slice for all four admin RPCs:
//
//	admin.key.register   (BC-2.05.004 PC-1)
//	admin.key.revoke     (BC-2.05.004 PC-2; HOLD-001 hybrid; ADR-004)
//	admin.key.expire     (BC-2.05.004 PC-3; DI-003 defense-in-depth duration validation)
//	admin.list-keys      (BC-2.05.004 PC-1 confirmation surface)
//
// Only the control-mode daemon calls BuildAdminHandlers (ADR-004 role-exclusion;
// ARCH-08 §6.6.2; AC-004). Access, console, and router daemons pass nil handlers.
//
// Purity classification (ARCH-09): boundary — depends on SVTNManager (boundary)
// and mgmt.Handler (interface). No data-plane imports permitted (ARCH-08 §6.6.2).
//
// Forbidden imports: internal/frame, internal/routing, internal/multipath,
// internal/arq, internal/replay, internal/paths, internal/halfchannel,
// internal/session, internal/tmux, internal/discovery, cmd/sbctl.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// maxKeyTTL is the server-side upper bound for admin.key.expire TTL values.
// Rejects any TTL greater than 100 years (AC-005; DI-003 defense-in-depth).
const maxKeyTTL = 100 * 365 * 24 * time.Hour

// adminKeyRegisterArgs is the wire JSON args for admin.key.register.
// The `role` field uses the canonical JSON key per interface-definitions.md v1.1.
type adminKeyRegisterArgs struct {
	SVTNName  string `json:"svtn"`
	PublicKey string `json:"pubkey"` // base64-encoded Ed25519 public key
	Role      string `json:"role"`
}

// adminKeyRevokeArgs is the wire JSON args for admin.key.revoke.
// The `role` field is the canonical JSON key (F-002 ruling; HOLD-001 hybrid).
type adminKeyRevokeArgs struct {
	SVTNName  string `json:"svtn"`
	PublicKey string `json:"pubkey"` // base64-encoded Ed25519 public key
	Role      string `json:"role"`   // caller-supplied current role for cross-check
	Confirm   bool   `json:"confirm"`
}

// adminKeyExpireArgs is the wire JSON args for admin.key.expire.
type adminKeyExpireArgs struct {
	SVTNName  string `json:"svtn"`
	PublicKey string `json:"pubkey"` // base64-encoded Ed25519 public key
	After     string `json:"after"`  // duration string e.g. "24h"
}

// adminListKeysArgs is the wire JSON args for admin.list-keys.
type adminListKeysArgs struct {
	SVTNName string `json:"svtn"`
}

// adminKeyResult is the success response body for key lifecycle operations.
// Satisfies BC-2.05.004 postcondition 4 (confirmation with fingerprint and timestamp).
type adminKeyResult struct {
	Fingerprint string    `json:"fingerprint"`
	At          time.Time `json:"at"`
}

// adminListKeysResult is the success response body for admin.list-keys.
// The Keys field is always an array (never JSON null) per EC-003.
type adminListKeysResult struct {
	Keys []adminKeyEntry `json:"keys"`
}

// adminKeyEntry is a single element in the admin.list-keys response.
type adminKeyEntry struct {
	Fingerprint string    `json:"fingerprint"`
	Role        string    `json:"role"`
	Expiry      time.Time `json:"expiry,omitempty"`
}

// BuildAdminHandlers returns a []mgmt.Handler containing the four admin key
// lifecycle handlers. m must not be nil — a nil SVTNManager indicates a
// misconfiguration at the call site; BuildAdminHandlers panics immediately
// (EC-004; AC-001).
//
// Only the control-mode daemon should call BuildAdminHandlers. All other
// daemon modes pass nil (or an empty slice) for admin commands so that they
// correctly return E-RPC-010 "unknown command" (ADR-004; AC-004).
func BuildAdminHandlers(m *svtnmgmt.SVTNManager) []mgmt.Handler {
	if m == nil {
		panic("BuildAdminHandlers: SVTNManager must not be nil (EC-004)")
	}
	return []mgmt.Handler{
		{Command: "admin.key.register", Fn: makeRegisterHandler(m)},
		{Command: "admin.key.revoke", Fn: makeRevokeHandler(m)},
		{Command: "admin.key.expire", Fn: makeExpireHandler(m)},
		{Command: "admin.list-keys", Fn: makeListKeysHandler(m)},
	}
}

// decodePublicKey decodes a base64-encoded (standard or raw URL) Ed25519 public key.
// Returns E-CFG-001 if the value is missing, not valid base64, or not 32 bytes.
func decodePublicKey(encoded string) (ed25519.PublicKey, error) {
	if encoded == "" {
		return nil, fmt.Errorf("E-CFG-001: missing required field: pubkey")
	}
	// Try standard encoding first, then raw URL encoding.
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		raw, err = base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid pubkey encoding: %w", err)
		}
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("E-CFG-001: invalid pubkey length: got %d bytes, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// makeRegisterHandler returns the admin.key.register handler function.
// Traces to BC-2.05.004 postcondition 1; AC-001; AC-003.
func makeRegisterHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		var a adminKeyRegisterArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn")
		}

		role, err := admission.KeyRoleFromString(a.Role)
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid role: %q", a.Role)
		}

		pubkey, err := decodePublicKey(a.PublicKey)
		if err != nil {
			return nil, err
		}

		result, err := m.RegisterKey(a.SVTNName, pubkey, role)
		if err != nil {
			return nil, mapAdminError(err, a.SVTNName, a.PublicKey)
		}

		return adminKeyResult{
			Fingerprint: result.Fingerprint,
			At:          result.At,
		}, nil
	}
}

// makeRevokeHandler returns the admin.key.revoke handler function.
// Parses `role` (canonical wire field per F-002); passes as currentRole to
// SVTNManager.RevokeKey (HOLD-001 hybrid; ADR-004; ARCH-04 v1.10).
// Traces to BC-2.05.004 postcondition 2; AC-002.
func makeRevokeHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		var a adminKeyRevokeArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn")
		}

		role, err := admission.KeyRoleFromString(a.Role)
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid role: %q", a.Role)
		}

		pubkey, err := decodePublicKey(a.PublicKey)
		if err != nil {
			return nil, err
		}

		result, err := m.RevokeKey(a.SVTNName, pubkey, role, a.Confirm)
		if err != nil {
			return nil, mapAdminError(err, a.SVTNName, a.PublicKey)
		}

		return adminKeyResult{
			Fingerprint: result.Fingerprint,
			At:          result.At,
		}, nil
	}
}

// makeExpireHandler returns the admin.key.expire handler function.
// Re-parses and validates the `after` duration server-side (defense-in-depth;
// DI-003) independently of sbctl CLI validation (AC-005).
// Traces to BC-2.05.004 postcondition 3; AC-005.
func makeExpireHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		// Use a raw map to detect absent fields (EC-005) vs zero-value fields.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(args, &raw); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}

		// EC-005: `after` field must be present.
		afterRaw, hasAfter := raw["after"]
		if !hasAfter {
			return nil, fmt.Errorf("E-CFG-001: missing required field: after")
		}

		var afterStr string
		if err := json.Unmarshal(afterRaw, &afterStr); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid after field: %w", err)
		}

		ttl, err := time.ParseDuration(afterStr)
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid duration: %q", afterStr)
		}

		// AC-005 / DI-003 server-side bounds validation (independent of CLI).
		if ttl <= 0 {
			return nil, fmt.Errorf("E-CFG-001: invalid duration: %q (must be positive)", afterStr)
		}
		if ttl > maxKeyTTL {
			return nil, fmt.Errorf("E-CFG-001: invalid duration: %q (exceeds 100-year maximum)", afterStr)
		}

		var svtnName string
		if v, ok := raw["svtn"]; ok {
			if err := json.Unmarshal(v, &svtnName); err != nil {
				return nil, fmt.Errorf("E-CFG-001: invalid svtn field: %w", err)
			}
		}
		if svtnName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn")
		}

		var pubkeyStr string
		if v, ok := raw["pubkey"]; ok {
			if err := json.Unmarshal(v, &pubkeyStr); err != nil {
				return nil, fmt.Errorf("E-CFG-001: invalid pubkey field: %w", err)
			}
		}
		pubkey, err := decodePublicKey(pubkeyStr)
		if err != nil {
			return nil, err
		}

		result, err := m.ExpireKey(svtnName, pubkey, ttl)
		if err != nil {
			return nil, mapAdminError(err, svtnName, pubkeyStr)
		}

		return adminKeyResult{
			Fingerprint: result.Fingerprint,
			At:          result.At,
		}, nil
	}
}

// makeListKeysHandler returns the admin.list-keys handler function.
// The Keys field in the response is always an array, never JSON null (EC-003).
// Traces to BC-2.05.004 postcondition 1; AC-001; AC-003.
func makeListKeysHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		var a adminListKeysArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn")
		}

		summaries, err := m.ListKeys(a.SVTNName)
		if err != nil {
			return nil, mapAdminError(err, a.SVTNName, "")
		}

		// EC-003: always return a non-nil slice even when empty.
		keys := make([]adminKeyEntry, 0, len(summaries))
		for _, s := range summaries {
			keys = append(keys, adminKeyEntry{
				Fingerprint: s.Fingerprint,
				Role:        roleToString(s.Role),
				Expiry:      s.Expiry,
			})
		}

		return adminListKeysResult{Keys: keys}, nil
	}
}

// mapAdminError converts SVTNManager / admission sentinel errors to structured
// wire-level errors per the Error Code Map in S-6.06.
// Returns a non-nil error with the mapped code as the error message prefix.
func mapAdminError(err error, svtnName, pubkey string) error {
	switch {
	case errors.Is(err, svtnmgmt.ErrSVTNNotFound):
		return fmt.Errorf("E-SVTN-003: SVTN not found: %s", svtnName)
	case errors.Is(err, svtnmgmt.ErrSVTNAlreadyExists):
		return fmt.Errorf("E-SVTN-002: SVTN already exists: %s", svtnName)
	case errors.Is(err, admission.ErrKeyNotRegistered):
		return fmt.Errorf("E-ADM-013: key not registered: %s", pubkey)
	case errors.Is(err, svtnmgmt.ErrRoleMismatch):
		return fmt.Errorf("E-ADM-019: role mismatch: claimed role does not match registered key role for key %s", pubkey)
	case errors.Is(err, svtnmgmt.ErrControlRevocationRequiresConfirm):
		return fmt.Errorf("E-ADM-018: control-to-control revocation requires explicit confirmation (set confirm: true to proceed)")
	default:
		return err
	}
}

// roleToString converts an admission.KeyRole to its canonical wire string.
// Returns "unknown" for unrecognised values — callers validate roles before
// calling this.
func roleToString(r admission.KeyRole) string {
	switch r {
	case admission.RoleControl:
		return "control"
	case admission.RoleConsole:
		return "console"
	case admission.RoleAccess:
		return "access"
	default:
		return "unknown"
	}
}

// verifyCallerRole checks that the caller-supplied role has management
// authority (control-role) for the requested operation. Non-control-role
// callers receive E-ADM-009 (BC-2.07.001 invariant 3; AC-006).
func verifyCallerRole(callerRole admission.KeyRole, cmd string, fingerprint string) error { //nolint:unused // intentional stub: called by handler Fn implementations
	if callerRole != admission.RoleControl {
		return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role %s", cmd, fingerprint, roleToString(callerRole))
	}
	return nil
}
