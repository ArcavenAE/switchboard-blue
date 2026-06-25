---
artifact_id: adv-S-2.01-pass-03
review_target: S-2.01-hmac-codec
producer: adversary
pass: 3
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 1101369
findings_count: 4
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 3, nitpick: 0}
verdict_adversary_self: CONVERGED (medium classified as spec-doc patch only)
verdict_orchestrator: NOT_CONVERGED (BC-5.39.001 requires zero findings; user 2026-06-24 elected to fix all 4 for strict convergence)
timestamp: 2026-06-24
---

# Adversary Pass 3 — S-2.01 (HMAC codec)

## Medium

### F-001 — EC-003 spec text is stale relative to type-enforced behavior
- Location: Story `.factory/stories/S-2.01-hmac-codec.md:82`; test `.worktrees/S-2.01/internal/hmac/hmac_test.go:110`
- Evidence: EC-003 defines "Tag slice is fewer than 8 bytes — VerifyHMAC returns false without panic." Impossible by `[TagSize]byte` signature. Test renamed to TestVerifyHMAC_ZeroTagRejected.
- Impact: Spec rev-2 still ships wording for a runtime case that cannot exist. Downstream reader will be confused.
- Route: product-owner
- Fix: One-line story EC-003 patch — note it's compile-time-enforced via `[TagSize]byte` signature; the test verifies the equivalent zero-tag rejection semantic.

## Low

### F-002 — VP-004 declared `proof_method: proptest` but implementation uses table-driven cases
- Location: `.factory/specs/verification-properties/VP-004.md:46`; `.worktrees/S-2.01/internal/hmac/hmac_test.go:151-190`
- Evidence: VP spec says proptest+gopter; impl uses 10-case table. Reasoned in test doc comment (HMAC-SHA256 deterministic; correctness by construction). VP-INDEX→impl mapping tooling will see VP-004 as uncovered.
- Route: architect
- Fix: Update VP-004 to permit table-driven as acceptable proof method for deterministic primitives; add a Revisions note.

### F-003 — VP-006 has no standalone test name matching the VP spec skeleton
- Location: `.factory/specs/verification-properties/VP-006.md:64`
- Evidence: VP-006 declares `TestPropVerifyHMAC_RejectsWrongKey`; implementation merged the property into `TestPropComputeVerifyConsistency`.
- Route: architect
- Fix: Update VP-006 harness skeleton to acknowledge the merger OR rename suggested test in skeleton; document the merger in Revisions.

### F-004 — hkdfSHA256 does not enforce RFC 5869 max output length (255*HashLen)
- Location: `.worktrees/S-2.01/internal/hmac/hmac.go:81-101`
- Evidence: Expand loop counter `i := byte(1)` wraps at 256; if `length > 8160`, `i` overflows silently. Doc comment claims support up to 8160 but no length check.
- Route: implementer
- Fix: Add `if length > 255 * sha256.Size { return nil }` (or panic for dev-time enforcement) at the top of hkdfSHA256. Unreachable from DeriveKey (hardcoded KeySize=32), but defensive for future internal callers.

## Observations

- KAT verified: RFC 4231 §4.2 case 2 (HMAC) + RFC 5869 §A.1 case 1 (HKDF), both byte-correct.
- Constant-time compare verified (`crypto/hmac.Equal`).
- ARCH-08/09 compliant; no secret leakage.
- 5 ACs + 3 VPs + 3 ECs all covered.
- AC-003 fuzz only flips byte-0 bit-0; defensible by HMAC diffusion but worth noting.
- Cross-module independence preserved.

## Routing decisions (2026-06-24 user approval after #260-aware re-prompting)

User elected "Fix all 4 — strict BC-5.39.001 convergence" rather than accepting adversary self-CONVERGED verdict. Streak attempt continues after fix burst at 0/3.

Convergence streak: 0/3. Pass 4 follows fix burst.
