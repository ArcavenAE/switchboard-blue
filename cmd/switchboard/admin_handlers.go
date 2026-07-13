// admin_handlers.go — daemon-side admin RPC handler builder for cmd/switchboard.
//
// BuildAdminHandlers returns the []mgmt.Handler slice for all admin RPCs:
//
//	admin.key.register   (BC-2.05.004 PC-1)
//	admin.key.revoke     (BC-2.05.004 PC-2; HOLD-001 hybrid; ADR-004)
//	admin.key.expire     (BC-2.05.004 PC-3; DI-003 defense-in-depth duration validation)
//	admin.key.list-keys  (BC-2.05.004 PC-1 confirmation surface; any admitted role; F-L2-001/F-L2-003)
//	admin.svtn.create    (BC-2.07.001 PC-1 + PC-2 + Inv-3; S-6.07)
//	admin.svtn.destroy   (BC-2.07.001 PC-3; RULING-W6TB-A; S-6.05)
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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
	"golang.org/x/crypto/ssh"
)

// maxKeyTTL is the server-side upper bound for admin.key.expire TTL values.
// Rejects any TTL greater than 100 years (AC-005; DI-003 defense-in-depth).
const maxKeyTTL = 100 * 365 * 24 * time.Hour

// adminKeyRegisterArgs is the wire JSON args for admin.key.register.
// The `role` field uses the canonical JSON key per interface-definitions.md v1.1.
type adminKeyRegisterArgs struct {
	SVTNName   string `json:"svtn_id"`
	PublicKey  string `json:"pubkey_openssh"` // OpenSSH-format Ed25519 public key (e.g. ssh-ed25519 AAAA... comment)
	Role       string `json:"role"`
	CallerRole string `json:"caller_role"` // optional; enforced by verifyCallerRole (AC-006)
}

// adminKeyRevokeArgs is the wire JSON args for admin.key.revoke.
// The `role` field is the canonical JSON key (F-002 ruling; HOLD-001 hybrid).
type adminKeyRevokeArgs struct {
	SVTNName   string `json:"svtn_id"`
	PublicKey  string `json:"pubkey_openssh"` // OpenSSH-format Ed25519 public key (e.g. ssh-ed25519 AAAA... comment)
	Role       string `json:"role"`           // caller-supplied current role for cross-check
	Confirm    bool   `json:"confirm"`
	CallerRole string `json:"caller_role"` // optional; enforced by verifyCallerRole (AC-006)
}

// adminKeyExpireArgs is the wire JSON args for admin.key.expire.
type adminKeyExpireArgs struct {
	SVTNName  string `json:"svtn_id"`
	PublicKey string `json:"pubkey_openssh"` // OpenSSH-format Ed25519 public key (e.g. ssh-ed25519 AAAA... comment)
	After     string `json:"after"`          // duration string e.g. "24h"
}

// adminListKeysArgs is the wire JSON args for admin.key.list-keys.
// admin.key.list-keys is read-only and admits any role (F-L2-003); CallerRole
// is accepted for fallback compatibility but not used for authority gating.
type adminListKeysArgs struct {
	SVTNName   string `json:"svtn_id"`
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
// Expiry is a pointer so that keys with no expiry omit the field entirely in
// JSON output — encoding/json does not treat a zero time.Time as empty for
// omitempty, so a value field would serialize as "0001-01-01T00:00:00Z" for
// non-expiring keys (F-P18L1-002).
type adminKeyEntry struct {
	Fingerprint string     `json:"fingerprint"`
	Role        string     `json:"role"`
	Expiry      *time.Time `json:"expiry,omitempty"`
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
		{Command: "admin.key.list-keys", Fn: makeListKeysHandler(m, ops)},
		{Command: "admin.svtn.create", Fn: makeAdminSVTNCreateHandler(m, ops)},
		{Command: "admin.svtn.destroy", Fn: makeAdminSVTNDestroyHandler(m, ops)},
		{Command: "admin.svtn.status", Fn: makeAdminSVTNStatusHandler(m, ops)},
	}
}

// decodePublicKey decodes an Ed25519 public key from either OpenSSH authorized_keys
// format ("ssh-ed25519 <base64> [comment]") or raw base64-encoded bytes.
// Returns E-CFG-001 if the value is missing, has the wrong key type, or does not
// decode to exactly 32 bytes (ed25519.PublicKeySize).
func decodePublicKey(encoded string) (ed25519.PublicKey, error) {
	if encoded == "" {
		return nil, fmt.Errorf("E-CFG-001: missing required field: pubkey_openssh")
	}

	// Detect OpenSSH authorized_keys format by the presence of a key-type prefix.
	// ssh.ParseAuthorizedKey handles "ssh-ed25519 <base64> [comment]" and similar.
	if strings.HasPrefix(encoded, "ssh-") || strings.HasPrefix(encoded, "ecdsa-") || strings.HasPrefix(encoded, "sk-") {
		sshPubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(encoded))
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid pubkey: cannot parse OpenSSH format: %w", err)
		}
		// Only ed25519 keys are accepted.
		if sshPubKey.Type() != "ssh-ed25519" {
			return nil, fmt.Errorf("E-CFG-001: invalid pubkey: key type %q is not supported; must be ed25519", sshPubKey.Type())
		}
		cryptoPub, ok := sshPubKey.(ssh.CryptoPublicKey)
		if !ok {
			return nil, fmt.Errorf("E-CFG-001: invalid pubkey: cannot extract crypto public key from ssh.PublicKey")
		}
		ed25519Pub, ok := cryptoPub.CryptoPublicKey().(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("E-CFG-001: invalid pubkey: ssh-ed25519 key did not yield ed25519.PublicKey")
		}
		return ed25519Pub, nil
	}

	// Fall back to raw base64 (standard or raw URL encoding) for backward compatibility.
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
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}

		// AC-006: resolve caller role server-side from authenticated pubkey in ctx
		// (F-001b / BC-2.05.004 Precondition 1 / DI-001). Falls back to CallerRole arg in unit tests.
		if _, err := resolveAndVerifyCallerRole(ctx, m, ops, a.SVTNName, a.CallerRole, "admin.key.register"); err != nil {
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
			return nil, mapAdminError(err, a.SVTNName, pubkey, a.Role)
		}

		return adminKeyResult{
			Fingerprint: result.Fingerprint,
			At:          result.At,
		}, nil
	}
}

// makeRevokeHandler returns the admin.key.revoke handler function.
// Parses `role` (canonical wire field per F-002); passes as currentRole to
// SVTNManager.RevokeKey (HOLD-001 hybrid; ADR-004; ARCH-04 v1.13).
// Traces to BC-2.05.004 postcondition 2; AC-002; AC-006.
func makeRevokeHandler(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var a adminKeyRevokeArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}

		// AC-006: resolve caller role server-side from authenticated pubkey in ctx
		// (F-001b / BC-2.05.004 Precondition 1 / DI-001). Falls back to CallerRole arg in unit tests.
		if _, err := resolveAndVerifyCallerRole(ctx, m, ops, a.SVTNName, a.CallerRole, "admin.key.revoke"); err != nil {
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
			return nil, mapAdminError(err, a.SVTNName, pubkey, a.Role)
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
		if v, ok := raw["svtn_id"]; ok {
			if err := json.Unmarshal(v, &svtnName); err != nil {
				return nil, fmt.Errorf("E-CFG-001: invalid svtn_id field: %w", err)
			}
		}
		if svtnName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}

		// AC-006: enforce caller authority server-side (F-006; F-001b).
		// Auth fires before input validation so BC-2.05.004 Precondition 1 "handler
		// gate fires BEFORE dispatch" is uniform across all admin handlers.
		// No CallerRole field in expire args — purely server-resolved.
		if _, err := resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.expire"); err != nil {
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
		if v, ok := raw["pubkey_openssh"]; ok {
			if err := json.Unmarshal(v, &pubkeyStr); err != nil {
				return nil, fmt.Errorf("E-CFG-001: invalid pubkey_openssh field: %w", err)
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
			return nil, mapAdminError(err, svtnName, pubkey, "")
		}

		return adminKeyResult{
			Fingerprint: result.Fingerprint,
			At:          result.At,
		}, nil
	}
}

// makeListKeysHandler returns the admin.key.list-keys handler function.
// admin.key.list-keys is a read-only operation accessible to any admitted role
// (F-L2-003 / interface-definitions.md); the control-only authority gate does NOT
// apply, but admission is still required (CWE-862 defense; BC-2.05.004).
// The Keys field in the response is always an array, never JSON null (EC-003).
// Traces to BC-2.05.004 postcondition 1; AC-001; AC-003.
func makeListKeysHandler(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var a adminListKeysArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNName == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}

		// F-L2-003: admin.key.list-keys is read-only; any admitted role may call it.
		// Admission (SVTN membership OR operator-set OR bootstrap) is REQUIRED —
		// the control-only authority gate is what F-L2-003 removes, NOT the
		// admission gate. See BC-2.05.004:155 and sec-triage report P5-pass-13.
		if err := resolveCallerAdmissionAnyRole(ctx, m, ops, a.SVTNName, "admin.key.list-keys"); err != nil {
			return nil, err
		}

		summaries, err := m.ListKeys(a.SVTNName)
		if err != nil {
			return nil, mapAdminError(err, a.SVTNName, nil, "")
		}

		// EC-003: always return a non-nil slice even when empty.
		keys := make([]adminKeyEntry, 0, len(summaries))
		for _, s := range summaries {
			entry := adminKeyEntry{
				Fingerprint: s.Fingerprint,
				Role:        roleToString(s.Role),
			}
			// Only set Expiry when a TTL has been configured. A nil pointer
			// causes encoding/json to omit the field entirely, so consumers
			// see no "expiry" key for permanent keys (F-P18L1-002).
			if !s.Expiry.IsZero() {
				t := s.Expiry.UTC()
				entry.Expiry = &t
			}
			keys = append(keys, entry)
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
//   - targetPub: already-decoded target public key (for E-ADM-013 / E-ADM-019
//     fingerprint computation). Nil/zero-length for operations where no target key
//     exists (e.g. list-keys); fingerprint is computed once at entry and reused.
//   - claimedRoleStr: canonical role string the caller supplied (for E-ADM-019
//     fallback when *RoleMismatchError is not available). May be empty.
//
// ErrInvalidDuration is handled here as a defense-in-depth arm (F-L1-B). The
// handler-side guards (ttl <= 0 or ttl > maxKeyTTL) already produce E-CFG-001
// before calling SVTNManager.ExpireKey, so this arm is unreachable in production.
// An explicit case prevents the default arm from swallowing it silently if the
// guard is ever bypassed.
func mapAdminError(err error, svtnName string, targetPub ed25519.PublicKey, claimedRoleStr string) error {
	// Compute fingerprint once; reused by E-ADM-013 and E-ADM-019 arms.
	// For operations without a target key (nil/zero pub), fp is the hash of an
	// empty byte slice — callers that need it always pass a real key.
	fp := keyFingerprintAdmin(targetPub)

	switch {
	case errors.Is(err, svtnmgmt.ErrSVTNNotFound):
		return &svtnNotFoundErr{name: svtnName, cause: err}
	case errors.Is(err, admission.ErrKeyNotRegistered):
		return &adminKeyNotFoundErr{fingerprint: fp, svtnName: svtnName, cause: err}
	case errors.Is(err, svtnmgmt.ErrRoleMismatch):
		// Extract per-call role detail from *admission.RoleMismatchError when available
		// (returned by RevokeKeyIfRoleMatches / SetKeyExpiryIfRoleMatches). Fall back
		// to the caller-supplied claimedRoleStr when the typed error is absent.
		var rmErr *admission.RoleMismatchError
		if errors.As(err, &rmErr) {
			return fmt.Errorf(
				"E-ADM-019: role mismatch: claimed role %s does not match registered key role %s for key %s: %w",
				rmErr.ClaimedRole, rmErr.RegisteredRole, fp, err,
			)
		}
		return fmt.Errorf(
			"E-ADM-019: role mismatch: claimed role %s does not match registered key role %s for key %s: %w",
			claimedRoleStr, "unknown", fp, err,
		)
	case errors.Is(err, svtnmgmt.ErrInvalidDuration):
		// Defense-in-depth: handler-side duration guards already fire before
		// ExpireKey is called, so this arm is unreachable in production (F-L1-B).
		return fmt.Errorf("E-CFG-001: invalid duration: %w", err)
	case errors.Is(err, svtnmgmt.ErrControlRevocationRequiresConfirm):
		return fmt.Errorf("E-ADM-018: control-to-control revocation requires explicit confirmation: use --confirm to proceed: %w", err)
	case errors.Is(err, svtnmgmt.ErrBootstrapKeyRevokeForbidden):
		return fmt.Errorf("E-ADM-020: bootstrap-key-revoke-forbidden: cannot revoke the bootstrap key in SVTN %s (permanent trust anchor): %w", svtnName, err)
	case errors.Is(err, svtnmgmt.ErrBootstrapKeyExpireForbidden):
		return fmt.Errorf("E-ADM-021: bootstrap-key-expire-forbidden: cannot expire the bootstrap key in SVTN %s (permanent trust anchor): %w", svtnName, err)
	case errors.Is(err, svtnmgmt.ErrDestroyUnauthorized):
		return fmt.Errorf("E-ADM-011: permission denied: %s key cannot destroy SVTN %s: %w", claimedRoleStr, svtnName, err)
	default:
		// Default arm is defense-in-depth: every sentinel SVTNManager can return
		// should have an explicit case above. If this arm fires it is a programmer
		// error. E-INT-999 is the catch-all programmer-error code (Ruling-12 §1
		// universality: every error returned from a handler must carry an E-* prefix
		// so the wire envelope always has a machine-readable code). Do NOT use
		// E-RPC-011 here — mgmt.go stamps that on the wire envelope; co-stamping
		// produces a malformed response.
		return fmt.Errorf("E-INT-999: unmapped internal condition, programmer error, please report: %w", err)
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
// On success it returns the caller's real AdmittedKey (Role + PublicKey populated
// from server-side resolution). Callers that need the resolved identity (e.g.
// makeAdminSVTNDestroyHandler) pass it directly to SVTNManager methods, preserving
// the defense-in-depth invariant even if the handler gate is later removed (F-P3L1-001).
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
func resolveAndVerifyCallerRole(ctx context.Context, m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet, svtnName, callerRoleStr, cmd string) (admission.AdmittedKey, error) {
	if ops == nil {
		ops = mgmt.NewOperatorKeySet(nil)
	}
	callerPub, hasPubkey := mgmt.CallerPubkey(ctx)
	if hasPubkey {
		// F-P4L1-003: use CallerKeyRoleActive — only active keys return a role.
		role, found := m.CallerKeyRoleActive(svtnName, callerPub)
		if found {
			fp := keyFingerprintAdmin(callerPub)
			if err := verifyCallerRole(role, cmd, fp); err != nil {
				return admission.AdmittedKey{}, err
			}
			return admission.AdmittedKey{Role: role, PublicKey: callerPub}, nil
		}

		// CallerKeyRoleActive returned (0, false). The key is either:
		//   (a) registered but revoked/expired — must deny immediately (F-P5L1-001),
		//   (b) genuinely not registered — may proceed to bootstrap/operator check.
		fp := keyFingerprintAdmin(callerPub)
		if m.IsRegisteredAnyState(svtnName, callerPub) {
			// Registered but inactive (revoked or expired) — fail-closed, no bypass.
			return admission.AdmittedKey{}, fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role unregistered", cmd, fp)
		}

		// Key is genuinely not registered. Check bootstrap and operator-key paths.
		// Bootstrap key (daemon's own trust anchor) is always allowed; carries RoleControl.
		if m.IsBootstrapKey(callerPub) {
			return admission.AdmittedKey{Role: admission.RoleControl, PublicKey: callerPub}, nil
		}
		// F-P4L1-001: operator-key bootstrap grant for admin.key.register only.
		// The condition is: SVTN has no active non-bootstrap control key (no other
		// human-registered control key exists). Operator keys are implicitly control-role.
		if cmd == "admin.key.register" && ops.IsAuthorized(callerPub) && !m.HasNonBootstrapControlKey(svtnName) {
			return admission.AdmittedKey{Role: admission.RoleControl, PublicKey: callerPub}, nil
		}
		return admission.AdmittedKey{}, fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role unregistered", cmd, fp)
	}

	// No pubkey in ctx — fallback when no handshake context is present.
	if callerRoleStr == "" {
		// Cannot confirm caller role: fail closed (BC-2.05.004 Precondition 1 / DI-001).
		// DEFER(S-HRD.02): structured-log admin auth rejection — see S-HRD.02 (daemon logging infrastructure).
		return admission.AdmittedKey{}, fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key (unknown) has role unregistered", cmd)
	}
	cr, err := admission.KeyRoleFromString(callerRoleStr)
	if err != nil {
		return admission.AdmittedKey{}, fmt.Errorf("E-CFG-001: invalid caller_role: %q", callerRoleStr)
	}
	if err := verifyCallerRole(cr, cmd, "(unknown)"); err != nil {
		return admission.AdmittedKey{}, err
	}
	// Fallback path: no pubkey available; Role is the resolved role from callerRoleStr.
	return admission.AdmittedKey{Role: cr}, nil
}

// resolveCallerAdmissionAnyRole verifies the caller is admitted to `svtnName` in
// ANY role (control, console, or access), or is an OperatorKeySet member, or is
// the bootstrap key. Used for read-only ops (e.g. admin.key.list-keys) where the
// control-only authority gate does NOT apply per BC-2.05.004 F-L2-003, but the
// admission requirement still holds (CWE-862 defense: prevent cross-SVTN
// enumeration by any caller holding a valid operator handshake).
//
// Returns E-ADM-009 when the caller cannot be resolved or is not admitted.
func resolveCallerAdmissionAnyRole(ctx context.Context, m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet, svtnName, cmd string) error {
	if ops == nil {
		ops = mgmt.NewOperatorKeySet(nil)
	}
	callerPub, hasPubkey := mgmt.CallerPubkey(ctx)
	if !hasPubkey {
		// No pubkey in ctx — fail closed (BC-2.05.004 Precondition 1 / DI-001).
		return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key (unknown) has role unregistered", cmd)
	}

	// Bootstrap key is the unconditional trust anchor.
	if m.IsBootstrapKey(callerPub) {
		return nil
	}

	// F-L2-003: operator-set member may call any read-only admin op unconditionally.
	if ops.IsAuthorized(callerPub) {
		return nil
	}

	// F-P4L1-003: use CallerKeyRoleActive — only active keys return a role.
	_, found := m.CallerKeyRoleActive(svtnName, callerPub)
	if found {
		return nil
	}

	fp := keyFingerprintAdmin(callerPub)
	// Registered but inactive (revoked or expired) — fail-closed, no bypass (F-P5L1-001).
	if m.IsRegisteredAnyState(svtnName, callerPub) {
		return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role unregistered", cmd, fp)
	}

	// Key is not admitted to this SVTN and is not bootstrap/operator.
	return fmt.Errorf("E-ADM-009: insufficient authority for operation %s: key %s has role unregistered", cmd, fp)
}

// svtnAlreadyExistsErr is returned by makeAdminSVTNCreateHandler when
// SVTNManager.Create returns ErrSVTNAlreadyExists. It implements Unwrap so
// that errors.Is(err, svtnmgmt.ErrSVTNAlreadyExists) returns true while the
// Error() message is derived from the SVTN name only — no stutter from
// concatenating the sentinel's own text (F-P1L1-004; F-P1L2-001).
//
// Message format: "E-SVTN-001: SVTN already exists: <name>"
// per BC-2.07.001 EC-001 canonical message and error-taxonomy.md row E-SVTN-001.
type svtnAlreadyExistsErr struct {
	name  string
	cause error
}

func (e *svtnAlreadyExistsErr) Error() string {
	return fmt.Sprintf("E-SVTN-001: SVTN already exists: %s", e.name)
}

func (e *svtnAlreadyExistsErr) Unwrap() error { return e.cause }

// svtnNotFoundErr is returned by mapAdminError when ErrSVTNNotFound is encountered.
// It implements Unwrap so that errors.Is(err, svtnmgmt.ErrSVTNNotFound) returns true
// while the Error() message is deduplicated — no stutter from concatenating the
// sentinel's "SVTN not found" text with the SVTN name again (F-P9L1-04).
//
// Message format: "E-SVTN-003: SVTN not found: <name>"
type svtnNotFoundErr struct {
	name  string
	cause error
}

func (e *svtnNotFoundErr) Error() string {
	return fmt.Sprintf("E-SVTN-003: SVTN not found: %s", e.name)
}

func (e *svtnNotFoundErr) Unwrap() error { return e.cause }

// adminKeyNotFoundErr is returned by mapAdminError when ErrKeyNotRegistered is
// encountered. It implements Unwrap so that errors.Is(err, admission.ErrKeyNotRegistered)
// returns true while the Error() message is deduplicated (F-P9L1-04).
//
// Message format: "E-ADM-013: key not found: fingerprint <fp> not registered in SVTN <name>"
type adminKeyNotFoundErr struct {
	fingerprint string
	svtnName    string
	cause       error
}

func (e *adminKeyNotFoundErr) Error() string {
	return fmt.Sprintf("E-ADM-013: key not found: no key with fingerprint %s registered in SVTN %s", e.fingerprint, e.svtnName)
}

func (e *adminKeyNotFoundErr) Unwrap() error { return e.cause }

// adminSVTNCreateArgs is the wire-format JSON args for admin.svtn.create.
// The `name` field carries the operator-supplied SVTN label.
//
// AC-002 / BC-2.07.001 PC-1 — wire format: {"command":"admin.svtn.create","args":{"name":"<name>"}}.
type adminSVTNCreateArgs struct {
	// Name is the human-readable SVTN label provided by the operator.
	Name string `json:"name"`
}

// adminSVTNCreateResult is the success data payload for admin.svtn.create.
// JSON field names match the AC-004 wire contract.
type adminSVTNCreateResult struct {
	// SVTNID is the hex-encoded 16-byte SVTN identifier (BC-2.07.001 postcondition 1).
	SVTNID string `json:"svtn_id"`
	// BootstrapFingerprint is the "SHA256:<base64>" fingerprint of the bootstrap
	// control key (BC-2.05.004 PC-4 canonical format; BC-2.07.001 PC-2).
	// Verbatim from svtnmgmt.keyFingerprint — do NOT re-encode to hex.
	BootstrapFingerprint string `json:"bootstrap_fingerprint"`
}

// makeAdminSVTNCreateHandler returns the admin.svtn.create handler function.
//
// Authority check (BC-2.07.001 Inv-3 / AC-003 / Ruling-5 / F-P2L1-001):
// admin.svtn.create is bootstrap-only — only the daemon's own bootstrap key
// (m.IsBootstrapKey) may create SVTNs. Cross-SVTN control-role keys are NOT
// authorized. The check fires BEFORE m.Create is called; non-bootstrap callers
// receive E-ADM-009 immediately. resolveAndVerifyCallerRole is NOT called here
// because the bootstrap-only constraint is stricter than the general control-role
// check used by admin.key.* handlers.
//
// On success (AC-004): returns adminSVTNCreateResult with svtn_id (hex) and
// bootstrap_fingerprint (SHA256:<base64> verbatim from svtnmgmt.keyFingerprint).
//
// On duplicate name (AC-005): propagates ErrSVTNAlreadyExists as
// "E-SVTN-001: SVTN already exists: <name>" to the RPC response.
//
// On non-duplicate Create failure: stamped E-INT-001 (F-P2L1-004).
//
// Traces to BC-2.07.001 PC-1 + PC-2 + Inv-3; AC-001; AC-003; AC-004; AC-005;
// Ruling-5; F-P2L1-001; F-P2L1-004.
func makeAdminSVTNCreateHandler(m *svtnmgmt.SVTNManager, _ *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var a adminSVTNCreateArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}

		// Args validation: name must be non-empty, non-whitespace-only, within
		// 255 bytes, and free of ASCII control characters (F-P2L2 exhaustive validation).
		if err := validateSVTNName(a.Name); err != nil {
			return nil, err
		}

		// Ruling-5 / F-P2L1-001: bootstrap-only pre-check. Only the daemon's own
		// bootstrap key may create SVTNs. This is stricter than the general
		// control-role check — a cross-SVTN control key does not have authority here.
		callerPub, hasPubkey := mgmt.CallerPubkey(ctx)
		if !hasPubkey || !m.IsBootstrapKey(callerPub) {
			fp := "(unknown)"
			role := "unregistered"
			if hasPubkey {
				fp = keyFingerprintAdmin(callerPub)
				// Diagnostic-only: resolve caller's active role across all registered
				// SVTNs so the rejection message matches BC-2.07.001 canonical format
				// "has role <role>". Does NOT affect the authority decision — the
				// bootstrap-only check above is the authoritative gate (Ruling-5 /
				// F-Lens1-02). If the key has no active role in any SVTN, "unregistered"
				// is reported to match the canonical label used by resolveAndVerifyCallerRole
				// (Ruling-12 §2).
				if r, found := m.CallerKeyRoleInAny(callerPub); found {
					role = r.String()
				}
			}
			return nil, fmt.Errorf("E-ADM-009: insufficient authority for operation admin.svtn.create: key %s has role %s", fp, role)
		}

		// Ruling-7 / F-Impl-002: defense-in-depth RoleControl check.
		// BC-2.07.001 Inv-3 mandates the bootstrap key implies RoleControl by
		// construction, but we verify explicitly so that future key-model changes
		// (e.g., a rotation flow that retains the is_bootstrap flag on a demoted key)
		// cannot silently bypass this gate. The check is skipped (allowed) when no
		// SVTNs exist yet — the first-ever create is the authorized genesis path.
		if hasExistingSVTNs := m.HasAnySVTN(); hasExistingSVTNs && !m.BootstrapKeyHasControlRole() {
			// Resolve actual role for canonical Ruling-12 §2 message shape.
			// callerPub is the bootstrap key (IsBootstrapKey passed above).
			bsFP := m.BootstrapFingerprint()
			bsRole := "unregistered"
			if r, found := m.CallerKeyRoleInAny(callerPub); found {
				bsRole = r.String()
			}
			return nil, fmt.Errorf("E-ADM-009: insufficient authority for operation admin.svtn.create: key %s has role %s", bsFP, bsRole)
		}

		result, err := m.Create(a.Name)
		if err != nil {
			// F-P1L2-001: check via errors.Is, not string matching, so that the
			// sentinel is correctly identified without depending on the error text.
			// F-P1L1-003: stamp E-SVTN-001 (not E-ADM-004) for duplicate names.
			// F-P1L1-004: derive the message from a.Name only — do NOT wrap
			// err.Error() which already contains "SVTN already exists", causing stutter.
			if errors.Is(err, svtnmgmt.ErrSVTNAlreadyExists) {
				return nil, &svtnAlreadyExistsErr{name: a.Name, cause: err}
			}
			// F-P2L1-004: non-duplicate Create failure (e.g. internal rand.Read failure)
			// stamped with E-INT-001. Use %w to preserve the error chain for operators
			// and allow errors.Is/As inspection by callers (go.md rule 4; F-Impl-003).
			return nil, fmt.Errorf("E-INT-001: internal error: admin.svtn.create: %w", err)
		}

		// AC-004: svtn_id as hex string; bootstrap_fingerprint verbatim from
		// svtnmgmt.BootstrapFingerprint (SHA256:<base64> canonical format).
		return adminSVTNCreateResult{
			SVTNID:               hex.EncodeToString(result.SVTN.ID[:]),
			BootstrapFingerprint: m.BootstrapFingerprint(),
		}, nil
	}
}

// adminSVTNDestroyArgs is the wire-format JSON args for admin.svtn.destroy.
// The `name` field carries the operator-supplied SVTN name to destroy.
//
// AC-003 / BC-2.07.001 PC-3 — wire format: {"command":"admin.svtn.destroy","args":{"name":"<name>"}}.
type adminSVTNDestroyArgs struct {
	// Name is the human-readable SVTN label to destroy.
	Name string `json:"name"`
}

// makeAdminSVTNDestroyHandler returns the admin.svtn.destroy handler function.
//
// Authority check (BC-2.07.001 Inv-3 / RULING-W6TB-A):
// admin.svtn.destroy uses the general control-role gate (resolveAndVerifyCallerRole),
// NOT the bootstrap-only gate used by admin.svtn.create. This is explicitly
// required by RULING-W6TB-A: any control-role key may destroy a SVTN, whereas
// only the bootstrap key may create one.
//
// See makeAdminSVTNCreateHandler for the bootstrap-only create handler that
// uses a stricter gate. The comment there notes: "resolveAndVerifyCallerRole is
// NOT called here [create] because the bootstrap-only constraint is stricter."
// The inverse applies here — Destroy MUST call resolveAndVerifyCallerRole.
//
// A non-control caller receives E-RPC-011 wrapping E-ADM-009 (the error code
// lifted from the E-ADM-009 message prefix by the wire-level code extractor in
// sendAdminRPC / sendAdminRPCAsKey). The SVTN is not destroyed.
//
// Traces to BC-2.07.001 PC-3; AC-001; AC-002; AC-003; AC-004; RULING-W6TB-A;
// VP-048 properties 2+3.
func makeAdminSVTNDestroyHandler(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		// F-Impl-001 / F-P5P8-A-004: reject invalid UTF-8 in the raw JSON bytes
		// before json.Unmarshal silently replaces bad bytes with U+FFFD.
		// This mirrors the create handler's invariant: name must be valid UTF-8.
		if !utf8.Valid(args) {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: request body is not valid UTF-8")
		}

		var a adminSVTNDestroyArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}

		// F-P5P8-A-004: validate name exhaustively (same arms as create) before
		// dispatching to Destroy. E-CFG-001 fires before any E-SVTN-003 lookup.
		if err := validateSVTNName(a.Name); err != nil {
			return nil, err
		}

		// RULING-W6TB-A: admin.svtn.destroy uses the general control-role gate,
		// NOT the bootstrap-only gate used by admin.svtn.create. Any active
		// control-role key may destroy a SVTN.
		//
		// resolveAndVerifyCallerRole returns the caller's real AdmittedKey (with the
		// server-resolved Role field). Propagating it into Destroy ensures the inner
		// defense-in-depth check (BC-2.07.001 Inv-3; RULING-W6TB-A §3) reflects the
		// caller's actual role rather than a synthesized RoleControl constant —
		// preserving the guard's integrity even if the outer gate is later removed
		// (F-P3L1-001).
		callerKey, err := resolveAndVerifyCallerRole(ctx, m, ops, a.Name, "", "admin.svtn.destroy")
		if err != nil {
			return nil, err
		}

		if err := m.Destroy(callerKey, a.Name); err != nil {
			return nil, mapAdminError(err, a.Name, nil, roleToString(callerKey.Role))
		}

		return struct {
			Status string `json:"status"`
		}{Status: "destroyed"}, nil
	}
}

// validateSVTNName checks that the SVTN name satisfies the admission constraints
// for admin.svtn.create (F-P2L2 exhaustive validation; F-Impl-001):
//   - non-empty
//   - not whitespace-only (strings.TrimSpace must be non-empty)
//   - at most 255 bytes (operator-readable label budget)
//   - valid UTF-8 encoding (F-Impl-001)
//   - no ASCII control characters (U+0000–U+001F, U+007F)
//   - no C1 controls (U+0080–U+009F) or Unicode Cc category — caught by unicode.IsControl
//   - no Unicode line separator (U+2028) or paragraph separator (U+2029) — explicit checks
//     because Go's unicode.IsControl covers only the Cc category, not Zl/Zp (F-Impl-001)
//
// Returns E-CFG-001 on any violation.
func validateSVTNName(name string) error {
	if name == "" {
		return fmt.Errorf("E-CFG-001: missing required field: name")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("E-CFG-001: invalid name: name must not be whitespace-only")
	}
	if len(name) > 255 {
		return fmt.Errorf("E-CFG-001: invalid name: name exceeds 255-byte maximum (got %d bytes)", len(name))
	}
	// F-Impl-001: reject invalid UTF-8 before iterating runes — range over an
	// invalid UTF-8 string silently replaces bad bytes with U+FFFD, masking the
	// violation. utf8.ValidString must be checked first.
	if !utf8.ValidString(name) {
		return fmt.Errorf("E-CFG-001: invalid name: name is not valid UTF-8")
	}
	// F-Impl-001: reject control characters from Unicode categories Cc (control),
	// Zl (line separator: U+2028), and Zp (paragraph separator: U+2029).
	// unicode.IsControl covers Cc (ASCII controls U+0000–U+001F, U+007F, and
	// C1 controls U+0080–U+009F). U+2028 and U+2029 are NOT in Cc — they are
	// Zl/Zp — so they must be checked explicitly.
	for _, r := range name {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return fmt.Errorf("E-CFG-001: invalid name: name contains control character U+%04X", r)
		}
	}
	return nil
}

// adminSVTNStatusArgs is the wire-format JSON args for admin.svtn.status.
//
// AC-005 / BC-2.07.001 v1.14 PC-4 — wire format: {"name": "<svtn-name>"}.
type adminSVTNStatusArgs struct {
	Name string `json:"name"`
}

// adminSVTNKeyCounts is the role-grouped key count breakdown returned by
// admin.svtn.status (AC-005 postcondition 2).
type adminSVTNKeyCounts struct {
	Control int `json:"control"`
	Console int `json:"console"`
	Access  int `json:"access"`
}

// adminSVTNStatusResult is the success response body for admin.svtn.status.
// Deliberately excludes session/health-indicator fields — internal/session
// remains a forbidden import for this file (AC-007 purity boundary).
type adminSVTNStatusResult struct {
	SVTNID    string             `json:"svtn_id"`
	Name      string             `json:"name"`
	CreatedAt string             `json:"created_at"`
	KeyCounts adminSVTNKeyCounts `json:"key_counts"`
}

// makeAdminSVTNStatusHandler returns the admin.svtn.status handler function.
//
// Wire contract (Decision 2): request args {"name": "<svtn-name>"}; response
// {"svtn_id": "<hex>", "name": "<svtn-name>", "created_at": "<RFC3339>",
// "key_counts": {"control": <n>, "console": <n>, "access": <n>}}. Deliberately
// excludes session/health-indicator fields — internal/session remains a
// forbidden import for this file (AC-007 purity boundary).
//
// Authority check (Decision 2 / F-L2-003 precedent): resolveCallerAdmissionAnyRole
// — any admitted role (control, console, access) in the target SVTN, OR
// operator-set member, OR bootstrap key. The admission gate still applies
// (CWE-862 defense against cross-SVTN roster/existence enumeration, mirrors
// BC-2.05.004 EC-008); only the control-only authority gate is skipped —
// same shape as makeListKeysHandler. The gate fires BEFORE SVTNByName/ListKeys
// so a caller cannot use this op to distinguish "SVTN exists" from "SVTN
// doesn't exist" via the error text (AC-006 postcondition 2/3 no-leak proof).
//
// On success: response fields sourced from m.SVTNByName (svtn_id, name,
// created_at) and role-grouped counts derived from m.ListKeys (AC-005
// postcondition 1/2).
//
// On not-found: mapAdminError maps svtnmgmt.ErrSVTNNotFound to E-SVTN-003
// (AC-006 postcondition 1).
//
// Traces to BC-2.07.001 v1.14 PC-4; AC-005; AC-006; AC-007.
func makeAdminSVTNStatusHandler(m *svtnmgmt.SVTNManager, ops *mgmt.OperatorKeySet) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		var a adminSVTNStatusArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.Name == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: name")
		}

		if err := resolveCallerAdmissionAnyRole(ctx, m, ops, a.Name, "admin.svtn.status"); err != nil {
			return nil, err
		}

		// Args validation: name must be non-empty, non-whitespace-only, within
		// 255 bytes, and free of ASCII control characters (F-P2L2 exhaustive
		// validation), mirroring makeAdminSVTNCreateHandler/
		// makeAdminSVTNDestroyHandler. Runs after the admission gate so the
		// AC-006 byte-identical denied-path oracle is unaffected — an
		// unauthorized caller still sees E-ADM-009 regardless of name shape.
		if err := validateSVTNName(a.Name); err != nil {
			return nil, err
		}

		svtn, found := m.SVTNByName(a.Name)
		if !found {
			return nil, mapAdminError(svtnmgmt.ErrSVTNNotFound, a.Name, nil, "")
		}

		summaries, err := m.ListKeys(a.Name)
		if err != nil {
			return nil, mapAdminError(err, a.Name, nil, "")
		}

		var counts adminSVTNKeyCounts
		for _, s := range summaries {
			switch s.Role {
			case admission.RoleControl:
				counts.Control++
			case admission.RoleConsole:
				counts.Console++
			case admission.RoleAccess:
				counts.Access++
			}
		}

		return adminSVTNStatusResult{
			SVTNID:    hex.EncodeToString(svtn.ID[:]),
			Name:      svtn.Name,
			CreatedAt: svtn.CreatedAt.Format(time.RFC3339),
			KeyCounts: counts,
		}, nil
	}
}
