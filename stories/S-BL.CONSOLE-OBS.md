---
artifact_id: S-BL.CONSOLE-OBS
document_type: story
level: ops
story_id: S-BL.CONSOLE-OBS
title: "Console daemon session-list observability: quality indicator + missCount"
status: backlog
producer: story-writer
timestamp: 2026-07-01T00:00:00
version: "0.1-backlog-stub"
phase: 2
epic: E-7
wave: backlog
priority: P1
scope_phase: PE
estimated_points: TBD
bc_traces:
  - BC-2.06.001
  - BC-2.06.002
vp_traces: []
subsystems: [console-operations, quality-observability]
architecture_modules: [internal/session, internal/metrics]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on:
  - S-5.01
  - S-7.03
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.001.md'
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.002.md'
  - '.factory/decisions/RULING-W6TB-C-console-transport.md'
acceptance_criteria_count: 0
backlog_origin:
  source: RULING-W6TB-C-console-transport
  ruling: RULING-W6TB-C
  drift_items_consumed:
    - DRIFT-001b
    - DRIFT-002
  notes: >
    AC-004 (console daemon session-list view with quality indicator, BC-2.06.001 PC-5
    console-half) and AC-005 (QualityIndicator.MissCount() accessor, BC-2.06.002 PC-3)
    were removed from S-7.03 per RULING-W6TB-C. Both require independent design work
    (new console session-list RPC surface; MissCount accessor specification) and do not
    depend on the console remote-control transport (AC-001/002/003). Decoupling them
    reduces S-7.03 to 3 points and prevents undefined-surface scope inflation.
---

# S-BL.CONSOLE-OBS: Console Daemon Session-List Observability

> **STATUS: BACKLOG STUB.** This story is a placeholder created per RULING-W6TB-C
> so that the deferred console observability obligations have a concrete target ID.
> Acceptance criteria, file structure, task list, and architecture mapping will be
> fleshed out when the story is scheduled.

## Narrative

- **As an** operator managing console sessions
- **I want to** see the quality indicator (green/yellow/red) and miss count for each
  active session via the console daemon session-list view
- **So that** I can assess session quality and diagnose degradation without leaving
  the console management surface

## Scope

This story owns the two deferred obligations moved from S-7.03 per RULING-W6TB-C:

1. **Console daemon session-list view with quality field** (BC-2.06.001 PC-5
   console-half, DRIFT-001b): The console daemon's session-list view must include a
   `quality` field for each active session showing the current green/yellow/red
   quality indicator value. The quality value is derived from `QualityIndicator`
   state (from `internal/metrics`, delivered by S-5.01). This is the "console
   session-list view" clause of BC-2.06.001 PC-5 — the complementary sbctl half
   is owned by S-5.02 AC-007.

2. **QualityIndicator.MissCount() accessor on internal/metrics** (BC-2.06.002 PC-3,
   DRIFT-002): Specify and implement `QualityIndicator.MissCount()` or an equivalent
   accessor on `internal/metrics` (S-5.01 addendum). This closes the DRIFT-002
   deferred observability obligation. The `missCount` concept maps to miss-event
   tracking in the quality subsystem; the exact accessor name and surface are TBD
   at scheduling time pending a design pass.

## Deferred Obligations Consumed

| DRIFT | Description | Source story |
|-------|-------------|--------------|
| DRIFT-001b | BC-2.06.001 PC-5 console session-list quality surfacing | Moved from S-7.03 per RULING-W6TB-C |
| DRIFT-002 | BC-2.06.002 PC-3 missCount observability export | Moved from S-7.03 per RULING-W6TB-C |

## Depends On

- `S-5.01` — `QualityIndicator` and `internal/metrics` quality subsystem must be
  merged before the console view or missCount accessor can be designed
- `S-7.03` — console daemon transport pattern must exist before adding an
  observability layer (session-list RPC follows the attach/detach/switch RPC pattern)

## When to Schedule

After S-7.03 is merged. Requires:
- Design pass on console session-list RPC surface (new surface, no precedent)
- Architect confirmation of `QualityIndicator.MissCount()` API shape
- Product-owner sign-off on `sbctl sessions status` missCount exposure scope

## Acceptance Criteria

TBD — to be defined when story moves out of backlog. At minimum must include:
- Console daemon session-list view exposes `quality: green|yellow|red` per active session
- `QualityIndicator.MissCount()` (or equivalent) accessor exists on `internal/metrics`
- `sbctl sessions status` exposes missCount via the new accessor
- BC-2.06.001 PC-5 (console-half) fully satisfied
- BC-2.06.002 PC-3 missCount observability fully satisfied

## Tasks

TBD.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-01 |
| Origin | RULING-W6TB-C (S-7.03 AC-004 + AC-005 deferred) |
| DRIFT items tracked | DRIFT-001b, DRIFT-002 |
| Status transitions | (none yet) |
