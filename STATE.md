---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-4-active
phase_3_active_wave: 4
phase_3_active_stories: []
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
wave_3_gate_adversary_streak: 3
wave_3_gate_adversary_converged: true
wave_3_gate_pass1_disposition: "C-1 deferred (C-1-W3P1-defer/S-BL.NI, ARCH-08 v2.2 §6.5.1); I-1 fixed PR #18 e9421d8"
wave_3_gate_pass2_disposition: "CONVERGED 0C/0H — contract-conformance"
wave_3_gate_pass3_disposition: "CONVERGED 0C/0H — security"
wave_3_gate_convergence_summary: "3/3 CLEAN passes (Pass-1 concurrency/lifecycle, Pass-2 contract-conformance, Pass-3 security); consistency-audit HIGH Finding-4.1 downgraded to traceability-only (T2 satisfied in code: TestForwardFramesTOCTOUCount50 + deterministic swapBarrier test)"
wave_3_gate_human_gate: APPROVED
wave_3_gate_closed_at: 2026-06-27
wave_3_gate_disposition: "APPROVED — close Wave 3; carry 5 tracked deferrals + process-gap #7 to Wave 4"
w3_c1_pr: 20
w3_c1_merge_sha: 418de54
w3_c1_disposition: "RESOLVED — WithFailureCounter wired buildRouter (threshold=5/window=60s); OBS-3 closed; network-ingress listener deferred S-BL.NI"
w3_t2_pr: 19
w3_t2_merge_sha: 849bd86
w3_t2_disposition: "DELIVERED — deterministic TOCTOU misclassification-branch test (ADR-011 v1.6 Obligation T2)"
wave_3_pre_gate_items: "COMPLETE — C-1 (PR #20/418de54) + T2 (PR #19/849bd86) both merged"
w3_i1_fix_pr: 18
w3_i1_fix_merge_sha: e9421d8
w3_i1_fix_merge_date: 2026-06-27
s_w3_05_per_story_adversary_streak: 3
wave_3_gate_adversary_passes: "RESTART run at 10dd880: r1 CONVERGED 0C/0H; r2 CONVERGED 0C/0H/4M; r3 NOT_CONVERGED 0C/2H (F-1 cmd-wiring, F-2 EC-006 — both scope-boundary; adjudicated 2026-06-27). W3-R3-F1 RESOLVED (all 6 ARCH-08 §6.5.1 wiring obligations met; stale SHA). W3-R3-F2 RATIFIED (BC-2.05.008 v1.3 + VP-059 v1.2 cover EC-006). SW305-M4 deferred W4-TEST-001."
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
s_4_03_adversary_streak: 0
develop_head: 36c5e98
open_prs: 0
timestamp: 2026-06-28T18:00:00Z
last_update: 2026-06-28
---

# Switchboard Factory State

## Current State

Wave 4 ACTIVE. S-4.01 (first Wave-4 delivery) COMPLETE — merged as e415d31 (PR #24, 7/7 ACs, 3/3 adversary clean). kos-scaffolding cleanup PR #23 also merged (36c5e98). develop HEAD = 36c5e98. 0 open PRs. Sub-wave 4A remaining: S-4.02, S-4.03, S-6.01 (not yet started). Sub-wave 4B: S-4.04 now UNBLOCKED (internal/paths on develop). W4-TEST-001 deferred (test-writer).

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 3: CLOSED; Wave 4: ACTIVE (S-4.01 MERGED; S-4.02/03/06.01 pending) | 2026-06-28 | Wave 3: 3/3 CLEAN; Wave 4: S-4.01 adversary 3/3 CLEAN @ aaff609; MERGED e415d31 |

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
| 3 | fix/W3-t2 | Deterministic TOCTOU test (ADR-011 T2) | completed | #19 | 849bd86 |
| 3 | fix/W3-c1 | WithFailureCounter wiring (OBS-3 resolved) | completed | #20 | 418de54 |
| 4 | S-4.01 | Per-path RTT/loss tracking + dedup/race dispatch | completed | #24 | e415d31 |

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| W3-R2-M2 | MED | Route-time LWW snapshot: concurrent RegisterForwardingEntry not atomic with HMAC verify. | architect/implementer | open |
| SW305-M4 | MED | Test coverage gap: RouteFrame fire-once E-ADM-017 integration test with real *admission.FailureCounter + WithNow injection. | test-writer | DEFER-WAVE-4 (W4-TEST-001) |
| process-gap-follow-up | OBS | Adversary nil-safety lens gap (missed SEC-001) — lesson recorded in cycles/cycle-1/lessons.md. Follow-up: candidate for self-improvement epic story. | orchestrator | open/deferred |
| W3-DEFER-1 | OBS | Process-gap: codify worktree-identity tuple in adversary dispatch templates (applied ad hoc this cycle). | orchestrator | deferred → Wave 4 process hardening |
| W3-DEFER-2 | MED | M-1 relay busy-spin: double-failure-no-PTY scenario — watchAndFallback exhausted + no PTY path; daemon behavior under total connector failure not integration-tested. | implementer | deferred → Wave 4 / S-BL.NI |
| W3-DEFER-3 | MED | Fired-source LRU eviction-priority inversion — WithFailureCounter evicts sources in insertion order, not by fired-first priority; live-traffic bias possible. | implementer | deferred → Wave 4 |
| W3-DEFER-4 | MED | M-2 unbounded E-ADM-016 log volume: per-source slot cap exists but per-burst log cardinality under sustained attack not bounded by spec (BC-2.05.005 gap). | product-owner/spec-steward | deferred → Wave 4 |
| W3-DEFER-5 | MED | EC-005: durable CI import-perimeter guard — BC-2.04.002 EC-005 comment corrected (PR #17), but no CI lint rule enforces the `internal/` import boundary structurally. | devops-engineer | deferred → Wave 4 |
| W3-DEFER-6 | MED | Real-connector PTY-EOF lifecycle integration test — TestForwardFramesPTYEOFExitsCleanly uses mock; no test exercises actual PTY master EOF through live SessionConnector. | test-writer | deferred → Wave 4 |
| S401-O3 | MED | BC-2.02.003 PC5: degraded-path flag (RTT >200ms) unimplemented in internal/paths — feeds quality-indicator subsystem (BC-2.06.001/ARCH-03), out of S-4.01 scope. Ranking already deprioritizes slow paths via score; flag wiring tracked for quality-indicator story. | product-owner/architect | deferred → quality-indicator story |
| S402-F006 | LOW | S-4.02: uint32 nextSeq wraparound in replay — document lifetime assumption. | architect | deferred |
| S402-F007 | LOW | S-4.02: ARCH-03 line 122 N=3 vs BC-2.02.004 N=5 default — reconcile ARCH-03 to N=5 (BC is behavior authority). | architect | open |
| S403-O1 | LOW | S-4.03: uint32 wraparound in arq — document lifetime assumption. | architect | deferred |
| S403-O4 | LOW | S-4.03: DegradationEvent single-seq vs BC-2.02.006 PC2 range — per-frame drop acceptable for MVP; product-owner note. | product-owner | deferred MVP |
| S403-H1-DEFER | MED | S-4.03: retransmit-SEND PC3 deferred to router/multipath wiring story (precedent: S-4.01/BC-2.02.009 router wiring deferral). | product-owner/architect | deferred |
Resolved items (C-1/OBS-3, T2, SW305-M1..M8, HF3, Phase-6 deferrals, wave-gate rows): `cycles/cycle-1/closed-drift.md`

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| HMAC algorithm | HMAC-SHA256, 16-byte tag, HKDF-SHA256 per-SVTN (ADR-001, ARCH-02/04) | 2026-06-23 |
| FEC group size | N=4 default; tunable (ADR-002, ARCH-03) | 2026-06-23 |
| Duplicate key registration | last-write-wins (ADR-003, ARCH-04) | 2026-06-23 |
| Console/access key permissions | control > console > access (ADR-004, ARCH-04) | 2026-06-23 |
| HMAC keying | per-(node, svtn) HKDF using node_admission_pubkey as IKM (ADR-001 amended) | 2026-06-23 |
| Marvel integration | explicitly deferred — no MVP integration | 2026-06-24 |
| S-3.03 repointed 5→8 | upstream-wiring scope expansion; Wave 3 total 29→32 pts | 2026-06-27 |
| S-W3.05 E-ADM-017 msg-format adjudication CORRECTED | specs authoritative — include "HMAC failure rate alert:" phrase; code/tests/story AC-003/AC-015 conform | 2026-06-27 |
| S-W3.05 re-arm semantics finalized | drain-only re-arm + per-source append-skip; reconciled BC-2.05.005 v1.6/VP-059 v1.1 | 2026-06-27 |
| S-W3.05 CONVERGED + SEC-001 fixed + PR #16 merged | 3 clean passes (10-12) at f6038d2; fa6345e | 2026-06-27 |
| S-W3.04 CONVERGED (BC-5.39.001) + PR #17 merged | 3 clean passes (10-12) at 1c3c864; aeb442d | 2026-06-27 |
| Per-story-delivery merge-handoff pathology (vsdd-factory#302) | Agent self-merge blocked by classifier; human-performed merge is the correct resolution | 2026-06-27 |
| Wave-3 Pass-1: C-1 deferred, I-1 fixed PR #18 e9421d8 | C-1 → ARCH-08 v2.2 §6.5.1 TRACKED-DEFER/S-BL.NI; I-1 (BC-2.04.007) fixed; streak 0/3 | 2026-06-27 |
| Wave-3 pre-gate consistency audit | PASS — 0 blocking; 3 non-blocking findings resolved: D5-1 (STORY-INDEX S-BL.NI row rewritten), T2-1 + V-1 (S-W3.04 AC-010 + ARCH-08 pin updated to v2.3) | 2026-06-27 |
| Wave 3 integration gate | APPROVED — close Wave 3; carry 5 tracked deferrals + process-gap #7 to Wave 4 | 2026-06-27 |
| W3-R3-F1 cmd-wiring adjudication | RESOLVED — all 6 ARCH-08 §6.5.1 wiring obligations met in real binary; adversary saw stale SHA; no Wave-4 work | 2026-06-27 |
| W3-R3-F2 EC-006 adjudication | RATIFY — BC-2.05.008 v1.3 / VP-059 v1.2 already specify implemented semantics; SW305-M2/M3 closed; SW305-M4 deferred to W4-TEST-001 | 2026-06-27 |
| S-4.01 per-story adversarial CONVERGED (BC-5.39.001) | 3 clean passes (3-5) at aaff609; O-2 fixed 327f5c6; O-1/O-3/O-4 logged; BC-2.02.009 router wiring deferred to S-4.04 | 2026-06-28 |
| S-4.01 PR #24 merge-ready | All 7 ACs PASS; adversary CONVERGED 0C/0H 3/3; security no Crit/High; pr-reviewer APPROVE 0 blocking; CI green; stopped at human-merge gate | 2026-06-28 |
| S-4.01 MERGED + PR #23 cleanup MERGED | S-4.01 squash e415d31 (7/7 ACs, 3/3 adversary clean, merged via explicit user consent — classifier guardrail, not a project gate); kos-scaffolding cleanup #23 squash 36c5e98; develop HEAD 36c5e98; 0 open PRs | 2026-06-28 |
| S-4.02 + S-4.03 adversary Pass-1 NOT_CONVERGED | S-4.02 1C/1H (F-001 unbounded pending, F-002 no-op latency gate); S-4.03 1C/2H (C-1 TLPKTDROP over-advance, H-1 gap-detection split/defer-send, H-2 false concurrency claim); fixes routed; rulings recorded | 2026-06-28 |

## Session Resume Checkpoint — 2026-06-28 (Wave 4 — between stories)

**Position:** Phase 3, Wave 4 ACTIVE. S-4.01 MERGED (e415d31, PR #24). PR #23 kos-scaffolding cleanup MERGED (36c5e98). develop HEAD = 36c5e98. 0 open PRs.
**Wave 4 scope:** S-4.01 (done), S-4.02, S-4.03, S-4.04, S-6.01 (29 pts). Sub-wave 4A remaining: S-4.02, S-4.03, S-6.01 (not yet started). Sub-wave 4B: S-4.04 UNBLOCKED (internal/paths on develop).
**S-4.01:** COMPLETE — 7/7 ACs, 3/3 adversary clean @ aaff609, merged e415d31. BC-2.02.009 router wiring deferred to S-4.04.
**Next:** start S-4.02, S-4.03, S-6.01 in parallel; S-4.04 unblocked.
**Deferred task:** W4-TEST-001 (RouteFrame fire-once E-ADM-017 integration test, owner: test-writer).
**Open Drift Items:** W3-DEFER-1..6, W3-R2-M2, SW305-M4/W4-TEST-001, S401-O3 (see Drift Items table).
**S-BL.NI:** ARCH-08 v2.3 §6.5.1 network-ingress listener + E-ADM-017 live-data-path integration test — Wave 4+.
**Previous checkpoint:** `cycles/cycle-1/session-checkpoints.md`.

## Historical Content

Burst logs, adversary passes, session checkpoints, closed-stories, closed-drift: `cycles/cycle-1/`
