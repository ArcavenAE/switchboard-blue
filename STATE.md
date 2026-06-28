---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-4-active
phase_3_active_wave: 4
phase_3_active_stories: [S-4.02, S-4.03]
phase_3_completed_stories: [S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03, S-4.01]
wave_2_gate_closed_at: 2026-06-25
wave_2_gate_disposition: "PASS_WITH_OBSERVATIONS"
product: switchboard
mode: greenfield
current_cycle: cycle-1
anchor_strategy: reference-via-frontmatter
phase_1_gate: APPROVED
phase_1_gate_date: 2026-06-24
phase_1_gate_disposition: approve-with-drift
phase_1_final_trajectory: "27 → 18 → 17 → 21 → 17 → 14 → 7 → 9"
phase_1_passes: 8
phase_2_gate: APPROVED
phase_2_gate_date: 2026-06-24
phase_2_gate_disposition: approve-proceed-to-wave-1
phase_2_complete: true
phase_2_epics: 8
phase_2_stories: 21
phase_2_waves: 7
phase_2_total_points: 132
phase_2_bc_coverage: "42/42"
l2_complete: true
l2_artifact_count: 11
l3_complete: true
l3_bc_count: 44
l3_cap_coverage: "30/30"
l4_complete: true
l4_vp_count: 58
arch_sections: 13
arch_adrs: 8
dtu_required: false
dtu_assessment: 2026-06-23
dtu_clones_built: n/a
dtu_services: []
wave_1_gate_closed_at: 2026-06-24
wave_1_gate_disposition: "pass-with-clean-drift"
wave_1_stories: "S-1.01 PR#1/1c76160, S-1.02 PR#2/9e9a98a, refactor PR#3/4be1b53 — all completed"
wave_2_complete: true
wave_2_stories: "S-2.01 PR#5/3c4104e, S-2.02 PR#6/a06b306, S-1.03 PR#7/f35e836 — all completed"
wave_2_points: 18
wave_3_stories_merged: 9
wave_3_points_complete: 48
wave_3_points_remaining: 0
wave_3_fix_prs: "I-1 PR#18/e9421d8, T2 PR#19/849bd86, C-1 PR#20/418de54 — all merged"
internal_packages: 18
plugin_version_adopted: "1.0.0-rc.21"
wave_3_gate_closed_at: 2026-06-27
wave_3_gate_disposition: "APPROVED — 3/3 adversary clean; 5 deferrals + process-gap #7 carried to Wave 4"
wave_3_stories_detail: "closed — see cycles/cycle-1/closed-stories.md + burst-log.md"
s_4_01_per_story_adversary_streak: 3
s_4_01_adversary_converged: true
s_4_01_impl_commit: aaff609
s_4_01_doc_commit: 327f5c6
s_4_01_pr_number: 24
s_4_01_pr_status: "MERGED (e415d31, 2026-06-28)"
s_4_01_merge_sha: e415d31
s_4_01_merge_date: 2026-06-28
s_4_01_status: completed
s_4_01_head_sha: ee75d83
s_4_01_demo_evidence: "7/7 ACs PASS (test-transcript based, S-W3.04 precedent); race-clean"
s_4_02_adversary_streak: 0
s_4_02_tip: 73781a4
s_4_02_tip_note: "comment/anchor-only changes since last clean pass; needs FRESH 3-consecutive-clean at 73781a4"
s_4_02_ruling: "RULING-002 + Amendment 1 — cycles/cycle-1/S-4.02/adversary/spec-adjudication.md"
s_4_03_adversary_streak: 0
s_4_03_tip: 34bc98f
s_4_03_tip_note: "3/3 clean at d4899ed; cosmetic relabel at 34bc98f → re-confirm 3 clean at 34bc98f before merge"
s_4_03_ruling: "RULING-003 v1.1 — cycles/cycle-1/S-4.03/adversary/ackseq-dos-ruling.md"
develop_head: 36c5e98
open_prs: 0
timestamp: 2026-06-28T22:00:00Z
last_update: 2026-06-28
---

# Switchboard Factory State

## Current State

Wave 4 ACTIVE. S-4.01 MERGED (e415d31, PR #24, 7/7 ACs, 3/3 adversary clean). S-4.02 (internal/replay) + S-4.03 (internal/arq) at final converged-candidate tips (73781a4 / 34bc98f) — comment/anchor/rename-only since last clean passes; FRESH confirmation rounds required before merge. develop HEAD = 36c5e98. 0 open PRs. DRIFT-S4.03-001 added (ADR-005 resync-on-reconnect deferred to S-5.01). Rulings on disk: RULING-002+Amendment-1 (S-4.02), RULING-003-v1.1 (S-4.03).

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 3: CLOSED; Wave 4: ACTIVE (S-4.01 MERGED; S-4.02/03/06.01 pending) | 2026-06-28 | Wave 3: 3/3 CLEAN; Wave 4: S-4.01 adversary 3/3 CLEAN @ aaff609; MERGED e415d31 |

## Wave / Story Status

Waves 1–3 complete (11 stories + 3 fix PRs, PRs #1–#20). Detail: `cycles/cycle-1/closed-stories.md`.

| Wave | Story | Title | Status | PR | SHA |
|------|-------|-------|--------|----|-----|
| 4 | S-4.01 | Per-path RTT/loss tracking + dedup/race dispatch | completed | #24 | e415d31 |
| 4 | S-4.02 | Upstream replay (internal/replay) | in-progress — confirm-round pending | — | 73781a4 |
| 4 | S-4.03 | Downstream ARQ + TLPKTDROP (internal/arq) | in-progress — confirm-round pending | — | 34bc98f |

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| W3-R2-M2 | MED | Route-time LWW snapshot: concurrent RegisterForwardingEntry not atomic with HMAC verify. | architect/implementer | open |
| SW305-M4 | MED | W4-TEST-001: RouteFrame fire-once E-ADM-017 integration test (real FailureCounter + WithNow). | test-writer | DEFER-WAVE-4 |
| process-gap-follow-up | OBS | Adversary nil-safety lens gap (missed SEC-001); lesson in lessons.md; candidate self-improvement story. | orchestrator | open/deferred |
| W3-DEFER-1 | OBS | Codify worktree-identity tuple in adversary dispatch templates. | orchestrator | deferred Wave 4 |
| W3-DEFER-2 | MED | M-1 relay busy-spin: double-failure-no-PTY not integration-tested. | implementer | deferred Wave 4/S-BL.NI |
| W3-DEFER-3 | MED | Fired-source LRU eviction-priority inversion (WithFailureCounter insertion-order, not fired-first). | implementer | deferred Wave 4 |
| W3-DEFER-4 | MED | M-2 unbounded E-ADM-016 log volume under sustained attack (BC-2.05.005 gap). | product-owner | deferred Wave 4 |
| W3-DEFER-5 | MED | EC-005: no CI lint rule enforces internal/ import boundary structurally. | devops-engineer | deferred Wave 4 |
| W3-DEFER-6 | MED | Real-connector PTY-EOF lifecycle integration test (mock-only today). | test-writer | deferred Wave 4 |
| S401-O3 | MED | BC-2.02.003 PC5: degraded-path flag (RTT >200ms) unimplemented in internal/paths. | product-owner/architect | deferred quality-indicator story |
| S402-F007 | LOW | S-4.02: ARCH-03 line 122 N=3 vs BC-2.02.004 N=5 — reconcile ARCH-03 (BC is authority). | architect | open |
| S403-O4 | LOW | S-4.03: DegradationEvent single-seq vs BC-2.02.006 PC2 range — per-frame drop OK for MVP. | product-owner | deferred MVP |
| S403-H1-DEFER | MED | S-4.03: retransmit-SEND PC3 deferred to router/multipath wiring story. | product-owner/architect | deferred |
| DRIFT-S4.03-001 | MED | ADR-005 resync-on-reconnect wire-mechanics deferred to S-5.01. | architect/implementer | open deferred S-5.01 |
Resolved items (C-1/OBS-3, T2, SW305-M1..M8, HF3, S402-F006, S403-O1, Phase-6 deferrals): `cycles/cycle-1/closed-drift.md`

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| HMAC algorithm | HMAC-SHA256, 16-byte tag, HKDF-SHA256 per-SVTN (ADR-001, ARCH-02/04) | 2026-06-23 |
| FEC group size | N=4 default; tunable (ADR-002, ARCH-03) | 2026-06-23 |
| Duplicate key registration | last-write-wins (ADR-003, ARCH-04) | 2026-06-23 |
| Console/access key permissions | control > console > access (ADR-004, ARCH-04) | 2026-06-23 |
| HMAC keying | per-(node, svtn) HKDF using node_admission_pubkey as IKM (ADR-001 amended) | 2026-06-23 |
| Marvel integration | explicitly deferred — no MVP integration | 2026-06-24 |
| Wave 3 gate APPROVED | 3/3 adversary clean; carry 5 deferrals + process-gap #7 to Wave 4 (2026-06-27) | 2026-06-27 |
| Per-story merge classifier (vsdd-factory#302) | Agent self-merge blocked; human-performed merge is correct resolution | 2026-06-27 |
| S-4.01 MERGED (e415d31, PR #24) | 7/7 ACs, 3/3 adversary clean; BC-2.02.009 router wiring deferred to S-4.04 | 2026-06-28 |
| S-4.02 RULING-002 + Amendment 1 | VP-042 removed; AC-004 per-call guard; BC-2.02.004 v1.3 invariant-5; AC-003 anchor corrected; story v1.2 | 2026-06-28 |
| S-4.03 RULING-003 v1.1 | ackSeq-DoS guard verified; EC-004→EC-005; BC-2.02.005 v1.3; ARCH-03 v1.3; DRIFT-S4.03-001 | 2026-06-28 |
| S-4.02 confirm-round at ce2ae7c → tip 73781a4 | 1/3 clean; F-001 HIGH + F-002 MED fixed via RULING-002; streak 0; re-confirm required | 2026-06-28 |
| S-4.03 confirm-round at d4899ed → tip 34bc98f | 3/3 CONVERGED; cosmetic relabel → streak 0; re-confirm at 34bc98f before merge | 2026-06-28 |
Older decisions (Wave 3 per-story): `cycles/cycle-1/burst-log.md` (archived 2026-06-28).

## Session Resume Checkpoint — 2026-06-28 (Wave 4 — S-4.02/S-4.03 confirmation-round pause)

**Position:** Phase 3 Wave 4 ACTIVE. S-4.01 merged (e415d31). S-4.02 + S-4.03 in parallel per-story delivery, both at final converged-candidate tips (S-4.02 tip 73781a4, S-4.03 tip 34bc98f). develop HEAD 36c5e98. 0 open PRs.

**NEXT ACTION on resume** (paused before final confirmation round to conserve context): run ONE fresh 6-pass adversarial confirmation round (3 passes per story, diverse lenses) at tips 73781a4 / 34bc98f. Both stories carry ONLY comment/anchor/rename changes since their last reviews — expectation 0C/0H — but BC-5.39.001 requires 3 consecutive clean passes on the FINAL unchanged artifact. If both converge 0C/0H ×3, proceed per per-story-delivery.md: demo-recorder → push → pr-manager full PR lifecycle → human-merge gate → worktree cleanup, for each story. Dispatch state-manager LAST.

**Worktrees:** feat/S-4.02-upstream-replay @ .worktrees/S-4.02 (tip 73781a4); feat/S-4.03-downstream-arq-tlpktdrop @ .worktrees/S-4.03 (tip 34bc98f). Both clean; just fmt/lint/test/test-race all pass.

**S-4.02 adversary history:** Pass-4 clean (pre-cleanup, superseded). Confirmation round at ce2ae7c: 1/3 clean; F-001 HIGH (test-docstring mis-anchor) + F-001/F-002 MED (AC-003 AC-004 anchors) all fixed via RULING-002 + Amendment 1. Streak = 0. Rulings on disk: cycles/cycle-1/S-4.02/adversary/spec-adjudication.md.

**S-4.03 adversary history:** 3/3 CONVERGED at d4899ed (RULING-003 v1.1 ackSeq-DoS guard verified). Cosmetic relabel at 34bc98f → recommend fresh confirm at 34bc98f. Streak = 0. Rulings on disk: cycles/cycle-1/S-4.03/adversary/ackseq-dos-ruling.md.

**Do NOT re-open settled rulings** (RULING-001/002/002-A1/003-v1.1) unless a fresh pass finds a NEW Critical/High.

**Context-pause rationale:** Session paused for context-compression management after ~1.3 convergence cycles (~28 agent runs since last compaction); the expensive 6-pass confirmation round + dual PR lifecycle was deferred to a fresh session to avoid mid-PR compaction.

**Open Drift Items:** W3-DEFER-1..6, W3-R2-M2, SW305-M4/W4-TEST-001, S401-O3, S402-F007, S403-H1-DEFER, S403-O4, DRIFT-S4.03-001 (see Drift Items table). Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.

## Historical Content

Burst logs, adversary passes, session checkpoints, closed-stories, closed-drift: `cycles/cycle-1/`
