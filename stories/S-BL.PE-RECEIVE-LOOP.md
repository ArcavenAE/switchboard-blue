---
artifact_id: S-BL.PE-RECEIVE-LOOP
document_type: story
level: ops
story_id: S-BL.PE-RECEIVE-LOOP
title: "PE-connection receive/forward loop — route incoming upstream frames through FrameArrivalHandler.OnFrameArrival"
status: backlog
producer: story-writer
timestamp: 2026-07-07T00:00:00Z
version: "0.1-backlog-stub"
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: PE
estimated_points: 5
bc_traces:
  - BC-2.02.008   # PC-3/EC-003 E-FWD-001 exhaustion discharge (postcondition 1 re-anchored from S-7.04-FU-PE-CONNECTOR AC-004)
  - BC-2.06.003   # PC-1 Failed-state via retransmit-driven path exhaustion (S404-OBS-F / S404-LOW-1 live send+forward re-confirmation)
vp_traces: []
subsystems: [deployment-operations, transport-layer]
architecture_modules:
  - internal/upstreamdial   # established TCP connections (dial loop, keepalive) — live connections provided by this dependency
  - internal/routing        # FrameArrivalHandler.OnFrameArrival — where E-FWD-001 lives (on_frame_arrival.go)
  - internal/arqsend        # arqsend.Retransmitter runRouter wiring for sustained-load path
  - cmd/switchboard         # runRouter receive goroutine wiring per PE connection
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on:
  - S-7.04-FU-PE-CONNECTOR   # provides established TCP connections; adversarial cycle active, pre-merge
blocks:
  - S-7.04-FU-DRAIN-WIRE   # DRAIN broadcast needs an operational receive loop to broadcast over PE connections
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.008.md'
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/stories/S-7.04-FU-PE-CONNECTOR.md'
acceptance_criteria_count: 0
backlog_origin:
  source: S-7.04-FU-PE-CONNECTOR
  adjudication: PO adjudication of adversary pass-1 F-P1-002 (AC-004 partial-discharge, class unmet-deps)
  drift_items_consumed:
    - S404-OBS-F   # E-FWD-001 rate-limit LATENT re-confirmation — re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 (unmet-deps)
    - S404-LOW-1   # live-egress re-confirmation (3 LOW + SEC-001) — re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 (same partial-discharge)
  notes: >
    S-7.04-FU-PE-CONNECTOR AC-004 (E-FWD-001 re-confirmation under sustained ARQ retransmit load)
    was partially discharged by PO ruling F-P1-002 (2026-07-07). Postcondition 2 (no spurious
    E-FWD-001 under normal load) was discharged by TestRunRouter_PE_EFWD001ReconfirmationUnderLoad.
    Postcondition 1 (E-FWD-001 fires under sustained path-exhaustion load) was re-anchored here
    because E-FWD-001 is emitted only from routing.FrameArrivalHandler.OnFrameArrival
    (on_frame_arrival.go). The Connector in PE-CONNECTOR dials, bootstraps, and keepalives — it
    has no receive/forward loop over PE connections. arqsend.Retransmitter is also not wired to
    runRouter in that story. A read goroutine per PE connection routing incoming frames through
    routing.FrameArrivalHandler.OnFrameArrival is required before E-FWD-001 can fire on a live
    PE upstream path.

    S404-OBS-F and S404-LOW-1 re-anchor here (from PE-CONNECTOR AC-004 v1.3) because the full
    send+forward path traversal requires both the dial loop (PE-CONNECTOR) and this receive loop.
    Unit-level E-FWD-001 proof already exists: on_frame_arrival_test.go BC-2.02.008 PC-3/AC-007
    covers the behavior in isolation. This story ships the live-daemon integration path.

    This story is the direct prerequisite for S-7.04-FU-DRAIN-WIRE: DRAIN broadcast over PE
    connections is meaningless without an operational receive/forward loop on those connections.
---

# S-BL.PE-RECEIVE-LOOP: PE-Connection Receive/Forward Loop

> **STATUS: BACKLOG STUB.** Created by PO adjudication of adversary pass-1 F-P1-002 on
> S-7.04-FU-PE-CONNECTOR (AC-004 partial-discharge, class unmet-deps). Acceptance criteria,
> file structure, and task list will be fleshed out when the story is scheduled.

## Narrative

- **As an** operator with an active PE router (established upstream connections via
  S-7.04-FU-PE-CONNECTOR)
- **I want** incoming frames from upstream PE connections to be routed through
  `routing.FrameArrivalHandler.OnFrameArrival`
- **So that** the full send+forward path is exercised, E-FWD-001 can fire under
  path-exhaustion load, and the DRAIN broadcast story (S-7.04-FU-DRAIN-WIRE) has a
  meaningful receive loop to build on

## Context

`S-7.04-FU-PE-CONNECTOR` delivered the outbound TCP dial loop: each configured upstream
router address is dialed, a bootstrap frame is written, and the connected-count atomic
tracks live connections. What that story does NOT provide is a receive goroutine per PE
connection that reads incoming frames and routes them through
`routing.FrameArrivalHandler.OnFrameArrival` (`internal/routing/on_frame_arrival.go`).

`E-FWD-001` (split-horizon drop + log event) lives exclusively in `OnFrameArrival`. The
`arqsend.Retransmitter` is not wired into `runRouter` in PE-CONNECTOR. Without a receive
loop and `arqsend` integration, the sustained-load exhaustion path that exercises E-FWD-001
cannot be reached from a live PE daemon.

Unit-level proof already exists: `on_frame_arrival_test.go` carries `BC-2.02.008 PC-3/AC-007`
covering E-FWD-001 behavior in isolation. This story ships the live-daemon integration path.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|-------------|
| BC-2.02.008 PC-3/EC-003 — E-FWD-001 exhaustion discharge (postcondition 1) | BC-2.02.008 / S404-OBS-F | S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (re-anchored, unmet-deps F-P1-002) | To discharge: E-FWD-001 fires under sustained path-exhaustion via live PE connection |
| BC-2.06.003 PC-1 — Failed-state via retransmit-driven path exhaustion | BC-2.06.003 | S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (re-anchored, same partial-discharge) | To discharge: Failed-state emission observable via full send+forward path |
| S404-OBS-F — E-FWD-001 rate-limit re-confirmation | S404-OBS-F | STATE.md row; re-anchored from PE-CONNECTOR AC-004 | Carries over: full send+forward path traversal required; depends on this story |
| S404-LOW-1 — live-egress re-confirmation (3 LOW + SEC-001) | S404-LOW-1 | STATE.md row; re-anchored from PE-CONNECTOR AC-004 | Carries over: same re-confirmation vehicle as S404-OBS-F |

## Sketched Scope

> Scope is illustrative. Exact AC boundaries will be confirmed at scheduling time.

**Receive goroutine per PE connection:** A read goroutine starts for each connection
established by the `upstreamdial.Connector`. Incoming frames are handed to
`routing.FrameArrivalHandler.OnFrameArrival`. The routing handler already exists
(unit-tested via `on_frame_arrival_test.go`); this story wires it to the live daemon path.

**arqsend.Retransmitter wiring:** `runRouter` gains an `arqsend.Retransmitter` instance
for the sustained-retransmit-load test. This is the path that drives path exhaustion and
triggers E-FWD-001 in integration.

**E-FWD-001 exhaustion discharge (AC-004 postcondition 1):** Integration test drives
sustained ARQ retransmit load over an established PE upstream path until path count is
exhausted, asserting E-FWD-001 fires in the router's writer output.

**Full send+forward re-confirmation (S404-OBS-F, S404-LOW-1):** The complete traversal
(send via `arqsend` → forward through PE upstream receive loop → E-FWD-001 under
exhaustion) is the re-confirmation vehicle for both drift anchors.

## When to Schedule

After `S-7.04-FU-PE-CONNECTOR` is merged (established TCP connections are the prerequisite).
`S-7.04-FU-DRAIN-WIRE` is blocked on this story and cannot be scheduled before it.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-07 |
| Origin | PO adjudication F-P1-002; AC-004 partial-discharge from S-7.04-FU-PE-CONNECTOR v1.3 |
| Anchors tracked | BC-2.02.008 PC-3, BC-2.06.003 PC-1, S404-OBS-F, S404-LOW-1 |
| Status transitions | (none yet) |
