---
artifact_id: S-BL.ACCESS-CONNECTOR
document_type: story
level: ops
story_id: S-BL.ACCESS-CONNECTOR
epic_id: "E-7"
title: "Access-node data-plane connector — dial, NODE_IDENTIFY admission, and live ARQ send-path wiring"
status: draft
producer: story-writer
timestamp: 2026-08-31T00:00:00Z
version: "1.1"
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: PE
points: "11-13"
estimated_points: "11-13"
estimated_days: null
target_module: internal/accessdial
inputs:
  - '.factory/decisions/S-BL.ACCESS-CONNECTOR-placement-note.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.009.md'
  - '.factory/specs/verification-properties/VP-081.md'
  - '.factory/specs/verification-properties/VP-082.md'
input-hash: "00a7ee9"
traces_to: '.factory/decisions/S-BL.ACCESS-CONNECTOR-placement-note.md'
assumption_validations: []   # no ASM-NNN identified as validated by this story
risk_mitigations: []   # no R-NNN identified as mitigated by this story
bc_traces:
  - BC-2.04.001   # existing, unedited — PC-3 ("admitted to an SVTN, or admission in progress") is the precondition this story's connector discharges
  - BC-2.09.003   # v2.3 — product-owner added PC-16 (RouterAddr host:port validation) / PC-17 (SVTNID hex/16-byte validation) as the formal validation home for this story's new config fields; also the PE-CONNECTOR PC-9 connect-half precedent this story's dial-loop pattern parallels
  - BC-2.04.009   # v1.1 — the commissioning BC, now authored in canonical form (product-owner). Every AC below traces to a specific postcondition/invariant/edge-case clause.
behavioral_contracts:
  - BC-2.04.001
  - BC-2.09.003
  - BC-2.04.009
vp_traces:
  - VP-081        # dial → admit → first-frame-write round trip; admission-failure-is-retryable; private key never on wire
  - VP-082        # v1.1 — reconnect preserves ARQ in-flight SendBuffer + session/console-authorization state; Scenario 3 corrected to Envelope byte-identity (FrameAuthKey is deterministic, no handshake entropy)
verification_properties:
  - VP-081
  - VP-082
subsystems: [session-access, admission-security]
architecture_modules:
  - internal/accessdial      # NEW package — connector state machine (canonical name per BC-2.04.009 frontmatter)
  - internal/nodeidentify    # NEW package — pure NODE_IDENTIFY codec, extracted from cmd/switchboard so both router and client sides can import it
  - cmd/switchboard          # node_identify_wire.go refactor; access.go wiring; new accessdial_wire.go dispatch site
  - internal/config          # new RouterAddr/SVTNID fields
  - internal/session         # SendKeystroke is the resolved inbound-delivery target; read-only consumer, no code change to internal/session itself
  - internal/arq             # EnqueueSend/OnAck composition for the regular-tick path; caller-owned *arq.ARQ passed by reference
  - internal/arqsend         # Retransmitter.Dispatch becomes accessdial.Connector.Send
  - internal/outerassembler  # Assemble — wire-frame assembly for the regular-tick send path
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []   # buildable now — no unmerged prerequisite; the server-side NODE_IDENTIFY primitives it needs already ship (node_identify_wire.go), though this story extracts and refactors them (AC-012) in a behavior-preserving way
blocks: []   # NOT a hard frontmatter dependency edge on S-BL.RESYNC-FRAME (that story predates this one and does not declare depends_on here) — this story is a prerequisite for S-BL.RESYNC-FRAME's "Layer 2" scope specifically (real end-to-end round trip), not its independent "Layer 1" scope. See S-BL.DATAPLANE-CONNECTOR-scoping-note.md §4 and S-BL.RESYNC-FRAME's "Architect Elaboration (2026-08-31)" note, which DOES declare S-BL.ACCESS-CONNECTOR in its own depends_on (Layer-2-scoped).
inputDocuments:
  - '.factory/decisions/S-BL.ACCESS-CONNECTOR-placement-note.md'          # v1.0 — BINDING. Full architect resolution of BC-2.04.009's three flagged open items + the net-new codec-extraction finding. This story transcribes it; where the two diverge, the note governs.
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.009.md'            # v1.1 — commissioning BC; EC-006 already added by product-owner
  - '.factory/specs/verification-properties/VP-081.md'
  - '.factory/specs/verification-properties/VP-082.md'
  - '.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md'
  - '.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md'
  - '.factory/stories/S-7.04-FU-PE-CONNECTOR.md'
  - '.factory/decisions/S-7.04-FU-PE-CONNECTOR-placement-note.md'
acceptance_criteria_count: 13
backlog_origin:
  source: S-BL.DATAPLANE-CONNECTOR-scoping-note
  adr_disposition: "N/A — not an ADR-005 lineage item; this is a newly-named Layer-2 prerequisite for S-BL.RESYNC-FRAME, surfaced by the architect's RESYNC placement note §3/§8 and elaborated in the connector scoping note"
  drift_items_consumed: []
  notes: >
    Named and bounded by .factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md §2,
    at the human's request, to make S-BL.RESYNC-FRAME-placement-note.md §3/§8's finding
    durable and tracked: no production or test-harness code path in this repository
    performs real wire-level session-data relay between an access node and a router.

    Elaborated 2026-08-31 from the v0.1-backlog-stub into a full decomposition,
    consuming .factory/decisions/S-BL.ACCESS-CONNECTOR-placement-note.md v1.0 (architect
    resolution of all three open architecture items BC-2.04.009 v1.0 flagged, plus a
    net-new NODE_IDENTIFY-codec-extraction finding) and BC-2.04.009 v1.1 (product-owner
    authored the BC in canonical form, added EC-006 per the placement note's §11 item 1
    recommendation). 13 ACs, 11-13 points, 12 implementation tasks + 1 CHANGELOG task.
    status remains draft (not ready) pending Step-4.5 adversarial convergence.
---

# S-BL.ACCESS-CONNECTOR: Access-Node Data-Plane Connector

> **Status note:** Elaborated 2026-08-31 from the v0.1-backlog-stub into a full,
> Step-4.5-ready decomposition. `status: draft` is correct per the S-7.01 Spec-First
> Gate's own text — BC-2.04.009 exists in canonical form (v1.1) and every AC below
> traces to it, but this story has not yet passed Step-4.5 adversarial convergence, so
> it does not advance to `ready` in this burst. Several implementation-time values are
> marked **PROVISIONAL** below (handshake timeout, backoff constants, `DstAddr`
> placeholder) — Step-4.5 or the implementer confirms or revises them; they are not
> blocking. **v1.1 (2026-08-31):** product-owner resolved two of the four flagged
> completeness items. `FrameAuthKey` derivation entropy is now **RESOLVED** —
> deterministic by architecture (HKDF from `svtn_id`+`pubkey` only, no handshake
> entropy), which corrects AC-010's Envelope assertion from "differs across reconnect"
> to "byte-identical across reconnect" (VP-082 v1.1 Scenario 3). BC-2.09.003 (v2.3) now
> carries PC-16/PC-17 as the formal validation home for `RouterAddr`/`SVTNID`. The
> proposed new error-taxonomy codes are corrected from `E-CFG-014`/`E-CFG-015`
> (already allocated to shipped admission stories, BC-2.09.003 v2.1/2.2 — collision
> avoided) to `E-CFG-018`/`E-CFG-019`. Two items remain **FLAGGED FOR PRODUCT-OWNER**
> (the `E-CFG-018`/`E-CFG-019` taxonomy-file addition itself, and whether AC-006/AC-007
> warrant their own VPs) — see Residual Open Questions.

## Narrative

- **As an** access-mode switchboard daemon
- **I want to** dial its configured router, complete a genuine client-side
  NODE_IDENTIFY admission handshake, and maintain a live, reconnecting TCP data-plane
  connection that carries both regular per-tick downstream emission and ARQ
  retransmissions, plus an inbound read loop delivering router-relayed frames
- **So that** `internal/arq`'s downstream `SendBuffer` and `arqsend.Retransmitter`
  dispatch real wire frames instead of the current in-process-only path, and dependent
  protocol work (`S-BL.RESYNC-FRAME`'s Layer 2, and any genuine end-to-end verification
  that session data flows over the wire) has a live, admitted connection to exercise

## Context

`S-BL.RESYNC-FRAME-placement-note.md` (§3, §8) found a load-bearing topology gap:
`internal/arq`'s downstream `SendBuffer` lives at the access node (ARCH-03
§"Downstream ARQ"), but `cmd/switchboard`'s access-mode daemon
(`runAccess`/`runAccessWithConnector` in `access.go`) had zero data-plane network code
— no `net.Dial`, no `outerassembler.Assemble`, no wire-frame read/write.
`.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` (§2) named and bounded
this gap as a proper backlog story so it could be tracked instead of remaining an
unnamed finding buried in a design note; `BC-2.04.009` (v1.0, product-owner) then
commissioned `internal/accessdial` to close it, but explicitly declined to resolve
three architecture questions — SVTN-ID/router-address sourcing, the regular-send-path
composition, and the inbound upstream-keystroke delivery target — flagging all three
for architect resolution rather than inventing designs.

`.factory/decisions/S-BL.ACCESS-CONNECTOR-placement-note.md` (v1.0, architect) resolves
all three, each grounded in code already in the repository, plus surfaces one net-new
finding not previously flagged anywhere: the NODE_IDENTIFY wire-codec functions this
story needs to reuse (`encodeNodeIdentify`, `encodeChallengeResponse`, and a new
`decodeChallenge`) are unexported, `package main`-scoped functions in
`cmd/switchboard/node_identify_wire.go` — `internal/accessdial`, a new package under
`internal/`, cannot import them as-is. A small, behavior-preserving extraction into a
new `internal/nodeidentify` package is required and is part of this story's scope
(AC-012), not deferred. Product-owner then authored BC-2.04.009 v1.1, adding EC-006
(inbound frame with no resolvable console binding is silently dropped) per the
placement note's own §11 recommendation.

This story is the direct, real-admission successor to `S-7.04-FU-PE-CONNECTOR`'s
shipped router-to-router connector (`internal/upstreamdial`), substituting genuine
identity-bound admission for that story's deferred zero-envelope bootstrap. It is the
**recommended-first** of the two `S-BL.RESYNC-FRAME` "Layer 2" connector prerequisites
(the other being the design-blocked `S-BL.CONSOLE-CONNECTOR`) — highest confidence, and
most directly unblocks real end-to-end verification of whether session data actually
flows over the wire.

**Story-writer's job here is transcription of the placement note's binding designs,
not re-derivation.** Where this story and the placement note appear to diverge, the
note governs.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|-------------|
| Access node discharges "admitted to an SVTN, or admission in progress" | BC-2.04.001 PC-3 | BC-2.04.001 (existing, unedited) | TO DISCHARGE — this story's connector reaching `StateLive` is precisely that discharge (AC-004, AC-008) |
| `RouterAddr`/`SVTNID` config-field validation | BC-2.09.003 PC-16 (`router_addr` host:port), PC-17 (`svtn_id` hex/16-byte) | BC-2.09.003 v2.3 (product-owner added PC-16/PC-17 as the formal validation home, superseding this story's v1.0 pattern-precedent-only framing) | TO DISCHARGE — AC-001's `RouterAddr`/`SVTNID` validation now directly implements PC-16/PC-17, reusing `Config.Validate()`'s existing host:port (`validateHostPort`, PC-6 precedent for `UpstreamRouter.Addr`) and structured-field-error conventions |
| Dial | BC-2.04.009 PC-1 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-001 (config-sourced dial address), AC-004 (dial as step (i) of the three-step "established" definition) |
| Client-side NODE_IDENTIFY handshake, genuine admission | BC-2.04.009 PC-2 (a-d) | BC-2.04.009 v1.1 | TO DISCHARGE — AC-002 (handshake mechanics a-c), AC-003 (DI-002 private-key non-transit, PC-2d) |
| Connection-established three-step definition | BC-2.04.009 PC-3 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-004 |
| Live ARQ send-path wiring — retransmit dispatch (4a) and regular per-tick emission (4b) | BC-2.04.009 PC-4 | BC-2.04.009 v1.1, resolved by placement note §2 (Open Item 2) | TO DISCHARGE — AC-005 (4a), AC-006 (4b) |
| Frame read loop (inbound) — ctl frames + upstream keystroke delivery target | BC-2.04.009 PC-5 | BC-2.04.009 v1.1, resolved by placement note §3 (Open Item 3); EC-006 added v1.1 | TO DISCHARGE — AC-007 |
| Per-connection lifecycle and reconnect — three states, backoff | BC-2.04.009 PC-6, Invariant 2 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-008 |
| ARQ/session state survives reconnect | BC-2.04.009 PC-7 | BC-2.04.009 v1.1; VP-082 | TO DISCHARGE — AC-010 |
| DI-002 — private keys never transit the network | BC-2.04.009 Invariant 1 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-003 |
| Reconnect backoff with jitter, not a tight redial loop | BC-2.04.009 Invariant 2 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-008 |
| Single-writer / externally-synchronized wire-write path | BC-2.04.009 Invariant 3 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-005, AC-006, AC-013 |
| DI-004 — no direct node-to-node communication | BC-2.04.009 Invariant 4 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-011 |
| Admission failure is wire-indistinguishable from ordinary network failure; both retryable, neither fatal | BC-2.04.009 Invariant 5 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-009 |
| EC-001 through EC-005 (router unreachable; handshake-phase closure; mid-session loss; flapping; SVTN-ID/router-addr unresolvable) | BC-2.04.009 v1.1 | BC-2.04.009 v1.1 | TO DISCHARGE — AC-008 (EC-001, EC-004), AC-009 (EC-002), AC-010 (EC-003), AC-001 (EC-005, via config validation) |
| EC-006 — inbound frame with no resolvable console binding is silently dropped | BC-2.04.009 v1.1 (added by product-owner per placement note §11 item 1) | BC-2.04.009 v1.1 | TO DISCHARGE — AC-007 |
| Dial → admit → first-frame-write round trip; handshake failure retryable, not fatal; private key never on wire | VP-081 | VP-081 v1.0 | TO DISCHARGE — AC-004 (property 1), AC-009 (property 2), AC-003 (property 3) |
| Reconnect preserves ARQ in-flight `SendBuffer` state and session/console-authorization state; Envelope (`SVTNID`/`SrcAddr`/`FrameAuthKey`) is byte-identical across reconnect (Scenario 3, corrected) | VP-082 | VP-082 v1.1 | TO DISCHARGE — AC-010 |
| NODE_IDENTIFY codec functions must be extracted into an importable package (net-new finding, not previously anchored anywhere) | Placement note §5 | S-BL.ACCESS-CONNECTOR-placement-note.md v1.0 | TO DISCHARGE — AC-012 |

---

## Design Constraints

The following subsections transcribe the placement note's binding decisions. They are
not re-derived here; where a code sketch is reproduced, it is the note's sketch, not a
new one.

### Config-Sourced Dial: `RouterAddr` / `SVTNID` (resolves BC-2.04.009 Precondition 2 / Open Item 1)

**Binding (per placement note §1).** New `internal/config.Config` fields, mirroring the
existing `UpstreamRouter`/`AdmissionKeyFile` conventions exactly, added near
`AdmissionKeyFile` (`config.go:153-165`):

```go
// RouterAddr is the TCP address of the router this access-mode daemon dials.
// Access-mode only. Required for access mode (empty/malformed is invalid —
// E-CFG-018, NEW, see Residual Open Questions). Validated as host:port by
// Config.Validate() per BC-2.09.003 PC-16 (v2.3, product-owner) — the same
// validateHostPort helper BC-2.09.003 PC-6 already applies to UpstreamRouter.Addr.
RouterAddr string `yaml:"router_addr"`

// SVTNID is the hex-encoded (32 lowercase hex chars, no separators) 16-byte SVTN
// identifier this access node is provisioned under. Access-mode only. Required for
// access mode. Config.Validate() checks the string decodes via
// encoding/hex.DecodeString to exactly 16 bytes (E-CFG-019, NEW, see Residual Open
// Questions), per BC-2.09.003 PC-17 (v2.3, product-owner) — it does NOT perform
// admission I/O; matches ARCH-06's Config-purity contract (Validate() never opens a
// socket or file).
SVTNID string `yaml:"svtn_id"`
```

Both values are hand-provisioned by the operator (or a deployment script) at the same
time `AdmissionKeyFile` is provisioned — mirroring how `UpstreamRouters` is
hand-provisioned for PE-mode routers today, and how `cmd/sbctl/admin.go`'s own
`SVTNID string` RPC fields already carry the 16-byte ID as a string once minted. This
does **not** add a live control-plane query at access-daemon startup.

`access.go` wiring: `runAccess`/`runAccessWithConnector` read `cfg.RouterAddr` and
`cfg.SVTNID` (hex-decoded once, at startup, alongside the existing `admissionKeyPath`
resolution block) and pass both into the new `internal/accessdial.Connector`
constructor. As a direct side effect, this is also the first real, non-test-only value
ever assigned to `discovery.Config.LocalSVTNID` (`access.go`'s existing discovery
wiring names this exact gap in its own comment: "populated by
S-BL.NODE-IDENTIFY-WIRE" — a forward obligation that remained open until now).
`discovery.Config.LocalNodeAddr` is populated from `frame.DeriveNodeAddress(svtnID,
admissionPubKey)`, the same derivation `nodeIdentifyHandshake` already uses
router-side.

### Client-Side NODE_IDENTIFY Handshake (resolves BC-2.04.009 Postcondition 2)

**Binding (per placement note §4.2, mirroring `internal/upstreamdial.dialLoop`'s
already-shipped shape, `connector.go:318-459`).** Once dialed, `Connector.admit(conn)`
drives the mirror-image, client-side half of BC-2.01.009's three-message exchange:

```go
func (c *Connector) admit(conn net.Conn) (outerassembler.Envelope, error) {
    conn.SetDeadline(time.Now().Add(c.handshakeTimeout)) // PROVISIONAL 10s — see
    defer conn.SetDeadline(time.Time{})                   // Residual Open Questions

    pubkey := c.keypair.Public().(ed25519.PublicKey)
    if _, err := conn.Write(nodeidentify.EncodeNodeIdentify(c.svtnID, pubkey)); err != nil {
        return outerassembler.Envelope{}, err
    }

    hdr, payload, err := frame.ReadOuterFrame(conn)
    if err != nil { return outerassembler.Envelope{}, err }
    challenge, err := nodeidentify.DecodeChallenge(payload)  // NEW function — AC-012
    if err != nil { return outerassembler.Envelope{}, err }

    sig := ed25519.Sign(c.keypair, challenge.Nonce[:])       // DI-002: only the
                                                                // signature crosses
                                                                // the wire next
    resp := admission.ChallengeResponse{NonceSig: sig}
    if _, err := conn.Write(nodeidentify.EncodeChallengeResponse(c.svtnID, resp)); err != nil {
        return outerassembler.Envelope{}, err
    }

    // BC-2.01.009 PC-10: the router returns no explicit success/failure message.
    // "Admission succeeded" is inferred by the connection remaining open long
    // enough for the connector's own first live write to succeed.
    nodeAddr := frame.DeriveNodeAddress(c.svtnID, pubkey)
    return outerassembler.Envelope{
        SVTNID:  c.svtnID,
        SrcAddr: nodeAddr,
        // DstAddr: PROVISIONAL zero value — see Non-Goals / Residual Open Questions.
        FrameAuthKey: deriveFrameAuthKey(c.svtnID, pubkey), // HKDF(svtn_id, pubkey)
                                                              // ONLY — deterministic,
                                                              // NO handshake entropy
                                                              // (RESOLVED v1.1, VP-082
                                                              // Scenario 3 — the router
                                                              // independently recomputes
                                                              // the identical key)
    }, nil
}
```

Genuine, identity-bound admission — not a placeholder. The resulting
`Envelope{SVTNID, SrcAddr, DstAddr, FrameAuthKey}` is real, so
`routing.LookupInterface`/`BindInterface` (keyed on `(svtnID, nodeAddr)`) resolve
correctly.

### Connector State Machine (resolves BC-2.04.009 Postconditions 3, 6; Invariants 2, 5)

**Binding (per placement note §4.1-4.2).** Public surface, mirroring
`internal/upstreamdial.Connector`'s `Handle` interface shape:

```go
package accessdial

type State int

const (
    StateDialing State = iota  // no live TCP connection; retrying with backoff
    StateAdmitting              // TCP connected; NODE_IDENTIFY exchange in progress
    StateLive                   // admitted; Envelope populated; safe to Send
)

type Handle interface {
    State() State                        // atomic.Load — no mutex (go.md rule 12)
    Envelope() outerassembler.Envelope    // value type; zero value if not StateLive
    Send(wire []byte) error               // the ONE exported write path
    Stop()                                 // cancels dial/handshake/live goroutines, waits for exit
}

func (c *Connector) SetFrameCallback(fn FrameFn)   // set once before Start() — set-once
type FrameFn func(hdr frame.OuterHeader, payload []byte) error  // pre-launch ordering,
                                                                   // mirrors upstreamdial's
                                                                   // own contract

func New(routerAddr string, svtnID [16]byte, keypair ed25519.PrivateKey, arqHandle *arq.ARQ) *Connector
func (c *Connector) Start()
```

State transitions:

```
StateDialing  --net.Dial succeeds-->  StateAdmitting
StateAdmitting --NodeIdentify sent, Challenge received+decoded, ChallengeResponse
                  sent, router does not close the connection--> StateLive
StateAdmitting --any read/write error, or router closes the connection
                  (BC-2.01.009 PC-10: no explicit rejection, only closure)--> StateDialing (backoff)
StateLive --read/write error, or explicit Stop()--> StateDialing (backoff) [or terminal, on Stop()]
```

No state ever carries session data except `StateLive` (BC-2.04.009 PC-6, mirroring
BC-2.01.009 PC-8's router-side "fully bound or closed, no unbound-open state"
invariant). Backoff-with-jitter (Invariant 2) uses **PROVISIONAL**
`BackoffBase=500ms`, `BackoffCap=30s`, `BackoffJitterFraction=0.25` — reused verbatim
from `internal/upstreamdial`; Invariant 2's own text states these are "a directly
reusable precedent, not a binding requirement," so implementer/Step-4.5 confirms the
actual values. No give-up ceiling — reconnect attempts continue indefinitely (EC-004),
matching `internal/upstreamdial`'s unbounded-retry precedent.

### Live ARQ Send-Path Wiring (resolves BC-2.04.009 Postcondition 4; Invariant 3)

**Binding (per placement note §2).** Two distinct call sites share the ONE exported
write path, `Connector.Send(wire []byte) error`:

1. **Retransmit dispatch (PC-4a):** `Connector.Send` is wired as the `Dispatch`
   callback supplied to `arqsend.Retransmitter.Retransmit`.
2. **Regular per-tick emission (PC-4b), mirroring `internal/testenv/loopback.go`'s
   already-merged, already-tested `onDownstreamTick` composition** (`loopback.go:220-291`,
   `S-BL.LOOPBACK-FULLSTACK`):

```go
chanSeq := f.ChanSeq   // captured before any transformation
if f.FrameType == halfchannel.FrameTypeData {
    arqHandle.EnqueueSend(chanSeq, f.Payload, time.Now().UTC())
}
// Unlike a test-only loopback harness, a REAL wire connection assembles and sends
// EVERY tick, including empty ones — BC-2.01.002 DI-008 is a wire-level liveness
// invariant, not scoped to any one story's harness.
cf := halfchannel.ChannelFrame{ChanID: f.ChanID, ChanSeq: chanSeq, FrameType: f.FrameType, Flags: f.Flags, Payload: f.Payload}
var zeroSACK [outerassembler.SACKBitmapSize]byte // this direction does not originate SACK
wire, err := outerassembler.Assemble(cf, zeroSACK, connector.Envelope())
if err != nil { /* log + continue — do not crash the bridge goroutine on one bad frame */ }
if err := connector.Send(wire); err != nil { /* log — Send's own retry/backoff posture is the connector's concern */ }
// Existing in-process fan-out is UNCHANGED, additive: an.DeliverFrame(...) stays as-is.
```

Added as a second function alongside the existing `startFramesBridge`
(`cmd/switchboard/access.go:564-585`) — e.g. `startWireBridge(an *session.AccessNode,
framesCh <-chan halfchannel.ChannelFrame, arqHandle *arq.ARQ, connector
*accessdial.Connector)` — or inlined into it; either shape satisfies PC-4b.
`startFramesBridge` itself stays independently testable exactly as it is today (no
`internal/accessdial` dependency).

**Single-writer requirement (Invariant 3) satisfied by construction:** `Send` is the
**one** exported write entry point on `*accessdial.Connector`. Both the regular-tick
path and `arqsend.Retransmitter`'s `Dispatch` callback call it — never `net.Conn.Write`
directly from outside the connector package. `Send` holds a mutex (or routes through a
single internal writer goroutine via a channel — implementer's choice) guarding the
live `net.Conn`.

### Inbound Frame Read Loop and Console Binding Registry (resolves BC-2.04.009 Postcondition 5; EC-006)

**Binding (per placement note §3).** Resolution part (a) — definitive: the inbound
upstream-keystroke delivery target is `session.AccessNode.SendKeystroke(key
ConsoleKey, sessionName string, payload []byte) error` — confirmed by
`internal/session/upstream.go`'s own doc comment ("the safe path for production
callers"; `DeliverFrame` is confirmed fan-out-only, the opposite direction). This
requires **no new method** on `internal/session`.

Resolution part (b) — the genuinely open sub-problem, bounded to this story's scope: a
narrow, explicit registry resolving `(SrcAddr, chan_id)` → `(ConsoleKey, sessionName)`,
owned by `cmd/switchboard` (mirroring `discovery_relay_wire.go`'s shape — policy/dispatch
code lives in `cmd/switchboard`, not in the generic `internal/accessdial` package):

```go
// cmd/switchboard/accessdial_wire.go
type consoleBindingKey struct {
    srcAddr [8]byte
    chanID  uint32
}

// consoleBindings is populated by whatever FUTURE mechanism resolves a console's
// wire-level attach to a (ConsoleKey, sessionName) pair — that mechanism is
// explicitly OUT of this story's scope (S-BL.CONSOLE-CONNECTOR or a
// session-bootstrap follow-on). This story defines the registry and its
// lookup/miss behavior; it does NOT populate it in production.
var consoleBindings sync.Map // consoleBindingKey -> (session.ConsoleKey, sessionName string)

func RegisterConsoleBinding(srcAddr [8]byte, chanID uint32, key session.ConsoleKey, sessionName string)
```

**On a binding miss (EC-006, added BC-2.04.009 v1.1):** the read-loop dispatcher logs
and drops the frame — no crash, no connection close, not treated as a protocol
violation. Mirrors BC-2.01.008 Postcondition 4's "unknown/forward-compat, silent-ignore"
posture, applied to an unrecognized-*sender* inbound frame rather than an unrecognized
opcode. **This is the expected common path in this story's own scope** — the binding
registry is populated only by a later story.

Router-relayed ctl frames dispatch to their respective handlers (a forward-compat
placeholder for `S-BL.RESYNC-FRAME`'s eventual `control_type = 0x02` dispatch — this
story does NOT implement RESYNC dispatch itself).

**Testability:** the narrow scope makes the read-loop dispatch unit-testable today,
without any console-attach wire protocol existing — a test calls
`RegisterConsoleBinding` directly, injects a hand-built inbound frame, and asserts
`SendKeystroke` was called with the right arguments (or, for the miss case, that it was
not called and no crash occurred) — mirroring `discovery_relay_wire_test.go`'s
`buildRelayRouter` pattern.

### Reconnect and State Preservation (resolves BC-2.04.009 Postcondition 7; VP-082)

**Binding (per placement note §6).** What survives a reconnect, and why, by
construction rather than by any explicit "preserve" step:

- **`*arq.ARQ` (the `SendBuffer`):** owned by `cmd/switchboard`/`access.go`,
  constructed once at daemon startup and passed **by reference** into
  `internal/accessdial.New(...)` — the connector's `Stop`/reconnect machinery never
  holds, closes, or reconstructs this handle; it only reads from it. State survives
  because nothing in the reconnect path ever touches the pointer.
- **`session.AccessNode` (console attachments, `ConsoleSet`, authorization state):**
  same argument — `access.go` constructs `an *session.AccessNode` once at startup
  (unchanged by this BC) and the connector never receives ownership of it, only a
  narrow callback surface (`SendKeystroke` calls). A reconnect tears down and rebuilds
  only the `net.Conn` and the `Envelope`; it has no code path that could reach into
  `an.consoles`.
- **What IS discarded and rebuilt:** exactly the `net.Conn` and the
  `outerassembler.Envelope` *struct instance* derived from the OLD admission
  handshake. The field VALUES within the rebuilt `Envelope` are not necessarily
  different, though: `SVTNID` is invariant across reconnects (config-sourced, not
  handshake-derived); `SrcAddr` is recomputed via `frame.DeriveNodeAddress(svtnID,
  pubkey)` each `admit()` call and is therefore byte-identical across reconnects for
  the same node (a deterministic function of two invariant inputs); `FrameAuthKey` is
  likewise deterministic — HKDF derived from `svtn_id`+`pubkey` **only, with no
  handshake-specific entropy** (RESOLVED v1.1, VP-082 Scenario 3 — the router
  independently recomputes the identical key from the same static inputs, so a
  nonce-salted key would break its forwarding-table cache on reconnect). Only
  `DstAddr` remains an open placeholder (PROVISIONAL zero value, see Non-Goals),
  independent of this correction.

This makes VP-082 provable without any new "preserve" logic to write — the property
holds because the architecture never wires a destructive path.

### NODE_IDENTIFY Codec Extraction — `internal/nodeidentify` (net-new finding; resolves AC-012)

**Binding (per placement note §5).** `cmd/switchboard/node_identify_wire.go` is
`package main`; its five codec functions (`encodeNodeIdentify`, `encodeChallenge`,
`encodeChallengeResponse`, `decodeNodeIdentify`, `decodeChallengeResponse`) are
unexported. `internal/accessdial` — a new package under `internal/` — cannot import
`cmd/switchboard` (backwards dependency direction) and could not call these functions
even if it could. Extract all five, the three payload-size constants, the
`nodeIdentifyControlType` constant, and the three `msgKind*` constants into a new pure
package, `internal/nodeidentify` (imports only `internal/frame`, `internal/admission`
for the `Challenge`/`ChallengeResponse` types, and stdlib `crypto/ed25519`). Export all
five (capitalized) and add the sixth, currently-missing function:

```go
// DecodeChallenge decodes the 100-byte Challenge payload — the client-side
// counterpart to DecodeChallengeResponse. No such function exists in the codebase
// today. Mirrors DecodeChallengeResponse's structure exactly.
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

`cmd/switchboard/node_identify_wire.go` is refactored to call the extracted functions
— **behavior-preserving**: identical wire bytes in, identical wire bytes out, so
`S-BL.NODE-IDENTIFY-WIRE`'s existing router-side tests continue to pass unmodified.

**Package placement:** `internal/nodeidentify` sits below both `cmd/switchboard` and
`internal/accessdial` in the import graph (both import it; it imports neither). Exact
numeric ARCH-08 §6.5 position is PROVISIONAL/TBD at implementation — see Architecture
Compliance Rules.

### Forbidden Edges (inherited unchanged from `S-7.04-FU-PE-CONNECTOR`'s own placement-note ruling)

- `internal/accessdial` → `internal/routing` — **forbidden**. The connector never
  makes a routing decision; it dials one configured address (AC-011).
- `internal/accessdial` → `internal/testenv` — **forbidden**. Test-composition root;
  never imported by production code.
- **Permitted, narrow:** `internal/accessdial` → `{internal/frame,
  internal/outerassembler, internal/arq, internal/nodeidentify, internal/admission}`
  (the last two for handshake codec + `Challenge`/`ChallengeResponse` types only — no
  admission *verification* logic, which stays router-side).

---

## Acceptance Criteria

### AC-001 (traces to BC-2.04.009 Precondition 2 discharge, Postcondition 1; EC-005)

`internal/config.Config` gains `RouterAddr string` (`yaml:"router_addr"`) and `SVTNID
string` (`yaml:"svtn_id"`) fields, access-mode-scoped. `Config.Validate()` validates
`RouterAddr` as host:port per BC-2.09.003 PC-16 (v2.3 — reusing the existing
`validateHostPort` helper, mirroring `UpstreamRouter.Addr` per BC-2.09.003 PC-6) and
`SVTNID` as exactly 32 lowercase hex characters decoding to 16 bytes via
`encoding/hex.DecodeString`, per BC-2.09.003 PC-17 (v2.3) — both required for
access mode; empty or malformed values are rejected at config-validation time, before
any dial attempt (discharges EC-005: "the access node's SVTN ID or router address is
not resolvable at startup" — the connector cannot begin dialing until config
validation passes, and validation failure is a clean startup abort, not a live network
error). `access.go`'s `runAccess`/`runAccessWithConnector` reads `cfg.RouterAddr`/
`cfg.SVTNID` (hex-decoded once) and constructs `internal/accessdial.New(...)`; the
same values populate `discovery.Config.LocalSVTNID`/`LocalNodeAddr` (previously
zero-valued). On startup, the connector performs `net.Dial("tcp", cfg.RouterAddr)`
against the configured address (Postcondition 1).

**Test:** `TestConfig_Validate_RouterAddrSVTNID` (table-driven: empty `RouterAddr`,
malformed `RouterAddr` host:port, empty `SVTNID`, non-hex `SVTNID`, wrong-length
`SVTNID`, valid case). `TestRunAccess_ConstructsConnectorFromConfig` (confirms
`access.go` wiring reads both fields and constructs `internal/accessdial.Connector`;
confirms `discovery.Config.LocalSVTNID`/`LocalNodeAddr` populated).

### AC-002 (traces to BC-2.04.009 Postcondition 2, sub-clauses a-c)

Once dialed, `Connector.admit(conn)` drives the client-side three-message NODE_IDENTIFY
exchange: (a) sends `NodeIdentify` via `nodeidentify.EncodeNodeIdentify(svtnID,
pubkey)`; (b) reads the outer frame via `frame.ReadOuterFrame` and decodes the
`Challenge` via the new `nodeidentify.DecodeChallenge` (AC-012); (c) signs the received
nonce (`ed25519.Sign`) and sends `ChallengeResponse` via
`nodeidentify.EncodeChallengeResponse(svtnID, ChallengeResponse{NonceSig: sig})`.
Genuine, identity-bound admission — not a placeholder — interoperating with the
router-side handshake BC-2.01.009 already implements.

**Test:** `TestAccessdial_Admit_HappyPath` — drives `admit()` against an in-process
router fixture implementing BC-2.01.009's `onAccept` three-message handshake over a
`net.Pipe`/loopback TCP pair; asserts `NodeIdentify` well-formed, `Challenge` decoded,
`ChallengeResponse`'s `NonceSig` verifies, the fixture calls its `BindInterface`
equivalent.

### AC-003 (traces to BC-2.04.009 Postcondition 2d, Invariant 1; VP-081 property 3)

Across the full handshake and every failure path, the admission private key is never
serialized into any wire message, log line, or diagnostic output — only the public key
(`NodeIdentify`) and the Ed25519 signature bytes (`ChallengeResponse`) cross the wire
(DI-002).

**Test:** `TestAccessdial_PrivateKeyNeverOnWire` (VP-081 property 3) — captures every
byte sequence written to the connection and every log line emitted across the happy
path and all handshake-failure scenarios (AC-009); asserts none contains the fixed,
greppable private-key byte value — reuses the VP-007/VP-057 audit shape.

### AC-004 (traces to BC-2.04.009 Postcondition 3; VP-081 property 1)

The connection is "established" — the connector transitions to `StateLive` — only
once, in order: (i) `net.Dial` succeeds; (ii) the three-message handshake completes
without the router closing the connection; (iii) the connector performs its first
successful frame write on the connection. No step is skippable or reorderable;
`net.Dial` success alone does not constitute `StateLive`.

**Test:** `TestAccessdial_ConnectionEstablished_ThreeStepOrder` — asserts `State()`
reports `StateDialing` until Dial succeeds, `StateAdmitting` until the handshake
completes, and `StateLive` only after the first write succeeds, against a fixture that
pauses at each phase.

### AC-005 (traces to BC-2.04.009 Postcondition 4a; BC-2.02.005 Postconditions 3/5)

Once `StateLive`, `Connector.Send` is wired as the `Dispatch` callback supplied to
`arqsend.Retransmitter.Retransmit` for downstream ARQ retransmissions — retransmitted
frames reach the wire through the connector's single write path.

**Test:** `TestAccessdial_Send_WiredAsRetransmitDispatch` — constructs
`arqsend.Retransmitter` with `connector.Send` as `Dispatch`; forces a retransmit;
asserts the frame reaches the wire via the connector.

### AC-006 (traces to BC-2.04.009 Postcondition 4b, Invariant 3)

The access node's regular, per-tick downstream frame emission
(`halfchannel.HalfChannel.Tick()` output) also reaches the wire through the SAME
`connector.Send` path: for each downstream tick, `arqHandle.EnqueueSend(chanSeq,
payload, now)` registers the frame as in-flight (data frames only), then
`outerassembler.Assemble` + `connector.Send` dispatches it — **including empty-tick
frames** (BC-2.01.002 DI-008: empty ticks are never skipped at the wire-liveness
level). Added alongside the existing in-process `an.DeliverFrame` fan-out
(`startFramesBridge`), not a replacement of it. `Send` is the ONE exported write entry
point on `*accessdial.Connector`; both this path and the retransmit `Dispatch` call it
— never `net.Conn.Write` directly from outside the connector package (Invariant 3).

**Test:** `TestAccessdial_RegularTickEmission_ReachesWire` — drives an access-node-side
downstream tick with and without pending data; asserts both a data frame and an
empty-tick frame reach the wire via `connector.Send`, and `EnqueueSend` is called
before dispatch for data frames. `TestAccessdial_Send_SingleWriterRace` (`go test
-race`) — concurrent calls to `connector.Send` from the regular-tick goroutine and a
simulated retransmit callback do not race.

### AC-007 (traces to BC-2.04.009 Postcondition 5; EC-006)

The connector runs a read loop against the live connection, decoding each inbound wire
frame. Router-relayed ctl frames dispatch toward their respective handlers (a
forward-compat placeholder for `S-BL.RESYNC-FRAME`'s future `control_type = 0x02`
dispatch — NOT implemented by this story). Upstream (keystroke) frames are resolved via
`cmd/switchboard/accessdial_wire.go`'s connector-owned console-binding registry
(`RegisterConsoleBinding(srcAddr, chanID, key, sessionName)`), keyed on `(SrcAddr,
chan_id)`, and on a match delivered to `session.AccessNode.SendKeystroke(key,
sessionName, payload)` — the confirmed production entry point. **On a binding miss
(EC-006, the expected common path in this story's own scope, since the registry is
populated only by a later story), the frame is silently dropped:** no error surfaced,
no connection closed, no crash.

**Test:** `TestAccessdialWire_ConsoleBinding_DispatchesToSendKeystroke` — registers a
binding directly, injects a hand-built inbound frame, asserts `SendKeystroke` called
with the right arguments. `TestAccessdialWire_ConsoleBinding_MissDropsSilently` — no
registered binding; asserts `SendKeystroke` NOT called, connector remains live, no
panic, no connection close (EC-006).

### AC-008 (traces to BC-2.04.009 Postcondition 6, Invariant 2; EC-001, EC-004)

The connector's lifecycle has exactly three states — dialing, admitting, live —
collapsing to dialing (with backoff+jitter, PROVISIONAL values — see Design
Constraints) on any read/write error, `net.Dial` failure, handshake failure, or
explicit shutdown. No state ever carries session data except live. Two or more rapid
successive drops increase backoff rather than tight-looping (EC-004); reconnect
attempts continue indefinitely (no give-up ceiling).

**Test:** `TestAccessdial_StateMachine_ThreeStates_NoUnboundOpenState` — asserts state
transitions dialing→admitting→live→dialing across a forced disconnect, and that no
observable state between admitting-without-live carries session data.
`TestAccessdial_Reconnect_BackoffIncreasesAcrossRepeatedFailures` (EC-004) — asserts
the second backoff interval exceeds the first, bounded by `BackoffCap`.
`TestAccessdial_DialFailure_RetriesWithoutCrash` (EC-001) — router unreachable at dial
time; daemon continues running.

### AC-009 (traces to BC-2.04.009 Invariant 5; EC-002; VP-081 property 2)

Every handshake-phase failure mode (unregistered/revoked/expired key, bad `NonceSig`,
replayed nonce, malformed frame, handshake timeout, duplicate `NodeIdentify`,
`ChallengeResponse` SVTNID mismatch — per BC-2.01.009's Error Codes table) discharges
as a bare connection close per BC-2.01.009 Postcondition 10 (no explicit rejection
message). The connector cannot and does not distinguish "admission was rejected" from
"the network dropped the connection at the same protocol phase" — both are handled by
the identical retry path; neither is fatal; the connector never crashes or exits the
daemon process. The connector's own handshake deadline (PROVISIONAL 10s — see Design
Constraints) fires identically to any other handshake-phase closure.

**Test:** `TestAccessdial_HandshakeFailureModes_AllRetryIdentically` — table-driven
across unregistered key, bad signature, handshake timeout (no `Challenge` ever sent),
and a plain network drop at each of the three message boundaries; asserts identical
connector behavior for all — state != Live, scheduled retry, no crash — confirming
Invariant 5's wire-indistinguishability claim empirically (VP-081 Test Scenarios 2-5).

### AC-010 (traces to BC-2.04.009 Postcondition 7; VP-082 v1.1)

A transition out of live does NOT reset the `arq.ARQ` in-flight `SendBuffer` state, nor
`session.AccessNode`'s console-attachment/authorization state — only the TCP `net.Conn`
and the `outerassembler.Envelope` *struct instance* are torn down and rebuilt on the
next successful admission. Holds by construction (see Design Constraints).
**Corrected in v1.1:** the rebuilt `Envelope`'s `SVTNID`, `SrcAddr`, and `FrameAuthKey`
fields are **byte-identical** to the prior `Envelope`'s across a same-node reconnect —
not "different," as this story's v1.0 draft incorrectly asserted. `FrameAuthKey` is
deterministic by architecture (HKDF from `svtn_id`+`pubkey` only, with no
handshake-specific entropy — the router independently recomputes the identical key
from the same static inputs, so a nonce-salted key would break its forwarding-table
cache on reconnect; RESOLVED, see Residual Open Questions). Only `DstAddr` is not
asserted either way here (it stays the PROVISIONAL zero-value placeholder, see
Non-Goals) — VP-082 Scenario 3 (v1.1) is an equality check on `SVTNID`/`SrcAddr`/
`FrameAuthKey`, not a change/negative-space check.

**Test:** `TestAccessdial_Reconnect_PreservesARQInFlightState` (VP-082 scenario 1) —
seed N in-flight sequence numbers via `EnqueueSend`, force a connection loss, let the
connector reconnect against the same fixture, assert `InFlightContains`/
`PayloadForInFlight` unchanged for all N. `TestAccessdial_Reconnect_PreservesSessionAuthState`
(VP-082 scenario 2) — seed M console attachments, force the same loss/reconnect, assert
all M attachments and their authorization state unchanged.
`TestAccessdial_Reconnect_EnvelopeStable` (VP-082 v1.1 scenario 3, **renamed from
`TestAccessdial_Reconnect_EnvelopeChanges` — the v1.0 name and assertion direction were
factually wrong**) — capture `Envelope()` before the loss; force a connection loss;
let the connector reconnect over a **fresh `net.Conn`** via a **genuine second
handshake** against the same fixture; assert `Envelope()` after equals `Envelope()`
before on `SVTNID`, `SrcAddr`, and `FrameAuthKey` (byte-for-byte equality) —
confirming `FrameAuthKey`'s deterministic derivation holds across a real reconnect,
not merely within a single `admit()` call.

### AC-011 (traces to BC-2.04.009 Invariant 4; DI-004)

The connector dials only the single configured router address (`cfg.RouterAddr`); it
has no mechanism to discover or contact another node's address directly, and this
story does not introduce one. `internal/accessdial` MUST NOT import
`internal/routing` (forbidden edge).

**Test:** `TestAccessdial_NoRoutingImport` — `go list` confirms
`internal/accessdial`'s import set excludes `internal/routing`.
`TestAccessdial_DialsOnlyConfiguredAddress` — constructs the connector with a fixed
`routerAddr`, asserts every dial attempt across multiple reconnects targets that exact
address, never a derived/discovered one.

### AC-012 (traces to the net-new NODE_IDENTIFY-codec-extraction finding; supports AC-002)

The NODE_IDENTIFY wire-codec functions (`EncodeNodeIdentify`, `EncodeChallenge`,
`EncodeChallengeResponse`, `DecodeNodeIdentify`, `DecodeChallengeResponse`) and their
size/msg-kind constants are extracted from `cmd/switchboard/node_identify_wire.go`
(currently unexported, `package main`-scoped) into a new pure-core package,
`internal/nodeidentify`, and exported. A sixth function, `DecodeChallenge` — the
client-side counterpart to `DecodeChallengeResponse`, not present in the codebase today
— is added, mirroring `DecodeChallengeResponse`'s structure exactly.
`cmd/switchboard/node_identify_wire.go` is refactored to call the extracted functions —
**behavior-preserving**: identical wire bytes in, identical wire bytes out.
`S-BL.NODE-IDENTIFY-WIRE`'s existing router-side tests continue to pass unmodified.

**Test:** `TestNodeIdentify_EncodeDecodeRoundTrip` — all six codec functions,
encode-then-decode identity. `TestNodeIdentify_DecodeChallenge_MalformedPayload` —
table-driven: wrong length, wrong `control_type` byte, wrong version byte, wrong
msg-kind byte. `node_identify_wire_test.go`'s existing suite, unmodified, passing
against the refactored call sites (regression, not new).

### AC-013 (traces to BC-2.04.009 Invariant 3 — supplementary race-safety guard)

The connector runs clean under `go test -race` / `just test-race`: no data race is
detected on the shared `net.Conn` write path across concurrent callers (the
regular-tick goroutine and the retransmit `Dispatch` caller).

**Test:** covered by AC-006's `TestAccessdial_Send_SingleWriterRace` — this AC exists
to make race-detector cleanliness an explicit, separately-checkable Definition-of-Done
item, not to add a new test function.

---

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|-----------------|
| `nodeidentify.EncodeNodeIdentify`/`EncodeChallenge`/`EncodeChallengeResponse`/`DecodeNodeIdentify`/`DecodeChallengeResponse`/`DecodeChallenge` | `internal/nodeidentify` | Pure-core (codec, extracted from `cmd/switchboard`) |
| `accessdial.Connector` (dial loop, `admit()`, read loop, `Send`, `Stop`) | `internal/accessdial` | Effectful-shell (network I/O, goroutines) |
| `accessdial.State`/`Handle` | `internal/accessdial` | Pure-core (type + accessor; `State()` is an `atomic.Load`) |
| `config.Config.RouterAddr`/`SVTNID` + `Validate()` checks | `internal/config` | Pure-core (no I/O in `Validate()`, per ARCH-06) |
| `accessdial_wire.go` — `consoleBindingKey`/`consoleBindings`/`RegisterConsoleBinding`/`FrameFn` dispatcher | `cmd/switchboard` | Effectful-shell (dispatch policy over the connector's inbound stream) |
| `access.go` wiring — connector construction/start, regular-send bridge, retransmit `Dispatch` wiring, `discovery.Config` field population | `cmd/switchboard` | Effectful-shell |
| `node_identify_wire.go` (refactored call sites) | `cmd/switchboard` | Effectful-shell, unchanged classification |
| `session.AccessNode.SendKeystroke` | `internal/session` | Read-only consumer — no code change |
| `arq.ARQ.EnqueueSend`/`OnAck` | `internal/arq` | Read-only consumer |
| `arqsend.Retransmitter.Dispatch` | `internal/arqsend` | Read-only consumer — `Connector.Send` becomes the `Dispatch` implementation |
| `outerassembler.Assemble` | `internal/outerassembler` | Read-only consumer |

## Edge Cases

| ID | Description | Handling | AC |
|----|-------------|----------|-----|
| EC-001 | Router unreachable at dial time | Connector retries with backoff; daemon continues running; local tmux/session capture unaffected | AC-008 |
| EC-002 | Router closes the connection during the NODE_IDENTIFY exchange (any BC-2.01.009 failure mode) | Connector observes a read/write error, not an explicit rejection code; logs, retries per backoff — wire-indistinguishable from a plain network drop | AC-009 |
| EC-003 | Connection lost mid-session, after reaching live | Connector transitions to dialing; ARQ in-flight `SendBuffer` and session/console-authorization state are NOT discarded | AC-010 |
| EC-004 | Two or more rapid successive connection drops (flapping) | Backoff increases per Invariant 2 rather than tight-looping; no give-up ceiling | AC-008 |
| EC-005 | SVTN ID or router address not resolvable at startup | Config validation rejects empty/malformed values before any dial attempt — clean startup abort, not a live network error | AC-001 |
| EC-006 | Inbound upstream frame with no resolvable `(SrcAddr, chan_id)` → console binding | Silently dropped — no error, no connection close, no crash; expected common path in this story's own scope | AC-007 |

## Purity Classification

| Module | Classification | Justification |
|--------|-----------------|----------------|
| `internal/nodeidentify` | pure-core | Codec-only; no I/O, no goroutines, no shared mutable state |
| `internal/accessdial` | effectful-shell | Owns a `net.Conn`, goroutines (dial/handshake/read/write loops), a mutex-guarded write path |
| `internal/config` (new fields + `Validate()` checks) | pure-core | `Validate()` performs no I/O — matches ARCH-06's Config-purity contract |
| `cmd/switchboard/accessdial_wire.go` | effectful-shell | Dispatch policy wired to the connector's live inbound stream and `session.AccessNode` |
| `cmd/switchboard/access.go` (wiring additions) | effectful-shell, unchanged classification | Construction/wiring code, same class as existing `runAccess` |

---

## Non-Goals

- **Does not populate the console binding registry in production.** `RegisterConsoleBinding`
  is exported but never called from a real wire-level console-attach flow in this
  story — that mechanism is `S-BL.CONSOLE-CONNECTOR` or a session-bootstrap follow-on
  (see Residual Open Questions).
- **Does not implement the console-side connector** — `S-BL.CONSOLE-CONNECTOR`,
  separate story, currently design-blocked.
- **Does not change `internal/outerassembler.Assemble`'s byte layout or wire schema.**
- **Does not resolve multi-console outbound `DstAddr` fan-out.** `Envelope.DstAddr` is
  a PROVISIONAL zero-value placeholder (mirroring `S-7.04-FU-PE-CONNECTOR`'s own
  zero-envelope precedent for its one unresolved field). The wire-level design question
  (per-console outbound frames vs. router-side fan-out) is explicitly out of this
  story's scope — see Residual Open Questions.
- **Does not implement RESYNC frame consumer logic** (`control_type = 0x02` dispatch)
  — `S-BL.RESYNC-FRAME` scope. The read loop only reserves a forward-compat dispatch
  point (AC-007).
- **Does not implement multi-router failover selection** — a single configured router
  address is the MVP scope (ARCH-03 §ADR-005: "E router has a single path").
- **Does not touch `internal/routing`, `internal/drain`, or `internal/testenv`** from
  the new package (forbidden-edge precedent, AC-011).
- **Does not implement key/credential persistence** beyond `internal/admission`'s
  existing client-side signing primitives (`loadOrGenerateAdmissionKeypair` in
  `access.go` is reused, not rebuilt).

---

## Token Budget Estimate

| Context Source | Estimated Tokens |
|-----------------|-------------------|
| This story spec | ~9k |
| Placement note (binding input, full read required) | ~7k |
| BC-2.04.009 + VP-081 + VP-082 (full read required) | ~6k |
| Referenced production code (`node_identify_wire.go`, `internal/upstreamdial/connector.go`, `internal/session/upstream.go`, `internal/testenv/loopback.go` — read-only precedent surfaces) | ~7k |
| Test infrastructure context (existing router-fixture patterns, `discovery_relay_wire_test.go`) | ~3k |
| **Total implementing-agent context** | **~32k — well within 20–30% of a 200k context window. No story split required.** |

## Tasks

1. [ ] Extract `internal/nodeidentify` from `cmd/switchboard/node_identify_wire.go`
   (AC-012) — pure refactor; existing router-side tests stay green.
2. [ ] Add `DecodeChallenge` to `internal/nodeidentify` (AC-012) — new function, new
   tests.
3. [ ] Refactor `node_identify_wire.go` to call the extracted package (AC-012) —
   behavior-preserving, existing tests green.
4. [ ] `internal/config`: `RouterAddr`/`SVTNID` fields + `Validate()` checks per
   BC-2.09.003 PC-16/PC-17 (v2.3) (AC-001). Use error codes `E-CFG-018`/`E-CFG-019`
   (corrected from `E-CFG-014`/`E-CFG-015`, which collide with codes already
   allocated to shipped admission stories — product-owner v1.1 correction); the
   `error-taxonomy.md` entry itself is still flagged to product-owner/spec-steward
   (see Residual Open Questions) — do not invent taxonomy content beyond the code
   literal itself.
5. [ ] `internal/accessdial`: `State`/`Handle`/`Connector` skeleton, `New`, dial loop
   shape (mirrors `upstreamdial.dialLoop`, no handshake yet — TCP dial + backoff only)
   (AC-004, AC-008).
6. [ ] `internal/accessdial`: `admit()` — the client-side three-message handshake,
   using `internal/nodeidentify` (AC-002, AC-003, AC-009).
7. [ ] `internal/accessdial`: read loop + `SetFrameCallback` + `Send`/single-writer
   guard (AC-006, AC-007, AC-013).
8. [ ] `internal/accessdial`: reconnect preserves injected `*arq.ARQ` pointer and
   caller-owned `session.AccessNode` state by construction (AC-010) — largely "don't
   write code that would break it," verified by the VP-082 tests, not a feature to
   build.
9. [ ] `cmd/switchboard/accessdial_wire.go`: console-binding registry + `FrameFn`
   dispatcher (AC-007).
10. [ ] `cmd/switchboard/access.go` wiring: construct/start the connector, wire the
    regular-send bridge and the retransmit `Dispatch` (both → `connector.Send`), wire
    `discovery.Config` fields (AC-001, AC-005, AC-006, AC-011).
11. [ ] VP-081 integration test (router fixture + connector — happy path, every
    handshake-failure mode, private-key-absence audit) (AC-002, AC-003, AC-004, AC-009).
12. [ ] VP-082 integration test (seeded `arq.ARQ` + seeded `session.AccessNode` +
    forced reconnect + before/after state-identity assertions, including the v1.1
    Scenario 3 Envelope-equality assertion on `SVTNID`/`SrcAddr`/`FrameAuthKey`)
    (AC-010).
13. [ ] Add a CHANGELOG entry under `[Unreleased] > Added` describing the shipped
    access-node data-plane connector behavior (dial, genuine NODE_IDENTIFY admission,
    live ARQ send-path wiring), before creating the PR.

## Previous Story Intelligence

| Predecessor | Lesson carried forward |
|-------------|--------------------------|
| `S-7.04-FU-PE-CONNECTOR` (merged PR #115, `8eb54a5`) | Direct structural precedent for the dial-loop shape and the "connection established" three-step definition — `internal/upstreamdial.dialLoop` (`connector.go:318-459`) maps almost one-to-one onto this story's state machine, with the bootstrap-write step substituted for the full handshake. Also the source of the forbidden-edge ruling (`internal/accessdial` MUST NOT import `internal/routing`/`internal/testenv`) and the zero-envelope `DstAddr` placeholder pattern this story reuses for its own unresolved `DstAddr` field. |
| `S-BL.NODE-IDENTIFY-WIRE` (merged) | Shipped the router-side `encodeNodeIdentify`/`encodeChallenge`/`encodeChallengeResponse`/`decodeNodeIdentify`/`decodeChallengeResponse` functions this story extracts and reuses. Left `discovery.Config.LocalSVTNID`/`LocalNodeAddr` intentionally zero-valued with a forward-obligation comment naming this exact gap — this story closes it as a byproduct of AC-001, not its primary purpose. |
| `S-BL.LOOPBACK-FULLSTACK` (merged PR #135, `72e6e36d`) | `internal/testenv/loopback.go`'s `onDownstreamTick` composition (`EnqueueSend` before dispatch, one shared write path for both regular ticks and retransmits) is the direct, already-tested precedent this story's AC-006 send-path composition mirrors, adapted from in-process delivery to a real `net.Conn`. Also demonstrated the value of capturing `chanSeq` before any frame-shape transformation, since downstream types don't carry it back out. |
| `S-BL.PE-RECEIVE-LOOP` / DRAIN-WIRE lineage | Every symbol claim in ACs must be grep-verified or marked "(new — defined by this story)" — done throughout this story's ACs. Forward-compat unknown-opcode/unknown-sender dispatch MUST default to silent-ignore, never a hard failure or connection close — directly applied to EC-006's binding-miss handling (AC-007), mirroring BC-2.01.008 PC-4's own precedent. |

## Architecture Compliance Rules

| Rule | Source | Enforcement |
|------|--------|--------------|
| `internal/accessdial` MUST NOT import `internal/routing` | Placement note §4.3; DI-004 (Invariant 4) | `go list` / static import-set check (AC-011) |
| `internal/accessdial` MUST NOT import `internal/testenv` | Placement note §4.3 | `go list` / static import-set check |
| `internal/accessdial`'s permitted import set is exactly `{internal/frame, internal/outerassembler, internal/arq, internal/nodeidentify, internal/admission}` | Placement note §4.3 | Code review; `go list` |
| `internal/nodeidentify` imports only `internal/frame`, `internal/admission` (types only), stdlib `crypto/ed25519` | Placement note §5 | Code review; `go list` |
| New packages `internal/accessdial` and `internal/nodeidentify` require ARCH-08 §6.4 registration | ARCH-08 §6.4 ("Adding a new internal package") | Exact numeric §6.5 position is PROVISIONAL/TBD at implementation — both packages must sit above the positions of everything they import (`internal/frame`, `internal/admission`, `internal/arq`, `internal/outerassembler`); `internal/nodeidentify` sits below both `cmd/switchboard` and `internal/accessdial`. Implementer/architect registers the exact position in the same commit that first introduces each import, per the `internal/upstreamdial` v2.6 precedent (pre-code registration). This is a required consequence of §6.4, not one of the 7 new/5 touched File Structure rows below — the placement note's own §7 File Structure table does not enumerate ARCH-08 as a touched file for this story. |
| `Connector.Send` is the sole exported write path on `*accessdial.Connector` | BC-2.04.009 Invariant 3 | Code review — no `net.Conn.Write` call sites outside `internal/accessdial`; `go test -race` (AC-006, AC-013) |
| `Config.Validate()` performs no I/O | ARCH-06 Config-purity contract | Code review (AC-001) |
| `session.AccessNode` gains no new method; `SendKeystroke` is reused as-is | Placement note §3(a) | Code review — confirms no diff to `internal/session/upstream.go` |
| go.md rule compliance | `.claude/rules/go.md` | New code MUST pass `go test -race -count=1 ./...` and `golangci-lint run` without suppressions |

## Library & Framework Requirements

Stdlib only: `net`, `crypto/ed25519`, `encoding/hex`, `sync`/`sync/atomic`, `time`.
Internal packages (all already vendored in-module, at their existing versions — Go
1.25.4 per `go.mod`): `internal/frame`, `internal/outerassembler`, `internal/arq`,
`internal/arqsend`, `internal/admission`, `internal/session` (read-only), `internal/
config`, plus the new `internal/nodeidentify` this story creates. No new external
dependency.

## File Structure Requirements

**New files (7):**

| File | Purpose |
|------|---------|
| `internal/nodeidentify/nodeidentify.go` | Extracted, exported codec: `EncodeNodeIdentify`, `EncodeChallenge`, `EncodeChallengeResponse`, `DecodeNodeIdentify`, `DecodeChallengeResponse`, new `DecodeChallenge`; size/msg-kind constants |
| `internal/nodeidentify/nodeidentify_test.go` | Round-trip tests for all six codec functions, including the new `DecodeChallenge` (encode-then-decode identity, malformed-payload rejection) |
| `internal/accessdial/connector.go` | `Connector` type, `State`/`Handle`, `New`, `Start`, `Send`, `Stop`, `SetFrameCallback`, `admit` (handshake driver), dial/read/write loop |
| `internal/accessdial/connector_test.go` | Unit tests: dial success/failure, admit success/each-handshake-failure-mode (VP-081), reconnect-preserves-injected-`*arq.ARQ`-pointer (VP-082 unit-level slice), `Send` single-writer safety under `go test -race` |
| `internal/accessdial/backoff.go` (or inline in `connector.go`) | Backoff constants/helpers — PROVISIONAL values reusing `internal/upstreamdial`'s `BackoffBase`/`BackoffCap`/`BackoffJitterFraction` |
| `cmd/switchboard/accessdial_wire.go` | Policy/dispatch layer (mirrors `discovery_relay_wire.go`'s shape): `consoleBindingKey`/`consoleBindings` registry, `RegisterConsoleBinding`, the `FrameFn` implementation discriminating ctl frames vs. upstream-keystroke frames and calling `an.SendKeystroke` |
| `cmd/switchboard/accessdial_wire_test.go` | Dispatch-table tests: known binding → `SendKeystroke` called correctly; binding miss → dropped, no crash, no `SendKeystroke` call (EC-006) |

**Touched files (5):**

| File | Change |
|------|--------|
| `internal/config/config.go` | New `RouterAddr`/`SVTNID` fields; `Validate()` gains the two new checks |
| `internal/config/config_test.go` | New `Validate()` test cases for the two fields |
| `cmd/switchboard/node_identify_wire.go` | Five codec functions' bodies replaced with calls into `internal/nodeidentify`; size/kind constants sourced from the new package — behavior-preserving, existing tests unmodified |
| `cmd/switchboard/access.go` | `runAccess`/`runAccessWithConnector`: read `cfg.RouterAddr`/`cfg.SVTNID`, construct `internal/accessdial.Connector`, wire `discovery.Config` fields, start the connector, add the wire-dispatch bridge alongside `startFramesBridge`, wire `arqsend.Retransmitter`'s `Dispatch` to `connector.Send` |
| `cmd/switchboard/access_test.go` (and/or a new `router_access_connector_test.go`, mirroring `router_pe_connector_test.go`'s naming) | Integration test: `runAccessWithConnector` end-to-end against a router fixture (VP-081's happy path, driven through `access.go`'s real construction path) |

---

## Residual Open Questions

Carried from the placement note §10/§11. Decomposition-time items resolved below with
PROVISIONAL values (Step-4.5 or the implementer confirms/revises); items that imply a
new error-taxonomy entry, a spec clarification, or a new VP are FLAGGED FOR
PRODUCT-OWNER rather than invented here.

**Resolved with PROVISIONAL values (not blocking):**

1. **Handshake timeout** (`c.handshakeTimeout` in `admit()`) — PROVISIONAL **10s**,
   mirroring the router-side `nodeIdentifyHandshakeTimeout=10s` (`node_identify_wire.go:56`).
   Whether the client should use an identical or shorter value (the client controls
   when it starts waiting, unlike the reactive router) is an implementation-time call.
2. **Backoff constant values** — PROVISIONAL: `BackoffBase=500ms`, `BackoffCap=30s`,
   `BackoffJitterFraction=0.25`, reused verbatim from `internal/upstreamdial` per
   Invariant 2's own text ("a directly reusable precedent, not a binding requirement").
3. **Outbound `DstAddr` for multi-console fan-out** — PROVISIONAL: zero-value
   placeholder, mirroring `S-7.04-FU-PE-CONNECTOR`'s own zero-envelope precedent.
   Bounded explicitly as a Non-Goal of this story (see Non-Goals) — the wire-level
   design question (per-console outbound frames vs. router-side fan-out) is likely
   `S-BL.CONSOLE-CONNECTOR`-adjacent or session-bootstrap-era scope, not decided here.

**RESOLVED in v1.1 (product-owner):**

1. **`FrameAuthKey` derivation entropy — RESOLVED.** `FrameAuthKey` is deterministic
   by architecture: HKDF derived from `svtn_id`+`pubkey` **only**, with **no
   handshake-specific entropy**. The router independently recomputes the identical
   key from the same static inputs on every admission; a nonce-salted (handshake-
   entropy-including) key would invalidate the router's forwarding-table cache on
   every reconnect. Confirmed by product-owner via VP-082 v1.1 Scenario 3 / ARCH-04 /
   BC-2.05.010. This was flagged as needing confirmation in v1.0; it is no longer
   open. **Consequence:** AC-010's Envelope assertion is corrected from "differs
   across reconnect" to "byte-identical across reconnect" for `SVTNID`/`SrcAddr`/
   `FrameAuthKey` — see AC-010 and its renamed test, `TestAccessdial_Reconnect_EnvelopeStable`.
2. **Possible new BC or BC-2.09.003 extension for `internal/config` validation of
   `RouterAddr`/`SVTNID` — RESOLVED.** Product-owner extended BC-2.09.003 to v2.3,
   adding PC-16 (`router_addr` host:port validation) and PC-17 (`svtn_id` hex/16-byte
   validation) as the formal validation home for these two fields — no separate new
   BC was needed. This story's Anchors Consumed table, Design Constraints, and AC-001
   now cite PC-16/PC-17 directly (see above) rather than treating BC-2.09.003 as a
   pattern-precedent-only analogy.
3. **One new edge case recommended for BC-2.04.009** was already resolved before
   v1.0 — product-owner added EC-006 in BC-2.04.009 v1.1 per the placement note's §11
   item 1 recommendation.

**FLAGGED FOR PRODUCT-OWNER (not resolved by this story):**

1. **New error-taxonomy codes required — numbers corrected in v1.1, taxonomy-file
   addition still pending.** `internal/config.Validate()`'s new `RouterAddr`/`SVTNID`
   checks (BC-2.09.003 PC-16/PC-17) need two new codes in the `E-CFG` family.
   **Correction (v1.1):** the v1.0 proposal (`E-CFG-014`/`E-CFG-015`) collided with
   codes already allocated to shipped admission stories (BC-2.09.003 v2.1/2.2) —
   `E-CFG-014` through `E-CFG-017` are taken. The current highest allocated slot is
   `E-CFG-017`; this story now proposes `E-CFG-018` (empty/malformed `RouterAddr`)
   and `E-CFG-019` (empty/malformed `SVTNID`) as the next free slots — **NEW, add to
   `.factory/specs/prd-supplements/error-taxonomy.md`.** Story-writer does not edit
   the taxonomy file; product-owner/spec-steward still needs to formally ratify the
   exact codes and canonical message strings in `error-taxonomy.md` before
   implementation relies on them — only the collision was corrected here, the file
   itself has not yet been updated.
2. **Whether AC-006 (regular-send composition) and AC-007 (inbound dispatch) warrant
   their own VPs** beyond VP-081/VP-082's existing scope — the placement note's own §8
   poses this question without answering it. This story covers both with ordinary unit
   test coverage (see AC-006/AC-007 above); product-owner/architect decides whether a
   dedicated VP is warranted for either. Unchanged by this v1.1 pass.

**Not blocking, implementer-level only (no PO flag needed):**

3. **`internal/testenv` extension** to support a real `internal/accessdial.Connector`
   in its `Env`/`RouterHandle` fixtures, needed for any integration test exercising
   `access.go`'s actual wiring rather than `internal/accessdial` in isolation. Likely
   needed for Task 11's integration test; not scoped in this story's File Structure
   Requirements (which cover the connector's own test surface). If the implementer
   finds this extension necessary, it is implementation-time scope, not a story-spec
   gap.

---

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-08-31 | **Re-sync burst — propagates product-owner's resolution of 4 completeness items flagged in v1.0** (per `bc_array_changes_propagate_to_body_and_acs`), story-body-only, no BC/VP files edited by this burst. **(1) Error-code collision fixed:** the v1.0 proposal `E-CFG-014`/`E-CFG-015` collided with codes already allocated to shipped admission stories (BC-2.09.003 v2.1/2.2); corrected to `E-CFG-018` (RouterAddr)/`E-CFG-019` (SVTNID) throughout the story body — Design Constraints code-comment block, AC-001, Task 4, Residual Open Questions, and the top-of-file status note. The `error-taxonomy.md` addition itself remains flagged to product-owner/spec-steward — only the numbers were corrected here, the taxonomy file was not edited. **(2) AC-010 assertion + test corrected:** VP-082 Scenario 3 (now v1.1, product-owner) established that `FrameAuthKey` is deterministic by architecture (HKDF from `svtn_id`+`pubkey` only, no handshake entropy — the router independently recomputes it, and a nonce-salted key would break its forwarding-table cache on reconnect), so `SVTNID`/`SrcAddr`/`FrameAuthKey` are BYTE-IDENTICAL across a same-node reconnect, not "different" as v1.0 incorrectly asserted. AC-010's body, the Design Constraints "Reconnect and State Preservation" section, the Anchors Consumed VP-082 row, and the `admit()` code sketch's `FrameAuthKey` comment are all corrected to the equality claim; the test `TestAccessdial_Reconnect_EnvelopeChanges` is renamed `TestAccessdial_Reconnect_EnvelopeStable` and its assertion direction reversed (fresh `net.Conn` + genuine second handshake + equality assertions on `SVTNID`/`SrcAddr`/`FrameAuthKey`, replacing the prior before-!=-after negative-space check); Task 12 updated to reference the corrected Scenario 3 assertion. **(3) Residual Open Question 2 (`FrameAuthKey` entropy) marked RESOLVED**, moved from "FLAGGED FOR PRODUCT-OWNER" into a new "RESOLVED in v1.1" subsection along with Residual Open Question 4 (the BC-2.09.003 extension question, also resolved by item 4 below); the remaining flagged list renumbers to 2 items (error-taxonomy file addition; whether AC-006/AC-007 warrant dedicated VPs — unchanged). **(4) BC-2.09.003 annotation updated** from "existing, unedited" to v2.3 (product-owner added PC-16 — `router_addr` host:port validation — and PC-17 — `svtn_id` hex/16-byte validation — as the formal validation home, superseding this story's v1.0 pattern-precedent-only framing): updated in the `bc_traces` frontmatter comment, the Anchors Consumed table row, the Design Constraints config code-comments, and AC-001's body; `bc_traces`/`behavioral_contracts` frontmatter arrays already included `BC-2.09.003` (confirmed unchanged, no array edit needed — only its annotation). `vp_traces` frontmatter comment for VP-082 updated to note the v1.1 Scenario 3 correction. Frontmatter `version` 1.0 → 1.1; AC count stays 13 (no new/removed AC — AC-010's assertion direction corrected in place); File Structure/Tasks counts unchanged (7 new + 5 touched files, 12 implementation tasks + 1 CHANGELOG task). `status` stays `draft`. No BC/VP files edited, no code written, no commit made (state-manager commits per the burst-splitting rule). |
| 1.0 | 2026-08-31 | Full decomposition from the v0.1-backlog-stub. Consumes `.factory/decisions/S-BL.ACCESS-CONNECTOR-placement-note.md` v1.0 (architect resolution of all three BC-2.04.009-flagged open items — SVTN-ID/router-address sourcing, regular-send-path composition, inbound upstream-keystroke delivery target — plus the net-new NODE_IDENTIFY codec-extraction finding) and BC-2.04.009 v1.1 (product-owner authored the BC in canonical form; added EC-006 per the placement note's own §11 recommendation). 13 ACs (AC-001 through AC-013), each tracing to a specific BC-2.04.009 postcondition/invariant/edge-case clause and, where applicable, a VP-081/VP-082 property. Full BC-2.04.009 clause coverage confirmed: all 4 preconditions, all 7 postconditions, all 5 invariants, and all 6 edge cases (EC-001 through EC-006) trace to at least one AC. 7 new + 5 touched files (File Structure Requirements), 12 implementation tasks + 1 CHANGELOG-entry task. Three PROVISIONAL values chosen for Step-4.5/implementer confirmation (handshake timeout, backoff constants, `DstAddr` placeholder); four items flagged for product-owner (two new `E-CFG` error-taxonomy codes, `FrameAuthKey` entropy confirmation, whether AC-006/AC-007 warrant dedicated VPs, a possible BC-2.09.003 extension). `status` stays `draft` (not `ready`) — BC-2.04.009 exists in canonical form and every AC traces bidirectionally, but Step-4.5 adversarial convergence has not yet run. No BC/VP files edited by this burst; no code written; no commit made (state-manager commits per the burst-splitting rule). |
| 0.1-backlog-stub | 2026-08-31 | Initial backlog stub — bookkeeping only, per `S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §2. See prior frontmatter `backlog_origin.notes` for full stub-era history. |
