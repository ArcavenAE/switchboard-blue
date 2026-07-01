---
artifact_id: wave-6-tranche-a-scope-rulings
document_type: decision
level: ops
version: "1.3"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-W5.04, S-6.07]
closes_findings: [F-P1L1-003, F-P1L1-004, F-P1L1-005, F-P1L1-003-stutter, F-P3L1-002, F-L2-01, F-Impl-002, F-P4L1-001, F-P4L1-002, O-P4L3-01, F-P4L2-07]
---

# Wave-6 Tranche A Scope Rulings

Product-owner rulings on two structural findings surfaced during BC-5.39.001
adversarial convergence of Wave-6 Tranche A (S-W5.04, S-6.07). Neither ruling
modifies story specs, BC files, or STORY-INDEX — those changes belong to the
fix-burst that follows.

---

## Ruling 1 — S-W5.04 F-P1L1-003: router_addr Wire-Shape Completion

**Finding summary:** `PathsList` in `.worktrees/S-W5.04/internal/metrics/handlers.go`
(lines 51–56) calls `PathEntryFromSnapshot(pathID, pathID, snap)`, passing `pathID`
as both the path identifier and the `router_addr` argument. BC-2.06.003 PC-1 mandates
`router_addr` is the peer's `host:port` — a distinct network coordinate from the
path handle. The spoofed shape also invalidates VP-047 AC-006's assertion on
`router_addr` field presence (F-P1L1-004). Root cause: `PathSnapshot` in
`internal/paths` (S-5.02 scope) has no `RouterAddr` field. Tracked in
`DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER`.

**Options considered:**

- **(A) Ship empty string** — emit `router_addr: ""` with a doc comment; follow-on
  story enriches `PathSnapshot` with a `RouterAddr` field before wave-convergence.
  VP-047 and AC-006 assertions must be narrowed to permit the empty-string interim
  state.

- **(B) Block S-W5.04** — same-tranche `PathSnapshot` enrichment story widens scope
  to S-5.02-owned code in `internal/paths`, which is already merged and has a clean
  convergence record.

- **(C) Derive from mgmt routing table** — a lookup helper crosses from
  `internal/metrics` (pure-core per ARCH-09 §3.1) into an effectful package.
  ARCH-09 §3.1 classifies `internal/metrics` as pure-core: business logic only,
  no network I/O, no imports of effectful packages. Any import chain reaching a
  routing table in `internal/routing` or `internal/mgmt` would introduce a
  forbidden edge per ARCH-08 §6.2.

**Ruling: Option (A).**

S-W5.04 SHALL ship `router_addr: ""` (empty string) rather than the spoofed
`pathID`. A doc comment on `PathEntryFromSnapshot` MUST note that `RouterAddr` is
intentionally empty in this wave pending `PathSnapshot` enrichment. Option (B) is
rejected: enriching `PathSnapshot` in a Wave-6 tranche story touches
`internal/paths` (S-5.02 scope, already merged, convergence-clean); reopening that
package creates scope bleed and risks destabilizing three clean adversarial passes.
Option (C) is rejected outright: it is architecturally forbidden per ARCH-09 §3.1
(pure-core packages must not import effectful packages).

The follow-on `PathSnapshot` enrichment story (stub: `S-BL.ROUTER-ADDR`) MUST land
before the Wave-6 wave-convergence adversarial pass. It owns adding `RouterAddr
string` to `internal/paths.PathSnapshot` and propagating it through
`PathEntryFromSnapshot`. DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER remains open until
that story merges.

### Story-Spec Impacts (S-W5.04 fix-burst)

These are the exact AC and VP text changes required in the S-W5.04 fix-burst:

**AC-006 (S-W5.04)** — change the field-presence assertion for `router_addr`:

> _Current:_ "...required fields present and non-null: `path_id`, `router_addr`, `rtt_ms`, `rtt_p99_ms` (float64 or `"pending"`), `loss_pct`, `status`."
>
> _Replace with:_ "...required fields present and non-null: `path_id`, `rtt_ms`, `rtt_p99_ms` (float64 or `"pending"`), `loss_pct`, `status`. `router_addr` MUST be present in the JSON output; its value is `""` (empty string) in this interim state pending `PathSnapshot.RouterAddr` enrichment (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER). The AC-006 integration test MUST assert `router_addr` key presence and accept `""` as a valid value."

**VP-047 Property Statement** — add an interim-state note to the `pathEntry` struct:

> Add a comment to the `pathEntry` struct in the proof harness skeleton noting that
> `RouterAddr` is present but may be `""` in the current wave: `RouterAddr *string
> \`json:"router_addr"\`` along with a harness assertion that checks key presence but
> does not reject the empty-string value. The Property Statement prose MAY add:
> "Note: `router_addr` key presence is asserted; the value may be `""` until
> `PathSnapshot.RouterAddr` is populated (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER)."

**BC-2.06.003 v1.9** — add a spec note to PC-1 `router_addr` field definition:

> After the existing definition `router_addr — remote router address (host:port)`,
> add: "**Interim state (Wave 6):** `router_addr` emits as `""` (empty string) until
> `internal/paths.PathSnapshot` is enriched with a `RouterAddr` field
> (DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER). Conformance tests MUST assert field
> presence and permit the empty-string value during this wave."

### Follow-on Stories

One new story stub must be added to STORY-INDEX.md in the fix-burst:

| Story ID | Title | Depends on | Wave target | Owner |
|----------|-------|-----------|-------------|-------|
| S-BL.ROUTER-ADDR | Enrich `PathSnapshot` with `RouterAddr` field and propagate to `PathEntry.router_addr` | S-W5.04 (merged) | Wave-6 wave-convergence gate | implementer + story-writer |

Story S-BL.ROUTER-ADDR scope: add `RouterAddr string` to `internal/paths.PathSnapshot`
(S-5.02-owned type); populate it at path-creation time from the router's listen
address; update `PathEntryFromSnapshot` to pass `snap.RouterAddr`; remove the
DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER drift entry; remove the interim-state note
from BC-2.06.003 PC-1 and VP-047; update AC-006 to assert a non-empty
`router_addr`. MUST merge before the Wave-6 wave-adversarial pass.

---

## Ruling 2 — S-6.07 F-P1L1-005: admin.svtn.create Authority Model

**Finding summary:** `makeAdminSVTNCreateHandler` in
`.worktrees/S-6.07/cmd/switchboard/admin_handlers.go` accepts only the daemon
bootstrap key as an authorized caller. BC-2.07.001 Inv-3 ("Only control-role keys
may create or destroy SVTNs") is ambiguous about whether a human operator holding
a control-role key in any existing SVTN may create a NEW SVTN, or whether only
the bootstrap key path may do so.

**Options considered:**

- **(A) Bootstrap-only** (current impl): only the daemon bootstrap key may initiate
  SVTN creation. Every SVTN's genesis is bootstrap-initiated. There is exactly one
  path to SVTN creation; no delegation surface exists.

- **(B) Any control-role**: any caller holding a control-role key in ANY registered
  SVTN may create a new SVTN. Delegates lifecycle authority to established operators
  across SVTN boundaries.

- **(C) Configurable**: bootstrap-only default with an opt-in daemon policy flag
  enabling control-role delegation.

**Ruling: Option (A).**

The daemon bootstrap key is the sole authorized caller for `admin.svtn.create`.
Option (B) is rejected: extending create authority to "any control-role key in
any existing SVTN" crosses SVTN isolation boundaries — a control operator for
SVTN-A could create SVTN-B, which is a privilege escalation relative to SVTN-A
scope. BC-2.07.001 Inv-1 ("control node manages SVTN lifecycle as a participant in
the user/data plane; it does not have privileged access to router internals") speaks
to scope confinement, not cross-SVTN authority grants. Option (C) is rejected: it
introduces configuration surface that the MVP does not need and that would require
additional BC coverage, error taxonomy entries, and test vectors before it could
converge. The bootstrap-only model is the simplest security posture, is already
implemented, and is consistent with BC-2.07.001 PC-2's "first control key
bootstrapped locally" design.

**Inv-3 clarification:** BC-2.07.001 Inv-3's phrase "control-role keys" refers
exclusively to the bootstrap path — the key that performs `admin.svtn.create` is the
daemon bootstrap key, which is established as the first control-role key
(BC-2.07.001 PC-1 + PC-2). After bootstrap, further SVTN lifecycle operations
(`admin.svtn.destroy`, managed by S-6.05) require a key that is registered as
control-role in that specific SVTN. The create operation has no such prior context
by definition — the SVTN does not yet exist — so bootstrap-only is the only coherent
interpretation of "control-role" for the create operation.

The BC-2.07.001 Inv-3 prose MUST be tightened in the fix-burst to eliminate the
ambiguity that generated this finding.

### Sub-ruling 2a — F-P1L1-003: E-ADM-004 vs. "SVTN already exists" error taxonomy

**Finding:** the current impl uses `E-ADM-004` for the "SVTN already exists"
duplicate error. `E-ADM-004` is already occupied in error-taxonomy.md as "address
collision: node address already admitted on SVTN" (BC-2.01.006). The canonical code
for duplicate SVTN creation is `E-SVTN-001` ("SVTN already exists: `<id>`"),
defined in error-taxonomy.md §SVTN. The impl must use `E-SVTN-001`, surfaced as
`ok: false, error.code: E-SVTN-001, error.message: "SVTN already exists: <name>"`
per AC-005 of S-6.07 (which already specifies `E-SVTN-001` correctly in the story).
The impl is out of sync with the story spec; the fix-burst must align the impl to
the story. No error-taxonomy changes are needed — `E-SVTN-001` is already defined.

### Sub-ruling 2b — F-P1L1-004: "SVTN already exists: name: SVTN already exists" stutter

**Finding:** the current impl produces a double-stutter message "SVTN already
exists: name: SVTN already exists" because it wraps `SVTNManager.Create()`'s error
string (which includes "SVTN already exists") into a message that again says "SVTN
already exists:". The canonical format is `E-SVTN-001 "SVTN already exists: <name>"`
where `<name>` is the SVTN name string only. The fix-burst must unwrap the inner
error and emit only the name token in the message, not re-wrap the error string.
This is a fix regardless of the authority-model ruling.

### Story-Spec Impacts (S-6.07 fix-burst)

**BC-2.07.001 Inv-3** — tighten the scope prose to eliminate authority-model
ambiguity:

> _Current:_ "Only control-role keys may create or destroy SVTNs. **Scope:** this
> invariant governs `admin.svtn.*` operations only."
>
> _Replace with:_ "Only the daemon bootstrap key may invoke `admin.svtn.create`
> (the SVTN does not yet exist at create time; there is no prior admitted-key set
> to authorize against). For `admin.svtn.destroy`, a key registered as
> `RoleControl` in the target SVTN is required (S-6.05 scope). Cross-SVTN create
> authority is not granted to control-role keys of existing SVTNs. **Scope:**
> this invariant governs `admin.svtn.*` operations only; `admin.key.*` authority
> is governed by BC-2.05.004 PC-1 / DI-001."

**S-6.07 AC-003** — align to the tightened invariant text:

> After "if the authenticated caller's key role is not `RoleControl`", add: "or
> if the caller is not the daemon bootstrap key, the handler returns `E-ADM-009`.
> Bootstrap-only authority is the sole path to SVTN creation."

**S-6.07 impl (`cmd/switchboard/admin_handlers.go`)** — two impl fixes required:

1. Replace the `E-ADM-004` error code with `E-SVTN-001` for the duplicate-name
   path (Sub-ruling 2a). Confirm the wire response is
   `ok: false, error.code: "E-SVTN-001", error.message: "SVTN already exists: <name>"`.

2. Fix the stutter: when `SVTNManager.Create()` returns an "already exists" sentinel
   error, extract the SVTN name from the original call arguments (not from the error
   message string) and format as `fmt.Sprintf("SVTN already exists: %s", name)`.
   Do not wrap `err.Error()` into the message (Sub-ruling 2b).

### Follow-on Stories

No new stories are required for this ruling. The S-BL.ROUTER-ADDR stub (from
Ruling 1) is the only new story generated by this decision document.

---

## Summary of Spec Changes

| File | Change Summary | Finding(s) Closed |
|------|----------------|-------------------|
| BC-2.06.003 | Add interim-state note to PC-1 `router_addr` definition permitting `""` pending PathSnapshot enrichment | F-P1L1-003, F-P1L1-004 |
| VP-047 | Add `router_addr` key-presence assertion (empty-string permitted); note interim state | F-P1L1-004 |
| S-W5.04 AC-006 | Narrow `router_addr` assertion: key must be present; empty-string permitted in this wave | F-P1L1-003, F-P1L1-004 |
| BC-2.07.001 Inv-3 | Tighten prose: bootstrap-only for create; control-role for destroy; no cross-SVTN authority | F-P1L1-005 |
| S-6.07 AC-003 | Align to tightened Inv-3: add "or if the caller is not the daemon bootstrap key" | F-P1L1-005 |
| S-6.07 impl | Fix E-ADM-004 → E-SVTN-001; fix stutter message format | F-P1L1-003 (admin), F-P1L1-004 (stutter) |
| STORY-INDEX.md | Add S-BL.ROUTER-ADDR stub (backlog; Wave-6 gate dependency) | — |

---

## Ruling 3 — S-W5.04 F-P2L1-003: Real PathTracker Wiring in Production

**Finding summary (Lens-1 BLOCK):** `cmd/switchboard/metrics_wire.go` ships
`emptyPathsSource` and `emptyRouterMetricsSource` as production wiring stubs.
AC-006's integration test injects `synthPathsListSource` directly, bypassing the
production code path — a classic vacuous-pass anti-pattern. The real production
daemon therefore returns empty path lists and empty router metrics despite having
live `PathTracker` state from S-5.02. The story's stated intent — "real (non-stub)
daemon" per AC-006 — is not met.

**Options considered:**

- **(A) In-scope fix — wire real PathTracker adapter in production paths:** S-W5.04
  adds a `PathsListSource` adapter over `PathTracker.Snapshot()` and registers it
  in both `runControl` and `runAccess`. The `emptyPathsSource` and
  `emptyRouterMetricsSource` stubs are deleted. AC-006 integration test is updated
  to spin the daemon with a real `PathTracker` rather than an injected synth.
  Estimated scope delta: ~30 LOC adapter + registration; ~1–2 days.

- **(B) Defer with drift entry:** ship the story as-is with the empty stubs;
  mint follow-on story `S-BL.PATH-TRACKER-MAP` (Wave-6 Tranche B or Wave-7) for
  real wiring; document `DRIFT-SW504-PATHTRACKER-EMPTY-STUB` in AC-006 acceptance
  text.

**Ruling: Option (A).**

AC-006 explicitly states "real (non-stub) daemon." Shipping empty stubs makes the
story deliver against its own spec. Option (B) is rejected: the
`PathTracker.Snapshot()` surface is already available from S-5.02 (merged,
convergence-clean); the wiring is a small, well-scoped adapter. Deferring
accumulates a known inaccuracy into the wave-convergence adversarial pass, which
is a compounding risk. The empty stubs must be deleted — not just bypassed in tests
— so that any future test that reaches the production wire path gets real data.

### Story-Spec Impacts (S-W5.04 fix-burst)

**S-W5.04 v1.5→v1.6 changelog** — add task to implementation list:

> "Wire real `PathTracker` → `PathsListSource` adapter in `runControl` and
> `runAccess` entry points in `cmd/switchboard/metrics_wire.go`. Delete
> `emptyPathsSource` and `emptyRouterMetricsSource` stubs. The adapter calls
> `PathTracker.Snapshot()` and converts the result to the `PathsListSource`
> interface. Register the adapter identically in both daemon mode paths."

**S-W5.04 AC-006** — expand the integration test fixture requirement:

> _Current:_ "AC-006 integration test injects `synthPathsListSource` directly."
>
> _Replace with:_ "AC-006 integration test spins a daemon with a real
> `PathTracker` instance populated with at least one synthetic path entry.
> The test asserts that `GET /paths` returns the path entry from the live
> `PathTracker` state. Injecting `synthPathsListSource` directly is no longer
> permitted as an AC-006 fixture."

### Follow-on Stories

No new stories are required by Ruling 3. `S-BL.PATH-TRACKER-MAP` (Option B) is
explicitly rejected; the work is absorbed into the S-W5.04 fix-burst.

---

## Ruling 4 — S-W5.04 F-P2L3-006: `status="failed"` Derivation Source

**Finding summary (Lens-3 MED):** BC-2.06.003 v1.9 PC-1 defines the status enum
as `{active, degraded, failed}`. However, `PathSnapshot` (S-5.02 scope) has no
`Failed` field or liveness signal. AC-005a states "liveness-down → 'failed'" without
naming the signal source. The story is spec-implementation incomplete for the
"failed" status value.

**Options considered:**

- **(A) Add `PathSnapshot.Failed bool`** in `internal/paths` — a cross-boundary
  change into S-5.02's already-merged, convergence-clean package.

- **(B) Derive "failed" from staleness heuristic** — infer from
  `PathSnapshot.LastRTT == 0 && SampleCount > 0 && time.Since(LastSampleAt) > threshold`;
  requires a new threshold config parameter.

- **(C) Defer "failed" to a follow-on story:** S-W5.04 v1.6 restricts the status
  enum to `{active, degraded}` for this story. BC-2.06.003 v1.10 documents that
  "failed" is reserved for a future liveness-signal story. Mint stub story
  `S-BL.PATH-FAILED-STATUS` in STORY-INDEX (Wave-7 Backlog).

**Ruling: Option (C).**

"failed" is not required by any Wave-6 acceptance criterion. AC-005a and EC-006
already handle the `degraded + pending` precedence case correctly. "failed" was
speculative scope in BC-2.06.003 that was never backed by a liveness signal in
`internal/paths`. Option (A) is rejected: reopening S-5.02's type surface in a
Wave-6 tranche story creates scope bleed into a convergence-clean package.
Option (B) is rejected: a staleness heuristic introduces a threshold config
parameter that would itself require BC coverage, error taxonomy entries, and
additional test vectors — disproportionate for a speculative status value. Deferring
is the correct product decision at this stage.

### Story-Spec Impacts (S-W5.04 fix-burst)

**BC-2.06.003 v1.9→v1.10** — retract "failed" from the status enum for this wave:

> _PC-1 status field current:_ "`status` — path health classification:
> `active | degraded | failed`."
>
> _Replace with:_ "`status` — path health classification: `active | degraded`.
> The value `failed` is reserved for a future liveness-signal story
> (`S-BL.PATH-FAILED-STATUS`). Implementations MUST NOT emit `failed` until that
> story lands. Conformance tests MUST reject `failed` in the status field during
> Wave 6."

**S-W5.04 v1.5→v1.6 changelog** — update AC-005a and EC-006 text to remove
"failed" from expected enum values and replace with "reserved (not emitted in
this wave)."

### Follow-on Stories

One new story stub must be added to STORY-INDEX.md in the fix-burst:

| Story ID | Title | Depends on | Wave target | Owner |
|----------|-------|-----------|-------------|-------|
| S-BL.PATH-FAILED-STATUS | Add liveness signal to `PathSnapshot` and derive `status="failed"` in `PathEntry` | S-W5.04 (merged) | Wave-7 Backlog | implementer + story-writer |

Story `S-BL.PATH-FAILED-STATUS` scope: add a liveness field or signal to
`internal/paths.PathSnapshot` (decision: `Failed bool` vs. staleness threshold
TBD in that story's spec); update the `PathsListSource` adapter to derive
`status: "failed"` from the signal; update BC-2.06.003 to re-admit "failed" to
the enum; update AC-005a / EC-006; add test coverage. MUST NOT merge before
`S-BL.ROUTER-ADDR`.

---

## Ruling 5 — S-6.07 F-P2L1-001: Bootstrap-Only Fast-Path Fix

**Finding summary (Lens-1 BLOCK / HIGH):** `admin.svtn.create` handler calls
`resolveAndVerifyCallerRole(ctx, m, ops, a.Name, "", "admin.svtn.create")`.
When `args.name` matches an existing SVTN in which the caller holds an active
`RoleControl` key, this call fast-paths through `CallerKeyRoleActive`, skips the
bootstrap-only check, and grants create authority to a non-bootstrap caller.
BC-2.07.001 Inv-3 (as tightened by Ruling 2) mandates bootstrap-only authority
for `admin.svtn.create`. The fast-path means a non-bootstrap control-role key
triggers `E-SVTN-001` (existence oracle) instead of `E-ADM-009` (unauthorized),
leaking SVTN existence information to callers who should receive only an auth
rejection. This violates Inv-3 and is a HIGH-severity security gap.

**Options considered:**

- **(A) Per-cmd bypass in handler (recommended):** in `makeAdminSVTNCreateHandler`,
  before delegating to `resolveAndVerifyCallerRole`, add an explicit
  `IsBootstrapKey(callerPub)` check. If false, return `E-ADM-009` immediately
  — do not call `resolveAndVerifyCallerRole` at all. This is a narrow, targeted
  fix that does not modify the shared helper.

- **(B) Cmd-scoped bypass list in `resolveAndVerifyCallerRole`:** add a
  `skipRoleActiveLookup []string` parameter to the shared helper; pass
  `"admin.svtn.create"` in the list.

**Ruling: Option (A).**

A per-handler guard in `makeAdminSVTNCreateHandler` is narrower, does not change
the shared helper's contract, and is less likely to introduce regressions in the
other call sites of `resolveAndVerifyCallerRole`. Option (B) is rejected: adding a
bypass list to a shared auth helper widens its interface and must be validated
across every call site — disproportionate for a single-command fix. The guard must
be placed BEFORE the `resolveAndVerifyCallerRole` call so that existence-oracle
leakage is impossible regardless of SVTN state.

No BC change is needed. BC-2.07.001 v1.4 Inv-3 text already mandates
bootstrap-only for `admin.svtn.create` (as tightened by Ruling 2 of this
document). This ruling is a pure implementation correction.

### Story-Spec Impacts (S-6.07 fix-burst)

**S-6.07 v1.2→v1.3 changelog** — add implementation note:

> "Fix fast-path auth bypass in `makeAdminSVTNCreateHandler`:
> add `IsBootstrapKey(callerPub)` guard before `resolveAndVerifyCallerRole`.
> Non-bootstrap callers MUST receive `E-ADM-009` before any SVTN state lookup
> occurs. The fast-path through `CallerKeyRoleActive` MUST NOT be reachable
> for `admin.svtn.create`."

**S-6.07 AC-006(b)** — add adversarial test case:

> "AC-006(b) [new]: caller holds a valid `RoleControl` key in an existing SVTN
> whose name matches `args.name` in the create request. Expected: handler returns
> `ok: false, error.code: E-ADM-009` before `SVTNManager.Create()` is called.
> The SVTN state MUST NOT be consulted. This test MUST be added in the fix-burst."

### Follow-on Stories

No new stories are required for this ruling. The fix is entirely within the
S-6.07 fix-burst scope.

---

## Summary of Spec Changes — Rulings 3–5

| File | Change Summary | Finding(s) Closed |
|------|----------------|-------------------|
| S-W5.04 v1.5→v1.6 | Add task: wire real `PathTracker` → `PathsListSource` adapter; delete empty stubs | F-P2L1-003 |
| S-W5.04 AC-006 | Require live `PathTracker` fixture; prohibit `synthPathsListSource` injection | F-P2L1-003 |
| BC-2.06.003 v1.9→v1.10 | Retract `failed` from status enum; document as reserved for `S-BL.PATH-FAILED-STATUS` | F-P2L3-006 |
| S-W5.04 AC-005a / EC-006 | Remove `failed` from expected enum values; note reserved status | F-P2L3-006 |
| S-6.07 v1.2→v1.3 | Add bootstrap-only guard in `makeAdminSVTNCreateHandler` before `resolveAndVerifyCallerRole` | F-P2L1-001 |
| S-6.07 AC-006(b) | Add adversarial test: `RoleControl` key for existing SVTN must get `E-ADM-009`, not existence oracle | F-P2L1-001 |
| STORY-INDEX.md | Add `S-BL.PATH-FAILED-STATUS` stub (Wave-7 Backlog; depends on S-W5.04) | — |

---

## Ruling 6 — S-W5.04 F-P3L1-002 / F-L2-01: Real PathTracker Wiring — Wave-6 vs. Wave-7 Backlog

**Finding summary (Pass-3 Lens-1 BLOCK + Lens-2 BLOCK):** Pass-2 Ruling-3 (this
document) required wiring a real `PathTracker → PathsListSource` adapter in
`cmd/switchboard/metrics_wire.go` and deleting `emptyPathsSource` /
`emptyRouterMetricsSource`. The implementer's Pass-2 fix (50c1825) renamed
`emptyRouterMetricsSource` → `pathTrackerSource` but left the tracker map
**permanently empty in production**: the map is populated only in tests (via
`pathTrackerSource.register`). Pass-3 Lens-1 (F-P3L1-002) and Lens-2 (F-L2-01)
both ruled BLOCK on this basis. The production daemon still returns an empty
`PathsListResponse` for any live running daemon — the cosmetic rename did not
satisfy the intent of Ruling-3's "real (non-stub)" requirement.

**Options considered:**

- **(A) In-scope (require real wiring in S-W5.04):** the routing subsystem must
  expose a registry of `(SVTN, endpoint) → PathTracker` instances that
  `metrics_wire.go` can enumerate at handler-serve time. `pathTrackerSource`
  (and any lingering empty-stub variant) is populated from that registry in the
  `runControl` / `runAccess` daemon entry points. AC-006 integration test is
  updated to spin a daemon with a live `PathTracker` and assert non-empty
  `PathsListResponse`. This may require changes to the routing package (S-5.02
  scope, already merged) to expose a tracker-enumeration surface.

- **(B) Defer to Wave-7 backlog:** accept that Wave-6 S-W5.04 delivers the
  metrics handler surface, response types, and a test-only populated map.
  Mint `S-BL.PATH-TRACKER-WIRING` in STORY-INDEX Backlog. Revise S-W5.04
  AC-006 to explicitly say "handler surface only; production tracker population
  deferred to S-BL.PATH-TRACKER-WIRING". Add a `// #DEFERRED: see
  S-BL.PATH-TRACKER-WIRING` comment on the `pathTrackerSource` field.

**Ruling: Option (B) — defer.**

Ruling-3 (this document) intended real wiring, and the implementer's rename was
insufficient. However, the adversarial context now clarifies that "real wiring"
requires touching the routing subsystem to expose a `PathTracker` registry — a
cross-package, cross-story-boundary change into S-5.02's already-merged and
convergence-clean package. Reopening S-5.02 in a Wave-6 fix-burst risks
destabilizing three clean adversarial passes of routing code. The metrics handler
surface (types, JSON shape, handler wiring, per-test-instance population) has
genuine value and is independently shippable. Landing "handler surface only" is
an honest, scoped delivery; the drift entry documents the gap clearly.

Ruling-3 Option (A) is hereby superseded by this Ruling-6 Option (B) for the
specific sub-question of production tracker population. The handler surface
requirement from Ruling-3 (delete true empty stubs, establish the `pathTrackerSource`
adapter shape) remains in force — only the production population step is deferred.

### Story-Spec Impacts (S-W5.04 fix-burst)

**S-W5.04 AC-006** — revise the fixture requirement to permit test-only population:

> _Replace:_ "AC-006 integration test spins a daemon with a real `PathTracker`
> instance populated with at least one synthetic path entry."
>
> _With:_ "AC-006 integration test populates `pathTrackerSource` with at least
> one synthetic `PathTracker` instance and asserts that `GET /paths` returns
> that entry. Production population of `pathTrackerSource` from the routing
> subsystem is deferred to `S-BL.PATH-TRACKER-WIRING`. The test MUST exercise
> the full handler→source→response code path; direct response fabrication or
> bypassing the source interface is not permitted."

**S-W5.04 `cmd/switchboard/metrics_wire.go`** — add `#DEFERRED` comment:

> On the `pathTrackerSource` field (or its initialization site), add:
> `// #DEFERRED: production population from routing registry deferred to
> S-BL.PATH-TRACKER-WIRING. Test-time registration via .register() is
> the only population path in this wave.`

**BC-2.06.003** — no change required. The response-shape and field-semantics
postconditions are unaffected; only the production data-source is deferred.

### Follow-on Stories

One new story stub must be added to STORY-INDEX.md in the fix-burst:

| Story ID | Title | Depends on | Wave target | Owner |
|----------|-------|-----------|-------------|-------|
| S-BL.PATH-TRACKER-WIRING | Wire `pathTrackerSource` to routing registry for live-daemon PathTracker enumeration | S-W5.04 (merged), S-BL.ROUTER-ADDR | Wave-7 Backlog | implementer + story-writer |

Story `S-BL.PATH-TRACKER-WIRING` scope: add a tracker-enumeration surface to the
routing subsystem (e.g., `func (r *Router) PathTrackers() map[string]paths.PathTracker`
or equivalent); populate `pathTrackerSource` from that registry in
`cmd/switchboard/metrics_wire.go`; update AC-006 to assert live-daemon non-empty
`PathsListResponse`; remove the `#DEFERRED` comment; remove the interim-state note
from S-W5.04 AC-006. MUST depend on `S-BL.ROUTER-ADDR` (routing address must be
populated before tracker wiring is tested end-to-end).

### Ordering

`S-BL.ROUTER-ADDR` → `S-BL.PATH-TRACKER-WIRING` → `S-BL.PATH-FAILED-STATUS`.
All three are Wave-7 Backlog. None block Wave-6 wave-convergence.

---

## Ruling 7 — S-6.07 F-Impl-002: AC-003 "AND RoleControl" — Descriptive or Defense-in-Depth?

**Finding summary (Pass-3 Lens-1 BLOCK):** AC-003 states the handler MUST verify
the caller is the bootstrap key AND that the caller's role is `RoleControl`. The
implementation at `cmd/switchboard/admin_handlers.go` (approximately line 610)
performs only `IsBootstrapKey(callerPub)`. It relies on the structural invariant
that `SVTNManager.Create` seeds the bootstrap key as `RoleControl`, and that
`ErrBootstrapKeyRevokeForbidden` / `ErrBootstrapKeyExpireForbidden` prevent role
transitions — so bootstrap ⟹ RoleControl by construction. Pass-3 Lens-1 flagged
this as BLOCK: AC-003's "AND RoleControl" wording is normative, and the
implementation satisfies only half of it.

**Options considered:**

- **(A) Reword AC-003 to match impl:** update AC-003 to read "handler verifies
  caller is bootstrap key (which by BC-2.07.001 Inv-3 implies RoleControl)."
  No handler code change. Add a code comment at the `IsBootstrapKey` call site
  in `admin_handlers.go` documenting the invariant that makes the role check
  redundant. The normative test assertion is narrowed to bootstrap-key presence.

- **(B) Defense-in-depth — add explicit role check:** add a two-line role check
  after the bootstrap check: fetch caller role via `keySet.Role(callerPub)`,
  return `E-ADM-009` if not `RoleControl`. Add a mutation test that exercises the
  role-check independently (i.e., a test that would fail if the role check were
  removed). AC-003 wording is unchanged.

**Ruling: Option (B) — defense-in-depth.**

AC-003's "AND RoleControl" phrasing is normative and was written intentionally.
Weakening it to "implies RoleControl" via an invariant argument is a spec retreat
that buys nothing — the explicit role check is two lines of code and one mutation
test. The deeper reason to prefer Option (B) is forward safety: any future change
that admits a non-control bootstrap key (e.g., a bootstrap-key rotation flow where
the outgoing bootstrap key retains its `is_bootstrap` flag but has been demoted)
would silently break the invariant and the bootstrap-only check would pass without
the role gate triggering. The invariant argument is sound today; it is not a
durable architectural guarantee. Defense-in-depth costs nothing and future-proofs
the gate.

Option (A) is rejected. Rewriting a normative AC to match an incomplete
implementation sets a bad precedent and removes a real safety property.

### Story-Spec Impacts (S-6.07 fix-burst)

**S-6.07 `cmd/switchboard/admin_handlers.go` (~line 610)** — add role check after
bootstrap-key verification:

> After `IsBootstrapKey(callerPub)` returns true, add:
>
> ```go
> // Defense-in-depth: BC-2.07.001 Inv-3 mandates bootstrap key implies
> // RoleControl, but verify explicitly so future key-model changes cannot
> // silently bypass this gate.
> if keySet.Role(callerPub) != auth.RoleControl {
>     return adminErrorResponse(E_ADM_009, "caller is not RoleControl"), nil
> }
> ```
>
> The `keySet.Role` call MUST occur inside the existing auth-verified scope (after
> signature verification, before business logic). The error code is `E-ADM-009`
> (unauthorized), consistent with the bootstrap-key failure path.

**S-6.07 AC-003** — add mutation-test requirement:

> After the existing assertion text, add: "A mutation test MUST be added that
> removes the `RoleControl` check independently of the bootstrap-key check and
> asserts the test fails. This verifies the role gate is independently effective
> and cannot be silently removed."

**BC-2.07.001 Inv-3** — add defense-in-depth note (no semantic change):

> After the existing tightened text (from Ruling 2 of this document), add:
> "Implementations MUST check `RoleControl` explicitly in addition to
> `IsBootstrapKey`; relying solely on the structural invariant that bootstrap ⟹
> RoleControl is insufficient — the role MUST be verified independently as
> defense-in-depth."

### Follow-on Stories

No new stories are required. The fix is entirely within the S-6.07 fix-burst scope.

### Ordering

Ruling 7 fix (role check + mutation test) is independent of all other S-6.07
fix-burst items. It may be applied in any order within the fix-burst. No
dependency on S-W5.04 or any backlog story.

---

## Summary of Spec Changes — Rulings 6–7

| File | Change Summary | Finding(s) Closed |
|------|----------------|-------------------|
| S-W5.04 AC-006 | Revise fixture: permit test-only `pathTrackerSource` population; add `#DEFERRED` prose; prohibit direct response fabrication | F-P3L1-002, F-L2-01 |
| S-W5.04 `metrics_wire.go` | Add `// #DEFERRED: S-BL.PATH-TRACKER-WIRING` comment at `pathTrackerSource` init | F-P3L1-002, F-L2-01 |
| STORY-INDEX.md | Add `S-BL.PATH-TRACKER-WIRING` stub (Wave-7 Backlog; depends on S-W5.04 + S-BL.ROUTER-ADDR) | — |
| S-6.07 `admin_handlers.go` (~line 610) | Add explicit `keySet.Role(callerPub) != RoleControl` check after `IsBootstrapKey`; return `E-ADM-009` on failure | F-Impl-002 |
| S-6.07 AC-003 | Add mutation-test requirement: test must fail if role check is removed independently | F-Impl-002 |
| BC-2.07.001 Inv-3 | Add defense-in-depth note: implementations MUST check `RoleControl` explicitly, not rely solely on structural invariant | F-Impl-002 |

---

## Ruling 8 — S-6.07 Genesis Carve-Out for Defense-in-Depth Role Check (F-P4L1-002 + O-P4L3-01)

### Background

BC-2.07.001 v1.6 Inv-3 and VP-048 v1.4 property (3) — both landed by Ruling-7 —
require: "Implementations MUST check `caller.role == RoleControl` explicitly after
`IsBootstrapKey(caller)` returns true. A caller passing the bootstrap-key check with
`role != RoleControl` MUST be rejected with E-ADM-009 **before any SVTN state is
consulted** (existence oracle closed)."

Pass-4 adversarial review (F-P4L1-002, O-P4L3-01) flags that the implementation at
`admin_handlers.go:627` short-circuits as:

```go
if hasExistingSVTNs := m.HasAnySVTN(); hasExistingSVTNs && !m.BootstrapKeyHasControlRole() {
    // ... reject
}
```

On genesis (zero SVTNs), `HasAnySVTN()` returns false and the whole expression
short-circuits — the `BootstrapKeyHasControlRole()` check is **skipped**. On the
non-genesis path, both sides of the `&&` are evaluated. The finding asks: does the
genesis skip violate Inv-3 / VP-048 property (3)?

**Root cause of the skip:** at genesis, no keySet entry yet exists for the bootstrap
key (SVTNs haven't been created yet, so no key registration has occurred). Calling
`BootstrapKeyHasControlRole()` in this state would return false by construction —
blocking the genesis create, which is the very operation the bootstrap key exists to
perform.

### Options Considered

- **(A) Spec-side narrow:** update BC-2.07.001 Inv-3 and VP-048 property (3) to
  explicitly acknowledge the genesis carve-out. The spec reads: "...except on the
  first-ever SVTN creation (`HasAnySVTN() == false`), when no keySet entry yet exists
  to check; on that path the `IsBootstrapKey` check alone suffices, and the bootstrap
  key is registered as `RoleControl` by the immediately-following `Create()` call as
  part of the genesis atomic operation." Implementation unchanged. Mutation test scope
  is limited to the non-genesis path.

- **(B) Impl-side keyset-free check:** add a `bootstrapRole admission.KeyRole` field to
  `SVTNManager`, seeded to `RoleControl` at `NewSVTNManager` construction.
  `BootstrapKeyHasControlRole()` returns `m.bootstrapRole == RoleControl` (pure struct
  predicate, never consults keySet). Remove the `HasAnySVTN()` gate. The DiD check
  becomes unconditional.

- **(C) Hybrid:** ship Option A now (spec narrow) + open `S-BL.BOOTSTRAP-ROLE-DID` as
  a Wave-7 backlog stub to implement Option B later.

### Ruling: Option A (spec-side narrow) + mandatory test-scope fix for F-P4L1-001

**Genesis carve-out:** the genesis path is authentically special. At the moment of
the first `admin.svtn.create` call, the manager IS the authoritative source for the
bootstrap key's role — it holds that role as a constructor invariant — and no keySet
entry exists yet to verify against. The manager's constructor guarantees `RoleControl`;
no external corruption path exists between construction and the first create call.
Layering a second in-memory predicate (`bootstrapRole` field) that mirrors the
constructor argument (Option B) provides defense in shape only: a bug in the constructor
propagates identically to both fields, so the check adds no independent failure signal.
The real defensive value of Ruling-7's explicit role check is on **non-genesis paths**
where key rotation or provisioning refactors could demote the bootstrap key mid-lifetime
— those paths have a live keySet to check against, and the check is unconditional there.

Option B is not wrong but is not urgent: the constructor-seeded field and the keySet
are both under the same package's control, and a rotation refactor would need to update
both consistently anyway. The Wave-7 backlog stub (Option C) is a reasonable hedge if
the team wants to carry the Option B work forward; however, it is not required for
convergence and is left to the implementer's discretion.

**BC-2.07.001 Inv-3 addendum (genesis carve-out):** after the existing defense-in-depth
note (Ruling-7), add:

> "**Genesis carve-out (Ruling-8, 2026-07-01):** On the first-ever SVTN creation
> (`HasAnySVTN() == false`), no keySet entry yet exists for the bootstrap key (the key
> is registered by the `Create()` call itself as part of the genesis atomic operation).
> On this path, `IsBootstrapKey(caller)` alone is sufficient authorization; the
> `caller.role == RoleControl` keySet lookup is skipped because the keySet is empty.
> The genesis carve-out is not a privilege bypass — the bootstrap role is guaranteed by
> the manager constructor. The explicit `RoleControl` check applies unconditionally on
> all non-genesis paths (i.e., when at least one SVTN already exists)."

**VP-048 property (3) addendum:** mirror the genesis carve-out in the mutation-test
scope note:

> "The mutation test for the `caller.role == RoleControl` guard targets the
> **non-genesis path only** (`HasAnySVTN() == true`). The genesis path
> (`HasAnySVTN() == false`) is exempt per the genesis carve-out; a mutation test
> on the genesis path would falsely pass because the keySet is empty and the role
> lookup returns false by construction regardless of the guard."

### Mandatory Action Item: F-P4L1-001 — Test Helper Must Move to Test Scope

`SeedSVTNWithoutBootstrapKeyForTest` is currently exported from
`internal/svtnmgmt/svtnmgmt.go` (a production file). Exporting test-helper functions
from production packages is a security-surface hygiene violation: it enlarges the
production binary's exported surface, signals to callers that deliberately constructing
a bootstrap-key-absent SVTN is a supported operation, and makes security audits harder
because the distinction between "real" and "test-only" APIs is lost.

**Ruling: this function MUST be moved** to a `_test.go` file within
`internal/svtnmgmt` (if only used in that package's tests) **or** to a dedicated
`internal/svtnmgmttest` helper package (if used by tests in other packages). There is
no negotiation on this point. The fix-burst for S-6.07 MUST include this relocation.

If the function is referenced by tests in packages outside `internal/svtnmgmt`:
- Create `internal/svtnmgmttest/helpers.go` containing `SeedSVTNWithoutBootstrapKeyForTest`
  and any other test-only construction helpers.
- Remove the export from the production file.
- Update all import sites.

If the function is only used within `internal/svtnmgmt` tests:
- Move the body to `internal/svtnmgmt/svtnmgmt_test.go` (or a new `*_test.go` file
  in the same package).
- Remove the export from the production file.

The production package MUST NOT export any symbol containing `ForTest`, `test`, or
`Test` in its name after the fix-burst.

### Downstream Artifact Impacts

| Artifact | Required Change |
|----------|----------------|
| BC-2.07.001 v1.6 → v1.7 | Append genesis carve-out paragraph to Inv-3 defense-in-depth note |
| VP-048 v1.4 → v1.5 | Append genesis carve-out scope note to mutation-test invariant (property 3) |
| S-6.07 fix-burst impl | Move `SeedSVTNWithoutBootstrapKeyForTest` out of production file (mandatory — F-P4L1-001) |
| S-6.07 story spec | Add implementation note: genesis carve-out is spec-authorized; no handler code change needed for the genesis path; test relocation is a mandatory action item |

### Ordering

BC-2.07.001 and VP-048 spec edits may be applied in any order within the fix-burst.
The `SeedSVTNWithoutBootstrapKeyForTest` relocation is a mandatory prerequisite for
S-6.07 pass-5 adversarial; it is not sequenced after any other Ruling-8 item.

---

## Ruling 9 — S-W5.04 AC-003 Status Enum Mapping for `Active=false` (F-P4L2-07)

### Background

AC-003 (story line 84–89) currently reads: "status: `"degraded"` when
`PathSnapshot.Degraded == true` and to `"active"` otherwise."

Under a strict reading of "otherwise," the mapping is:

| `Active` | `Degraded` | AC-003 strict output |
|----------|-----------|----------------------|
| true     | false     | `"active"`           |
| true     | true      | `"degraded"`         |
| false    | false     | `"active"` ← ambiguous |
| false    | true      | `"degraded"`         |

The implementation in `PathEntryFromSnapshot` (`handlers.go:139`) uses:
`Active=false OR Degraded=true → "degraded"`, and the test
`TestPathEntry_StatusFromDegraded` includes a row `active_false_is_degraded` that
asserts `Active=false, Degraded=false → "degraded"`. The impl and test agree; the
spec is silent on the `Active=false, Degraded=false` case.

BC-2.06.003 v1.11 PC-1 defines the status enum as `{active, degraded}` where
`"degraded"` covers "RTT-degraded liveness." It does not explicitly address the
`Active=false` case.

### Question

Does `Active=false, Degraded=false` (path provisioned but not yet carrying traffic)
map to `"active"` (the strict "otherwise" reading) or to `"degraded"` (the impl)?

### Options Considered

- **(A) Impl-preserving (update spec to match impl):** update AC-003 to: "status:
  `"degraded"` when `PathSnapshot.Active == false OR PathSnapshot.Degraded == true`;
  `"active"` when both `Active == true AND Degraded == false`." Also update
  BC-2.06.003 PC-1 to include this mapping normatively.

- **(B) Spec-preserving (update impl to match strict AC-003 reading):** update impl
  so `Active=false, Degraded=false → "active"`. Update the `active_false_is_degraded`
  test row. Rationale: an inactive-but-not-degraded path could reasonably be reported
  as `"active"` if operators interpret "active" as "provisioned and healthy" rather
  than "carrying traffic right now."

### Ruling: Option A (impl-preserving)

For operator-facing metrics, "active" MUST mean "carrying traffic AND healthy."
`Active=false` means the path is provisioned but not currently forwarding traffic —
operationally, this is a non-functional path from the operator's perspective.
Reporting it as `"active"` would mislead operators: they would see a path in the
list with status "active" that is not actually passing any frames. The resulting
diagnostic confusion (an "active" path that shows zero throughput) is worse than
the minor semantic stretch of calling a not-yet-live path "degraded."

Additionally: the `Active` flag in `PathSnapshot` represents whether the PathTracker
considers the path live at this instant. A path where `Active=false` has had its
liveness signal absent — whether because it was never started or because it stopped.
From an observability standpoint, this is indistinguishable from a degraded path: the
operator needs to take action in both cases. Collapsing the two into `"degraded"` is
the correct operator ergonomic.

Option B is rejected because the `"active"` label for a path with `Active=false`
creates a semantic inconsistency: the word "active" would mean "not degraded by
latency" rather than "currently active." This is a terminology trap — the existing
field name `Active` in `PathSnapshot` already carries the meaning of "currently
passing traffic," and reversing it in the output JSON status would require downstream
tooling to understand a non-obvious inversion.

**Spec update required (BC-2.06.003 PC-1):** add the following after the `status`
field definition:

> "**Status derivation rule (Ruling-9, 2026-07-01):** `status` is `"active"` if and
> only if `PathSnapshot.Active == true AND PathSnapshot.Degraded == false`. In all
> other cases — including `Active=false, Degraded=false` (path provisioned but not
> carrying traffic) — `status` is `"degraded"`. Rationale: `"active"` MUST reflect
> that the path is currently forwarding traffic; a path where `Active=false` is
> operationally non-functional regardless of its degradation flag."

**AC-003 update required (S-W5.04 story):** replace the current AC-003 status mapping
sentence with:

> "status: `"degraded"` when `PathSnapshot.Active == false OR PathSnapshot.Degraded == true`;
> `"active"` when `Active == true AND Degraded == false`. The `active_false_is_degraded`
> test row in `TestPathEntry_StatusFromDegraded` is normative. The strict
> `Degraded==false → active` reading that ignores `Active` is superseded by this ruling."

### Downstream Artifact Impacts

| Artifact | Required Change |
|----------|----------------|
| BC-2.06.003 v1.11 → v1.12 | Append Ruling-9 status derivation rule to PC-1 `status` field definition |
| S-W5.04 AC-003 (story line 84–89) | Replace status mapping sentence with `Active=false OR Degraded=true → "degraded"` formulation; add reference to Ruling-9 |
| S-W5.04 test `TestPathEntry_StatusFromDegraded` | No change required — `active_false_is_degraded` row is already correct and is now normatively anchored |
| No impl change required | Handler and test already implement the Ruling-9 mapping |

### Ordering

BC-2.06.003 and AC-003 edits are independent and may be applied in any order within
the fix-burst. No dependency on any other ruling. No new follow-on stories required.

---

## Summary of Spec Changes — Rulings 8–9

| Artifact | Change Summary | Finding(s) Closed |
|----------|----------------|-------------------|
| BC-2.07.001 v1.6 → v1.7 | Append genesis carve-out paragraph to Inv-3 defense-in-depth note: `HasAnySVTN()==false` path is exempt from keySet role check | F-P4L1-002, O-P4L3-01 |
| VP-048 v1.4 → v1.5 | Append genesis carve-out scope note to mutation-test invariant: mutation test targets non-genesis path only | F-P4L1-002, O-P4L3-01 |
| S-6.07 fix-burst impl | Move `SeedSVTNWithoutBootstrapKeyForTest` out of production `svtnmgmt.go` into `_test.go` or `internal/svtnmgmttest` (mandatory) | F-P4L1-001 |
| BC-2.06.003 v1.11 → v1.12 | Append Ruling-9 status derivation rule to PC-1: `active` iff `Active==true AND Degraded==false`; all other cases map to `"degraded"` | F-P4L2-07 |
| S-W5.04 AC-003 | Replace "otherwise" with explicit `Active==false OR Degraded==true → degraded` formulation; `active_false_is_degraded` test row is now normative | F-P4L2-07 |

---

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-01 | Initial: Rulings 1–2 (router_addr wire shape, authority model) |
| 1.1 | 2026-07-01 | Rulings 3–5 (PathTracker wiring P2, status=failed deferral, bootstrap fast-path fix) |
| 1.2 | 2026-07-01 | Rulings 6–7 (PathTracker wiring P3 — defer to S-BL; AC-003 defense-in-depth) |
| 1.3 | 2026-07-01 | Rulings 8–9 (genesis carve-out for DiD role check + SeedSVTNWithoutBootstrapKeyForTest relocation; Active=false status mapping) |
