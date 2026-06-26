---
artifact_id: adv-S-3.01b-pass-01
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 156f860
findings_count: 11
findings_by_severity: {critical: 2, high: 5, medium: 3, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 1 — S-3.01b

## Critical Findings

### F-PASS1-C-001 — reconnectFn is dead code; mid-session reconnect test exercises nothing it claims to test

**File:** `internal/tmux/pty_fallback_test.go:461-467`

```go
reconnectFn := func(_ context.Context) error {
    reconnectMu.Lock()
    reconnectCalls++
    reconnectMu.Unlock()
    return fmt.Errorf("%w: cannot reconnect", tmux.ErrControlModeUnavailable)
}
_ = reconnectFn // used by watchAndFallback; injected for assertion below
```

Comment claims reconnectFn is "injected for assertion below", but:
1. No injection seam exists — `pty_fallback.go:298-301` hard-codes `func(c context.Context) error { return sc.ctrl.Connect(c) }`
2. `reconnectCalls` is never asserted on
3. Test passes "for the wrong reason": `sc.ctrl.Connect(ctx)` returns ErrAlreadyConnected (control.go:207-208), counted as failure → 3 attempts → PTY fallback

Same S-3.01a "test-masks-defect" pattern. EC-003's "3 reconnect attempts" is not actually verified.

Confidence: HIGH. Severity: CRITICAL.

### F-PASS1-C-002 — Mid-session "reconnect" path can NEVER succeed; BC-2.04.002 EC-003 not implemented

**File:** `internal/tmux/pty_fallback.go:298-301, 386-415`

watchAndFallback calls `reconnectFn(ctx)` = `sc.ctrl.Connect(ctx)` on the SAME *ControlMode whose dispatchLoop just exited. Per control.go:39-46 ControlMode is SINGLE-USE; control.go:201-208 shows:
- c.closed == true → ErrControlModeClosed
- c.proc != nil || c.cancel != nil → ErrAlreadyConnected

After dispatchLoop EOF, proc/cancel are still set. So 3 "reconnect attempts" are theatrical: every attempt is immediate ErrAlreadyConnected, loop runs in microseconds, always falls through to PTY fallback.

BC-2.04.002 EC-003 intent (3 genuine reconnection attempts with opportunity to succeed) is not implemented.

Confidence: HIGH. Severity: CRITICAL.

## High Findings

### F-PASS1-H-001 — watchAndFallback goroutine + ctrl subprocess leak after PTY fallback

**File:** `internal/tmux/pty_fallback.go:298, 366-376, 407-414`

SessionConnector.Connect launches `go sc.watchAndFallback(...)` without sync.WaitGroup. After mid-session fallback succeeds (active=pty), original ctrl is never Close'd. SessionConnector.Close only closes `active` — and after fallback active==pty. ctrl retains live subprocess (defaultExecFn's cmd.Wait goroutine continues), stdin pipe (never closed), cancel context (never cancelled).

### F-PASS1-H-002 — Use-after-close race resurrecting sc.active

**File:** `internal/tmux/pty_fallback.go:366-376, 407-411`

T1: sc.Close acquires mu, sets active=nil. T2: watchAndFallback completes pty.Connect, acquires mu, sets active=sc.pty + inPTYMode=true. T1: returns; caller believes closed. Connector now has active=pty while owner thinks Close completed.

### F-PASS1-H-003 — Sentinel error classification by string-matching violates CLAUDE.md

**File:** `internal/tmux/pty_fallback.go:331-343`

CLAUDE.md go.md error handling: "Use errors.Is()/errors.As() — NEVER string matching." Code uses strings.Contains for "-CC flag not supported" and "no such file" detection. Brittle + project rule violation.

### F-PASS1-H-004 — Production default noopLogger silently violates BC-2.04.002 invariant 3

**File:** `internal/tmux/pty_fallback.go:124, 429-434`

NewPTYProxy defaults logger=noopLogger; Log discards. BC-2.04.002 invariant 3: "fallback state is clearly communicated — never silent." Production callers without WithLogger silently swallow mandatory PC-3/EC-* log messages.

### F-PASS1-H-005 — Publish error silently discarded for ALL errors

**File:** `internal/tmux/pty_fallback.go:170`

```go
_ = p.publisher.Publish(p.sessionName)
```

Comment justifies ignoring ErrSessionAlreadyPublished, but `_ =` discards ALL errors. If ErrSessionNameInvalid (or any other), synthetic session silently absent. CLAUDE.md rule 3 violation.

## Medium Findings

### F-PASS1-M-001 — Stale ARCH-08 §6.6 anchor (should be §6.5)

**File:** `internal/tmux/pty_fallback.go:16`

Docstring: "ARCH-08 §6.6". Post v1.6 promotion (S-3.01a merge), authoritative section is §6.5.

### F-PASS1-M-002 — E-SYS-001 message diverges from canonical taxonomy

**File:** `internal/tmux/pty_fallback.go:39`

Code: "PTY device unavailable: cannot start access node"
Taxonomy: "...cannot start access node. Install 'openpty' or check device permissions."

Missing actionable suffix. Story spec also uses colon vs semicolon (separate inconsistency).

### F-PASS1-M-003 — TestPTYProxy_NoAutoUpgrade proves nothing about long-term invariance

**File:** `internal/tmux/pty_fallback_test.go:507-547`

100ms sleep observation window. Weak temporal invariant assertion. If a hypothetical bug auto-upgrades after 500ms, test passes false-green.

## Low Findings

### F-PASS1-L-001 — Concurrency docstring inaccurate for Sessions()

**File:** `internal/tmux/pty_fallback.go:90-92, 214-216`

"All exported methods are protected by mu" — but Sessions() doesn't take mu (delegates to publisher's own thread-safety). Docstring is inaccurate.

## Resolution decisions (from human review)

- C-001: implement WithControlModeFactory option; test asserts factory called 3 times.
- C-002: make reconnect = construct NEW ControlMode each attempt via factory; 3 attempts = 3 fresh instances. Honor BC EC-003.
- H-001/H-002: closed flag + wg.Wait + Close both ctrl AND pty regardless of active.
- H-003: mint ErrControlModeUnsupportedFlag + ErrControlModeBinaryNotFound sentinels; defaultExecFn wraps; controlModeFailureLogMsg dispatches via errors.Is.
- H-004, H-005, M-001, M-002, M-003, L-001: fix mechanically per finding text.

## Novelty Assessment

Novelty: HIGH. Two CRITICAL findings are the S-3.01a "test-masks-defect" and "spec-not-actually-implemented" patterns recurring on the very first pass — exactly the pattern S-3.01a's 15 passes drilled into. The implementation correctly synthesizes a fake reconnectFn but never wires it to production AND the single-use ControlMode contract from S-3.01a's pass-12 makes the production reconnect path inert.
