---
pass: 24
subphase: Adv-B
verdict: HAS_FINDINGS
timestamp: 2026-07-03T00:00:00
worktree_head_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
factory_head_sha_at_dispatch: 89fef775acab150dbb2672feb34ef496f189f1c2
prior_passes_read: false
budget_minutes_used: <adversary-reported>
budget_reads_used: <adversary-reported>
---

# Phase 5 Pass 24 — Adversary B Review

**Lens:** Verification-coverage + test-rigor + cross-doc coherence (POL-006 / ARCH-11 traceability)
**Perimeter:** `.factory/` artifacts only — no source code reads
**Anti-findings checked:** Pass-23 adjudicated deferrals (O-P5P23-B-001 DEFERRED cosmetic)

## F-P5P24-B-001 — MEDIUM — POL-006 — ARCH-11 BC-2.02.001 Row Missing VP-042 Reverse-Trace

**Finding class:** Novel — POL-006 ARCH-11 reverse-trace propagation gap when VP-INDEX row
declares multiple BC anchors.

**Description:** ARCH-11 verification-coverage matrix line 53 contains the BC-2.02.001 row.
VP-INDEX line 68 anchors VP-042 to **both** BC-2.01.001 and BC-2.02.001 (dual-anchor row).
The BC-2.02.001 row in ARCH-11 L53 includes VP-041 in its verification column but omits VP-042.

The omission breaks the bidirectional traceability claim: VP-INDEX asserts that VP-042 covers
BC-2.02.001, but ARCH-11 does not propagate this back into the verification-coverage matrix for
that BC.

**Artifact:** ARCH-11 L53, BC-2.02.001 row — `vp_traces` column missing VP-042.
**VP-INDEX reference:** L68 — `anchors: [BC-2.01.001, BC-2.02.001]` (dual-anchor).

**Remediation:** SHIPPED in Burst 61 at `ffb028be9d65a2a9ac9a923a19d1503cb6c0b1b1`.

---

## F-P5P24-B-002 — MEDIUM — POL-006 — ARCH-11 BC-2.05.001 Row Missing VP-008 Reverse-Trace

**Finding class:** POL-006 ARCH-11 reverse-trace propagation gap (same novel axis as B-001).

**Description:** ARCH-11 verification-coverage matrix line 72 contains the BC-2.05.001 row.
VP-INDEX line 34 anchors VP-008 to **both** BC-2.05.001 and BC-2.05.002 (dual-anchor row).
The BC-2.05.001 row in ARCH-11 L72 omits VP-008 from its verification column.

Same structural defect as F-P5P24-B-001: VP-INDEX declares the bidirectional trace; ARCH-11
fails to propagate the reverse direction for the second of the two BCs anchored by VP-008.

**Artifact:** ARCH-11 L72, BC-2.05.001 row — `vp_traces` column missing VP-008.
**VP-INDEX reference:** L34 — `anchors: [BC-2.05.001, BC-2.05.002]` (dual-anchor).

**Remediation:** SHIPPED in Burst 61 at `ffb028be9d65a2a9ac9a923a19d1503cb6c0b1b1`.

---

## F-P5P24-B-003 — MEDIUM — POL-006 — ARCH-11 BC-2.04.003 Row Missing VP-012 Reverse-Trace

**Finding class:** POL-006 ARCH-11 reverse-trace propagation gap (same novel axis as B-001/B-002;
three simultaneous instances confirm this is a novel axis, not a one-off).

**Description:** ARCH-11 verification-coverage matrix line 67 contains the BC-2.04.003 row.
VP-INDEX line 38 anchors VP-012 to **both** BC-2.05.003 and BC-2.04.003 (dual-anchor row).
The BC-2.04.003 row in ARCH-11 L67 omits VP-012 from its verification column.

Three instances of the same structural defect (B-001, B-002, B-003) confirm a systematic gap:
when VP-INDEX anchors a VP to multiple BCs, only the first BC (or the "primary" anchor) receives
the reverse-trace propagation in ARCH-11. The secondary anchor is routinely missed. This is a
novel axis not previously surfaced — the per-BC and per-VP passes in prior rounds did not
cross-reference VP-INDEX dual-anchor rows against ARCH-11 BC coverage columns.

**Artifact:** ARCH-11 L67, BC-2.04.003 row — `vp_traces` column missing VP-012.
**VP-INDEX reference:** L38 — `anchors: [BC-2.05.003, BC-2.04.003]` (dual-anchor).

**Remediation:** SHIPPED in Burst 61 at `ffb028be9d65a2a9ac9a923a19d1503cb6c0b1b1`.

---

## O-P5P24-B-001 — OBSERVATION (cosmetic) — BC-2.01.001 Method Column Label

**Description:** BC-2.01.001 method column is labeled `proptest` but the verification property
set for this BC includes VP-041 and VP-042, which are benchmark VPs (not proptest). The method
label understates the verification footprint.

**Status:** SHIPPED opportunistically in Burst 61 (sweep included in the
`ffb028be9d65a2a9ac9a923a19d1503cb6c0b1b1` commit as part of the ARCH-11 reverse-trace
propagation sweep).

---

## O-P5P24-B-002 — OBSERVATION (cross-lens) — VP-043/044/045/050/055 Status: Draft with Merged Implementing Stories

**Description:** VP-043 (S-7.01), VP-044 (S-7.02), VP-045 (S-7.02), VP-050 (S-7.03), and
VP-055 (S-7.02) retain `status: draft` in their frontmatter while their implementing stories
S-7.01 (merged PR #43 5c658e7), S-7.02 (merged PR #55 c54a8ad), and S-7.03 (draft) have
progressed. The Wave-5 v2.19 analogous sweep (which advanced VP statuses after Wave-5 story
merges) was never run for Wave-6 Tranche B equivalents.

**Classification:** Cross-lens observation, borderline Adv-A perimeter (traceability vs
verification-coverage). The VP-status-draft condition for merged implementing stories is
cosmetic metadata drift — the verification properties themselves are not stale, only the
lifecycle flag.

**Status:** DEFERRED — cross-lens borderline, may be addressed in a separate targeted sweep
burst rather than within the Pass 24 remediation scope. S-7.03 is still draft so VP-050 may
be intentionally deferred pending that story's merge.

---

VERDICT: HAS_FINDINGS
