---
artifact_id: adv-S-3.04-pass-04
review_target: S-3.04-hmac-routeframe-wireup
producer: adversary
pass: 4
fresh_context: true
branch: feature/S-3.04-hmac-routeframe-wireup
base: develop @ d8d7ae6
tip: e214f8d
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 4 — S-3.04

## Critical Findings
None.

## High Findings
None.

## Medium Findings
None.

## Low Findings
None.

## Nitpicks
None.

## Review evidence checked

- `routing.go:112-155` — ordering matches ADR-009 v1.6 steps 1-7 exactly: forwarding-table RLock → entry lookup → [32]byte authKey copy → RUnlock → lock-free verifyFrameHMAC → IsAdmitted → SVTNRoute.
- `routing.go:39` — sentinel ErrHMACVerificationFailed maps to E-ADM-016 per BC-2.05.008 traceability and error-taxonomy.md:52.
- `routing.go:203` — //nolint:unused removed.
- `routing.go:204-210` — wire-tag snapshot before zeroing (S-2.02 pass-4 H-1 anti-tautology fix preserved).
- All 5 ACs covered with correct sentinel assertions.
- VP-058 properties 1-4 covered in routing_internal_test.go.
- Ordering invariants double-asserted (positive sentinel + negative-sentinel ordering check) in AC-003 and AC-005.
- FuzzRouteFrame_NonAdmittedNeverForwarded corpus = 80 bytes, admission check reached, assertion is errors.Is(err, admission.ErrNotAdmitted).
- Lock discipline: RLock-then-copy-then-release; no internal-pointer leaks; HMAC compute lock-free.
- No race: all state mutation via RegisterForwardingEntry under WLock; reads under RLock with defensive value copy of [32]byte.
- Trace anchors (BC-2.05.008 PC-1/2/3/4, EC-001..005, ADR-009 steps, VP-058 properties 1-4) all map to test assertions correctly.
- EncodeOuterHeader/verifyFrameHMAC symmetric: both zero HMACTag bytes 36-44 before computing MAC over (header || payload). Sender and verifier agree.

## Novelty Assessment

Novelty: LOW — second consecutive clean pass; full re-derivation finds zero defects.

## Streak status

Pass 1: NOT_CONVERGED (2 findings: 1M/1L)
Pass 2: NOT_CONVERGED (2 findings: 1M/1L)
Pass 3: CONVERGED (0 findings)
Pass 4: CONVERGED (0 findings)

**Streak: 2/3 toward BC-5.39.001.**
