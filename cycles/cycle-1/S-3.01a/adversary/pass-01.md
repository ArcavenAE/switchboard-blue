---
artifact_id: adv-S-3.01a-pass-01
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: f7525e9
findings_count: 7
findings_by_severity: {critical: 0, high: 3, medium: 3, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 1 — S-3.01a

## Critical Findings
None.

## High Findings

### F-01 — AC-002 spec drift: Connect() never enumerates existing tmux sessions

**File:** `internal/tmux/control.go:126-144`

AC-002 + BC-2.04.001 PC-2 require: "all current tmux sessions are enumerated and published; the returned session list matches what tmux ls reports." Implementation only registers a goroutine reacting to subsequent `%session-created` events. Never issues `list-sessions`/`tmux ls`, never parses a `%begin/%end` block. A real `tmux -C` server emits no `%session-created` for already-running sessions → PC-2 fails on first connect against any non-empty server.

Test `TestTmuxControlMode_Connect_EnumeratesSessions` (control_test.go:96-126) passes only because the fake stream synthesises `%session-created` lines — masking the production defect.

Confidence: HIGH. Severity: HIGH (load-bearing AC failure in production).

### F-02 — Three AC tests are not hermetic; shell out to real tmux

**File:** `internal/tmux/control_test.go:178-194, 223-238, 243-264`

`TestTmuxControlMode_OutputEventsFeedDownstream` (AC-004), `TestTmuxControlMode_NoSessionsOnStartup` (EC-003), `TestTmuxControlMode_ErrChannelSignalsDroppedConnection` (EC-002) all use `newTestControl(t)` (line 58-64) which does NOT pass `WithExecFunc`. Connect → defaultExecFn → exec.LookPath("tmux") → cmd.Start().

Violates the file's own hermetic constraint declared at lines 4-7. In CI without tmux: tests fail for environment reasons. With tmux: leaves orphan subprocess; ErrChannelSignalsDroppedConnection times out (real tmux -C does not exit on stdout EOF); NoSessionsOnStartup flaky.

TestOutputEventsFeedDownstream additionally never injects %output event and only asserts `_ = sessions` — does not test AC-004 at all.

Confidence: HIGH. Severity: HIGH.

### F-03 — Err() docstring contradicts implementation; channel documented as closed but is never closed

**File:** `internal/tmux/control.go:155-163, 194-219, 167-177`

Godoc line 157-158: "The channel is closed and sends ErrControlModeDropped when the event loop exits unexpectedly."

No `close(c.errCh)` exists anywhere (grep confirmed). dispatchLoop only performs non-blocking send on unexpected exit; Close does nothing to the channel.

S-3.01b consumers following the docstring will use `for err := range c.Err()` and block forever after Close. The contract S-3.01a publishes to S-3.01b is wrong on its face.

Confidence: HIGH. Severity: HIGH.

## Medium Findings

### F-04 — Connect is not idempotent and has no re-entry guard; leaks subprocess + goroutine on second call

**File:** `internal/tmux/control.go:124-144`

Godoc line 124-125 is internally contradictory: "Connect is idempotent — calling Connect on an already-connected ControlMode is an error."

No check on `c.proc != nil` or `c.cancel != nil`. Second Connect overwrites cancel (orphaning first context), overwrites proc (first ReadCloser unreferenced; child process leak), spawns second dispatchLoop sharing same publisher.

Severity MEDIUM — S-3.01b's reconnect logic is the realistic trigger.

### F-05 — Dead code: session.FrameTypeData re-export has no consumer

**File:** `internal/session/session.go:145-150`

Const `FrameTypeData = frame.FrameTypeData` documented as "so internal/tmux can reference it without importing internal/frame directly." grep "session.FrameTypeData" across the worktree returns ZERO hits. internal/tmux doesn't reference frame types — it Enqueues raw []byte into halfchannel (which assigns FrameTypeData internally at halfchannel.go:127).

GREEN-BY-DESIGN justification moot for code with no caller.

### F-06 — %output payload format wrong: pane-id parsing + octal-escape decoding both incorrect

**File:** `internal/tmux/control.go:240-249`

Real tmux control mode emits: `%output %<paneid> <octal-escaped data>`. Implementation does `strings.TrimPrefix(line, "%output ")` then splits on " " into ≤2 parts and takes parts[1] verbatim.

Pane id parsing happens to work (%12 treated as pane id). Data parsing wrong: tmux octal-escapes spaces, control bytes, non-printables as `\NNN`. Implementation forwards escaped string verbatim → downstream gets backslash-N-N-N sequences, not original byte stream.

AC-004 explicitly defers demonstration test to VP-031, but defect is in production code path VP-031 will exercise.

## Low Findings

### F-07 — Sentinel docstring mis-cites FM mappings

**File:** `internal/tmux/control.go:27-30`

ErrControlModeUnavailable docstring says "BC-2.04.001 EC-004 / FM-011" but body claims "FM-004 explicitly assigns no catalog code to this degradation signal." Mixes three failure modes (EC-002/FM-004 vs EC-004/FM-011 vs anonymous degradation).

## Observations

- Skipped test TestTmuxControlMode_Connect_EstablishesConnection has grep-discoverable VP-031 deferral docstring (axis N satisfied).
- ARCH-08 §6.6 import compliance: session={frame,admission}; tmux={halfchannel,session}. Compliant.
- ARCH-09 classification matches code (session=boundary; tmux=effectful).
- Lock discipline: Publisher uses sync.RWMutex; ListSessions/Get return value copies.
- UTC discipline: time.Now().UTC() at session.go:79.
- No init(), no package-level mutable globals.
- No panics outside testdata.

## Novelty Assessment

Novelty: HIGH — pass 1 against the diff. F-01, F-02, F-03 independent substantive defects each blocking convergence. F-01 (no enumeration on Connect) is load-bearing — falsifies AC-002/BC-2.04.001 PC-2 in production; test masks via fake stream's synthesised create events. Classic test-masks-defect pattern.
