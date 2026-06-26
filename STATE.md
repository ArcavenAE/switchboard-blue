---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-2-integration-gate-closed
phase_3_active_wave: 2
phase_3_active_stories: []
phase_3_completed_stories: [S-1.01, S-1.02, S-2.01, S-2.02, S-1.03]
phase_3_pause_point: "Wave 2 integration gate CLOSED (PASS_WITH_OBSERVATIONS 2026-06-25). Wave 3 planning pending explicit launch."
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
s_1_03_adversary_step_4_5: "CONVERGED (3/3 clean: passes 3, 4, 5) — BC-5.39.001 satisfied"
s_1_03_adversary_pass_03_sha: dc37fe1
s_1_03_adversary_pass_04_sha: 52ee1d3
s_1_03_adversary_pass_05_sha: 6bcde7d
s_1_03_merge_sha: f35e836
s_3_04_adversary_step_4_5: "CONVERGED (3/3 clean: passes 3, 4, 5) — BC-5.39.001 satisfied"
s_3_04_adversary_pass_03_sha: pending
s_3_04_adversary_pass_04_sha: pending
s_3_04_adversary_pass_05_sha: 5c3f93a
s_1_03_pr_number: 7
s_1_03_merge_date: 2026-06-25
s_1_03_status: completed
wave_2_complete: true
wave_2_stories_merged: 3
wave_2_points: 18
timestamp: 2026-06-26T00:00:00Z
last_update: 2026-06-26
---

# Switchboard Factory State

## Current State

Phase 3 TDD Implementation, Wave 2 integration gate CLOSED — PASS_WITH_OBSERVATIONS (2026-06-25).
Consistency-validator: 0C/0H/2M/3L/4O. Fresh-context cross-story audit: 0C/0H/1M/3L/3O.
Governance burst fully landed: ARCH-08 v1.0→v1.1 (halfchannel + §6 Import Constraints, `1d09664`);
demo-evidence backfill S-2.02/S-1.03 (`c4ee7db`); E-FWD-002 minted + PR #8 godoc cite merged at
`d8d7ae6` (`c4ee7db`); VP lifecycle policy `_LIFECYCLE.md` v1.0 + VP-007/008/009/010/057
→ implemented, VP-039 → deferred (`918acb4`); WAVE-2-MED-001 + WAVE-3-DEP-001 drift rows
(`cdac793`). Develop tip post-cleanup: `d8d7ae6`. Wave 3 planning pauses pending explicit launch.

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 2 gate: PASS_WITH_OBSERVATIONS | 2026-06-25 | Wave 2: 3/3 done; gate CLOSED; Wave 3 pending launch |

## Wave / Story Status

| Wave | Story | Title | Status | PR | SHA |
|------|-------|-------|--------|----|-----|
| 1 | S-1.01 | Frame codec | completed | #1 | 1c76160 |
| 1 | S-1.02 | Half-channel clock | completed | #2 | 9e9a98a |
| 1 | refactor | FrameType + MTU | completed | #3 | 4be1b53 |
| 2 | S-2.01 | HMAC codec | completed | #5 | 3c4104e |
| 2 | S-2.02 | Admission + SVTN isolation | completed | #6 | a06b306 |
| 2 | S-1.03 | Session continuity | completed | #7 | f35e836 |

Wave 2: 3/3 stories merged (18 pts). Gate: PASS_WITH_OBSERVATIONS — CLOSED 2026-06-25.
Governance burst archived to `cycles/cycle-1/burst-log.md` (Wave 2 Governance Burst entry).
Gate reports: `cycles/cycle-1/wave-2/`.

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| F-P8-004 | MED | VP-026 cites "transitivity" invariant missing from BC-2.02.003 | architect | open — Phase 3 test-writing for BC-2.02.003 |
| F-P8-005 | MED | VP-027 title "degradation goes down" but harness tests recovery direction | architect | open — Phase 3 test-writing |
| F-P8-009 | LOW | feasibility-report:61 deployment-ops range "(CAP-026–027)" should be "(CAP-026–028)" | architect | open — Phase 2 deferred |
| F-003 | LOW | Payload-MTU composed wire-format test | story S-BL.OA | deferred to outer-assembler story |
| F-004 | LOW | ARCH-02 channel-header serializer not implemented | story S-BL.OA | deferred to outer-assembler story |
| VP-036 testenv | Phase-6 hardening | S-1.03 unit tests cover AC-001..003; property test (TestProperty_VP036_SessionContinuity) deferred until internal/testenv.ConnectWithSourceIP exists | 2026-06-25 |
| SEC-003 | Phase-6 hardening | Sub-microsecond TOCTOU on now timestamp in ReAuthenticate; worst case one re-auth on just-expired key. Accepted disposition per pr-reviewer security review of PR #7 | 2026-06-25 |
| WAVE-2-MED-001 | Phase-6 hardening | ReAuthState not evicted on RevokeKey or RegisterKey reset; stale source-IP survives via CurrentSourceAddr; gated by IsAdmitted in RouteFrame but no cross-check in the accessor itself | 2026-06-25 |
| WAVE-3-DEP-001 | Wave 3 (HMAC wire-up) | verifyFrameHMAC is //nolint:unused on develop; Wave-2 router has zero frame-forgery defense until wired into RouteFrame; test scaffolding ready (S-2.02 pass-4 fix) | 2026-06-25 |
| VP-039-test-skip | Phase-6 hardening | t.Skip placeholder needed in internal/routing/*_test.go for VP-039 (deferred property test); spec-steward flagged during Wave-2 governance burst | 2026-06-25 |

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

## Session Resume Checkpoint — 2026-06-26

**Position:** Phase 3, Wave 3 in progress — S-3.04 Step 4.5 COMPLETE (adversary CONVERGED).
S-3.04 (HMAC RouteFrame wire-up): adversary passes 3/4/5 all clean (0 findings each).
BC-5.39.001 satisfied. Develop tip: `d8d7ae6`. S-3.04 branch tip: `e214f8d`.
factory-artifacts adversary pass-05 SHA: `5c3f93a`.

**S-3.04 adversary streak (Step 4.5):** Pass 1: 1M/1L, Pass 2: 1M/1L, Pass 3: 0, Pass 4: 0, Pass 5: 0 — CONVERGED.
Zero process-gap findings across passes 3/4/5.

**Carry-forward drift (not blockers):**
- WAVE-2-MED-001 (Phase-6): ReAuthState eviction on RevokeKey/RegisterKey — Phase-6 hardening target.
- WAVE-3-DEP-001 (Wave 3): resolved by S-3.04 implementation — pending PR merge.
- VP-036 (Phase-6): property test deferred to Phase-6 (needs `internal/testenv.ConnectWithSourceIP`).
- VP-039-test-skip (Phase-6): t.Skip placeholder needed in `internal/routing/*_test.go`.
- SEC-003 (Phase-6, ACCEPTED): sub-microsecond TOCTOU on `now` in ReAuthenticate.

**Immediate next action:**

S-3.04 Step 5 — per-AC demo recordings (demo-recorder agent, AC-001..AC-005 + EC-001..EC-005).

**KoS frontier open questions** (for future phases):
- Router-to-router PE phase Noise XX mutual auth?
- SACK bitmap window configurable (64-bit may be too narrow for PE high-latency)?
- Goroutine model for 1k concurrent sessions — per-session pair vs event-loop (NFR-004)?
- Drop cache — TTL eviction in addition to LRU?
- PE router-to-router Noise — share node admission keypair, or separate router identity?

## Historical Content

Burst logs, adversary pass details, session checkpoints, and closed-story narratives have been extracted to cycle files:

- Burst history: `cycles/cycle-1/burst-log.md`
- Convergence trajectory (all passes): `cycles/cycle-1/convergence-trajectory.md`
- Session checkpoints (archived): `cycles/cycle-1/session-checkpoints.md`
- Closed story summaries (S-1.02, S-2.01, S-2.02, S-1.03, Wave-1, Wave-2): `cycles/cycle-1/closed-stories.md`
- Wave gate reports: `cycles/cycle-1/wave-1/`, `cycles/cycle-1/wave-2/`
- Per-story adversary reports: `cycles/cycle-1/S-1.02/adversary/`, `cycles/cycle-1/S-2.01/adversary/`, `cycles/cycle-1/S-2.02/adversary/`, `cycles/cycle-1/S-1.03/`
