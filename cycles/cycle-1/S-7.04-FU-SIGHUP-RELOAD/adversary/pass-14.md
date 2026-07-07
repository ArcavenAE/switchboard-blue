---
pass: 14
story: S-7.04-FU-SIGHUP-RELOAD
story_version: "1.7"
code_sha: 48e3271
verdict: NO_FINDINGS
streak_before: 2
streak_after: 3
bc_5_39_001: SATISFIED
novelty: LOW
anti_findings: 12
observations: 5
date: 2026-07-07
---

# Adversarial Pass 14 — S-7.04-FU-SIGHUP-RELOAD

**Verdict:** NO_FINDINGS  
**Story version:** v1.7  
**Code lane:** 48e3271  
**Streak:** 2/3 → **3/3**  
**BC-5.39.001:** SATISFIED — CONVERGED  

---

## Anti-Findings (12)

**AF-001** Q1 real-signal guard: `TestRunRouterRun_RealSIGHUP_DoesNotExit` continues to exercise the
OS-level `syscall.Kill(os.Getpid(), syscall.SIGHUP)` path through `runRouter`; the sighupCh cap-1
channel drains cleanly under the dedicated-channel model. No shared signal bus contention possible.

**AF-002** Fail-closed — both arms: PEtoPE and PEtoE branches each propagate a parse failure through
`validateAndBuild`; `runRouter` returns the wrapped `E-CFG-001` error without mutating live state.
Fail-open is structurally impossible given the value-copy immutability invariant (AF-003 below).

**AF-003** Non-vacuous cfg immutability including `copy()` sufficiency: `upstreamRoutersFor` builds a
fresh `[]UpstreamRouter` slice on every call; `equalStringSlices` operates on independent copies;
post-reload mutation of the input slice cannot alias the running config. The value-struct copy
semantics are confirmed sufficient for all scalar fields. Slice field deep-copy was verified
non-vacuously at pass 8 (F-P8-001 fixed fa97154) and remains intact at 48e3271.

**AF-004** EC-004 verbatim single-line with control-char-strip chain: `config.go:305-313` strips
Unicode control characters from the `--config` path before any filesystem access; `config.go:499`
strips them before error-detail interpolation. The EC-004 "verbatim" postcondition refers to the
sanitised path string, not raw user input. No newline-injection vector at any call site in scope.
Novel probe angle EC-004: control-char-strip chain clean (see below).

**AF-005** Three E-CFG fail arms independently evidenced: E-CFG-001 (validation failure), E-CFG-003
(file not found / unreadable), and E-CFG-004/E-CFG-005 (path sanitisation) are each covered by
dedicated test cases. Cross-reading them finds no shared early-exit path that could suppress one.

**AF-006** Emission byte-parity: `modeELine` / `modePELine` / `scanForExactModeLine` helpers (added
pass 4 8e159f2) pin the exact byte strings emitted on graduation; pass 12/13 independent re-sweeps
confirm the helpers remain byte-identical to the `fmt.Fprintf` call sites.

**AF-007** Diff-guard all transitions including nil==empty: `equalStringSlices` returns `true` for
both nil and empty slices (consistent zero-value semantics); the diff guard therefore does not
spuriously trigger on a first-time PE config with an empty upstream list. The nil-path was
non-vacuously confirmed at pass 9 (F-P9-001 fix 48e3271) and independently re-verified here.

**AF-008** Untouched surfaces + both liveness probes both paths: `drainCoord`, `keepaliveIntervalFor`,
mgmt-plane startup, and console/access modes are structurally untouched by the SIGHUP reload path;
both `dialMgmtAndReadChallenge` liveness probes (happy-path AC-003 and BadConfig AC-002 PC-6
fail-path) exercise their respective reload outcomes correctly.

**AF-009** FCL 8-row independent re-sweep — all accurate, drift class confirmed closed: the File
Change List carries eight distinct call-site rows. Each was independently re-verified against the
current codebase at 48e3271. All eight are accurate. The FCL-drift recurrence class (five
instances P2/P4/P7/P10/P11) was closed at v1.7 via full-surface sweep. The class remains closed.

**AF-010** Novel probe: EC-004 newline-injection robustness via config control-char-strip chain
(`config.go:305-313` / `499`). A path containing embedded `\n` or `\r` characters is stripped
before E-CFG-004/E-CFG-005 Detail interpolation and before filesystem open. The sanitised path
propagates to the emitted error detail but cannot introduce log injection or path traversal.
**CLEAN** — no finding.

**AF-011** Novel probe: testenv lock/cleanup ordering under `-race`. `testenv.SetSighupCh` is called
before `t.Cleanup` registration in the test harness; the `SendReloadSignal` helper writes to the
channel after `SetSighupCh` establishes the reference. Under `-race` the write cannot race the
cleanup because `t.Cleanup` runs after the test body returns and the signal has already been
consumed or discarded. **CLEAN** — no finding.

**AF-012** go.md hygiene including `yaml.v3`-fixtures-only adjudication: no `interface{}` in scope,
no `init()`, no `log.Fatal` outside `main()`, no shadowed imports, no unbuffered channels used as
queues. The `yaml.v3` import in test fixtures only (not production code) was adjudicated at pass
13; confirmed unchanged at 48e3271.

---

## Observations (5 — all carried / parked confirmations; inert)

**O1** Inert-reload UX — carried to DRIFT-SIGHUP-INERT-RELOAD-UX (S-BL.CLI-SURFACE-COMPLETION era).
A valid SIGHUP that changes only non-upstream fields processes silently. Operator receives no
acknowledgement. Parked since pass 6; unchanged status. Not a defect in this story's scope.

**O2** Order-sensitive diff (`equalStringSlices` is order-sensitive, not set-equal) — PE-CONNECTOR
6th confirmation. The design intent (upstreams are ordered; reload detects positional changes) is
consistent but unratified. Observation anchored to S-7.04-FU-PE-CONNECTOR as the story that will
need to decide whether set-semantics are needed when the dial loop is wired. Inert for this story.

**O3** `upstreamRouters` shared-state under PE-CONNECTOR wiring — PE-CONNECTOR 6th confirmation.
The current single-goroutine runRouter reads `upstreamRouters` without a lock; this is safe today
but will require a mutex or channel-update when S-7.04-FU-PE-CONNECTOR wires the outbound dial
loop. Anchored. Inert for this story.

**O4** Dead guard (`if len(configPath) == 0`) — 5th confirmation, adjudicated-accepted. Three
independent passes (F-P1-009, F-P3-005c, F-P5-003) confirmed and accepted this as intentional
dead code maintaining a defensive posture. No new evidence to reopen. Accepted.

**O5** Hardcoded line-number comments (cosmetic) — the `// config.go:305` style comments in test
helpers are positional annotations that will drift if lines move. Non-blocking; cosmetic. No story
scope change warranted.

---

## Process Notes

- Pre-pass FCL sweep performed: 8-row FCL vs codebase at 48e3271 — all accurate (AF-009).
- Novelty assessment: two novel probe angles (EC-004 control-char-strip / newline-injection;
  testenv lock/cleanup ordering under -race) both clean. No new finding classes discovered.
- Streak accounting: pass 12 = 1/3, pass 13 = 2/3, **pass 14 = 3/3. BC-5.39.001 SATISFIED.**
- Adversary's own streak read verified by orchestrator as correct (3/3).

---

## BC-5.39.001 — CONVERGED

14 passes total. Passes 12, 13, 14 consecutive clean. Streak 3/3. **CONVERGED.**
