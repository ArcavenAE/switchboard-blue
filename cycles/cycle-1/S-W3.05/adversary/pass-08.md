---
artifact_id: adv-S-W3.05-pass-08
review_target: S-W3.05-hmac-failure-counter-e-adm-017-alert
producer: adversary
pass: 8
lens: concurrency + memory/resource-bounds
fresh_context: true
branch: feat/S-W3.05-hmac-failure-counter
impl_commit: b945aab
test_commit: 5c3d7ea
findings_count: 2
critical_count: 0
high_count: 0
medium_count: 0
low_count: 2
verdict: CONVERGED
date: 2026-06-27
---

# Adversary Pass 08 — Concurrency + Memory/Resource-Bounds Lens

## Verdict: CONVERGED

Zero CRITICAL. Zero HIGH. Two LOW observations (non-blocking).

## Scope

Independent re-derivation of stuck-state and memory-bound proofs.
Lock discipline audit. Goroutine-safety of all exported methods.

## Verification Summary

**Stuck-state proof (re-derived independently):**
Claim: no source can reach a permanent-fired state where it fires once and
never re-arms. Proof sketch: after fire, re-arm gate is `len(w.timestamps)==0`.
The append-skip bound means no new timestamps enter beyond the cap. The
eviction loop removes timestamps older than window from the front. Given
sufficient time (> one full window), all timestamps drain. Therefore
`len(w.timestamps)` reaches 0. Re-arm follows. QED — no permanent-fired
state is reachable.

**Fire-then-silent memory bound proof (re-derived independently):**
Claim: the per-source slice is bounded in memory. Proof: append-skip
enforces `len(w.timestamps) <= maxPerSource` (a constant). The outer map
is bounded by `maxTrackedSources=65536` via LRU eviction. Total memory
for timestamps: `65536 * maxPerSource * sizeof(time.Time)` ≈ bounded.
The property-e clause of VP-059 v1.1 (no unbounded growth) holds.

**Lock discipline:**
All exported methods acquire the per-FailureCounter mutex before
reading/writing shared state. `logger.Log(...)` is called after `Unlock()`
— no lock held during I/O. `Timestamps()` and `SourceCount()` return
deep copies (slices/int values), not interior pointers. go.md rule 12
satisfied. No goroutine can hold a reference into the counter's internal
state after a method returns.

**No additional goroutines:**
The FailureCounter spawns no background goroutines. All eviction is
eager (inline at RecordHMACFailure call time). No timer goroutines that
could outlive the counter or cause a goroutine leak.

## Findings

### O-1 [LOW] error-taxonomy.md:53 prose annotation previously stale — NOW RESOLVED

**Location:** .factory/specs/prd-supplements/error-taxonomy.md, line 53
**Observation:** Prior pass noted that the re-fire annotation described
the old "always re-arm" model rather than the drain-only re-arm introduced
in BC-2.05.005 v1.6. This has been resolved: error-taxonomy.md has been
updated to v2.0 with the correct drain-only annotation. The message-format
string itself is unchanged. Non-blocking; noting resolution.
**Action:** None. Closed.

### O-2 [LOW] Stale TrackedSourceCount() name in comments only

**Location:** failure_counter.go, internal comment block
**Observation:** One comment references the old method name
`TrackedSourceCount()` (pre-refactor). The exported method is correctly
named `SourceCount()`. Comment is cosmetic, does not affect behavior or
test coverage.
**Action:** Deferred — cosmetic comment cleanup, post-wave or PR
description note.

## Conclusion

Concurrency model is sound. Memory bounds are provably finite under
configured constants. Lock discipline satisfies go.md rule 12. No
goroutine leaks. Two LOW observations are cosmetic or already-resolved
documentation items. Pass 08 is CLEAN for streak counting purposes.
