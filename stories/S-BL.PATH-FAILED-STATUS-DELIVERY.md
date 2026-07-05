---
artifact_id: S-BL.PATH-FAILED-STATUS-DELIVERY
document_type: story-delivery
level: ops
story_id: S-BL.PATH-FAILED-STATUS
version: "1.0"
title: "admit failed to path status enum; derive from PathSnapshot.Failed via precedence"
status: delivered
producer: implementer
timestamp: 2026-07-05T22:34:00Z
modified: 2026-07-05T22:34:00Z
phase: 2
wave: 7-backlog
priority: P2
scope_phase: E
estimated_points: 2
delivered_points: 2
bc_traces:
  - BC-2.06.003   # v1.15 PC-1 — status enum widened to {active, degraded, failed}
  - BC-2.06.001   # AC-002 / AC-003 — path liveness observability
vp_traces:
  - EC-007        # quality-status orthogonality invariant (S502-DEFER-3)
subsystems: [transport-layer, observability]
architecture_modules:
  - internal/paths           # PathTracker.failed + PathSnapshot.Failed
  - internal/metrics         # PathEntryFromSnapshot precedence; enum widening
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-W5.04]
blocks: []
head_sha: 58012ad
branch: feat/s-bl-path-observability
base: origin/develop@b75a2f2 (this commit stacks on 0b229e9 for S-BL.PATH-TRACKER-WIRING)
worktree: .worktrees/path-observability-story
pr: 99
merged: 2026-07-05T22:35:52Z
merge_commit: c098827
drift_consumed:
  - id: Wave-6 Ruling-4
    description: >
      "The `failed` status value is reserved in BC-2.06.003 v1.14 PC-1
      but not yet emitted by any code path; enum stays {active, degraded}
      until a story lifts the reservation." Lifted here: BC-2.06.003
      v1.15 PC-1 now enumerates {active, degraded, failed}; the pre-
      existing Wave-6 panic guard in metrics.PathEntryFromSnapshot
      (which trapped on Status=="failed" as a "should be unreachable"
      assertion) is removed. Enum-closed regression test widened.
  - id: BC-2.06.003 v1.11 failed-enum deferral
    description: >
      Deferral banner in `.factory/specs/bc/BC-2.06.003.md` was updated
      to v1.15 by the spec-steward; this story provides the runtime
      that fulfils it.
  - id: Ruling-9 preservation (not consumed — verified preserved)
    description: >
      Ruling-9 stated "Active==false OR Degraded==true → 'degraded'"
      as a fall-through for the never-alive path with !Active. Verified
      preserved beneath the new Failed branch: the switch chain in
      PathEntryFromSnapshot is `case snap.Failed: "failed"; case
      snap.Degraded || !snap.Active: "degraded"; default: "active"` —
      Ruling-9's clause is intact.
adr_disposition:
  - id: S502-DEFER-3 / EC-007 quality-status orthogonality
    scope: "quality enum {green,yellow,red,pending} vs status enum {active,degraded,failed}"
    verdict: preserved-and-tested
    rationale: >
      "failed" NEVER appears as a quality value. The quality mapping
      in cmd/sbctl/router_status.go:qualityFromPathEntry maps
      status=="failed" → quality="red" (line 70), and pending p99 → "pending"
      regardless of status; the reverse (a quality value influencing
      status) is impossible by construction — status is derived from
      liveness, not from RTT bands. A new metrics test
      (`TestPathEntryFromSnapshot_QualityStatusIndependence`) explicitly
      covers `status:"failed"` + `quality:"pending"` to fence the axis.
---

# S-BL.PATH-FAILED-STATUS — Failed liveness signal (DELIVERY)

## What Landed

`internal/paths.PathTracker` gains a `failed bool` field that latches
when a previously-alive path deactivates via the
`consecutiveMissThreshold = 3` liveness check; `PathSnapshot` exposes
it as `Failed bool`; `internal/metrics.PathEntryFromSnapshot` derives
`PathEntry.Status` via precedence `Failed > Degraded > Active`. The
Wave-6 Ruling-4 reservation on the "failed" enum value is lifted.

Two composed constructs:

1. **`internal/paths` liveness latch** (BC-2.06.003 v1.15 PC-1
   AC-001) — `PathTracker.failed` sets to `true` in the OnProbe
   loss branch *only* when `!firstProbe` (a never-alive path cannot
   fail — it just stays `!Active`). `resetRTT` clears `failed`
   alongside `active` on reactivation, so a recovered path returns
   to `status:"active"` without operator intervention. `Snapshot()`
   populates `Failed` from tracker state.

2. **`internal/metrics.PathEntryFromSnapshot` precedence** —
   deterministic switch chain:
   ```go
   status := "active"
   switch {
   case snap.Failed:
       status = "failed"
   case snap.Degraded || !snap.Active:
       status = "degraded"
   }
   ```
   Failed dominates Degraded dominates Active. The Wave-6
   `panic("status enum admits only {active,degraded}")` guard that
   trapped Status=="failed" as unreachable is removed. Ruling-9 is
   preserved as the `!snap.Active` clause in the second branch — a
   never-alive path (`!Active` but not `Failed`) still falls through
   to `"degraded"`.

## Scope Delivered vs Deferred

**Delivered:**

- `internal/paths/paths.go` — `PathTracker.failed bool` field
  (Failed triggers only when `!firstProbe`); `resetRTT` clears
  failed alongside active; OnProbe loss branch sets failed post-
  threshold if previously alive; `PathSnapshot.Failed bool` field;
  `Snapshot()` populates Failed from tracker state.
- `internal/paths/paths_test.go` — new liveness tests covering:
  never-alive path fails to fail (stays `!Active && !Failed`), 3
  consecutive miss threshold flips Failed=true after having been
  alive, reactivation via successful probe clears Failed, Failed
  persists across intermediate probes until reactivation, snapshot
  copy is decoupled from live state.
- `internal/metrics/handlers.go` — precedence switch chain
  (Failed > Degraded > Active); panic guard removed; docstring
  widened to enumerate the precedence contract with per-branch spec
  citation.
- `internal/metrics/types.go` — package doc v1.14 → v1.15;
  `PathEntry.Status` doc widened to "active | degraded | failed" +
  cite Ruling-4 lift.
- `internal/metrics/handlers_test.go` — `StatusFromDegraded` widened
  to cover Failed>Degraded precedence; `StatusEnumClosed` accepts
  {active,degraded,failed}; new
  `QualityStatusIndependence` covers `status:"failed"` +
  `quality:"pending"` to fence the S502-DEFER-3 / EC-007 axis.

**Deferred (not this story):**

- **Retransmit-driven Failed** — the liveness signal is currently
  probe-driven (`OnProbe` consecutive-miss threshold). Sustained
  retransmit failures + ARQ timeout backpressure as an alternative
  Failed trigger belong to a live-egress observability story.
- **Failed-state event emission** — an operator-facing "path failed"
  log line or metrics counter is not shipped. The status transition
  is observable only via `sbctl paths list`; a push signal (event
  stream or SNMP-adjacent) is a follow-on.

## Findings Consumed

| Finding | Description | Where closed |
|---------|-------------|--------------|
| Wave-6 Ruling-4 | `failed` reserved in v1.14 enum; deferred until a story lifts | `internal/paths/paths.go` Failed latch + `internal/metrics/handlers.go` precedence + panic guard removal |
| BC-2.06.003 v1.11 failed-enum deferral banner | Spec lift shipped in v1.15 | Runtime side: this story |
| Ruling-9 (verified preserved) | `Active==false OR Degraded==true → "degraded"` fall-through | `case snap.Degraded || !snap.Active` in precedence switch |

## Liveness Design + Spec Citation

**Signal source:** `internal/paths/paths.go` `PathTracker.OnProbe`
loss branch (probe-driven). `consecutiveMissThreshold = 3`
(paths.go:28); after three consecutive loss samples, if the tracker
had ever been alive (`!firstProbe`), Failed latches.

**Never-alive protection:** Failed does NOT latch for a path that
has never received a successful probe. The `!firstProbe` guard is
load-bearing — a path that has always been down is "not yet alive,"
not "failed." Test:
`internal/paths/paths_test.go::TestNeverAlivePathDoesNotFail`.

**Reactivation:** `resetRTT` (called from the success branch of
`OnProbe`) clears both `active=false` and `failed=false`. A
recovered path returns to `status:"active"` without operator
intervention. Test:
`TestReactivationClearsFailed`.

**Precedence at the metrics boundary:** Failed > Degraded > Active.
The metrics layer never re-derives status — it observes the
snapshot. Test: `TestStatusFromDegraded_FailedDominates`.

**Ruling-9 preservation:** the fall-through `case snap.Degraded ||
!snap.Active` beneath the `case snap.Failed` clause preserves the
never-alive path's `"degraded"` mapping. A path with `!Active &&
!Degraded && !Failed` (never-alive, no RTT sample) still surfaces
as `"degraded"` at the operator boundary, matching v1.14 behavior.

**Spec citations:**

- BC-2.06.003 v1.15 PC-1 — status enum widened to {active, degraded,
  failed}.
- BC-2.06.001 AC-002 / AC-003 — path liveness observability contract.
- Wave-6 Ruling-4 (reservation lifted by this story).
- Wave-6 Ruling-9 (preserved by the precedence chain).
- S502-DEFER-3 / EC-007 (quality-status orthogonality fenced).

## Sentinel / SPEC Impact

**sbctl round-trip surface:** `sbctl paths list` and `sbctl router
status` both serialize the widened `PathEntry.Status`. The wire
schema in `cmd/sbctl/paths_list.go:8` already documented
`"status":"active|degraded|failed"` — that documentation matches
runtime as of this story. The quality mapping in
`cmd/sbctl/router_status.go:qualityFromPathEntry` handles
`status=="failed" → "red"` (line 70) unconditionally, but never
promotes a non-failed status to failed — the orthogonality axis is
one-way.

**SPEC-1..SPEC-5 (spec-runner.sh):** unchanged. These sentinels
cover unreachable-daemon (SPEC-1..SPEC-2) and usage-error
(SPEC-3..SPEC-5) paths; the happy-path status enum surface is not
exercised by any current SPEC assertion. All 5/5 pass on worktree
HEAD.

**INV-1..INV-10 + INV-7/INV-8 (smoke-quick):** unchanged. Sentinels
exercise process/daemon/sbctl lifecycle, not status-enum content.
All 14/14 pass.

**T3-2/T3-4 (tier3-tutorial):** unchanged. Tutorial exercises taxonomy
+ config extraction, not paths.list happy-path enumeration. All 4/4
pass.

## Test Inventory

**New/widened tests:**

- `internal/paths/paths_test.go` — 225 new lines covering
  never-alive protection, threshold-triggered failure,
  reactivation-clears-failed, failure persistence across
  intermediate probes, snapshot decoupling.
- `internal/metrics/handlers_test.go` — widened:
  - `TestStatusFromDegraded` covers Failed>Degraded precedence.
  - `TestStatusEnumClosed` accepts {active,degraded,failed}.
  - `TestPathEntryFromSnapshot_QualityStatusIndependence` (new) —
    fences S502-DEFER-3 / EC-007 by asserting `status:"failed"` +
    `quality:"pending"` coexists.

**Runs (all clean):**

- `just fmt` — no changes.
- `go vet ./internal/paths/... ./internal/metrics/...` — no output.
- `go test -race -count=3 ./internal/paths/... ./internal/metrics/...` — green (paths 2.066s, metrics 5.003s).
- `go test ./... -count=1` — all 21 packages green.
- `golangci-lint run ./...` — 0 issues.
- `just smoke-quick` — 14/14 (`.smoke/20260705T223612Z/report.jsonl`).
- `bash test/smoke/spec-runner.sh` — 5/5 (`.smoke/20260705T223618Z-spec/report.jsonl`).
- `bash test/smoke/tier3-tutorial.sh` — 4/4 (`.smoke/20260705T223619Z-tier3/report.jsonl`).
- CI on PR #99 — all checks green.

## Blast Radius

**1. Operator-visible surfaces touched:** `sbctl paths list` and
`sbctl router status` may now emit `"status":"failed"` (previously
this value was reserved and the metrics layer would panic if the
snapshot ever produced it — a defense that has now been removed
because the enum is legitimate). Downstream: `sbctl router status`
maps `status=="failed"` → `quality:"red"` per the pre-existing
`qualityFromPathEntry` chain. No new CLI flag, no new config field,
no new error taxonomy code. The wire schema documentation in
`cmd/sbctl/paths_list.go` already declared the widened enum; this
story makes the runtime match the docstring.

**2. Silent-failure risk:** low. The `!firstProbe` guard prevents
a spurious Failed on paths that never got a first probe (which
would over-report failure in freshly-started daemons); the
reactivation reset ensures a recovered path exits Failed without
operator ceremony. The removed panic guard has been replaced by
enum-closed regression test coverage — if the enum ever silently
widens again, `TestStatusEnumClosed` catches it. Orthogonality is
fenced by `TestPathEntryFromSnapshot_QualityStatusIndependence`.

**3. Smoke gate touched:** none. 14/14 sentinels + 5/5 spec-runner
+ 4/4 tier3 all pass. No smoke asset modified. Wire schema
documentation matches runtime post-story.

## Follow-ons (Filed as Deferrals, not Regressions)

- **Retransmit-driven Failed trigger** — sustained retransmit +
  ARQ timeout backpressure as an alternative Failed source. Belongs
  to a live-egress observability story alongside
  S404-OBS-F/S404-LOW-1 re-anchoring.
- **Failed-state event emission** — operator-facing push signal for
  a path transitioning to Failed (log line, metrics counter,
  events RPC). Currently observable only via poll of `sbctl paths
  list`.

## Commit Trail (on `feat/s-bl-path-observability`)

1. `58012ad` — `feat(paths,metrics): S-BL.PATH-FAILED-STATUS — failed liveness signal`

Base: stacks on `0b229e9` (S-BL.PATH-TRACKER-WIRING); ultimate base
is `origin/develop@b75a2f2`.
Head: `58012ad`.

**PR:** #99, merged 2026-07-05T22:35:52Z as merge commit `c098827`.
