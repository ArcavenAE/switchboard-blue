---
artifact_id: BC-2.07.001
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.07.001
subsystem: network-management
architecture_module: internal/svtnmgmt
capability: CAP-023
priority: P2
criticality: important
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified: []
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
2. **Bootstrap**: The first control key is added to the router's admitted set via a local operation (no network admission required — this is the trust anchor). The mechanism must be documented and auditable.
3. **Destroy**: All admitted keys for the SVTN are removed from the router; all active sessions on that SVTN are terminated; SVTN ID is freed.

## Invariants

1. **DI-012**: The control node manages SVTN lifecycle as a participant in the user/data plane. It does not have privileged access to router internals.
2. **DI-005**: SVTN IDs must be globally unique within the router's scope; duplicate SVTN IDs are rejected.
3. Only control-role keys may create or destroy SVTNs.

## Trigger

Operator runs `sbctl svtn create` or `sbctl svtn destroy` or equivalent API call.

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
| `sbctl svtn create --name=mynet` | SVTN created; SVTN ID returned; control key registered | happy-path |
| `sbctl svtn destroy --id=<svtn-id>` | All admitted keys removed; all sessions terminated; SVTN gone | happy-path |
| `sbctl svtn create --name=mynet` (already exists) | E-SVTN-001 "SVTN already exists" | error |
| Bootstrap: first control key added locally | Control key in admitted set; SVTN ready for node admissions | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | SVTN create is idempotent for the first invocation; error on duplicate | unit |
| VP-TBD | SVTN destroy removes all admitted keys | integration |
| VP-TBD | Only control-role keys can create/destroy SVTNs | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023 |
| L2 Domain Invariants | DI-012 (control node is a network participant, not a router manager), DI-005 (SVTN cryptographic isolation) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023 — this BC specifies the create/destroy lifecycle that CAP-023 defines as the prerequisite for all other operations |

## Related BCs

- BC-2.05.001 — depends on: SVTN must exist before admission is possible
- BC-2.07.002 — composes with: sbctl is the operator interface for SVTN management
