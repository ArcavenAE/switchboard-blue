---
artifact_id: BC-2.06.003
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.06.003
subsystem: SS-TBD
capability: CAP-022
priority: P1
criticality: important
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
traces_to: [CAP-022]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.06.003: Per-Path RTT and Loss Metrics Queryable via sbctl

## Description

Operators can query per-path latency and loss metrics via `sbctl` from both the node side (router connection quality as seen by the node) and the router side (forwarding metrics as seen by the router). This supports the diagnostic use case: distinguishing a network problem (high RTT on a specific path) from an application problem (high CPU on the access node). Metrics are reported in structured JSON and human-readable format.

## Preconditions

1. The target daemon (router, access node, or console node) is running and reachable by sbctl.
2. sbctl is authenticated (the operator's key is registered against the SVTN).

## Postconditions

1. `sbctl paths list` returns a list of all active paths for the node, each with: path ID, remote router address, current RTT (ms), p99 RTT (ms), loss rate (%), status (active/degraded/failed).
2. `sbctl router metrics --svtn=<id>` returns per-SVTN forwarding metrics: frame count, HMAC failure count, drop cache hit count, per-path frame distribution.
3. Metrics are returned as JSON with `--json` flag; human-readable table by default.
4. If the daemon is unreachable, sbctl returns E-NET-001 "daemon unreachable" (per BC-2.07.003).

## Invariants

1. Metrics reflect observed measurements, not configuration targets.
2. Metrics do not include session content, keystroke counts, or any user data.
3. Router-side metrics are aggregated per SVTN, not per node, to preserve SVTN isolation semantics.

## Trigger

Operator runs `sbctl paths list` or `sbctl router metrics`.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Node has no active paths | Returns empty path list with status "no active paths". Not an error. |
| EC-002 | Operator requests per-node breakdown on router | Returns per-SVTN aggregates only; no per-node breakdown (per-node data could enable traffic analysis). |
| EC-003 | Metrics not yet computed (node just started) | Returns available metrics; fields not yet measured are null or marked as "pending". |
| EC-004 | Operator requests historical metrics (trend data) | Out of scope for E router phase. Current implementation returns point-in-time metrics only. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl paths list` on a node with 2 active paths | JSON: [{path_id, router_addr, rtt_ms:15, rtt_p99_ms:22, loss_pct:0.1, status:active}, {path_id, router_addr, rtt_ms:45, rtt_p99_ms:68, loss_pct:0.0, status:active}] | happy-path |
| `sbctl paths list` with no active paths | JSON: [] with message "no active paths" | edge-case |
| `sbctl router metrics --svtn=abc123` | JSON: {frame_count, hmac_fail_count, drop_cache_hits, path_distribution} | happy-path |
| `sbctl paths list --json` on unreachable daemon | E-NET-001 "daemon unreachable: <address>"; exit code 1 | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Metrics output contains no session content or keystroke data | code-audit |
| VP-TBD | JSON output is valid JSON for all input combinations | fuzz |
| VP-TBD | RTT metrics reflect actual measured values (not configured targets) | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-022 ("Per-path latency and loss metrics via CLI") per capabilities.md §CAP-022 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — metrics contain no content) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-022 ("Per-path latency and loss metrics via CLI") per capabilities.md §CAP-022 — this BC specifies the `sbctl` interface for the per-path metrics that CAP-022 defines as available for both node-side and network-operator-side views |

## Related BCs

- BC-2.02.003 — depends on: per-path metrics collected here are the data source
- BC-2.07.003 — composes with: sbctl connection error handling is shared
