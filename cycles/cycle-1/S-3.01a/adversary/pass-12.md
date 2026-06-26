---
artifact_id: adv-S-3.01a-pass-12
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 12
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ 609c6ae
tip: 530a7b2
findings_count: 2
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 0, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 12 — S-3.01a

## Critical Findings
None.

## High Findings
None.

## Medium Findings

### F-PASS12-M-001 — Inline ctx.Done exit in dispatchLoop leaks c.frames

**File:** `internal/tmux/control.go:357-367` vs `:418-449`

dispatchLoop has four termination paths:
1. `%error` handler (380-396): closes both errCh and frames explicitly.
2. Post-loop graceful (ctx.Done at 419-424): closes errCh + falls through to closeFrames.Do at 446-448.
3. Post-loop unexpected EOF (425-442): sends ErrControlModeDropped + falls through to closeFrames.Do.
4. **Inline ctx.Done (357-367): closes only errCh and returns at line 365, bypassing closeFrames.Do.**

The asymmetry is the defect. errCh closed in all 4 paths; frames closed in only 3 of 4.

If ctx is cancelled while scanner is between iterations (rather than blocked in scanner.Scan), the inline branch fires and c.frames stays open. Consumers using `for frame := range cm.Frames()` block forever unless Close() is later called.

Confidence: HIGH. Severity: MEDIUM.

**Fix:** hoist `closeFrames.Do(...)` into a defer at top of dispatchLoop OR duplicate it into the inline arm before return.

### F-PASS12-M-002 — Documented SINGLE-USE contract not enforced; post-Close Connect panics

**File:** `internal/tmux/control.go:40-42` (contract), `:193-196` (guard), `:282-284` (Close nils), `:502` (unguarded send)

Godoc declares: "ControlMode instances are SINGLE-USE: after Close, the instance is terminal."

Connect guard checks `c.proc != nil || c.cancel != nil`. Close zeros these. Post-Close Connect passes the guard, spawns new dispatchLoop. First %output → `c.frames <- frame` (line 502) → **panic on closed channel** (frames already closed by prior Close).

errCh sends ARE guarded by closeErrCh.Do (silently no-op after first close). frames send is NOT guarded — bare `select { case c.frames <- frame: default: }` panics on closed channel.

Panic executes on the dispatch goroutine. Un-recoverable. Worst-case Go pattern: documented contract violation produces process-wide crash instead of error.

Confidence: HIGH. Severity: MEDIUM.

**Fix:** add a "closed" state flag that Connect checks; reject with new `ErrControlModeClosed`. OR Once-guard the c.frames send path like the errCh sends.

## Observations

- Branch is now clean of cross-namespace BC cites (post-pass-11 M-1 cleanup).
- All pass-11 fixes verified: %error handler, BC-5 cite removal, SINGLE-USE docstring, Frames() positive test, story task 8 reword.
- ARCH-08 §6.6 imports clean.
- ARCH-09 classifications match.
- Race detector clean across 3-count runs.
- Lint 0 issues.

## Novelty Assessment

Novelty: MEDIUM. Two genuinely novel concurrency contract gaps surfaced. M-1 is asymmetric channel-close across exit paths. M-2 is a documented contract that the code doesn't enforce — a sharp footgun. Both mechanical fixes.

## Pattern observation

Twelve passes. Defect-discovery curve has shifted: passes 1-7 surfaced load-bearing test-masks-defect patterns and protocol gaps. Passes 8-12 are surfacing deeper concurrency contract gaps (cleanup asymmetry, contract-vs-code drift). Each pass continues to find real defects. The convergence model is working; defect density is dropping (1H/2M → 1H/2M/1L → 0 → 0 → 1H/1M/1L → 1L → 0 → 0 → 1L → 2M/3L → 2M).

## Resolution decisions

- M-1: defer closeFrames.Do at top of dispatchLoop → all 4 exit paths close frames uniformly.
- M-2: add closed-state flag (e.g., c.closed bool guarded by c.mu, OR re-use closeErrCh's sync.Once); mint new ErrControlModeClosed sentinel; Connect rejects post-Close.

## Streak status

Pass 8-9: CONVERGED. Pass 10-12: NOT_CONVERGED. **Streak: 0/3 toward BC-5.39.001.**
