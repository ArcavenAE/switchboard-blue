---
artifact_id: adv-p1-pass-03
review_target: phase-1-spec-crystallization
producer: adversary
pass: 3
fresh_context: true
findings_count: 17
findings_by_severity: {critical: 4, high: 9, medium: 3, low: 1}
findings_with_process_gap: 1
verdict: NOT_CONVERGED
timestamp: 2026-06-23
---

# Adversarial Review — Pass 3

Three families of defects: (1) HMAC tag size — canonical wire-format says 8 bytes, ubiquitous-language/VP-001/VP-005/feasibility-report say 16; (2) channel header description — entities/glossary/capabilities/BC prose mention "sender timestamp" and "FEC metadata" fields that ARCH-02 canonical layout does not contain; (3) VP source-BC mis-anchoring — VP-016/VP-027/VP-040 cite source BC titles that don't exist.

## Critical

### F-P3-001 — HMAC tag size: canonical 8 bytes, multiple downstream references say 16
- Severity: critical · Category: wire-format · Confidence: high
- Canonical: ARCH-02:79 → 8 bytes. Contradictions: ubiquitous-language.md:124 ("16-byte"), VP-001.md:72/86 (`[16]byte` harness), VP-005.md:31,49-50 (`[16]byte`), architecture-feasibility-report.md:126 ("truncated to 16 bytes").
- Route: product-owner (glossary) + architect (VPs, feasibility report).
- Fix: Update glossary to "8-byte". Regenerate VP-001/005 harness with `[8]byte`. Correct feasibility-report.md:126.

### F-P3-002 — Channel header description vs canonical layout contradiction
- Severity: critical · Category: wire-format
- ARCH-02:115-122 canonical layout: chan_id, chan_seq, flags, reserved, sack_bitmap. Contradictions (claim "sender timestamp, FEC metadata"): ubiquitous-language.md:120-121, entities.md:120-122, capabilities.md:53-55 (CAP-003), BC-2.01.005.md:40 (description vs its own layout table on lines 54-64), BC-2.01.002.md:52.
- Route: architect (decide if timestamp/FEC tag belong in channel header) + product-owner (sync descriptions).
- Fix: If they don't belong, strike "sender timestamp, FEC metadata" from all 5 sources. If they do, add to ARCH-02 layout and recompute.

### F-P3-003 — VP-001 and VP-002 disagree on current major version
- Severity: critical · Category: spec contradiction
- VP-001:32-33 + harness line 81 → "Major == 1". VP-002:41 → "Current major version is 0". ARCH-02:73 + BC-2.01.004:73 → v0.1 = byte 0x01 = major 0 minor 1.
- Route: architect.
- Fix: VP-001 generator and statement → Major = 0. VP-002:56 description → "15 invalid major nibble values (0x1–0xF)".

### F-P3-004 — BC-2.09.003 mis-anchored to CAP-023, CAP-024 (neither covers startup config validation)
- Severity: critical · Category: mis-anchoring
- BC-2.09.003 title is "Router Startup Fails Cleanly on Malformed Config." capabilities.md:206-216 CAP-023 = SVTN lifecycle; CAP-024 = sbctl CLI. Neither describes router-startup config validation (that's FM-010). BC's `subsystem: deployment-operations` is correct; the CAP anchor is rationalization, not realization.
- Route: business-analyst + product-owner.
- Fix: Add new CAP (e.g., CAP-028 "Router daemon startup-time config validation") OR extend CAP-026 to cover startup behavior. Update BC-2.09.003 frontmatter, BC-INDEX, and CAP Coverage Verification table.

## High

### F-P3-005 — VP-016 fabricates BC source titles
- VP-016:41-42 cites "HalfChannel Tick Produces Exactly One Frame" and "HalfChannel Sequence Counter Increments on Every Tick" — actual BC titles per BC-INDEX:27,29 are "Timeslice Clock Fires on Every Tick…" and "Upstream and Downstream Half-Channels Operate with Independent Clocks and Sequence Spaces."
- Route: architect.
- Fix: Update VP-016 Source Contract block with actual BC titles. Re-evaluate whether VP-016 proves the cited postconditions.

### F-P3-006 — VP-027 fabricates BC source titles AND a postcondition
- VP-027:41-43 cites BC titles that don't exist and asserts "monotonically downward transitions" invariant — neither BC-2.06.001 nor BC-2.06.002 contains this invariant.
- Route: architect + product-owner.
- Fix: Either add the no-skip invariant to BC-2.06.001 and re-cite, or re-state VP-027 property to match an actual BC postcondition (hysteresis), or re-anchor to ARCH-03's monotone-downward statement.

### F-P3-007 — VP-040 fabricates BC title + postcondition; cites NFR-003 budget as postcondition of BC-2.02.003
- VP-040:39-40 cites BC-2.02.003 with title "Multipath Path Failover and Recovery" (actual: "Per-Path RTT and Loss Tracked via Keep-Alive Probes; Paths Ranked by Quality") and postcondition "Traffic resumes ≤2s after path failure" (actual BC-2.02.003:49-56 postconditions are about EWMA, ranking, degradation flagging — no recovery-time postcondition).
- Route: architect.
- Fix: Either add a failover-recovery-time postcondition to BC-2.02.003 (or create new BC-2.02.010), or reclassify VP-040 as an NFR-validation. NFRs cannot be `source_bc:`.

### F-P3-008 — module-criticality.md graph contradicts ARCH-08 despite declaring ARCH-08 canonical
- module-criticality.md:111-114 declares ARCH-08 canonical. Lines 124, 125, 128, 129 then state imports that contradict ARCH-08:108, 114, 111, 108: e.g., module-criticality says "halfchannel imports admission, session-auth" (ARCH-08: "halfchannel imports frame"); "multipath imports halfchannel, arq, paths" (ARCH-08: "multipath imports frame, paths"); "paths no local deps" (ARCH-08: "paths imports frame").
- Route: product-owner.
- Fix: Either delete the topological build-order section (let ARCH-08 own it) or rewrite to exactly mirror ARCH-08.

### F-P3-009 — Quality indicator: BC-2.06.001 says "best path"; ARCH-03 says "any path"
- BC-2.06.001:53-54 → green/yellow thresholds derived from BEST path. ARCH-03:191 → "green→yellow: any path RTT > 100ms OR loss > 5%". Concrete scenario: 2 paths {15ms, 200ms} 0% loss → BC says green; ARCH says yellow.
- Route: architect + product-owner.
- Fix: Decide best-path vs any-path semantics. Align BC, ARCH, VP-027.

### F-P3-010 — `sbctl admin` subcommand referenced by 3 artifacts; undefined in interface-definitions.md
- ARCH-04:101,112 reference `sbctl admin` for split-brain auth and `sbctl admin recover`. BC-2.05.001:74 and BC-2.05.004:46 reference `sbctl admin`. interface-definitions.md:58-93 does NOT include `admin`.
- Route: product-owner.
- Fix: Add `sbctl admin` subcommand spec including `sbctl admin recover`, bootstrap-key model, confirmation-token flow, E-codes.

### F-P3-011 — architecture-feasibility-report.md:126 records HMAC tag as "truncated to 16 bytes" — contradicts ARCH-02 (8 bytes)
- Route: architect.
- Fix: Update to "truncated to 8 bytes." Compounds F-P3-001.

### F-P3-012 — BC-2.01.007 lives in session-networking but architecture_module = internal/admission (registered to SS-05)
- ARCH-INDEX:78 SS-01 modules: internal/frame, internal/halfchannel. ARCH-INDEX:82 SS-05 modules: internal/hmac, internal/admission, internal/session.
- Route: architect + product-owner.
- Fix: Move BC-2.01.007 to subsystem `admission-security`, OR add `internal/admission` to SS-01 module list. Update PRD §2 BC tables, BC-INDEX, ARCH-11.

### F-P3-013 — BC-2.02.008 split-horizon: subsystem multipath-forwarding, module internal/routing (registered to SS-05)
- Same defect class as F-P3-012.
- Route: architect.
- Fix: Add `internal/routing` to SS-02 module list OR move BC. Likely the former — routing is genuinely cross-subsystem.

## Medium

### F-P3-014 — BC-2.02.009 has two distinct edge cases both labeled EC-004
- BC-2.02.009:74,76 — EC-004 appears twice (separated by EC-005 on line 75). Second should be EC-006.
- Route: product-owner.
- Fix: Renumber.

### F-P3-015 — ADR OQ-003 reference confusion
- ARCH-04:45 has "OQ-003 resolution:" introducing ADR-003 (LWW for duplicate registration). ARCH-04:62 has "(OQ-001, OQ-002, OQ-003, F-010)" for ADR-004 (permission hierarchy). invariants.md:158 OQ-003 is about permission hierarchy — ADR-004 resolves it. ADR-003 resolves DEC-007, not OQ-003.
- Route: architect.
- Fix: Rename ARCH-04:45 "OQ-003 resolution:" → "DEC-007 resolution:". Add explicit "OQ-004 resolution:" to ADR-005 in ARCH-03.

### F-P3-016 — VP-002:56 description self-contradicts harness (claims 16 values incl. 0x0; harness covers 15 values 0x1–0xF)
- Route: architect.
- Fix: Line 56 → "15 distinct major nibble values (0x1–0xF)" consistent with line 46 and the generator.

## Low

### F-P3-017 — BC-2.05.004 subsystem admission-security but module internal/svtnmgmt (registered to SS-07)
- Same cross-cutting concern as F-P3-012/013; tagged LOW because key lifecycle is conceptually admission, mechanically network-management.
- Route: product-owner.
- Fix: Either move BC, add module to SS-05, or document cross-cutting.

## Process Gap

### F-P3-018 [process-gap] — Two prior passes did not catch VP-source-BC title drift (F-P3-005/006/007) or BC-CAP anchor mismatch (F-P3-004)
- Severity: medium (process)
- Findings detectable by mechanical title-equality / capability-paragraph match.
- Route: orchestrator.
- Fix: Add Phase 1d sanity check — for every VP, assert Source Contract title matches BC-INDEX title for `source_bc:`. For every BC, assert declared `capability:` actually has BC realization in capabilities.md CAP-NNN paragraph.

## Routing Summary

| Agent | Findings |
|---|---|
| architect | F-P3-001 (VPs+feasibility), F-P3-002 (decision), F-P3-003, F-P3-005, F-P3-006, F-P3-007, F-P3-011, F-P3-012, F-P3-013, F-P3-015, F-P3-016 |
| product-owner | F-P3-001 (glossary), F-P3-002 (descriptions sync), F-P3-008, F-P3-009, F-P3-010, F-P3-014, F-P3-017 |
| business-analyst | F-P3-004 (new CAP) |
| orchestrator | F-P3-018 [process-gap] — codify into rules |

## Verdict

**NOT_CONVERGED**. HMAC tag size and channel header layout are blocking — wave-1 implementation would build a non-conformant binary. Mis-anchoring means VP coverage doesn't actually map to BC postconditions. Trajectory: 27 → 18 → 17.
