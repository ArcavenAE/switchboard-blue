---
artifact_id: S-BL.SVTN-LIST-WIRE
document_type: story
level: ops
story_id: S-BL.SVTN-LIST-WIRE
version: "1.0"
title: "SVTN list wire boundary: sbctl svtn list + admin.svtn.list handler"
status: backlog
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

> **Status:** Backlog stub. Full story decomposition required at scheduling time.

## Context

BC-2.07.002 v1.6 added a `PENDING-S-BL.SVTN-LIST-WIRE` annotation to the Canonical
Test Vectors table (happy-path row: `sbctl svtn list` with registered key → "List of
SVTNs returned"). As of develop@7fe3e29e, the `sbctl svtn list` subcommand dispatches
to wire command `svtn.list`, for which no daemon currently registers a handler. Any
invocation returns `E-RPC-010: unknown command: svtn.list` regardless of authentication
state, making the happy-path test vector's postcondition unreachable through the shipped
operator surface. This story closes that gap.

## Obligation

Register a wire handler for the `svtn.list` command (or `admin.svtn.list` — see
Alternative Surface below) in the daemon's management-plane admin surface. The preferred
approach mirrors the existing pattern in `cmd/switchboard/admin_handlers.go`, which
already registers `admin.svtn.create` and `admin.svtn.destroy`. Adding `admin.svtn.list`
alongside them keeps all SVTN lifecycle operations on the same admin socket with
consistent authentication and authorization semantics (operator OpenSSH key auth per
BC-2.07.002 PC-2/PC-3).

## Alternative Surface Option

If listing SVTNs is better exposed on a different socket (e.g., a read-only status
endpoint rather than an admin-scoped operation), the handler may be registered as
`svtn.list` on a non-admin wire surface, provided the implementation satisfies
BC-2.07.002 PC-2 (operator key auth) and returns a JSON-serializable list of SVTN
identifiers and their current state. The choice between `admin.svtn.list` and
`svtn.list` is a product-owner/architect decision at scheduling time.

## Deliverables

1. Daemon handler registration for `admin.svtn.list` (or `svtn.list`) returning the
   list of active SVTNs in JSON form.
2. Optional: `sbctl svtn list` subcommand wiring — confirm the existing sbctl dispatch
   path reaches the new handler end-to-end.
3. Integration test: verify that `sbctl svtn list` with a registered key returns the
   SVTN list (not `E-RPC-010`), satisfying BC-2.07.002 Canonical Test Vectors
   happy-path row.

## Refs

- Phase 5 Pass 1 Adv-A F-P5P1-A-001 (SVTN list orphan finding)
- BC-2.07.002 v1.6 PENDING-S-BL.SVTN-LIST-WIRE annotation in Canonical Test Vectors

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-02 | Backlog stub created. Closes phantom reference from BC-2.07.002 v1.6 PENDING-S-BL.SVTN-LIST-WIRE annotation. Refs Phase 5 Pass 1 F-P5P1-A-001. |
