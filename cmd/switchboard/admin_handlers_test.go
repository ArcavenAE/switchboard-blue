package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	m := newTestSVTNManager(t)
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

	_, handlerErr := expireFn(context.Background(), json.RawMessage(args))
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
	m := newTestSVTNManager(t)
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

			_, handlerErr := expireFn(context.Background(), json.RawMessage(args))
			if handlerErr == nil {
				t.Fatalf("expected E-CFG-001 for after=%q, got nil", tc.after)
			}
			if !strings.Contains(handlerErr.Error(), "E-CFG-001") {
				t.Errorf("expected E-CFG-001 for after=%q, got: %v", tc.after, handlerErr)
			}
		})
	}
}

// TestBuildAdminHandlers_FourHandlers asserts that BuildAdminHandlers returns
// exactly four handlers with the correct command names.
// Traces to AC-001; BC-2.05.004 PC-1..PC-3.
func TestBuildAdminHandlers_FourHandlers(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m, nil)

	want := map[string]bool{
		"admin.key.register":  false,
		"admin.key.revoke":    false,
		"admin.key.expire":    false,
		"admin.key.list-keys": false,
	}
	for _, h := range handlers {
		want[h.Command] = true
	}
	for cmd, found := range want {
		if !found {
			t.Errorf("expected handler %q not found in BuildAdminHandlers result", cmd)
		}
	}
	if len(handlers) != 4 {
		t.Errorf("expected 4 handlers, got %d", len(handlers))
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
// Traces to F-009 (go.md rule 4: %w wrapping).
func TestMapAdminError_ErrorWrapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		sentinel    error
		svtn        string
		pubkey      string
		claimedRole string
		wantCode    string
		wantDetail  string // non-empty: additional substring that must appear in the error
	}{
		{"ErrSVTNNotFound", svtnmgmt.ErrSVTNNotFound, "s", "k", "", "E-SVTN-003", ""},
		{"ErrKeyNotRegistered", admission.ErrKeyNotRegistered, "s", "k", "", "E-ADM-013", ""},
		{"ErrRoleMismatch", svtnmgmt.ErrRoleMismatch, "s", "k", "control", "E-ADM-019", ""},
		{"ErrControlRevocationRequiresConfirm", svtnmgmt.ErrControlRevocationRequiresConfirm, "s", "k", "", "E-ADM-018", ""},
		// ErrInvalidDuration is intentionally absent: mapAdminError does not handle it
		// because the handler-side ttl guards already produce E-CFG-001 with proper
		// detail before SVTNManager.ExpireKey is called. See mapAdminError doc.
		{"ErrBootstrapKeyRevokeForbidden", svtnmgmt.ErrBootstrapKeyRevokeForbidden, "s", "k", "", "E-ADM-020", "cannot revoke the last bootstrap key in SVTN s"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := mapAdminError(tc.sentinel, tc.svtn, tc.pubkey, tc.claimedRole)
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
//   - expired_key_non_control_rejected: a key with an expiry set and a non-control
//     role is still in the SVTN registry (expiry is enforced at admission, not at
//     role lookup); role check fails → E-ADM-009.
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

	t.Run("expired_key_non_control_rejected", func(t *testing.T) {
		t.Parallel()
		m, _ := newManagerWithSVTN(t)
		// Register an access-role key and set a short TTL in the future. The key
		// is still active (not yet expired); CallerKeyRoleActive returns RoleAccess.
		// verifyCallerRole rejects non-control role → E-ADM-009.
		keyPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		if _, err := m.RegisterKey(svtnName, keyPub, admission.RoleAccess); err != nil {
			t.Fatalf("register key: %v", err)
		}
		// Set a 1-hour TTL — key is still active (not expired yet).
		if _, err := m.ExpireKey(svtnName, keyPub, time.Hour); err != nil {
			t.Fatalf("expire key: %v", err)
		}
		// CallerKeyRoleActive returns RoleAccess (not expired). verifyCallerRole
		// rejects non-control role → E-ADM-009.
		ctx := mgmt.WithCallerPubkey(context.Background(), keyPub)
		gotErr := resolveAndVerifyCallerRole(ctx, m, nil, svtnName, "", cmd)
		if gotErr == nil {
			t.Fatal("expired non-control key: expected E-ADM-009, got nil")
		}
		if !strings.Contains(gotErr.Error(), "E-ADM-009") {
			t.Errorf("expired non-control key: expected E-ADM-009 in error, got: %v", gotErr)
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
