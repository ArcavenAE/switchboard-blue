---
artifact_id: adv-S-2.01-pass-01
review_target: S-2.01-hmac-codec
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-2.01-hmac-codec
base: develop @ 4be1b53
tip: 93cdc2c
findings_count: 9
findings_by_severity: {critical: 0, high: 4, medium: 3, low: 2, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-24
---

# Adversary Pass 1 — S-2.01 (HMAC codec)

## High

### F-001 — Missing VP-004 / VP-006 proptest harnesses
- Location: `.worktrees/S-2.01/internal/hmac/hmac_test.go` (no harnesses present); spec at `.factory/specs/verification-properties/VP-004.md:64-81` and `VP-006.md:64-85`.
- Evidence: Story `vp_traces: [VP-004, VP-005, VP-006]` and Task #9 mandate property tests; only one fuzz target (VP-005) exists.
- Impact: VP-004 (consistency: ComputeHMAC ↔ VerifyHMAC) and VP-006 (wrong-key rejection) are canonically defined in VP spec but not exercised in tests.
- Route: test-writer
- Fix: Add `TestPropComputeVerifyConsistency` and `TestPropVerifyHMAC_RejectsWrongKey` per VP harness skeletons.

### F-002 — VP-005 fuzz target semantic mismatch (AC-003 vs VP-005 conflict)
- Location: `.worktrees/S-2.01/internal/hmac/fuzz_test.go:24-31`
- Evidence: VP-005 specifies bit-flips in the **tag** across 64 bit positions (`.factory/specs/verification-properties/VP-005.md:31-33, 96-109`); current fuzz flips `frameBytes[0]`. AC-003 says "frame payload" (conflicts with VP-005).
- Impact: The canonical security property (tag forgery resistance via tag-bit perturbation) is not exercised. AC-003 wording conflicts with VP-005 — needs PO arbitration.
- Route: test-writer (after PO decides which to make canonical)
- Fix: **User decision (2026-06-24): cover BOTH** — keep existing frame-payload fuzz; ADD `FuzzVerifyHMAC_TagBitFlip` per VP-005 harness skeleton. AC-003 stays as frame-payload coverage.

### F-003 — DeriveKey per-(node, SVTN) key sensitivity not tested
- Location: `.worktrees/S-2.01/internal/hmac/hmac_test.go:69-78`
- Evidence: `TestDeriveKey_Deterministic` only verifies same-input → same-output. A constant-return implementation `return [32]byte{}` would pass. ARCH-04:175-180 specifies different pubkeys MUST produce different keys (per-node forge-resistance from BC-2.05.006).
- Impact: Core security property unguarded.
- Route: test-writer
- Fix: Add `TestDeriveKey_DistinctPubkeysProduceDistinctKeys` and `TestDeriveKey_DistinctSVTNsProduceDistinctKeys`.

### F-004 — Inlined HKDF without RFC 5869 Known-Answer Test + deviation from spec-mandated golang.org/x/crypto/hkdf
- Location: `.worktrees/S-2.01/internal/hmac/hmac.go:69-87`; story spec Library Requirements line 137 + ARCH-04:172.
- Evidence: Story spec MANDATES `golang.org/x/crypto/hkdf`; implementer inlined with stdlib-only justification. No KAT pins the inline algorithm against RFC 5869 test vectors.
- Impact: Spec deviation; silent breakage risk if HKDF refactored. **Same #260-family pattern**: orchestrator-side decision overrode MANDATORY story contract without spec amendment.
- Route: product-owner (amend Library Requirements) + test-writer (add RFC 5869 KAT)
- Fix: **User decision (2026-06-24): keep inline; PO amends story Library Requirements to permit inline HKDF; test-writer adds RFC 5869 KAT.**

## Medium

### F-005 — TestComputeHMAC_EightByteTag logs-not-fails on the all-zero case
- Location: `.worktrees/S-2.01/internal/hmac/hmac_test.go:34-38`
- Evidence: Test only verifies `[8]byte` return; if HMAC returns all-zeros it logs rather than fails. A constant non-zero return would pass.
- Impact: AC-001 test is weak; doesn't pin HMAC-SHA256 truncation correctness.
- Route: test-writer
- Fix: Replace log with `t.Fatalf`. Add an HMAC-SHA256 known-answer test using RFC 4231 test vectors (truncated to 8 bytes).

### F-006 — Stale EC-003 doc comment on VerifyHMAC
- Location: `.worktrees/S-2.01/internal/hmac/hmac.go:50`
- Evidence: Comment mentions "for a tag slice shorter than TagSize bytes (EC-003)" — impossible by signature since `tag` is `[TagSize]byte` (fixed 8 bytes).
- Impact: Misleading documentation; EC-003 was reformulated by the API but the doc still references the obsolete edge case.
- Route: implementer
- Fix: Rewrite comment to describe actual VerifyHMAC contract.

### F-007 — TestComputeHMAC_EmptyFrame has no positive assertion
- Location: `.worktrees/S-2.01/internal/hmac/hmac_test.go:104-109`
- Evidence: Test passes if function returns; doesn't verify "produces a valid 8-byte tag" per the story EC.
- Impact: EC-001 (empty payload) is structurally covered but not assertion-pinned.
- Route: test-writer
- Fix: Compute the expected HMAC of empty input with the test key; assert exact bytes.

## Low

### F-008 — TestVerifyHMAC_ShortTag is misnamed
- Location: `.worktrees/S-2.01/internal/hmac/hmac_test.go:92`
- Evidence: Test exercises "wrong tag value" not "short tag" (short tag is impossible by `[8]byte` signature).
- Impact: Naming confusion.
- Route: test-writer
- Fix: Rename to `TestVerifyHMAC_ZeroTagRejected` or similar.

### F-009 [process-gap] — Story MANDATORY library override resolved via code comment only
- Location: `.worktrees/S-2.01/internal/hmac/hmac.go:69-87` (inline HKDF) vs story Library Requirements line 137 (mandates external lib).
- Evidence: No spec-amendment trail. Implementer chose to inline; orchestrator dispatched with that permission. Adjacent to drbothen/vsdd-factory#260.
- Impact: Same family as #260 — orchestrator-side override of documented MANDATORY contract without surfacing for human approval.
- Route: orchestrator + product-owner
- Fix: PO amends story Library Requirements (resolves spec drift). Orchestrator's prompt should flag MANDATORY-section deviations in future dispatches.

## Routing decisions (per 2026-06-24 user approval after #260-aware re-prompting)

- **F-004 / F-009**: Keep inline HKDF; PO amends story; test-writer adds RFC 5869 KAT.
- **F-002**: Cover BOTH coverage angles (frame-payload + tag-bit fuzz targets).
- Remaining findings (F-001, F-003, F-005, F-006, F-007, F-008): single combined fix burst.

Convergence streak: 0/3. Pass 2 follows the fix burst.
