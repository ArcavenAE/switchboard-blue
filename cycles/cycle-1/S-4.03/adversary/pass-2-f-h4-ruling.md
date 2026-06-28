---
document_type: spec-ruling
ruling_id: F-H4
story: S-4.03
bc: BC-2.02.005
adversary_finding: F-H4
severity_claimed: Medium
disposition: A
ruled_by: product-owner
timestamp: 2026-06-28T00:00:00
---

# Ruling on F-H4: OnAck Cumulative-ACK Past Locally-Absent Sequence

## Finding Summary

In `ARQ.OnAck`, when the cumulative ACK covers a sequence number for which
`payloadFor(seq) == nil` (frame not in `inFlight`, not in `reorderBuf`), the
code advances `nextExpected` past it and deletes the key. The adversary flags
this as a potential violation of BC-2.02.005 invariant 2 ("No byte of terminal
output is permanently lost within the ARQ window") and postcondition 4 ("gaps
held until filled").

## Disposition: (A) — Silent Skip Is Correct and Intended

### Reasoning

**Role clarification (from ARCH-03):**

ARCH-03 §"Downstream ARQ" draws an explicit role split:

```
Sender (access node):  SendBuffer / inFlight  — tracks unACK'd outbound frames
Receiver (console):    RecvBuffer / reorderBuf / nextExpected — delivers in sequence
```

`OnAck` is called on the **sender side** (access node). The cumulative ACK
`ackSeq` is the console telling the sender "I have received and delivered
everything up to `ackSeq` in order." The sender's `nextExpected` (as used in
the loop `for seq := nextExpected+1; seq <= ackSeq`) is tracking *which
sequences the remote has acknowledged*, not *which sequences the local node
must deliver locally*.

**Invariant 2 scope:**

BC-2.02.005 invariant 2 reads: "No byte of terminal output is permanently
lost **within the ARQ window**."

The invariant governs delivery to the console (the downstream direction). It
binds the sender to retransmit until the console ACKs or TLPKTDROP fires. When
the cumulative ACK asserts `ackSeq`, the console is explicitly telling the
sender "those bytes reached me." Invariant 2 is therefore *satisfied* by the
ACK itself — the bytes were not lost, the remote received them. The fact that
the sender's local `inFlight` map no longer holds the frame (already cleaned
up in a prior pass, or sent before a resync event, or cleaned by an earlier
partial ACK) does not alter this.

**Postcondition 4 scope:**

BC-2.02.005 PC-4 ("gaps held until filled") is a **receiver-side** postcondition.
It governs the console's `reorderBuf`: frames received out-of-order at the
console must be buffered until the gap is filled. It does not govern the sender's
response to a cumulative ACK that covers sender-side gaps in `inFlight`.

**The comment in the code is accurate:**

The implementation comment ("the cumulative ACK tells us the remote received
them; we only surface what we have locally") correctly captures the invariant.
A cumulative ACK is an assertion by the remote, not a delivery trigger on the
sender. The sender has no obligation to "deliver" anything on receipt of an ACK
— it only needs to stop retransmitting those frames and advance its window.

**Is there a real-world scenario where a cumulatively-ACKed-but-locally-absent
seq indicates a bug?**

The only way a cumulative ACK can cover a seq the sender never held is:

1. The sender retransmitted the frame in a prior send window, then cleared
   `inFlight` (e.g., on a previous partial ACK). The console received it on
   the second path and is now ACKing it cumulatively. Correct: advance past it.
2. The sender restarted (resync per ADR-005) and replayed from `last_acked_seq
   + 1`. An ACK arriving from before the resync could cover seqs the new
   sender instance never held. Correct: the ADR-005 resync protocol explicitly
   acknowledges that in-flight frames during failover are lost; the cumulative
   ACK received after resync reflects the console's state, not the sender's.
3. A bug in the ACK counter at the console (overcounting). This would be
   a receiver-side bug, not a sender-side invariant violation. The sender
   trusting the ACK is correct ARQ behavior.

In none of these cases does advancing `nextExpected` past the locally-absent
seq cause permanent byte loss at the terminal — either the bytes reached the
console already (cases 1–2) or the ACK is malformed (case 3, a receiver bug
not addressable here).

## Required Follow-Up

### 1. Clarify BC-2.02.005 (mandatory, same burst)

Add a postcondition note to BC-2.02.005 making the scope boundary explicit so
future readers do not re-raise this question. Bump version to 1.2.

**Text to add as a new note under Postcondition 4:**

```markdown
> **Scope note (PC-4):** PC-4 is a receiver-side invariant (console `reorderBuf`).
> On the sender side, advancing `nextExpected` past a cumulatively-ACKed sequence
> that is not in `inFlight` is correct and intended: the cumulative ACK is an
> assertion by the remote that it received those bytes. The sender has no
> obligation to hold or re-deliver content already confirmed received. This does
> NOT violate invariant 2 — the bytes were not lost; the ACK proves they arrived.
> Locally-absent entries in the sender's ACK scan loop are a normal consequence
> of prior window advancement, QUIC-style retransmit cleanup, or post-resync
> state (ADR-005).
```

**Also add a new Invariant 4:**

```markdown
4. The cumulative ACK is trusted as an assertion by the remote. Sender-side
   `nextExpected` advancement past locally-absent sequences does not violate
   invariant 2 — remote receipt is the delivery guarantee, not local holding.
```

### 2. Add a pinning test (mandatory, before story closes)

A test must pin the intended behavior so a future implementer cannot
accidentally change it thinking it's a bug:

```
TestARQ_OnAck_CumulativeAckPastLocallyAbsentSeq

Scenario:
  - Sender sends frames 1, 2, 3.
  - Sender clears frame 2 from inFlight (simulating a prior partial ACK or
    retransmit cleanup) so only frames 1 and 3 remain in inFlight.
  - Remote sends cumulative ACK for seq=3.
  - Call: arq.OnAck(ackSeq=3, sack=0)

Expected:
  - No panic, no error.
  - nextExpected advances to 4.
  - inFlight is empty.
  - No "gap" or error event emitted.
  - Frames 1 and 3 are removed from inFlight (ACK consumed them).
```

This test must be added to `internal/arq/arq_test.go` as part of S-4.03
deliverables. It is a unit test, not a property test.

### 3. No implementation change required

The implementation behavior is spec-conformant as described. Do NOT change
the logic in `OnAck`.

## Spec Change Summary

| File | Change |
|------|--------|
| `.factory/specs/behavioral-contracts/ss-02/BC-2.02.005.md` | Add PC-4 scope note + Invariant 4; bump version 1.1 → 1.2; add changelog entry |

## What This Ruling Does NOT Address

- The structural concern that one `ARQ` struct conflates sender and receiver
  roles is noted as a design smell but is out of scope for this ruling. The
  dual-role struct is consistent with ARCH-03's description of `internal/arq`
  as a single module. If the roles diverge in a future phase (multipath
  sender needing independent state from the receiver buffer), that refactor
  belongs in a dedicated story, not here.

- ADR-005 resync state machine details are deferred to PE phase per
  ARCH-03 §ADR-005. The ruling does not change that deferral.
