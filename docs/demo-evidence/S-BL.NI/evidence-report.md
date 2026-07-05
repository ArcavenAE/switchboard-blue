# Demo Evidence Report: S-BL.NI

**Story:** S-BL.NI — Network Ingress: TCP listener reads self-delimiting outer-header frames and dispatches to `routing.RouteFrame`
**HEAD SHA:** 711691a0fe596224826e4ff685e1c9982f743fb5
**Branch:** `feat/s-bl-ni-network-ingress` (off `origin/develop@14fe0c2`)
**Recorded:** 2026-07-05

Consumes:

- **C-1-W3P1-defer** — network-ingress listener (was: accept-and-close skeleton in `runRouter`)
- **PROCESS-GAP-W4** — cross-component `-race` integration test (concurrent Register + ingress dispatch)
- **S-W3.05 AC-009** — E-ADM-017 live-path assertion (was gated on S-BL.NI)
- **BC-2.09.003 PC-9** — `cfg.ListenAddr` application closure (was deferred)

---

## Coverage Map

| AC | Description | Recording | Success Path | Error Path |
|----|-------------|-----------|:---:|:---:|
| AC-001 | Ingress accepts TCP, reads outer-header frame, dispatches to `routing.RouteFrame`; E-ADM-017 fires through the live path; append-skip on re-fire; concurrent Register+dispatch is race-clean | [AC-001-ingress-routeframe.tape](AC-001-ingress-routeframe.tape) | PASS | PASS |

Single-tape coverage: the tape exercises the wire assertions (grep of `netingress.Serve` in `mgmt_wire.go`, header of the `internal/netingress` package doc), the live-path E-ADM-017 integration test, the `-race` concurrent Register+dispatch test, and the framing unit tests. All ACs of the story are covered by one composed recording since the AC set is unified around "the ingress feeds RouteFrame."

---

## AC-001: Ingress Feeds RouteFrame (Live Path)

**File:** `AC-001-ingress-routeframe.tape` / `.gif` / `.webm`

**Demonstrates:**

- Success path:
  - `grep` on `cmd/switchboard/mgmt_wire.go` shows `netingress.Serve(...)` + `dataLn, err := net.Listen("tcp", cfg.ListenAddr)` — proof BC-2.09.003 PC-9 is applied at the listener bind.
  - `TestIntegration_EADM017_FiresThroughLiveIngress` (`internal/netingress/integration_test.go`) spins up a real TCP listener, a real `routing.Router` (with `admission.FailureCounter` + injected `WithNow` clock), and a `netingress.Serve` goroutine. Sends five frames from the same src (no forwarding entry → PATH-A drop). Asserts `E-ADM-016 >= 5` and `E-ADM-017 == 1`. A sixth frame stays at `E-ADM-017 == 1` (append-skip verified).
    - Traces: BC-2.05.005 PC-3, BC-2.05.008 invariant 5, S-W3.05 AC-009.
  - `TestIntegration_ConcurrentRegisterAndRouteRaceClean` (`internal/netingress/integration_test.go`) runs 4 concurrent `RegisterForwardingEntry` goroutines against 4 concurrent ingress dialers under `-race` for 200ms — no race reports. Consumes PROCESS-GAP-W4.
  - `TestReadFrame_*` / `TestServeConn_*` / `TestServe_*` (`internal/netingress/netingress_test.go`) — 14 unit tests cover self-delimiting framing via `OuterHeader.PayloadLen`, clean EOF at the header boundary (returns `io.EOF` only when zero bytes read), fail-closed on truncated header (`io.ErrUnexpectedEOF`), fail-closed on truncated payload, invalid version rejection, invalid frame-type rejection, back-to-back frames on the same stream, dispatch-until-EOF, ctx-cancel returns, route-error-drop-and-continue, malformed-frame-drops-connection, multi-conn accept + WaitGroup join on cancel, listener close on ctx cancel.
- Error path (in the same tape):
  - Malformed frame (invalid frame type) closes the connection with a wrapped `frame.ErrInvalidFrameType` (fail-closed) — asserted by `TestServeConn_MalformedFrameDropsConnection`.
  - Truncated header and truncated payload both surface `io.ErrUnexpectedEOF` (never `io.EOF`) — asserted by `TestReadFrame_TruncatedHeader` and `TestReadFrame_TruncatedPayload`.
  - Frames from unregistered sources drop fail-closed via PATH-A "auth key unavailable"; the integration test uses this precisely as the trigger for E-ADM-016 log lines and the E-ADM-017 alert — no forwarding entry is registered so every frame drops.

**Implementation:**

- `internal/netingress/netingress.go` — new boundary-layer package (ARCH-09 boundary; ARCH-08 §6 imports only `internal/frame`, receives a `RouteFn` closure to avoid importing `internal/routing`).
  - `ReadFrame(r io.Reader) (frame.OuterHeader, []byte, error)` — self-delimiting via `PayloadLen`.
  - `ServeConn(ctx, conn, route, logger) error` — bounded reads via `io.LimitReader(conn, MaxFrameBytes)`; per-frame loop; ctx-cancel closes conn; route errors drop-and-continue.
  - `Serve(ctx, ln, route, logger) error` — accept loop with semaphore cap (`MaxConcurrentConnections = 128`) + `sync.WaitGroup` join on return (ARCH-01 goroutine lifecycle contract).
  - `MaxFrameBytes = frame.OuterHeaderSize + int(^uint16(0))` — natural upper bound from the wire format (VP-066 CWE-400 bounded reads applied to the data plane).
- `cmd/switchboard/mgmt_wire.go` — `runRouter` Phase (c) replaces the accept-and-close skeleton with:
  - `buildRouter(admission.NewAdmittedKeySet(), routerLogger)`
  - `net.Listen("tcp", cfg.ListenAddr)` (BC-2.09.003 PC-9)
  - `netingress.Serve(ingressCtx, dataLn, route, routerLogger)` in a goroutine, where `route := func(hdr, payload) error { return routing.RouteFrame(hdr, payload, router) }`.
  - Graceful shutdown replaces the previous `_ = dataLn.Close()` with `ingressCancel()`, which unblocks the ingress goroutine and lets its own listener-close path run.
- `cmd/switchboard/access.go` — `newStdLogger(w io.Writer) stdLogger` extracted so `runRouter` can share the mgmt logger. Doc comments on `tickIntervalFor`, `runAccessWithConnector`, and `buildRouter` updated: `listen_addr` is applied at the netingress bind, and the FailureCounter is no longer dormant when routers run (live-path exercised by the new integration test).

**Transport / framing decision:**

- **Transport:** plain TCP for now — no TLS at this seam. The story is "make the data plane feed RouteFrame with real bytes," not "invent a wire protocol." The forthcoming S-BL.OA (outer-assembler) is the correct home for session establishment + inner-payload framing on top of this TCP fabric.
- **Framing:** self-delimiting via `OuterHeader.PayloadLen` (u16 big-endian). Read `frame.OuterHeaderSize` bytes, `ParseOuterHeader`, then `PayloadLen` bytes of payload. No external length-prefix layer is invented. This is the minimum honest reading of ARCH-02 §3.1.

**Scope statement:**

- Delivered: net-ingress package (`ReadFrame` / `ServeConn` / `Serve`); wired into `runRouter` in place of the accept-and-close loop; live-path integration test for E-ADM-017; cross-component `-race` test; unit-level framing coverage; doc-comment closure of the deferred-application notes; `bin/switchboard` builds; `just lint` clean; `-race x3` green.
- Not delivered (deferred to S-BL.OA and follow-on stories, per the "STOP and deliver the largest honest subset" guardrail): TLS-on-connect, session establishment / admission handshake, node-authentication key rotation, replay protection at the connection layer, tick-scheduling of frames, retransmit wiring at the ingress side (already lives in `internal/arq`), OuterHeader field validation beyond what `frame.ParseOuterHeader` already does (version-major nibble + frame-type validity — see `_test.go` cases). Marked in the tape and this report.

---

## Test Inventory

- `internal/netingress/netingress.go` — 207 lines (package doc, `Logger`, `nopLogger`, `RouteFn`, `ReadFrame`, `ServeConn`, `Serve`).
- `internal/netingress/netingress_test.go` — 14 tests:
  - `TestReadFrame_HappyPath`
  - `TestReadFrame_ZeroLenPayload`
  - `TestReadFrame_CleanEOFAtBoundary`
  - `TestReadFrame_TruncatedHeader`
  - `TestReadFrame_TruncatedPayload`
  - `TestReadFrame_InvalidVersion`
  - `TestReadFrame_InvalidFrameType`
  - `TestReadFrame_TwoFramesBackToBack`
  - `TestServeConn_DispatchesFramesUntilEOF`
  - `TestServeConn_CtxCancelReturns`
  - `TestServeConn_RouteErrorDoesNotDropConnection`
  - `TestServeConn_MalformedFrameDropsConnection`
  - `TestServe_AcceptsMultipleConnectionsAndJoinsOnCtxCancel`
  - `TestServe_ClosesListenerOnCtxCancel`
- `internal/netingress/integration_test.go` — 2 integration tests (package `netingress_test`):
  - `TestIntegration_EADM017_FiresThroughLiveIngress`
  - `TestIntegration_ConcurrentRegisterAndRouteRaceClean`

Full sweep: `go test ./...` green across all 21 packages; `go test -race -count=3 ./internal/netingress/ ./internal/routing/ ./cmd/switchboard/` green.

Smoke: `just smoke-quick` = 14/14 sentinels pass; `bash test/smoke/tier3-tutorial.sh` = 4/4; `bash test/smoke/spec-runner.sh` = 5/5.

---

## Artifact Inventory

```
docs/demo-evidence/S-BL.NI/
├── AC-001-ingress-routeframe.tape
└── evidence-report.md
```

**Total:** 1 tape, 1 evidence report. Rendered `.gif` / `.webm` are gitignored per POL-004.
