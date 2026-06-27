---
document_type: holdout-scenario
level: ops
version: "1.0"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
wave: 5
cycle: v1.0.0-greenfield
id: HS-005
category: integration-boundaries
must_pass: "true"
priority: must-pass
behavioral_contracts: [BC-2.06.001, BC-2.06.002, BC-2.07.001, BC-2.07.002, BC-2.07.003, BC-2.05.004]
lifecycle_status: active
introduced: v1.0.0-greenfield
last_evaluated: null
staleness_check: null
stale_reason: null
retired: null
---

# Holdout Scenario HS-005: Quality Indicator, Key Lifecycle, and sbctl Error Handling

> **WARNING:** This file must NEVER be shown to the implementer or test-writer. Information asymmetry is the quality mechanism.

## Scenario

**Quality Indicator:**
1. Path RTT is 80ms, loss 2%. QualityIndicator.Update is called. Indicator must be Green.
2. 3 consecutive probes arrive at RTT=600ms, loss=25%. After the 3rd, indicator becomes Red (degradation doesn't skip Yellow in downgrade direction).
3. 3 consecutive probes arrive at RTT=80ms, loss=2%. After the 3rd, indicator upgrades to Green (hysteresis = 3). A single good probe after Red does NOT upgrade.
4. 3 consecutive missing-frame events via OnMissingFrame. Indicator downgrades from Green.

**Key Lifecycle (sbctl admin):**
5. `sbctl admin key register --key <pubkeyA> --svtn <SVTN-1>` registers key A. Subsequent admission attempt with key A succeeds.
6. `sbctl admin key revoke --key <pubkeyA> --svtn <SVTN-1>` revokes key A. Subsequent admission attempt with key A returns E-ADM-005.
7. Attempting to revoke a control key without `--confirm` flag returns an error (ADR-004).

**sbctl Connection Error:**
8. `sbctl router status --target 127.0.0.1:19999` is run when no daemon is listening on that port. Exit code is 1. Stderr contains "E-NET-001" and the attempted address.
9. No stdout output appears before the error (no partial results).

## Behavioral Contract Linkage

| BC ID | Clause Tested | Scenario Aspect |
|-------|--------------|-----------------|
| BC-2.06.001 | postcondition 1 (threshold), invariant 1 (hysteresis) | Steps 1–3 |
| BC-2.06.002 | postcondition 1 (missing frame signal) | Step 4 |
| BC-2.05.004 | postcondition 1 (register), postcondition 2 (revoke) | Steps 5–6 |
| BC-2.05.004 | invariant 1 (control revocation --confirm) | Step 7 |
| BC-2.07.003 | postcondition 1 (E-NET-001, exit 1) | Step 8 |
| BC-2.07.003 | postcondition 2 (no partial output) | Step 9 |

## Verification Approach

```go
func TestHoldoutWave5_QualityHysteresis(t *testing.T) {
    qi := NewQualityIndicator()
    qi.Update(80, 2)
    assert(qi.Level() == Green)
    for i := 0; i < 3; i++ { qi.Update(600, 25) }
    assert(qi.Level() == Red)
    qi.Update(80, 2) // single good probe
    assert(qi.Level() == Red) // NOT upgraded yet
    for i := 0; i < 3; i++ { qi.Update(80, 2) }
    assert(qi.Level() == Green)
}
```

```bash
# sbctl connection error (integration):
sbctl router status --target 127.0.0.1:19999
echo "exit: $?"   # must be 1
# stderr must contain E-NET-001 and 127.0.0.1:19999
# stdout must be empty
```

## Evaluation Rubric

- **Functional correctness** (0.5): Quality hysteresis enforced (3 measurements for upgrade/downgrade); key lifecycle register/revoke/expire work; sbctl exits 1 with E-NET-001.
- **Edge case handling** (0.2): Single good probe after Red doesn't upgrade; revoke of non-existent key returns E-ADM-013.
- **Error quality** (0.2): E-NET-001 on connection refused; E-ADM-005 on revoked key; error message contains attempted address.
- **Performance** (0.1): sbctl connection failure detected within 2s.

## Edge Conditions

- `sbctl admin key revoke --key <unknown>`: E-ADM-013 (not E-ADM-002)
- Quality indicator with all-zero RTT and loss: Green (degenerate case)

## Failure Guidance

"HOLDOUT LOW: HS-005 (satisfaction: 0.XX) — quality indicator hysteresis incorrect, or sbctl did not emit E-NET-001 on connection failure, or key lifecycle register/revoke failed; check hysteresis counter and E-NET-001 error path"
