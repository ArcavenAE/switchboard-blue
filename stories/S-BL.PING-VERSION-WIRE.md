---
artifact_id: S-BL.PING-VERSION-WIRE
document_type: story
level: ops
story_id: S-BL.PING-VERSION-WIRE
version: "1.0"
title: "Ping + version wire handlers: connectivity smoke-test and version info RPC"
status: backlog
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

> **Status:** Backlog stub. Full story decomposition required at scheduling time.
> Product-owner decision required at delivery: implement ping handler or remove the
> sbctl case arm.

## Context

`cmd/sbctl/main.go` dispatches two wire commands that no daemon currently handles:

- **`version`** (main.go line 80): `sbctl` dispatches `version` to retrieve daemon
  build info and compares it against the sbctl build version, printing a warning on
  mismatch. As of develop@7fe3e29e, no daemon registers a `version` handler; the call
  returns `E-RPC-010: unknown command: version`. BC-2.07.002 EC-004 anchors the
  intended behavior ("Daemon returns version info; sbctl prints warning if version
  differs; command may still succeed if protocol is compatible.").

- **`ping`** (main.go line 82): `sbctl` dispatches `ping` as a connectivity
  smoke-test. As of develop@7fe3e29e, no daemon registers a `ping` handler; the call
  returns `E-RPC-010: unknown command: ping`. There is no BC anchor for `ping`
  today (EC-005 in BC-2.07.002 v1.7 documents the gap).

## Obligations

### version handler (anchored by BC-2.07.002 EC-004)

Register a `version` handler in the daemon that returns structured build info
(e.g., `{"version": "v0.3.1", "commit": "7fe3e29e", "built": "2026-07-01T00:00:00Z"}`
or equivalent). The sbctl side already knows how to compare versions and print a
warning on mismatch — the daemon side is the missing half.

### ping handler (product-owner decision required)

At delivery time, the implementer must choose one of:

1. **Implement**: register a trivial `ping` handler that returns `{"pong": true}` (or
   equivalent). This is the smallest possible connectivity smoke-test surface and has
   zero behavioral complexity.
2. **Remove**: delete the `ping` case arm from `cmd/sbctl/main.go`. Ping was
   never part of any BC; if no use case is identified, removing the dead code is
   cleaner than implementing a handler for it.

The product-owner makes this decision at scheduling time. Until then, both paths
are in scope.

## Deliverables

1. `version` daemon handler registration returning build info; sbctl version-compare
   path exercised end-to-end.
2. Either `ping` daemon handler (`{"pong": true}`) OR removal of the `ping`
   case arm from sbctl, depending on product-owner decision.
3. Integration test: verify `sbctl version` no longer returns `E-RPC-010`.

## Refs

- Phase 5 Pass 2 Adv-A F-P5P2-A-001 (`version` wire orphan finding)
- Phase 5 Pass 2 Adv-A F-P5P2-A-002 (`ping` wire orphan finding)
- BC-2.07.002 v1.7 EC-004 (version annotation), EC-005 (ping annotation)

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-02 | Backlog stub created. Bundles version + ping wire handler obligations. Refs Phase 5 Pass 2 F-P5P2-A-001, F-P5P2-A-002. |
