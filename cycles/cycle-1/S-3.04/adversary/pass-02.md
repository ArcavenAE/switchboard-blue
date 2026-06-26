---
artifact_id: adv-S-3.04-pass-02
review_target: S-3.04-hmac-routeframe-wireup
producer: adversary
pass: 2
fresh_context: true
branch: feature/S-3.04-hmac-routeframe-wireup
base: develop @ d8d7ae6
tip: 15353b1
findings_count: 2
findings_by_severity: {critical: 0, high: 0, medium: 1, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-25
note: Returned inline by adversary because tool profile is read-only; persisted by orchestrator via state-manager.
---

# Adversarial Review — Pass 2 — S-3.04

## Critical Findings
None.

## High Findings
None.

## Medium Findings

### M-1 — ADR-009 v1.6 step 3 not honored; routing.go:116-117 comment claims a copy-before-unlock that the code does not perform

**Files:**
- `internal/routing/routing.go:122-138`
- ADR-009 v1.6 in `.factory/specs/architecture/ARCH-04-admission-security.md:283-296`

The code captures `entry *ForwardingEntry` under RLock at line 122, calls `r.mu.RUnlock()` at line 128, then dereferences `entry.FrameAuthKey` at line 136 — AFTER the lock is released.

ADR-009 v1.6 step 3 prescribes "Copy FrameAuthKey into a local variable" BEFORE step 4 "Release RLock". The v1.6 "Rejected alternatives" section names the exact verify-without-copy pattern and rejects it: "defensive copying eliminates the concern entirely."

The comment at lines 116-117 reads: "FrameAuthKey is a [32]byte value type and is copied out before the lock is released; HMAC verification runs lock-free" — false against the code beneath.

**Behavioral impact:** Benign today (LWW immutability — RegisterForwardingEntry creates new *ForwardingEntry, never in-place mutation). But a future in-place mutation of FrameAuthKey would introduce a data race that ADR-009 v1.6 was specifically designed to prevent.

Confidence: HIGH. Severity: MEDIUM (spec drift + false comment).

**Fix:** Capture `authKey := entry.FrameAuthKey` before `r.mu.RUnlock()`; pass local to `verifyFrameHMAC`. Update comment to honestly describe the copy-before-unlock.

## Low Findings

### L-1 — FuzzRouteFrame_NonAdmittedNeverForwarded fuzz harness mis-anchors VP-008 post-S-3.04 (pending intent verification)

**File:** `internal/routing/routing_test.go:292-351`

The fuzz target sets up an unadmitted source, does NOT register a forwarding entry, does NOT compute a valid HMAC tag. Post-S-3.04 every iteration takes the early return at routing.go:130-132 (no forwarding entry → ErrHMACVerificationFailed). Admission check never reached.

The header docstring still claims VP-008 + BC-2.05.002 inv 1 trace — no longer exercised.

An intentional regression in admission.IsAdmitted would NOT be detected by this fuzz target.

Confidence: MEDIUM (pending intent). Severity: LOW.

**Fix options:**
- (a) Register forwarding entry for unadmittedAddr + compute valid HMAC tag → admission check reached, VP-008 genuinely exercised
- (b) Re-anchor to VP-058 property 4 (auth key unavailable → unverifiable → dropped)

## Observations

- All standard axes (A through K, M) clean except as noted.
- Sentinel godoc at routing.go:28-39 correctly cites E-ADM-016, BC-2.05.008 PCs 2/4, ADR-009.
- Pre-existing tests TestRouteFrame_DropsUnadmitted, TestRouteFrame_AdmittedSetCheckPrecedesForwarding correctly updated with forwarding entries + valid HMAC tags.
- All AC-001..005 + EC-001..005 tests present and traced.
- VP-058 proof harness ported from spec skeleton with correct API.
- //nolint:unused annotation fully removed.

## Novelty Assessment

Novelty: MEDIUM. Two substantive findings:
- M-1: ADR-009 v1.6 was specifically minted in the pass-1 fix burst to codify lock-free HMAC verify; the implementer fix at that time updated the comment but did NOT add the defensive copy. Partial-fix regression — the comment was changed without changing the code it described.
- L-1: pre-S-3.04 test became semantically stale; not caught in pre-impl spec-reviewer because the spec-reviewer wasn't analyzing pre-existing routing_test.go fuzz harnesses.

## Resolution decisions (from human review)

- M-1: implementer adds defensive copy line before RUnlock; updates comment to honestly describe new code (which now matches the comment claim).
- L-1: option (a) — restore VP-008 coverage by registering forwarding entry + computing valid HMAC tag in fuzz setup. test-writer dispatch.
