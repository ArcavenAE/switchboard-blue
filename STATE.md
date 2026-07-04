---
pipeline: IN_PROGRESS
phase: phase-5-adversarial-refinement
phase_step: phase-5-pass-27-concluded-has-findings
product: switchboard
mode: greenfield
current_cycle: cycle-1
anchor_strategy: reference-via-frontmatter
dtu_required: false
dtu_assessment: 2026-06-23
internal_packages: 18
plugin_version_adopted: "1.0.0-rc.21"
l2_complete: true
l3_complete: true
l3_bc_count: 45
l4_complete: true
l4_vp_count: 77
arch_sections: 13
arch_adrs: 8
phase_1_gate: APPROVED
phase_2_gate: APPROVED
wave_1_gate: PASS_WITH_CLEAN_DRIFT
wave_2_gate: PASS_WITH_OBSERVATIONS
wave_3_gate: APPROVED
wave_4_gate: APPROVED
wave_5_gate: CONVERGED
wave_6_gate: CONVERGED_3_OF_3
phase_4_gate: PASS_AT_THRESHOLD
phase_5_pass_4_gate: BC_5_39_001_SATISFIED
develop_head: 6deda15
open_prs: 0
alpha_release_tag: alpha-20260629-165045-d854978
awaiting: phase-5-pass-28-dispatch
historical_cycles: []
timestamp: 2026-07-03T00:00:00Z
last_update: 2026-07-03
---

# Switchboard Factory State

## Current State

Phase 5 adversarial refinement in progress. Latest closed pass: 26 (both lanes HAS_FINDINGS; remediated Bursts 68a c10d6ba + 68b cf135b9 + Burst 69 1f2f557 persistence + [process-gap] POL-006 codification). Streak stays 0/3. Pass 27 dispatched Burst 70.
develop HEAD: 6deda15 (unchanged since PR #70 admin-test scrub 2026-07-02). 45 BCs, 77 VPs, 54 stories (STORY-INDEX v3.79 sum-by-wave 185 pts), 18 internal packages.

NO-GOVERNING-BC obligations: `paths ping` (§77) + `svtn status` (§62) — architect ruling or new BC required before S-BL.CLI-SURFACE-COMPLETION scheduling.

Sidecar reviews: `.factory/cycles/cycle-1/adversarial-reviews/W-6-wavegate-pass-{1-6}-Adv-{A,B}.md`.
Phase 4 report: `.factory/holdout-scenarios/evaluations/HS-006-evaluation-2026-07-02.md`.

## Phase Progress

| Phase | Status | Latest Gate |
|-------|--------|-------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift (2026-06-24) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 (2026-06-24) |
| Phase 3 — TDD Implementation | COMPLETE | W6 CONVERGED 3/3 (2026-07-02); all waves merged |
| Phase 4 — Holdout Evaluation | COMPLETE | PASS_AT_THRESHOLD 0.85 (2026-07-02) |
| Phase 5 — Adversarial Refinement | PASS_19_HAS_FINDINGS | P1: 3H/3M/1L → REM → P2: 0H/3M/2L → REM → P3: 3H/4M/2L+6obs → Path B rem spec+code → P4 COMPLETE (9 findings → 3/3 CLEAN streak) → P5: 0H/4M/3L+2obs → REM (Burst 21: spec v1.18 + PR #64 d012dbf) → P6: Adv-A 1H/4M/1L + Adv-B CLEAN(2obs) → REM (Burst 23: PR #65 4d7d9e0 + v1.19/BC v1.9/S-6.03 v2.8) → P7: Adv-A 0H/3M/0L + Adv-B CLEAN(5obs) → REM (Burst 25: PR #66 b4ccd06, usageErrf sweep complete) → P8: Adv-A 2H/4M/1L + Adv-B 0H/2M+1obs → REM (Burst 27: PR #67 32ea461 + v1.20) → P9: Adv-A 1H/2M/3L (all spec-side) + Adv-B CLEAN(3obs) → REM (Burst 29: v1.21 spec-only) → P10: Adv-A 1H/1M + Adv-B 1L(2obs) → REM (Burst 31: PR #68 66e9ddc + v1.22) → P11: Adv-A 1H/1M/3obs HAS_FINDINGS + Adv-B CLEAN(3obs) → REM (Burst 33: spec-only v1.23, revoke confirm carve-out + §109 --role syntax) → P12: Adv-A 0H/2M/2obs HAS_FINDINGS + Adv-B CLEAN(3obs) → REM (Burst 35: spec-only v1.24, §111 list-keys exit codes + --svtn <svtn-name> placeholder class sweep incl. §130 recover + §108/§120 confirm-family symmetry) → P13: Adv-A 1H/1M/2obs + Adv-B 1L/2obs HAS_FINDINGS → REM (Burst 37: PR #69 03ce8e7 code; Burst 38: spec v1.25) → P14: Adv-B-v2 0H/0M/5F HAS_FINDINGS (B-001 DEFERRED, B-002..B-005 SHIPPED) → REM (Burst 40a/b: S-6.06 v1.25 + BC-2.05.004 v1.14 + VP-077 + PR #70 6deda15) → P15: Adv-A 0H/1M + Adv-B 0H/1M HAS_FINDINGS (A-001 interface-definitions v1.26 SHIPPED 5e42768; B-001 VP-077 v1.1 SHIPPED 5120c9e) → REM → P16: Adv-A 0H/1M HAS_FINDINGS (F-P5P16-A-001 $schema envelope drift SHIPPED 041ea2f v1.27); Adv-B CLEAN → REM → P17: Adv-A 0H/2M HAS_FINDINGS (F-P5P17-A-001 svtn_id phantom + F-P5P17-A-002 path_distribution ratio→count SHIPPED 2be16e5 v1.28); Adv-B CLEAN → REM → P18: Adv-A 0H/1M HAS_FINDINGS (F-P5P18-A-001 8-story frontmatter status drift SHIPPED f8b2d7e); Adv-B 0H/1M HAS_FINDINGS (F-P5P18-B-001 STORY-INDEX VP-coverage 76/76→77/77 SHIPPED bc79621) → REM → P19: Adv-A 0H/1M HAS_FINDINGS (F-P5P19-A-001 STORY-INDEX Wave-6 aggregate arithmetic SHIPPED e65e429 STORY-INDEX v3.75); Adv-B 2H/1L HAS_FINDINGS (F-P5P19-B-001 ARCH-11 VP-077 propagation SHIPPED dd97736 v1.16; F-P5P19-B-002 ARCH-07 VP-catalog+triplet SHIPPED 1a55096 v1.9; F-P5P19-B-003 VP-077 pin syntax SHIPPED e50f96d v1.2) → Streak 0/3; P20 next → P20: Adv-A 0H/1M HAS_FINDINGS (F-P5P20-A-001 STORY-INDEX v3.75 Wave-6 aggregate omits S-BL.ROUTER-ADDR SHIPPED 5fcf305 v3.76); Adv-B 0H/1H HAS_FINDINGS (F-P5P20-B-001 ARCH-11 v1.16+ARCH-07 v1.9 VP-043 method column stale SHIPPED 1e9fbff ARCH-11 v1.17+ARCH-07 v1.10) → Streak 0/3; P21 next → P21: Adv-A CLEAN (10 AF) + Adv-B CLEAN (11 AF); F-P5P20-A-001 + F-P5P20-B-001 remediations verified complete → PASS_21_CLEAN (1/3) → P22: Adv-A 0H/1M HAS_FINDINGS (F-P5P22-A-001 POL-002 sibling-sweep S-1.01+S-2.01 frontmatter status stale SHIPPED d1ef9a7+10b3e5e); Adv-B CLEAN 2obs (O-P5P22-B-001 POL-003-candidate VP-062 v1.13 pin drift; O-P5P22-B-002 cosmetic ARCH-11 v1.17 mod-note) → streak reset 1/3 → 0/3 → P23: Adv-A 1H HAS_FINDINGS (F-P5P23-A-001 POL-002 sibling-sweep recurrence S-W5.04+S-BL.LOOKUP frontmatter status: draft vs STORY-INDEX merged rows SHIPPED .factory 2256ba6); Adv-B CLEAN 1obs (O-P5P23-B-001 cosmetic ARCH-11 legacy "+audit" suffix L50/L78) → streak held 0/3 → P24: Adv-A 1H HAS_FINDINGS (F-P5P24-A-001 POL-002+POL-001 story-mint gap S-BL.ADMINWIRE-EXTRACTION absent STORY-INDEX 4 places SHIPPED ffb028b); Adv-B 3M HAS_FINDINGS (F-P5P24-B-001/002/003 POL-006 ARCH-11 reverse-trace propagation gap for VP-042/VP-008/VP-012 dual-anchor BCs SHIPPED ffb028b) → streak held 0/3 → P25: Adv-A 1H+2M HAS_FINDINGS (F-P5P25-A-001 HIGH POL-002 sprint-state 5+ axes systemic staleness SHIPPED dfa4d33; F-P5P25-A-002 MED STATE.md/sprint-state wave_6_gate cross-artifact SHIPPED dfa4d33; F-P5P25-A-003 MED sprint-state header phase/total_stories drift SHIPPED dfa4d33); Adv-B 2M HAS_FINDINGS (F-P5P25-B-001 MED POL-006 ARCH-11 L85 BC-2.07.002 missing VP-067 third instance of F-P5P24-B-*** class SHIPPED 99f1356; F-P5P25-B-002 MED POL-005 BC-2.07.002 L180 VP table method unit vs canonical integration SHIPPED 99f1356) → streak stays 0/3; P26: Adv-A 1H/1M HAS_FINDINGS (F-P5P26-A-001 HIGH POL-002 sprint-state 6-axis systemic staleness recurrence SHIPPED c10d6ba v1.51; F-P5P26-A-002 MED wave_5_gate cross-artifact drift absorbed into A-001 Axis 4); Adv-B 0H/2M HAS_FINDINGS (F-P5P26-B-001 MED POL-006 ARCH-11 L57 BC-2.02.003 missing VP-040 fourth-instance F-P5P24-B-*** class SHIPPED cf135b9 v1.20; F-P5P26-B-002 MED POL-006 ARCH-11 L50 BC-2.01.003 missing VP-016 fifth-consecutive-recurrence FIRST dual-anchor VP; Burst 68b full-file 77-VP reverse-trace sweep confirmed ONLY these 2 gaps SHIPPED cf135b9) → streak stays 0/3 → P27: Adv-A 1H/2M/1L HAS_FINDINGS (F-P5P27-A-001 HIGH POL-002 Wave-5 stories_merged+merge_shas missing S-W5.02 SHIPPED <burst-71a> v1.53; F-P5P27-A-002 MED STATE.md stale Pass-12 narrative SHIPPED <burst-71a>; F-P5P27-A-003 MED Wave-5 wave_adversary_converged false→true SHIPPED <burst-71a> v1.53; F-P5P27-A-004 LOW pending intent adjudicated by-design no-change); Adv-B 1H/1L/1obs HAS_FINDINGS (F-P5P27-B-001 HIGH POL-008 4 session-access rows Phase-column drift BC-2.04.001/003/004/005 SHIPPED <pending-71b> v1.21; F-P5P27-B-002 LOW POL-008 BC-2.05.007 stale "+audit" method annotation SHIPPED <pending-71b> v1.21; O-P5P27-B-001 full-file 12-dual-anchor-VP sweep confirmed clean baseline) → streak stays 0/3 |

Wave-by-wave detail: `cycles/cycle-1/burst-log.md` and `cycles/cycle-1/closed-stories.md`.

## Current Phase Steps

Older rows archived to `cycles/cycle-1/burst-log.md` (compact-state routing). Showing last 5 rows.

| Date | Step | Status | Result |
|------|------|--------|--------|
| 2026-07-03 | phase-5-pass-25-concluded-has-findings | completed | HAS_FINDINGS both lanes; Adv-A 1H+2M, Adv-B 2M; remediated Burst 65a dfa4d33 + 65b 99f1356; streak stays 0/3 |
| 2026-07-03 | phase-5-pass-26-dispatch | completed | fresh-context split-adversary dispatched (Adv-A + Adv-B in parallel) against develop@6deda15 factory@9f069ef |
| 2026-07-03 | phase-5-pass-26-concluded-has-findings | completed | HAS_FINDINGS both lanes; Adv-A 1H+1M sprint-state 6-axis sibling-sweep + wave_5_gate cross-artifact (absorbed); Adv-B 0H+2M POL-006 fifth-consecutive-recurrence FIRST dual-anchor; remediated Burst 68a c10d6ba + 68b cf135b9 (incl. proactive full-file 77-VP sweep — confirmed clean baseline); streak stays 0/3 |
| 2026-07-03 | phase-5-pass-27-dispatch | completed | fresh-context split-adversary dispatched (Adv-A + Adv-B in parallel) against develop@6deda15 factory@1f2f557 |
| 2026-07-03 | phase-5-pass-27-concluded-has-findings | completed | HAS_FINDINGS both lanes; Adv-A 1H/2M/1L (F-P5P27-A-001 S-W5.02 stories_merged gap, F-P5P27-A-002 stale narrative, F-P5P27-A-003 wave_adversary_converged contradiction, F-P5P27-A-004 LOW by-design); Adv-B 1H/1L/1obs (F-P5P27-B-001 4 session-access Phase-column drift, F-P5P27-B-002 BC-2.05.007 stale audit annotation); remediated Burst 71a/71b; streak stays 0/3 |

## Wave 6 Story Status

| Story | Title | Tranche | PR | SHA |
|-------|-------|---------|----|-----|
| S-BL.LOOKUP | AdmittedKeySet.Lookup value-return migration | A | #40 | eac5d0a |
| S-W5.04 | daemon paths.list/router.metrics/router.status handlers | A | #41 | 851e164 |
| S-6.07 | admin.svtn.create handler + sbctl CLI (v1.13) | A | #42 | 446efce |
| S-7.01 | XOR parity FEC for single-loss recovery | B | #43 | 5c658e7 |
| S-7.02 | SVTN-scoped multicast session discovery | B | #55 | c54a8ad |
| S-BL.ROUTER-ADDR | populate PathSnapshot.RouterAddr (BC-2.06.003 PC-1) | B | #56 | 91d5675 |
| S-7.03 | (Tranche C) | C | #60 | 7142146 |
| S-6.05 | (Tranche C) | C | #61 | 7fe3e29 |

Waves 1–5 detail: `cycles/cycle-1/closed-stories.md`.

## Open Drift Items

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| W3-R2-M2 | MED | Route-time LWW snapshot: concurrent RegisterForwardingEntry not atomic with HMAC verify. | architect/implementer | open |
| SW305-M4 | MED | W4-TEST-001: RouteFrame fire-once E-ADM-017 integration test (real FailureCounter + WithNow). | test-writer | DEFER-WAVE-4 |
| process-gap-follow-up | OBS | Adversary nil-safety lens gap (missed SEC-001); candidate self-improvement story. | orchestrator | open/deferred |
| W3-DEFER-1..6 | MED/OBS | Worktree tuple codification; M-1 relay busy-spin; fired-source LRU eviction; M-2 unbounded E-ADM-016 log; EC-005 import-boundary lint; real-connector PTY-EOF integration. Detail: `cycles/cycle-1/closed-drift.md`. | various | deferred |
| S402-F007 | LOW | S-4.02: ARCH-03 N=3 vs BC-2.02.004 N=5 — reconcile ARCH-03. | architect | open |
| S403-O4 / S403-H1-DEFER / DRIFT-S4.03-001 | LOW/MED | S-4.03 DegradationEvent per-frame; PC-3 retransmit anchored S-BL.ARQ-TX; ADR-005 resync wire-mechanics anchored S-BL.NI. | product-owner/architect | anchored |
| S404-OBS-F / S404-LOW-1 | OBS/LOW | S-4.04 E-FWD-001 rate-limit LATENT; 3 LOW + NITPICK (SEC-001 CRC32 accepted). | architect/implementer | re-confirm on production wiring |
| S601-SEC-001..002 | LOW | S-6.01 CWE-117 sanitize --config; CWE-400 explicit slice cap. | implementer | deferred cycle-close |
| OBS-VP-BENCH | OBS | VP-041/VP-042 unverified pending S-BL.BENCH story. | orchestrator | deferred S-BL.BENCH |
| PROCESS-GAP-W4 | OBS | [process-gap] S-BL.NI wave must carry cross-component lock-ordering integration -race test. | orchestrator/architect | target S-BL.NI wave planning |
| F-009 | LOW | ARCH-INDEX input-hash tooling field-name mismatch. | architect/devops | deferred maintenance |
| E-CFG-002 / E-CFG-006 | MED | Pre-existing config-key collision (joined tracking). | product-owner | deferred maintenance |
| PROCESS-GAP-W5A | OBS | [process-gap] Two false-greens in Wave 5; candidate: require `just test-race` evidence-paste before green-claim. | orchestrator | open — candidate codification |
| DRIFT-SW501-NITPICK | LOW | S-W5.01 Pass-3 nitpicks (stale RED-GATE comments, dead `_ = pub`). | implementer | Wave-6 hygiene story |
| PROCESS-GAP-P21..P25 | OBS | [process-gap] Sibling-sweep gap crystallized; vsdd-factory #361–#364 filed. | orchestrator/story-writer | open — issues filed |
| S502-DEFER-1..2 | MED | S-5.02 runRouterStatus auth-timeout gap; writeSuccess os.Exit(3) outside main(). | implementer | defer wave-gate/phase-5 |
| S502-DEFER-4..6 | LOW | S-5.02 ARCH-11/dep-graph VP totals; §Arch Compliance asymmetric; token-budget footnote. | architect/story-writer | defer post-conv sweep |
| SW502-DEFER-1..8 | LOW | S-W5.02 CR-002/005-009 + SEC-001/002. Detail: `cycles/cycle-1/closed-drift.md`. | implementer/test-writer | deferred wave-6 / phase-5 |
| PROCESS-GAP-W5-SIBLINGSWEEP | LOW | [process-gap] Codify orchestrator-level upstream-rooted sibling-sweep at BC/VP bumps. | orchestrator | policy-registry-update |
| PROCESS-GAP-STORY-INDEX-SUMMARY-SWEEP | OBS | [process-gap] STORY-INDEX aggregate rollups must sweep atomically on section moves (F-P2L3-M1). | orchestrator/story-writer | codify |
| S-7.01 CR-001/004/005/006/007 | LOW/nit | Post-merge deferrals filed as switchboard-blue #44–#48. | implementer | issues filed |
| S-7.02 Pass-10 O-1/O-2/O-3/nit | LOW/nit | Advertise() no-validate; nameLen==0 asymmetry; uint16 truncation; HMAC-tag comment. Issues #49–#52. | implementer | issues filed |
| S-BL.ROUTER-ADDR L-1/L-2 | LOW | PathEntryFromSnapshot param redundancy; sbctl PathEntry drift. Issues #53–#54. | implementer | issues filed |
| PROCESS-GAP-POL-001-INDEX | OBS | [process-gap] POL-001 scope unclear for INDEX artifacts. vsdd-factory#407 filed. | orchestrator | codify |
| PROCESS-GAP-FORCE-PUSH | HIGH | [process-gap] pr-manager reached for rebase+force-push over gh pr update-branch. vsdd-factory#408 + switchboard-blue#57 filed. | orchestrator/pr-manager | playbook fix upstream |
| PROCESS-GAP-DEMO-TAPE-PATHS | OBS | [process-gap] demo-recorder emits `.tape` files with hardcoded absolute worktree paths; local fix applied (25 files, PR #59/cdb2b66); upstream drbothen/vsdd-factory#418 filed for template fix. | orchestrator/demo-recorder | upstream fix pending |
| WAVE-GATE-DISPATCH-INTEGRITY | HIGH | [process-gap] Perimeter-2 (wave-gate) adversary dispatch lacks HEAD-SHA verification tuple; adversary caught mismatch opportunistically; silent-false-green risk if less-thorough pass proceeds. drbothen/vsdd-factory issue drafted in .vsdd-factory-issues-pending.md. | orchestrator | target: pipeline-hardening cycle |
| DRIFT-POL003-GOV-LEAF-ENFORCE | LOW | [process-gap] No structural enforcement of `governance_leaf` annotation on BC changelog rows declaring "No behavioral changes"/"governance-only". Two occurrences (BC-2.07.001 v1.13, BC-2.08.001 v1.3) — pattern recurred. Suggest pre-commit or CI check. | orchestrator / spec-steward | open — file drbothen/vsdd-factory follow-on |
| DRIFT-POL003-NAMING | LOW | POL-003 Exception A annotation reference wording drift: BC-2.07.001 v1.13 cites `drbothen/vsdd-factory#429 draft policy`; BC-2.08.001 v1.3/v1.5 cite `POL-003 Exception A`. Substance identical, naming inconsistent. Converge on `POL-003 Exception A` for future rows. Deferred — not blocking wave-gate. | spec-steward | open |
| DRIFT-BC207-V113-BODY-CHANGELOG-MISMATCH | LOW | BC-2.07.001 v1.13 changelog description states `Stories row cite S-6.05 v1.5 → v1.7` but body Traceability Stories row (line 206) reads `S-6.05 v1.8`. STORY-INDEX row 3.60 shows a subsequent v1.7 → v1.8 bump on the same day. Body updated to v1.8 without accompanying changelog row. Per POL-003 Exception A the lag is permitted, but the self-inconsistency warrants a follow-up governance-only v1.14 changelog row to reconcile. Deferred — not blocking. | spec-steward | open |
| DRIFT-POL003-VP-FRONTMATTER-VERSION-PIN | LOW | [process-gap] VP frontmatter `source_bc:` shape asymmetry across VPs weakens POL-003 machine-checkability. VP-048 uses `BC-2.07.001 v1.12` (with version suffix); VP-050 uses `BC-2.08.001` (no suffix). Tools auditing "does the downstream cite the current BC version?" cannot mechanically answer for VPs whose `source_bc:` omits the version pin. Substance is anchored via Story Trace tables — no correctness defect — but shape drift weakens auditability. Per Cycle-Closing Checklist (S-7.02) process-gap findings require either a follow-up story or justified deferral. Deferral: filed as candidate refinement to drbothen/vsdd-factory POL-003 tooling — require uniform `source_bc: BC-N.NN.NNN v<M.N>` frontmatter shape on all VPs for machine-checkable governance. Not blocking BC-5.39.001 closure. | orchestrator / spec-steward | open — drbothen/vsdd-factory POL-003 tooling backlog |
| DRIFT-HS006-ROUTER-DAEMON-STUB | MEDIUM | 2026-07-02 | Router daemon subcommand `./bin/switchboard router` prints `runRouter: not implemented` and exits 1. Steps 9 & 10 of HS-006 (live PE-mode config reload; connected nodes migrate on drain) cannot be exercised through operator surface. Config-side of PE graduation fully verified; only daemon runtime is stubbed. Deferred to follow-on router-daemon-runtime story. |
| DRIFT-HS006-DRAIN-CLI-MISSING | LOW | 2026-07-02 | No `sbctl router drain` / `sbctl admin drain` subcommand. Drain only reachable via SIGTERM signal handling. Control and console daemons SIGTERM→clean exit in 4–32ms (well within 2s BC-2.09.002 budget). Operator-surface convenience gap, not a behavior gap. Deferred to future operator-UX story. |
| DRIFT-HS006-DRAIN-TIMEOUT-FORCED-EXIT-UNEVIDENCED | LOW | 2026-07-02 | Drain-timeout forced-exit-with-log clause (BC-2.09.002 evidence question) not observable through public API with router daemon stubbed. Evidence-gap, not necessarily behavior-gap — requires live router with connected nodes and induced hang. Re-evaluate when DRIFT-HS006-ROUTER-DAEMON-STUB is closed. |
| DRIFT-P5P1-A001-SVTN-LIST-ORPHAN | HIGH | 2026-07-02 | sbctl svtn list wire cmd svtn.list has zero daemon handler; contradicts BC-2.07.002 canonical test vector happy-path (`.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md:135`). Wave-6 merged with this gap; internal-code adversary chain missed it because gap is only visible cross-cutting sbctl main.go vs daemon Command: literals. Remediation: PENDING-<S-BL.SVTN-LIST-WIRE> annotation on BC canonical test vector (Task #77) + file backlog story (Task #78). | closed-by-annotation-2026-07-02 (BC-2.07.002 v1.6, Burst 8, HEAD 4659cb88) |
| DRIFT-P5P1-A002-SESSIONS-LIST-ORPHAN | HIGH | 2026-07-02 | sbctl sessions list wire cmd sessions.list has zero daemon handler; contradicts BC-2.03.002 PC-1 "core operator experience" claim (`.factory/specs/behavioral-contracts/ss-03/BC-2.03.002.md:50`). S-7.02 (merged PR #55) implements Discovery.Enumerate() internally but no wire boundary. S-BL.DISCOVERY-WIRE exists in backlog. Remediation: PENDING-S-BL.DISCOVERY-WIRE annotation on BC PC-1 (Task #77). | closed-by-annotation-2026-07-02 (BC-2.03.002 v1.4, Burst 8, HEAD 4659cb88) |
| DRIFT-P5P1-A003-PING-VERSION-ORPHAN | MEDIUM | 2026-07-02 | sbctl ping / sbctl version dispatch to wire commands ping / version with zero daemon handlers; return E-RPC-010 masking as "unknown command" from a live daemon. No BC anchor — CLI-declared promise only. Remediation: either register no-op handlers or trim sbctl subcommands; deferred to future operator-UX story. | closed-by-annotation-2026-07-02 (BC-2.07.002 v1.7 EC-004+EC-005 + S-BL.PING-VERSION-WIRE stub minted, Burst 11) |
| DRIFT-P5P1-B-H001-ENET006-TAXONOMY-ORPHAN | HIGH | 2026-07-02 | error-taxonomy.md:119 E-NET-006 declares operator-facing error message ("router draining; connect to alternate router at <alternates_list>") with zero emission site in cmd/ or internal/. BC-2.09.002 anchor. S-7.04 pending. Remediation: PENDING-S-7.04 annotation on E-NET-006 row mimicking E-CFG-002 line 99 "defensive:" shape (Task #77). Closes F-P5-Adv-B-L-001 (annotation-shape inconsistency). | closed-by-annotation-2026-07-02 (error-taxonomy v4.2, Burst 8, HEAD 4659cb88). F-P5-Adv-B-L-001 also closed as side-effect of this edit. |
| DRIFT-P5P1-B-M002-BC209003-DEFERRED-UNTRACKED | MEDIUM | 2026-07-02 | BC-2.09.003 PC-7/PC-8/PC-9 have DEFERRED-APPLICATION obligation tied to S-7.04 (status: pending) via S-7.04 AC-005/AC-006/AC-007. No mechanism ensures these become release-gate blockers if S-7.04 deprioritizes. This drift row itself provides tracking; deprioritize-alarm remains a follow-on process gap. |
| DRIFT-P5P1-B-M001-POL003-QUANTIFICATION | LOW | 2026-07-02 | Expansion of DRIFT-POL003-VP-FRONTMATTER-VERSION-PIN with quantification: 1/76 VPs (VP-048 only) carry source_bc version-pin suffix. POL-003 as written cannot be a lint gate — no canonical shape to check against. Task #72 (upstream drbothen/vsdd-factory filing) subsumes this. |
| DRIFT-P5P2-A003-TEST-HELPER-WIRE-TYPO | LOW | 2026-07-02 | `cmd/sbctl/e2e_helpers_test.go:191` registers mock for `admin.key.list` where shipped surface is `admin.key.list-keys`. One-line fix, deferred to future test-writer follow-up. Refs F-P5P2-A-003. Status: open. |
| DRIFT-P5P2-B-O003-ECFG-COLLISION-MAINTENANCE | LOW | 2026-07-02 | E-CFG-002 + E-CFG-006 codespace collisions across two BC-2.09.003 minor bumps acknowledged but no maintenance-pass story scheduled. Refs O-P5P2-B-003. Status: open, awaiting maintenance-pass story. |
| DRIFT-P5P3-A001..A009/B001..B003/B17 | HIGH..LOW | 2026-07-02 | ALL RESOLVED (Bursts 16–18): PR #62 c76a8d5 (code), taxonomy v4.3/v4.4 (spec). Detail: `cycles/cycle-1/closed-drift.md`. |
| DRIFT-P5P4-PROMPT-SHORTID | MED | 2026-07-03 | RESOLVED (Burst 19): PR #63 cbd0272. |
| DRIFT-P5P4-ADMINWIRE-EXTRACTION | LOW | 2026-07-03 | DEFERRED: inline wire arg structs; future maintenance cycle or Wave-7+. |
| DRIFT-P5P5-TEST-CITATION-VERSION-FLOOR | LOW | 2026-07-03 | [process-gap] No version-floor rule on test taxonomy citations. vsdd-factory issue pending. |
| DRIFT-P5P6-ANNOTATION-EXITCODE | MED | 2026-07-03 | RESOLVED (Burst 23): PR #65 4d7d9e0. |
| DRIFT-P5P7-O1-TARGET-EMPTY-TEST | LOW | 2026-07-03 | router status --target= (empty value) path converted but lacks dedicated test case; 3 fs.Parse paths likewise; PR #66 review O-1. Follow-on micro-addition to the production_exit_code_test.go table. |
| DRIFT-P5P7-O4-INTERACTIVE-CONFIRM-PARITY | LOW | 2026-07-03 | admin.go:395 interactive-confirm mismatch returns plain fmt.Errorf while --confirm sibling uses usageErrf; needs adjudication whether interactive-mismatch is usage-class (spec §129/§130) before converting; PR #66 review O-4. |
| DRIFT-P5P9-STALE-RECONCILIATION-COMMENT | LOW | 2026-07-03 | RESOLVED (Burst 31): PR #68 66e9ddc — stale comment fixed; U+2028 hexdump comment rider applied. |
| DRIFT-P5P14-B-001-VP-SOURCE-BC-VERSION-PIN | MED | 2026-07-03 | DEFERRED — POL-003 candidate (VP source_bc version-pin) not ratified in .factory/policies.yaml. Sweep scope: 77 VP frontmatters. Governance workstream, not in-cycle fix. Target release: post-POL-003 ratification. See P5-pass-14-Adv-B.md finding F-P5P14-B-001. |
| POL-006-DEFERRED-LINT | OBS | POL-006 reverse-trace class recurred in 5 consecutive Lane-B passes (P22-obs, P24×3, P25×1, P26×2). Human-driven remediation caught each instance but the pattern is inherently machine-checkable via ARCH-11↔VP-INDEX bidirectional lint. Burst 68b established clean baseline via proactive full-file 77-VP sweep; future recurrences should be caught by upstream drbothen/vsdd-factory tooling. Deferred to post-Phase-5 upstream issue filing (target: cycle-2 or vsdd-factory 1.0.0-rc.22+). Rationale: not blocking Phase 5 convergence; class now baselined; upstream tool ownership. | orchestrator | deferred-upstream |

Resolved items (Waves 1–5 + Tranche A + Pass 3 F1): `cycles/cycle-1/closed-drift.md` and `cycles/cycle-1/blocking-issues-resolved.md`.

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| Architecture (HMAC/FEC/LWW/HKDF) | ADR-001..004; ARCH-02/03/04 | 2026-06-23 |
| Waves 3–5 + Phase 4 gate | All APPROVED/CONVERGED; HS-006 PASS_AT_THRESHOLD 0.85 | 2026-06-27–07-02 |
| Wave 6 all tranches + wave-gate | 7 stories merged (PRs #40–#43,#55–#56,#60–#61); W-6 CONVERGED 3/3 | 2026-07-01–07-02 |
| Phase 5 Pass 3 REMEDIATION COMPLETE | PR #62 c76a8d5; taxonomy v4.4; 7 DRIFTs closed | 2026-07-02 |
| Phase 5 Pass 4 COMPLETE (BC-5.39.001) | PR #63 cbd0272; 9 findings; streak 3/3 (passes 17/18/19) | 2026-07-03 |
| Phase 5 Pass 5 HAS_FINDINGS | 0H/4M/3L/2obs; streak reset 0/3; remediation pending | 2026-07-03 |
| Phase 5 Pass 5 REMEDIATION COMPLETE | Burst 21: interface-definitions v1.18, S-BL.ADMIN-RECOVER-WIRE stub, PR #64 d012dbf; streak 0/3; Pass 6 next | 2026-07-03 |
| Phase 5 Pass 6 HAS_FINDINGS | Adv-A 1H/4M/1L (CLI dispatch layer); Adv-B CLEAN 0/0/0+2obs; streak holds 0/3; Burst 23 remediation pending | 2026-07-03 |
| Phase 5 Pass 6 REMEDIATION COMPLETE | Burst 23: PR #65 4d7d9e0 (exit-code discrimination, sessions dispatch, bare-sbctl exit 2); interface-definitions v1.19; BC-2.07.002 v1.9 (EC-003 0→2); S-6.03 v2.8 (AC-012); S-BL.CLI-SURFACE-COMPLETION stub; F-A-004 adjudicated spec-side; streak 0/3; Pass 7 next | 2026-07-03 |
| Phase 5 Pass 7 HAS_FINDINGS | Adv-A 0H/3M/0L (console/router usageErrf gap — same class as P6 A-001, Burst 23 sweep missed these trees); Adv-B CLEAN 0/0/0+5obs; streak holds 0/3; Burst 25 remediation pending (code-only) | 2026-07-03 |
| Phase 5 Pass 7 REMEDIATION COMPLETE | Burst 25: PR #66 b4ccd06 (usageErrf sweep: console.go ×7, router_metrics.go ×1, router_status.go ×2; production_exit_code_test.go +12 cases); completeness grep clean; streak 0/3; Pass 8 next | 2026-07-03 |
| Phase 5 Pass 8 HAS_FINDINGS | Adv-A 2H/4M/1L (admin-key surface: confirm-gate wrong-command prefix, §108 unreachable exit codes, --role silent default, destroy name-validation gap, §109 E-ADM-011 vs E-ADM-019, paths verb message, hex-pubkey placeholders); Adv-B 0H/2M+1obs (test misattribution + vacuous cmd-dispatch oracle); streak 0/3; Burst 27 remediation pending | 2026-07-03 |
| Phase 5 Pass 8 REMEDIATION COMPLETE | Burst 27: PR #67 32ea461 (code track: 6 findings resolved); interface-definitions v1.20 (spec track: §108/§109 error surfaces, openssh-pubkey, --role default, §395 sweep); streak 0/3; Pass 9 next | 2026-07-03 |
| Phase 5 Pass 9 HAS_FINDINGS | Adv-A 1H/2M/3L (all spec-side: §94-95 version/ping unannotated, --target default undocumented, §110 expire exit codes incomplete, §120 E-SVTN-003, §48 synopsis --timeout, §128 --yes footnote); Adv-B CLEAN 0/0/0+3obs; ZERO code defects this pass; streak 0/3; v1.21 spec-only remediation next | 2026-07-03 |
| Phase 5 Pass 9 REMEDIATION COMPLETE | Burst 29: interface-definitions v1.21 (spec-only — six Adv-A findings, all documentation-side); ARCH-INDEX v1.7; zero code changes; develop stays 32ea461; streak 0/3; Pass 10 next | 2026-07-03 |
| Phase 5 Pass 10 HAS_FINDINGS | Adv-A 1H/1M (§110 phantom --at flag survived 9 passes [column-scoped attention]; E-CFG-001 token fragmentation zero/negative vs >100y); Adv-B 1L+2obs (test name↔assertion inversion; NoArgs meta-word disjunct; U+2028 arm-pinning); streak 0/3; Burst 31 remediation pending | 2026-07-03 |
| Phase 5 Pass 10 REMEDIATION COMPLETE | Burst 31: PR #68 66e9ddc (code track: E-CFG-001 prefix zero/negative F-A-002, test rename F-B-001, NoArgs tightened OBS-B-001, U+2028 arm-pinning OBS-B-002, DRIFT-P5P9 comment resolved); interface-definitions v1.22 (spec track: §110 --at→--after F-A-001 HIGH adjudicated spec-side, E-CFG-001 exit-class split + §186); streak 0/3; Pass 11 next | 2026-07-03 |
| Phase 5 Pass 11 HAS_FINDINGS | Adv-A 1H/1M/3obs (§131/§137 revoke listed in runDestroyConfirmGate family but impl uses boolStringFlag — spec never received carve-out; §109 syntax missing required --role); Adv-B CLEAN 0/0/0+3obs; both adjudicated spec-side; streak 0/3; Burst 33 spec-only remediation pending | 2026-07-03 |
| Phase 5 Pass 11 REMEDIATION COMPLETE | Burst 33: interface-definitions v1.23 spec-only — §131 revoke carve-out from runDestroyConfirmGate family (taxonomy v4.4 + E-ADM-018 already ruled bool-confirm shape); §137 scoped to svtn destroy + key register + admin recover; §109 --role REQUIRED with no-default annotation; zero code changes; develop stays 66e9ddc; streak 0/3; Pass 12 next | 2026-07-03 |
| Phase 5 Pass 12 HAS_FINDINGS | Adv-A 0H/2M/2obs (§111 list-keys exit codes missing E-SVTN-003 + E-CFG-001; §108/§109/§110 --svtn <id> placeholder class — daemon is name-keyed); Adv-B CLEAN 0/0/0+3obs; third consecutive zero-code-defect pass; streak 0/3; Burst 35 spec-only remediation pending | 2026-07-03 |
| Phase 5 Pass 12 REMEDIATION COMPLETE | Burst 35: interface-definitions v1.24 spec-only — §111 exit-code column extended (E-SVTN-003 + E-CFG-001), --svtn <svtn-name> placeholder sweep (§108/§109/§110/§130), §108/§120 confirm-family flag consistency touch; zero code changes; develop stays 66e9ddc; streak 0/3; Pass 13 next |
| Phase 5 Pass 13 HAS_FINDINGS | Adv-A 1H/1M/2obs (list-keys admission gate removed with authority gate — CWE-862; E-CFG-001 token absent from list-keys usageErrf); Adv-B 0H/0M/1L/2obs (e2e stub name admin.key.list vs admin.key.list-keys); streak 0/3; Bursts 37+38 remediation |
| Phase 5 Pass 13 REMEDIATION COMPLETE | Burst 37: PR #69 03ce8e7 (admission gate restored; E-CFG-001 token; stub name fix). Burst 38: spec-only — interface-definitions v1.25 (§111 auth sharpened; BC-2.05.004 v1.13 PC-1 F-L2-003 + EC-008; VP-075 v1.7 scope exclusion + CWE-862); streak 0/3; Pass 14 next | 2026-07-03 |

Full decision detail: `cycles/cycle-1/burst-log.md`.

## Historical Content

Burst logs, adversary pass details, session checkpoints, and lessons
have been extracted to cycle files:

- Burst history: `cycles/cycle-1/burst-log.md`
- Convergence trajectory: `cycles/cycle-1/convergence-trajectory.md`
- Session checkpoints: `cycles/cycle-1/session-checkpoints.md`
- Lessons learned: `cycles/cycle-1/lessons.md`
- Resolved blockers: `cycles/cycle-1/blocking-issues-resolved.md`

## Session Resume Checkpoint

**Timestamp:** 2026-07-03T00:00:00Z
**Post-burst:** Burst 71a (Phase 5 Pass 27 Adv-A remediation)
**Pipeline state:** Phase 5 Pass 27 CONCLUDED HAS_FINDINGS; streak stays 0/3; Pass 28 dispatch next
**Factory HEAD:** this commit (Burst 71a — Adv-A remediation: sprint-state v1.53 + STATE.md narrative + Adv-A sidecar — signed)
**Develop HEAD:** 6deda15def9326f28e96f133e237aff5ecb74d7b (unchanged — no code changes this burst)

**Pass 27 deltas:** Adv-A (spec-completeness + traceability lens) HAS_FINDINGS — F-P5P27-A-001 HIGH POL-002 Wave-5 stories_merged missing S-W5.02 (d881f99 PR #38) + merge_shas gap; now 8/8 matches STORY-INDEX v3.79. SHIPPED Burst 71a (sprint-state v1.52→v1.53). F-P5P27-A-002 MED POL-002 STATE.md Current State stale Pass-12 narrative (cites 66e9ddc / 76 VPs / 53 stories); replaced with authoritative P26 status. SHIPPED Burst 71a. F-P5P27-A-003 MED POL-002 Wave-5 wave_adversary_converged false contradicts sibling gate_disposition CONVERGED + STATE.md wave_5_gate CONVERGED; flipped to true. SHIPPED Burst 71a (sprint-state v1.53). F-P5P27-A-004 LOW pending-intent: Wave-6 `tranche-c-closed` vs Wave-5 `closed` — adjudicated BY-DESIGN (Wave-6 was tranched A/B/C in sequence; Wave-5 was not; terminology reflects underlying lifecycle difference). NO CHANGE. Adv-B (verification-coverage + test-rigor lens) HAS_FINDINGS — F-P5P27-B-001 HIGH POL-008 ARCH-11 4 session-access rows Phase-column drift (BC-2.04.001/003/004/005); F-P5P27-B-002 LOW POL-008 BC-2.05.007 stale "+audit" method annotation; O-P5P27-B-001 full-file 12-dual-anchor-VP sweep confirmed clean baseline. Burst 71b handling separately (ARCH-11 scope). Streak stays 0/3.

**Sidecar paths:** `.factory/cycles/cycle-1/adversarial-reviews/P5-pass-27-Adv-A.md` / `P5-pass-27-Adv-B.md` (Adv-B pending Burst 71b)

**Phase 5 trajectory:** P1→P26 (see session-checkpoints.md) → P27: Adv-A 1H/2M/1L HAS_FINDINGS (A-001/002/003 SHIPPED Burst 71a; A-004 by-design) + Adv-B 1H/1L/1obs HAS_FINDINGS (B-001/002 Burst 71b pending) → streak stays 0/3
**Next action:** Burst 71b Adv-B remediation (ARCH-11 scope, disjoint). Then Burst 72 Pass 28 fresh-context split-adversary dispatch. Adjudicated-deferrals list expands: add F-P5P27-A-001/002/003 all SHIPPED + F-P5P27-A-004 by-design. Three clean passes still needed for BC-5.39.001 convergence. Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.
