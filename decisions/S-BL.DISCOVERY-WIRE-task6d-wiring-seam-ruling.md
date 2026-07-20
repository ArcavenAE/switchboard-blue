---
artifact_id: S-BL.DISCOVERY-WIRE-task6d-wiring-seam-ruling
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
  - BC-2.01.008
related_docs:
  - decisions/S-BL.DISCOVERY-WIRE-fanout-resolution-ruling.md
  - stories/S-BL.DISCOVERY-WIRE.md
---

# Ruling: S-BL.DISCOVERY-WIRE Task 6d — Decision-Threading Seam for Hop-1 Ingest → Hop-2 Fan-Out

All factual claims are grep/read-verified against worktree `feature/S-BL.DISCOVERY-WIRE-FANOUT`
HEAD `4b82535c4a12318370b7a4fe13931abbb324c347`. File:symbol anchors cited per TD-031.

This ruling resolves the ONE open design seam blocking Task 6d of `S-BL.DISCOVERY-WIRE`: how the
`RouterIngestDecision` produced inside `wireDiscoveryListener`'s ingest loop reaches the rate-cap
check and `relayDispatch` call. It does NOT modify any story file, BC, STATE.md, or STORY-INDEX —
those downstream edits are flagged at the end and owned by story-writer/PO.

The parent ruling (`decisions/S-BL.DISCOVERY-WIRE-fanout-resolution-ruling.md` v1.0) resolved
the fan-out primitive (`Router.InterfacesForSVTN`), the dispatch closure seam (inline in
`runRouter`), and the rate-cap AC. It left the Phase-2 loop shape illustrative (lifecycle-only)
and explicitly flagged the decision-threading seam as the Task 6d gap. This ruling is the
completion of that gap.

---

## Verified Premises

| # | Premise | File:Symbol | Evidence |
|---|---|---|---|
| VP-1 | `wireDiscoveryListener` current signature is 5-arg: `(ctx context.Context, wg *sync.WaitGroup, svtnID [16]byte, ri *discovery.RouterIngest, w io.Writer) error` | `cmd/switchboard/discovery_wire.go`, `wireDiscoveryListener` | Direct read, line 91 |
| VP-2 | The ingest loop DISCARDS the `RouterIngestDecision`: `if _, ingestErr := ri.Ingest(raw); ingestErr != nil { continue }` — the `_` blank-identifier discard is the gap this ruling resolves | `cmd/switchboard/discovery_wire.go`, `wireDiscoveryListener` ingest loop | Direct read, line 122 |
| VP-3 | `RouterIngestDecision.Relay` is the field that distinguishes "accepted AND passes replay gate" (relay) from "accepted but stale sequence" (no relay); `RouterIngestDecision.Accept` records HMAC pass/fail | `internal/discovery/discovery_wire.go`, `RouterIngestDecision` | Direct read, lines 56-77 |
| VP-4 | `relayDispatch(router *routing.Router, sendMap *sync.Map, decision discovery.RouterIngestDecision)` is shipped in Task 6b at `cmd/switchboard/discovery_relay_wire.go`; explicitly documented as NOT checking `decision.Relay` ("the caller gates") | `cmd/switchboard/discovery_relay_wire.go`, `relayDispatch` | Direct read, lines 117-148 (doc comment line 128: "relayDispatch does NOT check decision.Relay") |
| VP-5 | `callWireDiscoveryListenerRecovered` in the existing test file calls `wireDiscoveryListener(ctx, wg, svtnID, ri, w)` — the current 5-arg form; a signature change ripples here | `cmd/switchboard/discovery_wire_test.go`, `callWireDiscoveryListenerRecovered` | Direct read, line 44 |
| VP-6 | `wireDiscoveryListener` BLOCKS until ctx cancel; caller does `wg.Add(1)` before `go wireDiscoveryListener(...)` per ARCH-01 §Goroutine WaitGroup Contract (F-DWIP3-001); `defer wg.Done()` is the only Done call site | `cmd/switchboard/discovery_wire.go`, `wireDiscoveryListener` | Direct read, lines 70-83, 92 |
| VP-7 | `sendMap sync.Map // routing.InterfaceID -> *nodeConn` is LOCAL to `runRouter`; `router` is a local variable; neither is a package-level var | `cmd/switchboard/mgmt_wire.go`, `runRouter` | Direct read, line 592 comment "Per-node send map (Q-SEAM)"; `router := buildRouter(...)` line ~538 |
| VP-8 | `ingressCtx, ingressCancel := context.WithCancel(context.WithoutCancel(ctx))` is declared at line ~584 in `runRouter`, BEFORE `sendMap` and `writerWG` | `cmd/switchboard/mgmt_wire.go`, `runRouter` | Direct read, lines 584-593 |
| VP-9 | `writerWG sync.WaitGroup` tracks ONLY per-connection writer goroutines started from `onAccept`; its shutdown-block comment (line ~1082-1088) explicitly states "no writerWG.Add(1) can occur after this point (grounded on dataWG.Wait having joined Serve's accept loop)" — mixing discovery listeners into `writerWG` would violate this invariant's established semantics | `cmd/switchboard/mgmt_wire.go`, `runRouter` shutdown block | Direct read, lines 1081-1089 |
| VP-10 | `routerKS.AllSVTNEntries()` returns `map[[16]byte][]AdmittedKey` — iterating it with `for svtnID := range` yields `[16]byte` SVTN keys; it is populated by `loadSnapshotFromFile` (Phase b1) and by `wireAdmissionSyncHandlers` push (Phase c3) | `internal/admission/admission.go`, `AllSVTNEntries`; `cmd/switchboard/mgmt_wire.go`, `runRouter` phases b1/c3 | Direct read, admission.go lines 584-600; mgmt_wire.go lines 527/557 |
| VP-11 | `RouterIngest` is declared "safe for concurrent use" and uses `sync.Mutex` on `lastSeen` and a mutex-guarded `tokenBucket`; one shared instance across multiple goroutines calling `Ingest` is correct | `internal/discovery/discovery_wire.go`, `RouterIngest` | Direct read, lines 197-205 ("All exported methods are safe for concurrent use") |
| VP-12 | `routerLogger` is available at the point `router` is built (~line 538) and is used as `routerLogger.Log` throughout `runRouter`; it is appropriate for `RouterIngestConfig.Logger` | `cmd/switchboard/mgmt_wire.go`, `runRouter` | Direct read, lines 520-538 |
| VP-13 | `ingressCancel()` is called in the shutdown block at line ~1062; `dataWG.Wait()` follows at line ~1079; `writerWG.Wait()` follows at line ~1089; there is no existing WaitGroup waited between `dataWG.Wait()` and `writerWG.Wait()` | `cmd/switchboard/mgmt_wire.go`, `runRouter` shutdown block | Direct read, lines 1062-1089 |
| VP-14 | The DRAIN observer (the precedent for inline closures in `runRouter` capturing `sendMap` and live state) is declared inline as `drainCoord.RegisterObserver(func(_ context.Context) { ... })` capturing `sendMap` and `routerLogger` by reference; NOT a package var | `cmd/switchboard/mgmt_wire.go`, `runRouter`, `drainCoord.RegisterObserver` call | Direct read, lines 831-850 |
| VP-15 | Parent ruling v1.0 Phase-2 loop uses `go wireDiscoveryListener(ingressCtx, &wg, svtnID, ri, w)` — unchanged 5-arg signature; "where `ri` is the `RouterIngest` instance the relay-dispatch closure feeds" — the connection was intended but the seam mechanism was deferred to this ruling | `.factory/decisions/S-BL.DISCOVERY-WIRE-fanout-resolution-ruling.md`, Decision 5 / Phase-2 loop | Direct read |

---

## Decision 1 — Decision-Threading Seam

**RULING:** Add an `onRelay func(discovery.RouterIngestDecision)` callback parameter as the sixth
argument of `wireDiscoveryListener`. The listener goroutine checks `decision.Relay` inside its
ingest loop and invokes `onRelay(decision)` only when `decision.Relay == true`. The callback is
an inline closure in `runRouter` — consistent with the DRAIN observer pattern (VP-14) and ruling
Decision 3 of the parent ruling ("direct closure in `runRouter`, NOT a package-level var").

### Exact new signature

```go
func wireDiscoveryListener(
    ctx context.Context,
    wg *sync.WaitGroup,
    svtnID [16]byte,
    ri *discovery.RouterIngest,
    w io.Writer,
    onRelay func(discovery.RouterIngestDecision),
) error
```

### Modified ingest loop body in `wireDiscoveryListener`

Replace the current blank-identifier discard (VP-2):

```go
// BEFORE (current — discards decision):
if _, ingestErr := ri.Ingest(raw); ingestErr != nil {
    continue
}

// AFTER (Task 6d — threads decision to caller):
decision, ingestErr := ri.Ingest(raw)
if ingestErr != nil {
    continue
}
if decision.Relay && onRelay != nil {
    onRelay(decision)
}
```

### Callback contract

The `onRelay` callback is invoked **only** when `decision.Relay == true`. The listener
checks `decision.Relay` BEFORE calling `onRelay` — the callback contract is therefore:
"you have been given a relay-worthy, HMAC-verified, replay-accepted datagram; dispatch it."
The callback never observes `decision.Relay == false` datagrams; no filtering is needed
inside the callback. The nil guard (`onRelay != nil`) is described fully in Decision 2.

### `onRelay` closure shape in `runRouter`

Declared inline in `runRouter` after `ri` and `relayRateCap` construction (see Decision 3):

```go
onRelay := func(decision discovery.RouterIngestDecision) {
    if !relayRateCap.allow(decision.SVTNID, decision.NodeAddr) {
        return // silent drop — AC-018, SEC-DW-09
    }
    relayDispatch(router, &sendMap, decision)
}
```

Captures `router` (for `relayDispatch`'s `InterfacesForSVTN` call), `&sendMap` (for
`relayDispatch`'s `sendMap.Load` fan-out), and `relayRateCap` (for the AC-018 rate cap).
NOT a package-level var — identical seam posture to the DRAIN observer (VP-14, parent ruling
Decision 3).

---

## Decision 2 — Backward Compatibility of the Signature Change

**RULING:** Add `onRelay` as a **required** (not variadic) 6th parameter. Update the
existing test helpers to pass `nil` for `onRelay`.

### Why required, not variadic

Go variadic functions (`...func(discovery.RouterIngestDecision)`) obscure whether the
parameter is meaningful at a given call site. The signature change is a single-story event;
every call site (the existing test, the new `runRouter` loop) is updated in the same delivery.
A required parameter makes the caller explicitly choose between `nil` (discard) and a real
callback — no accidental omissions.

### Test update

`callWireDiscoveryListenerRecovered` (discovery_wire_test.go line 38) must be updated to
the 6-arg form:

```go
func callWireDiscoveryListenerRecovered(
    ctx context.Context,
    wg *sync.WaitGroup,
    svtnID [16]byte,
    ri *discovery.RouterIngest,
    w io.Writer,
    onRelay func(discovery.RouterIngestDecision),
) (panicked any, err error) {
    defer func() {
        if r := recover(); r != nil {
            panicked = r
        }
    }()
    err = wireDiscoveryListener(ctx, wg, svtnID, ri, w, onRelay)
    return panicked, err
}
```

The existing `TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly` test (line 101)
passes `nil` as `onRelay`:

```go
p, err := callWireDiscoveryListenerRecovered(ctx, &wg, svtnID, ri, io.Discard, nil)
```

### Nil semantics — explicit

`nil` for `onRelay` means "no relay dispatch." This is:
- The same behavior as today's blank-identifier discard (VP-2).
- A **fail-safe**, not fail-open: nil suppresses relay amplification, which is the
  conservative direction (no spurious frames sent to admitted nodes). This is explicitly
  NOT a security-perimeter parameter in the sense of go.md rule 13 — that rule applies when
  nil disables an authentication or authorization check, allowing an attacker to bypass
  enforcement. Here nil disables an AMPLIFICATION effect; the security invariants of hop-1
  ingest (HMAC verification, replay gate, rate cap) are all enforced by `RouterIngest.Ingest`
  BEFORE `onRelay` is invoked and are unaffected by whether `onRelay` is nil. A nil callback
  is therefore explicitly permitted at any call site that does not need relay dispatch (tests
  exercising only the AC-001 multicast-join behavior).

---

## Decision 3 — Rate-Cap Ownership, Placement, and Concurrency Requirements

**RULING:**

1. `relayRateCap` is constructed ONCE in `runRouter` before the discovery listener loop,
   and captured by the `onRelay` inline closure — exactly as the DRAIN coordinator and
   observer capture state (VP-14).

2. The `allow(svtnID [16]byte, nodeAddr [8]byte) bool` check is called INSIDE the `onRelay`
   closure, BEFORE `relayDispatch`. A `false` result is a silent drop with no log line, no
   error return, no counter (consistent with AC-018's "silent drop" requirement and
   SEC-DW-09's rate-amplification defense posture). A visibility counter (threshold-crossing,
   non-gating, `FailureCounter`-style per parent ruling Task 6c / SEC-DW-09 philosophy) MAY
   be emitted by the rate-cap type itself — that is 6c's concern and does not change this
   ruling's seam.

3. `relayDispatch` (Task 6b) remains **stateless** — this ruling does not modify
   `discovery_relay_wire.go`. The rate cap is owned by the `onRelay` closure, not by
   `relayDispatch`.

4. **Concurrency requirement for the Task 6c type:** Multiple `wireDiscoveryListener`
   goroutines (one per SVTN, launched in the startup loop) call the SAME `onRelay` closure
   concurrently. The closure captures:
   - `router` — `InterfacesForSVTN` uses `r.mu.RLock()` internally; safe for concurrent
     callers.
   - `&sendMap` — `sync.Map`; safe for concurrent callers.
   - `relayRateCap` — **MUST be mutex-guarded** inside its own type. Its `allow()` method
     must acquire an internal mutex before touching the per-`(SVTNID, NodeAddr)` bucket map.
     Task 6c's implementation must satisfy this requirement; this ruling is the authority
     for it. Model: the existing `tokenBucket` type in `internal/discovery/discovery_wire.go`
     (mutex on `b.mu`, `allow()` holds `b.mu.Lock()`).

5. **Integration `-race` test obligation (binding):** a test that concurrently invokes
   `onRelay` from two goroutines (simulating two per-SVTN listener goroutines both producing
   relay decisions at the same time) MUST pass `go test -race`. This test is part of the
   Task 6d (or 6c) acceptance evidence; it is not optional.

### Construction point in `runRouter`

`ri` and `relayRateCap` are constructed right before the discovery listener loop, after the
DRAIN observer registration and before the main for-select loop (around line ~901/~933). Exact
placement rule: after `connector.Start()` and the info-log block, before the `for { select ...
}` main loop. This ensures `router`, `routerLogger`, `sendMap`, and `ingressCtx` are all in
scope:

```go
// Construct the shared RouterIngest instance (one per runRouter invocation;
// shared across all per-SVTN listener goroutines — RouterIngest is safe for
// concurrent use, VP-11).
ri := discovery.NewRouterIngest(discovery.RouterIngestConfig{
    Router: router,
    Logger: routerLogger,
})

// Construct the per-(SVTNID,NodeAddr) relay rate cap (AC-018, SEC-DW-09).
// Constructed once; captured by onRelay below. Must be mutex-guarded (Decision 3).
relayRateCap := newRelayRateCap()

// onRelay: inline closure capturing router, &sendMap, relayRateCap.
// Called by each wireDiscoveryListener goroutine when decision.Relay==true.
// Matches the DRAIN observer seam (VP-14, parent ruling Decision 3).
onRelay := func(decision discovery.RouterIngestDecision) {
    if !relayRateCap.allow(decision.SVTNID, decision.NodeAddr) {
        return
    }
    relayDispatch(router, &sendMap, decision)
}

// Discovery listener startup loop (Task 6d — Forward Obligation (e) residual,
// now unblocked; see fanout-resolution-ruling.md v1.0 Decision 5).
var discoveryWG sync.WaitGroup
for svtnID := range routerKS.AllSVTNEntries() {
    svtnID := svtnID // per-goroutine capture (Go loop variable semantics)
    discoveryWG.Add(1)
    go wireDiscoveryListener(ingressCtx, &discoveryWG, svtnID, ri, w, onRelay)
}
```

---

## Decision 4 — Startup Loop Lifecycle (ctx, WaitGroup, Phase Placement)

### Which WaitGroup

**RULING: Introduce a new `var discoveryWG sync.WaitGroup` local to `runRouter`.**

Rationale for NOT reusing `writerWG`: `writerWG`'s shutdown-block comment (VP-9,
lines 1082-1088) carries a carefully-maintained invariant: "no writerWG.Add(1) can occur
after `dataWG.Wait()` returns." Discovery listener goroutines are started BEFORE the main
loop (at startup, not on-demand), so they could not cause a concurrent-Add/Wait race on
`writerWG` after shutdown. However, the existing comment would become misleading if
`writerWG` tracked two semantically distinct goroutine families (writer goroutines and
discovery listeners). A dedicated `discoveryWG` keeps semantics clean, makes the shutdown
join site self-documenting, and does not require revising a carefully-worded invariant
comment.

### Which context

**RULING: `ingressCtx`.**

When `ingressCancel()` fires in the shutdown block, each listener's `go func() { <-ctx.Done();
conn.Close() }()` goroutine fires immediately, closing the UDP socket, causing `ReadFromUDP`
to return an error with `ctx.Err() != nil`, and the listener returns `nil`. The `defer
wg.Done()` then fires, decrementing `discoveryWG`. This is the identical cancel-via-close
idiom already used by `wireDiscoveryListener`'s own implementation (discovery_wire.go
lines 103-106). No change to that mechanism is needed.

### Where discoveryWG.Wait() lives in shutdown

**RULING:** Between `dataWG.Wait()` (line ~1079) and `writerWG.Wait()` (line ~1089).

```go
dataWG.Wait()
discoveryWG.Wait() // NEW: join all discovery listener goroutines
writerWG.Wait()
```

This is safe because `ingressCancel()` was already called at line ~1062 (before
`dataWG.Wait()`), so by the time `discoveryWG.Wait()` is reached, every listener's
context is cancelled and its socket is closed. The wait is proven prompt: no listener
goroutine can outlive `ingressCancel()` + `conn.Close()` + one `ReadFromUDP` error return.

The existing `writerWG.Wait()` invariant ("grounded on dataWG.Wait having joined Serve's
accept loop") is preserved unchanged — `discoveryWG.Wait()` is inserted between the two
existing Waits and interacts with neither.

### Phase placement of the startup loop

**RULING:** After `connector.Start()` and the info-log block (around line ~933), before
the `for { select ... }` main loop. Justification: `ingressCtx` (VP-8) is declared at
line ~584; `sendMap` at ~592; `router` at ~538; `routerKS` at ~521. All required captured
variables are in scope. "Register before serve" is satisfied: goroutines are up before the
main event loop starts, consistent with the analogous DRAIN observer registration at
lines 831-850 (VP-14).

### ri is constructed once and shared

**RULING:** `ri := discovery.NewRouterIngest(...)` is constructed once in `runRouter`,
before the startup loop, and passed as the same instance to every `wireDiscoveryListener`
goroutine. `RouterIngest` is safe for concurrent use (VP-11). One shared instance is correct:
`Ingest`'s `lastSeen` map is keyed by `(svtnID, nodeAddr)` — SVTN A's listener goroutine
and SVTN B's listener goroutine both update the shared `lastSeen` map under the same mutex,
which is the correct replay-gate semantics (a node's sequence is global, not per-SVTN-listener).

---

## Decision 5 — AllSVTNEntries at Startup Only vs. Dynamic

**RULING: Startup-snapshot enumeration is in-scope for Task 6d. Dynamic multicast group
join/leave is a NEW named Forward Obligation, explicitly deferred.**

`routerKS.AllSVTNEntries()` is called once at startup. Any SVTN admitted AFTER startup via
`wireAdmissionSyncHandlers` push does not get a corresponding discovery listener goroutine
during this process run. This is an accepted limitation for this story.

**Forward Obligation (g) — Dynamic discovery listener registration:**
When a new SVTN is pushed to the router via `internal.admission.register` (post-startup),
the router does not automatically join the corresponding multicast group. A follow-on story
must add an admission-event hook (equivalent to the nodeConnHook pattern, but for SVTN
admission events) and call `wireDiscoveryListener` for each newly admitted SVTN. This
requires new API surface in `wireAdmissionSyncHandlers` or a new `registerSVTNListenerHook`
seam — neither is authorized by the current story's File-Change List.

**Story-writer must add this as Forward Obligation row (g).** This ruling's scope ends at
startup-snapshot enumeration.

---

## Implementation Checklist for Task 6d

Ordered: RED (failing test) before GREEN (implementation).

### RED — new failing tests

- [ ] Update `callWireDiscoveryListenerRecovered` in `cmd/switchboard/discovery_wire_test.go`
      to the 6-arg form (add `onRelay func(discovery.RouterIngestDecision)` parameter) —
      this fails to compile against the current 5-arg signature. That compile failure IS the
      Red Gate.
- [ ] Update `TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly` call site to pass
      `nil` as `onRelay`.
- [ ] Write `TestWireDiscoveryListener_InvokesOnRelay_WhenRelayTrue` (unit, same file):
      construct an `ri` with a real admitted key; send a valid HMAC-authenticated datagram;
      verify the `onRelay` callback is invoked with `decision.Relay == true`. Verify `onRelay`
      is NOT invoked for a datagram that fails HMAC (ingest error path).
- [ ] Write a concurrent-onRelay test (e.g., `TestOnRelayClosureConcurrentAccess`) that
      calls `onRelay` from N goroutines simultaneously and verifies no data race on
      `relayRateCap.allow()`. This test must pass `go test -race`.

### GREEN — `wireDiscoveryListener` signature change

- [ ] Add `onRelay func(discovery.RouterIngestDecision)` as the 6th parameter of
      `wireDiscoveryListener` in `cmd/switchboard/discovery_wire.go`.
- [ ] Replace the blank-identifier discard (VP-2) with the decision-threading body:
      ```go
      decision, ingestErr := ri.Ingest(raw)
      if ingestErr != nil {
          continue
      }
      if decision.Relay && onRelay != nil {
          onRelay(decision)
      }
      ```
- [ ] Verify `callWireDiscoveryListenerRecovered` compiles with `nil` for `onRelay`.

### GREEN — `runRouter` wiring

- [ ] Construct `ri` once (`discovery.NewRouterIngest(...)` with `router` and `routerLogger`)
      before the discovery listener loop.
- [ ] Construct `relayRateCap` once (Task 6c's type) before `onRelay`. Confirm the type has
      an internal mutex (Decision 3 concurrency requirement).
- [ ] Declare `onRelay` inline closure capturing `router`, `&sendMap`, `relayRateCap` (shape
      in Decision 1/Decision 3).
- [ ] Declare `var discoveryWG sync.WaitGroup` (Decision 4).
- [ ] Iterate `routerKS.AllSVTNEntries()` with `for svtnID := range`; per svtnID, loop-variable
      capture, `discoveryWG.Add(1)`, `go wireDiscoveryListener(ingressCtx, &discoveryWG, svtnID,
      ri, w, onRelay)`.
- [ ] Add `discoveryWG.Wait()` to the shutdown block between `dataWG.Wait()` and `writerWG.Wait()`
      (Decision 4 / shutdown placement).

### GREEN — doc-comment update

- [ ] Update `cmd/switchboard/discovery_wire.go` package-doc: remove the deferral paragraph
      ("left to a follow-on story once an SVTN-admission-event source exists..."; lines 14-28
      of the file). Replace with a brief note that Task 6d has wired this call and that dynamic
      SVTN join/leave is Forward Obligation (g).
- [ ] Update `wireDiscoveryListener`'s own function doc comment to document the new `onRelay`
      parameter and its nil semantics (Decision 2).

### Integration and -race tests

- [ ] Extend or add an integration test verifying the full `runRouter` → `wireDiscoveryListener`
      → `relayDispatch` chain end-to-end: two admitted nodes on the same SVTN; advertisement
      arrives on the multicast group; relay frame appears on the non-originating node's send
      channel. This can extend `TestRunRouter_DiscoveryListener_JoinsGroup_RouterModeOnly` or
      live in a new test function.
- [ ] `go test -race` passes on the concurrent-`onRelay` test (RED step above) and on the full
      integration test.

### Quality gate

```sh
just fmt
just lint
just test-race
```

---

## ARCH-08 Compliance Note

- `cmd/switchboard` (position 18): `discovery_wire.go` already imports `internal/discovery`
  (position 14) for `RouterIngest` and `RouterIngestDecision`. The new `onRelay` parameter is
  `func(discovery.RouterIngestDecision)` — no new imports in `discovery_wire.go` beyond what
  is already present.
- `internal/discovery` (position 14): no changes to this package in Task 6d. Its import profile
  is unchanged.
- `cmd/switchboard/mgmt_wire.go` (position 18): already imports `internal/discovery` (via
  `discovery_wire.go`'s same package), `internal/routing`, and `sync`. The `onRelay` closure
  and `discoveryWG` additions require no new imports.
- Import DAG is NOT modified by this ruling. Verified: no new `import` statements required in
  any file.

---

## Downstream Touch-List (story-writer / PO — not executed here)

| Artifact | Change | Owner |
|---|---|---|
| `.factory/stories/S-BL.DISCOVERY-WIRE.md` | (1) Update Task 6d text to reference this ruling's exact signature and loop shape. (2) Update Task 6d's File-Change List row for `discovery_wire.go` to note the 6-arg signature and the updated ingest loop. (3) Update `discovery_wire_test.go` row to note the `callWireDiscoveryListenerRecovered` 6-arg update. (4) Add Forward Obligation row (g): "Dynamic discovery listener registration — post-startup SVTN admission does not join the multicast group; requires admission-event hook (deferred)." (5) No AC count change; no points change. | story-writer |
| `cmd/switchboard/discovery_wire.go` | Package-doc deferral comment update at Task 6d GREEN step (implementer obligation at delivery time, not pre-delivery) | implementer |

---

## Erratum / Consistency Note on Parent Ruling

The parent ruling's Phase-2 loop (Decision 5) shows:
```go
go wireDiscoveryListener(ingressCtx, &wg, svtnID, ri, w)
```
using the 5-arg (unchanged) signature. This is NOT an erratum — the parent ruling explicitly
labelled this as the lifecycle shape only ("where `ri` is the `RouterIngest` instance the
relay-dispatch closure feeds") and stated that the dispatch seam was the Task 6d gap to
resolve. The `&wg` in the prior ruling was illustrative and did not bind to any existing WG
in `runRouter`; this ruling names it `&discoveryWG` (a new WG). The 6th `onRelay` parameter
completing the loop is what this ruling adds. No prior ruling content is contradicted.

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-20 (v1.0) | architect | **Initial ruling, five decisions on the Task 6d wiring seam for S-BL.DISCOVERY-WIRE.** Verified against worktree feature/S-BL.DISCOVERY-WIRE-FANOUT @ `4b82535c4a12318370b7a4fe13931abbb324c347`. **(1) Decision-threading seam:** Add `onRelay func(discovery.RouterIngestDecision)` as required 6th param to `wireDiscoveryListener`. Listener checks `decision.Relay` and calls `onRelay(decision)` only when true. `onRelay` is an inline closure in `runRouter` capturing `router`, `&sendMap`, `relayRateCap` — DRAIN-observer pattern (VP-14, parent ruling Decision 3). Callback contract: caller receives only relay-worthy decisions; nil guard inside listener prevents panic. **(2) Backward compat:** Required 6th param (not variadic). Existing test passes `nil`; `nil` semantics = discard = today's behavior = fail-safe suppression of relay amplification, NOT a security perimeter (go.md rule 13 does NOT apply; nil is explicitly permitted). Update `callWireDiscoveryListenerRecovered` and the test call site. **(3) Rate-cap:** `relayRateCap` constructed once in `runRouter`, captured by `onRelay`. `allow(svtnID, nodeAddr)` checked INSIDE `onRelay` before `relayDispatch`. False = silent drop. `relayDispatch` (Task 6b) remains stateless. Task 6c's type MUST be mutex-guarded for concurrent access from multiple per-SVTN listener goroutines. Integration `-race` test is a binding obligation. **(4) Startup loop lifecycle:** New `var discoveryWG sync.WaitGroup` (not `writerWG` — VP-9 semantics-invariant conflict); `ingressCtx` (torn down by `ingressCancel()`); loop after `connector.Start()` and info-log block, before main select loop; `discoveryWG.Wait()` in shutdown block between `dataWG.Wait()` and `writerWG.Wait()`; `ri` constructed once and shared (RouterIngest concurrency-safe, VP-11). **(5) AllSVTNEntries:** Startup-snapshot only is in-scope. Dynamic join/leave on post-startup SVTN admission is a NEW Forward Obligation (g) — requires admission-event hook, outside this story's File-Change List. Story-writer adds FO(g) row. |
