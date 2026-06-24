---
artifact_id: adv-S-1.02-pass-01
review_target: S-1.02-halfchannel-clock
producer: adversary
pass: 1
fresh_context: true
findings_count: 9
findings_by_severity: {critical: 2, high: 3, medium: 2, low: 2, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 1 — S-1.02

## Critical

### F-001 — Benchmark measures `Tick()` call-cost, not inter-tick jitter — VP-041 / AC-005 is vacuously satisfied
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:196-224` (`BenchmarkHalfChannelTickJitter`) and helper at lines 16-21 (`tickOnce`).
- Evidence: The benchmark loop is:
  ```go
  for i := range samples {
      latencies[i] = tickOnce(hc)   // tickOnce returns time.Since(t0) AROUND hc.Tick()
  }
  ...
  p99ms := float64(latencies[p99idx]) / float64(time.Millisecond)
  b.ReportMetric(p99ms, "jitter_p99_ms")
  ```
  `tickOnce` records only the wall-clock time spent inside `hc.Tick()` itself, with no sleep between samples. Reported metric is therefore the per-call execution time of a pure state machine (nanoseconds), not the inter-arrival jitter that VP-041 demands.
- VP-041 canonical harness (`.factory/specs/verification-properties/VP-041.md:73-90`) measures `|actual_interval - configured_interval|` with `time.Sleep(configuredInterval - time.Since(prev))` and `actual := now.Sub(prev)`. Two completely different quantities.
- Impact: AC-005 ("p99 jitter of the tick interval must be ≤ 2ms (measured over 1000 ticks in test)") is unsatisfied. The current benchmark will always pass the 2ms gate even on a system that cannot meet the real NFR-009 budget — a textbook "test passes but spec violated" defect.
- Route: test-writer (primary) + implementer.
- Fix: Rewrite the benchmark to follow VP-041's skeleton: use `time.Sleep(interval - elapsed)` between calls and record `|actual_interval - configured_interval|`. The pure-core HalfChannel stays pure; the benchmark, being effectful test glue, may legitimately use `time.Now()` and `time.Sleep` — that does not violate ARCH-09.

### F-002 — AC-002 frame-type and `ParseOuterHeader` assertions are entirely missing; story spec contract not enforced
- Location: Story `.factory/stories/S-1.02-halfchannel-clock.md:54` (AC-002 text) vs. test `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:97-129` (`TestHalfChannelTick_EmptyFrameIsValid`).
- Evidence: AC-002 reads: *"When no payload is queued, `Tick()` produces an empty-tick frame with `frame_type=data` and zero-length payload. The frame is structurally valid (passes `ParseOuterHeader`)."* The test only asserts `len(frame.Payload) == 0`, `frame.ChanID == tc.chanID`, and `frame.ChanSeq == 1`. There is no `ParseOuterHeader` call, and no frame-type field is asserted (the `ChannelFrame` struct at `halfchannel.go:46-51` has no FrameType field at all).
- Compounding evidence: BC-2.01.002 postcondition 2 requires `frame type = EMPTY_TICK`; VP-053 asserts `frames[i].OuterHeader.FrameType == EMPTY_TICK (0x02)`. Story AC-002 says `frame_type=data` (probably a story-spec typo for `empty_tick`); the implementation tags no frame-type at all.
- Impact: A receiver cannot distinguish data frames from empty-tick frames in this implementation. Liveness vs. data is the very semantic BC-2.01.002 is asserting. The implementation silently drops the EMPTY_TICK signal that quality monitoring (BC-2.06.002) depends on.
- Route: product-owner (resolve AC-002 wording: is `frame_type=data` a typo for `empty_tick`? is the `ParseOuterHeader` clause in-scope for S-1.02 or deferred to the outer-assembly story?) → test-writer → implementer (add `FrameType` byte to `ChannelFrame`; `Tick()` sets it to `frame.FrameTypeData` when payload was queued, `frame.FrameTypeEmptyTick` otherwise).
- Fix: Two coherent options. (a) Add `FrameType byte` to `ChannelFrame` and have `Tick()` set it. (b) If outer-header assembly is genuinely out of scope, retire AC-002's `ParseOuterHeader` clause in the story spec and track as a deferred wave-gate finding.

## High

### F-003 — VP-053 / AC-006 has no frame-type assertion; only sequence contiguity is checked
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:232-255` (`TestHalfChannelEmptyTickSequence`).
- Evidence: The test asserts `seqs[0] == 1` and `seqs[i] == seqs[i-1]+1`. It does NOT assert that any of the K frames is structurally an empty-tick frame. Compare to VP-053's spec: for each frame, the harness must assert `OuterHeader.FrameType == EMPTY_TICK`, `PayloadLength == 0`, AND `HMAC is valid`.
- Impact: A regression that emits frames with non-zero payload bytes but contiguous seq numbers would pass this test. The receiver-side gap detection that BC-2.01.002 invariant 1 (DI-008) protects is unverified.
- Route: test-writer.
- Fix: Inside the loop, also assert `len(f.Payload) == 0` for every frame. Once `ChannelFrame` carries a FrameType (per F-002), add an assertion against `frame.FrameTypeEmptyTick`.

### F-004 — VP-016 property test is tautological: it asserts post-increment arithmetic but never validates frame count
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:335-354` (`TestProperty_VP016_SequenceStrictlyMonotonic`).
- Evidence: Test only re-checks the same arithmetic identity as `TestHalfChannelSequenceIncrement`. VP-016's actual property — "exactly one frame per Tick call" — is not testable in Go because the return type is singular `ChannelFrame`, making the property compile-time-true. The test pretends to check a property that is structurally enforced by the type system.
- Impact: False-positive coverage signal. Structurally indistinguishable from VP-017 (lines 361-377).
- Route: test-writer.
- Fix: Collapse `TestProperty_VP016_SequenceStrictlyMonotonic` into `TestProperty_VP017_SingleFramePerTick` with a comment noting VP-016 is enforced by the function signature.

### F-005 — `BenchmarkHalfChannelTickJitter` allocates the latency slice inside the `b.N` loop; `b.ReportMetric` overwrites per iteration
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:199-224`.
- Evidence: `for range b.N { latencies := make([]time.Duration, samples); ... }` re-pays sort cost per outer iteration. No `b.StopTimer`/`b.StartTimer` around sort+percentile.
- Impact: Measurement is contaminated; `b.N` scaling is wrong.
- Route: test-writer.
- Fix: Drop the outer `for range b.N` or use `b.N = 1` semantics; use `b.StopTimer()` around the sort+percentile computation; report the metric exactly once. Folds into F-001 fix.

## Medium

### F-006 — `EC-002` (sequence wraparound) covered only by `t.Skip` — coverage gap in story-declared edge case
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:286-288`.
- Evidence: `t.Skip("EC-002: wraparound covered once VP-016 harness or test constructor variant is available")`. The story declares EC-002 as required at `.factory/stories/S-1.02-halfchannel-clock.md:84` ("Sequence number wraps to 0; no overflow panic"). Trivially fixable: an internal `_test.go` in `package halfchannel` can seed `h.seq = math.MaxUint32 - 1` without changing the public API.
- Impact: Property tests for 10k iterations only stress 0..10_000 — three orders of magnitude away from wraparound.
- Route: test-writer.
- Fix: Add `internal_test.go` (or rename existing test file to `package halfchannel`) containing a wraparound test that seeds `seq = math.MaxUint32 - 1`, ticks 3 times, asserts seq goes MaxUint32 → 0 → 1.

### F-007 — Spec inconsistency between BC-2.01.001 canonical test vector (post-increment) and VP-017/VP-053 harness skeletons (pre-increment) — [process-gap]
- Location:
  - BC-2.01.001 (`.factory/specs/behavioral-contracts/ss-01/BC-2.01.001.md:81`): "sequence 1..10" (post-increment).
  - VP-017 (`.factory/specs/verification-properties/VP-017.md:84-91`): pre-increment.
  - VP-053 (`.factory/specs/verification-properties/VP-053.md:126`): `startSeq+uint32(i)` (pre-increment).
  - Implementation chose post-increment.
- Impact: VP harness skeletons are skeletons but will likely be lifted into the formal verification phase. If lifted unmodified, VP-017/VP-053 will fail against the chosen implementation.
- Route: architect.
- Fix: In `VP-017.md` and `VP-053.md`, rewrite harness skeletons to match BC-2.01.001's post-increment canonical vector.

## Low

### F-008 — Stale "Red Gate" stub-era comment and `_ = t0` blank assignment in `tickOnce` helper
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:13-21` and `391`.
- Evidence: `_ = t0 // consumed here to guard SA4006 during Red Gate; also used in return below` — `t0` is now unambiguously used on the next line; the blank assignment is dead.
- Impact: `ineffassign` (enabled in `.golangci.yml`) may flag this; stale comment misleads readers.
- Route: test-writer.
- Fix: Delete `_ = t0 // ...` (line 18). Update the comment at lines 391-392 to drop the Red-Gate framing.

### F-009 — Blank import of `internal/frame` is unused; either delete or actually use it to tag empty-tick frames
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:11-15`.
- Evidence: `_ "github.com/arcavenae/switchboard/internal/frame"` — nothing in the implementation references `frame.FrameTypeData` or `frame.FrameTypeEmptyTick`.
- Impact: Cosmetic + linter risk. Reinforces F-002.
- Route: implementer.
- Fix: Either delete the blank import or — preferably together with F-002 — use `frame.FrameTypeEmptyTick` / `frame.FrameTypeData` to populate a `FrameType` field on `ChannelFrame`.

## Axes Checked Clean

- Go rule 11 (UTC timestamps): `tickOnce` uses `time.Now().UTC()` — compliant.
- Go rule 5 (ST1005 error strings): `ErrNilPayload = errors.New("halfchannel: payload is nil")` — no trailing punctuation.
- Go rule 10 (no `init()`): none present.
- Go rule 6 (return concrete types): `New` returns `*HalfChannel` — correct.
- Aliasing / GC discipline in `Tick()`: `pending[0] = nil` is set before reslice — correct.
- Purity (ARCH-09): source file imports only `errors`, `time` (data type — permitted), and (blank) `internal/frame`. No goroutines, timers, `time.Now()`, or I/O. Compliant.
- Topological order (ARCH-08 position 7): only internal import is `internal/frame` — compliant.
- Receiver consistency: all methods on `*HalfChannel` — pointer receivers throughout.
- Concurrency claim: documented as not-safe-for-concurrent-use; no sync primitives used — consistent.
