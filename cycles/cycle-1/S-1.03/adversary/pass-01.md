---
artifact_id: adv-S-1.03-pass-01
review_target: S-1.03-node-identity-session-continuity
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-1.03-node-identity-session-continuity
base: develop @ a06b306
tip: 6ae2cdf
findings_count: 6
findings_by_severity: {critical: 0, high: 2, medium: 3, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 1 — S-1.03

## High Findings

### H-1: Test docstring cites WRONG error code (E-ADM-002) and WRONG BC edge case (BC EC-001) for the expired-key test — direct contradiction of story rev 1.1

**File:** `internal/admission/reauth_test.go:230-234`

Three independent defects:
1. **Wrong BC edge case.** BC-2.01.007 EC-001 is "IP changes while upstream frames are in-flight". The expired-key edge case is BC EC-005 ("Node re-authenticates after key expiry... E-ADM-015").
2. **Wrong error code.** The docstring cites `E-ADM-002` — that is the HMAC verification failed error, unrelated to key expiry. Correct code per story rev 1.1, BC v1.2, ARCH-04 v1.3, and error-taxonomy is **E-ADM-015**.
3. **Phantom ambiguity-flag reference.** The docstring says "see ambiguity flag in reauth.go", but reauth.go contains no ambiguity flag.

### H-2: reauth.go and reauth_test.go systematically mis-anchor old-path eviction to BC-2.01.007 EC-002 (which is actually "router unreachable / E-NET-003")

**Files:**
- `internal/admission/reauth.go:68-69, 107, 117, 188`
- `internal/admission/reauth_test.go:282-286`

BC EC-002 is router-unreachable / E-NET-003, not eviction. Story EC-002 has no dedicated BC anchor — resolution decision required (mint BC EC-006 OR reword to BC EC-003 LWW / PC3).

## Medium Findings

### M-1: Story AC-002 cites BC-2.01.007 precondition 1, but the wrong-keypair semantics belong to precondition 3

**File:** `.factory/stories/S-1.03-node-identity-session-continuity.md:52-54`

BC Pre1 = "active session exists". BC Pre3 = "keypair unchanged" — the correct anchor.

### M-2: Story AC-003 cites BC-2.01.007 postcondition 2, but node-address-stability is BC invariant 3

**File:** `.factory/stories/S-1.03-node-identity-session-continuity.md:56-57`

BC PC2 = "router verifies signature". BC Inv3 = "Session identity is channel ID + cryptographic node address, not IP 4-tuple" — the correct anchor.

### M-3: TestReauth_LastWriteWins is purely sequential — docstring claim "concurrent variant covered by race detector" is false; no concurrent ReAuthenticate test exists

**File:** `internal/admission/reauth_test.go:355-357, 363-418`

Test is single-goroutine. BC EC-003 explicitly names "concurrent". No `TestReAuthenticate_NoRace` exists.

## Low Findings

### L-1: Asymmetric nonce-consume semantics between AdmitNode and ReAuthenticate

**Files:**
- `internal/admission/admission.go:337-346` (records nonce BEFORE sig-verify)
- `internal/admission/reauth.go:157-184` (verifies BEFORE recording — failed sig does not burn nonce)

Diverges from BC-2.05.001 invariant 3 ordering pattern documented for AdmitNode.

## Observations

- BC-2.01.007 lacks a dedicated edge case for "old path eviction" — resolving H-2 requires either adding BC EC-006 or rewording anchors.
- VP-036 t.Skip deferral correctly handled.
- Lock discipline good (no ks.mu + rs.mu held simultaneously; locked-accessor contract preserved; UTC; constant-time crypto via stdlib).

## Novelty Assessment

Novelty: HIGH — first-pass findings. Pattern: rev-1.1 spec patch landed in primary anchors (reauth.go body, ARCH-04 v1.3, BC v1.2) but stale text persists in test docstring + story AC trace fields (partial-fix regression discipline target).

## Resolution decisions (from human review)

- H-2: Mint BC-2.01.007 EC-006 "old path evicted on new re-auth"; bump BC to v1.3; update 5 anchor sites.
- M-3: Add TestReAuthenticate_NoRace concurrent test mirroring TestAdmitNodeRevokeKey_NoRace pattern.
- L-1: Align ReAuthenticate to AdmitNode pattern (record-then-verify); update rationale comment.
- H-1, M-1, M-2: Unambiguous corrections; no decision required.
