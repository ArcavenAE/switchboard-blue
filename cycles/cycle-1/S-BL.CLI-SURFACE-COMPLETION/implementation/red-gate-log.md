# S-BL.CLI-SURFACE-COMPLETION — Red Gate Log

Story spec: `stories/S-BL.CLI-SURFACE-COMPLETION.md` (v2.6, converged v2.5 per
`.factory/cycles/cycle-1/S-BL.CLI-SURFACE-COMPLETION/adversary-convergence-state.json`).
Delivery worktree: `.worktrees/S-BL.CLI-SURFACE-COMPLETION` @
`feature/S-BL.CLI-SURFACE-COMPLETION`, branched from `develop` @ `4c276d9`.

Red Gate discipline per BC-5.38.001: stubs compile and pass registration
tests, then a failing test suite is written against those stubs before any
real implementation lands.

## Step 1 — Stubs (stub-architect)

**Commit:** `5d54af4` — `chore(cli-surface): S-BL.CLI-SURFACE-COMPLETION stubs — dispatch + wire skeleton (Red Gate)`

**Files:** 6 new, 5 modified (16 files changed, 386 insertions, 25 deletions).

New:
- `cmd/sbctl/paths_ping.go` — `runPathsPing` (AC-001..AC-004)
- `cmd/sbctl/router_reload.go` — `runRouterReload` (AC-015)
- `cmd/sbctl/router_drain.go` — `runRouterDrain` (AC-016)
- `cmd/sbctl/svtn.go` — `runSvtn`, `runSvtnStatus`, `runSvtnDestroyShim` (AC-005..AC-010)
- `cmd/switchboard/router_control_wire.go` — server-side control wiring
- `internal/mgmt/register_ping.go` — `RegisterPingHandler`

Modified:
- `cmd/sbctl/main.go`, `cmd/switchboard/main.go` — dispatch arms (paths
  `ping` sub-verb, router `reload`/`drain` sub-verbs, new top-level `svtn`
  case)
- `cmd/switchboard/metrics_wire.go`, `cmd/switchboard/admin_handlers.go`,
  `cmd/switchboard/mgmt_wire.go` — `runRouter` signature widened; 13 call
  sites across 6 files updated to match (per the pass-2 spec-adversarial
  enumeration)

**Stub discipline:** all new behavior bodies are `panic("not implemented: ...")`.
Registration outer-functions are real (not stubbed) so existing daemon tests
keep passing — same shape as the S-7.04-FU-PE-CONNECTOR precedent (`9d184db`).
Dispatch arms in `main.go` are real trivial routing, not stubs; only the
functions they call panic.

**Gates:** `go build` / `go vet` clean, `gofumpt` clean, `just lint` 0 issues.

**Two spec-intended pre-existing test regressions** surfaced at this step
(handler count 6→7; an orphan-svtn premise) — both resolved by step 2's
spec-driven test updates, not treated as stub-step defects.

## Step 2 — Failing tests (test-writer)

**Commits:** `8113b66` — `test(cli-surface): failing test suite for S-BL.CLI-SURFACE-COMPLETION — Red Gate`;
`5bdad28` (amendment) — `test(cli-surface): assert E-CFG-001 token in AC-008 missing-name subtest`.

**Files:** 4 new test files, 2 extended, 2 stale-premise updates, 1
fragile-coincidence hardening.

New: `cmd/sbctl/paths_ping_test.go`, `cmd/sbctl/router_control_test.go`,
`cmd/sbctl/svtn_test.go`, `cmd/switchboard/router_control_wire_test.go`.
Extended: `cmd/sbctl/main_test.go`, `cmd/switchboard/admin_handlers_test.go`.
Updated (stale premise / hardening): `cmd/sbctl/phase5_pass8_test.go`,
`cmd/switchboard/mgmt_wire_test.go`.

**Coverage:** 25 top-level failing test functions / 42 subtests, all
mapped to AC targets. All unrelated suites remain green. Race-clean.
`go build` / `go vet` / lint clean.

**ORCHESTRATOR-VERIFIED independently:** 25 failing tests confirmed at
`5bdad28`.

## Engineering note — cross-goroutine stub-panic hazard

Tests invoking a stub across a goroutine boundary (e.g. a subprocess-style
daemon dispatch) use subprocess re-exec (`runProductionMain` /
`runRouterControlScenario`) rather than a same-process `recover()`, since a
panic crossing goroutines in-process would crash the whole test binary and
take unrelated tests down with it. Same-goroutine, no-server unit-level
stub calls (e.g. `routerReloadRPCHandler`'s `configPath == ""` guard,
`admin.svtn.status`'s handler closure) use a local `recover()` wrapper
instead, since that is safe when the panic and the assertion happen on the
same goroutine. Anti-panic-trace assertions were added throughout to
prevent a stub panic from being misread as a false pass.

## Token audit

Every spec'd error-code token this story's own handlers emit has an
explicit assertion: `E-CFG-001`, `E-NET-001`, `E-ADM-010`, `E-CFG-004`,
`E-SVTN-003`, `E-ADM-009`, `E-RPC-010`. `E-CFG-008` and `E-CFG-011` appear
in the story only as cross-reference precedent shapes for other
`error-taxonomy.md` rows, not codes this story's handlers emit — correctly
left unasserted.

## Ambiguity adjudications (flagged for the step-4.5 adversary)

1. **AC-008/AC-010 error prose is unpinned by design.** Tests assert
   semantic substrings plus the spec'd `E-CFG-001` token rather than a
   byte-exact literal.
2. **`--router` override is unique to `paths ping`**, per BC-2.06.004 PC-1
   / the interface-definitions §77 adjudication; `reload`/`drain` use
   `--target` instead. This asymmetry is intentional, not an oversight.
3. **AC-012 PC-3 severance is tested via a defensible wire-level proxy**
   (drain fires, then a follow-up RPC confirms the server was not
   corrupted) rather than a full `runRouter`-daemon severance test, which
   is blocked on an ephemeral-key test seam that does not yet exist. The
   step-4.5 adversary should scrutinize this proxy specifically — it is
   the one test in this suite standing in for behavior it cannot fully
   exercise yet.

## Green Phase

**Implementation commits (implementer):**
- `99012b7` — AC-001..004 `paths.ping` (client + server)
- `84e0561` — AC-005..010 `admin.svtn.status` + `svtn` dispatch
- `70ea1f1` — AC-011..016 `router reload`/`router drain`

**Test-conflict fix (test-writer):** `409457d`.

**Test-suite conflict adjudicated mid-green:** `runTierOneAuthScenario`
passed `configPath=""` and collided with AC-011 PC-3's guard test on the
same handler. The implementer STOPPED per TDD discipline rather than
weaken the test. Orchestrator adjudicated fix-the-test: AC-014's own
premise is a production-reachable router, and the story's defense-in-depth
chain (`runRouter`'s `cfg==nil` guard + `main.go`'s `"router"` case)
guarantees `configPath != ""` for every production-reachable router — the
zero-value `configPath` in `runTierOneAuthScenario` was the test's premise
error, not a spec gap. Test-writer fixed at `409457d`, mirroring the
`reload_bridges_sighup` scenario's existing non-empty-`configPath`
pattern. Recorded here as a TDD-walls-working instance: the wall held,
the fix landed on the correct side of it.

**Latent bug found and fixed in scope:** `cmd/sbctl`'s `main()` never
called `os.Exit(0)` on success — invisible until this story's first
happy-path subprocess tests exercised it. One-line fix folded into
`99012b7`.

**Design note flagged for the step-4.5 adversary:** `paths ping` and
`admin.svtn.status` emit the JSON envelope unconditionally — AC-001 PC-4
specifies the JSON report with no `--json` gating, and the tests enforce
that. Whether this envelope-vs-bare-object convention is consistent with
sibling commands (which may gate JSON output behind a flag) is a step-4.5
check item, not resolved here.

Implementer notes from the spec-adversarial arc (N-CS-SP7-01 usage-hint
refresh, N-CS-SP9-01 help-string refresh) confirmed done in this pass.

**Final green evidence:** `go test -count=1 ./...` — all 25 packages
`ok` (orchestrator-verified), race-clean, `go vet` / `gofumpt` / lint all
clean.

## Step-4.5 pass 1

**Adversary:** adv-cs-i1, 2026-07-13. **Verdict:** HAS_FINDINGS, streak
0/3. **Diff reviewed:** `4c276d9..409457d`. Dispatch tuple — develop
`4c276d9`, feature `409457db843b824a88236978b8f3592b687f34a6`, factory
`aab243c` — POL-005 verified PASS by the adversary.

**F-CS-I1-001 (MED, coverage):** AC-004 had zero tests. The story-named
`TestWireMetricsHandlers_RegistersPingOnEveryMode` and
`TestPingHandler_EmptyArgsIn_PongOut_ZeroPathTrackerInteraction` were
never written in the Red suite — the Red inventory was complete against
itself but incomplete against the 16 ACs. Implementation was correct by
inspection; the gap was test coverage, not behavior.
Remediated at `1b0e010` (test-writer): `metrics_wire_test.go` +271,
`internal/mgmt/register_ping_test.go` +135 (new), 4 subtests including a
`go/ast` four-call-site static check and an `SVTNMetrics` tripwire spy.
Non-vacuity mutation-verified for every assertion (each mutation reverted
after confirming it failed as expected). Orchestrator-verified both tests
pass by name at `-count=1`.

**F-CS-I1-002 (LOW, convention):** `cmd/sbctl/paths_ping.go`'s
`--router`-requires-value branch stamped `E-CFG-010` (key-load code,
exit-1 taxonomy row) instead of `E-CFG-001` (client-side usage error,
exit 2) — wrong code, wrong exit-class pairing, inconsistent with the
sibling flag-validation branch in the same file.
Remediated at `4dad99e` (implementer): both occurrences corrected to
`E-CFG-001` (`"E-CFG-001: paths ping: --router requires a value"`); the
genuine key-load `E-CFG-010` use (`loadEd25519Key`) left untouched — that
one really is a key-load failure. Matches the Ruling 2 Addendum
client-side precedent. Orchestrator-verified.

**Sanctioned, no action:** N-CS-I1-01 (JSON-envelope divergence, same
item as the Green-phase design note), N-CS-I1-02 (AC prose
`--router`/`--target` asymmetry, cosmetic, already adjudicated
intentional), N-CS-I1-03 (no-dispatch assertion immaterial), N-CS-I1-04
(`os.Exit(0)` defer-skip harmless).

**Clean lenses:** existence-oracle byte-identical + no timing leak;
signal-coalescing semantics correct + race-clean; SIGTERM parity; test
honesty; all 3 Green-phase adjudications honored; the `409457d` test fix
verified exactly as narrow as described.

**Post-remediation:** feature head `1b0e010`, full suite 25 packages `ok`
at `-count=1`, race-clean (`cmd/switchboard`, `internal/mgmt`), lint 0
issues, gofumpt clean.

## Step-4.5 pass 2

**Adversary:** adv-cs-i2, 2026-07-13. **Verdict:** HAS_FINDINGS, streak
0/3. **Diff reviewed:** `4c276d9..1b0e010`. Dispatch tuple — develop
`4c276d935b089026fac4fa796612352374bb880f`, feature
`1b0e01048486cf07eed3fe728a4bfc1af1112a8a`, factory
`c37f5e17b3a852d133ecc20e1fbdc443927cdb24` — POL-005 verified PASS by the
adversary.

**F-CS-I2-001 (LOW, spec-governance, architect-owned — zero
implementation-code defects this pass):** Forward Obligation (b) was
unfulfilled at delivery. The ARCH-INDEX SS-06 row lacked `internal/mgmt`
despite BC-2.06.004's `architecture_module` citing it.
Remediated by the architect: ARCH-INDEX v1.9 → v1.10 (SS-06 gains
`internal/mgmt` (Wave 7), SS-07-style annotation).

**Nitpick dispositions:** N-CS-I2-01 TAKEN (story AC-015/016 `--router`
exemplars were wrong vs interface-definitions §82/83 — fixed in story
v2.7); N-CS-I2-02 SANCTIONED (`paths ping`/`svtn status` hardcode the
JSON envelope — deliberate design, no AC/spec mandates `--json` handling);
N-CS-I2-03 SANCTIONED (delivered test files retain Red-Gate-era header
comments — documentation-only; branch-freeze policy means no code churn
between impl passes; cleanup candidate at PR stage).

**Clean lenses:** 16/16 AC compliance (wire contracts byte-exact vs
interface-definitions §420-423; E-CFG-004 Variant 3 byte-exact 3-way
match; 14 `runRouter` call sites verified); test honesty STRONG
(subprocess isolation, AST four-call-site proof, tripwire spy, no
vacuous tests); taxonomy conformance PASS; security PASS (svtn existence
oracle CLOSED — byte-identical E-ADM-009 test); concurrency PASS
(coalescing correct); all 3 adjudications honored; POL-001/004/005
clean.

**Remediation burst (all factory-side; feature branch FROZEN at
`1b0e010` — pass 3 reviews identical code):**
- Architect: ARCH-INDEX v1.10 + BC-2.06.004 v1.3 → v1.4 (`VP-TBD-PING-A`/
  `VP-TBD-PING-B` → `VP-078`/`VP-079` minted per the VP-061/062
  precedent; new files `specs/verification-properties/VP-078.md` +
  `VP-079.md`; VP-INDEX v2.39 → v2.40) → **FO(d) DISCHARGED**.
- Product owner: `capabilities.md` v1.0 → v1.1 (CAP-029 minted — "On-demand
  reachability and round-trip-latency probe via sbctl", quality-observability,
  no PRD FR anchor, introduced by Ruling 1) + BC-2.06.004 v1.4 → v1.5
  (capability re-anchor CAP-022 → CAP-029, input-hash `9d4a662`) +
  BC-INDEX v3.4 → v3.5 → **FO(a) DISCHARGED** (architect recommendation
  + PO concurrence).
- Story-writer: story v2.6 → v2.7 (VP propagation, Forward Obligations
  table all four rows DISCHARGED, N-CS-I2-01 exemplar fix, input-hash
  `cbf07f7` → `d95ecfe`) + STORY-INDEX v4.92 → v4.93 (row 147 v2.7 + FO
  bracket).
- **Outcome: all four Forward Obligations now DISCHARGED.**

## Step-4.5 pass 3

**Adversary:** adv-cs-i3, 2026-07-13. **Verdict:** HAS_FINDINGS, streak
0/3. **Diff reviewed:** `4c276d9..1b0e010` — feature branch FROZEN,
identical code to pass 2. Dispatch tuple — develop
`4c276d935b089026fac4fa796612352374bb880f`, feature
`1b0e01048486cf07eed3fe728a4bfc1af1112a8a`, factory
`f573a6e7a459d80710d5f256cc71a89959b47374` — POL-005 verified PASS by
the adversary across 9 artifacts (story v2.7, rulings v1.2, taxonomy
v4.9, interface-definitions v1.31, BC-2.06.004 v1.5, ARCH-INDEX v1.10,
capabilities v1.1, BC-INDEX v3.5, VP-INDEX v2.40 — all matched).

**F-CS-I3-001 (LOW, story-artifact completeness, story-writer-owned —
zero implementation-code defects this pass):** the File-Change List
omitted two touched files present in `git diff --stat` against
`develop`: `cmd/sbctl/main_test.go` (+42/-10, `TestSbctl_OrphanSubcommands`
re-pointed — `svtn` became a real subcommand per AC-010) and
`cmd/sbctl/phase5_pass8_test.go` (+9/-3, `TestPathsUnknownVerb`
exemplar swapped `ping` → `trace` per AC-001). Both are correct,
necessary existing-test accommodations forced by this story's own
scope — implementation was more complete than the list, not scope
creep.
Remediated by the story-writer: story v2.7 → v2.8 (two File-Change List
rows added); STORY-INDEX v4.93 → v4.94 (POL-002); input-hash verified
UNCHANGED `d95ecfe` (`compute-input-hash --check` exit 0 — no declared
input file changed).

**Nitpick dispositions:** N-CS-I3-01 SANCTIONED (always-JSON for
`paths ping`/`svtn status` — third independent fresh-context
derivation, ≈N-CS-I1-01/N-CS-I2-02, documented design, no AC mandate);
N-CS-I3-02 SANCTIONED (JSON error envelope vs plain usage-line on the
missing-flag path for the two always-JSON verbs — single-print
contract, token, exit 2 all correct, cross-command cosmetic only);
N-CS-I3-03 SANCTIONED for the freeze window (stale Red-Gate header
comments in 4 test files — second consecutive pass raising it, queued
as comment-hygiene cleanup at PR stage once convergence lifts the
freeze).

**Clean lenses:** 16/16 AC compliance (E-CFG-004 v2/v3 byte-exact,
§60/§82/§83 literals, exit-code mapping, Ruling 2 pattern); test
honesty CLEAN (RTT-reflects-injected-delay, byte-identical oracle
assert, tripwire spy + AST proof, drain third-arm nil-return parity,
anti-panic guards); taxonomy CLEAN; security CLEAN
(admission-before-lookup, no existence leak, destroy shim never
dials); concurrency CLEAN (buffered-1 drop-coalescing, two-writers-one-
reader safe, never-closed channel); spec-code drift CLEAN including
VP-078/VP-079 consistency; policies clean.

## Step-4.5 pass 4

**Adversary:** adv-cs-i4, 2026-07-13. **Verdict:** HAS_FINDINGS, streak
0/3. **Diff reviewed:** `4c276d9..1b0e010`. Dispatch tuple — develop
`4c276d935b089026fac4fa796612352374bb880f`, feature
`1b0e01048486cf07eed3fe728a4bfc1af1112a8a`, factory
`be00331ad419a0d3ed536d4baa53457000d334e2` — POL-005 verified PASS
across 10 artifacts (incl. story v2.8, STORY-INDEX v4.94 — all matched).

**F-CS-I4-001 (MED, code, implementer-owned):** `runPathsPing` and
`runSvtnStatus` hardcoded `useJSON = true` — they always emitted the
`{"ok":true,"data":...}` envelope, making `--json` a silent no-op. This
violated interface-definitions v1.31 §214, diverged from the house
pattern (`paths list`, `router status`, `list-keys`, and this very
story's own `reload`/`drain`), and meant AC-001 PC-4 and AC-005 PC-1's
bare literals were never produced at the top level.

**Escalation record — the strongest BC-5.39.001 evidence this story has
produced:** the same observation was raised as a NITPICK by three
consecutive fresh-context passes (N-CS-I1-01, N-CS-I2-02, N-CS-I3-01)
and orchestrator-sanctioned each time on the shared reasoning "no AC
mandates `--json`." Pass 4 located the governing §214 clause *outside*
the ACs — the prior three sanctions were superseded, the finding was
accepted, and the code freeze that had held since pass 1 was **LIFTED**.
This is the multi-pass fresh-context discipline working exactly as
designed: a systemic blind spot shared by three independent reviewers
was still caught, because the fourth reviewer looked somewhere the
first three hadn't.

Remediated TDD-shaped:
- **RED** `dfd51bc` (test-writer): 3 happy-path tests split into
  `default_bare_data`/`json_flag_envelope` subtests with shared
  helpers, verified FAILING against the always-envelope implementation;
  confirmed no AC mandates the envelope in default mode; AC-005 PC-1
  also specifies a bare literal.
- **GREEN** `100d28890eb7a07541aa9aa93be8339faa8b5e4d` (implementer):
  `useJSON` threaded through every `writeError`/`writeSuccess`/
  `connectAndRun` call in both functions; both `//nolint:unparam`
  annotations dropped; no new lint issues.

**F-CS-I4-002 (LOW, code, implementer-owned):** client-side missing-flag
paths double-emitted `writeError(true, "E-CFG-001", ...)` followed by
`reported(usageErrf(...))` — the canonical pattern per Ruling 2 Addendum
and the §110/§111 siblings is a bare `usageErrf`.
Fixed in `100d288`: bare `usageErrf`, message text byte-identical,
shape-agnostic missing-flag tests stayed green.

**Nitpick dispositions:** N-CS-I4-01 FIXED in `100d288`
(`wireRouterControlHandlers`' doc comment reworded out of stale future
tense — the handler bodies are fully implemented). N-CS-I4-02 FIXED in
`dfd51bc` (4 test-file Red-Gate-era headers refreshed). The stale-comment
family raised across passes 2, 3, and 4 is now **CLOSED** — no longer
deferred to PR stage.

**Clean lenses:** security CLEAN (admission-before-lookup, byte-identical
E-ADM-009 re-verified, destroy shim never dials, fail-closed guards);
concurrency CLEAN (SIGHUP-synthesis coalescing parity, three-arm select
shared shutdown label); taxonomy CLEAN otherwise; test honesty CLEAN
including confirmation that the pass-3 accommodations were honest and
tightening; File-Change List 25/25 complete vs diff; POL-001/002/004
clean, POL-005 executed.

**Post-remediation:** feature head `100d28890eb7a07541aa9aa93be8339faa8b5e4d`
(`1b0e010` → `dfd51bc` red → `100d288` green); freeze LIFTED at pass 4;
full suite 25 packages `ok` at `-count=1` (orchestrator-run); race-clean
`cmd/sbctl`; lint 0 issues; gofumpt clean. Output contract now: default
= bare data object (AC-001 PC-4 / AC-005 PC-1 literals top-level);
`--json` = envelope; missing-flag = single plain `usageErrf` line, exit
2.

## Step-4.5 pass 5

**Adversary:** adv-cs-i5, 2026-07-13. **Verdict:** NITPICK_ONLY, zero
findings — **first clean verdict, streak starts at 1/3.** **Diff
reviewed:** `4c276d9..100d288`. Dispatch tuple — develop
`4c276d935b089026fac4fa796612352374bb880f`, feature
`100d28890eb7a07541aa9aa93be8339faa8b5e4d`, factory
`85eb5156600efe9b4e0a4c3e1f30017a9f60c550` — POL-005 verified PASS
across 10 artifacts, all matched.

**Infrastructure note:** the pass was interrupted mid-response by an
API connection failure (`idle_notification` reason "Connection closed
mid-response"); it resumed via a mailbox nudge with context intact and
delivered a complete report. No review-integrity impact — the
fresh-context wall was unaffected.

**N-CS-I5-01, SANCTIONED per the adversary's own recommendation:**
`runPathsPing` discards the `paths.ping` response body and never checks
`pong == true`. Defensible: BC-2.06.004 frames ping as a
`ping(8)`-style reachability-plus-latency probe, the handler always
returns `{"pong":true}`, and dispatch already rejects `ok:false`. The
adversary's own recommendation was to leave it as-is.

**Clean lenses (all 7 PASS):** 16/16 AC compliance including §214's
dual-mode output contract; test honesty (dual default/`--json`
subtests, `wantNoPanic` guards, field-count==1 guard, byte-identical
no-leak assert, tripwire spy, AST four-call-site proof); taxonomy
conformance (E-CFG-004 v3 exact, E-CFG-001 bare `usageErrf` per Ruling
2 Addendum); security (oracle closed, no secret leakage, no injection);
concurrency (buffered-1 drop-coalescing, nil-channel PE call sites
safe, no shared mutable state); spec-code drift none (13+1 call sites,
accommodations faithful, VP-078/079 consistent); policy clean. All 4
orchestrator adjudications reverified honored; all 4 Forward
Obligations reconfirmed DISCHARGED.

**Post-pass:** feature head `100d288`, unchanged — no remediation
needed.

## Step-4.5 pass 6

**Adversary:** adv-cs-i6, 2026-07-13. **Verdict:** NITPICK_ONLY, zero
findings — **second consecutive clean verdict, streak 2/3.** **Diff
reviewed:** `4c276d9..100d288`. Dispatch tuple — develop
`4c276d935b089026fac4fa796612352374bb880f`, feature
`100d28890eb7a07541aa9aa93be8339faa8b5e4d`, factory
`50d09a08e2b559d97df9d8281a20ae7617600c4e` — POL-005 verified PASS
across 10 artifacts, all matched. The adversary ran gates read-only in
addition to inspection: build/vet clean, targeted test packages ok,
race clean on reload/drain/router/ping/svtn-status tests, golangci-lint
0 issues.

**N-CS-I6-01, FIXED post-pass in hygiene commit
`ef3e5c58411902d0117e9948815895490b8fd9dd`** (test-writer): two inner
doc comments still claimed "currently panics unconditionally,"
contradicting the pass-4-corrected file headers; one same-class
comment turned up in the sweep. Orchestrator-verified comment-only:
19+/19-, zero non-comment content lines, mechanical grep both sides.

**N-CS-I6-02, SANCTIONED:** the unused `sio` param on
`runSvtnDestroyShim` — nolint-annotated with rationale; dropping it
would touch production and test call sites, a larger change surface
than the nitpick warrants at streak 2/3.

**N-CS-I6-03, SANCTIONED:** AC-002's no-dispatch-on-auth-fail is proven
structurally — dispatch is unreachable before `Authenticate`; a
symmetry tripwire is optional.

**N-CS-I6-04, SANCTIONED, not touched:** two `t.Errorf` format strings
with "does not yet bridge/send" phrasing, found in the same hygiene
sweep — failure-message text (displays only on regression, where the
phrasing is semantically accurate), not documentation claims. Left
untouched to honor zero-assertion-change discipline.

**Clean lenses:** 16/16 ACs including §214 dual-mode; test honesty (AST
four-mode proof plus live dispatch, byte-identical oracle assert);
taxonomy (no E-RPC-001 leak into `internal/mgmt`); security
(E-SVTN-003 reachable only by entitled callers); concurrency
(race-detector clean); drift none (VP-078/079 consistent); POL-001/002
satisfied, POL-004 n/a, POL-005 executed. All 5 adjudications honored;
completeness grep 25/25.

**Post-pass:** feature head after hygiene is
`ef3e5c58411902d0117e9948815895490b8fd9dd` (`100d288` +
comment-only `ef3e5c5`). Streak note: verdicts drive the clock —
comment-only hygiene between clean passes does not reset (spec-phase
N-CS-SP8-01 precedent).

## Status

Red Gate COMPLETE. Green COMPLETE @ `409457d`. Step-4.5 pass 1
HAS_FINDINGS, remediated @ `1b0e010`. Step-4.5 pass 2 HAS_FINDINGS —
spec-governance only, zero code defects, remediated factory-side (all
four Forward Obligations DISCHARGED). Step-4.5 pass 3 HAS_FINDINGS —
story File-Change List completeness only, zero code defects, remediated
@ story v2.8. Step-4.5 pass 4 HAS_FINDINGS — F-CS-I4-001 (MED, §214
`--json` contract) + F-CS-I4-002 (LOW, `usageErrf` shape), remediated
TDD-shaped @ `100d288`; code freeze lifted. Step-4.5 pass 5
NITPICK_ONLY — first clean verdict, streak 1/3. Step-4.5 pass 6
NITPICK_ONLY — second consecutive clean verdict, streak 2/3;
comment-only hygiene at `ef3e5c5` did not reset the streak. Next:
step-4.5 pass 7 — **CONVERGENCE PASS** (BC-5.39.001/BC-5.39.002, diff
range `4c276d9..ef3e5c5`).
