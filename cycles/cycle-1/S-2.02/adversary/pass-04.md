---
artifact_id: adv-S-2.02-pass-04
review_target: S-2.02-admission-svtn-isolation
producer: adversary
pass: 4
fresh_context: true
branch: feature/S-2.02-admission-svtn-isolation
base: develop @ 3c4104e
tip: a888d0b
findings_count: 3
findings_by_severity: {critical: 0, high: 1, medium: 2, low: 0, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
---

# Adversary Pass 4 — S-2.02 (Admission + SVTN isolation)

## High

### H-1 — verifyFrameHMAC is tautological — verifies a freshly-computed tag against itself, never consults wire tag
- Location: `.worktrees/S-2.02/internal/routing/routing.go:139-153`
- Evidence:
  ```go
  func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool {
      hdrForMAC := hdr
      hdrForMAC.HMACTag = [8]byte{}                // clear tag (correct so far)
      encoded := frame.EncodeOuterHeader(hdrForMAC)
      msg := make([]byte, len(encoded)+len(payload))
      copy(msg, encoded[:])
      copy(msg[len(encoded):], payload)
      tag := hmac.ComputeHMAC(authKey[:], msg)      // L151: compute tag
      return hmac.VerifyHMAC(authKey[:], msg, tag)  // L152: verify the just-computed tag against itself
  }
  ```
  Function never reads `hdr.HMACTag` (the wire tag). It computes the expected MAC and immediately verifies that expected MAC against itself. Tautological — always returns true.
  Docstring claims "Fail-closed: returns false on any verification failure" — false. Returns true for every input including frames with any attacker-chosen HMACTag.
  `//nolint:unused` (L138) actively suppresses the warning that would have flagged this for review.
- Impact: When wired into RouteFrame next wave, BC-2.05.002 invariant 1 ("Every frame carrying SVTN-scoped traffic is verified against the admitted key set at the first router. No exceptions.") + BC-2.05.005 fail-closed will be SILENTLY BYPASSED at the HMAC verification layer. An attacker could forge any HMAC.
- Route: implementer (fix function) + test-writer (add regression test asserting verifyFrameHMAC returns false for wrong tag)
- Fix:
  ```go
  func verifyFrameHMAC(hdr frame.OuterHeader, payload []byte, authKey [hmac.KeySize]byte) bool {
      wireTag := hdr.HMACTag                         // SAVE wire tag BEFORE clearing
      hdrForMAC := hdr
      hdrForMAC.HMACTag = [8]byte{}
      encoded := frame.EncodeOuterHeader(hdrForMAC)
      msg := make([]byte, len(encoded)+len(payload))
      copy(msg, encoded[:])
      copy(msg[len(encoded):], payload)
      return hmac.VerifyHMAC(authKey[:], msg, wireTag)  // verify WIRE tag, not freshly-computed
  }
  ```
  Add test: verifyFrameHMAC returns false when hdr.HMACTag is wrong (e.g., all-zeros against non-zero-MAC).

## Medium

### M-1 — SVTNRoute returns ErrNotAdmitted (E-ADM-003) for "destination not in forwarding table" — semantic misuse of sentinel
- Location: `.worktrees/S-2.02/internal/routing/routing.go:121-124`
- Evidence:
  ```go
  if entry == nil {
      // DstAddr not in forwarding table for this SVTN — SVTN isolation enforced.
      return admission.ErrNotAdmitted
  }
  ```
  Per error-taxonomy.md L52, E-ADM-003 is canonically "frame from non-admitted source." Implementation returns same sentinel when DESTINATION address not in forwarding table. RouteFrame (L91) and SVTNRoute (L123) both return admission.ErrNotAdmitted but for different reasons. errors.Is(err, ErrNotAdmitted) cannot distinguish.
- Impact: TestRouteFrame_AdmittedSetCheckPrecedesForwarding (routing_test.go:233-262) passes by detecting ErrNotAdmitted but cannot prove which check fired. Future logging/metrics granularity blocked.
- Route: implementer (new sentinel) + test-writer (precision pin)
- Fix: Introduce routing.ErrNoForwardingEntry (or BC-2.05.006-specific sentinel); SVTNRoute returns it for forwarding-table miss; update TestRouteFrame_AdmittedSetCheckPrecedesForwarding to assert the precise sentinel that proves precedence order.

### M-2 — AC-006 / AC-007 / VP-007 tests are structurally weak — zero-value field-length checks, not byte-substring property test
- Location: `.worktrees/S-2.02/internal/admission/admission_test.go:155-173` (AC-006); `:183-223` (AC-007)
- Evidence:
  - VP-007 specifies: ∀ privKey [64]byte ..., challenge [32]byte: let resp = SignChallenge(privKey, challenge); let wire = MarshalChallengeResponse(resp); ¬ (∃ i : wire[i:i+32] == privKey[0:32] ∨ wire[i:i+32] == privKey[32:64])
  - VP-007 harness skeleton uses gopter prop.ForAll + containsSubslice byte-scan.
  - AC-006 test only checks `len(Challenge.Nonce) == 32` and `len(ChallengeResponse.NonceSig) != 64` on zero-value structs. No keygen, no signing, no wire serialization, no byte-scan.
  - AC-007 test compares Nonce byte-for-byte against privBytes[:32], RouterSig byte-for-byte against privBytes — only at byte position 0. Does NOT byte-scan for any 32-byte substring at any offset.
- Impact: Verification chain to VP-007 broken. Tests pass for any implementation including one that intentionally embedded private-key bytes outside [32]byte Nonce. AC-006 declared as "property test" but is not — runs once, tests neither field absence nor encoded representation.
- Route: test-writer
- Fix options:
  1. Add gopter-based property test matching VP-007 harness skeleton (requires gopter dep — likely deferred).
  2. Document VP-007 as proof-deferred-to-S-X.YY in story; replace AC-006/AC-007 tests with serialized-wire byte-scan using random keys (statistical evidence) WITHOUT gopter dependency.
  Recommend Option 2 (stdlib-only) for this story.

## Observations (non-blocking, documented for future)

- admission.go:302-312 — nonce-before-verify comment self-contradicts (says "CPU DoS mitigation" then "NOT a timing-oracle defence"). Reader confusion. Cosmetic.
- admission.go:171, 175 — RevokeKey returns ErrNotAdmitted for "key not found." Per error-taxonomy.md L62, canonical sentinel is E-ADM-013. Out of strict story scope (BC-2.05.004 not in bc_traces); defer to wave-gate.
- admission_test.go:47 + routing_test.go:36 — both test files re-implement frame.DeriveNodeAddress to avoid internal/frame import. Both source files already import frame. Test-helper duplication = drift risk.
- routing.go:89, 112 — RouteFrame/SVTNRoute are package functions taking `r *Router` rather than methods. Inconsistent with Router.RegisterForwardingEntry. Style residue.
- routing_test.go:151 — `tc := tc` shadowing no longer needed in Go 1.22+ (project pins 1.25.4). Cosmetic.

## Convergence

Streak reset to 0/3. Pass 5 follows fix burst.

H-1 is a high-priority latent crypto defect masked by `//nolint:unused`. The next wave that wires verifyFrameHMAC into RouteFrame would have silently broken HMAC verification across the entire admission pipeline.
