# Improvement Proposals — cycle-1 (2026-07-12)

Status: awaiting human disposition (72h auto-defer to improvement-backlog.md on 2026-07-15 per session-review skill).

**IP-C1-01** | category: **pattern** | HIGH priority
**Title:** Bound the session-review trigger — 165 unsynthesized sidecar markers over 23h is the loop failing in production
**Evidence:** `.factory/sidecar-learning.md`, 165 markers 2026-07-11T21:08→2026-07-12T20:33, zero synthesis before this run. Local instance of `drbothen/vsdd-factory#584`.
**Proposed change:** Invoke session-review at a bounded marker-count or elapsed-time threshold during steady-state (which has no natural "pipeline complete" moment to hang the existing trigger on), not only at full-pipeline completion. Alternative: surface a visible staleness warning once marker count exceeds a threshold (same shape as the aae-orc orchestrator's own 48h bd-staleness reminder pattern).
**Routes to:** ENGINE — this is a genuine gap in the upstream trigger mechanism for steady-state/incremental delivery paths (F1–F7), which don't have a single terminal "pipeline complete" event the way greenfield does. Cross-reference #584; this run supplies the first concrete measurement (165/23h) for that issue.
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-02** | category: **convergence** | HIGH priority
**Title:** Endorse and prioritize #620 — spec-adversarial methodology needs an execute-the-discharge-trace step
**Evidence:** `lessons.md` #19; 12 consecutive text-based passes on DRAIN-WIRE converged cleanly on a false premise about `ingressCtx`'s parent context, invisible to all three traversal angles because no spec sentence asserted or denied the runtime relationship.
**Proposed change:** No new filing needed — #620 already exists and is HIGH. Recommend this review flag it for priority scheduling upstream, since it represents a structural blind spot (verification against documents/snapshots only, never against a running discharge trace) rather than a one-off miss.
**Routes to:** ENGINE (already filed, drbothen/vsdd-factory#620) — routing is escalation/prioritization, not a new filing.
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-03** | category: **gate** | HIGH priority
**Title:** WAVE-GATE-DISPATCH-INTEGRITY (#448) is still open and was caught by luck, not design
**Evidence:** `STATE.md` open-drift row `WAVE-GATE-DISPATCH-INTEGRITY`, filed 2026-07-02, HIGH severity, local mitigation still reads "target: pipeline-hardening cycle" (unscheduled) as of cycle close.
**Proposed change:** Schedule the local pipeline-hardening cycle this drift row has been waiting on since 2026-07-02, or escalate the upstream fix priority — a wave-gate silent-false-green is possible today if a less-thorough adversary pass had run instead of the one that happened to notice the SHA mismatch.
**Routes to:** ENGINE (upstream #448) + LOCAL (schedule the hardening cycle it's blocked on).
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-04** | category: **quality** | MEDIUM priority
**Title:** Re-run HS-006 holdout evaluation against current develop — the score is stale
**Evidence:** HS-006 evaluated 2026-07-02 at develop tip `7fe3e29`, scored 0.85 partly because the router daemon subcommand was a stub. S-BL.ROUTER-RUNTIME (PR #92, 2026-07-05) shipped the router daemon 3 days later and closed `DRIFT-HS006-ROUTER-DAEMON-STUB`. No re-evaluation has occurred since.
**Proposed change:** Re-run the HS-006 holdout evaluation at current develop tip to get a current, non-stale quality signal — the two PARTIAL-credit clauses (PE graduation live-reload, router-mode drain) may now score full credit.
**Routes to:** LOCAL — trigger a holdout re-evaluation as a maintenance-sweep task.
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-05** | category: **pattern** / **workflow** | MEDIUM priority
**Title:** Extend the machine-derived-aggregate rule to STATE.md phase-summary cells
**Evidence:** STATE.md's Phase 6 summary line cites "53M+ combined execs" for fuzz targets; summing the actual evidence in `fuzz-lane-report.md` + `gapclose-lane-report.md` yields ~40.9M — a ~23% overstatement. `lessons.md` #3 already mandates machine-derived aggregates for burst reports; it was not applied to the STATE.md summary line itself.
**Proposed change:** Extend Lesson #3's rule explicitly to phase-progress-table summary cells in STATE.md, not just burst reports.
**Routes to:** LOCAL — state-manager protocol update (`.claude` / factory dispatch conventions for this project).
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-06** | category: **gate** / **workflow** | MEDIUM priority
**Title:** Concurrent Phase-6 lane dispatch should default to isolated worktrees, not just for mutation sampling
**Evidence:** `phase-6/secscan-lane-report.md` "Incident" — fuzz and secscan lanes shared one checkout; secscan's deliberately-inverted mutation code leaked into the fuzz lane's view. Contained by the fuzz lane's own vigilance, not by structural isolation. `lessons.md` #2 currently scopes the isolation rule only to "mutation sampling."
**Proposed change:** Generalize the isolation rule to any Phase-6 lane whose own verification methodology involves mutating source/test-double state, and require isolated worktrees by default for concurrent Phase-6 lane dispatch.
**Routes to:** ENGINE — Phase-6 dispatch contract (`vsdd-factory:phase-6-formal-hardening`); currently local-session-fixed per `lessons.md` #2 note ("no upstream filing... candidate enhancement only").
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-07** | category: **convergence** | MEDIUM priority
**Title:** Consider a separate streak counter for doc/citation-only findings vs. code-correctness findings
**Evidence:** S-6.06 hit a clean pass at Pass-16 then ground through 8 more passes (17–24) purely on citation/sibling-propagation drift before genuinely closing at Pass-28. PE-CONNECTOR similarly ground through 25+ low-severity doc-drift findings after code correctness had been silent for many passes. BC-5.39.001's single unified 3-consecutive-clean criterion forces full streak restarts for doc-only findings that never threatened correctness.
**Proposed change:** Evaluate (do not yet implement — needs cross-run data) whether doc/citation-classified findings (already distinguished in the ledger as `[doc-drift]`/`[process-gap]` vs correctness) should retire against a separate streak so pass-count doesn't inflate without actual risk retirement.
**Routes to:** ENGINE — convergence criteria (BC-5.39.001 definition). This is speculative; mark as a pattern-database watch-for, not an immediate change.
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-08** | category: **agent** | MEDIUM priority
**Title:** Endorse #621 — concurrency-ordering remediations must enumerate their own join-obligations in the same pass
**Evidence:** `lessons.md` #20 — DRAIN-WIRE's shutdown-ordering sequence produced a 5-pass chain (SP3-005→SP7-001) where each fix closed its assigned race while opening a new unwaited-goroutine gap in its own new mechanism, caught only by the next pass.
**Proposed change:** No new filing — #621 already exists (MED). Recommend endorsing/prioritizing.
**Routes to:** ENGINE (already filed, #621).
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-09** | category: **cost** | LOW priority (infrastructure)
**Title:** Seed cost tracking — Dimension 1 was entirely unanalyzable this run
**Evidence:** No `cost-summary.md` exists anywhere under `.factory/`; this review could not compute total-vs-budget, per-phase, or per-agent cost at all.
**Proposed change:** Wire cost-summary generation into the factory dispatch protocol so the next session review has Dimension 1 data.
**Routes to:** ENGINE — cost tracking is a core factory capability gap, not switchboard-specific.
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-10** | category: **agent** | LOW priority
**Title:** pr-manager force-push reach-around — confirm the playbook fix landed
**Evidence:** `STATE.md` open-drift row `PROCESS-GAP-FORCE-PUSH` (HIGH) — pr-manager reached for rebase+force-push over `gh pr update-branch`. Filed #408 + switchboard-blue#57.
**Proposed change:** Verify whether the upstream playbook fix for #408 has landed; if not, this remains an open HIGH-severity agent-behavior gap worth escalating.
**Routes to:** ENGINE (already filed, #408) — verification-only ask.
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-11** | category: **template** | LOW priority
**Title:** Correct stale `plugin_version_adopted` in STATE.md frontmatter
**Evidence:** `STATE.md:12` records `plugin_version_adopted: "1.0.0-rc.21"`, one version behind this review's stated current engine (rc.22).
**Proposed change:** Small hygiene fix; also a live instance of Lesson #6's "status fields go stale silently" pattern worth noting for the pattern database rather than fixing in isolation.
**Routes to:** LOCAL — trivial STATE.md correction, next touch.
**Disposition:** APPROVE / DEFER / REJECT: ____

**IP-C1-12** | category: **pattern** | LOW priority (research)
**Title:** Reconcile the 120-file / 39-pass discrepancy in Phase 5 adversarial-reviews accounting
**Evidence:** `.factory/cycles/cycle-1/adversarial-reviews/` contains 120 `P5-pass-N-Adv-A/B.md` files; STATE.md's headline count for the same phase is 39 passes. Not reconciled in this review.
**Proposed change:** A future session (or this review's human follow-up) should grep-derive the actual pass/dispatch/file relationship rather than leave the two counts silently disagreeing — itself an instance of the aggregate-count-must-be-machine-derived pattern this run's own lessons codify.
**Routes to:** LOCAL — quick audit task, low cost.
**Disposition:** APPROVE / DEFER / REJECT: ____

Prioritization note for human disposition: IP-C1-01, IP-C1-02, and IP-C1-03 are the three highest-signal items — one is a live process gap in this project's use of the factory (session-review loop was silent for 23h/165 markers), the other two are HIGH-severity engine gaps this project's rigor happened to surface cleanly and that remain open/unscheduled upstream. Recommend disposing those first; the rest are lower-urgency hygiene and research items.
