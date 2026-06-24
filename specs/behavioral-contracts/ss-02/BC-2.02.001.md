---
artifact_id: BC-2.02.001
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.001
subsystem: multipath-forwarding
architecture_module: internal/multipath
capability: CAP-005
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
traces_to: [CAP-005]
kos_anchors:
  - elem-dual-fastest-path-forwarding
---

# Behavioral Contract BC-2.02.001: Duplicate-and-Race — Same Frame Sent on Two Fastest Paths Simultaneously

## Description

When a node has two or more connected router paths, each outbound frame is sent simultaneously on the two paths with the lowest measured RTT (the "two fastest paths"). The first copy to arrive at the destination is delivered; the duplicate is discarded. This provides resilience against single-path failure and reduces effective latency to the minimum of the two path RTTs.

## Preconditions

1. The node has at least two active router connections with measured RTT values.
2. Path rankings are current (updated within the last keep-alive interval, per BC-2.02.003).
3. The frame to be sent is a data or empty-tick frame (not a router control frame).

## Postconditions

1. Exactly two copies of the frame are dispatched: one per fastest-ranked path.
2. Both copies carry identical outer headers and channel headers (same sequence number, same HMAC).
3. If only one path is available, the frame is sent on that single path (no error; single-path fallback).
4. If zero paths are available, frame is queued; E-NET-002 (no active paths) is raised after timeout.
5. The path rankings used for dispatch are snapshotted at the moment of dispatch; a rank change mid-burst does not affect frames already queued for dispatch.

## Invariants

1. **DI-009**: The receiver must deduplicate: first arrival wins, subsequent copies discarded.
2. At most two paths are used per frame (not three or more) regardless of how many paths are available.
3. The two chosen paths are the two with lowest current RTT; ties broken by path stability (fewer recent losses).

## Trigger

Frame ready for transmission in the send queue.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Node has exactly one connected router | Frame sent on that router only; no duplicate. Quality indicator notes single-path mode. |
| EC-002 | Node has three connected routers | Frame sent on the two lowest-RTT routers only. The third router does not receive the frame. |
| EC-003 | Both selected paths fail between ranking and dispatch | Frame queued; paths re-ranked on next keep-alive; frame dispatched on next available path(s). |
| EC-004 | The two fastest paths have the same RTT | Tie broken by loss rate; if still tied, either path is acceptable (implementation choice). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| 2 paths: RTT [10ms, 25ms]; 1 frame queued | Frame dispatched on both paths simultaneously | happy-path |
| 3 paths: RTT [10ms, 15ms, 40ms]; 1 frame queued | Frame dispatched on 10ms and 15ms paths only | happy-path |
| 1 path available | Frame dispatched on single path; no error | edge-case |
| 0 paths available for > 5s | E-NET-002 raised; session quality indicator red | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-024 | At most 2 paths used per frame dispatch | unit |
| VP-024 | Selected paths are the two lowest-RTT ranked paths | proptest |
| VP-024 | Frame identical on both paths (same bytes) | unit |
| VP-042 | Keystroke-to-echo: p99 ≤ 100ms | benchmark |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-005 ("Dual-path frame forwarding with duplicate-and-race") per capabilities.md §CAP-005 |
| L2 Domain Invariants | DI-009 (receiver deduplication: first arrival wins) |
| Architecture Module | internal/multipath |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-005 ("Dual-path frame forwarding with duplicate-and-race") per capabilities.md §CAP-005 — this BC is the direct behavioral specification of the duplicate-and-race forwarding strategy |

## Related BCs

- BC-2.02.002 — composes with: receiver must deduplicate what this BC sends
- BC-2.02.003 — depends on: path rankings must be current for correct path selection
