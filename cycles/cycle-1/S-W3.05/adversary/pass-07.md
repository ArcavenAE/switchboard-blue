---
artifact_id: adv-S-W3.05-pass-07
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 7
lens: spec-conformance + anti-tautology
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: b945aab
test_commit: 5c3d7ea
findings_count: 3
critical_count: 0
high_count: 0
medium_count: 0
low_count: 3
verdict: CONVERGED
date: 2026-06-27
---

# Adversary Pass 07 — Spec-Conformance + Anti-Tautology Lens

## Verdict: CONVERGED

Zero CRITICAL. Zero HIGH. Three LOW observations (non-blocking).

## Scope

Verified all five targets from the prior fix-loop cycle against canonical
specs. Focus: (1) exact message-phrase conformance against error-taxonomy.md:53
and BC-2.05.005; (2) drain-only re-arm logic; (3) per-source append-skip bound;
(4) VP-059 proptest non-tautology; (5) dead-key eviction discriminating test.

## Verification Summary

**E-ADM-017 canonical message phrase:**
error-taxonomy.md:53 specifies the phrase "HMAC failure rate alert:" as the
opening token of the alert message. Confirmed present in b945aab. The
formatted string matches: `"HMAC failure rate alert: ≥%d failures in %.0fs
from src %s"`. Cross-referenced BC-2.05.005 v1.6 and BC-2.05.008 v1.3 —
both cite the same phrase. No divergence.

**Drain-only re-arm:**
The dead `keep[0].After(lastFire)` branch has been removed. Re-arm triggers
only when the window fully drains (len(w.timestamps) == 0 after eviction).
BC-2.05.005 v1.6 § drain-only invariant satisfied.

**Per-source append-skip bound (CWE-770 / EC-011):**
Each source's timestamp slice is capped before append. When
len(w.timestamps) >= maxTrackedSources the new timestamp is skipped and a
counter incremented. BC-2.05.005 v1.6 §4 and VP-059 v1.1 EC-011 clause
satisfied.

**VP-059 proptest non-tautological:**
AC-016 uses 10k iterations per seed (not the 1M mentioned in a comment
placeholder). Seeds 1337+idx across 3 configs. The oracle is the independent
reference model (sequential scan over timestamps), not a re-statement of the
production code. Assertions discriminate — a tautological oracle (always-pass)
would fail the 3-config variance check. Non-tautological property confirmed.

**Dead-key eviction discriminating:**
AC-012 injects a source that fires then goes silent for >2 window-lengths.
Confirms eviction reduces TrackedSourceCount(). Test fails if eviction is
absent (verified by injecting a no-evict stub in test harness review).

## Findings

### O-1 [LOW] AC-016 iteration count 10k vs placeholder comment

**Location:** admission_test.go, AC-016 proptest
**Observation:** The test uses 10k iterations per seed. An adjacent comment
references 1M. The 10k figure is a documented deliberate choice (fast CI);
the comment is stale. Non-blocking; cosmetic.
**Action:** Deferred — cosmetic comment cleanup, post-wave or S-W3.05 PR
description note.

### O-2 [LOW] %.0fs window formatting is exact only for whole-second windows

**Location:** failure_counter.go, format string
**Observation:** `%.0f` truncates fractional seconds. For sub-second windows
the display rounds to 0s. In practice, window durations are configured in
whole seconds (BC-2.05.005 §2 window_seconds: int). No test exercises a
fractional-second window. Non-blocking; the spec uses integer seconds.
**Action:** No action required. Document as known precision limitation if
sub-second windows are ever introduced.

### O-3 [OBS] Step2/Step3 re-arm partial redundancy

**Location:** failure_counter.go, re-arm logic
**Observation:** The drain check (Step 2) and the lastFire reset (Step 3)
are both reachable via the same path when len(w.timestamps)==0. The
redundancy is harmless and was noted in pass-06. Not re-escalated.
**Action:** Deferred. No behavioral impact.

## Conclusion

All five fix-loop targets verified against canonical specs. No regressions
introduced by b945aab/5c3d7ea. Three LOW observations are cosmetic or
precision-precision edge cases outside the configured operating range.
Pass 07 is CLEAN for streak counting purposes.
