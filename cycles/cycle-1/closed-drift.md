# Closed Drift Items — cycle-1

Resolved drift items archived here to keep STATE.md under the 200-line limit.
Pointer in STATE.md: `cycles/cycle-1/closed-drift.md`

## Resolved Items

| ID | Severity | Description | Owner | Resolved |
|----|----------|-------------|-------|----------|
| WG3-TAX-001 | — | Wave 3 gate audit found retired/incorrect error codes in holdout + story specs: wave-3.md HS-003 cited retired E-SES-005 (→E-ADM-007, must-pass fix); wave-5.md revoke-not-found cited E-ADM-007 (→E-ADM-013) and revoked-key re-admission cited E-ADM-002 (→E-ADM-005); S-6.02 EC-002 cited E-ADM-007 for key-not-found (→E-ADM-013). All corrected against error-taxonomy.md v1.6 canonical codes. | consistency-validator/product-owner/story-writer | RESOLVED 2026-06-26 |
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
| W3-PG-001 | LOW | Security-perimeter default-polarity inconsistency — candidate go.md rule. | rules/governance | open — cycle-close |
| F-P8-004/005 | MED | VP-026 "transitivity" invariant missing from BC-2.02.003; VP-027 title/harness direction mismatch. | architect | open — Phase 3 test-writing |

## Stable-Deferred Phase-6 Hardening Items (archived from STATE.md 2026-06-27)

These items have no active work pending before Phase 6. Archived to keep STATE.md ≤200 lines.

| ID | Severity | Description | Owner | Status |
|----|----------|-------------|-------|--------|
| VP-036 testenv | Phase-6 hardening | property test (TestProperty_VP036_SessionContinuity) deferred until internal/testenv.ConnectWithSourceIP exists | — | deferred to Phase 6 |
| SEC-003 | Phase-6 hardening | Sub-microsecond TOCTOU on now in ReAuthenticate; accepted per pr-reviewer PR #7 security review | — | accepted/deferred Phase 6 |
| WAVE-2-MED-001 | Phase-6 hardening | ReAuthState not evicted on RevokeKey/RegisterKey reset; stale source-IP survives via CurrentSourceAddr | — | deferred to Phase 6 |
| VP-039-test-skip | Phase-6 hardening | t.Skip placeholder needed in internal/routing/*_test.go for VP-039 (deferred property test) | — | deferred to Phase 6 |

## Pre-Restart Wave 3 Adversary Passes (superseded by restart run at 10dd880)

Prior run (before PR #15 fix): pass-01 CONVERGED (0C/0H/3M/2L/3O), pass-02 NOT_CONVERGED
(HIGH: E-ADM-016 not logged — now resolved), pass-03 NOT_CONVERGED (HIGH: same F-1 —
now resolved). Reports: `cycles/cycle-1/wave-3/adversary/pass-01.md`,
`pass-02.md`, `pass-03.md`. All superseded; restart run begins at 10dd880.
