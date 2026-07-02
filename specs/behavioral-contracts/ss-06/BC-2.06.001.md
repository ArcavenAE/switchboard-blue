---
artifact_id: BC-2.06.001
document_type: behavioral-contract
level: L3
version: "1.7"
status: draft
producer: product-owner
timestamp: 2026-07-02T00:00:00
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
modified:
  - date: 2026-06-29
    version: "1.2"
    actor: product-owner
    change: >
      VP table disambiguated. Removed dual VP-027 rows (unit+proptest) that created
      ambiguity. VP-027 row retained for proptest (monotone-transition property).
      New VP-074 (unit) added for threshold classification correctness. L-001 finding.
  - date: 2026-06-29
    version: "1.3"
    actor: product-owner
    change: >
      Task 4 reconverge (S-5.01 + S-6.02 Pass-1 adversarial, F-C3 / lens3 F-001):
      (1) PC-3/PC-4 OR-form precedence note added: Red takes precedence over Yellow
      when inputs satisfy both band predicates simultaneously. (2) Stories cell
      updated from "[filled by story-writer]" to S-5.01 + S-5.02 + S-7.03 trace.
  - date: 2026-06-30
    version: "1.4"
    actor: product-owner
    change: >
      S-5.02 Pass-3 Ruling 1 (F-C1 / F-T3-005): Drop S-5.02 sbctl PC-5 trace from
      Stories cell. The sbctl half of PC-5 surfacing is deferred to S-7.03 alongside
      the console half — both halves now land in S-7.03. The tautological AC-007 test
      (TestSbctlSessionsStatus_QualityFieldPresent) is dropped from S-5.02 scope.
      Stories cell updated to reflect S-5.01 + S-7.03 only.
  - date: 2026-07-01
    version: "1.5"
    actor: spec-steward
    change: >
      F-P8L3-002 (MED): Traceability Stories cell extended — added S-W5.04 (quality
      serialization on wire for router.status). S-W5.04 AC-005a references BC-2.06.001
      for the green/yellow/red state machine over p99 RTT and loss thresholds. The
      `pending` fourth state referenced by AC-005a derives from BC-2.06.003 EC-007
      (SampleCount<10 precedence rule), NOT from BC-2.06.001.
  - date: 2026-07-01
    version: "1.6"
    actor: product-owner
    change: >
      F-P9L3A-01 (attribution cleanup): Corrected v1.5 modified-list entry to remove
      semantic mis-anchoring. BC-2.06.001 defines only the ternary green/yellow/red
      state machine; the `pending` state (SampleCount<10 precedence rule) belongs
      exclusively to BC-2.06.003 EC-007. The v1.5 phrasing "green/yellow/red/pending
      state machine" implied BC-2.06.001 governs pending — it does not.
  - date: 2026-07-02
    version: "1.7"
    actor: spec-steward
    change: >
      F-P2L3-02: Retract S-7.03 reverse-trace from Stories cell per RULING-W6TB-C
      §2 DRIFT-001b. PC-5 console-half surfacing moved from S-7.03 to S-BL.CONSOLE-OBS.
      Stories cell updated: S-7.03 reference replaced with S-BL.CONSOLE-OBS (future
      owner; anchor move authorized by ruling).
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
2. Green: best path RTT p99 ≤ 100ms AND loss ≤ 5%.
3. Yellow: best path RTT p99 in (100ms, 500ms] OR loss in (5%, 20%].
4. Red: best path RTT p99 > 500ms OR best path loss > 20% OR no paths available.

   **Precedence note (F-C3):** When inputs simultaneously satisfy both Yellow (PC-3) and Red (PC-4) predicates — e.g., RTT=600ms and loss=10% — Red takes precedence over Yellow. The implementation evaluates Red first; if any Red condition holds, the indicator is Red regardless of Yellow conditions. This matches the implementation in `internal/metrics` and codifies the OR-form precedence for single-band vs multi-band inputs.
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
| VP-074 | (RTT, loss) → {Green, Yellow, Red} threshold classification is exact; enum cardinality = 3; all boundary values correct | unit |
| VP-027 | Degradation transitions only downward (green→yellow→red); no red→green skip | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-021 ("Per-session quality indicator (green/yellow/red)") per capabilities.md §CAP-021 |
| L2 Domain Invariants | DI-008 (timeslice clock fires — empty ticks are liveness probes) |
| Architecture Module | internal/metrics |
| Stories | S-5.01 (QualityIndicator internal/metrics implementation), S-BL.CONSOLE-OBS (console + sbctl PC-5 surfacing — deferred per RULING-W6TB-C §2 DRIFT-001b; previously anchored to S-7.03 per DRIFT-001, moved to S-BL.CONSOLE-OBS at S-7.03 v1.2), S-W5.04 (quality serialization on wire for router.status) |
| Capability Anchor Justification | CAP-021 ("Per-session quality indicator (green/yellow/red)") per capabilities.md §CAP-021 — this BC specifies the computation that CAP-021 defines as "derived from measured path latency and loss" |

## Related BCs

- BC-2.02.003 — depends on: per-path RTT and loss metrics are the inputs
- BC-2.06.002 — composes with: missing frame events are additional input to quality computation
