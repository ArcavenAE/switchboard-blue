---
artifact_id: S-W5.04-daemon-paths-metrics-handlers
document_type: story
level: ops
story_id: S-W5.04
title: "daemon-side paths.list / router.metrics / router.status RPC handlers and response types"
status: draft
producer: product-owner
timestamp: 2026-06-30T00:00:00
phase: 2
epic: E-5
wave: 5
# Wave 5 capacity permitting; otherwise Wave 6.
priority: P1
scope_phase: E
estimated_points: 5
version: "1.0"
bc_traces:
  - BC-2.06.003
vp_traces: [VP-047]
subsystems: [quality-observability, network-management]
architecture_modules: [internal/metrics, internal/mgmt]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-5.02, S-W5.01]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
  - '.factory/specs/architecture/ARCH-12-daemon-management-plane.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
acceptance_criteria_count: 0
# Stub minted per Pass-4 Ruling 1 (S-5.02-pass4-scope-ruling.md).
# ACs, tasks, and file structure are TBD at wave-planning promotion.
# story-writer must flesh out this stub into a full story before implementation.
---

# S-W5.04: Daemon-Side paths.list / router.metrics / router.status RPC Handlers

> **Execute:** `/vsdd-factory:deliver-story S-W5.04`

## Origin

Minted 2026-06-30 per S-5.02 Pass-4 Ruling 1
(`decisions/S-5.02-pass4-scope-ruling.md`). S-5.02 delivered the sbctl
client-side surface (cmd/sbctl) and the internal/paths histogram. This story
delivers the daemon-side half: the RPC handler registration and response type
serialization that allows a real daemon to respond to `sbctl paths list`,
`sbctl router metrics --svtn=<id>`, and `sbctl router status --target <router>`.

## Scope

This story owns:

1. **Response types** in `internal/metrics`: `PathsListResponse`, `PathEntry`,
   `RouterMetricsResponse`, `RTTValue` (float64 | "pending" union type).
2. **RPC handler registration** in `internal/mgmt` (or a new `internal/metrics`
   handler file): register `paths.list`, `router.metrics`, and `router.status`
   handlers on the management server.
3. **Wire `PathSnapshot.P99RTTMs` → `rtt_p99_ms`** in JSON output: float64 when
   `SampleCount ≥ 10`, string `"pending"` when < 10 (BC-2.06.003 EC-003).
4. **Wire `PathSnapshot.Degraded` → `status: "degraded"`** in JSON output
   (via `internal/metrics`).
5. **VP-047 integration test**: `sbctl paths list --json` returns paths with
   required fields (`rtt_ms`, `rtt_p99_ms`, `loss_pct`, `status`) present and
   non-null (or `"pending"` for `rtt_p99_ms` when < 10 samples).

This story does NOT own the sbctl client dispatch (owned by S-5.02) or the
internal/paths histogram (owned by S-5.02).

## Behavioral Contracts

| BC | Title | PCs covered |
|----|-------|------------|
| BC-2.06.003 | Per-Path RTT and Loss Metrics Queryable via sbctl | PC-1 (paths.list response), PC-2 (router.metrics response), PC-3 (router.status alias), PC-4 (--json), PC-5 (daemon unreachable) |

## VP Coverage

| VP | Property | Proof Method |
|----|----------|-------------|
| VP-047 | `sbctl paths list --json` returns paths with required fields present and non-null (or `"pending"` for `rtt_p99_ms` when < 10 samples) | integration |

## Scope Boundary

- sbctl client-side dispatch (`cmd/sbctl`): owned by S-5.02.
- internal/paths histogram and `PathSnapshot`: owned by S-5.02.
- `internal/mgmt` server authentication and transport: owned by S-W5.01.
- Router-side `router.metrics` aggregation per SVTN: this story adds the handler;
  the underlying metric counters (frame counts, HMAC failures, drop-cache hits)
  must be confirmed available from existing `internal/routing` state or deferred.

## Previous Story Intelligence (MANDATORY — fill before implementation)

| Story | Key Decisions | Patterns Established | Gotchas Discovered |
|-------|--------------|---------------------|-------------------|
| S-5.02 | sbctl client dispatch wired; rttHistogram + p99() added to internal/paths; PathSnapshot.P99RTTMs populated | JSON envelope shape; qualityFromPathEntry stub-posture | Daemon handlers absent — this is S-W5.04's entry point |
| S-W5.01 | internal/mgmt server wired to all 4 daemon modes; management socket registered | mgmt.Server.Register() is the handler registration surface | Handler names are method strings (e.g. "paths.list") |

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.0 | 2026-06-30 | product-owner | Stub minted per S-5.02 Pass-4 Ruling 1. ACs/tasks TBD at wave-planning. |
