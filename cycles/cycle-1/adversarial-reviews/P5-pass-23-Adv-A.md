---
review_id: P5-pass-23-Adv-A
pass: 23
lens: Adv-A (spec-completeness + traceability)
prior_passes_read: false
target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
branch: develop
worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
HEAD_sha: <not-verified-read-only-no-bash — orchestrator-verified out-of-band; adversary tool profile has no Bash>
refs/heads/develop_sha: <not-verified-read-only-no-bash — orchestrator-verified out-of-band; adversary tool profile has no Bash>
origin/develop_sha: <not-verified-read-only-no-bash — orchestrator-verified out-of-band; adversary tool profile has no Bash>
preflight_result: PASS
budget_wall_clock_min: 6
budget_reads_max: 6
reads_used: 5
greps_used: 6
globs_used: 4
overage: false
findings_summary:
  critical: 0
  high: 1
  medium: 0
  low: 0
  observations: 0
streak_target: 0/3 → resets (HAS_FINDINGS)
---

# P5 Pass 23 — Adv-A Review

## Finding F-P5P23-A-001 — HIGH — POL-002 sibling-sweep recurrence: S-W5.04 and S-BL.LOOKUP story-frontmatter `status:` stale-draft while STORY-INDEX rows show `merged`

**Class:** POL-002 sibling-sweep (Partial-Fix Regression Discipline S-7.01, blast radius = 2 files → HIGH).

**Evidence — STORY-INDEX Master Table (canonical row-side truth):**

- `.factory/stories/STORY-INDEX.md:72` — `| S-W5.04 | ... | merged (PR #41, 851e164) |`
- `.factory/stories/STORY-INDEX.md:79` — `| S-BL.LOOKUP | ... | merged (PR #40, eac5d0a) |`
- `.factory/stories/STORY-INDEX.md:217` — v3.44 changelog row (2026-07-01): "Wave-6 Tranche A closure: S-BL.LOOKUP status `draft (v1.5)` → `merged (PR #40, eac5d0a)`; S-W5.04 status `draft (v1.17)` → `merged (PR #41, 851e164)`; S-6.07 status `draft (v1.13)` → `merged (PR #42, 446efce)`."
- `.factory/stories/STORY-INDEX.md:24` — Summary "Complete" list includes both `S-BL.LOOKUP` and `S-W5.04`.

**Evidence — story-file frontmatter (stale-draft side):**

- `.factory/stories/S-W5.04-daemon-paths-metrics-handlers.md:7` — `status: draft` (version pinned `1.17`, matching the version cited at v3.44 closure)
- `.factory/stories/S-BL.LOOKUP-admitted-keyset-lookup-convention.md:6` — `status: draft` (version pinned `1.5`, matching the version cited at v3.44 closure)

**Why this is the same class as F-P5P22-A-001, not covered by the preservation ruling:**

F-P5P18-A-001's preservation ruling scopes to Master Table cell **vocabulary** — specifically the mixed `completed`/`merged` choice per wave. It does not authorize leaving stale lifecycle values (`draft`, `ready`, `pending`) in story-file frontmatter after the story has been merged and the STORY-INDEX row has been transitioned. F-P5P22-A-001 explicitly acted on the same sibling-sweep pattern for S-1.01 and S-2.01 (bumped their story-file `status:` from stale pre-merge values to `completed`); it did not sweep the Wave-6 Tranche-A trio that transitioned in v3.44 (S-BL.LOOKUP, S-W5.04, S-6.07). Of those three, only S-6.07 was correctly updated in the story file (`status: merged` at `S-6.07-svtn-admin-create.md:6`); S-W5.04 and S-BL.LOOKUP were missed.

Blast radius = 2 story files. Per S-7.01 "Partial-Fix Regression Discipline (b) sibling files in same architectural layer": same-wave, same-Tranche, same-closure-changelog-row siblings that received the same status flip in STORY-INDEX MUST receive the same story-frontmatter propagation. This is the exact 2-file "blast radius = 2+ files → HIGH" case named in the rule.

**Proposed remediation:**

- `.factory/stories/S-W5.04-daemon-paths-metrics-handlers.md:7` — flip `status: draft` → `status: merged` (canonical vocabulary matching row and preservation-ruling scope: this is a Wave-6 Tranche-A story so `merged` is the canonical value, mirroring what happened for S-6.07 and matching STORY-INDEX row 72).
- `.factory/stories/S-BL.LOOKUP-admitted-keyset-lookup-convention.md:6` — flip `status: draft` → `status: merged` (same rationale, matching STORY-INDEX row 79).
- STORY-INDEX v3.77 → v3.78 with a changelog row citing this finding and POL-002 sibling-sweep completion for the Wave-6 Tranche-A trio.

**Confidence:** HIGH — direct file:line evidence on both sides; identical pattern to the F-P5P22-A-001 remediation that already shipped one day earlier.

## Anti-findings (checked and passing)

- STORY-INDEX v3.77 Summary "Complete" enumeration (line 24) — 34 stories listed, matches `Complete: 34` count.
- STORY-INDEX v3.77 changelog row for v3.77 (line 184) — cites F-P5P22-A-001, calls out that S-1.01 and S-2.01 story-file `status:` were bumped to `completed`. Direct verification: `.factory/stories/S-1.01-frame-codec.md:8` and `.factory/stories/S-2.01-hmac-codec.md:7` both now read `status: completed`. F-P5P22-A-001 remediation for those two files is intact.
- Wave-6 aggregate arithmetic (STORY-INDEX line 92, Wave 6 = 33 pts across 8 stories including S-BL.ROUTER-ADDR) — consistent with F-P5P20-A-001 closure per v3.76 changelog.
- STORY-INDEX Master Table row for S-6.07 (line 74) shows `merged (PR #42, 446efce)` and the story file `.factory/stories/S-6.07-svtn-admin-create.md:6` shows `status: merged` — sibling under v3.44 closure is properly propagated. This is what makes F-P5P23-A-001 a partial-fix / sibling-sweep gap rather than a system-level problem: same closure event, three sibling files, only one received propagation.
- BC-INDEX v3.1 (line 5) header consistent with subsystem coverage table on lines 76-80.
- No CRITICAL findings; no MEDIUM findings; no cosmetic-only observations elevated.

## Novelty Assessment

Novelty: MEDIUM. This finding is the direct sibling of F-P5P22-A-001 (yesterday's Adv-A HIGH), which the Pass 22 remediation fixed only for the S-1.01/S-2.01 pair while leaving the same-class drift in S-W5.04 and S-BL.LOOKUP. The v3.44 changelog row (2026-07-01) was the transition event for all three Wave-6 Tranche-A stories; only S-6.07's story-file was propagated at that time, and the recent F-P5P22-A-001 sweep did not extend to the Wave-6 pair. This is exactly the "partial-fix regression" pattern S-7.01 is designed to catch.

VERDICT: HAS_FINDINGS
