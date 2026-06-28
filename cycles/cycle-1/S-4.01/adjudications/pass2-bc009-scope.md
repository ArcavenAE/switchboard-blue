---
artifact_id: S-4.01-pass2-bc009-scope
document_type: adjudication
level: ops
story_id: S-4.01
title: "Pass 2 — BC-2.02.009 Scope Rulings: Router Wiring and Hit Counter"
status: final
producer: product-owner
timestamp: 2026-06-27T00:00:00
phase: 3
cycle: v1.0.0-greenfield
findings_addressed:
  - F-H1 (router drop-cache wiring scope)
  - F-H2 (hit counter + collision logging)
---

# S-4.01 Pass 2 — BC-2.02.009 Scope Rulings

These rulings are authoritative. Implementer and test-writer must satisfy the
contracts as ruled below. Do not re-adjudicate unless the cited spec is
version-bumped by the product-owner.

---

## RULING 1 — F-H1: Router Drop-Cache Wiring Scope

### Question

`Multipath` constructs a `dropCache` field (compound-key, per BC-2.02.009) but no
method on `Multipath` ever reads or writes it — there is no `Forward`/`OnFrameArrival`
method that consults the drop cache. Must S-4.01 include a reachable router-side
forwarding method that wires the `dropCache` for loop suppression? Or is `DropCache`
a primitive that a different story wires into a router, making the unused `dropCache`
field dead code that must be removed?

### Ruling

**DEFER. The `dropCache` field on `Multipath` is dead code and must be removed from
`Multipath`. The router-side wiring of `DropCache` into a forwarding path belongs to
a later story.**

S-4.01's scope is strictly defined by its six acceptance criteria. Only AC-006 covers
BC-2.02.009:

> **AC-006 (traces to BC-2.02.009 postcondition 1):** `DropCache` never exceeds its
> configured capacity (LRU eviction); checksum-based lookup is O(1).

AC-006 traces to **postcondition 1** only. It specifies `DropCache` as a testable
primitive with bounded capacity and O(1) lookup. It does not specify that `DropCache`
must be wired into any forwarding path within S-4.01. The Architecture Mapping table
in the story lists `DropCache` under `internal/multipath` with classification
`pure-core` — it is a data structure, not a wired effectful forwarder.

### Why the Forward/OnFrameArrival Path Is Out of Scope

The ARCH-03 `OnFrameArrival` pseudocode (v1.1, §Duplicate-and-Race) describes
router-level behavior. The existing `internal/routing` package (from S-2.02) owns
`RouteFrame` / `SVTNRoute`, which is where router-level frame handling lives. A
`Forward` method on `Multipath` that consults `dropCache` would need to either:

(a) live in `internal/routing` (violating import constraints — routing already owns
    the routing hot path, and ARCH-08 position 11 says multipath imports frame+paths,
    not the other direction), or

(b) introduce a new effectful forwarding method on `Multipath` that the router calls,
    which is an integration concern belonging to the wave that wires multipath into the
    routing dispatch loop.

Neither option is scoped to S-4.01. The dependency graph (dependency-graph.md) shows
S-4.04 ("split-horizon extends routing; needs paths from S-4.01 for ordering") as the
story that wires path-quality data into the routing engine. S-4.04 is the natural home
for wiring the router-side `DropCache` into the forwarding path alongside split-horizon
(BC-2.02.008), since both are router-level loop-prevention mechanisms that operate on
the same arriving frame.

### Dead-Code Removal Required in S-4.01

The unused `dropCache *DropCache` field on `Multipath` MUST be removed. An exported
primitive (`DropCache`) exists; callers construct it directly. Embedding it as a dead
field in `Multipath` is:

1. Misleading — it implies `Multipath.Receive` or `Multipath.Send` performs router-side
   loop suppression, which it does not.
2. A YAGNI violation per Go quality rules (do not add unused fields "for future use").
3. Inconsistent with the pure-core classification — the `Multipath` struct's documented
   purpose is dispatch and endpoint dedup, not router-side forwarding.

### Deferral Target: S-4.04

S-4.04 ("split-horizon extends routing; needs paths from S-4.01") is the owning story
for router-side forwarding extensions. When S-4.04 implements BC-2.02.008 (split-horizon)
it must also wire BC-2.02.009 router-side loop suppression, because:

- Both BCs (2.02.008 and 2.02.009) describe router-level frame handling on arrival.
- Both require consulting per-frame state (arrival interface, compound checksum key)
  before forwarding.
- ARCH-03 §Duplicate-and-Race's `OnFrameArrival` pseudocode shows them as a single
  frame-arrival handler.
- Wiring them separately in two different stories would require two passes over the
  same frame-arrival code path.

S-4.04 already traces to BC-2.02.008. Its scope must be extended to include the
router-side `DropCache` wiring per BC-2.02.009. This is a story-writer task (add
BC-2.02.009 to S-4.04's `bc_traces` and add the router forwarding AC).

### Authoritative Citations

**BC-2.02.009 description (v1.1):**
> "Each **router** maintains a bounded LRU cache of recently-forwarded
> `(frame_checksum, arrival_interface_id)` pairs. When a frame arrives whose compound
> key matches an entry in the cache, the frame is silently discarded as a loop
> duplicate."

The subject is "each router" — the router, not the endpoint Multipath dispatcher.

**BC-2.02.009 Trigger:**
> "Frame received at **router** after HMAC verification."

The trigger explicitly scopes this BC to the router layer. S-4.01's `Multipath` is
the endpoint dispatch primitive (pure-core, no I/O). The router layer is `internal/routing`.

**S-4.01 AC-006 (story v current):**
> "`DropCache` never exceeds its configured capacity (LRU eviction); checksum-based
> lookup is O(1). Test: `TestDropCache_BoundedCapacity`"

AC-006 tests the DropCache primitive in isolation. It says nothing about wiring.

**ARCH-03 §Duplicate-and-Race `OnFrameArrival` pseudocode (v1.1):**
```
OnFrameArrival(frame, arrival_interface_id):
  checksum = crc32(frame.outer_header || frame.payload)
  key = (checksum, arrival_interface_id)
  if DropCache.contains(key):
    silently discard (BC-2.02.009, DI-009)
    return
  DropCache.add(key)
  deliver(frame)
```
This pseudocode describes a method that does not exist on `Multipath` in S-4.01 and
is not required by any of S-4.01's six ACs. It is an architectural sketch for a
router-layer method, not a specification for S-4.01's deliverable.

### Precise Implementer Contract for S-4.01

1. Remove the `dropCache *DropCache` field from the `Multipath` struct.
2. Remove the `dropCache: NewDropCache(dropCacheCapacity)` initialization from
   `NewMultipath`.
3. `DropCache` remains as a standalone exported type in `internal/multipath` with
   `NewDropCache`, `Contains`, `Add`, `AddIfAbsent`, and `Len` methods — it is a
   reusable primitive that S-4.04 will wire.
4. No `Forward`, `OnFrameArrival`, or router-path method is required on `Multipath`
   in S-4.01.

### Required Story Edit (story-writer owns)

Add to S-4.01 story body a deferral note:

> **Deferral: BC-2.02.009 Router Wiring** — The router-side `OnFrameArrival`
> forwarding path that consults `DropCache` for loop suppression is deferred to
> S-4.04. `DropCache` is delivered as a standalone primitive in S-4.01 per AC-006.
> S-4.04 must add BC-2.02.009 to its `bc_traces` and implement the router-side
> wiring per ARCH-03 §Duplicate-and-Race.

S-4.04's story must have `BC-2.02.009` added to its `bc_traces` frontmatter array.

---

## RULING 2 — F-H2: Hit Counter and Collision Logging

### Question

BC-2.02.009 postcondition 2 mandates a "drop cache hit counter incremented (for
operator diagnostics)." EC-005 mandates collision events be "logged as a potential
collision event for investigation." `DropCache` exposes no `Hits()` accessor and no
collision hook. Must the hit counter be implemented within S-4.01? Must EC-005
collision logging be implemented in S-4.01?

### Ruling

**Hit counter: FIX IN S-4.01.** BC-2.02.009 postcondition 2 is unambiguous and
S-4.01 is the sole story that covers BC-2.02.009. The hit counter must be added to
`DropCache` in S-4.01.

**EC-005 collision logging: DEFER.** Collision logging is an effectful concern
(requires a logger/observer). `internal/multipath` is classified `pure-core`
(S-4.01 Purity Classification table). Effectful logging belongs at the wired layer,
not the pure-core primitive. The collision-log hook is deferred to S-4.04 (or
whichever story wires `DropCache` into a router forwarding path with an injected
logger).

### Authoritative Citations for the Hit Counter

**BC-2.02.009 postcondition 2 (v1.1) — exact text:**
> "On cache hit: frame is silently discarded; **drop cache hit counter incremented
> (for operator diagnostics)**."

This postcondition is mandatory. "incremented (for operator diagnostics)" is a
behavioral requirement, not a note. It is a postcondition of the BC, not an
observation or NFR.

**BC-2.02.009 canonical test vector 2 (v1.1):**
> "Same frame (checksum 0xABCD) arrives again → Frame dropped silently;
> **hit counter incremented**"

The canonical test vector lists "hit counter incremented" as an expected output.
Test-writers must verify this.

**BC-2.02.009 EC-003 (v1.1):**
> "Same frame arrives on the same interface twice within the cache window → Second
> arrival is suppressed. The compound key matches the cache entry from the first
> arrival — **drop cache hit counter incremented**."

The edge case also cites the counter. Three separate BC clauses (postcondition 2,
test vector 2, EC-003) mandate the hit counter. There is no ambiguity about whether
it is required.

**S-4.01 AC-006 and dependency-graph.md:** BC-2.02.009 traces exclusively to S-4.01
(dependency-graph.md BC-to-Stories Matrix, row BC-2.02.009 → S-4.01). No other story
can deliver this postcondition. If S-4.01 does not deliver the hit counter, the
requirement is permanently orphaned.

### Rationale for EC-005 Collision Logging Deferral

**BC-2.02.009 EC-005 (v1.1) — exact text:**
> "Two different frames hash to the same checksum on the same interface (collision)
> → Legitimate frame incorrectly suppressed. Probability negligible with 32-bit
> checksum at typical traffic rates. **Logged as a potential collision event for
> investigation.**"

"Logged" implies an effectful action — writing to a log sink. `internal/multipath`
is pure-core with no I/O (S-4.01 Purity Classification table). Injecting a logger
into `DropCache` or `Multipath` purely to satisfy EC-005 would violate the pure-core
classification and introduce an effectful dependency at the wrong layer.

The canonical pattern for pure-core packages in this codebase is logger injection at
the boundary layer (see `internal/routing.Router` with `WithLogger`). The same
pattern applies: when S-4.04 wires `DropCache` into the router forwarding path, it
injects the logger at that layer. At that point EC-005 collision logging can be
implemented without violating purity.

EC-005 is an edge case, not a postcondition. Its probability is noted as "negligible
with 32-bit checksum at typical traffic rates." Deferring logging to the wired layer
is appropriate — the detection invariant (a hit IS a potential collision OR a loop
duplicate) is preserved by the hit counter itself; the logging adds observability but
does not change correctness.

### Precise Implementer Contract for S-4.01 (Hit Counter)

**On `DropCache`:**

1. Add an `int64` hit counter field (use `sync/atomic` — counter increments must be
   race-safe without holding the main mutex, or increment under the existing mutex).
   Given that `DropCache` already holds a `sync.Mutex` for all operations, incrementing
   under the existing lock is acceptable and avoids mixed-locking complexity.

2. Add a `Hits() int64` method that returns the current hit count. The method must
   read the counter under the lock (or atomically) consistent with how it is written.

3. In `Contains`: do NOT increment here — `Contains` is a read-only probe that does
   not perform a hit action.

4. In `AddIfAbsent`: when the key is already present (the `if elem, ok := c.index[key]`
   branch), increment the hit counter before returning `false`. This is the canonical
   suppression path per the BC ("on cache hit: frame is silently discarded; drop cache
   hit counter incremented").

5. In `Add`: when the key is already present (the `if elem, ok := c.index[key]`
   branch), increment the hit counter. `Add` is called by callers who have already
   used `Contains` to check; a re-add of a present key IS a hit.

**Collision logging (EC-005): deferred to S-4.04.** No logger injection into
`DropCache` or `Multipath` is required in S-4.01.

### Precise Test-Writer Contract for S-4.01 (Hit Counter)

The existing `TestBC_2_02_009_DropCache_HitSuppresses` and
`TestBC_2_02_009_DropCache_MissForwards` tests must be extended or supplemented:

1. **`TestBC_2_02_009_DropCache_HitCounterIncremented`** — new test required:
   - Create `NewDropCache(10)`.
   - `Add(0xABCD, 1)` — miss path, no hit recorded.
   - Assert `dc.Hits() == 0`.
   - Call `AddIfAbsent(0xABCD, 1)` — this is a hit (key already present).
   - Assert `dc.Hits() == 1`.
   - Call `AddIfAbsent(0xABCD, 1)` again.
   - Assert `dc.Hits() == 2`.
   - Distinct key: `AddIfAbsent(0xEEEE, 1)` — miss.
   - Assert `dc.Hits() == 2` (no change).
   - Traces to: BC-2.02.009 postcondition 2 canonical test vector ("hit counter
     incremented").

2. **`TestBC_2_02_009_DropCache_ConcurrentHitCount`** — concurrent safety of the
   counter under `go test -race`:
   - Multiple goroutines each `Add` the same key, then call `AddIfAbsent` on it.
   - After all goroutines complete, assert `dc.Hits()` equals the expected total
     number of hits (goroutines × 1, since the key was added once and all subsequent
     `AddIfAbsent` calls hit it).

3. The existing `TestBC_2_02_009_DropCache_HitSuppresses` test (postcondition 2)
   should be updated to also assert `dc.Hits() == 1` after the hit.

### Required Story Edit (story-writer owns)

Add to S-4.01 story:

1. A new AC in the story body:
   > **AC-007 (traces to BC-2.02.009 postcondition 2):** `DropCache.Hits()` returns the
   > cumulative count of cache hits (frames suppressed). The counter increments on every
   > `AddIfAbsent` call where the key is already present. Test:
   > `TestBC_2_02_009_DropCache_HitCounterIncremented`.

2. A deferral note for EC-005:
   > **Deferral: BC-2.02.009 EC-005 collision logging** — Collision event logging
   > (EC-005) requires effectful logger injection and is deferred to S-4.04, where
   > `DropCache` is wired into the router forwarding path with an injected logger
   > (following the pattern established by `internal/routing.WithLogger`).

---

## Summary

| Finding | Verdict | Action Owner | Detail |
|---------|---------|-------------|--------|
| F-H1 — router drop-cache wiring | **DEFER to S-4.04** | implementer (remove dead field); story-writer (add deferral note + update S-4.04 bc_traces) | Remove `dropCache` field from `Multipath`. `DropCache` stays as standalone primitive. S-4.04 owns router-side wiring per ARCH-03 §OnFrameArrival. |
| F-H2 hit counter | **FIX IN S-4.01** | implementer + test-writer | Add `hits int64` counter to `DropCache`; add `Hits() int64` method; increment in `AddIfAbsent` and `Add` on cache hit. Add `TestBC_2_02_009_DropCache_HitCounterIncremented` and concurrent-safety variant. |
| F-H2 EC-005 collision logging | **DEFER to S-4.04** | story-writer (add deferral note) | `internal/multipath` is pure-core; logger injection at wired layer (S-4.04) per project pattern. |

---

## Cross-Reference

| Artifact | Change Required | Owner |
|----------|----------------|-------|
| `internal/multipath/multipath.go` | Remove `dropCache *DropCache` field from `Multipath` struct and its initialization in `NewMultipath` | implementer |
| `internal/multipath/multipath.go` | Add `hits int64` to `DropCache`; add `Hits() int64` method; increment counter on hit in `AddIfAbsent` and `Add` | implementer |
| `internal/multipath/multipath_test.go` | Add `TestBC_2_02_009_DropCache_HitCounterIncremented`; add concurrent-safety variant; update existing hit test to assert `Hits()==1` | test-writer |
| S-4.01 story body | Add AC-007 (hit counter); add deferral notes for router wiring and EC-005 logging | story-writer |
| S-4.04 story | Add `BC-2.02.009` to `bc_traces`; add AC for router-side `OnFrameArrival` + `DropCache` wiring + EC-005 collision logging with injected logger | story-writer |
| BC-2.02.009 | No behavioral change. No version bump required. | — |
