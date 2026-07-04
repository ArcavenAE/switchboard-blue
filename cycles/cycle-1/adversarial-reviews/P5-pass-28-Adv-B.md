---
pass_id: P5-pass-28-Adv-B
lane: B
phase: 5
cycle: cycle-1
timestamp: 2026-07-03T00:00:00Z
worktree_identity_tuple:
  factory_state_step: phase-5-pass-27-concluded-has-findings
  develop_tip_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  factory_head_sha: 9121350
  go_module: github.com/arcavenae/switchboard
  readme_title: Switchboard
  note: read-only-adversary-cannot-run-git; orchestrator verified SHAs out-of-band before dispatch; factory_head_sha is Burst-71c (9121350) SHA-patch commit
verdict: HAS_FINDINGS
findings_count: 2
critical: 0
high: 0
medium: 2
low: 0
observations: 0
findings: [F-P5P28-B-001, F-P5P28-B-002]
reconstructed_from_orchestrator_adjudication: true
---

# Phase 5 Pass 28 — Adversary B Review

**Lens:** Verification-coverage + test-rigor + cross-doc coherence (POL-005 / POL-006)
**Perimeter:** `.factory/` artifacts only — no source code reads
**Anti-findings checked:** Pass-27 adjudicated remediations (F-P5P27-B-001/002 all SHIPPED 8613b4e Burst 71b; O-P5P27-B-001 full-file 12-dual-anchor-VP VP-list sweep confirmed clean baseline)

> **Note:** This sidecar was reconstructed from orchestrator adjudication records at Burst 75
> (Pass 29 remediation). The original Adv-B dispatch occurred as part of the Pass 28
> split-adversary dispatch from Burst 72. Findings are verbatim from the orchestrator's
> adjudication records in STATE.md Phase-5 progress row and sprint-state.yaml L1547-1565
> (pass_28 block). Cross-reference: P5-pass-28-Adv-A.md (Burst 73a 0b3f55f).
>
> [process-gap] POL-006-SWEEP-EXPAND drift item was codified in Burst 73a as a result of
> this pass being the sixth-consecutive Lane-B POL-006 propagation-gap recurrence. The
> drift item mandates expanding the sweep protocol from single-column-per-pass to all 4
> dual-anchor-derived columns of ARCH-11 (VP-list + Phase + Method + Module). VP-list
> swept Burst 68b (v1.20); Phase-column swept Burst 71b (v1.21); Module-column swept
> Burst 73b (v1.22). Method-column sweep confirmed clean in Pass 29 Adv-B (NO_FINDINGS),
> closing POL-006-SWEEP-EXPAND.

---

## F-P5P28-B-001 — MEDIUM — POL-006 — ARCH-11 L83 BC-2.05.008 Module Cell Missing internal/admission

**Finding class:** POL-006 — ARCH-11 module-column propagation gap; sixth-consecutive Lane-B recurrence (P24×3, P25×2, P26×2, P27×2 Phase-col, P28 this instance + B-002); FIRST module-column instance of the class.

**Description:** ARCH-11 verification-coverage matrix L83 — the BC-2.05.008 row Module column reads `internal/routing`. VP-059 is a dual-anchor VP anchored to both BC-2.05.005 and BC-2.05.008 per VP-INDEX v2.36 L85. VP-059 lives in `internal/admission` per VP-INDEX v2.36. The sibling BC-2.05.005 (ARCH-11 L80) correctly shows `internal/hmac, internal/admission (PC-3)` in its Module column. The BC-2.05.008 Module cell is missing `internal/admission`.

This is the first module-column instance of the POL-006 propagation-gap class. The pattern reveals that ARCH-11 module-column updates for dual-anchor VPs propagate to the primary/first-listed BC but not to the secondary/second-listed BC — the same asymmetry observed in the VP-list and Phase-column instances.

**Cited evidence:**
- ARCH-11 L83: BC-2.05.008 row Module column — `internal/routing` (missing `internal/admission`)
- ARCH-11 L80: BC-2.05.005 row Module column — `internal/hmac, internal/admission (PC-3)` (correct)
- VP-INDEX v2.36 L85: VP-059 dual-anchor to BC-2.05.005 + BC-2.05.008; module `internal/admission`
- Blast radius: 1 ARCH-11 matrix cell → MEDIUM.

**Remediation:** SHIPPED at `.factory` commit `49032a3` (Burst 73b — ARCH-11 v1.21→v1.22 BC-2.05.008 Module column corrected from `internal/routing` to `internal/routing, internal/admission`).

---

## F-P5P28-B-002 — MEDIUM — POL-006 — ARCH-11 L57 BC-2.02.001 Module Cell Missing internal/halfchannel

**Finding class:** POL-006 — ARCH-11 module-column propagation gap; seventh consecutive Lane-B finding in the POL-006 class; second module-column instance.

**Description:** ARCH-11 verification-coverage matrix L57 — the BC-2.02.001 row Module column reads `internal/multipath`. VP-042 is a dual-anchor VP anchored to both BC-2.01.001 and BC-2.02.001 per VP-INDEX v2.36 L68. VP-042 lives in `internal/halfchannel` per VP-INDEX v2.36. The sibling BC-2.01.001 (ARCH-11 L50) correctly lists `internal/halfchannel` in its Module column. The BC-2.02.001 Module cell is missing `internal/halfchannel`.

This is the same dual-anchor asymmetry pattern as F-P5P28-B-001: primary-anchor BC row carries the module correctly; secondary-anchor BC row does not. VP-042 was previously involved in F-P5P24-B-001 (VP-list column propagation gap at BC-2.02.001) — the same VP and BC pair is now surfacing in the module-column axis.

**Cited evidence:**
- ARCH-11 L57: BC-2.02.001 row Module column — `internal/multipath` (missing `internal/halfchannel`)
- ARCH-11 L50: BC-2.01.001 row Module column — lists `internal/halfchannel` (correct)
- VP-INDEX v2.36 L68: VP-042 dual-anchor to BC-2.01.001 + BC-2.02.001; module `internal/halfchannel`
- Blast radius: 1 ARCH-11 matrix cell → MEDIUM.

**Remediation:** SHIPPED at `.factory` commit `49032a3` (Burst 73b — ARCH-11 v1.22 BC-2.02.001 Module column corrected from `internal/multipath` to `internal/multipath, internal/halfchannel`).

---

## [process-gap] POL-006-SWEEP-EXPAND — Sixth-Consecutive Lane-B Recurrence; Module-Column New Subclass

**Class:** process-gap — POL-006 ARCH-11 propagation-gap class has now appeared in six consecutive Lane-B adversarial passes (P24×3, P25×2, P26×2, P27×2-Phase-col, P28 this pass×2-Module-col).

**Evidence:** The recurrence pattern has now covered three distinct column axes of ARCH-11:
- **VP-list column:** Passes 24-26 (F-P5P24-B-001/002/003, F-P5P25-B-001, F-P5P26-B-001/002); Burst 68b full-file 77-VP sweep established clean baseline for this axis.
- **Phase column:** Pass 27 (F-P5P27-B-001 — 4 session-access rows + 2 proactive expansion); Burst 71b clean baseline for this axis.
- **Module column:** Pass 28 (F-P5P28-B-001/002 — this pass); Burst 73b sweep covers this axis.
- **Method column:** Not yet swept at Pass 28 close — pending Pass 29 Adv-B.

**POL-006-SWEEP-EXPAND directive (codified Burst 73a as drift item in STATE.md):** Sweep protocol must expand from single-column-per-pass to all 4 dual-anchor-derived columns of ARCH-11 (VP-list + Phase + Method + Module). Pass 29 Adv-B is tasked with Method-column verification as the remaining unswepped axis.

**Disposition:** [process-gap] codified as STATE.md drift item POL-006-SWEEP-EXPAND. Upstream drbothen/vsdd-factory tracker entry pending (filed in .vsdd-factory-issues-pending.md). Not blocking Phase 5 convergence. Method-column sweep confirmed clean in Pass 29 Adv-B — closes POL-006-SWEEP-EXPAND.

---

VERDICT: HAS_FINDINGS
