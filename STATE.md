---
pipeline: IN_PROGRESS
phase: phase-3-tdd-implementation
phase_step: wave-5-mgmt-plane-planned
phase_3_active_wave: 5
phase_3_active_stories: []
phase_3_completed_stories: [S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03, S-4.01, S-4.02, S-4.03, S-4.04, S-6.01]
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
develop_head: 01ae50c
open_prs: 0
timestamp: 2026-06-29T00:00:00Z
last_update: 2026-06-28
---

# Switchboard Factory State

## Current State

Wave 5 RE-SCOPED to 7 stories / 38 pts (Observability + CLI + Management Plane). Net-new: S-W5.01 (internal/mgmt server + E-CFG-008/009 + cmd/switchboard wiring for all 4 daemon modes, 8pt) and S-W5.02 (e2e management plane harness, 5pt). S-6.03 re-scoped v2.0 to client-auth-only boundary (Authenticate() fail-closed, 5pt). S-5.02 repointed 3→5. Management plane ADR-012: NDJSON over Unix/TCP socket, Ed25519 challenge-response, 64 KiB bounded reads, fail-closed Authenticate(). BC-2.07.004 minted (45 total); VP-064..VP-067 minted (67 total). Fresh-context gate audit C=0 H=3 M=4 L=3 — all H/M resolved; F-009 (ARCH-INDEX input-hash field-name mismatch) converted to tracked TODO. S-5.03 merged via PR #30 (01ae50c) on origin/develop — local develop is 1 commit behind (pull before TDD). Serialization: S-6.03 → {S-6.02, S-5.02} in sequence; S-W5.01 ∥ sbctl-side stories (no cmd/sbctl conflict); S-W5.02 gates on S-6.03 + S-W5.01.

## Phase Progress

| Phase | Status | Gate | Date | Finding Progression |
|-------|--------|------|------|---------------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift | 2026-06-24 | 27→18→17→21→17→14→7→9 (8 passes) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 | 2026-06-24 | — |
| Phase 3 — TDD Implementation | IN_PROGRESS | Wave 4: GATE CLOSED/APPROVED — 5/5 merged, 6/6 adversary clean, consistency audit PASS | 2026-06-28 | W4: 6/6 passes C=0/H=0/M=0 |

## Wave / Story Status

Waves 1–3 complete (11 stories + 3 fix PRs, PRs #1–#20). Detail: `cycles/cycle-1/closed-stories.md`.

| Wave | Story | Title | Status | PR | SHA |
|------|-------|-------|--------|----|-----|
| 4 | S-4.01 | Per-path RTT/loss tracking + dedup/race dispatch | MERGED | #24 | e415d31 |
| 4 | S-4.02 | Upstream replay (internal/replay) | MERGED | #25 | 95729c7 |
| 4 | S-4.03 | Downstream ARQ + TLPKTDROP (internal/arq) | MERGED | #26 | 8d9744f |
| 4 | S-4.04 | Split-horizon loop prevention + drop-cache router wiring | MERGED | #27 | 42c51e2 |
| 4 | S-6.01 | Config parsing and validation | MERGED | #28 | abeba27 |
| 4 | hygiene | Doc-hygiene: stale ref + leftover stub docstring fix | MERGED | #29 | 7ef43b8 |
| 5 | S-5.03 | flag paths degraded when EWMA RTT > 200ms | MERGED | #30 | 01ae50c |
| 5 | S-5.01 | Green/yellow/red quality indicator with hysteresis | pending | — | — |
| 5 | S-5.02 | sbctl paths list / router metrics + alias + p99 | pending | — | — |
| 5 | S-6.02 | SVTN lifecycle and key management via sbctl admin | pending | — | — |
| 5 | S-6.03 | sbctl client auth (Authenticate() fail-closed), flag parsing, JSON, error | pending | — | — |
| 5 | S-W5.01 | internal/mgmt server + E-CFG-008/009 + cmd/switchboard wiring (4 modes) | draft | — | — |
| 5 | S-W5.02 | e2e management plane harness: sbctl auth + RPC across 4 daemon types | draft | — | — |

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
Resolved items (C-1/OBS-3, T2, SW305-M1..M8, HF3, S402-F006, S403-O1, Phase-6 deferrals, BC-2.09.003-STALE, S601-NITPICK-A..E, S601-DRAFT-STORY, S403-COS1/2, S404-OBS-G, S401-O3, W5-gate-H1..H3/M1..M4): `cycles/cycle-1/closed-drift.md`

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
| S-4.04 MERGED (42c51e2, PR #27) | 7/7 ACs, 3/3 adversary clean; SEC-001 accepted per BC-2.02.009 EC-004 | 2026-06-28 |
| S-6.01 MERGED (abeba27, PR #28) | 9/9 ACs, 3/3 adversary clean; SEC-001/SEC-002 deferred LOW | 2026-06-28 |
| Wave 4 gate APPROVED | 6/6 diverse-lens passes C=0/H=0/M=0; consistency audit CONDITIONAL PASS (14 findings all resolved); doc-hygiene PR #29 (7ef43b8) closed L-1 + S403-COS1/COS2 | 2026-06-28 |
| VP-061/VP-062 minted (S-5.02 Phase-6 hardening) | VP-061: metrics content-absence code-audit (DI-001); VP-062: JSON well-formedness fuzz (all CLI forms + alias). Both trace BC-2.06.003. | 2026-06-28 |
| VP-063 minted (S-5.03 Wave-5 functional) | Dedicated proptest for PathTracker.IsDegraded() EWMA vs DegradedRTTThresholdMS (200 ms). Traces BC-2.02.003 PC-5. | 2026-06-28 |
| BC-2.06.003 v1.3 (sbctl canonical+alias + rtt_p99_ms) | Reconciles sbctl metrics surface: canonical `paths list`, router-metrics alias `router metrics`, router-status alias `router status`; adds rtt_p99_ms field. Closes consistency-audit F-001..F-007. | 2026-06-28 |
| S-5.03 degraded-path-flag (new story) | New Wave-5 story closing drift S401-O3; implements BC-2.02.003 PC-5 IsDegraded() in internal/paths; VP-063 is its formal property. | 2026-06-28 |
| Build whole management plane (Wave 5) | net-new internal/mgmt server + ADR-012 wire protocol (NDJSON, Ed25519 challenge-response, 64 KiB bounded reads, fail-closed Authenticate()) + e2e across 4 daemon types; S-6.03 re-scoped, S-W5.01/S-W5.02 created; +13pt. BC-2.07.004 + VP-064..VP-067 minted. | 2026-06-28 |
Older decisions (Wave 3 per-story, S-4.01..S-4.03 rulings): `cycles/cycle-1/burst-log.md` (archived 2026-06-28).

## Session Resume Checkpoint — 2026-06-28 (Wave 5 management plane planned)

**Position:** Phase 3 Wave 5 RE-SCOPED (7 stories / 38 pts). Spec change set committed to factory-artifacts. Fresh-context gate audit C=0 H=3 M=4 L=3 — all H/M resolved; F-009 tracked TODO. origin/develop HEAD = 01ae50c (S-5.03 merged PR #30). Local develop is 1 commit behind — run `git pull` before starting Wave-5 TDD. l4_vp_count = 67 (VP-061..VP-067 minted); l3_bc_count = 45 (BC-2.07.004 added). 0 open PRs.

**Wave 5 stories (7 / 38 pts):** S-5.03 MERGED (01ae50c); S-5.01 quality-indicator (5pt, depends S-5.03); S-5.02 sbctl-metrics-query (5pt, needs S-5.01 + S-6.03); S-6.02 svtn-lifecycle (8pt); S-6.03 sbctl-client-auth v2.0 (5pt); S-W5.01 internal/mgmt server (8pt, parallel-safe with sbctl stories); S-W5.02 e2e harness (5pt, gates on S-6.03 + S-W5.01). Constraint: S-6.02 ∥ S-5.02 FORBIDDEN (both edit cmd/sbctl/main.go).

**NEXT ACTION on resume:** Pull develop (01ae50c). Begin Wave-5 TDD with S-6.03 + S-W5.01 (two roots with no intra-wave deps). S-6.03 creates cmd/sbctl scaffold; S-W5.01 creates internal/mgmt on a parallel branch.

**Open deferred LOW items:** S601-SEC-001 (CWE-117), S601-SEC-002 (CWE-400), S404-LOW-1 (3 LOW + NITPICK from S-4.04 adversary). Address in Wave 5 or dedicated hardening pass.

**Settled rulings:** RULING-001/002/002-A1/003-v1.1 and F-A-001 (VP-052 re-anchored) — do NOT re-open unless a fresh pass finds a NEW Critical/High.

Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.

## Historical Content

Burst logs, adversary passes, session checkpoints, closed-stories, closed-drift: `cycles/cycle-1/`
