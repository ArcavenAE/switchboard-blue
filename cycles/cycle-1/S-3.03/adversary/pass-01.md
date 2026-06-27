---
artifact_id: adv-S-3.03-pass-01
review_target: S-3.03-tier2-session-authorization
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-3.03-tier2-session-authorization
findings_count: 9
findings_by_severity: {critical: 1, high: 2, medium: 3, low: 1, observations: 2}
verdict: NOT_CONVERGED
streak_after_pass: 0
streak_reset_reason: critical and high findings present
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 1 — S-3.03

## Critical Findings

### C-1 — AccessNode.Attach never consults the Authorizer

**File:** `internal/session/upstream.go ~lines 203-219`

Tier-2 authorization was enforced ONLY on the `SendKeystroke` path. An unauthorized or read-only console could call `Attach` and receive the entire downstream output stream without restriction — violating BC-2.05.003 PC-2 / EC-001 ("channel is not established" for an unauthorized console).

No test drove `Attach` with an unregistered key, masking the gap entirely.

**Resolution:** added attach-time `Authorize` gate (implementer commit a3aa1bd) + failing-first tests `TestAccessNode_Attach_UnauthorizedKey_Rejected` / `_EmptyAuthList_Rejected` / `_AuthorizedKey_Succeeds` (test-writer commit 79c28f0).

Confidence: HIGH. Severity: CRITICAL.

## High Findings

### H-1 — E-ADM-006 / E-ADM-007 error messages diverged from canonical format

**Files:** `internal/session/upstream.go` (error message construction); `.factory/specs/prd-supplements/error-taxonomy.md`

Error messages for E-ADM-006 and E-ADM-007 diverged from the canonical error-taxonomy format and omitted the mandated interpolated fields (`<key_fingerprint>`, `<session_name>`, `<node_addr>`).

**Resolution:** messages now interpolate key and session and wrap the sentinels with `%w` (implementer commit 1cd39b3); `node_addr` is best-effort where not available.

### H-2 — Error-format test was a weak substring check

**File:** `internal/session/` (authorization tests)

The only error-format test used a weak `strings.Contains("E-ADM-006")` substring check — incapable of detecting H-1's interpolation gaps.

**Resolution:** `TestSessionAuth_ErrorMessages_MatchTaxonomy` added, asserting that interpolated fields are present in error output (test-writer commit 79c28f0).

## Medium Findings

### M-1 — BC-2.04.005 PC-5 (read-only subscriber presence/attached) unimplemented

**Files:** `.factory/specs/behavioral-contracts/ss-04/BC-2.04.005.md` (PC-5)

BC-2.04.005 PC-5 (read-only subscriber presence/attached state) is unimplemented in S-3.03.

**Adjudication (architect):** out of scope for S-3.03. Mechanism owned by BC-2.03.003 Inv-3 / `internal/discovery` (scope_phase: PE). Spec cross-reference annotation added to BC-2.04.005 PC-5. Tracked as drift, not a blocker.

### M-2 — Apparent contradiction: BC-2.05.003 PC-4 vs BC-2.04.005 EC-003/EC-005

**Files:** `.factory/specs/behavioral-contracts/ss-04/BC-2.04.005.md`; `.factory/specs/behavioral-contracts/ss-05/BC-2.05.003.md`

Apparent contradiction between BC-2.05.003 PC-4 (auth enforced per-session, per-key) and BC-2.04.005 EC-003/EC-005 (read-only scope described as per-SVTN).

**Adjudication (architect):** not contradictory. The per-(session, key) data model is sufficient; "SVTN-wide" describes a provisioning pattern, not the enforcement granularity. A clarifying sentence was added to EC-003 (BC-2.04.005 v1.2).

### M-3 — AC-006 empty-tick test proved "not rejected" but not "forwarded"

**File:** `internal/session/` (read-only console tests)

AC-006's empty-tick test asserted the tick was not rejected but did not assert it was forwarded to the capture sink.

**Resolution:** `TestReadOnlyConsole_EmptyTickForwarded_WithCaptureSink` added (test-writer commit 79c28f0). Empty ticks are in fact FORWARDED — non-finding; test retained as regression guard.

## Low Findings

### L-1 — Minor interpolation gap (folded into H-1 fix)

Minor error-message interpolation gap, subsumed by H-1 remediation (implementer commit 1cd39b3). No independent action required.

## Observations

### O-1 [process-gap] — E-ADM-007 / E-SES-005 dual coverage of read-only upstream rejection

Both E-ADM-007 and E-SES-005 described read-only-upstream rejection, creating ambiguity about the canonical code.

**Adjudication (architect):** E-ADM-007 is canonical. E-SES-005 RETIRED in `error-taxonomy.md` v1.5.

### O-2 — No issues found in clean areas

Concurrency discipline was clean: `RWMutex` usage correct, lock ordering `sinkMu→sa.mu` consistent, no go.md rule-12 internal-pointer leaks observed. VP-012 router-has-no-Tier-2-state code audit passed. `{frame, admission}`-only import boundary (ARCH-08 §6.5) verified.

## Clean Areas

Confirmed by adversary review:

- Concurrency discipline (RWMutex, lock ordering `sinkMu→sa.mu`, no go.md rule-12 internal-pointer leak)
- VP-012: router has no Tier-2 state — code audit passed
- `{frame, admission}`-only import boundary (ARCH-08 §6.5)

## Fix Commits

- Test-writer: 79c28f0
- Implementer: 1cd39b3, a3aa1bd
- Worktree: `feature/S-3.03-tier2-session-authorization`

## Spec Edits

- `error-taxonomy.md` v1.5 — E-SES-005 retired (see O-1)
- `BC-2.04.005.md` v1.2 — EC-003 clarifying sentence added; PC-5 cross-reference annotation added (see M-1, M-2)

## Novelty Assessment

Novelty: HIGH. Pass 1, no prior reviews. C-1 is a substantive authorization-bypass gap — Attach path was entirely unguarded. H-1/H-2 represent error-taxonomy conformance drift. M-1/M-2 required architect adjudication on scope and spec interpretation. Convergence streak reset to 0.
