---
artifact_id: adv-S-2.01-pass-06
review_target: S-2.01-hmac-codec
producer: adversary
pass: 6
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 93959cb
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 6 — S-2.01 (HMAC codec)

## Verdict: CONVERGED — Zero Findings

Second consecutive clean pass. Streak 2/3.

## Audit summary

### Implementation correctness — clean
- HMAC-SHA256, truncate to 8 bytes via `crypto/hmac.Sum(nil)[:8]`.
- Constant-time verify via `crypto/hmac.Equal`.
- Inline HKDF per RFC 5869 with bounds check (length < 0 || > 255*32).
- info = "switchboard-frame-auth", KeySize = 32.
- No I/O, no key logging.
- Pure-core, stdlib-only.

### Test coverage — clean
- RFC 4231 §4.2 HMAC KAT (TagSize truncation pinned).
- RFC 5869 §A.1 HKDF KAT (algorithm pinned via internal-package test).
- 10-case consistency table (`TestPropComputeVerifyConsistency`).
- Fuzz dual coverage: frame-bit-flip + tag-bit-flip across all 64 positions.
- Distinct-pubkey + distinct-SVTN forge-resistance tests.
- All-zero-SVTN EC, empty-frame EC, zero-tag rejection EC.

### ARCH/BC alignment — clean
- BC-2.05.005 postconditions covered (post pass-4 trace corrections).
- ARCH-02 8-byte tag ✓.
- ARCH-04 v1.1 HKDF-SHA256 keying ✓.
- ADR-001 (amended) ✓.
- VP-004 v1.1, VP-005 v1.1, VP-006 v1.1 all implemented.

### go.md compliance — clean
- Table-driven tests.
- stdlib `testing` only.
- t.Parallel() consistent.
- No init(), no log.Fatal, no panics in lib code.
- Value receivers where appropriate.

### Process gaps — none identified.

## Convergence streak: 2/3
Need pass 7 also clean for BC-5.39.001 closure.
