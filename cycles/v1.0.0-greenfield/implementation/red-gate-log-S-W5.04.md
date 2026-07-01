# Red Gate Log — S-W5.04 Daemon Paths Metrics Handlers

**Story:** S-W5.04 — daemon-side paths.list / router.metrics / router.status RPC handlers  
**Date:** 2026-07-01  
**Phase:** TDD (test-writer pass)  
**BC-5.38.001 Status:** RED GATE VERIFIED

## Summary

20 new test functions written across 2 files. All 20 fail against the current
stubs (panic: TODO: not yet implemented). Pre-existing passing tests (metrics_test.go,
metrics_prop_test.go) still pass.

## Test Files

| File | New Tests | Status |
|------|-----------|--------|
| `internal/metrics/handlers_test.go` | 18 | FAILING (Red Gate — panic from stubs) |
| `internal/metrics/integration_test.go` | 2 | FAILING (Red Gate — panic from RegisterMetricsHandlers stub) |

## Per-Test Red Gate Results

| Test Name | AC/BC Trace | Failure Mode |
|-----------|-------------|--------------|
| `TestDaemonPathsList_HandlerRegistered` | AC-001 / BC-2.06.003 PC-1 | `panic: TODO: S-W5.04 PathsList not yet implemented` |
| `TestDaemonPathsList_EmptySource` | AC-001 / BC-2.06.003 EC-001 | `panic: TODO: S-W5.04 PathsList not yet implemented` |
| `TestPathEntry_RTTValueSerialization/row_a_count_0` | AC-002 / BC-2.06.003 PC-1 EC-003 | `panic: TODO: S-W5.04 RTTValue.MarshalJSON not yet implemented` |
| `TestPathEntry_RTTValueSerialization/row_b_count_9` | AC-002 / BC-2.06.003 PC-1 EC-003 | `panic: TODO: S-W5.04 RTTValue.MarshalJSON not yet implemented` |
| `TestPathEntry_RTTValueSerialization/row_c_count_10` | AC-002 / BC-2.06.003 PC-1 EC-003 | `panic: TODO: S-W5.04 RTTValue.MarshalJSON not yet implemented` |
| `TestPathEntry_RTTValueSerialization/row_d_count_100` | AC-002 / BC-2.06.003 PC-1 EC-003 | `panic: TODO: S-W5.04 RTTValue.MarshalJSON not yet implemented` |
| `TestPathEntry_StatusFromDegraded/active_false_is_failed` | AC-003 / BC-2.06.001 BC-2.06.003 PC-1 | `panic: TODO: S-W5.04 PathEntryFromSnapshot not yet implemented` |
| `TestPathEntry_StatusFromDegraded/active_degraded_is_degraded` | AC-003 / BC-2.06.001 | `panic: TODO: S-W5.04 PathEntryFromSnapshot not yet implemented` |
| `TestPathEntry_StatusFromDegraded/active_ok_is_active` | AC-003 / BC-2.06.001 | `panic: TODO: S-W5.04 PathEntryFromSnapshot not yet implemented` |
| `TestDaemonRouterMetrics_HandlerRegistered` | AC-004 / BC-2.06.003 PC-2 | `panic: TODO: S-W5.04 RouterMetrics not yet implemented` |
| `TestDaemonRouterMetrics_SVTNNotFound` | AC-004 / BC-2.06.003 EC-004 (E-RPC-011) | `panic: TODO: S-W5.04 RouterMetrics not yet implemented` |
| `TestDaemonRouterStatus_HandlerRegistered` | AC-005 / BC-2.06.003 PC-3 | `panic: TODO: S-W5.04 RouterStatus not yet implemented` |
| `TestDaemonRouterStatus_FailedAndPendingPrecedence/row_a_*` | AC-005a / BC-2.06.003 v1.8 EC-007 S502-DEFER-3 | `panic: TODO: S-W5.04 RouterStatus not yet implemented` |
| `TestQualityFromEntry_PendingWhenSampleCountLow` | BC-2.06.003 EC-006 EC-007 | `panic: TODO: S-W5.04 QualityFromEntry not yet implemented` |
| `TestQualityFromEntry_PendingWinsOverFailed` | BC-2.06.003 v1.8 EC-007 S502-DEFER-3 | `panic: TODO: S-W5.04 QualityFromEntry not yet implemented` |
| `TestQualityFromEntry_GreenWithSufficientSamples` | BC-2.06.003 PC-3 | `panic: TODO: S-W5.04 QualityFromEntry not yet implemented` |
| `TestQualityFromEntry_NeverEmitsFailed/*` | BC-2.06.003 PC-3 S502-DEFER-3 | `panic: TODO: S-W5.04 QualityFromEntry not yet implemented` |
| `TestVP047_SbctlPathsList_EndToEnd` | AC-006 / VP-047 / BC-2.06.003 PC-1 | `panic: TODO: S-W5.04 RegisterMetricsHandlers not yet implemented` |

## AC Coverage Map

| AC | Test Functions |
|----|---------------|
| AC-001 | TestDaemonPathsList_HandlerRegistered, TestDaemonPathsList_EmptySource |
| AC-002 | TestPathEntry_RTTValueSerialization (4 rows) |
| AC-003 | TestPathEntry_StatusFromDegraded (3 rows) |
| AC-004 | TestDaemonRouterMetrics_HandlerRegistered, TestDaemonRouterMetrics_SVTNNotFound |
| AC-005 | TestDaemonRouterStatus_HandlerRegistered |
| AC-005a | TestDaemonRouterStatus_FailedAndPendingPrecedence (4 rows), TestQualityFromEntry_PendingWinsOverFailed, TestQualityFromEntry_NeverEmitsFailed |
| AC-006 / VP-047 | TestVP047_SbctlPathsList_EndToEnd |

## Notes

- Lint: 5 pre-existing lint errors from stub production code remain; 0 new lint errors introduced by test code.
- AC-006 oracle: enters daemon through production mgmt.Server + RegisterMetricsHandlers path (not test-local handler construction). sbctl-binary oracle deferred to Wave-6 per story v1.4 rationale (binary not available until S-W5.04 handlers compiled into cmd/switchboard).
- Stub production code intentionally NOT modified (vsdd-factory #374 compliance).
