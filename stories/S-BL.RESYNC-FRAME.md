---
artifact_id: S-BL.RESYNC-FRAME
document_type: story
level: ops
story_id: S-BL.RESYNC-FRAME
title: "RESYNC control-frame protocol — ADR-005 second half"
status: backlog
producer: story-writer
timestamp: 2026-07-06T00:00:00Z
version: "0.2-backlog-stub"
phase: 2
epic: E-3
wave: backlog
priority: P2
scope_phase: PE
estimated_points: 5
bc_traces:
  - BC-2.01.002   # EMPTY_TICK / DATA / RESYNC FrameType discriminator extension (ADR-005 wire format) — SUPERSEDED, see "Architect Elaboration (2026-08-31)" below; NOT re-decomposed in this edit, left intact for scheduling time
  - BC-2.01.004   # outer header layout — RESYNC as a new ChannelFrame.Type value — SUPERSEDED, see "Architect Elaboration (2026-08-31)" below; NOT re-decomposed in this edit, left intact for scheduling time
vp_traces: []
subsystems: [transport-layer]
architecture_modules:
  - internal/outerassembler   # RESYNC frame emission (new FrameType via Assemble) — SUPERSEDED, see "Architect Elaboration (2026-08-31)"; NOT re-decomposed in this edit
  - internal/arq              # last_acked_seq access for RESYNC trigger and replay-from target
  - internal/netingress       # reconnect state machine; RESYNC receiver co-located here — SUPERSEDED (netingress is accept-only), see "Architect Elaboration (2026-08-31)"; NOT re-decomposed in this edit
  - internal/frame            # RESYNC FrameType constant — SUPERSEDED, see "Architect Elaboration (2026-08-31)"; NOT re-decomposed in this edit
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on:
  - S-BL.OA       # MERGED — outerassembler wire primitive; Assemble() is the RESYNC emitter
  - S-BL.ARQ-TX   # MERGED — ARQ retransmit; replay-from-last-acked-seq is arqsend.Retransmitter's
  - S-BL.ACCESS-CONNECTOR    # Layer 2 ONLY (real end-to-end round trip) — the replay-actor / access-node leg. Layer 1 (wire assemble/parse/relay against fakes) has NO dependency on this story. See "Architect Elaboration (2026-08-31)" below and .factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md §4.
  - S-BL.CONSOLE-CONNECTOR   # Layer 2 ONLY (real end-to-end round trip) — the RESYNC-emitter / console leg, itself design-blocked pending an architecture pre-step (see its own story file). Layer 1 has NO dependency on this story.
inputDocuments:
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'   # ADR-005 §Downstream ARQ Continuity
  - '.factory/stories/S-BL.OA-outer-assembler-DELIVERY.md'   # ADR-005 disposition: separable-still-anchored
  - '.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md'   # architect design elaboration (v1.0, 2026-08-30) — frame-model correction, Layer 1/Layer 2 split, BC amendment needs
  - '.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md'   # architect scoping note (v1.0, 2026-08-31) — names/bounds the two Layer-2 connector prerequisite stories
acceptance_criteria_count: 0
backlog_origin:
  source: S-BL.OA-DELIVERY
  adr_disposition: ADR-005 second half (separable-still-anchored per S-BL.OA DELIVERY adr_disposition)
  drift_items_consumed:
    - S403-O4-LINEAGE          # ADR-005 resync wire-mechanics; narrowed from S403-O4 through S-BL.NI→S-BL.OA
  notes: >
    ADR-005 (ARCH-03 §Downstream ARQ Continuity Under Router Failover) resolves OQ-004 with
    a resync-from-last-ACK approach: on path failover the downstream half-channel sends a RESYNC
    control frame requesting retransmit from last_acked_seq + 1.

    S-BL.OA (merged PR #96) delivered the wire-format primitive: internal/outerassembler's
    Assemble(ChannelFrame, sackBitmap, Envelope) produces wire bytes with HMAC matching
    routing.verifyFrameHMAC. S-BL.OA's adr_disposition for ADR-005 is `separable-still-anchored`:
    the wire-mechanics primitive (encode/decode channel header, compose wire frame) is shipped;
    the RESYNC frame as a *control-frame type* with its own state machine is orthogonal and ships here.

    This story implements the ADR-005 second half:
    1. RESYNC FrameType — a new internal/frame.FrameType constant (parallel to EMPTY_TICK and DATA
       in BC-2.01.002); the outerassembler emits it via Assemble with RESYNC in the ChannelFrame.Type.
    2. RESYNC emitter — co-located with the send-path buffer: fires when the receiver detects a
       gap (missing chan_seq) on reconnect. Trigger: post-reconnect, the receiver's last_acked_seq
       drives the RESYNC payload (retransmit from N).
    3. RESYNC receiver + replay — co-located with the ARQ replay buffer in internal/arq or netingress:
       on receiving a RESYNC frame, the sender replays from last_acked_seq + 1 using arqsend.Retransmitter.
    4. Reconnect state machine — the netingress reconnect path that arms the RESYNC emitter when
       a new router connection is established (surviving connection churn without losing ARQ state).

    S403-O4 lineage: DRIFT-S4.03-001 (DegradationEvent per-frame observation) narrowed through
    S-BL.NI→S-BL.OA; what remains in this row is the ADR-005 protocol work, which is this story.
---

# S-BL.RESYNC-FRAME: RESYNC Control-Frame Protocol

> **STATUS: BACKLOG STUB.** This story is the ADR-005 second half. The wire-format
> primitive was delivered by S-BL.OA (PR #96). Acceptance criteria, file structure,
> and task list will be fleshed out when the story is scheduled.

## Architect Elaboration (2026-08-31)

> Recorded per `.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md` (architect
> design pass, v1.0, 2026-08-30). **This is an annotation, not a re-decomposition** —
> the Sketched Acceptance Criteria below (other than the AC-001 supersession noted
> here) are left intact; full re-derivation, including a corrected `bc_traces`/
> `architecture_modules` frontmatter, happens when this story is scheduled.

- **Frame-model correction (SUPERSEDES sketched AC-001):** RESYNC is **not** a new
  `internal/frame.FrameType` constant. The outer-header `frame_type` enum has exactly
  six canonical values (`0x01`–`0x06`), all assigned; RESYNC has carried a *reserved*
  `ctl(0x03)` payload discriminator slot — `control_type = 0x02` — since BC-2.01.008
  v1.0 (2026-07-11), which predates this stub (2026-07-06). AC-001 as sketched ("A new
  `FrameType` constant `RESYNC` is added to `internal/frame`...") is **superseded in
  full**; `internal/frame/frame.go` is untouched by this story, and RESYNC frames are
  **not** built via `outerassembler.Assemble` (that composes session-data channel
  framing, not control-frame framing — every shipped `ctl(0x03)` opcode hand-assembles
  its own bytes instead). See placement note §1.
- **Layer 1 / Layer 2 split:** **Layer 1** — wire assemble/parse
  (`assembleResyncFrame`/`parseResyncFrame`), router-side dispatch+relay (a new
  `case 0x02:` arm in `buildRoute`, using `routing.LookupInterface`), and the
  emitter/replay-trigger *logic* as isolated, unit-testable primitives against fakes —
  is buildable and testable **today**, independent of any connector story, and also
  discharges FO-DRAIN-WIRE-001 (the DRAIN PR's forward obligation to dispatch
  `control_type=0x02`). **Layer 2** — a real two-daemon, wire-level, "no content loss"
  round trip (the sketched AC-005 as originally written) — is **blocked** on two
  prerequisite connector stories, neither of which exists yet:
  `S-BL.ACCESS-CONNECTOR` (the replay-actor / access-node leg — the party that runs
  `arqsend.Retransmitter`) and `S-BL.CONSOLE-CONNECTOR` (the RESYNC-emitter / console
  leg, itself design-blocked — see that story's own "Open Question / Blocked"
  section). Sketched AC-005 needs rescoping at scheduling time to test Layer 1 against
  fakes (mirroring `discovery_relay_wire_test.go`'s `buildRelayRouter` pattern), with
  the true end-to-end assertion deferred until both connector stories land. Both new
  `depends_on` entries above are scoped to Layer 2 only — they do not gate Layer 1 work.
  See placement note §8, and `.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md`
  for the two connector stories' own scoping.
- **BC amendment required (flagged for product-owner, not made here):** BC-2.01.008
  v1.3's 4-byte control header has no room for RESYNC's required `chan_id`/
  `resync_from_seq` payload fields. A new sibling BC in the ARQ subsystem (candidate ID
  **BC-2.02.010**, per the DISCOVERY_RELAY precedent of defining extended-payload
  semantics in the owning subsystem's BC rather than inline in BC-2.01.008) is
  recommended, alongside a mechanical registry-row update to BC-2.01.008 itself. See
  placement note §6 for the exact proposed payload layout and content, itemized for
  product-owner to execute directly. This story's `bc_traces` (currently
  `BC-2.01.002`, `BC-2.01.004`) will need correcting at scheduling time — the placement
  note's ruling is that `BC-2.01.002` is not a genuine RESYNC trace at all (it is
  `ChannelFrame.Type`/`EMPTY_TICK`-specific), and `BC-2.01.008` (the actual `ctl`
  schema home) is missing from the current list entirely.
- **Pointer:** full elaboration, all open questions (Q1–Q9), and the traceability
  summary are in `.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md`.

## Narrative

- **As a** node that has failed over from one router to another
- **I want to** send a RESYNC control frame requesting retransmit from my last ACKed
  sequence number
- **So that** my downstream ARQ session continues without losing data that was in-flight
  during the failover (ADR-005)

## Context

ADR-005 (ARCH-03 §Downstream ARQ Continuity) decides: on path failover, the downstream
half-channel performs a resync rather than stateful ARQ state transfer. Resync is safe
because: (a) the SACK bitmap tells the receiver what it has and hasn't seen, (b) retransmit
carries the original `chan_seq` so deduplication works, (c) terminal state is recoverable.

S-BL.OA delivered the channel-header codec and `Assemble()` that produces wire bytes. The
RESYNC FrameType, emitter, receiver/replay, and reconnect state machine are the second half.

S-BL.ARQ-TX delivered `internal/arqsend.Retransmitter` — the replay-from primitive (gap-walk
→ PayloadForInFlight → Assemble with new ChanSeq → Dispatch). The RESYNC receiver drives
this Retransmitter to replay from `last_acked_seq + 1`.

## ADR-005 Anchor

ADR-005 (ARCH-03 §ADR-005) disposition from S-BL.OA DELIVERY:
`separable-still-anchored` — wire-format primitive shipped; RESYNC control-frame type +
state machine is orthogonal; follow-on story S-BL.RESYNC-FRAME builds on outerassembler
without re-litigating the byte layout.

## Anchors Consumed

| Anchor | Verbatim ID | Source |
|--------|-------------|--------|
| ADR-005 second half — RESYNC control-frame type + state machine | ADR-005 | ARCH-03 §ADR-005; S-BL.OA DELIVERY adr_disposition |
| S403-O4 lineage — ADR-005 resync wire-mechanics narrowed | S403-O4 / DRIFT-S4.03-001 | STATE.md row; narrowed through S-BL.NI→S-BL.OA |

## Sketched Acceptance Criteria

> ACs are illustrative. Exact scope, test names, and BC postcondition references will
> be confirmed at scheduling time.

**AC-001 (BC-2.01.002 / internal/frame):** A new `FrameType` constant `RESYNC` is added
to `internal/frame`, parallel to `EMPTY_TICK` and `DATA`. The outerassembler emits it via
`Assemble(ChannelFrame{Type: frame.RESYNC, ...}, ...)`.

**AC-002 (ADR-005 emitter):** On reconnect, the receiver detects the gap between its
`last_acked_seq` and the first received `chan_seq`. The RESYNC emitter fires a RESYNC frame
requesting retransmit from `last_acked_seq + 1`. Emitter is co-located with the send-path
buffer in netingress.

**AC-003 (ADR-005 receiver + replay):** On receiving a RESYNC frame, the sender invokes
`arqsend.Retransmitter.Retransmit(from: last_acked_seq+1)`. The replayed frames carry the
original `chan_seq` values so deduplication at the receiver (SACK bitmap) prevents duplicates.

**AC-004 (reconnect state machine):** The netingress reconnect path arms the RESYNC emitter
when a new router connection is established. Connection churn (disconnect + reconnect) does
not lose ARQ state — `last_acked_seq` persists across reconnects.

**AC-005 (round-trip):** Integration test: two-daemon in-process stack; simulate router
failover; assert RESYNC fires and session data is recovered from `last_acked_seq + 1` with
no content loss.

## Non-Goals

- Does not implement the PE outbound dial loop. That is `S-7.04-FU-PE-CONNECTOR`.
- Does not change `Assemble()`'s byte layout — that is frozen per S-BL.OA delivery.
- Does not implement stateful ARQ state transfer between routers (rejected alternative, ADR-005).

## When to Schedule

After S-BL.OA and S-BL.ARQ-TX are merged (both are merged). Requires per-node connection
concept in netingress (for the reconnect trigger). Can be prototyped against the existing
outerassembler + arqsend primitives immediately.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-06 |
| Origin | S-BL.OA DELIVERY adr_disposition: separable-still-anchored (ADR-005 second half) |
| Anchors tracked | ADR-005, S403-O4 lineage |
| Status transitions | 2026-08-31: architect elaboration recorded (frame-model correction, Layer 1/Layer 2 split, BC amendment flagged) — see "Architect Elaboration (2026-08-31)" above; `depends_on` gains two Layer-2-only entries (S-BL.ACCESS-CONNECTOR, S-BL.CONSOLE-CONNECTOR). Story remains `backlog`/unscheduled; no re-decomposition performed. |

## Changelog

| Version | Change |
|---------|--------|
| 0.2-backlog-stub | Architect elaboration annotated (not a re-decomposition). Added "Architect Elaboration (2026-08-31)" section recording: AC-001 superseded (RESYNC is `ctl(0x03)`/`control_type=0x02`, not a new `internal/frame.FrameType`); the Layer 1 (buildable/testable now, independent) vs Layer 2 (blocked on two new connector prerequisite stories) scope split; a BC amendment requirement (candidate `BC-2.02.010`, flagged for product-owner, not made here). `depends_on` gains `S-BL.ACCESS-CONNECTOR` and `S-BL.CONSOLE-CONNECTOR`, both explicitly scoped Layer-2-only. `inputDocuments` gains the two source design notes. `bc_traces`/`architecture_modules` entries annotated SUPERSEDED inline but left otherwise unedited — full correction deferred to scheduling time, per this pass's bookkeeping-only scope. Source: `.factory/decisions/S-BL.RESYNC-FRAME-placement-note.md` (v1.0) and `.factory/decisions/S-BL.DATAPLANE-CONNECTOR-scoping-note.md` (v1.0). |
| 0.1-backlog-stub | Initial backlog stub authored 2026-07-06, per S-BL.OA DELIVERY adr_disposition (ADR-005 second half, separable-still-anchored). |
