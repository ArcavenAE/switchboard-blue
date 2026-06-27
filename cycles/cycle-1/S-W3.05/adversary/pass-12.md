---
artifact_id: adv-S-W3.05-pass-12
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 12
lens: concurrency / integration
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
reviewed_head: f6038d2
findings_count: 3
critical_count: 0
high_count: 0
medium_count: 0
low_count: 3
verdict: CONVERGED
date: 2026-06-27
streak_pass: 3
---

# Adversary Pass 12 — Concurrency / Integration Lens

## Verdict: CONVERGED

Zero CRITICAL. Zero HIGH. Three LOW observations (non-blocking).

## Scope

Lock discipline at f6038d2; Router↔FailureCounter deadlock potential;
RecordHMACFailure exactly-once per failed frame; srcAddr hex
key==message consistency; EC-006 e2e correctness; ARCH-08 import
constraint; SEC-001 concurrency regression check; stuck-state and
memory-bound proofs; `-race` coverage.

## Verification Summary

**Lock discipline:**
`logger.Log` is called after `fc.mu.Unlock()` in `RecordHMACFailure`.
`Timestamps()` and `SourceCount()` return value copies (slices
re-allocated, counter by value). No internal pointer escapes the mutex
boundary.

**No Router↔FailureCounter deadlock:**
`RouteFrame` calls `r.hmacFailureRecorder.RecordHMACFailure(srcAddr)`
after releasing the RouteTable read-lock (`rmu.RUnlock()` happens before
the HMAC verification failure path calls `RecordHMACFailure`). The
FailureCounter acquires its own `fc.mu` inside `RecordHMACFailure`.
No other mutex is held concurrently. No circular dependency.

**RecordHMACFailure exactly-once per failed frame:**
Two failure paths in `RouteFrame`: (1) key not found; (2) HMAC tag
mismatch. Both converge at the single `RecordHMACFailure` call site
before returning the error. The success path bypasses this call.
Exactly one call per failed frame on both paths.

**srcAddr hex key==message:**
`srcAddr` passed to `RecordHMACFailure` is the same `[]byte` used as
the hex-encoded map key. The log message formats the same value with
`%x`. Key and message field are identical in encoding.

**EC-006 e2e:**
`routing_hmac_counter_test.go` exercises frame arrival → HMAC failure
→ FailureCounter increment → threshold cross → E-ADM-017 alert through
the logger seam. The assertion checks the "HMAC failure rate alert:"
prefix. AC-009 is covered end-to-end.

**ARCH-08 import constraint:**
`admission` package imports: confirmed `routing` is absent. Seam wired
through the `hmacFailureRecorder` interface injected into `routing`.
No circular import.

**SEC-001 no concurrency regression:**
The nil-logger guard is a single upfront check in `NewFailureCounter`
before any state is initialized. It does not introduce any new mutex
acquisition, channel operation, or shared-state write path. No
concurrency regression possible.

**Stuck-state proof:**
`RecordHMACFailure` always unlocks `fc.mu` via `defer fc.mu.Unlock()`.
No code path holds the lock indefinitely. No blocking I/O under the
lock.

**Memory-bound proof:**
LRU eviction caps `len(fc.sources) <= cap`. `appendSkip` caps per-source
slice. Combined: total entries bounded by `cap * cap` (source count ×
window depth). No unbounded growth.

**-race coverage:**
`go test -race ./internal/admission/...` and
`go test -race ./internal/routing/...` pass at f6038d2. No data races
detected.

## Findings

### OBS-1 [LOW] evictLRU empty-slice dead branch (carried)

**Location:** `internal/admission/failure_counter.go`, `evictLRU`
**Observation:** Same as pass-10 OBS-1. Unreachable guard. Cosmetic.
**Action:** Defer to post-wave cleanup.

### OBS-2 [LOW] VP-059 skeleton TrackedSourceCount drift (FIXED in v1.2)

**Location:** VP-059.md v1.2
**Observation:** VP-059 v1.1 had a skeleton `TrackedSourceCount` property
that drifted from the production method name `SourceCount`. VP-059 v1.2
corrects this. The doc-only fix was applied in the spec-hygiene pass.
Code already uses `SourceCount`. No residual drift.
**Action:** FIXED. Noting for completeness.

### OBS-3 [LOW] BC-2.05.005 stale test-vector / VP-table re-arm rows (FIXED in v1.8)

**Location:** BC-2.05.005.md v1.8
**Observation:** v1.7 had stale test-vector rows referencing the pre-v1.6
`fired map[string]bool` re-arm model. v1.8 removed these rows. The
fix was applied in the spec-hygiene pass. Code conforms.
**Action:** FIXED. Noting for completeness.

## Conclusion

Lock discipline is sound. No Router↔FailureCounter deadlock risk.
RecordHMACFailure fires exactly once per failed frame. srcAddr hex
encoding consistent. EC-006 e2e covered. ARCH-08 constraint holds.
SEC-001 guard introduces no concurrency regression. Memory and stuck-state
bounds proven. `-race` passes at f6038d2. Three LOWs: one carried cosmetic
dead branch, two already FIXED in v1.8/v1.2 spec-hygiene bumps. Pass 12
is CLEAN for streak counting purposes.
