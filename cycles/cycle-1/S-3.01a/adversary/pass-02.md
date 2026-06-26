---
artifact_id: adv-S-3.01a-pass-02
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: c07a3ec
findings_count: 5
findings_by_severity: {critical: 0, high: 4, medium: 1, low: 0, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 2 — S-3.01a

## Critical Findings
None.

## High Findings

### H-01 — Zombie subprocess leak: cmd.Start() without cmd.Wait()

**File:** `internal/tmux/control.go:80-97` (defaultExecFn)

`cmd.Start()` called; `cmd.Wait()` never called anywhere. The cmd variable goes out of scope after defaultExecFn returns. When the tmux subprocess exits (EOF, ctx cancel, kill on cancel), it remains a zombie PID until process death because no goroutine reaps it.

`exec.CommandContext` only sends a signal on ctx-cancel — does NOT reap. Long-running access node → PID exhaustion / fd leak per restart.

Production codepath only; not exercised in tests.

Confidence: HIGH. Severity: HIGH.

### H-02 — Send-on-closed-channel panic race

**File:** `internal/tmux/control.go:259-276` (dispatchLoop exit path)

```go
select {
case <-ctx.Done(): // graceful
    closeErrCh.Do(close(errCh))
default:
    select {
    case c.errCh <- ErrControlModeDropped:  // line 269
    default:
    }
    c.closeErrCh.Do(...)
}
```

Race: T1 scanner returns false (EOF, NOT via Close). T2 dispatchLoop evaluates `<-ctx.Done()` — not done, takes default. T3 caller invokes Close(), which calls cancel() AND `c.closeErrCh.Do(close(c.errCh))`. T4 dispatchLoop reaches line 269, sends to closed channel → **panic: send on closed channel**.

sync.Once only guards double-close; does NOT prevent send-after-close. select-with-default does NOT protect against send-on-closed. Violates go.md "No panics in library code".

Confidence: HIGH. Severity: HIGH.

### H-03 — F-01 enumeration is fictional (test-masked-fix recurrence)

**File:** `internal/tmux/control.go:120-156` Connect; lines 235-253 dispatchLoop %begin/%end branch

Connect docstring claims "sends a list-sessions command immediately after establishing the stream; the resulting %begin/%end block is consumed". The code does NOT send any command. `execFunc` returns only `io.ReadCloser` — no stdin write path exists. `defaultExecFn` only captures `cmd.StdoutPipe()`.

Real `tmux -C` does NOT spontaneously emit a list-sessions result block — that only appears in response to an explicit `list-sessions` (or `\nlist-sessions\n` injected on stdin).

Test passes only because fake stream synthesises %begin/%end block out of band. Against real tmux, AC-002 ("all current tmux sessions are enumerated and published") will fail: %begin/%end parsing branch never triggers.

execFunc signature design forecloses any honest fix without an API break. BC-2.04.001 PC-2 not satisfied in production.

**This is the SAME test-masks-defect pattern that pass 1 surfaced for F-01, recurring in the pass-1 fix itself.** Implementer added parsing but never added stdin write.

Confidence: HIGH. Severity: HIGH.

### H-04 — Connect not concurrent-safe despite struct doc claim

**File:** `internal/tmux/control.go:51-65` (struct doc), 134-156 (Connect)

Doc: "All exported methods are safe to call from any goroutine."

Connect at lines 134-156 reads and writes c.proc, c.cancel, c.execFn without any mutex. Two concurrent Connect calls both see nil, both spawn dispatchLoops, double-leak. Even Connect/Close concurrent: Close reads c.cancel/c.proc at 181/185 unsynchronised against Connect's writes at 139/151.

Struct has NO mutex despite the concurrency doc claim. Race detector will fire on any concurrent caller test.

Confidence: HIGH. Severity: HIGH.

## Medium Findings

### M-01 — AC-001 named test is a skipped stub; story trace mis-anchored

**File:** `internal/tmux/control_test.go:71-76`

Story spec S-3.01a line 47: AC-001 → "Test: TestTmuxControlMode_Connect_EstablishesConnection". This test calls `t.Skip("stub: todo() — implement Connect before enabling (S-3.01a AC-001)")`. Skip message contradicts the fact that Connect IS implemented.

AC-001 behavior ("Connect succeeds against fake stream") is in fact covered by `TestTmuxControlMode_Connect_EnumeratesSessions` (line 90), but the spec→test trace named in the story is to a skipped test.

Per axis A (AC↔BC↔test trace correctness), this is mis-anchoring and blocks convergence.

## Observations

- ARCH-08 §6.6 import compliance: session={frame,admission}; tmux={halfchannel,session}. Compliant.
- ARCH-09 classification matches code.
- Lock discipline: Publisher uses sync.RWMutex correctly; ListSessions/Get return value copies.
- UTC discipline maintained at session.go:79.
- No init(), no package-level mutable globals.
- No panics outside testdata in current code (H-02 is a latent runtime panic, not a code-literal panic).
- AC-004 test assertion is now real: `ds.Seq() >= 1` and `pub.Get("session-1")` — improvement over pass-1 `_ = sessions`.

## Novelty Assessment

Novelty: HIGH — pass 2 found defects more serious than pass 1. Three findings (H-01, H-02, H-04) are pre-existing latent bugs not surfaced in pass 1; H-03 is a TEST-MASKED-FIX recurrence of pass 1's F-01 — the implementer added parsing but never added the stdin write that would produce data to parse. Same test that masked pass-1 F-01 (synthesised %begin/%end out of band) masks the partial fix.

## Resolution decisions (from human review)

- H-03: change execFunc signature to (io.WriteCloser, io.ReadCloser, error); Connect writes "list-sessions\n" to stdin after Start.
- H-04: add sync.Mutex protecting lifecycle fields (proc, cancel, execFn).
- M-01: un-skip TestTmuxControlMode_Connect_EstablishesConnection; add real fake-stream assertion.
- H-01: add cmd.Wait() in a reaper goroutine inside defaultExecFn (or similar).
- H-02: restructure dispatchLoop exit path to avoid send-after-close (e.g., only-close-once at the end with a sync.Once, and don't send-then-close; just close and let consumers errors.Is via the closed-channel return value, OR send on a separate buffered channel before close).
