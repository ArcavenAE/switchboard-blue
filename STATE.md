---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-3-adversarial-convergence-3of3-clean-consistency-audit-remediated-human-gate-pending
phase_3_active_wave: 3
phase_3_active_stories: []
phase_3_completed_stories: [S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03]
phase_3_pause_point: "S-W3.05 per-story adversary: 6 passes NOT_CONVERGED (restart passes 4-6: 0C/1H/2M/1L, 2C/1H/3L, 0C/1H). Blocking: msg-format (3/3), missing VP-059 proptest (C-2), AC-012 dead-key untested (H-1 pass-05). Wave-level gate r1+r2 CONVERGED, r3 NOT_CONVERGED (2 adj-HIGHs). All convergence paused pending fixes."
wave_2_gate_closed_at: 2026-06-25
wave_2_gate_disposition: "PASS_WITH_OBSERVATIONS"
wave_2_gate_consistency_validator: "PASS_WITH_OBSERVATIONS (0C/0H/2M/3L/4O)"
wave_2_gate_fresh_context_audit: "PASS_WITH_OBSERVATIONS (0C/0H/1M/3L/3O)"
wave_2_gate_governance_burst_sha: "c4ee7db"
wave_2_governance_arch_bump_sha: "1d09664"
wave_2_governance_vp_lifecycle_sha: "918acb4"
wave_2_governance_drift_rows_sha: "cdac793"
s_1_03_post_cleanup_develop_tip: "d8d7ae6"
e_fwd_002_pr_number: 8
e_fwd_002_merge_sha: d8d7ae6
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
s_1_01_merge_sha: 1c76160
s_1_01_pr_number: 1
s_1_02_merge_sha: 9e9a98a
s_1_02_pr_number: 2
s_1_02_alpha_tag: alpha-20260624-193019-9e9a98a
s_1_02_status: completed
s_2_01_merge_sha: 3c4104e
s_2_01_pr_number: 5
s_2_01_alpha_tag: alpha-20260625-023528-3c4104e
s_2_01_status: completed
s_2_02_merge_sha: a06b306
s_2_02_pr_number: 6
s_2_02_alpha_tag: alpha-20260625-135909-a06b306
s_2_02_status: completed
refactor_frametype_mtu_pr: 3
refactor_frametype_mtu_merge_sha: 4be1b53f85655110035de4f0f38422662afa2ed9
cicd_setup_complete: true
internal_packages: 18
plugin_version_adopted: "1.0.0-rc.21"
s_1_03_adversary_converged: "CONVERGED (passes 3–5 clean, 5 total) — cycles/cycle-1/S-1.03/adversary/"
s_1_03_merge_sha: f35e836
s_3_04_adversary_converged: "CONVERGED (passes 3–5 clean, 5 total) — cycles/cycle-1/S-3.04/adversary/"
s_3_04_merge_sha: d54bf1a
s_3_04_pr_number: 9
s_3_04_merge_date: 2026-06-26
s_3_04_status: completed
s_3_01a_adversary_converged: "CONVERGED (passes 13–15 clean, 15 total, 8 NOT+7 OK) — cycles/cycle-1/S-3.01a/adversary/"
s_3_01a_merge_sha: 43208ab
s_3_01a_pr_number: 11
s_3_01a_merge_date: 2026-06-26
s_3_01a_status: completed
s_3_01b_adversary_converged: "CONVERGED (passes 10–12 clean, 12 total, 9 NOT+3 OK) — cycles/cycle-1/S-3.01b/adversary/"
s_3_01b_merge_sha: 56ec9c7
s_3_01b_pr_number: 12
s_3_01b_merge_date: 2026-06-26
s_3_01b_status: completed
s_3_02_adversary_converged: "CONVERGED (passes 6-8 clean, 8 total; pass-5 1H/1M/2L → decayed to 0/0) — cycles/cycle-1/S-3.02/adversary/"
s_3_02_merge_sha: 1ff74f5
s_3_02_pr_number: 13
s_3_02_merge_date: 2026-06-27
s_3_02_status: completed
s_3_03_adversary_converged: "CONVERGED (passes 2-5 clean, 5 total; pass-1 1C/2H/3M decayed to 0/0; 4 consecutive clean passes) — cycles/cycle-1/S-3.03/adversary/"
s_3_03_merge_sha: b68e498
s_3_03_pr_number: 14
s_3_03_merge_date: 2026-06-27
s_3_03_status: completed
s_3_03_alpha_tag: alpha-20260627-042402-b68e498
s_1_03_pr_number: 7
s_1_03_merge_date: 2026-06-25
s_1_03_status: completed
wave_2_complete: true
wave_2_stories_merged: 3
wave_2_points: 18
wave_3_stories_merged: 7
wave_3_points_complete: 48
wave_3_points_remaining: 0
s_3_01a_supporting_merge_pr10: "BC-5.38.001 chore cleanup merged during S-3.01a lifecycle"
wave_3_gate_adversary_streak: 3
wave_3_gate_adversary_converged: true
wave_3_gate_pass1_disposition: "C-1 deferred (C-1-W3P1-defer/S-BL.NI, ARCH-08 v2.2 §6.5.1); I-1 fixed PR #18 e9421d8"
wave_3_gate_pass2_disposition: "CONVERGED 0C/0H — contract-conformance"
wave_3_gate_pass3_disposition: "CONVERGED 0C/0H — security"
wave_3_gate_convergence_summary: "3/3 CLEAN passes (Pass-1 concurrency/lifecycle, Pass-2 contract-conformance, Pass-3 security); consistency-audit HIGH Finding-4.1 downgraded to traceability-only (T2 satisfied in code: TestForwardFramesTOCTOUCount50 + deterministic swapBarrier test)"
wave_3_gate_human_gate: PENDING
w3_i1_fix_pr: 18
w3_i1_fix_merge_sha: e9421d8
w3_i1_fix_merge_date: 2026-06-27
s_w3_05_per_story_adversary_streak: 3
wave_3_gate_adversary_passes: "RESTART run at 10dd880: r1 CONVERGED 0C/0H; r2 CONVERGED 0C/0H/4M; r3 NOT_CONVERGED 0C/2H (F-1 cmd-wiring, F-2 EC-006 alert — both ADJUDICATION-DEPENDENT scope-boundary findings, NOT raw code defects). Passes r1+r2 rated F-1 as in-scope-deferred OBSERVATION. ADJUDICATION IN PROGRESS: architect (F-1 cmd-wiring deferral vs ARCH-08 position-18) + product-owner (F-2 EC-006 ownership/deferral). Convergence on hold pending scope decision."
s_wave3_f1_fix_pr: 15
s_wave3_f1_fix_merge_sha: 10dd880
s_wave3_f1_fix_merge_date: 2026-06-27
s_w3_05_adversary_status: "RE-CONVERGED at f6038d2 — 3 fresh passes (10,11,12); 0C/0H. Streak reset by SEC-001 HIGH (nil-logger deref, CWE-476) found post-5c3d7ea, fixed f6038d2. 2026-06-27."
s_w3_05_adversary_converged: "RE-CONVERGED (passes 10-12 clean at f6038d2, 12 total, 6 NOT+3 superseded+3 OK) — cycles/cycle-1/S-W3.05/adversary/"
s_w3_05_impl_commit: f6038d2
s_w3_05_test_commit: 5c3d7ea
s_w3_05_pr_number: 16
s_w3_05_pr_status: "MERGED (fa6345e, 2026-06-27)"
s_w3_05_merge_sha: fa6345e
s_w3_05_merge_date: 2026-06-27
s_w3_05_status: completed
s_w3_04_adversary_converged: "CONVERGED (passes 10/11/12 clean, 3 consecutive; 0C/0H) at tip 1c3c864; comment-only 77c6229 preserves convergence (zero behavioral delta) — cycles/cycle-1/S-W3.04/adversary/"
s_w3_04_impl_commit: 1c3c864
s_w3_04_comment_follow_up_sha: 77c6229
s_w3_04_demo_evidence: "test-transcripts AC-001..AC-009; AC-005 indirect (story-declared); AC-008 SKIP sandbox (no /dev/ptmx) — PC-2 confirmed TestRunAccessWithConnectorPC2"
s_w3_04_pr_number: 17
s_w3_04_merge_sha: aeb442d
s_w3_04_merge_date: 2026-06-27
s_w3_04_status: completed
timestamp: 2026-06-27T22:00:00Z
last_update: 2026-06-27
---

# Switchboard Factory State

## Current State

Wave-3 wave-level adversarial convergence COMPLETE — 3/3 CLEAN passes achieved (Pass-1 concurrency/lifecycle 0C/0H, Pass-2 contract-conformance 0C/0H, Pass-3 security 0C/0H). Fresh-context consistency audit remediated: lone HIGH Finding-4.1 (ARCH-01 v1.6 T2 binding) downgraded to traceability-only; T2 IS satisfied in code (TestForwardFramesTOCTOUCount50 + deterministic swapBarrier test); S-W3.04 v1.4 + ARCH-INDEX backfill committed. Human gate PENDING. Non-blocking deferred findings remain open: M-1 relay busy-spin, fired-source LRU eviction-priority, M-2 log-volume cardinality, OBS-3 no-CI-guard partial-wiring.

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 3 gate: HUMAN_GATE_PENDING | 2026-06-27 | Wave 3: 3/3 CLEAN passes (concurrency/lifecycle, contract-conformance, security); consistency-audit F-4.1 resolved traceability-only |

## Wave / Story Status

| Wave | Story | Title | Status | PR | SHA |
|------|-------|-------|--------|----|-----|
| 1 | S-1.01 | Frame codec | completed | #1 | 1c76160 |
| 1 | S-1.02 | Half-channel clock | completed | #2 | 9e9a98a |
| 1 | refactor | FrameType + MTU | completed | #3 | 4be1b53 |
| 2 | S-2.01 | HMAC codec | completed | #5 | 3c4104e |
| 2 | S-2.02 | Admission + SVTN isolation | completed | #6 | a06b306 |
| 2 | S-1.03 | Session continuity | completed | #7 | f35e836 |
| 3 | S-3.04 | HMAC RouteFrame wire-up | completed | #9 | d54bf1a |
| 3 | S-3.01a | Tmux control mode integration | completed | #11 | 43208ab |
| 3 | S-3.01b | PTY proxy fallback | completed | #12 | 56ec9c7 |
| 3 | S-3.02 | Console attach/detach + multi-console | completed | #13 | 1ff74f5 |
| 3 | S-3.03 | Tier-2 per-session authorization | completed | #14 | b68e498 |
| 3 | S-W3.05 | HMAC failure counter + E-ADM-017 | completed | #16 | fa6345e |
| 3 | S-W3.04 | Full daemon assembly | completed | #17 | aeb442d |
| 3 | fix/W3-i1 | Ticker wg-join (I-1, BC-2.04.007) | completed | #18 | e9421d8 |

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| W3-R3-F1 | HIGH? | cmd wiring: none of 5 Wave-3 subsystems wired; E-ADM-016 discarded by nopLogger in real builds. | architect | pending-adjudication |
| W3-R3-F2 | HIGH? | BC-2.05.008 EC-006 ownership/deferral — S-W3.05 implements the mechanism; ratification pending. | product-owner | pending-adjudication |
| W3-R2-M2 | MED | Route-time LWW snapshot: concurrent RegisterForwardingEntry not atomic with HMAC verify. | architect/implementer | open |
| SW305-M2 | MED | WithFailureCounter uses unexported iface not spec-pinned *admission.FailureCounter — ratify or revert. | product-owner | open |
| SW305-M3 | MED | WithNow clock seam + threshold<=0 guard absent from BC contract. | product-owner→implementer | open |
| SW305-M4 | MED | Integration test doesn't pin fire-once end-to-end (no 6th/7th through RouteFrame). | test-writer | open |
| SW305-cosmetic | LOW | Stale comments: Red-Gate test (pre-v1.6 model), TrackedSourceCount() name, AC-016 count, v1.7 citation in test header. | cosmetic | defer post-wave |
| process-gap-follow-up | OBS | Adversary nil-safety lens gap (missed SEC-001) — lesson recorded in cycles/cycle-1/lessons.md. Follow-up: candidate for self-improvement epic story. | orchestrator | open/deferred |
| C-1-W3P1-defer | DEFER | FailureCounter/E-ADM-017 daemon wiring deferred to S-BL.NI (network-ingress). RouteFrame test-only in Wave 3; wiring now is dead code. ARCH-08 v2.2 §6.5.1 TRACKED-DEFER (14a61d2). S-BL.NI MUST wire routing.WithFailureCounter(fc) + E-ADM-017 integration test. Refs: BC-2.05.005 PC-3, S-W3.05 AC-6. | architect-ruled | intentional-deferred/S-BL.NI |
Resolved SW305-M1/M5/M6/M7/M8/HF3 + stable Phase-6 deferrals + wave-gate rows: `cycles/cycle-1/closed-drift.md`
Drift note: BC-2.05.005 v1.8 + VP-059 v1.2 + story v1.3 + taxonomy v2.0 are hygiene/precondition-sanction only; message-format string UNCHANGED. Wave-3 gate drift check may flag input-hash change — expected; clear after confirming format-string identity.

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| HMAC algorithm | HMAC-SHA256, 16-byte tag, HKDF-SHA256 per-SVTN (ADR-001, ARCH-02/04) | 2026-06-23 |
| FEC group size | N=4 default; tunable (ADR-002, ARCH-03) | 2026-06-23 |
| Duplicate key registration | last-write-wins (ADR-003, ARCH-04) | 2026-06-23 |
| Console/access key permissions | control > console > access (ADR-004, ARCH-04) | 2026-06-23 |
| HMAC keying | per-(node, svtn) HKDF using node_admission_pubkey as IKM (ADR-001 amended) | 2026-06-23 |
| Wave-1 rollback/re-closure | all drift items routed concretely; vsdd-factory#260 | 2026-06-24 |
| Marvel integration | explicitly deferred — no MVP integration | 2026-06-24 |
| S-3.03 repointed 5→8 | upstream-wiring scope expansion; Wave 3 total 29→32 pts | 2026-06-27 |
| S-W3.05 E-ADM-017 msg-format adjudication CORRECTED | specs authoritative — include "HMAC failure rate alert:" phrase; code/tests/story AC-003/AC-015 conform | 2026-06-27 |
| S-W3.05 re-arm semantics finalized | drain-only re-arm + per-source append-skip; reconciled BC-2.05.005 v1.6/VP-059 v1.1 | 2026-06-27 |
| S-W3.05 CONVERGED + SEC-001 fixed + PR #16 merged | 3 clean passes (10-12) at f6038d2; fa6345e | 2026-06-27 |
| S-W3.04 CONVERGED (BC-5.39.001) + PR #17 merged | 3 clean passes (10-12) at 1c3c864; aeb442d | 2026-06-27 |
| Per-story-delivery merge-handoff pathology (vsdd-factory#302) | Agent self-merge blocked by classifier; human-performed merge is the correct resolution | 2026-06-27 |
| Wave-3 Pass-1: C-1 deferred, I-1 fixed PR #18 e9421d8 | C-1 → ARCH-08 v2.2 §6.5.1 TRACKED-DEFER/S-BL.NI; I-1 (BC-2.04.007) fixed; streak 0/3 | 2026-06-27 |

## Session Resume Checkpoint — 2026-06-27 (Wave 3 convergence 3/3 CLEAN; consistency-audit remediated; human gate PENDING)

**Position:** Phase 3, Wave 3. All Wave-3 stories merged + I-1 fix merged. Wave-level adversarial convergence COMPLETE (3/3 CLEAN passes).
**Convergence summary:** Pass-1 concurrency/lifecycle 0C/0H (C-1 deferred ARCH-08 v2.2 §6.5.1/S-BL.NI; I-1 fixed PR #18). Pass-2 contract-conformance 0C/0H. Pass-3 security 0C/0H. Consistency-audit Finding-4.1 HIGH downgraded to traceability-only — T2 satisfied in code (TestForwardFramesTOCTOUCount50 + deterministic swapBarrier test); resolved via S-W3.04 v1.4 + ARCH-INDEX backfill.
**Next immediate step:** Human gate review of Wave 3. Gate PENDING approval.
**Non-blocking deferred findings (open):** M-1 relay busy-spin, fired-source LRU eviction-priority, M-2 log-volume cardinality, OBS-3 no-CI-guard partial-wiring.
**Wave-gate open drift:** C-1-W3P1-defer (intentional, S-BL.NI target). W3-R3-F2 EC-006 ownership (PO adjudication pending). SW305-M2/M3/M4 open/deferred.
**Open Wave-4 follow-ups (3 items):** (a) EC-005 durable CI import-perimeter guard; (b) real-connector PTY-EOF lifecycle integration test; (c) embed worktree-identity tuple in adversary dispatches.
**Previous checkpoint:** `cycles/cycle-1/session-checkpoints.md`.

## Historical Content

Burst logs, adversary passes, session checkpoints, closed-stories, closed-drift: `cycles/cycle-1/`
