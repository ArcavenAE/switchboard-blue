---
artifact_id: RULING-W6TB-G-heartbeat-oracle
document_type: decision
level: ops
version: "1.0"
status: final
producer: product-owner
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-7.02]
closes_findings: [H-1]
referenced_by:
  - .factory/stories/S-7.02-session-discovery.md
  - .factory/decisions/RULING-W6TB-D-discovery-scope.md
  - .factory/decisions/wave-6-tranche-a-scope-rulings.md
---

# Ruling W6TB-G — S-7.02 Heartbeat Oracle: Tolerance vs. Exact-N

**Adjudicator:** product-owner
**Date:** 2026-07-01
**Trigger:** S-7.02 Pass-3 L1 finding H-1 (HIGH)

---

## Finding Summary

AC-001b (S-7.02 v1.2, story line 59) requires: "the test oracle can assert that N
ticks produce **exactly N** heartbeat events." It further explicitly states: "A
no-op tick body that a removed ticker would leave green is insufficient and
explicitly rejected."

The Pass-2 fix-burst widened the oracle in
`internal/discovery/discovery_test.go` lines 222–229 from `[N-1, N+1]` to
`[N/2, 2N]` to resolve CI flakiness (F-H-001) caused by Go `time.Ticker` jitter
and `ctx.Done()` races. Pass-3 now flags that the widened `[N/2, 2N]` bound drifts
from the AC-001b spec text, which was strengthened in RULING-W6TB-D v1.2 to
require observability via an injected tick source — precisely to avoid relying on
wall-clock `time.Ticker` for oracle precision.

The root tension: the existing test still drives the heartbeat through the real
`time.Ticker` path (measuring real-time tick counts within a wall-clock window)
rather than through an injected deterministic tick channel as AC-001b mandates.
The tolerance widening is a symptom of using the wrong test mechanism, not a
calibration of an acceptable measurement.

---

## Options Considered

### Option A — Refactor to injected tick source, assert exact-N

Replace the real-`time.Ticker` heartbeat test with a test that injects a
`func() <-chan time.Time` (or equivalent) via `Config`. The production `Run` loop
reads ticks from the injected factory when non-nil; falls back to
`time.NewTicker(cfg.HeartbeatInterval)` when nil. The test sends exactly N ticks
into the injected channel and asserts the `HeartbeatObserver` fires exactly N
times.

Pros: fully satisfies AC-001b verbatim; ticker-based tests disappear entirely
from the exact-N obligation; no tolerance needed.

Cons: requires a new `TickerFactory func(d time.Duration) <-chan time.Time` (or
`TickSource <-chan time.Time`) field on `Config`, expanding the production surface.
The production code path changes (reads from injected channel vs. creates its own
ticker). The `_IsIndependent` sibling test must be adapted to use the same injection
surface or left on the real-ticker path with a weaker `>= 1` oracle.

### Option B — Amend AC-001b to document tolerance

Rewrite AC-001b to: "the heartbeat timer fires at least `floor(N/2)` and no more
than `2N` ticks over a window of `N*interval`, and no ticks after `ctx.Done()` +
1 tick." Remove the "exactly N" language. Document that wall-clock jitter prevents
precise tick counting without deterministic injection.

Pros: no production surface change; existing test logic is preserved.

Cons: removes the "exactly N" safety guarantee. The original obligation from
RULING-W6TB-D was to make the heartbeat **verifiably not a no-op** — a `[N/2, 2N]`
range still permits a ticker that fires at half-rate or double-rate and passes.
More importantly, the "no-op removal test" failure mode identified in C-1 is not
fully addressed: a timer that fires at `N/2` still leaves the test green even if
the body does nothing observable. Option B weakens the spec to accommodate an
insufficient test mechanism.

### Option C — Widened tolerance test as integration sanity check PLUS new deterministic unit test for exact-N

Keep the existing `TestDiscovery_Advertise_PeriodicHeartbeat` with `[N/2, 2N]`
tolerance as a **CI flake-resistant integration sanity check** confirming the
heartbeat fires at approximately the right rate. Add a NEW test
`TestDiscovery_Advertise_PeriodicHeartbeat_ExactN` that injects a deterministic
`TickSource <-chan time.Time` via `Config`, sends exactly N ticks synchronously,
and asserts `HeartbeatObserver` fires exactly N times.

The exact-N test satisfies AC-001b verbatim. The widened-tolerance test retains
value as a real-ticker integration smoke check. The production surface addition
(`TickSource` or `TickerFactory` on `Config`) is shared between both tests.

---

## Decision: Option C

**Ruling: Option C — keep the widened tolerance test as integration sanity, add a
deterministic exact-N unit test. Document in this ruling. Bump S-7.02 to v1.3.**

### Rationale

**Option B is rejected.** Amending AC-001b to accept `[N/2, 2N]` is a spec
retreat that codifies an insufficient oracle. The original purpose of
RULING-W6TB-D AC-001b strengthening was to make the no-op-ticker failure mode
detectable. A `[N/2, 2N]` wall-clock range does not reliably detect a ticker
body that has been silently removed: if the ticker fires 0 times, the test fails;
but if the body is a no-op and the ticker fires at `N/2` due to a scheduling
anomaly, the test still passes. This is the very scenario AC-001b "explicitly
rejects." Option B is ruled out.

**Option A is rejected as the sole approach.** Replacing the wall-clock ticker
test entirely with an injected-tick test removes a genuine integration-level check
that the `time.Ticker` path works in production. The `Run()` goroutine's behavior
under real scheduler pressure is distinct from its behavior under a synchronous
test harness. Retaining the wall-clock test at a reasonable tolerance preserves
that integration signal.

**Option C delivers both properties:**

1. **Exact-N determinism (AC-001b satisfaction):** The new
   `TestDiscovery_Advertise_PeriodicHeartbeat_ExactN` test injects exactly N ticks
   via a buffered `chan time.Time` assigned to `Config.TickSource`. The
   `HeartbeatObserver` counter is checked after each tick send, allowing the test
   to assert exact N without any wall-clock sensitivity. A ticker body removed from
   `Run()` will cause this test to fail with count = 0 regardless of scheduling.

2. **CI flake resistance (integration smoke check):** The existing
   `TestDiscovery_Advertise_PeriodicHeartbeat` with `[N/2, 2N]` tolerance runs
   against the real `time.Ticker` path (when `Config.TickSource == nil`, `Run`
   creates its own ticker). It detects catastrophic failures (heartbeat never
   fires, always fires 0 times) while remaining tolerant of Go scheduler jitter.
   This is appropriate scope for an integration test.

**Production surface delta is minimal.** One optional field `TickSource
<-chan time.Time` on `Config`. When nil, `Run` creates a `time.NewTicker` as
before. When non-nil (test-only), `Run` selects on `TickSource` instead. This
is the narrowest possible seam — no factory function needed, no interface type,
no callback indirection.

### Production Code Change Specification

In `internal/discovery/discovery.go`:

1. Add to `Config`:
   ```go
   // TickSource, if non-nil, is used instead of time.NewTicker for the
   // heartbeat timer. Injected in tests to provide deterministic tick delivery.
   // Production callers MUST leave this nil.
   TickSource <-chan time.Time
   ```

2. In `Run()`, replace:
   ```go
   ticker := time.NewTicker(d.cfg.HeartbeatInterval)
   defer ticker.Stop()
   // ... select { case <-ticker.C: ... }
   ```
   With:
   ```go
   var tickCh <-chan time.Time
   var ticker *time.Ticker
   if d.cfg.TickSource != nil {
       tickCh = d.cfg.TickSource
   } else {
       ticker = time.NewTicker(d.cfg.HeartbeatInterval)
       defer ticker.Stop()
       tickCh = ticker.C
   }
   // ... select { case <-tickCh: ... }
   ```

### Test Change Specification

In `internal/discovery/discovery_test.go`:

1. **Keep** `TestDiscovery_Advertise_PeriodicHeartbeat` with `[N/2, 2N]` tolerance.
   Update its comment to: "Integration sanity check using real `time.Ticker`. Wide
   tolerance [N/2, 2N] is intentional: wall-clock jitter prevents exact counting.
   Exact-N oracle is in `TestDiscovery_Advertise_PeriodicHeartbeat_ExactN`
   (RULING-W6TB-G). This test detects catastrophic failures only (heartbeat never
   fires)."

2. **Add** `TestDiscovery_Advertise_PeriodicHeartbeat_ExactN`:
   ```go
   // TestDiscovery_Advertise_PeriodicHeartbeat_ExactN verifies AC-001b exactly:
   // N injected ticks produce exactly N HeartbeatObserver calls.
   //
   // Uses Config.TickSource for deterministic tick delivery — no wall-clock
   // sensitivity. A removed ticker body causes count == 0 and test failure
   // (RULING-W6TB-G: the no-op-removal oracle).
   //
   // BC-2.03.001 PC-4; AC-001b (S-7.02 v1.3).
   func TestDiscovery_Advertise_PeriodicHeartbeat_ExactN(t *testing.T) {
       t.Parallel()
       const N = 5
       var count int
       var mu sync.Mutex

       tickCh := make(chan time.Time, N)
       cfg := discovery.Config{
           LocalNodeAddr:     nodeA1,
           LocalSVTNID:       svtnA,
           Router:            newTestRouter(t),
           HeartbeatInterval: time.Second, // irrelevant; TickSource overrides
           HeartbeatObserver: func() {
               mu.Lock()
               count++
               mu.Unlock()
           },
           TickSource: tickCh,
       }
       d := discovery.New(cfg)

       ctx, cancel := context.WithCancel(context.Background())
       t.Cleanup(cancel)

       runDone := make(chan error, 1)
       go func() {
           runDone <- d.Run(ctx)
       }()

       // Send exactly N ticks and verify each one fires the observer.
       now := time.Now().UTC()
       for i := range N {
           tickCh <- now
           // Poll with a short deadline to detect a stuck observer.
           deadline := time.Now().Add(100 * time.Millisecond)
           for {
               mu.Lock()
               got := count
               mu.Unlock()
               if got == i+1 {
                   break
               }
               if time.Now().After(deadline) {
                   t.Fatalf("tick %d: HeartbeatObserver not called within 100ms (got %d, want %d)", i+1, got, i+1)
               }
               runtime.Gosched()
           }
       }

       cancel()
       if err := <-runDone; err != nil && !errors.Is(err, context.Canceled) {
           t.Fatalf("Run: unexpected error: %v", err)
       }

       mu.Lock()
       got := count
       mu.Unlock()
       if got != N {
           t.Errorf("HeartbeatObserver called %d times after %d ticks, want exactly %d (BC-2.03.001 PC-4 exact-N oracle)", got, N, N)
       }
   }
   ```

Note: `runtime.Gosched()` requires adding `"runtime"` to the import block if not
already present.

---

## S-7.02 Story Delta (v1.2 → v1.3)

### AC-001b amendment

Replace the AC-001b test specification:
```
- **Test:** `TestDiscovery_Advertise_PeriodicHeartbeat` — inject tick source via `Config`; advance N ticks; assert counter/channel records N heartbeat events.
```
With:
```
- **Tests:**
  - `TestDiscovery_Advertise_PeriodicHeartbeat_ExactN` (primary oracle) — inject `Config.TickSource` (buffered `chan time.Time`); send exactly N ticks; assert observer fires exactly N times. A removed ticker body causes count == 0 and MUST fail (RULING-W6TB-G no-op-removal oracle).
  - `TestDiscovery_Advertise_PeriodicHeartbeat` (integration sanity) — real `time.Ticker`; wide tolerance `[N/2, 2N]`; detects catastrophic failures only (RULING-W6TB-G).
```

### Config surface addition

Add to the File Structure table note column for `internal/discovery/discovery.go`:

> Add `TickSource <-chan time.Time` to `Config` — test-only tick injection seam;
> `Run()` selects on this when non-nil instead of creating `time.NewTicker`.
> (RULING-W6TB-G)

### Frontmatter delta

| Field | v1.2 | v1.3 |
|-------|------|------|
| `version` | `"1.2"` | `"1.3"` |
| `changed_by_rulings` | `[RULING-W6TB-D]` | `[RULING-W6TB-D, RULING-W6TB-G]` |

### Changelog addition

```
| v1.3 | 2026-07-01 | product-owner | RULING-W6TB-G: AC-001b oracle split into
ExactN (primary, deterministic TickSource injection) + PeriodicHeartbeat (sanity,
real-ticker [N/2, 2N] tolerance). Config.TickSource seam added to discovery.go.
Resolves H-1 oracle-vs-spec gap without dropping the integration smoke check. |
```

---

## Downstream Dispatch Table

| Artifact | Change | Agent | When |
|----------|--------|-------|------|
| `.factory/stories/S-7.02-session-discovery.md` | AC-001b test spec + Config surface note + v1.2→v1.3 + changelog | story-writer | Same burst as ruling |
| `wave-6-tranche-a-scope-rulings.md` | Add §11 changelog entry for RULING-W6TB-G | spec-steward | Same burst as ruling |
| `internal/discovery/discovery.go` (worktree) | Add `TickSource` to Config; update `Run()` ticker selection | implementer | S-7.02 fix-burst |
| `internal/discovery/discovery_test.go` (worktree) | Add `TestDiscovery_Advertise_PeriodicHeartbeat_ExactN`; update existing test comment | implementer | S-7.02 fix-burst |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | product-owner | Option C adopted. Exact-N oracle satisfied via injected `TickSource`; real-ticker test retained as integration sanity with `[N/2, 2N]` tolerance. Config.TickSource is the minimal production surface seam. Rationale: Option B weakens the no-op-removal detection guarantee that RULING-W6TB-D established; Option A loses integration coverage of the real `time.Ticker` path. Option C delivers both oracle strength and integration breadth at minimal cost (one optional Config field). |
