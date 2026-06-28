---
artifact_id: BC-2.02.002
document_type: behavioral-contract
level: L3
version: "1.2"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.002
subsystem: multipath-forwarding
architecture_module: internal/multipath
capability: CAP-005
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - version: "1.2"
    date: 2026-06-28
    author: product-owner
    reason: "RULING-001: update EC-004 to document 32-bit wrap as outside MVP scope; add EC-005 for seq=0 reservation"
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/edge-cases.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-005]
kos_anchors:
  - elem-dual-fastest-path-forwarding
---

# Behavioral Contract BC-2.02.002: Receiver Delivers First-Arriving Copy and Silently Discards Subsequent Duplicates

## Description

When the same frame arrives at the receiver more than once (a consequence of duplicate-and-race forwarding or multi-hop routing loops), the first-arriving copy is delivered to the application layer and all subsequent copies with the same sequence number are silently discarded. Retransmits carry different content (new data in the same sequence slot for upstream replay; ARQ retransmit for downstream) and are not suppressed.

## Preconditions

1. The receiver has an active channel with a known sequence space for the sending half-channel.
2. The frame has a valid outer header (HMAC verified by router) and valid channel header.

## Postconditions

1. The first-arriving copy of a frame (identified by sequence number) is delivered to the application layer.
2. Any subsequent frame with the same sequence number from the same channel is discarded without error, without ACK side-effects.
3. Delivery order is maintained: frame with sequence N is not delivered before frame N-1 (unless the half-channel is upstream idempotent replay, where ordering is not guaranteed and deduplication is the primary function).
4. A retransmit carrying different content in the same sequence slot (upstream replay) is not suppressed — it is applied per the idempotent replay semantics.

## Invariants

1. **DI-009**: First arrival wins. This invariant is enforced at the receiver, not the router.
2. Deduplication window must cover the maximum expected inter-path delivery difference (implementation: at least 1 second of sequence history).

## Trigger

Frame arrives at receiver endpoint.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-009) | Router drop cache detects routing loop duplicate | Router discards at router level before forwarding. Receiver also deduplicates if one copy passes. |
| EC-002 | Duplicate arrives 500ms after original on slow path | If within deduplication window, discarded silently. If outside window (implementation edge): treated as out-of-order frame per the half-channel's sequence handling. |
| EC-003 | Upstream idempotent replay: same keystroke sequence arrives twice | Both copies pass deduplication if they differ in content (replay window content). Receiver applies idempotent dedup by keystroke sequence, not frame sequence. |
| EC-004 | Frame sequence number wraps around (overflow) | Sequence space must be large enough to prevent overlap within the duplication window. Implementation: 32-bit sequence space; at 10ms tick rate, wrap takes ~497 days. **32-bit wraparound across an active session is outside MVP scope** (RULING-001 §R2). Sessions are assumed to terminate before the wrap boundary. Receiver-side comparison loops (including cumulative-ACK scan in OnAck) need not handle the MaxUint32→1 transition for MVP. Implementations SHOULD add a doc comment at wrap-adjacent comparisons citing this assumption. |
| EC-005 | Frame arrives on wire with chan_seq=0 | chan_seq=0 is reserved and never a valid frame sequence number (RULING-001 §R1). The receiver MUST discard the frame without delivery or ACK side-effects. This applies to both replay deduplication and ARQ reorder-buffer insertion. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame seq=42 arrives on path A at t=0ms; same frame arrives on path B at t=8ms | Frame delivered at t=0ms; second copy at t=8ms discarded silently | happy-path |
| Frame seq=42 arrives twice within 1ms on same path | Second copy discarded; no duplicate delivery | happy-path |
| Frame seq=42 duplicate arrives 2s after original (outside dedup window) | Treated as unexpected late frame; logged; discarded | edge-case |
| Two frames: seq=42 with content "abc", seq=43 with content "def" | Both delivered in order; no suppression | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-024 | Receiver delivers first-arriving copy, silently discards second | integration |
| VP-054 | First-arriving copy delivered; identical duplicate discarded silently with no ACK side-effects | integration |
| VP-025 | Deduplication window bounded (drop cache never exceeds capacity; ≥1s history covered by bounded cache) | proptest |
| VP-054 | Discarded duplicates produce no ACK side-effects (verified in VP-054 harness ackRecorder assertion) | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-005 ("Dual-path frame forwarding with duplicate-and-race") per capabilities.md §CAP-005 |
| L2 Domain Invariants | DI-009 (receiver deduplication: first arrival wins) |
| Architecture Module | internal/multipath |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-005 ("Dual-path frame forwarding with duplicate-and-race") per capabilities.md §CAP-005 — this BC specifies the receiver-side deduplication that is the essential complement to the dual-path dispatch in BC-2.02.001 |

## Related BCs

- BC-2.02.001 — depends on: duplicate-and-race dispatch creates the duplicates this BC handles
- BC-2.02.009 — related to: router-level drop cache handles loop duplicates before they reach receiver
