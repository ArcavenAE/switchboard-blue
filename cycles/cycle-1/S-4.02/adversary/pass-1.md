---
story: S-4.02
pass: 1
reviewed_commit: c509504
verdict: NOT_CONVERGED
severity_summary: 1C/1H
streak: 0
streak_target: 3
date: 2026-06-28
---

# S-4.02 Adversarial Review — Pass 1

**Verdict:** NOT_CONVERGED 1C/1H (streak reset to 0/3)
**Tip reviewed:** c509504

---

## Critical

### F-001 — impl — Unbounded `pending` buffer — replay.go:113-117

**Severity:** Critical
**Category:** impl

The `pending` map is unbounded. A buggy or malicious peer can stream frames with higher sequence numbers while never filling the gap at `nextSeq`, growing memory without bound. This violates BC-2.02.004 invariant 3 (memory bounded) and PC5 (once N+1 frames are irrecoverable, delivery is blocked indefinitely with no eviction path).

**Ruling:** VALID-IMPL — bound `pending` to a configured window size; discard frames where `seq >= nextSeq + windowSize` (return appropriate error or nil per design).

---

## High

### F-002 — test — VP-042 latency benchmark is a no-op — replay_test.go:644-660

**Severity:** High
**Category:** test

The VP-042 latency benchmark constructs `New` inside the timed loop and asserts nothing. The AC-004 gate (p99 ≤ 100ms) is therefore never evaluated — VP-042 is unverified.

**Ruling:** VALID-TEST — rewrite the benchmark to measure the steady-state delivery path with `New` constructed outside the timed loop and add a real p99 ≤ 100ms assertion.

---

## Medium

### F-003 — test — TestReplay_VP023_SortedDelivery_Canonical mislabeled — replay_test.go

**Severity:** Medium
**Category:** test

`TestReplay_VP023_SortedDelivery_Canonical` claims to cover loss recovery but exercises only plain in-order delivery. VP-023 concerns sorted delivery under loss; the test does not model any loss or reordering.

**Ruling:** VALID-TEST — rewrite the test to model real loss (a gap in the sequence) followed by recovery (the missing frame arrives out-of-order) to faithfully exercise VP-023.

---

### F-004 — spec/test — Evicted-seq redelivery returns nil not ErrAlreadyDelivered

**Severity:** Medium
**Category:** spec/test

Evicted-sequence redelivery returns `nil` rather than `ErrAlreadyDelivered`. The spec (AC-001) did not explicitly distinguish in-window duplicates from out-of-window redelivery of evicted frames.

**Ruling:** ADJUDICATED — `ErrAlreadyDelivered` applies to in-window duplicates only; out-of-window redelivery of an evicted frame returns `nil` (PC2 exactly-once still holds because the frame was already delivered before eviction). Story AC-001 is to be clarified to make this distinction explicit. Add an evicted-redelivery test confirming the `nil` return.

---

## Low

### F-005 — impl — seq==0 silently discarded, untested — replay.go

**Severity:** Low
**Category:** impl

`seq==0` is silently discarded with no guard and no test. The intent (treat seq==0 as unset/invalid) is not documented.

**Ruling:** VALID-LITE — add an explicit guard that returns `nil` for `seq==0` (treat as unset/discard) and add a test asserting this behavior.

---

### F-006 — impl — uint32 nextSeq wraparound unhandled — replay.go

**Severity:** Low
**Category:** impl

uint32 `nextSeq` wraparound at 2^32 is unhandled and untested. Behavior at wraparound is undefined.

**Ruling:** DEFER — document the lifetime assumption (maximum frames before wraparound) as a drift item. Track as S402-F006 in open drift register.

---

### F-007 — spec — Default N=3 (ARCH-03 line 122) vs N=5 (BC-2.02.004 PC2) — architecture

**Severity:** Low
**Category:** spec

ARCH-03 line 122 documents a default window of N=3, while BC-2.02.004 PC2 specifies N=5 as the default. BC is the behavior authority; ARCH-03 contains a stale value.

**Ruling:** VALID-SPEC — reconcile ARCH-03 to N=5 (BC-2.02.004 is authoritative). Track as S402-F007 in open drift register; architect doc fix required.

---

## Process Gaps

None identified.
