---
document_type: blocking-issues-resolved
level: ops
version: "1.0"
status: archive
producer: state-manager
timestamp: 2026-07-02T00:00:00Z
cycle: cycle-1
inputs: [STATE.md]
traces_to: STATE.md
---

# Resolved Blocking Issues — cycle-1

<!-- Blocking issues that were resolved and archived from STATE.md.
     Open blocking issues remain in STATE.md. -->

## Extracted from STATE.md on 2026-07-02

| ID | Issue | Severity | Blocked Phase | Owner | Resolution | Resolved Date |
|----|-------|----------|--------------|-------|------------|---------------|
| DRIFT-BC-2-08-001-V1-3-GOV-LEAF | BC-2.08.001 v1.3 governance_leaf annotation gap — Wave-6 combined wave-gate Pass 3 F1 finding. Adversary flagged missing `governance_leaf: true` annotation on changelog row declaring "No behavioral changes". | MEDIUM | Phase 3 Wave-6 wave-gate | spec-steward | Retro-annotated at BC-2.08.001 v1.5 (2026-07-02) — Pass 3 F1 remediation. F1 remediated; Pass 4 through Pass 6 subsequently CLEAN; BC-5.39.001 3/3 CONVERGED. | 2026-07-02 |

## Extracted from STATE.md on 2026-07-04 (compact-state post-BC-5.39.001-convergence)

| ID | Issue | Severity | Blocked Phase | Owner | Resolution | Resolved Date |
|----|-------|----------|--------------|-------|------------|---------------|
| DRIFT-P5P1-A001-SVTN-LIST-ORPHAN | sbctl svtn list wire cmd svtn.list has zero daemon handler; contradicts BC-2.07.002 canonical test vector happy-path. Wave-6 merged with this gap; internal-code adversary chain missed it because gap is only visible cross-cutting sbctl main.go vs daemon Command: literals. | HIGH | Phase 5 | product-owner | Closed by annotation 2026-07-02 (BC-2.07.002 v1.6, Burst 8, HEAD 4659cb88). PENDING-<S-BL.SVTN-LIST-WIRE> annotation added on BC canonical test vector. | 2026-07-02 |
| DRIFT-P5P1-A002-SESSIONS-LIST-ORPHAN | sbctl sessions list wire cmd sessions.list has zero daemon handler; contradicts BC-2.03.002 PC-1 "core operator experience" claim. S-7.02 (merged PR #55) implements Discovery.Enumerate() internally but no wire boundary. | HIGH | Phase 5 | product-owner | Closed by annotation 2026-07-02 (BC-2.03.002 v1.4, Burst 8, HEAD 4659cb88). PENDING-S-BL.DISCOVERY-WIRE annotation added on BC PC-1. | 2026-07-02 |
| DRIFT-P5P1-A003-PING-VERSION-ORPHAN | sbctl ping / sbctl version dispatch to wire commands ping / version with zero daemon handlers; return E-RPC-010 masking as "unknown command" from a live daemon. No BC anchor — CLI-declared promise only. | MEDIUM | Phase 5 | product-owner | Closed by annotation 2026-07-02 (BC-2.07.002 v1.7 EC-004+EC-005 + S-BL.PING-VERSION-WIRE stub minted, Burst 11). | 2026-07-02 |
| DRIFT-P5P1-B-H001-ENET006-TAXONOMY-ORPHAN | error-taxonomy.md:119 E-NET-006 declares operator-facing error message with zero emission site in cmd/ or internal/. BC-2.09.002 anchor. S-7.04 pending. | HIGH | Phase 5 | spec-steward | Closed by annotation 2026-07-02 (error-taxonomy v4.2, Burst 8, HEAD 4659cb88). PENDING-S-7.04 annotation added. F-P5-Adv-B-L-001 also closed as side-effect. | 2026-07-02 |
| DRIFT-P5P3-A001..A009/B001..B003/B17 | Phase 5 Pass 3 findings — multiple HIGH..LOW defects in code and spec tracks. | HIGH..LOW | Phase 5 | implementer / spec-steward | ALL RESOLVED (Bursts 16–18): PR #62 c76a8d5 (code), taxonomy v4.3/v4.4 (spec). Detail: `cycles/cycle-1/closed-drift.md`. | 2026-07-02 |
| DRIFT-P5P4-PROMPT-SHORTID | Pass 4 finding: prompt short-ID display issue in sbctl. | MED | Phase 5 | implementer | RESOLVED (Burst 19): PR #63 cbd0272. | 2026-07-03 |
| DRIFT-P5P6-ANNOTATION-EXITCODE | Pass 6 finding: annotation exit-code discrimination gap in CLI dispatch layer. | MED | Phase 5 | implementer | RESOLVED (Burst 23): PR #65 4d7d9e0. | 2026-07-03 |
| DRIFT-P5P9-STALE-RECONCILIATION-COMMENT | P5 Pass 9 finding: stale reconciliation comment in code. | LOW | Phase 5 | implementer | RESOLVED (Burst 31): PR #68 66e9ddc — stale comment fixed; U+2028 hexdump comment rider applied. | 2026-07-03 |
| POL-006-SWEEP-EXPAND | Sixth-consecutive Lane-B POL-006 propagation-gap recurrence. Sweep protocol needed expansion from single-column-per-pass to all 4 dual-anchor-derived columns of ARCH-11 (VP-list + Method + Phase + Module). | OBS | Phase 5 | orchestrator | RESOLVED: Pass 29 Adv-B confirmed Method-column clean across all 45 BCs (12 dual-anchor VPs + 33 single-anchor VPs). All 4 dual-anchor-derived columns verified clean. Six-consecutive Lane-B POL-006 recurrence class (P24-P29) terminates at P29. | 2026-07-03 |
| STATE-MANAGER-SIBLING-SWEEP | ESCALATED — SEVENTH-CONSECUTIVE RECURRENCE (SECOND recursive-inside-codification; third-order failure demonstrated). Seven instances of state-manager failing sibling-sweep discipline resulting in POL-002 regressions. | HIGH | Phase 5 | orchestrator | CLOSED 2026-07-04 per Burst 86: 4-Adv-B-clean-consecutive threshold met at P33+P34+P35+P36 (per Adv-B sidecars). Closure per orchestrator adjudication on Adv-B O-2 recommendation. | 2026-07-04 |
| DRIFT-P5P36-PHANTOM-ERPC-004 | Pass 36 Adv-A: E-RPC-004 cited in wave-6-tranche-a-scope-rulings.md Ruling-12 §1 and S-6.07-svtn-admin-create.md L78 but never catalog-defined in error-taxonomy.md at any point. | HIGH | Phase 5 | spec-steward | CLOSED — Remediated Burst 87 (wave-6-rulings v1.14, S-6.07 v1.14). Redirected 2 citation sites from phantom E-RPC-004 to catalog-anchored E-RPC-010. | 2026-07-04 |
| DRIFT-P5P36-RULING-11-12-AUTHORSHIP-PREMISE-SIBLINGS | Pass 36 Adv-A: Ruling-11 and Ruling-12 cite E-RPC-002 as established at authorship (2026-07-01), but catalog row was minted 2026-07-04 in Burst 82. Authorship-premise drift at 4 sibling sites. | MEDIUM | Phase 5 | spec-steward | CLOSED — Remediated Burst 87. 4 dated audit-trail footnotes added at Ruling-11 §1 L1021, L1035, Ruling-12 §1 L1120, L1129. | 2026-07-04 |
| DRIFT-P5P34-TAXONOMY-ORPHAN-ERPC-002-003 | Pass 34 Adv-A: E-RPC-002 and E-RPC-003 emitted from production code paths with no error-taxonomy.md catalog rows. Ruling-14 §10 authorized emission on false premise. | HIGH | Phase 5 | spec-steward | CLOSED — Pass 35 Adv-A verified all catalog rows present, E-RPC-010 forbidden clause correct, interface-definitions.md v1.29 closed-set enumeration correct; taxonomy-orphan class fully remediated. | 2026-07-03 |
| DRIFT-P5P35-RULING-14-GOVERNANCE-PREMISE-STALE | Pass 35 Adv-A: wave-6-tranche-a-scope-rulings.md §10 (Ruling-14) asserts "E-RPC-002 is already defined in error-taxonomy.md" — factually wrong at ruling authorship (2026-07-01). False premise preserved verbatim despite amendments. | MEDIUM | Phase 5 | spec-steward | CLOSED — Burst 85 wave-6-tranche-a-scope-rulings.md v1.13: §10 Impact Assessment inline footnote annotation added + v1.13 changelog row minted; governance-text-vs-taxonomy premise corrected. | 2026-07-04 |
