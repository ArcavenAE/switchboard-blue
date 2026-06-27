# Review Findings — S-3.02

**PR:** #13  
**URL:** https://github.com/ArcavenAE/switchboard-blue/pull/13  
**Story:** S-3.02 — Console attach/detach and multi-console fan-out  

## Convergence Table

| Cycle | Findings | Blocking | Fixed | Remaining |
|-------|----------|----------|-------|-----------|
| 1 | 5 (2 MINOR, 3 COMMENT) | 0 | 5 | 0 → APPROVE |

Converged in 1 cycle.

## Cycle 1 Details

| ID | Severity | Description | Route | Resolution |
|----|----------|-------------|-------|------------|
| M1 | MINOR | Evidence files are .txt transcripts (VHS unavailable) | N/A | Accepted — fallback documented in evidence-report.md; -race PASS is authoritative for concurrent ACs |
| M2 | MINOR | "27 tests + 1 example" count unverifiable from diff | N/A | Resolved — ExamplePublisher_publishUnpublish in example_test.go (pre-existing file) |
| C1 | COMMENT | WithClock does not bind Publisher clock | N/A | Noted for S-3.03 |
| C2 | COMMENT | Vestigial upstream channel S-3.02-FM1 | N/A | Formally deferred to S-3.03; drift item recorded |
| C3 | COMMENT | AC coverage complete (positive) | N/A | No action |

## Security Review Summary

| Finding | Severity | In This Diff? | Resolution |
|---------|----------|---------------|------------|
| SEC-001 ($SHELL env var, pty_alloc_*.go) | HIGH | No — pre-existing S-3.01b | Out of scope |
| SEC-002 (unescapeTmuxOutput resource, control.go) | MEDIUM | No — pre-existing S-3.01a | Out of scope |
| SEC-003 (TOCTOU SendKeystroke/Detach) | LOW | Yes | Design-acknowledged; documented in PR |
| SEC-004 (NoOpAuthorizer default) | LOW | Yes | By-design per spec; documented |
| SEC-005 (upstream channel panic surface) | LOW | Yes | Contract documented; production path safe |

## Adversarial History (pre-PR)

| Pass | Verdict | Critical | High | Medium | Low |
|------|---------|----------|------|--------|-----|
| 01 | FINDINGS | — | — | — | — |
| 02 | FINDINGS | — | — | — | — |
| 03 | FINDINGS | — | — | — | — |
| 04 | FINDINGS | 0 | 0 | 0 | multiple |
| 05 | FINDINGS | 0 | 0 | 0 | multiple |
| 06 | CONVERGED | 0 | 0 | 0 | 1 |
| 07 | CONVERGED | 0 | 0 | 0 | 1 |
| 08 | CONVERGED | 0 | 0 | 0 | 0 |

## CI Status

| Check | Status |
|-------|--------|
| CodeQL | PASS |
| Analyze (go) | PASS |
| Quality Gate | PASS |
| dependency-review | PASS |
| Build Binaries | SKIP (expected on feature branch) |

## Local Quality Gates

| Gate | Status |
|------|--------|
| just fmt | PASS |
| just lint (0 issues) | PASS |
| go test -race ./... | PASS (27 tests + 1 example, race-clean) |

## Final Status

PRE-MERGE. Awaiting human approval.
