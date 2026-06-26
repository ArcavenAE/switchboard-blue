---
artifact_id: adv-S-3.01a-pass-04
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 4
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: 73de969
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 4 — S-3.01a

## Critical Findings
None.

## High Findings
None.

## Medium Findings
None.

## Low Findings
None.

## Nitpicks
None.

## Observations (informational, non-blocking)

- unescapeTmuxOutput (control.go:384-417) is exercised only indirectly via TestTmuxControlMode_OutputEventsFeedDownstream, which asserts ds.Seq() >= 1 but not payload bytes. Direct table-driven coverage of malformed octal, trailing backslash, and `\\` cases would tighten AC-004; function is correct against canonical test vector; boundary check i+3 < len(s) is sound.
- dispatchLoop treats every %begin/%end block as a session-list response (control.go:296-302). Safe today because Connect only ever issues one command (list-sessions). If future stories add additional commands, inSessionList flag will mis-classify response payloads. Note for S-3.01b / S-3.02 design surface; no current defect.

## Audit axes traversed (zero findings on each)

- AC-001..AC-004 trace fidelity: all four ACs have a named test that exercises spec-described behavior.
- ARCH-08 §6.6 imports: session imports {admission} (subset of allowed {frame, admission}); tmux imports {halfchannel, session}. No forbidden edges.
- ARCH-09 classification: session boundary; tmux effectful. Headers match.
- Error taxonomy: ErrControlModeUnavailable/ErrControlModeDropped/ErrAlreadyConnected sentinels at package level with godoc; ErrSessionNotFound/ErrSessionAlreadyPublished likewise.
- No log.Fatal, os.Exit, panics, init(), globals.
- Lock discipline: Publisher.mu (RWMutex) guards all map access; readers RLock, writers Lock. ControlMode.mu guards lifecycle fields; dispatchLoop never touches those.
- Goroutine lifecycle: Connect spawns one dispatchLoop. Close → cancel() → exec.CommandContext SIGKILL → reap goroutine awaits cmd.Wait → no zombie. dispatchLoop exits via scanner EOF or in-loop ctx select; both funnel through closeErrCh.Do.
- H-02 close safety: dispatchLoop's send-to-errCh wrapped in same sync.Once.Do as close(errCh) (control.go:334-341); send+close atomic relative to Close's competing Do(close). No send-on-closed-channel race.
- H-04 idempotency: second Connect on non-nil c.proc||c.cancel returns ErrAlreadyConnected (control.go:167-170); test asserts (control_test.go:107).
- Close idempotency: zeroes pointers + Once-guards close. Multiple calls are no-ops.
- Locked-accessor contract: Publisher.ListSessions returns []Info value copies, not pointers; explicit test mutates returned slice and re-reads to confirm no leakage.
- UTC timestamps: Publisher.Publish uses time.Now().UTC().
- Hermetic tests: all four AC tests inject WithExecFunc; no test shells out to tmux.
- Subprocess reaping: cmd.Wait() invoked in goroutine on every successful Start.
- F-01 enumeration genuine: AC-002 verifies three session names from %begin/%end block + post-block lifecycle.
- Octal-unescape boundary: i+3 < len(s) correctly allows last 4-char \NNN sequence at string tail.
- Errors wrapped with %w; errors.Is used by callers.
- Story spec trace fidelity matches BC-2.04.001 PC-1..PC-5, ARCH-09, ARCH-08 §6.6.

## Novelty Assessment

Novelty: NONE — no findings of any severity. Implementation internally consistent with BC-2.04.001, AC-001..AC-004, ARCH-08 §6.6 positions 6/7, and ARCH-09 boundary/effectful classifications. Lock discipline, goroutine lifecycle, channel-close safety, subprocess reaping, idempotency, and hermetic testing all check out under fresh-context review.

## Streak status

Pass 1: NOT_CONVERGED (7 findings: 3H/3M/1L)
Pass 2: NOT_CONVERGED (5 findings: 4H/1M)
Pass 3: NOT_CONVERGED (2 findings: 1M/1L)
Pass 4: CONVERGED (0 findings)

**Streak: 1/3 toward BC-5.39.001.**
