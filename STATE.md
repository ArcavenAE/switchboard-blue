---
pipeline: IN_PROGRESS
phase: phase-5-adversarial-refinement
phase_step: phase-5-pass-5-remediation-complete
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
phase_5_pass_4_gate: BC_5_39_001_SATISFIED
develop_head: d012dbf
open_prs: 0
alpha_release_tag: alpha-20260629-165045-d854978
awaiting: phase-5-pass-6-dispatch
historical_cycles: []
timestamp: 2026-07-03T00:00:00Z
last_update: 2026-07-03
---

# Switchboard Factory State

## Current State

Phase 5 Pass 5 REMEDIATION COMPLETE. Burst 21: spec v1.18 + PR #64 d012dbf merged. BC-5.39.001 streak 0/3 — Pass 6 fresh-context dispatch next.
develop HEAD: d012dbf. 45 BCs, 76 VPs, 52 stories (backlog +1 S-BL.ADMIN-RECOVER-WIRE), 18 internal packages.

Sidecar reviews: `.factory/cycles/cycle-1/adversarial-reviews/W-6-wavegate-pass-{1-6}-Adv-{A,B}.md`.
Phase 4 report: `.factory/holdout-scenarios/evaluations/HS-006-evaluation-2026-07-02.md`.

## Phase Progress

| Phase | Status | Latest Gate |
|-------|--------|-------------|
| Phase 1 — Spec Crystallization | COMPLETE | approve-with-drift (2026-06-24) |
| Phase 2 — Story Decomposition | COMPLETE | approve-proceed-to-wave-1 (2026-06-24) |
| Phase 3 — TDD Implementation | COMPLETE | W6 CONVERGED 3/3 (2026-07-02); all waves merged |
| Phase 4 — Holdout Evaluation | COMPLETE | PASS_AT_THRESHOLD 0.85 (2026-07-02) |
| Phase 5 — Adversarial Refinement | PASS_5_REM_COMPLETE | P1: 3H/3M/1L → REM → P2: 0H/3M/2L → REM → P3: 3H/4M/2L+6obs → Path B rem spec+code → P4 COMPLETE (9 findings → 3/3 CLEAN streak) → P5: 0H/4M/3L+2obs → REM (Burst 21: spec v1.18 + PR #64 d012dbf) → P6 dispatch next |

Wave-by-wave detail: `cycles/cycle-1/burst-log.md` and `cycles/cycle-1/closed-stories.md`.

## Current Phase Steps

| Date | Step | Status | Result |
|------|------|--------|--------|
| 2026-07-02 | Phase 5 Pass 2 Adv-B dispatched (test-rigor + traceability lens, opus, ≤6min) | COMPLETED | HAS_FINDINGS 0H/1M/1L/4obs |
| 2026-07-02 | Phase 5 Pass 2 remediated — 2 BCs annotated, 2 backlog stubs minted, 1 DEFERRED row reconciled | COMPLETED | BC-2.07.002 v1.7 (EC-004+EC-005), BC-2.09.003 v1.8 (listen_addr row), S-BL.SVTN-LIST-WIRE + S-BL.PING-VERSION-WIRE stubs, HEAD dc51b06 → burst-12. Closes F-P5P2-A-001, A-002, B-002. Streak remains 0/3 — Pass 3 next. |
| 2026-07-02 | Phase 5 Pass 3 HAS_FINDINGS both lenses (fresh-context adversary rejects annotate-and-track for wire-orphans; 3 code-side canonical-message drift findings) | COMPLETED | Adv-A 3H/4M/2L/3obs, Adv-B 0H/1M/2L/3obs. 12 DRIFTs opened, streak reset 0/3. Awaiting human decision on wire-orphan register-vs-delete; code-side drift needs fix-burst. |
| 2026-07-02 | Phase 5 Pass 3 Path B remediation spec-side complete (Burst 16); code-side fix-PR merged PR #62 c76a8d5 (Burst 17) | COMPLETED | 5 spec files edited (BC-2.07.002 v1.8, error-taxonomy v4.3→v4.4, VP-043 v1.2, VP-062 v1.7, VP-INDEX v2.35) + BC-2.09.003 v1.9; 2 backlog stories retired wont-fix; 7 DRIFTs closed spec+code; taxonomy v4.4 corrects E-ADM-018 bool-flag form. Agents: product-owner + spec-steward + state-manager (Bursts 16+18) |
| 2026-07-02 | Phase 5 Pass 3 REMEDIATION COMPLETE — Pass 4 dispatch ready | COMPLETED | PR #62 c76a8d5 merged; taxonomy v4.4; all 6 code-side DRIFTs closed; develop_head c76a8d5; sprint-state streak 0 pending_pass 4 |
| 2026-07-03 | Phase 5 Pass 4 Burst 19 — wire-contract remediation (svtn_id wire field, OpenSSH pubkey, taxonomy drift, prompt short-id substitution, --confirm symmetry) | COMPLETED | PR #63 cbd0272 merged; 9 findings resolved (F-A-001..010); taxonomy v4.5; BC-5.39.001 streak 3/3 at passes 17/18/19 SATISFIED; DRIFT-P5P4-PROMPT-SHORTID RESOLVED; Pass 5 dispatch ready |
| 2026-07-03 | Phase 5 Pass 5 split-adversary — Adv-A (public-surface/operator-UX) + Adv-B (test-rigor/traceability) | COMPLETED | HAS_FINDINGS: Adv-A 0H/2M/2L/1obs (F-P5P5-A-001..004 + OBS-P5P5-A-001); Adv-B 0H/2M/1L/1obs (F-P5P5-B-001..003 + OBS-1); streak reset 0/3; Adv-B files_read 7 vs read_cap 6 (overage self-disclosed). Burst 21 remediation pending. |
| 2026-07-03 | Phase 5 Pass 5 remediation — Burst 21 (Track 1: product-owner interface-definitions v1.18; Track 1b: story-writer S-BL.ADMIN-RECOVER-WIRE stub; Track 2: test-writer + pr-manager PR #64 merged d012dbf) | COMPLETED | 4 A-findings remediated (F-P5P5-A-001..004); 3 B-findings remediated in PR #64 (F-P5P5-B-001..003); interface-definitions v1.18; S-BL.ADMIN-RECOVER-WIRE v1.0 stub minted; STORY-INDEX v3.70 (backlog 8→9, total 51→52); adjudication: F-P5P5-A-002 annotate-and-defer; tw citations unchanged (historical provenance); stub records 2 open design obligations. Streak 0/3. Pass 6 next. |

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
| DRIFT-P5P3-A001-SVTN-LIST-WIRE-ORPHAN | HIGH | 2026-07-02 | RESOLVED spec+code (Burst 16+17): BC-2.07.002 v1.8 removes svtn list row; case-arm deleted PR #62 c76a8d5. Refs F-P5P3-A-001. |
| DRIFT-P5P3-A002-PING-VERSION-WIRE-ORPHAN | HIGH | 2026-07-02 | RESOLVED spec+code (Burst 16+17): BC-2.07.002 v1.8 removes EC-004/EC-005 rows; version/ping case-arms deleted PR #62 c76a8d5. Refs F-P5P3-A-002. |
| DRIFT-P5P3-A003-EADM018-CODE-DRIFT | HIGH | 2026-07-02 | RESOLVED code+spec (Burst 17+18): admin_handlers.go:413 corrected; taxonomy v4.4 corrects canonical to `"use --confirm to proceed"` (bool-flag form). PR #62 c76a8d5. Refs F-P5P3-A-003. |
| DRIFT-P5P3-A004-SBCTL-SVTN-SILENT-DISCARD | MED | 2026-07-02 | RESOLVED code (Burst 17): sbctl svtn trailing-arg discard fixed. PR #62 c76a8d5. Refs F-P5P3-A-004. |
| DRIFT-P5P3-A005-EINT999-CODE-DRIFT | MED | 2026-07-02 | RESOLVED code (Burst 17): admin_handlers.go:428 corrected to canonical message. PR #62 c76a8d5. Refs F-P5P3-A-005. |
| DRIFT-P5P3-A006-EADM011-CODE-DRIFT | MED | 2026-07-02 | RESOLVED code (Burst 17): admin_handlers.go:419 E-ADM-011 V2 discriminators restored. PR #62 c76a8d5. Refs F-P5P3-A-006. |
| DRIFT-P5P3-A007-ECFG-COLLISION | MED | 2026-07-02 | RESOLVED spec-side (Burst 16): error-taxonomy v4.3 reconciles E-CFG-002 (private-key-export → E-CFG-011) and E-CFG-006 (sbctl --yes → E-CFG-012). BC-2.09.003 v1.9 removes collision-flag row. Emission-site updates (E-CFG-011/012) pending Burst 17 feature branch. Refs F-P5P3-A-007. |
| DRIFT-P5P3-A008-EC004-NOT-SHIPPING | LOW | 2026-07-02 | RESOLVED spec-side (Burst 16): BC-2.07.002 v1.8 removes EC-004 row entirely (surface withdrawn, not annotated). Refs F-P5P3-A-008. |
| DRIFT-P5P3-A009-UNKNOWN-SUBCOMMAND-NO-HINT | LOW | 2026-07-02 | RESOLVED code (Burst 17): sbctl unknown-subcommand hint added. PR #62 c76a8d5. Refs F-P5P3-A-009. |
| DRIFT-P5P3-B001-VP043-METHOD-BUCKET-MISLABEL | MED | 2026-07-02 | RESOLVED spec-side (Burst 16): VP-043 v1.2 reclassifies proof_method proptest → strong-oracle; removes gopter harness skeleton. VP-INDEX v2.35 reclassifies row 69 Proptest→Unit; arithmetic 33+4+22+10+2+2+3=76. Refs F-P5P3-B-001. |
| DRIFT-P5P3-B002-VP043-POL003-PIN-MISSING | LOW | 2026-07-02 | RESOLVED spec-side (Burst 16): VP-043 v1.2 frontmatter adds source_bc: BC-2.02.007 v1.3 pin. POL-003 conformance 2/76 → 3/76. Refs F-P5P3-B-002. |
| DRIFT-P5P3-B003-VP062-POL003-PIN-MISSING | LOW | 2026-07-02 | RESOLVED spec-side (Burst 16): VP-062 v1.7 frontmatter adds source_bc: BC-2.06.003 v1.13 pin. Refs F-P5P3-B-003. |

| DRIFT-P5P3-B17-CASE-ARM-DELETION | HIGH | 2026-07-02 | RESOLVED code (Burst 17): case "svtn" / "version" / "ping" arms deleted from cmd/sbctl/main.go. PR #62 c76a8d5. Refs F-P5P3-A-001, F-P5P3-A-002. |
| DRIFT-P5P4-PROMPT-SHORTID | MED | 2026-07-03 | RESOLVED code (Burst 19): Interactive `sbctl admin svtn destroy` prompt now substitutes actual SVTN short-id from operation context; literal `<short-id>` placeholder retired. PR #63 cbd0272. Refs F-A-009. interface-definitions.md §125/§129 interim-rendering annotation now superseded. |
| DRIFT-P5P4-ADMINWIRE-EXTRACTION | LOW | 2026-07-03 | DEFERRED: Wire arg struct types (`KeyRegisterArgs`, `KeyRevokeArgs`, `SVTNDestroyArgs`) currently defined inline in `cmd/switchboard/admin_handlers.go`. Both sbctl-side and switchboard-side tests cross-assert the wire contract (see `admin_handlers_wire_shared_pkg_test.go` + `admin_test.go:2093`). A future refactor may extract to `internal/adminwire` shared package — see code comment in `admin_handlers_wire_shared_pkg_test.go:7,33`. No behavior gap; deferred to maintenance cycle or Wave-7+. |
| DRIFT-P5P5-TEST-CITATION-VERSION-FLOOR | LOW | 2026-07-03 | [process-gap] No rule enforces that tests asserting a code minted in taxonomy vX.Y cite ≥ vX.Y. Source: F-P5P5-B-001 (E-CFG-013 anchored to v4.4 in test docstrings, but E-CFG-013 was minted in v4.6 — historically impossible citation). Deferred to upstream: vsdd-factory issue draft pending (Batch 30 tracker). Target: next maintenance cycle. |

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
**Post-burst:** Burst 21 (Track 1: product-owner interface-definitions v1.18; Track 1b: story-writer S-BL.ADMIN-RECOVER-WIRE stub; Track 2: test-writer + pr-manager PR #64 merged d012dbf)
**Pipeline state:** Phase 5 Pass 5 REMEDIATION COMPLETE; streak 0/3; Pass 6 fresh-context dispatch next
**Factory HEAD:** (see `git -C .factory log -1 --format='%h %s'`)
**Develop HEAD:** d012dbf (PR #64 squash-merge)

**Burst 21 deltas:**
- Track 1 (product-owner): interface-definitions v1.17→v1.18 — F-P5P5-A-001 (§116 authority→bootstrap-only), F-P5P5-A-002 (§119-125 PENDING-S-BL.ADMIN-RECOVER-WIRE annotation), F-P5P5-A-003 (§116/§117 exit-code enumeration E-CFG-001/E-INT-001), F-P5P5-A-004 (§59 alias REMOVED record)
- Track 1b (story-writer): S-BL.ADMIN-RECOVER-WIRE v1.0 stub minted; STORY-INDEX v3.69→v3.70 (backlog 8→9, total 51→52); adjudication annotate-and-defer; stub records 2 open design obligations (recovery semantics; --svtn id-vs-name ambiguity)
- Track 2 (test-writer + pr-manager): PR #64 squash-merged d012dbf (commits fa824c6/a1e1466/f638032) — wire-tag guards, version stamps, GREEN docstrings; CI all green; pr-reviewer APPROVED; LOW-5 fixed in f638032; NIT-6 waived; F-P5P5-B-001..003 resolved
- Adjudications: (a) F-P5P5-A-002 annotate-and-defer consistent with five prior wire deferrals; (b) tw "v1.1 §" citations in admin_test.go lines 1642/1834/1855/2433/2477/2522 left unchanged — historical provenance comments, not assertion anchors; documented in PR #64 body; (c) stub records two open design obligations
- BC-5.39.001 streak: 0/3 (Pass 6 is the next fresh-context attempt)

**Phase 5 trajectory:** P1 (3H/3M/1L → REM) → P2 (0H/3M/2L → REM) → P3 (3H/4M/2L+6obs → Path B rem spec+code) → P4 COMPLETE (9 findings → 3/3 CLEAN streak) → P5 (0H/4M/3L+2obs → REM (Burst 21: spec v1.18 + PR #64 d012dbf))

**Next action:** Phase 5 Pass 6 — fresh-context split-adversary dispatch (streak 0/3; target 0→1). Auto Mode: active (Path B). Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.
