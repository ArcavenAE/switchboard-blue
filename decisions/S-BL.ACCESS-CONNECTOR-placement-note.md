---
artifact_id: S-BL.ACCESS-CONNECTOR-placement-note
document_type: architect-design-note
story_id: S-BL.ACCESS-CONNECTOR
title: "Access-node data-plane connector — resolving BC-2.04.009's three flagged open architecture items"
status: draft
producer: architect
timestamp: 2026-08-31T00:00:00Z
version: "1.1"
bc_traces:
  - BC-2.04.009   # the commissioning BC this note decomposes — all three open items resolved below
  - BC-2.01.009   # NODE_IDENTIFY three-message handshake — router-side counterpart the client driver mirrors
  - BC-2.01.008   # control_type schema home — the codec-extraction finding below touches this BC's stated architecture_module split
  - BC-2.02.005   # downstream ARQ send/receive semantics — EnqueueSend composition for the regular-tick path
  - BC-2.04.001   # local tmux half — startFramesBridge/an.DeliverFrame precedent this note extends, not replaces
  - BC-2.04.003   # console attach/downstream-stream/upstream-keystroke semantics — SendKeystroke is this BC's own upstream entry point
vp_traces:
  - VP-081
  - VP-082
architecture_modules:
  - internal/accessdial     # new package — connector state machine (canonical name per BC-2.04.009 frontmatter)
  - internal/nodeidentify    # new package (proposed) — pure NODE_IDENTIFY codec, extracted from cmd/switchboard so both router and client sides can import it
  - cmd/switchboard          # node_identify_wire.go refactored to call the extracted codec; access.go wiring; new accessdial_wire.go dispatch site
  - internal/config          # new RouterAddr/SVTNID fields
  - internal/session         # SendKeystroke is the resolved Open Item 3 target; no code change to internal/session itself
  - internal/arq              # EnqueueSend composition (Open Item 2); no new exported primitive required here
  - internal/arqsend         # Retransmitter.Dispatch becomes accessdial.Connector.Send
related_documents:
  - .factory/specs/behavioral-contracts/ss-04/BC-2.04.009.md
  - .factory/specs/verification-properties/VP-081.md
  - .factory/specs/verification-properties/VP-082.md
  - .factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md
  - .factory/decisions/S-BL.RESYNC-FRAME-placement-note.md
  - .factory/stories/S-7.04-FU-PE-CONNECTOR.md
  - internal/upstreamdial/connector.go
  - cmd/switchboard/node_identify_wire.go
  - cmd/switchboard/access.go
  - internal/session/upstream.go
  - internal/halfchannel/halfchannel.go
  - internal/testenv/loopback.go
  - internal/config/config.go
---

## Changelog

| Version | Change |
|---------|--------|
| 1.1 | Corrected §6: struck the v1.0 claim that `FrameAuthKey`'s HKDF derivation is "fresh per handshake" — SUPERSEDED per product-owner spec-completeness ruling (VP-082 v1.1). `FrameAuthKey` is deterministic in `(svtn_id, node_admission_pubkey)` per ARCH-04-admission-security.md's canonical formula and BC-2.05.010's "derived deterministically" precedent — no handshake-specific entropy, by design (the router independently recomputes the same key; a nonce-salted key would break its forwarding-table cache, BC-2.05.008, on every reconnect). Consequence now stated: across a same-node reconnect the entire `outerassembler.Envelope` is byte-identical, not merely `SVTNID`-invariant. Cross-references VP-082 v1.1's corrected Test Scenario 3 (behavioral non-vacuity check, replacing the now-known-false value-inequality assertion). No other section touched. |
| 1.0 | Initial decomposition-ready placement note. Resolves BC-2.04.009's three flagged open architecture items with concrete, code-cited designs; specifies the connector package's public surface, state machine, reconnect/ARQ-preservation behavior, file structure, and task breakdown; surfaces one net-new architectural finding (the NODE_IDENTIFY codec functions are unexported and package-`main`-scoped, blocking direct reuse — a small extraction is required) and one residual scope boundary (multi-console outbound `DstAddr` fan-out is explicitly out of this story's scope). |

---

## 0. Summary

BC-2.04.009 (v1.0) commissions `internal/accessdial` but explicitly declines to resolve three architecture questions, naming them for architect resolution. All three are resolved here, each grounded in code already in the repository — two of the three (Open Items 2 and 3) resolve to primitives and precedents that *already exist and are already merged* (`internal/testenv/loopback.go`'s `onDownstreamTick` composition; `session.AccessNode.SendKeystroke`'s own doc comment naming itself "the safe path for production callers"), and the third (Open Item 1) resolves to a small, conventional `internal/config.Config` schema addition following the codebase's own `UpstreamRouter`/`AdmissionKeyFile` naming pattern. One additional finding surfaced during this resolution, not previously flagged anywhere: the NODE_IDENTIFY wire-codec functions `S-BL.ACCESS-CONNECTOR` needs to reuse (`encodeNodeIdentify`, `encodeChallengeResponse`, and the new `decodeChallenge`) are unexported, `package main`-scoped functions in `cmd/switchboard/node_identify_wire.go` — `internal/accessdial` cannot import them as-is. A small, behavior-preserving extraction (§5) is required and is included in this story's task breakdown, not deferred.

None of the three resolutions require editing BC-2.04.009's postconditions — each fills exactly the gap that postcondition's own text already flagged as "not settled by this BC" or "flagged for architect... resolution." One new edge case is recommended as an addition to BC-2.04.009 (§11) — flagged for product-owner, not made here.

---

## 1. Open Item 1 — SVTN-ID / router-address sourcing

**Resolution: new `internal/config.Config` fields, following the existing `UpstreamRouter`/`AdmissionKeyFile` conventions exactly.**

Verified: `internal/config/config.go`'s `Config` struct (`config.go:118-190`) has no field carrying either value today. The closest existing analog, `UpstreamRouters []UpstreamRouter` (`config.go:130`, `UpstreamRouter{ Addr string }` at `config.go:205-208`), is router-mode-only (PE upstream list, plural, validated as host:port by `Config.Validate()` per BC-2.09.003 PC-6) and is not reusable as-is — access mode needs exactly one router address, not a list, and needs an SVTN identifier that `UpstreamRouter` has no field for at all.

**New fields, added to `Config` alongside the existing access-mode-scoped fields (near `AdmissionKeyFile`, `config.go:153-165`):**

```go
// RouterAddr is the TCP address of the router this access-mode daemon dials
// (S-BL.ACCESS-CONNECTOR, BC-2.04.009). Access-mode only — ignored by
// router/console/control modes, mirroring the RouterManagementEndpoints /
// AdmissionStateFile mode-scoping convention already used for other fields.
// Required for access mode (empty is invalid — E-CFG-0NN, next free code).
// Validated as host:port by Config.Validate() — the same validateHostPort
// helper BC-2.09.003 PC-6 already applies to UpstreamRouter.Addr.
RouterAddr string `yaml:"router_addr"`

// SVTNID is the hex-encoded (32 lowercase hex chars, no separators) 16-byte
// SVTN identifier this access node is provisioned under (S-BL.ACCESS-CONNECTOR,
// BC-2.04.009). Access-mode only. Required for access mode. Config.Validate()
// checks the string decodes via encoding/hex.DecodeString to exactly 16 bytes
// (E-CFG-0NN, next free code) — it does NOT perform admission I/O; matching
// ARCH-06's Config-purity contract (Validate() never opens a socket or file).
SVTNID string `yaml:"svtn_id"`
```

**Why hex-string, not a structured type, and why this is the right operator-facing shape:** `internal/svtnmgmt.SVTN.ID` is canonically `[16]byte` (`svtnmgmt.go:87-89`, "derived from crypto/rand at creation time"), addressed operator-side by a human-readable `Name` (`svtn.go:90-92`, `SVTNByName(name string)`) — but that name→ID resolution lives entirely inside the **control**-mode daemon's `SVTNManager`, which the access-mode daemon has no coupling to and this story does not add one (that would be a materially larger scope addition — a live control-plane query at access-daemon startup — not named anywhere in this BC or the scoping note that commissioned it). `cmd/sbctl/admin.go` already carries a `SVTNID string` field on its RPC-facing structs (`admin.go:54,65,81,171`) for exactly this reason: the 16-byte ID crosses process/operator boundaries as a string once minted, and the operator is expected to copy the ID (not the name) from `sbctl admin svtn create`'s output into any config that needs to address that SVTN directly, offline, without a live control-plane round trip. `RouterAddr`/`SVTNID` in access-mode config is the same pattern applied one layer earlier: the operator provisions both values by hand (or by a deployment script) at the same time they provision `AdmissionKeyFile`, exactly mirroring how `UpstreamRouters` is hand-provisioned for PE-mode routers today.

**Does this change any BC-2.04.009 postcondition?** No. Precondition 2 already states *"the access-mode daemon has, by some mechanism, a single router address... and the target SVTN ID"* and explicitly declines to define the mechanism. This resolution supplies that mechanism; it does not alter what the BC requires once the values exist.

**`access.go` wiring:** `runAccess`/`runAccessWithConnector` read `cfg.RouterAddr` and `cfg.SVTNID` (hex-decoded once, at startup, alongside the existing `admissionKeyPath` resolution block, `access.go:255-282`) and pass both into the new `internal/accessdial.Connector` constructor. This also, as a direct side effect, is the first real, non-test-only value ever assigned to `discovery.Config.LocalSVTNID` (`access.go`'s existing discovery wiring, `discoveryCfg := discovery.Config{ LocalNodeAdmissionPubkey: ... }`, `access.go:313-318`, whose own comment already names this exact gap: *"LocalSVTNID: populated by S-BL.NODE-IDENTIFY-WIRE"* — a forward obligation from that story that has remained open in the codebase; `cfg.SVTNID` closes it here as a byproduct, not as this story's primary purpose). `discovery.Config.LocalNodeAddr` is populated from `frame.DeriveNodeAddress(svtnID, admissionPubKey)` at the same call site, using the same derivation `nodeIdentifyHandshake` already uses router-side (`node_identify_wire.go:320`).

---

## 2. Open Item 2 — regular (non-retransmit) downstream send-path composition

**Resolution: mirror `internal/testenv/loopback.go`'s already-merged, already-tested `onDownstreamTick` composition — `EnqueueSend` before dispatch, one shared `Send`/`Write` path for both regular ticks and retransmits — applied to a real `net.Conn` instead of an in-process delivery.**

**Where the regular per-tick emission currently goes, and why it's not wired to any wire connection today:** `cmd/switchboard/access.go`'s `startFramesBridge` (`access.go:564-585`) is the *existing* consumer of the daemon's per-tick downstream `ChannelFrame` stream — it already runs today, already fires on every tick (its `framesCh <-chan halfchannel.ChannelFrame` is `sc.Frames()`, i.e. `internal/tmux`'s `SessionConnector`, which owns ticking the `ds` half-channel constructed at `access.go:197` and enqueuing tmux control-mode output into it — the ticking itself is *already fully wired*, just never reaches a wire). Today `startFramesBridge` does exactly one thing per frame: `an.DeliverFrame(frame.OuterHeader{FrameType: f.FrameType, PayloadLen: uint16(len(f.Payload))})` — an **in-process-only** fan-out to locally-attached consoles (real for `internal/testenv`'s loopback harness; a no-op in production today, since production has no locally-attached consoles — remote consoles are exactly what `S-BL.CONSOLE-CONNECTOR` would eventually add). `startFramesBridge` never touches the wire, never calls `outerassembler.Assemble`, never calls `arq.ARQ.EnqueueSend`.

**The composition to add, precedented directly by `internal/testenv/loopback.go`'s `onDownstreamTick` (`loopback.go:220-291`, merged and tested under `S-BL.LOOPBACK-FULLSTACK`):**

```go
// [B1] capture chanSeq before any transformation — matches the loopback
// driver's own documented reason (multipath.Frame / outerassembler input
// shapes don't carry ChanSeq back out; capture it up front).
chanSeq := f.ChanSeq

if f.FrameType == halfchannel.FrameTypeData {
    // EnqueueSend registers this frame as in-flight BEFORE it is
    // dispatched, so a later GapsToRetransmit/RESYNC-driven replay (once
    // S-BL.RESYNC-FRAME Layer 1 lands) has something to retransmit FROM.
    // Precedent: loopback.go:253, `d.downstreamARQ.EnqueueSend(chanSeq, f.Payload, time.Now().UTC())`.
    arqHandle.EnqueueSend(chanSeq, f.Payload, time.Now().UTC())
}

// Unlike the loopback harness (which deliberately does NOT wire-dispatch
// empty ticks — its own comment: "Non-Goals excludes packet loss and
// reordering" scoped that decision to ITS harness, not to production wire
// behavior), a REAL wire connection must assemble and send EVERY tick,
// including empty ones: BC-2.01.002 DI-008 ("empty-tick frames are never
// skipped... an implementation that omits empty-tick frames when no data
// is pending violates this invariant and breaks quality monitoring") is a
// wire-level liveness invariant, not scoped to any one story's harness.
cf := halfchannel.ChannelFrame{
    ChanID:    f.ChanID,
    ChanSeq:   chanSeq,
    FrameType: f.FrameType,
    Flags:     f.Flags,
    Payload:   f.Payload,
}
var zeroSACK [outerassembler.SACKBitmapSize]byte // this direction does not originate SACK; SACK flows the other way, piggybacked on inbound upstream frames the connector's read loop receives (§Open Item 3) — matches arqsend.Retransmitter's own zero-sackBitmap usage exactly (arqsend.go:147)
wire, err := outerassembler.Assemble(cf, zeroSACK, connector.Envelope())
if err != nil { /* log + continue — do not crash the bridge goroutine on one bad frame */ }
if err := connector.Send(wire); err != nil { /* log — Send's own retry/backoff posture is the connector's concern, not the bridge's; see §4 */ }

// Existing in-process fan-out is UNCHANGED, additive, not replaced — a
// future access node may have both locally-attached (test/loopback) and
// remotely-attached (S-BL.CONSOLE-CONNECTOR, once it exists) consoles at
// once; startFramesBridge's own an.DeliverFrame call stays exactly as-is.
an.DeliverFrame(frame.OuterHeader{FrameType: f.FrameType, PayloadLen: uint16(len(f.Payload))})
```

This is added as a **second function alongside `startFramesBridge`**, not a rewrite of it — e.g. `startWireBridge(an *session.AccessNode, framesCh <-chan halfchannel.ChannelFrame, arqHandle *arq.ARQ, connector *accessdial.Connector)`, started as its own goroutine consuming a `framesCh` fan-out (or, more simply, `startFramesBridge` itself gains the wire-dispatch lines inline and keeps its existing name — implementer's call; either shape satisfies BC-2.04.009 Postcondition 4b). Keeping them separable is preferable: `startFramesBridge` remains independently testable exactly as it is today (no `internal/accessdial` dependency at all), and the new wire-dispatch logic is testable in isolation against a fake `connector.Send` (§9).

**The single-writer requirement (BC-2.04.009 Invariant 3) is satisfied by construction**: `connector.Send(wire []byte) error` is the **one** exported write entry point on `*accessdial.Connector`. Both this regular-tick path and `arqsend.Retransmitter`'s `Dispatch` callback (§4) call it — never `net.Conn.Write` directly from outside the connector package. `Send` itself holds a mutex (or routes through a single internal writer goroutine via a channel — implementer's choice; a mutex is simpler and matches `arqsend.Retransmitter`'s own single-writer discipline note in its package doc) guarding the live `net.Conn`.

**Does this change any BC-2.04.009 postcondition?** No. Postcondition 4b's own text already anticipates and requests exactly this resolution ("flagged for architect/story-writer resolution").

---

## 3. Open Item 3 — inbound upstream-keystroke frame delivery target

**Resolution, part (a) — the delivery target is definitively `session.AccessNode.SendKeystroke(key ConsoleKey, sessionName string, payload []byte) error`, not `DeliverFrame`.**

This is settled directly by `internal/session/upstream.go`'s own doc comments on `Attach` and `SendKeystroke` — not an inference, a stated fact in the code:

- `Attach`'s doc comment (`upstream.go:176-189`): *"AccessNode is goroutine-free: no per-console consumer goroutine is started. Keystrokes are forwarded synchronously via SendKeystroke → sink.SendInput... The upstream channel is still returned for callers that write directly to it (e.g. test helpers); however, AccessNode does NOT drain it — callers must use SendKeystroke for the authorizer + serialization guarantees... The safe path for production callers is SendKeystroke, which does not write to the upstream channel at all; direct channel writes are for test harnesses only."*
- `SendKeystroke` (`upstream.go:254-301`) is exactly the production entry point: it consults `ConsoleSet.Session` to verify the console is attached and to the right session, runs the `Authorizer` gate (Tier-2 authorization, BC-2.05.003), and forwards to `sink.SendInput` under `sinkMu` (serialization, BC-2.04.006 Invariant 3) — precisely the "resolves a frame to its authorized console and forwards it to `KeystrokeSink`" language BC-2.04.009 Postcondition 5 already specifies as the requirement.
- `DeliverFrame` (`upstream.go:304-306`, one line: `a.consoles.Deliver(hdr)`) is confirmed fan-out-only, matching BC-2.04.009's own suspicion.

This resolves cleanly and requires **no new method on `internal/session`** — `SendKeystroke`'s signature and behavior are exactly right for this purpose already, and `internal/testenv/loopback.go`'s own `deliverUpstream` (`loopback.go:313-327`) is a direct, already-merged precedent for calling it from a wire-adjacent delivery path.

**Resolution, part (b) — the genuinely open sub-problem: resolving `(ConsoleKey, sessionName)` from an inbound `(frame.OuterHeader, payload)`.**

`SendKeystroke` needs a `ConsoleKey` (`type ConsoleKey string`, `fanout.go:27` — an opaque string token, not derivable by any existing formula from `frame.OuterHeader.SrcAddr [8]byte`) and a `sessionName`. The connector's read loop receives `(hdr, payload)` with `hdr.SrcAddr` identifying the sending console's node address and the channel header's `chan_id` identifying which half-channel — neither is, by itself, a `ConsoleKey`. There is no existing binding from `(SrcAddr, chan_id)` to `(ConsoleKey, sessionName)` anywhere in the codebase today, because — as `S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §3 already found — the wire-level console-attach flow that would naturally populate such a binding (`S-BL.CONSOLE-CONNECTOR`) does not exist yet either; `AccessNode.Attach` is invoked only by test code today (`internal/testenv/loopback.go:463`, `d.access.Attach(d.loopbackConsoleKey, sessionName)`).

**This story's bounded scope**: `internal/accessdial` (or a small adjacent piece in `cmd/switchboard`) owns a narrow, explicit registry —

```go
// In cmd/switchboard (new file, accessdial_wire.go — see §7 File Structure),
// mirroring discovery_relay_wire.go's shape: policy/dispatch code lives in
// cmd/switchboard, not in the generic internal/accessdial package.
type consoleBindingKey struct {
    srcAddr [8]byte
    chanID  uint32
}

// consoleBindings is populated by whatever FUTURE mechanism resolves a
// console's wire-level attach to a (ConsoleKey, sessionName) pair — that
// mechanism is explicitly OUT of this story's scope (S-BL.CONSOLE-CONNECTOR
// or a session-bootstrap follow-on; see §10 residual open questions).
// This story defines the registry and its lookup/miss behavior; it does
// NOT populate it in production. A RegisterConsoleBinding(srcAddr, chanID,
// key, sessionName) method is exported for that future caller.
var consoleBindings sync.Map // consoleBindingKey -> (session.ConsoleKey, sessionName string)
```

On a **binding miss**, the read-loop dispatcher logs and drops the frame — it does not crash, does not close the connection, and does not treat the miss as a protocol violation. This mirrors the already-established "unknown/forward-compat, silent-ignore" posture BC-2.01.008 Postcondition 4 sets for unrecognized `control_type` values, applied here to an unrecognized-*sender* inbound data frame rather than an unrecognized opcode — the same shape, a different axis of "not yet known to this receiver, not an error."

**Testability**: exactly this narrow scope makes the read-loop dispatch unit-testable today, without any console-attach wire protocol existing — a test calls `RegisterConsoleBinding` directly (mirroring how `discovery_relay_wire_test.go`'s `buildRelayRouter` populates router state directly via `BindInterface` rather than driving a real handshake), injects a hand-built inbound frame, and asserts `SendKeystroke` was called with the right arguments (or, for the miss case, that it was not called and no crash occurred).

**Does this change any BC-2.04.009 postcondition?** No — Postcondition 5's text already names `SendKeystroke`/`KeystrokeSink` as the target and explicitly flags "the concrete `session.AccessNode`-side target... is not identified by this BC" as the open item; this resolves exactly that, without altering the postcondition's requirement. One new **edge case** is worth adding to BC-2.04.009 (not present today: what happens on a binding miss) — flagged for product-owner in §11, not made here.

---

## 4. Connector package design — `internal/accessdial`

### 4.1 Public surface (mirrors `internal/upstreamdial.Connector`'s `Handle` interface shape, `S-7.04-FU-PE-CONNECTOR` Design Constraints §"Dial-Loop Architecture")

```go
package accessdial

type State int

const (
    StateDialing State = iota  // no live TCP connection; retrying with backoff
    StateAdmitting              // TCP connected; NODE_IDENTIFY exchange in progress
    StateLive                   // admitted; Envelope populated; safe to Send
)

type Handle interface {
    State() State                        // atomic.Load — safe from any goroutine, no mutex (go.md rule 12), mirrors upstreamdial.Connector.Mode()
    Envelope() outerassembler.Envelope    // value type; current live envelope, or zero value if not StateLive
    Send(wire []byte) error               // the ONE exported write path (§2); returns an error if not StateLive
    Stop()                                 // cancels the dial/handshake/live goroutines and waits for exit, mirrors upstreamdial.Connector.Stop()
}

// SetFrameCallback registers the inbound-frame dispatcher, set once before
// Start() — set-once pre-launch ordering identical to upstreamdial.Connector's
// own SetFrameCallback contract (connector.go, F-SP4-002 precedent).
func (c *Connector) SetFrameCallback(fn FrameFn)

type FrameFn func(hdr frame.OuterHeader, payload []byte) error

func New(routerAddr string, svtnID [16]byte, keypair ed25519.PrivateKey, arqHandle *arq.ARQ) *Connector
func (c *Connector) Start()
```

### 4.2 State machine (BC-2.04.009 Postcondition 6) — mirrors `internal/upstreamdial.dialLoop`'s already-shipped shape

`internal/upstreamdial.dialLoop` (`connector.go:318-459`, PE-CONNECTOR, merged) is the direct structural precedent — its own steps 1–8 (`net.Dial` → bootstrap-assemble → write → spawn receive goroutine → `maintainConn` keepalive loop → join on exit → decrement/backoff → redial) map almost one-to-one onto BC-2.04.009's three-state model, with the bootstrap-write step (PE-CONNECTOR's single `outerassembler.Assemble`+`conn.Write` of a placeholder `FrameTypePEConnect` frame) replaced by the full three-message client-side handshake:

```
StateDialing  --net.Dial succeeds-->  StateAdmitting
StateAdmitting --NodeIdentify sent, Challenge received+decoded, ChallengeResponse sent,
                  router does not close the connection-->  StateLive
StateAdmitting --any read/write error, or router closes the connection
                  (BC-2.01.009 PC-10: no explicit rejection, only closure)--> StateDialing (backoff)
StateLive --read/write error, or explicit Stop()--> StateDialing (backoff) [or terminal, on Stop()]
```

No state ever carries session data except `StateLive` — the router-side mirror of this same rule is BC-2.01.009 Postcondition 8's "fully bound or closed, no unbound-open state," and BC-2.04.009 Postcondition 6 states the connector-side half of the identical invariant explicitly.

The `StateAdmitting` handshake itself (client-side, the mirror of `nodeIdentifyHandshake`, `node_identify_wire.go:269-392`):

```go
func (c *Connector) admit(conn net.Conn) (outerassembler.Envelope, error) {
    conn.SetDeadline(time.Now().Add(c.handshakeTimeout)) // value TBD at implementation (§10); mirrors nodeIdentifyHandshakeTimeout=10s, node_identify_wire.go:56
    defer conn.SetDeadline(time.Time{})

    pubkey := c.keypair.Public().(ed25519.PublicKey)
    if _, err := conn.Write(nodeidentify.EncodeNodeIdentify(c.svtnID, pubkey)); err != nil {
        return outerassembler.Envelope{}, err
    }

    hdr, payload, err := frame.ReadOuterFrame(conn)
    if err != nil { return outerassembler.Envelope{}, err }
    challenge, err := nodeidentify.DecodeChallenge(payload)  // NEW function — see §5
    if err != nil { return outerassembler.Envelope{}, err }

    sig := ed25519.Sign(c.keypair, challenge.Nonce[:])       // DI-002: only the signature crosses the wire next, never c.keypair itself
    resp := admission.ChallengeResponse{NonceSig: sig}
    if _, err := conn.Write(nodeidentify.EncodeChallengeResponse(c.svtnID, resp)); err != nil {
        return outerassembler.Envelope{}, err
    }

    // BC-2.01.009 PC-10: the router returns no explicit success/failure
    // message. "Admission succeeded" is inferred by the connection
    // remaining open long enough for the connector's own first live write
    // to succeed (BC-2.04.009 Postcondition 3.iii) — there is no fourth
    // message to wait for.
    nodeAddr := frame.DeriveNodeAddress(c.svtnID, pubkey)     // node_identify_wire.go:320 precedent
    return outerassembler.Envelope{
        SVTNID:  c.svtnID,
        SrcAddr: nodeAddr,
        // DstAddr: the router's own node address — see §10, residual open question:
        // routers are not addressed as "nodes" (BC-2.01.008 Inv-2a: SrcAddr/DstAddr
        // identify nodes, not routers); what DstAddr value a session-data frame from
        // the access node should carry is not settled by this note. A zero DstAddr
        // (or a router-provided value TBD at handshake time) is a placeholder,
        // exactly mirroring PE-CONNECTOR's own zero-envelope precedent for the one
        // field its own handshake didn't resolve either — see §10.
        FrameAuthKey: deriveFrameAuthKey(...), // HKDF derivation, mirrors hmac.DeriveKey precedent already used router-side
    }, nil
}
```

### 4.3 Forbidden edges (directly inherited from PE-CONNECTOR's own placement note ruling — same DAG-layering reasoning applies unchanged)

- `internal/accessdial` → `internal/routing` — **forbidden**. `internal/accessdial` never makes a routing decision; it dials one configured address. (Mirrors `S-7.04-FU-PE-CONNECTOR`'s own Q4 ruling verbatim.)
- `internal/accessdial` → `internal/testenv` — **forbidden**. Test-composition root; never imported by production code.
- `internal/accessdial` → `internal/drain` — not applicable/needed; not imported.
- **Permitted, narrow**: `internal/accessdial` → `{internal/frame, internal/outerassembler, internal/arq, internal/nodeidentify, internal/admission}` (the last two for handshake codec + `Challenge`/`ChallengeResponse` types only — no admission *verification* logic, which stays router-side).

---

## 5. A net-new finding: the NODE_IDENTIFY codec functions must be extracted into an importable package

`cmd/switchboard/node_identify_wire.go` is `package main`. `encodeNodeIdentify`, `encodeChallenge`, `encodeChallengeResponse`, `decodeNodeIdentify`, `decodeChallengeResponse` are all **unexported** (lowercase). `internal/accessdial` — a new package under `internal/` — cannot import `cmd/switchboard` (backwards dependency direction; `cmd/*` packages import `internal/*`, never the reverse) and could not call these functions even if it could, since they are unexported. This was not flagged by BC-2.04.009 or by `S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §2.1, both of which state the client-side handshake "reuses `encodeNodeIdentify`/`encodeChallengeResponse`" without checking this.

**Resolution**: extract the five pure codec functions, the three payload-size constants (`nodeIdentifyPayloadSize`, `challengePayloadSize`, `challengeResponsePayloadSize`), the `nodeIdentifyControlType` constant, and the three `msgKind*` constants into a new package, **`internal/nodeidentify`** (pure-core; imports only `internal/frame`, `internal/admission` (for the `Challenge`/`ChallengeResponse` types only), and stdlib `crypto/ed25519`). Export all five functions (capitalize: `EncodeNodeIdentify`, `EncodeChallenge`, `EncodeChallengeResponse`, `DecodeNodeIdentify`, `DecodeChallengeResponse`) and add the sixth, currently-missing one:

```go
// DecodeChallenge decodes the 100-byte Challenge payload — the client-side
// counterpart to DecodeChallengeResponse. No such function exists in the
// codebase today (BC-2.04.009 Postcondition 2b names this exact gap).
// Mirrors DecodeChallengeResponse's structure exactly (node_identify_wire.go:226-247).
func DecodeChallenge(payload []byte) (admission.Challenge, error) {
    if len(payload) != ChallengePayloadSize { return admission.Challenge{}, fmt.Errorf(...) }
    if payload[0] != ControlType { return admission.Challenge{}, fmt.Errorf(...) }
    if payload[1] != frame.VersionByte { return admission.Challenge{}, fmt.Errorf(...) }
    if payload[2] != MsgKindChallenge { return admission.Challenge{}, fmt.Errorf(...) }
    if payload[3] != 0x00 { return admission.Challenge{}, fmt.Errorf(...) }
    var c admission.Challenge
    copy(c.Nonce[:], payload[4:36])
    c.RouterSig = append([]byte(nil), payload[36:100]...)
    return c, nil
}
```

`cmd/switchboard/node_identify_wire.go` is refactored to call the extracted functions (thin wrappers, or the file's own five function bodies deleted in favor of direct calls to `nodeidentify.Encode*`/`nodeidentify.Decode*`) — **behavior-preserving**: identical wire bytes in, identical wire bytes out, so `S-BL.NODE-IDENTIFY-WIRE`'s existing router-side tests (`node_identify_wire_test.go`) continue to pass unmodified, exercising the same logic through its new location. This is a small, low-risk refactor of already-merged code, included explicitly in this story's task list (§7) rather than assumed away.

**Package placement**: DAG position TBD at implementation (architect/story-writer assigns per ARCH-08 §6's registration convention, same obligation `S-7.04-FU-PE-CONNECTOR`'s own placement note names for `internal/upstreamdial`). `internal/nodeidentify` sits below both `cmd/switchboard` and `internal/accessdial` in the import graph (both import it; it imports neither).

---

## 6. Reconnect + ARQ-state preservation design (VP-082)

**What survives a reconnect, and why, by construction rather than by any explicit "preserve" step:**

- **`*arq.ARQ` (the `SendBuffer`)** — owned by `cmd/switchboard`/`access.go`, constructed once at daemon startup (mirroring `arq.New(arq.Config{...})`, the `internal/testenv/loopback.go:173` precedent), and passed **by reference** into `internal/accessdial.New(...)` (§4.1's constructor signature already reflects this: `arqHandle *arq.ARQ` is a parameter, not something the connector constructs itself). The connector's `Stop`/reconnect machinery never holds, closes, or reconstructs this handle — it only reads from it (`EnqueueSend` on the regular-send path, §2; eventually `GapsToRetransmit`/`PayloadForInFlight` once `S-BL.RESYNC-FRAME` Layer 1's replay logic wires in). **The state survives because nothing in the reconnect path ever touches the pointer.** This is the same shape `internal/testenv`'s `ConnectWithSourceIP` precedent already establishes for session identity: *"reconnecting with a new source IP preserves the session ID"* (`testenv.go`, cited in `S-BL.RESYNC-FRAME-placement-note.md` §7.3) — the pattern generalizes directly: continuity is achieved by keying long-lived state on something *other than* the connection, and by construction, not by an explicit migration step.
- **`session.AccessNode` (console attachments, `ConsoleSet`, authorization state)** — same argument: `access.go` constructs `an *session.AccessNode` once at startup (`buildAccessComponents`, unchanged by this BC per its own Precondition 3) and the connector never receives ownership of it, only a narrow callback surface (`SendKeystroke` calls, §3). A reconnect tears down and rebuilds the `net.Conn` and the `Envelope`; it has no code path that could reach into `an.consoles` even if it wanted to.
- **What IS discarded and rebuilt — corrected (product-owner spec-completeness ruling, VP-082 v1.1):** ~~the claim in the prior revision of this note that `FrameAuthKey`'s HKDF derivation is "fresh per handshake" is SUPERSEDED and factually wrong~~. `FrameAuthKey` is **deterministic**, not handshake-salted: ARCH-04-admission-security.md's canonical key-derivation formula is `HKDF-Extract(salt=svtn_id, ikm=node_admission_pubkey)` → `HKDF-Expand(info="switchboard-frame-auth", 32)`, and BC-2.05.010 §"FrameAuthKey and NodeAddr are derived, not stored" independently confirms `FrameAuthKey` is "derived deterministically from (svtnID, pubkey)." There is no challenge-nonce or other handshake-specific entropy in the derivation — this is load-bearing, not an oversight: the router independently recomputes the same `FrameAuthKey` from the same `(svtnID, pubkey)` pair (BC-2.05.008's forwarding-table cache depends on this), and a nonce-salted key would silently break that cache on every reconnect. **Consequence:** across a reconnect of the *same node*, the entire `outerassembler.Envelope{SVTNID, SrcAddr, DstAddr, FrameAuthKey}` is expected to be **byte-identical** before and after — `SVTNID` is config-sourced and invariant (already correctly noted above), `SrcAddr` is a deterministic function of `(SVTNID, pubkey)` (VP-014: `DeriveNodeAddress` is deterministic), `DstAddr` is a fixed placeholder, and `FrameAuthKey` is now established as deterministic too. Only the `net.Conn` (the physical TCP socket) is actually discarded and rebuilt on reconnect; the `Envelope` derived from the rebuilt handshake recomputes to the same bytes, it does not diverge. See **VP-082 v1.1 Test Scenario 3** (corrected from a value-inequality negative-space check to a behavioral non-vacuity check: a genuinely second completed handshake — via `LocalAddr()` change and a handshake-completion counter or distinct `Challenge.Nonce` per attempt — with explicit equality assertions `envelopeAfter.SVTNID/SrcAddr/FrameAuthKey == envelopeBefore.*`, the correct expected outcome) for how this property is now proved without asserting a false inequality.

This design makes VP-082 **provable without any new "preserve" logic to write and therefore test for correctness** — the property holds because the architecture never wires a destructive path, which is exactly the shape VP-082's own Feasibility Assessment anticipates ("this VP's harness composes three pieces this same authoring burst's BC-2.04.009 explicitly leaves open... the property itself... does not depend on their resolution" — true, and now those three pieces have concrete shapes to compose against).

---

## 7. File structure

**New files:**

| File | Purpose |
|------|---------|
| `internal/nodeidentify/nodeidentify.go` | Extracted, exported codec: `EncodeNodeIdentify`, `EncodeChallenge`, `EncodeChallengeResponse`, `DecodeNodeIdentify`, `DecodeChallengeResponse`, new `DecodeChallenge`; size/msg-kind constants (§5) |
| `internal/nodeidentify/nodeidentify_test.go` | Round-trip tests for all six codec functions, including the new `DecodeChallenge` (encode-then-decode identity, malformed-payload rejection per each of the existing decoders' guard patterns) |
| `internal/accessdial/connector.go` | `Connector` type, `State`/`Handle`, `New`, `Start`, `Send`, `Stop`, `SetFrameCallback`, `admit` (handshake driver), dial/read/write loop (§4.2, mirrors `upstreamdial.dialLoop`) |
| `internal/accessdial/connector_test.go` | Unit tests: dial success/failure, admit success/each-handshake-failure-mode (VP-081), reconnect-preserves-injected-`*arq.ARQ`-pointer (VP-082 unit-level slice), `Send` single-writer safety under `go test -race` |
| `internal/accessdial/backoff.go` (or inline in `connector.go`) | Backoff constants/helpers — reuse `internal/upstreamdial`'s `BackoffBase`/`BackoffCap`/`BackoffJitterFraction` VALUES as the starting point (BC-2.04.009 Invariant 2 explicitly calls these "a directly reusable precedent, not a binding requirement") |
| `cmd/switchboard/accessdial_wire.go` | Policy/dispatch layer (mirrors `discovery_relay_wire.go`'s shape): `consoleBindingKey`/`consoleBindings` registry (§3b), `RegisterConsoleBinding`, the `FrameFn` implementation that discriminates ctl frames (forward-compat placeholder for `S-BL.RESYNC-FRAME` Layer 1's eventual `case 0x02:`) vs. upstream-keystroke frames and calls `an.SendKeystroke` |
| `cmd/switchboard/accessdial_wire_test.go` | Dispatch-table tests: known binding → `SendKeystroke` called correctly; binding miss → dropped, no crash, no `SendKeystroke` call |

**Touched files:**

| File | Change |
|------|--------|
| `internal/config/config.go` | New `RouterAddr`/`SVTNID` fields (§1); `Validate()` gains the two new checks |
| `internal/config/config_test.go` | New `Validate()` test cases for the two fields |
| `cmd/switchboard/node_identify_wire.go` | Five codec functions' bodies replaced with calls into `internal/nodeidentify` (or deleted outright in favor of direct call sites in `nodeIdentifyHandshake`); size/kind constants likewise sourced from the new package (§5) — **behavior-preserving**, existing tests unmodified |
| `cmd/switchboard/access.go` | `runAccess`/`runAccessWithConnector`: read `cfg.RouterAddr`/`cfg.SVTNID`, construct `internal/accessdial.Connector`, wire it into the `discovery.Config` LocalSVTNID/LocalNodeAddr fields (§1), start the connector, add the new wire-dispatch bridge (§2) alongside the existing `startFramesBridge`, wire `arqsend.Retransmitter`'s `Dispatch` to `connector.Send` (mirrors `internal/testenv/loopback.go`'s `downstreamARQ`+`Retransmitter` composition pattern, adapted from in-process to the live connector) |
| `cmd/switchboard/access_test.go` (and/or a new `router_access_connector_test.go`, mirroring `router_pe_connector_test.go`'s naming) | Integration test: `runAccessWithConnector` end-to-end against a router fixture (VP-081's happy path, driven through `access.go`'s real construction path, not just `internal/accessdial`'s own unit tests) |

**Task breakdown (12 tasks; a starting point for story-writer, not a final decomposition):**

1. Extract `internal/nodeidentify` from `cmd/switchboard/node_identify_wire.go` (§5) — pure refactor, existing tests green.
2. Add `DecodeChallenge` to `internal/nodeidentify` (§5) — new function, new tests.
3. Refactor `node_identify_wire.go` to call the extracted package — behavior-preserving, existing tests green.
4. `internal/config`: `RouterAddr`/`SVTNID` fields + `Validate()` checks (§1).
5. `internal/accessdial`: `State`/`Handle`/`Connector` skeleton, `New`, dial loop shape (mirrors `upstreamdial.dialLoop`, no handshake yet — TCP dial + backoff only).
6. `internal/accessdial`: `admit()` — the client-side three-message handshake (§4.2), using `internal/nodeidentify`.
7. `internal/accessdial`: read loop + `SetFrameCallback` + `Send`/single-writer guard (§2, §4.1).
8. `internal/accessdial`: reconnect preserves injected `*arq.ARQ` pointer and caller-owned state by construction (§6) — this is largely "don't write code that would break it," verified by VP-082's tests, not a feature to build.
9. `cmd/switchboard/accessdial_wire.go`: console-binding registry + `FrameFn` dispatcher (§3b).
10. `cmd/switchboard/access.go` wiring: construct/start the connector, wire the regular-send bridge (§2) and the retransmit `Dispatch` (both → `connector.Send`), wire `discovery.Config` fields (§1).
11. VP-081 integration test (router fixture + connector, happy path + every handshake-failure mode + private-key-absence audit).
12. VP-082 integration test (seeded `arq.ARQ` + seeded `session.AccessNode` + forced reconnect + before/after state-identity assertions).

**Points estimate** (carried from and refined against `S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §2's 8–13 range): **11–13 points** — the codec-extraction task (1–3) and the console-binding registry (9) are real, previously-unbudgeted additions this note's investigation surfaced; the connector/handshake/reconnect core (5–8, 10) tracks close to PE-CONNECTOR's own 8-point precedent.

---

## 8. AC → BC-2.04.009 postcondition mapping guidance (for story-writer)

| Candidate AC | BC-2.04.009 postcondition(s) | VP |
|---|---|---|
| AC-001 — dial + genuine NODE_IDENTIFY admission + first-write reaches *live* | PC-1, PC-2 (incl. 2a–2d), PC-3 | VP-081 property 1 |
| AC-002 — handshake failure (any BC-2.01.009 failure mode) is retried, never fatal; private key never on the wire | PC-2d, Invariant 5 | VP-081 properties 2, 3 |
| AC-003 — regular per-tick downstream emission reaches the wire through the connector, composed with `arq.ARQ.EnqueueSend`, sharing the single-writer `Send` path with retransmit dispatch | PC-4a, PC-4b, Invariant 3 | (no VP minted yet for this specific composition — see §10) |
| AC-004 — inbound frame read loop dispatches ctl frames and upstream-keystroke frames to their resolved targets; binding-miss degrades gracefully | PC-5 | (no VP minted yet — see §10, §11) |
| AC-005 — reconnect: three-state lifecycle, backoff with jitter, ARQ/session state survives unchanged | PC-6, PC-7, Invariant 2 | VP-082 |
| AC-006 — `internal/config` schema: `RouterAddr`/`SVTNID` fields, validated, wired into `runAccess` | Precondition 2 (discharges it) | (no VP — config validation is typically unit-tested directly, per `BC-2.09.003`'s own precedent, not VP-anchored) |

Story-writer should confirm whether AC-003/AC-004 warrant their own VPs (this note's §10 flags this as a residual question, not a decision made here) or are adequately covered by VP-081/VP-082's existing scope plus ordinary unit/integration test coverage.

---

## 9. Testability seams (Layer-1-style component tests, no live two-daemon stack required)

Mirroring the seam discipline `S-BL.RESYNC-FRAME-placement-note.md` §8 established for its own Layer 1:

- **`internal/nodeidentify`**: pure functions, trivially unit-testable, no fixture needed.
- **`internal/accessdial`**: unit-testable against a `net.Pipe()` pair or a loopback TCP fixture implementing the router-side of the handshake (a small test-only fixture — VP-081's own Proof Method table already specifies this: *"an in-process router fixture implementing BC-2.01.009's `onAccept` three-message handshake... driving the connector over a real `net.Conn` pair"*). No real router process, no real access daemon.
- **`cmd/switchboard/accessdial_wire.go`'s dispatch table**: unit-testable by calling `RegisterConsoleBinding` directly and injecting hand-built frames — mirrors `discovery_relay_wire_test.go`'s `buildRelayRouter` pattern exactly (populate state directly, skip the real handshake).
- **The `access.go` wiring itself**: integration-testable via `internal/testenv`'s existing `Env`/`RouterHandle` fixtures (`testenv.go`) once those fixtures are extended to accept a real `internal/accessdial.Connector` — flagged as likely-needed `testenv` extension work in §10, not committed to here.

---

## 10. Residual open questions (for story-writer / product-owner, not resolved by this note)

1. **Outbound `DstAddr` for multi-console fan-out.** This note's §4.2 `admit()` sketch leaves `Envelope.DstAddr` a placeholder. The current architecture has exactly one shared `ds` half-channel per access-node daemon (`access.go:197`, `chanID=1`) feeding potentially many attached consoles via in-process fan-out (`an.DeliverFrame`) — but a single outbound wire frame can only carry one `DstAddr`. Whether the wire-level design needs per-console outbound frames (requiring the daemon to assemble and send N copies, one per attached remote console, each with its own `DstAddr` and — critically — its own `ChanSeq`/ARQ tracking) or a router-side fan-out mechanism (the router expands one access-node-sent frame to N router-known consoles) is **not decided by this note or by BC-2.04.009**. This is likely a `S-BL.CONSOLE-CONNECTOR`-adjacent or session-bootstrap-era question, not fully in `S-BL.ACCESS-CONNECTOR`'s scope — flagged for story-writer to bound explicitly as a Non-Goal if it can't be resolved within this story's points budget.
2. **Handshake timeout value** for the client side (`c.handshakeTimeout` in §4.2) — `nodeIdentifyHandshakeTimeout=10s` is the router-side precedent (`node_identify_wire.go:56`); whether the client should use the identical value or a shorter one (the client controls when it starts waiting, unlike the router which is reactive) is an implementation-time call.
3. **Backoff constant values** — Invariant 2 already states `internal/upstreamdial`'s constants are "a directly reusable precedent, not a binding requirement"; story-writer/architect confirms the actual values.
4. **`FrameAuthKey` derivation specifics** — §4.2 flags that the HKDF derivation should include handshake-specific entropy so VP-082's Test Scenario 3 (envelope changes across reconnect) holds even under a byte-identical keypair/SVTN pairing across two consecutive handshakes; the exact derivation input set needs implementer-level confirmation against whatever `hmac.DeriveKey`-equivalent function is reused.
5. **Whether AC-003 (regular-send composition) and AC-004 (inbound dispatch) warrant their own VPs** beyond VP-081/VP-082's existing scope (§8) — this note surfaces the question, does not answer it.
6. **`internal/testenv` extension** to support a real `internal/accessdial.Connector` in its `Env`/`RouterHandle` fixtures, needed for any integration test that exercises `access.go`'s actual wiring rather than `internal/accessdial` in isolation — likely needed, not scoped here.

---

## 11. BC/VP adjustments flagged for product-owner (not made here)

1. **BC-2.04.009 — one new edge case recommended**, in the same table shape as the existing EC-001 through EC-005: *"EC-006 | Inbound upstream frame arrives with no resolvable `(SrcAddr, chan_id)` → `(ConsoleKey, sessionName)` binding | Frame is silently dropped; connector remains live; no error surfaced, no connection closed — mirrors BC-2.01.008 PC-4's unrecognized-opcode posture applied to an unrecognized sender."* This is new behavior this note's Open Item 3 resolution introduces that BC-2.04.009 v1.0 does not currently enumerate.
2. **No change required to BC-2.04.009's postconditions, invariants, or VP-081/VP-082** — every resolution in this note fills a gap those artifacts' own text already named as open, without altering what they require. Confirmed individually in §1, §2, §3.
3. **Possible new BC or BC-2.09.003 extension for `internal/config` validation of `RouterAddr`/`SVTNID`** (§1) — BC-2.09.003 ("Router Startup Fails Cleanly on Malformed Config") is titled router-specific but its `Validate()`-purity-contract pattern (ARCH-06) is the right home for these two new field's validation rules; whether that BC's scope is generic enough to cite directly or needs a small access-mode-specific addendum is product-owner's call, not resolved here.

---

## 12. Traceability summary

| Open item | Resolution | Confidence | Primary citations |
|---|---|---|---|
| 1: SVTN-ID / router-addr sourcing | New `Config.RouterAddr`/`Config.SVTNID` (hex string), mirroring `UpstreamRouter`/`AdmissionKeyFile` conventions | High | `internal/config/config.go:118-208`; `cmd/sbctl/admin.go` `SVTNID string` precedent; `internal/svtnmgmt/svtnmgmt.go:86-95` |
| 2: regular send-path composition | Mirrors `internal/testenv/loopback.go:220-291`'s `onDownstreamTick` (`EnqueueSend` + `Assemble` + single-writer dispatch), added alongside (not replacing) `startFramesBridge` | High — direct, already-merged, already-tested precedent | `access.go:564-585`; `loopback.go:220-291`; `arqsend/arqsend.go:64,147` |
| 3: inbound upstream delivery target | `session.AccessNode.SendKeystroke`, confirmed by its own doc comment as "the safe path for production callers"; new connector-owned binding registry, out-of-band from this story's scope to populate | Definitive for (a); High for (b)'s bounded scope | `internal/session/upstream.go:172-301`; `internal/testenv/loopback.go:313-327` |
| Codec reuse gap (net-new finding) | Extract to `internal/nodeidentify`, export, add missing `DecodeChallenge` | Definitive (grep-verified: `package main`, lowercase functions) | `cmd/switchboard/node_identify_wire.go:23,97-247` |
| State machine shape | Mirrors `internal/upstreamdial.dialLoop` almost one-to-one, handshake step substituted | High | `internal/upstreamdial/connector.go:318-459` |
| Reconnect/ARQ preservation (VP-082) | Survives by construction — connector never owns `*arq.ARQ`/`*session.AccessNode`, only references them | High | this note §6; `S-BL.RESYNC-FRAME-placement-note.md` §7.3 |
