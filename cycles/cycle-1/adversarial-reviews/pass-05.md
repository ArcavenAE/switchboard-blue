---
artifact_id: adv-p1-pass-05
review_target: phase-1-spec-crystallization
producer: adversary
pass: 5
fresh_context: true
findings_count: 17
findings_by_severity: {critical: 0, high: 8, medium: 7, low: 2}
findings_with_process_gap: 0
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 5

**Trajectory:** 27 → 18 → 17 → 21 → **17** (zero critical). Substantial improvement; remaining are content drift across documents.

## High

### F-P5-001 — sbctl admin exit codes cite wrong E-codes (4 mismatches)
- interface-definitions.md:108 `register-key` cites E-ADM-008 (nonce replay) + E-CFG-004 (config file not found) — neither fits "key already registered"
- :109 `revoke-key` cites E-ADM-009 (annotated "not found"); error-taxonomy:58 says E-ADM-009 = "insufficient authority"
- :110 `recover` cites E-ADM-010 (annotated "bootstrap mismatch"); error-taxonomy:59 says E-ADM-010 = "authentication failed: key not authorized"
- :117 `--yes` cites E-CFG-005; error-taxonomy:70 says E-CFG-005 = "config parse error: invalid YAML"
- Route: product-owner. Fix: add new codes E-ADM-012 (key already registered), E-ADM-013 (key not found), E-ADM-014 (bootstrap mismatch), E-CFG-006 (--yes used without --confirm); update interface-definitions.md rows.

### F-P5-002 — ARCH-07 says VP-002 rejects "version_major != 1"; current major is 0
- ARCH-07:40 vs VP-002.md:41 (current major is 0) and BC-2.01.004:56 (v0.1 = byte 0x01 = major=0).
- Route: architect. Fix: ARCH-07:40 → "version_major != 0".

### F-P5-003 — Quality threshold off-by-one at 100ms boundary (BC vs ARCH)
- BC-2.06.001:53 "Green: best path RTT p99 < 100ms" (strict). ARCH-03:192 "green: best path RTT p99 ≤ 100ms" (inclusive). At exactly 100ms, BC=yellow, ARCH=green.
- Route: product-owner + architect. Fix: pick `≤ 100ms` is green (matches NFR-001), propagate to BC-2.06.001.

### F-P5-004 — BC-2.01.005:92 test vector references `timestamp, fec_meta` fields the same BC's layout (lines 54-62) and ARCH-02:124 explicitly exclude
- Route: product-owner. Fix: replace with "chan_id, chan_seq, flags (FEC_present/ARQ_req/SACK_present), sack_bitmap (when present)".

### F-P5-005 — VP-015 module: VP-INDEX/VP-015 say internal/routing; ARCH-11/ARCH-09 say internal/frame
- ARCH-11 per-module count of internal/frame includes VP-015 in double-count.
- Route: architect. Fix: VP-015 belongs to internal/routing (router-side enforcement test). Update ARCH-11:33 and ARCH-09:37.

### F-P5-006 — PRD §2.09 header says "(CAP-026–CAP-027)" but BC-2.09.003 traces to CAP-028
- prd.md:190 stale after pass-3 CAP-028 addition.
- Route: product-owner. Fix: "### 2.09 Deployment Operations (CAP-026–CAP-028)".

### F-P5-007 — ARCH-05:138-141 says PRD §7 RTM Module column "is now populated"; PRD §7 has no Module column
- prd.md:295 RTM header is "BC ID | Source (L2 CAP) | Subsystem | Priority | Scope Phase | Test Type".
- Route: architect or product-owner. Fix: either add Module(s) column to PRD §7 from ARCH-05 BC→Package table (preferred), OR delete the ARCH-05 §"PRD Section 7 RTM" paragraph.

### F-P5-008 — VP-024 uses `checksum` alone where BC-2.02.009 / ARCH-03 require compound (checksum, arrival_interface_id)
- VP-024 statement (line 31) and harness (lines 82-94) don't include arrival_interface_id; ARCH-03 F-006 resolution requires it.
- Route: architect. Fix: rewrite VP-024 property to include arrival_interface_id and dedup-at-endpoint semantics.

## Medium

### F-P5-009 — BC-2.02.002 body Verification Properties omits VP-024
- ARCH-11:37 + VP-INDEX:50 + VP-024.md all list BC-2.02.002 as a source for VP-024; BC body table only lists VP-054, VP-025.
- Route: product-owner. Fix: add VP-024 row to BC-2.02.002 §Verification Properties.

### F-P5-010 — ARCH-11 per-module method counts wrong for internal/multipath and internal/arq
- internal/multipath ARCH-11:95 says "proptest (3), integration (1)"; VPs are VP-024 (proptest), VP-025 (proptest), VP-040 (e2e), VP-054 (integration) = proptest(2), e2e(1), integration(1).
- internal/arq ARCH-11:93 says "proptest-PE" — not a category in VP-INDEX.
- Route: architect. Fix: recompute method columns from VP-INDEX.

### F-P5-011 — architecture-feasibility-report Decision Log omits ADR-008 (tick interval)
- feasibility-report.md:124-132 lists ADR-001..007; line 29 prose says "five deferred decisions" but there are 8 ADRs.
- Route: architect. Fix: append ADR-008 row to Decision Log; refresh prose count.

### F-P5-013 — ARCH-01 frontmatter has duplicate `traces_to` key
- ARCH-01-core-services.md:10 + :19 both have `traces_to: ARCH-INDEX.md`.
- Route: architect. Fix: delete line 19.

### F-P5-014 — BC-2.05.005 / BC-2.05.006 omit HKDF `info="switchboard-frame-auth"` string
- The info string is part of the cryptographic identity; ADR-001/VP-057/ARCH-04 specify it but BC bodies don't.
- Route: product-owner. Fix: add info string to BC-2.05.005:47 and BC-2.05.006:41,59 referencing ADR-001.

### F-P5-015 — BC-2.08.001 architecture_module = `cmd/sbctl` (effectful per ARCH-09); session-mutation belongs in `internal/session`
- ARCH-09:54 says cmd/sbctl has "No business logic". BC-2.08.001 postconditions describe state changes — that's internal/session territory.
- Route: product-owner. Fix: `architecture_module: internal/session`.

### F-P5-016 — Four P1 BCs marked `criticality: critical` (P1+critical pairing needs explicit justification)
- BC-2.04.003, BC-2.04.004, BC-2.04.005, BC-2.05.004: P1 priority but critical criticality. Inconsistent with other P1 BCs (BC-2.06.001/002/003, BC-2.03.001/002/003 all P1+high).
- Route: product-owner. Fix: demote criticality to `high` OR promote priority to P0. (pending intent verification)

## Low

### F-P5-017 — ARCH-01 heading "Single Binary, Six Runtime Modes" coexists with anchor "elem-single-binary-three-modes"
- Topology has 3 modes (E/PE/P); CLI has 6 subcommands. Disambiguate.
- Route: architect. Fix: rename heading "Single Binary, Six Subcommands".

### F-P5-018 — Interface-definitions has both `# Admin` inline block (lines 92-95) and `### sbctl admin` table (lines 103-111) with the same subcommands
- Route: product-owner. Fix: consolidate into single definitive table.

## Routing Summary

| Agent | Findings |
|---|---|
| architect | F-P5-002, F-P5-005, F-P5-008, F-P5-010, F-P5-011, F-P5-013, F-P5-017 |
| product-owner | F-P5-001, F-P5-003 (BC side), F-P5-004, F-P5-006, F-P5-009, F-P5-014, F-P5-015, F-P5-016, F-P5-018 |
| both (joint decision) | F-P5-003 (architect adopts BC's choice), F-P5-007 (joint — either side fixes) |

## Verdict

**NOT_CONVERGED.** 17 findings (0 critical, 8 high, 7 medium, 2 low). Zero critical is a significant inflection — the spec is no longer at risk of producing a non-conformant binary. Remaining are content drift across documents and small mis-anchors. One more refinement round + pass 6 should reach convergence.
