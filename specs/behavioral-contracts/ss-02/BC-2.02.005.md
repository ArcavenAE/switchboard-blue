---
artifact_id: BC-2.02.005
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.005
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
  - elem-asymmetric-half-channels
---

# Behavioral Contract BC-2.02.005: Downstream ARQ with Piggybacked ACK and SACK Bitmap

## Description

The downstream half-channel (terminal output from access node to console) uses reliable ordered delivery with automatic repeat request (ARQ). The console piggybacks acknowledgements (ACK) and selective acknowledgements (SACK bitmap) on its upstream frames. The access node detects gaps via the SACK bitmap and retransmits missing frames using new frames carrying old content (QUIC-style — retransmit carries the content, not the original frame's sequence number). This avoids head-of-line blocking and adapts to out-of-order delivery.

## Preconditions

1. The downstream half-channel is active.
2. The console's upstream half-channel is active to carry piggybacked ACKs (if the console is read-only with no payload-bearing upstream, SACK is piggybacked on empty-tick frames per BC-2.01.002 + BC-2.01.005).
3. The ARQ window size is configured (implementation: based on negotiated RTT).

## Postconditions

1. The console sends a cumulative ACK (next expected sequence) and SACK bitmap (received-out-of-order frames) in every upstream frame payload.
2. The access node detects gaps by comparing sent sequence numbers against the SACK bitmap.
3. On gap detection, the access node retransmits the missing content in a new frame with the current send sequence number.
4. The console delivers downstream frames to the terminal in sequence order: gaps held until filled.
5. Retransmit frames carry the original content but a new frame sequence number (QUIC retransmit model).

## Invariants

1. Terminal output is delivered to the console in the order it was produced by the access node.
2. No byte of terminal output is permanently lost within the ARQ window (loss beyond window: TLPKTDROP, see BC-2.02.006).
3. ACK/SACK is not a separate connection — it is piggybacked on the normal upstream half-channel traffic.

## Trigger

Console receives an out-of-order downstream frame; ACK timer fires; access node detects unacknowledged frames.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Console receives frames out of order: seq [1,3,2] | Frames 1 and 3 buffered; gap at 2 noted in SACK; access node retransmits 2; console delivers 1,2,3 in order. |
| EC-002 | ACK piggyback lost (upstream frame lost) | ARQ timer at access node expires; access node resends; console reACKs on next upstream frame. |
| EC-003 (read-only console, no payload-bearing upstream) | Read-only consoles have no payload-bearing upstream half-channel | Read-only consoles piggyback SACK on empty-tick frames produced by the degenerate upstream half-channel (SACK_present=1 in channel header flags, per BC-2.01.002 + BC-2.01.005). Empty-tick frames carry SACK bitmaps when needed; no separate channel is used. |
| EC-004 | SACK bitmap overflow (too many out-of-order frames simultaneously) | Bitmap covers a fixed range; frames outside the range trigger a NACK or rely on the ARQ timeout to retransmit. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Downstream frames: [1,2,3] all arrive in order | Console ACKs 3; terminal receives bytes in order | happy-path |
| Downstream frames: [1,3] arrive; 2 missing | SACK indicates gap at 2; access node retransmits 2; console delivers [1,2,3] | edge-case |
| All downstream frames lost for 500ms | ARQ retransmit on timer; frames delivered on recovery | edge-case |
| Frame 2 retransmit arrives before original (which arrives later) | Original frame 2 deduplicated on arrival; content applied once | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-019, VP-020 | Downstream bytes delivered in order regardless of path reordering | proptest |
| VP-019, VP-020 | Every lost frame within ARQ window is retransmitted exactly once | proptest |
| VP-019, VP-020 | SACK bitmap accurately reflects received/missing frames | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-008 ("Downstream reliable ordered delivery with ARQ") per capabilities.md §CAP-008 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — downstream payload is SSH-encrypted) |
| Architecture Module | internal/arq |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-008 ("Downstream reliable ordered delivery with ARQ") per capabilities.md §CAP-008 — this BC specifies the ARQ mechanism with piggybacked ACK/SACK that CAP-008 defines as the D-A MVP strategy |

## Related BCs

- BC-2.01.003 — depends on: downstream is an independent half-channel
- BC-2.02.006 — composes with: TLPKTDROP fires when ARQ timeout exceeds perception deadline
