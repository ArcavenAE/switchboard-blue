---
artifact_id: S-4.03-adversary-pass-2-adjudication
document_type: adjudication-note
story_id: S-4.03
adjudicator: architect
timestamp: 2026-06-28T00:00:00
severity_addressed: CRITICAL
defect_ref: pass-1-adversarial-review (goroutine-chain / purity violation)
traces_to:
  - BC-2.02.005
  - BC-2.02.006
  - S-4.03-downstream-arq-tlpktdrop.md
  - ARCH-03-routing-engine.md
---

# S-4.03 Pass-2 Adjudication — Delivery Contract and F-H3 Scope Rulings

## Defect Summary

The current implementation of `ARQ.OnAck` spawns one goroutine per call,
chained via `prevDelivery chan struct{}` to preserve FIFO order while keeping
`OnAck` non-blocking. This pattern violates two hard constraints:

1. **Purity violation.** The story's Purity Classification table and the ARCH-03
   routing engine spec both classify `internal/arq` as pure-core: "Pure state
   machine; no I/O; TLPKTDROP signals via channel not OS calls." Goroutine
   spawning inside OnAck is a side effect; pure-core modules may not spawn
   goroutines. This is the same constraint that governs `internal/halfchannel`
   (see its package doc: "no goroutines, no timers, no I/O") and `internal/replay`.

2. **Unbounded leak.** If the consumer of `DeliveredFrames` ever stalls (terminal
   disconnect, slow reader, teardown), every subsequent `OnAck` call spawns a
   goroutine that blocks forever at the channel send. There is no `Close`, `Stop`,
   or context cancellation. This is a DoS vector; severity Critical.

`TLPKTDROP` has a related but lesser variant of the same problem: it sends to
`DegradationEvents` outside the lock but synchronously on the caller goroutine.
If `DegradationEvents` is unbuffered and the consumer is not draining, `TLPKTDROP`
blocks the caller. This must also be resolved.

---

## Ruling 1 — Delivery Contract (Critical Fix)

### Decision: Option (a) — synchronous return value

`OnAck` and `TLPKTDROP` MUST return delivered/dropped frames synchronously as
`[][]byte`. The goroutine chain, `prevDelivery chan struct{}`, and
`DeliveredFrames chan []byte` are eliminated from the ARQ struct. The struct field
`DegradationEvents chan DegradationEvent` is retained (see rationale below).

### Rationale

**Option (a) is the correct choice.** It is the only option consistent with:

- The pure-core precedent set by `internal/halfchannel`. `HalfChannel.Tick()`
  returns a `ChannelFrame` synchronously; the caller's tick loop routes the result.
  There is no internal channel, no goroutine, no back-pressure. ARQ must follow
  the same pattern.
- Go rule 12 (no internal-pointer leak): returning `[][]byte` where the slices are
  owned by the caller is safe. Option (b) would require the caller to drain a
  shared channel while also calling `OnAck` — a concurrency surface that does not
  exist in any other pure-core package in this repo.
- The story's single-writer contract: "ARQ is driven by a single half-channel tick
  loop." The tick loop already owns the goroutine. There is no motivation for ARQ
  to spawn its own goroutine.

**Option (b) is rejected.** Synchronous send under a documented "caller-drain
back-pressure contract" moves the blocking problem from a background goroutine to
the caller's tick loop. Under a stalled consumer the tick loop blocks, which is
equally bad and inconsistent with the half-channel model where `Tick()` always
returns without blocking.

**Why keep `DegradationEvents chan DegradationEvent`?**

Degradation events are consumed by a different subsystem (`internal/metrics`,
BC-2.06.001) that is not the ARQ caller. Making `TLPKTDROP` return a
`*DegradationEvent` (non-nil when dropped) is acceptable, but a buffered channel
for degradation events is also consistent with how the metrics layer is wired in
ARCH-03. Either shape is acceptable for `DegradationEvents`; the critical fix is
eliminating the delivery goroutine and returning frames synchronously. The
`DegradationEvents` channel, if retained, MUST be buffered; sending to it must be
done with a non-blocking select or it must be eliminated and replaced with a return
value. See exact signatures below.

### Exact Method Signatures

```go
// OnAck advances the receiver state machine.
//
// Returns the ordered slice of frame payloads ready for terminal output.
// The returned slices are owned by the caller; ARQ retains no reference to
// them after return. The caller's tick loop forwards them to the terminal.
//
// DegradationEvents is not affected by OnAck.
//
// Concurrency: must be called from a single goroutine (the tick loop).
// Not safe for concurrent calls.
func (a *ARQ) OnAck(ackSeq uint32, sackBitmap [SACKBitmapBytes]byte) ([][]byte, error)
```

```go
// TLPKTDROP terminates the overdue frame identified by overdueSeq.
//
// Returns ErrSequenceNotInFlight if overdueSeq is not in the retransmit queue.
// Returns ErrFrameNotOverdue if the frame has not exceeded its deadline (AC-005).
//
// On success, sends a DegradationEvent to DegradationEvents (non-blocking;
// the channel must be buffered — see New). The caller must drain
// DegradationEvents on its tick loop. The dropped sequence is also returned
// as a DegradationEvent value for callers that prefer synchronous inspection.
//
// Concurrency: must be called from a single goroutine (the tick loop).
func (a *ARQ) TLPKTDROP(overdueSeq uint32, now time.Time) (DegradationEvent, error)
```

The zero value of `DegradationEvent` (DroppedSeq == 0) is the sentinel for "no
event" (e.g. on error return). Sequence 0 is never a valid in-flight sequence in
practice (seqs start at 1 per the wire format), so zero is unambiguous.

### Struct Changes

Remove from `ARQ`:
- `prevDelivery chan struct{}`
- `DeliveredFrames chan []byte`

Remove from `Config`:
- `DeliveredBufSize int`

Retain (but ensure buffered):
- `DegradationEvents chan DegradationEvent`
- `DegradationBufSize int` in `Config`

`DegradationBufSize` of 0 must NOT create an unbuffered channel; `New` must
default to a minimum buffer of 1 if `DegradationBufSize == 0`. Document this
in the `Config` struct comment.

### Ownership and Copy Rule for `payloadFor`

`payloadFor` currently deep-copies payloads from `inFlight` before returning.
This is correct and must be preserved in the new design. The rule is:

> Every `[]byte` returned by `OnAck` is a fully-owned copy — ARQ has no further
> reference to it. The caller may hold, forward, or mutate the slice freely.
> Payloads sourced from `reorderBuf` are already copies (cloned on SACK insert);
> payloads sourced from `inFlight` are copied in `payloadFor` before return.
> No `[]byte` returned by `OnAck` aliases any internal ARQ slice.

This satisfies go.md rule 12 (never return internal pointers from a locked
accessor).

### `sync.Mutex` Removal

With the goroutine chain eliminated and the single-writer contract enforced, `ARQ`
no longer requires a `sync.Mutex`. The mutex and all lock/unlock calls must be
removed. The package doc comment already states "NOT safe for concurrent OnAck or
TLPKTDROP calls from multiple goroutines"; removing the mutex makes this explicit
in the type definition and eliminates the dead-lock/race surface. If the
implementer judges that `EnqueueSend` may be called from a different goroutine
than `OnAck`/`TLPKTDROP` (e.g. from the sender-side wiring in S-5.01), the mutex
may be retained on `inFlight` only, with a doc comment explaining which methods
take the lock.

### Test Changes Required

All tests that currently:
- Drain `a.DeliveredFrames` via channel receive must be updated to collect the
  `[][]byte` return value of `OnAck` directly.
- Use `mustDrainOne`, `assertNoPending` helpers that select on `a.DeliveredFrames`
  must be rewritten to assert on the return value.
- Use `mustDrainOneDeg` / `assertNoPendingDeg` for `TLPKTDROP` must drain
  `a.DegradationEvents` (channel send remains for that path).

The large-scale property tests (`TestBC_2_02_005_VP019_VP020_LargeScale`,
`TestBC_2_02_005_VP019_VP020_NoDoubleDelivery`) use a drain loop over
`a.DeliveredFrames` with `time.After`; these must be replaced with direct
collection of return values — this also removes the non-deterministic
`time.After` race from the property tests, which is a correctness improvement.

---

## Ruling 2 — F-H3 Scope: `inFlight` Window Bound

### Finding

`inFlight` (the retransmit queue appended by `EnqueueSend`) has no window bound.
If ACKs never arrive, it grows without limit. The story notes `blocks: [S-5.01]`
(send-side wiring), which is the story that will wire up retransmit driving and
ultimately control how `inFlight` is populated under a live send loop.

### Decision: Defer to S-5.01, with documentation obligation

Bounding `inFlight` is **out of scope for S-4.03** and deferred to S-5.01. The
reasoning:

1. `inFlight` is populated by `EnqueueSend`, which is called by the sender-side
   wiring — not yet implemented. S-4.03 is a pure ARQ state machine; it cannot
   enforce a window bound without knowing the negotiated window size, which is a
   sender-side concern that S-5.01 must establish.
2. Adding a window bound here would require `New` to accept a `MaxInFlight int`
   config field and `EnqueueSend` to return an error or silently drop when the
   window is full — neither is specified in the current BC or story.
3. The existing `TestARQ_ReorderBuf_BoundedByWindow` test correctly bounds
   `reorderBuf` (the receiver-side buffer), which is S-4.03's scope.

### Documentation Obligation (mandatory before merge)

The implementer MUST add the following comment to `inFlight` in the struct definition
before this story is closed:

```go
// inFlight is the sender's retransmit queue: frames sent but not yet ACKed.
//
// NOTE: inFlight has no window bound in this package. The caller (sender-side
// wiring, S-5.01) is responsible for bounding the number of concurrent
// EnqueueSend calls to the negotiated ARQ window size. Unbounded EnqueueSend
// calls without corresponding OnAck calls will grow this map without limit.
// Window enforcement is deferred to S-5.01 (BC-2.02.005 precondition 3).
inFlight map[uint32]*inFlightFrame
```

### Test Rename Obligation

`TestARQ_ReorderBuf_BoundedByWindow` MUST be renamed to make its actual scope
explicit. The current name implies both `reorderBuf` and `inFlight` are bounded,
which is false for `inFlight`. Rename to:

```
TestARQ_ReorderBuf_BoundedByWindowSize
```

And add a comment at the top of the test body:

```go
// This test verifies only that reorderBuf is bounded by the SACK window (64
// positions above ackSeq). The inFlight map has no window bound in this package;
// window enforcement is deferred to S-5.01. See inFlight field comment.
```

---

## ARCH-03 Spec Update

The following minimal edit to ARCH-03-routing-engine.md §Downstream ARQ is
required to reflect the ruling. The implementer should NOT make this edit;
it is the architect's obligation.

In the "Downstream ARQ" section, replace the prose:

> Frames newly made deliverable in-order are sent on DeliveredFrames.

with:

> `OnAck` returns deliverable frames synchronously as `[][]byte`. The caller's
> tick loop forwards them to the terminal. There is no internal channel or
> goroutine for frame delivery (pure-core constraint; see S-4.03 pass-2
> adjudication).

No version bump is required for ARCH-03 at this time; the change is a
clarification of the implementation contract, not an architectural decision
reversal. A changelog entry is sufficient.

---

## Summary Checklist for Implementer

- [ ] Remove `prevDelivery chan struct{}` and `DeliveredFrames chan []byte` from
      `ARQ` struct and `Config`
- [ ] Change `OnAck` signature to `([][]byte, error)`; return `toDeliver` directly
- [ ] Change `TLPKTDROP` signature to `(DegradationEvent, error)`; send to
      `DegradationEvents` non-blocking (select with default or buffered-only)
- [ ] Remove `sync.Mutex` (or scope it to `inFlight` only if S-5.01 requires
      cross-goroutine access — document the decision)
- [ ] Remove `New`'s `prevDelivery` pre-signal setup
- [ ] Add `inFlight` comment about unbounded growth / S-5.01 deferral
- [ ] Rename `TestARQ_ReorderBuf_BoundedByWindow` to
      `TestARQ_ReorderBuf_BoundedByWindowSize` and add scope comment
- [ ] Update all tests to collect `[][]byte` from `OnAck` return value; remove
      channel drain helpers `mustDrainOne` / `assertNoPending`
- [ ] Run `just test-race` — the mutex removal eliminates a race-detector surface;
      verify no new races under the single-writer model
