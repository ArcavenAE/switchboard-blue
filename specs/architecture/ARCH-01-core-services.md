---
artifact_id: ARCH-01-core-services
document_type: architecture-section
level: L3
version: "1.6"
status: draft
producer: architect
timestamp: 2026-06-23T00:00:00
modified:
  - 2026-06-25T00:00:00 # v1.1 — Added ADR-010: tmux control mode primary, PTY proxy fallback (Wave 3 / S-3.01)
  - 2026-06-25T00:00:00 # v1.2 — ADR-010: allow mid-session PTY fallback on control-mode loss (Wave-3 reviewer F-W3-H-003 + user decision; BCs win over initial-connect-only restriction)
  - 2026-06-27T00:00:00 # v1.3 — Added ADR-011: SessionConnector.Frames() forwarding-channel API design (S-4.00 daemon assembly; drift W3-R2-M4)
  - 2026-06-27T00:00:00 # v1.4 — ADR-011 §Concurrency: amend relay-drop counter contract (relay-level drops MUST be metered via sc.relayDropped), relay busy-spin contract (ctx.Done guard REQUIRED in outer relay loop), and daemon sc.Err() drain obligation; per S-W3.04 adversarial convergence adjudication
  - 2026-06-27T00:00:00 # v1.5 — ADR-011 §Terminal-source EOF: three new rulings from S-W3.04 adversarial convergence pass-2: (HIGH-A) terminal PTY-source EOF hot-spin — PTY EOF without prior Close() is a session-fatal backend loss; relay MUST detect and surface via sc.Err(); new EC-008 (PTY-death); (HIGH-B) runAccess injection seam — split into runAccess/runAccessWithConnector; (MEDIUM) §6.5.2 internal/frame import addition
  - 2026-06-27T00:00:00 # v1.6 — ADR-011 §HIGH-A tightened: TOCTOU fix — {src, srcCh, inPTYMode} MUST be read as a single atomic snapshot under one sc.mu acquisition via activeSourceSnapshot(); two separate locked reads can straddle a watchAndFallback swap and misclassify in-flight failover as terminal PTY-EOF (EC-002 regression 1-in-5); soundness proof for the atomic-snapshot discriminator; updated test obligations (EC-002 stress + EOF unit)
phase: 1b
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
  - '.factory/specs/behavioral-contracts/ss-04/BC-2.04.001.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.003.md'
kos_anchors:
  - elem-single-binary-three-modes
  - elem-node-router-architecture
---

# ARCH-01: Core Services

## Single Binary, Six Subcommands

Per elem-single-binary-three-modes, one binary serves all deployment roles.
Mode is selected by subcommand, not by build flags (except P router).

| Subcommand | Runtime Mode | Primary Role | Phase |
|------------|-------------|-------------|-------|
| `switchboard router` | E or PE router | Frame forwarding, HMAC auth, admission | E (MVP) |
| `switchboard access` | Access node | tmux publishing, session I/O, Tier 2 auth | E (MVP) |
| `switchboard console` | Console node | Session attach/detach, downstream render | E (MVP) |
| `switchboard control` | Control node | SVTN lifecycle, key registration | E (MVP) |
| `switchboard version` | Version query | Print version, exit | E (MVP) |
| `switchboard help` | Help | Print usage, exit | E (MVP) |

The E vs PE router distinction is purely config-driven: `upstream_routers: []` = E,
any entries = PE. The binary contains both code paths; the config selects.

## cmd/switchboard Package Layout

```
cmd/
  switchboard/         # main package
    main.go            # entrypoint: run(stdout, args) function pattern
    main_test.go       # integration smoke tests
    router.go          # router subcommand handler
    access.go          # access node subcommand handler
    console.go         # console node subcommand handler
    control.go         # control node subcommand handler

cmd/
  sbctl/               # operator CLI (separate binary)
    main.go
    commands/          # one file per subcommand group
```

The existing `main.go` stub (wave-0) establishes the `run(stdout io.Writer, args []string) error`
pattern — the real implementation replaces the stub body while preserving this signature.

## Daemon Lifecycle

```
main() → run(stdout, os.Args)
  → parse global flags (--config, --log-level, --log-format)
  → dispatch to mode handler (router/access/console/control)
  → mode handler:
      1. loadConfig(path) → validate → fail with actionable error if invalid (BC-2.09.003)
      2. initLogger(level, format)
      3. buildDependencies() → pure-core modules initialized first
      4. startServices() → bind/listen (after config validated — NFR-011)
      5. installSignalHandlers(SIGTERM → graceful drain, SIGHUP → reload)
      6. serve() → event loop until shutdown
      7. shutdown() → drain active sessions, close listeners
```

## Signal Handling

| Signal | Router | Access | Console | Control |
|--------|--------|--------|---------|---------|
| SIGTERM | graceful drain (BC-2.09.002) | close sessions | detach | close |
| SIGHUP | reload config | reload config | — | reload config |
| SIGINT | same as SIGTERM | same | same | same |
| SIGKILL | immediate exit (FM-009) | immediate | immediate | immediate |

`sbctl router drain` sends the equivalent of SIGTERM over the management socket.

## Supervision and Error Handling

- No `log.Fatal` or `os.Exit` outside `main()`. All errors propagate via `error` return.
- Config validation must complete before `bind`/`listen` call (NFR-011). Any config
  error exits with code 1 and a human-readable message identifying the field.
- Panics in the daemon are recovered at the event loop boundary and reported as
  exit code 3 (internal error).
- No `init()` functions. All dependencies are passed explicitly via constructors.

## Mode Multiplexing for BCs

| BC | Mode | Package |
|----|------|---------|
| BC-2.04.001, BC-2.04.002 | access node | internal/tmux |
| BC-2.04.003–006 | console node | internal/session |
| BC-2.08.001 | console node + sbctl | cmd/sbctl, internal/session |
| BC-2.09.003 | all modes | internal/config |
| BC-2.07.001 | control node | internal/svtnmgmt |

## Concurrency Model

Each daemon uses a single-threaded event loop per logical connection, with a
shared-memory pool for frame buffers. The goroutine model for 1,000 concurrent
sessions is an open question (NFR-004 notes in ARCH-INDEX Open Frontier Questions).
Initial design: one goroutine pair (reader + writer) per connection. Profiling gates
refactoring to an event-loop model before PE phase.

## ADR-010: Terminal Session Backend — Tmux Control Mode Primary, PTY Proxy Fallback (S-3.01)

**Decision:** `internal/tmux` uses tmux control mode (`tmux -C`) as the primary
terminal session backend. PTY proxy mode is the automatic fallback, triggered by
control-mode failure at **any point in the session lifecycle** — both initial connect
failure and mid-session control-mode loss.

**Why tmux control mode is preferred:**
1. Machine-readable event stream: `%output`, `%session-changed`, `%window-add`, and
   `%exit` events arrive as structured lines — no screen-scraping required.
2. Named session addressing: consoles connect by session name (`tmux attach -t NAME`);
   control mode natively enumerates sessions.
3. Session persistence: the tmux server persists sessions independently of the access
   node process. If the access node restarts, it reconnects to the existing tmux
   server rather than losing session state.
4. Fan-out compatibility: `ConsoleSet` fan-out (S-3.02) distributes the event stream
   to multiple consoles from a single tmux control mode connection, avoiding N×tmux
   connections for N attached consoles.

**Why PTY fallback is included:**
- tmux may not be installed on the target host. PTY proxy provides degraded-mode
  operation so the access node does not hard-fail.
- PTY mode does not support named sessions; the fallback is a single-session proxy.
- PTY mode provides functionally equivalent keystroke-to-echo behavior (AC-004) but
  lacks session listing, named session attach, and persistence.

**Fallback semantics:**
- PTY fallback is triggered by any control-mode failure: initial `TmuxControlMode.Attach`
  failure OR mid-session control-mode loss (e.g., tmux server crash, control socket
  disconnect). This matches BC-2.04.001 EC-002 and BC-2.04.002 EC-003 documented
  behavior.
- Once in PTY fallback mode, the session remains in PTY mode for the lifetime of that
  connection. There is no automatic upgrade back to control mode within the same
  connection.
- At next session start (new connection or daemon restart), `TmuxControlMode.Attach`
  is retried before falling back. Control mode is NOT retried mid-session after a
  mid-session failure; that would risk inconsistent state on an already-active PTY session.

**Rejected alternatives:**
- PTY-only mode: loses session naming, persistence, and efficient fan-out.
- screen as alternative: no structured event protocol; screen-scraping required.
  Adds fragile parsing, not a clean boundary.
- libvterm embedding: complex C dependency; not justified for MVP LAN target.
- Initial-connect-only fallback (prior v1.1 decision): rejected as too restrictive.
  BC-2.04.001 EC-002 and BC-2.04.002 EC-003 explicitly describe mid-session
  control-mode failure → PTY fallback transitions. Restricting fallback to initial
  connect only would leave mid-session control-mode loss unhandled (hard failure vs.
  degraded-mode operation). BCs are authoritative; ADR-010 v1.1 was overridden.

**References:** BC-2.04.001 (control mode attach, EC-002 mid-session fallback),
BC-2.04.002 (PTY fallback, EC-003 mid-session fallback), S-3.01 fallback semantics.

## ADR-011: SessionConnector.Frames() — Forwarding Channel Design for Failover-Stable Frame Delivery (S-4.00)

**Context:** `ControlMode.Frames()` and `PTYProxy.Frames()` each return a
`<-chan halfchannel.ChannelFrame` tied to the lifetime of that mode instance.
`SessionConnector` orchestrates ctrl→PTY failover: when control mode drops,
`watchAndFallback` closes the old `ControlMode` and activates `PTYProxy`. Any
goroutine in `cmd/switchboard` that is ranging over `ctrl.Frames()` will see the
channel closed without the new frames from `PTYProxy.Frames()` being delivered.
This is drift W3-R2-M4.

**Decision:** Add `SessionConnector.Frames() <-chan halfchannel.ChannelFrame` as a
new exported method on `*SessionConnector`. The implementation holds an internal
forwarding channel (`sc.frames chan halfchannel.ChannelFrame`, buffered to
`framesBufferSize`) and a single relay goroutine (`sc.forwardFrames`) that is
started by `Connect`. The relay goroutine:

1. Ranges over the current mode's `Frames()` channel, writing each frame to
   `sc.frames` (non-blocking: drops on full, same backpressure semantics as
   `ControlMode`).
2. When the source channel closes (mode switch or shutdown), re-reads
   `sc.activeFrSource()` under `sc.mu` and continues ranging over the new
   source.
3. Exits when `sc.frames` is closed — triggered by `sc.Close()` calling
   `sc.closeForwardFrames.Do(func() { close(sc.frames) })`.

The forwarding channel is closed exactly once via `sync.Once`.

**Why this design over alternatives:**

- *Re-subscribe signal (document only):* Requires `cmd/switchboard` to implement
  its own re-subscription logic, duplicating the fallback-awareness concern in
  every call site. Violates the encapsulation principle — `SessionConnector` already
  knows when the mode switches.
- *Channel-of-channels:* Requires the caller to handle a meta-channel; more complex
  to range over in a simple goroutine and error-prone under Close.
- *Merged/demux approach:* Same result as the forwarding-channel approach but with
  more goroutines and no clear ownership of the close signal.

**Why a forwarding channel is clean:**
- `cmd/switchboard` calls `sc.Frames()` once, gets one channel, and ranges
  over it for the lifetime of the session. No re-subscribe, no mode awareness.
- The forwarding relay goroutine is the single point of mode-switch awareness.
- It is symmetric with `ControlMode.Frames()` and `PTYProxy.Frames()` in
  terms of API shape, so callers do not need to handle a different type.
- Drop semantics are preserved: if `sc.frames` is full, the frame is dropped
  (same policy as the underlying sources).

**Concurrency contract (amended v1.4):**
- `sc.frames` is created in `NewSessionConnector` (buffered).
- The relay goroutine is started in `sc.Connect` after the active mode is
  confirmed. It is registered with `sc.wg`.
- `sc.Close()` closes `sc.frames` via `closeForwardFrames.Do` after `sc.wg.Wait()`.
- No lock is held while writing to `sc.frames` (non-blocking select).
- `sc.activeFrSource()` acquires `sc.mu` to snapshot the current active source;
  the relay goroutine holds no lock during the range-over.

**Relay-drop counter contract (REQUIRED — anti-silent-failure; SOUL.md #4):**

Two distinct drop points exist in the frame delivery pipeline, and BOTH must be
metered:

- **Relay-level drop** (`forwardFrames` in `connector_frames.go`): when
  `sc.frames` is full and the non-blocking select takes the `default:` branch,
  a frame is dropped at the `SessionConnector` relay layer. This drop is
  invisible to `AccessNode.FramesDropped()` (which only counts drops inside
  `ConsoleSet.Deliver`). The relay-level drop MUST be counted by an atomic
  counter on `SessionConnector` — concretely, `sc.relayDropped uint64` (or
  equivalent), incremented via `atomic.AddUint64` on every `default:` branch in
  `forwardFrames`. A new exported method `SessionConnector.RelayDropped() uint64`
  exposes the count. `cmd/switchboard` MUST include `sc.RelayDropped()` alongside
  `accessNode.FramesDropped()` in the AC-006 observability log line, so an
  operator sees the aggregate dropped-frame count (relay + console) in a single
  log entry.

- **ConsoleSet-level drop** (`ConsoleSet.Deliver`): already counted by
  `AccessNode.FramesDropped()` and surfaced by the AC-006 ticker.

The relay-level counter and the ConsoleSet-level counter are intentionally
separate (different layers, different root causes: relay drop = consumer
goroutine not keeping up with the backend; console drop = console peer not
draining). They MUST NOT be merged into one counter. The log line format for
AC-006 MUST report both: `"frames_dropped relay=%d consoles=%d"` (or structured
key-value equivalent).

**Relay busy-spin guard (REQUIRED — CPU safety):**

`forwardFrames` re-reads `activeFrSource()` after each source-channel close.
If `watchAndFallback` has not yet swapped `sc.active`, `activeFrSource()` may
return the same already-closed source, causing `for f := range srcCh` to return
immediately and the outer `for {}` to loop again — a hot spin until the swap
lands. With `factory == nil` (the Wave-3 daemon config), the spin window spans
the full `pty.Connect` call time.

The relay goroutine MUST include `ctx.Done()` coverage in its outer loop. The
correct pattern is to pass a `ctx context.Context` into `forwardFrames` (derived
from the `connectCancel` context created in `Connect`), and select on
`<-ctx.Done()` at the top of the outer loop before re-reading
`activeFrSource()`. This eliminates the busy-spin: if the connector is being
closed while a mode swap is in flight, the relay exits cleanly rather than
spinning. The `forwardFrames` signature becomes
`func (sc *SessionConnector) forwardFrames(ctx context.Context)` and
`startForwardFrames` passes the same `innerCtx` used by `watchAndFallback`.

If a closed-but-unchanged source is returned (same channel as before, already
closed), the relay MUST yield via `runtime.Gosched()` before retrying — this
is a second-line defence against spinning in the narrow window between channel
close and the swap becoming visible under `sc.mu`.

**Daemon sc.Err() drain obligation (REQUIRED — BC-2.04.002 invariant 3):**

`cmd/switchboard`'s `runAccess` MUST start a goroutine that drains `sc.Err()`
for the lifetime of the session. On receiving a non-nil error from `sc.Err()`
(the "both paths down" scenario), the daemon MUST log the error at ERROR level
as E-SYS-002 format and cancel the root context (triggering the PC-2 clean
shutdown path). Rationale: a mid-session double-failure leaves the relay goroutine
running over a dead source with no frames flowing; the only correct action is to
surface the failure (never-silent — SOUL.md #4) and shut down. The goroutine
exits when `sc.Err()` is closed (which happens in `sc.Close()` on normal
shutdown). This goroutine MUST be added to the `sync.WaitGroup` in `runAccess`.

**Impact on internal/tmux package:** One new method `Frames()` on `*SessionConnector`
in `pty_fallback.go` (or a new file `connector_frames.go`). No new exported types.
No new packages. No forbidden import edges introduced. Full buildability on develop.

**References:** W3-R2-M4 (drift), W3-M-2/M-3 (related), ARCH-08 §6.5.1 obligation 4,
BC-2.04.006 (FramesDropped observability, NFR-004), S-4.00 daemon assembly.

---

### ADR-011 Amendment (v1.5): Terminal-Source EOF Contract and runAccess Injection Seam

#### HIGH-A: Terminal PTY-source EOF — Session-Fatal Backend Loss (amended v1.6)

**Problem (original v1.5):** When the access node is in PTY mode, the PTY master
reaches EOF because the proxied shell process exits. `PTYProxy.ioRelay` returns
and closes `p.frames`, but `PTYProxy` does NOT set `p.closed`. The `forwardFrames`
relay hot-spins: `for f := range srcCh` returns immediately; `activeFrSource()`
returns the same live PTYProxy; `srcCh == prevSrcCh` → `runtime.Gosched()` →
repeat. The `ctx.Done()` guard only saves the relay if ctx is already cancelled.

**v1.5 fix and its TOCTOU regression:** v1.5 ruled: discriminate by `sc.InPTYMode()`
when `srcCh == prevSrcCh`. However, `sc.InPTYMode()` acquires `sc.mu` as a
*separate* lock acquisition from the `activeFrSource()` call that produced `src`.
`watchAndFallback` sets `sc.active = sc.pty` and `sc.inPTYMode = true` under a
single `sc.mu` hold. Two separate lock acquisitions can therefore straddle a swap:

> Relay reads src=ctrl (lock 1: activeFrSource) → swap lands (watchAndFallback)
> → Relay reads inPTYMode=true (lock 2: InPTYMode) → misclassifies as terminal EOF
> → sends ErrPTYSourceEOF, prematurely closes sc.frames, breaks EC-002

This is a logical TOCTOU (not a Go data race — both reads are individually
lock-protected). Reproduced as ~20% failure rate on `TestSessionConnectorFramesSurvivesMidSessionFailover`.

**v1.6 decision:** The root cause is reading `{active source, inPTYMode}` under
two separate `sc.mu` acquisitions. The fix is an **atomic snapshot helper** that
returns `{framesSource, <-chan ChannelFrame, bool}` under a single lock hold, so the
source identity, its frames channel, and the mode flag are always mutually consistent.

**(a) PTY EOF is a session-fatal backend loss on sc.Err():** CONFIRMED (unchanged
from v1.5). Terminal PTY EOF without a prior `sc.Close()` call constitutes a
session-fatal backend loss and MUST be surfaced on `sc.Err()` as `ErrPTYSourceEOF`.

**(b) Atomic snapshot requirement (CORE OF v1.6):** The `forwardFrames` relay MUST
obtain `{source, srcCh, inPTYMode}` as a single atomic snapshot under one
`sc.mu` acquisition. A new unexported helper replaces the existing
`activeFrSource()` + separate `sc.InPTYMode()` pattern:

```
// activeSourceSnapshot returns the current active source, its Frames() channel,
// and whether the connector is in PTY mode — all read under a single sc.mu hold.
// Returns (nil, nil, false) if the connector is closed or has no active source.
// The Frames() call on src is made inside the lock because src is the interface
// value captured from sc.active; calling src.Frames() outside the lock would
// allow a swap to replace sc.active between the two reads.
func (sc *SessionConnector) activeSourceSnapshot() (framesSource, <-chan halfchannel.ChannelFrame, bool)
```

Inside `forwardFrames`, replace:
```
src := sc.activeFrSource()
...
srcCh := src.Frames()
```
with:
```
src, srcCh, inPTY := sc.activeSourceSnapshot()
```

The `srcCh == prevSrcCh` + mode discrimination then uses `inPTY` from the same
snapshot — not from a subsequent `sc.InPTYMode()` call.

**(c) Soundness proof for the atomic-snapshot discriminator:**

`watchAndFallback` always sets `sc.active = sc.pty` AND `sc.inPTYMode = true`
inside a single `sc.mu.Lock()` block (verified: pty_fallback.go lines ~732-736
and ~820-823). Therefore, any snapshot taken under one `sc.mu` acquisition will
see one of exactly two self-consistent states:

| Snapshot state | Meaning | `inPTY` | `srcCh` relative to `prevSrcCh` |
|---|---|---|---|
| `{ctrl, ctrl.frames, false}` | Swap not yet landed | false | May equal prevSrcCh if ctrl.frames closed and swap is in flight |
| `{pty, pty.frames, true}` | Swap complete, PTY active | true | ≠ prevSrcCh (pty.frames is a freshly allocated channel, distinct from ctrl.frames) |

The condition `srcCh == prevSrcCh AND inPTY == true` can therefore only be true
when the snapshot is `{pty, pty.frames, true}` AND `pty.frames` is the same
channel as the previous iteration's `srcCh`. This happens exclusively when:
- The PTY swap has already landed (inPTY=true is sound), AND
- The PTY's own `ioRelay` has exited, closing `pty.frames` (same closed channel observed twice).

It CANNOT happen during a still-in-flight failover because a freshly-landed PTY
swap introduces a new `pty.frames` channel object (allocated in `NewPTYProxy`),
which is guaranteed ≠ `prevSrcCh` (which was `ctrl.frames`). So
`srcCh != prevSrcCh` and the terminal-EOF branch is not entered — the relay
starts ranging over the new PTY source normally.

This makes `srcCh == prevSrcCh AND inPTY == true` a **sound and sufficient**
terminal-EOF signal, given the atomic snapshot.

**Critical implementation constraint:** `src.Frames()` MUST be called inside the
`sc.mu` lock, not after releasing it. If `src.Frames()` is called outside the
lock, a swap can replace `sc.active` between the snapshot read and the `Frames()`
call, returning the channel of the NEW source rather than the snapshotted one,
breaking the soundness proof. `PTYProxy.Frames()` and `ControlMode.Frames()` are
pure channel accessors (no locks, no side effects) — calling them inside
`sc.mu` is safe.

**(d) Updated relay termination semantics:** After the inner `for f := range srcCh`
loop exits (srcCh closed):

1. Take an atomic snapshot: `newSrc, newSrcCh, inPTY := sc.activeSourceSnapshot()`
2. If `newSrc == nil` (connector closed): `return`.
3. If `newSrcCh == srcCh AND inPTY` (same closed channel, PTY mode, swap already
   landed): terminal EOF — send `ErrPTYSourceEOF` on `sc.errCh` via `closeErrCh.Do`
   and `return`.
4. If `newSrcCh == srcCh AND !inPTY` (same closed channel, control mode, swap in
   flight): `runtime.Gosched()` and continue the outer loop.
5. Otherwise (`newSrcCh != srcCh`): new source has arrived (mode swap landed).
   Set `prevSrcCh = newSrcCh` and range over `newSrcCh`.

The top-of-loop `srcCh == prevSrcCh` guard (v1.4 busy-spin guard) is REPLACED by
this post-range discrimination; the outer loop structure simplifies to a single
snapshot-and-branch per iteration.

**(e) Error sentinel and taxonomy:** Unchanged from v1.5. `ErrPTYSourceEOF` in
`internal/tmux`; E-SYS-003 cross-reference in error-taxonomy.md; operator-visible
message uses E-SYS-002 format.

**Test obligations (v1.6 — two obligations):**

**Obligation T1 — terminal-EOF bounded-exit (unchanged from v1.5):**
In `internal/tmux/connector_frames_test.go` (or `connector_eof_test.go`):
1. Construct a `SessionConnector` in PTY mode via `WithPTYAllocFunc` injecting a
   fake that returns an `io.Pipe()` pair (master = pipe reader).
2. Call `sc.Connect(ctx)`.
3. Close the write end of the pipe (simulating PTY shell exit) WITHOUT calling
   `sc.Close()`.
4. Assert within ≤100ms (enforced by `t.Cleanup` + `time.AfterFunc` deadline):
   - `sc.Err()` delivers an error satisfying `errors.Is(err, ErrPTYSourceEOF)`.
5. The test MUST FAIL (timeout) if the relay is still hot-spinning. No `runtime.Gosched()`
   trick in the test — deadline enforcement only.
6. After assertion, call `sc.Close()` and assert it returns promptly (no hang).

**Obligation T2 — EC-002 failover under stress (NEW — catches TOCTOU regression):**
In `internal/tmux/connector_test.go` or `connector_frames_test.go`:
1. Run `TestSessionConnectorFramesSurvivesMidSessionFailover` with `-count=50`
   in CI (or equivalent stress loop in the test body using `t.Run` in a loop).
2. The test MUST pass all 50 iterations. Any single failure indicates a TOCTOU
   regression in the `{source, srcCh, inPTYMode}` snapshot logic.
3. Alternatively (or additionally), use a deterministic interleaving hook: inject a
   `swapBarrier` (e.g., a `chan struct{}`) into `activeSourceSnapshot` or
   `watchAndFallback` so that the test can force the relay to be mid-read when a
   swap lands. This is preferred over count-repetition because it deterministically
   exercises the interleaving rather than relying on timing. The hook MUST be a
   no-op in production (`swapBarrier == nil` → skip).
4. Test must verify that after a ctrl→PTY swap, frames from the new PTY source
   are delivered through `sc.Frames()` (no premature close of `sc.frames`).

#### HIGH-B: runAccess Injection Seam

**Problem:** `runAccess` in `cmd/switchboard/access.go` constructs its own
`*tmux.SessionConnector` internally (calling `tmux.New`, `tmux.NewPTYProxy`,
`tmux.NewSessionConnector`). The PC-2.6 exit-code latch branch (`internalFailure`
atomic set by the drain goroutine before `cancel()`, read after `wg.Wait()`) and
the PC-2 clean path are NOT testable end-to-end through `runAccess` with a fake or
stub connector — the test must reconstruct the drain logic in parallel
(tautological for the production branch).

**Decision:** Split `runAccess` into two functions:

1. `runAccess(ctx context.Context, stderr io.Writer) error` — thin wrapper that
   constructs the real `*tmux.SessionConnector` (same construction as current) and
   calls `runAccessWithConnector`.

2. `runAccessWithConnector(ctx context.Context, stderr io.Writer, sc connectorIface) error`
   (or equivalent unexported name) — contains all orchestration logic (wiring
   obligations 1–6, PC-1/PC-2/PC-2.6 lifecycle). `connectorIface` is a minimal
   interface covering `Connect(ctx) error`, `Frames() <-chan halfchannel.ChannelFrame`,
   `Err() <-chan error`, `Close() error`, `RelayDropped() uint64`.

The real `*tmux.SessionConnector` satisfies `connectorIface` by construction. Tests
inject a `fakeConnector` (unexported stub in `access_test.go`) whose `Err()` field
can be pre-populated:
- PC-2.6 test: `fakeConnector.Err()` yields a non-nil error → assert `runAccessWithConnector`
  returns non-nil and writes E-SYS-002 format to the injected stderr writer.
- PC-2 test: `fakeConnector` with no error on `Err()`, context cancelled externally →
  assert `runAccessWithConnector` returns nil (exit 0).

**Seam shape (concrete):**

```go
// connectorIface is the minimal subset of *tmux.SessionConnector used by
// runAccessWithConnector. *tmux.SessionConnector satisfies this interface.
type connectorIface interface {
    Connect(ctx context.Context) error
    Frames() <-chan halfchannel.ChannelFrame
    Err() <-chan error
    Close() error
    RelayDropped() uint64
}
```

`buildAccessComponents` (and by extension `buildRouter`) continue to accept
`*tmux.SessionConnector` concretely (they are called inside `runAccess`, before
the call to `runAccessWithConnector`). The `runAccessWithConnector` signature
accepts `connectorIface`; the router and accessNode are threaded in as separate
parameters constructed by `runAccess`. Alternatively (simpler), `buildAccessComponents`
is called from `runAccess` before the delegation, and the results are passed into
`runAccessWithConnector` alongside the connector interface. The exact signature
is delegated to the implementer, subject to the constraint that test code MUST be
able to call `runAccessWithConnector` directly with a fake connector and assert
PC-2.6/PC-2 outcomes.

**ARCH-08 §6.5.1 note:** The seam is an internal refactoring within
`cmd/switchboard`. It does not introduce new packages or new import edges. It is
noted in ARCH-08 §6.5.1 (see that document) as a testability refinement to
obligation 4.

**Test obligation:** Both PC-2 (clean) and PC-2.6 (mid-session double-failure →
E-SYS-002 → exit 1) MUST be exercised through the real `runAccessWithConnector`
call graph (the production function, not a test-local reimplementation). The fake
connector's `Err()` channel is the injection point.

#### EC-005 Wording Note (Accepted Wave-4 Follow-up)

The EC-005 "forbidden import guard" comment in `access.go` states "CI enforces
this structurally." This overstates current enforcement. The real enforcement
mechanism is: the forbidden packages (`internal/config`, `internal/drain`,
`internal/metrics`) do not exist on develop, so the compiler would refuse the
import. A durable `go list`/`go vet`-based CI assertion that explicitly forbids
these imports (even after the packages are created in Wave 4+) does not yet exist.
Story-writer MUST correct the comment to read: "Build fails because these packages
do not yet exist on develop; a durable go-list CI assertion enforcing this boundary
after the packages land is deferred to Wave 4." This is an accepted follow-up; it
does not block S-W3.04 convergence.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-23 | Initial core services architecture |
| 1.1 | 2026-06-25 | Added ADR-010: tmux control mode primary, PTY proxy fallback (Wave 3 / S-3.01) |
| 1.2 | 2026-06-25 | ADR-010: revised fallback semantics to allow mid-session PTY fallback on control-mode loss (Wave-3 reviewer F-W3-H-003; BCs win: BC-2.04.001 EC-002, BC-2.04.002 EC-003; initial-connect-only restriction deemed too restrictive) |
| 1.3 | 2026-06-27 | Added ADR-011: SessionConnector.Frames() forwarding-channel design (S-4.00 daemon assembly; drift W3-R2-M4 resolution) |
| 1.4 | 2026-06-27 | ADR-011 §Concurrency amended: (1) relay-drop counter contract — `sc.relayDropped` atomic, `RelayDropped()` method, AC-006 log MUST report both relay and console drop counts; (2) relay busy-spin guard — `forwardFrames` receives `ctx` param, selects on `ctx.Done()` in outer loop, calls `runtime.Gosched()` on unchanged-closed-source retry; (3) daemon `sc.Err()` drain obligation — `runAccess` MUST drain Err() in a wg-tracked goroutine, cancel context on non-nil error (E-SYS-002, BC-2.04.002 invariant 3). Per S-W3.04 adversarial convergence adjudication. |
| 1.5 | 2026-06-27 | ADR-011 Amendment v1.5: (HIGH-A) Terminal PTY-source EOF is a session-fatal backend loss — relay MUST detect `srcCh==prevSrcCh` in PTY mode and send `ErrPTYSourceEOF` on `sc.errCh` then exit (no hot-spin); discrimination: PTY mode → fatal; control mode → yield-and-retry (swap in flight). New E-SYS-003 taxonomy entry (canonical display remains E-SYS-002 format). Test obligation: inject pipe-pair PTY, close write end without sc.Close(), assert Err() delivers ErrPTYSourceEOF within 100ms. (HIGH-B) runAccess injection seam — split into `runAccess` (thin ctor wrapper) + `runAccessWithConnector(ctx, stderr, connectorIface)` (all logic); `connectorIface` covers Connect/Frames/Err/Close/RelayDropped; tests inject fake connector for PC-2 and PC-2.6 end-to-end coverage. (MEDIUM) EC-005 wording note: "CI enforces this structurally" overstates enforcement; corrected wording deferred to story-writer. Per S-W3.04 adversarial convergence pass-2. |
| 1.6 | 2026-06-27 | ADR-011 HIGH-A tightened (TOCTOU fix): two separate sc.mu acquisitions (activeFrSource + InPTYMode) can straddle a watchAndFallback swap, causing ~20% EC-002 false-EOF misclassification. Fix: new activeSourceSnapshot() helper reads {src, srcCh=src.Frames(), inPTYMode} under one sc.mu hold; src.Frames() called inside the lock. Soundness proof: watchAndFallback sets sc.active+sc.inPTYMode atomically; freshly-landed PTY introduces pty.frames != prevSrcCh, so srcCh==prevSrcCh AND inPTY=true is sound terminal-EOF signal only. Two test obligations: T1 terminal-EOF bounded-exit (unchanged) + T2 EC-002 stress (-count=50 or deterministic swap-barrier interleaving). Per S-W3.04 adversarial convergence pass-3. |
