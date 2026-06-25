---
artifact_id: adv-S-2.02-pass-02
review_target: S-2.02-admission-svtn-isolation
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-2.02-admission-svtn-isolation
base: develop @ 3c4104e
tip: 32a165d
findings_count: 8
findings_by_severity: {critical: 0, high: 0, medium: 4, low: 4, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
---

# Adversary Pass 2 — S-2.02 (Admission + SVTN isolation)

## Medium

### M-1 — Test docstring AC-004 trace cites BC-2.05.002 PC1 instead of PC2
- Location: `.worktrees/S-2.02/internal/routing/routing_test.go:51-52`
- Evidence: TestRouteFrame_DropsUnadmitted docstring says "Traces to BC-2.05.002 postcondition 1 (frame from non-admitted source → dropped; E-ADM-003 logged)." Story rev 1.1 line 64-65 (AC-004) cites BC-2.05.002 postcondition 2. PC1 = forward-on-admitted (happy path); PC2 = drop-on-not-admitted. Test verifies PC2.
- Impact: Pin-point trace mis-anchor. Story-spec H-3 patch fixed STORY but missed test docstring.
- Route: test-writer
- Fix: Change docstring to cite "BC-2.05.002 postcondition 2".

### M-2 — VP-trace mis-anchor in admission fuzz section header
- Location: `.worktrees/S-2.02/internal/admission/admission_test.go:470` (section header) vs `:476` (docstring)
- Evidence: Section header says "Fuzz harness: VP-009 — admission rejects unregistered keys" but VP-009 is "Admission Rejects Replayed Nonce". Fuzz function FuzzAdmitNode_UnregisteredKey correctly tests VP-008. Docstring at line 476 says VP-008 (correct). Header mismatched.
- Route: test-writer
- Fix: Update section header to "VP-008".

### M-3 — VP-trace mis-anchor in routing fuzz target (both header and docstring)
- Location: `.worktrees/S-2.02/internal/routing/routing_test.go:264, 270`
- Evidence: Section header AND function docstring both say "VP-057 — no frame from non-admitted source reaches any destination". VP-057 is "Node Private Keys Never Appear as Literal Bytes in Any Emitted Frame" — about private key non-transit. The fuzz tests unadmitted-source rejection = VP-008. Result: VP-057 has only one effective covering test (TestGenerateChallenge_NoChallengeContainsPrivateKey).
- Route: test-writer
- Fix: Change both references to VP-008.

### M-4 — TestRegisterKey_AfterRevoke_ClearsRevokedFlag does not actually verify un-revoke
- Location: `.worktrees/S-2.02/internal/admission/admission_test.go:441-468`
- Evidence: After RegisterKey→RevokeKey→RegisterKey, test calls RevokeKey again and asserts no error. But RevokeKey unconditionally sets `revoked=true` and returns nil whenever entry is present — does NOT inspect prior `revoked` state. Test would pass even if RegisterKey did NOT clear `revoked`. Implementer flagged this concern explicitly.
- Route: test-writer
- Fix: Rewrite with full handshake. Register → Revoke → assert ErrKeyRevoked on AdmitNode → re-Register → assert AdmitNode succeeds with fresh challenge+sig.

## Low

### L-1 — Self-addressed guard returns ErrNotAdmitted (semantically wrong code; story doesn't request this guard)
- Location: `.worktrees/S-2.02/internal/routing/routing.go:126-129, 137-146`
- Evidence: When hdr.DstAddr == hdr.SrcAddr, SVTNRoute returns admission.ErrNotAdmitted (E-ADM-003 "frame from non-admitted source") despite source being admitted. BC-2.02.008 (split-horizon) explicitly deferred per comment. No AC requests this guard. No test covers it.
- Route: implementer (per USER DECISION 2026-06-25: REMOVE the guard entirely)
- Fix: Delete isSelfAddressed function + callers. Self-loops route normally per BC-2.02.001/002.

### L-2 — LWW re-registration silently un-admits (unspecified in ADR-003)
- Location: `.worktrees/S-2.02/internal/admission/admission.go:139-159`
- Evidence: RegisterKey creates `&AdmittedKey{...}` with `admitted` zero-initialized (false). LWW re-registration on previously admitted entry replaces old via map assignment — silently dropping prior admitted=true. ADR-003 doesn't specify preserve-vs-reset.
- Route: product-owner (per USER DECISION 2026-06-25: RESET behavior is intentional security default; document in spec — no code change)
- Fix: PO amends ADR-003 in ARCH-04 OR story body to make explicit: "LWW re-registration resets admitted=false; node must re-handshake. Security-by-default: any key change forces re-authentication."

### L-3 — Fuzz targets do not exercise the input space they declare
- Location: `.worktrees/S-2.02/internal/admission/admission_test.go:477-510`; `.worktrees/S-2.02/internal/routing/routing_test.go:272-320`
- Evidence: FuzzAdmitNode_UnregisteredKey discards its `[]byte` fuzz input (`_ []byte`) and regenerates keys via rand.Reader each iteration. FuzzRouteFrame_NonAdmittedNeverForwarded uses only 2 bytes for SVTN/dst address; keys still randomly generated. Effective: randomized loop, not input-driven fuzz.
- Route: test-writer
- Fix: Drive Ed25519 keys from corpus bytes (or seed from the fuzz input via deterministic expansion). Document the corpus expectations in test godoc.

### L-4 — Timing-oracle defence rationale in comment is incorrect
- Location: `.worktrees/S-2.02/internal/admission/admission.go:303-309`
- Evidence: Comment says "Record nonce BEFORE verifying to prevent timing oracle." Recording-before-verify means replay path skips ~50μs ed25519.Verify — making replay detection MORE timing-distinguishable, not less. Actual benefit: CPU DoS mitigation (reject replays without spending compute).
- Route: implementer
- Fix: Replace comment with accurate rationale (CPU DoS mitigation; not timing oracle).

## Observations (carry forward — all clean)

- recordNonceUnlocked lock-discipline invariant VERIFIED CORRECT (caller at admission.go:326 holds the write lock from line 315).
- H-1 race-fix audit VERIFIED CORRECT (snapshot-inside-RLock + re-fetch + re-check revoked under write lock).
- M-3 deep-clone audit VERIFIED CORRECT (only PublicKey is slice; single append clone suffices).
- L-1 rename audit VERIFIED (single call-site updated).

## Routing decisions (USER 2026-06-25, surfaced not announced)

- L-1: REMOVE self-addressed guard (Option a; story doesn't request).
- L-2: RESET admitted on LWW is intentional default; PO documents (Option a; no code change).
- All findings: single combined fix burst.

Convergence streak: 0/3. Pass 3 follows fix burst.
