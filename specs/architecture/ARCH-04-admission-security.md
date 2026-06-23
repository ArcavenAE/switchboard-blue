---
artifact_id: ARCH-04-admission-security
document_type: architecture-section
level: L3
version: "1.0"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.001.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.002.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.003.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.004.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.005.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.006.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.007.md'
  - '.factory/specs/domain-spec/invariants.md'
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# ARCH-04: Admission & Security

## Two-Tier Key Model

| Tier | Purpose | Enforced By | BC |
|------|---------|------------|-----|
| Tier 1 | SVTN network admission | Router | BC-2.05.001, BC-2.05.002 |
| Tier 2 | Per-session access | Access node | BC-2.05.003 |

DI-011: The same keypair may serve both roles, but the authorization scopes are
independent. Revoking Tier 1 removes the node from the network; revoking Tier 2
removes access to a specific session.

## ADR-003: Duplicate Public Key Registration Policy

**Decision:** Last-write-wins (LWW).

**OQ-003 resolution:** When a public key is registered twice (same key, different
roles or same key, same role), the most recent registration takes effect. The
previous entry is overwritten.

**Rationale:**
- Reject-on-duplicate would require tracking all historical registrations and
  creates operational friction on key rotation.
- LWW is simple and predictable: the operator who registers last controls the key.
- The control node who performs the registration must be authenticated (DI-012), so
  LWW does not weaken the trust model — the last writer is an authenticated operator.
- If an operator registers a key accidentally, they can correct it by re-registering.

**Security implication:** LWW means a compromised control node key could overwrite
key registrations. Mitigated by: (a) control node admission uses the same signed
challenge as other nodes; (b) key registration events are logged with the registrant's
key fingerprint; (c) key revocation is available.

## ADR-004: Console Key Registration Model (OQ-001, OQ-002)

**Decision:** Key management (register, revoke, expire) is exclusive to the **control
node** role. Console nodes cannot register new Tier 1 keys. Access nodes have no key
management capability.

**OQ-001 resolution:** Console nodes cannot register new Tier 1 admission keys. They
can view their own key status but not modify the key store.

**OQ-002 resolution:** Access nodes have no key management capability whatsoever. They
hold their own keypair for admission but cannot modify the SVTN key store.

**Permission hierarchy:**
```
Control node:
  - Create/destroy SVTNs
  - Register, revoke, expire any key (any role) — subject to LWW (ADR-003)
  - Query SVTN status and key inventory

Console node:
  - Attach/detach sessions
  - Query session list and quality
  - Remote console control (sbctl commands)
  - Cannot modify key store

Access node:
  - Publish sessions over SVTN
  - Enforce Tier 2 session authorization
  - Cannot modify key store
```

**DI-012 consistency:** The control node is a network participant (not a router
manager). It does not have privileged access to router forwarding tables. Key
registration propagates via the router's distributed database as data-plane traffic.

**OQ-002 note:** The above assigns no key management capability to access nodes.
This is a conservative choice. Expansion (e.g., access nodes can expire their own
session-auth keys) can be added in PE phase when operational patterns are understood.

## Tier 1 Admission Protocol (internal/admission, BC-2.05.001)

```
1. Node → Router: CONNECT (svtn_id, node_addr, pubkey_fingerprint)
2. Router → Node: CHALLENGE (nonce, router_sig)
   nonce = crypto/rand.Read(32)
   router_sig = Sign(router_private_key, nonce)  [prevents nonce forgery]
3. Node → Router: CHALLENGE_RESPONSE (nonce_sig)
   nonce_sig = Sign(node_private_key, nonce)
4. Router: verify nonce_sig against pubkey in admitted_key_set[svtn_id]
   success → node enters admitted_nodes[svtn_id]
   failure → E-ADM-001, connection closed
5. Router → Node: ADMITTED (session_token)
```

**DI-002 enforcement:** Private key never leaves the node. Only `Sign(private_key, nonce)`
is computed; the private key bytes are never serialized or logged.

**Nonce uniqueness:** Nonces are 32-byte crypto/rand values. Router maintains a
used-nonce set (TTL = 60s) to prevent replay. BC-2.05.001 EC-003.

## HMAC Frame Authentication (internal/hmac, BC-2.05.005)

After admission, every frame carries an HMAC in the outer header (bytes 40..43
of the 44-byte header; actually the last 4 bytes of a 16-byte HMAC field — see
ARCH-02 for field layout).

Wait — re-reading ARCH-02: the HMAC occupies the last field in the outer header.
The 44-byte outer header layout in ARCH-02 ends at byte 43 after sequence (4 bytes).
Correction: the full 44-byte layout as designed places HMAC in a dedicated field.
The exact byte offsets are specified in `internal/frame`; the HMAC field is 16 bytes
(truncated HMAC-SHA256 as per ADR-001).

**Hot path:** HMAC verification is the first operation after parsing the outer header.
Frames failing HMAC are dropped before routing logic executes. This is the
"fail-closed" behavior of BC-2.05.002.

**Key scope:** HMAC key = `HKDF-SHA256(router_master_key, svtn_id || "hmac-frame-auth")`.
The router derives one HMAC key per SVTN. Admitted nodes compute the same key as part
of the admission exchange (key material passed in the ADMITTED message).

## SVTN Cryptographic Isolation (internal/routing, BC-2.05.006, DI-005)

The router maintains a separate admitted node set and HMAC key per SVTN. A frame
for SVTN-A is verified against SVTN-A's HMAC key. Even if an attacker knows SVTN-B's
HMAC key, they cannot forge valid frames for SVTN-A (different key derivation input).

Router forwarding table is partitioned by SVTN ID: `forwardingTable[svtn_id][dst_addr]`.
A frame for SVTN-A cannot be forwarded to a node in SVTN-B because the lookup is
always scoped to the frame's svtn_id field.

**NFR-013 enforcement:** CI integration test: two SVTNs on same router; verify no
cross-SVTN delivery under all conditions including error paths.

## Key Lifecycle (internal/svtnmgmt, BC-2.05.004)

```
Register: control_node → router → admitted_key_set[svtn_id].add(pubkey, role)
Revoke:   control_node → router → admitted_key_set[svtn_id].remove(pubkey)
Expire:   control_node → router → admitted_key_set[svtn_id].set_expiry(pubkey, time)
```

Expiry check is at re-authentication time: if `now > expiry`, the node is not
re-admitted (E-ADM-005 "key expired"). Between expiry and re-authentication, the
node continues operating (FM-007 documented tradeoff).

**Key propagation (PE phase):** In multi-router deployments, key changes propagate
via the router distributed database. Propagation delay is a known gap (FM-007).
Immediate revocation requires `sbctl router reload` on each router individually.

## Risk Mitigations

| Risk | Mitigation |
|------|-----------|
| R-001 (content separation) | Router HMAC and routing logic operate only on outer header fields; `payload []byte` is never parsed in router code path |
| R-009 (traffic analysis) | Explicitly in-scope per DI-003; documented in operator guide |
| R-010 (DoS via forged frames) | HMAC verification at first router boundary; forged frames never reach routing logic |
