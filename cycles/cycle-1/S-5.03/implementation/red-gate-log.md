# Red Gate Log — S-5.03 Degraded-Path Flag

**Date:** 2026-06-28
**Branch:** feature/S-5.03-degraded-path-flag
**Test files modified:**
- `internal/paths/paths_test.go` — added 5 new test functions + subtests
- `internal/paths/paths_prop_test.go` — new file, gopter property tests

---

## Test Functions Added

| Test Function | AC | Result |
|---|---|---|
| `TestBC_2_02_003_PathTracker_IsDegraded` (6 subtests) | AC-001, AC-002 | FAIL |
| `TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation` (4 subtests) | AC-003 | FAIL |
| `TestBC_2_02_003_PathTracker_IsDegraded_Race` | AC-004 | PASS (race-safety only) |
| `TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded` | AC-001 (Snapshot consistency) | FAIL |
| `TestProp_IsDegraded_TracksEWMAThreshold` (gopter) | AC-005, VP-063 | FAIL |
| `TestProp_IsDegraded_RecoveryClears` (gopter) | AC-005, VP-063 recovery | FAIL |

## Red Gate Verification

**New tests failing (RED):** 5 test functions fail (the race test passes — by design).

**Failure reason for all failing tests:** `IsDegraded()=false, want true` — the
`updateDegraded` method is a no-op stub (`func (t *PathTracker) updateDegraded(_ float64) {}`),
so `t.degraded` is never set to true regardless of EWMA RTT value.

**Existing tests (21 functions, all PASS):** All 21 pre-existing tests pass without modification.

## Gopter Added

- **gopter v0.2.9** added to `go.mod` via `go get github.com/leanovate/gopter@v0.2.9`
- Used in `internal/paths/paths_prop_test.go` for VP-063 property tests
- No prior gopter usage in the project; VP-026 used a stdlib grid sweep instead

## Red Gate Command

```
go test ./internal/paths/ 2>&1
# Expected: 5 FAIL, 24 PASS (21 pre-existing + 1 race + 2 sub-tests that pass vacuously)
```

## Handoff Note

The implementer must make each failing test pass by replacing the no-op stub:

```go
// TODO(S-5.03): not implemented — stub body
func (t *PathTracker) updateDegraded(_ float64) {}
```

with:

```go
func (t *PathTracker) updateDegraded(rttMS float64) {
    t.degraded = rttMS > DegradedRTTThresholdMS
}
```

No other changes are needed in `paths.go` — `updateDegraded` is already called at the
correct callsites in `OnProbe` and `resetRTT`.
