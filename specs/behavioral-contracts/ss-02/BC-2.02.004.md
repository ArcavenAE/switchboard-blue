---
artifact_id: BC-2.02.004
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.004
subsystem: multipath-forwarding
architecture_module: internal/replay
capability: CAP-007
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified: []
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
traces_to: [CAP-007]
kos_anchors:
  - elem-asymmetric-half-channels
---

# Behavioral Contract BC-2.02.004: Upstream Idempotent Replay Window — Each Frame Carries Last N Keystrokes

## Description

The upstream half-channel (keystrokes from console to access node) uses an idempotent replay window for loss recovery. Each outbound upstream frame carries not only the current keystroke(s) but also a replay window of the last N keystrokes. If a frame is lost in transit, the receiver can recover the missing keystrokes from subsequent frames that carry the same replay window content. The receiver deduplicates by keystroke sequence number, making delivery self-healing without explicit retransmit requests.

## Preconditions

1. The upstream half-channel is active.
2. The replay window size N is configured (implementation default: N=5 keystrokes or equivalent bytes; exact policy is architecture decision).
3. The console's keystroke buffer contains pending input.

## Postconditions

1. Each upstream frame's payload includes the current keystroke(s) plus the last N-1 keystrokes (the replay window).
2. The access node deduplicates keystrokes by sequence number: each keystroke is applied exactly once to the tmux session.
3. A frame carrying only replay window content (no new keystrokes) is still emitted on the tick (as an idempotent replay-only frame).
4. Loss of a single frame is self-healing: the next frame's replay window covers the lost content.
5. Loss of N+1 consecutive frames results in a gap that is irrecoverable without retransmit (accepted; treated as permanent loss, session may visually glitch).

## Invariants

1. **DI-001**: The replay window contains keystroke content which is SSH-encrypted end-to-end; it is opaque to routers.
2. Keystroke sequence numbers are monotonically increasing within a channel; duplicates are discarded.
3. The replay window size N is fixed for the lifetime of a channel (not adaptive).

## Trigger

Timeslice clock fires on the upstream half-channel with pending keystrokes.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | User types faster than one keystroke per tick | Multiple keystrokes coalesced into one frame payload; replay window covers all recent keystrokes. |
| EC-002 | Replay window N=5; 6 consecutive frames lost | First 5 keystroke losses self-healed by subsequent frames. Keystroke 1 (6th in the lost sequence) cannot be recovered. Access node notes the gap. |
| EC-003 | Empty replay window (no keystrokes in the last N ticks) | Frame carries empty payload; this is an empty-tick frame from the upstream half-channel's perspective. |
| EC-004 | Access node receives the same keystroke sequence twice (from different paths or replay) | Keystroke deduplication by sequence number ensures exactly-once application. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Keystroke seq=10 (key='a'); replay window=[seq 6–9] | Frame payload contains seq=10 'a' + seq 6–9 replay | happy-path |
| Frame with seq=10 lost; next frame seq=11 carries replay seq 7–10 | Access node recovers seq=10 from replay window in frame seq=11 | edge-case |
| Access node receives seq=10 'a' twice (replay + original delayed) | Keystroke 'a' applied once; second occurrence discarded | happy-path |
| All keystrokes in one burst: seq 1–20 in rapid succession | Frames batch multiple keystrokes; replay window covers last N | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Each keystroke is applied exactly once at access node | proptest |
| VP-TBD | Single frame loss is recovered from next frame's replay window | proptest |
| VP-TBD | Replay window contents are monotonically increasing by sequence | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-007 ("Upstream idempotent replay (U-C sliding window)") per capabilities.md §CAP-007 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — replay content is SSH-encrypted) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-007 ("Upstream idempotent replay (U-C sliding window)") per capabilities.md §CAP-007 — this BC is the full behavioral specification of the U-C replay strategy |

## Related BCs

- BC-2.01.003 — depends on: upstream half-channel is an independent channel
- BC-2.02.002 — related to: receiver deduplication handles replay duplicates
