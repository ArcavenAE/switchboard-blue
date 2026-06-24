---
artifact_id: wave-1-adversary-pass-01
producer: adversary
wave: 1
pass: 1
fresh_context: true
develop_tip: 9e9a98a
findings_count: 4
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 2, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Wave-1 Integration Adversary Pass-01

## Critical
None.

## High
None.

## Medium

### F-001 — Payload-size contract gap between halfchannel and frame.OuterHeader.PayloadLen
- Location: `internal/halfchannel/halfchannel.go:128-140` (Enqueue) and `internal/frame/frame.go:54-56` (OuterHeader.PayloadLen)
- Cross-module: halfchannel produces `ChannelFrame.Payload []byte` of arbitrary length; frame's `OuterHeader.PayloadLen` is `uint16` (max 65535) and per ARCH-02 must equal `channel_header_size + len(payload)` (12 or 20 bytes channel header). Neither module today documents or enforces the implied MTU contract.
- Evidence: Enqueue has no upper-bound check; PayloadLen u16 → max ~65523 payload. Halfchannel godoc silent about any maximum.
- Impact: The first outer-assembler that consumes `ChannelFrame.Payload` will silently truncate when computing `uint16(len(channel_header) + len(payload))` for any payload > ~65523 bytes. Integration defect that survives unit tests on both sides because each side meets its own spec.
- Route: product-owner (write MTU invariant into BC; assign enforcement to outer-assembler story)
- Fix: Add precondition to BC-2.01.002 establishing max ChannelFrame.Payload length consistent with OuterHeader.PayloadLen uint16. Either halfchannel documents caller-side responsibility, or accepts maxPayloadSize at New() and rejects over-MTU with ErrPayloadTooLarge.

### F-002 — No named `FrameType` type across the module boundary; cross-module assignments allow any byte value
- Location: `internal/frame/frame.go:27-33` (constants) + `internal/frame/frame.go:53` (`FrameType byte`) + `internal/halfchannel/halfchannel.go:18-21` (aliases) + `internal/halfchannel/halfchannel.go:66` (`FrameType byte`)
- Cross-module: outer-assembler will read `ChannelFrame.FrameType byte` from halfchannel and assign to `OuterHeader.FrameType byte` in frame. Both fields are unconstrained byte — any caller can assign any value without compile-time rejection.
- Evidence: `frame.go:53` is `FrameType byte` (not `FrameType FrameType`); constants are untyped; `ParseOuterHeader` accepts any byte at offset 1 silently (no enum validation).
- Impact: An outer-assembler bug writing an out-of-range FrameType (e.g., 0x00 from a zero-initialized struct, or 0xFF) would produce a wire frame routers may forward but downstream parsers cannot classify. Type system cannot prevent this.
- Route: architect (decide whether to introduce named type + Valid() method)
- Fix: Introduce `type FrameType byte` in internal/frame with five constants typed as FrameType. Have OuterHeader.FrameType and ChannelFrame.FrameType use the named type. Optionally add `(FrameType).Valid() bool` and reject invalid values in ParseOuterHeader with new sentinel ErrInvalidFrameType.

## Low

### F-003 — No composed wire-format test crossing the module boundary
- Location: test trees at `internal/frame/*_test.go` and `internal/halfchannel/*_test.go`
- Cross-module: No test exercises composed wire format: emit ChannelFrame from halfchannel + construct OuterHeader from frame + serialize-parse + assert field equality.
- Impact: Expected state for wave-1; outer-assembler is downstream. Becomes a defect only if the outer-assembler story doesn't deliver this test.
- Route: orchestrator (track as coverage marker)
- Fix: When outer-assembler story is drafted, require an AC asserting composed-wire-format round-trip.

### F-004 [per-story-scope] — ARCH-02 channel-header layout not exposed by halfchannel as a serializer
- Location: halfchannel.ChannelFrame surfaces semantic fields; channel-header byte encoding not implemented in either module.
- Impact: Per-story-scope — downstream story's deliverable.
- Route: architect (decide which package owns the channel-header serializer)
- Fix: Defer to outer-assembler story.

## Observations

- ARCH-08 topological order: VERIFIED. frame imports stdlib only; halfchannel imports stdlib + frame. No cycles or upward edges.
- ARCH-09 purity: VERIFIED. frame is pure (no time.Now, no I/O). halfchannel uses time as Duration only; no time.Now/Sleep in production; benchmark uses time.Now/Sleep as documented effectful glue.
- Cross-module constant alias mechanism is sound; documented at halfchannel.go:14-17.
- BC-2.01.002 PC2 split (pure-core sets ChannelFrame.FrameType; outer-assembler sets OuterHeader.frame_type) correctly implemented.
- Sequence-wraparound semantics agree across modules.
- BC-2.01.001 PC4 jitter budget correctly deferred to Phase-6.
- S-1.02 Spec Patches (passes 1-6) propagated cleanly to wave-1 code.

## Novelty Assessment

Pass-01 of wave-gate; novelty inherently high. Two mediums are genuine cross-module findings (MTU contract, named FrameType) — only surface when both packages read together. Per-story-scope F-004 is downstream-deliverable.

## Verdict

CONVERGED. Zero critical, zero high integration findings. Two mediums are boundary-contract gaps for downstream stories, not wave-1 blockers. Per-story-scope finding (F-004) excluded from gate.
