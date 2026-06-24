---
artifact_id: adv-p1-pass-06
review_target: phase-1-spec-crystallization
producer: adversary
pass: 6
fresh_context: true
findings_count: 14
findings_by_severity: {critical: 0, high: 7, medium: 6, low: 1}
findings_with_process_gap: 0
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 6

**Trajectory:** 27 → 18 → 17 → 21 → 17 → **14**. Zero critical for second consecutive pass.

## High

### F-006-001 — Systematic priority drift for BC-2.04.003/004/005: PRD+ARCH-11 say P1, BC-INDEX+BC-file say P0
- prd.md:141-143, :318-320 → P1
- ARCH-11.md:50-52 → P1
- BC-INDEX.md:48-50 → P0
- BC files frontmatter → P0
- capabilities.md CAP-014/015 → P1
- Pattern: pass-5 F-P5-016 promoted BCs P1→P0 in frontmatter + BC-INDEX, did not propagate to PRD or ARCH-11.
- Route: product-owner. Fix: harmonize all five views.

### F-006-002 — BC-2.05.004 priority drift (same pattern as 001)
- prd.md:155, :325 + ARCH-11:57 → P1
- BC-INDEX:55 + BC frontmatter → P0
- capabilities.md CAP-019 → P1; VP-046 (VP-INDEX:72) → P1
- Route: product-owner. Fix: decide canonical priority and propagate.

### F-006-003 — BC-2.05.004 description vs precondition: internal contradiction
- BC-2.05.004.md:42 description: "Control nodes **and admitted console nodes** can manage..."
- Same file line 46 Precondition 1: "Per ADR-004: key management is exclusive to the control node. Console nodes have read-only access..."
- Route: product-owner. Fix: remove "and admitted console nodes" from description.

### F-006-004 — BC-2.06.001 Postcondition 4 vs Invariant 4: all-paths vs best-path contradiction
- BC-2.06.001.md:55 Postcondition 4: "Red: **all paths** RTT p99 > 500ms OR loss > 20% OR no paths available"
- BC-2.06.001.md:63 Invariant 4: "session-aggregated quality indicator is derived from the BEST current path"
- ARCH-03:194: "red: best path RTT p99 > 500ms"
- Route: product-owner. Fix: Postcondition 4 → "Red: best path RTT p99 > 500ms OR best path loss > 20% OR no paths available."

### F-006-005 — error-taxonomy severity↔exit-code rule violated: E-ADM-011/012/013/014 are "broken" severity with exit 0
- error-taxonomy.md:41 severity rule: "broken | ... | Non-zero exit"
- error-taxonomy.md:60-63 E-ADM-011/012/013/014 → broken / 0 (continues)
- Operator scripts cannot detect sbctl admin failures via exit code.
- Route: product-owner. Fix: set exit code 1 for E-ADM-011 through E-ADM-014 OR clarify daemon-side "continues" vs client-side exit semantics in the severity definitions.

### F-006-006 — `--yes` / `--confirm` flag interaction self-contradictory
- interface-definitions.md:109: "`--yes` ... Cannot be combined with `--confirm` (usage error, exit 2)"
- interface-definitions.md:111: "`--yes` bypasses the check entirely with a stderr warning (E-CFG-006)"
- error-taxonomy.md:74 E-CFG-006 message: "`--yes` used WITHOUT `--confirm` target: specify `--confirm=<svtn-short-id>` or omit `--yes`"
- The interaction rule and the error message describe mutually exclusive failure conditions.
- Route: product-owner. Fix: pick (a) `--yes` alone is valid (rewrite E-CFG-006 to fire on `--yes`+`--confirm`), or (b) `--yes` requires `--confirm` (rewrite interaction rule).

### F-006-007 — ARCH-11 per-module VP count diverges from VP-INDEX (frame: 3 vs 4; routing: 5 vs 4); arithmetic self-balances and masks the error
- ARCH-11:90: `internal/frame | 3 | proptest (3)` — VP-INDEX has VP-001/002/003/014 = 4
- ARCH-11:99: `internal/routing | 5 | proptest (2), fuzz+audit (2), e2e (1)` — VP-INDEX has VP-010/011/015/039 = 4
- ARCH-11 BC table omits VP-002 from BC-2.01.004 row (VP-INDEX:28 confirms VP-002 has source_bc=BC-2.01.004)
- Route: architect. Fix: ARCH-11 frame→4, routing→4; re-add VP-002 to BC-2.01.004 row.

## Medium

### F-006-008 — module-criticality.md mis-locates drop cache in `routing` (canonical: `multipath`)
- module-criticality.md:44 says "routing ... drop cache"
- ARCH-03:48, ARCH-05:52, BC-2.02.009.md:12, VP-INDEX:51 all say drop cache is in `internal/multipath`
- Route: product-owner. Fix: move drop cache from routing description to multipath description.

### F-006-009 — BC-2.09.003 traces to DI-007 (outer header stability) — wrong invariant
- BC-2.09.003 is about daemon startup config validation (CAP-028); DI-007 is about wire-format stability
- Route: product-owner. Fix: remove DI-007 reference or replace with FM-010 (which BC-2.09.003 already cites as primary anchor).

### F-006-010 — architecture-feasibility-report:61 says "deployment-operations (CAP-026–027)" — omits CAP-028
- Pass-3 added CAP-028; feasibility-report not updated
- Route: architect. Fix: "(CAP-026–028)".

### F-006-011 — Pervasive VP↔BC reverse-trace asymmetry: multi-BC VPs propagated to one BC only
- VP-012 (sources BC-2.05.003 + BC-2.04.003): BC-2.04.003 omits VP-012
- VP-016 (sources BC-2.01.001 + BC-2.01.003): BC-2.01.003 omits VP-016
- VP-018 (sources BC-2.01.001 + BC-2.01.002): BC-2.01.002 omits VP-018
- VP-008 (sources BC-2.05.001 + BC-2.05.002): BC-2.05.001 omits VP-008
- VP-042 (sources BC-2.01.001 + BC-2.02.001): BC-2.02.001 omits VP-042
- Route: product-owner. Fix: mechanical sweep — for each VP whose source_bc has >1 BC, ensure every cited BC's VP table contains the VP.

### F-006-012 — `sbctl debug --export-keys` referenced in BC-2.05.007:68 EC-001 but not defined in interface-definitions.md
- Route: product-owner. Fix: either define `sbctl debug` OR rewrite EC-001 as a hypothetical ("Operator attempts a key-export operation; no such CLI exists per BC-2.05.007").

### F-006-013 — `sbctl svtn keys ...` (interface-definitions:64-67) and `sbctl admin ...key` (line 102) duplicate with flag-name drift
- `sbctl svtn keys register --key=<path>` vs `sbctl admin register-key --pubkey <path>` — same operation, different flags
- Route: product-owner. Fix: either make `sbctl admin` canonical and remove `sbctl svtn keys register/revoke/expire`, OR document the relationship (sbctl svtn keys = non-control-role ops without --confirm gating). Resolve flag drift.

## Low

### F-006-014 — BC-2.07.003 trace to DI-002 is tangential (private keys never transit)
- BC-2.07.003 is about clear connection-error reporting; DI-002 applies indirectly (don't leak keys in errors)
- Route: product-owner. Fix: consider removing or accept as is.

## Verdict

**NOT_CONVERGED.** Two NEW defect classes introduced by pass-5 fixes (F-006-005 broken-severity rule violation enabled by new admin error codes; F-006-006 --yes/--confirm contradiction enabled by new admin UX). Plus a systematic priority drift (F-006-001/002) where pass-5 promoted BCs in some files but not all five views.

Trajectory: 27 → 18 → 17 → 21 → 17 → 14. Linear refinement is hitting diminishing returns.
