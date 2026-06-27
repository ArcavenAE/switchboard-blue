---
artifact_id: S-W3.04-daemon-assembly
document_type: story
level: ops
story_id: S-W3.04
title: "full daemon assembly — wire all Wave-3 subsystems in cmd/switchboard"
status: ready
producer: story-writer
timestamp: 2026-06-27T00:00:00Z
phase: 2
epic: E-3
wave: 3
priority: P0
scope_phase: E
estimated_points: 8
bc_traces:
  - BC-2.04.001
  - BC-2.04.002
  - BC-2.04.003
  - BC-2.04.004
  - BC-2.04.005
  - BC-2.04.006
  - BC-2.04.007
  - BC-2.05.008
vp_traces: [VP-058]
subsystems: [session-access, admission-security]
architecture_modules: [cmd/switchboard, internal/tmux]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-3.03, S-3.01b, S-3.04]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.001.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.002.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.003.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.004.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.005.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.006.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.007.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.008.md'
  - '.factory/specs/architecture/ARCH-01-core-services.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'
  - '.factory/specs/prd-supplements/error-taxonomy.md'
acceptance_criteria_count: 9
version: "1.3"
# BC-2.04.007 authored by PO 2026-06-27. AC-007 traces to BC-2.04.007 PC-1;
# AC-008 traces to BC-2.04.007 PC-2. Gate blocker resolved; status flipped to ready.
# v1.2 2026-06-27: adversarial-convergence adjudication; architect ARCH-01 v1.4 /
# ARCH-08 v2.0 rulings applied. AC-001 non-tautology wiring obligation; EC-003
# two-counter clarification; AC-006 dual-counter log format (relay=<N> consoles=<M>);
# AC-007 mid-session double-failure path (BC-2.04.007 PC-2.6/EC-007/Inv-5); Tasks
# updated with shared-keyset, RelayDropped, Err() drain, and busy-spin guard
# implementation obligations.
# v1.3 2026-06-27: S-W3.04 adversarial convergence pass-2 — HIGH-A/HIGH-B/MEDIUM
# rulings from ARCH-01 v1.5 + ARCH-08 v2.1 + BC-2.04.002 v1.3 + BC-2.04.007 v1.2
# + error-taxonomy v2.1 propagated. (HIGH-A) AC-009 added: PTY-EOF no-spin obligation
# (ErrPTYSourceEOF on sc.Err(), relay exits within bounded time, E-SYS-002 drain path).
# (HIGH-B) runAccess injection seam: task 6b added (split into runAccess/
# runAccessWithConnector + connectorIface); task 7 updated — TestRunAccessWithConnectorPC26
# and TestRunAccessWithConnectorPC2 replace tautological TestDaemonMidSessionDoubleFailureExitsNonZero.
# (MEDIUM §6.5.2) internal/frame added to approved import set in task 10 and Architecture
# Compliance table. (MEDIUM EC-005) task 13 added — correct EC-005 comment; Wave-4
# CI follow-up recorded explicitly.
---

# S-W3.04: Full Daemon Assembly — Wire All Wave-3 Subsystems in cmd/switchboard

> **Execute:** `/vsdd-factory:deliver-story S-W3.04`

> **Classification:** Wave 3 FIX-NOW gate blocker (F-1). Closes drift items W3-R3-F1,
> W3-M-3, W3-R2-M3, W3-R2-M4, and the SessionConnector half of W3-M-2.

## Narrative

- **As an** operator deploying Switchboard in access-node mode
- **I want** `cmd/switchboard` to wire all Wave-3 subsystems with real, non-nil
  implementations (routing logger, live session auth, sweep eviction, frame delivery
  bridge, and observability ticker)
- **So that** the access node behaves as specified by BCs 2.04.001–006 and 2.05.008
  in production, rather than silently failing-open or dropping observability

## Open Question for Product Owner (GATE-BLOCKING — resolve before TDD dispatch)

> **BC-2.04.007 (Daemon Lifecycle) — PO decision required before this story enters TDD.**
>
> The architect flagged that NO existing BC captures the daemon **startup and shutdown
> lifecycle** as a formal behavioral contract. Specifically:
>
> - `main()` → `run(stdout, args)` dispatch, signal handling (SIGTERM/SIGINT → context
>   cancel, SIGKILL → immediate exit), and non-zero exit on connect failure are exercised
>   by AC-007 (TestDaemonConnectFailureExitsNonZero) and AC-008 (TestDaemonCleanShutdown)
>   but have no formal BC anchor.
> - AC-007 and AC-008 are currently traced to BC-2.04.001 (access-node startup premise)
>   as the nearest behavioral anchor. This is a weak trace — BC-2.04.001 specifies tmux
>   control-mode behavior, not daemon exit semantics.
>
> **PO must decide ONE of:**
> 1. Author a new BC-2.04.007 ("daemon lifecycle: startup succeeds or exits non-zero;
>    SIGTERM triggers clean context cancel and zero exit; SIGKILL is immediate") before
>    dispatching this story to the implementer. AC-007 and AC-008 would then trace to
>    BC-2.04.007 PC-1 / PC-2.
> 2. Accept existing BCs + integration test coverage as sufficient for Wave 3, noting
>    the gap in the dependency-graph Gap Register (GAP-W3.04-LC). This story may then
>    proceed with the current weak BC-2.04.001 trace for AC-007/008.
>
> **Do NOT author BC-2.04.007 in this story.** The story-writer is flagging the gap;
> the product-owner owns BC authorship.

## Context

`cmd/switchboard/main.go` is a version-printing stub on develop at `b68e498`. All
Wave-3 internal packages (`internal/tmux`, `internal/session`, `internal/admission`,
`internal/routing`) are merged and tested in isolation. This story wires them together
in the access-node entrypoint. Per ARCH-08 §6.5.1, there are **six wiring obligations**
(all confirmed buildable-now, no hard blockers). Per ADR-011, this story also adds
`SessionConnector.Frames()` — a single new exported method in `internal/tmux` that
provides a failover-stable forwarding channel for the ctrl→PTY swap transition.

**Deferred items (explicit non-scope):** File-based config loading (`internal/config`,
Wave 4), graceful drain (`internal/drain`, Wave 4 — S-W3.04 uses `os/signal` +
`context.WithCancel` for clean-exit only), HTTP metrics endpoint (`internal/metrics`,
Wave 4), `sbctl` CLI integration. These items MUST NOT be implemented in this story;
their absence is intentional per ARCH-08 §6.5.2.

## Acceptance Criteria

### AC-001 (traces to BC-2.05.008 PC-2 + invariant 1)
`cmd/switchboard` constructs a `routing.Router` via `routing.NewRouter` with a real
`routing.Logger` injected (not `nil` or a no-op sink). When `RouteFrame` is called on
the router instance constructed by the daemon (the same `*routing.Router` produced by
`buildRouter(keys)` in `runAccess`, sharing the daemon's single `AdmittedKeySet`),
with an HMAC-bad frame, the log event E-ADM-016 is written to the injected logger's
output. The test MUST exercise the daemon's OWN router instance — not a separately
constructed parallel router — so the production wiring (shared `AdmittedKeySet`, real
non-noop `Logger`) is verified non-tautologically. The router is NOT in the live frame
data path in Wave 3 (no network-ingress listener); this obligation verifies it is
constructed and logger-wired correctly for a future ingress story (ARCH-08 v2.0
§6.5.1 obligation 1).
- **Test:** `TestRouterLoggerEmitsEADM016`

### AC-002 (traces to BC-2.04.005 PC-3 + BC-2.04.003 PC-3)
`cmd/switchboard` wires `*session.SessionAuth` as the live `Authorizer` (not
`NoOpAuthorizer` or nil). An unregistered console key attempting an upstream keystroke
is rejected with E-ADM-007; a registered read-only key's upstream frames are rejected;
a registered full-access key's upstream frames are forwarded. The fail-open default
(W3-M-3) is closed.
- **Test:** `TestDaemonAuthRejectsUnregisteredConsole`

### AC-003 (traces to BC-2.04.004 PC-1 + PC-3)
`cmd/switchboard` instantiates a `time.Ticker` in `main()` (or the access mode handler)
and calls `accessNode.Sweep(deadline)` on each tick. After the sweep deadline passes, a
console that has not sent a keepalive is removed from the fan-out set; subsequent
`SendKeystroke` calls for that console return `ErrConsoleNotFound`.
- **Test:** `TestDaemonSweepEvictsStaleConsole`

### AC-004 (traces to BC-2.04.001 PC-5 + BC-2.04.002 PC-4)
`SessionConnector.Frames()` is added to `internal/tmux` (per ADR-011). The method
returns a stable forwarding channel (`sc.frames`) that continues delivering frames
across a ctrl→PTY failover. When control mode drops and `SessionConnector` activates
PTY proxy, frames from the new PTY backend appear on the same channel returned by
the earlier `Frames()` call without the consumer goroutine needing to re-subscribe.
- **Test:** `TestSessionConnectorFramesSurviveFailover`

### AC-005 (traces to BC-2.04.001 PC-5 + BC-2.04.003 PC-2)
`cmd/switchboard` pipes `sc.Frames()` → `accessNode.DeliverFrame()` in a goroutine
after `sc.Connect(ctx)` succeeds. Frames emitted by the `SessionConnector` arrive in
`AccessNode`'s downstream fan-out within the same goroutine cycle (no extra buffering
introduced). This wiring closes W3-R2-M4.
The `Frames()` → `DeliverFrame` bridge goroutine exits cleanly when `sc.frames` is
closed (i.e., when `sc.Close()` is called on daemon shutdown).
- (No standalone test for the bridge goroutine alone; covered by integration tests
  in AC-004 and AC-008.)

### AC-006 (traces to BC-2.04.006 invariant 4 — NFR-004 observability obligation)
`cmd/switchboard` starts a 30-second `time.Ticker` after `sc.Connect(ctx)` succeeds.
On each tick, a structured log line is written at INFO level: `"frames_dropped
relay=<N> consoles=<M>"`, where `<N>` is `sc.RelayDropped()` (frames dropped at the
relay layer in `forwardFrames`) and `<M>` is `accessNode.FramesDropped()` (frames
dropped at the ConsoleSet fan-out layer). Both counters are read without resetting
(cumulative). The log line is emitted regardless of whether either counter is zero, so
operators can distinguish a relay-layer overload (`relay=<N>` non-zero) from a stalled
console (`consoles=<M>` non-zero). This closes drift W3-R2-M3 and implements BC-2.04.006
v1.4 invariant 4 (both counters required in the same log line, per ARCH-01 v1.4 counter
scope clarification).
- **Test:** `TestDaemonFramesDroppedLoggedOnTick`

### AC-007 (traces to BC-2.04.007 PC-1 + PC-2.6 + EC-007 + invariant 5)
If `sc.Connect(ctx)` returns a non-nil error (both control mode and PTY fallback fail),
`cmd/switchboard` logs the error at ERROR level, emits a human-readable message to
stdout/stderr, and exits with non-zero exit code. The process does not panic.
Additionally, if `sc.Err()` delivers a non-nil error AFTER `sc.Connect` succeeds
(mid-session double-failure: control-mode drop followed by PTY-fallback failure, OR
PTY-source EOF via `ErrPTYSourceEOF` — both are triggers of the same PC-2.6 drain path),
the daemon logs E-SYS-002 at ERROR level to the INJECTED stderr, cancels the root
context, and exits with code 1 (BC-2.04.007 v1.1 PC-2.6 / EC-007 / invariant 5). The
`sc.Err()` drain goroutine MUST be tracked in `sync.WaitGroup` so it is joined during
shutdown (invariant 5). If `sc.Err()` is not drained, BC-2.04.002 invariant 3
("never silent") is violated.
PC-2.6 and PC-2 MUST be tested through the real `runAccessWithConnector` call graph
(ARCH-01 ADR-011 v1.5 §HIGH-B) — NOT through a test-local reconstruction of the drain
logic. See task 6b and task 7 below.
- **Test:** `TestDaemonConnectFailureExitsNonZero`
- **Test:** `TestRunAccessWithConnectorPC26` (supersedes `TestDaemonMidSessionDoubleFailureExitsNonZero`)
- **Test:** `TestRunAccessWithConnectorPC2` (clean ctx-cancel → nil return, no E-SYS-002 written)

### AC-008 (traces to BC-2.04.007 PC-2)
When SIGTERM or SIGINT is received, the daemon cancels its root context. All goroutines
that depend on the context (relay goroutine, sweep ticker, frames-dropped ticker)
observe the cancellation and exit within one ticker period or immediately (whichever
comes first). The process exits with code 0. The shutdown does not leak goroutines
(verified with `t.Cleanup` and a short timeout in integration tests).
- **Test:** `TestDaemonCleanShutdown`

### AC-009 (traces to BC-2.04.002 EC-008; BC-2.04.007 EC-007 + PC-2.6)
When the access node is in PTY mode and the PTY shell process exits (EOF on PTY master)
WITHOUT `sc.Close()` being called, `sc.Err()` delivers an error satisfying
`errors.Is(err, tmux.ErrPTYSourceEOF)` and the `forwardFrames` relay goroutine exits
within a bounded time. The relay MUST NOT busy-spin. The `forwardFrames` relay detects
`srcCh==prevSrcCh` in PTY mode and sends `ErrPTYSourceEOF` on `sc.errCh` via
`sc.closeErrCh.Do` (buffered-1, non-blocking) then returns. The daemon's PC-2.6 drain
path (the `sc.Err()` drain goroutine in `runAccessWithConnector`) receives the error,
logs it at ERROR level as E-SYS-002 format
(`"fatal: cannot connect to session backend: session connector: PTY source EOF"`), cancels
the root context, and the process exits with code 1. The test MUST FAIL on a
busy-spin/no-exit regression (i.e., a relay still running after the bounded timeout
causes the test to fail).
- **Test:** `TestForwardFramesPTYEOFExitsCleanly` in `internal/tmux` (connector_frames_test.go
  or connector_eof_test.go) — construct `SessionConnector` in PTY mode via `WithPTYAllocFunc`
  (injected fake returning a pipe-pair master), call `sc.Connect(ctx)`, close the write end
  of the pipe simulating PTY shell exit, assert within ≤100ms that `sc.Err()` delivers
  `ErrPTYSourceEOF` (or a wrapping error satisfying `errors.Is`) and the relay goroutine
  has exited. A `t.Cleanup` with `time.AfterFunc` enforces the deadline; test fails if relay
  is still running.

## Edge Cases

| ID | Source | Description | Expected Behavior |
|----|--------|-------------|-------------------|
| EC-001 | BC-2.04.001 EC-002 | tmux control mode fails; PTY fallback activated at startup | `SessionConnector` transparently activates PTY; `Frames()` channel remains open; daemon continues normally |
| EC-002 | BC-2.04.002 EC-003 | Mid-session tmux server crash (control-mode loss after connect) | `SessionConnector.Frames()` forwarding relay re-reads `activeFrSource()` under `sc.mu` and switches to PTY without closing the consumer channel (ADR-011) |
| EC-003 | ADR-011 §Concurrency | `sc.frames` full when relay tries to write a new frame | Frame dropped at relay layer (non-blocking select in `forwardFrames`); `sc.RelayDropped()` counter incremented (atomic); no panic. Note: `FramesDropped()` (ConsoleSet-level) is NOT incremented by a relay-level drop — two separate counters at two separate layers, both surfaced by the AC-006 log ticker (BC-2.04.006 v1.4). |
| EC-004 | BC-2.04.004 EC-002 | Console crashes without sending explicit detach | `Sweep` detects keepalive absence past deadline; console evicted; downstream channel closed |
| EC-005 | ARCH-08 §6.2 | `cmd/switchboard` imports `internal/config`, `internal/drain`, or `internal/metrics` | FORBIDDEN — these packages do not exist on develop; any import is a build failure |

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|----------------|
| access mode handler (`access.go`) | cmd/switchboard | effectful (process lifecycle) |
| `NewRouter` + `WithLogger` wiring | internal/routing | boundary |
| `NewAccessNode` + `SessionAuth` wiring | internal/session | boundary |
| Sweep ticker | cmd/switchboard + internal/session | effectful (time.Ticker) |
| `SessionConnector.Frames()` forwarding channel | internal/tmux | effectful (goroutine + channel) |
| Frames() → DeliverFrame bridge goroutine | cmd/switchboard | effectful |
| FramesDropped ticker | cmd/switchboard | effectful (time.Ticker) |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| cmd/switchboard | effectful | Top-level binary; owns process lifecycle, signal handling, OS I/O |
| internal/tmux | effectful | Child process (tmux -C), PTY I/O, goroutine lifecycle |
| internal/session | boundary | Mutable session authorization state; enforces Tier-2 boundary |
| internal/routing | boundary | Mutable forwarding table; enforces HMAC boundary |

## Token Budget Estimate

| Context Source | Estimated Tokens |
|----------------|-----------------|
| This story spec | ~1,700 |
| BC-2.04.001.md | ~800 |
| BC-2.04.002.md (v1.3 — EC-008 added) | ~900 |
| BC-2.04.003.md | ~700 |
| BC-2.04.004.md | ~800 |
| BC-2.04.005.md | ~600 |
| BC-2.04.006.md | ~900 |
| BC-2.04.007.md (v1.2 — EC-007 extended, E-SYS-003) | ~800 |
| BC-2.05.008.md | ~900 |
| ARCH-01-core-services.md (ADR-010, ADR-011 v1.5) | ~1,500 |
| ARCH-08-dependency-graph.md (§6.5–§6.6 v2.1) | ~1,100 |
| error-taxonomy.md (v2.1 — E-SYS-003) | ~500 |
| internal/tmux package (existing, ~4 files) | ~1,500 |
| internal/session package (existing, ~4 files) | ~1,200 |
| internal/routing/routing.go | ~500 |
| internal/admission (AdmittedKeySet) | ~400 |
| cmd/switchboard/main.go (stub) | ~100 |
| New/modified test files (4 files) | ~1,800 |
| Tool outputs overhead | ~300 |
| **Total** | **~17,000** |
| Agent context window | 200K |
| **Budget usage** | **~8.5%** |

## Tasks

1. [ ] Read BC-2.04.001, BC-2.04.002, BC-2.04.003, BC-2.04.004, BC-2.04.005, BC-2.04.006,
       BC-2.04.007, BC-2.05.008, ARCH-01 (ADR-010 + ADR-011), ARCH-08 §6.5.1–§6.5.2
2. [ ] Add `SessionConnector.Frames() <-chan halfchannel.ChannelFrame` to `internal/tmux`
       per ADR-011 design (forwarding channel + relay goroutine in `pty_fallback.go` or
       new `connector_frames.go`; closed exactly once via `sync.Once` in `sc.Close()`)
3. [ ] Write failing tests: `TestSessionConnectorFramesSurviveFailover` (in `internal/tmux`)
4. [ ] Verify Red Gate for `SessionConnector.Frames()` — test fails before implementation
5. [ ] Implement `SessionConnector.Frames()` relay goroutine per ADR-011 concurrency contract
6. [ ] Create `cmd/switchboard/access.go` (or expand `main.go`) with the access mode handler:
       a. Construct `admission.AdmittedKeySet`, `session.Publisher`, `session.SessionAuth`
       b. Wire `session.NewAccessNode(pub, auth, session.WithKeystrokeSink(sc))`
       c. Construct `routing.NewRouter(admittedKeySet, routing.WithLogger(log.New(os.Stderr, "", 0)))`
       d. Call `sc.Connect(ctx)` — on error: log + exit non-zero (AC-007)
       e. Start `sc.Frames()` → `accessNode.DeliverFrame()` bridge goroutine
       f. Start sweep ticker (`time.NewTicker(sweepInterval)`) → `accessNode.Sweep(deadline)` loop
       g. Start frames-dropped ticker (`time.NewTicker(30 * time.Second)`) → log AC-006
       h. Install `os/signal` handler for SIGTERM/SIGINT → `cancel()` (AC-008)
       i. Block on context cancellation; wait for goroutines via `sync.WaitGroup`; exit 0
6a. [ ] **Adversarial-convergence wiring obligations (ARCH-01 v1.4 / ARCH-08 v2.0 rulings — apply
        before writing any access.go implementation body):**
        (1) Construct ONE shared `*admission.AdmittedKeySet` and pass it to BOTH `buildAccessNode`
            AND `buildRouter` — no separate keyset constructed inside `buildAccessNode`. The router
            instance returned by `buildRouter(keys)` MUST be assigned to a named variable (never
            `_ = buildRouter(keys)`); it is used by AC-001's test surface.
        (2) Add `SessionConnector.RelayDropped() uint64` and an `atomic.Uint64 relayDropped` field
            to `SessionConnector`; increment it atomically on each non-blocking-select drop in
            `forwardFrames`. This is the relay-layer counter (distinct from `AccessNode.FramesDropped()`).
        (3) Add `sc.Err() <-chan error` (or equivalent) if not already present; drain it in a
            `wg`-tracked goroutine that, on receiving a non-nil error, logs E-SYS-002 at ERROR
            level and calls `cancel()` (mid-session double-failure path per BC-2.04.007 PC-2.6 /
            EC-007 / invariant 5). The drain goroutine MUST be added to the `sync.WaitGroup` before
            the relay goroutine starts.
        (4) In the `forwardFrames` relay goroutine's outer loop, select on `ctx.Done()` so the
            goroutine exits cleanly when the context is cancelled. Add a `runtime.Gosched()` call
            on the retry path when the source backend channel is closed but context is not yet
            cancelled (ARCH-01 v1.4 §Relay busy-spin guard — prevents hot spin on closed source).
        **Test-quality rule:** AC-001/AC-002 tests MUST assert against the router/access-node
        instances returned by the production constructors (`buildRouter`, `buildAccessNode`), not
        against separately constructed parallel instances. Separate reconstruction would make the
        test tautological (it would not exercise the shared `AdmittedKeySet` wiring).
6b. [ ] **runAccess injection seam (ARCH-01 ADR-011 v1.5 §HIGH-B + ARCH-08 v2.1 §6.5.1 obligation 4):**
        Split `runAccess` into two functions in `cmd/switchboard/access.go`:
        (1) `runAccess(ctx context.Context, stderr io.Writer) error` — thin wrapper: constructs the
            real `*tmux.SessionConnector` via the default PTY allocator (same construction as current
            `runAccess` body), calls `buildAccessComponents` to obtain `an *session.AccessNode` and
            `router *routing.Router`, then delegates to `runAccessWithConnector`.
        (2) `runAccessWithConnector(ctx context.Context, stderr io.Writer, sc connectorIface,
            an *session.AccessNode, router *routing.Router) error` — holds ALL orchestration logic:
            wiring obligations 3–6 from ARCH-08 §6.5.1, the PC-1 connect-failure path, the PC-2
            clean-shutdown path, the PC-2.6 `sc.Err()` drain path, and the `internalFailure` exit-code
            latch. `buildAccessComponents` signature is UNCHANGED; it is called from `runAccess` before
            the delegation.
        Define `connectorIface` (unexported) with exactly five methods:
            `Connect(ctx context.Context) error`
            `Frames() <-chan halfchannel.ChannelFrame`
            `Err() <-chan error`
            `Close() error`
            `RelayDropped() uint64`
        `*tmux.SessionConnector` satisfies `connectorIface` by construction (no changes to
        `internal/tmux` needed for the interface itself). The seam is an internal refactoring within
        `cmd/switchboard`; it introduces no new packages or import edges.
7. [ ] Write failing tests: `TestRouterLoggerEmitsEADM016`, `TestDaemonAuthRejectsUnregisteredConsole`,
       `TestDaemonSweepEvictsStaleConsole`, `TestDaemonFramesDroppedLoggedOnTick`,
       `TestDaemonConnectFailureExitsNonZero`, `TestDaemonCleanShutdown` (in `cmd/switchboard/`);
       also `TestRunAccessWithConnectorPC26` and `TestRunAccessWithConnectorPC2`
       (in `cmd/switchboard/access_test.go`) — both MUST call `runAccessWithConnector` directly
       with an injected `fakeConnector` (unexported stub in `access_test.go`):
         - `TestRunAccessWithConnectorPC26`: `fakeConnector.Err()` yields a non-nil error
           → assert `runAccessWithConnector` returns non-nil AND writes E-SYS-002 format to the
           injected `stderr` writer. This supersedes `TestDaemonMidSessionDoubleFailureExitsNonZero`
           (the old test exercised a test-local reconstruction of drain logic, not the production
           function — making it tautological per ARCH-01 ADR-011 v1.5 §HIGH-B).
         - `TestRunAccessWithConnectorPC2`: `fakeConnector` with no error on `Err()`, context
           cancelled externally → assert `runAccessWithConnector` returns nil (exit 0) AND
           E-SYS-002 format is NOT written to stderr.
8. [ ] Verify Red Gate: all daemon tests (including the two new `runAccessWithConnector` tests) fail
       before access.go implementation
9. [ ] Implement access.go (wiring body including the `runAccess`/`runAccessWithConnector` split and
       `connectorIface`) to make all tests green
10. [ ] Verify ARCH-08 §6.5.2: `cmd/switchboard` imports only
        `{internal/admission, internal/frame, internal/routing, internal/session, internal/tmux,
        internal/halfchannel}` plus stdlib — no `internal/config`, `internal/drain`,
        `internal/metrics`. Note: `internal/frame` is now part of the approved set (ARCH-08 v2.1
        §6.5.2 MEDIUM ruling) — `startFramesBridge` uses `frame.OuterHeader` when calling
        `AccessNode.DeliverFrame`. `internal/frame` is a DAG position-2 leaf; no forbidden edge.
11. [ ] `just fmt && just lint` pass with zero warnings
12. [ ] `just test-race` passes (relay goroutine and `sync.Once` close must be data-race free)
13. [ ] **EC-005 comment correction (ARCH-01 ADR-011 v1.5 §EC-005 / ARCH-08 v2.1 §6.5.2 accepted
        Wave-4 follow-up):** In `cmd/switchboard/access.go`, locate the EC-005 comment on the
        forbidden-package guard and replace the text `"CI enforces this structurally"` with:
        `"Build fails because internal/config, internal/drain, internal/metrics do not yet exist
        on develop; a durable go-list CI assertion enforcing this boundary after those packages
        land is deferred to Wave 4."` This is NOT a Wave-3 blocker; the current enforcement
        (compile-time nonexistence) remains effective. The durable CI import-guard is an explicit
        Wave-4 follow-up item — do not implement it here.

## Previous Story Intelligence

| Decision | Rationale | Applies To |
|----------|-----------|-----------|
| `ConsoleSet` never returns internal pointers (S-3.02, go.md rule 12) | Lock protects map, not values | `AccessNode` methods used from `cmd/switchboard` must receive value copies; mutation goes through methods under lock |
| `SessionAuth` as live `Authorizer` replaces `NoOpAuthorizer` (S-3.03) | Fail-open default is unacceptable at production wiring | `NewAccessNode(pub, auth, ...)` — `auth` MUST be `*session.SessionAuth`, never nil or no-op |
| `verifyFrameHMAC` wired into `RouteFrame` (S-3.04); `ErrHMACVerificationFailed` sentinel exists | Wire-layer HMAC is confirmed working | `WithLogger` injection is the only gap; no changes to `verifyFrameHMAC` |
| `ControlMode.Frames()` and `PTYProxy.Frames()` each return a per-instance channel (S-3.01a, S-3.01b) | Per ADR-011: these channels close on mode switch | `SessionConnector.Frames()` solves this via the forwarding channel; `cmd/switchboard` ranges over `sc.Frames()` only |
| `sync.Once` for channel close (S-3.01b PTY proxy pattern) | Channel close panics on double-close | ADR-011 specifies `closeForwardFrames sync.Once` in `SessionConnector.Close()` |

## Architecture Compliance Rules

| Rule | Source | Enforcement |
|------|--------|-------------|
| `cmd/switchboard` imports only positions 1–7 plus stdlib | ARCH-08 §6.5.2 v2.1 (import set enumerated, includes `internal/frame`) | `go vet` import cycle detection; lint |
| `internal/config`, `internal/drain`, `internal/metrics` MUST NOT be imported | ARCH-08 §6.5.2 deferred-packages list | Build fails if those packages don't exist; CI fails if import added |
| `SessionConnector.Frames()` relay goroutine exits via `sc.frames` close | ADR-011 §Concurrency contract | `TestDaemonCleanShutdown` + `just test-race` |
| No `log.Fatal` / `os.Exit` outside `main()` | ARCH-01 §Supervision; go.md rule | golangci-lint |
| No `init()` functions; all deps via constructors | ARCH-01 §Supervision; go.md rule | golangci-lint |
| `context.Context` is first parameter on all funcs that take one | go.md rule 7 | golangci-lint |
| `sync.Mutex` held for trim+append+check — no internal pointer leaks | go.md rule 12 | `just test-race` |
| Timestamps in UTC | go.md rule 11 | Code review |
| `SessionConnector.Frames()` closes exactly once via `sync.Once` | ADR-011; prevents double-close panic | `just test-race` |

## Forbidden Dependencies

The following packages MUST NOT appear in `cmd/switchboard`'s import graph (go.mod / go list):

| Package | Reason |
|---------|--------|
| `internal/config` | Not on develop; Wave 4+ |
| `internal/drain` | Not on develop; Wave 4+ |
| `internal/metrics` | Not on develop; Wave 4+ |
| `cmd/sbctl` | Top leaf; never imported |

If `cmd/switchboard` gains a dependency on any of these, the build MUST fail (they
do not exist on develop and cannot be imported). The EC-005 comment in `access.go`
MUST read: "Build fails because internal/config, internal/drain, internal/metrics do
not yet exist on develop; a durable go-list CI assertion enforcing this boundary after
those packages land is deferred to Wave 4." (ARCH-01 ADR-011 v1.5 §EC-005; task 13.)
Note: `internal/frame` is NOT in the forbidden set — it is an approved import per
ARCH-08 §6.5.2 v2.1 (DAG position 2 leaf, no forbidden edge).

## Library & Framework Requirements

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Per go.mod |
| `os/signal` | stdlib | SIGTERM/SIGINT handling |
| `context` | stdlib | Root context + cancellation |
| `sync` | stdlib | `sync.WaitGroup` (goroutine lifecycle), `sync.Once` (channel close) |
| `time` | stdlib | `time.Ticker` (sweep + frames-dropped); `time.Now().UTC()` |
| `log` | stdlib | `log.New(os.Stderr, "", 0)` for router logger |
| `testing` | stdlib | Table-driven tests; `t.Parallel()`; no testify |

## File Structure Requirements

| File | Action | Purpose |
|------|--------|---------|
| `internal/tmux/pty_fallback.go` (or new `connector_frames.go`) | modify / create | Add `SessionConnector.Frames() <-chan halfchannel.ChannelFrame`; add `sc.frames` field + relay goroutine + `closeForwardFrames sync.Once` per ADR-011 |
| `internal/tmux/pty_fallback_test.go` (or `connector_frames_test.go`) | create / extend | `TestSessionConnectorFramesSurviveFailover` |
| `internal/tmux/connector_frames_test.go` (or `connector_eof_test.go`) | create | `TestForwardFramesPTYEOFExitsCleanly` (AC-009 — PTY EOF no-spin regression guard) |
| `cmd/switchboard/access.go` | create | Access-mode wiring: all six obligations from ARCH-08 §6.5.1; `runAccess` + `runAccessWithConnector` + `connectorIface` |
| `cmd/switchboard/access_test.go` | create | `TestRunAccessWithConnectorPC26`, `TestRunAccessWithConnectorPC2` (inject `fakeConnector`) |
| `cmd/switchboard/main.go` | modify | Replace version-stub `run()` body with subcommand dispatch; retain `run(stdout io.Writer, args []string) error` signature |
| `cmd/switchboard/main_test.go` | create / extend | Integration tests: AC-001 through AC-008 (excluding AC-007 PC-2.6/PC-2 which live in access_test.go) |

## Spec Patches

| Version | Date | Change |
|---------|------|--------|
| 1.3 | 2026-06-27 | S-W3.04 adversarial convergence pass-2 — ARCH-01 v1.5 / ARCH-08 v2.1 / BC-2.04.002 v1.3 / BC-2.04.007 v1.2 / error-taxonomy v2.1 propagated. (HIGH-A) AC-009 added: PTY-EOF no-spin obligation — relay detects `srcCh==prevSrcCh` in PTY mode, sends `ErrPTYSourceEOF` on `sc.errCh` via `sc.closeErrCh.Do`, exits without busy-spinning; E-SYS-002 drain path logs and exits 1; test `TestForwardFramesPTYEOFExitsCleanly` in `internal/tmux` must fail on busy-spin/no-exit regression. Traces to BC-2.04.002 EC-008 and BC-2.04.007 EC-007 + PC-2.6. (HIGH-B) runAccess injection seam: task 6b added — split `runAccess` into thin ctor wrapper + `runAccessWithConnector(ctx, stderr, connectorIface, an, router)` holding all orchestration/PC-1/PC-2/PC-2.6 logic; `connectorIface` (unexported, 5 methods: Connect/Frames/Err/Close/RelayDropped); `buildAccessComponents` signature unchanged; AC-007 updated to reference `runAccessWithConnector` end-to-end tests; `TestRunAccessWithConnectorPC26` + `TestRunAccessWithConnectorPC2` in `access_test.go` replace tautological `TestDaemonMidSessionDoubleFailureExitsNonZero`. (MEDIUM §6.5.2) `internal/frame` added to approved import set in task 10, Architecture Compliance table, and Forbidden Dependencies note. (MEDIUM EC-005) task 13 added: correct EC-005 comment from "CI enforces this structurally" to accurate wording; Wave-4 durable CI import-guard is an explicit follow-up non-blocker. Token budget updated: +300 tokens (v1.3 story size, error-taxonomy.md, additional test file). |
| 1.2 | 2026-06-27 | Adversarial-convergence adjudication + architect ARCH-01 v1.4 / ARCH-08 v2.0 rulings. (A) AC-001: replaced tautology-risk sentence with non-tautological wiring obligation — test must target daemon's OWN router instance (shared `AdmittedKeySet`, `buildRouter` return value must not be discarded); router is constructed-but-not-in-live-data-path per ARCH-08 v2.0 §6.5.1 obligation 1. (B) EC-003: corrected counter name from `FramesDropped` to `sc.RelayDropped()` (relay-layer); added clarification that ConsoleSet-level `FramesDropped()` is a separate counter not incremented by relay drops (BC-2.04.006 v1.4). (C) AC-006: updated log format to `"frames_dropped relay=<N> consoles=<M>"` (both counters, cumulative, no reset); log emitted unconditionally per BC-2.04.006 v1.4 invariant 4. (D) AC-007: extended with mid-session double-failure path (BC-2.04.007 v1.1 PC-2.6 / EC-007 / invariant 5) — `sc.Err()` non-nil triggers E-SYS-002, cancel, exit 1; drain goroutine must be wg-tracked; added `TestDaemonMidSessionDoubleFailureExitsNonZero`. (E) Tasks: added item 6a with five implementation obligations (shared keyset, `RelayDropped` counter, `sc.Err()` drain goroutine, `forwardFrames` ctx.Done select, busy-spin guard) and test-quality rule against parallel reconstruction. |
| 1.1 | 2026-06-27 | PO decision (A): BC-2.04.007 authored (daemon lifecycle contract); AC-007 re-traced to BC-2.04.007 PC-1; AC-008 re-traced to BC-2.04.007 PC-2; E-SYS-002 registered in error-taxonomy.md; BC-INDEX updated; story flipped draft→ready |
| 1.0 | 2026-06-27 | Initial story — Wave 3 FIX-NOW gate blocker F-1; closes W3-R3-F1, W3-M-3, W3-R2-M3, W3-R2-M4, SessionConnector half of W3-M-2; BC-2.04.007 lifecycle gap flagged for PO |
