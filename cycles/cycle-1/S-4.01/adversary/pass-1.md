---
story: S-4.01
pass: 1
reviewed_commit: 571a31b
verdict: NOT_CONVERGED
severity_summary: 2C/4H
date: 2026-06-27
---

# S-4.01 Adversarial Review — Pass 1

**Verdict:** NOT_CONVERGED 2C/4H

---

## Critical

### F-001 — nil-safety / CWE-476 — paths.go:190-191

`Rank()` calls `rp.Tracker.IsActive()` / `Score()` with no nil guard. `RankedPath` is a public struct whose zero value has `Tracker == nil`. `Multipath.Send` feeds caller-supplied `pathSet` into `Rank` after only a copy, so a nil tracker → production panic in routing hot path. No test covers nil Tracker.

### F-002 — spec conformance — multipath.go:228-237

`Receive` (the ENDPOINT deduplicator) keys the drop cache on compound `(checksum, arrivalInterfaceID)`. VP-024 "Note on dedup scope" and BC-2.02.002 postconditions 1-2 / invariant DI-009 require endpoint dedup by **checksum alone**. With the compound key, duplicate-and-race copies arriving on different interfaces both miss the cache and BOTH get delivered — defeating DI-009 first-arrival-wins. `TestBC_2_02_002_Receive_DifferentInterfaceSameChecksumNotSuppressed` (multipath_test.go:454-483) pins this WRONG behavior.

---

## High

### F-003 — silent failure — multipath.go:200-206

`Send` swallows per-path `fn` errors; if both paths fail, returns `([], nil)` — total send failure invisible; no `SendResult{Sent:false}` recorded.

### F-004 — test gap / concurrency — both `_test.go` files

Both test files have zero `go func` / `WaitGroup`; the 22 `t.Parallel()` calls only parallelize independent subtests on separate objects. The shared `DropCache` / `Multipath` / `PathTracker` are never driven under contention, so `go test -race` passing is NOT evidence locking is correct.

### F-005 — concurrency / TOCTOU — multipath.go:231-235

`Receive` does `Contains()` (lock/unlock) then `Add()` (lock/unlock) non-atomically; two concurrent copies of the same frame both observe a miss and both deliver, breaking DI-009. Needs a single locked `AddIfAbsent`-style operation.

### F-006 — spec conformance / edge case — paths.go:108-129

Once `consecutiveMisses >= 3` sets `active = false`, a later successful probe resets the counter but never restores `active = true`; a path that recovers is permanently stranded. Contradicts BC-2.02.003 EC-001 ("recovers on good probes"). No reactivation test.

---

## Observations

### F-007 — MEDIUM — test gap — paths_test.go:260

`EWMAConvergence` ±5ms window is loose and would also pass a last-value (`alpha=1`) implementation; identical-probe inputs don't distinguish EWMA from last-value.

### F-008 — MEDIUM — test gap

First-probe RTT-override path (paths.go:121-123) only tested at `alpha=1.0` where override and EWMA are indistinguishable; the override is never directly asserted.

### F-009 — LOW — multipath_test.go:47

Dead-code no-op `copy()` into nil slice, immediately overwritten line 48; misleading comment.

### F-010 — LOW — spec contradiction (pending intent verification)

BC-2.02.001 postcondition 3 / story EC-001 say single path → one send; ARCH-03:45-46 says degenerate case sends BOTH copies to the same path. Code follows BC/story. Needs spec reconciliation.

---

## Process Gaps

None. `[process-gap]` not tagged — all findings are within-perimeter content defects.
