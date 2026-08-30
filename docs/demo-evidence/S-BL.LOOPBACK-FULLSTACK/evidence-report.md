# Evidence Report — S-BL.LOOPBACK-FULLSTACK: Full-Stack Loopback Testenv Extension for VP-042

**Story:** S-BL.LOOPBACK-FULLSTACK v1.21 (spec-converged AND human-approved at v1.21/`7967a2f`, 2026-08-30)
**Branch:** `feature/S-BL.LOOPBACK-FULLSTACK`
**Frozen implementation HEAD:** `235bb5a` — Phase-3 (TDD Implementation) Step-4.5 adversarial convergence
CONVERGED 3/3 NITPICK_ONLY (BC-5.39.001) at this exact SHA (three consecutive passes A/B/C,
zero product defects). `.factory/STATE.md` names this recording (Step 5) as the next step after that
convergence.
**Recorded:** 2026-08-30
**Toolchain:** VHS 0.11.0 (terminal recordings of `go test` runs — see "Story nature" below)
**Go:** go1.25.4 darwin/arm64

---

## Story nature — why this "demo" is `go test`, not a CLI/UI walkthrough

`S-BL.LOOPBACK-FULLSTACK` is a **test-harness / library story**. It extends `internal/testenv` with a
tick-driven, full-stack loopback driver (`loopbackDriver`) that powers the VP-042 keystroke-to-echo
latency harness (`internal/halfchannel` + `internal/arq` + `internal/multipath` + `internal/paths` wired
together). There is **no interactive CLI surface and no browser/UI** delivered by this story — every
acceptance criterion is a property of `internal/testenv`'s Go API, asserted by a Go test.

The honest demo evidence for each AC is therefore: **run that AC's test, on camera, and show it PASS.**
Each `.tape` in this directory does exactly that — one VHS recording per AC (or per closely-related pair
of tests belonging to the same AC), running the real `go test -race -run <pattern>` command against
`internal/testenv` and waiting for the terminal to show `PASS`. Nothing here fabricates a CLI walkthrough
this story does not implement.

AC-013 (the merged VP-042 benchmark cross-reference) has no new unit test of its own — see its row below.

---

## AC-001 — discharged, no demo required

AC-001 (the `arq.OnAck` call-contract sign-off gate — single shared `*arq.ARQ` instance for the
downstream direction) was **DISCHARGED at the spec level on 2026-07-12** (verdict REVISED), before this
story left draft/unscheduled status. It is a pre-implementation design gate, not a runtime behavior with
its own test — the topology it mandates is exercised indirectly by every downstream-ARQ test below
(AC-006, AC-014, AC-016), and directly regression-guarded by AC-014's dedicated round-trip-completes test.
Per the task scope for this recording pass, **AC-001 carries no demo file.**

---

## Coverage Summary — AC-002 .. AC-017 (16 ACs)

| AC | Description | Demonstrating Test(s) | Tape File | Result |
|----|-------------|------------------------|-----------|--------|
| AC-002 | Tick-interval validation (a) + independent per-direction tick schedule (b); BC-2.01.001, BC-2.01.003 harness-scope | `TestNewLoopback_RejectsOutOfBoundsTickInterval`, `TestLoopbackDriver_TicksFireOnSchedule` | `AC-002-tick-interval-and-schedule.tape` | **PASS** |
| AC-003 | Empty ticks fire on schedule but are never wire-dispatched; BC-2.01.002 | `TestLoopbackDriver_EmptyTicksNotDispatched` | `AC-003-empty-ticks-not-dispatched.tape` | **PASS** |
| AC-004 | Duplicate-and-race dispatch over both synthetic paths; BC-2.02.001, BC-2.02.003 harness-scope | `TestLoopbackDriver_DuplicateAndRaceDispatch` | `AC-004-duplicate-and-race-dispatch.tape` | **PASS** |
| AC-005 | Endpoint checksum dedup discards the second-arriving duplicate (upstream + downstream); BC-2.02.002 | `TestLoopbackDriver_EndpointDedupDiscardsSecondArrival` | `AC-005-endpoint-dedup-discards-second-arrival.tape` | **PASS** |
| AC-006 | Downstream ARQ `EnqueueSend`+`OnAck` wired once per data tick, same shared instance; BC-2.02.005 | `TestLoopbackDriver_DownstreamARQWiring` | `AC-006-downstream-arq-wiring.tape` | **PASS** |
| AC-007 | Dedicated shard — driver never mutates `env.defaultShard`; no `SetSink` escape hatch | `TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation`, `TestSessionAccessNode_NoSetSinkMethod` | `AC-007-dedicated-shard-no-setsink.tape` | **PASS** |
| AC-008 | `RoundTrip` token API isolates concurrent/stale round trips from `WaitForEcho` | `TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips`, `TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer` | `AC-008-roundtrip-token-isolation.tape` | **PASS** |
| AC-009 | `RoundTrip.done` buffered-1 send never blocks/leaks on a `WaitForEcho` timeout | `TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak` | `AC-009-waitforecho-timeout-no-leak.tape` | **PASS** |
| AC-010 | Synthetic path trackers report `IsActive()==true` with zero probes | `TestNewLoopbackPaths_TrackersActiveWithoutProbe` | `AC-010-pathtracker-active-without-probe.tape` | **PASS** |
| AC-011 | Non-blocking teardown diagnostic on undrained `driver.pending` entries | `TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak`, `TestLoopbackEnv_Cleanup_SilentWhenDrained` | `AC-011-cleanup-diagnostic-on-pending-leak.tape` | **PASS** |
| AC-012 | Both ticker goroutines join deterministically on `Env.Close()` | `TestLoopbackEnv_TickerGoroutines_JoinOnClose` | `AC-012-ticker-goroutines-join-on-close.tape` | **PASS** |
| AC-013 | Merged bench file updated to the token-based `RoundTrip` API (no new unit test — see below) | *(none — bench-file modification)* | `AC-013-vp042-bench.tape` | **PASS** (see VP-042 section) |
| AC-014 | Full round trip completes on the single shared `*arq.ARQ` instance (regression guard vs. AC-001 Addendum) | `TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance` | `AC-014-roundtrip-completes-single-arq-instance.tape` | **PASS** |
| AC-015 | Loopback driver runs clean under `go test -race` (per-direction HalfChannel mutexes, H2) | `TestLoopbackDriver_NoRaceUnderConcurrentSendEcho` | `AC-015-no-race-under-concurrent-send-echo.tape` | **PASS** |
| AC-016 | Downstream `OnAck` error surfaces loud (`driver.failLoud`), never a silent timeout | `TestLoopbackDriver_OnAckError_SurfacesLoud` | `AC-016-onack-error-surfaces-loud.tape` | **PASS** |
| AC-017 | Upstream `SendKeystroke` delivery error surfaces loud, in-place inside `deliverUpstream` | `TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud` | `AC-017-upstream-delivery-error-surfaces-loud.tape` | **PASS** |

**All 16 ACs (AC-002 .. AC-017): PASS.** AC-001 is discharged (see above), zero DEMO-ISSUE flags.

Every `-run` pattern above was executed for real, with `-race`, against `./internal/testenv/`, immediately
before this report was written (see "Captured Test Output" below for the genuine transcripts).

---

## Per-AC Detail

### AC-002 — tick-interval validation + independent per-direction schedule

**(a)** `NewLoopback` validates `cfg.TickIntervalUpstream`/`TickIntervalDownstream` against
`halfchannel.MinTickInterval`/`MaxTickInterval` and `b.Fatalf`s on an out-of-bounds value (table-driven:
below min, above max, exactly at max = legal).
**(b)** Both ticker goroutines fire `HalfChannel.Tick()` on their own configured schedule independent of
`Enqueue` timing — a keystroke enqueued between ticks waits for the next tick, never an out-of-band
delivery. This is also the BC-2.01.003 harness-scope clock-independence aspect (sequence-space
independence holds by construction, not by test assertion — see the story's Anchors Consumed table).

### AC-003 — empty ticks fire, but are never wire-dispatched

The upstream/downstream tickers call `Tick()` every interval regardless of whether data is enqueued
(BC-2.01.002 — empty ticks ARE produced), but an empty-tick frame (`FrameType != frame.FrameTypeData`) is
never passed to `multipath.Send`. Harness-scope boundary (Non-Goals), not a production behavior change.

### AC-004 — duplicate-and-race dispatch over two synthetic paths

`upstreamMP`/`downstreamMP` each dispatch a single ticked payload to both synthetic `paths.RankedPath`s
(BC-2.02.001); `deliverUpstream`/`deliverDownstream` fires exactly twice per data tick. Incidentally
exercises the BC-2.02.003 harness-scope path-ranking aspect (`multipath.Send` calls `paths.Rank()` on
every dispatch) — not itself asserted.

### AC-005 — endpoint dedup discards the second-arriving copy

Upstream: `accessNode.SendKeystroke` runs exactly once per ticked data frame (gated directly by the
`upstreamMP.Receive` dedup inside `deliverUpstream`). Downstream: the dedup gates `deliverDownstream`
itself (one `nil` + one `ErrDuplicate` per tick), but does **not** gate `downstreamARQ.OnAck` — `OnAck`'s
cadence is tick-driven, not dedup-driven (see AC-006).

### AC-006 — downstream ARQ wiring, single shared instance

Every downstream data tick: `EnqueueSend` before dispatch, then — after `Send`/`deliverDownstream` return
— the SAME tick body calls the SAME `*arq.ARQ` instance's `OnAck` exactly once, per the AC-001-discharged
call contract. `GapsToRetransmit`/`TLPKTDROP` never fire in this no-loss harness.

### AC-007 — dedicated shard, no production `SetSink` escape hatch

`loopbackDriver` builds its own `Publisher`/`SessionAuth`/`AccessNode` triple with
`WithKeystrokeSink(loopbackSink)` from construction; `env.defaultShard` is never touched.
`TestSessionAccessNode_NoSetSinkMethod` is a reflection guard confirming production `session.AccessNode`
gained no sink-mutation method.

### AC-008 — `RoundTrip` token API isolation

`WaitForEcho` consumes exactly one token's own completion channel — a concurrent round trip's frame
cannot satisfy a different token's wait, and a pre-populated `Env.CollectFrames` buffer is never read.

### AC-009 — buffered `done` channel: no leak, no block, on timeout

`RoundTrip.done` is buffered 1. The downstream ticker's completion path unconditionally deletes the
`driver.pending` entry and sends into `done` whether or not `WaitForEcho` is still listening — a timed-out
waiter never causes the ticker goroutine to block, and no goroutine leak results.

### AC-010 — path trackers active without any probe

`paths.NewPathTracker(1.0, 0.125).IsActive()` is `true` immediately at construction — insurance against a
future `internal/paths` change silently breaking loopback path activation.

### AC-011 — non-blocking pending-map diagnostic

`LoopbackEnv` registers a `t.Cleanup` diagnostic that reports (via `Logf`, never `t.Fatalf`) any entry
still in `driver.pending` at teardown. Verified deterministically via an injected decode-mismatch
scenario (fires) and a normal drained round trip (silent).

### AC-012 — ticker goroutines join deterministically on `Close()`

Both ticker goroutines register on the existing `Env.wg`/`Env.closeCh` (no new WaitGroup); `Env.Close()`
blocks until both have exited, with a bounded timeout guarding against a hang.

### AC-013 — merged bench file updated to the token-based API (no new unit test)

`internal/bench/keystroke_echo_testenv_bench_test.go` (merged on `develop`, PR #121 @ `4c276d9`,
`//go:build integration`-tagged) is on the token-based shape: `rt := lb.SendKeystroke(b, sessionID, "x");
payload, ok := lb.WaitForEcho(b, rt, 500*time.Millisecond)`. There is no new unit test for this AC — per
the story's binding Verification Method (v1.13 F-R15-LENSB-01: `go build` never compiles `_test.go`
files, tag-independent, so it verifies nothing here), the AC verifies via:

1. **Compile-gate that actually reaches the tagged test binary:**
   `go test -tags integration -run '^$' -count=1 ./internal/bench/`
2. **The VP-042 benchmark run itself** (harness-delivered evidence, not a `verification_lock` flip):
   `go test -tags integration -run '^$' -bench 'KeystrokeToEcho' -benchtime 1x -count=1 ./internal/bench/`

Both ran cleanly this session — see the "VP-042 Benchmark" section below.

### AC-014 — round trip completes on the single shared ARQ instance (regression guard)

Drives a complete `SendKeystroke` → `WaitForEcho` round trip through a real `LoopbackEnv` and asserts (a)
`ok == true` (no timeout) and (b) `decodeRTID(payload)` succeeds and matches the sent round-trip id — the
non-empty, correctly-correlated delivery is the load-bearing assertion, guarding against a silent
`arqServer`/`arqClient` two-instance regression (which would make every `OnAck` call return `(nil, nil)`
forever).

### AC-015 — race-clean under concurrent send/echo

Run under `go test -race` with 6 writer goroutines (alternating 250ms/50ms-equivalent load) and 6 reader
goroutines hitting `SendKeystroke`/`WaitForEcho` concurrently. Zero `DATA RACE` annotations — the
per-direction `upstreamHCMu`/`downstreamHCMu` mutexes (H2) hold.

### AC-016 — downstream `OnAck` error surfaces loud

Fault-injection via the `onDownstreamTick()` package-private seam drives `downstreamARQ.OnAck` into
`ErrAckOutOfWindow` (65 empty ticks + 1 data tick, single-goroutine, no ticker started). `driver.failLoud`
(`t.Errorf`, captured by a `recordingTB` stub wrapping the real `*testing.T`) fires exactly once — never a
silent `WaitForEcho` timeout.

### AC-017 — upstream delivery error surfaces loud, in-place

Symmetric with AC-016: `deliverUpstream` checks `accessNode.SendKeystroke`'s error **in-place**, at the
call site (not via `upstreamMP.Send`'s masked tick-body return value, which can never observe a failing
delivering path because the dedup sibling always returns `nil` first). `driver.failLoud` fires exactly
once per failing tick.

---

## VP-042 Benchmark — `AC-013-vp042-bench.tape`

**Attempted this session — ran cleanly, in-worktree, no environmental blockers.**

```
$ go test -tags integration -run '^$' -count=1 ./internal/bench/
ok  	github.com/arcavenae/switchboard/internal/bench	0.293s [no tests to run]

$ go test -tags integration -run '^$' -bench 'KeystrokeToEcho' -benchtime 1x -count=1 ./internal/bench/
goos: darwin
goarch: arm64
pkg: github.com/arcavenae/switchboard/internal/bench
cpu: Apple M1
BenchmarkKeystrokeToEcho_P99-8   	       1	25002084041 ns/op	        52.04 p99_rtt_ms
PASS
ok  	github.com/arcavenae/switchboard/internal/bench	25.271s
```

**Result: p99 = 52.04ms, under the VP-042 / NFR-001 100ms ceiling** (`b.Errorf` did not fire). This is the
harness-delivered measurement the story ships — per the story's Context and VP-042 trace annotation
(`"HARNESS DELIVERED, NOT LOCKED"`), running this benchmark once for evidence is exactly what this story
does; **the separate `verification_lock` flip is explicitly out of this story's scope** (a subsequent act,
per the Forward Obligation section of the story). No forward-obligation note is needed here — the run
completed cleanly in this worktree with real numbers, not a deferred manual step.

Note for the record: `just bench` is explicitly **out of AC-013's scope** — it runs the separate, tag-free
`BenchmarkKeystrokeEcho_P99` (no "To") lower-bound recipe from `keystroke_echo_bench_test.go` (S-BL.BENCH),
unaffected by and irrelevant to this story.

---

## Captured Test Output (real, from this session)

### Full suite — `go test -v -race ./internal/testenv/`

Ran once, in full, before per-AC breakdown. Exit code 0. Representative PASS lines (all 19
loopback-related test functions plus the pre-existing `testenv`/`multicast_loopback` suite, zero
regressions):

```
--- PASS: TestLoopbackDriver_NoRaceUnderConcurrentSendEcho (0.06s)
--- PASS: TestMulticastLoopbackInterface_ResolvesLoopback (0.00s)
--- PASS: TestNew_EnvIsUsable (0.00s)
--- PASS: TestConnectWithKey_RegisteredIsAdmitted (0.01s)
--- PASS: TestConnectWithKey_RevokedIsRejected (0.01s)
--- PASS: TestStartRouter_Restart_EntersPEMode (0.01s)
--- PASS: TestCreateSVTN_Unique (0.00s)
--- PASS: TestNewLoopback_Compiles (0.00s)
--- PASS: TestLoopbackDriver_OnAckError_SurfacesLoud (0.00s)
--- PASS: TestNewLoopbackPaths_TrackersActiveWithoutProbe (0.00s)
--- PASS: TestClose_NoGoroutineLeak (0.00s)
--- PASS: TestLoopbackEnv_TickerGoroutines_JoinOnClose (0.00s)
--- PASS: TestStartRouter_ModeE (0.00s)
--- PASS: TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak (0.00s)
    --- PASS: TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak/subtest (0.00s)
--- PASS: TestLoopbackDriver_EndpointDedupDiscardsSecondArrival (0.00s)
--- PASS: TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud (0.00s)
--- PASS: TestSessionAccessNode_NoSetSinkMethod (0.00s)
--- PASS: TestRouterHandle_Restart_TwicePE (0.02s)
--- PASS: TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation (0.00s)
--- PASS: TestLoopbackDriver_DuplicateAndRaceDispatch (0.00s)
--- PASS: TestLoopbackDriver_TicksFireOnSchedule (0.04s)
--- PASS: TestLoopbackDriver_DownstreamARQWiring (0.00s)
--- PASS: TestLoopbackDriver_EmptyTicksNotDispatched (0.00s)
--- PASS: TestNewWithRouters_CloseRouterConnection (0.05s)
--- PASS: TestConnectWithSourceIP_SessionIDPreserved (0.06s)
--- PASS: TestDetach_SessionSurvives (0.00s)
--- PASS: TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance (0.05s)
--- PASS: TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak (0.06s)
--- PASS: TestCreateSession_Unique (0.00s)
--- PASS: TestLoopbackEnv_Cleanup_SilentWhenDrained (0.05s)
    --- PASS: TestLoopbackEnv_Cleanup_SilentWhenDrained/subtest (0.05s)
--- PASS: TestNewLoopback_RejectsOutOfBoundsTickInterval (0.00s)
--- PASS: TestCreateSessionInSVTN_AliveAfterCreate (0.00s)
--- PASS: TestAttachConsole_ReceivesFrames (0.02s)
--- PASS: TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer (0.05s)
--- PASS: TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips (0.10s)
--- PASS: TestConnectWithKey_ExpiredIsRejected (0.21s)
--- PASS: TestAttachConsole_Detach_StopsDelivery (0.17s)
--- PASS: TestMultiConsole_FanOut (0.23s)
--- PASS: TestSVTNIsolation (0.26s)
PASS
ok  	github.com/arcavenae/switchboard/internal/testenv	1.820s
```

### Per-AC filtered runs (each executed independently with `-race`, matching each `.tape`'s command)

All 15 per-AC `go test` invocations below (AC-002 through AC-017, excluding AC-013 which has no unit
test) exited 0 in this session, immediately before this report was written:

| AC | `-run` pattern | Exit |
|----|-----------------|------|
| AC-002 | `^(TestNewLoopback_RejectsOutOfBoundsTickInterval\|TestLoopbackDriver_TicksFireOnSchedule)$` | 0 |
| AC-003 | `^TestLoopbackDriver_EmptyTicksNotDispatched$` | 0 |
| AC-004 | `^TestLoopbackDriver_DuplicateAndRaceDispatch$` | 0 |
| AC-005 | `^TestLoopbackDriver_EndpointDedupDiscardsSecondArrival$` | 0 |
| AC-006 | `^TestLoopbackDriver_DownstreamARQWiring$` | 0 |
| AC-007 | `^(TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation\|TestSessionAccessNode_NoSetSinkMethod)$` | 0 |
| AC-008 | `^(TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips\|TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer)$` | 0 |
| AC-009 | `^TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak$` | 0 |
| AC-010 | `^TestNewLoopbackPaths_TrackersActiveWithoutProbe$` | 0 |
| AC-011 | `^(TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak\|TestLoopbackEnv_Cleanup_SilentWhenDrained)$` | 0 |
| AC-012 | `^TestLoopbackEnv_TickerGoroutines_JoinOnClose$` | 0 |
| AC-014 | `^TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance$` | 0 |
| AC-015 | `^TestLoopbackDriver_NoRaceUnderConcurrentSendEcho$` | 0 |
| AC-016 | `^TestLoopbackDriver_OnAckError_SurfacesLoud$` | 0 |
| AC-017 | `^TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud$` | 0 |

Two representative full per-AC transcripts:

```
$ go test -v -race -run "^TestLoopbackDriver_DownstreamARQWiring$" ./internal/testenv/
=== RUN   TestLoopbackDriver_DownstreamARQWiring
=== PAUSE TestLoopbackDriver_DownstreamARQWiring
=== CONT  TestLoopbackDriver_DownstreamARQWiring
--- PASS: TestLoopbackDriver_DownstreamARQWiring (0.02s)
PASS
ok  	github.com/arcavenae/switchboard/internal/testenv	1.329s
```

```
$ go test -v -race -run "^TestLoopbackDriver_OnAckError_SurfacesLoud$" ./internal/testenv/
=== RUN   TestLoopbackDriver_OnAckError_SurfacesLoud
=== PAUSE TestLoopbackDriver_OnAckError_SurfacesLoud
=== CONT  TestLoopbackDriver_OnAckError_SurfacesLoud
--- PASS: TestLoopbackDriver_OnAckError_SurfacesLoud (0.02s)
PASS
ok  	github.com/arcavenae/switchboard/internal/testenv	1.343s
```

---

## Recordings Index

| Tape File | AC | Command |
|-----------|----|---------|
| AC-002-tick-interval-and-schedule.tape | AC-002 | `go test -v -race -run "^(TestNewLoopback_RejectsOutOfBoundsTickInterval\|TestLoopbackDriver_TicksFireOnSchedule)$" ./internal/testenv/` |
| AC-003-empty-ticks-not-dispatched.tape | AC-003 | `go test -v -race -run "^TestLoopbackDriver_EmptyTicksNotDispatched$" ./internal/testenv/` |
| AC-004-duplicate-and-race-dispatch.tape | AC-004 | `go test -v -race -run "^TestLoopbackDriver_DuplicateAndRaceDispatch$" ./internal/testenv/` |
| AC-005-endpoint-dedup-discards-second-arrival.tape | AC-005 | `go test -v -race -run "^TestLoopbackDriver_EndpointDedupDiscardsSecondArrival$" ./internal/testenv/` |
| AC-006-downstream-arq-wiring.tape | AC-006 | `go test -v -race -run "^TestLoopbackDriver_DownstreamARQWiring$" ./internal/testenv/` |
| AC-007-dedicated-shard-no-setsink.tape | AC-007 | `go test -v -race -run "^(TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation\|TestSessionAccessNode_NoSetSinkMethod)$" ./internal/testenv/` |
| AC-008-roundtrip-token-isolation.tape | AC-008 | `go test -v -race -run "^(TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips\|TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer)$" ./internal/testenv/` |
| AC-009-waitforecho-timeout-no-leak.tape | AC-009 | `go test -v -race -run "^TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak$" ./internal/testenv/` |
| AC-010-pathtracker-active-without-probe.tape | AC-010 | `go test -v -race -run "^TestNewLoopbackPaths_TrackersActiveWithoutProbe$" ./internal/testenv/` |
| AC-011-cleanup-diagnostic-on-pending-leak.tape | AC-011 | `go test -v -race -run "^(TestLoopbackEnv_Cleanup_DiagnosticOnPendingLeak\|TestLoopbackEnv_Cleanup_SilentWhenDrained)$" ./internal/testenv/` |
| AC-012-ticker-goroutines-join-on-close.tape | AC-012 | `go test -v -race -run "^TestLoopbackEnv_TickerGoroutines_JoinOnClose$" ./internal/testenv/` |
| AC-013-vp042-bench.tape | AC-013 | compile-gate + `go test -tags integration -bench 'KeystrokeToEcho' ...` |
| AC-014-roundtrip-completes-single-arq-instance.tape | AC-014 | `go test -v -race -run "^TestLoopbackEnv_RoundTripCompletes_SingleSharedARQInstance$" ./internal/testenv/` |
| AC-015-no-race-under-concurrent-send-echo.tape | AC-015 | `go test -v -race -run "^TestLoopbackDriver_NoRaceUnderConcurrentSendEcho$" ./internal/testenv/` |
| AC-016-onack-error-surfaces-loud.tape | AC-016 | `go test -v -race -run "^TestLoopbackDriver_OnAckError_SurfacesLoud$" ./internal/testenv/` |
| AC-017-upstream-delivery-error-surfaces-loud.tape | AC-017 | `go test -v -race -run "^TestLoopbackDriver_UpstreamDeliveryError_SurfacesLoud$" ./internal/testenv/` |

Per POL-004 / `docs/DEMO-EVIDENCE-POLICY.md`, only `.tape` sources and this report are committed. Rendered
`.gif`/`.webm` are gitignored under `docs/demo-evidence/**` — regenerate locally with
`vhs docs/demo-evidence/S-BL.LOOPBACK-FULLSTACK/<file>.tape` if a visual render is needed.

---

## DEMO-ISSUE Log

None. All 16 in-scope ACs (AC-002 .. AC-017) recorded at FULL coverage. AC-001 is discharged (no demo
required, see above). The VP-042 benchmark ran cleanly in-worktree — no forward-obligation deferral was
needed for this recording pass.
