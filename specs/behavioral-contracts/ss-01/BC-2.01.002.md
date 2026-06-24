---
artifact_id: BC-2.01.002
document_type: behavioral-contract
level: L3
version: "1.3"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.01.002
subsystem: session-networking
architecture_module: internal/halfchannel
capability: CAP-001
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-06-24T00:00:00
    reason: "F-002 resolution: postcondition 2 clarified to distinguish ChannelFrame.FrameType (pure-core surface) from OuterHeader.frame_type (outer-assembler responsibility per ARCH-09)"
  - date: 2026-06-24T00:00:00
    reason: "F-007 resolution: added precondition 4 rejecting len(payload)==0 with ErrEmptyPayload; F-008 resolution: PC3 reworded to remove phantom EMPTY_TICK flag-bit reference and clarify discriminator lives in ChannelFrame.FrameType"
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-001]
kos_anchors:
  - elem-timeslice-framing
---

# Behavioral Contract BC-2.01.002: Empty-Tick Frame Is a Valid Liveness Signal

## Description

When the timeslice clock fires and no application payload is available, the half-channel emits an empty-tick frame. This frame has semantic meaning: its presence confirms the half-channel is alive and the path is reachable. Its absence where one was expected is a degradation signal. The quality observability subsystem depends on this invariant to distinguish "path dead" from "no data to send."

## Preconditions

1. A half-channel is active and in the "no-data-pending" state.
2. The timeslice clock fires (per BC-2.01.001).
3. The receiver has a known expected tick interval for this half-channel.
4. **Enqueue rejection (F-007):** `HalfChannel.Enqueue` rejects any call where `len(payload) == 0` (including nil slices and `[]byte{}`), returning `ErrEmptyPayload`. There is no legitimate MVP use case for a zero-byte data frame; the "no-data-pending" state must only be entered by not calling `Enqueue`, not by enqueuing an empty slice. This collapses the nil/empty-slice asymmetry and prevents a silent protocol violation where a queued-but-empty enqueue would produce a `FrameTypeData` frame with zero-length payload.

## Postconditions

1. An empty-tick frame is emitted with zero-length payload.
2. The `ChannelFrame` returned by `HalfChannel.Tick()` carries `FrameType = EMPTY_TICK (0x02)` as a discriminator field. The outer-assembler layer (outside `internal/halfchannel`, per ARCH-09 pure-core boundary) reads this field and sets `OuterHeader.frame_type = 0x02`. The outer header itself is populated fully (version, frame type = EMPTY_TICK, SVTN ID, destination, source, length=0, HMAC) by the outer-assembler, not by `internal/halfchannel`.
3. The channel header is fully populated (chan_id, chan_seq, flags=0). The EMPTY_TICK discriminator does not live in channel-header flags — the channel-header flags field (ARCH-02 §3.2) defines only bit 0=FEC_present, bit 1=ARQ_req, bit 2=SACK_present; no EMPTY_TICK bit exists. The empty-tick discriminator lives in `ChannelFrame.FrameType` (the pure-core surface); the outer-assembler reads this field and sets `OuterHeader.frame_type = 0x02` (EMPTY_TICK) in the wire frame. See BC-2.01.005 for the channel-header layout and BC-2.01.004 for the outer-header frame_type encoding.
4. The frame is forwarded by the router identically to a data frame (same routing path selection).
5. On receipt, the receiver does not surface the empty-tick frame as application data; it uses it only for liveness and path metric updates.

## Invariants

1. **DI-008**: Empty-tick frames are never skipped. An implementation that omits empty-tick frames when no data is pending violates this invariant and breaks quality monitoring.
2. The frame type field distinguishes empty-tick from data frames; the router forwards both the same way.
3. An empty-tick frame increments the sequence number (maintaining the "one frame per tick" invariant of BC-2.01.001).

## Trigger

Timeslice clock fires with empty application data queue.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Path is lossy: some empty-tick frames lost | Receiver counts missed expected ticks. After threshold (implementation: ≥3 consecutive missed ticks), quality indicator degrades. Not an error — this is the detection mechanism. |
| EC-002 | Both endpoints are active but SVTN has been idle for 30 seconds | Empty-tick frames continue to flow. The absence of empty-tick frames would incorrectly signal path failure. |
| EC-003 | Receiver receives empty-tick frame from unexpected sequence | Frame is accepted; sequence gap detection handles ordering (see BC-2.02.005). |
| EC-004 | Empty-tick frame is double-delivered via dual paths | Duplicate suppression (BC-2.02.002) discards the second copy. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| No application data pending for 5 consecutive ticks | 5 empty-tick frames emitted; sequence increments; quality indicator unchanged (path alive) | happy-path |
| 3 consecutive expected empty-tick frames missing at receiver | Quality indicator moves to yellow; TLPKTDROP signal emitted | edge-case |
| Empty-tick frame arrives: receiver checks frame type | Frame type = EMPTY_TICK; payload length = 0; no application data surfaced | happy-path |
| Empty-tick frame HMAC check fails | Frame rejected at router (E-ADM-002); liveness signal not credited | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-053 | K consecutive empty ticks emit K frames with contiguous seq nums, EMPTY_TICK type, zero payload | proptest |
| VP-052 | Missing expected tick within deadline triggers quality indicator downgrade (Green→Yellow, Yellow→Red) | integration |
| VP-016 | Router routes empty-tick frames via same half-channel emit path as data frames (one frame per tick) | proptest |
| VP-018 | HalfChannel emits empty frame when no payload | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-001 ("Timeslice-driven frame assembly and transmission") per capabilities.md §CAP-001 |
| L2 Domain Invariants | DI-008 (timeslice clock fires whether or not there is data); DI-003 (router compromise → availability, not confidentiality) |
| Architecture Module | internal/halfchannel |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-001 ("Timeslice-driven frame assembly and transmission") per capabilities.md §CAP-001 — this BC specifies the semantic meaning of empty-tick frames, which CAP-001 defines as "the frame departs whether full or empty" |

## Related BCs

- BC-2.01.001 — depends on: empty-tick frame is emitted by the timeslice clock
- BC-2.06.002 — composes with: missing-frame degradation signal depends on empty-tick regularity
- BC-2.02.002 — related to: duplicate suppression handles double-delivered empty-tick frames
