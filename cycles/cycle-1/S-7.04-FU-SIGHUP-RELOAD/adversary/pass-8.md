---
artifact_id: S-7.04-FU-SIGHUP-RELOAD-adversary-pass-8
document_type: adversarial-review
story_id: S-7.04-FU-SIGHUP-RELOAD
pass: 8
verdict: HAS_FINDINGS
novelty: MED
code_lane_sha: 3c3ce0e
story_version: "1.4"
reviewer_model: fresh-context
timestamp: 2026-07-07T04:00:00Z
---

# Adversarial Review — S-7.04-FU-SIGHUP-RELOAD Pass 8

## Summary

**Verdict:** HAS_FINDINGS  
**Novelty:** MED  
**Code lane SHA:** 3c3ce0e (reviewed at pass-8); **remediation SHA:** fa97154  
**Story version reviewed:** v1.4  
**Streak:** HOLDS 0/3

2 findings (both LOW, test-strength class). 4 observations. 12 anti-findings. Novelty MED — F-P8-001 contradicted a prior anti-finding with concrete evidence, demonstrating fresh-context value.

---

## Findings

### F-P8-001 [LOW] [test-strength] cfg-immutability-asserts-vacuous

**Class:** vacuous-assert — input class covers only the empty pole.

**Evidence:** The immutability assertions for the `upstreamRouters` field in the reload path (both PEtoPE and PEtoE branches) operated on a freshly-initialized empty slice. The test code constructed a cfg with `UpstreamRouters: nil` or `[]string{}` and then asserted that the field in the reloaded config was still empty after reload. This proves only "empty stayed empty" — it does not verify that a deep copy was produced (i.e., that mutations to the original slice's backing array cannot propagate through the copy).

The load-bearing check for immutability is: start with a non-empty slice, trigger reload, mutate the original, verify the copy is unaffected. None of the four assertions at 3c3ce0e exercised this pattern.

**Contradiction of prior anti-finding:** Pass-7 anti-finding 3 stated "AC-001 through AC-004 behavioral contracts intact — no code change at 3c3ce0e since pass 5; all four ACs remain correctly specified and their test assertions remain valid." The assertion-validity claim was structurally correct but vacuously true: the tests passed because empty-to-empty is always verified regardless of copy semantics. This finding contradicts the adequacy inference in that anti-finding with evidence.

**Why this was not caught earlier:** Passes 1–7 examined whether the assertions *existed* and whether they *passed*. A fresh-context pass without knowledge of the prior anti-finding examined what the assertions actually *proved*. The information-asymmetry property of adversarial review produced the finding.

**→ FIXED fa97154:** PEtoPE and PEtoE reload branches gain non-empty deep-copy assertions: a non-empty upstream_routers slice is constructed pre-reload, reload is triggered, the original slice element is mutated post-reload, and the assertion verifies the reloaded copy did not reflect the mutation.

---

### F-P8-002 [LOW] [test-strength] EC-003-input-class-untested

**Class:** input class gap — EC-003 "invalid entries" pole not covered through reload path.

**Evidence:** EC-003 specifies: "upstream_routers resolves as empty or contains invalid entries — reload aborts." At 3c3ce0e, the reload test suite covered the empty pole (no upstream_routers entries → reload refused, E-CFG-001 emitted). The invalid-entries pole — specifically a structurally valid config that passes `Validate()` at the outer layer but contains an upstream address that cannot be resolved or dialed — was not exercised through the reload path.

BC-2.09.001 PC-1 specifies E-CFG-001 as the reload-abort signal. The Cross-BC Note added at story v1.4 (for O-P7-O3) correctly defers the literal-code rendering question (what specific E-CFG-001 Detail fragment is emitted for invalid-upstream-addr) to PE-CONNECTOR elaboration. However, the absence of any test for the invalid-upstream-addr input class through the reload path is a separate gap from the rendering question.

**→ FIXED fa97154:** `TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed` added: a reload config carrying an invalid upstream address (structurally valid but not a real host:port pair) is fed through the reload path; the test asserts E-CFG-001 is returned as the outer error, and that the upstream_routers[0].addr fragment appears in the detail. The Cross-BC Note is cited in the test comment; the literal-code rendering question remains parked at PE-CONNECTOR as specified.

---

## Observations (non-findings)

**O-P8-001 (anchored — 4th PE-CONNECTOR forward obligation):** `upstreamRouters` in `runRouter` is currently accessed only on the single goroutine running the main select loop. Under the current architecture (no dial goroutine), this access pattern is race-free. Under PE-CONNECTOR (which adds a dial goroutine that holds a reference to the upstream_routers snapshot during dial), this single-goroutine assumption becomes a data race unless the snapshot is either (a) passed by value at PE-CONNECTOR startup, or (b) access is coordinated. ANCHORED to PE-CONNECTOR elaboration as the 4th forward obligation (prior 3: dial-loop integration, Failed-state trigger, retransmit-send boundary).

**O-P8-002 (informational):** Inert-reload (valid SIGHUP cfg with changes only to non-upstream fields such as drain_timeout or keepalive_interval) was re-confirmed as correctly parked at DRIFT-SIGHUP-INERT-RELOAD-UX with anchor S-BL.CLI-SURFACE-COMPLETION. No new angle surfaces at this pass.

**O-P8-003 (fixed — modeELine comment):** The `modeELine` helper comment described the helper as "matching PE-to-E mode line" but the helper is used to match the canonical mode log line regardless of direction (both E-to-PE and PE-to-E callers). The directional framing in the comment was misleading. FIXED fa97154 (comment reworded to direction-neutral framing — no behavioral change).

**O-P8-004 (adjudicated — goto-shutdown accepted):** The use of `goto shutdown` in the signal-loop to reach the cleanup block at function tail is Go-idiomatic for this pattern. All goto targets point to a single labeled block at the end of the function. No control-flow branching or re-entrance. Consistent with ADR-style "consolidate cleanup at one point." ADJUDICATED-ACCEPTED — no action required.

---

## Anti-findings (12)

1. **F-P8-001 + F-P8-002 FIXED (fa97154)** — both test-strength defects remediated in same burst; PEtoPE + PEtoE non-empty deep-copy asserts present; invalid-upstream-addr reload failure path covered.
2. **All pass-1 through pass-7 remediations held** — no regression across any of the 28 prior findings (12 P1 + 5 P2 + 5 P3 + 4 P4 + 3 P5 accepted/fixed + 1 P7 fixed; all confirmed stable at fa97154).
3. **AC-001 through AC-004 behavioral correctness intact** — 2 findings both LOW test-strength; zero behavioral-correctness findings this pass.
4. **nil-config guard (go.md rule 13)** — `NewAccessNode`-class fail-closed constructor default intact; no security-perimeter parameter change.
5. **cap-1 channel semantics correct** — `make(chan os.Signal, 1)` drop-on-full guarantee confirmed; no concurrent-SIGHUP race angle surfaced.
6. **E-CFG-003-reload coverage extended** — `Validate()` on reload path remains structurally present; fa97154 adds invalid-upstream-addr input class, extending EC-003 pole coverage from {empty} to {empty, invalid-addr}.
7. **Code lane perimeter clean** — fa97154 remediation touches only test files (non-empty deep-copy asserts + invalid-addr reload test + comment correction); no production surface expansion; scope within story perimeter.
8. **POL-001 compliant** — pass-8.md authored in canonical `adversary/` subdirectory with complete frontmatter; artifact_id schema-compliant.
9. **POL-002 compliant** — STORY-INDEX row updated to 8 passes / pass 9 pending; changelog row v3.97 present; no undocumented version drift.
10. **POL-004 compliant** — code lane SHA pinned in frontmatter (3c3ce0e reviewed; fa97154 remediation); perimeter verified; no scope drift from ACs.
11. **O-P7-001 / O-P7-002 stability confirmed** — IdempotentResend window observation and Task-3-sketch observation both remain non-actionable; no new angle surfaces.
12. **F-P7-001 FCL fix (story v1.4) propagated** — `cmd/switchboard/main_test.go` FCL row present in story v1.4; recurrence-of-F-P2-004-class gap closed; no regression.

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
| P8 | MED | 2 (test-strength) | 0 correctness |

Streak: HOLDS **0/3**. P9 required.

Novelty note: MED is warranted because F-P8-001 contradicted a prior anti-finding with evidence. The prior pass confirmed test-assertion validity structurally; fresh context (without that prior assessment) examined what the assertions actually proved and found vacuous coverage. This is the canonical demonstration of adversarial-review information-asymmetry value — the finding is genuine novelty within the test-strength axis, even though no behavioral defect was found.

Orchestrator observation: the vacuous-assert class (empty input as the only exercised pole) is a recurring risk in deep-copy and immutability tests. Consider adding a process-level note that immutability assertions should always include a non-empty input + post-copy-mutation check as part of the test-structure checklist for stories that involve config cloning.
