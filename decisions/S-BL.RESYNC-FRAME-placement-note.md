---
artifact_id: S-BL.RESYNC-FRAME-placement-note
document_type: architect-design-note
story_id: S-BL.RESYNC-FRAME
title: "RESYNC control-frame protocol design elaboration — ADR-005 second half, reconciled with the ctl(0x03)/control_type wire model"
status: draft
producer: architect
timestamp: 2026-08-30T00:00:00Z
version: "1.0"
bc_traces:
  - BC-2.01.002   # superseded trace — see Q1; ChannelFrame.FrameType discriminator is NOT where RESYNC lives
  - BC-2.01.004   # outer-header frame_type enum (6/6 slots assigned); ctl(0x03) payload note already names control_type=0x02 RESYNC as reserved
  - BC-2.01.005   # channel-header opacity + router-terminated-ctl carve-out (PC-2); VP-015 preservation
  - BC-2.01.008   # control_type discriminator schema home; RESYNC=0x02 already reserved (PC-2, Inv-4); FO-DRAIN-WIRE-001 (PC-4)
  - BC-2.01.010   # (svtnID, nodeAddr) -> InterfaceID identity binding, consulted for router-side relay resolution
  - BC-2.02.005   # downstream ARQ send/receive buffer semantics this story's replay behavior extends
vp_traces: []   # no VPs minted by this note — see "VP needs" section; next free id is VP-081+
architecture_modules:
  - cmd/switchboard        # dispatch site (mgmt_wire.go buildRoute), new resync_wire.go (assemble/parse/relay), access-mode dial gap
  - internal/frame          # FrameType enum is NOT extended (Q1) — no change here
  - internal/outerassembler # NOT used for RESYNC framing (Q2) — ctl frames are hand-assembled, same as DRAIN/DISCOVERY_RELAY/NODE_IDENTIFY
  - internal/arq             # candidate new primitive: InFlightSeqsFrom(minSeq) (Q7)
  - internal/arqsend        # Retransmitter.Retransmit is the replay primitive, invoked in a loop (Q7)
  - internal/routing         # LookupInterface(svtnID, nodeAddr) is the relay-target-resolution primitive (Q5) — already shipped
  - internal/netingress      # accept-only; NOT the reconnect state machine (Q3 corrects the stub's architecture_modules claim)
related_documents:
  - .factory/stories/S-BL.RESYNC-FRAME.md
  - .factory/specs/architecture/ARCH-03-routing-engine.md
  - .factory/specs/behavioral-contracts/ss-01/BC-2.01.002.md
  - .factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md
  - .factory/specs/behavioral-contracts/ss-01/BC-2.01.005.md
  - .factory/specs/behavioral-contracts/ss-01/BC-2.01.008.md
  - .factory/code-delivery/S-7.04-FU-DRAIN-WIRE/pr-description.md
  - .factory/decisions/S-7.04-FU-DRAIN-WIRE-placement-note.md
  - .factory/decisions/S-BL.DISCOVERY-WIRE-fanout-options.md
  - .factory/decisions/RULING-W6TB-C-console-transport.md
  - internal/frame/frame.go
  - internal/outerassembler/assemble.go
  - internal/arqsend/arqsend.go
  - internal/arq/arq.go
  - internal/routing/identity.go
  - internal/netingress/netingress.go
  - cmd/switchboard/mgmt_wire.go
  - cmd/switchboard/discovery_relay_wire.go
  - cmd/switchboard/node_identify_wire.go
  - cmd/switchboard/access.go
  - internal/testenv/loopback.go
  - internal/testenv/testenv.go
---

## Changelog

| Version | Change |
|---------|--------|
| 1.0 | Initial design elaboration. Resolves the central frame-model reconciliation (ctl/control_type, not a new top-level FrameType), locates the FO-DRAIN-WIRE-001 discharge site, and surfaces a load-bearing prerequisite-topology finding (access-mode has no data-plane network connection in production code today) that the stub's `internal/netingress`-centric scoping did not anticipate. Produced against ARCH-03 v1.8, BC-2.01.002 v1.4, BC-2.01.004 v1.5, BC-2.01.005 v1.2, BC-2.01.008 v1.3, S-BL.RESYNC-FRAME.md v0.1-backlog-stub. |

---

## 0. Summary (TL;DR)

1. **The stub's AC-001 is superseded.** RESYNC is **not** a new `internal/frame.FrameType` constant. The outer-header `frame_type` enum has exactly six canonical values (`0x01`–`0x06`), all already assigned, and RESYNC has been carrying a *reserved* slot in the `ctl (0x03)` frame's `control_type` payload discriminator (`control_type = 0x02`) since **BC-2.01.008 v1.0** (2026-07-11) — before this stub was even written. RESYNC is a `ctl(0x03)` frame with `control_type = 0x02`, following the exact wire shape DRAIN (`control_type=0x01`) and DISCOVERY_RELAY (`control_type=0x03`) already ship in production.
2. **FO-DRAIN-WIRE-001's discharge site is identified and is a small, well-precedented change**: `cmd/switchboard/mgmt_wire.go`'s `buildRoute` closure gains a `case 0x02:` arm (currently `control_type=0x02` falls into the `default:` silent-ignore arm). VP-015 (routing payload-independence) is preserved by construction — ctl frames never reach `routing.RouteFrame`/`SVTNRoute` at all; RESYNC dispatch happens entirely inside the netingress `route` closure, before the routing fallthrough, exactly like DRAIN/DISCOVERY_RELAY/NODE_IDENTIFY.
3. **A load-bearing topology finding, not anticipated by the stub**: `internal/arq`'s downstream `SendBuffer` lives at the **access node**, per ARCH-03's own "Downstream ARQ" section — but `cmd/switchboard`'s access-mode daemon (`runAccess`/`runAccessWithConnector` in `access.go`) has **zero data-plane network code today** (no `net.Dial`, no `outerassembler.Assemble`, no `netingress`/wire-frame read). The only production data-plane dial-out is `internal/upstreamdial.Connector`, used exclusively for router-to-router PE-mode uplinks. This means the wire leg RESYNC's *receiver* (the access node, which must run `arqsend.Retransmitter`) would arrive on **does not exist yet**. This is a genuine prerequisite gap, structurally identical in kind to (and larger in scope than) the "node-identity-to-connection binding" gap DISCOVERY_RELAY's Ruling 3(f) flagged and which has since been resolved (`routing.LookupInterface`, shipped by S-BL.NODE-IDENTIFY-WIRE).
4. **Recommended scope split**: this story can and should deliver the wire-format layer (assemble/parse RESYNC frames), the router-side dispatch-and-relay logic (using the now-available `routing.LookupInterface` primitive), and the emitter/replay-trigger *logic* as isolated, unit-testable primitives — mirroring exactly how DRAIN, DISCOVERY_RELAY, and NODE_IDENTIFY were each built and merged. The stub's **AC-005** ("two-daemon in-process stack... assert RESYNC fires and session data is recovered... with no content loss") is **not achievable as written** today, because there is no live access-node-to-router wire connection to exercise. AC-005 needs rescoping by story-writer (see §8, §9).
5. **A BC-2.01.008 (or, preferably, a new sibling BC) amendment is required** to define RESYNC's payload fields beyond the shared 4-byte control header — BC-2.01.008 v1.3 currently states the DRAIN/RESYNC message *is* the 4-byte header, full stop, which has no room for a sequence number. This is flagged for product-owner, not made here (§6).

---

## 1. Q1 — Frame-model reconciliation (the crux)

### 1.1 The stub's model vs. the current wire model

The stub (`S-BL.RESYNC-FRAME.md` v0.1-backlog-stub, written 2026-07-06) sketches:

> **AC-001 (BC-2.01.002 / internal/frame):** A new `FrameType` constant `RESYNC` is added to `internal/frame`, parallel to `EMPTY_TICK` and `DATA`. The outerassembler emits it via `Assemble(ChannelFrame{Type: frame.RESYNC, ...}, ...)`.

This predates the `ctl`/`control_type` control-frame model that BC-2.01.008 established (also 2026-07-11, five days after the stub) and that DRAIN (S-7.04-FU-DRAIN-WIRE), DISCOVERY_RELAY (S-BL.DISCOVERY-WIRE), and NODE_IDENTIFY (S-BL.NODE-IDENTIFY-WIRE) have since shipped against. **AC-001 is superseded.**

### 1.2 Why RESYNC cannot be a new top-level FrameType — three independent, converging citations

**(a) The wire enum is structurally exhausted.** `internal/frame/frame.go:33-46`:

```go
const (
    FrameTypeData      FrameType = 0x01
    FrameTypeEmptyTick FrameType = 0x02
    FrameTypeCtl       FrameType = 0x03 // payload carries a control_type discriminator byte (BC-2.01.008 schema home)
    FrameTypeArq       FrameType = 0x04
    FrameTypeFec       FrameType = 0x05
    FrameTypePEConnect FrameType = 0x06 // (ARCH-02 §3.1)
)

func (f FrameType) Valid() bool {
    return f >= FrameTypeData && f <= FrameTypePEConnect
}
```

Six canonical values, all six assigned. `ParseOuterHeader` rejects anything outside `0x01`–`0x06` with `ErrInvalidFrameType`. `BC-2.01.004` v1.5 PC-2's outer-header layout table lists the identical six-value enum and explicitly annotates the `ctl (0x03)` row: *"payload carries a leading `control_type` discriminator byte (0x01=DRAIN, 0x02=RESYNC reserved) — opcode schema home: BC-2.01.008"*. There is no seventh slot to give RESYNC.

**(b) The architecture record already made this exact ruling, in writing, for the sibling opcode DISCOVERY_RELAY** — and named RESYNC as the same-shape precedent. ARCH-03 v1.8 changelog (2026-07-13, adjudication S-BL.DISCOVERY-WIRE-rulings v1.3 Ruling 3):

> "the router relays validated advertisements via the existing `FrameTypeCtl (0x03)` outer frame with a new `control_type=0x03` (DISCOVERY_RELAY) discriminator (BC-2.01.008 is the schema home, gains a new registry row), **not a new outer FrameType (the 6-slot canonical enum is exhausted at FrameTypePEConnect=0x06)**."

**(c) RESYNC's `control_type = 0x02` slot has been reserved in the schema-home BC since before this stub existed.** `BC-2.01.008` v1.0 (2026-07-11) Postcondition 2's registry table:

| control_type | Value | Defined by | Description |
|-------------|-------|------------|-------------|
| DRAIN | 0x01 | S-7.04-FU-DRAIN-WIRE | ... |
| **RESYNC** | **0x02** | **S-BL.RESYNC-FRAME (reserved, not yet dispatched)** | **Session resynchronization signal** |
| DISCOVERY_RELAY | 0x03 | S-BL.DISCOVERY-WIRE | ... |
| NODE_IDENTIFY | 0x04 | S-BL.NODE-IDENTIFY-WIRE | ... |

...and Invariant 4: *"control_type=0x02 (RESYNC) is reserved but not dispatched until S-BL.RESYNC-FRAME lands. Any receiver encountering 0x02 before that story MUST apply the silent-ignore rule (Postcondition 4)."* This BC already names this story as RESYNC's implementor.

### 1.3 Ruling

**RESYNC = `ctl (frame_type=0x03)` outer frame, `control_type = 0x02` payload discriminator.** Confidence: definitive — three independent artifacts (the wire-format enum's own `Valid()` bound, an explicit prior architecture ruling for the structurally identical DISCOVERY_RELAY case, and BC-2.01.008's own pre-existing reservation) converge on the same answer with no dissenting evidence anywhere in the current spec set.

**Consequences for the stub:**
- **AC-001 is superseded in full.** No `internal/frame.FrameType` constant is added. `internal/frame/frame.go` is untouched by this story.
- The stub's `bc_traces: [BC-2.01.002, BC-2.01.004]` needs revision at story-writer time: **BC-2.01.002 is not a real trace for RESYNC** — BC-2.01.002 is specifically about `ChannelFrame.FrameType` / `EMPTY_TICK` liveness semantics (see its title: "Empty-Tick Frame Is a Valid Liveness Signal") and never mentions RESYNC anywhere in its text. The correct traces are BC-2.01.004 (outer-header `ctl` row, already documents RESYNC's reservation) and **BC-2.01.008** (the actual schema home — missing from the stub's `bc_traces` entirely, which is itself evidence the stub predates BC-2.01.008's existence).
- `outerassembler.Assemble` is **not used** to build RESYNC frames, for the same reason it is not used for DRAIN, DISCOVERY_RELAY, or NODE_IDENTIFY: `Assemble` unconditionally composes a `ChannelHeader` (12/20 bytes) after the outer header (`internal/outerassembler/assemble.go:78-86`), which is the *session-data* framing, not the *control-frame* framing. Every existing `ctl(0x03)` emitter (`mgmt_wire.go:843-849` for DRAIN, `discovery_relay_wire.go:78-114` for DISCOVERY_RELAY, `node_identify_wire.go:100-145` for NODE_IDENTIFY) hand-assembles `frame.EncodeOuterHeader(...)` directly followed by the raw control payload bytes — no channel header, no HMAC (`HMACTag` left zero; see §1.4). RESYNC's emitter must follow this identical shape, not `outerassembler.Assemble`.

### 1.4 HMAC posture (inherited, not re-litigated)

Every shipped `ctl(0x03)` frame leaves `HMACTag` at its zero value — the trust boundary is the already-admitted TCP connection (post-NODE_IDENTIFY), not a per-frame HMAC (`discovery_relay_wire.go:99-102`, matching `mgmt_wire.go`'s DRAIN precedent exactly). This is not a RESYNC-specific decision to make; it is the established posture for every router-terminated ctl opcode. The one live caveat, on record since the DRAIN PR's own security review (`pr-description.md` "MEDIUM — Ctl-frame path bypasses per-frame HMAC authentication" and the FO-DRAIN-WIRE-001 forward obligation cited there), is that **this posture was explicitly named as needing re-adjudication when RESYNC lands**, because RESYNC (unlike DRAIN/DISCOVERY_RELAY, which are router-originated or router-authenticated-relay) triggers **state-changing retransmission behavior at another node** in response to a frame the router itself did not originate. §9 (Open Questions) surfaces this explicitly rather than silently inheriting the DRAIN posture.

---

## 2. Q2 — Terminal-consumer / VP-015 preservation

BC-2.01.008 Invariant 2 (v1.1) establishes, as a fact about the **current architecture** (not merely a design choice): every `ctl(0x03)` frame arriving at a router via `internal/netingress` (the node-facing accept loop) is **terminal-consumer by construction** — it can never be "in transit" to another node, because (a) the outer header carries no router-identity/addressing field, (b) `internal/routing.SVTNRoute` performs forwarding-table *validation* only and does not relay frame bytes to any connection (verified directly in code — see §5.1), and (c) `netingress` shares no code path with any router-to-router uplink.

This is confirmed empirically in the live dispatch code. `cmd/switchboard/mgmt_wire.go:603-658`'s `buildRoute` closure:

```go
buildRoute := func(conn net.Conn) netingress.RouteFn {
    return func(hdr frame.OuterHeader, payload []byte) error {
        if hdr.FrameType == frame.FrameTypeCtl {
            if len(payload) < 4 { /* E-PRT-002, drop */ return nil }
            controlType := payload[0]
            switch controlType {
            case 0x01: // DRAIN
                ...
            case 0x04: // NODE_IDENTIFY duplicate
                ...
            default:
                // includes 0x02 RESYNC today — silent-ignore, FO-DRAIN-WIRE-001
            }
            return nil   // <-- ctl frames NEVER fall through to routing.RouteFrame
        }
        return routing.RouteFrame(hdr, payload, router)
    }
}
```

Every `ctl` frame terminates inside this closure and **never reaches `routing.RouteFrame`/`SVTNRoute`**, regardless of `control_type`. This is exactly the shape BC-2.01.005 PC-2's carve-out note requires for VP-015 preservation: *"Control payload parsing occurs in `cmd/switchboard` post-routing, outside the `SVTNRoute` call graph."*

**Discharge path for FO-DRAIN-WIRE-001**: add a `case 0x02:` arm to this same `switch` (between the existing `case 0x01:` and `case 0x04:` arms, matching the opcode-registry's numeric ordering). RESYNC's terminal-consumer parsing happens **inside this closure**, at the router, exactly where DRAIN/NODE_IDENTIFY parsing already happens — never inside `SVTNRoute`. VP-015 is preserved by the same structural argument that already covers DRAIN, DISCOVERY_RELAY, and NODE_IDENTIFY: this dispatch point is architecturally upstream of, and disjoint from, the routing call graph.

---

## 3. Q3 — Topology finding: the access-mode data-plane gap (not anticipated by the stub)

This is the most consequential finding of this design pass, and it changes the story's real scope more than the frame-model question does.

### 3.1 Where ARQ state actually lives

ARCH-03 §"Downstream ARQ (internal/arq, BC-2.02.005)" is explicit about which endpoint holds which state:

```
Sender (access node):
  SendBuffer: sliding window of unacknowledged frames
Receiver (console):
  RecvBuffer: reorder buffer, delivers in sequence
```

`arqsend.Retransmitter` (the primitive `S-BL.ARQ-TX` delivered, and the one this story's Non-Goals/deps section names as the replay mechanism) wraps `*arq.ARQ` — a handle to exactly this `SendBuffer` state. Per ADR-005's own text: *"the receiver sends a RESYNC control frame requesting the sender to retransmit... The sender replays from last_acked_seq+1."* The **receiver** is the console (downstream ARQ receiver, per ARCH-03); the **sender** — the party that must run `arqsend.Retransmitter` in response to RESYNC — is the **access node**.

### 3.2 The access-mode daemon has no data-plane network connection today

`cmd/switchboard/access.go`'s `runAccess`/`runAccessWithConnector` (the access-mode CLI entry point) construct `session.AccessNode`, a downstream `halfchannel.HalfChannel`, and a *local, not-in-the-live-data-path* `routing.Router` instance (its own doc comment: *"the router is constructed-but-not-in-live-data-path... retained so [tests] can call RouteFrame on the daemon's own instance"*). A repository-wide grep of `cmd/switchboard/*.go` production code for `net.Dial(...)` / `net.Listen(...)` finds exactly:

- `mgmt_wire.go:565` — `runRouter`'s **router-mode** data-plane `net.Listen` (accepts nodes).
- `mgmt_wire.go` / various — management-plane Unix-socket listeners (local admin, not session data).
- `discovery_wire.go:108` — router-mode multicast listener (discovery, not session data).
- `admission_sync_client.go:161` — control-plane admission sync (unrelated to session data).
- `internal/upstreamdial.Connector`, used **only** by `runRouter` for router-to-router PE-mode uplinks (`mgmt_wire.go:901-909`).

**There is no `net.Dial` anywhere for an access-mode or console-mode data-plane connection to a router.** `internal/testenv` corroborates this from the test-harness side: `testenv/loopback.go`'s entire tick-driven stack (`onUpstreamTick`/`onDownstreamTick`/`deliverUpstream`/`deliverDownstream`) wires `halfchannel` → `multipath` → `arq` → `session.AccessNode` **entirely in-process**, with zero `frame.OuterHeader`, zero `outerassembler.Assemble`, zero TCP — by design (its own doc comment: extends "a same-goroutine `DeliverFrame` shortcut" into "a tick-driven, protocol-accurate loopback stack," but still bypassing the wire entirely). `testenv.go`'s `Env`/`NewWithRouters` harness (the more elaborate "real router" test environment) is the same: `Env.SendKeystroke` calls `sh.access.DeliverFrame(hdr)` directly — an in-process function call, not a wire write — and its `PERouterAddr` fixture listener is a bare accept-and-drain stub that never parses a frame.

**Conclusion: no production or test-harness code path in this repository today performs real wire-level access-node ⟷ router session-data relay.** The only wire-level session-data path that exists is router ⟷ router (PE mode, `upstreamdial.Connector` on one side, `netingress.Serve` + the "PE receive path" `FrameArrivalHandler`/`SplitHorizon` on the other), and that path is explicitly scoped to multi-router topology, not access-node-to-router.

This is fully consistent with — and gives a concrete mechanism for — ADR-005's own framing: *"Open question for KoS: This sketches the PE-phase design. The exact RESYNC frame format and state machine are deferred to PE implementation. E router has a single path; failover only occurs on manual restart."* ADR-005 was written knowing this leg was not yet built.

### 3.3 What this means for the stub's story scope

- The stub's `architecture_modules: [internal/outerassembler, internal/arq, internal/netingress, internal/frame]` names `internal/netingress` as owning "the reconnect state machine." This is not correct against the current codebase: `internal/netingress` is documented and implemented as **accept-only** (`netingress.go:1-13`: *"a TCP listener that accepts node connections... No routing decisions are made here"*) — it has no dial, no reconnect, no backoff concept at all. The only reconnect/backoff state machine that exists in production is `internal/upstreamdial.Connector` (router-to-router), which is architecturally the closest analog to what an access-node-side connector would need, but is not itself reusable as-is (it is wired for the PE-router-hierarchy role, not the access-node role, and its `Envelope` bootstrap is a stub — see its own doc comment: *"Envelope carries zero node-identity fields: the full bootstrap... is deferred to the session-bootstrap story"*).
- **An access-node-side network connector (dial + reconnect/backoff + NODE_IDENTIFY handshake + frame read/write loop) does not exist and is a genuine prerequisite** for any live end-to-end RESYNC round trip. Building it is a substantial piece of work in its own right, architecturally symmetric to `internal/upstreamdial.Connector` but is **out of this story's stated Non-Goals** ("Does not implement the PE outbound dial loop. That is S-7.04-FU-PE-CONNECTOR") and was not identified by the stub as a dependency at all.
- This is not a reason to block the wire-format and dispatch-logic work (§8 Layer 1 is fully buildable and testable today, in isolation, the same way DRAIN/DISCOVERY_RELAY/NODE_IDENTIFY's payload-assembly and dispatch-arm code was built and merged before their full end-to-end paths existed). It **is** a reason the stub's **AC-005** ("two-daemon in-process stack... session data is recovered... with no content loss") cannot be delivered as literally written by this story alone.

---

## 4. Q4 — Router-side relay design (the DISCOVERY_RELAY-shaped half)

### 4.1 Why RESYNC needs a relay hop, not a local-terminate-and-act (unlike DRAIN)

DRAIN and NODE_IDENTIFY are router-originated or router-consumed-and-acted-upon locally — the router itself is the terminal *actor*, not just the terminal *parser*. RESYNC is structurally different: the router is the terminal **parser** (per §2, all ctl frames terminate here), but the terminal **actor** must be the access node holding the relevant `arqsend.Retransmitter` (§3.1) — an entity the router has no direct handle to. This is the same shape DISCOVERY_RELAY has (router parses hop-1, then must relay hop-2 to other admitted nodes), not the shape DRAIN has (router parses and acts, full stop).

### 4.2 The target-resolution primitive DISCOVERY_RELAY's own design flagged as missing now exists

`S-BL.DISCOVERY-WIRE-rulings.md` Ruling 3(f), and ARCH-03 v1.8's own changelog, named an explicit open dependency for hop-2 relay: *"resolving which live connections currently serve a given SVTN's admitted nodes requires node-identity-to-connection binding that does not yet exist in production code."*

That gap has since been closed by **S-BL.NODE-IDENTIFY-WIRE**, chronologically after the DISCOVERY_RELAY ruling (BC-2.01.008 v1.3, 2026-07-15, postdates v1.2's 2026-07-13 DISCOVERY_RELAY entry). `internal/routing/identity.go` now ships exactly this binding:

```go
func (r *Router) BindInterface(svtnID [16]byte, nodeAddr [8]byte, ifaceID InterfaceID)
func (r *Router) LookupInterface(svtnID [16]byte, nodeAddr [8]byte) (InterfaceID, bool)
func (r *Router) UnbindInterface(svtnID [16]byte, nodeAddr [8]byte, callerIfaceID InterfaceID)
func (r *Router) InterfacesForSVTN(svtnID [16]byte, excludeNodeAddr [8]byte) []InterfaceID
```

`BindInterface` is called from `onAccept` immediately after a successful NODE_IDENTIFY handshake (`mgmt_wire.go`, `node_identify_wire.go:383`), for **every** admitted node — access nodes and consoles alike (BC-2.01.010's own scope is general, not discovery-specific). `LookupInterface(svtnID, nodeAddr) (InterfaceID, bool)` is the exact single-target primitive RESYNC needs (as opposed to DISCOVERY_RELAY's fan-out `InterfacesForSVTN`): given the target access node's `nodeAddr` (carried in the RESYNC frame's outer-header `DstAddr`, set by the console when it composes the frame — it already knows which access node its session is bound to), the router resolves the access node's **current** `InterfaceID`, independent of whichever specific TCP connection is live at the moment (LWW-updated on every reconnect, per `BindInterface`'s doc comment).

**This is worth recording as a correction to the architecture record**: DISCOVERY_RELAY's flagged open dependency (Ruling 3(f)) is resolved as of this writing. RESYNC's relay design should cite `LookupInterface`, not re-flag the same gap.

### 4.3 Router-side relay shape (mirrors `discovery_relay_wire.go` exactly)

```go
// cmd/switchboard/resync_wire.go (new file, mirrors discovery_relay_wire.go)

const resyncControlType = 0x02

// assembleResyncFrame builds the ctl(0x03)/control_type=0x02 outer frame.
// Pure function — no I/O. See §6 for the payload layout (pending BC amendment).
func assembleResyncFrame(svtnID [16]byte, srcAddr, dstAddr [8]byte, chanID uint32, resyncFromSeq uint32) []byte {
    payload := make([]byte, 0, 12)
    payload = append(payload, resyncControlType, frame.VersionByte, 0x00, 0x00)
    payload = binary.BigEndian.AppendUint32(payload, chanID)
    payload = binary.BigEndian.AppendUint32(payload, resyncFromSeq)

    ehdr := frame.EncodeOuterHeader(frame.OuterHeader{
        Version:    frame.VersionByte,
        FrameType:  frame.FrameTypeCtl,
        SVTNID:     svtnID,
        SrcAddr:    srcAddr, // the console's own node address
        DstAddr:    dstAddr, // the target access node's node address
        PayloadLen: uint16(len(payload)),
    })
    return append(append([]byte{}, ehdr[:]...), payload...)
}

// relayResyncToAccessNode resolves dstAddr's live InterfaceID and forwards
// the RESYNC frame verbatim onto that connection's send channel.
// Best-effort, non-blocking — matches relayDispatch's TOCTOU/full-channel posture exactly.
func relayResyncToAccessNode(router *routing.Router, sendMap *sync.Map, hdr frame.OuterHeader, raw []byte) {
    ifaceID, ok := router.LookupInterface(hdr.SVTNID, hdr.DstAddr)
    if !ok {
        return // target not currently connected — silent drop, same posture as relayDispatch
    }
    val, ok := sendMap.Load(ifaceID)
    if !ok {
        return // TOCTOU: connection closed between lookup and send
    }
    nc := val.(*nodeConn)
    select {
    case nc.send <- raw:
    default: // best-effort — matches relayDispatch/DRAIN posture
    }
}
```

Wired into `buildRoute`'s switch as `case 0x02: relayResyncToAccessNode(router, &sendMap, hdr, append(append([]byte{}, /* re-encode hdr */), payload...))` — or, more simply, relay the **original received bytes verbatim** (outer header + payload, exactly as the router read them off the console's connection) rather than re-assembling, since — unlike DISCOVERY_RELAY, which deliberately re-serializes to strip hop-1's meaningless HMAC tag — RESYNC's hop-1 tag is already zero (§1.4) and the payload is already in its final wire shape. This is a simpler relay than DISCOVERY_RELAY's, closer to a raw byte forward.

**Difference from DISCOVERY_RELAY worth flagging explicitly**: DISCOVERY_RELAY fans out to *every other* admitted node (`InterfacesForSVTN`, exclude-originator). RESYNC is a **single-target** relay (`LookupInterface`, one specific access node) — the console names its own access node in `DstAddr`, and the router forwards to exactly that one connection. Confirm this reading with product-owner/story-writer (§9) — it depends on the session model actually being one-access-node-per-session, which ARCH-03's "Sender (access node)" singular language supports but which this note has not independently verified against `internal/session`.

---

## 5. Q5 — Verification that this preserves the routing model (VP-015 recap)

### 5.1 `SVTNRoute` genuinely does not deliver bytes — confirmed in code, not just asserted in the BC

`internal/routing/routing.go:335-353`:

```go
func SVTNRoute(hdr frame.OuterHeader, payload []byte, r *Router) error {
    ...
    if entry == nil {
        return ErrNoForwardingEntry
    }
    _ = payload // payload is forwarded but not parsed here (R-001)
    _ = entry   // entry holds the authKey for wire-layer HMAC; available for future use
    return nil
}
```

`SVTNRoute` is validation-only — it never writes a byte to any connection. This matches BC-2.01.008 Invariant 2(b) verbatim (*"SVTNRoute performs forwarding-table validation only and does not relay frame bytes to any other connection"*) and confirms the relay design in §4 is correctly placed **outside** `SVTNRoute`'s call graph, alongside DRAIN's `sendMap.Range` broadcast and DISCOVERY_RELAY's `relayDispatch` — not as a modification to `SVTNRoute` itself. VP-015 is unaffected by this story for exactly the reason BC-2.01.005 PC-2's carve-out already states.

---

## 6. Q6 — RESYNC control-payload layout: a BC amendment IS required (flagged, not made)

### 6.1 The gap

BC-2.01.008 v1.3 Postcondition 3 currently states:

> "The control message header at offsets 0–3 (control_type, version, reserved) is fixed at 4 bytes for every opcode... **For the DRAIN and RESYNC opcodes specifically, this 4-byte header is the entire control message layout**."

This is correct for DRAIN (a pure signal, no payload needed) but is **not sufficient for RESYNC**, which — per ADR-005's own text — must convey a specific resync-from sequence number (`last_acked_seq + 1`) to the access node, and (per §4.3's design, needed to disambiguate which half-channel/session a multi-channel access node should replay) a channel identifier. Four bytes leaves zero room for either: bytes 0–1 are `control_type`+`version`, bytes 2–3 are `reserved` and *"must be zero-filled; receiver MUST ignore"* (PC-3). There is no field to carry a `uint32` sequence number today.

### 6.2 Required amendment — exact content, for product-owner to adjudicate and execute

This note does **not** edit BC-2.01.008 or mint a new BC. It specifies the exact amendment content needed so product-owner can execute it (or a variant they prefer) without re-deriving the analysis:

**Option A (recommended) — new sibling BC in the ARQ subsystem, matching the DISCOVERY_RELAY precedent.** DISCOVERY_RELAY's own extended payload is *not* defined inline in BC-2.01.008 — it is defined in `BC-2.03.001` (the discovery subsystem's own BC), with BC-2.01.008 carrying only a one-line registry citation + a short "extension" note (BC-2.01.008 v1.2, Postcondition 3's "DISCOVERY_RELAY extension" callout). RESYNC's ARQ-replay semantics belong in the ARQ subsystem (`ss-02`), not the frame-schema subsystem (`ss-01`), by the same logic. Next available ID in `ss-02`: **BC-2.02.010**.

Proposed new BC — **BC-2.02.010: RESYNC-Triggered Downstream ARQ Replay on Reconnect** — must define, at minimum:
- **Payload layout**, extending the shared 4-byte control header per BC-2.01.008 Invariant 5 (DI-007)'s pre-existing "future control messages MAY extend beyond byte 3" allowance:

  | Offset | Size | Field | Notes |
  |--------|------|-------|-------|
  | 0 | 1 | control_type | `0x02` |
  | 1 | 1 | version | `0x01` |
  | 2 | 2 | reserved | zero-filled; receiver MUST ignore |
  | 4 | 4 | chan_id | u32 big-endian; identifies the downstream half-channel this resync targets (matches `ChannelHeader.ChanID`'s type, BC-2.01.005 PC-3) |
  | 8 | 4 | resync_from_seq | u32 big-endian; the sequence number to retransmit FROM — i.e., `last_acked_seq + 1`, computed by the emitter (console) at emission time |

  Total: 12 bytes fixed, no variable-length tail.
- **Emitter trigger condition** (gap detection on reconnect — see §7.1).
- **Receiver behavior** (replay-loop semantics — see §7.2), including what happens when `resync_from_seq` names a sequence the access node's `SendBuffer` no longer holds (already-evicted / never-sent — an edge case BC-2.02.005's own sliding-window semantics bound, but RESYNC's specific EC needs its own row).
- **Relationship to BC-2.02.005**: RESYNC-triggered replay is a *second entry point* into the same retransmit machinery BC-2.02.005 PC-3/PC-5 already governs (gap-detected retransmit) — this BC should cite BC-2.02.005 as a Related BC and be explicit that it does not redefine PC-5's "original payload, new `chan_seq`" QUIC-model contract, only adds a new trigger for it.
- **HMAC/authentication re-adjudication** (§1.4's caveat) — whether RESYNC, as a state-changing trigger, needs a stronger boundary than the DRAIN precedent's bare "connection-is-the-boundary" posture. Recommend at minimum an explicit statement of the accepted risk (a hostile or compromised console connection can force redundant retransmission of already-delivered data — bounded by BC-2.02.005's own sliding-window size, not unbounded amplification) rather than silent inheritance.

**Option B (not recommended, offered for completeness) — extend BC-2.01.008 inline**, following the "DISCOVERY_RELAY extension" callout pattern used for the note under BC-2.01.008 PC-3, rather than a new BC. This keeps all `ctl`-opcode schema in one place but stacks unrelated-subsystem semantics (frame-schema BC now also owns ARQ-replay trigger semantics) onto BC-2.01.008, which is titled and scoped as "Router-Terminated Control Frame Payload Schema" — a framing/schema BC, not a behavioral BC for what the opcode *does*. Not the house pattern (compare: BC-2.01.008 does NOT define DISCOVERY_RELAY's own behavior, only cites BC-2.03.001 for it).

Either way, BC-2.01.008 itself needs a small, mechanical update regardless of which option is chosen: its registry-table RESYNC row's "Description" column and Postcondition 3's DRAIN/RESYNC lead sentence need updating to reflect that RESYNC is no longer a bare-4-byte opcode (matching the exact wording pattern BC-2.01.008 v1.2 used when DISCOVERY_RELAY stopped being bare-4-byte) — and, if Option A is chosen, a citation to the new BC-2.02.010 alongside the existing "S-BL.DISCOVERY-WIRE... payload layout defined there, not here" precedent in BC-2.01.008's Related BCs section.

**This amendment is not made by this note.** Per this task's constraint, it is specified in full so product-owner can execute it directly.

---

## 7. Q7 — Emitter and receiver+replay design

### 7.1 Emitter (console side): gap detection

Per ADR-005: *"the receiver detects the gap between its `last_acked_seq` and the first received `chan_seq` post-reconnect."* Mechanically, on the console side, after a data-plane reconnect completes (console re-establishes its wire connection to a router — itself gated on the same access-mode-analog connector work flagged in §3.3, since the console side has the identical "no data-plane dial exists yet" gap as the access-node side, per §3.2's grep), the console's downstream ARQ receiver state (`internal/arq.ARQ`'s receive-side `nextExpected`/reorder state, or equivalent console-side tracking) already knows its own `last_acked_seq` (the cumulative-ACK value it last computed via `OnAck`, per ARCH-03 §"Downstream ARQ"'s receiver-side `RecvBuffer`). On the first post-reconnect frame's arrival, compare its `chan_seq` against `last_acked_seq + 1`:
- If `chan_seq == last_acked_seq + 1`: no gap, no RESYNC needed — the sender's own SendBuffer window still covered the outage (a genuinely available fast path when the outage was shorter than the ARQ window).
- If `chan_seq > last_acked_seq + 1`: a gap exists. Emit `assembleResyncFrame(..., resyncFromSeq: last_acked_seq + 1)` immediately, once, on the reconnected connection.

This gap-detection comparison itself is a small, pure, unit-testable piece of logic (`uint32` comparison with wraparound handling matching `internal/arq`'s existing `OnAck` unsigned-subtraction convention, per ARCH-03's RULING-003 citation) — independent of whether the surrounding reconnect machinery exists yet. It can be built and tested today (§8, Layer 1).

### 7.2 Receiver + replay (access node side): the `arqsend.Retransmitter` loop

`arqsend.Retransmitter.Retransmit(oldSeq, newSeq uint32, now time.Time, dispatch Dispatch) error` (`internal/arqsend/arqsend.go:133-164`) replays **one** sequence per call — it takes an explicit `(oldSeq, newSeq)` pair, not a range. RESYNC's semantic ("replay everything from `resync_from_seq` onward") requires the caller to **enumerate** the in-flight sequence range and call `Retransmit` once per entry:

```go
// Sketch — NOT the arq.ARQ API today (see gap below).
for oldSeq := resyncFromSeq; arq.InFlightContains(oldSeq); oldSeq++ {
    newSeq := nextSendSeq()
    if err := retransmitter.Retransmit(oldSeq, newSeq, time.Now().UTC(), dispatch); err != nil {
        // ErrSequenceNotInFlight is not an error here — walking off the end of the window is the natural loop terminator.
        break
    }
}
```

**Gap identified**: `internal/arq.ARQ` today exposes `InFlightContains(seq uint32) bool`, `PayloadForInFlight(seq uint32) []byte`, `RemoveInFlight(seq uint32)`, and `GapsToRetransmit(ackSeq uint32, sackBitmap [64]byte) []uint32` — but `GapsToRetransmit` is SACK-bitmap-driven (designed for the steady-state "receiver tells me exactly which frames it's missing" path, BC-2.02.005 PC-3), not "replay everything from N to the current send-window head regardless of SACK state" (which is RESYNC's actual need — the console just reconnected and has no meaningful SACK state for the outage window). There is **no existing `internal/arq.ARQ` method to enumerate "every in-flight seq >= N"**. A loop using `InFlightContains` in a naive incrementing `for` starting at `resyncFromSeq` works only if in-flight sequences are contiguous (they are, by construction, for a sliding-window sender that has not yet seen any loss — which is exactly RESYNC's precondition, since loss/reordering is explicitly out of scope for this codebase's current ARQ model per `internal/testenv/loopback.go`'s own "Non-Goals excludes packet loss and reordering" note) — but this should be an explicit, named primitive rather than an inline assumption baked into the caller. **Recommend a new `internal/arq.ARQ` method**, e.g. `InFlightSeqsFrom(minSeq uint32) []uint32` (or an iterator), added by this story or flagged as a prerequisite sub-task within it — this is `internal/arq` implementation work, not a BC change, and is within this story's own `architecture_modules` scope (the stub already lists `internal/arq`).

### 7.3 Reconnect state machine — where `last_acked_seq` persists

The stub's Non-Goals/scheduling note says this story "requires per-node connection concept in netingress." Per §3.3, that framing is not quite right — `netingress` is accept-only and holds no per-node identity concept beyond the current connection's `InterfaceID`. The actual continuity requirement is: **`last_acked_seq` (console-side) and the `SendBuffer`/in-flight state (access-node-side) must survive a TCP reconnect**, i.e., must be keyed by a stable identity (session/channel), not by the ephemeral `InterfaceID`/`net.Conn` of any one connection.

There is already a precedent for exactly this kind of continuity in the codebase, worth citing rather than re-inventing: `internal/testenv/testenv.go`'s `ConnectWithSourceIP` doc comment — *"Reconnecting with new source IP preserves the session ID"* — backed by `e.connsByKey[keyHex]` keyed on the node's Ed25519 public key, not on any transport-layer connection identity. The router's own `identityIfaceMap` (§4.2) is the production-code analog: `(svtnID, nodeAddr) → InterfaceID` is **already** LWW-updated across reconnects (`BindInterface`'s own doc comment: *"a node reconnect with a new TCP connection overwrites the prior binding"*). The correct place for `last_acked_seq`/`SendBuffer` continuity is the **session layer** (`internal/session`/`internal/arq`, keyed by session/channel identity), not `internal/netingress` (which is, and should remain, stateless about node identity beyond one connection's lifetime — that separation of concerns is exactly what `internal/routing`'s identity-binding module was built to own instead). Story-writer should scope this as: *"ARQ/session state is keyed by `(svtnID, nodeAddr, chan_id)` and is not torn down by a `netingress`/`upstreamdial`-style connection close — only by explicit session teardown."* Whether this invariant already holds in `internal/arq`/`internal/session` as built, or needs new work, is an open question for story-writer to verify against current code (§9) — this note's grounding did not extend to a full audit of `internal/session`'s lifecycle handling.

---

## 8. Q8 — Recommended scope split

### Layer 1 — buildable and testable now, within this story, following the DRAIN/DISCOVERY_RELAY/NODE_IDENTIFY shape exactly

1. `cmd/switchboard/resync_wire.go`: `assembleResyncFrame` (pure) + `parseResyncFrame` (pure, payload → `(chanID, resyncFromSeq, error)`, with the `payload_len < 4`/`< 12` truncation guard per BC-2.01.008 EC-002's pattern) + `relayResyncToAccessNode` (effectful, §4.3).
2. `buildRoute`'s `switch` in `mgmt_wire.go` gains `case 0x02:` — discharges FO-DRAIN-WIRE-001.
3. Gap-detection pure logic (§7.1) — a small function, e.g. `shouldEmitResync(lastAckedSeq, firstPostReconnectChanSeq uint32) (resyncFromSeq uint32, needed bool)`, independently unit-testable with hand-constructed inputs, no live connection needed.
4. Replay-loop logic (§7.2) against `arqsend.Retransmitter`, using a fake `Dispatch` (exactly the pattern `arqsend`'s own existing unit tests already use, per its package doc's description of the seam) — testable without any real network connection, same as `arqsend_test.go` presumably already does for the plain retransmit path.
5. The candidate `internal/arq.InFlightSeqsFrom` primitive (§7.2).
6. Unit/integration tests for 1–5, in isolation — router-side dispatch test (inject a hand-built RESYNC frame into `buildRoute`, assert relay onto a fake `sendMap` entry, mirroring `discovery_relay_wire_test.go`'s `buildRelayRouter` pattern), emitter-logic test, replay-loop test.

### Layer 2 — blocked on prerequisites outside this story's current scope

7. A real access-node-side data-plane connector (dial + NODE_IDENTIFY handshake + reconnect/backoff + frame read/write loop) — does not exist, is architecturally comparable in size to `internal/upstreamdial.Connector`, and is not named as a dependency anywhere in the current spec set. This is the actual blocker for a genuine end-to-end round trip.
8. A real console-side data-plane connector — same gap, console side (distinct from the console's *management-plane* Unix-socket transport for attach/detach/switch, which `RULING-W6TB-C-console-transport.md` already settled as a **separate** transport from session data; that ruling does not cover the terminal I/O data stream itself).
9. The stub's AC-005 two-daemon, real-wire, "no content loss" integration test — depends on 7 and 8.

**Recommendation**: story-writer schedules Layer 1 as this story's actual scope, rewrites AC-005 to test the router-side relay and the emitter/replay logic against fakes/mocks (matching Layer-1's own test shape — no live two-daemon wire stack), and files Layer 2 as an explicit new forward-obligation/backlog item (an access-mode data-plane connector story) that S-BL.RESYNC-FRAME's own AC-005, once Layer 2 lands, can be re-opened against for genuine end-to-end verification. This mirrors exactly how DRAIN, DISCOVERY_RELAY, and NODE_IDENTIFY were each delivered against fakes/mocks first, with real end-to-end wiring landing incrementally across follow-on stories.

---

## 9. Open design questions for story-writer / product-owner

1. **BC-2.02.010 (or equivalent) must be created, or BC-2.01.008 amended inline** — product-owner decision, exact content in §6.2. Blocks finalizing this story's `bc_traces`.
2. **Single-target vs. any-fan-out relay** (§4.3) — confirm the one-access-node-per-session model holds; if a session can ever have multiple upstream access-node legs (not evidenced anywhere in the current spec set, but not independently disproven either), the relay design needs revisiting.
3. **Layer 2's access-mode/console-mode data-plane connector is not named as a dependency anywhere today.** Should it be filed as a new backlog story (this note recommends: yes, as an explicit prerequisite, symmetric in shape to `S-7.04-FU-PE-CONNECTOR` but for the access-node/console leg), and should S-BL.RESYNC-FRAME's dependency list be updated to name it once filed?
4. **HMAC/authentication posture for RESYNC** (§1.4, §6.2) — does inheriting the DRAIN/DISCOVERY_RELAY "connection is the trust boundary, zero-tag" posture need explicit re-adjudication given RESYNC's request-triggers-retransmission behavior (as the DRAIN PR's own security review already flagged as a named forward obligation), or is the "bounded to sliding-window-size redundant retransmission" risk framing in §6.2 sufficient?
5. **`internal/arq.InFlightSeqsFrom` (or equivalent enumeration primitive)** — confirm this is in-scope for this story (it already sits inside the stub's own `architecture_modules: [internal/arq]`) rather than a separate sub-task.
6. **Session/ARQ state continuity across reconnect** (§7.3) — needs a story-writer-time audit of `internal/session`/`internal/arq`'s actual lifecycle-teardown behavior (does closing a `netingress`/access-connector connection today tear down the associated ARQ/session state, or does it already survive independently?) — this note's grounding did not extend that far.
7. **AC-005 rescoping** (§8) — confirm the Layer 1 / Layer 2 split and the corresponding AC-005 rewrite.
8. **`bc_traces` correction** — drop `BC-2.01.002` (not a genuine RESYNC trace, §1.3), add `BC-2.01.008` (the actual schema home) and the new/amended ARQ-subsystem BC from §6.

---

## 10. Effort estimate

Original stub estimate: 5 points, against AC-001 through AC-005 as sketched (which included a full end-to-end round trip). Given the Q1 frame-model correction (removes the `internal/frame` FrameType work, adds a smaller wire-assembly task matching DISCOVERY_RELAY's shape) and the Q3/Q8 scope split (removes the unbuildable full end-to-end AC-005, narrows to Layer 1):

- **Layer 1 alone** (this note's recommended scope): comparable in shape and size to S-BL.DISCOVERY-WIRE's hop-2 relay slice (assemble + parse + relay + dispatch-arm + unit/integration tests against fakes) plus the `internal/arq` enumeration primitive and the emitter/replay pure-logic pieces. Estimate: **5–8 points** — similar order of magnitude to the original stub estimate, despite the narrower AC-005, because the BC-amendment dependency (§6) and the `internal/arq` primitive (§7.2) add real work the stub did not anticipate.
- **Layer 2** (access/console data-plane connectors): out of this story's estimate entirely; comparable in size to `S-7.04-FU-PE-CONNECTOR` itself (a full connector story), likely **8–13 points** as its own story, ×2 if access-mode and console-mode connectors are not shared code.

---

## 11. Traceability summary

| Question | Ruling | Confidence | Primary citations |
|---|---|---|---|
| Q1: frame model | `ctl(0x03)`/`control_type=0x02`, not new FrameType | Definitive | `internal/frame/frame.go:33-46`; ARCH-03 v1.8 changelog; BC-2.01.008 v1.0 PC-2/Inv-4; BC-2.01.004 v1.5 PC-2 |
| Q2: terminal-consumer / VP-015 | Preserved; dispatch inside `buildRoute`, never reaches `SVTNRoute` | Definitive | BC-2.01.008 Inv-2; BC-2.01.005 v1.2 PC-2; `mgmt_wire.go:603-658`; `routing.go:335-353` |
| Q3: access-mode data-plane gap | No production/test wire-level access-node↔router path exists today | High (grep-verified across all of `cmd/switchboard`) | `access.go` (no net.Dial); `mgmt_wire.go:565` (only dial-out is `upstreamdial`); `testenv/loopback.go`, `testenv/testenv.go` (in-process simulation only) |
| Q4: RESYNC payload | BC amendment required; content specified, not made | Definitive gap, recommended fix is a judgment call for product-owner | BC-2.01.008 v1.3 PC-3; BC-2.01.008 v1.2 changelog (DISCOVERY_RELAY precedent) |
| Q5: relay target resolution | `routing.LookupInterface` — DISCOVERY_RELAY's flagged gap is now resolved | High | `internal/routing/identity.go`; `node_identify_wire.go:383`; S-BL.DISCOVERY-WIRE-rulings.md Ruling 3(f) (superseded) |
| Q7: replay enumeration | No existing `arq.ARQ` primitive for "in-flight from N"; new primitive recommended | High | `internal/arq/arq.go` (exported func list); `internal/arqsend/arqsend.go:133-164` |
