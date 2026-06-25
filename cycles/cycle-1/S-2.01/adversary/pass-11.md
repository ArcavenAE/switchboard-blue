---
artifact_id: adv-S-2.01-pass-11
review_target: S-2.01-hmac-codec
producer: adversary
pass: 11
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 9a1ef34
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 11 — S-2.01 (HMAC codec)

## Verdict: CONVERGED — Zero Findings

Second consecutive clean pass. Streak 2/3.

## Audit summary

20-axis review:
1. Spec drift: BC PC2/PC3 logging/alerting correctly deferred to router per ARCH-09 purity.
2. HKDF Extract per RFC 5869 — no auto-substitute on empty salt; DeriveKey always passes 16-byte svtnID.
3. HKDF Expand correctness — KAT verified against RFC 5869 §A.1.
4. Constant-time compare via crypto/hmac.Equal.
5. All ACs (5) + ECs (3) covered.
6. File structure 4 files; no internal imports.
7. Concurrency/state: pure functions.
8. Input validation: spec doesn't require for codec layer.
9. Mis-anchoring: all bc_traces/vp_traces resolve correctly.
10. Frontmatter-body coherence verified.
11. Fuzz harnesses well-formed; array-copy semantics correct.
12. Determinism tested.
13. Distinctness (forge-resistance) tested on both axes.
14. KATs: RFC 4231 §4.2 + RFC 5869 §A.1.
15. HKDF length=0 not reachable through public API.
16. Length boundary (>8160) returns nil; unreachable from DeriveKey.
17. Tag uniqueness across keys tested.
18. Verification recomputation per VP-004.
19. Aliasing in fuzz: [TagSize]byte array copies.
20. Lint/style (.claude/rules/go.md): all rules pass.

## Convergence streak: 2/3
Need pass 12 also clean for BC-5.39.001 closure.
