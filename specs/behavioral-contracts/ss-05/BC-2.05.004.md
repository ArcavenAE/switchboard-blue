---
artifact_id: BC-2.05.004
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.05.004
subsystem: admission-security
architecture_module: internal/svtnmgmt
capability: CAP-019
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
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-019]
kos_anchors:
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.05.004: Key Lifecycle — Register, Revoke, and Expire Admission and Session-Authorization Keys

## Description

Control nodes can manage the public key registry for an SVTN: registering new keys (with role designation), revoking existing keys, and setting expiry dates. Per ADR-004, key management is exclusive to the control node; console and access nodes have no key-management capability. Key changes propagate via the router's distributed key store to all routers serving the SVTN. Registration allows a new node to join; revocation removes a node's SVTN membership; expiry sets an automatic future revocation.

## Preconditions

1. The requesting node is admitted to the SVTN with a key that has management authority for the operation. Per **ADR-004**: key management is exclusive to the control node. Console nodes have read-only access to the admitted-key set; they cannot register, revoke, or expire keys. Access nodes have no key-management capability. Console-to-control revocation is prohibited. Control-to-control revocation requires `sbctl admin` human authorization (split-brain mitigation).
2. The key operation is well-formed: a valid OpenSSH public key in authorized_keys format.
3. The router's distributed key store is reachable.

## Postconditions

1. **Register**: The public key is added to the admitted key set with the specified role (control, console, access). The key becomes active for admission challenges. Propagation: key is pushed to all routers serving the SVTN.
2. **Revoke**: The public key is removed from the admitted key set. Existing sessions using this key continue until the next re-authentication challenge (propagation delay acknowledged, FM-007). Propagation: revocation is pushed to all routers.
3. **Expire**: An expiry timestamp is associated with the key. At expiry, the key is automatically revoked by routers that honor the expiry timestamp.
4. Key changes are confirmed with a success response including the key fingerprint and operation timestamp.

## Invariants

1. **DI-011**: Revoking a Tier 1 key removes the node from the network; revoking a Tier 2 key removes access to a specific session. These are independent operations.
2. **DI-002**: Key registration and revocation operations use public keys only; private keys are never transmitted.
3. Key management operations are authenticated: the requesting node's signature is verified before any change is applied.
4. **DI-012**: The control node manages keys as a network participant; it does not have privileged router API access.

## Trigger

Operator runs `sbctl svtn keys register|revoke|expire` or equivalent API call.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-005) | Key revoked; node has active session | Node's session continues until next re-authentication challenge. Revocation propagates asynchronously. |
| EC-002 (FM-007) | Key revocation propagation is slow (one router not updated) | The un-updated router may still admit the revoked key. Propagation completes within the eventual consistency window. |
| EC-003 (DEC-007) | Same public key registered twice with different roles | Per **ADR-003** (last-write-wins for duplicate key registration): the most recent registration supersedes earlier registrations for the same `(node_pubkey, svtn_id)` pair. The operation returns "updated" with the new role; no conflict; no manual reconciliation required. |
| EC-004 | Key expires while session is active | Same behavior as revocation: session continues until next re-authentication challenge. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl svtn keys register --key=<pubkey> --role=console` | Key registered; fingerprint returned; propagation initiated | happy-path |
| `sbctl svtn keys revoke --key=<pubkey>` | Key revoked; active sessions continue until re-auth; propagation initiated | happy-path |
| `sbctl svtn keys register --key=<same-pubkey-already-registered>` | Response: "updated" with new role (per ADR-003: last-write-wins) | edge-case |
| Key operation by node without management authority | E-ADM-009 "insufficient authority for key operation" | error |
| Console or readonly key attempts to revoke a control key | E-ADM-011 "permission denied: console key cannot revoke control key (control > console > readonly)" | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-046 | Key registration makes key available for admission on all propagated routers | integration |
| VP-046 | Revocation propagates to all routers within eventual consistency window | integration |
| VP-046 | Private key never appears in key management wire messages | property |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 |
| L2 Domain Invariants | DI-002 (private keys never transit), DI-011 (role separation between Tier 1 and Tier 2 keys), DI-012 (control node is a participant) |
| Architecture Module | internal/svtnmgmt |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 — this BC specifies the complete key lifecycle operations that CAP-019 defines as the revocation path |

## Related BCs

- BC-2.05.001 — depends on: registered keys are what the admission challenge verifies
- BC-2.05.002 — related to: revocation eventually removes key from admitted set
