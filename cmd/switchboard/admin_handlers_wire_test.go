// admin_handlers_wire_test.go — RED tests for Phase 5 Pass 4 remediation.
//
// Covers F-A-001 / F-A-002 (HIGH): wire field name drift.
// sbctl structs use json:"svtn_id" but daemon structs use json:"svtn".
// A JSON round-trip through the wire MUST populate the daemon SVTNName field.
// These tests MUST FAIL until the daemon structs are updated to json:"svtn_id".
//
// Test naming follows BC-based convention per VSDD factory test-writer rules.
// Each function is named TestNewInBurst19_<subject> so that the Red Gate
// command `go test ./cmd/... -run TestNewInBurst19` selects all new tests.
//
// BC-2.05.004 / BC-2.07.001 — wire field names must be consistent.
// F-A-001 (HIGH): all four daemon key/SVTN arg structs use json:"svtn" (wrong).
// F-A-002 (HIGH): sbctl already uses json:"svtn_id" (correct spec-truth).
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// sbctlSideKeyRegisterArgs mirrors the sbctl struct that already uses
// json:"svtn_id". This is the correct, spec-truth shape.
type sbctlSideKeyRegisterArgs struct {
	SVTNID string `json:"svtn_id"`
	Pubkey string `json:"pubkey_openssh"`
	Role   string `json:"role"`
}

// sbctlSideKeyRevokeArgs mirrors the sbctl revoke struct.
type sbctlSideKeyRevokeArgs struct {
	SVTNID  string `json:"svtn_id"`
	Pubkey  string `json:"pubkey_openssh"`
	Role    string `json:"role"`
	Confirm bool   `json:"confirm"`
}

// sbctlSideKeyExpireArgs mirrors the sbctl expire struct.
type sbctlSideKeyExpireArgs struct {
	SVTNID string `json:"svtn_id"`
	Pubkey string `json:"pubkey_openssh"`
	After  string `json:"after"`
}

// sbctlSideListKeysArgs mirrors the sbctl list-keys struct.
// (sbctl does not currently have a separate list-keys args struct but may
// pass svtn_id inline; this covers the daemon's adminListKeysArgs field.)
type sbctlSideListKeysArgs struct {
	SVTNID     string `json:"svtn_id"`
	CallerRole string `json:"caller_role"`
}

// TestNewInBurst19_WireField_KeyRegister_SvtnID verifies that JSON marshaled
// by sbctl (using svtn_id) is correctly unmarshaled by the daemon's
// adminKeyRegisterArgs struct.
//
// MUST FAIL until daemon's adminKeyRegisterArgs is updated to json:"svtn_id".
// Covers F-A-001 / F-A-002 (HIGH) for admin.key.register.
func TestNewInBurst19_WireField_KeyRegister_SvtnID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sbctlArgs  sbctlSideKeyRegisterArgs
		wantSVTN   string
		wantPubkey string
		wantRole   string
	}{
		{
			name:       "standard_svtn_id",
			sbctlArgs:  sbctlSideKeyRegisterArgs{SVTNID: "my-svtn", Pubkey: "AAAA", Role: "console"},
			wantSVTN:   "my-svtn",
			wantPubkey: "AAAA",
			wantRole:   "console",
		},
		{
			name:       "control_role",
			sbctlArgs:  sbctlSideKeyRegisterArgs{SVTNID: "prod-svtn", Pubkey: "BBBB", Role: "control"},
			wantSVTN:   "prod-svtn",
			wantPubkey: "BBBB",
			wantRole:   "control",
		},
		{
			name:       "hyphenated_svtn_name",
			sbctlArgs:  sbctlSideKeyRegisterArgs{SVTNID: "alpha-beta-gamma", Pubkey: "CCCC", Role: "access"},
			wantSVTN:   "alpha-beta-gamma",
			wantPubkey: "CCCC",
			wantRole:   "access",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Marshal using sbctl shape (json:"svtn_id").
			payload, err := json.Marshal(tc.sbctlArgs)
			if err != nil {
				t.Fatalf("marshal sbctl args: %v", err)
			}

			// Unmarshal into the daemon's struct. If the daemon uses json:"svtn"
			// (current broken state), SVTNName will be empty after unmarshal.
			var daemonArgs adminKeyRegisterArgs
			if err := json.Unmarshal(payload, &daemonArgs); err != nil {
				t.Fatalf("unmarshal into daemon adminKeyRegisterArgs: %v", err)
			}

			// This assertion FAILS with current code because daemon uses json:"svtn"
			// but payload contains "svtn_id".
			if daemonArgs.SVTNName != tc.wantSVTN {
				t.Errorf("SVTNName round-trip: got %q, want %q\n  payload was: %s\n  (daemon struct uses json:\"svtn\" — must be json:\"svtn_id\")",
					daemonArgs.SVTNName, tc.wantSVTN, payload)
			}
			if daemonArgs.PublicKey != tc.wantPubkey {
				t.Errorf("PublicKey round-trip: got %q, want %q", daemonArgs.PublicKey, tc.wantPubkey)
			}
			if daemonArgs.Role != tc.wantRole {
				t.Errorf("Role round-trip: got %q, want %q", daemonArgs.Role, tc.wantRole)
			}
		})
	}
}

// TestNewInBurst19_WireField_KeyRevoke_SvtnID verifies the same round-trip for
// adminKeyRevokeArgs.
//
// MUST FAIL until daemon's adminKeyRevokeArgs is updated to json:"svtn_id".
// Covers F-A-001 / F-A-002 (HIGH) for admin.key.revoke.
func TestNewInBurst19_WireField_KeyRevoke_SvtnID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      sbctlSideKeyRevokeArgs
		wantSVTN string
	}{
		{
			name:     "console_role_no_confirm",
			src:      sbctlSideKeyRevokeArgs{SVTNID: "test-svtn", Pubkey: "DDDD", Role: "console", Confirm: false},
			wantSVTN: "test-svtn",
		},
		{
			name:     "control_role_with_confirm",
			src:      sbctlSideKeyRevokeArgs{SVTNID: "prod-svtn", Pubkey: "EEEE", Role: "control", Confirm: true},
			wantSVTN: "prod-svtn",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tc.src)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var daemonArgs adminKeyRevokeArgs
			if err := json.Unmarshal(payload, &daemonArgs); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if daemonArgs.SVTNName != tc.wantSVTN {
				t.Errorf("SVTNName round-trip: got %q, want %q\n  payload: %s\n  (daemon uses json:\"svtn\" — must be json:\"svtn_id\")",
					daemonArgs.SVTNName, tc.wantSVTN, payload)
			}
		})
	}
}

// TestNewInBurst19_WireField_KeyExpire_SvtnID verifies the same round-trip for
// adminKeyExpireArgs.
//
// MUST FAIL until daemon's adminKeyExpireArgs is updated to json:"svtn_id".
// Covers F-A-001 / F-A-002 (HIGH) for admin.key.expire.
func TestNewInBurst19_WireField_KeyExpire_SvtnID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      sbctlSideKeyExpireArgs
		wantSVTN string
	}{
		{
			name:     "standard_expiry",
			src:      sbctlSideKeyExpireArgs{SVTNID: "alpha-svtn", Pubkey: "FFFF", After: "24h"},
			wantSVTN: "alpha-svtn",
		},
		{
			name:     "long_duration",
			src:      sbctlSideKeyExpireArgs{SVTNID: "beta-svtn", Pubkey: "GGGG", After: "8760h"},
			wantSVTN: "beta-svtn",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tc.src)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var daemonArgs adminKeyExpireArgs
			if err := json.Unmarshal(payload, &daemonArgs); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if daemonArgs.SVTNName != tc.wantSVTN {
				t.Errorf("SVTNName round-trip: got %q, want %q\n  payload: %s\n  (daemon uses json:\"svtn\" — must be json:\"svtn_id\")",
					daemonArgs.SVTNName, tc.wantSVTN, payload)
			}
		})
	}
}

// TestNewInBurst19_WireField_ListKeys_SvtnID verifies the same round-trip for
// adminListKeysArgs.
//
// MUST FAIL until daemon's adminListKeysArgs is updated to json:"svtn_id".
// Covers F-A-001 / F-A-002 (HIGH) for admin.key.list-keys.
func TestNewInBurst19_WireField_ListKeys_SvtnID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		src      sbctlSideListKeysArgs
		wantSVTN string
	}{
		{
			name:     "list_keys_by_svtn",
			src:      sbctlSideListKeysArgs{SVTNID: "list-svtn", CallerRole: "control"},
			wantSVTN: "list-svtn",
		},
		{
			name:     "list_keys_no_caller_role",
			src:      sbctlSideListKeysArgs{SVTNID: "gamma-svtn"},
			wantSVTN: "gamma-svtn",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tc.src)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var daemonArgs adminListKeysArgs
			if err := json.Unmarshal(payload, &daemonArgs); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if daemonArgs.SVTNName != tc.wantSVTN {
				t.Errorf("SVTNName round-trip: got %q, want %q\n  payload: %s\n  (daemon uses json:\"svtn\" — must be json:\"svtn_id\")",
					daemonArgs.SVTNName, tc.wantSVTN, payload)
			}
		})
	}
}

// TestNewInBurst19_WireField_KeyRegister_MarshaledJSONContainsSvtnID verifies
// that JSON marshaled from daemon args also uses "svtn_id" as the key
// (regression guard: ensures both decode AND encode use svtn_id).
//
// MUST FAIL until daemon's adminKeyRegisterArgs is updated to json:"svtn_id".
func TestNewInBurst19_WireField_KeyRegister_MarshaledJSONContainsSvtnID(t *testing.T) {
	t.Parallel()

	args := adminKeyRegisterArgs{SVTNName: "test-svtn", PublicKey: "HHHH", Role: "control"}
	payload, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal daemon args: %v", err)
	}

	// The marshaled JSON must contain "svtn_id", not "svtn".
	// This FAILS with current code because the tag is json:"svtn".
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if _, hasSvtnID := decoded["svtn_id"]; !hasSvtnID {
		t.Errorf("marshaled adminKeyRegisterArgs must have key \"svtn_id\"; got keys: %v\n  payload: %s",
			mapKeys(decoded), payload)
	}
	if _, hasSvtn := decoded["svtn"]; hasSvtn {
		t.Errorf("marshaled adminKeyRegisterArgs must NOT have key \"svtn\" (stale tag); payload: %s", payload)
	}
}

// mapKeys returns the sorted string slice of map keys for readable test output.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestNewInBurst19_WireField_StaleField_SvtnRejected verifies that a payload
// using the pre-v1.13 stale field name "svtn" (not "svtn_id") is rejected
// with E-CFG-001 by the daemon handler, confirming the field name contract
// is enforced on the inbound wire path (no silent compat shim).
//
// The test uses a two-case discriminating oracle to prove the assertion is not
// vacuously true (i.e., it would not pass for an empty payload):
//
//	case A: {"svtn":"stale-name"}                → must fail with E-CFG-001
//	        and the parsed SVTNName must be empty (the stale field was ignored)
//	case B: {"svtn_id":"actual-name",...}         → must succeed
//	        (proves the correct field name is accepted by the same handler path)
//
// Without case B the test cannot distinguish "stale field silently dropped AND
// handler correctly rejects empty SVTNName" from "handler always errors".
func TestNewInBurst19_WireField_StaleField_SvtnRejected(t *testing.T) {
	t.Parallel()

	t.Run("stale_svtn_field_rejected", func(t *testing.T) {
		t.Parallel()
		m := newTestSVTNManager(t)
		handler := makeRegisterHandler(m, nil)

		// Payload uses the stale "svtn" key — json.Unmarshal silently ignores it,
		// leaving SVTNName empty; the handler must return E-CFG-001 for missing svtn_id.
		stalePayload := `{"svtn":"test-svtn-id","pubkey_openssh":"ssh-ed25519 AAAA test","role":"control"}`
		_, err := handler(context.Background(), json.RawMessage(stalePayload))
		if err == nil {
			t.Fatal("expected error for stale 'svtn' field, got nil")
		}
		if !strings.Contains(err.Error(), "E-CFG-001") {
			t.Errorf("expected E-CFG-001 for missing svtn_id, got: %v", err)
		}
		if !strings.Contains(err.Error(), "svtn_id") {
			t.Errorf("expected error to mention svtn_id, got: %v", err)
		}

		// Additional assertion: confirm the stale "svtn" value was NOT silently
		// accepted by unmarshaling it and checking SVTNName is empty.
		var parsed adminKeyRegisterArgs
		if jsonErr := json.Unmarshal(json.RawMessage(stalePayload), &parsed); jsonErr != nil {
			t.Fatalf("unmarshal stale payload: %v", jsonErr)
		}
		if parsed.SVTNName != "" {
			t.Errorf("stale 'svtn' field must NOT populate SVTNName; got %q (silent acceptance)", parsed.SVTNName)
		}
	})

	t.Run("canonical_svtn_id_field_accepted", func(t *testing.T) {
		t.Parallel()
		m := newTestSVTNManager(t)
		handler := makeRegisterHandler(m, nil)

		// Generate a fresh 32-byte Ed25519 public key (raw bytes) so it is
		// guaranteed not to collide with the zero key already registered on
		// test-svtn by newTestSVTNManager.  Encode as raw base64 — decodePublicKey
		// accepts raw base64 directly (see admin_handlers.go decodePublicKey).
		var rawKey [32]byte
		if _, randErr := rand.Read(rawKey[:]); randErr != nil {
			t.Fatalf("generate fresh pubkey: %v", randErr)
		}
		freshKey := base64.StdEncoding.EncodeToString(rawKey[:])

		// Build the canonical payload using json.Marshal so escaping is correct.
		canonicalArgs := map[string]any{
			"svtn_id":        "test-svtn",
			"pubkey_openssh": freshKey,
			"role":           "console",
			"caller_role":    "control",
		}
		canonicalPayload, marshalErr := json.Marshal(canonicalArgs)
		if marshalErr != nil {
			t.Fatalf("marshal canonical payload: %v", marshalErr)
		}

		// The canonical "svtn_id" key must be accepted by the handler.
		// CallerRole "control" satisfies resolveAndVerifyCallerRole's fallback
		// path in unit tests (no pubkey in context).
		_, err := handler(context.Background(), json.RawMessage(canonicalPayload))
		if err != nil {
			t.Errorf("canonical svtn_id field must be accepted; got error: %v", err)
		}
	})
}
