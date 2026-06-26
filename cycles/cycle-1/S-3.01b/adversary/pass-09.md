---
artifact_id: adv-S-3.01b-pass-09
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 9
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 2550cbd
findings_count: 3
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 3, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 9 — S-3.01b

## Low Findings

### L-001 — ControlModeFactory docstring does not require Connected state

**File:** `internal/tmux/pty_fallback.go:332-341` (and option doc at 346-351)

```go
// ControlModeFactory constructs a fresh ControlMode for reconnect attempts.
// Per ControlMode SINGLE-USE contract (control.go ADR-010), each reconnect
// must produce a new instance. The factory captures the construction
// parameters (publisher, downstream, options) closed over from New.
```

Phrase "constructs a fresh ControlMode" doesn't say returned instance must already be Connected. At lines 660-674 watchAndFallback swaps sc.ctrl = newCtrl and then re-enters via recursive spawn at 687-688 without ever calling newCtrl.Connect(ctx). If factory implementer follows literal "constructs" wording and returns unconnected ControlMode, recursive `for err := range sc.ctrl.Err()` blocks forever (dispatchLoop never started; errCh never written).

Implicit-contract gap. EC-003 happy reconnect can deadlock silently in misuse — invariant 3 says "never silent."

Fix: add contract sentence to godoc requiring Connect(ctx) completed successfully before return.

### L-002 — watchAndFallback reconnect-success branch lacks nil-guard for newCtrl

**File:** `internal/tmux/pty_fallback.go:660-674`

```go
newCtrl, connErr := sc.factory(ctx)
if connErr == nil {
    sc.mu.Lock()
    if sc.closed {
        sc.mu.Unlock()
        _ = newCtrl.Close()   // nil deref if newCtrl == nil
        return
    }
    oldCtrl := sc.ctrl
    sc.ctrl = newCtrl         // sc.ctrl set to nil if newCtrl == nil
    sc.active = newCtrl
```

Factory returning (nil, nil) — contract violation but unguarded — causes sc.ctrl = nil. Subsequent recursive `go sc.watchAndFallback(ctx)` crashes on sc.ctrl.Err() (nil pointer deref). Closed-during-swap branch also has latent nil deref.

Pairs with L-001 (ambiguous contract → violating factory plausible).

### L-003 — No test exercises successful factory reconnect path

**File:** `internal/tmux/pty_fallback_test.go` (test file as a whole); production code path at `pty_fallback.go:661-690`

All 3 factory-using tests inject factories that ALWAYS return (nil, error):
- TestSessionConnector_AsyncEC001_PTYFallback (lines 458-463)
- TestSessionConnector_MidSessionFallback_ReconnectAttempts (lines 639-645)
- TestSessionConnector_NoAutoUpgrade_AfterFallback (lines 736-741)

The "factory returns connected ctrl" branch — including ctrl swap, oldCtrl.Close(), and recursive `go sc.watchAndFallback(ctx)` at lines 687-688 — has NO test coverage. wg-balance assertion in comment 685-686 not exercised.

Untested code in fallback subsystem. Successful-reconnect path performs subtlest goroutine handoff in the file (Add(1) → spawn → return → defer Done() must balance correctly across swap). Regression here would silently leak goroutines or stop watching the new ctrl. Per BC-2.04.002 EC-003 semantics, successful reconnect is the explicitly-allowed happy path.

Fix: add test supplying factory whose first call returns freshly-Connected ControlMode over controllable fake stream; assert (a) factory called once, (b) sc.InPTYMode() remains false, (c) sc.Sessions() reflects new ctrl's sessions, (d) subsequent drop on new ctrl triggers another reconnect/fallback cycle.

## Resolution decisions (mechanical)

- L-001: implementer adds Connect-required sentence to ControlModeFactory godoc.
- L-002: implementer adds `newCtrl != nil` guard in both branches of the success path.
- L-003: implementer adds TestSessionConnector_FactoryReconnectSucceeds covering the recursive watch + wg balance.

## Novelty Assessment

Novelty: MEDIUM. Findings localized to rarely-exercised "successful factory reconnect" sub-path — region distinct from EC-001 async fallback that earlier passes scrutinized. L-001 (factory contract) and L-002 (nil guard) are companion defects; L-003 is structural reason both remained latent.

## Pattern observation

9 passes; defect-density curve still finding 1-3 issues per pass. Notable that each pass-N fix tends to surface a related defect class in pass-N+1 — pass-7 routed sentinel; pass-8 caught downstream consumer didn't handle it; pass-9 catches the path-that-shows-it-works has no test. Convergence model working but slowly; defect-density per pass is now LOW which suggests we're nearing fixed point.
