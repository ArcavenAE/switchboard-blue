---
artifact_id: wave-2-fresh-context-audit
review_target: wave-2-combined (S-2.01 + S-2.02 + S-1.03)
producer: adversary
fresh_context: true
branch: develop
tip: f35e836
findings_count: 7
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 3, observation: 3}
verdict: PASS_WITH_OBSERVATIONS
timestamp: 2026-06-25
note: Returned inline by adversary (read-only tool profile); persisted by orchestrator via state-manager.
---

# Wave 2 Fresh-Context Cross-Story Audit

## Cross-cutting checks executed

| Check | Result |
|-------|--------|
| Import-graph DAG (ARCH-08) | PASS — frame→∅, hmac→∅, admission→{frame,hmac}, routing→{frame,hmac,admission} |
| Sentinel collisions across packages | PASS — 8 sentinels across 4 packages, all string-distinct |
| UTC discipline (go.md #11) | PASS — only 2 time.Now() call sites in production code; both .UTC() |
| Adversary worktrees on factory-artifacts | PASS — S-2.01/12, S-2.02/8, S-1.03/5 passes |
| State-machine emergent flows | PASS — TestRegisterKey_AfterRevoke_ClearsRevokedFlag exercises LWW un-revoke path |

## Findings

### MEDIUM-001 — ReAuthState has no eviction hook for RevokeKey or RegisterKey-reset

**File:** `internal/admission/reauth.go:57-60`

The package itself flags this with `// TODO(phase-6): no eviction path; map grows monotonically with admitted nodes.` Cross-package consequence:
- RevokeKey() clears admitted=true but leaves ReAuthState.addrs[svtnID][nodeAddr] populated with stale source IP.
- RegisterKey() of an existing tuple resets admitted=false (correct LWW un-revoke) but leaves ReAuthState.addrs populated with stale IP.
- CurrentSourceAddr returns the stale prior IP. IsAdmitted gate at routing.go:99 protects RouteFrame, so currently not exploitable in the routing path. But CurrentSourceAddr is a publicly exported accessor with no cross-check against ks.IsAdmitted.

Confidence: HIGH. Severity: MEDIUM (data-consistency gap; tracked in code TODO but not in STATE.md drift register as a security-impact item).

### LOW-001 — Source-IP enforcement absent at router

**File:** `internal/routing/routing.go:97-104` (RouteFrame); `internal/admission/reauth.go:200` (CurrentSourceAddr)

RouteFrame gates on hdr.SrcAddr but does NOT verify wire-arrival source IP matches ReAuthState. By design: source-spoofing defense is HMAC tag (BC-2.05.005), wired-ready in verifyFrameHMAC (//nolint:unused) pending wave-3.

Cross-story consequence: until verifyFrameHMAC is wired in a future wave, Wave-2's router has zero frame-forgery defense. Acknowledged by team but no explicit STATE.md drift row.

### LOW-002 — Worktree convergence sanity: demo-evidence/per-ac-evidence.md missing for S-2.02 and S-1.03

S-2.01 has the artifact; S-2.02 and S-1.03 do not. The godoc Examples in example_test.go serve as functional demo-evidence, but the standard `.factory/cycles/cycle-1/S-X.YZ/demo-evidence/per-ac-evidence.md` artifact is absent.

(Also flagged by consistency-validator MEDIUM-002.)

### LOW-003 — Pass-count asymmetry across the wave (12/8/5) — process observation

S-2.01 = 12 adversary passes, S-2.02 = 8, S-1.03 = 5. All satisfied BC-5.39.001 ≥3 consecutive clean. Consistent with adversary novelty decay and team learning.

### OBSERVATION-001 — verifyFrameHMAC is dead-code-on-merge but test-covered

`internal/routing/routing.go:142-172` defines verifyFrameHMAC with //nolint:unused. The tautological-verify defect that surfaced in S-2.02 pass-4 (read wire tag BEFORE zeroing for MAC computation) is now correctly implemented. Wire-up pending Wave 3.

### OBSERVATION-002 — Lock-order: ks.mu acquired then released BEFORE rs.mu (no deadlock potential)

`internal/admission/reauth.go:167-196`: explicit Lock/Unlock (no defer) so ks.mu is fully released before rs.setSourceAddr acquires rs.mu. Textbook lock-discipline pattern.

### OBSERVATION-003 — Cross-package nonce-replay correctness

AdmitNode and ReAuthenticate share the same ks.nonces map via recordNonceUnlocked. An attacker cannot replay a nonce burned by initial admission as part of a re-auth attempt. No test explicitly pins this cross-flow invariant — a property test would harden this against future refactors.

## Wave-3 prep recommendations

1. Add MEDIUM-001 + LOW-001 as STATE.md drift items (this dispatch persists them).
2. Confirm verifyFrameHMAC wire-up is on Wave-3 critical path; test scaffolding is ready.
3. Consider adding a property test pinning the shared-nonce-map invariant across AdmitNode/ReAuthenticate.
