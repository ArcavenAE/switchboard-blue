---
artifact_id: S-BL.ARQ-TX-arq-retransmit-send-wiring
document_type: story
level: ops
story_id: S-BL.ARQ-TX
title: "wire ARQ retransmit-SEND path into router/multipath dispatch (BC-2.02.005 PC-3)"
status: backlog
producer: product-owner
timestamp: 2026-06-28T00:00:00
phase: 2
epic: E-4
wave: backlog
priority: P0
scope_phase: E
estimated_points: TBD
bc_traces:
  - BC-2.02.005
vp_traces: [VP-019, VP-020]
subsystems: [multipath-forwarding]
architecture_modules: [internal/arq, internal/routing]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-4.03]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.005.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
acceptance_criteria_count: 0
revision: "0.1-backlog-stub"
backlog_origin:
  source: S403-H1-DEFER drift item (Wave 4 audit)
  drift_items_consumed:
    - S403-H1-DEFER
  notes: >
    S-4.03 implemented internal/arq (ARQ state machine: OnAck, TLPKTDROP, in-order
    delivery buffer). BC-2.02.005 PC-3 specifies that "on gap detection, the access
    node retransmits the missing content in a new frame with the current send sequence
    number." The ARQ primitive exists post-S-4.03 but the SEND side — wiring ARQ.Retransmit
    into the router/multipath dispatch path so that the access node actually sends the
    retransmit frame over the live path — was deferred as S403-H1-DEFER because it requires
    router wiring that does not exist in S-4.03's scope (internal/arq is pure-core; it has
    no I/O). This story is the concrete backlog anchor for that obligation.
---

# S-BL.ARQ-TX: Wire ARQ Retransmit-SEND Path into Router/Multipath Dispatch

> **STATUS: BACKLOG STUB.** This story is a placeholder created per S403-H1-DEFER drift
> item so the deferred BC-2.02.005 PC-3 retransmit-SEND obligation has a concrete target
> ID. Acceptance criteria, file structure, task list, and architecture mapping will be
> fleshed out when the story is scheduled into a wave (Wave 5+ recommended, after S-4.03
> and S-4.04 are merged and the routing dispatch path is stable).

## Narrative

- **As an** access node
- **I want to** retransmit missing frames (detected via SACK gap) back to the console using the existing multipath dispatch path
- **So that** BC-2.02.005 PC-3 is fully satisfied end-to-end: the ARQ state machine's gap detection triggers an actual frame send, not just an in-memory state update

## Drift item consumed

| Drift ID | Severity | Source | Description |
|----------|----------|--------|-------------|
| S403-H1-DEFER | MED | STATE.md Wave 4 drift table | S-4.03: retransmit-SEND PC-3 deferred to router/multipath wiring story. This story IS that story. |

## Scope

S-4.03 delivered the ARQ state machine (`internal/arq`). The remaining obligation per BC-2.02.005 PC-3 is:

1. When `ARQ.OnAck` detects a SACK gap, it must trigger a retransmit of the missing content.
2. The retransmit must travel through the router dispatch path (not just update in-memory state).
3. The retransmit frame carries the original content but a **new** frame sequence number (QUIC retransmit model, BC-2.02.005 postcondition 5).
4. A wiring test must verify that gap detection → retransmit → frame-on-wire is observable end-to-end (BC-2.02.005 PC-3, VP-019/VP-020).

`internal/arq` is pure-core and must remain so. The send wiring belongs in the effectful layer that calls ARQ methods — either a new `internal/arq/sender.go` adapter or within the routing dispatch. Architect decision when scheduled.

## When to schedule

Wave 5+ after:
- S-4.03 merged (ARQ state machine exists on develop)
- S-4.04 merged (router wiring patterns established)
- Architect confirms whether retransmit-SEND lives in `internal/arq` adapter or `internal/routing`

## Acceptance criteria

TBD — to be defined when story moves out of backlog. At minimum must include:
- End-to-end test: ARQ gap detected → retransmit frame sent via dispatch → console receives in-order delivery
- BC-2.02.005 PC-3 and PC-5 both verified (new seq number on retransmit)

## Tasks

TBD.

## File Structure Requirements

TBD. Candidate files (architect decision):
- `internal/arq/sender.go` (effectful adapter if ARQ pure-core constraint is preserved)
- or wiring in `internal/routing` dispatch path

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-06-28 |
| Origin | S403-H1-DEFER drift item (Wave 4 consistency audit) |
| Drift item closed | S403-H1-DEFER (this stub is the concrete backlog anchor; state-manager should move S403-H1-DEFER to resolved once this file is committed) |
| Status transitions | (none yet) |
