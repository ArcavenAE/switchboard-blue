---
artifact_id: adv-S-2.02-pass-01
review_target: S-2.02-admission-svtn-isolation
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-2.02-admission-svtn-isolation
base: develop @ 3c4104e
tip: de7ecee
findings_count: 9
findings_by_severity: {critical: 0, high: 3, medium: 3, low: 2, nitpick: 1}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
---

# Adversary Pass 1 — S-2.02 (Admission + SVTN isolation)

## High

### H-1 — Data race on AdmittedKey.revoked (TOCTOU + Go memory-model violation)
- Location: `.worktrees/S-2.02/internal/admission/admission.go:254-270` (AdmitNode) and `:151-165` (RevokeKey)
- Evidence: AdmitNode acquires RLock, reads `existingEntry := svtnMap[nodeAddr]` (a *AdmittedKey), releases RUnlock, then reads `existingEntry.revoked` (line 268). RevokeKey acquires Lock then writes `entry.revoked = true` (line 163) on the same *AdmittedKey value owned by the map. Two concurrent accesses on the same memory location without synchronization.
- Impact: Go memory-model violation. `go test -race` will flag it as soon as any caller invokes AdmitNode and RevokeKey concurrently. Semantically: between RUnlock and revoked read, concurrent RevokeKey can mark entry revoked → AdmitNode sees stale revoked=false → admits a revoked node. Current tests don't exercise this race (CI passes by accident).
- Route: implementer
- Fix: Read `revoked` while still holding the lock — either check inside RLock-held block at line 254-260, OR snapshot `existingEntry := *entry` (value copy) inside RLock so subsequent `existingEntry.revoked` is on local copy. Pattern (b) matches Lookup at line 183.

### H-2 — BC-2.05.001 postcondition 4 not implemented (active node set distinction collapsed)
- Location: `admission.go:243-285` (AdmitNode) and `:187-204` (IsAdmitted)
- Evidence: BC-2.05.001 PC4: "On success: node is added to the router's active node set for this SVTN." ARCH-04 step 4 distinguishes `admitted_key_set[svtn_id]` from `admitted_nodes[svtn_id]`. AdmitNode body performs NO map mutation on success — only verifies. Only RegisterKey adds to the keys map. IsAdmitted returns true after RegisterKey alone, even if no challenge handshake ever took place. TestSVTNRoute_AdmittedFrameForwardedToCorrectSVTN demonstrates this — calls RegisterKey, never AdmitNode, yet SVTNRoute returns nil.
- Impact: BC postcondition not satisfied. In Wave 2 where HMAC verification not yet wired (`verifyFrameHMAC` is `//nolint:unused`), a node whose pubkey has been registered is routable WITHOUT completing challenge-response. Wave-3 HMAC enforcement will close most of the gap, but BC-as-written contract is not delivered by this story.
- Route: implementer (USER DECISION 2026-06-25: implement two-state model — add `admitted bool` to AdmittedKey defaulting to false at RegisterKey; AdmitNode sets true on success; IsAdmitted AND-gates on it)
- Fix: Add `admitted bool` field to AdmittedKey; RegisterKey leaves it false; AdmitNode sets true after signature verification + nonce record success; IsAdmitted checks `entry.admitted && !entry.revoked`.

### H-3 — Story spec AC trace anchors and EC error codes systematically wrong
- Location: `.factory/stories/S-2.02-admission-svtn-isolation.md:55-93`
- Evidence (cross-checked against BC-2.05.001/002 and error-taxonomy.md):
  - Line 55 — AC-002 traces to "precondition 1" (which is "node has keypair"). Should be **postcondition 5**.
  - Line 59 — AC-003 traces to "invariant 1" (DI-002 private keys never leave). Should be **invariant 3** (nonce single-use).
  - Line 60 — AC-003 says E-ADM-003 for nonce replay. Should be **E-ADM-008** (E-ADM-003 is "frame from non-admitted source").
  - Line 63 — AC-004 traces to "postcondition 1" (happy path). Should be **postcondition 2** (drop-on-unadmitted).
  - Line 64 — AC-004 says E-ADM-005 for RouteFrame non-admitted. Should be **E-ADM-003** (E-ADM-005 is "key revoked").
  - Line 91 — EC-001 says E-ADM-005 for unregistered SVTN key. Should be **E-ADM-003**.
  - Line 93 — EC-003 says E-ADM-002 for revoked-key admission. Should be **E-ADM-005** (E-ADM-002 is "HMAC verification failed").
- Impact: 3 of 7 AC trace anchors + 2 of 3 EC error-code citations are factually wrong. Implementer code-corrected silently but test docstrings (admission_test.go:90, 120, etc.) propagate the wrong anchors. Semantic anchoring rubric: HIGH; never block as Observation.
- Route: product-owner

## Medium

### M-1 — VP anchor mis-attribution in test docstrings
- Location: `.worktrees/S-2.02/internal/admission/admission_test.go:326, 383`
- Evidence: Line 326 (TestGenerateChallenge_NonceUniqueness) docstring says "VP-007, VP-009." VP-007 is "Admission Private Key Bytes Never Appear in ChallengeResponse Wire Struct" (private-key absence); VP-009 is "Admission Rejects Replayed Nonce" (depends on uniqueness but not itself a uniqueness property). Line 383 (FuzzAdmitNode_UnregisteredKey) claims "Traces to VP-009 (admission fails for any key not in the admitted set)" — that's VP-008 ("Admission Fails for Unregistered Key").
- Route: test-writer
- Fix: Line 326 cite "BC-2.05.001 invariant 3 (nonce uniqueness)"; line 383 cite VP-008.

### M-2 — recordNonce O(N) purge loop on every AdmitNode call
- Location: `admission.go:289-306`
- Evidence: Every call iterates full nonces map to purge expired entries. At steady state |nonces| ≈ M × 60s, so cost is O(M²×60s) work/sec total.
- Impact: Quadratic-amplification path under sustained admission load; DoS surface in PE phase.
- Route: implementer
- Fix: Lazy-purge gated by `if len(s.nonces) > threshold` or `if now.Sub(lastPurge) > 1s`. Simplest correct fix.

### M-3 — Lookup shallow-copies ed25519.PublicKey (slice; backing array shared)
- Location: `admission.go:170-185`
- Evidence: `cp := *entry` is shallow. PublicKey is `ed25519.PublicKey` (alias for []byte). Slice header copied; backing array shared with live map entry. Caller mutation → races against any goroutine holding lock and reading PublicKey.
- Impact: Partial enforcement of go.md rule 12. Dormant in current code paths but Lookup docstring claims "callers do not hold a pointer into internal state" — false for PublicKey backing array.
- Route: implementer
- Fix: `cp.PublicKey = append(ed25519.PublicKey(nil), entry.PublicKey...)`.

## Low

### L-1 — splitHorizon predicate is self-addressed-frame check, not split-horizon
- Location: `internal/routing/routing.go:137-143, 127-129`
- Evidence: `return hdr.DstAddr == arrivalNodeAddr`; caller passes `hdr.SrcAddr` as `arrivalNodeAddr` — predicate fires only when DstAddr == SrcAddr (self-loop). BC-2.02.008 defines split-horizon as interface-based, not address-based. Story `bc_traces` doesn't include BC-2.02.008.
- Impact: Function name + BC-2.02.008 forward-reference comment misleads future implementer.
- Route: implementer
- Fix: Rename to `isSelfAddressed`; defer real split-horizon to BC-2.02.008 implementation story.

### L-2 — Re-registering revoked key silently un-revokes; no test pins LWW semantic
- Location: `admission.go:127-145`; TestDuplicateKeyRegistration_LastWriteWins only tests role replacement
- Evidence: RegisterKey constructs fresh AdmittedKey with zero-valued revoked=false; overwrites map slot. Behavior is ADR-003 last-write-wins side effect but no test pins it.
- Route: test-writer
- Fix: Add TestRegisterKey_AfterRevoke_ClearsRevokedFlag verifying register→revoke→register→IsAdmitted==true.

## Nitpick

### N-1 — Stale //nolint:staticcheck comments in test files
- Locations: admission_test.go (11 sites); routing_test.go (13 sites)
- Evidence: Variables protected by nolint comments ARE consumed in live tests now (post-stub TDD). Comments are leftover from stub phase. golangci-lint unused-nolint not enabled, so these survive.
- Route: test-writer
- Fix: Sweep both test files; remove every //nolint:staticcheck whose target variable is now used; remove dead `_ = varname` blank assignments.

## Implementer-flagged concern verdicts

1. TOCTOU race in AdmitNode — CONFIRMED → H-1.
2. Forwarding-entry pointer stability — NOT A DEFECT today but fragile; snapshot recommended.
3. splitHorizon semantics — CORRECT to flag → L-1.

## Routing decisions (per 2026-06-25 user approval)

- H-2: implement two-state model (Option a).
- All findings: single combined fix burst (state-manager + implementer + PO + test-writer parallel).

Convergence streak: 0/3. Pass 2 follows fix burst.
