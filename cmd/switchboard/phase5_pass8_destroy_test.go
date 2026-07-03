// Package main — Phase 5 Pass 8 RED test for admin.svtn.destroy name validation.
//
// F-P5P8-A-004 (MED): makeAdminSVTNDestroyHandler (admin_handlers.go:777-810)
// only checks for empty name (E-CFG-001: missing required field: name).
// Spec §120 (interface-definitions.md v1.19) promises E-CFG-001 for all five
// validateSVTNName arms, mirroring the create handler.  The five un-checked arms
// are:
//   - whitespace-only name
//   - name exceeding 255 bytes
//   - invalid UTF-8
//   - ASCII control characters
//   - Unicode line/paragraph separators (U+2028, U+2029)
//
// Currently whitespace-only, >255-byte, and control-char names fall through
// to the SVTNManager.Destroy call and return E-SVTN-003 (not found) instead
// of E-CFG-001.
//
// Spec authority: interface-definitions.md v1.19 §120; F-P2L2 (exhaustive
// name validation); F-Impl-001 (UTF-8 + control-char handling).
// Finding: F-P5P8-A-004.
package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/arcavenae/switchboard/internal/mgmt"
	"github.com/arcavenae/switchboard/internal/svtnmgmt"
)

// TestAdminSVTNDestroyHandler_NameValidation_E_CFG_001 is a table-driven test
// covering the five validateSVTNName arms on the destroy handler.  All cases
// must return E-CFG-001 — never E-SVTN-003 (not-found) or any other code.
//
// RED (F-P5P8-A-004): cases 2-5 currently fall through to m.Destroy and receive
// E-SVTN-003 because makeAdminSVTNDestroyHandler only validates empty-name.
//
// Mirrors the style of TestAdminSVTNDestroy_ArgsValidation_E_CFG_001 and
// TestValidateSVTNName in admin_handlers_test.go.
//
// Traces to BC-2.07.001 PC-3; interface-definitions.md v1.19 §120;
// F-P2L2 (exhaustive validation); F-Impl-001; F-P5P8-A-004.
func TestAdminSVTNDestroyHandler_NameValidation_E_CFG_001(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	destroyFn := destroyHandlerFn(t, m, mgmt.NewOperatorKeySet(nil))
	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)

	cases := []struct {
		name    string // test case name
		svtnArg string // the value passed as "name" in the JSON args
		// wantErrCode is the exact error-code prefix the response error must contain.
		// All cases must be E-CFG-001 (not E-SVTN-003).
		wantErrCode string
	}{
		{
			// Case 1: empty name — already validated by current code (guard).
			// This must stay E-CFG-001; if it regresses the test setup is broken.
			name:        "empty_name",
			svtnArg:     "",
			wantErrCode: "E-CFG-001",
		},
		{
			// Case 2: whitespace-only name — validateSVTNName arm 2.
			// Current code: falls through to m.Destroy → E-SVTN-003 (not found).
			// Expected after fix: E-CFG-001.
			name:        "whitespace_only_spaces",
			svtnArg:     "   ",
			wantErrCode: "E-CFG-001",
		},
		{
			// Case 3: name exceeding 255 bytes — validateSVTNName arm 3.
			// Current code: falls through to m.Destroy → E-SVTN-003 (not found).
			// Expected after fix: E-CFG-001.
			name:        "name_exceeds_255_bytes",
			svtnArg:     strings.Repeat("a", 256),
			wantErrCode: "E-CFG-001",
		},
		{
			// Case 4: invalid UTF-8 — validateSVTNName arm 4 (F-Impl-001).
			// The byte sequence 0x80 is a bare continuation byte — invalid UTF-8.
			// Current code: falls through to m.Destroy → E-SVTN-003 (not found).
			// Expected after fix: E-CFG-001.
			// Note: json.Marshal rejects invalid UTF-8, so we construct the raw
			// JSON arg directly to bypass Go's encoding layer.
			name:        "invalid_utf8",
			svtnArg:     "\x80bad",
			wantErrCode: "E-CFG-001",
		},
		{
			// Case 5: ASCII control character — validateSVTNName arm 5 (F-Impl-001).
			// U+0007 (BEL) is a control character caught by unicode.IsControl.
			// Current code: falls through to m.Destroy → E-SVTN-003 (not found).
			// Expected after fix: E-CFG-001.
			name:        "ascii_control_char_BEL",
			svtnArg:     "valid-prefix\x07rest",
			wantErrCode: "E-CFG-001",
		},
		{
			// Case 6: Unicode line separator U+2028 — validateSVTNName arm 5
			// (F-Impl-001: explicit check because unicode.IsControl does not cover
			// Zl/Zp categories).
			// Current code: falls through to m.Destroy → E-SVTN-003 (not found).
			// Expected after fix: E-CFG-001.
			name:        "unicode_line_separator_U2028",
			svtnArg:     "valid name",
			wantErrCode: "E-CFG-001",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Construct the raw JSON args.  For the invalid-UTF-8 case we build
			// the JSON string manually since json.Marshal would fail or replace
			// the bad bytes with U+FFFD — defeating the test intent.
			var rawArgs json.RawMessage
			if tc.name == "invalid_utf8" {
				// Hand-craft the JSON: {"name":"bad"} is NOT what we want
				// (that's valid UTF-8).  We need the raw byte 0x80 inside a JSON
				// string.  RFC 7159 §8.1 requires UTF-8, but the handler must
				// reject it at unmarshal time or name-validation time.
				// json.Unmarshal on a Go string with embedded 0x80 treats it as
				// an opaque byte sequence; when it hits validateSVTNName's
				// utf8.ValidString check the bad byte is caught.
				// Encode the args struct normally to get a valid envelope, then
				// replace the name value with a raw byte sequence.
				rawArgs = json.RawMessage(`{"name":"` + "\x80bad" + `"}`)
			} else {
				b, err := json.Marshal(adminSVTNDestroyArgs{Name: tc.svtnArg})
				if err != nil {
					// If json.Marshal itself rejects the input (e.g. invalid UTF-8 for
					// some Go versions), the raw bytes go directly to the handler.
					rawArgs = json.RawMessage(`{"name":"` + tc.svtnArg + `"}`)
					_ = err
				} else {
					rawArgs = json.RawMessage(b)
				}
			}

			_, err := destroyFn(ctx, rawArgs)

			if err == nil {
				t.Fatalf("F-P5P8-A-004: %s: expected %s error; got nil (SVTN was destroyed or no error returned)",
					tc.name, tc.wantErrCode)
			}

			if !strings.Contains(err.Error(), tc.wantErrCode) {
				t.Errorf("F-P5P8-A-004: %s: expected error containing %q; got: %v\n"+
					"(if error contains E-SVTN-003 the handler is not calling validateSVTNName on the destroy path)",
					tc.name, tc.wantErrCode, err)
			}

			// Secondary guard: must NOT be E-SVTN-003 (not-found), which is the
			// current (wrong) fallthrough behavior.
			if strings.Contains(err.Error(), "E-SVTN-003") {
				t.Errorf("F-P5P8-A-004: %s: handler fell through to Destroy and returned E-SVTN-003; "+
					"name validation (E-CFG-001) must fire first", tc.name)
			}
		})
	}
}

// TestAdminSVTNDestroyHandler_NameValidation_ValidNames_Unaffected verifies
// that the new name validation does not reject well-formed names.
//
// This is a GREEN guard: the cases here must pass both before and after the
// implementer adds validateSVTNName to the destroy handler.  If they regress
// the fix incorrectly tightened the validation gate.
//
// Traces to BC-2.07.001 PC-3; F-P5P8-A-004 (positive guard).
func TestAdminSVTNDestroyHandler_NameValidation_ValidNames_Unaffected(t *testing.T) {
	t.Parallel()

	m, bootstrapPub := newTestSVTNManagerDetailed(t)
	destroyFn := destroyHandlerFn(t, m, mgmt.NewOperatorKeySet(nil))
	ctx := mgmt.WithCallerPubkey(context.Background(), bootstrapPub)

	validNames := []string{
		"test-svtn", // pre-seeded in newTestSVTNManagerDetailed — will succeed
		"valid-name",
		"A",
		strings.Repeat("x", 255), // exactly at the limit
	}

	for _, name := range validNames {
		name := name
		t.Run("valid_name_"+name[:min(len(name), 12)], func(t *testing.T) {
			t.Parallel()

			b, _ := json.Marshal(adminSVTNDestroyArgs{Name: name})
			_, err := destroyFn(ctx, json.RawMessage(b))

			// For pre-seeded "test-svtn": expect success.
			// For other names that don't exist: expect E-SVTN-003 (not-found) —
			// the name passed validation and the error came from Destroy itself.
			// The key invariant: must NOT be E-CFG-001.
			if err != nil && strings.Contains(err.Error(), "E-CFG-001") {
				t.Errorf("F-P5P8-A-004 (guard): valid name %q incorrectly rejected with E-CFG-001: %v",
					name, err)
			}
		})
	}
}

// min returns the smaller of a and b.  Replaces the builtin only available in
// Go 1.21+; the go.mod may target an earlier version.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// svtnDestroyArgs is an unexported alias used in this test file so it does not
// collide with adminSVTNDestroyArgs from admin_handlers.go (which is accessible
// within the same package).
//
// Note: adminSVTNDestroyArgs is defined in admin_handlers.go and accessible
// here since both are in package main.
var _ = adminSVTNDestroyArgs{}

// Ensure destroyHandlerFn is accessible (it is defined in admin_handlers_test.go,
// also package main, so it is always in scope).
var _ = func() func(context.Context, json.RawMessage) (any, error) {
	return nil
}

// Ensure svtnmgmt and mgmt are used (they appear in the test body above).
var (
	_ *svtnmgmt.SVTNManager
	_ = mgmt.NewOperatorKeySet
)
