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
