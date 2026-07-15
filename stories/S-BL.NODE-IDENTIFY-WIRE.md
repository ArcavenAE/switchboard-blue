---
artifact_id: S-BL.NODE-IDENTIFY-WIRE
document_type: story
level: ops
story_id: S-BL.NODE-IDENTIFY-WIRE
version: "1.1"
title: "NODE_IDENTIFY wire: connect-time identify handshake binding (SVTNID, NodeAddr) → IfaceID for hop-2 fan-out target resolution"
status: draft
producer: story-writer
timestamp: 2026-07-14T00:00:00Z
modified: 2026-07-15T00:00:00Z
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: E
estimated_points: TBD
bc_traces:
  - BC-2.01.008
depends_on: []
blocks: []
acceptance_criteria_count: 0
provenance:
  origin: "S-BL.DISCOVERY-WIRE Ruling 3(f) Forward Obligation — story-ready human gate disposition, 2026-07-14"
  spec_annotation: "S-BL.DISCOVERY-WIRE-rulings.md v1.9, Ruling 3(f) subsection item (j) — the human gate disposition naming and scoping this story"
  adjudication: "S-BL.DISCOVERY-WIRE-fanout-options.md v1.1 Option 1 selected at the story-ready human gate — Option 1's NODE_IDENTIFY handshake mechanism delivered via Option 3's name-and-schedule-now shape"
---

# S-BL.NODE-IDENTIFY-WIRE: NODE_IDENTIFY Wire — Connect-Time Identify Handshake, Fan-Out Target Resolution

> **STATUS: DRAFT BACKLOG STUB.** Acceptance criteria, file structure, task list, and
> architecture mapping will be fleshed out when this story is scheduled. No architect
> ruling adjudicates this handshake's wire mechanics yet — architect elaboration is
> required before decomposition.

## Context

`S-BL.DISCOVERY-WIRE`'s Ruling 3(f) verified — not invented — that hop-2 fan-out
**target resolution** has no production implementation today: binding a connecting
node's identity (`NodeAddr`) to its live connection's `InterfaceID`/`nodeConn` does not
exist anywhere in `cmd/switchboard`. `admission.AdmitNode` (the primitive that would
produce this binding) has zero production call sites; `sendMap` carries no `NodeAddr`.
This gap gates `S-BL.DISCOVERY-WIRE`'s AC-017 (SVTN-scoped, exclude-originator fan-out
dispatch), AC-018 (relay-dispatch rate cap), and Task 6 (hop-2 fan-out dispatch).

At `S-BL.DISCOVERY-WIRE`'s story-ready human gate (2026-07-14), the human rejected both
originally-offered resolution paths — an unnamed sequencing dependency on "whatever
future story," and a narrow story-local `Router.BindInterface` seam with no identity
signal to call it with — and asked for better options.
`S-BL.DISCOVERY-WIRE-fanout-options.md` v1.0 (six options, grounded directly in the
shipped code) was produced in response; the human selected its **Option 1**: the
`NODE_IDENTIFY` handshake mechanism (Option 1's substance), delivered as this new,
immediately-named, immediately-scheduled companion story (Option 3's shape), rather than
grafted inline into `S-BL.DISCOVERY-WIRE`'s own 8-point scope. See
`S-BL.DISCOVERY-WIRE-rulings.md` v1.9, Ruling 3(f) subsection item (j), and
`S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1's Disposition section for the full record.

This story delivers that handshake. `S-BL.DISCOVERY-WIRE` gains a `depends_on` edge to
this story's ID; its AC-017/AC-018/Task 6 gate on this story by name.

## Mechanism (per the fanout-options document's Option 1 analysis)

Add one new `control_type = 0x04` (`NODE_IDENTIFY`) opcode. Immediately after TCP
connect, before any session-data frame, the connecting node sends a `NODE_IDENTIFY`
frame carrying its Ed25519 public key (or `NodeAddr`, router re-derives via
`frame.DeriveNodeAddress` and looks up the pubkey via `AdmittedKeySet.LookupByPubkey`).
The router responds with a `Challenge` (`admission.GenerateChallenge`, already
implemented) over the same connection; the node replies with a `ChallengeResponse`
(`Sign(node_priv, nonce)`, already the documented wire shape). The router calls the
**existing, already-tested** `admission.AdmitNode(challenge, resp, pubKey, svtnID, ks)`
unmodified. On success, a new `Router.BindInterface(svtnID, nodeAddr, ifaceID)`-shaped
method records `(SVTNID, NodeAddr) → IfaceID` in a small map alongside `nodeConn`.

`admission.AdmitNode`, `GenerateChallenge`, `Challenge`, `ChallengeResponse`,
`LookupByPubkey` are reused verbatim — this story introduces zero changes to
`internal/admission`.

## BC Anchors

| BC | Why anchored |
|----|-------------|
| BC-2.01.008 | New `control_type = 0x04` opcode requires a Postcondition 2 registry-table row addition, the same pattern `DISCOVERY_RELAY = 0x03` used (Ruling 3(g)). Invariant 3 (append-only, sequential assignment) governs the allocation. |

## Scope (at scheduling time)

1. Register a new `control_type = 0x04` (`NODE_IDENTIFY`) case in `route()`'s switch in
   `cmd/switchboard/mgmt_wire.go` (same shape as the existing DRAIN `case 0x01` and
   `DISCOVERY_RELAY` `case 0x03`).
2. Implement a small challenge/response wire codec for the `NODE_IDENTIFY` frame and the
   `Challenge`/`ChallengeResponse` frames carried over it.
3. `onAccept` gains a call-out to send the `Challenge` once a new connection is
   registered, immediately after TCP connect and before any session-data frame is
   accepted.
4. Implement `Router.BindInterface(svtnID, nodeAddr, ifaceID)` (or equivalently-shaped
   method) recording `(SVTNID, NodeAddr) → IfaceID` in a new map alongside `nodeConn`,
   called on `admission.AdmitNode` success.
5. Add the `NODE_IDENTIFY = 0x04` row to `BC-2.01.008`'s Postcondition 2 `control_type`
   registry table.
6. Unit and integration tests: successful handshake, wrong-SVTN, revoked-key, and
   replayed-nonce paths (the last three already covered by `admission.AdmitNode`'s
   existing test suite; the wire-transport wrapper needs its own coverage).

## Open Design Obligations (must be resolved before scheduling)

### 1. `BC-2.01.008` opcode registry amendment

`NODE_IDENTIFY = 0x04` is the next free value after `DISCOVERY_RELAY = 0x03` per
Invariant 3's append-only, sequential-assignment rule — but the registry-table row
itself has not been added. Needs a product-owner/architect amendment to `BC-2.01.008`
before implementation, mirroring Ruling 3(g)'s `DISCOVERY_RELAY` precedent.

### 2. Challenge-transcript wire format

`admission.GenerateChallenge`/`Challenge`/`ChallengeResponse` are implemented as
in-process Go types today; no wire serialization exists for carrying them inside a
`control_type = 0x04` frame. The exact byte layout (challenge nonce encoding, response
signature encoding, frame boundaries) needs architect elaboration — not adjudicated
here.

### 3. Re-identify / rebind semantics

Unspecified: what happens if an already-bound connection sends a second
`NODE_IDENTIFY` frame, or a node reconnects (new TCP connection, same admitted
identity) while a prior `(SVTNID, NodeAddr) → IfaceID` binding is still held. Does
`BindInterface` overwrite the prior binding? Is the prior connection torn down? Needs
an architect ruling before implementation.

### 4. Failure paths

A node that never completes the handshake (bad clock, revoked key, network drop
mid-handshake, wrong-SVTN, replayed nonce) simply never gets bound — the same
fail-closed posture `IsAdmitted` already has elsewhere, per the fanout-options
document's Option 1 "Failure modes" note. The exact observable behavior (does the
connection stay open unbound indefinitely? is there a handshake timeout?) needs
elaboration at scheduling time.

### 5. BLOCKER — `admission.AdmitNode` is verification-only against an always-empty router-mode keyset (`S-BL.DISCOVERY-WIRE-rulings.md` v1.10 Ruling 4, 2026-07-15)

This story's handshake **cannot succeed today, regardless of how faithfully it wires the
existing primitives.** `admission.AdmitNode` is verification-only — it looks up the
connecting node's key in the local `AdmittedKeySet` (`ks.keys[svtnID]`, returning
`ErrNotAdmitted` if absent) rather than admitting a new key. The router-mode process's
own `AdmittedKeySet` is always empty in production: admission writes
(`admin.key.register`, the only production `RegisterKey` caller) happen exclusively in
the separate, disconnected control-mode OS process (`main.go`'s mode switch makes
router/control/console mutually exclusive processes, each constructing its own,
never-synced `AdmittedKeySet`); no cross-process admission-sync mechanism exists
anywhere in the codebase today. Calling `admission.AdmitNode` against a router
process's keyset will therefore fail unconditionally — `ErrNotAdmitted` — no matter how
correctly this story implements the `NODE_IDENTIFY` opcode, the challenge/response wire
codec, or `Router.BindInterface`.

**Prerequisite:** a new follow-on story, **`S-BL.ADMISSION-SYNC-WIRE`** (working name —
not yet created, not multi-option-vetted the way `S-BL.NODE-IDENTIFY-WIRE` itself was;
PO/architect to confirm name + scope), must land admission state reaching the router
process before this story's handshake can be scheduled for implementation. Add
`S-BL.ADMISSION-SYNC-WIRE` to this story's `depends_on` once that story exists — not yet
added here, since it has no ID to add yet (frontmatter `depends_on: []` stays empty this
edit). This obligation is upstream of — and gates — obligations 1-4 above: none of the
opcode/wire-format/rebind/failure-path questions matter until admission state can reach
the router in the first place.

Full adjudication: `S-BL.DISCOVERY-WIRE-rulings.md` v1.10, new "## Ruling 4 — Task 3
daemon-lifecycle wiring" section (also names this same gap as `S-BL.DISCOVERY-WIRE`'s new
Forward Obligation (e)).

## Non-Goals (per the fanout-options document's Option 1 scoping)

- **Key rotation UX** — out of scope; this story wires the existing static-admitted-key
  handshake, not a rotation flow.
- **Mid-connection re-admission** — out of scope; see Open Design Obligation 3 above.
- **Revocation-at-handshake handling** — out of scope beyond whatever `AdmitNode`
  already does for a revoked key (fail-closed, unmodified).

## Provenance

- **Origin:** `S-BL.DISCOVERY-WIRE.md` Forward Obligations table, row (a) — Ruling 3(f)
  verified that hop-2 fan-out target resolution (binding `NodeAddr` to a live
  connection's `InterfaceID`) does not exist in production code today.
- **Disposition:** story-ready human gate for `S-BL.DISCOVERY-WIRE`, 2026-07-14 —
  the human rejected both originally-offered resolution paths and asked for better
  options; `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.0 (six options) was produced in
  response, and the human selected its Option 1 (`S-BL.DISCOVERY-WIRE-fanout-options.md`
  v1.1 Disposition section). `S-BL.DISCOVERY-WIRE-rulings.md` v1.9, Ruling 3(f)
  subsection item (j), is the authoritative disposition record.
- **Unblocks:** `S-BL.DISCOVERY-WIRE`'s AC-017, AC-018, and Task 6 gate on this story by
  name.
- **Status:** stays `draft` — no architect ruling exists yet on this handshake's wire
  mechanics; architect elaboration is required before ACs/tasks/files can be decomposed
  (see Open Design Obligations).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-07-15 | Added Open Design Obligation 5 — a BLOCKER, not a scoping question like obligations 1-4: `S-BL.DISCOVERY-WIRE-rulings.md` v1.10 Ruling 4 (dispatched by team-lead as a Green-step implementation-time finding, verified independently) found that `admission.AdmitNode` is verification-only against the local `AdmittedKeySet`, and the router-mode process's own keyset is always empty in production — admission writes happen exclusively in the separate, disconnected control-mode OS process, with no cross-process sync mechanism anywhere in the codebase. This story's handshake cannot succeed until admission state reaches the router process, regardless of how correctly the `NODE_IDENTIFY` opcode/codec/`BindInterface` are implemented. Prerequisite named: a new follow-on story, `S-BL.ADMISSION-SYNC-WIRE` (working name, not yet created); `depends_on` stays `[]` until that story exists to be added. Obligation 5 gates obligations 1-4 (upstream of all of them). No ACs/tasks exist yet to amend; this is a scoping-stage addition. Frontmatter version 1.0 → 1.1. |
| 1.0 | 2026-07-14 | Backlog stub created per `S-BL.DISCOVERY-WIRE`'s Ruling 3(f) Forward Obligation and its story-ready human gate disposition (`S-BL.DISCOVERY-WIRE-rulings.md` v1.9 item (j); `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1 Option 1 selected). Delivers the `control_type=0x04` `NODE_IDENTIFY` handshake wiring the existing `admission.AdmitNode`/`admission.GenerateChallenge` primitives over the live connection and a new `Router.BindInterface`-shaped method recording `(SVTNID, NodeAddr) → IfaceID`. Unblocks `S-BL.DISCOVERY-WIRE`'s AC-017/AC-018/Task 6. No architect ruling adjudicates the opcode registry amendment, challenge-transcript wire format, or re-identify/rebind semantics yet; full decomposition deferred to scheduling time. |
