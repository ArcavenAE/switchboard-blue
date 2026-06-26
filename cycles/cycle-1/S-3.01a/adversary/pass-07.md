---
artifact_id: adv-S-3.01a-pass-07
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 7
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: a8660e9
findings_count: 3
findings_by_severity: {critical: 0, high: 1, medium: 1, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 7 — S-3.01a

## High Findings

### F-PASS7-H-001 — Tick() discards ChannelFrame; payload destroyed (silent data loss)

**File:** `internal/tmux/control.go:432`

```go
for i := 0; i < len(data); i += halfchannel.MaxPayloadSize {
    end := i + halfchannel.MaxPayloadSize
    if end > len(data) { end = len(data) }
    if err := c.downstream.Enqueue(data[i:end]); err != nil { break }
    c.downstream.Tick()   // <-- return ChannelFrame discarded
}
```

Per `internal/halfchannel/halfchannel.go:117`, `Tick()` returns the dequeued ChannelFrame carrying the payload. That return is the ONLY mechanism by which payload leaves the half-channel.

Discarding it = payload destroyed. No other code path consumes the half-channel (verified: grep '.Tick(' shows control.go:432 is the only production caller; cmd/ has no tmux.New site).

BC-2.04.001 PC-5: "Session output events from control mode feed the downstream half-channel."
AC-004: "at least 99% of events emitted... are delivered to the downstream stream."

Current implementation delivers 0% — every byte of %output data is Enqueued, immediately Ticked out, dropped on the floor.

Test `TestTmuxControlMode_OutputEventsFeedDownstream` only asserts ds.Seq() >= 1, measuring Tick invocation NOT data delivery. Contrived test that passes despite data loss.

Confidence: HIGH. Severity: HIGH.

**This is the 4th load-bearing test-masked defect surfaced across 7 passes** (after F-01 enumeration in pass 1, H-03 stdin in pass 2, M-1 fragmentation in pass 5).

## Medium Findings

### F-PASS7-M-001 — Hardcoded time.Sleep(20ms) race in 3 tests

**Files:**
- `internal/tmux/control_test.go:152` (AC-002 EnumeratesSessions)
- `internal/tmux/control_test.go:201` (AC-003 SessionLifecycleEvents)
- `internal/tmux/control_test.go:325` (NoSessionsOnStartup)

Each test injects a fake stream then waits with `time.Sleep(20ms)` before asserting. No synchronisation primitive couples the test's read to dispatchLoop's processing.

Risk: on slow CI runner (race detector, container under load, cold cache), 20ms can elapse before dispatchLoop processes the events. Spurious pass or flaky fail.

Contrast: `TestTmuxControlMode_OutputEventsFeedDownstream` (line 258-265) and `LargeOutputLine_NoFalseDrop` (line 420-425) synchronise via `<-cm.Err()`. That pattern is available but unused in lifecycle tests.

## Low Findings

### F-PASS7-L-001 — Close → Connect reuse silently disables drop signalling (pending intent verification)

**File:** `internal/tmux/control.go:145-152, 168-216, 242-281, 376-385`

c.errCh and c.closeErrCh are constructed in New() and never reset. After first Close(), closeErrCh is consumed. Subsequent Connect() succeeds (idempotency guard checks only c.proc/c.cancel). But on next stream EOF in the new connection, closeErrCh.Do is a no-op — ErrControlModeDropped silently skipped.

Not exercised by S-3.01a ACs (no reconnect ACs; EC-002 deferred to S-3.01b), so OUT OF SCOPE for blocking.

Reporting as LOW pending intent: either (a) document ControlMode as single-use, or (b) reset errCh+closeErrCh inside Connect() when starting fresh.

## Resolution decisions (from human review)

- F-PASS7-H-001: pipe Tick() result into c.frames channel that caller drains. Closes the data-loss gap; preserves story scope.
- F-PASS7-M-001: replace time.Sleep with Err() channel sync pattern (matches existing test patterns).
- F-PASS7-L-001: DEFERRED to S-3.01b per user — out of S-3.01a scope (no reconnect ACs). Track in drift register.

## Pattern observation

Seven passes, 4 load-bearing test-masked defects: F-01 (enumeration synthesized by fake), H-03 (stdin write absent, fake synthesized %begin/%end response), M-1 (oversize Enqueue silently dropped), H-1 pass-7 (Tick result discarded). Per user decision: continue ratcheting; trust BC-5.39.001 to converge. The system is working — each pass surfaces deeper layers and each fix lands cleanly. This is NOT a process failure; it's the convergence model functioning as designed.
