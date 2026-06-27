# Red Gate Log — S-W3.04 Daemon Assembly

**Story:** S-W3.04 — Full Daemon Assembly  
**Date:** 2026-06-27  
**Phase:** TDD (test-writer pass)  
**BC-5.38.001 Status:** RED GATE VERIFIED

## Summary

7 new tests written across 2 files. All 7 fail against the current stubs
(panic: not implemented). All 34 existing tests still pass.

## Test Files

| File | New Tests | Status |
|------|-----------|--------|
| `internal/tmux/connector_frames_test.go` | 1 | FAILING (Red Gate) |
| `cmd/switchboard/main_test.go` | 6 | FAILING (Red Gate) |

## Per-Test Red Gate Results

| Test Name | AC/BC Trace | Failure Mode |
|-----------|-------------|--------------|
| `TestSessionConnectorFramesSurviveFailover` | AC-004 / BC-2.04.001 PC-5 + BC-2.04.002 PC-4 | `panic: not implemented: SessionConnector.Frames() relay goroutine (S-W3.04 AC-004)` |
| `TestRouterLoggerEmitsEADM016` | AC-001 / BC-2.05.008 PC-2 | `panic: not implemented: buildRouter (S-W3.04 AC-001)` |
| `TestDaemonAuthRejectsUnregisteredConsole` | AC-002 / BC-2.04.005 PC-3 + BC-2.04.003 PC-3 | `panic: not implemented: buildAccessNode (S-W3.04 AC-002)` |
| `TestDaemonSweepEvictsStaleConsole` | AC-003 / BC-2.04.004 PC-1 + PC-3 | `panic: not implemented: startSweepTicker goroutine (S-W3.04 AC-003)` |
| `TestDaemonFramesDroppedLoggedOnTick` | AC-006 / BC-2.04.006 invariant 4 | `panic: not implemented: startFramesDroppedTicker goroutine (S-W3.04 AC-006)` |
| `TestDaemonConnectFailureExitsNonZero` | AC-007 / BC-2.04.007 PC-1 | `panic: not implemented: runAccess daemon wiring (S-W3.04 AC-001–AC-008)` |
| `TestDaemonCleanShutdown` | AC-008 / BC-2.04.007 PC-2 | `panic: not implemented: runAccess daemon wiring (S-W3.04 AC-001–AC-008)` |

## Existing Test Baseline (still passing)

```
ok  github.com/arcavenae/switchboard/cmd/switchboard    (3 tests)
ok  github.com/arcavenae/switchboard/internal/admission (cached)
ok  github.com/arcavenae/switchboard/internal/frame     (cached)
ok  github.com/arcavenae/switchboard/internal/halfchannel (cached)
ok  github.com/arcavenae/switchboard/internal/hmac      (cached)
ok  github.com/arcavenae/switchboard/internal/routing   (cached)
ok  github.com/arcavenae/switchboard/internal/session   (cached)
ok  github.com/arcavenae/switchboard/internal/tmux      (31 tests pass; new test excluded)
```

## Red Gate Verification Command

```bash
# All new tests fail:
go test ./internal/tmux/ -run TestSessionConnectorFramesSurviveFailover -count=1
go test ./cmd/switchboard/ -run "TestRouterLoggerEmitsEADM016|TestDaemonAuthRejectsUnregisteredConsole|TestDaemonSweepEvictsStaleConsole|TestDaemonFramesDroppedLoggedOnTick|TestDaemonConnectFailureExitsNonZero|TestDaemonCleanShutdown" -count=1

# Existing tests pass (excluding new failing tests):
go test ./cmd/switchboard/ -run "^TestRun$|^TestVersionNonEmpty$|^TestRun_WriteError$" -count=1
go test ./internal/tmux/ -run "TestSessionConnector_|TestPTYProxy_|TestTmuxControlMode_|TestControlMode_|TestClassifyStderr" -count=1
```

## AC-005 Note

AC-005 (Frames→DeliverFrame bridge) has no standalone test per the story spec.
It is covered structurally by AC-004 (TestSessionConnectorFramesSurviveFailover,
which exercises the relay goroutine) and AC-008 (TestDaemonCleanShutdown, which
verifies the bridge goroutine exits cleanly on sc.Close()).

## Implementer Handoff

Make each test pass, one at a time, with minimum code:

1. `internal/tmux/connector_frames.go`: implement `Frames()`, `activeFrSource()`,
   `forwardFrames()` per ADR-011 §Concurrency contract.
2. `cmd/switchboard/access.go`: implement `buildRouter`, `buildAccessNode`,
   `startSweepTicker`, `startFramesDroppedTicker`, `runAccess`.
