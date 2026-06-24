---
artifact_id: adv-S-2.01-pass-02
review_target: S-2.01-hmac-codec
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 5d6f928
findings_count: 2
findings_by_severity: {critical: 0, high: 1, medium: 0, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 2 — S-2.01 (HMAC codec)

## High

### F-001 — Spec-mandated test name TestDeriveKey_RFC5869_KAT is absent; substituted with self-circular deterministic anchor
- Location: `.worktrees/S-2.01/internal/hmac/hmac_test.go:239` (`TestDeriveKey_RFC5869_DeterministicAnchor`); `.factory/stories/S-2.01-hmac-codec.md:66`; `.factory/specs/architecture/ARCH-04-admission-security.md:182-184`
- Evidence: Story rev 2 + ARCH-04 v1.1 both mandate `TestDeriveKey_RFC5869_KAT` covering the RFC 5869 §A.1 vector. Implementation provides only `TestDeriveKey_RFC5869_DeterministicAnchor`, expected hex literal at hmac_test.go:246-251 derived from the same stdlib primitives the implementation uses (self-circular).
- Impact: A subtle HKDF bug (e.g., wrong T(1) counter byte, missing salt expand on PRK) would not be detected because the anchor was minted by the same code path. Spec-required RFC 5869 KAT pinning the algorithm against externally-validated ground truth is missing.
- Implementer self-flagged at hmac_test.go:238: "(F-004 follow-through; adversary pass-2 to adjudicate.)"
- Real obstruction: API shape (`DeriveKey(pubkey, svtnID [16]byte) → [32]byte`) cannot exercise RFC 5869 §A.1 vectors directly because (a) info is hard-coded to "switchboard-frame-auth"; (b) salt is type-locked to [16]byte (§A.1 uses 13-byte salt); (c) L is fixed at 32 (§A.1 uses 42).
- Route: PO + implementer + test-writer (user adjudication required per #260 family — three viable paths)
- Fix (three options):
  1. Expose unexported `hkdfSHA256(ikm, salt, info []byte, length int) []byte` helper in hmac.go and KAT it directly against §A.1 (recommended — minimal API surface change, preserves "auditable in context").
  2. Amend ARCH-04 v1.2 / story rev 3 to accept the deterministic-anchor approach with explicit citation to the third-party tool used to compute the anchor bytes (e.g., Python `hkdf` library output reproduced in test comment).
  3. Add `golang.org/x/crypto/hkdf` as a test-only dependency for an in-test cross-validation KAT — eliminates self-circularity without removing inline production code.

## Low

### F-002 — Stale package-level comment claims golang.org/x/crypto may be imported
- Location: `.worktrees/S-2.01/internal/hmac/hmac.go:7-9`
- Evidence: "Only stdlib and golang.org/x/crypto are permitted" — but post-rev-2, the inline-HKDF decision eliminated the x/crypto option for production code. Implementation imports only `crypto/hmac` and `crypto/sha256` from stdlib.
- Impact: Future maintainer may add an x/crypto dependency under the impression it's sanctioned.
- Route: implementer
- Fix: Tighten the comment to "Only stdlib is permitted; no internal/ imports and no external dependencies." (Note: this constraint may need to relax IF F-001 is resolved via Option 3 — test-only x/crypto.)

## Observations

- HMAC-SHA256 truncation correct.
- Constant-time compare verified (`crypto/hmac.Equal`).
- RFC-4231 §4.2 HMAC vector pinned — true ground truth for HMAC primitive.
- Per-(node, SVTN) forge-resistance covered by two distinctness tests.
- Fuzz harness aliasing correctly handled (array-by-value copy, not slice).
- Pure-core (ARCH-09) discipline holds.
- ARCH-08 leaf position respected.
- All 5 story ACs + EC-001 + EC-002 covered.
- VP-004 (consistency), VP-005 (single-bit tag flip), VP-006 (wrong-key rejection) each covered.
- t.Parallel() consistent.

## Convergence

Streak reset to 0/3. Pass 3 follows F-001 resolution + F-002 fix.
