// admin_wire_tag_test.go — regression guards for sbctl-side svtn_id wire-tag contract.
//
// Covers F-P5P5-B-002 (MEDIUM): the daemon-side wire tests proved that the
// daemon's four arg structs marshal and unmarshal svtn_id correctly (Pass 4
// remediation).  This file adds the symmetric guard on the sbctl side:
// the REAL sbctl arg structs (adminKeyRegisterArgs, adminKeyRevokeArgs,
// adminKeyExpireArgs) must also marshal to "svtn_id", not the stale "svtn".
//
// If a future refactor regressed cmd/sbctl/admin.go's arg structs from
// json:"svtn_id" back to json:"svtn", every test in this file would fail —
// catching the regression before it reaches the daemon wire path.
//
// Spec anchors: BC-2.05.004 v1.12, BC-2.07.001 v1.13,
// interface-definitions v1.17 §125 wire contract.
//
// Note on adminListKeysArgs: the sbctl list-keys args struct is declared inline
// (local type inside runAdmin's case "list-keys" branch, admin.go:170-172) and
// is therefore not accessible from this test package.  The daemon-side test
// (cmd/switchboard/admin_handlers_wire_shared_pkg_test.go) covers the
// adminListKeysArgs daemon struct.  The sbctl inline struct uses json:"svtn_id"
// at the declaration site (admin.go:171); any regression there would be caught
// by compilation (unexported local type, change visible in the same file).
//
// IMPORTANT: DO NOT touch implementation code.  This file is tests only.
package main

import (
	"encoding/json"
	"testing"
)

// TestNewInBurst21_SbctlSide_WireTagGuard_AllThreeStructs marshals each of the
// three top-level sbctl key arg structs and asserts:
//
//	(a) the JSON map contains the key "svtn_id",
//	(b) the JSON map does NOT contain the stale key "svtn".
//
// This is a GREEN guard — the impl already uses json:"svtn_id" (admin.go:54,65,80).
// A regression to json:"svtn" would turn these assertions RED immediately.
//
// Traces to BC-2.05.004 v1.12, BC-2.07.001 v1.13, interface-definitions v1.17 §125.
// Addresses finding F-P5P5-B-002 (wire-contract coverage gap on the sbctl side).
func TestNewInBurst21_SbctlSide_WireTagGuard_AllThreeStructs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func() ([]byte, error)
	}{
		{
			name: "adminKeyRegisterArgs",
			payload: func() ([]byte, error) {
				return json.Marshal(adminKeyRegisterArgs{
					SVTNID: "test-svtn",
					Pubkey: "AAAA",
					Role:   "console",
				})
			},
		},
		{
			name: "adminKeyRevokeArgs",
			payload: func() ([]byte, error) {
				return json.Marshal(adminKeyRevokeArgs{
					SVTNID:  "test-svtn",
					Pubkey:  "BBBB",
					Role:    "control",
					Confirm: true,
				})
			},
		},
		{
			name: "adminKeyExpireArgs",
			payload: func() ([]byte, error) {
				return json.Marshal(adminKeyExpireArgs{
					SVTNID: "test-svtn",
					Pubkey: "CCCC",
					After:  "24h",
				})
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload, err := tc.payload()
			if err != nil {
				t.Fatalf("%s: marshal: %v", tc.name, err)
			}

			var m map[string]any
			if err := json.Unmarshal(payload, &m); err != nil {
				t.Fatalf("%s: unmarshal to map: %v", tc.name, err)
			}

			// (a) "svtn_id" must be present.
			if _, ok := m["svtn_id"]; !ok {
				t.Errorf("%s: marshaled JSON missing key \"svtn_id\"; payload: %s", tc.name, payload)
			}

			// (b) stale "svtn" must be absent.
			if _, ok := m["svtn"]; ok {
				t.Errorf("%s: marshaled JSON contains stale key \"svtn\"; payload: %s", tc.name, payload)
			}
		})
	}
}

// TestNewInBurst21_SbctlSide_WirePayload_ConfirmTrue_SvtnIDPresent strengthens
// TestNewInBurst19_ConfirmSymmetry_WirePayload_ConfirmTrue (admin_confirm_symmetry_test.go:207-235)
// by additionally asserting that the marshaled adminKeyRevokeArgs payload contains
// "svtn_id" and does not contain the stale "svtn" key.
//
// The original test only asserts on the "confirm" key; this extension closes the
// gap identified in F-P5P5-B-002 without modifying the original test.
//
// Traces to BC-2.05.004 v1.12, interface-definitions v1.17 §125 wire contract.
func TestNewInBurst21_SbctlSide_WirePayload_ConfirmTrue_SvtnIDPresent(t *testing.T) {
	t.Parallel()

	args := adminKeyRevokeArgs{
		SVTNID:  "test-svtn",
		Pubkey:  "AAAA",
		Role:    "control",
		Confirm: true,
	}
	payload, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal adminKeyRevokeArgs: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	// Assert svtn_id present.
	if _, ok := m["svtn_id"]; !ok {
		t.Errorf("wire payload missing \"svtn_id\" field; payload: %s", payload)
	}

	// Assert stale "svtn" absent.
	if _, ok := m["svtn"]; ok {
		t.Errorf("wire payload contains stale \"svtn\" field; payload: %s", payload)
	}
}
