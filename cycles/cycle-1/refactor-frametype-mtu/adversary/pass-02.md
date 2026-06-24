---
artifact_id: adv-refactor-frametype-mtu-pass-02
review_target: refactor-frametype-mtu (F-001+F-002 combined PR)
producer: adversary
pass: 2
fresh_context: true
branch: feature/refactor-frametype-mtu
base: develop @ 9e9a98a
tip: 4f4c07c
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 2 — Refactor F-001+F-002

## Verdict: CONVERGED — Zero Findings

Second consecutive clean pass. Convergence streak 2/3.

## Axes Checked Clean

### F-001 closure (MTU validation)
- halfchannel.go:77 defines MaxPayloadSize = 65535 - 20 = 65515, matching BC-2.01.002 v1.4 PC5 conservative bound (SACK_present=1).
- Enqueue (line 143-154) orders validation correctly: empty check then oversize check; wraps ErrPayloadTooLarge with %w and includes len(payload)/max context.
- Tests at halfchannel_test.go:586-632 cover the constant pin, oversize rejection (errors.Is(ErrPayloadTooLarge)), and exact-boundary acceptance.

### F-002 closure (typed FrameType + enum validation)
- frame.go:29 defines `type FrameType byte` with five typed constants (lines 33-37).
- Valid() method (line 42) correctly bounds the byte range; uint8 ordering is safe.
- ErrInvalidFrameType defined (line 49); ParseOuterHeader validates frame_type AFTER length+version (lines 111-114), wrapped with %w and the offending byte value.
- OuterHeader.FrameType (frame.go:69) and ChannelFrame.FrameType (halfchannel.go:67) both use the named frame.FrameType type, so the type system enforces typed-cross-module propagation.
- Tests at frame_test.go:505-591 cover Valid() truth table, parser rejection of invalid bytes, and acceptance of all five canonical values.

### Other axes verified
- Spec alignment: BC-2.01.002 v1.4 PC5 documents both 65523 (SACK=0) and 65515 (SACK=1); implementation uses 65515 with rationale comment. ARCH-02 §3.1 frame_type enum (0x01-0x05) matches frame.go constants.
- Validation ordering ParseOuterHeader: len → version → frame_type. Correct.
- Validation ordering Enqueue: empty → oversize. Correct.
- Type-system rigor: No byte→FrameType conversions bypass Valid() outside ParseOuterHeader; encoder permissive by design (trust caller); parser strict (validate wire). Standard inverted-Postel pattern.
- ARCH-09 purity: internal/frame and internal/halfchannel remain pure-core; no time.Now, no I/O, no goroutines introduced.
- ARCH-08 DAG: halfchannel → frame edge preserved via constant aliases; no upward import.
- go.md compliance: errors wrapped with %w; no ST1005 trailing punctuation; no init/any/panics; Valid() value receiver appropriate; pointer-receiver consistency on *HalfChannel.
- Tests use errors.Is consistently; no string matching; stdlib testing + table-driven + t.Parallel().
- Round-trip soundness: TestParseEncodeRoundTrip and FuzzEncodeParseRoundTrip still pass through the new validation path.

## Convergence streak: 2/3
Need pass 3 also clean for BC-5.39.001 closure.
