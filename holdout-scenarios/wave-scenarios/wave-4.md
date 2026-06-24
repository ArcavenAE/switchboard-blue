---
document_type: holdout-scenario
level: ops
version: "1.0"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
wave: 4
cycle: v1.0.0-greenfield
id: HS-004
category: behavioral-subtleties
must_pass: "true"
priority: must-pass
behavioral_contracts: [BC-2.02.001, BC-2.02.002, BC-2.02.004, BC-2.02.005, BC-2.02.006, BC-2.02.008, BC-2.02.009, BC-2.09.003]
lifecycle_status: active
introduced: v1.0.0-greenfield
last_evaluated: null
staleness_check: null
stale_reason: null
retired: null
---

# Holdout Scenario HS-004: Reliability Layer — Duplicate-and-Race, ARQ, SACK, and Loop Prevention

> **WARNING:** This file must NEVER be shown to the implementer or test-writer. Information asymmetry is the quality mechanism.

## Scenario

**Duplicate-and-race:**
1. A frame is dispatched on two paths simultaneously. The path with lower simulated RTT delivers first. The receiver delivers it once and discards the duplicate from the slower path.
2. The DropCache is pre-filled to exactly its capacity. One additional frame arrives via the slow path (it is a duplicate). LRU eviction occurs and the cache stays at capacity.

**ARQ + SACK:**
3. 10 downstream frames are sent. Frame 5 is dropped (simulated path loss). The console sends a SACK bitmap indicating frame 5 missing. The access node retransmits frame 5. The console delivers all 10 frames in order.
4. Frame 7 is overdue (deadline exceeded). TLPKTDROP fires. Frame 7 is terminated and a degradation event is emitted. Frames 8–10 continue to be delivered (session not terminated).

**Split-horizon:**
5. A frame arrives on interface-2. The router forwards it on interface-1 and interface-3 only (not interface-2). Channel header bytes are arbitrary garbage — router must not panic or parse them.

**Config validation:**
6. A router config file with `tick_interval: 75ms` (above 50ms max) is presented to `Config.Validate()`. The error message includes the field name, the invalid value, and the valid range.

## Behavioral Contract Linkage

| BC ID | Clause Tested | Scenario Aspect |
|-------|--------------|-----------------|
| BC-2.02.001 | postcondition 1 (dup-and-race send on two paths) | Step 1 |
| BC-2.02.002 | postcondition 1 (first copy delivered), postcondition 2 (silent discard) | Step 1 |
| BC-2.02.009 | postcondition 1 (drop cache bounded) | Step 2 |
| BC-2.02.005 | postcondition 1 (no double delivery), postcondition 2 (in-order) | Step 3 |
| BC-2.02.006 | postcondition 1 (TLPKTDROP terminates), postcondition 2 (degradation) | Step 4 |
| BC-2.02.004 | postcondition 1 (replay no double delivery) | Step 3 |
| BC-2.02.008 | postcondition 1 (no forward toward arrival) | Step 5 |
| BC-2.09.003 | postcondition 1 (actionable error message) | Step 6 |

## Verification Approach

```go
func TestHoldoutWave4_DupAndRace(t *testing.T) {
    mp := NewMultipath(dropCacheCapacity: 100)
    frame := buildTestFrame()
    mp.Receive(frame)   // first copy: delivered
    result := mp.Receive(frame) // second copy: ErrDuplicate
    assert(result == ErrDuplicate)
}

func TestHoldoutWave4_ARQWithSACK(t *testing.T) {
    arq := NewARQ()
    for i := 0; i < 10; i++ { arq.Send(frames[i]) }
    // Drop frame 5, receive SACK bitmap with bit 5 set
    arq.OnAck(4, sackBitmapMissingFrame5)
    // Verify frame 5 is retransmitted
    arq.OnAck(10, allAcked)
    deliveredSeqs := arq.DeliveredSequences()
    assert(deliveredSeqs == []int{0,1,2,3,4,5,6,7,8,9})
}

func TestHoldoutWave4_SplitHorizon_GarbageChannelHeader(t *testing.T) {
    router := NewRouter()
    frame := buildFrameWithGarbageChannelHeader()
    result := router.Route(frame, arrivalInterface: 2)
    assertNotForwardedOn(result, 2)
    assertForwardedOn(result, 1, 3)
}

func TestHoldoutWave4_ConfigActionableError(t *testing.T) {
    cfg := Config{TickInterval: 75 * time.Millisecond}
    err := cfg.Validate()
    assertContains(err.Error(), "tick_interval")
    assertContains(err.Error(), "75ms")
    assertContains(err.Error(), "5ms")
    assertContains(err.Error(), "50ms")
}
```

## Evaluation Rubric

- **Functional correctness** (0.5): All 6 steps produce exact expected outcomes.
- **Edge case handling** (0.2): DropCache LRU eviction at capacity; TLPKTDROP doesn't terminate session; split-horizon with garbage channel header doesn't panic.
- **Error quality** (0.2): Config error includes field name, value, and valid range.
- **Performance** (0.1): ARQ retransmit+delivery for 10 frames < 10ms.

## Edge Conditions

- TLPKTDROP on frame 1 (first frame ever): subsequent frames still deliverable
- DropCache with capacity=1: every second duplicate cache miss is a new eviction

## Failure Guidance

"HOLDOUT LOW: HS-004 (satisfaction: 0.XX) — reliability layer has duplicate delivery, incorrect SACK processing, or split-horizon forwarding toward arrival interface; check ARQ SACK bitmap logic and DropCache key format (checksum, interface_id)"
