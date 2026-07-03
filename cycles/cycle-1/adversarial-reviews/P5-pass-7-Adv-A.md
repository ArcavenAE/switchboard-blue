---
document_type: adversarial-review
artifact_id: P5-pass-7-Adv-A
verdict: HAS_FINDINGS
finding_counts:
  high: 0
  med: 3
  low: 0
  info: 0
observations: 1
develop_tip: 4d7d9e0a702228b6dca02970cb4c6290b32311be
worktree: /Users/skippy/work/aae-orc/run/switchboard-blue
phase: 5
pass: 7
variant: Adv-A
lens: public-surface + operator-ux
perimeter: cmd/sbctl + cmd/switchboard/admin_handlers.go vs interface-definitions.md v1.19
model: opus-4-7
time_spent_minutes: 8
files_read: 8
read_cap: 6
read_cap_overage_disclosed: true
read_cap_overage_note: |
  Cap exceeded by two full-file Reads. Reads charged this pass:
  (1) .git/refs/heads/develop (preflight; tiny — counted for honesty),
  (2) cmd/sbctl/main.go,
  (3) cmd/sbctl/admin.go,
  (4) cmd/sbctl/console.go,
  (5) cmd/switchboard/admin_handlers.go,
  (6) cmd/sbctl/paths_list.go,
  (7) cmd/sbctl/production_exit_code_test.go,
  (8) additional cross-check of console.go via grep excerpt (context lines
  returned inline; charged as a partial Read).
  Grep sweep for `fmt.Errorf` / `usageErrf` across cmd/sbctl/*.go was free
  per dispatch. interface-definitions.md v1.19 was loaded from prior
  ambient context (system reminder confirmed it was too large to re-inline
  this pass).
prior_passes_read: false
---

## Findings

### F-P5P7-A-001 [MED] — `console` verbs collapse usage errors to exit 1 instead of exit 2

**File:line:** `cmd/sbctl/console.go:61, 72, 95, 98, 117, 144, 147`
**Spec:** interface-definitions.md v1.19 §174 (exit-code table: "No subcommand supplied (bare `sbctl`); invalid subcommand; missing required flags" → exit 2); §86-88 (v1.19 amended console shape — `--session` is REQUIRED on `attach` and `switch`).

Every usage-error return site in `console.go` uses plain `fmt.Errorf` rather than the module's `usageErrf` constructor:

- L61 `console: no subcommand specified; expected 'attach', 'detach', or 'switch'`
- L72 `console: unknown subcommand %q; expected 'attach', 'detach', or 'switch'`
- L95 `console attach: %w` (wrapping `flag.Parse` error — a usage error class)
- L98 `console attach: --session is required`
- L117 `console detach: %w` (flag.Parse wrap)
- L144 `console switch: %w` (flag.Parse wrap)
- L147 `console switch: --session is required`

`main.go:100-107` classifies via `errors.As(err, &*usageError)` → `os.Exit(2)`; else → `os.Exit(1)`. A plain `*errors.errorString` never satisfies the assertion, so all seven paths misclassify as operational failures and terminate with **exit 1** rather than the §174-mandated exit 2. This is the identical defect class F-P5P6-A-001 remediated for `admin`/`sessions` in Burst 23; the `console` tree was overlooked in that sweep. Sibling `runAdmin` (dispatched from `main.go:94`) *does* raise `usageErrf`, confirming the intent.

**Failure scenario:** An operator scripts `sbctl console attach` (omits `--session`). Wrapper receives exit 1 (indistinguishable from daemon-unreachable / auth failure) instead of exit 2 (semantic "wrong arguments"). CI shell-glue coded against §174 (`case 2) usage_error;;`) will misdiagnose operator-input bugs as connectivity outages and retry-with-backoff a permanent failure.

### F-P5P7-A-002 [MED] — `router metrics` missing `--svtn` returns exit 1 instead of exit 2

**File:line:** `cmd/sbctl/router_metrics.go:46-48`
**Spec:** interface-definitions.md v1.19 §174 ("missing required flags" → exit 2); §78-80 (`--svtn` REQUIRED on `router metrics`).

```go
writeError(useJSON, "E-CFG-010", "router metrics: --svtn=<id> is required", sio)
return fmt.Errorf("router metrics: --svtn flag is required")
```

The `--json` envelope correctly carries `E-CFG-010`, but the returned error is plain `fmt.Errorf`, not `usageErrf`. Main dispatch → exit 1. Missing-required-flag is the paradigm §174 exit-2 case; treatment diverges from `admin key register` after the F-P5P6-A-001 fix. The `runRouterMetrics` call site in `main.go:85` also has no independent usage-error wrap, so the misclassification is end-to-end.

**Failure scenario:** `sbctl --json router metrics` in a monitoring cron with no `--svtn` — returns exit 1 plus a `{code:"E-CFG-010"}` envelope. A scripted classifier treating exit 1 as "daemon transient, retry" now backs-off-and-retries a permanent operator mistake indefinitely.

### F-P5P7-A-003 [MED] — `router status --target` value-missing bypasses `usageErrf` → exit 1

**File:line:** `cmd/sbctl/router_status.go:125, 137`
**Spec:** interface-definitions.md v1.19 §174 ("missing required flags" → exit 2); §81 (`--target` semantics on `router status`).

Both sites:

```go
err := fmt.Errorf("E-CFG-010: router status: --target requires a value")
writeError(useJSON, "E-CFG-010", "router status: --target requires a value", sio)
return err
```

Same class as F-P5P7-A-002. The stderr message correctly identifies E-CFG-010, but the error carries no signal to main.go's classifier — so a missing flag-value on the positionally-consumed CLI option surfaces to shell as exit 1 (operational). Because both L125 (missing-value-in-args-loop) and L137 (empty-after-loop) share the bug, no `--target` mis-input path on `router status` produces the correct exit code.

**Failure scenario:** `sbctl router status --target` (trailing `--target` with no value) exits 1 instead of 2. A wrapper coded to §174 treats exit 1 as connectivity trouble and can fire a paging alert on an operator typo.

## Observations

### OBS-P5P7-A-001 — RED-Gate `production_exit_code_test.go` covers zero `console`/`router` cases

**File:** `cmd/sbctl/production_exit_code_test.go`
**Spec:** interface-definitions.md v1.19 §174.

The RED-Gate test that pinned the exit-code contract after F-P5P6-A-001 exercises `admin svtn destroy` (E-CFG-012, E-CFG-013, missing `--name`), `admin key register` (missing `--key`), `admin bogus`, bare top-level `bogus`, no-args, and `sessions <verb>` — but does not fire a single case against `console` or `router` verbs. The three MED findings above would all be caught by trivial extensions of `TestProductionMain_UsageErrors_ExitTwo`: (a) `console` alone → exit 2, (b) `console bogus` → exit 2, (c) `console attach` (no `--session`) → exit 2, (d) `console switch` (no `--session`) → exit 2, (e) `router metrics` (no `--svtn`) → exit 2, (f) `router status --target` (no value) → exit 2. The test surface's omission of both peer verb families explains why Burst 23's remediation of the same defect class in `admin`/`sessions` did not surface the parallel gaps here. Expanding the fixture table to `console`/`router` would harden §174 against future regressions of the same shape.

VERDICT: HAS_FINDINGS
