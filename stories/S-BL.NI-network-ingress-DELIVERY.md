---
artifact_id: S-BL.NI-DELIVERY
document_type: story-delivery
level: ops
story_id: S-BL.NI
version: "1.0"
title: "network ingress — TCP listener reads self-delimiting frames and feeds routing.RouteFrame"
status: delivered
producer: implementer
timestamp: 2026-07-05T20:35:00Z
modified: 2026-07-05T20:35:00Z
phase: 2
epic: E-6
wave: 7
priority: P0
scope_phase: E
estimated_points: 3
delivered_points: 3
bc_traces:
  - BC-2.05.005
  - BC-2.05.008
  - BC-2.09.003
vp_traces:
  - VP-066   # CWE-400 bounded reads
  - VP-070   # CWE-770 goroutine exhaustion
subsystems: [transport-layer, network-management]
architecture_modules:
  - cmd/switchboard
  - internal/netingress   # NEW
  - internal/routing
  - internal/frame
  - internal/admission
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-BL.ROUTER-RUNTIME, S-W3.05]
blocks: [S-BL.OA]
head_sha: 711691a0fe596224826e4ff685e1c9982f743fb5
branch: feat/s-bl-ni-network-ingress
base: origin/develop@14fe0c2
worktree: .worktrees/ni-story
drift_consumed:
  - id: C-1-W3P1-defer
    description: "network-ingress listener (was accept-and-close skeleton in runRouter)"
  - id: PROCESS-GAP-W4
    description: "cross-component -race integration test (concurrent Register + ingress dispatch)"
  - id: S-W3.05 AC-009
    description: "E-ADM-017 live-path assertion (was gated on S-BL.NI)"
  - id: BC-2.09.003 PC-9
    description: "cfg.ListenAddr application closure (was deferred to S-BL.NI)"
---

# S-BL.NI — Network Ingress (DELIVERY)

## What Landed

A real data-plane TCP listener replaces the accept-and-close skeleton in
`cmd/switchboard/mgmt_wire.go::runRouter`. The listener reads
self-delimiting outer-header frames from every accepted connection and
dispatches each frame to `routing.RouteFrame`. This closes four drift
items in a single story: the deferred `cfg.ListenAddr` application
(BC-2.09.003 PC-9), the network-ingress listener (C-1-W3P1-defer), the
E-ADM-017 live-path assertion (S-W3.05 AC-009), and the cross-component
`-race` integration test demanded by PROCESS-GAP-W4.

## Scope Delivered vs Deferred

**Delivered:**

- `internal/netingress` — new boundary-layer package (ARCH-09 boundary;
  ARCH-08 §6 imports only `internal/frame`; receives a `RouteFn` closure
  from callers to avoid the routing back-edge).
  - `ReadFrame(r io.Reader) (frame.OuterHeader, []byte, error)` —
    self-delimiting via `OuterHeader.PayloadLen`.
  - `ServeConn(ctx, conn, route, logger) error` — per-frame loop under
    `io.LimitReader(conn, MaxFrameBytes)`; ctx-cancel closes the conn;
    route errors drop-and-continue (routing already logs E-ADM-016).
  - `Serve(ctx, ln, route, logger) error` — accept loop with a semaphore
    cap (`MaxConcurrentConnections = 128`, mirrors `internal/mgmt`) +
    `sync.WaitGroup` join on return (ARCH-01 goroutine lifecycle).
  - `MaxFrameBytes = frame.OuterHeaderSize + int(^uint16(0))` — natural
    upper bound implied by the wire format (VP-066 CWE-400 applied to
    the data plane).
- `cmd/switchboard/mgmt_wire.go::runRouter` — Phase (c) now:
  - `router := buildRouter(admission.NewAdmittedKeySet(), routerLogger)`
  - `dataLn, err := net.Listen("tcp", cfg.ListenAddr)` (BC-2.09.003 PC-9)
  - `netingress.Serve(ingressCtx, dataLn, route, routerLogger)` in a
    goroutine, where `route := func(hdr, payload) error { return
    routing.RouteFrame(hdr, payload, router) }`.
  - Graceful shutdown replaces `_ = dataLn.Close()` with
    `ingressCancel()`; the ingress goroutine drives its own listener
    close and WaitGroup join.
- `cmd/switchboard/access.go` — extracts `newStdLogger(w io.Writer)
  stdLogger` so `runRouter` can share the mgmt logger. Doc comments
  updated:
  - `tickIntervalFor`: `listen_addr` is now applied by S-BL.NI at the
    netingress listener bind.
  - `runAccessWithConnector`: `listen_addr` binding is owned by S-BL.NI.
  - `buildRouter`: FailureCounter is no longer dormant when routers
    run — the live path is exercised by
    `TestIntegration_EADM017_FiresThroughLiveIngress`.

**Deferred to S-BL.OA and follow-ons (per the "STOP and deliver the
largest honest subset" guardrail):**

- TLS-on-connect (this seam is plain TCP; the specs contemplate a full
  session-establishment layer in S-BL.OA).
- Session establishment / admission handshake — S-BL.OA owns this.
- Node-authentication key rotation, replay protection at the connection
  layer, tick-scheduling of frames, retransmit wiring at the ingress
  side (`internal/arq` already owns retransmit for the TX path).
- OuterHeader validation beyond what `frame.ParseOuterHeader` already
  does (version-major nibble + `FrameType.Valid`).
- Sentinel-invariant addition ("data plane accepts frames") — deferred
  as a follow-on; see "Follow-ons" below.

## Transport / Framing Decisions

- **Transport:** plain TCP. Rationale: the story is "make the data plane
  feed RouteFrame with real bytes," not "invent a wire protocol." Adding
  TLS/session handshake here would put speculative code in front of the
  routing seam ARQ+HMAC already protect fail-closed; S-BL.OA is the
  correct home for that layer.
- **Framing:** self-delimiting via `OuterHeader.PayloadLen` (u16
  big-endian). Read 44 header bytes → `ParseOuterHeader` → read
  `PayloadLen` bytes of payload. No external length-prefix layer is
  invented. This is the minimum honest reading of the wire format
  documented in the ARCH-02 / frame specs.

## Test Inventory

- `internal/netingress/netingress_test.go` — 14 unit tests:
  - `TestReadFrame_HappyPath`, `_ZeroLenPayload`, `_CleanEOFAtBoundary`,
    `_TruncatedHeader`, `_TruncatedPayload`, `_InvalidVersion`,
    `_InvalidFrameType`, `_TwoFramesBackToBack`
  - `TestServeConn_DispatchesFramesUntilEOF`, `_CtxCancelReturns`,
    `_RouteErrorDoesNotDropConnection`, `_MalformedFrameDropsConnection`
  - `TestServe_AcceptsMultipleConnectionsAndJoinsOnCtxCancel`,
    `_ClosesListenerOnCtxCancel`
- `internal/netingress/integration_test.go` — 2 integration tests
  (package `netingress_test`):
  - `TestIntegration_EADM017_FiresThroughLiveIngress` — real TCP
    listener + real `routing.Router` + real
    `admission.NewFailureCounter(5, 60s, ..., WithNow(...))` + real
    `netingress.Serve`. Five frames from same src → PATH-A drops →
    `E-ADM-016 >= 5` + `E-ADM-017 == 1`. Sixth frame keeps
    `E-ADM-017 == 1` (append-skip). Traces BC-2.05.005 PC-3,
    BC-2.05.008 invariant 5, S-W3.05 AC-009.
  - `TestIntegration_ConcurrentRegisterAndRouteRaceClean` — 4 register
    writers × 4 ingress dialers for 200ms under `-race`. Race detector
    is the assertion. Consumes PROCESS-GAP-W4.
- `cmd/switchboard/mgmt_wire_test.go::TestRunRouter_DataListenerBinds`
  scope note updated: bind is still the observable; frame-dispatch
  coverage lives in `internal/netingress/*_test.go`.

**Runs:**

- `just fmt` — clean.
- `just lint` — 0 issues.
- `go test -race -count=3 ./internal/netingress/ ./internal/routing/
  ./cmd/switchboard/` — all green.
- `go test ./...` — all 21 packages green.
- `just smoke-quick` — 14/14 sentinels pass.
- `bash test/smoke/tier3-tutorial.sh` — 4/4 pass.
- `bash test/smoke/spec-runner.sh` — 5/5 pass.

## Drift Items Consumed

| ID | Description | Where closed |
|----|-------------|--------------|
| C-1-W3P1-defer | Network-ingress listener (accept-and-close skeleton) | `cmd/switchboard/mgmt_wire.go` runRouter Phase (c) replaced with `netingress.Serve` |
| PROCESS-GAP-W4 | Cross-component `-race` integration test | `TestIntegration_ConcurrentRegisterAndRouteRaceClean` |
| S-W3.05 AC-009 | E-ADM-017 live-path assertion | `TestIntegration_EADM017_FiresThroughLiveIngress` |
| BC-2.09.003 PC-9 | `cfg.ListenAddr` application closure | `net.Listen("tcp", cfg.ListenAddr)` in runRouter Phase (c); doc comments updated in `access.go` |

## Blast Radius

- **New package:** `internal/netingress` — one boundary-layer package,
  no downstream consumers outside `cmd/switchboard/mgmt_wire.go`.
- **Modified callers:** `cmd/switchboard/mgmt_wire.go` (`runRouter`
  Phase (c)) — replaces a stub, not existing production behavior.
- **Non-modification:** `internal/routing`, `internal/frame`,
  `internal/admission` unchanged (used as-is, with no
  interface/signature changes).
- **Wire-visible change:** the router now actually accepts data-plane
  connections and reads frames. Previously the port was open but every
  connection was closed immediately. Downstream tests that assumed the
  connection would drop remain valid: they use no frame writes.
- **Ownership envelope respected:** touched only `cmd/switchboard/`,
  `internal/netingress/`, `docs/demo-evidence/S-BL.NI/`, and doc
  comments in `internal/routing`-adjacent files. Did NOT touch
  `cmd/sbctl/`, `internal/config/`, `.beads/`, `.run.yaml`.

## Follow-ons (Filed as Deferrals, not Regressions)

- **Sentinel invariant addition** — "data plane accepts frames end-to-end
  under normal config." Not added here because the existing sentinel set
  is stable and the story's live-path integration test is a stronger
  assertion for what's now provable. Candidate for a Wave-8 sentinel
  refresh alongside S-BL.OA.
- **S-BL.OA** — outer-assembler now unblocked; owns TLS + session
  handshake + inner-payload framing on top of this TCP fabric.
- **Ingress-side ARQ hookup** — RX-path retransmit awareness is not
  wired at the ingress layer; still lives in the ARQ subsystem. Not
  regressed, just still to do.

## Commit Trail (on `feat/s-bl-ni-network-ingress`)

1. `eaa6da3` — `feat(netingress): TCP ingress reader dispatching frames to RouteFn (S-BL.NI)`
2. `2acdb28` — `feat(router): wire netingress into runRouter + E-ADM-017 live-path test (S-BL.NI)`
3. `4ae8fe5` — `docs(comments): update deferred-application notes for S-BL.NI closure`
4. `711691a` — `chore(netingress): silence lint findings on close and route signatures`
5. `714eb8b` — `docs(demo-evidence): S-BL.NI AC-001 tape + evidence report`

PR: (see coordinator; opened as `feat(router): S-BL.NI network ingress
— live data path feeding RouteFrame`.)
