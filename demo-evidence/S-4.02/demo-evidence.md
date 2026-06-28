# Demo Evidence — S-4.02: Upstream Idempotent Replay Window

## Header

| Field | Value |
|-------|-------|
| Story | S-4.02 |
| Tip SHA | `73781a484e238c06482fec4f4f5c47e97fdc410b` |
| Worktree | `.worktrees/S-4.02` |
| Date captured | 2026-06-28 |
| Race-clean | YES — `go test -race ./internal/replay/...` passed in 1.423s with no data race warnings |
| Recording method | Test-transcript-based (pure-core library; no UI surface) |

## Rationale for Test-Transcript Evidence

`internal/replay` is a pure-core, in-memory state machine with no I/O, no UI, and no binary
entry point. There is no surface to record with VHS or Playwright. Per the S-W3.04 / S-4.01
precedent established in this project, library stories of this type use captured `go test -v`
and `go test -race` output as demo evidence, with test functions mapped verbatim to each
acceptance criterion.

---

## AC Coverage Table

| AC | BC Trace | Test Function(s) | PASS |
|----|----------|-----------------|------|
| AC-001 | BC-2.02.004 PC2 (no duplicate delivery) | `TestReplay_NoDuplicateDelivery`, `TestReplay_NoDuplicateDelivery_MultipleSeqs`, `TestReplay_VP022_NoDoubleDelivery_Property`, `TestReplay_EC002_AllWindowFramesResent`, `TestReplay_EvictedSeqRedeliveryReturnsNil`, `TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered` | PASS |
| AC-002 | BC-2.02.004 invariant 2 (in-order delivery) | `TestReplay_InOrderDelivery`, `TestReplay_InOrderDelivery_LongerGap`, `TestReplay_InOrderDelivery_TableDriven` (4 sub-cases), `TestReplay_VP023_InOrderDelivery_Property`, `TestReplay_VP023_SortedDelivery_Canonical`, `TestReplay_EC003_GapBufferedThenFilled`, `TestReplay_BC_2_02_004_invariant_window_monotonic_seqs` | PASS |
| AC-003 | BC-2.02.004 invariant 3 + invariant 5 (fixed-window discard) | `TestReplay_WindowBoundary`, `TestReplay_WindowBoundary_ExactBoundarySeq`, `TestReplay_DistWindowSizeBoundary` (2 sub-cases), `TestReplay_BoundedPendingBuffer`, `TestReplay_SeqZeroDiscarded` | PASS |
| AC-004 | BC-2.02.004 invariant 5 (bounded-state per-call latency ≤ 1µs median) | `TestReplay_OnUpstream_MedianPerCall` | PASS |

---

## Per-AC Evidence

### AC-001 — Dedup Exactly-Once (BC-2.02.004 PC2)

**Requirement:** `Replay.OnUpstream(frame)` never delivers the same sequence number twice.
Second delivery of the same seq returns `ErrAlreadyDelivered`.

**Test functions and PASS lines:**

```
--- PASS: TestReplay_NoDuplicateDelivery (0.00s)
--- PASS: TestReplay_NoDuplicateDelivery_MultipleSeqs (0.00s)
--- PASS: TestReplay_VP022_NoDoubleDelivery_Property (0.00s)
--- PASS: TestReplay_EC002_AllWindowFramesResent (0.00s)
--- PASS: TestReplay_EvictedSeqRedeliveryReturnsNil (0.00s)
--- PASS: TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered (0.00s)
```

**Coverage notes:**
- `TestReplay_NoDuplicateDelivery`: canonical case — seq 1 delivered once, second call returns `ErrAlreadyDelivered`; deliver callback not invoked again
- `TestReplay_NoDuplicateDelivery_MultipleSeqs`: seqs 1–5 delivered in order, then each re-sent; all return `ErrAlreadyDelivered`; delivery count stays at 5
- `TestReplay_VP022_NoDoubleDelivery_Property`: 1,000 randomised permutations with duplicates (seed=42); VP-022 verified: no seq appears in delivery log more than once across any scenario
- `TestReplay_EC002_AllWindowFramesResent`: EC-002 — all N=5 window frames re-sent; all return `ErrAlreadyDelivered`; delivery count unchanged
- `TestReplay_EvictedSeqRedeliveryReturnsNil`: evicted-from-window seq returns nil (silent discard), not `ErrAlreadyDelivered`, and deliver not called again
- `TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered`: in-window duplicate still returns `ErrAlreadyDelivered` (complement of evicted case)

---

### AC-002 — In-Order Delivery / Out-of-Order Buffering (BC-2.02.004 invariant 2)

**Requirement:** `Replay.OnUpstream` delivers keystrokes in sequence order. If seq N+1
arrives before N, N+1 is buffered and delivered after N arrives.

**Test functions and PASS lines:**

```
--- PASS: TestReplay_InOrderDelivery (0.00s)
--- PASS: TestReplay_InOrderDelivery_LongerGap (0.00s)
--- PASS: TestReplay_InOrderDelivery_TableDriven (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/strict_order (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/single (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/interleaved (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/reverse_order (0.00s)
--- PASS: TestReplay_VP023_InOrderDelivery_Property (0.00s)
--- PASS: TestReplay_VP023_SortedDelivery_Canonical (0.00s)
--- PASS: TestReplay_EC003_GapBufferedThenFilled (0.00s)
--- PASS: TestReplay_BC_2_02_004_invariant_window_monotonic_seqs (0.00s)
```

**Coverage notes:**
- `TestReplay_InOrderDelivery`: seq=2 arrives before seq=1; seq=2 buffered (0 deliveries), then seq=1 fills gap and both drain in order [1, 2]
- `TestReplay_InOrderDelivery_LongerGap`: seqs 5,4,3 arrive before seq=2; only seq=1 delivered; seq=2 arrival drains [2,3,4,5] in order
- `TestReplay_InOrderDelivery_TableDriven`: four arrival permutations (strict, reverse, interleaved, single) all deliver [1..n] in ascending order
- `TestReplay_VP023_InOrderDelivery_Property`: 1,000 randomised permutations (seed=137); VP-023 verified: delivered frames always in strictly ascending seq order
- `TestReplay_VP023_SortedDelivery_Canonical`: BC-2.02.004 canonical test vector — seq=11 arrives while seq=10 is "lost"; seq=11 buffered; seq=10 recovered from replay window; delivery order [1..10, 11]
- `TestReplay_EC003_GapBufferedThenFilled`: EC-003 — seqs 3,4,5 buffered while seq=2 absent; seq=2 fills gap, drains [2,3,4,5] in order
- `TestReplay_BC_2_02_004_invariant_window_monotonic_seqs`: 1,000 random permutations of seqs 1..15 with windowSize=5; delivered set is always a sorted, contiguous prefix starting at 1 (no internal gaps)

---

### AC-003 — Fixed-Window Discard (BC-2.02.004 invariant 3 + invariant 5)

**Requirement:** The replay window carries the last N keystrokes (N configurable). Frames
older than the window are discarded without error (nil return). Frames with distance ≥
windowSize from delivery frontier are discarded, not buffered.

**Test functions and PASS lines:**

```
--- PASS: TestReplay_WindowBoundary (0.00s)
--- PASS: TestReplay_WindowBoundary_ExactBoundarySeq (0.00s)
--- PASS: TestReplay_DistWindowSizeBoundary (0.00s)
    --- PASS: TestReplay_DistWindowSizeBoundary/dist=4_(windowSize-1)_is_buffered (0.00s)
    --- PASS: TestReplay_DistWindowSizeBoundary/dist=5_(==windowSize)_is_discarded (0.00s)
--- PASS: TestReplay_BoundedPendingBuffer (0.00s)
--- PASS: TestReplay_SeqZeroDiscarded (0.00s)
```

**Coverage notes:**
- `TestReplay_WindowBoundary`: windowSize=5; seqs 1–6 delivered; seq=1 then outside window — returns nil (silent discard), no extra delivery
- `TestReplay_WindowBoundary_ExactBoundarySeq`: EC-001 — seq exactly at boundary evicted when seq=4 advances window (windowSize=3); seq=1 post-eviction returns nil; seq=2 still in-window returns `ErrAlreadyDelivered`
- `TestReplay_DistWindowSizeBoundary/dist=4_(windowSize-1)_is_buffered`: dist = windowSize-1 (=4 < 5) must be buffered, not discarded; drains automatically when gap filled
- `TestReplay_DistWindowSizeBoundary/dist=5_(==windowSize)_is_discarded`: dist == windowSize (=5) must be discarded (nil return); explicit re-delivery after gap is not `ErrAlreadyDelivered` (was discarded, not recorded)
- `TestReplay_BoundedPendingBuffer`: far-future seq=50 (dist=48 >> windowSize=5) sent when nextSeq=2; returns nil; delivering seq=2..49 does NOT auto-drain seq=50 from pending (it was discarded, not buffered); nextSeq confirms at 50; explicit seq=50 delivery succeeds
- `TestReplay_SeqZeroDiscarded`: seq=0 (unset/invalid sentinel) returns nil and is not delivered; subsequent seq=1 works normally

---

### AC-004 — Bounded-State Per-Call Latency Guard (BC-2.02.004 invariant 5)

**Requirement:** `Replay.OnUpstream` per-call overhead ≤ 1µs (median) under no-contention
conditions with windowSize=64 and pre-warmed window, over 10,000 iterations.

**Test functions and PASS lines:**

```
--- PASS: TestReplay_OnUpstream_MedianPerCall (0.00s)
```

**Coverage notes:**
- `TestReplay_OnUpstream_MedianPerCall`: windowSize=64; pre-warms seqs 1..64; measures 10,000 steady-state sequential calls; sorts samples; asserts median ≤ 1µs (time.Microsecond). Test passed — no O(N²) or allocation regression detected.
- The companion `BenchmarkReplay_OnUpstream_PerCall` is available for deeper latency profiling (`go test -bench=BenchmarkReplay_OnUpstream_PerCall -benchtime=10s`).
- Note: VP-042 (keystroke-to-echo p99 ≤ 100ms) is out of scope here; it is verified at the `internal/halfchannel` integration level per RULING-002.

---

## Adversarial / Regression Tests (Bonus Coverage)

The following tests from `wraparound_test.go` and `pass3_test.go` cover adversarial findings
from pass-2 and pass-3 review (not directly AC-assigned but strengthening AC-002, AC-003, AC-004):

```
--- PASS: TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered (0.00s)
--- PASS: TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap (0.00s)
--- PASS: TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow (0.00s)
--- PASS: TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow (0.00s)
```

| Test | Finding | AC Strengthened |
|------|---------|----------------|
| `TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered` | F-001 (pass-2): uint32 addition overflow near MaxUint32 would discard in-window frame; fixed by wrap-safe modular distance | AC-002, AC-003 |
| `TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap` | F-001 (v1.3): pending/seen maps bounded under sustained never-filling-gap traffic; len(pending) ≤ windowSize-1, len(seen) ≤ windowSize after every call | AC-003, AC-004 |
| `TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow` | F-001 (pass-3): non-wrap-safe lower-bound guard misclassified in-window future frames near MaxUint32 as "too-old"; fixed by removing separate lower-bound guard in favour of unified modular-distance check | AC-002, AC-003 |
| `TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow` | pass-3: seen-set eviction path not exercised when nextSeq never advances; 5,000-frame advancing-window stream verifies len(seen) ≤ windowSize and len(pending) ≤ windowSize-1 after every call | AC-003, AC-004 |

---

## Raw Test Run Transcripts

### Verbose run (`go test -v ./internal/replay/...`)

```
=== RUN   TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow
=== PAUSE TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow
=== RUN   TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow
=== PAUSE TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow
=== RUN   TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered
=== PAUSE TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered
=== RUN   TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap
=== PAUSE TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap
=== RUN   TestReplay_NoDuplicateDelivery
=== PAUSE TestReplay_NoDuplicateDelivery
=== RUN   TestReplay_NoDuplicateDelivery_MultipleSeqs
=== PAUSE TestReplay_NoDuplicateDelivery_MultipleSeqs
=== RUN   TestReplay_InOrderDelivery
=== PAUSE TestReplay_InOrderDelivery
=== RUN   TestReplay_InOrderDelivery_LongerGap
=== PAUSE TestReplay_InOrderDelivery_LongerGap
=== RUN   TestReplay_InOrderDelivery_TableDriven
=== PAUSE TestReplay_InOrderDelivery_TableDriven
=== RUN   TestReplay_WindowBoundary
=== PAUSE TestReplay_WindowBoundary
=== RUN   TestReplay_WindowBoundary_ExactBoundarySeq
=== PAUSE TestReplay_WindowBoundary_ExactBoundarySeq
=== RUN   TestReplay_DistWindowSizeBoundary
=== PAUSE TestReplay_DistWindowSizeBoundary
=== RUN   TestReplay_EC002_AllWindowFramesResent
=== PAUSE TestReplay_EC002_AllWindowFramesResent
=== RUN   TestReplay_EC003_GapBufferedThenFilled
=== PAUSE TestReplay_EC003_GapBufferedThenFilled
=== RUN   TestReplay_VP022_NoDoubleDelivery_Property
=== PAUSE TestReplay_VP022_NoDoubleDelivery_Property
=== RUN   TestReplay_VP023_InOrderDelivery_Property
=== PAUSE TestReplay_VP023_InOrderDelivery_Property
=== RUN   TestReplay_VP023_SortedDelivery_Canonical
=== PAUSE TestReplay_VP023_SortedDelivery_Canonical
=== RUN   TestReplay_WindowSize
=== PAUSE TestReplay_WindowSize
=== RUN   TestReplay_NextSeq
=== PAUSE TestReplay_NextSeq
=== RUN   TestReplay_New_PanicsOnZeroWindowSize
=== PAUSE TestReplay_New_PanicsOnZeroWindowSize
=== RUN   TestReplay_New_PanicsOnNilDeliver
=== PAUSE TestReplay_New_PanicsOnNilDeliver
=== RUN   TestReplay_BC_2_02_004_invariant_window_monotonic_seqs
=== PAUSE TestReplay_BC_2_02_004_invariant_window_monotonic_seqs
=== RUN   TestReplay_OnUpstream_MedianPerCall
=== PAUSE TestReplay_OnUpstream_MedianPerCall
=== RUN   TestReplay_BoundedPendingBuffer
=== PAUSE TestReplay_BoundedPendingBuffer
=== RUN   TestReplay_EvictedSeqRedeliveryReturnsNil
=== PAUSE TestReplay_EvictedSeqRedeliveryReturnsNil
=== RUN   TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered
=== PAUSE TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered
=== RUN   TestReplay_SeqZeroDiscarded
=== PAUSE TestReplay_SeqZeroDiscarded
=== CONT  TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow
--- PASS: TestReplay_BC_2_02_004_WraparoundTooOldGuardBuffersInWindow (0.00s)
=== CONT  TestReplay_InOrderDelivery
--- PASS: TestReplay_InOrderDelivery (0.00s)
=== CONT  TestReplay_VP022_NoDoubleDelivery_Property
=== CONT  TestReplay_SeqZeroDiscarded
--- PASS: TestReplay_SeqZeroDiscarded (0.00s)
=== CONT  TestReplay_New_PanicsOnZeroWindowSize
--- PASS: TestReplay_New_PanicsOnZeroWindowSize (0.00s)
=== CONT  TestReplay_NextSeq
--- PASS: TestReplay_NextSeq (0.00s)
=== CONT  TestReplay_WindowSize
=== CONT  TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered
--- PASS: TestReplay_InWindowDuplicateReturnsErrAlreadyDelivered (0.00s)
=== CONT  TestReplay_VP023_SortedDelivery_Canonical
--- PASS: TestReplay_VP023_SortedDelivery_Canonical (0.00s)
=== CONT  TestReplay_VP023_InOrderDelivery_Property
=== CONT  TestReplay_EvictedSeqRedeliveryReturnsNil
--- PASS: TestReplay_EvictedSeqRedeliveryReturnsNil (0.00s)
=== CONT  TestReplay_DistWindowSizeBoundary
=== RUN   TestReplay_DistWindowSizeBoundary/dist=4_(windowSize-1)_is_buffered
=== PAUSE TestReplay_DistWindowSizeBoundary/dist=4_(windowSize-1)_is_buffered
=== RUN   TestReplay_DistWindowSizeBoundary/dist=5_(==windowSize)_is_discarded
=== PAUSE TestReplay_DistWindowSizeBoundary/dist=5_(==windowSize)_is_discarded
=== CONT  TestReplay_EC003_GapBufferedThenFilled
--- PASS: TestReplay_EC003_GapBufferedThenFilled (0.00s)
=== CONT  TestReplay_EC002_AllWindowFramesResent
--- PASS: TestReplay_EC002_AllWindowFramesResent (0.00s)
=== CONT  TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap
=== CONT  TestReplay_BoundedPendingBuffer
--- PASS: TestReplay_BoundedPendingBuffer (0.00s)
=== CONT  TestReplay_NoDuplicateDelivery_MultipleSeqs
--- PASS: TestReplay_NoDuplicateDelivery_MultipleSeqs (0.00s)
=== CONT  TestReplay_NoDuplicateDelivery
--- PASS: TestReplay_NoDuplicateDelivery (0.00s)
=== CONT  TestReplay_WindowBoundary
--- PASS: TestReplay_WindowBoundary (0.00s)
=== CONT  TestReplay_WindowBoundary_ExactBoundarySeq
--- PASS: TestReplay_WindowBoundary_ExactBoundarySeq (0.00s)
=== CONT  TestReplay_InOrderDelivery_TableDriven
=== RUN   TestReplay_InOrderDelivery_TableDriven/strict_order
=== PAUSE TestReplay_InOrderDelivery_TableDriven/strict_order
=== RUN   TestReplay_InOrderDelivery_TableDriven/reverse_order
=== PAUSE TestReplay_InOrderDelivery_TableDriven/reverse_order
=== RUN   TestReplay_InOrderDelivery_TableDriven/interleaved
=== PAUSE TestReplay_InOrderDelivery_TableDriven/interleaved
=== RUN   TestReplay_InOrderDelivery_TableDriven/single
=== PAUSE TestReplay_InOrderDelivery_TableDriven/single
=== CONT  TestReplay_OnUpstream_MedianPerCall
=== CONT  TestReplay_BC_2_02_004_invariant_window_monotonic_seqs
--- PASS: TestReplay_BC_2_02_004_BoundedStateUnderNeverFillingGap (0.00s)
=== CONT  TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow
=== CONT  TestReplay_New_PanicsOnNilDeliver
=== RUN   TestReplay_WindowSize/#00
=== PAUSE TestReplay_WindowSize/#00
=== RUN   TestReplay_WindowSize/#01
=== PAUSE TestReplay_WindowSize/#01
=== RUN   TestReplay_WindowSize/#02
=== CONT  TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered
=== CONT  TestReplay_InOrderDelivery_LongerGap
--- PASS: TestReplay_New_PanicsOnNilDeliver (0.00s)
--- PASS: TestReplay_InOrderDelivery_LongerGap (0.00s)
=== CONT  TestReplay_DistWindowSizeBoundary/dist=4_(windowSize-1)_is_buffered
=== PAUSE TestReplay_WindowSize/#02
=== CONT  TestReplay_InOrderDelivery_TableDriven/strict_order
=== CONT  TestReplay_InOrderDelivery_TableDriven/single
=== CONT  TestReplay_InOrderDelivery_TableDriven/interleaved
=== CONT  TestReplay_InOrderDelivery_TableDriven/reverse_order
--- PASS: TestReplay_InOrderDelivery_TableDriven (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/strict_order (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/single (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/interleaved (0.00s)
    --- PASS: TestReplay_InOrderDelivery_TableDriven/reverse_order (0.00s)
=== CONT  TestReplay_DistWindowSizeBoundary/dist=5_(==windowSize)_is_discarded
=== CONT  TestReplay_WindowSize/#00
=== CONT  TestReplay_WindowSize/#02
=== CONT  TestReplay_WindowSize/#01
--- PASS: TestReplay_WindowSize (0.00s)
    --- PASS: TestReplay_WindowSize/#00 (0.00s)
    --- PASS: TestReplay_WindowSize/#02 (0.00s)
    --- PASS: TestReplay_WindowSize/#01 (0.00s)
--- PASS: TestReplay_BC_2_02_004_WraparoundInWindowFrameBuffered (0.00s)
--- PASS: TestReplay_DistWindowSizeBoundary (0.00s)
    --- PASS: TestReplay_DistWindowSizeBoundary/dist=5_(==windowSize)_is_discarded (0.00s)
    --- PASS: TestReplay_DistWindowSizeBoundary/dist=4_(windowSize-1)_is_buffered (0.00s)
--- PASS: TestReplay_OnUpstream_MedianPerCall (0.00s)
--- PASS: TestReplay_VP023_InOrderDelivery_Property (0.00s)
--- PASS: TestReplay_VP022_NoDoubleDelivery_Property (0.00s)
--- PASS: TestReplay_BC_2_02_004_invariant_window_monotonic_seqs (0.00s)
--- PASS: TestReplay_BC_2_02_004_SeenBoundedUnderAdvancingWindow (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/replay	(cached)
```

### Race detector run (`go test -race -count=1 ./internal/replay/...`)

```
ok  	github.com/arcavenae/switchboard/internal/replay	1.423s
```

No data race warnings. Exit code 0.

---

## Summary

| AC | Status | Test Count |
|----|--------|-----------|
| AC-001 (dedup exactly-once) | PASS | 6 test functions |
| AC-002 (in-order delivery) | PASS | 7 test functions (11 including sub-cases) |
| AC-003 (fixed-window discard) | PASS | 6 test functions (8 including sub-cases) |
| AC-004 (bounded-state latency ≤ 1µs) | PASS | 1 test function |
| Race-clean | PASS | full suite, 1.423s |

**All 4 ACs: PASS. Race-clean: YES.**
