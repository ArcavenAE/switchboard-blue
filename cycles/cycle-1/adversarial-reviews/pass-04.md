---
artifact_id: adv-p1-pass-04
review_target: phase-1-spec-crystallization
producer: adversary
pass: 4
fresh_context: true
findings_count: 21
findings_by_severity: {critical: 4, high: 9, medium: 6, low: 2}
findings_with_process_gap: 1
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 4

## Critical

### F-P4-001 — PRD §7 RTM maps BC-2.09.003 to CAP-023/024 (stale; should be CAP-028)
- prd.md:338 still shows pre-CAP-028 mapping. capabilities.md §CAP-028:248 says "Realized by: BC-2.09.003". BC-2.09.003 frontmatter `capability: CAP-028`. PRD §7 not propagated.
- Route: product-owner. Fix: PRD line 338 column 2 → CAP-028.

### F-P4-002 — ARCH-04:98 uses E-ADM-007 for role-hierarchy revocation; correct code is E-ADM-011
- error-taxonomy.md:60 defines E-ADM-011 for hierarchy violation. E-ADM-007 (line 56) is bound to BC-2.04.005 read-only console rejection. BC-2.05.004:85 uses E-ADM-011 correctly; ARCH-04 still uses E-ADM-007.
- Route: architect. Fix: ARCH-04:98 E-ADM-007 → E-ADM-011.

### F-P4-003 — interface-definitions.md `sbctl admin` cites wrong error codes
- Line 108 register-key cites E-ADM-008 (= "nonce replay"); line 109 revoke-key cites E-ADM-009 (= "insufficient authority") for "not found"; line 110 recover cites E-ADM-010 (= "auth failed") for "bootstrap mismatch"; line 113 cites E-CFG-005 (= "config parse error") for `--yes` bypass warning. None of these match error-taxonomy semantics.
- Route: product-owner. Fix: Allocate new codes (E-ADM-012 nonce-replay-on-admin, E-ADM-013 key-not-found, E-ADM-014 bootstrap-mismatch, E-CFG-006 confirm-bypass) OR reuse only where canonical semantics fit.

### F-P4-004 — BC-2.06.001 "best path" vs ARCH-03 "any path" quality contradiction
- BC-2.06.001:53: "Green: best path RTT p99 < 100ms AND loss < 5%." ARCH-03:192: "green → yellow: any path RTT > 100ms OR loss > 5%". Same observable state yields different verdicts. Pass-3 added BC-2.06.001 invariant 4 clarifying "aggregated indicator uses BEST path" — but ARCH-03 wasn't updated to match.
- Route: architect + product-owner. Fix: ARCH-03:192 → "best path > 100ms OR best path loss > 5%". Or pick any-path semantic and update BC.

## High

### F-P4-005 — Systemic VP source-contract title drift from BC-INDEX (≥27 VPs)
- Pattern: VP source-contract entries quote BC titles that don't match BC-INDEX. Samples: VP-018, VP-024, VP-028, VP-029, VP-030, VP-026, VP-021, VP-044, VP-022 + ~18 more.
- Route: architect. Fix: Mechanical sweep — rewrite each VP "Source Contract — BC:" title to match BC-INDEX H1 verbatim.

### F-P4-006 — VP-028/029 cite postconditions that don't exist in BC-2.09.003
- BC-2.09.003 postconditions are about exit codes / stderr / clean shutdown; VPs claim "tick_interval range" / "missing required field" postconditions. Neither exists in BC-2.09.003.
- Route: architect. Fix: Add the postconditions to BC-2.09.003 (or create BC-2.09.004 for general Config.Validate guarantees) OR re-source the VPs.

### F-P4-007 — Error code count drift (task brief said "32 codes"; catalog has 30)
- error-taxonomy.md actual: ADM 11 + CFG 5 + NET 6 + PRT 3 + FWD 1 + SES 1 + SVTN 2 + SYS 1 = 30.
- Route: product-owner. Fix: Add explicit totals table; reconcile count claim.

### F-P4-008 — `sbctl router routes` and `sbctl router frames` referenced by BCs but undefined in interface-definitions.md
- BC-2.02.008:70 uses `sbctl router routes`. BC-2.01.005:94 uses `sbctl router frames`. interface-definitions.md §sbctl router lists only status/metrics/reload/drain.
- Route: product-owner. Fix: Add `sbctl router routes [--svtn=<id>]` and `sbctl router frames [--svtn=<id>]` OR replace BC references with `sbctl router status`.

### F-P4-009 — ARCH-11 per-module VP counts disagree with VP-INDEX in 5 entries
- internal/frame, internal/halfchannel, internal/paths, internal/metrics, internal/config all show drift from VP-INDEX. Bottom-line total = 57 only by cancellation.
- Route: architect. Fix: Recompute ARCH-11 per-module table mechanically from VP-INDEX.

### F-P4-010 — ARCH-07 VP method declarations diverge from VP-INDEX
- VP-005 (ARCH-07: "proptest/fuzz" vs VP-INDEX: "fuzz"); VP-028 (ARCH-07: "unit" vs VP-INDEX: "proptest").
- Route: architect. Fix: ARCH-07 method column = VP-INDEX for every row.

### F-P4-011 — BC-2.01.005:92 test vector references `timestamp` and `fec_meta` fields ARCH-02 excluded
- BC-2.01.005's own layout table on lines 56–64 is canonical (no timestamp, no fec_meta); the test vector on line 92 stayed stale.
- Route: product-owner. Fix: Rewrite line 92 test vector to "chan_id, chan_seq, flags (incl. sack_present), sack_bitmap when set, correctly extracted".

### F-P4-012 — BC-2.05.001 body VP list omits VP-008
- VP-INDEX:34 + ARCH-11:54 confirm VP-008 covers BC-2.05.001 (alongside BC-2.05.002). BC body lists only VP-007/009.
- Route: product-owner. Fix: Add VP-008 row to BC-2.05.001 §Verification Properties.

### F-P4-013 — BC-2.03.003 frontmatter capability: single-valued (CAP-011); body and BC-INDEX list two
- Frontmatter line 13: `capability: CAP-011`. Line 31: `traces_to: [CAP-011, CAP-012]`. Body and BC-INDEX list both.
- Route: product-owner. Fix: Make capability field list-valued.

## Medium

### F-P4-014 — VP-001 generator uses uint32 for payload_len; ARCH-02 says u16 big-endian
- VP-001:34 "PayloadLen is in [0, 2^32-1]"; line 71 generator `gen.UInt32()`. ARCH-02:75 says u16 big-endian (max 65,535).
- Route: architect. Fix: VP-001 → uint16 in property statement and generator.

### F-P4-015 — VP-003 references "major protocol version 1" but current major is 0
- Route: architect. Fix: VP-003:33 → "major protocol version 0 (v0.x)".

### F-P4-016 — invariants.md OQ-001/002/003/004 listed as open; ADR-004/005 resolved them
- Route: business-analyst. Fix: Add resolution column or note explicitly tying each OQ to the resolving ADR.

### F-P4-017 — module-criticality.md "Total: 18" but lists 17 modules; missing cmd/switchboard
- Route: product-owner. Fix: Add cmd/switchboard row (LOW) OR reduce total to 17.

### F-P4-018 — module-criticality.md VP Count column all TBD; data is in VP-INDEX [process-gap]
- Route: product-owner. Fix: Backfill VP Count column. [process-gap] Codify VP-INDEX → derivative-doc propagation rule.

### F-P4-019 — architecture-feasibility-report.md:58 lists "admission-security (CAP-017–020)" omitting CAP-020a/020b
- Route: architect. Fix: → "(CAP-017–020, CAP-020a, CAP-020b)".

## Low

### F-P4-020 — BC-2.03.003 references VP-045 but VP-045 source_bc is BC-2.03.002 only
- Route: architect. Fix: Either remove VP-045 from BC-2.03.003 OR amend VP-045 frontmatter to `[BC-2.03.002, BC-2.03.003]`.

### F-P4-021 — BC-2.01.001 lists VP-016/018/041/042 in VP Anchors; ARCH-11 maps only VP-016/018
- Route: architect. Fix: Decide whether benchmark VPs count as anchors; apply consistently.

## Verdict

**NOT_CONVERGED.** Trajectory: 27 → 18 → 17 → 21. The reversal signals fresh-context audits keep surfacing pre-existing defects (PRD §7, best-path/any-path, VP title drift, u16/u32 in VP-001) AND each refinement round leaves new propagation gaps (E-ADM-011 incomplete, ARCH-11 counts). Linear incremental refinement is not converging this spec.
