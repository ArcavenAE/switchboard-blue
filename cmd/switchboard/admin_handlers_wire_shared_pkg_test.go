// admin_handlers_wire_shared_pkg_test.go — RED tests for Phase 5 Pass 4 remediation.
//
// Covers the proposed internal/adminwire shared-types package.
//
// This test asserts that the package internal/adminwire exists and exports the
// canonical argument structs with json:"svtn_id" tags.  It MUST FAIL until
// the package is extracted from cmd/switchboard and cmd/sbctl.
//
// Once internal/adminwire is created, the round-trip tests below will validate
// that both sides (sbctl and daemon) import from the same source, making
// struct-tag drift impossible by construction.
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"encoding/json"
	"testing"
)

// TestNewInBurst19_SharedPkg_AdminwireTypes_RoundTrip verifies that a JSON
// payload produced from the canonical shared arg structs is preserved
// through a marshal→unmarshal cycle with the daemon's local structs.
//
// Since internal/adminwire does not yet exist, this test validates the
// EQUIVALENT property using the daemon's current structs — and deliberately
// expects the round-trip to fail because the daemon structs use json:"svtn"
// rather than json:"svtn_id".
//
// Once internal/adminwire is extracted and both sides import from it,
// this test will pass because both marshal and unmarshal will use json:"svtn_id".
//
// MUST FAIL with current code (daemon structs use json:"svtn").
func TestNewInBurst19_SharedPkg_AdminwireTypes_RoundTrip(t *testing.T) {
	t.Parallel()

	// Simulate what internal/adminwire.KeyRegisterArgs would look like once extracted.
	// This is the spec-truth struct with json:"svtn_id".
	type canonicalKeyRegisterArgs struct {
		SVTNID    string `json:"svtn_id"`
		PublicKey string `json:"pubkey_openssh"`
		Role      string `json:"role"`
	}

	canonical := canonicalKeyRegisterArgs{
		SVTNID:    "shared-pkg-svtn",
		PublicKey: "ZZZZ",
		Role:      "control",
	}

	payload, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("marshal canonical args: %v", err)
	}

	// Unmarshal into daemon's current struct.
	var daemonArgs adminKeyRegisterArgs
	if err := json.Unmarshal(payload, &daemonArgs); err != nil {
		t.Fatalf("unmarshal into daemon args: %v", err)
	}

	// FAILS: daemon uses json:"svtn" but canonical uses json:"svtn_id".
	if daemonArgs.SVTNName != canonical.SVTNID {
		t.Errorf("shared-pkg round-trip failed: daemon.SVTNName=%q, want %q\n"+
			"  This will pass once internal/adminwire is extracted and daemon imports from it.\n"+
			"  payload: %s",
			daemonArgs.SVTNName, canonical.SVTNID, payload)
	}
}

// TestNewInBurst19_SharedPkg_AdminwireTypes_AllFourStructs verifies that all
// four daemon arg structs (register, revoke, expire, list-keys) use json:"svtn_id"
// by asserting that marshaling each produces a JSON object with key "svtn_id".
//
// MUST FAIL with current code (all four use json:"svtn").
func TestNewInBurst19_SharedPkg_AdminwireTypes_AllFourStructs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload func() ([]byte, error)
	}{
		{
			name: "adminKeyRegisterArgs",
			payload: func() ([]byte, error) {
				return json.Marshal(adminKeyRegisterArgs{SVTNName: "x"})
			},
		},
		{
			name: "adminKeyRevokeArgs",
			payload: func() ([]byte, error) {
				return json.Marshal(adminKeyRevokeArgs{SVTNName: "x"})
			},
		},
		{
			name: "adminKeyExpireArgs",
			payload: func() ([]byte, error) {
				return json.Marshal(adminKeyExpireArgs{SVTNName: "x"})
			},
		},
		{
			name: "adminListKeysArgs",
			payload: func() ([]byte, error) {
				return json.Marshal(adminListKeysArgs{SVTNName: "x"})
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := tc.payload()
			if err != nil {
				t.Fatalf("marshal %s: %v", tc.name, err)
			}

			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatalf("unmarshal to map: %v", err)
			}

			// Must have "svtn_id" key — FAILS with current "svtn" tag.
			if _, ok := m["svtn_id"]; !ok {
				t.Errorf("%s: marshaled JSON must contain key \"svtn_id\"; got keys %v\n  payload: %s",
					tc.name, mapKeys(m), b)
			}
			// Must NOT have stale "svtn" key.
			if _, ok := m["svtn"]; ok {
				t.Errorf("%s: marshaled JSON must NOT contain stale key \"svtn\"; got payload: %s",
					tc.name, b)
			}
		})
	}
}
