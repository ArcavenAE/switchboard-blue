---
artifact_id: adv-S-W3.05-pass-10
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 10
lens: security / nil-safety (CWE-476, CWE-400, CWE-117, CWE-770)
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
reviewed_head: f6038d2
findings_count: 2
critical_count: 0
high_count: 0
medium_count: 0
low_count: 2
verdict: CONVERGED
date: 2026-06-27
streak_pass: 1
---

# Adversary Pass 10 — Security / Nil-Safety Lens

## Verdict: CONVERGED

Zero CRITICAL. Zero HIGH. Two LOW observations (non-blocking).

## Context

This pass re-runs against the actual merge-tip f6038d2, which adds the
nil-logger eager guard in `NewFailureCounter` (SEC-001, CWE-476). Prior
passes 07-09 ran at 5c3d7ea (test commit), before the security-reviewer
found the nil-logger HIGH. This is the first fresh-context pass after
the fix.

## Scope

Nil-safety across all constructor-injected dependencies; CWE-476
reachability (nil-dereference panics); CWE-400 (uncontrolled resource
consumption — LRU eviction, append-skip bounds); CWE-117 (log injection
— srcAddr formatting); CWE-770 (allocation of resources without limits
— append-skip slice growth). SEC-001 fix correctness and absence of
regression to passes 07-09 findings (re-arm semantics, append-skip,
message format).

## Verification Summary

**SEC-001 nil-logger eager guard:**
`NewFailureCounter` checks `logger == nil` immediately, before any field
assignment, and panics with the exact message
`"admission: NewFailureCounter: logger must not be nil"`. No lazy
nil-deref path exists post-fix. The standalone test
`TestNewFailureCounter_PanicsOnNilLogger` covers this sub-case as the
third AC-013 row with `wantPanic: true` and confirms the exact message
string.

**No other reachable nil-deref:**
`threshold`, `window`, and `now` parameters are value types or function
values with zero-value semantics; none require nil guards beyond the
logger. The `hmacFailureRecorder` seam in routing is nil-checked before
calling `RecordHMACFailure`. No additional nil-deref paths found.

**CWE-117 log injection — srcAddr %x:**
`srcAddr` is passed to `RecordHMACFailure` as a `[]byte` and formatted
with `%x` (hex encoding) in the log message. Hex output is restricted to
`[0-9a-f]` characters; no newline, ANSI, or control character injection
is possible through this field.

**CWE-770 / LRU resource bounds:**
The LRU cap is a constructor parameter. `evictLRU` removes the
lowest-fire-count source when the cap is exceeded. `appendSkip` enforces
a per-source slice bound. Both bounds are compile-time-constant defaults
with override capability. Total memory is bounded.

**CWE-400 / append-skip:**
`appendSkip` does not append when `len >= cap`. The cap is the same LRU
cap. No unbounded growth.

**No regression to passes 07-09:**
Re-arm semantics (drain-only), E-ADM-017 canonical phrase
"HMAC failure rate alert:", dead-key discriminating test
(`wantPanic: false` control row), AC-012 dead-key path — all unchanged
by f6038d2 which only adds the nil-logger guard and its test.

## Findings

### OBS-1 [LOW] evictLRU empty-slice dead branch

**Location:** `internal/admission/failure_counter.go`, `evictLRU`
**Observation:** `evictLRU` checks `len(fc.sources) == 0` before
iterating to find the minimum. This branch cannot be reached: `evictLRU`
is called only when `len(fc.sources) >= cap`, and `cap >= 1` by
construction. The dead branch adds a redundant guard but causes no
behavioral issue.
**Action:** Cosmetic. Defer to post-wave cleanup.

### OBS-2 [LOW] append-skip lastFire-local trace not covered by focused test

**Location:** `internal/admission/failure_counter.go`, `appendSkip`
**Observation:** The `appendSkip` path that returns early (slice full)
is exercised indirectly by the LRU/cap tests but not by a single-focus
test explicitly asserting "no append when slice is at cap". The
behavioral invariant is correct and the CWE-770 bound holds; test
coverage is marginally sub-optimal.
**Action:** Cosmetic coverage gap. Defer to post-wave or include in a
future spec-hygiene burst.

## Conclusion

SEC-001 fix is correct and complete. The nil-logger guard is eager,
exact-message, and covered by a discriminating test. No other
nil-deref paths exist. CWE-117/400/770 bounds hold. No regression to
prior convergence. Pass 10 is CLEAN for streak counting purposes.
