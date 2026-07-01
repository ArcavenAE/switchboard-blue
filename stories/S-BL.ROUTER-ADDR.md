---
artifact_id: S-BL.ROUTER-ADDR
document_type: story
level: ops
story_id: S-BL.ROUTER-ADDR
title: "populate PathSnapshot.RouterAddr with real resolved host:port (BC-2.06.003 PC-1)"
status: backlog
producer: product-owner
timestamp: 2026-07-01T12:00:00
phase: 2
epic: E-6
wave: backlog
priority: P1
scope_phase: E
estimated_points: TBD
bc_traces:
  - BC-2.06.003
vp_traces: [VP-047, VP-062]
subsystems: [quality-observability, multipath-forwarding]
architecture_modules: [internal/metrics, internal/paths, cmd/sbctl]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-W5.04]
blocks: [S-BL.PATH-TRACKER-WIRING]
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/decisions/wave-6-tranche-a-scope-rulings.md'
acceptance_criteria_count: 0
revision: "0.1-backlog-stub"
backlog_origin:
  source: wave-6-tranche-a-scope-rulings Ruling-1 + DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER
  ruling: Ruling-1
  drift_items_consumed:
    - DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER
  notes: >
    BC-2.06.003 v1.9 (Ruling-1) permits `router_addr: ""` as an interim sentinel while
    PathSnapshot is not enriched with a real resolved host:port. Consumers MUST treat ""
    as a valid sentinel meaning "address not yet resolved." When this story ships,
    router_addr will always be a non-empty `host:port` string. Blocking on S-BL.PATH-TRACKER-WIRING
    which needs a populated router_addr to make path enumeration meaningful.
---

# S-BL.ROUTER-ADDR: Populate PathSnapshot.RouterAddr with Real Resolved host:port

> **STATUS: BACKLOG STUB.** This story is a placeholder created per wave-6-tranche-a-scope-rulings
> Ruling-1 (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER) so that the deferred `router_addr` population
> has a concrete target ID. Acceptance criteria, file structure, task list, and architecture mapping
> will be fleshed out when the story is scheduled into Wave 7.

## Narrative

- **As an** operator running `sbctl paths list`
- **I want to** see a non-empty `router_addr` field showing the resolved `host:port` for each path
- **So that** BC-2.06.003 PC-1 `router_addr` semantics are fully satisfied and operators can
  identify which router address each path is associated with

## Ruling consumed

| Ruling | Source | Description |
|--------|--------|-------------|
| Ruling-1 | wave-6-tranche-a-scope-rulings.md | `router_addr: ""` is a valid Wave-6 interim sentinel. Consumers MUST NOT treat it as an error. `router_addr` will be a non-empty `host:port` when this story ships. DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER is the tracking drift item. |

## Scope

The interim `router_addr: ""` sentinel ships in Wave 6 per Ruling-1. When this story ships:
1. Enrich `PathSnapshot` to carry a resolved `host:port` string for the remote router.
2. Update the `PathTracker` to accept/store router address at construction or registration time.
3. The `router_addr` field in `sbctl paths list` output MUST be a non-empty `host:port` string.
4. Remove or update the DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER drift annotation in BC-2.06.003 PC-1.
5. VP-062 fuzz seeds for `router_addr=""` remain valid (harness asserts key presence, not non-empty value) — but seed documentation should note this is no longer the production case.

## Depends on

- `S-W5.04` — PathSnapshot and PathEntry types must be defined before enrichment

## Blocks

- `S-BL.PATH-TRACKER-WIRING` — path enumeration benefits from a populated router_addr

## When to schedule

Wave 7, after:
- S-W5.04 merged (PathSnapshot type defined)
- Architect confirms how router address is resolved and when it becomes available (connection time vs. routing time)

## Acceptance criteria

TBD — to be defined when story moves out of backlog. At minimum must include:
- `sbctl paths list` returns non-empty `router_addr` values for all active paths
- BC-2.06.003 PC-1 `router_addr` field description updated: `""` sentinel removed
- DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER resolved

## Tasks

TBD.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-01 |
| Origin | wave-6-tranche-a-scope-rulings Ruling-1 + DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER (S-W5.04 Pass-1) |
| Drift item closed | DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER (this stub is the concrete backlog anchor) |
| Status transitions | (none yet) |
