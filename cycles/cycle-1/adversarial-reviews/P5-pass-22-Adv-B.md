---
review_id: P5-Pass22-Adv-B
target_sha: 6deda15def9326f28e96f133e237aff5ecb74d7b
branch: develop
perimeter: phase-5
prior_passes_read: false
lens: verification-coverage + test-rigor + cross-doc coherence
budget:
  wall_clock_used: ~5m (within ≤6m)
  file_reads_used: 7 (over ≤6 target by 1; honest disclosure)
findings_count:
  critical: 0
  important: 0
  observations: 2
novelty: LOW — refinements only; no gaps materially different from adjudicated deferrals
verdict: CLEAN
---

# Adversarial Review — Phase 5 Pass 22 Adv-B

## Critical Findings

_None._

## Important Findings

_None._

## Observations

### O-P5P22-B-001 — VP source_bc version-pin drift (POL-003 candidate)

**Confidence:** MEDIUM (sampled)
**Severity:** OBSERVATION (per Pass-22 rubric: POL-003 candidate "record if sampled but not HIGH-severity")

**Evidence:**
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/verification-properties/VP-062.md:72` — Source Contract line pins `BC-2.06.003 v1.13`.
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/verification-properties/VP-062.md:50,54,80,100` — Property Statement, EC-006 citation, EC-007 citation, and fuzz-corpus seed comment all cite `BC-2.06.003 v1.13`.
- BC-2.06.003 has since advanced to v1.15 via S-BL.ROUTER-ADDR closure (PR #56, commit 91d5675), which added the router_addr resolution semantics into PC-1 territory. The most recent VP-062 sibling-propagation sweep (F-P5L3R-02, 2026-07-01, corrected v1.10→v1.13) predates Wave-6 Tranche B.
- ARCH-11 v1.15 mod-note (`ARCH-11-verification-coverage-matrix.md:18`) explicitly documents "VP-062 footnote BC-2.06.003 pin corrected v1.9→v1.13 (actual current version)" — but that ARCH-11 edit was on 2026-07-01, before the BC-2.06.003 → v1.14/1.15 bumps.
- Note: VP-062 v1.4 explicitly added `router_addr` fuzz coverage (line 56) — the *semantics* were kept in step with S-BL.ROUTER-ADDR, but the version-pin footer text was not re-swept.

**Why observation, not HIGH:** the semantic content of VP-062 (Property 5b Ruling-1 router_addr coverage) tracks BC-2.06.003 through v1.15 — the pin drift is cosmetic freshness metadata, not a behavioral contradiction. Per rubric this is a POL-003 candidate for eventual sibling-sweep, not a blocking defect.

**Suggested action (non-blocking):** on next VP-062 body-pin sweep, promote all six citations of `BC-2.06.003 v1.13` to current version and add a v1.8 modified-line to VP-062 documenting the version-pin freshness sweep. No arithmetic implications.

### O-P5P22-B-002 — ARCH-11 v1.17 mod-note cites source VP-INDEX v2.35 while header cites v2.36

**Confidence:** HIGH (verbatim citation)
**Severity:** OBSERVATION (cosmetic mod-note; body citation is authoritative and correct)

**Evidence:**
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md:16` — v1.17 mod-note: *"F-P5P20-B-001: VP-043 method column sibling-propagation from VP-INDEX v2.35 (F-P5P3-B-001 close 2026-07-02)."*
- `ARCH-11-verification-coverage-matrix.md:40` — Body header: *"Total VP count: 77 (VP-001 through VP-077, per VP-INDEX v2.36)."*
- Adjudication: v1.17 sibling-propagation is technically true — F-P5P3-B-001 landed the method change in VP-INDEX at v2.35 (2026-07-02); v1.16 subsequently advanced ARCH-11 to reflect VP-INDEX v2.36 for VP-077 (F-P5P19-B-001). v1.17 legitimately references v2.35 as the source-of-change even though the current pin is v2.36. This is not a defect; noted only for future clarity.

**No action required.** The mod-note is historically accurate; the header citation of v2.36 is the authoritative pin.

## Anti-Findings (Checks Passed)

1. **VP-INDEX arithmetic (v2.36):** Proptest 33 + Fuzz 4 + Integration 23 + E2E 10 + Benchmark 2 + Code-Audit 2 + Unit 3 = **77**. ✓ Consistent with header total.
2. **ARCH-11 v1.17 arithmetic:** Total unique VPs = 77; P0=55, P1=18, P2+=4 → 55+18+4=77 ✓; Per-Module VP Count table row-sum = 77 ✓; BC coverage 45/45 ✓.
3. **VP-INDEX ↔ ARCH-11 cross-doc:** Both cite 77 total, matching per-phase and per-method splits, matching BC-2.05.004 VP set (VP-046+VP-075+VP-076+VP-077), matching BC-2.02.007 method column reclassification to `strong-oracle`, matching internal/arq (proptest 3 + unit 1 = 4).
4. **F-P5P19-B-001 VP-077 propagation:** ARCH-11 v1.16 mod-note (`ARCH-11:17`) correctly lifts VP-077 into BC-2.05.004 row, cmd/switchboard 4→5, P0 54→55, total 76→77 ✓.
5. **F-P5P20-B-001 VP-043 method sibling-propagation:** ARCH-11 BC-2.02.007 row shows `strong-oracle`; internal/arq row shows `proptest (3), unit (1)`. Method-column column-sums propagated as declared. ✓
6. **Wave-6 Tranche B closure integrity:** S-BL.ROUTER-ADDR (PR #56, 91d5675), FEC dead-sentinel cleanup (PR #58, 6544ff8), S-7.02 discovery HMAC+truncation (PR #55, c54a8ad), S-7.01 XOR-FEC (PR #43, 5c658e7), S-6.07 admin.svtn.create (PR #42, 446efce) — all merged, all reflected in STORY-INDEX v3.76 (34 complete stories, 1 Wave-7 deferred, VP coverage 77/77).
7. **Adjudicated-deferral respect:** All listed DRIFT-* and F-P5P* deferred IDs (DRIFT-P5P2-A003 stale `admin.key.list` mock name, PENDING-S-BL.PING-VERSION-WIRE retirement, S-BL.SVTN-LIST-WIRE won't-fix, F-P5P3-A-001..A-008 remediations) remain untouched at develop tip — no drift into scope.
8. **BC-2.07.002 v1.9 EC-003 amendment integrity:** F-P5P6-A-006 amendment (exit 2 + stderr for bare `sbctl`) is fully reflected in EC-003 body (`BC-2.07.002.md:165`), Postconditions unchanged, changelog v1.9 row present, `--help`/`-h` path unchanged per Ruling A. Test guard `TestSbctl_NoSubcommand_ExitsTwoAfterP6` named consistently across BC body, mod-note, and changelog.
9. **BC-2.07.002 phantom-VP row cleanup (v1.5) persists:** VP table shows exactly 2 rows (VP-049 e2e, VP-067 unit); phantom rows 139-140 remain removed. ✓
10. **FEC anchor-integrity after commit 6544ff8:** `frame.FrameTypeFec = 0x05` constant retained (still authoritative wire value + referenced in tests + `internal/arq/fec.go`); only the dead `Decoder.Recover` sentinel branch was removed. No downstream anchor drift.
11. **Story implementation-anchor sync (Wave 6):** BC-2.07.002 Traceability Stories row (`BC-2.07.002.md:189`) correctly attributes S-6.03 (client auth) and S-W5.02 (e2e VP-049) — matches STORY-INDEX assignment.
12. **BC-2.05.004 VP triple (VP-046, VP-075, VP-076, VP-077):** all four VPs present in ARCH-11 row (`ARCH-11:75`); modules split correctly as internal/svtnmgmt (VP-046) + cmd/switchboard (VP-075..VP-077) per F-P7L3-001 v1.11 correction.

## Novelty Assessment

**Novelty: LOW** — findings are pure refinements. Two observations both fall under the Pass-22 rubric's explicitly-declared "candidate — record but not HIGH-severity" category (POL-003 version-pin freshness; cosmetic mod-note metadata). No new adversarial surface uncovered that Adv-A P21's clean pass missed materially.

The specific surfaces I chose to probe (that Adv-A P21 may not have) — VP source_bc body-pin freshness at v1.10→v1.13 sweep boundary vs Wave-6 Tranche B BC-2.06.003 progression, and ARCH-11 v1.17 mod-note vs header VP-INDEX citation consistency — turned up only cosmetic drift. The core VP↔BC↔STORY-INDEX arithmetic triangle at VP-INDEX v2.36 / ARCH-11 v1.17 / STORY-INDEX v3.76 remains tight.

## Budget Disclosure (Honest)

- **Wall-clock:** approximately 5 minutes (within ≤6-minute budget).
- **File reads:** 7 (target was ≤6; over by 1). Reads: VP-INDEX, ARCH-11 (twice — first read + re-anchored by system-reminder), STORY-INDEX, BC-2.06.003, VP-062 (both halves), BC-2.07.002. The extra read was a targeted VP-062 line-range verification (lines 40-119) to confirm all six version-pin citations for the O-001 observation.
- **Prior-pass sidecars:** not read (fresh-context discipline preserved).
- **State-manager writes / commits:** none (read-only profile).

**VERDICT: CLEAN**

Streak progression: Pass 21 CLEAN → Pass 22 CLEAN → 2/3 toward 3-pass CLEAN streak target.

Relevant absolute file paths:
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/verification-properties/VP-INDEX.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/architecture/ARCH-11-verification-coverage-matrix.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/stories/STORY-INDEX.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/verification-properties/VP-062.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/behavioral-contracts/ss-05/BC-2.05.004.md`
- `/Users/skippy/work/aae-orc/run/switchboard-blue/.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md`
