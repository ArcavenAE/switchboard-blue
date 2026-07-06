---
artifact_id: S-BL.TESTENV
document_type: story
level: ops
story_id: S-BL.TESTENV
title: "internal/testenv e2e harness — multi-daemon in-process rig unblocking ten deferred VPs"
status: backlog
producer: story-writer
timestamp: 2026-07-06T00:00:00Z
version: "0.1-backlog-stub"
phase: 2
epic: E-1
wave: backlog
priority: P1
scope_phase: E
estimated_points: 13
bc_traces:
  - BC-2.04.003   # VP-033: Console.Attach — downstream frame delivery
  - BC-2.04.004   # VP-033: Console.Detach — session survives
  - BC-2.04.005   # VP-034: Multi-console fan-out
  - BC-2.03.003   # VP-036: Session continuity across IP change (ConnectWithSourceIP)
  - BC-2.09.002   # VP-037: drain within window (NewWithRouters + SendDrainSignal)
  - BC-2.09.001   # VP-038: E→PE via config-only (StartRouter + Restart + RouterConfig)
  - BC-2.05.006   # VP-039: SVTN isolation (CreateSVTN + CreateSessionInSVTN + AttachProbe)
  - BC-2.02.003   # VP-040: multipath failover < 2s (NewWithRouters + WaitForPaths)
  - BC-2.05.003   # VP-046: key lifecycle register/revoke/expire (ConnectWithKey)
vp_traces:
  - VP-033   # UNPROVEN-BLOCKED — blocker: testenv.New + CreateSession + AttachConsole + SendKeystroke + CollectFrames + SessionAlive
  - VP-034   # UNPROVEN-BLOCKED — blocker: testenv.New + AttachConsole (multi-console variant)
  - VP-036   # UNPROVEN-BLOCKED — blocker: testenv.ConnectWithSourceIP (t.Skip placeholder in reauth_test.go:520)
  - VP-037   # UNPROVEN-BLOCKED — blocker: testenv.NewWithRouters + SendDrainSignal
  - VP-038   # UNPROVEN-BLOCKED — blocker: testenv.New + StartRouter + Restart + RouterConfig{UpstreamRouters}
  - VP-039   # UNPROVEN-BLOCKED — blocker: testenv.CreateSVTN + CreateSessionInSVTN + AttachProbe.FramesFromSVTN
  - VP-031   # PARTIAL — gap: real-tmux integration harness (control-mode completeness threshold)
  - VP-032   # PARTIAL — gap: real-PTY openpty(3) integration (t.Skip at pty_fallback_test.go:1311)
  - VP-040   # PARTIAL — gap: testenv.NewWithRouters + WaitForPaths + CloseRouterConnection + wall-clock timing
  - VP-046   # PARTIAL — gap: testenv.ConnectWithKey + admission-handshake infrastructure
subsystems: [transport-layer, network-management, deployment-operations, session-management]
architecture_modules:
  - internal/testenv   # NEW package — entire package is this story's deliverable
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on:
  - S-1.01    # frame codec — testenv sends/receives frames
  - S-1.02    # halfchannel — testenv tick infrastructure
  - S-2.01    # HMAC codec — testenv sessions use HMAC-authenticated frames
  - S-2.02    # admission + SVTN — CreateSVTN, ConnectWithKey require admission layer
  - S-3.01a   # tmux control mode — VP-031 real-tmux harness needs control-mode client
  - S-3.01b   # PTY proxy — VP-032 real-PTY harness needs pty fallback
  - S-4.01    # multipath dispatch — VP-040 NewWithRouters needs path selection
  - S-BL.NI   # network ingress — testenv.New requires real in-process listener
  - S-BL.OA   # outer assembler — testenv frame composition
blocks:
  - S-BL.BENCH    # VP-042 BenchmarkKeystrokeToEcho_P99 depends on testenv.NewLoopback
inputDocuments:
  - '.factory/specs/verification-properties/VP-033.md'
  - '.factory/specs/verification-properties/VP-034.md'
  - '.factory/specs/verification-properties/VP-036.md'
  - '.factory/specs/verification-properties/VP-037.md'
  - '.factory/specs/verification-properties/VP-038.md'
  - '.factory/specs/verification-properties/VP-039.md'
  - '.factory/specs/verification-properties/VP-031.md'
  - '.factory/specs/verification-properties/VP-032.md'
  - '.factory/specs/verification-properties/VP-040.md'
  - '.factory/specs/verification-properties/VP-046.md'
  - '.factory/specs/verification-properties/VP-042.md'
acceptance_criteria_count: 0
backlog_origin:
  source: Phase-6-VP-sweep
  blocking_vps: [VP-033, VP-034, VP-036, VP-037, VP-038, VP-039, VP-031, VP-032, VP-040, VP-046]
  notes: >
    The Phase-6 formal hardening VP sweep (2026-07-06) audited 77 VPs and found 14 as
    deferred/blocked. Of those 14, ten converge on a single missing infrastructure package:
    `internal/testenv`. This story creates that package.

    The testenv helper surface is derived directly from the `deferred_reason` and `Lifecycle`
    fields of the ten VP files. Every helper name below appears verbatim in at least one VP's
    proof harness skeleton.

    VP-031 and VP-032 are PARTIAL rather than UNPROVEN-BLOCKED: they have hermetic unit-test
    evidence (fake control streams, injected PTY) but require real-tmux / real-openpty(3)
    integration for the e2e gap. The testenv package provides the integration test harness
    infrastructure that enables these real-device tests.

    VP-046 is PARTIAL: the three key lifecycle properties (register, revoke, expire) are
    discharged by unit tests against real SVTNManager + admission. The remaining gap is the
    end-to-end `env.ConnectWithKey` wrapper that exercises the full admission handshake.

    VP-040 is PARTIAL: path-tracker inactivation on sustained loss is proven by unit tests
    (TestBC_2_02_003_PathTracker_InactiveAfterMisses). The gap is the wall-clock NFR-003
    timing claim (traffic resumes < 2s) which requires NewWithRouters + CloseRouterConnection
    + WaitForPaths.

    This is the single highest-leverage verification-infrastructure story in the backlog:
    six VPs are fully unblocked and four more are partially unblocked.
---

# S-BL.TESTENV: `internal/testenv` E2E Harness

> **STATUS: BACKLOG STUB.** This story creates the `internal/testenv` package that
> unblocks 10 deferred VPs from the Phase-6 audit. Acceptance criteria, task list,
> and API surface will be confirmed at scheduling time, likely with an architect
> design pass to finalize the constructor and helper shapes.

## Narrative

- **As an** implementer and formal-verifier
- **I want** an `internal/testenv` package that spins up multi-daemon in-process
  switchboard stacks with controllable SVTN configurations, session helpers, and
  connectivity probes
- **So that** the ten VP harness skeletons blocked by missing testenv infrastructure
  can be discharged without relying on external processes or flaky real-network tests

## Why This Story Is the Highest-Leverage Verification Infrastructure Story

Phase-6 VP sweep audited 63/77 VPs as PROVEN. The 14 deferred/blocked VPs split as:
- 6 blocked on `internal/testenv` (UNPROVEN-BLOCKED: VP-033/034/036/037/038/039)
- 4 partially blocked on `internal/testenv` (PARTIAL: VP-031/032/040/046)
- 2 blocked on S-BL.BENCH stable CI hardware (VP-041/042, owned by S-BL.BENCH)

One package eliminates the blocker on 10 of the 14 deferred VPs.

## Required Helper Surface (Derived from VP Deferred Reasons)

Each helper appears verbatim in at least one VP's proof harness skeleton. This is the
minimum API surface this story must deliver to unblock the listed VPs.

### Base constructors

| Helper | Signature sketch | Required by |
|--------|-----------------|-------------|
| `testenv.New` | `New(t testing.TB, ctx context.Context) *Env` | VP-033, VP-034, VP-036, VP-038, VP-039, VP-046 |
| `testenv.NewWithRouters` | `NewWithRouters(t testing.TB, ctx context.Context, n int) *Env` | VP-037, VP-040 |
| `testenv.NewLoopback` | `NewLoopback(b testing.TB, ctx context.Context, cfg LoopbackConfig) *LoopbackEnv` | VP-042 (S-BL.BENCH) |

### Session and SVTN helpers

| Helper | Signature sketch | Required by |
|--------|-----------------|-------------|
| `env.CreateSession` | `CreateSession(t testing.TB) SessionID` | VP-033, VP-034, VP-038 |
| `env.CreateSVTN` | `CreateSVTN(t testing.TB) SVTNID` | VP-039 |
| `env.CreateSessionInSVTN` | `CreateSessionInSVTN(t testing.TB, svtnID SVTNID) SessionID` | VP-039 |

### Console and probe helpers

| Helper | Signature sketch | Required by |
|--------|-----------------|-------------|
| `env.AttachConsole` | `AttachConsole(t testing.TB, sessionID SessionID) *Console` | VP-033, VP-034 |
| `console.CollectFrames` | `CollectFrames(t testing.TB, timeout time.Duration) []Frame` | VP-033, VP-034 |
| `console.Detach` | `Detach(t testing.TB)` | VP-033 |
| `env.AttachProbe` | `AttachProbe(t testing.TB) *Probe` | VP-039 |
| `probe.FramesFromSVTN` | `FramesFromSVTN(svtnID SVTNID, timeout time.Duration) []Frame` | VP-039 |

### Connectivity helpers

| Helper | Signature sketch | Required by |
|--------|-----------------|-------------|
| `env.SendKeystroke` | `SendKeystroke(t testing.TB, sessionID SessionID, key string)` | VP-033, VP-034 |
| `env.SessionAlive` | `SessionAlive(t testing.TB, sessionID SessionID) bool` | VP-033 |
| `env.ConnectWithSourceIP` | `ConnectWithSourceIP(t testing.TB, addr, srcIP string) *Conn` | VP-036 |
| `env.ConnectWithKey` | `ConnectWithKey(t testing.TB, key ed25519.PublicKey) *Conn` | VP-046 |
| `env.WaitForEcho` | `WaitForEcho(b testing.TB, sessionID SessionID, text string, timeout time.Duration)` | VP-042 |

### Router control helpers (PE and drain)

| Helper | Signature sketch | Required by |
|--------|-----------------|-------------|
| `env.StartRouter` | `StartRouter(t testing.TB, cfg RouterConfig) *RouterHandle` | VP-038 |
| `router.Restart` | `Restart(t testing.TB, cfg RouterConfig)` | VP-038 |
| `env.SendDrainSignal` | `SendDrainSignal(t testing.TB)` | VP-037 |
| `env.WaitForPaths` | `WaitForPaths(t testing.TB, n int, timeout time.Duration)` | VP-040 |
| `env.CloseRouterConnection` | `CloseRouterConnection(t testing.TB, idx int)` | VP-040 |

### Config types

| Type | Fields | Required by |
|------|--------|-------------|
| `RouterConfig` | `UpstreamRouters []string` + tick config | VP-038 |
| `LoopbackConfig` | `TickIntervalUpstream, TickIntervalDownstream time.Duration` | VP-042 |
| Mode enums | `ModeE`, `ModePE` | VP-038 |

### Real-device integration (VP-031/032 partial gap)

| Gap | Notes |
|-----|-------|
| Real tmux control-mode harness | VP-031: 99% output completeness at 10 KB/s; requires live tmux subprocess via control mode (`internal/tmux/control_test.go:120` t.Skip note) |
| Real openpty(3) harness | VP-032: real PTY integration deferred from injected-fake PTY via `WithPTYAllocFunc`; `pty_fallback_test.go:1311` t.Skip placeholder |

The real-device harnesses may live in a separate `internal/testenv/realdevice` subdirectory
(build-tagged `integration`) to keep the in-process stack separable from real-OS dependency
tests. Architecture decision is deferred to the scheduling design pass.

## Sketched Acceptance Criteria

> ACs are sketched by VP. Exact test names and BC postcondition references confirmed at
> scheduling time. Each AC removes a t.Skip or discharges a VP lifecycle gap.

**AC-001 (VP-033 — console attach/detach lifecycle):** `TestE2E_Console_AttachDetachLifecycle`
runs end-to-end using `testenv.New` + `CreateSession` + `AttachConsole` + `SendKeystroke` +
`CollectFrames` + `console.Detach` + `SessionAlive`. Discharges VP-033 UNPROVEN-BLOCKED.

**AC-002 (VP-034 — multi-console fan-out):** `TestE2E_MultiConsole_FanOut` verifies that two
consoles attached to the same session both receive downstream frames. Uses `AttachConsole ×2`
+ `CollectFrames`. Discharges VP-034 UNPROVEN-BLOCKED.

**AC-003 (VP-036 — session continuity across IP change):** `TestProperty_VP036_SessionContinuity`
t.Skip at `internal/admission/reauth_test.go:520` is removed. Uses `testenv.ConnectWithSourceIP`
to simulate IP change and verify session continuity per BC-2.03.003. Discharges VP-036 UNPROVEN-BLOCKED.

**AC-004 (VP-039 — SVTN isolation):** `TestE2E_VP039_SVTNIsolation` uses `testenv.CreateSVTN ×2`
+ `CreateSessionInSVTN` + `AttachProbe.FramesFromSVTN` to assert no cross-SVTN frame leakage.
t.Skip placeholder at `internal/routing/routing_test.go:193` removed. Discharges VP-039 UNPROVEN-BLOCKED.

**AC-005 (VP-037 partial — drain with real observer):** `testenv.NewWithRouters` + `SendDrainSignal`
enable the drain-within-window integration test. Full VP-037 lock depends on `S-7.04-FU-DRAIN-WIRE`
also landing; testenv provides the harness, the story provides the observer.

**AC-006 (VP-038 partial — E→PE via config):** `testenv.New` + `StartRouter` + `Restart(RouterConfig{
UpstreamRouters})` enable the in-process router restart e2e test. Full VP-038 lock depends on
`S-7.04-FU-SIGHUP-RELOAD` + `S-7.04-FU-PE-CONNECTOR` landing.

**AC-007 (VP-040 partial — multipath failover timing):** `testenv.NewWithRouters` + `CloseRouterConnection`
+ `WaitForPaths` + wall-clock measurement enables the NFR-003 < 2s failover timing test.

**AC-008 (VP-046 partial — key lifecycle e2e):** `testenv.ConnectWithKey` enables the full
admission-handshake integration for register/revoke/expire key lifecycle. VP-046 unit tests
already pass; this closes the `integration` proof-method gap.

**AC-009 (VP-031/032 partial — real-device harness stubs):** `t.Skip("VP-031-real-tmux")` and
`t.Skip("VP-032-real-pty")` placeholders are replaced with real-device test scaffolding
(build-tagged `//go:build integration`). The real-device tests may use `t.Skip` again if
the runtime environment lacks tmux/pty — the key deliverable is removing the generic
"integration harness not yet built" defer and replacing it with a real test that skips
cleanly when the device is absent rather than being permanently deferred.

**AC-010 (VP-042 loopback harness):** `testenv.NewLoopback(b, ctx, LoopbackConfig{...})` is
implemented. `BenchmarkKeystrokeToEcho_P99` in S-BL.BENCH can compile and link against it.
(Full VP-042 lock is in S-BL.BENCH; this AC is the harness prerequisite.)

## Non-Goals

- Does not implement the product code those VPs verify (drain wire, PE connector, etc.).
- Does not add `b.Errorf` gates to VP-041 or VP-042. That is `S-BL.BENCH`.
- Does not change any BC or VP file; does not flip `verification_lock`. Those are done
  by the stories that discharge each VP using the harness this story provides.

## When to Schedule

P1. This is the highest-leverage verification-infrastructure story. Can be prioritized
independently of any wave story. Unblocks 10 VP proofs.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-06 |
| Origin | Phase-6 VP sweep; 10 deferred VPs converge on missing internal/testenv |
| VP traces | VP-033, VP-034, VP-036, VP-037, VP-038, VP-039 (fully unblocked); VP-031, VP-032, VP-040, VP-046 (partially unblocked) |
| Points estimate | 13 (infrastructure story; multiple helper families) |
| Status transitions | (none yet) |
