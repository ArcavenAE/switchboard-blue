// admin_handlers_wire_shared_pkg_test.go — wire-tag shape tests for the
// daemon-side admin argument structs (cmd/switchboard package main).
//
// These tests verify that the four daemon arg structs (adminKeyRegisterArgs,
// adminKeyRevokeArgs, adminKeyExpireArgs, adminListKeysArgs) serialise with
// json:"svtn_id" as the SVTN identifier key.  The structs are defined inline
// in cmd/switchboard; a future refactor may extract them to internal/adminwire
// (tracked as a separate story) but that extraction is not required for these
// tests to pass.
//
// Note: the test body for TestNewInBurst19_SharedPkg_AdminwireTypes_RoundTrip
// checks that daemonArgs.SVTNName receives the value marshaled as "svtn_id",
// which only holds once both the canonical struct and the daemon struct use
// json:"svtn_id".
//
// IMPORTANT: DO NOT touch implementation code. This file is tests only.
package main

import (
	"encoding/json"
	"testing"
)

// TestNewInBurst19_SharedPkg_AdminwireTypes_RoundTrip verifies that a JSON
// payload produced from the canonical shared arg structs is preserved through
// a marshal→unmarshal cycle with the daemon's local structs.
//
// Both the canonical struct and the daemon struct use json:"svtn_id" so the
// round-trip must succeed: daemonArgs.SVTNName must equal canonical.SVTNID.
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

	// Both sides use json:"svtn_id"; the round-trip must preserve the value.
	if daemonArgs.SVTNName != canonical.SVTNID {
		t.Errorf("shared-pkg round-trip failed: daemon.SVTNName=%q, want %q\n"+
			"  payload: %s",
			daemonArgs.SVTNName, canonical.SVTNID, payload)
	}
}

// TestNewInBurst19_SharedPkg_AdminwireTypes_AllFourStructs verifies that all
// four daemon arg structs (register, revoke, expire, list-keys) use json:"svtn_id"
// by asserting that marshaling each produces a JSON object with key "svtn_id".
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

			// Must have "svtn_id" key.
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
