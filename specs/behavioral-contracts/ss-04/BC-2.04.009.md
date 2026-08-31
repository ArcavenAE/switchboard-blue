---
artifact_id: BC-2.04.009
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-08-31T00:00:00Z
phase: 1a
inputs:
  - '.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md'
  - '.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.001.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.001.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.008.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.009.md'
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.005.md'
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
input-hash: "934ffbc"
extracted_from: null
bc_id: BC-2.04.009
subsystem: session-access
architecture_module: internal/accessdial
capability: CAP-013
priority: P2
criticality: high
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-08-31
    version: "1.1"
    actor: product-owner
    change: >
      Additive edge-case amendment recommended by
      S-BL.ACCESS-CONNECTOR-placement-note.md Open Item 3 / §11 item 1: added
      EC-006 (inbound upstream frame with no resolvable console binding is
      silently dropped, mirroring BC-2.01.008 PC-4's silent-ignore posture;
      this is the expected common path in this story's own scope since the
      binding registry is populated only by a later story). Added a minimal
      cross-reference from Postcondition 5. No existing postcondition,
      invariant, precondition, or EC-001..EC-005 behavior changed.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md'
  - '.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md'
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-013]
kos_anchors:
  - elem-node-router-architecture
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.04.009: Access Node Dials Its Router, Completes Genuine NODE_IDENTIFY Admission, and Wires the Live ARQ Send-Path

## Description

The access-mode daemon opens a reconnecting TCP data-plane connection to its
configured router and completes a genuine, client-side `NODE_IDENTIFY`
admission handshake (BC-2.01.009's three-message exchange, from the node's
side) before any session data crosses the wire. Once admitted, the
connection becomes the access node's live egress path — both for
`arqsend.Retransmitter`-driven downstream ARQ retransmissions and for the
daemon's regular per-tick downstream frame emission — and its live ingress
path for frames the router relays down to this access node. This BC is what
actually discharges `BC-2.04.001` Precondition 3 ("the access node is
admitted to an SVTN, or admission in progress"), which that BC states as a
given without defining a mechanism. It is the direct, real-admission
successor to `S-7.04-FU-PE-CONNECTOR`'s shipped router-to-router connector,
substituting genuine identity-bound admission for that story's deferred
zero-envelope bootstrap.

## Preconditions

1. The access-mode daemon has started and holds a loaded (or freshly
   generated) Ed25519 admission keypair (`loadOrGenerateAdmissionKeypair`,
   `access.go` — unchanged by this BC).
2. The access-mode daemon has, by some mechanism, a single router address to
   dial and the target SVTN ID under which its keypair is (or will be)
   registered. **This precondition is currently unmet in the codebase as
   written** — see Architecture Anchors, Open Item 1. This BC assumes both
   values become available by the time its connector runs; it does not
   itself define how.
3. `session.AccessNode`, its `Publisher`, and the downstream
   `halfchannel.HalfChannel` are already constructed (per `runAccess`/
   `buildAccessComponents`) and unchanged by this BC.
4. The configured router implements the router-side `NODE_IDENTIFY`
   handshake receiver (BC-2.01.009) and, once admission succeeds, resolves
   `(SVTNID, NodeAddr)` bindings via `Router.BindInterface`/
   `LookupInterface` (BC-2.01.010).

## Postconditions

1. **Dial.** On startup, and after every connection loss (Postcondition 6),
   the connector opens a TCP connection to the configured router address.
   `net.Dial` success alone does NOT constitute "connected" for this BC's
   purposes — see Postcondition 3.

2. **Client-side `NODE_IDENTIFY` handshake — genuine admission, not a
   placeholder.** Once dialed, the connector performs the mirror-image,
   client-side half of BC-2.01.009's three-message exchange:
   a. Sends `NodeIdentify` (reusing the existing `encodeNodeIdentify`
      function in `cmd/switchboard/node_identify_wire.go`) carrying the
      access node's SVTN ID and Ed25519 public key.
   b. Reads and decodes the router's `Challenge` response via a **new**
      `decodeChallenge` function — the client-side counterpart to the
      router's existing `encodeChallenge`; no such decoder exists in the
      codebase today.
   c. Signs the received nonce with the access node's private key
      (`ed25519.Sign`) and sends `ChallengeResponse` (reusing the existing
      `encodeChallengeResponse` function, also in `node_identify_wire.go`).
   d. **DI-002.** The private key itself is never serialized into any wire
      message, log line, or diagnostic output at any step of a–c — only the
      public key (step a) and the signature bytes (step c) cross the wire.

3. **Connection-established definition (three-step; PE-CONNECTOR's Q6
   shape, with genuine admission substituted for its zero-envelope
   placeholder).** The connection is "established" only once ALL of the
   following hold, in order:
   i. `net.Dial` succeeds (Postcondition 1);
   ii. the three-message `NODE_IDENTIFY` exchange (Postcondition 2)
       completes without the router closing the connection;
   iii. the connector performs its first successful frame write on the
        connection.
   Unlike `S-7.04-FU-PE-CONNECTOR`'s own shipped shortcut (a zero-valued
   `outerassembler.Envelope`, admission explicitly deferred as "not-core"),
   step (ii) here is genuine, identity-bound admission — the resulting
   `Envelope{SVTNID, SrcAddr, DstAddr, FrameAuthKey}` is real, so
   `routing.LookupInterface`/`BindInterface` (which key on
   `(svtnID, nodeAddr)`) resolve correctly for this connection.

4. **Live ARQ send-path wiring.** Once established (Postcondition 3), the
   connector's `net.Conn.Write` — guarded so it is safe to call from more
   than one goroutine (Invariant 3) — becomes:
   a. the `Dispatch` callback supplied to `arqsend.Retransmitter.Retransmit`
      for downstream ARQ retransmissions (BC-2.02.005 Postconditions 3/5);
      and
   b. **[OPEN — see Architecture Anchors, Open Item 2]** the sink for the
      access node's own regular, per-tick downstream frame emission
      (`halfchannel.HalfChannel.Tick()` output) — i.e., this connection is
      the daemon's sole egress path onto the wire once live, not merely a
      retransmit-only side channel. The exact composition (which goroutine
      drains `Tick()`'s output onto this connection, and how it
      interleaves with retransmit dispatch on the same `net.Conn`) is not
      settled by this BC or by the scoping note that names this
      prerequisite — flagged for architect/story-writer resolution.

5. **Frame read loop (inbound).** The connector runs a read loop against
   the live connection (reusing `netingress.ReadFrame`'s bounded-read
   discipline, or an equivalent) that decodes each inbound wire frame and
   delivers it toward `session.AccessNode`'s existing processing:
   router-relayed ctl frames (e.g. RESYNC, once `S-BL.RESYNC-FRAME` Layer 1
   lands) are dispatched to their respective handlers; upstream (keystroke)
   frames are delivered toward whichever entry point resolves a frame to
   its authorized console and forwards it to `KeystrokeSink`
   (BC-2.04.005/BC-2.04.006). **[OPEN — see Architecture Anchors, Open Item
   3]**: the concrete `session.AccessNode`-side target for inbound upstream
   keystroke frames is not identified by this BC — `AccessNode.DeliverFrame`
   has a matching `(hdr frame.OuterHeader)` shape but fans frames OUT to
   attached consoles (the opposite direction), so it is not self-evidently
   the right target. When no binding can be resolved for an inbound frame,
   see Edge Cases EC-006 for the expected (silent-drop) behavior.

6. **Per-connection lifecycle and reconnect.** The connector's lifecycle
   has exactly three states — *dialing*, *admitting*, *live* — collapsing
   to *dialing* (with backoff, Invariant 2) on any read/write error,
   `net.Dial` failure, handshake failure (including a router-initiated
   close during or immediately after the `NODE_IDENTIFY` exchange — see
   Edge Cases), or explicit shutdown request. There is no "connected but
   unadmitted" state that ever carries session data — mirroring BC-2.01.009
   Postcondition 8's router-side "fully bound or closed, no unbound-open
   state" invariant, applied to the connector's own view of the same
   connection.

7. **ARQ/session state survives reconnect.** A transition out of *live*
   (Postcondition 6) does NOT reset the `arq.ARQ` in-flight `SendBuffer`
   state, nor the access node's session/console-authorization state — only
   the TCP connection and its `NODE_IDENTIFY`-derived envelope
   (`SrcAddr`/`DstAddr`/`SVTNID`/`FrameAuthKey`) are torn down and rebuilt
   on the next successful admission. This is required so that a subsequent
   `S-BL.RESYNC-FRAME` replay request can be served from state that
   survived the outage (see Related BCs).

## Invariants

1. **DI-002 (node private keys never transit the network).** Enforced at
   every handshake step (Postcondition 2d); only the public key and
   signature bytes cross the wire.
2. **Reconnect uses backoff with jitter, not a tight redial loop.**
   `internal/upstreamdial`'s constants (`BackoffBase=500ms`,
   `BackoffCap=30s`, `BackoffJitterFraction=0.25`) are a directly reusable
   precedent, not a binding requirement — story-writer/architect confirms
   the actual values at implementation time.
3. **The connector's wire-write path is single-writer or externally
   synchronized.** Both the retransmit `Dispatch` callback (Postcondition
   4a) and the regular per-tick emission path (Postcondition 4b) target the
   same `net.Conn`; concurrent, uncoordinated `Write` calls from more than
   one goroutine without synchronization is a data race (go.md rule 12;
   mirrors `arqsend`'s own documented single-writer contract).
4. **DI-004 (no direct node-to-node communication).** The connector dials
   only the configured router; it has no mechanism to discover or contact
   another node's address directly, and this BC does not introduce one.
5. **Admission failure is wire-indistinguishable from ordinary network
   failure.** Per BC-2.01.009 Postcondition 10, the router returns no
   explicit rejection message on any handshake failure path — only
   connection closure. The connector MUST treat any handshake-phase
   connection closure as a retryable admission/connectivity failure (Edge
   Cases EC-002 through EC-004), not a fatal daemon error.

## Trigger

Access-mode daemon startup (`runAccess`/`runAccessWithConnector`);
connection loss at any point after establishment.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Router unreachable at dial time | Connector retries with backoff (Invariant 2); daemon continues running (does not crash); local tmux/session capture is unaffected. |
| EC-002 | Router closes the connection during the `NODE_IDENTIFY` exchange (bad signature, revoked/expired/unregistered key, timeout, or any of BC-2.01.009's E-ADM-001/003/005/008/015/022/023/024 paths) | Connector observes a read/write error, not an explicit rejection code (Invariant 5). Logs the failure; retries per backoff. There is no distinct "rejected" vs. "network failure" code path available at the wire level. |
| EC-003 | Connection lost mid-session, after having reached *live* | Connector transitions to *dialing*. ARQ in-flight `SendBuffer` state and session/console-authorization state are NOT discarded (Postcondition 7). Any frames that should have been delivered during the outage window are a `S-BL.RESYNC-FRAME` concern, not this BC's. |
| EC-004 | Two or more rapid successive connection drops (flapping) | Backoff increases per Invariant 2 rather than tight-looping. This BC does not define a give-up ceiling — reconnect attempts continue indefinitely, matching `internal/upstreamdial`'s own unbounded-retry precedent. |
| EC-005 | The access node's SVTN ID or router address is not resolvable at startup (Precondition 2's open gap) | Out of this BC's scope to resolve. The connector cannot begin dialing until that value is available by whatever mechanism the owning story settles on; this BC does not invent one. |
| EC-006 | Inbound upstream frame arrives with no resolvable console binding (`SrcAddr`/`chan_id` not present in the connector's binding registry) | The frame is silently dropped — no error surfaced, no connection closed, no crash — mirroring BC-2.01.008 Postcondition 4's silent-ignore posture for unrecognized `control_type` values, applied here to an unrecognized-sender inbound frame. This is the expected common path in `S-BL.ACCESS-CONNECTOR`'s own scope: the binding registry is populated only by a future story, so an inbound frame with no resolvable binding is not an anomaly during this story's window. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Access node configured with a reachable router address and a valid, registered admission keypair | Dial succeeds; `NodeIdentify`/`Challenge`/`ChallengeResponse` exchange completes; connector reaches *live*; first frame write succeeds | happy-path |
| Router address unreachable | Connector retries with backoff; daemon continues running; no crash | edge-case |
| Router closes connection after `NodeIdentify` (key not registered for the target SVTN) | Connector observes closure (not an explicit rejection); retries per backoff | edge-case |
| Live connection drops mid-session; router remains reachable | Connector reconnects; ARQ in-flight `SendBuffer` state is unchanged across the drop; a fresh admission handshake completes before any frame is written on the new connection | edge-case |
| Two consecutive dial failures | Second backoff interval is longer than the first (bounded by `BackoffCap`), not identical | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-081 | Dial → admit → first-frame-write round trip completes for a valid, registered keypair; any handshake-phase closure is treated as retryable, not fatal | integration |
| VP-082 | Reconnect preserves ARQ in-flight `SendBuffer` state and session/console-authorization state; only the wire connection and its envelope are torn down and rebuilt | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 |
| L2 Domain Invariants | DI-002 (node private keys never transit the network); DI-004 (no direct node-to-node communication — the access node reaches the network only via its router) |
| Architecture Module | internal/accessdial (candidate package name per `S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §2 item 1; final name is architect's/story-writer's call at scheduling time) |
| Stories | S-BL.ACCESS-CONNECTOR |
| Capability Anchor Justification | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 — CAP-013's own capability text states the access node "publishes available sessions over the SVTN"; that publication is meaningless without a live, admitted network connection to a router, which is exactly what this BC establishes. `BC-2.04.001` (CAP-013's existing anchor) covers the LOCAL tmux-connects half of CAP-013; this BC covers the NETWORK-connects half. Together they discharge CAP-013 in full — this BC is specifically what discharges `BC-2.04.001` Precondition 3 ("the access node is admitted to an SVTN, or admission in progress"), which that BC states as a given without defining a mechanism. |

## Related BCs

- BC-2.04.001 — depends on: this BC discharges BC-2.04.001 Precondition 3 ("admitted to an SVTN, or admission in progress"); BC-2.04.001's own behavioral content (`internal/tmux` local control-mode connection) is unedited — only its Related BCs citation is updated in the same burst.
- BC-2.01.009 — depends on: this BC's client-side handshake (Postcondition 2) is the mirror image of BC-2.01.009's router-side three-message exchange; reuses `encodeNodeIdentify`/`encodeChallengeResponse`, adds the new client-side `decodeChallenge`.
- BC-2.09.001 — composes with (pattern precedent, not shared code): the dial-loop shape and the Q6 three-step "connection established" definition (Postcondition 3) parallel `internal/upstreamdial.Connector`'s router-to-router PE-mode uplink; not directly reused, per the scoping note's DAG-layering rationale (a new package, not `internal/upstreamdial` itself).
- BC-2.02.005 — depends on: this BC's live wire connection is what BC-2.02.005's downstream ARQ retransmit machinery (`arqsend.Retransmitter`) dispatches onto once wired (Postcondition 4a).
- BC-2.04.003 — related to: the inbound frame read loop (Postcondition 5) eventually carries the same class of session data BC-2.04.003 governs (console attach/downstream-stream semantics); BC-2.04.003 remains transport-agnostic and is not itself amended by this BC.

## Architecture Anchors

- `S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §2 — full scope boundary, non-goals, and the precise PE-CONNECTOR relationship (parallels/reuses/does-not-reuse) this BC is grounded in.
- `S-BL.RESYNC-FRAME-placement-note.md` §3, §8 — the topology finding that surfaced this BC's need (no production or test-harness code path performs real wire-level access-node ⟷ router session-data relay today).
- `ARCH-03-routing-engine.md` §"Downstream ARQ" — the `SendBuffer`-lives-at-the-access-node topology this BC's Postcondition 4 wires a live connection into.
- `ARCH-INDEX.md` §SS-04 (session-access; `internal/tmux`, `internal/session`).

**Open items — flagged for architect, not resolved by the scoping note or by this BC:**

1. **SVTN-ID / router-address sourcing.** `internal/config.Config` has no
   field today carrying either the access node's target SVTN ID or a
   router address to dial (confirmed: no `SVTNID`/`RouterAddr`-shaped field
   exists in the `internal/config` package). `access.go`'s own in-code
   comment already names this exact gap for a sibling concern
   (`discovery.Config.LocalSVTNID`/`LocalNodeAddr` are "intentionally
   zero-valued... populated by S-BL.NODE-IDENTIFY-WIRE" — a forward
   obligation that appears to remain open in the current codebase even
   though `S-BL.NODE-IDENTIFY-WIRE` itself shipped its wire-protocol
   scope). This BC's Preconditions 1–2 assume both values are available by
   the time the connector runs; where they come from (a new config field
   analogous to `upstream_routers`, a value derived from the admission
   keypair's provisioning response, or another source) is unresolved.
2. **Composition of the regular (non-retransmit) downstream send path with
   this connector.** The scoping note (§2 item 3) explicitly wires only
   the ARQ *retransmit* `Dispatch` callback to the connector's
   `net.Conn.Write`; it does not specify how the daemon's ordinary
   per-tick `HalfChannel.Tick()` downstream frame emission reaches the
   wire. Postcondition 4b states this BC's expectation that the same
   connection is the sole egress path for both, but the exact
   goroutine/composition shape is unresolved.
3. **Inbound upstream-keystroke frame delivery target.** Postcondition 5's
   read loop needs a concrete `session.AccessNode` (or sibling) entry
   point for inbound upstream keystroke frames. `AccessNode.DeliverFrame`
   is the only existing method with a matching `(hdr frame.OuterHeader)`
   shape, but it fans a frame OUT to attached consoles (the downstream
   direction) — the opposite of what an inbound upstream frame needs. No
   existing method visibly fits; a new one may be required.

## Story Anchor

S-BL.ACCESS-CONNECTOR (backlog stub as of this BC's authoring — v0.1;
full acceptance-criteria decomposition is pending story-writer, gated on
this BC existing in canonical form per the S-7.01 Spec-First Gate).

## VP Anchors

VP-081 — dial + admit + first-frame-write round trip (Postconditions 1–3);
admission-failure-is-retryable (Invariant 5).
VP-082 — reconnect preserves ARQ in-flight `SendBuffer` state and
session/console-authorization state (Postcondition 7); also closes the
open question named in `S-BL.RESYNC-FRAME-placement-note.md` §7.3/§9 item 6
("does closing a connection today tear down associated ARQ/session state,
or does it already survive independently?").

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-08-31 | Additive edge-case amendment per `S-BL.ACCESS-CONNECTOR-placement-note.md` Open Item 3 / §11 item 1: added EC-006 (inbound upstream frame with no resolvable console binding is silently dropped, mirroring BC-2.01.008 PC-4's silent-ignore posture — the expected common path in this story's own scope, since the connector-owned binding registry is populated only by a later story). Added a minimal cross-reference from Postcondition 5 to EC-006. No change to any existing postcondition, invariant, precondition, or EC-001 through EC-005. |
| 1.0 | 2026-08-31 | Initial commission. Authors the access-node data-plane connection-establishment BC named and bounded by `S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §2 and §5 item 1, discharging `BC-2.04.001` Precondition 3. Flags three open architecture questions (SVTN-ID/router-address sourcing; regular-send-path composition with the retransmit dispatch; inbound upstream-keystroke-frame delivery target) for architect resolution rather than inventing designs for them. Mints VP-081 (dial+admit+first-write round trip) and VP-082 (reconnect preserves ARQ/session state — also closes the open question in `S-BL.RESYNC-FRAME-placement-note.md` §7.3/§9 item 6). |
