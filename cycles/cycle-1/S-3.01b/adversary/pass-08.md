---
artifact_id: adv-S-3.01b-pass-08
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 8
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 1cbc737
findings_count: 2
findings_by_severity: {critical: 0, high: 1, medium: 0, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 8 — S-3.01b

## High Findings

### H-001 — SessionConnector silently drops async ErrControlModeUnsupportedFlag; no PTY fallback for production EC-001

**Files:** `internal/tmux/pty_fallback.go:591-594`, `internal/tmux/control.go:557-564`

```go
// pty_fallback.go:591-594
for err := range sc.ctrl.Err() {
    if !errors.Is(err, ErrControlModeDropped) {
        continue
    }
```

**Failure scenario (production EC-001):**
1. Real tmux exists in PATH; exec.LookPath succeeds
2. cmd.Start succeeds (subprocess starts then exits with "unknown option -- C")
3. defaultExecFn returns (pipes, classifyCh, nil) — no sync error
4. Connect returns nil; SessionConnector.Connect enters success path, starts watchAndFallback
5. dispatchLoop reads EOF, waits on classifyCh, receives ErrControlModeUnsupportedFlag (pass-7 logic), sends it to errCh, closes errCh
6. watchAndFallback receives the error; errors.Is(err, ErrControlModeDropped) is FALSE → continue → loop terminates (channel closed)
7. watchAndFallback returns with NO factory call, NO pty.Connect, NO log, NO sc.errCh signal
8. sc.active==sc.ctrl (dead), sc.inPTYMode==false, no ErrPTYDeviceUnavailable emitted

**Operator gets nothing.** BC-2.04.002 EC-001 ("PTY fallback; log: 'tmux version does not support -CC flag'") + invariant 3 ("fallback state is clearly communicated — never silent") violated.

**Test coverage gap confirms defect:**
- TestPTYProxy_EC001_OldTmuxVersion exercises SYNC error path (ctrlErr != nil branch)
- TestPTYProxy_EC001_OldTmuxVersion_ViaAsyncClassify exercises ControlMode.Err() directly but never wraps in SessionConnector
- Zero tests verify SessionConnector engages PTY fallback when classifyCh delivers ErrControlModeUnsupportedFlag

This is a NEW defect induced BY the pass-7 fix. Pass 7 routed the sentinel through Err(); pass 8 surfaces that the consumer was never updated.

Confidence: HIGH. Severity: HIGH.

## Low Findings

### L-001 — ControlMode.Err() godoc stale — does not document ErrControlModeUnsupportedFlag

**File:** `internal/tmux/control.go:340-345`

```go
// Err returns the error channel that receives a non-nil error when the
// control mode connection is lost (BC-2.04.001 EC-002; S-3.01b API surface).
//
// The channel receives ErrControlModeDropped and is then closed when the
// event loop exits unexpectedly.
```

Post pass-7, channel can also deliver ErrControlModeUnsupportedFlag when classification fires. Doc claims only ErrControlModeDropped + anchors only BC-2.04.001 EC-002 — missing EC-001. Pass-7's own contract documentation at lines 70-77 acknowledges both sentinels; public Err() godoc does not. This stale doc directly led to H-001 (watchAndFallback implementer trusted the godoc).

## Observations

- Pass-7 mechanics correct: select with classifyCh + 200ms timeout in control.go:539-550 well-formed.
- TestControlMode_ClassifySupersedesDrop + TestControlMode_ClassifyTimeoutFallback correctly pin both select branches at ControlMode layer.
- Close()/wg.Wait() ordering correct; 200ms wait may delay graceful Close in rare race but doesn't break correctness.
- ARCH-08 §6.5 imports clean across all 6 tmux files.

## Novelty Assessment

Novelty: HIGH. H-001 is currently-undetected production defect at SessionConnector layer; pass-7 surfaced but didn't address. NOT a refinement of a prior pass — a NEW gap induced by the pass-7 restructure.

## Resolution decisions (mechanical)

- H-001: broaden watchAndFallback filter to handle ErrControlModeUnsupportedFlag. Either match any non-nil sentinel and dispatch on type, or explicitly add a second case for ErrControlModeUnsupportedFlag → log EC-001 message + invoke PTY fallback. Add SessionConnector-layer test exercising async-delivered ErrControlModeUnsupportedFlag.
- L-001: update Err() godoc to document both sentinels.

## Pattern observation

This is the 6th "test-masks-defect" pattern across S-3.01a + S-3.01b. The test boundary (ControlMode-only) ended where the production failure mode begins (SessionConnector consumer). vsdd-factory issue #287/#288 series captures the systemic class. Each pass continues to find one defect; the convergence model is working but slow.
