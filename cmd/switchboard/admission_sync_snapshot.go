// admission_sync_snapshot.go — VLR-local admitted-state snapshot
// serialization/deserialization and atomic write (S-BL.ADMISSION-SYNC-WIRE;
// BC-2.05.010; decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md v1.2 Ruling 3).
//
// Snapshot JSON schema (schema_version: 1):
//
//	{
//	  "schema_version": 1,
//	  "timestamp": "<RFC3339 UTC>",
//	  "svtns": [
//	    {
//	      "svtn_id": "<32 lowercase hex chars = [16]byte UUID>",
//	      "keys": [
//	        {
//	          "pubkey": "<base64url no-padding, 32-byte raw Ed25519 key>",
//	          "role":   "<control|console|access>",
//	          "revoked": false,
//	          "expiry": "<RFC3339 UTC, omitempty>"
//	        }
//	      ]
//	    }
//	  ]
//	}
//
// NOT stored in snapshot: admitted (always false on load), FrameAuthKey
// (derived on load by RegisterKey), NodeAddr (derived on load), nonces (ephemeral).
//
// Atomic write: write serialised JSON to <path>.tmp, then os.Rename.
//
// ARCH-08 compliance: cmd/switchboard (position 18, the top).
// Purity classification (ARCH-09): boundary-effectful (file I/O).

package main

import (
	"errors"

	"github.com/arcavenae/switchboard/internal/admission"
)

// errSnapshotNotImplemented is the stub sentinel returned by snapshot functions.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — real snapshot logic would satisfy AC-006/007; therefore stubs.
var errSnapshotNotImplemented = errors.New("admission snapshot: not implemented")

// snapshotCurrentSchemaVersion is the schema_version written and accepted by
// this implementation (BC-2.05.010; Decision 6). Forward-compat gate: an
// unrecognised schema_version causes fail-closed on startup (EC-011 / E-KEY-002).
const snapshotCurrentSchemaVersion = 1

// snapshotFile is the JSON representation of the full admitted-state snapshot.
// Field ordering mirrors Decision 6's schema definition.
type snapshotFile struct {
	SchemaVersion int           `json:"schema_version"`
	Timestamp     string        `json:"timestamp"`
	SVTNs         []snapshotSVTN `json:"svtns"`
}

// snapshotSVTN groups all key entries for a single SVTN in the snapshot.
type snapshotSVTN struct {
	SVTNID string         `json:"svtn_id"` // 32 lowercase hex chars = [16]byte UUID
	Keys   []snapshotKey  `json:"keys"`
}

// snapshotKey is a single key entry within a SVTN's key list.
type snapshotKey struct {
	PubKey  string `json:"pubkey"`            // base64url no-padding, 32-byte raw Ed25519 key
	Role    string `json:"role"`              // "control", "console", or "access"
	Revoked bool   `json:"revoked"`
	Expiry  string `json:"expiry,omitempty"`  // RFC3339 UTC; omit if no expiry
}

// marshalSnapshot serialises the current state of ks to a snapshotFile.
//
// STUB: returns errSnapshotNotImplemented so AC-006 tests FAIL at Red Gate.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func marshalSnapshot(_ *admission.AdmittedKeySet) (*snapshotFile, error) {
	return nil, errSnapshotNotImplemented
}

// unmarshalSnapshot deserialises a snapshotFile and calls RegisterKey (plus
// RevokeKey and SetKeyExpiry as appropriate) on ks for each entry.
//
// Forward-compat gate: returns an error wrapping E-KEY-002 when
// snap.SchemaVersion != snapshotCurrentSchemaVersion (EC-011).
//
// STUB: returns errSnapshotNotImplemented so AC-007 tests FAIL at Red Gate.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func unmarshalSnapshot(_ *snapshotFile, _ *admission.AdmittedKeySet) error {
	return errSnapshotNotImplemented
}

// writeSnapshotAtomic serialises ks and writes the JSON to path atomically:
// write to <path>.tmp, then os.Rename (BC-2.05.010 Invariant 1 / Decision 6).
//
// STUB: returns errSnapshotNotImplemented so AC-005/006 tests FAIL at Red Gate.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func writeSnapshotAtomic(_ string, _ *admission.AdmittedKeySet) error {
	return errSnapshotNotImplemented
}

// loadSnapshotFromFile reads and deserialises the snapshot file at path into ks.
// Semantics:
//   - path == "": no-op, return nil (empty keyset, existing behaviour).
//   - file absent: return nil with empty keyset + caller logs INFO (Decision 7b).
//   - file present but invalid JSON or unknown schema_version: return non-nil
//     error wrapping E-KEY-002 so runRouter fails closed (Decision 7c / EC-011).
//   - file present and valid: populate ks and return nil (Decision 7d).
//
// STUB: returns errSnapshotNotImplemented so AC-007 tests FAIL at Red Gate.
//
// Self-Check (BC-5.38.005 invariant 1): "If I include this real implementation,
// will the test for this function pass trivially without any implementer work?"
// Answer: Yes — therefore stub.
func loadSnapshotFromFile(_ string, _ *admission.AdmittedKeySet) error {
	return errSnapshotNotImplemented
}
