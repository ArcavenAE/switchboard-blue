---
artifact_id: adv-S-3.02-pass-05
review_target: S-3.02-session-attach-detach-fanout
producer: adversary
pass: 5
fresh_context: true
branch: feature/S-3.02-session-attach-detach-fanout
base: develop @ 1bde664
tip: 11148b7
findings_count: 6
findings_by_severity: {critical: 0, high: 1, medium: 1, low: 2, observation: 2}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Fresh-context pass-5. Surfaced F-H-1 test-masks-defect on AC-007 serialization (sinkMu untested), F-M-1 AC-008 spec drift (time.Time vs time.Duration signature). Streak remains 0.
---

# Adversarial Review — Pass 5 — S-3.02

## High Findings

### F-H-1 (HIGH) — AC-007 serialization test is tautological; removing sinkMu leaves it green
Files: internal/session/session_test.go:471-528 (TestSession_ConcurrentKeystrokes_Serialized) and recordingSink at :182-199; production lock internal/session/upstream.go:272-273 (a.sinkMu.Lock()).
BC-2.04.006 Inv-3 mandates serialization enforced at the sinkMu boundary wrapping the sink write; tests must target post-mutex tmux delivery order. The test's recordingSink.SendInput takes its own r.mu and copies each payload before append. Each goroutine passes a distinct, never-mutated per-console slice (payloads[idx], :488), so torn-write assertions (:519-527) and total-count (:510-514) hold whether or not AccessNode.sinkMu exists. Deleting a.sinkMu.Lock()/Unlock() from SendKeystroke would not fail this test, and -race stays clean (only recordingSink + NoOpSink used; both internally synchronized or stateless). Serialization invariant is unverified. The in-code comment at session_test.go:466-470 claims the rewrite fixed this; it did not — a self-locking sink cannot observe absence of an outer serialization mutex.
Fix direction: use a sink that detects concurrency at the dispatch boundary — e.g. SendInput increments an unsynchronized in-flight counter, asserts <=1, Gosched, decrements (so -race catches a missing sinkMu); or records overlapping entry/exit timestamps.

## Medium Findings

### F-M-1 (MEDIUM) — Spec drift: AC-008 specifies EvictStale(deadline time.Time) (absolute); implementation is EvictStale(deadline time.Duration) (relative)
Story AC-008: .factory/stories/S-3.02-session-attach-detach-fanout.md:77 — "ConsoleSet.EvictStale(deadline time.Time) takes an absolute deadline — the test passes time.Now().Add(-keepaliveTTL - 1)." Implementation: internal/session/fanout.go:234 — func (cs *ConsoleSet) EvictStale(deadline time.Duration) int with cutoff := cs.nowFn().Add(-deadline) at :235. Signature, parameter semantics, and prescribed test technique all diverge from the AC. The shipped clock-injection approach is arguably better, but the AC was not reconciled. Disposition: patch the AC to describe the time.Duration + injected-clock contract actually shipped.

## Low Findings

### F-L-1 (LOW) — Allow (authorizer) invoked before attachment/session-mismatch validation in SendKeystroke
internal/session/upstream.go:264-281. a.authorizer.Allow(...) at :264 runs before IsAttached/Session checks at :275-281. Harmless for NoOpAuthorizer, but when S-3.03 wires a real SessionAuth this exposes the authorizer to keystrokes for non-attached/mismatched consoles and inverts natural precedence. Undocumented — verify intent or reorder.

### F-L-2 (LOW) — WithClock silently discards the ConsoleSet built by NewAccessNode
internal/session/upstream.go:113-119 — WithClock does a.consoles = NewConsoleSet(ConsoleSetWithClock(fn)), replacing the set allocated at :160. Safe today but option-ordering is significant and undocumented; a future option touching a.consoles before WithClock would be silently clobbered. Prefer applying ConsoleSetWithClock to the existing set or document the ordering contract.

## Observations
- VP-056 (BC-2.04.004 proof method: integration) covered only by unit test TestSession_CrashDetach_EvictsFromFanOut (session_test.go:537). Integration-tier classification is a system/verification-tier concern — defer to wave-gate/phase-5.
- [process-gap] The pass-3 remediation comment (session_test.go:466-470) documents a fix that does not achieve mutation-detection (see F-H-1). Recurring anti-pattern: a self-synchronizing test double substituted for the production synchronization primitive under test, with an in-code coverage claim no mutation test validated. The adversary/TDD workflow has no mutation-kill gate. Codification follow-up warranted (mutation-test or "delete-the-lock" smoke check for any test claiming to verify a concurrency primitive).

Verdict: NOT_CONVERGED. Streak 0.
