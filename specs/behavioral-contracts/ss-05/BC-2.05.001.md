---
artifact_id: BC-2.05.001
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.05.001
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
  - '.factory/specs/domain-spec/edge-cases.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-017]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.05.001: Tier 1 SVTN Admission via Signed Key Challenge

## Description

A node joins an SVTN by proving possession of a private key whose corresponding public key is registered against that SVTN. The router issues a signed challenge (nonce); the node signs the challenge with its private key; the router verifies the signature against the admitted key set. On success, the node is admitted and may exchange SVTN-scoped traffic. On failure, the connection is rejected with an explicit error.

## Preconditions

1. The node has an OpenSSH keypair (admission key).
2. The node's public key is registered against the target SVTN with an appropriate role (control, console, or access).
3. The router has an up-to-date admitted key list for the SVTN.

## Postconditions

1. The router issues a challenge: a random nonce, signed by the router's own key to prevent replay.
2. The node signs the challenge nonce with its private admission key.
3. The router verifies the signature using the stored public key.
4. On success: node is added to the router's active node set for this SVTN; node may send and receive SVTN-scoped frames.
5. On failure: router sends E-ADM-001 "admission denied: signature verification failed"; connection closed.

## Invariants

1. **DI-002**: The node's private key is used only to sign the challenge locally. It never leaves the node.
2. **DI-006**: HMAC frame authentication (subsequent to admission) depends on the same keypair.
3. The challenge nonce must be unique per challenge attempt to prevent replay attacks.
4. **DI-012**: The control node's admission is via the same challenge mechanism as any other node. The control node has no privileged router access.

## Trigger

Node initiates a connection to the router and begins the admission handshake.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-005) | Node's key is revoked between admission and re-authentication | Revocation does not instantly disconnect. Node remains admitted until the next re-authentication challenge. After that, E-ADM-005 "key revoked". |
| EC-002 (DEC-007) | Same public key registered twice with different roles | Per **ADR-003** (last-write-wins for duplicate key registration): the most recent registration request authenticated through `sbctl admin` supersedes earlier registrations for the same `(node_pubkey, svtn_id)` pair. No conflict; no manual reconciliation required. |
| EC-003 | Challenge nonce replay attempt | Nonces are single-use. Router rejects a signature over an already-consumed nonce with E-ADM-008 "nonce replay". |
| EC-004 | Node is a router joining as a peer (PE-to-PE connection) | Peer router admission uses the same signed-challenge mechanism with a router-role key. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Node with registered ed25519 key connects and signs challenge | Admission success; node enters active set | happy-path |
| Node signs challenge with wrong key (key not registered) | E-ADM-001 "admission denied"; connection closed | error |
| Node replays a previous signed challenge | E-ADM-008 "nonce replay"; connection closed | error |
| Node's key revoked; re-authentication challenge issued | E-ADM-005 "key revoked"; connection closed | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-007, VP-009 | Private key never appears in network traffic during admission | property/audit |
| VP-007, VP-009 | Admission fails for any key not in the admitted set | proptest |
| VP-007, VP-009 | Nonce is unique and single-use | unit |
| VP-008 | Admission fails for unregistered key | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-017 ("SVTN admission via signed key challenge (Tier 1)") per capabilities.md §CAP-017 |
| L2 Domain Invariants | DI-002 (private keys never transit), DI-006 (HMAC frame auth at first router), DI-012 (control node is a participant, not a router manager) |
| Architecture Module | internal/admission |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-017 ("SVTN admission via signed key challenge (Tier 1)") per capabilities.md §CAP-017 — this BC is the direct behavioral specification of the "signed challenge" admission mechanism CAP-017 defines as the network entry gate |

## Related BCs

- BC-2.05.002 — composes with: router rejects frames from nodes that failed this challenge
- BC-2.05.004 — related to: key revocation affects re-authentication under this BC
