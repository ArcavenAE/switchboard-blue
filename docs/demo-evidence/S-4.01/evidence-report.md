# Demo Evidence Report — S-4.01

**Story:** S-4.01 — Per-Path RTT/Loss Tracking and Duplicate-and-Race Dispatch
**Packages:** `internal/paths`, `internal/multipath`
**Product type:** Library / pure-core Go packages (no runnable binary surface)
**Recording method:** Test-transcript captures (precedent: S-W3.04)
**Worktree:** `.worktrees/S-4.01`
**Date:** 2026-06-28

---

## AC Coverage Map

| AC | BC Trace | Criterion | Test(s) | Transcript | Result |
|----|----------|-----------|---------|------------|--------|
| AC-001 | BC-2.02.003 postcondition 1 | `PathScore(rtt_ms, loss_pct)` ranks paths deterministically: lower RTT and loss produce a higher score; the ranking is transitive | `TestBC_2_02_003_PathScore_Transitive`, `TestBC_2_02_003_PathScore_LowerRTTLowerScore`, `TestBC_2_02_003_PathScore_HigherLossRaisesScore`, `TestBC_2_02_003_PathScore_ZeroLossPureRTT`, `TestBC_2_02_003_PathScore_Formula`, `TestBC_2_02_003_PathTracker_ScoreDelegates`, `TestBC_2_02_003_PathScore_PropertyTransitive_Manual` | `AC-001-pathscore-transitive.txt` | PASS |
| AC-002 | BC-2.02.003 postcondition 2 | `PathTracker.OnProbe(arrival_rtt_ms, loss_event)` updates EWMA RTT and loss. After 3 probe arrivals, score converges | `TestBC_2_02_003_PathTracker_EWMAConvergence`, `TestBC_2_02_003_PathTracker_EWMAConvergence_ThreeProbes` | `AC-002-ewma-convergence.txt` | PASS |
| AC-003 | BC-2.02.001 postcondition 1 | `Multipath.Send(frame, path_set)` dispatches frame on the two highest-scoring paths simultaneously | `TestBC_2_02_001_Send_TwoFastestPaths`, `TestBC_2_02_001_Send_ThreePathsSelectLowest`, `TestBC_2_02_001_Send_SinglePathFallback`, `TestBC_2_02_001_Send_AtMostTwoPaths`, `TestBC_2_02_001_Send_IdenticalBytesOnBothPaths` | `AC-003-send-two-fastest-paths.txt` | PASS |
| AC-004 | BC-2.02.002 postcondition 1 | `Multipath.Receive(frame)` delivers first-arriving copy (nil) and returns `ErrDuplicate` for subsequent copies with the same checksum | `TestBC_2_02_002_Receive_FirstCopyDelivered`, `TestBC_2_02_002_Receive_DuplicateReturnsErr` | `AC-004-first-copy-delivered.txt` | PASS |
| AC-005 | BC-2.02.002 postcondition 2 | Duplicate discards are silent: no error surfaced to session layer for discarded duplicate | `TestBC_2_02_002_Receive_DuplicateDiscardSilent` | `AC-005-duplicate-discard-silent.txt` | PASS |
| AC-006 | BC-2.02.009 postcondition 1 | `DropCache` never exceeds configured capacity (LRU eviction); checksum-based lookup is O(1) | `TestBC_2_02_009_DropCache_BoundedCapacity`, `TestBC_2_02_009_DropCache_LRUEvictsOldest`, `TestBC_2_02_009_DropCache_Len`, `TestBC_2_02_009_DropCache_BoundedCapacity_PropertySweep` | `AC-006-dropcache-bounded-capacity.txt` | PASS |
| AC-007 | BC-2.02.009 postcondition 2 | `DropCache.Hits()` returns cumulative hit count; increments on every `AddIfAbsent`/`Add` call where key is already present | `TestBC_2_02_009_DropCache_HitCounterIncremented`, `TestBC_2_02_009_DropCache_HitCounterConcurrent` | `AC-007-dropcache-hit-counter.txt` | PASS |

**Overall: 7/7 ACs — all PASS**

---

## Race Detector Evidence

The full test suite (all 7 ACs plus supporting tests) was run under `go test -race` with zero data races detected.

- Transcript: `race-detector-full-suite.txt`
- Concurrency tests covered:
  - `TestBC_2_02_003_PathTracker_ConcurrentOnProbeScore` — PathTracker concurrent safety (F-004)
  - `TestBC_2_02_002_Receive_ConcurrentFirstArrivalWins` — TOCTOU first-arrival-wins invariant (F-005, DI-009)
  - `TestBC_2_02_009_DropCache_ConcurrentAddContains` — DropCache concurrent safety (F-004)
  - `TestBC_2_02_001_Send_ConcurrentWithUpdatePaths` — Send/UpdatePaths lock contention (F-M3)
  - `TestBC_2_02_009_DropCache_HitCounterConcurrent` — Hits() counter race safety (F-H2)

---

## Deferrals (per S-4.01 story §Deferrals)

| Item | Status | Owner |
|------|--------|-------|
| BC-2.02.009 router-side `OnFrameArrival` / forwarding wiring | Deferred to S-4.04 | S-4.04 |
| BC-2.02.009 EC-005 collision-event logging | Deferred to S-4.04 | S-4.04 |
| BC-2.02.001 EC-003 queue-with-timeout + E-NET-002 | Out of scope S-4.01; future wave-4+ story | TBD |
| VP-040 end-to-end failover-recovery < 2s assertion (e2e, `internal/testenv`) | Deferred to integration harness wave-gate | integration harness |
| VP-054 end-to-end proof (requires `internal/testenv`) | Deferred to integration harness wave-gate | integration harness |

No AC is blocked by these deferrals — they are out-of-scope items for the router-wiring layer (S-4.04) and the integration harness, not for the pure-core `internal/paths` / `internal/multipath` modules delivered here.

---

## Evidence Files

| File | Contents |
|------|----------|
| `AC-001-pathscore-transitive.txt` | PathScore determinism and transitivity test transcripts |
| `AC-002-ewma-convergence.txt` | EWMA convergence test transcripts |
| `AC-003-send-two-fastest-paths.txt` | Duplicate-and-race Send dispatch test transcripts |
| `AC-004-first-copy-delivered.txt` | First-copy delivery and ErrDuplicate test transcripts |
| `AC-005-duplicate-discard-silent.txt` | Silent duplicate discard test transcript |
| `AC-006-dropcache-bounded-capacity.txt` | DropCache LRU bounded-capacity test transcripts |
| `AC-007-dropcache-hit-counter.txt` | DropCache hit counter test transcripts |
| `race-detector-full-suite.txt` | Full suite under `go test -race`, zero races detected |
