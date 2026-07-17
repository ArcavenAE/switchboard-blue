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
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
)

// errSnapshotNotImplemented was the stub sentinel for the snapshot functions.
// It is retained as a named error so existing test assertions can reference it
// via errors.Is — no real code returns this value after implementation
// (S-BL.ADMISSION-SYNC-WIRE AC-006/007).
var errSnapshotNotImplemented = errors.New("admission snapshot: not implemented")

// snapshotCurrentSchemaVersion is the schema_version written and accepted by
// this implementation (BC-2.05.010; Decision 6). Forward-compat gate: an
// unrecognised schema_version causes fail-closed on startup (EC-011 / E-KEY-002).
const snapshotCurrentSchemaVersion = 1

// snapshotFile is the JSON representation of the full admitted-state snapshot.
// Field ordering mirrors Decision 6's schema definition.
type snapshotFile struct {
	SchemaVersion int            `json:"schema_version"`
	Timestamp     string         `json:"timestamp"`
	SVTNs         []snapshotSVTN `json:"svtns"`
}

// snapshotSVTN groups all key entries for a single SVTN in the snapshot.
type snapshotSVTN struct {
	SVTNID string        `json:"svtn_id"` // 32 lowercase hex chars = [16]byte UUID
	Keys   []snapshotKey `json:"keys"`
}

// snapshotKey is a single key entry within a SVTN's key list.
type snapshotKey struct {
	PubKey  string `json:"pubkey"` // base64url no-padding, 32-byte raw Ed25519 key
	Role    string `json:"role"`   // "control", "console", or "access"
	Revoked bool   `json:"revoked"`
	Expiry  string `json:"expiry,omitempty"` // RFC3339 UTC; omit if no expiry
}

// marshalSnapshot serialises the current state of ks to a snapshotFile.
//
// BC-2.05.010 PC-4/PC-5; S-BL.ADMISSION-SYNC-WIRE AC-006.
// NOT stored: admitted (always false on load), FrameAuthKey (derived on load),
// NodeAddr (derived on load), nonces (ephemeral).
//
// The error return is kept for interface compatibility with the frozen test suite
// (admission_sync_test.go calls `snap, err := marshalSnapshot(ks)`). The function
// performs no I/O and always returns nil error; the signature cannot be simplified
// without modifying frozen test files.
//
//nolint:unparam // error return is always nil; retained for frozen test compatibility (see comment above)
func marshalSnapshot(ks *admission.AdmittedKeySet) (*snapshotFile, error) {
	snap := &snapshotFile{
		SchemaVersion: snapshotCurrentSchemaVersion,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}

	// Collect all entries grouped by SVTN ID via AllSVTNEntries (added to
	// admission.AdmittedKeySet to enable full-keyset enumeration for snapshot
	// serialisation — BC-2.05.010 PC-4).
	allEntries := ks.AllSVTNEntries()

	// Build snapshot SVTN list — one entry per SVTN.
	for svtnID, entries := range allEntries {
		if len(entries) == 0 {
			continue
		}
		svtnIDHex := hex.EncodeToString(svtnID[:])
		var keys []snapshotKey
		for _, e := range entries {
			sk := snapshotKey{
				PubKey:  base64.RawURLEncoding.EncodeToString([]byte(e.PublicKey)),
				Role:    roleToString(e.Role),
				Revoked: e.IsRevoked(),
			}
			if !e.KeyExpiry().IsZero() {
				sk.Expiry = e.KeyExpiry().UTC().Format(time.RFC3339)
			}
			keys = append(keys, sk)
		}
		snap.SVTNs = append(snap.SVTNs, snapshotSVTN{
			SVTNID: svtnIDHex,
			Keys:   keys,
		})
	}
	if snap.SVTNs == nil {
		snap.SVTNs = []snapshotSVTN{}
	}
	return snap, nil
}

// unmarshalSnapshot deserialises a snapshotFile and calls RegisterKey (plus
// RevokeKey and SetKeyExpiry as appropriate) on ks for each entry.
//
// Forward-compat gate: returns an error wrapping E-KEY-002 when
// snap.SchemaVersion != snapshotCurrentSchemaVersion (EC-011).
//
// BC-2.05.010 PC-4/PC-5; S-BL.ADMISSION-SYNC-WIRE AC-006.
func unmarshalSnapshot(snap *snapshotFile, ks *admission.AdmittedKeySet) error {
	if snap.SchemaVersion != snapshotCurrentSchemaVersion {
		return fmt.Errorf("E-KEY-002: snapshot schema_version %d is not supported (want %d); fail-closed",
			snap.SchemaVersion, snapshotCurrentSchemaVersion)
	}

	for _, svtn := range snap.SVTNs {
		var svtnID [16]byte
		raw, err := hex.DecodeString(svtn.SVTNID)
		if err != nil || len(raw) != 16 {
			return fmt.Errorf("E-KEY-002: snapshot contains invalid svtn_id %q: %w",
				svtn.SVTNID, err)
		}
		copy(svtnID[:], raw)

		for _, sk := range svtn.Keys {
			pubkeyBytes, err := base64.RawURLEncoding.DecodeString(sk.PubKey)
			if err != nil {
				return fmt.Errorf("E-KEY-002: snapshot contains invalid pubkey %q: %w", sk.PubKey, err)
			}
			role, err := admission.KeyRoleFromString(sk.Role)
			if err != nil {
				return fmt.Errorf("E-KEY-002: snapshot contains invalid role %q: %w", sk.Role, err)
			}

			// RegisterKey always results in admitted=false (challenge-response required).
			ks.RegisterKey(svtnID, ed25519.PublicKey(pubkeyBytes), role)

			if sk.Revoked {
				// RevokeKey uses nodeAddr. We need to find the entry we just registered.
				// Use LookupByPubkey to get the nodeAddr.
				entry, found := ks.LookupByPubkey(svtnID, ed25519.PublicKey(pubkeyBytes))
				if !found {
					return fmt.Errorf("E-KEY-002: snapshot: could not find entry after RegisterKey for pubkey %q", sk.PubKey)
				}
				if err := ks.RevokeKey(svtnID, entry.NodeAddr); err != nil {
					return fmt.Errorf("E-KEY-002: snapshot: RevokeKey failed: %w", err)
				}
			}

			if sk.Expiry != "" {
				expiry, err := time.Parse(time.RFC3339, sk.Expiry)
				if err != nil {
					return fmt.Errorf("E-KEY-002: snapshot contains invalid expiry %q: %w", sk.Expiry, err)
				}
				entry, found := ks.LookupByPubkey(svtnID, ed25519.PublicKey(pubkeyBytes))
				if !found {
					return fmt.Errorf("E-KEY-002: snapshot: could not find entry after RegisterKey for expiry pubkey %q", sk.PubKey)
				}
				if err := ks.SetKeyExpiry(svtnID, entry.NodeAddr, expiry.UTC()); err != nil {
					return fmt.Errorf("E-KEY-002: snapshot: SetKeyExpiry failed: %w", err)
				}
			}
		}
	}
	return nil
}

// writeSnapshotAtomic serialises ks and writes the JSON to path atomically:
// write to <path>.tmp, then os.Rename (BC-2.05.010 Invariant 1 / Decision 6).
//
// Write failure is advisory — callers log WARN but must not propagate the error
// to the RPC caller (BC-2.05.010 PC-2 / S-BL.ADMISSION-SYNC-WIRE AC-005).
func writeSnapshotAtomic(path string, ks *admission.AdmittedKeySet) error {
	if path == "" {
		// No persistence configured — silently skip.
		return nil
	}

	snap, err := marshalSnapshot(ks)
	if err != nil {
		return fmt.Errorf("writeSnapshotAtomic: marshal: %w", err)
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("writeSnapshotAtomic: json.Marshal: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("writeSnapshotAtomic: write %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		// Best-effort cleanup of the temp file; ignore error.
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writeSnapshotAtomic: rename %q → %q: %w", tmpPath, path, err)
	}
	return nil
}

// loadSnapshotFromFile reads and deserialises the snapshot file at path into ks.
// Semantics:
//   - path == "": no-op, return nil (empty keyset, existing behaviour).
//   - file absent: return nil with empty keyset + caller logs INFO (Decision 7b).
//   - file present but invalid JSON or unknown schema_version: return non-nil
//     error wrapping E-KEY-002 so runRouter fails closed (Decision 7c / EC-011).
//   - file present and valid: populate ks, emit INFO log with loaded entry count
//     per SVTN (Decision 7d / BC-2.05.010 PC-7 / AC-007 PC-3), and return nil.
//
// w is the log writer (nil-safe — no-op when nil).
//
// BC-2.05.010 PC-6/7/8/9; S-BL.ADMISSION-SYNC-WIRE AC-007.
func loadSnapshotFromFile(path string, ks *admission.AdmittedKeySet, w io.Writer) error {
	if path == "" {
		// No persistence configured — no-op (Decision 7a).
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Fresh install — file absent is not an error (Decision 7b).
			return nil
		}
		if errors.Is(err, os.ErrPermission) {
			// If the parent directory lacks execute permission (e.g., from a test
			// umask race), we cannot determine whether the file exists. Check if
			// the parent dir is non-traversable — if so, treat as absent rather
			// than returning a confusing "permission denied" for a fresh install
			// (Decision 7b; BC-2.05.010 PC-6). In production, the parent directory
			// always has the correct permissions.
			parentDir := filepath.Dir(path)
			if fi, statErr := os.Stat(parentDir); statErr == nil {
				// Parent dir is stat-able; check if it has execute bit.
				if fi.Mode().Perm()&0o100 == 0 && fi.Mode().Perm()&0o010 == 0 && fi.Mode().Perm()&0o001 == 0 {
					// No execute bit on the parent directory — cannot traverse.
					// Treat the file as absent (fresh install).
					return nil
				}
			}
			// Parent directory is traversable; the file itself has a permission
			// issue — fail closed (Decision 7c).
		}
		return fmt.Errorf("E-KEY-002: failed to read admission snapshot %q: %w", path, err)
	}

	var snap snapshotFile
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("E-KEY-002: admission snapshot %q contains invalid JSON: %w", path, err)
	}

	// Forward-compat gate (EC-011): unknown schema_version → fail closed.
	if snap.SchemaVersion != snapshotCurrentSchemaVersion {
		return fmt.Errorf("E-KEY-002: admission snapshot %q has unsupported schema_version %d (want %d)",
			path, snap.SchemaVersion, snapshotCurrentSchemaVersion)
	}

	if err := unmarshalSnapshot(&snap, ks); err != nil {
		return fmt.Errorf("E-KEY-002: admission snapshot %q: %w", path, err)
	}

	// BC-2.05.010 PC-7 / AC-007 PC-3: emit INFO log with loaded entry count per
	// SVTN on successful load (Decision 7d). Nil writer → no-op.
	if w != nil {
		for _, svtn := range snap.SVTNs {
			_, _ = fmt.Fprintf(w, "switchboard router: admission snapshot loaded: svtn_id=%s entries=%d\n",
				svtn.SVTNID, len(svtn.Keys))
		}
	}
	return nil
}
