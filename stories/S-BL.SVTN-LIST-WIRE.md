---
artifact_id: S-BL.SVTN-LIST-WIRE
document_type: story
level: ops
story_id: S-BL.SVTN-LIST-WIRE
version: "1.1"
title: "SVTN list wire boundary: sbctl svtn list + admin.svtn.list handler"
status: wont-fix
retired: 2026-07-02
retired_reason: >
  Wire orphan surface removed from BC-2.07.002 v1.8 (Phase 5 Pass 3 Path B
  remediation). The sbctl svtn case-arm deletion is pending Burst 17 code-side
  fix-PR on develop. Story is won't-fix because there is nothing to wire up —
  the operator surface has been withdrawn from spec rather than implemented.
producer: product-owner
timestamp: 2026-07-02T00:00:00
modified: 2026-07-02T00:00:00
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: E
estimated_points: 3
bc_traces:
  - BC-2.07.002
depends_on: []
blocks: []
acceptance_criteria_count: 0
---

# S-BL.SVTN-LIST-WIRE: SVTN List Wire Boundary — sbctl svtn list + admin.svtn.list Handler

> **Status:** Won't-fix. Wire orphan surface withdrawn from BC-2.07.002 v1.8 (Phase 5
> Pass 3 Path B remediation). Code-side case-arm deletion pending Burst 17 fix-PR on
> develop. There is nothing to wire up — the operator surface has been removed from spec.

## Context

BC-2.07.002 v1.6 added a `PENDING-S-BL.SVTN-LIST-WIRE` annotation to the Canonical
Test Vectors table (happy-path row: `sbctl svtn list` with registered key → "List of
SVTNs returned"). As of develop@7fe3e29e, the `sbctl svtn list` subcommand dispatches
to wire command `svtn.list`, for which no daemon currently registers a handler. Any
invocation returns `E-RPC-010: unknown command: svtn.list` regardless of authentication
state, making the happy-path test vector's postcondition unreachable through the shipped
operator surface.

## Why Won't-Fix

Phase 5 Pass 3 fresh-context adversarial review (Adv-A) rejected the annotate-and-track
pattern for shipping public-surface defects of this class. Path B was selected: withdraw
the surface from spec rather than implement it. BC-2.07.002 v1.8 removes the sbctl svtn
list canonical row, the EC-004 sbctl version row, and the EC-005 sbctl ping row from the
spec. Burst 17 will delete the corresponding case-arms from `cmd/sbctl/main.go` on a
feature branch.

This story has no remaining obligation — the surface it was minted to wire up no longer
exists in spec.

## Refs

- Phase 5 Pass 1 Adv-A F-P5P1-A-001 (SVTN list orphan finding)
- Phase 5 Pass 3 Adv-A F-P5P3-A-001 (wire-orphan rejection of annotate-and-track)
- BC-2.07.002 v1.8 (canonical row removed)
- DRIFT-P5P3-A001-SVTN-LIST-WIRE-ORPHAN (spec-side resolved; code-side pending Burst 17)

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-07-02 | Retired won't-fix. Wire orphan surface removed from BC-2.07.002 v1.8 (Phase 5 Pass 3 Path B remediation). Case-arm deletion pending Burst 17 code-side fix-PR on develop. Story won't-fix because there is nothing to wire up — operator surface withdrawn from spec. Closes DRIFT-P5P3-A001-SVTN-LIST-WIRE-ORPHAN (spec-side). Refs F-P5P3-A-001. |
| 1.0 | 2026-07-02 | Backlog stub created. Closes phantom reference from BC-2.07.002 v1.6 PENDING-S-BL.SVTN-LIST-WIRE annotation. Refs Phase 5 Pass 1 F-P5P1-A-001. |
