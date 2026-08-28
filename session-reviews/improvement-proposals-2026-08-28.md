# Improvement Proposals — cycle-1-post (2026-08-28)

Status: PENDING HUMAN DISPOSITION. 9 proposals, all evidence-backed against artifacts read this run (see review-2026-08-28-cycle-1-post.md for full citations).

**IP-C2-01** | category: **workflow** / **pattern** | **HIGH priority — repeat of IP-C1-01, escalated**
**Title:** The session-review habit did not survive one cycle — replace the prose convention with a mechanical trigger
**Evidence:** IP-C1-01 (2026-07-12) was APPROVED and dispositioned "local half ratified: /session-review added to the steady-state session-close habit." Since then: 0 invocations, 599 unsynthesized markers, 589 of which accumulated in the 11 days immediately following ratification (before any dormancy) and 10 more from today's dispatch. PAT-06's watch_for question is now answered empirically: it kept accumulating, immediately, at a HIGHER absolute rate than the first (unratified) instance.
**Proposed change:** Stop relying on session-close discipline as the trigger. Build the structural half of IP-C1-01's ORIGINAL proposal, approved in principle but never implemented: a bounded marker-count or elapsed-time threshold check that runs automatically (e.g. within `factory-worktree-health` or any STATE.md-reading entry point) and either (a) refuses/warns before further phase dispatch when `sidecar-learning.md` exceeds N unsynthesized markers or M days since the last SYNTHESIZED line, or (b) auto-invokes `/session-review` at that threshold. Same shape as this project's own 48h bd-staleness reminder — the analogy was named last time but not built.
**Routes to:** LOCAL (trigger mechanism) + ENGINE (drbothen/vsdd-factory#584; recommend a follow-up comment citing this run's numbers as a second data point: 165/23h → 599/47 days, i.e. the gap grew ~3.6x rather than shrinking).

**IP-C2-02** | category: **gate** | **HIGH priority**
**Title:** PR #129's false-convergence (weak AC assertion + premature merge) is a compound failure, not two independent ones
**Evidence:** PR #129 merged past a creation-only dispatch scope (root cause: AUTHORIZE_MERGE fires independent of declared dispatch scope, `#756` HIGH) carrying an AC-003 test whose assertion was strictly weaker than its postcondition (extends open `#337`). A real defect (SVTNID guard incomplete) reached `develop`, caught only post-merge.
**Proposed change:** Two local, structural mitigations: (1) require every pr-manager dispatch to carry an explicit machine-checkable `dispatch_scope: creation-only | full-lifecycle` field that AUTHORIZE_MERGE itself gates on, not prose the hook cannot see; (2) require every AC test whose postcondition is more specific than "the log line contains substring X" to include a companion mutation-style negative case asserting failure against the specific wrong-path implementation the AC exists to exclude.
**Routes to:** ENGINE (`#756`, `#337` — filed) + LOCAL (adopt `dispatch_scope` pre-emptively; the failure is intermittent, and a local convention costs nothing now).

**IP-C2-03** | category: **convergence** | **MEDIUM priority**
**Title:** S-BL.LOOPBACK-FULLSTACK is stranded mid-convergence at R5 with a stale STATE.md checkpoint — resume or formally re-park
**Evidence:** 5 rounds run 2026-07-22→23, all NOT CLEAN, net-decaying trend. STATE.md's Session Resume Checkpoint is dated 2026-07-22 and predates R3-R5 entirely — a resuming session reading STATE.md alone would not know R3-R5 happened, that R3's HIGH was fixed, or that the story is 1 MED from a clean streak.
**Proposed change:** Either dispatch R6 (findings are narrow — all in the same H3/AC-017 provisioning area, likely a single architect+story-writer touch) or explicitly update STATE.md's checkpoint to reflect the true parked state, so the next session need not reconstruct 36 days of drift from git-log archaeology, which is what this review had to do.
**Routes to:** LOCAL — state-manager/orchestrator housekeeping, no engine gap.

**IP-C2-04** | category: **pattern** | **MEDIUM priority**
**Title:** PAT-01 crossed the 3-cross-story-instance threshold and gained an intra-story sub-instance — promote to structural mitigation
**Evidence:** 3rd cross-story instance (DISCOVERY-WIRE v2.26→v2.27, commented `#470`); separate intra-story instance (`decodeRTID` arity straggler, 3 rounds to close). task-workflow.md's three-instance rule treats 3+ recurrences as the line between content defect and process gap.
**Proposed change:** The mitigation already named in v2.27's changelog ("variant patterns added to the standing sweep-set") should become MECHANICAL — an automatic exhaustive-structural-form grep (suffix/paren/possessive/line-wrap/arity) on every fix-burst commit touching a renamed or re-arity'd symbol, not a manual sweep reinvented at each new granularity.
**Routes to:** LOCAL — no new upstream filing; `#470`'s comment carries the evidence.

**IP-C2-05** | category: **agent** | **MEDIUM priority**
**Title:** PAT-04 — 4th occurrence, still only caught by opportunistic independent verification
**Evidence:** pr-manager Step-9 branch-deletion misverify (`ls-remote` exit-code misread), `#746`. Same shape as 3 prior instances.
**Proposed change:** Make the orchestrator's post-cleanup verification MANDATORY (assert `ls-remote` returns EMPTY OUTPUT, not "the exit code looked right") for every pr-manager Step-9 dispatch, rather than relying on opportunistic re-checking. Small, local, immediately adoptable independent of upstream.
**Routes to:** ENGINE (`#746`, filed) + LOCAL (adopt assertion-not-interpretation now).

**IP-C2-06** | category: **wall** | **MEDIUM priority**
**Title:** #448 hit its 4th+ instance — IP-C1-03's "schedule the hardening cycle" was approved and never executed
**Evidence:** R9 wrong-tree dispatch (nested-worktree-glob collision + substring tripwire), same-session mitigated but the 4th+ occurrence of a class the prior review already called HIGH-severity-open. STATE.md's `WAVE-GATE-DISPATCH-INTEGRITY` row still reads "mitigated-local," unchanged in six weeks.
**Proposed change:** Actually schedule the hardening cycle (dispatch-integrity tuple as structural policy across every adversary dispatch, not per-incident marker tweaks). If genuinely deprioritized, say so explicitly in STATE.md rather than leaving six-week-old "QUEUED as next major work item" language standing unchallenged.
**Routes to:** ENGINE (`#448`) + LOCAL (the scheduling decision).

**IP-C2-07** | category: **quality** | **LOW priority**
**Title:** HS-006 holdout score is stale again (0.895 @ f73676d, 3 stories old)
**Evidence:** Same staleness shape as IP-C1-04, recurring in miniature — NODE-IDENTIFY-WIRE, DISCOVERY-WIRE, and NODE-IDENTIFY-SVTNID-CONSISTENCY all shipped after the 2026-07-12 evaluation.
**Proposed change:** Re-run HS-006 at current `develop` tip when LOOPBACK-FULLSTACK work resumes, rather than waiting for a 3rd-instance pattern.
**Routes to:** LOCAL — maintenance-sweep task.

**IP-C2-08** | category: **workflow** | **LOW priority**
**Title:** S-7.02 cycle-close disposition was left half-finished
**Evidence:** The 3 process-gap candidates named in STATE.md's "NEXT" line were filed upstream (`#470` comment, `#746`, `#747`) on 2026-07-21, but STATE.md's checkpoint still names "S-7.02 cycle-close disposition" as pending — the local close-out was never written before the pivot to LOOPBACK-FULLSTACK and the dormancy.
**Proposed change:** Close the loop in STATE.md — a one-line "S-7.02 disposed: #470/#746/#747 filed 2026-07-21" — on the next STATE.md touch.
**Routes to:** LOCAL — trivial correction.

**IP-C2-09** | category: **pattern** | **LOW priority (research)**
**Title:** Sidecar-marker density (599 this period) does not track delivery density (9 PRs)
**Evidence:** 2026-07-13 through 2026-07-15 carries 368 markers (174+100+94); adding 07-16 (65) makes 433 across four days — against roughly 4 commits in the same window.
**Proposed change:** A future review with dispatch-log access should determine whether this reflects many short-lived sub-agent dispatches each firing their own session-end hook, versus retried/aborted work. Not diagnosed here — PAT-09, watch-for only.
**Routes to:** LOCAL — research question, no action now.

---

**Cross-references carried forward, still open:** IP-C1-07 (deferred, needs cross-run PAT-01/PAT-02 recurrence data — that data now exists per IP-C2-04, though the literal "≥2 more session reviews" trigger is only partially satisfied by this being review #2); IP-C1-09/cost-tracking (still no local data, tracked upstream #583).

**Prioritization note for human disposition:** IP-C2-01 and IP-C2-02 are the highest-signal items. IP-C2-01 because the exact gap this project already named and "fixed" once got measurably worse, not better — a finding about how this project responds to its own session-review findings, not just about the factory. IP-C2-02 because it is the first confirmed instance of a real defect reaching `develop` behind a false convergence signal, rather than a slow-but-honest grind.
