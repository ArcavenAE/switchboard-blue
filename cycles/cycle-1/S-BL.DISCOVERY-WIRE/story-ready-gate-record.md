---
artifact_id: STORY-READY-GATE-S-BL.DISCOVERY-WIRE
document_type: gate-record
level: ops
version: "1.0"
status: complete
producer: state-manager
timestamp: 2026-07-14T00:00:00Z
cycle: cycle-1
inputs:
  - decisions/S-BL.DISCOVERY-WIRE-rulings.md
  - decisions/S-BL.DISCOVERY-WIRE-fanout-options.md
  - specs/behavioral-contracts/ss-03/BC-2.03.002.md
  - stories/S-BL.DISCOVERY-WIRE.md
traces_to: stories/S-BL.DISCOVERY-WIRE.md
---

# Story-Ready Human Gate Record — S-BL.DISCOVERY-WIRE

## (a) Gate context

Spec-adversarial convergence for `S-BL.DISCOVERY-WIRE` reached COMPLETE at
pass 16, per `cycles/cycle-1/S-BL.DISCOVERY-WIRE/adversary-convergence-state.json`:
pass 14 was NITPICK_ONLY, passes 15 and 16 were CLEAN — three consecutive
clean/nitpick-only passes satisfying the convergence bar. 16 total passes
run, 11 fix-bursts applied, 8 findings raised across the arc. Convergence
left the story (`stories/S-BL.DISCOVERY-WIRE.md`, then v2.11) in `status:
draft` with three items deliberately withheld from `status: ready`,
pending human sign-off at this gate. This record documents the 2026-07-14
disposition of those items, executed as a single story-ready gate.

## (b) Dispositions

| # | Item | Disposition |
|---|------|-------------|
| 1 | **SEC-DW-07** — Sequence design and both residual replay-window bounds (Case 1: ≤1s same-wall-clock-second crash-loop; Case 2: ≈N backward host-clock adjustment) | **APPROVED as documented.** No further hardening requested; both residual bounds accepted as-is. |
| 2 | **Discovery UDP port** | **`49201` ADOPTED.** Bikeshed closed — no longer a placeholder; rulings v1.9 and story v2.12 both record it as adjudicated. |
| 3 | **Fan-out target resolution** (which live connections belong to a given admitted node, for hop-2 relay dispatch) | Both originally-offered options **REJECTED by human** (neither the unnamed sequencing dependency nor the narrow story-local seam with no identity signal was acceptable). The architect produced `S-BL.DISCOVERY-WIRE-fanout-options.md` (six additional options); the human selected **Option 1**: a new, immediately-named, immediately-scheduled companion story, **`S-BL.NODE-IDENTIFY-WIRE`** (`control_type=0x04` `NODE_IDENTIFY` handshake, binding `(SVTNID, NodeAddr) → IfaceID`). `S-BL.DISCOVERY-WIRE`'s AC-017, AC-018, and Task 6 now gate on it by name. |
| 4 | **`sessions.list` PC-1 exposure** | Resolved to a follow-on story, **`S-BL.SESSIONS-LIST-WIRE`** (console-facing RPC handler over the mgmt wire exposing `discovery.Enumerate()`). `BC-2.03.002.md` bumped to v1.5, re-pointing PC-1's annotation to it. |
| 5 | **PG-DWSP6-01 sweep-blind-spot lineage** | Resolved to a follow-up story, **`S-M.03-sweep-methodology-hardening`** — hardens the spec-artifact citation-pin sweep methodology (canonical pattern library + semantic-claim verification) that PG-DWSP6-01 and the pass-13/pass-14 rulings-version-attribution findings exposed as a recurring class. |

## (c) Resulting artifacts

- `decisions/S-BL.DISCOVERY-WIRE-rulings.md` — v1.9
- `decisions/S-BL.DISCOVERY-WIRE-fanout-options.md` — v1.1 (decided; new file)
- `specs/behavioral-contracts/ss-03/BC-2.03.002.md` — v1.5
- `stories/S-BL.DISCOVERY-WIRE.md` — v2.12, **status `draft` → `ready`**, input-hash `a39b7ad` → `f5135e6`
- `stories/STORY-INDEX.md` — v4.110
- `stories/S-BL.NODE-IDENTIFY-WIRE.md` — v1.0 (new stub, draft, wave backlog)
- `stories/S-BL.SESSIONS-LIST-WIRE.md` — v1.0 (new stub, draft, wave backlog)
- `stories/S-M.03-sweep-methodology-hardening.md` — v1.0 (new stub, draft, wave backlog)

## (d) Next

Phase-3 per-story delivery of Tasks 1-5 (AC-001..AC-016) proceeds per
standing approval — the story-ready gate clears these for delivery now.
Task 6 and AC-017/AC-018 remain deferred, gated on `S-BL.NODE-IDENTIFY-WIRE`
by name, until that companion story is itself delivered.
