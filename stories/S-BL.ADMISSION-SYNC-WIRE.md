---
artifact_id: S-BL.ADMISSION-SYNC-WIRE
document_type: story
level: ops
story_id: S-BL.ADMISSION-SYNC-WIRE
epic_id: E-7
title: "Admission-state sync wire: control pushes key mutations to routers via internal.admission.* RPC + VLR-local JSON snapshot"
status: draft
producer: story-writer
timestamp: 2026-07-15T00:00:00Z
modified:
  - date: 2026-07-15
    version: "1.0"
    change: >
      Initial full decomposition — admission-state-sync push RPC (BC-2.05.009) and
      VLR-local admitted-state snapshot (BC-2.05.010) per architect rulings
      decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md v1.1 (all 9 rulings, zero open
      human flags). Leaf prerequisite for S-BL.NODE-IDENTIFY-WIRE. 10 ACs, 8 points.
  - date: 2026-07-16
    version: "1.1"
    change: >
      Propagate rulings v1.2 svtn_id hex-[16]byte wire encoding fix (admissionSyncer
      interface svtnName→svtnID [16]byte; Decisions 2/5, AC-003/004/005);
      BC-2.05.009 ref 1.0→1.1; removed stale free-text input-hash citation from
      POL-005 note.
version: "1.1"
phase: 2
epic: E-7
wave: backlog
priority: P1
scope_phase: PE
points: 8
estimated_points: 8
inputs:
  - 'decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md'
  - 'specs/behavioral-contracts/ss-05/BC-2.05.009.md'
  - 'specs/behavioral-contracts/ss-05/BC-2.05.010.md'
  - 'specs/behavioral-contracts/ss-09/BC-2.09.003.md'
input-hash: "dc9752c"
traces_to: 'decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md'
behavioral_contracts:
  - BC-2.05.009
  - BC-2.05.010
  - BC-2.09.003
verification_properties: []
bc_traces:
  - BC-2.05.009
  - BC-2.05.010
  - BC-2.09.003
vp_traces: []
subsystems: [admission-security, deployment-operations]
target_module: "cmd/switchboard"
architecture_modules:
  - internal/config
  - internal/admission
  - internal/mgmt
  - cmd/switchboard
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []
blocks: [S-BL.NODE-IDENTIFY-WIRE]
rulings_doc: "decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md"
estimated_days: null
assumption_validations: []
risk_mitigations: []
acceptance_criteria_count: 10
inputDocuments:
  - 'decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md'   # v1.1 — BINDING. All 9 rulings: push RPC via internal/mgmt JSON-over-TCP; four internal.admission.* commands; control dials routers (TCP, dial-on-demand); retry-with-backoff (100ms/2x/10s/5); push failure advisory WARN no-rollback; RouterManagementEndpoints config field; admission_state_file config field; JSON snapshot schema_version:1; router TCP listener no loopback restriction (Ruling 9, ADR-012 is auth boundary); nil-syncer no-op; full-snapshot push on control startup.
  - 'specs/behavioral-contracts/ss-05/BC-2.05.009.md'  # v1.1 — admission-state-sync push RPC: four write paths, internal.admission.* commands, push-failure advisory (WARN no rollback), admitted=false on load, full-snapshot on control startup, nil-syncer no-op. Invariant 4 exempts svtn_id from admin-args encoding parity (it is the hex UUID, not the name).
  - 'specs/behavioral-contracts/ss-05/BC-2.05.010.md'  # v1.0 — VLR-local admitted-state snapshot: JSON schema_version:1 format, atomic write-on-receive, load-on-startup, fail-closed-on-corrupt (E-KEY-002), missing-file→empty-keyset, admitted=false invariant, no FrameAuthKey/NodeAddr/nonces stored.
  - 'specs/behavioral-contracts/ss-09/BC-2.09.003.md'  # v2.1 — PC-13 (admission_state_file: non-empty when present, E-CFG-015); PC-14 (router_management_endpoints: each addr host:port, E-CFG-016, NO loopback restriction per Ruling 9).
---

# S-BL.ADMISSION-SYNC-WIRE: Admission-State Sync Wire — Control Pushes Key Mutations to Routers via Internal RPC + VLR-Local JSON Snapshot

## Narrative

- **As a** router-mode daemon that must verify connecting nodes against an `AdmittedKeySet`
- **I want to** receive admission-state mutations from the control-mode daemon over the existing
  `internal/mgmt` JSON-over-TCP protocol, and persist them as a VLR-local JSON snapshot
- **So that** I can serve admission verification (`admission.AdmitNode`) against a real, populated
  keyset even after a restart or while the control daemon is temporarily unreachable — satisfying
  the HLR/VLR hard requirement for control-detachment resilience

## Context

`S-BL.DISCOVERY-WIRE-rulings.md` v1.10 Ruling 4 and `decisions/identity-cluster-architecture.md` v1.2
verified that the router-mode process's own `AdmittedKeySet` is always empty in production:
`admin.key.register` (the only production `RegisterKey` caller) runs exclusively in the
control-mode OS process, and there is no cross-process admission-sync mechanism anywhere in the
codebase. `admission.AdmitNode` called against the router's always-empty keyset therefore fails
unconditionally — `ErrNotAdmitted` — regardless of what the control daemon has registered.

This story closes that gap. It is the ROUTER-SIDE leg of the identity-cluster: the control daemon
learns to push its `AdmittedKeySet` mutations to each configured router management endpoint
via the existing `internal/mgmt` JSON-over-TCP protocol, and each router learns to persist those
mutations as a VLR-local JSON snapshot so it can recover after a restart without requiring control
to be reachable.

**Scope boundary.** This story does NOT implement the `NODE_IDENTIFY` wire opcode or
`Router.BindInterface` — those are `S-BL.NODE-IDENTIFY-WIRE`. It does NOT provision the access
node's own admission keypair — that is `S-BL.NODE-ADMISSION-PROVISIONING`. It delivers full
key-material sync (pubkey + role + revoked + expiry), not a SVTN-presence-only slice.

## Previous Story Intelligence (MANDATORY)

| Predecessor | Lesson carried forward |
|-------------|------------------------|
| `S-7.04-FU-DRAIN-WIRE` (DELIVERED PR #120 @ f73676d) | The register-before-serve invariant (F-P2L1-001): new handler-registration functions (`wireXHandlers`) are called from `runRouter` AFTER `newMgmtServer` and BEFORE `serveMgmtServer`. The `admission_sync_wire.go` registration function follows this identical pattern. |
| `S-W5.01` (merged PR #31) | `BuildAdminHandlers` already accepts `m *svtnmgmt.SVTNManager` and `ops *mgmt.OperatorKeySet`. Extending its signature with a `syncClient admissionSyncer` parameter (nil for non-control modes) follows the same established injection pattern. |
| `router_control_wire.go` | The file structure pattern for wiring mgmt handlers from `cmd/switchboard`: a top-level `wireX(srv *mgmt.Server, ...)` function + `mgmt.Handler` registrations. `admission_sync_wire.go` mirrors this shape. |
| `S-6.01` (config validation, merged PR #28) | The `validateHostPort` function already exists and is used for `upstream_routers[N].addr`. The `router_management_endpoints[N].addr` validation (E-CFG-016) reuses the same function and exhaustive-collection pattern. |
| BC-2.05.004 (admin.key.register etc, S-6.06 merged PR #36) | The four admin write handlers are in `admin_handlers.go` (`makeRegisterHandler`, `makeRevokeHandler`, `makeExpireHandler`, `makeAdminSVTNDestroyHandler`). The push call is inserted AFTER the successful `SVTNManager.*` write and BEFORE constructing the response — consistent with BC-2.05.009's "success then push" ordering. |

## Adjudicated Design Decisions

Transcribed from `decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md` v1.1 (binding — all 9 rulings).

### Decision 1 — Push protocol: reuse internal/mgmt JSON-over-TCP (Ruling 1)

The push from control to each router uses the existing `internal/mgmt` JSON-over-TCP protocol
(Ed25519 challenge-response on connect, ADR-012; newline-delimited JSON; per-RPC
`{"type":"request","id":"...","command":"...","args":{...}}` envelope). No new protocol is
invented. Control authenticates to the router using its own `daemonPriv` Ed25519 key. The
router must have control's daemon public key in its `authorized_operator_keys` config (operator
configuration requirement, not enforced by this story's code).

### Decision 2 — Four push commands (Ruling 1)

| Event | Command | Args |
|-------|---------|------|
| `RegisterKey` | `internal.admission.register` | `{svtn_id, pubkey_openssh, role}` |
| `RevokeKey` | `internal.admission.revoke` | `{svtn_id, pubkey_openssh, role, confirm}` |
| `SetKeyExpiry` | `internal.admission.expire` | `{svtn_id, pubkey_openssh, after}` |
| `RemoveSVTN` | `internal.admission.remove-svtn` | `{svtn_id}` |

The `internal.` prefix distinguishes these from operator-facing `admin.` commands. The router
registers these four; control/console/access never register them (ADR-004/AC-004 intact).
Args encoding matches existing `admin.*` handler conventions (same `pubkey_openssh`,
`role`, `after`, `confirm` conventions).

> **Wire encoding note (rulings v1.2):** The wire `svtn_id` in all four commands is the
> **32-lowercase-hex-char encoding of the `[16]byte` SVTN UUID** (matching the snapshot
> schema's `svtns[].svtn_id` field), not the human-readable SVTN name. Control-side admin
> handlers resolve name→`[16]byte` via `m.SVTNByName(name).ID` before constructing the push
> call. BC-2.05.009 Invariant 4 exempts `svtn_id` from admin-args encoding parity.

### Decision 3 — Push failure is advisory; no rollback (Ruling 2)

Push failure (connection refused, timeout, auth failure, retry exhaustion) is logged at `WARN`
level. The control-side write remains committed. The `sbctl` caller receives a success response
reflecting the authoritative control-side state. The router is temporarily stale; it resynchronises on the next control startup (via `PushFullSnapshot`) or the next per-write push.

### Decision 4 — Retry-with-backoff (Ruling 2)

Each push attempt to a given endpoint: initial delay 100ms, multiplier 2, maximum delay 10s,
maximum attempts 5. After retry exhaustion, log WARN and skip. The next push event re-attempts
from scratch.

### Decision 5 — admissionSyncer interface; nil is no-op (Ruling 2)

```go
type admissionSyncer interface {
    PushRegisterKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole) error
    PushRevokeKey(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, role admission.KeyRole, confirm bool) error
    PushSetKeyExpiry(ctx context.Context, svtnID [16]byte, pubkey ed25519.PublicKey, ttl time.Duration) error
    PushRemoveSVTN(ctx context.Context, svtnID [16]byte) error
}
```

A nil `admissionSyncer` passed to `BuildAdminHandlers` skips the push silently (no error, no
panic). Only control mode passes a non-nil `*admissionSyncClient`. Router/console/access modes
pass nil — unchanged from today. Control-side admin handlers resolve name→`[16]byte` via
`m.SVTNByName(name).ID` at the call site before calling `Push*`.

### Decision 6 — Snapshot format: JSON, schema_version:1 (Ruling 3)

```json
{
  "schema_version": 1,
  "timestamp": "<RFC3339 UTC>",
  "svtns": [
    {
      "svtn_id": "<32 hex chars = 16-byte UUID>",
      "keys": [
        {
          "pubkey": "<base64url no-padding, 32-byte raw Ed25519 key>",
          "role": "<control|console|access>",
          "revoked": false,
          "expiry": "<RFC3339 UTC, omitempty>"
        }
      ]
    }
  ]
}
```

The snapshot does NOT store: `admitted` (always `false` on load), `FrameAuthKey` (derived on
load by `RegisterKey`), `NodeAddr` (derived on load), nonces (ephemeral). Atomic write:
write to `<admission_state_file>.tmp`, then `os.Rename`. Forward-compat gate: unrecognised
`schema_version` → fail-closed (treat as corrupt).

### Decision 7 — Router startup load semantics (Ruling 3)

- `admission_state_file` absent in config → start with empty keyset (existing behaviour).
- File configured but absent on disk → empty keyset + INFO log `"admission_state_file not found; starting with empty keyset — awaiting push from control"`.
- File present and valid (`schema_version: 1`) → call `ks.RegisterKey` / `ks.RevokeKey` / `ks.SetKeyExpiry` for each entry; loaded entries have `admitted=false`. INFO log with count per SVTN.
- File present but corrupt or unknown `schema_version` → fail-closed: `runRouter` returns non-nil error, daemon refuses to start (E-KEY-002).

### Decision 8 — Router TCP management listener: no loopback restriction (Ruling 9)

The router may bind its management listener to any `host:port` (including `0.0.0.0:PORT` and
non-loopback addresses). The ADR-012 challenge-response handshake is the authentication
boundary; network-level access restriction is the operator's firewall-policy responsibility.
The `isMgmtLoopbackHost` guard from `buildMgmtListener` is NOT applied to router-mode TCP
management endpoints. On startup, an INFO log is emitted naming the bound address:
`"router management listener bound to %s (ensure firewall policy restricts access as appropriate)"`.

### Decision 9 — Config fields (Rulings 2 and 3)

**Control-mode:** `RouterManagementEndpoints []RouterManagementEndpoint` / YAML `router_management_endpoints` in `config.Config`. Each entry: `Addr string \`yaml:"addr"\``. Structurally identical to `UpstreamRouters`. Validation: each `Addr` validated by `validateHostPort` (E-CFG-016). No loopback restriction. SIGHUP reload updates the client's endpoint list.

**Router-mode:** `AdmissionStateFile string` / YAML `admission_state_file`. Optional; absent means no persistence. Validation in `Config.Validate()`: present-and-whitespace-only → E-CFG-015 (no file I/O). Both fields follow the existing PC-10/PC-11 shape in `BC-2.09.003 v2.1` PC-12/PC-13/PC-14.

### Decision 10 — Full-snapshot push on control startup (Ruling 2)

After `SVTNManager` is initialised in `runControl`, before the management server begins serving,
`admissionSyncClient.PushFullSnapshot(ctx)` iterates all `(svtn, pubkey, role)` triples in the
current `AdmittedKeySet` (via `ListBySVTN` across all SVTNs) and issues
`internal.admission.register` (plus `internal.admission.expire` for entries with non-zero expiry)
to each configured router endpoint.

### Decision 11 — ARCH-08 compliance

All new code lives in `cmd/switchboard` (position 18, the top). No new `internal/` package.
New files: `cmd/switchboard/admission_sync_client.go` (control-side client) and
`cmd/switchboard/admission_sync_wire.go` (router-side handler registration). Both import only
packages already imported by `mgmt_wire.go` (`internal/admission`, `internal/mgmt`,
`internal/config`). No new ARCH-08 position registration needed.

## Acceptance Criteria

### AC-001 — Config.Validate() validates admission_state_file (whitespace rejected E-CFG-015; absent accepted) and router_management_endpoints (each addr host:port, E-CFG-016; no loopback restriction; empty list accepted) (BC-2.09.003 v2.1 PC-13 and PC-14)

**BC Anchor:** BC-2.09.003 v2.1 Postconditions 13 and 14.

**Postconditions:**
1. When `admission_state_file` is absent or empty string, `Config.Validate()` accepts it.
2. When `admission_state_file` is present with a whitespace-only value, `Config.Validate()`
   returns an error containing E-CFG-015:
   `"config error: admission_state_file: must not be empty. Fix: set to a valid writable file path, e.g. '/var/lib/switchboard/admission-state.json', or remove the field to start with an empty keyset"`.
3. When `router_management_endpoints` is absent or an empty list, `Config.Validate()` accepts it.
4. When `router_management_endpoints` contains entries, each entry's `addr` is validated as a
   valid `host:port` via `validateHostPort`. An invalid `addr` returns E-CFG-016:
   `"config error: router_management_endpoints[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' or '<hostname>:<port>' format, e.g. '10.0.0.2:9093'"`.
5. When `router_management_endpoints` has `addr: "0.0.0.0:9093"` or any non-loopback address,
   `Config.Validate()` accepts it without error (NO loopback restriction — Ruling 9).
6. Exhaustive error collection: all `router_management_endpoints` addr errors (multiple invalid
   entries) are collected before returning.
7. `Config.Validate()` performs no file I/O for either field.

**Test names:**
- `TestConfig_Validate_AdmissionStateFile_AbsentAccepted`
- `TestConfig_Validate_AdmissionStateFile_WhitespaceOnlyRejectsE_CFG_015`
- `TestConfig_Validate_RouterManagementEndpoints_EmptyListAccepted`
- `TestConfig_Validate_RouterManagementEndpoints_InvalidAddrRejectsE_CFG_016`
- `TestConfig_Validate_RouterManagementEndpoints_NonLoopbackAccepted`
- `TestConfig_Validate_RouterManagementEndpoints_MultipleInvalidExhaustiveErrors`

---

### AC-002 — The four internal.admission.* push commands are registered on router-mode management server; control/console/access modes do not register them (BC-2.05.009 PC-1 / ADR-004 / AC-004 role-exclusion)

**BC Anchor:** BC-2.05.009 Postconditions 1 and 3; BC-2.05.009 Invariant 3.

**Postconditions:**
1. `wireAdmissionSyncHandlers(srv *mgmt.Server, ks *admission.AdmittedKeySet, snapshotPath string)`
   is called from `runRouter` AFTER `newMgmtServer` and BEFORE `serveMgmtServer`
   (register-before-serve invariant F-P2L1-001).
2. After `wireAdmissionSyncHandlers`, the server's handler table contains exactly these four
   commands: `internal.admission.register`, `internal.admission.revoke`,
   `internal.admission.expire`, `internal.admission.remove-svtn`.
3. The router-mode server does NOT register any `admin.key.*` or `admin.svtn.*` handlers
   (ADR-004/AC-004 role-exclusion unchanged).
4. Control mode, console mode, and access mode do NOT call `wireAdmissionSyncHandlers`.

**Test names:**
- `TestWireAdmissionSyncHandlers_RegisteredOnRouterServer`
- `TestWireAdmissionSyncHandlers_NotRegisteredOnControlServer`
- `TestRouterMode_AdminHandlersNotRegistered` (or verify existing AC-004 test still holds)

---

### AC-003 — admin.key.register on control pushes internal.admission.register to configured routers; push failure does not roll back control write (BC-2.05.009 PC-1 and PC-2)

**BC Anchor:** BC-2.05.009 Postconditions 1 and 2.

**Postconditions:**
1. After a successful `admin.key.register` call on the control-mode daemon (which writes to
   `SVTNManager`/`AdmittedKeySet`), the `admissionSyncClient.PushRegisterKey` is called with
   `(svtnID, pubkey, role)` — where `svtnID` is the `[16]byte` UUID resolved from the name via
   `m.SVTNByName(name).ID` — before the handler returns its response to the caller.
2. If the push to a configured router endpoint fails (connection refused, retry exhaustion),
   the control-side write remains committed (not rolled back). A WARN log is emitted with
   endpoint address and error. The handler response to `sbctl` reflects success.
3. A nil `admissionSyncer` (e.g., in a unit test not exercising push) results in a no-op with
   no error and no panic.

**Test names:**
- `TestAdmissionSync_RegisterKey_PushCalledAfterControlWrite`
- `TestAdmissionSync_RegisterKey_PushFailureDoesNotRollbackControlWrite`
- `TestAdmissionSync_NilSyncer_NoOp`

---

### AC-004 — admin.key.revoke, admin.key.expire, admin.svtn.destroy on control push corresponding internal.admission.* commands; push failure is advisory (BC-2.05.009 PC-1 and PC-2)

**BC Anchor:** BC-2.05.009 Postconditions 1 and 2.

**Postconditions:**
1. `admin.key.revoke` → `admissionSyncer.PushRevokeKey(ctx, svtnID, pubkey, role, confirm)` called after successful control write; `svtnID` is `[16]byte` resolved via `m.SVTNByName(name).ID`.
2. `admin.key.expire` → `admissionSyncer.PushSetKeyExpiry(ctx, svtnID, pubkey, ttl)` called after successful control write; `svtnID` resolved same way.
3. `admin.svtn.destroy` → `admissionSyncer.PushRemoveSVTN(ctx, svtnID)` called after successful control write; `svtnID` resolved same way.
4. For each: push failure is advisory (WARN log, no rollback, handler returns success to caller).
5. For each: nil `admissionSyncer` is a no-op with no error.

**Test names:**
- `TestAdmissionSync_RevokeKey_PushCalledAfterControlWrite`
- `TestAdmissionSync_ExpireKey_PushCalledAfterControlWrite`
- `TestAdmissionSync_RemoveSVTN_PushCalledAfterControlWrite`
- `TestAdmissionSync_PushFailure_AllWritePaths_Advisory`

---

### AC-005 — Router internal.admission.register handler populates AdmittedKeySet with admitted=false; snapshot written atomically after each push (BC-2.05.009 PC-8 / BC-2.05.010 PC-1 and PC-3)

**BC Anchor:** BC-2.05.009 Postcondition 8; BC-2.05.010 Postconditions 1 and 3.

**Postconditions:**
1. When the router receives `internal.admission.register` with `{svtn_id, pubkey_openssh, role}`,
   it hex-decodes the 32-lowercase-hex-char `svtn_id` to `[16]byte` and calls
   `ks.RegisterKey(svtnID, pubkey, role)`. The resulting keyset entry has `admitted=false`.
2. After the successful `RegisterKey` call, the VLR-local snapshot is written atomically to the
   configured `admission_state_file` path: write serialised JSON to `<path>.tmp`, then `os.Rename`.
3. If the write fails (disk full, permission denied), a WARN is logged. The push handler returns
   success (the in-memory keyset is up to date). The snapshot file remains from the previous write.
4. The snapshot written after `RegisterKey` contains the new entry in `svtns[].keys[]` with
   correct `pubkey`, `role`, `revoked: false`, and no `expiry` (unless expiry was set).

**Test names:**
- `TestRouterAdmissionHandler_Register_AdmittedFalse`
- `TestRouterAdmissionHandler_Register_SnapshotWritten`
- `TestRouterAdmissionHandler_Register_SnapshotWriteFailure_Advisory`

---

### AC-006 — Snapshot JSON round-trip: schema_version:1, correct field encoding, FrameAuthKey/NodeAddr/nonces not stored (BC-2.05.010 PC-4 and PC-5)

**BC Anchor:** BC-2.05.010 Postconditions 4 and 5.

**Postconditions:**
1. The serialized snapshot JSON contains exactly the fields specified in Decision 6's schema:
   `schema_version: 1`, `timestamp` (RFC3339 UTC), `svtns[].svtn_id` (32 lowercase hex chars),
   `svtns[].keys[].pubkey` (base64url no-padding, 32-byte raw Ed25519 key),
   `svtns[].keys[].role`, `svtns[].keys[].revoked`, and `svtns[].keys[].expiry` (omitempty).
2. `FrameAuthKey`, `NodeAddr`, `admitted` flag, and nonce map are NOT present in the snapshot.
3. A round-trip test: serialize a known `AdmittedKeySet` → write to file → deserialize →
   call `RegisterKey` for each entry → `ListBySVTN` returns the same entries; all have
   `admitted=false`.
4. Revoked entries: `revoked: true` in snapshot → `RevokeKey` called after `RegisterKey` on load.
5. Entries with expiry: `expiry` field in snapshot → `SetKeyExpiry` called after `RegisterKey`
   on load.

**Test names:**
- `TestSnapshot_JSON_FieldEncoding_CorrectSchema`
- `TestSnapshot_RoundTrip_EntriesMatch`
- `TestSnapshot_RoundTrip_AdmittedAlwaysFalse`
- `TestSnapshot_RoundTrip_RevokedEntryCallsRevokeKey`
- `TestSnapshot_RoundTrip_ExpiryEntryCallsSetKeyExpiry`
- `TestSnapshot_NoFrameAuthKey_NoNodeAddr_NoNonces`

---

### AC-007 — Router startup: file absent → empty keyset + INFO log; file valid → load entries; file corrupt → fail-closed E-KEY-002 (BC-2.05.010 PC-6 through PC-9)

**BC Anchor:** BC-2.05.010 Postconditions 6, 7, 8, 9.

**Postconditions:**
1. If `admission_state_file` is not configured (empty string), router starts with empty keyset,
   no snapshot I/O performed.
2. If `admission_state_file` is configured but the file does not exist, router starts with empty
   keyset and logs INFO: `"admission_state_file not found; starting with empty keyset — awaiting push from control"`.
3. If the file is present and valid (`schema_version: 1`), entries are loaded via `RegisterKey`
   (+ `RevokeKey` / `SetKeyExpiry` as needed); INFO log with count per SVTN; all loaded entries
   have `admitted=false`.
4. If the file is present but contains invalid JSON, or `schema_version` is not 1, `runRouter`
   returns a non-nil error (E-KEY-002) and the daemon refuses to start. The error message
   includes the file path and parse reason.
5. Loaded entries have `admitted=false` — the challenge-response handshake is required to flip
   `admitted=true` (this story does not implement the handshake; that is `S-BL.NODE-IDENTIFY-WIRE`).

**Test names:**
- `TestRouterStartup_AdmissionStateFile_NotConfigured_EmptyKeyset`
- `TestRouterStartup_AdmissionStateFile_ConfiguredFileAbsent_EmptyKeyset_InfoLog`
- `TestRouterStartup_AdmissionStateFile_ValidFile_EntriesLoaded`
- `TestRouterStartup_AdmissionStateFile_CorruptJSON_FailClosed_EKEY002`
- `TestRouterStartup_AdmissionStateFile_UnknownSchemaVersion_FailClosed`
- `TestRouterStartup_LoadedEntries_AdmittedFalse`

---

### AC-008 — Router TCP management listener accepts non-loopback bind; startup INFO log of bind address (BC-2.09.003 v2.1 PC-14 / Ruling 9)

**BC Anchor:** BC-2.09.003 v2.1 Postcondition 14; rulings v1.1 Ruling 9.

**Postconditions:**
1. A `router_management_endpoints` entry with `addr: "0.0.0.0:9093"` (non-loopback) is accepted
   by `Config.Validate()` without error (Ruling 9 — no `isMgmtLoopbackHost` guard applied).
2. When the router's management listener binds to a non-loopback address, a startup INFO log is
   emitted: `"router management listener bound to <addr> (ensure firewall policy restricts access as appropriate)"`.
3. The INFO log is inspectable in integration tests (the bound address is visible in log output
   before the management server begins accepting connections).

**Test names:**
- `TestRouterMgmtListener_NonLoopbackBindAccepted`
- `TestRouterMgmtListener_StartupInfoLog_BindAddress`

---

### AC-009 — Full-snapshot push on control startup: PushFullSnapshot pushes all AdmittedKeySet entries to configured routers (BC-2.05.009 PC-7)

**BC Anchor:** BC-2.05.009 Postcondition 7.

**Postconditions:**
1. When `runControl` starts and `SVTNManager` is initialised, `admissionSyncClient.PushFullSnapshot(ctx)`
   is called before the management server begins serving.
2. `PushFullSnapshot` iterates all `(svtn, pubkey, role)` triples in the current `AdmittedKeySet`
   (via `ListBySVTN` across all SVTNs) and issues `internal.admission.register` to each configured
   router endpoint for each entry.
3. For entries with non-zero expiry, `internal.admission.expire` is also issued.
4. After `PushFullSnapshot`, the router's keyset matches the control daemon's current `AdmittedKeySet`
   state (modulo push failures, which are logged at WARN and do not block startup).

**Test names:**
- `TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter` (integration: two in-process mgmt.Server instances)
- `TestAdmissionSync_PushFullSnapshot_ExpiryPushed`
- `TestAdmissionSync_PushFullSnapshot_EmptyKeysetNoPushAttempt`

---

### AC-010 — SIGHUP reload updates RouterManagementEndpoints on admissionSyncClient; in-flight pushes not interrupted (BC-2.05.009 Invariant 5)

**BC Anchor:** BC-2.05.009 Invariant 5.

**Postconditions:**
1. When `runControl` receives SIGHUP and reloads config, the `admissionSyncClient`'s endpoint
   list is updated from the new config's `RouterManagementEndpoints`.
2. Any push in progress at the time of the SIGHUP is not interrupted. The new endpoint list
   takes effect for the next push event.
3. If the new config has an empty `RouterManagementEndpoints` list, the client's list is updated
   to empty (no further pushes attempted until another SIGHUP restores endpoints).

**Test names:**
- `TestAdmissionSync_SIGHUPReload_EndpointListUpdated`
- `TestAdmissionSync_SIGHUPReload_NewListUsedOnNextPush`

---

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| admissionSyncer interface | cmd/switchboard/admission_sync_client.go | pure-core |
| admissionSyncClient | cmd/switchboard/admission_sync_client.go | effectful-shell |
| wireAdmissionSyncHandlers | cmd/switchboard/admission_sync_wire.go | effectful-shell |
| Snapshot serialization | cmd/switchboard/admission_sync_snapshot.go | pure-core |
| Config validation (E-CFG-015/016) | internal/config/config.go | pure-core |
| runRouter startup load | cmd/switchboard/router.go | effectful-shell |
| runControl PushFullSnapshot | cmd/switchboard/control.go | effectful-shell |

## Non-Goals

- **NODE_IDENTIFY wire opcode / Router.BindInterface** — that is `S-BL.NODE-IDENTIFY-WIRE`.
- **Node-side admission keypair provisioning** — that is `S-BL.NODE-ADMISSION-PROVISIONING`.
- **HLR-side admission state** — the HLR/VLR architecture described in `identity-cluster-architecture.md`
  §8 is a forward architecture. This story delivers only the near-term VLR-local snapshot; the
  HLR replication protocol is a future follow-on.
- **Bulk `router.admission.full-snapshot` RPC** — the near-term startup snapshot is implemented
  as a loop of individual `internal.admission.register` calls (Ruling 1). A bulk endpoint is a
  follow-on optimization.
- **Unix socket fallback** — when `RouterManagementEndpoints` is empty AND a co-located router
  socket exists, an optional Unix socket fallback path is described in Ruling 2 but is NOT
  required for the near-term story. If not implemented, document as a follow-on.
- **Discovery multicast group join** — while `S-BL.DISCOVERY-WIRE` Ruling 4 noted that
  `wireDiscoveryListener` needs to know which SVTNs the router serves, this story's `AdmittedKeySet`
  population is the prerequisite that makes such an enumeration possible; the actual multicast
  group join wiring is gated on `S-BL.DISCOVERY-WIRE` AC-001's Forward Obligation (e) and is
  NOT part of this story's scope.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `RouterManagementEndpoints` empty list | No push attempted; control writes succeed locally; `admissionSyncer` nil or no-op. |
| EC-002 | Router endpoint unreachable (connection refused) | Retry-with-backoff up to 5 attempts; WARN log; push skipped for this event. Router is stale until next control startup. |
| EC-003 | Router responds with auth failure | Treated as push failure; WARN logged; no rollback; operator must add control's daemon pubkey to router's `authorized_operator_keys`. |
| EC-004 | Rapid burst of `admin.key.register` (N writes in quick succession) | Each write triggers a separate per-write push; N TCP connections opened. No batching in near-term. |
| EC-005 | `admissionSyncer` is nil (router/console/access mode) | Push skipped silently; no error; existing handler behavior unchanged. |
| EC-006 | `internal.admission.register` sent for SVTN router has no record of | Router's `RegisterKey` creates the entry idempotently. |
| EC-007 | Control restarts after prior push failures | `PushFullSnapshot(ctx)` on startup pushes full current `AdmittedKeySet` to all routers, correcting any staleness. |
| EC-008 | Router restarts while control is detached; snapshot present | Router loads snapshot, serves existing admitted keys without control. HARD REQUIREMENT (b) satisfied. |
| EC-009 | Snapshot write fails (disk full) after successful push | WARN logged; push handler returns success; in-memory state correct; snapshot may be stale from last successful write. |
| EC-010 | `router_management_endpoints: [{addr: "0.0.0.0:9093"}]` | Validate() accepts; non-loopback bind used; startup INFO log of bound address. |
| EC-011 | Snapshot `schema_version: 999` on router startup | E-KEY-002; `runRouter` returns error; daemon exits 1. |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| `admissionSyncer` interface | pure-core | Defines behaviour only; no I/O |
| `admissionSyncClient` | effectful-shell | Dials TCP, sends JSON RPCs, retries |
| `wireAdmissionSyncHandlers` | effectful-shell | Registers handlers that mutate keyset and write snapshot to disk |
| Snapshot marshal/unmarshal | pure-core | JSON encode/decode with no I/O side-effects |
| `Config.Validate()` extensions | pure-core | String validation; no file I/O |
| `runRouter` snapshot load | effectful-shell | Reads file from disk on startup |
| `runControl` PushFullSnapshot | effectful-shell | Iterates keyset, dials routers, sends RPCs |

## File-Change List

| File | Change | Justification |
|------|--------|---------------|
| `internal/config/config.go` | Add `RouterManagementEndpoints []RouterManagementEndpoint` + `RouterManagementEndpoint` type + YAML tags; add `AdmissionStateFile string` + YAML tag; extend `Config.Validate()` with E-CFG-015 (admission_state_file whitespace) and E-CFG-016 (router_management_endpoints addr host:port, exhaustive, no loopback restriction) | BC-2.09.003 v2.1 PC-13 and PC-14 |
| `cmd/switchboard/admission_sync_client.go` (new) | `admissionSyncer` interface; `admissionSyncClient` type holding endpoints + daemonPriv; `PushRegisterKey`, `PushRevokeKey`, `PushSetKeyExpiry`, `PushRemoveSVTN` methods (dial-on-demand, retry-with-backoff); `PushFullSnapshot`; SIGHUP endpoint-list update | BC-2.05.009 Rulings 1–2 |
| `cmd/switchboard/admission_sync_wire.go` (new) | `wireAdmissionSyncHandlers(srv *mgmt.Server, ks *admission.AdmittedKeySet, snapshotPath string)` + four `mgmt.Handler` registrations for `internal.admission.*`; per-handler snapshot write after successful keyset update | BC-2.05.009 Ruling 3; BC-2.05.010 |
| `cmd/switchboard/admission_sync_snapshot.go` (new, or inline in wire.go) | Snapshot serialization/deserialization (JSON marshal/unmarshal to/from `admission.AdmittedKeySet` state); atomic write; load-on-startup logic | BC-2.05.010 |
| `cmd/switchboard/admin_handlers.go` | Extend `BuildAdminHandlers` signature with `syncClient admissionSyncer` parameter; add push call after each successful `SVTNManager.*` write in `makeRegisterHandler`, `makeRevokeHandler`, `makeExpireHandler`, `makeAdminSVTNDestroyHandler` | BC-2.05.009 PC-1/PC-2 |
| `cmd/switchboard/router.go` (or equivalent `runRouter`) | Add `wireAdmissionSyncHandlers(...)` call (after `newMgmtServer`, before `serveMgmtServer`); add snapshot load-on-startup from `cfg.AdmissionStateFile`; emit non-loopback bind INFO log | BC-2.05.010 PC-6/7/8/9; Ruling 9 |
| `cmd/switchboard/control.go` (or equivalent `runControl`) | Construct `admissionSyncClient`; pass to `BuildAdminHandlers`; call `PushFullSnapshot(ctx)` on startup; update client endpoints on SIGHUP | BC-2.05.009 PC-7; Invariant 5 |
| `internal/config/config_test.go` | Extended table-driven tests for E-CFG-015 and E-CFG-016 (AC-001) | AC-001 |
| `cmd/switchboard/admission_sync_test.go` (new) | Integration tests for push RPCs, snapshot round-trips, fail-closed load, startup snapshot push (AC-002 through AC-010) | All ACs |

## Token Budget Estimate

| Component | Estimate |
|-----------|---------|
| `internal/config` field + validate extension | ~80 tokens |
| `cmd/switchboard/admission_sync_client.go` (syncer interface + client + retry) | ~250 tokens |
| `cmd/switchboard/admission_sync_wire.go` (four handler registrations + snapshot write) | ~200 tokens |
| Snapshot serialization/deserialization | ~150 tokens |
| `admin_handlers.go` push wiring (4 sites) | ~80 tokens |
| `runRouter` / `runControl` wiring | ~100 tokens |
| Tests (10 ACs, ~25 test functions, including two-mgmt-server integration tests) | ~700 tokens |
| **Overall** | ~1,560 tokens — this story is 8-point scope; the token budget is consistent with that |

## Tasks (MANDATORY)

1. [ ] Write failing tests for AC-001 (config validation E-CFG-015 / E-CFG-016) — test-writer
2. [ ] Write failing tests for AC-002 (handler registration on router, not on control) — test-writer
3. [ ] Write failing tests for AC-003/AC-004 (push called after control write; nil no-op) — test-writer
4. [ ] Write failing tests for AC-005 (router handler populates keyset + snapshot write) — test-writer
5. [ ] Write failing tests for AC-006 (snapshot JSON round-trip) — test-writer
6. [ ] Write failing tests for AC-007 (startup load semantics: absent / valid / corrupt) — test-writer
7. [ ] Write failing tests for AC-008 (non-loopback bind; startup INFO log) — test-writer
8. [ ] Write failing tests for AC-009 (PushFullSnapshot on control startup) — test-writer
9. [ ] Write failing tests for AC-010 (SIGHUP endpoint-list update) — test-writer
10. [ ] Verify Red Gate: `go test ./...` fails with compile or test failures for all ACs
11. [ ] Implement `internal/config` fields + `Config.Validate()` extensions — implementer
12. [ ] Implement `admission_sync_client.go` (interface + retry client + PushFullSnapshot) — implementer
13. [ ] Implement `admission_sync_wire.go` (four handler registrations) — implementer
14. [ ] Implement snapshot serialization/deserialization — implementer
15. [ ] Wire push calls into `admin_handlers.go` — implementer
16. [ ] Wire `runRouter` startup load + non-loopback bind log — implementer
17. [ ] Wire `runControl` client construction + PushFullSnapshot + SIGHUP reload — implementer
18. [ ] Run `go test ./... -race`; confirm all AC tests pass
19. [ ] Update STATE.md

## Architecture Compliance Rules

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| ARCH-08 §Import DAG | All new code in `cmd/switchboard` (position 18); no new `internal/` package | Compile-time; `go list -deps` test recommended |
| F-P2L1-001 register-before-serve | `wireAdmissionSyncHandlers` called after `newMgmtServer`, before `serveMgmtServer` | Verified by AC-002 test `TestWireAdmissionSyncHandlers_RegisteredOnRouterServer` |
| ADR-004 / AC-004 role-exclusion | `internal.admission.*` handlers registered on router only; not on control/console/access | Verified by AC-002 |
| ADR-012 / BC-2.05.009 Invariant 3 | `internal.admission.*` are internal-RPC only; never operator-facing `admin.*` verbs | Naming convention; verified by code review |
| DI-002 | Push RPCs carry public keys only (`pubkey_openssh`); private keys never transmitted | BC-2.05.009 Invariant 2; verified by code review |
| BC-2.05.010 Invariant 1 | Snapshot write is atomic (temp-file + rename idiom) | Verified by AC-005 test |
| Ruling 9 | No `isMgmtLoopbackHost` guard on router-mode TCP management endpoints | Verified by AC-008 test `TestRouterMgmtListener_NonLoopbackBindAccepted` |
| BC-2.05.010 Invariant 2 | Loaded entries have `admitted=false` | Verified by AC-007 test `TestRouterStartup_LoadedEntries_AdmittedFalse` |

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 (per `go.mod`) | Language runtime — all new files are Go |
| `crypto/ed25519` | stdlib | Ed25519 key types used in `admissionSyncer` interface |
| `encoding/json` | stdlib | Snapshot serialization/deserialization |
| `time` | stdlib | Retry backoff delays; RFC3339 timestamp encoding |
| `os` | stdlib | Atomic snapshot write (`os.Rename`) and file I/O on startup |
| `net` | stdlib | TCP dial-on-demand to router management endpoints |
| `internal/admission` | project-local | `AdmittedKeySet`, `KeyRole`, `AdmitNode` |
| `internal/mgmt` | project-local | `mgmt.Server`, `mgmt.Handler`, JSON-over-TCP protocol |
| `internal/config` | project-local | `Config`, `validateHostPort` |

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `internal/config/config.go` | modify | Add `RouterManagementEndpoints`, `RouterManagementEndpoint`, `AdmissionStateFile`; extend `Validate()` with E-CFG-015 / E-CFG-016 |
| `internal/config/config_test.go` | modify | Table-driven tests for AC-001 |
| `cmd/switchboard/admission_sync_client.go` | create | `admissionSyncer` interface; `admissionSyncClient` with retry-with-backoff; `PushFullSnapshot`; SIGHUP endpoint-list update |
| `cmd/switchboard/admission_sync_wire.go` | create | `wireAdmissionSyncHandlers`; four `internal.admission.*` handler registrations; per-handler snapshot write |
| `cmd/switchboard/admission_sync_snapshot.go` | create | Snapshot JSON marshal/unmarshal; atomic write; load-on-startup logic |
| `cmd/switchboard/admin_handlers.go` | modify | Extend `BuildAdminHandlers` with `syncClient admissionSyncer`; add push calls in four write handlers |
| `cmd/switchboard/router.go` | modify | Call `wireAdmissionSyncHandlers`; snapshot load on startup; non-loopback bind INFO log |
| `cmd/switchboard/control.go` | modify | Construct `admissionSyncClient`; pass to `BuildAdminHandlers`; call `PushFullSnapshot`; SIGHUP reload |
| `cmd/switchboard/admission_sync_test.go` | create | Integration tests for AC-002 through AC-010 |

## POL-005 Delivery Plan Note

This story is a leaf prerequisite in the identity-cluster (`depends_on: []`). It does not
depend on `S-BL.NODE-ADMISSION-PROVISIONING`. Implementations targeting `S-BL.NODE-IDENTIFY-WIRE`
should deliver both this story and `S-BL.NODE-ADMISSION-PROVISIONING` before scheduling
`S-BL.NODE-IDENTIFY-WIRE`.

TDD discipline: the Red Gate (failing tests) must be established before any implementation
code is written. Two-mgmt-server integration tests (AC-003, AC-005, AC-009) require building
in-process `mgmt.Server` instances using the existing `startMgmtServer` helper + `net.Pipe()`
or a loopback TCP listener — the same infrastructure `mgmt_wire_test.go` establishes.

Run `compute-input-hash <artifact> --check` before beginning implementation to verify inputs
have not changed.

## Provenance

- **Origin:** `S-BL.DISCOVERY-WIRE-rulings.md` v1.10 Ruling 4 (Forward Obligation (e)) and
  `decisions/identity-cluster-architecture.md` v1.2 (three-leg cluster design, §3 and §9).
- **Rulings:** `decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md` v1.1 (all 9 rulings, zero open
  human flags — fully decomposition-ready as of 2026-07-15).
- **Unblocks:** `S-BL.NODE-IDENTIFY-WIRE`'s Open Design Obligation 5 (admission.AdmitNode
  is verification-only against the router's always-empty keyset — this story populates that keyset).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-07-16 | Propagate rulings v1.2 svtn_id hex-[16]byte wire encoding fix (admissionSyncer interface svtnName→svtnID [16]byte; Decisions 2/5, AC-003/004/005); BC-2.05.009 ref 1.0→1.1; removed stale free-text input-hash citation from POL-005 note. |
| 1.0 | 2026-07-15 | Initial full decomposition — 10 ACs, 8 points, leaf prerequisite with `depends_on: []`. Admission-state sync push RPC (BC-2.05.009) + VLR-local JSON snapshot (BC-2.05.010). Per rulings v1.1 (Option A + Ruling 9 router TCP listener). |
