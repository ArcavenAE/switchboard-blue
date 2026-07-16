---
artifact_id: BC-2.05.009
document_type: behavioral-contract
level: L3
version: "1.1"
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
input-hash: "3e7d080"
extracted_from: null
bc_id: BC-2.05.009
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
  - date: 2026-07-16
    version: "1.1"
    change: >
      Amend Invariant 4: exempt svtn_id from encoding-parity rule — on the
      internal.admission.* wire it is the 32-lowercase-hex-char encoding of
      the [16]byte SVTN UUID (not the human-readable name). Rulings v1.2
      (commit 3d64ac2) / svtn_id hex-encoding fix.
  - date: 2026-07-15
    version: "1.0"
    change: >
      Initial draft — admission-state-sync push RPC: control pushes to router
      management endpoints on each RegisterKey/RevokeKey/ExpireKey/RemoveSVTN write;
      push failure does not roll back control write; admitted=false on load.
      Authored per S-BL.ADMISSION-SYNC-WIRE BC groundwork list item A1
      (S-BL.ADMISSION-SYNC-WIRE-rulings.md §7, identity-cluster-architecture.md §7–§9).
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

# Behavioral Contract BC-2.05.009: Admission-State Sync — Control Pushes Key Mutations to Router Management Endpoints via Internal RPC

## Description

When a control-mode daemon writes to its `AdmittedKeySet` (via `admin.key.register`,
`admin.key.revoke`, `admin.key.expire`, or `admin.svtn.destroy`), it also pushes the
corresponding mutation to each configured router management endpoint using the existing
`internal/mgmt` JSON-over-TCP protocol with four new `internal.admission.*` commands.
Push failure is advisory (WARN log) — the control-side write has already committed and
is not rolled back. On control startup, a full-snapshot push synchronises each router
with the current `AdmittedKeySet` state. Loaded entries are always `admitted=false`
on the router side (challenge-response handshake is required to flip `admitted=true`).

## Preconditions

**Per-write push preconditions:**
1. The control daemon has just successfully written a mutation to its own `AdmittedKeySet`
   via one of the four write paths (`RegisterKey`, `RevokeKeyIfRoleMatches`,
   `SetKeyExpiryIfRoleMatches`, `RemoveSVTN`).
2. `RouterManagementEndpoints` is non-empty in the control daemon's config (BC-2.09.003 v3.0
   amendment, A3).
3. An `admissionSyncer` (interface) is wired into `BuildAdminHandlers` (non-nil).

**Full-snapshot push preconditions:**
1. The control daemon is starting (`runControl` entry).
2. `SVTNManager` is initialized and holds any persisted admission state.
3. `RouterManagementEndpoints` is non-empty.

## Postconditions

### Per-write push postconditions

1. **Push attempted to all configured router endpoints:** After the successful control-side
   write, the `admissionSyncer` attempts to push the corresponding `internal.admission.*`
   RPC to each entry in `RouterManagementEndpoints`. The push commands are:
   - `RegisterKey` → `internal.admission.register` with `{svtn_id, pubkey_openssh, role}`
   - `RevokeKey` → `internal.admission.revoke` with `{svtn_id, pubkey_openssh, role, confirm}`
   - `SetKeyExpiry` → `internal.admission.expire` with `{svtn_id, pubkey_openssh, after}`
   - `RemoveSVTN` → `internal.admission.remove-svtn` with `{svtn_id}`

2. **Push failure does NOT roll back control write:** If the push to a router endpoint fails
   (connection refused, timeout, auth failure, or any transport error), the control-side
   write remains committed. The push failure is logged at WARN level with the endpoint
   address and error. The response returned to the `sbctl` caller reflects the control-side
   write success (the authoritative operation).

3. **Push uses existing mgmt protocol:** The push connection uses the `internal/mgmt`
   JSON-over-TCP protocol with Ed25519 challenge-response authentication (ADR-012). Control
   authenticates to the router using its own `daemonPriv` Ed25519 key. The router must have
   control's daemon public key in its `authorized_operator_keys` config (operator configuration
   requirement, not enforced by this BC).

4. **Retry-with-backoff per endpoint:** Each push attempt to a given endpoint uses bounded
   exponential backoff: initial delay 100ms, multiplier 2, maximum delay 10s, maximum
   attempts 5 (or as documented in code comments citing this ruling). After N consecutive
   failures, the endpoint is logged at WARN and retried on the next push event.

5. **Dial-on-demand:** There is no persistent idle connection from control to routers. Each
   push event opens a new TCP connection, completes the mgmt challenge-response handshake,
   sends the RPC, reads the response, and closes the connection.

6. **Nil syncer is a no-op:** If the `admissionSyncer` passed to `BuildAdminHandlers` is nil
   (router/console/access modes, which never push to other routers), the push is skipped
   entirely. No error is produced. This preserves backward compatibility with existing
   non-control handler registrations (ADR-004 / AC-004 role-exclusion).

### Full-snapshot push postconditions

7. **Startup full-snapshot:** On control startup, after `SVTNManager` is initialized,
   `PushFullSnapshot(ctx)` iterates all `(svtn, pubkey, role)` triples in the current
   `AdmittedKeySet` (via `ListBySVTN` across all SVTNs) and issues
   `internal.admission.register` (plus `internal.admission.expire` for entries with a
   non-zero expiry) to each configured router endpoint.

8. **admitted=false on load:** Entries pushed to a router and loaded from the router's
   VLR-local snapshot (BC-2.05.010) are always `admitted=false`. The challenge-response
   handshake (`S-BL.NODE-IDENTIFY-WIRE`) is required to flip `admitted=true`. The push
   does NOT propagate the live `admitted` state from control.

## Invariants

1. **Control is the authoritative write authority.** Push failure means the router is
   temporarily stale; it does NOT mean the operation failed. The operator's `sbctl`
   confirmation of success reflects the authoritative control-side state.
2. **DI-002:** The push RPCs carry public keys only (`pubkey_openssh` in OpenSSH wire
   format). Private keys are never transmitted.
3. **`internal.admission.*` commands are internal-RPC only.** They are registered on the
   router-mode management server (not on control/console/access-mode servers), and they are
   NEVER operator-facing `admin.*` verbs. ADR-004 / AC-004 role-exclusion remains intact.
4. **Encoding parity with `admin.*` handlers, with `svtn_id` exemption:** Args structs
   for `internal.admission.*` commands reuse or lightly adapt the same encoding as the
   corresponding `admin.key.*` args for all fields EXCEPT `svtn_id`:
   - `pubkey_openssh` (base64-encoded OpenSSH wire format) — same as `admin.key.*`
   - `role` (canonical string) — same as `admin.key.*`
   - `after` (Go duration string) — same as `admin.key.*`
   - `confirm` (bool) — same as `admin.key.*`
   - `svtn_id` — **EXEMPTED from encoding parity.** On the `internal.admission.*` wire,
     `svtn_id` is the **32-lowercase-hex-char encoding of the `[16]byte` SVTN UUID**
     (matching the VLR-local snapshot schema in BC-2.05.010), NOT the human-readable
     SVTN name used by the `admin.key.*` / `admin.svtn.*` args. The control-side admin
     handler resolves the human-readable name to `[16]byte` via `SVTNManager.SVTNByName`
     before constructing the push args. This distinction exists because the router has no
     `SVTNManager` to resolve names and derives `FrameAuthKey`/`NodeAddr` from the raw
     `[16]byte` UUID (rulings v1.2, commit 3d64ac2).
5. **SIGHUP reload integration:** When control receives SIGHUP and reloads config, the
   `admissionSyncClient` endpoint list is updated from the new config. In-flight pushes
   are not interrupted; the new list is used for the next push event.

## Trigger

- `admin.key.register`, `admin.key.revoke`, `admin.key.expire`, or `admin.svtn.destroy`
  handler in control-mode daemon completes a successful write to `SVTNManager` / `AdmittedKeySet`.
- Control daemon startup (`runControl` entry), after `SVTNManager` initialization.

## Error Codes

| Code | Condition | Severity | Exit Code | Notes |
|------|-----------|----------|-----------|-------|
| (WARN log) | Push to a router endpoint fails after retry exhaustion | degraded | — (daemon continues) | Logged at WARN with endpoint addr + error; no formal error code (internal operational signal, not operator-visible in sbctl response) |

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `RouterManagementEndpoints` is empty | `admissionSyncer.Push*` is a no-op (nil syncer or empty endpoint list); control write succeeds normally; no push attempted. |
| EC-002 | Router management endpoint unreachable (connection refused) | Retry-with-backoff up to 5 attempts; then WARN log; push skipped for this event. Router is stale until next control startup or next successful push event. |
| EC-003 | Router responds with auth failure (control key not in router's `authorized_operator_keys`) | Treated as push failure; WARN logged with endpoint and auth error. No rollback. Operator must add control's daemon pubkey to router's config. |
| EC-004 | Rapid burst of `admin.key.register` calls (N writes in quick succession) | Each write triggers a separate per-write push. N TCP connections opened. No batching in the near-term story. |
| EC-005 | `admissionSyncer` is nil (router/console/access mode) | Push skipped silently. No error. Existing behavior for non-control modes is preserved. |
| EC-006 | `internal.admission.register` sent to router for an SVTN the router has no record of | Router's `RegisterKey` creates the entry idempotently (per Ruling 1 — `RegisterKey` MUST be idempotent from the router's perspective for fresh-install races). |
| EC-007 | Control restarts after prior push failures | `PushFullSnapshot(ctx)` on startup pushes the full current `AdmittedKeySet` to all configured routers, correcting any staleness accumulated during downtime. |
| EC-008 | Router is temporarily restarting when push arrives | Push fails (connection refused); WARN logged. Router loads from its VLR-local snapshot (BC-2.05.010) on restart and receives a fresh full-snapshot push when control next starts or the next per-write push arrives. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `admin.key.register` succeeds on control; push to router succeeds | Router's `AdmittedKeySet` has the entry; `admitted=false` | happy-path |
| `admin.key.register` succeeds on control; push to router fails (no listener) | Control write committed; WARN logged; sbctl reports success; router is stale | push-failure isolation |
| `admin.key.revoke` on control; push succeeds | Router marks key revoked | happy-path |
| `admin.key.expire` on control; push succeeds | Router's keyset has non-zero expiry for the entry | happy-path |
| `admin.svtn.destroy` on control; push succeeds | Router has no entries for that SVTN | happy-path |
| Control startup with populated `AdmittedKeySet`; push to router | Router keyset matches control after `PushFullSnapshot` | startup-snapshot |
| `admissionSyncer` is nil (router/console mode) | No push attempted; no error; existing handler behavior unchanged | nil-syncer |
| Push to multiple endpoints; one fails, one succeeds | Both attempted; failure logged for the failing endpoint; no rollback | multi-endpoint |

## Verification Properties

| VP-NNN | Property | Proof Method | Notes |
|--------|----------|-------------|-------|
| test-as-evidence | After `admin.key.register` on control, router's `AdmittedKeySet` has the entry (push succeeded) | integration (two in-process `mgmt.Server` instances) | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Push failure does not roll back control write; sbctl reports success | integration | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Nil `admissionSyncer` → no-op, no panic | unit | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | `PushFullSnapshot` on control startup pushes all keyset entries to configured routers | integration | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Router handler: `internal.admission.register` creates entry with `admitted=false` | unit | S-BL.ADMISSION-SYNC-WIRE AC |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 |
| L2 Domain Invariants | DI-002 (private keys never transit), DI-001 (carrier-grade content separation — keys managed here authenticate transport/admission layer) |
| Architecture Module | cmd/switchboard (admissionSyncClient, admission_sync_wire.go, admin_handlers.go) |
| Stories | S-BL.ADMISSION-SYNC-WIRE (all postconditions and ACs) |
| Capability Anchor Justification | CAP-019 ("Key lifecycle management") — this BC extends the key lifecycle operations of BC-2.05.004 with the admission-state replication path that makes key registrations visible to routers. Without this replication, key registration on control has no effect on the routers that serve those keys. |

## Related BCs

- BC-2.05.004 — extends: every `admin.key.*` write path in BC-2.05.004 gains a push-failure postcondition via this BC (see also BC-2.05.004 amendment in this groundwork batch)
- BC-2.05.010 — composes with: this BC is the write path; BC-2.05.010 is the router-side persistence and load path
- BC-2.09.003 — amendment: `router_management_endpoints` config field validation added by groundwork item A3 (consolidated in BC-2.09.003 v3.0)

## Architecture Anchors

- decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md Ruling 1 (push RPC protocol, four write paths, wire shape)
- decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md Ruling 2 (control-side config field, dial client, failure behavior)
- decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md Ruling 3 (router-side push handlers)
- decisions/identity-cluster-architecture.md §7–§9 (Option A ratification, VLR-local snapshot, ODO-5 resolution)

## Story Anchor

S-BL.ADMISSION-SYNC-WIRE — all postconditions in this BC trace to acceptance criteria for this story.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-07-16 | Amend Invariant 4: exempt `svtn_id` from encoding-parity rule — on the `internal.admission.*` wire it is the 32-lowercase-hex-char encoding of the `[16]byte` SVTN UUID (not the human-readable name). Rulings v1.2 (commit 3d64ac2) / svtn_id hex-encoding fix. |
| 1.0 | 2026-07-15 | Initial draft — admission-state-sync push RPC: four write paths, `internal.admission.*` commands, push failure advisory (WARN, no rollback), `admitted=false` on load, full-snapshot on control startup, nil-syncer no-op. Authored per S-BL.ADMISSION-SYNC-WIRE BC groundwork item A1. |
