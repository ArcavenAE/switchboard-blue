---
artifact_id: adv-S-W3.05-pass-11
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 11
lens: spec-conformance / anti-tautology
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
reviewed_head: f6038d2
findings_count: 4
critical_count: 0
high_count: 0
medium_count: 0
low_count: 4
verdict: CONVERGED
date: 2026-06-27
streak_pass: 2
---

# Adversary Pass 11 — Spec-Conformance / Anti-Tautology Lens

## Verdict: CONVERGED

Zero CRITICAL. Zero HIGH. Four LOW observations (non-blocking).

## Scope

Conformance of code and tests to canonical specs at f6038d2: AC-013
three-panic-case coverage, E-ADM-017 canonical message phrase, v2.0
re-fire annotation semantics, VP-059 differential oracle
non-tautology. Cross-check story v1.3, BC-2.05.005 v1.7→v1.8
(sanction of SEC-001 precondition), BC-2.05.008 v1.3, error-taxonomy
v2.0.

## Verification Summary

**AC-013 three panic cases — non-tautology confirmed:**
The adversarial test table has three rows:
(1) `threshold <= 0` → panic;
(2) `window <= 0` → panic;
(3) `logger == nil` → panic (added by f6038d2).
A fourth row `wantPanic: false` with valid inputs serves as the control,
proving the panic assertion is non-tautological — a valid constructor
call does NOT panic.

**E-ADM-017 canonical message:**
The alert log line produced by the implementation contains the phrase
"HMAC failure rate alert:" — confirmed unchanged by f6038d2. The
error-taxonomy v2.0 annotation for re-fire is prose/drain-only
clarification; the message-format string is unchanged.

**v2.0 re-fire annotation — drain-only:**
error-taxonomy.md v2.0 annotates re-fire as occurring only when the
window drains fully (no live timestamps remain). The implementation
guards re-arm on `len(fc.sources[addr].times) == 0` (after drain). The
annotation and the code agree.

**VP-059 differential oracle — non-tautological:**
VP-059 proptest uses a stateful reference model. The oracle checks
(a) that the alert fires when the model says it should and (b) that it
does NOT fire when the model says it should not. Both polarities are
exercised across the three parameterized configs (seed 1337+idx). The
oracle is non-tautological.

**BC-2.05.005 v1.8 sanction:**
v1.8 adds the constructor precondition "logger must not be nil →
panic with message …" as a formally-sanctioned AC-013 sub-case.
The spec and the code now agree. No semantic change to alerting
behavior.

## Findings

### OBS-1 [LOW] comment / func-name drift (stale Red-Gate, TrackedSourceCount)

**Location:** `admission_test.go` Red-Gate comment; `failure_counter.go`
comment referencing TrackedSourceCount
**Observation:** Two pre-v1.6/v1.7 comments remain stale. Cosmetic
only; no behavioral impact.
**Action:** Defer to post-wave cosmetic cleanup.

### OBS-2 [LOW] AC-016 10k vs 1M magnitude placeholder

**Location:** `failure_counter_adversarial_test.go`, AC-016 iteration
comment
**Observation:** Iteration count comment reads 10k; spec annotation
mentions 1M as a validation target. The test is functionally correct
(behavior does not depend on iteration count); the comment is a
documentation placeholder.
**Action:** Discriminating at scale; cosmetic at current size. Defer.

### OBS-3 [LOW] stale Red-Gate header comment describes pre-v1.6 fired map[string]bool

**Location:** `failure_counter_adversarial_test.go`, Red-Gate test
setup
**Observation:** Comment describes the pre-v1.6 `fired map[string]bool`
model. The implementation uses per-source FailureWindow structs. The
test logic is correct; the comment is misleading.
**Action:** Cosmetic. Defer to post-wave cleanup.

### OBS-4 [LOW] BC-2.05.005 v1.7 vs v1.8 inline spec-version citations in test file

**Location:** `failure_counter_adversarial_test.go` header comment
**Observation:** The test file header cites "BC-2.05.005 v1.7"; after
the doc-only bump to v1.8 (SEC-001 sanction), the citation is one
version behind. No behavioral impact.
**Action:** Minor citation hygiene. Defer.

## Conclusion

Code and tests conform to canonical specs at f6038d2. AC-013
three-panic table is complete and non-tautological. E-ADM-017 phrase
unchanged. v2.0 drain-only re-fire annotation matches implementation.
VP-059 oracle is non-tautological. BC-2.05.005 v1.8 sanction is
consistent with the nil-logger guard. Four LOW cosmetic observations.
Pass 11 is CLEAN for streak counting purposes.
