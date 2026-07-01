---
artifact_id: S-BL.ROUTER-ADDR
document_type: story
level: ops
story_id: S-BL.ROUTER-ADDR
title: "populate PathSnapshot.RouterAddr with real resolved host:port (BC-2.06.003 PC-1)"
status: ready-for-red-gate
producer: product-owner
timestamp: 2026-07-01T12:00:00
phase: 2
epic: E-6
wave: backlog
priority: P1
scope_phase: E
estimated_points: 2
bc_traces:
  - BC-2.06.003
vp_traces: [VP-047, VP-062]
subsystems: [quality-observability, multipath-forwarding]
architecture_modules: [internal/metrics, internal/paths, cmd/sbctl]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-W5.04]
blocks: [S-BL.PATH-TRACKER-WIRING]
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/decisions/wave-6-tranche-a-scope-rulings.md'
  - '.factory/decisions/RULING-W6TB-B-router-addr-seam.md'
  - '.factory/decisions/RULING-W6TB-F-s-bl-router-addr-vp047.md'
acceptance_criteria_count: 5
revision: "1.1-ready-for-red-gate"
changed_by_rulings:
  - RULING-W6TB-B
  - RULING-W6TB-F
backlog_origin:
  source: wave-6-tranche-a-scope-rulings Ruling-1 + DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER
  ruling: Ruling-1
  drift_items_consumed:
    - DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER
  notes: >
    BC-2.06.003 v1.9 (Ruling-1) permits `router_addr: ""` as an interim sentinel while
    PathSnapshot is not enriched with a real resolved host:port. Consumers MUST treat ""
    as a valid sentinel meaning "address not yet resolved." When this story ships,
    router_addr will always be a non-empty `host:port` string for PathTrackers constructed
    with NewPathTrackerWithAddr. RULING-W6TB-B (2026-07-01) establishes this as a
    unit-scope story; end-to-end observability deferred to S-BL.PATH-TRACKER-WIRING.
---

# S-BL.ROUTER-ADDR: Populate PathSnapshot.RouterAddr with Real Resolved host:port

> **Execute:** `/vsdd-factory:deliver-story S-BL.ROUTER-ADDR`

## Scope Note (RULING-W6TB-B)

**Unit-scope only (Option A).** This story adds `RouterAddr string` to
`PathSnapshot`, populates it at `PathTracker` construction time via a new
`NewPathTrackerWithAddr(addr string, ...)` constructor, and enriches the
`internal/metrics.PathsList` handler to pass the stored addr through
`PathEntryFromSnapshot`. **End-to-end observability** (non-empty `router_addr`
from `sbctl paths list` against a running daemon) is NOT achievable in this story
and is deferred to **S-BL.PATH-TRACKER-WIRING**. See
`.factory/decisions/RULING-W6TB-B-router-addr-seam.md`.

`NewPathTracker` (the existing constructor) is preserved unchanged for callers that
do not yet have an addr. `blocks: [S-BL.PATH-TRACKER-WIRING]` is correct: this
story must land before PATH-TRACKER-WIRING (which wires production `PathTracker`
instances to the routing registry and supplies real `host:port` values).

## Narrative

- **As an** operator running `sbctl paths list`
- **I want to** see a non-empty `router_addr` field showing the resolved `host:port` for each path
- **So that** BC-2.06.003 PC-1 `router_addr` semantics are fully satisfied and operators can
  identify which router address each path is associated with

## Behavioral Contracts

| BC | Title | PCs covered | Version |
|----|-------|------------|---------|
| BC-2.06.003 | Path Enumeration + Metrics (daemon-side) | PC-1 (`router_addr` field) | v1.15 (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER annotation removed; Ruling-1 interim wording retracted) |

## Acceptance Criteria

### AC-001 (traces to BC-2.06.003 PC-1 — RouterAddr field)
`PathSnapshot` gains a `RouterAddr string` field. `PathTracker.Snapshot()` copies
the tracker's stored addr verbatim into `PathSnapshot.RouterAddr`. For a
`PathTracker` constructed via `NewPathTrackerWithAddr("127.0.0.1:9000", ...)`,
`Snapshot().RouterAddr == "127.0.0.1:9000"`. For a `PathTracker` constructed via
`NewPathTracker(...)` (no addr), `Snapshot().RouterAddr == ""`.
- **Test:** `TestPathTracker_Snapshot_RouterAddr` — assert non-empty addr propagated;
  `TestPathTracker_Snapshot_RouterAddr_Empty` — assert `NewPathTracker` still yields `""`

### AC-002 (traces to BC-2.06.003 PC-1 — PathsList handler)
`PathsList` handler passes `snap.RouterAddr` (not the literal empty string `""`) to
`PathEntryFromSnapshot` when the snapshot carries a non-empty addr. The existing
hard-coded `""` at `internal/metrics/handlers.go` lines 65–67 must be replaced with
`snap.RouterAddr`.
- **Test:** `TestPathsList_PassesRouterAddr` — construct a `PathsListSource` stub
  returning a snapshot with `RouterAddr="127.0.0.1:9000"`; call `PathsList`; verify
  JSON output contains `"router_addr":"127.0.0.1:9000"`

### AC-003 (traces to BC-2.06.003 PC-1 — constructor variant)
New constructor `NewPathTrackerWithAddr(addr string, initialRTTMS float64, alpha float64) *PathTracker`
exists and stores `addr` immutably. The existing `NewPathTracker(initialRTTMS float64, alpha float64) *PathTracker`
is preserved unchanged — all existing call sites continue to compile and behave identically.
- **Test:** `TestNewPathTrackerWithAddr_StoresAddr` — verify stored addr equals constructor arg;
  `TestNewPathTracker_Unchanged` — verify existing `NewPathTracker` test suite still passes unmodified

### AC-004 (traces to BC-2.06.003 PC-1 — spec annotation cleanup)
BC-2.06.003 PC-1 DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER annotation is updated:
- Remove the `""` sentinel-permission clause (no longer permitted from a `PathTracker`
  with a set addr)
- Remove the "PathSnapshot enrichment tracked in follow-on story" note
- Replace with a permanent note: `router_addr` is populated from `PathSnapshot.RouterAddr`
  (set at construction via `NewPathTrackerWithAddr`)
- Bump BC-2.06.003 to v1.15; add changelog entry citing RULING-W6TB-B
- DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER is closed by this story
- **Test:** `TestBC2_06_003_PC1_DrftAnnotationRemoved` (spec validator — not a Go test;
  this is a story-level obligation, not a runtime test)

### AC-005 (traces to BC-2.06.003 PC-1 — drift closure)
DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER is closed. The story's integration test oracle
for VP-047 AC-006 is updated: replace the `router_addr == ""` assertion (previously
correct per Ruling-1 interim sentinel) with `router_addr == <stub-addr>` for any
`PathTracker` constructed with `NewPathTrackerWithAddr`.
- **Test:** `TestVP047_RouterAddrNonEmpty` — integration test verifying VP-047 AC-006
  router_addr field assertion with non-empty stub addr; existing `router_addr == ""`
  oracle in S-W5.04 test suite must be flipped for `NewPathTrackerWithAddr`-constructed
  paths (paths from `NewPathTracker` may still return `""`)

## Deferred: End-to-End Observability

Per RULING-W6TB-B, the following is NOT in scope for this story:

| Deferred observable | Required by |
|--------------------|-------------|
| `sbctl paths list` returns non-empty `router_addr` for live daemon paths | S-BL.PATH-TRACKER-WIRING |
| Production `PathTracker` instances constructed with `NewPathTrackerWithAddr` | S-BL.PATH-TRACKER-WIRING |

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| PathSnapshot.RouterAddr field | internal/paths | pure-core |
| NewPathTrackerWithAddr constructor | internal/paths | pure-core |
| PathsList handler — snap.RouterAddr pass-through | internal/metrics | boundary |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| internal/paths | pure-core | Struct field and constructor; no I/O; no new package deps |
| internal/metrics | boundary | Reads snapshot addr; passes to JSON encoder |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~1,200 |
| BC-2.06.003.md (v1.15 target) | ~700 |
| RULING-W6TB-B.md | ~700 |
| internal/paths/paths.go (current) | ~600 |
| internal/paths/paths_test.go (current) | ~400 |
| internal/metrics/handlers.go (lines 60–80) | ~300 |
| internal/metrics/handlers_test.go | ~400 |
| VP-047 (AC-006 oracle flip) | ~400 |
| Tool outputs overhead | ~200 |
| **Total** | **~4,900** |
| Agent context window | 200K |
| **Budget usage** | **~2.5%** |

## Tasks (MANDATORY: red-first TDD)

1. [ ] Read RULING-W6TB-B (`.factory/decisions/RULING-W6TB-B-router-addr-seam.md`) — understand unit-scope rationale, constructor design, why `NewPathTracker` is preserved
2. [ ] Read `internal/paths/paths.go` — understand `PathSnapshot` struct and `PathTracker.Snapshot()` method
3. [ ] Read `internal/paths/paths_test.go` — match existing test style
4. [ ] Read `internal/metrics/handlers.go` lines 65–67 — see hard-coded `""` interim placeholder comment
5. [ ] Read `internal/metrics/handlers_test.go` — understand existing `PathsList` test structure
6. [ ] **RED:** Write failing tests for AC-001 (`TestPathTracker_Snapshot_RouterAddr`), AC-002 (`TestPathsList_PassesRouterAddr`), AC-003 (`TestNewPathTrackerWithAddr_StoresAddr`), AC-005 (`TestVP047_RouterAddrNonEmpty` oracle flip)
7. [ ] Verify Red Gate (all new tests fail; existing tests still pass; density ≥ 0.5)
8. [ ] **GREEN:** Add `RouterAddr string` field to `PathSnapshot` in `internal/paths/paths.go`
9. [ ] **GREEN:** Add `NewPathTrackerWithAddr(addr string, initialRTTMS float64, alpha float64) *PathTracker` constructor; store addr on tracker; `Snapshot()` copies it; `NewPathTracker` unchanged
10. [ ] **GREEN:** Replace hard-coded `""` at `internal/metrics/handlers.go:65–67` with `snap.RouterAddr`; remove DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER comment
11. [ ] **GREEN:** Bump BC-2.06.003 to v1.15; remove DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER annotation from PC-1; remove `""` sentinel-permission clause; add changelog entry (AC-004)
12. [ ] Flip VP-047 AC-006 oracle: `router_addr == ""` → `router_addr == <stub-addr>` for `NewPathTrackerWithAddr`-constructed paths (AC-005)
13. [ ] just fmt && just lint pass
14. [ ] just test-race — all packages clean

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| internal/paths/paths.go | MODIFY | Add `RouterAddr string` to `PathSnapshot`; add `NewPathTrackerWithAddr` constructor |
| internal/paths/paths_test.go | MODIFY | Add AC-001 + AC-003 tests |
| internal/metrics/handlers.go | MODIFY | Lines 65–67: replace `""` with `snap.RouterAddr`; remove DRIFT comment |
| internal/metrics/handlers_test.go | MODIFY | Add AC-002 `TestPathsList_PassesRouterAddr` |
| internal/metrics/integration_test.go | MODIFY | Flip VP-047 AC-006 oracle (AC-005) |
| .factory/specs/verification-properties/VP-047.md | MODIFY | v1.3→v1.4: retract Ruling-1 interim clauses; property statement says router_addr MUST equal PathSnapshot.RouterAddr; "" valid only for addr-less NewPathTracker; DRIFT closed (Ruling RULING-W6TB-F §Ruling 1) |
| .factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md | MODIFY | Bump v1.14→v1.15; remove DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER annotation from PC-1 (AC-004) |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|-------------|
| `RouterAddr` immutable after construction — set once, never mutated | RULING-W6TB-B §3 "When RouterAddr is set" | No setter method; field only written in constructor |
| `NewPathTracker` (addr-less) continues to produce `PathSnapshot.RouterAddr == ""` | RULING-W6TB-B | `TestNewPathTracker_Unchanged` verifies backward compat |
| No new package dependencies in `internal/paths` | RULING-W6TB-B §2 pure-core boundary | `go build ./internal/paths/...` must not add imports |
| End-to-end observability NOT claimed — `router_addr` only non-empty in unit tests with stub source | RULING-W6TB-B §3 | No integration test asserts non-empty from live daemon paths |
| DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER closed by this story | RULING-W6TB-B + wave-6-tranche-a-scope-rulings Ruling-1 | BC-2.06.003 v1.15 annotation removal (AC-004) |

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.1-ready-for-red-gate | 2026-07-01 | product-owner | RULING-W6TB-F: add VP-047.md MODIFY row to File Structure (spec target for AC-005 oracle flip was missing — F-L3-001). Strengthen TestVP047_FieldSwapOracle seed: routerAddr "abcdefghi" → "127.0.0.1:9000" (valid host:port; non-overlapping oracle preserved — F-LENS2-01). Bump revision. |
| 1.0-ready-for-red-gate | 2026-07-01 | story-writer | RULING-W6TB-B AC set: promote backlog stub to ready-for-red-gate. Add 5 concrete ACs (RouterAddr field, PathsList pass-through, NewPathTrackerWithAddr constructor, BC-2.06.003 annotation cleanup, DRIFT closure + oracle flip). Points TBD→2. File Structure rows: internal/paths/paths.go MODIFY, internal/metrics/handlers.go MODIFY, internal/paths/paths_test.go MODIFY, internal/metrics/handlers_test.go + integration_test.go MODIFY, BC-2.06.003.md MODIFY. Red-first task list. Scope boundary: unit-scope only; end-to-end deferred to S-BL.PATH-TRACKER-WIRING (Wave-7). Add changed_by_rulings: RULING-W6TB-B. |
| 0.1-backlog-stub | 2026-07-01 | product-owner | Initial backlog stub per wave-6-tranche-a-scope-rulings Ruling-1 + DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER. |
