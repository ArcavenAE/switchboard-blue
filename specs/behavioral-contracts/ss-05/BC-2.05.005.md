---
artifact_id: BC-2.05.005
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.05.005
subsystem: SS-TBD
capability: CAP-020
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
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-020]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.05.005: HMAC Frame Authentication at First Router Boundary

## Description

Every SVTN-scoped frame carries a 16-byte HMAC in the outer header, computed by the sending node using its admission key. The first router that receives the frame verifies the HMAC before forwarding. Frames with invalid HMACs are rejected before forwarding — fail-closed. This ensures every forwarded frame originated from an admitted node holding the expected private key.

## Preconditions

1. The sending node is admitted to the SVTN and has a valid admission key.
2. The HMAC is computed over the full frame (outer header fields + payload) using HMAC-SHA256 (or equivalent) with the node's admission key. Output truncated to 16 bytes.
3. The first router has the sending node's public key in its admitted key set.

## Postconditions

1. HMAC verification succeeds: frame forwarded to destination.
2. HMAC verification fails: frame dropped; E-ADM-002 "HMAC verification failed: <svtn_id>, <src_addr>, <frame_type>" logged at the router; the sending node receives no delivery confirmation.
3. Repeated HMAC failures from the same source address trigger an admission alert (implementation: ≥5 failures in 60 seconds).

## Invariants

1. **DI-006**: Every frame carrying SVTN-scoped traffic is verified against the admitted key set by the first router that receives it. No exceptions.
2. **DI-003**: HMAC authentication proves identity (admitted node) but does not protect content confidentiality at the router (content is SSH-encrypted separately).
3. The HMAC is recomputed fresh by the router for verification — there is no HMAC caching.

## Trigger

Frame arrival at the first router after transmission from the source node.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (FM-006) | Frame arrives with non-member HMAC | E-ADM-002 logged; frame dropped silently (no rejection sent to source). |
| EC-002 | Frame corruption in transit causes HMAC mismatch | Same as non-member HMAC — E-ADM-002 logged; frame dropped. Sending node will retransmit. |
| EC-003 | Key rotation: node uses new key, router has old key | HMAC verification fails until router receives new key propagation. Node retransmits; after key propagation, HMAC succeeds. |
| EC-004 | Empty-tick frame has no payload | HMAC computed over outer header fields + zero-length payload. This is valid; verification proceeds normally. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Valid frame with correct HMAC | Frame forwarded; no log event | happy-path |
| Frame with HMAC computed with wrong key | E-ADM-002 logged; frame dropped | error |
| Frame with HMAC field all-zeros | E-ADM-002 logged; frame dropped | error |
| Empty-tick frame with correct HMAC | Frame forwarded normally | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | For all admitted nodes: frames with correct HMAC are forwarded | proptest |
| VP-TBD | For all non-admitted sources: frames are dropped | proptest |
| VP-TBD | HMAC covers all outer header fields + payload | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.md §CAP-020 |
| L2 Domain Invariants | DI-006 (HMAC frame authentication at first router), DI-003 (router compromise → availability/quality, not content) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-020 ("HMAC frame authentication at router boundary") per capabilities.md §CAP-020 — this BC is the direct behavioral specification of the HMAC verification that CAP-020 defines as "The first router verifies and rejects frames from non-admitted sources before forwarding" |

## Related BCs

- BC-2.05.002 — composes with: admitted-set check + HMAC together enforce the SVTN boundary
- BC-2.01.004 — depends on: HMAC field is in the outer header defined by BC-2.01.004
