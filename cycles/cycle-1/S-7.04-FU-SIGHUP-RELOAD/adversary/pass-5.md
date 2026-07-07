---
artifact_id: S-7.04-FU-SIGHUP-RELOAD-adversary-pass-5
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 5
verdict: HAS_FINDINGS
novelty: MED
code_lane_sha: 8e159f2
story_version: "1.3"
reviewer_model: fresh-context
timestamp: 2026-07-07T00:00:00Z
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 5

## Summary

**Verdict:** HAS_FINDINGS  
**Novelty:** MED (seam-vs-OS-signal axis was novel — all prior passes exercised SIGHUP via the in-process buffered channel; the OS-level signal registration path in `main.go` was never exercised by any test)  
**Code lane SHA:** 8e159f2  
**Story version:** v1.3  
**Streak:** 0/3 (pass 6 pending)

3 findings, all LOW severity. Zero correctness findings. All remediated same burst (3c3ce0e).

## Findings

### F-P5-001 — LOW: main.go SIGHUP registration at 0% coverage [FIXED 3c3ce0e]

**Category:** test-rigor  
**Status:** FIXED (3c3ce0e)

The binding architect's Q1 decision correctly wired `signal.Notify` for `syscall.SIGHUP` in
`cmd/switchboard/main.go`, routing the OS signal into the `sighupCh` buffered channel fed
into `run()`. However, this registration path had no regression guard in any test: all prior
seam tests bypass `main.go` entirely, calling `runRouter` or `run()` directly with a
caller-controlled `sighupCh`. A change to the `main.go` signal registration (wrong signal,
wrong channel, missing `signal.Notify` call) would go undetected.

The seam tests remain valid for their scope (isolating the reload-logic seam from OS signal
delivery), but the question "does SIGHUP actually reach the reload path when sent via the
OS?" had zero test coverage.

**Remediation:** `TestRunRouterRun_RealSIGHUP_DoesNotExit` added in
`cmd/switchboard/router_sighup_test.go` (commit 3c3ce0e). The test calls `run()` directly
(not `runRouter`) with a real goroutine and sends `syscall.Kill(os.Getpid(), syscall.SIGHUP)`
via the OS. The assertion is that `run()` does not exit within the observation window,
distinguishing SIGHUP-triggered reload (non-exit) from drain-shutdown (exits). End-to-end
reload emission via `run()` was declined because `os.Stderr` is not observable without
production plumbing; the load-bearing assertion is not-exiting, which is the correct
observable for the seam-vs-OS-signal axis.

### F-P5-002 — LOW: AC-002 PC-6 fail-path liveness unpinned [FIXED 3c3ce0e]

**Category:** test-rigor  
**Status:** FIXED (3c3ce0e)

AC-002 (postcondition 6) states that after a config-reload failure, the router remains
operational: existing connections survive and new connections are accepted. The existing
`BadConfig` test verified that a reload failure did not crash the process, but it did not
probe the management plane after the failure — leaving the liveness assertion implicit
rather than directly observed.

**Remediation:** `BadConfig` test extended in `cmd/switchboard/mgmt_wire_test.go`
(commit 3c3ce0e): a held-connection read-deadline probe and a `dialMgmtAndReadChallenge`
post-failure call are added, reusing the helpers introduced in 8e159f2. The probe dials
the mgmt listener after a reload failure and reads the challenge byte, asserting the mgmt
channel type. This directly evidences PC-6 liveness without changing the failure-path
semantics.

### F-P5-003 — LOW: dead configPath guard uncovered [ADJUDICATED-ACCEPTED]

**Category:** coverage / defensive-code  
**Status:** ADJUDICATED-ACCEPTED (third confirmation; no remediation)

The `configPath == ""` guard in the SIGHUP reload case of `runRouter` has been surfaced
by three consecutive adversarial passes (F-P1-009, F-P3-005c, F-P5-003). The guard is
a defensive check that cannot be reached via any test path: reaching it requires passing
an empty `configPath` to a live `sighupCh`, which no caller does by design.

Three consecutive independent reviewers surfaced this branch as uncovered. Three
consecutive times the adjudication has been the same: the guard is legitimately defensive,
the missing-coverage is an intrinsic property of the design (the branch is provably
unreachable from correct call sites), and adding a test to cover it would require
artificially constructing an incorrect call that the type system does not prevent.

**Ruling:** ADJUDICATED-ACCEPTED. No further remediation warranted. The triple-confirmation
pattern is noted — this branch is a stable dead-code observation, not a latent defect.
Future passes should treat this branch as a known-adjudicated item and exclude it from
novel findings.

---

## Anti-findings (12)

1. **POL-001 compliance confirmed** — pass-5.md file authored and indexed correctly.
2. **POL-002 compliance confirmed** — story v1.3 changelog rows present and accurate.
3. **All 26 prior-pass remediations held** — P1 (12 findings), P2 (5 findings), P3 (5 findings),
   P4 (4 findings), all confirmed present at 8e159f2. No regression.
4. **`modeELine`/`modePELine`/`scanForExactModeLine` helpers stable** — full-line pin format
   introduced in 8e159f2 covers all emission paths; no substring-match drift reintroduced.
5. **`dialMgmtAndReadChallenge` helper correct** — post-reload mgmt probe introduced in 8e159f2
   correctly verifies channel type; helper reuse in F-P5-002 remediation is coherent.
6. **AC-001 emission format end-to-end pinned** — `TestRunRouter_SIGHUPReload_EtoPE` pins
   startup `mode=E` and reload `mode=PE` full-line; no partial match.
7. **AC-004 VP-038 in-process channel injection confirmed** — `SendReloadSignal` uses buffered
   channel; no `syscall.Kill` call in this path (orthogonal to F-P5-001 OS-path test).
8. **Order-sensitive diff already PE-CONNECTOR-anchored** — upstream_routers ordering concern
   addressed in prior passes; no new instance surfaced.
9. **`isNetError` direct-assertion approach noted** — the current indirect-assertion shape
   (behavioral probe) is the correct observable for this seam; direct isNetError assertion
   would tie tests to implementation internals.
10. **SIGHUP-during-shutdown behavior benign** — signal arrives after drain starts; reload
    path is no-op because `sighupCh` is drained or context is cancelled; no livelock risk.
11. **AC-004 seam theatrical-but-compliant** — `SendReloadSignal` injects into the channel
    seam, not via OS; this is an AC-004 PC-2 emission-based observable per story v1.2 ruling;
    not a coverage gap.
12. **`just test-race` and `just lint` clean** — zero race detector findings and zero
    golangci-lint warnings at 8e159f2.

---

## Finding Decay Trajectory

| Pass | Novelty | Findings | Correctness |
|------|---------|----------|-------------|
| P1 | HIGH | 12 | 0 correctness |
| P2 | MED | 5 | 0 correctness |
| P3 | MED | 5 | 0 correctness |
| P4 | LOW | 4 | 0 correctness |
| P5 | MED | 3 | 0 correctness |

Note: novelty MED (not LOW) because the seam-vs-OS-signal axis was genuinely novel — not
a derivative of prior passes but an orthogonal coverage dimension uncovered for the first
time this pass. Trajectory remains strongly convergent; the novel axis is now closed (3c3ce0e).
Pass 6 pending (target 3/3 clean streak).
