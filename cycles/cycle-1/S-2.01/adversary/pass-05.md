---
artifact_id: adv-S-2.01-pass-05
review_target: S-2.01-hmac-codec
producer: adversary
pass: 5
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 93959cb
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 5 — S-2.01 (HMAC codec)

## Verdict: CONVERGED — Zero Findings

First clean pass in the convergence streak. Streak 1/3.

## Audit summary by axis

### A. Cryptographic correctness — clean
- HMAC: `crypto/hmac.New(sha256.New, key)` + 8-byte left-truncation matches ADR-001/ARCH-02 (hmac.go:37-44).
- HKDF-SHA256 inline: Extract `PRK = HMAC(salt, IKM)`, Expand `T(i) = HMAC(PRK, T(i-1) || info || i)`, truncate (hmac.go:82-106). Counter overflow impossible inside bounded length range due to early-return guard.
- DeriveKey wires salt=svtn_id, IKM=node_admission_pubkey, info="switchboard-frame-auth", L=32 — matches ARCH-04:168-171.
- Constant-time compare via `crypto/hmac.Equal` (hmac.go:63) satisfies BC-2.05.005 PC3 timing requirement.

### B. BC compliance + AC↔BC trace accuracy — clean post rev-4 patch
- AC-001→PC1, AC-002→PC1, AC-003/AC-004→PC2, AC-005→precondition 2 (HKDF definition). All AC trace pointers in story rev 4 reference real BC sections.
- EC-003 reframed correctly as zero-tag rejection (type-enforced short-tag impossibility).

### C. Test quality — adequate
- RFC 4231 §4.2 KAT (HMAC) + RFC 5869 §A.1 KAT (HKDF) — both pin externally-validated ground truth.
- Wrong-key + wrong-SVTN distinctness tests prove forge-resistance.
- Fuzz harness exercises all 64 tag-bit positions per seed with correct array-copy semantics.
- VP-004/005/006 covered.

### D. Implementation quality / helper API discipline — clean
- hkdfSHA256 unexported; KATted via internal test file (canonical pattern).
- HKDF max-length guard prevents pathological internal-caller misuse.

### E. ARCH compliance — clean
- ARCH-08: no internal imports from hmac.go (leaf invariant).
- ARCH-09: pure-core (no I/O, no clock, no rand, no log).
- ARCH-04 v1.1 inline-HKDF KAT mandate satisfied.

### F. go.md rules — clean
- No init(), no panics, no interface{}, no log/fmt.Print, no nil-vs-len anti-patterns.

### G. Process gaps — none identified.

## Novelty Assessment

Zero — the artifact set has converged across cryptographic correctness, spec alignment, test coverage, and architectural discipline.

## Convergence streak: 1/3
Need passes 6 and 7 also clean for BC-5.39.001 closure.
