---
artifact_id: adv-S-2.02-pass-06
review_target: S-2.02-admission-svtn-isolation
producer: adversary
pass: 6
fresh_context: true
branch: feature/S-2.02-admission-svtn-isolation
base: develop @ 3c4104e
tip: 356fd6d
findings_count: 0
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 0, nitpick: 0}
verdict: CONVERGED
timestamp: 2026-06-25
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 6 — S-2.02

## Critical Findings
None.

## Important Findings
None.

## Observations
None.

## Novelty Assessment

**Novelty: ZERO — no new defects.** Reviewed admission.go (372 lines), routing.go (172 lines), all three test files, and the seven spec inputs (story rev 1.3, BC-2.05.001/002/006/007, ARCH-02/04/08/09, VP-007/008/009/010/039/057, error taxonomy).

Verified:
- All 7 ACs trace to existing tests with correct assertions; EC-001/002/003 + EC-004 covered.
- BC-2.05.001 PC4 two-state model (registered ≠ admitted) AND-gated in `IsAdmitted` line 237; pinned by `TestIsAdmitted_FailsBeforeHandshake`.
- BC-2.05.002 PC3 (admitted-set check before forwarding) enforced at `RouteFrame` line 99 before `SVTNRoute`; pinned by `TestRouteFrame_AdmittedSetCheckPrecedesForwarding` with both positive (ErrNotAdmitted) and negative (NOT ErrNoForwardingEntry) assertions.
- BC-2.05.006 PC1/2/4 enforced by SVTN-partitioned `forwardingTable[svtnID][nodeAddr]` lookup at `SVTNRoute` line 124-128.
- BC-2.05.007 inv 1 / DI-002 verified at type level (AC-006/007) and runtime via `TestProperty_VP007_PrivateKeyByteSubstringAbsent` (1000 samples).
- H-1 race fix (snapshot under RLock, re-fetch under Lock) at `AdmitNode` lines 287-329; race regression test `TestAdmitNodeRevokeKey_NoRace` present.
- H-1 wire-HMAC tautology fix verified by `routing_internal_test.go` `TestVerifyFrameHMAC_RejectsWrongTag` (rejects wrong tag, accepts correct tag).
- L-2 sentinel fix: `RevokeKey` returns `ErrKeyNotRegistered` (E-ADM-013), not `ErrNotAdmitted`; pinned by table-driven `TestRevokeKey_ReturnsErrKeyNotRegistered`.
- ADR-003 LWW re-handshake semantic (admitted=false on re-registration): `RegisterKey` line 152-157 zero-initialises `admitted`; pinned by `TestRegisterKey_AfterRevoke_ClearsRevokedFlag` 3-step flow.
- M-2 lazy nonce purge implemented at `recordNonceUnlocked` lines 357-363; M-3 deep-clone of PublicKey at `Lookup` line 212.
- VP-007 / VP-008 / VP-009 / VP-010 / VP-057 (admission-wire-struct subset) covered by tests; VP-039 covered by VP-010 unit cluster (`NoCrossContamination`, `SVTNPartitionBoundary`); VP-057 frame-type-full-coverage deferral cited in story task 8 and rev 1.3 patches.
- Nonce-consume-before-verify order documented as deliberate trade-off (admission.go lines 311-316, 337-339). DoS via junk-sig nonce burn recoverable via fresh challenge; consistent with ARCH-04 §"Nonce uniqueness".
- ARCH-08 import constraints satisfied: admission imports {frame, hmac}; routing imports {frame, hmac, admission}. ARCH-09 boundary classification matches package headers.
- Error sentinels (ErrSignatureVerificationFailed/ErrKeyRevoked/ErrNonceReplay/ErrNotAdmitted/ErrKeyNotRegistered/ErrNoForwardingEntry) map cleanly to E-ADM-001/005/008/003/013 and routing-local sentinel respectively; all errors.Is-compatible.

The implementation, story, and BCs are coherent. Prior-pass fixes (H-1 race, H-1 wire-HMAC tautology, L-2 sentinel split, L-2/M-4 LWW re-handshake, M-1 fuzz corpus length, M-2 lazy purge, M-3 deep-clone, L-3 VP-057 deferral, H-3 trace anchors) have all propagated to code, tests, story body, and ARCH-04. Spec has converged.
