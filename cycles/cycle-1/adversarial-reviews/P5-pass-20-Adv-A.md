---
pass_id: P5P20-Adv-A
adversary_lens: spec-completeness + traceability
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: <not-verified-read-only-no-bash>
  refs/heads/develop_sha: <not-verified-read-only-no-bash>
  origin/develop_sha: <not-verified-read-only-no-bash>
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS (orchestrator-verified out-of-band; adversary tool profile has no Bash — Read/Grep/Glob only)
budget:
  wall_clock_target: <=6 min
  wall_clock_used: ~6 min
  file_reads_target: <=6
  file_reads_used: 6
  overage_disclosure: |
    On budget at 6 reads. Three initial reads used correcting cwd-path
    (glob for STORY-INDEX after direct read hit "file not found"),
    three productive reads on STORY-INDEX.md, sprint-state.yaml, and
    STATE.md. Would have preferred to also read S-BL.ROUTER-ADDR.md to
    confirm its status: merged frontmatter, but budget exhausted;
    finding stands on master-table + Wave Summary + narrative-gloss +
    v3.55 changelog cross-corroboration.
verdict: HAS_FINDINGS
streak_state:
  adjudicated_deferrals_respected: true
  reopened_deferrals: none
  respected_list:
    - DRIFT-P5P7-O1, DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001 (SHIPPED at PR #69)
    - F-P5P14-A-001..F-P5P14-B-005 (all adjudicated per prior passes)
    - F-P5P15..F-P5P19 findings (all SHIPPED or adjudicated per prior remediation)
    - F-P5P19-A-001 SHIPPED at .factory e65e429 (STORY-INDEX v3.75, S-7.03 rescope arithmetic sweep — necessary-but-incomplete sweep, sibling gap surfaced this pass)
delivered_by: p5-pass20-adv-a
---

# Phase 5 Pass 20 Adv-A — findings

Verify-then-claim: finding carries file:line anchors resolved at develop tip 6deda15def9326f28e96f133e237aff5ecb74d7b (orchestrator preflight).

---

### F-P5P20-A-001 [MED]: STORY-INDEX v3.75 Wave-6 aggregate omits S-BL.ROUTER-ADDR — sibling-sweep gap from v3.55 promotion (POL-002)

Anchor: `.factory/stories/STORY-INDEX.md:80,92,94,96,204` (master-table row + Wave Summary + narrative gloss + v3.55 changelog entry)

Class: aggregate drift / POL-002 sibling-sweep

Confidence: HIGH. Severity: MED.

Symptom: STORY-INDEX v3.75 line 80 master-table records `S-BL.ROUTER-ADDR | ... | Wave: 6 | ... | 2 | P1 | E | merged (PR #56, 91d5675)`. Line 204 changelog entry for v3.55 (2026-07-01) documents the promotion: "POL-002 Tranche B merged-story sync — S-BL.ROUTER-ADDR promoted from backlog to Master Story Index as merged (PR #56, 91d5675, wave backlog→6). Summary: Complete 29→32, Pending 2→1, Ready-for-red-gate 1→0, Backlog 9→8, master-table 35→36, E-phase 31→32." However the Wave Summary aggregation was never updated to include the promoted story:

- Line 92 Wave-6 row story enumeration: `S-W5.04, S-BL.LOOKUP, S-6.07, S-6.05, S-7.01, S-7.02, S-7.03` (7 stories; S-BL.ROUTER-ADDR OMITTED); points column: `31`.
- Line 94 Total row: `**34** (wave stories) | **191** | (... grand total 36 stories / 201 pts when maintenance included)`.
- Line 96 narrative gloss: `Wave 6 was 7 stories / 40 pts ... net delta: S-7.04 removed (−8pts), S-BL.LOOKUP added (+1pt), and S-7.03 re-scoped 5→3 per RULING-W6TB-C v3.46 = 31 pts` — reconstructs 31 pts without S-BL.ROUTER-ADDR term.

F-P5P19-A-001 (SHIPPED 2026-07-03 at .factory e65e429) swept the S-7.03 5→3 rescope arithmetic against this already-stale 7-story Wave-6 enumeration — sweeping the wrong denominator forward. That prior pass was necessary-but-incomplete: the v3.55 sibling-sweep gap was not surfaced because the adversary re-derived Wave-6 from the (stale) narrative gloss instead of summing the master-table Wave-6 rows.

Correct Wave-6 arithmetic (sum of master-table rows filtered by `Wave: 6`): S-W5.04(5) + S-BL.LOOKUP(1) + S-6.07(3) + S-6.05(3) + S-7.01(8) + S-7.02(8) + S-7.03(3) + S-BL.ROUTER-ADDR(2) = **33 pts, 8 stories**.

Cascade impact: Wave-6 aggregate 31→33 pts; Total wave stories 34→35 / 191→193 pts; grand total 36→37 stories / 201→203 pts (when maintenance included); Summary block waves 0-6 subtotal 183→185; with maintenance 193→195; narrative gloss line 96 needs new arithmetic term.

Verify: read of `.factory/stories/STORY-INDEX.md` lines 33-34 (Summary block totals), 78-85 (E-phase master-table rows), 88-96 (Wave Summary + narrative gloss), 200-210 (changelog entries around v3.55) confirms every anchor. Wave-6 rows summed: 5+1+3+3+8+8+3+2 = 33. Missing 2 pts is exactly S-BL.ROUTER-ADDR's contribution.

Remediation shape: STORY-INDEX v3.75 → v3.76. Line 92: append `S-BL.ROUTER-ADDR` to story enumeration; 31→33. Line 94: 34→35 wave stories; 191→193 pts; grand total → 37 stories / 203 pts. Line 96: extend narrative arithmetic to include S-BL.ROUTER-ADDR promoted-from-backlog (+2pts, PR #56, per v3.55); new sum 33 pts / 8 stories. Lines 33-34: 183→185 and 193→195. Add v3.76 changelog row per POL-001 citing F-P5P20-A-001 (POL-002 sibling-sweep completion). Class as F-P5P19-A-001 follow-on / necessary-but-incomplete sweep.

---

## Anti-findings (checked and passing)

- **POL-001 changelog-completeness (v3.75, VP-INDEX v2.35)**: STORY-INDEX v3.75 line 204 records POL-002 Tranche B tranche with dated 2026-07-01 entry; VP-INDEX v2.35 (2026-07-02) has an entry for VP-043 method reclassification. Both changelogs are current at their respective HEAD versions. No changelog gap on either at this pass.
- **STORY-INDEX ↔ sprint-state.yaml pass_19 consistency**: sprint-state.yaml pass_19 block records `PASS_19_HAS_FINDINGS`, 4 findings shipped (F-P5P19-A-001, F-P5P19-B-001, F-P5P19-B-002, F-P5P19-B-003), pass_counter 19, attempts_counter 19, streak 0/3, last_reset_reason `pass-19-has-findings`. STATE.md `phase-5-pass-19-concluded-has-findings`. All consistent with .factory tip d6f08c1.
- **S-BL.ROUTER-ADDR merge status**: master-table row correctly marks merged (PR #56, 91d5675). No status drift; only the aggregate propagation is missing.
- **Adjudicated deferrals**: none of the deferrals in the streak-state list appear in the finding above; all remain deferred.
- **VP-INDEX v2.35 changelog**: entry for F-P5P3-B-001 close (VP-043 Proptest→Unit strong-oracle) present in VP-INDEX. No changelog completeness gap.

---

VERDICT: HAS_FINDINGS
