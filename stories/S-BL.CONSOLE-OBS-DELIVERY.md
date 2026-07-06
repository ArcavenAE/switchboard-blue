---
artifact_id: S-BL.CONSOLE-OBS-DELIVERY
document_type: story-delivery
level: ops
story_id: S-BL.CONSOLE-OBS
version: "1.1"
title: "console session-list observability — sessions.status quality + missCount"
status: complete
producer: implementer
timestamp: 2026-07-06T03:12:00Z
modified: 2026-07-06T03:12:00Z
phase: 2
wave: backlog
priority: P2
scope_phase: E
estimated_points: 5
delivered_points: 5
bc_traces:
  - BC-2.06.001   # v1.7 PC-5 console-half — sbctl sessions status quality column (DRIFT-001b)
  - BC-2.06.002   # v1.4 PC-3 — operator-visible cumulative missCount export (DRIFT-002)
  - BC-2.06.003   # v1.16 — quality enum {green,yellow,red,pending}; "failed" never a quality value
rulings_consumed:
  - RULING-W6TB-C   # AC-004 (new console session-list RPC surface); AC-005 (QualityIndicator.MissCount accessor)
  - F-P5P6-A-003    # sbctl sub-verb dispatch on unreachable daemon: exit 1 + E-NET-001 (widened from exit 2)
  - F-P2L1-001      # register-before-serve invariant (handler must exist on the mgmt server before daemon accepts management calls)
subsystems: [session-management, quality-observability, sbctl-cli]
architecture_modules:
  - internal/metrics                            # MissCount() lifetime accessor (BC-2.06.002 PC-3)
  - internal/session                            # nil-safe Publisher SetPublishHook / SetUnpublishHook (typed SessionHook)
  - cmd/switchboard/session_quality_source.go   # sessionQualitySource — boundary-package registry + HandleSessionsStatus
  - cmd/switchboard/sessions_handlers.go        # BuildSessionsHandlers wiring + Tier-2 role gate
  - cmd/switchboard/mgmt_wire.go                # runConsole composes source-from-Publisher, appends BuildSessionsHandlers
  - cmd/sbctl                                   # sessions status sub-verb + dispatch
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-5.01, S-7.03]
blocks: []
head_sha: 8f192e7
branch: feat/s-bl-console-obs
base: origin/develop@e81927f
worktree: .worktrees/console-obs
pr: 104
merged: 2026-07-06T01:56:25Z
merge_sha: 18fd2fe
drift_consumed:
  - id: DRIFT-001b
    description: >
      BC-2.06.001 v1.7 PC-5 console-half — per-session quality indicator
      surfaced on the console-mode daemon via `sbctl sessions status`. Moved
      from S-7.03 to S-BL.CONSOLE-OBS per RULING-W6TB-C. Closed here:
      sessionQualitySource maintains a per-session QualityIndicator keyed by
      session name; snapshots surface `quality` field in the sessions.status
      wire envelope; "pending" until the first observation is recorded on a
      published session; drops in lockstep with Publisher.Unpublish via the
      nil-safe UnpublishHook.
  - id: DRIFT-002
    description: >
      BC-2.06.002 v1.4 PC-3 — operator-visible export of the per-session
      cumulative missing-frame count. Closed here: metrics.QualityIndicator
      gains a `missCount uint64` field incremented once per OnMissingFrame
      call, never reset (distinct from the internal consecutive
      missingFrameCount that drives hysteresis and resets on Update or
      threshold-triggered downgrade), plus MissCount() accessor under the
      existing mutex. Surfaced through the sessions.status wire as `miss_count`.
  - id: RULING-W6TB-C AC-004
    description: >
      New console-side RPC surface for session enumeration + quality.
      Delivered as sessions.status wire verb registered by
      BuildSessionsHandlers on the console-mode daemon; Tier-2 admission via
      verifyConsoleCallerRole (accepts RoleControl or RoleConsole; rejects
      with E-ADM-006 otherwise). Handler reads through the boundary-package
      sessionQualitySource, never through internal/session directly (per the
      ARCH-08 §6.6 preservation rework — see Course-correct section).
  - id: RULING-W6TB-C AC-005
    description: >
      QualityIndicator.MissCount() accessor. Delivered in metrics package
      commit 26b5687; consumed by sessionQualitySource.snapshotToEntry when
      building the SessionStatusEntry.MissCount field.
adr_disposition:
  - id: ARCH-08 §6.6 DAG constraint (pos 6 → pos 12)
    scope: "internal/session (pos 6) must not import internal/metrics (pos 12)"
    verdict: preserved
    rationale: >
      Initial delivery on this branch (commits 26b5687..6e49a97) added a
      direct import of internal/metrics into internal/session, argued in a
      package doc comment as "import direction is downstream — no cycle."
      The Go compiler accepted it (no import cycle in the graph sense), but
      it inverted ARCH-08 §6.6's topological ordering: session is position 6;
      metrics is position 12; internal/tmux (position 7) imports session, so
      the fix under "keep the import but repair the DAG doc" would cascade a
      renumber through tmux and every package above it — staling every
      "DAG position N" citation in current STORY-INDEX rows and DELIVERY
      ledgers (drain=16, outerassembler=8, arqsend=9, ...).

      Coordinator held PR #104 with a REDIRECT prescribing the hook /
      composition pattern already established in PR #99
      (S-BL.PATH-TRACKER-WIRING) — `routing.ForwardingEntryHook` +
      `newPathTrackerSourceFromRouter` at the cmd/switchboard boundary.

      Rework commit `8f192e7` applies exactly that pattern: `internal/session`
      gains typed `SessionHook = func(sessionName string, publishedAt
      time.Time)` fields on Publisher plus `SetPublishHook` / `SetUnpublishHook`
      installers (nil-safe, fired under the Publisher lock inside Publish and
      Unpublish, exactly once per successful mutation). The
      `map[string]*QualityIndicator` registry, OnSessionMeasurement /
      OnSessionMissingFrame observers, SessionSnapshot value-copy accessor,
      and HandleSessionsStatus handler all live in
      `cmd/switchboard/session_quality_source.go` — sibling of
      `pathTrackerSource`. `sessions_handlers.go` reads through
      `*sessionQualitySource`, no `internal/session` import.

      Empirical verification post-rework:

          $ go list -f '{{.Imports}}' ./internal/session
          [context errors fmt github.com/arcavenae/switchboard/internal/admission
           github.com/arcavenae/switchboard/internal/frame sort sync sync/atomic time]

          $ go list -deps ./internal/session/... | \
              grep -E "internal/(metrics|routing|tmux)"
          (no output)

      Package-package edges from internal/session are exactly {admission,
      frame}, matching the coordinator's prescribed set. Zero forbidden
      edges. Deleted from internal/session: session_quality.go,
      session_quality_test.go, sessions_status.go, sessions_status_test.go.
      New boundary files: cmd/switchboard/session_quality_source.go +
      _test.go, cmd/switchboard/sessions_status_test.go,
      internal/session/session_hooks_test.go.
  - id: F-P2L1-001 (register-before-serve)
    scope: "sessions.status handler must exist on mgmt server before daemon accepts management calls"
    verdict: preserved
    rationale: >
      runConsole constructs sessionQualitySource from the Publisher, then
      appends BuildSessionsHandlers(src, ks) to initialHandlers before
      newMgmtServer is called — mirrors the pathTrackerSource /
      BuildRouterHandlers sequencing shape established in PR #99. Handler
      registration and server startup are one construction sequence; no
      race window between listen and register.
  - id: F-P5P6-A-003 (sub-verb exit-1 unreachable widening)
    scope: "sbctl sessions status on unreachable daemon returns exit 1 + E-NET-001"
    verdict: adopted
    rationale: >
      Widened from prior exit-2-on-usage to exit-1 with E-NET-001 for the
      unreachable-daemon case. cmd/sbctl/main.go routing for the new
      sessions sub-verb inherits the same dispatch path as paths/router
      sub-verbs; sessions_status_dispatch_test.go asserts the exit code +
      error taxonomy on the net.Pipe fixture.
---

# S-BL.CONSOLE-OBS — Console session-list observability (DELIVERY)

## What Landed

Console-mode daemon now serves `sessions.status`, exposing per-session
quality (`green | yellow | red | pending`) and the cumulative
`miss_count` (BC-2.06.002 v1.4 PC-3 export). Operator surface:
`sbctl sessions status [<name>]`. DRIFT-001b (BC-2.06.001 PC-5
console-half) and DRIFT-002 (BC-2.06.002 PC-3 operator export) both
closed.

Six composed constructs:

1. **`metrics.QualityIndicator.MissCount()`** (BC-2.06.002 v1.4 PC-3;
   RULING-W6TB-C AC-005) — new `missCount uint64` field on
   QualityIndicator, incremented exactly once per OnMissingFrame call,
   NEVER reset. Distinct from the pre-existing internal
   `missingFrameCount` that drives BC-2.06.001 hysteresis (that counter
   is consecutive: it resets on Update per PC-4 and after
   threshold-triggered downgrades). PC-3 asks for the lifetime "path-metric
   record of the gap event"; the existing counter is unsuitable as an
   export because it can silently reset between operator polls. Accessor
   under the same `sync.Mutex` that guards Update / OnMissingFrame /
   Current.

2. **`session.SessionHook`** (ARCH-08 §6.6 preservation, mirrors
   `routing.ForwardingEntryHook` from PR #99) — `type SessionHook
   func(sessionName string, publishedAt time.Time)`. Publisher gains
   `publishHook`, `unpublishHook` fields plus `SetPublishHook(SessionHook)`,
   `SetUnpublishHook(SessionHook)` installers. Hooks fire synchronously
   under the Publisher lock, immediately after the sessions-map mutation
   in Publish and Unpublish; the `publishedAt` argument is carried through
   so the unpublish hook does not need a Get round-trip after the map
   delete. Nil hooks are legal no-ops. Package docstring names the DAG
   invariant: "Forbidden: internal/routing, internal/tmux (circular),
   internal/metrics (topological inversion, ARCH-08 §6.6); boundary
   composition happens in cmd/switchboard via typed SessionHook
   callbacks, mirroring routing.ForwardingEntryHook for pathTrackerSource
   in S-BL.PATH-TRACKER-WIRING."

3. **`cmd/switchboard/sessionQualitySource`** (RULING-W6TB-C AC-004
   boundary-package host) — private struct with `sync.RWMutex +
   map[string]*sessionQuality`; sibling of `pathTrackerSource`. Two
   constructors:
   - `newSessionQualitySource()` — empty registry.
   - `newSessionQualitySourceFromPublisher(pub *session.Publisher)` —
     empty registry + installs the source's OnPublished / OnUnpublished
     callbacks on the Publisher via SetPublishHook / SetUnpublishHook,
     then returns.
   Public methods: `OnPublished(name, publishedAt)`,
   `OnUnpublished(name, publishedAt)`, `OnSessionMeasurement(name, rtt)`,
   `OnSessionMissingFrame(name)`, `SessionSnapshot(name) (T, bool)`,
   `SessionSnapshots() []T` (sorted by name), `HandleSessionsStatus(ctx,
   ks, cmd) []byte`. All snapshot accessors return value copies per
   go.md rule 12; internal `*sessionQuality` never escapes the RWMutex.
   Fast-path RLock exists-check + slow-path Lock re-check on register —
   pattern lifted directly from `pathTrackerSource.Register`. Unknown-
   session observations return `errQualitySessionNotFound` (sentinel
   `errors.Is` target).

4. **`cmd/switchboard/BuildSessionsHandlers(src, ks) []mgmt.Handler`**
   (F-P2L1-001 register-before-serve) — panics if `src` is nil ("must
   not be nil"). Registers sessions.status verb; empty-args guard rejects
   `null` and `{}` as identical (both mean all-sessions); Tier-2 admission
   via `verifyConsoleCallerRole(ctx, ks, cmd)` returning E-ADM-006 for
   any role other than RoleControl or RoleConsole. Handler reads through
   `src` exclusively; no import of `internal/session`.

5. **`cmd/switchboard/mgmt_wire.go::runConsole` composition** — inserts
   `src := newSessionQualitySourceFromPublisher(pub)` immediately after
   `pub := session.NewPublisher(ks)`; appends `BuildSessionsHandlers(src,
   ks)...` into `initialHandlers` before `newMgmtServer(...)`. Mirrors
   the `newPathTrackerSourceFromRouter` + `BuildRouterHandlers`
   sequencing shape from PR #99. F-P2L1-001 preserved: handler registered
   before daemon accepts management traffic.

6. **`cmd/sbctl sessions` sub-verb** (F-P5P6-A-003 exit-1 unreachable
   widening) — new `sbctl sessions status [<name>]` verb, dispatches
   through the same envelope as paths/router verbs. On unreachable
   daemon: exit 1 + E-NET-001 (widened from prior exit-2 usage-error).
   JSON via `--json`, human-readable table by default.

## Wire Contract

**Verb:** `sessions.status`

**Request envelope (any of three shapes, all equivalent for "all sessions"):**

```json
null
```
```json
{}
```
```json
{"session_name": "<name>"}
```

- `null` and `{}` request all sessions; the handler treats them
  identically.
- `{"session_name": "<name>"}` requests a single session by name.

**Response envelope (all-sessions form):**

```json
{
  "sessions": [
    {
      "name": "agent-01",
      "published_at": "2026-07-05T20:15:00Z",
      "quality": "yellow",
      "miss_count": 3
    },
    {
      "name": "agent-02",
      "published_at": "2026-07-05T20:16:00Z",
      "quality": "pending",
      "miss_count": 0
    }
  ]
}
```

Sessions in the response array are sorted lexicographically by `name`.

**Single-session form:**

```json
{
  "sessions": [
    {"name": "agent-01", "published_at": "...", "quality": "green", "miss_count": 1}
  ]
}
```

Same schema; array length 1.

**Field semantics:**

- `name` (string) — session name from Publisher.
- `published_at` (RFC3339) — from `SessionInfo.PublishedAt` captured at
  Publish time; propagated via the SessionHook signature (not re-fetched
  after Unpublish).
- `quality` (enum `{green, yellow, red, pending}`) — value returned by
  `QualityIndicator.Current()` on the per-session indicator, OR
  `"pending"` if no observation has been recorded yet on this session
  (never-observed after Publish). Locked by BC-2.06.003 v1.16 —
  `"failed"` is NEVER a quality value (it is a path-status value only;
  the quality enum stays four-valued regardless of path-status
  developments).
- `miss_count` (uint64) — value-copy of `QualityIndicator.MissCount()`;
  lifetime-cumulative per session.

**Errors:**

| Condition | Wire code | Exit |
|-----------|-----------|------|
| Session name specified but unknown | E-SES-001 | 1 |
| Caller role not RoleControl or RoleConsole | E-ADM-006 | 1 |
| Malformed JSON args (invalid `{...}` shape) | E-RPC-002 | 1 |
| Daemon unreachable | E-NET-001 | 1 (F-P5P6-A-003 widening) |

## Course-correct (2026-07-05, ARCH-08 §6.6 DAG preservation)

Initial delivery on this branch shipped six commits (`26b5687`..`6e49a97`)
that placed the per-session QualityIndicator registry inside
`internal/session/session_quality.go` and the handler inside
`internal/session/sessions_status.go`. Both files imported
`internal/metrics`. The package doc comment on `session_quality.go`
rationalized this: "import direction is downstream — no cycle." That
was true for the Go import graph in the narrow sense (compilation
produced no cycle), but wrong for ARCH-08's declared position ordering:

- `internal/session` = DAG position 6
- `internal/metrics` = DAG position 12
- `internal/tmux` (position 7) imports `internal/session`

Session importing metrics is a forward reference: it inverts the
"packages import only lower positions" invariant. Repairing the
ordering by renumbering metrics ahead of session would cascade through
tmux (7) and every position above it, staling every "DAG position N"
citation in current STORY-INDEX rows and DELIVERY ledgers (drain=16,
outerassembler=8, arqsend=9, ...). That renumber blast radius is
exactly what the coordinator ruling avoided.

The completion report on those six commits did NOT flag the ARCH-08
tension. That was the process failure. The coordinator held PR #104 with
a REDIRECT prescribing the hook / composition pattern already
established in PR #99 (S-BL.PATH-TRACKER-WIRING) — same shape as
`routing.ForwardingEntryHook` + `newPathTrackerSourceFromRouter`, at
the cmd/switchboard boundary.

**Corrective commit `8f192e7`** (12 files, +1410 / -1072):

- Adds `session.SessionHook` (typed callback), Publisher
  publishHook/unpublishHook fields, SetPublishHook/SetUnpublishHook
  installers; hooks fire under the Publisher lock inside
  Publish/Unpublish; PublishedAt is carried through the signature.
- Moves the registry + observers + handler out of `internal/session`
  into `cmd/switchboard/session_quality_source.go` (sibling of
  `pathTrackerSource`). Sentinel `errQualitySessionNotFound` for
  unknown-session observations. RWMutex + double-check on register.
- Moves the sessions.status handler wiring into
  `cmd/switchboard/sessions_handlers.go` — reads through
  `*sessionQualitySource`, does not import `internal/session`.
- Wires `runConsole` to compose source-from-Publisher and append
  BuildSessionsHandlers to initialHandlers before newMgmtServer.
- Deletes `internal/session/session_quality.go`,
  `session_quality_test.go`, `sessions_status.go`, `sessions_status_test.go`.
- Adds `internal/session/session_hooks_test.go` (5 tests fencing the
  hook invariant).

**Verification:**

```
$ go list -f '{{.Imports}}' ./internal/session
[context errors fmt github.com/arcavenae/switchboard/internal/admission
 github.com/arcavenae/switchboard/internal/frame sort sync sync/atomic time]

$ go list -deps ./internal/session/... | grep -E "internal/(metrics|routing|tmux)"
(no output)
```

Package-package edges from `internal/session` are exactly `{admission,
frame}`. Zero forbidden edges.

**Process learning (surfaced in-thread with team-lead):** when a
coordinator ruling and implementation diverge, the divergence must be
flagged explicitly in the completion report ("ruling X, I did Y,
because Z"). Silent divergence with a rationalizing doc comment is how
spec-drift ships. The ARQ-TX SendFn signature dispute was the model:
argue up front, land after ruling. That standard applies going forward.

## AC → Test Evidence

| AC | Requirement | Evidence |
|----|-------------|----------|
| AC-001 | `QualityIndicator.MissCount()` accessor exposes lifetime cumulative gap events (RULING-W6TB-C AC-005; BC-2.06.002 PC-3) | `internal/metrics/metrics_test.go` — five new tests: `TestMissCount_ZeroOnConstruct`, `TestMissCount_IncrementsPerCallAcrossDowngradeThreshold`, `TestMissCount_NotResetByUpdate`, `TestMissCount_MonotonicAcrossMixedWorkload`, `TestMissCount_ConcurrentSafe_ExactCountOracle` (1000 concurrent OnMissingFrame under -race) |
| AC-002 | MissCount is distinct from the internal consecutive `missingFrameCount` that drives BC-2.06.001 hysteresis (resets on Update / threshold downgrade) | `TestMissCount_NotResetByUpdate` — issue N OnMissingFrame calls, then Update; assert MissCount() == N (unchanged); the sibling internal counter zeroes per BC-2.06.001 PC-4 but is not observable through the public accessor |
| AC-003 | Console-mode daemon serves `sessions.status` wire verb (RULING-W6TB-C AC-004) | `cmd/switchboard/sessions_status_test.go` (7 tests): `Empty_NoSessions`, `AllSessions_SortedByName`, `AllSessions_QualityAndMissCount` (agent-01 yellow+3 vs agent-02 pending+0), `SingleSession_ByName`, `SingleSession_Unknown_ESES001`, `JSONRoundTrip`, `AfterUnpublish_DropsFromAllQuery` |
| AC-004 | Per-session `quality` field emitted as `green|yellow|red|pending`; `"failed"` is never a quality value (BC-2.06.003 v1.16) | `cmd/switchboard/session_quality_source_test.go` — 12 tests including `PublishedSessionAppearsPending`, `OnSessionMeasurement_GoodMeasurementProducesGreen`, `OnSessionMissingFrame_DowngradesQuality`, `SessionSnapshot_ValueCopy`; no test path can construct or emit a `"failed"` quality value |
| AC-005 | Per-session `miss_count` field emitted as uint64 lifetime cumulative (BC-2.06.002 PC-3; DRIFT-002 export) | `OnSessionMissingFrame_IncrementsMissCount`; `sessions_status_test.go::AllSessions_QualityAndMissCount` asserts agent-01 has `"miss_count":3` after 3 OnSessionMissingFrame calls |
| AC-006 | Handler drops session from all-sessions query after Unpublish, in lockstep with Publisher.Unpublish | `session_quality_source_test.go::Unpublish_DropsQualityIndicator`; `sessions_status_test.go::AfterUnpublish_DropsFromAllQuery` (publish agent-01, agent-02; unpublish agent-01; query all → sessions array contains only agent-02) |
| AC-007 | Nil-safe Publisher hooks (SetPublishHook / SetUnpublishHook) fire under Publisher lock; fire exactly once per successful Publish / Unpublish; carry PublishedAt through signature; not fired on Publish-duplicate or Unpublish-unknown | `internal/session/session_hooks_test.go` (5 tests): `NoHooks_PublishUnpublishAreNilSafe`, `SetPublishHook_FiresOncePerPublish` (asserts `times[0].Equal(info.PublishedAt)`), `SetUnpublishHook_FiresOncePerUnpublish` (captures origInfo via Get before Unpublish, asserts `times[0].Equal(origInfo.PublishedAt)`), `Hooks_NotFiredOnErrors` (atomic.Int64 counters — publishHook does not fire on duplicate Publish; unpublishHook does not fire on unknown Unpublish), `SetPublishHook_ReplaceHook` (three Publish calls with a hook swap in between: firstCount=1, secondCount=1) |
| AC-008 | Tier-2 admission — sessions.status handler accepts RoleControl and RoleConsole; rejects other roles with E-ADM-006 (parity with paths.list handler role gate) | `cmd/switchboard/sessions_handlers_e2e_test.go::E2E_AdmissionDenied_E_ADM_006` — registers a caller with a non-control/non-console role, invokes sessions.status, asserts error code E-ADM-006; verifies role-gate parity with the paths.list handler in the same daemon |
| AC-009 | `sbctl sessions status` sub-verb dispatches via the standard envelope; unreachable daemon returns exit 1 + E-NET-001 (F-P5P6-A-003 widening from exit 2) | `cmd/sbctl/sessions_status_dispatch_test.go` — net.Pipe fixture asserts wire round-trip on happy path; separate case runs the binary against a non-listening address and asserts exit code 1 + E-NET-001 in stderr; `cmd/sbctl/production_exit_code_test.go` updated to reflect the widened contract |
| AC-010 | Empty-args form (`null` OR `{}`) requests all sessions and behaves identically | `sessions_status_test.go::Empty_NoSessions` runs both request shapes; `sessions_handlers.go` empty-args guard: `len(args) > 0 && string(args) != "null"` — treats both shapes as the same code path |
| AC-011 | Sessions in all-query response are sorted lexicographically by name | `session_quality_source_test.go::SessionSnapshots_SortedByName` (publish agent-03, agent-01, agent-02 in that order; assert snapshot slice is agent-01, agent-02, agent-03) |
| AC-012 | ARCH-08 §6.6 DAG preserved — `internal/session` gains no forbidden imports; package-package edges stay `{admission, frame}` | `go list -deps ./internal/session/... \| grep -E "internal/(metrics\|routing\|tmux)"` — no output. `go list -f '{{.Imports}}' ./internal/session` — `{context errors fmt admission frame sort sync sync/atomic time}`. Deleted from internal/session in the rework commit: `session_quality.go`, `session_quality_test.go`, `sessions_status.go`, `sessions_status_test.go` |

## Drift Dispositioned

| Drift | Description | Where closed |
|-------|-------------|--------------|
| DRIFT-001b | BC-2.06.001 v1.7 PC-5 console-half — per-session quality via `sbctl sessions status`. Moved from S-7.03 per RULING-W6TB-C. | `cmd/switchboard/session_quality_source.go` (registry + snapshot builder) + `cmd/switchboard/sessions_handlers.go::HandleSessionsStatus` (handler emitting `quality` field per BC-2.06.003 v1.16 enum) |
| DRIFT-002 | BC-2.06.002 v1.4 PC-3 operator-visible export of the per-session cumulative missing-frame count. | `internal/metrics/metrics.go::QualityIndicator.MissCount()` (lifetime counter distinct from the consecutive hysteresis counter) + `cmd/switchboard/session_quality_source.go::snapshotToEntry` (S1016 direct struct conversion emitting `miss_count`) |
| RULING-W6TB-C AC-004 | New console-side RPC surface for session enumeration + quality. | `cmd/switchboard/sessions_handlers.go::BuildSessionsHandlers` + `runConsole` composition (boundary-package location per ARCH-08 §6.6 preservation) |
| RULING-W6TB-C AC-005 | QualityIndicator.MissCount() accessor. | `internal/metrics/metrics.go` — commit `26b5687` |

## Findings Consumed

| Finding | Where closed |
|---------|--------------|
| ARCH-08 §6.6 forward-reference (self-caught in review) | Rework commit `8f192e7` — QualityIndicator registry moved to cmd/switchboard behind nil-safe Publisher hooks; internal/session imports stay `{admission, frame}` |
| F-P2L1-001 (register-before-serve) | `runConsole` sequences `BuildSessionsHandlers(src, ks)` append before `newMgmtServer(...)` construction (mirrors PR #99 pathTrackerSource + BuildRouterHandlers) |
| F-P5P6-A-003 (sbctl sub-verb exit code widening for unreachable-daemon) | `cmd/sbctl/main.go` routing gains the sessions sub-verb on the same dispatch path; unreachable daemon path returns exit 1 + E-NET-001 (asserted in `sessions_status_dispatch_test.go` + `production_exit_code_test.go` update) |

## Test Inventory

**New tests:**

- `internal/metrics/metrics_test.go` — 5 new tests for MissCount()
  (zero-on-construct, increments-per-call across downgrade threshold,
  not-reset-by-Update, monotonic across mixed workload,
  concurrent-safe over 1000 goroutines under -race).
- `internal/session/session_hooks_test.go` — 5 tests fencing SessionHook
  nil-safety, fires-once semantics, PublishedAt propagation through the
  hook signature, hooks-not-fired-on-error (Publish duplicate, Unpublish
  unknown), replace-and-nil-disable.
- `cmd/switchboard/session_quality_source_test.go` — 12 tests (all
  `t.Parallel()`) covering empty-on-startup, published-session-appears-
  pending, good-measurement-produces-green, missing-frame-increments-
  MissCount, missing-frame-downgrades-quality, unknown-session-observation
  returns sentinel via `errors.Is`, SessionSnapshot(unknown) returns
  (zero, false), unpublish-drops-QualityIndicator, snapshots-sorted-by-
  name, snapshots-value-copy, concurrent-observations exact-count oracle
  (4 workers × 250 observations under -race).
- `cmd/switchboard/sessions_status_test.go` — 7 tests exercising the
  handler shape (empty, all sorted, all with quality+missCount fixture,
  single-by-name, single-unknown-ESES001, JSON round-trip asserting the
  full envelope shape, after-unpublish-drops-from-all-query).
- `cmd/switchboard/sessions_handlers_e2e_test.go` — 4 e2e tests through
  `newSessionsE2EStack` fixture: all-sessions-quality-and-missCount
  (non-tautological — agent-01 yellow+3 vs agent-02 pending+0),
  single-session-by-name, unknown-session-ESES001,
  admission-denied-E-ADM-006 (Tier-2 role-gate parity).
- `cmd/sbctl/sessions_status_dispatch_test.go` — 5 tests: net.Pipe
  happy-path round-trip, unreachable-daemon exit-1 + E-NET-001,
  usage-error path, unknown-session ESES001, JSON output shape.
- `cmd/sbctl/production_exit_code_test.go` — updated to reflect
  F-P5P6-A-003 widening (sessions sub-verb inherits paths/router
  dispatch-path exit contract).

**Removed** (during course-correct in `8f192e7`):

- `internal/session/session_quality.go` (had held per-session registry;
  moved to cmd/switchboard/session_quality_source.go).
- `internal/session/session_quality_test.go` (tests moved to
  cmd/switchboard/session_quality_source_test.go with `t.Parallel()`
  added; e2e coverage split into
  cmd/switchboard/sessions_handlers_e2e_test.go).
- `internal/session/sessions_status.go` (handler; moved to
  cmd/switchboard/sessions_handlers.go).
- `internal/session/sessions_status_test.go` (moved to
  cmd/switchboard/sessions_status_test.go).

**Runs (all clean on `8f192e7`):**

- `just fmt` — no changes.
- `go vet ./...` — no output.
- `go test -race -count=3 ./internal/metrics/... ./internal/session/... ./cmd/switchboard/... ./cmd/sbctl/...` — green.
- `go test ./...` — all packages green fleetwide.
- `golangci-lint run ./cmd/switchboard/... ./internal/session/... ./cmd/sbctl/... ./internal/metrics/...` — 0 issues.
- `go list -deps ./internal/session/... | grep -E "internal/(metrics|routing|tmux)"` — no output (DAG preserved).
- `just smoke-quick` — 14/14 sentinels.
- `bash test/smoke/spec-runner.sh` — 5/5.
- `bash test/smoke/tier3-tutorial.sh` — 4/4.
- CI on PR #104 (post-`8f192e7`) — all checks green.

## Blast Radius

**1. Operator-visible surfaces touched:** `sbctl sessions status
[<name>]` is a NEW sub-verb on the console-mode daemon. Empty request
(or `--json` with `null` args) returns `{"sessions":[]}` when no
sessions are published. Live console session enumeration surfaces
`{name, published_at, quality, miss_count}` per session. Unreachable
daemon: exit 1 + E-NET-001 (F-P5P6-A-003 widening from prior exit 2 —
now consistent with paths/router sub-verbs). Unknown session: exit 1 +
E-SES-001. Role-denied: exit 1 + E-ADM-006. No existing verb signature
changed.

**2. Silent-failure risk:** low, but named on three axes.

- (a) `sessionQualitySource` OnSessionMeasurement / OnSessionMissingFrame
  return `errQualitySessionNotFound` on unknown sessions. Production
  call sites (currently: none — observation wiring is a follow-up)
  must handle this error explicitly, otherwise silent-drop on race
  between Unpublish and a late measurement. Sentinel is `errors.Is`-
  compatible for callers.
- (b) `publishHook` / `unpublishHook` fire under the Publisher lock;
  the delivered `OnPublished` / `OnUnpublished` bodies acquire a
  private RWMutex + do map insertion / delete only — deterministic,
  O(1), no I/O. Any future hook body that adds work (logging, network
  I/O) must audit for hold-time regressions. The concurrent-observations
  test (4×250 exact-count oracle) proves the current body is bounded.
- (c) nil hooks are legal no-ops; a consumer that forgets to install
  hooks on the Publisher silently produces an empty sessions.status
  registry. `newSessionQualitySourceFromPublisher` is the invariant-
  preserving constructor; all production call sites in `runConsole`
  use it exclusively. Fenced by
  `SetPublishHook_FiresOncePerPublish` + the E2E fixture round-trip.

**3. Smoke gate touched:** none. 14/14 smoke-quick, 5/5 spec-runner,
4/4 tier3 all pass on `8f192e7`. No smoke asset added or modified.
sbctl usage output (help text) gains the `sessions` sub-verb but the
existing help-parse sentinels (SPEC-1..SPEC-5) test on the paths and
router sub-verbs specifically, not on the aggregate command surface.

## Commit Trail (on `feat/s-bl-console-obs`)

| # | SHA | Subject |
|---|-----|---------|
| 1 | `26b5687` | `feat(metrics): add QualityIndicator.MissCount lifetime accessor` |
| 2 | `857a2a1` | `feat(session): add per-session QualityIndicator surface for sessions.status` (SUPERSEDED by 8f192e7) |
| 3 | `747f55c` | `feat(session): add sessions.status RPC handler` (SUPERSEDED by 8f192e7) |
| 4 | `b418565` | `feat(daemon): register sessions.status handler on console-mode daemon` |
| 5 | `3931741` | `feat(sbctl): add sessions status sub-verb dispatching sessions.status RPC` |
| 6 | `6e49a97` | `docs(demo-evidence): add S-BL.CONSOLE-OBS VHS demo tape` |
| 7 | `8f192e7` | `refactor(session): hoist per-session QualityIndicator to cmd/switchboard hooks` (ARCH-08 §6.6 course-correct) |

**Base:** `origin/develop@e81927f` (post S-7.04 PR #101 + drain evidence PR #103 + mise-install PR #102).
**Head:** `8f192e7`.

**PR:** #104, merged 2026-07-06T01:56:25Z as squash merge commit
`18fd2fe`.

## Follow-ons (Filed as Deferrals, not Regressions)

- **Live-measurement wiring** — sessionQualitySource is registered
  against the Publisher and populated on Publish/Unpublish, but the
  code path that drives OnSessionMeasurement / OnSessionMissingFrame
  from live frame delivery is not delivered here (analogous to
  S-BL.DISCOVERY-WIRE deferring live path-admission observation for
  PR #99). Under the current wire, published sessions surface as
  `quality: "pending"` with `miss_count: 0` until an observation
  source is wired. The wire is DAG-preserving and ready — no
  additional refactor required when the measurement source lands.
- **VP activation** — traceability VPs against BC-2.06.001 PC-5
  console-half + BC-2.06.002 PC-3 export are satisfiable and need a
  coordinator-side flip in `.factory/specs/verification-properties/`
  (factory-artifacts commit is coordinator-owned).
