---
artifact_id: S-BL.DISCOVERY-WIRE-map-bounding-ruling
document_type: decision
level: ops
version: "1.0"
status: final
producer: architect
timestamp: 2026-07-20T00:00:00Z
cycle: cycle-1
stories_in_scope: [S-BL.DISCOVERY-WIRE]
bc_traces:
  - BC-2.03.001
related_docs:
  - decisions/S-BL.DISCOVERY-WIRE-task6d-wiring-seam-ruling.md
  - decisions/S-BL.DISCOVERY-WIRE-fanout-resolution-ruling.md
  - stories/S-BL.DISCOVERY-WIRE.md
---

# Ruling: S-BL.DISCOVERY-WIRE — Bounding the Two Unbounded Per-`(SVTNID, NodeAddr)` Maps

All factual claims are read-verified against worktree `feature/S-BL.DISCOVERY-WIRE-FANOUT`
HEAD `8058104432a549711220a96b2d334e79baf9ccd0`, branch `feature/S-BL.DISCOVERY-WIRE-FANOUT`.
File:symbol anchors cited per TD-031.

This ruling resolves one open design question scoped explicitly into `S-BL.DISCOVERY-WIRE` by
the human architect: how to bound the two per-`(SVTNID, NodeAddr)` maps that grow with the
admitted-node population over process lifetime with no prune or cap. The human accepted that
the fix touches already-merged out-of-story code on `develop` (Tasks 1–5 are merged; the
maps are live in the shipped codebase).

This ruling does NOT modify the story file, STATE.md, or STORY-INDEX. Required story-file
changes are flagged at the end and owned by story-writer.

---

## Context and CWE-770 Standing

The codebase's own attacker-facing map (`internal/admission/failure_counter.go`,
`maxTrackedSources = 65536`, LRU-eviction on insert, CWE-770 comment at line 38)
establishes the local precedent: maps that could grow unboundedly SHOULD be bounded with a
named constant and a defined eviction strategy, even when the growth rate is low. Both maps
below are populated ONLY after HMAC verification of an admitted identity — the
attacker-reachable CWE-770 vector is closed. The bounding here is defensive-in-depth, not
a first-line-of-defense fix.

---

## Verified Premises

| # | Premise | File:Symbol | Evidence |
|---|---|---|---|
| VP-A | `relayRateCap.last` is `map[relayRateKey]time.Time`, written at `allow()` line 80 and read at line 71; guarded by `c.mu sync.Mutex`; keyed on composite struct `{svtnID [16]byte; nodeAddr [8]byte}` | `cmd/switchboard/relay_rate_cap.go`, `relayRateCap.last` | Direct read, lines 32, 65–81 |
| VP-B | An entry in `relayRateCap.last` whose timestamp is older than `c.interval` (1 second) carries ZERO information — `allow()` at line 71 tests `t.Sub(last) < c.interval`; a stale entry returns `true` immediately and overwrites itself at line 80 | `cmd/switchboard/relay_rate_cap.go`, `allow` | Direct read, lines 69–81 |
| VP-C | `RouterIngest.lastSeen` is `map[lastSeenKey]uint64`, written at `Ingest()` line 343 and read at line 341; guarded by `ri.mu sync.Mutex`; keyed on `{svtnID [16]byte; nodeAddr [8]byte}` | `internal/discovery/discovery_wire.go`, `RouterIngest.lastSeen` | Direct read, lines 202, 338–345 |
| VP-D | `lastSeen` entries carry a live SECURITY invariant: dropping an entry means the next datagram for that key is treated as a cold-start (AC-008 — "no prior `lastSeen` entry is always accepted"). An evicted key re-opens a replay window bounded to one heartbeat interval (EC-006) | `internal/discovery/discovery_wire.go`, `RouterIngest.Ingest` replay gate; story AC-008/EC-006 | Direct read, lines 337–346; story v2.20 lines 1303–1328, 1739–1740 |
| VP-E | `maxTrackedSources = 65536` in `internal/admission/failure_counter.go` is the existing codebase precedent for map bounding; it uses O(N) LRU scan on insert when at capacity; cited as CWE-770 mitigation | `internal/admission/failure_counter.go`, `maxTrackedSources`; `evictLRU` | Direct read, lines 38–41, 189–212 |
| VP-F | `relayRateCap.allow()` already holds `c.mu.Lock()` for its entire body; any prune logic added inside `allow()` executes under the existing lock with no additional lock overhead | `cmd/switchboard/relay_rate_cap.go`, `allow` lines 64–81 | Direct read |
| VP-G | `RouterIngest.Ingest()` acquires `ri.mu.Lock()` at line 338 before touching `lastSeen`; the lock is held via `defer ri.mu.Unlock()` to the function return | `internal/discovery/discovery_wire.go`, `Ingest`, lines 338–346 | Direct read |
| VP-H | The admitted-node population for `relayRateCap.last` and `lastSeen` is the SAME population: entries are only inserted after HMAC verification of an admitted `(SVTNID, NodeAddr)` pair. The realistic upper bound for a healthy deployment is hundreds to low-thousands, not tens-of-thousands, of admitted nodes | Story Non-Goals; AC-012 postcondition 3 (aggregate rate cap blocks pre-auth floods) | Architectural context; `maxTrackedSources = 65536` is conservatively above this |
| VP-I | go.md rule 12 ("Never return internal pointers from a locked accessor") applies: any prune pass must happen INSIDE the existing lock in both types, not in a separate goroutine that would race with map mutations | `.claude/rules/go.md`, rule 12 | Rule text |
| VP-J | ARCH-08 §6.5 import-DAG position 14 (`internal/discovery`) may import ONLY `internal/routing`; position 5 (`cmd/switchboard`) may import any internal package | Story architecture rules; ARCH-08 §6.5 | Story v2.20 line 889; fanout-resolution-ruling.md VP-5 |

---

## Decision 1 — `relayRateCap.last`: Prune-by-Age on `allow()`

### Ruling

**Prune stale entries opportunistically inside `allow()`, under the existing `c.mu.Lock()`.**
An entry is stale when `t.Sub(last) >= c.interval` (i.e., a re-allowance has already
occurred for that key, or the entry was written and then no further calls arrived within
the window). The prune is unconditional: delete every entry whose stored timestamp is older
than `c.interval` from `now()`.

### Why prune-by-age is safe and lossless here

VP-B establishes the critical semantic: an entry older than `c.interval` would be
unconditionally overwritten by the next `allow()` call for that key regardless of whether
it was pruned. The entry encodes no information beyond "this key was last allowed at time
T" — once T is more than `c.interval` ago, the key is in the "allowed" state. Deleting it
now produces exactly the same result as leaving it: the next `allow()` for that key sees no
entry, returns `true`, and records a fresh timestamp. Lossless by construction.

### Exact mechanism: amortized prune threshold

A full map scan on every `allow()` call is O(N) and unnecessary for a map expected to hold
at most hundreds of active entries. However, given the ~1/sec admission rate per key,
entries naturally idle out between calls for a given key, so the map DOES accumulate
genuinely-stale entries over time as keys disappear from the admitted set (nodes depart).

**Ruling:** Add a constant `maxRelayRateCapEntries` and prune ALL stale entries when `len(c.last) > maxRelayRateCapEntries/2` at the top of `allow()`, before the key lookup. This amortizes the O(N) scan across many calls and bounds peak map size to `maxRelayRateCapEntries`.

```go
// At top of allow(), under c.mu.Lock(), before the key lookup:
if len(c.last) > maxRelayRateCapEntries/2 {
    now := c.now() // already needed below; compute once
    for k, ts := range c.last {
        if now.Sub(ts) >= c.interval {
            delete(c.last, k)
        }
    }
}
```

The implementation must restructure `allow()` to compute `t := c.now()` ONCE at the top
and reuse it in both the prune pass and the existing key-lookup/update logic — no second
`c.now()` call.

### Complexity

The prune pass is O(N) when triggered, but triggered at most once per `maxRelayRateCapEntries/2`
net-new insertions. Amortized O(1) per `allow()` call over the lifetime of the map. For the
expected admitted-node scale (hundreds), N is small and the O(N) scan cost is negligible.

### Cap constant

```go
const maxRelayRateCapEntries = 65536
```

Matches `maxTrackedSources` in `internal/admission/failure_counter.go` (VP-E). Rationale
for matching: both maps track admitted-node-population-sized state; the admitted population
is the same upper bound for both; using the same constant makes the relationship explicit
and avoids independent justification for a different value. Named `maxRelayRateCapEntries`
(not `maxTrackedSources`) to be local to the rate-cap file and avoid cross-package coupling.

---

## Decision 2 — `RouterIngest.lastSeen`: LRU Cap

### Why prune-by-age is NOT safe here

VP-D establishes the security constraint: evicting a `lastSeen` entry means the key
transitions from "known, with a high watermark" to "cold-start". AC-008 states:
"a cold-start datagram — no prior `lastSeen` entry — is always accepted." Therefore,
evicting `lastSeen[k]` for a key whose legitimate sender is still active silently opens a
one-advertisement replay window for that key. An adversary who can observe the multicast
channel and has a copy of a recently-captured (but stale) advertisement for key k can replay
it immediately after eviction, and it will be accepted as a cold-start.

Prune-by-age (drop `lastSeen[k]` when the key has been inactive for N seconds) is
therefore NOT safe for `lastSeen` without additional constraints, because "inactive for N
seconds" is not a reliable signal that the sender is gone — the sender's heartbeat interval
may simply be longer than N, or the multicast path may have been transiently lossy.

### Ruling: LRU cap, same size as `maxTrackedSources`

**Bound `lastSeen` to a maximum of `maxLastSeenEntries` entries using O(N) LRU eviction
on insert, modeled exactly on `FailureCounter.evictLRU`.**

```go
const maxLastSeenEntries = 65536
```

On each `Ingest()` call that would insert a new key (the `!seen` branch at line 341), if
`len(ri.lastSeen) >= maxLastSeenEntries`, evict the key with the LOWEST stored `Sequence`
watermark before inserting the new entry. The eviction check and evictLRU call happen INSIDE
the existing `ri.mu.Lock()` scope (VP-G, VP-I).

### Security trade-off analysis (required by dispatch)

Evicting the LRU (lowest-watermark) `lastSeen` entry re-opens a cold-start window for
the evicted key. The adversarial question is: can an attacker force eviction of a target key
by flooding with new admitted-identity datagrams?

**Answer: No, for the following reasons:**

1. **Insertion into `lastSeen` requires HMAC verification** (Ingest line 304–312). An
   attacker cannot manufacture a valid HMAC for a fabricated `(SVTNID, NodeAddr)` pair
   without the corresponding admission key. Only admitted nodes can insert entries.
2. **The admitted-node population is the same population bounded by `maxTrackedSources =
   65536`** (VP-H). In any realistic deployment, the number of admitted nodes that could
   simultaneously be generating heartbeats is orders of magnitude below 65536. Reaching the
   cap requires 65536 distinct admitted `(SVTNID, NodeAddr)` pairs ALL sending heartbeats
   simultaneously — an operational scenario that implies a deployment of unprecedented scale,
   or a compromised set of admitted credentials large enough that replay protection of any
   individual node is no longer the binding concern.
3. **The replay window on eviction is bounded to one heartbeat interval** (EC-006 already
   accepts this bound: "First frame for that `(SVTNID,NodeAddr)` pair is accepted
   unconditionally regardless of its `Sequence`, bounding the residual replay window to at
   most one heartbeat interval"). Eviction-induced cold-start has the same bound as
   router-restart cold-start — explicitly accepted by EC-006 and SEC-DW-07.
4. **LRU (lowest sequence) vs. LRU (oldest timestamp):** Evicting the entry with the lowest
   stored `Sequence` watermark is preferred over evicting the entry with the oldest last-seen
   timestamp because it targets the node that has sent the fewest RECENT advertisements — i.e.,
   the node most likely to have already left the network. This minimizes operational disruption.
   "Oldest timestamp" is not available in the current `lastSeen` structure (which stores only
   `uint64 sequence`, not `time.Time`). Adding a timestamp field to track LRU-by-time would
   increase the map's memory footprint and is not warranted. LRU-by-sequence is a reasonable
   proxy.

**Accepted residual:** At a deployment scale of 65536+ admitted nodes all simultaneously
active, the LRU entry may be an active node whose replay window opens on eviction. This is
accepted because:
- It requires an admitted-node population at the cap boundary, which is far beyond any
  realistic deployment the product currently targets.
- The opened window is bounded to one heartbeat interval (EC-006 posture).
- The alternative (no bound) is worse: unbounded map growth until OOM.

### Eviction implementation

Add a helper `evictLRULastSeen()` to `RouterIngest`, called under `ri.mu.Lock()`:

```go
// evictLRULastSeen removes the lastSeen entry with the lowest stored Sequence
// watermark (the node least recently active). Called when len(ri.lastSeen) >= maxLastSeenEntries.
// Must be called with ri.mu already held.
func (ri *RouterIngest) evictLRULastSeen() {
    var lruKey lastSeenKey
    var lruSeq uint64
    first := true
    for k, seq := range ri.lastSeen {
        if first || seq < lruSeq {
            lruKey = k
            lruSeq = seq
            first = false
        }
    }
    if !first {
        delete(ri.lastSeen, lruKey)
    }
}
```

Insert call site in `Ingest()`, inside the `ri.mu.Lock()` scope, before the current
`k := lastSeenKey{...}` line:

```go
// Bound the lastSeen map (SEC-DW-07 bounding, map-bounding-ruling.md Decision 2).
// Only check on a cold-start path to avoid O(N) scan on every Ingest call.
k := lastSeenKey{svtnID: svtnID, nodeAddr: nodeAddr}
if _, seen := ri.lastSeen[k]; !seen && len(ri.lastSeen) >= maxLastSeenEntries {
    ri.evictLRULastSeen()
}
last, seen := ri.lastSeen[k]
```

**Complexity:** O(N) scan on the cold-start path only (new key insertions). Hot path
(existing key, replay-discard or forward-advance) has no scan overhead. For the expected
admitted-node scale (hundreds), N is small.

---

## Decision 3 — Spec Anchor

### Does this need a new AC?

**No new AC is required.** The bounding of both maps is a resource-safety obligation that
falls under the existing spec anchors:

- **`relayRateCap.last` bounding** → **AC-018 postcondition 1 + BC-2.03.001 Postcondition 5
  (SEC-DW-09).** AC-018 already governs `relayRateCap` completely. Postcondition 5 of
  BC-2.03.001 ("bounded, fixed-size read buffer...") establishes the resource-bounding
  posture of this story's security layer. This ruling extends that posture to the rate-cap
  map; no new behavioral contract is needed because the cap-by-age fix is a transparent
  implementation detail of the rate-cap mechanism AC-018 specifies.

- **`lastSeen` bounding** → **AC-009/AC-010 + BC-2.03.001 Postcondition 5 (SEC-DW-07).**
  The lastSeen map's correctness is already specified by AC-008/AC-009/AC-010 and SEC-DW-07.
  The bounding behavior (eviction on cap) produces a cold-start transition indistinguishable
  from a router restart — already characterized by EC-006 and accepted by the Human Gate
  sign-off on SEC-DW-07 residuals (story v2.20 Human Gate disposition, rulings v1.11).
  The acceptance criterion for eviction-induced cold-start is already covered by AC-008's
  cold-start postcondition.

However, **a new SEC-DW clause is warranted** to document the bounding decisions
explicitly for future security reviewers. This ruling itself serves as that record.
Story-writer MUST add a reference to this ruling as an additional `inputs:` entry (see
Decision 5 below), which propagates it into the story's traceability chain.

No new VP is needed: the eviction behavior is a degraded-but-bounded form of the security
invariant VP-080 already asserts, with the degradation bounded and accepted per EC-006.

### New SEC-DW clause

**SEC-DW-10 (LOW).** Both per-`(SVTNID, NodeAddr)` process-lifetime maps (`relayRateCap.last`
and `RouterIngest.lastSeen`) MUST be bounded to prevent unbounded growth over the router's
lifetime. Each map MUST have a named constant upper bound of `maxRelayRateCapEntries =
65536` and `maxLastSeenEntries = 65536` respectively, matching `maxTrackedSources` in
`internal/admission/failure_counter.go`. Bounding strategy differs by map semantics:
`relayRateCap.last` uses prune-by-age (safe and lossless per VP-B); `lastSeen` uses
LRU-by-lowest-sequence eviction on insert (bounded replay exposure per EC-006 and the
Human Gate SEC-DW-07 residual acceptance).

This clause is carried in this ruling document; story-writer may optionally add an
SEC-DW-10 paragraph to the rulings.md or to the Security Consult Addendum, but is NOT
required to do so in this story's spec-freeze cycle.

---

## Decision 4 — Test Obligations (RED-first)

Both fixes require RED-first tests before implementation. Tests MUST be written before the
implementation lands and MUST fail prior to the fix.

### `relayRateCap.last` tests (file: `cmd/switchboard/relay_rate_cap_test.go`)

**Test 1 — Map size is bounded after N distinct keys + time advance**

```
TestRelayRateCap_MapBounded_AfterStaleEntries
```
- Construct a `relayRateCap` with injectable clock.
- Insert `maxRelayRateCapEntries + 1` distinct keys by calling `allow()` for each at
  time T.
- Advance clock to T + 2s (beyond `c.interval = 1s`) so all entries are stale.
- Call `allow()` for one more distinct key.
- Assert `len(c.last) <= maxRelayRateCapEntries` (prune triggered).
- Anti-vacuity: assert `len(c.last) > 0` (the map is not completely empty — the new
  call's entry was written).

**Test 2 — Prune is lossless: a key re-encountered after stale prune is re-allowed**

```
TestRelayRateCap_StalePrunedKey_ReAllowed
```
- Insert key K at time T (`allow()` returns true).
- Advance clock past interval.
- Trigger prune by inserting enough new keys to exceed threshold.
- Call `allow()` for K again — assert returns `true` (same result as if entry had
  never existed).

**Test 3 — Active keys within the interval are NOT pruned**

```
TestRelayRateCap_ActiveKeys_NotPruned
```
- Insert key K at time T. Immediately call `allow()` for K again at T+500ms.
- Assert K is still in `c.last` (not pruned — its timestamp is fresh).
- Assert `allow()` for K at T+500ms returns false (within interval — not a new allowance).

### `RouterIngest.lastSeen` tests (file: `internal/discovery/discovery_wire_test.go`)

**Test 1 — Map size is bounded after cap insertions**

```
TestRouterIngest_LastSeenMap_BoundedAtCap
```
- Construct a `RouterIngest` with a `MockRouter` pre-populated with
  `maxLastSeenEntries + 2` distinct admitted `(SVTNID, NodeAddr)` pairs, each with a
  valid HMAC key.
- Call `Ingest()` with a valid cold-start datagram for each of the `maxLastSeenEntries + 2`
  pairs (using strictly-increasing sequences to ensure cold-start acceptance).
- After all ingests, lock `ri.mu` (via test-internal accessor or white-box inspection)
  and assert `len(ri.lastSeen) <= maxLastSeenEntries`.

**Test 2 — Replay still rejected for a non-evicted key after cap**

```
TestRouterIngest_ReplayRejected_AfterCapEviction
```
- Insert `maxLastSeenEntries` entries (keys K1..KN with sequences 1000..1000+N-1, making K1
  the LRU with the lowest sequence).
- Insert a new key K_new — this triggers eviction of K1 (lowest sequence = 1000).
- Assert that a replay attempt for K2 (sequence = 1001, which should be above K2's
  watermark) is correctly discarded (K2 was not evicted).
- Assert `len(ri.lastSeen) <= maxLastSeenEntries`.

**Test 3 — Evicted key's cold-start is accepted**

```
TestRouterIngest_EvictedKey_ColdStartAccepted
```
- Insert `maxLastSeenEntries` entries. K1 has the lowest sequence.
- Insert K_new, evicting K1.
- Call `Ingest()` with a datagram for K1 carrying a LOW sequence (lower than K1's
  original watermark). Assert `decision.Accept == true` and `decision.Relay == true`
  (cold-start, AC-008 path).
- This test documents the security trade-off explicitly: it is NOT a regression test —
  it documents ACCEPTED behavior. Add a comment in the test body to that effect.

**Test 4 — LRU selection targets lowest sequence**

```
TestRouterIngest_LastSeen_LRU_EvictsLowestSequence
```
- Insert two entries: K_low with sequence 1, K_high with sequence 9999.
- Insert a third key K_new that triggers eviction (with a total map size of
  `maxLastSeenEntries`; set up so the cap is exactly hit).
- Assert that K_low (lowest sequence) was evicted (not K_high).

---

## Decision 5 — Story-Update Requirements

The following story-file changes are REQUIRED. Story-writer owns all of them.

### 1. New `inputs:` entry (MANDATORY)

Add this ruling to `S-BL.DISCOVERY-WIRE.md`'s `inputs:` list:

```yaml
- '.factory/decisions/S-BL.DISCOVERY-WIRE-map-bounding-ruling.md'  # v1.0 — BINDING.
  # Decisions for bounding relayRateCap.last (prune-by-age, maxRelayRateCapEntries=65536,
  # Decision 1) and RouterIngest.lastSeen (LRU-by-lowest-sequence cap,
  # maxLastSeenEntries=65536, Decision 2). New SEC-DW-10 clause introduced (Decision 3).
  # Story-writer must recompute input-hash after adding this entry.
```

After adding this entry, `input-hash` MUST be recomputed via `compute-input-hash --update`.
The `acceptance_criteria_count` stays 18. The `version` bumps by a minor increment (patch
form: 2.20 → 2.21 or equivalent per this story's versioning convention). `points` stay 8.
`status` stays `ready`.

### 2. File-Change List updates (MANDATORY)

Add two rows to the story's File-Change List table:

| File | Op | What |
|------|----|------|
| `cmd/switchboard/relay_rate_cap.go` | modify | Add `maxRelayRateCapEntries = 65536` constant; restructure `allow()` to compute `now` once; add amortized prune-by-age sweep when `len(c.last) > maxRelayRateCapEntries/2` (Decision 1, SEC-DW-10). |
| `cmd/switchboard/relay_rate_cap_test.go` | modify | Add three new RED-first tests: `TestRelayRateCap_MapBounded_AfterStaleEntries`, `TestRelayRateCap_StalePrunedKey_ReAllowed`, `TestRelayRateCap_ActiveKeys_NotPruned` (Decision 4). |
| `internal/discovery/discovery_wire.go` | modify | Add `maxLastSeenEntries = 65536` constant; add `evictLRULastSeen()` method; call it in `Ingest()` on cold-start path when at cap (Decision 2, SEC-DW-10). |
| `internal/discovery/discovery_wire_test.go` | modify | Add four new RED-first tests: `TestRouterIngest_LastSeenMap_BoundedAtCap`, `TestRouterIngest_ReplayRejected_AfterCapEviction`, `TestRouterIngest_EvictedKey_ColdStartAccepted`, `TestRouterIngest_LastSeen_LRU_EvictsLowestSequence` (Decision 4). |

### 3. Forward Obligation (OPTIONAL)

Story-writer MAY add a new Forward Obligation row for the SEC-DW-10 reference if the
rulings.md or Security Consult Addendum are to be updated in a future pass. This is
optional and non-blocking; the ruling doc itself is the authoritative record.

---

## Decision 6 — Execution Sequence

**Recommended sequence:**

1. **Story-writer** — Update `S-BL.DISCOVERY-WIRE.md`: add this ruling to `inputs:`,
   add File-Change List rows (Decision 5), bump version, recompute input-hash.
2. **Test-writer** — Write the RED-first tests (Decision 4, seven tests across two files)
   against the existing code. All seven MUST fail (map is currently unbounded). Commit.
3. **Implementer** — Implement the two fixes (Decision 1 for `relayRateCap.allow()`,
   Decision 2 for `RouterIngest.Ingest()` + `evictLRULastSeen()`). Tests must go GREEN.
   `go test -race` must pass. Commit.

Story-writer runs first because the input-hash recompute is a bookkeeping prerequisite for
the story's audit trail. Test-writer and implementer may run in sequence within the same
worktree (no parallelism advantage for two-file changes of this scope).

---

## Decision 7 — ARCH-08/Import-DAG Compliance

Both changes comply with ARCH-08 position-14 boundary (VP-J):

- `cmd/switchboard/relay_rate_cap.go` is in `cmd/switchboard` (position 5 in the DAG);
  the change adds only a constant and a prune loop within the existing package. No new
  imports.
- `internal/discovery/discovery_wire.go` is in `internal/discovery` (position 14); the
  change adds a constant, a helper method, and a call site within the existing package.
  No new imports. The `evictLRULastSeen()` helper uses only built-in Go map operations.

go.md rule 12 (lock discipline): both fixes execute inside the existing mutex scope
(`c.mu.Lock()` and `ri.mu.Lock()` respectively). No new lock acquisition points are
introduced. The constraint "prune must happen under the existing lock" (VP-F, VP-G, VP-I)
is satisfied by construction for both decisions.

go.md rule 11 (timestamps in UTC): not applicable — `relayRateCap` already uses
`c.now()` (injectable, defaults to `time.Now` not `time.Now().UTC()`); this is consistent
with the existing pattern in `tokenBucket.last` in the same file. No UTC change is required
or introduced.
