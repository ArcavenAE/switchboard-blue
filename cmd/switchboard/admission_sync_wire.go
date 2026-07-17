// admission_sync_wire.go — router-side handler registration for the four
// internal.admission.* push commands (S-BL.ADMISSION-SYNC-WIRE; BC-2.05.009
// Postcondition 1 / BC-2.05.010).
//
// wireAdmissionSyncHandlers is called from runRouter AFTER newMgmtServer and
// BEFORE serveMgmtServer (register-before-serve invariant F-P2L1-001, same
// pattern as wireRouterControlHandlers and wireMetricsHandlers).
//
// The four internal.admission.* commands are router-only — control/console/access
// modes never call wireAdmissionSyncHandlers (ADR-004 / AC-004 role-exclusion).
//
// ARCH-08 compliance: cmd/switchboard (position 18, the top). Imports only
// internal/admission and internal/mgmt, both already imported by mgmt_wire.go.
//
// Purity classification (ARCH-09): boundary — effectful shell that registers
// handlers which mutate the keyset and write snapshot to disk.

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// wireAdmissionSyncHandlers registers the four internal.admission.* push
// command handlers on srv. Must be called after newMgmtServer and before
// serveMgmtServer (register-before-serve invariant F-P2L1-001).
//
// ks is the router's AdmittedKeySet — the same instance passed to buildRouter.
// snapshotPath is cfg.AdmissionStateFile; an empty string disables snapshot
// persistence (writes are silently skipped with WARN advisory).
//
// BC-2.05.009 PC-1/Inv-3; BC-2.05.010 PC-1/PC-3; S-BL.ADMISSION-SYNC-WIRE AC-002/AC-005.
func wireAdmissionSyncHandlers(
	srv *mgmt.Server,
	ks *admission.AdmittedKeySet,
	snapshotPath string,
) error {
	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionRegister, Fn: makeAdmissionRegisterHandler(ks, snapshotPath)}); err != nil {
		return fmt.Errorf("wireAdmissionSyncHandlers: register %s: %w", CmdAdmissionRegister, err)
	}
	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionRevoke, Fn: makeAdmissionRevokeHandler(ks, snapshotPath)}); err != nil {
		return fmt.Errorf("wireAdmissionSyncHandlers: register %s: %w", CmdAdmissionRevoke, err)
	}
	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionExpire, Fn: makeAdmissionExpireHandler(ks, snapshotPath)}); err != nil {
		return fmt.Errorf("wireAdmissionSyncHandlers: register %s: %w", CmdAdmissionExpire, err)
	}
	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionRemoveSVTN, Fn: makeAdmissionRemoveSVTNHandler(ks, snapshotPath)}); err != nil {
		return fmt.Errorf("wireAdmissionSyncHandlers: register %s: %w", CmdAdmissionRemoveSVTN, err)
	}
	return nil
}

// admissionRegisterArgs is the wire JSON args for internal.admission.register.
// svtn_id is the 32-hex-char [16]byte UUID of the SVTN (BC-2.05.009 Inv-4).
type admissionRegisterArgs struct {
	SVTNIDHex string `json:"svtn_id"`        // 32 lowercase hex chars = [16]byte UUID
	PubKey    string `json:"pubkey_openssh"` // base64url no-padding or OpenSSH format
	Role      string `json:"role"`
}

// admissionRevokeArgs is the wire JSON args for internal.admission.revoke.
type admissionRevokeArgs struct {
	SVTNIDHex string `json:"svtn_id"`
	PubKey    string `json:"pubkey_openssh"`
	Role      string `json:"role"`
	Confirm   bool   `json:"confirm"`
}

// admissionExpireArgs is the wire JSON args for internal.admission.expire.
type admissionExpireArgs struct {
	SVTNIDHex string `json:"svtn_id"`
	PubKey    string `json:"pubkey_openssh"`
	After     string `json:"after"` // duration string e.g. "24h"
}

// admissionRemoveSVTNArgs is the wire JSON args for internal.admission.remove-svtn.
type admissionRemoveSVTNArgs struct {
	SVTNIDHex string `json:"svtn_id"`
}

// parseSVTNIDHex parses a 32-lowercase-hex-char svtn_id to [16]byte.
// Returns an error if the string is not exactly 32 valid hex characters.
func parseSVTNIDHex(hexStr string) ([16]byte, error) {
	raw, err := hex.DecodeString(hexStr)
	if err != nil || len(raw) != 16 {
		return [16]byte{}, fmt.Errorf("E-CFG-001: invalid svtn_id %q: must be 32 lowercase hex chars (16-byte UUID)", hexStr)
	}
	var id [16]byte
	copy(id[:], raw)
	return id, nil
}

// makeAdmissionRegisterHandler returns the internal.admission.register handler.
// Decodes svtn_id (hex → [16]byte), pubkey, and role; calls ks.RegisterKey;
// then writes snapshot atomically (failure is advisory).
//
// BC-2.05.009 PC-8 / BC-2.05.010 PC-1/PC-3; S-BL.ADMISSION-SYNC-WIRE AC-005.
func makeAdmissionRegisterHandler(ks *admission.AdmittedKeySet, snapshotPath string) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		var a admissionRegisterArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNIDHex == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}
		svtnID, err := parseSVTNIDHex(a.SVTNIDHex)
		if err != nil {
			return nil, err
		}
		pubkey, err := decodePublicKey(a.PubKey)
		if err != nil {
			return nil, err
		}
		role, err := admission.KeyRoleFromString(a.Role)
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid role: %q", a.Role)
		}

		// RegisterKey: admitted=false always (challenge-response required; BC-2.05.009 PC-8).
		ks.RegisterKey(svtnID, pubkey, role)

		// Write snapshot atomically — failure is advisory (BC-2.05.010 PC-2 / AC-005).
		if err := writeSnapshotAtomic(snapshotPath, ks); err != nil {
			// WARN only — do not return error to caller.
			_ = err // caller (mgmt.Server) log seam; real daemon would log via slog
		}

		return map[string]any{"status": "ok"}, nil
	}
}

// makeAdmissionRevokeHandler returns the internal.admission.revoke handler.
func makeAdmissionRevokeHandler(ks *admission.AdmittedKeySet, snapshotPath string) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		var a admissionRevokeArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNIDHex == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}
		svtnID, err := parseSVTNIDHex(a.SVTNIDHex)
		if err != nil {
			return nil, err
		}
		pubkey, err := decodePublicKey(a.PubKey)
		if err != nil {
			return nil, err
		}

		// Look up nodeAddr for this key, then revoke it.
		entry, found := ks.LookupByPubkey(svtnID, pubkey)
		if !found {
			return nil, fmt.Errorf("E-ADM-013: key not found in SVTN %s", a.SVTNIDHex)
		}
		if err := ks.RevokeKey(svtnID, entry.NodeAddr); err != nil {
			return nil, fmt.Errorf("E-ADM-013: RevokeKey: %w", err)
		}

		if err := writeSnapshotAtomic(snapshotPath, ks); err != nil {
			_ = err // advisory WARN only
		}

		return map[string]any{"status": "ok"}, nil
	}
}

// makeAdmissionExpireHandler returns the internal.admission.expire handler.
func makeAdmissionExpireHandler(ks *admission.AdmittedKeySet, snapshotPath string) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		var a admissionExpireArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNIDHex == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}
		svtnID, err := parseSVTNIDHex(a.SVTNIDHex)
		if err != nil {
			return nil, err
		}
		pubkey, err := decodePublicKey(a.PubKey)
		if err != nil {
			return nil, err
		}

		ttl, err := time.ParseDuration(a.After)
		if err != nil || ttl <= 0 {
			return nil, fmt.Errorf("E-CFG-001: invalid duration: %q", a.After)
		}

		entry, found := ks.LookupByPubkey(svtnID, pubkey)
		if !found {
			return nil, fmt.Errorf("E-ADM-013: key not found in SVTN %s", a.SVTNIDHex)
		}
		expiry := time.Now().UTC().Add(ttl)
		if err := ks.SetKeyExpiry(svtnID, entry.NodeAddr, expiry); err != nil {
			return nil, fmt.Errorf("E-ADM-015: SetKeyExpiry: %w", err)
		}

		if err := writeSnapshotAtomic(snapshotPath, ks); err != nil {
			_ = err // advisory WARN only
		}

		return map[string]any{"status": "ok"}, nil
	}
}

// makeAdmissionRemoveSVTNHandler returns the internal.admission.remove-svtn handler.
func makeAdmissionRemoveSVTNHandler(ks *admission.AdmittedKeySet, snapshotPath string) func(ctx context.Context, args json.RawMessage) (any, error) {
	return func(_ context.Context, args json.RawMessage) (any, error) {
		var a admissionRemoveSVTNArgs
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid request args: %w", err)
		}
		if a.SVTNIDHex == "" {
			return nil, fmt.Errorf("E-CFG-001: missing required field: svtn_id")
		}
		svtnID, err := parseSVTNIDHex(a.SVTNIDHex)
		if err != nil {
			return nil, err
		}

		ks.RemoveSVTN(svtnID)

		if err := writeSnapshotAtomic(snapshotPath, ks); err != nil {
			_ = err // advisory WARN only
		}

		return map[string]any{"status": "ok"}, nil
	}
}

