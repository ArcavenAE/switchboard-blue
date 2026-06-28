# PR Review — #28 feat/S-6.01-config-validation (Cycle 2)

**Verdict: APPROVE**

Independent fresh-eyes review. Cycle-1 blocking finding (demo evidence gitignored
/ absent from diff) is **resolved**: `docs/demo-evidence/S-6.01/` is now committed
at `07a4b00` and present in the diff (12 files: evidence-report.md + 11 transcripts).

I verified the PR on its own merits — diff, description, and test evidence — and
also ran the test suite, race detector, `go vet`, and `golangci-lint` against the
changed packages. All green.

---

## What I verified

| Check | Result |
|-------|--------|
| `go test ./internal/config/... ./cmd/switchboard/...` | PASS |
| `go test -race -count=1` (both packages) | PASS (config 2.0s, cmd 1.7s) |
| `go vet` (both packages) | clean |
| `golangci-lint run` (both packages) | 0 issues |
| All 9 ACs traced to proving tests | yes (AC-001..AC-009) |
| Demo evidence in diff | yes — evidence-report.md + 11 AC transcripts |
| Diff coherence (all changes relate to S-6.01) | yes |
| Description accuracy (body vs diff) | accurate |
| Go rules (no testify, table-driven, %w, UTC, no init, error checks) | conform |

### Checklist walkthrough

1. **Diff Coherence** — All 22 changed files relate to S-6.01: new `internal/config`
   package, `cmd/switchboard` wiring (`--config` flag, `tickIntervalFor`,
   `newHalfChannel` seam), tests, testdata fixtures, demo evidence, and the
   `gopkg.in/yaml.v3` dependency addition in go.mod/go.sum. No unrelated changes.
2. **Description Accuracy** — PR body matches the diff. The corrected items from
   cycle 1 are all present and accurate: evidence path committed (SHA `07a4b00`),
   `stripControlChars`/`sanitizeAddrForError` naming matches config.go, ADR states
   "1 MiB" (`1 << 20`) matching `maxConfigFileSize`. SEC-001/SEC-002 deferred LOW
   findings documented.
3. **Test Coverage** — Changed lines are covered. The cmd wiring (`run` --config
   branch, `tickIntervalFor`, `newHalfChannel` seam) is exercised by
   `TestRouterStartup_ExitsWithActionableError`, `TestBC_2_09_003_*`,
   `TestConfigTickIntervalApplied`, and `TestBC_2_09_003_TickIntervalWiredToHalfChannel`.
   The seam test closes the regression gap where a hardcoded default would leave
   `TestConfigTickIntervalApplied` green — good defensive testing.
4. **Demo Evidence** — evidence-report.md present, per-AC transcripts for all 9 ACs,
   plus a 6-scenario CLI error-surface demo with real binary invocations and exit
   codes (success path = Demo 6, error paths = Demos 1–5). See observation below
   re: medium.
5. **Commit Quality** — Conventional format, story-scoped, clear messages, signed.
   Red→green TDD progression is visible in the history.
6. **Diff Size** — 3727 insertions, but ~2118 lines are config_test.go and ~493 are
   cmd test additions; production code is modest (config.go 418, small cmd deltas).
   Test-heavy is expected for strict-TDD. Reasonable.
7. **Missing Changes** — Spec scope honored. AC-009 correctly applies only
   `tick_interval`; `listen_addr`/`drain_timeout`/`upstream_routers`/
   `keepalive_interval` application is deferred to named owning stories (S-BL.NI,
   S-7.04) per spec SP-004/SP-005, while validation of all fields is enforced now.
8. **Dependency Status** — `depends_on: [S-1.01]` (internal/frame); the package
   builds and tests pass against develop, so the dependency is satisfied.

---

## Findings

### MINOR — Demo evidence is `.txt` transcripts, not animated recordings

| Field | Value |
|-------|-------|
| Severity | minor (non-blocking) |
| Category | coverage |
| Location | `docs/demo-evidence/S-6.01/*.txt` |

The evidence is text transcripts of real binary invocations rather than `.gif`/`.webm`
recordings. For a pure-core config-validation / CLI error-surface story (no TUI/UI
surface, deterministic stderr + exit-code output), text transcripts capturing the
actual `$ sb access --config <file>` invocation, the emitted `E-CFG-*` line, and the
exit code are the appropriate evidence medium — an animated recording of a one-shot
CLI error would add no information. I am classifying this as a non-blocking observation
rather than a blocker. The transcripts cover both success (Demo 6) and error
(Demos 1–5) paths, satisfying the intent of the demo-evidence requirement.

Note: Demo 6 (valid config) notes PTY access is sandbox-restricted, so the daemon
reaches `runAccess` and fails at PTY connect rather than config validation — the
absence of any `E-CFG-*` error correctly proves validation passed. This is a sound
way to demonstrate the boundary given the sandbox.

### NITPICK — Two error-message shapes for field validation

| Field | Value |
|-------|-------|
| Severity | nitpick |
| Category | coherence |
| Location | `internal/config/config.go` — `ValidationError.Error()` vs inline addr/duration `fmt.Sprintf` |

`tick_interval` and missing-field errors are produced via `ValidationError.Error()`
(`"config error: <field>: value <v> <problem>. Fix: <suggestion>"`), while
`listen_addr`, `upstream_routers[N].addr`, `drain_timeout`, and `keepalive_interval`
errors are built with inline `fmt.Sprintf` literals to match the exact canonical
strings the spec mandates (AC-005..AC-008). The result is correct and every assertion
passes, but the `ValidationError` struct ends up used for only two of the six field
errors. A future cleanup could route all six through the struct (or drop the struct in
favor of plain strings) for one consistent construction path. Not worth changing now —
the canonical-string requirements make the inline form the pragmatic choice.

### Observation — deferred LOW security findings are reasonable

SEC-001 (config path echoed unsanitized in E-CFG-004 — local-access-gated) and
SEC-002 (no explicit cap on `upstream_routers` slice length, implicitly bounded by the
1 MiB file guard) are documented as deferred tech debt. Both are genuinely low-risk
and correctly classified; deferring them does not block merge.

---

## Security / correctness spot-checks (passed)

- TOCTOU window closed: single `os.Open` + `f.Stat()` on the handle + `io.LimitReader`
  bounded read; oversized and non-regular files rejected (regression tests present).
- Log-injection (CWE-117): `stripControlChars` uses `unicode.IsControl` covering C0,
  DEL, and the full C1 block (U+0080–U+009F); applied at both addr sites and in the
  yaml parse-error detail path. C1-block test and escaped-value test pass.
- Strict decoding (CWE-20): `dec.KnownFields(true)` rejects typo'd keys with E-CFG-005
  naming the offending key.
- Validate is pure-core (no I/O, no mutation) — `validate_does_not_mutate_config`
  subtest confirms; binding sequence in main.go validates before `runAccess`.
- Port range guarded `[0, 65535]`; empty host allowed (`:9090`) per AC-005.

---

## Verdict

**APPROVE.** The cycle-1 blocker is resolved, all 9 ACs are covered by passing
race-clean tests, lint and vet are clean, the description accurately reflects the
diff, and scope (validate-all-fields / apply-tick_interval-only) matches the spec.
The two non-blocking items above (text-vs-animated evidence, dual error-shape) are
quality observations, not merge blockers.
