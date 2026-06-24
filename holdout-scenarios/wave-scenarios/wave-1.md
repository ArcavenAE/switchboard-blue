---
document_type: holdout-scenario
level: ops
version: "1.1"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
wave: 1
cycle: v1.0.0-greenfield
id: HS-001
category: integration-boundaries
must_pass: "true"
priority: must-pass
behavioral_contracts: [BC-2.01.004, BC-2.01.001, BC-2.01.003]
lifecycle_status: active
introduced: v1.0.0-greenfield
last_evaluated: 2026-06-24
staleness_check: null
stale_reason: null
retired: null
modified:
  - "2026-06-24: Patched Step 5/6 wording to match BC-2.01.001 PC5 post-increment canonical vector. Resolves wave-1 holdout FAIL on sequence-semantics ambiguity."
---

# Holdout Scenario HS-001: Wire Format Codec Round-Trip Under Adversarial Inputs

> **WARNING:** This file must NEVER be shown to the implementer or test-writer. Information asymmetry is the quality mechanism.

## Scenario

1. A test harness constructs 1,000 OuterHeaders with randomly generated field values (version=1, valid frame_type enum values, random svtn_id, src_addr, dst_addr, and hmac_tag bytes; payload_len between 0 and 65535).
2. Each header is encoded via `EncodeOuterHeader` and immediately decoded via `ParseOuterHeader`.
3. The decoded header must exactly match the original on all 6 fields.
4. Additionally: three malformed inputs are presented — (a) 43-byte buffer, (b) 45-byte buffer, (c) 44-byte buffer with version=255. For (a) and (b): ParseOuterHeader must return error (E-FRM-001 for short, silence for oversized-but-parseable). For (c): ParseOuterHeader must return E-FRM-002.
5. A HalfChannel (chanID 42, Upstream, 10ms interval) is driven for 100 ticks with no payload. The sequence counter is initialized at 0. The first emitted frame carries `ChanSeq = 1` (post-increment per BC-2.01.001 PC5 canonical vector "sequence 1..10"); subsequent frames must be strictly contiguous with delta 1. All 100 frames must therefore have `ChanSeq ∈ {1, 2, ..., 100}` in order, with payload zero-length and `FrameType == FrameTypeEmptyTick (0x02)`.
6. Two independent HalfChannels (upstream chanID 42, downstream chanID 43) are ticked 50 times each — all Upstream ticks first, then all Downstream ticks. After the 50 Upstream ticks: `up.Seq() == 50` and the 50 Upstream frames have `ChanSeq` 1..50 in order. After the 50 Downstream ticks: `down.Seq() == 50` and the 50 Downstream frames have `ChanSeq` 1..50 in order (Downstream's counter is initialized at 0 independently, so its first emit is also 1). At no point does ticking one channel advance the other's counter: `up.Seq()` remains 50 throughout the Downstream ticks, and `down.Seq()` was 0 throughout the Upstream ticks.

## Behavioral Contract Linkage

| BC ID | Clause Tested | Scenario Aspect |
|-------|--------------|-----------------|
| BC-2.01.004 | postcondition 1 (44-byte layout), postcondition 2 (round-trip) | Steps 1–3 |
| BC-2.01.004 | precondition 1 (too-short rejection), precondition 2 (version mismatch) | Step 4 |
| BC-2.01.001 | postcondition 1 (one frame per tick) | Step 5 |
| BC-2.01.003 | postcondition 2 (independent sequences) | Step 6 |

## Verification Approach

```go
// Run with: go test ./internal/frame/... ./internal/halfchannel/... -run TestHoldoutWave1 -v
func TestHoldoutWave1_RoundTrip(t *testing.T) {
    // 1000 random round-trips
    for i := 0; i < 1000; i++ {
        h := randomOuterHeader()
        encoded := EncodeOuterHeader(h)
        decoded, err := ParseOuterHeader(encoded)
        require(decoded == h, "round-trip mismatch at iteration %d", i)
    }
}

func TestHoldoutWave1_MalformedInputs(t *testing.T) {
    // 43-byte buffer: E-FRM-001
    _, err := ParseOuterHeader(make([]byte, 43))
    assertErrorCode(err, "E-FRM-001")
    // version=255: E-FRM-002
    buf := make([]byte, 44)
    buf[0] = 255
    _, err = ParseOuterHeader(buf)
    assertErrorCode(err, "E-FRM-002")
}

func TestHoldoutWave1_HalfChannelContinuity(t *testing.T) {
    ch := NewHalfChannel()
    for i := 1; i <= 100; i++ {
        f := ch.Tick(nil)
        // Post-increment: counter initialized at 0, first emit is 1 (BC-2.01.001 PC5)
        assert(f.Seq == uint32(i), "seq mismatch at tick %d: got %d, want %d", i, f.Seq, i)
    }
}
```

## Evaluation Rubric

- **Functional correctness** (0.5): All 1000 round-trips produce identical outputs; malformed inputs return correct E-codes; 100-tick sequence is `ChanSeq ∈ {1..100}` (post-increment, counter initialized at 0 per BC-2.01.001 PC5).
- **Edge case handling** (0.2): Version=255 rejected with E-FRM-002; 43-byte buffer rejected; half-channel sequence independent.
- **Error quality** (0.2): Error codes match E-FRM-001, E-FRM-002 exactly (not generic errors).
- **Performance** (0.1): 1000 round-trips complete in < 100ms.

## Edge Conditions

- Sequence wrapping (uint32 overflow not tested in Wave 1; deferred to Phase 3 property test)
- 44-byte buffer with all-zeros: valid (svtn_id=0 is valid per wire format)
- payload_len=65535: valid encoding; ParseOuterHeader does not validate payload_len against actual payload

## Failure Guidance

"HOLDOUT LOW: HS-001 (satisfaction: 0.XX) — frame codec round-trip or HalfChannel tick sequence failed; check EncodeOuterHeader field order and HalfChannel sequence state initialization"

## Canonical sequence semantic (reference)

Sequence numbering follows BC-2.01.001 PC5: the counter is uint32 starting at 0. `Tick()` performs post-increment (`counter += 1; emit(counter)`), so the canonical 10-tick vector is `sequence 1..10`. VP-017/VP-053 harness skeletons (post pass-01 F-007) and the S-1.02 implementation reflect this. HS-001 v1.0 originally read "contiguous sequence numbers starting at 0" — that wording was ambiguous between "counter initialized at 0" and "first emitted seq is 0"; this revision (v1.1) makes the canonical choice (counter-initialized-at-0, first-emit-is-1) explicit. See wave-1 adversarial trail commit `e4fc8c9` and holdout-evaluator FAIL report `<persist-sha>` for the disagreement that surfaced the wording defect.
