---
artifact_id: BC-2.05.001
document_type: behavioral-contract
level: L3
version: "1.3"
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
modified:
  - version: "1.3"
    date: 2026-07-18
    author: product-owner
    change: >
      Consistency-audit Findings 3 + 4 (Option A):
      (F4) Fix PC-5 label imprecision: "key not admitted" → "signature verification
      failed" — distinct from E-ADM-003 (not registered); PC-5 is the E-ADM-001 path.
      (F3, Option A) Add Postcondition 7 (ErrKeyRevoked / E-ADM-005 when key is
      revoked at initial admission) — symmetric analog to PC-6 (expired).
      AdmitNode already returns ErrKeyRevoked (internal/admission/admission.go lines
      486-488 and 506-508); this PC documents the existing behavior to close the
      over-claim in BC-2.01.009 PC-5 which cited "not revoked" against BC-2.05.001
      PCs 3-6 but PCs 3-6 had no revoked postcondition. PC-6 and
      AC-006/AC-013 citations (PC-6/Invariant 5) are UNCHANGED — PC-7 appended after PC-6.
  - version: "1.2"
    date: 2026-07-18
    author: product-owner
    change: >
      O-1 ruling (S-BL.NODE-IDENTIFY-WIRE-rulings.md §15, human-ratified 2026-07-18):
      AdmitNode MUST enforce key expiry at initial admission — add Postcondition 6
      (ErrKeyExpired / E-ADM-015 when expiry is set and past), mirroring the existing
      expiry-check pattern in ReAuthenticate.
      Add EC-005 (expired key at initial admission → E-ADM-015, connection closed).
      Add canonical test vector for expired-key path.
      Add VP row for AdmitNode expiry enforcement.
      Amend Invariant 5: initial admission (AdmitNode) now enforces expiry, closing
      the previous gap where an expired key could complete the initial handshake
      even though ReAuthenticate would reject it.
      Story trace updated: S-BL.NODE-IDENTIFY-WIRE added.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/edge-cases.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
input-hash: "815cf63"
extracted_from: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/edge-cases.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
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
5. On failure — signature verification failed: router returns E-ADM-001 "admission denied: signature verification failed"; connection closed. (Distinct from E-ADM-003 "frame from non-admitted source" — this postcondition covers the path where the key IS registered but the signature does not verify.)
6. On failure — key expired: if the node's registered key has a non-zero expiry timestamp and `time.Now().UTC()` is after that timestamp, `AdmitNode` returns `ErrKeyExpired` (E-ADM-015); connection closed. This check mirrors the existing expiry enforcement in `ReAuthenticate` and closes the gap where an expired key could be admitted at connect time but rejected on the next re-authentication. (O-1 ruling, S-BL.NODE-IDENTIFY-WIRE-rulings.md §15, human-ratified 2026-07-18.)
7. On failure — key revoked: if the node's registered key is marked revoked (`snap.revoked == true` or `liveEntry.revoked == true`), `AdmitNode` returns `ErrKeyRevoked` (E-ADM-005); connection closed. This check is performed at both the snapshot step (under RLock) and the write-lock re-check (under Lock) to defend against a concurrent `RevokeKey` call that races between the two steps. Implementation: `internal/admission/admission.go` lines 486–488 (snapshot check) and 506–508 (write-lock re-check). Symmetric analog to PC-6 (expiry enforcement). (Consistency-audit Finding 3, Option A; documents existing code behavior.)

## Invariants

1. **DI-002**: The node's private key is used only to sign the challenge locally. It never leaves the node.
2. **DI-006**: HMAC frame authentication (subsequent to admission) depends on the same keypair.
3. The challenge nonce must be unique per challenge attempt to prevent replay attacks.
4. **DI-012**: The control node's admission is via the same challenge mechanism as any other node. The control node has no privileged router access.
5. **Expiry enforcement is symmetric across `AdmitNode` and `ReAuthenticate`**: both the initial admission handshake and every subsequent re-authentication challenge reject expired keys with `ErrKeyExpired` (E-ADM-015). No window exists in which an expired key may be admitted at connect time but rejected at re-auth — the policy is fail-closed at both gates. Implementation anchor: the expiry check is added to `admission.AdmitNode` in `internal/admission` after the snapshot-revoked read and before the write-lock acquire, mirroring `admission.ReAuthenticate`'s expiry-check pattern (`if !snap.expiry.IsZero() && now.After(snap.expiry)`).

## Trigger

Node initiates a connection to the router and begins the admission handshake.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-005) | Node's key is revoked between admission and re-authentication | Revocation does not instantly disconnect. Node remains admitted until the next re-authentication challenge. After that, E-ADM-005 "key revoked". |
| EC-002 (DEC-007) | Same public key registered twice with different roles | Per **ADR-003** (last-write-wins for duplicate key registration): the most recent registration request authenticated through `sbctl admin` supersedes earlier registrations for the same `(node_pubkey, svtn_id)` pair. No conflict; no manual reconciliation required. |
| EC-003 | Challenge nonce replay attempt | Nonces are single-use. Router rejects a signature over an already-consumed nonce with E-ADM-008 "nonce replay". |
| EC-004 | Node is a router joining as a peer (PE-to-PE connection) | Peer router admission uses the same signed-challenge mechanism with a router-role key. |
| EC-005 | Node's key has a non-zero expiry timestamp and `time.Now().UTC()` is after that timestamp at the moment of initial admission | `AdmitNode` returns `ErrKeyExpired` (E-ADM-015); connection closed. The operator has expressed that the credential is no longer valid after the expiry time; the initial admission gate must enforce this consistently with `ReAuthenticate`. (Postcondition 6 / Invariant 5.) |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Node with registered ed25519 key connects and signs challenge | Admission success; node enters active set | happy-path |
| Node signs challenge with wrong key (key not registered) | E-ADM-001 "admission denied"; connection closed | error |
| Node replays a previous signed challenge | E-ADM-008 "nonce replay"; connection closed | error |
| Node's key revoked; re-authentication challenge issued | E-ADM-005 "key revoked"; connection closed | error |
| Node's key is revoked at the moment of initial admission (`snap.revoked == true`) | `AdmitNode` returns E-ADM-005 "key revoked"; connection closed — Postcondition 7 | error |
| Node's key has non-zero expiry; `time.Now().UTC()` is after expiry at connect time | `AdmitNode` returns E-ADM-015 "key expired"; connection closed | error — Postcondition 6 |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-007, VP-009 | Private key never appears in network traffic during admission | property/audit |
| VP-007, VP-009 | Admission fails for any key not in the admitted set | proptest |
| VP-007, VP-009 | Nonce is unique and single-use | unit |
| VP-008 | Admission fails for unregistered key | proptest |
| test-as-evidence | `AdmitNode` returns `ErrKeyExpired` (E-ADM-015) when key expiry is set and `time.Now().UTC()` is after expiry — Postcondition 6 / Invariant 5 | unit (analog: `ReAuthenticate` expiry test cases in `reauth_test.go`) |
| test-as-evidence | `AdmitNode` returns `ErrKeyRevoked` (E-ADM-005) when key is revoked at initial admission — Postcondition 7 | unit (existing revoke path in `internal/admission/admission.go` lines 486-488; analog test in `admission_test.go`) |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-017 ("SVTN admission via signed key challenge (Tier 1)") per capabilities.md §CAP-017 |
| L2 Domain Invariants | DI-002 (private keys never transit), DI-006 (HMAC frame auth at first router), DI-012 (control node is a participant, not a router manager) |
| Architecture Module | internal/admission |
| Stories | S-2.02 (PR #6, initial implementation); S-BL.NODE-IDENTIFY-WIRE (Postcondition 6 / Invariant 5 — `AdmitNode` expiry enforcement via O-1 ruling) |
| Capability Anchor Justification | CAP-017 ("SVTN admission via signed key challenge (Tier 1)") per capabilities.md §CAP-017 — this BC is the direct behavioral specification of the "signed challenge" admission mechanism CAP-017 defines as the network entry gate |

## Related BCs

- BC-2.05.002 — composes with: router rejects frames from nodes that failed this challenge
- BC-2.05.004 — related to: key revocation affects re-authentication under this BC
- BC-2.01.007 — related to: re-authentication uses the same mechanism; expiry enforcement is symmetric (EC-005 here, EC-005 there)
- BC-2.01.009 — composes with: the three-message NODE_IDENTIFY handshake invokes `AdmitNode` defined here; Postconditions 6 and 7 apply at the ChallengeResponse step

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.3 | 2026-07-18 | Consistency-audit Findings 3 + 4 (Option A). **(F4)** PC-5 label imprecision fixed: "key not admitted" → "signature verification failed (E-ADM-001)" — the old label was confusingly close to E-ADM-003 ("not admitted"); PC-5 is the path where the key IS registered but the nonce signature does not verify, triggering E-ADM-001. **(F3, Option A)** Add Postcondition 7 — `AdmitNode` returns `ErrKeyRevoked` (E-ADM-005) when the key is revoked at initial admission, documenting existing code behavior (`internal/admission/admission.go` lines 486-488 snapshot check + 506-508 write-lock re-check). This closes the over-claim in BC-2.01.009 PC-5 which cited "not revoked" as covered by BC-2.05.001 PCs 3-6, but no PC in 3-6 covered the revoked path. PC-7 appended after PC-6 to preserve AC-006/AC-013's PC-6/Invariant-5 anchor citations unchanged. Also added test vector and VP row for the revoked-at-admission path. |
| 1.2 | 2026-07-18 | O-1 ruling (S-BL.NODE-IDENTIFY-WIRE-rulings.md §15, human-ratified 2026-07-18): add Postcondition 6 — `AdmitNode` returns `ErrKeyExpired` (E-ADM-015) when key expiry is set and past. Add Invariant 5 — expiry enforcement is symmetric across `AdmitNode` and `ReAuthenticate`; implementation anchor: expiry check added to `admission.AdmitNode` mirroring `admission.ReAuthenticate` pattern. Add EC-005 (expired key at initial admission). Add canonical test vector for expired-key path. Add VP row for `AdmitNode` expiry enforcement. Add `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` to inputDocuments. Stories row gains `S-BL.NODE-IDENTIFY-WIRE`. |
| 1.1 | 2026-06-23 | Initial commission (S-2.02 implementation; PR #6). |
