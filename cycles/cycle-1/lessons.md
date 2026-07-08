---
document_type: lessons-learned
level: ops
version: "1.1"
status: in-progress
producer: state-manager
timestamp: 2026-06-27T00:00:00Z
cycle: cycle-1
inputs: [STATE.md]
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

8. **Symbol deletion/rename remediations must include a mechanical same-artifact co-reference grep with per-hit adjudication (live prose = fix; struck-through/changelog = preserve) at remediation time.** [codified] One helper deletion (F-P1-007, `upstreamRoutersAsSet`) produced four straggler findings across passes 7-10 because each fix swept only its primary location. Pass-7 swept ARCH-08 §6.5 and testenv self-doc. Pass-8 swept ARCH-08 arithmetic and content. Pass-9 swept ARCH-08 census membership via toolchain re-derivation. Pass-10 found three live story references — AC-001 postcondition 5 citing the helper as normative mechanism, the test-mapping row naming it as unit-under-test, and the FO-1 resolution column citing it in present tense — all in the same story file that was modified in P1 and P3. Root cause: each remediation dispatch named a specific defect location; none included an instruction to grep the entire artifact (or all artifacts mentioning the symbol) for co-references. The sweep is now mandatory: when dispatching a symbol-deletion or rename remediation, include a step: "grep the full artifact (and sibling artifacts) for every variant of the symbol name; adjudicate each hit as live-prose (fix), historical-record/changelog (preserve), or struck-through (preserve). The sweep converts O(passes) straggler discovery into O(1)."
   _Discovered: adversary pass 10, 2026-07-07. Triggered by F-P1-007 deletion of `upstreamRoutersAsSet` helper; four stragglers across P7/P8/P9/P10. Codified immediately._
