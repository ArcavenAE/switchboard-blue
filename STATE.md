---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-4-active
phase_3_active_wave: 4
phase_3_active_stories: []
phase_3_completed_stories: [S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03, S-4.01, S-4.02, S-4.03]
s_4_04_adversary_streak: 3
s_4_04_adversary_converged: true
s_4_04_impl_sha: 24c4378
s_4_04_impl_branch: feat/S-4.04-split-horizon-drop-cache
s_4_04_status: adversary-converged
s_6_01_adversary_streak: 3
s_6_01_adversary_converged: true
s_6_01_impl_sha: 37d45fa
s_6_01_impl_branch: feat/S-6.01-config-validation
s_6_01_status: adversary-converged
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
s_4_02_adversary_streak: 3
s_4_02_adversary_converged: true
s_4_02_tip: d7a1925
s_4_02_pr_number: 25
s_4_02_pr_status: "MERGED (95729c7, 2026-06-28)"
s_4_02_merge_sha: 95729c7
s_4_02_merge_date: 2026-06-28
s_4_02_status: completed
s_4_02_demo_evidence: "4/4 ACs PASS, race-clean"
s_4_02_ruling: "RULING-002 + Amendment 1 — cycles/cycle-1/S-4.02/adversary/spec-adjudication.md"
s_4_03_adversary_streak: 3
s_4_03_adversary_converged: true
s_4_03_tip: 02f317d
s_4_03_pr_number: 26
s_4_03_pr_status: "MERGED (8d9744f, 2026-06-28)"
s_4_03_merge_sha: 8d9744f
s_4_03_merge_date: 2026-06-28
s_4_03_status: completed
s_4_03_demo_evidence: "5/5 ACs PASS, race-clean"
s_4_03_ruling: "RULING-003 v1.1 — cycles/cycle-1/S-4.03/adversary/ackseq-dos-ruling.md; F-A-001 VP-052 mis-anchor fixed @ 02f317d (re-anchored to BC-2.02.005 SACK-accuracy / VP-019-020; story v1.1)"
develop_head: 8d9744f
open_prs: 0
timestamp: 2026-06-28T23:59:00Z
last_update: 2026-06-28
---

# Switchboard Factory State

## Current State

Wave 4 ACTIVE — 3/5 stories merged; 2/5 adversary-converged awaiting delivery. S-4.01 MERGED (e415d31, #24). S-4.02 MERGED (95729c7, #25). S-4.03 MERGED (8d9744f, #26). S-4.04 ADVERSARY-CONVERGED (3/3 clean, HEAD 24c4378, branch feat/S-4.04-split-horizon-drop-cache). S-6.01 ADVERSARY-CONVERGED (3/3 clean, HEAD 37d45fa, branch feat/S-6.01-config-validation). develop HEAD = 8d9744f. 0 open PRs. Next: per-story delivery (demo → push → PR → human-merge gate → cleanup) for S-4.04 then S-6.01.

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 3: CLOSED; Wave 4: ACTIVE (S-4.01/S-4.02/S-4.03 MERGED; S-4.04/S-6.01 ADVERSARY-CONVERGED, pending delivery) | 2026-06-28 | Wave 3: 3/3 CLEAN; Wave 4: S-4.01/S-4.02/S-4.03 MERGED; S-4.04/S-6.01 3/3 CONVERGED |

## Wave / Story Status

Waves 1–3 complete (11 stories + 3 fix PRs, PRs #1–#20). Detail: `cycles/cycle-1/closed-stories.md`.

| Wave | Story | Title | Status | PR | SHA |
|------|-------|-------|--------|----|-----|
| 4 | S-4.01 | Per-path RTT/loss tracking + dedup/race dispatch | MERGED | #24 | e415d31 |
| 4 | S-4.02 | Upstream replay (internal/replay) | MERGED | #25 | 95729c7 |
| 4 | S-4.03 | Downstream ARQ + TLPKTDROP (internal/arq) | MERGED | #26 | 8d9744f |
| 4 | S-4.04 | Split-horizon loop prevention + drop-cache router wiring | ADVERSARY-CONVERGED | — | 24c4378 |
| 4 | S-6.01 | Config parsing and validation | ADVERSARY-CONVERGED | — | 37d45fa |

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
| S403-COS1 | OBS | S-4.03 arq.go SACKPopCount doc comment stale ref to "encoding/binary" (uses package-local bitmapToUint64). | implementer | carry to S-4.04 or maintenance sweep (artifact merged at 8d9744f; cosmetic-only) |
| S403-COS2 | OBS | S-4.03 arq_test.go:949 leftover "GREEN-BY-DESIGN per stub notes" docstring (now a real characterisation test). | implementer | carry to S-4.04 or maintenance sweep (artifact merged at 8d9744f; cosmetic-only) |
| S601-NITPICK-A | NITPICK | S-6.01 story File Structure table omits cmd/switchboard/access.go though Task 17 mandates modifying it (doc completeness gap). | story-writer | cycle-close |
| S601-NITPICK-B | NITPICK | S-6.01 story EC ids diverge from BC EC ids (e.g. keepalive story-EC-012 vs BC-EC-009) — cosmetic id drift. | story-writer | cycle-close |
| S601-NITPICK-C | NITPICK | S-6.01 E-CFG-005 reused for non-regular/too-large files; E-CFG-004 reused for non-ErrNotExist open/stat/read errors — no dedicated BC code. | product-owner | cycle-close cosmetic |
| S601-NITPICK-D | NITPICK | S-6.01 ValidationError.Error() inserts "value" token not present in BC canonical template (byte-level cosmetic). | implementer | cycle-close |
| S601-NITPICK-E | OBS | S-6.01 yaml.v3 billion-laughs bound is implicit/library-version-dependent — optional decode-site comment suggested. | implementer | cycle-close optional |
| S404-OBS-F | OBS | S-4.04 E-FWD-001 emission is per-event/not-rate-limited (unlike EC-005 sibling); LATENT CWE-779 only if production caller makes eligible-interface set attacker-steerable. Deferred cross-story. | architect/product-owner | re-confirm when production caller lands |
| S404-OBS-G | OBS | S-4.04 BC-2.02.008 PC-4 (split-horizon/drop-cache independence) has no dedicated negative test — satisfied structurally. | test-writer | cycle-close |
| BC-2.09.003-STALE | NITPICK | BC-2.09.003 traceability table + Story Anchor say "AC-001 through AC-006"; story now reaches AC-009. Needs refresh to AC-009. | story-writer/spec-steward | cycle-close |
| S601-DRAFT-STORY | OBS | Dedicated SIGHUP/reload story (BC-2.09.003 Inv-3/EC-004) to be opened as draft. | product-owner | cycle-close |
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
| Wave 3 gate APPROVED | 3/3 adversary clean; carry 5 deferrals + process-gap #7 to Wave 4 | 2026-06-27 |
| Per-story merge classifier (vsdd-factory#302) | Agent self-merge blocked; human-performed merge is correct resolution | 2026-06-27 |
| S-4.01 MERGED (e415d31, PR #24) | 7/7 ACs, 3/3 adversary clean; BC-2.02.009 router wiring deferred to S-4.04 | 2026-06-28 |
| S-4.02 RULING-002 + Amendment 1 | VP-042 removed; AC-004 per-call guard; BC-2.02.004 v1.3 invariant-5; AC-003 anchor corrected; story v1.2 | 2026-06-28 |
| S-4.03 RULING-003 v1.1 | ackSeq-DoS guard verified; EC-004→EC-005; BC-2.02.005 v1.3; ARCH-03 v1.3; DRIFT-S4.03-001 | 2026-06-28 |
| S-4.02 CONVERGED 3/3 + PR #25 | fresh 6-pass confirmation round 3/3 clean @ 73781a4; CI fix @ d7a1925 (race-build-tag latency skip); merge-ready | 2026-06-28 |
| S-4.03 F-A-001 VP-052 mis-anchor (NEW HIGH) | fresh confirm-round Pass A found VP-052 (belongs to S-5.01/internal/metrics) wrongly anchored to SACK pop-count; fixed @ 02f317d + story v1.1; re-confirmed 3/3 clean; PR #26 merge-ready | 2026-06-28 |
| S-4.02 MERGED (95729c7, PR #25) | 4/4 ACs, 3/3 adversary clean; squash-merged into develop; CI all SUCCESS | 2026-06-28 |
| S-4.03 MERGED (8d9744f, PR #26) | 5/5 ACs, 3/3 adversary clean; F-A-001 VP-052 fix included; squash-merged via auto-merge; CI all SUCCESS | 2026-06-28 |
| S-4.04 ADVERSARY-CONVERGED (24c4378) | 7/7 ACs (AC-007 E-FWD-001 added v1.5); 3 consecutive 6-lens clean rounds (spec/BC↔AC; security/CWE; concurrency/race); BC-5.39.001 C=0 H=0 M=0. Pending demo/push/PR/merge. | 2026-06-28 |
| S-6.01 ADVERSARY-CONVERGED (37d45fa) | 9/9 ACs; 3 consecutive 6-lens clean rounds; final fix 37d45fa (io.LimitReader + close TOCTOU, F-SEC-L1). All prior findings closed with regression tests. BC-5.39.001 NITPICK_ONLY. Pending demo/push/PR/merge. | 2026-06-28 |
Older decisions (Wave 3 per-story): `cycles/cycle-1/burst-log.md` (archived 2026-06-28).

## Session Resume Checkpoint — 2026-06-28 (Wave 4 — S-4.04 + S-6.01 delivery next)

**Position:** Phase 3 Wave 4. S-4.01 MERGED (#24, e415d31). S-4.02 MERGED (#25, 95729c7). S-4.03 MERGED (#26, 8d9744f). S-4.04 ADVERSARY-CONVERGED (3/3, HEAD 24c4378, branch feat/S-4.04-split-horizon-drop-cache). S-6.01 ADVERSARY-CONVERGED (3/3, HEAD 37d45fa, branch feat/S-6.01-config-validation). develop HEAD = 8d9744f. 0 open PRs.

**NEXT ACTION on resume:** Per-story delivery for S-4.04 then S-6.01 (demo → push → PR → human-merge gate → worktree cleanup). After both merged: Wave 4 integration gate + wave-gate.

**S-4.04 notes:** 7/7 ACs. AC-007 (E-FWD-001 emission on all-paths split-horizon drop, BC-2.02.008 PC-3) added v1.5 and fully traced/tested. Branch feat/S-4.04-split-horizon-drop-cache at 24c4378.

**S-6.01 notes:** 9/9 ACs. Final fix 37d45fa (io.LimitReader + TOCTOU close, F-SEC-L1). Branch feat/S-6.01-config-validation at 37d45fa. Residual NITPICKs logged in Drift Items table (S601-NITPICK-A..E); all below re-open bar, deferred to cycle-close.

**Settled rulings:** RULING-001/002/002-A1/003-v1.1 and F-A-001 (VP-052 re-anchored) — do NOT re-open unless a fresh pass finds a NEW Critical/High.

**Open Drift Items:** W3-DEFER-1..6, W3-R2-M2, SW305-M4/W4-TEST-001, S401-O3, S402-F007, S403-H1-DEFER, S403-O4, DRIFT-S4.03-001, S403-COS1, S403-COS2, S601-NITPICK-A..E, S404-OBS-F/G, BC-2.09.003-STALE, S601-DRAFT-STORY (see Drift Items table). Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.

## Historical Content

Burst logs, adversary passes, session checkpoints, closed-stories, closed-drift: `cycles/cycle-1/`
