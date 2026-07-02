---
artifact_id: S-BL.PING-VERSION-WIRE
document_type: story
level: ops
story_id: S-BL.PING-VERSION-WIRE
version: "1.1"
title: "Ping + version wire handlers: connectivity smoke-test and version info RPC"
status: wont-fix
retired: 2026-07-02
retired_reason: >
  Wire orphan surface removed from BC-2.07.002 v1.8 (Phase 5 Pass 3 Path B
  remediation). The sbctl version and ping case-arm deletion is pending Burst 17
  code-side fix-PR on develop. Story is won't-fix because there is nothing to
  wire up — the operator surface has been withdrawn from spec rather than
  implemented.
producer: product-owner
timestamp: 2026-07-02T00:00:00
modified: 2026-07-02T00:00:00
phase: 2
epic: E-7
wave: backlog
priority: P3
scope_phase: E
estimated_points: 2
bc_traces:
  - BC-2.07.002
depends_on: []
blocks: []
acceptance_criteria_count: 0
---

# S-BL.PING-VERSION-WIRE: Ping + Version Wire Handlers — Connectivity Smoke-Test and Version Info RPC

> **Status:** Won't-fix. Wire orphan surface withdrawn from BC-2.07.002 v1.8 (Phase 5
> Pass 3 Path B remediation). Code-side case-arm deletion pending Burst 17 fix-PR on
> develop. There is nothing to wire up — the operator surface has been removed from spec.

## Context

`cmd/sbctl/main.go` dispatches two wire commands that no daemon currently handles:

- **`version`** (main.go line 80): `sbctl` dispatches `version` to retrieve daemon
  build info and compares it against the sbctl build version, printing a warning on
  mismatch. As of develop@7fe3e29e, no daemon registers a `version` handler; the call
  returns `E-RPC-010: unknown command: version`. BC-2.07.002 EC-004 previously anchored
  the intended behavior.

- **`ping`** (main.go line 82): `sbctl` dispatches `ping` as a connectivity
  smoke-test. As of develop@7fe3e29e, no daemon registers a `ping` handler; the call
  returns `E-RPC-010: unknown command: ping`. BC-2.07.002 EC-005 previously documented
  the gap.

## Why Won't-Fix

Phase 5 Pass 3 fresh-context adversarial review (Adv-A) rejected the annotate-and-track
pattern for shipping public-surface defects of this class. Path B was selected: withdraw
the surface from spec rather than implement it. BC-2.07.002 v1.8 removes EC-004 (sbctl
version) and EC-005 (sbctl ping) rows from the spec. Burst 17 will delete the
corresponding case-arms from `cmd/sbctl/main.go` on a feature branch.

This story has no remaining obligation — the surface it was minted to wire up no longer
exists in spec.

## Refs

- Phase 5 Pass 2 Adv-A F-P5P2-A-001 (version wire orphan finding)
- Phase 5 Pass 2 Adv-A F-P5P2-A-002 (ping wire orphan finding)
- Phase 5 Pass 3 Adv-A F-P5P3-A-002 (wire-orphan rejection of annotate-and-track)
- BC-2.07.002 v1.8 (EC-004 + EC-005 rows removed)
- DRIFT-P5P3-A002-PING-VERSION-WIRE-ORPHAN (spec-side resolved; code-side pending Burst 17)

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-07-02 | Retired won't-fix. Wire orphan surface removed from BC-2.07.002 v1.8 (Phase 5 Pass 3 Path B remediation). Case-arm deletion pending Burst 17 code-side fix-PR on develop. Story won't-fix because there is nothing to wire up — operator surface withdrawn from spec. Closes DRIFT-P5P3-A002-PING-VERSION-WIRE-ORPHAN (spec-side). Refs F-P5P3-A-001, F-P5P3-A-002. |
| 1.0 | 2026-07-02 | Backlog stub created. Bundles version + ping wire handler obligations. Refs Phase 5 Pass 2 F-P5P2-A-001, F-P5P2-A-002. |
