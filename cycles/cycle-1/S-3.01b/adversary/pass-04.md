---
artifact_id: adv-S-3.01b-pass-04
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 4
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 7650bf6
findings_count: 9
findings_by_severity: {critical: 0, high: 0, medium: 4, low: 5, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 4 — S-3.01b

## Medium Findings

### M-001 — PTY master leaked on publisher.Publish failure path

**File:** `internal/tmux/pty_fallback.go:195-206`

If publisher.Publish returns non-idempotent error, Connect returns the error but p.master remains set to an open io.ReadWriteCloser. ioRelay not started. Caller has no convention to call Close on failed Connect. OS-level PTY + shell process leak.

Fix: On publish failure (non-idempotent), call `_ = master.Close()` and reset `p.master = nil` before returning.

### M-002 — Race on stderrBuf in defaultExecFn

**File:** `internal/tmux/control.go:178-200`

cmd.Wait() does NOT synchronize with user-launched io.Copy goroutines reading from StderrPipe(). Comment at L186-187 claims it does — incorrect. Go memory model requires explicit sync. Currently masked because classification is no-op TODO; will break when phase-6 wires it.

Fix: Use sync.WaitGroup to synchronize drain goroutine completion before reaper reads stderrBuf.

### M-003 — Mid-session PTY-fallback failure is silent to caller

**File:** `internal/tmux/pty_fallback.go:594-603`

If sc.pty.Connect fails after mid-session ctrl drop, goroutine returns silently. sc.inPTYMode stays false; sc.active still points to dropped ctrl. No SessionConnector.Err() channel. Caller has no programmatic way to detect mid-session fallback failure. Violates BC-2.04.002 invariant 3 at program-state level.

Fix: Expose SessionConnector.Err() <-chan error delivering ErrPTYDeviceUnavailable when mid-session fallback fails.

### M-004 — Zero unit coverage for new defaultExecFn stderr-capture logic

**File:** `internal/tmux/control.go:130-203`

S-3.01b additions (StderrPipe + drain goroutine + reaper + classification) reachable only via real tmux. All _test.go cases inject WithExecFunc. Real-tmux deferred to VP-032. New code has zero coverage; M-002 race wouldn't surface from existing tests.

Fix: Extract classifyStderr pure helper testable in isolation, OR add test with custom exec returning a process that writes to stderr.

## Low Findings

### L-001 — PTYProxy.Sessions() docstring claims "at most one session active" but returns all publisher sessions

**File:** `internal/tmux/pty_fallback.go:258-264`

When SessionConnector shares publisher between ctrl and pty, PTYProxy.Sessions() returns pty- session PLUS orphans from prior ControlMode (ControlMode.Close doesn't unpublish).

### L-002 — AC-002 test substring too loose

**File:** `internal/tmux/pty_fallback_test.go:229,272`

Test asserts only prefix; could match truncated message. Tighten to include "Functionality limited".

### L-003 — controlModeFailureLogMsg uses strings.Contains (Go-rules violation, transitional)

**File:** `internal/tmux/pty_fallback.go:444-460`

Falls back to string-matching despite project rule "never string matching". Documented as transitional. Tests never exercise the sentinel errors.Is branches in production form.

### L-004 — ErrControlModeUnsupportedFlag defined but never produced

**File:** `internal/tmux/control.go:54,197`

Sentinel referenced via `_ =` to suppress unused-import lint; never actually returned. errors.Is branch in controlModeFailureLogMsg is dead code in production.

### L-005 — defaultPTYAlloc is unconditional-failure stub; production has no PTY path

**File:** `internal/tmux/pty_fallback.go:607-616`

Production always returns ErrPTYDeviceUnavailable. Story Library table lists golang.org/x/sys/unix for PTY allocation; story Tasks never explicitly mandate the impl. Release-readiness concern.

## Observations

- Spec invariants confirmed: ARCH-08 §6.5 row 7 (tmux imports halfchannel+session); ARCH-09 effectful.
- AC-001..AC-003 + EC-001..EC-004 each have a test (verified by name match).
- closed+sync.Once+wg.Wait close pattern in both PTYProxy.Close and SessionConnector.Close is well-formed.
- E-SYS-001 sentinel text matches error-taxonomy.md (terse prefix; full guidance via logger.Log).
- Initial sc.ctrl race hypothesis WITHDRAWN after deeper trace (Close acquires mu before reading sc.ctrl-modifying path's prerequisites; happens-before chain through mu enforces safety).

## Resolution decisions (from human review)

- M-001: implementer closes master before returning on publish failure path.
- M-002: sync.WaitGroup synchronizes drain goroutine completion before reaper reads stderrBuf.
- M-003: implementer adds SessionConnector.Err() <-chan error; test asserts ErrPTYDeviceUnavailable delivered on mid-session fallback failure.
- M-004: extract classifyStderr pure helper; unit-test it.
- L-001..L-004: mechanical fixes.
- L-005: implement unix.Openpty in defaultPTYAlloc (significant scope expansion; ~50 LOC + go.mod change for golang.org/x/sys/unix).

## Novelty Assessment

Novelty: MEDIUM-HIGH. Genuinely new concerns: publisher.Publish leak path, stderrBuf race (memory-model violation masked by no-op), silent mid-session PTY-fallback failure (contract gap), lack of unit coverage for new defaultExecFn logic.
