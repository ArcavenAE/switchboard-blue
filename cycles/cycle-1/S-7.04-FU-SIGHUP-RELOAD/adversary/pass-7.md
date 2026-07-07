---
artifact_id: S-7.04-FU-SIGHUP-RELOAD-adversary-pass-7
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 7
verdict: HAS_FINDINGS
novelty: LOW
code_lane_sha: 3c3ce0e
story_version: "1.3"
reviewer_model: fresh-context
timestamp: 2026-07-07T02:00:00Z
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 7

## Summary

**Verdict:** HAS_FINDINGS  
**Novelty:** LOW  
**Code lane SHA:** 3c3ce0e (unchanged since pass 5 — no code findings since pass 2)  
**Story version reviewed:** v1.3 (pre-remediation); **story version post-remediation:** v1.4  
**Streak:** 1/3 → 0/3 (RESET — doc-class finding)

1 finding (LOW, process-gap class). 12 anti-findings. Novelty LOW.

---

## Findings

### F-P7-001 [LOW] [process-gap] File-Change-List omits `cmd/switchboard/main_test.go`

**Class:** recurrence of F-P2-004 class — test file added during adversarial remediation without corresponding File-Change-List row update in the same burst.

**Evidence:** Pass 5 F-SIGHUP-P5-001 remediation (code commit 3c3ce0e) added `TestRunRouterRun_RealSIGHUP_DoesNotExit` to `cmd/switchboard/main_test.go`. Pass 6 O-P6-001 noted this file was not an explicit FCL row; pass 6 adjudicated it informational. Pass 7 re-examines this against the F-P2-004 precedent: the F-P2-004 finding established that test files added during adversarial remediation must land in the story's File-Change-List — it was a full finding (not an observation) and was remediated in story v1.1. The same class applies here.

**Why pass-6 adjudicated it informational:** Pass 6 reasoned that the test lives in `router_sighup_test.go` (not `main_test.go`), so no out-of-scope file was modified. This reasoning is correct as to scope — but scope-compliance and FCL completeness are distinct requirements. The FCL documents every file touched by the story's implementation, including files touched during adversarial-remediation commits. `main_test.go` was modified as part of pass-5 remediation; it is not in the FCL.

**Orchestrator lesson:** Pass-5 code fix (3c3ce0e) and story-doc update (FCL row) must land in the same burst. The pass-5 burst recorded the code remediation correctly but did not propagate the FCL row — the pass-6 burst, which was a doc-only pass (story v1.3 → no FCL change), also missed it. The gap cost a streak reset.

**→ FIXED:** Story updated to v1.4 — `cmd/switchboard/main_test.go` row added to File-Change-List (with test name and provenance), Cross-BC Note added for pass-7 O3.

---

## Anti-findings (12)

1. **F-P7-001 FIXED (story v1.4)** — FCL row for `main_test.go` added; Cross-BC Note for O3 anchored.
2. **All pass-1 through pass-6 remediations held** — no regression across any of the 26 prior findings (12 P1, 5 P2, 5 P3, 4 P4, 3 P5 + accepted; 12 P6 anti-findings all confirmed stable).
3. **AC-001 through AC-004 behavioral contracts intact** — no code change at 3c3ce0e since pass 5; all four ACs remain correctly specified and their test assertions remain valid.
4. **nil-config guard (go.md rule 13)** — `NewAccessNode`-class fail-closed constructor default remains intact; no security-perimeter parameter change at 3c3ce0e.
5. **cap-1 channel semantics correct** — `make(chan os.Signal, 1)` drop-on-full guarantee confirmed; no concurrent-SIGHUP race angle newly surfaced.
6. **E-CFG-003-reload coverage** — `Validate()` on reload path covers E-CFG-003 structurally; AC-002 integration test exercises the failure path; no coverage regression.
7. **Code lane perimeter** — diff between 8e159f2 and 3c3ce0e confirms only `router_sighup_test.go` and `mgmt_wire_test.go` touched; `main_test.go` touched between pre-pass-5 and 3c3ce0e (pass-5 remediation); no unintended production surface expansion.
8. **POL-001 compliant** — pass-7.md authored in canonical adversary/ subdirectory with complete frontmatter.
9. **POL-002 compliant** — story v1.4 changelog row present and accurate; no undocumented version drift.
10. **POL-004 compliant** — code lane SHA pinned in frontmatter; perimeter verified; no scope drift from ACs.
11. **O-P6-001 class resolved** — the observation from pass 6 (FCL implicit vs explicit) is now closed by F-P7-001 remediation.
12. **Cross-BC O3 anchored** — EC-003 vs E-CFG-001/E-CFG-003 rendering nuance from pass-7 O3 added to story as Cross-BC Note; correctly deferred to S-7.04-FU-PE-CONNECTOR elaboration; no action required here.

---

## Observations (non-findings)

**O-P7-001 (informational):** The `IdempotentResend` test (P3 F-005a remediation) window is not strictly discriminating — the test asserts non-duplication over a time window that may be shorter than the reload round-trip under load. This is noted but generates no action: (a) the window is sufficient for integration-test determinism; (b) streak test-churn during pass history makes this a known stable trade-off; (c) no new behavioral angle surfaces.

**O-P7-002 (informational):** Task-3 sketch prefix in the story body reads `case <-sighupCh:` followed by a `// TODO` comment that was correct at v1.0 but is now cosmetically resolved-in-effect by F-P4-001 (actual implementation replaces the sketch). The sketch in the story is a design reference, not a code obligation. No fix required.

---

## Finding Decay Trajectory

| Pass | Novelty | Findings | Correctness |
|------|---------|----------|-------------|
| P1 | HIGH | 12 | 0 correctness |
| P2 | MED | 5 | 0 correctness |
| P3 | MED | 5 | 0 correctness |
| P4 | LOW | 4 | 0 correctness |
| P5 | MED | 3 | 0 correctness |
| P6 | LOW | 0 | — |
| P7 | LOW | 1 (doc/process-gap) | 0 correctness |

Streak: 1/3 → **0/3 (reset)**. P8 required.

Orchestrator lesson recorded: doc-class remediations must land in the same burst as their code fix. The pass-5 code fix (3c3ce0e) should have included the FCL row for `main_test.go` in the same burst; deferring it to story-doc-only bursts allowed the gap to persist two clean-pass attempts before re-classification as a finding.
