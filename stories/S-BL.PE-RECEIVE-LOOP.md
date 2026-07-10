---
artifact_id: S-BL.PE-RECEIVE-LOOP
document_type: story
level: ops
story_id: S-BL.PE-RECEIVE-LOOP
title: "PE-connection receive/forward loop — frame.ReadOuterFrame goroutine, FrameTypePEConnect constant, and E-FWD-001 exhaustion discharge"
status: ready
producer: story-writer
timestamp: 2026-07-08T00:00:00Z
version: "1.8"
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: PE
points: 5
bc_traces:
  - BC-2.02.008   # PC-3/EC-003 — E-FWD-001 exhaustion discharge (binding anchor; re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 postcondition 1)
  - BC-2.06.003   # PC-1 — non-discharging prerequisite trace; receive loop makes the full send+forward path live for future path-liveness observability
  - BC-2.09.001   # AC-001 anchor, PC-2/PC-3 — upstream connections established; router is in PE mode (non-discharging contextual anchor)
vp_traces: []
subsystems: [deployment-operations, transport-layer]
architecture_modules:
  - internal/frame
  - internal/upstreamdial
  - internal/routing
  - internal/multipath
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
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.001.md'
  - '.factory/stories/S-7.04-FU-PE-CONNECTOR.md'
  - '.factory/decisions/S-BL.PE-RECEIVE-LOOP-placement-note.md'   # v1.7 — Q2 FrameFn return-value contract (discard-and-continue), Q1/Q8 SetFrameCallback ordering contract; F-SP5-001 READ-error disposition (binding), F-SP5-OBS-1 bounded-read divergence (accepted), F-SP5-OBS-2 connector_test.go fixture pattern; F-SP6-001 conn.Close() read-error teardown wiring (binding), F-SP6-002 Option A SetFrameCallback concrete-only, F-SP6-003 AC observables substitutes, F-SP6-004 blast-radius 8→10; F-SP7-001 mode=PE retracted as establishment observable (BINDING), F-SP7-002 accepted-timing corrected (BINDING), F-SP7-004 Task-1 version citation, F-SP7-005 transient stale-ModePE window; F-SP8 remediation is story-side only (placement note v1.7 unchanged this round)
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
emitted exclusively from `OnFrameArrival`. Without a receive loop, an upstream fixture
that writes frames directly to the accepted PE connection, and callback-seam integration,
the path-exhaustion case that exercises E-FWD-001 cannot be reached from a live PE daemon.

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

**Token Budget Estimate (forecast).** Story spec: ~12k tokens (v1.5). Referenced production code
(`connector.go`, `mgmt_wire.go`, `frame.go`, `on_frame_arrival.go`): ~6k tokens.
Test infrastructure (`testenv.go`, existing test patterns, `outerassembler.go` for frame assembly): ~5k tokens.
Total implementing-agent context: ~22k tokens — well within 20–30% of a 200k context window.
No story split required.

## Anchors Consumed

| Anchor | Verbatim ID | Source | Disposition |
|--------|-------------|--------|-------------|
| BC-2.02.008 PC-3/EC-003 — E-FWD-001 fires when only eligible interface is arrival interface | BC-2.02.008 / S404-OBS-F | S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (re-anchored, unmet-deps F-P1-002) | TO DISCHARGE — E-FWD-001 fires (deterministically) via single-interface-set split-horizon block in `FrameArrivalHandler.OnFrameArrival` closure per Q8; upstream fixture (`peWriteFixture`) is the test-internal frame producer per Q9 (arqsend `Dispatch → net.Dial(ListenAddr)` shape superseded — physically disjoint from PE receive goroutine); S404-OBS-F + S404-LOW-1 re-confirmation: "send" = `peWriteFixture.WriteFrame`, "forward attempt" = `OnFrameArrival` through split-horizon (Q9.4 disposition) |
| BC-2.06.003 PC-1 — `status: "failed"` via path liveness failure | BC-2.06.003 | S-7.04-FU-PE-CONNECTOR AC-004 v1.3 (re-anchored, same partial-discharge) | **Non-discharging prerequisite trace.** This story ships the receive goroutine that makes the full send+forward path live. BC-2.06.003 PC-1 `status: "failed"` (path liveness) is NOT discharged here — it requires the keepalive missed-probe mechanism (`internal/paths`), which is orthogonal to E-FWD-001 (split-horizon, `internal/routing`). Future path-liveness observability testing depends on the infrastructure this story ships. (Disposition per S-BL.PE-RECEIVE-LOOP-disposition-ruling.md v1.0 Q-A option (a).) |
| BC-2.09.001 PC-2/PC-3 — upstream connections established; router is in PE mode | BC-2.09.001 | AC-001 anchor (contextual; router is in PE mode with live upstream connections as the precondition for the receive goroutine to be active) | **Non-discharging contextual anchor.** BC-2.09.001 PC-2/PC-3 (router-mode transition and upstream-connection establishment) were discharged by S-7.04-FU-PE-CONNECTOR. This story takes PE mode + established connections as a given precondition. The anchor is cited in AC-001 to establish the precondition context; no new PC-2 or PC-3 discharge obligation arises here. |
| FO-PE-LOOP-001 — define `frame.FrameTypePEConnect`; flip `dialLoop` bootstrap | FO-PE-LOOP-001 | S-7.04-FU-PE-CONNECTOR F-P26-001 (v1.24 deferral) | TO DISCHARGE — `FrameTypePEConnect = 0x06` defined; `Valid()` upper bound updated; ARCH-02 `frame_type` row amended; `dialLoop` bootstrap flipped from `halfchannel.FrameTypeData` placeholder (AC-003) |
| S404-OBS-F — E-FWD-001 rate-limit re-confirmation | S404-OBS-F | STATE.md row; re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 | DISCHARGED via AC-004 — `peWriteFixture.WriteFrame` (upstream fixture) writes assembled frame to accepted PE connection; `OnFrameArrival` through split-horizon = "send+forward" re-confirmation (Q9.4 disposition: arqsend not required; peWriteFixture injection path satisfies the obligation) |
| S404-LOW-1 — live-egress re-confirmation (3 LOW + SEC-001) | S404-LOW-1 | STATE.md row; re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 | DISCHARGED via AC-004 — full send+forward path traversal demonstrated end-to-end via peWriteFixture injection path; same disposition as S404-OBS-F (Q9.4) |

---

## Design Constraints

### Receive Goroutine Ownership and Callback Seam (Q1, Q2)

**Binding (per placement note Q1 and Q2).**

The receive goroutine lives inside `internal/upstreamdial.Connector` (position 19),
one goroutine per established connection, started after step-3 success in `dialLoop`.
`upstreamdial` remains routing-free per the forbidden-edge constraint (ARCH-08 §6.6.2:
`routing` is explicitly listed as a forbidden import for `upstreamdial`).

The seam is a callback on the concrete `*upstreamdial.Connector` type (amended v1.6 — F-SP6-002):

```go
// In internal/upstreamdial (new — defined by this story):
type FrameFn func(hdr frame.OuterHeader, raw []byte) error

// Method on the concrete *Connector ONLY (new — defined by this story):
// SetFrameCallback is NOT added to the Handle interface (F-SP6-002, Option A).
func (c *Connector) SetFrameCallback(fn FrameFn)
```

**`SetFrameCallback` is NOT added to the `upstreamdial.Handle` interface (amended v1.6 — F-SP6-002, Option A).** The `Handle` interface (`ReloadAddrs`/`Mode`/`Stop`) is unchanged. `runRouter` in `cmd/switchboard/mgmt_wire.go` holds the connector as a concrete `*Connector` between `New()` and `Start()` and calls `connector.SetFrameCallback(fn)` there (on the concrete type). `fakeConnectorHandle` in `router_pe_connector_test.go` (implements only `ReloadAddrs`/`Mode`/`Stop`) is NOT affected; `router_pe_connector_test.go` remains existing, unmodified.

The closure passed to `SetFrameCallback` calls `arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, fn)` on a `*routing.FrameArrivalHandler` constructed via `routing.NewFrameArrivalHandler(multipath.NewDropCache(multipath.DefaultDropCacheSize))` with `routing.WithFrameArrivalLogger(routerLogger)` applied (all verified at `8eb54a5`). **Q8 supersedes the original Q1/Q2 `routing.RouteFrame` wiring** — the PE receive path uses `FrameArrivalHandler.OnFrameArrival` rather than `RouteFrame`; the `netingress.Serve` data-plane path retains its existing `RouteFrame` closure unchanged.

### Framing Primitive: frame.ReadOuterFrame (Q2) — byte-contract (v1.3, F-SP3-001)

**Binding (per placement note Q2 v1.3).**

A new function `frame.ReadOuterFrame(r io.Reader) (frame.OuterHeader, []byte, error)` is
added to `internal/frame/frame.go` (position 2). Like `netingress.ReadFrame` (verified at
`8eb54a5` in `internal/netingress/netingress.go`), it returns **payload-only**: read
`frame.OuterHeaderSize` (= 44) bytes via `frame.ParseOuterHeader`, then read
`hdr.PayloadLen` bytes — the `[]byte` return is the payload slice only and does NOT include
the outer header bytes. `netingress.ReadFrame` may delegate to it or retain its own copy
with a cross-reference comment — implementer's choice.

**Receive goroutine full-frame reconstruction (F-SP3-001 correction — replaces any
prior implication that `raw` is payload-only):** Because `FrameFn raw` MUST be the full
wire frame (outer header + payload) per the `OnFrameArrival` contract, the receive goroutine
reconstructs the full frame at the single call site using `frame.EncodeOuterHeader`
(verified at `8eb54a5` in `internal/frame/frame.go` —
`func EncodeOuterHeader(h OuterHeader) [OuterHeaderSize]byte`):

```go
hdr, payload, err := frame.ReadOuterFrame(conn)
if err != nil {
    // READ error: exit the loop regardless of error type.
    // continue-on-read-error is FORBIDDEN (framing desync / busy-loop). (v1.5 — F-SP5-001)
    // BINDING (v1.6 — F-SP6-001): close conn to trigger maintainConn write failure → redial.
    _ = conn.Close()
    return
}
ehdr := frame.EncodeOuterHeader(hdr)
raw := append(ehdr[:], payload...)
_ = frameFn(hdr, raw)  // discard-and-continue; see FrameFn return-value contract (F-SP4-001)
```

The `FrameFn` callback parameter `raw []byte` is therefore ALWAYS the full wire frame
(outer header + payload). This is required because `OnFrameArrival` computes its
drop-cache key as `crc32.ChecksumIEEE(frameBytes)` over the full frame (verified at
`8eb54a5` in `internal/routing/on_frame_arrival.go`). If `raw` were payload-only, two
frames differing only in their outer header (e.g. different `SrcAddr` fields) would
produce identical checksums, causing the second frame to be silently suppressed as a
false loop duplicate. `frame.EncodeOuterHeader` is an EXISTING function at `8eb54a5`
(not new — defined by this story).

**FrameFn return-value contract (v1.4 — F-SP4-001, binding):** A non-nil return value from
`frameFn(hdr, raw)` MUST NOT terminate the receive loop or close the connection. The receive
goroutine MUST discard the error and continue reading the next frame (discard-and-continue
semantics). `OnFrameArrival` returns non-nil on two normal-operation paths:
`ErrAllPathsSplitHorizon` (E-FWD-001, every forwarding candidate is split-horizon blocked) and
`ErrDropCacheHit` (loop-duplicate suppression). Neither is fatal. If the receive loop exited
on the first non-nil return, the pin test
`TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` would fail (frame B is never
read, the second E-FWD-001 emission never fires), defeating the byte-contract validation.

The normative precedent is `netingress.ServeConn` (verified at `8eb54a5`): a non-nil `RouteFn`
return is NOT a signal to close the connection — the error is discarded and the loop continues
(`continue` idiom). `OnFrameArrival` already logs E-FWD-001 and EC-005 internally; the receive
goroutine MUST NOT log the error again (double-count rationale mirrors `netingress.RouteFn`
contract). The correct idiom is:

```go
_ = frameFn(hdr, raw)
```

This satisfies `errcheck` (verified enabled in `.golangci.yml` at `8eb54a5`) — a bare `_ =`
assignment is a legitimate explicit discard. The `//nolint:errcheck` directive MUST NOT be
used. The exit-on-error form is explicitly forbidden:

```go
// FORBIDDEN — exits the loop on E-FWD-001, defeating TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader
if err := frameFn(hdr, raw); err != nil {
    return
}
```

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
hdr, payload, err := frame.ReadOuterFrame(conn)
if err != nil {
    // READ error: exit the loop regardless of error type. (v1.5 — F-SP5-001)
    // continue-on-read-error is FORBIDDEN (framing desync / busy-loop).
    // BINDING (v1.6 — F-SP6-001): close conn → maintainConn write failure → dialLoop redial.
    _ = conn.Close()
    return
}
// Discrimination step runs only on successful reads:
ehdr := frame.EncodeOuterHeader(hdr)        // reconstruct full frame (header + payload)
raw := append(ehdr[:], payload...)
if hdr.FrameType == frame.FrameTypePEConnect {
    // bootstrap acknowledgment: silent discard (no reply defined in this story's scope)
} else {
    // session data / ctl / arq / fec frame: pass to FrameFn callback
    _ = frameFn(hdr, raw)   // discard-and-continue; non-nil return MUST NOT terminate loop (F-SP4-001)
}
```

Bootstrap frames (type 0x06) are silently discarded. All other frame types are forwarded
to the caller-supplied `FrameFn`. The `raw` argument to `FrameFn` is ALWAYS the full wire
frame (outer header + payload) — never payload-only. `frame.EncodeOuterHeader` is an
existing function at `8eb54a5` (not new — defined by this story).

### arqsend.Retransmitter Wiring (Q4 — production-wiring ruling retained; test role superseded by Q9)

**Binding (per placement note Q4 and Q9).**

**Production wiring (Q4 ruling — RETAINED):** `arqsend.New` is NOT wired into the
production `runRouter` datapath for this story. The production ARQ retransmit path is
node-side, not router-side. A persistent `Retransmitter` instance in `runRouter` would
be production-dead code outside this story's scope.

**Test-internal frame production (Q9 ruling — supersedes Q4 dispatch shape):**
`arqsend.Retransmitter` is NOT the frame producer in the AC-004 E-FWD-001 integration
test. The Q4 shape (`arqsend.Dispatch` dialing `routerListenAddr` via `net.Dial`) is
physically disjoint from the dialed PE connection where the receive goroutine lives —
those bytes enter via `netingress.Serve → RouteFrame`, bypassing `OnFrameArrival`
entirely. Q9 rules this injection path undischargeable.

**Correct injection path (Q9):** the upstream PE fixture (`peWriteFixture`, test-local
in `cmd/switchboard/router_pe_receive_test.go` — new defined by this story) writes an
assembled outer frame directly to the accepted PE connection. The frame must use a
non-bootstrap `FrameType` (e.g. `frame.FrameTypeData`) so it passes the
`FrameTypePEConnect` discard check in the receive goroutine and reaches
`arrivalHandler.OnFrameArrival`.

Frame assembly in the test:

```go
fixture := startPEWriteFixture(t)  // new — defined by this story
// cfg.UpstreamRouters points at fixture.addr
wire, err := outerassembler.Assemble(
    halfchannel.ChannelFrame{
        FrameType: frame.FrameTypeData,  // non-bootstrap → reaches OnFrameArrival
        ChanID:    1,
        ChanSeq:   1,
        Payload:   []byte{0x01},
    },
    [outerassembler.SACKBitmapSize]byte{},
    outerassembler.Envelope{},           // zero env — HMAC bypass per Q8 §8.5
)
// outerassembler.Assemble: func(cf halfchannel.ChannelFrame, sackBitmap [SACKBitmapSize]byte, env Envelope) ([]byte, error)
// outerassembler.SACKBitmapSize, halfchannel.ChannelFrame, frame.FrameTypeData — all verified at 8eb54a5
if err != nil { t.Fatalf("Assemble: %v", err) }
fixture.WriteFrame(t, wire)  // new — defined by this story
```

No `arqsend`, `arq`, or `net.Dial(routerListenAddr)` in the AC-004 test body.
`internal/arqsend` is removed from the story's `architecture_modules` frontmatter list.

### Receive Goroutine Lifecycle and doneCh Contract (Q6)

**Binding (per placement note Q6).**

The receive goroutine is owned by `dialLoop` and exits when the per-address connection is
closed (`conn.Close()` called by `dialLoop` teardown causes `frame.ReadOuterFrame` to return
`io.EOF` or a net error). No separate stop channel is needed; the goroutine drains naturally
on conn close.

**Exactly-once semantics (F-P29-001 lesson applied symmetrically):** the receive goroutine
MUST NOT access `c.connectedCount` or any other shared state. It has exactly **two outputs**
(amended v1.6 — F-SP6-001): (1) calling the `FrameFn` callback with received bytes; (2)
calling `_ = conn.Close()` on read-error exit to trigger `maintainConn` write failure →
`dialLoop` teardown → reconnect. `conn.Close()` ownership: `dialLoop` step 8 (normal
teardown) OR receive goroutine (abnormal read-error exit); double-close is safe/idempotent
on `net.Conn`.

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

### READ-error disposition contract (v1.5 — F-SP5-001, binding; amended v1.6 — F-SP6-001)

**Binding (per placement note F-SP5-001 v1.5; conn.Close() wiring per F-SP6-001 v1.6).**

On ANY non-nil return from `frame.ReadOuterFrame`, the receive goroutine MUST (1) call
`_ = conn.Close()` and (2) exit the loop (`return`). `continue`-on-read-error is FORBIDDEN
— this is the exact mirror of the v1.4 callback-error return-FORBIDDEN rule. The per-site
disposition follows the `netingress.ServeConn` precedent (verified at `8eb54a5`):

> read error → **call `_ = conn.Close()`** then **exit** the loop (return)
> callback error → **continue** (discard-and-continue)

Rationale: `continue`-on-read-error produces one of two failure modes:
1. **Busy-loop on conn-close EOF** — `frame.ReadOuterFrame` returns `io.EOF` on every
   iteration; the goroutine never exits; `Connector.Stop()` blocks on the per-reconnect
   join forever; AC-005 leak tests hang.
2. **Permanent framing desync on malformed frame** — if a semi-trusted upstream sends a
   malformed frame (`ErrInvalidFrameType` or truncation `io.ErrUnexpectedEOF`) WITHOUT
   closing the conn, every subsequent 44-byte header read consumes mid-frame garbage.
   `maintainConn` keepalive writes still succeed (full-duplex), so the conn is never
   torn down and never reconnected. The connection is permanently desynced.

Exit with `_ = conn.Close()` → `maintainConn`'s next write fails → `dialLoop` teardown
and re-dial, which is the ONLY correct resync for a byte-misaligned stream. (amended v1.6 —
F-SP6-001: the v1.5 "dialLoop's existing teardown/reconnect path" phrasing is corrected —
`maintainConn` is write-only and never observes read-goroutine exit; `_ = conn.Close()` is
the explicit wiring that converts the read-side failure into a write-side event.)

**Logging disposition:** Two cases:
- **Clean exit** (`io.EOF` at a frame boundary, or any read error when `ctx.Err() != nil`
  — conn-close during `Stop()`/reconnect teardown): **silent exit, no log**. These are
  expected lifecycle events; the double-count constraint does NOT apply because
  `OnFrameArrival` never saw the frame.
- **Abnormal read error** (parse error such as `ErrInvalidFrameType`, truncation
  `io.ErrUnexpectedEOF`, or net error other than context cancellation): **one log line
  permitted** at the implementer's discretion before returning. The v1.4 double-count
  constraint does NOT apply here because `OnFrameArrival` never received the frame. A
  silent exit is also acceptable given that `dialLoop` will log EC-001 on the subsequent
  redial failure if the connection is truly broken. The implementer MUST NOT log on the
  clean-exit path.

**Receive-goroutine sketch (updated — v1.6, replaces v1.5 sketch — F-SP6-001):**

```go
for {
    hdr, payload, err := frame.ReadOuterFrame(conn)
    if err != nil {
        // READ error: exit the loop regardless of error type.
        // continue-on-read-error is FORBIDDEN (framing desync / busy-loop).
        // BINDING (v1.6 — F-SP6-001): close the conn to trigger maintainConn
        // write failure → dialLoop teardown → backoff → redial.
        // Double-close is safe/idempotent on net.Conn.
        _ = conn.Close()
        return
    }
    ehdr := frame.EncodeOuterHeader(hdr)
    raw := append(ehdr[:], payload...)
    if hdr.FrameType == frame.FrameTypePEConnect {
        // bootstrap acknowledgment: silent discard
        continue
    }
    _ = frameFn(hdr, raw)  // discard-and-continue; non-nil return MUST NOT terminate loop (F-SP4-001)
}
```

### Reconnect latency bound and AC-005 timeout guidance (v1.6 — F-SP6-001)

After the receive goroutine calls `_ = conn.Close()`, redial is initiated within ≤
`keepaliveInterval` (next keepalive tick's `SetWriteDeadline` + `conn.Write` fails) plus
backoff. Backoff resets to `operativeBase(keepaliveInterval)` on each successful connect
(verified at `8eb54a5` in `dialLoop`), so after a connection that subsequently fails
(malformed-frame-then-close scenario) the next redial begins at `operativeBase` delay
(keepaliveInterval, floored at `BackoffBase = 500ms`). **`TestConnector_ReceiveLoop_ExitsOnReadError`
timeout MUST accommodate `keepaliveInterval` + `operativeBase` backoff**; tests SHOULD use a
short `keepaliveInterval` (10–20ms, consistent with the existing `connector_test.go` pattern
at `8eb54a5`). A repeatedly malformed upstream produces at most one reconnect per
`operativeBase` interval — the malformed-frame reconnect-storm risk is bounded.

**Transient stale-ModePE window (v1.7 — F-SP7-005, accepted):** After the receive goroutine
calls `_ = conn.Close()` and exits, `connectedCount.Add(-1)` has NOT yet fired — `maintainConn`
must observe its next write failure first, then `dialLoop` decrements the count. During this
window, `Mode()` transiently reports `ModePE` for up to `keepaliveInterval` after the receive
goroutine exits. This is accepted with no AC obligation: no AC in this story asserts `Mode()`
during this window, and no `FrameFn` consumer runs after the receive goroutine exits. The
transient is bounded by `keepaliveInterval`. Future stories asserting `Mode()` after deliberate
teardown MUST account for this window. (v1.7 — F-SP7-005)

### Bounded-read divergence (v1.5 — F-SP5-OBS-1, accepted with rationale)

No `LimitReader` or read deadline is applied on the PE receive path. The divergence from the
`netingress.ServeConn` `io.LimitReader` pattern is accepted with rationale:
1. `PayloadLen` is `uint16` — maximum frame allocation is 44 + 65 535 = 65 579 bytes per
   frame. This is a hard codec-level bound; a malformed `PayloadLen = 0xFFFF` allocates at
   most ~64 KB with no amplification possible.
2. The PE upstream connection is a DIALED connection to a configured, semi-trusted upstream
   router — not an arbitrary accepted connection from an unknown client (the `netingress`
   threat model). The upstream address is operator-controlled.
3. The READ-error exit contract (F-SP5-001) ensures any malformed frame causes immediate
   connection teardown and reconnect — the allocation is bounded per connection, not per-attack-loop.

No implementation change required. Observation recorded per placement note F-SP5-OBS-1.

### SetFrameCallback Ordering Contract (v1.4 — F-SP4-002, binding)

**Binding (per placement note Q1/Q8 v1.4).**

`SetFrameCallback` MUST be called before `Start()`. The `frameFn` field on `Connector` is
set-once pre-launch. The `happens-before` edge created by goroutine creation (Go memory model
§"Goroutine creation") guarantees that any `frameFn` value written before `Start()` is visible
to all goroutines launched by `Start()`. No additional field synchronization (mutex, atomic) is
required for this field, because it is written exactly once before any reader goroutine is created.

**Production wiring order in `runRouter` (binding):**

```
construct → SetFrameCallback → Start
```

Concretely in `mgmt_wire.go` — the current code at `8eb54a5` has
`connector := upstreamdial.New(...)` immediately followed by `connector.Start()`; this story
inserts `connector.SetFrameCallback(frameFn)` between those two lines:

```go
connector := upstreamdial.New(w, outerassembler.Envelope{}, keepaliveInterval, upstreamRouters)
connector.SetFrameCallback(frameFn)  // MUST precede Start
connector.Start()
```

The receive goroutine MAY assume `frameFn` is non-nil under this ordering contract (the field
is guaranteed visible and non-nil before the goroutine is created).

**Nil-guard posture (defense-in-depth):** as a belt-and-suspenders guard against future callers
that construct a `Connector` without calling `SetFrameCallback` before `Start`, the receive
goroutine SHOULD apply a nil check before invoking the callback and silently discard the frame
if `frameFn` is nil (no log emission — logging every discarded frame would be noise without
context). This does NOT replace the ordering obligation; a nil `frameFn` at receive time is a
programming error, not an expected condition.

**Post-Start mutation is forbidden.** Any call to `SetFrameCallback` after `Start()` returns is
a data race (dial goroutines are already reading `frameFn`). The `Connector` implementation MUST
NOT permit it — it may panic or silently ignore the call, but MUST NOT proceed with an
unsynchronized field write.

### Test Harness Rule: runRouter Goroutine Pattern Mandatory for OnFrameArrival ACs (Q9 / F-SP2-003)

**Binding (per placement note Q9.3).**

Every AC that asserts `OnFrameArrival` is reached — specifically AC-001, AC-002, and
AC-004 — MUST use the real `runRouter` goroutine pattern, NOT `testenv.New`/`Restart`.

`testenv.Restart` (verified at `8eb54a5` in `internal/testenv/testenv.go`) builds a bare
`upstreamdial.New(...).Start()` without calling `SetFrameCallback`. With no callback
registered, `OnFrameArrival` is never invoked; E-FWD-001 never fires. A test using
`testenv.Restart` for these ACs would pass trivially for the wrong reason.

The real `runRouter` goroutine pattern:

```go
buf := &syncBuffer{}
ctx, cancel := context.WithCancel(context.Background())
errCh := make(chan error, 1)
go func() {
    errCh <- runRouter(ctx, buf, cfg, cfgPath, nil)
}()
t.Cleanup(func() {
    cancel()
    select { case <-errCh: case <-time.After(3 * time.Second): }
})
```

`runRouter` is the code path that constructs the `FrameArrivalHandler` and calls
`connector.SetFrameCallback(frameFn)` per the Q8 ruling. This pattern is already
established in `router_pe_connector_test.go`; `router_pe_receive_test.go` MUST
follow the same pattern for all `OnFrameArrival`-asserting tests.

**AC-005 harness adjudication (F-SP3-002 ruling):** AC-005 asserts goroutine lifecycle
behavior — that the receive goroutine exits cleanly and `Stop()` returns without leak
across a flap cycle. AC-005 does NOT assert that `OnFrameArrival` is reached; it asserts
only that the goroutine terminates and the per-reconnect-iteration join prevents
accumulation. Per the F-SP3-002 ruling (placement note v1.3, Q6 annotation and Q9 §9.5
item 5): the AC-005 flap-cycle test lives in `internal/upstreamdial/connector_test.go`,
NOT in `router_pe_receive_test.go`. The test uses a hand-rolled
`heldConn`+`Close()` harness following the existing `TestConnector_BackoffParameters`
pattern (verified at `8eb54a5`): fresh `Connector` + `SetFrameCallback` with a counting
`FrameFn`; Phase 1 connect + frame delivered; Phase 2 server-side `conn.Close()` to
trigger reconnect; Phase 3 second listener accepts redial + frame delivered again;
Phase 4 `Stop()` completes within timeout; no-goroutine-leak assertion.
`peWriteFixture` is NOT involved in AC-005 — the flap harness is entirely self-contained
in `connector_test.go`. AC-005 does not require the `runRouter` goroutine pattern; the
harness rule applies only to tests asserting `OnFrameArrival` (AC-001, AC-002, AC-004).

---

## Acceptance Criteria

### AC-001 — Receive goroutine active per established PE connection; incoming frames reach FrameArrivalHandler

**BC Anchors:** BC-2.09.001 PC-2/PC-3 (upstream connections established; router is in PE
mode); placement note Q1, Q2, Q8 (traces to BC-2.09.001 PC-2, PC-3).

**Precondition:** The test uses the real `runRouter` goroutine pattern (see Q9.3 harness
rule above — `testenv.Restart` MUST NOT be used here). A `peWriteFixture` (new — defined
by this story in `cmd/switchboard/router_pe_receive_test.go`) is started; `cfg.UpstreamRouters`
points at `fixture.addr`. `runRouter` constructs the `Connector`, calls
`connector.SetFrameCallback(fn)` (new — defined by this story) with a closure routing
through `routing.FrameArrivalHandler.OnFrameArrival` (verified at `8eb54a5` in
`internal/routing/on_frame_arrival.go`) on an `arrivalHandler` constructed via
`routing.NewFrameArrivalHandler(multipath.NewDropCache(multipath.DefaultDropCacheSize))`
(all verified at `8eb54a5`; see Q8 ruling), and starts the connector. The fixture's
`accepted` channel receives the connector's dialed connection.

**Postconditions:**

1. After `dialLoop` step-3 success, a receive goroutine is started on the established
   `net.Conn`. The goroutine calls `frame.ReadOuterFrame(conn)` (new — defined by this
   story) in a loop.
2. `peWriteFixture.WriteFrame(t, wire)` (new — defined by this story) writes a
   pre-assembled outer frame (assembled via `outerassembler.Assemble` with
   `frame.FrameTypeData` — non-bootstrap, passes the `FrameTypePEConnect` discard
   check) to the accepted PE connection. The PE receive goroutine's
   `frame.ReadOuterFrame(conn)` call returns `(hdr, payload, nil)` where `payload` is
   payload-only. The goroutine then reconstructs the full wire frame via
   `ehdr := frame.EncodeOuterHeader(hdr)` + `raw := append(ehdr[:], payload...)`
   (`frame.EncodeOuterHeader` is an existing function at `8eb54a5`, not new — F-SP3-001
   byte-contract ruling), and invokes the `FrameFn` callback with the full-frame `raw`.
   The callback calls
   `arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, fn)`
   (verified at `8eb54a5` in `internal/routing/on_frame_arrival.go`). The `arrivalHandler`
   is a `*routing.FrameArrivalHandler` constructed by `runRouter` via
   `routing.NewFrameArrivalHandler` and configured with `routing.WithFrameArrivalLogger`
   (both verified at `8eb54a5`). **Note:** the PE receive path bypasses `routing.RouteFrame`'s
   HMAC admission check — PE upstream connections are established outbound by the connector
   itself (bootstrap handshake via `dialLoop`), not arbitrary ingress; this is acceptable for
   this story's scope. Flagged as a security-review item for the PR (SEC follow-on per
   disposition convention); admission-on-PE-receive is revisited in the DRAIN-WIRE/session-
   bootstrap era. `cmd/switchboard/mgmt_wire.go` gains an `internal/multipath` import (new
   at this layer — lawful as `cmd/switchboard` is at the top of the DAG; verified at
   `8eb54a5` that `multipath` is currently absent from `mgmt_wire.go` imports).
3. PE establishment is confirmed via an observable substitute (amended v1.6 — F-SP6-003;
   further corrected v1.7 — F-SP7-001 + F-SP7-002; amended v1.8 — F-SP8-001: live text
   restructured for coherence — `peWriteFixture.accepted` receipt is THE establishment gate
   for this PC; mode=PE demoted to do-not-use note below):
   `connector.Mode()` is unassertable from the `runRouter` goroutine harness (connector
   is an unexported local). `peWriteFixture.accepted` receipt is THE establishment gate
   for AC-001 PC-3: when the `accepted` channel receives a value, the connector has dialed
   and the TCP session is open ~~(connector has completed step 3, atomically incrementing
   `connectedCount`, the same event that makes `Mode()` return `ModePE`)~~
   **(amended v1.7 — F-SP7-002: RETRACTED — `accepted` fires at TCP-accept, strictly
   BEFORE `connectedCount.Add(1)`; it is an early/approximate establishment gate, NOT
   a `ModePE` assertion)**. This matches AC-004's already-coherent phrasing (which uses
   `peWriteFixture.accepted` receipt as the sole precondition gate).
   **Do not use `"mode=PE"` as an establishment gate:** ~~the `"mode=PE"` writer-output
   line via the existing `waitForConnections`/`scanForLine` pattern in
   `router_pe_connector_test.go` (verified at `8eb54a5`) — this line is emitted by
   `runRouter` on PE-mode transition and fires after `connectedCount.Add(1)`, making it
   the stronger guarantee if a strict ordering is required~~ **(amended v1.7 —
   F-SP7-001: RETRACTED — `"mode=PE"` is a PE-CONFIG PRESENCE signal only, emitted when
   `len(upstreamRouters) > 0` at startup or SIGHUP, strictly before any dial attempt; it
   fires even against an unreachable upstream)** — the `"mode=PE"` writer line asserts
   PE-CONFIG PRESENCE only and MUST NOT be used as an establishment gate (per the binding
   three-observable table below). `connector.Mode()` direct assertions are valid only in
   `connector_test.go` unit tests (in-package, concrete `*Connector` type).

   **Binding three-observable semantics (v1.7 — F-SP7-001 + F-SP7-002):**

   | Observable | What it proves | Correct use |
   |---|---|---|
   | `"mode=PE"` in writer output | PE-CONFIG PRESENCE: `len(upstreamRouters) > 0` at startup/SIGHUP only | Use ONLY to assert PE config was applied. **MUST NOT be used as an establishment gate.** |
   | `peWriteFixture.accepted` receive | TCP-accept-level establishment — TCP session open, strictly BEFORE `connectedCount.Add(1)` | Use as the establishment gate for AC-001 PC-3 and AC-004 precondition. Sufficient for "connector has dialed the upstream". |
   | Frame arrival on `FrameFn` / E-FWD-001 emission | Receive-goroutine is live and forwarding frames | The ONLY true establishment + liveness observable. Required for ACs asserting the receive loop is active. |

**Test names:**

- `TestConnector_ReceiveLoop_DataFrameForwardedToCallback` (unit, `internal/upstreamdial/connector_test.go` — sends a data frame on the upstream fixture side using the in-package accept-and-write pattern: local `net.Listen` + accept + `outerassembler.Assemble` + `conn.Write`; asserts FrameFn callback invoked with the correct hdr + payload; same fixture pattern as the AC-005 flap harness and the F-SP5-001 read-error pin test — no shared helper, pattern duplicated per-test or in a test-local helper at implementer's discretion) **(v1.5 — F-SP5-OBS-2)**
- `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` (integration, `cmd/switchboard/router_pe_receive_test.go` — starts testenv PE router; sends well-formed frame from upstream fixture; asserts OnFrameArrival path is reached via writer output change or log event)

**Test level:** unit (Connector callback) + integration (runRouter end-to-end)
**Test files:** `internal/upstreamdial/connector_test.go`, `cmd/switchboard/router_pe_receive_test.go`

---

### AC-002 — runRouter constructs FrameArrivalHandler and wires SetFrameCallback closure through OnFrameArrival (Q8)

**BC Anchors:** BC-2.02.008 PC-3 (frame routing path is live; traces to BC-2.02.008 PC-3);
placement note Q8.

**Precondition:** The test uses the real `runRouter` goroutine pattern (Q9.3 harness rule —
`testenv.Restart` MUST NOT be used here). A `peWriteFixture` (new — defined by this story)
is started; `cfg.UpstreamRouters` points at `fixture.addr`. `runRouter` has executed
Phase b (router construction via `buildRouter`) and the `Connector` has been constructed
via `upstreamdial.New` (verified at `8eb54a5`).

**Postconditions:**

1. `runRouter` constructs a `*multipath.DropCache` via
   `multipath.NewDropCache(multipath.DefaultDropCacheSize)` (verified at `8eb54a5`:
   `NewDropCache(capacity int) *DropCache`; `DefaultDropCacheSize = 10_000`), then constructs
   a `*routing.FrameArrivalHandler` via `routing.NewFrameArrivalHandler(dc)` (verified at
   `8eb54a5`), and applies `routing.WithFrameArrivalLogger(routerLogger)` (verified at
   `8eb54a5`). These constructions occur after Phase b (router built) and before Phase c
   (connector started).
2. `runRouter` calls `connector.SetFrameCallback(fn)` (new — defined by this story) with a
   `FrameFn` closure that calls:
   `arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, fn)`
   where `peIfaceID` is a `routing.InterfaceID` (verified at `8eb54a5` — `type InterfaceID uint64`)
   assigned at construction time, and `fn` is a `routing.ForwardFunc` (verified at `8eb54a5` —
   `type ForwardFunc func(iface InterfaceID, frameBytes []byte) error`). `OnFrameArrival`
   signature (verified at `8eb54a5`):
   `func (h *FrameArrivalHandler) OnFrameArrival(frameBytes []byte, arrivalIface InterfaceID, interfaceSet []InterfaceID, fn ForwardFunc) error`.
   With `interfaceSet == []routing.InterfaceID{peIfaceID}` (arrival interface is the only
   candidate), `SplitHorizon.Forward` finds no eligible output interface → E-FWD-001 fires
   on every non-bootstrap data frame. This makes AC-004's exhaustion test deterministic:
   E-FWD-001 fires because the split-horizon topology guarantees no forwarding path (single-
   interface set), not because of load-dependent path-count exhaustion.
   **Insertion point (v1.4 — F-SP4-002):** `connector.SetFrameCallback(fn)` is inserted
   between the existing `upstreamdial.New(...)` and `connector.Start()` lines in `runRouter`
   — the construct → SetFrameCallback → Start ordering is binding. See FCL row 6 and the
   SetFrameCallback Ordering Contract section above.
3. No `routing` import is introduced in `internal/upstreamdial` — the callback seam
   preserves ARCH-08 §6.6.2 forbidden-edge constraint. The `netingress.Serve` data-plane
   accept loop in `runRouter` retains its existing wiring unchanged (per Q8.4 — the
   `FrameArrivalHandler` path is strictly the PE upstream receive goroutine).
4. `cmd/switchboard/mgmt_wire.go` gains an `internal/multipath` import (only new production
   import at this layer). No ARCH-08 §6.4 registration required for `cmd/switchboard`.

**Test names:**

- `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` (integration, `cmd/switchboard/router_pe_receive_test.go` — sends a data frame on the upstream side; asserts `OnFrameArrival` path is reached, e.g. via `"E-FWD-001"` or routing-activity log event; confirms no routing import in `internal/upstreamdial` via `go list -deps`)

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
- `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` (unit, `internal/upstreamdial/connector_test.go` — sends a FrameTypePEConnect frame on upstream side using the in-package accept-and-write pattern: local `net.Listen` + accept + `outerassembler.Assemble` + `conn.Write`; asserts FrameFn callback is NOT invoked; reuses the same fixture pattern as the AC-001 unit test and the F-SP5-001 read-error pin test — `peWriteFixture` stays test-local to `cmd/switchboard`; no new shared helper) **(v1.5 — F-SP5-OBS-2)**

**Test level:** unit (both tests)
**Test files:** `internal/frame/frame_test.go`, `internal/upstreamdial/connector_test.go`

---

### AC-004 — E-FWD-001 split-horizon discharge (BC-2.02.008 PC-3/EC-003); S404-OBS-F and S404-LOW-1 re-confirmation

**BC Anchors:** BC-2.02.008 PC-3/EC-003 (split-horizon drop + E-FWD-001 event logged —
binding anchor per disposition ruling v1.0; traces to BC-2.02.008 PC-3, EC-003); S404-OBS-F;
S404-LOW-1; placement note Q5, Q8.

**S404-OBS-F and S404-LOW-1 note:** Both drift anchors (re-confirmed at live egress via
the full send+forward path) are discharged by this AC. The `"E-FWD-001"` emission in the
writer output IS the re-confirmation vehicle for both. The "send" is
`peWriteFixture.WriteFrame`; the "forward attempt" is `OnFrameArrival` routing through
the split-horizon path (Q9.4 disposition — arqsend not required).

**Exhaustion mechanism (Q8 ruling):** E-FWD-001 fires because the `FrameFn` closure wired in
`runRouter` passes `interfaceSet == []routing.InterfaceID{peIfaceID}` — the arrival interface
is the sole candidate. `SplitHorizon.Forward` (verified at `8eb54a5` in
`internal/routing/split_horizon.go`) finds no eligible output interface and emits
`ErrAllPathsSplitHorizon` → E-FWD-001 logs. This mechanism is deterministic: the split-horizon
block fires on every non-bootstrap frame because the single-interface set always exhausts,
regardless of load.

**Injection topology (Q9 ruling — supersedes prior arqsend dispatch shape):**
The upstream PE fixture (`peWriteFixture`, new — defined by this story) writes an
assembled outer frame directly to the accepted PE connection. `arqsend.Retransmitter` is
NOT the frame producer. No `net.Dial(routerListenAddr)` dispatch closure is used.

**HMAC bypass note:** Because the PE receive `FrameFn` routes directly to
`OnFrameArrival` (bypassing `RouteFrame`'s HMAC admission check), test frames from the
fixture do NOT need a valid HMAC to reach `OnFrameArrival`. A zero `outerassembler.Envelope`
is sufficient. This is acceptable — PE upstream connections are established outbound by the
connector itself, not arbitrary ingress.
**Flagged as a SEC follow-on for the PR** (admission-on-PE-receive revisited in the
DRAIN-WIRE/session-bootstrap era per Q8 ruling).

**Precondition:** The test uses the real `runRouter` goroutine pattern (Q9.3 harness rule —
`testenv.New`/`Restart` MUST NOT be used here; `testenv.Restart` never calls
`SetFrameCallback`). A `peWriteFixture` (new — defined by this story) is started via
`fixture := startPEWriteFixture(t)`. `cfg.UpstreamRouters` points at `fixture.addr`.
`runRouter` is launched as a goroutine; PE establishment is confirmed via
`peWriteFixture.accepted` receive — when the channel receives, the connector has dialed
and the TCP session is open (early/approximate gate; sufficient to proceed with WriteFrame)
**(amended v1.7 — F-SP7-001: the prior `"mode=PE"` writer-output line poll is
RETRACTED as an establishment gate; `"mode=PE"` is a PE-config-presence signal emitted
at startup/SIGHUP before any dial attempt, not an establishment observable; use
`peWriteFixture.accepted` receipt as the precondition gate per the v1.7 binding ruling)**
(amended v1.6 — F-SP6-003; `connector.Mode()` is unassertable from the harness as the
connector is an unexported local).

**Postconditions:**

1. `peWriteFixture.WriteFrame(t, wire)` (new — defined by this story) writes a
   pre-assembled outer frame to the accepted PE connection, where `wire` is assembled via:
   `outerassembler.Assemble(halfchannel.ChannelFrame{FrameType: frame.FrameTypeData, ChanID: 1, ChanSeq: 1, Payload: []byte{0x01}}, [outerassembler.SACKBitmapSize]byte{}, outerassembler.Envelope{})`
   (all symbols verified at `8eb54a5`; `Assemble` form:
   `func(cf halfchannel.ChannelFrame, sackBitmap [SACKBitmapSize]byte, env Envelope) ([]byte, error)`).
   The PE receive goroutine reads the frame via `frame.ReadOuterFrame(conn)` (returns
   payload-only), reconstructs the full wire frame via
   `ehdr := frame.EncodeOuterHeader(hdr)` + `raw := append(ehdr[:], payload...)`
   (`frame.EncodeOuterHeader` is an existing function at `8eb54a5` — F-SP3-001 byte-contract
   ruling), and passes the full-frame `raw` to the `FrameFn` closure. The closure calls
   `arrivalHandler.OnFrameArrival` with `interfaceSet = []routing.InterfaceID{peIfaceID}`.
   Because the arrival interface is the only forwarding candidate, `SplitHorizon.Forward`
   returns `ErrAllPathsSplitHorizon` (verified at `8eb54a5`) and the router's writer output
   contains the string `"E-FWD-001"`.
   This is the spec-anchored event code (F-P11-001 lesson from S-7.04-FU-PE-CONNECTOR:
   do NOT assert `"split-horizon-blocked"` or `"all paths split-horizon"` — the event code
   tag is stable across prose rewording). The production emission string (verified at
   `8eb54a5` in `internal/routing/on_frame_arrival.go`) is:
   `"all paths split-horizon-blocked: frame dropped (checksum=0x%08x iface=%d) (BC-2.02.008 E-FWD-001)"`.
   The assertion key `"E-FWD-001"` resolves against this production string.
2. E-FWD-001 fires on the first non-bootstrap frame written — the exhaustion is
   topologically guaranteed by the single-interface set, not load-dependent. A single
   `peWriteFixture.WriteFrame` call is sufficient to trigger the assertion.
3. The `TestScanForLine_DetectsEFWD001ProductionEmission` mutation pin (verified at
   `8eb54a5` in `cmd/switchboard/router_pe_connector_test.go`) validates that `"E-FWD-001"`
   detects the production emission string. This test MUST remain unmodified and green.

**Test names:**

- `TestRunRouter_PE_EFWD001ExhaustionUnderLoad` (integration, `cmd/switchboard/router_pe_receive_test.go` — runRouter goroutine pattern; peWriteFixture writes assembled outer frame directly to accepted PE connection; assert "E-FWD-001" in writer output; re-confirms S404-OBS-F + S404-LOW-1 via peWriteFixture injection path per Q9)
- `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` (**byte-contract pin test + loop-continuation pin**, `cmd/switchboard/router_pe_receive_test.go` — two frames assembled with identical payload but differing `OuterHeader.SrcAddr` ([8]byte `0x01...` vs `0x02...`); both `frame.FrameTypeData` (non-bootstrap); assert ≥2 `"E-FWD-001"` emissions in writer output; proves full-frame reconstruction is wired: payload-only `crc32` would collide on identical payloads → false-duplicate suppression → only 1 emission; F-SP3-001 per placement note Q9 §9.1a; **[v1.4 F-SP4-001 annotation]** requiring 2 E-FWD-001 emissions means the receive loop MUST continue after the first `frameFn` invocation returns `ErrAllPathsSplitHorizon` — the ≥2-emission observable IS the observable loop-continuation pin; a loop that exits on non-nil `frameFn` return would yield only 1 emission and fail the assertion)
- `TestScanForLine_DetectsEFWD001ProductionEmission` (**existing normative pin** in `cmd/switchboard/router_pe_connector_test.go` — must remain unmodified and green; documents the `"E-FWD-001"` assertion key)

**Test level:** integration (`router_pe_receive_test.go`) + existing normative pin (must stay green)
**Test files:** `cmd/switchboard/router_pe_receive_test.go`, `cmd/switchboard/router_pe_connector_test.go` (existing, unmodified)

---

### AC-005 — Receive goroutine lifecycle: per-reconnect join, doneCh ordering, Stop() blocks until all receive goroutines return

**BC Anchors:** Q6 ruling (placement note v1.1, F-SP1-005 per-reconnect join requirement);
F-P29-001 concurrent-transition lesson (S-7.04-FU-PE-CONNECTOR pass-29; traces to Q6
concurrency contract).

**Precondition:** A `Connector` is running with ≥1 established PE upstream connection
(receive goroutine active). `connector.Mode()` returns `upstreamdial.ModePE` (verified at
`8eb54a5`).

**Postconditions:**

1. When `conn.Close()` is called by `dialLoop` teardown (via context cancellation or
   `Stop()`), the receive goroutine's `frame.ReadOuterFrame(conn)` (new — defined by this
   story) call returns `io.EOF` or a net error, and the goroutine exits. The per-address
   `done chan struct{}` is NOT closed until the receive goroutine has confirmed exit (via
   `sync.WaitGroup` or per-connection done signal).
2. **Per-reconnect-iteration join (Q6 binding — F-SP1-005):** Before `dialLoop` begins
   dialing a new connection for the same address (step 1 of a reconnect iteration), it MUST
   join — that is, block until completion of — the receive goroutine from the previous
   iteration. A `sync.WaitGroup` (Add(1) before goroutine start, Done() in the receive
   goroutine's deferred return) or a per-connection `done chan struct{}` satisfies this
   requirement. The join MUST occur at the end of each dial iteration, before the reconnect
   backoff sleep. Failure to join creates a goroutine-leak vector: a "flapping" upstream
   (rapid connect/disconnect/reconnect) can accumulate O(N) receive goroutines reading from
   closed or recycled connections.
3. `Connector.Stop()` (which calls `stopOnce.Do(close(c.stopCh))` and blocks on `c.doneCh`,
   verified at `8eb54a5`) returns only after BOTH `maintainConn` AND the receive goroutine
   have returned for every active address. No goroutine leak survives `Stop()`.
4. The receive goroutine does NOT access `c.connectedCount` or any shared mutable state
   beyond the `net.Conn` passed to it by `dialLoop`. Concurrent `Stop()` + receive-goroutine
   exit has exactly-once semantics (F-P29-001 lesson applied symmetrically).

**Test names:**

- `TestConnector_ReceiveLoop_ExitsOnConnClose` (unit, `internal/upstreamdial/connector_test.go` — establishes connection with receive goroutine active; closes the upstream server; asserts Stop() returns within deadline without goroutine leak; `go test -race` clean)
- `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` (unit, `internal/upstreamdial/connector_test.go` — **flap-cycle test, re-homed from router_pe_receive_test.go per F-SP3-002**; hand-rolled `heldConn`+`Close()` harness following `TestConnector_BackoffParameters` pattern (verified at `8eb54a5`); Phase 1: fresh Connector → `SetFrameCallback(countingFrameFn)` **[call precedes `Start()` per F-SP4-002 ordering contract]** → `Start()`; connect, frame delivered; Phase 2: server-side conn.Close() → triggers reconnect; Phase 3: second listener accepts redial, frame delivered again; Phase 4: Stop() completes within timeout; no-goroutine-leak assertion via `goleak.VerifyNone` or equivalent; `go test -race` clean; `peWriteFixture` is NOT used; validates per-reconnect-iteration join per Q6 binding)
- `TestConnector_ReceiveLoop_ExitsOnReadError` **(v1.5 — F-SP5-001 pin test; amended v1.6 — F-SP6-001)** (unit, `internal/upstreamdial/connector_test.go` — inject malformed bytes (e.g. single byte `0xFF` as `FrameType`, causing `ErrInvalidFrameType`) to the upstream fixture connection WITHOUT closing the conn; uses the same in-package accept-and-write fixture pattern as the AC-005 flap harness (`net.Listen` + accept + `conn.Write`); assert: (a) the receive goroutine exits (via per-connection done channel or `goleak.VerifyNone`), AND (b) the connector initiates a reconnect cycle (dials the fixture again — the reconnect is triggered via `_ = conn.Close()` in the receive goroutine causing `maintainConn` write failure); **timeout MUST accommodate `keepaliveInterval` + `operativeBase` backoff**; use a short `keepaliveInterval` (10–20ms per existing `connector_test.go` pattern); proves exit-on-read-error and conn.Close() wiring — a `continue` implementation would busy-loop and the done channel would never close)

**Note on `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak`:** this test exercises both
`Stop()` teardown (also covered by `TestConnector_ReceiveLoop_ExitsOnConnClose`) AND the
per-reconnect-iteration join path (unique to flap scenarios). The flap cycle is mandatory:
a test that only exercises `Stop()` after one successful connection does not exercise the
per-iteration join path and would not detect the goroutine-leak vector described in PC-2.
The test lives entirely in `connector_test.go` — no `runRouter` involvement (F-SP3-002
ruling: this is a Connector-level unit test, not an integration test).

**Test level:** unit (all three tests, `connector_test.go`)
**Test files:** `internal/upstreamdial/connector_test.go`

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
| 1 | `internal/frame/frame.go` (MODIFIED) | Add `FrameTypePEConnect FrameType = 0x06` (with `// (ARCH-02 §3.1)` inline citation, same-commit-as-constant obligation); update `Valid()` upper bound to `<= FrameTypePEConnect`; update `FrameType` type doc comment ("five" → "six canonical values"); update `Valid()` doc comment ("five canonical…0x06..0xFF" → "six canonical…0x07..0xFF"); update `ErrInvalidFrameType` doc comment ("five canonical" → "six canonical" / "not in {0x01..0x06}"); **[F-SP3-003 item 8]** update `OuterHeader.FrameType` field comment from `"identifies the frame kind (data, ctl, arq, fec, empty-tick)"` → `"identifies the frame kind (data, ctl, arq, fec, empty-tick, pe_connect)"` (verified at `8eb54a5`: exhaustive enumeration claim, not illustrative example; must be updated in same commit as FrameTypePEConnect definition) | AC-003 / FO-PE-LOOP-001 / F-SP1-002 / F-SP3-003 |
| 2 | `internal/frame/frame.go` (MODIFIED) | Add `frame.ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error)` (new — defined by this story): read `OuterHeaderSize` bytes via `ParseOuterHeader`, then read `hdr.PayloadLen` bytes of payload | AC-001 / Q2 |
| 3 | `internal/frame/frame_test.go` (MODIFIED) | Add `TestFrameType_Valid_PEConnect`: asserts `FrameTypePEConnect.Valid() == true` and `FrameType(0x07).Valid() == false`; change `just_above_max` case from `FrameType(0x06)` to `FrameType(0x07)` (verified at `8eb54a5`: `{"just_above_max", frame.FrameType(0x06), false}` → now invalid since `0x06` becomes `FrameTypePEConnect`); change `invalids` slice `0x06` entry to `0x07` (verified at `8eb54a5`: `invalids := []byte{0x00, 0x06, 0x77, 0xFF}` → `[]byte{0x00, 0x07, 0x77, 0xFF}`); update `"five canonical enum values"` description comment to `"six canonical enum values"` (verified at `8eb54a5`); update `"Bytes not in {0x01..0x05}"` comment to `"Bytes not in {0x01..0x06}"` (verified at `8eb54a5`); **[F-SP2-004]** update `TestParseOuterHeader_AcceptsAllValidFrameTypes` doc comment: `"all five canonical FrameType values"` → `"all six canonical FrameType values"` (verified at `8eb54a5`); append `frame.FrameTypePEConnect` as the sixth element to the `valid` slice in `TestParseOuterHeader_AcceptsAllValidFrameTypes` (verified at `8eb54a5`: currently 5-element slice `{FrameTypeData, FrameTypeEmptyTick, FrameTypeCtl, FrameTypeArq, FrameTypeFec}`); **[F-SP6-004 items 9–10]** update `"five canonical enum values"` comment at `frame_test.go` ~:501 → `"six canonical enum values"`; update `frame_test.go` ~:540 — BOTH `"{0x01..0x05}"` → `"{0x01..0x06}"` AND `"canonical five"` → `"canonical six"` in the same edit | AC-003 / F-SP1-002 / F-SP2-004 / F-SP6-004 — **10 blast-radius locations total** (item 8 in `frame.go` see FCL row 1; items 9–10 added v1.6 F-SP6-004 in `frame_test.go` ~:501 and ~:540) |
| 4 | `internal/upstreamdial/connector.go` (MODIFIED) | Add `type FrameFn func(hdr frame.OuterHeader, raw []byte) error` (new); add `SetFrameCallback(fn FrameFn)` as a method on the concrete `*Connector` ONLY — **NOT added to the `Handle` interface (amended v1.6 — F-SP6-002, Option A)**; `fakeConnectorHandle` in `router_pe_connector_test.go` is NOT affected; add `frameFn FrameFn` field to `Connector` — **set-once pre-Start per the ordering contract (v1.4, F-SP4-002)**; receive goroutine in `dialLoop` MAY assume non-nil; post-Start mutation MUST NOT be permitted (panic or ignore, never unsynchronized write); add receive goroutine in `dialLoop` after step-3 success: calls `frame.ReadOuterFrame(conn)` in a loop, on read error calls `_ = conn.Close()` then `return` **(amended v1.6 — F-SP6-001: close wires read-side failure into write-side teardown; double-close safe/idempotent)**, discriminates `FrameTypePEConnect` (discard) vs all other types (invoke `_ = c.frameFn(hdr, raw)` — discard-and-continue; F-SP4-001); flip bootstrap `ChannelFrame.FrameType` from `halfchannel.FrameTypeData` to `frame.FrameTypePEConnect` (FO-PE-LOOP-001); add direct `internal/frame` import; add per-connection lifecycle sync (WaitGroup or done chan) so `dialLoop` teardown waits for receive goroutine exit before reconnect (per-reconnect-iteration join, F-SP1-005) | AC-001, AC-002, AC-003, AC-005 / Q1, Q2, Q3, Q6 / F-SP4-001 / F-SP4-002 / F-SP6-001 / F-SP6-002 |
| 5 | `internal/upstreamdial/connector_test.go` (MODIFIED) | Unit tests: `TestConnector_ReceiveLoop_DataFrameForwardedToCallback`, `TestConnector_ReceiveLoop_PEConnectFrameDiscarded`, `TestConnector_ReceiveLoop_ExitsOnConnClose`; **[F-SP3-002 AC-005 flap-cycle re-homing]** add `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` (hand-rolled heldConn+Close() flap harness per `TestConnector_BackoffParameters` pattern; fresh Connector → `SetFrameCallback` **[before `Start()` per F-SP4-002 ordering contract]** → `Start()` + counting FrameFn; Phase 1 connect+frame; Phase 2 server Close()→reconnect; Phase 3 second listener+frame; Phase 4 Stop()+no-leak; `peWriteFixture` NOT used; follows existing connector_test.go harness pattern, verified at `8eb54a5`); **[v1.5 F-SP5-001 read-error exit pin test]** add `TestConnector_ReceiveLoop_ExitsOnReadError` (inject `0xFF` FrameType byte to upstream fixture WITHOUT closing conn; uses same in-package accept-and-write pattern as flap harness; assert receive goroutine exits AND reconnect initiated within timeout) | AC-001, AC-003, AC-005 / F-SP3-002 / F-SP4-002 / F-SP5-001 |
| 6 | `cmd/switchboard/mgmt_wire.go` (MODIFIED) | Construct `multipath.NewDropCache(multipath.DefaultDropCacheSize)` and `routing.NewFrameArrivalHandler(dc)` after Phase b; apply `routing.WithFrameArrivalLogger(routerLogger)` (all verified at `8eb54a5`); call `connector.SetFrameCallback(fn)` (new — defined by this story) with `FrameFn` closure routing through `arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, fn)` per Q8 ruling; **[v1.4 F-SP4-002] insertion point binding**: `SetFrameCallback(fn)` inserted between the existing `upstreamdial.New(...)` and `connector.Start()` lines — construct → SetFrameCallback → Start ordering is mandatory; verified at `8eb54a5` that current `mgmt_wire.go` has `New(...)` immediately followed by `Start()` with no call in between; this story inserts the call; add `internal/multipath` import (only new production import at `cmd/switchboard` layer; no ARCH-08 §6.4 registration required) | AC-002 / Q8 / F-SP1-001 / F-SP4-002 |
| 7 | `cmd/switchboard/router_pe_receive_test.go` (NEW) | Integration tests: `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect`, `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival`, `TestRunRouter_PE_EFWD001ExhaustionUnderLoad`; **[F-SP3-001 byte-contract pin test]** `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` (two peWriteFixture.WriteFrame frames identical payload but differing `OuterHeader.SrcAddr`; assert ≥2 `"E-FWD-001"` emissions; proves full-frame reconstruction per Q9 §9.1a); **also defines test-local upstream fixture:** `peWriteFixture` struct (new — `addr string`, `accepted chan net.Conn`, `ln net.Listener`), `startPEWriteFixture(t *testing.T) *peWriteFixture` (new), `(*peWriteFixture).WriteFrame(t *testing.T, wire []byte)` (new) — all three are test-local, not exported (Q9.2 fixture specification; Appendix A Delta v1.2 in placement note). **AC-005 flap-cycle test is NOT in this file** — re-homed to `connector_test.go` per F-SP3-002 ruling; `peWriteFixture` is NOT used by AC-005. | AC-001, AC-002, AC-004 / F-SP2-002 / F-SP3-001 |
| 8 | `.factory/specs/architecture/ARCH-08-dependency-graph.md` (MODIFIED) | §6.5 update: `internal/upstreamdial` allowed imports `{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}`; must land in the same commit that introduces the `frame.ReadOuterFrame` import in `connector.go` | Q2 / ARCH-08 §6.4 amendment |
| 9 | `.factory/specs/architecture/ARCH-02-protocol-stack.md` (MODIFIED) | §"Outer Header Format" `frame_type` table row: add `pe_connect=0x06` — amend row to `u8 enum: data=0x01, empty_tick=0x02, ctl=0x03, arq=0x04, fec=0x05, pe_connect=0x06`; must land in the same commit that defines `FrameTypePEConnect` in `internal/frame/frame.go` (parallel obligation to ARCH-08 §6.5 amendment) | AC-003 / F-SP1-003 / ARCH-02 canonical wire-format source-of-truth |

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
- **`cmd/switchboard` multipath import:** `internal/multipath` is added as a new import at
  the `cmd/switchboard` layer. `cmd/switchboard` is at the top of the DAG; this import is
  unconditionally lawful and requires no ARCH-08 §6.4 registration.
- **ARCH-02 §"Outer Header Format" amendment:** `frame.FrameTypePEConnect = 0x06` is a
  wire-format change. ARCH-02 is the canonical source of truth for the outer-header wire
  format; the `frame_type` row MUST be amended in the same commit that defines
  `FrameTypePEConnect` in `internal/frame/frame.go`. Failing to amend ARCH-02 leaves the
  wire-format spec inconsistent with the implementation.
- **Pure-core / effectful boundary:** `frame.ReadOuterFrame` (new) is effectful (I/O).
  It belongs in `internal/frame` at position 2 — the position constraint allows effectful
  functions at any layer.
- **go.md rule 12:** `connector.Mode()` reads `c.connectedCount` via `atomic.Load` (no mutex
  needed, established precedent from S-7.04-FU-PE-CONNECTOR). The receive goroutine MUST NOT
  call `c.connectedCount.Add` — the concurrent-transition lesson from F-P29-001 applies.

## Library & Framework Requirements

All imports must use existing module versions pinned in `go.mod` at develop `8eb54a5`. No
new external dependencies are introduced. Internal packages used:

| Package | Import path | Position | Verified at | Used by |
|---------|-------------|----------|-------------|---------|
| `frame` | `internal/frame` | 2 | `8eb54a5` | `upstreamdial`, `cmd/switchboard` (new `FrameTypePEConnect`, `ReadOuterFrame`) |
| `halfchannel` | `internal/halfchannel` | 4 | `8eb54a5` | `upstreamdial` (existing) |
| `outerassembler` | `internal/outerassembler` | 8 | `8eb54a5` | `upstreamdial` (existing); test frame assembly via `peWriteFixture` usage (AC-004) |
| `multipath` | `internal/multipath` | — (position ≤17) | `8eb54a5` | `cmd/switchboard/mgmt_wire.go` (new — `NewDropCache`, `DefaultDropCacheSize`) |
| `routing` | `internal/routing` | 17 | `8eb54a5` | `cmd/switchboard/mgmt_wire.go` (existing; gains `NewFrameArrivalHandler`, `WithFrameArrivalLogger`, `OnFrameArrival`, `InterfaceID`, `ForwardFunc`) |
| `upstreamdial` | `internal/upstreamdial` | 19 | `8eb54a5` | gains `FrameFn` type, `SetFrameCallback`, direct `frame` import (new) |
| `testenv` | `internal/testenv` | 23 | `8eb54a5` | integration tests |

## File Structure Requirements

New files created by this story:
- `cmd/switchboard/router_pe_receive_test.go` — integration tests for receive loop (AC-001, AC-002, AC-004, AC-005) + test-local upstream fixture (`peWriteFixture` struct, `startPEWriteFixture`, `WriteFrame` — F-SP2-002)

Modified files:
- `internal/frame/frame.go` — `FrameTypePEConnect` constant, `Valid()` update, doc-comment updates (five→six), `ReadOuterFrame` function, `OuterHeader.FrameType` field comment (item 8, F-SP3-003: append "pe_connect" to kind enumeration)
- `internal/frame/frame_test.go` — `TestFrameType_Valid_PEConnect`; `just_above_max` 0x06→0x07; `invalids` slice 0x06→0x07; description-comment "five canonical"→"six canonical"; range-comment `{0x01..0x05}`→`{0x01..0x06}`; `TestParseOuterHeader_AcceptsAllValidFrameTypes` doc comment "all five"→"all six" (F-SP2-004); append `frame.FrameTypePEConnect` to `valid` slice (F-SP2-004); **[F-SP6-004 items 9–10]** `~:501` "five canonical enum values"→"six canonical enum values"; `~:540` both `"{0x01..0x05}"`→`"{0x01..0x06}"` AND `"canonical five"`→`"canonical six"` — **10 blast-radius locations total** (items 1–7 in frame_test.go, item 8 in frame.go, items 9–10 in frame_test.go ~:501 and ~:540)
- `internal/upstreamdial/connector.go` — `FrameFn` type, `SetFrameCallback`, receive goroutine with `frame.EncodeOuterHeader`+append reconstruction, bootstrap flip, per-reconnect join
- `internal/upstreamdial/connector_test.go` — 5 new unit tests (3 original + `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` flap-cycle F-SP3-002 + `TestConnector_ReceiveLoop_ExitsOnReadError` read-error exit pin test F-SP5-001)
- `cmd/switchboard/mgmt_wire.go` — `DropCache`+`FrameArrivalHandler` construction, `SetFrameCallback` call with `OnFrameArrival` closure, `internal/multipath` import
- `.factory/specs/architecture/ARCH-08-dependency-graph.md` — §6.5 import-set amendment
- `.factory/specs/architecture/ARCH-02-protocol-stack.md` — §"Outer Header Format" `frame_type` row: add `pe_connect=0x06`

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
| `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` | **[F-SP3-002 AC-005 flap-cycle, re-homed from router_pe_receive_test.go]** Full flap cycle at Connector level: Phase 1 fresh Connector → `SetFrameCallback` **[before `Start()` per F-SP4-002]** → `Start()` + frame delivered; Phase 2 server-side Close() → reconnect triggered; Phase 3 second listener accepts + frame delivered again; Phase 4 Stop() within timeout; no goroutine leak; validates per-reconnect-iteration join per Q6/F-SP1-005; `go test -race` clean; `peWriteFixture` NOT used |
| `TestConnector_ReceiveLoop_ExitsOnReadError` | **[F-SP5-001 read-error exit pin test — v1.5; amended v1.6 — F-SP6-001]** Inject malformed bytes (`0xFF` FrameType → `ErrInvalidFrameType`) to upstream fixture WITHOUT closing conn; assert receive goroutine exits (done channel closes) AND connector re-dials (reconnect triggered via `_ = conn.Close()` → `maintainConn` write failure); **timeout MUST accommodate `keepaliveInterval` + `operativeBase` backoff**; use short `keepaliveInterval` (10–20ms per existing `connector_test.go` pattern); proves exit-on-read-error and conn.Close() teardown wiring |

**`cmd/switchboard/router_pe_receive_test.go` (NEW — integration):**

| Function | Proves |
|----------|--------|
| `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` | Frame from upstream fixture reaches `routing.FrameArrivalHandler.OnFrameArrival` callback chain; `RouterHandle.Mode()` == `testenv.ModePE` |
| `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` | `SetFrameCallback` closure wires to `arrivalHandler.OnFrameArrival` (Q8); no routing import in `internal/upstreamdial` (`go list -deps` verified) |
| `TestRunRouter_PE_EFWD001ExhaustionUnderLoad` | `peWriteFixture.WriteFrame` (new — defined by this story) writes assembled `frame.FrameTypeData` outer frame directly to accepted PE connection; single-interface set guarantees split-horizon block → `"E-FWD-001"` in writer output (deterministic per Q8/Q9); S404-OBS-F + S404-LOW-1 re-confirmation via peWriteFixture injection path |
| `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` | **[F-SP3-001 byte-contract pin test + F-SP4-001 loop-continuation pin, Q9 §9.1a]** Two frames with identical payload but differing `OuterHeader.SrcAddr` ([8]byte `0x01...` vs `0x02...`) both produce E-FWD-001 (≥2 emissions); proves crc32 is computed over full-frame bytes (header+payload), not payload-only — payload-only would collide → false-duplicate suppression → only 1 emission; the ≥2-emission requirement additionally pins loop-continuation: a loop that exits on non-nil `frameFn` return would deliver only 1 emission and fail (F-SP4-001) |

**Existing test that must remain unmodified and green:**

| Function | File | Constraint |
|----------|------|------------|
| `TestScanForLine_DetectsEFWD001ProductionEmission` | `cmd/switchboard/router_pe_connector_test.go` | F-P11-001 mutation pin from S-7.04-FU-PE-CONNECTOR — documents `"E-FWD-001"` assertion key; MUST NOT be modified |

**Estimated new test count (forecast):** ~10 net-new (1 `frame_test` + 5 `connector_test` + 4
integration). **connector_test.go** gains 5 tests (3 original + 1 AC-005 flap-cycle re-homed
from integration per F-SP3-002 + 1 read-error exit pin test per F-SP5-001). **router_pe_receive_test.go** has 4 integration tests
(3 original + 1 byte-contract pin test per F-SP3-001); `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak`
(AC-005 flap-cycle test) lives in `internal/upstreamdial/connector_test.go`, NOT in this file. All tests in
`router_pe_receive_test.go` that assert `OnFrameArrival` use the real `runRouter` goroutine
pattern (Q9.3 harness rule). The file additionally defines the test-local `peWriteFixture`
struct + `startPEWriteFixture` + `WriteFrame` (F-SP2-002); these are not test functions but
infrastructure symbols — they do not increase the test count.
This is a pre-implementation forecast; adversarial hardening typically adds additional
regression tests (S-7.04-FU-PE-CONNECTOR added +11 tests above forecast during its
32-pass cycle). Roll-up to be recast in delivered tense after implementation.

---

## Tasks

1. [ ] Read placement note `decisions/S-BL.PE-RECEIVE-LOOP-placement-note.md` v1.7 (current version per frontmatter) and disposition ruling `decisions/S-BL.PE-RECEIVE-LOOP-disposition-ruling.md` v1.0 before writing any code
2. [ ] Update ARCH-08 §6.5: `internal/upstreamdial` import set `{halfchannel, outerassembler}` → `{frame, halfchannel, outerassembler}` (required in same commit as first `frame` import in `connector.go`)
3. [ ] Amend ARCH-02 §"Outer Header Format" `frame_type` row to add `pe_connect=0x06` in the same commit that defines `FrameTypePEConnect` (parallel obligation to Task 2; F-SP1-003)
4. [ ] Add `frame.FrameTypePEConnect = 0x06` constant (with `// (ARCH-02 §3.1)` citation) + update `Valid()` upper bound in `internal/frame/frame.go`
5. [ ] Update `internal/frame/frame.go` doc comments: `FrameType` type ("five" → "six canonical values"), `Valid()` ("0x06..0xFF" → "0x07..0xFF", "five" → "six"), `ErrInvalidFrameType` ("five" → "six" or "not in {0x01..0x06}") (F-SP1-002); **[F-SP3-003 item 8]** update `OuterHeader.FrameType` field comment: append `, pe_connect` to the enumeration (verified at `8eb54a5`: `"identifies the frame kind (data, ctl, arq, fec, empty-tick)"` → `"identifies the frame kind (data, ctl, arq, fec, empty-tick, pe_connect)"`) — must land in same commit as FrameTypePEConnect definition
6. [ ] Add `frame.ReadOuterFrame(r io.Reader) (OuterHeader, []byte, error)` to `internal/frame/frame.go`
7. [ ] Update `internal/frame/frame_test.go`: change `just_above_max` from `FrameType(0x06)` to `FrameType(0x07)`; change `invalids` slice `0x06` entry to `0x07`; update `"five canonical enum values"` comment; update `"Bytes not in {0x01..0x05}"` comment (F-SP1-002); update `TestParseOuterHeader_AcceptsAllValidFrameTypes` doc comment `"all five canonical"` → `"all six canonical"`; append `frame.FrameTypePEConnect` to the 5-element `valid` slice (F-SP2-004); **[F-SP6-004 items 9–10]** update `"five canonical enum values"` comment at `~:501` → `"six canonical enum values"`; update `~:540` — BOTH `"{0x01..0x05}"` → `"{0x01..0x06}"` AND `"canonical five"` → `"canonical six"` in the same edit; item 8 (`OuterHeader.FrameType` field comment) is in `frame.go` via Task 5 — **10 blast-radius locations total** across both files (items 1–7 and 9–10 in `frame_test.go`, item 8 in `frame.go`)
8. [ ] Add `TestFrameType_Valid_PEConnect` to `internal/frame/frame_test.go` (RED gate)
9. [ ] Add `FrameFn` type + `SetFrameCallback(fn FrameFn)` as a method on the concrete `*Connector` in `internal/upstreamdial/connector.go` — **NOT on the `Handle` interface (amended v1.6 — F-SP6-002, Option A)**; `Handle` interface (`ReloadAddrs`/`Mode`/`Stop`) is UNCHANGED; `fakeConnectorHandle` in `router_pe_connector_test.go` is NOT affected
10. [ ] Add receive goroutine in `dialLoop` with `frame.ReadOuterFrame` loop (returns payload-only), `frame.EncodeOuterHeader`+append reconstruction of full frame before passing to `FrameFn` (F-SP3-001 byte-contract — `raw` MUST be full outer-header+payload), `FrameTypePEConnect` discrimination, and per-connection lifecycle sync (WaitGroup or done-chan join before reconnect; F-SP1-005); **[v1.5 F-SP5-001] on read error**: MUST call `_ = conn.Close()` THEN `return` — **[amended v1.6 — F-SP6-001: `conn.Close()` is the wiring that converts read-side failure into write-side teardown; `maintainConn` is write-only and cannot observe receive-goroutine exit; double-close is safe/idempotent]**; **[v1.4 F-SP4-001] discard-and-continue** — non-bootstrap frames invoke `_ = frameFn(hdr, raw)` (blank-identifier discard); non-nil return MUST NOT terminate the loop or trigger logging (OnFrameArrival already logs E-FWD-001/EC-005 internally); the exit-on-error form `if err := frameFn(...); err != nil { return }` is FORBIDDEN (defeats pin test); **[v1.4 F-SP4-002] nil-guard** — optional defense-in-depth nil check on `frameFn` before invocation; silently discard with no log if nil (not a replacement for the ordering obligation)
11. [ ] Flip `dialLoop` bootstrap `ChannelFrame.FrameType` from `halfchannel.FrameTypeData` to `frame.FrameTypePEConnect`
12. [ ] Write unit tests for receive goroutine (AC-001, AC-003, AC-005) including `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` (AC-005 flap-cycle, F-SP3-002 — hand-rolled heldConn+Close() harness, NOT runRouter or peWriteFixture) — RED gate before step 10
13. [ ] In `cmd/switchboard/mgmt_wire.go`: construct `multipath.NewDropCache` + `routing.NewFrameArrivalHandler` after Phase b; apply `routing.WithFrameArrivalLogger`; wire `SetFrameCallback` with `FrameFn` closure routing through `arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, fn)` per Q8 (not `routing.RouteFrame`); add `internal/multipath` import
14. [ ] Write integration tests in `cmd/switchboard/router_pe_receive_test.go` using the real `runRouter` goroutine pattern (Q9.3 harness rule — NOT `testenv.Restart`); define test-local `peWriteFixture` struct + `startPEWriteFixture` + `WriteFrame` in the same file (Q9.2, F-SP2-002); include byte-contract pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` (F-SP3-001, Q9 §9.1a); **NOTE:** flap-cycle test `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` (AC-005) is in connector_test.go (Task 12), NOT here (F-SP3-002 ruling) — RED gate before step 13
15. [ ] Verify `go test -race -count=1 ./...` full green; `golangci-lint` 0 issues; `gofumpt` no diffs
16. [ ] Verify `TestScanForLine_DetectsEFWD001ProductionEmission` still passes unmodified

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
| 1.1 | 2026-07-08 | Remediate spec-adversarial pass-1 findings. Governing artifact: placement note v1.1. F-SP1-001 (HIGH [spec-defect]): Q8 ruling supersedes Q1/Q2 RouteFrame wiring — AC-001 PC-2 rewritten to FrameArrivalHandler.OnFrameArrival wiring with full construction spec (NewDropCache/NewFrameArrivalHandler/WithFrameArrivalLogger/OnFrameArrival/InterfaceID/ForwardFunc all verified at 8eb54a5); AC-002 title + all PCs rewritten to Q8 wiring spec (DropCache construction, arrivalHandler.OnFrameArrival, multipath import, deterministic exhaustion via single-interface set); AC-004 title + mechanism reframed (arqsend remains frame driver per Q4, but exhaustion is topologically guaranteed by interfaceSet=={peIfaceID}, not load-dependent; HMAC bypass noted + SEC follow-on flagged); FCL row 6 rewritten (RouteFrame closure → OnFrameArrival closure + multipath import); Design Constraints Q1/Q2 prose updated to cite Q8 supersession; AC-001 BC trace adds Q8 citation. F-SP1-002 (HIGH [spec-gap]): FCL row 3 expanded with frame_test.go blast-radius amendments (just_above_max 0x06→0x07, invalids 0x06→0x07, "five canonical" comments, "{0x01..0x05}" comment); FCL row 1 expanded with frame.go doc-comment updates (FrameType/"five"→"six", Valid() range, ErrInvalidFrameType); Tasks 5/7 added; File Structure Requirements updated. F-SP1-003 (HIGH [spec-gap]): FCL row 9 added (ARCH-02 §"Outer Header Format" frame_type row amendment, pe_connect=0x06, same-commit-as-constant obligation); Architecture Compliance Rules + File Structure Requirements + Task 3 added. F-SP1-004 (MED [doc-drift]): BC-2.09.001 added to frontmatter bc_traces; Anchors Consumed table gains BC-2.09.001 non-discharging contextual anchor row; AC-001 BC Anchors updated. F-SP1-005 (MED [spec-gap]): AC-005 PC-2 added (per-reconnect-iteration join, binding per Q6 v1.1); AC-005 test names updated (TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop recast as flap-cycle test; rationale note added); FCL row 7 flap-cycle description added; Estimated Test Surface table row updated. F-SP1-007 (LOW [doc-drift]): TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop added to AC-005 Test-names block + reconciled across FCL row 7 + Estimated Test Surface. (F-SP1-006 was placement-note-internal — fixed in placement note v1.1; cited as governing-artifact context only.) FCL: 8→9 rows. |
| 1.2 | 2026-07-09 | Remediate spec-adversarial pass-2 findings. Governing artifact: placement note v1.2 (Q9 ruling). F-SP2-001 (CRITICAL [spec-defect]): AC-004 injection topology rewritten — `net.Dial(routerListenAddr)` + `arqsend.Dispatch` closure removed from AC-004 precondition and Design Constraints Q4 section (two occurrences: ~Q4 design constraint block and ~AC-004 precondition block); replaced with `peWriteFixture.WriteFrame(t, wire)` injection path; AC-004 precondition rewritten to use real `runRouter` goroutine pattern; AC-004 PC-1 rewritten (peWriteFixture writes `outerassembler.Assemble` frame with `frame.FrameTypeData` to accepted PE connection; full `outerassembler.Assemble` call form cited); S404-OBS-F + S404-LOW-1 Anchors Consumed wording updated to peWriteFixture discharge framing (Q9.4 disposition); arqsend context sentence in narrative updated; Q4 design constraint section retitled and redesigned to distinguish production-wiring ruling (retained) from test-role supersession (Q9). F-SP2-002 (HIGH [spec-gap]): FCL row 7 expanded with `peWriteFixture` struct + `startPEWriteFixture` + `WriteFrame` fixture definitions (test-local, new); Q9.2 and Appendix A Delta v1.2 cited; `internal/arqsend` removed from frontmatter `architecture_modules`; Library & Framework Requirements table: arq/arqsend rows removed, outerassembler description updated to peWriteFixture usage. F-SP2-003 (MED [spec-defect]): Q9.3 harness rule section added to Design Constraints (binding: every AC asserting OnFrameArrival — AC-001, AC-002, AC-004 — MUST use real `runRouter` goroutine pattern; NOT `testenv.Restart`); AC-001 and AC-002 preconditions rewritten to reference `runRouter` goroutine pattern + peWriteFixture; AC-005 harness adjudication recorded (lifecycle-only assertions; runRouter used for fidelity but not a harness-rule obligation); Task 14 updated; Estimated Test Surface updated with harness rule note. F-SP2-004 (MED [doc-drift]): FCL row 3 expanded with two additional blast-radius locations (6: `TestParseOuterHeader_AcceptsAllValidFrameTypes` doc comment "all five canonical" → "all six canonical"; 7: append `frame.FrameTypePEConnect` to `valid` slice); FCL row count note updated ("7 blast-radius locations total"); Task 7 updated. Frontmatter: version 1.1→1.2; `internal/arqsend` removed from `architecture_modules`; inputDocuments placement-note reference updated v1.1→v1.2. Token budget updated (~9k). STORY-INDEX: backlog row bumped to ready (v1.2, pass-2 remediated). |
| 1.4 | 2026-07-09 | Remediate spec-adversarial pass-4 findings. Governing artifact: placement note v1.4 (F-SP4-001/002). F-SP4-001 (HIGH [spec-gap]): FrameFn return-value contract — new Design Constraints subsection added (discard-and-continue semantics; non-nil return MUST NOT terminate loop; `_ = frameFn(hdr, raw)` is the only permitted form; exit-on-error `if err := frameFn(...); err != nil { return }` explicitly forbidden; normative precedent is `netingress.ServeConn` drop-and-continue; double-count rationale cited; errcheck compliance via blank-identifier discard; `//nolint:errcheck` MUST NOT be used); Q2 reconstruction code block bare `frameFn(hdr, raw)` → `_ = frameFn(hdr, raw)` with discard comment; discrimination contract bare `frameFn(hdr, raw)` → `_ = frameFn(hdr, raw)` with discard comment; Task 10 updated with discard-and-continue and no-logging obligations and forbidden exit-on-error form; FCL row 4 updated with `_ = c.frameFn(hdr, raw)` discard-and-continue; pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` annotated in AC-004 test names and Estimated Test Surface table as loop-continuation pin: ≥2-emission observable pins that loop continues after first non-nil `frameFn` return. F-SP4-002 (HIGH [spec-gap]): SetFrameCallback ordering contract — new Design Constraints subsection added (MUST be called before `Start()`; `frameFn` is set-once pre-launch; goroutine-creation happens-before covers visibility; construct → SetFrameCallback → Start wiring order in `runRouter` is binding; receive goroutine MAY assume non-nil; nil-guard defense-in-depth silent discard as optional; post-Start mutation forbidden — panic or ignore, never unsynchronized write); AC-002 PC-2 amended with insertion-point annotation (between `New(...)` and `Start()` in `runRouter`); FCL row 4 updated with set-once pre-Start note and post-Start mutation prohibition; FCL row 5 updated with `SetFrameCallback`-before-`Start()` annotation in flap harness description; FCL row 6 updated with binding insertion-point detail; AC-005 flap-cycle test name and FCL row 5 / Estimated Test Surface table updated with explicit before-`Start()` ordering in Phase 1 sequence. Frontmatter version 1.3→1.4; inputDocuments placement-note reference v1.3→v1.4. Token budget ~10k → ~11k. |
| 1.3 | 2026-07-09 | Remediate spec-adversarial pass-3 findings. Governing artifact: placement note v1.3 (F-SP3-001/002/003). F-SP3-001 (HIGH [spec-defect]): Q2 framing-primitive section title updated and rewritten — `frame.ReadOuterFrame` returns payload-only (consistent with `netingress.ReadFrame`; retracted v1.2 false claim); receive goroutine reconstruction obligation added (`ehdr := frame.EncodeOuterHeader(hdr)` + `raw := append(ehdr[:], payload...)`); `frame.EncodeOuterHeader` cited as EXISTING function at `8eb54a5` (not new); `FrameFn raw` is ALWAYS full outer-header+payload (binding); discrimination contract code block updated with reconstruction step; AC-001 PC-2 rewritten with reconstruction path; AC-004 PC-1 rewritten with reconstruction step; byte-contract pin test `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` added to AC-004 test-names block, Estimated Test Surface table, and FCL row 7; Task 10 updated with byte-contract obligation; Task 14 updated with pin test note. F-SP3-002 (HIGH [spec-gap]): AC-005 harness adjudication paragraph rewritten — flap-cycle test re-homed from `router_pe_receive_test.go` to `connector_test.go`; new test name `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak`; all occurrences of `TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop` replaced (AC-005 test-names block, FCL row 7 de-attribution, FCL row 5 addition, Estimated Test Surface table); `peWriteFixture` de-attributed from AC-005 (NOT used); AC-005 test-level/test-files updated to unit-only; Tasks 12 and 14 updated; FCL row 7 explicit "AC-005 is NOT in this file" note. F-SP3-003 (MED [doc-drift]): FCL row 1 expanded with item-8 `OuterHeader.FrameType` field comment update (`"(data, ctl, arq, fec, empty-tick)"` → `"(data, ctl, arq, fec, empty-tick, pe_connect)"`); FCL row 3 blast-radius count "7" → "8"; Task 5 updated with item-8 obligation; Task 7 updated with 8-location note; File Structure Requirements frame.go line updated. Test forecast count updated ~8 → ~9 net-new (1 `frame_test` + 4 `connector_test` + 4 integration). Frontmatter version 1.2→1.3; inputDocuments placement-note reference v1.2→v1.3. Token budget ~9k → ~10k. STORY-INDEX: row v1.2→v1.3, pass-3 remediated. |
| 1.5 | 2026-07-09 | Remediate spec-adversarial pass-5 findings. Governing artifact: placement note v1.5. F-SP5-001 (HIGH [spec-gap]) — READ-error disposition contract: on any non-nil return from `frame.ReadOuterFrame`, the receive goroutine MUST exit the loop (`return`); `continue`-on-read-error is FORBIDDEN (exact mirror of v1.4 callback-error return-FORBIDDEN rule); per-site disposition follows `netingress.ServeConn` precedent (read error → exit; callback error → continue); rationale: `continue` produces busy-loop on EOF or permanent framing desync on malformed-without-close while keepalive writes keep conn alive; exit → `dialLoop` teardown/reconnect is the only correct resync; logging disposition: clean io.EOF/ctx-cancel exit is silent; abnormal read error permits one log line at implementer's discretion; double-count constraint does NOT apply (OnFrameArrival never saw the frame). New "READ-error disposition contract" subsection added to Design Constraints. Q2 reconstruction sketch updated: `if err != nil { ... }` → explicit `if err != nil { return }` with FORBIDDEN comment (v1.5 marker). Discrimination contract block updated with read-error branch above discrimination step. AC-005 gains pin test `TestConnector_ReceiveLoop_ExitsOnReadError` (inject `0xFF` FrameType → `ErrInvalidFrameType` WITHOUT closing conn; assert goroutine exits AND reconnect initiated; uses same in-package accept-and-write pattern as flap harness). FCL row 5 updated (5th test added). Estimated test forecast: connector_test.go 4→5; total net-new ~9→~10. Token budget ~11k→~12k. F-SP5-OBS-1 (LOW [spec-divergence]) — bounded-read divergence accepted: no `LimitReader`/read-deadline on PE receive path; rationale: `uint16 PayloadLen` ≤64KB allocation bound; configured/semi-trusted dialed upstream vs arbitrary ingress; READ-error exit bounds per-connection exposure; keepalive write failures detect dead conns. New "Bounded-read divergence" subsection added to Design Constraints; no implementation change. F-SP5-OBS-2 (LOW [spec-completeness]) — connector_test.go fixture pattern clarified: AC-001 and AC-003 test descriptions each gain a clarifying sentence noting the in-package accept-and-write pattern (local `net.Listen` + accept + `outerassembler.Assemble` + `conn.Write`); `peWriteFixture` stays test-local to `cmd/switchboard`; no new shared helper. Frontmatter version 1.4→1.5; inputDocuments placement-note reference v1.4→v1.5. STORY-INDEX: row ready (v1.4, pass-4 remediated) → ready (v1.5, pass-5 remediated). |
| 1.6 | 2026-07-09 | Remediate spec-adversarial pass-6 findings. Governing artifact: placement note v1.6. F-SP6-001 (HIGH [spec-defect]) — read-error teardown wiring: v1.5 "exit → dialLoop's existing teardown/reconnect path" claim corrected; `maintainConn` is write-only and never reads the conn; receive goroutine MUST call `_ = conn.Close()` before returning on read-error exit to trigger `maintainConn` write failure → `dialLoop` teardown → redial; double-close is safe/idempotent; reconnect latency ≤ keepaliveInterval + operativeBase; new "Reconnect latency bound" subsection added; `TestConnector_ReceiveLoop_ExitsOnReadError` timeout guidance added (accommodate keepaliveInterval + operativeBase; use 10–20ms keepaliveInterval); all three receive-goroutine sketches updated (`_ = conn.Close()` added before `return` in error branch); Lifecycle section amended (two outputs; conn.Close() ownership); FCL row 4 updated. F-SP6-002 (HIGH [spec-gap]) — SetFrameCallback concrete-only (Option A): `SetFrameCallback` is NOT added to the `upstreamdial.Handle` interface; method exists only on the concrete `*Connector`; `runRouter` calls it between `New()` and `Start()`; `fakeConnectorHandle` in `router_pe_connector_test.go` NOT affected; Q1/Q2 Design Constraints seam description corrected; FCL row 4 updated; all "Added to the Handle interface" text corrected with "(amended v1.6 — F-SP6-002)" markers. F-SP6-003 (MED [spec-defect]) — AC observable substitutes: AC-001 PC-3 "connector.Mode() returns ModePE" replaced with `peWriteFixture.accepted` channel receipt OR `"mode=PE"` writer-output line; accepted-fires-before-Add(1) nuance documented (use `"mode=PE"` line for strict assertion); AC-004 precondition "polls for upstreamdial.ModePE" replaced with `"mode=PE"` writer-output line poll; `connector.Mode()` direct assertions noted valid only in `connector_test.go`. F-SP6-004 (LOW [doc-drift]) — blast radius 8→10: FCL row 3 and File Structure Requirements updated; items 9 (frame_test.go ~:501 "five canonical enum values"→"six canonical enum values") and 10 (frame_test.go ~:540 — both `{0x01..0x05}`→`{0x01..0x06}` AND "canonical five"→"canonical six") added; Task 7 updated with items 9–10. Frontmatter version 1.5→1.6; inputDocuments placement-note reference v1.5→v1.6. STORY-INDEX: row ready (v1.5, pass-5 remediated) → ready (v1.6, pass-6 remediated 2026-07-09). |
| 1.7 | 2026-07-09 | Remediate spec-adversarial pass-7 findings. Governing artifact: placement note v1.7 (F-SP7-001 through F-SP7-005). Covers also the already-applied frontmatter version bump (1.6→1.7) and inputDocuments placement-note citation update to v1.7 (POL-001: row covers all substantive changes). F-SP7-001 (HIGH [spec-defect]) — `"mode=PE"` retracted as establishment observable: AC-001 PC-3 option (b) `"mode=PE"` claim that it fires "on PE-mode transition" and "after connectedCount.Add(1)" struck and annotated RETRACTED; `"mode=PE"` is a PE-CONFIG PRESENCE signal emitted at startup/SIGHUP when `len(upstreamRouters) > 0`, strictly before any dial attempt — fires even against unreachable upstreams (TestRunRouter_PE_UnreachableUpstream_PartialPE proves it); MUST NOT be used as an establishment gate; AC-004 precondition `"mode=PE"` poll struck and replaced with `peWriteFixture.accepted` receipt per v1.7 binding ruling; binding three-observable semantics table added to AC-001 PC-3. F-SP7-002 (MED [spec-divergence]) — `peWriteFixture.accepted` timing corrected: AC-001 PC-3 option (a) parenthetical "(atomically incrementing connectedCount, the same event that makes Mode() return ModePE)" struck and annotated RETRACTED; `accepted` fires at TCP-accept, strictly BEFORE `connectedCount.Add(1)` (bootstrap Write at :350 precedes Add(1) at :365); it is an early/approximate establishment gate, sufficient for "connector has dialed" but NOT for `Mode() == ModePE` assertion. F-SP7-003 (MED [spec-divergence]) — note-internal only; grep verification on story confirmed zero live "to Handle interface" / "Handle gains" claims — clean, no story edits required. F-SP7-004 (LOW [doc-drift]) — Task 1 "v1.2" citation updated to "v1.7 (current version per frontmatter)". F-SP7-005 (LOW [spec-completeness]) — transient stale-ModePE window acknowledged: after receive goroutine's `conn.Close()` exit, `Mode()` transiently reports `ModePE` until `maintainConn`'s next-tick write failure decrements `connectedCount` (bounded by `keepaliveInterval`); no AC asserts `Mode()` during the window; no `FrameFn` consumer runs then; new sentence added to Reconnect latency bound subsection. Frontmatter already at version 1.7 (applied by prior agent); inputDocuments placement-note reference already at v1.7 (applied by prior agent). STORY-INDEX: row v1.6→v1.7, pass-7 remediated 2026-07-09. |
| 1.8 | 2026-07-09 | Remediate spec-adversarial pass-8 findings (story-side only; placement note v1.7 unchanged). F-SP8-001 (MED [spec-defect]) — AC-001 PC-3 live text restructured for coherence: the "Use one of: (a) ... OR (b) ..." enumeration (which still grammatically offered the retracted mode=PE observable as an establishment gate, contradicting the v1.7 retraction annotation and binding table) rewritten — peWriteFixture.accepted receipt is THE establishment gate for this PC; mode=PE demoted to an explicit do-not-use note (PE-CONFIG PRESENCE only, MUST NOT be used as an establishment gate); v1.7 strikethrough/annotation blocks preserved; binding three-observable table unchanged. F-SP8-002 (LOW [doc-drift]) — stale test name in Estimated Test Surface roll-up: TestRunRouter_PE_ReceiveLoop_LifecycleClean_OnStop (renamed AND re-homed in v1.3) replaced with TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak living in internal/upstreamdial/connector_test.go; historical changelog rows untouched. Frontmatter version 1.7→1.8; inputDocuments comment annotated story-side-only. STORY-INDEX: row v1.7→v1.8, pass-8 remediated. |
