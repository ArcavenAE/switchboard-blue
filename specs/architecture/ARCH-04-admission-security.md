---
artifact_id: ARCH-04-admission-security
document_type: architecture-section
level: L3
version: "1.6"
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
  - 2026-06-24T00:00:00 # v1.1 — permit inline HKDF for 32-byte single-block case (refs drbothen/vsdd-factory#260 family, S-2.01 rev 2)
  - 2026-06-25T00:00:00 # v1.2 — clarify ADR-003 LWW resets admitted=false (security-by-default; refs adversary pass-2 L-2, S-2.02 rev 1.2)
  - 2026-06-25T00:00:00 # v1.3 — Key Lifecycle: replace E-ADM-005 "key expired" with E-ADM-015 (new sentinel minted per S-1.03 spec patch rev 1.1; E-ADM-005 = key revoked, not expired)
  - 2026-06-25T00:00:00 # v1.4 — ADR-009: HMAC enforcement at RouteFrame boundary (S-3.04 wire-up); declares fail-fast ordering and forbidden bypass paths
  - 2026-06-25T00:00:00 # v1.5 — ADR-009: fix three contradictions (spec-reviewer C-2/C-3): correct verifyFrameHMAC signature (bool not error, value args not pointers), correct auth key location (forwardingTable not admitted_key_set), clarify ordering vs. single-lock-acquisition (sequential checks, shared RLock)
  - 2026-06-25T00:00:00 # v1.6 — ADR-009: amended to permit lock-free HMAC verify; RLock released after [32]byte key copy; HMAC runs lock-free; admitted check re-locks internally; sequential ordering preserved by line order not lock holding (Wave-3 pass-1 M-1)
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

**Admission reset on re-registration (security-by-default):** Any `RegisterKey` call
replaces the prior `AdmittedKey` entry in full, including resetting `admitted=false`.
A previously admitted node whose key is re-registered (e.g., to change role) MUST
complete a fresh challenge-response handshake (`AdmitNode`) before it appears in the
active admitted set again. The implementation zero-initializes new `AdmittedKey`
structs, so this reset is automatic and unconditional. Rationale: an operator who
re-registers a key (even to the same value) cannot be assumed to have validated that
the prior session is still trusted; forcing re-handshake is the safer default.
References: S-2.02 EC-004, BC-2.05.001 PC4.

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

The key-derivation function is HKDF-SHA256 per RFC 5869. Implementations MAY use
either:

- `golang.org/x/crypto/hkdf` (canonical Go implementation), OR
- An inline implementation using `crypto/hmac` + `crypto/sha256` directly, suitable
  for the 32-byte single-block case (RFC 5869 Extract → single Expand iteration,
  approximately 6 lines of auditable code). Inline avoids the external dependency
  for the entire module.

Inline implementations MUST include an RFC 5869 §A.1 Known-Answer Test
(`TestDeriveKey_RFC5869_KAT`) to pin algorithm correctness. The library path requires
no KAT — the upstream library is presumed RFC-compliant.

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

**Router hot path:** The router maintains `FrameAuthKey` per admitted node
in `Router.forwardingTable[svtnID][nodeAddr].FrameAuthKey`. HMAC verification at
the first router boundary uses the per-node key retrieved from the forwarding table.
This is an O(1) lookup under `RLock`; see ADR-009 ordering for the exact sequence.

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
re-admitted (E-ADM-015 "key expired"). Between expiry and re-authentication, the
node continues operating (FM-007 documented tradeoff).

**Key propagation (PE phase):** In multi-router deployments, key changes propagate
via the router distributed database. Propagation delay is a known gap (FM-007).
Immediate revocation requires `sbctl router reload` on each router individually.

## ADR-009: HMAC Enforcement at RouteFrame Boundary (S-3.04)

**Decision:** `internal/routing.RouteFrame` calls `verifyFrameHMAC` as the **first
operation after outer header parsing**, before any admitted-set lookup, before any
forwarding table consultation, and before any logging of frame content.

**Ordering rationale:**
1. HMAC verification gates all downstream routing logic. A frame that fails HMAC is
   dropped immediately. This prevents unauthenticated frames from touching forwarding
   state, timing side-channels, or log infrastructure.
2. The admitted-set lookup happens AFTER HMAC verification. Calling admitted-set lookup
   before HMAC would allow an unauthenticated frame to probe the admitted-set (timing
   oracle). The correct order is: (a) parse outer header → (b) verify HMAC →
   (c) check admitted set → (d) route frame.
3. Frames from source addresses not in the admitted set are dropped during step (c).
   Frames with invalid HMAC from admitted-set members are dropped during step (b) —
   this handles the key rotation window where a node is admitted but is using a stale
   frame_auth_key.

**Fail-fast on bad MAC:** `verifyFrameHMAC` returns `false` on HMAC mismatch.
`RouteFrame` MUST drop the frame and return without further processing when the return
value is `false`. The caller (daemon main loop) MUST NOT retry or buffer failed frames.
The failure is logged at DEBUG level with the source address only (not frame content,
per R-001 content separation).

**Key lookup in RouteFrame:** `verifyFrameHMAC` receives the per-node frame_auth_key
from `Router.forwardingTable[svtnID][srcNodeAddr].FrameAuthKey`. This is an O(1) read
under `RLock`; the key is copied (value type) before the lock is released. If
`srcNodeAddr` is not present in the forwarding table at key-lookup time, the key is
unavailable and the frame is dropped immediately (fail-closed; return
ErrHMACVerificationFailed — key unknown).

**Ordering specification (lock-free HMAC verify — amended v1.6):**

1. Acquire `RLock` on `Router.forwardingTable`.
2. Look up entry for `(svtnID, srcNodeAddr)`. If absent: release lock and return
   `ErrHMACVerificationFailed` (fail-closed; key unknown).
3. **Copy** `FrameAuthKey` (a `[32]byte` value type, not a pointer) into a local
   variable.
4. Release the forwarding-table `RLock`.
5. **Lock-free:** call `verifyFrameHMAC(hdr, payload, copiedAuthKey)`. If `false`:
   return `ErrHMACVerificationFailed` (E-ADM-016).
6. Call `admittedKeySet.IsAdmitted(svtnID, nodeAddr)` (which acquires its own internal
   `RLock` on the admission state). If `false`: return `ErrNotAdmitted`.
7. Proceed to SVTNRoute / forward.

**Rationale for lock-free HMAC verify:**
- The `RLock` would be held during a CPU-bound (~10–50µs) HMAC computation, hurting
  concurrency on the forwarding table. Releasing early removes this contention.
- `FrameAuthKey` is a `[32]byte` value type; copying it is cheap and atomic from a
  sharing perspective. No pointer aliasing into the forwarding-table entry.
- LWW (last-write-wins, ADR-003) on `RegisterForwardingEntry` creates a new
  `*ForwardingEntry` struct; even if `RegisterForwardingEntry` races after the
  `RUnlock`, the copied `[32]byte` reflects the value authoritative at lookup time.
  Caller gets a consistent verify against whichever entry was live at lookup time.
- Sequential ordering (HMAC-before-admitted) is preserved by line ordering in
  `RouteFrame`, not by lock holding.

**Rejected alternatives:**
- **Single RLock spanning steps 1–6 (previous v1.5 prescription):** would hold the
  forwarding-table `RLock` through CPU-bound HMAC and into `admittedKeySet.IsAdmitted`,
  introducing lock-order risk between `r.mu` and `admittedKeySet.mu`, and stretching
  the critical section longer than necessary. Rejected in favour of the copy-and-release
  pattern.
- **Verify-without-copy:** holding the pointer to `entry.FrameAuthKey` after `RUnlock`
  would risk a data race if `RegisterForwardingEntry` overwrote the field in-place.
  Although LWW replacement creates a new entry (not in-place mutation), defensive
  copying eliminates the concern entirely.
- **Verify HMAC after admitted-set check:** rejected — timing oracle on admitted set.
- **Skip HMAC for frames from known-good source addresses:** rejected — defeats the
  per-frame authentication guarantee of BC-2.05.002 and BC-2.05.005.
- **Async HMAC verification (separate goroutine):** rejected — adds complexity without
  throughput benefit for the MVP LAN target (HMAC-SHA256 over 44-byte outer header
  is approximately 200ns on modern hardware; sync is fine).

**Implementation note (S-3.04):** The `verifyFrameHMAC` function signature (actual, from
`internal/routing/routing.go`):
```go
func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool
```
`hdr` is the outer header passed by value. `payload` is the channel header + payload
(the bytes over which the HMAC was computed). `authKey` is the 32-byte per-node
frame_auth_key, passed as a fixed-size array (not a slice). The function saves
`hdr.HMACTag` before zeroing it, computes HMAC-SHA256 over the modified header bytes
`||` `payload`, and compares against the saved wire tag. Returns `true` on success,
`false` on mismatch. Error wrapping (if any) is the caller's responsibility at the
`RouteFrame` call site.

Note: error-type return (`error`) was considered during design and rejected — the
function has exactly two outcomes (valid / invalid); `bool` is unambiguous and avoids
allocating an error value on every frame in the hot path.

**References:** BC-2.05.002 (fail-closed HMAC enforcement), BC-2.05.005 (HMAC frame
authentication), ARCH-02 (outer header `hmac_tag` field at bytes 36–43),
ARCH-04 HMAC Keying section.

## Risk Mitigations

| Risk | Mitigation |
|------|-----------|
| R-001 (content separation) | Router HMAC and routing logic operate only on outer header fields; `payload []byte` is never parsed in router code path |
| R-009 (traffic analysis) | Explicitly in-scope per DI-003; documented in operator guide |
| R-010 (DoS via forged frames) | HMAC verification at first router boundary (ADR-009 ordering); forged frames never reach routing logic; per-node keying prevents cross-node forgery |
