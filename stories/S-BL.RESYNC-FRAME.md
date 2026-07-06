---
artifact_id: S-BL.RESYNC-FRAME
document_type: story
level: ops
story_id: S-BL.RESYNC-FRAME
title: "RESYNC control-frame protocol — ADR-005 second half"
status: backlog
producer: story-writer
timestamp: 2026-07-06T00:00:00Z
version: "0.1-backlog-stub"
phase: 2
epic: E-3
wave: backlog
priority: P2
scope_phase: PE
estimated_points: 5
bc_traces:
  - BC-2.01.002   # EMPTY_TICK / DATA / RESYNC FrameType discriminator extension (ADR-005 wire format)
  - BC-2.01.004   # outer header layout — RESYNC as a new ChannelFrame.Type value
vp_traces: []
subsystems: [transport-layer]
architecture_modules:
  - internal/outerassembler   # RESYNC frame emission (new FrameType via Assemble)
  - internal/arq              # last_acked_seq access for RESYNC trigger and replay-from target
  - internal/netingress       # reconnect state machine; RESYNC receiver co-located here
  - internal/frame            # RESYNC FrameType constant
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on:
  - S-BL.OA       # MERGED — outerassembler wire primitive; Assemble() is the RESYNC emitter
  - S-BL.ARQ-TX   # MERGED — ARQ retransmit; replay-from-last-acked-seq is arqsend.Retransmitter's
inputDocuments:
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'   # ADR-005 §Downstream ARQ Continuity
  - '.factory/stories/S-BL.OA-outer-assembler-DELIVERY.md'   # ADR-005 disposition: separable-still-anchored
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
| Status transitions | (none yet) |
