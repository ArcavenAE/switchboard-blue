---
title: "Wave 3 Integration Gate — Convergence RESTART Pass 2 (fresh context, post-PR#15, concurrency-first)"
tree: "develop @ 10dd880"
verdict: "CONVERGED"
critical: 0
high: 0
medium: 4
low: 3
observations: 3
pass_label: r2
run: restart
produced_at: 2026-06-27
---

# Wave 3 Integration Gate — Convergence RESTART Pass 2

**Tree:** develop @ 10dd880
**Verdict:** CONVERGED — 0C / 0H / 4M / 3L / 3 OBSERVATION

## Findings

### MEDIUM

**M-1**
**Location:** routing.go:177-180
PATH-A E-ADM-016 message diverges from single canonical taxonomy string
(error-taxonomy.md:52 "tag mismatch"); PATH-A says "auth key unavailable"; unpinned by
test; operator log-parser keyed on "tag mismatch" misses PATH-A drops.
Fix: add PATH-A canonical variant to taxonomy OR align wording.
**Owner:** product-owner/implementer

**M-2**
**Location:** routing.go:160-201 + SVTNRoute 220-227
TOCTOU: RouteFrame copies authKey + RUnlocks (line 172), verifies HMAC, then SVTNRoute
takes a FRESH RLock and re-looks-up dest entry. Concurrent RegisterForwardingEntry (LWW)
between unlock and re-lock means verify-key and forward-table state are not one atomic
snapshot. Does NOT admit forged frames (verify happened against a real registered key) →
MEDIUM not HIGH.
Fix: document LWW-during-route window in BC-2.05.008/ADR-009, or pass resolved entry into
SVTNRoute under one lock.
**Owner:** architect/implementer

**M-3**
**Location:** fanout.go:298 `ConsoleSet.FramesDropped()` / upstream.go:328
`AccessNode.FramesDropped()`
BC-2.04.006 Inv-4 requires counter readable by monitoring without restart; exposed only via
Go API; cmd/switchboard wires nothing so no metrics/log/CLI surface. Deferred integration
obligation vs future wiring story.
**Owner:** architect/product-owner

**M-4**
**Location:** pty_fallback.go:735-749
`watchAndFallback` ctrl→ctrl swap: per-instance frames channels; SessionConnector exposes
no aggregated `Frames()`; consumer holding `oldCtrl.Frames()` sees it close on swap with
no SessionConnector API to reach newCtrl frames → frames on newCtrl unreachable. Latent
wiring gap (no crash).
Fix: `SessionConnector.Frames()` surviving swaps, or document re-subscribe on InPTYMode
change.
**Owner:** architect/implementer

### LOW

**L-1**
SVTNRoute forwarding no-op (`_=payload;_=entry`) per BC-2.04.006 PC-2 [PARTIAL] deferral
— no end-to-end frame traverses routing→session this wave. Product-owner: ensure deferred
router-multicast story reopens VP coverage.

**L-2**
`controlModeFailureLogMsg` string-match fallback + `-CC` vs `-C` literal mismatch (prod
uses `-C`) → fallback message may be omitted.
**Owner:** implementer

**L-3**
`unescapeTmuxOutput` octal boundary off-by-one borderline.
**Owner:** test-writer (add boundary test)

### OBSERVATIONS

**Obs-1** [process-gap]
cmd/switchboard wires none of 5 subsystems (consistent with deferred wave plan; integration
verified via tests not daemon).

**Obs-2**
E-SES-005 correctly absent.

**Obs-3**
Tests-as-spec quality high & mutation-resistant (auth STEP-1/2 fail on removal+inversion;
RouterHasNoTier2State structural; no-success-logging pins; Count==1 catches double-log).

## Tried and Could Not Break

Slow-logger lock stall (Log after RUnlock); HMAC ordering/fail-open; key/payload leak;
close-during-send (Deliver RLock across loop); SendKeystroke TOCTOU (sinkMu first);
go.md rule-12 (value copies); constructor defaults.

**Novelty:** MEDIUM
