---
artifact_id: adv-S-W3.05-pass-01
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 1
lens: spec-conformance
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: 1e74d76
findings_count: 9
findings_by_severity: {critical: 0, high: 1, medium: 3, low: 0, observations: 5}
verdict: NOT_CONVERGED
streak_after_pass: 0
streak_reset_reason: high and medium findings present
timestamp: 2026-06-27
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 1 — S-W3.05

**Lens:** Spec-Conformance
**Verdict:** NOT_CONVERGED
**Counts:** 0C / 1H / 3M / 0L / 5 OBSERVATION

---

## High Findings

### C1 (HIGH) — Hysteresis re-fire rule contradicts BC-2.05.005 EC-005/PC-3 + AC-004

**Files:** `internal/admission/failure_counter.go:103-121`

The implemented hysteresis re-fire rule ("re-arm when oldest surviving entry is newer than last-fire timestamp") contradicts BC-2.05.005 EC-005/PC-3 and story AC-004 ("reset only when count drops below threshold") AS WRITTEN.

Under sustained ≥5/60s attack: the BC-literal rule fires once then remains silent; the implementation re-fires approximately every window. Neither behaviour is pinned by a test — tests assert only drain-to-zero end totals, making them non-discriminating (tautology smell). A mutant re-fire rule yields the same test outcome.

**Owner:** product-owner (adjudicate canonical re-fire semantics) → implementer + test-writer

---

## Medium Findings

### I1 (MED) [process-gap] — Hysteresis test cannot distinguish correct from wrong implementation

**File:** `internal/admission/failure_counter_test.go:333-372`

The only hysteresis test drains the window fully before every candidate re-arm check. Every candidate re-arm rule (correct OR mutant) yields exactly 2 alerts — the test cannot distinguish correct from wrong implementation. This is a discriminability gap in the test suite.

**Owner:** test-writer (add sustained-attack discriminator test)

### I2 (MED) — E-ADM-017 message format diverges from canonical taxonomy literal

**File:** `internal/admission/failure_counter.go:123-128`

The E-ADM-017 message prepends `"E-ADM-017 "` and parameterizes `"≥%d/%.0fs"` rather than using the canonical taxonomy literal `"≥5 failures in 60s"`. Tests only substring-match, so the divergence is invisible. Same drift class as W3-R3-F3/F4.

**Owner:** product-owner + implementer/test-writer

### I3 (MED) [process-gap] — WithNow clock seam absent from BC-2.05.005 PC-3 constructor signature

`WithNow` is a load-bearing clock seam for tests, but it is absent from BC-2.05.005 PC-3's constructor signature specification. No test asserts that the default clock is UTC (go.md rule 11).

**Owner:** product-owner + test-writer

---

## Observations

- **O1:** PATH-A E-ADM-016 "auth key unavailable" wording differs from taxonomy "tag mismatch" — pre-existing S-3.04 code, out of S-W3.05 scope. Deferred.
- **O2:** PC-5 wire-up CORRECT — both failure paths call `RecordHMACFailure`, success path does not. VERIFIED.
- **O3:** Boundary trim `<` vs `<=` CORRECT — keeps the boundary entry.
- **O4:** Concurrency + go.md rule 12 CLEAN — `Timestamps()` returns a copy.
- **O5:** E-ADM-017 correctly DISTINCT from E-ADM-016. VERIFIED.
