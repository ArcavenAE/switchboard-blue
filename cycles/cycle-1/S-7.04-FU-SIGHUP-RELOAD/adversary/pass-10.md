---
artifact_id: adversary-pass-10-sighup-reload
pass: 10
story: S-7.04-FU-SIGHUP-RELOAD
code_sha: 48e3271
story_version: "1.5"
verdict: HAS_FINDINGS
novelty: MED
streak_before: 0/3
streak_after: 0/3
streak_ruling: holds
timestamp: 2026-07-07T00:00:00Z
---

# Adversary Pass 10 — S-7.04-FU-SIGHUP-RELOAD

**Verdict:** HAS_FINDINGS  
**Code lane:** 48e3271 (UNCHANGED from pass 9 — zero code findings since pass 2; pass-10 finding is story-doc only)  
**Story version at dispatch:** v1.5  
**Story version post-remediation:** v1.6  
**Streak:** holds 0/3 (findings gate strictly regardless of finding class per BC-5.39.001)  

---

## Findings

### F-P10-001 — FCL-and-Task-2-undercount-nine-vs-ten [process-gap] LOW

**Class:** FCL-drift (4th recurrence: P2-004 → P4-003 → P7-001 → P10-001)

**Observation:** The File-Change List (FCL) row for `cmd/switchboard/router_sighup_test.go` and the
Task 2 description both state the file contains **nine** integration tests. The actual count
(machine-verified by grep) is **ten**:

1. TestRunRouter_SIGHUPReload_EtoPE (AC-001)
2. TestRunRouter_SIGHUPReload_BadConfig_FailClosed (AC-002)
3. TestRunRouter_SIGHUPReload_SessionsNotInterrupted (AC-003)
4. TestRunRouter_VP038_EtoPEViaConfigOnly (AC-004)
5. TestRunRouter_SIGHUPReload_LoadFileNotFound (P1 F-005)
6. TestRunRouter_SIGHUPReload_MalformedYAML (P2 F-001)
7. TestRunRouter_SIGHUPReload_PEtoE (P3 F-005a)
8. TestRunRouter_SIGHUPReload_PEtoPE (P3 F-005a)
9. TestRunRouter_SIGHUPReload_IdempotentResend (P3 F-005a)
10. TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed (P8 F-P8-002, commit fa97154)

**Root cause:** Pass-8 remediation (commit fa97154) added `TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed`
without a paired story edit — violating the pass-9 orchestrator lesson ("doc-class remediations
must land in the same burst as their code fix"). The story-writer was not invoked when fa97154
landed, so the count in the story remained at nine.

**Remediation:** Story v1.6 (same burst — paired edit per orchestrator lesson):
- FCL row for `router_sighup_test.go` corrected: nine → ten; remediation-added list five → six,
  naming `TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed` with commit fa97154 provenance
  and P8 F-P8-002 citation.
- Task 2 description corrected: "four failing tests" header wording preserved (AC story count);
  body description updated to reflect ten total (four AC + six remediation-added).

**Machine verification:** `grep -c '^func Test' cmd/switchboard/router_sighup_test.go` → 10
(verified independently by both story-writer and orchestrator).

**Disposition:** FIXED in story v1.6 (same burst, zero code changes).

---

## Observations (non-finding, informational)

### O1 — Dead-guard `if configPath == ""` in reload path (4th confirmation)

**Status:** ACCEPTED (adjudicated-accepted at P3-005c, P5-003, P7-obs — triple-confirmed).
The empty-path guard at the top of the `sighupCh` case body is structurally dead in production
(`main.go` always passes `*configPath`, which is validated non-empty at daemon startup) but serves
as a belt-and-suspenders fail-closed defense. No test exercises it by design. Adversary
acknowledges the triple-prior acceptance; no new evidence offered. Carry as accepted.

### O2 — `equalStringSlices` order-sensitive diff (PE-CONNECTOR anchored)

`equalStringSlices` compares slices positionally — `[a, b]` ≠ `[b, a]`. If two valid YAML
configs with the same upstream set listed in different order trigger a spurious re-emission.
Acknowledged. The order-sensitivity question is anchored as a PE-CONNECTOR forward obligation
(5th, codified at pass-9 remediation commit 48e3271). Not a finding per the anchoring.

### O3 — `upstreamRouters` local variable single-goroutine read safety (PE-CONNECTOR anchored)

When the PE-CONNECTOR TCP dial loop (`S-7.04-FU-PE-CONNECTOR`) is wired, the `upstreamRouters`
local variable will be read from a second goroutine without a lock. Safe today (only `runRouter`
goroutine touches it); will require attention at PE-CONNECTOR. Anchored as PE-CONNECTOR 4th
forward obligation (pass-8 O1). No new evidence; carry forward.

### O4 — `IdempotentResend` bounded absence-window (non-blocking)

`TestRunRouter_SIGHUPReload_IdempotentResend` verifies that sending SIGHUP on the same PE config
twice does not emit a second `mode=PE` line (idempotent diff). The test relies on a bounded scan
window — if the `runRouter` goroutine emits a second `mode=PE` after the scan window closes,
the test passes silently. The window is short enough that a genuine double-emit would likely
fall within it in practice. Accepted as a known test-design limitation; not a correctness defect.

---

## Anti-Findings (independent re-derivation)

Full hardening stack re-derived fresh-context; all 16 pass with HAS_FINDINGS verdict still
applying (single process-gap finding).

1. Q1 guard: `TestRunRouterRun_RealSIGHUP_DoesNotExit` (main_test.go) provides real-OS-signal
   evidence for the dedicated-channel architectural decision. Guard is non-vacuous.
2. Cfg immutability both paths: PEtoPE + PEtoE tests assert non-empty deep-copy independence
   (commit fa97154). Vacuity objection closed.
3. All-ten no-return asserts: passes 3–9 confirmed goroutine-continues asserts on all nine prior
   tests; pass-10 confirms the tenth (InvalidUpstreamAddr) also carries a no-return assert.
4. Both liveness arms: AC-001 (valid reload) and AC-002 (fail-closed) both assert daemon
   continues running after SIGHUP handling. No single-arm gap.
5. Seam-doc sync: SetSighupCh + SendReloadSignal doc comments updated at 48e3271 to reflect
   transitional-seam shape and PE-CONNECTOR construction-time-wiring obligation. Consistent.
6. Byte-parity format: `mode=PE upstream_routers=[...]` reload emission format matches
   startup-emission format from S-7.04 (same `fmt.Fprintf(w, "mode=PE upstream_routers=%v\n", ...)` shape).
7. POL-001 compliance: story v1.6 changelog row present; version bump 1.5→1.6 recorded.
8. POL-002 compliance: STORY-INDEX row cell updated in same burst (v1.5, 9 passes, streak 0/3,
   pass 10 pending) → (v1.6, 10 passes, streak 0/3, pass 11 pending).
9. POL-004 compliance: no BC or VP frontmatter touched; no BC-INDEX or VP-INDEX bump required.
10. Code lane unchanged: zero code findings across passes 3–10. fa97154 was the last code commit;
    48e3271 added only seam-doc comments and a no-return assert. Code lane perimeter clean.
11. FCL completeness (post-remediation): eight-file FCL verified against commit diff. All files
    listed. Post-v1.6 count is accurate at ten.
12. E-CFG-001 taxonomy: reload failure message wraps LoadFile / Validate errors verbatim per
    EC-004 format — `config reload failed: <err>; continuing with previous config`. E-CFG-001
    text inside `<err>` for Validate failures; E-CFG-004/E-CFG-005 text inside `<err>` for I/O
    failures. Taxonomy rendering verified.
13. E-CFG-003 reload coverage: InvalidUpstreamAddr test (`upstream_routers: ["not-a-host:port"]`)
    exercises the `loaded.Validate()` error path carrying E-CFG-001 outer + E-CFG-003 detail
    fragment. Cross-BC Note in story acknowledges the rendering nuance (E-CFG-001 renders,
    E-CFG-003 appears as detail text, not as a standalone code). Structurally covered.
14. Atomicity: state mutations (upstreamRouters assignment + `w` write) occur only after
    full successful LoadFile + Validate + diff sequence in a single goroutine with no
    partial-application path. AC-001 precondition §6 satisfied.
15. AC-003 untouched-construct list complete: ingressCtx/ingressCancel, dataWG, drainCoord,
    mgmtSrv/mgmtWG, parent ctx — all five named and tested via AC-003 assertion. No missing
    construct.
16. PE-CONNECTOR forward-obligation registry current: 5 obligations recorded (O1 order-sensitivity,
    O2 dead-code guard future-test, O3 AC-004 RouterHandle.Mode() assertion, O4 upstreamRouters
    race under dial goroutine, O5 construction-time sighupCh wiring). Registry complete at
    current code lane.

---

## Orchestrator Codified Lessons (from this pass)

Two process-discipline rules codified as a result of the FCL-drift 4th recurrence:

**Rule 1 (same-burst paired story edit):** Any burst that adds or renames tests in an in-flight
story's test file MUST include a paired story-writer edit in the same atomic commit. The story
edit is not a follow-up — it is part of the burst's definition of done. No exceptions.

**Rule 2 (pre-pass count verification):** Before dispatching each adversary pass, the orchestrator
runs `grep -c '^func Test' <test-file>` and compares the result against the story's current FCL
count. If they diverge, a story-writer burst fires first. The adversary pass does not dispatch
against a stale story.

These rules are intended to permanently close the FCL-drift class. The 4-pass recurrence
(P2-004 → P4-003 → P7-001 → P10-001) demonstrates that post-hoc codification without
prevention-mechanism change is insufficient. The prevention mechanism change is Rule 2
(detection gate before dispatch) and Rule 1 (obligation at mutation time).

---

## Verdict Summary

| Dimension | Assessment |
|-----------|------------|
| Correctness | No correctness findings (8 consecutive passes with zero code findings) |
| Test strength | No test-strength findings (closed at P8) |
| Consistency/polish | No further gaps after v1.6 remediation |
| Process-gap | F-P10-001 FCL-drift 4th recurrence → FIXED story v1.6 |
| Novelty | MED (recurrence-despite-codified-lesson was the novel signal; prior lessons insufficient without prevention mechanism change) |
| Streak ruling | Holds 0/3 per BC-5.39.001 (findings gate strictly regardless of class) |
| Awaiting | Adversary pass 11 (streak 0/3) |
