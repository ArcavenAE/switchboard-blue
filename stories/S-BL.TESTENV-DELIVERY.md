---
artifact_id: S-BL.TESTENV-DELIVERY
document_type: delivery-ledger
story_id: S-BL.TESTENV
status: pr-open
pr_number: 110
pr_url: https://github.com/ArcavenAE/switchboard-blue/pull/110
branch: feat/s-bl-testenv
commit: 1f47659
timestamp: 2026-07-06T20:16:00Z
---

# S-BL.TESTENV Delivery Ledger

## Deliverables

### New package: internal/testenv

File: `internal/testenv/testenv.go` (DAG position 22 — top test-helper composition root)

**ARCH-08 registration required (coordinator action, post-merge):**

| Position | Package | Allowed imports (internal) | Classification | Story |
|---|---|---|---|---|
| 22 | `internal/testenv` | {admission, drain, frame, session} | test helper | S-BL.TESTENV |

Import set machine-derived: `go list -f '{{.ImportPath}} {{.Imports}}' ./internal/testenv/...`
Result: `[admission drain frame session]` (internal imports only)

**Design decisions:**
- Per-SVTN shard design: each SVTN backed by its own (publisher, auth, accessNode) triple. This enforces SVTN isolation structurally rather than via filtering — frames sent to accessNode-A are never delivered to accessNode-B's consoles. Honest in-process proof of VP-039.
- Admission handshake: RegisterKey performs the full production `admission.AdmitNode` challenge-response using a per-Env router keypair. IsAdmitted returns true after RegisterKey. Callers must use GenerateKey() so the private key is stored for the handshake.
- Goroutine lifecycle: Close() uses sync.WaitGroup. Every background goroutine selects on closeCh and calls defer wg.Done(). Close() blocks until all goroutines exit.
- NewLoopback: provides LoopbackEnv wrapper for VP-042 (S-BL.BENCH) benchmark compile prerequisite.

### New integration test files

All use `//go:build integration` tag.

| File | VP | Test name | Disposition |
|------|-----|-----------|-------------|
| `internal/session/vp033_034_e2e_test.go` | VP-033, VP-034 | TestE2E_Console_AttachDetachLifecycle, TestE2E_MultiConsole_FanOut | activated-and-discharged |
| `internal/admission/vp036_e2e_test.go` | VP-036 | TestE2E_Session_ContinuityAcrossIPChange | activated-and-discharged |
| `internal/drain/vp037_e2e_test.go` | VP-037 | TestE2E_RouterDrain_NodesMigrateWithin2s | harness-ready-test-partial (full lock needs S-7.04-FU-DRAIN-WIRE) |
| `internal/config/vp038_e2e_test.go` | VP-038 | TestE2E_EtoPE_GraduationByConfigChange | harness-ready-test-partial (full lock needs S-7.04-FU-PE-CONNECTOR + SIGHUP-RELOAD) |
| `internal/routing/vp039_e2e_test.go` | VP-039 | TestE2E_SVTN_Isolation_NoCrossSVTNDelivery | activated-and-discharged |
| `internal/multipath/vp040_e2e_test.go` | VP-040 | TestE2E_Multipath_FailoverRecovery | harness-ready-test-partial (wall-clock NFR-003 claim needs production multipath path-selection) |
| `internal/svtnmgmt/vp046_e2e_test.go` | VP-046 | TestIntegration_KeyLifecycle | activated-and-discharged |
| `internal/tmux/vp031_e2e_test.go` | VP-031 | TestTmux_ControlMode_OutputCompleteness | harness-ready-test-partial (skips cleanly when real-tmux not echoing; env has tmux version incompatibility) |

### Modified files

| File | Change |
|------|--------|
| `internal/admission/reauth_test.go` | TestProperty_VP036_SessionContinuity: removed t.Skip deferred placeholder, replaced with log line pointing to vp036_e2e_test.go |
| `internal/tmux/pty_fallback_test.go` | TestPTYProxy_RealPTY_Integration (line 1311): replaced unconditional t.Skip("VP-032-deferred-real-pty") with /dev/ptmx availability probe; skips cleanly when device absent, passes on this machine |

### testenv unit test file

`internal/testenv/testenv_test.go`: 18 unit tests covering all Env methods, -race clean.

## Pre-PR Gates (machine-derived)

| Gate | Result | Evidence |
|------|--------|---------|
| `go test -race ./...` | PASS | 23 packages, 0 failures |
| `just smoke-quick` | 14/14 PASS | INV-1..INV-10 all green |
| `golangci-lint run ./internal/testenv/... [+touched packages]` | 0 issues | |
| Integration tests (`-tags integration`) | VP-033/034/036/037/038/039/040/046 PASS; VP-031 SKIP (clean); VP-032 PASS | |

## VP Disposition Summary

| VP | Lifecycle before | Disposition | Still-deferred reason |
|----|-----------------|-------------|----------------------|
| VP-033 | UNPROVEN-BLOCKED | activated-and-discharged | — |
| VP-034 | UNPROVEN-BLOCKED | activated-and-discharged | — |
| VP-036 | UNPROVEN-BLOCKED | activated-and-discharged | — |
| VP-037 | UNPROVEN-BLOCKED | harness-ready-test-partial | S-7.04-FU-DRAIN-WIRE must also land for full lock |
| VP-038 | UNPROVEN-BLOCKED | harness-ready-test-partial | S-7.04-FU-PE-CONNECTOR + S-7.04-FU-SIGHUP-RELOAD must also land |
| VP-039 | UNPROVEN-BLOCKED | activated-and-discharged | — |
| VP-040 | PARTIAL | harness-ready-test-partial | Wall-clock NFR-003 <2s claim needs production multipath path-selection loop |
| VP-046 | PARTIAL | activated-and-discharged | — |
| VP-031 | PARTIAL | harness-ready-test-partial | Real-device tmux env: tmux sessions enumerable but not echoing via control mode; skips cleanly |
| VP-032 | PARTIAL | harness-ready-test-partial | /dev/ptmx available; replaces permanent-skip with env-conditional skip |

## Ruling Divergences

None. No instructions from the dispatch were deviated from.

**Note on VP-037 and VP-040:** The VP skeletons call `env.CollectFrames(t, sessionID, timeout)` as a top-level Env method. This signature is implemented exactly. VP-037 skeleton calls `pre := env.CollectFrames(t, sessionID, 2*time.Second)` (not `console.CollectFrames`). Implemented as specified.

**Note on VP-039 t.Skip:** VP-039 spec said a t.Skip placeholder "should be present in internal/routing/*_test.go". The actual routing_test.go at line 193 had a comment trace but no t.Skip function body — the deferred function was never added. Rather than adding a placeholder and then removing it, the e2e test was written directly in the integration file. The unit-level proptest coverage (VP-010, S-2.02) is untouched.

## Post-Merge Actions Required

1. **ARCH-08 §6.5 update**: coordinator registers `internal/testenv` at position 22 with import set {admission, drain, frame, session}.
2. **VP lifecycle updates**: coordinator flips VP-033/034/036/039/046 verification_lock per their respective stories once this PR merges and CI confirms green.
3. **VP-037/038/040/031/032 remain open**: still-deferred items tracked per VP files; no lock flip until their blocking stories land.
