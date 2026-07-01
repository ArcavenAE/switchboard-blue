---
artifact_id: BC-2.07.001
document_type: behavioral-contract
level: L3
version: "1.9"
status: draft
producer: product-owner
timestamp: 2026-06-30T00:00:00
phase: 1a
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
    version: "1.9"
    actor: spec-steward
    change: >
      Ruling-11 wire-envelope audit (decisions/wave-6-tranche-a-scope-rulings.md v1.6): no changes;
      canonical test vectors specify CLI-level message format, not wire-envelope JSON shape. Wire-envelope
      contract formalized in S-6.07 §Wire Envelope Contract. No behavioral changes.
  - date: 2026-07-01
    version: "1.8"
    actor: spec-steward
    change: >
      F-P5L3-03 cleanup: backfill missing modified-list entries for v1.1, v1.4, v1.5 (previously absent);
      reorder modified list chronologically (v1.1 through v1.8). Add genesis-path Canonical Test Vector
      (F-P5L3-06). No behavioral changes.
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

## Invariants

1. **DI-012**: The control node manages SVTN lifecycle as a participant in the user/data plane. It does not have privileged access to router internals.
2. **DI-005**: SVTN IDs must be globally unique within the router's scope; duplicate SVTN IDs are rejected.
3. Only control-role keys may create or destroy SVTNs. **Scope:** this invariant governs `admin.svtn.*` operations only. Authority enforcement for `admin.key.*` operations is governed by BC-2.05.004 PC-1 / DI-001, not this contract. **Bootstrap-only restriction for `admin.svtn.create` (S-6.07 F-P1L1-005 closure):** `admin.svtn.create` requires the daemon bootstrap key as the authorized caller. Cross-SVTN control-role keys (i.e., control keys admitted through the standard challenge-response path on a different SVTN) are NOT authorized to invoke `admin.svtn.create`. Only the bootstrap key — seeded via the local-operation path (PC-2) — may trigger SVTN creation. This removes the ambiguity in which any control-role key might be assumed to have `admin.svtn.create` authority. Post-create, additional control keys may be registered via `admin.key.register`, but they cannot themselves create further SVTNs on this daemon. **Defense-in-depth note (Ruling-7, 2026-07-01):** Implementations MUST check `caller.role == RoleControl` explicitly after `IsBootstrapKey(caller)` returns true. Even though the bootstrap key is provisioned with `RoleControl` by construction, an explicit role check is required as defense-in-depth against future bootstrap-key rotation or provisioning refactors that might inadvertently relax the role assignment. A caller passing the bootstrap-key check with `role != RoleControl` MUST be rejected with E-ADM-009 before any SVTN state is consulted (existence oracle closed). **Genesis Carve-Out (Ruling-8, 2026-07-01):** On the first-ever SVTN creation (when `HasAnySVTN() == false`), no keySet entry yet exists for the bootstrap key. On that path the `IsBootstrapKey(caller)` check alone suffices, and the bootstrap key is registered as `RoleControl` by the `Create()` call as part of the genesis atomic operation. The `role == RoleControl` explicit check applies only when SVTN state exists (non-genesis path).

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

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-048 | SVTN create is idempotent for the first invocation; error on duplicate | unit |
| VP-048 | SVTN destroy removes all admitted keys | integration |
| VP-048 | Only control-role keys can create/destroy SVTNs | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023 |
| L2 Domain Invariants | DI-012 (control node is a network participant, not a router manager), DI-005 (SVTN cryptographic isolation) |
| Architecture Module | internal/svtnmgmt |
| Stories | PC-1 (Create): S-6.02 (SVTNManager Go method), S-6.07 (CLI + handler, RPC reachability); PC-2 (Bootstrap): S-6.02 (local side-effect of Create); PC-3 (Destroy): S-6.05 (CLI + handler — Wave 6, depends_on S-6.02; CR-009 ruling 2026-06-29) |
| Capability Anchor Justification | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023 — this BC specifies the create/destroy lifecycle that CAP-023 defines as the prerequisite for all other operations |

## Related BCs

- BC-2.05.001 — depends on: SVTN must exist before admission is possible
- BC-2.07.002 — composes with: sbctl is the operator interface for SVTN management

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.9 | 2026-07-01 | spec-steward | Ruling-11 wire-envelope audit (decisions/wave-6-tranche-a-scope-rulings.md v1.6): no changes; canonical test vectors specify CLI-level message format (`E-ADM-009 "insufficient authority..."`, `E-SVTN-001 "SVTN already exists: <name>"`), not wire-envelope JSON shape. Wire-envelope contract formalized in S-6.07 §Wire Envelope Contract; not needed here. No behavioral changes. |
| 1.8 | 2026-07-01 | spec-steward | F-P5L3-03/F-P5L3-06 cleanup: backfill modified-list entries for v1.1, v1.4, v1.5 (previously absent); reorder modified list chronologically; add genesis-path Canonical Test Vector (HasAnySVTN()==false, caller==bootstrap → Create succeeds, control-role bootstrap key registered atomically, Inv-3 genesis carve-out per Ruling-8). No behavioral changes. |
| 1.7 | 2026-07-01 | spec-steward | Ruling-8: narrow Inv-3 DiD check to non-genesis path; genesis creation exempt (bootstrap key registered as RoleControl atomically in genesis Create). Ref: rulings v1.3 Ruling-8. |
| 1.6 | 2026-07-01 | spec-steward | Ruling-7 defense-in-depth (Pass-3 L3 handoff): Inv-3 extended with explicit `caller.role == RoleControl` check obligation after `IsBootstrapKey(caller)`. Defense-in-depth against bootstrap-key rotation refactors. Callers passing bootstrap-key check with `role != RoleControl` MUST be rejected E-ADM-009 before SVTN state consulted (existence oracle closed). VP-048 updated to v1.4. |
| 1.5 | 2026-07-01 | spec-steward | S-6.07 F-P2L3-003 test-vector gap closure: added Canonical Test Vector for cross-SVTN control-role key attempting `admin.svtn.create` — expects E-ADM-009 (not existence oracle) per Ruling-5 bootstrap-only guard. Handler MUST fire `IsBootstrapKey` check before `resolveAndVerifyCallerRole` to prevent SVTN existence leak. |
| 1.4 | 2026-07-01 | product-owner | Wave-6 Tranche-A Ruling-2: Inv-3 tightened to codify bootstrap-only caller restriction for `admin.svtn.create`. Cross-SVTN control-role keys are NOT authorized for `admin.svtn.create`. Only the daemon bootstrap key (seeded via PC-2 local operation) may invoke SVTN creation. Removes ambiguity flagged in S-6.07 F-P1L1-005. |
| 1.3 | 2026-06-30 | product-owner | S-6.06 lens-3 F-003 ruling (Inv-3 scope): Inv-3 scoped to `admin.svtn.*` only; does not extend to `admin.key.*`; S-6.06 removed from cite path; story-writer to drop BC-2.07.001 from S-6.06 bc_traces. |
| 1.2 | 2026-06-29 | product-owner | Task 2 reconverge (S-5.01 + S-6.02 Pass-1 adversarial): Trigger updated to `sbctl admin svtn create/destroy`; test vectors updated; Stories cell updated with PC split; PC-2 trust-anchor addendum added. |
| 1.1 | 2026-06-28 | product-owner | Initial draft — SVTN lifecycle create/destroy + bootstrap first control key. |
