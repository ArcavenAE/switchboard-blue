---
story: S-4.03
pass: 1
reviewed_commit: b6718ae
verdict: NOT_CONVERGED
severity_summary: 1C/2H
streak: 0
streak_target: 3
date: 2026-06-28
---

# S-4.03 Adversarial Review — Pass 1

**Verdict:** NOT_CONVERGED 1C/2H (streak reset to 0/3)
**Tip reviewed:** b6718ae

**NOTE:** SACK channel-header offsets bytes 12-19, flags bit 0x04, and the truncated-header error path are CORRECT — F-P8-007 compliance holds. These are not findings.

---

## Critical

### C-1 — impl — TLPKTDROP nextExpected over-advance — arq.go:329-331

**Severity:** Critical
**Category:** impl

The TLPKTDROP handler silently advances `nextExpected` past all undelivered non-dropped lower frames, abandoning them. This violates BC-2.02.006 PC5, which states "only the overdue frame's content is abandoned" — non-dropped frames below the overdue frame must not be silently discarded.

**Ruling:** VALID-IMPL — drop only `overdueSeq`; advance `nextExpected` only when `overdueSeq == nextExpected+1`, then flush `reorderBuf`; never leapfrog over non-dropped frames.

---

## High

### H-1 — scope-split — Sender-side SACK gap-detection/retransmit not implemented — arq.go

**Severity:** High
**Category:** scope-split

Sender-side SACK gap-detection and retransmit (BC-2.02.005 PC2/PC3 and story EC-002) are not implemented.

**Ruling:**
- PC2 gap-DETECTION (pure computation) is IN SCOPE — add pure method `GapsToRetransmit(ackSeq uint32, sackBitmap uint64) []uint32` with no side effects.
- PC3 retransmit-SEND (constructing a new frame and sending it) is DEFERRED to the router/multipath wiring story, following the S-4.01/BC-2.02.009 router wiring deferral precedent.
- EC-002 test must exercise real gap-detection using `GapsToRetransmit`.

Track deferred PC3 retransmit-send as S403-H1-DEFER in open drift register.

---

### H-2 — impl — Concurrent OnAck can reorder channel sends — arq.go

**Severity:** High
**Category:** impl

`OnAck` performs channel sends outside the struct lock, meaning concurrent `OnAck` calls can reorder sends and violate BC-2.02.005 invariant 1 (in-order). The struct documentation falsely claims "safe for concurrent use."

**Ruling:** ARQ is tick-driven single-writer per half-channel (consistent with the `HalfChannel` precedent). REMOVE the false concurrency claim from the struct documentation and document the single-writer contract explicitly.

---

## Medium

### M-1 — impl/test — ErrDuplicateSequence never returned; test is tautological — arq.go

**Severity:** Medium
**Category:** impl/test

`ErrDuplicateSequence` is declared and documented but never returned. The corresponding test accepts "nil OR err" — a tautological assertion that cannot fail.

**Ruling:** Idempotent ACK returns `nil`; remove the dead sentinel `ErrDuplicateSequence`. Tighten the test to assert `nil` exactly.

---

### M-2 — test — TestBC_2_02_005_SACKNotInOuterHeader name over-promises — arq_test.go

**Severity:** Medium
**Category:** test

`TestBC_2_02_005_SACKNotInOuterHeader` only exercises the truncated-header error path; its name implies full BC-2.02.005 SACK coverage but does not deliver it. Actual SACK offset/flag coverage is already provided by `TestARQ_SACKInChannelHeader`.

**Ruling:** Rename `TestBC_2_02_005_SACKNotInOuterHeader` to reflect the truncated-header error path it actually tests. The broader SACK in channel-header coverage is already present in `TestARQ_SACKInChannelHeader`.

---

### M-3 — test — EC-002 test doesn't exercise retransmit — arq_test.go

**Severity:** Medium
**Category:** test

EC-002 test does not exercise retransmit. Folds into H-1 resolution: once `GapsToRetransmit` is implemented, EC-002 test must call it and assert the returned gap list.

---

## Low / Observations

### O-1 — impl — uint32 wraparound — arq.go

**Severity:** Low

uint32 sequence number wraparound unhandled.

**Ruling:** DEFER — document lifetime assumption. Track as S403-O1 in open drift register.

---

### O-2 — impl — Unbounded channel-send is documented caller contract — arq.go

**Severity:** Low

Unbounded channel sends are a documented caller contract.

**Ruling:** ACCEPT as-is; consider setting default channel buffer > 0 as a quality-of-life improvement in a future pass.

---

### O-3 — impl — Unbounded reorderBuf/inFlight — arq.go

**Severity:** Low

`reorderBuf` and `inFlight` maps are unbounded.

**Ruling:** Bound both to the configured window size, consistent with the S-4.02 F-001 family of fixes.

---

### O-4 — spec — DegradationEvent carries single seq vs BC-2.02.006 PC2 "range" — arq.go / spec

**Severity:** Low

`DegradationEvent` carries a single sequence number, but BC-2.02.006 PC2 refers to a "range" of affected frames.

**Ruling:** DEFER — per-frame drop is acceptable for MVP; record a product-owner note. Track as S403-O4 in open drift register.

---

## Process Gaps

None identified.
