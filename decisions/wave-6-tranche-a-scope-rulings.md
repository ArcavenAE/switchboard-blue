---
artifact_id: wave-6-tranche-a-scope-rulings
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-W5.04, S-6.07]
closes_findings: [F-P1L1-003, F-P1L1-004, F-P1L1-005, F-P1L1-003-stutter]
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
