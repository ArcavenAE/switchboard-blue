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
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/arcavenae/switchboard/internal/admission"
	"github.com/arcavenae/switchboard/internal/mgmt"
)

// routerPersister serialises concurrent persist calls from the four router-side
// push handlers (register, revoke, expire, remove-svtn) via a shared mutex.
//
// F-P6-01 fix (BC-2.05.010 Invariant 1): without this mutex, two concurrent
// push RPCs (e.g., register + revoke arriving simultaneously) each call
// writeSnapshotAtomic independently. A register goroutine that reads the keyset
// BEFORE a concurrent revoke mutation can rename a stale snapshot AFTER the revoke
// snapshot — resurrecting a revoked key on disk (same lost-update race as F-4A
// on the control side, now fixed on the router side).
//
// Fix: hold mu across the entire {ks.write + marshal + write-temp + rename}
// sequence so rename order matches keyset-read order, which matches mutation order.
// The control side uses controlPersister (admission_sync_snapshot.go) — this
// mirrors that pattern for the router side.
type routerPersister struct {
	path string
	mu   sync.Mutex
	w    io.Writer // log writer for WARN messages (nil-safe)
}

// persist writes the current keyset state to disk under the persist mutex.
// The mutex serialises {snapshot-read, marshal, write, rename} across all four
// router push handlers so rename order == mutation order (F-P6-01 fix).
// Write failure is advisory WARN (BC-2.05.010 PC-2/EC-008 / F-3 fix).
// Safe to call with a nil receiver (no-op when no persistence configured).
func (p *routerPersister) persist(ks *admission.AdmittedKeySet) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := writeSnapshotAtomic(p.path, ks); err != nil {
		// Advisory: WARN only (BC-2.05.010 PC-2/EC-008 / F-3 fix).
		if p.w != nil {
			_, _ = fmt.Fprintf(p.w, "switchboard router: WARN: admission snapshot write failed: path=%s err=%v\n",
				p.path, err)
		}
	}
}

// wireAdmissionSyncHandlers registers the four internal.admission.* push
// command handlers on srv. Must be called after newMgmtServer and before
// serveMgmtServer (register-before-serve invariant F-P2L1-001).
//
// ks is the router's AdmittedKeySet — the same instance passed to buildRouter.
// snapshotPath is cfg.AdmissionStateFile; an empty string disables snapshot
// persistence (writes are silently skipped with WARN advisory).
// w is the log writer for advisory WARN messages (nil-safe — no-op when nil).
// BC-2.05.010 PC-2/EC-008 mandates WARN log on snapshot-write failure (F-3 fix).
//
// A single routerPersister (with its shared mutex) is constructed here and closed
// over by all four handlers — this serialises concurrent snapshot writes from
// simultaneous RPCs (F-P6-01 fix / BC-2.05.010 Invariant 1).
//
// BC-2.05.009 PC-1/Inv-3; BC-2.05.010 PC-1/PC-3; S-BL.ADMISSION-SYNC-WIRE AC-002/AC-005.
func wireAdmissionSyncHandlers(
	srv *mgmt.Server,
	ks *admission.AdmittedKeySet,
	snapshotPath string,
	w io.Writer,
) error {
	// F-P6-01 fix: shared routerPersister serialises concurrent snapshot writes
	// from all four handlers (BC-2.05.010 Invariant 1). A nil persister (empty
	// snapshotPath) is a no-op — same semantics as before.
	var rp *routerPersister
	if snapshotPath != "" {
		rp = &routerPersister{path: snapshotPath, w: w}
	}

	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionRegister, Fn: makeAdmissionRegisterHandler(ks, rp)}); err != nil {
		return fmt.Errorf("wireAdmissionSyncHandlers: register %s: %w", CmdAdmissionRegister, err)
	}
	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionRevoke, Fn: makeAdmissionRevokeHandler(ks, rp)}); err != nil {
		return fmt.Errorf("wireAdmissionSyncHandlers: register %s: %w", CmdAdmissionRevoke, err)
	}
	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionExpire, Fn: makeAdmissionExpireHandler(ks, rp)}); err != nil {
		return fmt.Errorf("wireAdmissionSyncHandlers: register %s: %w", CmdAdmissionExpire, err)
	}
	if err := srv.Register(mgmt.Handler{Command: CmdAdmissionRemoveSVTN, Fn: makeAdmissionRemoveSVTNHandler(ks, rp)}); err != nil {
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
// then persists via rp (serialised under rp's mutex — F-P6-01 fix).
//
// rp is the shared routerPersister (nil-safe — no-op when nil/no persistence).
// BC-2.05.010 PC-2/EC-008: snapshot-write failure is logged at WARN (F-3 fix).
//
// BC-2.05.009 PC-8 / BC-2.05.010 PC-1/PC-3; S-BL.ADMISSION-SYNC-WIRE AC-005.
func makeAdmissionRegisterHandler(ks *admission.AdmittedKeySet, rp *routerPersister) func(ctx context.Context, args json.RawMessage) (any, error) {
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

		// Persist under the shared mutex (F-P6-01 fix / BC-2.05.010 Invariant 1).
		// Failure is advisory WARN (BC-2.05.010 PC-2/EC-008 / F-3 fix).
		rp.persist(ks)

		return map[string]any{"status": "ok"}, nil
	}
}

// makeAdmissionRevokeHandler returns the internal.admission.revoke handler.
// rp is the shared routerPersister (nil-safe) for serialised snapshot writes (F-P6-01).
//
// Key-not-found handling (Ruling 13 / BC-2.05.009 v1.4 PC-7c): when
// PushFullSnapshot issues internal.admission.revoke for a revoked entry on a fresh
// router that has never seen the key, the revoke arrives for an absent key. This
// handler treats "key not found" as SUCCESS (return nil) — absent is the correct
// non-admissible terminal state. Only genuine keyset errors (not ErrKeyNotRegistered)
// are returned as errors.
func makeAdmissionRevokeHandler(ks *admission.AdmittedKeySet, rp *routerPersister) func(ctx context.Context, args json.RawMessage) (any, error) {
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
		// Key-not-found is treated as success (Ruling 13 / BC-2.05.009 v1.4 PC-7c):
		// if the key is absent from the router's keyset, the router is already in the
		// correct non-admissible terminal state — no action required, return success.
		entry, found := ks.LookupByPubkey(svtnID, pubkey)
		if !found {
			// Absent key = correct non-admissible terminal state.
			// Ruling 13: PushFullSnapshot sends revoke-only for revoked entries;
			// on a fresh router the revoke arrives for an absent key — this is success.
			return map[string]any{"status": "ok"}, nil
		}
		if err := ks.RevokeKey(svtnID, entry.NodeAddr); err != nil {
			// ErrKeyNotRegistered should have been caught above via LookupByPubkey,
			// but handle it defensively — treat as success (absent = correct state).
			if errors.Is(err, admission.ErrKeyNotRegistered) {
				return map[string]any{"status": "ok"}, nil
			}
			return nil, fmt.Errorf("E-ADM-013: RevokeKey: %w", err)
		}

		// Persist under the shared mutex (F-P6-01 fix / BC-2.05.010 Invariant 1).
		rp.persist(ks)

		return map[string]any{"status": "ok"}, nil
	}
}

// makeAdmissionExpireHandler returns the internal.admission.expire handler.
// rp is the shared routerPersister (nil-safe) for serialised snapshot writes (F-P6-01).
func makeAdmissionExpireHandler(ks *admission.AdmittedKeySet, rp *routerPersister) func(ctx context.Context, args json.RawMessage) (any, error) {
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
		if err != nil {
			return nil, fmt.Errorf("E-CFG-001: invalid duration: %q", a.After)
		}
		// ttl == 0 is rejected (ambiguous — use a non-zero value).
		// ttl < 0 is accepted: a negative duration means the expiry is in the past
		// (BC-2.05.009 v1.4 PC-7 / Invariant 6 / EC-010: PushFullSnapshot propagates
		// original expiry regardless of past/future; router marks already-past entries
		// expired).
		if ttl == 0 {
			return nil, fmt.Errorf("E-CFG-001: invalid duration: %q (zero duration not allowed)", a.After)
		}

		entry, found := ks.LookupByPubkey(svtnID, pubkey)
		if !found {
			return nil, fmt.Errorf("E-ADM-013: key not found in SVTN %s", a.SVTNIDHex)
		}
		expiry := time.Now().UTC().Add(ttl)
		if err := ks.SetKeyExpiry(svtnID, entry.NodeAddr, expiry); err != nil {
			return nil, fmt.Errorf("E-ADM-015: SetKeyExpiry: %w", err)
		}

		// Persist under the shared mutex (F-P6-01 fix / BC-2.05.010 Invariant 1).
		rp.persist(ks)

		return map[string]any{"status": "ok"}, nil
	}
}

// makeAdmissionRemoveSVTNHandler returns the internal.admission.remove-svtn handler.
// rp is the shared routerPersister (nil-safe) for serialised snapshot writes (F-P6-01).
func makeAdmissionRemoveSVTNHandler(ks *admission.AdmittedKeySet, rp *routerPersister) func(ctx context.Context, args json.RawMessage) (any, error) {
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

		// Persist under the shared mutex (F-P6-01 fix / BC-2.05.010 Invariant 1).
		rp.persist(ks)

		return map[string]any{"status": "ok"}, nil
	}
}
