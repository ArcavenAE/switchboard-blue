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

// makeRegisterHandler returns the admin.key.register handler function.
// Traces to BC-2.05.004 postcondition 1; AC-001; AC-003.
func makeRegisterHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		panic("todo: AC-001 admin.key.register handler — implement in S-6.06")
	}
}

// makeRevokeHandler returns the admin.key.revoke handler function.
// Parses `role` (canonical wire field per F-002); passes as currentRole to
// SVTNManager.RevokeKey (HOLD-001 hybrid; ADR-004; ARCH-04 v1.10).
// Traces to BC-2.05.004 postcondition 2; AC-002.
func makeRevokeHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		panic("todo: AC-002 admin.key.revoke handler — implement in S-6.06")
	}
}

// makeExpireHandler returns the admin.key.expire handler function.
// Re-parses and validates the `after` duration server-side (defense-in-depth;
// DI-003) independently of sbctl CLI validation (AC-005).
// Traces to BC-2.05.004 postcondition 3; AC-005.
func makeExpireHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		panic("todo: AC-005 admin.key.expire handler — implement in S-6.06")
	}
}

// makeListKeysHandler returns the admin.list-keys handler function.
// The Keys field in the response is always an array, never JSON null (EC-003).
// Traces to BC-2.05.004 postcondition 1; AC-001; AC-003.
func makeListKeysHandler(m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		panic("todo: AC-001 admin.list-keys handler — implement in S-6.06")
	}
}

// mapAdminError converts SVTNManager / admission sentinel errors to structured
// wire-level errors per the Error Code Map in S-6.06.
// Returns a non-nil error with the mapped code embedded in the message.
// Callers check the returned error; the mgmt.Server wraps it in E-RPC-011
// unless we return a pre-formatted error string the implementer exposes
// directly as the wire message.
//
// Handler implementations call this after receiving an error from SVTNManager.
func mapAdminError(err error) error {
	panic("todo: error mapping table — implement in S-6.06")
}

// roleToString converts an admission.KeyRole to its canonical wire string.
// Returns "unknown" for unrecognised values — callers validate roles before
// calling this.
func roleToString(r admission.KeyRole) string {
	panic("todo: roleToString — implement in S-6.06")
}

// verifyCallerRole checks that the caller-supplied role has management
// authority (control-role) for the requested operation. Non-control-role
// callers receive E-ADM-009 (BC-2.07.001 invariant 3; AC-006).
func verifyCallerRole(callerRole admission.KeyRole, cmd string, fingerprint string) error {
	panic("todo: AC-006 caller-role enforcement — implement in S-6.06")
}

// Ensure sentinel error values are referenced so imports are used and
// the compiler does not drop them. These are compile-time guard assertions
// only; they are never evaluated at runtime.
var (
	_ = svtnmgmt.ErrSVTNNotFound
	_ = svtnmgmt.ErrSVTNAlreadyExists
	_ = svtnmgmt.ErrControlRevocationRequiresConfirm
	_ = svtnmgmt.ErrRoleMismatch
	_ = admission.ErrKeyNotRegistered
	_ = fmt.Sprintf
	_ = errors.Is
)
