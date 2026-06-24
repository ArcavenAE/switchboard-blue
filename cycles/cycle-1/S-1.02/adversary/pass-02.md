---
artifact_id: adv-S-1.02-pass-02
review_target: S-1.02-halfchannel-clock
producer: adversary
pass: 2
fresh_context: true
findings_count: 11
findings_by_severity: {critical: 0, high: 4, medium: 4, low: 3, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 2 — S-1.02

## High

### F-001 — AC-005 jitter benchmark does not assert the 2ms p99 gate (silent regression)
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:217-257`
- Evidence: `b.ReportMetric(float64(p99)/float64(time.Millisecond), "jitter_p99_ms")` with no `b.Errorf`/`b.Fatalf`. VP-041 spec harness (`.factory/specs/verification-properties/VP-041.md:101-103`) has `if p99 > maxP99Jitter { b.Errorf(...) }`. `just test` runs `go test ./... -v` without `-bench` flag (`justfile:30-31`), so the benchmark is not executed in CI.
- Impact: Story AC-005 (`.factory/stories/S-1.02-halfchannel-clock.md:67-69`) explicitly says "p99 jitter ... must be ≤ 2ms ... This is the benchmark gate for VP-041." The benchmark cannot fail; a regression to 50ms jitter would pass. AC-005 is functionally unverified.
- Route: product-owner (or test-writer if story intent is to defer gate to Phase 6)
- Fix: Either (a) add `if p99 > 2*time.Millisecond { b.Errorf(...) }` and wire `-bench` into a CI job; or (b) revise story AC-005 to say "metric reporting only, gate deferred to Phase 6" and drop the "benchmark gate" phrasing.

### F-002 — FrameType constants duplicated in halfchannel; story spec mandates importing internal/frame
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:7-19`
- Evidence: `halfchannel.go` imports only `errors` and `time`. Defines `FrameTypeData byte = 0x01` and `FrameTypeEmptyTick byte = 0x02` locally with a "keep in sync" comment. `internal/frame/frame.go:27-33` already exports identical constants. Story (`.factory/stories/S-1.02-halfchannel-clock.md:142`) Architecture Compliance Rule: "internal/halfchannel imports ONLY internal/frame".
- Impact: (1) Spec drift — implementation does not match architectural constraint. (2) Maintenance hazard — wire-format byte-value drift between two const blocks. (3) Untyped vs byte-typed constants may cause subtle conversion issues.
- Route: implementer
- Fix: `import "github.com/arcavenae/switchboard/internal/frame"`, define `FrameTypeData = frame.FrameTypeData`, `FrameTypeEmptyTick = frame.FrameTypeEmptyTick` (alias). OR drop halfchannel constants entirely and reference `frame.FrameTypeData` directly.

### F-003 — Enqueue nil-payload test does not assert the ErrNilPayload sentinel
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:307-316`
- Evidence: Test asserts only `err != nil`. No `errors.Is(err, halfchannel.ErrNilPayload)`. The sentinel is declared at `halfchannel.go:64` as the contract for BC-2.01.002 precondition.
- Impact: A regression where `Enqueue(nil)` returns `fmt.Errorf("payload bad")` would pass this test silently. `.claude/rules/go.md` §"Error handling" requires `errors.Is()`, never string matching. Sentinel is public API; nothing pins it.
- Route: test-writer
- Fix: `if !errors.Is(err, halfchannel.ErrNilPayload) { t.Errorf(...) }`.

### F-004 — Direction field stored but never read; no accessor and never validated
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:74, 84-90`
- Evidence: `direction` field set in `New` but never read by any method. No public `Direction()` accessor. Tests only PASS direction to `New`, never read it back. No bound check (`Direction(99)` passes silently).
- Impact: (a) Downstream code that needs "am I upstream or downstream?" cannot ask. (b) No validation. (c) The `Direction` type is exported public API with no operational meaning today.
- Route: implementer (decide intent), then test-writer
- Fix: If direction is forward-compat metadata, add `Direction()` accessor and a test. If unused, drop the field and the parameter.

## Medium

### F-005 — TickInterval() accessor has zero test coverage
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:141-143`
- Evidence: No test invokes `TickInterval()`.
- Impact: A regression where `TickInterval()` returns `0` or `h.seq` (typo) is not caught.
- Route: test-writer
- Fix: Add `if got := hc.TickInterval(); got != 10*time.Millisecond { t.Errorf(...) }`.

### F-006 — AC-001 "exactly one frame per call" test is tautological by author's own admission
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:11-13, 21-80`
- Evidence: Top-of-file comment says "VP-016 is enforced structurally by func (h *HalfChannel) Tick() ChannelFrame — a singular return value cannot return zero or more than one frame. No runtime test asserts this; the type system does." `TestHalfChannelTick_OneFramePerCall` checks only `frame.ChanID == tc.chanID`.
- Impact: AC-001 (`.factory/stories/S-1.02-halfchannel-clock.md:51-53`) claims `TestHalfChannelTick_OneFramePerCall` validates BC-2.01.001 postcondition 1. The test name implies an assertion that the test does not make.
- Route: test-writer or product-owner (which way to reconcile)
- Fix: Rename test to `TestHalfChannelTick_ChanIDPropagation` and amend AC-001 trace to say "verified by Go type system; no runtime test"; OR add a structural assertion at runtime (verbose but explicit).

### F-007 — Tick() with explicit []byte{} (non-nil empty slice) emits FrameTypeData, contradicting BC-2.01.002 intent
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:97-117, 122-130`
- Evidence: `Enqueue` rejects only `payload == nil`. After `hc.Enqueue([]byte{})`, `Tick()` emits a frame with `FrameType=FrameTypeData` and `len(Payload)==0`. BC-2.01.002 PC2 says zero-length payload → `FrameType = EMPTY_TICK`.
- Impact: An upstream caller that legitimately enqueues a zero-byte message produces a frame the receiver classifies as application data with empty payload. The asymmetry "nil → empty-tick, []byte{} → data with len=0" is not documented anywhere.
- Route: product-owner (clarify BC-2.01.002), then test-writer + implementer
- Fix: Either (a) reject `len(payload)==0` in Enqueue; (b) document and test the divergence; (c) use payload-byte-length, not pending-queue length, to set FrameType. Add a test pinning the chosen behavior.

### F-008 — BC-2.01.002 PC3 cites an "EMPTY_TICK indicator" flag bit that ARCH-02 channel-header layout does not define [process-gap]
- Location: spec contradiction
  - `.factory/specs/behavioral-contracts/ss-01/BC-2.01.002.md:54` — PC3: "channel header is fully populated (channel ID, sequence number, flags (FEC_present=0, EMPTY_TICK indicator))"
  - `.factory/specs/architecture/ARCH-02-protocol-stack.md:119` — channel header flags: "bit 0=FEC_present, bit 1=ARQ_req, bit 2=SACK_present" (no EMPTY_TICK bit)
  - `.factory/specs/behavioral-contracts/ss-01/BC-2.01.005.md:60` — same as ARCH-02
- Evidence: Implementation hardcodes `Flags: 0`. This matches ARCH-02/BC-2.01.005 but contradicts BC-2.01.002 PC3.
- Impact: A future implementer reading BC-2.01.002 in isolation will look for an EMPTY_TICK flag bit, find none in ARCH-02, and either invent one (breaking layout) or leave the contradiction unresolved.
- Route: product-owner
- Fix: Reword BC-2.01.002 PC3 to: "channel header is fully populated (chan_id, chan_seq, flags=0; the EMPTY_TICK discriminator lives in `ChannelFrame.FrameType` and is propagated to the outer-header `frame_type` byte by the outer-assembler per ARCH-09 boundary)."

## Low

### F-009 — Exported MinTickInterval / MaxTickInterval constants have no consumers
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:32-38`
- Evidence: Grep finds only the declaration. No use in `New`, tests, benchmark, or any other package.
- Impact: Dead exported API. Suggests intent (validation in `New`) was abandoned. golangci `unused` linter does not flag exported identifiers.
- Route: implementer
- Fix: Either drop them, validate `tickInterval` in `New` against them, or add an explanatory comment + a constant-equality test pinning the documented values.

### F-010 — prev := time.Now().UTC() placed before b.ResetTimer() in jitter benchmark
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:231-234`
- Evidence: `prev := time.Now().UTC()` captured before `b.ResetTimer()`. First `actual := now.Sub(prev)` absorbs setup time.
- Impact: First-sample deviation artificially inflated by setup cost. (p99 over 1000 samples is robust to one outlier, so the metric is largely correct, but noise floor is worse than necessary.)
- Route: test-writer
- Fix: Move `prev := time.Now().UTC()` after `b.ResetTimer()`.

### F-011 — TestHalfChannelIndependentSequences tests two-struct-instance independence, not upstream-vs-downstream
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:158-180`
- Evidence: Test creates two channels with DIFFERENT chanIDs (100, 200). What's tested is "two separate HalfChannel instances don't share state" — vacuous structural property. BC-2.01.003 PC1 is "upstream sequence space INDEPENDENT of downstream sequence space."
- Impact: Test does not pin the BC's actual claim about upstream/downstream isolation. If implementer ever shared state by direction (e.g., `var globalUpstreamSeq uint32`), the test would still pass because A and B are separate struct instances with different chanIDs.
- Route: test-writer
- Fix: Strengthen `TestHalfChannelIndependentSequences` to use the SAME chanID but DIFFERENT directions (or two upstream instances), so the test actually validates no global-by-direction state.

## Axes Checked Clean

- `init()` functions: none present in halfchannel/.
- `interface{}` / `any`: none in production code.
- ST1005 error punctuation: `ErrNilPayload` text clean.
- Wraparound coverage: `wraparound_internal_test.go` covers MaxUint32 → 0 via seeded internal field. Adequate.
- Pure-core purity: `halfchannel.go` imports only `errors` and `time` (only `time.Duration`/`time.Millisecond` constants used). ARCH-09 satisfied.
- GC discipline: `pending[0] = nil` set before reslice.
- Race-detector compatibility: no goroutines, no channels, no sync primitives. Race-clean by construction.
- Pointer receivers: consistent across all methods.
- Topological import order: halfchannel imports no other internal package (zero internal imports), trivially satisfies ARCH-08 position-7 (though F-002 argues it SHOULD import frame).
- File / package naming: compliant.
