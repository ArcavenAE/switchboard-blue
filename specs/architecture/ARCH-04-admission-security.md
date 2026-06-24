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
modified:
  - 2026-06-23T00:00:00
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

**DEC-007 resolution:** When a public key is registered twice (same key, different
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

## ADR-004: Console Key Registration Model and Permission Hierarchy (OQ-001, OQ-002, OQ-003, F-010)

**Decision:** Key management (register, revoke, expire) is exclusive to the **control
node** role. Console nodes cannot register new Tier 1 keys. Access nodes have no key
management capability.

**OQ-001 resolution:** Console nodes cannot register new Tier 1 admission keys. They
can view their own key status but not modify the key store.

**OQ-002 resolution:** Access nodes have no key management capability whatsoever. They
hold their own keypair for admission but cannot modify the SVTN key store.

**OQ-003 resolution:** A permission hierarchy exists among key roles: control > console > readonly. A console-role or readonly-role key cannot revoke a control-role key (such attempts fail with E-ADM-011). Control-to-control revocation requires `sbctl admin` human authorization (split-brain mitigation, ADR-004 paragraph above).

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

**Role hierarchy (explicit — F-010):** `control > console > readonly`. Lower-tier
roles cannot revoke higher-tier roles.

**Can a console-role key revoke a control-role key?** NO. Console role is below
control in the hierarchy. Any revocation operation by a console-role key on a
control-role key is rejected with E-ADM-011.

**Can a control node revoke another control node's key?** YES — but only with
human operator approval through `sbctl admin` audit. Control-key changes are NOT
automated. Operational constraint: all control-role key changes require out-of-band
human authorization to prevent split-brain (two control nodes simultaneously revoking
each other). The `sbctl admin` subcommand enforces this by requiring a confirmation
token from an offline operator key. This is a known operational constraint documented
in the operator guide.

**Split-brain mitigation:** If two control nodes simultaneously attempt to revoke
each other's keys, the LWW semantics (ADR-003) determine the final state based on
timestamp ordering. However, because control-key changes require human authorization,
simultaneous automated split-brain revocation is not possible in normal operation.
Emergency recovery procedure: manual `sbctl admin recover` with a bootstrap key.

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

Every frame from an admitted node carries an HMAC tag in the outer header bytes 36–43
(the 8-byte `hmac_tag` field, as specified in ARCH-02). HMAC verification is the
first operation after parsing the outer header. Frames failing HMAC are dropped
before routing logic executes. This is the "fail-closed" behavior of BC-2.05.002.

The `hmac_tag` field carries the first 8 bytes of HMAC-SHA256 output, computed over
the full frame (outer header bytes 0–35 || channel header || payload), with
`hmac_tag` bytes treated as zeros during computation.

## HMAC Keying (F-003)

**Decision:** The HMAC key for frame authentication is derived per admitted node per
SVTN using HKDF-SHA256. This binds authentication to the specific admitted node,
preventing cross-node frame forgery even within the same SVTN.

The `node_admission_pubkey` is the Ed25519 public key presented during node admission
(BC-2.05.002, step 1 CONNECT message). The router stores this key in
`admitted_key_set[svtn_id][node_addr]` after successful challenge-response.

**Key derivation (canonical):**
```
HKDF-Extract(salt=svtn_id, ikm=node_admission_pubkey) → PRK
HKDF-Expand(PRK, info="switchboard-frame-auth", length=32) → frame_auth_key
```

This uses Go stdlib `crypto/sha256` and `golang.org/x/crypto/hkdf`. No new
transitive dependencies beyond the Go standard library.

**Why per-node, not per-SVTN?** Per-SVTN keying (`HKDF(router_master_key, svtn_id)`)
would allow any admitted node to forge frames bearing another admitted node's source
address — the HMAC key would be the same for all nodes in the SVTN. Per-node keying
ensures that Node-A's `frame_auth_key` is different from Node-B's, even within the
same SVTN. This satisfies BC-2.05.002 EC-002 (forged source address → HMAC failure)
and BC-2.05.006 invariant 3 (HMAC keys scoped per (node, SVTN) pair).

**Key distribution:** After successful challenge-response, the router sends the
derived `frame_auth_key` to the node in the `ADMITTED` message (encrypted with the
node's public key). The node stores this key locally and uses it to compute the
`hmac_tag` for all subsequent frames.

**Router hot path:** The router maintains `frame_auth_key` per admitted node
in `admitted_key_set[svtn_id][node_addr].frame_auth_key`. HMAC verification at
first router boundary uses the per-node key. This is an O(1) lookup after
admitted-set check.

**References:** BC-2.05.002 (fail-closed enforcement), BC-2.05.006 (per-node,
per-SVTN isolation guarantee), ARCH-02 outer header `hmac_tag` field.

## SVTN Cryptographic Isolation (internal/routing, BC-2.05.006, DI-005)

The router maintains a separate admitted node set and per-node HMAC key per SVTN.
A frame for SVTN-A is verified against the source node's `frame_auth_key` in SVTN-A.
Even if an attacker knows SVTN-B's keys for a specific node, they cannot forge valid
frames for SVTN-A (different key derivation — `salt=svtn_id` differs between SVTNs).

Router forwarding table is partitioned by SVTN ID: `forwardingTable[svtn_id][dst_node_addr]`.
A frame for SVTN-A cannot be forwarded to a node in SVTN-B because the lookup is
always scoped to the frame's `svtn_id` field.

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
| R-010 (DoS via forged frames) | HMAC verification at first router boundary; forged frames never reach routing logic; per-node keying prevents cross-node forgery |
