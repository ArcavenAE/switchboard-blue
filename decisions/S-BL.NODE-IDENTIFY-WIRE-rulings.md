---
artifact_id: S-BL.NODE-IDENTIFY-WIRE-rulings
document_type: decision
level: ops
version: "1.0"
status: final
producer: architect
timestamp: 2026-07-15T00:00:00Z
cycle: cycle-1
stories_in_scope: [S-BL.NODE-IDENTIFY-WIRE]
bc_traces:
  - BC-2.01.008
related_docs:
  - decisions/identity-cluster-architecture.md v1.1 Section 4
  - decisions/S-BL.DISCOVERY-WIRE-rulings.md v1.11 (control-frame precedent)
  - stories/S-BL.NODE-IDENTIFY-WIRE.md v1.2 Open Design Obligation 2
---

# Ruling: S-BL.NODE-IDENTIFY-WIRE — Challenge-Transcript Wire Format

This document resolves **Open Design Obligation 2** from `S-BL.NODE-IDENTIFY-WIRE.md` v1.2:
the challenge-transcript byte layout for the `NODE_IDENTIFY` handshake (control_type=0x04).

All factual claims are grep/read-verified against the main checkout at
`d249f88` (develop HEAD, post S-BL.CLI-SURFACE-COMPLETION merge).
File:line anchors are cited per claim.

This ruling does NOT resolve Obligations 3-6. Obligations 5 and 6 are BLOCKERS (no
production key material exists). Obligations 3 and 4 (re-identify semantics,
handshake timeout) are gated on `S-BL.ADMISSION-SYNC-WIRE`'s mechanism and require a
separate ruling at scheduling time.

---

## Verified Premises

| Premise | File:Line | Evidence |
|---|---|---|
| `FrameTypeCtl = 0x03` is the outer-header discriminator for all control frames | `internal/frame/frame.go:36` | `FrameTypeCtl FrameType = 0x03 // payload carries a control_type discriminator byte` |
| Control frame payload layout: `[control_type(1), version(1), reserved(2)]` followed by message body | `cmd/switchboard/mgmt_wire.go:691` (DRAIN); `cmd/switchboard/discovery_relay_wire.go:80` (DISCOVERY_RELAY) | DRAIN: `{0x01, 0x01, 0x00, 0x00}`; DISCOVERY_RELAY: `{0x03, 0x01, 0x00, 0x00, nodeAddr(8), seq(8), sessions(...)}` |
| DRAIN and DISCOVERY_RELAY outer headers have zero `HMACTag` — trust boundary is the admitted TCP connection, not a per-frame HMAC | `cmd/switchboard/discovery_relay_wire.go:94-98`; `cmd/switchboard/mgmt_wire.go:691-696` | Explicit comment: "HMACTag is deliberately left as the zero value (AC-015): hop-2's trust boundary is the admitted TCP connection" |
| `SVTNID` IS set in the outer header for DISCOVERY_RELAY (needed for SVTN scoping) but NOT set for DRAIN (global broadcast) | `cmd/switchboard/discovery_relay_wire.go:98-103`; `cmd/switchboard/mgmt_wire.go:692-697` | DISCOVERY_RELAY sets `SVTNID: svtnID`; DRAIN leaves it zero |
| `OuterHeaderSize = 44`, layout: `version(1)+frame_type(1)+payload_len(2)+svtn_id(16)+src_addr(8)+dst_addr(8)+hmac_tag(8)` | `internal/frame/frame.go:14-17` | Constant and field documentation |
| `admission.Challenge` struct: `Nonce [32]byte` + `RouterSig []byte` (Ed25519, always 64 bytes) | `internal/admission/admission.go:189-195` | `Challenge { Nonce [32]byte; RouterSig []byte }` |
| `admission.ChallengeResponse` struct: `NonceSig []byte` (Ed25519, always 64 bytes) | `internal/admission/admission.go:202-206` | `ChallengeResponse { NonceSig []byte }` |
| `admission.GenerateChallenge(routerPrivKey)` generates a 32-byte crypto/rand nonce and signs it: `RouterSig = ed25519.Sign(routerPrivKey, nonce[:])` | `internal/admission/admission.go:428-439` | Signs the nonce slice; Ed25519 signature is always 64 bytes |
| `admission.AdmitNode(challenge, resp, pubKey, svtnID, ks)` verifies `resp.NonceSig` = `ed25519.Sign(nodePrivKey, challenge.Nonce[:])` | `internal/admission/admission.go:457-525` | Signature verify: `ed25519.Verify(pubKey, challenge.Nonce[:], resp.NonceSig)` |
| Ed25519 public key is always 32 bytes, signature is always 64 bytes | `crypto/ed25519` stdlib | `PublicKeySize = 32`, `SignatureSize = 64` |
| `frame.DeriveNodeAddress(svtnID, pubkey)` derives the 8-byte NodeAddr from (svtnID, publicKey) | `internal/admission/admission.go:241,391`; `internal/frame/address.go` | Called at RegisterKey time; `LookupByPubkey` also derives via this |
| `routing.InterfaceID` is `uint64` defined in `internal/routing/split_horizon.go:27` | `internal/routing/split_horizon.go:24-27` | `type InterfaceID uint64` |
| `Router` in `internal/routing` holds the admitted key set and forwarding table, protected by `r.mu sync.RWMutex` | `internal/routing/routing.go:150-157` | `Router { mu sync.RWMutex; admittedKeySet *admission.AdmittedKeySet; forwardingTable ... }` |
| `internal/routing` is ARCH-08 position 5, already imports `internal/admission` | `internal/routing/routing.go:2-16`; `.factory/specs/architecture/ARCH-08-dependency-graph.md` | `admission` is imported; adding identity-map fields requires no new imports |
| `onAccept` in `runRouter` fires as the FIRST ACT of the per-conn goroutine, has access to `net.Conn`, runs before `ServeConn` starts reading | `internal/netingress/netingress.go:177-193` | "fires as the FIRST ACT of the newly spawned per-conn goroutine ... strictly before ServeConn starts reading" |
| `route` closure signature is `func(hdr frame.OuterHeader, payload []byte) error` — no connection access, stateless routing function | `cmd/switchboard/mgmt_wire.go:541` | Closure receives decoded hdr + payload; no `net.Conn` in scope |
| `DISCOVERY_RELAY = 0x03` and `DRAIN = 0x01` are the current control_type registry occupants; `0x02` is RESYNC (reserved, not dispatched) | `cmd/switchboard/discovery_relay_wire.go:28`; `cmd/switchboard/mgmt_wire.go:569` | "reserved-but-undispatched 0x02 RESYNC opcode until S-BL.RESYNC-FRAME lands" |
| `admission.nonceTTL = 60s` — replay prevention window; nonces consumed by `AdmitNode` are recorded in `AdmittedKeySet.nonces` map | `internal/admission/admission.go:142-145,562-583` | Used-nonce set with lazy purge, TTL-gated |
| `sendMap` in `runRouter` is `routing.InterfaceID → *nodeConn` — the fan-out map used by DISCOVERY_RELAY Task 6 | `cmd/switchboard/mgmt_wire.go:538,596,698` | `var sendMap sync.Map` + Store at `h.IfaceID` + `sendMap.Range` in broadcast |

---

## Ruling — NODE_IDENTIFY Challenge-Transcript Wire Format

### 1. Document-choice rationale

A dedicated rulings document (`S-BL.NODE-IDENTIFY-WIRE-rulings.md`) is the right
artifact rather than appending to `identity-cluster-architecture.md` (Section 4).
Rationale:

- Section 4 of the architecture document describes WHAT is specifiable now
  (purpose and readiness). This document specifies HOW (concrete bytes, bounds,
  and method signatures) — the natural separation between design intent and
  wire specification.
- Obligations 3-6 each require their own ruling sections as they are resolved
  at scheduling time. The architecture document would otherwise accumulate
  per-story byte-level implementation detail that properly belongs in a story-scoped
  rulings file, consistent with the DISCOVERY-WIRE precedent.
- `identity-cluster-architecture.md` is a cross-story cluster document; byte layouts
  for one story's wire format do not belong there.

**Story-stub delta required**: the story-writer should add a `see_also` or `rulings_doc`
reference to `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` in the story frontmatter
or provenance section, mirroring how DISCOVERY-WIRE references its rulings doc.
This is a follow-on action, not a gate on this ruling.

---

### 2. Control-frame sub-protocol: single control_type=0x04 with msg_kind discriminator

**DECISION**: The three messages of the NODE_IDENTIFY handshake share a single
`control_type = 0x04` (`NODE_IDENTIFY`) opcode and are distinguished by a `msg_kind`
byte at payload offset `[2]` — replacing the first of the two reserved bytes in the
4-byte control header (the second reserved byte at offset `[3]` is kept hard-reserved
for future extension, consistent with all existing ctl frames).

The BC-2.01.008 opcode registry gains exactly **one new row** (`NODE_IDENTIFY = 0x04`)
for this entire handshake sub-protocol.

**Rationale for single control_type + msg_kind over three separate control_type values:**

The three messages (`NodeIdentify`, `Challenge`, `ChallengeResponse`) are one logical
protocol — a two-round-trip admission handshake. Assigning three sequential opcode
registry entries would fragment what is semantically a single protocol into three
disconnected discriminators. The `msg_kind` byte at payload offset `[2]` provides a
clean sub-protocol layer inside the `0x04` namespace, keeping the opcode registry slot
usage at one and making it unambiguous that all three frames belong to the same
handshake sequence.

The `DISCOVERY_RELAY` precedent is a one-way push (one message, no handshake) so the
sub-protocol question doesn't arise there. NODE_IDENTIFY is the first three-message
exchange in this codebase, warranting the sub-discriminator approach.

**4-byte control header layout for all NODE_IDENTIFY frames:**

```
offset [0]  control_type = 0x04 (NODE_IDENTIFY — BC-2.01.008 registry)
offset [1]  version      = 0x01 (frame.VersionByte)
offset [2]  msg_kind     = 0x01 | 0x02 | 0x03 (see below)
offset [3]  reserved     = 0x00 (hard-reserved; decoder MUST reject non-zero)
```

| msg_kind | Name              | Direction      | Payload after control header |
|----------|-------------------|----------------|------------------------------|
| `0x01`   | `NodeIdentify`    | node → router  | `node_pubkey [32]byte`       |
| `0x02`   | `Challenge`       | router → node  | `nonce [32]byte` + `router_sig [64]byte` |
| `0x03`   | `ChallengeResponse` | node → router | `nonce_sig [64]byte`       |

---

### 3. Outer header fields for all three messages

**Security posture (zero HMACTag — identical to DRAIN/DISCOVERY_RELAY precedent):**

The challenge-response IS the authentication mechanism for this handshake. There is no
per-node `FrameAuthKey` at the time `NodeIdentify` is sent (none has been established
yet — that's what this handshake establishes). Using a zero `HMACTag` in the outer
header for all three messages is correct:

1. Before the handshake completes, no `FrameAuthKey` is available for this connection.
2. The `RouterSig` in the `Challenge` frame and the `NonceSig` in the `ChallengeResponse`
   frame provide mutual authentication across the three messages — the outer header HMAC
   would be redundant.
3. The TCP connection provides transport-layer integrity, preventing MITM injection of
   forged messages (identical reasoning to DRAIN and DISCOVERY_RELAY).

**SVTNID: set in all three messages (distinguishes NODE_IDENTIFY from DRAIN):**

The router must know which SVTN the connecting node is requesting admission to — this
is the key passed to `AdmitNode(challenge, resp, pubKey, svtnID, ks)`. Unlike DRAIN
(global broadcast, no SVTN scoping), NODE_IDENTIFY is SVTN-scoped. All three messages
carry the same `SVTNID` value in the outer header.

**SrcAddr / DstAddr: zero in all three messages (consistent with existing ctl precedent):**

Both DRAIN and DISCOVERY_RELAY leave `SrcAddr` and `DstAddr` as zero in the outer header —
addresses are only meaningful for the data-plane routing path (`RouteFrame`), and ctl
frames do not go through that path. The node's identity is in the payload (`node_pubkey`),
not the outer header address fields. The router derives `NodeAddr = frame.DeriveNodeAddress(svtnID, pubkey)` from the payload after reading the `NodeIdentify` message.

---

### 4. Message 1: NodeIdentify (node → router)

Sent by the connecting node immediately after TCP connect, before any session-data frame.

**Outer header (44 bytes):**

```
[0]      version     = 0x01
[1]      frame_type  = 0x03 (FrameTypeCtl)
[2:4]    payload_len = 36   (big-endian uint16)
[4:20]   svtn_id     = [16-byte SVTN ID the node is joining]
[20:28]  src_addr    = [8 zero bytes]
[28:36]  dst_addr    = [8 zero bytes]
[36:44]  hmac_tag    = [8 zero bytes]
```

**Payload (36 bytes fixed):**

```
[0]     = 0x04 (control_type = NODE_IDENTIFY)
[1]     = 0x01 (version = frame.VersionByte)
[2]     = 0x01 (msg_kind = NodeIdentify)
[3]     = 0x00 (reserved — decoder MUST reject non-zero)
[4:36]  = node_pubkey [32 bytes — Ed25519 public key, ed25519.PublicKeySize]
```

Total frame size: **80 bytes** (44 header + 36 payload).

**Decoder preconditions** (fail-closed; malformed frame closes the connection):
- `hdr.FrameType == FrameTypeCtl` AND `hdr.PayloadLen == 36`
- `payload[0] == 0x04` AND `payload[1] == 0x01` AND `payload[2] == 0x01` AND `payload[3] == 0x00`
- Exactly 32 payload bytes remain after offset [4]

Router action on valid receipt: derive `nodeAddr = frame.DeriveNodeAddress(hdr.SVTNID, payload[4:36])`,
then call `GenerateChallenge(routerPrivKey)` and send the Challenge frame (Message 2).

---

### 5. Message 2: Challenge (router → node)

Sent by the router in response to a valid NodeIdentify frame. Serializes `admission.Challenge`.

**Outer header (44 bytes):**

```
[0]      version     = 0x01
[1]      frame_type  = 0x03 (FrameTypeCtl)
[2:4]    payload_len = 100  (big-endian uint16)
[4:20]   svtn_id     = [16-byte SVTN ID, echoed from the NodeIdentify frame]
[20:28]  src_addr    = [8 zero bytes]
[28:36]  dst_addr    = [8 zero bytes]
[36:44]  hmac_tag    = [8 zero bytes]
```

**Payload (100 bytes fixed):**

```
[0]      = 0x04 (control_type = NODE_IDENTIFY)
[1]      = 0x01 (version = frame.VersionByte)
[2]      = 0x02 (msg_kind = Challenge)
[3]      = 0x00 (reserved — decoder MUST reject non-zero)
[4:36]   = nonce    [32 bytes — Challenge.Nonce, crypto/rand-generated]
[36:100] = router_sig [64 bytes — Challenge.RouterSig, ed25519.SignatureSize]
           = ed25519.Sign(routerPrivKey, nonce[:])
```

Total frame size: **144 bytes** (44 header + 100 payload).

**Decoder preconditions** (fail-closed):
- `hdr.FrameType == FrameTypeCtl` AND `hdr.PayloadLen == 100`
- `payload[0] == 0x04` AND `payload[1] == 0x01` AND `payload[2] == 0x02` AND `payload[3] == 0x00`
- Exactly 32 nonce bytes at `[4:36]` and exactly 64 router_sig bytes at `[36:100]`

Reconstructed `admission.Challenge`:
```go
challenge := admission.Challenge{
    Nonce:     [32]byte(payload[4:36]),
    RouterSig: payload[36:100],
}
```

---

### 6. Message 3: ChallengeResponse (node → router)

Sent by the node in response to a valid Challenge frame. Serializes `admission.ChallengeResponse`.

**Outer header (44 bytes):**

```
[0]      version     = 0x01
[1]      frame_type  = 0x03 (FrameTypeCtl)
[2:4]    payload_len = 68   (big-endian uint16)
[4:20]   svtn_id     = [16-byte SVTN ID, same as NodeIdentify/Challenge]
[20:28]  src_addr    = [8 zero bytes]
[28:36]  dst_addr    = [8 zero bytes]
[36:44]  hmac_tag    = [8 zero bytes]
```

**Payload (68 bytes fixed):**

```
[0]     = 0x04 (control_type = NODE_IDENTIFY)
[1]     = 0x01 (version = frame.VersionByte)
[2]     = 0x03 (msg_kind = ChallengeResponse)
[3]     = 0x00 (reserved — decoder MUST reject non-zero)
[4:68]  = nonce_sig [64 bytes — ChallengeResponse.NonceSig, ed25519.SignatureSize]
          = ed25519.Sign(nodePrivKey, challenge.Nonce[:])
```

Total frame size: **112 bytes** (44 header + 68 payload).

**Decoder preconditions** (fail-closed):
- `hdr.FrameType == FrameTypeCtl` AND `hdr.PayloadLen == 68`
- `payload[0] == 0x04` AND `payload[1] == 0x01` AND `payload[2] == 0x03` AND `payload[3] == 0x00`
- Exactly 64 nonce_sig bytes at `[4:68]`

Reconstructed `admission.ChallengeResponse`:
```go
resp := admission.ChallengeResponse{
    NonceSig: payload[4:68],
}
```

---

### 7. Handshake sequence and direction

The three messages form a single synchronous exchange over the TCP connection, mediated
by the `onAccept` closure in `runRouter`. The `route` function (stateless frame
dispatcher) is NOT used for any of these three messages — they are read and written
directly on `net.Conn` before `netingress.ServeConn` starts reading.

```
Node (connecting)                Router (onAccept closure)
─────────────────────────────    ──────────────────────────────────────────────
  [TCP connect completes]
                                 netingress fires onAccept(conn, h)
  ── NodeIdentify (80 bytes) ──► read from conn (io.ReadFull outer header + payload)
                                 validate frame + pubkey; derive nodeAddr;
                                 call GenerateChallenge(routerPrivKey)
  ◄─ Challenge (144 bytes) ────  write to conn
  ── ChallengeResponse (112) ──► read from conn
                                 call AdmitNode(challenge, resp, pubKey, hdr.SVTNID, ks)
                                 on success: Router.BindInterface(hdr.SVTNID, nodeAddr, h.IfaceID)
                                 onAccept returns cleanup func
                                 netingress.ServeConn starts — normal frame routing begins
  ◄═ session data frames ══════  [bidirectional normal routing, admission verified]
```

**Failure posture**: Any error at any step (malformed frame, key not registered,
bad signature, nonce replay) closes the connection immediately. The connection is never
left in an unbound-but-open state after `onAccept` returns — either the handshake
succeeds and the binding is recorded, or the connection is closed before `ServeConn`
starts. This is the fail-closed posture BC-2.05.001 mandates for admission failures.

---

### 8. Router.BindInterface method signature

Added to `internal/routing` (new method on `*Router` or a new sibling file
`internal/routing/identity.go`). Backed by a new field:

```go
// identityIfaceMap maps (svtnID, nodeAddr) → InterfaceID for the
// DISCOVERY_RELAY fan-out path (S-BL.NODE-IDENTIFY-WIRE; unblocks
// S-BL.DISCOVERY-WIRE Task 6 / AC-017/AC-018).
// Protected by r.mu (same mutex as forwardingTable and admittedKeySet accesses).
identityIfaceMap map[[16]byte]map[[8]byte]InterfaceID
```

**Methods:**

```go
// BindInterface records (svtnID, nodeAddr) → ifaceID after a successful
// NODE_IDENTIFY handshake. Called from onAccept in runRouter on AdmitNode
// success. Last-write-wins (ADR-003): a node reconnect overwrites the prior
// binding — the prior connection's cleanup func removes it via UnbindInterface.
//
// Traces to BC-2.01.008 (NODE_IDENTIFY opcode delivers this binding);
// unblocks S-BL.DISCOVERY-WIRE AC-017, AC-018, Task 6.
func (r *Router) BindInterface(svtnID [16]byte, nodeAddr [8]byte, ifaceID InterfaceID)

// LookupInterface returns the InterfaceID for (svtnID, nodeAddr), or 0 and
// false if no binding exists. Used by the DISCOVERY_RELAY fan-out closure
// (S-BL.DISCOVERY-WIRE Task 6) to resolve a NodeAddr to a send-map key.
func (r *Router) LookupInterface(svtnID [16]byte, nodeAddr [8]byte) (InterfaceID, bool)

// UnbindInterface removes the (svtnID, nodeAddr) binding. Called from the
// per-connection cleanup func (the func() returned by onAccept) when a node
// disconnects, so the identity map stays consistent with sendMap.
func (r *Router) UnbindInterface(svtnID [16]byte, nodeAddr [8]byte)
```

**Concurrency contract**: All three methods hold `r.mu` (write lock for Bind/Unbind,
read lock for Lookup) — identical to `RegisterForwardingEntry` / `LookupInterface`
protocol, consistent with `go.md` rule 12 (return value copies, never internal pointers).

**ARCH-08 impact**: Clean. `internal/routing` is already position 5 in the DAG and
already imports `internal/admission` and `internal/hmac`. Adding `identityIfaceMap` and
three new methods requires zero new imports and zero DAG position changes.

---

### 9. Bounds and guard summary

| Guard | Value | Defect class prevented |
|-------|-------|------------------------|
| Minimum ctl payload length | 4 bytes | E-PRT-002 truncated control frame (existing guard in `route`, also apply in `onAccept` reader) |
| NodeIdentify exact payload size | 36 bytes | Off-by-one pubkey reads (same class as F-DWIP1-001 on the hop-1 HMAC path) |
| Challenge exact payload size | 100 bytes | Truncated nonce (32) or router_sig (64) reads |
| ChallengeResponse exact payload size | 68 bytes | Truncated nonce_sig (64) reads |
| `payload[3] == 0x00` (reserved byte) | hard-reserved | Forward-compat: non-zero reserved byte is a hard decoder error, not a silent ignore (this is a pre-admission connection; lenient decoding creates ambiguity about handshake state) |
| `hdr.SVTNID` non-zero | required for AdmitNode scoping | A zero SVTN ID would match any empty-keyset lookup; the admission keyset lookup itself returns ErrNotAdmitted for unregistered SVTNs, but an explicit SVTNID != zero check makes the precondition explicit |

The `payload_len` in the outer header is the primary guard (enforced by
`frame.ReadOuterFrame` / `io.ReadFull`). The explicit per-message size checks above are
secondary, fail-closed guards against payload_len-matches-but-body-structure-wrong
cases — the same discipline `assembleDiscoveryRelayFrame` applies for DISCOVERY_RELAY.

---

### 10. What remains gated (obligations not addressed by this ruling)

**BLOCKER — Obligation 5 (`S-BL.ADMISSION-SYNC-WIRE`):**
Production code cannot call `admission.AdmitNode` successfully from the router
process because the router's `AdmittedKeySet` is always empty. The wire codec (this
ruling) and the `onAccept` handshake skeleton can be implemented and unit-tested
(with a manually-populated test keyset), but the end-to-end handshake cannot succeed
until `S-BL.ADMISSION-SYNC-WIRE` lands.

**BLOCKER — Obligation 6 (`S-BL.NODE-ADMISSION-PROVISIONING`):**
The node-side `ChallengeResponse` step requires `ed25519.Sign(nodePrivKey, nonce)`. No
production code path supplies the connecting process with a stable admission private key
(`internal/config.Config` has no admission-keypair field; `runAccess`'s `daemonPriv` is
explicitly ephemeral). The node-side signing function can be written as:
```go
func buildChallengeResponse(nodePrivKey ed25519.PrivateKey, challenge admission.Challenge) admission.ChallengeResponse {
    return admission.ChallengeResponse{NonceSig: ed25519.Sign(nodePrivKey, challenge.Nonce[:])}
}
```
but no production caller can invoke it until `S-BL.NODE-ADMISSION-PROVISIONING` lands.

**SCOPING NOTE — Obligation 3 (re-identify / rebind semantics):**
If an already-bound connection sends a second `NodeIdentify` frame, or if a node
reconnects (new TCP, same identity) while the prior binding is live: not addressed here.
`Router.BindInterface` uses LWW semantics (ADR-003) for the binding MAP — but the
question of what happens to the prior TCP connection (close it? coexist?) requires
a separate architect ruling at scheduling time, likely informed by what connection-
lifecycle event hooks `S-BL.ADMISSION-SYNC-WIRE` wires.

**SCOPING NOTE — Obligation 4 (handshake timeout):**
The `onAccept` body blocks on `io.ReadFull` calls. Without a `conn.SetDeadline`, a node
that never completes the handshake (bad clock, network drop, deliberate stall) holds an
`InterfaceID` slot and the `onAccept` goroutine indefinitely. A reasonable default of
30s (`conn.SetDeadline(time.Now().Add(30 * time.Second))` before the first read, cleared
on success) is suggested but requires a ruling at scheduling time on the canonical value.

---

### 11. Security note: RouterSig scope

`admission.GenerateChallenge` signs only the 32-byte nonce
(`RouterSig = ed25519.Sign(routerPrivKey, nonce[:])`). The signed data does NOT include
the node's public key, the SVTN ID, or a connection identifier. This is a pre-existing
property of `internal/admission` — this story introduces zero changes to that package.

The lack of SVTN-ID binding in `RouterSig` does not introduce a new attack path because:
1. Cross-SVTN replay of a captured `(nonce, router_sig)` would still fail at `AdmitNode`
   — the `svtnID`-scoped keyset lookup returns `ErrNotAdmitted` if the node's pubkey is
   registered under SVTN-A but the replay targets SVTN-B.
2. Within-SVTN replay is prevented by the used-nonce set (`admission.nonceTTL = 60s`).
3. The `RouterSig` is intended to prevent nonce *forgery* (a MitM generating a fake
   nonce), not to bind the challenge to a specific identity. The node's identity
   binding happens at `AdmitNode`'s keyset lookup, not at challenge generation.

This is recorded here for completeness. If a future audit determines that binding the
RouterSig to `(svtnID, nodeAddr)` is desirable, that is a change to `internal/admission`
(specifically `GenerateChallenge`) — out of scope for this wire story.

---

## Follow-on Actions

| Action | Owner | Gate |
|--------|-------|------|
| Add `NODE_IDENTIFY = 0x04` row to BC-2.01.008 Postcondition 2 registry table | product-owner | None (Obligation 1; being handled in parallel) |
| Add `rulings_doc` reference to `S-BL.NODE-IDENTIFY-WIRE.md` frontmatter | story-writer | None |
| Resolve Obligation 3 (re-identify / rebind semantics) | architect | Scheduling time; informed by ADMISSION-SYNC-WIRE's connection-lifecycle event model |
| Resolve Obligation 4 (handshake timeout value) | architect | Scheduling time |
| File a ruling section in this document for each resolved obligation | architect | As obligations are resolved |

---

## Summary

| Item | Value |
|---|---|
| Document choice | New `S-BL.NODE-IDENTIFY-WIRE-rulings.md` (not appended to identity-cluster-architecture.md) |
| Ruling | Single control_type=0x04 with msg_kind sub-byte at payload offset [2] |
| BC-2.01.008 opcode registry rows consumed | 1 (NODE_IDENTIFY = 0x04) |
| NodeIdentify frame size | 80 bytes (44 outer header + 36 payload) |
| Challenge frame size | 144 bytes (44 outer header + 100 payload) |
| ChallengeResponse frame size | 112 bytes (44 outer header + 68 payload) |
| HMACTag in all three frames | Zero (trust boundary is the TCP connection; handshake IS the auth) |
| SVTNID in all three frames | Set to the target SVTN (required for AdmitNode scoping) |
| SrcAddr / DstAddr in all three frames | Zero (consistent with DRAIN / DISCOVERY_RELAY ctl precedent) |
| Handshake dispatcher | `onAccept` closure in `runRouter` (not the `route` closure) — reads/writes directly on `net.Conn` before ServeConn starts |
| BindInterface method location | `internal/routing` — new method on `*Router`, backed by new `identityIfaceMap` field |
| BindInterface signature | `BindInterface(svtnID [16]byte, nodeAddr [8]byte, ifaceID InterfaceID)` |
| LookupInterface signature | `LookupInterface(svtnID [16]byte, nodeAddr [8]byte) (InterfaceID, bool)` |
| UnbindInterface signature | `UnbindInterface(svtnID [16]byte, nodeAddr [8]byte)` |
| Concurrent writes gated by | `r.mu` (existing Router mutex) |
| Gated (production ChallengeResponse signing) | S-BL.NODE-ADMISSION-PROVISIONING — node needs a stable private key |
| Gated (end-to-end integration test) | Both S-BL.ADMISSION-SYNC-WIRE (populated router keyset) AND S-BL.NODE-ADMISSION-PROVISIONING |
| Gated (rebind semantics ruling) | S-BL.ADMISSION-SYNC-WIRE (informs connection-lifecycle model) |
| Gated (timeout ruling) | Scheduling time — suggest 30s SetDeadline, needs ratification |
| New flag surfaced | RouterSig scope (nonce-only signing in GenerateChallenge) — pre-existing property of admission package, not a new concern for this wire story; documented for future audit transparency |
