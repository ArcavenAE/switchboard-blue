---
artifact_id: BC-2.01.006
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.01.006
subsystem: session-networking
architecture_module: internal/frame
capability: CAP-004
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
traces_to: [CAP-004]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.01.006: Session Identity Is Cryptographic — Node Address Derived from hash(SVTN-ID, public-key)

## Description

A node's address on the SVTN is not an IP address or hostname — it is a deterministic 8-byte value derived from `hash(SVTN-ID || public-key)`. This means two nodes with the same keypair on different SVTNs have different addresses. No registration authority is needed: a node can compute its own address and announce it, and other nodes can verify it from the public key alone. This is the mechanism that enables session continuity across IP changes.

## Preconditions

1. The node has an OpenSSH keypair (admission key for the SVTN).
2. The SVTN ID is known (16-byte identifier).
3. A cryptographic hash function is available (implementation: SHA-256, output truncated to first 8 bytes).

## Postconditions

1. The node address is exactly 8 bytes.
2. The same (SVTN-ID, public-key) pair always produces the same address (deterministic).
3. Different SVTNs produce different addresses for the same keypair.
4. The address is included in every frame's outer header as the source address field.
5. The router verifies the source address matches the admitted key set before forwarding.

## Invariants

1. **DI-004**: No direct node-to-node communication. The address is used for routing through the router, not for direct contact.
2. **DI-002**: The private key is never included in the address derivation output or used outside of the signing operation. The address is derived from the public key only.
3. Address collisions are treated as a security violation; the router must not admit two nodes with the same derived address on the same SVTN.

## Trigger

Node initialization; SVTN admission; frame transmission.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Two nodes attempt to join with the same derived address (collision) | Router rejects the second admission with E-ADM-004 (address collision). Collisions should be cryptographically negligible with 8-byte hash; log if detected. |
| EC-002 | Node's IP address changes but SVTN-ID and keypair unchanged | Node address is unchanged. Router recognizes the node's cryptographic identity on re-authentication. Session continuity preserved. |
| EC-003 | Node uses a different keypair for a second SVTN | Different SVTN-ID + different keypair → different address on that SVTN. Both addresses valid simultaneously. |
| EC-004 | Attempt to spoof a node address (forged source address in outer header) | HMAC tag verification fails at router (E-ADM-002) — the `frame_auth_key` is derived per `(node_admission_pubkey, svtn_id)` via HKDF-SHA256; the forger does not have the legitimate node's admission key and cannot produce a valid tag. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| svtn_id=0x0102...(16B), pubkey=ed25519 test key | address = SHA-256(svtn_id \|\| pubkey)[0:8] (deterministic; implementation: SHA-256 truncated to first 8 bytes) | happy-path |
| Same svtn_id, different pubkey | Different address | property |
| Different svtn_id, same pubkey | Different address | property |
| Forged source address in frame outer header | Router HMAC check fails; frame rejected | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-014 | address(svtn_id, pk) is deterministic: same inputs → same output | unit |
| VP-014 | address(svtn_id1, pk) != address(svtn_id2, pk) for svtn_id1 != svtn_id2 | proptest |
| VP-014 | address does not contain or leak the private key | code-audit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-004 ("Session continuity across network transitions") per capabilities.md §CAP-004 |
| L2 Domain Invariants | DI-004 (no direct node-to-node communication), DI-002 (private keys never transit) |
| Architecture Module | internal/frame |
| Stories | [filled by story-writer] |
| Architecture Decision | ADR-001 (amended): HKDF-SHA256 key derivation; node address uses SHA-256 truncated to 8 bytes |
| Capability Anchor Justification | CAP-004 ("Session continuity across network transitions") per capabilities.md §CAP-004 — this BC specifies the cryptographic identity mechanism that makes continuity across IP changes possible |

## Related BCs

- BC-2.01.007 — composes with: IP change tolerance depends on cryptographic identity
- BC-2.05.001 — depends on: Tier 1 admission uses the same public key that this address is derived from
