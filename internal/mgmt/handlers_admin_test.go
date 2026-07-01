// handlers_admin_test.go — unit tests for the admin.svtn.create handler seam
// in internal/mgmt (MakeAdminSVTNCreateHandler).
//
// These tests exercise the handler builder at the mgmt package boundary — they
// do NOT import internal/svtnmgmt (ARCH-12 §Package DAG Constraints). The
// concrete SVTNCreator is a test double. Full integration tests (binding the real
// *svtnmgmt.SVTNManager) live in cmd/switchboard/admin_handlers_test.go.
//
// Traceability:
//
//	BC-2.07.001 PC-1  — admin.svtn.create handler is registered and dispatches
//	BC-2.07.001 Inv-3 — non-control-role caller receives E-ADM-009
//	BC-2.07.001 EC-001 — duplicate SVTN name propagates SVTN-exists error
//	AC-001  — BuildAdminHandlers registers admin.svtn.create
//	AC-003  — non-control-role caller → E-ADM-009
//	AC-005  — duplicate name → SVTN-exists error
//	AC-006  — control-role caller succeeds; non-control fails; duplicate fails
//
// Red Gate: all tests below MUST fail before MakeAdminSVTNCreateHandler is
// implemented (BC-5.38.001).
package mgmt

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// fakeSVTNCreator is a test double for SVTNCreator.
type fakeSVTNCreator struct {
	// result is returned on Create if err is nil.
	result SVTNCreateResult
	// err is returned on Create if non-nil.
	err error
	// called records whether Create was called.
	called bool
	// lastName records the last svtnName passed to Create.
	lastName string
}

func (f *fakeSVTNCreator) Create(svtnName string) (SVTNCreateResult, error) {
	f.called = true
	f.lastName = svtnName
	return f.result, f.err
}

// errSVTNAlreadyExists is a test sentinel for the duplicate-name error path.
// In production, svtnmgmt.ErrSVTNAlreadyExists is passed here.
var errSVTNAlreadyExists = errors.New("SVTN already exists")

// TestMakeAdminSVTNCreateHandler_ControlCallerSucceeds verifies AC-006 (first
// sub-case): a handler built via MakeAdminSVTNCreateHandler dispatches to the
// SVTNCreator when the roleChecker approves the caller.
//
// BC-2.07.001 PC-1 — handler dispatches to creator on control-role approval.
// AC-001 — handler is invokable after BuildAdminHandlers registration.
// AC-006 sub-case: control-role caller succeeds.
func TestMakeAdminSVTNCreateHandler_ControlCallerSucceeds(t *testing.T) {
	t.Parallel()

	// AC-006 / BC-2.07.001 PC-1 — control-role caller must succeed.
	// MakeAdminSVTNCreateHandler currently panics → RED.
	creator := &fakeSVTNCreator{
		result: SVTNCreateResult{
			SVTNID:               "deadbeefdeadbeef",
			BootstrapFingerprint: "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		},
	}
	approveAll := func(_ context.Context, _ string) error { return nil }

	handlerFn := MakeAdminSVTNCreateHandler(creator, approveAll)

	args, err := json.Marshal(adminSVTNCreateArgs{Name: "new-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, err := handlerFn(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("AC-006 control-caller: expected success; got error: %v", err)
	}
	if result == nil {
		t.Fatal("AC-006 control-caller: expected non-nil result data")
	}

	// Verify creator was called with the right name.
	if !creator.called {
		t.Error("AC-006 control-caller: creator.Create was not called")
	}
	if creator.lastName != "new-svtn" {
		t.Errorf("AC-006 control-caller: creator.Create called with %q; want %q", creator.lastName, "new-svtn")
	}

	// AC-004: verify response shape (svtn_id and bootstrap_fingerprint present).
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var wire adminSVTNCreateResponse
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatalf("unmarshal result to adminSVTNCreateResponse: %v", err)
	}
	if wire.SVTNID == "" {
		t.Error("AC-004: svtn_id field is empty in response")
	}
	if wire.BootstrapFingerprint == "" {
		t.Error("AC-004: bootstrap_fingerprint field is empty in response")
	}
}

// TestMakeAdminSVTNCreateHandler_NonControlCallerDenied verifies AC-006 (second
// sub-case) and AC-003: when the roleChecker returns an error (non-control-role
// caller), the handler returns E-ADM-009 and does NOT call creator.Create.
//
// BC-2.07.001 Inv-3 — authority check fires before dispatch.
// AC-003 — non-control-role caller → E-ADM-009.
// AC-006 sub-case: non-control-role caller receives E-ADM-009.
func TestMakeAdminSVTNCreateHandler_NonControlCallerDenied(t *testing.T) {
	t.Parallel()

	// AC-003 / BC-2.07.001 Inv-3 — non-control caller must receive E-ADM-009.
	// MakeAdminSVTNCreateHandler currently panics → RED.
	creator := &fakeSVTNCreator{}
	denyAll := func(_ context.Context, _ string) error {
		return errors.New("E-ADM-009: insufficient authority for operation admin.svtn.create: key SHA256:test= has role console")
	}

	handlerFn := MakeAdminSVTNCreateHandler(creator, denyAll)

	args, err := json.Marshal(adminSVTNCreateArgs{Name: "new-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, err = handlerFn(context.Background(), json.RawMessage(args))

	// AC-003: must return non-nil error containing E-ADM-009.
	if err == nil {
		t.Fatal("AC-003 non-control-caller: expected E-ADM-009 error; got nil")
	}
	if !containsString(err.Error(), "E-ADM-009") {
		t.Errorf("AC-003 non-control-caller: expected error to contain E-ADM-009; got %q", err.Error())
	}

	// BC-2.07.001 Inv-3: creator.Create must NOT be called.
	if creator.called {
		t.Error("BC-2.07.001 Inv-3: creator.Create was called despite non-control role; must not dispatch before auth check")
	}
}

// TestMakeAdminSVTNCreateHandler_DuplicateNameError verifies AC-006 (third
// sub-case) and AC-005: when the SVTNCreator returns ErrSVTNAlreadyExists,
// the handler propagates a SVTN-exists error to the response.
//
// BC-2.07.001 EC-001 — duplicate SVTN name returns SVTN-exists error.
// AC-005 — duplicate-name caller receives SVTN-exists error.
// AC-006 sub-case: duplicate-name caller receives SVTN-exists error.
func TestMakeAdminSVTNCreateHandler_DuplicateNameError(t *testing.T) {
	t.Parallel()

	// AC-005 / BC-2.07.001 EC-001 — duplicate SVTN name must propagate error.
	// MakeAdminSVTNCreateHandler currently panics → RED.
	creator := &fakeSVTNCreator{err: errSVTNAlreadyExists}
	approveAll := func(_ context.Context, _ string) error { return nil }

	handlerFn := MakeAdminSVTNCreateHandler(creator, approveAll)

	args, err := json.Marshal(adminSVTNCreateArgs{Name: "existing-svtn"})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, err = handlerFn(context.Background(), json.RawMessage(args))

	// AC-005: must return non-nil error containing SVTN-exists text.
	if err == nil {
		t.Fatal("AC-005 duplicate-name: expected SVTN-exists error; got nil")
	}
	if !containsString(err.Error(), "SVTN already exists") {
		t.Errorf("AC-005 duplicate-name: expected error to contain 'SVTN already exists'; got %q", err.Error())
	}
}

// TestMakeAdminSVTNCreateHandler_MissingNameReturnsError verifies that a
// missing or empty name field in the args returns E-CFG-001 (not a panic).
//
// BC-2.07.001 PC-1 — handler validates required args before dispatch.
func TestMakeAdminSVTNCreateHandler_MissingNameReturnsError(t *testing.T) {
	t.Parallel()

	// BC-2.07.001 PC-1 — missing name must return validation error.
	// MakeAdminSVTNCreateHandler currently panics → RED.
	creator := &fakeSVTNCreator{}
	approveAll := func(_ context.Context, _ string) error { return nil }

	handlerFn := MakeAdminSVTNCreateHandler(creator, approveAll)

	// Empty name field.
	args, err := json.Marshal(adminSVTNCreateArgs{Name: ""})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	_, err = handlerFn(context.Background(), json.RawMessage(args))
	if err == nil {
		t.Error("BC-2.07.001 PC-1: missing name: expected non-nil error; got nil")
	}

	// creator must not be called for an invalid request.
	if creator.called {
		t.Error("BC-2.07.001 PC-1: creator.Create was called despite missing name field")
	}
}

// TestAdminSVTNCreateResponse_JSONRoundTrip verifies that adminSVTNCreateResponse
// serialises with the correct JSON field names (svtn_id and bootstrap_fingerprint)
// as required by AC-004.
//
// BC-2.07.001 PC-1 + PC-2 — wire format for admin.svtn.create success response.
// AC-004 — response carries svtn_id (hex) and bootstrap_fingerprint (SHA256:<base64>).
func TestAdminSVTNCreateResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	// AC-004 / BC-2.07.001 PC-1+PC-2 — wire format field names.
	original := adminSVTNCreateResponse{
		SVTNID:               "aabbccddeeff0011aabbccddeeff0011",
		BootstrapFingerprint: "SHA256:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(adminSVTNCreateResponse): %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}

	// AC-004: svtn_id must be present.
	if _, ok := raw["svtn_id"]; !ok {
		t.Error("AC-004: adminSVTNCreateResponse: missing JSON field 'svtn_id'")
	}
	// AC-004: bootstrap_fingerprint must be present.
	if _, ok := raw["bootstrap_fingerprint"]; !ok {
		t.Error("AC-004: adminSVTNCreateResponse: missing JSON field 'bootstrap_fingerprint'")
	}

	var decoded adminSVTNCreateResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(adminSVTNCreateResponse): %v", err)
	}
	if decoded.SVTNID != original.SVTNID {
		t.Errorf("svtn_id round-trip: got %q; want %q", decoded.SVTNID, original.SVTNID)
	}
	if decoded.BootstrapFingerprint != original.BootstrapFingerprint {
		t.Errorf("bootstrap_fingerprint round-trip: got %q; want %q", decoded.BootstrapFingerprint, original.BootstrapFingerprint)
	}
}

// TestAdminSVTNCreateArgs_JSONRoundTrip verifies that adminSVTNCreateArgs
// serialises with the correct JSON field name (name) per the AC-002 envelope.
//
// AC-002 — wire args shape: {"command":"admin.svtn.create","args":{"name":"<name>"}}.
func TestAdminSVTNCreateArgs_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	// AC-002 — wire args field name for admin.svtn.create.
	original := adminSVTNCreateArgs{Name: "my-network"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal(adminSVTNCreateArgs): %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map: %v", err)
	}

	if _, ok := raw["name"]; !ok {
		t.Error("AC-002: adminSVTNCreateArgs: missing JSON field 'name'")
	}

	var decoded adminSVTNCreateArgs
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(adminSVTNCreateArgs): %v", err)
	}
	if decoded.Name != original.Name {
		t.Errorf("name round-trip: got %q; want %q", decoded.Name, original.Name)
	}
}

// containsString is a simple substring helper for test assertions.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
