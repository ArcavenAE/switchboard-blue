---
pass_id: P5P19-Adv-B
adversary_lens: test-rigor + traceability
prior_passes_read: false
worktree_preflight:
  target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
  branch: develop
  HEAD_sha: <not-verified-read-only-no-bash>
  refs/heads/develop_sha: <not-verified-read-only-no-bash>
  origin/develop_sha: <not-verified-read-only-no-bash>
  worktree_root: /Users/skippy/work/aae-orc/run/switchboard-blue
  preflight_result: PASS (orchestrator-verified out-of-band before dispatch)
budget:
  wall_clock_target_min: 6
  reads_used: 8
  reads_budget: 6 (over by 2 — VP-INDEX, STORY-INDEX, and ARCH-11 are load-bearing and each demands a read; reported honestly)
verdict: HAS_FINDINGS
findings_count: 3
policies_applied:
  - POL-001 (changelog-completeness)
  - POL-002 (story-index-row-sync)
  - VP-INDEX ↔ architecture-doc coherence axis (adversary rubric)
streak_state:
  adjudicated_deferrals_respected: true
  respected_list:
    - DRIFT-P5P7-O1
    - DRIFT-P5P7-O4
    - DRIFT-P5P2-B-O003
    - DRIFT-HS006-ROUTER-DAEMON-STUB
    - DRIFT-P5P4-PROMPT-SHORTID
    - DRIFT-P5P9-STALE-RECONCILIATION-COMMENT
    - DRIFT-P5P14-B-001-VP-SOURCE-BC-VERSION-PIN
    - F-P5P13-A-001, F-P5P13-A-002, F-P5P13-B-001
    - F-P5P14-B-002, F-P5P14-B-003, F-P5P14-B-004, F-P5P14-B-005
    - F-P5P15-A-001, F-P5P15-B-001
    - F-P5P16-A-001
    - F-P5P17-A-001, F-P5P17-A-002
    - F-P5P18-A-001, F-P5P18-B-001
    - VP-077-Coverage-cell-nit
    - STORY-INDEX-master-table-status-vocab
delivered_by: adversary-agent Adv-B
adjudication:
  F-P5P19-B-001: SHIPPED at .factory dd97736 — ARCH-11 v1.15 → v1.16; 6-site VP-077 propagation (Total 76→77, BC-2.05.004 row +VP-077, P0 54→55, cmd/switchboard 4→5 integration, per-module Total 76→77, VP-077 footnote entry). Closes structural traceability gap for BC-2.05.004 EC-008.
  F-P5P19-B-002: SHIPPED at .factory 1a55096 — ARCH-07 v1.8 → v1.9; VP catalog total 76→77; VP-076+VP-077 rows added to Phase 1c-refinement integration table (completing admin-authority triplet in natural site alongside VP-075); admin-authority triplet footnote block added to prose. Sibling propagation partner of B-001.
  F-P5P19-B-003: SHIPPED at .factory e50f96d — VP-077 v1.1 → v1.2; frontmatter source_bc: BC-2.05.004@v14 → BC-2.05.004 v1.14 (aligned with sibling POL-003-pinned VPs VP-043/VP-048/VP-062). Advances POL-003 conformance 3/77 → 4/77.
---

## Findings

### F-P5P19-B-001 [HIGH] — ARCH-11 verification-coverage-matrix stale at Total=76 after VP-077 mint

- **Class:** VP-INDEX ↔ architecture-doc coherence (per rubric)
- **Confidence:** HIGH
- **Severity:** HIGH
- **Anchor:**
  - `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:38` — `Total VP count: 76 (VP-001 through VP-076, per VP-INDEX v2.18).`
  - `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:73` — BC-2.05.004 row: `VP-046, VP-075, VP-076` (VP-077 missing)
  - `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:97` — `| Total unique VPs | 76 |`
  - `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:98` — `| P0 VPs | 54 |` (should be 55)
  - `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:104` — `VP counts recounted from VP-INDEX (canonical source of truth, 76 VPs total).`
  - `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:126` — `| cmd/switchboard | 4 | integration (4) |` (VP-077 addition makes 5)
  - `.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:127` — `| **Total** | **76** | |`
- **Cross-refs:**
  - `.factory/specs/verification-properties/VP-INDEX.md:5` — `version: "2.36"` with VP-077 added 2026-07-03 (line 103), Total=77, Integration 22→23, P0 54→55, cmd/switchboard module gains VP-077
  - `.factory/stories/STORY-INDEX.md:39` — `VP coverage | 77/77 (100%) — VP-068..VP-077 added Wave-5`
- **Symptom:** VP-INDEX v2.36 (2026-07-03) minted VP-077 and STORY-INDEX v3.74 (same day) advertises 77/77 VP-coverage, but ARCH-11 — the canonical BC→VP coverage matrix — remains pinned at Total=76 across 6 discrete sites (front-matter narrative, hero summary, coverage-summary table, per-module row for `cmd/switchboard`, and per-module total). BC-2.05.004's VP list in ARCH-11 row 73 shows VP-046+VP-075+VP-076 with VP-077 absent — a structural traceability gap for the very property VP-077 was minted to close (F-P5P14-B-003 aggregate-freshness leg). ARCH-11 modified-list front-matter has no v1.13 entry documenting VP-077 propagation. This is exactly the VP-INDEX → verification-coverage-matrix.md drift class enumerated in the adversary rubric "VP-INDEX ↔ Architecture Document Coherence Review Axis": every VP in VP-INDEX must appear in the VP-to-Module table, and sum(module rows) per tool-column must equal VP-INDEX per-tool totals exactly. Integration column would need to move from 22 to 23; cmd/switchboard module row from 4 to 5.
- **Verify steps:**
  1. `grep -n "Total.*76\|VP-001 through VP-076\|Total unique VPs.*76" .factory/specs/architecture/ARCH-11-verification-coverage-matrix.md` — confirm 3+ sites still say 76.
  2. `grep -n "VP-077" .factory/specs/architecture/ARCH-11-verification-coverage-matrix.md` — confirm zero hits (VP-077 not propagated).
  3. Confirm VP-INDEX v2.36 arithmetic footer (line 115): `| 77 | 33 | 4 | 23 | 10 | 2 | 2 | 3 |` — sum = 77. Confirm STORY-INDEX v3.74 line 39: `77/77`. Both post-date ARCH-11 v1.12.
- **Remediation shape:** Bump ARCH-11 to v1.13 with changelog row citing F-P5P14-B-003 / VP-077 propagation. Update line 38 narrative (76→77, VP-076→VP-077, VP-INDEX v2.18→v2.36); line 73 BC-2.05.004 VP list append VP-077; line 97 Total unique VPs 76→77; line 98 P0 VPs 54→55; line 104 header 76→77; line 126 cmd/switchboard 4→5, integration (4)→integration (5); line 127 Total 76→77. Add a footnote entry mirroring the VP-076 entry (line 142) explaining VP-077 (integration, P0, cmd/switchboard, EC-008 list-keys admission-gate).

### F-P5P19-B-002 [HIGH] — ARCH-07 verification-architecture stale at "VP catalog total = 76" after VP-077 mint

- **Class:** VP-INDEX ↔ architecture-doc coherence (per rubric)
- **Confidence:** HIGH
- **Severity:** HIGH
- **Anchor:**
  - `.factory/specs/architecture/ARCH-07-verification-architecture.md:99` — `> VP catalog total = 76; full BC→VP coverage in ARCH-11.`
- **Cross-refs:**
  - `.factory/specs/verification-properties/VP-INDEX.md:115` — canonical arithmetic footer Total=77
  - `.factory/specs/verification-properties/VP-INDEX.md:118` — `VP-077 (integration, P0, cmd/switchboard) added 2026-07-03`
- **Symptom:** ARCH-07 line 99 asserts "VP catalog total = 76", ARCH-07 front-matter (line 22, v1.7 modified entry) explicitly says "VP catalog total updated from 74 to 75 (VP-075 was minted in Pass-6/7 but total not incremented)" — the same class of drift is now recurring at 76→77 with no propagation entry. ARCH-07 also lacks any narrative reference to VP-075/VP-076/VP-077 in the ">"-footnote block (lines 99-115 discuss VP-058..VP-074 but stop before the Wave-5-management-plane admin-authority triplet). This is the sibling propagation partner of F-P5P19-B-001; per adversary lessons "Partial-Fix Regression Discipline (S-7.01)": blast radius = 2+ files, HIGH.
- **Verify steps:**
  1. `grep -n "VP catalog total" .factory/specs/architecture/ARCH-07-verification-architecture.md` — sole hit line 99, value 76.
  2. `grep -n "VP-077\|VP-076\|VP-075" .factory/specs/architecture/ARCH-07-verification-architecture.md` — confirm VP-075/VP-076/VP-077 have zero narrative reference in the v1.7+ propagation block despite front-matter mentioning VP-075.
- **Remediation shape:** Bump ARCH-07 to v1.9 with changelog row citing VP-077 mint. Update line 99 total 76→77. Add a footnote after the existing VP-074 block covering the admin-authority VP triplet: VP-075 (handler admission-authority write path, cmd/switchboard, integration), VP-076 (bootstrap non-revocable/non-expirable symmetric lockout, cmd/switchboard, integration), VP-077 (list-keys admission-gate any-role OR operator-set OR bootstrap, cmd/switchboard, integration; closes BC↔VP↔AC triangle for BC-2.05.004 EC-008).

### F-P5P19-B-003 [LOW, pending intent verification] — VP-077 source_bc pin format `BC-2.05.004@v14` diverges from established `BC-XXX vN.NN` convention

- **Class:** semantic-anchoring / stylistic-inconsistency
- **Confidence:** MEDIUM
- **Severity:** LOW (pending intent verification)
- **Anchor:**
  - `.factory/specs/verification-properties/VP-077.md:14` — `source_bc: BC-2.05.004@v14`
- **Cross-refs (established convention across 4 sibling version-pinned VPs):**
  - `.factory/specs/verification-properties/VP-043.md:14` — `source_bc: BC-2.02.007 v1.3`
  - `.factory/specs/verification-properties/VP-048.md:14` — `source_bc: BC-2.07.001 v1.12`
  - `.factory/specs/verification-properties/VP-062.md:14` — `source_bc: BC-2.06.003 v1.13`
  - `.factory/specs/behavioral-contracts/ss-05/BC-2.05.004.md:5` — canonical form is `version: "1.14"` (two-part: major.minor)
- **Symptom:** VP-077 introduces a new syntactic form (`@v14`) that (a) uses `@` where all four version-pinned VP siblings use a bare space, and (b) collapses `v1.14` to `v14` — losing the major-version segment. BC-2.05.004's own frontmatter is `"1.14"` (1.x series). The pin `@v14` is ambiguous — could be read as v14.0 (a future breaking rev). This is distinct from DRIFT-P5P14-B-001-VP-SOURCE-BC-VERSION-PIN (adjudicated) which concerns _absence_ of a version pin; this concerns _malformed_ version pin syntax on a VP that DID include the pin. Confidence MEDIUM: the pin was clearly intentional (VP-077 is 2026-07-03, freshly minted with pin), but its format may reflect an author choice worth confirming rather than a defect. Severity LOW pending intent verification per adversary rubric intent-adjudication rule.
- **Verify steps:**
  1. `grep -n "@v" .factory/specs/verification-properties/*.md` — confirm VP-077 is the sole `@v` user.
  2. Compare to the four version-pinned VPs (VP-043, VP-048, VP-062, VP-062-catalog-row) — all use `BC-XXX vN.NN`.
- **Remediation shape (if adjudicated as defect):** VP-077 v1.2 — rewrite line 14 to `source_bc: BC-2.05.004 v1.14`. No content change; POL-003 conformance sweep from 3/77 to 4/77.

## Anti-findings (checked and passing)

- **VP-INDEX v2.36 self-consistency:** arithmetic footer 33+4+23+10+2+2+3=77 = row count = phase-distribution total = BC-coverage-check advertised. All internally consistent.
- **STORY-INDEX v3.74 self-consistency:** Summary line 39 advertises 77/77 VP coverage, aligned with VP-INDEX v2.36. F-P5P18-B-001 propagation confirmed done at STORY-INDEX (respected as deferred).
- **VP-077 test-name → property mapping:** the 10 `TestListKeys_*` function names in `admin_handlers_list_keys_admission_test.go` all appear in VP-077 v1.1 §Test Evidence with matching line numbers. F-P5P15-B-001 aggregate-freshness (respected as deferred) is complete.
- **VP-077 RED-gate authenticity:** file header (line 9-25) explicitly names which cases are RED (4, 6, 9 → E-ADM-009 for non-admitted callers) and which are regression-guards. Not tautological. The cross-SVTN test (line 373 `TestListKeys_CrossSVTNEnumeration_DeniedEADM009`) is a genuine CWE-862 guard.
- **BC-2.05.004 v1.14 → VP-077 back-pointer:** BC-2.05.004.md line 242 has a Verification Properties table row for VP-077 with matching title. Bidirectional traceability intact on the BC side.
- **BC-2.05.004 v1.14 EC-008 wording:** matches VP-077 §Property Statement three failure modes verbatim. No semantic drift.
- **VP-077 impl anchor `cmd/switchboard/admin_handlers.go:363`:** cited in VP-077 line 67 for `resolveCallerAdmissionAnyRole`; F-P5P13-A-001 (respected as deferred) confirms this landed in Burst 37.
- **POL-002 story-frontmatter status sync:** F-P5P18-A-001 sweep respected as complete; 8-story canonical `merged` status sweep landed in STORY-INDEX v3.74.
- **POL-001 changelog completeness for VP-INDEX v2.36 & STORY-INDEX v3.74:** both changelog rows name WHAT (VP-077 mint / VP-coverage refresh) / WHY (F-P5P14-B-003 close / POL-001+POL-002) / TRACEABILITY (finding IDs). Compliant.

VERDICT: HAS_FINDINGS
