---
artifact_id: adv-S-2.01-pass-08
review_target: S-2.01-hmac-codec
producer: adversary
pass: 8
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 9a1ef34
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 8 — S-2.01 (HMAC codec)

## Verdict: CONVERGED — Zero Findings

First clean pass after pass-7 LOW resolved. Streak 1/3.

## Verification trail

- HMAC algorithm: ComputeHMAC truncates HMAC-SHA256 to first 8 bytes (hmac.go:37-44); RFC 4231 §4.2 KAT pinned. Matches ARCH-02 §HMAC tag and BC-2.05.005 precondition 2.
- HKDF impl: Inline RFC 5869 Extract+Expand (hmac.go:84-108); RFC 5869 §A.1 KAT pinned (hkdf_internal_test.go:23-45). Salt = svtnID[:], info = "switchboard-frame-auth", L=32.
- HKDF loop safety: byte counter `i` bounded by length-validation gate (length > 255*sha256.Size returns nil).
- Constant-time compare: crypto/hmac.Equal in VerifyHMAC.
- Tag aliasing eliminated: [TagSize]byte signature ensures `flipped := tag` copies; documented in fuzz comment.
- AC→BC traces: AC-001..AC-005 (story rev 4 lines 47-66) trace to valid BC sections; pass-4 patch confirmed correct.
- Purity: no I/O, no logging, no key exposure. ARCH-09 pure-core.
- Imports: stdlib only (crypto/hmac, crypto/sha256). ARCH-08 leaf invariant.
- VP coverage: VP-004 (TestPropComputeVerifyConsistency 10 cases), VP-005 (dual fuzz), VP-006 (merged into VP-004 cases per VP-006 v1.1).
- Forge-resistance: distinct-pubkey + distinct-SVTN tests assert per-(node, SVTN) invariant.

## Convergence streak: 1/3
Need passes 9 and 10 also clean for BC-5.39.001 closure.
