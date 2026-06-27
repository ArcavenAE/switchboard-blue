# S-W3.04 Review Findings — Convergence Tracking

**PR:** #17 — feat(S-W3.04): full daemon assembly — wire all Wave-3 subsystems
**Branch:** feat/S-W3.04-daemon-assembly → develop
**HEAD SHA:** 77c6229

## Convergence Table

| Cycle | Findings | Blocking | Advisory | Observations | Fixed | Remaining |
|-------|----------|----------|----------|--------------|-------|-----------|
| 1 | 4 | 0 | 0 | 4 | 1 (OBSERVATION-2 description correction) | 3 (no-action deferrals) |

**Result: APPROVE after cycle 1 (0 blocking findings)**

## Security Review (Step 4)

| Severity | Count | IDs |
|----------|-------|-----|
| CRITICAL | 0 | — |
| HIGH | 0 | — |
| MEDIUM | 1 | SEC-002: runtime.Gosched spin in ctrl→PTY swap window (latent Wave-4 concern) |
| LOW | 4 | SEC-001 (error detail in stderr), SEC-003 (ticker goroutines not in WaitGroup), SEC-004 (swapBarrier in prod struct), SEC-005 (double signal.NotifyContext) |
| INFO | 1 | SEC-006 (fmt.Errorf %s vs %w) |

No CRITICAL/HIGH. No implementer dispatch required.

## Review Cycle 1 Findings

### OBSERVATION-1 — Sweep & drop ticker goroutines not joined via WaitGroup
- **File:** `cmd/switchboard/access.go:208,214`
- **Detail:** `startSweepTicker` and `startFramesDroppedTicker` goroutines observe ctx.Done() cancellation and exit cleanly, but are not registered with `wg`. `wg.Wait()` returns after drain+bridge exit; tickers exit independently (bounded, not leaked). AC-008 test tolerates +1 goroutine. Acceptable for this story's scope.
- **Action:** None (tracked as SEC-003 for Wave-4 review)

### OBSERVATION-2 — PR description BC↔AC label inaccuracies (FIXED)
- **Detail:** Mermaid diagram and traceability table had AC-001 labeled as BC-2.04.001 (correct: BC-2.05.008 PC-2) and AC-009 labeled as BC-2.05.008 (correct: BC-2.04.002 EC-008 + BC-2.04.007 EC-007).
- **Action:** Fixed — pr-description.md updated; PR body updated via `gh pr edit 17`.

### OBSERVATION-3 — AC-008 TestDaemonCleanShutdown skips in sandbox
- **Detail:** No `/dev/ptmx` in sandbox environment. PC-2 clean-shutdown covered by `TestRunAccessWithConnectorPC2` (fakeConnector, no PTY). Expected and documented.
- **Action:** None (tracked as pending task #6: Wave-4 real-connector PTY-EOF lifecycle integration test)

### OBSERVATION-4 — AC-009 daemon-level integration deferred to Wave 4
- **Detail:** `TestForwardFramesPTYEOFExitsCleanly` verifies at `internal/tmux` unit boundary. Full daemon PC-2.6 drain → E-SYS-002 chain for PTY-EOF trigger not end-to-end tested with real connector.
- **Action:** None (tracked as pending task #6)

## Gate Status at PR Completion

| Gate | Status |
|------|--------|
| Per-story adversarial convergence (BC-5.39.001) | SATISFIED (3 clean passes 10/11/12 at 1c3c864) |
| Security review | PASS (0C/0H) |
| PR reviewer | APPROVE (cycle 1, 0 blocking) |
| CI checks | PASS (all required checks green) |
| Dependency PRs | ALL MERGED (#14, #12, #9) |
| Human two-party review | PENDING (merge gate) |
