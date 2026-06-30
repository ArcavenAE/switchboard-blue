---
artifact_id: ARCH-04-admission-security
document_type: architecture-section
level: L3
version: "1.11"
status: draft
producer: architect
timestamp: 2026-06-29T00:00:00
# v1.11 (2026-06-29): F-P2-005 — align ErrRoleMismatch sentinel string to implementation:
#   "revoke: supplied role does not match registered role" →
#   "admission: role mismatch: stored role differs from expected role"
#   (implementation is in internal/admission/admission.go, not internal/svtnmgmt/svtnmgmt.go).
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
  - 2026-06-29T00:00:00 # v1.9 — F-005 ruling: Lookup/LookupByPubkey return-type convention — Option A chosen ((AdmittedKey, bool) value + present-flag); migration deferred to DRIFT-F005-LOOKUP-CONVENTION follow-on story
  - 2026-06-29T00:00:00 # v1.10 — Lens1 F-004: rename `readonly` → `access` in role-hierarchy prose (RoleAccess canonical); Lens2 F-CS-002: ADR-004 Addendum H2 — atomic RevokeKeyIfRoleMatches primitive added; error codes updated to E-ADM-018 (confirm-gate) and E-ADM-019 (role-mismatch); BC-2.07.001 v1.2 PC-2 cross-link; Task 6 note distinguishing daemon-mode exclusion vs caller-key-role (BC-5.39.001)
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

**OQ-003 resolution:** A permission hierarchy exists among key roles: control > console > access. A console-role or access-role key cannot revoke a control-role key (such attempts fail with E-ADM-011). Control-to-control revocation requires `sbctl admin` human authorization (split-brain mitigation, ADR-004 paragraph above).

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

**Role hierarchy (explicit — F-010):** `control > console > access`. Lower-tier
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

## ADR-004 Addendum: RevokeKey `currentRole` Parameter — Caller-Supplied vs. Internal Lookup (HOLD-001, S-6.02)

**Decision date:** 2026-06-29
**Resolved by:** architect (HOLD-001 from S-6.02 test-writer)
**Applies to:** `SVTNManager.RevokeKey` signature in `internal/svtnmgmt`

### Decision: Hybrid — caller-supplied with mandatory manager-side cross-check

The `currentRole admission.KeyRole` parameter is RETAINED as a caller-supplied argument.
The implementation MUST cross-check the supplied role against the role stored in the
`AdmittedKeySet` registry before applying the confirm gate. If the supplied role diverges
from the stored role, `RevokeKey` MUST return an error (distinct sentinel, E-ADM-019,
`ErrRoleMismatch: "revoke: supplied role does not match registered role"`). The confirm
gate (`ErrControlRevocationRequiresConfirm`) is applied only AFTER the role check passes.

### Rationale

**Who is the caller?** `cmd/sbctl admin revoke-key`, running on the control node under
credentials already authenticated by the mgmt-plane Ed25519 handshake (S-6.03, ADR-006).
The caller is not an untrusted external party — it is the local operator CLI.

**Why not pure internal lookup (Option B)?** The caller-supplied role is not being
used to grant authority; it is used as a *classification input* that determines whether
the confirm gate fires. Authority is established by the mgmt-plane handshake, not by the
role parameter. Dropping the parameter would change the API surface, forcing a rewrite of
all S-6.02 test call sites with no security gain.

**Why not bare caller-supplied with no cross-check (Option A, pure)?** If the caller
passes `RoleConsole` for a key that is actually `RoleControl` in the registry, the confirm
gate is bypassed for what is in fact a control-to-control revocation. This is a local
privilege-escalation path available to any code that can reach the mgmt socket, even if
the socket is local-only. The cross-check closes it.

**Why hybrid is strictly more secure than pure Option B:** It catches both (a) a
caller that misrepresents the target key's role (role mismatch error before the confirm
gate) and (b) a caller that correctly identifies a control key but omits confirm (existing
`ErrControlRevocationRequiresConfirm` gate). Option B catches only (b).

**`currentRole` semantics post-decision:** The parameter is a *declared intent and
confirmation token*. The operator is asserting "I know this key is role X and I intend
to revoke a key of that role." The manager verifies the assertion against the registry
before proceeding. This is analogous to optimistic locking: the caller states its
expectation; the server validates it.

### Implementation Invariant for `SVTNManager.RevokeKey`

```
1. Validate svtnName exists → ErrSVTNNotFound if not
2. Look up target key in AdmittedKeySet via AdmittedKeySet.LookupByPubkey(svtnID, pubkey)
   — do NOT call frame.DeriveNodeAddress directly in svtnmgmt (ARCH-08 §6.5 position 15:
   svtnmgmt imports only {admission, config}; derivation is admission's internal concern).
   → ErrKeyNotRegistered (E-ADM-013) if LookupByPubkey returns nil
3. Compare stored.Role == currentRole
   → ErrRoleMismatch (E-ADM-019) if they diverge
4. If currentRole == RoleControl AND confirm == false
   → ErrControlRevocationRequiresConfirm (E-ADM-018)
5. Call AdmittedKeySet.RevokeKey → propagate error
6. Return KeyOpResult{Fingerprint, At: time.Now().UTC()}
```

### Sentinel Error Required

Implemented in `internal/admission/admission.go` (not `internal/svtnmgmt/svtnmgmt.go` — the atomic primitive lives at the admission layer):

```go
// ErrRoleMismatch is returned by RevokeKeyIfRoleMatches when the caller-supplied
// currentRole does not match the role stored in the AdmittedKeySet registry
// (E-ADM-019). This prevents the confirm gate from being bypassed by supplying
// a lower role for a control key.
var ErrRoleMismatch = errors.New("admission: role mismatch: stored role differs from expected role")
```

### Test Validity Assessment

All tests in `internal/svtnmgmt/svtnmgmt_test.go` as written against the current
caller-supplied API remain VALID. No call sites need to change. The tests correctly:
- Pass `RoleConsole` when revoking a console key (role matches → no mismatch error)
- Pass `RoleControl` when revoking a control key (role matches → confirm gate fires)
- Expect `ErrControlRevocationRequiresConfirm` on control revoke without confirm

A NEW test should be added (but is not required before green — it covers a defence-in-depth
path, not an AC):

```go
// TestSVTNManager_RevokeKey_RoleMismatchReturnsError (optional, defence-in-depth)
// RevokeKey with a role that does not match the registered role returns ErrRoleMismatch.
```

This test can be added by the implementer alongside the implementation; it does not block
the Red Gate.

## CR-001 Ruling: `internal/svtnmgmt` Must Not Import `internal/frame` (ARCH-08 §6.5 position 15)

**Decision date:** 2026-06-29
**Resolved by:** architect (CR-001 from S-6.02 code review)
**Applies to:** `internal/svtnmgmt/svtnmgmt.go` lines 225 and 288; `internal/admission/admission.go`

### Violation

`internal/svtnmgmt` imports `github.com/arcavenae/switchboard/internal/frame` (line 25 of
`svtnmgmt.go`) to call `frame.DeriveNodeAddress(svtnID, []byte(pubkey))` at lines 225 and 288.
ARCH-08 §6.5 position 15 enumerates svtnmgmt's permitted internal imports as `{admission, config}`
exhaustively. `internal/frame` is not permitted. The package doc's own constraint statement
(lines 8–11) is correct; the import violates it.

### Resolution

**Add `AdmittedKeySet.LookupByPubkey` to `internal/admission`.** The derivation of nodeAddr from
(svtnID, pubkey) is already admission's internal concern — `RegisterKey` and `AdmitNode` both call
`frame.DeriveNodeAddress` internally. svtnmgmt must not replicate that dependency.

### New Symbol: `AdmittedKeySet.LookupByPubkey`

Add to `internal/admission/admission.go` after the `Lookup` method:

```go
// LookupByPubkey returns a copy of the AdmittedKey for the node identified by
// (svtnID, pubkey), or nil if not found. The 8-byte node address is derived
// internally via frame.DeriveNodeAddress — callers do not need to import
// internal/frame (ARCH-08 §6.5 position 15).
//
// Returns a value copy; the backing array of PublicKey is deep-cloned
// (go.md rule 12; finding-032-store-sync-contract-leak).
//
// Traces to BC-2.05.004 (key lookup by public key identity).
func (s *AdmittedKeySet) LookupByPubkey(svtnID [16]byte, pubkey ed25519.PublicKey) *AdmittedKey {
    nodeAddr := frame.DeriveNodeAddress(svtnID, []byte(pubkey))
    return s.Lookup(svtnID, nodeAddr)
}
```

Thread-safety and deep-clone guarantees are inherited from `Lookup`. No mutex logic needed here.

### svtnmgmt Changes

1. Remove `"github.com/arcavenae/switchboard/internal/frame"` import.
2. In `RevokeKey` (line 225): replace the two-line derive+lookup with `stored := m.keySet.LookupByPubkey(svtnID, pubkey)`.
3. In `ExpireKey` (line 288): replace `nodeAddr := frame.DeriveNodeAddress(...)` with a `LookupByPubkey` call; read `nodeAddr` from the returned `AdmittedKey.NodeAddr` for the subsequent `SetKeyExpiry` call. Return `ErrKeyNotRegistered` if nil.

### ADR-004 Addendum H2: Atomic Revocation Primitive — `RevokeKeyIfRoleMatches` (HOLD-001 TOCTOU Resolution, F-CS-002)

**Decision date:** 2026-06-29
**Resolved by:** architect (F-CS-002 HOLD-001 TOCTOU, Lens2 adversarial pass)
**Applies to:** `AdmittedKeySet` in `internal/admission`

#### Mandate

The non-atomic Lookup → cross-check → RevokeKey sequence (formerly Steps 2/3/5 of the
HOLD-001 hybrid analysis) introduces a TOCTOU window: between the `LookupByPubkey` call
and the `RevokeKey` call, a concurrent `RegisterKey` (LWW, ADR-003) could change the key's
role, causing the role cross-check performed in `SVTNManager.RevokeKey` to compare against
a stale value.

To close this window, `AdmittedKeySet` MUST expose a new atomic primitive:

```go
// RevokeKeyIfRoleMatches atomically looks up the key identified by (svtnID, pubkey),
// compares its stored role to expectedRole, and — if they match — marks the key revoked.
// The entire operation executes under a single Write lock.
//
// Returns:
//   - (existingRole, nil) on success (role matched; key marked revoked).
//   - (existingRole, ErrRoleMismatch) if stored role ≠ expectedRole (maps to E-ADM-019);
//     the key is NOT mutated.
//   - (0, ErrKeyNotRegistered) if the key is not present (maps to E-ADM-013).
//
// ErrRoleMismatch maps to E-ADM-019.
// ErrKeyNotRegistered maps to E-ADM-013.
func (s *AdmittedKeySet) RevokeKeyIfRoleMatches(
    svtnID [16]byte,
    pubkey ed25519.PublicKey,
    expectedRole admission.KeyRole,
) (existingRole admission.KeyRole, err error)
```

#### Critical Section Contract

The single Write lock acquisition covers:
1. Derive `nodeAddr` via `frame.DeriveNodeAddress(svtnID, []byte(pubkey))` (internal).
2. Look up the entry. If absent → release lock, return `(0, ErrKeyNotRegistered)`.
3. Read `entry.Role`. If `entry.Role != expectedRole` → release lock, return
   `(entry.Role, ErrRoleMismatch)` without mutating. No state change.
4. Mark `entry.revoked = true`. Release lock.
5. Return `(entry.Role, nil)`.

No intermediate lock release between steps 2–4. This eliminates the TOCTOU gap.

#### SVTNManager.RevokeKey Integration

`SVTNManager.RevokeKey` MUST call `RevokeKeyIfRoleMatches` instead of the two-step
`LookupByPubkey` + `RevokeKey` sequence:

```
1. Validate svtnName exists → ErrSVTNNotFound if not
2. Call AdmittedKeySet.RevokeKeyIfRoleMatches(svtnID, pubkey, currentRole)
   → ErrKeyNotRegistered (E-ADM-013) if not present
   → ErrRoleMismatch (E-ADM-019) if role diverges
   → returns (existingRole, nil) on success
3. If currentRole == RoleControl AND confirm == false
   → ErrControlRevocationRequiresConfirm (E-ADM-018)
   (use existingRole returned from step 2, not the caller-supplied currentRole,
    for the confirm-gate check — they are guaranteed equal at this point)
4. Return KeyOpResult{Fingerprint, At: time.Now().UTC()}
```

The previous five-step sequence (Steps 2/3/5 annotated in HOLD-001) is superseded by
this two-call sequence. The confirm gate (step 3) still fires on the role returned from
the atomic step, not the caller's assertion, removing the residual TOCTOU concern.

#### BC-2.07.001 v1.2 PC-2 Cross-Reference

The atomic revocation primitive satisfies BC-2.07.001 v1.2 PC-2 (trust-anchor addendum):
a control node's revocation of another key is required to be role-consistent and atomic.
Specifically, the RevokeKeyIfRoleMatches primitive is the mechanism by which PC-2's
"only a control-role key may destroy an SVTN or revoke a control key" invariant is
enforced at the admission layer without a TOCTOU gap.

### Daemon-Mode vs Caller-Key-Role: Inv-3 Disambiguation (Task 6 — BC-2.07.001 Inv-3, BC-5.39.001)

ADR-004 addresses two distinct role-enforcement concerns that must not be conflated:

| Concern | Layer | Mechanism | BC Anchor |
|---------|-------|-----------|-----------|
| **Daemon-mode exclusion** | Process-level | A daemon launched in `access` or `console` mode MUST NOT register as a control node; enforced at startup by mode flags and rejected by `SVTNManager.RegisterKey` if a non-control-mode daemon attempts control operations | BC-5.39.001 (daemon role exclusion) |
| **Caller-key-role check** | Per-RPC | Each RPC that mutates SVTN state (create, destroy, revoke) checks the caller's key role in `AdmittedKeySet` before executing; enforced by `RevokeKeyIfRoleMatches` (this addendum) and the SVTN create/destroy guards | BC-2.07.001 Inv-3 (S-6.06, S-6.07) |

The daemon-mode exclusion is a process-level architectural constraint: it prevents
the wrong binary mode from registering control capabilities at boot time. The
caller-key-role check is a per-RPC runtime check: it prevents any connected client —
regardless of which mode the daemon was launched in — from performing privileged
operations if its key is not registered as `RoleControl` in the admission registry.

Both constraints must hold simultaneously. The daemon-mode constraint is a necessary
condition for a well-formed deployment; the per-RPC constraint is the cryptographic
enforcement layer that closes the authorization gap even if the process-level constraint
is somehow bypassed.

**BC-2.07.001 Inv-3** ("Only control-role keys may create or destroy SVTNs") is anchored
to S-6.06 (SVTN destroy) and S-6.07 (admin.svtn.create handler + CLI). Inv-3 is enforced
by the per-RPC caller-key-role check, not by daemon-mode exclusion alone.

### ADR-004 Original Decision (unchanged)

The core ADR-004 decision is unchanged: key management is exclusive to control nodes;
console-to-control revocation is prohibited; control-to-control revocation requires
`sbctl admin` human authorization. This addendum narrows the implementation contract for
the confirm gate only.
| R-010 (DoS via forged frames) | HMAC verification at first router boundary (ADR-009 ordering); forged frames never reach routing logic; per-node keying prevents cross-node forgery |

## F-005 Ruling: `Lookup` / `LookupByPubkey` Return-Type Convention

**Finding source:** adversary pass 1 on S-6.02, finding F-005 (MEDIUM, Security/Concurrency lens)
**Decision date:** 2026-06-29
**Resolved by:** architect
**Applies to:** `AdmittedKeySet.Lookup` and `AdmittedKeySet.LookupByPubkey` in `internal/admission`

### Decision: Option A — change signature to `(AdmittedKey, bool)`

The `Lookup` and `LookupByPubkey` methods MUST be changed from `*AdmittedKey` to
`(AdmittedKey, bool)`. The second return value is a present-flag following the idiomatic
Go map-lookup pattern. Callers change from `if stored == nil` to `if !ok`.

### Evidence

**Struct size:** `AdmittedKey` is approximately 92 bytes on 64-bit targets:
`PublicKey []byte` (24), `NodeAddr [8]byte` (8), `Role KeyRole / uint8` (1),
`FrameAuthKey [32]byte` (32), `expiry time.Time` (24), `revoked bool` (1),
`admitted bool` (1), plus alignment padding to ~96 bytes. This is below the
~64-byte "prefer pointer receiver" threshold at which copying meaningfully hurts
performance (go.md rule 9), and admission lookups are not on the per-frame hot path
(IsAdmitted and HMAC key retrieval are; Lookup is a management-plane operation).

**Project-wide convention survey (locked accessors returning data):**
- `session.Publisher.ListSessions()` → `[]Info` (value slice)
- `session.Publisher.Get()` → `(Info, error)` (value + error)
- `paths.PathTracker.Snapshot()` → `PathSnapshot` (value; comments explicitly cite go.md rule 12)
- `paths.PathTracker.Score()`, `RTT()`, `LossPct()`, `IsActive()`, `IsDegraded()` → all primitives/values
- `routing.SVTNRoute` → reads `*ForwardingEntry` under lock but copies a `[32]byte` field before returning; does NOT return the pointer to callers

`Lookup` returning `*AdmittedKey` is the **sole pointer-return outlier** in the codebase.
Every other locked accessor follows the value-return or value-with-bool/error pattern.

**Documentary inconsistency:** The `Lookup` godoc says "Returns a value copy" while
the signature returns a pointer. The body correctly deep-clones the struct (go.md rule 12
compliance in implementation) but the pointer return type contradicts both the godoc and
the project convention, inviting future callers to write pointer-nil checks instead of
bool-checks, and creating a false API signal that the pointer might be "live" state.

**go.md rule 9 note:** Rule 9 prescribes pointer receivers for large structs (>~64 bytes).
`AdmittedKey` at ~92 bytes sits in the border zone for receiver style, but the question
here is return type (snapshot, not receiver), which is governed by rule 12. Rule 12
prescribes value returns from locked accessors unconditionally; it does not carve out an
exception for large structs. The correct return-type convention is `(AdmittedKey, bool)`.

### Rejected Alternative: Option B (keep pointer, improve godoc)

Option B — retaining `*AdmittedKey` and adding a prominent godoc warning that mutations
are not propagated — was rejected for three reasons:

1. It contradicts the project-wide locked-accessor convention (all others return values).
   The convention, not the implementation cleverness of the deep-clone, is the defensive
   layer. A future developer sees `*AdmittedKey` and infers pointer semantics.
2. It perpetuates the type-signature/godoc mismatch (godoc says "value copy"; pointer
   says otherwise).
3. Codifying "pointer return is OK here because we deep-clone" creates a precedent that
   would need to be re-argued for every future locked accessor. The rule should be uniform.

### Caller Migration

Current callers (as of 2026-06-29):
- `AdmittedKeySet.LookupByPubkey` delegates to `Lookup` — both signatures change together.
- `internal/svtnmgmt.SVTNManager.RevokeKey` and `ExpireKey` (in S-6.02 worktree) call
  `LookupByPubkey` and currently check `if stored == nil`. These become `stored, ok :=
  ...; if !ok { return ErrKeyNotRegistered }`.

Migration is a **one-line change per call site**: nil-check → bool-check. No logic changes.

### Migration Plan

**This migration MUST NOT be applied to the S-6.02 worktree mid-flight.** The S-6.02
implementer is actively delivering F-003 and F-006. A signature change now would collide.

The migration is tracked as `DRIFT-F005-LOOKUP-CONVENTION` in the tech-debt register.
Target: a dedicated follow-on story in Wave 6, estimated 1 point. See tech-debt register
for full migration checklist.

### Implementation Invariant Post-Migration

After migration, the canonical Lookup signature is:

```go
func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool)
func (s *AdmittedKeySet) LookupByPubkey(svtnID [16]byte, pubkey ed25519.PublicKey) (AdmittedKey, bool)
```

The body logic is unchanged: deep-clone of `PublicKey` backing array is still required
(M-3) because `ed25519.PublicKey` is `[]byte` and a value copy of the struct shares the
slice header's backing array.

```go
// RIGHT — value return with present-flag
func (s *AdmittedKeySet) Lookup(svtnID [16]byte, nodeAddr [8]byte) (AdmittedKey, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    svtnMap, ok := s.keys[svtnID]
    if !ok {
        return AdmittedKey{}, false
    }
    entry, ok := svtnMap[nodeAddr]
    if !ok {
        return AdmittedKey{}, false
    }
    cp := *entry
    cp.PublicKey = append(ed25519.PublicKey(nil), entry.PublicKey...)
    return cp, true
}
```
