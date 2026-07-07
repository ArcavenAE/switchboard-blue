---
artifact_id: S-7.04-FU-SIGHUP-RELOAD-adversary-pass-4
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 4
verdict: HAS_FINDINGS
novelty: LOW
code_lane_sha: 8a40a0a
story_version: "1.3"
reviewer_model: fresh-context
timestamp: 2026-07-06T23:30:00Z
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 4

## Summary

**Verdict:** HAS_FINDINGS  
**Novelty:** LOW  
**Code lane SHA:** 8a40a0a  
**Story version:** v1.3  
**Streak:** 0/3 (pass 5 pending)

4 findings, all LOW severity. Zero correctness findings. All remediated same burst.

## Findings

### F-P4-001 — LOW: PC-4 emission format unpinned in prior story version [FIXED 8e159f2]

**Category:** test-rigor  
**Status:** FIXED (8e159f2)

The `mode=PE upstream_routers=[...]` emission format used in the AC-001 test assertion
was previously matched with a substring scan rather than a full-line match, leaving the
exact wire format (bracket notation, spacing around `=`) unpinned against future drift.
Additionally, the inverse path (`mode=E`) lacked a symmetric full-line assertion helper.

**Remediation:** `modeELine`, `modePELine`, and `scanForExactModeLine` helpers introduced
in `cmd/switchboard/router_sighup_test.go` (commit 8e159f2). AC-001 (`TestRunRouter_SIGHUPReload_EtoPE`),
`PEtoE`, and `PEtoPE` tests updated to pin the full-line format. AC-001 and AC-004
(`TestRunRouter_VP038_EtoPEViaConfigOnly`) also pin the startup `mode=E` and reload
`mode=PE` lines end-to-end.

### F-P4-002 — LOW: Phantom E-CFG-002 example in AC-002 precondition [process-gap] [FIXED story v1.3]

**Category:** spec accuracy  
**Status:** FIXED (story v1.3)

AC-002 precondition described `listen_addr: "notaport"` as triggering E-CFG-002. This is
incorrect: `internal/config` wraps all validation failures under E-CFG-001. E-CFG-002 is
a code comment naming convention only, never rendered in user-visible error output. A test
built against "E-CFG-002" semantics would either fail immediately or test the wrong error
taxonomy.

**Remediation:** Story v1.3 replaces the precondition example with an empty `listen_addr`
field, which correctly triggers E-CFG-001 naming the failing field. The E-CFG-001 taxonomy
note in AC-002 is updated to match.

### F-P4-003 — LOW: File-Change List and Task 2 description stated four tests, not nine [process-gap] [FIXED story v1.3]

**Category:** spec accuracy / traceability  
**Status:** FIXED (story v1.3)

The File-Change List entry for `router_sighup_test.go` and the Task 2 description both
originally stated "four integration tests" (one per AC). By the end of adversarial passes
P1–P3, five additional tests were added (LoadFileNotFound, MalformedYAML, PEtoE, PEtoPE,
IdempotentResend). The story body had not been updated to reflect the final count.

**Remediation:** Story v1.3 corrects both the File-Change List entry and Task 2 description
to state nine tests, with the five additional tests named and their pass-of-origin noted.

### F-P4-004 — LOW: AC-003 mgmt-plane clause unasserted in tests [FIXED 8e159f2]

**Category:** test-rigor  
**Status:** FIXED (8e159f2)

AC-003 postcondition 2 lists five named constructs that must not be touched by the reload
path (`ingressCtx`/`ingressCancel`, `dataWG`, `drainCoord`, `mgmtSrv`/`mgmtWG`, parent `ctx`).
The existing `TestRunRouter_SIGHUPReload_SessionsNotInterrupted` verified that an accepted
TCP connection survived the reload but did not explicitly probe the management plane side —
specifically that the mgmt server continued serving requests post-reload.

**Remediation:** `dialMgmtAndReadChallenge` post-reload probe added in
`cmd/switchboard/mgmt_wire_test.go` (commit 8e159f2). The probe dials the mgmt listener
and reads the challenge byte after SIGHUP completes, asserting the mgmt channel type.
This closes the AC-003 mgmt-plane assertion gap without touching `drainCoord` or `dataWG`
directly (the proof is behavioral: if the mgmt server had been stopped, the dial would fail).

---

## Anti-findings (12)

1. **POL-001 compliance confirmed** — pass-4.md file authored and indexed correctly.
2. **POL-002 compliance confirmed** — story v1.3 changelog row present and accurate.
3. **All 22 prior-pass remediations held** — P1 (12 findings), P2 (5 findings), P3 (5 findings),
   all confirmed present at 8a40a0a. No regression.
4. **`equalStringSlices` coverage 100%** — both same-slice and differing-slice cases exercised
   by `IdempotentResend` and `PEtoPE` tests.
5. **`upstreamRoutersFor` coverage 100%** — E-mode (empty) and PE-mode (non-empty) paths
   each exercised.
6. **`runRouter` coverage 90.5%** — uncovered lines adjudicated dead code (the `configPath == ""`
   guard shortcircuit in the reload case; a legitimate defensive check that can only be
   reached by passing an empty configPath to a live sighupCh, which no test does by design).
7. **AC-001 emission format now full-line-pinned** — `scanForExactModeLine` helper eliminates
   substring-match ambiguity.
8. **AC-002 fail-closed path remains single-path** — no partial application possible; both
   LoadFile-error and Validate-error paths route to the same `fmt.Fprintf(w, "config reload failed: %s; ...")` template.
9. **AC-003 TCP connection survivability confirmed** — open-connection probe passes before and
   after SIGHUP under `-race`.
10. **AC-004 VP-038 in-process channel injection confirmed** — `SendReloadSignal` uses the
    buffered channel directly; no `syscall.Kill` call present.
11. **`just test-race` clean** — zero race detector findings at 8a40a0a across the full package.
12. **`just lint` clean** — zero golangci-lint warnings at 8a40a0a.

---

## Finding Decay Trajectory

| Pass | Novelty | Findings | Correctness |
|------|---------|----------|-------------|
| P1 | HIGH | 12 | 0 correctness |
| P2 | MED | 5 | 0 correctness |
| P3 | MED | 5 | 0 correctness |
| P4 | LOW | 4 | 0 correctness |

Decay strongly supports convergence. Pass 5 pending (target 3/3 clean streak).
