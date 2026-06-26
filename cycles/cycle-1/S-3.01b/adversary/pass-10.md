---
artifact_id: adv-S-3.01b-pass-10
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 10
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 3628624
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager. First clean pass after 9 NOT_CONVERGED passes — streak 1/3 toward BC-5.39.001.
---

# Adversarial Review — Pass 10 — S-3.01b

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

## Attack Axes Exercised (no defects found)

1. **Spec drift (AC↔code↔spec):** AC-001..AC-003 each traced to BC-2.04.002 PC-1/PC-2/PC-3/EC-004 and exercised by named tests. Mandatory log strings match BC-2.04.002 text exactly.

2. **Silent failure / SOUL #4:** pty.Connect error path emits operator-facing guidance before returning sentinel. EC-003→PTY-fail surfaces ErrPTYDeviceUnavailable via Err() channel. Err() channel closed on graceful shutdown.

3. **Concurrency / races:** Reviewed Close vs watchAndFallback reconnect-swap interleaving (pass-6 H-01 lock-ordering holds). wg accounting balanced across reconnect-recursion. closeErrCh sync.Once invariant across watchAndFallback's three send paths and Close's idempotent close. PTYProxy.Connect's mutex-held duration vs ioRelay's bound master param clean. No race detected.

4. **Resource leaks:** ptyMaster.Close kills child shell before closing master FD. Reaper goroutine ensures cmd.Wait completes (no zombies). PTYProxy.Connect publish-failure path closes master to prevent leak. Factory contract-violation path closes returned newCtrl when accompanied by nil error. Setctty: true, Ctty: 0 correct for child's FD-0 namespace.

5. **Cross-platform build:** Build tags partition correctly. ptyMaster only defined where used. Unsupported-platform stub returns ErrPTYDeviceUnavailable.

6. **ARCH-08 §6.5 imports:** pty_fallback.go imports only halfchannel + session — matches position-7 allowed-import set. No forbidden imports (admission, routing, hmac) in production .go files.

7. **ARCH-09 purity:** internal/tmux is effectful in ARCH-09 — matches PTY alloc forking shell, ioRelay goroutine, OS pipes.

8. **Error sentinel correctness:** ErrPTYDeviceUnavailable wraps with errors.Is semantics. ST1005 compliance (no trailing period/newline in sentinel). controlModeFailureLogMsg uses errors.Is first; strings.Contains fallback documented.

9. **AC-003 + E-SYS-001 mapping:** Story AC-003 text matches implementation logger emit at line 194 exactly. error-taxonomy.md line 140 maps E-SYS-001 to same format.

10. **EC-003 reconnect count:** maxReconnectAttempts = 3 matches BC-2.04.002 EC-003 spec. Test asserts factory call count == 3.

11. **No-auto-upgrade invariant:** Once sc.inPTYMode = true is set, no code path resets it to false. State transition is one-way. Test enforces structurally via sticky factory-call count.

12. **Story frontmatter↔body coherence:** bc_traces, subsystems, architecture_modules all match ARCH-INDEX SS-04 mapping and BC declarations.

13. **Spec mis-anchoring:** None found. CAP-013, VP-032 anchors correct.

## Novelty Assessment

Novelty: NONE — no findings produced. Implementation has converged. After fresh-context attack across all 13 axes above, found zero defects. Multiple prior-pass fix markers (H-001..H-03, M-001..M-003, L-001..L-004, F-PASS7-H-001, F-04, M-2) and their accompanying tests demonstrate defects discovered in earlier passes all fixed in place.

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

**Streak: 1/3 toward BC-5.39.001.**
