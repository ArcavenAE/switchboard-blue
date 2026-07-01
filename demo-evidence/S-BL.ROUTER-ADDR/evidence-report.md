# Demo Evidence Report — S-BL.ROUTER-ADDR

**Story:** S-BL.ROUTER-ADDR v1.4 — populate PathSnapshot.RouterAddr with real resolved host:port (BC-2.06.003 PC-1)
**HEAD:** dffc27e
**Status:** CONVERGED (Pass-8/9/10 all clean 3-lens under BC-5.39.001)
**Recorded:** 2026-07-01

## Coverage Matrix

| AC | Title | Test Function(s) | Recording | Pass/Fail |
|----|-------|-----------------|-----------|-----------|
| AC-001 | PathSnapshot.RouterAddr field propagation | TestBC_2_06_003_Snapshot_RouterAddr_Propagates, TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess | AC-001-router-addr-snapshot.{gif,webm} | PASS |
| AC-002 | PathsList handler passes snap.RouterAddr | TestPathsList_PassesRouterAddr | AC-002-paths-list-passes-router-addr.{gif,webm} | PASS |
| AC-003 | NewPathTrackerWithAddr constructor + NewPathTracker backward compat | TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr, TestBC_2_06_003_NewPathTracker_Unchanged, TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction | AC-003-new-path-tracker-with-addr.{gif,webm} | PASS |
| AC-004 | BC-2.06.003 v1.15 annotation cleanup — DRIFT-SW504 removed | Spec obligation (not a Go test): grep confirms 0 DRIFT-SW504 occurrences and v1.15 + NewPathTrackerWithAddr references present | AC-004-bc-annotation-cleanup.{gif,webm} | PASS |
| AC-005 | DRIFT-SW504 closed; VP-047 oracle flip to non-empty router_addr | TestVP047_RouterAddrNonEmpty (Parts A + B: handler_seam_non_empty, constructor_through_snapshot) | AC-005-drift-closure-oracle-flip.{gif,webm} | PASS |

## Race Test

`go test -race ./internal/paths/... ./internal/metrics/...` — all packages clean.

Full output: `race-test-transcript.txt`

Key results:
- `ok  github.com/arcavenae/switchboard/internal/paths` (race-clean)
- `ok  github.com/arcavenae/switchboard/internal/metrics` (race-clean)

## Files

```
AC-001-router-addr-snapshot.tape
AC-001-router-addr-snapshot.gif
AC-001-router-addr-snapshot.webm
AC-002-paths-list-passes-router-addr.tape
AC-002-paths-list-passes-router-addr.gif
AC-002-paths-list-passes-router-addr.webm
AC-003-new-path-tracker-with-addr.tape
AC-003-new-path-tracker-with-addr.gif
AC-003-new-path-tracker-with-addr.webm
AC-004-bc-annotation-cleanup.tape
AC-004-bc-annotation-cleanup.gif
AC-004-bc-annotation-cleanup.webm
AC-005-drift-closure-oracle-flip.tape
AC-005-drift-closure-oracle-flip.gif
AC-005-drift-closure-oracle-flip.webm
race-test-transcript.txt
```

## Notes

- AC-004 is a spec-level obligation (DRIFT annotation removal), not a Go test. The recording
  demonstrates the live spec file confirming zero DRIFT-SW504 occurrences and the v1.15 +
  NewPathTrackerWithAddr entries are present.
- AC-001 error path (NewPathTracker yields RouterAddr=="") is covered by
  TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess, exercised in the AC-001 recording.
- AC-003 immutability (RouterAddr set once, never mutated) is covered by
  TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction, exercised in the AC-003 recording.
- VP-047 Part A (handler_seam_non_empty) and Part B (constructor_through_snapshot) are both
  demonstrated in the AC-005 recording via TestVP047_RouterAddrNonEmpty subtests.
