---
artifact_id: adv-S-3.01a-pass-06
review_target: S-3.01a-tmux-control-mode
producer: adversary
pass: 6
fresh_context: true
branch: feature/S-3.01a-tmux-control-mode
base: develop @ d54bf1a
tip: cb31ae1
findings_count: 1
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Adversary declared CONVERGED inline; orchestrator overrides to NOT_CONVERGED per BC-5.39.001 strict "ANY finding blocks convergence" reading. The single LOW finding (F-LOW-01) is a doc-only mechanical fix; streak resets to 0/3 but next clean pass starts the streak. Returned inline by adversary because tool profile is read-only.
---

# Adversarial Review — Pass 6 — S-3.01a

## Critical Findings
None.

## High Findings
None.

## Medium Findings
None.

## Low Findings

### F-LOW-01 — Stale "%sessions-changed" docstring promises behavior not implemented

**File:** `internal/tmux/control.go:291`

The dispatchLoop docstring (lines 283-297) lists `%sessions-changed — re-enumerate sessions` as a handled event. `handleLine` (lines 385-432) only switches on `%session-created`, `%session-closed`, `%output`. No re-enumeration path exists.

BC-2.04.001 PC-3/PC-4 are satisfied by per-event handlers; spec does not require `%sessions-changed`. Comment is aspirational and misleads readers.

Confidence: HIGH. Severity: LOW.

**Suggested action:** Remove the `%sessions-changed` bullet from the docstring or add TODO with story reference. No code change required.

## Observations

- Re-verified all pass-5 fixes correctly applied:
  - H-1: 2 MiB scanner buffer (control.go:304-305)
  - M-1: fragmentation loop (control.go:417-429); off-by-one verified by tracing 200 KiB → 4 chunks
  - M-2: c.wg.Wait() before proc.Close() (control.go:268-272)
  - H-02: sync.Once atomicity for errCh close
  - H-03: stdin pipe + list-sessions write
  - H-04: idempotency via proc!=nil || cancel!=nil guard
  - H-01: subprocess reap goroutine
- ARCH-08 §6.6 imports clean: session imports admission only (frame permitted, unused); tmux imports halfchannel+session only.
- ARCH-09 classification consistent: session=boundary, tmux=effectful.
- unescapeTmuxOutput boundary `i+3 < len(s)` correctly bounded.
- ListSessions returns value-copy (go.md rule 12 satisfied).
- M-1 fragmentation loop accounting: chunk boundaries cover [0, len(data)) exactly once.
- Close idempotency contract holds: nil-Connect Close + double-Close both no-ops.
- Pipe-close ordering: cancel → stdin.Close → wg.Wait → proc.Close → errCh close. Avoids proc.Close racing scanner.Scan.
- AC-004 docstring deferral ("VP-031 e2e against real tmux deferred") documented in test + story; not a blocker.

## Novelty Assessment

Novelty: LOW — only one stale doc comment. Pass-5 fixes (H-1, M-1, M-2, H-01..H-04, F-01..F-06) all verified end-to-end. Implementation has functionally converged; one comment cleanup remains.

## Streak status

Pass 1: NOT_CONVERGED (7 findings: 3H/3M/1L)
Pass 2: NOT_CONVERGED (5 findings: 4H/1M)
Pass 3: NOT_CONVERGED (2 findings: 1M/1L)
Pass 4: CONVERGED (0 findings)
Pass 5: NOT_CONVERGED (4 findings: 1H/2M/1L)
Pass 6: NOT_CONVERGED (1 finding: 1L) [adversary declared CONVERGED; orchestrator overrides per BC-5.39.001 strict reading]

**Streak: 0/3 toward BC-5.39.001.**

## Resolution decision (mechanical)

F-LOW-01: remove or annotate the stale `%sessions-changed` bullet in dispatchLoop docstring. No code change. Implementer one-line fix.
