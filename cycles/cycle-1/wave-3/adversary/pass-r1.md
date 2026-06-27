---
title: "Wave 3 Integration Gate — Convergence RESTART Pass 1 (fresh context, post-PR#15)"
tree: "develop @ 10dd880"
verdict: "CONVERGED"
critical: 0
high: 0
medium: 0
low: 2
observations: 3
pass_label: r1
run: restart
produced_at: 2026-06-27
---

# Wave 3 Integration Gate — Convergence RESTART Pass 1

**Tree:** develop @ 10dd880
**Verdict:** CONVERGED — 0C / 0H / 0M / 2L / 3 OBSERVATION

## Findings

### OBS-1 (integration gap)
**Location:** cmd/switchboard/main.go:12-34
None of the 5 subsystems instantiated/wired; integration exists only at package-API + test
level. Verified IN-SCOPE-DEFERRED: no Wave 3 story scopes cmd wiring (checked all 5 story
specs). Track as Phase-≥4 wiring story.
**Owner:** architect

### OBS-2 (integration seam)
**Location:** tmux control.go:369 `Frames() <-chan halfchannel.ChannelFrame` vs session
upstream.go:304 `DeliverFrame(frame.OuterHeader)`
No production path bridges tmux output → session fan-out; type-incompatible; only tests
exercise each half. Bridge belongs in wiring story.
**Owner:** architect + implementer

### L-3 (LOW)
**Location:** routing.go:177-180
PATH-A logs "auth key unavailable for SVTN %x from src %x (E-ADM-016)" — omits canonical
"tag mismatch" substring; two distinct operator strings both carry E-ADM-016; test asserts
only shared prefix+code+ids so variance unpinned. PC-4 doesn't mandate a log. Adjudicate:
document PATH-A variant in taxonomy or align wording.
**Owner:** product-owner

### L-4 (LOW)
**Location:** auth.go:30
`RoleFull=iota=0` zero-value most-permissive; safe today (entries only via RegisterKey;
missing key → ErrSessionAuthDenied fail-closed before role read); documented. Latent
footgun. Optional: RoleUnset before RoleFull.
**Owner:** implementer

### OBS-3
**Location:** pty_fallback.go:560-566
`controlModeFailureLogMsg` string-match fallback brittle/dead in prod path (sentinels
used). Consider removing.
**Owner:** implementer

## Verified Clean (tried hardest to break)

New E-ADM-016 logging — logs on both failure paths, only on failure (success-suppression
test is mutation-killer), no key/payload leak (only SVTNID+SrcAddr hex), Log called AFTER
RUnlock (line 172) so no lock stall/deadlock; routing_log_test.go non-tautological
(HasAll + Count==1). VP-012 layering holds (routing imports only frame/hmac/admission;
TestSessionAuth_RouterHasNoTier2State greps routing source). Fail-closed auth verified on
all paths (Attach gates Allow; SendKeystroke re-checks under sinkMu; vestigial channel not
drained). E-SES-005 absent. Failover: fresh ControlMode per reconnect, one-way PTY, no
double-attach.

**Novelty:** LOW-MEDIUM
