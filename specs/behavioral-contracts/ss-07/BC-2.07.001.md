---
artifact_id: BC-2.07.001
document_type: behavioral-contract
level: L3
version: "1.15"
status: draft
producer: product-owner
timestamp: 2026-06-30T00:00:00
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "e6efb60"
extracted_from: null
bc_id: BC-2.07.001
subsystem: network-management
architecture_module: internal/svtnmgmt
capability: CAP-023
priority: P2
criticality: high
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-07-12
    version: "1.15"
    actor: story-writer
    change: >
      Traceability Stories cell PC-4 (Status) filled: S-BL.CLI-SURFACE-COMPLETION — the distinct
      story-writer pass PO deferred at v1.14 PC-4 extension. Governance-only; no PC/AC behavior
      change.
  - date: 2026-07-12
    version: "1.14"
    actor: product-owner
    change: >
      S-BL.CLI-SURFACE-COMPLETION Ruling 2 (S-BL.CLI-SURFACE-COMPLETION-rulings.md):
      extend BC with new Postcondition PC-4 (Status) — wires `admin.svtn.status`,
      authority `resolveCallerAdmissionAnyRole` (any admitted role, mirroring
      BC-2.05.004 F-L2-003 list-keys precedent), response excludes session/health
      data (ARCH-09 purity boundary). Three new Canonical Test Vectors
      (happy-path, not-found E-SVTN-003, admission-denied E-ADM-009). Two new
      VP-048 sibling rows. No change to existing PC-1/PC-2/PC-3 or Invariants.
  - date: 2026-06-28
    version: "1.1"
    actor: product-owner
    change: >
      Initial draft — SVTN lifecycle create/destroy + bootstrap first control key.
  - date: 2026-06-29
    version: "1.2"
    actor: product-owner
    change: >
      Task 2 reconverge (S-5.01 + S-6.02 Pass-1 adversarial): (1) Trigger updated to
      `sbctl admin svtn create/destroy`; (2) Canonical Test Vectors updated to
      `sbctl admin svtn create --name=mynet`; (3) Stories cell updated with PC split
      (PC-1: S-6.02 + S-6.07; PC-2: S-6.02; PC-3: S-6.05); (4) PC-2 "trust anchor"
      addendum added clarifying admitted=false bootstrap semantics.
  - date: 2026-06-30
    version: "1.3"
    actor: product-owner
    change: >
      S-6.06 lens-3 F-003 ruling (Inv-3 scope): BC-2.07.001 Invariant 3 ("Only
      control-role keys may create or destroy SVTNs") is SVTN-lifecycle authority
      scoped to admin.svtn.* operations only. It does NOT extend authority scope
      to admin.key.* RPCs. S-6.06's caller-role enforcement is anchored exclusively
      on BC-2.05.004 PC-1 / DI-001 ("admission-control authority"). S-6.06 is
      dropped from BC-2.07.001's cite path; story-writer must remove BC-2.07.001
      from S-6.06 bc_traces frontmatter and BC table under
      bc_array_changes_propagate_to_body_and_acs policy.
  - date: 2026-07-01
    version: "1.4"
    actor: product-owner
    change: >
      Wave-6 Tranche-A Ruling-2: Inv-3 tightened to codify bootstrap-only caller restriction for
      `admin.svtn.create`. Cross-SVTN control-role keys are NOT authorized for `admin.svtn.create`.
      Only the daemon bootstrap key (seeded via PC-2 local operation) may invoke SVTN creation.
      Removes ambiguity flagged in S-6.07 F-P1L1-005.
  - date: 2026-07-01
    version: "1.5"
    actor: spec-steward
    change: >
      S-6.07 F-P2L3-003 test-vector gap closure: added Canonical Test Vector for cross-SVTN
      control-role key attempting `admin.svtn.create` — expects E-ADM-009 (not existence oracle)
      per Ruling-5 bootstrap-only guard. Handler MUST fire `IsBootstrapKey` check before
      `resolveAndVerifyCallerRole` to prevent SVTN existence leak.
  - date: 2026-07-01
    version: "1.6"
    actor: spec-steward
    change: >
      Ruling-7 defense-in-depth (Pass-3 L3 handoff): added note to Invariant 3 — implementations
      MUST check `caller.role == RoleControl` explicitly after `IsBootstrapKey(caller)`. Defense-in-depth
      against future bootstrap-key rotation or provisioning refactors that might inadvertently relax the
      role assignment. VP-048 updated to v1.4 referencing this note.
  - date: 2026-07-01
    version: "1.7"
    actor: spec-steward
    change: >
      Ruling-8: narrow Inv-3 DiD check to non-genesis path; genesis creation exempt (bootstrap key
      registered as RoleControl atomically in genesis Create). Genesis carve-out appended to Inv-3
      defense-in-depth note. Ref: rulings v1.3 Ruling-8.
  - date: 2026-07-01
    version: "1.8"
    actor: spec-steward
    change: >
      F-P5L3-03 cleanup: backfill missing modified-list entries for v1.1, v1.4, v1.5 (previously absent);
      reorder modified list chronologically (v1.1 through v1.8). Add genesis-path Canonical Test Vector
      (F-P5L3-06). No behavioral changes.
  - date: 2026-07-01
    version: "1.9"
    actor: spec-steward
    change: >
      Ruling-11 wire-envelope audit (decisions/wave-6-tranche-a-scope-rulings.md v1.6): no changes;
      canonical test vectors specify CLI-level message format, not wire-envelope JSON shape. Wire-envelope
      contract formalized in S-6.07 §Wire Envelope Contract. No behavioral changes.
  - date: 2026-07-01
    version: "1.10"
    actor: spec-steward
    change: >
      Ruling-12 §5 hygiene (F-P7L3R2-02): reorder modified-list entries chronologically ascending
      (v1.6 → v1.7 → v1.8 → v1.9); v1.8 and v1.9 were previously swapped. No behavioral changes.
  - date: 2026-07-01
    version: "1.11"
    actor: spec-steward
    change: >
      RULING-W6TB-A clarification: explicitly scope Inv-3 bootstrap-only restriction to
      admin.svtn.create only; add destroy authority note (admin.svtn.destroy uses
      resolveAndVerifyCallerRole general control-role gate, returns E-ADM-009 at RPC layer
      wrapped in E-RPC-011; ErrDestroyUnauthorized / E-ADM-011 Variant 2 is Go-API-layer
      defense-in-depth only). Genesis re-open after destroying last SVTN is permitted
      (recovery semantics, not privilege escalation). Add two canonical test vectors for
      destroy authority. Add PC/EC entries citing destroy authority and genesis-recreate
      carve-out. Update Traceability Stories row to reflect S-6.05 v1.3 anchoring. No
      behavioral changes — clarification only.
  - date: 2026-07-02
    version: "1.12"
    actor: spec-steward
    change: >
      F-P3L3-M-05: Sync Stories-row narrative — S-6.05 anchor updated v1.3 → v1.5
      (v1.4 was a reverted spec regression against RULING-W6TB-A; v1.5 restores
      Destroy(caller admission.AdmittedKey, svtnName string) signature and Go-API
      defense-in-depth check). No behavioral changes.
  - date: 2026-07-02
    version: "1.13"
    actor: spec-steward
    change: >
      F-P4L3-MED-2 (POL-002): Traceability Stories row cite S-6.05 v1.5 → v1.7
      (this fix-burst bumps story to v1.7). Governance-only.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-023]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.07.001: Control Node Creates and Destroys SVTNs; First Control Key Bootstrapped Locally

## Description

A control node creates a new SVTN by generating a SVTN ID, establishing the initial admitted key set (starting with the control node's own key), and registering the SVTN with the router. The first control key is bootstrapped locally on the E router without requiring a pre-existing SVTN. SVTN destruction removes the SVTN and all associated admitted keys from the router.

## Preconditions

1. For SVTN creation: The E router is running; no SVTN with the proposed ID exists.
2. For bootstrap: The control node's keypair is present locally.
3. For SVTN destruction: The control node has management authority for the SVTN.

## Postconditions

1. **Create**: New SVTN ID registered on the router; control node's key added as the first admitted control-role key; SVTN is ready for additional key registrations and node admissions.
2. **Bootstrap**: The first control key is added to the router's admitted set via a local operation (no network admission required — this is the trust anchor). **Trust anchor semantics (F-CS-004 addendum):** the key is initially registered with `admitted=false`; the control node completes the standard challenge-response admission protocol to flip its own key to `admitted=true`. This means the bootstrap path is not a privilege bypass — it merely seeds the admitted-key set so that the challenge-response can proceed. The mechanism must be documented and auditable.
3. **Destroy**: All admitted keys for the SVTN are removed from the router; all active sessions on that SVTN are terminated; SVTN ID is freed.
4. **Status (Ruling 2, S-BL.CLI-SURFACE-COMPLETION-rulings.md, 2026-07-12):** Returns the SVTN's `svtn_id` (hex), `name`, `created_at`, and admitted-key counts grouped by role. Wired as `admin.svtn.status`, registered in `BuildAdminHandlers` (control-mode-daemon-only, same as create/destroy — router/access/console pass nil admin handlers per ADR-004 and correctly return E-RPC-010). Authority: any admitted role in the target SVTN, OR operator-set member, OR bootstrap key (`resolveCallerAdmissionAnyRole`, mirroring BC-2.05.004 F-L2-003 list-keys precedent) — the CONTROL-only AUTHORITY gate is skipped but the ADMISSION gate still applies (CWE-862 defense against cross-SVTN roster/existence enumeration; same reasoning as BC-2.05.004 EC-008). Does **not** include active-session or health data — out of the control-mode daemon's accessible state (ARCH-09 purity boundary; `internal/session` is a forbidden import for `cmd/switchboard/admin_handlers.go`; no health indicator is proposed for the same reason — there is no accessible signal at this boundary to compute one from). Not-found is E-SVTN-003 (reuse the existing `mapAdminError` `ErrSVTNNotFound` arm). CLI: `sbctl svtn status --name=<svtn-name>` — a genuine standalone top-level dispatch (not routed through `sbctl admin` framing), since status is read-only/non-destructive and carries none of the confirm-gate duplication risk that motivates the `svtn destroy` migration-shim disposition (see interface-definitions.md §60/§62).

## Invariants

1. **DI-012**: The control node manages SVTN lifecycle as a participant in the user/data plane. It does not have privileged access to router internals.
2. **DI-005**: SVTN IDs must be globally unique within the router's scope; duplicate SVTN IDs are rejected.
3. Only control-role keys may create or destroy SVTNs. **Scope:** this invariant governs `admin.svtn.*` operations only. Authority enforcement for `admin.key.*` operations is governed by BC-2.05.004 PC-1 / DI-001, not this contract. **Bootstrap-only restriction for `admin.svtn.create` (S-6.07 F-P1L1-005 closure):** `admin.svtn.create` requires the daemon bootstrap key as the authorized caller. Cross-SVTN control-role keys (i.e., control keys admitted through the standard challenge-response path on a different SVTN) are NOT authorized to invoke `admin.svtn.create`. Only the bootstrap key — seeded via the local-operation path (PC-2) — may trigger SVTN creation. This removes the ambiguity in which any control-role key might be assumed to have `admin.svtn.create` authority. Post-create, additional control keys may be registered via `admin.key.register`, but they cannot themselves create further SVTNs on this daemon. **Defense-in-depth note (Ruling-7, 2026-07-01):** Implementations MUST check `caller.role == RoleControl` explicitly after `IsBootstrapKey(caller)` returns true. Even though the bootstrap key is provisioned with `RoleControl` by construction, an explicit role check is required as defense-in-depth against future bootstrap-key rotation or provisioning refactors that might inadvertently relax the role assignment. A caller passing the bootstrap-key check with `role != RoleControl` MUST be rejected with E-ADM-009 before any SVTN state is consulted (existence oracle closed). **Genesis Carve-Out (Ruling-8, 2026-07-01):** On the first-ever SVTN creation (when `HasAnySVTN() == false`), no keySet entry yet exists for the bootstrap key. On that path the `IsBootstrapKey(caller)` check alone suffices, and the bootstrap key is registered as `RoleControl` by the `Create()` call as part of the genesis atomic operation. The `role == RoleControl` explicit check applies only when SVTN state exists (non-genesis path). **Destroy authority (S-6.05 W6TB-A Ruling, 2026-07-01):** `admin.svtn.destroy` does NOT require the daemon bootstrap key. Any admitted control-role key may invoke destroy. The handler gate fires `resolveAndVerifyCallerRole` (returning E-ADM-009 to non-control callers at the RPC layer, wrapped in E-RPC-011); `SVTNManager.Destroy` applies `ErrDestroyUnauthorized` (E-ADM-011 Variant 2) as a defense-in-depth Go-API check only. `admin.svtn.create` is the special case that bypasses `resolveAndVerifyCallerRole` in favor of the stricter bootstrap-only gate; destroy does not share this exception. Destroying the last SVTN causes `HasAnySVTN()` to return false, which re-opens the genesis carve-out for a subsequent `admin.svtn.create` — this is permitted recovery semantics, not privilege escalation (the daemon is in a zero-SVTN, zero-admitted-key state; the bootstrap key is the trust anchor for re-initialization).

## Trigger

Operator runs `sbctl admin svtn create` or `sbctl admin svtn destroy` or equivalent management RPC.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | `sbctl svtn create` with a SVTN ID that already exists | E-SVTN-001 "SVTN already exists: <id>"; no action taken. |
| EC-002 | `sbctl svtn destroy` with active sessions | Sessions terminated with notification; then SVTN destroyed. Nodes receive session-terminated signals. |
| EC-003 | Bootstrap: multiple control nodes attempt to bootstrap the same router simultaneously | First write wins; second bootstrap attempt returns E-SVTN-002 "SVTN bootstrap already complete". |
| EC-004 | SVTN destroyed by control node on a PE router with multiple nodes | Termination signals sent to all admitted nodes; propagation to other routers in the SVTN topology. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl admin svtn create --name=mynet` | SVTN created; SVTN ID returned; bootstrap fingerprint returned; control key in admitted set | happy-path |
| `sbctl admin svtn destroy --name=mynet` | All admitted keys removed; all sessions terminated; SVTN ID freed | happy-path |
| `sbctl admin svtn create --name=mynet` (already exists) | E-SVTN-001 "SVTN already exists: mynet" | error |
| Bootstrap: first control key added locally | Control key in admitted set with `admitted=false`; control node completes challenge-response to flip `admitted=true`; SVTN ready for additional key registrations | happy-path |
| Non-control-role key attempts `sbctl admin svtn create` | E-ADM-009 "insufficient authority for operation admin.svtn.create: key <fp> has role <role>" | error |
| Cross-SVTN control-role key (admitted to an existing SVTN, role=control) attempts `sbctl admin svtn create` targeting a new SVTN name | E-ADM-009 "insufficient authority for operation admin.svtn.create: key <fp> has role control" — handler returns E-ADM-009 before consulting SVTN state; bootstrap-only check fires first; no existence oracle leak (Ruling-5, S-6.07 F-P2L3-003) | error |
| Genesis SVTN create: `HasAnySVTN()==false`, caller==bootstrap key | Create succeeds; SVTN ID returned; control-role bootstrap key registered atomically in admitted set (`admitted=false`, flipped to `admitted=true` via challenge-response); Inv-3 genesis carve-out applies — `role == RoleControl` keySet lookup correctly skipped on genesis path (Ruling-8). Reference: decisions/wave-6-tranche-a-scope-rulings.md Ruling-8. | happy-path |
| Non-bootstrap control-role key invokes `sbctl admin svtn destroy --name=mynet` | SVTN destroyed; all admitted keys removed; all active sessions terminated; confirmation returned — handler uses `resolveAndVerifyCallerRole` gate (not bootstrap-only check) (W6TB-A) | happy-path |
| Non-control (console/readonly) key invokes `admin.svtn.destroy` via mgmt RPC | E-RPC-011 wrapping E-ADM-009 "insufficient authority for operation admin.svtn.destroy: key <fp> has role <role>"; SVTN unchanged — `resolveAndVerifyCallerRole` fires at handler layer before `SVTNManager.Destroy` is reached (W6TB-A) | error |
| `sbctl svtn status --name=mynet` (caller admitted to `mynet` in any role) | `{"svtn_id":"<hex>","name":"mynet","created_at":"<RFC3339>","key_counts":{"control":1,"console":0,"access":2}}`; exit code 0 (Ruling 2, PC-4) | happy-path |
| `sbctl svtn status --name=doesnotexist` | E-SVTN-003 "SVTN not found: doesnotexist"; exit code 1 (Ruling 2, PC-4) | error |
| `sbctl svtn status --name=mynet` (caller has a valid operator key admitted only to a DIFFERENT SVTN, not `mynet`, not operator-set, not bootstrap) | E-ADM-009 "insufficient authority for operation admin.svtn.status: key <fp> has role <role>"; SVTN roster/existence not disclosed — admission gate fires before status is computed (Ruling 2, PC-4; CWE-862 defense, mirrors BC-2.05.004 EC-008) | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-048 | SVTN create is idempotent for the first invocation; error on duplicate | unit |
| VP-048 | SVTN destroy removes all admitted keys | integration |
| VP-048 | Only control-role keys can create/destroy SVTNs | integration |
| VP-048 | `admin.svtn.status` returns accurate admitted-key counts grouped by role, scoped to the target SVTN only, with no session/health fields present in the response | integration |
| VP-048 | `admin.svtn.status` admission-gate enforcement: any admitted role (control/console/access) in the target SVTN, OR operator-set member, OR bootstrap key succeeds; a caller with no admission relationship to the target SVTN is denied E-ADM-009 | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023 |
| L2 Domain Invariants | DI-012 (control node is a network participant, not a router manager), DI-005 (SVTN cryptographic isolation) |
| Architecture Module | internal/svtnmgmt |
| Stories | PC-1 (Create): S-6.02 (SVTNManager Go method), S-6.07 (CLI + handler, RPC reachability); PC-2 (Bootstrap): S-6.02 (local side-effect of Create); PC-3 (Destroy): S-6.05 v1.8 (CLI + handler — Wave 6, depends_on [S-6.02, S-6.07]; CR-009 ruling 2026-06-29; W6TB-A destroy authority model); PC-4 (Status): S-BL.CLI-SURFACE-COMPLETION |
| Capability Anchor Justification | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023 — this BC specifies the create/destroy lifecycle that CAP-023 defines as the prerequisite for all other operations |

## Related BCs

- BC-2.05.001 — depends on: SVTN must exist before admission is possible
- BC-2.07.002 — composes with: sbctl is the operator interface for SVTN management

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.15 | 2026-07-12 | story-writer | Traceability Stories cell PC-4 (Status) filled: `S-BL.CLI-SURFACE-COMPLETION` — the distinct story-writer pass PO deferred at v1.14 PC-4 extension. Governance-only; no PC/AC behavior change. |
| 1.14 | 2026-07-12 | product-owner | S-BL.CLI-SURFACE-COMPLETION Ruling 2 (`S-BL.CLI-SURFACE-COMPLETION-rulings.md`): extend BC with new Postcondition PC-4 (Status) — wires `admin.svtn.status`, registered in `BuildAdminHandlers` (control-mode-only). Authority: any admitted role in the target SVTN, OR operator-set member, OR bootstrap key (`resolveCallerAdmissionAnyRole`, list-keys precedent, BC-2.05.004 F-L2-003) — authority gate bypassed, admission gate retained. Response schema (`svtn_id`, `name`, `created_at`, `key_counts`) deliberately excludes session/health data — ARCH-09 purity boundary, `internal/session` is a forbidden import for `cmd/switchboard/admin_handlers.go`. Not-found reuses E-SVTN-003. CLI dispatch is `sbctl svtn status --name=<svtn-name>` (bare top-level, not `sbctl admin`-prefixed — read-only, no confirm-gate duplication risk). Three new Canonical Test Vectors added (happy-path, not-found, admission-denied). Two new VP-048 sibling rows added. No change to PC-1/PC-2/PC-3 or Invariants. |
| 1.13 | 2026-07-02 | spec-steward | F-P4L3-MED-2 (POL-002): Traceability Stories row cite S-6.05 v1.5 → v1.7 (this fix-burst bumps story to v1.7). Governance-only. [governance_leaf: true — downstream story/VP pins DO NOT need to re-sync per drbothen/vsdd-factory#429 draft policy] |
| 1.12 | 2026-07-02 | spec-steward | F-P3L3-M-05: Sync Stories-row narrative — S-6.05 anchor updated v1.3 → v1.5 (v1.4 was reverted spec regression against RULING-W6TB-A; v1.5 restores Destroy(caller admission.AdmittedKey, svtnName string) signature and Go-API defense-in-depth check). No behavioral changes. |
| 1.11 | 2026-07-01 | spec-steward | RULING-W6TB-A (decisions/RULING-W6TB-A-svtn-destroy-authority.md): add destroy authority clarification to Inv-3 — admin.svtn.destroy uses resolveAndVerifyCallerRole general control-role gate (not bootstrap-only); E-ADM-009 at RPC handler layer wrapped in E-RPC-011; ErrDestroyUnauthorized / E-ADM-011 Variant 2 is Go-API-layer defense-in-depth only. Genesis re-open after last-SVTN destroy is permitted recovery semantics. Add two canonical test vectors for destroy authority (non-bootstrap control-role happy-path; non-control error path). Update Traceability Stories row to cite S-6.05 v1.3 anchoring. No behavioral changes — clarification and explicit scoping only. |
| 1.10 | 2026-07-01 | spec-steward | Ruling-12 §5 hygiene (F-P7L3R2-02): reorder modified-list entries chronologically ascending (v1.6 → v1.7 → v1.8 → v1.9); v1.8 and v1.9 were previously swapped in the frontmatter modified: list. No behavioral changes. |
| 1.9 | 2026-07-01 | spec-steward | Ruling-11 wire-envelope audit (decisions/wave-6-tranche-a-scope-rulings.md v1.6): no changes; canonical test vectors specify CLI-level message format (`E-ADM-009 "insufficient authority..."`, `E-SVTN-001 "SVTN already exists: <name>"`), not wire-envelope JSON shape. Wire-envelope contract formalized in S-6.07 §Wire Envelope Contract; not needed here. No behavioral changes. |
| 1.8 | 2026-07-01 | spec-steward | F-P5L3-03/F-P5L3-06 cleanup: backfill modified-list entries for v1.1, v1.4, v1.5 (previously absent); reorder modified list chronologically; add genesis-path Canonical Test Vector (HasAnySVTN()==false, caller==bootstrap → Create succeeds, control-role bootstrap key registered atomically, Inv-3 genesis carve-out per Ruling-8). No behavioral changes. |
| 1.7 | 2026-07-01 | spec-steward | Ruling-8: narrow Inv-3 DiD check to non-genesis path; genesis creation exempt (bootstrap key registered as RoleControl atomically in genesis Create). Ref: rulings v1.3 Ruling-8. |
| 1.6 | 2026-07-01 | spec-steward | Ruling-7 defense-in-depth (Pass-3 L3 handoff): Inv-3 extended with explicit `caller.role == RoleControl` check obligation after `IsBootstrapKey(caller)`. Defense-in-depth against bootstrap-key rotation refactors. Callers passing bootstrap-key check with `role != RoleControl` MUST be rejected E-ADM-009 before SVTN state consulted (existence oracle closed). VP-048 updated to v1.4. |
| 1.5 | 2026-07-01 | spec-steward | S-6.07 F-P2L3-003 test-vector gap closure: added Canonical Test Vector for cross-SVTN control-role key attempting `admin.svtn.create` — expects E-ADM-009 (not existence oracle) per Ruling-5 bootstrap-only guard. Handler MUST fire `IsBootstrapKey` check before `resolveAndVerifyCallerRole` to prevent SVTN existence leak. |
| 1.4 | 2026-07-01 | product-owner | Wave-6 Tranche-A Ruling-2: Inv-3 tightened to codify bootstrap-only caller restriction for `admin.svtn.create`. Cross-SVTN control-role keys are NOT authorized for `admin.svtn.create`. Only the daemon bootstrap key (seeded via PC-2 local operation) may invoke SVTN creation. Removes ambiguity flagged in S-6.07 F-P1L1-005. |
| 1.3 | 2026-06-30 | product-owner | S-6.06 lens-3 F-003 ruling (Inv-3 scope): Inv-3 scoped to `admin.svtn.*` only; does not extend to `admin.key.*`; S-6.06 removed from cite path; story-writer to drop BC-2.07.001 from S-6.06 bc_traces. |
| 1.2 | 2026-06-29 | product-owner | Task 2 reconverge (S-5.01 + S-6.02 Pass-1 adversarial): Trigger updated to `sbctl admin svtn create/destroy`; test vectors updated; Stories cell updated with PC split; PC-2 trust-anchor addendum added. |
| 1.1 | 2026-06-28 | product-owner | Initial draft — SVTN lifecycle create/destroy + bootstrap first control key. |
