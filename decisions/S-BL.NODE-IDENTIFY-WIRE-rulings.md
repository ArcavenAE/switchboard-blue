---
artifact_id: S-BL.NODE-IDENTIFY-WIRE-rulings
document_type: decision
level: ops
version: "1.3"
status: final
producer: architect
timestamp: 2026-07-15T00:00:00Z
modified:
  - 2026-07-18T00:00:00Z
  - 2026-07-19T00:00:00Z
cycle: cycle-1
stories_in_scope: [S-BL.NODE-IDENTIFY-WIRE]
bc_traces:
  - BC-2.01.008
related_docs:
  - decisions/identity-cluster-architecture.md v1.2 Section 4
  - decisions/S-BL.DISCOVERY-WIRE-rulings.md v1.11 (control-frame precedent)
  - stories/S-BL.NODE-IDENTIFY-WIRE.md v1.4 Open Design Obligations
  - decisions/S-BL.ADMISSION-SYNC-WIRE-rulings.md (PR #126 @ 92a2c65)
  - decisions/S-BL.NODE-ADMISSION-PROVISIONING-rulings.md (PR #125 @ ce06f6a)
---

# Ruling: S-BL.NODE-IDENTIFY-WIRE — Challenge-Transcript Wire Format and Handshake Semantics

This document resolves all Open Design Obligations for `S-BL.NODE-IDENTIFY-WIRE`.

**v1.0 (2026-07-15)** resolved Obligation 2: challenge-transcript byte layout for
`control_type=0x04` (`NODE_IDENTIFY`), including frame layouts, handshake sequence,
`Router.BindInterface/LookupInterface/UnbindInterface` method signatures, and bounds
guards. All factual claims in §§1–11 were verified against develop HEAD at `d249f88`.

**v1.1 (2026-07-18)** resolves all remaining obligations (3, 4, O-1) and documents
Obligations 5 and 6 as resolved by delivery. All v1.1 factual claims are verified
against develop HEAD at `92a2c65` (post `S-BL.ADMISSION-SYNC-WIRE` merge, PR #126).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.3 | 2026-07-19 | §17 added: E-ADM-023 connection-teardown mechanism ruling. Adjudicates the production defect where `route()` case 0x04 returned an error that `netingress.ServeConn` dropped (drop-and-continue contract). Ruling: Option B — per-conn `net.Conn` close handle injected into the route closure for the E-ADM-023 path only; no RouteFn contract change; E-ADM-016/017 drop-and-continue preserved. BC-2.01.009 Invariant 7 mechanism note added (how "close immediately" is achieved). Follow-on Actions table updated with downstream cascade. POL-001: modified entry added. |
| 1.2 | 2026-07-18 | Errata §8 and Summary: `UnbindInterface` signature corrected from 2-arg `(svtnID, nodeAddr)` to 3-arg `(svtnID [16]byte, nodeAddr [8]byte, callerIfaceID InterfaceID)`. The 2-arg form was inconsistent with PC-9's stale-cleanup guard (BC-2.01.010), which requires the caller's own `ifaceID` to guard against LWW-overwrite-then-stale-cleanup deleting the new binding. No behavioral change — PC-9's guard semantics are unchanged; only the signature is corrected to make them implementable. Red Gate stubbing surfaced this contradiction; errata adjudicated by architect. BC-2.01.010 aligned to 3-arg form (v1.2). Story body (line 175, Task 15, AC-012) requires cascade update by story-writer/spec-steward. |
| 1.1 | 2026-07-18 | Sections 12–16 added. Obligation 3 resolved: LWW overwrite on reconnect; second NodeIdentify on same conn = hard error. Obligation 4 resolved: `nodeIdentifyHandshakeTimeout = 10s`; failure path table with E-ADM-* codes; eventual-consistency race disposition. Obligations 5/6 marked RESOLVED-BY-DELIVERY citing PR #125 (node keypair) and PR #126 (admission-sync). O-1 resolved: `AdmitNode` must gain expiry check (Option A); BC-2.05.001 amendment required. Follow-on Actions table updated. Summary table updated. Human confirmation flag raised for O-1 policy change. POL-001: `modified:` frontmatter entry added. |
| 1.0 | 2026-07-15 | Initial ruling: wire format for Obligation 2 (challenge-transcript byte layout). Obligations 5/6 identified as blockers; Obligations 3/4 gated. |
File:line anchors are cited per claim.

**v1.0 (2026-07-15):** This ruling resolved Obligations 1 and 2 (opcode registry,
challenge-transcript wire format). Obligations 5 and 6 were BLOCKERS (no production key
material existed). Obligations 3 and 4 were gated.

**v1.1 (2026-07-18):** All obligations resolved. Prerequisites `S-BL.ADMISSION-SYNC-WIRE`
(PR #126 @ 92a2c65) and `S-BL.NODE-ADMISSION-PROVISIONING` (PR #125 @ ce06f6a) have
been delivered. This version adds Sections 12–16 resolving Obligations 3, 4, O-1, and
the 5/6 resolved-by-delivery status update. See §§12–16.

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

// UnbindInterface removes the (svtnID, nodeAddr) binding if the stored
// ifaceID matches callerIfaceID (stale-cleanup guard — PC-9 of BC-2.01.010).
// Called from the per-connection cleanup func (the func() returned by onAccept)
// when a node disconnects, so the identity map stays consistent with sendMap.
// callerIfaceID is the connection's own InterfaceID (h.IfaceID from onAccept).
func (r *Router) UnbindInterface(svtnID [16]byte, nodeAddr [8]byte, callerIfaceID InterfaceID)
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

### 10. Obligation status (v1.1 update)

| Obligation | Status | Resolution |
|---|---|---|
| 1 — BC-2.01.008 opcode registry row | **RESOLVED** (v1.0) | `NODE_IDENTIFY = 0x04` registered |
| 2 — Challenge-transcript wire format | **RESOLVED** (v1.0) | §§2–9 above |
| 3 — Re-identify / rebind semantics | **RESOLVED** (v1.1) | §12 |
| 4 — Failure paths and handshake timeout | **RESOLVED** (v1.1) | §13 |
| 5 — Router AdmittedKeySet always empty | **RESOLVED-BY-DELIVERY** (PR #126) | §14 |
| 6 — Node has no stable admission keypair | **RESOLVED-BY-DELIVERY** (PR #125) | §15 |
| O-1 — AdmitNode does not check expiry | **RESOLVED** (v1.1) | §16 |

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

---

### 12. Obligation 3 — Re-identify / rebind semantics (RESOLVED)

#### Code baseline (disk-verified at develop@92a2c65)

`ADMISSION-SYNC-WIRE` (PR #126) delivered `wireAdmissionSyncHandlers` registering
four router-side push RPC handlers. It does NOT deliver per-connection lifecycle hooks:
no `onDisconnect`, no `UnregisterObserver`-style teardown callback, and no per-node
observer registration (the drain observer model from `S-7.04-FU-DRAIN-WIRE` is a
global-broadcast pattern, not per-node). The `onAccept` cleanup func returned to
`netingress.Serve` (current implementation: `sendMap.Delete(h.IfaceID)` + drain-done
close) is the only connection-teardown event available from `runRouter`.

`Router.BindInterface`, `LookupInterface`, and `UnbindInterface` do NOT exist in
`internal/routing` yet — they are still to-be-built by this story (as specified in §8).
The existing `Router` in `internal/routing/routing.go:150–157` holds
`mu sync.RWMutex`, `admittedKeySet`, and `forwardingTable` but no `identityIfaceMap`.

#### Decision: LWW overwrite for binding map; connection NOT torn down

**RULING: A second `NodeIdentify` from an already-bound connection, or a reconnecting
node with the same admitted identity (new TCP connection), causes `BindInterface` to
overwrite the prior `(SVTNID, NodeAddr) → IfaceID` binding. The prior TCP connection
is NOT actively torn down.**

Rationale:

1. **ADR-003 (LWW, last-write-wins) is the existing ordering invariant for keyset
   mutations.** Extending it to the `identityIfaceMap` is consistent with the codebase's
   established concurrency model. The alternative (reject-if-bound) would require the
   router to track "which connection holds the binding" and actively close it — a new
   state machine that does not exist today.

2. **The stale connection is self-removing.** The prior TCP connection's cleanup func
   (the `func()` returned by `onAccept`) calls `UnbindInterface(svtnID, nodeAddr)` when
   the connection eventually closes. Until that close, two IfaceIDs for the same
   `(SVTNID, NodeAddr)` may transiently coexist in `sendMap` — but only the latest
   binding is visible to `LookupInterface`. The stale connection will continue to receive
   frames forwarded to its IfaceID directly (sendMap lookup by IfaceID still works), but
   `LookupInterface` returns the new IfaceID, so discovery fan-out resolves to the new
   connection. This is acceptable: the stale connection's traffic is self-extinguishing
   as its TCP keepalive or application-level activity detects the dead connection.

3. **Security: rebind REQUIRES full re-handshake.** A reconnecting node must go through
   the full `NodeIdentify → Challenge → ChallengeResponse → AdmitNode` sequence on the
   new TCP connection. The overwrite only occurs after `AdmitNode` returns nil. A
   different public key claiming the same `NodeAddr` cannot overwrite an existing binding
   because `AdmitNode` verifies the signature against the registered public key — a node
   that was admitted under key `K1` and whose binding a different public key `K2` attempts
   to hijack will fail `AdmitNode` with `ErrNotAdmitted` (K2 is not registered for this
   `NodeAddr`). The overwrite is therefore same-identity rebind only.

4. **No active connection-teardown in `runRouter` precedent.** DRAIN (S-7.04-FU-DRAIN-WIRE)
   broadcasts to all connections but does not selectively tear down individual connections.
   `DISCOVERY_RELAY` has no teardown path. Introducing selective per-node TCP close as a
   new pattern for the rebind case adds non-trivial complexity (race with the writer
   goroutine's `nc.done` close) without operational benefit given point 2 above.

#### Concurrency contract

`BindInterface(svtnID, nodeAddr, ifaceID)` acquires `r.mu` write lock — same discipline
as `RegisterForwardingEntry` and the existing `forwardingTable` mutations. `UnbindInterface`
also acquires write lock. `LookupInterface` acquires read lock. Three-method pattern is
identical to the concurrency model documented in §8.

The `onAccept` cleanup func MUST call `UnbindInterface(svtnID, nodeAddr)` in addition to
`sendMap.Delete(h.IfaceID)`. This is the only teardown needed — no additional connection-
lifecycle plumbing is required.

#### Same-identity re-identify on the SAME connection

If an already-bound connection sends a second `NodeIdentify` frame (unusual — this would
be an application-level protocol violation by the connecting node), the router MUST
treat it as an error: the connection already has a binding. The router closes the
connection immediately and logs an error. This is fail-closed: a well-behaved node
never sends a second `NODE_IDENTIFY` on the same connection. This is distinct from
the reconnect (new TCP) case above.

**Decision: hard error + connection close on second `NodeIdentify` from same connection.**

Implementation note: `onAccept` can track whether the handshake has already completed
using a local bool in the closure. If `NODE_IDENTIFY` arrives after the handshake is
complete, close the connection and log.

---

### 13. Obligation 4 — Failure paths and handshake timeout (RESOLVED)

#### Handshake timeout value and enforcement point

**RULING: `const nodeIdentifyHandshakeTimeout = 10 * time.Second`**

Set via `conn.SetDeadline(time.Now().Add(nodeIdentifyHandshakeTimeout))` at the start
of the `onAccept` closure, before the first `io.ReadFull`. Clear the deadline on
successful completion of the handshake (`conn.SetDeadline(time.Time{})`).

Rationale: `admission_sync_client.go:154` establishes `const handshakeTimeout = 10 *
time.Second` for the ADMISSION-SYNC-WIRE push-RPC handshake. The NODE_IDENTIFY
handshake is a similar three-message synchronous exchange over a live TCP connection.
Using the same value preserves internal consistency and avoids introducing a second
timeout constant with a different value and no justification for the difference.
The v1.0 ruling's 30s suggestion is superseded by this canonically-established 10s
value (10s is sufficient for all three frames totalling 336 bytes over any plausible
management-plane link; 30s would allow deliberate stalls to hold IfaceID slots for an
unnecessarily long window).

#### Enumerated failure paths

| Path | Trigger | Connection disposition | Log / error code |
|---|---|---|---|
| Malformed `NodeIdentify` frame | `hdr.PayloadLen != 36`, wrong `control_type/version/msg_kind/reserved` | Close immediately | Log WARN: "node_identify: malformed NodeIdentify frame: {reason}" |
| `SVTNID == zero` | All-zero bytes in outer header SVTN ID field | Close immediately | Log WARN: "node_identify: zero SVTN ID rejected" |
| `AdmitNode` returns `ErrNotAdmitted` | Node's pubkey not registered for this SVTN | Close immediately | Log WARN: "node_identify: E-ADM-003 not admitted svtn={svtnID}" |
| `AdmitNode` returns `ErrKeyRevoked` | Node's key has been revoked | Close immediately | Log WARN: "node_identify: E-ADM-005 key revoked svtn={svtnID}" |
| `AdmitNode` returns `ErrNonceReplay` | Challenge nonce already consumed | Close immediately | Log WARN: "node_identify: E-ADM-008 nonce replay svtn={svtnID}" |
| `AdmitNode` returns `ErrSignatureVerificationFailed` | `NonceSig` does not verify against pubkey | Close immediately | Log WARN: "node_identify: E-ADM-001 sig verify failed svtn={svtnID}" |
| `AdmitNode` returns `ErrKeyExpired` | Key expired (§16 ruling applied — expiry enforced) | Close immediately | Log WARN: "node_identify: E-ADM-015 key expired svtn={svtnID}" |
| Handshake timeout | 10s elapsed without completing three-message exchange | Close immediately (SetDeadline fires, io.ReadFull returns deadline-exceeded error) | Log WARN: "node_identify: handshake timeout svtn={svtnID}" |
| Eventual-consistency race (key not yet pushed) | Control registered a key, push to this router not yet delivered; node connects immediately | Indistinguishable from `ErrNotAdmitted` — close immediately | Same as `ErrNotAdmitted` path |
| Second `NodeIdentify` on same connection | Application-level protocol violation by node | Close immediately | Log WARN: "node_identify: duplicate NodeIdentify on established connection" |
| Malformed `ChallengeResponse` frame | `hdr.PayloadLen != 68`, wrong discriminators | Close immediately | Log WARN: "node_identify: malformed ChallengeResponse: {reason}" |

**Connection lifecycle invariant**: After `onAccept` returns, the connection is either:
(a) fully bound (`Router.BindInterface` called, `sendMap` entry live, normal frame routing
begins), or (b) closed. There is no "unbound but open" state.

#### Eventual-consistency race path

`ADMISSION-SYNC-WIRE` pushes admission state from control to the router on every
`RegisterKey` write. The push is synchronous from control's perspective but not from
the node's perspective: if a node connects to the router before the push for its key
has been processed by the router, the router's `AdmittedKeySet` will not yet contain
the key, and `AdmitNode` returns `ErrNotAdmitted`.

This is not a new problem for NODE_IDENTIFY to "solve" — it is a property of the
eventual-consistency push model. The correct disposition is:

- The router closes the connection with `ErrNotAdmitted` (same as any other
  not-admitted path).
- The node can retry after a brief backoff. A well-provisioned deployment ensures
  the admission push completes before the node is directed to connect to the router;
  this is an operator/deployment concern, not a protocol concern.
- The eventual-consistency race is NOT a protocol defect — it is a documented
  transitional state. No special error code is needed beyond `ErrNotAdmitted`.

**This behavior feeds into Obligation 3 (above): the race is self-resolving via
retry; the router need not buffer or defer handshakes for eventually-pushed keys.**

#### New error codes for NODE_IDENTIFY wire path

The existing E-ADM-* codes (E-ADM-001, -003, -005, -008, -015) cover all the
admission-logic failure paths above. No new E-ADM-* codes are needed for those paths;
they are re-used exactly as in `ReAuthenticate`.

Two wire-specific failure paths require new error codes at the CMD layer:

| Code | Name | Trigger |
|---|---|---|
| E-ADM-022 | `node_identify: handshake timeout` | `conn.SetDeadline` fires during three-message exchange |
| E-ADM-023 | `node_identify: duplicate NodeIdentify` | Second `NodeIdentify` frame on an already-handshaken connection |

These are `cmd/switchboard`-scope constants (not `internal/admission` — they describe
wire-handler protocol violations, not admission-keyset semantics). Both are
WARN-level log messages; neither requires a new sentinel error var since the connection
is closed immediately after logging.

---

### 14. Obligations 5 and 6 — RESOLVED-BY-DELIVERY

#### Obligation 5: router AdmittedKeySet always-empty (RESOLVED by PR #126)

`S-BL.ADMISSION-SYNC-WIRE` (merged to develop @ 92a2c65, 2026-07-18) delivers:

- `wireAdmissionSyncHandlers` in `cmd/switchboard/admission_sync_wire.go` registers four
  router-side push RPC handlers (`internal.admission.register`,
  `internal.admission.revoke`, `internal.admission.expire`,
  `internal.admission.remove-svtn`). Each handler calls the corresponding
  `AdmittedKeySet` mutation method (`RegisterKey`, `RevokeKey`, etc.) on the router's
  own `ks`, then persists the snapshot via `routerPersister`.
- `admissionSyncClient` in `cmd/switchboard/admission_sync_client.go` implements the
  control-side push logic, dialing each `RouterManagementEndpoints` entry on each
  `admin.key.*` write.
- `admission_sync_snapshot.go` provides `writeSnapshotAtomic` and `loadSnapshot` for
  VLR-local durable admission state.

**Verification**: `admission.AdmitNode` called from the NODE_IDENTIFY handshake will no
longer return `ErrNotAdmitted` unconditionally — after a `RegisterKey` push is
processed by the router's `wireAdmissionSyncHandlers`, the key IS present in the
router's `AdmittedKeySet`, and `AdmitNode` will verify the challenge-response and
return nil on success.

**Residual**: as described in §13 (eventual-consistency race), a node that connects
immediately after `admin.key.register` on control — before the push to the router is
processed — will see `ErrNotAdmitted`. This is expected behavior of the push model,
not a defect.

#### Obligation 6: node has no stable admission keypair (RESOLVED by PR #125)

`S-BL.NODE-ADMISSION-PROVISIONING` (merged to develop @ ce06f6a, 2026-07-16) delivers:

- `config.Config.AdmissionKeyFile string` — new config field at `internal/config/config.go:166`.
- `loadOrGenerateAdmissionKeypair(stderr io.Writer, keyPath string) (ed25519.PrivateKey, error)` —
  at `cmd/switchboard/access.go:677`. Generates a new PKCS#8 Ed25519 keypair and writes
  it atomically to `keyPath` if absent; loads and parses it if present.
- `runAccess` at `cmd/switchboard/access.go:287` calls `loadOrGenerateAdmissionKeypair`
  (Phase d), extracts the public key, populates `discovery.Config.LocalNodeAdmissionPubkey`,
  and constructs the `discovery.Discovery` via `newDiscovery(discoveryCfg)`.
- `runAccessWithConnector` at `cmd/switchboard/access.go:465` calls `d.Run(runCtx)` in
  a goroutine when `disc` is non-nil — `Discovery.Run` is now wired into the access
  daemon lifecycle for the first time.

**Verification**: The access node process now holds a stable, restart-persistent
Ed25519 private key at the configured `admission_key_file` path. The `ChallengeResponse`
can be constructed as:
```go
resp := admission.ChallengeResponse{NonceSig: ed25519.Sign(admissionPrivKey, challenge.Nonce[:])}
```
where `admissionPrivKey` is the loaded/generated private key from `loadOrGenerateAdmissionKeypair`.
This is the production caller that §10 (v1.0) documented as absent. It now exists.

---

### 15. O-1 — AdmitNode expiry check (RESOLVED)

#### Verified split between AdmitNode and ReAuthenticate (disk-verified at develop@92a2c65)

`internal/admission/admission.go:457–525` (`AdmitNode`): checks `snap.revoked` (Step 1
read-lock snapshot), re-checks `liveEntry.revoked` under write lock (Step 3). Does NOT
check `expiry` at any point. The field `expiry time.Time` is present in `AdmittedKey`
(`admission.go:167–171`), but `AdmitNode` never reads it.

`internal/admission/reauth.go:172–240` (`ReAuthenticate`): checks `snap.expiry` at
line 196 (`if !snap.expiry.IsZero() && now.After(snap.expiry) { return ErrKeyExpired }`)
and re-checks `liveEntry.expiry` under write lock. Returns `ErrKeyExpired` (E-ADM-015)
on expiry.

This confirms the split: **initial admission (`AdmitNode`) does not enforce expiry;
re-authentication (`ReAuthenticate`) does.** An expired key can be admitted at connect
time even though it would be rejected at re-authentication.

#### Security assessment of the gap

A past-expiry key that was registered and had its `expire` push delivered to the router
before the node connected presents a real security concern: the operator has expressed
the intent that the credential is no longer valid after time T, but a node connecting
after T can still complete the `NODE_IDENTIFY` handshake successfully.

The severity is mitigated by three factors:
1. `ReAuthenticate` (called on IP-change events, `S-BL.REAUTH-WIRE`) WILL reject the
   key on the next re-auth attempt with `ErrKeyExpired`.
2. Key expiry is an operator action requiring explicit `admin.key.expire` execution —
   it is not automatic key rotation.
3. The window of vulnerability is the interval between expiry time T and the next
   re-auth event for that connection.

However, the initial-admission gap is a policy inconsistency: `AdmitNode` is the
"who may enter" gate, and an expired credential should not pass that gate regardless
of when it expires relative to the connection attempt.

#### RULING: NODE_IDENTIFY handshake MUST enforce key expiry at initial admission

**DECISION: The NODE_IDENTIFY handshake must reject expired keys at connect time.**

Implementation path: two viable options; one is chosen here.

**Option A: Add expiry check to `AdmitNode` (modifies `internal/admission`).**
`AdmitNode` gains an expiry check mirroring `ReAuthenticate`'s Step 2:
```go
// After snap.revoked check, before write-lock acquire:
if !snap.expiry.IsZero() && time.Now().UTC().After(snap.expiry) {
    return ErrKeyExpired
}
```
This is a pure, side-effect-free addition (no new lock, no I/O). The re-check under
write lock mirrors `ReAuthenticate`'s pattern.

**Option B: Call a new `AdmitNodeWithExpiryCheck` wrapper in the handshake handler.**
A new function in `cmd/switchboard/node_identify_wire.go` that checks expiry from the
`AdmittedKeySet` before calling `AdmitNode`. Requires an additional `AdmittedKeySet`
read (under RLock) separate from `AdmitNode`'s own read — introducing a TOCTOU window.

**RULING: Option A is the correct fix.** Expiry enforcement is a property of the
admission keyset, not of the wire handler. Placing it in `AdmitNode` is architecturally
correct: `AdmitNode` IS the "may this node be admitted?" gate, and it should answer
that question completely. The TOCTOU exposure in Option B is unnecessary.

**Scope note for story-writer**: This ruling adds ONE new task to `S-BL.NODE-IDENTIFY-WIRE`:
add the expiry check to `admission.AdmitNode` in `internal/admission/admission.go`,
mirroring `ReAuthenticate`'s expiry-check pattern. This is a change to `internal/admission`
— the story spec's original "introduces zero changes to `internal/admission`" assertion
is superseded by this ruling. The change is small (4 lines, mirrors existing code) and
does not alter the call signature of `AdmitNode`.

**New test obligation**: The story's test suite must include a test case:
`AdmitNode` returns `ErrKeyExpired` when `key.expiry` is set and `time.Now()` is after
the expiry. This test already has a natural analog in `reauth_test.go`'s expiry tests.

**BC implication**: BC-2.05.001 postcondition list should gain an entry documenting that
`AdmitNode` returns `ErrKeyExpired` (E-ADM-015) when expiry is set and past. The PO
must amend BC-2.05.001 accordingly (see Follow-on Actions table). This is a behavioral
contract change driven by the security ruling above.

**Human confirmation flag**: This ruling changes `internal/admission.AdmitNode` behavior
in a way that is observable to any existing caller of `AdmitNode` with a past-expiry key.
Current production callers of `AdmitNode` are: none (zero call sites outside tests until
`S-BL.NODE-IDENTIFY-WIRE` lands). Test callers: `internal/admission/*_test.go`.
Existing tests do NOT set expiry on keys used in `AdmitNode` test cases (confirmed by
`ReAuthenticate` being the only path that has expiry-related tests in `reauth_test.go`).
Therefore, no existing test will break from this change. However, the human should confirm
that the behavioral change to `AdmitNode` (initial admission now rejects expired keys)
is the intended policy, since it modifies a shared `internal/` primitive.

---

### 17. E-ADM-023 Connection-Teardown Mechanism (RESOLVED — v1.3)

#### Defect statement (disk-verified at feature/S-BL.NODE-IDENTIFY-WIRE @ 4179ee3)

`cmd/switchboard/mgmt_wire.go` `route()` `case 0x04` (lines 628–629):

```go
routerLogger.Log("node_identify: duplicate NodeIdentify on established connection (E-ADM-023)")
return errors.New("node_identify: duplicate NodeIdentify on established connection (E-ADM-023)")
```

The comment claims this triggers connection teardown via `netingress.ServeConn`'s error path.

`internal/netingress/netingress.go` `ServeConn` (lines 151–153, disk-verified):

```go
if err := route(hdr, payload); err != nil {
    // Drop-and-continue: routing already logs E-ADM-016/017 per BC-2.05.008.
    continue
}
```

`route()` errors are DROPPED. The connection is NOT closed. `ServeConn` returns only
on `io.EOF` / read-error / ctx-cancel — never on a non-nil `route()` return.

**Net**: BC-2.01.009 Invariant 7 ("second `NodeIdentify` on the same connection is a
hard error: the router closes the connection immediately") is NON-FUNCTIONAL. A duplicate
`NODE_IDENTIFY` is silently ignored; the connection stays open. AC-011 / Invariant 7 are
not enforced by the current stub.

#### Architectural premises (disk-verified)

| Premise | Evidence |
|---------|---------|
| `RouteFn` drop-and-continue is CORRECT and INTENTIONAL for E-ADM-016/017 | `netingress.go:67-72` RouteFn doc: "RouteFn returning a non-nil error is NOT a signal to close the connection…the ingress keeps reading. See BC-2.05.008 PC-4." Drop-and-continue is the explicit spec for the existing route-error class. |
| `route` closure is stateless — no `net.Conn` in scope | `mgmt_wire.go:541` closure signature: `func(hdr frame.OuterHeader, payload []byte) error`. The closure captures `router`, `routerLogger`, `sendMap`, etc., but NOT the per-conn `net.Conn` or `nc` — confirmed by reading the closure body (lines 566–640). Route is constructed once in `runRouter` and shared across all per-conn goroutines via `netingress.Serve`. |
| `nc.done` / `nc.doneOnce` signals the WRITER GOROUTINE to exit, NOT a conn-close | `mgmt_wire.go:388-394`: `doneOnce`/`done` are the single writer-goroutine exit signal. `conn.Close()` is owned by `Serve`'s per-conn goroutine via `defer func() { _ = c.Close() }()` at `netingress.go:269`. Closing `nc.done` stops the writer but does not close the TCP connection. |
| The ONLY path that closes the TCP conn from inside `ServeConn` is a read-error or ctx-cancel | `netingress.go:140-154`: `ReadFrame` returning non-EOF error → `return err`; ctx-cancel closes conn via `conn.Close()` in the ctx-watcher goroutine at line 130. There is no "close on route error" path. |
| There is no pre-existing selective per-conn teardown precedent in `runRouter` | §12 of this ruling (v1.1) noted: "No active connection-teardown precedent in runRouter." DRAIN broadcasts but does not selectively close connections. The prior `doneOnce`/`done` path is a writer-goroutine-exit signal, not a TCP close. |

#### Option evaluation

**Option A — Teardown-class `RouteFn` error (sentinel error type in `netingress`):**

Add a sentinel error type (e.g., `type ErrCloseConn struct{ Cause error }`) to
`internal/netingress`. `ServeConn` tests `errors.As(err, &ErrCloseConn{})` and
returns / closes on that class; all other errors continue. E-ADM-016/017 remain
drop-and-continue by not wrapping in `ErrCloseConn`.

Assessment:
- Requires modifying `internal/netingress` — a shared package (ARCH-08 position 18).
- Requires changing the `RouteFn` _effective_ contract (callers that return
  `ErrCloseConn` trigger close; callers that don't, continue). The _signature_ is
  unchanged but the _semantics_ are extended.
- `ServeConn` callers: `netingress.Serve` (line 289) and all callers of `Serve` —
  `mgmt_wire.go` is the only production caller. Blast radius is internal but real.
- E-ADM-016/017 preservation: yes, provably — they use `return errors.New(…)` not
  `return ErrCloseConn{…}`.
- Testability: clean — test passes `ErrCloseConn` from a mock route, asserts conn closes.
- Concern: introduces a semantic extension to the `RouteFn` contract that is invisible
  in the type signature. Future maintainers adding new route errors must know about the
  close-class sentinel or they get silent-ignore. The contract is now split-documented
  across `netingress.go` and wherever `ErrCloseConn` is defined.

**Option B — Per-conn close handle in the `case 0x04` path (no RouteFn change):**

The `route` closure in `runRouter` is currently stateless. For the E-ADM-023 path only,
close the connection by capturing the specific per-conn `net.Conn` directly in the part
of the route closure that handles `case 0x04`. This is structurally possible because
`route` is a closure over `runRouter`-local state — it can capture anything
`runRouter` has in scope. The mechanism: `runRouter` passes `conn net.Conn` into
the route closure for each connection (or the closure is constructed per-conn rather
than once globally), so `case 0x04` can call `conn.Close()` directly.

Assessment:
- `RouteFn` signature unchanged. `ServeConn` unchanged. `netingress` unchanged.
- Blast radius: ZERO for all `netingress` callers. The change is entirely within
  `runRouter`'s `route` closure construction.
- E-ADM-016/017 preservation: provably yes — they are in `case 0x01` (DRAIN) and
  `case 0x03` (DISCOVERY_RELAY), which do not call `conn.Close()`.
- Race safety: `conn.Close()` called from inside the per-conn goroutine (the same
  goroutine that owns `ServeConn`) is race-free with respect to that goroutine's
  read loop. `ServeConn`'s next `ReadFrame` call will return a read error on a
  closed conn, causing `ServeConn` to return naturally, which triggers the deferred
  `cleanup()` → `sendMap.Delete` → `nc.doneOnce.Do(close(nc.done))` chain.
  `doneOnce` ensures the writer goroutine sees exactly one close, regardless of whether
  the shutdown-flush pass fires concurrently. This is the standard conn-close ordering
  already proven safe by `netingress`'s ctx-cancel path (which also calls `conn.Close()`
  from a separate goroutine while `ServeConn` reads — the `go.md` rule 12 pattern).
- Implementation: `runRouter`'s `route` closure changes from being constructed once
  (shared across all conns) to being constructed once per connection inside `onAccept`
  — or equivalently, `case 0x04` is extracted to a per-conn helper that receives
  `conn net.Conn`.
- Testability: clean — integration test over `net.Pipe` can directly observe that
  the server-side conn closes after sending a second `NodeIdentify`.

**Option C — Detect duplicate in `onAccept`-adjacent state, not `route()`:**

`onAccept` runs before `ServeConn`. A second `NodeIdentify` frame arrives during
`ServeConn` (post-handshake). `onAccept` has already returned by the time the duplicate
arrives. There is no `onAccept`-adjacent hook that fires on subsequent frames.

**Ruling: ELIMINATED.** The duplicate arrives at `route()`, not at `onAccept`. There
is no hook point between `onAccept` and `route()` for post-handshake frames.

#### Decision: Option B

**RULING: The E-ADM-023 teardown mechanism is Option B — per-conn `net.Conn` close
handle injected into the `route` closure for `case 0x04`, without any change to the
`RouteFn` contract or `netingress.ServeConn`.**

Rationale summary:

1. **Zero blast radius.** Option A modifies a shared package (`internal/netingress`)
   and extends the effective `RouteFn` contract invisibly. Option B confines all changes
   to `runRouter`'s route closure construction — a single function in `cmd/switchboard`.
   No exported API changes; no shared-package changes.

2. **E-ADM-016/017 drop-and-continue is provably preserved.** The other `case` branches
   in `route()` do not receive `conn` and cannot call `conn.Close()`. The drop-and-continue
   behaviour for those paths is untouched structurally.

3. **Reuses established conn-close semantics.** `conn.Close()` from within the per-conn
   goroutine (or a goroutine that shares the conn) is the existing pattern in
   `netingress`'s ctx-cancel path. The shutdown ordering invariant (doneOnce) is already
   proven race-safe and requires no new mechanism.

4. **Option A's type-invisible contract extension is a future-maintainer hazard.**
   The `RouteFn` type signature gives no indication that some errors trigger close.
   Option B keeps the semantics of `RouteFn` exactly as documented: errors are
   drop-and-continue. The close for E-ADM-023 is explicit in the implementation,
   not hidden in a sentinel type.

#### Concrete implementation contract for the implementer

The implementer must make the `route` closure per-connection rather than shared.
In `runRouter`, the current `route` closure is constructed once outside `onAccept`
and passed to `netingress.Serve`. The change:

1. **Move route closure construction inside `onAccept`**, so each call to `onAccept`
   gets its own `route` closure that captures `conn net.Conn` (the specific connection
   for this `onAccept` invocation).

2. **Pass the per-conn `route` to `netingress.Serve` via `ServeConn`.** `netingress.Serve`
   currently calls `_ = ServeConn(ctx, c, route, logger)` with the shared `route`.
   After this change, `onAccept` must return the per-conn route to `Serve`. The cleanest
   way: `ServeConfig.OnAccept` receives `conn` and `h`; the per-conn `route` is
   constructed there and stored so that the per-conn goroutine's `ServeConn` call uses
   it. Since `ServeConn` is called in the same goroutine that ran `onAccept`, the
   per-conn route can be a closure variable captured in the per-conn goroutine's scope.

   Concretely, the pattern in `netingress.Serve` is:
   ```
   cleanup = cfg.OnAccept(c, h)      // per-conn setup; may also return a per-conn route
   _ = ServeConn(ctx, c, route, logger)
   ```
   One idiomatic way: add an optional `PerConnRoute` field to `ServeConfig`:
   ```go
   // PerConnRoute, if non-nil, is called after OnAccept to obtain the
   // route function for this specific connection. It receives the same
   // conn and NodeHandle as OnAccept. If nil, the shared route is used.
   // This is the mechanism for E-ADM-023 teardown — the per-conn route
   // captures conn and calls conn.Close() on a duplicate NodeIdentify.
   PerConnRoute func(conn net.Conn, h NodeHandle) RouteFn
   ```
   If `PerConnRoute` is nil, `Serve` falls back to the shared `route` (backward-
   compatible; all existing callers pass nil and see no behavior change).

   **This is the preferred approach** — it avoids changing the `RouteFn` contract
   while keeping `netingress.Serve` generic. The change to `netingress` is additive
   (new optional field, nil-safe fallback); the existing `RouteFn` doc comment is
   unchanged; blast radius is one new field with a nil guard.

   **Alternative if `PerConnRoute` is rejected:** construct the `route` closure entirely
   inside `onAccept` and wire it by having `Serve` accept a `RouteFactory` instead of a
   `RouteFn`. This is a larger API change and not recommended — the `PerConnRoute` field
   is the minimal-impact path.

3. **`case 0x04` in the per-conn route closure:**
   ```go
   case 0x04: // NODE_IDENTIFY — duplicate on established connection
       routerLogger.Log("node_identify: duplicate NodeIdentify on established connection (E-ADM-023)")
       _ = conn.Close() // triggers ServeConn read-error → return → cleanup()
       return errors.New("node_identify: E-ADM-023 duplicate NodeIdentify — connection closed")
   ```
   After `conn.Close()`, `ServeConn`'s next `ReadFrame` returns a read error, causing
   `ServeConn` to return, which runs `cleanup()` (`sendMap.Delete` +
   `nc.doneOnce.Do(close(nc.done))` + `router.UnbindInterface`). The `return` from
   `route()` is a belt-and-suspenders signal; `ServeConn` will drop it (drop-and-continue),
   but the conn is already closed so the loop terminates on the next read anyway.

4. **Teardown call sequence** (full, in order):
   - `conn.Close()` called from `case 0x04` inside per-conn goroutine's `route` closure
   - `ServeConn`'s `ReadFrame` returns a net error (use of closed network connection)
   - `ServeConn` returns (non-nil error, but `Serve` discards it at line 289)
   - Deferred `cleanup()` runs: `sendMap.Delete(h.IfaceID)`, `router.UnbindInterface(svtnID, nodeAddr, h.IfaceID)`, `nc.doneOnce.Do(func() { close(nc.done) })`
   - Deferred `func() { _ = c.Close() }()` runs (no-op, already closed)
   - `wg.Done()` fires; `sem` slot released

#### E-ADM-016/017 drop-and-continue preservation proof

E-ADM-016 and E-ADM-017 are logged and returned by `RouteFrame` inside
`internal/routing/routing.go`, called from `route`'s outer dispatch as:
```go
return routing.RouteFrame(hdr, payload, router)
```
This returns a non-nil error on malformed/unauthorized frames. Under Option B:

- The `case 0x04` branch is the ONLY branch that calls `conn.Close()`.
- The outer `routing.RouteFrame` call path (handling `FrameTypeData` frames for
  E-ADM-016/017) does NOT call `conn.Close()` — it returns a non-nil error that
  `ServeConn` drops per the existing drop-and-continue contract.
- Even if the per-conn `route` closure captures `conn`, it only uses it in `case 0x04`.
  The other branches are structurally unchanged.

**Therefore: Option B does not change the handling of any error path other than `case
0x04`. E-ADM-016/017 drop-and-continue is preserved by construction.**

#### Race-safety argument

`conn.Close()` is called from inside the per-conn goroutine — the same goroutine
executing `ServeConn`. This is NOT a concurrent close; it is a synchronous call from
within the read loop's dispatch (the `route()` call). After `conn.Close()`, the
goroutine returns from `route()`, `ServeConn` drops the error (continue), then
attempts `ReadFrame` on the now-closed conn, which returns a net error causing
`ServeConn` to `return err`. This is the standard "close from within the read goroutine"
pattern — no concurrent write goroutine is racing to close the same conn.

The writer goroutine (`nc.send` / `nc.done` loop) does NOT call `conn.Close()`. It
exits when `nc.done` is closed (via `doneOnce`). `doneOnce` ensures that regardless
of whether the teardown fires from (a) the E-ADM-023 `conn.Close()` → cleanup path,
or (b) the router-wide shutdown flush pass at `mgmt_wire.go:967`, only one `close(nc.done)`
executes. The second caller's `doneOnce.Do(…)` is a no-op. This invariant is already
proven by the existing DRAIN-wire shutdown tests; Option B adds no new close sites.

The `defer func() { _ = c.Close() }()` in `Serve`'s per-conn goroutine (line 269) calls
`conn.Close()` again after `ServeConn` returns — this is a no-op on an already-closed
conn (idiomatic Go, returns `net.ErrClosed`).

#### ServeConn caller blast radius

`grep -rn "ServeConn\b"` in the worktree shows `ServeConn` is called in exactly two
production sites:
1. `internal/netingress/netingress.go:289` — inside `Serve`'s per-conn goroutine
2. Directly in `internal/netingress/netingress_test.go` and `fuzz_test.go` tests

Neither production caller changes under Option B (if the `PerConnRoute` field approach
is used). `netingress.Serve` gains one new optional field (`PerConnRoute`) with a
nil-safe fallback — zero behavior change for callers that don't set it. All existing
tests continue to pass without modification.

**ServeConn blast radius: ZERO production callers broken.**

If the `PerConnRoute` field is rejected in favor of a `RouteFactory` approach, the
`Serve` signature changes — that would require updating `mgmt_wire.go`'s `Serve` call
site. That is still a blast radius of ONE known caller. The `PerConnRoute` field is
recommended precisely because it avoids this.

#### BC-2.01.009 Invariant 7 mechanism note

Invariant 7 currently reads (disk-verified):

> **Second `NodeIdentify` on the same connection is a hard error:** If an already-admitted
> connection sends a second `NODE_IDENTIFY` frame, the router closes the connection and
> logs E-ADM-023.

This says "closes the connection" without specifying HOW. The PO should amend BC-2.01.009
Invariant 7 by appending the following mechanism sentence:

> Implementation: `conn.Close()` is called directly in the per-conn `route` closure for
> `case 0x04`, via the per-conn close handle injected by `runRouter` (§17 of the
> `S-BL.NODE-IDENTIFY-WIRE-rulings.md`). `ServeConn` does not need to be modified —
> the TCP close causes the next `ReadFrame` to return an error, which terminates
> `ServeConn` naturally and triggers the cleanup func.

**PO action**: BC-2.01.009 v1.1 → v1.2. Append mechanism sentence to Invariant 7.
No PC renumbering. No AC citation changes (AC-011 cites Invariant 7 by number, not by
its text body — the append does not invalidate any citation). POL-001 changelog required.

---

## Follow-on Actions (v1.3 — updated)

| Action | Owner | Status | Notes |
|--------|-------|--------|-------|
| Add `NODE_IDENTIFY = 0x04` row to BC-2.01.008 Postcondition 2 registry table | product-owner | **DONE** (v1.0 ruling; Obligation 1 resolved) | `NODE_IDENTIFY = 0x04` registered per BC-2.01.008 v1.3 |
| Add `rulings_doc` reference to story frontmatter | story-writer | **DONE** (story v1.3) | `rulings_doc:` field present |
| Resolve Obligation 3 (re-identify / rebind semantics) | architect | **DONE** (§12, this ruling) | LWW overwrite; prior connection self-removing; second NodeIdentify on same conn = hard error |
| Resolve Obligation 4 (handshake timeout + failure paths) | architect | **DONE** (§13, this ruling) | `nodeIdentifyHandshakeTimeout = 10 * time.Second`; failure path table |
| Mark Obligations 5 and 6 resolved-by-delivery | architect | **DONE** (§14, this ruling) | PR #126 (admission-sync) + PR #125 (node-keypair) |
| Resolve O-1 (AdmitNode expiry gap) | architect | **DONE** (§15, this ruling) | AdmitNode must gain expiry check; BC-2.05.001 amendment required |
| Amend BC-2.05.001 to add `ErrKeyExpired` postcondition for `AdmitNode` | product-owner | **OPEN** | O-1 ruling (§15) requires this. New postcondition: `AdmitNode` returns `ErrKeyExpired` (E-ADM-015) when key expiry is set and past. |
| Story-writer: decompose `S-BL.NODE-IDENTIFY-WIRE` into ACs | story-writer | **OPEN — story-writer gate** | All obligations resolved; decomposition unblocked. Scope includes: wire codec + handshake handler (§§2–9); `Router.BindInterface/LookupInterface/UnbindInterface` (§8); rebind/reconnect semantics (§12); timeout + failure paths (§13); `AdmitNode` expiry check (§15); E-ADM-023 teardown wiring (§17). |
| PO: verify BC-2.01.008 Postcondition 2 has `NODE_IDENTIFY = 0x04` row (if not already present in current version) | product-owner | **OPEN — verify** | Obligation 1 was marked RESOLVED per story v1.3 changelog; confirm BC-2.01.008 current version reflects this |
| Resolve E-ADM-023 teardown mechanism defect | architect | **DONE** (§17, this ruling) | Option B: per-conn `conn net.Conn` handle injected into the duplicate-NodeIdentify detection path; `conn.Close()` called directly; no RouteFn contract change |
| Story-writer: update Task 19 wording and add Task 20 wiring subtask for E-ADM-023 teardown | story-writer | **OPEN** | §17 specifies exact Task 19 wording correction and new Task 20 subtask. story input-hash must be refreshed after story update. |
| Implementer: build E-ADM-023 teardown wiring per §17 ruling | implementer | **OPEN** | story-writer cascade must complete first; then implementer builds the per-conn close handle injection into `runRouter`'s route closure for case 0x04 |
| Test-writer: write AC-011 test for E-ADM-023 → connection close | test-writer | **OPEN** | §17 specifies: integration test over `net.Pipe`, admits connection, sends second `NodeIdentify` frame, asserts conn closes within deadline, asserts no data frames are routed after the duplicate |
| BC-2.01.009 Invariant 7 mechanism note | architect | **DONE** (§17, this ruling) | Invariant 7 currently says "closes the connection immediately" without HOW. §17 adds the mechanism note: "via `conn.Close()` called directly in the route closure for `case 0x04`, through the per-conn close handle injected by `runRouter`." No PC renumbering — mechanism note appended to Invariant 7 body in BC-2.01.009 v1.1 → v1.2. |
| Update BC-2.01.009 Invariant 7 with mechanism note | product-owner | **OPEN** | PO adds one sentence to Invariant 7 body per §17 specification. Version bump v1.1 → v1.2 with POL-001 changelog entry. No PC or AC number changes. |

### Named downstream BC work (for product-owner)

The following new or amended behavioral contracts are implied by this ruling:

| ID | Type | One-line description | Source ruling |
|---|---|---|---|
| BC-2.05.001 (amend) | Amendment | Add `ErrKeyExpired` (E-ADM-015) return postcondition to `AdmitNode`; expiry is now enforced at initial admission | §15 (O-1 ruling) |
| BC-2.01.008 (verify) | Verify existing | Confirm `NODE_IDENTIFY = 0x04` row is in PC-2 registry; if absent, PO adds it | Obligation 1 (v1.0) |
| New BC for NODE_IDENTIFY wire handshake | New | Postconditions for the three-message `NodeIdentify → Challenge → ChallengeResponse` exchange, including all failure paths enumerated in §13; binding recorded on success | §§7, 12, 13 |
| New BC for BindInterface binding semantics | New | (SVTNID, NodeAddr) → IfaceID binding lifecycle: created on `AdmitNode` success, LWW overwrite on reconnect, removed on connection close via cleanup func | §12 |

**PO must author BC bodies; architect ruling provides the decision content only.**

### Story-writer scope summary (for AC decomposition)

The story-writer receives all six obligations resolved. The story's ACs should cover:

1. Successful handshake: admitted key, valid signature → binding recorded in
   `Router.identityIfaceMap`; `LookupInterface(svtnID, nodeAddr)` returns the
   correct `IfaceID`; `ServeConn` begins after handshake.
2. Revoked key → `AdmitNode` returns `ErrKeyRevoked`; connection closed.
3. Not-admitted key → `AdmitNode` returns `ErrNotAdmitted`; connection closed.
4. Nonce replay → `AdmitNode` returns `ErrNonceReplay`; connection closed.
5. Signature verification failure → `AdmitNode` returns `ErrSignatureVerificationFailed`;
   connection closed.
6. Expired key → `AdmitNode` returns `ErrKeyExpired` (§15 ruling); connection closed.
7. Handshake timeout (10s) → connection closed.
8. Reconnect (new TCP, same identity) → `BindInterface` overwrites prior binding (LWW).
   Prior connection's cleanup func will call `UnbindInterface` when it eventually closes.
9. Second `NodeIdentify` on same connection → connection closed immediately.
10. `UnbindInterface` called from cleanup func on connection close → binding removed from
    `identityIfaceMap`.
11. `AdmitNode` expiry check added to `internal/admission` — new unit test in
    `internal/admission/*_test.go` for expired-key path.

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
| UnbindInterface signature | `UnbindInterface(svtnID [16]byte, nodeAddr [8]byte, callerIfaceID InterfaceID)` — 3-arg; callerIfaceID required for stale-cleanup guard (PC-9) |
| Concurrent writes gated by | `r.mu` (existing Router mutex) |
| Rebind semantics (Obligation 3) | LWW overwrite on reconnect; no active prior-conn teardown; second NodeIdentify on same conn = hard error + close |
| Handshake timeout (Obligation 4) | `const nodeIdentifyHandshakeTimeout = 10 * time.Second` (matches admission_sync_client.go precedent); `conn.SetDeadline` before first read, cleared on success |
| Failure paths (Obligation 4) | All E-ADM-* error returns → close immediately; eventual-consistency race = ErrNotAdmitted (retry); new wire-layer codes E-ADM-022 (timeout) and E-ADM-023 (duplicate NodeIdentify) |
| Obligations 5/6 status | RESOLVED-BY-DELIVERY — PR #126 (admission-sync, router keyset populated) + PR #125 (node keypair provisioned) |
| O-1 expiry ruling | AdmitNode MUST gain expiry check (Option A) — mirrors ReAuthenticate pattern; BC-2.05.001 amendment required |
| Obligation status | All six obligations + O-1 resolved. Story is decomposition-ready pending BC-2.05.001 amendment by PO. |
| Human confirmation flag | O-1 ruling changes AdmitNode behavior (was: no expiry check; will be: ErrKeyExpired on past-expiry key). Zero existing production callers affected. PO/human should confirm this is the intended policy before story-writer decomposes ACs. |
| RouterSig scope note | Pre-existing property of admission package — nonce-only signing in GenerateChallenge; not a new concern for this wire story; documented in §11 for future audit transparency |
| E-ADM-023 teardown mechanism (§17) | Option B — per-conn `net.Conn` handle injected into the `route` closure for `case 0x04`; `conn.Close()` called directly; no `RouteFn` contract change; E-ADM-016/017 drop-and-continue preserved |
| RouteFn contract change | NONE — `RouteFn` signature and `ServeConn` drop-and-continue contract are unchanged |
| ServeConn caller blast radius | Zero — no change to `netingress.ServeConn` or `netingress.Serve` |
| Human flag (§17) | Option B requires `runRouter` to pass `conn net.Conn` into the `route` closure. This is an internal `runRouter` refactor (no exported API change). No human sign-off required — blast radius confirmed zero. |
