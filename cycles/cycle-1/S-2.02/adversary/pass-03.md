---
artifact_id: adv-S-2.02-pass-03
review_target: S-2.02-admission-svtn-isolation
producer: adversary
pass: 3
fresh_context: true
branch: feature/S-2.02-admission-svtn-isolation
base: develop @ 3c4104e
tip: 1319334
findings_count: 2
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 0, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
---

# Adversary Pass 3 — S-2.02 (Admission + SVTN isolation)

## Medium

### M-1 — Admission fuzz seed corpus only feeds 16 bytes to ed25519.GenerateKey (which needs 32) → seed skipped on every run (L-3 regression)
- Location: `.worktrees/S-2.02/internal/admission/admission_test.go:501-547` (seed at line 504; router keygen at line 525)
- Evidence:
  - Line 504 corpus literal "seed-node-keypair-00000000000000seed-router-keypair-000000000000" = 64 bytes.
  - Line 514: `ed25519.GenerateKey(bytes.NewReader(data[:32]))` — 32 bytes, OK.
  - Line 525: `ed25519.GenerateKey(bytes.NewReader(data[48:64]))` — feeds 16 bytes into ed25519.GenerateKey which internally calls io.ReadFull(rand, seed[:32]). Returns io.ErrUnexpectedEOF.
  - Lines 526-529: on error, test calls t.Skip(). **Seed corpus is skipped on every run** defeating L-3 stated goal ("Crashes are reproducible from the corpus entry" — line 503 comment).
- Impact: Test-quality defect; not a security regression in production code. Mutator may eventually generate inputs ≥64 bytes with sufficient entropy, but the corpus-driven reproducibility claim is false.
- Route: test-writer
- Fix: Either (a) grow corpus to 80 bytes (16 SVTN + 32 node + 32 router) and re-slice as [0:32] node, [32:48] SVTN, [48:80] router; OR (b) derive keys via deterministic HKDF/SHA expand from 16-byte seeds.

### M-2 — Routing fuzz seed corpus 70 bytes but length gate requires ≥80 → seed skipped on first check (L-3 regression)
- Location: `.worktrees/S-2.02/internal/routing/routing_test.go:273-325` (seed at line 276; length gate at line 280)
- Evidence:
  - Line 276 corpus literal "seed-unadmitted-keypair-00000000seed-admitted-keypair-000000000000svtn" = 70 bytes (verified by character count).
  - Line 280: `if len(data) < 80 { t.Skip() }` — seed fails immediately.
  - Comment at line 275 claims "Crashes are reproducible from the corpus entry" — false; corpus never reaches keygen step.
- Impact: Same false-green pattern as M-1. Fuzz target's purpose (assert RouteFrame returns non-nil error for unadmitted source per VP-008 + BC-2.05.002 invariant 1) is functionally untested by the seed corpus.
- Route: test-writer
- Fix: Grow corpus literal to 80 bytes (32 + 32 + 16) by appending characters or restructuring the literal.

## Observations (carry forward — all clean)

- AC-007 wording vs test scope: AC-007 says "node's private key" but test correctly checks router private key bytes are absent. Function signature only accepts router key, so AC text could clarify. Not a defect (story rev 1.2 wording stable enough).
- Lookup returns pointer-to-fresh-copy with deep-cloned PublicKey (rule 12 satisfied).
- isSelfAddressed fully removed from routing.go (L-1 fix complete).
- verifyFrameHMAC wired for next wave (//nolint:unused with clear justification).
- recordNonceUnlocked lock-invariant maintained.
- H-1 race fix audit holds.

## Convergence

Streak reset to 0/3. Pass 4 follows test-writer's seed-corpus byte-count fix.

This is the second test-writer regression on the same test-quality axis — pass-2 introduced the input-driven fuzz pattern but didn't verify the seed literal byte counts against the slice indices. Worth noting in pass-3 conclusions for process improvement (`[process-gap]` candidate but not tagged since the fix is straightforward).
