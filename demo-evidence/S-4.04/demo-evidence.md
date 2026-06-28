# Demo Evidence — S-4.04: Split-Horizon Loop Prevention + DropCache Wiring

## Header

| Field | Value |
|-------|-------|
| Story | S-4.04 |
| HEAD SHA | 24c4378487cdb24f783920a99a1117b60949d7c2 |
| Branch | feat/S-4.04-split-horizon-drop-cache |
| Package | `github.com/arcavenae/switchboard/internal/routing` |
| Overall race-detector pass | `go test -race ./internal/routing/...` → `ok` (1.479s) |
| Evidence captured | 2026-06-28 |

Overall suite confirmation:

```
ok  	github.com/arcavenae/switchboard/internal/routing	1.479s
```

---

## AC-001 — No Forward Toward Arrival Interface

**AC text:** `SplitHorizon.Forward(frame, arrival_interface_id, interface_set)` does not forward the frame on the `arrival_interface_id` interface.
**Traces to:** BC-2.02.008 postcondition 1
**Proving test:** `TestSplitHorizon_NoForwardTowardArrivalInterface`

```
=== RUN   TestSplitHorizon_NoForwardTowardArrivalInterface
=== PAUSE TestSplitHorizon_NoForwardTowardArrivalInterface
=== CONT  TestSplitHorizon_NoForwardTowardArrivalInterface
--- PASS: TestSplitHorizon_NoForwardTowardArrivalInterface (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.444s
```

**What it demonstrates:** With arrival interface A and set {A, B, C}, verifies ForwardFunc is never called with interface A — split-horizon exclusion is enforced.

---

## AC-002 — Forward on All Other Interfaces

**AC text:** `SplitHorizon.Forward` forwards the frame on all other interfaces in the interface set.
**Traces to:** BC-2.02.008 postcondition 2
**Proving test:** `TestSplitHorizon_ForwardOnAllOtherInterfaces`

```
=== RUN   TestSplitHorizon_ForwardOnAllOtherInterfaces
=== PAUSE TestSplitHorizon_ForwardOnAllOtherInterfaces
=== CONT  TestSplitHorizon_ForwardOnAllOtherInterfaces
--- PASS: TestSplitHorizon_ForwardOnAllOtherInterfaces (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.252s
```

**What it demonstrates:** With arrival interface 10 and set {10, 20, 30}, verifies ForwardFunc is called for both 20 and 30 — all eligible (non-arrival) interfaces receive the frame.

---

## AC-003 — Channel Header Opaque (Fuzz)

**AC text:** The router code never parses channel header payload; injecting arbitrary bytes into the channel header section does not affect routing decisions.
**Traces to:** BC-2.02.008 invariant 1, VP-015
**Proving test:** `FuzzSplitHorizon_ChannelHeaderOpaque` (fuzz mode, 5s / 1,463,941 executions)

```
=== RUN   FuzzSplitHorizon_ChannelHeaderOpaque
fuzz: elapsed: 0s, gathering baseline coverage: 0/4 completed
fuzz: elapsed: 0s, gathering baseline coverage: 4/4 completed, now fuzzing with 8 workers
fuzz: elapsed: 3s, execs: 869610 (289796/sec), new interesting: 1 (total: 5)
fuzz: elapsed: 5s, execs: 1463941 (283173/sec), new interesting: 1 (total: 5)
--- PASS: FuzzSplitHorizon_ChannelHeaderOpaque (5.10s)
=== NAME
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	5.397s
```

**What it demonstrates:** After 1.46M executions with arbitrary `(outerHdr, channelHeaderBytes)` inputs, the routing decision (arrival interface excluded, all others forwarded) was never influenced by channel header byte content — VP-015 invariant holds.

---

## AC-004 — DropCache Wiring (Compound Key)

**AC text:** The router `OnFrameArrival` path consults the compound-key `(checksum, arrival_interface_id)` `DropCache` before forwarding. Cache miss → key added and frame forwarded. Cache hit → frame silently discarded.
**Traces to:** BC-2.02.009 postcondition 1
**Proving test:** `TestBC_2_02_009_Router_DropCacheWiring` (4 sub-tests)

```
=== RUN   TestBC_2_02_009_Router_DropCacheWiring
=== PAUSE TestBC_2_02_009_Router_DropCacheWiring
=== CONT  TestBC_2_02_009_Router_DropCacheWiring
=== RUN   TestBC_2_02_009_Router_DropCacheWiring/cache_miss_first_arrival_returns_nil
=== PAUSE TestBC_2_02_009_Router_DropCacheWiring/cache_miss_first_arrival_returns_nil
=== RUN   TestBC_2_02_009_Router_DropCacheWiring/cache_hit_second_arrival_returns_ErrDropCacheHit
=== PAUSE TestBC_2_02_009_Router_DropCacheWiring/cache_hit_second_arrival_returns_ErrDropCacheHit
=== RUN   TestBC_2_02_009_Router_DropCacheWiring/compound_key_same_checksum_different_interface_is_miss
=== PAUSE TestBC_2_02_009_Router_DropCacheWiring/compound_key_same_checksum_different_interface_is_miss
=== RUN   TestBC_2_02_009_Router_DropCacheWiring/absent_key_gets_added_to_cache
=== PAUSE TestBC_2_02_009_Router_DropCacheWiring/absent_key_gets_added_to_cache
=== CONT  TestBC_2_02_009_Router_DropCacheWiring/cache_miss_first_arrival_returns_nil
=== CONT  TestBC_2_02_009_Router_DropCacheWiring/compound_key_same_checksum_different_interface_is_miss
=== CONT  TestBC_2_02_009_Router_DropCacheWiring/absent_key_gets_added_to_cache
=== CONT  TestBC_2_02_009_Router_DropCacheWiring/cache_hit_second_arrival_returns_ErrDropCacheHit
--- PASS: TestBC_2_02_009_Router_DropCacheWiring (0.00s)
    --- PASS: TestBC_2_02_009_Router_DropCacheWiring/cache_miss_first_arrival_returns_nil (0.00s)
    --- PASS: TestBC_2_02_009_Router_DropCacheWiring/compound_key_same_checksum_different_interface_is_miss (0.00s)
    --- PASS: TestBC_2_02_009_Router_DropCacheWiring/cache_hit_second_arrival_returns_ErrDropCacheHit (0.00s)
    --- PASS: TestBC_2_02_009_Router_DropCacheWiring/absent_key_gets_added_to_cache (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.285s
```

**What it demonstrates:** Four sub-tests cover: first arrival returns nil (miss), second arrival returns `ErrDropCacheHit` (hit), same checksum on a different interface is a miss (compound key, not checksum-only per ARCH-INDEX F-006), and after a miss the key is immediately added so the next arrival is a hit.

---

## AC-005 — Collision-Event Logging: Rate-Limited + Bounded Memory

**AC text:** Drop-cache hits are logged as potential collision events via injected logger. Logging is: (a) bounded memory — tracking structure capped at DropCache capacity; (b) bounded aggregate log volume — N distinct-key hits produce far fewer than N log lines; (c) best-effort first-occurrence observability; (d) per-key flood still bounded.
**Traces to:** BC-2.02.009 postcondition 2 and EC-002
**Proving tests:**
- `TestBC_2_02_009_Router_CollisionLogRateLimited` (per-key rate limit: 3 sub-tests)
- `TestBC_2_02_009_Router_CollisionLog_DistinctKeyFlood_Bounded` (K=20,000 distinct-key flood)

```
=== RUN   TestBC_2_02_009_Router_CollisionLog_DistinctKeyFlood_Bounded
--- PASS: TestBC_2_02_009_Router_CollisionLog_DistinctKeyFlood_Bounded (0.01s)
=== CONT  TestBC_2_02_009_Router_CollisionLogRateLimited
=== RUN   TestBC_2_02_009_Router_CollisionLogRateLimited/first_hit_produces_at_least_one_log_line
=== PAUSE TestBC_2_02_009_Router_CollisionLogRateLimited/first_hit_produces_at_least_one_log_line
=== RUN   TestBC_2_02_009_Router_CollisionLogRateLimited/no_logger_injected_does_not_panic
=== PAUSE TestBC_2_02_009_Router_CollisionLogRateLimited/no_logger_injected_does_not_panic
=== RUN   TestBC_2_02_009_Router_CollisionLogRateLimited/flood_of_hits_produces_bounded_log_lines
=== PAUSE TestBC_2_02_009_Router_CollisionLogRateLimited/flood_of_hits_produces_bounded_log_lines
=== CONT  TestBC_2_02_009_Router_CollisionLogRateLimited/first_hit_produces_at_least_one_log_line
=== CONT  TestBC_2_02_009_Router_CollisionLogRateLimited/flood_of_hits_produces_bounded_log_lines
=== CONT  TestBC_2_02_009_Router_CollisionLogRateLimited/no_logger_injected_does_not_panic
--- PASS: TestBC_2_02_009_Router_CollisionLogRateLimited (0.00s)
    --- PASS: TestBC_2_02_009_Router_CollisionLogRateLimited/first_hit_produces_at_least_one_log_line (0.00s)
    --- PASS: TestBC_2_02_009_Router_CollisionLogRateLimited/no_logger_injected_does_not_panic (0.00s)
    --- PASS: TestBC_2_02_009_Router_CollisionLogRateLimited/flood_of_hits_produces_bounded_log_lines (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.398s
```

**What it demonstrates:**
- `first_hit_produces_at_least_one_log_line`: first drop-cache hit on a key logs at least one line (best-effort observability preserved).
- `no_logger_injected_does_not_panic`: default nopLogger prevents nil-dereference on cache hit.
- `flood_of_hits_produces_bounded_log_lines`: 1,000 rapid identical-key hits produce ≤ max(2, 10) = 10 log lines (CWE-779 log-spam mitigation).
- `DistinctKeyFlood_Bounded`: K=20,000 distinct-key hits produce ≤ max(10, 400) = 400 aggregate log lines AND tracking structure holds ≤ DefaultDropCacheSize=10,000 keys (CWE-401/400 memory bound).

---

## AC-006 — End-to-End Composition: DropCache + SplitHorizon

**AC text:** `OnFrameArrival` composes DropCache suppression and split-horizon forwarding. DropCache MISS: add key, then call `SplitHorizon.Forward` (frame forwarded on all non-arrival interfaces). DropCache HIT: discard without calling `SplitHorizon.Forward`. `SplitHorizon.Forward` has at least one non-test caller.
**Traces to:** BC-2.02.009 postcondition 1 + BC-2.02.008 postcondition 2
**Proving test:** `TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss` (2 sub-tests)

```
=== RUN   TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss
=== PAUSE TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss
=== CONT  TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss
=== RUN   TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/miss_forwards_on_non_arrival_interfaces
=== PAUSE TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/miss_forwards_on_non_arrival_interfaces
=== RUN   TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/hit_discards_without_forwarding
=== PAUSE TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/hit_discards_without_forwarding
=== CONT  TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/miss_forwards_on_non_arrival_interfaces
=== CONT  TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/hit_discards_without_forwarding
--- PASS: TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss (0.00s)
    --- PASS: TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/miss_forwards_on_non_arrival_interfaces (0.00s)
    --- PASS: TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss/hit_discards_without_forwarding (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.253s
```

**What it demonstrates:**
- `miss_forwards_on_non_arrival_interfaces`: on a DropCache miss, `OnFrameArrival` calls `SplitHorizon.Forward` and ForwardFunc is invoked for exactly the 2 non-arrival interfaces (ifaceB=2, ifaceC=3), not the arrival interface (1).
- `hit_discards_without_forwarding`: on a DropCache hit, `OnFrameArrival` returns `ErrDropCacheHit` and ForwardFunc is never called — no forwarding occurs.

---

## AC-007 — E-FWD-001 Log on All-Paths Split-Horizon Drop

**AC text:** When `SplitHorizon.Forward` determines that the only interface in the set is the arrival interface (all eligible paths blocked), `OnFrameArrival` MUST drop the frame AND emit exactly one E-FWD-001 log event via its injected logger, including the arrival interface ID and frame checksum. No forward call occurs.
**Traces to:** BC-2.02.008 postcondition 3
**Proving test:** `TestOnFrameArrival_AllPathsSplitHorizon_LogsEFWD001`

```
=== RUN   TestOnFrameArrival_AllPathsSplitHorizon_LogsEFWD001
=== PAUSE TestOnFrameArrival_AllPathsSplitHorizon_LogsEFWD001
=== CONT  TestOnFrameArrival_AllPathsSplitHorizon_LogsEFWD001
--- PASS: TestOnFrameArrival_AllPathsSplitHorizon_LogsEFWD001 (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/routing	0.286s
```

**What it demonstrates:** With interfaceSet = {arrivalIface=17} only, `OnFrameArrival` returns `ErrAllPathsSplitHorizon` (assert i), the logger receives exactly one line containing `"E-FWD-001"`, the checksum `0x...` of the frame bytes, and the interface ID `17` (assert ii), and the ForwardFunc is never called (assert iii). This closes BC-2.02.008 PC-3 / conformance gap F-L1-001 from story v1.5.

---

## Summary

| AC | Test(s) | Status |
|----|---------|--------|
| AC-001 | `TestSplitHorizon_NoForwardTowardArrivalInterface` | PASS |
| AC-002 | `TestSplitHorizon_ForwardOnAllOtherInterfaces` | PASS |
| AC-003 | `FuzzSplitHorizon_ChannelHeaderOpaque` (1.46M executions, 5s) | PASS |
| AC-004 | `TestBC_2_02_009_Router_DropCacheWiring` (4 sub-tests) | PASS |
| AC-005 | `TestBC_2_02_009_Router_CollisionLogRateLimited` (3 sub-tests) + `TestBC_2_02_009_Router_CollisionLog_DistinctKeyFlood_Bounded` | PASS |
| AC-006 | `TestOnFrameArrival_ForwardsViaSplitHorizon_AfterDropCacheMiss` (2 sub-tests) | PASS |
| AC-007 | `TestOnFrameArrival_AllPathsSplitHorizon_LogsEFWD001` | PASS |
| Race detector | `go test -race ./internal/routing/...` | PASS (1.479s) |
