## Fix: VP-042 testenv-integrated keystroke-echo benchmark

**Finding:** VP-042 residual investigation (formal-verifier, Phase 6 VP sweep follow-on).
See `.factory/specs/verification-properties/VP-042.md` v1.3 lifecycle note and
STATE.md `awaiting` row ("VP-042 testenv residual").
**Phase:** 6 (formal hardening residual)
**Severity:** LOW (test/bench-only change; no product code touched)

Migrates the VP-042 proof-harness benchmark from an inline echo-sink loopback to
the canonical `testenv.NewLoopback` API now that S-BL.TESTENV has shipped, and
documents — via commit message and package comment, not code — the finding that
this migration does not close the VP-042 verification gap: `testenv.NewLoopback`
still bypasses tick scheduling, ARQ, and multipath.

---

## Blast Radius

**1. Operator-visible surfaces touched:**

None. This PR only adds a new `//go:build integration`-tagged benchmark file
(`internal/bench/keystroke_echo_testenv_bench_test.go`) and edits doc comments in
an existing benchmark file. No CLI flags, no `--help`/`--version` output, no config
schema, no wire frame layout, no error taxonomy, no log format, no
`docs/getting-started.md` changes. Neither benchmark file is wired into any
required CI check (per ADR-007, diagnostic-only), so nothing here reaches a
released binary or a default `go build`/`go test ./...` run.

**2. Silent-failure risk:**

The real risk in this PR is documentation-shaped, not code-shaped: if the doc
comments overclaimed that the testenv-integrated benchmark constitutes full-stack
VP-042 evidence, a future reader or automation could flip
`VP-042.verification_lock` incorrectly. Both bench files' comments were explicitly
reviewed for this (see PR review F1, fixed in the second commit) — every reference
now says the lock stays deferred and names the concrete follow-on
(S-BL.LOOPBACK-FULLSTACK) rather than implying S-BL.TESTENV alone closes the gap.
Separately: because this benchmark is `integration`-tagged and not CI-gated, a
future regression in `internal/testenv`'s loopback delivery path (e.g. it becomes
slower or starts blocking) would not be caught automatically — someone has to run
`go test -tags integration -bench=BenchmarkKeystrokeToEcho_P99 ./internal/bench/`
by hand. That's a pre-existing property of ADR-007's diagnostic-benchmark design,
not something this PR introduces or worsens.

**3. Smoke gate touched:**

No. `test/smoke/invariants.sh` covers operator-visible CLI/binary behavior; this
PR has none of that surface (see prompt 1).

---

## Architecture Changes

None. This PR touches only `internal/bench/*_test.go` (a test/benchmark package).
No production code, no new dependencies, no new components. The finding documented
in this PR (testenv's `LoopbackConfig` is dead, `testenv` imports neither
`halfchannel` nor `arq` nor `multipath`) describes the *architecture of an existing
package* (`internal/testenv`); it does not change it. Any wiring change to route
`testenv.NewLoopback` through `halfchannel.Tick()` + `arq` + `multipath` is
out of scope here and belongs to the follow-on story (see Story Dependencies).

---

## Story Dependencies

| Story | Relationship | Status |
|-------|--------------|--------|
| S-BL.TESTENV | upstream dependency — delivered `testenv.NewLoopback` this PR migrates onto | DELIVERED (PR #110) |
| S-BL.BENCH | prior partial VP-042 adoption (lower-bound loopback benchmark, PR #109 cd67394) | DELIVERED, superseded-in-part by this PR's package comment update |
| S-BL.LOOPBACK-FULLSTACK | downstream — required before `VP-042.verification_lock` can flip; must route `testenv`'s loopback through `halfchannel.Tick()` + `internal/arq` + `internal/multipath` (ARCH-08 import-set implications) | NOT YET AUTHORED — ARCH-08 re-registration in progress per architect; this PR's finding is what makes that story's scope legible |

This PR does not block or unblock any other in-flight story; it closes the
"migrate to canonical testenv API" half of the VP-042 residual named in STATE.md
and leaves the "full-stack lock" half explicitly open.

---

## Spec Traceability

| Source | Item | This PR |
|--------|------|---------|
| Verification Property | VP-042 — Keystroke-to-echo p99 ≤ 100ms (NFR-001) | Adds testenv-integrated measurement; `verification_lock` stays `false` (unchanged) |
| Behavioral Contract | BC-2.01.001 — timeslice clock fires on every tick regardless of data availability | Benchmark does NOT exercise this BC yet — `testenv` doesn't drive `halfchannel.Tick()` (documented finding) |
| Behavioral Contract | BC-2.02.001 — duplicate-and-race: same frame sent on two fastest paths simultaneously | Benchmark does NOT exercise this BC yet — `testenv` doesn't import `internal/multipath` (documented finding) |
| Test | `BenchmarkKeystrokeToEcho_P99` (`internal/bench/keystroke_echo_testenv_bench_test.go`, `integration`-tagged) | NEW |
| Test | `BenchmarkKeystrokeEcho_P99` (`internal/bench/keystroke_echo_bench_test.go`) | comment-only update, behavior unchanged |

---

## Test Evidence

| Check | Result |
|-------|--------|
| `just fmt` (gofumpt) | clean, no diff |
| `just lint` (golangci-lint, default tags) | 0 issues |
| `golangci-lint run --build-tags integration ./...` | 0 issues |
| `just test` (`go test ./...`) | 25 packages, all `ok` |
| `go build -tags integration ./...` | clean |
| `go test -tags integration -race -run '^$' -bench=BenchmarkKeystrokeToEcho_P99 -benchtime=1x -count=1 ./internal/bench/` | PASS, p99 ≈ 0.1095ms (single-run race-detector smoke) |
| Full 500-sample benchmark (author-reported, re-verified shape) | p99 0.086–0.132ms across runs, statistically equivalent to the existing loopback lower bound — see commit message for the finding this measurement supports |

No coverage/mutation/holdout metrics apply — this is a benchmark addition to an
already-tested package, not new production logic under TDD.

### New Tests (This PR)

| Test | Result |
|------|--------|
| `BenchmarkKeystrokeToEcho_P99` (`integration`-tagged) | PASS (see above) |

---

## Demo Evidence

Not applicable. Transparent fix per fix-pr-delivery criteria: this change adds a
benchmark and updates comments. There is no user-observable behavior, CLI output,
API response, or error message change to demonstrate.

---

## Non-goals / explicit scope boundary

This PR does not make `testenv.LoopbackConfig` live, does not add
`halfchannel`/`arq`/`multipath` wiring to `testenv`, and does not flip
`VP-042.verification_lock`. That work is out of scope here and belongs to
S-BL.LOOPBACK-FULLSTACK.

---

## Pre-Merge Checklist

- [x] All CI status checks passing (fmt, lint default + integration tags, full test suite, integration build — re-verified locally before push; CI re-verifies on the PR)
- [x] No production code changed — coverage/mutation deltas not applicable
- [x] No security-relevant change (test/bench-only) — security review not required per fix-pr-delivery conditional criteria, but available on request
- [x] Rollback is trivial: `git revert` — no schema, no config, no runtime behavior change
- [ ] Feature flag — not applicable
- [ ] Human review — per fix-pr-delivery autonomy; PR-reviewer + triage convergence gates this PR
