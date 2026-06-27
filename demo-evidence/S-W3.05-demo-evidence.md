# S-W3.05 Demo Evidence: Per-Source HMAC Failure Counter and E-ADM-017 Alert

**Story:** S-W3.05 — per-source HMAC failure counter and admission alert (BC-2.05.005 PC-3)
**Story spec version:** v1.2
**Evidence type:** Go test transcripts (project standing preference — no VHS/terminal recordings)
**Date:** 2026-06-27
**Branch:** feat/S-W3.05-hmac-failure-counter

## Commit SHAs

| Role | SHA | Message |
|------|-----|---------|
| Implementation | b945aab | feat(S-W3.05): drain-only re-arm + append-skip + restore E-ADM-017 phrase |
| Tests (final) | 5c3d7ea | test(admission): parameterize VP-059 property tests over multiple configs |

## Test Files

| File | Package | ACs covered |
|------|---------|-------------|
| `internal/admission/failure_counter_test.go` | `admission_test` | AC-001–008, AC-010 |
| `internal/admission/failure_counter_adversarial_test.go` | `admission_test` | AC-004, AC-011–016 |
| `internal/admission/failure_counter_property_test.go` | `admission_test` | AC-017 (VP-059 a–e) |
| `internal/routing/routing_hmac_counter_test.go` | `routing_test` | AC-009 |

---

## Per-AC Coverage Table

| AC | Description | Covering Test(s) | Verdict | Notes |
|----|-------------|------------------|---------|-------|
| AC-001 | `FailureCounter` constructor fields, `NewFailureCounter` signature | `TestNewFailureCounter_ConstructorFields` | **PASS** | Non-nil, callable, no alert on 1 call |
| AC-002 | Sliding-window trimming (stale entries evicted) | `TestFailureCounter_SlidingWindowTrimsStaleEntries` | **PASS** | 3 failures at T=0, 2 more at T=61s → post-trim count=2, 2 timestamps held |
| AC-003 | E-ADM-017 emitted at threshold (exactly 1, contains code + srcAddr + full phrase) | `TestFailureCounter_EmitsEADM017AtThreshold` | **PASS** | Canonical form: "E-ADM-017 HMAC failure rate alert: ≥5 failures in 60s from src …" |
| AC-004 | Hysteresis / drain-only re-arm; no re-fire; `TestFailureCounter_RearmOccursOnFirstCallAfterDrain` | `TestFailureCounter_FiresOncePerCrossing`, `TestFailureCounter_RearmOccursOnFirstCallAfterDrain` (adversarial) | **PASS** | Fire-once-per-crossing; drain at T=65 clears firedAt; first post-drain call appends (len==1) |
| AC-005 | Hysteresis: re-fires after window fully expires | `TestFailureCounter_HysteresisRefirersAfterWindowExpires` | **PASS** | Batch-1 at T=0 → 1 alert; batch-2 at T=61–65 → 2nd alert; total=2 |
| AC-006 | Below-threshold: no alert for N−1 failures | `TestFailureCounter_BelowThresholdNoAlert` | **PASS** | 4 failures → 0 alerts, 4 timestamps held; 5th → 1 alert |
| AC-007 | Multi-source isolation: two srcAddrs counted independently | `TestFailureCounter_MultiSourceIsolation`, `TestFailureCounter_MultiSourceInterleaved` | **PASS** | addr-A fires 1 alert at 5; addr-B fires 1 alert independently |
| AC-008 | Boundary entry kept (strictly-less-than trim) | `TestFailureCounter_BoundaryEntryIsKept` | **PASS** | 1st entry at T=0; 5th at T=0+60s; T=0 kept (not < T=0); count=5 → alert fires |
| AC-009 | `WithFailureCounter` injection seam; both failure paths call `RecordHMACFailure`; success path does not | `TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths` (PATH-A, PATH-B, SUCCESS subtests), `TestRouteFrame_FiveConsecutiveFailures_TriggersEADM017` | **PASS** | PATH-A (no forwarding entry) and PATH-B (tag mismatch) each call counter once; SUCCESS path: 0 calls |
| AC-010 | Concurrent calls race-safe; `Timestamps()` returns copy | `TestFailureCounter_ConcurrentCallsRaceSafe` | **PASS** | 10 goroutines → 1 alert (fire-once); appending to returned slice does not mutate internal state |
| AC-011 | Memory cap: ≤65,536 tracked sources (LRU eviction) | `TestFailureCounter_SourceCapBoundsMapGrowth` | **PASS** | 65,537 insertions → `SourceCount()` ≤ 65,536 |
| AC-012 | Dead-key eviction: `delete(counts, srcAddr)` on drain; firedAt cleared | `TestFailureCounter_DeadKeyEvictedAfterDrain` | **PASS** | srcA+srcB drain; eviction observed via re-arm + fresh alert re-fire; SourceCount() correct throughout |
| AC-013 | Constructor panics on invalid args (`threshold<1` or `windowDuration<=0`) | `TestNewFailureCounter_PanicsOnInvalidArgs`, `TestFailureCounter_ConstructorValidation` | **PASS** | Zero/negative threshold panics; zero/negative window panics; valid args do not panic |
| AC-014 | Sustained attack re-fires periodically (not permanently silent) | `TestFailureCounter_SustainedAttackReFires` | **PASS** | 2 alert batches total; `SourceCount()==1` after sustained attack |
| AC-015 | E-ADM-017 canonical format: BOTH "E-ADM-017" AND "HMAC failure rate alert:" present | `TestFailureCounter_AlertMessageFormat`, `TestRouteFrame_EndToEnd_EADMAlertMessageFormat` | **PASS** | Full canonical: "E-ADM-017 HMAC failure rate alert: ≥5 failures in 60s from src deadbeef01020304" |
| AC-016 | Append-skip: per-source slice bounded at threshold entries under high-rate attack | `TestFailureCounter_HighRateAttackBoundedSlice` | **PASS** | 10,000 extra calls with frozen clock; `len(Timestamps)==5` (not 10,005) |
| AC-017 | VP-059 v1.1 property-based tests: properties a–e, stateful model checker | `TestFailureCounter_PropertiesABCD` (15 subtests), `TestFailureCounter_PropertyE_MemoryBound`, `TestFailureCounter_BoundaryEC008`, `TestFailureCounter_ConstructorValidation` | **PASS** | All 3 configs × 5 subtests each; memory bound with 66,536 adversarial sources; EC-008 boundary |

**Coverage: 17/17 ACs covered. No gaps.**

---

## Transcript Excerpts

### Admission Package — All Tests

Command: `go test -v -count=1 ./internal/admission/`

Key PASS lines for S-W3.05 tests (trimmed; full run also includes pre-existing admission tests):

```
--- PASS: TestNewFailureCounter_ConstructorFields (0.00s)
--- PASS: TestFailureCounter_SlidingWindowTrimsStaleEntries (0.00s)
--- PASS: TestFailureCounter_EmitsEADM017AtThreshold (0.00s)
--- PASS: TestFailureCounter_FiresOncePerCrossing (0.00s)
--- PASS: TestFailureCounter_HysteresisRefirersAfterWindowExpires (0.00s)
--- PASS: TestFailureCounter_BelowThresholdNoAlert (0.00s)
--- PASS: TestFailureCounter_MultiSourceIsolation (0.00s)
--- PASS: TestFailureCounter_BoundaryEntryIsKept (0.00s)
--- PASS: TestFailureCounter_NoAlertOnSuccess (0.00s)
--- PASS: TestFailureCounter_ConcurrentCallsRaceSafe (0.00s)
--- PASS: TestFailureCounter_MultiSourceInterleaved (0.00s)
--- PASS: TestFailureCounter_SustainedAttackReFires (0.00s)
--- PASS: TestFailureCounter_RearmOccursOnFirstCallAfterDrain (0.00s)
--- PASS: TestFailureCounter_SourceCapBoundsMapGrowth (0.02s)
--- PASS: TestFailureCounter_DeadKeyEvictedAfterDrain (0.00s)
--- PASS: TestNewFailureCounter_PanicsOnInvalidArgs (0.00s)
    --- PASS: TestNewFailureCounter_PanicsOnInvalidArgs/zero_threshold_panics (0.00s)
    --- PASS: TestNewFailureCounter_PanicsOnInvalidArgs/negative_threshold_panics (0.00s)
    --- PASS: TestNewFailureCounter_PanicsOnInvalidArgs/zero_window_duration_panics (0.00s)
    --- PASS: TestNewFailureCounter_PanicsOnInvalidArgs/negative_window_duration_panics (0.00s)
    --- PASS: TestNewFailureCounter_PanicsOnInvalidArgs/valid_args_do_not_panic (0.00s)
--- PASS: TestFailureCounter_AlertMessageFormat (0.00s)
--- PASS: TestFailureCounter_HighRateAttackBoundedSlice (0.00s)
--- PASS: TestRouteFrame_EndToEnd_EADMAlertMessageFormat (0.00s)
--- PASS: TestFailureCounter_SourceCount_RaceSafe (0.00s)
--- PASS: TestFailureCounter_PropertiesABCD (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/fires_exactly_at_threshold/threshold5_window60s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/fires_exactly_at_threshold/threshold3_window30s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/fires_exactly_at_threshold/threshold10_window120s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/suppressed_within_same_window/threshold5_window60s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/suppressed_within_same_window/threshold3_window30s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/suppressed_within_same_window/threshold10_window120s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/rearm_after_window_drain/threshold5_window60s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/rearm_after_window_drain/threshold3_window30s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/rearm_after_window_drain/threshold10_window120s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/periodic_refire_sustained_attack/threshold5_window60s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/periodic_refire_sustained_attack/threshold3_window30s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/periodic_refire_sustained_attack/threshold10_window120s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/stateful_model_generated_sequence/threshold5_window60s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/stateful_model_generated_sequence/threshold3_window30s (0.00s)
    --- PASS: TestFailureCounter_PropertiesABCD/stateful_model_generated_sequence/threshold10_window120s (0.00s)
--- PASS: TestFailureCounter_PropertyE_MemoryBound (0.86s)
--- PASS: TestFailureCounter_ConstructorValidation (0.00s)
    --- PASS: TestFailureCounter_ConstructorValidation/threshold_zero_panics (0.00s)
    --- PASS: TestFailureCounter_ConstructorValidation/window_zero_panics (0.00s)
--- PASS: TestFailureCounter_BoundaryEC008 (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/admission	1.151s
```

### Routing Package — RouteFrame + FailureCounter Tests

Command: `go test -v -count=1 -run 'TestRouteFrame' ./internal/routing/`

```
--- PASS: TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths (0.00s)
    --- PASS: TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths/PATH-A_no_forwarding_entry_calls_RecordHMACFailure (0.00s)
    --- PASS: TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths/PATH-B_tag_mismatch_calls_RecordHMACFailure (0.00s)
    --- PASS: TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths/SUCCESS_valid_HMAC_does_NOT_call_RecordHMACFailure (0.00s)
--- PASS: TestRouteFrame_FiveConsecutiveFailures_TriggersEADM017 (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.285s
```

---

## Race Detector Summary

Command: `go test -race -count=1 ./internal/admission/ ./internal/routing/`

```
ok  	github.com/arcavenae/switchboard/internal/admission	11.096s
ok  	github.com/arcavenae/switchboard/internal/routing	1.721s
```

**Result: PASS — zero data races detected.** AC-010 concurrent safety requirement verified.

---

## Coverage Gaps

None. All 17 acceptance criteria have at least one passing test transcript.

---

## Evidence Notes

- Evidence format: Go `go test -v` transcripts per project standing preference.
  No VHS/terminal recordings, no `.tape`/`.gif`/`.cast` files produced.
- Test transcripts are deterministic (clock-injected via `WithNow`; no wall-clock waits).
- Race detector run covers both packages containing S-W3.05 implementation.
- `TestFailureCounter_PropertyE_MemoryBound` took 0.86s for 66,536 adversarial source
  insertions — well within the 5s bound noted in the test comment.
- `TestFailureCounter_SourceCapBoundsMapGrowth` took 0.02s for 65,537 insertions
  (sequential, not parallel, per test comment).
