---
document_type: lessons-learned
level: ops
version: "1.1"
status: in-progress
producer: state-manager
timestamp: 2026-06-27T00:00:00Z
cycle: cycle-1
inputs: [STATE.md]
input-hash: "455be0a"
traces_to: STATE.md
---

# Lessons Learned — cycle-1

## Process-Level

1. **Adversarial lenses must include explicit nil-safety / panic-path sweep for constructor-injected dependencies** — S-W3.05 passes 07-09 achieved CONVERGED at 5c3d7ea, but all three lenses (spec-conformance/anti-tautology, concurrency/memory-bounds, integration/RouteFrame) missed a reachable nil-logger deref panic in `NewFailureCounter` (CWE-476 / SEC-001, HIGH). The finding was caught by the security-reviewer during PR #16 review — not by the per-story adversary passes. Root cause: none of the three lenses explicitly targeted constructor precondition completeness or nil-safety of injected dependencies. The streak reset required three additional passes (10-12) at the fixed tip f6038d2. Going forward, at least one adversary pass per story should apply a dedicated nil-safety / panic-path lens covering every constructor-injected parameter.
   _Discovered: PR #16 security review, 2026-06-27. Streak reset: passes 07-09 superseded; re-converged at passes 10-12._

## Policy Candidates

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 1 | Require nil-safety / panic-path adversary lens for constructor-injected dependencies | Per-story adversarial review lens selection | proposed |

## Phase-5 Deferred Items

### TaskList #117 — ARCH-04 + error-taxonomy modified-list monotonicity (routed 2026-06-30)

**Source:** S-6.06 Pass-26, lens-3 observations O-P26L3-001 + O-P26L3-002. Adjudicated out-of-perimeter for S-6.06 per-story scope per BC-5.39.002 PC2 (architectural / system-level).

**O-P26L3-001:** ARCH-04.md lines 30-40 modified-list non-monotonic — missing version entries v1.7, v1.8, v1.11, v1.12 and v1.13 appears before v1.9 in the list.

**O-P26L3-002:** error-taxonomy.md lines 9-23 modified-list mixed ascending/descending ordering.

**Action:** Phase-5 adversarial refinement should sweep ARCH-04 and error-taxonomy modified-list ordering as part of the spec consistency pass. Both documents have accumulated version entries out of order through successive narrowing fix-bursts. Recommend architect agent sorts modified-list entries by version number ascending in both files during the next spec-evolution burst.

---

## Phase-7 Lessons (2026-07-06)

2. **Mutation sampling must run in an isolated worktree/clone, never a shared checkout.** [codified] Running cargo-mutants against a shared checkout risks cross-contaminating the working tree, breaking concurrent test runs, and producing misleading survivor counts from uncommitted mutations being visible to other processes. Codified in `cycles/cycle-1/phase-6/secscan-lane-report.md` §Incident.

3. **Adjudication-style burst reports must be machine-derived from committed artifacts, not memory.** [codified] Three instances this cycle where a burst report asserted aggregate counts (VP coverage, green-claim status, arch-lane idle) that contradicted the actual artifact state at commit time. Root cause in each: the reporting agent synthesized from working-memory rather than re-running the grep/count at the committed SHA. Filed upstream as comment on drbothen/vsdd-factory#513. Going forward: every aggregate in a burst report must cite the command and commit SHA used to derive it.

4. **VP `proof_method` and API citations need a spec-steward review pass before Phase 6.** Skeleton-vs-shipped-API drift is a Phase-6 blocker: VP-028/VP-029 referenced a phantom `Config.Validate()` API shape; VP-056/VP-062 referenced phantom helpers. Discovery during formal hardening forces unplanned spec-side bursts that break Phase-6 timing. A targeted spec-steward sweep of all VP `proof_method` fields and any API symbol referenced in a VP against the actual codebase at Phase-5-exit HEAD would surface these cheaply.

5. **Declared-divergence protocol works — first post-CONSOLE-OBS instance confirmed.** [codified] The Phase-6 fuzz-lane two-commit deviation was declared rather than silently absorbed into a single commit message. The protocol held without coordinator intervention. Codified upstream as drbothen/vsdd-factory#521.

6. **Drift rows citing "issue pending" go stale silently.** Two instances this cycle: WAVE-GATE-DISPATCH-INTEGRITY cited "drafted" for months before it was actually filed as drbothen/vsdd-factory#448; DRIFT-P5P5-TEST-CITATION-VERSION-FLOOR was marked "pending" before filing as drbothen/vsdd-factory#471. Pattern: the row author knows the filing is imminent and uses "pending" or "drafted" as a placeholder; the filing then happens in a different burst; the row is never updated. Fix: at row creation time, either file the issue immediately (preferred) or mark the row explicitly as `upstream-filing-pending` with a follow-up obligation in the current burst's checklist.

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 2 | Mutation sampling must use isolated worktree/clone | Phase-6 pre-flight checklist | proposed |
| 3 | Aggregate counts in burst reports must be machine-derived with SHA citation | Per-burst report discipline | proposed |
| 4 | Spec-steward VP API-symbol sweep before Phase-6 dispatch | Phase-5-exit gate | proposed |
| 5 | Declared-divergence protocol is validated; document as mandatory | Phase-6 multi-commit policy | codified |
| 6 | Drift rows must not use "pending"/"drafted" placeholder without same-burst filing obligation | Drift row authorship | proposed |

---

## PATTERN-CLOSE-P21-P25 — Sibling-sweep policy now reliable (2026-06-30)

**Source:** S-6.06 Passes 21–28 convergence trajectory.

**Pattern observed:** PROCESS-GAP-P21 through PROCESS-GAP-P25 (passes 19–25) represented seven consecutive recurrences of the same sibling-sweep gap on successive axes: BC body VP table → EC tables → story Error Code Map → story Task Refs → VP downstream cites → impl source comments → story body upstream-version cites. Each pass found a new surface the previous fix-burst had not swept.

**Resolution:** The exhaustive upstream-rooted sweep rule codified after PROCESS-GAP-P25 — "any document citing an artifact must be re-grepped when that artifact's version bumps" — held cleanly across Passes 26, 27, and 28 with zero recurrence. The three-consecutive-clean-pass streak was achieved without a single new sibling-sweep finding.

**Lesson crystallized:** The sibling-sweep policy introduced from PROCESS-GAP-P21 through P25 is now empirically validated as sufficient. The pattern closed cleanly at Pass-26 and remained stable through P27/P28. The seven-recurrence sequence was the cost of discovering all axes of the gap; the policy cost is now reduced to a mechanical grep sweep per fix-burst.

**Recommendation:** Codify the upstream-rooted sweep rule as a mandatory step in the story-writer and product-owner fix-burst checklist. The adversary now has high confidence this class of gap will not recur.

---

## PE-CONNECTOR Adversarial Cycle (2026-07-07)

7. **Census/ledger sweeps must re-derive SET MEMBERSHIP from the toolchain (`go list`), not just verify arithmetic and per-row content of rows already in the table.** [codified] P8 applied a full-artifact arithmetic sweep that confirmed all row counts, cross-references, and per-row content in ARCH-08 §6.5 — and issued a class-closing claim ("no further discrepancies"). P9 falsified it with a one-liner: `go list ./internal/... @ 62e38d3` returned 23 packages; the census table had 22 rows, omitting `internal/bench` (shipped in PR #109 cd67394). Root cause: the failure axis is orthogonal to arithmetic and content. A table can be internally consistent, arithmetically correct, and cross-reference-accurate while still missing an entry — because the verification never asked "are all packages in the codebase represented?" but only "are the packages in the table correctly described?" Class-closing claims on census/ledger artifacts require the generating command to be re-run at the anchor SHA, not the artifact to be self-consistent. This extends and tags alongside the renumber-neighbor lesson from P7/P8.
   _Discovered: adversary pass 9, 2026-07-07. Streak reset: P8 class-closing claim falsified; streak reset to 0/3. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 7 | Census/ledger class-closing claims require re-running the generating command at anchor SHA, not self-consistency verification | Adversarial review checklist for registry/census artifacts | [codified] |
| 8 | Symbol deletion/rename remediations must include a mechanical same-artifact co-reference grep with per-hit adjudication | Remediation dispatch checklist for symbol deletion/rename | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #9 (2026-07-07)

9. **Negative (absence) assertions must key on spec-anchored event codes, not prose fragments of the emission string.** [codified] A space-vs-hyphen typo (`"split-horizon blocked"` vs. production `"split-horizon-blocked"`) made an AC-004 "must NOT fire" assertion vacuously true for the entire eleven-pass cycle. Ten passes (P1–P10) checked positive-emission polarity exclusively — none examined whether the key in an absence assertion could actually match the production string. Root cause: absence assertions are structurally harder to falsify than presence assertions; a presence assertion that miskeys will produce a test failure, but an absence assertion that miskeys will silently pass regardless of runtime behavior. Standing bar: (1) every absence assertion must use a spec-anchored event code (e.g., `"E-FWD-001"`) rather than a prose fragment of the emission string, so that minor wording changes cannot silently break the key; (2) every absence assertion gets a companion pin test that embeds the verbatim production emission string and proves the search key matches it (non-vacuousness proof) AND that the defect shape (incorrect key) does not match it.
   _Discovered: adversary pass 11, 2026-07-07. AC-004 absence assertion key `"split-horizon blocked"` (space) never matched production `"split-horizon-blocked"` (hyphenated) — vacuous for entire cycle. Codified immediately via companion pin test `TestScanForLine_DetectsEFWD001ProductionEmission` (6e00332)._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 9 | Absence assertions must key on spec-anchored event codes; companion pin tests prove key matches verbatim production emission | Adversarial review + test-writer checklist for absence assertions | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #10 (2026-07-08)

10. **Remediation commits must re-clear the FULL local CI gate (golangci-lint run ./..., go vet, race tests, gofumpt) before a pass is declared remediated — not just the test that motivated the fix. Also: prefer stable mechanism anchors (function/guard/branch descriptions) over line-number citations in test comments — line numbers rot on every unrelated edit to the cited file.** [codified] The P11 fix commit introduced an `errcheck` violation that made the branch unmergeable: three `buf.Write` calls in the new `TestScanForLine_DetectsEFWD001ProductionEmission` pin test were unchecked. Eleven prior passes had validated feature/fix commits against the full CI gate, but the convention had never been explicitly applied to remediation commits specifically — the state-manager declared P11 remediated after confirming test correctness and race cleanliness, without running golangci-lint. Root cause: the remediation checklist was implicitly scoped to "does the fix work?" rather than "does the branch merge-clean?" The full CI gate (`golangci-lint run ./...`, `go vet ./...`, `go test -race ./...`, `gofumpt -l .`) must be run against every commit — feature, fix, or remediation — before a pass is declared closed. The same burst also retired seven stale line-number citations across two files (`connector.go` and `mgmt_wire.go`); these had accumulated across prior passes as the line numbers drifted without test failures, invisible until the lint gate caught the new violation and prompted a broader sweep. Prefer stable anchors (function name, guard condition, branch description) over line numbers in test comments — they survive refactoring without becoming stale.
   _Discovered: adversary pass 12, 2026-07-08. P11 fix commit d882686 introduced errcheck violation; branch unmergeable until 14ae327 closed the gap. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 10 | Remediation commits must re-clear the full local CI gate (golangci-lint, vet, race tests, gofumpt) before declaring a pass remediated; prefer stable mechanism anchors over line-number citations in test comments | Remediation checklist + test-writer guidance | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #11 (2026-07-08)

11. **Remediation output is itself un-reviewed input: an anchor-stabilization fix confabulated a phantom symbol that existed nowhere in the repo — strictly worse than the stale line number it replaced.** [codified] The P12 citation-stabilization task replaced five stale `mgmt_wire.go` line-number citations in `router_sighup_test.go` with stable mechanism anchors. The replacement text cited `"runRouter/buildAndWireConnector"` — a function that does not exist in the codebase. Connector construction logic is inline in `runRouter` at `mgmt_wire.go:408`; there is no `buildAndWireConnector` anywhere. The stale citation at least pointed to a real line (albeit the wrong one). The phantom citation points to nothing. Root cause: the remediation agent treated "anchor stabilization" as a creative naming task rather than a verification task — it named what the code ought to do rather than what it actually does. Standing bar: every symbol cited in an authored comment or anchor must be grep-resolved against the codebase before the commit is declared remediated. A symbol-resolution table (`symbol → file:line`) must appear in the remediation report. If grep returns zero hits, the symbol is phantom and must not be cited.
   _Discovered: adversary pass 13, 2026-07-08. P12 commit 14ae327 cited `buildAndWireConnector` which has 0 grep hits in the entire repo. Fixed at 0a350d6. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 11 | Every symbol cited in an authored comment or anchor must be grep-resolved before commit; symbol-resolution table required in remediation report | Remediation checklist; adversarial review bar | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #8 (2026-07-07)

12. **Deferral/shipped claims in code doc headers must be re-adjudicated by the story that changes their truth value.** [codified] A doc header authored accurately at base ("PE connector ships in a follow-on story") became false the moment this story wired the connector. Passes P1–P13 applied citation/symbol/census bars that could not catch it because the claims were semantically false rather than referentially broken — all cited symbols resolved, all line numbers were accurate, the prose was simply wrong. When a story ships something a comment declares deferred, the wiring commit must update the claim in the same change. This is a new bar orthogonal to all prior adversarial bars: "does the prose accurately describe the current state of the codebase after this story lands?"
   _Discovered: adversary pass 14, 2026-07-08. mgmt_wire.go runRouter doc header inherited from base; corrected at 34e51d6. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 12 | Deferral/shipped claims in doc headers must be re-adjudicated when the story changes their truth value; wiring commit must update the claim in the same change | Adversarial review checklist for doc headers; remediation checklist | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #13 (2026-07-08)

13. **Test setup that wires a double must be observed before anything discards it — an orphaned fake satisfies the reader but not the runtime, and the assertion may pass via an unrelated mechanism.** [codified] `TestRunRouter_PE_RouterHandleModeReflectsLiveState` wired `fakeConnE` as an inverse-delegation double to verify ModeE behavior. However, `Restart` discarded the fake before the assertion ran; the final assertion was satisfied by the live connector's failed dial (EC-001 ctx-cancel path). The comment truthfully described the intended verification but falsely described the actual runtime path. The mutation proof was definitive: flipping `fakeConnE`'s return value produced no test failure because the fake was never consulted. This is the third shape in the "comment claims a code path the test doesn't exercise" family (P11 — vacuous absence key; P13 — phantom symbol in anchor text; P15 — orphaned fake + misattributed mechanism). Standing bar extended: for every test double wired, verify an assertion consumes it before teardown/restart, and prove liveness by mutation — flip the double's value; the test must fail.
   _Discovered: adversary pass 15, 2026-07-08. fakeConnE wired but discarded by Restart before assertion; final assertion satisfied by live failed-dial. Fixed at 79c1284: mutation-pinned inverse-delegation assertion added. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 13 | Every wired test double must be observed before teardown/restart; liveness proven by mutation (flip value → test must fail) | Test-writer checklist + adversarial review bar for test doubles | [codified] |

---

14. **Symbol resolution is necessary but not sufficient for comment accuracy: a doc comment that cited a REAL method (`ReloadAddrs`) which the documented path never invokes passes grep-resolution but is still false. The stronger bar is claim→code mapping: every behavioral sentence in an authored doc comment must be traceable to the specific code lines that implement it, verified at authoring time.** [codified] `testenv.go` `Restart`'s doc comment described a control path ("already wired → ReloadAddrs reuse"; "in both cases polls") that the body never takes — the body unconditionally tears down and recreates the connector, with an empty-upstreams early-return that exits before any reuse. `ReloadAddrs` exists in the codebase and grep-resolves, so the symbol-resolution bar passes. Only claim→code mapping catches it: tracing each behavioral sentence to the specific code line that implements it revealed that no line in `Restart` calls `ReloadAddrs`, and no line implements conditional reuse. This is the fourth shape in the cycle's comment-vs-code-path family (P11 vacuous key; P13 phantom symbol; P15 orphaned fake; P16 real-symbol-never-invoked). The progression shows a narrowing gap: each prior shape was catchable by a progressively weaker mechanical bar; this shape requires full behavioral tracing. Going forward: when authoring or reviewing a method doc comment, produce a claim→code mapping row for every behavioral sentence, citing the specific line(s) that implement the claim. If no line implements the claim, the sentence is false and must be removed or corrected.
   _Discovered: adversary pass 16, 2026-07-08. `testenv.go Restart` doc comment cited `ReloadAddrs` which grep-resolves; the described path is never taken. Fixed at 7daed41 (doc-only). Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 14 | Every behavioral sentence in an authored doc comment must be traceable to specific code lines via claim→code mapping; symbol-resolution is necessary but not sufficient | Doc-comment authorship + adversarial review bar | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #15 (2026-07-08) [codified]

15. **Red-gate test suites annotate expected post-implementation behavior in comments ("After AC-NNN: ..." / "Stub: ..."); the green/TDD implementation pass reconciles assertions but not caller-side comments. When a remediation ratifies a mechanism (e.g., P16's Restart teardown-recreate), sweep the RATIFIED CLAIM'S negation across the full perimeter — not just the file being fixed.** [codified] `TestE2E_EtoPEGraduationByConfigChange` carried a forward-looking "After AC-006: calls `connector.ReloadAddrs()`" annotation authored at the red-suite commit (d3bac4c, when the stub was live). The green-gate pass wired `upstreamdial.New` and retired the stub, but no sweep checked whether caller-side "after this story" annotations had been reconciled. Similarly, "Stub: sets `r.mode=ModePE` unconditionally, dials nothing." was authored at red-gate and became false at green-gate. P16's remediation ratified the Restart-teardown-vs-SIGHUP-seam-ReloadAddrs division; that ratification was the signal to sweep for every comment claiming the opposite. **Mechanical:** at green-gate close AND at any mechanism-ratifying remediation, grep `// Stub:` and `// After AC-` (and equivalent "ships-later" markers) across all perimeter files. Each hit is a candidate stale annotation; adjudicate live-prose (fix) vs changelog/historical (preserve). This class is now confined: the P17 sweep returned only these two lines across all 7 core perimeter files, confirming the class is retired for this story.
   _Discovered: adversary pass 17, 2026-07-08. Authored at d3bac4c (red-suite), survived green-gate through cee8e8b, never reconciled. Fixed at 7c6d841. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 15 | At green-gate close and at any mechanism-ratifying remediation, sweep `// Stub:` and `// After AC-` markers across all perimeter files; each hit is a candidate stale annotation requiring adjudication | Implementer + adversary checklist; green-gate close protocol | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #16 (2026-07-08) [codified]

16. **Class-closure sweeps must enumerate the ORTHOGRAPHIES of the class, not just its canonical spelling: P12 closed "line-number citations" by grepping the prefixed form `file.go:NNN`; the bare form `:NNN` (filename implied by table context) survived 7 more passes in the same document. When closing a textual defect class, first ask "how many ways can this class be spelled?" and sweep each; record the enumerated forms in the closure claim.** [codified] P12 issued a class-closing claim ("line-citation class structurally closed") for line-number citations. The sweep covered `file.go:NNN` (the most visible spelling) but not the bare `:NNN` form, which appears in table cells where the filename is implied by the column header. The bare form was used in FCL row 1 of the story (the table column header names the file). Four such citations survived P12's sweep through P18 — 7 additional passes — because nothing in the closure process enumerated and tested the second spelling. The bare citations were also stale (shifted by P14's ctx-first refactor), but staleness was not the reason they survived: even if accurate, they would have been invisible to P12's closure grep. The root cause is that "closing a class" implicitly committed to a single syntactic form. The fix is systematic: every class-closure claim must list ALL orthographies of the class it purports to close, run a separate grep for each, and record them in the closure claim so future passes can independently verify completeness.
   _Discovered: adversary pass 19, 2026-07-08. Four bare `:NNN` citations in story FCL row 1 survived P12 class-closure through P18. Story v1.18 both-orthography sweep closes the class. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 16 | Class-closure sweeps must enumerate all orthographies of the class and sweep each; record enumerated forms in the closure claim | Adversarial review + remediation checklist for textual defect classes | [codified] |

---

## PE-CONNECTOR Adversarial Cycle — Lesson #17 (2026-07-08) [codified]

17. **Remediation dispatches must PIN the adjudicated finding classification (severity + class verbatim from the pass ledger) in the dispatch text — an author left to self-assess produced a changelog row disagreeing with 15 sibling statements on both axes. The record of a fix is part of the fix; classification strings in remediation artifacts are copied from the ledger, never re-derived.** [codified] Story v1.18 was dispatched to remediate F-P19-001 without the classification pinned in the dispatch text. The author re-derived the classification as "MED [doc-drift]" — a plausible answer for a citation-drift finding, but wrong. The adjudicated classification in the ledger, STATE.md, and STORY-INDEX is "LOW [process-gap]" (15 statements vs 1). The discrepancy was caught by pass 20's fresh-context adversary. Root cause: the dispatch text said "reconcile the bare-form citations" but did not say "the finding is classified LOW [process-gap]". The author chose a classification that felt appropriate, producing a record inconsistent with the authoritative source. Standing bar: every remediation dispatch must include the verbatim adjudicated classification from the pass ledger — "F-NNN-001 LOW [process-gap]" or "F-NNN-001 MED [doc-drift]" — so that the author copies it, not re-derives it. The classification of a finding is settled at adjudication time; it is not open for re-interpretation by the remediation author.
   _Discovered: adversary pass 20, 2026-07-08. Story v1.18 changelog row classified F-P19-001 as "MED [doc-drift]" while 15 sibling statements record "LOW [process-gap]". Corrected at story v1.19. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 17 | Remediation dispatches must include verbatim adjudicated classification (severity + class from the pass ledger); authors copy, never re-derive | Remediation dispatch checklist + state-manager protocol | [codified] |

---

18. **When a function's comments have produced TWO defects across the adversarial cycle, the THIRD fix must sweep the ENTIRE function's comments — not just the flagged line. The unit of sweep for comment defects is the function, not the line.** [codified] `RouterHandle.Restart` produced three separate comment defects across six passes: P16 fixed the doc header; P17 fixed the caller-side red-gate annotation; P22 fixed the inline poll-tail. Each fix reconciled only its flagged fragment, each time leaving adjacent comments unexamined. The P22 finding was doubly wrong (unreachable AND outcome-inverted), which could only be discovered by tracing the full function. Root cause: the escalation signal — two prior defects in the same function — was never applied as a sweep trigger. Standing rule: when dispatching a fix for the second comment defect in a function, the remediation must include an explicit per-comment adjudication of ALL comments in that function, not just the flagged one. This extends the full-surface-sweep lesson (previously applied to documents) down to function-comment granularity: the unit of sweep is the function, not the line.
   _Discovered: adversary pass 22, 2026-07-08. Third consecutive Restart-internal comment defect; full-function sweep at 4f2807c closed the class. Codified immediately._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 18 | When a function's comments have produced 2+ defects, the next fix must sweep ALL comments in that function with per-comment adjudication | Remediation dispatch checklist + adversarial review bar | [codified] |

---

8. **Symbol deletion/rename remediations must include a mechanical same-artifact co-reference grep with per-hit adjudication (live prose = fix; struck-through/changelog = preserve) at remediation time.** [codified] One helper deletion (F-P1-007, `upstreamRoutersAsSet`) produced four straggler findings across passes 7-10 because each fix swept only its primary location. Pass-7 swept ARCH-08 §6.5 and testenv self-doc. Pass-8 swept ARCH-08 arithmetic and content. Pass-9 swept ARCH-08 census membership via toolchain re-derivation. Pass-10 found three live story references — AC-001 postcondition 5 citing the helper as normative mechanism, the test-mapping row naming it as unit-under-test, and the FO-1 resolution column citing it in present tense — all in the same story file that was modified in P1 and P3. Root cause: each remediation dispatch named a specific defect location; none included an instruction to grep the entire artifact (or all artifacts mentioning the symbol) for co-references. The sweep is now mandatory: when dispatching a symbol-deletion or rename remediation, include a step: "grep the full artifact (and sibling artifacts) for every variant of the symbol name; adjudicate each hit as live-prose (fix), historical-record/changelog (preserve), or struck-through (preserve). The sweep converts O(passes) straggler discovery into O(1)."
   _Discovered: adversary pass 10, 2026-07-07. Triggered by F-P1-007 deletion of `upstreamRoutersAsSet` helper; four stragglers across P7/P8/P9/P10. Codified immediately._

---

## S-7.04-FU-DRAIN-WIRE Delivery — Lessons #19-21 (2026-07-12) [codified]

19. **Spec-adversarial passes need an execute-the-discharge-trace-against-baseline obligation — internal-consistency review, however exhaustive, cannot catch a false premise buried in a runtime object graph the text never names.** [codified] Twelve consecutive text-based spec-adversarial passes on S-7.04-FU-DRAIN-WIRE (passes 1-12, three distinct traversal angles — ledger-first, code-first, obligations-first) converged CLEAN on internal consistency, cross-reference accuracy, and obligation-to-landing-site tracing. None of the twelve traced `ingressCtx`'s PARENT. The spec's entire Shutdown Ordering Guarantee, ratified across v1.3-v1.9, rested on the unstated premise that `ingressCtx` was independent of the caller's own cancellation — it was in fact `context.WithCancel(ctx)`, a cancel-linked child, so the caller's `cancel()` closed every connection ~140µs before the shutdown flush pass ever ran, defeating the guarantee's entire purpose. The defect was invisible to every text-based pass because it lived in a runtime relationship (parent-of-context) that no spec sentence asserted or denied — there was nothing inconsistent to find, only something unverified. It surfaced only at the implementer's first empirical contact (RED tests that could not be made to pass). Root cause: the engine's spec-adversarial methodology verifies documents against each other and against a static code snapshot, but has no step that executes a claim's discharge trace against the running system to confirm a premise the prose merely assumes. Filed upstream as **drbothen/vsdd-factory#620** (HIGH). No product-repo story warranted — this is an engine methodology gap, not a switchboard defect.
    _Discovered: implementer first empirical contact (F-DW-IMPL-001), S-7.04-FU-DRAIN-WIRE post-convergence reopen, 2026-07-12. Twelve prior text-based passes (1-12) all missed it. Remediated at placement-note v1.10 (`context.WithCancel(context.WithoutCancel(ctx))`). Codified via upstream filing #620._

20. **Concurrency-ordering remediations that close one race must enumerate the join-obligations of their OWN fix in the same pass — otherwise the fix becomes the next pass's finding.** [codified] The drain-wire shutdown-ordering sequence produced a chain of five consecutive spec-adversarial passes (F-DW-SP3-005 → SP4-001/004 → SP5-001 → SP6-001 → SP7-001) where each remediation closed the race it was dispatched to fix while opening or leaving open a *new* race or join-gap in the mechanism it introduced — SP6-001's snapshot-scoped flush wait itself spawned N+1 unwaited helper goroutines, caught only by SP7-001 the following pass. Each individual remediation was locally correct; the class kept recurring because no remediation dispatch required the fixer to enumerate what THEIR OWN new mechanism now owed a join to. Standing gap: concurrency-ordering fixes need a mandatory "list every goroutine/channel/waitgroup this fix introduces, and confirm each is joined or explicitly justified as unjoined" step inside the SAME remediation that closes the original finding — not deferred to the next adversarial pass to discover. Filed upstream as **drbothen/vsdd-factory#621** (MED).
    _Discovered: retrospective synthesis while writing up the drain-wire delivery arc, 2026-07-12 — the five-pass relocation chain (SP3-005 through SP7-001) was known individually but not previously named as a single systemic gap. Codified via upstream filing #621._

21. **Line-number citations in spec documents need a stated coordinate-baseline convention (baseline-relative vs landed-tree-relative) — without one, a correct fix reads as citation drift.** [codified] A fresh-context delta-verification pass confirmed the v1.10 `ingressCtx` fix SOUND on all six checks against the landed feature-branch tree, but flagged F-DW-DV-001 (LOW): the placement note's and story's line-number citations were pinned relative to `develop@ef1ee1e` (the pre-reopen baseline), while the fix itself landed at `mgmt_wire.go:523` on the feature branch — a citation of `:471` against the landed tree reads as wrong even though it was correct relative to its stated (but unstated-as-a-convention) baseline. This is the second instance of the line-number-citation-coordinate-system problem in this cycle (the first was the P12/P19 orthography-closure lessons — #16 above — which addressed spelling forms, not coordinate systems). Remediated locally via the minimal-fix option: a citation-convention blockquote at the top of both documents stating line-number citations are baseline-relative to `develop@ef1ee1e`, not the landed feature branch — mechanism anchors remain the authoritative locator, not the line number. Filed upstream as **drbothen/vsdd-factory#622** (LOW) for the engine-level template fix (a standard citation-convention header in the placement-note and story templates); the local remediation pattern (convention blockquote, not per-commit re-pinning) is recommended as the template default.
    _Discovered: delta-verification adversary pass, S-7.04-FU-DRAIN-WIRE post-convergence reopen, 2026-07-12. Remediated at placement-note v1.11 + story v1.11 (citation-convention blockquote). Codified via upstream filing #622._

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 19 | Spec-adversarial cycles require an execute-the-discharge-trace-against-baseline obligation to catch false premises in runtime object graphs, not just internal-document consistency | Spec-adversarial review checklist; filed drbothen/vsdd-factory#620 | [codified] |
| 20 | Concurrency-ordering remediations must enumerate the join-obligations of their own new mechanism in the same pass that closes the original finding | Remediation dispatch checklist for concurrency findings; filed drbothen/vsdd-factory#621 | [codified] |
| 21 | Spec documents need a stated line-number citation coordinate-baseline convention (baseline-relative vs landed-tree-relative), stated once via a convention blockquote rather than per-commit re-pinning | Spec-doc template (placement-note, story); filed drbothen/vsdd-factory#622 | [codified] |

---
