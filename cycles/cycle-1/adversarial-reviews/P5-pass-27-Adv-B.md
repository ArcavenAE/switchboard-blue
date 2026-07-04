---
pass_id: P5-pass-27-Adv-B
lane: B
phase: 5
cycle: cycle-1
timestamp: 2026-07-03T00:00:00Z
worktree_identity_tuple:
  factory_state_step: phase-5-pass-26-concluded-has-findings
  develop_tip_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  factory_head_sha: 1f2f557
  go_module: github.com/arcavenae/switchboard
  readme_title: Switchboard
  note: read-only-adversary-cannot-run-git; orchestrator verified SHAs out-of-band before dispatch; factory_head_sha is the Burst-69 persistence commit preceding Burst-70 Pass-27 dispatch
verdict: HAS_FINDINGS
findings_count: 3
critical: 0
high: 1
medium: 0
low: 1
observations: 1
findings: [F-P5P27-B-001, F-P5P27-B-002, O-P5P27-B-001]
reconstructed_from_orchestrator_adjudication: true
---

# Phase 5 Pass 27 — Adversary B Review

**Lens:** Verification-coverage + test-rigor + cross-doc coherence (POL-005 / POL-006 / POL-008)
**Perimeter:** `.factory/` artifacts only — no source code reads
**Anti-findings checked:** Pass-26 adjudicated remediations (F-P5P26-B-001/002 all SHIPPED cf135b9 Burst 68b; proactive full-file 77-VP reverse-trace sweep confirmed clean baseline)

> **Note:** This sidecar was reconstructed from orchestrator adjudication records at Burst 75
> (Pass 29 remediation). The original Adv-B dispatch occurred as part of the Pass 27
> split-adversary dispatch from Burst 70. Findings are verbatim from the orchestrator's
> adjudication records in STATE.md Phase-5 progress row and sprint-state.yaml L1528-1546
> (pass_27 block). Cross-reference: P5-pass-27-Adv-A.md (Burst 71a 63ba29c).

---

## F-P5P27-B-001 — HIGH — POL-008 — ARCH-11 Session-Access BC Rows Phase-Column Drift (4 rows + 2 proactive-sweep-expansion)

**Finding class:** POL-008 — ARCH-11 verification-coverage matrix Phase-column drift across session-access BC rows; first Phase-column instance of the POL-006/POL-008 propagation-gap class.

**Description:** ARCH-11 verification-coverage matrix — four session-access BC rows carry stale Phase-column values inconsistent with VP-INDEX v2.36 authoritative phase data. The per-row union rule applies: ARCH-11 BC row Phase-column must equal the union of all VP phases across the BC's VP set per VP-INDEX.

Affected rows (initial finding):
- BC-2.04.001 — Phase-column stale
- BC-2.04.003 — Phase-column stale
- BC-2.04.004 — Phase-column stale
- BC-2.04.005 — Phase-column stale

**Proactive sweep expansion:** After filing the initial four-row finding, a proactive sweep of the Phase-column across all ARCH-11 BC rows revealed two additional propagation-gap instances in the same class:
- BC-2.05.004 (L78) — Phase-column P0 → corrected to P0/P1 per VP-INDEX v2.36
- BC-2.07.004 (L89) — Phase-column P0 → corrected to P0/P1 per VP-INDEX v2.36

All six rows remediated in a single burst.

**Pattern class:** This is the first Phase-column instance of the ARCH-11 propagation-gap class. Preceded by VP-list column instances in Passes 24-26 (F-P5P24-B-001/002/003, F-P5P25-B-001/002, F-P5P26-B-001/002). The POL-006-SWEEP-EXPAND drift item now requires the sweep protocol to cover all 4 dual-anchor-derived columns (VP-list + Phase + Method + Module) in each pass.

**Blast radius:** 6 ARCH-11 matrix cells across 2 BC subsections → HIGH.

**Remediation:** SHIPPED at `.factory` commit `8613b4e` (Burst 71b — ARCH-11 v1.20→v1.21 Phase-column corrections for all 4 session-access rows + 2 proactive-sweep-expansion rows). POL-008 Phase-column sweep executed and confirmed clean for all remaining rows.

---

## F-P5P27-B-002 — LOW — POL-008 — ARCH-11 BC-2.05.007 Method-Column Stale "+audit" Annotation

**Finding class:** POL-008 — ARCH-11 method-column stale annotation; cosmetic cleanup of pre-formalization shorthand surfaced during Phase-column sweep.

**Description:** ARCH-11 L81 BC-2.05.007 Method column carries "+audit" annotation suffix. Cross-referencing VP-INDEX v2.36: VP-007 (proptest) and VP-057 (proptest) are the only VPs anchored to BC-2.05.007. Neither VP-007 nor VP-057 carries an audit-method claim in VP-INDEX. The "+audit" suffix was a pre-formalization shorthand that does not correspond to any active VP audit method.

This is adjacent to the O-P5P23-B-001 cosmetic observation that flagged the same "+audit" pattern on ARCH-11 L50 BC-2.01.005 and L78 BC-2.05.007 — the O-P5P23-B-001 observation was a non-blocking cosmetic; this finding elevates it to LOW because the method annotation is directly verifiable against VP-INDEX and represents a machine-checkable drift.

**Cited evidence:**
- ARCH-11 L81: BC-2.05.007 row Method column — "+audit" suffix present
- VP-INDEX v2.36: VP-007 (method: proptest) + VP-057 (method: proptest) — no audit anchor
- Blast radius: 1 ARCH-11 matrix cell → LOW.

**Remediation:** SHIPPED at `.factory` commit `8613b4e` (Burst 71b — ARCH-11 v1.21 BC-2.05.007 method column corrected from "proptest + audit" to "proptest").

---

## O-P5P27-B-001 — OBSERVATION — Full-File 12-Dual-Anchor-VP Sweep Baseline Confirmed

**Observation class:** Proactive sweep — VP-list column baseline verification following Burst 68b proactive full-file 77-VP sweep.

**Description:** A full-file sweep of the VP-list column in ARCH-11 was executed covering all 12 dual-anchor VPs (VPs with ≥2 BC entries in VP-INDEX). The sweep verified that every (VP, BC) anchor pair for dual-anchor VPs appears correctly in the ARCH-11 VP-list column. No propagation gaps found on the VP-list axis.

This confirms the clean baseline established by Burst 68b (cf135b9) for the VP-list column remains intact after Burst 71b's Phase-column remediations. The POL-006-SWEEP-EXPAND drift item's VP-list column axis is verified clean.

**12 dual-anchor VPs swept:** VP-008, VP-012, VP-016, VP-040, VP-042, VP-043, VP-044, VP-059, VP-062, VP-067, and additional dual-anchor VPs per VP-INDEX v2.36 — all clean on VP-list column axis. Phase-column axis addressed in F-P5P27-B-001. Method-column axis (subsequent drift axis) addressed in Pass 28 F-P5P28-B-001/002. Module-column axis (fourth drift axis) not yet swept at Pass 27 time; addressed in Pass 28 Adv-B.

**Disposition:** Non-blocking observation. Establishes VP-list column as clean for Pass 28 baseline. Phase-column sweep conducted under F-P5P27-B-001. All 6 phase-drift instances remediated in same burst.

---

VERDICT: HAS_FINDINGS
