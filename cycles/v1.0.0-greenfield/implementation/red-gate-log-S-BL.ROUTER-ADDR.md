# Red Gate Log — S-BL.ROUTER-ADDR

**Story:** S-BL.ROUTER-ADDR v1.0 — populate PathSnapshot.RouterAddr with real resolved host:port
**BC:** BC-2.06.003 v1.15 PC-1
**Stub commit:** ec9397a
**Red Gate verified:** 2026-07-01

## Summary

| Metric | Value |
|--------|-------|
| Total test functions | 101 |
| New tests added | 12 |
| Failing at Red Gate | 5 |
| Passing at Red Gate | 96 |
| Packages tested | internal/paths, internal/metrics |

## New Tests — Red Gate Status

### internal/paths/paths_test.go

| Test | Status | Reason |
|------|--------|--------|
| `TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr` (4 sub-cases) | **FAIL** | `NewPathTrackerWithAddr` panics (BC-5.38.001 stub) |
| `TestBC_2_06_003_Snapshot_RouterAddr_Propagates` | **FAIL** | `NewPathTrackerWithAddr` panics (BC-5.38.001 stub) |
| `TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess` | PASS | GREEN-BY-DESIGN: `NewPathTracker` already sets `routerAddr=""` |
| `TestBC_2_06_003_NewPathTracker_Unchanged` | PASS | GREEN-BY-DESIGN: backward compat — `NewPathTracker` unchanged |
| `TestBC_2_06_003_NewPathTrackerWithAddr_RejectsInvalidAlpha` | PASS | GREEN-BY-DESIGN: stub panics unconditionally, so invalid-alpha cases also panic (correct behavior; will be strengthened post-impl to distinguish alpha-vs-stub panic) |
| `TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction` | **FAIL** | `NewPathTrackerWithAddr` panics (BC-5.38.001 stub) |
| `TestBC_2_06_003_RouterAddr_ConcurrentSnapshot` | **FAIL** | `NewPathTrackerWithAddr` panics (BC-5.38.001 stub) |

### internal/metrics/handlers_test.go

| Test | Status | Reason |
|------|--------|--------|
| `TestPathsList_PassesRouterAddr` | PASS | GREEN-BY-DESIGN: stub-architect already wired `snap.RouterAddr` through `PathsList` in `handlers.go:65`; test exercises the seam via `fakePathsListSource` (bypasses constructor) |
| `TestPathsList_RouterAddrEmptyForAddrLessSnapshot` | PASS | GREEN-BY-DESIGN: addr-less backward compat |

### internal/metrics/integration_test.go

| Test | Status | Reason |
|------|--------|--------|
| `TestVP047_RouterAddrNonEmpty/handler_seam_non_empty` | PASS | GREEN-BY-DESIGN: handler seam exercised via fake source |
| `TestVP047_RouterAddrNonEmpty/constructor_through_snapshot` | **FAIL** | `NewPathTrackerWithAddr` panics (BC-5.38.001 stub) |

## Failing Tests — Root Cause

All 5 failing tests share one root cause: `NewPathTrackerWithAddr` panics with:

```
paths: NewPathTrackerWithAddr not yet implemented (S-BL.ROUTER-ADDR stub — BC-5.38.001)
```

This is the correct Red Gate state. The stub-architect placed the panic per BC-5.38.001.

## AC Traceability

| AC | Tests | Red Gate |
|----|-------|----------|
| AC-001 (Snapshot().RouterAddr propagated) | `TestBC_2_06_003_Snapshot_RouterAddr_Propagates`, `TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess` | FAIL + PASS (GREEN-BY-DESIGN) |
| AC-002 (PathsList passes snap.RouterAddr) | `TestPathsList_PassesRouterAddr`, `TestPathsList_RouterAddrEmptyForAddrLessSnapshot` | PASS (GREEN-BY-DESIGN — handler seam already wired) |
| AC-003 (NewPathTrackerWithAddr constructor) | `TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr`, `TestBC_2_06_003_NewPathTracker_Unchanged`, `TestBC_2_06_003_NewPathTrackerWithAddr_RejectsInvalidAlpha`, `TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction`, `TestBC_2_06_003_RouterAddr_ConcurrentSnapshot` | FAILs + PASSes |
| AC-005 (VP-047 oracle flip) | `TestVP047_RouterAddrNonEmpty` | FAIL (constructor sub-test) |

## Green-by-Design Analysis

The following tests pass at Red Gate without implementation being complete. This is intentional and documented:

1. **`TestPathsList_PassesRouterAddr`** — tests the handler seam only (via `fakePathsListSource`). The stub-architect already replaced the hardcoded `""` with `snap.RouterAddr` per the story spec. This is the handler pass-through, not the constructor. The corresponding constructor test (`TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr`) fails.

2. **`TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess`** — `NewPathTracker` already sets `routerAddr=""` and `Snapshot()` already copies it. Correct per AC-001.

3. **`TestBC_2_06_003_NewPathTracker_Unchanged`** — pure backward compat check; `NewPathTracker` is not modified by this story.

4. **`TestVP047_RouterAddrNonEmpty/handler_seam_non_empty`** — handler seam test; same rationale as #1.

These GREEN-BY-DESIGN tests are deliberate: they exercise the seam that was already wired by the stub-architect per RULING-W6TB-B, and they will continue to pass post-implementation.

## Verdict

Red Gate **VERIFIED**. 5 tests fail for the correct reason (stub panic). All pre-existing tests pass (96 total). Implementer instruction: implement `NewPathTrackerWithAddr` to make all 5 failing tests pass.
