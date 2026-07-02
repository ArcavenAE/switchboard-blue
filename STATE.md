---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-6-tranche-b-closed
phase_3_active_wave: 6
phase_3_active_stories: []
phase_3_completed_stories: [S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03, S-W3.04, S-W3.05, S-4.01, S-4.02, S-4.03, S-4.04, S-6.01, S-5.03, S-6.03, S-W5.01, S-5.01, S-6.02, S-6.06, S-5.02, S-W5.02, S-BL.LOOKUP, S-W5.04, S-6.07, S-7.01, S-7.02, S-BL.ROUTER-ADDR]
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
l3_bc_count: 45
l3_cap_coverage: "30/30"
l4_complete: true
l4_vp_count: 67
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
wave_2_gate_closed_at: 2026-06-25
wave_2_gate_disposition: "PASS_WITH_OBSERVATIONS"
wave_3_stories_merged: 9
wave_3_points_complete: 48
wave_3_points_remaining: 0
wave_3_fix_prs: "I-1 PR#18/e9421d8, T2 PR#19/849bd86, C-1 PR#20/418de54 — all merged"
internal_packages: 18
plugin_version_adopted: "1.0.0-rc.21"
wave_3_gate_closed_at: 2026-06-27
wave_3_gate_disposition: "APPROVED — 3/3 adversary clean; 5 deferrals + process-gap #7 carried to Wave 4"
wave_3_stories_detail: "closed — see cycles/cycle-1/closed-stories.md + burst-log.md"
wave_4_gate: APPROVED
wave_4_gate_closed_at: 2026-06-28
wave_4_adversary_converged: true
wave_4_adversary_passes: 6
wave_4_adversary_streak: "6/6 C=0/H=0/M=0 (2 rounds x 3 lenses)"
wave_4_wavegate_consistency_audit: "CONDITIONAL PASS — 14 findings, all resolved in cycle-close burst; 0 CRITICAL"
wave_4_integration_gate: PASSED
wave_4_integration_gate_date: 2026-06-28
wave_4_integration_evidence: "build clean; race 13/13 ok; lint 0 issues @ abeba27"
wave_5_gate: CONVERGED
wave_5_gate_closed_at: 2026-06-30
wave_5_gate_disposition: converged-clean
wave_5_convergence_passes: 6
wave_5_final_trajectory: "8 BLOCK → 2 BLOCK → 2 BLOCK → 3 CLEAN → 3 CLEAN → 2 CLEAN"
wave_6_scope_decision: 2026-06-30
wave_6_stories: 7
wave_6_points: 33
wave_6_deferred: "S-7.04 → Wave 7"
wave_6_tranche_a: "[S-W5.04 PR#40/eac5d0a, S-BL.LOOKUP PR#41/851e164, S-6.07(v1.13) PR#42/446efce — all merged 2026-07-01]"
wave_6_tranche_a_closed_at: 2026-07-01T19:04:40Z
wave_6_tranche_b: "[S-7.01 MERGED PR#43/5c658e7; S-7.02 MERGED PR#55/c54a8ad; S-BL.ROUTER-ADDR MERGED PR#56/91d5675 — Tranche B CLOSED]"
wave_6_tranche_b_closed_at: 2026-07-01
wave_6_tranche_b_pass6_fix: "b3c93b5 — F-P6L2-01 stale RED-GATE recover-guard removed from integration_test.go Part B"
wave_6_tranche_b_p7_findings: "F-P7L2-MED-01 tautological HMAC-first oracle; F-P7L2-MED-02 TruncatesOversize maximality; F-P7L2-MED-03 mid-rune exact-content"
develop_head: 91d5675
open_prs: 0
alpha_release_tag: alpha-20260629-165045-d854978
timestamp: 2026-07-01T00:00:00Z
last_update: 2026-07-01
---

# Switchboard Factory State

## Current State

Wave 6 Tranche B CLOSED. All 3 Tranche B stories merged (BC-5.39.001 3/3 each). S-7.01 PR#43/5c658e7, S-7.02 PR#55/c54a8ad, S-BL.ROUTER-ADDR PR#56/91d5675. develop HEAD: 91d5675. 45 BCs, 76 VPs, 48 stories.

**Tranche B CLOSED:** S-7.01 MERGED (5c658e7 PR#43). S-7.02 MERGED (c54a8ad PR#55; Pass-10 CLEAN 3/3). S-BL.ROUTER-ADDR MERGED (91d5675 PR#56; Pass-10 CLEAN 3/3; gh pr update-branch used for base catch-up; force-push introspection filed as vsdd-factory#408 + switchboard-blue#57).

Historical burst detail: `cycles/cycle-1/burst-log.md`.

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 4: APPROVED. Wave 5: ALL 8 MERGED. W6-TrA: ALL 3 MERGED. W6-TrB CLOSED: S-7.01 MERGED PR#43/5c658e7; S-7.02 MERGED PR#55/c54a8ad; S-BL.ROUTER-ADDR MERGED PR#56/91d5675. | 2026-07-01 | W6-TrB CLOSED: all 3 merged (BC-5.39.001 3/3 each). develop HEAD 91d5675. |

## Wave / Story Status

Waves 1–3 complete (11 stories + 3 fix PRs, PRs #1–#20). Detail: `cycles/cycle-1/closed-stories.md`.

Wave 4 complete: S-4.01 #24/e415d31, S-4.02 #25/95729c7, S-4.03 #26/8d9744f, S-4.04 #27/42c51e2, S-6.01 #28/abeba27, hygiene #29/7ef43b8.

Wave 5 complete: S-5.03 #30/01ae50c, S-5.01 #35/c1c2c3d, S-5.02 #37/98eb8b7, S-6.02 #34/b36cb9b, S-6.03 #32/d854978, S-W5.01 #31/0d499ac, S-6.06 #36/3ee9c38, S-W5.02 #38/d881f99. (S-W5.04 moved to Wave 6 per F-W5P1-004.)

| Wave | Story | Title | Status | PR | SHA |
|------|-------|-------|--------|----|-----|
| 6 | S-BL.LOOKUP | Migrate AdmittedKeySet.Lookup to value-return form | MERGED | #40 | eac5d0a |
| 6 | S-W5.04 | daemon-side paths.list / router.metrics / router.status RPC handlers | MERGED | #41 | 851e164 |
| 6 | S-6.07 | Register admin.svtn.create handler + sbctl admin svtn create CLI (v1.13) | MERGED | #42 | 446efce |
| 6 | S-7.01 | XOR parity FEC for single-loss recovery | MERGED | #43 | 5c658e7 |
| 6 | S-7.02 | SVTN-scoped multicast session discovery | MERGED | #55 | c54a8ad |
| 6 | S-BL.ROUTER-ADDR | populate PathSnapshot.RouterAddr (BC-2.06.003 PC-1) | MERGED | #56 | 91d5675 |

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| W3-R2-M2 | MED | Route-time LWW snapshot: concurrent RegisterForwardingEntry not atomic with HMAC verify. | architect/implementer | open |
| SW305-M4 | MED | W4-TEST-001: RouteFrame fire-once E-ADM-017 integration test (real FailureCounter + WithNow). | test-writer | DEFER-WAVE-4 |
| process-gap-follow-up | OBS | Adversary nil-safety lens gap (missed SEC-001); lesson in lessons.md; candidate self-improvement story. | orchestrator | open/deferred |
| W3-DEFER-1 | OBS | Codify worktree-identity tuple in adversary dispatch templates. | orchestrator | deferred |
| W3-DEFER-2 | MED | M-1 relay busy-spin: double-failure-no-PTY not integration-tested. | implementer | deferred S-BL.NI |
| W3-DEFER-3 | MED | Fired-source LRU eviction-priority inversion (WithFailureCounter insertion-order, not fired-first). | implementer | deferred |
| W3-DEFER-4 | MED | M-2 unbounded E-ADM-016 log volume under sustained attack (BC-2.05.005 gap). | product-owner | deferred |
| W3-DEFER-5 | MED | EC-005: no CI lint rule enforces internal/ import boundary structurally. | devops-engineer | deferred |
| W3-DEFER-6 | MED | Real-connector PTY-EOF lifecycle integration test (mock-only today). | test-writer | deferred |
| S402-F007 | LOW | S-4.02: ARCH-03 line 122 N=3 vs BC-2.02.004 N=5 — reconcile ARCH-03 (BC is authority). | architect | open |
| S403-O4 | LOW | S-4.03: DegradationEvent single-seq vs BC-2.02.006 PC2 range — per-frame drop OK for MVP. | product-owner | deferred MVP |
| S403-H1-DEFER | MED | BC-2.02.005 PC-3 retransmit-SEND now anchored to S-BL.ARQ-TX (depends S-4.03). | product-owner/architect | anchored to S-BL.ARQ-TX (was orphaned) |
| DRIFT-S4.03-001 | MED | ADR-005 resync-on-reconnect wire-mechanics deferred; owner updated to S-BL.NI (backlog) per ADR-005/ARCH-03 v1.4. | architect/implementer | deferred S-BL.NI |
| S404-OBS-F | OBS | S-4.04 E-FWD-001 emission is per-event/not-rate-limited; LATENT CWE-779 only if production caller makes eligible-interface set attacker-steerable. | architect/product-owner | re-confirm when production caller lands |
| S404-LOW-1 | LOW | S-4.04: 3 LOW + NITPICK findings from adversary final pass (SEC-001 CRC32 collision accepted per BC-2.02.009 EC-004). | implementer | cycle-close follow-up |
| S601-SEC-001 | LOW | S-6.01: CWE-117 — sanitize operator-supplied --config PATH arg at 3 LoadFile error sites. | implementer | deferred cycle-close |
| S601-SEC-002 | LOW | S-6.01: CWE-400 — explicit length cap on upstream_routers slice; implicitly bounded by 1 MiB file guard. | product-owner/architect | deferred cycle-close |
| OBS-VP-BENCH | OBS | VP-041/VP-042 unverified pending S-BL.BENCH integration-benchmark story (not yet created). | orchestrator | deferred S-BL.BENCH |
| PROCESS-GAP-W4 | OBS | [process-gap] S-BL.NI network-ingress wave must carry an explicit cross-component lock-ordering review axis + integration -race test driving a frame through routing→arq→replay→multipath concurrently. Per-package -race suite cannot catch future cross-package lock-order inversion. | orchestrator/architect | target S-BL.NI wave planning |
| F-009 | LOW | ARCH-INDEX input-hash tooling field-name mismatch (pre-existing, hash tooling does not emit `input_hash` field). | architect/devops | tracked TODO — deferred maintenance |
| E-CFG-002 | MED | Pre-existing config-key collision (joins tracked E-CFG-006). | product-owner | deferred maintenance |
| E-CFG-006 | MED | Pre-existing config-key collision (tracked from prior audit). | product-owner | deferred maintenance |
| PROCESS-GAP-W5A | OBS | [process-gap] Two false-greens caught in Wave 5 (S-W5.01: 3/4 daemon modes still had orphaned listeners; S-6.03: `go test -race` intermittently failed on homeDirFunc race). Candidate discipline: require `just test-race` evidence-paste before green-claim. | orchestrator | open — candidate codification |
| DRIFT-SW501-NITPICK | LOW | S-W5.01 Pass-3 nitpicks (non-gating, cosmetic): stale "Stub: ... Red Gate" comments in internal/config/config.go ~L236 & ~L244 (functions fully implemented+tested); dead `_ = pub` in internal/mgmt/mgmt.go ~L462. | implementer | cannot-action-without-owner (source-code edit; spec-steward scope is .factory/ only; needs implementer in Wave-6 hygiene story) |
| PROCESS-GAP-P21..P25 | OBS | [process-gap] 7 consecutive recurrences of sibling-sweep gap (BC/VP narrowing, story-body prose, downstream-doc version cites, upstream-rooted sweep). Rule crystallized: any artifact version bump must sweep all downstream cites atomically. vsdd-factory #361–#364 filed; #361 carries all 7 recurrence comments. | orchestrator/story-writer | open — vsdd-factory issues filed |
| S502-DEFER-1 | MED | S-5.02: runRouterStatus at cmd/sbctl/router_status.go:164-167 lacks auth-timeout wrap (BC-2.06.003 PC-3 / BC-2.07.003 Inv-2 alias-parity gap). | implementer | defer wave-gate |
| S502-DEFER-2 | MED | S-5.02: writeSuccess at cmd/sbctl/main.go:101 calls os.Exit(3) outside main() — violates go.md rule. | implementer | defer phase-5 |
| S502-DEFER-4..6 | LOW | S-5.02: ARCH-11/dep-graph VP total stale (75/67 vs 76); S-W5.04 §Arch Compliance asymmetric (VP-047 only); token-budget footnote cosmetic. | architect/story-writer | defer post-convergence arch-doc sweep |
| SW502-DEFER-1..8 | LOW | S-W5.02 CR-002/005/006/007/008/009 + SEC-001/002: 8 LOW deferrals (intentional design choices, test-only observations, cosmetic). Detail: `cycles/cycle-1/closed-drift.md`. | implementer/test-writer | deferred wave-6 / phase-5-hardening |
| PROCESS-GAP-W5-SIBLINGSWEEP | LOW | [process-gap] Codify orchestrator-level upstream-rooted sibling-sweep enforcement at BC/VP version bumps (superset of PROCESS-GAP-P19..25); currently only external vsdd-factory issue #361 comment. | orchestrator | orchestrator-policy-registry-update |
| PROCESS-GAP-STORY-INDEX-SUMMARY-SWEEP | OBS | [process-gap] STORY-INDEX multi-location aggregate rollups (Summary Total, stubs rollup, section counts) must be swept atomically when a story row moves sections. F-P2L3-M1 exposed this at v3.24. | orchestrator/story-writer | open — process rule to codify |
Resolved items (C-1/OBS-3, T2, SW305-M1..M8, HF3, S402-F006, S403-O1, Phase-6 deferrals, BC-2.09.003-STALE, S601-NITPICK-A..E, S601-DRAFT-STORY, S403-COS1/2, S404-OBS-G, S401-O3, W5-gate-H1..H3/M1..M4, S502-DEFER-3, DRIFT-SW504-ROUTER_ADDR-PLACEHOLDER): `cycles/cycle-1/closed-drift.md`

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| HMAC/FEC/LWW/key-permissions/HKDF architecture | ADR-001..004; ARCH-02/03/04 | 2026-06-23 |
| Wave 3 gate APPROVED | 3/3 adversary clean; 5 deferrals + process-gap #7 carried to Wave 4 | 2026-06-27 |
| Wave 4 gate APPROVED | 6/6 diverse-lens passes; consistency audit CONDITIONAL PASS (14 findings resolved) | 2026-06-28 |
| Wave 5 complete (8 stories + hygiene) | All PRs #30-#38 merged; BC-2.07.004 + VP-064..VP-067 minted; 45 BCs, 76 VPs | 2026-06-30 |
| Wave-6 scope decided | 7 stories, 33 pt; Tranche A: S-W5.04 ∥ S-BL.LOOKUP ∥ S-6.07; Tranche B: S-7.01/S-7.02/S-7.03 | 2026-06-30 |
| S-BL.LOOKUP MERGED (eac5d0a, PR #40) | AdmittedKeySet.Lookup value-return migration; Wave-6 Tranche A | 2026-07-01 |
| S-W5.04 MERGED (851e164, PR #41) | daemon-side paths.list/router.metrics/router.status RPC handlers; BC-2.06.003 PC-1/2; VP-047 | 2026-07-01 |
| S-6.07 MERGED (446efce, PR #42) | admin.svtn.create handler + sbctl CLI; BC-2.07.001; Wave-6 Tranche A CLOSED | 2026-07-01 |
| S-7.01 MERGED (5c658e7, PR #43) | XOR parity FEC; BC-2.02.007; first Tranche B story to converge under BC-5.39.001 | 2026-07-02 |
| S-7.02 MERGED (c54a8ad, PR #55) | SVTN-scoped multicast session discovery; BC-2.03.001/002/003; VP-044/045/055; Pass-10 3/3 CLEAN | 2026-07-01 |
| S-BL.ROUTER-ADDR MERGED (91d5675, PR #56) | PathSnapshot.RouterAddr BC-2.06.003 PC-1; VP-047; Pass-10 3/3 CLEAN; Wave-6 Tranche B CLOSED | 2026-07-01 |
Older decisions (Waves 1-5 per-story): `cycles/cycle-1/burst-log.md`.

## Session Resume Checkpoint — 2026-07-01 (Wave-6 Tranche B CLOSED)

**Position:** Phase 3 Wave 6 Tranche B CLOSED. All 3 stories merged: S-7.01 PR#43/5c658e7, S-7.02 PR#55/c54a8ad, S-BL.ROUTER-ADDR PR#56/91d5675. develop HEAD: 91d5675. Worktree + branch for S-BL.ROUTER-ADDR removed.

**BC-5.39.001 status:** All 3 Tranche B stories — SATISFIED (3/3 clean diverse-lens passes each).

**Force-push introspection:** During S-BL.ROUTER-ADDR PR #56 base catch-up, pr-manager reached for rebase+force-push when `gh pr update-branch` was the correct non-destructive tool. Auto-mode classifier blocked the force-push. `gh pr update-branch` used successfully on second attempt. Issues filed: drbothen/vsdd-factory#408 (pr-manager playbook) + ArcavenAE/switchboard-blue#57 (merge-serialization hazard under "require branches up to date").

**Follow-up issues filed this cycle:** switchboard-blue #44-54, #57; drbothen/vsdd-factory #407, #408.

**NEXT ACTION on resume:** Wave-6 Tranche B wave-level adversarial convergence (BC-5.39.001 wave-level, 3 clean fresh-context passes required against all 3 merged stories), OR proceed to Tranche C planning (S-6.05, S-7.03).

**Open deferred observations (carry forward):**
- S502-DEFER-1..6 / SW502-DEFER-1..8: S-5.02 + S-W5.02 LOW deferrals in Open Drift Items.
- PROCESS-GAP-W5-SIBLINGSWEEP: vsdd-factory #361-364.
- PROCESS-GAP-STORY-INDEX-SUMMARY-SWEEP: open/codify.
- TaskList #115: S-6.06 lens-1 post-merge polish.
- S-7.01 CR-001/004/005/006/007: follow-up issues filed post-merge.

Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.

## Session Log

| Date | Entry |
|------|-------|
| 2026-07-02 | **Pass-8/9:** S-7.01 MERGED PR#43/5c658e7. S-7.02 CLEAN 2/3 (HEAD a9bf936). S-BL.ROUTER-ADDR CLEAN 2/3 (HEAD dffc27e). Pass-10 dispatched. |
| 2026-07-01 | **Pass-10 + Tranche B CLOSE:** S-7.02 MERGED PR#55/c54a8ad (3/3). S-BL.ROUTER-ADDR MERGED PR#56/91d5675 (3/3; gh pr update-branch; vsdd-factory#408+#57 filed). Cleanup done. |

## Historical Content

Burst logs, adversary passes, session checkpoints, closed-stories, closed-drift: `cycles/cycle-1/`
