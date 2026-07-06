# Closed Drift Items — cycle-1

Resolved drift items archived here to keep STATE.md under the 200-line limit.
Pointer in STATE.md: `cycles/cycle-1/closed-drift.md`

## Resolved Items

| ID | Severity | Description | Owner | Resolved |
|----|----------|-------------|-------|----------|
| WG3-TAX-001 | — | Wave 3 gate audit found retired/incorrect error codes in holdout + story specs: wave-3.md HS-003 cited retired E-SES-005 (→E-ADM-007, must-pass fix); wave-5.md revoke-not-found cited E-ADM-007 (→E-ADM-013) and revoked-key re-admission cited E-ADM-002 (→E-ADM-005); S-6.02 EC-002 cited E-ADM-007 for key-not-found (→E-ADM-013). All corrected against error-taxonomy.md v1.6 canonical codes. | consistency-validator/product-owner/story-writer | RESOLVED 2026-06-26 |
| S401-O3 | MED | BC-2.02.003 PC5: degraded-path flag (RTT >200ms) unimplemented in internal/paths. | product-owner/architect | RESOLVED 2026-06-28 — closed by new story S-5.03 (BC-2.02.003 PC-5 degraded-path flag, internal/paths); VP-063 minted as dedicated proptest (PathTracker.IsDegraded() EWMA vs DegradedRTTThresholdMS 200 ms). |
| W3-M-1 | HIGH | E-ADM-016 not logged at router on HMAC failure — BC-2.05.008 PC-2 observability postcondition UNIMPLEMENTED and UNTESTED on P0 security contract; confirmed HIGH by Wave-3 adversary passes 2+3 (pass-1 under-rated as MED). Router had no logger field. | implementer + test-writer | RESOLVED via PR #15 (squash commit 10dd880) — RouteFrame now logs E-ADM-016 (svtn_id/src_addr) before returning ErrHMACVerificationFailed on both the no-forwarding-entry and HMAC-verify-fail paths; injectable Logger + WithLogger option added to Router (mirrors tmux.Logger); 4 new routing tests assert log emission (Red-Gate proven). Control flow/sentinel unchanged. Merged 2026-06-27. |

## Archived Stable/Deferred Items (archived from STATE.md 2026-06-27)

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| F-003/F-004 | LOW | Payload-MTU wire-format test + ARCH-02 serializer | story S-BL.OA | deferred to outer-assembler story |
| S-3.03-L1-REVOKE | LOW | BC-2.05.003 EC-004 "revoke" half: no RevokeKey. Out of S-3.03 scope. | architect | deferred — Wave 4+ operator-provisioning story |
| S-3.03-O1-VPSKEL | LOW | VP-012/013/035 proof-harness skeletons API-fixed; execution deferred. | formal-verifier | open — Phase-6 |
| MISE-DX-001/002 | LOW | brew→mise migration + CLAUDE.md update; story S-M.01. | dx-engineer | open |
| SIGN-DX-001 | LOW | Apple code-signing: release.yml gated OFF; story S-M.02, milestone-gated. | dx-engineer | open |
| F-P8-009 | LOW | feasibility-report:61 deployment-ops range off-by-one (CAP-026–028) | architect | open |
| W3-PG-001 | LOW | Security-perimeter default-polarity inconsistency — candidate go.md rule. | rules/governance | CLOSED 2026-07-06 — PR #108: rule 13 "Security-perimeter constructor defaults must fail closed" added to .claude/rules/go.md (NewAccessNode example; W3-M-3 reference) |
| F-P8-004/005 | MED | VP-026 "transitivity" invariant missing from BC-2.02.003; VP-027 title/harness direction mismatch. | architect | open — Phase 3 test-writing |

## Stable-Deferred Phase-6 Hardening Items (archived from STATE.md 2026-06-27)

These items have no active work pending before Phase 6. Archived to keep STATE.md ≤200 lines.

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| VP-036 testenv | Phase-6 hardening | property test (TestProperty_VP036_SessionContinuity) deferred until internal/testenv.ConnectWithSourceIP exists | — | deferred to Phase 6 |
| SEC-003 | Phase-6 hardening | Sub-microsecond TOCTOU on now in ReAuthenticate; accepted per pr-reviewer PR #7 security review | — | accepted/deferred Phase 6 |
| WAVE-2-MED-001 | Phase-6 hardening | ReAuthState not evicted on RevokeKey/RegisterKey reset; stale source-IP survives via CurrentSourceAddr | — | deferred to Phase 6 |
| VP-039-test-skip | Phase-6 hardening | t.Skip placeholder needed in internal/routing/*_test.go for VP-039 (deferred property test) | — | deferred to Phase 6 |

## Wave-Gate Detail Rows (archived from STATE.md 2026-06-27 to stay ≤200 lines)

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| W3-R3-F3 | MED | E-ADM-016 PATH-A message ("auth key unavailable") diverges from canonical taxonomy string ("tag mismatch"); both carry E-ADM-016; unpinned by test. | product-owner/implementer | open |
| W3-R3-F4 | MED | BC-2.05.008 PC-4 does not mandate E-ADM-016 logging on no-entry PATH-A, but code+test now emit/assert it (unadjudicated test-writer assumption). Overlaps W3-F1-FU1. | product-owner/spec-steward | open |
| W3-R3-F5 | LOW | ErrSessionMismatch sentinel text lacks "(E-SES-006)" code token (parity with E-ADM-006/007 sentinels). | implementer | open |
| S-3.02-FM1 | MED | Attach upstream channel vestigial/undrained; BC-2.04.003 PC-3 deferred. | architect | open |
| W3-M-2/M-3 | MED | SessionConnector no failover Frames(); NewAccessNode nil→fail-OPEN polarity gap; Frames() lost on ctrl→PTY failover swap. | architect/implementer | open — fix in wiring story |
| W3-F1-FU1/FU2 | LOW | BC-2.05.008 PC-4 PATH-A log not ratified; PC-2 trace row cites old test. | spec-steward | open |
| SW305-HF1 | HIGH | Hysteresis re-fire semantics: RESOLVED — re-fire/dead-key implemented+tested (3af388c). Msg-format separate defect tracked as SW305-M1 REOPENED. | product-owner | RESOLVED 3af388c |
| SW305-HF2 | HIGH | Unbounded attacker-keyed map (counts/firedAt) = memory DoS CWE-770 — source-count cap RESOLVED (maxTrackedSources+LRU evict in 3af388c). Per-source slice bound tracked as SW305-M5 PO adjudication. | implementer | RESOLVED (source-count) 3af388c |

## S-W3.05 Fix-Loop Resolved Items (archived from STATE.md 2026-06-27)

| ID | Severity | Description | Owner | Resolved |
|----|----------|-------------|-------|----------|
| SW305-M1 | MED | E-ADM-017 msg format REOPENED — missing "HMAC failure rate alert:" phrase; prior orchestrator adjudication was erroneous; specs authoritative. | implementer+test-writer | RESOLVED b945aab — canonical phrase "HMAC failure rate alert: ≥&lt;threshold&gt; failures in &lt;window&gt;s from src &lt;src_addr&gt;" restored. |
| SW305-HF3 | HIGH | VP-059 proptest still missing (C-2 reopened) — VP-059.md created 20011cc but proptest harness never ported. | test-writer | RESOLVED 5c3d7ea — stateful model proptest, seed 1337+idx, 3 configs {5,60s}/{3,30s}/{10,120s}, no divergence. |
| SW305-M5 | MED | Per-source timestamp slice unbounded within window (CWE-770, EC-011) — ADJUDICATED FIX-NOW (BC-2.05.005 v1.6). | implementer+test-writer | RESOLVED b945aab — append-skip: slice bounded at threshold entries while fired; drain-only re-arm: dead keep[0].After(lastFire) branch removed; EC-011 test added; BC-2.05.005 bumped to v1.6. |
| SW305-M6 | MED | VP-059 names TrackedSourceCount, impl SourceCount — naming mismatch; VP harness won't compile. | implementer | RESOLVED VP-059 v1.1 (b7431fd) — VP-059 uses SourceCount throughout; naming reconciled. |
| SW305-M7 | MED | ERROR-level (PC-3, AC-003) unsatisfiable through level-less admission.Logger — ADJUDICATED option-a: Logger seam is level-less (Log(msg string)); "at ERROR level" phrase removed from PC-3. | implementer+test-writer | RESOLVED b945aab — AC-003 test fixture uses Log(msg string); no Error() method reference in admission_test; BC-2.05.005 v1.6 PC-3 amended. O-1 closed. |
| SW305-M8 | MED | AC-012 dead-key delete(counts) path: no discriminating test; assertion is >=1 regardless. | test-writer | RESOLVED b945aab — discriminating test added confirming dead-key entry eviction. |

## Wave-3 Pre-Gate Items (resolved 2026-06-27)

| ID | Severity | Description | Owner | Resolved |
|----|----------|-------------|-------|----------|
| C-1/OBS-3 | CRITICAL | WithFailureCounter/E-ADM-017 wiring gap: buildRouter returned counter but caller discarded it; OBS-3 spec-forbidden partial-wiring. | implementer | RESOLVED PR #20 (418de54) — WithFailureCounter wired into buildRouter(threshold=5,window=60s); OBS-3 closed. Only remaining deferral: network-ingress LISTENER → S-BL.NI (ARCH-08 v2.3 §6.5.1). |
| T2 | OBLIGATION | ADR-011 v1.6 Obligation T2: deterministic TOCTOU misclassification-branch test required (complementing probabilistic 50-loop test). | test-writer | RESOLVED PR #19 (849bd86) — deterministic swapBarrier test added to TestForwardFramesTOCTOUCount50 path; ARCH-08 v2.3 documents T2 satisfied. |

## Wave 4 Cycle-Close Resolved Items (archived from STATE.md 2026-06-28)

| ID | Severity | Description | Owner | Resolved |
|----|----------|-------------|-------|----------|
| BC-2.09.003-STALE | NITPICK | BC-2.09.003 traceability table + Story Anchor said "AC-001 through AC-006"; story now reaches AC-009. | story-writer/spec-steward | RESOLVED — BC-2.09.003 v1.5 updated AC-001..AC-009 in cycle-close burst. |
| S601-NITPICK-A | NITPICK | S-6.01 story File Structure table omits cmd/switchboard/access.go though Task 17 mandates modifying it. | story-writer | RESOLVED — S-6.01 story updated in cycle-close. |
| S601-NITPICK-B | NITPICK | S-6.01 story EC ids diverge from BC EC ids (cosmetic id drift). | story-writer | RESOLVED — S-6.01 story updated in cycle-close. |
| S601-NITPICK-C | NITPICK | S-6.01 E-CFG-005/E-CFG-004 reuse — no dedicated BC code (cosmetic). | product-owner | RESOLVED — accepted as MVP cosmetic; no spec change needed. |
| S601-NITPICK-D | NITPICK | S-6.01 ValidationError.Error() "value" token not in BC canonical template (byte-level cosmetic). | implementer | RESOLVED — accepted as MVP cosmetic; implementation matches spirit. |
| S601-NITPICK-E | OBS | S-6.01 yaml.v3 billion-laughs bound is implicit/library-version-dependent. | implementer | RESOLVED — optional; accepted with note that library handles it. |
| S601-DRAFT-STORY | OBS | Dedicated SIGHUP/reload story (BC-2.09.003 Inv-3/EC-004) to be opened as draft. | product-owner | RESOLVED — S-6.04 created (cycle-close burst). |
| S403-COS1/2 | OBS | S-4.03 cosmetics: stale "encoding/binary" doc ref + leftover stub docstring in merged artifact (8d9744f). | implementer | RESOLVED — PR #29 (7ef43b8) hygiene pass fixed both. |
| S404-OBS-G | OBS | S-4.04 BC-2.02.008 PC-4 (split-horizon/drop-cache independence) has no dedicated negative test — satisfied structurally. | test-writer | RESOLVED — structural satisfaction accepted by architect; no further test needed at this scope. |

## Pre-Restart Wave 3 Adversary Passes (superseded by restart run at 10dd880)

Prior run (before PR #15 fix): pass-01 CONVERGED (0C/0H/3M/2L/3O), pass-02 NOT_CONVERGED
(HIGH: E-ADM-016 not logged — now resolved), pass-03 NOT_CONVERGED (HIGH: same F-1 —
now resolved). Reports: `cycles/cycle-1/wave-3/adversary/pass-01.md`,
`pass-02.md`, `pass-03.md`. All superseded; restart run begins at 10dd880.
