---
pipeline: IN_PROGRESS
phase: phase-5-adversarial-refinement
phase_step: pending-dispatch
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
l4_vp_count: 76
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
develop_head: 7fe3e29
open_prs: 0
alpha_release_tag: alpha-20260629-165045-d854978
historical_cycles: []
timestamp: 2026-07-02T00:00:00Z
last_update: 2026-07-02
---

# Switchboard Factory State

## Current State

Wave 6 CONVERGED (3/3 clean wave-gate passes). Phase 4 HS-006 holdout: PASS_AT_THRESHOLD (0.85).
develop HEAD: 7fe3e29. 45 BCs, 76 VPs, 49 stories, 18 internal packages.
Wave-6 stories merged: S-BL.LOOKUP, S-W5.04, S-6.07 (Tr-A); S-7.01, S-7.02, S-BL.ROUTER-ADDR (Tr-B); S-7.03, S-6.05 (Tr-C).

Sidecar reviews: `.factory/cycles/cycle-1/adversarial-reviews/W-6-wavegate-pass-{1-6}-Adv-{A,B}.md`.
Phase 4 report: `.factory/holdout-scenarios/evaluations/HS-006-evaluation-2026-07-02.md`.

## Phase Progress

| Phase | Status | Latest Gate |
|-------|--------|-------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift (2026-06-24) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 (2026-06-24) |
| Phase 3 — TDD Implementation | COMPLETE | W6 CONVERGED 3/3 (2026-07-02); all waves merged |
| Phase 4 — Holdout Evaluation | COMPLETE | PASS_AT_THRESHOLD 0.85 (2026-07-02) |
| Phase 5 — Adversarial Refinement | PASS_3_REMEDIATION_IN_PROGRESS_SPEC_LANDED_CODE_PENDING | P1: 3H/3M/1L → REM → P2: 0H/3M/2L → REM → P3: 3H/4M/2L/6obs (Adv-A: 3H/4M/2L/3obs code-drift+wire-orphans; Adv-B: 0H/1M/2L/3obs VP-043 method drift+POL-003 pins). Streak 0/3. → Path B spec-side landed Burst 16 (5 spec files + 2 stories retired) |

Wave-by-wave detail: `cycles/cycle-1/burst-log.md` and `cycles/cycle-1/closed-stories.md`.

## Current Phase Steps

| Date | Step | Status | Result |
|------|------|--------|--------|
| 2026-07-02 | Burst 8 product-owner annotate BC-2.07.002/BC-2.03.002/error-taxonomy E-NET-006 | COMPLETED | HEAD 4659cb88; BC-2.07.002 v1.6, BC-2.03.002 v1.4, error-taxonomy v4.2 |
| 2026-07-02 | Phase 5 Pass 1 remediation applied — 4 findings closed by annotation | COMPLETED | Closes F-P5P1-A-001, F-P5P1-A-002, F-P5-Adv-B-H-001, F-P5-Adv-B-L-001. Streak 0/3 — Pass 2 pending. |
| 2026-07-02 | Phase 5 Pass 2 Adv-A dispatched (public-surface lens, opus, ≤6min) | COMPLETED | HAS_FINDINGS 0H/2M/1L/3obs |
| 2026-07-02 | Phase 5 Pass 2 Adv-B dispatched (test-rigor + traceability lens, opus, ≤6min) | COMPLETED | HAS_FINDINGS 0H/1M/1L/4obs |
| 2026-07-02 | Phase 5 Pass 2 remediated — 2 BCs annotated, 2 backlog stubs minted, 1 DEFERRED row reconciled | COMPLETED | BC-2.07.002 v1.7 (EC-004+EC-005), BC-2.09.003 v1.8 (listen_addr row), S-BL.SVTN-LIST-WIRE + S-BL.PING-VERSION-WIRE stubs, HEAD dc51b06 → burst-12. Closes F-P5P2-A-001, A-002, B-002. Streak remains 0/3 — Pass 3 next. |
| 2026-07-02 | Phase 5 Pass 3 HAS_FINDINGS both lenses (fresh-context adversary rejects annotate-and-track for wire-orphans; 3 code-side canonical-message drift findings) | COMPLETED | Adv-A 3H/4M/2L/3obs, Adv-B 0H/1M/2L/3obs. 12 DRIFTs opened, streak reset 0/3. Awaiting human decision on wire-orphan register-vs-delete; code-side drift needs fix-burst. |
| 2026-07-02 | Phase 5 Pass 3 Path B remediation spec-side complete (Burst 16); code-side fix-PR pending (Burst 17 feature branch off develop) | COMPLETED | 5 spec files edited (BC-2.07.002 v1.8, error-taxonomy v4.3, VP-043 v1.2, VP-062 v1.7, VP-INDEX v2.35) + BC-2.09.003 v1.9 (collision-flag cleanup); 2 backlog stories retired wont-fix; 7 DRIFTs closed spec-side; 6 DRIFTs remain code-side (5 message-drift/UX + 1 case-arm-deletion). POL-003 conformance 2/76 → 3/76. | Agents: product-owner + spec-steward + state-manager |

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
| DRIFT-P5P3-A001-SVTN-LIST-WIRE-ORPHAN | HIGH | 2026-07-02 | RESOLVED spec-side (Burst 16): BC-2.07.002 v1.8 removes svtn list canonical row; S-BL.SVTN-LIST-WIRE retired won't-fix. Code-side case-arm deletion pending Burst 17 feature branch. Refs F-P5P3-A-001. |
| DRIFT-P5P3-A002-PING-VERSION-WIRE-ORPHAN | HIGH | 2026-07-02 | RESOLVED spec-side (Burst 16): BC-2.07.002 v1.8 removes EC-004 sbctl version + EC-005 sbctl ping rows; S-BL.PING-VERSION-WIRE retired won't-fix. Code-side case-arm deletion pending Burst 17 feature branch. Refs F-P5P3-A-002. |
| DRIFT-P5P3-A003-EADM018-CODE-DRIFT | HIGH | 2026-07-02 | admin_handlers.go:413 emits `"pass --confirm"` where taxonomy canonical is `"use --confirm=<svtn-id> to proceed"`. Code-fix required. Refs F-P5P3-A-003. |
| DRIFT-P5P3-A004-SBCTL-SVTN-SILENT-DISCARD | MED | 2026-07-02 | sbctl svtn silently discards trailing args and dispatches to svtn.list; `sbctl svtn destroy foo` becomes a stealth list. Code-fix required. Refs F-P5P3-A-004. |
| DRIFT-P5P3-A005-EINT999-CODE-DRIFT | MED | 2026-07-02 | admin_handlers.go:428 emits "unmapped admin error" vs canonical "unmapped internal condition, programmer error, please report". Code-fix required. Refs F-P5P3-A-005. |
| DRIFT-P5P3-A006-EADM011-CODE-DRIFT | MED | 2026-07-02 | admin_handlers.go:419 drops role + svtn_name discriminators in E-ADM-011 Variant 2. Code-fix required. Refs F-P5P3-A-006. |
| DRIFT-P5P3-A007-ECFG-COLLISION | MED | 2026-07-02 | RESOLVED spec-side (Burst 16): error-taxonomy v4.3 reconciles E-CFG-002 (private-key-export → E-CFG-011) and E-CFG-006 (sbctl --yes → E-CFG-012). BC-2.09.003 v1.9 removes collision-flag row. Emission-site updates (E-CFG-011/012) pending Burst 17 feature branch. Refs F-P5P3-A-007. |
| DRIFT-P5P3-A008-EC004-NOT-SHIPPING | LOW | 2026-07-02 | RESOLVED spec-side (Burst 16): BC-2.07.002 v1.8 removes EC-004 row entirely (surface withdrawn, not annotated). Refs F-P5P3-A-008. |
| DRIFT-P5P3-A009-UNKNOWN-SUBCOMMAND-NO-HINT | LOW | 2026-07-02 | sbctl unknown-subcommand error gives no discovery path ("run 'sbctl' for usage" missing). Trivial code-fix. Refs F-P5P3-A-009. |
| DRIFT-P5P3-B001-VP043-METHOD-BUCKET-MISLABEL | MED | 2026-07-02 | RESOLVED spec-side (Burst 16): VP-043 v1.2 reclassifies proof_method proptest → strong-oracle; removes gopter harness skeleton. VP-INDEX v2.35 reclassifies row 69 Proptest→Unit; arithmetic 33+4+22+10+2+2+3=76. Refs F-P5P3-B-001. |
| DRIFT-P5P3-B002-VP043-POL003-PIN-MISSING | LOW | 2026-07-02 | RESOLVED spec-side (Burst 16): VP-043 v1.2 frontmatter adds source_bc: BC-2.02.007 v1.3 pin. POL-003 conformance 2/76 → 3/76. Refs F-P5P3-B-002. |
| DRIFT-P5P3-B003-VP062-POL003-PIN-MISSING | LOW | 2026-07-02 | RESOLVED spec-side (Burst 16): VP-062 v1.7 frontmatter adds source_bc: BC-2.06.003 v1.13 pin. Refs F-P5P3-B-003. |

| DRIFT-P5P3-B17-CASE-ARM-DELETION | HIGH | 2026-07-02 | code-fix-required: delete case "svtn" / "version" / "ping" case-arms from cmd/sbctl/main.go per BC-2.07.002 v1.8 (surface withdrawn from spec). Feature branch off develop; fix-PR via Burst 17. Refs F-P5P3-A-001, F-P5P3-A-002. |

Resolved items (Waves 1–5 + Tranche A + Pass 3 F1): `cycles/cycle-1/closed-drift.md` and `cycles/cycle-1/blocking-issues-resolved.md`.

## Decisions Log

| Decision | Outcome | Date |
|----------|---------|------|
| HMAC/FEC/LWW/key-permissions/HKDF architecture | ADR-001..004; ARCH-02/03/04 | 2026-06-23 |
| Wave 3 gate APPROVED | 3/3 adversary clean; 5 deferrals carried | 2026-06-27 |
| Wave 4 gate APPROVED | 6/6 diverse-lens passes; audit CONDITIONAL PASS | 2026-06-28 |
| Wave 5 CONVERGED (8 stories + hygiene) | PRs #30–#38 merged; 45 BCs, 76 VPs | 2026-06-30 |
| Wave-6 scope decided | 7 stories, 33 pt; 2 tranches | 2026-06-30 |
| Wave 6 Tranche A CLOSED | S-BL.LOOKUP #40, S-W5.04 #41, S-6.07 #42 | 2026-07-01 |
| Wave 6 Tranche B CLOSED | S-7.01 #43, S-7.02 #55, S-BL.ROUTER-ADDR #56 all merged with BC-5.39.001 3/3 | 2026-07-01 |
| Force-push introspection | vsdd-factory#408 + switchboard-blue#57 filed; `gh pr update-branch` adopted as standard | 2026-07-01 |
| Wave 6 Tranche B wave-level CONVERGED | 3/3 clean fresh-context passes (P2/P3/P4); FEC hygiene PR #58 merged; demo-tape-paths PR #59 merged; develop@cdb2b66 | 2026-07-01 |
| Wave 6 Tranche C CLOSED + wave-level CONVERGED | S-7.03 PR#60/7142146 + S-6.05 PR#61/7fe3e29 merged; per-story 3/3 each; W-6.C wave-level CONVERGED 3/3 | 2026-07-02 |
| W-6 combined wave-gate CONVERGED (BC-5.39.001) | 6 passes; streak 3/3 (Pass 4/5/6 clean post-F1 remediation); Task #22 CLOSED | 2026-07-02 |
| Phase 4 HS-006 PASS_AT_THRESHOLD | Satisfaction 0.85 (at threshold); Task #71 CLOSED | 2026-07-02 |

Per-pass wave-gate detail: `cycles/cycle-1/burst-log.md`.

## Historical Content

Burst logs, adversary pass details, session checkpoints, and lessons
have been extracted to cycle files:

- Burst history: `cycles/cycle-1/burst-log.md`
- Convergence trajectory: `cycles/cycle-1/convergence-trajectory.md`
- Session checkpoints: `cycles/cycle-1/session-checkpoints.md`
- Lessons learned: `cycles/cycle-1/lessons.md`
- Resolved blockers: `cycles/cycle-1/blocking-issues-resolved.md`

## Session Resume Checkpoint

**Timestamp:** 2026-07-02T00:00:00Z
**Post-burst:** Burst 16 (state-manager: Pass 3 Path B spec-side commit + backlog retire + DRIFT closure)
**Pipeline state:** Phase 5 Pass 3 spec-side remediation landed; code-side fix-PR pending
**Factory HEAD:** (see `git -C .factory log -1 --format='%h %s'`)
**Develop HEAD:** 7fe3e29e4358df16e4e2f1de65a4e0d972540b4a (unchanged)

**Spec-side deltas (Burst 15 + 16):**
- BC-2.07.002 v1.7 → v1.8: EC-004, EC-005, sbctl svtn list canonical row removed
- error-taxonomy v4.2 → v4.3: E-CFG-002/006 collisions reconciled onto E-CFG-011/012
- VP-043 v1.1 → v1.2: proof_method proptest → strong-oracle (matches shipped LCG test); gopter harness skeleton removed; source_bc BC-2.02.007 v1.3 pin
- VP-062 v1.6 → v1.7: source_bc BC-2.06.003 v1.13 pin
- VP-INDEX v2.34 → v2.35: row 69 Proptest→Unit reclass; POL-003 count 2/76→3/76; arithmetic 33+4+22+10+2+2+3=76
- BC-2.09.003 v1.8 → v1.9: collision-flag annotation row removed (reconciled by error-taxonomy v4.3)
- S-BL.SVTN-LIST-WIRE + S-BL.PING-VERSION-WIRE → wont-fix (v1.1)

**DRIFTs remaining code-side (6):**
- DRIFT-P5P3-B17-CASE-ARM-DELETION: cmd/sbctl/main.go svtn/version/ping case-arms (Burst 17)
- DRIFT-P5P3-A003-EADM018-CODE-DRIFT: admin_handlers.go:413 canonical message drift
- DRIFT-P5P3-A005-EINT999-CODE-DRIFT: admin_handlers.go:428 canonical message drift
- DRIFT-P5P3-A006-EADM011-CODE-DRIFT: admin_handlers.go:419 V2 discriminators drop
- DRIFT-P5P3-A004-SBCTL-SVTN-SILENT-DISCARD: sbctl svtn silent-discard (main.go)
- DRIFT-P5P3-A009-UNKNOWN-SUBCOMMAND-NO-HINT: sbctl unknown-subcommand no-hint (main.go)

**Next action:** Burst 17 — open feature branch off develop tip 7fe3e29e for code-side fix-PR:
1. Delete case "svtn" / "version" / "ping" arms from cmd/sbctl/main.go
2. Fix 3 canonical-message emission sites in admin_handlers.go (typed-error emitters)
3. Fix sbctl unknown-subcommand hint
4. Fix sbctl svtn arg-discard
5. Emission-site updates for E-CFG-011 + E-CFG-012 to sync with taxonomy v4.3

Pipeline via test-writer → implementer → pr-manager. After merge, Burst 18 Pass 4 fresh-context split-adversary.

**Auto Mode:** active (Path B, human approved).

Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.
