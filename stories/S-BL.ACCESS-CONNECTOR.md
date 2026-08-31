---
artifact_id: S-BL.ACCESS-CONNECTOR
document_type: story
level: ops
story_id: S-BL.ACCESS-CONNECTOR
title: "Access-node data-plane connector — dial, NODE_IDENTIFY admission, and live ARQ send-path wiring"
status: backlog
producer: story-writer
timestamp: 2026-08-31T00:00:00Z
version: "0.1-backlog-stub"
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: PE
estimated_points: "8-13"
# BC status: pending PO authorship — BC-2.04.009 below is a CANDIDATE id only (not yet
# authored). Do not promote this story's status to `ready` until it exists in canonical
# form (BC-\d+\.\d{2}\.\d{3}) and every AC below cites it via a real bidirectional trace
# (S-7.01 Spec-First Gate).
bc_traces:
  - BC-2.04.001   # existing, unedited — PC-3 ("admitted to an SVTN, or admission in progress") is the precondition this story's connector discharges
  - BC-2.09.003   # existing, unedited — PE-CONNECTOR's PC-9 connect-half precedent this story's dial-loop pattern parallels
  - BC-2.04.009   # CANDIDATE — NOT YET CREATED. Product-owner to author at scheduling time. See S-BL.DATAPLANE-CONNECTOR-scoping-note.md §5 item 1.
vp_traces: []   # CANDIDATE VPs only, not yet minted (next free id VP-081+). See scoping note §5 item 3: (1) dial+admit+first-frame-write round trip, (2) reconnect preserves session/ARQ state.
subsystems: [session-access, admission-security]
architecture_modules:
  - cmd/switchboard          # runAccess/runAccessWithConnector (access.go) gains the dial site
  - internal/accessdial      # NEW package (name TBD at scoping) — Connector type: dial + reconnect/backoff + client-side NODE_IDENTIFY handshake
  - internal/upstreamdial    # pattern precedent (dial-loop shape, Q6 three-step "connection established") — NOT directly reused code
  - internal/session         # AccessNode/Publisher/ConsoleKey — the session layer this connector feeds into
  - internal/arq             # downstream SendBuffer this connector's dispatch must be constructed against
  - internal/arqsend         # Retransmitter — its Dispatch callback becomes this connector's net.Conn.Write
  - internal/outerassembler  # Assemble — session-bootstrap frame assembly (Q6 pattern, reused)
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []   # buildable now — no unmerged prerequisite; the server-side NODE_IDENTIFY primitives it needs already ship (node_identify_wire.go)
blocks: []   # NOT a hard frontmatter dependency edge on S-BL.RESYNC-FRAME (that story predates this one and does not declare depends_on here) — this story is a prerequisite for S-BL.RESYNC-FRAME's "Layer 2" scope specifically (real end-to-end round trip), not its independent "Layer 1" scope. See S-BL.DATAPLANE-CONNECTOR-scoping-note.md §4 and the S-BL.RESYNC-FRAME "Architect Elaboration (2026-08-31)" note.
inputDocuments:
  - '.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md'
  - '.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md'
  - '.factory/stories/S-7.04-FU-PE-CONNECTOR.md'
  - '.factory/decisions/S-7.04-FU-PE-CONNECTOR-placement-note.md'
acceptance_criteria_count: 0
backlog_origin:
  source: S-BL.DATAPLANE-CONNECTOR-scoping-note
  adr_disposition: "N/A — not an ADR-005 lineage item; this is a newly-named Layer-2 prerequisite for S-BL.RESYNC-FRAME, surfaced by the architect's RESYNC placement note §3/§8 and elaborated in the connector scoping note"
  drift_items_consumed: []
  notes: >
    Named and bounded by .factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md §2,
    at the human's request to make S-BL.RESYNC-FRAME-placement-note.md §3/§8's finding
    durable and tracked: no production or test-harness code path in this repository
    performs real wire-level session-data relay between an access node and a router.
    runAccess/runAccessWithConnector (cmd/switchboard/access.go) construct
    session.AccessNode, a downstream halfchannel.HalfChannel, and a router instance
    that is explicitly "constructed-but-not-in-live-data-path," but never net.Dial,
    never call outerassembler.Assemble, and never read/write a wire frame.

    This story gives the access-mode daemon a live, reconnecting TCP data-plane
    connection to its router, with genuine (not deferred/placeholder) NODE_IDENTIFY
    admission — an improvement over S-7.04-FU-PE-CONNECTOR's own shipped shortcut
    (a zero-valued outerassembler.Envelope, admission explicitly deferred as
    "not-core"), which was a real constraint of PE-CONNECTOR's own timing (it merged
    2026-07-08, a week before BC-2.01.008 v1.3 shipped the client-admittable
    NODE_IDENTIFY handshake primitives this story now reuses).

    This is bookkeeping only — a backlog stub to make the prerequisite durable and
    tracked. Full decomposition (ACs, file structure, task list) happens when the
    story is scheduled. No BC/VP content is created or edited here; scoping note §5
    item 1/3 is routed to product-owner for adjudication and drafting.
---

# S-BL.ACCESS-CONNECTOR: Access-Node Data-Plane Connector

> **STATUS: BACKLOG STUB.** This story is a newly-named prerequisite, scoped by the
> architect in `.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` (§2) at
> the human's request. Acceptance criteria, file structure, and task list will be
> fleshed out when the story is scheduled. This entry is bookkeeping only — it makes
> the prerequisite durable and tracked, not a full decomposition.

## Narrative

- **As an** access-mode switchboard daemon
- **I want to** dial its configured router, complete a genuine client-side
  NODE_IDENTIFY admission handshake, and maintain a live, reconnecting TCP data-plane
  connection
- **So that** `internal/arq`'s downstream `SendBuffer` and `arqsend.Retransmitter` can
  dispatch real wire frames — instead of the current in-process-only path — and so
  that dependent protocol work (e.g. `S-BL.RESYNC-FRAME`'s real end-to-end round trip)
  has a live wire connection to exercise

## Context

`.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md` (§3, §8) found a load-bearing
topology gap while elaborating RESYNC's frame-model design: `internal/arq`'s downstream
`SendBuffer` lives at the **access node** (ARCH-03 §"Downstream ARQ"), but
`cmd/switchboard`'s access-mode daemon (`runAccess`/`runAccessWithConnector` in
`access.go`) has **zero data-plane network code today** — no `net.Dial`, no
`outerassembler.Assemble`, no wire-frame read/write. A repository-wide grep confirms
the only production data-plane dial-out is `internal/upstreamdial.Connector`, used
exclusively for router-to-router PE-mode uplinks. The access-mode daemon constructs
`session.AccessNode`, a downstream `halfchannel.HalfChannel`, and a router instance
that is explicitly documented as "constructed-but-not-in-live-data-path" — real
scaffolding, never wired to the network.

`.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` (§2) names and bounds
this gap as a proper backlog story at the human's request, so it can be tracked instead
of remaining an unnamed finding buried in a design note. It is directly precedented by
`S-7.04-FU-PE-CONNECTOR` (merged PR #115, `internal/upstreamdial`) — same dial-loop
shape, same Q6 three-step "connection established" definition (dial → bootstrap-assemble
→ write) — but this story cannot repeat PE-CONNECTOR's shipped shortcut of a
zero-valued `outerassembler.Envelope` with admission deferred as "not-core." RESYNC's
Layer 2 (and any genuine end-to-end verification of session data flowing over the wire)
needs a real, identity-bound admitted connection so `routing.LookupInterface`/
`BindInterface` — which key on `(svtnID, nodeAddr)` — resolve correctly. The server-side
handshake primitives this needs (`encodeNodeIdentify`, `encodeChallengeResponse`) already
ship in `cmd/switchboard/node_identify_wire.go`; PE-CONNECTOR didn't have these available
when it was scoped (it merged 2026-07-08, a week before BC-2.01.008 v1.3 landed
2026-07-15). See the scoping note §2.1 for the full parallels/reuses/does-not-reuse
breakdown against PE-CONNECTOR.

## Scope Boundary (per scoping note §2 — subject to revision at scheduling)

1. New effectful package (`internal/accessdial` or equivalent — exact name is
   story-writer's/architect's call at scoping time): a `Connector` type performing
   `net.Dial("tcp", routerAddr)` with reconnect/backoff.
2. Client-side NODE_IDENTIFY handshake — send `NodeIdentify` (reusing
   `encodeNodeIdentify`), receive/decode the router's `Challenge` response (a **new**
   `decodeChallenge` function — only the router-side `encodeChallenge` exists today),
   sign the nonce and send `ChallengeResponse` (reusing `encodeChallengeResponse`). This
   is genuine admission, not a placeholder.
3. Integration with `internal/outerassembler.Assemble` / `internal/arqsend.Retransmitter`:
   the connector's `net.Conn.Write` becomes the `Dispatch` callback
   `arqsend.Retransmitter.Retransmit` already expects.
4. A frame **read** loop on the live connection, to receive frames the router relays
   down to this access node (including, eventually, RESYNC frames).
5. Per-connection lifecycle (established → admitted → live-data), teardown-and-reconnect
   with backoff on error.
6. `runAccess`/`runAccessWithConnector` wiring: construct the `Connector` at startup,
   pass the router address from a new config field, wire received frames into
   `session.AccessNode`'s existing delivery path.

Full scope boundary, including exact function/file citations, is in the scoping note §2.

## Non-Goals

- Does **not** implement RESYNC's own emitter/receiver/replay-trigger logic — that
  remains `S-BL.RESYNC-FRAME`'s own scope (its Layer 1, independent of this story).
- Does **not** implement multi-router failover selection — a single configured router
  address is the MVP scope (ARCH-03 §ADR-005: "E router has a single path").
- Does **not** touch `internal/routing`, `internal/drain`, or `internal/testenv` from
  the new package (forbidden-edge precedent inherited from `S-7.04-FU-PE-CONNECTOR`'s
  own placement note Q4 ruling).
- Does **not** resolve the console-side leg — see `S-BL.CONSOLE-CONNECTOR` (separate
  story, design-blocked pending an architecture pre-step).
- Does **not** implement key/credential persistence beyond what `internal/admission`'s
  existing client-side signing primitives require (`loadOrGenerateAdmissionKeypair` in
  `access.go` is reused, not rebuilt).

## Sketched Acceptance Criteria

> ACs are illustrative only — placeholders naming the shape of what will be asserted.
> Exact scope, test names, and BC postcondition references are confirmed at scheduling
> time, once BC-2.04.009 (or an equivalent) is authored by product-owner. Per the
> S-7.01 Spec-First Gate, this story cannot move to `ready` until that BC exists in
> canonical form and every AC below cites it via `(traces to BC-2.04.009 postcondition N)`.

- **AC-001 (candidate, traces to BC-2.04.009 — not yet authored):** access-mode daemon
  dials its configured router and completes the connection-established sequence (dial →
  NODE_IDENTIFY handshake → first frame write).
- **AC-002 (candidate):** connection loss triggers reconnect with backoff; ARQ/session
  state (last-acked-seq, in-flight `SendBuffer` entries) survives the reconnect.
- **AC-003 (candidate):** frames the router relays down to this access node are read off
  the live connection and delivered into `session.AccessNode`'s existing path.
- **AC-004 (candidate):** admission failure (bad signature, rejected challenge) does not
  crash the daemon; it retries per the backoff policy, matching `internal/upstreamdial`'s
  precedent.

## When to Schedule

Can start immediately — no blocking dependency on anything not already merged. This is
the **recommended-first** prerequisite of the two connector stories (see
`.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §4 "Recommended
sequencing"): highest confidence, and most directly unblocks real end-to-end
verification of `S-BL.RESYNC-FRAME` (its Layer 2) and the broader "does session data
actually flow over the wire yet" question. Requires product-owner to author BC-2.04.009
(or an equivalent) before the story can move past `draft`/`backlog` toward `ready`
(S-7.01 Spec-First Gate).

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-08-31 |
| Origin | `.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §2 — architect scoping note, human-requested |
| Related design notes | `.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md` (§3, §8 — the finding that surfaced this gap) |
| BC/VP status | BC-2.04.009 and all VPs are CANDIDATE ONLY — not yet authored. Routed to product-owner (scoping note §5). |
| Anchors tracked | S-BL.RESYNC-FRAME "Layer 2" prerequisite (recommended-first of the two connector stories) |
| Status transitions | (none yet) |
