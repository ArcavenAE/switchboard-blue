---
title: "Wave 3 Integration Gate — Convergence RESTART Pass 3 (fresh context, post-PR#15, spec-conformance-first)"
tree: "develop @ 10dd880"
verdict: "NOT_CONVERGED"
critical: 0
high: 2
medium: 3
low: 0
observations: 2
pass_label: r3
run: restart
produced_at: 2026-06-27
adjudication_required: true
---

# Wave 3 Integration Gate — Convergence RESTART Pass 3

**Tree:** develop @ 10dd880
**Verdict:** NOT_CONVERGED — 0C / 2H / 3M / 2 OBSERVATION

Note: both HIGHs are adjudication-dependent — may be legitimate scope deferrals.
Passes r1+r2 rated F-1 as OBSERVATION/in-scope-deferred.

## BLOCKING HIGH

**F-1 (HIGH)**
**Location:** cmd/switchboard/main.go:12-28
Binary is version-printing stub; never references Router/RouteFrame/AccessNode/
SessionConnector/SessionAuth/ControlMode/PTYProxy (grep: zero matches in cmd/).
All 5 Wave 3 stories wired only at package-API/test level.

Consequence: production Logger for RouteFrame E-ADM-016 never instantiated →
E-ADM-016 silently discarded (nopLogger) in any real build; AccessNode Sweep eviction
timer (delegated to "a timer in cmd/switchboard", fanout.go:234) never created.
"Wave 3 as working whole" not demonstrable from an entrypoint.

Fix: wire subsystems in cmd/switchboard OR explicitly scope integration to a later
wave in STORY-INDEX.

**ADJUDICATION NEEDED:** passes r1+r2 rated this OBSERVATION/in-scope-deferred
(ARCH-08 places cmd/switchboard at position 18 target/planned; no Wave 3 story
scopes cmd wiring).
**Owner:** architect (scope decision) + implementer (wiring)

**F-2 (HIGH)**
**Location:** routing.go (absent)
BC-2.05.008 EC-006 + BC-2.05.005 PC-3 — failure-rate counter/alert ("≥5 HMAC
failures in 60s from same src → admission alert") does not exist anywhere (grep
RecordFailure|failureCount|HMACFailure|alert|counter finds nothing relevant).
RouteFrame logs each failure but never increments per-src_addr counter; EC-006
unsatisfiable.

EC-006 NOT marked [DEFERRED] in BC-2.05.008 (unlike BC-2.04.003 PC-4/PC-5 which
carry explicit [DEFERRED]). Forged-frame flood logged line-by-line but raises no alert.

Fix: implement per-source failure counter+alert OR product-owner adds explicit
[DEFERRED] annotation + target story.
**ADJUDICATION NEEDED:** product-owner must decide ownership/deferral.
**Owner:** product-owner (adjudicate) → implementer

## MEDIUM

**F-3**
routing.go:177-180 PATH-A message format divergence from canonical E-ADM-016
(same as r2 M-1; product-owner: pick canonical form).

**F-4**
routing_log_test.go:340-389 PATH-A logging test self-documents it is GUESSING
PC-4 requires a log ("if spec-steward rules PC-4 does not require a log, revise
to assert NO log"); BC-2.05.008 PC-4 does not mandate the log; unadjudicated
test-writer assumption hardened into code.
Fix: product-owner amend PC-4 to state explicitly.

**F-5 (LOW)**
ErrSessionMismatch text lacks "(E-SES-006)" code token unlike ErrSessionAuthDenied/
ErrUpstreamReadOnly which embed (E-ADM-006)/(E-ADM-007); grep/classifier keyed on
E-SES-006 finds nothing.
**Owner:** implementer (add token)

## OBSERVATIONS

**F-6**
Test_BC_2_05_008_no_log_on_hmac_success asserts only `!HasAll("E-ADM-016")`, not
`err==nil` → weak negative. Test-writer: strengthen.

**F-7**
BC-2.04.001/002 spec says tmux `-CC` (double-C) but code launches `-C` (single-C,
correct for pipe use) → BC text↔code flag drift. Product-owner: reconcile BC text.

## Clean Seams

Failover (SessionConnector IS the KeystrokeSink, swap re-points atomically under
sc.mu; same shared Publisher; single-use guards); fail-closed enforcement all paths;
HMAC mandatory-path + lock safety (Log lock-free after RUnlock, no leak); VP-012
layering statically enforced.

**Novelty:** F-1/F-2 substantive whole-system seam gaps — per-story review
structurally cannot see these.
