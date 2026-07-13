---
artifact_id: BC-2.09.001
document_type: behavioral-contract
level: L3
version: "1.3"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/assumptions.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "f13bf15"
extracted_from: null
bc_id: BC-2.09.001
subsystem: deployment-operations
architecture_module: internal/config
capability: CAP-026
priority: P2
criticality: medium
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-07-12
    version: "1.3"
    actor: story-writer
    change: >
      Traceability Stories cell filled: S-7.04-FU-SIGHUP-RELOAD (PC-1 SIGHUP half, historical
      backfill) + S-BL.CLI-SURFACE-COMPLETION (PC-1 RPC-trigger governance addendum) — the
      distinct story-writer pass PO deferred. Governance-only; no PC/AC behavior change.
  - date: 2026-07-12
    version: "1.2"
    actor: product-owner
    change: >
      S-BL.CLI-SURFACE-COMPLETION Ruling 4 (governance-only addendum, no PC/AC
      behavior change): PC-1 gains a clarifying sentence — RPC-triggered reload
      via the `router.reload` wire verb is dispatched through the same `sighupCh`
      channel the SIGHUP OS-signal path consumes; the two triggers are
      code-path-identical from that point forward. Resolves
      DRIFT-HS006-DRAIN-CLI-MISSING (reload half). [governance_leaf: true]
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/assumptions.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-026]
kos_anchors:
  - elem-single-binary-three-modes
  - elem-mvp-scope-single-lan
---

# Behavioral Contract BC-2.09.001: E Router Graduates to PE Mode by Adding Upstream Router Connections in Config

## Description

An E router (no upstream connections) graduates to PE mode by adding upstream router connection entries to its configuration file and reloading. No reinstall, no binary replacement, no rearchitecture. The same binary reads the new config, establishes the upstream router connections, and begins PE operation. Active sessions on the E router are maintained during graduation.

## Preconditions

1. An E router is running with at least one node connected.
2. Upstream router(s) are running and reachable.
3. The operator has added upstream router addresses to the router's config file.

## Postconditions

1. The router reloads its config (SIGHUP or `sbctl router reload`). **RPC-trigger note (Ruling 4, S-BL.CLI-SURFACE-COMPLETION-rulings.md, 2026-07-12, governance-only):** RPC-triggered reload via the `router.reload` wire verb is dispatched through the same `sighupCh` channel the SIGHUP OS-signal path consumes; the two triggers are code-path-identical from that point forward. See `S-BL.CLI-SURFACE-COMPLETION-rulings.md` Ruling 4.
2. The router establishes connections to the configured upstream routers.
3. The router is now a PE router: it has both node-facing and router-facing interfaces active.
4. Active sessions are not interrupted during the config reload.
5. New path options (via upstream routers) become available for path selection.

## Invariants

1. Binary is unchanged; mode is determined solely by config (upstream_routers: [] = E; upstream_routers: [...] = PE).
2. Session state is not lost during config reload.
3. Upstream router connections use the same admission mechanism as node admissions.

## Trigger

Operator adds upstream router entries to config and reloads: `sbctl router reload` or SIGHUP.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Upstream router address in config is unreachable | Router starts in partial PE mode; logs "upstream router <addr> unreachable". Retries in background. Existing sessions unaffected. |
| EC-002 | Config reload while active sessions are running | Sessions maintained; new upstream connections established without interrupting existing paths. |
| EC-003 | Invalid upstream router address format in config | Config validation fails; router refuses to reload; E-CFG-003 "invalid upstream router address: <addr>". |
| EC-004 | Graduated PE router loses all upstream connections | Falls back to E-router-equivalent behavior (single-LAN only). Sessions on remaining node connections continue. Logs warning. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| E router; operator adds upstream_routers: ["10.0.0.1:9090"]; `sbctl router reload` | Upstream connection established; router now PE; existing sessions preserved | happy-path |
| Upstream router unreachable | Router logs warning; runs as partial PE; retries upstream | edge-case |
| Invalid address in config | Config reload rejected; E-CFG-003; router continues as E | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-038 | Sessions not interrupted during config reload | integration |
| VP-038 | PE mode activated when upstream_routers is non-empty | unit |
| VP-038 | Same binary runs in both E and PE modes | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-026 ("E-to-PE router graduation") per capabilities.md §CAP-026 |
| L2 Domain Invariants | DI-004 (all traffic through routers — graduation adds more routers to the graph) |
| Architecture Module | internal/config |
| Stories | PC-1 (SIGHUP reload path): S-7.04-FU-SIGHUP-RELOAD; PC-1 (RPC-triggered reload via `router.reload`, governance addendum only): S-BL.CLI-SURFACE-COMPLETION |
| Capability Anchor Justification | CAP-026 ("E-to-PE router graduation") per capabilities.md §CAP-026 — this BC specifies the "same binary, no reinstall" graduation behavior that CAP-026 defines as the progressive-deployment promise |

## Related BCs

- BC-2.09.003 — related to: config errors discovered on reload use the same error mechanism

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.3 | 2026-07-12 | story-writer | Traceability Stories cell filled: `S-7.04-FU-SIGHUP-RELOAD` (PC-1 SIGHUP half, historical backfill) + `S-BL.CLI-SURFACE-COMPLETION` (PC-1 RPC-trigger governance addendum) — the distinct story-writer pass PO deferred. Governance-only; no PC/AC behavior change. |
| 1.2 | 2026-07-12 | product-owner | S-BL.CLI-SURFACE-COMPLETION Ruling 4 (`S-BL.CLI-SURFACE-COMPLETION-rulings.md`): governance-only addendum — PC-1 gains a clarifying sentence that RPC-triggered reload via the `router.reload` wire verb is dispatched through the same `sighupCh` channel the SIGHUP OS-signal path already consumes; the two triggers are code-path-identical from that point forward. No PC/AC behavior change. Resolves the reload half of `DRIFT-HS006-DRAIN-CLI-MISSING`. [governance_leaf: true — mirrors the POL-005/governance-leaf pattern, e.g. BC-2.07.001.md v1.13] |
| 1.1 | 2026-06-23 | product-owner | Initial draft — E router graduates to PE mode by adding upstream router connections in config. |
