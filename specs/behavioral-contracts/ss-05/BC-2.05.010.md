---
artifact_id: BC-2.05.010
document_type: behavioral-contract
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-07-15T00:00:00Z
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
input-hash: "27b8999"
extracted_from: null
bc_id: BC-2.05.010
subsystem: admission-security
architecture_module: cmd/switchboard
capability: CAP-019
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-07-15
    version: "1.0"
    change: >
      Initial draft — admission-state-snapshot (VLR-local): JSON schema_version:1,
      write-on-receive atomic temp+rename, load-on-startup-if-valid, fail-closed-on-corrupt,
      missing→empty keyset. Does NOT store admitted/FrameAuthKey/NodeAddr (all derived).
      Authored per S-BL.ADMISSION-SYNC-WIRE BC groundwork list item A2
      (S-BL.ADMISSION-SYNC-WIRE-rulings.md §7 Ruling 3, §6).
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
traces_to: [CAP-019]
kos_anchors:
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.05.010: Admission-State Snapshot — Router VLR-Local Persistence (Write-on-Receive, Load-on-Startup, Fail-Closed-on-Corrupt)

## Description

The router-mode daemon maintains a VLR-local durable snapshot of its `AdmittedKeySet`
state so that it can survive restarts and control-node detachment (HLR/VLR hard requirement,
constraint (b), `identity-cluster-architecture.md` §8). After each successful push-handler
invocation (`internal.admission.*`), the router writes an atomic JSON snapshot to the
configured `admission_state_file` path. On startup, if the file is present and valid, the
router loads it and populates its `AdmittedKeySet` from the snapshot, enabling it to serve
admission queries without requiring control to be reachable. Missing files start with an
empty keyset; corrupt files cause a fail-closed startup refusal.

This BC covers the snapshot format, write semantics, load semantics, and error handling.
It does NOT cover the push-handler wire protocol (BC-2.05.009) or the config-field
validation for `admission_state_file` (BC-2.09.003 v3.0 amendment, consolidated item A4).

## Preconditions

**Write-on-receive preconditions:**
1. The router's `internal.admission.*` push-handler has just successfully applied a write
   to its `AdmittedKeySet` (via `RegisterKey`, `RevokeKey`, `SetKeyExpiry`, or `RemoveSVTN`).
2. `admission_state_file` is non-empty in the router's config.

**Load-on-startup preconditions:**
1. The router daemon is starting (`runRouter` entry).
2. `admission_state_file` is non-empty in the router's config.
3. The `AdmittedKeySet` has been constructed and is empty (before management server starts).

## Postconditions

### Write-on-receive postconditions

1. **Atomic write:** The snapshot is serialized to JSON and written atomically:
   `os.WriteFile` to `<admission_state_file>.tmp` (or equivalent temp path in the same
   directory), then `os.Rename` to `admission_state_file`. This prevents a partial write
   leaving a corrupt snapshot file.

2. **Write failure is advisory:** If the atomic write fails (disk full, permission denied),
   the failure is logged at WARN level. The push handler returns success regardless — the
   in-memory `AdmittedKeySet` is already updated and correct. The snapshot file remains from
   the previous write (stale but not corrupt). No rollback of the in-memory write.

3. **Snapshot reflects post-write state:** After a `RegisterKey` push, the snapshot contains
   the new entry. After a `RemoveSVTN` push, the snapshot no longer contains entries for
   that SVTN. The snapshot is a full serialization of the router's current `AdmittedKeySet`
   state (not an incremental/delta log).

### Snapshot JSON schema

4. The snapshot file format is JSON with the following schema (`schema_version: 1`):

   ```json
   {
     "schema_version": 1,
     "timestamp": "<RFC3339 UTC>",
     "svtns": [
       {
         "svtn_id": "<32 hex chars = 16-byte UUID>",
         "keys": [
           {
             "pubkey": "<base64url no-padding, 32-byte Ed25519 raw key>",
             "role": "<control|console|access>",
             "revoked": false,
             "expiry": "<RFC3339 UTC, omitempty>"
           }
         ]
       }
     ]
   }
   ```

   Field semantics:
   - `schema_version` (int): currently 1. Forward-compatibility gate: if a future snapshot
     has an unrecognised `schema_version`, the router treats it as corrupt (fail-closed).
   - `timestamp` (RFC3339 UTC string): write time; informational only.
   - `svtns[].svtn_id` (string): 32 hex chars (lowercase) encoding the `[16]byte` SVTN UUID.
   - `svtns[].keys[].pubkey` (string): base64url no-padding encoding of the raw 32-byte
     Ed25519 public key.
   - `svtns[].keys[].role` (string): one of `"control"`, `"console"`, `"access"`.
   - `svtns[].keys[].revoked` (bool): true if the key has been revoked.
   - `svtns[].keys[].expiry` (string, omitempty): RFC3339 UTC; absent if no expiry is set.

5. **What the snapshot does NOT include:**
   - `admitted` boolean: loaded entries are always `admitted=false`; live admission state
     is a per-connection runtime event, not persisted.
   - `FrameAuthKey`: derived deterministically from `(svtnID, pubkey)` via
     `hmac.DeriveDiscoveryKey` / `routing.DeriveDiscoveryKey` — NOT stored. `RegisterKey`
     recomputes it on load, same as on a fresh push.
   - `NodeAddr`: derived deterministically from `(svtnID, pubkey)` via
     `frame.DeriveNodeAddress` — NOT stored; recomputed on load.
   - Nonces (replay-prevention map): ephemeral per-connection state; not persisted. Nonce
     map starts empty on every startup (the router restart invalidates outstanding nonces,
     which is correct — nodes must re-identify).

### Load-on-startup postconditions

6. **File absent → empty keyset:** If `admission_state_file` is configured but the file does
   not exist, the router starts with an empty `AdmittedKeySet` and logs INFO:
   `"admission_state_file not found; starting with empty keyset — awaiting push from control"`.
   This is the correct fresh-install behavior.

7. **File present and valid → load entries:** The file is read, JSON-decoded, and
   `schema_version` is validated (must be 1). For each `svtns[].keys[]` entry:
   - `ks.RegisterKey(svtnID, pubkey, role)` is called.
   - If `revoked==true`, `ks.RevokeKey(svtnID, nodeAddr)` is called after registration.
   - If `expiry` is non-zero, `ks.SetKeyExpiry(svtnID, nodeAddr, expiry)` is called after
     registration.
   An INFO log is emitted with the count of loaded entries per SVTN.

8. **Loaded entries are `admitted=false`:** Nodes loaded from the snapshot have
   `admitted=false`. They must complete the challenge-response handshake
   (`S-BL.NODE-IDENTIFY-WIRE`) to become `admitted=true` before the router forwards
   their frames.

9. **Fail-closed on corrupt file:** If the file exists but fails to parse (invalid JSON,
   unrecognised `schema_version`, missing required fields), `runRouter` returns a non-nil
   error and the daemon refuses to start. A FATAL-level log is emitted with the file path
   and parse error. The operator must either delete the corrupt file (fresh-start semantics)
   or restore from backup.

## Invariants

1. **Atomic write prevents corruption:** The temp-file + rename idiom ensures that the
   snapshot is either the previous complete snapshot or the new complete snapshot — never
   a partially-written file.
2. **Loaded entries start `admitted=false`:** The `admitted` flag reflects a live
   challenge-response handshake result, not a persisted state. The snapshot explicitly
   excludes it; `RegisterKey` initialises new entries with `admitted=false`.
3. **`FrameAuthKey` and `NodeAddr` are derived, not stored:** Storing derived material
   would create a stale-derivation risk if the derivation function changes. Computing them
   from `(svtnID, pubkey)` at load time is equivalent to a fresh `RegisterKey` call.
4. **Fail-closed on corrupt:** A corrupt snapshot is more dangerous than a missing snapshot
   (a missing snapshot starts fresh cleanly; a corrupt one may represent partial state or
   a malformed attack). Fail-closed on corrupt is the correct posture.
5. **Missing file is NOT an error:** A missing `admission_state_file` is the normal
   fresh-install state. The router starts with an empty keyset, which is correct — it has
   never received any admitted keys yet.
6. **schema_version gate:** A future snapshot with `schema_version > 1` is treated as
   corrupt (fail-closed) until the router is updated to support the new version. This is
   an explicit forward-compatibility gate, not a silent ignore.

## Trigger

- Router push-handler (`internal.admission.*`) completes a successful `AdmittedKeySet` write
  (write-on-receive path).
- `runRouter` startup, after `AdmittedKeySet` construction (load-on-startup path).

## Error Codes

| Code | Condition | Severity | Exit Code | Notes |
|------|-----------|----------|-----------|-------|
| E-KEY-002 | Snapshot file exists but fails to parse (invalid JSON, unknown `schema_version`, missing required fields) | broken | 1 | `"admission_state_file: <path>: <reason>"` — emitted at FATAL level; daemon refuses to start |
| (WARN log) | Snapshot write fails after successful push handler | degraded | — (daemon continues) | Advisory only; in-memory state is correct; snapshot remains from previous write |

> **Note on E-KEY-002:** Uses the KEY error family introduced in BC-2.09.004 (E-KEY-001 for
> admission keypair load failure). E-KEY-002 covers snapshot parse failure. Both are
> daemon-startup failures, distinct from config-validation failures (E-CFG-*).

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `admission_state_file` absent in config | Router starts with empty keyset; no snapshot written or read. Existing behavior unchanged. |
| EC-002 | `admission_state_file` configured; file absent | INFO log "admission_state_file not found; starting with empty keyset"; router starts normally (fresh-install). |
| EC-003 | `admission_state_file` configured; file present with `schema_version: 1`; valid entries | Entries loaded; `admitted=false` for all; INFO log with entry count; router starts. |
| EC-004 | `admission_state_file` configured; file present with `schema_version: 999` | Fail-closed: E-KEY-002; daemon exits 1. Operator must delete file. |
| EC-005 | `admission_state_file` configured; file contains invalid JSON (truncated) | Fail-closed: E-KEY-002; daemon exits 1. |
| EC-006 | `admission_state_file` configured; file has `revoked: true` for an entry | `RegisterKey` called first, then `RevokeKey` after registration. Key is in keyset as revoked. |
| EC-007 | `admission_state_file` configured; file has `expiry` set for an entry | `RegisterKey` called first, then `SetKeyExpiry` after registration. Key has non-zero expiry. |
| EC-008 | Write fails (disk full) after successful push | WARN logged; push handler returns success; in-memory state correct; snapshot may be stale from last successful write. |
| EC-009 | Router restarts while control is detached; snapshot present and valid | Router loads snapshot, serves existing admitted keys. Control's absence does not delay or block router startup. HARD REQUIREMENT (b) satisfied. |
| EC-010 | Control reconnects after detachment; sends `PushFullSnapshot` | Router handler applies all register/revoke/expire pushes; writes updated snapshot after each push. Router keyset is current. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Snapshot write: `RegisterKey(svtn, pubkey, role)` on `AdmittedKeySet`; serialize to JSON | JSON contains correct `svtn_id`, `pubkey` (base64url), `role`, `revoked: false`, no `expiry` | happy-path |
| Snapshot round-trip: serialize → write to temp file → read → deserialize → `RegisterKey` for each entry | `ListBySVTN` returns same entries as before serialization; `admitted=false` for all | happy-path |
| Load with `revoked: true` entry | `RevokeKey` called post-load; `IsAdmitted` returns false | edge-case |
| Load with non-zero `expiry` entry | `SetKeyExpiry` called post-load; expiry set correctly | edge-case |
| File absent | Empty keyset; INFO log; router starts | missing-file |
| File has `schema_version: 999` | E-KEY-002; `runRouter` returns error; daemon exits 1 | fail-closed |
| File has invalid JSON | E-KEY-002; `runRouter` returns error; daemon exits 1 | fail-closed |
| Loaded entry — `admitted` field | Not stored in snapshot; loaded entry has `admitted=false` regardless | invariant |
| Loaded entry — `FrameAuthKey` | Not stored in snapshot; derived on load by `RegisterKey` | invariant |
| Config without `admission_state_file` | Router starts with empty keyset; no snapshot I/O | config-absent |

## Verification Properties

| VP-NNN | Property | Proof Method | Notes |
|--------|----------|-------------|-------|
| test-as-evidence | Snapshot round-trip: serialize + write + load + verify keyset contents match | unit | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Loaded entries have `admitted=false`; no `FrameAuthKey` / `NodeAddr` in snapshot | unit | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Corrupt file (invalid JSON, unknown `schema_version`) → fail-closed; `runRouter` returns error | unit | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Missing file → empty keyset; INFO log; router starts normally | unit | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Atomic write: temp-file + rename; no partial-write corruption | unit (inject write failure at rename) | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | `revoked: true` entry → `RevokeKey` called post-load | unit | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | `expiry` present → `SetKeyExpiry` called post-load | unit | S-BL.ADMISSION-SYNC-WIRE AC |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — snapshot preserves only admission eligibility, not live connection state) |
| Architecture Module | cmd/switchboard (admission_sync_wire.go, runRouter) |
| Stories | S-BL.ADMISSION-SYNC-WIRE (all postconditions and ACs) |
| Capability Anchor Justification | CAP-019 ("Key lifecycle management") — this BC specifies the durable VLR-local cache that makes key lifecycle operations (register, revoke, expire) survive router restarts and control detachment. Without this snapshot, registered keys are lost on every router restart. |

## Related BCs

- BC-2.05.009 — depends on: this BC is the persistence layer for the push-handler writes specified by BC-2.05.009
- BC-2.09.003 — amendment: `admission_state_file` config field validation consolidated into BC-2.09.003 v3.0 via groundwork item A4
- identity-cluster-architecture.md §7–§8 — HLR/VLR motivation: this snapshot IS the VLR-local durable state; its format is designed to be forward-compatible with a future replication protocol

## Architecture Anchors

- decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md Ruling 3 (VLR-local snapshot: JSON format, atomic write, load-on-startup, fail-closed)
- decisions/identity-cluster-architecture.md §7 (near-term ADMISSION-SYNC-WIRE scope: router-side VLR-local snapshot IN SCOPE for constraint (b) HARD REQUIREMENT)

## Story Anchor

S-BL.ADMISSION-SYNC-WIRE — all postconditions in this BC trace to acceptance criteria for this story.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-15 | Initial draft — VLR-local admitted-state snapshot: JSON schema_version:1 format, atomic write-on-receive, load-on-startup, fail-closed-on-corrupt (E-KEY-002), missing-file→empty-keyset, `admitted=false` invariant, no `FrameAuthKey`/`NodeAddr`/nonces stored. Authored per S-BL.ADMISSION-SYNC-WIRE BC groundwork item A2. |
