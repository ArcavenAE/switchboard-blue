---
artifact_id: adv-S-1.02-pass-04
review_target: S-1.02-halfchannel-clock
producer: adversary
pass: 4
fresh_context: true
findings_count: 5
findings_by_severity: {critical: 0, high: 1, medium: 1, low: 2, nitpick: 1}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 4 — S-1.02

## High

### F-001 — TestProperty_VP018_Independence is mis-anchored: verifies VP-051, not VP-018
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:528-563`
- Evidence: Test header says "VP-018: two channels maintain independent sequence spaces." But VP-018 (`/Users/skippy/work/switchboard-blue/.factory/specs/verification-properties/VP-018.md:27-49`) defines VP-018 as "HalfChannel Emits Empty Frame When No Payload" — the test verifies VP-051 (`/Users/skippy/work/switchboard-blue/.factory/specs/verification-properties/VP-051.md:27`: "HalfChannel Independence — B's Tick Schedule Unaffected by A's Frame Production").
- Impact: Traceability is broken. Phase-6 verification lookups for VP-018 will land on the wrong assertion. VP-018 (empty-frame emission) is only covered by `TestHalfChannelTick_EmptyFrameIsValid` (unit test, not property-style); VP-051 (independence) is covered under VP-018's name.
- Route: test-writer
- Fix: Rename to `TestProperty_VP051_Independence`. Update godoc to "VP-051: two channels maintain independent sequence spaces and clocks." Optionally add a small property-style harness for VP-018 (nil-payload emits empty frame); table-driven over random `(chanID, direction)` cases is sufficient.

## Medium

### F-002 — VP-053 Property Statement references phantom `FlagEmptyTick` flag bit
- Location: `.factory/specs/verification-properties/VP-053.md:38-49`
- Evidence: Property Statement reads `frames[i].Flags & FlagEmptyTick != 0`. BC-2.01.002 PC3 (after pass-02 F-008 patch): "The EMPTY_TICK discriminator does not live in channel-header flags — no EMPTY_TICK bit exists. The empty-tick discriminator lives in `ChannelFrame.FrameType`." The harness skeleton in the same file (lines 103-145) correctly does NOT check Flags.
- Impact: Pass-02 F-008 patch did not propagate from BC-2.01.002 into VP-053's Property Statement. A future implementer reading VP-053's prose would attempt to use the nonexistent flag bit and reintroduce the bug F-008 closed.
- Route: architect
- Fix: Replace `frames[i].Flags & FlagEmptyTick != 0` with `frames[i].FrameType == FrameTypeEmptyTick (0x02)`. Add a revisions-log entry noting alignment with BC-2.01.002 PC3.

## Low

### F-003 — TestProperty_VP017_SingleFramePerTick name conflates VP-016 (cardinality) with VP-017 (increment)
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:450`
- Evidence: Test name "SingleFramePerTick" describes VP-016 (`.factory/specs/verification-properties/VP-016.md:27`: "HalfChannel.Tick Emits Exactly One Frame Per Tick"). Assertion `delta != 1` verifies VP-017 ("Sequence Increments by Exactly 1"). The header comment correctly notes VP-016 is structurally enforced; the name disagrees.
- Impact: Future grep for "VP-016 test" will land here and conclude VP-016 IS runtime-tested.
- Route: test-writer
- Fix: Rename to `TestProperty_VP017_SequenceIncrementsByOne` (matches VP-017.md:74 harness skeleton name `TestProp_Tick_SequenceIncrementsByOne`).

### F-004 — Benchmark measurement methodology biases jitter upward by including Tick overhead
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:276-310`
- Evidence: In the loop, `now := time.Now().UTC()` is captured BEFORE `_ = hc.Tick()`. `prev = now` for the next iteration. Tick's execution time therefore eats into the next iteration's sleep budget. What's measured is "OS sleep accuracy + tick overhead variance," not pure scheduling jitter.
- Impact: Metric is observational in Phase 3 per AC-005, but Phase-6 CI verification on stable hardware may see falsely-elevated jitter and miss the 2ms p99 gate due to Tick overhead being counted.
- Route: test-writer
- Fix: Capture `now` after `hc.Tick()` returns so inter-tick interval includes scheduled work, OR document explicitly in the godoc that this benchmark measures sleep-accuracy + tick-overhead. First option matches NFR-009 end-to-end intent.

## Nitpick

### F-005 — wraparound_internal_test.go uses non-standard _internal_test.go suffix
- Location: `.worktrees/S-1.02/internal/halfchannel/wraparound_internal_test.go`
- Evidence: Filename ends in `_internal_test.go`. Go convention is `_test.go`. The package declaration `package halfchannel` is what makes it an internal test (vs `package halfchannel_test`).
- Impact: None functionally; mildly non-standard.
- Route: implementer
- Fix: Rename to `wraparound_test.go` (or `internal_wraparound_test.go`). Keep `package halfchannel` declaration. Style-only.

## Axes Checked Clean

- AC↔BC traceability: All 6 ACs trace to BC postconditions; AC test names match after F-001 fix.
- Edge cases: EC-001, EC-002, EC-003 all covered.
- Spec patches passes 1–3 present in code.
- Purity (ARCH-09): only `errors`, `time` (Duration), `internal/frame` imports in production source. Compliant.
- Dependency graph (ARCH-08): position 7 → imports internal/frame only.
- Zero-copy contract: documented and pinned by `TestHalfChannelTick_PayloadZeroCopy`.
- Wraparound: covered by internal-package test; uint32 modular arithmetic correct.
- Error sentinel hygiene: ErrEmptyPayload exported, godoc'd, used via errors.Is. No trailing punctuation.
- No `init()`, no `interface{}`, no panics in lib code.
- Pointer receivers consistent.
- FrameType aliasing via internal/frame avoids constant drift.
- Direction accessor documented for S-3.01/S-4.03/ADR-008 future consumers.
- Race detector compatibility: single-goroutine contract documented.
- Imports order gofumpt-compliant.
