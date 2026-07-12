---
artifact_id: S-BL.LOOPBACK-FULLSTACK-placement-note
document_type: architect-design-note
story_id: S-BL.LOOPBACK-FULLSTACK
title: "Full-stack loopback testenv extension: tick-driven halfchannel + arq + multipath wiring for VP-042"
status: draft
producer: architect
timestamp: 2026-07-12T00:00:00Z
version: "1.0"
bc_traces:
  - BC-2.01.001   # timeslice clock fires every tick regardless of data availability
  - BC-2.01.002   # empty-tick frame semantics
  - BC-2.02.001   # duplicate-and-race dispatch
  - BC-2.02.002   # endpoint checksum-only dedup
  - BC-2.02.005   # downstream ARQ (piggybacked ACK/SACK, TLPKTDROP)
vp_traces:
  - VP-042        # keystroke-to-echo p99 <= 100ms
architecture_modules:
  - internal/testenv
  - internal/halfchannel
  - internal/arq
  - internal/multipath
  - internal/paths
related_documents:
  - .factory/specs/verification-properties/VP-042.md
  - .factory/specs/architecture/ARCH-08-dependency-graph.md
  - .factory/specs/architecture/ARCH-03-routing-engine.md
---

# Architect Design Note: Full-Stack Loopback for VP-042
## Story: S-BL.LOOPBACK-FULLSTACK

This note answers the design questions needed to unblock story-writer for the
extension of `internal/testenv`'s `NewLoopback`/`LoopbackEnv` API from a
same-goroutine `DeliverFrame` shortcut into a tick-driven, protocol-accurate
loopback stack that can finally lock VP-042. All file:line anchors refer to
develop `f73676d`.

## Ground Truth (established by formal-verifier and this session's reading)

- `testenv.NewLoopback` (`internal/testenv/testenv.go:383`) discards its
  `LoopbackConfig` and calls `newEnv(ctx, b, 1)`. `LoopbackConfig.TickIntervalUpstream`
  / `TickIntervalDownstream` (`testenv.go:364`) are dead fields.
- `Env.SendKeystroke` (`testenv.go:744`) does **not** go through
  `session.AccessNode.SendKeystroke`/`KeystrokeSink` at all — it directly calls
  `sh.access.DeliverFrame(hdr)`, i.e. it synthesizes a *downstream* fan-out frame
  under the name "SendKeystroke". `AccessNode` is goroutine-free
  (`internal/session/upstream.go:128`); there is no tick scheduler anywhere in
  the path.
- ARCH-08 position 22 (test-only composition root, now 23) imports
  `{admission, drain, frame, outerassembler, session, upstreamdial}`. It does
  not import `halfchannel`/`arq`/`multipath`, so nothing in testenv drives
  `halfchannel.Tick()`.
- `Env.CollectFrames` (`testenv.go:758`) and `Conn`/`Console.CollectFrames`
  (`testenv.go:86`, `:161`) poll an **accumulating** slice — `WaitForEcho`
  (`testenv.go:1057`) returns as soon as the slice is non-empty, so a second
  concurrent or leftover round trip's frame satisfies a `WaitForEcho` call that
  isn't waiting for it. This is a distinct bug from the tick/protocol gap and
  must be fixed independently of it (Q5 below).
- ARCH-03 §Downstream ARQ / §Upstream Idempotent Replay / §F-023 (read-only
  console ACK) pin the real protocol asymmetry: **upstream keystroke delivery
  uses `internal/replay` (idempotent replay window), not ARQ** — "keystroke
  loss is self-healing without explicit ARQ" (ARCH-03 line 159). ARQ applies
  only to the **downstream** direction (access node = sender, console =
  receiver); the console's SACK bitmap acknowledging downstream frames rides
  on the console's own upstream channel header (F-023), not a separate ACK
  channel.
- No production code calls `arq.OnAck` today. `internal/arqsend` (the only
  production consumer of `*arq.ARQ`) only exercises the sender-side subset
  (`PayloadForInFlight`/`EnqueueSend`/`RemoveInFlight`). This design is
  therefore the **first proposed call site for `OnAck`** in the codebase — see
  Q4 for the specific call contract this note commits to, and the Risks
  section for why that commitment needs architect/adversarial sign-off before
  implementation, not just story-writer transcription.

---

## Q1 — Does this expand `internal/replay` scope too, per the team's request phrasing?

**Decision: No. Scope is exactly `{halfchannel, arq, multipath}` (+ the
transitively-required `internal/paths`), matching ARCH-08 v2.13. `internal/replay`
is explicitly out of scope.**

The dispatch request describes routing keystrokes "upstream through halfchannel
framing + arq + multipath duplicate-and-race." Read literally that could imply
ARQ on the upstream leg. ARCH-03 is unambiguous that it does not: upstream
keystroke reliability is `internal/replay`'s job (self-healing sliding replay
window), and ARQ is documented as downstream-only in both its package doc
(`internal/arq/arq.go:1`) and ARCH-03's "Downstream ARQ (internal/arq,
BC-2.02.005)" section. VP-042's own Source Contract cites BC-2.01.001 (tick)
and BC-2.02.001 (duplicate-and-race) — not BC-2.02.004 (replay) or BC-2.02.005
(ARQ) — as the two BCs it exists to verify; ARQ enters only because the
downstream leg of the round trip is architecturally required to carry it.

Consequence: this design puts `arq` on the **downstream** half-channel only.
Upstream keystroke delivery is halfchannel + multipath, with no reliability
layer beyond multipath's endpoint dedup — architecturally correct (loss would
be self-healing via replay in production; this benchmark has no simulated loss,
so replay's absence changes nothing observable). If the team wants full
BC-2.02.004 fidelity in the harness later, `internal/replay` (position 13,
also below 23) is a lawful, independent follow-on addition — it does not
change this design's shape.

---

## Q2 — Where does the tick-driving live: a new type, or methods bolted onto `Env`?

**Decision: a new unexported `loopbackDriver` type inside `internal/testenv`,
owned by `LoopbackEnv`, with `SendKeystroke`/`WaitForEcho`/`CreateSession` as
NEW methods on `*LoopbackEnv` — not on `*Env`.**

`LoopbackEnv` is currently `struct { Env *Env }` — a **named field**, not
Go anonymous embedding (confirmed: the existing WIP bench test does
`env := lb.Env; env.CreateSession(b)`, never `lb.CreateSession(b)`; if `Env`
were embedded, both forms would resolve). This means new methods on
`*LoopbackEnv` do not collide with or shadow `*Env`'s methods — they are
simply a separate method set reached via `lb.Foo(...)` instead of
`lb.Env.Foo(...)`.

**Why not extend `Env.SendKeystroke`/`Env.CollectFrames` in place:** those
methods back 10 other VPs (VP-033, 034, 036, 037, 038, 039, 040, 046 per the
package doc) via SVTN-shard fan-out semantics that are deliberately generic
("did a frame arrive on this session") — not round-trip-specific. Rewiring
them to be tick-driven and round-trip-tagged would be a blast-radius change
across every other testenv consumer for a semantics none of them asked for.
`LoopbackEnv` getting its own narrow, protocol-accurate method set is the
minimal-diff option: `NewLoopback` keeps calling `newEnv(ctx, b, 1)` (so
`lb.Env.Close()`/generic surface stay available, harmless if unused), and
`LoopbackEnv` additionally constructs and owns a `*loopbackDriver` with its
own dedicated session/shard.

**Why the loopback driver needs its own dedicated shard, not `env.defaultShard`:**
`newShard` (`testenv.go:534`) hardcodes
`session.WithKeystrokeSink(session.NoOpSink{})`. `session.AccessNode` has no
`SetSink` — the `KeystrokeSink` is fixed at construction via functional
option (`internal/session/upstream.go:104`), by design (production callers
inject a stable sink once; a mutable-sink escape hatch would weaken that
invariant for every other consumer of `AccessNode`, not just testenv). Rather
than add that escape hatch to production `session.AccessNode`, the loopback
driver builds its own `Publisher`/`SessionAuth`/`AccessNode` triple —
identical in shape to `newShard`, but with `WithKeystrokeSink(loopbackSink)`
from the start, where `loopbackSink` is the driver's own echo-generating sink
(Q4). This is a few lines of duplication against `newShard`, isolated to the
loopback path; it does not touch `newShard` or any other VP's shard.

---

## Q3 — Upstream flow: keystroke → server delivery

```
LoopbackEnv.SendKeystroke(t, sessionID, key)
    │  mints RoundTrip{id: driver.rtSeq.Add(1)}; registers a completion
    │  channel under that id in driver.pending (map[uint64]chan frame.OuterHeader,
    │  guarded by driver.mu)
    │  payload := append([]byte(key), encodeRTID(id)...)   // 8-byte BE suffix
    ▼
driver.upstreamHC.Enqueue(payload)      // pure, non-blocking, halfchannel.go:143
    │  (returns to caller immediately — SendKeystroke does NOT block on
    │   delivery; this is deliberate: it models "the client queued a
    │   keystroke," not "the keystroke arrived." BC-2.01.001 requires the
    │   tick to fire on its own schedule regardless of enqueue timing.)
    ▼
[async] upstream ticker goroutine (Q6), every cfg.TickIntervalUpstream:
    f := driver.upstreamHC.Tick()                          // halfchannel.go:117
    if f.FrameType == frame.FrameTypeData {                // has payload
        driver.upstreamMP.Send(toMPFrame(f), driver.deliverUpstream)
    }
    // empty ticks are produced (BC-2.01.002) but not wire-dispatched —
    // see Non-Goals.
    ▼
driver.deliverUpstream(pathID, mpFrame) error   // called once per selected
    │  path (up to 2, duplicate-and-race, multipath.go:244) — the SAME
    │  callback for both, since both loopback paths terminate in this
    │  one process
    ▼
driver.upstreamMP.Receive(mpFrame)     // endpoint checksum dedup, multipath.go:318
    │  ErrDuplicate on the second-arriving copy → discard, return nil
    │  nil (first arrival) → continue
    ▼
driver.accessNode.SendKeystroke(loopbackConsoleKey, sessionName, mpFrame.Payload)
    │  internal/session/upstream.go:276 — authorizer check, sinkMu-serialized,
    │  synchronous call into the injected KeystrokeSink
    ▼
loopbackSink.SendInput(payload) error   // Q4
```

**Why `SendFunc` is called from inside the ticker goroutine, not spawned into
its own goroutine per path:** `multipath.Send`'s doc explicitly says `fn` is
called "without holding any internal lock" — it is safe to do real work
in `fn`. Both loopback paths have zero synthetic added latency (see
Non-Goals: no real network), so there's no concurrency benefit to spawning;
running both calls sequentially in the ticker goroutine is simpler and avoids
a class of races (out-of-order dedup-cache insertion) that a fully-faithful
network simulation would have to reckon with but this design deliberately does
not model.

---

## Q4 — Downstream flow: echo generation → client delivery → round-trip completion

`loopbackSink.SendInput` (the `KeystrokeSink` injected into the loopback
driver's dedicated `AccessNode`, per Q2) is the echo generator:

```go
func (s *loopbackSink) SendInput(payload []byte) error {
    return s.driver.downstreamHC.Enqueue(payload)   // echoes the FULL
}                                                     // payload verbatim,
                                                       // including the
                                                       // embedded RT-ID —
                                                       // the sink does not
                                                       // need to understand
                                                       // the correlation
                                                       // scheme; it just
                                                       // echoes bytes, like
                                                       // real tmux would.
```

`SendInput` is called while `AccessNode` holds `sinkMu`
(`internal/session/upstream.go:63`: "must not call back into AccessNode under
any lock"). `Enqueue` only touches the downstream `HalfChannel`'s own pending
queue — it never calls back into `AccessNode` — so this is safe by construction,
and it is also the *correct* modeling of BC-2.01.001: the echo is queued, not
delivered synchronously; the downstream ticker decides when it actually goes
out.

```
[async] downstream ticker goroutine, every cfg.TickIntervalDownstream:
    f := driver.downstreamHC.Tick()
    if f.FrameType == frame.FrameTypeData {
        driver.arqServer.EnqueueSend(f.ChanSeq, f.Payload, time.Now())  // arq.go:339
        driver.downstreamMP.Send(toMPFrame(f), driver.deliverDownstream)
    }
    ▼
driver.deliverDownstream(pathID, mpFrame) error
    ▼
driver.downstreamMP.Receive(mpFrame)    // endpoint dedup; first arrival only
    ▼
delivered, err := driver.arqClient.OnAck(mpFrame.ChanSeq(), zeroSACK)  // arq.go:201
    │  ackSeq = this frame's own ChanSeq (see rationale below); SACK bitmap
    │  is all-zero (nothing out-of-order to report — no loss is simulated)
    ▼
for each payload in delivered:
    id := decodeRTID(payload)
    driver.mu.Lock(); ch := driver.pending[id]; delete(driver.pending, id); driver.mu.Unlock()
    if ch != nil { ch <- frameFor(payload) }   // unblocks WaitForEcho
```

**`arqClient.OnAck` call-contract decision (flagged for architect sign-off,
see Risks):** no production code calls `OnAck` yet, so there is no existing
call-site convention to match. This design treats `OnAck`'s `ackSeq` argument
as "the highest downstream `ChanSeq` this receiver has now observed in order"
— i.e. **locally-derived from arrival**, not a peer-supplied value — and
calls it once per received (post-dedup) downstream frame with that frame's
own `ChanSeq`. Because the loopback has a single downstream producer emitting
strictly increasing `ChanSeq` values one tick at a time, and no synthetic
loss/reordering (Non-Goals), this call is equivalent to "advance cumulative
delivery by exactly one" on every call — it never needs `OnAck`'s SACK-buffer
or gap-handling paths in the happy path, but it does exercise `OnAck`'s real
window-validation (`RULING-003`/`ErrAckOutOfWindow`, arq.go:220) and
delivery-pointer bookkeeping on every sample.

**Why not call `GapsToRetransmit`/`TLPKTDROP` at all:** see Non-Goals — there
is no simulated loss, so `arqServer.inFlight` never accumulates a real gap.
Wiring an active poll for a condition that structurally cannot occur in this
harness would be dead code exercised by nothing. `EnqueueSend` alone still
gives an honest measurement of the sender-side bookkeeping cost (map insert +
deadline computation) that production incurs on every downstream tick.

---

## Q5 — Fixing the `CollectFrames` accumulation short-circuit

**Decision: a new `RoundTrip` token type, opaque outside the package, carrying
a private completion channel. `LoopbackEnv.SendKeystroke` returns one;
`LoopbackEnv.WaitForEcho` consumes one. Neither reads the shared/accumulating
frame buffer that `Env.CollectFrames` uses.**

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
// elapses (fails t via t.Fatalf/b.Errorf on timeout — mirrors the existing
// Env.WaitForEcho failure convention). Unlike Env.WaitForEcho, which returns
// as soon as ANY frame is buffered on the session (correct for VP-033/034/
//036/039's "did anything arrive" semantics but wrong for VP-042's per-sample
// semantics), this reads only rt's own completion channel — a concurrent or
// stale round trip's frame cannot satisfy it.
func (lb *LoopbackEnv) WaitForEcho(t testing.TB, rt RoundTrip, timeout time.Duration)
```

This sidesteps the accumulation bug entirely rather than patching
`CollectFrames`'s polling loop — no shared growing slice is in this path at
all. `Env.CollectFrames`/`Conn`/`Console.CollectFrames` are unchanged; their
accumulation semantics are correct for the VPs that use them (probes and
consoles legitimately want "everything received so far").

The correlation ID rides in the payload bytes (8-byte big-endian suffix,
`encodeRTID`/`decodeRTID` — trivial, package-private), not in
`frame.OuterHeader` — the outer header is a fixed 44-byte wire layout
(`internal/frame/frame.go:66`) with no spare field, so payload-embedding is
the only option that doesn't touch the wire format. This also means the
`loopbackSink` (Q4) doesn't need to know about correlation at all — it just
echoes bytes, matching how a real KeystrokeSink (tmux) works.

---

## Q6 — Goroutine / lifecycle plan

**Decision: two ticker goroutines (upstream, downstream), registered on the
*existing* `Env.wg`/`Env.closeCh` — no new WaitGroup or close-channel.**

`Env` already has `wg sync.WaitGroup`, `closeCh chan struct{}`, `closeOnce
sync.Once` (`testenv.go:434-436`), and `Env.Close()` already does
`closeOnce.Do(func() { close(closeCh); wg.Wait() })` (`testenv.go:561`),
registered via `t.Cleanup(e.Close)` in `newEnv` (`testenv.go:528`). Both
`AttachConsole` and `AttachProbe` already start goroutines this exact way
(`wg.Add(1)` before `go func() { defer wg.Done(); select { case <-closeCh:
return; ...} }()`, `testenv.go:664-680`). The loopback ticker goroutines
should use the identical pattern — same file, same package, same idiom
already used twice in this exact struct — rather than invent a second
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

This is also the same shape as `cmd/switchboard/access.go:460`
(`startSweepTicker`) and `:500` (`startFramesDroppedTicker`) — the
production idiom for "ticker + WaitGroup + cancellation-channel" per
go.md rule 12's spirit and the S-4.00 wg-join clarification (ARCH-08
§6.5.1 obligations 3/6). No new Close() method is needed on `LoopbackEnv`;
`b.Cleanup(env.Close)` (already registered by `newEnv`) tears everything
down, and `wg.Wait()` blocks until both ticker goroutines have observed
`closeCh` and returned — deterministic, no leaked goroutines, matching the
existing `AttachConsole`/`AttachProbe` guarantee.

`NewLoopback` must validate `cfg.TickIntervalUpstream`/`TickIntervalDownstream`
against `halfchannel.MinTickInterval`/`MaxTickInterval` (5ms–50ms,
`halfchannel.go:44`) and `b.Fatalf` on an out-of-bounds value — matching the
existing fail-loud convention (`t.Fatalf` on illegal construction throughout
this file, e.g. `NewWithRouters` at `testenv.go:454`). Note VP-042's own
`downstreamInterval` (50ms) sits exactly at `MaxTickInterval` — legal, but
worth a comment at the validation site since it's the boundary case.

---

## Q7 — Synthetic path construction for `multipath.NewMultipath`

**Decision: two `paths.RankedPath`s per direction (4 total), each backed by
a `paths.NewPathTracker(initialRTTMS, alpha)` — no `OnProbe` calls needed.**

`paths.NewPathTracker` (`internal/paths/paths.go:115`) sets `active: true`
at construction — a fresh tracker is immediately eligible for `Rank`
(`paths.go:407`) without any probe history. `multipath.NewMultipath` requires
`[]paths.RankedPath` at construction (`multipath.go:215`), and
`multipath.Send` internally calls `paths.Rank` on every call (`multipath.go:252`)
— so testenv must import `internal/paths` **directly** to reference
`paths.RankedPath`/`paths.NewPathTracker`, even though the team's dispatch
only named `{halfchannel, arq, multipath}`. This is a Go-imposed transitive
requirement (referencing an exported type from an indirectly-imported package
requires a direct import), not a scope expansion I'm choosing — ARCH-08 v2.13
(already amended) includes `paths` at position 11 for exactly this reason.

```go
func newLoopbackPaths() []paths.RankedPath {
    return []paths.RankedPath{
        {ID: 1, Tracker: paths.NewPathTracker(1.0, 0.125)},
        {ID: 2, Tracker: paths.NewPathTracker(1.0, 0.125)},
    }
}
```

Two `*multipath.Multipath` instances are constructed — one per direction
(`upstreamMP`, `downstreamMP`) — each combining the pathSet used by
whichever side is the sender for that direction, and the `recvDedup` cache
used by whichever side is the receiver for that direction. This is the
minimal shape: one process, one loopback, no cross-process boundary means
there's no reason to split sender-state and receiver-state into separate
instances per endpoint.

---

## Q8 — New `internal/` package required?

**No new package.** All of this lands inside `internal/testenv` (existing
position 23, test-only composition root). ARCH-08 §6.4's new-package
protocol does not apply — this is an import-set expansion of an existing
package, the same class of change as v2.6 (`upstreamdial` pre-code
registration) and v2.8/v2.11 (testenv import-set corrections), already
amended into ARCH-08 v2.13 (this session).

---

## What VP-042's Benchmark Looks Like Against This

```go
func BenchmarkKeystrokeToEcho_P99(b *testing.B) {
    ctx := context.Background()
    lb := testenv.NewLoopback(ctx, b, testenv.LoopbackConfig{
        TickIntervalUpstream:   10 * time.Millisecond,
        TickIntervalDownstream: 50 * time.Millisecond,
    })
    sessionID := lb.CreateSession(b)

    latencies := make([]time.Duration, 0, 500)
    b.ResetTimer()
    for i := 0; i < 500; i++ {
        start := time.Now()
        rt := lb.SendKeystroke(b, sessionID, "x")
        lb.WaitForEcho(b, rt, 500*time.Millisecond)
        latencies = append(latencies, time.Since(start))
    }
    b.StopTimer()
    // ... sort, p99, b.ReportMetric, b.Errorf on breach — unchanged from
    // the existing VP-042 skeleton / keystroke_echo_bench_test.go pattern.
}
```

This is a small, deliberate divergence from the VP-042.md proof-harness
skeleton's exact call shape (`env.SendKeystroke` / `env.WaitForEcho` two-call
form with no token) — the skeleton predates the discovery that a token is
required to fix the accumulation bug (Q5). `test-writer`/`story-writer`
should treat the skeleton as directionally correct and this note's shape as
the binding API. Expected latency distribution: dominated by tick-cadence
wait (~half the upstream interval + half the downstream interval on average,
≈30ms; worst free-running case approaching the sum, ≈60ms, still comfortably
inside VP-042's 100ms ceiling — consistent with VP-042.md's own "~30s for 500
samples" estimate at these intervals, i.e. ~60ms/sample).

---

## Non-Goals (Explicit)

This story does NOT implement:

- **Real network I/O or cross-process operation.** Both "paths" are
  synthetic, zero-added-latency, in-process function calls. No sockets, no
  serialization to wire bytes (no `outerassembler.Assemble`/`DecodeChannelHeader`
  round trip) — `multipath.Frame`/`halfchannel.ChannelFrame` are passed as Go
  structs, not encoded bytes. If a future VP wants byte-level wire-format
  coverage in a loopback harness, that is a separate, additive story (see
  Risks).
- **Simulated packet loss, retransmission, or TLPKTDROP.** `GapsToRetransmit`
  and `TLPKTDROP` are not called on any schedule. `internal/arqsend` and
  `internal/outerassembler`-based real retransmit dispatch are **not** added
  to testenv's import set by this story — they would only be needed for a
  loss-injection follow-on. `internal/arq`'s own pure-core unit tests already
  cover the reorder/gap/TLPKTDROP state machine; this benchmark's job is
  realistic tick-driven happy-path latency, not re-proving ARQ correctness.
- **`internal/replay` / upstream idempotent-window fidelity.** See Q1 — out
  of scope per ARCH-03, and not part of the requested import set.
- **Empty-tick wire dispatch.** Empty ticks are produced by `Tick()` (BC-2.01.002
  compliance) but not dispatched over multipath in this harness — they carry
  no round-trip token and would not change the measured property. A
  full-fidelity extension could add this later at zero cost to the measured
  p99.
- **Changing `Env.SendKeystroke`/`Env.CollectFrames`/`Env.WaitForEcho`.**
  These remain exactly as they are for the 10 other VPs that use them.
- **A VP-042 verification_lock flip inside this story.** This story delivers
  the harness; locking VP-042 is a separate, subsequent act (run the
  benchmark, record evidence, update VP-042.md's frontmatter) once the
  harness lands and the architect/adversary have signed off on the `OnAck`
  call-contract decision in Q4 (see Risks).

---

## Package Impact Summary

| Package | Change | ARCH-08 §6.4 required? |
|---------|--------|------------------------|
| `internal/testenv` | New `loopbackDriver` type; `LoopbackEnv.SendKeystroke`/`WaitForEcho`/`CreateSession`/`RoundTrip`; `NewLoopback` wires halfchannel/arq/multipath/paths instead of discarding `LoopbackConfig` | No (existing package) — but import-set expansion requires the §6.4-equivalent pre-code registration already done in ARCH-08 v2.13 (this session) |
| `internal/halfchannel` | None — read-only consumer (`New`, `Tick`, `Enqueue`) | No |
| `internal/arq` | None — read-only consumer (`New`, `EnqueueSend`, `OnAck`); first production(-adjacent) call site for `OnAck` (see Risks) | No |
| `internal/multipath` | None — read-only consumer (`NewMultipath`, `Send`, `Receive`) | No |
| `internal/paths` | None — read-only consumer (`NewPathTracker`, `RankedPath`) | No |
| `internal/bench` | `keystroke_echo_testenv_bench_test.go` (WIP on `fix/vp-042-testenv-integrated-bench`) updated to the token-based two-call shape; package comment's "lower bound only" framing retired once the full stack lands | No |

**No new `internal/` package. ARCH-08 registration is the import-set
amendment already applied (v2.13, DRAFT/PROSPECTIVE, this session) — it
becomes final at this story's merge per the same machine-verification
protocol used for every prior testenv import-set change (v2.5, v2.8, v2.11).**

---

## Story-Sizing Estimate

**Estimate: 5–8 points (medium-large single story, or split into two:
"tick-driven halfchannel + multipath wiring" then "arq + round-trip-token
API").**

Rationale:
- The tick-driving mechanism (Q6) is low-risk and small — it's a direct copy
  of an idiom already used twice in the same file and twice more in
  `cmd/switchboard/access.go`.
- The multipath wiring (Q3, Q7) is low-risk — `Send`/`Receive`/`NewMultipath`
  are small, well-documented, already-tested pure APIs; the synthetic path
  construction is a few lines.
- The ARQ wiring (Q4) is the size and risk driver: it commits to a call
  contract (`OnAck`'s `ackSeq` semantics) that has no existing production
  precedent to copy, and that commitment should be reviewed (architect
  sign-off or an adversarial pass) before implementer treats it as settled —
  see Risks. If that review surfaces a different call contract, the
  downstream half of this design changes, not the upstream half or the
  tick-driving mechanism.
- The round-trip-token API (Q5) touches the WIP bench test
  (`fix/vp-042-testenv-integrated-bench`) and VP-042.md's harness skeleton,
  both of which need updating to the new two-call-with-token shape —
  small but real fan-out.
- No new package, no CI/deployment surface, no cross-cutting production code
  change — everything is additive inside `internal/testenv` plus the ARCH-08
  spec amendment already applied.

---

## Risks / Open Questions for story-writer to Encode as ACs

1. **`arq.OnAck` call-contract (Q4) needs explicit sign-off before
   implementation, not just transcription.** This design proposes a specific,
   internally-consistent convention (`ackSeq` = locally-observed frame's own
   `ChanSeq`, zero SACK in the no-loss happy path) because no production call
   site exists to copy. Story-writer should add an AC requiring either (a) an
   architect placement-note addendum confirming this contract before
   `dev-story` begins, or (b) a fast adversarial pass on this note
   specifically targeting Q4 before implementation starts. Getting this wrong
   doesn't break VP-042's measured number (the happy path is forgiving) but
   would misinform whatever *next* story tries to reuse `OnAck` for a real
   ACK/SACK path (e.g. a future loss-injection VP).
2. **`PathTracker.IsActive()` initial-state dependency.** This design relies
   on `NewPathTracker` defaulting `active: true` (confirmed by reading
   `paths.go:115-124` in this session) so no `OnProbe` warm-up is needed.
   Implementer should add a one-line assertion/test confirming this rather
   than re-deriving it from source at implementation time — cheap insurance
   against a future `paths` package change silently breaking the loopback's
   path activation.
3. **`RoundTrip.done` channel buffering and double-delivery.** If a
   `WaitForEcho` call times out and the corresponding entry is never read
   from `driver.pending`, the downstream ticker's completion-signal send
   (`ch <- frameFor(payload)`) would block forever unless `done` is buffered
   (proposed: buffer 1). Story-writer should add an explicit AC for the
   timeout-then-late-arrival case: the driver must still `delete` the pending
   entry and not leak it, and the buffered send must not block the ticker
   goroutine even if nobody ever reads it.
4. **Bounded `pending` map growth under a hung round trip.** If
   `WaitForEcho` is never called for a `RoundTrip` (test bug), `driver.pending`
   accumulates permanently until `Close`. This is a `t.Fatalf`-shaped
   programmer-error case, not a production concern (testenv is test-only) — a
   docstring warning is likely sufficient, but story-writer should decide
   whether it warrants an active safeguard (e.g. `t.Cleanup` asserting the map
   is empty) or is out of scope.
5. **This story does not itself flip VP-042's `verification_lock`.** See
   Non-Goals. Story-writer should scope the story to "harness lands, and is
   run once manually to produce evidence for the VP-042.md changelog" —
   the *lock* decision (editing `verification_lock: true`) is a separate
   PO/architect act per existing VP lifecycle convention (compare how VP-042's
   own history table already distinguishes "audited"/"partial evidence" from
   a lock flip).
