---
artifact_id: BC-2.06.002
document_type: behavioral-contract
level: L3
version: "1.3"
status: draft
producer: product-owner
timestamp: 2026-06-29T00:00:00
phase: 1a
bc_id: BC-2.06.002
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
      F-001 disposition (1): PC-3 "Gap events are recorded in path metrics" reworded
      to make explicit that the missCount counter increment IS the gap-event record.
      No new accessor or external telemetry export required at this layer. VP-052
      Verification Properties row updated to match. Observability surface deferred
      per DRIFT-001. S-5.01 adversarial convergence finding.
  - date: 2026-06-29
    version: "1.3"
    actor: product-owner
    change: >
      Task 4 reconverge (S-5.01 + S-6.02 Pass-1 adversarial, lens3 F-002):
      Stories cell updated from "[filled by story-writer]" to S-5.01 + S-7.03 trace.
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
traces_to: [CAP-021]
kos_anchors:
  - elem-timeslice-framing
---

# Behavioral Contract BC-2.06.002: Missing Expected Frame Is a Degradation Signal Triggering Indicator Downgrade

## Description

The timeslice framing model guarantees one frame per tick. When a frame that was expected (based on the tick interval and last received sequence) does not arrive within the timeout window, this absence is treated as a degradation signal. The console's quality indicator is updated to reflect the detected gap. This is the mechanism that makes the quality indicator meaningful for non-data sessions (where empty-tick frames carry the liveness signal).

## Preconditions

1. A half-channel has an established sequence and a known tick interval.
2. The last received frame's timestamp and sequence number are known.
3. A "frame expected by" timeout is computed: last_received + tick_interval + jitter_budget.

## Postconditions

1. If a frame does not arrive by the expected-by time, the missing frame is recorded as a gap event.
2. After N consecutive gap events (implementation: N=3), the quality indicator degrades one level (green→yellow or yellow→red).
3. Each gap event is recorded by incrementing the QualityIndicator's internal missing-frame counter (missCount). This counter IS the path-metric record of the gap event — no separate accessor or external telemetry export is required at this layer. (Export to an operator-visible surface, e.g. `sbctl sessions status`, is deferred to the observability story per DRIFT-001.)
4. When frames resume, the gap count decreases; quality indicator recovers after M consecutive good frames (implementation: M=3).

## Invariants

1. **DI-008**: This mechanism depends on the timeslice clock always firing. If the sender skips empty ticks, the receiver incorrectly detects gaps (false degradation). DI-008 violation breaks this BC.
2. Gap events are not errors — they are quality signals. A gap does not close the session.
3. The jitter budget must be at least 2× the tick interval to avoid false positives under OS scheduling jitter (per ASM-002).

## Trigger

"Frame expected by" timer fires without receiving the expected frame.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-010) | Path is degraded but not failed; empty-tick frames arrive intermittently | Quality indicator correctly degrades based on gap frequency. Not a false negative: the indicator reflects actual path quality. |
| EC-002 | Frame is delayed beyond expected-by time but arrives later | Gap event recorded; if N gaps triggered indicator downgrade, the late frame does not retroactively undo it. Quality recovery requires M consecutive good frames. |
| EC-003 | Router drops exactly every 10th frame (patterned loss) | 10% loss rate → quality indicator yellow after 3 missed frames. |
| EC-004 | Both paths deliver frames reliably but one path has high RTT | High-RTT path frames arrive late but before the expected-by timeout (set per path RTT budget). No false gap on the slow path. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| 3 consecutive frames missing | Quality indicator degrades one level | happy-path |
| 3 frames missing then 3 frames received | Quality indicator degrades then recovers | edge-case |
| 1 frame late (arrives 50ms after expected-by); otherwise healthy | Gap event recorded; insufficient for indicator change (N=3) | edge-case |
| DI-008 violated: sender skips empty ticks | False gap events; quality indicator degrades incorrectly | property (violation scenario) |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-052 | N=3 consecutive gaps trigger indicator downgrade | unit |
| VP-052 | M=3 consecutive good frames trigger indicator recovery | unit |
| VP-052 | Each OnMissingFrame() call increments missCount (the internal gap-event record); verified indirectly by TestMissingFrame_GreenToYellow streak-count assertions | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-021 ("Per-session quality indicator (green/yellow/red)") per capabilities.md §CAP-021 |
| L2 Domain Invariants | DI-008 (timeslice clock fires whether or not there is data — absence is a signal) |
| Architecture Module | internal/metrics |
| Stories | S-5.01 (QualityIndicator + OnMissingFrame implementation in internal/metrics), S-7.03 (operator-visible export of missCount to sbctl sessions status / console session-list — deferred per DRIFT-002) |
| Capability Anchor Justification | CAP-021 ("Per-session quality indicator (green/yellow/red)") per capabilities.md §CAP-021 — this BC specifies the "missing frame is a degradation signal" mechanism that CAP-021 defines as "a missing frame is a degradation signal" |

## Related BCs

- BC-2.01.002 — depends on: empty-tick frames are the liveness signals this BC monitors
- BC-2.06.001 — composes with: gap events are additional input to the quality indicator computation
