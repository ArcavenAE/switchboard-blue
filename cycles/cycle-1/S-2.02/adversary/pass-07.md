---
artifact_id: adv-S-2.02-pass-07
review_target: S-2.02-admission-svtn-isolation
producer: adversary
pass: 7
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

# Adversarial Review — Pass 7 — S-2.02

## Critical Findings
None.

## Important Findings
None.

## Observations
None.

## Novelty Assessment

S-2.02 implementation at tip `356fd6d` cleanly implements BC-2.05.001/002/006/007 within the worktree perimeter.

Verified items:
- AdmitNode's snapshot-then-relock pattern correctly handles concurrent AdmitNode/RevokeKey (race-tested via `TestAdmitNodeRevokeKey_NoRace`).
- Nonce-consume-before-verify ordering prevents same-challenge replay; failed-handshake nonces purged via M-2 lazy purge after 60s.
- IsAdmitted AND-gates on `admitted && !revoked` enforcing two-state model.
- SVTNRoute partitions strictly by SVTN ID with no override path; `forwardingTable[svtnID][nodeAddr]` lookup is correct.
- RouteFrame admitted-set check precedes any forwarding lookup; pinned by `TestRouteFrame_AdmittedSetCheckPrecedesForwarding`.
- Lookup deep-clones PublicKey, defending the locked-accessor contract (go.md rule 12).
- ARCH-08 dependency constraints satisfied (admission ↔ {frame, hmac}; routing ↔ {frame, hmac, admission}).
- ARCH-09 boundary classification matches package headers.

Test coverage:
- All 7 ACs (AC-001–007) covered with appropriate assertions.
- EC-001–004 covered including LWW reset (EC-004 via `TestRegisterKey_AfterRevoke_ClearsRevokedFlag`).
- VP-007 (1000-sample byte-substring property), VP-008 (fuzz target with 80-byte seed corpus), VP-039 (SVTN partition), VP-057 (admission-wire-struct subset, frame-type deferral cited).
- Race regression: `TestAdmitNodeRevokeKey_NoRace`.
- Wire-HMAC anti-tautology regression: `TestVerifyFrameHMAC_RejectsWrongTag`.

`verifyFrameHMAC` is correctly `//nolint:unused` until wire-layer wiring in next wave (already test-exercised via `routing_internal_test.go`).

Error sentinels (ErrNotAdmitted, ErrKeyRevoked, ErrNonceReplay, ErrSignatureVerificationFailed, ErrKeyNotRegistered, ErrNoForwardingEntry) cleanly support errors.Is dispatch.

Fuzz seed corpora verified at exact byte counts:
- FuzzAdmitNode: 80 bytes (32 unadmitted seed + 32 admitted seed + 16 SVTN ID)
- FuzzRouteFrame: 80 bytes (32 node seed + 16 SVTN + 32 router seed)

No defects within perimeter. Implementation, story, and BCs remain coherent across two consecutive fresh-context passes.
