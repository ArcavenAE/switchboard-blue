---
document_type: holdout-scenario
level: ops
version: "1.0"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
wave: 2
cycle: v1.0.0-greenfield
id: HS-002
category: security-probes
must_pass: "true"
priority: must-pass
behavioral_contracts: [BC-2.05.005, BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007]
lifecycle_status: active
introduced: v1.0.0-greenfield
last_evaluated: null
staleness_check: null
stale_reason: null
retired: null
---

# Holdout Scenario HS-002: HMAC Authentication and SVTN Isolation Under Adversarial Frames

> **WARNING:** This file must NEVER be shown to the implementer or test-writer. Information asymmetry is the quality mechanism.

## Scenario

**HMAC Authentication:**
1. Two nodes, A and B, are admitted to SVTN-1. HMAC keys are derived per-(node, SVTN) via HKDF.
2. Node A produces a frame with a valid HMAC tag. Node B verifies it: must succeed.
3. A bit-flip is applied to the frame payload (position 12). Node B re-verifies: must return false.
4. The original frame is re-presented with the wrong key (SVTN-2's derived key): must return false.

**Admission Security:**
5. A node with an unregistered key attempts to send a frame through the router. The router must drop it (E-ADM-005) without forwarding anything.
6. The same challenge nonce is replayed by a different node. AdmitNode must return E-ADM-003 (replay prevention).

**SVTN Isolation:**
7. SVTN-1 has nodes A and B. SVTN-2 has nodes C and D. Node A sends a frame to B (svtn_id=SVTN-1). Nodes C and D must receive zero frames.
8. Private key bytes are extracted from a running admission session's wire structs. None of the wire structs (challenge, response, admission confirmation) contain private key bytes.

## Behavioral Contract Linkage

| BC ID | Clause Tested | Scenario Aspect |
|-------|--------------|-----------------|
| BC-2.05.005 | postcondition 2 (VerifyHMAC valid), postcondition 3 (bit flip) | Steps 2–3 |
| BC-2.05.005 | postcondition 4 (HKDF per-node-SVTN) | Step 4 |
| BC-2.05.001 | postcondition 1 (valid challenge), invariant 1 (replay) | Steps 5–6 |
| BC-2.05.002 | postcondition 1 (fail-closed forwarding) | Step 5 |
| BC-2.05.006 | postcondition 1 (SVTN isolation) | Step 7 |
| BC-2.05.007 | invariant 1 (private key absent) | Step 8 |

## Verification Approach

```go
func TestHoldoutWave2_HMACBitFlip(t *testing.T) {
    key := DeriveKey(nodeAPubKey, svtn1ID)
    frame := buildTestFrame()
    tag := ComputeHMAC(key, frame)
    assert(VerifyHMAC(key, frame, tag))       // Step 2: valid
    frame[12] ^= 0x01                          // Step 3: bit flip
    assert(!VerifyHMAC(key, frame, tag))
    frame[12] ^= 0x01                          // restore
    wrongKey := DeriveKey(nodeAPubKey, svtn2ID)
    assert(!VerifyHMAC(wrongKey, frame, tag)) // Step 4: wrong key
}

func TestHoldoutWave2_SVTNIsolation(t *testing.T) {
    router := buildTestRouter()
    router.Admit(nodeA, svtn1ID)
    router.Admit(nodeB, svtn1ID)
    router.Admit(nodeC, svtn2ID)
    router.Admit(nodeD, svtn2ID)
    frame := buildFrameFor(nodeA, nodeB, svtn1ID)
    delivered := router.Route(frame)
    // C and D must not appear in delivered
    assertNotContains(delivered, nodeC, nodeD)
}
```

## Evaluation Rubric

- **Functional correctness** (0.5): All 8 steps produce exact expected outcomes (VerifyHMAC true/false, drop on unadmitted, SVTN isolation).
- **Security** (0.3): No frame leaks across SVTN boundary; private key absent from wire structs.
- **Error quality** (0.1): E-ADM-003 for replay; E-ADM-005 for unadmitted.
- **Performance** (0.1): HMAC verification < 1ms per frame.

## Edge Conditions

- SVTN-ID all zeros: valid; tested in Wave 1 HS-001
- Challenge nonce = 0: AdmitNode still rejects replay (nonce 0 used once → rejected on second use)

## Failure Guidance

"HOLDOUT LOW: HS-002 (satisfaction: 0.XX) — HMAC verification failed under bit-flip or wrong-key case, or SVTN isolation allowed cross-SVTN frame delivery; check HKDF key derivation and SVTN routing partition"
