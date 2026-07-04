---
pass_id: P5P22-Adv-A
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
  wall_clock_used: ~5 min
  file_reads_target: <=6
  file_reads_used: 7 (5 targeted Reads + 2 Grep frontmatter-status sweeps + Glob enumerations)
  overage_disclosure: on-budget (Reads within envelope; Grep/Glob calls used for census scans not deep reads)
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
    - F-P5P14-A-001 through F-P5P14-B-005 (adjudicated)
    - F-P5P15..F-P5P18 findings (all SHIPPED or adjudicated)
    - F-P5P19-A-001 SHIPPED at .factory e65e429
    - F-P5P19-B-001, F-P5P19-B-002, F-P5P19-B-003 (SHIPPED per Pass 19 remediation)
    - F-P5P20-A-001 SHIPPED at .factory 5fcf305 (STORY-INDEX v3.76 Wave-6 aggregate)
    - F-P5P20-B-001 SHIPPED at .factory 1e9fbff (ARCH-11 v1.17 + ARCH-07 v1.10 VP-043 Method column)
delivered_by: p5-pass22-adv-a
---

# Phase 5 Pass 22 Adv-A — findings

## Finding F-P5P22-A-001 — MEDIUM — POL-002 sibling-sweep gap: S-1.01 + S-2.01 story-frontmatter `status:` field is stale relative to STORY-INDEX Master Table row

**Confidence:** HIGH.

**Category:** spec-completeness / traceability — POL-002 row-sync + sibling-sweep drift on story-file frontmatter headers.

**Evidence (paths and lines):**

1. `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/STORY-INDEX.md:46` — Master Table row:
   `S-1.01 | Implement 44-byte outer header codec | E-1 | 1 | ... | completed`

   `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/S-1.01-frame-codec.md:8` — frontmatter:
   `status: ready`

2. `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/STORY-INDEX.md:49` — Master Table row:
   `S-2.01 | Implement HMAC-SHA256 frame authentication | E-2 | 2 | ... | completed (PR #5, merge 3c4104e)`

   `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/S-2.01-hmac-codec.md:7` — frontmatter:
   `status: pending`

**Why this is a finding, not an adjudicated preservation:**

STORY-INDEX v3.74 (`.factory/stories/STORY-INDEX.md:186`) landed a "systematic story-frontmatter status sync — 8 stories bumped to canonical `status: merged` (S-6.05, S-6.07, S-7.03, S-1.02, S-1.03, S-2.02, S-W3.04, S-W3.05)" — the same Wave-1..3 tier as S-1.01 and S-2.01. The v3.74 note's explicit adjudication scope is master-table cell **vocabulary** ("mixed `completed`/`merged` for Wave-0..4 vs later waves intentionally preserved to bound diff scope"). That preservation covers vocabulary choice (completed vs merged), not stale lifecycle values.

S-1.01 frontmatter `status: ready` and S-2.01 frontmatter `status: pending` are pre-merge lifecycle values, not vocabulary variants of "completed". Both stories are shipped:
- S-1.01 is Wave-1 root of the internal/frame DAG (`dependency-graph.md:23`), and STORY-INDEX Wave Summary line 87 records Wave 1 "CLOSED 2026-06-24".
- S-2.01 STORY-INDEX row cites PR #5 merge SHA `3c4104e`.

Both frontmatter values are objectively incorrect against the merged reality asserted by the Master Table row.

**Contrast with adjudicated Pass-18 scope:** F-P5P18-A-001 preserved master-table cell diversity ("completed (PR #X, merge Y)" vs "merged (PR #X, Y)"). It does NOT license `status: ready`/`status: pending` in story frontmatter when the story is merged — that is a distinct row-sync gap (POL-002 core: "STORY-INDEX Master Table rows must match story-file headers").

**Blast radius:** 2 story files → sibling-sweep. Per S-7.02 partial-fix regression discipline: v3.74's Wave-1 sweep (which included S-1.02 same-wave) was a same-layer partial fix that left S-1.01 unchanged. That is exactly the "same-subsystem BC/story sibling not updated" pattern flagged in the codified lessons.

**Actionable fix:**

1. Bump `S-1.01-frame-codec.md` frontmatter `status: ready` → `status: completed` (or `merged` if aligning to v3.74 canonical). Add a `last_modified` / `modified:` audit entry citing this pass.
2. Bump `S-2.01-hmac-codec.md` frontmatter `status: pending` → `status: completed` (or `merged`). Add audit entry.
3. STORY-INDEX v3.77 changelog row citing this finding-id, WHAT changed, WHY (POL-002 sibling-sweep gap for Wave-1..2 frontmatter status), and version delta (per POL-001).
4. Optionally sweep the other Wave-3 "completed" frontmatters (S-3.01a, S-3.01b, S-3.02, S-3.03, S-3.04) to the canonical spelling picked in v3.74; if vocabulary preservation is intentional, add an explicit adjudication note in STORY-INDEX §Changelog so future passes stop rediscovering the divergence.

**Severity rationale (MEDIUM):** POL-002 baseline for row drift is MED; sibling-sweep gap is MED-HIGH; the 2-file blast radius keeps this at MED. Not HIGH because both stories' primary anchors (STORY-INDEX rows, PR-merge citations, BC/VP traces) are correct — only the story-file lifecycle header is stale, so an implementer routed through STORY-INDEX gets correct guidance; only an implementer reading the story file first would be misled.

## Anti-findings (checked and passing)

- STORY-INDEX Summary line 23 arithmetic: `36 + 1 + 10 + 2 + 2 + 2 = 53` — matches "Total stories | 53". PASS.
- STORY-INDEX Summary line 24 "Complete | 34" enumerated list count = 34. PASS.
- STORY-INDEX Summary line 30 E-phase count = 32 (verified against Scope column of Master Table). PASS.
- STORY-INDEX Summary line 31 PE-phase count = 4 (S-7.01..S-7.04). PASS.
- v3.74 changelog row present and cites F-P5P18-A-001/B-001 (POL-001). PASS.
- v3.76 changelog row present and cites F-P5P20-A-001 with Wave-6 arithmetic delta (POL-001). PASS.
- S-6.06 frontmatter status: merged aligns with Master Table cell "merged (PR #36, 3ee9c38) (v1.25)". PASS.
- S-6.05 frontmatter status: merged aligns with Master Table cell "merged (PR #61, 7fe3e29)". PASS.
- S-7.03 frontmatter status: merged aligns with Master Table cell "merged (PR #60, 7142146)". PASS.
- BC-INDEX v3.1 status enumeration for BC-2.01.004..BC-2.01.006 correctly cites S-1.01 as `implemented (S-1.01 / PR #1)` — wait, actually BC-INDEX lines 30–32 read "active" not "implemented" for BC-2.01.004..006. Verified this is consistent with the BC-INDEX status-column convention where `active` covers non-final-shipping BCs; not treated as a finding (no evidence of BC-INDEX row-sync anchor to STORY-INDEX row status).
- S-BL.ADMIN-RECOVER-WIRE, S-BL.CLI-SURFACE-COMPLETION backlog stubs present and cited from STORY-INDEX v3.70 + v3.71 changelog rows (POL-001 compliant). PASS.
- Adjudicated deferrals: no reopening — F-P5P20-A-001/B-001 status remains SHIPPED at cited SHAs; STORY-INDEX v3.76 row confirms Wave-6 aggregate reconciliation.

---

VERDICT: HAS_FINDINGS
