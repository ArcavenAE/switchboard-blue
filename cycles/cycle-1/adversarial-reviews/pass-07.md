---
artifact_id: adv-p1-pass-07
review_target: phase-1-spec-crystallization
producer: adversary
pass: 7
fresh_context: true
findings_count: 7
findings_by_severity: {critical: 0, high: 2, medium: 4, low: 1}
findings_with_process_gap: 0
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 7

**Trajectory:** 27 → 18 → 17 → 21 → 17 → 14 → **7**. Zero critical for third consecutive pass.

## High

### F-P7-001 — ARCH-07 phase categorization mis-anchored: 14 P0 VPs listed under "P1 (Should Prove)"
- ARCH-07:55-73 declares "P1 Properties (Should Prove)" then lists VP-016 through VP-030. VP-INDEX:42-56 classifies VP-016/017/018/019/020/021/022/023/024/025/026/028/029/030 as P0 (only VP-027 is P1).
- ARCH-07:104 lists VP-051 under "P1 Properties Added in Phase 1c-refinement"; VP-INDEX:77 says VP-051 = P0.
- Per-row phase columns in ARCH-11:29-70 match VP-INDEX. So canonical phase is on VP-INDEX/ARCH-11 side; ARCH-07's section headers are wrong.
- Route: architect. Fix: Reframe ARCH-07 section headers as proof-method buckets (Pure-core proptest / Boundary / Integration / Race) rather than phase buckets, OR regenerate phase-grouped tables from VP-INDEX.

### F-P7-002 — CAP↔BC priority inversion: 4 CAPs at P1/P2 with P0 realizing BCs
- capabilities.md:127 CAP-014 P1 → realized by BC-2.04.003/004 (BC-INDEX = P0)
- capabilities.md:133 CAP-015 P1 → realized by BC-2.04.005 (P0)
- capabilities.md:161 CAP-019 P1 → realized by BC-2.05.004 (P0)
- capabilities.md:211 CAP-024 P2 → realized by BC-2.07.003 (P0)
- L2-INDEX:119 priority distribution table reflects the old (P1) reading.
- Pass-6 promoted BCs to P0 across BC files/PRD/ARCH-11 but capabilities.md and L2-INDEX weren't updated.
- Route: business-analyst (or PO). Fix: Lift CAPs to match their P0 BCs. Update capabilities.md `(PN)` annotations, L2-INDEX priority distribution count (P0 17→20+).

## Medium

### F-P7-003 — E-ADM-007 severity "broken" with exit code 0 contradicts severity definition
- error-taxonomy:41 rule: broken = non-zero exit. Line 56: E-ADM-007 broken / 0 (continues).
- The "continues" semantic (session keeps running on read-only console upstream rejection) fits `degraded` (line 42 "Zero exit with warning") not `broken`.
- Route: product-owner. Fix: reclassify E-ADM-007 from `broken` to `degraded`.

### F-P7-004 — BC-2.05.005 HMAC input description omits channel header (ARCH-02 includes it)
- BC-2.05.005:47 says "computed over the full frame (outer header fields + payload)".
- ARCH-02:152-154 canonical: "outer header bytes 0–35 || channel header || payload, with hmac_tag bytes treated as zeros".
- Security-relevant: omitting channel header means SACK bits / FEC flags could be flipped without HMAC failure.
- Route: product-owner. Fix: replace BC-2.05.005 phrasing with ARCH-02's exact text.

### F-P7-005 — BC-2.01.001 §Verification Properties table assigns same VP set to 3 different properties
- Three rows all cite `VP-016, VP-018, VP-041, VP-042`.
- Row 1 "exactly one frame per tick" → VP-016 correct; VP-041/042 (jitter/latency) wrong.
- Row 2 "sequence monotonicity" → VP-017 is the exact match but missing.
- Row 3 "empty-tick zero-length payload" → VP-018 correct; VP-053 (added Phase 1c) is the better anchor and missing.
- Route: product-owner. Fix: rebuild table such that each row's VP list is property-specific. Add VP-017 + VP-053.

### F-P7-006 — module-criticality.md Module Inventory bullet list has 17 items; Classification table has 18 (cmd/switchboard added but not in inventory)
- Inventory (lines 35-51): 17 items. Classification (line 74): adds `switchboard | cmd/switchboard | LOW`. Summary line 106: "Total | 18".
- Pass-3 added cmd/switchboard to the table but missed the bullet list.
- Route: product-owner. Fix: append `**switchboard** — Daemon entry point. No business logic.` to inventory list.

## Low

### F-P7-007 — VP-057 ARCH-04 line reference off-by-3 (164-168 vs actual 167-170)
- VP-057:55 and :103 cite "ARCH-04 §HMAC keying, lines 164-168" but the HKDF block is at ARCH-04:167-170. Section name correct; range wrong.
- Route: architect. Fix: replace "164-168" → "167-170" (2 occurrences in VP-057).

## Verdict

**NOT_CONVERGED.** 7 findings (0 critical, 2 high, 4 medium, 1 low). Three consecutive passes at 0 critical. The 2 highs are tractable structural fixes (ARCH-07 categorization, CAP priority alignment); the 4 mediums are single-file edits. Trajectory 27→18→17→21→17→14→7 shows asymptotic convergence.
