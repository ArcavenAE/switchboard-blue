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
  - date: 2026-07-17
    version: "1.2"
    change: >
      Propagate rulings v1.3 Ruling 10 (F-2 fix): router mgmt listener auto-detects
      TCP-vs-unix on management_socket; AC-008 postconditions rewritten (5 PCs incl.
      real TCP-bind + push-handshake assertions) + 2 new test names
      (TCPBind_ConnectionSucceeds, TCPBind_PushHandshakeSucceeds); implementer task
      for mgmtNetwork/buildMgmtListener auto-detect. rulings ref v1.2→v1.3.
  - date: 2026-07-17
    version: "1.3"
    change: >
      Propagate rulings v1.4 (Rulings 11+12) + BC-2.09.003 v2.2 + BC-2.05.009 v1.2:
      add AC-011 (control-side keyset persistence via control_admission_state_file —
      F-P3-01, now in-scope) + AC-012 (mgmt loopback guard scope: control/access
      loopback-only TCP, router-only exemption — F-P3-02); amend AC-009 (load-then-push);
      remove control-persistence deferral from Non-Goals; points 8→11; AC count 10→12;
      note F-P3-03 shutdown-drain bound as impl task.
  - date: 2026-07-17
    version: "1.4"
    change: >
      Propagate BC-2.05.009 v1.3: AC-009 now requires PushFullSnapshot semantic-equivalence
      (revoked→revoked, past-expiry→expired, not register-only) + 2 new test names
      (RevokedKeyStaysRevoked, PastExpiryStaysExpired) — fixes F-1 revocation-un-propagation
      (adversary pass 5); add impl tasks for F-3 (WARN log on advisory failures) + F-4
      (bind-log for console/access modes).
  - date: 2026-07-17
    version: "1.5"
    change: >
      Propagate Ruling 13 (F-P6-02) + BC-2.05.009 v1.4: AC-009 PC-3c amended — MUST NOT
      issue internal.admission.register for revoked entries; revoke-only RPC (router treats
      key-not-found as success — absent = correct non-admissible terminal state).
      Register+revoke two-RPC pattern PROHIBITED for revoked entries (partial failure leaves
      key active on fresh router, violating Invariant 6). Strengthen
      TestAdmissionSync_PushFullSnapshot_RevokedKeyStaysRevoked with BOTH precondition cases
      (fresh router → key ABSENT; existing-entry router → key REVOKED). Add
      TestAdmissionSync_PushFullSnapshot_RevokedKey_RegisterNotSent (regression guard). Add
      impl task 17d (skip-register for revoked entries in PushFullSnapshot + router revoke
      handler treats key-not-found as success) and 17e (F-P6-01 concurrent snapshot-write
      mutex fix — no spec change). BC-2.05.009 ref v1.3→v1.4; rulings ref v1.4→v1.5.
      AC count 12, points 11 unchanged.
  - date: 2026-07-18
    version: "1.6"
    change: >
      Propagate Ruling 14 (F-P7-01) + BC-2.05.009 v1.5: AC-009 PC-3b amended — add
      compensating best-effort internal.admission.revoke when expire fails AND expiry is
      already in the past, so router ends non-admissible (traces to BC-2.05.009 v1.5 PC-7b).
      Future-expiry expire-fail is PC-5 stale/missing (permitted) — no compensating action.
      Add TestAdmissionSync_PushFullSnapshot_PastExpiry_ExpireFails_CompensatingRevoke (required)
      with assertions: PushRevokeKey called after failed PushSetKeyExpiry for past-expiry entry;
      key ends non-admissible on router. Add impl task 17f (PushFullSnapshot past-expiry
      partial-failure handling + register-fail continue). Add impl task 17g (LOW: routerPersister
      nil-writer fallback to os.Stderr). BC-2.05.009 ref v1.4→v1.5; rulings ref v1.5→v1.6.
      AC count 12, points 11 unchanged.
  - date: 2026-07-18
    version: "1.7"
    change: >
      Ruling 15 propagation (F-P8-01 + F-P8-02): add AC-013 (PushFullSnapshot multi-endpoint
      per-endpoint sequencing — each reachable endpoint independently reaches correct terminal
      state regardless of other endpoints' reachability; traces to BC-2.05.009 v1.6 PC-7
      per-endpoint sequencing obligation + Invariant 6 / Ruling 15 / F-P8-01). Two new test
      names: TestAdmissionSync_PushFullSnapshot_MultiEndpoint_LastUnreachable_PastExpiry_
      ReachableEndpointNonAdmissible; TestAdmissionSync_PushFullSnapshot_MultiEndpoint_
      FirstUnreachable_ReachableEndpointCorrect. Add impl task 17h (pushSnapshotToEndpoint
      helper + PushFullSnapshot outer-loop refactor; Ruling 15 / F-P8-01). Add impl task 17i
      (runControlWithKey test seam + TestControlAdmission_RunControl_LoadThenPush_E2E amendment;
      Ruling 15 / F-P8-02). Add F-P8-02 note (TestControlAdmission_RunControl_LoadThenPush_E2E)
      to AC-011 test names. Update BC-2.05.009 ref v1.5→v1.6; rulings ref v1.6→v1.7.
      AC count 12→13; points 11→12.
version: "1.7"
phase: 2
epic: E-7
wave: backlog
priority: P1
scope_phase: PE
points: 12
estimated_points: 12
inputs:
  - 'decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md'
  - 'specs/behavioral-contracts/ss-05/BC-2.05.009.md'
  - 'specs/behavioral-contracts/ss-05/BC-2.05.010.md'
  - 'specs/behavioral-contracts/ss-09/BC-2.09.003.md'
input-hash: "096ff0f"
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
acceptance_criteria_count: 13
inputDocuments:
  - 'decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md'   # v1.7 — BINDING. All 15 rulings: push RPC via internal/mgmt JSON-over-TCP; four internal.admission.* commands; control dials routers (TCP, dial-on-demand); retry-with-backoff (100ms/2x/10s/5); push failure advisory WARN no-rollback; RouterManagementEndpoints config field; admission_state_file config field; JSON snapshot schema_version:1; router TCP listener no loopback restriction (Ruling 9, ADR-012 is auth boundary); mgmt listener auto-detects TCP-vs-unix on management_socket via validateHostPort (Ruling 10, F-2 fix); nil-syncer no-op; full-snapshot push on control startup; control_admission_state_file persistence field + synchronous write-on-mutation + load-on-startup BEFORE push (Ruling 11, F-P3-01, now IN-SCOPE); buildMgmtListener loopback guard extends to console/control/access modes, router-only exemption (Ruling 12, F-P3-02); PushFullSnapshot MUST NOT issue register for revoked entries — revoke-only RPC, router treats key-not-found as success, register+revoke two-RPC pattern PROHIBITED (Ruling 13, F-P6-02); PushFullSnapshot past-expiry partial-failure — compensating best-effort revoke when expire fails AND expiry is past (Ruling 14, F-P7-01); AdmitNode does NOT check expiry (only ReAuthenticate does); PushFullSnapshot per-endpoint sequencing — endpoints outer/entries inner, new pushSnapshotToEndpoint helper, delta-push paths unchanged (Ruling 15, F-P8-01); runControlWithKey test seam for load→push ordering guard (Ruling 15, F-P8-02).
  - 'specs/behavioral-contracts/ss-05/BC-2.05.009.md'  # v1.6 — admission-state-sync push RPC: four write paths, internal.admission.* commands, push-failure advisory (WARN no rollback), admitted=false on load, full-snapshot on control startup, nil-syncer no-op. Invariant 4 exempts svtn_id from admin-args encoding parity. PC-7b (v1.5, unchanged): register+expire pair for active entries with non-zero expiry; if expire fails AND expiry is past → best-effort compensating revoke (advisory-continue); if expire fails for future-expiry → no compensating action (PC-5 stale/missing). PC-7c (v1.4, unchanged): MUST NOT issue internal.admission.register for revoked entries — skip register entirely; issue internal.admission.revoke ONLY (router treats key-not-found as success). Invariant 6 (v1.5, unchanged): PC-5 vs Invariant 6 distinction + future-vs-past-expiry carve-out + AdmitNode no-expiry-check evidence. EC-009 (v1.4, unchanged): skip-register semantics. EC-010 (v1.5, unchanged): compensating revoke fires on past-expiry expire-fail; router ends non-admissible. Per-endpoint sequencing obligation (v1.6): added to PC-7 — register→expire→compensate state machine MUST be applied per configured router endpoint independently (Ruling 15 / F-P8-01).
  - 'specs/behavioral-contracts/ss-05/BC-2.05.010.md'  # v1.0 — VLR-local admitted-state snapshot: JSON schema_version:1 format, atomic write-on-receive, load-on-startup, fail-closed-on-corrupt (E-KEY-002), missing-file→empty-keyset, admitted=false invariant, no FrameAuthKey/NodeAddr/nonces stored.
  - 'specs/behavioral-contracts/ss-09/BC-2.09.003.md'  # v2.2 — PC-13 (admission_state_file: non-empty when present, E-CFG-015); PC-14 (router_management_endpoints: each addr host:port, E-CFG-016, NO loopback restriction per Ruling 9); PC-15 (control_admission_state_file: non-empty when present, E-CFG-017, control-mode only); PC-11b (mgmt loopback guard: console/control/access must bind loopback TCP, router exempt).
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

Transcribed from `decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md` v1.7 (binding — all 15 rulings).

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

### Decision 8 — Router mgmt listener auto-detects TCP-vs-unix on management_socket (Rulings 9 + 10)

The router's `mgmtNetwork`/`buildMgmtListener` logic must auto-detect transport from the
`management_socket` config value:

- If `validateHostPort(management_socket)` returns nil (value parses as `host:port` via
  `net.SplitHostPort` + numeric port check) → bind TCP on that address, WITHOUT applying the
  `isMgmtLoopbackHost` guard (Ruling 9: no loopback restriction for router).
- Otherwise (absent, empty, or a filesystem path) → bind unix socket (default preserved; zero
  regression to existing router deployments).

This resolves adversary finding F-2 (rulings v1.3): `mgmtNetwork` previously hardcoded
router→unix while the push client (`admissionSyncClient`) always dials TCP, causing every push
to fail with connection-refused. The auto-detect makes TCP management endpoints functional in
production. The ADR-012 challenge-response handshake remains the authentication boundary;
network-level restriction is the operator's firewall responsibility. On startup, when TCP is
selected, an INFO log is emitted: `"router management listener bound to <addr> (ensure firewall
policy restricts access as appropriate)"`.

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

### AC-008 — Router mgmt listener auto-detects TCP-vs-unix on management_socket; non-loopback bind accepted; real TCP client can connect and push (BC-2.09.003 v2.1 PC-14 / Rulings 9 + 10)

**BC Anchor:** BC-2.09.003 v2.1 Postcondition 14; rulings v1.3 Ruling 9 + Ruling 10.

**Postconditions:**
1. `Config.Validate()` accepts a non-loopback `router_management_endpoints` addr without error
   (Ruling 9 — no `isMgmtLoopbackHost` guard). [existing PC-1, keep]
2. When a router's `management_socket` is set to a `host:port`, `runRouter` opens a TCP
   listener on that address — verified by `net.DialTimeout("tcp", addr, ...)` succeeding after
   startup.
3. A real `admissionSyncClient` pointing at the router's TCP management address completes the
   ADR-012 handshake and pushes an `internal.admission.register` RPC that the router's
   `AdmittedKeySet` receives.
4. Startup INFO log `"router management listener bound to <addr> (ensure firewall policy restricts access as appropriate)"` is emitted (the bound address is inspectable before the mgmt server accepts connections). [existing, keep]
5. When `management_socket` is absent or a filesystem path, the router binds a unix socket
   (default preserved — auto-detect returns unix on non-host:port values).

**Test names:**
- `TestRouterMgmtListener_NonLoopbackBindAccepted` (PC-1, existing)
- `TestRouterMgmtListener_StartupInfoLog_BindAddress` (PC-4, existing)
- `TestRouterMgmtListener_TCPBind_ConnectionSucceeds` (PC-2, NEW — runRouter with a TCP management_socket, net.Dial("tcp", addr) succeeds after start; use a loopback host:port with ephemeral port to avoid CI flakiness)
- `TestRouterMgmtListener_TCPBind_PushHandshakeSucceeds` (PC-3, NEW — real admissionSyncClient pushes internal.admission.register to a runRouter with TCP management_socket; routerKS receives the entry)

---

### AC-009 — Full-snapshot push on control startup: control loads persisted keyset THEN pushes complete authoritative state to configured routers (BC-2.05.009 PC-7 v1.5 + Invariant 6 v1.5)

**BC Anchor:** BC-2.05.009 Postcondition 7 (v1.6) + Invariant 6 (v1.5). PC-7 (v1.6): adds per-endpoint sequencing obligation paragraph — register→expire→compensate state machine MUST be applied per configured router endpoint independently (Ruling 15 / F-P8-01). PC-7b (v1.5, unchanged): for active entries with non-zero expiry, issue register+expire pair; if expire fails AND expiry is past at failure time → best-effort compensating `internal.admission.revoke` (advisory-continue; Ruling 14 / F-P7-01); if expire fails for a FUTURE-expiry entry → no compensating action (PC-5 stale/missing, permitted). PC-7c (v1.4, unchanged): MUST NOT issue `internal.admission.register` for revoked entries — revoke-only RPC; router treats "key not found" as success. Invariant 6 (v1.5, unchanged): adds future-vs-past-expiry carve-out; AdmitNode does NOT check expiry (only ReAuthenticate does); past-expiry key left active-and-non-expiring is immediately exploitable at admission handshake. EC-009 (unchanged). EC-010 (v1.5, unchanged): compensating revoke fires on past-expiry expire-fail.

**Postconditions:**
1. When `runControl` starts and `SVTNManager` is initialised, `admissionSyncClient.PushFullSnapshot(ctx)`
   is called before the management server begins serving.
2. When `control_admission_state_file` is configured in the control daemon's config, `runControl`
   loads the keyset from that file via `loadSnapshotFromFile` BEFORE constructing the sync client
   and calling `PushFullSnapshot`. The test setup MUST configure `control_admission_state_file`
   and pre-populate it with a known non-empty keyset to exercise this path. A push of a
   manually-populated in-memory keyset (without load-from-file) does not satisfy this postcondition.
3. `PushFullSnapshot` iterates all entries in the loaded `AdmittedKeySet` (via `ListBySVTN` across
   all SVTNs) and for each entry:
   (a) issues `internal.admission.register` to each configured router endpoint;
   (b) for entries with a non-zero expiry timestamp (past or future), issues
       `internal.admission.expire(originalExpiry)` so the router preserves the original expiry
       timestamp, not a new one. If `internal.admission.expire` fails after retry exhaustion AND
       the original expiry timestamp is already in the past at the time of failure, issues a
       best-effort `internal.admission.revoke` as a compensating action to leave the router in a
       non-admissible state (revoked). The compensating revoke is advisory-continue on failure
       (WARN log). If expire fails for a FUTURE-expiry entry, no compensating action is required
       — the router holds the key active, which correctly represents control's current state (the
       key is active now; the staleness manifests only after the expiry elapses — PC-5 stale/missing,
       not an Invariant 6 violation). If register (step a) fails, skip expire entirely (the router
       has no entry for this key; the expire RPC would return E-ADM-013; the continue on
       register-fail avoids a wasted dial). AdmitNode does NOT check expiry; a past-expiry key
       left active-and-non-expiring on the router is immediately admissible at initial handshake —
       sharpening the Invariant 6 obligation for past-expiry partial failure (Ruling 14 / F-P7-01;
       traces to BC-2.05.009 v1.5 PC-7b postcondition);
   (c) **for REVOKED entries (Ruling 13 / F-P6-02):** does NOT issue
       `internal.admission.register`. A fresh router has no entry for the revoked key, which is
       the correct non-admissible terminal state — absent is sufficient. If the router already
       holds an entry for the key (partial prior push or pre-revocation registration), issues
       `internal.admission.revoke` only, to mark it revoked. In either case the router MUST NOT
       be left with the key registered-as-active. The register+revoke two-RPC pattern is
       PROHIBITED for revoked entries because a partial failure (register succeeds, revoke fails)
       leaves the key active on a fresh router, violating Invariant 6. The router's
       `internal.admission.revoke` handler MUST treat "key not present" as success — absent is
       the correct non-admissible terminal state (BC-2.05.009 PC-7c v1.4; traces to
       BC-2.05.009 PC-7c postcondition).
4. After `PushFullSnapshot` completes, the router's `AdmittedKeySet` is SEMANTICALLY EQUIVALENT
   to the control daemon's loaded `AdmittedKeySet`: same keys, roles, revoked status, and expiries.
   Specifically: a key that is revoked in control MUST be either ABSENT or REVOKED on the router
   (both are non-admissible — both satisfy Invariant 6); it MUST NOT be registered-as-active.
   A key with a past expiry MUST be expired on the router (NOT left active-and-non-expiring).
   This satisfies BC-2.05.009 PC-7 v1.4 and Invariant 6 v1.4 (router MUST NOT be left in a
   less-restrictive state than control).
5. Push failures are logged at WARN and do not block startup. They do not satisfy the semantic-
   equivalence guarantee — a router that failed to receive the full snapshot is stale until the
   next per-write push or a subsequent control restart.
6. When `control_admission_state_file` is absent/empty, `PushFullSnapshot` on startup pushes an
   empty keyset (EC-007 resync is inert — this is expected and tested by the EmptyKeyset test below).

**Test names:**
- `TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter` (integration: `control_admission_state_file` configured + pre-populated; two in-process mgmt.Server instances; router keyset receives loaded keys)
- `TestAdmissionSync_PushFullSnapshot_ExpiryPushed`
- `TestAdmissionSync_PushFullSnapshot_EmptyKeysetNoPushAttempt` (no `control_admission_state_file` → empty keyset → no push attempt)
- `TestAdmissionSync_PushFullSnapshot_RevokedKeyStaysRevoked` — STRENGTHENED per Ruling 13: must cover BOTH router precondition cases:
  - **Fresh router (no prior entry):** after `PushFullSnapshot` with a revoked entry, the router has NO entry for that key (ABSENT — not registered-then-revoked, not active). `IsAdmitted` returns false; the key does not appear in `ListBySVTN`. Verifies that PC-3c's skip-register path leaves the router in a non-admissible absent state (BC-2.05.009 PC-7c v1.4 / Invariant 6 / EC-009).
  - **Existing-entry router (key was registered before revocation, or partial prior push):** after `PushFullSnapshot` with a revoked entry, the router has the entry marked REVOKED (`IsRevoked()=true`, `IsAdmitted()=false`). Verifies that the revoke-only RPC is correctly applied to a pre-existing entry.
- `TestAdmissionSync_PushFullSnapshot_RevokedKey_RegisterNotSent` — regression guard for Ruling 13 / F-P6-02: asserts that `PushFullSnapshot` for a control keyset containing a revoked entry does NOT send an `internal.admission.register` RPC to the router for that entry. Verify via one of: (a) a spy/recording `admissionSyncer` that records which Push* methods were called — assert `PushRegisterKey` was NOT called for the revoked key's svtnID+pubkey tuple; OR (b) on a fresh router after `PushFullSnapshot`, assert the router's keyset has NO entry for the revoked key (absent, not "registered with revoked=true"). A router that never received a register for a key is the canonical proof that register was not sent.
- `TestAdmissionSync_PushFullSnapshot_PastExpiryStaysExpired` (a past-expiry entry in the pushed keyset → router shows it expired, not active-and-non-expiring; verifies BC-2.05.009 PC-7 v1.5 + EC-010)
- `TestAdmissionSync_PushFullSnapshot_PastExpiry_ExpireFails_CompensatingRevoke` — **REQUIRED** (Ruling 14 / F-P7-01 / BC-2.05.009 v1.5 PC-7b): simulate `internal.admission.expire` failing for a PAST-expiry active entry (use a spy/recording syncer that records which `Push*` methods were called and in what order, returning an error from `PushSetKeyExpiry` for the past-expiry entry while succeeding on `PushRegisterKey` and `PushRevokeKey`). Assert: (a) `PushRevokeKey` IS subsequently called for that entry after the failed `PushSetKeyExpiry` (spy records the call sequence: `PushRegisterKey` → `PushSetKeyExpiry`-FAIL → `PushRevokeKey`); (b) the key ends NON-ADMISSIBLE on the router (revoked state, not active-and-non-expiring). **Negative case** (sub-assertion or companion test): a FUTURE-expiry entry where `PushSetKeyExpiry` fails MUST NOT trigger a compensating `PushRevokeKey` — the spy must record `PushRegisterKey` → `PushSetKeyExpiry`-FAIL with no subsequent `PushRevokeKey` for that entry. This gates the past-expiry check (`e.KeyExpiry().Before(time.Now().UTC())`) is present in the implementation.

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

### AC-011 — Control-side keyset persistence via control_admission_state_file: validate, persist-on-mutation, load-on-startup (BC-2.05.009 PC-7 v1.2 / BC-2.09.003 PC-15 v2.2 / Ruling 11)

**BC Anchor:** BC-2.05.009 Postcondition 7 (v1.2); BC-2.09.003 Postcondition 15 (v2.2).

**Postconditions:**
1. `Config.Validate()` accepts `control_admission_state_file` absent or empty string without error.
   When `control_admission_state_file` is present with a whitespace-only value, `Config.Validate()`
   returns an error containing E-CFG-017:
   `"config error: control_admission_state_file: must not be empty. Fix: set to a valid writable file path, e.g. '/var/lib/switchboard/control-admission-state.json', or remove the field to disable control-side persistence"`.
   No file I/O in `Validate()`.
2. After a successful `admin.key.register` (and `admin.key.revoke`, `admin.key.expire`,
   `admin.svtn.destroy`) on control with `control_admission_state_file` configured, the file
   contains the current `AdmittedKeySet` in schema_version:1 JSON (reusing `writeSnapshotAtomic`).
   The persist-write is **synchronous** on the handler goroutine path, BEFORE `dispatchPush`.
   Write failure is advisory (WARN log; no rollback of the `m.*` write; handler returns success).
3. On control startup with `control_admission_state_file` set and the file present and valid,
   control loads the keyset via `loadSnapshotFromFile` BEFORE constructing the sync client and
   calling `PushFullSnapshot`. A corrupt file causes `runControl` to return E-KEY-002 (fail-closed).
   A missing file results in an empty keyset (fresh install — no error).
4. With `control_admission_state_file` configured and populated, `PushFullSnapshot` on startup
   pushes the recovered keys to routers (EC-007 resync is real). Without the field,
   `PushFullSnapshot` pushes an empty keyset (EC-007 inert).

**Test names:**
- `TestControlAdmission_PersistOnMutation`
- `TestControlAdmission_LoadAndPushFullSnapshot`
- `TestControlAdmission_FailClosedOnCorruptSnapshot`
- `TestControlAdmission_MissingFileEmptyKeyset`
- `TestConfig_Validate_ControlAdmissionStateFile_WhitespaceRejectsE_CFG_017`
- `TestConfig_Validate_ControlAdmissionStateFile_AbsentAccepted`
- `TestControlAdmission_RunControl_LoadThenPush_E2E` — **amended by Ruling 15 / F-P8-02**: must use `runControlWithKey` with a prepared `controlPriv` pre-registered in the router's `authorized_operator_keys`; assert `routerKS.AllSVTNEntries()` directly — active key present on router, revoked key absent (Ruling 13: register skipped for revoked entries); removes the direct-helper fallback that provided no coverage of `runControl`'s actual load→push ordering code path. See impl task 17i.

---

### AC-012 — Mgmt listener loopback guard scope: console/control/access reject non-loopback TCP; router remains exempt (BC-2.09.003 PC-11b v2.2 / Ruling 12)

**BC Anchor:** BC-2.09.003 Postcondition 11b (v2.2).

**Postconditions:**
1. A control-mode daemon with `management_socket: "0.0.0.0:<port>"` (non-loopback TCP) fails at
   `buildMgmtListener` with E-CFG-008:
   `"config error: management_socket: control mode requires a loopback address (127.0.0.1, [::1], or localhost); got: 0.0.0.0:<port>"`.
   The daemon exits 1 before accepting any connections.
2. A control-mode daemon with `management_socket: "127.0.0.1:<port>"` (loopback TCP) binds
   successfully. The loopback guard passes.
3. Router mode remains exempt from the loopback guard: a non-loopback `management_socket` TCP
   address is accepted (Ruling 9 unchanged — that is AC-008, not changed by this AC).
4. On a successful TCP bind in any mode, the daemon emits an INFO log:
   `"<mode> management listener bound to <address>"`.

**Test names:**
- `TestControlMgmtListener_NonLoopbackRejected`
- `TestControlMgmtListener_LoopbackTCPAccepted`

---

### AC-013 — PushFullSnapshot multi-endpoint per-endpoint sequencing: each reachable endpoint independently reaches its correct terminal state (BC-2.05.009 v1.6 PC-7 per-endpoint sequencing obligation + Invariant 6 / Ruling 15 / F-P8-01)

**BC Anchor:** BC-2.05.009 v1.6 Postcondition 7 (per-endpoint sequencing obligation paragraph) + Invariant 6 (v1.5). Ruling 15 is an IMPL-ONLY fix — the spec's "each configured router endpoint" intent in PC-7 was always per-endpoint sequential; the prior implementation violated it. The `admissionSyncer` interface signatures (`PushRegisterKey`, `PushRevokeKey`, `PushSetKeyExpiry`, `PushRemoveSVTN`) are UNCHANGED; delta-push paths (admin handlers) are UNCHANGED. (traces to BC-2.05.009 v1.6 PC-7 per-endpoint sequencing obligation + Invariant 6)

**Postconditions:**
1. When `RouterManagementEndpoints` contains multiple entries and one endpoint is unreachable,
   `PushFullSnapshot` processes each endpoint independently through the full per-entry
   register→expire→compensating-revoke state machine. A failure on endpoint B does NOT suppress
   expire or compensating-revoke for endpoint A (which may have already received register
   successfully). Each reachable endpoint independently reaches its correct terminal state
   regardless of any unreachable endpoint's outcome. Unreachable endpoints remain stale
   (PC-5 stale/missing — permitted).
2. For a past-expiry active entry: if endpoint A receives `internal.admission.register`
   successfully and endpoint B is unreachable (register fails), the compensating-revoke for
   past-expiry expire-fail fires only against endpoint A (if A's expire also fails), not against B.
   B remains stale (PC-5 permitted). A ends non-admissible (Invariant 6 satisfied).
3. The `admissionSyncer` interface signatures for `PushRegisterKey`, `PushRevokeKey`,
   `PushSetKeyExpiry`, and `PushRemoveSVTN` are UNCHANGED. Delta-push paths (admin handlers
   calling `Push*` methods after each `admin.key.*` write) are UNCHANGED. Only
   `PushFullSnapshot`'s internal structure changes (new `pushSnapshotToEndpoint` helper with
   outer-endpoint / inner-entry loop). (traces to BC-2.05.009 v1.6 PC-7 per-endpoint
   sequencing obligation / Ruling 15)

**Mechanism:** Use one real in-process router TCP management server (via `startRouterMgmtServerTCP` or equivalent — a real `runRouter` instance or an in-process `mgmt.Server` with `wireAdmissionSyncHandlers` wired) for the reachable endpoint, and a black-hole address (e.g. `"192.0.2.1:9"` — RFC 5737 documentation prefix, guaranteed non-listening) for the unreachable endpoint. Configure `RouterManagementEndpoints` with both. After `PushFullSnapshot`, inspect the real router's `AdmittedKeySet` (via `routerKS.AllSVTNEntries()` or equivalent) to assert the correct terminal state. The black-hole endpoint's failure confirms per-endpoint independence: if the prior aggregate-error implementation were restored, the reachable endpoint would end in the wrong state.

**Test names:**
- `TestAdmissionSync_PushFullSnapshot_MultiEndpoint_LastUnreachable_PastExpiry_ReachableEndpointNonAdmissible`
  — **This is the exact F-P8-01 failure-vector regression test.** Two endpoints configured: first is reachable (real in-process router TCP server), second is unreachable (black-hole address). Keyset contains a PAST-expiry active entry. After `PushFullSnapshot`: assert the REACHABLE (first) endpoint ends NON-ADMISSIBLE for that entry (expired or revoked — not active-and-non-expiring). If the fix is reverted to the prior aggregate-flatten-across-endpoints behavior, this test MUST fail (the reachable endpoint's past-expiry entry would be skipped by the `continue` on B's register-fail, leaving it active).
- `TestAdmissionSync_PushFullSnapshot_MultiEndpoint_FirstUnreachable_ReachableEndpointCorrect`
  — Two endpoints configured: first is unreachable (black-hole), second is reachable (real in-process router TCP server). Keyset contains active entries (some with expiry, some without) and revoked entries. After `PushFullSnapshot`: assert the REACHABLE (second) endpoint receives the full per-entry state machine correctly (register + expire for active entries; revoke-only for revoked). Verifies that per-endpoint independence holds regardless of which endpoint position is unreachable. First endpoint remains stale (PC-5 — not asserted).

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
- **Full HLR/VLR replication protocol** — the HLR/VLR architecture described in
  `identity-cluster-architecture.md` §8 is a forward architecture. This story delivers both
  the VLR-local router snapshot (BC-2.05.010) and the control-side keyset persistence
  (`control_admission_state_file`, Ruling 11 — now IN-SCOPE per human decision). What remains
  deferred is the full HLR replication protocol (cross-cluster multi-HLR state replication),
  which requires infrastructure not yet defined.
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
| `internal/config/config.go` | Add `RouterManagementEndpoints []RouterManagementEndpoint` + `RouterManagementEndpoint` type + YAML tags; add `AdmissionStateFile string` + YAML tag; add `ControlAdmissionStateFile string` + YAML tag; extend `Config.Validate()` with E-CFG-015 (admission_state_file whitespace), E-CFG-016 (router_management_endpoints addr host:port, exhaustive, no loopback restriction), E-CFG-017 (control_admission_state_file whitespace) | BC-2.09.003 v2.2 PC-13, PC-14, PC-15 |
| `cmd/switchboard/admission_sync_client.go` (new) | `admissionSyncer` interface; `admissionSyncClient` type holding endpoints + daemonPriv; `PushRegisterKey`, `PushRevokeKey`, `PushSetKeyExpiry`, `PushRemoveSVTN` methods (dial-on-demand, retry-with-backoff); `PushFullSnapshot`; SIGHUP endpoint-list update | BC-2.05.009 Rulings 1–2 |
| `cmd/switchboard/admission_sync_wire.go` (new) | `wireAdmissionSyncHandlers(srv *mgmt.Server, ks *admission.AdmittedKeySet, snapshotPath string)` + four `mgmt.Handler` registrations for `internal.admission.*`; per-handler snapshot write after successful keyset update | BC-2.05.009 Ruling 3; BC-2.05.010 |
| `cmd/switchboard/admission_sync_snapshot.go` (new, or inline in wire.go) | Snapshot serialization/deserialization (JSON marshal/unmarshal to/from `admission.AdmittedKeySet` state); atomic write; load-on-startup logic | BC-2.05.010 |
| `cmd/switchboard/admin_handlers.go` | Extend `BuildAdminHandlers` signature with `syncClient admissionSyncer` parameter; add push call after each successful `SVTNManager.*` write in `makeRegisterHandler`, `makeRevokeHandler`, `makeExpireHandler`, `makeAdminSVTNDestroyHandler` | BC-2.05.009 PC-1/PC-2 |
| `cmd/switchboard/router.go` (or equivalent `runRouter`) | Add `wireAdmissionSyncHandlers(...)` call (after `newMgmtServer`, before `serveMgmtServer`); add snapshot load-on-startup from `cfg.AdmissionStateFile`; emit non-loopback bind INFO log | BC-2.05.010 PC-6/7/8/9; Ruling 9 |
| `cmd/switchboard/control.go` (or equivalent `runControl`) | Construct `admissionSyncClient`; pass to `BuildAdminHandlers`; load `ControlAdmissionStateFile` snapshot BEFORE constructing sync client and calling `PushFullSnapshot`; call `PushFullSnapshot(ctx)` on startup; update client endpoints on SIGHUP; add synchronous `writeSnapshotAtomic` call in four admin write handlers BEFORE `dispatchPush` | BC-2.05.009 PC-7 v1.2; Invariant 5; Ruling 11 |
| `cmd/switchboard/mgmt_wire.go` (or `buildMgmtListener`) | Extend `buildMgmtListener` loopback guard from `if mode == "console"` to `if mode != "router"` — applies guard to console, control, and access modes; router-only exemption (Ruling 12) | BC-2.09.003 PC-11b v2.2 |
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
| Tests (13 ACs, ~30 test functions, including multi-endpoint integration tests for AC-013) | ~800 tokens |
| `pushSnapshotToEndpoint` helper + `PushFullSnapshot` outer-loop refactor (task 17h) | ~80 tokens |
| `runControlWithKey` seam + `TestControlAdmission_RunControl_LoadThenPush_E2E` amendment (task 17i) | ~60 tokens |
| **Overall** | ~1,700 tokens — this story is 12-point scope; the token budget is consistent with that |

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
9a. [ ] Write failing tests for AC-011 (control-side persistence: validate E-CFG-017; persist-on-mutation; load-on-startup; fail-closed on corrupt) — test-writer
9b. [ ] Write failing tests for AC-012 (mgmt loopback guard scope: control/access non-loopback rejected; loopback accepted) — test-writer
10. [ ] Verify Red Gate: `go test ./...` fails with compile or test failures for all ACs
11. [ ] Implement `internal/config` fields + `Config.Validate()` extensions (add `ControlAdmissionStateFile` + E-CFG-017) — implementer
12. [ ] Implement `admission_sync_client.go` (interface + retry client + PushFullSnapshot) — implementer
13. [ ] Implement `admission_sync_wire.go` (four handler registrations) — implementer
14. [ ] Implement snapshot serialization/deserialization — implementer
15. [ ] Wire push calls into `admin_handlers.go` — implementer
15a. [ ] Wire synchronous `writeSnapshotAtomic` into the four admin write handlers (`makeRegisterHandler`, `makeRevokeHandler`, `makeExpireHandler`, `makeAdminSVTNDestroyHandler`) BEFORE `dispatchPush`; write failure is advisory (WARN, no rollback) — implementer [Ruling 11]
16. [ ] Wire `runRouter` startup load + non-loopback bind log — implementer
16a. [ ] Implement `mgmtNetwork`/`buildMgmtListener` auto-detect: if `validateHostPort(management_socket)` passes → bind TCP (no `isMgmtLoopbackHost` guard for router); else → bind unix (Ruling 10 / F-2 fix) — implementer
16b. [ ] Extend `buildMgmtListener` loopback guard from `if mode == "console"` to `if mode != "router"` — control/access/console all require loopback TCP; router remains exempt (Ruling 12) — implementer
17. [ ] Wire `runControl`: (a) load `ControlAdmissionStateFile` snapshot via `loadSnapshotFromFile` BEFORE constructing sync client + calling `PushFullSnapshot`; (b) client construction + PushFullSnapshot + SIGHUP reload — implementer [Ruling 11]
17a. [ ] F-P3-03 (impl obligation, no new AC): bound the push dialer timeout and bound `pushWG.Wait()` shutdown drain so a black-holed router endpoint cannot stall daemon shutdown indefinitely — implementer
17b. [ ] F-3 (impl obligation, no new AC — BC-2.05.009 PC-2/PC-4, BC-2.05.010 PC-2): advisory push/write failures that currently swallow the error with `_ = err` MUST emit a WARN log instead. The push handlers in `admission_sync_client.go` and the snapshot-write path in `admission_sync_wire.go` need a log writer threaded in. Push failure WARN must include the endpoint address and the underlying error. Snapshot-write failure WARN must include the file path and error. — implementer
17c. [ ] F-4 (impl obligation, no new AC — BC-2.09.003 PC-11b / AC-012 PC-4): the bind-address INFO log `"<mode> management listener bound to <address>"` must fire for console and access modes in addition to router and control. AC-012 PC-4 states "any mode that binds TCP." Extend the INFO log emission in `buildMgmtListener` to cover all four modes, not just router+control. — implementer
17d. [ ] PushFullSnapshot revoked-entry handling (Ruling 13 / F-P6-02): in the `PushFullSnapshot` loop body in `admission_sync_client.go`, for entries where `entry.IsRevoked()` is true, skip `PushRegisterKey` entirely; call `PushRevokeKey` only (advisory — failure logged at WARN, same as any push failure). On a fresh router that has no entry for the revoked key, the revoke RPC arrives for an absent key; the router's `internal.admission.revoke` handler MUST treat "key not found" as success (return nil, not error) — absent is the correct non-admissible terminal state for a revoked key. The net loop structure changes from `register → expire? → revoke?` to: if `IsRevoked()`: `revoke` + continue; else: `register → expire?`. Document the router handler's idempotent behavior for the key-not-found case. Reference: Ruling 13 / F-P6-02 / BC-2.05.009 PC-7c v1.4. — implementer
17e. [ ] F-P6-01 concurrent snapshot-write race (impl fix, no spec change — BC-2.05.010 Invariant 1 already mandates correct serialized snapshot writes): in `wireAdmissionSyncHandlers` (admission_sync_wire.go), serialize the four router-side push-handler calls to `writeSnapshotAtomic` with a shared write mutex. Two concurrent push RPCs (e.g., register + expire arriving simultaneously) currently each call `writeSnapshotAtomic` without serialization; the last-write-wins on a potentially stale keyset capture can drop intermediate updates. Fix: acquire a per-snapshot-file `sync.Mutex` before `ks.*` write + `writeSnapshotAtomic`, release after — mirrors the `controlPersister` approach. No BC change required (BC-2.05.010 PC-1 and Invariant 1 already cover "correct serialized snapshot"). — implementer
17f. [ ] PushFullSnapshot past-expiry partial-failure handling (Ruling 14 / F-P7-01 / BC-2.05.009 v1.5 PC-7b): in the active-entry (non-revoked) loop body in `admission_sync_client.go`: (i) add `continue` on register-fail — if `PushRegisterKey` returns an error, skip `PushSetKeyExpiry` for that entry (the router has no entry; expire would return E-ADM-013; the wasted dial is eliminated). (ii) after a FAILED `PushSetKeyExpiry`, check `e.KeyExpiry().Before(time.Now().UTC())` — if the expiry is already in the past at failure time, issue a best-effort `PushRevokeKey(ctx, svtnID, e.PublicKey, e.Role, true)` as a compensating action; log WARN if compensating revoke also fails (router may hold the past-expiry key as active — a double-failure residual; next `PushFullSnapshot` on restart will retry). (iii) FUTURE-expiry expire-fail: no compensating action — router holds key active, which is correct at push time (PC-5 permitted staleness). No change to the router expire handler (`admission_sync_wire.go`) or expire wire args. Reference: Ruling 14 / F-P7-01 / BC-2.05.009 v1.5 PC-7b. — implementer
17g. [ ] LOW consistency fix (pass-7 adversary advisory, no AC): `routerPersister.persist` should fall back to `os.Stderr` when its writer field is nil, matching the `controlPersister` pattern (`admission_sync_wire.go` ~67–70). Production is unaffected since `runRouter` always threads a real writer; the fix prevents a nil-pointer panic in any test or future call path that constructs a `routerPersister` without a writer. — implementer
17h. [ ] PushFullSnapshot per-endpoint sequencing (Ruling 15 / F-P8-01): replace the current `pushSnapshotEntries(ctx, c, allEntries, nil)` delegation in `PushFullSnapshot` with an outer loop over `c.currentEndpoints()` calling a new private `c.pushSnapshotToEndpoint(ctx, ep.Addr, allEntries, nil)` helper. The helper replicates the `pushSnapshotEntries` per-entry state machine (Rulings 13+14 preserved: revoke-only for revoked entries; register→expire→compensating-revoke for active entries) using a single-endpoint retry wrapper (`pushOneRPC`) that calls `pushRPC` directly for one endpoint at a time — NOT `pushWithRetry` (which fans to all endpoints). `pushSnapshotEntries` and the `admissionSyncer` interface remain UNCHANGED (used by spy tests and delta-push paths). `pushWithRetry` and all four `Push*` methods remain UNCHANGED. Single-endpoint behavior must be identical to the current behavior (Rulings 13/14 semantics preserved exactly; the outer loop iterates once for a single-endpoint config). Reference: Ruling 15 / F-P8-01. — implementer
17i. [ ] runControl load→push ordering guard (Ruling 15 / F-P8-02): add `runControlWithKey(ctx context.Context, w io.Writer, cfg *config.Config, configPath string, sighupCh <-chan os.Signal, daemonPriv ed25519.PrivateKey) error` in `mgmt_wire.go`; refactor `runControl` to be a thin wrapper that generates a fresh ephemeral keypair via `ed25519.GenerateKey` and delegates to `runControlWithKey`. `runControlWithKey` is unexported (test-internal seam, not a public API). Amend `TestControlAdmission_RunControl_LoadThenPush_E2E` to: (a) use `runControlWithKey` with the test's prepared `controlPriv` (pre-registered in the router's `authorized_operator_keys`) so control authenticates successfully and `PushFullSnapshot` can actually push to the router; (b) remove the direct-helper fallback path that bypassed `runControl`'s load→push ordering code; (c) assert `routerKS.AllSVTNEntries()` directly — the active key from the pre-populated `control_admission_state_file` snapshot MUST be present on the router after `runControlWithKey` starts; the revoked key MUST be absent (Ruling 13: `PushFullSnapshot` skips register for revoked entries, so fresh router has no entry). Reference: Ruling 15 / F-P8-02. — implementer
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
| Ruling 10 | `mgmtNetwork`/`buildMgmtListener` auto-detects TCP-vs-unix via `validateHostPort(management_socket)` — TCP when host:port, unix otherwise | Verified by AC-008 tests `TestRouterMgmtListener_TCPBind_ConnectionSucceeds` and `TestRouterMgmtListener_TCPBind_PushHandshakeSucceeds` |
| BC-2.05.010 Invariant 2 | Loaded entries have `admitted=false` | Verified by AC-007 test `TestRouterStartup_LoadedEntries_AdmittedFalse` |
| Ruling 11 (F-P3-01) | Control-side persist-write is synchronous BEFORE push dispatch; failure is advisory (WARN, no rollback); load-on-startup BEFORE PushFullSnapshot | Verified by AC-011 tests |
| Ruling 12 (F-P3-02) | `buildMgmtListener` guard is `if mode != "router"` — console/control/access require loopback TCP; router-only exemption | Verified by AC-012 tests `TestControlMgmtListener_NonLoopbackRejected` and `TestControlMgmtListener_LoopbackTCPAccepted` |
| BC-2.09.003 PC-15 v2.2 | `control_admission_state_file` whitespace-only → E-CFG-017; absent accepted; no file I/O in Validate() | Verified by AC-011 tests `TestConfig_Validate_ControlAdmissionStateFile_*` |

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
| `internal/config/config.go` | modify | Add `RouterManagementEndpoints`, `RouterManagementEndpoint`, `AdmissionStateFile`, `ControlAdmissionStateFile`; extend `Validate()` with E-CFG-015 / E-CFG-016 / E-CFG-017 |
| `internal/config/config_test.go` | modify | Table-driven tests for AC-001 |
| `cmd/switchboard/admission_sync_client.go` | create | `admissionSyncer` interface; `admissionSyncClient` with retry-with-backoff; `PushFullSnapshot`; SIGHUP endpoint-list update |
| `cmd/switchboard/admission_sync_wire.go` | create | `wireAdmissionSyncHandlers`; four `internal.admission.*` handler registrations; per-handler snapshot write |
| `cmd/switchboard/admission_sync_snapshot.go` | create | Snapshot JSON marshal/unmarshal; atomic write; load-on-startup logic |
| `cmd/switchboard/admin_handlers.go` | modify | Extend `BuildAdminHandlers` with `syncClient admissionSyncer`; add push calls in four write handlers |
| `cmd/switchboard/router.go` | modify | Call `wireAdmissionSyncHandlers`; snapshot load on startup; non-loopback bind INFO log |
| `cmd/switchboard/control.go` | modify | Construct `admissionSyncClient`; pass to `BuildAdminHandlers`; load `ControlAdmissionStateFile` snapshot BEFORE sync-client construction; call `PushFullSnapshot`; SIGHUP reload; synchronous `writeSnapshotAtomic` in four admin write handlers BEFORE `dispatchPush` |
| `cmd/switchboard/mgmt_wire.go` (or file containing `buildMgmtListener`) | modify | Extend loopback guard from `if mode == "console"` to `if mode != "router"` — Ruling 12; add `runControlWithKey` unexported function + refactor `runControl` to delegate to it — Ruling 15 / F-P8-02 (task 17i) |
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
- **Rulings:** `decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md` v1.7 (all 15 rulings, zero open
  human flags — Ruling 10 added 2026-07-17 (F-2 fix: router mgmt listener TCP-vs-unix auto-detect);
  Ruling 11 added 2026-07-17 (F-P3-01: control-side keyset persistence via control_admission_state_file,
  now IN-SCOPE); Ruling 12 added 2026-07-17 (F-P3-02: buildMgmtListener loopback guard extended to
  console/control/access, router-only exemption); Ruling 13 added 2026-07-17 (F-P6-02: PushFullSnapshot
  MUST NOT issue register for revoked entries — revoke-only RPC; router treats key-not-found as success;
  register+revoke two-RPC pattern PROHIBITED; F-P6-01 concurrent snapshot-write race is impl-only fix);
  Ruling 14 added 2026-07-18 (F-P7-01: PushFullSnapshot past-expiry partial-failure gap — compensating
  best-effort revoke when expire fails AND expiry is past; future-expiry expire-fail is PC-5
  stale/missing, permitted; AdmitNode does NOT check expiry confirmed at admission.go:457–526);
  Ruling 15 added 2026-07-18 (F-P8-01: PushFullSnapshot multi-endpoint per-endpoint sequencing —
  endpoints outer/entries inner, new pushSnapshotToEndpoint helper using single-endpoint pushRPC-with-retry,
  delta-push paths and admissionSyncer interface unchanged; F-P8-02: runControlWithKey test seam —
  runControl refactored to delegate to runControlWithKey(daemonPriv), TestControlAdmission_RunControl_
  LoadThenPush_E2E amended to use seam + assert routerKS.AllSVTNEntries() directly)).
- **Unblocks:** `S-BL.NODE-IDENTIFY-WIRE`'s Open Design Obligation 5 (admission.AdmitNode
  is verification-only against the router's always-empty keyset — this story populates that keyset).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.7 | 2026-07-18 | Ruling 15 propagation (F-P8-01 + F-P8-02): add AC-013 (PushFullSnapshot multi-endpoint per-endpoint sequencing — each reachable endpoint independently reaches correct terminal state regardless of other endpoints' reachability; traces to BC-2.05.009 v1.6 PC-7 per-endpoint sequencing obligation + Invariant 6 / Ruling 15 / F-P8-01). Two new test names: `TestAdmissionSync_PushFullSnapshot_MultiEndpoint_LastUnreachable_PastExpiry_ReachableEndpointNonAdmissible`; `TestAdmissionSync_PushFullSnapshot_MultiEndpoint_FirstUnreachable_ReachableEndpointCorrect`. Add impl task 17h (`pushSnapshotToEndpoint` helper + `PushFullSnapshot` outer-loop refactor; Ruling 15 / F-P8-01). Add impl task 17i (`runControlWithKey` test seam + `TestControlAdmission_RunControl_LoadThenPush_E2E` amendment; Ruling 15 / F-P8-02). AC count 12→13; points 11→12. BC-2.05.009 ref v1.5→v1.6; rulings ref v1.6→v1.7. |
| 1.6 | 2026-07-18 | Propagate Ruling 14 (F-P7-01) + BC-2.05.009 v1.5: AC-009 PC-3b amended — add compensating best-effort `internal.admission.revoke` when expire fails AND expiry is already in the past, so router ends non-admissible (traces to BC-2.05.009 v1.5 PC-7b). Future-expiry expire-fail is PC-5 stale/missing (permitted) — no compensating action. Add `TestAdmissionSync_PushFullSnapshot_PastExpiry_ExpireFails_CompensatingRevoke` (required) with assertions: `PushRevokeKey` called after failed `PushSetKeyExpiry` for past-expiry entry; key ends non-admissible on router. Negative case (future-expiry expire-fail must NOT trigger compensating revoke) documented as sub-assertion. Add impl task 17f (PushFullSnapshot past-expiry partial-failure handling: skip expire on register-fail; compensating revoke on past-expiry expire-fail; no action on future-expiry expire-fail; Ruling 14 / F-P7-01). Add impl task 17g (LOW consistency: `routerPersister.persist` nil-writer fallback to `os.Stderr`, matching `controlPersister`). BC-2.05.009 ref v1.4→v1.5; rulings ref v1.5→v1.6. AC count 12, points 11 unchanged. |
| 1.5 | 2026-07-17 | Propagate Ruling 13 (F-P6-02) + BC-2.05.009 v1.4: AC-009 PC-3c amended — MUST NOT issue internal.admission.register for revoked entries; revoke-only RPC (router treats key-not-found as success — absent = correct non-admissible terminal state). Register+revoke two-RPC pattern PROHIBITED for revoked entries. Strengthen TestAdmissionSync_PushFullSnapshot_RevokedKeyStaysRevoked with BOTH precondition cases (fresh router → key ABSENT; existing-entry router → key REVOKED). Add TestAdmissionSync_PushFullSnapshot_RevokedKey_RegisterNotSent (regression guard). Add impl tasks 17d (skip-register for revoked entries + router revoke handler treats key-not-found as success) and 17e (F-P6-01 concurrent snapshot-write mutex). BC-2.05.009 ref v1.3→v1.4; rulings ref v1.4→v1.5. AC count 12, points 11 unchanged. |
| 1.4 | 2026-07-17 | Propagate BC-2.05.009 v1.3: AC-009 now requires PushFullSnapshot semantic-equivalence (revoked→revoked, past-expiry→expired, not register-only) + 2 new test names (RevokedKeyStaysRevoked, PastExpiryStaysExpired) — fixes F-1 revocation-un-propagation (adversary pass 5); add impl tasks for F-3 (WARN log on advisory failures) + F-4 (bind-log for console/access modes). |
| 1.3 | 2026-07-17 | Propagate rulings v1.4 (Rulings 11+12) + BC-2.09.003 v2.2 + BC-2.05.009 v1.2: add AC-011 (control-side keyset persistence via control_admission_state_file — F-P3-01, now in-scope) + AC-012 (mgmt loopback guard scope: control/access loopback-only TCP, router-only exemption — F-P3-02); amend AC-009 (load-then-push); remove control-persistence deferral from Non-Goals; points 8→11; AC count 10→12; note F-P3-03 shutdown-drain bound as impl task. |
| 1.2 | 2026-07-17 | Propagate rulings v1.3 Ruling 10 (F-2 fix): router mgmt listener auto-detects TCP-vs-unix on management_socket; AC-008 postconditions rewritten (5 PCs incl. real TCP-bind + push-handshake assertions) + 2 new test names (TCPBind_ConnectionSucceeds, TCPBind_PushHandshakeSucceeds); implementer task for mgmtNetwork/buildMgmtListener auto-detect. rulings ref v1.2→v1.3. |
| 1.1 | 2026-07-16 | Propagate rulings v1.2 svtn_id hex-[16]byte wire encoding fix (admissionSyncer interface svtnName→svtnID [16]byte; Decisions 2/5, AC-003/004/005); BC-2.05.009 ref 1.0→1.1; removed stale free-text input-hash citation from POL-005 note. |
| 1.0 | 2026-07-15 | Initial full decomposition — 10 ACs, 8 points, leaf prerequisite with `depends_on: []`. Admission-state sync push RPC (BC-2.05.009) + VLR-local JSON snapshot (BC-2.05.010). Per rulings v1.1 (Option A + Ruling 9 router TCP listener). |
