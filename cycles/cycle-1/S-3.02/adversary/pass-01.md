---
artifact_id: adv-S-3.02-pass-01
review_target: S-3.02-session-attach-detach-fanout
producer: adversary
pass: 1
fresh_context: true
branch: feature/S-3.02-session-attach-detach-fanout
base: develop @ 56ec9c7
tip: c379c4f
findings_count: 7
findings_by_severity: {critical: 1, high: 3, medium: 2, low: 1, nitpick: 0}
verdict: NOT_CONVERGED
timestamp: 2026-06-26
note: Returned inline by adversary; persisted via state-manager.
---

# Adversarial Review — Pass 1 — S-3.02

## Critical Findings

### F-01 — Send-on-closed-channel panic race in ConsoleSet.Deliver

**File:** `internal/session/fanout.go:126-142`

Snapshot pattern unsafe vs concurrent Remove. Deliver copies channel refs under RLock, releases lock, then sends via `select { case ch <- hdr: default: }`. Between RUnlock and send, concurrent Remove closes the channel. select+default does NOT catch closed channel — sending on closed channel panics unconditionally.

Concurrent scenario:
1. Goroutine A: Deliver snapshots channels and releases lock
2. Goroutine B: Remove acquires WLock, closes channel, deletes entry
3. Goroutine A resumes loop with now-closed channel → runtime error: send on closed channel

Comment at line 138-139 conflates "channel full" (handled by default) with "channel closed" (not handled). Production paths calling Deliver from different goroutine than Remove will crash. No unit test covers concurrent Deliver/Remove. go test -race doesn't catch send-on-closed.

Confidence: HIGH. Severity: CRITICAL.

## High Findings

### F-02 — SendKeystroke is a no-op stub disguised as production code

**File:** `internal/session/upstream.go:114-143`

Function: (1) calls Authorizer.Allow ✓, (2) verifies console attached via Snapshot ✓, (3) locks upstreamMu, then DISCARDS payload: `_ = payload`.

AC-007 traces to BC-2.04.006 Invariant 3 ("All full-access console keystrokes are serialized by the access node before forwarding to tmux"). Implementation never forwards anywhere — not to upstream channel, not to tmux, not to any sink. Serialization mutex protects nothing observable. TestSession_ConcurrentKeystrokes_Serialized only asserts nil error returns — cannot detect that serialization is vacuous.

Compounding: upstream channel returned by Attach has NO READER. Two parallel upstream paths exist with no producer-consumer wiring.

Test-masks-defect pattern: test was written to verify a contract the production code does not implement. AC-007's coverage is hollow.

### F-03 — AC-008 test does not exercise spec'd crash-detect path

**Files:** `internal/session/session_test.go:394-428`, `fanout.go:144-160`

AC-008 traces to BC-2.04.004 EC-002 ("Access node detects channel closure on next delivery attempt; evicts console"). Test "simulates" crash by calling `an.Detach("crash-victim")` — but Detach is GRACEFUL path, same as AC-004. By the time DeliverFrame runs, victim is absent from consoles map; snapshot in Deliver never sees closed channel.

Implementer comment openly acknowledges: "the implementer must handle crash detection inside DeliverFrame via recover on send-to-closed. This test exercises the detectable outcome." But implementation does NOT use recover; Evict() is no-op for crash detection.

Additionally: downstream channel returned as `<-chan` (receive-only). Caller (console) CANNOT close it. Real "console crash closes its channel" is impossible by construction.

Story task 9 ("Implement crash detection: evict consoles whose channels are closed (AC-008)") is unfulfilled.

### F-04 — Local error-code claims E-SES-002, E-SES-003 conflict with other specs

**Files:** `internal/session/fanout.go:15-21`; `.factory/specs/prd-supplements/error-taxonomy.md:123-127`; `.factory/stories/S-7.03-console-remote-control.md:70`; `.factory/holdout-scenarios/wave-scenarios/wave-3.md:33,58,68`

fanout.go annotates sentinels with codes that don't exist in error-taxonomy.md:
- ErrConsoleAlreadyAttached → E-SES-002 (unallocated; S-7.03 claims E-SES-002 = "not attached" — opposite semantics)
- ErrConsoleNotFound → E-SES-003 (unallocated; wave-3 holdout claims E-SES-003 = "read-only upstream rejected" — different semantics)

Canonical taxonomy only declares E-SES-001. Implementation unilaterally claims codes whose semantics conflict with downstream specs.

## Medium Findings

### F-05 — Evict() is misnamed/misdesigned

**File:** `internal/session/fanout.go:113-117, 144-160`

Remove appends to evictQueue AFTER deleting console from map and closing channel. Evict just drains the queue and returns length. There is NO actual eviction logic — Evict returns counter of Removes since last call.

Misleading API: godoc says "drains internal evict queue" but consoles were not evicted BY Evict — they were evicted by Remove. evictQueue creates unbounded slice (cleared only on Evict() calls). If DeliverFrame is rarely called, evictQueue grows without bound (memory leak vector).

### F-06 — AC-002 trace claim overstates test coverage

**File:** `internal/session/session_test.go:185-202`

AC-002 traces to BC-2.04.003 PC-3 ("upstream half-channel ... accepted by access node and forwarded to tmux session"). Test verifies upstream channel accepts a write without blocking. Does NOT verify the keystroke was received by access-node consumer or forwarded — because NO consumer exists.

Test comment honestly notes "Real tmux forwarding is out of scope for unit tests" — but the AC claims test coverage of PC-3 in full.

## Low Findings

### F-07 — Stale `//nolint:staticcheck` rationale

**Files:** `internal/session/session_test.go:270, 403`, `internal/session/fanout_test.go:164, 168`

Lint suppression comments still reference Red-Gate scaffold state ("stub panics before assignment"). Implementation no longer a stub. Justification is stale.

## Observations

- AC-005 (read-only observer) exercised but ConsoleSet API has no concept of full-access vs read-only role flag. Distinction deferred to S-3.03.
- Backpressure documented (downstream=64, upstream=16). Drop semantics silently lose frames per console; no log or metric.
- ARCH-08 §6.5 imports verified clean (admission + frame only).
- UTC discipline clean.
- No init(), no panic(), no log.Fatal/os.Exit in package code.

## Resolution decisions (from human review)

- F-01: hold RLock for entire Deliver loop; Remove marks-for-eviction-don't-close. Restructure: Remove only deletes from map; close deferred to Evict under exclusive WLock.
- F-02 + F-03: BOTH are S-3.02 scope. Implementer must (a) actually forward keystrokes via upstream channel to consumer in AccessNode (b) restructure API so console CAN signal close (CloseDownstream method, bidirectional chan, or ctx cancellation per console).
- F-04: PO mints E-SES-002 + E-SES-003 in error-taxonomy.md with S-3.02 semantics; patches S-7.03 to use E-SES-004 (read-only upstream rejected); patches wave-3 holdout to use E-SES-005.
- F-05: redesign Evict to perform actual crash-detection probe (probe each channel via recover on Deliver, or expose CloseSignal per-console).
- F-06: add consumer goroutine in AccessNode that reads upstream channel; test asserts consumer receives the keystroke.
- F-07: drop stale nolint or update rationale to reflect post-stub state.

## Novelty Assessment

Novelty: HIGH. Pass 1, no prior reviews. F-01..F-04 are substantive contract gaps, not nitpicks. Implementation appears to be stubbed scaffold satisfying test signatures without implementing production contracts.
