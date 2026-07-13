---
artifact_id: BC-2.06.004
document_type: behavioral-contract
level: L3
version: "1.2"
status: draft
producer: product-owner
timestamp: 2026-07-12T00:00:00
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "e6efb60"
extracted_from: null
bc_id: BC-2.06.004
subsystem: quality-observability
architecture_module: internal/mgmt
capability: CAP-022
priority: P2
criticality: medium
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

# Behavioral Contract BC-2.06.004: On-Demand Single-Target Reachability Probe via `sbctl paths ping`

## Description

Operators run `sbctl paths ping --router=<addr>` to issue a one-shot reachability and round-trip-latency probe against a specific, arbitrarily-dialed daemon. This is architecturally distinct from `sbctl paths list`/`sbctl router status` (BC-2.06.003): those report historical, keep-alive-derived, EWMA-smoothed per-path metrics accumulated over time by a `PathTracker`; `paths ping` performs no accumulation and no per-path metrics computation — it dials, Tier-1-authenticates, issues a bodyless `paths.ping` RPC, and reports the client-measured round-trip time. Commissioned as a standalone BC (not an extension of BC-2.06.003) per `S-BL.CLI-SURFACE-COMPLETION-rulings.md` Ruling 1, to keep the RPC-name-based audit trail unambiguous between "real path enumeration" (`paths.list`) and "one-shot stopwatch" (`paths.ping`).

## Preconditions

1. The target daemon at `--router=<addr>` is reachable by sbctl and Tier-1-authenticates the operator's key (shared preconditions with BC-2.07.002).

## Postconditions

1. **[CANONICAL]** `sbctl paths ping --router=<addr>` dials `<addr>` directly (overriding `--target`), authenticates, and issues `paths.ping` with empty args (`{}`). On success, the daemon returns `{"pong": true}` and sbctl reports round-trip time in milliseconds measured client-side, from dial-start to response-decode-complete: `{"router": "<addr>", "rtt_ms": <float64>}`.
2. If the daemon is unreachable (before connection), sbctl returns E-NET-001 "daemon unreachable: <address>" (per BC-2.07.003); exit 1.
3. If authentication fails (after connection), sbctl returns E-ADM-010; exit 1.
4. `paths.ping` performs no per-path metrics computation and returns no quality classification (no green/yellow/red field). A connection that succeeds but is slow is NOT an error — `rtt_ms` simply reports a larger value, exactly like `ping(8)`. Quality classification remains exclusively `router.status`'s job (BC-2.06.003 PC-3); `paths ping` does not re-couple the two capabilities.

## Invariants

1. `paths.ping` requires no additional Tier-2 authority beyond the daemon's standard Tier-1 operator-key authentication — the same bar as `paths.list`/`router.metrics`/`router.status`, none of which carry an additional Tier-2 role gate.
2. The response carries no session content, keystroke data, or per-path metrics state (DI-001-equivalent content absence — the wire payload is the literal constant `{"pong": true}`).

## Trigger

Operator runs `sbctl paths ping --router=<addr>`.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Target daemon at `--router=<addr>` is unreachable before connection | E-NET-001 "daemon unreachable: <address>"; exit 1. Same shared connection-error handling as every other sbctl command (BC-2.07.003). |
| EC-002 | Connection succeeds but Tier-1 authentication fails | E-ADM-010; exit 1. No `paths.ping` RPC is dispatched — auth failure occurs before command dispatch. |
| EC-003 | Connection succeeds, authentication succeeds, but the round trip is slow (high latency) | NOT an error. `rtt_ms` reports the measured (larger) value; exit 0. `paths ping` performs no quality classification — there is no green/yellow/red output, unlike `router.status`. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl paths ping --router=127.0.0.1:9090` (daemon reachable, authenticates) | JSON: `{"router":"127.0.0.1:9090","rtt_ms":3.2}`; exit code 0 | happy-path |
| `sbctl paths ping --router=127.0.0.1:9090` (daemon reachable, authenticates, but round trip measures 480ms) | JSON: `{"router":"127.0.0.1:9090","rtt_ms":480.0}`; exit code 0 — high latency reported as a value, not an error, no quality field emitted | edge-case |
| `sbctl paths ping --router=10.0.0.99:9090` (daemon unreachable) | E-NET-001 `"daemon unreachable: 10.0.0.99:9090"`; exit code 1 | error |
| `sbctl paths ping --router=127.0.0.1:9090` (connection succeeds, operator key not authorized) | E-ADM-010; exit code 1 | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD-PING-A | `sbctl paths ping --router=<addr>` reports `rtt_ms` as a float64 and never emits a quality/status classification field, for both fast and slow round trips | integration |
| VP-TBD-PING-B | The `paths.ping` RPC handler performs zero per-path metrics reads/writes (no `PathTracker` interaction) — request args `{}` in, response `{"pong": true}` out, no other side effect | code-audit |

Note: VP IDs are placeholders pending architect assignment — Ruling 1 did not mint a VP for `paths.ping`. Placeholder pattern follows BC-2.06.003 v1.1's original `VP-TBD-A`/`VP-TBD-B` precedent (later assigned VP-061/VP-062 at v1.3).

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-022 ("Per-path latency and loss metrics via CLI") per capabilities.md §CAP-022. **Anchor note:** Ruling 1 deliberately separates `paths.ping`'s RPC mechanism and target scope from BC-2.06.003's accumulated-metrics contract (see Description); this BC nonetheless anchors to the same L2 capability as the closest-fit "operator diagnostic query via sbctl" capability in the domain spec. Flagged for architect/product-owner confirmation — no dedicated CAP was minted by the ruling, and capabilities.md was outside this burst's authorized artifact set. |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — the `{"pong": true}` response carries no session content or keystroke data) |
| Architecture Module | internal/mgmt (new handler, e.g. `mgmt.RegisterPingHandler`, registered from `wireMetricsHandlers`; interpretation — the ruling's Implementation Constraints name `internal/mgmt` as the registration home, not `internal/metrics`/`internal/paths` per the SS-06 ARCH-INDEX registry, since `paths.ping` deliberately does no metrics-package work) |
| Stories | PC-1..PC-4 (all): S-BL.CLI-SURFACE-COMPLETION |
| Capability Anchor Justification | See L2 Capability row above. |

## Related BCs

- BC-2.06.003 — related to: shares the "diagnostic query via sbctl" family and the connection/auth precondition shape, but is deliberately NOT extended by `paths.ping` (Ruling 1) — `paths.list`/`router.status` own accumulated per-path metrics and quality classification; `paths.ping` owns only the one-shot reachability probe.
- BC-2.07.002 — composes with: sbctl connection dial + Tier-1 operator-key authentication is the shared precondition mechanism.
- BC-2.07.003 — composes with: E-NET-001 unreachable-daemon handling is the shared connection-error mechanism.

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.2 | 2026-07-12 | story-writer | Traceability Stories cell filled: `S-BL.CLI-SURFACE-COMPLETION` (PC-1..PC-4, all) — the distinct story-writer pass PO deferred at v1.1 commission. Governance-only; no PC/AC behavior change. |
| 1.1 | 2026-07-12 | product-owner | Initial commission per `S-BL.CLI-SURFACE-COMPLETION-rulings.md` Ruling 1: new BC for the `paths.ping` wire verb (bodyless RTT probe, client-measured round-trip time, no per-path metrics computation, no quality classification). Registered in BC-INDEX under quality-observability / CAP-022. |
