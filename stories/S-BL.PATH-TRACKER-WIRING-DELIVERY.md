---
artifact_id: S-BL.PATH-TRACKER-WIRING-DELIVERY
document_type: story-delivery
level: ops
story_id: S-BL.PATH-TRACKER-WIRING
version: "1.0"
title: "wire pathTrackerSource to routing registry via typed forwarding-entry hook"
status: delivered
producer: implementer
timestamp: 2026-07-05T22:34:00Z
modified: 2026-07-05T22:34:00Z
phase: 2
wave: 7-backlog
priority: P2
scope_phase: E
estimated_points: 3
delivered_points: 3
bc_traces:
  - BC-2.06.003   # v1.15 PC-1 — paths.list PathTracker enumeration
vp_traces:
  - VP-062        # forwarding-entry registry authoritative source (activation flagged as coordinator follow-up)
subsystems: [transport-layer, observability]
architecture_modules:
  - internal/routing         # ForwardingEntryHook + WithForwardingEntryHook option added
  - cmd/switchboard          # pathTrackerSource + newPathTrackerSourceFromRouter
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-W5.04, S-BL.ROUTER-ADDR]
blocks: []
head_sha: 0b229e9
branch: feat/s-bl-path-observability
base: origin/develop@b75a2f2
worktree: .worktrees/path-observability-story
pr: 99
merged: 2026-07-05T22:35:52Z
merge_commit: c098827
drift_consumed:
  - id: S-W5.04 Ruling-6
    description: >
      "PathTracker enumeration deferred to S-BL.PATH-TRACKER-WIRING —
      routing.Router is DAG position 5 and paths is DAG position 8;
      routing MUST NOT import paths per ARCH-08 §6, so the wire cannot
      be a direct call from RegisterForwardingEntry into a paths method.
      Deferred pending a callback/hook plumbing pattern." Closed here:
      routing exposes a typed ForwardingEntryHook (func([16]byte, [8]byte))
      fired under the write lock; cmd/switchboard installs the hook and
      maintains the PathTracker registry on its side. No paths import
      from routing.
  - id: S-BL.PATH-TRACKER-WRITER (folded per Ruling-11)
    description: >
      "PathTracker registry write-safety (RWMutex + snapshot decoupling)
      deferred as a sibling of PATH-TRACKER-WIRING." Ruling-11 called for
      the two to fold since the writer construct IS the wiring construct.
      Folded here: pathTrackerSource.mu (sync.RWMutex) guards Register +
      AllSnapshots; readers use RLock, writers double-check under Lock,
      returned snapshots are fully decoupled per go.md rule 12 (value
      copies of paths.PathTracker.Snapshot()).
  - id: metrics_wire.go #DEFERRED marker
    description: >
      Removed. The comment block anchoring "TODO: wire pathTrackerSource
      to the routing subsystem when the boundary lands" is gone;
      newPathTrackerSourceFromRouter is the replacement construct.
adr_disposition:
  - id: ARCH-08 §6 DAG constraint
    scope: "routing (pos 5) may not import paths (pos 8)"
    verdict: preserved
    rationale: >
      The hook is a func-typed callback stored on routing.Router; routing
      does not import paths at any position. cmd/switchboard (which imports
      both) owns the callback body that constructs paths.PathTracker
      instances. The DAG is unbroken.
---

# S-BL.PATH-TRACKER-WIRING — Wire pathTrackerSource to routing registry (DELIVERY)

## What Landed

`cmd/switchboard/metrics_wire.go` now owns a live PathTracker registry
that observes every forwarding-entry admission in the routing subsystem
via a typed callback the router fires under its own write lock. Ruling-6
is closed without violating ARCH-08 §6.

Two composed constructs:

1. **`routing.ForwardingEntryHook`** (BC-2.06.003 PC-1 v1.15 wiring
   contract, DAG-preserving) — `func([16]byte, [8]byte)` field on
   `Router`. Installed via the constructor option
   `WithForwardingEntryHook(hook)` OR post-construction via
   `Router.SetForwardingEntryHook(hook)` (the setter is what lets
   `newPathTrackerSourceFromRouter` compose the router-then-source-then-
   hook wire without cyclic construction). Fired synchronously from
   inside `Router.RegisterForwardingEntry` under `r.mu.Lock` — the hook
   invocation and forwarding-table write are one atomic unit. Zero-value
   / nil hook is legal and is a no-op.

2. **`cmd/switchboard.pathTrackerSource`** — private struct with
   `sync.RWMutex + map[pathID]*paths.PathTracker`. Two constructors:
   - `newPathTrackerSource()` → empty registry (console/control modes
     that never own a router).
   - `newPathTrackerSourceFromRouter(r *routing.Router)` → constructs
     the empty registry, installs its `register` method as the
     forwarding-entry hook on r via `SetForwardingEntryHook`, and
     returns the source. Any subsequent
     `r.RegisterForwardingEntry(svtn, endpoint)` fires the hook, which
     acquires `pathTrackerSource.mu.Lock`, double-checks presence, and
     inserts a new `paths.PathTracker` keyed by `pathID` derived from
     the (svtn, endpoint) tuple.
   - `AllSnapshots()` acquires `mu.RLock` and returns
     `[]paths.PathSnapshot` — value copies via `PathTracker.Snapshot()`,
     fully decoupled per go.md rule 12 (no leaked pointers into locked
     state).

The `metrics_wire.go` `#DEFERRED S-BL.PATH-TRACKER-WIRING` anchor
comment is removed; the replacement wiring is documented in place.

## Scope Delivered vs Deferred

**Delivered:**

- `internal/routing/routing.go` — `ForwardingEntryHook` type,
  `WithForwardingEntryHook` functional option, `SetForwardingEntryHook`
  setter, hook-firing inside `RegisterForwardingEntry` under the write
  lock, nil-hook no-op guard.
- `cmd/switchboard/metrics_wire.go` — `pathTrackerSource` struct,
  `newPathTrackerSource()`, `newPathTrackerSourceFromRouter(r)`,
  private `register(svtn, endpoint)` hook body, `AllSnapshots()` reader.
- `cmd/switchboard/access.go` + `mgmt_wire.go` — call-site swap from
  the empty-source constructor to `newPathTrackerSourceFromRouter(r)`
  in the code paths that own a router; console/control keep the empty
  source.
- `cmd/switchboard/pathtracker_source_test.go` (293 lines, 5 tests) —
  first-sight construction; idempotent identity across re-Register
  (RTT witness proves the same tracker instance is reused, not
  overwritten); AllSnapshots returns value copies (mutation isolation);
  4-writer × 4-reader × 128-writes concurrent race-clean;
  `newPathTrackerSourceFromRouter` installs the hook end-to-end.
- `internal/routing/routing_pathtrackers_test.go` (186 lines) — hook
  fires exactly once per RegisterForwardingEntry; hook receives the
  right (svtn, endpoint) tuple; nil hook is a no-op; concurrent
  Register + hook-invocation race-clean under -race.

**Deferred (not this story):**

- **VP-062 activation** — the traceability verification property that
  gates on the wiring existing at all is now satisfiable but requires
  a coordinator (state-manager) flip in `.factory/specs/verification-
  properties/`. Flagged in PR #99 body; not touched here because
  factory-artifacts commits are coordinator-owned.
- **Live-daemon end-to-end** — pathTrackerSource observes the routing
  registry, but a live inbound frame currently ingresses through
  netingress → routing.RouteFrame (which routes an already-admitted
  path); admissions today come from admin RPCs and discovery. When
  discovery wiring lands (S-BL.DISCOVERY-WIRE) the observed
  activations will flow into pathTrackerSource without additional
  code — the hook already fires on every RegisterForwardingEntry.

## Findings Consumed

| Finding | Description | Where closed |
|---------|-------------|--------------|
| S-W5.04 Ruling-6 | PathTracker enumeration deferred pending DAG-preserving wire pattern | `internal/routing/routing.go` ForwardingEntryHook + `cmd/switchboard/metrics_wire.go` newPathTrackerSourceFromRouter |
| S-BL.PATH-TRACKER-WRITER (folded per Ruling-11) | RWMutex + snapshot-decouple for concurrent register/read | `cmd/switchboard/metrics_wire.go` pathTrackerSource.mu + AllSnapshots value-copy pattern + `cmd/switchboard/pathtracker_source_test.go::TestConcurrentRegisterAndAllSnapshots_RaceClean` |
| metrics_wire.go `#DEFERRED` marker | Anchor comment for the wire | Marker removed; new construct documented in place |

## Writer-Deferral Adjudication (Ruling-11)

**Verdict: folded into this story. Ruling-11 upheld.**

`wave-6-tranche-a-scope-rulings.md` Ruling-11 called for
S-BL.PATH-TRACKER-WRITER to fold into S-BL.PATH-TRACKER-WIRING because
the two touch the same construct: the writer is the wire. Attempting
to ship them separately would produce either (a) a partial wire that
publishes `AllSnapshots` before the mutex exists (racy), or (b) a
mutex on a construct that has no writers yet (dead code). The delivered
implementation puts them together: `pathTrackerSource.mu`,
`AllSnapshots` decoupling, and the concurrent-writer race test all
land in the same commit as the wire itself.

The 4-writer × 4-reader × 128-writes race test
(`TestConcurrentRegisterAndAllSnapshots_RaceClean` in
`pathtracker_source_test.go`) is the load-bearing evidence: run under
`-race -count=3` in every check below.

## Liveness Design + Spec Citation

The wire delivers the *enumeration* wire (Ruling-6 concern), not a
new liveness signal. Liveness (Active / Degraded / Failed) is the
sibling story S-BL.PATH-FAILED-STATUS. This story wires the tracker
into place so the sibling has somewhere to write; the sibling defines
what "alive," "degraded," and "failed" mean and how they flow through
`PathSnapshot` → `PathEntry.Status`.

**Spec citations for the wire:**

- BC-2.06.003 v1.15 PC-1 — paths.list must enumerate all registered
  forwarding paths. The wire is the enumeration channel.
- ARCH-08 §6 — routing (pos 5) MUST NOT import paths (pos 8). The
  hook-callback pattern keeps the DAG unbroken.
- S-W5.04 Ruling-6 — the deferral this story lifts.
- Ruling-11 — the writer story folds into the wire story.

## Sentinel / SPEC Impact

**sbctl round-trip surface:** `sbctl paths list` and `sbctl router
status` both dispatch the `paths.list` RPC (cmd/sbctl/paths_list.go
and cmd/sbctl/router_status.go). Wire schema unchanged by this story
— `PathEntry.Status` field already existed with docstring "active |
degraded | failed" per prior BC-2.06.003 v1.15 (the sibling story
lifted the reservation; this story does not touch the wire schema).
Runtime effect: paths that get RegisterForwardingEntry-admitted now
surface with `status:"active"` (via `paths.PathTracker.Snapshot()`
defaults) where previously they surfaced as an empty array. That's a
strictly-additive change to the operator surface; SPEC-1..SPEC-5
sentinels (which cover the unreachable-daemon and usage-error paths,
not the happy-path enumeration) are unaffected.

**Smoke suite touch:** none. INV-1..INV-10, INV-7:{access,router,
console,control}, INV-8:{switchboard,sbctl} — all 14/14 pass on
worktree HEAD. SPEC-1..SPEC-5 all 5/5. Tier-3 T3-2/T3-4 all 4/4.
No smoke asset modified.

## Test Inventory

**New tests:**

- `cmd/switchboard/pathtracker_source_test.go` — 5 tests, 293 lines
  1. `TestNewPathTrackerSource_EmptyOnConstruction` — empty registry
     baseline.
  2. `TestRegister_IdempotentIdentityAcrossReRegister` — RTT witness
     proves the same PathTracker instance is reused, not overwritten.
  3. `TestAllSnapshots_ReturnsValueCopies` — mutation of the returned
     slice does not affect internal state (go.md rule 12).
  4. `TestConcurrentRegisterAndAllSnapshots_RaceClean` — 4 writers × 4
     readers × 128 writes each under `-race`.
  5. `TestNewPathTrackerSourceFromRouter_InstallsHookEndToEnd` —
     construct + Register in the router + AllSnapshots on the source
     surfaces the registered path.
- `internal/routing/routing_pathtrackers_test.go` — 4 tests, 186 lines
  1. `TestForwardingEntryHook_FiresOnceOnRegister` — exactly one hook
     call per RegisterForwardingEntry.
  2. `TestForwardingEntryHook_ReceivesCorrectTuple` — SVTN + endpoint
     round-trip through the hook signature.
  3. `TestForwardingEntryHook_NilHook_IsNoOp` — zero-value Router safe.
  4. `TestForwardingEntryHook_ConcurrentRegisterAndHook_RaceClean` —
     hook fires under `r.mu.Lock`, no torn writes.

**Runs (all clean):**

- `just fmt` — no changes.
- `go vet ./cmd/switchboard/... ./internal/routing/...` — no output.
- `go test -race -count=3 ./internal/routing/... ./cmd/switchboard/...` — green (routing 2.691s, cmd/switchboard 3.444s).
- `go test ./... -count=1` — all 21 packages green.
- `golangci-lint run ./...` — 0 issues.
- `just smoke-quick` — 14/14 (`.smoke/20260705T223612Z/report.jsonl`).
- `bash test/smoke/spec-runner.sh` — 5/5 (`.smoke/20260705T223618Z-spec/report.jsonl`).
- `bash test/smoke/tier3-tutorial.sh` — 4/4 (`.smoke/20260705T223619Z-tier3/report.jsonl`).
- CI on PR #99 — all checks green (Quality Gate, Analyze go / CodeQL,
  Blast Radius declaration, dependency-review, StepSecurity
  Harden-Runner). Alpha-release jobs SKIPPED per gitflow (fires on
  develop push).

## Blast Radius

**1. Operator-visible surfaces touched:** `sbctl paths list` and
`sbctl router status` output when a router-owning daemon is running.
Before this wire: `[]` (no paths ever enumerated because the source
was empty). After this wire: entries appear with `status:"active"`
(via `paths.PathTracker.Snapshot()` defaults on a freshly-registered
tracker) as forwarding entries land. Additive-only surface change:
existing sentinels (SPEC-1..SPEC-5 which cover unreachable-daemon +
usage-error paths, not happy-path enumeration) are unaffected;
`PathEntry.Status` enum already includes "active" (the value produced).
No new CLI flag, no new config field, no new error taxonomy code.

**2. Silent-failure risk:** low, but named. The hook fires under
`r.mu.Lock`; a slow hook body would slow forwarding-entry admission.
The delivered `register` body acquires a private mutex and does map
insertion only — deterministic, O(1), no I/O. Any future hook body
that does more (logging, metrics counters, network I/O) must audit
for hold-time regressions. The 4-writer race test proves the current
body is bounded. Second risk: nil-hook is a no-op; if a consumer
forgets to install the hook after construction, the source silently
stays empty. This is caught by
`TestNewPathTrackerSourceFromRouter_InstallsHookEndToEnd` — the
production call sites (`cmd/switchboard/access.go`,
`cmd/switchboard/mgmt_wire.go`) use the *FromRouter* constructor
exclusively.

**3. Smoke gate touched:** none. 14/14 sentinels + 5/5 spec-runner +
4/4 tier3 all pass on worktree HEAD. No smoke asset added or
modified. sbctl output schema unchanged (status enum already ships
`active|degraded|failed` per the sibling story's spec lift; the wire
only ensures entries populate).

## Follow-ons (Filed as Deferrals, not Regressions)

- **VP-062 activation** — traceability verification property gate
  now satisfiable; coordinator flip in
  `.factory/specs/verification-properties/`. Flagged in PR #99 body.
- **S-BL.DISCOVERY-WIRE** — when discovery-driven admissions arrive
  via RegisterForwardingEntry, they'll surface in AllSnapshots
  without further code (the hook already fires for every admission
  path).

## Commit Trail (on `feat/s-bl-path-observability`)

1. `0b229e9` — `feat(routing,cmd): S-BL.PATH-TRACKER-WIRING — hook + pathTrackerSource wire`

Base: `origin/develop@b75a2f2` (post S-BL.ARQ-TX PR #98 merge).
Head: `0b229e9`.

**PR:** #99, merged 2026-07-05T22:35:52Z as merge commit `c098827`.
