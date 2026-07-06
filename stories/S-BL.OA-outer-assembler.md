---
artifact_id: S-BL.OA-outer-assembler
document_type: story
level: ops
story_id: S-BL.OA
title: "outer-assembler — compose ChannelFrame + OuterHeader into wire frames"
status: merged
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
epic: TBD
wave: backlog
priority: P1
scope_phase: E
estimated_points: TBD
bc_traces: []
vp_traces: []
subsystems: [session-networking]
architecture_modules: [internal/frame, internal/channelheader, internal/halfchannel]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-1.01, S-1.02, S-2.01]
blocks: []
inputDocuments:
  - '.factory/specs/architecture/ARCH-02-protocol-stack.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.005.md'
  - '.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md'
  - '.factory/cycles/cycle-1/wave-1/wave-adversary-pass-01.md'
acceptance_criteria_count: 0
revision: "0.1-backlog-stub"
backlog_origin:
  source: wave-1-adversary-pass-01
  drift_items_received:
    - wave-adv-F-001 (spec side closed in BC-2.01.002 v1.4 PC5; code side in F-001+F-002 refactor PR)
    - wave-adv-F-003 (composed wire-format test — LOW)
    - wave-adv-F-004 (channel-header serializer — LOW, per-story-scope)
  related_issues:
    - drbothen/vsdd-factory#260
---

# S-BL.OA: Outer-Assembler — Compose ChannelFrame + OuterHeader into Wire Frames

> **STATUS: BACKLOG STUB.** This story is a placeholder created per drbothen/vsdd-factory#260
> rollback so drift items have a concrete target ID. Acceptance criteria, file structure,
> task list, and architecture mapping will be fleshed out when the story is scheduled into
> a wave.

## Narrative

- **As a** session endpoint producing channel frames (via halfchannel.HalfChannel.Tick)
- **I want to** serialize them into wire frames combining the 44-byte outer header (per ARCH-02 §3.1) and the 12/20-byte channel header (per ARCH-02 §3.2 / BC-2.01.005)
- **So that** routers and receivers can forward and decode the bytes per the canonical protocol

## Drift items consumed

This story is the target for three drift items from wave-1 adversary pass-01:

| Drift ID | Severity | Scope | Notes |
|----------|----------|-------|-------|
| wave-adv F-001 | MED | Payload-MTU contract gap | **Spec side already addressed** in BC-2.01.002 v1.4 PC5 (burst A, commit `6c064d9`). Code-side Enqueue validation in F-001+F-002 refactor PR (separate cycle, lands before S-2.01). This story inherits the invariant on the encode side: outer-assembler must compute `OuterHeader.PayloadLen = channel_header_size + len(ChannelFrame.Payload)` per ARCH-02; relies on halfchannel having enforced len(payload) <= MaxPayloadSize. |
| wave-adv F-003 | LOW | No composed wire-format test | Add a cross-module test that: takes a ChannelFrame from halfchannel.Tick(), assembles it via the outer-assembler, encodes to bytes, parses back, asserts field-for-field equivalence including FrameType, ChanID, ChanSeq, Flags, Payload. AC for this story. |
| wave-adv F-004 | LOW (per-story-scope) | ARCH-02 channel-header serializer | Implement encode/decode of the 12/20-byte channel header per ARCH-02 §3.2. Owner decision (architect, when scheduled): does this live in `internal/frame` (canonical wire-format codec) or `internal/channelheader` (sibling package)? `architecture_modules` frontmatter lists both as candidates. |

## Inputs (when scheduled)

- ARCH-02 protocol-stack — outer header (§3.1) + channel header (§3.2) layouts
- BC-2.01.004 — outer-header codec contract (now v1.2+ after burst A invariant 3 fix)
- BC-2.01.005 — channel-header layout contract
- wave-1 adversary pass-01 — origin of the 3 drift items consumed
- Future BCs to be added for outer-assembler behavior (TBD when scheduled)

## When to schedule

Wave 3 or later, after:
- S-2.01, S-2.02, S-1.03 complete (Wave 2 establishes admission + session continuity primitives that the outer-assembler must respect)
- Architect decides the channel-header serializer package home

## Acceptance criteria

TBD — to be defined when story moves out of backlog. Anchored to the drift items above plus any newly-surfaced needs from wave-2/wave-3 work.

## Tasks

TBD.

## File Structure Requirements

TBD.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-06-24 |
| Origin | wave-1 adversary pass-01 + drbothen/vsdd-factory#260 rollback |
| Status transitions | (none yet) |
