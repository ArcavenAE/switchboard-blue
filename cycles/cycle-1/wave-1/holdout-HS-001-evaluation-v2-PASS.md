---
artifact_id: holdout-HS-001-wave-1-v2
producer: holdout-evaluator
wave: 1
scenario_id: HS-001
scenario_version: "1.1"
develop_tip: 9e9a98a
must_pass: true
timestamp: 2026-06-24
information_asymmetry_honored: true
verdict: PASS
mean_satisfaction: 1.00
critical_min_satisfaction: 1.00
---

# Holdout Evaluation Report — Wave 1 (HS-001 v1.1, re-evaluation)

## Summary

- Scenarios evaluated: 1 (HS-001, 6 steps)
- Per-step pass count: 6 / 6
- Mean satisfaction: 1.00
- Critical scenario minimum: 1.00
- Gate: PASS

## Per-Step Results

| Step | Aspect | Score | Notes |
|------|--------|-------|-------|
| 1-3 | 1000 random OuterHeader round-trips, all 6 fields preserved | 1.0 | 1000 random `EncodeOuterHeader` → `ParseOuterHeader` round-trips matched. Total duration: 1.31ms (~76x headroom vs 100ms budget). |
| 4a | 43-byte buffer → ErrFrameTooShort | 1.0 | `errors.Is(err, frame.ErrFrameTooShort)` true. Sentinel traces to E-PRT-002 / BC-2.01.004 PC1. |
| 4b | 45-byte buffer → silent parse | 1.0 | Parsed without error, consistent with "silence for oversized-but-parseable" branch. |
| 4c | version=255 → ErrVersionMismatch | 1.0 | `errors.Is(err, frame.ErrVersionMismatch)` true. Sentinel traces to E-PRT-001 / BC-2.01.004 PC2. |
| 5 | 100 ticks: ChanSeq 1..100, EmptyTick, ChanID=42 | 1.0 | New(42, Upstream, 10ms). First Tick ChanSeq=1 (post-increment per BC-2.01.001 PC5). 100 contiguous frames, all EmptyTick, all ChanID=42. Final Seq=100. |
| 6 | Independent up/down sequences | 1.0 | up=New(42,Upstream,10ms), down=New(43,Downstream,10ms). After 50 Upstream ticks: up.Seq=50, down.Seq=0; upstream frames ChanSeq 1..50 in order. After 50 Downstream ticks: down.Seq=50; downstream frames ChanSeq 1..50; up.Seq unchanged at 50. Independence invariant holds. |

## Rubric scoring

| Dimension | Weight | Score | Weighted |
|-----------|--------|-------|----------|
| Functional correctness | 0.5 | 1.0 | 0.50 |
| Edge case handling | 0.2 | 1.0 | 0.20 |
| Error quality | 0.2 | 1.0 | 0.20 |
| Performance | 0.1 | 1.0 | 0.10 |
| Total | 1.0 | — | 1.00 |

## Minor observation (non-blocking)

The scenario document references error codes `E-FRM-001` and `E-FRM-002`, but error-taxonomy.md enumerates these as `E-PRT-002` (header truncated) and `E-PRT-001` (unsupported version). Frame package godoc explicitly traces `ErrFrameTooShort` → `E-PRT-002` and `ErrVersionMismatch` → `E-PRT-001`. Sentinel-identity match (errors.Is) is the load-bearing assertion and held in both cases. Recommend spec/taxonomy authors reconcile the `E-FRM-*` ↔ `E-PRT-*` namespaces in a future cycle.

## Procedural Integrity

- Files read: `.factory/holdout-scenarios/wave-scenarios/wave-1.md`, `.factory/specs/prd-supplements/error-taxonomy.md`, `go.mod`
- Godoc commands: `go doc internal/frame`, `go doc internal/halfchannel`, `go doc internal/frame.OuterHeader`, `go doc -all` for both packages
- Forbidden artifacts NOT read: any .go source in internal/*, BCs, VPs, ARCH-*, stories, prior reviews/holdouts, STATE.md, PRD
- Harness location: `.factory/holdout-tmp/hs001-v1.1/harness_test.go` (sibling to internal/, legally imports internal packages); deleted post-run; worktree byte-identical to pre-eval state
- Information asymmetry honored: TRUE

## Verdict

PASS. Wave-1 integration gate clears the holdout component on this evaluation.
