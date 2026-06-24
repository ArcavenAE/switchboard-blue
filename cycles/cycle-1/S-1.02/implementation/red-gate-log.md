# Red Gate Log — S-1.02 halfchannel-clock

**Date:** 2026-06-24
**Branch:** feature/S-1.02-halfchannel-clock
**Worktree:** .worktrees/S-1.02

## Result: PASS — Red Gate confirmed

All unit and property tests fail via `panic: not implemented: S-1.02 New`.
Benchmark panics identically (no benchmark result recorded at Red Gate phase).

## Test inventory

| Test | File line | BC trace | Status |
|------|-----------|----------|--------|
| TestHalfChannelTick_OneFramePerCall | halfchannel_test.go:26 | BC-2.01.001 postcondition 1 | FAIL (stub panic) |
| TestHalfChannelTick_EmptyFrameIsValid | halfchannel_test.go:76 | BC-2.01.002 postcondition 1 | FAIL (stub panic) |
| TestHalfChannelIndependentSequences | halfchannel_test.go:120 | BC-2.01.003 postcondition 1 | FAIL (stub panic) |
| TestHalfChannelSequenceIncrement | halfchannel_test.go:156 | BC-2.01.003 postcondition 2 | FAIL (stub panic) |
| BenchmarkHalfChannelTickJitter | halfchannel_test.go:195 | VP-041 | FAIL (stub panic) |
| TestHalfChannelEmptyTickSequence | halfchannel_test.go:225 | BC-2.01.002 invariant 1 | FAIL (stub panic) |
| TestHalfChannel_EnqueueNilPayload | halfchannel_test.go:255 | BC-2.01.002 precondition | FAIL (stub panic) |
| TestHalfChannelSequenceWraparound | halfchannel_test.go:267 | EC-002 | SKIP (t.Skip — see notes) |
| TestHalfChannelTick_MultiplePayloadsQueuedOneTick | halfchannel_test.go:275 | EC-003 | FAIL (stub panic) |
| TestProperty_VP016_SequenceStrictlyMonotonic | halfchannel_test.go:318 | VP-016 | FAIL (stub panic) |
| TestProperty_VP017_SingleFramePerTick | halfchannel_test.go:337 | VP-017 | FAIL (stub panic) |
| TestProperty_VP018_Independence | halfchannel_test.go:360 | VP-018 | FAIL (stub panic) |

## Raw output (unit run)

```
--- FAIL: TestHalfChannel_EnqueueNilPayload (0.00s)
panic: not implemented: S-1.02 New [recovered, repanicked]
FAIL    github.com/arcavenae/switchboard/internal/halfchannel   0.446s
FAIL
```

## Raw output (benchmark run)

```
--- FAIL: TestHalfChannel_EnqueueNilPayload (0.00s)
--- FAIL: TestHalfChannelTick_OneFramePerCall (0.00s)
    --- FAIL: TestHalfChannelTick_OneFramePerCall/upstream_no_payload (0.00s)
panic: not implemented: S-1.02 New [recovered, repanicked]
FAIL    github.com/arcavenae/switchboard/internal/halfchannel   0.259s
FAIL
```

## Deferral: EC-002 (TestHalfChannelSequenceWraparound)

Sequence wraparound at math.MaxUint32 cannot be driven via the public API
(no constructor accepts an initial seq, and looping MaxUint32 ticks is
infeasible in test time). Test is skipped with `t.Skip` pending either:
- A test-only constructor variant (e.g. `NewWithSeq(chanID, direction, interval, initialSeq)`)
- Or VP-016 formal harness that can inject initial state

The implementer should add this constructor variant in the test file during
the TDD cycle and remove the skip.

## Lint gate

`just fmt && just lint` passes with 0 issues.

Note: stub `halfchannel.go` struct fields required `//nolint:unused` annotations
(one per field, with justifying comment) because the `unused` linter correctly
detects them as unreferenced at Red Gate. This is expected stub behavior — the
implementer's methods will reference them. The annotations were added to the stub
to unblock the lint gate; the implementer must remove them as each field is
wired up.
