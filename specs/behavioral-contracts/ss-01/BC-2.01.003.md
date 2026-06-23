---
artifact_id: BC-2.01.003
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.01.003
subsystem: session-networking
architecture_module: internal/halfchannel
capability: CAP-002
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
traces_to: [CAP-002]
kos_anchors:
  - elem-asymmetric-half-channels
---

# Behavioral Contract BC-2.01.003: Upstream and Downstream Half-Channels Operate with Independent Clocks and Sequence Spaces

## Description

Each terminal session channel consists of two independent half-channels: upstream (console-to-access-node, carrying keystrokes) and downstream (access-node-to-console, carrying terminal output). These half-channels have completely independent tick clocks, sequence number spaces, and loss recovery strategies. An event on one half-channel (loss, delay, backpressure) does not affect the other.

## Preconditions

1. A channel is established between a console and an access node.
2. Both the upstream and downstream half-channels have been initialized with their respective configured tick intervals.
3. The upstream half-channel uses the idempotent replay strategy (CAP-007).
4. The downstream half-channel uses ARQ with ACK/SACK (CAP-008).

## Postconditions

1. The upstream half-channel's sequence number space is independent of the downstream sequence number space.
2. A loss on the upstream half-channel does not retrigger the downstream half-channel's recovery mechanism.
3. A downstream ARQ retransmit does not affect the upstream tick schedule.
4. Each half-channel's empty-tick clock fires on its own schedule.
5. Path selection for upstream and downstream may differ (the best path for upstream may not be best for downstream).

## Invariants

1. **DI-001**: Carrier-grade content separation. Channel headers (carrying sequence state) are opaque to routers; routers see only outer headers.
2. Upstream sequence space starts at 0 on channel establishment; downstream sequence space starts at 0 independently.
3. Upstream recovery is idempotent replay; downstream recovery is ARQ. These strategies are fixed per direction.

## Trigger

Channel establishment; ongoing operation of an active terminal session.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Downstream ARQ retransmit storm while upstream is idle | Upstream continues emitting empty-tick frames at normal interval; downstream manages its own congestion. |
| EC-002 | Upstream tick configured at 10ms; downstream at 50ms | Both clocks run independently. Downstream emits one frame per 50ms; upstream emits one per 10ms. Frame counts diverge as expected. |
| EC-003 | Channel half-closed: upstream shuts down (read-only mode) | Downstream half-channel continues operating normally. Upstream emits no frames but does not shut down the downstream. |
| EC-004 | One path is selected for upstream, a different path for downstream | Both paths carry their respective half-channel traffic independently. Path metrics tracked per direction. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Upstream sends 10 keystrokes; downstream has 2MB of output pending | Upstream ticks at 10ms; downstream ticks at 50ms; sequence spaces diverge; no cross-contamination | happy-path |
| Simulate downstream loss of 5 frames → ARQ triggers | Upstream continues unaffected; downstream retransmits those 5 frames; upstream sequence unaffected | edge-case |
| Upstream configured with idempotent replay N=5; downstream configured ARQ SACK | Both operate independently; upstream window size 5; downstream SACK bitmap active | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Upstream and downstream sequence spaces never share values | proptest |
| VP-TBD | Upstream loss does not trigger downstream retransmit | integration |
| VP-TBD | Each half-channel's tick fires independently | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-002 ("Asymmetric half-channel operation") per capabilities.md §CAP-002 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation), DI-008 (timeslice clock fires whether or not there is data) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-002 ("Asymmetric half-channel operation") per capabilities.md §CAP-002 — this BC specifies the independence invariant between half-channels that CAP-002 defines as "independent sequence spaces, clocks, and recovery strategies" |

## Related BCs

- BC-2.01.001 — depends on: each half-channel has its own instance of the timeslice clock
- BC-2.02.004 — composes with: upstream uses idempotent replay strategy
- BC-2.02.005 — composes with: downstream uses ARQ strategy
