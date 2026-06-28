---
artifact_id: S-4.03-adversary-ackseq-dos-ruling
document_type: spec-ruling
ruling_id: RULING-003
title: "Out-of-Window Cumulative-ACK DoS Guard — OnAck Input Validation"
story_id: S-4.03
adjudicator: architect
timestamp: 2026-06-28T00:00:00
severity_addressed: HIGH
traces_to:
  - BC-2.02.005
  - ARCH-03-routing-engine.md
related_rulings:
  - RULING-001  # wraparound and seq=0 reservation
  - S-4.03-adversary-pass-2-adjudication  # delivery contract and inFlight deferral
modified:
  - version: "1.1"
    date: 2026-06-28
    author: architect
    reason: "Stale-anchor correction: EC-004 references updated to EC-005 throughout. EC-005 ('Out-of-window cumulative ACK received') was added to BC-2.02.005 v1.3 by this ruling and is the canonical contract for the OnAck guard. EC-004 is 'SACK bitmap overflow' — a distinct scenario. ARCH-03 v1.3 already correctly cited EC-005; this ruling and its derived code/test comment strings are now corrected to match. Also appended EC-003 test disposition section (Finding 2 ruling, Option A)."
---

# RULING-003: Out-of-Window Cumulative-ACK DoS Guard

## Finding

**Severity:** HIGH

`OnAck`'s Step-1 loop iterates `for seq := a.nextExpected + 1; seq <= ackSeq; seq++`.
`ackSeq` is the cumulative-ACK value parsed from the console's channel header — i.e.
peer- or attacker-controlled wire data. There is no validation that `ackSeq` lies
within one ARQ window of `nextExpected`.

A malformed or malicious cumulative ACK (e.g. `ackSeq = 0xFFFFFFFF` while
`nextExpected = 0`) drives approximately 4.29 billion loop iterations, each executing
two map-delete operations, stalling the single-threaded sender tick loop for the entire
duration from a single frame.

This is a per-frame compute-amplification DoS against an untrusted input surface.

### Distinctions from prior adjudicated items

This finding is distinct from both prior adjudications in scope:

| Item | Source | Nature |
|------|--------|--------|
| `inFlight` unbounded growth (RULING-002 / pass-2 §Ruling 2) | `EnqueueSend` call count | Slow accumulation from sender-side wiring; deferred to S-5.01 |
| RFC-1982 wrap deferral (RULING-001 §R2, arq.go line 209) | Long-session monotonic exhaustion | Session-lifetime-bounded; documented out of MVP |
| **This finding** | Single malformed frame from wire | Instant compute amplification; no time-budget protection |

The inFlight and wrap deferrals are appropriate because their threat model requires
sustained sessions or sender-side cooperation. This finding requires neither: one
frame from a hostile peer is sufficient.

---

## Disposition: FIX-HERE

The guard MUST be implemented in `OnAck` before this story is merged.

### Rationale

1. **Untrusted input surface.** `ackSeq` is parsed from the channel header of a frame
   received from the network. The ARQ package is pure-core, but its inputs arrive from
   the wire. Failing to validate them at the intake point violates the security boundary
   between effectful-shell (frame parsing) and pure-core (ARQ state machine). The pure
   core is responsible for asserting its own invariants on entry.

2. **Protocol invariant is cheap and well-defined.** A legal cumulative ACK can advance
   at most `sackWindowSize` (64) positions beyond `nextExpected` — the SACK bitmap only
   covers 64 positions, and a cumulative ACK logically cannot advance beyond the far edge
   of the window. The check is a single subtraction and comparison: O(1), zero allocation,
   no branching into per-frame data.

3. **Concrete bound exists in code.** `sackWindowSize = 64` is already a named constant
   in `arq.go`. The guard requires no new configuration; it derives from the existing
   window size constant.

4. **Incidental mitigation of near-MaxUint32 bitmap-seq wrap.** Step 2 computes
   `seq := ackSeq + 1 + uint32(i)`, which wraps silently if `ackSeq` is near
   `MaxUint32`. Rejecting out-of-window `ackSeq` values in Step 1 also eliminates
   this overflow path for the in-window case, without requiring RFC-1982 arithmetic.

5. **Precedent: reject, do not clamp.** Clamping `ackSeq` to `nextExpected + sackWindowSize`
   would silently ignore part of the peer's stated ACK state, potentially masking retransmit
   suppression attacks (peer sends large ackSeq to falsely advance the sender's delivery
   pointer without the sender having actually received the frames). Rejection is the correct
   protocol response to a protocol-illegal message, consistent with how `SACKFromChannelHeader`
   returns an error on a malformed header rather than silently truncating.

---

## Guard Semantics — Exact Contract

### New sentinel error

```go
// ErrAckOutOfWindow is returned by OnAck when the cumulative ACK sequence
// number falls outside the valid window: ackSeq must satisfy
// ackSeq - nextExpected <= sackWindowSize (64). An out-of-window ackSeq is
// a protocol-illegal frame; the caller should log and discard it.
// Traces to: BC-2.02.005 PC-3, EC-005; RULING-003.
var ErrAckOutOfWindow = fmt.Errorf("arq: cumulative ACK out of window")
```

### Guard placement and logic

The guard is inserted at the top of `OnAck`, before Step 1, after the
`prevNextExpected` snapshot:

```go
// Validate ackSeq is within one ARQ window of nextExpected.
// ackSeq is wire-derived (peer/attacker-controlled). A legal cumulative ACK
// advances at most sackWindowSize (64) positions. An out-of-window value
// would drive the Step-1 loop for up to 2^32 iterations — a per-frame DoS.
// Reject without iterating (RULING-003; BC-2.02.005 PC-3, EC-005).
//
// The subtraction is unsigned: if ackSeq < nextExpected the result wraps to
// a large uint32 (> sackWindowSize), so the guard also correctly rejects
// stale (already-ACKed) values without a separate comparison.
if ackSeq-a.nextExpected > sackWindowSize {
    return nil, ErrAckOutOfWindow
}
```

### Key properties of this guard

- **Zero-iteration path on invalid input.** `OnAck` returns immediately without
  touching the Step-1 loop, Step-2 bitmap scan, or Step-3 flush.
- **Unsigned subtraction handles the stale-ACK case.** If `ackSeq < nextExpected`
  (the peer sent an ACK for something already delivered), `ackSeq - a.nextExpected`
  wraps to a large uint32, exceeding `sackWindowSize`, and the frame is rejected.
  This is correct: stale cumulative ACKs are protocol-illegal and should not trigger
  the SACK re-scan.
- **`ackSeq == nextExpected` is valid (no-op ACK).** `ackSeq - nextExpected == 0`,
  which is not `> sackWindowSize`. Step 1 loop body does not execute (start > end).
  Step 3 guard `ackSeq > prevNextExpected` is false. Returns `(nil, nil)`. This is
  the idempotent "no progress" path; already exercised by existing tests.
- **`ackSeq == nextExpected + sackWindowSize` is valid.** The diff is exactly
  `sackWindowSize` (64), which is not `> sackWindowSize`. Maximum one-window advance
  is permitted.
- **Does not interact with RFC-1982 wrap.** RULING-001 §R2 documents that
  MaxUint32 session wrap is out of MVP scope. This guard uses the same uint32
  arithmetic as the rest of the package. No additional wrap handling is required.

### Return contract change to `OnAck`

`OnAck` already returns `([][]byte, error)` per the pass-2 adjudication. The `error`
return was previously always `nil` (no error path existed). This ruling adds the first
non-nil error path. The error is a protocol-layer rejection, not a state-corruption
event — ARQ state is unmodified on this return path.

Callers must check `errors.Is(err, arq.ErrAckOutOfWindow)` and handle the frame as
malformed (log + discard at the caller's tick loop; do not advance state).

---

## BC-2.02.005 Trace

| Clause | Relevance |
|--------|-----------|
| **PC-3** ("ARQ window configured") | The configured window size bounds legal cumulative-ACK advance. An ackSeq outside this window is a precondition violation that the receiver may reject. |
| **EC-005** ("Out-of-window cumulative ACK received — `ackSeq - nextExpected > 64`") | This is the dedicated contract for the OnAck guard. `OnAck` returns `ErrAckOutOfWindow`; ARQ state is unmodified; caller discards the frame. Added to BC-2.02.005 v1.3 by this ruling. |
| **EC-004** ("SACK bitmap overflow — frames outside range trigger NACK or rely on ARQ timeout") | Analogous out-of-window scenario for the SACK bitmap side; distinct from the cumulative-ACK guard. Referenced for context only — the canonical anchor for the `ErrAckOutOfWindow` guard is EC-005. |

---

## Red-Gate Test Obligation

The implementer MUST add the following test (or equivalent table row) before merge.
The test must be present as a failing test (red gate) before the guard is implemented,
confirming that the pre-guard code drives the loop without the protection.

### Test: `TestOnAck_OutOfWindowAckSeq_RejectsWithoutIteration`

```go
// TestOnAck_OutOfWindowAckSeq_RejectsWithoutIteration verifies that OnAck
// rejects a cumulative ACK whose distance from nextExpected exceeds
// sackWindowSize (64) without executing the Step-1 iteration loop.
//
// Red-gate obligation: RULING-003. Traces to BC-2.02.005 PC-3, EC-005.
func TestOnAck_OutOfWindowAckSeq_RejectsWithoutIteration(t *testing.T) {
    t.Parallel()
    cases := []struct {
        name         string
        nextExpected uint32
        ackSeq       uint32
    }{
        {
            name:         "large gap from zero",
            nextExpected: 0,
            ackSeq:       sackWindowSize + 1, // 65 — first illegal value
        },
        {
            name:         "max uint32 attack",
            nextExpected: 0,
            ackSeq:       0xFFFFFFFF,
        },
        {
            name:         "stale ack (already delivered)",
            nextExpected: 100,
            ackSeq:       50, // behind nextExpected — wraps to large uint32
        },
        {
            name:         "exactly one over window",
            nextExpected: 10,
            ackSeq:       10 + sackWindowSize + 1, // 75 — first illegal from 10
        },
    }

    for _, tc := range cases {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            a := New(Config{DropTimeout: time.Second})
            a.nextExpected = tc.nextExpected

            // Enqueue a frame at ackSeq to confirm the loop would have done
            // work if it ran (absence of side-effects proves no iteration).
            a.EnqueueSend(tc.ackSeq, []byte("payload"), time.Now().UTC())

            var zeroSACK [SACKBitmapBytes]byte
            frames, err := a.OnAck(tc.ackSeq, zeroSACK)

            if !errors.Is(err, ErrAckOutOfWindow) {
                t.Fatalf("want ErrAckOutOfWindow, got %v", err)
            }
            if len(frames) != 0 {
                t.Fatalf("want no frames delivered on rejection, got %d", len(frames))
            }
            // nextExpected must be unchanged — no state mutation on rejection.
            if a.nextExpected != tc.nextExpected {
                t.Fatalf("want nextExpected unchanged (%d), got %d",
                    tc.nextExpected, a.nextExpected)
            }
            // inFlight entry must still be present — no deletes executed.
            if _, ok := a.inFlight[tc.ackSeq]; !ok {
                t.Fatalf("want inFlight entry preserved on rejection, got deleted")
            }
        })
    }
}
```

### Test: `TestOnAck_BoundaryWindowValues_Accepted`

```go
// TestOnAck_BoundaryWindowValues_Accepted verifies that ackSeq values at exactly
// the window boundary are accepted (ackSeq - nextExpected == sackWindowSize).
//
// Companion to TestOnAck_OutOfWindowAckSeq_RejectsWithoutIteration.
// Traces to: RULING-003, BC-2.02.005 PC-3.
func TestOnAck_BoundaryWindowValues_Accepted(t *testing.T) {
    t.Parallel()
    cases := []struct {
        name         string
        nextExpected uint32
        ackSeq       uint32
    }{
        {
            name:         "exactly at window edge",
            nextExpected: 0,
            ackSeq:       sackWindowSize, // 64 — last legal value
        },
        {
            name:         "no-op ack (ackSeq == nextExpected)",
            nextExpected: 5,
            ackSeq:       5,
        },
        {
            name:         "one step advance",
            nextExpected: 10,
            ackSeq:       11,
        },
    }

    for _, tc := range cases {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            a := New(Config{DropTimeout: time.Second})
            a.nextExpected = tc.nextExpected

            var zeroSACK [SACKBitmapBytes]byte
            _, err := a.OnAck(tc.ackSeq, zeroSACK)
            if err != nil {
                t.Fatalf("want nil error for in-window ackSeq, got %v", err)
            }
        })
    }
}
```

---

## Summary

| Dimension | Decision |
|-----------|----------|
| Disposition | FIX-HERE |
| Guard type | REJECT (return `ErrAckOutOfWindow`), not clamp |
| Guard expression | `ackSeq - a.nextExpected > sackWindowSize` (unsigned, handles stale-ACK wrap) |
| New sentinel | `ErrAckOutOfWindow` (package-level var, consistent with `ErrSequenceNotInFlight`, `ErrFrameNotOverdue`) |
| State mutation on rejection | None — ARQ state is unmodified |
| BC trace | BC-2.02.005 PC-3 (window configured), EC-005 (out-of-window cumulative ACK); EC-004 (SACK bitmap overflow, analogous scenario — distinct) |
| ARCH-03 update | Yes — one line in §Downstream ARQ; see §Spec Changes (Status) below |
| Test obligation | Two test functions (rejection table + boundary acceptance table) |
| Interaction with RULING-001 | None — uint32 wrap is documented out of MVP; this guard uses same unsigned arithmetic |
| Interaction with pass-2 §Ruling 2 | None — `inFlight` growth deferral to S-5.01 unchanged |

---

## Spec Changes (Status)

### BC-2.02.005 — COMPLETE (v1.3 already applied)

EC-005 ("Out-of-window cumulative ACK received") was added to the Edge Cases table
and the `modified` block was updated in BC-2.02.005 v1.3 (timestamp 2026-06-28,
author: architect, reason: "RULING-003: add EC-005 out-of-window cumulative-ACK
rejection contract"). No further action required on BC-2.02.005.

### ARCH-03-routing-engine.md — COMPLETE (v1.3 already applied)

The §Downstream ARQ input validation paragraph was added in ARCH-03 v1.3
(changelog entry 2026-06-28, adjudication: S-4.03 RULING-003 ackseq-dos-ruling).
The paragraph correctly cites EC-005. No further action required on ARCH-03.

---

## EC-003 Test Disposition (Finding 2 — RULING-003 Addendum)

**Finding:** `TestBC_2_02_006_EC003_TLPKTDROPDuringFailover` claims in its docstring
to model ADR-005 resync-on-reconnect, but its body performs no resync. It does
`TLPKTDROP(99)` then `EnqueueSend(100)` + `OnAck(100)` — a plain linear continuation
behaviorally identical to `TestBC_2_02_006_TLPKTDROP_SessionContinues`. The
`nextExpected=98` setup exists only to satisfy the RULING-003 out-of-window guard.
The ADR-005 "resync on reconnect" semantic is not actually verified.

**Disposition: Option A — Relabel the test; defer ADR-005 resync mechanics to S-5.01.**

**Rationale:**

1. ADR-005 resync requires a `RESYNC` control frame, a reconnect event, and a
   delivery-pointer reset (`last_acked_seq + 1`) — all of which are router/reconnect-layer
   concerns in the effectful shell. `internal/arq` is pure-core: it has no concept of
   "reconnect" or "new connection." The resync event arrives at `internal/arq` as a
   state mutation from the outside; the pure core cannot initiate or observe it.

2. Precedent is consistent. S-4.01 / BC-2.02.009 router wiring was deferred to S-4.04.
   S-4.03's own retransmit-SEND PC-3 was deferred to S-5.01. ADR-005's wire-mechanic
   implementation (RESYNC frame format, reconnect-state machine, in-flight-loss on
   failover) belongs at the same scope boundary and the same deferral phase (S-5.01).

3. What the test body DOES verify is real and worth pinning: after a TLPKTDROP event
   (simulating degradation during failover), the ARQ accepts the next enqueued frame
   and acknowledges it normally. That is a meaningful post-drop-continuity check that
   deserves its own canonical name.

**Exact implementer instructions:**

1. **Rename the test** from `TestBC_2_02_006_EC003_TLPKTDROPDuringFailover` to
   `TestBC_2_02_006_EC003_DegradationAndPostDropContinuation`.

2. **Replace the docstring** with the following (do not change the test body):

   ```go
   // TestBC_2_02_006_EC003_DegradationAndPostDropContinuation verifies that after
   // a TLPKTDROP event — as would occur during a router failover — the ARQ emits
   // the degradation signal and continues accepting subsequent frames normally.
   //
   // Traces to: BC-2.02.006 EC-003 (TLPKTDROP during failover, degradation emitted).
   //
   // NOTE: ADR-005 resync mechanics (RESYNC control frame, delivery-pointer reset
   // from last_acked_seq+1, in-flight loss handling on reconnect) are router/reconnect-
   // layer concerns deferred to S-5.01. internal/arq is pure-core; it has no concept
   // of reconnect and cannot initiate or observe a resync event. The S-5.01 story
   // will wire the reconnect trigger and add tests for the full ADR-005 state machine.
   // Drift item: DRIFT-S4.03-001.
   ```

3. **Add the following drift item comment** at the top of the test file (above the
   package declaration), or in a dedicated `// DRIFT:` block near the test if a
   top-of-file comment is not appropriate for this codebase's style:

   ```go
   // DRIFT-S4.03-001: ADR-005 resync mechanics (RESYNC frame, delivery-pointer reset,
   // in-flight-loss on failover) are not verified in internal/arq. Deferred to S-5.01
   // (router reconnect layer). See RULING-003 EC-003 disposition.
   ```

   If the codebase style does not use top-of-file drift comments, place this comment
   immediately above `TestBC_2_02_006_EC003_DegradationAndPostDropContinuation` instead.

**No spec edits required.** BC-2.02.006 EC-003 already reads "TLPKTDROP fires during
router failover — Degradation signal emitted; ARQ resync on reconnect (ADR-005)."
The "ARQ resync on reconnect (ADR-005)" clause describes the full system behavior
across the failover; it does not assert that `internal/arq` alone implements the
resync. The test relabel and drift comment provide sufficient traceability without
a BC version bump. ARCH-03 ADR-005 already states "The exact RESYNC frame format
and state machine are deferred to PE implementation."

**S-5.01 obligation (informational):** When S-5.01 is written, the story MUST
include an acceptance criterion that verifies the ADR-005 resync state machine:
`RESYNC` control frame received → sender replays from `last_acked_seq + 1` →
in-flight frames at failover time are discarded. Reference DRIFT-S4.03-001.
