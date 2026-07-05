---
artifact_id: S-BL.PATH-FAILED-STATUS
document_type: story
level: ops
story_id: S-BL.PATH-FAILED-STATUS
title: "re-introduce status: failed for PathSnapshot liveness failures (BC-2.06.003 PC-1)"
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
architecture_modules: [internal/metrics, internal/paths, cmd/sbctl]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-W5.04, S-BL.PATH-TRACKER-WIRING]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/decisions/wave-6-tranche-a-scope-rulings.md'
acceptance_criteria_count: 0
revision: "0.1-backlog-stub"
backlog_origin:
  source: wave-6-tranche-a-scope-rulings Ruling-4 (Wave-6 Tranche A Pass-2)
  ruling: Ruling-4
  notes: >
    BC-2.06.003 v1.10 (Ruling-4) retracted `status: "failed"` from the normative
    Wave-6 vocabulary and reserved it for this future story. Wave-6 implementations
    MUST NOT emit `failed`; conformance tests MUST reject it. When this story ships,
    `failed` re-enters the status enum as the liveness-failure signal
    (≥3 consecutive missed keep-alives → PathSnapshot.Degraded escalation to failed).
    S502-DEFER-3 precedence rule will be extended: failed + SampleCount<10 → quality:"pending".
---

# S-BL.PATH-FAILED-STATUS: Re-Introduce status: "failed" for PathSnapshot Liveness Failures

> **STATUS: BACKLOG STUB.** This story is a placeholder created per wave-6-tranche-a-scope-rulings
> Ruling-4 so that the deferred `status: "failed"` liveness-signal has a concrete target ID.
> Acceptance criteria, file structure, task list, and architecture mapping will be fleshed out when
> the story is scheduled into Wave 7.

## Narrative

- **As an** operator running `sbctl paths list`
- **I want to** see `status: "failed"` for paths with ≥3 consecutive missed keep-alives
- **So that** BC-2.06.003 PC-1 provides the full `{active, degraded, failed}` status vocabulary
  and operators can distinguish liveness failures from degraded-but-alive paths

## Ruling consumed

| Ruling | Source | Description |
|--------|--------|-------------|
| Ruling-4 | wave-6-tranche-a-scope-rulings.md | `status: "failed"` RESERVED in Wave 6. Implementations MUST NOT emit `failed`; conformance tests MUST reject it. Re-introduction deferred to this Wave-7 story. |

## Scope

The `{active, degraded}` vocabulary shipped in Wave 6 is complete for Wave 6 liveness semantics.
When this story ships:
1. Add liveness-failure escalation: ≥3 consecutive missed keep-alives → `PathSnapshot` new state → `status: "failed"` emitted by metrics handler.
2. Extend S502-DEFER-3 precedence rule: `failed` + `SampleCount < 10` (p99 indeterminate) → `quality: "pending"` (orthogonality principle, BC-2.06.003 PC-3).
3. Update fuzz corpus seed 8 in VP-062 harness (currently annotated as Wave-7-only): activate `Degraded=true` + `rttP99Valid=false` seed and add `status:"failed"` assertion.
4. Conformance tests that currently reject `failed` MUST be updated to accept and verify it.

## Depends on

- `S-W5.04` — PathTracker adapter interface and handler surface must exist
- `S-BL.PATH-TRACKER-WIRING` — production tracker must be wired before liveness signals are meaningful

## When to schedule

Wave 7, after:
- S-BL.PATH-TRACKER-WIRING merged (real tracking data flowing)
- Architect confirms liveness-signal design in `internal/paths` / `PathTracker`

## Acceptance criteria

TBD — to be defined when story moves out of backlog. At minimum must include:
- `sbctl paths list` emits `status: "failed"` for paths with ≥3 consecutive missed keep-alives
- S502-DEFER-3 precedence rule verified: `failed` + pending p99 → `quality: "pending"`
- VP-062 fuzz seed 8 activated (no longer Wave-7-annotated)
- BC-2.06.003 PC-1 status vocab `{active, degraded, failed}` fully covered

## Tasks

TBD.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-01 |
| Origin | wave-6-tranche-a-scope-rulings Ruling-4 (Wave-6 Tranche A) |
| Status transitions | (none yet) |
