---
artifact_id: BC-2.06.003
document_type: behavioral-contract
level: L3
version: "1.5"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.06.003
subsystem: quality-observability
architecture_module: internal/metrics
capability: CAP-022
priority: P1
criticality: high
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - 2026-06-28T00:00:00
  - 2026-06-30T00:00:00
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

1. **[CANONICAL]** `sbctl paths list` returns a list of all active paths for the node, each with the following fields:
   - `path_id` — opaque path identifier (string)
   - `router_addr` — remote router address (host:port)
   - `rtt_ms` — most-recent EWMA RTT sample in milliseconds (float64)
   - `rtt_p99_ms` — p99 of per-path RTT samples over the observation window (float64); computed from the rolling sample buffer maintained by the PathTracker; "pending" (string) if fewer than 10 samples have been collected
   - `loss_pct` — packet loss rate as a percentage (float64, 0.0–100.0)
   - `status` — one of: `active`, `degraded` (RTT > 200ms sustained), `failed` (≥ 3 consecutive missed keep-alives)
2. **[CANONICAL]** `sbctl router metrics --svtn=<id>` returns per-SVTN forwarding metrics: frame count, HMAC failure count, drop cache hit count, per-path frame distribution.
3. **[ALIAS]** `sbctl router status --target <router>` is a convenience alias for `sbctl paths list`. It produces an equivalent per-path listing (same JSON schema as PC-1) with an additional `quality` column (green/yellow/red quality indicator derived from the status + rtt_p99_ms fields). Both commands route through the same underlying query path in `internal/metrics`; there are no divergent code paths. The `--target <router>` flag overrides the default daemon address, equivalent to `sbctl --target <router> paths list`. The alias exists to match the command surface introduced by S-5.02 (F-P8-002 ruling).

   **Pending-p99 quality semantics (F-M3):** When `rtt_p99_ms` is `"pending"` (fewer than 10 samples collected), the `quality` field MUST be emitted as `"pending"` — mirroring the p99 sentinel value. Implementers MUST NOT substitute a default quality value (green/yellow/red) when p99 data is insufficient. `quality: "pending"` is a valid emit value from `cmd/sbctl/router_status.go`. The quality state machine in `internal/metrics` must treat a pending p99 as an indeterminate input, not a green or zero-value input.
4. Metrics are returned as JSON with `--json` flag; human-readable table by default. Both the canonical form and the alias respect `--json`.
5. If the daemon is unreachable, sbctl returns E-NET-001 "daemon unreachable" (per BC-2.07.003).

## Invariants

1. Metrics reflect observed measurements, not configuration targets.
2. Metrics do not include session content, keystroke counts, or any user data.
3. Router-side metrics are aggregated per SVTN, not per node, to preserve SVTN isolation semantics.

## Trigger

Operator runs `sbctl paths list` (canonical), `sbctl router metrics --svtn=<id>` (canonical), or `sbctl router status --target <router>` (alias for `sbctl paths list`).

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Node has no active paths | Returns empty path list with status "no active paths". Not an error. |
| EC-002 | Operator requests per-node breakdown on router | Returns per-SVTN aggregates only; no per-node breakdown (per-node data could enable traffic analysis). |
| EC-003 | Metrics not yet computed (node just started) | Returns available metrics; `rtt_p99_ms` is `"pending"` (string) if fewer than 10 RTT samples have been collected; other fields present with their current values. `rtt_ms` is available after the first keep-alive round-trip. |
| EC-004 | Operator requests historical metrics (trend data) | Out of scope for E router phase. Current implementation returns point-in-time metrics only. |
| EC-005 | Operator uses alias `sbctl router status --target <router>` | Output is identical to `sbctl paths list` plus a `quality` column (green/yellow/red). Exit code, JSON schema, and error handling are identical to the canonical command. There is exactly one code path in `internal/metrics` serving both invocations — the alias is a CLI dispatch shim only. |
| EC-006 | `sbctl router status --target <router>` on a path with fewer than 10 RTT samples | `rtt_p99_ms` is `"pending"` (string) AND `quality` is `"pending"` (string). The quality column MUST NOT be green/yellow/red when p99 is pending; the p99 sentinel propagates to the quality output. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl paths list` on a node with 2 active paths (≥10 probes collected) | JSON: `[{"path_id":"<id>","router_addr":"<host:port>","rtt_ms":15.0,"rtt_p99_ms":22.0,"loss_pct":0.1,"status":"active"}, {"path_id":"<id>","router_addr":"<host:port>","rtt_ms":45.0,"rtt_p99_ms":68.0,"loss_pct":0.0,"status":"active"}]` | happy-path |
| `sbctl paths list` on a node with <10 probes collected | JSON: `[{"path_id":"<id>","router_addr":"<host:port>","rtt_ms":12.0,"rtt_p99_ms":"pending","loss_pct":0.0,"status":"active"}]` | edge-case |
| `sbctl paths list` with no active paths | JSON: `{"paths":[],"message":"no active paths"}`; exit code 0 | edge-case |
| `sbctl router metrics --svtn=abc123` | JSON: `{"frame_count":<n>,"hmac_fail_count":<n>,"drop_cache_hits":<n>,"path_distribution":{<path_id>:<frame_count>}}` | happy-path |
| `sbctl router status --target 127.0.0.1:9000` on a node with 1 active path (alias) | Same JSON as `sbctl paths list` plus `"quality":"green"` field; exit code 0 | happy-path |
| `sbctl paths list --json` on unreachable daemon | E-NET-001 `"daemon unreachable: <address>"`; exit code 1 | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-047 | `sbctl paths list --json` returns paths with required fields (`rtt_ms`, `rtt_p99_ms`, `loss_pct`, `status`) present and non-null (or `"pending"` for `rtt_p99_ms` when < 10 samples) | integration |
| VP-061 | Metrics output contains no session content or keystroke data (DI-001 enforcement) | code-audit |
| VP-062 | JSON output is valid JSON for all CLI input combinations including alias form | fuzz |

Note: VP-047 is the confirmed integration VP for per-path field presence (see `specs/verification-properties/VP-047.md`). VP-061 and VP-062 are Phase 6 hardening properties; not blocking Wave 5 implementation.

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-022 ("Per-path latency and loss metrics via CLI") per capabilities.md §CAP-022 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — metrics contain no content) |
| Architecture Module | internal/metrics |
| Stories | S-5.02 |
| Capability Anchor Justification | CAP-022 ("Per-path latency and loss metrics via CLI") per capabilities.md §CAP-022 — this BC specifies the `sbctl` interface for the per-path metrics that CAP-022 defines as available for both node-side and network-operator-side views |

## Related BCs

- BC-2.02.003 — depends on: per-path metrics collected here are the data source
- BC-2.07.003 — composes with: sbctl connection error handling is shared

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.5 | 2026-06-30 | spec-steward | F-M3: add explicit pending-p99 quality semantics to PC-3 — when `rtt_p99_ms` is `"pending"`, `quality` MUST also be `"pending"` (not green/yellow/red); the quality state machine must treat pending p99 as indeterminate. Add EC-006 documenting this behavior. Note for implementers: `quality: "pending"` is now a valid emit value from `cmd/sbctl/router_status.go`. |
| 1.1 | 2026-06-23 | product-owner | Initial draft with `sbctl paths list` + `sbctl router metrics` canonical surface |
| 1.2 | 2026-06-28 | product-owner | Wave-5 reconciliation: canonicalize `sbctl paths list` + `sbctl router metrics --svtn=<id>`; add `sbctl router status --target <router>` as documented alias (F-P8-002 ruling, S-5.02 alignment); strengthen `rtt_p99_ms` field semantics (p99 of rolling sample buffer, "pending" when <10 samples); add EC-005 for alias; fix VP table (VP-047 was listed three times — now distinct VP-047/VP-TBD-A/VP-TBD-B); expand test vectors with alias vector and pending-state vector |
| 1.4 | 2026-06-29 | state-manager | F-P2-005: fill Stories traceability cell — `[filled by story-writer]` → `S-5.02`. No behavioral change. |
| 1.3 | 2026-06-28 | architect | Assign VP IDs to placeholders: VP-TBD-A → VP-061 (code-audit, DI-001 content-absence); VP-TBD-B → VP-062 (fuzz, JSON well-formedness). No behavioral change. |
