---
artifact_id: BC-2.07.001
document_type: behavioral-contract
level: L3
version: "1.2"
status: draft
producer: product-owner
timestamp: 2026-06-29T00:00:00
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
  - date: 2026-06-29
    version: "1.2"
    actor: product-owner
    change: >
      Task 2 reconverge (S-5.01 + S-6.02 Pass-1 adversarial): (1) Trigger updated to
      `sbctl admin svtn create/destroy`; (2) Canonical Test Vectors updated to
      `sbctl admin svtn create --name=mynet`; (3) Stories cell updated with PC split
      (PC-1: S-6.02 + S-6.07; PC-2: S-6.02; PC-3: S-6.05); (4) PC-2 "trust anchor"
      addendum added clarifying admitted=false bootstrap semantics.
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
3. Only control-role keys may create or destroy SVTNs.

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
