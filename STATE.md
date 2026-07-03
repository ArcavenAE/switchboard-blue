---
pipeline: IN_PROGRESS
phase: phase-5-adversarial-refinement
phase_step: phase-5-pass-10-remediation-complete
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
develop_head: 66e9ddc
open_prs: 0
alpha_release_tag: alpha-20260629-165045-d854978
awaiting: phase-5-pass-11-dispatch
historical_cycles: []
timestamp: 2026-07-03T00:00:00Z
last_update: 2026-07-03
---

# Switchboard Factory State

## Current State

Phase 5 Pass 10 split-adversary COMPLETE (Burst 30). Adv-A HAS_FINDINGS 1H/1M: F-P5P10-A-001 [HIGH] §110 documents --at RFC3339-timestamp flag that does not exist in impl (impl: --after duration; no RFC3339 parsing); F-P5P10-A-002 [MED] E-CFG-001 token fragmentation — zero/negative duration branch emits plain prose via usageErrf (exit 2, no E-CFG-001 token) while >100 years branch emits E-CFG-001 via daemon (exit 1). Adv-B HAS_FINDINGS 0H/0M/1L+2obs: F-P5P10-B-001 [LOW] test name BoolFlagRejectsNonBoolValue contradicts body (body verifies acceptance, not rejection); OBS-B-001 NoArgs oracle meta-word disjunct; OBS-B-002 U+2028 arm-pinning. BC-5.39.001 streak 0/3. Burst 31: small code track (E-CFG-001 prefix + test fixes + DRIFT-P5P9 comment rider) + spec track (§110 --at→--after adjudication per F-A-004 precedent).
develop HEAD: 32ea461. 45 BCs, 76 VPs, 53 stories (backlog +1 S-BL.CLI-SURFACE-COMPLETION), 18 internal packages.

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
| Phase 5 — Adversarial Refinement | PASS_10_REMEDIATION_COMPLETE | P1: 3H/3M/1L → REM → P2: 0H/3M/2L → REM → P3: 3H/4M/2L+6obs → Path B rem spec+code → P4 COMPLETE (9 findings → 3/3 CLEAN streak) → P5: 0H/4M/3L+2obs → REM (Burst 21: spec v1.18 + PR #64 d012dbf) → P6: Adv-A 1H/4M/1L + Adv-B CLEAN(2obs) → REM (Burst 23: PR #65 4d7d9e0 + v1.19/BC v1.9/S-6.03 v2.8) → P7: Adv-A 0H/3M/0L + Adv-B CLEAN(5obs) → REM (Burst 25: PR #66 b4ccd06, usageErrf sweep complete) → P8: Adv-A 2H/4M/1L + Adv-B 0H/2M+1obs → REM (Burst 27: PR #67 32ea461 + v1.20) → P9: Adv-A 1H/2M/3L (all spec-side) + Adv-B CLEAN(3obs) → REM (Burst 29: v1.21 spec-only) → P10: Adv-A 1H/1M + Adv-B 1L(2obs) → REM (Burst 31: PR #68 66e9ddc + v1.22) → P11 dispatch next |

Wave-by-wave detail: `cycles/cycle-1/burst-log.md` and `cycles/cycle-1/closed-stories.md`.

## Current Phase Steps

| Date | Step | Status | Result |
|------|------|--------|--------|
| 2026-07-02 | Phase 5 Passes 2+3 (adversary + remediation) | COMPLETED | P2: HAS_FINDINGS 0H/1M/1L; REM: BC-2.07.002 v1.7, BC-2.09.003 v1.8, stubs SVTN-LIST-WIRE+PING-VERSION-WIRE. P3: 3H/4M/2L+6obs; REM (Bursts 16-18): PR #62 c76a8d5, taxonomy v4.4, 7 DRIFTs closed. |
| 2026-07-03 | Phase 5 Pass 4 (Burst 19) + Pass 5 (Burst 20+21) | COMPLETED | P4 REM: PR #63 cbd0272, 9 findings, taxonomy v4.5, streak 3/3 SATISFIED then reset. P5: Adv-A 0H/2M/2L+Adv-B 0H/2M/1L → REM (Burst 21: v1.18, S-BL.ADMIN-RECOVER-WIRE, PR #64 d012dbf). Streak 0/3. |
| 2026-07-03 | Phase 5 Pass 6 split-adversary vs d012dbf + interface-definitions v1.18 | COMPLETED | Adv-A HAS_FINDINGS 1H/4M/1L (F-P5P6-A-001..006); Adv-B CLEAN 0/0/0+2obs. Burst 23 remediation pending. |
| 2026-07-03 | Phase 5 Pass 6 remediation (Burst 23) | COMPLETED | Code track: PR #65 4d7d9e0 merged (usageError exit-code discrimination, sessions sub-verb routing, bare-sbctl exit 2; RED 8692237 → GREEN e83c69e → triage 4540180; reviewer APPROVED 6 LOW, 4 applied 2 deferred). Spec track: interface-definitions v1.19 + BC-2.07.002 v1.9 (EC-003 0→2) + S-6.03 v2.8 (AC-012) — all claims file:line-verified against merged tree. Stub S-BL.CLI-SURFACE-COMPLETION minted (paths ping + svtn status flagged NO-GOVERNING-BC design obligations). F-A-004 adjudicated spec-side (S-7.03 converged shape authoritative). |
| 2026-07-03 | Phase 5 Pass 7 split-adversary vs 4d7d9e0 + v1.19 | COMPLETED | Adv-A HAS_FINDINGS 0H/3M/0L+1obs (F-P5P7-A-001..003 — console/router usage errors still plain fmt.Errorf → exit 1; same class as F-P5P6-A-001, console/router trees missed by Burst 23 sweep; OBS-A-001: production_exit_code_test.go covers zero console/router cases — RED enumeration was the effective contract). Adv-B CLEAN 0/0/0+5obs (exit-code discriminator coverage adequate on covered branches; OBS-B-003 stale docstring, OBS-B-004 comment drift, others cosmetic). Adv-A read overage 8/6 self-disclosed. Burst 25 remediation pending (code-only, no spec changes — §174 correct, impl stale). |
| 2026-07-03 | Phase 5 Pass 7 remediation (Burst 25, code-only) | COMPLETED | PR #66 b4ccd06 merged: 10 usage-error sites converted to usageErrf (console.go ×7, router_metrics.go ×1, router_status.go ×2); production_exit_code_test.go table extended to 12 cases (6 console/router RED-first); completeness grep audit: no residual usage-error-class fmt.Errorf in cmd/sbctl. Reviewer: no blockers, MINOR count-cosmetic + 2 follow-ons. OBS-B-003/004 comment fixes included. |
| 2026-07-03 | Phase 5 Pass 8 split-adversary vs b4ccd06 + v1.19 | COMPLETED | Adv-A HAS_FINDINGS 2H/4M/1L (F-P5P8-A-001..007 — admin-key surface: confirm-gate wrong-command prefix, §108 unreachable error codes, --role silent default, destroy name-validation gap, §109 E-ADM-011 vs E-ADM-019, paths verb message, hex-pubkey placeholders); Adv-B HAS_FINDINGS 0H/2M+1obs (F-P5P8-B-001 finding-ID misattribution in test failure arm [process-gap], F-P5P8-B-002 canned-daemon cmd-dispatch oracle vacuous [process-gap]). Both read-cap overages self-disclosed (A: 7 reads, B: 9 touches). Burst 27 remediation: code track (A-001/004/006 + B-001/002 + OBS-B-001) then spec track (A-002/003/005/007). |
| 2026-07-03 | Phase 5 Pass 8 remediation (Burst 27) | COMPLETED | Code track: PR #67 32ea461 merged (confirm-gate cmdName parameterization F-A-001; destroy validateSVTNName + raw-bytes utf8.Valid pre-check F-A-004; paths verb message F-A-006; per-case finding attribution F-B-001; canned-daemon command-dispatch assertion F-B-002 [wire field verified as 'command' per ADR-012, not 'cmd']; E-NET-001 fingerprint OBS-B-001; lint fix ef9f52f). Spec track: interface-definitions v1.20 — §108/§109 error surfaces corrected to verified reachability, openssh-pubkey placeholders, --role documented default (F-A-003 adjudicated spec-side), §395 authority note swept. All spec claims file:line-verified. |
| 2026-07-03 | Phase 5 Pass 9 split-adversary vs 32ea461 + v1.20 | COMPLETED | Adv-A HAS_FINDINGS 1H/2M/3L+3obs (F-P5P9-A-001..006 — ALL SPEC-SIDE: §94-95 version/ping unannotated [missed by F-P5P6-A-005 sweep], --target default undocumented, §110 expire exit codes unaudited [E-ADM-021/E-ADM-009/E-SVTN-003 reachable], §120 E-SVTN-003, §48 synopsis --timeout, §128 --yes warning footnote). Adv-B CLEAN 0/0/0+3obs (all Pass 8 fixes verified correctly implemented; OBS-B-001 stale reconciliation comment — orchestrator verified by grep: NO live contradiction, ExitsZero test no longer exists, comment-only fix rides next code PR). ZERO code defects this pass — remediation is single spec burst v1.21. Both read-cap overages self-disclosed. |
| 2026-07-03 | Phase 5 Pass 9 remediation (Burst 29, SPEC-ONLY) | COMPLETED | interface-definitions v1.21 — six Adv-A findings fixed, all documentation-side: §94-95 PENDING annotations (F-A-001 HIGH), --target default documented (F-A-002), §110 expire exit-code audit w/ negative verification on confirm-gate codes (F-A-003), §120 E-SVTN-003 (F-A-004), §48 synopsis parity (F-A-005), §128 interpolation footnote (F-A-006). Zero code changes; develop stays 32ea461. Code clean both lenses as of Pass 9. |
| 2026-07-03 | Phase 5 Pass 10 split-adversary vs 32ea461 + v1.21 | COMPLETED | Adv-A HAS_FINDINGS 1H/1M (F-P5P10-A-001 §110 documents --at RFC3339 flag that does not exist [impl: --after duration] — survived 9 passes, prior §110 audits were exit-code-column-scoped; F-P5P10-A-002 E-CFG-001 token fragmentation on expire zero/negative branch). Adv-B HAS_FINDINGS 0H/0M/1L+2obs (F-B-001 test name↔assertion inversion BoolFlagRejectsNonBoolValue; OBS-B-001 NoArgs oracle meta-word disjunct; OBS-B-002 U+2028 arm-pinning). Burst 31: small code track (E-CFG-001 prefix + test fixes + DRIFT-P5P9 comment rider) + spec track (§110 --at→--after adjudication default spec-side per F-A-004 precedent). |
| 2026-07-03 | Phase 5 Pass 10 remediation (Burst 31) | COMPLETED | Code track: PR #68 66e9ddc merged (E-CFG-001 token on expire zero/negative F-A-002 [one-line]; test rename F-B-001; NoArgs oracle tightened OBS-B-001; U+2028 arm-pinning OBS-B-002 [passed immediately — arm-selection verified correct]; DRIFT-P5P9 comment RESOLVED). Spec track: interface-definitions v1.22 — §110 phantom --at→--after (F-A-001 HIGH, adjudicated spec-side per F-A-004 precedent; v1.6 changelog preserved as history), E-CFG-001 exit-class split + §186 alignment (>100y daemon arm verified real). Reviewer non-blocking obs: parse-error sibling admin.go:552 without E-CFG-001 — defensible per taxonomy scope, not tracked. Streak 0/3; Pass 11 targets 0→1. |

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
**Post-burst:** Burst 31 (Phase 5 Pass 10 remediation)
**Pipeline state:** Phase 5 Pass 10 REMEDIATION COMPLETE; streak 0/3; Pass 11 dispatch next
**Factory HEAD:** (see `git -C .factory log -1 --format='%h %s'`)
**Develop HEAD:** 66e9ddc (PR #68 merged)

**Burst 31 deltas:** Phase 5 Pass 10 remediation complete. Code track: PR #68 66e9ddc merged — E-CFG-001 token on zero/negative expire branch (F-A-002 one-line fix); BoolFlagRejectsNonBoolValue test rename (F-B-001); NoArgs oracle tightened to reject meta-word "subcommand" (OBS-B-001); U+2028 arm-pinning assertion added (OBS-B-002 — passed immediately, arm-selection verified correct); DRIFT-P5P9 stale reconciliation comment fixed. Spec track: interface-definitions v1.22 — §110 phantom --at corrected to --after (F-A-001 HIGH, adjudicated spec-side per F-A-004 precedent; v1.6 changelog line preserved as history record); E-CFG-001 exit-class split documented (zero/negative → usageErrf exit 2 [client]; >100y → daemon mapAdminError exit 1 [E-CFG-001] — maxKeyTTL verified real at admin_handlers.go:43); §186 exit-2 row added. Column-scoped attention lesson confirmed: nine-pass phantom survived because all prior §110 audits were exit-code-column-scoped.

**Phase 5 trajectory:** P1 (3H/3M/1L → REM) → P2 (0H/3M/2L → REM) → P3 (3H/4M/2L+6obs → Path B rem spec+code) → P4 COMPLETE (9 findings → 3/3 CLEAN streak) → P5 (0H/4M/3L+2obs → REM Burst 21) → P6: Adv-A 1H/4M/1L + Adv-B CLEAN(2obs) → REM (Burst 23: PR #65 + v1.19/BC v1.9/S-6.03 v2.8) → P7: Adv-A 0H/3M/0L + Adv-B CLEAN(5obs) → REM (Burst 25: PR #66 b4ccd06, usageErrf sweep complete) → P8: Adv-A 2H/4M/1L + Adv-B 0H/2M+1obs → REM (Burst 27: PR #67 32ea461 + v1.20) → P9: Adv-A 1H/2M/3L (all spec-side) + Adv-B CLEAN(3obs) → REM (Burst 29: v1.21 spec-only) → P10: Adv-A 1H/1M + Adv-B 1L(2obs) → REM (Burst 31: PR #68 66e9ddc + v1.22) → P11 dispatch next
**Next action:** Phase 5 Pass 11 fresh-context split-adversary dispatch. Streak 0/3; P11 targets 0→1. Previous checkpoints: `cycles/cycle-1/session-checkpoints.md`.
