---
artifact_id: adv-S-3.01b-pass-03
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 3
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 2abdca6
findings_count: 3
findings_by_severity: {critical: 0, high: 1, medium: 1, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 3 — S-3.01b

## High Findings

### H-001 — PTYProxy.ioRelay discards Tick() return; no Frames() accessor

**File:** `internal/tmux/pty_fallback.go:213-234` (ioRelay)

Line 227: `_ = p.downstream.Tick()` — silently dropped. ControlMode publishes each dequeued frame to a buffered c.frames channel exposed via Frames() <-chan halfchannel.ChannelFrame (control.go:519-529). PTYProxy has NO equivalent accessor.

Consequence: every byte read from PTY master by ioRelay is enqueued into ds, immediately dequeued by Tick(), and dropped. PTY proxy mode is functionally non-delivering. BC-2.04.002 PC-2 ("publishes the PTY session to the SVTN") and PC-4 ("PTY session accessible for attach by a console") cannot be satisfied — frames never reach a consumer.

**Same defect class as S-3.01a pass-7 H-001** (Tick discard → no Frames() accessor). The asymmetry is created by S-3.01b's diff; PTYProxy was new in S-3.01b and did not mirror S-3.01a's wave-7 pattern.

Confidence: HIGH. Severity: HIGH.

## Medium Findings

### M-001 — ErrControlModeUnsupportedFlag declared but never emitted in production

**Files:** `internal/tmux/control.go:48-53`, `pty_fallback.go:396-412`, `pty_fallback_test.go:331-368`

Sentinel exported + consulted via errors.Is. No production code path produces it. defaultExecFn (control.go:133-164) wraps only with ErrControlModeUnavailable and ErrControlModeBinaryNotFound. TODO at control.go:50-52 acknowledges stderr-capture detection is deferred.

controlModeFailureLogMsg falls back to strings.Contains() but defaultExecFn never produces an error containing "-CC flag not supported" (cmd.Start failures don't carry tmux's stderr text).

EC-001 test passes vacuously via injected fake. BC-2.04.002 EC-001 log unreachable from defaultExecFn. Meta-gap: sentinel + test give appearance of coverage that production cannot exhibit.

Confidence: HIGH. Severity: MEDIUM.

## Low Findings

### L-001 — SessionConnector.Close() can block on wg.Wait if factory reconnect in flight

**File:** `pty_fallback.go:435-461` (Close), `:478-502` (factory loop)

Close sets sc.closed=true, closes ctrl+pty, calls wg.Wait(). Inside watchAndFallback, factory loop only checks sc.closed AFTER successful reconnect. For FAILED attempts, loop continues without re-checking sc.closed. If factory blocks (slow exec.LookPath or stderr-capture handshake), Close blocks on wg.Wait for the entire 3-attempt window.

Close does not cancel any context. Well-behaved factory honors ctx; graceful shutdown depending on Close alone (without caller ctx cancellation) can stall.

## Resolution decisions (from human review)

- H-001: Add PTYProxy.Frames() channel symmetric to ControlMode (buffered + sync.Once close + non-blocking send in ioRelay).
- M-001: Implement stderr-capture in defaultExecFn NOW (not defer). On cmd.Start or first-line stderr containing -C-related markers, wrap with ErrControlModeUnsupportedFlag.
- L-001: Add ctx cancellation in Close; factory loop checks sc.closed between attempts.

## Novelty Assessment

Novelty: HIGH for H-001 (functional integration gap; same class as S-3.01a pass-7); MEDIUM for M-001 (meta-gap pattern); LOW for L-001 (graceful shutdown nuance).
