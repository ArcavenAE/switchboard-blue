---
artifact_id: BC-2.02.006
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.006
subsystem: multipath-forwarding
architecture_module: internal/arq
capability: CAP-008
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
traces_to: [CAP-008]
kos_anchors:
  - elem-timeslice-framing
  - elem-asymmetric-half-channels
---

# Behavioral Contract BC-2.02.006: TLPKTDROP Terminates Overdue Downstream Frames and Signals Degradation

## Description

When a downstream frame cannot be delivered within the latency perception deadline (TLPKTDROP threshold), the access node terminates that frame's retransmit attempts and sends a TLPKTDROP signal to the console. The console advances its degradation indicator. This prevents the terminal from freezing indefinitely — it tells the user "the network is struggling" rather than hanging silently. The approach is borrowed from the SRT protocol.

## Preconditions

1. A downstream frame is overdue: it has not been acknowledged within the TLPKTDROP threshold.
2. The TLPKTDROP threshold is configured (implementation: 2× the current path RTT budget, minimum 500ms).
3. The access node has attempted at least one retransmit (ARQ per BC-2.02.005).

## Postconditions

1. The access node stops retransmitting the overdue frame.
2. The access node sends a TLPKTDROP control frame to the console, identifying the dropped sequence number range.
3. The console receives the TLPKTDROP signal and advances past the dropped frame (console session stream shows a gap signal, not a freeze).
4. The console's quality indicator moves to yellow or red depending on drop frequency.
5. Session continues: subsequent frames are delivered normally; only the overdue frame's content is abandoned.

## Invariants

1. TLPKTDROP is a quality signal, not a session termination. The session continues.
2. The dropped frame's content is permanently lost at the terminal output level; there is no user-transparent recovery after TLPKTDROP.
3. TLPKTDROP rate is tracked per session for quality indicator computation (BC-2.06.001).

## Trigger

ARQ retransmit timeout exceeds TLPKTDROP threshold.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | TLPKTDROP fires on a frame carrying a large tmux refresh (full-screen redraw) | Console receives TLPKTDROP; displays gap indicator. Subsequent frames contain new content; session recovers on next tmux redraw event. |
| EC-002 | TLPKTDROP fires continuously for 10 seconds | Quality indicator moves to red after first TLPKTDROP; session stays alive. User sees red indicator and degraded output. |
| EC-003 | TLPKTDROP signal itself is lost in transit | Console does not receive the signal; it freezes on the undelivered frame. Mitigated by: TLPKTDROP signal is retransmitted via the ARQ mechanism until ACKed. |
| EC-004 | TLPKTDROP threshold too low (100ms) | Overly aggressive drop; causes perceptible quality indicator degradation on normal LAN jitter. Threshold must be validated against ASM-001 latency budget. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame seq=50 overdue for 500ms; TLPKTDROP threshold=500ms | Access node sends TLPKTDROP(seq=50); console advances; quality indicator degrades | happy-path |
| TLPKTDROP fires; next frame seq=51 arrives immediately | Console processes seq=51 normally; gap at seq=50 noted; session continues | edge-case |
| TLPKTDROP signal lost in transit | TLPKTDROP re-sent on next tick; console receives on second attempt | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-021 | Session does not terminate on TLPKTDROP | integration |
| VP-021 | Console advances past dropped frame on receiving TLPKTDROP | unit |
| VP-021 | TLPKTDROP fires exactly once per overdue frame (not repeated after first fire) | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-008 ("Downstream reliable ordered delivery with ARQ") per capabilities.md §CAP-008 |
| L2 Domain Invariants | DI-008 (timeslice clock always fires — absence of frame is a signal) |
| Architecture Module | internal/arq |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-008 ("Downstream reliable ordered delivery with ARQ") per capabilities.md §CAP-008 — TLPKTDROP is the defined termination mechanism for overdue frames specified within CAP-008's "TLPKTDROP terminates overdue frames with degradation signal" |

## Related BCs

- BC-2.02.005 — depends on: TLPKTDROP fires when ARQ timeout exceeded
- BC-2.06.001 — composes with: TLPKTDROP rate feeds quality indicator computation
