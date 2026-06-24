---
artifact_id: adv-S-1.02-pass-03
review_target: S-1.02-halfchannel-clock
producer: adversary
pass: 3
fresh_context: true
findings_count: 7
findings_by_severity: {critical: 0, high: 1, medium: 3, low: 2, nitpick: 1}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 3 — S-1.02

## High

### F-001 — Benchmark ignores `b.N`, producing meaningless ns/op and a constant 10-second run regardless of -benchtime
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:238-270`
- Evidence: `const samples = 1000` with `for i := 0; i < samples; i++ { ... }`. Never references `b.N`. The Go benchmark harness varies `b.N` to stabilize per-op measurements.
- Impact: (a) `ns/op` is junk regardless of what runtime requests; (b) `-benchtime=Nx` and `-benchtime=Ts` flags have no effect; (c) every run takes ~10s minimum; (d) `benchstat` statistical convergence breaks. VP-041's Phase-6 gate cannot scale samples via `b.N`.
- Route: test-writer
- Fix: Replace hardcoded `samples=1000` with `b.N`, so `-benchtime=10000x` actually drives 10k samples.

## Medium

### F-002 — Story AC-001 names a test that no longer exists
- Location: Story `.factory/stories/S-1.02-halfchannel-clock.md:53` vs test `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:12-26`
- Evidence: Story AC-001 says `**Test:** TestHalfChannelTick_OneFramePerCall` but test was renamed to `TestHalfChannelTick_ChanIDPropagation`.
- Impact: Traceability rot. Tooling and human reviewers searching for the AC-001 test by the spec's name will report it as missing.
- Route: product-owner (or story-writer)
- Fix: Update story AC-001 to name `TestHalfChannelTick_ChanIDPropagation`, and add a sentence noting the cardinality clause is enforced structurally by the single-value return type.

### F-003 — VP-017 property test is a tautology: asserts `f.ChanSeq == hc.Seq()` rather than monotonicity
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:409-425`
- Evidence: Test loop body is `f := hc.Tick(); wantSeq := hc.Seq(); if f.ChanSeq != wantSeq { ... }`. Both values are populated from the same `h.seq++` in the impl. An "increment by 2" implementation bug would still pass.
- Impact: VP-017 ("sequence increments by exactly 1") is mis-anchored. The actual step-1 invariant is covered by `TestHalfChannelSequenceIncrement` (line 201-224) which uses a deterministic counter, but that test isn't tagged as the VP-017 harness.
- Route: test-writer
- Fix: Rewrite `TestProperty_VP017_SingleFramePerTick` to capture `f1.ChanSeq` and `f2.ChanSeq` from CONSECUTIVE ticks and assert `f2.ChanSeq - f1.ChanSeq == 1` (uint32 subtraction so wraparound gives 1).

### F-004 — `TestHalfChannelTick_DataFrameType` does not assert that Payload round-trips the enqueued bytes
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:136-155`
- Evidence: Test asserts `frame.FrameType == FrameTypeData` and `len(frame.Payload) > 0`, but never that `frame.Payload` equals `payload`. An impl emitting a fixed sentinel byte for every data tick would pass.
- Impact: Payload identity isn't pinned by any AC-002 test.
- Route: test-writer
- Fix: Add `if !bytes.Equal(frame.Payload, payload) { t.Errorf(...) }`. Requires importing `bytes`.

## Low

### F-005 — Payload zero-copy aliasing semantics under-tested
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:127-139`
- Evidence: Godoc says "The payload is not copied; the caller must not mutate it after passing it to Enqueue." No test asserts this aliasing contract.
- Impact: A future "defensive copy" refactor would silently break the documented zero-copy contract.
- Route: test-writer
- Fix: Add a test asserting `&frame.Payload[0] == &p[0]` (same backing array) after `Enqueue(p)` + `Tick()`.

### F-006 — Direction field is dead — only the accessor verifies storage; no behavior is tested against the field
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:154-160`
- Evidence: Godoc admits "The pure-core HalfChannel does not behave differently by direction; the field exists so effectful upstream code can route by direction." The only test (line 433-453) just round-trips the value through the accessor.
- Impact: YAGNI risk per `.claude/rules/go.md` #2. Borderline acceptable because BC-2.01.003 names upstream/downstream as semantically distinct, but no in-package consumer exists.
- Route: orchestrator (adjudicate retention vs. removal until S-2/S-3 consumes it)
- Fix: Either (a) drop Direction and Direction(); or (b) keep them but add a doc comment naming the future consumer (e.g., S-3.01 ARQ routing).

## Nitpick

### F-007 — TestTickIntervalConstants missing t.Parallel()
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:478-485`
- Evidence: Read-only constant check; every other test in the file uses `t.Parallel()`.
- Impact: None functionally; mildly inconsistent.
- Route: test-writer
- Fix: Add `t.Parallel()` as first line.

## Axes Checked Clean

- EC-001/002/003: all covered.
- VP-018 independence: `TestProperty_VP018_Independence` uses local counters, sound check.
- GC discipline: `pending[0] = nil` before reslice — correct.
- Sentinel error contract: `ErrEmptyPayload` declared, godoc'd, asserted via `errors.Is`.
- No `init()`, no `interface{}`, no panics in lib code.
- Purity: no `time.Now`/`time.Sleep`/goroutines in production source.
- Topological import order (ARCH-08): `internal/frame` only.
- ST1005: no trailing punctuation.
- BC-2.01.005 router opacity: out of scope, correctly disclaimed.
- MinTickInterval/MaxTickInterval pinned by TestTickIntervalConstants.
