---
artifact_id: BC-2.05.006
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.05.006
subsystem: admission-security
architecture_module: internal/routing
capability: CAP-020b
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
traces_to: [CAP-020b]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.05.006: SVTN Cryptographic Isolation — Admitted Node on SVTN-A Cannot See SVTN-B Traffic

## Description

A node admitted to SVTN-A cannot receive or observe any frames from SVTN-B, even if both SVTNs are served by the same physical router infrastructure. The SVTN ID field in every frame's outer header provides logical separation; the `frame_auth_key` is derived per `(node_admission_pubkey, svtn_id)` via HKDF-SHA256 (per ADR-001 amended), so a node's key for SVTN-A is cryptographically distinct from its key for SVTN-B — a node admitted to SVTN-A cannot produce valid HMAC tags for SVTN-B frames. A node that holds keys for both SVTNs has two separate admitted identities with separate derived keys.

## Preconditions

1. A router serves multiple SVTNs simultaneously.
2. A node is admitted to exactly one SVTN (e.g., SVTN-A).

## Postconditions

1. The node receives only frames with SVTN ID matching its admitted SVTN.
2. The router's forwarding engine partitions frame processing by SVTN ID: SVTN-A frames are forwarded only to SVTN-A admitted nodes.
3. The node cannot see frame counts, frame sizes, or timing of traffic on other SVTNs (no side-channel).
4. There is no administrative override that routes SVTN-B traffic to SVTN-A admitted nodes.

## Invariants

1. **DI-005**: Cross-SVTN visibility requires possession of keys registered against both SVTNs. There is no single-key cross-SVTN access path.
2. SVTN ID in the frame outer header is router-visible metadata that enforces routing isolation.
3. `frame_auth_key` (per-node-per-SVTN HKDF derivation) is scoped per `(node_admission_pubkey, svtn_id)` pair: a node's key for SVTN-A is cryptographically distinct from its key for SVTN-B.

## Trigger

Frame routing decision at the router when multiple SVTNs share the router.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Node has keys for both SVTN-A and SVTN-B | Node receives frames for both SVTNs separately. Cross-SVTN traffic is not possible even with dual-SVTN membership. |
| EC-002 | Router misconfiguration routes SVTN-A frame to SVTN-B subscriber | HMAC verification fails at the SVTN-B subscriber (the frame's HMAC is keyed to SVTN-A credentials). Frame rejected. Defense-in-depth. |
| EC-003 | New SVTN created by control node on an existing router | Router creates a new SVTN partition; existing SVTN admitted nodes are unaffected. |
| EC-004 | Router is compromised by an attacker | DI-003: compromised router can perform traffic analysis (observe who communicates on which SVTN, frame counts). Cannot see cross-SVTN content or inject content. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame with SVTN-A ID arrives at router; Node-B is admitted to SVTN-B only | Frame not delivered to Node-B; SVTN-B admitted nodes do not receive SVTN-A frames | happy-path |
| Node-A attempts to send frame with SVTN-B ID | HMAC verification fails (Node-A has no valid SVTN-B HMAC key); frame rejected | error |
| Same router serves 3 SVTNs; frame for SVTN-2 | Forwarded only to SVTN-2 admitted nodes; SVTN-1 and SVTN-3 nodes unaffected | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-010, VP-039 | Frames with SVTN-X ID never delivered to nodes admitted only to SVTN-Y | integration/property |
| VP-010, VP-039 | HMAC key is scoped to (node, SVTN) pair | unit |
| VP-010, VP-039 | No cross-SVTN traffic under any router configuration | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-020b ("SVTN cryptographic isolation") per capabilities.md §CAP-020b |
| L2 Domain Invariants | DI-005 (SVTN cryptographic isolation), DI-003 (router compromise → availability/quality, not content) |
| Architecture Module | internal/routing |
| Stories | [filled by story-writer] |
| Architecture Decision | ADR-001 (amended): per-(node, svtn) HMAC keying via HKDF-SHA256 is the mechanism by which SVTN isolation is enforced — a node's frame_auth_key for SVTN-A is cryptographically distinct from its key for SVTN-B |
| Capability Anchor Justification | CAP-020b ("SVTN cryptographic isolation") per capabilities.md §CAP-020b — this BC specifies the per-(node, SVTN) HMAC keying as the mechanism that enforces SVTN isolation, which is exactly what CAP-020b defines |

## Related BCs

- BC-2.05.005 — depends on: HMAC verification enforces the SVTN boundary that this BC describes as isolated
- BC-2.05.002 — composes with: admitted-set check is also per-SVTN
