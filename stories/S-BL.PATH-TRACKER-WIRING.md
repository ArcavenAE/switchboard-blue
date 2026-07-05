---
artifact_id: S-BL.PATH-TRACKER-WIRING
document_type: story
level: ops
story_id: S-BL.PATH-TRACKER-WIRING
title: "wire cmd/switchboard/metrics_wire.go pathTrackerSource to real routing-subsystem registry"
status: merged
producer: product-owner
timestamp: 2026-07-01T12:00:00
modified: 2026-07-05T22:45:00
phase: 2
epic: E-6
wave: 7
priority: P1
scope_phase: E
estimated_points: TBD
bc_traces:
  - BC-2.06.003
vp_traces: [VP-047, VP-062]
subsystems: [quality-observability, multipath-forwarding]
architecture_modules: [cmd/switchboard, internal/metrics, internal/paths]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-W5.04, S-BL.ROUTER-ADDR]
blocks: [S-BL.PATH-FAILED-STATUS]
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/decisions/wave-6-tranche-a-scope-rulings.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
acceptance_criteria_count: 0
revision: "0.1-backlog-stub"
backlog_origin:
  source: wave-6-tranche-a-scope-rulings Ruling-6 (S-W5.04 Pass-3 L3)
  ruling: Ruling-6
  drift_items_consumed:
    - DRIFT-SW504-PATH-TRACKER-WIRING
  notes: >
    S-W5.04 (daemon-side paths/metrics handlers) implements the handler surface,
    response types, and adapter interface. Production pathTrackerSource population
    — enumerating (SVTN, endpoint) → PathTracker at handler-serve time by wiring
    into the real routing-subsystem registry — was deferred from S-W5.04 to this
    story per Ruling-6. S-W5.04 carries a #DEFERRED: S-BL.PATH-TRACKER-WIRING
    comment in cmd/switchboard/metrics_wire.go at the point where the real source
    would be wired. This story is the concrete backlog anchor for that obligation.
    Depends on S-BL.ROUTER-ADDR so that PathSnapshot.RouterAddr is populated before
    wiring the tracker.
---

# S-BL.PATH-TRACKER-WIRING: Wire Production PathTracker Source in metrics_wire.go

> **STATUS: BACKLOG STUB.** This story is a placeholder created per wave-6-tranche-a-scope-rulings
> Ruling-6 so that the deferred production `pathTrackerSource` population has a concrete target ID.
> Acceptance criteria, file structure, task list, and architecture mapping will be fleshed out when
> the story is scheduled into Wave 7.

## Narrative

- **As an** operator running `sbctl paths list`
- **I want to** see live per-path RTT and loss metrics populated from the real routing subsystem
- **So that** BC-2.06.003 PC-1 is fully satisfied with production data, not test-only stubs

## Ruling consumed

| Ruling | Source | Description |
|--------|--------|-------------|
| Ruling-6 | wave-6-tranche-a-scope-rulings.md | Production PathTracker wiring deferred from S-W5.04 to Wave-7 backlog. S-W5.04 defines the adapter interface; this story wires the real source. Supersedes Ruling-3's "delete empty stubs" for the production-population step. |

## Scope

S-W5.04 delivered:
- `PathTrackerSource` adapter interface in `internal/metrics`
- Handler surface (`paths.list`, `router.metrics`, `router.status`) wired to an injected source
- Test-only `PathTracker` population in test helpers
- `#DEFERRED: S-BL.PATH-TRACKER-WIRING` comment in `cmd/switchboard/metrics_wire.go`

The remaining obligation per BC-2.06.003 PC-1 is:
1. At handler-serve time, enumerate (SVTN, endpoint) → PathTracker by consulting the real routing-subsystem registry (post-S-BL.ROUTER-ADDR, PathSnapshot.RouterAddr is populated).
2. Wire the real source into `metrics_wire.go`, replacing the `#DEFERRED` stub.
3. Remove the `#DEFERRED` comment once wired.
4. Integration tests must verify that live routing-subsystem metrics flow through to `sbctl paths list` response.

## Depends on

- `S-W5.04` — adapter interface must exist before this story can wire the real source
- `S-BL.ROUTER-ADDR` — `PathSnapshot.RouterAddr` must be populated (non-empty `host:port`) before wiring is meaningful

## Blocks

- `S-BL.PATH-FAILED-STATUS` — production tracker must be wired before liveness-signal (`failed` status) work begins

## When to schedule

Wave 7, after:
- S-W5.04 merged (adapter interface exists)
- S-BL.ROUTER-ADDR merged (RouterAddr field populated)
- Architect confirms the routing-subsystem registry API for enumerating PathTrackers

## Acceptance criteria

TBD — to be defined when story moves out of backlog. At minimum must include:
- `cmd/switchboard/metrics_wire.go` `#DEFERRED` comment removed; real source wired
- Integration test: `sbctl paths list` returns live RTT/loss data from routing subsystem
- BC-2.06.003 PC-1 verified end-to-end with production data source

## Tasks

TBD.

## File Structure Requirements

TBD. Candidate files:
- `cmd/switchboard/metrics_wire.go` (remove stub, wire real source)
- Possible new helper in `internal/metrics` if registry enumeration is nontrivial

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-01 |
| Origin | wave-6-tranche-a-scope-rulings Ruling-6 (S-W5.04 Pass-3 L3 F-L3-006) |
| Drift item | DRIFT-SW504-PATH-TRACKER-WIRING — this stub is the concrete backlog anchor |
| Status transitions | (none yet) |
