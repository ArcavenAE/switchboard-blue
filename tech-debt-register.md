---
artifact_id: tech-debt-register
document_type: tech-debt-register
version: "1.0"
status: active
last_updated: 2026-06-29
---

# Technical Debt Register: Switchboard

> Open items carry-forwarded from story adversary/pr-reviewer cycles.
> Resolved items move to `cycles/<cycle>/blocking-issues-resolved.md`.

## Open Items

| ID | Severity | Source | Description | Target | Status |
|----|----------|--------|-------------|--------|--------|
| SEC-LOG-001 | LOW (security) | S-W5.01 adversarial convergence / architect Ruling 1 | Security-event log on post-auth challenge_response protocol violation (BC-2.07.004 PC-3 / EC-004) is unimplemented. internal/mgmt/mgmt.go post-auth guard (~line 608) sends AUTH_FAIL + closes but emits no audit log; the internal/mgmt package has no logger seam and the daemon has no daemon-wide structured logging infrastructure (only one-off stdlib log.New(stderr) router scaffolding in cmd/switchboard/access.go). Fail-closed CONTROL (AUTH_FAIL + E-ADM-010 + close) is fully implemented and tested via VP-065; only the audit-log side effect is deferred. | S-HRD.02 (daemon logging infrastructure / slog seam on mgmt.Server) | deferred |
| F-002 | LOW | S-3.01b pr-reviewer | godoc Example sleeps (20/50ms) flaky on slow CI — bump to 200ms or poll-with-deadline | Wave 4 / test-hardening epic | open |
| F-003 | LOW | S-3.01b pr-reviewer | TestSessionConnector_NoAutoUpgrade_AfterFallback uses 200ms sleep as negative oracle — expose `lastReconnectAttemptedAt` test hook | Wave 4 / test-hardening epic | open |
| F-004 | LOW | S-3.01b pr-reviewer | PTYProxy.Connect(_ context.Context) discards ctx — pass ctx to ptyAlloc when alloc supports cancellation | Wave 4 | open |
| SEC-001 | LOW (security) | S-3.01b pr-reviewer | SHELL env var used without allowlist validation — add exec.LookPath-based allowlist in defaultPTYAlloc | Phase-6 hardening (security review) | open |
| VP-032 | deferred | S-3.01b story task 8 | Real-PTY integration test for pty_alloc_* — requires PTY device in test environment | Phase-6 / future wave | deferred |
| DRIFT-001 | LOW | consistency-validator (H-003/M-002/L-003) | BC-2.06.001 PC-5 ("surfaced via `sbctl sessions status` and console session-list view") ownership: sbctl half now owned by S-5.02 (AC-007, added v1.3); console session-list half deferred to S-7.03. BC-2.06.001 was previously orphaned with no story claiming PC-5. | S-7.03 (console session-list surfacing) | open |

## Resolved Items

> None yet. Resolved items will be moved to `cycles/cycle-1/blocking-issues-resolved.md`.
