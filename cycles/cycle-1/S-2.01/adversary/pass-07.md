---
artifact_id: adv-S-2.01-pass-07
review_target: S-2.01-hmac-codec
producer: adversary
pass: 7
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 93959cb
findings_count: 1
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 1, nitpick: 0}
verdict_adversary_self: CONVERGED (zero critical+high+medium)
verdict_orchestrator: NOT_CONVERGED (strict BC-5.39.001 requires zero findings; user 2026-06-24 elected strict consistency)
timestamp: 2026-06-24
---

# Adversary Pass 7 — S-2.01 (HMAC codec)

## Low

### F-001 — VerifyHMAC doc-comment mis-anchors constant-time rationale to BC-2.05.005 PC3
- Location: `.worktrees/S-2.01/internal/hmac/hmac.go:49-50, 61-62`
- Evidence: Doc comments cite "BC-2.05.005 postcondition 3" as the rationale for using `crypto/hmac.Equal` (constant-time comparison). BC-2.05.005 PC3 is actually "Repeated HMAC failures from the same source address trigger an admission alert (implementation: ≥5 failures in 60 seconds)." That postcondition has nothing to do with timing-oracle prevention.
- Impact: Behavior is correct (`ghmac.Equal` is the right call); only the comment's BC pointer is wrong. Cosmetic doc-comment mis-anchor. No runtime impact.
- Route: implementer
- Fix: Replace "BC-2.05.005 postcondition 3" with either no BC clause (defense-in-depth) OR cite ARCH-04 §HMAC verification generally. The constant-time-compare practice is implementer best-practice, not a spec-anchored requirement.

## Observations (carry over from prior passes — all clean)

- AC→test coverage complete (5 ACs + 3 ECs + 3 VPs).
- HKDF inline correct per RFC 5869; KAT pinned via RFC §A.1.
- HMAC truncation pinned via RFC 4231 §4.2.
- Constant-time compare verified (correct usage, wrong BC citation).
- Pure-core discipline.
- ARCH-08 leaf invariant.
- No key material leakage.
- Forge-resistance covered structurally.
- VP-004/005/006 alignment post fix-burst.

## Convergence

Streak reset to 0/3. User elected strict consistency: fix the LOW + run passes 8/9/10. Pass 8 follows implementer's 1-line doc fix.
