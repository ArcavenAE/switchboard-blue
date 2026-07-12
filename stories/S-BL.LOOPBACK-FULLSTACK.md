---
artifact_id: S-BL.LOOPBACK-FULLSTACK
document_type: story
level: ops
story_id: S-BL.LOOPBACK-FULLSTACK
epic_id: E-1
title: "Full-stack loopback testenv extension: tick-driven halfchannel + arq + multipath wiring for VP-042"
status: draft
producer: story-writer
timestamp: 2026-07-12T00:00:00Z
version: "1.0"
phase: 2
epic: E-1
wave: backlog
priority: P2
scope_phase: E
points: 8
inputs:
  - .factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md
  - .factory/specs/verification-properties/VP-042.md
  - .factory/specs/architecture/ARCH-08-dependency-graph.md
input-hash: "efb29b5"
traces_to: .factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md
behavioral_contracts:
  - BC-2.01.001   # timeslice clock fires every tick regardless of data availability
  - BC-2.01.002   # empty-tick frame semantics
  - BC-2.02.001   # duplicate-and-race dispatch
  - BC-2.02.002   # endpoint checksum-only dedup
  - BC-2.02.005   # downstream ARQ (piggybacked ACK/SACK, TLPKTDROP)
verification_properties:
  - VP-042   # keystroke-to-echo p99 <= 100ms — harness delivery only; lock flip is a separate subsequent act, see Non-Goals / Forward Obligation
target_module: internal/testenv
estimated_days: null
assumption_validations: []
risk_mitigations: []   # placement note's 5 Risks are note-local (not ASM/R-registry IDs); addressed via AC-001/AC-009/AC-010/AC-011 instead of registry references
bc_traces:
  - BC-2.01.001   # timeslice clock fires every tick regardless of data availability
  - BC-2.01.002   # empty-tick frame semantics
  - BC-2.02.001   # duplicate-and-race dispatch
  - BC-2.02.002   # endpoint checksum-only dedup
  - BC-2.02.005   # downstream ARQ (piggybacked ACK/SACK, TLPKTDROP)
vp_traces:
  - VP-042   # keystroke-to-echo p99 <= 100ms — harness delivery only; lock flip is a separate subsequent act, see Non-Goals / Forward Obligation
subsystems: [transport-layer, quality-observability, session-networking]
architecture_modules:
  - internal/testenv
  - internal/halfchannel
  - internal/arq
  - internal/multipath
  - internal/paths
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: []   # S-BL.TESTENV already MERGED (PR #110, 62e38d3) — this story extends its NewLoopback/LoopbackEnv API; it is not blocked on that story, it builds on shipped code
blocks: []
inputDocuments:
  - '.factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md'   # v1.0 — BINDING. Q1-Q8 + Non-Goals + Package Impact + 5 Risks. Where this story and the note diverge, the note governs.
  - '.factory/specs/verification-properties/VP-042.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'   # v2.13 — this story's merge finalizes the PROSPECTIVE pos-23 import-set amendment
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
  - '.factory/stories/S-BL.TESTENV.md'
  - '.factory/stories/S-BL.PE-RECEIVE-LOOP.md'   # precedent for the Env.wg/closeCh ticker-goroutine idiom (Q6) and story-writer conventions (grep-resolved symbols, no line-number citations)
acceptance_criteria_count: 13
backlog_origin:
  source: architect design note
  adjudication: "Human disposition, 2026-07-12: author now, deliver later — status draft, unscheduled. Not an adversarial-pass or PO-adjudication origin; commissioned directly to answer the open design questions VP-042.md v1.3's own history flagged (\"lock deferred to a testenv-integrated measurement post S-BL.TESTENV\") and to finalize ARCH-08 v2.13's PROSPECTIVE registration."
  drift_items_consumed: []
---

# S-BL.LOOPBACK-FULLSTACK: Full-Stack Loopback Testenv Extension for VP-042

> **Status note:** This story is authored to full spec but is deliberately **draft / unscheduled** per human disposition (2026-07-12) — "author now, deliver later." It has not been through story-writer's normal wave-planning promotion or an adversarial spec-review cycle. Treat AC-001 (the `arq.OnAck` sign-off gate) as blocking regardless of when this story is picked up — do not let "the story file already exists" substitute for that gate having actually been cleared.

## Narrative

- **As a** verification engineer trying to lock VP-042 (keystroke-to-echo p99 ≤ 100ms)
- **I want** `internal/testenv`'s `NewLoopback`/`LoopbackEnv` extended from a same-goroutine
  `DeliverFrame` shortcut into a tick-driven, protocol-accurate loopback stack spanning
  `internal/halfchannel` + `internal/arq` + `internal/multipath` + `internal/paths`
- **So that** VP-042's benchmark measures the real round-trip path (tick cadence, duplicate-and-race
  dispatch, endpoint dedup, downstream ARQ bookkeeping) instead of an in-process echo shortcut that
  bypasses all of it, and the harness can be run once to produce honest evidence for a future
  `verification_lock` decision

## Context

`S-BL.TESTENV` (merged PR #110, `62e38d3`) shipped `internal/testenv` including `NewLoopback` and
`LoopbackEnv`, but `NewLoopback` (`testenv.go:383`) discards its `LoopbackConfig` and calls
`newEnv(ctx, b, 1)` — `LoopbackConfig.TickIntervalUpstream`/`TickIntervalDownstream` (`testenv.go:364`)
are dead fields. `Env.SendKeystroke` (`testenv.go:744`) does not go through
`session.AccessNode.SendKeystroke`/`KeystrokeSink` at all; it directly calls `sh.access.DeliverFrame(hdr)`,
synthesizing a downstream fan-out frame under the name "SendKeystroke." There is no tick scheduler
anywhere in the path. `S-BL.BENCH` (merged PR #109, `cd67394`) recorded VP-042 as **adopted-partial**:
an honest lower-bound-only measurement (in-process loopback echo p99 ~0.002ms vs the 100ms limit) with
a declared divergence — the inline echo path bypasses `arq`/`multipath`/tick-scheduling entirely. VP-042
v1.3's own changelog states the lock is "deferred to a testenv-integrated measurement post S-BL.TESTENV."

This story is that testenv-integrated measurement. It is scoped and designed entirely by the architect
design note listed as this story's binding input
(`.factory/decisions/S-BL.LOOPBACK-FULLSTACK-placement-note.md` v1.0) — **story-writer's job here is
transcription, not re-derivation.** Where this story and the placement note appear to diverge, the note
governs; where this story and VP-042.md's older proof-harness skeleton diverge (the skeleton's two-call
`env.SendKeystroke`/`env.WaitForEcho` shape vs. this story's token-based `RoundTrip` API), the placement
note's shape is binding — the skeleton predates the discovery (Q5) that a token is required to fix a
distinct accumulation bug in `Env.CollectFrames`.

**Also discharged by this story:** ARCH-08 v2.13's PROSPECTIVE amendment to `internal/testenv`'s §6.5
pos-23 import set — `{admission, drain, frame, outerassembler, session, upstreamdial}` →
`{admission, arq, drain, frame, halfchannel, multipath, outerassembler, paths, session, upstreamdial}`
— becomes final, machine-verified (`go list`), at this story's merge, per the same protocol used for
every prior testenv import-set change (v2.5, v2.8, v2.11).

## Story-Sizing Rationale (points: 8, architect range 5–8)

The placement note's own estimate is 5–8 points, broken down as: tick-driving (Q6) is low-risk and
small — a direct copy of an idiom already used twice in `testenv.go` (`AttachConsole`/`AttachProbe`)
and twice more in `cmd/switchboard/access.go`; multipath wiring (Q3, Q7) is low-risk — small,
well-tested pure APIs, a few lines of synthetic path construction; the round-trip-token API (Q5)
touches the WIP bench test and VP-042.md's skeleton, small but real fan-out. **The ARQ wiring (Q4) is
the size and risk driver** — it commits to a call contract (`arq.OnAck`'s `ackSeq` semantics) that has
no existing production precedent to copy, and the note itself flags that commitment as needing
architect/adversarial sign-off before an implementer treats it as settled (Risk 1 below).

Story-writer selects the **upper end of the range (8)**, not the midpoint, for three reasons beyond the
placement note's own text: (1) AC-001 is a hard pre-implementation gate, not just a risk note — it adds
real process latency before `dev-story` can properly start, which the note's code-size-only estimate
doesn't price in; (2) four of the five Risks (not just Risk 1) resolve into their own gating or
decision-bearing ACs (AC-009, AC-010, AC-011) rather than being absorbed silently into the main
implementation tasks; (3) the WIP bench cross-reference (Package Impact, `internal/bench` row) is real
fan-out into a file on a different branch (`fix/vp-042-testenv-integrated-bench`), which is coordination
overhead the tick/multipath/token estimate doesn't include.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|-------------|
| Timeslice clock fires on every tick regardless of data availability | BC-2.01.001 | VP-042 Source Contract; placement note Q3, Q6 | TO DISCHARGE (harness-scope) — upstream/downstream ticker goroutines call `HalfChannel.Tick()` on a fixed schedule per `cfg.TickIntervalUpstream`/`TickIntervalDownstream`, independent of `Enqueue` timing; `NewLoopback` validates both intervals against `halfchannel.MinTickInterval`/`MaxTickInterval` |
| Empty-tick frame semantics | BC-2.01.002 | placement note Q1, Non-Goals | TO DISCHARGE (partial, harness-scope) — `Tick()` produces an empty-tick frame on schedule when nothing is enqueued; this story does NOT wire-dispatch empty ticks over multipath (Non-Goals) — a harness-scope boundary, not a production behavior change |
| Duplicate-and-race: same frame sent on two fastest paths simultaneously | BC-2.02.001 | VP-042 Source Contract; placement note Q3, Q7 | TO DISCHARGE — `multipath.Send` dispatches every payload over both synthetic `paths.RankedPath`s per direction; `deliverUpstream`/`deliverDownstream` is called once per selected path |
| Endpoint checksum-only dedup | BC-2.02.002 | placement note frontmatter; Q3 | TO DISCHARGE — `multipath.Receive` returns `ErrDuplicate` on the second-arriving copy of a duplicate-and-raced frame; discarded before reaching `accessNode`/`arqClient` |
| Downstream ARQ (piggybacked ACK/SACK, TLPKTDROP) | BC-2.02.005 | placement note Q1, Q4 | TO DISCHARGE (downstream leg only — upstream ARQ is explicitly out of scope per Q1/ARCH-03) — every downstream tick's data frame passes through `arqServer.EnqueueSend`; every post-dedup downstream arrival calls `arqClient.OnAck` per the Q4 call-contract, **gated on AC-001 sign-off** |
| Keystroke-to-echo p99 ≤ 100ms | VP-042 | VP-042.md | HARNESS DELIVERED, NOT LOCKED — this story ships the measurement harness and runs it once for evidence; the `verification_lock` flip is a separate subsequent act (see Forward Obligation) |

---

## Design Constraints

The following subsections transcribe the placement note's binding decisions (Q2–Q8). They are not
re-derived here; where a code sketch is reproduced, it is the note's sketch, not a new one.

### Loopback Driver Ownership and Dedicated Shard (Q2)

**Binding (per placement note Q2).**

A new unexported `loopbackDriver` type lives inside `internal/testenv`, owned by `LoopbackEnv`.
`SendKeystroke`/`WaitForEcho`/`CreateSession` are **new methods on `*LoopbackEnv`**, not on `*Env`.
`LoopbackEnv` is `struct { Env *Env }` — a named field, not Go embedding (confirmed: the existing WIP
bench test does `env := lb.Env; env.CreateSession(b)`, never `lb.CreateSession(b)`) — so new
`*LoopbackEnv` methods do not collide with or shadow `*Env`'s method set.

`Env.SendKeystroke`/`Env.CollectFrames` are **not** extended in place: those methods back 10 other VPs
via generic SVTN-shard fan-out semantics that none of them asked to become tick-driven or
round-trip-tagged. `NewLoopback` keeps calling `newEnv(ctx, b, 1)` (so `lb.Env.Close()`/generic surface
stay available, harmless if unused); `LoopbackEnv` additionally constructs and owns a `*loopbackDriver`
with its own dedicated session/shard.

The driver needs a **dedicated shard**, not `env.defaultShard`: `newShard` hardcodes
`session.WithKeystrokeSink(session.NoOpSink{})`, and `session.AccessNode` has no `SetSink` — the
`KeystrokeSink` is fixed at construction via functional option, by design (a mutable-sink escape hatch
would weaken that invariant for every other `AccessNode` consumer, not just testenv). The loopback
driver instead builds its own `Publisher`/`SessionAuth`/`AccessNode` triple — identical in shape to
`newShard`, but with `WithKeystrokeSink(loopbackSink)` from the start, where `loopbackSink` is the
driver's own echo-generating sink (Q4). This duplication is isolated to the loopback path; it does not
touch `newShard` or any other VP's shard, and it does not add a `SetSink` escape hatch to production
`session.AccessNode`.

### Upstream Flow: Keystroke → Server Delivery (Q3)

**Binding (per placement note Q3).**

```
LoopbackEnv.SendKeystroke(t, sessionID, key)
    mints RoundTrip{id: driver.rtSeq.Add(1)}; registers a completion channel
    under that id in driver.pending, guarded by driver.mu
    payload := append([]byte(key), encodeRTID(id)...)   // 8-byte BE suffix
    ↓
driver.upstreamHC.Enqueue(payload)   // pure, non-blocking — returns to caller
                                      // immediately; SendKeystroke does NOT
                                      // block on delivery (BC-2.01.001 requires
                                      // the tick to fire on its own schedule
                                      // regardless of enqueue timing)
    ↓
[async] upstream ticker, every cfg.TickIntervalUpstream:
    f := driver.upstreamHC.Tick()
    if f.FrameType == frame.FrameTypeData {
        driver.upstreamMP.Send(toMPFrame(f), driver.deliverUpstream)
    }
    // empty ticks are produced (BC-2.01.002) but not wire-dispatched (Non-Goals)
    ↓
driver.deliverUpstream(pathID, mpFrame) error   // called once per selected
    path (up to 2, duplicate-and-race) — the SAME callback for both, since
    both loopback paths terminate in this one process
    ↓
driver.upstreamMP.Receive(mpFrame)   // endpoint checksum dedup
    ErrDuplicate on second-arriving copy → discard, return nil
    ↓
driver.accessNode.SendKeystroke(loopbackConsoleKey, sessionName, mpFrame.Payload)
    ↓
loopbackSink.SendInput(payload) error   // Q4
```

`SendFunc` is called from inside the ticker goroutine, not spawned into its own goroutine per path —
`multipath.Send`'s doc states `fn` is called without holding any internal lock, so real work in `fn` is
safe; with zero synthetic added latency (Non-Goals: no real network) there is no concurrency benefit to
spawning, and running both calls sequentially avoids a class of out-of-order dedup-cache-insertion races
that a fully-faithful network simulation would have to reckon with but this design deliberately does not
model.

### Downstream Flow: Echo Generation → Round-Trip Completion (Q4) — GATED, see AC-001

**Binding, subject to AC-001 sign-off (per placement note Q4).**

`loopbackSink.SendInput` — the `KeystrokeSink` injected into the driver's dedicated `AccessNode` — is
the echo generator:

```go
func (s *loopbackSink) SendInput(payload []byte) error {
    return s.driver.downstreamHC.Enqueue(payload)   // echoes the FULL payload
}                                                     // verbatim, including the
                                                       // embedded RT-ID — the sink
                                                       // does not need to understand
                                                       // the correlation scheme; it
                                                       // just echoes bytes, like real
                                                       // tmux would
```

`SendInput` is called while `AccessNode` holds `sinkMu` ("must not call back into `AccessNode` under any
lock"); `Enqueue` only touches the downstream `HalfChannel`'s own pending queue, never calling back into
`AccessNode`, so this is safe by construction — and it is the correct modeling of BC-2.01.001: the echo
is queued, not delivered synchronously; the downstream ticker decides when it actually goes out.

```
[async] downstream ticker, every cfg.TickIntervalDownstream:
    f := driver.downstreamHC.Tick()
    if f.FrameType == frame.FrameTypeData {
        driver.arqServer.EnqueueSend(f.ChanSeq, f.Payload, time.Now())
        driver.downstreamMP.Send(toMPFrame(f), driver.deliverDownstream)
    }
    ↓
driver.deliverDownstream(pathID, mpFrame) error
    ↓
driver.downstreamMP.Receive(mpFrame)   // endpoint dedup; first arrival only
    ↓
delivered, err := driver.arqClient.OnAck(mpFrame.ChanSeq(), zeroSACK)
    // ackSeq = this frame's own ChanSeq (locally-derived from arrival, not
    // peer-supplied); SACK bitmap all-zero (no loss simulated)
    ↓
for each payload in delivered:
    id := decodeRTID(payload)
    driver.mu.Lock(); ch := driver.pending[id]; delete(driver.pending, id); driver.mu.Unlock()
    if ch != nil { ch <- frameFor(payload) }   // unblocks WaitForEcho
```

**`arq.OnAck` call-contract — no existing production precedent.** No production code calls `OnAck`
today; `internal/arqsend` (the only production consumer of `*arq.ARQ`) only exercises the sender-side
subset (`PayloadForInFlight`/`EnqueueSend`/`RemoveInFlight`). This design is the **first proposed call
site for `OnAck`** in the codebase. The convention proposed — `ackSeq` = the highest downstream `ChanSeq`
this receiver has now observed in order (locally-derived from arrival, not a peer-supplied value), called
once per received (post-dedup) downstream frame with that frame's own `ChanSeq` — is internally
consistent given a single downstream producer emitting strictly increasing `ChanSeq` values with no
synthetic loss/reordering, and exercises `OnAck`'s real window-validation (`RULING-003`/
`ErrAckOutOfWindow`) and delivery-pointer bookkeeping on every sample, but it is a **proposal**, not a
verified contract — see AC-001.

`GapsToRetransmit`/`TLPKTDROP` are deliberately **not** called — there is no simulated loss, so
`arqServer.inFlight` never accumulates a real gap; wiring an active poll for a condition that structurally
cannot occur in this harness would be dead code (Non-Goals).

### RoundTrip Token API — Fixing the CollectFrames Accumulation Short-Circuit (Q5)

**Binding (per placement note Q5).**

`Env.CollectFrames` (`testenv.go:758`) and `Conn`/`Console.CollectFrames` poll an **accumulating**
slice — `Env.WaitForEcho` returns as soon as the slice is non-empty, so a second concurrent or leftover
round trip's frame satisfies a `WaitForEcho` call that isn't waiting for it. This is a distinct bug from
the tick/protocol gap and is fixed independently of it, by sidestepping it entirely rather than patching
`CollectFrames`'s polling loop:

```go
// RoundTrip identifies one SendKeystroke → echo round trip in a loopback
// environment. Returned by LoopbackEnv.SendKeystroke; consumed exactly once
// by LoopbackEnv.WaitForEcho.
type RoundTrip struct {
    id   uint64
    done chan frame.OuterHeader // buffered 1; written by the downstream
                                 // ticker goroutine on delivery
}

// SendKeystroke drives a keystroke through the full loopback protocol stack
// (Q3) and returns a token identifying this specific round trip.
func (lb *LoopbackEnv) SendKeystroke(t testing.TB, sessionID SessionID, key string) RoundTrip

// WaitForEcho blocks until the echo tagged with rt arrives, or timeout
// elapses (fails t via t.Fatalf/b.Errorf on timeout). Unlike Env.WaitForEcho,
// which returns as soon as ANY frame is buffered on the session, this reads
// only rt's own completion channel — a concurrent or stale round trip's
// frame cannot satisfy it.
func (lb *LoopbackEnv) WaitForEcho(t testing.TB, rt RoundTrip, timeout time.Duration)
```

No shared growing slice is in this path at all. `Env.CollectFrames`/`Conn`/`Console.CollectFrames` are
unchanged — their accumulation semantics remain correct for the VPs that use them. The correlation ID
rides in the payload bytes (8-byte big-endian suffix, `encodeRTID`/`decodeRTID`, package-private), not in
`frame.OuterHeader` (which is a fixed 44-byte wire layout with no spare field) — this also means
`loopbackSink` doesn't need to know about correlation at all; it just echoes bytes, matching how a real
`KeystrokeSink` (tmux) works.

### Goroutine / Lifecycle Plan (Q6)

**Binding (per placement note Q6).**

Two ticker goroutines (upstream, downstream), registered on the **existing** `Env.wg`/`Env.closeCh` — no
new `WaitGroup` or close channel. `Env` already has `wg sync.WaitGroup`, `closeCh chan struct{}`,
`closeOnce sync.Once`; `Env.Close()` already does `closeOnce.Do(func() { close(closeCh); wg.Wait() })`,
registered via `t.Cleanup(e.Close)` in `newEnv`. `AttachConsole` and `AttachProbe` already start
goroutines this exact way (`wg.Add(1)` before `go func() { defer wg.Done(); select { case <-closeCh:
return; ... } }()`) — the loopback tickers use the identical pattern rather than inventing a second
lifecycle mechanism:

```go
func startLoopbackTicker(
    env *Env,
    hc *halfchannel.HalfChannel,
    interval time.Duration,
    onTick func(halfchannel.ChannelFrame),
) {
    env.wg.Add(1)
    go func() {
        defer env.wg.Done()
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            select {
            case <-env.closeCh:
                return
            case <-ticker.C:
                onTick(hc.Tick())
            }
        }
    }()
}
```

This is also the same shape as `cmd/switchboard/access.go`'s `startSweepTicker`/
`startFramesDroppedTicker` — the production idiom for "ticker + WaitGroup + cancellation-channel." No
new `Close()` method is needed on `LoopbackEnv`; `t.Cleanup(env.Close)` (already registered by `newEnv`)
tears everything down, and `wg.Wait()` blocks until both ticker goroutines have observed `closeCh` and
returned — deterministic, no leaked goroutines, matching the existing `AttachConsole`/`AttachProbe`
guarantee.

`NewLoopback` must validate `cfg.TickIntervalUpstream`/`TickIntervalDownstream` against
`halfchannel.MinTickInterval`/`MaxTickInterval` (5ms–50ms) and `b.Fatalf` on an out-of-bounds value,
matching the existing fail-loud convention (`t.Fatalf` on illegal construction throughout this file, e.g.
`NewWithRouters`). **VP-042's own `downstreamInterval` (50ms) sits exactly at `MaxTickInterval`** — legal,
but the validation site needs a comment noting this, since it's the boundary case (AC-002).

### Synthetic Path Construction (Q7)

**Binding (per placement note Q7).**

Two `paths.RankedPath`s per direction (4 total), each backed by `paths.NewPathTracker(1.0, 0.125)` — no
`OnProbe` calls needed. `paths.NewPathTracker` sets `active: true` at construction, so a fresh tracker is
immediately eligible for `Rank` without any probe history.

```go
func newLoopbackPaths() []paths.RankedPath {
    return []paths.RankedPath{
        {ID: 1, Tracker: paths.NewPathTracker(1.0, 0.125)},
        {ID: 2, Tracker: paths.NewPathTracker(1.0, 0.125)},
    }
}
```

`multipath.NewMultipath` requires `[]paths.RankedPath` at construction, and `multipath.Send` internally
calls `paths.Rank` on every call — so testenv must import `internal/paths` **directly**, a Go-imposed
transitive requirement (referencing an exported type from an indirectly-imported package requires a
direct import), not a scope expansion story-writer is choosing. ARCH-08 v2.13 already includes `paths` at
position 11 for exactly this reason. Two `*multipath.Multipath` instances are constructed — one per
direction (`upstreamMP`, `downstreamMP`) — each combining the pathSet used by whichever side is the
sender for that direction and the `recvDedup` cache used by whichever side is the receiver.

### No New Package (Q8)

**Binding (per placement note Q8).** All of this lands inside `internal/testenv` (existing position 23,
test-only composition root). ARCH-08 §6.4's new-package protocol does not apply — this is an import-set
expansion of an existing package, the same class of change as v2.6/v2.8/v2.11.

---

## Acceptance Criteria

**AC-001 is a gate. It must be resolved before implementation begins on any other AC in this list.**

### AC-001 (GATE — pre-implementation sign-off; traces to Q4 / Risk 1)

The `arq.OnAck` call-contract proposed in Q4 (`ackSeq` = the locally-observed frame's own `ChanSeq`,
zero SACK in the no-loss happy path) has no existing production call site to copy — this design is the
first proposed caller of `OnAck` in the codebase. Before `dev-story` begins implementing the downstream
flow (Q4), **one of the following must be produced and attached to this story**:

(a) an architect placement-note addendum explicitly confirming the `OnAck` call contract, or
(b) a fast adversarial pass on the placement note specifically targeting Q4.

**Test:** none — this is a process gate, not a code test. `dev-story` MUST refuse to start Q4-dependent
implementation work (downstream ticker's `EnqueueSend`/`OnAck` wiring) without evidence of (a) or (b)
attached. Getting this wrong does not break VP-042's measured number (the happy path is forgiving) but
would misinform whatever future story reuses `OnAck` for a real loss-injection path.

### AC-002 (traces to BC-2.01.001; Q6)

`NewLoopback` validates `cfg.TickIntervalUpstream` and `cfg.TickIntervalDownstream` against
`halfchannel.MinTickInterval`/`MaxTickInterval` and `b.Fatalf`s on an out-of-bounds value. The validation
site carries a comment noting that VP-042's own `downstreamInterval` (50ms) sits exactly at
`MaxTickInterval` — legal, boundary case. Both ticker goroutines fire `HalfChannel.Tick()` on their
configured schedule independent of `Enqueue` timing (a keystroke enqueued between ticks waits for the
next tick, never triggers an out-of-band delivery).

**Test:** `TestNewLoopback_RejectsOutOfBoundsTickInterval` (table-driven: below `MinTickInterval`, above
`MaxTickInterval`, exactly at `MaxTickInterval` = legal); `TestLoopbackDriver_TicksFireOnSchedule`
(enqueue between ticks, assert delivery does not precede the next tick boundary).

### AC-003 (traces to BC-2.01.002; Non-Goals)

The upstream and downstream tickers call `Tick()` every interval regardless of whether data is enqueued
(empty ticks are produced, satisfying BC-2.01.002), but an empty-tick `ChannelFrame` (`FrameType !=
frame.FrameTypeData`) is never passed to `multipath.Send` — only data frames are wire-dispatched. This
is a harness-scope boundary (Non-Goals), not a production behavior change.

**Test:** `TestLoopbackDriver_EmptyTicksNotDispatched` — assert `Tick()` is called on every interval
(instrument via a tick-count hook) while `multipath.Send` call count only increments on data-bearing
ticks.

### AC-004 (traces to BC-2.02.001; Q3, Q7)

`upstreamMP`/`downstreamMP` are each constructed via `multipath.NewMultipath` with the two synthetic
`paths.RankedPath`s from `newLoopbackPaths()`. A single `Enqueue`d payload, once ticked, is dispatched by
`multipath.Send` to both paths (duplicate-and-race); `deliverUpstream`/`deliverDownstream` is invoked
once per selected path.

**Test:** `TestLoopbackDriver_DuplicateAndRaceDispatch` — instrument `deliverUpstream` (or
`deliverDownstream`) with a call-count hook, assert it fires exactly twice per ticked data frame (once
per synthetic path).

### AC-005 (traces to BC-2.02.002; Q3, Q4)

The second-arriving copy of a duplicate-and-raced frame is discarded by `multipath.Receive`'s endpoint
checksum dedup (`ErrDuplicate`) before reaching `driver.accessNode`/`driver.arqClient` — i.e., exactly
one of the two `deliverUpstream`/`deliverDownstream` calls per AC-004 results in forward progress
(`accessNode.SendKeystroke` call or `arqClient.OnAck` call), not two.

**Test:** `TestLoopbackDriver_EndpointDedupDiscardsSecondArrival` — assert `accessNode.SendKeystroke`
(upstream) and `arqClient.OnAck` (downstream) are each called exactly once per ticked data frame despite
two `deliverUpstream`/`deliverDownstream` invocations.

### AC-006 (traces to BC-2.02.005; Q4 — gated by AC-001)

Every downstream tick's data frame is passed to `driver.arqServer.EnqueueSend(f.ChanSeq, f.Payload,
time.Now())` before dispatch. Every post-dedup downstream arrival calls `driver.arqClient.OnAck` with
that frame's own `ChanSeq` and an all-zero SACK bitmap, per the AC-001-signed-off call contract.
`GapsToRetransmit`/`TLPKTDROP` are not called on any schedule (Non-Goals).

**Test:** `TestLoopbackDriver_DownstreamARQWiring` — assert `EnqueueSend` is called once per downstream
data tick and `OnAck` is called once per post-dedup downstream arrival with the frame's own `ChanSeq`;
assert `GapsToRetransmit`/`TLPKTDROP` are never invoked in this harness.

### AC-007 (traces to Q2 — dedicated shard)

`loopbackDriver` constructs its own `Publisher`/`SessionAuth`/`AccessNode` triple at construction time,
with `session.WithKeystrokeSink(loopbackSink)` set from the start. `env.defaultShard` is untouched — the
loopback driver never mutates it, and no `SetSink` method is added to production `session.AccessNode`.

**Test:** `TestLoopbackDriver_DedicatedShard_NoDefaultShardMutation` — assert `env.defaultShard`'s
`KeystrokeSink` remains `session.NoOpSink{}` after a `LoopbackEnv` is constructed and exercised;
`TestSessionAccessNode_NoSetSinkMethod` — a compile-time/reflection guard confirming
`session.AccessNode` gained no new sink-mutation method.

### AC-008 (traces to Q5 — RoundTrip token API)

`LoopbackEnv.SendKeystroke` returns a `RoundTrip` token. `LoopbackEnv.WaitForEcho` consumes exactly one
token, reading only that token's own completion channel — it never reads `Env.CollectFrames`'
accumulating buffer. A concurrent or stale round trip's frame cannot satisfy a `WaitForEcho` call for a
different token.

**Test:** `TestLoopbackEnv_WaitForEcho_DoesNotConsumeOtherRoundTrips` — issue two concurrent
`SendKeystroke` calls, `WaitForEcho` on the second token first, assert it does not return early on the
first token's frame; `TestLoopbackEnv_WaitForEcho_IgnoresStaleCollectFramesBuffer` — pre-populate
`Env.CollectFrames`' buffer with an unrelated frame before issuing a round trip, assert `WaitForEcho`
still waits for its own token.

### AC-009 (traces to Risk 3 — `RoundTrip.done` buffering and no-leak/no-block)

`RoundTrip.done` is buffered 1. On a `WaitForEcho` timeout, `driver.pending`'s entry for that round trip
is still deleted by the downstream ticker's eventual (or already-happened) send, and the buffered send
into `done` does not block the ticker goroutine even if nobody ever reads from `done` again.

**Test:** `TestLoopbackEnv_WaitForEcho_TimeoutThenLateArrival_NoLeak` — issue `SendKeystroke`, call
`WaitForEcho` with a timeout shorter than the configured tick cadence so it times out, then allow the
echo to arrive; assert (a) the ticker goroutine's send into `done` does not block/deadlock, (b)
`driver.pending` no longer holds the entry after the late arrival is processed, (c) no goroutine leak is
detected (`t.Cleanup` + goroutine-count check, mirroring the `Env.Close()`/`wg.Wait()` leak-check
convention used elsewhere in this package).

### AC-010 (traces to Risk 2 — `PathTracker.IsActive()` initial-state assertion)

An explicit, cheap assertion/test confirms `paths.NewPathTracker(1.0, 0.125).IsActive()` returns `true`
immediately at construction, with no `OnProbe` call — insurance against a future `internal/paths` change
silently breaking the loopback's path activation and producing a confusing downstream failure (e.g.
`multipath.Send` silently excluding a path from `Rank`) instead of a clear, localized one.

**Test:** `TestNewLoopbackPaths_TrackersActiveWithoutProbe` — construct `newLoopbackPaths()`, assert
`IsActive()` is `true` on every returned `paths.RankedPath.Tracker` with zero `OnProbe` calls made.

### AC-011 / DECISION (traces to Risk 4 — pending-map growth safeguard)

**Decision (story-writer, per the placement note's invitation to make this call): adopt the cheap
safeguard.** If `WaitForEcho` is never called for a `RoundTrip` (a test bug), `driver.pending` would
otherwise accumulate permanently until `Env.Close()`. Rather than leaving this as a docstring-only
warning, `LoopbackEnv` construction registers a `t.Cleanup` that asserts `driver.pending` is empty at
environment teardown — this is a `testing.TB`-only assertion (no runtime cost added to the driver's hot
path) that turns a silent, hard-to-diagnose test bug into a loud, localized failure at the point of the
bug's own test, rather than surfacing later as an unrelated flake or resource-leak symptom. This mirrors
the existing `t.Cleanup(env.Close)` idiom already used throughout `internal/testenv`.

**Test:** `TestLoopbackEnv_Cleanup_AssertsPendingMapEmpty` — construct a `LoopbackEnv`, issue a
`SendKeystroke` without a matching `WaitForEcho`, assert the `t.Cleanup`-registered check fails loudly
(verified via a sub-test harness that captures the assertion rather than fatal-ing the outer test);
companion `TestLoopbackEnv_Cleanup_PassesWhenPendingDrained` — normal usage (every `SendKeystroke`
followed by a `WaitForEcho`) leaves the map empty at teardown with no assertion failure.

### AC-012 (traces to Q6 — goroutine lifecycle)

Both ticker goroutines (upstream, downstream) register on the existing `Env.wg`/`Env.closeCh` — no new
`WaitGroup` or close channel is introduced. `t.Cleanup(env.Close)` (already registered by `newEnv`) tears
both goroutines down deterministically; `wg.Wait()` blocks until both have observed `closeCh` and
returned. No `Close()` method is added to `LoopbackEnv`.

**Test:** `TestLoopbackEnv_TickerGoroutines_JoinOnClose` — construct a `LoopbackEnv`, call
`lb.Env.Close()` (or trigger the registered cleanup), assert both ticker goroutines have exited via a
`sync.WaitGroup`-based join-confirmation, with a bounded timeout guarding against a hang (matching the
existing `AttachConsole`/`AttachProbe` leak-check pattern in this package).

### AC-013 (traces to Package Impact — WIP bench cross-reference)

`internal/bench/keystroke_echo_testenv_bench_test.go` on branch `fix/vp-042-testenv-integrated-bench` is
updated from its current two-call `env.SendKeystroke`/`env.WaitForEcho` shape (the VP-042.md skeleton
shape, now superseded — see Context) to the token-based shape: `rt := lb.SendKeystroke(b, sessionID,
"x"); lb.WaitForEcho(b, rt, 500*time.Millisecond)`. The package comment's "lower bound only" framing
(inherited from S-BL.BENCH's honest-partial-evidence disclosure) is retired once this full stack lands,
since the divergence it disclosed (bypassing arq/multipath/tick-scheduling) no longer exists.

**Test:** no new test — this AC is a modification of an existing benchmark file. Verification is that
`go build ./internal/bench/...` succeeds against the new `LoopbackEnv` API and `just bench` runs the
updated benchmark to completion, producing a `p99_rtt_ms` metric.

---

## Non-Goals

Transcribed from the placement note. This story does NOT implement:

- **Real network I/O or cross-process operation.** Both synthetic paths are zero-added-latency,
  in-process function calls — no sockets, no serialization to wire bytes (no
  `outerassembler.Assemble`/`DecodeChannelHeader` round trip); `multipath.Frame`/`halfchannel.ChannelFrame`
  are passed as Go structs, not encoded bytes. Byte-level wire-format coverage in a loopback harness would
  be a separate, additive future story.
- **Simulated packet loss, retransmission, or TLPKTDROP.** `GapsToRetransmit` and `TLPKTDROP` are not
  called on any schedule. `internal/arqsend` and `internal/outerassembler`-based real retransmit dispatch
  are not added to testenv's import set — they would only be needed for a loss-injection follow-on.
  `internal/arq`'s own pure-core unit tests already cover the reorder/gap/TLPKTDROP state machine; this
  benchmark's job is realistic tick-driven happy-path latency, not re-proving ARQ correctness.
- **`internal/replay` / upstream idempotent-window fidelity.** Out of scope per Q1 and ARCH-03 — ARQ is
  documented as downstream-only; upstream keystroke reliability is `internal/replay`'s job in production,
  and this benchmark has no simulated loss, so replay's absence changes nothing observable.
- **Empty-tick wire dispatch.** Empty ticks are produced by `Tick()` (BC-2.01.002 compliance) but not
  dispatched over multipath in this harness — they carry no round-trip token and would not change the
  measured property.
- **Changing `Env.SendKeystroke`/`Env.CollectFrames`/`Env.WaitForEcho`.** These remain exactly as they are
  for the 10 other VPs that use them.
- **A VP-042 `verification_lock` flip.** See Forward Obligation below — this story delivers and runs the
  harness once for evidence; locking VP-042 is a separate, subsequent PO/architect act.

---

## Architecture Mapping

| Component | Package | New / Modified | Notes |
|-----------|---------|-----------------|-------|
| `loopbackDriver` (type) | `internal/testenv` | New | Owns dedicated `Publisher`/`SessionAuth`/`AccessNode`, both `Multipath` instances, both `HalfChannel`s, the `arq.ARQ` client/server pair, `pending` map |
| `RoundTrip` (type) | `internal/testenv` | New | Opaque outside the package; carries `id` + buffered-1 `done` channel |
| `loopbackSink` (type) | `internal/testenv` | New | Implements `session.KeystrokeSink`; echoes payload verbatim into `downstreamHC.Enqueue` |
| `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession` | `internal/testenv` | New (methods on `*LoopbackEnv`) | Do not collide with `*Env`'s method set (named field, not embedding) |
| `startLoopbackTicker` (helper) | `internal/testenv` | New | Registers on `Env.wg`/`Env.closeCh`; identical shape to `AttachConsole`/`AttachProbe` |
| `newLoopbackPaths` (helper) | `internal/testenv` | New | Two `paths.RankedPath`s per direction |
| `NewLoopback` | `internal/testenv` | Modified | Wires halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig`; adds Min/MaxTickInterval validation |
| `halfchannel.HalfChannel` | `internal/halfchannel` | Read-only consumer | `New`, `Tick`, `Enqueue` |
| `arq.ARQ` | `internal/arq` | Read-only consumer | `New`, `EnqueueSend`, `OnAck` — first production-adjacent `OnAck` call site (AC-001 gate) |
| `multipath.Multipath` | `internal/multipath` | Read-only consumer | `NewMultipath`, `Send`, `Receive` |
| `paths.PathTracker`/`RankedPath` | `internal/paths` | Read-only consumer | `NewPathTracker`, `RankedPath` |
| `keystroke_echo_testenv_bench_test.go` | `internal/bench` | Modified | Token-based two-call shape (AC-013) |

## Edge Cases

| Edge Case | Handling |
|-----------|----------|
| `WaitForEcho` times out, echo arrives later | `RoundTrip.done` buffered 1; downstream ticker's send never blocks even if nobody reads it; `driver.pending` entry is still deleted (AC-009) |
| `WaitForEcho` never called for a `RoundTrip` (test bug) | `driver.pending` would otherwise accumulate until `Env.Close()`; `t.Cleanup` asserts the map is empty at teardown (AC-011) |
| Duplicate frame arrival (same payload, two synthetic paths) | `multipath.Receive` returns `ErrDuplicate` on the second arrival — discarded before `accessNode`/`arqClient` (AC-005) |
| Tick interval exactly at `MaxTickInterval` (50ms) | Legal — VP-042's own `downstreamInterval` sits exactly here; validation site carries a boundary comment (AC-002) |
| Fresh `paths.RankedPath` with no probe history | `NewPathTracker` defaults `active: true`; `Rank()` considers it eligible with zero `OnProbe` calls (AC-010) |
| `OnAck` window-validation / `ErrAckOutOfWindow` path | Not exercised by this harness's no-loss happy path (single producer, strictly increasing `ChanSeq`); a future loss-injection story would exercise it (Non-Goals) |
| Two concurrent `SendKeystroke`/`WaitForEcho` round trips | Each has its own `RoundTrip.id` and `done` channel; AC-008 guarantees no cross-talk |

## Purity Classification

| Component | Classification | Rationale |
|-----------|-----------------|-----------|
| `loopbackDriver`, ticker goroutines, `RoundTrip` | Effectful (test infrastructure) | Goroutines, tickers, channel synchronization — same class as existing `AttachConsole`/`AttachProbe` |
| `halfchannel`, `arq`, `multipath`, `paths` (as consumed) | Pure-core, UNCHANGED | testenv becomes an effectful DRIVER of their `Tick()`/`OnAck()`/`Send()` entry points; their own purity boundary is unchanged by this edge (ARCH-08 v2.13 rationale) |

## Package Impact Summary

(Transcribed from the placement note.)

| Package | Change | ARCH-08 §6.4 required? |
|---------|--------|------------------------|
| `internal/testenv` | New `loopbackDriver` type; `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession`/`RoundTrip`; `NewLoopback` wires halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig` | No (existing package) — import-set expansion requires the §6.4-equivalent pre-code registration already done in ARCH-08 v2.13 |
| `internal/halfchannel` | None — read-only consumer (`New`, `Tick`, `Enqueue`) | No |
| `internal/arq` | None — read-only consumer (`New`, `EnqueueSend`, `OnAck`); first production-adjacent call site for `OnAck` (AC-001) | No |
| `internal/multipath` | None — read-only consumer (`NewMultipath`, `Send`, `Receive`) | No |
| `internal/paths` | None — read-only consumer (`NewPathTracker`, `RankedPath`) | No |
| `internal/bench` | `keystroke_echo_testenv_bench_test.go` (branch `fix/vp-042-testenv-integrated-bench`) updated to the token-based two-call shape; "lower bound only" framing retired (AC-013) | No |

**No new `internal/` package.** ARCH-08 registration is the import-set amendment already applied
(v2.13, DRAFT/PROSPECTIVE) — it becomes final at this story's merge per the same machine-verification
protocol used for every prior testenv import-set change (v2.5, v2.8, v2.11).

---

## Token Budget Estimate (forecast)

| Component | Est. tokens |
|-----------|-------------|
| This story spec | ~9k |
| Placement note (binding input, full read required) | ~6k |
| Referenced production code (`testenv.go`, `halfchannel.go`, `arq.go`, `multipath.go`, `paths.go` — read-only consumer surfaces) | ~7k |
| Test infrastructure context (existing `testenv` patterns, WIP bench test) | ~3k |
| **Total implementing-agent context** | **~25k — well within 20–30% of a 200k context window. No story split required.** |

## Tasks (MANDATORY)

1. [ ] **GATE:** Confirm AC-001 is resolved — architect placement-note addendum OR a fast adversarial
   pass on Q4 signs off the `arq.OnAck` call-contract. Do not proceed past this task until evidence of
   sign-off is attached to this story.
2. [ ] Implement `loopbackDriver` inside `internal/testenv` with its own `Publisher`/`SessionAuth`/
   `AccessNode` triple constructed via `session.WithKeystrokeSink(loopbackSink)` (Q2, AC-007).
3. [ ] Implement `RoundTrip` + `driver.pending map[uint64]chan frame.OuterHeader` (buffered-1 channels)
   + `rtSeq atomic.Uint64` (Q5, AC-008, AC-009).
4. [ ] Implement `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession` on `*LoopbackEnv` (Q2, Q5).
5. [ ] Implement upstream flow: `Enqueue` → upstream ticker `Tick()` → `upstreamMP.Send` →
   `deliverUpstream` → `upstreamMP.Receive` dedup → `accessNode.SendKeystroke` → `loopbackSink.SendInput`
   (Q3, AC-004, AC-005).
6. [ ] Implement downstream flow: `loopbackSink.SendInput` → `downstreamHC.Enqueue` → downstream ticker
   `Tick()` → `arqServer.EnqueueSend` + `downstreamMP.Send` → `deliverDownstream` →
   `downstreamMP.Receive` dedup → `arqClient.OnAck` → `driver.pending` lookup → completion send (Q4,
   AC-006 — depends on Task 1 gate).
7. [ ] Implement `NewLoopback` config validation against `halfchannel.MinTickInterval`/
   `MaxTickInterval`, `b.Fatalf` on violation, with the 50ms-boundary comment (Q6, AC-002).
8. [ ] Register both ticker goroutines on the existing `Env.wg`/`Env.closeCh` via `startLoopbackTicker`
   (Q6, AC-012) — no new `WaitGroup`/close channel.
9. [ ] Implement synthetic path construction — two `paths.RankedPath`s per direction backed by
   `paths.NewPathTracker(1.0, 0.125)`, plus the `PathTracker.IsActive()` initial-state assertion (Q7,
   AC-010).
10. [ ] Wire the `driver.pending`-empty `t.Cleanup` safeguard (AC-011); update
    `keystroke_echo_testenv_bench_test.go` on `fix/vp-042-testenv-integrated-bench` to the token-based
    shape (AC-013).
11. [ ] Run the harness once manually to produce VP-042 evidence; hand off to PO/architect for the
    `verification_lock` decision — **this is explicitly NOT this story's Definition of Done; see Forward
    Obligation.**

## Previous Story Intelligence (MANDATORY)

| Predecessor | Lesson carried forward |
|-------------|--------------------------|
| S-BL.TESTENV (merged PR #110, `62e38d3`) | Ships the `NewLoopback`/`LoopbackConfig`/`LoopbackEnv` skeleton this story extends. `LoopbackEnv` is a named field (`struct { Env *Env }`), not embedding — confirmed via the existing WIP bench call shape `env := lb.Env; env.CreateSession(b)`. |
| S-BL.BENCH (merged PR #109, `cd67394`) | VP-042 partial evidence already recorded (in-process loopback echo p99 ~0.002ms) is an honest LOWER-BOUND-ONLY measurement — declared divergence: the inline echo path bypasses arq/multipath/tick-scheduling. This story removes that divergence. |
| S-BL.PE-RECEIVE-LOOP (merged PR #118, `e940fc2`) | Established the `env.wg`/`env.closeCh`-registered ticker-goroutine idiom as house convention for test goroutines needing deterministic teardown — `startLoopbackTicker` (Q6) reuses the identical shape. Also: every new symbol claim must be grep-resolved or marked "(new — defined by this story)"; line-number citations are forbidden in story prose — use mechanism-anchor descriptions (both followed in this story). |
| VP-042.md v1.3 | The VP's own proof-harness skeleton (`env.SendKeystroke`/`env.WaitForEcho`, no token) is directionally correct but superseded by this story's `RoundTrip`-token two-call shape (Q5) — the skeleton predates the discovery that a token is required to fix `CollectFrames`'s accumulation short-circuit. |

## Architecture Compliance Rules (MANDATORY)

| Rule | Compliance |
|------|------------|
| ARCH-08 §6.5 pos-23 import set | This story's merge FINALIZES the PROSPECTIVE v2.13 amendment; implementer runs the §6.4-equivalent machine-verification (`go list`) at merge per the testenv v2.5/v2.8/v2.11 precedent, flipping the ARCH-08 entry from PROSPECTIVE to verified. This story does not itself edit ARCH-08 prose (owned by architect). |
| §6.2 forbidden-edge check | No forbidden edge — `halfchannel`/`arq`/`multipath`/`paths` gain no new import; `testenv` remains a leaf (imported by nothing outside `_test` files). |
| `session.AccessNode` fixed-sink invariant | Preserved — `KeystrokeSink` is injected once at construction via `WithKeystrokeSink(loopbackSink)` on the driver's own `AccessNode`; no `SetSink` escape hatch is added to production `session.AccessNode` (Q2, AC-007). |
| `Env.SendKeystroke`/`Env.CollectFrames`/`Env.WaitForEcho` | Unchanged — the 10 other VPs depending on their generic SVTN-shard fan-out semantics are unaffected (Non-Goals). |

## Library & Framework Requirements (MANDATORY)

Stdlib only: `testing`, `time` (ticker), `sync`/`sync/atomic`. Internal packages: `internal/halfchannel`,
`internal/arq`, `internal/multipath`, `internal/paths` (all already vendored in-module, read-only
consumption). No new external dependency.

## File Structure Requirements (MANDATORY)

| File | Change |
|------|--------|
| `internal/testenv/loopback.go` (new — implementer's choice of filename, or inline in `testenv.go`) | `loopbackDriver`, `RoundTrip`, `loopbackSink`, `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession`, `startLoopbackTicker`, `newLoopbackPaths` |
| `internal/testenv/testenv.go` | `NewLoopback` modified to wire halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig` |
| `internal/bench/keystroke_echo_testenv_bench_test.go` (branch `fix/vp-042-testenv-integrated-bench`) | Modified — token-based two-call shape (AC-013); "lower bound only" comment retired |
| `.factory/specs/architecture/ARCH-08-dependency-graph.md` | §6.5 pos-23 row: PROSPECTIVE → machine-verified at merge (architect/implementer act at merge time, not a story-writer edit) |

---

## Delivery Plan Note — POL-005

Any adversarial or evaluation dispatch for this story (per-story pass, wave-gate Perimeter-2, or any
other evaluation dispatch) **MUST embed the POL-005 (`adversary-dispatch-integrity`, HIGH) verification
tuple** in the dispatch prompt — `{repo path, branch, expected HEAD SHA at dispatch time, artifact IDs +
versions under review}` — per `.factory/policies.yaml` POL-005 (registered 2026-07-12). The dispatched
agent's first action must verify its observed `git rev-parse HEAD` and artifact versions against the
tuple before proceeding; on mismatch, it must ABORT the pass and report the divergence as the pass
result rather than reviewing stale state.

## Forward Obligation — VP-042 `verification_lock` (explicitly NOT part of this story)

This story delivers the harness and, per AC-013/Task 11, is run once manually to produce evidence for
VP-042.md's changelog. **Flipping `verification_lock: false → true` in VP-042.md's frontmatter is a
separate, subsequent PO/architect act** — it requires explicit sign-off distinct from "the harness
compiles and its own tests pass." Do not treat this story's merge, by itself, as a VP-042 lock event.
This mirrors how VP-042's own history table already distinguishes "audited"/"partial evidence" entries
from a lock flip.
