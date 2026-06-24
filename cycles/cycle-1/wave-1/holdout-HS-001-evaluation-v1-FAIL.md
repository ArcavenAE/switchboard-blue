---
artifact_id: holdout-HS-001-wave-1-v1
producer: holdout-evaluator
wave: 1
scenario_id: HS-001
scenario_version: "1.0"
develop_tip: 9e9a98a
must_pass: true
timestamp: 2026-06-24
information_asymmetry_honored: true
verdict: FAIL
root_cause: HS-001 v1.0 wording defect — "starting at 0" read as first-emit-is-0; visible spec converged on post-increment (first-emit-is-1) per BC-2.01.001 PC5 canonical vector. To be re-evaluated against HS-001 v1.1.
---

# Holdout Evaluation — Wave 1 / HS-001 v1.0 (FAIL)

## Summary
- Sub-steps PASS: 4/6
- Sub-steps FAIL: 2/6 (both rooted in single off-by-one)
- Gate: FAIL (must_pass=true; postcondition "starting at 0" not satisfied)

## Per-Step Results

### Step 1: Round-trip 1,000 random OuterHeaders
PASS — 1000/1000 exact, 14.6 ms elapsed.

### Step 2a: 43-byte → ErrFrameTooShort
PASS — sentinel returned, errors.Is chain intact.

### Step 2b: 45-byte → silent parse
PASS — implementation chose silent-parse branch (scenario allows).

### Step 2c: 44-byte version=255 → ErrVersionMismatch
PASS — sentinel returned with descriptive message.

### Step 3: 100 empty ticks → contiguous from 0
FAIL — first emitted ChanSeq is 1, not 0. Post-increment implementation; sequence 1..100 instead of expected 0..99.

### Step 4: Independent sequence spaces (Up + Down × 50)
FAIL by scenario letter (absolute values 1..50 instead of 0..49), PASS by independence intent (cross-channel isolation works correctly).

## Root cause

Visible spec (BC-2.01.001 PC5 canonical vector "sequence 1..10", VP-017/VP-053 harnesses post pass-01 F-007, implementation, 9 S-1.02 adversary passes) converged on post-increment semantics. HS-001 v1.0 Step 5 wording "sequence numbers starting at 0" is ambiguous between "counter initialized at 0" (consistent with visible spec) and "first emitted ChanSeq == 0" (inconsistent with visible spec). The literal reading is the latter; the implementation honors the former.

This is the holdout system working as designed — surfacing a hidden contract that the visible-spec convergence failed to test against. The orchestrator-with-user resolution (2026-06-24) is to patch HS-001 to v1.1 with explicit post-increment wording, NOT to reverse the visible spec.

## Information Asymmetry Attestation

- Files read: `.factory/holdout-scenarios/wave-scenarios/wave-1.md`, `.factory/specs/prd-supplements/error-taxonomy.md`, `go doc` output for `internal/frame` and `internal/halfchannel` exported surfaces only.
- Files NOT read: all .go source/tests under `internal/*`, all BCs/VPs/ARCH, all stories, all prior reviews, STATE.md, PRD.
- Harness location: built in `/tmp/`, relocated to `.factory/holdout-tmp/` due to Go's internal-package import rule, then deleted; worktree unchanged from pre-evaluation state.
- Information asymmetry honored: TRUE.

## Next step

Re-run HS-001 v1.1 (after PO patch) to confirm PASS before declaring wave-1 gate closed.
