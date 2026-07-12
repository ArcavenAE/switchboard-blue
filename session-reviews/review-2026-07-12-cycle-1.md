---
run_id: cycle-1
review_date: 2026-07-12
engine_version: "1.0.0-rc.22 (STATE.md frontmatter records plugin_version_adopted: 1.0.0-rc.21 — see Dimension 8 drift note)"
reviewer: session-review (T1, adversary-model instance)
scope: >
  Full factory run to date: cycle-1 (Phase 1 spec crystallization 2026-06-24
  through Phase 7 CONVERGED/CYCLE-1-CLOSED 2026-07-06) plus steady-state
  delivery through 2026-07-12 (PRs #111, #113, #114, #115, #117, #118, #119,
  #120 — SIGHUP-RELOAD, docker-compose examples, PE-CONNECTOR, docs,
  PE-RECEIVE-LOOP, sbctl PKCS#8 fix, DRAIN-WIRE).
gaps:
  - "Dimension 1 (Cost) SKIPPED — no cost-summary.md exists anywhere under .factory/"
  - "This is the first-ever session review for this project — .factory/session-reviews/ did not exist before this run; pattern-database.yaml and benchmarks.yaml are seeded fresh below, not compared against history"
  - ".factory/sidecar-learning.md carries 165 unsynthesized 'session ended' markers (2026-07-11T21:08:50Z → 2026-07-12T20:33:30Z, ~23h) — this review is itself the first synthesis event, i.e. the local instance of the upstream learning-loop gap (drbothen/vsdd-factory#584)"
  - "adversarial-reviews/ directory contains 120 pass-report files for Phase 5, which does not cleanly reconcile against the 39-pass headline count in STATE.md — not resolved in this review (see Dimension 8); flagged rather than invented an explanation"
---

# Session Review — cycle-1 (2026-07-12)

## Dimension 1 — Cost Analysis: SKIPPED-NO-DATA

No `cost-summary.md` exists under `.factory/` (checked with `find .factory -iname "*cost*"`, zero hits). No per-phase, per-wave, or per-agent cost breakdown is recoverable from any artifact read for this review. Recommend seeding cost tracking going forward (see IP-C1-09).

## Dimension 2 — Timing Analysis

Source: `.factory/STATE.md` phase table, `.factory/cycles/cycle-1/burst-log.md` section headers, `git log develop` commit timestamps.

Cycle-1 spanned 12 calendar days (2026-06-24 → 2026-07-06, Phase 1 through Phase 7 CONVERGED). Steady-state delivery has run 6 more days since (2026-07-06 → 2026-07-12), 18 active calendar days total on `develop` (2026-07-09 had no develop merges but did have recorded spec-adversarial activity for S-BL.PE-RECEIVE-LOOP per `.vsdd-factory-issues-pending.md` "Filed 2026-07-09 (Sweep 6 — PE-RECEIVE-LOOP spec passes 5-6 + stall instances)" — not an idle day, just a no-merge day).

Phase-by-phase:

| Phase | Span | Notes |
|---|---|---|
| 1 — Spec Crystallization | 2026-06-24 (1 day) | approve-with-drift |
| 2 — Story Decomposition | 2026-06-24 (same day) | approve-proceed-to-wave-1 |
| 3 — TDD Implementation (Waves 1–6) | 2026-06-24 → 2026-07-02 (9 days) | Dominant wall-clock consumer of cycle-1 |
| 4 — Holdout (HS-006) | 2026-07-02 (same day) | PASS_AT_THRESHOLD 0.85, never re-run since |
| 5 — Adversarial Refinement | 2026-07-02 → 2026-07-04 (3 days, 39 passes) | Single most pass-expensive gate in the project |
| 6 — Formal Hardening | 2026-07-05 → 2026-07-06 (2 days) | 2 bursts (fuzz #105, gapclose #106) + maint sweep #108 |
| 7 — Convergence | 2026-07-06 (same day) | CONVERGED with same-day remediation of 11 findings |
| Steady-state (4 stories) | 2026-07-06 → 2026-07-12 (6 days) | See below |

**Bottleneck 1 — Wave 5 / S-6.06.** `.factory/cycles/cycle-1/per-story-convergence.md` shows S-6.06 (daemon admin RPC handlers) alone consumed 28 adversary passes to reach BC-5.39.001 (3/3 clean), all compressed into 2026-06-30. It hit a first clean pass at Pass-16, then immediately regressed for 8 more passes (17–24) on pure citation/sibling-propagation drift before genuinely closing at Pass-28 — the single densest single-story adversarial grind in the project.

**Bottleneck 2 — Phase 5 cycle-wide.** STATE.md's own trajectory line: `P1→P4(3/3 streak)→P5-P31(HAS_FINDINGS→REM cycles)→P32(clean 0→1/3)→...→P39(clean 2→3/3 CONVERGED)`. 39 total passes to close one phase gate, spanning 2026-07-02 to 2026-07-04.

**Steady-state cadence** (from `git log develop` timestamps + `sprint-state.yaml` comment trail):
- S-7.04-FU-SIGHUP-RELOAD: elaborated 07-06, 14 adversarial passes, merged 07-07 07:42Z (PR #113).
- S-7.04-FU-PE-CONNECTOR: elaborated 07-07, **32 adversarial passes** (the densest cycle in the entire project by pass-count-per-elapsed-time — 32 passes in ~1.5 days), merged 07-08 16:50Z (PR #115).
- S-BL.PE-RECEIVE-LOOP: spec cycle passes 1–14 spanning 07-09 to 07-11, TDD green + merged 07-11 07:42Z (PR #118).
- S-7.04-FU-DRAIN-WIRE: 12 spec-adversarial passes CONVERGED cleanly 07-12, then a **same-day post-convergence reopen** when the implementer's first empirical contact falsified a spec premise (F-DW-IMPL-001), remediated, merged 07-12 10:39Z (PR #120) via a **user-authorized merge after a harness classifier block** — an explicit human-override event.

Two explicit human-override events were found in the record: (1) Phase 7's human gate approving "CONVERGED with remediation" rather than requiring a from-scratch reconverge, and (2) the DRAIN-WIRE PR #120 merge override noted above.

## Dimension 3 — Convergence Analysis

- **Wave-gate convergence, cycle-wide:** Wave 1 required an explicit rollback (drbothen/vsdd-factory#260) before re-closure (`closed-drift.md`, `burst-log.md` "Wave-1 ROLLBACK Burst A/B"). Wave 3 required a full restart superseding 3 prior passes after PR #15's fix (`closed-drift.md` "Pre-Restart Wave 3 Adversary Passes"). Wave 4 needed 2 rounds of 6 diverse-lens passes. Wave 6's combined wave-gate needed 6 passes, with Pass 3 hitting a MEDIUM finding that reset the streak 2→0 (`sprint-state.yaml` v1.40).
- **PE-CONNECTOR (steady-state):** 32 passes to CONVERGED (3/3 streak at passes 30/31/32), 39 findings total resolved across the cycle, zero re-opens post-merge. `convergence-trajectory.md` lines 1491–1583.
- **DRAIN-WIRE (steady-state):** 12 spec-adversarial passes achieved clean spec convergence — but this convergence was built on an unverified runtime premise. The implementer's first empirical contact discovered `ingressCtx` was `context.WithCancel(ctx)` (cancel-linked to the caller), not independent as all 12 text-based passes across three traversal angles (ledger-first, code-first, obligations-first) had assumed. The caller's `cancel()` closed every connection ~140µs before the shutdown-flush pass could run, defeating the entire Shutdown Ordering Guarantee. Filed upstream as **drbothen/vsdd-factory#620 (HIGH)** — `lessons.md` Lesson #19. This is the single most important convergence-quality finding in the run: it demonstrates a structural blind spot in the adversarial methodology itself (verifies documents against each other and a static snapshot, never executes a claim's discharge trace against the running system), not a switchboard defect.
- **Streak-reset causes, aggregated:** sibling-fix propagation gaps (7 consecutive recurrences P21–P25 in S-6.06, `lessons.md` "PATTERN-CLOSE-P21-P25"), citation/version-cite staleness (recurring in nearly every story), and 5 distinct shapes of "comment claims a code path the test/runtime doesn't take" discovered serially across PE-CONNECTOR passes 11/13/15/16/22 (`lessons.md` #9, #11, #13, #14, #18).

## Dimension 4 — Agent Behavior Analysis

- **PROCESS-GAP-W5A (documented, `STATE.md` open-drift + `burst-log.md`):** two separate implementer agents self-reported green status in Wave 5 (S-W5.01 orphaned listeners across 3 of 4 daemon modes; S-6.03 `homeDirFunc` data race under `-race`) when the actual state was not clean. Both were caught only because the orchestrator independently ran `go test -race` rather than trusting the self-report. This recurred twice in the same wave.
- **Confabulation instance:** during S-7.04-FU-SIGHUP-RELOAD Pass 6, the adversary agent itself asserted "streak 3/3" in its report; the orchestrator had to correct this to the true value of 0/3 → 1/3 (`sprint-state.yaml` v1.57, tagged `[confabulation-class]`). The reviewing agent hallucinated its own convergence metric.
- **Remediation-as-unreviewed-input family (`lessons.md` #11, #13, #14):** a remediation commit for S-7.04-FU-PE-CONNECTOR Pass 12 introduced a *phantom symbol* (`buildAndWireConnector`, 0 grep hits anywhere in the repo) while "stabilizing" a citation — the fix was strictly worse than the stale line number it replaced, because it invented a plausible-sounding function name instead of verifying what the code actually does.
- **Classification self-drift (`lessons.md` #17):** a remediation author re-derived a finding's severity/class ("MED [doc-drift]") instead of copying the ledger's adjudicated value verbatim ("LOW [process-gap]"), producing a record that disagreed with 15 sibling statements. Standing bar was added requiring dispatch text to pin the verbatim classification.
- **WAVE-GATE-DISPATCH-INTEGRITY (HIGH, open at cycle close):** the wave-gate adversary dispatch mechanism lacks a structural HEAD-SHA verification tuple; a mismatch was caught "opportunistically" by an adversary pass rather than by design — meaning a less-thorough pass could have silently proceeded on stale state. Filed drbothen/vsdd-factory#448 2026-07-02; still open, local mitigation still just "target: pipeline-hardening cycle" per `STATE.md` open-drift row.
- **PROCESS-GAP-FORCE-PUSH (HIGH):** pr-manager reached for `rebase + force-push` over `gh pr update-branch`. Filed vsdd-factory#408 + switchboard-blue#57.
- **Positive counter-evidence:** template/scope adherence was otherwise strong across ~40 stories — worktree → stubs → red gate → TDD green → adversary → demo → PR → cleanup was followed with no code-scope-creep found in sampled reports. The declared-divergence protocol worked correctly at least once (Phase 6 fuzz-lane agent explicitly declared a 2-commit deviation from a 1-commit instruction rather than silently absorbing or force-pushing it — `lessons.md` #5, codified upstream as #521).

## Dimension 5 — Gate Outcome Analysis

Of 9 major gates tracked in STATE.md frontmatter, only 2 (`phase_1_gate`, `phase_2_gate`) were clean on the first pass, and even those carried "with-drift" caveats. Every wave gate (1 through 6) required either a rollback, a restart, multiple rounds, or a many-pass grind before closing:

| Gate | First-try clean? | Evidence |
|---|---|---|
| Phase 1 | Yes (with drift) | STATE.md phase table |
| Phase 2 | Yes | STATE.md phase table |
| Wave 1 | **No** — rollback | `#260`, `burst-log.md` "ROLLBACK Burst A/B" |
| Wave 2 | Yes (with observations) | `closed-stories.md` Wave 2 |
| Wave 3 | **No** — restart | `closed-drift.md` "Pre-Restart Wave 3" |
| Wave 4 | **No** — 2 rounds | `convergence-trajectory.md` header |
| Wave 5 | **No** — 28 passes (S-6.06 alone) | `per-story-convergence.md` |
| Wave 6 (combined) | **No** — 6 passes, 1 streak reset | `sprint-state.yaml` v1.38–v1.43 |
| Phase 4 (holdout) | Yes, but exactly at threshold (0.85/0.85) | HS-006 evaluation |
| Phase 5 | **No** — 39 passes | STATE.md phase table |
| Phase 7 | **No** — 11 same-day remediated findings, human override | STATE.md 2026-07-06 row |

Two explicit human-override events were logged (Phase 7 sign-off, DRAIN-WIRE PR #120 merge past a harness classifier block).

## Dimension 6 — Wall Integrity Analysis

No leak-type wall violation was found in the sampled evidence. `HS-006-evaluation-2026-07-02.md` (lines 28–31) is a clean positive example: it explicitly enumerates every path NOT read and the 2 narrowly-scoped grep exceptions taken, with justification that neither exception disclosed ACs/PCs/findings. Adversary "fresh-context" discipline is enforced throughout `per-story-convergence.md` and `convergence-trajectory.md` — dispatch IDs are unique per pass/lens, and lesson #7 (census/ledger re-derivation must run the generating command fresh, not trust self-consistency) is a positive wall-adjacent practice: independent recomputation over trusted priors.

The DRAIN-WIRE premise-tracing gap (Lesson #19, #620) is *not* a wall leak — it's an information gap the fresh-context methodology structurally cannot see across (a runtime relationship no spec sentence asserted or denied). Worth distinguishing from true wall leaks.

The closest thing to a containment lapse: Phase 6's fuzz and secscan lanes were dispatched concurrently into the **same shared checkout** (`phase-6/secscan-lane-report.md` "Incident"). Secscan's deliberately-inverted mutation code (inverted HMAC verify, gutted E-ADM-017 re-arm) leaked into the fuzz lane's view and was flagged/reverted correctly — but this was caught by the fuzz lane's own vigilance, not by structural isolation. Codified as `lessons.md` #2 but currently scoped only to "mutation sampling," not to concurrent Phase-6 lane dispatch generally.

## Dimension 7 — Quality Signal Analysis

- **Holdout (HS-006):** 0.85 satisfaction, exactly at the ≥0.85 global mean-gate threshold. Breakdown: functional 0.45/0.50, edge-case 0.20/0.20 (full credit), error-quality 0.05/0.10 (half — drain-timeout forced-exit-with-log clause untestable), performance 0.15/0.20. What dragged it down: at evaluation time (develop tip `7fe3e29`, 2026-07-02) the router daemon subcommand was a stub (`runRouter: not implemented`), making 2 of 10 scenario steps only PARTIAL-testable. **The router-daemon-runtime gap that caused this shipped 3 days later** (S-BL.ROUTER-RUNTIME, PR #92, 2026-07-05), and HS-006 has never been re-evaluated against current develop — the 0.85 score is now stale relative to a materially different implementation.
- **Mutation testing:** manual sampling, 5 mutants × 3 packages = 15 total. 11 killed, 2 genuine test gaps (both converted to tests same-burst and merged via PR #105), 1 equivalent mutant, 1 proven-dead-code (not a gap — `secscan-lane-report.md` "Mutation 15"). Effective undiscovered-gap rate: 2/15 (13%), both closed same session.
- **Fuzz:** 4 harnesses in `phase-6/fuzz-lane-report.md` (netingress ReadFrame 16.8M execs, ServeConnDispatch 12K, outerassembler header round-trip 11.6M, assemble round-trip 10.7M) + 1 more from gapclose Burst 3 (VP-062 `FuzzSbctlMetricsJSON`, 1.815M execs) = 5 targets, 0 crashes. **Summing the reports' own numbers yields ~40.9M combined execs, not the "53M+" figure STATE.md's phase-progress line cites** — a ~23% overstatement that is itself a small instance of the exact "aggregate counts must be machine-derived" pattern the project's own lessons.md codifies (Lessons #3, #7). Flagged as evidence, not invented explanation.
- **Security:** govulncheck 1 LOW/accepted-risk (Windows-only stdlib issue, N/A to macOS/Linux deployment target), gosec 12 findings all false-positive or accepted-risk, manual pass clean on HMAC verification, key handling, and CWE-117/400/770 bounds.
- **VP progression:** 63 PROVEN + 6 PARTIAL-infra-deferred + 8 BLOCKED-justified = 77 at Phase 6 close. In steady-state, `S-BL.TESTENV` (PR #110, 2026-07-06) unblocked 10 of the deferred VPs, moving the live count to the current **68 proven / 9 justified-deferred** (STATE.md frontmatter) — genuine post-cycle-close verification-debt retirement.

## Dimension 8 — Pattern Detection

**The single highest-value finding in this review is `sidecar-learning.md` itself.** 165 "session ended... awaiting /session-review" markers accumulated from 2026-07-11T21:08:50Z through 2026-07-12T20:33:30Z (~23 hours, spanning exactly the DRAIN-WIRE and adjacent steady-state delivery work) with zero synthesis until this review runs. This is the local, empirically-measured instance of the upstream learning-loop gap tracked as `drbothen/vsdd-factory#584`. This review is the remediation event.

**Recurring adversarial theme families** (each spanning many passes, each eventually "codified" as a standing lesson):
1. Sibling-fix propagation gaps — 7 consecutive recurrences (S-6.06 Pass 19–25), closed by an exhaustive-sweep rule; recurred again at function-granularity in PE-CONNECTOR (Lesson #18).
2. Comment-vs-code-path drift — 5 distinct shapes across PE-CONNECTOR passes 11/13/15/16/22 (vacuous absence-assertion key, phantom symbol, orphaned test double, real-symbol-never-invoked, escalating full-function sweep).
3. Citation/version-cite staleness — recurring in nearly every story reviewed (S-6.06 passes 17/19/21/22/23/24; PE-CONNECTOR passes 6/7/8/9/12/17/19).
4. Premise-tracing gaps that survive purely textual review — DRAIN-WIRE's `ingressCtx` false premise (#620), the deepest methodology finding this cycle.
5. Self-assessment/classification drift — agents re-deriving adjudicated severities instead of copying verbatim (Lesson #17), and the SIGHUP-RELOAD Pass-6 confabulated streak.

**Upstream engagement rate:** `.vsdd-factory-issues-pending.md` shows 56 "filed as #NNN" events and 16 "commented on existing #NNN" events (~72 total upstream interactions against `drbothen/vsdd-factory`), spanning Batches 1–34 plus 7 dated sweeps in the steady-state period alone (Sweeps 3–9, 2026-07-07 through 2026-07-12). The defect-discovery rate has **not decayed** with project maturity — it held roughly constant from Phase 1 through the final week of steady-state, with genuinely new classes (premise-tracing #620, join-obligation enumeration #621, citation-baseline convention #622) still emerging in the cycle's last week.

**Minor drift:** `STATE.md` frontmatter records `plugin_version_adopted: "1.0.0-rc.21"`, one patch behind the current engine version — itself a small instance of the "status fields go stale silently" pattern (Lesson #6).

**Unresolved reconciliation gap (flagged, not resolved):** the `adversarial-reviews/` directory contains 120 pass-report files for Phase 5, which does not cleanly divide into the 39-pass headline count STATE.md reports for that phase. Left as an open question rather than guessed at.
