---
artifact_id: adv-S-2.01-pass-10
review_target: S-2.01-hmac-codec
producer: adversary
pass: 10
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 9a1ef34
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 10 — S-2.01 (HMAC codec)

## Verdict: CONVERGED — Zero Findings

First clean pass after pass-9 strict-reset. Streak 1/3.

## Audit summary

### AC↔BC↔test alignment
- AC-001 / BC PC1 (tag size): TagSize=8, ComputeHMAC returns [8]byte taking first 8 bytes — correct.
- AC-002 / BC PC1 (verify success): VerifyHMAC uses crypto/hmac.Equal constant-time — correct.
- AC-003 / BC PC2 / VP-005 (bit-flip rejection): both fuzz functions present; bit-flip uses array-copy semantics — correct.
- AC-004 / BC PC2 / VP-006 (wrong key): TestVerifyHMAC_WrongKey + merged property test — correct.
- AC-005 / BC PC2 (HKDF determinism + RFC 5869 KAT): TestDeriveKey_Deterministic + hkdf_internal_test.go §A.1 KAT — correct.

### Cryptographic correctness
- HMAC truncation correct.
- VerifyHMAC constant-time via hmac.Equal.
- hkdfSHA256 RFC 5869-compliant Extract+Expand with length bounds [0, 255*32].
- DeriveKey wires svtnID[:] as salt, pubkey as IKM, info="switchboard-frame-auth", L=32. Matches ARCH-04:168-171.

### Implementation discipline
- ARCH-08 boundary: only stdlib (crypto/hmac, crypto/sha256). Leaf invariant.
- ARCH-09 pure-core: no I/O, no logging, no time.
- Story rev 5 File Structure table includes all 4 files (post pass-9 fix).

### Test coverage
- 5 ACs + 3 ECs all covered.
- VP-004/005/006 covered.
- Forge-resistance proven on both axes (distinct pubkeys, distinct SVTNs).
- RFC 5869 §A.1 KAT (L=42 exercises multi-block expand).
- RFC 4231 §4.2 KAT (HMAC truncation).
- No-aliasing fuzz via [TagSize]byte array-copy.

### Go quality (.claude/rules/go.md)
- Lowercase single-word package name.
- snake_case file naming.
- No init().
- No nil-check before len.
- No WriteString+Sprintf.
- Constants well-documented.

## Convergence streak: 1/3
Need passes 11 and 12 also clean for BC-5.39.001 closure.
