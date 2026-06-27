# Closed Drift Items — cycle-1

Resolved drift items archived here to keep STATE.md under the 200-line limit.
Pointer in STATE.md: `cycles/cycle-1/closed-drift.md`

## Resolved Items

| ID | Severity | Description | Owner | Resolved |
|----|----------|-------------|-------|----------|
| WG3-TAX-001 | — | Wave 3 gate audit found retired/incorrect error codes in holdout + story specs: wave-3.md HS-003 cited retired E-SES-005 (→E-ADM-007, must-pass fix); wave-5.md revoke-not-found cited E-ADM-007 (→E-ADM-013) and revoked-key re-admission cited E-ADM-002 (→E-ADM-005); S-6.02 EC-002 cited E-ADM-007 for key-not-found (→E-ADM-013). All corrected against error-taxonomy.md v1.6 canonical codes. | consistency-validator/product-owner/story-writer | RESOLVED 2026-06-26 |
| W3-M-1 | HIGH | E-ADM-016 not logged at router on HMAC failure — BC-2.05.008 PC-2 observability postcondition UNIMPLEMENTED and UNTESTED on P0 security contract; confirmed HIGH by Wave-3 adversary passes 2+3 (pass-1 under-rated as MED). Router had no logger field. | implementer + test-writer | RESOLVED via PR #15 (squash commit 10dd880) — RouteFrame now logs E-ADM-016 (svtn_id/src_addr) before returning ErrHMACVerificationFailed on both the no-forwarding-entry and HMAC-verify-fail paths; injectable Logger + WithLogger option added to Router (mirrors tmux.Logger); 4 new routing tests assert log emission (Red-Gate proven). Control flow/sentinel unchanged. Merged 2026-06-27. |
