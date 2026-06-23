---
artifact_id: BC-2.05.002
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.05.002
subsystem: admission-security
architecture_module: internal/admission
capability: CAP-017
priority: P0
criticality: critical
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
traces_to: [CAP-017]
kos_anchors:
  - elem-node-router-architecture
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.05.002: Router Rejects Non-Admitted Nodes Before Forwarding — Fail-Closed

## Description

The router's default posture is fail-closed: any frame whose source is not in the admitted node set for the SVTN is rejected before forwarding. The router does not forward first and check later. This is the enforcement point for SVTN boundary security. A frame that arrives before admission completion is rejected; the node must complete admission (BC-2.05.001) before any SVTN-scoped frames will be forwarded.

## Preconditions

1. The router has an admitted node set for the SVTN (potentially empty at SVTN creation).
2. A frame arrives at the router with a source address.

## Postconditions

1. If the source address is in the admitted set: frame is forwarded normally.
2. If the source address is NOT in the admitted set: frame is dropped; E-ADM-003 logged (not sent to the source — no connection exists to send on); counter incremented.
3. The admitted set check happens before any forwarding logic.
4. An admitted node whose key is subsequently revoked remains in the admitted set until the next re-authentication challenge (per FM-007 acknowledged gap).

## Invariants

1. **DI-006**: Every frame carrying SVTN-scoped traffic is verified against the admitted key set at the first router. No exceptions.
2. Fail-closed: default action for any ambiguous frame is reject, not forward.
3. The router does not communicate rejection back to the source (because the source is by definition not an admitted SVTN member and has no incoming path for the error response).

## Trigger

Frame arrival at the router from any source.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Node admitted but key revoked (propagation delay, FM-007) | Router has not received revocation; frame forwarded. This is the acknowledged propagation gap. |
| EC-002 | Frame with forged source address of an admitted node | HMAC tag verification fails (BC-2.05.005 — the `frame_auth_key` is derived per-node-per-SVTN via HKDF-SHA256; the forger does not have the real node's admission key and cannot produce a valid tag). Frame rejected as HMAC failure. |
| EC-003 | Frame arrives during SVTN bootstrap (admitted set empty) | No frames forwarded until at least one node is admitted. First node is the control node bootstrapped locally. |
| EC-004 | Non-SVTN frame (router management frame) | Management frames use a separate authentication path and are exempt from SVTN admitted-set check. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame from admitted node address | Frame forwarded normally | happy-path |
| Frame from non-admitted address | Frame dropped; E-ADM-003 logged | error |
| Frame with forged admitted-node source address | HMAC failure; dropped (handled by BC-2.05.005) | error |
| Frame from node mid-admission (challenge not yet complete) | Dropped; E-ADM-003 logged | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-008 | No frame from non-admitted source reaches any destination | proptest/integration |
| VP-008 | Admitted-set check is performed before forwarding decision | code-audit |
| VP-008 | Fail-closed: empty admitted set → no frames forwarded | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-017 ("SVTN admission via signed key challenge (Tier 1)") per capabilities.md §CAP-017 |
| L2 Domain Invariants | DI-006 (HMAC frame authentication at first router), DI-005 (SVTN cryptographic isolation) |
| Architecture Module | internal/admission |
| Stories | [filled by story-writer] |
| Architecture Decision | ADR-001 (amended): frame_auth_key (per-node-per-SVTN HKDF derivation) is the basis for HMAC tag verification enforced at this boundary |
| Capability Anchor Justification | CAP-017 ("SVTN admission via signed key challenge (Tier 1)") per capabilities.md §CAP-017 — this BC specifies the router-side enforcement ("grants or denies admission") that is the flip side of the node-side challenge in BC-2.05.001 |

## Related BCs

- BC-2.05.001 — depends on: admission challenge is the mechanism that adds nodes to the admitted set
- BC-2.05.005 — composes with: HMAC verification is a second enforcement layer on top of admitted-set check
