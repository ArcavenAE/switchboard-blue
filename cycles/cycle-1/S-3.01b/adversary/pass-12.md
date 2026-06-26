---
artifact_id: adv-S-3.01b-pass-12
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 12
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 3628624
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager. Final pass in 3-consecutive-zero-finding streak (passes 10/11/12) — BC-5.39.001 satisfied for S-3.01b.
---

# Adversarial Review — Pass 12 — S-3.01b

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

## Review Coverage Summary

Verified across 15 standard axes:

1. **Spec-implementation alignment**: AC-001/002/003 named tests; EC-001/002/003/004 covered including both sync (string-match) and async (sentinel) classification paths.
2. **ARCH-08 §6.5 imports**: internal/tmux production limited to {halfchannel, session}. Tests import admission (allowed in _test.go).
3. **ARCH-09 classification**: internal/tmux declared effectful.
4. **Cross-platform builds**: Build tags exhaustive (darwin||linux, darwin, linux, !darwin && !linux). golang.org/x/sys/unix gated behind linux tag.
5. **Concurrency**: mu protects all SessionConnector lifecycle state. sync.Once on closeErrCh/closeFrames. wg.Wait coordination. H-01 (pass-6) snapshot-under-lock ensures coherent capture.
6. **Child-process lifecycle**: ptyMaster.Close kills child shell before master FD close. cmd.Wait reaper prevents zombies. Stderr drain synchronized via drainWG.
7. **Error sentinels**: All package-level vars with godoc. ST1005 compliance.
8. **E-SYS-001 mapping**: Sentinel text matches error-taxonomy.md. Full guidance via logger.
9. **Mandatory logs**: BC-2.04.002 PC-3 verbatim at pty_fallback.go:222. EC-001/EC-002 routed via controlModeFailureLogMsg.
10. **No silent failure**: Initial-connect double-failure → Connect return. Mid-session double-failure → Err() channel (M-003 path).
11. **No auto-upgrade**: Sticky inPTYMode. Structural test assertion (factory-call count stable post-fallback).
12. **Single-use ControlMode**: M-2 (pass-12) closed flag enforces ErrControlModeClosed.
13. **Factory contract defense**: L-002 (pass-9) (nil, nil) guard.
14. **3-attempt reconnect**: Bounded by maxReconnectAttempts=3.
15. **Test hermetism**: All non-skipped tests use WithExecFunc + WithPTYAllocFunc. Real-PTY deferred to VP-032.

## Novelty Assessment

Novelty: NONE — third consecutive clean pass. Implementation has reached steady state.

## Streak status

Pass 1: NOT_CONVERGED (2C/5H/3M/1L)
Pass 2: NOT_CONVERGED (1M/1L)
Pass 3: NOT_CONVERGED (1H/1M/1L)
Pass 4: NOT_CONVERGED (4M/5L)
Pass 5: NOT_CONVERGED (3H/3M/2L)
Pass 6: NOT_CONVERGED (1H/4L)
Pass 7: NOT_CONVERGED (1H/1M)
Pass 8: NOT_CONVERGED (1H/1L)
Pass 9: NOT_CONVERGED (3L)
Pass 10: CONVERGED (0 findings)
Pass 11: CONVERGED (0 findings)
Pass 12: CONVERGED (0 findings)

**Three consecutive clean passes — BC-5.39.001 satisfied for S-3.01b Step 4.5.**

## 12-pass summary

Cumulative findings caught + fixed: ~40 distinct defects across 9 NOT_CONVERGED passes. Defect classes surfaced:
- Test-masks-defect patterns (4× in S-3.01b, mirroring S-3.01a's 4×)
- Concurrency contract gaps (race, leak, lifecycle)
- Cross-platform PTY bugs (Ctty=slave.Fd wrong, child-shell kill missing, go.mod indirect)
- Spec-implementation drift (E-SYS-001 text, BC anchors, docstring accuracy)
- Stub-architect cross-namespace BC cite (filed as vsdd-factory #288)

All process-gap-class issues filed against drbothen/vsdd-factory (#272-#288).
