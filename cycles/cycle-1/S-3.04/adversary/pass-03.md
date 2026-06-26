---
artifact_id: adv-S-3.04-pass-03
review_target: S-3.04-hmac-routeframe-wireup
producer: adversary
pass: 3
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

# Adversarial Review — Pass 3 — S-3.04

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

## Axis-by-axis verification

- **A. AC↔BC↔test trace correctness:** AC-001 → BC-2.05.008 PC-1 → TestRouteFrame_ValidHMAC_ProceedsToAdmission. AC-002 → PC-2. AC-003 → PC-3/VP-058. AC-004 → PC-4/VP-058 prop 4. AC-005 → EC-005/inv 2. All five trace correctly.
- **B. Error code E-ADM-016:** sentinel string matches error-taxonomy.md line 52 and BC-2.05.008 §Traceability.
- **C. Sentinel godoc:** ErrHMACVerificationFailed at routing.go:28-39 cites E-ADM-016, BC-2.05.008 PCs 2/4, ADR-009. Distinct-from-ErrNotAdmitted callout present.
- **D. HMAC-before-admitted ordering (ADR-009 v1.6):** routing.go:138-151 — lookup → HMAC verify → admitted-set → SVTNRoute. Matches ADR-009 step order.
- **E. Lock discipline:** RLock at line 124, authKey [32]byte copied to local at line 133, RUnlock at line 136. HMAC verify (line 144) lock-free against local copy. Defensive-copy pattern matches ADR-009 v1.6 step 3.
- **F. Forwarding-table absence:** entry nil → ErrHMACVerificationFailed at line 138-140. Fail-closed per PC-4.
- **G. //nolint:unused:** removed; grep returns 0 matches.
- **H. Pre-existing tests:** TestRouteFrame_DropsUnadmitted + TestRouteFrame_AdmittedSetCheckPrecedesForwarding updated minimally to register forwarding entries + compute valid tags; original invariants preserved.
- **I. GREEN-BY-DESIGN tests:** all five AC tests assert distinct paths; sentinel-collision guards actively assert inverse conditions; no tautologies.
- **J. VP-058 harness:** routing_internal_test.go faithfully ports VP-058 §Proof Harness Skeleton with correct API (RegisterKey + GenerateChallenge + AdmitNode).
- **K. FuzzRouteFrame_NonAdmittedNeverForwarded:** post-fix, forwarding entry + valid tag registered → admission reached. Assertion is errors.Is(err, admission.ErrNotAdmitted) (NOT ErrHMACVerificationFailed). VP-008 genuinely exercised.
- **L. Spec drift:** Story v1.0, BC-2.05.008 v1.1, ARCH-04 v1.6, VP-058 v1.1 mutually consistent. verifyFrameHMAC signature matches ADR-009 v1.6 §Implementation note verbatim.
- **M. Comment accuracy:** routing.go:113-123 honestly describes defensive-copy pattern citing "ADR-009 v1.6 step 3", "defensive copy", "lock-free against that local copy", "sequential ordering preserved by statement order".
- **N. No race / data race:** lock-free verify operates on stack-local authKey ([32]byte) and stack-local entry pointer (only nil-checked after RUnlock). LWW replacement (ADR-003) of *ForwardingEntry preserves the copied [32]byte. No race surface.

## Novelty Assessment

Novelty: LOW — full re-derivation across 14 axes found zero defects. Implementation, tests, story, BC, VP, and ARCH-04 ADR-009 v1.6 are mutually consistent. Story has converged.

## Streak status

Pass 1: NOT_CONVERGED (2 findings: 1M/1L)
Pass 2: NOT_CONVERGED (2 findings: 1M/1L)
Pass 3: CONVERGED (0 findings)

**Streak: 1/3 toward BC-5.39.001.**
