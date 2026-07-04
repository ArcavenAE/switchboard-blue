---
pass_id: P5-pass-26-Adv-B
lane: B
phase: 5
cycle: cycle-1
timestamp: 2026-07-03T00:00:00Z
worktree_identity_tuple:
  factory_state_step: phase-5-pass-25-concluded-has-findings
  develop_tip_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  factory_head_sha: 9f069efee00b0f8c75b5ea155c8465c8deab1f3b
  go_module: github.com/arcavenae/switchboard
  readme_title: Switchboard
  note: read-only-adversary-cannot-run-git; orchestrator verified SHAs out-of-band before dispatch
verdict: HAS_FINDINGS
findings_count: 2
critical: 0
high: 0
medium: 2
low: 0
observations: 0
findings: [F-P5P26-B-001, F-P5P26-B-002]
process_gap: true
reconstructed_from_orchestrator_adjudication: true
---

# Phase 5 Pass 26 — Adversary B Review

**Lens:** Verification-coverage + test-rigor + cross-doc coherence (POL-005 / POL-006)
**Perimeter:** `.factory/` artifacts only — no source code reads
**Anti-findings checked:** Pass-25 adjudicated deferrals (F-P5P25-B-001/002 all SHIPPED 99f1356
Burst 65b; O-P5P24-B-002 DEFERRED cross-lens)

---

## F-P5P26-B-001 — MEDIUM — POL-006 — ARCH-11 L57 BC-2.02.003 Row Missing VP-040 Reverse-Trace

**Finding class:** POL-006 — ARCH-11 reverse-trace propagation gap; fourth instance of
F-P5P24-B-*** propagation gap class (P24×3, P25×1, P26 this instance)

**Description:** ARCH-11 verification-coverage matrix L57 — the BC-2.02.003 row does not
include `VP-040` in its VPs column. VP-INDEX v2.36 L66 anchors VP-040 solely to BC-2.02.003
as its single BC anchor (e2e, P1, internal/multipath). When a VP anchors to a BC, the ARCH-11
reverse-trace matrix must carry that VP in the anchored BC's row.

**Pattern class:** This is the fourth consecutive instance of the same ARCH-11 reverse-trace
propagation gap identified across Passes 24 through 26:
- F-P5P24-B-001: ARCH-11 L53 BC-2.02.001 row missing VP-042 (dual-anchor VP-INDEX L68)
- F-P5P24-B-002: ARCH-11 L72 BC-2.05.001 row missing VP-008 (dual-anchor VP-INDEX L34)
- F-P5P24-B-003: ARCH-11 L67 BC-2.04.003 row missing VP-012 (dual-anchor VP-INDEX L38)
- F-P5P25-B-001: ARCH-11 L85 BC-2.07.002 row missing VP-067 (dual-anchor secondary)
- **F-P5P26-B-001:** ARCH-11 L57 BC-2.02.003 row missing VP-040 (single anchor)

The Burst 65b remediation (SHIPPED 99f1356) closed F-P5P25-B-001 but did not perform a
proactive full-corpus sweep of all 77 VPs for remaining propagation gaps.

**Cited evidence:**
- ARCH-11 L57: BC-2.02.003 row VPs column — VP-040 absent
- VP-INDEX v2.36 L66: VP-040 anchored to BC-2.02.003 (e2e, P1, internal/multipath)
- Blast radius: 1 ARCH-11 matrix cell — MEDIUM.

**Blast radius:** 1 ARCH-11 matrix cell → MEDIUM.

**Remediation:** SHIPPED at `.factory` commit `cf135b9` (Burst 68b — spec-steward ARCH-11
v1.19→v1.20 VP-040 → BC-2.02.003 row added). Burst 68b also executed a proactive full-file
POL-006 reverse-trace sweep across all 77 VPs confirming ONLY these 2 gaps existed; no
additional drift found. Clean baseline established for Pass 27.

---

## F-P5P26-B-002 — MEDIUM — POL-006 — ARCH-11 L50 BC-2.01.003 Row Missing VP-016 Reverse-Trace

**Finding class:** POL-006 — ARCH-11 reverse-trace propagation gap; fifth consecutive Lane-B
recurrence (P22-obs deferred, P24×3, P25×1, P26 this instance + B-001); FIRST dual-anchor
instance in this class where one anchor is correctly present.

**Description:** ARCH-11 verification-coverage matrix L50 — the BC-2.01.003 row does not
include `VP-016` in its VPs column. VP-INDEX v2.36 L42 anchors VP-016 to BOTH BC-2.01.001
(correctly present at ARCH-11 L48) and BC-2.01.003 (systematically dropped from L50).

This is the first instance in the F-P5P24-B-*** propagation gap class where a dual-anchor VP
is correctly traced in one BC row but dropped in the other. The pattern reveals that ARCH-11
updates for dual-anchor VPs tend to propagate to the primary/first-listed BC but not to the
secondary/second-listed BC — a systematic asymmetry in how the matrix is maintained.

**Cited evidence:**
- ARCH-11 L48: BC-2.01.001 row VPs column — VP-016 present (correct)
- ARCH-11 L50: BC-2.01.003 row VPs column — VP-016 absent (propagation gap)
- VP-INDEX v2.36 L42: VP-016 anchors to both BC-2.01.001 and BC-2.01.003
- Blast radius: 1 ARCH-11 matrix cell — MEDIUM.

**Blast radius:** 1 ARCH-11 matrix cell → MEDIUM.

**Remediation:** SHIPPED at `.factory` commit `cf135b9` (Burst 68b — spec-steward ARCH-11
v1.19→v1.20 VP-016 → BC-2.01.003 row added). Same Burst 68b proactive full-file sweep
confirmed this was the only remaining dual-anchor propagation gap after VP-040 fix.

---

## [process-gap] POL-006 Reverse-Trace Class — 5-Consecutive-Pass Recurrence

**Class:** process-gap — POL-006 reverse-trace propagation gap (machine-checkable via
ARCH-11↔VP-INDEX bidirectional lint)

**Evidence:** POL-006 reverse-trace propagation gap class has now appeared in five consecutive
Lane-B adversarial passes:
- Pass 22 (obs): O-P5P22-B-001 POL-003-candidate VP-062 v1.13 pin drift (deferred, boundary)
- Pass 24: F-P5P24-B-001/002/003 (3 findings) ARCH-11 reverse-trace propagation gap —
  VP-042/BC-2.02.001, VP-008/BC-2.05.001, VP-012/BC-2.04.003
- Pass 25: F-P5P25-B-001 (1 finding) ARCH-11 L85 BC-2.07.002 missing VP-067
- Pass 26: F-P5P26-B-001/002 (2 findings) ARCH-11 L57 BC-2.02.003 missing VP-040 +
  ARCH-11 L50 BC-2.01.003 missing VP-016 (first dual-anchor asymmetry instance)

**Pattern:** Every remediation burst fixes the specific cited instances but does not perform a
full corpus sweep, leaving residual gaps for the next pass to find. The class is inherently
machine-checkable: parse VP-INDEX BC(s) column, parse ARCH-11 BC→VP Coverage Table, assert
every (VP, BC) anchor pair appears in both. Especially catches dual-anchor VPs (≥2 BC entries
in VP-INDEX) where one BC row drops the VP.

**Burst 68b baseline:** Executed proactive full-file reverse-trace sweep across all 77 VPs.
Confirmed that after cf135b9, ONLY VP-040 and VP-016 were missing from ARCH-11 (the two
findings above). No additional gaps exist as of this baseline. Future recurrences would
indicate new VPs added without corresponding ARCH-11 propagation.

**Disposition:** [process-gap] codified. Upstream drbothen/vsdd-factory issue to be filed
post-Phase-5 convergence proposing machine-checkable ARCH-11↔VP-INDEX reverse-trace lint
(target: cycle-2 or vsdd-factory 1.0.0-rc.22+). Deferred — not blocking Phase 5 convergence;
class now baselined with clean sweep; upstream tool ownership.

---

VERDICT: HAS_FINDINGS
