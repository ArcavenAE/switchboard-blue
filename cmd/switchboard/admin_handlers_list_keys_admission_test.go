// admin_handlers_list_keys_admission_test.go — RED tests for F-P5P13-A-001 [HIGH].
//
// This file contains admission-check tests for admin.key.list-keys.
// The spec anchor is interface-definitions.md v1.24 §111 (F-L2-003 ruling:
// "any admitted role OR operator-set member may call admin.key.list-keys").
//
// Current handler (makeListKeysHandler, admin_handlers.go:348-385) does NOT call
// resolveAndVerifyCallerRole at all — it accepts any caller without any admission
// check.  These tests are RED because:
//
//   - Cases 4, 6, 9 expect E-ADM-009 (fail-closed) for callers that have no
//     active, admitted role in the target SVTN and are not in the operator set and
//     are not the bootstrap key.  The current handler returns the key list to them.
//
//   - Case 6 (cross-SVTN enumeration) is the security regression guard: a caller
//     admitted to SVTN-A must NOT be able to enumerate SVTN-B's key roster.  The
//     current handler will happily return SVTN-B's roster to any caller — RED.
//
//   - Cases 1, 2, 3, 5, 7, 8 are GREEN against the current code (handler returns
//     ok), but they are included to give per-arm signal so a future mutation of the
//     admission gate cannot trivially pass a subset while breaking a sibling case.
//
// RED discipline: every test in cases 4, 6, 9 MUST fail against develop tip today.
// Cases 1, 2, 3, 5, 7, 8 may pass today; they are included as regression guards
// to be confirmed GREEN in the same `go test` run.
//
// Spec authority: interface-definitions.md v1.24 §111, F-L2-003.
// Finding: F-P5P13-A-001 [HIGH], CWE-862 (Missing Authorization).
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// newListKeysAdmissionManager builds a SVTNManager suitable for the admission
// test matrix.  It creates "svtn-a" and "svtn-b" as two distinct SVTNs.
// Each call to newListKeysAdmissionManager produces a fresh manager with its
// own bootstrap key so tests can run in parallel without sharing state.
//
// Returns:
//   - m: the SVTNManager
//   - bootstrapPub: daemon bootstrap key (satisfies m.IsBootstrapKey; always allowed)
//   - bootstrapPriv: corresponding private key (unused here but kept for symmetry)
func newListKeysAdmissionManager(t *testing.T) (m *svtnmgmt.SVTNManager, bootstrapPub ed25519.PublicKey) {
	t.Helper()

	bsPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("newListKeysAdmissionManager: generate bootstrap key: %v", err)
	}
	ks := admission.NewAdmittedKeySet()
	m = svtnmgmt.NewSVTNManager(ks, bsPub)

	if _, err := m.Create("svtn-a"); err != nil {
		t.Fatalf("newListKeysAdmissionManager: create svtn-a: %v", err)
	}
	if _, err := m.Create("svtn-b"); err != nil {
		t.Fatalf("newListKeysAdmissionManager: create svtn-b: %v", err)
	}
	return m, bsPub
}

// extractListKeysFn walks handlers returned by BuildAdminHandlers and returns
// the admin.key.list-keys handler function, or fails the test if absent.
func extractListKeysFn(t *testing.T, m *svtnmgmt.SVTNManager) func(ctx context.Context, args json.RawMessage) (any, error) {
	t.Helper()
	handlers := BuildAdminHandlers(m, nil, nil, nil)
	for _, h := range handlers {
		if h.Command == "admin.key.list-keys" {
			return h.Fn
		}
	}
	t.Fatal("admin.key.list-keys handler not found in BuildAdminHandlers result")
	return nil
}

// marshalListKeysArgs encodes adminListKeysArgs to JSON for test call sites.
func marshalListKeysArgs(t *testing.T, svtnName string) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(adminListKeysArgs{SVTNName: svtnName})
	if err != nil {
		t.Fatalf("marshalListKeysArgs: %v", err)
	}
	return json.RawMessage(b)
}

// TestListKeys_AdmittedControlRole_Allowed verifies that a caller with an
// active control role in the target SVTN receives the key list (case 1).
//
// Spec anchor: F-L2-003 "any admitted role may call admin.key.list-keys".
// RED status: GREEN (current handler admits all callers — this case passes).
func TestListKeys_AdmittedControlRole_Allowed(t *testing.T) {
	t.Parallel()

	m, _ := newListKeysAdmissionManager(t)

	// Register a control-role caller key in svtn-a.
	callerPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller key: %v", err)
	}
	if _, err := m.RegisterKey("svtn-a", callerPub, admission.RoleControl); err != nil {
		t.Fatalf("register control key: %v", err)
	}

	listFn := extractListKeysFn(t, m)
	ctx := mgmt.WithCallerPubkey(context.Background(), callerPub)
	result, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-a"))
	if handlerErr != nil {
		t.Fatalf("F-P5P13-A-001 case 1 (control role): expected success; got: %v", handlerErr)
	}
	listResult, ok := result.(adminListKeysResult)
	if !ok {
		t.Fatalf("F-P5P13-A-001 case 1: expected adminListKeysResult; got %T", result)
	}
	if listResult.Keys == nil {
		t.Error("F-P5P13-A-001 case 1: Keys must not be nil (EC-003)")
	}
}

// TestListKeys_AdmittedConsoleRole_Allowed verifies that a caller with an
// active console role in the target SVTN receives the key list (case 2).
//
// Spec anchor: F-L2-003 "any admitted role may call admin.key.list-keys".
// RED status: GREEN (current handler admits all callers — this case passes).
func TestListKeys_AdmittedConsoleRole_Allowed(t *testing.T) {
	t.Parallel()

	m, _ := newListKeysAdmissionManager(t)

	callerPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller key: %v", err)
	}
	if _, err := m.RegisterKey("svtn-a", callerPub, admission.RoleConsole); err != nil {
		t.Fatalf("register console key: %v", err)
	}

	listFn := extractListKeysFn(t, m)
	ctx := mgmt.WithCallerPubkey(context.Background(), callerPub)
	result, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-a"))
	if handlerErr != nil {
		t.Fatalf("F-P5P13-A-001 case 2 (console role): expected success; got: %v", handlerErr)
	}
	listResult, ok := result.(adminListKeysResult)
	if !ok {
		t.Fatalf("F-P5P13-A-001 case 2: expected adminListKeysResult; got %T", result)
	}
	if listResult.Keys == nil {
		t.Error("F-P5P13-A-001 case 2: Keys must not be nil (EC-003)")
	}
}

// TestListKeys_AdmittedAccessRole_Allowed verifies that a caller with an
// active access role in the target SVTN receives the key list (case 3).
//
// Spec anchor: F-L2-003 "any admitted role may call admin.key.list-keys".
// RED status: GREEN (current handler admits all callers — this case passes).
func TestListKeys_AdmittedAccessRole_Allowed(t *testing.T) {
	t.Parallel()

	m, _ := newListKeysAdmissionManager(t)

	callerPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller key: %v", err)
	}
	if _, err := m.RegisterKey("svtn-a", callerPub, admission.RoleAccess); err != nil {
		t.Fatalf("register access key: %v", err)
	}

	listFn := extractListKeysFn(t, m)
	ctx := mgmt.WithCallerPubkey(context.Background(), callerPub)
	result, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-a"))
	if handlerErr != nil {
		t.Fatalf("F-P5P13-A-001 case 3 (access role): expected success; got: %v", handlerErr)
	}
	listResult, ok := result.(adminListKeysResult)
	if !ok {
		t.Fatalf("F-P5P13-A-001 case 3: expected adminListKeysResult; got %T", result)
	}
	if listResult.Keys == nil {
		t.Error("F-P5P13-A-001 case 3: Keys must not be nil (EC-003)")
	}
}

// TestListKeys_RevokedExpiredRole_DeniedEADM009 verifies that a caller whose
// key was registered and then revoked/expired is denied with E-ADM-009 (case 4).
//
// Spec anchor: F-P5L1-001 fail-closed: registered-any-state but inactive keys
// must not receive any authority.  admin.key.list-keys is not exempt.
//
// RED (F-P5P13-A-001): the current handler has no admission check — it returns
// the key list to a revoked caller.  MUST FAIL at develop tip.
func TestListKeys_RevokedExpiredRole_DeniedEADM009(t *testing.T) {
	t.Parallel()

	t.Run("revoked_key_denied", func(t *testing.T) {
		t.Parallel()

		m, _ := newListKeysAdmissionManager(t)

		// Revoke without confirm (console-role revoke; control requires confirm=true).
		// Use a console key so we can revoke it without the confirm gate.
		consolePub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate console key: %v", err)
		}
		if _, err := m.RegisterKey("svtn-a", consolePub, admission.RoleConsole); err != nil {
			t.Fatalf("register console key: %v", err)
		}
		if _, err := m.RevokeKey("svtn-a", consolePub, admission.RoleConsole, false); err != nil {
			t.Fatalf("revoke console key: %v", err)
		}

		listFn := extractListKeysFn(t, m)
		// Caller is registered (in any-state) but revoked — IsRegisteredAnyState true,
		// CallerKeyRoleActive returns (0, false).  Must be denied E-ADM-009.
		ctx := mgmt.WithCallerPubkey(context.Background(), consolePub)
		_, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-a"))
		// RED: this assertion MUST FAIL at develop tip (handler currently returns ok).
		if handlerErr == nil {
			t.Fatal("F-P5P13-A-001 case 4 (revoked key): expected E-ADM-009; got nil — " +
				"RED: admission check missing from makeListKeysHandler")
		}
		if !strings.Contains(handlerErr.Error(), "E-ADM-009") {
			t.Errorf("F-P5P13-A-001 case 4: expected E-ADM-009 in error; got: %v", handlerErr)
		}
	})

	t.Run("expired_key_denied", func(t *testing.T) {
		t.Parallel()

		m, _ := newListKeysAdmissionManager(t)

		// Register a control key, expire it immediately (1ns TTL), wait for expiry.
		callerPub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		if _, err := m.RegisterKey("svtn-a", callerPub, admission.RoleControl); err != nil {
			t.Fatalf("register key: %v", err)
		}
		if _, err := m.ExpireKey("svtn-a", callerPub, time.Nanosecond); err != nil {
			t.Fatalf("expire key: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // ensure now > expiry

		listFn := extractListKeysFn(t, m)
		ctx := mgmt.WithCallerPubkey(context.Background(), callerPub)
		_, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-a"))
		// RED: this assertion MUST FAIL at develop tip.
		if handlerErr == nil {
			t.Fatal("F-P5P13-A-001 case 4 (expired key): expected E-ADM-009; got nil — " +
				"RED: admission check missing from makeListKeysHandler")
		}
		if !strings.Contains(handlerErr.Error(), "E-ADM-009") {
			t.Errorf("F-P5P13-A-001 case 4 (expired): expected E-ADM-009; got: %v", handlerErr)
		}
	})
}

// TestListKeys_OperatorSetMember_AllowedUnconditionally verifies that a caller
// in the OperatorKeySet may enumerate any SVTN even without being admitted to it
// (case 5 — F-L2-003 "or operator-set member" unconditionally).
//
// Spec anchor: F-L2-003 operator-set carve-out.
// RED status: GREEN (current handler admits all callers — this case passes).
func TestListKeys_OperatorSetMember_AllowedUnconditionally(t *testing.T) {
	t.Parallel()

	m, _ := newListKeysAdmissionManager(t)

	// Generate an operator key — NOT registered in svtn-a.
	operatorPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate operator key: %v", err)
	}
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})
	handlers := BuildAdminHandlers(m, ops, nil, nil)
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

	ctx := mgmt.WithCallerPubkey(context.Background(), operatorPub)
	result, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-a"))
	if handlerErr != nil {
		t.Fatalf("F-P5P13-A-001 case 5 (operator-set member): expected success; got: %v", handlerErr)
	}
	if _, ok := result.(adminListKeysResult); !ok {
		t.Fatalf("F-P5P13-A-001 case 5: expected adminListKeysResult; got %T", result)
	}
}

// TestListKeys_OperatorSetMember_MissingSVTN_ReturnsESVTN003 verifies that an
// operator-set caller requesting a nonexistent SVTN receives E-SVTN-003, NOT
// E-ADM-009.  This is the operator-set × missing-SVTN diagonal: the SVTN-existence
// check must run regardless of whether the caller cleared the admission gate.
//
// If a mutation reorders the admission gate before the SVTN-existence check for
// operator-set callers, the caller (who passes admission) would receive E-SVTN-003
// from the list call.  If the SVTN-existence check were skipped entirely for
// admitted callers, it would return success (or a different error).  This test
// closes that detection gap.
//
// Spec anchor: F-L2-003 operator-set carve-out; mapAdminError ErrSVTNNotFound → E-SVTN-003.
// RED status: GREEN (current handler returns E-SVTN-003 for any missing SVTN).
func TestListKeys_OperatorSetMember_MissingSVTN_ReturnsESVTN003(t *testing.T) {
	t.Parallel()

	m, _ := newListKeysAdmissionManager(t)

	// Register an operator key — NOT admitted to any SVTN.
	operatorPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate operator key: %v", err)
	}
	ops := mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})
	handlers := BuildAdminHandlers(m, ops, nil, nil)
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

	ctx := mgmt.WithCallerPubkey(context.Background(), operatorPub)
	_, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-does-not-exist"))
	if handlerErr == nil {
		t.Fatal("F-P5P14-B-005: operator-set caller + missing SVTN: expected E-SVTN-003; got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-SVTN-003") {
		t.Errorf("F-P5P14-B-005: expected E-SVTN-003 in error; got: %v", handlerErr)
	}
}

// TestListKeys_CrossSVTNEnumeration_DeniedEADM009 verifies that a caller admitted
// to SVTN-A cannot enumerate SVTN-B's key roster (case 6 — cross-SVTN security
// regression guard, CWE-862).
//
// Setup:
//   - caller is admitted to svtn-a with control role
//   - request targets svtn-b (different SVTN)
//
// Expected: E-ADM-009 (caller is not admitted to svtn-b, and is not the bootstrap
// key, and is not in the operator set).
//
// RED (F-P5P13-A-001 [HIGH]): the current handler has no admission check — it calls
// m.ListKeys(a.SVTNName) without verifying the caller's role in THAT SVTN.
// A caller admitted to svtn-a can today freely enumerate svtn-b.
// This MUST FAIL at develop tip — this is the primary security regression guard.
func TestListKeys_CrossSVTNEnumeration_DeniedEADM009(t *testing.T) {
	t.Parallel()

	m, _ := newListKeysAdmissionManager(t)

	// Register caller only in svtn-a, NOT in svtn-b.
	callerPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate caller key: %v", err)
	}
	if _, err := m.RegisterKey("svtn-a", callerPub, admission.RoleControl); err != nil {
		t.Fatalf("register caller in svtn-a: %v", err)
	}
	// Seed svtn-b with a separate key so it has content (not just the bootstrap key).
	svtnBKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate svtn-b key: %v", err)
	}
	if _, err := m.RegisterKey("svtn-b", svtnBKey, admission.RoleConsole); err != nil {
		t.Fatalf("register svtn-b key: %v", err)
	}

	listFn := extractListKeysFn(t, m)
	// Caller is admitted to svtn-a but requests svtn-b — must be denied.
	ctx := mgmt.WithCallerPubkey(context.Background(), callerPub)
	_, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-b"))
	// RED: MUST FAIL at develop tip — current handler has no SVTN-scoped admission gate.
	if handlerErr == nil {
		t.Fatal("F-P5P13-A-001 case 6 (cross-SVTN enumeration): expected E-ADM-009; got nil — " +
			"RED SECURITY: makeListKeysHandler has no admission gate; " +
			"caller admitted to svtn-a can enumerate svtn-b roster (CWE-862)")
	}
	if !strings.Contains(handlerErr.Error(), "E-ADM-009") {
		t.Errorf("F-P5P13-A-001 case 6: expected E-ADM-009 in error; got: %v", handlerErr)
	}
}

// TestListKeys_BootstrapKey_Allowed verifies that the daemon bootstrap key may
// enumerate any SVTN it created (case 7).
//
// Spec anchor: bootstrap key is the unconditional trust anchor (BC-2.05.004 PC-1).
// RED status: GREEN (current handler admits all callers — this case passes).
func TestListKeys_BootstrapKey_Allowed(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newListKeysAdmissionManager(t)

	listFn := extractListKeysFn(t, m)
	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	result, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-a"))
	if handlerErr != nil {
		t.Fatalf("F-P5P13-A-001 case 7 (bootstrap key): expected success; got: %v", handlerErr)
	}
	if _, ok := result.(adminListKeysResult); !ok {
		t.Fatalf("F-P5P13-A-001 case 7: expected adminListKeysResult; got %T", result)
	}
}

// TestListKeys_TargetSVTNNotFound_ReturnsESVTN003 verifies that requesting
// list-keys for a nonexistent SVTN returns E-SVTN-003 (case 8 — preserves
// existing behavior, error mapping correctness).
//
// Spec anchor: mapAdminError ErrSVTNNotFound arm → E-SVTN-003.
// RED status: GREEN (current handler returns E-SVTN-003 for missing SVTNs).
func TestListKeys_TargetSVTNNotFound_ReturnsESVTN003(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newListKeysAdmissionManager(t)

	listFn := extractListKeysFn(t, m)
	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)
	_, handlerErr := listFn(ctx, marshalListKeysArgs(t, "svtn-does-not-exist"))
	if handlerErr == nil {
		t.Fatal("F-P5P13-A-001 case 8: expected E-SVTN-003 for nonexistent SVTN; got nil")
	}
	if !strings.Contains(handlerErr.Error(), "E-SVTN-003") {
		t.Errorf("F-P5P13-A-001 case 8: expected E-SVTN-003 in error; got: %v", handlerErr)
	}
}

// TestListKeys_NoCaller_DeniedEADM009 verifies that a request with no CallerPubkey
// in ctx AND no CallerRole in args is denied with E-ADM-009 (case 9 — defensive
// fail-closed; BC-2.05.004 Precondition 1 / DI-001).
//
// Spec anchor: resolveAndVerifyCallerRole fail-closed path when both server-resolved
// pubkey and fallback callerRoleStr are absent.
//
// RED (F-P5P13-A-001): the current handler has no admission check — it accepts
// requests with no caller identity at all.  MUST FAIL at develop tip.
func TestListKeys_NoCaller_DeniedEADM009(t *testing.T) {
	t.Parallel()

	m, _ := newListKeysAdmissionManager(t)

	listFn := extractListKeysFn(t, m)
	// context.Background() has no CallerPubkey; adminListKeysArgs has no CallerRole.
	// The handler must fail closed.
	_, handlerErr := listFn(context.Background(), marshalListKeysArgs(t, "svtn-a"))
	// RED: MUST FAIL at develop tip — current handler has no admission check.
	if handlerErr == nil {
		t.Fatal("F-P5P13-A-001 case 9 (no caller identity): expected E-ADM-009; got nil — " +
			"RED: makeListKeysHandler must fail-closed when no CallerPubkey and no CallerRole")
	}
	if !strings.Contains(handlerErr.Error(), "E-ADM-009") {
		t.Errorf("F-P5P13-A-001 case 9: expected E-ADM-009 in error; got: %v", handlerErr)
	}
}
