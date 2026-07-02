# PR #60 — Fresh-eyes review

**Story:** S-7.03 v1.6 — Console remote control via sbctl
**PR head:** `ce96e58` (matches convergence baseline stated in task context)
**Base:** `develop`  |  **Mergeable:** CLEAN
**CI:** Quality Gate pass, CodeQL pass, dependency-review pass, StepSecurity pass
**Verdict:** **APPROVE**

## What I verified

### Diff correctness vs. spec

Story spec S-7.03 v1.6 lists three file-structure requirements:

| Spec-required file | Present in diff | Purpose |
|---|---|---|
| `cmd/sbctl/console.go` | yes (+152) | sbctl `console attach/detach/switch` CLI dispatch |
| `cmd/sbctl/console_test.go` | yes (+355) | CLI-layer AC-001/002/003 tests |
| `internal/session/remote.go` | yes (+199) | Daemon-side RPC handlers |

Additional supporting changes (all justified):

- `cmd/sbctl/main.go` — replaces the placeholder `connectAndRun(... "console.attach" ...)` with `runConsole(...)` dispatch. Correct wiring.
- `cmd/switchboard/console_handlers.go` — daemon-side handler builder with Tier-2 admission (RoleControl or RoleConsole). Wire codes and error envelopes match the spec's Error Envelope Reference table.
- `cmd/switchboard/mgmt_wire.go` — the `runConsole` daemon entrypoint now actually runs (was a `errors.New("not implemented")` placeholder). Register-before-serve ordering follows the F-P2L1-001/002 pattern. Console keys registered under `zeroSVTN` per ARCH-04 §Console Key Scope.
- `internal/session/session.go` — adds `Publisher.Exists(name)` under `RLock`. Small, focused; satisfies the `SessionRegistry` interface.
- `cmd/switchboard/admin_handlers_e2e_test.go` — drops `//go:build integration` and adds a `//nolint:unparam` on `newE2ESVTNManager`. Necessary because `console_handlers_e2e_test.go` reuses the E2E helpers (`startE2EServer`, `sendAdminRPCAsKey`, `startE2EServerWithOps`). Broader-than-story scope but structurally required.

### AC / PC coverage

- **AC-001 / PC-1 attach:** CLI dispatches `console.attach` with `session_name`; handler validates existence via `SessionRegistry.Exists`, returns E-SES-001 on miss, sets `state.current` under lock on success. CLI test suite covers success + E-SES-001 + E-ADM-006 (wrapped in E-RPC-011). Handler unit test + E2E test present.
- **AC-002 / PC-2 detach:** CLI dispatches `console.detach` with no args; handler returns `ErrConsoleNotAttached` (E-SES-004) when `state.current == ""`, else echoes the previously-attached name and clears state. Detach does not close the session — verified by design of `HandleConsoleDetach` (no cascade into `ConsoleSet`).
- **AC-003 / PC-3 switch:** CLI dispatches `console.switch` with `session_name`; handler validates target first (E-SES-001), then requires attached state (E-SES-004), then sets `state.current = req.SessionName` **not `""`** (L1-C3 fix, explicitly documented in godoc lines 174-177 and verified by the post-switch detach assertion in `TestHandleConsoleSwitch_Success` and `TestConsoleRemote_E2E_VP050`).

### Tier-2 admission (L1-C4)

`verifyConsoleCallerRole` fails closed on:
1. no caller pubkey in context (unauthenticated call)
2. key not in `AdmittedKeySet` for the console partition
3. role not in {`RoleControl`, `RoleConsole`}

All three failure modes tested through the mgmt-plane stack in `TestConsoleRemote_E2E_AdmissionDenied` (attach), `TestConsoleRemote_E2E_AdmissionDenied_SwitchAndDetach` (switch, detach), and `TestConsoleRemote_E2E_ControlRoleAllowed` (positive case for RoleControl).

### Race safety

`ConsoleState.current` is mutex-protected. `Current()` returns a value copy under lock (go.md rule 12). 100-goroutine race test `TestConsoleState_ConcurrentAttachDetachSwitchIsRaceFree` verifies no torn writes. `just test-race` reported PASS per PR body.

### Transport compliance (RULING-W6TB-C)

No data-plane imports in `cmd/sbctl/console.go` (only `context`, `flag`, `fmt`). No `internal/routing`, `internal/arq`, `internal/multipath`, `internal/halfchannel` anywhere in the diff.

### Demo evidence (factory-artifacts branch)

`demo-evidence/S-7.03/` contains:
- `evidence-report.md` (well-formed, links each recording to its test subcases and PC trace)
- `AC-001-attach.{gif,webm,tape}`
- `AC-002-detach.{gif,webm,tape}`
- `AC-003-switch.{gif,webm,tape}`

One recording per AC, both success and error paths covered per the evidence report.

### CI

- Quality Gate: pass (1m7s)
- CodeQL analyze go: pass (1m15s)
- CodeQL: pass
- dependency-review: pass
- StepSecurity Harden-Runner: pass
- Release-signing jobs (Build Binaries / Alpha Release / Sign & Notarize / Homebrew): skipping — correct, this is a feature branch not a release.

## Findings

None blocking.

**Non-blocking observations (all pre-acknowledged in PR body / spec deferrals):**

- O-1 (VP-050 skeleton `env.ConsoleState` illustrative marker): accepted per POL-003 Exception A.
- O-2 (BC-2.08.001 line 135 `(Inv-3 v1.2)` origin pointer): accepted per POL-003 Exception A.
- F-P5L2-LOW-1 / F-P5L2-LOW-2 (stale "Unix socket" nomenclature in `cmd/sbctl/console.go` and `console_test.go` docstrings): deferred to companion doc-sweep PR.
- SEC-001 MEDIUM (revoked/expired key bypass at Tier-2): pre-existing defense-in-depth gap, not introduced by this PR.
- TOCTOU on `Exists → state.mu.Lock` in `HandleConsoleAttach`: explicitly acknowledged in godoc (lines 130-137) with sound rationale — `state.current` is a name string not a live pointer, downstream rendering re-validates against the live registry.

## Verdict

**APPROVE.** The diff faithfully implements S-7.03 v1.6, matches the CONVERGED 3/3 adversarial baseline at `ce96e58`, all three ACs and their error envelopes are covered by both unit and E2E tests, demo evidence is complete, and CI is green.
