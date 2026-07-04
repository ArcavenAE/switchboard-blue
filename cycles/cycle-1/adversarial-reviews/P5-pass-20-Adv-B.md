---
pass_id: P5P20-Adv-B
adversary_lens: verification-coverage + cross-doc coherence
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: <not-verified-read-only-no-bash>
  refs/heads/develop_sha: <not-verified-read-only-no-bash>
  origin/develop_sha: <not-verified-read-only-no-bash>
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS (orchestrator-verified out-of-band; adversary tool profile has no Bash — Read/Grep/Glob only)
budget:
  wall_clock_target: <=6 min
  wall_clock_used: ~5 min
  file_reads_target: <=6
  file_reads_used: 6
  overage_disclosure: on-budget; 6/6 reads.
verdict: HAS_FINDINGS
streak_state:
  adjudicated_deferrals_respected: true
  reopened_deferrals: none
  respected_list:
    - DRIFT-P5P7-O1, DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - F-P5P13-A-001..F-P5P19-B-003 (all SHIPPED or adjudicated per prior remediation)
    - F-P5P19-A-001 SHIPPED at .factory e65e429
delivered_by: p5-pass20-adv-b
---

# Phase 5 Pass 20 Adv-B — findings

Verify-then-claim: finding carries file:line anchors resolved at develop tip 6deda15def9326f28e96f133e237aff5ecb74d7b (orchestrator preflight).

---

### F-P5P20-B-001 [HIGH]: ARCH-11 v1.16 + ARCH-07 v1.9 VP-043 method column stale after VP-INDEX v2.35 (POL-002 + VP-INDEX↔ARCH coherence)

Anchors:
- `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:58` (BC-2.02.007 row, Method cell)
- `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:112` (internal/arq per-module row, Methods cell)
- `.factory/specs/architecture/ARCH-07-verification-architecture.md:183` (Phase-1c-refinement Test-Sufficient table, VP-043 Method cell)

Class: POL-002 sibling-sweep gap + VP-INDEX ↔ ARCH cross-doc coherence

Confidence: HIGH. Severity: HIGH.

Symptom: VP-INDEX v2.35 (2026-07-02, per F-P5P3-B-001 close) explicitly reclassified VP-043 from Proptest to strong-oracle Unit; the changelog row documents the shift. Downstream architecture rows carrying VP-043 Method cells were not swept in the same propagation:

- ARCH-11 line 58 BC-2.02.007 row Method cell: still says `proptest` (should be `strong-oracle`).
- ARCH-11 line 112 internal/arq per-module row Methods cell: still says `proptest (4)` (should be `proptest (3), unit (1)` since VP-043 is one of the 4 arq VPs and now classifies as Unit).
- ARCH-07 line 183 Phase-1c-refinement Test-Sufficient table VP-043 Method cell: still says `proptest` (should be `strong-oracle`).

Column arithmetic drift (ARCH-11 Method totals): sum of `proptest` column across all BC rows = 34 vs canonical VP-INDEX Proptest tally 33 (one over-count = VP-043 stale mention); sum of `unit` column = 2 vs canonical VP-INDEX Unit tally 3 (one under-count = VP-043 missing). ARCH-11 column arithmetic no longer reconciles with the canonical VP-INDEX v2.35.

This is a POL-002 sibling-sweep gap: VP-INDEX v2.35 bump (F-P5P3-B-001 close) did not sweep the sibling ARCH-11 and ARCH-07 documents that carry Method cells sourced from VP-INDEX. The verification-coverage matrix (ARCH-11) losing arithmetic reconciliation is HIGH severity because Phase 6 formal-verification planning depends on the ARCH-11 module-level counts to schedule proptest campaign scope.

Verify: read of VP-INDEX v2.35 changelog confirms VP-043 method reclassification is committed at canonical HEAD. Read of ARCH-11 lines 55-60 (BC-2.02.007 row region), 108-115 (arq module row region) confirms stale `proptest` mentions. Read of ARCH-07 lines 180-186 (Phase-1c-refinement table region) confirms stale `proptest` in VP-043 row. Column sums computed by manual tally of Method cells across all rows in ARCH-11 verification-coverage table.

Remediation shape:
- ARCH-11 v1.16 → v1.17. Line 58 BC-2.02.007 Method: `proptest` → `strong-oracle` (or `unit (strong-oracle)` if adjacent rows use that format — inspect BC-2.05.004 row for convention). Line 112 arq module Methods: `proptest (4)` → `proptest (3), unit (1)`. Changelog row citing F-P5P20-B-001 with WHAT (VP-043 method column propagation from VP-INDEX v2.35) / WHY (F-P5P3-B-001 close 2026-07-02 sibling-sweep gap) / column-sum reconciliation (proptest 34→33, unit 2→3).
- ARCH-07 v1.9 → v1.10. Line 183 VP-043 Method: `proptest` → `strong-oracle`. Changelog row citing F-P5P20-B-001.

Recommend combining ARCH-11 + ARCH-07 into a single spec-steward commit since both propagate the same F-P5P20-B-001 root cause (single VP-INDEX v2.35 reclassification).

---

## Anti-findings (checked and passing)

- **POL-001 changelog-completeness (ARCH-11 v1.16)**: ARCH-11 v1.16 changelog row is present with WHAT/WHY/version-bump form; the missing VP-043 propagation is not a POL-001 defect (v1.16 addressed a different concern) — it's a sibling-sweep gap that needs its own v1.17 row.
- **POL-002 story-index-row-sync (unrelated to VP-043)**: STORY-INDEX v3.75 master-table rows still align with story-file headers for non-Wave-6-aggregate purposes.
- **VP-INDEX v2.35 internal consistency**: VP-INDEX itself is coherent post-F-P5P3-B-001; VP-043 row correctly reflects strong-oracle Unit; row-count aggregate at bottom is 77. Only the downstream propagation to ARCH-11 + ARCH-07 is missing.
- **Adjudicated deferrals**: none of the deferrals in streak-state appear in this finding; all remain deferred.
- **ARCH-11 line 58 vs adjacent rows**: BC rows immediately above/below use `proptest`, `strong-oracle`, and `unit (strong-oracle)` — remediation should match convention of adjacent post-VP-INDEX-v2.35 rows (choose the form used consistently in the closest reconciled section).
- **BC-2.02.007 story anchors**: BC-2.02.007 EC/AC references (S-7.01 in Wave 6) not affected by the Method cell change; only cell content, not row identity.
- **VP-043 body content (VP-INDEX v2.35 canonical)**: property text intact; only the classification metadata propagated wrong to downstream docs.

---

VERDICT: HAS_FINDINGS
