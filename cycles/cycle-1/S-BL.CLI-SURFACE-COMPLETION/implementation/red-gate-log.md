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

## Status

Red Gate COMPLETE. Green COMPLETE @ `409457d` (25/25 target tests, full
suite green, race-clean, vet/fmt/lint clean). Next: step-4.5
implementation-diff spec-adversarial convergence (BC-5.39.001/BC-5.39.002,
diff range `4c276d9..409457d`).
