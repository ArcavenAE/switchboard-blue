# Review Findings — S-W5.02

**Story:** S-W5.02 — E2E Management Plane Integration Harness Across All Four Daemon Types
**PR:** #38 — https://github.com/ArcavenAE/switchboard-blue/pull/38
**Merge commit:** d881f99
**Merged at:** 2026-06-30

## Convergence Summary

| Cycle | Findings | Blocking | Fixed | Remaining |
|-------|----------|----------|-------|-----------|
| 1 | 9 | 1 | 1 | 0 |
| 2 | 0 | 0 | 0 | 0 → APPROVE |

Converged in 2 cycles (1 blocking finding resolved).

## Review Cycle 1

**Reviewer:** pr-reviewer (APPROVE, 0 blocking) + code-reviewer (REQUEST_CHANGES, 4 originally claimed blocking)

### Finding Triage

| ID | Reviewer | Severity | Category | Adjudication | Disposition |
|----|----------|----------|----------|--------------|-------------|
| CR-001 | code-reviewer | HIGH | code-quality | BLOCKING — double t.Cleanup(cancel) + redundant rawLn.Close after srv.Shutdown in bootstrap test | Fixed in 920b1c1 |
| CR-002 | code-reviewer | HIGH | spec-fidelity | Downgraded to NON_BLOCKING — closingConn.Read fires on both io.EOF and net.ErrClosed; design is intentional; baseline mechanism correctly scopes the observation window; adversarial review (3/3 clean) accepted this; race detector clean 17/17 | Deferred post-merge |
| CR-005 | code-reviewer | MEDIUM | correctness | Downgraded to NON_BLOCKING — goroutines drain on srv.Shutdown (force-closes all conns → ErrClosed); race detector clean | Deferred post-merge |
| CR-006 | code-reviewer | MEDIUM | correctness | Downgraded to NON_BLOCKING — net.Conn.Close() is idempotent; second call returns net.ErrClosed, discarded by _ =; no test failure | Deferred post-merge |
| CR-007 | code-reviewer | LOW | test-quality | NON_BLOCKING — bootstrap variant tests auth path only; AC-003 data assertions live in primary 4-daemon test | Deferred post-merge |
| CR-008 | code-reviewer | LOW | test-quality | NON_BLOCKING — handlers are test stubs; wire-protocol correctness is the assertion target | Deferred post-merge |
| CR-009 | code-reviewer | LOW | code-quality | NON_BLOCKING — dead code in closingListenerWrapper.closed map; minor technical debt | Deferred post-merge |
| SEC-001 | security-reviewer | LOW | security (CWE-400) | NON_BLOCKING — test-only polling busy-wait in waitForCloseAfter | Deferred post-merge |
| SEC-002 | security-reviewer | LOW | security (CWE-330) | NON_BLOCKING — nonConstantID() fallback to time.UnixNano; test-only, no crypto secret | Deferred post-merge |
| SEC-003 | security-reviewer | LOW | security (CWE-675) | Resolved by CR-001 fix — same double-close issue | Resolved in 920b1c1 |

### Fix Committed

**920b1c1** — `fix(S-W5.02): remove double-cancel and redundant rawLn.Close in bootstrap test`
- Removed standalone `t.Cleanup(cancel)` — cancel is called inside the consolidated cleanup closure
- Removed `_ = rawLn.Close()` — srv.Shutdown already closes the listener via `s.ln.Close()`

## Review Cycle 2

**Reviewer:** pr-reviewer
**Verdict:** APPROVE — 0 blocking findings
- CR-001 fix verified correct
- No new findings introduced
- All 5 ACs still covered in updated diff

## Post-Merge Deferred Items

| ID | Severity | Description |
|----|----------|-------------|
| CR-002 | LOW | closingConn.Read conflates server-shutdown ErrClosed with client FIN — intentional design, consider documenting |
| CR-005 | LOW | closingListenerWrapper goroutines not tracked in WaitGroup — drain on Shutdown, consider adding context cancellation |
| CR-006 | LOW | dialConn t.Cleanup double-close path — benign, consider sync.Once or comment |
| CR-007 | LOW | bootstrap test missing resp.Data assertion |
| CR-008 | LOW | mode-specific handler response payload not shape-asserted |
| CR-009 | LOW | closed map in closingListenerWrapper is dead code — can be removed |
| SEC-001 | LOW | waitForCloseAfter polling busy-wait — consider channel-based notification |
| SEC-002 | LOW | nonConstantID fallback pattern — consider t.Fatal instead of silent degradation |
