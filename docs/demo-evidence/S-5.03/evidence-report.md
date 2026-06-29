# Demo Evidence Report — S-5.03: Flag Paths as Degraded When EWMA RTT Exceeds 200ms Threshold

**Story ID:** S-5.03
**Branch:** feature/S-5.03-degraded-path-flag
**HEAD:** 0f00ec5
**Date:** 2026-06-28
**Product type:** Pure internal library (`internal/paths`) — no CLI or UI surface.
**Recording tool:** Captured `go test -v -race` output (VHS not applicable; no binary or TUI to record).

---

## Evidence Artifacts

| File | Description |
|------|-------------|
| `AC-all-degraded-path-flag.txt` | Full `go test -v -race -run 'IsDegraded\|Degraded' ./internal/paths/` output — all 15 cases PASS |

---

## AC → Test Mapping

### AC-001: IsDegraded() returns true when EWMA RTT > 200ms; Snapshot().Degraded mirrors it

**BC trace:** BC-2.02.003 postcondition 5

**Proving tests** (all in `TestBC_2_02_003_PathTracker_IsDegraded`, table-driven):

| Sub-case | What it proves |
|----------|---------------|
| `always_below_threshold` | RTTs of 10–199.9ms → IsDegraded=false for every probe |
| `always_above_threshold` | RTTs of 250–400ms → IsDegraded=true for every probe |
| `boundary_exactly_at_threshold` | RTT=200.0ms → IsDegraded=false (exclusive `>`, EC-001) |
| `loss_event_does_not_change_degraded` | Loss events (lossEvent=true) do not touch degraded flag (EC-004) |

**Supporting:** `TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded` — 9-step sequential probe sequence verifying IsDegraded() and Snapshot().Degraded agree with expected contract values at each step, including boundary at 200.0ms and 200.1ms.

---

### AC-002: Flag clears on same tick where EWMA drops below threshold; recovery is symmetric

**BC trace:** BC-2.02.003 postcondition 5 (recovery branch)

**Proving tests** (sub-cases in `TestBC_2_02_003_PathTracker_IsDegraded`):

| Sub-case | What it proves |
|----------|---------------|
| `transitions_above_then_recovers` | Single-probe recovery: OnProbe(250)→degraded=true, OnProbe(150)→degraded=false in same tick |
| `sustained_degradation_then_sustained_recovery` | 5 above-threshold probes (alpha=0.5, steady at 300ms), then first recovery probe at 50ms: EWMA=175ms < 200ms → degraded=false immediately. Subsequent probes stay false |

---

### AC-003: No stale degraded flag across reactivation (resetRTT path)

**BC trace:** BC-2.02.003 postcondition 6 (resetRTT branch); EC-003

**Proving test:** `TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation` — 4 sub-cases:

| Sub-case | Reactivating RTT | Expected |
|----------|-----------------|----------|
| `reactivation_at_250ms_degraded_true` | 250ms | IsDegraded=true |
| `reactivation_at_150ms_degraded_false` | 150ms | IsDegraded=false |
| `reactivation_at_200ms_boundary_not_degraded` | 200.0ms | IsDegraded=false (exclusive) |
| `reactivation_at_200_1ms_degraded_true` | 200.1ms | IsDegraded=true |

Each sub-case first establishes a non-degraded baseline at 50ms, deactivates via 3 consecutive misses, then reactivates with the test RTT — verifying the flag is set from the reactivating probe, not carried from pre-deactivation state.

---

### AC-004: Concurrent IsDegraded()/OnProbe()/Snapshot() — no data races under -race

**BC trace:** BC-2.02.003 invariant 2

**Proving test:** `TestBC_2_02_003_PathTracker_IsDegraded_Race`

- 6 writer goroutines × 100 probes each, alternating 250ms/50ms RTTs to exercise both transition directions
- 6 reader goroutines × 100 iterations each calling IsDegraded(), Snapshot(), IsActive() concurrently
- Run with `go test -race`: 0 data race reports

---

### AC-005: Property test — IsDegraded() iff EWMA > 200ms, no hysteresis (VP-063)

**BC trace:** BC-2.02.003 postcondition 5; anchors VP-063

**Proving tests** (in `paths_prop_test.go`, gopter v0.2.9+):

| Property | Sub-test | What it proves |
|----------|----------|---------------|
| Primary | `TestProp_IsDegraded_TracksEWMAThreshold` | For any non-empty RTT sequence (gopter generates 100 cases, RTT ∈ [0, 500]ms), IsDegraded() equals (ewmaFromSamples(samples) > 200.0). Algebraically verifies the contract against the reference EWMA formula. |
| Recovery | `TestProp_IsDegraded_RecoveryClears` | For any highRTT ∈ [200.1, 400.0] and recoveryCount ∈ [1, 50]: (a) 20 probes at highRTT+100ms must set IsDegraded=true (drive-phase check — prevents vacuous pass from stub); (b) recoveryCount+20 probes at 10ms must clear IsDegraded=false. |

Both properties: **100/100 cases passed** (gopter seed-based, deterministic replay available via `-gopter.seed`).

---

## Snapshot: Snapshot Mirrors IsDegraded

Additional coverage from `TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded`:
- Verifies `Snapshot().Degraded == IsDegraded()` at every probe step
- Also verifies both equal the **expected contract value** (not just that they agree with each other) — rules out stub returning a consistent but wrong constant

---

## Test Execution Summary

```
command: go test -v -race -run 'IsDegraded|Degraded' ./internal/paths/
```

| Test | Sub-cases | Result |
|------|-----------|--------|
| TestProp_IsDegraded_TracksEWMAThreshold | 100 generated | PASS |
| TestProp_IsDegraded_RecoveryClears | 100 generated | PASS |
| TestBC_2_02_003_PathTracker_IsDegraded | 6 sub-cases | PASS |
| TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation | 4 sub-cases | PASS |
| TestBC_2_02_003_PathTracker_IsDegraded_Race | concurrency | PASS |
| TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded | 9-step sequence | PASS |

**Total: 15 test functions / 215+ sub-cases — all PASS, 0 race conditions detected.**

Full suite (`go test -v -race ./internal/paths/`) also PASS with no regressions to previously delivered S-4.01 tests.

---

## Coverage Mapping Summary

| AC | Covered | Test(s) |
|----|---------|---------|
| AC-001 | Yes | TestBC_2_02_003_PathTracker_IsDegraded (always_below, always_above, boundary), TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded |
| AC-002 | Yes | TestBC_2_02_003_PathTracker_IsDegraded (transitions_above_then_recovers, sustained_degradation_then_sustained_recovery) |
| AC-003 | Yes | TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation (4 sub-cases) |
| AC-004 | Yes | TestBC_2_02_003_PathTracker_IsDegraded_Race (-race, 12 goroutines) |
| AC-005 | Yes | TestProp_IsDegraded_TracksEWMAThreshold, TestProp_IsDegraded_RecoveryClears (100 cases each, VP-063) |

**All 5 ACs covered. Success and error/boundary paths recorded for each.**
