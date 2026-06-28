---
document_type: spec-ruling
title: "Unified Wraparound and seq=0 Reservation Ruling"
ruling_id: RULING-001
date: 2026-06-28
author: product-owner
stories_affected: [S-4.02, S-4.03]
findings_adjudicated: [S-4.02-F-002, S-4.03-M-2, S-4.03-M-3]
deferrals_reconciled: [S402-F006, S403-O1]
specs_modified:
  - .factory/specs/architecture/ARCH-02-protocol-stack.md
  - .factory/specs/behavioral-contracts/ss-02/BC-2.02.004.md
  - .factory/specs/behavioral-contracts/ss-02/BC-2.02.002.md
---

# Unified Wraparound and seq=0 Reservation Ruling

## Context

Three adversarial findings raised across S-4.02 (internal/replay) and S-4.03
(internal/arq) concern the same two protocol-level questions: whether seq=0 is
reserved, and whether 32-bit sequence wraparound is in-scope for MVP. The
findings are interdependent and must be resolved as a single coherent ruling
rather than per-story adjudications. Two pre-existing deferrals (S402-F006,
S403-O1) also requested architectural documentation on these topics.

## Findings Being Adjudicated

| Finding | Story | Severity | Description |
|---------|-------|----------|-------------|
| F-002 | S-4.02 | High | After delivering seq=MaxUint32, nextSeq wraps to 0; the next legitimate frame with seq=0 is unconditionally discarded as the "unset" sentinel |
| M-2 | S-4.03 | Medium | TLPKTDROP uses DegradationEvent{DroppedSeq:0} as "no event" sentinel, claimed valid because "seq 0 is never a valid in-flight seq" — but this is unspecified in any arch doc |
| M-3 | S-4.03 | Medium | OnAck cumulative-ACK loop `for seq := nextExpected+1; seq <= ackSeq; seq++` breaks at MaxUint32→0 wrap |

## R1 — seq=0 Reservation: YES, seq=0 is reserved

**Decision:** chan_seq is reserved to start at 1. seq=0 is the "unset/none"
sentinel and is never a valid wire-frame sequence number. The sender MUST start
at 1 and MUST skip 0 on wrap (i.e., the sequence after MaxUint32 is 1, not 0).

**Rationale:**

1. The existing implementation already treats seq=0 as "unset" in both
   `internal/replay` (the discard-if-zero check) and `internal/arq`
   (DroppedSeq=0 sentinel). The behavior is therefore already present; what was
   missing was the spec authority.

2. Reserving 0 is the simplest mechanism that eliminates the class of
   "unset-vs-valid" ambiguity with zero cost to the protocol (the sequence space
   is still 2^32 - 1 = 4,294,967,295 distinct values, ample for any session).

3. It is idiomatic. TCP ISN avoids 0 for similar reasons; many protocols
   reserve 0 as invalid.

4. The alternative — removing the seq=0 sentinel and handling seq=0 as a valid
   frame — would require additional state to represent "no drop event", or a
   pointer/optional type, with no benefit.

**Spec obligations from R1:**

- ARCH-02 must state: "chan_seq starts at 1; 0 is reserved and never a valid
  frame sequence number. On wrap from MaxUint32, the next value is 1 (skip 0)."
- BC-2.02.002 must add an edge case documenting that seq=0 frames received from
  the wire are treated as malformed and discarded.
- BC-2.02.004 must add a precondition guard: chan_seq starts at 1.

**Implementer obligations from R1:**

- **F-002 (S-4.02):** The replay receiver's seq=0 discard is now spec-conformant
  (seq=0 is always reserved). However, the sender-side wrap MUST skip 0: after
  emitting seq=MaxUint32, the next frame's chan_seq MUST be 1, not 0. Add a
  compile-time or test assertion in the tick loop: `if nextSeq == 0 { nextSeq = 1 }`.
  The existing seq=0 discard at the receiver requires no code change — it is
  correct per this ruling. Add a doc comment to the discard guard citing this
  ruling.

- **M-2 (S-4.03):** DegradationEvent{DroppedSeq:0} as "no event" sentinel is
  now spec-conformant. No code change required. Add a doc comment on the
  DroppedSeq field stating "0 = no event; 0 is reserved per ARCH-02 §chan_seq".

- **EnqueueSend / OnUpstream:** Add a precondition assertion (or guard + error
  return) that rejects seq=0 input frames. This is a defensive guard; the sender
  side already guarantees seq>=1 per this ruling, but the receiver-side guard
  makes the invariant testable.

## R2 — 32-bit Wraparound Scope: OUT of MVP scope, with documented session-lifetime assumption

**Decision:** 32-bit sequence wraparound (the MaxUint32→0/1 boundary) is OUT of
MVP scope. Sessions are assumed bounded to a duration short enough that wraparound
does not occur.

**Rationale:**

1. BC-2.02.002 EC-004 already established: "at normal tick rates, wrap takes
   > 24 hours." The domain spec therefore implicitly treated this as a long-lived
   but bounded session constraint.

2. The wrap-safe distance arithmetic in the window-bound check (S-4.02 F-001 fix)
   is being applied anyway because it is correct and free. This ruling does NOT
   retroactively mandate RFC-1982 serial-number arithmetic everywhere — only the
   window-distance calculation benefits from it.

3. The ACK loop (M-3) and the receiver seq-0 discard (F-002) are the two
   wrap-boundary failure modes. Under this ruling: the ACK loop wrap is
   documented-out; the seq-0 discard is resolved by R1's skip-0-on-wrap mandate.

4. Implementing RFC-1982-style arithmetic across all uint32 comparisons (ACK
   loop, dedup window, SACK bitmap offset) is non-trivial, carries test surface,
   and has no user-visible benefit for MVP sessions that terminate long before
   24h at any normal tick rate.

**The documented assumption** (written into BC-2.02.004 and BC-2.02.002):
"Session duration is bounded such that chan_seq does not wrap within a session
(32-bit space at 10ms tick rate requires > 497 days; at 100Hz it still requires
> 49 days). 32-bit wrap across an active session is not a supported scenario for
MVP. Implementations need not handle the MaxUint32→1 transition in receiver-side
comparison loops."

**Implementer obligations from R2:**

- **M-3 (S-4.03):** The OnAck cumulative-ACK loop wrap is DOCUMENTED-OUT for
  MVP. No code fix required. The implementer MUST add a doc comment to the loop:
  "Note: this loop does not handle uint32 wraparound; sessions are assumed to
  terminate before wrap per ARCH-02 §chan_seq and BC-2.02.004 EC-005." A test
  that seeds a near-MaxUint32 state and verifies graceful behavior (even if that
  behavior is merely "loop exits immediately without delivery" rather than
  "delivers across wrap") would be nice-to-have but is NOT required for MVP.

- **S-4.02 F-001 (wrap-safe distance arithmetic):** This fix is in-flight
  regardless of this ruling. It is a best-practice improvement to the
  window-bound check and does not constitute handling full wraparound.

## R3 — Reconciliation with Existing Deferrals

**S402-F006** (LOW, deferred to architect): "uint32 nextSeq wraparound in replay
— document lifetime assumption."

**S403-O1** (LOW, deferred to architect): "uint32 wraparound in arq — document
lifetime assumption."

**Ruling:** Both S402-F006 and S403-O1 are **fully subsumed and closed** by
R1+R2 above. Specifically:

- The "document lifetime assumption" obligation is satisfied by the ARCH-02 and
  BC-2.02.002 / BC-2.02.004 edits this ruling requires.
- S402-F006 also required "document sender-side skip-0-on-wrap contract" — that
  is satisfied by R1's ARCH-02 change.
- No residual remains. Both items should be marked `status: closed, closed_by:
  RULING-001` in STATE.md.

## Per-Finding Obligation Summary

| Finding | Resolution | Code obligation | Spec obligation |
|---------|------------|-----------------|-----------------|
| S-4.02 F-002 | R1 makes seq=0 discard conformant; R1 mandates skip-0-on-wrap | Add `if nextSeq == 0 { nextSeq = 1 }` guard in tick loop after increment; add doc comment on receiver discard | BC-2.02.004 EC-005 (skip-0) added; ARCH-02 §chan_seq updated |
| S-4.03 M-2 | R1 confirms DroppedSeq=0 sentinel is spec-conformant | Add doc comment on DroppedSeq field citing ARCH-02 | ARCH-02 §chan_seq now the authority; no BC change needed |
| S-4.03 M-3 | R2 documents this out of MVP scope | Add doc comment to ACK loop citing this ruling | BC-2.02.002 EC-004 updated to state wrap is out-of-MVP-scope explicitly |

## Spec Files Modified by This Ruling

1. **ARCH-02-protocol-stack.md** — §chan_seq bullet: add initial value = 1,
   reserved 0, skip-0-on-wrap, session-lifetime assumption. Version bump to v1.X.
2. **BC-2.02.004.md** — add EC-005 "chan_seq starts at 1; skip 0 on wrap" and
   EC-006 "session-lifetime assumption"; add Invariant 4 "chan_seq >= 1 always".
   Version bump to 1.2.
3. **BC-2.02.002.md** — update EC-004 to state "32-bit wrap is outside MVP scope
   (session bounded < wrap interval at any normal tick rate)." Version bump to 1.2.

## Deferrals Closed

| ID | Status | Closed by |
|----|--------|-----------|
| S402-F006 | closed | RULING-001 (R1 + R2) |
| S403-O1 | closed | RULING-001 (R1 + R2) |
