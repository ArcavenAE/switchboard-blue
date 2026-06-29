---
artifact_id: BC-2.02.003
document_type: behavioral-contract
level: L3
version: "1.3"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.003
subsystem: multipath-forwarding
architecture_module: internal/paths
capability: CAP-006
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - 2026-06-27T00:00:00
  - 2026-06-28T00:00:00 # v1.3 — VP-id assignment: add VP-063 as dedicated degraded-flag property for PC-5; no behavioral change
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
traces_to: [CAP-006]
kos_anchors:
  - elem-dual-fastest-path-forwarding
---

# Behavioral Contract BC-2.02.003: Per-Path RTT and Loss Tracked via Keep-Alive Probes; Paths Ranked by Quality

## Description

Each node maintains real-time per-path quality metrics (RTT and loss rate) by sending periodic keep-alive probe frames and measuring round-trip response times. Paths are ranked by these metrics; the two lowest-RTT paths are selected for duplicate-and-race forwarding (BC-2.02.001). Empty-tick frames serve double duty as keep-alive probes (per BC-2.01.002), eliminating the need for a separate probe channel.

## Preconditions

1. The node has at least one active router connection.
2. The keep-alive interval is configured (implementation default: 1 second).
3. Path metrics are initialized with a high-RTT default on first connection (conservative initial ranking).

## Postconditions

1. After each keep-alive round-trip, the path RTT is updated using an EWMA (exponentially weighted moving average).
2. Loss rate is updated when expected keep-alives are not received within the timeout window.
3. Paths are re-ranked by (RTT, loss_rate) in ascending order after each metric update.
4. Path metrics are available for query via sbctl (BC-2.06.003).
5. A path whose RTT exceeds the degradation threshold (implementation: >200ms) is flagged as degraded.
6. A path with > N consecutive missed keep-alives (implementation: N=3) is marked as failed and removed from the active path set. A failed path is re-added to the active path set upon the first successful keep-alive round-trip; its RTT is initialized from the reactivating probe's measured RTT and its loss EWMA resets to 0. Probes continue to be sent to failed paths so that recovery is detected.

## Invariants

1. Path metrics are per-path, not per-SVTN. The same SVTN may have paths with very different RTTs.
2. Path ranking is updated atomically — no frame dispatch sees a partially-updated ranking.
3. Metrics reflect network quality to the router, not end-to-end session quality (which depends on access node as well).

## Trigger

Keep-alive probe frame sent; keep-alive response received; keep-alive timeout.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-003) | Path RTT degrades from 10ms to 250ms | EWMA smoothly transitions ranking; path moves to lower priority. After sustained degradation, path removed from active set. Failover within 2 seconds. |
| EC-002 | All paths degrade simultaneously (DEC-004) | Quality indicator goes red. Sessions remain connected but interactive responsiveness degrades (FM-002). |
| EC-003 | New path added (node connects to second router) | Path initialized with conservative RTT; keep-alives begin; after first measured RTT, path ranked appropriately. |
| EC-004 | Router responds to keep-alive probe with very low RTT due to local cache | RTT measured to router only; this is correct — end-to-end quality requires both path quality and access node health. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Path A RTT=10ms (5 probes), Path B RTT=50ms (5 probes) | Path A ranked #1; Path B ranked #2; both selected for duplicate-and-race | happy-path |
| Path A: 3 consecutive missed keep-alives | Path A removed from active path set; E-NET-004 logged; only Path B used | edge-case |
| New path connects; first probe RTT=unknown | Path initialized at max RTT (conservative); ranked last until measured | happy-path |
| Path RTT spikes to 300ms for 2 probes then recovers | EWMA smooths spike; path briefly degrades in ranking; recovers on good probes | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-026 | Path score ranking is transitive (Score ordering is total and consistent) | proptest |
| VP-040 | Multipath failover: path recovery detected within 2s after RTT drops below threshold | e2e |
| VP-063 | PathTracker.IsDegraded() is true iff EWMA-smoothed RTT > DegradedRTTThresholdMS (200.0 ms); recovery below threshold clears flag | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-006 ("Latency-based path selection and ranking") per capabilities.md §CAP-006 |
| L2 Domain Invariants | DI-008 (empty-tick frames serve as liveness probes) |
| Architecture Module | internal/paths |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-006 ("Latency-based path selection and ranking") per capabilities.md §CAP-006 — this BC specifies how RTT and loss are measured and how paths are ranked, which is exactly what CAP-006 defines |

## Related BCs

- BC-2.01.002 — depends on: empty-tick frames serve as keep-alive probes
- BC-2.02.001 — composes with: path rankings from this BC select paths for duplicate-and-race
- BC-2.06.001 — related to: per-path metrics feed the quality indicator
