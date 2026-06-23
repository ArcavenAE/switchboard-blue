---
artifact_id: BC-2.01.001
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.01.001
subsystem: session-networking
architecture_module: internal/halfchannel
capability: CAP-001
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
traces_to: [CAP-001]
kos_anchors:
  - elem-timeslice-framing
---

# Behavioral Contract BC-2.01.001: Timeslice Clock Fires on Every Tick Regardless of Data Availability

## Description

The half-channel transmit clock fires on a fixed periodic interval (the "tick") whether or not there is application data to send. A frame departs on every tick: either a data frame if payload is queued, or an empty-tick frame if no payload is pending. This is the core mechanism that makes liveness detection and latency guarantees possible.

## Preconditions

1. A half-channel is initialized with a configured tick interval in the range 5–50ms.
2. The half-channel is in the "active" state (connected to a router path).
3. The system clock is available and not frozen.

## Postconditions

1. Exactly one frame departs on each tick boundary regardless of whether application data is queued.
2. If application data is queued, it is included in the frame payload.
3. If no application data is queued, an empty-tick frame (zero-payload) is emitted.
4. The tick interval is maintained within ±1ms jitter under normal OS scheduling (ASM-002).
5. The frame sequence number increments by exactly 1 on each tick.

## Invariants

1. **DI-008**: The timeslice clock fires on every tick. An implementation that skips empty ticks violates this invariant.
2. The tick interval is fixed for the lifetime of the half-channel; it does not adapt to network conditions.
3. Upstream and downstream half-channels have independent tick schedules (see BC-2.01.003).

## Trigger

The periodic timer fires on the configured tick interval boundary.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Tick fires while previous frame is still being transmitted | New frame is queued; previous transmission completes first; clock continues on schedule. No tick is skipped. |
| EC-002 | Multiple payloads accumulate between ticks | All pending payloads up to MTU are coalesced into a single frame on the next tick. Overflow is queued for subsequent ticks. |
| EC-003 | OS scheduler delays the tick by > 5ms | Frame departs on OS wakeup. Jitter is recorded in path metrics. Quality indicator may degrade if sustained. Tick is not "caught up" — the next tick fires at the next scheduled boundary. |
| EC-004 | Tick interval configured at 0ms | Startup fails with E-CFG-001 (tick interval must be in [5ms, 50ms]). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Tick fires, 100 bytes of payload queued | Single frame emitted: outer header + channel header + 100-byte payload; sequence incremented | happy-path |
| Tick fires, no payload queued | Single empty-tick frame emitted: outer header + channel header + zero-length payload; sequence incremented | happy-path |
| 10 ticks fire with no payload | 10 empty-tick frames emitted; sequence 1..10; no ticks skipped | property |
| Tick interval set to 3ms | Startup error E-CFG-001: "tick interval 3ms is outside allowed range [5ms, 50ms]" | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-016, VP-018, VP-041, VP-042 | For all tick intervals in [5ms, 50ms], exactly one frame is emitted per tick | proptest |
| VP-016, VP-018, VP-041, VP-042 | Sequence number increments monotonically across all ticks | proptest |
| VP-016, VP-018, VP-041, VP-042 | Empty-tick frames have zero-length payload field | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-001 ("Timeslice-driven frame assembly and transmission") per capabilities.md §CAP-001 |
| L2 Domain Invariants | DI-008 (timeslice clock fires whether or not there is data) |
| Architecture Module | internal/halfchannel |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-001 ("Timeslice-driven frame assembly and transmission") per capabilities.md §CAP-001 — this BC specifies the exact clock behavior that CAP-001 defines as the framing primitive |

## Related BCs

- BC-2.01.002 — composes with: empty-tick frame semantics depend on this clock
- BC-2.06.002 — depends on: missing frame detection depends on tick regularity

## Architecture Anchors

- `.factory/specs/architecture/ARCH-02-protocol-stack.md` — timeslice framing and clock
- `.factory/specs/architecture/ARCH-09-purity-boundary-map.md` — `internal/frame` classified pure-core

## Story Anchor

[S-N.MM — filled by story-writer]

## VP Anchors

- VP-016, VP-018, VP-041, VP-042
