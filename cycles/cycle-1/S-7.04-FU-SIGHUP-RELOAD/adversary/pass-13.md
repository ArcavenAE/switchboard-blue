---
pass: 13
story: S-7.04-FU-SIGHUP-RELOAD
story_version: v1.7
code_lane: 48e3271
verdict: NO_FINDINGS
novelty: LOW
streak_pre: 1/3
streak_post: 2/3
date: 2026-07-07
---

# Adversary Pass 13 — S-7.04-FU-SIGHUP-RELOAD

**Verdict: NO_FINDINGS**
**Code lane:** 48e3271 (story v1.7)
**Streak:** 1/3 → 2/3

## Anti-Findings (11)

**AF-1: Q1 real-signal guard — confirmed.**
The dedicated `sighupCh` receives only from the OS signal registration (signal.Notify), and the test TestRunRouterRun_RealSIGHUP_DoesNotExit sends a real `syscall.Kill(os.Getpid(), syscall.SIGHUP)` through `run()`. No path allows a spurious wakeup on `sighupCh` to trigger reload. The Q1 guard (dedicated channel, cap-1, single-writer signal.Notify) is non-vacuous and fully exercised.

**AF-2: Fail-closed both arms — confirmed.**
PEtoPE branch: non-empty deep-copy assertions verify copy independence post-reload (F-P8-001 fix, fa97154). PEtoE branch: same deep-copy pattern. Both arms reload the config from disk and replace the in-memory state atomically; failure returns E-CFG-001. No fail-open path exists.

**AF-3: Non-vacuous cfg immutability including value-struct copy() sufficiency — confirmed.**
The `equalStringSlices` helper is 100%-covered. The `upstreamRoutersFor` helper produces a fresh slice on every call (no aliasing). The PEtoPE and PEtoE immutability tests construct non-empty slices, trigger reload, then mutate the original — the copy is unaffected. Copy sufficiency confirmed for the Config value struct (no pointer fields that would require deep clone beyond slices).

**AF-4: EC-004 verbatim single-line with control-char-strip — confirmed.**
The mode-line emission helpers (`modeELine`, `modePELine`, `scanForExactModeLine`) pin the exact string written to stdout. The control-char strip (CWE-117, PR #95 7a974f6) runs on the `--config` path argument before it appears in E-CFG-004/E-CFG-005 Detail interpolation. EC-004 emission in the reload path uses these same helpers, verified by AC-001 and AC-002 tests.

**AF-5, AF-6, AF-7: Three E-CFG fail arms — confirmed.**
- E-CFG-001 (parse failure): `TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed` pins this path.
- E-CFG-003 (reload structurally covered): the success path exercises the "reload processed" arm; the bad-config reload test exercises the fail arm.
- E-CFG-004/E-CFG-005 (mode line emission): AC-001 (PEtoE) + AC-002 (PEtoPE) both assert exact mode-line format post-reload.

**AF-8: Emission byte-parity — confirmed.**
`scanForExactModeLine` uses `bufio.Scanner` to read stdout line-by-line and match the full line verbatim. No trailing whitespace or newline discrepancy can silently pass.

**AF-9: Diff-guard all transitions including nil==empty — confirmed.**
`equalStringSlices` returns false for (nil, []string{}) and ([]string{}, nil) in the same direction — both are treated as unequal. The TestRunRouter_SIGHUPReload tests cover the nil-original-to-non-empty and non-empty-to-non-empty cases. The nil==empty edge is not silently elided.

**AF-10: Untouched surfaces and both liveness probes both paths — confirmed.**
`drainCoord` is intentionally untouched (DRAIN-WIRE era scope, documented in transitional-seam comments). Both liveness probes — AC-003 (session-non-interruption: `select { case <-doneCh: t.Fatal(...) }` no-return assert pattern, identical to 9 siblings) and AC-002 (management plane reachable post-reload: `dialMgmtAndReadChallenge`) — pass on both the PEtoE and PEtoPE reload paths.

**AF-11: FCL 8-row independent re-sweep all accurate — drift class confirmed closed.**
Full File-Change-List sweep (pass-11 F-P11-001 class-closure fix):
1. `internal/config/config.go` — reload logic: accurate.
2. `internal/config/config_test.go` — config unit tests: accurate.
3. `cmd/switchboard/main.go` — SIGHUP registration: accurate (TestRunRouterRun_RealSIGHUP_DoesNotExit added pass 5).
4. `cmd/switchboard/main_test.go` — real-signal test: accurate.
5. `cmd/switchboard/router.go` — runRouter reload dispatch: accurate.
6. `cmd/switchboard/router_test.go` — reload integration tests: accurate (ten tests per pass-10 v1.6 fix).
7. `internal/testenv/testenv.go` — SetSighupCh post-hoc setter: accurate (v1.5 pass-9 F-P9-002 fix).
8. `internal/testenv/testenv_test.go` — testenv harness tests: accurate.
All 8 rows verified against code at 48e3271. Drift class confirmed CLOSED.

**AF-11b: go.md hygiene including yaml.v3-fixtures-only adjudication — confirmed.**
No yaml.v3 imports in the reload path. yaml.v3 is used only in testdata fixtures and is not in the production signal handler or config reload path. The `nolint:errorlint` on `isNetError` (F-P6-O4 triple-confirmed, pass-7 adjudicated-accepted) remains the only nolint in scope; its justification (idiomatic Go net.Error type-assert, not an errors.Is candidate) stands.

**AF-11c: POL-001/POL-002/POL-004 compliance — confirmed.**
- POL-001 (BC traceability): BC-2.09.001 PC-1 and BC-2.09.003 EC-004 are both traced in the story.
- POL-002 (story-index row sync): STORY-INDEX row at v4.01 carries `ready (v1.7, 2026-07-07; adversarial cycle: 12 passes, streak 1/3, pass 13 pending)` — accurately reflects the pre-pass-13 state.
- POL-004 (VP traceability): VP-038 traced in story frontmatter; VP-038 v1.2 deferral annotation (thin-by-design, awaiting S-7.04-FU-PE-CONNECTOR production multipath) confirmed correct and unchanged.

## Observations (5 — all non-defect confirmations of parked/anchored items)

**O1: Inert-reload drift (DRIFT-SIGHUP-INERT-RELOAD-UX) — parked, confirmed.**
Valid SIGHUP reloads that change only non-upstream fields (drain_timeout, keepalive_interval) produce no operator-visible output change. This is tracked as DRIFT-SIGHUP-INERT-RELOAD-UX (LOW, anchor S-BL.CLI-SURFACE-COMPLETION). No new information; parked anchor still accurate.

**O2: Order-sensitive diff observation — PE-CONNECTOR obligation, confirmed.**
The upstream_routers comparison uses `equalStringSlices` which is order-sensitive. When two configs have the same router addresses in different order, a reload is triggered unnecessarily. This is the 6th confirmation of the PE-CONNECTOR forward obligation (upstreamRouters-order-sensitive-diff). No new information.

**O3: upstreamRouters shared-state under PE-CONNECTOR — PE-CONNECTOR obligation, confirmed.**
Under PE-CONNECTOR (S-7.04-FU-PE-CONNECTOR), `upstreamRoutersFor(cfg)` will feed a live connection-dialing goroutine. A concurrent reload could race on the slice. This is the 6th confirmation of PE-CONNECTOR obligation #4 (race safety under concurrent dialing). No new information.

**O4: Dead guard (5th confirmation) — accepted.**
The empty `configPath` guard (`if configPath == ""`) at the top of runRouter is dead code — the caller always passes a non-empty path (validated upstream by E-CFG-001 at startup). Triple-confirmed at passes 1/3/5; adjudicated accepted. Now 5th confirmation. No change warranted.

**O5: Hardcoded line-number comments — cosmetic, confirmed.**
Several test helper functions carry inline line-number comments from the initial implementation (e.g., `router.go:141`). These are cosmetic and will drift as the file evolves. Not a defect; noted as maintenance-class hygiene for a future sweep.

## Novel probe angles explored (both clean)

**YAML round-trip fidelity probe:** Does `config.Load()` → `equalStringSlices` comparison round-trip correctly for YAML arrays with trailing commas, quoted strings, or YAML anchors? Verified: the test fixtures use canonical YAML without anchors; the comparison is on the parsed Go slice values, not the raw YAML bytes. No round-trip fidelity gap.

**Cross-test signal interference probe:** Could TestRunRouterRun_RealSIGHUP_DoesNotExit's `syscall.Kill(os.Getpid(), syscall.SIGHUP)` leak into a concurrently-running test in the same process? Verified: the test uses `t.Parallel()` isolation and the signal handler is registered inside `run()` with a scoped `signal.Stop()` + defer in the test helper. No cross-test interference observed.

## Summary

Pass 13 is NO_FINDINGS at 48e3271 / story v1.7. All 11 anti-findings confirmed independently. Five observations confirmed as non-defect (all parked/anchored from prior passes). Two novel probe angles (YAML round-trip fidelity, cross-test signal interference) both clean. Novelty LOW — the search space is exhausted at this code lane. **Streak advances: 1/3 → 2/3. Pass 14 is the convergence pass — if clean, BC-5.39.001 is satisfied for S-7.04-FU-SIGHUP-RELOAD.**
