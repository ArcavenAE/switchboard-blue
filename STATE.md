---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-2-integration-gate-closed
phase_3_active_wave: 3
phase_3_active_stories: []
phase_3_completed_stories: [S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02]
phase_3_pause_point: "Wave 3 in progress — S-3.04 (PR #9), S-3.01a (PR #11), S-3.01b (PR #12), S-3.02 (PR #13) MERGED. 1 story remaining: S-3.03."
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
l3_bc_count: 43
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
s_1_03_pr_number: 7
s_1_03_merge_date: 2026-06-25
s_1_03_status: completed
wave_2_complete: true
wave_2_stories_merged: 3
wave_2_points: 18
wave_3_stories_merged: 4
wave_3_points_complete: 24
wave_3_points_remaining: 5
s_3_01a_supporting_merge_pr10: "BC-5.38.001 chore cleanup merged during S-3.01a lifecycle"
timestamp: 2026-06-27T12:00:00Z
last_update: 2026-06-27
---

# Switchboard Factory State

## Current State

Phase 3, Wave 3. S-3.02 MERGED (PR #13, 1ff74f5). Wave 3: 4/5 stories (24/29 pts).
4 tech-debt items carried forward (F-002, F-003, F-004, SEC-001). VP-032 deferred.
Next and FINAL Wave 3 story: S-3.03 (Tier-2 per-session authorization, BC-2.04.005 + BC-2.05.003), then Wave 3 integration gate.
Drift item S-3.02-FM1 (vestigial upstream channel) deferred to S-3.03.

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 2 gate: PASS_WITH_OBSERVATIONS | 2026-06-25 | Wave 2: 3/3 done; Wave 3: 4/5 done (S-3.04 PR #9; S-3.01a PR #11; S-3.01b PR #12; S-3.02 PR #13 1ff74f5) |

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
| 3 | S-3.03 | Tier-2 per-session authorization | pending | — | — |

Wave 2: 3/3 stories merged (18 pts). Gate: PASS_WITH_OBSERVATIONS — CLOSED 2026-06-25.
Wave 3: 4/5 stories merged (24/29 pts). S-3.04 CLOSED 2026-06-26. S-3.01a CLOSED 2026-06-26. S-3.01b CLOSED 2026-06-26. S-3.02 CLOSED 2026-06-27 (PR #13). Next: S-3.03 (Tier-2 auth, 5 pts) → Wave 3 integration gate.
Gate reports: `cycles/cycle-1/wave-2/`. S-3.04 adversary reports: `cycles/cycle-1/S-3.04/adversary/`.

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| F-P8-004 | MED | VP-026 cites "transitivity" invariant missing from BC-2.02.003 | architect | open — Phase 3 test-writing for BC-2.02.003 |
| F-P8-005 | MED | VP-027 title "degradation goes down" but harness tests recovery direction | architect | open — Phase 3 test-writing |
| F-P8-009 | LOW | feasibility-report:61 deployment-ops range "(CAP-026–027)" should be "(CAP-026–028)" | architect | open — Phase 2 deferred |
| F-003 | LOW | Payload-MTU composed wire-format test | story S-BL.OA | deferred to outer-assembler story |
| F-004 | LOW | ARCH-02 channel-header serializer not implemented | story S-BL.OA | deferred to outer-assembler story |
| VP-036 testenv | Phase-6 hardening | property test (TestProperty_VP036_SessionContinuity) deferred until internal/testenv.ConnectWithSourceIP exists | 2026-06-25 |
| SEC-003 | Phase-6 hardening | Sub-microsecond TOCTOU on now in ReAuthenticate; accepted per pr-reviewer PR #7 security review | 2026-06-25 |
| WAVE-2-MED-001 | Phase-6 hardening | ReAuthState not evicted on RevokeKey/RegisterKey reset; stale source-IP survives via CurrentSourceAddr | 2026-06-25 |
| VP-039-test-skip | Phase-6 hardening | t.Skip placeholder needed in internal/routing/*_test.go for VP-039 (deferred property test) | 2026-06-25 |
| S-3.02-FM1 | MED | Upstream channel from Attach is vestigial in production (SendKeystroke forwards directly to sink); close-race contract guards an unwritten path. Reconcile BC-2.04.003 PC-3 to SendKeystroke path OR stop returning the channel until S-3.03 adds a draining consumer | architect/story-writer | deferred to S-3.03 |

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| HMAC algorithm | HMAC-SHA256, 16-byte tag, HKDF-SHA256 per-SVTN key derivation (ADR-001, ARCH-02/04) | 2026-06-23 |
| FEC group size | N=4 default; tunable (ADR-002, ARCH-03) | 2026-06-23 |
| Duplicate key registration | last-write-wins (ADR-003, ARCH-04) | 2026-06-23 |
| Console/access key permissions | control > console > access (ADR-004, ARCH-04) | 2026-06-23 |
| Downstream ARQ failover | resync from last ACK; in-flight lost (ADR-005, ARCH-03) | 2026-06-23 |
| HMAC keying | per-(node, svtn) HKDF using node_admission_pubkey as IKM (amended ADR-001) | 2026-06-23 |
| Wave-1 rollback/re-closure | all drift items routed concretely; drbothen/vsdd-factory#260 | 2026-06-24 |
| Marvel integration | explicitly deferred — no MVP or PE-phase integration | 2026-06-24 |

## Session Resume Checkpoint — 2026-06-27 (S-3.02 MERGED)

**Position:** Phase 3, Wave 3. S-3.02 MERGED (PR #13, squash SHA 1ff74f5, develop tip). Wave 3: 4/5 stories merged (24/29 pts).

**S-3.02:** Completed. 8 adversary passes (passes 6-8 CONVERGED). PR #13 approved (pr-reviewer: 0 blocking), CI green, squash-merged 2026-06-27. Drift item S-3.02-FM1 (vestigial upstream channel) deferred to S-3.03.

**Tech-debt carry-forward (tech-debt-register.md):** F-002, F-003, F-004 (Wave 4), SEC-001 (Phase-6). VP-032 deferred.

**Next:** S-3.03 (Tier-2 per-session authorization, BC-2.04.005 + BC-2.05.003, 5 pts) — FINAL Wave 3 story. After S-3.03 merge: Wave 3 integration gate. Open Phase-6 deferred items: see Open Drift Items table above.

## Historical Content

Burst logs, adversary passes, session checkpoints, and closed-story narratives: `cycles/cycle-1/`
(burst-log.md, convergence-trajectory.md, session-checkpoints.md, closed-stories.md, wave-1/,
wave-2/, S-1.02/adversary/, S-2.01/adversary/, S-2.02/adversary/, S-1.03/, S-3.04/, S-3.01a/)
