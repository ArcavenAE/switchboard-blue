---
document_type: security-review
level: ops
version: "1.0"
status: complete
producer: security-reviewer
timestamp: "2026-08-30T14:15:00Z"
phase: 5
inputs:
  - internal/testenv/loopback.go
  - internal/testenv/testenv.go
  - internal/testenv/loopback_test.go
  - internal/bench/keystroke_echo_testenv_bench_test.go
  - docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/
input-hash: "7967a2f"
traces_to: .factory/stories/S-BL.LOOPBACK-FULLSTACK.md
total_findings: 0
critical: 0
high: 0
medium: 0
low: 0
files_reviewed: 4
---

# Security Review: S-BL.LOOPBACK-FULLSTACK (PR #135)

## Executive Summary

CLEAN. No CRITICAL/HIGH/MEDIUM/LOW findings against `internal/testenv`'s new tick-driven loopback
driver or its updated bench-file consumer. This is test-harness/library code, provably unreachable
from any production request path; the four highest-risk surfaces (dedicated auth triple, per-direction
mutexes, round-trip token map, bench-file API update) were each reviewed directly and found sound.

## Findings

None at CRITICAL/HIGH/MEDIUM/LOW severity.

### INFO-001: Shared `zeroSACK` package variable
- **Severity:** INFO
- **CWE:** N/A
- **Attack Vector:** N/A — not exploitable
- **Impact:** None. Passed by value at every call site; no aliasing/mutation risk observed.
- **Evidence:** `internal/testenv/loopback.go` package-level `zeroSACK` var, read-only usage confirmed
  at all call sites.
- **Proposed Mitigation:** None required; noted for completeness only.

### INFO-002: `SetSink`-absence reflection guard (favorable)
- **Severity:** INFO (positive finding, not a defect)
- **CWE:** N/A
- **Attack Vector:** N/A
- **Impact:** N/A — this is a regression-guard test (`TestSessionAccessNode_NoSetSinkMethod`, AC-007)
  that would catch a future weakening of production `session.AccessNode`'s construction-time-fixed
  `KeystrokeSink` invariant. Called out as a defensive measure worth preserving.
- **Evidence:** `internal/testenv/loopback_test.go`, AC-007 test group.
- **Proposed Mitigation:** None required — keep this test as-is.

## Summary Table

| ID | Severity | CWE | Location | Status |
|----|----------|-----|----------|--------|
| INFO-001 | INFO | N/A | internal/testenv/loopback.go (zeroSACK var) | accepted (non-issue) |
| INFO-002 | INFO | N/A | internal/testenv/loopback_test.go (AC-007) | accepted (positive finding) |

## Positive Findings (Defensive Measures Present)

- **Dedicated AccessNode/Publisher/SessionAuth triple** is real, fully-enforcing auth (not stubbed),
  structurally identical to the existing per-VP shard pattern — not a security shortcut.
- **Provably unreachable from production**: `grep -rl "internal/testenv" --include="*.go" . | grep -v
  _test.go` returns zero results — this harness cannot be imported by any shipped binary.
- **Fail-closed default**: `AccessNode`'s default sink fails closed (`ErrNoKeystrokeSink`) per repo
  convention; the loopback driver installs a real sink, no bypass of that invariant.
- **Per-direction halfchannel mutexes** correctly guard every `upstreamHC`/`downstreamHC` call site;
  no bypass found. `arq.ARQ`'s single-writer contract holds (`EnqueueSend`+`OnAck` always from the
  same single-goroutine call site). No lock-ordering deadlock cycle between `sinkMu` and
  `downstreamHCMu`.
- **RoundTrip token** is a monotonic atomic counter (not a security token, no collision risk); the
  `driver.pending` map has dedicated leak-prevention tests (AC-009, AC-011, AC-017) and its growth is
  bounded by test/bench call volume only — no untrusted input reaches this code.
- **`TestSessionAccessNode_NoSetSinkMethod`** (AC-007) is a reflection guard specifically designed to
  catch a future regression class: someone adding a `SetSink` escape hatch to production
  `session.AccessNode` to make this harness's shard-duplication pattern easier to replicate elsewhere.
  No other existing test in the repo would catch that regression.

## Recommendations Priority

### Immediate (before merge)

None.

### Before Release

None — this is test-harness/library code with no release-surface exposure.

### Post-Release

None.
