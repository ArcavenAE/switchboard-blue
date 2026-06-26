---
artifact_id: adv-S-3.01b-pass-07
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 7
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: c278116
findings_count: 2
findings_by_severity: {critical: 0, high: 1, medium: 1, low: 0, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 7 — S-3.01b

## High Findings

### F-PASS7-H-001 — EC-001 ("old tmux, -CC unsupported") log requirement not satisfied in production

**Files:** `internal/tmux/control.go:150-221,312-335,517-547`; `internal/tmux/pty_fallback.go:451,479-495,654`

With a real old-tmux binary: cmd.Start succeeds → defaultExecFn returns nil error → ControlMode.Connect returns nil. Flag rejection detected by stderr classification only AFTER subprocess exits. By that time, dispatchLoop has already observed stdout EOF and written ErrControlModeDropped to errCh via c.closeErrCh.Do (control.go:539). The classify goroutine waits on classifyCh, which is written only AFTER cmd.Wait + drainWG.Wait — strictly AFTER the stdout EOF that fires dispatchLoop.

sync.Once always elects ErrControlModeDropped; ErrControlModeUnsupportedFlag dropped on floor.

Downstream: watchAndFallback receives ErrControlModeDropped → attempts factory reconnect → logs EC-003 message ("tmux control mode lost; falling back to PTY proxy"). The EC-001-mandated "tmux version does not support -CC flag" message is never emitted in production.

Unit test TestPTYProxy_EC001_OldTmuxVersion bypasses this via fakeExecFuncErr — synchronous error defaultExecFn cannot produce. Test passes; production behavior diverges. **AC-001 / BC-2.04.002 EC-001 log unmet.**

Confidence: HIGH. Severity: HIGH.

## Medium Findings

### F-PASS7-M-001 — classify-vs-dispatchLoop race makes ErrControlModeUnsupportedFlag effectively unreachable through Err()

**File:** `internal/tmux/control.go:71-77,202-218,321-335,539-546`

Design comment (control.go:73-76) states classification "supersedes" ErrControlModeDropped. But:
- classifyCh written only after cmd.Wait + drainWG.Wait
- stdout EOF strictly precedes cmd.Wait return
- Both goroutines write through same c.closeErrCh.Do (sync.Once)
- dispatchLoop wins by causal ordering, not by chance

Contract violation: Err() channel guarantee is false. Callers ranging Err() never see ErrControlModeUnsupportedFlag on old-tmux-fail runs.

This is the underlying mechanism for H-001 but logged separately as standalone API-contract violation.

## Observations

- ADR-010 v1.2 (ARCH-01:153-154) still asserts "Control mode is NOT retried mid-session" but BC-2.04.002 EC-003 + story task 6 require 3-attempt reconnect. Implementation follows BC + story per pass-1 decision. ADR-010 wording lags. **Out of perimeter** (cross-story spec content).
- ARCH-08 §6.5 import constraints honored.
- ioRelay fragmentation loop dead code under 4096-byte buffer (single chunk per read); symmetry with dispatchLoop reasonable.
- defaultExecFn drainWG.Wait before stderrBuf.String() correctly fixes the StderrPipe-vs-Wait race.

## Resolution decision (from human review)

H-001+M-001 fix: dispatchLoop defers drop-signal until classifyCh resolves with timeout. On stdout EOF, dispatchLoop waits on classifyCh (bounded timeout, e.g., 100ms). If classifyCh delivers non-nil sentinel, write that; if timeout or nil, write ErrControlModeDropped. Honors documented "classification supersedes" contract.

## Pattern observation

This is the 5th "test passes for wrong reason" pattern across S-3.01a + S-3.01b. Fake injection bypasses the production code path the test is supposed to verify. Cumulative process gap; existing vsdd-factory issue #287/#288 series captures the systemic class.

## Novelty Assessment

Novelty: HIGH. EC-001 production-path gap not detectable via existing unit-test surface; required reasoning about causal ordering of stdout-EOF vs cmd.Wait return. Fresh context surfaced it.
