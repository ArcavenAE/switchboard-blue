---
artifact_id: S-BL.PE-RECEIVE-LOOP-DELIVERY
document_type: delivery
level: ops
story_id: S-BL.PE-RECEIVE-LOOP
version: "1.0"
title: "PE-connection receive/forward loop — DELIVERY"
status: final
producer: story-writer
timestamp: 2026-07-11T00:00:00Z
merged_at: "2026-07-11T12:42:38Z"
merge_pr: 118
merge_sha: e940fc20028ce8548381b52a98caeb48c20a53dc
branch: story/s-bl-pe-receive-loop
base_sha: 42baa8c    # develop binding base at spec-cycle start
head_sha: 7b84c2b    # demo evidence commit (13th on branch)
phase: 2
wave: steady-state
priority: P2
scope_phase: PE
epic: E-7
bc_traces:
  - BC-2.02.008   # PC-3/EC-003 — E-FWD-001 exhaustion discharge (binding anchor re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 postcondition 1)
  - BC-2.06.003   # PC-1 — non-discharging prerequisite trace; receive loop makes full send+forward path live
  - BC-2.09.001   # AC-001/AC-002 anchor, PC-2/PC-3 — upstream connections established; router is in PE mode (contextual anchor)
vp_traces: []
subsystems: [deployment-operations, transport-layer]
architecture_modules:
  - internal/frame
  - internal/upstreamdial
  - internal/routing
  - internal/multipath
  - internal/testenv
  - cmd/switchboard
cycle: v1.0.0-greenfield
---

# S-BL.PE-RECEIVE-LOOP — DELIVERY

## What Landed

This story discharges FO-PE-LOOP-001 (from S-7.04-FU-PE-CONNECTOR) and completes
the PE-connection receive/forward loop deferred by that predecessor. It delivers the
`frame.FrameTypePEConnect` constant, `frame.ReadOuterFrame`, a per-connection receive
goroutine in `internal/upstreamdial`, and the `routing.FrameArrivalHandler.OnFrameArrival`
callback seam in `runRouter`, closing the E-FWD-001 exhaustion discharge re-anchored
from `S-7.04-FU-PE-CONNECTOR AC-004`. S-7.04-FU-DRAIN-WIRE is now unblocked.

Three composed code changes + spec artifacts:

1. **`internal/frame/frame.go` + `frame_test.go` (MODIFIED)** — `FrameTypePEConnect
   FrameType = 0x06` constant with `// (ARCH-02 §3.1)` citation; `Valid()` upper bound
   widened to `<= FrameTypePEConnect`; doc comments updated "five" → "six canonical
   values" in four locations; `ErrInvalidFrameType` doc updated; `OuterHeader.FrameType`
   field comment appended `, pe_connect`; new function `frame.ReadOuterFrame(r io.Reader)
   (OuterHeader, []byte, error)` — reads `OuterHeaderSize` bytes via `ParseOuterHeader`
   then `hdr.PayloadLen` bytes of payload (payload-only return, matching
   `netingress.ReadFrame`). `frame_test.go` blast-radius sweep: `just_above_max`
   case `0x06` → `0x07`; `invalids` slice `0x06` → `0x07`; five-count comments
   → six-count throughout; `TestParseOuterHeader_AcceptsAllValidFrameTypes` updated
   with sixth element; items 9–10 at `~:501`/`~:540` updated. Unified blast-radius:
   10 frame sweep locations + 2 ARCH-08 import-edge-prose locations = 12.

2. **`internal/upstreamdial/connector.go` (MODIFIED)** — `type FrameFn func(hdr
   frame.OuterHeader, raw []byte) error` (new); `SetFrameCallback(fn FrameFn)` as a
   method on the concrete `*Connector` ONLY (NOT added to the `Handle` interface —
   F-SP6-002 Option A; `fakeConnectorHandle` in `router_pe_connector_test.go` is
   unchanged); `frameFn FrameFn` field set-once pre-`Start()` per the ordering
   contract (post-`Start()` mutation is caller-responsibility — F-IP2-001 Option b:
   guard cannot be made race-safe without new sync primitive; sole production caller
   `runRouter` is provably correct); direct `internal/frame` import (new — ARCH-08
   §6.5 amendment); receive goroutine in `dialLoop` after step-3 success: calls
   `frame.ReadOuterFrame(conn)` in a loop; on ANY non-nil return calls
   `_ = conn.Close()` then exits (`continue`-on-read-error FORBIDDEN — F-SP5-001;
   `conn.Close()` wires read-side failure into `maintainConn` write failure →
   `dialLoop` teardown → redial, F-SP6-001; double-close safe/idempotent;
   unconditional close upheld after empirical TCP half-close hole validation,
   F-GP1-001); `FrameTypePEConnect` discrimination (silently discarded;
   discard-and-CONTINUE, not discard-and-close — F-SP18-001); non-bootstrap
   frames invoke `_ = frameFn(hdr, raw)` (discard-and-continue; non-nil return
   MUST NOT terminate loop — F-SP4-001); `raw` is ALWAYS full wire frame via
   `ehdr := frame.EncodeOuterHeader(hdr)` + `raw := append(ehdr[:], payload...)`
   (F-SP3-001 byte-contract); per-connection `sync.WaitGroup` for per-reconnect-
   iteration join (F-SP1-005); bootstrap `ChannelFrame.FrameType` flipped from
   `halfchannel.FrameTypeData` placeholder to `frame.FrameTypePEConnect`
   (FO-PE-LOOP-001 discharged).

3. **`cmd/switchboard/mgmt_wire.go` (MODIFIED)** — constructs
   `multipath.NewDropCache(multipath.DefaultDropCacheSize)` then
   `routing.NewFrameArrivalHandler(dc)` with `routing.WithFrameArrivalLogger(routerLogger)`
   applied; calls `connector.SetFrameCallback(fn)` between the existing
   `upstreamdial.New(...)` and `connector.Start()` (construct → SetFrameCallback →
   Start ordering is binding — F-SP4-002); `FrameFn` closure routes through
   `arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, fn)`
   per Q8 ruling (`peIfaceID = 1`, `ForwardFunc fn` is nil — safe for single-interface
   set: split-horizon always exhausts before `fn` is invoked; forward obligation falls
   on the interface-set-widening story); `internal/multipath` import (only new
   production import at `cmd/switchboard` layer; no ARCH-08 §6.4 registration required).

4. **Spec artifacts** — `9792605` (spec-side commit, cross-cited with code stub
   `c316aed`): ARCH-08-dependency-graph.md v2.11 (§6.5 import-set `{halfchannel,
   outerassembler}` → `{frame, halfchannel, outerassembler}`; §6.5 parenthetical
   reconciled; §6.6.2 forbidden-edges bullet replaced); ARCH-02-protocol-stack.md
   v1.1 (`frame_type` row: `pe_connect=0x06` added); BC-2.01.004.md v1.3 (Postcondition
   2 outer-header layout table `frame_type` row updated; wire-format spec pair with
   ARCH-02, same-commit parallel obligation per F-SP14-001).

5. **Test files (new + modified):**
   - `internal/frame/frame_test.go` — 1 new test (`TestFrameType_Valid_PEConnect`)
   - `internal/upstreamdial/connector_test.go` — 9 new unit tests (see Scope Delivered)
   - `cmd/switchboard/router_pe_receive_loop_test.go` (NEW) — 4 integration tests +
     test-local `peWriteFixture` struct, `startPEWriteFixture`, `WriteFrame`

## Commit Trail

Branch `story/s-bl-pe-receive-loop`, merge-base `develop@42baa8c`.

| SHA | Description |
|-----|-------------|
| `c316aed` | stubs — `internal/frame` + `internal/upstreamdial` scaffold + `mgmt_wire.go` construction site |
| `a3d5117` | RED — 12 tests + `frame_test.go` blast-radius sweep; 27-package suite RED on stubs |
| `ae2ea7d` | harness fix — `return` → `break` + `dialCount++` in server goroutine inner loop (RED-harness return-vs-break observability bug; see Green-Phase Discoveries) |
| `e85c9df` | `frame.ReadOuterFrame` + `FrameTypePEConnect` + `Valid()` update + blast-radius sweep |
| `8e8296c` | receive goroutine + per-connection `sync.WaitGroup` + `frameFn` field + `SetFrameCallback` |
| `5274cf1` | `mgmt_wire.go` — `DropCache` + `FrameArrivalHandler` + `SetFrameCallback` closure + `multipath` import |
| `9c1b21d` | F-GP1-001 — unconditional `conn.Close()` on any non-nil `ReadOuterFrame` return (EOF carve-out rejected: TCP half-close hole) |
| `75c5904` | F-GP1-001 predecessor test fix — `TestConnector_BackoffParameters` Phase-3 stamp logic: Mode-drop poll + 2-stamp redial gap (teardown-path-robust) |
| `a23deae` | gofmt |
| `e397157` | F-IP1-001 — `TestUpstreamdialImportPerimeter` perimeter regression guard (`go list -deps` + positive-coverage guard) |
| `c3fca02` | F-IP2-002 — `TestUpstreamdialImportPerimeter` doc comment false attribution corrected (comment-only) |
| `7cedc34` | F-IP4-001 — `TestConnector_BootstrapFrameTypePEConnect` outgoing bootstrap frame_type pin (accept-and-read fixture, kills silent revert to `FrameTypeData`) |
| `7b84c2b` | demo evidence (`docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/`) |

Code lane behavior-final at `7cedc34` (first behavior-change commit since `9c1b21d`; `c3fca02` and `7cedc34` are test-only; `7b84c2b` is docs-only). HEAD `7b84c2b` is docs-only (demo evidence).

**Spec-side commit** `9792605` landed before stubs: ARCH-08 v2.11, ARCH-02 v1.1, BC-2.01.004 v1.3 (cross-cited with code `c316aed`; on develop or spec branch, not on story branch).

**Changed files (production + test):**

| File | Change |
|------|--------|
| `internal/frame/frame.go` (MODIFIED) | `FrameTypePEConnect = 0x06`; `Valid()` widened; `ReadOuterFrame`; doc-comment blast-radius (items 1, 5, 8) |
| `internal/frame/frame_test.go` (MODIFIED) | Blast-radius sweep items 2–7, 9–10; `TestFrameType_Valid_PEConnect` (new) |
| `internal/upstreamdial/connector.go` (MODIFIED) | `FrameFn` type; `SetFrameCallback`; receive goroutine with `ReadOuterFrame` + discrimination + `conn.Close()` wiring; bootstrap flip; per-connection WaitGroup join; direct `frame` import |
| `internal/upstreamdial/connector_test.go` (MODIFIED) | 9 new unit tests (see below) |
| `cmd/switchboard/mgmt_wire.go` (MODIFIED) | `DropCache`/`FrameArrivalHandler`/`SetFrameCallback` closure; `internal/multipath` import |
| `cmd/switchboard/router_pe_receive_loop_test.go` (NEW) | 4 integration tests + test-local `peWriteFixture` |
| `.factory/specs/architecture/ARCH-08-dependency-graph.md` | §6.5 import-set amendment + parenthetical reconciliation + §6.6.2 forbidden-edges replacement (spec-side `9792605`) |
| `.factory/specs/architecture/ARCH-02-protocol-stack.md` | `frame_type` row: `pe_connect=0x06` (spec-side `9792605`) |
| `.factory/specs/behavioral-contracts/ss-01/BC-2.01.004.md` | Postcondition 2 `frame_type` row: `pe_connect=0x06` (spec-side `9792605`) |
| `docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/` | 5 tape scripts + `evidence-report.md` (commit `7b84c2b`) |

## Scope Delivered

**Delivered (all 5 ACs fully discharged — see Declared Divergences for accepted observations):**

- **AC-001 (BC-2.09.001 PC-2/PC-3)** — Receive goroutine active per established PE connection;
  incoming frames reach `FrameArrivalHandler`. After `dialLoop` step-3 success, a per-connection
  receive goroutine calls `frame.ReadOuterFrame(conn)` in a loop. `peWriteFixture.WriteFrame(t,
  wire)` writes a pre-assembled outer frame (`frame.FrameTypeData`, non-bootstrap) to the accepted
  PE connection. The goroutine reconstructs the full wire frame (`frame.EncodeOuterHeader` +
  `append`) and invokes the `FrameFn` callback. The `FrameFn` closure in `runRouter` calls
  `arrivalHandler.OnFrameArrival(raw, peIfaceID, []routing.InterfaceID{peIfaceID}, fn)`.
  Tests: `TestConnector_ReceiveLoop_DataFrameForwardedToCallback` (unit),
  `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` (integration).

- **AC-002 (BC-2.02.008 PC-3)** — `runRouter` constructs `FrameArrivalHandler` and wires
  `SetFrameCallback` closure. `multipath.NewDropCache(DefaultDropCacheSize)` →
  `routing.NewFrameArrivalHandler(dc)` → `routing.WithFrameArrivalLogger(routerLogger)` constructed
  after Phase b; `connector.SetFrameCallback(fn)` inserted between `New(...)` and `Start()`;
  closure routes through `arrivalHandler.OnFrameArrival`. No `internal/routing` import in
  `internal/upstreamdial` — ARCH-08 §6.6.2 perimeter preserved; enforced by
  `TestUpstreamdialImportPerimeter` (standalone `go list -deps` regression guard).
  Tests: `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` (integration),
  `TestUpstreamdialImportPerimeter` (unit, `connector_test.go`).

- **AC-003 (FO-PE-LOOP-001)** — `FrameTypePEConnect` constant, `Valid()` bound, `dialLoop` flip,
  discrimination. `frame.FrameTypePEConnect = 0x06` defined; `Valid()` returns `true` for `0x06`
  and `false` for `0x07` (upper bound not over-widened); `dialLoop` bootstrap flipped from
  `halfchannel.FrameTypeData` placeholder; discrimination: `FrameTypePEConnect` frames silently
  discarded (discard-and-continue, NOT discard-and-close — F-SP18-001); non-bootstrap frames
  (including `FrameTypeCtl` — RESYNC consumer path per Non-Goals) forwarded to `FrameFn`.
  Tests: `TestFrameType_Valid_PEConnect` (unit, `frame_test.go`),
  `TestConnector_BootstrapFrameTypePEConnect` (unit, mutation-verified: flip to `FrameTypeData`
  kills the test), `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` (unit, extended with
  second-frame data assertion on same conn per F-SP18-001),
  `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` (unit, F-SP17-001).

- **AC-004 (BC-2.02.008 PC-3/EC-003 — binding anchor; S404-OBS-F; S404-LOW-1)** —
  E-FWD-001 split-horizon exhaustion discharge. The `FrameFn` closure passes
  `interfaceSet == []routing.InterfaceID{peIfaceID}` (single interface); `SplitHorizon.Forward`
  finds no eligible output → `ErrAllPathsSplitHorizon` → E-FWD-001 fires deterministically.
  HMAC bypass on PE receive path: `RouteFrame`'s admission check is bypassed; PE upstream
  connections are outbound-established by the connector itself (SEC follow-on flagged for PR).
  S404-OBS-F and S404-LOW-1 re-confirmed: "send" = `peWriteFixture.WriteFrame`; "forward
  attempt" = `OnFrameArrival` through split-horizon (Q9.4 disposition; arqsend not involved).
  Byte-contract pin: `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` (≥2
  E-FWD-001 emissions for two frames with identical payload but differing `OuterHeader.SrcAddr`
  — proves full-frame `crc32`, not payload-only; additionally pins loop-continuation: loop
  must continue after first non-nil `frameFn` return).
  Tests: `TestRunRouter_PE_EFWD001ExhaustionUnderLoad` (integration),
  `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` (integration),
  `TestScanForLine_DetectsEFWD001ProductionEmission` (existing normative pin — unmodified).

- **AC-005 (Q6 lifecycle binding; F-P29-001 lesson)** — Receive goroutine lifecycle: per-reconnect
  join, doneCh ordering, `Stop()` blocks until all receive goroutines return. Goroutine exits on
  ANY non-nil `ReadOuterFrame` return (calls `_ = conn.Close()` then `return`); per-address
  `sync.WaitGroup` (`Add(1)` before goroutine start, `Done()` deferred) — `dialLoop` teardown
  MUST wait for goroutine exit before reconnect (F-SP1-005 per-reconnect-iteration join);
  `conn.Close()` ownership: `dialLoop` teardown OR receive goroutine (double-close safe);
  `Connector.Stop()` blocks on `c.doneCh` until all per-address done channels drained.
  Tests: `TestConnector_ReceiveLoop_ExitsOnConnClose` (unit),
  `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` (unit, flap-cycle with
  `runtime.NumGoroutine` before/after gate — OBS-1 accepted pin-limitation documented),
  `TestConnector_ReceiveLoop_ExitsOnReadError` (unit, complete 44-byte header
  byte[0]=0x01/byte[1]=0x07/bytes[2:4]=0x0000/bytes[4:44]=0x00, conn NOT closed →
  `ParseOuterHeader` → `ErrInvalidFrameType` → exit + reconnect; F-SP11-001 corrected recipe),
  `TestConnector_ReceiveLoop_ExitsOnVersionMismatch` (unit, byte[0]=0xFF →
  `ErrVersionMismatch`; same exit contract; F-SP11-001 companion).

**Total test count:** 14 net-new tests.

| Test file | Net-new tests |
|-----------|--------------|
| `internal/frame/frame_test.go` (MODIFIED) | 1 |
| `internal/upstreamdial/connector_test.go` (MODIFIED) | 9 |
| `cmd/switchboard/router_pe_receive_loop_test.go` (NEW) | 4 |

**Overage note:** Pre-implementation forecast was ~14 net-new. Delivered 14 (at forecast).
Adversarial hardening added 2 tests above the original ~12-test estimate from passes 1 and 4:
F-IP1-001 added `TestUpstreamdialImportPerimeter`; F-IP4-001 added
`TestConnector_BootstrapFrameTypePEConnect`.

## Declared Divergences

Two accepted observations declared per the declared-divergence protocol. No behavioral
divergences from binding spec contract.

**Divergence 1 — Bounded-read divergence: no `LimitReader` / read deadline (class: spec-divergence-accepted — F-SP5-OBS-1)**

No `io.LimitReader` or `SetReadDeadline` on the PE receive path, diverging from
`netingress.ServeConn`'s `io.LimitReader` pattern. Accepted with rationale: (1) `PayloadLen`
is `uint16` — maximum frame allocation is 44 + 65,535 = 65,579 bytes; no amplification
possible. (2) The PE upstream is a DIALED connection to a configured, semi-trusted upstream
router — not an arbitrary accepted connection from an unknown client. (3) READ-error exit
(F-SP5-001) ensures any malformed frame causes immediate teardown and reconnect — per-connection
allocation is bounded. A withheld-payload stall is bounded to one connection and resolves on
`Stop()`. No implementation change required.

**Divergence 2 — Transient stale-`ModePE` window after receive goroutine exit (class: accepted-timing — F-SP7-005)**

After the receive goroutine calls `_ = conn.Close()` and exits, `connectedCount.Add(-1)` has
NOT yet fired — `maintainConn` must observe its next write failure first. During this window
(bounded by `keepaliveInterval`), `Mode()` transiently reports `ModePE`. No AC in this story
asserts `Mode()` during this window. Future stories asserting `Mode()` after deliberate teardown
MUST account for this interval.

## Forward Obligations

### Consumed by This Story

| FO ID | Origin | Description | Disposition |
|-------|--------|-------------|-------------|
| FO-PE-LOOP-001 | S-7.04-FU-PE-CONNECTOR F-P26-001 (v1.24 deferral) | Define `frame.FrameTypePEConnect = 0x06`; update `Valid()` upper bound; flip `dialLoop` bootstrap from `halfchannel.FrameTypeData` placeholder; receive loop must discriminate bootstrap from session-data frames | DISCHARGED — AC-003 + FCL rows 1, 4; constant defined; `Valid()` widened to `<= 0x06`; bootstrap flip pinned by `TestConnector_BootstrapFrameTypePEConnect` (mutation-verified); discrimination: bootstrap discarded, all other types forwarded |

### Emitted by This Story

| FO ID | Target | Description |
|-------|--------|-------------|
| FO-RECV-FWD-001 | interface-set-widening story | `mgmt_wire.go` `ForwardFunc fn` is nil — split-horizon always exhausts before `fn` is invoked in the single-interface set. When the router's interface set widens (second interface added), `ForwardFunc` MUST be wired to route frames to the correct egress interface. The nil `fn` is explicitly safe-for-now and explicitly annotated in code. |

## Adversarial Convergence Summary

Two convergence phases for this story.

### Spec-Adversarial Cycle (story development — S-BL.PE-RECEIVE-LOOP)

Reference: `.factory/cycles/cycle-1/convergence-trajectory.md` (S-BL.PE-RECEIVE-LOOP
spec-adversarial sections, passes SP-1 through SP-24).

| Metric | Value |
|--------|-------|
| Total spec passes | 24 |
| Clean streak passes | SP-22, SP-23, SP-24 (streak 3/3) |
| Final story version at spec-convergence | v1.20 |
| Total spec findings | ~28 |
| Notable finding classes | spec-defect (Q8 wiring supersession, byte-contract, injection topology), spec-gap (ordering contract, FrameFn return-value, READ-error disposition, conn.Close() teardown wiring), doc-drift (blast-radius sweeps, mode=PE retraction, frame constant doc-comments) |

**GREEN-phase discoveries (two; reference: convergence-trajectory.md §"GREEN phase 2026-07-11"):**

| Discovery | Nature | Resolution |
|-----------|--------|------------|
| RED-harness return-vs-break observability bug | Test-writer defect: server goroutine's inner read loop used `return` rather than `break`-with-dialCount-increment, capping `dialCount` at 1 regardless of reconnection cycles. | Test-writer self-corrected at commit `ae2ea7d`. Pattern noted as candidate upstream finding. |
| F-GP1-001 — EOF carve-out deviation | Implementer introduced `if err == io.EOF { return }` (no close), intending clean half-close semantics. TCP half-close hole: peer `CloseWrite` → receive goroutine exits on `io.EOF` WITHOUT closing → keepalive writes continue ACKing → conn permanently read-dead → no reconnect trigger. `TestConnector_BackoffParameters` failed 3/3 deterministically with carve-out present. | Unconditional `_ = conn.Close()` upheld (commit `9c1b21d`). EOF carve-out rejected. Predecessor test made teardown-path-robust (commit `75c5904`): Mode-drop poll + 2-stamp redial gap. |

### Per-Story Implementation Adversarial Cycle (BC-5.39.001)

Reference: `.factory/cycles/cycle-1/convergence-trajectory.md` (PE-RECEIVE-LOOP per-story
adversarial sections, passes IP-1 through IP-7).

| Metric | Value |
|--------|-------|
| Total passes | 7 |
| HAS_FINDINGS passes | P1–P4 (4 passes) |
| Clean passes (streak) | P5, P6, P7 — streak P5/P6/P7 |
| BC-5.39.001 satisfied | Pass 7 — streak 3/3 |
| Final code SHA | 7cedc34 |
| Demo evidence SHA | 7b84c2b |
| Final story version | v1.25 |
| Total findings | 6 |
| Open findings | 0 |

**Streak table:**

| Pass | Verdict | Notes |
|------|---------|-------|
| P5 | NO_FINDINGS | Streak 0→1/3 |
| P6 | NO_FINDINGS | Streak 1→2/3 |
| **P7** | **NO_FINDINGS** | **Streak 2→3/3 — PER-STORY CONVERGED** |

**Finding-decay shape:** `1 → 3 → 1(+2 obs) → 1(+1 obs) → 0 → 0 → 0`

**Findings by class:**

| Class | Count | Description |
|-------|-------|-------------|
| spec-gap / false-enforcement-claim | 1 | F-IP1-001: perimeter assertion promised but undelivered + "build MUST fail" claim factually wrong (acyclic edge) |
| spec-gap / unimplemented-clause | 1 | F-IP2-001: post-Start mutation guard clause never implemented (resolved caller-responsibility; no code change) |
| partial-fix propagation gap | 1 | F-IP2-002: F-IP1-001 remediation corrected story prose but left false attribution in `connector_test.go` doc comment |
| doc-drift / dual-changelog parity | 1 | F-IP2-003: ARCH-08 v2.11 `modified:` frontmatter bumped without corresponding changelog-table row |
| incomplete-sweep / note-side propagation | 1 | F-IP3-001: F-IP2-001 ruling required note-side Option-b propagation; note `Q1 :194-199` left unannotated; 9th incomplete-sweep instance |
| test-set underdetermination | 1 | F-IP4-001: outgoing bootstrap `frame_type` revertible to `FrameTypeData` without triggering any test failure; same class as F-SP17-001/F-SP18-001 receive-side |

**Notable findings:**

- **F-IP1-001 (perimeter guard undelivered)** — The story promised a `go list -deps`
  assertion in `connector_test.go` to enforce the ARCH-08 §6.6.2 forbidden edge; the test
  was never written. Additionally, the "build MUST fail" enforcement claim was factually
  wrong — the `upstreamdial` → `routing` edge is acyclic (position 19 > 17); Go's toolchain
  does not reject it at build time. The perimeter is enforced only at test-time. Fixed by
  standalone `TestUpstreamdialImportPerimeter` with `go list -deps` + positive-coverage guard.

- **F-IP4-001 (bootstrap frame_type pin absent)** — The production line
  `connector.go` setting `FrameType: frame.FrameTypePEConnect` was revertible to
  `halfchannel.FrameTypeData` without triggering any test failure — the entire GREEN suite
  passed with the mutant (property behaviorally inert within-story scope; no peer parses
  the bootstrap field yet). Forward obligation FO-PE-LOOP-001 would have remained
  revertible through the DRAIN-WIRE gap. Fixed by `TestConnector_BootstrapFrameTypePEConnect`
  (accept-and-read fixture, `io.ReadFull` 44 bytes + `ParseOuterHeader` + `FrameType`
  assertion; mutation verification: flip to `FrameTypeData` → `hdr.FrameType = 0x01, want
  FrameTypePEConnect (0x06)`).

**Certification audit (pass 7) — 5 load-bearing claims independently verified:**

| Claim | Verification method | Result |
|-------|---------------------|--------|
| (a) Receive goroutine leak-free across `Stop()` | Full exit-path trace — all paths reach `recvWg.Done()` before `Stop()` returns | VERIFIED |
| (b) Unconditional close → reconnect signal path | recv error → `conn.Close()` → `maintainConn` `SetWriteDeadline` error → `dialLoop` redial; bound ≤ keepalive + backoff; pinned by `ExitsOnReadError` `dialCount≥2` | VERIFIED |
| (c) Byte-contract bit-exact for all 6 frame types | `FuzzEncodeParseRoundTrip` + fresh-alloc `append` reconstruction path | VERIFIED |
| (d) Bootstrap pin false-pass-proof | `TestConnector_BootstrapFrameTypePEConnect` mutation kill at `connector.go` bootstrap line independently re-confirmed | VERIFIED |
| (e) Import-perimeter false-pass-proof | `TestUpstreamdialImportPerimeter` independently re-derived as non-bypassable | VERIFIED |

**14 implementation bars at final pass-7 state (all PASS):** ReadOuterFrame function, receive
goroutine started per established connection, bootstrap flip to `FrameTypePEConnect`, `DropCache`/
`FrameArrivalHandler`/`SetFrameCallback` wiring, `FrameFn` invocation discipline (discard-and-
continue), `conn.Close()` on read error, E-FWD-001 exhaustion via single-interface set,
lifecycle/doneCh, per-reconnect join, perimeter enforcement, bootstrap pin false-pass-proof,
byte-contract full-frame reconstruction, discard-continuation (not discard-and-close), flap-
cycle join-pin (runtime.NumGoroutine gate).

## Follow-On Stories

| Story | Relationship | Status |
|-------|-------------|--------|
| S-7.04-FU-DRAIN-WIRE | Unblocked by this delivery. Drain broadcast wire over PE connections requires an operational receive/forward loop on those connections (PO ruling F-P1-002 from S-7.04-FU-PE-CONNECTOR). | backlog |

## PR-Time Obligations

The following obligations are not defects in this delivery but must be completed at PR time
before merge:

1. **`gh pr update-branch`** — `develop` has advanced from merge-base `42baa8c`. Verify the
   delivery branch has zero file overlap with intervening commits before merge.

2. **SEC follow-on for PR description** — The PE receive `FrameFn` routes directly to
   `OnFrameArrival` bypassing `RouteFrame`'s HMAC admission check. PE upstream connections
   are established outbound by the connector (not arbitrary ingress), making this acceptable
   for this story's scope. Must be called out explicitly in the PR description as a noted
   security property for the reviewer; admission-on-PE-receive is revisited in the
   DRAIN-WIRE/session-bootstrap era per Q8 ruling.

3. **netingress behavioral-delta note for PR description** — Valid() widening to include
   `FrameTypePEConnect` (0x06) means a `pe_connect` frame arriving on network ingress now
   parses-and-drops fail-closed via E-ADM-016 (conn stays open) instead of
   teardown-on-parse-error. This is consistent with all sibling types (Data, Ctl, etc.)
   which also parse-and-drop on admission failure. No BC or VP asserts teardown-on-parse-error
   for pe_connect ingress. No change required. MUST be mentioned in PR description as the
   relevant behavioral delta visible to a reviewer (pass-7 LOW observation, per-story
   adversarial cycle).

4. **`merged_at` / `merge_pr` / `merge_sha` anchor true-up** — Update this DELIVERY
   document's frontmatter fields with actual merge timestamp, PR number, and merge commit SHA
   after `git merge`. Status field transitions `draft` → `final`.

## Demo Evidence

Location: `docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/` (commit `7b84c2b`)

| Item | Coverage |
|------|----------|
| `AC-001-receive-loop-active.tape` | `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` — frame from upstream fixture reaches `OnFrameArrival`; E-FWD-001 liveness observable (VHS source, POL-004 compliant) |
| `AC-002-framecallback-wired.tape` | `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` — SetFrameCallback closure wired to OnFrameArrival; `TestUpstreamdialImportPerimeter` — routing absent from upstreamdial transitive deps |
| `AC-003-peconnect-discrimination.tape` | `TestFrameType_Valid_PEConnect` + `TestConnector_BootstrapFrameTypePEConnect` + `TestConnector_ReceiveLoop_PEConnectFrameDiscarded` + `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` — constant, Valid(), bootstrap flip, discrimination |
| `AC-004-efwd001-exhaustion.tape` | `TestRunRouter_PE_EFWD001ExhaustionUnderLoad` + `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` — E-FWD-001 exhaustion; byte-contract + loop-continuation pin (≥2 emissions) |
| `AC-005-lifecycle-no-leak.tape` | `TestConnector_ReceiveLoop_ExitsOnConnClose` + `TestConnector_ReceiveLoop_ExitsOnReadError` + `TestConnector_ReceiveLoop_ExitsOnVersionMismatch` + `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` — goroutine lifecycle, per-reconnect join, no leak |
| `evidence-report.md` | Full AC discharge table; adversarial convergence summary (7 passes, 3/3 clean) |

No rendered binaries committed (`.gif`/`.webm` gitignored per POL-004).

## Sentinel / Spec Impact

**Wire protocol:** `frame.FrameTypePEConnect = 0x06` is a NEW wire-format constant.
The `frame_type` field in the outer header now has six canonical values (`0x01`–`0x06`)
instead of five. ARCH-02 §"Outer Header Format" and BC-2.01.004 Postcondition 2 amended
in spec-side commit `9792605` (same-commit parallel obligation; wire-format spec pair with
`frame.go` definition).

**Valid() widening — netingress behavioral delta:** `FrameType.Valid()` now returns `true`
for `0x06`. A `pe_connect` frame arriving on the network ingress data plane (`netingress.Serve`)
now parses successfully and is subsequently dropped by the HMAC admission check via E-ADM-016
(conn stays open), rather than triggering a teardown-on-parse-error response. This is
consistent with all other canonical frame types (Data, Ctl, Arq, Fec, EmptyTick) which all
parse-and-drop on admission failure. No BC text mandates teardown-on-parse-error for
`pe_connect` ingress. No change required; noted for reviewer awareness in PR description.

**ARCH-08 DAG:** `internal/upstreamdial` (position 19) gains a direct import edge to
`internal/frame` (position 2). §6.5 import set updated `{halfchannel, outerassembler}` →
`{frame, halfchannel, outerassembler}`; §6.6.2 forbidden-edges bullet replaced with binding
replacement (allowed imports `{frame, halfchannel, outerassembler}` at positions 2, 5, 8
only; cycle-freeness enumeration includes `frame` pos 2; F-P1-001 historical reconcile note
preserved). Forward edge only; no back-edges; no cycle.

**runRouter construction sequence amended:** `connector.SetFrameCallback(fn)` inserted
between `upstreamdial.New(...)` and `connector.Start()`. `runRouter` gains
`internal/multipath` import (new at this layer). No change to `runRouter`'s external
signature (`configPath string`, `sighupCh <-chan os.Signal`).

**testenv API:** Unchanged. `internal/testenv` is not modified by this story. The
`testenv.Restart` path does NOT call `SetFrameCallback` and is therefore prohibited for
any test asserting `OnFrameArrival` is reached (Q9.3 harness rule — enforced by
`TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival` and documented in story constraints).

**nil `ForwardFunc`:** `runRouter` wires `fn ForwardFunc` as nil in the `SetFrameCallback`
closure. In the single-interface set (`peIfaceID` is the only candidate), split-horizon
exhaustion fires on every non-bootstrap data frame before `fn` is ever invoked. This is
correct and annotated in code. FO-RECV-FWD-001 records the forward obligation for the
interface-set-widening story.

## Quality Gates

All gates green on code lane HEAD `7cedc34` (and docs-only HEAD `7b84c2b`):

- `just fmt` — clean
- `just lint` (`golangci-lint run ./...`) — 0 issues
- `just test-race` (`go test -race ./...`) — all 27 packages green; no sanctioned skips
- Full test suite: 14 net-new tests in 3 files all pass under `-race`
- `go vet ./...` — clean
- `gofmt -l ./...` — no diffs
- `TestScanForLine_DetectsEFWD001ProductionEmission` (existing normative pin in
  `cmd/switchboard/router_pe_connector_test.go`) — unmodified and green

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-11 | Initial DELIVERY document. Per-story adversarial cycle CONVERGED 3/3 (passes 5–7 of 7; 6 findings remediated). Demo evidence at `7b84c2b` (POL-004). Status: draft — pending merge. |
