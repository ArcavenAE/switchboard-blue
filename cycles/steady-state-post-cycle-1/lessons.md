---
document_type: lessons-learned
level: ops
version: "1.0"
status: in-progress
producer: state-manager
timestamp: 2026-08-30T00:00:00Z
cycle: steady-state-post-cycle-1
inputs: [session-reviews/review-2026-08-30-loopback-v1.21.md, session-reviews/improvement-proposals-2026-08-30-loopback-v1.21.md, STATE.md]
input-hash: "09f2094"
traces_to: STATE.md
---

# Lessons Learned — steady-state-post-cycle-1

<!-- S-7.02 Cycle-Closing harvest for the S-BL.LOOPBACK-FULLSTACK v1.20→v1.21
     spec-convergence sub-cycle. Codifies the process-gap findings surfaced
     by the session-review into durable dispositions so the sub-cycle can be
     formally CLOSED. Source review: session-reviews/review-2026-08-30-loopback-v1.21.md.
     Source proposals: session-reviews/improvement-proposals-2026-08-30-loopback-v1.21.md
     (IP-C3-01 through IP-C3-04, PROPOSED, non-blocking 72h human-disposition
     auto-defer — same convention as the IP-C2 set in improvement-proposals-2026-08-28.md).

     S-7.02 Codification — S-BL.LOOPBACK-FULLSTACK v1.20→v1.21 Sub-Cycle
     (2026-08-30): six process-gap findings surfaced across the R35–R40
     rounds, the two reconvergence arcs (v1.19→v1.20, v1.21), and the
     session-review of this sub-cycle. Each is dispositioned below — every
     one resolves to either a follow-up proposal (`[codified]`) or a
     justified deferral (`[deferred]`), per S-7.02. -->

## Agent-Level

_No agent-level lessons this sub-cycle._ All six findings below are process
findings (dispatch construction, artifact maintenance, convergence
methodology, review coverage) rather than agent-behavior findings — the
session-review's Dimension 4 (Agent-Behavior Analysis) recorded strong
template adherence, zero scope creep, and zero dropped/timed-out dispatches
across R35–R40 (`session-reviews/review-2026-08-30-loopback-v1.21.md`).

## Process-Level

1. **[codified] Dispatch-hygiene: POL-005 tuple paraphrased source paths.** R37
   Lens B named source paths as `internal/transport/{halfchannel,multipath,arq}.go`;
   the real paths are `internal/{halfchannel,multipath,arq}/`. Lens B self-corrected
   against the real tree and the verdict was unaffected (near-miss, not a
   defect that changed an outcome) — but the story's `architecture_modules`
   frontmatter already carried the correct paths, so the paraphrase was
   avoidable. Disposition: proposal **IP-C3-01** (category: agent/workflow,
   severity: LOW) — POL-005 dispatch construction must copy source paths
   verbatim from the target story's `architecture_modules:` frontmatter,
   never reconstruct them from memory. Tracked as `PENDING-IP-C3-01` in
   STATE.md Open Drift; routes LOCAL (orchestrator dispatch-checklist), no
   engine filing warranted — authoring habit, not a factory defect.
   _Discovered: rereview-R37-2026-08-29.md._

2. **[codified] STORY-INDEX frontmatter `version:` lagged its own changelog
   row.** Commit 06d5fe9 added the v4.175 changelog row + ref-column content
   but left the frontmatter `version:` field at "4.174"; fixed by 8018ab5,
   caught before R38 dispatch. This is a NEW intra-document granularity of
   the recurring propagation-gap family (PAT-01) — prior instances were
   cross-document (the STORY-INDEX ref-column mega-cell vs. externally cited
   sources, per R22-LENSA-PROCESS-GAP); this instance is intra-document
   (frontmatter field vs. the document's own changelog row). Disposition:
   proposal **IP-C3-02** (category: agent/template, severity: MEDIUM) —
   whichever step bumps STORY-INDEX must run one mechanical assertion after
   any changelog-row-add: frontmatter `version:` == newest changelog row
   version, before the burst completes. Also recorded in
   `session-reviews/pattern-database.yaml` as a PAT-01 update (3rd flavor).
   Tracked as `PENDING-IP-C3-02` in STATE.md Open Drift; routes LOCAL
   (STORY-INDEX maintenance convention).
   _Discovered: commit 06d5fe9 (lag) → fixed 8018ab5, before R38._

3. **[codified] Prefer story-only anchoring over a note edit when
   architecturally sufficient.** The note is a declared input; a note edit
   forces an input-hash recompute. During the v1.21 anchor burst an
   architect note-edit to v1.17 was drafted to anchor BC-2.02.003, then
   reverted in favor of a story-only anchor — keeping the v1.19 (BC-2.01.003)
   and v1.21 (BC-2.02.003) anchors procedurally symmetric and avoiding an
   unnecessary input-hash cascade. (Contrast: R33 legitimately bumped the
   note v1.15→v1.16 for a real TLPKTDROP fix and accepted the recompute
   4902d5d→7967a2f — the discipline is not "never touch the note," it is
   "don't touch it when a story-only anchor is sufficient.") Disposition:
   proposal **IP-C3-03** (category: convergence/pattern, severity: LOW,
   guideline) — codify this discipline as a named non-mandatory guideline in
   the Step-4.5/§1.7 playbook so future sessions default to asking "is a
   note edit strictly necessary, or can this be anchored story-only per the
   GAP-4/v1.19 precedent?" before touching a declared input. Tracked as
   `PENDING-IP-C3-03` in STATE.md Open Drift; routes LOCAL (methodology
   guidance) — behavior was already correct this sub-cycle; this reduces
   future drift risk.
   _Discovered: v1.21 human-gate anchor burst, 2026-08-30 (note-edit drafted
   then reverted)._

4. **[codified] Contract-completeness heuristic for §1.7-class checks.**
   F-R40-LENSB-01 (NONE severity): the story's Q4 safety prose (L413-416)
   addressed only one clause (no-callback-under-lock) of the multi-clause
   `KeystrokeSink` "must not retain" contract cited by
   `loopbackSink.SendInput`/`downstreamHC.Enqueue`, and was silent on the
   retain clause. Adjudicated safe by construction (fresh per-tick copy via
   `toMPFrame`, story L1210) — the human reviewed and **accepted as-is**,
   declining the optional §1.12-style hardening sentence. Disposition:
   proposal **IP-C3-04** (category: pattern/quality, severity: LOW,
   optional/research) — record "safe-by-construction-but-textually-incomplete
   contract clause" as a named survivor class; consider (not mandate) a
   future §1.7-class heuristic that enumerates every clause of a cited
   multi-clause contract and confirms the safety prose addresses each, so
   this class of gap is found systematically rather than by chance. Tracked
   as `PENDING-IP-C3-04` in STATE.md Open Drift; routes
   LOCAL/ENGINE-adjacent (possible future §1.7 sub-check) — low priority per
   its research framing; the human already exercised discretion on the
   motivating instance.
   _Discovered: R40, F-R40-LENSB-01; human approval gate, 2026-08-30._

5. **[deferred] Recurring triple-ledger / STORY-INDEX mega-cell
   `[process-gap]`.** R22-LENSA-PROCESS-GAP (lineage: F-R25, F-R33-LENSA-O1,
   F-R34-LENSC-01) was quiet across R35–R40 (zero re-raises in this window)
   but is NOT retired — finding #2 above (STORY-INDEX frontmatter vs. its
   own changelog row) is a sibling instance of the same family surfacing on
   a different field-pair, which reinforces rather than closes the
   underlying gap class. Disposition: **deferred**. The standing
   recommendation remains a mechanical validator/generator at the human
   gate rather than a per-finding manual fix; this is carried in STATE.md
   Open Drift as the standing item. IP-C3-02 (finding #2) is the cheap
   first slice of this larger fix and should be read as a down payment
   against it, not a substitute for it. Justification for deferral: the
   full validator/generator is a larger engineering item than fits a
   single-sub-cycle remediation burst, and the gap did not recur in its
   original form this window — no urgency signal to escalate beyond the
   existing standing recommendation.
   _Discovered (this window): quiet R35–R40; reinforced by finding #2's
   POL-001 incident (commits 06d5fe9/8018ab5)._

6. **[deferred] PAT-06 sidecar-learning non-synthesis backlog.** The
   sidecar-learning.md marker count is accelerating: 165 (2026-07-12) → 599
   (2026-08-28) → 809 (2026-08-30), +210 in roughly two days. Disposition:
   **deferred**. IP-C2-01 (proposed 2026-08-28, `session-reviews/
   improvement-proposals-2026-08-28.md`) is the standing structural fix for
   this backlog and remains PENDING human disposition (STATE.md Open Drift,
   `PENDING-IP-C2-01`..`09`); this sub-cycle re-flags the worsening trend in
   `session-reviews/benchmarks.yaml` rather than opening a duplicate
   proposal. Justification for deferral: a second proposal targeting the
   same root cause would fragment the fix; the correct action is to keep
   IP-C2-01 as the single tracked remediation and let this entry serve as
   trend evidence supporting its priority at human disposition time.
   _Discovered: session-reviews/review-2026-08-30-loopback-v1.21.md
   Dimension 8 (PAT-06); benchmarks.yaml trend update, 2026-08-30._

### Informational (non-process-gap; no codification action required)

- **OBS-LENSB-R39-WINDOW-STEADYSTATE** (R39, NONE severity) and
  **F-R40-LENSB-01** (R40, NONE severity, also codified above as the
  motivating instance for IP-C3-04) are recorded as documented survivor
  classes for this story. Both were reviewed and accepted as-is at the
  human approval gate; neither requires further codification beyond the
  IP-C3-04 disposition already recorded for F-R40-LENSB-01.
- **`cycles/cycle-1/convergence-trajectory.md` template-compliance drift**
  is pre-existing, non-blocking, and unrelated to this sub-cycle (it fails
  `validate-template-compliance` for sections the file never had). No
  action proposed; carried as informational context only.

## Infrastructure-Level

_No infrastructure-level lessons this sub-cycle._ This sub-cycle is a
spec-only Step-4.5 adversarial convergence loop for a draft/unscheduled
story (no implementation, CI, or deployment activity) — see
`session-reviews/review-2026-08-30-loopback-v1.21.md` Dimensions 1/2/6/7,
all marked N/A for this window (no cost-summary, no budget, no code
delivered, spec-only).

## Policy Candidates

| Lesson | Proposed Policy | Scope | Status |
|--------|----------------|-------|--------|
| 1 | POL-005 dispatch construction must copy source paths verbatim from story `architecture_modules:` frontmatter, never reconstruct from memory | Orchestrator dispatch-checklist (IP-C3-01) | proposed |
| 2 | STORY-INDEX version-bump step must assert frontmatter `version:` == newest changelog row version before burst completes | STORY-INDEX maintenance convention (IP-C3-02) | proposed |
| 3 | Prefer story-only anchoring over a note edit when architecturally sufficient; named non-mandatory guideline | Step-4.5/§1.7 playbook (IP-C3-03) | proposed |
| 4 | Enumerate every clause of a cited multi-clause contract when a spec's safety prose invokes it, for future §1.7-class checks | Possible future §1.7 sub-check (IP-C3-04), optional/research | proposed |
| 5 | Mechanical validator/generator for the STORY-INDEX triple-ledger / ref-column mega-cell class | Standing recommendation, carried in STATE.md Open Drift | deferred |
| 6 | Sidecar-learning.md synthesis mechanism (see IP-C2-01) | Session-review learning-loop closure | deferred (tracked via IP-C2-01) |

**S-7.02 satisfied — every process-gap finding has a follow-up proposal or a
justified deferral; sub-cycle CLOSED.**
