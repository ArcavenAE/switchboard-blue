---
artifact_id: S-W3.05-hmac-failure-counter
document_type: story
level: ops
story_id: S-W3.05
title: "per-source HMAC failure counter and admission alert (BC-2.05.005 PC-3)"
status: ready
producer: story-writer
timestamp: 2026-06-27T00:00:00Z
phase: 2
epic: E-2
wave: 3
priority: P0
scope_phase: E
estimated_points: 5
bc_traces:
  - BC-2.05.005
  - BC-2.05.008
vp_traces: [VP-059, VP-058]
subsystems: [admission-security]
architecture_modules: [internal/admission, internal/routing]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-2.01, S-3.04]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.005.md'
  - '.factory/specs/behavioral-contracts/ss-05/BC-2.05.008.md'
  - '.factory/specs/verification-properties/VP-059.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'
acceptance_criteria_count: 10
version: "1.0"
---

# S-W3.05: Per-Source HMAC Failure Counter and Admission Alert

> **Execute:** `/vsdd-factory:deliver-story S-W3.05`

> **Classification:** Wave 3 FIX-NOW gate blocker (F-2). Implements BC-2.05.005 PC-3
> (per-source sliding-window alert). Error codes: E-ADM-016 (per-failure, owned by
> BC-2.05.008 / S-3.04), E-ADM-017 (aggregate alert threshold, new in this story).

## Narrative

- **As a** router operator
- **I want** the router to track per-source HMAC failure rates and emit a structured
  E-ADM-017 alert when any source exceeds 5 failures within a 60-second sliding window
- **So that** active key-forgery or credential-leak attacks are surfaced immediately
  without relying on per-failure E-ADM-016 log scanning

## Behavioral Contracts

| BC | Title | Clause Covered |
|----|-------|---------------|
| BC-2.05.005 | HMAC Frame Authentication at First Router Boundary | PC-3 (FailureCounter API + sliding window); EC-005–EC-008 |
| BC-2.05.008 | RouteFrame Wire-Layer HMAC Enforcement | EC-006 (RecordHMACFailure call on every ErrHMACVerificationFailed path); PC-5 |

## Acceptance Criteria

### AC-001 (traces to BC-2.05.005 PC-3 postcondition — FailureCounter type contract)
`admission.FailureCounter` is defined in `internal/admission/failure_counter.go`.
Constructor: `NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger) *FailureCounter`.
Method: `RecordHMACFailure(srcAddr string)` — no `context.Context` argument (pure
in-memory; mutex-guarded). Internal state: `map[string][]time.Time` guarded by
`sync.Mutex`. No package-level global state; logger is injected via constructor.
- **Test:** `TestNewFailureCounter_ConstructorFields`

### AC-002 (traces to BC-2.05.005 PC-3 — sliding window trimming)
On each `RecordHMACFailure` call, timestamps older than `now - windowDuration` are
trimmed before appending the new entry. Trimming uses strictly-less-than comparison:
entries with `timestamp < now - windowDuration` are removed; entries at exactly
`now - windowDuration` are kept (boundary is inclusive). This implements a sliding
window, not a fixed bucket.
- **Test:** `TestFailureCounter_SlidingWindowTrimsStaleEntries`

### AC-003 (traces to BC-2.05.005 PC-3 — E-ADM-017 emission on threshold crossing)
When the post-trim count for a `srcAddr` reaches or exceeds `threshold`, the
`FailureCounter` emits a structured log event at ERROR level with code E-ADM-017.
Format: `"HMAC failure rate alert: ≥5 failures in 60s from src <src_addr>"`.
The event is emitted via the injected logger. No global state. No direct call to
`log.Printf` or any package-level logger.
- **Test:** `TestFailureCounter_EmitsEADM017AtThreshold`

### AC-004 (traces to BC-2.05.005 PC-3 — fire-once-per-threshold-crossing)
E-ADM-017 fires exactly once when the count crosses the threshold (on the Nth call
where N = threshold). Subsequent `RecordHMACFailure` calls in the same window (the
N+1th, N+2th, etc.) do NOT re-emit E-ADM-017. The counter MUST track whether the
threshold has already fired for the current window crossing; the fired state resets
only when the count drops back below threshold (all entries trimmed, window expired).
- **Test:** `TestFailureCounter_FiresOncePerCrossing`

### AC-005 (traces to BC-2.05.005 EC-005 — hysteresis: alert resets after window expires)
After a threshold crossing fires E-ADM-017, if all timestamps for that `srcAddr` age
out of the window (count drops to 0 after trimming), a subsequent batch of ≥N failures
fires E-ADM-017 again. Test scenario: 5 failures at T=0 → 1 alert; 61-second pause
(entries now older than windowDuration); 5 more failures at T=61s → 1 more alert.
Total: exactly 2 E-ADM-017 events.
- **Test:** `TestFailureCounter_HysteresisRefirersAfterWindowExpires`

### AC-006 (traces to BC-2.05.005 EC-006 — below-threshold: no alert)
Exactly `threshold - 1` failures within the window do NOT emit E-ADM-017. The
counter holds the entries and returns without alerting. On a subsequent `RecordHMACFailure`
call that brings the count to exactly `threshold`, E-ADM-017 fires once.
- **Test:** `TestFailureCounter_BelowThresholdNoAlert`

### AC-007 (traces to BC-2.05.005 EC-007 — multi-source isolation)
Two distinct `srcAddr` values ("addr-A" and "addr-B") each accumulate failure counts
independently. When both reach the threshold, two separate E-ADM-017 events are emitted,
one per source. Calls interleaved between sources do not cause cross-counter interference.
The `sync.Mutex` covers the full per-call trim+append+check sequence for each `srcAddr`.
- **Test:** `TestFailureCounter_MultiSourceIsolation`

### AC-008 (traces to BC-2.05.005 EC-008 — boundary: 5th failure exactly at windowDuration after 1st)
When the 5th failure arrives at timestamp `t1 + windowDuration` (exactly `windowDuration`
after the 1st failure at `t1`), the trim comparison is `timestamp < now - windowDuration`.
The 1st entry has `timestamp == now - windowDuration` (not strictly less); it is kept.
Post-trim count = 5; E-ADM-017 fires. Implementations that use `<=` (trim-at-boundary)
instead of `<` (keep-at-boundary) will produce count = 4 and fail this test.
- **Test:** `TestFailureCounter_BoundaryEntryIsKept`

### AC-009 (traces to BC-2.05.008 PC-5 + invariant 5 — RouteFrame calls RecordHMACFailure)
`RouteFrame` in `internal/routing` calls `router.failureCounter.RecordHMACFailure(hdr.SrcAddr)`
immediately before returning `ErrHMACVerificationFailed` on BOTH failure paths:
(a) tag mismatch (`verifyFrameHMAC` returns false), and (b) no forwarding-table entry
(auth key unavailable). `RecordHMACFailure` is NOT called on a successful HMAC
verification. The `failureCounter *admission.FailureCounter` field is added to the
`Router` struct and wired via constructor injection (`NewRouter` or `WithFailureCounter`
option).
- **Test:** `TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths`

### AC-010 (traces to BC-2.05.005 PC-3 — concurrency safety)
`RecordHMACFailure` is safe for concurrent calls from multiple goroutines. The
`sync.Mutex` is held for the complete trim+append+threshold-check sequence. No
internal slice is returned by reference to callers. If a `Timestamps(srcAddr string)
[]time.Time` inspector is needed (e.g., in tests), it returns a copy of the slice
(go.md rule 12: no internal pointer leaks from locked accessors). `go test -race` MUST
pass with concurrent callers.
- **Test:** `TestFailureCounter_ConcurrentCallsRaceSafe` (run with `-race`)

## Edge Cases

| ID | Source | Description | Expected Behavior |
|----|--------|-------------|-------------------|
| EC-001 | BC-2.05.005 EC-005 | 6 failures in 60s, then 61s pause, then 5 more | 2 E-ADM-017 events total (one per crossing); no extra events |
| EC-002 | BC-2.05.005 EC-006 | 4 failures in 60s, no 5th | No E-ADM-017; 4 timestamps held in map |
| EC-003 | BC-2.05.005 EC-007 | ≥5 from addr-A and ≥5 from addr-B, interleaved | 2 independent E-ADM-017 events; addr-A and addr-B slices never share entries |
| EC-004 | BC-2.05.005 EC-008 | 5th timestamp exactly at boundary (`t1 + windowDuration`) | Boundary entry kept; count = 5; E-ADM-017 fires |
| EC-005 | BC-2.05.008 EC-006 | 5 consecutive HMAC failures from same src_addr | `RouteFrame` calls `RecordHMACFailure` 5 times; on 5th call `FailureCounter` emits E-ADM-017 |
| EC-006 | BC-2.05.008 EC-003 | Frame from admitted node, forwarding-table entry purged | `ErrHMACVerificationFailed` returned (postcondition 4); `RecordHMACFailure` called (invariant 5 covers this path) |
| EC-007 | BC-2.05.008 EC-005 | Valid HMAC, node not in admitted set | `admission.ErrNotAdmitted` returned; `RecordHMACFailure` NOT called (HMAC passed) |

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|----------------|
| `FailureCounter` type + `NewFailureCounter` | internal/admission | boundary (mutex-guarded in-memory state; no I/O) |
| `RecordHMACFailure(srcAddr string)` | internal/admission | boundary |
| `failureCounter` field on `Router` struct | internal/routing | boundary |
| `RouteFrame` RecordHMACFailure call site | internal/routing | boundary |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| internal/admission | boundary | Mutable in-memory state (FailureCounter map) guarded by sync.Mutex; no network/file I/O; "pure-enough" per BC-2.05.005 PC-3 annotation |
| internal/routing | boundary | Existing classification unchanged; FailureCounter is injected as a dependency |

## Token Budget Estimate

| Context Source | Estimated Tokens |
|----------------|-----------------|
| This story spec | ~1,200 |
| BC-2.05.005.md | ~1,200 |
| BC-2.05.008.md | ~900 |
| VP-059.md | ~400 |
| ARCH-08 §6.5 (positions 4 + 5 import constraints) | ~400 |
| internal/admission (existing, ~3 files) | ~600 |
| internal/routing/routing.go (existing, ~170 lines) | ~500 |
| internal/routing/routing_test.go (existing) | ~400 |
| New test files (2 new) | ~1,200 |
| Tool outputs overhead | ~200 |
| **Total** | **~7,000** |
| Agent context window | 200K |
| **Budget usage** | **~3.5%** |

## Tasks

1. [ ] Read BC-2.05.005 (full, including PC-3 API contract + EC-005–EC-008), BC-2.05.008
       (full, including EC-006 + PC-5 + invariant 5), ARCH-08 §6.5 (positions 4 and 5)
2. [ ] Write failing tests for AC-001 through AC-010 in:
       - `internal/admission/failure_counter_test.go` (AC-001 through AC-008, AC-010)
       - `internal/routing/routing_test.go` extension (AC-009)
3. [ ] Verify Red Gate: all 10 tests fail before any implementation
4. [ ] Create `internal/admission/failure_counter.go`:
       - `type FailureCounter struct` with `mu sync.Mutex`, `counts map[string][]time.Time`,
         `threshold int`, `windowDuration time.Duration`, `logger Logger`, `fired map[string]bool`
       - `NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger) *FailureCounter`
       - `RecordHMACFailure(srcAddr string)` — trim (`timestamp < now - windowDuration`),
         append, check threshold, emit E-ADM-017 once per crossing, update `fired` state
       - No background goroutine; lazy eviction on every call
       - `Timestamps(srcAddr string) []time.Time` returning a copy (if needed for testing)
5. [ ] Confirm UTC timestamps: all `time.Now()` calls in `RecordHMACFailure` use `time.Now().UTC()`
6. [ ] Confirm no internal pointer returned: `Timestamps()` returns `append([]time.Time{}, ...)` copy
7. [ ] Modify `internal/routing/routing.go`:
       - Add `failureCounter *admission.FailureCounter` field to `Router` struct
       - Wire via `WithFailureCounter(fc *admission.FailureCounter)` option (or constructor param)
       - In `RouteFrame`: call `r.failureCounter.RecordHMACFailure(hdr.SrcAddr)` on BOTH
         `ErrHMACVerificationFailed` paths (tag mismatch + no forwarding entry),
         BEFORE the return statement; no call on success
8. [ ] Verify `internal/routing` imports unchanged: {frame, hmac, admission} — position 5 constraint holds
9. [ ] `just fmt && just lint` pass with zero warnings
10. [ ] `just test-race` passes (AC-010 concurrent safety)

## Previous Story Intelligence

| Decision | Rationale | Applies To |
|----------|-----------|-----------|
| `ErrHMACVerificationFailed` sentinel exists in `internal/routing` (S-3.04) | Minted and tested by S-3.04; remove any duplicate declaration | `RecordHMACFailure` call sites use the already-existing sentinel return points |
| Forwarding table has `FrameAuthKey` field (S-2.02) | LWW semantics; `ForwardingEntry.FrameAuthKey [hmac.KeySize]byte` | The "no forwarding entry" path (BC-2.05.008 PC-4) also calls `RecordHMACFailure` — confirmed by BC-2.05.008 invariant 5 |
| `AdmittedKeySet` uses `sync.RWMutex` (S-2.02) | Existing pattern for concurrent-safe boundary types in `internal/admission` | `FailureCounter` uses `sync.Mutex` (write-heavy; no read-only inspection in hot path) |
| No testify; table-driven tests; `t.Parallel()` (project-wide go.md rule) | Project standard | All tests in `failure_counter_test.go` must follow this pattern |

## Architecture Compliance Rules

| Rule | Source | Enforcement |
|------|--------|-------------|
| `internal/admission` at position 4 — imports {frame, hmac} only | ARCH-08 §6.5 | `go vet` + import guard; `failure_counter.go` MUST NOT import `internal/routing` |
| `internal/routing` at position 5 — imports {frame, hmac, admission} | ARCH-08 §6.5 | Import of `internal/admission` for `*FailureCounter` type is already in the allowed set |
| `FailureCounter` internal map slice NEVER returned by reference | go.md rule 12 | `Timestamps()` returns value copy; `just test-race` verifies |
| Timestamps in UTC | go.md rule 11 | `time.Now().UTC()` in `RecordHMACFailure` |
| Logger injected; no package-level logger | BC-2.05.005 PC-3 API contract | Constructor takes `logger Logger`; no `log.SetOutput` or global |
| `RecordHMACFailure` called on BOTH `ErrHMACVerificationFailed` return paths | BC-2.05.008 PC-5 + invariant 5 | `TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths` |
| `RecordHMACFailure` NOT called on successful HMAC verification | BC-2.05.008 PC-5 (negative) | Verified by TestRouteFrame test for success path (AC-009 negative assertion) |
| E-ADM-017 is distinct from E-ADM-016 | BC-2.05.005 PC-3 (E-ADM-017) vs BC-2.05.008 PC-2 (E-ADM-016) | String format; do NOT reuse E-ADM-016 for the aggregate alert |
| Boundary trimming: `timestamp < now - windowDuration` (strictly less) | BC-2.05.005 EC-008 | `TestFailureCounter_BoundaryEntryIsKept` |

## Forbidden Dependencies

The following packages MUST NOT appear in `internal/admission`'s import graph:

| Package | Reason |
|---------|--------|
| `internal/routing` | admission is at position 4; routing is position 5 — importing routing would create a cycle |
| `internal/session` | Not in admission's allowed import set (ARCH-08 §6.5) |
| `internal/tmux` | Not in admission's allowed import set |

If `internal/admission/failure_counter.go` imports any of the above, the build MUST
fail (Go import cycle detection via `go vet`/`go build`).

## Library & Framework Requirements

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Per go.mod |
| `sync` | stdlib | `sync.Mutex` for FailureCounter internal state |
| `time` | stdlib | `time.Time`, `time.Duration`, `time.Now().UTC()` |
| `testing` | stdlib | Table-driven tests; `t.Parallel()`; no testify |

## File Structure Requirements

| File | Action | Purpose |
|------|--------|---------|
| `internal/admission/failure_counter.go` | create | `FailureCounter` type; `NewFailureCounter`; `RecordHMACFailure`; lazy eviction; fire-once-per-crossing; E-ADM-017 emission via injected logger |
| `internal/admission/failure_counter_test.go` | create | AC-001 through AC-008, AC-010: sliding window, hysteresis, boundary, multi-source, concurrent-safe |
| `internal/routing/routing.go` | modify | Add `failureCounter *admission.FailureCounter` to `Router` struct; wire via option; call `RecordHMACFailure` on both `ErrHMACVerificationFailed` return paths |
| `internal/routing/routing_test.go` | extend | AC-009: `TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths`; verify not called on success |

## Spec Patches

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-27 | Initial story — Wave 3 FIX-NOW gate blocker F-2; implements BC-2.05.005 PC-3; adds E-ADM-017 aggregate alert |
