// admin_handlers.go — daemon-side admin RPC handler builder for cmd/switchboard.
//
// BuildAdminHandlers returns the []mgmt.Handler slice for all four admin RPCs:
//
//	admin.key.register   (BC-2.05.004 PC-1)
//	admin.key.revoke     (BC-2.05.004 PC-2; HOLD-001 hybrid; ADR-004)
//	admin.key.expire     (BC-2.05.004 PC-3; DI-003 defense-in-depth duration validation)
//	admin.key.list-keys  (BC-2.05.004 PC-1 confirmation surface; any admitted role; F-L2-001/F-L2-003)
//
// Only the control-mode daemon calls BuildAdminHandlers (ADR-004 role-exclusion
// (ARCH-04 disambiguation table); AC-004). Access, console, and router daemons pass nil handlers.
//
// Purity classification (ARCH-09): boundary — depends on SVTNManager (boundary)
// and mgmt.Handler (struct). No data-plane imports permitted (ADR-004 + ARCH-12 data-plane/management-plane separation).
//
// Forbidden imports: internal/frame, internal/routing, internal/multipath,
// internal/arq, internal/replay, internal/paths, internal/halfchannel,
// internal/session, internal/tmux, internal/discovery, cmd/sbctl.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
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
	SVTNName   string `json:"svtn"`
	PublicKey  string `json:"pubkey"` // base64-encoded Ed25519 public key
	Role       string `json:"role"`
	CallerRole string `json:"caller_role"` // optional; enforced by verifyCallerRole (AC-006)
}

// adminKeyRevokeArgs is the wire JSON args for admin.key.revoke.
// The `role` field is the canonical JSON key (F-002 ruling; HOLD-001 hybrid).
type adminKeyRevokeArgs struct {
	SVTNName   string `json:"svtn"`
	PublicKey  string `json:"pubkey"` // base64-encoded Ed25519 public key
	Role       string `json:"role"`   // caller-supplied current role for cross-check
	Confirm    bool   `json:"confirm"`
	CallerRole string `json:"caller_role"` // optional; enforced by verifyCallerRole (AC-006)
}

// adminKeyExpireArgs is the wire JSON args for admin.key.expire.
type adminKeyExpireArgs struct {
	SVTNName  string `json:"svtn"`
	PublicKey string `json:"pubkey"` // base64-encoded Ed25519 public key
	After     string `json:"after"`  // duration string e.g. "24h"
}

// adminListKeysArgs is the wire JSON args for admin.key.list-keys.
// admin.key.list-keys is read-only and admits any role (F-L2-003); CallerRole
// is accepted for fallback compatibility but not used for authority gating.
type adminListKeysArgs struct {
	SVTNName   string `json:"svtn"`
	CallerRole string `json:"caller_role"` // optional; NOT gated — admin.key.list-keys is any-role (F-L2-003)
}

// adminKeyResult is the success response body for key lifecycle operations.
// Satisfies BC-2.05.004 postcondition 4 (confirmation with fingerprint and timestamp).
// JSON field names match AC-001 wire contract: key_fingerprint and timestamp.
type adminKeyResult struct {
	Fingerprint string    `json:"key_fingerprint"`
	At          time.Time `json:"timestamp"`
}

// adminListKeysResult is the success response body for admin.key.list-keys.
// The Keys field is always an array (never JSON null) per EC-003.
type adminListKeysResult struct {
	Keys []adminKeyEntry `json:"keys"`
}

// adminKeyEntry is a single element in the admin.key.list-keys response.
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
// ops is the OperatorKeySet for bootstrap-grant authority (F-P4L1-001): an
// operator-set member may call admin.key.register for a SVTN with no active
// control key. Passing nil is equivalent to an empty OperatorKeySet (no
// operator keys configured; bootstrap mode uses the daemon's own key via
// SVTNManager.IsBootstrapKey).
//
// Only the control-mode daemon should call BuildAdminHandlers. All other
// daemon modes pass nil (or an empty slice) for admin commands so that they
// correctly return E-RPC-010 "unknown command" (ADR-004; AC-004).
func BuildAdminHandlers(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) []mgmt.Handler {
	if m == nil {
		panic("BuildAdminHandlers: SVTNManager must not be nil (EC-004)")
	}
	if ops == nil {
		ops = mgmt.NewOperatorKeySet(nil)
	}
	return []mgmt.Handler{
		{Command: "admin.key.register", Fn: makeRegisterHandler(m, ops)},
		{Command: "admin.key.revoke", Fn: makeRevokeHandler(m, ops)},
		{Command: "admin.key.expire", Fn: makeExpireHandler(m, ops)},
		{Command: "admin.key.list-keys", Fn: makeListKeysHandler(m)},
	}
}

// decodePublicKey decodes a base64-encoded (standard or raw URL) Ed25519 public key.
// Returns E-CFG-001 if the value is missing or does not decode to exactly 32 bytes
// (ed25519.PublicKeySize). The raw-string fallback was removed (F-005): all keys
// must be valid base64 and decode to exactly 32 bytes.
func decodePublicKey(encoded string) (ed25519.PublicKey, error) {
	if encoded == "" {
		return nil, fmt.Errorf("E-CFG-001: missing required field: pubkey")
	}
	// Try standard encoding first, then raw URL encoding.
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		raw, err = base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid pubkey: not valid base64: %w", err)
		}
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("E-CFG-001: invalid pubkey: must be 32-byte Ed25519 public key (got %d bytes)", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

// makeRegisterHandler returns the admin.key.register handler function.
// Traces to BC-2.05.004 postcondition 1; AC-001; AC-003; AC-006.
func makeRegisterHandler(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var a adminKeyRegisterArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn")
		}

		// AC-006: resolve caller role server-side from authenticated pubkey in ctx
		// (F-001b / BC-2.05.004 Precondition 1 / DI-001). Falls back to CallerRole arg in unit tests.
		if err := resolveAndVerifyCallerRole(ctx, m, ops, a.SVTNName, a.CallerRole, "admin.key.register"); err != nil {
			return nil, err
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
			return nil, mapAdminError(err, a.SVTNName, a.PublicKey, a.Role)
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
// Traces to BC-2.05.004 postcondition 2; AC-002; AC-006.
func makeRevokeHandler(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var a adminKeyRevokeArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn")
		}

		// AC-006: resolve caller role server-side from authenticated pubkey in ctx
		// (F-001b / BC-2.05.004 Precondition 1 / DI-001). Falls back to CallerRole arg in unit tests.
		if err := resolveAndVerifyCallerRole(ctx, m, ops, a.SVTNName, a.CallerRole, "admin.key.revoke"); err != nil {
			return nil, err
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
			return nil, mapAdminError(err, a.SVTNName, a.PublicKey, a.Role)
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
func makeExpireHandler(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		// Use a raw map to detect absent fields (EC-005) vs zero-value fields.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(args, &raw); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
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

		// AC-006: enforce caller authority server-side (F-006; F-001b).
		// Auth fires before input validation so BC-2.05.004 Precondition 1 "handler
		// gate fires BEFORE dispatch" is uniform across all admin handlers.
		// No CallerRole field in expire args — purely server-resolved.
		if err := resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.expire"); err != nil {
			return nil, err
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
			// expire has no caller-supplied role; empty claimedRoleStr falls back
			// gracefully in mapAdminError (E-ADM-019 path uses *RoleMismatchError detail).
			return nil, mapAdminError(err, svtnName, pubkeyStr, "")
		}

		return adminKeyResult{
			Fingerprint: result.Fingerprint,
			At:          result.At,
		}, nil
	}
}

// makeListKeysHandler returns the admin.key.list-keys handler function.
// admin.key.list-keys is a read-only operation accessible to any admitted role
// (F-L2-003 / interface-definitions.md); no E-ADM-009 gate is applied.
// The Keys field in the response is always an array, never JSON null (EC-003).
// Traces to BC-2.05.004 postcondition 1; AC-001; AC-003.
func makeListKeysHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var a adminListKeysArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn")
		}

		// F-L2-003: admin.key.list-keys is read-only; any admitted role may call it.
		// No resolveAndVerifyCallerRole call here — the authority gate does NOT apply.

		summaries, err := m.ListKeys(a.SVTNName)
		if err != nil {
			return nil, mapAdminError(err, a.SVTNName, "", "")
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
// All arms use %w to preserve the error chain (go.md rule 4; F-009).
//
// Parameters:
//   - svtnName: the SVTN name in scope at the call site (for E-SVTN-*, E-ADM-018,
//     and E-ADM-020).
//   - targetPubEncoded: base64-encoded target public key (for E-ADM-013 / E-ADM-019
//     fingerprint computation). May be empty for list-keys where no target key exists.
//   - claimedRoleStr: canonical role string the caller supplied (for E-ADM-019
//     fallback when *RoleMismatchError is not available). May be empty.
//
// Note: ErrInvalidDuration is NOT handled here. The handler-side guards (ttl <= 0 or
// ttl > maxKeyTTL) already produce E-CFG-001 with proper detail before calling
// SVTNManager.ExpireKey, so ErrInvalidDuration from SVTNManager is unreachable in
// practice. Removing the arm avoids dead code and keeps the switch exhaustive for
// the errors that production callers can actually surface.
func mapAdminError(err error, svtnName, targetPubEncoded, claimedRoleStr string) error {
	switch {
	case errors.Is(err, svtnmgmt.ErrSVTNNotFound):
		return fmt.Errorf("E-SVTN-003: SVTN not found: %s: %w", svtnName, err)
	case errors.Is(err, admission.ErrKeyNotRegistered):
		// Compute fingerprint from the target public key bytes for the canonical message.
		// Falls back to the encoded string if decode fails (key was already validated
		// before the call, so decode failure here is a programmer error).
		fp := targetPubEncoded
		if pub, decErr := decodePublicKey(targetPubEncoded); decErr == nil {
			fp = keyFingerprintAdmin(pub)
		}
		return fmt.Errorf("E-ADM-013: key not found: no key with fingerprint %s registered in SVTN %s: %w", fp, svtnName, err)
	case errors.Is(err, svtnmgmt.ErrRoleMismatch):
		// Extract per-call role detail from *admission.RoleMismatchError when available
		// (returned by RevokeKeyIfRoleMatches / SetKeyExpiryIfRoleMatches). Fall back
		// to the caller-supplied claimedRoleStr when the typed error is absent.
		fp := targetPubEncoded
		if pub, decErr := decodePublicKey(targetPubEncoded); decErr == nil {
			fp = keyFingerprintAdmin(pub)
		}
		var rmErr *admission.RoleMismatchError
		if errors.As(err, &rmErr) {
			return fmt.Errorf(
				"E-ADM-019: role mismatch: claimed role %s does not match registered key role %s for key %s: %w",
				rmErr.ClaimedRole, rmErr.RegisteredRole, fp, err,
			)
		}
		return fmt.Errorf(
			"E-ADM-019: role mismatch: claimed role %s does not match registered key role for key %s: %w",
			claimedRoleStr, fp, err,
		)
	case errors.Is(err, svtnmgmt.ErrControlRevocationRequiresConfirm):
		return fmt.Errorf("E-ADM-018: control-to-control revocation requires explicit confirmation: use --confirm=%s to proceed: %w", svtnName, err)
	case errors.Is(err, svtnmgmt.ErrBootstrapKeyRevokeForbidden):
		return fmt.Errorf("E-ADM-020: bootstrap-key-revoke-forbidden: cannot revoke the bootstrap key in SVTN %s (permanent trust anchor): %w", svtnName, err)
	case errors.Is(err, svtnmgmt.ErrBootstrapKeyExpireForbidden):
		return fmt.Errorf("E-ADM-021: bootstrap-key-expire-forbidden: cannot expire the bootstrap key in SVTN %s (permanent trust anchor): %w", svtnName, err)
	default:
		// Default arm is defense-in-depth: every sentinel SVTNManager can return
		// should have an explicit case above. If this arm fires it is a programmer
		// error. Do NOT stamp E-RPC-011 here — mgmt.go is the sole authority for
		// stamping that code on the wire envelope. Co-stamping produces a malformed
		// response: {code:"E-RPC-011", message:"E-RPC-011: unmapped admin error: ..."}.
		// This arm's only role is to surface inner detail for the operator.
		return fmt.Errorf("unmapped admin error: %w", err)
	}
}

// roleToString converts an admission.KeyRole to its canonical wire string.
// Panics on unrecognised values — callers validate roles before calling this,
// and an unknown role in a switch indicates a programmer error (F-006a).
func roleToString(r admission.KeyRole) string {
	switch r {
	case admission.RoleControl:
		return "control"
	case admission.RoleConsole:
		return "console"
	case admission.RoleAccess:
		return "access"
	default:
		panic(fmt.Sprintf("unhandled KeyRole: %d", r))
	}
}

// verifyCallerRole checks that the caller-supplied role has management
// authority (control-role) for the requested operation. Non-control-role
// callers receive E-ADM-009 (BC-2.05.004 Precondition 1 / DI-001; AC-006).
func verifyCallerRole(callerRole admission.KeyRole, cmd string, fingerprint string) error {
	if callerRole != admission.RoleControl {
		return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role %s", cmd, fingerprint, roleToString(callerRole))
	}
	return nil
}

// keyFingerprintAdmin computes "SHA256:<base64>" for a pubkey.
// Mirrors svtnmgmt.keyFingerprint (unexported there; duplicated here to avoid
// package coupling — F-001b).
func keyFingerprintAdmin(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return "SHA256:" + base64.StdEncoding.EncodeToString(h[:])
}

// resolveAndVerifyCallerRole resolves the authenticated caller's key role via
// the server-side context (preferred) and enforces that it is control-role.
//
// Server-resolved path (F-001b / BC-2.05.004 Precondition 1 / DI-001 / AC-006):
//  1. Look up pubkey via CallerKeyRoleActive (F-P4L1-003): only active (not
//     revoked, not expired) entries yield a role. Revoked or expired keys are
//     denied immediately — they do NOT fall through to the bootstrap-grant path
//     (F-P5L1-001 fail-open regression fix).
//  2. If active and found with control role → allow.
//  3. If active and found with non-control role → deny with E-ADM-009.
//  4. If CallerKeyRoleActive returns (0, false):
//     a. If the key IS registered in any state (revoked/expired) → deny with
//     E-ADM-009 immediately (fail-closed, F-P5L1-001).
//     b. If the key is genuinely not registered AND cmd=="admin.key.register"
//     AND caller is in ops OperatorKeySet AND SVTN has no active non-bootstrap
//     control key → allow (operator-key bootstrap grant, F-P4L1-001 /
//     BC-2.05.004 EC-005).
//     c. If the daemon's own bootstrap key → allow (trust anchor).
//     d. Otherwise → deny with E-ADM-009 (fail-closed, F-P2L1-001).
//
// Fallback path (no handshake context):
//   - If ctx has no caller pubkey and callerRoleStr is non-empty, parse and
//     check it. This path is only reachable when handlers are called outside
//     the mgmt handshake (e.g., unit tests that inject a known role string).
//   - If ctx has no caller pubkey and callerRoleStr is empty, the caller's
//     role cannot be confirmed → reject with E-ADM-009 (fail-closed,
//     BC-2.05.004 Precondition 1 / DI-001).
func resolveAndVerifyCallerRole(ctx context.Context, m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet, svtnName, callerRoleStr, cmd string) error {
	if ops == nil {
		ops = mgmt.NewOperatorKeySet(nil)
	}
	callerPub, hasPubkey := mgmt.CallerPubkey(ctx)
	if hasPubkey {
		// F-P4L1-003: use CallerKeyRoleActive — only active keys return a role.
		role, found := m.CallerKeyRoleActive(svtnName, callerPub)
		if found {
			fp := keyFingerprintAdmin(callerPub)
			return verifyCallerRole(role, cmd, fp)
		}

		// CallerKeyRoleActive returned (0, false). The key is either:
		//   (a) registered but revoked/expired — must deny immediately (F-P5L1-001),
		//   (b) genuinely not registered — may proceed to bootstrap/operator check.
		fp := keyFingerprintAdmin(callerPub)
		if m.IsRegisteredAnyState(svtnName, callerPub) {
			// Registered but inactive (revoked or expired) — fail-closed, no bypass.
			return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role unregistered", cmd, fp)
		}

		// Key is genuinely not registered. Check bootstrap and operator-key paths.
		// Bootstrap key (daemon's own trust anchor) is always allowed.
		if m.IsBootstrapKey(callerPub) {
			return nil
		}
		// F-P4L1-001: operator-key bootstrap grant for admin.key.register only.
		// The condition is: SVTN has no active non-bootstrap control key (no other
		// human-registered control key exists).
		if cmd == "admin.key.register" && ops.IsAuthorized(callerPub) && !m.HasNonBootstrapControlKey(svtnName) {
			return nil
		}
		return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role unregistered", cmd, fp)
	}

	// No pubkey in ctx — fallback when no handshake context is present.
	if callerRoleStr == "" {
		// Cannot confirm caller role: fail closed (BC-2.05.004 Precondition 1 / DI-001).
		// DEFER(S-HRD.02): structured-log admin auth rejection — see S-HRD.02 (daemon logging infrastructure).
		return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key (unknown) has role unregistered", cmd)
	}
	cr, err := admission.KeyRoleFromString(callerRoleStr)
	if err != nil {
		return fmt.Errorf("E-CFG-001: invalid caller_role: %q", callerRoleStr)
	}
	return verifyCallerRole(cr, cmd, "(unknown)")
}
