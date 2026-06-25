---
artifact_id: adv-S-1.03-pass-02
review_target: S-1.03-node-identity-session-continuity
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-1.03-node-identity-session-continuity
base: develop @ a06b306
tip: 66a1a9d
findings_count: 3
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 2 — S-1.03

## Medium Findings

### F-M1: ReAuthenticate docstring mis-anchors "node remains admitted; address unchanged" as PC1

**File:** `internal/admission/reauth.go:115`

BC-2.01.007 v1.3 PC1 = "The node initiates a re-authentication challenge to the router from its new IP." The docstring's claim "node remains admitted; cryptographic node address is unchanged" actually corresponds to PC4 (session traffic resumes) and/or Invariant 3 (session identity = channel ID + node addr, not IP).

This is the same mis-anchor pattern that pass-1 M-2 fixed for AC-003 in the story spec, now leaked into the code.

Suggested fix: `(PC1, VP-036)` → `(PC4 / Inv3, VP-036)`.

### F-M2: Test TestSessionContinuity_ReauthOnIPChange docstring + story AC-001 trace mis-anchor PC1

**File:** `internal/admission/reauth_test.go:31` and `.factory/stories/S-1.03-node-identity-session-continuity.md:49`

The test docstring claims "BC-2.01.007 postcondition 1 (session continues after re-auth from new IP)". But the test asserts:
- `ReAuthenticate` returns nil
- `IsAdmitted` true after re-auth (PC4)
- `CurrentSourceAddr` reflects new IP (PC3)

Story AC-001 itself (line 49) still traces to PC1. The pass-1 fix corrected AC-003 PC2→Inv3 but did not catch the same semantic class of error in AC-001 (and the code/test docstrings written from it). Story rev 1.3 needs AC-001 trace correction: PC1 → PC4 (or PC4 + PC3).

## Low Findings

### F-L1: Stale step numbering in ReAuthenticate (Steps 1, 2, 3, 5 — no Step 4)

**File:** `internal/admission/reauth.go:126, 146, 152, 188`

Numbered comment blocks skip Step 4. Likely a refactor leftover where two steps were merged. Renumber Step 5 → Step 4.

## Observations

- Nonce-before-verify alignment with AdmitNode correctly implemented (reauth.go:177 record, reauth.go:182 verify; mirrors admission.go:337 then :344). Same-nonce probe protection on re-auth path sound.
- ReAuthState.CurrentSourceAddr returns netip.Addr by value — no pointer leak.
- Lookup deep-clones PublicKey — locked-accessor contract preserved.
- TestReAuthenticate_NoRace exercises distinct-challenge concurrent re-auth correctly.
- VP-036 t.Skip grep-discoverable; deferral documented.
- All time.Now().UTC() — go.md rule 11 satisfied.

## Novelty Assessment

Novelty: MEDIUM. F-M1 and F-M2 identify a NEW mis-anchor in implementation docstring + test + story AC-001 that mirrors pass-1's AC-003 fix pattern but lands in DIFFERENT artifacts. Partial-fix regression — pass-1 spec patch corrected story-level AC-003 trace but the same semantic error was carried into AC-001 (line 49) and into the code/test docstrings.
