---
artifact_id: S-BL.SESSIONS-LIST-WIRE
document_type: story
level: ops
story_id: S-BL.SESSIONS-LIST-WIRE
version: "1.0"
title: "sessions.list wire: console-facing RPC handler over the mgmt wire exposing discovery.Enumerate()"
status: draft
producer: story-writer
timestamp: 2026-07-14T00:00:00Z
modified: 2026-07-14T00:00:00Z
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: E
estimated_points: TBD
bc_traces:
  - BC-2.03.002
  - BC-2.03.001
depends_on:
  - S-7.02
blocks: []
acceptance_criteria_count: 0
provenance:
  origin: "S-BL.DISCOVERY-WIRE Forward Obligation (d) — story-ready human gate disposition, 2026-07-14"
  spec_annotation: "BC-2.03.002.md v1.4 Postcondition 1 PENDING-S-BL.DISCOVERY-WIRE annotation — to be re-pointed to this story's ID by product-owner in a companion burst (not performed by this stub)"
  adjudication: "follow-on story (S-7.02 cycle-closing checklist disposition (a)) — explicitly OUT of S-BL.DISCOVERY-WIRE per its own Non-Goals section + Forward Obligation (d)"
---

# S-BL.SESSIONS-LIST-WIRE: sessions.list Wire — Console-Facing RPC Handler over the Mgmt Wire

> **STATUS: DRAFT BACKLOG STUB.** Acceptance criteria, file structure, task list, and
> architecture mapping will be fleshed out when this story is scheduled. No architect
> ruling adjudicates this RPC yet — architect elaboration is required before decomposition.

## Context

`internal/discovery.Enumerate()` (implemented under S-7.02, merged PR #55) satisfies
BC-2.03.002's "internal Go API ... or equivalent API call" clause today, but nothing
exposes it over the management wire. `sbctl sessions list` dispatches wire command
`sessions.list`, for which no daemon handler is registered.

`BC-2.03.002.md` v1.4 Postcondition 1 carries a `PENDING-S-BL.DISCOVERY-WIRE` annotation
(added Phase 5 Pass 1, F-P5P1-A-002, closing DRIFT-P5P1-A002-SESSIONS-LIST-ORPHAN)
anticipating this wire exposure would land as part of the DISCOVERY-WIRE story. During
`S-BL.DISCOVERY-WIRE`'s story-ready human gate (2026-07-14), Forward Obligation (d)
confirmed none of the three architect rulings (`S-BL.DISCOVERY-WIRE-rulings.md` v1.8)
adjudicate a console-facing enumeration RPC — Rulings 1/2/3 scope is exclusively the
UDP-multicast advertisement transport (hop-1 ingest + hop-2 relay), not `sessions.list`.
`S-BL.DISCOVERY-WIRE`'s own Non-Goals section names this explicitly:

> `sbctl sessions list` / `sessions.list` RPC wire exposure — BC-2.03.002 Postcondition
> 1's `PENDING-S-BL.DISCOVERY-WIRE` annotation anticipates a console-facing enumeration
> RPC, but none of the three rulings adjudicate one. This story's scope is exclusively
> the advertisement transport (hop-1 ingest + hop-2 relay). Flagged as Forward
> Obligation (d) — a distinct scope question for PO/architect, not silently absorbed
> here.

Human disposition on 2026-07-14: this is a follow-on story (checklist disposition (a),
per the S-7.02 cycle-closing discipline) rather than an extension of DISCOVERY-WIRE's
scope.

This stub is that follow-on story. It closes BC-2.03.002 Postcondition 1's
`PENDING-S-BL.DISCOVERY-WIRE` annotation once scheduled — **product-owner will re-point
that annotation from `S-BL.DISCOVERY-WIRE` to `S-BL.SESSIONS-LIST-WIRE` in a companion
burst; this stub does not itself edit `BC-2.03.002.md`.**

## BC Anchors

| BC | Why anchored |
|----|-------------|
| BC-2.03.002 | Postcondition 1's `PENDING-S-BL.DISCOVERY-WIRE` annotation is the spec-side trigger for this story — "or equivalent API call" is satisfied by `discovery.Enumerate()` today; the CLI-form / wire-RPC form is the gap this story closes. |
| BC-2.03.001 | `discovery.Enumerate()`'s registry model (admitted-node session state) is the data source this RPC handler reads from; no new discovery-domain behavior is added, only a wire-boundary exposure of existing state. |

## Scope (at scheduling time)

1. Register a `sessions.list` handler in the daemon's mgmt-wire dispatch table (the
   `BuildAdminHandlers`-equivalent registration point for non-admin RPCs — exact
   location TBD by architect elaboration).
2. Handler marshals `discovery.Enumerate()`'s result set to the wire response shape.
3. Confirm/wire `sbctl sessions list` CLI dispatch against the `sessions.list` wire
   command name (verify against the current `cmd/sbctl` dispatch table).
4. Determine authorization scope for `sessions.list` — not adjudicated by any existing
   ruling.
5. Unit and integration tests traced to BC-2.03.002 Postcondition 1.

## Open Design Obligations (must be resolved before scheduling)

### 1. Authorization scope for `sessions.list`

No architect ruling adjudicates whether `sessions.list` requires the same
admission-authenticated-connection trust boundary as other mgmt-wire RPCs, or a
narrower/broader authority class. BC-2.03.002 does not specify an authority model for
enumeration. Needs an architect ruling or a BC-2.03.002 amendment before implementation.

### 2. Response shape and pagination

`discovery.Enumerate()`'s in-memory result set has no defined wire serialization or
pagination contract. For large admitted-node counts an unbounded response may be
undesirable. Needs elaboration at scheduling time — not adjudicated here.

### 3. Interaction with S-BL.DISCOVERY-WIRE's real-socket transport

`discovery.Enumerate()` currently reflects whatever the in-process registry holds
(S-7.02's in-process trigger model). Once `S-BL.DISCOVERY-WIRE`'s real-socket UDP
transport ships, `Enumerate()`'s result set will additionally include state populated
by real advertisement traffic. This story's wire exposure is agnostic to that — it
reads whatever the registry holds — but implementers should confirm no ordering
dependency is introduced. `depends_on` currently lists only `S-7.02` (MERGED), not
`S-BL.DISCOVERY-WIRE` (still draft), reflecting Forward Obligation (d)'s framing of
this as a distinct scope question, not a DISCOVERY-WIRE sub-scope.

## Provenance

- **Origin:** `S-BL.DISCOVERY-WIRE.md` Forward Obligations table, row (d) —
  BC-2.03.002 Postcondition 1's `PENDING-S-BL.DISCOVERY-WIRE` annotation anticipated a
  `sessions.list` wire handler that none of the three architect rulings
  (`S-BL.DISCOVERY-WIRE-rulings.md` v1.8) adjudicate.
- **Disposition:** story-ready human gate for `S-BL.DISCOVERY-WIRE`, 2026-07-14 —
  follow-on story (checklist disposition (a), S-7.02 cycle-closing discipline),
  explicitly OUT of `S-BL.DISCOVERY-WIRE`'s scope per its own Non-Goals section.
- **Spec annotation:** `BC-2.03.002.md` v1.4 Postcondition 1's
  `PENDING-S-BL.DISCOVERY-WIRE` annotation is the spec-side closure target —
  product-owner will re-point it to this story's ID (`S-BL.SESSIONS-LIST-WIRE`) in a
  companion burst. This stub does not itself edit `BC-2.03.002.md`.
- **Status:** stays `draft` — no architect ruling exists yet; architect elaboration is
  required before ACs/tasks/files can be decomposed (see Open Design Obligations).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-14 | Backlog stub created per S-BL.DISCOVERY-WIRE Forward Obligation (d) and its story-ready human gate disposition (follow-on story, checklist disposition (a)). BC-2.03.002 Postcondition 1's `PENDING-S-BL.DISCOVERY-WIRE` annotation is the spec-side trigger; re-pointing that annotation to this story's ID is deferred to a companion product-owner burst (not performed here). No architect ruling adjudicates authorization scope or response-shape questions yet; full decomposition deferred to scheduling time. |
