---
artifact_id: S-BL.NODE-IDENTIFY-WIRE
document_type: story
level: ops
story_id: S-BL.NODE-IDENTIFY-WIRE
version: "1.4"
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
points: TBD
inputs:
  - '.factory/decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
input-hash: "e4ebb26"
traces_to: "decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md"
epic_id: E-7
behavioral_contracts:
  - BC-2.01.008
bc_traces:
  - BC-2.01.008
verification_properties: []
subsystems: [session-networking]
target_module: "cmd/switchboard"
tdd_mode: strict
cycle: v1.0.0-greenfield
estimated_days: null
assumption_validations: []
risk_mitigations: []
depends_on: [S-BL.ADMISSION-SYNC-WIRE, S-BL.NODE-ADMISSION-PROVISIONING]
blocks: []
acceptance_criteria_count: 0
rulings_doc: "decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md"
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

### 1. `BC-2.01.008` opcode registry amendment — RESOLVED

`NODE_IDENTIFY = 0x04` has been added to the `BC-2.01.008` v1.3 registry table
(PO/architect amendment executed, mirroring Ruling 3(g)'s `DISCOVERY_RELAY`
precedent). The append-only, sequential-assignment rule (Invariant 3) is satisfied.

### 2. Challenge-transcript wire format — RESOLVED (see rulings doc v1.0)

Elaborated in `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.0. The ruling
specifies: `control_type=0x04` frame with a `msg_kind` sub-byte discriminator
(NodeIdentify / Challenge / ChallengeResponse variants); byte layouts for each
frame type; and `BindInterface`, `LookupInterface`, and `UnbindInterface` as the
three `*Router` methods that record and query the `(SVTNID, NodeAddr) → IfaceID`
binding. See the rulings doc for the full byte-level specification. Obligations
3–6 remain open/gated as described below.

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

### 6. BLOCKER — no production path provisions a node's own admission keypair (`S-BL.DISCOVERY-WIRE-rulings.md` v1.11 Ruling 5, 2026-07-15)

A second, distinct blocker alongside obligation 5 — solving obligation 5 alone does not
unblock this story. This handshake's `ChallengeResponse` step requires the connecting
node to sign a nonce with its own admission Ed25519 **private key** (`Sign(node_priv,
nonce)`). No production code path supplies a running node process (access-mode today)
with that private key, or even its public half: `internal/config.Config` has no
admission-keypair field; `runAccess` generates only an ephemeral, restart-unstable
keypair for its own mgmt identity, unrelated to admission. Independently confirmed
during the same pass: `internal/discovery.New`/`Discovery.Run` — the sender
daemon-lifecycle loop — have zero production callers anywhere in the repository, a
second, compounding absence.

**Prerequisite:** a second new follow-on story, **`S-BL.NODE-ADMISSION-PROVISIONING`**
(working name — not yet created, not multi-option-vetted; PO/architect to confirm name
+ scope), must land before this story's `ChallengeResponse` step can be implemented
against a real node identity. Distinct from obligation 5's `S-BL.ADMISSION-SYNC-WIRE`
— opposite direction (this story's own signing key vs. the router's verification-side
keyset). Add `S-BL.NODE-ADMISSION-PROVISIONING` to this story's `depends_on` alongside
`S-BL.ADMISSION-SYNC-WIRE` once both exist — not yet added here (frontmatter
`depends_on: []` stays empty this edit). Obligations 5 and 6 are BOTH upstream of — and
gate — obligations 1-4.

Full adjudication: `S-BL.DISCOVERY-WIRE-rulings.md` v1.11, new "## Ruling 5 — Sender-side
key-derivation fix (F-DWIP1-001) sanctioned as Ruling-1-faithful; node-side
admission-identity provisioning named as a THIRD, distinct leg of the
identity-distribution cluster" section.

## Non-Goals (per the fanout-options document's Option 1 scoping)

- **Key rotation UX** — out of scope; this story wires the existing static-admitted-key
  handshake, not a rotation flow.
- **Mid-connection re-admission** — out of scope; see Open Design Obligation 3 above.
- **Revocation-at-handshake handling** — out of scope beyond whatever `AdmitNode`
  already does for a revoked key (fail-closed, unmodified).

## Narrative

- **As a** router-mode daemon serving an SVTN
- **I want to** verify the identity of a connecting node via a `NODE_IDENTIFY` challenge-response handshake
- **So that** hop-2 fan-out target resolution (`S-BL.DISCOVERY-WIRE` AC-017/AC-018/Task 6) can be unblocked and the router can bind `(SVTNID, NodeAddr) → IfaceID` for admitted nodes

> **STATUS: DRAFT BACKLOG STUB.** Full Narrative, ACs, and Tasks will be populated when this story is scheduled. Obligations 3/4 remain open; Obligations 5/6 are blockers gated on `S-BL.ADMISSION-SYNC-WIRE` and `S-BL.NODE-ADMISSION-PROVISIONING`.

## Acceptance Criteria

> **[TODO: populate at scheduling time — blocked on Obligations 3–6]**

No ACs yet. This stub has `acceptance_criteria_count: 0`. Full decomposition is deferred to scheduling time when all Open Design Obligations (§3–6) are resolved.

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| NODE_IDENTIFY handshake | cmd/switchboard | effectful-shell |
| Router.BindInterface | internal/routing or cmd/switchboard | [TODO at scheduling time] |
| admission.AdmitNode | internal/admission | pure-core |

## Edge Cases

| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| EC-001 | [TODO: populate at scheduling time] | [TODO] |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| internal/admission | pure-core | No I/O — existing classification from S-2.02 |
| cmd/switchboard (handshake) | effectful-shell | TCP I/O, challenge-response over live connection |

## Token Budget Estimate (MANDATORY)

> **[TODO: populate at scheduling time]**

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | [TBD] |
| Referenced code files | [TBD] |
| Test files | [TBD] |
| Tool outputs overhead | [TBD] |
| **Total** | **[TBD]** |

## Tasks (MANDATORY)

> **[TODO: populate at scheduling time — blocked on Obligations 3–6]**

1. [ ] Resolve Open Design Obligations 3 and 4 (re-identify/rebind semantics; handshake timeout) via architect ruling
2. [ ] Write failing tests (test-writer)
3. [ ] Implement to pass tests (implementer)
4. [ ] Verify purity boundaries
5. [ ] Update STATE.md

## Previous Story Intelligence (MANDATORY)

| Story | Key Decisions | Patterns Established | Gotchas Discovered |
|-------|--------------|---------------------|-------------------|
| `S-BL.DISCOVERY-WIRE` (PR #123 @ d249f88) | `control_type=0x03` DISCOVERY_RELAY precedent; register-before-serve invariant (F-P2L1-001) | `wireXHandlers` pattern; zero-HMACTag connection-trust boundary | AC-017/AC-018/Task 6 gated on this story |
| `S-BL.ADMISSION-SYNC-WIRE` (prerequisite) | Router keyset population via push RPC | [populated at scheduling time] | [populated at scheduling time] |
| `S-BL.NODE-ADMISSION-PROVISIONING` (prerequisite) | Node admission keypair + Discovery.Run lifecycle | [populated at scheduling time] | [populated at scheduling time] |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|-------------|
| ARCH-08 §Import DAG — cmd/switchboard at position 18 | ARCH-08-dependency-graph.md | All new code in cmd/switchboard; no new internal/ package |
| ADR-012 challenge-response handshake | ADR-012 | NODE_IDENTIFY uses same Ed25519 challenge-response protocol as mgmt plane auth |
| Invariant 3 append-only sequential control_type allocation | BC-2.01.008 Inv-3 | NODE_IDENTIFY=0x04 already registered (Obligation 1 RESOLVED) |

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go stdlib crypto/ed25519 | stdlib | Challenge-response signing/verification |
| internal/admission | current | GenerateChallenge, AdmitNode, ChallengeResponse — reused verbatim |
| internal/mgmt | current | Existing mgmt wire protocol for control_type dispatch |

## File Structure Requirements (MANDATORY)

> **[TODO: finalize at scheduling time]**

| File | Action | Purpose |
|------|--------|---------|
| cmd/switchboard/mgmt_wire.go | modify | Add `case 0x04` (NODE_IDENTIFY) to route() switch |
| cmd/switchboard/node_identify_wire.go | create | NODE_IDENTIFY frame codec + handshake handler + BindInterface wiring |
| cmd/switchboard/node_identify_wire_test.go | create | Handshake unit + integration tests |
| specs/behavioral-contracts/ss-01/BC-2.01.008.md | modify | Add NODE_IDENTIFY=0x04 row to PC-2 registry table (Obligation 1 RESOLVED) |

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
- **Wire-format ruling:** `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.0 — resolves
  Open Design Obligation 2 (challenge-transcript wire format: control_type=0x04 with
  msg_kind sub-byte; NodeIdentify/Challenge/ChallengeResponse frame layouts;
  BindInterface/LookupInterface/UnbindInterface on `*Router`).
- **Cluster design:** `decisions/identity-cluster-architecture.md` v1.1 — the three-leg
  identity-distribution cluster design (`S-BL.NODE-IDENTIFY-WIRE`,
  `S-BL.ADMISSION-SYNC-WIRE`, `S-BL.NODE-ADMISSION-PROVISIONING`); Option E disposition
  for node-admission provisioning and Option A near-term stepping stone for admission
  sync; HLR/VLR forward architecture (Section 8).
- **Unblocks:** `S-BL.DISCOVERY-WIRE`'s AC-017, AC-018, and Task 6 gate on this story by
  name.
- **Status:** stays `draft` — Obligation 2 wire format is now specified (rulings doc v1.0);
  Obligations 3/4 remain open (re-identify semantics, handshake timeout); Obligations 5/6
  remain blockers (admission-sync and node-keypair provisioning prerequisite stories not
  yet elaborated). Full decomposition deferred to scheduling time.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.4 | 2026-07-15 | `depends_on` updated from `[]` to `[S-BL.ADMISSION-SYNC-WIRE, S-BL.NODE-ADMISSION-PROVISIONING]` — both prerequisite stories now exist and have been assigned IDs. Obligations 5 and 6 (the two blockers) are therefore reflected in the dependency graph. Obligations 3/4 remain open/gated; full decomposition still deferred to scheduling time. Frontmatter version 1.3 → 1.4. |
| 1.3 | 2026-07-15 | Added `rulings_doc: decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` frontmatter field (v1.0 on disk). Marked Obligation 1 RESOLVED — `NODE_IDENTIFY=0x04` is now in `BC-2.01.008` v1.3 registry (PO/architect amendment executed). Marked Obligation 2 RESOLVED — challenge-transcript wire format elaborated in `decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md` v1.0 (control_type=0x04 with msg_kind sub-byte; NodeIdentify/Challenge/ChallengeResponse frame layouts; BindInterface/LookupInterface/UnbindInterface on `*Router`). Added cross-reference to `decisions/identity-cluster-architecture.md` v1.1 in Provenance section. Obligations 3/4/5/6 unchanged (3/4 still open/gated; 5/6 still blockers). `depends_on` stays `[]` — prerequisite stories `S-BL.ADMISSION-SYNC-WIRE` and `S-BL.NODE-ADMISSION-PROVISIONING` not yet elaborated. |
| 1.2 | 2026-07-15 | Added Open Design Obligation 6 — a SECOND BLOCKER, distinct from obligation 5: `S-BL.DISCOVERY-WIRE-rulings.md` v1.11 Ruling 5 (dispatched by team-lead as a Step-4.5 pass-1 fix-burst finding, verified independently) found no production code path supplies a running node process with its own admission keypair — needed for this story's `ChallengeResponse` signing step — and independently found `internal/discovery.New`/`Discovery.Run` have zero production callers anywhere, a compounding absence. Prerequisite named: a second new follow-on story, `S-BL.NODE-ADMISSION-PROVISIONING` (working name, not yet created), distinct in direction from obligation 5's `S-BL.ADMISSION-SYNC-WIRE`. `depends_on` stays `[]` until both prerequisite stories exist. Obligations 5 and 6 both gate obligations 1-4. Frontmatter version 1.1 → 1.2. |
| 1.1 | 2026-07-15 | Added Open Design Obligation 5 — a BLOCKER, not a scoping question like obligations 1-4: `S-BL.DISCOVERY-WIRE-rulings.md` v1.10 Ruling 4 (dispatched by team-lead as a Green-step implementation-time finding, verified independently) found that `admission.AdmitNode` is verification-only against the local `AdmittedKeySet`, and the router-mode process's own keyset is always empty in production — admission writes happen exclusively in the separate, disconnected control-mode OS process, with no cross-process sync mechanism anywhere in the codebase. This story's handshake cannot succeed until admission state reaches the router process, regardless of how correctly the `NODE_IDENTIFY` opcode/codec/`BindInterface` are implemented. Prerequisite named: a new follow-on story, `S-BL.ADMISSION-SYNC-WIRE` (working name, not yet created); `depends_on` stays `[]` until that story exists to be added. Obligation 5 gates obligations 1-4 (upstream of all of them). No ACs/tasks exist yet to amend; this is a scoping-stage addition. Frontmatter version 1.0 → 1.1. |
| 1.0 | 2026-07-14 | Backlog stub created per `S-BL.DISCOVERY-WIRE`'s Ruling 3(f) Forward Obligation and its story-ready human gate disposition (`S-BL.DISCOVERY-WIRE-rulings.md` v1.9 item (j); `S-BL.DISCOVERY-WIRE-fanout-options.md` v1.1 Option 1 selected). Delivers the `control_type=0x04` `NODE_IDENTIFY` handshake wiring the existing `admission.AdmitNode`/`admission.GenerateChallenge` primitives over the live connection and a new `Router.BindInterface`-shaped method recording `(SVTNID, NodeAddr) → IfaceID`. Unblocks `S-BL.DISCOVERY-WIRE`'s AC-017/AC-018/Task 6. No architect ruling adjudicates the opcode registry amendment, challenge-transcript wire format, or re-identify/rebind semantics yet; full decomposition deferred to scheduling time. |
