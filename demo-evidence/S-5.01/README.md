# Demo Evidence — S-5.01: Green/Yellow/Red Quality Indicator with Hysteresis

**Story:** S-5.01  
**Branch:** feat/S-5.01-quality-indicator  
**Module under test:** `internal/metrics` (`QualityIndicator`)

## Recording method

VHS is not installed in this environment. Evidence is captured as
reproducible `go test -v` command output (text files). Each file
contains the exact terminal output produced by running the named
`go test -run` command against the worktree at
`.worktrees/S-5.01/`.

To reproduce any artifact:

```
cd .worktrees/S-5.01
go test ./internal/metrics/... -v -count=1 -run <PATTERN>
```

---

## AC-001 — Threshold classification (Green / Yellow / Red)

**Traces to:** BC-2.06.001 PC-2, PC-3, PC-4  
**Canonical test:** `TestQualityIndicator_ThresholdClassification`  
**Artifact:** `AC-001-threshold-classification.txt`

**Command:**
```
go test ./internal/metrics/... -v -count=1 \
  -run "TestQualityIndicator_ThresholdClassification|TestQualityIndicator_ThresholdBoundary"
```

**What is demonstrated:**
- Green: RTT ≤ 100 ms AND loss ≤ 5% (exact boundary values inclusive)
- Yellow: RTT in (100, 500] ms OR loss in (5%, 20%] (OR-form per BC-2.06.001 PC-3)
- Red: RTT > 500 ms OR loss > 20% (Red-over-Yellow precedence per PC-4 / F-C3)
- 14 table-driven sub-tests covering all three bands and all boundary values
- Canonical BC-2.06.001 test vectors (RTT=15ms/loss=0% → Green; RTT=150ms/loss=3% → Yellow)

**Result:** PASS (14/14 sub-tests)

---

## AC-002 — Hysteresis on upgrade (3 consecutive measurements required)

**Traces to:** BC-2.06.001 invariant 3  
**Canonical test:** `TestQualityIndicator_HysteresisUpgrade`  
**Artifact:** `AC-002-hysteresis-upgrade.txt`

**Command:**
```
go test ./internal/metrics/... -v -count=1 \
  -run "TestQualityIndicator_HysteresisUpgrade|TestQualityIndicator_RedToGreenViaSixMeasurements|TestQualityIndicator_SingleGoodMeasurementNoUpgrade|TestQualityIndicator_HysteresisResetOnBadMeasurement"
```

**What is demonstrated:**
- Red → Yellow requires exactly HysteresisCount (3) consecutive Yellow-range measurements
- Yellow → Green requires exactly HysteresisCount (3) consecutive Green-range measurements
- Full Red → Green path requires 6 measurements (two 3-measurement windows, one per level)
- Single good measurement after Red does not upgrade (story EC-001)
- A bad measurement interrupting a streak resets the consecutive counter

**Result:** PASS (4 tests, 4 sub-tests)

---

## AC-003 — OnMissingFrame downgrade signal

**Traces to:** BC-2.06.002 PC-2, VP-052  
**Canonical test:** `TestQualityIndicator_MissingFrameDowngradeGreenToYellow`  
**Artifact:** `AC-003-missing-frame-downgrade.txt`

**Command:**
```
go test ./internal/metrics/... -v -count=1 \
  -run "TestQualityIndicator_MissingFrameDowngrade|TestQualityIndicator_MissingFrameSubthreshold|TestQualityIndicator_MissingFrameCounterReset"
```

**What is demonstrated:**
- 3 consecutive `OnMissingFrame()` calls degrade Green → Yellow (canonical BC-2.06.002 test vector)
- 3 more consecutive missing frames degrade Yellow → Red (one level at a time)
- Fewer than 3 consecutive missing frames do not downgrade (story EC-003)
- A successful `Update()` resets the missing-frame counter (BC-2.06.002 PC-4)
- A Yellow-range `Update()` also resets the counter (any received frame breaks the streak)

**Result:** PASS (5 tests)

---

## AC-004 — Degradation only goes down; upgrade only via hysteresis

**Traces to:** BC-2.06.001 invariant 3, BC-2.06.002 PC-2, VP-027  
**Canonical test:** `TestQualityIndicator_DegradationNeverSkipsLevel`  
**Artifact:** `AC-004-degradation-only-goes-down.txt`

**Command:**
```
go test ./internal/metrics/... -v -count=1 \
  -run "TestQualityIndicator_DegradationNeverSkipsLevel|TestQualityIndicator_RecoveryNeverSkipsLevel|TestQualityIndicator_DowngradeIsImmediate"
```

**What is demonstrated:**
- Green → Yellow → Red: no Green → Red skip during sustained degradation (VP-027)
- Red → Yellow → Green: no Red → Green skip during recovery (VP-027)
- Downgrade is immediate (no hysteresis in the "worse" direction; AC-004 / BC-2.06.001)
- Level sequence is observed and validated transition-by-transition

**Result:** PASS (3 tests)

---

## Property tests (VP-027, VP-052, VP-074)

**Artifact:** `property-tests-VP027-VP052-VP074.txt`

**Command:**
```
go test ./internal/metrics/... -v -count=1 -run "TestProp"
```

**What is demonstrated:**
- `TestProp_BC_2_06_001_NoSkipTransitionDuringDegradation`: 1000 random workloads — no Green→Red skip (VP-027)
- `TestProp_BC_2_06_001_NoRedToGreenSkipDuringRecovery`: 1000 random workloads — no Red→Green skip (VP-027)
- `TestProp_BC_2_06_001_QualityIsAlwaysValidEnum`: 1000 runs — Current() always in {Green, Yellow, Red}
- `TestProp_BC_2_06_001_GreenToRedSingleStep`: 1000 runs — single Red-range update from Green lands on Yellow
- `TestProp_BC_2_06_002_MissingFrameNeverSkipsLevel`: 1000 runs — OnMissingFrame degrades at most one level per call (VP-052)
- `TestQualityIndicator_OnMissingFrame_PropertyMonotone`: 1000 runs — monotone property

**Result:** PASS (6 property tests, 1000 samples each)

---

## Race detector (concurrent access)

**Artifact:** `full-suite-race-detector.txt`

**Command:**
```
go test -race ./internal/metrics/... -count=1
```

**What is demonstrated:**
- `TestQualityIndicator_ConcurrentUpdates`: 10 goroutines × 100 calls each (Update + OnMissingFrame + Current concurrently)
- No data races detected by `go test -race`
- Final state is always a valid Quality enum value

**Result:** PASS (no races detected)

---

## Summary

| AC | Artifact | Tests | Result |
|----|----------|-------|--------|
| AC-001 | `AC-001-threshold-classification.txt` | `TestQualityIndicator_ThresholdClassification` (14 sub-tests) + `TestQualityIndicator_ThresholdBoundary` | PASS |
| AC-002 | `AC-002-hysteresis-upgrade.txt` | `TestQualityIndicator_HysteresisUpgrade` + 3 supporting tests | PASS |
| AC-003 | `AC-003-missing-frame-downgrade.txt` | `TestQualityIndicator_MissingFrameDowngradeGreenToYellow` + 4 supporting tests | PASS |
| AC-004 | `AC-004-degradation-only-goes-down.txt` | `TestQualityIndicator_DegradationNeverSkipsLevel` + 2 supporting tests | PASS |
| VP-027/052/074 | `property-tests-VP027-VP052-VP074.txt` | 6 property tests × 1000 samples | PASS |
| Concurrency | `full-suite-race-detector.txt` | `go test -race` full suite | PASS (no races) |
