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

---

## Wave-3 Finding I-1 — Goroutine Join Red Gate (2026-06-27)

**Story:** S-W3.04 — Finding I-1 (ticker wg-tracking)
**Branch:** `fix/W3-i1-ticker-wg-join`
**Commit:** `74a6de2`
**BC-5.38.001 Status:** RED GATE VERIFIED (build failure)

### Test

`cmd/switchboard/access_goroutine_join_test.go`
`TestRunAccessWithConnectorNoGoroutineLeak` (AC-008 / BC-2.04.007 PC-2 postcon-6)

### Red Gate Result

```
# github.com/arcavenae/switchboard/cmd/switchboard [github.com/arcavenae/switchboard/cmd/switchboard.test]
cmd/switchboard/access_goroutine_join_test.go:105:2: cannot assign to framesDroppedInterval (neither addressable nor a map index expression)
cmd/switchboard/access_goroutine_join_test.go:106:21: cannot assign to framesDroppedInterval (neither addressable nor a map index expression)
FAIL    github.com/arcavenae/switchboard/cmd/switchboard [build failed]
```

The test assigns `framesDroppedInterval = time.Millisecond` to inject a fast tick
interval. `framesDroppedInterval` is currently a `const` in `access.go`, which is
not assignable. The build failure is the Red Gate.

### Why This Is The Correct Red Gate

The compile error enforces the full scope of the required fix: the implementer must
change `framesDroppedInterval` from `const` to `var` (testability) AND add `wg.Add(1)`
before both `startSweepTicker` and `startFramesDroppedTicker` (correctness). Neither
change alone is sufficient — both are required for the test to build AND pass.

### Discriminating Mechanism (channel handshake)

Once `framesDroppedInterval` is a `var`:
- `blockingRelayConnector.RelayDropped()` parks the ticker goroutine on first call
- Test cancels ctx, then selects: if `done` closes within 150ms → goroutine not joined → RED
- On fixed code: `wg.Wait()` blocks until goroutine is released → `done` stays open → PASS

### Required Fix

1. In `access.go`: change `const framesDroppedInterval = 30 * time.Second` to
   `var framesDroppedInterval = 30 * time.Second`
2. In `runAccessWithConnector`: add `wg.Add(1)` + `defer wg.Done()` inside both
   `startSweepTicker` and `startFramesDroppedTicker` goroutines

---

## Implementer Handoff

Make each test pass, one at a time, with minimum code:

1. `internal/tmux/connector_frames.go`: implement `Frames()`, `activeFrSource()`,
   `forwardFrames()` per ADR-011 §Concurrency contract.
2. `cmd/switchboard/access.go`: implement `buildRouter`, `buildAccessNode`,
   `startSweepTicker`, `startFramesDroppedTicker`, `runAccess`.

---

# Red Gate Log — S-4.04 Split-Horizon Loop Prevention + DropCache Router Wiring

**Story:** S-4.04 — Split-Horizon Loop Prevention in internal/routing  
**Date:** 2026-06-28  
**Phase:** TDD (test-writer pass)  
**BC-5.38.001 Status:** RED GATE VERIFIED

## Summary

12 tests + 1 fuzz target written across 2 files. All 12 unit tests fail against
current stubs (`panic: not implemented`). Fuzz target compiles. Lint passes (0
issues). Build passes.

## Test Files

| File | Tests | Status |
|------|-------|--------|
| `internal/routing/split_horizon_test.go` | 6 unit + 1 fuzz + 1 property | FAILING (Red Gate) |
| `internal/routing/on_frame_arrival_test.go` | 7 unit (incl. subtests) | FAILING (Red Gate) |

## Per-Test Red Gate Results

| Test Name | AC/BC Trace | Failure Mode |
|-----------|-------------|--------------|
| `TestSplitHorizon_NoForwardTowardArrivalInterface` | AC-001 / BC-2.02.008 PC-1 | `panic: not implemented` (SplitHorizon.Forward stub) |
| `TestSplitHorizon_ForwardOnAllOtherInterfaces` | AC-002 / BC-2.02.008 PC-2 | `panic: not implemented` (SplitHorizon.Forward stub) |
| `TestSplitHorizon_OnlyArrivalInterfaceInSet` | EC-001 / BC-2.02.008 PC-3 | `panic: not implemented` (SplitHorizon.Forward stub) |
| `TestSplitHorizon_EmptyInterfaceSet` | BC-2.02.008 PC-3 (empty set) | `panic: not implemented` (SplitHorizon.Forward stub) |
| `TestSplitHorizon_UnknownArrivalInterfaceID` | EC-002 / BC-2.02.008 | `panic: not implemented` (SplitHorizon.Forward stub) |
| `TestSplitHorizon_VP011_ArrivalIfaceNeverForwarded` | VP-011 (1024-case property) | `panic: not implemented` (SplitHorizon.Forward stub) |
| `FuzzSplitHorizon_ChannelHeaderOpaque` | AC-003 / BC-2.02.008 invariant 1 / VP-015 | compiles; fuzz run would panic (Forward stub) |
| `TestBC_2_02_009_Router_DropCacheWiring` | AC-004 / BC-2.02.009 PC-1 | `panic: not implemented` (OnFrameArrival stub) |
| `TestBC_2_02_009_Router_DropCacheHitCounterIncremented` | EC-003 / BC-2.02.009 PC-2 | `panic: not implemented` (OnFrameArrival stub) |
| `TestBC_2_02_009_Router_CollisionEventLogged` | AC-005 / BC-2.02.009 PC-2 / EC-005 | `panic: not implemented` (WithFrameArrivalLogger stub) |
| `TestBC_2_02_009_Router_HashCollisionLogged` | EC-004 / BC-2.02.009 EC-005 | `panic: not implemented` (WithFrameArrivalLogger stub) |

## Red Gate Verification Command

```bash
go test ./internal/routing/... -run "TestSplitHorizon|TestBC_2_02_009" 2>&1
# Expected: FAIL — all tests panic with "not implemented" from stubs
```

## Signature Mismatch Notes

None. Stubs compiled cleanly before tests were written. No production file
modifications were needed.

### nolint:staticcheck Justification (SA4006)

Three `//nolint:staticcheck` directives appear in `on_frame_arrival_test.go`.
Each marks a `h := NewFrameArrivalHandler(dc)` assignment that is immediately
followed by `WithFrameArrivalLogger(...)(h)`. Because `WithFrameArrivalLogger`
panics in the stub, staticcheck's SSA considers the code after the call
unreachable and flags `h` as "never used". The nolint is justified: `h` IS used
after the stub is replaced by the implementer. The directives must be removed
after the stubs are implemented.

## Implementer Handoff (S-4.04)

Make each test pass, one at a time, with minimum code:

1. `SplitHorizon.Forward`: filter arrivalIface from interfaceSet; call fn for
   each remaining interface; return ErrAllPathsSplitHorizon if no interfaces
   remain. Never parse frameBytes.
2. `WithFrameArrivalLogger(l Logger) func(*FrameArrivalHandler)`: return a
   function that sets `h.logger = l`.
3. `FrameArrivalHandler.OnFrameArrival`: compute CRC32 of frameBytes, call
   `h.dropCache.AddIfAbsent(checksum, uint64(arrivalIface))`. On hit: log via
   `h.logger.Log(...)`, return `ErrDropCacheHit`. On miss: return nil.
4. Remove `//nolint:staticcheck` directives once stubs are implemented.

---

# Red Gate Log — S-6.03 sbctl Client Auth v2.2

**Story:** S-6.03 — sbctl Client Auth (Authenticate() fail-closed, flag parsing, JSON envelope, connection error)
**Story version:** 2.2
**Date:** 2026-06-29
**Phase:** TDD (test-writer pass — v2.2 AC update)
**BC-5.38.001 Status:** RED GATE VERIFIED (compile failure)

## Summary

Updated `cmd/sbctl/client_test.go` and `cmd/sbctl/main_test.go` for v2.2 ACs.
The entire `cmd/sbctl` test package fails to build against current production code.
Failure is a compile error, not a runtime panic — which is the correct Red Gate
when the required API signature changes are the gating constraint.

## Test Files Modified

| File | Tests Added/Changed | Status |
|------|---------------------|--------|
| `cmd/sbctl/client_test.go` | Updated all `Authenticate(...)` calls to ctx-first signature; added `TestAuthenticate_FailClosed_VP067` sub-case (i) (deadline expiry); added `TestSbctl_KeyLoadFailure_ExitsOneWithECFG010`; added `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001`; added `TestSbctl_TildeExpansion_DefaultKey` (4 sub-cases); added `TestSbctl_ConnectAndRun_ReturnsError` | FAILING (compile — Red Gate) |
| `cmd/sbctl/main_test.go` | Added `KeyLoadFailure` and `RPCDispatchFailure` subprocess cases to `subprocessEntrypoint` | Compiles only when client_test.go compiles (same package) — FAILING |

## Per-Test Red Gate Results

| Test Name | AC/BC Trace | Failure Mode |
|-----------|-------------|--------------|
| `TestAuthenticate_FailClosed_VP067` (all sub-cases including i) | AC-002 / BC-2.07.002 PC-2 / VP-067 | Compile error: `too many arguments in call to Authenticate` — `context.Context` first param missing from current signature |
| `TestAuthenticate_PrivKeyNeverTransmitted` | BC-2.07.002 Inv-2 / DI-002 | Same compile error |
| `TestSbctl_KeyLoadFailure_ExitsOneWithECFG010` | AC-003 / BC-2.07.003 EC-005 | Compile error (same package); behaviorally: current `connectAndRun` emits `E-NET-001` not `E-CFG-010` |
| `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001` | AC-004 / BC-2.07.003 EC-006 | Compile error (same package); behaviorally: current `dispatch()` returns "not implemented" and `connectAndRun` maps to `E-NET-001` not `E-RPC-001` |
| `TestSbctl_TildeExpansion_DefaultKey` (4 sub-cases) | AC-008 / BC-2.07.003 EC-007 + PC-3 | Compile error: `undefined: homeDirFunc` — injectable var not yet declared in `client.go` |
| `TestSbctl_ConnectAndRun_ReturnsError` | AC-009 / go.md "no os.Exit outside main()" | Compile error: `connectAndRun(...) (no value) used as value` — current signature returns nothing; also wrong arg types (ctx-first + no timeout param) |

## Red Gate Verification Command

```bash
go test -gcflags=-e ./cmd/sbctl/... 2>&1
# Expected: build failed — 30+ compile errors covering all 3 missing API changes
```

## Full Error Count (go test -gcflags=-e)

- 12 × `too many arguments in call to Authenticate` (ctx-first signature missing)
- 12 × `undefined: homeDirFunc` (injectable home-dir lookup var missing)
- 3 × `connectAndRun(...)` errors (no return value + wrong arg types for ctx-first refactor)

All 27 errors resolve once the implementer introduces the 3 new production symbols listed below.

## Required New Production Symbols (implementer must introduce)

1. **`Authenticate` signature change** in `cmd/sbctl/client.go`:
   ```go
   // BEFORE (current — must change):
   func Authenticate(conn net.Conn, privKey ed25519.PrivateKey) error

   // AFTER (required):
   func Authenticate(ctx context.Context, conn net.Conn, privKey ed25519.PrivateKey) error
   ```
   Implementation must: compute effective deadline from `ctx.Deadline()` (fallback
   `time.Now().Add(mgmt.HandshakeTimeout)` — or a local const `10*time.Second` since
   `cmd/sbctl` must NOT import `internal/mgmt`); call `conn.SetReadDeadline(deadline)`
   before the first `json.NewDecoder` read.

2. **`var homeDirFunc = os.UserHomeDir`** in `cmd/sbctl/client.go`:
   ```go
   // Package-level injectable for testing (AC-008 tilde expansion).
   var homeDirFunc = os.UserHomeDir
   ```
   `loadEd25519Key` must use `homeDirFunc()` instead of `os.UserHomeDir()` when
   expanding `~/` prefixes. Sub-case (a): if `homeDirFunc()` errors → return
   `fmt.Errorf("key load failed: %s: %w", originalPath, err)`. Sub-case (b): if
   expansion succeeds but file open fails → return
   `fmt.Errorf("key load failed: %s: %w", expandedPath, err)`.

3. **`connectAndRun` signature change** in `cmd/sbctl/main.go`:
   ```go
   // BEFORE (current — must change):
   func connectAndRun(target, keyPath string, useJSON bool, timeout time.Duration, command string, cmdArgs any)

   // AFTER (required):
   func connectAndRun(ctx context.Context, target, keyPath string, useJSON bool, command string, cmdArgs any) error
   ```
   Must return errors instead of calling `os.Exit`. Only `main()` calls `os.Exit`.
   The `timeout` parameter moves out of `connectAndRun` — callers wrap with
   `context.WithTimeout` before calling (as shown in AC-002 call-site example).
   Also: key-load failure must emit `E-CFG-010` (not `E-NET-001`); post-auth
   dispatch failure must emit `E-RPC-001` (not `E-NET-001`).

## Notes for Implementer

- `cmd/sbctl` MUST NOT import `internal/mgmt` (ARCH-12 §Package DAG Constraints).
  Use a local const `handshakeTimeout = 10 * time.Second` in `client.go` instead
  of referencing `mgmt.HandshakeTimeout`.
- `subprocessEntrypoint` in `main_test.go` still calls the old `connectAndRun`
  signature (no ctx, void). Once the implementer updates the production signature,
  they must also update `subprocessEntrypoint` to call `connectAndRun(ctx, ...)`,
  check the returned error, and call `os.Exit(1)` based on it. The entrypoint is the
  one place in test code permitted to call `os.Exit` (it runs in a subprocess).
- The `~` expansion in `loadEd25519Key` must treat `~/` (tilde-slash only, not bare
  `~`) as requiring expansion. A bare `~` without a following `/` is ambiguous; per
  BC-2.07.003 EC-007 the default path is `~/.ssh/id_ed25519` so `~/` is the
  expansion trigger. `~username/...` is treated as a literal.

## Implementer Handoff (S-6.03)

Make each test pass, one at a time, with minimum code:

1. Add `var homeDirFunc = os.UserHomeDir` to `client.go`; update `loadEd25519Key`
   to call `homeDirFunc()` for `~/` prefix expansion; return `E-CFG-010`-format
   errors for all key-load failure modes.
2. Change `Authenticate` signature to ctx-first; add `conn.SetReadDeadline` from
   ctx deadline (fallback `10s`).
3. Change `connectAndRun` to return `error`; update `main()` to call `os.Exit` on
   returned error; map key-load failure → `E-CFG-010`; map dispatch failure →
   `E-RPC-001`.
4. Implement `dispatch()` to send the RPC envelope and parse the response; return
   `E-RPC-001`-format error on `ok:false`, decode failure, or connection drop.
5. Update `subprocessEntrypoint` in `main_test.go` to use the new `connectAndRun`
   signature (ctx-first, error-return) — this is a test-file change, not production.
