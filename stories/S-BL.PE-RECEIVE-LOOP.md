---
artifact_id: S-BL.PE-RECEIVE-LOOP
document_type: story
level: ops
story_id: S-BL.PE-RECEIVE-LOOP
title: "PE-connection receive/forward loop — frame.ReadOuterFrame goroutine, FrameTypePEConnect constant, and E-FWD-001 exhaustion discharge"
status: ready
producer: story-writer
timestamp: 2026-07-08T00:00:00Z
version: "1.0"
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: PE
points: 5
bc_traces:
  - BC-2.02.008   # PC-3/EC-003 — E-FWD-001 exhaustion discharge (binding anchor; re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 postcondition 1)
  - BC-2.06.003   # PC-1 — non-discharging prerequisite trace; receive loop makes the full send+forward path live for future path-liveness observability
vp_traces: []
subsystems: [deployment-operations, transport-layer]
architecture_modules:
  - internal/frame
  - internal/upstreamdial
  - internal/routing
  - internal/arqsend
  - internal/testenv
  - cmd/switchboard
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on:
  - S-7.04-FU-PE-CONNECTOR   # MERGED — PR #115 @ 8eb54a5; established TCP connections; FrameTypeData placeholder in dialLoop bootstrap to be replaced by this story
blocks:
  - S-7.04-FU-DRAIN-WIRE   # DRAIN broadcast over PE connections requires an operational receive/forward loop on those connections
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-02/BC-2.02.008.md'
  - '.factory/specs/behavioral-contracts/ss-06/BC-2.06.003.md'
  - '.factory/stories/S-7.04-FU-PE-CONNECTOR.md'
  - '.factory/decisions/S-BL.PE-RECEIVE-LOOP-placement-note.md'
  - '.factory/decisions/S-BL.PE-RECEIVE-LOOP-disposition-ruling.md'
acceptance_criteria_count: 5
backlog_origin:
  source: S-7.04-FU-PE-CONNECTOR
  adjudication: PO adjudication of adversary pass-1 F-P1-002 (AC-004 partial-discharge, class unmet-deps)
  drift_items_consumed:
    - S404-OBS-F   # E-FWD-001 rate-limit LATENT re-confirmation — re-anchored from S-7.04-FU-PE-CONNECTOR AC-004
    - S404-LOW-1   # live-egress re-confirmation (3 LOW + SEC-001) — re-anchored from S-7.04-FU-PE-CONNECTOR AC-004
---

# S-BL.PE-RECEIVE-LOOP: PE-Connection Receive/Forward Loop

## Narrative

- **As an** operator with an active PE router (established upstream connections via
  `S-7.04-FU-PE-CONNECTOR`)
- **I want** a per-connection receive goroutine that reads incoming frames from each PE
  upstream via `frame.ReadOuterFrame` (new — defined by this story) and routes them
  through `routing.FrameArrivalHandler.OnFrameArrival`
- **So that** the full send+forward path is exercised, E-FWD-001 can fire under
  path-exhaustion load, and `S-7.04-FU-DRAIN-WIRE` has a meaningful receive loop to
  broadcast DRAIN frames over

## Context

`S-7.04-FU-PE-CONNECTOR` (merged PR #115 @ `8eb54a5`) delivered the outbound TCP dial
loop: each configured upstream router address is dialed, a bootstrap `halfchannel.ChannelFrame`
is written, and the `connected-count atomic.Int32` tracks live connections. What that story
does NOT provide is a receive goroutine per PE connection that reads incoming frames and
routes them through `routing.FrameArrivalHandler.OnFrameArrival`
(`internal/routing/on_frame_arrival.go`). E-FWD-001 (split-horizon drop + log event) is
emitted exclusively from `OnFrameArrival`. Without a receive loop, `arqsend.Retransmitter`
test-internal wiring, and callback-seam integration, the sustained-load path-exhaustion case
that exercises E-FWD-001 cannot be reached from a live PE daemon.

This story also discharges FO-PE-LOOP-001 (from S-7.04-FU-PE-CONNECTOR F-P26-001 v1.24):
define `frame.FrameTypePEConnect = 0x06` (new — defined by this story), update `Valid()`
upper bound, and flip the `dialLoop` bootstrap construction from the
`halfchannel.FrameTypeData` placeholder to the new constant. The receive loop discriminates
bootstrap frames and discards them, so session-data frames pass through to the callback
without interference.

**Previous Story Intelligence.** `S-7.04-FU-PE-CONNECTOR` ran 32 adversarial passes
(39 findings). Key lessons carried forward: (1) every new symbol in ACs must be
grep-resolved at `8eb54a5` or marked "(new — defined by this story)"; (2) test count
roll-ups must state estimates in forecast tense; (3) line-number citations are forbidden
in story prose — use mechanism-anchor descriptions; (4) the concurrent-transition lesson
(F-P29-001) applies symmetrically here — the receive goroutine MUST NOT share mutable
state with `dialLoop` beyond the `net.Conn`; (5) the `addrsCh` fast-path pattern (F-P5-001)
is established idiom in `internal/upstreamdial` — do not reintroduce blocking inner-receive.

**Token Budget Estimate (forecast).** Story spec: ~8k tokens. Referenced production code
(`connector.go`, `mgmt_wire.go`, `frame.go`, `on_frame_arrival.go`): ~6k tokens.
Test infrastructure (`testenv.go`, `arqsend.go`, existing test patterns): ~5k tokens.
Total implementing-agent context: ~19k tokens — well within 20–30% of a 200k context window.
No story split required.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|-------------|
| BC-2.02.008 PC-3/EC-003 — E-FWD-001 fires when only eligible interface is arrival interface | BC-2.02.008 / S404-OBS-F | S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (re-anchored, unmet-deps F-P1-002) | TO DISCHARGE — E-FWD-001 fires under sustained path-exhaustion via live PE connection + arqsend retransmit (AC-004; S404-OBS-F + S404-LOW-1 re-confirmation rides AC-004) |
| BC-2.06.003 PC-1 — `status: "failed"` via path liveness failure | BC-2.06.003 | S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (re-anchored, same partial-discharge) | **Non-discharging prerequisite trace.** This story ships the receive goroutine that makes the full send+forward path live. BC-2.06.003 PC-1 `status: "failed"` (path liveness) is NOT discharged here — it requires the keepalive missed-probe mechanism (`internal/paths`), which is orthogonal to E-FWD-001 (split-horizon, `internal/routing`). Future path-liveness observability testing depends on the infrastructure this story ships. (Disposition per S-BL.PE-RECEIVE-LOOP-disposition-ruling.md v1.0 Q-A option (a).) |
| FO-PE-LOOP-001 — define `frame.FrameTypePEConnect`; flip `dialLoop` bootstrap | FO-PE-LOOP-001 | S-7.04-FU-PE-CONNECTOR F-P26-001 (v1.24 deferral) | TO DISCHARGE — `FrameTypePEConnect = 0x06` defined; `Valid()` upper bound updated; `dialLoop` bootstrap flipped from `halfchannel.FrameTypeData` placeholder (AC-003) |
| S404-OBS-F — E-FWD-001 rate-limit re-confirmation | S404-OBS-F | STATE.md row; re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 | DISCHARGED via AC-004 (E-FWD-001 exhaustion integration test) |
| S404-LOW-1 — live-egress re-confirmation (3 LOW + SEC-001) | S404-LOW-1 | STATE.md row; re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 | DISCHARGED via AC-004 (full send+forward path traversal demonstrated end-to-end) |

---

## Design Constraints

### Receive Goroutine Ownership and Callback Seam (Q1, Q2)

**Binding (per placement note Q1 and Q2).**

The receive goroutine lives inside `internal/upstreamdial.Connector` (position 19),
one goroutine per established connection, started after step-3 success in `dialLoop`.
`upstreamdial` remains routing-free per the forbidden-edge constraint (ARCH-08 §6.6.2:
`routing` is explicitly listed as a forbidden import for `upstreamdial`).

The seam is a callback added to the `upstreamdial.Handle` interface:

```go
// In internal/upstreamdial (new — defined by this story):
type FrameFn func(hdr frame.OuterHeader, raw []byte) error

// Added to the Handle interface (new — defined by this story):
SetFrameCallback(fn FrameFn)
```

This mirrors the `netingress.RouteFn` pattern (`type RouteFn func(hdr frame.OuterHeader, payload []byte) error`, verified at `8eb54a5` in `internal/netingress/netingress.go`). `runRouter` in `cmd/switchboard/mgmt_wire.go` calls `connector.SetFrameCallback(fn)` after construction, passing a closure that calls `routing.RouteFrame` (verified at `8eb54a5` in `internal/routing/routing.go`).

### Framing Primitive: frame.ReadOuterFrame (Q2)

**Binding (per placement note Q2).**

A new function `frame.ReadOuterFrame(r io.Reader) (frame.OuterHeader, []byte, error)` is
added to `internal/frame/frame.go` (position 2). It implements the same read-header-then-
payload logic as `netingress.ReadFrame` (verified at `8eb54a5` in
`internal/netingress/netingress.go`): read `frame.OuterHeaderSize` (= 44) bytes via
`frame.ParseOuterHeader`, then read `hdr.PayloadLen` bytes of payload. `netingress.ReadFrame`
may delegate to it or retain its own copy with a cross-reference comment — implementer's
choice.

**ARCH-08 §6.5 amendment obligation (Q2):** `internal/upstreamdial` gains a direct import
edge to `internal/frame` (position 2). The allowed-import row must be updated in §6.5:
`{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}`. This is a §6.4
amendment (import-set extension of an existing package). The implementer must update
ARCH-08 §6.5 in the same commit that introduces the `frame.ReadOuterFrame` import.

### FrameTypePEConnect Constant and Valid() Bound (Q3 — FO-PE-LOOP-001)

**Binding (per placement note Q3).**

```go
// In internal/frame/frame.go (new — defined by this story):
FrameTypePEConnect FrameType = 0x06
```

`frame.FrameType.Valid()` currently reads `return f >= FrameTypeData && f <= FrameTypeFec`
with `FrameTypeFec = 0x05` (verified at `8eb54a5` in `internal/frame/frame.go`). Adding
`FrameTypePEConnect = 0x06` REQUIRES updating `Valid()` to
`return f >= FrameTypeData && f <= FrameTypePEConnect`. Failing to update `Valid()` will
cause `frame.ParseOuterHeader` to return `ErrInvalidFrameType` for every PE-CONNECT
bootstrap frame, silently dropping all upstream bootstraps.

**dialLoop bootstrap flip obligation:** `dialLoop` in `internal/upstreamdial/connector.go`
(verified at `8eb54a5`) currently sets `FrameType: halfchannel.FrameTypeData` as a
placeholder (per the F-P26-001 shipped-deferral note in S-7.04-FU-PE-CONNECTOR v1.24).
This story flips it to `frame.FrameTypePEConnect` (new — defined by this story). The
`frame` import required for this constant is covered by the same §6.5 ARCH-08 amendment.

**Receive-loop discrimination contract:**

```
After frame.ReadOuterFrame returns (hdr, payload):
  if hdr.FrameType == frame.FrameTypePEConnect {
      // bootstrap acknowledgment: silent discard (no reply defined in this story's scope)
  } else {
      // session data / ctl / arq / fec frame: pass to FrameFn callback
  }
```

Bootstrap frames (type 0x06) are silently discarded. All other frame types are forwarded
to the caller-supplied `FrameFn`.

### arqsend.Retransmitter Wiring (Q4)

**Binding (per placement note Q4).**

`arqsend.New` (verified at `8eb54a5` in `internal/arqsend/arqsend.go`) is constructed
INSIDE the integration test that exercises E-FWD-001 under sustained load. It is NOT
wired into the production `runRouter` datapath. A per-test construction inside the test
body is the correct shape:

```go
a := arq.New(arq.Config{...})
rt := arqsend.New(a, outerassembler.Envelope{}, arqsend.WithChanID(1))
dispatch := func(wire []byte) error {
    conn, err := net.Dial("tcp", routerListenAddr)
    if err != nil { return err }
    defer conn.Close()
    _, err = conn.Write(wire)
    return err
}
```

`arqsend.Retransmitter` is pure-core (no goroutines, no I/O — verified at `8eb54a5`);
its lifecycle is bounded to the test function. No new production import is needed.

### Receive Goroutine Lifecycle and doneCh Contract (Q6)

**Binding (per placement note Q6).**

The receive goroutine is owned by `dialLoop` and exits when the per-address connection is
closed (`conn.Close()` called by `dialLoop` teardown causes `frame.ReadOuterFrame` to return
`io.EOF` or a net error). No separate stop channel is needed; the goroutine drains naturally
on conn close.

**Exactly-once semantics (F-P29-001 lesson applied symmetrically):** the receive goroutine
MUST NOT access `c.connectedCount` or any other shared state. It has exactly one output:
calling the `FrameFn` callback with received bytes.

**Per-address done channel:** The per-address `done chan struct{}` (same pattern as
`addrCancel.done` in `Connector.reconcile`) MUST NOT be closed until BOTH `maintainConn`
AND the receive goroutine have returned. `Connector.Stop()` blocks on `c.doneCh`, which is
closed by `reconcileLoop` only after all `addrCancel.done` channels are drained. The
implementer must use a `sync.WaitGroup` or a per-connection `done chan struct{}` to
synchronize `dialLoop` teardown with the receive goroutine's exit.

**`dialLoop` goroutine ordering:**

```
1. dial
2. bootstrap (FrameTypePEConnect frame — new constant)
3. connectedCount.Add(+1)
4. START receive goroutine (owns conn, ctx.Done())
5. maintainConn(addr, conn, ctx.Done(), tick)  ← blocks
6. receive goroutine exits (conn closed / ctx done)
7. connectedCount.Add(-1)  [independent of receive goroutine state]
8. conn.Close()
9. per-address done signal (only after receive goroutine confirms exit)
```

---

## Acceptance Criteria

### AC-001 — Receive goroutine active per established PE connection; incoming frames reach FrameArrivalHandler

**BC Anchors:** BC-2.09.001 PC-2/PC-3 (upstream connections established; router is in PE
mode); placement note Q1, Q2.

**Precondition:** `runRouter` is executing with a PE config. A `Connector` has been
constructed with a PE upstream address. `connector.SetFrameCallback(fn)` (new — defined by
this story) has been called with a closure wiring to `routing.RouteFrame` (verified at
`8eb54a5`). A cooperative upstream fixture (`e.PERouterAddr(t)`) is accepting connections.

**Postconditions:**

1. After `dialLoop` step-3 success, a receive goroutine is started on the established
   `net.Conn`. The goroutine calls `frame.ReadOuterFrame(conn)` (new — defined by this
   story) in a loop.
2. A non-bootstrap incoming frame (any type other than `frame.FrameTypePEConnect`) received
   from the upstream fixture is passed to the `FrameFn` callback, which routes it via the
   `routing.RouteFrame` closure in `runRouter`. The path reaches
   `routing.FrameArrivalHandler.OnFrameArrival` (verified at `8eb54a5` in
   `internal/routing/on_frame_arrival.go`).
3. `connector.Mode()` returns `upstreamdial.ModePE` (≥1 upstream connected), confirming the
   connection is established and the receive goroutine is live.

**Test names:**

- `TestConnector_ReceiveLoop_DataFrameForwardedToCallback` (unit, `internal/upstreamdial/connector_test.go` — sends a data frame on the upstream fixture side; asserts FrameFn callback invoked with the correct hdr + payload)
- `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` (integration, `cmd/switchboard/router_pe_receive_test.go` — starts testenv PE router; sends well-formed frame from upstream fixture; asserts OnFrameArrival path is reached via writer output change or log event)

**Test level:** unit (Connector callback) + integration (runRouter end-to-end)
**Test files:** `internal/upstreamdial/connector_test.go`, `cmd/switchboard/router_pe_receive_test.go`

---

### AC-002 — runRouter wires SetFrameCallback closure through routing.FrameArrivalHandler.OnFrameArrival

**BC Anchors:** BC-2.02.008 PC-3 (frame routing path is live); placement note Q1, Q2.

**Precondition:** `runRouter` constructs a `Connector` (via `upstreamdial.New`, verified at
`8eb54a5`) and calls `connector.SetFrameCallback(fn)` (new — defined by this story) with a
closure that captures the router instance.

**Postconditions:**

1. The closure passed to `SetFrameCallback` calls `routing.RouteFrame(hdr, raw, router)`
   (verified at `8eb54a5` in `internal/routing/routing.go`), which dispatches to
   `routing.FrameArrivalHandler.OnFrameArrival` (verified at `8eb54a5`).
2. `cmd/switchboard/mgmt_wire.go` gains the `SetFrameCallback` call. No `routing` import
   is introduced in `internal/upstreamdial` — the callback seam preserves the
   ARCH-08 §6.6.2 forbidden-edge constraint.

**Test names:**

- `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` (integration, `cmd/switchboard/router_pe_receive_test.go` — sends a frame that reaches OnFrameArrival; asserts the writer output reflects routing activity; confirms no routing import in upstreamdial via `go list -deps`)

**Test level:** integration
**Test file:** `cmd/switchboard/router_pe_receive_test.go`

---

### AC-003 — FO-PE-LOOP-001 discharged: FrameTypePEConnect constant, Valid() bound, dialLoop flip, bootstrap discrimination

**BC Anchors:** FO-PE-LOOP-001 (from S-7.04-FU-PE-CONNECTOR F-P26-001); placement note Q3.

**Precondition:** `internal/frame/frame.go` (verified at `8eb54a5`) defines
`FrameTypeFec FrameType = 0x05` as the current upper bound in `Valid()`. `dialLoop` in
`internal/upstreamdial/connector.go` sets `FrameType: halfchannel.FrameTypeData` as a
placeholder.

**Postconditions:**

1. `frame.FrameTypePEConnect FrameType = 0x06` is defined in `internal/frame/frame.go`
   (new — defined by this story). `frame.FrameType(0x06).Valid()` returns `true`.
   `frame.FrameType(0x07).Valid()` returns `false` (upper-bound not over-widened).
2. `frame.FrameType.Valid()` upper bound is updated to `<= frame.FrameTypePEConnect` in the
   same commit that defines the constant. `frame.ParseOuterHeader` no longer returns
   `ErrInvalidFrameType` for bootstrap frames.
3. `dialLoop` bootstrap construction sets `FrameType: frame.FrameTypePEConnect` (new —
   defined by this story) instead of `halfchannel.FrameTypeData`. The
   `internal/upstreamdial` package gains a direct `internal/frame` import (ARCH-08 §6.5
   amendment landed in the same commit).
4. The receive goroutine applies the discrimination contract: a frame with
   `hdr.FrameType == frame.FrameTypePEConnect` is silently discarded and NOT forwarded to
   the `FrameFn` callback.

**Test names:**

- `TestFrameType_Valid_PEConnect` (unit, `internal/frame/frame_test.go` — asserts `frame.FrameTypePEConnect.Valid() == true` and `frame.FrameType(0x07).Valid() == false`)
- `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` (unit, `internal/upstreamdial/connector_test.go` — sends a FrameTypePEConnect frame on upstream side; asserts FrameFn callback is NOT invoked)

**Test level:** unit (both tests)
**Test files:** `internal/frame/frame_test.go`, `internal/upstreamdial/connector_test.go`

---

### AC-004 — E-FWD-001 exhaustion integration discharge (BC-2.02.008 PC-3/EC-003); S404-OBS-F and S404-LOW-1 re-confirmation

**BC Anchors:** BC-2.02.008 PC-3/EC-003 (split-horizon drop + E-FWD-001 event logged —
binding anchor per disposition ruling v1.0); S404-OBS-F; S404-LOW-1; placement note Q5.

**S404-OBS-F and S404-LOW-1 note:** Both drift anchors (re-confirmed at live egress under
sustained retransmit via the full send+forward path) are discharged by this AC. The
`"E-FWD-001"` emission in the writer output IS the re-confirmation vehicle for both.

**Precondition:** The test router is started in PE mode via `testenv.New(ctx, t)` with
`UpstreamRouters: []string{e.PERouterAddr(t)}`. `connector.Mode()` is polled for
`testenv.ModePE` (via `testenv.RouterHandle.Mode()`, verified at `8eb54a5`), confirming
both the dial loop (PE-CONNECTOR) and the receive goroutine (this story) are live.
An `arqsend.Retransmitter` instance is constructed test-internally (Q4 ruling; NOT wired
into production `runRouter`):

```go
a := arq.New(arq.Config{...})           // arq.New verified at 8eb54a5
rt := arqsend.New(a, outerassembler.Envelope{}, arqsend.WithChanID(1))
// arqsend.New, arqsend.WithChanID verified at 8eb54a5
dispatch := func(wire []byte) error {
    conn, err := net.Dial("tcp", routerListenAddr)
    if err != nil { return err }
    defer conn.Close()
    _, err = conn.Write(wire)
    return err
}
```

**Postconditions:**

1. Under sustained ARQ retransmit load via `arqsend.Retransmitter.Retransmit` (verified at
   `8eb54a5`) with `dispatch` writing well-formed `outerassembler.Assemble` (verified at
   `8eb54a5`) output to the router's data-plane `ListenAddr`, the router's writer output
   contains the string `"E-FWD-001"`. This is the spec-anchored event code (F-P11-001
   lesson from S-7.04-FU-PE-CONNECTOR: do NOT assert `"split-horizon-blocked"` or
   `"all paths split-horizon"` — the event code tag is stable across prose rewording of
   the emission text). The production emission string (verified at `8eb54a5` in
   `internal/routing/on_frame_arrival.go`) is:
   `"all paths split-horizon-blocked: frame dropped (checksum=0x%08x iface=%d) (BC-2.02.008 E-FWD-001)"`.
   The assertion key `"E-FWD-001"` resolves against this production string.
2. Path exhaustion condition: in the single-router `testenv` setup, the router's forwarding
   table has only the connection used to send test frames as the registered interface for
   the SVTN. Any frame arriving on that interface has no other eligible forwarding target
   (split-horizon blocks return to arrival interface), so E-FWD-001 fires naturally without
   special forwarding-table setup. The implementer must verify this assumption at
   implementation time; if the assumption does not hold, a dedicated loopback fixture
   pre-registers a forwarding entry pointing back to the arrival interface.
3. The `TestScanForLine_DetectsEFWD001ProductionEmission` mutation pin (verified at
   `8eb54a5` in `cmd/switchboard/router_pe_connector_test.go`) validates that `"E-FWD-001"`
   detects the production emission string. This test MUST remain unmodified and green.

**Test names:**

- `TestRunRouter_PE_EFWD001ExhaustionUnderLoad` (integration, `cmd/switchboard/router_pe_receive_test.go` — testenv.New, PE upstream fixture, arqsend dispatch, assert "E-FWD-001" in writer output; re-confirms S404-OBS-F + S404-LOW-1)
- `TestScanForLine_DetectsEFWD001ProductionEmission` (**existing normative pin** in `cmd/switchboard/router_pe_connector_test.go` — must remain unmodified and green; documents the `"E-FWD-001"` assertion key)

**Test level:** integration (`router_pe_receive_test.go`) + existing normative pin (must stay green)
**Test files:** `cmd/switchboard/router_pe_receive_test.go`, `cmd/switchboard/router_pe_connector_test.go` (existing, unmodified)

---

### AC-005 — Receive goroutine lifecycle: joins per-address done channel; Stop() blocks until both maintainConn and receive goroutine return

**BC Anchors:** Q6 ruling (placement note); F-P29-001 concurrent-transition lesson
(S-7.04-FU-PE-CONNECTOR pass-29).

**Precondition:** A `Connector` is running with ≥1 established PE upstream connection
(receive goroutine active). `connector.Mode()` returns `upstreamdial.ModePE` (verified at
`8eb54a5`).

**Postconditions:**

1. When `conn.Close()` is called by `dialLoop` teardown (via context cancellation or
   `Stop()`), the receive goroutine's `frame.ReadOuterFrame(conn)` (new — defined by this
   story) call returns `io.EOF` or a net error, and the goroutine exits. The per-address
   `done chan struct{}` is NOT closed until the receive goroutine has confirmed exit (via
   `sync.WaitGroup` or per-connection done signal).
2. `Connector.Stop()` (which calls `stopOnce.Do(close(c.stopCh))` and blocks on `c.doneCh`,
   verified at `8eb54a5`) returns only after BOTH `maintainConn` AND the receive goroutine
   have returned for every active address. No goroutine leak survives `Stop()`.
3. The receive goroutine does NOT access `c.connectedCount` or any shared mutable state
   beyond the `net.Conn` passed to it by `dialLoop`. Concurrent `Stop()` + receive-goroutine
   exit has exactly-once semantics (F-P29-001 lesson applied symmetrically).

**Test names:**

- `TestConnector_ReceiveLoop_ExitsOnConnClose` (unit, `internal/upstreamdial/connector_test.go` — establishes connection with receive goroutine active; closes the upstream server; asserts Stop() returns within deadline without goroutine leak; `go test -race` clean)

**Test level:** unit
**Test file:** `internal/upstreamdial/connector_test.go`

---

## Non-Goals

- Does not implement DRAIN message broadcast to connected nodes. That is `S-7.04-FU-DRAIN-WIRE`.
- Does not implement RESYNC control-frame protocol. That is `S-BL.RESYNC-FRAME`. A RESYNC
  frame arriving on a PE connection passes to the `FrameFn` callback as a `FrameTypeCtl`
  frame (verified at `8eb54a5` in `internal/frame/frame.go`); further dispatch is that
  story's concern.
- Does not implement the full `internal/admission` challenge-response handshake as part of
  the PE-CONNECT bootstrap exchange. The `outerassembler.Envelope` zero-field deferral from
  S-7.04-FU-PE-CONNECTOR remains OPEN — this story implements the distinct frame type
  (`FrameTypePEConnect`) but does NOT add session identity derivation
  (`frame.DeriveNodeAddress`, HMAC key derivation). That remains the session-bootstrap
  follow-on story.
- Does not add BC-2.06.003 PC-1 `status: "failed"` integration assertion. The receive loop
  shipped here is a structural prerequisite for future path-liveness observability testing,
  but `PathSnapshot.Failed` is set only by the keepalive consecutive-miss threshold in
  `internal/paths` — a fully orthogonal mechanism already shipped by S-BL.PATH-FAILED-STATUS
  (PR #99, `c098827`). See disposition ruling v1.0 Q-A option (a) for the full rationale.

---

## Files-Changed List (FCL)

| # | File | Change | BC / Anchor |
|---|------|--------|-------------|
| 1 | `internal/frame/frame.go` (MODIFIED) | Add `FrameTypePEConnect FrameType = 0x06`; update `Valid()` upper bound to `<= FrameTypePEConnect` | AC-003 / FO-PE-LOOP-001 |
| 2 | `internal/frame/frame.go` (MODIFIED) | Add `frame.ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error)` (new — defined by this story): read `OuterHeaderSize` bytes via `ParseOuterHeader`, then read `hdr.PayloadLen` bytes of payload | AC-001 / Q2 |
| 3 | `internal/frame/frame_test.go` (MODIFIED) | Add `TestFrameType_Valid_PEConnect`: asserts `FrameTypePEConnect.Valid() == true` and `FrameType(0x07).Valid() == false` | AC-003 |
| 4 | `internal/upstreamdial/connector.go` (MODIFIED) | Add `type FrameFn func(hdr frame.OuterHeader, raw []byte) error` (new); add `SetFrameCallback(fn FrameFn)` to `Handle` interface (new); add `frameFn FrameFn` field to `Connector`; add receive goroutine in `dialLoop` after step-3 success: calls `frame.ReadOuterFrame(conn)` in a loop, discriminates `FrameTypePEConnect` (discard) vs all other types (invoke `c.frameFn`); flip bootstrap `ChannelFrame.FrameType` from `halfchannel.FrameTypeData` to `frame.FrameTypePEConnect` (FO-PE-LOOP-001); add direct `internal/frame` import; add per-connection lifecycle sync (WaitGroup or done chan) so `dialLoop` teardown waits for receive goroutine exit | AC-001, AC-002, AC-003, AC-005 / Q1, Q2, Q3, Q6 |
| 5 | `internal/upstreamdial/connector_test.go` (MODIFIED) | Unit tests: `TestConnector_ReceiveLoop_DataFrameForwardedToCallback`, `TestConnector_ReceiveLoop_PEConnectFrameDiscarded`, `TestConnector_ReceiveLoop_ExitsOnConnClose` | AC-001, AC-003, AC-005 |
| 6 | `cmd/switchboard/mgmt_wire.go` (MODIFIED) | Call `connector.SetFrameCallback(fn)` (new — defined by this story) after constructing the `Connector`, with closure `fn = func(hdr frame.OuterHeader, raw []byte) error { return routing.RouteFrame(hdr, raw, router) }` (routing.RouteFrame verified at `8eb54a5`); add `internal/routing` import if not already present on the `mgmt_wire.go` import path | AC-002 / Q1 |
| 7 | `cmd/switchboard/router_pe_receive_test.go` (NEW) | Integration tests: `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect`, `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival`, `TestRunRouter_PE_EFWD001ExhaustionUnderLoad`, `TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop` | AC-001, AC-002, AC-004, AC-005 |
| 8 | `.factory/specs/architecture/ARCH-08-dependency-graph.md` (MODIFIED) | §6.5 update: `internal/upstreamdial` allowed imports `{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}`; must land in the same commit that introduces the `frame.ReadOuterFrame` import in `connector.go` | Q2 / ARCH-08 §6.4 amendment |

---

## Architecture Compliance Rules

- **ARCH-08 §6.6.2 forbidden edges for `internal/upstreamdial`:** `drain`, `routing`,
  `testenv`, and packages at positions 20–23 MUST NOT be imported. The callback seam
  preserves this: `upstreamdial` imports `frame` (position 2) but not `routing` (position 17).
- **ARCH-08 §6.5 import-set extension:** the `frame` import is lawful (position 2 ≤ 19).
  The §6.4 amendment must land in the same commit as the first use of `frame.ReadOuterFrame`
  or `frame.FrameTypePEConnect` in `connector.go`. Build-time violation: if `internal/upstreamdial`
  gains a `routing` import, the build MUST fail (enforced by `ARCH-08 §6.6.2` and
  `go list -deps` verification in the integration test).
- **Pure-core / effectful boundary:** `frame.ReadOuterFrame` (new) is effectful (I/O).
  It belongs in `internal/frame` at position 2 — the position constraint allows effectful
  functions at any layer.
- **go.md rule 12:** `connector.Mode()` reads `c.connectedCount` via `atomic.Load` (no mutex
  needed, established precedent from S-7.04-FU-PE-CONNECTOR). The receive goroutine MUST NOT
  call `c.connectedCount.Add` — the concurrent-transition lesson from F-P29-001 applies.

## Library & Framework Requirements

All imports must use existing module versions pinned in `go.mod` at develop `8eb54a5`. No
new external dependencies are introduced. Internal packages used:

| Package | Import path | Position | Verified at |
|---------|-------------|----------|-------------|
| `frame` | `internal/frame` | 2 | `8eb54a5` |
| `halfchannel` | `internal/halfchannel` | 4 | `8eb54a5` |
| `outerassembler` | `internal/outerassembler` | 8 | `8eb54a5` |
| `arq` | `internal/arq` | 14 | `8eb54a5` |
| `arqsend` | `internal/arqsend` | 15 | `8eb54a5` |
| `routing` | `internal/routing` | 17 | `8eb54a5` |
| `upstreamdial` | `internal/upstreamdial` | 19 | `8eb54a5` |
| `testenv` | `internal/testenv` | 23 | `8eb54a5` |

## File Structure Requirements

New files created by this story:
- `cmd/switchboard/router_pe_receive_test.go` — integration tests for receive loop (AC-001, AC-002, AC-004, AC-005)

Modified files:
- `internal/frame/frame.go` — `FrameTypePEConnect` constant, `Valid()` update, `ReadOuterFrame` function
- `internal/frame/frame_test.go` — `TestFrameType_Valid_PEConnect`
- `internal/upstreamdial/connector.go` — `FrameFn` type, `SetFrameCallback`, receive goroutine, bootstrap flip
- `internal/upstreamdial/connector_test.go` — 3 new unit tests
- `cmd/switchboard/mgmt_wire.go` — `SetFrameCallback` call with routing closure
- `.factory/specs/architecture/ARCH-08-dependency-graph.md` — §6.5 import-set amendment

---

## Estimated Test Surface

Estimated test counts are forecasts; actual delivered count may differ after adversarial hardening.

**`internal/frame/frame_test.go` (MODIFIED — unit):**

| Function | Proves |
|----------|--------|
| `TestFrameType_Valid_PEConnect` | `FrameTypePEConnect.Valid()` == true; `FrameType(0x07).Valid()` == false (bounds not over-widened) |

**`internal/upstreamdial/connector_test.go` (MODIFIED — unit):**

| Function | Proves |
|----------|--------|
| `TestConnector_ReceiveLoop_DataFrameForwardedToCallback` | Data frame received from upstream → FrameFn invoked with correct hdr + payload; FrameTypePEConnect frame NOT forwarded |
| `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` | Frame with FrameTypePEConnect (new — defined by this story) is silently discarded; FrameFn NOT called |
| `TestConnector_ReceiveLoop_ExitsOnConnClose` | Upstream server close → ReadOuterFrame returns EOF → receive goroutine exits → Stop() returns without goroutine leak (`go test -race` clean) |

**`cmd/switchboard/router_pe_receive_test.go` (NEW — integration):**

| Function | Proves |
|----------|--------|
| `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` | Frame from upstream fixture reaches `routing.FrameArrivalHandler.OnFrameArrival` callback chain; `RouterHandle.Mode()` == `testenv.ModePE` |
| `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` | `SetFrameCallback` closure wires to `routing.RouteFrame`; no routing import in `internal/upstreamdial` (`go list -deps` verified) |
| `TestRunRouter_PE_EFWD001ExhaustionUnderLoad` | `arqsend.Retransmitter.Retransmit` dispatch to `routerListenAddr` under path exhaustion → `"E-FWD-001"` in writer output; S404-OBS-F + S404-LOW-1 re-confirmation |
| `TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop` | Connector with live receive goroutine; `connector.Stop()` returns within deadline; no goroutine leak; `go test -race` clean |

**Existing test that must remain unmodified and green:**

| Function | File | Constraint |
|----------|------|------------|
| `TestScanForLine_DetectsEFWD001ProductionEmission` | `cmd/switchboard/router_pe_connector_test.go` | F-P11-001 mutation pin from S-7.04-FU-PE-CONNECTOR — documents `"E-FWD-001"` assertion key; MUST NOT be modified |

**Estimated new test count (forecast):** ~8 net-new (1 `frame_test` + 3 `connector_test` + 4
integration). This is a pre-implementation estimate; adversarial hardening may add additional
regression tests as observed in S-7.04-FU-PE-CONNECTOR (+11 tests above forecast during its
29-pass cycle).

---

## Tasks

1. [ ] Read placement note `decisions/S-BL.PE-RECEIVE-LOOP-placement-note.md` v1.0 and disposition ruling `decisions/S-BL.PE-RECEIVE-LOOP-disposition-ruling.md` v1.0 before writing any code
2. [ ] Update ARCH-08 §6.5: `internal/upstreamdial` import set `{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}` (required in same commit as first `frame` import in `connector.go`)
3. [ ] Add `frame.FrameTypePEConnect = 0x06` constant + update `Valid()` upper bound in `internal/frame/frame.go`
4. [ ] Add `frame.ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error)` to `internal/frame/frame.go`
5. [ ] Add `TestFrameType_Valid_PEConnect` to `internal/frame/frame_test.go` (RED gate)
6. [ ] Add `FrameFn` type + `SetFrameCallback(fn FrameFn)` to `Handle` interface in `internal/upstreamdial/connector.go`
7. [ ] Add receive goroutine in `dialLoop` with `frame.ReadOuterFrame` loop, `FrameTypePEConnect` discrimination, and per-connection lifecycle sync
8. [ ] Flip `dialLoop` bootstrap `ChannelFrame.FrameType` from `halfchannel.FrameTypeData` to `frame.FrameTypePEConnect`
9. [ ] Write unit tests for receive goroutine (AC-001, AC-003, AC-005) — RED gate before step 7
10. [ ] Wire `SetFrameCallback` in `cmd/switchboard/mgmt_wire.go` with `routing.RouteFrame` closure
11. [ ] Write integration tests in `cmd/switchboard/router_pe_receive_test.go` — RED gate before step 10/11
12. [ ] Verify `go test -race -count=1 ./...` full green; `golangci-lint` 0 issues; `gofumpt` no diffs
13. [ ] Verify `TestScanForLine_DetectsEFWD001ProductionEmission` still passes unmodified

---

## Forward Obligations Consumed

| FO ID | Origin | Description | Consumed by | Notes |
|-------|--------|-------------|-------------|-------|
| FO-PE-LOOP-001 | S-7.04-FU-PE-CONNECTOR F-P26-001 (v1.24 deferral) | Define the distinct PE-CONNECT bootstrap frame type (`frame.FrameTypePEConnect`) and flip `dialLoop` bootstrap construction from `halfchannel.FrameTypeData` placeholder; receive loop must discriminate bootstrap from session-data frames | AC-003 + FCL rows 1, 4 | `FrameTypePEConnect = 0x06`; `Valid()` upper bound updated; discrimination: bootstrap frames silently discarded |

---

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 0.1-backlog-stub | 2026-07-07 | Initial backlog stub. Created by PO adjudication F-P1-002 (AC-004 partial-discharge, class unmet-deps on S-7.04-FU-PE-CONNECTOR). No ACs, no FCL. Status: backlog. |
| 1.0 | 2026-07-08 | Elaborated stub → sprint-ready. Governing artifacts: placement note v1.0 (Q1–Q7 architect rulings; all symbols grep-verified at `8eb54a5`), disposition ruling v1.0 (Q-A option (a): BC-2.06.003 is non-discharging prerequisite trace; binding anchor is BC-2.02.008 PC-3/EC-003; Q-B: single story, 5 pts). ACs: AC-001 (receive goroutine active; frames reach OnFrameArrival), AC-002 (runRouter SetFrameCallback wiring), AC-003 (FO-PE-LOOP-001 discharge: FrameTypePEConnect + Valid() + dialLoop flip + discrimination), AC-004 (E-FWD-001 exhaustion integration + S404-OBS-F/S404-LOW-1 re-confirmation), AC-005 (receive goroutine lifecycle/doneCh). Anchors Consumed: BC-2.06.003 PC-1 row corrected from "To discharge" to "Non-discharging prerequisite trace" per disposition ruling v1.0 Q-A. FCL: 8 rows. Estimated test surface: ~8 net-new. FO-PE-LOOP-001 consumed. Version: 1.0; status: ready; points: 5; acceptance_criteria_count: 5. |
