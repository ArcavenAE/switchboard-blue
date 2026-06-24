---
artifact_id: BC-2.06.001
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.06.001
subsystem: quality-observability
architecture_module: internal/metrics
capability: CAP-021
priority: P1
criticality: high
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
  - '.factory/specs/domain-spec/edge-cases.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-021]
kos_anchors:
  - elem-timeslice-framing
---

# Behavioral Contract BC-2.06.001: Quality Indicator (green/yellow/red) Derived from Measured Path Latency and Loss

## Description

Each active session displays a quality indicator (green/yellow/red) at the console. The indicator is derived from measured per-path RTT and loss rate. Green means within latency budget; yellow means degraded but functional; red means significantly degraded. The indicator is updated on every keep-alive measurement cycle. This is the operator's primary signal for session health.

## Preconditions

1. A console has an active session subscription.
2. Per-path RTT and loss metrics are being collected (per BC-2.02.003).
3. Quality thresholds are configured (implementation defaults: green <100ms p99, yellow 100–500ms, red >500ms; or >5% loss → yellow, >20% loss → red).

## Postconditions

1. After each metric update cycle, the quality indicator is recomputed.
2. Green: best path RTT p99 < 100ms AND loss < 5%.
3. Yellow: best path RTT p99 in [100ms, 500ms] OR loss in [5%, 20%].
4. Red: all paths RTT p99 > 500ms OR loss > 20% OR no paths available.
5. The indicator is surfaced via `sbctl sessions status` and in the console's session list view.

## Invariants

1. **DI-008**: The indicator depends on empty-tick frame liveness. If the timeslice clock breaks (DI-008 violation), the indicator becomes unreliable (FM-008).
2. The indicator reflects path quality to the router, not end-to-end terminal quality (access node health is a separate signal).
3. Indicator transitions are hysteretic: a brief spike to yellow does not immediately return to green (implementation: 3-consecutive-measurement hysteresis).
4. The session-aggregated quality indicator is derived from the BEST current path. Per-path scoring (for routing decisions) uses each path's own metrics independently per ARCH-03. These are two distinct computations: this BC governs the aggregated console indicator; ARCH-03 governs per-path scoring for routing.

## Trigger

Keep-alive metric update; empty-tick frame liveness probe result; TLPKTDROP event.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-010) | Path healthy but carrying only empty-tick frames (no payload) | Empty-tick RTT measures path liveness correctly; indicator reflects actual path quality. |
| EC-002 (FM-008) | Keep-alive succeeds but data frames are losing; empty-tick broken | Bug in implementation. DI-008 prevents this: if empty-tick is working, the loss on data path is also loss on probe path. |
| EC-003 (DEC-004) | All paths fail | Red indicator; session frozen. Indicator goes red before session disconnects. |
| EC-004 | Quality spike for 2 measurements then recovers | Hysteresis: indicator stays yellow for 3 measurements, then returns to green after 3 consecutive green measurements. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| 10 probes: RTT=15ms, loss=0% | Indicator: green | happy-path |
| 10 probes: RTT=150ms, loss=3% | Indicator: yellow (RTT in yellow range) | happy-path |
| All paths unavailable for 3 probe cycles | Indicator: red | edge-case |
| RTT spikes to 200ms for 2 cycles then returns to 20ms | Indicator stays yellow (hysteresis); returns to green after 3 green cycles | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-027 | Quality indicator is always one of: green, yellow, red | unit |
| VP-027 | Green threshold: RTT p99 < 100ms AND loss < 5% | unit |
| VP-027 | Hysteresis prevents rapid toggling | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-021 ("Per-session quality indicator (green/yellow/red)") per capabilities.md §CAP-021 |
| L2 Domain Invariants | DI-008 (timeslice clock fires — empty ticks are liveness probes) |
| Architecture Module | internal/metrics |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-021 ("Per-session quality indicator (green/yellow/red)") per capabilities.md §CAP-021 — this BC specifies the computation that CAP-021 defines as "derived from measured path latency and loss" |

## Related BCs

- BC-2.02.003 — depends on: per-path RTT and loss metrics are the inputs
- BC-2.06.002 — composes with: missing frame events are additional input to quality computation
