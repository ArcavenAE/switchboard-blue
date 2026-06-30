# Review Findings — S-6.02: SVTN Lifecycle and Key Management

**PR:** #34 — feat(S-6.02): SVTN lifecycle and key management via sbctl admin
**Branch:** feat/S-6.02-svtn-lifecycle-sbctl-admin
**Base:** develop
**Reviewer:** pr-manager / pr-reviewer (cycle 1)

---

## Convergence Table

| Cycle | Findings | Blocking | Fixed | Remaining |
|-------|----------|----------|-------|-----------|
| Security review | 5 (1 HIGH, 2 MEDIUM, 2 LOW) | 1 (SEC-001) | 1 (SEC-001) | 4 deferred |
| 1 (pr-reviewer) | 4 informational | 0 | 0 | 4 deferred |
| **Final** | **0 blocking** | **0** | **—** | **0** |

**Verdict: APPROVE**

---

## Security Review Findings

| ID | Severity | CWE | Description | Status |
|----|----------|-----|-------------|--------|
| SEC-001 | HIGH | CWE-285 | Admin RPC handlers not yet wired — role string parser gap risk for S-6.06 | FIXED — KeyRoleFromString added (commit 33ebc03) |
| SEC-002 | MEDIUM | CWE-20 | SVTN name unvalidated (length, charset) | DEFERRED to S-6.06 (Create not CLI-reachable in this PR) |
| SEC-003 | MEDIUM | CWE-362 | ExpireKey has lookup-then-act window (narrow TOCTOU) | DEFERRED to S-6.06 atomic primitive addition |
| SEC-004 | LOW | CWE-345 | fakeMgmtServer uses ephemeral daemon key not verified by client | DEFERRED — test-design observation for S-6.06 mutual-auth hardening |
| SEC-005 | LOW | CWE-770 | Orphan bootstrap keys under failed concurrent-Create not GC'd | ACCEPTED risk — management socket requires operator auth; entries unreachable |

---

## PR Review Cycle 1 Findings

| Finding | Severity | Category | Status |
|---------|----------|----------|--------|
| INFO-1: LookupByPubkey returns *AdmittedKey (go.md rule 12 drift DRIFT-F005) | INFORMATIONAL | Code style | Deferred — out of perimeter |
| INFO-2: E-ADM-004 vs E-ADM-018 cross-story error code alignment | INFORMATIONAL | Error codes | Deferred to wave-5 gate (documented in PR) |
| INFO-3: Demo evidence on factory-artifacts branch (.txt fallback) | INFORMATIONAL | Evidence | Accepted — VHS unavailable |
| INFO-4: mgmt_wire.go TODO(CR-002) unusual placement | INFORMATIONAL | Code style | Lint-clean, valid |

---

## Pre-Merge Gate Status

| Gate | Status |
|------|--------|
| Security review — 0 CRITICAL/HIGH | PASS (SEC-001 resolved) |
| PR reviewer — 0 blocking findings | PASS (APPROVE, cycle 1) |
| CI — all required checks green | PASS (CodeQL, Quality Gate, dependency-review, Harden-Runner, Analyze/go) |
| Dependencies merged | PASS (S-2.02 #6, S-6.01 #28, S-6.03 #32) |
| Merge authorization | PENDING — orchestrator must authorize |
