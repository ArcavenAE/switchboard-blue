---
artifact_id: S-W5.04-daemon-paths-metrics-handlers
document_type: story
level: ops
story_id: S-W5.04
title: "daemon-side paths.list / router.metrics / router.status RPC handlers and response types"
status: draft
producer: story-writer
timestamp: 2026-07-01T00:00:00
phase: 2
epic: E-5
wave: 6
# Re-scheduled Wave 5 → Wave 6 per F-W5P1-004 (Wave-5 wave-adversarial Pass-1) — Wave 5 declared complete at 8 stories; depends on S-5.02 + S-W5.01 (both merged, unblocked).
priority: P1
scope_phase: E
estimated_points: 5
version: "1.8"
bc_traces:
  - BC-2.06.001
  - BC-2.06.003
vp_traces: [VP-047, VP-062]
# VP-062 transferred from S-5.02 per Pass-6 Ruling (F-P6L3-003): fuzz harness requires
# daemon-side types (metrics.PathEntry, metrics.PathsListResponse, metrics.RTTValue,
# metrics.RouterMetricsResponse) which are minted in this story. Mirrors VP-047 Pass-4 Ruling-3 precedent.
subsystems: [quality-observability, network-management]
architecture_modules: [internal/metrics, internal/mgmt]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-5.02, S-W5.01]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
  - '.factory/specs/architecture/ARCH-12-daemon-management-plane.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
acceptance_criteria_count: 6
---

# S-W5.04: Daemon-Side paths.list / router.metrics / router.status RPC Handlers

> **Execute:** `/vsdd-factory:deliver-story S-W5.04`

## Origin

Minted 2026-06-30 per S-5.02 Pass-4 Ruling 1
(`decisions/S-5.02-pass4-scope-ruling.md`). S-5.02 delivered the sbctl
client-side surface (`cmd/sbctl`) and the `internal/paths` histogram.
This story delivers the daemon-side half: RPC handler registration in
`internal/mgmt` and the response types in `internal/metrics` that allow
a real daemon to respond to `sbctl paths list`, `sbctl router metrics --svtn=<id>`,
and `sbctl router status --target <router>`.

## Narrative

- **As an** operator
- **I want to** run `sbctl paths list --json` against a live daemon and receive
  per-path RTT, p99 RTT, loss, and quality metrics sourced from the daemon's
  real PathTracker state
- **So that** I can diagnose connection quality using live daemon data, not just
  stub responses

## Acceptance Criteria

### AC-001 (traces to BC-2.06.003 postcondition 1 — paths.list handler)
The `paths.list` RPC handler is registered in `internal/mgmt` dispatch. When
invoked, it reads all active `PathSnapshot` values from `internal/paths` and
returns a `PathsListResponse{paths: []PathEntry}` JSON-serialized per the
BC-2.06.003 PC-1 schema. The `PathEntry` type lives in `internal/metrics`.
- **Test:** `TestDaemonPathsList_HandlerRegistered` — call handler via mgmt
  server; assert response conforms to `PathsListResponse` schema with at least
  one entry.

### AC-002 (traces to BC-2.06.003 postcondition 1, EC-003 — rtt_p99_ms union serialization)
`PathEntry.rtt_p99_ms` serializes as a float64 when `PathSnapshot.SampleCount ≥ 10`
and as the JSON string `"pending"` when `SampleCount < 10`, per BC-2.06.003
v1.11 postcondition 1 (fixed-bucket histogram, counts never reset) and EC-003
(pending sentinel). The union serialization is handled by `RTTValue` type in
`internal/metrics` implementing `json.Marshaler`.
- **Test:** `TestPathEntry_RTTValueSerialization` — table-driven: row (a) SampleCount=0
  → `"pending"`, row (b) SampleCount=9 → `"pending"`, row (c) SampleCount=10 →
  float64, row (d) SampleCount=100 → float64.

### AC-003 (traces to BC-2.06.001 postcondition — PathEntry.status from Degraded)
`PathEntry.status` is set to `"degraded"` when `PathSnapshot.Degraded == true`
and to `"active"` otherwise, reflecting the quality state machine from BC-2.06.001.
The valid status enum in this story is `{active, degraded}` per BC-2.06.003 v1.11 PC-1;
`"failed"` is reserved for a future liveness-signal story (S-BL.PATH-FAILED-STATUS).
- **Test:** `TestPathEntry_StatusFromDegraded` — assert `status: "degraded"` when
  snapshot has `Degraded: true`, `status: "active"` when `Degraded: false`.

### AC-004 (traces to BC-2.06.003 postcondition 2 — router.metrics handler)
The `router.metrics` RPC handler is registered in `internal/mgmt` dispatch.
When invoked with `svtn=<id>`, it returns a `RouterMetricsResponse` containing
aggregate forwarding metrics (`frame_count`, `hmac_fail_count`, `drop_cache_hits`,
`path_distribution`) for the requested SVTN, JSON-serialized per BC-2.06.003 PC-2.
If the SVTN is not found, returns an error envelope (E-RPC-011).
- **Test:** `TestDaemonRouterMetrics_HandlerRegistered` — call handler; assert
  response conforms to `RouterMetricsResponse` schema.

### AC-005 (traces to BC-2.06.003 postcondition 3 — router.status handler)
The `router.status` RPC handler is registered in `internal/mgmt` dispatch and
returns a daemon-level status envelope with a per-path quality summary. The
response JSON structure (minus the `quality` field) is structurally identical
to `paths.list` response, consistent with the sbctl alias design in S-5.02.
- **Test:** `TestDaemonRouterStatus_HandlerRegistered` — call handler; assert
  response fields match the paths.list shape plus a `quality` summary field.

  **AC-005a (traces to BC-2.06.003 EC-007 — status/quality independence + pending precedence, S502-DEFER-3):**
  Status and quality are independent fields with distinct derivations. The status enum is
  `{active, degraded}` in this story; `"failed"` is reserved for a future liveness-signal story
  (S-BL.PATH-FAILED-STATUS) — do not emit it.
  - `PathSnapshot.Degraded == true` maps to `status: "degraded"` per BC-2.06.001 quality state machine.
  - When `SampleCount < 10` (p99 indeterminate → `rtt_p99_ms: "pending"`), the `quality` field MUST be `"pending"` regardless of `status`. The quality enum is `{green, yellow, red, pending}`; `"failed"` is not a valid quality value.
  - The EC-007 precedence rule: quality="pending" when SampleCount<10, regardless of status value.
  - **Test:** `TestDaemonRouterStatus_QualityStatusIndependence` — table-driven:
    row (b) Degraded=true (→ status `"degraded"`) + SampleCount=10 → quality derived from p99 (not pending);
    row (c) status=`"active"` + SampleCount=5 → quality `"pending"`.

### AC-006 (traces to BC-2.06.003 postcondition 1 — VP-047 integration: end-to-end field presence)
`sbctl paths list --json` against a real (non-stub) daemon returns paths with all
required fields present and non-null: `path_id`, `rtt_ms`,
`rtt_p99_ms` (float64 or `"pending"`), `loss_pct`, `status`. Covers at least one
path in pending state (`SampleCount < 10`) and one in green state (`SampleCount ≥ 10`).
`router_addr` MUST be present in the JSON output; its value is `""` (empty string)
in this interim state pending `PathSnapshot.RouterAddr` enrichment
(DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER). The AC-006 integration test MUST assert
`router_addr` key presence and accept `""` as a valid value. See wave-6-tranche-a-scope-rulings.md
Ruling-1 (Option A) and follow-on story S-BL.ROUTER-ADDR, which will land a real
`host:port` value before Wave-6 wave-convergence.
This AC is the implementation target for VP-047.

**Handler surface, response types, and adapter interface are in scope for S-W5.04.**
The `pathTrackerSource` adapter interface is defined and the integration test populates
it with synthetic `PathTracker` instances (test-only population). **Production
`pathTrackerSource` population from the routing subsystem registry is DEFERRED to
`S-BL.PATH-TRACKER-WIRING`.** See wave-6-tranche-a-scope-rulings.md Ruling-6 (Option B).
Any residual empty-source dead code (e.g., a fully empty stub with no register method)
should be deleted if present; otherwise skip that step.

- **Test:** `TestVP047_SbctlPathsList_EndToEnd` — integration test that populates
  `pathTrackerSource` with at least one synthetic `PathTracker` instance and asserts
  that `GET /paths` returns that entry. `router_addr` key presence asserted, `""` accepted.
  The test MUST exercise the full handler→source→response code path; direct response
  fabrication or bypassing the source interface is not permitted.
  Production population of `pathTrackerSource` from the routing subsystem is deferred
  to `S-BL.PATH-TRACKER-WIRING`. `// #DEFERRED: S-BL.PATH-TRACKER-WIRING` comment
  MUST be present on the `pathTrackerSource` field or initialization site.

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| paths.list handler | internal/mgmt | effectful (reads PathTracker state) |
| router.metrics handler | internal/mgmt | effectful (reads routing counters) |
| router.status handler | internal/mgmt | effectful (reads PathTracker + routing state) |
| PathsListResponse, PathEntry, RTTValue, RouterMetricsResponse types | internal/metrics | pure-core (data types + serialization) |
| PathSnapshot read path | internal/paths | pure-core (Snapshot() read under mutex) |

## Behavioral Contracts

| BC | Title | PCs covered |
|----|-------|------------|
| BC-2.06.001 | Quality indicator (green/yellow/red) derived from measured path latency and loss | Quality state machine: `PathSnapshot.Degraded == true` → `status: "degraded"`; quality field derived from p99 RTT when available |
| BC-2.06.003 | Per-Path RTT and Loss Metrics Queryable via sbctl | PC-1 (PathsListResponse + PathEntry + rtt_p99_ms union), PC-2 (RouterMetricsResponse), PC-3 (router.status handler + EC-007 failed+pending precedence), PC-4 (--json), PC-5 (daemon unreachable — inherited from S-5.02 client) |

## VP Coverage

| VP | Property | Proof Method | AC |
|----|----------|--------------|----|
| VP-047 | `sbctl paths list --json` returns paths with required fields present; at least one pending + one green path | integration | AC-006 |
| VP-062 | JSON output valid for all sbctl metrics CLI input combinations; pending-p99 quality sentinel propagation | fuzz | Phase-6 hardening; no Wave-5 AC anchor — daemon types minted by S-W5.04 AC-001..AC-006 |

## Scope Boundary

- sbctl client-side dispatch (`cmd/sbctl`): owned by S-5.02. Do NOT re-implement or modify client dispatch here.
- `internal/paths` histogram and `PathSnapshot`: owned by S-5.02. Do NOT change `PathTracker`, `rttHistogram`, or `PathSnapshot` internals.
- `internal/mgmt` server authentication and transport: owned by S-W5.01. This story calls `mgmt.Server.Register()` to register handlers; it does not modify the server core.
- Router-side forwarding metric counters (`frame_count`, `hmac_fail_count`, `drop_cache_hits`): confirm availability from existing `internal/routing` state before implementing AC-004; if not available, scope AC-004 to return zeroed counters with a TODO marker and file a follow-on story.
- BC-2.06.003 v1.11 text: do NOT modify. PO bumped to v1.11 per wave-6-tranche-a-scope-rulings.md Ruling-4 (status enum retracted to `{active, degraded}`; `"failed"` reserved for S-BL.PATH-FAILED-STATUS) and Pass-3 L3 F-L3-001 (PC-3 S502-DEFER-3 rewrite).

## Edge Cases

| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| EC-001 | No active paths | `paths.list` returns `{"paths":[],"message":"no active paths"}`; exit 0 |
| EC-002 | All paths pending (SampleCount < 10) | All PathEntry.rtt_p99_ms values are `"pending"` string |
| EC-003 | Mixed pending + green paths | Pending entries have `rtt_p99_ms: "pending"`; green entries have float64 |
| EC-004 | SVTN not found for router.metrics | E-RPC-011 error envelope; client receives structured error |
| EC-005 | Degraded path in paths.list response | `PathEntry.status: "degraded"` when `PathSnapshot.Degraded == true` |
| EC-006 | router.status on a path with Degraded=true AND SampleCount<10 | `status: "degraded"` AND `rtt_p99_ms: "pending"` AND `quality: "pending"`. Degraded==true → status `"degraded"`; quality="pending" takes precedence when SampleCount<10. See BC-2.06.003 v1.11 EC-007 and Ruling-4 (wave-6-tranche-a-scope-rulings.md). |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~1,800 |
| BC-2.06.003.md (v1.11) | ~1,200 |
| ARCH-03 §p99 RTT Accumulator + §PathSnapshot | ~2,500 |
| ARCH-12 daemon management plane | ~1,500 |
| interface-definitions (JSON envelope, E-RPC-011) | ~400 |
| internal/mgmt/mgmt.go (handler registration surface) | ~1,500 |
| internal/paths/paths.go (PathSnapshot type, Snapshot()) | ~2,000 |
| internal/metrics/metrics.go (existing query path) | ~1,500 |
| Test files | ~1,500 |
| Tool outputs overhead | ~400 |
| **Total** | **~14,300** |
| Agent context window | 200K |
| **Budget usage** | **~7.2%** |

## Tasks (MANDATORY)

1. [ ] Read BC-2.06.003 v1.11 (full), ARCH-03 v1.6 §p99 RTT Accumulator, ARCH-12, interface-definitions.md
2. [ ] Read `internal/mgmt/mgmt.go` — identify `mgmt.Server.Register()` signature and handler interface
3. [ ] Read `internal/paths/paths.go` — confirm `PathSnapshot` fields: `P99RTTMs float64`, `SampleCount uint64`, `Degraded bool`
4. [ ] Read `internal/metrics/metrics.go` — identify existing types and query surface
5. [ ] Write failing tests for AC-001 through AC-006 (table-driven for AC-002 RTTValue serialization; integration test for AC-006/VP-047)
5a. [ ] Add negative tests: (a) data-race regression for register-before-serve; (b) discriminating status oracle when Degraded=false + SampleCount>=10; (c) malformed-args decode error path; (d) VP-047 field-swap oracle (`path_id` ↔ `router_addr` collisions detected); (e) EC-006 row test
6. [ ] Verify Red Gate — all AC tests must fail before implementation
7. [ ] Define `RTTValue` union type in `internal/metrics` implementing `json.Marshaler` (float64 | `"pending"` per SampleCount)
7a. [ ] Fix `RTTValue.UnmarshalJSON` to preserve source discrimination using a `Kind` enum (`PendingKind`, `FloatKind`) rather than sentinel-based approach (which cannot distinguish `nil` from `float64(0)`)
8. [ ] Define `PathEntry` type in `internal/metrics` (path_id, router_addr, rtt_ms, rtt_p99_ms RTTValue, loss_pct, status)
9. [ ] Define `PathsListResponse` type in `internal/metrics` (paths []PathEntry)
10. [ ] Define `RouterMetricsResponse` type in `internal/metrics` (frame_count, hmac_fail_count, drop_cache_hits, path_distribution)
11. [ ] Implement `paths.list` handler: read all PathSnapshots via `internal/paths`, build PathEntry slice, serialize PathsListResponse
12. [ ] Implement `router.metrics` handler: read per-SVTN forwarding counters, serialize RouterMetricsResponse
13. [ ] Implement `router.status` handler: reuse paths.list logic; add quality summary field
14. [ ] Register all three handlers via `mgmt.Server.Register()` in the appropriate daemon init path
15. [ ] Integration test (VP-047): spin up daemon with live PathTracker seeded with two snapshots, run `sbctl paths list --json`, assert required fields for AC-006
15a. [ ] Define `pathTrackerSource` adapter interface in `cmd/switchboard/metrics_wire.go`; add `// #DEFERRED: production population from routing registry deferred to S-BL.PATH-TRACKER-WIRING. Test-time registration via .register() is the only population path in this wave.` comment at the field/initialization site. Adapter interface defined + test-populated; production wiring deferred. Delete any residual empty-source dead code (a stub with no register method) if present; otherwise skip.
16. [ ] `just fmt && just lint` pass

## Previous Story Intelligence (MANDATORY)

| Story | Key Decisions | Patterns Established | Gotchas Discovered |
|-------|--------------|---------------------|-------------------|
| S-5.02 | sbctl client dispatch wired; rttHistogram + p99() added to internal/paths; PathSnapshot.P99RTTMs populated; startCannedDaemon stub used for all client-side tests | JSON envelope shape; qualityFromPathEntry in cmd/sbctl/router_status.go | Daemon handlers absent at S-5.02 delivery — this is S-W5.04's entry point. All S-5.02 tests use a canned stub; VP-047 was explicitly deferred. |
| S-W5.01 | internal/mgmt server wired to all 4 daemon modes; management socket registered | mgmt.Server.Register() is the handler registration surface; handler names are method strings (e.g. "paths.list") | Authentication and transport are S-W5.01's concern; this story only calls Register() |
| S-5.01 | QualityIndicator green/yellow/red/pending; pure-core quality state machine in internal/quality | Hysteresis = 3 canonical | PathTracker.Degraded bool reflects the quality state machine output |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|-------------|
| Handler registration via `mgmt.Server.Register()` only; do NOT open new sockets | ARCH-12 | Code review |
| `internal/metrics` types (PathEntry, RTTValue, etc.) are pure data + serialization; no I/O | ARCH-03 §Purity | Pure/Effectful classification table |
| Read PathSnapshots via `Snapshot()`, never via individual field accessors | ARCH-03 §PathSnapshot; go.md rule 12 (no internal pointer leak) | Code review + `TestDaemonPathsList_HandlerRegistered` |
| `rtt_p99_ms` serializes as float64 when SampleCount ≥ 10, string `"pending"` when < 10 | BC-2.06.003 v1.11 EC-003 | `TestPathEntry_RTTValueSerialization` |
| `status` field derived from `PathSnapshot.Degraded` only; no re-implementation of quality state machine | BC-2.06.001; ARCH-03 | `TestPathEntry_StatusFromDegraded` |
| `status` enum is `{active, degraded}` in this story; `"failed"` is reserved for a future liveness-signal story (S-BL.PATH-FAILED-STATUS) and MUST NOT be emitted. When SampleCount<10, `quality` MUST be `"pending"` regardless of `status`; `status` and `quality` are independent fields. | BC-2.06.003 v1.11 PC-1, EC-007; Ruling-4 (wave-6-tranche-a-scope-rulings.md) | `TestDaemonRouterStatus_QualityStatusIndependence` |
| `RTTValue` round-trips: `MarshalJSON(UnmarshalJSON(x)) == x` for both pending and float variants; use `Kind` enum (`PendingKind`, `FloatKind`) not sentinel-based discrimination | F-P2L1-004 | `TestRTTValue_RoundTrip` |
| VP-047 integration test requires the production wiring path (real PathTracker via PathsListSource adapter); MUST NOT inject `synthPathsListSource` directly | VP-047 (transferred from S-5.02); Ruling-3 (wave-6-tranche-a-scope-rulings.md) | `TestVP047_SbctlPathsList_EndToEnd` |
| Do NOT modify cmd/sbctl, internal/paths, or internal/mgmt core transport | S-5.02 scope boundary; S-W5.01 scope boundary | File structure requirements |

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Per go.mod |
| encoding/json | stdlib | RTTValue MarshalJSON + response serialization |
| sync | stdlib | PathTracker.mu (existing) guards Snapshot() reads |

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| internal/metrics/types.go | create | PathEntry, PathsListResponse, RouterMetricsResponse, RTTValue types |
| internal/metrics/handlers.go | create | paths.list, router.metrics, router.status handler implementations |
| internal/metrics/handlers_test.go | create | Unit tests for AC-001–AC-005 (handler registration + serialization) |
| internal/metrics/integration_test.go | create | VP-047 integration test (AC-006): end-to-end sbctl→daemon round-trip |
| internal/mgmt/register_metrics.go | create | Register all three handlers via mgmt.Server.Register() in daemon init |

> Do NOT create or modify: `cmd/sbctl/` (S-5.02 scope), `internal/paths/` (S-5.02 scope),
> `internal/mgmt/server.go` or transport core (S-W5.01 scope).

## Deferred Scope

The following items are explicitly deferred out of S-W5.04 scope:

| Item | Deferred To | Reference |
|------|------------|-----------|
| Production `pathTrackerSource` population from routing subsystem registry (enumerate (SVTN, endpoint) → PathTracker at handler-serve time) | S-BL.PATH-TRACKER-WIRING | Ruling-6 (wave-6-tranche-a-scope-rulings.md); `// #DEFERRED: S-BL.PATH-TRACKER-WIRING` comment in metrics_wire.go |
| `status: "failed"` emission (requires liveness signal in PathSnapshot) | S-BL.PATH-FAILED-STATUS | Ruling-4 (wave-6-tranche-a-scope-rulings.md); BC-2.06.003 v1.11 PC-1 reserved enum |
| `router_addr` real host:port (requires PathSnapshot.RouterAddr enrichment) | S-BL.ROUTER-ADDR | Ruling-1 (wave-6-tranche-a-scope-rulings.md); DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER |

> **#DEFERRED: S-BL.PATH-TRACKER-WIRING** — Production tracker population is deferred. This story delivers the handler surface, response types, and adapter interface; test-only population via `.register()` is the only population path in this wave.

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.8 | 2026-07-01 | spec-steward | F-P4L3-001: Swept all non-changelog BC-2.06.003 version pins from v1.10 → v1.11 (8 sites: AC-002 postcondition cite, AC-003 PC-1 cite, Scope Boundary note, EC-006 EC-007 cite, Token Budget row, Task 1, Arch Compliance Rules EC-003 row, Arch Compliance Rules PC-1/EC-007 row). BC-2.06.003 is at v1.11 (Pass-3 L3 F-L3-001 PC-3 S502-DEFER-3 rewrite + traceability section). |
| 1.7 | 2026-07-01 | story-writer | Ruling-6 propagation (F-P3L1-002, F-L2-01): AC-006 revised — handler surface, response types, and adapter interface in scope; test-only tracker population permitted; production wiring deferred to S-BL.PATH-TRACKER-WIRING; `#DEFERRED` comment requirement added. Task 15a revised: adapter interface defined + test-populated; production wiring deferred; delete residual empty-source dead code if present. §Deferred Scope section added with explicit cross-reference `#DEFERRED: S-BL.PATH-TRACKER-WIRING`. OBS-2 fix: BC-2.06.001 added to §Behavioral Contracts body table (was in bc_traces frontmatter but missing from body table). Pass-3 L3 fixes: F-L3-001 through F-L3-005 propagated. |
| 1.6 | 2026-07-01 | story-writer | Ruling-3 (F-P2L1-003): task step 15a added — wire real PathTracker → PathsListSource adapter in production runControl/runAccess; delete emptyPathsSource/emptyRouterMetricsSource stubs from cmd/switchboard/metrics_wire.go; AC-006 test updated to require production wiring path (MUST NOT inject synthPathsListSource directly). Ruling-4 (F-P2L3-006): status enum retracted to `{active, degraded}` throughout — `"failed"` reserved for S-BL.PATH-FAILED-STATUS; AC-005a rewritten; row (a) status="failed" removed from AC-005a table; EC-006 parenthetical removed; Arch Compliance Rules status-enum row updated; BC-2.06.003 pin bumped v1.9 → v1.10 in Token Budget, AC-002, Tasks step 1, all Arch Compliance rule rows, Scope Boundary. F-P2L3-002: BC-2.06.001 added to bc_traces frontmatter (body already cites BC-2.06.001 in AC-003, AC-005a, Arch Compliance Rules). F-P2L1-004: task step 7a added — fix RTTValue.UnmarshalJSON to use Kind enum (PendingKind, FloatKind) instead of sentinel; Arch Compliance Rules row added for RTTValue round-trip invariant. Expanded negative-test task list at step 5a: data-race regression, status oracle, malformed-args decode, VP-047 field-swap oracle, EC-006 row test. |
| 1.5 | 2026-07-01 | story-writer | F-P1L1-003 router_addr Ruling-1: empty string interim per wave-6-tranche-a-scope-rulings.md Option A; AC-006 narrowed to assert key presence + accept `""`, S-BL.ROUTER-ADDR cross-ref added. F-P1L3-001 AC-005a Degraded==failed mis-anchor corrected: Degraded==true → `"degraded"` (not `"failed"`); `"failed"` is distinct liveness state; EC-006 corrected accordingly; test renamed TestDaemonRouterStatus_QualityStatusIndependence. F-P1L3-002 AC-003 `"ok"`→`"active"` per BC-2.06.003 v1.9 status enum `{active,degraded,failed}`. Sweep BC-2.06.003 pin v1.7→v1.9 in Token Budget, all Arch Compliance rule rows, AC-002, Tasks step 1, Scope Boundary note. |
| 1.4 | 2026-06-30 | product-owner | S502-DEFER-3 closure: extend AC-005 with AC-005a (failed+pending precedence per BC-2.06.003 v1.8 EC-007); add EC-006 to Edge Cases table; add Architecture Compliance Rules row for the precedence ruling; update BC table PC-3 annotation. Bump BC-2.06.003 pin from v1.7 to v1.8. |
| 1.3 | 2026-07-01 | story-writer | F-P8L3-001 (HIGH, frontmatter↔body drift): added VP-062 row to §VP Coverage table. Pass-6 F-P6L3-003 anchor transfer updated vp_traces frontmatter but did not propagate to body table. No semantic change. |
| 1.2 | 2026-07-01 | story-writer | F-P7L3-002 (HIGH, sibling-propagation): swept 5 stale BC-2.06.003 v1.6 pins in body to v1.7. Pass-6 F-P6L3-001 swept S-5.02 v1.7→v1.8 but same-BC sibling S-W5.04 was not covered. No semantic change (BC v1.7 introduced no behavioral change vs v1.6). Also incorporates VP-062 anchor transfer from S-5.02 per Pass-6 F-P6L3-003 (vp_traces update landed at 7b70af0). |
| 1.1 | 2026-06-30 | story-writer | Flesh out stub into full story per Pass-4 Ruling 1 (decisions/S-5.02-pass4-scope-ruling.md). Add §Narrative, §AC-001–AC-006 derived from S-5.02 deferred scope, §Architecture Mapping, §Behavioral Contracts, §VP Coverage, §Edge Cases, §Token Budget, §Tasks (1–16), §Previous Story Intelligence, §Architecture Compliance Rules, §Library & Framework Requirements, §File Structure Requirements. VP-047 integration test assigned to AC-006. acceptance_criteria_count set to 6. |
| 1.0 | 2026-06-30 | product-owner | Stub minted per S-5.02 Pass-4 Ruling 1. ACs/tasks TBD at wave-planning. |
