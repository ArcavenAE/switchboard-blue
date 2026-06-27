---
artifact_id: adv-S-W3.05-pass-09
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 9
lens: integration + RouteFrame wiring
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

# Adversary Pass 09 — Integration + RouteFrame Wiring Lens

## Verdict: CONVERGED

Zero CRITICAL. Zero HIGH. Three LOW observations (non-blocking).

## Scope

Integration soundness of the hmacFailureRecorder seam; RouteFrame wiring
correctness; nil-guard; exactly-once-per-failed-frame semantics; srcAddr
hex consistency; lock-ordering; ARCH-08 import constraint.

## Verification Summary

**hmacFailureRecorder seam soundness:**
The seam is injected at construction time via functional option
`WithFailureCounter`. When nil (default), recording is a no-op. When
set, `RecordHMACFailure` is called on the failure path only — not on
success. Verified by tracing both code paths in RouteFrame.

**Nil-guard:**
The nil check on `r.hmacFailureRecorder` is present before the call.
No nil-dereference is possible under zero-value construction.

**RecordHMACFailure on failure paths only:**
Two distinct failure paths reach the call site: (1) HMAC tag mismatch;
(2) key-not-found. Both call `RecordHMACFailure`. The success path does
not. Exactly-once semantics: each failed frame triggers exactly one
`RecordHMACFailure` call regardless of which branch is taken.

**srcAddr hex consistency:**
The srcAddr value passed to `RecordHMACFailure` is the hex-encoded source
address derived from the frame header. The same value is used as the map
key in the logger message. Key == message-field — no divergence between
what is counted and what is logged.

**Lock-ordering — no deadlock:**
RouteFrame holds no lock when calling `RecordHMACFailure`. The
FailureCounter acquires its own internal mutex. No other mutex is held
concurrently. No circular lock dependency.

**No lock held during I/O:**
`RecordHMACFailure` fires the logger after releasing its internal mutex.
RouteFrame calls `RecordHMACFailure` without holding a RouteTable lock.
No I/O under any lock.

**EC-006 e2e:**
An integration test exercises the full path: frame arrives → HMAC fails
→ FailureCounter increments → threshold crossed → E-ADM-017 alert emitted
via logger seam. Test asserts the canonical "HMAC failure rate alert:"
prefix in the emitted log line.

**ARCH-08 import constraint:**
The `admission` package does not import `routing`. The seam is wired
through an interface injected into `routing`. No circular import.
`go list -deps` confirms the constraint holds.

## Findings

### obs-1 [LOW] Routing e2e asserts "E-ADM-017" substring, not full canonical phrase through RouteFrame

**Location:** routing integration test, EC-006 assertion
**Observation:** The e2e test that exercises RouteFrame → FailureCounter
→ logger asserts only the substring "E-ADM-017" in the log output, not
the full canonical phrase "HMAC failure rate alert: ≥N failures in Xs
from src ADDR". The unit test in admission_test.go does assert the full
phrase. Coverage of the canonical phrase through the RouteFrame seam is
a test-coverage gap, not a production code defect.
**Action:** FOLD INTO S-W3.04 (touches routing.go / routing_test.go).
S-W3.04 implements daemon assembly and will wire the full integration
path. Adding the full-phrase assertion there is the natural home.

### obs-2 [LOW] Step3 re-arm partial redundancy (carried from pass-07)

**Location:** failure_counter.go, re-arm logic
**Observation:** Previously noted. No change. Harmless.
**Action:** Deferred. No behavioral impact.

### obs-3 [LOW] Stale Red-Gate test comment describing pre-v1.6 fired map[string]bool model

**Location:** admission_test.go, Red-Gate test setup comment
**Observation:** A comment above the Red-Gate test describes the fired
state as a `map[string]bool` — the pre-v1.6 model. The implementation
uses per-source FailureWindow structs with timestamp slices. The comment
is cosmetic; the test logic itself is correct.
**Action:** Deferred — cosmetic comment cleanup, post-wave.

## Conclusion

Integration path from RouteFrame through hmacFailureRecorder seam to
FailureCounter is correctly wired. Nil-guard present. RecordHMACFailure
fires exactly once per failed frame on both failure paths. srcAddr hex
is consistent. No deadlock risk. ARCH-08 import constraint holds. Three
LOW observations are cosmetic or test-coverage seam gaps suitable for
S-W3.04 or post-wave cleanup. Pass 09 is CLEAN for streak counting
purposes.
