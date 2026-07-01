---
artifact_id: wave-6-tranche-a-scope-rulings
document_type: decision
level: ops
version: "1.10"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
modified:
  - 2026-07-01T00:00:00 # v1.5 — F-P5L3R-09 (Pass-6 L3): Ruling-9 downstream impact table corrected — BC-2.06.003 target version changed from v1.11→v1.12 to v1.11→v1.13 at two sites (Downstream Artifact Impacts table and Summary of Spec Changes table); v1.12 was an interim hop, actual delivered version is v1.13.
  - 2026-07-01T00:00:00 # v1.6 — Ruling-11: mgmt-layer wire envelope contract formalized; S-6.07 AC-003/AC-004/AC-005 wire-envelope amendments; E-ADM-009 message-format fix (F-Lens1-02); Ruling-6 pre-emption on pathTrackerSource.mu accepted; POL-002 story-index-row-sync policy flag (spec-steward applies).
  - 2026-07-01T00:00:00 # v1.7 — Ruling-12: wire-envelope universality note (E-ADM-009/E-SVTN-001/E-CFG-001/E-INT-001 all follow E-RPC-011 pattern); canonical role-label for unresolvable caller unified to "unregistered"; BC-2.07.001 genesis-path vector; POL-002 schema alignment flagged; BC-2.07.001 modified-list reorder; S-BL.POLICY-SCHEMA-VALIDATOR stub flagged.
  - 2026-07-01T00:00:00 # v1.8 — Ruling-12 §1 amended: E-INT-999 added to enumerated handler-code list as catch-all default sentinel; Ruling-12 §7 (new): process policy requiring synchronized three-part update when introducing a new handler-code family.
  - 2026-07-01T16:30:00 # v1.9 — §8 (new): BC-2.06.003 v1.14 EC-008 spec-tightening (empty-paths quality:'pending' ratification).
  - 2026-07-01T00:00:00 # v1.10 — Ruling-13 (§9): F-P12L1-02 ruled BY DESIGN — E-RPC-001 as dispatch bucket is intentional; discrimination by message prefix is the spec contract; noted in S-6.07 §Wire Envelope Contract. Ruling-14 (§10): F-P12L1-01 ruled IN SCOPE — dispatch() response decode MUST wrap io.ErrUnexpectedEOF with E-RPC-002 per ADR-012 §6 Authenticate parity.
cycle: v1.0.0-greenfield
stories_in_scope: [S-W5.04, S-6.07]
closes_findings: [F-P1L1-003, F-P1L1-004, F-P1L1-005, F-P1L1-003-stutter, F-P3L1-002, F-L2-01, F-Impl-002, F-P4L1-001, F-P4L1-002, O-P4L3-01, F-P4L2-07, F-L2-A1-02, F-L2-A1-03, F-L2-A1-04]
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
| BC-2.06.003 v1.11 → v1.13 | Append Ruling-9 status derivation rule to PC-1 `status` field definition (note: v1.12 was an interim hop during the fix-burst; actual delivered version is v1.13) |
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
| BC-2.06.003 v1.11 → v1.13 | Append Ruling-9 status derivation rule to PC-1: `active` iff `Active==true AND Degraded==false`; all other cases map to `"degraded"` (note: v1.12 was an interim hop; actual delivered version is v1.13) | F-P4L2-07 |
| S-W5.04 AC-003 | Replace "otherwise" with explicit `Active==false OR Degraded==true → degraded` formulation; `active_false_is_degraded` test row is now normative | F-P4L2-07 |

---

---

## Ruling 10 — "Production Package" Definition + F-P4L1-004/F-L2-A1-01/F-L2-A1-02/F-L2-A1-03 Closure

### Background

Adversarial Pass-5 (S-6.07) surfaced ambiguity in Ruling-8 F-P4L1-001's phrase
"the production package MUST NOT export any symbol containing `ForTest`/`test`/`Test`".
Two interpretations existed: (a) only `internal/svtnmgmt` counts as production;
(b) `internal/svtnmgmttest` (a test-helper sibling package containing no `_test.go`
files) also counts.

### Ruling

**1. "Production package" definition.** "Production package" in F-P4L1-001 refers to
the ORIGINAL OWNER PACKAGE ONLY (`internal/svtnmgmt`). Sibling test-helper packages
under the `internal/*test/` naming convention (e.g. `svtnmgmttest`) are NOT production
for this rule, mirroring Go stdlib convention (`httptest`, `iotest`, `fstest`).

**2. `InsertRawSVTN` retention with runtime guard.** `InsertRawSVTN` may remain an
exported production method on `*SVTNManager` PROVIDED it:

- Adds a runtime guard at method entry (Go 1.21+):
  ```go
  if !testing.Testing() {
      panic("InsertRawSVTN: test-only mutation seam invoked from production")
  }
  ```
- Carries a `// SECURITY:` docstring flagging it as a bootstrap-invariant-bypass
  seam reachable only from test binaries.
- Sets `CreatedAt: time.Now().UTC()` on the inserted `SVTN`, achieving parity with
  `Create()` (closes F-L2-A1-02).
- Gets a direct unit test `TestInsertRawSVTN_DuplicateName` covering the
  duplicate-name error branch (closes F-L2-A1-03).

**3. `svtnmgmttest.SeedSVTNWithoutBootstrapKey` postcondition assertions.**
`svtnmgmttest.SeedSVTNWithoutBootstrapKey` (no `ForTest` suffix; see Task 9a
update below) gains inline postcondition assertions at the call site:
`HasAnySVTN()==true`, `!BootstrapKeyHasControlRole()`. These make the helper
mutation-resistant (closes F-L2-A1-04).

**4. S-6.07 story Task 9a update.** Update Task 9a in S-6.07 to:

- (a) Crystallize option (b) as the ratified location for the relocated symbol
  (`internal/svtnmgmttest`).
- (b) Name the post-relocation symbol as `SeedSVTNWithoutBootstrapKey` (drop the
  `ForTest` suffix — the package name already signals test-only scope).
- (c) Reference this Ruling-10 for the runtime-guard, `// SECURITY:` docstring, and
  `CreatedAt: time.Now().UTC()` requirements on `InsertRawSVTN`.

### Version Bumps Triggered

| Artifact | From → To | Purpose |
|----------|-----------|---------|
| decisions/wave-6-tranche-a-scope-rulings.md | v1.3 → v1.4 | Ruling-10 appended |
| stories/S-6.07-svtn-admin-create.md | v1.4 → v1.5 | Task 9a crystallize + Ruling-10 anchor |

### Downstream Artifact Impacts

| Artifact | Required Change |
|----------|----------------|
| S-6.07 Task 9a (story v1.4 → v1.5) | Crystallize `internal/svtnmgmttest` as ratified location; rename symbol to `SeedSVTNWithoutBootstrapKey`; reference Ruling-10 for `InsertRawSVTN` requirements |
| `internal/svtnmgmt/svtnmgmt.go` | Add `testing.Testing()` runtime guard + `// SECURITY:` docstring to `InsertRawSVTN`; set `CreatedAt: time.Now().UTC()` on insert (F-L2-A1-02) |
| `internal/svtnmgmt/*_test.go` | Add `TestInsertRawSVTN_DuplicateName` table-driven test (F-L2-A1-03) |
| `internal/svtnmgmttest/helpers.go` | Rename symbol from `SeedSVTNWithoutBootstrapKeyForTest` → `SeedSVTNWithoutBootstrapKey`; add postcondition assertions (F-L2-A1-04) |

### Ordering

All four implementation items above are within the S-6.07 fix-burst scope. No new
follow-on stories are required. No dependency on any backlog story. The story-writer
handles story body propagation (bc_array_changes_propagate_to_body_and_acs) after
this ruling is committed.

---

## Summary of Spec Changes — Ruling 10

| Artifact | Change Summary | Finding(s) Closed |
|----------|----------------|-------------------|
| `internal/svtnmgmt/svtnmgmt.go` | Add `testing.Testing()` runtime guard + `// SECURITY:` docstring to `InsertRawSVTN` | F-P4L1-004, F-L2-A1-01 |
| `internal/svtnmgmt/svtnmgmt.go` | Set `CreatedAt: time.Now().UTC()` on `InsertRawSVTN` insert, matching `Create()` parity | F-L2-A1-02 |
| `internal/svtnmgmt/*_test.go` | Add `TestInsertRawSVTN_DuplicateName` covering duplicate-name error branch | F-L2-A1-03 |
| `internal/svtnmgmttest/helpers.go` | Rename to `SeedSVTNWithoutBootstrapKey`; add `HasAnySVTN()==true` + `!BootstrapKeyHasControlRole()` postcondition assertions | F-L2-A1-04 |
| S-6.07 Task 9a (v1.4 → v1.5) | Crystallize `svtnmgmttest` location; rename symbol; anchor Ruling-10 requirements | — |

---

## Ruling 11 — Mgmt-Layer Wire Envelope Contract Formalization + Ruling-6 Preemption Acceptance

### Background

Pass-6 Lens-1 on S-6.07 surfaced that the mgmt-layer RPC envelope stamps
`error.code = "E-RPC-011"` for all handler failures, while S-6.07's AC-005 as
written required `error.code: "E-SVTN-001"`. The mgmt-layer stamping predates
S-6.07 and is documented as "mgmt.go is the sole authority for stamping E-RPC-011
on the wire envelope" (`admin_handlers.go:417-422`). This is the intentional
wire-envelope contract — handler-specific codes are carried as `error.message`
prefix, not as `error.code`.

### Ruling

**1. Wire envelope contract (formalized as project convention):**

- `resp.Error.Code` = envelope-level code (always `E-RPC-011` for handler
  failures; other codes for transport/decode failures such as `E-RPC-002`).
- `resp.Error.Message` = handler-specific error, formatted as
  `"<HANDLER-CODE>: <message>"` (e.g., `"E-SVTN-001: SVTN already exists: mynet"`).
- Handler-specific codes (`E-SVTN-*`, `E-ADM-*`, `E-CFG-*`) are extracted by
  parsing the message prefix; they are NOT wire-envelope codes.
- sbctl and any other RPC client MUST parse `resp.Error.Message` for
  handler-code discrimination.

**2. S-6.07 AC amendments (story-writer applies):**

- AC-003: change "returns E-ADM-009" → "returns wire envelope
  `{error: {code: 'E-RPC-011', message: 'E-ADM-009: insufficient authority for
  operation admin.svtn.create: key <fp> has role <role>'}}`".
- AC-004: same amendment pattern for `E-RPC-002` transport errors (which DO
  surface as wire `code`).
- AC-005: change to "returns wire envelope
  `{error: {code: 'E-RPC-011', message: 'E-SVTN-001: SVTN already exists: <name>'}}`".

**3. F-Lens1-02 message-format reconcile (implementer applies):**

The E-ADM-009 message when `IsBootstrapKey==false` currently reads
`"key <fp> is not the daemon bootstrap key"`. BC-2.07.001 canonical test vector
requires `"key <fp> has role <role>"`. Fix: in the E-ADM-009 rejection branch,
when `IsBootstrapKey` returns false, resolve the caller's actual role via
`m.CallerKeyRoleActive(callerPub)` for diagnostic message ONLY (does NOT change
authority decision — still rejects). If role lookup returns not-found, use
literal string `"unknown"`. Message format becomes:
`"insufficient authority for operation admin.svtn.create: key <fp> has role <role>"`
matching BC canonical vector.

**4. Accept Ruling-6 pre-emption on `pathTrackerSource.mu` (F-L1-01):**

The `sync.RWMutex` field on `cmd/switchboard/metrics_wire.go:40` is pre-landed
for `S-BL.PATH-TRACKER-WIRING` per Ruling-6 defer scope. Doc comment already
explains ("reserved for Wave-7 writer"). NOT a defect. No action.

**5. POL-002 story-index-row-sync process-gap policy (spec-steward applies):**

The following policy is flagged for spec-steward to add to
`.factory/policies.yaml`:

- **ID:** POL-002
- **Name:** `story-index-row-sync`
- **Description:** When a story's frontmatter `version:` is bumped, STORY-INDEX
  row's status cell (`draft (vX.Y)`) MUST be updated in the same fix-burst or
  before the next per-story adversarial pass. Missing sync = MED finding.
- **Scope:** `stories/*.md` ↔ `stories/STORY-INDEX.md`
- **Severity:** MED

### Version Bumps Triggered

| Artifact | From → To | Purpose |
|----------|-----------|---------|
| decisions/wave-6-tranche-a-scope-rulings.md | v1.5 → v1.6 | Ruling-11 appended |
| stories/S-6.07-svtn-admin-create.md | v1.5 → v1.6 | AC-003/AC-004/AC-005 amendments |

### Downstream Artifact Impacts

| Artifact | Required Change |
|----------|----------------|
| S-6.07 AC-003 (story v1.5 → v1.6) | Change "returns E-ADM-009" to wire-envelope form: `{error: {code: 'E-RPC-011', message: 'E-ADM-009: insufficient authority for operation admin.svtn.create: key <fp> has role <role>'}}` |
| S-6.07 AC-004 (story v1.5 → v1.6) | Same amendment pattern for E-RPC-002 transport errors |
| S-6.07 AC-005 (story v1.5 → v1.6) | Change to wire-envelope form: `{error: {code: 'E-RPC-011', message: 'E-SVTN-001: SVTN already exists: <name>'}}` |
| `cmd/switchboard/admin_handlers.go` | Fix E-ADM-009 rejection branch message: call `m.CallerKeyRoleActive(callerPub)` for diagnostic role resolution; format as `"insufficient authority for operation admin.svtn.create: key <fp> has role <role>"`; fall back to `"unknown"` if role lookup returns not-found |
| `.factory/policies.yaml` | Add POL-002 `story-index-row-sync` policy (spec-steward applies) |

### Ordering

S-6.07 story spec edits (AC-003/AC-004/AC-005) are applied by story-writer in the
same fix-burst. The E-ADM-009 message-format fix is applied by the implementer.
POL-002 is applied by spec-steward independently and does not block the S-6.07
fix-burst. The `pathTrackerSource.mu` finding is closed with no action.

---

## Ruling 12 — Wire-Envelope Contract: Universal Handler-Code Coverage + Genesis-Path Role Label

### Background

Pass-7 Lens-1 on S-6.07 surfaced four gaps in Ruling-11's coverage:

1. **F-Impl1-02:** E-INT-001 (crypto/rand failure) handler code not enumerated in AC list.
2. **O-Impl1-03:** E-CFG-001 (args validation) handler code not enumerated in AC list.
3. **F-Impl1-05:** Genesis-path E-ADM-009 message resolves role to string literal `"unknown"` — untested.
4. **F-Impl1-06:** `resolveAndVerifyCallerRole` uses `"unregistered"` while `admin.svtn.create` uses
   `"unknown"` for the same semantic condition (caller cannot be role-resolved).

### Ruling

**1. Wire-envelope contract is UNIVERSAL for all handler-code errors.**

Every handler-code error — `E-ADM-*`, `E-SVTN-*`, `E-CFG-*`, `E-INT-*`, and any future
`E-XXX-YYY` family — follows the Ruling-11 envelope pattern:

```
{ code: "E-RPC-011", message: "E-XXX-YYY: <detail>" }
```

Only transport-layer codes (`E-RPC-002`, `E-RPC-004`, etc.) surface as the wire envelope
`code` directly. A §Universality note MUST be added to S-6.07's Wire Envelope Contract
section explicitly enumerating the following as E-RPC-011-wrapped handler codes:

- E-ADM-009 (insufficient authority)
- E-SVTN-001 (SVTN already exists)
- E-CFG-001 (args validation failure)
- E-INT-001 (crypto/rand internal failure)
- E-INT-999 (unmapped internal condition — catch-all default arm of `mapAdminError`; v1.8 amendment)

E-RPC-002 is the transport-layer exception and is NOT wrapped.

**2. Canonical role-label for "caller cannot be role-resolved" is `"unregistered"`, NOT `"unknown"`.**

Across ALL admin authority paths — bootstrap check, `resolveAndVerifyCallerRole`,
`CallerKeyRoleInAny` fallback — the E-ADM-009 message MUST use `"unregistered"` when no
active role is found for the caller key. The current `"unknown"` literal in
`admin_handlers.go` at lines 615 and 628 is incorrect and must be replaced with
`"unregistered"`. The canonical message format is:

```
"insufficient authority for operation admin.svtn.create: key <fp> has role unregistered"
```

This aligns with `resolveAndVerifyCallerRole`'s existing behaviour and eliminates the
semantic bifurcation between `"unknown"` and `"unregistered"` for the same condition.

**3. BC-2.07.001 canonical test vector accepts `"unregistered"` on the genesis path.**

An explicit vector row MUST be added to BC-2.07.001's canonical test vectors for the
genesis-path unauthorized caller case (`HasAnySVTN() == false`, non-bootstrap caller):

| Scenario | Input | Expected wire message |
|----------|-------|-----------------------|
| Genesis path, non-bootstrap caller, no role | Bootstrap key absent; non-bootstrap key presented; zero SVTNs | `"insufficient authority for operation admin.svtn.create: key <fp> has role unregistered"` |

**4. Ruling-11 §5 POL-002 schema alignment (spec-steward applies).**

POL-002 as currently drafted in `.factory/policies.yaml` uses `name:` and `description:`
fields inconsistent with POL-001's `title:` and `rule:` schema. Spec-steward MUST align
POL-002 to POL-001's canonical schema:

```
id: POL-002
title: story-index-row-sync
severity: MED
scope: "stories/*.md <-> stories/STORY-INDEX.md"
rule: >
  When a story's frontmatter version: is bumped, STORY-INDEX row's status cell
  (draft (vX.Y)) MUST be updated in the same fix-burst or before the next
  per-story adversarial pass. Missing sync = MED finding.
rationale: >
  Index drift causes adversarial passes to report stale version numbers as
  findings, wasting a convergence round on a mechanical inconsistency.
```

Bump `.factory/policies.yaml` v1.1 → v1.2 when applying.

**5. BC-2.07.001 modified-list chronological ordering (spec-steward applies).**

The BC-2.07.001 frontmatter `modified:` list currently has v1.9 listed before v1.8
(both dated 2026-07-01), which violates the chronologically-ascending convention
established by Rulings 8 and 9. Spec-steward MUST reorder to:
`v1.6 → v1.7 → v1.8 → v1.9 → v1.10`.

Note: the changelog table in the BC body MAY retain descending order (most recent on
top) — that is the project convention. Only the frontmatter `modified:` list must be
ascending. Bump BC-2.07.001 v1.9 → v1.10 with this hygiene correction.

**7. [Process policy] Introducing a new handler-code family requires a synchronized three-part update.**

When any implementer, product-owner, or spec-steward introduces a new handler-code family
(e.g., a new `E-FOO-*` prefix with its own sentinel set), the following three artifacts MUST
be updated in the same fix-burst:

1. **error-taxonomy.md** — add at least one row for the new family under a correctly labeled
   section header.
2. **Anchor story spec §Universality** — add the new code to the §Universality enumeration
   in the Wire Envelope Contract section of any story that exercises the handler.
3. **Ruling-12 §1 enumeration (this document)** — append the new code to the bullet list in
   §1 above.

Omitting any one of the three is a MED-severity POL-001 compliance gap. E-INT-999
(catch-all default arm) is the reference example: it was added simultaneously to
error-taxonomy.md §INT, to the §1 bullet list above, and to this §7 definition — all in
v1.8 of this document.

**6. [Process gap] policies.yaml lacks a schema validator (follow-on stub).**

The schema drift between POL-001 and POL-002 reflects the absence of any automated
policy-schema enforcement. A stub story `S-BL.POLICY-SCHEMA-VALIDATOR` MUST be added
to STORY-INDEX (P3, S t-shirt, network-management epic) in the next fix-burst. Scope:
implement a `golangci-lint` custom rule OR a `just` target that parses
`.factory/policies.yaml` and rejects policies with divergent schemas (missing required
fields, wrong field names). This story does NOT block the current S-6.07 fix-burst.

### Version Bumps Triggered

| Artifact | From → To | Purpose |
|----------|-----------|---------|
| decisions/wave-6-tranche-a-scope-rulings.md | v1.6 → v1.7 | Ruling-12 appended |
| stories/S-6.07-svtn-admin-create.md | v1.6 → v1.7 | Wire Envelope §Universality note + AC amendments for E-CFG-001/E-INT-001 |
| specs/behavioral-contracts/ss-07/BC-2.07.001.md | v1.9 → v1.10 | Genesis-path canonical vector + modified-list chronological reorder |
| .factory/policies.yaml | v1.1 → v1.2 | POL-002 schema alignment to POL-001 canonical fields |

### Downstream Artifact Impacts

| Artifact | Required Change |
|----------|----------------|
| S-6.07 (story v1.6 → v1.7) | Add §Universality note to Wire Envelope Contract; enumerate E-ADM-009/E-SVTN-001/E-CFG-001/E-INT-001 as E-RPC-011-wrapped; E-RPC-002 remains transport-layer exception |
| `cmd/switchboard/admin_handlers.go` lines 615/628 | Replace `"unknown"` with `"unregistered"` in E-ADM-009 message for unresolvable-role case |
| BC-2.07.001 (v1.9 → v1.10) | Add genesis-path `"unregistered"` canonical test vector row; reorder `modified:` frontmatter list chronologically ascending |
| `.factory/policies.yaml` (v1.1 → v1.2) | Align POL-002 to POL-001 schema: replace `name:`/`description:` with `title:`/`rule:`; add missing `severity:`, `scope:`, `rationale:` fields |
| STORY-INDEX.md | Add `S-BL.POLICY-SCHEMA-VALIDATOR` stub (P3, backlog) |

### Ordering

1. Implementer applies `"unknown"` → `"unregistered"` fix to `admin_handlers.go` (unblocks S-6.07 Pass-7 re-review).
2. Story-writer applies §Universality note to S-6.07 AC amendments (same fix-burst).
3. Spec-steward applies BC-2.07.001 v1.10 vector + modified-list reorder, POL-002 schema alignment, and STORY-INDEX stub.

Items 2 and 3 are independent and may proceed in parallel. None block the `admin_handlers.go` fix.

---

## §8 — Spec-Tightening Record: BC-2.06.003 v1.14 EC-008 Ratification

**Date:** 2026-07-01T16:30:00

**What was ratified:** BC-2.06.003 v1.14 adds EC-008 (edge case: empty-paths
response quality field). When the daemon has no paths to report, the paths list
is empty (`[]`) and the `quality` field is set to `"pending"` rather than a
computed numeric value. This is a spec-tightening, not a behavioral change.

**Scope of change:**
- BC-2.06.003 bumped to v1.14 — EC-008 added defining the `quality: "pending"`
  value for empty-paths responses.
- S-W5.04 body updated under §Edge Cases to anchor EC-008 (cross-reference only;
  no story-scope change).

**Story-scope impact:** none. EC-008 ratification is a spec-tightening that
clarifies already-implemented behavior. No new implementation work is required
in S-W5.04 or any other in-scope story. No STORY-INDEX changes. No sprint-state
changes.

**Downstream artifact note:** story-writer handles body propagation under
`bc_array_changes_propagate_to_body_and_acs` after this ruling is committed.

---

## §9 — Ruling-13: F-P12L1-02 — E-RPC-001 as CLI Dispatch Bucket (BY DESIGN)

**Date:** 2026-07-01
**Finding:** F-P12L1-02 (MED, pending intent)
**Scope:** S-6.07 `cmd/sbctl/client.go:379-383`

### Finding Summary

`connectAndRun` collapses all `dispatch()` failures to `E-RPC-001` in the operator-visible
CLI JSON output. When the daemon returns an envelope `{code: "E-RPC-011", message: "E-SVTN-001:
SVTN already exists: foo"}`, the CLI surfaces `{ok: false, error: {code: "E-RPC-001",
message: "rpc failed: admin.svtn.create: E-SVTN-001: ..."}}`. The top-level `error.code`
visible to the operator is `E-RPC-001`, not `E-SVTN-001` (nor the envelope `E-RPC-011`).
Discrimination requires message-prefix parsing.

### Options Considered

- **(A) BY DESIGN — document as Ruling-13, no code change:** the CLI top-level `error.code`
  is intentionally a stable dispatch-bucket (`E-RPC-001` = "dispatched RPC failed"). Downstream
  tooling MUST parse the message prefix for handler-code discrimination, consistent with the
  existing AC-004/AC-005 §Wire Envelope Contract language. Add a clarifying note to S-6.07
  §Wire Envelope Contract. No code change.

- **(B) IN SCOPE — surface daemon envelope code:** refactor `dispatch()` to return a structured
  `(code, message)` error; refactor `connectAndRun` to write the daemon-emitted envelope code
  as the CLI's top-level `error.code`. Larger change; potentially breaking for tooling that
  currently matches on `E-RPC-001`.

- **(C) DEFER — new story S-BL.CLI-ENVELOPE-CODE-SURFACING.**

### Ruling: Option (A) — BY DESIGN

**Rationale:** AC-004/AC-005 §Wire Envelope Contract already specifies: "clients that need
to discriminate handler-specific failure modes MUST parse the message prefix." This phrasing
is normative — it is the project's explicit decision that message-prefix parsing is the
discrimination channel, not code promotion.

The two-tier code scheme (`E-RPC-001` at CLI surface, handler codes in message) is consistent
with Ruling-11's formalization of the wire envelope contract. The daemon layer uses `E-RPC-011`
as the envelope bucket; the CLI layer uses `E-RPC-001` as the dispatch bucket. Handler codes
(`E-SVTN-001`, `E-ADM-009`, etc.) are always in the message, never in the top-level `code`
at either layer. This is a coherent layered design.

Option (B) is rejected: promoting the daemon envelope code (`E-RPC-011`) or handler codes
(`E-SVTN-001`) to the CLI's top-level `error.code` would break any tooling already parsing
`E-RPC-001` for dispatch-level failures. The refactor also creates a new surface (structured
error return from `dispatch()`) that requires its own BC coverage and adversarial pass.
Option (C) is rejected: there is nothing to defer — the design is intentional and correct.

### Required Action (no code change)

**S-6.07 §Wire Envelope Contract** — add an operator-surface clarification note:

> "**Operator-surface top-level `error.code` (Ruling-13, 2026-07-01):** When `sbctl` invokes
> an admin RPC and the dispatch fails, the CLI JSON output uses `E-RPC-001` as the top-level
> `error.code` regardless of the daemon envelope code (`E-RPC-011`) or handler-specific code
> (`E-SVTN-001`, `E-ADM-009`, etc.). This is intentional: `E-RPC-001` is the sbctl
> dispatch-level bucket. Tooling that needs to discriminate handler-specific failure modes
> MUST parse the `error.message` prefix (per AC-004/AC-005 §Wire Envelope Contract). No
> layer surfaces a handler code as the top-level CLI `error.code`."

### Downstream Artifact Impacts

| Artifact | Required Change |
|----------|----------------|
| S-6.07 §Wire Envelope Contract (story v1.10+) | Add Ruling-13 operator-surface clarification note (see above) |
| No code change | `connectAndRun` behavior is correct as designed |
| No BC change | AC-004/AC-005 §Wire Envelope Contract already contains the "MUST parse message prefix" norm |

### Ordering

Story-writer applies the §Wire Envelope Contract note to S-6.07 in the next fix-burst.
No implementation work. No dependency on any other ruling.

---

## §10 — Ruling-14: F-P12L1-01 — dispatch() Response Decode MUST Wrap io.ErrUnexpectedEOF (IN SCOPE)

**Date:** 2026-07-01
**Finding:** F-P12L1-01 (MED)
**Scope:** S-6.07 `cmd/sbctl/client.go:300-303`

### Finding Summary

`dispatch()` decodes the RPC response body behind an `io.LimitReader` (64 KiB bound).
When the server sends a response larger than the limit, `json.NewDecoder(io.LimitReader(...))`
returns `io.ErrUnexpectedEOF` because the JSON stream is truncated at the limit. The
decode-error branch at lines 300-303 returns the raw error without stamping the
`E-RPC-002: message too large` prefix.

By contrast, `Authenticate()` at lines 210-218 uses the same `io.LimitReader` pattern and
its decode-error branch DOES wrap `io.ErrUnexpectedEOF` with `E-RPC-002: message too large: %w`.

ADR-012 §6 states the bounded-read guard applies to "all reads." The asymmetry between
the oversized-challenge path (correctly stamped `E-RPC-002`) and the oversized-RPC-response
path (bare `unexpected EOF`) violates ADR-012 §6 uniformity.

### Options Considered

- **(A) IN SCOPE — add symmetric io.ErrUnexpectedEOF arm to dispatch():** in `dispatch()`'s
  decode-error branch, add an `errors.Is(err, io.ErrUnexpectedEOF)` check that wraps the error
  as `fmt.Errorf("E-RPC-002: message too large: %w", err)`. Add a symmetric test
  `TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002`.

- **(B) DEFER — new story S-BL.CLI-BOUNDED-READ-SYMMETRY** — drift record + follow-on story stub.

### Ruling: Option (A) — IN SCOPE

**Rationale:** ADR-012 §6 uniformity is a hard invariant, not a recommendation. The bounded-read
guard is documented to apply to "all reads." The `Authenticate()` → `dispatch()` asymmetry means
an oversized server response to any admin RPC surfaces as a cryptic `unexpected EOF` while an
oversized challenge is correctly diagnosed as `E-RPC-002`. This is operator-hostile: the two
conditions are caused by the same mechanism (LimitReader truncation) and should produce the same
error code.

The fix is narrow: two lines of code (`errors.Is` arm + `fmt.Errorf` wrap) and one test. It
does not touch the bounded-read limit value, the LimitReader construction, or any shared helper.
Option (B) is rejected: deferring a two-line ADR uniformity fix accumulates a known
inconsistency and generates a finding in every subsequent adversarial pass until it lands.

### Required Changes

**`cmd/sbctl/client.go` — `dispatch()` decode-error branch (~line 300-303):**

In the `if err != nil` block after `json.NewDecoder(io.LimitReader(...)).Decode(...)`, add:

```go
if errors.Is(err, io.ErrUnexpectedEOF) {
    return fmt.Errorf("E-RPC-002: message too large: %w", err)
}
```

This arm MUST appear BEFORE any generic error return in the same decode-error block.
The `errors` and `fmt` packages are already imported. The pattern is identical to
`Authenticate()`'s existing wrap at lines 210-218.

**Test — `TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002`:**

Add a test that:
1. Spins a test HTTP server returning a response body that exceeds 64 KiB (e.g., 65 KiB of
   valid-JSON-prefix bytes followed by truncation).
2. Calls `dispatch()` against it.
3. Asserts the returned error contains the prefix `"E-RPC-002: message too large"`.
4. Asserts `errors.Is(err, io.ErrUnexpectedEOF)` via the wrapped chain.

The test MUST be symmetric with any existing `TestSbctlAuth_OversizedChallenge_*` test that
covers the `Authenticate()` path.

### Downstream Artifact Impacts

| Artifact | Required Change |
|----------|----------------|
| `cmd/sbctl/client.go` (~line 300-303) | Add `errors.Is(err, io.ErrUnexpectedEOF)` arm wrapping `E-RPC-002: message too large` |
| `cmd/sbctl/client_test.go` (or equivalent) | Add `TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002` |
| S-6.07 story spec (AC or implementation notes) | Add implementation note: dispatch() decode-error branch MUST wrap io.ErrUnexpectedEOF with E-RPC-002 per ADR-012 §6; cite Authenticate() parity and Ruling-14 |
| No BC change | E-RPC-002 is already defined in error-taxonomy.md; the fix applies the existing code to a missing branch |

### Ordering

The `dispatch()` fix and test are within the S-6.07 fix-burst scope. No dependency on any
other ruling. The story-writer adds the implementation note to S-6.07. No new follow-on stories
required.

---

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-01 | Initial: Rulings 1–2 (router_addr wire shape, authority model) |
| 1.1 | 2026-07-01 | Rulings 3–5 (PathTracker wiring P2, status=failed deferral, bootstrap fast-path fix) |
| 1.2 | 2026-07-01 | Rulings 6–7 (PathTracker wiring P3 — defer to S-BL; AC-003 defense-in-depth) |
| 1.3 | 2026-07-01 | Rulings 8–9 (genesis carve-out for DiD role check + SeedSVTNWithoutBootstrapKeyForTest relocation; Active=false status mapping) |
| 1.4 | 2026-07-01 | Ruling 10 ("production package" definition; InsertRawSVTN runtime guard + CreatedAt parity + DuplicateName test; SeedSVTNWithoutBootstrapKey rename + postcondition assertions; F-L2-A1-02/03/04 closure) |
| 1.5 | 2026-07-01 | F-P5L3R-09 (Pass-6 L3) correction: Ruling-9 downstream impact table — BC-2.06.003 target version corrected from v1.11→v1.12 to v1.11→v1.13 at two sites; v1.12 was an interim hop, actual delivered version is v1.13 |
| 1.6 | 2026-07-01 | Ruling-11: mgmt-layer wire envelope contract formalized; S-6.07 AC-003/AC-004/AC-005 wire-envelope amendments; E-ADM-009 message-format fix (F-Lens1-02); Ruling-6 pre-emption on pathTrackerSource.mu accepted (F-L1-01 closed no-action); POL-002 story-index-row-sync policy flagged for spec-steward |
| 1.8 | 2026-07-01 | Ruling-12 §1 amended: E-INT-999 added to enumerated handler-code list as catch-all default-arm sentinel for `mapAdminError`. Ruling-12 §7 (new): process policy — introducing a new handler-code family requires (a) error-taxonomy row, (b) §Universality anchor story amendment, (c) §1 enumeration update — all in the same fix-burst. error-taxonomy.md bumped v4.0 → v4.1 (E-INT-999 row added to INT section). |
| 1.7 | 2026-07-01 | Ruling-12: wire-envelope universality (E-ADM-009/E-SVTN-001/E-CFG-001/E-INT-001 all E-RPC-011-wrapped); canonical role-label unified to "unregistered"; BC-2.07.001 genesis-path vector added; POL-002 schema alignment to POL-001 canonical fields; BC-2.07.001 modified-list chronological reorder; S-BL.POLICY-SCHEMA-VALIDATOR stub flagged |
| 1.9 | 2026-07-01 | §8 (new): BC-2.06.003 v1.14 EC-008 ratified (empty-paths quality:'pending'); anchored in S-W5.04 body via §Edge Cases; no story-scope change beyond spec-tightening |
| 1.10 | 2026-07-01 | Ruling-13 (§9): F-P12L1-02 ruled BY DESIGN — E-RPC-001 is the intentional sbctl dispatch bucket; operator discrimination is by message prefix per AC-004/AC-005; §Wire Envelope Contract clarification note added to S-6.07. Ruling-14 (§10): F-P12L1-01 ruled IN SCOPE — dispatch() response decode MUST add errors.Is(io.ErrUnexpectedEOF) arm wrapping E-RPC-002 per ADR-012 §6 Authenticate parity; test TestSbctlAdmin_OversizedRPCResponse_ReturnsE_RPC_002 required. |
