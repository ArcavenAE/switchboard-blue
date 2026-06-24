---
artifact_id: adv-S-1.02-pass-06
review_target: S-1.02-halfchannel-clock
producer: adversary
pass: 6
fresh_context: true
findings_count: 3
findings_by_severity: {critical: 0, high: 1, medium: 1, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 6 — S-1.02

## High

### F-001 — Story EC-003 contradicts BC-2.01.001 EC-002 (payload coalescing dropped silently)
- Location: `.factory/stories/S-1.02-halfchannel-clock.md:90` vs `.factory/specs/behavioral-contracts/ss-01/BC-2.01.001.md:71`; impl `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:106-126`; test `halfchannel_test.go:414-446`.
- Evidence:
  - BC-2.01.001 EC-002: "Multiple payloads accumulate between ticks | All pending payloads up to MTU are coalesced into a single frame on the next tick. Overflow is queued for subsequent ticks."
  - Story EC-003: "Multiple payloads queued before one tick | Single tick emits one frame; remaining payloads queued for subsequent ticks"
  - Implementation: `Tick()` pulls only `h.pending[0]` per call — no coalescing.
  - Test `TestHalfChannelTick_MultiplePayloadsQueuedOneTick` asserts the non-coalescing behavior.
- Impact: Story silently overrides a BC-stated behavior. BC-2.01.001 PC2 + EC-002 implies coalescing up to MTU. Will compound: future stories needing coalescing (throughput optimization) hit a contract mismatch. ARCH-02 §"dequeueUpstream(replay_window)" also reads as one-payload-per-tick, so BC-2.01.001 EC-002 may itself need reconciliation.
- Route: product-owner (decide coalescing fate); architect (verify ARCH-02 alignment).
- Fix: One of: (a) revise BC-2.01.001 EC-002 to match non-coalescing semantic; (b) extend story+impl to coalesce; (c) defer coalescing to a future story with explicit BC note. Recommendation: (c) — coalescing is a throughput optimization, not core MVP behavior.

## Medium

### F-002 — AC-005 mis-anchored: traces to BC-2.01.001 invariant 1 (no-skip / DI-008) but verifies PC4 (jitter / NFR-009)
- Location: `.factory/stories/S-1.02-halfchannel-clock.md:70-72`; target BC `.factory/specs/behavioral-contracts/ss-01/BC-2.01.001.md:53-58`.
- Evidence:
  - Story AC-005 header: "(traces to BC-2.01.001 invariant 1)"
  - AC-005 body records `p99_jitter_ms` over 1,000 ticks; references VP-041 (jitter ≤ 2ms).
  - BC-2.01.001 invariant 1 (DI-008): "The timeslice clock fires on every tick. An implementation that skips empty ticks violates this invariant." — no-skip, not jitter.
  - BC-2.01.001 PC4: "The tick interval is maintained within ±2ms p99 jitter (NFR-009 budget)." — actual target.
- Impact: Semantic anchoring defect. Trace label points to no-skip invariant; AC body + benchmark measure jitter (BC-2.01.001 PC4 / NFR-009 / VP-041).
- Route: product-owner
- Fix: Change AC-005 trace to "(traces to BC-2.01.001 postcondition 4 / NFR-009)". Add a Spec Patches row for pass-6.

## Low

### F-003 — wraparound_test.go not enumerated in story File Structure Requirements
- Location: `.factory/stories/S-1.02-halfchannel-clock.md:161-166`; file `.worktrees/S-1.02/internal/halfchannel/wraparound_test.go`.
- Evidence: File Structure table lists only `halfchannel.go` and `halfchannel_test.go`. Implementation creates a third file `wraparound_test.go` in `package halfchannel` (internal) to access unexported `seq` for EC-002 wraparound coverage.
- Impact: Minor spec drift. Implementation chose the correct technical approach but the story spec didn't anticipate the file split.
- Route: product-owner
- Fix: Add row `| internal/halfchannel/wraparound_test.go | create | EC-002 sequence wraparound test (internal-package, accesses unexported seq) |` to File Structure Requirements.

## Axes Checked Clean

- Test tautology: VP-017/VP-018/VP-051 tests run deterministic loops; each performs meaningful comparison that would catch off-by-one or independence regressions.
- Wraparound coverage: TestSequenceWraparound correctly seeds seq = MaxUint32-1 and verifies wrap to 0 → 1.
- Empty-payload rejection: Both `Enqueue(nil)` and `Enqueue([]byte{})` rejected with ErrEmptyPayload via `errors.Is`.
- Zero-copy contract: TestHalfChannelTick_PayloadZeroCopy asserts backing-array identity.
- GC discipline: `h.pending[0] = nil` before reslice.
- Sequence post-increment: first frame `ChanSeq == 1`; aligns with BC canonical vector.
- ARCH-09 purity (production source): halfchannel.go uses `time` only as Duration; no `time.Now`/`time.Sleep`. Imports only `errors`, `time`, `internal/frame`. Topological order satisfied.
- Benchmark correctness: ResetTimer after allocation; StopTimer before metric aggregation; b.N==0 guard; UTC timestamps; p99 index always `< b.N`.
- ST1005 / error-string style: ErrEmptyPayload message lowercase, no trailing punctuation.
- Pointer receivers consistent.
- No init(), no interface{}/any, no panics in production code.
- FrameType constants aliased from internal/frame.
- Direction godoc cites real downstream consumers (S-3.01, S-4.03, ADR-008).
- All ACs (AC-001 through AC-006) have named tests; all ECs (EC-001/002/003) covered.
- vp_traces (VP-016/017/018/041/051/053) all have named tests or benchmarks.
