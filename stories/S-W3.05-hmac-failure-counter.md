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
estimated_points: 8
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
acceptance_criteria_count: 17
version: "1.3.2"
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

### AC-003 (traces to BC-2.05.005 PC-3 + error-taxonomy v2.2 / E-ADM-017 — E-ADM-017 emission on threshold crossing)
When the post-trim count for a `srcAddr` reaches or exceeds `threshold`, the
`FailureCounter` emits a structured log event via the injected `Logger` interface
(`Log(msg string)` — level-less seam; severity is taxonomy-owned, not a logger level)
carrying the canonical E-ADM-017 message format:
`"E-ADM-017 HMAC failure rate alert: ≥<threshold> failures in <window_seconds>s from src <src_addr>"`.
The code literal "E-ADM-017" is embedded at message start for operator grep-ability.
`<threshold>` and `<window_seconds>` are the FailureCounter's configured values (default:
5 and 60); `<src_addr>` is the lowercase hex encoding of the 8-byte SrcAddr field.
Severity is `degraded` (daemon continues) per error-taxonomy v2.2 — the Logger seam
does NOT encode severity as a logger level. No global state. No direct call to
`log.Printf` or any package-level logger.
- **Test:** `TestFailureCounter_EmitsEADM017AtThreshold`

### AC-004 (traces to BC-2.05.005 PC-3 + EC-005 + EC-009 — hysteresis/re-fire, drain-only re-arm)
E-ADM-017 fires on the Nth call that causes the sliding-window count to reach threshold
(N = threshold). Subsequent `RecordHMACFailure` calls in the same un-re-armed window do
NOT re-emit the alert (append-skip is in force: new timestamps are NOT appended while
`firedAt[srcAddr]` is set and re-arm has not yet triggered). The counter re-arms using
**drain-only re-arm**: re-arm triggers when `len(keep) == 0` after trim — i.e., all
pre-fire entries have aged out of the window. Under the append-skip policy (EC-011), no
post-fire timestamps are ever appended, so `keep[0].After(firedAt[srcAddr])` is dead
code and is NOT a re-arm condition. On re-arm, `firedAt[srcAddr]` is deleted and normal
append+counting resumes. Period between alerts ≈ `windowDuration` under sustained attack.
- **Test:** `TestFailureCounter_HysteresisNoBriefRefire`,
  `TestFailureCounter_RearmOccursOnFirstCallAfterDrain` (renamed per BC-2.05.005 v1.8
  test-writer hand-off; replaces `TestFailureCounter_RearmBoundaryAtLastFireTimestamp`
  which tested now-dead code under append-skip).
(traces to BC-2.05.005 PC-3 + EC-005 + EC-009)

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

### AC-009 (traces to BC-2.05.008 EC-006 — injection seam)
`RouteFrame`'s Router is wired via `WithFailureCounter(fc hmacFailureRecorder)` — an
unexported interface `{ RecordHMACFailure(srcAddr string) }`; `*admission.FailureCounter`
is the production implementation; tests may inject a fake satisfying the interface.
`RecordHMACFailure` is called immediately before returning `ErrHMACVerificationFailed`
on BOTH failure paths: (a) tag mismatch (`verifyFrameHMAC` returns false), and (b) no
forwarding-table entry (auth key unavailable). `RecordHMACFailure` is NOT called on a
successful HMAC verification.
- **Test:** `TestRouteFrame_WithFailureCounterInterface`. (traces to BC-2.05.008 EC-006)

### AC-010 (traces to BC-2.05.005 PC-3 — concurrency safety)
`RecordHMACFailure` is safe for concurrent calls from multiple goroutines. The
`sync.Mutex` is held for the complete trim+append+threshold-check sequence. No
internal slice is returned by reference to callers. If a `Timestamps(srcAddr string)
[]time.Time` inspector is needed (e.g., in tests), it returns a copy of the slice
(go.md rule 12: no internal pointer leaks from locked accessors). `go test -race` MUST
pass with concurrent callers.
- **Test:** `TestFailureCounter_ConcurrentCallsRaceSafe` (run with `-race`)

### AC-011 (traces to BC-2.05.005 EC-010 — memory cap)
After inserting `maxTrackedSources+1` (65,537) distinct `srcAddr`s, `len(counts) <=
maxTrackedSources` (65,536) — map growth is capped. The LRU key (oldest most-recent-failure
timestamp) is evicted from both `counts` and `firedAt` before inserting the new key.
- **Test:** `TestFailureCounter_SourceCapBoundsMapGrowth`. (traces to BC-2.05.005 EC-010)

### AC-012 (traces to BC-2.05.005 EC-005 — dead-key eviction after drain)
After 5 failures then a full window drain (all entries age out), the source re-arms — a
subsequent post-drain threshold crossing fires E-ADM-017 again; `firedAt[srcAddr]` is
cleared on drain. Both `counts` and `firedAt` entries are deleted entirely when the
post-trim slice is empty (dead-key eviction — no unbounded map growth). **Discriminating
test requirement:** the test must distinguish an implementation that calls
`delete(counts, srcAddr)` on drain from one that leaves an empty slice. The test must
observe the key deletion directly — e.g., by asserting that `SourceCount()` drops when
drain occurs without a same-call re-append (drain-then-rearm ordering). Asserting only
`SourceCount() >= 1` is non-discriminating and insufficient.
- **Test:** `TestFailureCounter_DeadKeyEvictedAfterDrain`. (traces to BC-2.05.005 EC-005)

### AC-013 (traces to BC-2.05.005 PC-3 — constructor validation)
`NewFailureCounter` rejects three classes of invalid arguments eagerly at construction
time; all are programmer-error guards, not runtime error paths:
1. `NewFailureCounter(0, 60s, logger)` panics — `threshold < 1`.
2. `NewFailureCounter(5, 0, logger)` panics — `windowDuration <= 0`.
3. `NewFailureCounter(5, 60s, nil)` panics with message
   `"admission: NewFailureCounter: logger must not be nil"` — a nil logger would be
   dereferenced at E-ADM-017 emission (CWE-476 / SEC-001); the same class of
   programmer error as threshold < 1 or windowDuration <= 0.

All three panics use `panic(...)` in `NewFailureCounter` and are covered by the
`TestNewFailureCounter_PanicsOnInvalidArgs` test (sub-cases: threshold=0, windowDuration=0,
nil-logger; the nil-logger sub-case was added in commit f6038d2 and resides in
`internal/admission/failure_counter_adversarial_test.go`).
- **Test:** `TestNewFailureCounter_PanicsOnInvalidArgs` (covers all three sub-cases).
  (traces to BC-2.05.005 PC-3)

### AC-014 (traces to BC-2.05.005 EC-009 — sustained attack re-fires)
With `WithNow` clock injection — 5 failures at T=0 fire 1 E-ADM-017 alert; advancing
to T=61s drains the window and re-arms the counter; 5 more failures at T=61–62s fire a
2nd E-ADM-017 alert; total exactly 2 alerts.
- **Test:** `TestFailureCounter_SustainedAttackReFires`. (traces to BC-2.05.005 EC-009)

### AC-015 (traces to error-taxonomy v2.2 / E-ADM-017 — alert message format, FULL canonical form)
The E-ADM-017 log message MUST match the FULL canonical parameterized format from
error-taxonomy v2.2 (row E-ADM-017):
`"E-ADM-017 HMAC failure rate alert: ≥<threshold> failures in <window_seconds>s from src <src_addr>"`.
The format requires **both** the leading "E-ADM-017" code literal **and** the
"HMAC failure rate alert:" phrase — neither may be omitted. `<threshold>` and
`<window_seconds>` are the FailureCounter's configured integer values (not string
literals); `<src_addr>` is the lowercase hex encoding of the 8-byte SrcAddr field.
The test MUST assert the phrase "HMAC failure rate alert:" IS present in the emitted
message (not absent). Any implementation that drops this phrase is non-conformant.
Note: a prior erroneous reconciliation note incorrectly claimed AC-015 should drop
the "HMAC failure rate alert:" phrase — that was false. Error-taxonomy v2.2 includes
the phrase and the changelog NEVER removed it. This AC restores the canonical form.
- **Test:** `TestFailureCounter_AlertMessageFormat`. (traces to error-taxonomy v2.2 / E-ADM-017)

### AC-016 (traces to BC-2.05.005 EC-011 — per-source slice bound, CWE-770 amplification mitigation)
After an alert fires for a `srcAddr` (i.e., `firedAt[srcAddr]` is non-zero and re-arm
has not yet triggered), new timestamps MUST NOT be appended to the slice for that
source (append-skip policy). The per-source slice is bounded at `threshold` entries at
all times — the entries present at or before the alert threshold-crossing. Under a
high-rate attack (`rate >> threshold/windowDuration`), memory per source is bounded at
`threshold × sizeof(time.Time)` regardless of call rate. The test injects `threshold`
failures to fire the alert, then injects 1,000,000 additional calls with the clock
frozen (no entries can age out), and asserts that `len(Timestamps(src)) == threshold`
(slice did not grow beyond `threshold` entries).
- **Test:** `TestFailureCounter_HighRateAttackBoundedSlice`. (traces to BC-2.05.005 EC-011)

### AC-017 (traces to VP-059 v1.1 — property-based test, stateful model checker)
A property-based test using a stateful model checker over arbitrary generated call
sequences with injected clock verifies VP-059 properties (a)–(e):
- **(a)** E-ADM-017 fires exactly on the call that brings the post-trim count to
  threshold (not before).
- **(b)** Subsequent calls in the same un-re-armed window do NOT fire E-ADM-017.
- **(c)** After re-arm (drain-only: `len(keep) == 0` after trim, which is the sole
  re-arm trigger under append-skip), the next threshold crossing fires E-ADM-017 again.
- **(d)** Under a continuous stream of failures at rate ≥ threshold/windowDuration,
  E-ADM-017 alert count is ≥ 2 (counter never goes permanently silent).
- **(e)** Live key count `len(counts)` is always ≤ `maxTrackedSources` (65,536)
  regardless of distinct source count.

The `capturingLogger` must satisfy the `admission.Logger` interface as `Log(msg string)`
(level-less seam — not `Error(msg string)`). Clock injection is via `WithNow`. All
vectors are deterministic; no real-time waits.
- **Tests:** `TestFailureCounter_PropertiesABCD` (properties a–d, stateful model);
  `TestFailureCounter_PropertyE_MemoryBound` (property e, adversarial injection:
  2 × maxTrackedSources distinct sources). (traces to VP-059 v1.1)

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
| New test files (2 new; includes VP-059 proptest) | ~1,500 |
| Tool outputs overhead | ~200 |
| **Total** | **~7,300** |
| Agent context window | 200K |
| **Budget usage** | **~3.5%** |

## Tasks

1. [ ] Read BC-2.05.005 (full, including PC-3 API contract + EC-005–EC-008), BC-2.05.008
       (full, including EC-006 + PC-5 + invariant 5), ARCH-08 §6.5 (positions 4 and 5)
2. [ ] Write failing tests for AC-001 through AC-017 in:
       - `internal/admission/failure_counter_test.go` (AC-001 through AC-008, AC-010 through AC-017)
       - `internal/routing/routing_test.go` extension (AC-009)
3. [ ] Verify Red Gate: all 17 tests fail before any implementation
4. [ ] Create `internal/admission/failure_counter.go`:
       - `type FailureCounter struct` with `mu sync.Mutex`, `counts map[string][]time.Time`,
         `threshold int`, `windowDuration time.Duration`, `logger Logger`,
         `firedAt map[string]time.Time` (last-fire timestamp per srcAddr), `now func() time.Time`
       - `NewFailureCounter(threshold int, windowDuration time.Duration, logger Logger, opts ...FailureCounterOption) *FailureCounter`
         — panic if `threshold < 1`, `windowDuration <= 0`, or `logger == nil` (AC-013)
       - `WithNow(fn func() time.Time) FailureCounterOption` — clock injection for tests (AC-014)
       - `hmacFailureRecorder` unexported interface `{ RecordHMACFailure(srcAddr string) }` (AC-009)
       - `RecordHMACFailure(srcAddr string)` — trim (`timestamp < now - windowDuration`),
         drain-only re-arm check (if `firedAt[srcAddr]` set and `len(keep)==0`, delete
         `firedAt[srcAddr]` — AC-004), dead-key eviction when count=0 after trim+re-arm
         (delete both `counts` and `firedAt` entries — AC-012), LRU source cap at
         65,536 (AC-011), **append-skip**: only append new timestamp if `firedAt[srcAddr]`
         is zero (not currently fired — AC-016), check threshold, emit canonical E-ADM-017
         message `"E-ADM-017 HMAC failure rate alert: ≥<threshold> failures in <window_seconds>s from src <src_addr>"`,
         update `firedAt` state (AC-014)
       - No background goroutine; lazy eviction on every call
       - `Timestamps(srcAddr string) []time.Time` returning a copy (if needed for testing)
       - Compile-time constant `maxTrackedSources = 65536` (AC-011)
5. [ ] Confirm UTC timestamps: all `time.Now()` calls in `RecordHMACFailure` use `time.Now().UTC()`
6. [ ] Confirm no internal pointer returned: `Timestamps()` returns `append([]time.Time{}, ...)` copy
7. [ ] Modify `internal/routing/routing.go`:
       - Add `failureCounter hmacFailureRecorder` field to `Router` struct (unexported interface)
       - Wire via `WithFailureCounter(fc hmacFailureRecorder)` option
       - In `RouteFrame`: call `r.failureCounter.RecordHMACFailure(fmt.Sprintf("%x", hdr.SrcAddr))`
         on BOTH `ErrHMACVerificationFailed` paths (tag mismatch + no forwarding entry),
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
| `internal/routing` at position 5 — imports {frame, hmac, admission} | ARCH-08 §6.5 | Import of `internal/admission` for the `hmacFailureRecorder` interface is already in the allowed set |
| `FailureCounter` internal map slice NEVER returned by reference | go.md rule 12 | `Timestamps()` returns value copy; `just test-race` verifies |
| Timestamps in UTC | go.md rule 11 | `time.Now().UTC()` in `RecordHMACFailure` |
| Logger injected; no package-level logger | BC-2.05.005 PC-3 API contract | Constructor takes `logger Logger`; no `log.SetOutput` or global |
| `RecordHMACFailure` called on BOTH `ErrHMACVerificationFailed` return paths | BC-2.05.008 PC-5 + invariant 5 | `TestRouteFrame_CallsRecordHMACFailureOnBothFailurePaths` |
| `RecordHMACFailure` NOT called on successful HMAC verification | BC-2.05.008 PC-5 (negative) | Verified by TestRouteFrame test for success path (AC-009 negative assertion) |
| E-ADM-017 is distinct from E-ADM-016 | BC-2.05.005 PC-3 (E-ADM-017) vs BC-2.05.008 PC-2 (E-ADM-016) | String format; do NOT reuse E-ADM-016 for the aggregate alert |
| Boundary trimming: `timestamp < now - windowDuration` (strictly less) | BC-2.05.005 EC-008 | `TestFailureCounter_BoundaryEntryIsKept` |
| E-ADM-017 message MUST include BOTH "E-ADM-017" prefix AND "HMAC failure rate alert:" phrase | error-taxonomy v2.2 row E-ADM-017 + BC-2.05.005 PC-3 canonical test vector | `TestFailureCounter_AlertMessageFormat` asserts both substrings present |
| Logger seam is level-less: `Log(msg string)` — severity is NOT encoded as a logger level | BC-2.05.005 v1.5 O-1 adjudication; VP-059 v1.1 | `capturingLogger.Log(msg string)` in tests; no `Error()` method on Logger interface |
| Append-skip: no post-fire timestamp appended while `firedAt[srcAddr]` is set | BC-2.05.005 PC-3 / EC-011 (CWE-770 M-1) | `TestFailureCounter_HighRateAttackBoundedSlice` |
| Re-arm is drain-only: `len(keep) == 0` after trim | BC-2.05.005 v1.8 PC-3; `keep[0].After(firedAt)` path is dead code under append-skip | `TestFailureCounter_RearmOccursOnFirstCallAfterDrain` |

### Additional API / Constructor Constraints

- `WithNow(fn func() time.Time)` functional option on `FailureCounter` enables clock injection for deterministic tests (BC-2.05.005 PC-3 v1.4).
- Constructor panics if `threshold < 1`, `windowDuration <= 0`, or `logger == nil` (BC-2.05.005 PC-3 v1.8 constructor contract; nil-logger panic added in v1.7 per SEC-001/CWE-476).

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
| `internal/admission/failure_counter_test.go` | create | AC-001 through AC-008, AC-010 through AC-017: sliding window, hysteresis, boundary, multi-source, concurrent-safe, append-skip bound (AC-016), VP-059 proptest (AC-017) |
| `internal/routing/routing.go` | modify | Add `failureCounter *admission.FailureCounter` to `Router` struct; wire via option; call `RecordHMACFailure` on both `ErrHMACVerificationFailed` return paths |
| `internal/routing/routing_test.go` | extend | AC-009: `TestRouteFrame_WithFailureCounterInterface`; verify not called on success |

## Spec Patches

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-06-27 | Initial story — Wave 3 FIX-NOW gate blocker F-2; implements BC-2.05.005 PC-3; adds E-ADM-017 aggregate alert |
| 1.1 | 2026-06-27 | PO adjudication of adversarial-convergence findings: new ACs 011-015, amended AC-004/009, points 5→8 (BC-2.05.005 v1.4, BC-2.05.008 v1.3, error-taxonomy v1.9) |
| 1.3 | 2026-06-27 | Propagate BC-2.05.005 v1.7 nil-logger constructor precondition (SEC-001/CWE-476) into AC-013: NewFailureCounter(5, 60s, nil) panics with "admission: NewFailureCounter: logger must not be nil"; update all forward-facing BC-2.05.005 v1.6 references to v1.7 |
| 1.3.2 | 2026-06-27 | Version-pin refresh (consistency-audit Finding 2.3): all error-taxonomy version citations updated v1.9→v2.2 (AC-003 heading + body, AC-015 heading + body × 2, Architecture Compliance table row). E-ADM-017 format is unchanged across v1.9→v2.2; no behavioral change. |
| 1.3.1 | 2026-06-27 | Wave-3 consistency audit F-2.2: corrected stale BC-2.05.005 version citation v1.7→v1.8 (spec-hygiene bump per adversary OBS-3; no behavioral change) |
| 1.2 | 2026-06-27 | Correct defect introduced in prior reconciliation pass: (1) AC-003 and AC-015 now assert the FULL canonical E-ADM-017 message format from error-taxonomy v1.9 — `"E-ADM-017 HMAC failure rate alert: ≥<threshold> failures in <window_seconds>s from src <src_addr>"` — including BOTH the "E-ADM-017" prefix AND the "HMAC failure rate alert:" phrase (prior pass erroneously dropped the phrase); (2) AC-003 O-1 fix: Logger seam is level-less (`Log(msg string)`), severity is taxonomy-owned (degraded), not a logger ERROR level; (3) AC-004 rewritten to drain-only re-arm per BC-2.05.005 v1.6 — `len(keep)==0` after trim is the sole re-arm condition; `keep[0].After(firedAt)` is dead code under append-skip and removed; test renamed to `TestFailureCounter_RearmOccursOnFirstCallAfterDrain`; (4) AC-012 discrimination note added — test must observe `delete(counts, srcAddr)` directly; (5) AC-016 added: per-source slice bound/CWE-770 (BC-2.05.005 EC-011, `TestFailureCounter_HighRateAttackBoundedSlice`); (6) AC-017 added: VP-059 v1.1 property-based test mandate (`TestFailureCounter_PropertiesABCD` + `TestFailureCounter_PropertyE_MemoryBound`); AC count 15→17; points unchanged at 8 |
