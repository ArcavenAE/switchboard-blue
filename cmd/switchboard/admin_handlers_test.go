package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// newTestSVTNManagerDetailed returns a SVTNManager pre-populated with the SVTNs
// that happy-path and error-mapping tests reference, along with the bootstrap
// public key. Tests that invoke handlers without a callerRoleStr (e.g.,
// expire, list-keys) must inject the bootstrap key into the context via
// mgmt.WithCallerPubkey — the bootstrap key satisfies m.IsBootstrapKey and is
// allowed unconditionally (BC-2.05.004 Precondition 1 / DI-001).
//
// SVTNs created:
//   - "test-svtn": created; canonical zero key (32 zero bytes, "AAAA...=") registered
//     as control. Required by KeyRegister/Revoke/Expire/ListKeys happy-path tests and
//     KeyRevoke_ControlRequiresConfirm.
//   - "existing-svtn": created; canonical zero key NOT registered (only the random
//     bootstrap key is present). Required by KeyRevoke_ErrorMapping E-ADM-013 subtest.
//   - "rolematch-svtn": created; canonical zero key registered as ROLE_CONTROL.
//     Required by KeyRevoke_ErrorMapping E-ADM-019 subtest (role-mismatch), which
//     revokes with ROLE_BOOTSTRAP to trigger ErrRoleMismatch.
//   - "empty-svtn": created; no additional keys. Required by ListKeys_EmptySliceNotNil.
//   - "nonexistent-svtn": intentionally absent. Required by KeyRegister_ErrorMapping.
//
// A random Ed25519 key is used as the manager bootstrap control key so that the
// canonical zero key remains absent from "existing-svtn" and "empty-svtn" until
// explicitly registered.
func newTestSVTNManagerDetailed(t *testing.T) (*svtnmgmt.SVTNManager, ed25519.PublicKey) {
	t.Helper()

	// Generate a random bootstrap key distinct from the canonical test key (32
	// zero bytes). This ensures Create() does not pre-register the zero key on
	// SVTNs where it must be absent (E-ADM-013 / EC-003 cases).
	bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("newTestSVTNManagerDetailed: generate bootstrap key: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)

	// "test-svtn": create and register the canonical zero key as control so that
	// happy-path revoke / expire / list-keys tests can act on it.
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("newTestSVTNManagerDetailed: create test-svtn: %v", err)
	}
	zeroKey := make([]byte, ed25519.PublicKeySize)
	if _, err := m.RegisterKey("test-svtn", zeroKey, admission.RoleControl); err != nil {
		t.Fatalf("newTestSVTNManagerDetailed: register zero key on test-svtn: %v", err)
	}

	// "existing-svtn": create with no additional keys. The E-ADM-013 subtest
	// expects the canonical zero key to be absent so that revocation returns
	// "key not registered" rather than "SVTN not found".
	if _, err := m.Create("existing-svtn"); err != nil {
		t.Fatalf("newTestSVTNManagerDetailed: create existing-svtn: %v", err)
	}

	// "empty-svtn": create with no additional keys. Required by
	// TestBuildAdminHandlers_ListKeys_EmptySliceNotNil (EC-003).
	if _, err := m.Create("empty-svtn"); err != nil {
		t.Fatalf("newTestSVTNManagerDetailed: create empty-svtn: %v", err)
	}

	// "rolematch-svtn": create and register the canonical zero key as ROLE_CONTROL.
	// Required by KeyRevoke_ErrorMapping/role_mismatch_yields_E-ADM-019, which
	// revokes with ROLE_BOOTSTRAP to trigger ErrRoleMismatch. Using a separate
	// SVTN from "existing-svtn" avoids conflict with the sibling subtest that
	// requires the zero key to be absent.
	if _, err := m.Create("rolematch-svtn"); err != nil {
		t.Fatalf("newTestSVTNManagerDetailed: create rolematch-svtn: %v", err)
	}
	if _, err := m.RegisterKey("rolematch-svtn", zeroKey, admission.RoleControl); err != nil {
		t.Fatalf("newTestSVTNManagerDetailed: register zero key on rolematch-svtn: %v", err)
	}

	// "nonexistent-svtn" is intentionally not created; KeyRegister_ErrorMapping
	// expects E-SVTN-003 for that name.
	return m, bootstrapPub
}

// newTestSVTNManager is a convenience wrapper around newTestSVTNManagerDetailed
// for tests that use the callerRoleStr fallback path (register/revoke supply a
// non-empty role string in the args JSON so the bootstrap key is not needed).
func newTestSVTNManager(t *testing.T) *svtnmgmt.SVTNManager {
	t.Helper()
	m, _ := newTestSVTNManagerDetailed(t)
	return m
}

// assertAdminKeyResult round-trips a handler result through JSON and asserts
// the AC-001 wire contract: key_fingerprint is a non-empty 64-hex-char string
// (SHA256:<base64> prefix + 44-char base64) and timestamp is a non-zero UTC time.
//
// The helper serialises to JSON so that tag names are exercised — if the struct
// tags were wrong the fields would be absent in JSON and the assertions would
// catch it on the parsed side.
func assertAdminKeyResult(t *testing.T, result any) {
	t.Helper()

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("assertAdminKeyResult: marshal: %v", err)
	}

	var wire struct {
		KeyFingerprint string    `json:"key_fingerprint"`
		Timestamp      time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatalf("assertAdminKeyResult: unmarshal: %v", err)
	}

	if wire.KeyFingerprint == "" {
		t.Error("assertAdminKeyResult: key_fingerprint is empty; expected non-empty SHA256:<base64> string")
	}
	// SHA256:<base64> is "SHA256:" (7 chars) + 44-char standard base64 of 32 bytes = 51 chars total.
	const wantFPLen = 51
	if len(wire.KeyFingerprint) != wantFPLen {
		t.Errorf("assertAdminKeyResult: key_fingerprint len=%d, want %d; got %q", len(wire.KeyFingerprint), wantFPLen, wire.KeyFingerprint)
	}

	if wire.Timestamp.IsZero() {
		t.Error("assertAdminKeyResult: timestamp is zero; expected a non-zero UTC time")
	}
}

// TestDecodePublicKey_RejectsBadSize verifies that decodePublicKey returns E-CFG-001
// for decoded keys that are not exactly ed25519.PublicKeySize (32) bytes.
func TestDecodePublicKey_RejectsBadSize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		encoded string
	}{
		// base64 of 16 bytes (too short)
		{"16_bytes", base64.StdEncoding.EncodeToString(make([]byte, 16))},
		// base64 of 64 bytes (too long — private key size)
		{"64_bytes", base64.StdEncoding.EncodeToString(make([]byte, 64))},
		// raw string that is not valid base64 AND not 32 bytes raw
		{"short_raw_not_base64", "hello"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodePublicKey(tc.encoded)
			if err == nil {
				t.Fatalf("expected E-CFG-001 error for %q, got nil", tc.encoded)
			}
			if !strings.Contains(err.Error(), "E-CFG-001") {
				t.Errorf("expected E-CFG-001 in error, got: %v", err)
			}
		})
	}
}

// TestBuildAdminHandlers_KeyRegister_HappyPath asserts that the
// admin.key.register handler returns ok=true for a valid registration request.
// Traces to AC-001; BC-2.05.004 PC-1.
func TestBuildAdminHandlers_KeyRegister_HappyPath(t *testing.T) {
	t.Parallel()
	// Inject bootstrap key: CallerRole in args is empty; resolveAndVerifyCallerRole
	// must take the server-resolved IsBootstrapKey path (BC-2.05.004 Precondition 1 / DI-001).
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var registerFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.register" {
			registerFn = h.Fn
			break
		}
	}
	if registerFn == nil {
		t.Fatal("admin.key.register handler not found in BuildAdminHandlers result")
	}

	args, err := json.Marshal(adminKeyRegisterArgs{
		SVTNName:  "test-svtn",
		PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Role:      "control",
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	result, handlerErr := registerFn(ctx, json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error from register handler: %v", handlerErr)
	}
	assertAdminKeyResult(t, result)
}

// TestBuildAdminHandlers_KeyRevoke_HappyPath asserts that the admin.key.revoke
// handler returns ok=true when role matches and key exists.
// Traces to AC-001; BC-2.05.004 PC-2.
func TestBuildAdminHandlers_KeyRevoke_HappyPath(t *testing.T) {
	t.Parallel()
	// Inject bootstrap key: CallerRole in args is empty; resolveAndVerifyCallerRole
	// must take the server-resolved IsBootstrapKey path (BC-2.05.004 Precondition 1 / DI-001).
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var revokeFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.revoke" {
			revokeFn = h.Fn
			break
		}
	}
	if revokeFn == nil {
		t.Fatal("admin.key.revoke handler not found in BuildAdminHandlers result")
	}

	args, err := json.Marshal(adminKeyRevokeArgs{
		SVTNName:  "test-svtn",
		PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Role:      "control",
		Confirm:   true,
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	result, handlerErr := revokeFn(ctx, json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error from revoke handler: %v", handlerErr)
	}
	assertAdminKeyResult(t, result)
}

// TestBuildAdminHandlers_KeyExpire_HappyPath asserts that admin.key.expire
// returns ok=true for a valid TTL.
// Traces to AC-001; BC-2.05.004 PC-3.
func TestBuildAdminHandlers_KeyExpire_HappyPath(t *testing.T) {
	t.Parallel()
	// admin.key.expire passes "" as callerRoleStr so the handler always takes the
	// server-resolved path. Inject the bootstrap key via WithCallerPubkey so that
	// resolveAndVerifyCallerRole takes the IsBootstrapKey fast-path (allowed
	// unconditionally per BC-2.05.004 Precondition 1 / DI-001).
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var expireFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.expire" {
			expireFn = h.Fn
			break
		}
	}
	if expireFn == nil {
		t.Fatal("admin.key.expire handler not found in BuildAdminHandlers result")
	}

	args, err := json.Marshal(adminKeyExpireArgs{
		SVTNName:  "test-svtn",
		PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		After:     "24h",
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	result, handlerErr := expireFn(ctx, json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error from expire handler: %v", handlerErr)
	}
	assertAdminKeyResult(t, result)
}

// TestBuildAdminHandlers_ListKeys_HappyPath asserts that admin.key.list-keys
// returns ok=true and a non-nil keys array.
// Traces to AC-001; BC-2.05.004 PC-1.
func TestBuildAdminHandlers_ListKeys_HappyPath(t *testing.T) {
	t.Parallel()
	// admin.key.list-keys' args struct has CallerRole but the test omits it (empty
	// string). Inject the bootstrap key into the context so resolveAndVerifyCallerRole
	// takes the server-resolved IsBootstrapKey fast-path (BC-2.05.004 Precondition 1 / DI-001).
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var listFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.list-keys" {
			listFn = h.Fn
			break
		}
	}
	if listFn == nil {
		t.Fatal("admin.key.list-keys handler not found in BuildAdminHandlers result")
	}

	args, err := json.Marshal(adminListKeysArgs{SVTNName: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	result, handlerErr := listFn(ctx, json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error from list-keys handler: %v", handlerErr)
	}

	listResult, ok := result.(adminListKeysResult)
	if !ok {
		t.Fatalf("expected adminListKeysResult, got %T", result)
	}
	// EC-003: Keys must be an empty array, not nil, when no keys are registered.
	if listResult.Keys == nil {
		t.Error("list-keys returned nil Keys; expected empty slice (EC-003)")
	}
}

// TestBuildAdminHandlers_KeyRegister_ErrorMapping asserts that
// admin.key.register propagates ErrSVTNNotFound as E-SVTN-003.
// Traces to AC-001 error mapping; BC-2.05.004 PC-1.
func TestBuildAdminHandlers_KeyRegister_ErrorMapping(t *testing.T) {
	t.Parallel()
	// Inject bootstrap key: CallerRole in args is empty; auth must pass before
	// reaching the SVTN-not-found domain error under test.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var registerFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.register" {
			registerFn = h.Fn
			break
		}
	}
	if registerFn == nil {
		t.Fatal("admin.key.register handler not found")
	}

	// Request for a SVTN that does not exist → ErrSVTNNotFound → E-SVTN-003.
	args, err := json.Marshal(adminKeyRegisterArgs{
		SVTNName:  "nonexistent-svtn",
		PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Role:      "control",
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, handlerErr := registerFn(ctx, json.RawMessage(args))
	if handlerErr == nil {
		t.Fatal("expected error for nonexistent SVTN, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-SVTN-003") {
		t.Errorf("expected E-SVTN-003 in error, got: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_KeyRevoke_ErrorMapping asserts that
// admin.key.revoke maps ErrKeyNotRegistered → E-ADM-013 and
// ErrRoleMismatch → E-ADM-019.
// Traces to AC-001 error mapping; BC-2.05.004 PC-2.
func TestBuildAdminHandlers_KeyRevoke_ErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        adminKeyRevokeArgs
		wantErrCode string
	}{
		{
			name: "key not registered yields E-ADM-013",
			args: adminKeyRevokeArgs{
				SVTNName:  "existing-svtn",
				PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				Role:      "control",
				Confirm:   true,
			},
			wantErrCode: "E-ADM-013",
		},
		{
			name: "role mismatch yields E-ADM-019",
			args: adminKeyRevokeArgs{
				// "rolematch-svtn" has the zero key registered as ROLE_CONTROL;
				// claiming ROLE_CONSOLE triggers ErrRoleMismatch → E-ADM-019.
				// Using a separate SVTN from "existing-svtn" avoids conflict with
				// the sibling subtest that requires the zero key to be absent there.
				SVTNName:  "rolematch-svtn",
				PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				Role:      "console", // claimed console; key registered as control
				Confirm:   false,
			},
			wantErrCode: "E-ADM-019",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Use newTestSVTNManagerDetailed so we get the bootstrap pubkey for
			// context injection. resolveAndVerifyCallerRole now fails-closed when
			// no authenticated key is present in ctx (BC-2.05.004 Precondition 1 / DI-001); inject
			// the bootstrap key so the handler proceeds to the domain-level check.
			m, bootstrapPub := newTestSVTNManagerDetailed(t)
			handlers := BuildAdminHandlers(m, nil)

			var revokeFn func(ctx context.Context, args json.RawMessage) (any, error)
			for _, h := range handlers {
				if h.Command == "admin.key.revoke" {
					revokeFn = h.Fn
					break
				}
			}
			if revokeFn == nil {
				t.Fatal("admin.key.revoke handler not found")
			}

			rawArgs, err := json.Marshal(tc.args)
			if err != nil {
				t.Fatalf("marshal args: %v", err)
			}

			ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
			_, handlerErr := revokeFn(ctx, json.RawMessage(rawArgs))
			if handlerErr == nil {
				t.Fatalf("expected error %s, got nil", tc.wantErrCode)
			}
			if !strings.Contains(handlerErr.Error(), tc.wantErrCode) {
				t.Errorf("expected %s in error, got: %v", tc.wantErrCode, handlerErr)
			}
		})
	}
}

// TestBuildAdminHandlers_NilManager asserts that BuildAdminHandlers panics
// when passed a nil SVTNManager (EC-004).
func TestBuildAdminHandlers_NilManager(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil SVTNManager, got none")
		}
	}()
	BuildAdminHandlers(nil, nil)
}

// TestBuildAdminHandlers_KeyRegister_MalformedJSON asserts that
// admin.key.register returns E-CFG-001 when the args JSON is malformed.
// Traces to AC-001 edge case EC-001; BC-2.05.004 PC-1 precondition (well-formed request).
func TestBuildAdminHandlers_KeyRegister_MalformedJSON(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m, nil)

	var registerFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.register" {
			registerFn = h.Fn
			break
		}
	}
	if registerFn == nil {
		t.Fatal("admin.key.register handler not found")
	}

	// Malformed JSON (EC-001): handler must reject with E-CFG-001.
	_, handlerErr := registerFn(context.Background(), json.RawMessage(`{bad json`))
	if handlerErr == nil {
		t.Fatal("expected error for malformed JSON args, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-CFG-001") {
		t.Errorf("expected E-CFG-001 in error for malformed JSON, got: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_KeyRevoke_UnknownRole asserts that admin.key.revoke
// returns E-CFG-001 when the role field is an unrecognised string.
// Traces to AC-001 edge case EC-002; BC-2.05.004 PC-2 precondition (well-formed request).
func TestBuildAdminHandlers_KeyRevoke_UnknownRole(t *testing.T) {
	t.Parallel()
	// Inject bootstrap key so auth passes and the handler reaches the role
	// validation logic that returns E-CFG-001 for the unknown target role.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var revokeFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.revoke" {
			revokeFn = h.Fn
			break
		}
	}
	if revokeFn == nil {
		t.Fatal("admin.key.revoke handler not found")
	}

	// Unknown role string (EC-002): must reject with E-CFG-001.
	args, err := json.Marshal(map[string]any{
		"svtn":    "test-svtn",
		"pubkey":  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		"role":    "superadmin", // unrecognised role
		"confirm": false,
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, handlerErr := revokeFn(ctx, json.RawMessage(args))
	if handlerErr == nil {
		t.Fatal("expected error for unknown role, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-CFG-001") {
		t.Errorf("expected E-CFG-001 in error for unknown role, got: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_KeyExpire_MissingAfterField asserts that
// admin.key.expire returns E-CFG-001 when the `after` field is absent.
// Traces to AC-001 edge case EC-005; AC-005; BC-2.05.004 PC-3.
func TestBuildAdminHandlers_KeyExpire_MissingAfterField(t *testing.T) {
	t.Parallel()
	// Auth fires before TTL validation (BC-2.05.004 Precondition 1). Inject the
	// bootstrap key so auth passes and the E-CFG-001 path is exercised.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var expireFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.expire" {
			expireFn = h.Fn
			break
		}
	}
	if expireFn == nil {
		t.Fatal("admin.key.expire handler not found")
	}

	// Missing `after` field (EC-005): must reject with E-CFG-001 "missing required field: after".
	args, err := json.Marshal(map[string]any{
		"svtn":   "test-svtn",
		"pubkey": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		// "after" is intentionally absent
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, handlerErr := expireFn(ctx, json.RawMessage(args))
	if handlerErr == nil {
		t.Fatal("expected error for missing after field, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-CFG-001") {
		t.Errorf("expected E-CFG-001 in error for missing after field, got: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_KeyExpire_NegativeTTL asserts that admin.key.expire
// rejects a negative TTL with E-CFG-001 (server-side validation, not CLI-side).
// Traces to AC-005; DI-003 defense-in-depth.
func TestBuildAdminHandlers_KeyExpire_NegativeTTL(t *testing.T) {
	t.Parallel()
	// Auth fires before TTL validation (BC-2.05.004 Precondition 1). Inject the
	// bootstrap key so auth passes and the E-CFG-001 bounds check is exercised.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var expireFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.expire" {
			expireFn = h.Fn
			break
		}
	}
	if expireFn == nil {
		t.Fatal("admin.key.expire handler not found")
	}

	tests := []struct {
		name  string
		after string
	}{
		{"negative ttl", "-1h"},
		{"zero ttl", "0s"},
		{"ttl exceeding 100 years", "876001h"},
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args, err := json.Marshal(adminKeyExpireArgs{
				SVTNName:  "test-svtn",
				PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				After:     tc.after,
			})
			if err != nil {
				t.Fatalf("marshal args: %v", err)
			}

			_, handlerErr := expireFn(ctx, json.RawMessage(args))
			if handlerErr == nil {
				t.Fatalf("expected E-CFG-001 for after=%q, got nil", tc.after)
			}
			if !strings.Contains(handlerErr.Error(), "E-CFG-001") {
				t.Errorf("expected E-CFG-001 for after=%q, got: %v", tc.after, handlerErr)
			}
		})
	}
}

// TestBuildAdminHandlers_FiveHandlers asserts that BuildAdminHandlers returns
// exactly five handlers with the correct command names.
// S-6.07 added admin.svtn.create as the fifth handler (AC-001).
// Traces to AC-001; BC-2.05.004 PC-1..PC-3; BC-2.07.001 PC-1.
func TestBuildAdminHandlers_FiveHandlers(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m, nil)

	want := map[string]bool{
		"admin.key.register":  false,
		"admin.key.revoke":    false,
		"admin.key.expire":    false,
		"admin.key.list-keys": false,
		"admin.svtn.create":   false,
	}
	for _, h := range handlers {
		want[h.Command] = true
	}
	for cmd, found := range want {
		if !found {
			t.Errorf("expected handler %q not found in BuildAdminHandlers result", cmd)
		}
	}
	if len(handlers) != 5 {
		t.Errorf("expected 5 handlers, got %d", len(handlers))
	}
}

// TestBuildAdminHandlers_ListKeys_EmptySliceNotNil asserts that list-keys on
// an SVTN with zero keys returns an empty slice, not nil (EC-003).
// Traces to EC-003; BC-2.05.004 PC-1.
func TestBuildAdminHandlers_ListKeys_EmptySliceNotNil(t *testing.T) {
	// Note: this is a duplicate focus of TestBuildAdminHandlers_ListKeys_HappyPath
	// but isolated for clarity of the EC-003 invariant. Both must fail Red Gate.
	t.Parallel()
	// Inject bootstrap key so resolveAndVerifyCallerRole takes the server-resolved
	// IsBootstrapKey fast-path (no callerRoleStr in list-keys args; BC-2.05.004 Precondition 1 / DI-001).
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var listFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.list-keys" {
			listFn = h.Fn
			break
		}
	}
	if listFn == nil {
		t.Fatal("admin.key.list-keys handler not found")
	}

	// Request list-keys for an SVTN with no registered keys.
	args, err := json.Marshal(adminListKeysArgs{SVTNName: "empty-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	result, handlerErr := listFn(ctx, json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error: %v", handlerErr)
	}
	listResult, ok := result.(adminListKeysResult)
	if !ok {
		t.Fatalf("expected adminListKeysResult, got %T", result)
	}
	if listResult.Keys == nil {
		t.Error("EC-003: list-keys returned nil Keys; must be empty slice not nil")
	}
}

// TestMapAdminError_ErrorWrapping verifies that mapAdminError preserves the
// original sentinel via errors.Is for all arms.
// Traces to F-009 (go.md rule 4: %w wrapping); F-L1-B defense-in-depth arm.
func TestMapAdminError_ErrorWrapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		sentinel    error
		svtn        string
		pub         ed25519.PublicKey // nil for arms where no target key fingerprint is needed
		claimedRole string
		wantCode    string
		wantDetail  string // non-empty: additional substring that must appear in the error
	}{
		{"ErrSVTNNotFound", svtnmgmt.ErrSVTNNotFound, "s", nil, "", "E-SVTN-003", ""},
		{"ErrKeyNotRegistered", admission.ErrKeyNotRegistered, "s", nil, "", "E-ADM-013", ""},
		{"ErrRoleMismatch", svtnmgmt.ErrRoleMismatch, "s", nil, "control", "E-ADM-019", ""},
		{"ErrControlRevocationRequiresConfirm", svtnmgmt.ErrControlRevocationRequiresConfirm, "s", nil, "", "E-ADM-018", ""},
		// ErrInvalidDuration: defense-in-depth arm (F-L1-B). Handler-side guards already
		// produce E-CFG-001 before calling ExpireKey, so this arm is unreachable in
		// production — but an explicit case prevents silent default-arm swallowing if the
		// guard is ever bypassed.
		{"ErrInvalidDuration", svtnmgmt.ErrInvalidDuration, "s", nil, "", "E-CFG-001", "invalid duration"},
		{"ErrBootstrapKeyRevokeForbidden", svtnmgmt.ErrBootstrapKeyRevokeForbidden, "s", nil, "", "E-ADM-020", "cannot revoke the bootstrap key in SVTN s (permanent trust anchor)"},
		{"ErrBootstrapKeyExpireForbidden", svtnmgmt.ErrBootstrapKeyExpireForbidden, "s", nil, "", "E-ADM-021", "cannot expire the bootstrap key in SVTN s (permanent trust anchor)"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := mapAdminError(tc.sentinel, tc.svtn, tc.pub, tc.claimedRole)
			if !strings.Contains(err.Error(), tc.wantCode) {
				t.Errorf("expected %s in error, got: %v", tc.wantCode, err)
			}
			if tc.wantDetail != "" && !strings.Contains(err.Error(), tc.wantDetail) {
				t.Errorf("expected detail substring %q in error, got: %v", tc.wantDetail, err)
			}
			if !errors.Is(err, tc.sentinel) {
				t.Errorf("errors.Is(err, sentinel): expected true, got false; err=%v", err)
			}
		})
	}
}

// TestMapAdminError_DefaultArm verifies the default arm of mapAdminError:
//   - preserves the inner sentinel via %w (errors.Is true)
//   - message contains "unmapped admin error"
//   - message does NOT contain "E-RPC-011" (mgmt.go is sole authority for that code)
//   - message does NOT start with "E-" (no error code stamped)
//
// Traces to mapAdminError default-arm doc comment (admin_handlers.go).
func TestMapAdminError_DefaultArm(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("synthetic-unmapped")
	result := mapAdminError(sentinel, "s", nil, "")
	if !errors.Is(result, sentinel) {
		t.Errorf("errors.Is(result, sentinel): expected true, got false; result=%v", result)
	}
	if !strings.Contains(result.Error(), "unmapped admin error") {
		t.Errorf("expected \"unmapped admin error\" in result, got: %v", result)
	}
	if strings.Contains(result.Error(), "E-RPC-011") {
		t.Errorf("default arm must not stamp E-RPC-011; got: %v", result)
	}
	if strings.HasPrefix(result.Error(), "E-") {
		t.Errorf("default arm must not stamp any E-* code; got: %v", result)
	}
}

// TestBuildAdminHandlers_KeyRevoke_BootstrapKeyForbidden asserts that revoking
// the bootstrap control key returns E-ADM-020 (bootstrap-key-revoke-forbidden).
// Traces to bootstrap revocability invariant; F-010.
func TestBuildAdminHandlers_KeyRevoke_BootstrapKeyForbidden(t *testing.T) {
	t.Parallel()

	// Create manager with a known bootstrap key.
	bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Create() already registered the bootstrap key. Try to revoke it with confirm=true.
	handlers := BuildAdminHandlers(m, nil)
	var revokeFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.revoke" {
			revokeFn = h.Fn
		}
	}
	if revokeFn == nil {
		t.Fatal("admin.key.revoke handler not found")
	}

	bootstrapPubEncoded := base64.StdEncoding.EncodeToString([]byte(bootstrapPub))
	args, err := json.Marshal(adminKeyRevokeArgs{
		SVTNName:  "test-svtn",
		PublicKey: bootstrapPubEncoded,
		Role:      "control",
		Confirm:   true,
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	// Inject the bootstrap key so auth passes (IsBootstrapKey fast-path) and
	// the handler reaches the revocation-forbidden check (E-ADM-020).
	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, handlerErr := revokeFn(ctx, json.RawMessage(args))
	if handlerErr == nil {
		t.Fatal("expected E-ADM-020, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-ADM-020") {
		t.Errorf("expected E-ADM-020, got: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_KeyExpire_BootstrapKeyForbidden asserts that setting a
// TTL on the bootstrap control key returns E-ADM-021 (bootstrap-key-expire-forbidden).
// Mirrors TestBuildAdminHandlers_KeyRevoke_BootstrapKeyForbidden.
// Traces to BC-2.05.004 EC-007 v1.12; F-P18L1-001.
func TestBuildAdminHandlers_KeyExpire_BootstrapKeyForbidden(t *testing.T) {
	t.Parallel()

	// Create a manager with a known bootstrap key.
	bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	handlers := BuildAdminHandlers(m, nil)
	var expireFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.expire" {
			expireFn = h.Fn
			break
		}
	}
	if expireFn == nil {
		t.Fatal("admin.key.expire handler not found")
	}

	bootstrapPubEncoded := base64.StdEncoding.EncodeToString([]byte(bootstrapPub))
	args, err := json.Marshal(adminKeyExpireArgs{
		SVTNName:  "test-svtn",
		PublicKey: bootstrapPubEncoded,
		After:     "1h",
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	// Inject the bootstrap key so auth passes and the handler reaches
	// the expire-forbidden guard (E-ADM-021).
	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, handlerErr := expireFn(ctx, json.RawMessage(args))
	if handlerErr == nil {
		t.Fatal("expected E-ADM-021, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-ADM-021") {
		t.Errorf("expected E-ADM-021, got: %v", handlerErr)
	}
	if !errors.Is(handlerErr, svtnmgmt.ErrBootstrapKeyExpireForbidden) {
		t.Errorf("errors.Is(ErrBootstrapKeyExpireForbidden): expected true; err=%v", handlerErr)
	}
}

// TestAdminKeyEntry_ZeroExpiryOmittedFromJSON asserts that an adminKeyEntry with
// a zero Expiry does not emit an "expiry" field in JSON output.
// encoding/json does not treat zero time.Time as empty for omitempty — using
// *time.Time is required for correct omission (F-P18L1-002).
func TestAdminKeyEntry_ZeroExpiryOmittedFromJSON(t *testing.T) {
	t.Parallel()

	// Zero expiry — must produce no "expiry" key.
	entryNoExpiry := adminKeyEntry{
		Fingerprint: "SHA256:abc",
		Role:        "control",
		// Expiry intentionally nil
	}
	data, err := json.Marshal(entryNoExpiry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "expiry") {
		t.Errorf("zero-expiry entry must not contain 'expiry' field; got: %s", data)
	}

	// Non-zero expiry — must produce an "expiry" key.
	ts := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	entryWithExpiry := adminKeyEntry{
		Fingerprint: "SHA256:def",
		Role:        "access",
		Expiry:      &ts,
	}
	data2, err := json.Marshal(entryWithExpiry)
	if err != nil {
		t.Fatalf("marshal with expiry: %v", err)
	}
	if !strings.Contains(string(data2), "expiry") {
		t.Errorf("non-zero-expiry entry must contain 'expiry' field; got: %s", data2)
	}
}

// TestBuildAdminHandlers_KeyRevoke_ControlRequiresConfirm asserts that
// admin.key.revoke for a control key without confirm=true returns E-ADM-018.
// Traces to AC-002; BC-2.05.004 PC-2; ADR-004.
func TestBuildAdminHandlers_KeyRevoke_ControlRequiresConfirm(t *testing.T) {
	t.Parallel()

	// Use newTestSVTNManagerDetailed so "test-svtn" exists with the zero key
	// registered as ROLE_CONTROL, and we get the bootstrap pubkey for context
	// injection. Without "test-svtn" the handler hits E-SVTN-003 before reaching
	// ErrControlRevocationRequiresConfirm; without the bootstrap key in ctx,
	// resolveAndVerifyCallerRole fails-closed before reaching the confirm check.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var revokeFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.revoke" {
			revokeFn = h.Fn
			break
		}
	}
	if revokeFn == nil {
		t.Fatal("admin.key.revoke handler not found")
	}

	// Control-key revocation without confirm=true must return E-ADM-018
	// (ErrControlRevocationRequiresConfirm), not E-SVTN-003.
	args, err := json.Marshal(adminKeyRevokeArgs{
		SVTNName:  "test-svtn",
		PublicKey: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Role:      "control",
		Confirm:   false, // missing confirm
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, handlerErr := revokeFn(ctx, json.RawMessage(args))
	if handlerErr == nil {
		t.Fatal("expected E-ADM-018 for control revocation without confirm, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-ADM-018") {
		t.Errorf("expected E-ADM-018, got: %v", handlerErr)
	}
}

// TestResolveAndVerifyCallerRole_ServerSidePath exercises the server-side context
// branch of resolveAndVerifyCallerRole (when ctx carries a caller pubkey set by
// the mgmt handshake via mgmt.WithCallerPubkey). Table-driven over four cases:
//
//   - bootstrap_key_allowed: the daemon's own bootstrap key is the trust anchor;
//     resolveAndVerifyCallerRole must return nil regardless of the SVTN registry.
//   - unknown_key_rejected: a key not in the SVTN and not the bootstrap key is
//     rejected with E-ADM-009 (F-P2L1-001 fail-closed).
//   - revoked_key_rejected: a key that was registered then revoked is absent from
//     the SVTN registry; treated as unknown → E-ADM-009.
//   - expired_non_control_key_treated_as_inactive: a control-role key is set to
//     expire in 1ns and then looked up after expiry; CallerKeyRoleActive takes the
//     expiry branch (svtnmgmt.go:433-435) and returns (0, false) → E-ADM-009
//     via the inactive-key route (F-P4L1-003).
//
// Traces to AC-006; BC-2.05.004 Precondition 1 / DI-001; F-P2L1-001 fail-closed.
func TestResolveAndVerifyCallerRole_ServerSidePath(t *testing.T) {
	t.Parallel()

	const svtnName = "test-svtn"
	const cmd = "admin.key.register"

	// newManagerWithSVTN returns a SVTNManager with svtnName pre-created and
	// the bootstrap public key distinct from the test keys.
	newManagerWithSVTN := func(t *testing.T) (*svtnmgmt.SVTNManager, ed25519.PublicKey) {
		t.Helper()
		bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate bootstrap key: %v", err)
		}
		ks := admission.NewAdmittedKeySet()
		m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)
		if _, err := m.Create(svtnName); err != nil {
			t.Fatalf("create SVTN: %v", err)
		}
		return m, bootstrapPub
	}

	t.Run("bootstrap_key_allowed", func(t *testing.T) {
		t.Parallel()
		m, bootstrapPub := newManagerWithSVTN(t)
		// ctx carries the bootstrap key — must be allowed unconditionally.
		ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
		err := resolveAndVerifyCallerRole(ctx, m, nil, svtnName, "", cmd)
		if err != nil {
			t.Errorf("bootstrap key: expected nil, got: %v", err)
		}
	})

	t.Run("unknown_key_rejected", func(t *testing.T) {
		t.Parallel()
		m, _ := newManagerWithSVTN(t)
		// Generate a key that is never registered in the SVTN and is not the
		// bootstrap key. resolveAndVerifyCallerRole must return E-ADM-009
		// (fail-closed: F-P2L1-001).
		unknownPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate unknown key: %v", err)
		}
		ctx := mgmt.WithCallerPubkey(context.Background(), unknownPub)
		gotErr := resolveAndVerifyCallerRole(ctx, m, nil, svtnName, "", cmd)
		if gotErr == nil {
			t.Fatal("unknown key: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("unknown key: expected E-ADM-009 in error, got: %v", gotErr)
		}
	})

	t.Run("revoked_key_rejected", func(t *testing.T) {
		t.Parallel()
		m, _ := newManagerWithSVTN(t)
		// Register a console-role key, then revoke it. CallerKeyRoleActive returns
		// (0, false) for revoked keys (F-P4L1-003) — treated as unregistered → E-ADM-009.
		keyPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		if _, err := m.RegisterKey(svtnName, keyPub, admission.RoleConsole); err != nil {
			t.Fatalf("register key: %v", err)
		}
		if _, err := m.RevokeKey(svtnName, keyPub, admission.RoleConsole, false); err != nil {
			t.Fatalf("revoke key: %v", err)
		}
		// Key is revoked — CallerKeyRoleActive returns (0, false) → E-ADM-009.
		ctx := mgmt.WithCallerPubkey(context.Background(), keyPub)
		gotErr := resolveAndVerifyCallerRole(ctx, m, nil, svtnName, "", cmd)
		if gotErr == nil {
			t.Fatal("revoked key: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("revoked key: expected E-ADM-009 in error, got: %v", gotErr)
		}
	})

	t.Run("expired_non_control_key_treated_as_inactive", func(t *testing.T) {
		t.Parallel()
		m, _ := newManagerWithSVTN(t)
		// Register a control-role key, then immediately expire it with a
		// 1ns TTL so CallerKeyRoleActive takes the expiry branch
		// (svtnmgmt.go:433-435) and returns (0, false).
		// resolveAndVerifyCallerRole treats an inactive key (revoked or expired)
		// as unregistered → E-ADM-009 (F-P4L1-003).
		keyPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		if _, err := m.RegisterKey(svtnName, keyPub, admission.RoleControl); err != nil {
			t.Fatalf("register key: %v", err)
		}
		// Set a 1ns TTL. The key will expire effectively immediately; sleep 1ms
		// to guarantee time.Now().UTC() is past the expiry before the lookup.
		if _, err := m.ExpireKey(svtnName, keyPub, time.Nanosecond); err != nil {
			t.Fatalf("expire key: %v", err)
		}
		time.Sleep(time.Millisecond)
		// CallerKeyRoleActive returns (0, false) via expiry branch →
		// resolveAndVerifyCallerRole returns E-ADM-009 (inactive-key route).
		ctx := mgmt.WithCallerPubkey(context.Background(), keyPub)
		gotErr := resolveAndVerifyCallerRole(ctx, m, nil, svtnName, "", cmd)
		if gotErr == nil {
			t.Fatal("expired key: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expired key: expected E-ADM-009 in error, got: %v", gotErr)
		}
	})
}

// TestResolveAndVerifyCallerRole_OperatorKeyBootstrapGrant exercises the
// F-P4L1-001 bootstrap grant: an operator-set key may call admin.key.register
// when no active control key exists in the SVTN.
//
// Cases:
//   - operator_register_empty_svtn: operator key, no control key in SVTN → allow.
//   - operator_register_with_control_key: operator key, control key present → deny (E-ADM-009).
//   - operator_revoke_empty_svtn: operator key on admin.key.revoke → deny (bootstrap grant is register-only).
//
// Traces to BC-2.05.004 EC-005; F-P4L1-001.
func TestResolveAndVerifyCallerRole_OperatorKeyBootstrapGrant(t *testing.T) {
	t.Parallel()

	const svtnName = "boot-svtn"

	newManagerWithSVTN := func(t *testing.T) *svtnmgmt.SVTNManager {
		t.Helper()
		bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate bootstrap key: %v", err)
		}
		ks := admission.NewAdmittedKeySet()
		m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)
		if _, err := m.Create(svtnName); err != nil {
			t.Fatalf("create SVTN: %v", err)
		}
		return m
	}

	t.Run("operator_register_empty_svtn", func(t *testing.T) {
		t.Parallel()
		m := newManagerWithSVTN(t)
		// Create an operator key and a fresh SVTN with no control key registered yet.
		// The bootstrap key is the daemon's own key (not the operator), so the SVTN
		// has no active control key via HasControlKey.
		operatorPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate operator key: %v", err)
		}
		ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})
		ctx := mgmt.WithCallerPubkey(context.Background(), operatorPub)
		// admin.key.register + operator-set member + no active control key → allow (F-P4L1-001).
		err = resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.register")
		if err != nil {
			t.Errorf("operator bootstrap register: expected nil, got: %v", err)
		}
	})

	t.Run("operator_register_with_control_key", func(t *testing.T) {
		t.Parallel()
		m := newManagerWithSVTN(t)
		// Register a control key in the SVTN first (simulates non-empty SVTN).
		controlPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate control key: %v", err)
		}
		if _, err := m.RegisterKey(svtnName, controlPub, admission.RoleControl); err != nil {
			t.Fatalf("register control key: %v", err)
		}
		operatorPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate operator key: %v", err)
		}
		ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})
		ctx := mgmt.WithCallerPubkey(context.Background(), operatorPub)
		// Control key exists → bootstrap grant does NOT apply → E-ADM-009.
		gotErr := resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.register")
		if gotErr == nil {
			t.Fatal("operator with existing control key: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expected E-ADM-009, got: %v", gotErr)
		}
	})

	t.Run("operator_revoke_empty_svtn", func(t *testing.T) {
		t.Parallel()
		m := newManagerWithSVTN(t)
		operatorPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate operator key: %v", err)
		}
		ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})
		ctx := mgmt.WithCallerPubkey(context.Background(), operatorPub)
		// admin.key.revoke: bootstrap grant is register-only → E-ADM-009.
		gotErr := resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.revoke")
		if gotErr == nil {
			t.Fatal("operator on admin.key.revoke: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expected E-ADM-009, got: %v", gotErr)
		}
	})
}

// TestResolveAndVerifyCallerRole_RevokedExpiredDenial exercises the F-P4L1-003
// ruling: a revoked or past-expiry key is denied with E-ADM-009 even if it
// authenticated successfully at the mgmt connection layer.
//
// Cases:
//   - revoked_control_key_denied: a control key that was revoked cannot exercise admin authority.
//   - past_expiry_control_key_denied: a control key past its expiry time is denied.
//
// Traces to BC-2.05.004 EC-006; F-P4L1-003.
func TestResolveAndVerifyCallerRole_RevokedExpiredDenial(t *testing.T) {
	t.Parallel()

	const svtnName = "deny-svtn"

	newManagerWithSVTN := func(t *testing.T) *svtnmgmt.SVTNManager {
		t.Helper()
		bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate bootstrap key: %v", err)
		}
		ks := admission.NewAdmittedKeySet()
		m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)
		if _, err := m.Create(svtnName); err != nil {
			t.Fatalf("create SVTN: %v", err)
		}
		return m
	}

	t.Run("revoked_control_key_denied", func(t *testing.T) {
		t.Parallel()
		m := newManagerWithSVTN(t)
		// Register a control key, then revoke it. Despite being previously control-role,
		// a revoked key must be denied (F-P4L1-003 / BC-2.05.004 EC-006).
		controlPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate control key: %v", err)
		}
		if _, err := m.RegisterKey(svtnName, controlPub, admission.RoleControl); err != nil {
			t.Fatalf("register control key: %v", err)
		}
		if _, err := m.RevokeKey(svtnName, controlPub, admission.RoleControl, true); err != nil {
			t.Fatalf("revoke control key: %v", err)
		}
		ctx := mgmt.WithCallerPubkey(context.Background(), controlPub)
		// CallerKeyRoleActive returns (0, false) for revoked key → E-ADM-009.
		gotErr := resolveAndVerifyCallerRole(ctx, m, nil, svtnName, "", "admin.key.register")
		if gotErr == nil {
			t.Fatal("revoked control key: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expected E-ADM-009, got: %v", gotErr)
		}
	})

	t.Run("past_expiry_control_key_denied", func(t *testing.T) {
		t.Parallel()
		m := newManagerWithSVTN(t)
		// Register a control key and set a past expiry (1ns TTL, so it expires immediately).
		controlPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate control key: %v", err)
		}
		if _, err := m.RegisterKey(svtnName, controlPub, admission.RoleControl); err != nil {
			t.Fatalf("register control key: %v", err)
		}
		// Use the minimum positive duration so the key expires essentially immediately.
		if _, err := m.ExpireKey(svtnName, controlPub, time.Nanosecond); err != nil {
			t.Fatalf("expire control key: %v", err)
		}
		// Sleep 1ms to ensure now >= expiry.
		time.Sleep(time.Millisecond)

		ctx := mgmt.WithCallerPubkey(context.Background(), controlPub)
		// CallerKeyRoleActive returns (0, false) for past-expiry key → E-ADM-009.
		gotErr := resolveAndVerifyCallerRole(ctx, m, nil, svtnName, "", "admin.key.register")
		if gotErr == nil {
			t.Fatal("past-expiry control key: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expected E-ADM-009, got: %v", gotErr)
		}
	})

	t.Run("revoked_in_operator_set_denied", func(t *testing.T) {
		t.Parallel()
		// F-P5L1-001 regression: a key that is BOTH in the OperatorKeySet AND
		// registered-then-revoked in the SVTN must not receive the bootstrap grant.
		// Before the fix, CallerKeyRoleActive returned (0, false), ops.IsAuthorized
		// returned true, HasNonBootstrapControlKey returned false (revoked keys are
		// excluded from that check), and the bootstrap-grant arm fired — allowing
		// a revoked key to register a new control key.
		m := newManagerWithSVTN(t)
		// Generate a key that will be both in the operator-set and registered
		// (then revoked) in the SVTN.
		dualPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate dual-role key: %v", err)
		}
		// Register it as control, then revoke it.
		if _, err := m.RegisterKey(svtnName, dualPub, admission.RoleControl); err != nil {
			t.Fatalf("register dual-role key: %v", err)
		}
		if _, err := m.RevokeKey(svtnName, dualPub, admission.RoleControl, true); err != nil {
			t.Fatalf("revoke dual-role key: %v", err)
		}
		// Also add it to the OperatorKeySet — this is the attack vector.
		ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{dualPub})
		ctx := mgmt.WithCallerPubkey(context.Background(), dualPub)
		// Must be denied — registered-but-revoked beats operator-set membership.
		gotErr := resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.register")
		if gotErr == nil {
			t.Fatal("revoked key in operator set: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expected E-ADM-009, got: %v", gotErr)
		}
	})
}

// TestResolveAndVerifyCallerRole_EC006_BootstrapAC exercises BC-2.05.004 EC-006
// two-phase boundary: operator-set member + empty SVTN → allow; same key +
// control key already registered → deny (F-P5L2-001).
//
// Cases:
//   - operator_empty_svtn_allow: operator key + no non-bootstrap control key → ok.
//   - operator_after_control_key_deny: same operator key after one control key is
//     registered in the SVTN → E-ADM-009.
//
// Traces to BC-2.05.004 EC-006; F-P4L1-001; F-P5L2-001.
func TestResolveAndVerifyCallerRole_EC006_BootstrapAC(t *testing.T) {
	t.Parallel()

	const svtnName = "ec006-svtn"

	newManagerWithSVTN := func(t *testing.T) *svtnmgmt.SVTNManager {
		t.Helper()
		bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate bootstrap key: %v", err)
		}
		ks := admission.NewAdmittedKeySet()
		m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)
		if _, err := m.Create(svtnName); err != nil {
			t.Fatalf("create SVTN: %v", err)
		}
		return m
	}

	t.Run("operator_empty_svtn_allow", func(t *testing.T) {
		t.Parallel()
		m := newManagerWithSVTN(t)
		operatorPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate operator key: %v", err)
		}
		ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})
		ctx := mgmt.WithCallerPubkey(context.Background(), operatorPub)
		// Operator key + no non-bootstrap control key → bootstrap grant applies.
		if err := resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.register"); err != nil {
			t.Errorf("EC-006 phase 1: expected nil, got: %v", err)
		}
	})

	t.Run("operator_after_control_key_deny", func(t *testing.T) {
		t.Parallel()
		m := newManagerWithSVTN(t)
		// Register a non-bootstrap control key to simulate post-bootstrap state.
		controlPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate control key: %v", err)
		}
		if _, err := m.RegisterKey(svtnName, controlPub, admission.RoleControl); err != nil {
			t.Fatalf("register control key: %v", err)
		}
		operatorPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate operator key: %v", err)
		}
		ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})
		ctx := mgmt.WithCallerPubkey(context.Background(), operatorPub)
		// Non-bootstrap control key now exists → bootstrap grant no longer applies → E-ADM-009.
		gotErr := resolveAndVerifyCallerRole(ctx, m, ops, svtnName, "", "admin.key.register")
		if gotErr == nil {
			t.Fatal("EC-006 phase 2: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("EC-006 phase 2: expected E-ADM-009, got: %v", gotErr)
		}
	})
}

// ── S-6.07: admin.svtn.create handler tests ──────────────────────────────────

// TestBuildAdminHandlers_SVTNCreate_Registered verifies AC-001: BuildAdminHandlers
// registers the "admin.svtn.create" command so that the handler is reachable via
// the mgmt dispatch loop (control-mode daemon only).
//
// BC-2.07.001 PC-1 — handler is registered.
// AC-001 — BuildAdminHandlers registers admin.svtn.create.
func TestBuildAdminHandlers_SVTNCreate_Registered(t *testing.T) {
	t.Parallel()

	// AC-001 / BC-2.07.001 PC-1 — admin.svtn.create must appear in the handler
	// slice returned by BuildAdminHandlers (control-mode daemon only).
	// makeAdminSVTNCreateHandler currently panics → RED.
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m, nil)

	var found bool
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AC-001: 'admin.svtn.create' not registered in BuildAdminHandlers result")
	}
}

// TestAdminSVTNCreate_ControlCallerSucceeds verifies AC-006 (first sub-case):
// a control-role caller can invoke admin.svtn.create and receives a success
// response with svtn_id and bootstrap_fingerprint fields.
//
// BC-2.07.001 PC-1 + PC-2 — create handler dispatches to SVTNManager.Create.
// AC-004 — success response carries svtn_id (hex) and bootstrap_fingerprint (SHA256:<base64>).
// AC-006 sub-case: control-role caller succeeds.
func TestAdminSVTNCreate_ControlCallerSucceeds(t *testing.T) {
	t.Parallel()

	// AC-006 / AC-004 — control-role caller succeeds; response has correct shape.
	// makeAdminSVTNCreateHandler currently panics → RED.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("AC-001: admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	args, err := json.Marshal(adminSVTNCreateArgs{Name: "brand-new-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, err := svtnCreateFn(ctx, json.RawMessage(args))
	if err != nil {
		t.Fatalf("AC-006 control-caller: expected success; got error: %v", err)
	}

	// AC-004: verify wire shape.
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var wire adminSVTNCreateResult
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatalf("unmarshal result to adminSVTNCreateResult: %v", err)
	}
	if wire.SVTNID == "" {
		t.Error("AC-004: svtn_id is empty in response")
	}
	if wire.BootstrapFingerprint == "" {
		t.Error("AC-004: bootstrap_fingerprint is empty in response")
	}
	// AC-004: bootstrap_fingerprint must be SHA256:<base64> format (not hex).
	if !strings.HasPrefix(wire.BootstrapFingerprint, "SHA256:") {
		t.Errorf("AC-004: bootstrap_fingerprint must start with 'SHA256:'; got %q", wire.BootstrapFingerprint)
	}

	_ = bootstrapPub
}

// TestAdminSVTNCreate_NonControlCallerDenied verifies AC-003 and AC-006 (second
// sub-case): a non-control-role caller receives E-ADM-009, and SVTNManager.Create
// is NOT called.
//
// BC-2.07.001 Inv-3 — authority check fires before dispatch.
// AC-003 — non-control-role caller → E-ADM-009.
// AC-006 sub-case: non-control-role caller receives E-ADM-009.
func TestAdminSVTNCreate_NonControlCallerDenied(t *testing.T) {
	t.Parallel()

	// AC-003 / BC-2.07.001 Inv-3 — non-control caller must receive E-ADM-009.
	// makeAdminSVTNCreateHandler currently panics → RED.
	m, _ := newTestSVTNManagerDetailed(t)
	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("AC-001: admin.svtn.create not registered in BuildAdminHandlers")
	}

	// Generate a console-role caller key not registered in any SVTN.
	// resolveAndVerifyCallerRole will deny this as "unregistered" → E-ADM-009.
	consolePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate console key: %v", err)
	}
	ctx := mgmt.WithCallerPubkey(context.Background(), consolePub)

	args, err := json.Marshal(adminSVTNCreateArgs{Name: "should-not-be-created"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, err = svtnCreateFn(ctx, json.RawMessage(args))
	if err == nil {
		t.Fatal("AC-003 non-control-caller: expected E-ADM-009 error; got nil")
	}
	if !strings.Contains(err.Error(), "E-ADM-009") {
		t.Errorf("AC-003: expected 'E-ADM-009' in error; got %q", err.Error())
	}
}

// TestAdminSVTNCreate_DuplicateNameError verifies AC-005 and AC-006 (third
// sub-case): a duplicate SVTN name propagates the SVTN-exists error.
//
// BC-2.07.001 EC-001 — duplicate name → SVTN-exists error.
// AC-005 — duplicate-name caller receives SVTN-exists error (E-RPC-011 wrapping).
// AC-006 sub-case: duplicate-name caller receives SVTN-exists error.
func TestAdminSVTNCreate_DuplicateNameError(t *testing.T) {
	t.Parallel()

	// AC-005 / BC-2.07.001 EC-001 — duplicate SVTN name must propagate error.
	// makeAdminSVTNCreateHandler currently panics → RED.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("AC-001: admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)

	// "test-svtn" was created in newTestSVTNManagerDetailed — this is the duplicate.
	args, err := json.Marshal(adminSVTNCreateArgs{Name: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, err = svtnCreateFn(ctx, json.RawMessage(args))
	if err == nil {
		t.Fatal("AC-005 duplicate-name: expected SVTN-exists error; got nil")
	}
	// AC-005: error message must contain "SVTN already exists".
	if !strings.Contains(err.Error(), "SVTN already exists") {
		t.Errorf("AC-005: expected 'SVTN already exists' in error; got %q", err.Error())
	}

	_ = bootstrapPub
}

// TestAdminSVTNCreate_CallerRoleResolution_FromContext verifies that the handler
// resolves caller role from the authenticated context pubkey (server-side),
// NOT from a client-supplied request field (BC-2.07.001 Inv-3 / S-6.06 pattern).
//
// The test injects a console-role pubkey into ctx WITHOUT any callerRoleStr field
// in the args (there is no caller_role field in adminSVTNCreateArgs). It then
// injects a control-role bootstrap pubkey to confirm success. Both code paths go
// through resolveAndVerifyCallerRole, not a client-supplied field.
//
// AC-003 / BC-2.07.001 Inv-3 — authority check reads from context, not request args.
func TestAdminSVTNCreate_CallerRoleResolution_FromContext(t *testing.T) {
	t.Parallel()

	// BC-2.07.001 Inv-3 — caller role must come from ctx pubkey, not args.
	// makeAdminSVTNCreateHandler currently panics → RED.
	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	ops := mgmt.NewOperatorKeySet(nil)
	handlers := BuildAdminHandlers(m, ops)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("AC-001: admin.svtn.create not registered in BuildAdminHandlers")
	}

	argsJSON, err := json.Marshal(adminSVTNCreateArgs{Name: "ctx-role-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	// Sub-case A: console-role pubkey in ctx (not registered in any SVTN) → E-ADM-009.
	// No caller_role field in args — the only source of authority is the context pubkey.
	consolePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate console key: %v", err)
	}
	ctxConsole := mgmt.WithCallerPubkey(context.Background(), consolePub)
	_, err = svtnCreateFn(ctxConsole, json.RawMessage(argsJSON))
	if err == nil {
		t.Error("AC-003 ctx-role A: console-role context pubkey: expected E-ADM-009; got nil")
	} else if !strings.Contains(err.Error(), "E-ADM-009") {
		t.Errorf("AC-003 ctx-role A: expected E-ADM-009 in error; got %q", err.Error())
	}

	// Sub-case B: bootstrap (control-role) pubkey in ctx → success.
	// The handler must read the role from ctx, not from an absent request field.
	argsJSON2, err := json.Marshal(adminSVTNCreateArgs{Name: "ctx-role-svtn-b"})
	if err != nil {
		t.Fatalf("marshal args B: %v", err)
	}
	ctxControl := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	result, err := svtnCreateFn(ctxControl, json.RawMessage(argsJSON2))
	if err != nil {
		t.Errorf("AC-003 ctx-role B: bootstrap context pubkey: expected success; got error: %v", err)
	}
	if result == nil {
		t.Error("AC-003 ctx-role B: expected non-nil result; got nil")
	}

	_ = bootstrapPub
}

// TestAdminSVTNCreate_ArgsValidation_E_CFG_001 verifies that malformed or
// missing-name args return E-CFG-001, consistent with all other admin handlers.
//
// F-P1L1-002: non-duplicate failure paths must be code-stamped.
// BC-2.07.001 PC-1 — handler validates required args before dispatch.
func TestAdminSVTNCreate_ArgsValidation_E_CFG_001(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)

	tests := []struct {
		name    string
		rawArgs json.RawMessage
	}{
		{
			name:    "malformed_json",
			rawArgs: json.RawMessage(`{bad json`),
		},
		{
			name:    "empty_name_field",
			rawArgs: func() json.RawMessage { b, _ := json.Marshal(adminSVTNCreateArgs{Name: ""}); return b }(),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := svtnCreateFn(ctx, tc.rawArgs)
			if err == nil {
				t.Fatalf("expected E-CFG-001 error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), "E-CFG-001") {
				t.Errorf("expected E-CFG-001 in error for %s; got: %v", tc.name, err)
			}
		})
	}
}

// TestAdminSVTNCreate_DuplicateName_E_SVTN_001 verifies that a duplicate SVTN
// name returns E-SVTN-001 (not E-ADM-004 and not any other code).
//
// F-P1L1-003: E-ADM-004 → E-SVTN-001 for duplicate name.
// BC-2.07.001 EC-001 — duplicate name → E-SVTN-001 "SVTN already exists: <name>".
func TestAdminSVTNCreate_DuplicateName_E_SVTN_001(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	// "test-svtn" already exists in newTestSVTNManagerDetailed.
	args, err := json.Marshal(adminSVTNCreateArgs{Name: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
	if gotErr == nil {
		t.Fatal("expected error for duplicate SVTN name, got nil")
	}
	// F-P1L1-003: must stamp E-SVTN-001, not E-ADM-004.
	if !strings.Contains(gotErr.Error(), "E-SVTN-001") {
		t.Errorf("expected E-SVTN-001 in error; got: %v", gotErr)
	}
	if strings.Contains(gotErr.Error(), "E-ADM-004") {
		t.Errorf("must not stamp E-ADM-004 (address collision) for duplicate SVTN name; got: %v", gotErr)
	}
	// F-P1L2-001: error must be produced via errors.Is check, not string matching.
	if !errors.Is(gotErr, svtnmgmt.ErrSVTNAlreadyExists) {
		t.Errorf("errors.Is(err, ErrSVTNAlreadyExists): expected true; got false; err=%v", gotErr)
	}
}

// TestAdminSVTNCreate_NoStutterInDuplicateMessage verifies that the duplicate
// SVTN name error does not repeat "SVTN already exists" more than once.
//
// F-P1L1-004: stutter fix — message derived from args.name, not from wrapping
// err.Error() (which already contains "SVTN already exists").
// BC-2.07.001 EC-001 — canonical format: "E-SVTN-001: SVTN already exists: <name>".
func TestAdminSVTNCreate_NoStutterInDuplicateMessage(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	// "test-svtn" already exists.
	args, err := json.Marshal(adminSVTNCreateArgs{Name: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
	if gotErr == nil {
		t.Fatal("expected error for duplicate SVTN name, got nil")
	}

	// F-P1L1-004: count occurrences of "SVTN already exists" — must be exactly 1.
	msg := gotErr.Error()
	const phrase = "SVTN already exists"
	count := strings.Count(msg, phrase)
	if count != 1 {
		t.Errorf("F-P1L1-004 stutter: expected exactly 1 occurrence of %q in error message; got %d; full message: %q", phrase, count, msg)
	}
}

// TestAdminSVTNCreate_BootstrapOnly_CrossSVTNKeyDenied verifies that only the
// daemon bootstrap key (IsBootstrapKey) may invoke admin.svtn.create. A key
// registered as control in an existing SVTN (cross-SVTN control key) is NOT
// authorized and must receive E-ADM-009.
//
// F-P1L1-005: bootstrap-only authority model.
// BC-2.07.001 Inv-3 — admin.svtn.create requires the daemon bootstrap key with
// RoleControl; cross-SVTN control-role keys are NOT authorized.
func TestAdminSVTNCreate_BootstrapOnly_CrossSVTNKeyDenied(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	argsNewSVTN, err := json.Marshal(adminSVTNCreateArgs{Name: "new-svtn-for-bootstrap-test"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	t.Run("bootstrap_key_succeeds", func(t *testing.T) {
		t.Parallel()
		ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
		result, err := svtnCreateFn(ctx, json.RawMessage(argsNewSVTN))
		if err != nil {
			t.Errorf("bootstrap key with RoleControl: expected success; got: %v", err)
		}
		if result == nil {
			t.Error("bootstrap key: expected non-nil result")
		}
	})

	t.Run("cross_svtn_control_key_denied", func(t *testing.T) {
		t.Parallel()
		// "test-svtn" has the zero key registered as control in newTestSVTNManagerDetailed.
		// Inject the zero key (registered as control in test-svtn) as the caller.
		// This key is a cross-SVTN key — it is NOT the bootstrap key, and the target
		// SVTN "cross-svtn-target" does not yet exist, so CallerKeyRoleActive returns
		// (0, false) → resolveAndVerifyCallerRole fails-closed with E-ADM-009.
		crossSVTNControl, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate cross-svtn key: %v", err)
		}
		// Register this key as control in "test-svtn" (an existing SVTN).
		if _, err := m.RegisterKey("test-svtn", crossSVTNControl, admission.RoleControl); err != nil {
			t.Fatalf("register cross-SVTN control key: %v", err)
		}

		// Now try to create a new SVTN using this cross-SVTN control key.
		argsNewSVTN2, err := json.Marshal(adminSVTNCreateArgs{Name: "cross-svtn-target"})
		if err != nil {
			t.Fatalf("marshal args: %v", err)
		}
		ctx := mgmt.WithCallerPubkey(context.Background(), crossSVTNControl)
		_, gotErr := svtnCreateFn(ctx, json.RawMessage(argsNewSVTN2))
		if gotErr == nil {
			t.Fatal("cross-SVTN control key: expected E-ADM-009; got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("cross-SVTN control key: expected E-ADM-009; got: %v", gotErr)
		}
	})

	t.Run("non_control_role_key_denied", func(t *testing.T) {
		t.Parallel()
		// An unregistered key (no role anywhere) must be denied.
		consolePub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate console key: %v", err)
		}
		argsNewSVTN3, err := json.Marshal(adminSVTNCreateArgs{Name: "non-control-target"})
		if err != nil {
			t.Fatalf("marshal args: %v", err)
		}
		ctx := mgmt.WithCallerPubkey(context.Background(), consolePub)
		_, gotErr := svtnCreateFn(ctx, json.RawMessage(argsNewSVTN3))
		if gotErr == nil {
			t.Fatal("non-control-role key: expected E-ADM-009; got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("non-control-role key: expected E-ADM-009; got: %v", gotErr)
		}
	})
}

// TestAdminSVTNCreate_NonBootstrapControlKey_RejectsWithEADM009 verifies that
// a non-bootstrap key with RoleControl receives E-ADM-009 (insufficient authority)
// rather than E-SVTN-001 (duplicate name) when calling admin.svtn.create.
//
// This is the F-P3L2-02 test: the handler must not fall through to m.Create() for
// non-bootstrap keys — the bootstrap-only gate must fire first and return E-ADM-009.
//
// Ruling-7 / BC-2.07.001 Inv-3 — non-bootstrap control-role key denied (E-ADM-009).
func TestAdminSVTNCreate_NonBootstrapControlKey_RejectsWithEADM009(t *testing.T) {
	t.Parallel()

	m, _ := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	// Generate a key that is NOT the bootstrap key but has RoleControl in "test-svtn".
	nonBootstrapControlPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate non-bootstrap control key: %v", err)
	}
	if _, err := m.RegisterKey("test-svtn", nonBootstrapControlPub, admission.RoleControl); err != nil {
		t.Fatalf("register non-bootstrap control key: %v", err)
	}

	// "test-svtn" already exists. If the handler incorrectly bypassed the bootstrap
	// check and called m.Create("test-svtn"), it would return E-SVTN-001.
	// The correct behavior is E-ADM-009 from the bootstrap-only gate, which fires
	// BEFORE m.Create() is ever called.
	args, err := json.Marshal(adminSVTNCreateArgs{Name: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	ctx := mgmt.WithCallerPubkey(context.Background(), nonBootstrapControlPub)
	_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
	if gotErr == nil {
		t.Fatal("non-bootstrap RoleControl key: expected E-ADM-009; got nil")
	}
	if !strings.Contains(gotErr.Error(), "E-ADM-009") {
		t.Errorf("non-bootstrap RoleControl key: expected E-ADM-009 (not E-SVTN-001); got: %v", gotErr)
	}
	if strings.Contains(gotErr.Error(), "E-SVTN-001") {
		t.Errorf("non-bootstrap RoleControl key: got E-SVTN-001 (handler fell through to Create); want E-ADM-009")
	}
}

// TestAdminSVTNCreate_MutationTest_RoleControlCheckMustFireIndependently is the
// Ruling-7 mutation test (AC-003 requirement, F-Impl-002). It verifies that the
// RoleControl check in the handler fires INDEPENDENTLY of IsBootstrapKey — i.e.,
// when a key passes IsBootstrapKey but has no RoleControl in any existing SVTN,
// the handler must return E-ADM-009, not allow the create to proceed.
//
// Scenario: "demoted bootstrap key" — a key that still satisfies IsBootstrapKey
// (structural check) but has been removed from its SVTN's keySet as RoleControl
// (simulating a future key rotation flow). Without the explicit RoleControl check
// (Ruling-7), the handler would silently allow this caller.
//
// The SeedSVTNWithoutBootstrapKeyForTest method is used to construct this state:
// it creates an SVTN record without registering the bootstrap key, producing the
// HasAnySVTN()==true / BootstrapKeyHasControlRole()==false condition.
//
// Ruling-7 / AC-003 mutation test / BC-2.07.001 Inv-3 defense-in-depth.
func TestAdminSVTNCreate_MutationTest_RoleControlCheckMustFireIndependently(t *testing.T) {
	t.Parallel()

	bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate bootstrap key: %v", err)
	}
	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)

	// Seed an SVTN WITHOUT registering the bootstrap key as RoleControl.
	// This simulates the "demoted bootstrap key" post-rotation state.
	// After this call: HasAnySVTN() == true AND BootstrapKeyHasControlRole() == false.
	if err := m.SeedSVTNWithoutBootstrapKeyForTest("demoted-bootstrap-svtn"); err != nil {
		t.Fatalf("SeedSVTNWithoutBootstrapKeyForTest: %v", err)
	}

	// Verify the precondition: HasAnySVTN true, BootstrapKeyHasControlRole false.
	if !m.HasAnySVTN() {
		t.Fatal("precondition: HasAnySVTN() must be true after seeding")
	}
	if m.BootstrapKeyHasControlRole() {
		t.Fatal("precondition: BootstrapKeyHasControlRole() must be false after SeedSVTNWithoutBootstrapKeyForTest")
	}

	handlers := BuildAdminHandlers(m, nil)
	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	args, err := json.Marshal(adminSVTNCreateArgs{Name: "should-not-be-created"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	// Bootstrap key passes IsBootstrapKey but has no RoleControl — Ruling-7 must fire.
	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
	if gotErr == nil {
		t.Fatal("Ruling-7 mutation: demoted bootstrap key must be denied E-ADM-009; got nil (role check is missing or skipped)")
	}
	if !strings.Contains(gotErr.Error(), "E-ADM-009") {
		t.Errorf("Ruling-7 mutation: expected E-ADM-009; got: %v", gotErr)
	}
}

// TestAdminSVTNCreateResult_JSONFieldNames verifies that adminSVTNCreateResult
// serialises with correct JSON field names per AC-004 wire contract.
//
// AC-004 — svtn_id (hex) and bootstrap_fingerprint (SHA256:<base64>).
func TestAdminSVTNCreateResult_JSONFieldNames(t *testing.T) {
	t.Parallel()

	// AC-004 — wire field names for admin.svtn.create success response.
	original := adminSVTNCreateResult{
		SVTNID:               "aabbccddeeff0011aabbccddeeff0011",
		BootstrapFingerprint: "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}

	if _, ok := raw["svtn_id"]; !ok {
		t.Error("AC-004: missing JSON field 'svtn_id'")
	}
	if _, ok := raw["bootstrap_fingerprint"]; !ok {
		t.Error("AC-004: missing JSON field 'bootstrap_fingerprint'")
	}

	// Round-trip.
	var decoded adminSVTNCreateResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal round-trip: %v", err)
	}
	if decoded.SVTNID != original.SVTNID {
		t.Errorf("svtn_id round-trip: got %q; want %q", decoded.SVTNID, original.SVTNID)
	}
	if decoded.BootstrapFingerprint != original.BootstrapFingerprint {
		t.Errorf("bootstrap_fingerprint round-trip: got %q; want %q", decoded.BootstrapFingerprint, original.BootstrapFingerprint)
	}
}

// ── Pass-2 tests ──────────────────────────────────────────────────────────────

// TestAdminSVTNCreate_CryptoRandFailure_E_INT_001 verifies that a non-duplicate
// Create() failure (e.g. internal rand.Read failure) is stamped E-INT-001 and
// does NOT fall through to E-RPC-011 (F-P2L1-004).
//
// The test uses a thin svtnmgmt.SVTNManager wrapper via a local erroring Create
// stub injected through a fake manager that returns a synthetic error. Because
// SVTNManager is a concrete struct without an interface, we verify the E-INT-001
// path by creating a real SVTNManager and making Create return an error via a
// duplicate name (ErrSVTNAlreadyExists) — then confirming a non-duplicate path
// by using a mock that wraps the error. However, SVTNManager.Create only returns
// F-P2L1-004: non-duplicate Create() errors → E-INT-001.
//
// This test drives the real handler through the E-INT-001 branch by injecting a
// zero-byte io.Reader into SVTNManager via NewSVTNManagerWithRandSource. When
// Create() calls io.ReadFull on the injected reader, it receives io.ErrUnexpectedEOF
// (0 bytes read into the 16-byte SVTN ID buffer), causing Create() to return a
// non-ErrSVTNAlreadyExists error. The handler must stamp E-INT-001.
//
// Prior implementation (Pass-2) was vacuous: it constructed a local error string
// and asserted substring against itself without calling the handler. This test
// drives the handler end-to-end (F-P3L2-01 remediation).
func TestAdminSVTNCreate_CryptoRandFailure_E_INT_001(t *testing.T) {
	t.Parallel()

	bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate bootstrap key: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	// Inject a zero-byte reader: io.ReadFull will return io.ErrUnexpectedEOF
	// when trying to fill the 16-byte SVTN ID buffer (strings.NewReader("")
	// returns 0 bytes on first Read). This exercises the rand failure path in
	// SVTNManager.Create without patching crypto/rand globally.
	failingRand := strings.NewReader("")
	m := svtnmgmt.NewSVTNManagerWithRandSource(ks, bootstrapPub, failingRand)

	handlers := BuildAdminHandlers(m, nil)
	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	args, err := json.Marshal(adminSVTNCreateArgs{Name: "rand-fail-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
	if gotErr == nil {
		t.Fatal("expected E-INT-001 error from rand failure; got nil")
	}
	if !strings.Contains(gotErr.Error(), "E-INT-001") {
		t.Errorf("expected E-INT-001 stamp on rand failure path; got: %v", gotErr)
	}
	// Verify the inner error is wrapped (go.md rule 4 / F-Impl-003: %w used).
	// strings.NewReader("") returns io.EOF on the first read (0 bytes read into
	// a 16-byte buffer); io.ReadFull returns io.EOF when no bytes were read at
	// all (vs io.ErrUnexpectedEOF when some-but-not-all bytes were read).
	if !errors.Is(gotErr, io.EOF) {
		t.Errorf("expected errors.Is(err, io.EOF) == true (error chain preserved via %%w); got: %v", gotErr)
	}
}

// TestAdminSVTNCreate_ArgsValidation_E_CFG_001_Exhaustive extends the basic
// args-validation test with whitespace-only names, control characters, and
// names that exceed 255 bytes (F-P2L2 exhaustive validation).
//
// BC-2.07.001 PC-1 — handler validates required args before dispatch.
// F-P2L2 exhaustive validation.
func TestAdminSVTNCreate_ArgsValidation_E_CFG_001_Exhaustive(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)

	// Build a name that is exactly 256 bytes (one over the 255-byte limit).
	longName := strings.Repeat("a", 256)

	tests := []struct {
		name    string
		svtnArg adminSVTNCreateArgs
		rawJSON []byte // if non-nil, bypasses json.Marshal and sends raw bytes directly
	}{
		{
			name:    "nil_name_empty_string",
			svtnArg: adminSVTNCreateArgs{Name: ""},
		},
		{
			name:    "whitespace_only_spaces",
			svtnArg: adminSVTNCreateArgs{Name: "   "},
		},
		{
			name:    "whitespace_only_tab_newline",
			svtnArg: adminSVTNCreateArgs{Name: "\t\n"},
		},
		{
			name:    "control_char_null_byte",
			svtnArg: adminSVTNCreateArgs{Name: "foo\x00bar"},
		},
		{
			name:    "control_char_stx",
			svtnArg: adminSVTNCreateArgs{Name: "foo\x02bar"},
		},
		{
			name:    "control_char_del_0x7f",
			svtnArg: adminSVTNCreateArgs{Name: "foo\x7fbar"},
		},
		{
			name:    "name_exceeds_255_bytes",
			svtnArg: adminSVTNCreateArgs{Name: longName},
		},
		// C1 control: U+0085 NEL (NEXT LINE), encoded as 0xC2 0x85 in UTF-8.
		// Note: invalid UTF-8 bytes cannot be tested here because json.Unmarshal
		// silently replaces them with U+FFFD before validateSVTNName runs.
		// See TestValidateSVTNName for direct function-level coverage of that case.
		// Valid UTF-8 but unicode.IsControl returns true (category Cc).
		{
			name:    "c1_control_nel_u0085",
			rawJSON: []byte("{\"name\":\"foo\xc2\x85bar\"}"),
		},
		// Line separator: U+2028, encoded as 0xE2 0x80 0xA8 in UTF-8.
		// Valid UTF-8; not caught by unicode.IsControl (Zl, not Cc);
		// caught by explicit r == '\u2028' check in validateSVTNName (F-Impl-001).
		{
			name:    "unicode_line_separator_u2028",
			rawJSON: []byte("{\"name\":\"foo\xe2\x80\xa8bar\"}"),
		},
		// Paragraph separator: U+2029, encoded as 0xE2 0x80 0xA9 in UTF-8.
		{
			name:    "unicode_paragraph_separator_u2029",
			rawJSON: []byte("{\"name\":\"foo\xe2\x80\xa9bar\"}"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var rawArgs []byte
			if tc.rawJSON != nil {
				rawArgs = tc.rawJSON
			} else {
				var err error
				rawArgs, err = json.Marshal(tc.svtnArg)
				if err != nil {
					t.Fatalf("marshal args: %v", err)
				}
			}
			_, gotErr := svtnCreateFn(ctx, json.RawMessage(rawArgs))
			if gotErr == nil {
				t.Fatalf("expected E-CFG-001 for %s, got nil", tc.name)
			}
			if !strings.Contains(gotErr.Error(), "E-CFG-001") {
				t.Errorf("expected E-CFG-001 in error for %s; got: %v", tc.name, gotErr)
			}
		})
	}
}

// TestValidateSVTNName exercises validateSVTNName directly, covering cases that
// cannot reach it through the handler path because json.Unmarshal sanitises
// invalid UTF-8 bytes before they reach the validation layer (F-Impl-001).
func TestValidateSVTNName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errCode string
	}{
		// valid inputs
		{name: "simple_ascii", input: "prod", wantErr: false},
		{name: "unicode_letters", input: "réseau-α", wantErr: false},
		{name: "max_length_255_bytes", input: strings.Repeat("a", 255), wantErr: false},

		// empty / whitespace
		{name: "empty_string", input: "", wantErr: true, errCode: "E-CFG-001"},
		{name: "whitespace_only", input: "   ", wantErr: true, errCode: "E-CFG-001"},

		// length
		{name: "exceeds_255_bytes", input: strings.Repeat("a", 256), wantErr: true, errCode: "E-CFG-001"},

		// invalid UTF-8 — tested here directly because json.Unmarshal sanitises
		// these bytes before validateSVTNName can see them through the handler.
		{name: "invalid_utf8_lone_byte_0xff", input: "foo\xffbar", wantErr: true, errCode: "E-CFG-001"},
		{name: "invalid_utf8_lone_byte_0xfe", input: "\xfe", wantErr: true, errCode: "E-CFG-001"},
		{name: "invalid_utf8_truncated_sequence", input: "foo\xe2\x80", wantErr: true, errCode: "E-CFG-001"},

		// ASCII C0 controls
		{name: "c0_null_u0000", input: "foo\x00bar", wantErr: true, errCode: "E-CFG-001"},
		{name: "c0_tab_u0009", input: "foo\x09bar", wantErr: true, errCode: "E-CFG-001"},
		{name: "c0_lf_u000a", input: "foo\nbar", wantErr: true, errCode: "E-CFG-001"},
		{name: "c0_cr_u000d", input: "foo\rbar", wantErr: true, errCode: "E-CFG-001"},
		{name: "c0_del_u007f", input: "foo\x7fbar", wantErr: true, errCode: "E-CFG-001"},

		// C1 controls (U+0080–U+009F, category Cc) — caught by unicode.IsControl
		{name: "c1_nel_u0085", input: "foo\xc2\x85bar", wantErr: true, errCode: "E-CFG-001"},
		{name: "c1_u0080", input: "foo\xc2\x80bar", wantErr: true, errCode: "E-CFG-001"},
		{name: "c1_u009f", input: "foo\xc2\x9fbar", wantErr: true, errCode: "E-CFG-001"},

		// Line/paragraph separators (Zl/Zp categories, NOT caught by unicode.IsControl)
		// — caught by explicit rune checks (F-Impl-001).
		{name: "line_sep_u2028", input: "foo\xe2\x80\xa8bar", wantErr: true, errCode: "E-CFG-001"},
		{name: "para_sep_u2029", input: "foo\xe2\x80\xa9bar", wantErr: true, errCode: "E-CFG-001"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSVTNName(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("validateSVTNName(%q): expected error containing %q, got nil", tc.input, tc.errCode)
				}
				if !strings.Contains(err.Error(), tc.errCode) {
					t.Errorf("validateSVTNName(%q): expected %q in error, got: %v", tc.input, tc.errCode, err)
				}
			} else {
				if err != nil {
					t.Errorf("validateSVTNName(%q): expected nil, got: %v", tc.input, err)
				}
			}
		})
	}
}

// TestAdminSVTNCreate_DuplicateName_E_SVTN_001_VacuityControl extends
// TestAdminSVTNCreate_DuplicateName_E_SVTN_001 with a positive-control
// assertion: verify the pre-seeded SVTN actually exists via Lookup before
// triggering the duplicate-create. Without this, the test would pass vacuously
// if the seed had silently failed.
//
// F-P2 vacuity control — pre-condition: seed SVTN must be present.
// BC-2.07.001 EC-001 — duplicate name → E-SVTN-001.
func TestAdminSVTNCreate_DuplicateName_E_SVTN_001_VacuityControl(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	handlers := BuildAdminHandlers(m, nil)

	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)

	// Positive-control assertion: "test-svtn" must exist before we attempt the
	// duplicate create. list-keys on "test-svtn" succeeds iff the SVTN is present.
	// If the SVTN were absent, list-keys would return E-SVTN-003, not an empty list.
	var listFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.key.list-keys" {
			listFn = h.Fn
			break
		}
	}
	if listFn == nil {
		t.Fatal("admin.key.list-keys handler not found")
	}
	listArgs, err := json.Marshal(adminListKeysArgs{SVTNName: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal list-keys args: %v", err)
	}
	_, listErr := listFn(ctx, json.RawMessage(listArgs))
	if listErr != nil {
		t.Fatalf("vacuity-control: test-svtn must already exist before duplicate-create test; list-keys returned: %v", listErr)
	}

	// Now confirm that attempting to create "test-svtn" again returns E-SVTN-001.
	args, err := json.Marshal(adminSVTNCreateArgs{Name: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
	if gotErr == nil {
		t.Fatal("expected E-SVTN-001 for duplicate SVTN name, got nil")
	}
	if !strings.Contains(gotErr.Error(), "E-SVTN-001") {
		t.Errorf("expected E-SVTN-001 in error; got: %v", gotErr)
	}
	if !errors.Is(gotErr, svtnmgmt.ErrSVTNAlreadyExists) {
		t.Errorf("errors.Is(err, ErrSVTNAlreadyExists): expected true; got false; err=%v", gotErr)
	}
}

// TestAdminSVTNCreate_BootstrapOnly_CrossSVTNKeyDenied_CreateNotCalled verifies
// that SVTNManager.Create is never called when a non-bootstrap caller is denied.
//
// This test adds the Create-not-called and name-collision sub-cases required by
// F-P2L1-001 to ensure the bootstrap-only gate is not accidentally bypassed.
//
// Sub-case: name-collision with existing SVTN — E-ADM-009 fires before Create.
// Sub-case: cross-SVTN control key, distinct name — E-ADM-009 + Create not called.
//
// Traces to Ruling-5; F-P2L1-001; BC-2.07.001 Inv-3.
func TestAdminSVTNCreate_BootstrapOnly_CrossSVTNKeyDenied_CreateNotCalled(t *testing.T) {
	t.Parallel()

	m, _ := newTestSVTNManagerDetailed(t)
	// "existing-svtn" was created in newTestSVTNManagerDetailed.

	// Generate a cross-SVTN control key and register it in "test-svtn".
	crossSVTNControl, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate cross-svtn key: %v", err)
	}
	if _, err := m.RegisterKey("test-svtn", crossSVTNControl, admission.RoleControl); err != nil {
		t.Fatalf("register cross-SVTN control key: %v", err)
	}

	handlers := BuildAdminHandlers(m, nil)
	var svtnCreateFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.svtn.create" {
			svtnCreateFn = h.Fn
			break
		}
	}
	if svtnCreateFn == nil {
		t.Fatal("admin.svtn.create not registered in BuildAdminHandlers")
	}

	ctx := mgmt.WithCallerPubkey(context.Background(), crossSVTNControl)

	t.Run("name_collision_e_adm_009_before_create", func(t *testing.T) {
		t.Parallel()
		// "existing-svtn" already exists. Even with a name collision, the bootstrap
		// pre-check must fire E-ADM-009 BEFORE any Create attempt (Ruling-5 / Inv-3).
		// Positive-control: verify "existing-svtn" is present first.
		var listFn func(ctx context.Context, args json.RawMessage) (any, error)
		for _, h := range handlers {
			if h.Command == "admin.key.list-keys" {
				listFn = h.Fn
				break
			}
		}
		if listFn != nil {
			listArgs, merr := json.Marshal(adminListKeysArgs{SVTNName: "existing-svtn"})
			if merr != nil {
				t.Fatalf("marshal list-keys args: %v", merr)
			}
			// list-keys admits any authenticated caller — use crossSVTNControl which
			// is a registered control key in test-svtn (admitted, active). The SVTN
			// "existing-svtn" was created in newTestSVTNManagerDetailed.
			_, listErr := listFn(mgmt.WithCallerPubkey(context.Background(), crossSVTNControl), json.RawMessage(listArgs))
			if listErr != nil {
				// E-SVTN-003 means "existing-svtn" is absent — the test fixture is broken.
				// Any error here is a setup failure: the positive-control fails and the
				// subsequent E-ADM-009 assertion on svtn.create is not meaningful (F-P3L2-03).
				t.Fatalf("vacuity-control: existing-svtn must be present before name-collision test; list-keys returned: %v", listErr)
			}
		}

		args, err := json.Marshal(adminSVTNCreateArgs{Name: "existing-svtn"})
		if err != nil {
			t.Fatalf("marshal args: %v", err)
		}
		_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
		if gotErr == nil {
			t.Fatal("expected E-ADM-009; got nil")
		}
		// Must get E-ADM-009 (bootstrap pre-check), NOT E-SVTN-001 (duplicate name).
		// E-SVTN-001 would indicate Create was called before the auth check.
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expected E-ADM-009 (auth pre-check fires before Create); got: %v", gotErr)
		}
		if strings.Contains(gotErr.Error(), "E-SVTN-001") {
			t.Errorf("E-SVTN-001 appeared — Create was called before auth check (Ruling-5 violated); got: %v", gotErr)
		}
	})

	t.Run("cross_svtn_control_key_distinct_name_create_not_called", func(t *testing.T) {
		t.Parallel()
		// Cross-SVTN control key with a fresh name (B). E-ADM-009 must fire.
		// Create-not-called is proven by the E-ADM-009 return (if Create ran,
		// it would return success or E-SVTN-001, not E-ADM-009).
		args, err := json.Marshal(adminSVTNCreateArgs{Name: "fresh-name-B"})
		if err != nil {
			t.Fatalf("marshal args: %v", err)
		}
		_, gotErr := svtnCreateFn(ctx, json.RawMessage(args))
		if gotErr == nil {
			t.Fatal("cross-SVTN control key distinct name: expected E-ADM-009; got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("cross-SVTN control key: expected E-ADM-009; got: %v", gotErr)
		}
		// Verify "fresh-name-B" was NOT created. We use list-keys on a non-existent
		// SVTN to infer absence. If Create had run, listing keys on "fresh-name-B"
		// would succeed (returning an empty slice); it must return E-SVTN-003 instead.
		var listFn func(ctx context.Context, args json.RawMessage) (any, error)
		for _, h := range handlers {
			if h.Command == "admin.key.list-keys" {
				listFn = h.Fn
				break
			}
		}
		if listFn != nil {
			// Use a bootstrap context for the Lookup so auth passes.
			// We don't have bootstrapPub here — use a new manager check instead.
			// Actually, list-keys does not call resolveAndVerifyCallerRole, so any
			// caller pubkey in ctx is irrelevant for auth on list-keys handler.
			listArgs, _ := json.Marshal(adminListKeysArgs{SVTNName: "fresh-name-B"})
			_, listErr := listFn(ctx, json.RawMessage(listArgs))
			if listErr == nil {
				t.Error("Create-not-called assertion: 'fresh-name-B' SVTN should not exist; list-keys returned success (Create was called)")
			} else if !strings.Contains(listErr.Error(), "E-SVTN-003") {
				// Any error other than E-SVTN-003 is a genuine verification failure:
				// the absence oracle depends on E-SVTN-003 being returned for an
				// absent SVTN. An unexpected error code means the test cannot prove
				// Create was not called (F-P3L2-04).
				t.Fatalf("list-keys on absent 'fresh-name-B' returned unexpected error: %v (E-SVTN-003 expected for absent SVTN)", listErr)
			}
		}
	})
}
