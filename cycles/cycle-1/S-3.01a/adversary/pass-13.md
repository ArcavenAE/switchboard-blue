---
artifact_id: adv-S-3.01a-pass-13
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 13
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ 609c6ae
tip: 5e54aa4
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 13 — S-3.01a

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

## Verification Summary (axes A–W, all clean)

1. **ARCH-08 §6.6 import positions** — internal/session imports {admission} (frame permitted-unused, documented); internal/tmux imports {halfchannel, session}. No forbidden edges.
2. **ARCH-09 classifications** — session=boundary; tmux=effectful. Matches.
3. **BC-2.04.001 PCs** — PC-1 (Connect), PC-2 (enumerate via list-sessions block), PC-3/4 (%session-created/closed), PC-5 (%output → downstream Tick + fragmented). All wired and tested.
4. **AC coverage** — AC-001..AC-004 each have named test. VP-031 deferred to integration harness per task 8 (rev 1.1).
5. **Concurrency** — single dispatchLoop goroutine joined via wg.Wait on Close. sync.Once guards both close(errCh) and close(frames). Send-on-closed avoided by atomic send+close inside Do. dispatchLoop is sole writer to downstream (halfchannel constraint). Test-side ds.Seq() reads sequenced after <-cm.Err() (happens-before via channel close/receive).
6. **Resource leaks** — subprocess reaped via cmd.Wait goroutine. stdin/stdout pipes closed on every error path. stdout closed after wg.Wait. frames buffered 256 with non-blocking drop.
7. **Single-use contract** — c.closed flag set in Close; Connect rejects with ErrControlModeClosed. Connect idempotency separately enforced via proc!=nil||cancel!=nil.
8. **Octal-escape decoder** — bounds-correct (i+1>=len(s) short-circuit; three-digit octal requires i+3<len(s)). \\\\ handled. Unrecognised escapes pass through.
9. **Fragmentation** — preserves PC-5 by chunking %output payloads to fit Enqueue's 65515-byte limit. Tested by TestTmuxControlMode_OversizePayload_Fragmented.
10. **Scanner buffer** — 2 MiB cap accommodates 100 KiB+ %output lines. scanner.Err() consumed; no silent ErrTooLong false-drop.
11. **%begin/%end block handling** — inSessionList flag gates Publish path. tmux protocol does not interleave async events inside command response blocks.
12. **Locked-accessor (go.md rule 12)** — Publisher.ListSessions returns []Info value copies; no internal pointer leak. Get returns Info by value. Test verifies mutation isolation.
13. **Error taxonomy** — ErrSessionNotFound = E-SES-001. ErrControlModeUnavailable/ErrControlModeDropped correspond to BC-2.04.001 EC-004/FM-011 and EC-002/FM-004 (no E-code per FM mapping: "No error code — degradation signal + log").
14. **UTC timestamps** — time.Now().UTC() per go.md rule 11.
15. **No init(), no log.Fatal, no panic** — clean.
16. **Multi-%w wrapping** — Go 1.20+ syntax; project pins Go 1.25.4. Compatible.
17. **No WriteString+Sprintf** — only io.WriteString with literal string. Acceptable per go.md rule 1.

## Novelty Assessment

Novelty: NONE — implementation clean across all standard adversarial axes. Specs coherent with code.

## Streak status

Pass 8: CONVERGED (0)
Pass 9: CONVERGED (0)
Pass 10: NOT_CONVERGED (1L)
Pass 11: NOT_CONVERGED (2M/3L)
Pass 12: NOT_CONVERGED (2M)
Pass 13: CONVERGED (0)

**Streak: 1/3 toward BC-5.39.001.**
