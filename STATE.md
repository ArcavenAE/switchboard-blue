---
document_type: pipeline-state
level: ops
version: "2.0"
status: active
producer: state-manager
project: switchboard
inputs: []
input-hash: "[live-state]"
traces_to: ""
pipeline: STEADY_STATE
phase: steady-state-post-cycle-1
phase_step: steady-state-discovery-wire-reconverge-v2.26-5c8db39
product: switchboard
mode: greenfield
current_cycle: cycle-1
anchor_strategy: reference-via-frontmatter
dtu_required: false
dtu_assessment: 2026-06-23
internal_packages: 23
plugin_version_adopted: "1.0.0-rc.22"
l2_complete: true
l3_complete: true
l3_bc_count: 45
l4_complete: true
l4_vp_count: 77
vp_proven: 68
vp_justified_deferred: 9
arch_sections: 13
arch_adrs: 12
phase_1_gate: APPROVED
phase_2_gate: APPROVED
wave_1_gate: PASS_WITH_CLEAN_DRIFT
wave_2_gate: PASS_WITH_OBSERVATIONS
wave_3_gate: APPROVED
wave_4_gate: APPROVED
wave_5_gate: CONVERGED
wave_6_gate: CONVERGED_3_OF_3
phase_4_gate: "PASS 0.895 re-eval 2026-07-12 @ f73676d (original PASS_AT_THRESHOLD 0.85 @ 7fe3e29 2026-07-02, IP-C1-04)"
phase_5_pass_4_gate: BC_5_39_001_SATISFIED
develop_head: 7fcf0cf
sprint_state_code_lane_head: cee8e8b
open_prs: 0
alpha_release_tag: alpha-20260629-165045-d854978
awaiting: "Step-4.5 reconvergence for S-BL.DISCOVERY-WIRE — Pass-2 traceability F-1 (MED) remediated at factory `43f2e47` (story v2.26 / def6b7b; declared-input BC pins re-synced BC-2.03.001 v1.7 / BC-2.01.008 v1.3); code `5c8db39` (26 commits) + ruling v1.2 unchanged; counter 0/3 RESET; fresh 3-pass NITPICK_ONLY reconvergence against 5c8db39 / v2.26 next (POL-005 HEAD-SHA tuple mandatory). (b) S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY (ready v1.0, 3 ACs, 3 pts). S-BL.LOOPBACK-FULLSTACK parked (P2, 8pts)."
current_step: "DISCOVERY-WIRE Step-4.5 Pass-2 traceability F-1 (MED) remediated at factory 43f2e47 (story v2.26 / def6b7b; input-hash UNCHANGED — prose/metadata-only; declared-input BC pins re-synced BC-2.03.001 v1.7 / BC-2.01.008 v1.3); code 5c8db39 (26 commits) + ruling v1.2 unchanged; counter 0/3 RESET. develop @ 7fcf0cf. D-chain cite D-446 latest greenfield. trajectory-tail →21→7→4→3"
historical_cycles: []
timestamp: 2026-07-21T02:00:00Z
last_update: 2026-07-20
---

<!--
  STATE.md SIZE BUDGET (per D-421(c)):
  Hard cap (500 lines) margin from soft-target = 500 - 196 = 304; margin from actual = 500 - 196 = 304 (D-446(c) dual-margin form). 196 lines (wc-l).
  Hard cap: 500 lines.
-->

| **Last Updated** | 2026-07-20 — S-BL.DISCOVERY-WIRE Step-4.5 Pass-2 traceability F-1 (MED) remediated at factory 43f2e47: story v2.26 / def6b7b (input-hash unchanged — prose/metadata-only); declared-input BC pins re-synced BC-2.03.001 v1.6→v1.7 / BC-2.01.008 v1.2→v1.3; code 5c8db39 (26 commits) + ruling v1.2 unchanged; counter 0/3 RESET; trajectory-tail →21→7→4→3 |

# Switchboard Factory State

## Project Metadata

| Field | Value |
|-------|-------|
| **Product** | switchboard |
| **Repository** | ArcavenAE/switchboard-blue |
| **Mode** | greenfield |
| **Language** | Go |
| **Target Workspace** | run/switchboard-blue |
| **Started** | 2026-06-23 |
| **Last Updated** | 2026-07-20 |
| **Current Phase** | steady-state (post-cycle-1) |
| **Current Step** | DISCOVERY-WIRE Step-4.5 Pass-2 traceability F-1 (MED) remediated at factory 43f2e47 (story v2.26 / def6b7b; input-hash unchanged — prose/metadata-only); declared-input BC pins re-synced BC-2.03.001 v1.7 / BC-2.01.008 v1.3; code 5c8db39 (26 commits) + ruling v1.2 unchanged; counter 0/3 RESET. Fresh 3-pass NITPICK_ONLY reconvergence against 5c8db39 / v2.26 next. develop @ 7fcf0cf. |

## Phase Progress

| Phase | Status | Finding Progression |
|-------|--------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift (2026-06-24) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 (2026-06-24) |
| Phase 3 — TDD Implementation | COMPLETE | W6 CONVERGED 3/3 (2026-07-02); all waves merged |
| Phase 4 — Holdout Evaluation | COMPLETE | PASS 0.895 re-eval 2026-07-12 @ f73676d |
| Phase 5 — Adversarial Refinement | **CONVERGED** BC-5.39.001 — streak 3/3 (P37/P38/P39); 39 findings remediated; MERGED PR #115 @ 8eb54a5 | →21→7→4→3 |
| Phase 6 — Formal Hardening | COMPLETE 2026-07-06 — 63/77 VPs PROVEN; fuzzers clean; security scan clean | evidence: cycles/cycle-1/phase-6/ |
| Phase 7 — Convergence | **CONVERGED** 2026-07-06 (human-approved); fresh-context audit CONVERGENCE-CLEAN; CYCLE-1 CLOSED | evidence: cycles/cycle-1/phase-7/ |
| pass-12 adversary (S-BL.ADMISSION-SYNC-WIRE Step-4.5) | CONVERGED — 12 passes total; passes 1-9 HAS_FINDINGS; passes 10/11/12 NITPICK_ONLY (3/3 clean streak) | →3→3→3→3 |
| fix burst (S-BL.ADMISSION-SYNC-WIRE Step-4.5) | Rulings 12–15; BC-2.05.009 v1.0→v1.6; code HEAD ab043c5→92a2c65 (squash) | 4 fix bursts |
| S-BL.NODE-IDENTIFY-WIRE Step-4.5 adversary | **DELIVERED** PR #127 @ 7fcf0cf; Step-4.5 3/3 NITPICK_ONLY (BC-5.39.001); F-1 stored-key + F-2 log + MED-1 + LOW-1 fixed | →2→0→0→0 |

Wave-by-wave detail: `cycles/cycle-1/burst-log.md` and `cycles/cycle-1/closed-stories.md`.

## Convergence Status

Trajectory →21→7→4→3. Phase 5 aggregate: 39 passes. ADMISSION-SYNC-WIRE: CONVERGED 3/3 2026-07-18. NODE-IDENTIFY-WIRE: CONVERGED 3/3 NITPICK_ONLY 2026-07-19 (PR #127 @ 7fcf0cf).

## Current Phase Steps

Older rows archived to `cycles/cycle-1/burst-log.md`. Showing last 5 rows.

| Date | Step | Status | Result |
|------|------|--------|--------|
| 2026-07-20 | **S-BL.DISCOVERY-WIRE Step-4.5 map-bounding arc: Pass-14 LOW (relayRateCap.last) → fix both maps; ruling v1.1 (Option A); 52c422a + 545429f (28 commits); story v2.22 / 7ff0732; SEC-DW-10; 7 tests; all 6 gates green; counter RESET 0/3 (major edit).** | map-bounding-arc | develop @ 7fcf0cf. |
| 2026-07-20 | **S-BL.DISCOVERY-WIRE Step-4.5 Pass-A/B reconvergence: Pass-A MED-1 (stale cold-start-only eviction comments, 2 instances in discovery_wire.go) + Pass-B N-1 (false tie-determinism claim) — fixed comment-only at dd7b821 (29 commits); class sweep confirms zero remaining; all 6 gates green; counter 0/3 RESET.** | pass-ab-comment-fix | develop @ 7fcf0cf. |
| 2026-07-20 | **S-BL.DISCOVERY-WIRE Step-4.5 second remediation burst: ruling v1.2 (Decision-8 self-eviction guarantee corrected); reconvergence F-1/F-2/F-3 + v1.1→v1.2 sweep applied; story v2.24 / def6b7b at factory 930f266; code 5c8db39 (26 commits); all 6 gates green; counter 0/3 RESET.** | remediation-burst-2 | develop @ 7fcf0cf. |
| 2026-07-20 | **S-BL.DISCOVERY-WIRE Step-4.5 Pass-1 F-1 (MED) remediated: story-body SEC-DW-10 self-eviction guarantee corrected to "improbable-not-impossible; benign if evicted" matching ruling v1.2 Decision 8 (v2.24 burst missed this transcription); story v2.25 / def6b7b (input-hash unchanged); factory 68fb3fe; code 5c8db39 (26 commits) + ruling v1.2 unchanged; counter 0/3 RESET.** | pass-1-f1-remediation | develop @ 7fcf0cf. |
| 2026-07-20 | **S-BL.DISCOVERY-WIRE Step-4.5 Pass-2 traceability F-1 (MED) remediated: declared-input BC pins re-synced to canonical (BC-2.03.001 v1.6→v1.7, BC-2.01.008 v1.2→v1.3; substance unchanged — prose-label drift from narrowed re-cert sweep-set); story v2.26 / def6b7b (input-hash unchanged); factory 43f2e47; code 5c8db39 (26 commits) + ruling v1.2 unchanged; counter 0/3 RESET.** | pass-2-traceability-f1-remediation | develop @ 7fcf0cf. |

## Wave 6 Story Status

| Story | Title | Tranche | PR | SHA |
|-------|-------|---------|----|-----|
| S-BL.LOOKUP | AdmittedKeySet.Lookup value-return migration | A | #40 | eac5d0a |
| S-W5.04 | daemon paths.list/router.metrics/router.status handlers | A | #41 | 851e164 |
| S-6.07 | admin.svtn.create handler + sbctl CLI (v1.14) | A | #42 | 446efce |
| S-7.01 | XOR parity FEC for single-loss recovery | B | #43 | 5c658e7 |
| S-7.02 | SVTN-scoped multicast session discovery | B | #55 | c54a8ad |
| S-BL.ROUTER-ADDR | populate PathSnapshot.RouterAddr (BC-2.06.003 PC-1) | B | #56 | 91d5675 |
| S-7.03 | (Tranche C) | C | #60 | 7142146 |
| S-6.05 | (Tranche C) | C | #61 | 7fe3e29 |

Waves 1–5 detail: `cycles/cycle-1/closed-stories.md`.

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| DRIFT-SIGHUP-MODE-ASYMMETRY | LOW | kill -HUP reloads router but terminates other modes. Anchor: S-BL.CLI-SURFACE-COMPLETION. | architect | open |
| DRIFT-SIGHUP-INERT-RELOAD-UX | LOW | Valid SIGHUP reload with no upstream changes is silently inert. Anchor: S-BL.CLI-SURFACE-COMPLETION. | product-owner | open |
| W3-DEFER-1..6 | MED/OBS | Worktree tuple; M-1 relay busy-spin; fired-source LRU; M-2 unbounded log; EC-005; PTY-EOF. Detail: `cycles/cycle-1/closed-drift.md`. | various | deferred |
| OBS-VP-BENCH | OBS | VP-042 re-anchored → S-BL.LOOPBACK-FULLSTACK (draft v1.1, AC-001 OnAck gate). | orchestrator | re-anchored |
| WAVE-GATE-DISPATCH-INTEGRITY | HIGH | HEAD-SHA tuple absent from adversary dispatch. POL-005 local mitigation. Upstream: drbothen/vsdd-factory#448. | orchestrator | mitigated-local |
| F-DW-IMPL-001 | HIGH | execute-against-baseline premise-tracing gap. Upstream: drbothen/vsdd-factory#620. | orchestrator | filed upstream |
| DRIFT-DOCS-LOG-LEVEL | LOW | docs/* cite log_level but config.Config rejects it (E-CFG-005). | technical-writer | open |
| CI-FLAKE-DISCOVERY-HEARTBEAT | LOW | TestDiscovery_Advertise_PeriodicHeartbeat timing flake @ 92a2c65 (run #29659181289). Dispositioned FLAKE; NOT a merge-blocker. | orchestrator | known-flake |
| NODEADDR-WIDTH-8B | OBS | 8-byte DeriveNodeAddress width ADR candidate. Anchor: rulings §18. | architect | deferred |
| SEC-NIDW-SVTNID-CONSISTENCY | MED | ChallengeResponse outer-header SVTNID not validated vs NodeIdentify SVTNID. Post-merge sec review, PR #127. | security-reviewer | story-authored (S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY v1.0, ready) |
| FO(g) — DISCOVERY-WIRE | LOW | Dynamic discovery-listener registration for post-startup SVTNs. Deferred per task6d ruling v1.0 Decision 5. Cold-start: empty snapshot → zero listeners spawned → hop-2 inert until restart. Target: future story. | architect | open (non-blocking) |
| FO(h) — DISCOVERY-WIRE | LOW | Full-daemon e2e relay fan-out integration test deferred. Unit+inspection+seam-test covered (TestRelayDispatch_* 6b/6c, onRelay-seam 6d, daemon-join oracle TestRunRouter_WithAdmittedSVTN_ShutsDownCleanly); no single e2e sending a real HMAC-authenticated advertisement and observing DISCOVERY_RELAY on a live TCP connection. Deferred as too flaky/heavy for a deterministic per-story gate. Target: future story. | architect | open (non-blocking) |

Additional drift items: `cycles/cycle-1/closed-drift.md`.

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| **DISCOVERY-WIRE Task 6a-6d CODE-COMPLETE** | Worktree af91335 (12 commits); story v2.17; task6d ruling v1.0; FO(g) deferred | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 passes 1-3 ALL FIXED** | HIGHs + MEDs + LOWs fixed; story v2.20 (input-hash 5a4d0da); worktree 88d015e→1cd8457; counter 0/3 | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 pass-7 LOW fixed + 2 self-corrections** | Comment-only fixes; worktree 0821149→7d48e14 (22 commits); story v2.20 / 5a4d0da unchanged; counter 0/3 reset | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 passes 8+9 concurrent — prior sweep incomplete** | 3 more stale-comment instances fixed comment-only at f638535 (23 commits); class fully retired; all 6 gates green; story v2.20 / 5a4d0da unchanged; counter 0/3 reset | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 TD-031 NITPICK fix at 8058104** | Passes 10/11/12 NITPICK_ONLY 3/3 at f638535; user-approved: volatile/drifted line-citation class fixed comment-only at 8058104 (24 commits); story v2.20 / 5a4d0da UNCHANGED; counter RESET 0/3 | 2026-07-20 |
| **DISCOVERY-WIRE map-bounding arc at 545429f** | Pass-14 LOW (unbounded relayRateCap.last) escalated to fix both maps; ruling v1.1 (Option A); 52c422a + 545429f (28 commits); story v2.22 / 7ff0732; SEC-DW-10; 7 mutation-verified tests; all 6 gates green; counter RESET 0/3 | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 Pass-A/B comment-only fix at dd7b821** | Pass-A MED-1 (stale cold-start-only eviction comments, 2 instances) + Pass-B N-1 (false tie-determinism claim) fixed comment-only; class swept; worktree dd7b821 (29 commits); story v2.22 / 7ff0732 UNCHANGED; all 6 gates green; counter RESET 0/3 | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 Pass-C F-1 citation-drift fix at ddc9bf9** | 2 FCL rows + 2 code comments cited map-bounding-ruling v1.0 (Decisions 1 & 4 byte-identical v1.0→v1.1; version pin stale); fixed comment-only at ddc9bf9 (25 commits); story v2.23 / 7ff0732 at factory 1f40273; all 6 gates green; counter RESET 0/3 | 2026-07-20 |
| **DISCOVERY-WIRE v2.24 second remediation burst at 930f266** | ruling v1.2 (Decision-8 self-eviction guarantee corrected — watermark-first makes advancing key improbable not impossible LRU victim; eviction benign per EC-006); reconvergence F-1/F-2/F-3 + v1.1→v1.2 sweep; story v2.24 / def6b7b; code 5c8db39 (26 commits); all 6 gates green; counter RESET 0/3 | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 Pass-1 F-1 (MED) remediated at factory 68fb3fe** | story-body SEC-DW-10 self-eviction guarantee corrected to "improbable-not-impossible; benign if evicted" matching ruling v1.2 Decision 8 (v2.24 burst missed this transcription); story v2.25 / def6b7b (input-hash unchanged — body-only); code 5c8db39 + ruling v1.2 unchanged; counter 0/3 RESET | 2026-07-20 |
| **DISCOVERY-WIRE Step-4.5 Pass-2 traceability F-1 (MED) remediated at factory 43f2e47** | declared-input BC pins re-synced to canonical: BC-2.03.001 v1.6→v1.7 (10 spots; v1.7 added PC-4=LocalNodeAdmissionPubkey, annotated out-of-scope) + BC-2.01.008 v1.2→v1.3 (2 spots; v1.3 added NODE_IDENTIFY=0x04 row); substance unchanged; story v2.26 / def6b7b (input-hash unchanged — prose/metadata-only); code 5c8db39 + ruling v1.2 unchanged; counter 0/3 RESET | 2026-07-20 |

Full decision detail: `cycles/cycle-1/burst-log.md` and `cycles/cycle-1/state-history-discovery-wire.md` (9 older rows extracted 2026-07-20).

## Skip Log

| Step | Skipped? | Justification |
|------|----------|---------------|
| UX Spec | yes | CLI/daemon product — no UI surfaces |

## Blocking Issues

| ID | Issue | Severity | Blocking Phase | Owner | Resolution |
|----|-------|----------|---------------|-------|------------|
| (none open) | All blockers resolved or deferred to cycle files | — | — | — | — |

## Historical Content

Burst logs, adversary pass details, session checkpoints, and lessons
have been extracted to cycle files:

- Burst history: `cycles/cycle-1/burst-log.md`
- Convergence trajectory: `cycles/cycle-1/convergence-trajectory.md`
- Session checkpoints: `cycles/cycle-1/session-checkpoints.md`
- Lessons learned: `cycles/cycle-1/lessons.md`
- Resolved blockers: `cycles/cycle-1/blocking-issues-resolved.md`

## Session Resume Checkpoint

**Position:** S-BL.DISCOVERY-WIRE Step-4.5 Pass-2 traceability F-1 (MED) remediated (2026-07-20). Story bumped v2.25→v2.26 / input-hash def6b7b UNCHANGED (prose/metadata-only fix) at factory `43f2e47`. Fix: two declared-input BCs had stale version pins — BC-2.03.001 v1.6→v1.7 (10 spots: inputDocuments comment + 3 body refs + 7 Anchors Consumed rows; v1.7 added PC-4=LocalNodeAdmissionPubkey owned by BC-2.09.004/S-BL.NODE-ADMISSION-PROVISIONING, annotated out-of-scope) and BC-2.01.008 v1.2→v1.3 (2 spots; v1.3 added NODE_IDENTIFY=0x04 row); substance cited unchanged between pinned and canonical versions — no AC/behavior affected. Code HEAD `5c8db39` (26 commits vs develop). Ruling v1.2 UNCHANGED. BC-5.39.001 counter RESET 0/3. develop @ 7fcf0cf.

**Next candidates:** (a) S-BL.DISCOVERY-WIRE Step-4.5 fresh 3-pass NITPICK_ONLY reconvergence against `5c8db39` / story v2.26 (POL-005 HEAD-SHA tuple mandatory: factory HEAD at commit time + code worktree HEAD `5c8db39` both required in dispatch prompt); (b) S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY (ready v1.0, 3 ACs, 3 pts). S-BL.LOOPBACK-FULLSTACK parked (P2, 8pts, AC-001 OnAck gate).

**Resume protocol:** (1) `factory-worktree-health` check FIRST; (2) read STATE.md + `stories/sprint-state.yaml`; (3) dispatch Step-4.5 adversarial reconvergence pass-1 for S-BL.DISCOVERY-WIRE (code worktree `5c8db39`, 26 commits, story v2.26 / def6b7b, POL-005 HEAD-SHA tuple mandatory) or deliver S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY.

## Concurrent Cycles

| Cycle | Status |
|-------|--------|
| cycle-1 (v1.0.0-greenfield) | ACTIVE — steady-state story delivery; trajectory-tail →21→7→4→3 |
