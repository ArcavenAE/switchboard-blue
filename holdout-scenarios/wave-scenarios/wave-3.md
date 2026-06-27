---
document_type: holdout-scenario
level: ops
version: "1.1"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
wave: 3
cycle: v1.0.0-greenfield
id: HS-003
category: integration-boundaries
must_pass: "true"
priority: must-pass
behavioral_contracts: [BC-2.04.001, BC-2.04.002, BC-2.04.003, BC-2.04.004, BC-2.04.005, BC-2.04.006, BC-2.05.003]
lifecycle_status: active
introduced: v1.0.0-greenfield
last_evaluated: null
staleness_check: null
stale_reason: null
retired: null
---

# Holdout Scenario HS-003: End-to-End Session Access — Attach, Detach, Multi-Console, Read-Only

> **WARNING:** This file must NEVER be shown to the implementer or test-writer. Information asymmetry is the quality mechanism.

## Scenario

1. An access node connects to a real local tmux session via control mode.
2. Console A (read-write key) attaches to the session by name. It receives the live output stream.
3. Console B (read-write key) also attaches simultaneously. Both A and B receive all downstream frames — fan-out verified.
4. Console C (read-only key) attaches. It receives the downstream stream. Console C sends an upstream keystroke. The access node rejects it (E-ADM-007). Consoles A and B are unaffected — they still receive frames.
5. Console A detaches. The session continues. Console B and C still receive frames. Console A receives nothing further.
6. A keystroke is typed in the tmux session. Console B receives it within 200ms (latency gate).
7. tmux control mode is killed (simulated). The access node falls back to PTY mode automatically. Session stream continues.

## Behavioral Contract Linkage

| BC ID | Clause Tested | Scenario Aspect |
|-------|--------------|-----------------|
| BC-2.04.001 | postcondition 1 (tmux control mode attach) | Step 1 |
| BC-2.04.002 | postcondition 1 (PTY fallback on failure) | Step 7 |
| BC-2.04.003 | postcondition 1 (bidirectional stream) | Step 2 |
| BC-2.04.004 | postcondition 1 (detach; session continues) | Step 5 |
| BC-2.04.005 | postcondition 1 (read-only upstream rejected) | Step 4 |
| BC-2.04.006 | postcondition 1 (multi-console fan-out) | Steps 2–3 |
| BC-2.05.003 | postcondition 1 (Tier-2 access node enforces, not router) | Step 4 |

## Verification Approach

```bash
# Requires: tmux installed, test SVTN with 3 admitted keys (A=rw, B=rw, C=ro)
# Run: go test ./internal/tmux/... ./internal/session/... -run TestHoldoutWave3 -v -timeout 30s

# Step 1: access node connects to tmux
# Step 2-3: A and B attach; inject 10 downstream frames; both must receive all 10
# Step 4: C attaches; C sends upstream; assert errors.Is(err, session.ErrUpstreamReadOnly) / error message carries E-ADM-007; assert A and B still receive
# Step 5: A detaches; inject 5 more frames; B and C receive 5; A receives 0
# Step 6: keystroke injected; latency measured; assert < 200ms
# Step 7: kill tmux socket; assert PTY fallback without error; assert stream continues
```

## Evaluation Rubric

- **Functional correctness** (0.5): All 7 steps succeed in order; exact frame counts match expected fan-out.
- **Edge case handling** (0.2): C's upstream rejected without affecting A or B; A's detach doesn't close session.
- **Error quality** (0.1): E-ADM-007 returned for C's upstream keystroke (errors.Is(err, session.ErrUpstreamReadOnly) must hold; error message must carry E-ADM-007).
- **Performance** (0.1): Keystroke-to-echo < 200ms (soft gate; 100ms is P99 NFR target).
- **Data integrity** (0.1): PTY fallback stream is byte-identical to control mode stream for simple output.

## Edge Conditions

- All consoles detach before step 7: session persists (access node remains connected)
- PTY fallback while multiple consoles attached: all consoles get fallback stream

## Failure Guidance

"HOLDOUT LOW: HS-003 (satisfaction: 0.XX) — fan-out delivery, read-only rejection, or PTY fallback failed; check ConsoleSet fan-out logic and Tier-2 read-only enforcement"
