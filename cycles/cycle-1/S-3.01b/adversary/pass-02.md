---
artifact_id: adv-S-3.01b-pass-02
review_target: S-3.01b-pty-proxy-fallback
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-3.01b-pty-proxy-fallback
base: develop @ 43208ab
tip: 161f2e3
findings_count: 2
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 2 — S-3.01b

## Medium Findings

### M-001 — E-SYS-001 three-way text drift

**Files:** pty_fallback.go:40 (sentinel), :34 (docstring), .factory/stories/S-3.01b:55 (AC-003), .factory/specs/behavioral-contracts/ss-04/BC-2.04.002.md:76 (EC-004)

Sentinel: `errors.New("PTY device unavailable: cannot start access node")`
Docstring: "Install 'openpty' support or check kernel PTY configuration"
BC/story: "Install 'openpty' or check device permissions."

Pass-1 M-002 ("ErrPTYDeviceUnavailable text includes...") landed only on story/BC. Implementation has THREE-WAY drift. Test (TestPTYProxy_NoPTY_ReturnsErrSysOne) asserts only errors.Is — misses text drift.

BC invariant 3 ("fallback failure never silent") + EC-004 explicit text → operator must receive "Install 'openpty'..." guidance. Currently no logger.Log emits it when PTY allocation fails.

### L-001 — ARCH-08 §6.6 vs §6.5 inconsistency within pty_fallback.go

**File:** pty_fallback.go:3 says "§6.5"; pty_fallback.go:16 says "§6.6"

Partial-fix regression on pass-1 M-001 — header anchor updated; inline import-rule comment missed.

## Resolution decisions (mechanical)

- M-001: implementer adds logger.Log emit of full BC-mandated text in PTY allocation-failure path; test asserts substring "Install 'openpty'" via grep-discoverable assertion. Sentinel stays terse for ST1005 (Go error idiom).
- L-001: implementer updates line 16 to §6.5.

## Novelty Assessment

Novelty: LOW. Both partial-fix regressions from pass-1. Pass-1 C-001/C-002/H-001..H-005 verified holding. ST1005 vs BC-mandated multi-sentence text creates tension; resolution = sentinel terse + logger emits full text.
