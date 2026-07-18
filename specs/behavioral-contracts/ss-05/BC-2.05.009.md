---
artifact_id: BC-2.05.009
document_type: behavioral-contract
level: L3
version: "1.5"
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
input-hash: "e16e6ce"
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
  - date: 2026-07-18
    version: "1.5"
    change: >
      Ruling 14 propagation (F-P7-01): PC-7b amended — add compensating best-effort
      internal.admission.revoke when expire fails AND expiry is already in the past, so
      router ends non-admissible (revoked, satisfying Invariant 6). Future-expiry expire-fail
      is PC-5 stale/missing (permitted) — no compensating action. Invariant 6 amended with
      explicit future-vs-past-expiry carve-out and AdmitNode no-expiry-check evidence
      (AdmitNode checks only revoked; only ReAuthenticate checks expiry). EC-010 updated:
      compensating revoke fires on past-expiry expire-fail; router ends non-admissible, never
      active-non-expiring. No wire format change; expire-handler key-not-found stays
      E-ADM-013; PC-5, PC-7c, Inv-4 svtn_id exemption unchanged.
  - date: 2026-07-17
    version: "1.4"
    change: >
      Ruling 13 propagation (F-P6-02): PC-7c amended — MUST NOT issue
      internal.admission.register for revoked entries; issue
      internal.admission.revoke only. Invariant 6 amended — explicit
      PC-5 vs Invariant 6 distinction: stale/missing (permitted) vs
      actively less-restrictive (forbidden). EC-009 updated to
      skip-register semantics: fresh router ends with no entry (absent
      = non-admissible = correct), not registered-then-revoked.
      No wire format change; PC-5, EC-008, EC-010, Inv-4 unchanged.
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

7. **Startup full-snapshot — COMPLETE authoritative state (v1.3):** On control startup,
   after `SVTNManager` is initialized, `PushFullSnapshot(ctx)` iterates all entries in
   the current `AdmittedKeySet` (via `ListBySVTN` across all SVTNs) and pushes each
   entry's **complete authoritative state** to each configured router endpoint. For every
   entry, the push sequence is:

   a. Issue `internal.admission.register` with `{svtn_id, pubkey_openssh, role}`.
   b. If the entry has a non-zero expiry timestamp, issue `internal.admission.expire` with
      the **original expiry timestamp** (whether in the past or future). The register+expire
      pair MUST be issued regardless of whether the expiry timestamp is past or future. If
      `internal.admission.expire` fails (after retry exhaustion) AND the expiry timestamp is
      already in the past at the time of failure, issue a best-effort
      `internal.admission.revoke` as a compensating action to leave the router in a
      non-admissible state. The compensating revoke is advisory-continue on failure (WARN
      log). If expire fails for a FUTURE-expiry entry, no compensating action is required —
      the router holds the key active, which correctly represents control's current state
      (the key is active now; the staleness manifests only after the expiry timestamp elapses,
      which is PC-5 permitted stale/missing, not an Invariant 6 violation). If register (step
      a) fails, skip expire entirely (the router has no entry for this key; the expire RPC
      would return E-ADM-013).
   c. **For REVOKED entries (Ruling 13 / F-P6-02):** do NOT issue
      `internal.admission.register`. A fresh router has no entry for the revoked key,
      which is the correct non-admissible terminal state. If the router already holds an
      entry for the key (partial prior push or pre-revocation registration), issue
      `internal.admission.revoke` with `{svtn_id, pubkey_openssh, role, confirm}` to
      mark it revoked. In either case, the router MUST NOT be left with the key
      registered-as-active. The register+revoke two-RPC pattern is PROHIBITED for
      revoked entries because a partial failure (register succeeds, revoke fails) leaves
      the router in a less-restrictive state, violating Invariant 6. The router's revoke
      handler MUST treat "key not present" as success — absent is the correct
      non-admissible terminal state for a revoked key.

   **Semantic-equivalence end-state postcondition:** After `PushFullSnapshot` completes
   (best-effort across all endpoints), the router's `AdmittedKeySet` is SEMANTICALLY
   EQUIVALENT to control's current `AdmittedKeySet`: same keys, same roles, same revoked
   status, same expiries. The router MUST NOT be left with any key in a LESS-RESTRICTIVE
   state than control holds — no un-revoking a revoked key, no un-expiring an expired key,
   no silently re-registering a revoked key as active.

   **EC-007 resync guarantee — conditional on `control_admission_state_file` (Ruling 11):**
   The EC-007 resync guarantee holds ONLY when `control_admission_state_file` is configured
   in the control daemon's config (BC-2.09.003 PC-15) AND the snapshot was successfully
   written on each prior mutation (write-on-mutation path, Ruling 11). The mechanism is:
   on startup, control loads its persisted keyset from `control_admission_state_file` via
   `loadSnapshotFromFile` BEFORE constructing the sync client and calling
   `PushFullSnapshot` — so the full-snapshot push carries the recovered keys including
   their revoked status and expiry timestamps.
   When `control_admission_state_file` is absent or empty, control constructs a fresh
   empty `AdmittedKeySet` and `PushFullSnapshot` immediately hits the empty-keyset
   early-return — no keys are pushed and EC-007 is inert for that startup. Operators
   who require EC-007 MUST configure `control_admission_state_file`.

   **Control-side persist-write (synchronous, before push dispatch):** Each committed
   `admin.key.*` / `admin.svtn.*` mutation also triggers `writeSnapshotAtomic(path, ks)`
   synchronously on the handler goroutine, BEFORE `dispatchPush`. Write failure is
   advisory (WARN log); it does NOT roll back the `m.*` write and does NOT fail the RPC
   response. This is the same advisory policy as the router-side snapshot write
   (BC-2.05.010 PC-2 / Ruling 3). The atomic snapshot write reuses the existing
   `writeSnapshotAtomic` implementation in `admission_sync_snapshot.go`.

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
6. **Full-snapshot must not leave a router in a less-restrictive state (security-correctness
   invariant):** `PushFullSnapshot` MUST NOT produce a router state that is less restrictive
   than control's authoritative state. Specifically:
   - A key that is REVOKED in control MUST be revoked on the router after the snapshot push;
     it MUST NOT be left registered-as-active.
   - A key that is EXPIRED (expiry timestamp in the past) in control MUST be expired/inactive
     on the router after the snapshot push; it MUST NOT be left registered-as-non-expiring.
   Violation of this invariant silently un-revokes or un-expires a key on every control
   restart, defeating the durability guarantees of `admin.key.revoke` and
   `admin.key.expire` — and becomes exploitable once challenge-response
   (`S-BL.NODE-IDENTIFY-WIRE`) is wired.

   **PC-5 vs Invariant 6 distinction (Ruling 13):** PC-5's advisory-push carve-out covers
   "router is missing an update" — the router did not receive a push (e.g. due to a push
   failure) and remains in its PRIOR state. For a key it never had, that prior state is
   ABSENT = non-admissible. This is PERMITTED by PC-5 (stale, but not less restrictive
   than before). It does NOT cover "router was actively made less restrictive by a partial
   push." The two states are distinct:
   - **Stale / missing** — router did not receive an update; remains in prior state
     (absent or as last known). This is the PC-5 carve-out. PERMITTED.
   - **Actively less-restrictive** — router was MADE admissible for a key that is
     revoked/expired in control (e.g., register succeeded, revoke failed in a two-RPC
     sequence). This is NOT covered by PC-5. FORBIDDEN by Invariant 6, absolutely.
   PC-5 tolerates a router MISSING updates. Invariant 6 forbids a push (or partial push)
   that leaves a router MORE permissive than before for a revoked/expired key. Approach (c)
   — skip register for revoked entries — satisfies both: a partial failure on the revoke-only
   RPC cannot leave a revoked key active, because the register was never issued.

   **Future-vs-past-expiry carve-out (Ruling 14):** The past-vs-future-expiry distinction
   matters for the Invariant 6 boundary. A key whose expiry is in the PAST is expired/inactive
   in control at push time — leaving it registered-active on the router is immediately less
   restrictive (Invariant 6 violation). A key whose expiry is in the FUTURE is currently active
   in control at push time — the router's active-no-expiry state correctly represents the
   current admission status; the staleness only materializes after the expiry elapses (PC-5
   stale/missing, not Invariant 6). AdmitNode does NOT enforce expiry (it only checks
   `revoked`; `IsAdmitted` checks `admitted && !revoked`); only `ReAuthenticate` checks
   expiry. This confirms that a past-expiry key left active-and-non-expiring on the router is
   IMMEDIATELY exploitable at the initial admission handshake — sharpening the Invariant 6
   obligation for past-expiry partial failure. The compensating revoke in PC-7b enforces this
   obligation when expire fails for a past-expiry entry.

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
| EC-007 | Control restarts after prior push failures | `PushFullSnapshot(ctx)` on startup pushes the COMPLETE authoritative `AdmittedKeySet` state — including revoked status and expiry timestamps — to all configured routers, converging router state to semantic equivalence with control. "Correcting any staleness" means full state convergence, not merely re-adding active keys. |
| EC-008 | Router is temporarily restarting when push arrives | Push fails (connection refused); WARN logged. Router loads from its VLR-local snapshot (BC-2.05.010) on restart and receives a fresh full-snapshot push when control next starts or the next per-write push arrives. |
| EC-009 | Control restarts; keyset contains a REVOKED key | `PushFullSnapshot` MUST NOT issue `internal.admission.register` for the revoked entry. If the router already holds an entry for the key (partial prior push or pre-revocation registration), issue `internal.admission.revoke` to mark it revoked. If the router has no entry for the key (fresh router), issue nothing — absent = non-admissible = correct terminal state. The router's resulting state is either "absent" or "revoked," both non-admissible, satisfying Invariant 6. The register+revoke two-RPC pattern is PROHIBITED for revoked entries (Ruling 13 / F-P6-02). |
| EC-010 | Control restarts; keyset contains a key with a past-expiry timestamp | `PushFullSnapshot` issues `internal.admission.register` + `internal.admission.expire(originalExpiry)` regardless of whether the expiry is past or future. If `expire` fails after retry exhaustion AND the original expiry timestamp is in the past at the time of failure, `PushFullSnapshot` issues a best-effort `internal.admission.revoke` as a compensating action (router ends non-admissible: revoked or absent). If expire succeeds, the router marks the entry expired/inactive. The router MUST NOT be left with a past-expiry key registered-active-and-non-expiring (Invariant 6). If expire fails for a FUTURE-expiry entry, no compensating action is needed — the router's active-no-expiry state correctly represents control's current state (PC-5 stale/missing). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `admin.key.register` succeeds on control; push to router succeeds | Router's `AdmittedKeySet` has the entry; `admitted=false` | happy-path |
| `admin.key.register` succeeds on control; push to router fails (no listener) | Control write committed; WARN logged; sbctl reports success; router is stale | push-failure isolation |
| `admin.key.revoke` on control; push succeeds | Router marks key revoked | happy-path |
| `admin.key.expire` on control; push succeeds | Router's keyset has non-zero expiry for the entry | happy-path |
| `admin.svtn.destroy` on control; push succeeds | Router has no entries for that SVTN | happy-path |
| Control startup with populated `AdmittedKeySet` (all active keys); push to router | Router keyset matches control after `PushFullSnapshot` — same keys, same roles, same expiries | startup-snapshot |
| Control startup; keyset has a REVOKED key; push to router | Router has the key with revoked=true after `PushFullSnapshot` — NOT registered as active | startup-revoked-key |
| Control startup; keyset has a key with past expiry timestamp T; push to router | Router has the key with expiry=T (expired/inactive) — NOT registered as active-and-non-expiring | startup-past-expiry |
| `admissionSyncer` is nil (router/console mode) | No push attempted; no error; existing handler behavior unchanged | nil-syncer |
| Push to multiple endpoints; one fails, one succeeds | Both attempted; failure logged for the failing endpoint; no rollback | multi-endpoint |

## Verification Properties

| VP-NNN | Property | Proof Method | Notes |
|--------|----------|-------------|-------|
| test-as-evidence | After `admin.key.register` on control, router's `AdmittedKeySet` has the entry (push succeeded) | integration (two in-process `mgmt.Server` instances) | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Push failure does not roll back control write; sbctl reports success | integration | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | Nil `admissionSyncer` → no-op, no panic | unit | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | `PushFullSnapshot` on control startup pushes all keyset entries to configured routers | integration | S-BL.ADMISSION-SYNC-WIRE AC |
| test-as-evidence | `PushFullSnapshot` with a REVOKED key in control's keyset: router has key with revoked=true after push (EC-009) | integration | security-correctness — revocation durability |
| test-as-evidence | `PushFullSnapshot` with a past-expiry key in control's keyset: router has key as expired/inactive, not active-and-non-expiring (EC-010) | integration | security-correctness — expiry durability |
| test-as-evidence | `PushFullSnapshot` result: router AdmittedKeySet is semantically equivalent to control's (same keys, same roles, same revoked status, same expiries) — Invariant 6 | integration | semantic-equivalence end-state |
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
| 1.5 | 2026-07-18 | Ruling 14 propagation (F-P7-01): PC-7b amended — add compensating best-effort `internal.admission.revoke` when `expire` fails AND expiry is already in the past, so router ends non-admissible (revoked, satisfying Invariant 6). Future-expiry expire-fail is PC-5 stale/missing (permitted) — no compensating action. Invariant 6 amended with explicit future-vs-past-expiry carve-out and AdmitNode no-expiry-check evidence (AdmitNode checks only `revoked`; only ReAuthenticate checks expiry). EC-010 updated: compensating revoke fires on past-expiry expire-fail; router ends non-admissible, never active-non-expiring. No wire format change; expire-handler key-not-found stays E-ADM-013; PC-5, PC-7c, Inv-4 svtn_id exemption unchanged. |
| 1.4 | 2026-07-17 | Ruling 13 propagation (F-P6-02): PC-7c amended — MUST NOT issue `internal.admission.register` for revoked entries; issue `internal.admission.revoke` only (router treats "key not found" as success — absent = non-admissible = correct terminal state). The register+revoke two-RPC pattern is PROHIBITED for revoked entries. Invariant 6 amended — explicit PC-5 vs Invariant 6 distinction: "stale/missing" (router did not receive update, permitted by PC-5) vs "actively less-restrictive" (router was made more permissive by a partial push, forbidden absolutely by Invariant 6). EC-009 updated to skip-register semantics: fresh router ends with no entry for the revoked key (absent), not registered-then-revoked; existing-entry router ends revoked. No wire format change; PC-5, EC-008, EC-010, Inv-4 svtn_id exemption unchanged. |
| 1.3 | 2026-07-17 | PC-7: full-snapshot push must propagate COMPLETE authoritative state including revoked status and expiry timestamps (not register-only). Router must converge to semantic equivalence with control (same keys, same roles, same revoked status, same expiries). Add Invariant 6 (security-correctness: PushFullSnapshot MUST NOT leave a router in a less-restrictive state than control holds — no un-revoking, no un-expiring). Add EC-009 (revoked key on restart: register then revoke, not register-as-active). Add EC-010 (past-expiry key on restart: register + expire(originalExpiry), not active-and-non-expiring). Update EC-007 wording: "correcting any staleness" means full state convergence including revocations. Add test vectors and VPs for revoked-key and past-expiry scenarios. Fixes F-1 (adversary pass 5): revoked key was silently resurrected as active on every control restart. |
| 1.2 | 2026-07-17 | Amend PC-7 (startup full-snapshot): add qualifier that EC-007 resync guarantee holds ONLY when `control_admission_state_file` is configured (BC-2.09.003 PC-15); without it, `PushFullSnapshot` pushes an empty keyset and EC-007 is inert. Document control-side persist-write: synchronous `writeSnapshotAtomic` call on each committed `admin.key.*`/`admin.svtn.*` mutation, BEFORE `dispatchPush`, advisory on failure. Per Ruling 11 (F-P3-01). |
| 1.1 | 2026-07-16 | Amend Invariant 4: exempt `svtn_id` from encoding-parity rule — on the `internal.admission.*` wire it is the 32-lowercase-hex-char encoding of the `[16]byte` SVTN UUID (not the human-readable name). Rulings v1.2 (commit 3d64ac2) / svtn_id hex-encoding fix. |
| 1.0 | 2026-07-15 | Initial draft — admission-state-sync push RPC: four write paths, `internal.admission.*` commands, push failure advisory (WARN, no rollback), `admitted=false` on load, full-snapshot on control startup, nil-syncer no-op. Authored per S-BL.ADMISSION-SYNC-WIRE BC groundwork item A1. |
