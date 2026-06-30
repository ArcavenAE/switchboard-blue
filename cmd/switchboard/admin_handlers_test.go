package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// newTestSVTNManager returns a SVTNManager pre-populated with the SVTNs that
// happy-path and error-mapping tests reference.
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
func newTestSVTNManager(t *testing.T) *svtnmgmt.SVTNManager {
	t.Helper()

	// Generate a random bootstrap key distinct from the canonical test key (32
	// zero bytes). This ensures Create() does not pre-register the zero key on
	// SVTNs where it must be absent (E-ADM-013 / EC-003 cases).
	bootstrapPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("newTestSVTNManager: generate bootstrap key: %v", err)
	}

	ks := admission.NewAdmittedKeySet()
	m := svtnmgmt.NewSVTNManager(ks, bootstrapPub)

	// "test-svtn": create and register the canonical zero key as control so that
	// happy-path revoke / expire / list-keys tests can act on it.
	if _, err := m.Create("test-svtn"); err != nil {
		t.Fatalf("newTestSVTNManager: create test-svtn: %v", err)
	}
	zeroKey := make([]byte, ed25519.PublicKeySize)
	if _, err := m.RegisterKey("test-svtn", zeroKey, admission.RoleControl); err != nil {
		t.Fatalf("newTestSVTNManager: register zero key on test-svtn: %v", err)
	}

	// "existing-svtn": create with no additional keys. The E-ADM-013 subtest
	// expects the canonical zero key to be absent so that revocation returns
	// "key not registered" rather than "SVTN not found".
	if _, err := m.Create("existing-svtn"); err != nil {
		t.Fatalf("newTestSVTNManager: create existing-svtn: %v", err)
	}

	// "empty-svtn": create with no additional keys. Required by
	// TestBuildAdminHandlers_ListKeys_EmptySliceNotNil (EC-003).
	if _, err := m.Create("empty-svtn"); err != nil {
		t.Fatalf("newTestSVTNManager: create empty-svtn: %v", err)
	}

	// "rolematch-svtn": create and register the canonical zero key as ROLE_CONTROL.
	// Required by KeyRevoke_ErrorMapping/role_mismatch_yields_E-ADM-019, which
	// revokes with ROLE_BOOTSTRAP to trigger ErrRoleMismatch. Using a separate
	// SVTN from "existing-svtn" avoids conflict with the sibling subtest that
	// requires the zero key to be absent.
	if _, err := m.Create("rolematch-svtn"); err != nil {
		t.Fatalf("newTestSVTNManager: create rolematch-svtn: %v", err)
	}
	if _, err := m.RegisterKey("rolematch-svtn", zeroKey, admission.RoleControl); err != nil {
		t.Fatalf("newTestSVTNManager: register zero key on rolematch-svtn: %v", err)
	}

	// "nonexistent-svtn" is intentionally not created; KeyRegister_ErrorMapping
	// expects E-SVTN-003 for that name.
	return m
}

// TestBuildAdminHandlers_KeyRegister_HappyPath asserts that the
// admin.key.register handler returns ok=true for a valid registration request.
// Traces to AC-001; BC-2.05.004 PC-1.
func TestBuildAdminHandlers_KeyRegister_HappyPath(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

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

	// This call will panic("todo:...") until the handler is implemented.
	// The test is red by design (BC-5.38.001 Red Gate).
	_, handlerErr := registerFn(context.Background(), json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error from register handler: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_KeyRevoke_HappyPath asserts that the admin.key.revoke
// handler returns ok=true when role matches and key exists.
// Traces to AC-001; BC-2.05.004 PC-2.
func TestBuildAdminHandlers_KeyRevoke_HappyPath(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

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

	_, handlerErr := revokeFn(context.Background(), json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error from revoke handler: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_KeyExpire_HappyPath asserts that admin.key.expire
// returns ok=true for a valid TTL.
// Traces to AC-001; BC-2.05.004 PC-3.
func TestBuildAdminHandlers_KeyExpire_HappyPath(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

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

	_, handlerErr := expireFn(context.Background(), json.RawMessage(args))
	if handlerErr != nil {
		t.Fatalf("unexpected error from expire handler: %v", handlerErr)
	}
}

// TestBuildAdminHandlers_ListKeys_HappyPath asserts that admin.list-keys
// returns ok=true and a non-nil keys array.
// Traces to AC-001; BC-2.05.004 PC-1.
func TestBuildAdminHandlers_ListKeys_HappyPath(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

	var listFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.list-keys" {
			listFn = h.Fn
			break
		}
	}
	if listFn == nil {
		t.Fatal("admin.list-keys handler not found in BuildAdminHandlers result")
	}

	args, err := json.Marshal(adminListKeysArgs{SVTNName: "test-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, handlerErr := listFn(context.Background(), json.RawMessage(args))
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
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

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

	_, handlerErr := registerFn(context.Background(), json.RawMessage(args))
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
			m := newTestSVTNManager(t)
			handlers := BuildAdminHandlers(m)

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

			_, handlerErr := revokeFn(context.Background(), json.RawMessage(rawArgs))
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
	BuildAdminHandlers(nil)
}

// TestBuildAdminHandlers_KeyRegister_MalformedJSON asserts that
// admin.key.register returns E-CFG-001 when the args JSON is malformed.
// Traces to AC-001 edge case EC-001; BC-2.05.004 PC-1 precondition (well-formed request).
func TestBuildAdminHandlers_KeyRegister_MalformedJSON(t *testing.T) {
	t.Parallel()
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

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
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

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

	_, handlerErr := revokeFn(context.Background(), json.RawMessage(args))
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
	handlers := BuildAdminHandlers(m)

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
	handlers := BuildAdminHandlers(m)

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
	handlers := BuildAdminHandlers(m)

	want := map[string]bool{
		"admin.key.register": false,
		"admin.key.revoke":   false,
		"admin.key.expire":   false,
		"admin.list-keys":    false,
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
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

	var listFn func(ctx context.Context, args json.RawMessage) (any, error)
	for _, h := range handlers {
		if h.Command == "admin.list-keys" {
			listFn = h.Fn
			break
		}
	}
	if listFn == nil {
		t.Fatal("admin.list-keys handler not found")
	}

	// Request list-keys for an SVTN with no registered keys.
	args, err := json.Marshal(adminListKeysArgs{SVTNName: "empty-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, handlerErr := listFn(context.Background(), json.RawMessage(args))
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

// TestBuildAdminHandlers_KeyRevoke_ControlRequiresConfirm asserts that
// admin.key.revoke for a control key without confirm=true returns E-ADM-018.
// Traces to AC-002; BC-2.05.004 PC-2; ADR-004.
func TestBuildAdminHandlers_KeyRevoke_ControlRequiresConfirm(t *testing.T) {
	t.Parallel()

	// Use newTestSVTNManager so "test-svtn" exists with the zero key registered
	// as ROLE_CONTROL. Without this, the handler hits E-SVTN-003 (SVTN not
	// found) before it can reach the ErrControlRevocationRequiresConfirm check.
	m := newTestSVTNManager(t)
	handlers := BuildAdminHandlers(m)

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

	_, handlerErr := revokeFn(context.Background(), json.RawMessage(args))
	if handlerErr == nil {
		t.Fatal("expected E-ADM-018 for control revocation without confirm, got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-ADM-018") {
		t.Errorf("expected E-ADM-018, got: %v", handlerErr)
	}
}
