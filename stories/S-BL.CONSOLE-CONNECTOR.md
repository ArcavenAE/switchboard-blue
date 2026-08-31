---
artifact_id: S-BL.CONSOLE-CONNECTOR
document_type: story
level: ops
story_id: S-BL.CONSOLE-CONNECTOR
title: "Console-node data-plane connector — NOT READY TO SIZE: open architecture question (see Open Question / Blocked section)"
status: backlog
producer: story-writer
timestamp: 2026-08-31T00:00:00Z
version: "0.1-backlog-stub"
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: PE
estimated_points: TBD   # scoping note §3: "Not estimated pending the caveat" — see Open Question / Blocked section
# BC status: pending PO authorship — no BC exists, and none can be confidently drafted
# yet. DESIGN-BLOCKED: this story's default-assumption scope is UNCONFIRMED. See the
# "Open Question / Blocked" section below before treating this as a normal backlog stub.
bc_traces: []   # none — cannot be anchored until the architecture pre-step (or explicit PO scope acceptance) resolves which transport console data-plane delivery uses. See scoping note §3.
vp_traces: []   # none minted; blocked on the same open question as bc_traces.
subsystems: [session-access]
architecture_modules:
  - cmd/switchboard          # runConsole (mgmt_wire.go:1192+) — currently mgmt-plane-only, ZERO data-plane scaffolding today
  - internal/session         # AccessNode/Publisher/ConsoleKey — reused IF the default-assumption scope is accepted
  - internal/arq             # receive-side reorder/SACK state — reused IF the default-assumption scope is accepted
  - internal/outerassembler  # reused IF the default-assumption scope is accepted
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []   # no code-level dependency, but see "Open Question / Blocked" — a design pre-step (or explicit PO scope acceptance) gates SIZING, not merely scheduling
blocks: []   # would be a prerequisite of S-BL.RESYNC-FRAME's "Layer 2" scope, symmetric to S-BL.ACCESS-CONNECTOR, IF its default-assumption scope is accepted — see body and S-BL.RESYNC-FRAME's "Architect Elaboration (2026-08-31)" note
inputDocuments:
  - '.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md'
  - '.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md'
  - '.factory/decisions/RULING-W6TB-C-console-transport.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.003.md'
acceptance_criteria_count: 0
backlog_origin:
  source: S-BL.DATAPLANE-CONNECTOR-scoping-note
  adr_disposition: "N/A — not an ADR-005 lineage item; named alongside S-BL.ACCESS-CONNECTOR as RESYNC's Layer-2 emitter-side prerequisite, but flagged materially lower-confidence"
  drift_items_consumed: []
  notes: >
    Named by .factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md §3 as the
    console-side twin of S-BL.ACCESS-CONNECTOR, but explicitly flagged as NOT
    confidently scopable today: runConsole (cmd/switchboard/mgmt_wire.go:1192+) has
    ZERO data-plane scaffolding of any kind (a stronger absence than the access-node
    case, which at least constructs a halfchannel.HalfChannel and a session.AccessNode
    — there is nothing in runConsole to extend), and
    .factory/decisions/RULING-W6TB-C-console-transport.md already made a considered,
    final decision to move console's own CONTROL surface OFF the SVTN data-plane model
    onto the mgmt-plane Unix socket, specifically to avoid building a second protocol
    stack for console. That ruling is scoped to attach/detach/switch, not the
    downstream terminal-output DATA stream — but its reasoning is in direct tension
    with proposing an SVTN-wire dial-loop for console's data, and the scoping note is
    explicit this is a genuinely open architectural question, not merely an
    unbuilt-but-settled one.

    This is bookkeeping only — a backlog stub to make the named prerequisite durable
    and tracked, NOT a scoping commitment. No BC/VP content is created here. Per the
    scoping note §3 recommendation, this story should not be sized as a normal backlog
    item until either (a) a short architecture pre-pass resolves whether console
    session-data delivery follows the same SVTN dial-loop model as the access node, or
    (b) product-owner explicitly accepts the default-assumption scope described below
    as a working scope, with the assumption flagged unconfirmed.
---

# S-BL.CONSOLE-CONNECTOR: Console-Node Data-Plane Connector

> **STATUS: BACKLOG STUB — CAVEAT-FLAGGED, NOT READY TO SIZE.** Unlike a normal
> backlog stub, this story's scope is NOT confidently known. Read the "Open Question /
> Blocked" section below before doing anything with this story other than tracking its
> existence. This entry is bookkeeping only — it makes the named prerequisite durable
> and tracked, not a scoping commitment.

## Narrative (proposed, default-assumption — see caveat below)

- **As a** console-mode switchboard daemon
- **I want to** (possibly) dial its configured router, complete a client-side
  NODE_IDENTIFY admission handshake, and maintain a live data-plane connection
  symmetric to `S-BL.ACCESS-CONNECTOR`
- **So that** it can receive downstream session output over the wire and carry RESYNC
  emission — **IF** this is in fact the right transport model for console data, which
  is not yet settled (see below)

## Open Question / Blocked

**This story is not ready to size normally.** Two independent facts make its scope a
genuinely open architectural question rather than an unbuilt-but-settled gap:

1. **`runConsole` has zero data-plane scaffolding of any kind**
   (`cmd/switchboard/mgmt_wire.go:1192+`) — no halfchannel, no ARQ, no frame codec use,
   not even a deferred/stubbed field. `runConsole` constructs only a management-plane
   RPC server (`newMgmtServer(cfg, "console", ...)` + `BuildConsoleHandlers`/
   `BuildSessionsHandlers`) and its own comment states *"Console mode has no routing
   subsystem — pass nil router."* This is a **stronger absence** than the access-node
   case (`S-BL.ACCESS-CONNECTOR`), which at least constructs a `halfchannel.HalfChannel`
   and a `session.AccessNode` — there is nothing in `runConsole` to extend.
2. **`.factory/decisions/RULING-W6TB-C-console-transport.md` already made a considered,
   final (`status: final`) decision to move console's own CONTROL surface OFF the SVTN
   data-plane model** and onto the mgmt-plane Unix socket, specifically to avoid
   building a second protocol stack for console (its own rationale: implementing the
   SVTN-channel interpretation "would require building a second control-message
   protocol inside the SVTN data plane — a major architectural undertaking with no
   precedent in the codebase"). That ruling is scoped to attach/detach/switch
   (**control**), not the downstream terminal-output **data** stream itself — but its
   reasoning is in direct tension with proposing an SVTN-wire dial-loop for console's
   data, and this tension is not this story's (or the scoping note's) to resolve.
3. **`BC-2.04.003`** ("Console Attaches to Session by Name; Receives Downstream Stream
   and Sends Upstream Keystrokes") is deliberately transport-agnostic — it describes
   only that "the console establishes a channel with the access node," never
   specifying the physical transport. It does not settle the question either way.

**Before this story can be sized (estimated_points assigned, ACs drafted, BCs
authored), one of two things must happen first:**

- **(a)** A short, focused architecture pre-pass answers: does console session-data
  delivery ride the same SVTN dial-loop model as the access node
  (`S-BL.ACCESS-CONNECTOR` symmetric), or does `RULING-W6TB-C`'s mgmt-plane precedent
  extend to data too, via some streaming/long-poll RPC mechanism instead? — **or**
- **(b)** Product-owner explicitly accepts the "default-assumption scope" below as a
  working scope, with the assumption recorded as unconfirmed in this story's own
  frontmatter/notes at that time.

Full framing: `.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §3.

## Context

`.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md` (§3, §8) found that no
production or test-harness code path performs real wire-level session-data relay in
this repository — the finding that also produced `S-BL.ACCESS-CONNECTOR`. That note's
own topology analysis is access-node-focused; `S-BL.DATAPLANE-CONNECTOR-scoping-note.md`
(§3) extends the question to the console side and, unlike the access-node case,
concludes the console leg's target shape is a genuinely open question rather than an
unbuilt-but-settled gap — for the two reasons in "Open Question / Blocked" above.

## Scope Boundary — proposed, default-assumption ONLY (per scoping note §3)

**Not a confident scope.** If the console leg follows the same model as
`S-BL.ACCESS-CONNECTOR` (the default architectural assumption, absent a decision
otherwise): `net.Dial` to a router, a client-side NODE_IDENTIFY handshake (identical
primitives to the access-node story), a downstream frame-read loop feeding
`internal/arq`'s receive-side reorder/SACK state, and (once `S-BL.RESYNC-FRAME` Layer 1
lands) the RESYNC gap-detection/emission logic on top of that connection.

## Non-Goals (if scoped per the default assumption)

Same non-goals as `S-BL.ACCESS-CONNECTOR`, plus: does **not** reopen or re-litigate
`RULING-W6TB-C`'s control-surface decision (attach/detach/switch stay mgmt-plane —
this story, if it proceeds, touches console's DATA path only).

## Sketched Acceptance Criteria

> **Not drafted.** Per the Open Question above, acceptance criteria depend on an
> architecture decision this stub does not make. Sketching ACs against the
> default-assumption scope before that decision would risk locking in an unconfirmed
> transport model. ACs will be sketched once (a) or (b) above resolves.

## When to Schedule

**Not before the Open Question resolves.** Per
`.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §4 "Recommended
sequencing," this is the last item in the two-connector sequence: `S-BL.ACCESS-CONNECTOR`
first (independent, highest confidence), `S-BL.RESYNC-FRAME` Layer 1 in parallel
(independent of both connectors), then resolve this story's caveat (architecture
pre-pass or explicit product-owner scope acceptance) — which can happen in parallel
with the first two — before this story itself is scheduled.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-08-31 |
| Origin | `.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` §3 — architect scoping note, human-requested |
| Related design notes | `.factory/decisions/RULING-W6TB-C-console-transport.md` (the tension this story surfaces but does not resolve) |
| BC/VP status | None exist; none can be confidently drafted until the Open Question resolves. |
| Design blocker | See "Open Question / Blocked" above. `estimated_points` intentionally TBD. |
| Anchors tracked | S-BL.RESYNC-FRAME "Layer 2" prerequisite (emitter side) — IF the default-assumption scope is accepted |
| Status transitions | (none yet) |
