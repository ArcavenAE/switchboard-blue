---
document_type: holdout-scenario
level: ops
version: "1.0"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
wave: 6
cycle: v1.0.0-greenfield
id: HS-006
category: integration-boundaries
must_pass: "true"
priority: must-pass
behavioral_contracts: [BC-2.02.007, BC-2.03.001, BC-2.03.002, BC-2.03.003, BC-2.08.001, BC-2.09.001, BC-2.09.002]
lifecycle_status: active
introduced: v1.0.0-greenfield
last_evaluated: null
staleness_check: null
stale_reason: null
retired: null
---

# Holdout Scenario HS-006: PE-Phase — FEC Recovery, Session Discovery, Remote Console Control, and Drain

> **WARNING:** This file must NEVER be shown to the implementer or test-writer. Information asymmetry is the quality mechanism.

## Scenario

**XOR FEC:**
1. A group of 4 frames is encoded; 1 parity frame is produced with frame_type=fec=0x05.
2. Frame 3 of the group is dropped (simulated). FEC.Recover(group_with_gap, parity_frame) reconstructs frame 3 exactly.
3. Two frames in the same group are dropped. FEC.Recover returns ErrTooManyLosses.

**Session Discovery:**
4. An access node with an active tmux session sends a presence advertisement. The advertisement payload includes: session_name, attachment_status=attached, and quality_indicator=Green.
5. A console calls Discovery.Enumerate() and receives the session in the list without providing the access node's IP address.
6. The access node session detaches (all consoles leave). A new presence advertisement fires immediately with attachment_status=detached.

**Console Remote Control:**
7. `sbctl console attach --target <console_addr> --session my-session` is run. The console daemon attaches to "my-session".
8. `sbctl console switch --target <console_addr> --session other-session` detaches from "my-session" and attaches to "other-session" atomically.

**PE Graduation + Drain:**
9. An E-mode router config gains `upstream_routers: [router2:9090]` entry. Config.Validate() accepts it. Router reloads and enters PE mode (no binary change).
10. `Drain.Signal()` is called. All connected nodes receive the drain message and migrate to alternate routers within 2 seconds. Router exits cleanly.

## Behavioral Contract Linkage

| BC ID | Clause Tested | Scenario Aspect |
|-------|--------------|-----------------|
| BC-2.02.007 | postcondition 1 (parity frame), postcondition 2 (single loss recovery), precondition 1 (two losses fail) | Steps 1–3 |
| BC-2.03.001 | postcondition 1 (advertise on state change + heartbeat) | Steps 4, 6 |
| BC-2.03.002 | postcondition 1 (enumerate without hostname) | Step 5 |
| BC-2.03.003 | postcondition 1 (required fields) | Step 4 |
| BC-2.08.001 | postcondition 1 (attach), postcondition 3 (switch) | Steps 7–8 |
| BC-2.09.001 | postcondition 1 (E→PE config-only) | Step 9 |
| BC-2.09.002 | postcondition 1 (drain within 2s), postcondition 2 (clean exit) | Step 10 |

## Verification Approach

```go
func TestHoldoutWave6_FECRecovery(t *testing.T) {
    fec := NewFEC(groupSize: 4)
    group := [4]Frame{f0, f1, f2, f3}
    parity := fec.Encode(group)
    assert(parity.FrameType == 0x05) // fec=0x05
    group[2] = Frame{} // drop frame 3
    recovered := fec.Recover(group, parity)
    assert(recovered == f2)
    // Two losses:
    group[1] = Frame{}
    _, err := fec.Recover(group, parity)
    assert(errors.Is(err, ErrTooManyLosses))
}

func TestHoldoutWave6_SessionDiscovery(t *testing.T) {
    // Start access node with active session
    // Enumerate without hostname
    sessions := discovery.Enumerate()
    assertContains(sessions, SessionPresence{
        Name: "my-session",
        AttachmentStatus: Attached,
        QualityIndicator: Green,
    })
}
```

```bash
# Drain test (e2e, 2s timeout):
router drain &
sleep 2.1
assert_all_nodes_migrated
assert_router_exited_cleanly
```

## Evaluation Rubric

- **Functional correctness** (0.5): FEC single-loss recovery works; session discovery returns sessions without hostname; console remote control attaches/switches; PE graduation via config; drain within 2s.
- **Edge case handling** (0.2): FEC two-losses returns ErrTooManyLosses; detach triggers immediate re-advertisement.
- **Error quality** (0.1): Drain timeout (>2s) results in forced exit with log.
- **Performance** (0.2): Drain completes in < 2s; FEC recovery < 1ms.

## Edge Conditions

- PE graduation with invalid upstream_routers format: Config.Validate() returns E-CFG-001
- Discovery with no active sessions: Enumerate() returns empty list (no error)

## Failure Guidance

"HOLDOUT LOW: HS-006 (satisfaction: 0.XX) — XOR FEC recovery, session discovery enumeration, or router drain failed; check FEC XOR logic, discovery multicast scope, and drain 2s timeout enforcement"
