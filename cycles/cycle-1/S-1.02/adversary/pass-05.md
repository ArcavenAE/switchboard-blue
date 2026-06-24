---
artifact_id: adv-S-1.02-pass-05
review_target: S-1.02-halfchannel-clock
producer: adversary
pass: 5
fresh_context: true
findings_count: 4
findings_by_severity: {critical: 0, high: 1, medium: 0, low: 3, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 5 — S-1.02

## High

### F-001 — AC-004 mis-anchored to BC-2.01.003 PC2 (actual target is BC-2.01.001 PC5)
- Location: `.factory/stories/S-1.02-halfchannel-clock.md:66-68`
- Evidence:
  ```
  ### AC-004 (traces to BC-2.01.003 postcondition 2)
  Each successive call to `Tick()` on the same half-channel increments the sequence number by exactly 1.
  ```
  BC-2.01.003 PC2 (`.factory/specs/behavioral-contracts/ss-01/BC-2.01.003.md:52`): "A loss on the upstream half-channel does not retrigger the downstream half-channel's recovery mechanism." — a recovery-independence claim, not a sequence-increment claim.
  The "+1 per tick" invariant is BC-2.01.001 PC5 (`BC-2.01.001.md:54`): "The frame sequence number increments by exactly 1 on each tick."
- Impact: AC↔BC traceability mis-anchor. A future implementer reading AC-004 would consult BC-2.01.003 expecting recovery-independence; the postcondition matching the AC text lives in BC-2.01.001. Mis-anchoring blocks convergence per Semantic Anchoring Audit policy.
- Route: product-owner
- Fix: Change AC-004 trace from `(traces to BC-2.01.003 postcondition 2)` to `(traces to BC-2.01.001 postcondition 5)`. Add a Spec Patches row for pass-5.

## Low

### F-002 — Stale filename reference in test comment
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:397-399`
- Evidence:
  ```go
  // TestHalfChannelSequenceWraparound: EC-002 wraparound is now covered by the
  // internal-package test in wraparound_internal_test.go, which seeds hc.seq
  // directly. A public-API-only test cannot reach MaxUint32 in reasonable time.
  ```
  Actual file is `wraparound_test.go` (renamed in pass-4 F-005, commit `68f491f`).
- Impact: Stale doc reference. A developer following the breadcrumb would 404.
- Route: test-writer
- Fix: Update the comment to reference `wraparound_test.go`.

### F-003 — `ChannelFrame.Flags == 0` invariant has no test pin
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel.go:119-125` (Flags: 0 hardcoded); `halfchannel_test.go` has no assertion on `frame.Flags`.
- Evidence: BC-2.01.002 PC3 (`.factory/specs/behavioral-contracts/ss-01/BC-2.01.002.md:57`): "The channel header is fully populated (chan_id, chan_seq, flags=0)." Grep for `Flags|f\.Flags` in the test file returns zero assertions.
- Impact: A future change that accidentally sets a non-zero default Flags would not be caught.
- Route: test-writer
- Fix: Add `if frame.Flags != 0 { t.Errorf(...) }` to `TestHalfChannelTick_EmptyFrameIsValid` and `TestHalfChannelTick_DataFrameType`.

### F-004 — Misleading "wraparound-handling" comment on VP-017 test that doesn't exercise wraparound
- Location: `.worktrees/S-1.02/internal/halfchannel/halfchannel_test.go:449-451`
- Evidence:
  ```go
  // This test asserts f.ChanSeq - prev.ChanSeq == 1 using uint32
  // modular subtraction, so it correctly handles the MaxUint32 → 0 wraparound.
  // VP-017: invariant — every Tick increments ChanSeq by exactly 1.
  ```
  Test runs `iterations = 10_000` from seq=0 — never reaches `math.MaxUint32` (~4.29 × 10^9). Wraparound coverage lives in `wraparound_test.go`.
- Impact: Reader is misled to believe `TestProperty_VP017_SequenceIncrementsByOne` exercises wraparound. A future reviewer might delete `wraparound_test.go` thinking it's redundant.
- Route: test-writer
- Fix: Replace with: "VP-017 asserts delta-by-1 using uint32 arithmetic; the modular subtraction is wraparound-safe by construction but this test does not exercise wraparound — EC-002 is covered by wraparound_test.go."

## Axes Checked Clean

- BC↔ARCH↔code FrameType discriminator consistent.
- Channel-header flag bit layout matches ARCH-02 §3.2 and BC-2.01.005.
- AC test-name traceability for AC-001/-002/-003/-005/-006 — all six tests exist with exact names.
- ErrEmptyPayload sentinel hygiene: exported, godoc'd, used via `errors.Is`.
- Go quality rules: no `init()`, no `interface{}`, no nil-before-len, no error-string punctuation, pointer receivers consistent, UTC timestamp, no panics in library code.
- ARCH-08 topological order: halfchannel (position 7) imports only frame (position 2). No back-edges.
- Property-test rigor: VP-017 (10k), VP-018 (5×100), VP-051 (10k interleaved). Adequate.
- Wraparound (EC-002) covered by wraparound_test.go.
- Race-detector compatibility: documented single-goroutine contract; tests use `t.Parallel()` on independent instances.
- Zero-copy contract pinned by TestHalfChannelTick_PayloadZeroCopy.
- GC discipline correct.
- Purity (ARCH-09): no goroutines, no time.Now/Sleep in production source.
- Direction godoc names real downstream consumers (S-3.01, S-4.03, ADR-008).
