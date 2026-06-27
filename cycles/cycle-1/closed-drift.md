# Closed Drift Items — cycle-1

Resolved drift items archived here to keep STATE.md under the 200-line limit.
Pointer in STATE.md: `cycles/cycle-1/closed-drift.md`

## Resolved Items

| ID | Severity | Description | Owner | Resolved |
|----|----------|-------------|-------|----------|
| WG3-TAX-001 | — | Wave 3 gate audit found retired/incorrect error codes in holdout + story specs: wave-3.md HS-003 cited retired E-SES-005 (→E-ADM-007, must-pass fix); wave-5.md revoke-not-found cited E-ADM-007 (→E-ADM-013) and revoked-key re-admission cited E-ADM-002 (→E-ADM-005); S-6.02 EC-002 cited E-ADM-007 for key-not-found (→E-ADM-013). All corrected against error-taxonomy.md v1.6 canonical codes. | consistency-validator/product-owner/story-writer | RESOLVED 2026-06-26 |
| W3-M-1 | HIGH | E-ADM-016 not logged at router on HMAC failure — BC-2.05.008 PC-2 observability postcondition UNIMPLEMENTED and UNTESTED on P0 security contract; confirmed HIGH by Wave-3 adversary passes 2+3 (pass-1 under-rated as MED). Router had no logger field. | implementer + test-writer | RESOLVED via PR #15 (squash commit 10dd880) — RouteFrame now logs E-ADM-016 (svtn_id/src_addr) before returning ErrHMACVerificationFailed on both the no-forwarding-entry and HMAC-verify-fail paths; injectable Logger + WithLogger option added to Router (mirrors tmux.Logger); 4 new routing tests assert log emission (Red-Gate proven). Control flow/sentinel unchanged. Merged 2026-06-27. |

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
