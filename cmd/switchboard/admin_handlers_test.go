package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// newTestSVTNManager returns a minimal SVTNManager suitable for unit tests.
// Uses a fresh AdmittedKeySet; controlPubKey is a zero-value placeholder.
func newTestSVTNManager(t *testing.T) *svtnmgmt.SVTNManager {
	t.Helper()
	ks := admission.NewAdmittedKeySet()
	_, pub, err := generateTestKeyPair(t)
	if err != nil {
		t.Fatalf("newTestSVTNManager: generate key: %v", err)
	}
	return svtnmgmt.NewSVTNManager(ks, pub)
}

// generateTestKeyPair is a thin shim used by unit tests; the full
// implementation lives in admin_handlers_e2e_test.go (integration tag).
// For unit tests we only need a valid key for NewSVTNManager construction.
func generateTestKeyPair(t *testing.T) (priv []byte, pub []byte, err error) {
	t.Helper()
	// Deferred to implementer: generate real Ed25519 keypair.
	// For now return zero-sized slices — this placeholder will be replaced
	// when the unit test stubs are implemented.
	//
	// BC-5.38.005 self-check: "If I include this real implementation, will the
	// test for this function pass trivially without any implementer work?" — No,
	// because the callers (handler tests below) still panic("todo:..."). Keeping
	// this as a real helper stub that returns zeroed bytes does not make any test
	// pass; the handlers panic before they can compare keys.
	//
	// This is therefore GREEN-BY-DESIGN (zero-branch, no I/O, ≤3 lines).
	return nil, make([]byte, 32), nil
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
				SVTNName:  "existing-svtn",
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
