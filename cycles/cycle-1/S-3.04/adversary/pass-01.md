---
artifact_id: adv-S-3.04-pass-01
review_target: S-3.04-hmac-routeframe-wireup
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-3.04-hmac-routeframe-wireup
base: develop @ d8d7ae6
tip: 30bfa69
findings_count: 2
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 1 — S-3.04

## Critical Findings
None.

## High Findings
None.

## Medium Findings

### M-1 — ADR-009 v1.5 "single RLock acquisition" not honored; in-code comment contradicts implementation

**Files:** `internal/routing/routing.go:112-141`
**Anchor:** ADR-009 v1.5 in ARCH-04-admission-security.md:281-295

ADR-009 v1.5 prescribes ordering 1-6: acquire RLock → look up → extract FrameAuthKey → verify HMAC → proceed to admitted-set/routing → release RLock. The implementation acquires `r.mu.RLock()` at line 116, releases at line 122 BEFORE verifyFrameHMAC is called, then `r.admittedKeySet.IsAdmitted(...)` at line 135 acquires its own (separate) lock internally.

Aggravating: routing.go:115 comment claims "Hold the lock across steps 1–2 (single RLock acquisition per ADR-009)" — misleading because (a) lock is NOT held across step 2 (HMAC verify runs lock-free), (b) ADR specifies single RLock spans steps 1–6, not 1–2.

Sequential HMAC-before-admitted ordering is still satisfied (line 130 < line 135); no observable data race (FrameAuthKey is [32]byte value type, copied; LWW replacement preserves old pointer). Spec drift, not security bug.

**Resolution options:** (a) restructure to hold RLock across HMAC verify and admitted-set; (b) amend ADR-009 to permit lock-free HMAC verify; (c) fix comment only.

Confidence: HIGH. Severity: MEDIUM.

## Low Findings

### L-1 — Stale docstrings in routing test files

**Files:** `internal/routing/routing_internal_test.go:1-10` (package doc) + `internal/routing/routing_test.go` (~7 test docstrings)

Stale claims: "verifyFrameHMAC is //nolint:unused until S-3.04 wires it" and "RED against S-3.04 stub; becomes GREEN once wired". Both invalid post-wire-up: //nolint:unused removed; tests are GREEN.

Severity: LOW. Confidence: HIGH.

## Observations

- All 5 ACs trace to BC-2.05.008 postconditions; test docstrings match.
- E-ADM-016 used consistently in routing code; zero E-ADM-002 references in routing source/tests.
- ErrHMACVerificationFailed defined with godoc citing E-ADM-016, BC-2.05.008 PC-2/PC-4, ADR-009.
- //nolint:unused annotation removed from verifyFrameHMAC declaration.
- Pre-existing tests (TestRouteFrame_DropsUnadmitted, TestRouteFrame_AdmittedSetCheckPrecedesForwarding, ExampleRouter_dropsUnadmitted, ExampleRouter_svtnIsolation) modified MINIMALLY to construct valid HMAC tags; original invariants still exercised.
- VP-058 harness tests construct genuine admitted-but-invalid-HMAC scenarios; would fail loudly if ordering broken.
- GREEN-BY-DESIGN tests honestly labeled with BC-5.38.005 justifications.
- BC-2.05.008 PC-2 "E-ADM-016 logged at router before return" language unfulfilled (no logger in internal/routing); same pattern as BC-2.05.002 — project-wide deferred concern, NOT S-3.04-specific.

## Novelty Assessment

First pass — all findings genuinely new. Both real defects (M-1 is documented decision-record contradiction + in-code comment misrepresentation; L-1 is verifiable stale narrative).

## Resolution decisions (from human review)

- M-1: amend ADR-009 v1.5→v1.6 to permit lock-free HMAC verify (architect re-dispatch); fix routing.go:115 comment to honestly describe code (implementer re-dispatch).
- L-1: clean up stale docstrings in routing_internal_test.go + routing_test.go (implementer re-dispatch).
