---
artifact_id: adv-S-2.02-pass-05
review_target: S-2.02-admission-svtn-isolation
producer: adversary
pass: 5
fresh_context: true
branch: feature/S-2.02-admission-svtn-isolation
base: develop @ 3c4104e
tip: c744d54
findings_count: 3
findings_by_severity: {critical: 0, high: 0, medium: 0, low: 3, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
---

# Adversary Pass 5 — S-2.02 (Admission + SVTN isolation)

## Low

### L-1 — Stale comment in AdmitNode claims signature verification happens outside the lock; actual code holds write lock through ed25519.Verify
- Location: `.worktrees/S-2.02/internal/admission/admission.go:302` (comment) vs `:336` (Verify call inside write-lock at :318)
- Evidence: Line 302 says `// Step 2: verify signature outside the lock — pure computation, no shared state.` Actual ed25519.Verify is invoked at line 336, INSIDE the write-lock section beginning at line 318 (`ks.mu.Lock(); defer ks.mu.Unlock()`). Inline comments at 333-335 correctly explain the verify-inside-lock design (preserves nonce-consume-before-verify invariant), so the Step 2 header is stale.
- Impact: Doc-comment mismatch confuses maintainers about the actual lock-hold time.
- Route: implementer (1-line comment fix)
- Fix: Replace line 302 comment with accurate description: verify is INSIDE the write lock as part of the atomic nonce-consume + verify + admit critical section.

### L-2 — RevokeKey returns ErrNotAdmitted (E-ADM-003 frame-routing sentinel) for "key not registered" condition
- Location: `.worktrees/S-2.02/internal/admission/admission.go:165-178` (RevokeKey body); admission.go:38-41 (ErrNotAdmitted doc binding to E-ADM-003)
- Evidence: RevokeKey returns ErrNotAdmitted at lines 171 + 175 when svtnID or nodeAddr is unknown. ErrNotAdmitted is documented as "frame from non-admitted source (E-ADM-003; BC-2.05.002 postcondition 2)" — frame-routing sentinel, not key-lifecycle. error-taxonomy.md:62 defines E-ADM-013 ("key not found") as the correct code; no Go sentinel for E-ADM-013 exists.
- Impact: errors.Is(err, ErrNotAdmitted) will conflate frame-routing rejection with revoke-of-unknown-key when downstream code (e.g., S-2.04 key-lifecycle) introduces revoke flows.
- Route: implementer
- Fix (USER DECISION 2026-06-25): Add ErrKeyNotRegistered sentinel; RevokeKey returns it for unknown svtnID/nodeAddr; test-writer adds precision test. Same pattern as M-1 fix in pass-4.

### L-3 — VP-057 in story vp_traces array but no test cites it; trace hygiene gap
- Location: `.factory/stories/S-2.02-admission-svtn-isolation.md:22` (vp_traces lists VP-057); story task 8 (line 129) mentions VP-057; `grep -r VP-057` over `.worktrees/S-2.02/` returns 0 hits
- Evidence: Closest test is TestProperty_VP007_PrivateKeyByteSubstringAbsent (admission_test.go:243) which only inspects ChallengeResponse.NonceSig. VP-057 spec requires coverage across {DATA, EMPTY_TICK, ADMISSION_CHALLENGE, ADMISSION_RESPONSE, CONTROL_DRAIN, CONTROL_KEY_REG, CONTROL_KEY_REVOKE} — frame types mostly not yet defined in internal/frame.
- Impact: VP listed in vp_traces should either be cited by at least one test or explicitly carved out as deferred. As written, VP-057 silently free-rides on VP-007 coverage of admission wire structs only.
- Route: product-owner (story carve-out) + test-writer (test docstring cite)
- Fix (USER DECISION 2026-06-25): Option (a) — add deferral cite to existing VP-007 test. PO updates task 8 wording: VP-057 admission-wire-struct subset is covered by TestProperty_VP007; full frame-type extension deferred to wave where DATA/CTL/ARQ/FEC frames are emitted. Test-writer adds `// Traces to VP-007 (admission scope) + VP-057 (admission subset; full frame-type coverage deferred)` comment.

## Observations

- Core admission/isolation/lock-cloning/nonce-replay/fail-closed/HMAC-tautology logic is sound. No correctness, concurrency, or security defects found.
- AdmitNode write-lock holds the verify (~50μs) by design (per inline comments at 333-335) — choice is defensible (atomic nonce + verify + admit critical section).

## Routing decisions (USER 2026-06-25)

- L-1: implementer 1-line comment fix.
- L-2: add ErrKeyNotRegistered sentinel now (Option a — same pattern as pass-4 M-1).
- L-3: deferral cite + task 8 wording update (Option a — PO updates story; test-writer adds docstring cite).

Convergence streak: 0/3. Pass 6 follows fix burst.
