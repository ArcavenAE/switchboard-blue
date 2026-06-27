---
artifact_id: adv-S-3.03-pass-05
review_target: S-3.03-tier2-session-authorization
producer: adversary
pass: 5
fresh_context: true
branch: feature/S-3.03-tier2-session-authorization
findings_count: 6
findings_by_severity: {critical: 0, high: 0, medium: 2, low: 1, observations: 3}
verdict: CONVERGED
streak_after_pass: 4
streak_reset_reason: null
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 5 — S-3.03

## Disposition

CONVERGED — zero CRITICAL, zero HIGH. Streak: 4 consecutive clean passes (2,3,4,5) — Step 4.5 convergence criterion (3 clean) EXCEEDED. Reviewed tip: 9992276.

Fresh-context adversary attacked all 8 surfaces and documented attack-by-attack that it could NOT find any critical/high. Enforcement core confirmed across 5 independent passes. Production code frozen since 0a94efd (pass-5 tip only added the AC-006 forwarding test).

## Verified-Clean (attack-by-attack)

1. **Attach gate ordering** — authorize before `consoles.Add`; `(nil,nil,err)` on denial; no partial state.
2. **Read-only SendKeystroke** — payload rejected E-ADM-007; empty-tick forwarded; unregistered key + empty-tick still DENIED (Allow→Authorize first, decision-matrix row).
3. **Per-session isolation** — `sessions[name][key]`, no SVTN structure (PC-4/VP-013).
4. **Concurrency/lock ordering** — `sinkMu→ConsoleSet.mu→SessionAuth.mu` never reversed/nested; `sinkMu` closes TOCTOU; no rule-12 leak.
5. **Error messages** — E-ADM-006/007 interpolate key+session, `%w` wrap, ST1005 clean; "not found for session" (E-SES-003); v1.6 layering note honored.
6. **VP-012** — routing imports `{errors,sync,admission,frame,hmac}`, no session/Tier-2 symbols; static audit test.
7. **Test integrity** — AC-006 empty-tick test genuinely asserts sink receipt (non-vacuous); enforcement tests mutation-resistant.
8. **ACs 001-006 + Task 7** (SessionAuth wired as live Authorizer, compile-guard `TestSessionAuth_ImplementsAuthorizer`) all satisfied.

## Findings (NON-BLOCKING)

Medium does not reset streak; findings are coverage/mutation-resistance gaps, not logic defects.

### M-1 (MEDIUM) — No positive attach test for read-only key

**Spec reference:** BC-2.04.005

No positive test that an authorized-but-READ-ONLY console attaches (only `RoleFull` positive test exists). Read-only attach passes only via the empty-payload exemption, coupling attach-admission coverage to the empty-tick branch.

**Resolution:** test-writer adding `TestAccessNode_Attach_ReadOnlyKey_Succeeds` (pass-5).

**Status:** OPEN (non-blocking) — test-writer task dispatched.

### M-2 (MEDIUM) — No -race test for concurrent SessionAuth mutation

**Spec reference:** BC-2.05.003 EC-004

No `-race` test mutating `SessionAuth` concurrently (`RegisterKey` vs `Allow`/`Authorize`); a dropped `RLock` would not be caught. BC-2.05.003 EC-004 contemplates mid-session auth changes.

**Resolution:** test-writer adding `TestSessionAuth_ConcurrentRegisterAndAuthorize` (pass-5).

**Status:** OPEN (non-blocking) — test-writer task dispatched.

### O-1 (LOW) [process-gap] — VP-skeleton stale API references

**Scope:** spec-side only — out of worktree perimeter.

VP-013/VP-035 proof-harness skeletons reference a stale API (`session.NewAuthList`/`AddWithRole`/`AuthorizeFrameType`/`ErrReadOnlyUpstreamDenied`) and VP-035's property statement cites E-ADM-003 for read-only upstream rejection, but the canonical taxonomy maps it to E-ADM-007 (E-SES-005 retired). Implementation correctly uses E-ADM-007; the VP skeletons were not refreshed when the error code was re-anchored.

**Resolution:** DEFERRED — tracked as drift item S-3.03-O1-VPSKEL; codification follow-up at cycle-close.

**Status:** DEFERRED (process-gap, out of perimeter).

## Observations

### O-2 (OBS) — Vestigial upstream channel (S-3.02-FM1)

Upstream channel dangling-but-documented; nothing reads it; safe. Delete in future story (YAGNI).

### O-3 (OBS) — Allow re-derives role via Authorize

`Allow` re-derives role via `Authorize` — single lock acquisition, race-free, `Role` is value copy.

## Fix Commits This Pass

- None (M-1 and M-2 are non-blocking; test-writer additions dispatched post-convergence)

## Spec Edits This Pass

- None

## Deferred

- O-1 — VP-013/VP-035 skeleton stale API + error-code citation: deferred to architect/formal-verifier, Phase-6 hardening; tracked as drift item S-3.03-O1-VPSKEL

## Novelty Assessment

Novelty: NONE. Pass 5 following four prior clean passes (pass-2 streak-1, pass-3 streak-2, pass-4 streak-3, pass-5 streak-4). Adversary mounted attack-by-attack across all 8 surfaces; none produced a CRITICAL or HIGH. M-1 and M-2 are coverage gaps with test-writer resolution dispatched; they do not reset the convergence streak. Step 4.5 convergence criterion (3 clean passes) EXCEEDED at 4 clean passes. Reviewed tip: 9992276.
