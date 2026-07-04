---
pass_id: P5P19-Adv-A
adversary_lens: public-surface + operator-UX drift lens
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: <not-verified-read-only-no-bash>
  refs/heads/develop_sha: <not-verified-read-only-no-bash>
  origin/develop_sha: <not-verified-read-only-no-bash>
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS (orchestrator-verified out-of-band before dispatch)
budget:
  wall_clock_target_min: 6
  reads_used: 6
  reads_budget: 6
verdict: HAS_FINDINGS
findings_count: 1
anti_findings_count: 6
policies_applied: [POL-001, POL-002]
streak_state:
  adjudicated_deferrals_respected: true
  respected_list:
    - DRIFT-P5P7-O1
    - DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - DRIFT-P5P9-STALE-RECONCILIATION-COMMENT
    - DRIFT-P5P14-B-001-VP-SOURCE-BC-VERSION-PIN
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001
    - F-P5P14-B-002, F-P5P14-B-003, F-P5P14-B-004, F-P5P14-B-005
    - F-P5P15-A-001, F-P5P15-B-001
    - F-P5P16-A-001
    - F-P5P17-A-001, F-P5P17-A-002
    - F-P5P18-A-001, F-P5P18-B-001
    - VP-077 wording nit
    - STORY-INDEX completed/merged mixed vocabulary
delivered_by: Adv-A
adjudication:
  F-P5P19-A-001: SHIPPED at .factory e65e429 — STORY-INDEX v3.74 → v3.75; Wave-6 aggregate arithmetic sweep after S-7.03 5→3 rescope (v3.46 cascade); Wave 6 33→31, waves 0-6 total 192→183, incl. maintenance 202→193, grand total 200→191; grand-total-with-maintenance narrative 210→201 pts (bonus cascade). POL-002 aggregate-freshness closure.
---

## Critical Findings

(none)

## Important Findings

### F-P5P19-A-001 — STORY-INDEX Wave-6 point arithmetic stale after S-7.03 re-scope 5→3 (POL-002 adjacent, aggregate-row drift)

- **Class**: STORY-INDEX aggregate-row drift / POL-002-adjacent (aggregate rollups not swept after story-cell re-scope)
- **Confidence**: HIGH (arithmetic verifiable against master-table cells + changelog v3.46 provenance)
- **Severity**: MED
- **Anchors**:
  - `.factory/stories/STORY-INDEX.md:33` — `Total points (waves 0–6) | 192`
  - `.factory/stories/STORY-INDEX.md:34` — `Total points (incl. S-M.01 + S-M.02) | 202`
  - `.factory/stories/STORY-INDEX.md:77` — S-7.03 master-table cell shows `3` points (correct after re-scope)
  - `.factory/stories/STORY-INDEX.md:92` — Wave 6 row: `S-W5.04, S-BL.LOOKUP, S-6.07, S-6.05, S-7.01, S-7.02, S-7.03 | 33 |`
  - `.factory/stories/STORY-INDEX.md:94` — Wave Summary Total row: `**34** (wave stories) | **200**`
  - `.factory/stories/STORY-INDEX.md:96` — Wave Summary narrative: `Wave 6 total: 7 stories, 33 pts` and `Total points including Wave 0: 200 (waves 0–7)`
  - `.factory/stories/STORY-INDEX.md:212` — changelog v3.46: `S-7.03 estimated_points 5→3`
  - `.factory/stories/S-7.03-console-remote-control.md:15` — `estimated_points: 3` (source of truth)

- **Symptom**:
  Wave 6 was documented as `40 pts baseline → 33 pts` (`-8` for S-7.04 removal, `+1` for S-BL.LOOKUP add) in the changelog v3.24 note. Separately, changelog v3.46 (2026-07-01) re-scoped S-7.03 from 5 → 3 points per RULING-W6TB-C, but the aggregate arithmetic in the Summary section, Wave Summary row, Wave Summary total row, and Wave Summary narrative gloss were never re-swept. Actual sum of Wave-6 cell values in the master table:
  ```
  S-W5.04(5) + S-BL.LOOKUP(1) + S-6.07(3) + S-6.05(3) + S-7.01(8) + S-7.02(8) + S-7.03(3) = 31 pts
  ```
  Reported values are 33 pts (wave row), 192 pts (Summary waves 0–6), 202 pts (incl. maintenance), and 200 pts (grand total waves 0–7). Correct values are 31, 183, 193, and 191 respectively — a systematic +2 offset in every rollup that includes Wave 6.

  An operator or new implementer reading the Summary section will compute wave-planning capacity from stale point totals, and the Wave Summary narrative gloss on line 96 states "Wave 6 total: 7 stories, 33 pts" as a specific factual claim that conflicts with the point cells themselves. This is the class of drift POL-002 exists to prevent: a story-file lifecycle change (re-scope 5→3) reflected in the master-table row but not in the aggregate rollups.

- **Verify steps**:
  1. Read `.factory/stories/STORY-INDEX.md` lines 33–34 and note "192" / "202" claims.
  2. Read lines 87–96 and note Wave 6 = 33 pts, Total = 200 pts.
  3. Enumerate Wave-6 point cells from master-table lines 72–77 + 79–80 (S-W5.04, S-BL.LOOKUP, S-6.07, S-6.05, S-7.01, S-7.02, S-7.03) → sum = 31.
  4. Read changelog line 212 (v3.46) which acknowledges S-7.03 5→3 re-scope but shows no compensating aggregate update.
  5. Confirm `estimated_points: 3` on S-7.03 story frontmatter (line 15).

- **Remediation shape** (spec-only edit; produces its own changelog row per POL-001 + POL-002):
  1. Line 33: `Total points (waves 0–6) | 192` → `183`.
  2. Line 34: `Total points (incl. S-M.01 + S-M.02) | 202` → `193`.
  3. Line 92 (Wave Summary row 6): trailing `| 33 |` → `| 31 |`.
  4. Line 94 (Wave Summary Total row): `**200**` → `**191**`.
  5. Line 96 narrative gloss:
     - `Wave 6 total: 7 stories, 33 pts` → `31 pts`
     - `net delta: S-7.04 removed (−8pts), S-BL.LOOKUP added (+1pt) = 33 pts` → extend note with `(and S-7.03 re-scoped 5→3 per RULING-W6TB-C v3.46) = 31 pts`
     - `Total points including Wave 0: 200 (waves 0–7)` → `191 (waves 0–7)`
  6. Add STORY-INDEX changelog row `3.75 | 2026-07-03 | F-P5P19-A-001: Wave-6 aggregate arithmetic sweep — S-7.03 re-scope 5→3 (v3.46) not previously propagated to Summary rollups; Wave 6 33→31, waves 0–6 total 192→183, incl. maintenance 202→193, grand total 200→191.`

- **Reopened deferral?** No — the "STORY-INDEX master-table completed/merged mixed status vocabulary" deferral is about status-label vocabulary. This finding concerns aggregate-point arithmetic, a distinct axis; not on the adjudicated-deferrals list.

## Anti-findings (things checked that passed)

- **AF-1** — Frontmatter-status sweep from F-P5P18-A-001 landed. Spot-verified `status: merged` in frontmatter of S-6.05:7 (v1.12), S-6.07:6 (v1.13), S-7.03:8 (v1.6), S-1.02:8 (rev 1.5), S-W3.05:8. No stragglers found among sampled stories.
- **AF-2** — VP-coverage refresh from F-P5P18-B-001 landed. STORY-INDEX.md:39 reads `VP coverage | 77/77 (100%)` with the VP-068..VP-077 gloss.
- **AF-3** — STORY-INDEX v3.74 changelog row (line 184) is well-formed per POL-001: names WHAT (Summary counter refresh + 8-story frontmatter sweep), WHY (POL-001+POL-002), and TRACEABILITY (F-P5P18-B-001 + F-P5P18-A-001).
- **AF-4** — Master-table `Complete` count (Summary line 24, `Complete | 34`) matches the enumerated list (34 completed/merged rows across S-0.01 … S-BL.ROUTER-ADDR).
- **AF-5** — Wave-6 master-table story count (7 rows: S-W5.04, S-BL.LOOKUP, S-6.07, S-6.05, S-7.01, S-7.02, S-7.03) matches Wave Summary line 92 "7 stories" and narrative line 96 "Wave 6 total: 7 stories".
- **AF-6** — Master-table draft row count (`Master-table drafts | 1 (S-W5.03)`, line 28) matches the sole remaining `draft` status row in the master table (S-W5.03 line 71). S-6.04 draft-stub is correctly separated in the Draft-stubs subsection (line 148), not double-counted.

VERDICT: HAS_FINDINGS
