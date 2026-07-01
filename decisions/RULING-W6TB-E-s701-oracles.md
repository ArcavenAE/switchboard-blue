---
artifact_id: RULING-W6TB-E-s701-oracles
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
story: S-7.01
pass: Pass-1 LENS-2
closes_findings: [H-1-LENS2, H-2-LENS2, M-3-LENS2]
cycle: v1.0.0-greenfield
---

# Ruling W6TB-E: S-7.01 Oracle Adjudication (Pass-1 LENS-2)

Rulings on three findings surfaced during adversarial Pass-1 LENS-2 (mutation-
resistance review) of S-7.01 XOR parity FEC story.

---

## Finding H-1 (HIGH): AC-004 Composition Tautology

**Adversary claim:** `TestBC_2_02_007_FallbackToARQ_OnMultiLoss` (fec_test.go lines
296–350) is structurally disconnected: (1) `dec.Recover` returns `ErrTooManyLosses`
(lines 314–317); (2) a fresh `arq.New(...)` instance is constructed and
`GapsToRetransmit` called (lines 321–329) independently of result (1). A mutant
that made `Recover` unconditionally return `ErrTooManyLosses`, or that removed the
`GapsToRetransmit` call entirely, could survive the test.

**Assessment — ACCEPTED IN SUBSTANCE, with a narrowed fix obligation:**

The finding is structurally correct. The two halves of AC-004 are causally
unconnected: there is no caller closure that branches on `ErrTooManyLosses` before
invoking `GapsToRetransmit`. The test as written verifies two independent facts
(FEC returns the sentinel; ARQ can report gaps) but does not verify the
**conditional dispatch** that BC-2.02.007 PC-4 requires — "the caller MUST NOT
drop the group silently — it MUST invoke the ARQ SACK/retransmit path."

However, Posture C (defer to wave-level integration test) is premature: the frame-
handler wire-up is not yet in scope, and deferring leaves the composition
precondition entirely unverified for this story's delivery. Posture A (accept as-is)
leaves a mutation-surviving tautology in the suite — AC-003 catches the sentinel
identity but does not cover the dispatch branch.

**Ruling: Posture B — strengthen the oracle in-story.**

The fix is scoped narrowly to the test body. The production `fec.go` and `arq.go`
are NOT touched. The test-writer must replace the unconditional `GapsToRetransmit`
call with a `handle` closure that mirrors the branch a real production caller would
take. The closure must also exercise the negative path (single-loss scenario where
`Recover` returns nil) to verify that `handle` returns nil and does NOT invoke ARQ.

**Test file delta specification (fec_test.go):**

Current structure (lines 319–349):
```
// Caller receives ErrTooManyLosses — it MUST engage the ARQ retransmit path.
// Construct an ARQ sender with all 4 frames in-flight (seq 1..4).
a := arq.New(...)
for seq := uint32(1); seq <= groupSize; seq++ {
    a.EnqueueSend(...)
}
var zeroSACK [arq.SACKBitmapBytes]byte
gaps := a.GapsToRetransmit(0, zeroSACK)
// Assert gaps non-empty and contains lossA+1, lossB+1
```

Required structure (net ~15 LOC addition, ~2 LOC removal):
```go
// Build the ARQ sender once (same 4 in-flight frames).
sendTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
a := arq.New(arq.Config{DropTimeout: 100 * time.Millisecond})
for seq := uint32(1); seq <= groupSize; seq++ {
    a.EnqueueSend(seq, payloads[seq-1], sendTime)
}

// handle models the production caller dispatch: on ErrTooManyLosses, invoke ARQ
// retransmit path; on nil error (single or zero loss), do NOT invoke ARQ.
var zeroSACK [arq.SACKBitmapBytes]byte
handle := func(recoverErr error) []uint32 {
    if errors.Is(recoverErr, arq.ErrTooManyLosses) {
        return a.GapsToRetransmit(0, zeroSACK)
    }
    return nil
}

// Multi-loss path: handle MUST engage ARQ.
gaps := handle(err)
if len(gaps) == 0 {
    t.Fatal("ARQ retransmit fallback: handle returned empty; ...")
}
lossSeqs := map[uint32]bool{uint32(lossA + 1): true, uint32(lossB + 1): true}
gapSet := make(map[uint32]bool, len(gaps))
for _, g := range gaps { gapSet[g] = true }
for seq := range lossSeqs {
    if !gapSet[seq] {
        t.Errorf("ARQ retransmit fallback: lost seq=%d not in gap list %v", seq, gaps)
    }
}

// Negative path: single-loss scenario returns nil err → handle MUST NOT invoke ARQ.
singleGap := make([][]byte, groupSize)
copy(singleGap, payloads)
singleGap[0] = nil
_, singleErr := dec.Recover(singleGap, parity)  // must not be ErrTooManyLosses
nilGaps := handle(singleErr)
if nilGaps != nil {
    t.Errorf("single-loss path: handle should return nil (no ARQ dispatch), got %v", nilGaps)
}
```

**Mutation resistance after fix:** A mutant that unconditionally returns
`ErrTooManyLosses` from `Recover` will fail the negative-path assertion (singleErr
will be `ErrTooManyLosses`, `handle` will return non-nil, test fails). A mutant
that skips `GapsToRetransmit` inside `handle` will fail the `len(gaps)==0` check.

**Story frontmatter delta:** Version bumped from v1.1 → v1.2. AC-004 test body
description updated to note the `handle` closure pattern. No new ACs; no BC
modifications.

---

## Finding H-2 (MEDIUM): Decoder Reuse Across Loss Positions

**Adversary claim:** In `TestBC_2_02_007_Recover_TwoLossesFail` (lines 235–272),
`dec := arq.NewDecoder(...)` is constructed once outside the table-driven subtest
loop and reused across all 4 loss-pair subtests. If `Decoder` ever gained per-call
state (e.g., a loss counter, a buffer, or a cache), reuse would cause subtests to
pollute each other, and the test would miss it.

**Assessment — ACCEPTED as a low-cost hygiene fix.**

The current `Decoder` struct holds only `groupSize int` and carries zero trial
state. The concern is about future brittleness, not a current bug. The fix is
trivially cheap: move `dec := arq.NewDecoder(...)` inside the `for _, tc := range
cases` loop (or inside the `t.Run` closure). The parallel subtest semantics are
preserved. A doc comment on `Decoder` noting it is stateless-per-call is also
acceptable as an alternative, but the constructor-inside-loop approach is
unambiguous and has zero ongoing maintenance burden.

**Ruling: Fix in-story. Move decoder construction inside the subtest loop.**

**Test file delta specification (fec_test.go, lines 238–271):**

Remove line 241:
```go
dec := arq.NewDecoder(arq.FECConfig{GroupSize: groupSize})
```
Add equivalent line inside the `t.Run` closure (after `t.Parallel()`):
```go
dec := arq.NewDecoder(arq.FECConfig{GroupSize: groupSize})
```

Net change: 0 LOC added, 0 LOC removed — purely a relocation within the function.
The variable's scope narrows from function-level to subtest-level.

Note: `TestBC_2_02_007_Recover_SingleLoss` has the same pattern — `enc` and `dec`
constructed once and reused across loss-position subtests (lines 195–196, loop at
201). The single-loss decoder is also stateless, but for consistency the same fix
should be applied: move `enc` and `dec` construction inside the subtest loop OR
confirm in a comment that encoder and decoder are stateless across calls. Because
the encoder IS stateful (it holds a group buffer), the encoder must be constructed
fresh per loss-position subtest. The test is currently correct in its outcome
(encoding happens before the loop, so the parity is reused validly), but the
encoder construction pattern should be documented or restructured. The test-writer
may apply the same inside-loop treatment or add a comment block. This is
**ADVISORY** — not blocking for this ruling.

---

## Finding M-3 (MEDIUM): VP-043 Assertion Count Claim Unverified

**Adversary claim:** `TestBC_2_02_007_VP043_SingleLossRecovery_Property` (line 451)
contains the comment: "7 group sizes × (2+3+4+5+6+7+8) loss positions × 1000
payload variants = 175 000 recovery assertions." The count is a comment claim;
there is no runtime counter or `t.Logf` verification that this count is actually
reached.

**Assessment — ACCEPTED. The count is also arithmetically wrong.**

The claimed 175,000 is incorrect. The actual count is:
- Loss positions per group: groupSize values 2–8 have 2, 3, 4, 5, 6, 7, 8 positions
  respectively. Sum = 35 total (group_size, loss_index) combinations.
- Per combination: 1,000 trials.
- Total recovery assertions: 35 × 1,000 = **35,000** (not 175,000).
- Additionally there are 7 × 1,000 = 7,000 parity oracle checks (one per trial per
  group size), for 42,000 total assertions.

The comment overstates the count by 5×. This is not a cosmetic discrepancy — a
reader trusting the comment would have incorrect beliefs about test coverage breadth.

**Ruling: Fix in-story. Correct the count in the comment AND add a runtime counter.**

**Test file delta specification (fec_test.go, VP-043 test body):**

1. Replace the comment at lines 451–453:

   Old:
   ```
   // Coverage: 7 group sizes × (2+3+4+5+6+7+8) loss positions × 1000 payload
   // variants = 175 000 recovery assertions. The payload bytes are generated with
   // a deterministic Knuth MMIX LCG so runs are reproducible.
   ```
   New:
   ```
   // Coverage: 7 group sizes × Σ(2..8)=35 loss positions × 1000 payload
   // variants = 35 000 recovery assertions; 7 × 1000 = 7 000 parity oracle checks;
   // 42 000 total assertions. The payload bytes are generated with a deterministic
   // Knuth MMIX LCG so runs are reproducible.
   ```

2. Add a runtime assertion counter. Declare `var totalRecovery, totalParity int64`
   before the `for groupSize` loop. Increment `totalRecovery` on each successful
   `dec.Recover` call, `totalParity` on each parity oracle check. After all subtests
   complete (note: subtests run in parallel — use `sync/atomic` for the counters or
   use a `t.Run` wrapper that calls `WaitGroup`). Alternatively, since the count is
   deterministic, add a post-loop `t.Logf` with the computed expected values and a
   final equality assertion.

   Simplest correct approach (avoids race): compute expected counts before the loop
   (deterministic), run loop, and at the very end of the outermost test function
   assert counts reached. Because subtests are spawned with `t.Parallel()`, the
   outer test function returns before subtests complete. The `t.Run` barrier handles
   this: add a final synchronous `t.Run("count_verify", ...)` after the parallel
   groupSize subtests that uses `t.Logf` to report the expected counts and
   `t.Errorf` if the numbers are wrong.

   Concrete implementation left to test-writer's judgment on atomics vs. barrier
   pattern — either is acceptable provided the count is verified at runtime and
   logged. The key constraint: `t.Logf("VP-043 total recovery assertions: %d
   (expected 35000)", n)` followed by `if n != 35000 { t.Errorf(...) }`.

---

## Story Frontmatter Delta

| Field | v1.1 | v1.2 |
|-------|------|------|
| `version` | `"1.1"` | `"1.2"` |
| `modified` | `2026-07-01T00:00:00` | `2026-07-01T00:00:00` (timestamp unchanged, new entry added) |

Story body changes required:
- AC-004 test description: add note that the test uses a `handle` closure to model
  production caller dispatch, and exercises both the positive path (ErrTooManyLosses
  → ARQ engaged) and the negative path (nil err → ARQ NOT engaged).
- Changelog: add v1.2 entry summarizing H-1/H-2/M-3 oracle fixes.

No BC modifications. No new ACs. No VP modifications. The VP-043 property
specification in BC-2.02.007 is unchanged; only the test implementation changes.

---

## Summary

| Finding | ID | Severity | Ruling | Disposition | LOC Impact |
|---------|----|----------|--------|-------------|------------|
| AC-004 composition tautology | H-1 | HIGH | Posture B | Fix in-story: add `handle` closure + negative path | +~15 LOC |
| Decoder reuse across subtests | H-2 | MEDIUM | Fix in-story | Move `NewDecoder` inside subtest loop | 0 net LOC |
| VP-043 count unverified + wrong | M-3 | MEDIUM | Fix in-story | Correct comment (35k not 175k) + runtime counter | +~8 LOC |

**No production code changes.** All fixes are confined to `internal/arq/fec_test.go`.
Story version bumps to v1.2. Test-writer applies all three deltas in the same pass.
