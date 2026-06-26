---
artifact_id: adv-S-3.01b-pass-11
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 11
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 3628624
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager. Second consecutive clean pass — streak 2/3 toward BC-5.39.001.
---

# Adversarial Review — Pass 11 — S-3.01b

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

## Coverage walk

Verified axes:

- **AC coverage:** AC-001 (TestPTYProxy_FallbackOnInitialConnectFailure), AC-002 (TestPTYProxy_PublishesSessionAndLogs), AC-003 (TestPTYProxy_NoPTY_ReturnsErrSysOne).
- **EC coverage:** EC-001 sync + async, EC-002 string + sentinel, EC-003 (factoryCalls==3), EC-004, no-auto-upgrade, Err() surfacing, Err() close on graceful shutdown, successful factory reconnect.
- **BC-2.04.002 postconditions:** PC-1 sc.inPTYMode=true; PC-2 synthetic name "pty-<pid>"; PC-3 canonical log exact-match; PC-4 sessions accessible; Invariant 3 (never silent).
- **Error taxonomy:** E-SYS-001 exact-match against error-taxonomy.md:140. Sentinel wraps via %w.
- **ARCH-08 §6.5 position 7:** internal/tmux imports only halfchannel+session. No admission/routing/hmac in production code.
- **ARCH-09:** internal/tmux marked effectful (correct).
- **Cross-platform build:** pty_alloc_unix.go (darwin||linux), per-OS defaultPTYAlloc, pty_alloc_other.go stub.
- **Concurrency:** sync.Once guards closeErrCh/closeFrames. Mutex guards sc.active/ctrl/closed/connectCancel/inPTYMode. wg joins ioRelay + watchAndFallback (including reconnect-success spawn). Inner context cancel allows Close to unblock. closed-check between reconnect attempts.
- **Resource cleanup:** master.Close on Publish failure; cmd.Process.Kill in ptyMaster.Close; reaper goroutine consumes exit status (no zombie).
- **Classification supersedes drop:** dispatchLoop drains classifyCh with 200ms timeout.
- **Single-use ControlMode contract:** Connect rejects after Close with ErrControlModeClosed; factory creates fresh ctrl per reconnect; atomic swap under mu.
- **Spec anchoring:** subsystem session-access (SS-04) consistent across BC/story/ARCH-INDEX. architecture_module internal/tmux consistent. CAP-013 + DI-001 traced.

## Novelty Assessment

Novelty: LOW — fresh-context derivation reached the same architecturally-coherent state. No new defect classes identified. Code accommodates all four documented edge cases with explicit tests; matches BC/ADR/taxonomy text byte-for-byte where exact strings are mandated.

## Streak status

Pass 10: CONVERGED (0)
Pass 11: CONVERGED (0)

**Streak: 2/3 toward BC-5.39.001.**
