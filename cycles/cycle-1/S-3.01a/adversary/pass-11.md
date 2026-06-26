---
artifact_id: adv-S-3.01a-pass-11
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 11
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: cc5bccf
findings_count: 5
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 3, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 11 — S-3.01a

## Critical Findings
None.

## High Findings
None.

## Medium Findings

### M-1 — BC-5.38.001 non-existent BC reference embedded in product code

**Files:** `internal/tmux/control.go:43, 98-99`

Two godoc blocks cite "(BC-5.38.001; test hermetic constraint)". Glob of `.factory/specs/behavioral-contracts/**/BC-5.38*` returns no files. Switchboard's BC namespace is BC-2.NN.NNN; BC-5.NN.NNN is vsdd-factory's TDD discipline namespace (Red Gate stubs), not switchboard's.

Production code references a BC not resolvable in this repo's BC tree. Breaks semantic-anchoring contract.

Pattern: stub-architect plants this BC-5.38.001 citation in every story's stubs (S-2.02, S-1.03, S-3.04 all touched; one cite leaked to develop in `internal/admission/reauth_test.go:37`).

Confidence: HIGH. Severity: MEDIUM.

### M-2 — %error control-mode response not handled; inSessionList stuck on tmux command failure

**Files:** `internal/tmux/control.go:368-385`

The dispatch loop tracks `inSessionList` and only resets it on `%end`. The tmux control-mode protocol uses `%begin/%end` for command success and `%begin/%error` for command failure. If `list-sessions` fails (server starting, no permissions, internal tmux error), tmux emits `%error` to close the block. Current code: `inSessionList` remains true indefinitely, causing every subsequent line (including future %session-created, %output) to be passed verbatim to publisher.Publish as session names.

Silently corrupts the session list with garbage entries shaped like `"%session-created foo"`.

BC-2.04.001 EC-001 mandates fallback on control-mode init failure. %error during list-sessions IS such a failure.

Confidence: HIGH. Severity: MEDIUM.

## Low Findings

### L-1 — Connect docstring implies reconnection supported; closed channels make it unsafe

**File:** `internal/tmux/control.go:38-39, 182-183`

`ErrAlreadyConnected` documented as "Callers must Close before reconnecting." But Close closes c.errCh AND c.frames via sync.Once permanently. Subsequent Connect would pass idempotency guard (c.proc==nil) and spawn new dispatchLoop. First %output → send on closed channel → panic.

### L-2 — VP-031 deferral inconsistency between story task and test docstring

**File:** `.factory/stories/S-3.01a-tmux-control-mode.md:107` vs `internal/tmux/control_test.go:4-7, 81-83`

Story task line 107: "Integration test with real tmux session (VP-031)". Test file: hermetic-only constraint; AC-001 docstring: "VP-031 (e2e against real tmux) is deferred to the integration test harness."

### L-3 — Frames() channel has no positive test coverage

**File:** `internal/tmux/control_test.go`

ControlMode.Frames() is the primary fan-out channel (F-PASS7-H-001). No test reads from it. AC-004 verifies ds.Seq() >= 1 via underlying halfchannel, not via exposed Frames(). Bug in c.frames send path would be undetected.

## Resolution decisions (from human review)

- M-1: implementer removes BC-5.38.001 cites from S-3.01a control.go (3 files: forward-fix). Separate PR cleans BC-5.38.001 from already-merged internal/admission/reauth_test.go (1 site on develop). vsdd-factory issue (P27) files the systemic stub-architect pattern.
- M-2: add %error handler that clears inSessionList AND signals ErrControlModeDropped (PTY fallback per ADR-010 + BC-2.04.001 EC-001).
- L-1: fix all 3 LOWs in this burst. Clarify Connect docstring as single-use; reword story task 8; add minimal Frames() test.

## Novelty Assessment

Novelty: MEDIUM. M-1 and M-2 independent of prior passes — M-1 is systemic (stub-architect cross-namespace cite pattern not caught in S-2.02/S-1.03/S-3.04 audits); M-2 is a real protocol gap. L-1 is the resurfacing of pass-7 L-1 (deferred Close→Connect reuse). L-2 and L-3 are scope-deferral inconsistencies.

## Streak status

Pass 1-3: NOT_CONVERGED. Pass 4: CONVERGED (in retrospect missed defects). Pass 5-7: NOT_CONVERGED. Pass 8-9: CONVERGED. Pass 10-11: NOT_CONVERGED.

**Streak: 0/3 toward BC-5.39.001.**
