# Demo Evidence Report — S-BL.PE-RECEIVE-LOOP

**Story:** PE receive/forward loop — frame.ReadOuterFrame, FrameTypePEConnect, receive goroutine, SetFrameCallback wiring  
**Story version:** v1.25  
**Branch:** story/s-bl-pe-receive-loop  
**Code HEAD:** 7cedc34  
**Note version:** v1.22  
**Story-index version:** v4.65 (impl 7cedc34)  
**Adversarial convergence:** 3/3 clean passes (passes 5–7) over 7 total passes  
**Evidence date:** 2026-07-11

---

## POL-004 Note

Per `docs/DEMO-EVIDENCE-POLICY.md` (ratified 2026-07-04), rendered binaries
(`.gif`, `.webm`, `.mp4`, `.png`) are **not committed**. `.tape` scripts and
this evidence report are the source of truth. To regenerate locally:

```bash
vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-001-receive-loop-active.tape
vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-002-framecallback-wired.tape
vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-003-peconnect-discrimination.tape
vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-004-efwd001-exhaustion.tape
vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-005-lifecycle-no-leak.tape
```

This story is a daemon/library story. The primary evidence is targeted test
execution; tape scripts provide a reproducible terminal replay recipe. All
test commands below are run from the worktree root.

---

## Summary Table

| AC | Tape | Discharge status | Test(s) |
|----|------|-----------------|---------|
| AC-001 | `AC-001-receive-loop-active.tape` | FULL | `TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect` |
| AC-002 | `AC-002-framecallback-wired.tape` | FULL | `TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival`, `TestUpstreamdialImportPerimeter` |
| AC-003 | `AC-003-peconnect-discrimination.tape` | FULL | `TestFrameType_Valid_PEConnect`, `TestConnector_BootstrapFrameTypePEConnect`, `TestConnector_ReceiveLoop_PEConnectFrameDiscarded`, `TestConnector_ReceiveLoop_CtlFrameForwardedToCallback` |
| AC-004 | `AC-004-efwd001-exhaustion.tape` | FULL | `TestRunRouter_PE_EFWD001ExhaustionUnderLoad`, `TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader` |
| AC-005 | `AC-005-lifecycle-no-leak.tape` | FULL | `TestConnector_ReceiveLoop_ExitsOnConnClose`, `TestConnector_ReceiveLoop_ExitsOnReadError`, `TestConnector_ReceiveLoop_ExitsOnVersionMismatch`, `TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak` |

---

## AC-001 — Receive loop active after connect (E-FWD-001 liveness via runRouter)

**Tape:** `AC-001-receive-loop-active.tape`  
**BC anchors:** BC-2.09.001 (receive/forward loop), E-FWD-001 liveness  
**Test files:** `cmd/switchboard/router_pe_receive_loop_test.go`

After a PE connect, `runRouter` (via `mgmt_wire.go`) starts a receive goroutine
on the upstream connection. The integration test verifies the goroutine is active
by injecting a frame via the upstream fixture and confirming `OnFrameArrival` is
called (E-FWD-001 liveness observable).

**Evidence command:**

```
go test ./cmd/switchboard/ -run TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect -count=1 -v
```

**Captured output:**

```
=== RUN   TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect
--- PASS: TestRunRouter_PE_ReceiveLoop_ActiveAfterConnect (0.05s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.480s
```

**Discharge status:** FULL.

---

## AC-002 — FrameCallback wired to OnFrameArrival + import perimeter

**Tape:** `AC-002-framecallback-wired.tape`  
**BC anchors:** BC-2.09.001 (frame forwarding wire), import perimeter  
**Test files:** `cmd/switchboard/router_pe_receive_loop_test.go`, `internal/upstreamdial/import_perimeter_test.go`

`runRouter` calls `connector.SetFrameCallback(routing.OnFrameArrival)` so that
frames received on the PE upstream connection are forwarded to the routing layer.
The integration test verifies the callback is exercised. The import perimeter test
confirms `internal/upstreamdial` does not directly import `internal/routing` (the
callback is injected via function parameter, maintaining purity of the upstreamdial
package).

**Evidence command (callback wiring):**

```
go test ./cmd/switchboard/ -run TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival -count=1 -v
```

**Captured output:**

```
=== RUN   TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival
--- PASS: TestRunRouter_PE_FrameCallback_WiredToOnFrameArrival (0.05s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.330s
```

**Evidence command (import perimeter):**

```
go test ./internal/upstreamdial/ -run TestUpstreamdialImportPerimeter -count=1 -v
```

**Captured output:**

```
=== RUN   TestUpstreamdialImportPerimeter
--- PASS: TestUpstreamdialImportPerimeter (0.05s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	0.335s
```

**Discharge status:** FULL.

---

## AC-003 — FrameTypePEConnect constant, Valid(), bootstrap-flip, discrimination

**Tape:** `AC-003-peconnect-discrimination.tape`  
**BC anchors:** BC-2.09.001 (PEConnect discrimination), frame.FrameType validity  
**Test files:** `internal/frame/frame_type_test.go`, `internal/upstreamdial/connector_test.go`

`frame.FrameTypePEConnect` (0x06) is defined as a distinct constant; `Valid()` returns
`true` for it. The bootstrap frame type is flipped from the prior `halfchannel.FrameTypeData`
placeholder to `FrameTypePEConnect`. The receive loop discards frames with type
`FrameTypePEConnect` (does not forward to callback — they are bootstrap/handshake frames
not routing data). Control frames with other types are forwarded to the callback.

**Evidence command (constant + Valid):**

```
go test ./internal/frame/ -run TestFrameType_Valid_PEConnect -count=1 -v
```

**Captured output:**

```
=== RUN   TestFrameType_Valid_PEConnect
=== PAUSE TestFrameType_Valid_PEConnect
=== CONT  TestFrameType_Valid_PEConnect
--- PASS: TestFrameType_Valid_PEConnect (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/frame	0.281s
```

**Evidence command (bootstrap-flip + discard + forward):**

```
go test ./internal/upstreamdial/ -run 'TestConnector_BootstrapFrameTypePEConnect|TestConnector_ReceiveLoop_PEConnectFrameDiscarded|TestConnector_ReceiveLoop_CtlFrameForwardedToCallback' -count=1 -v
```

**Captured output:**

```
=== RUN   TestConnector_ReceiveLoop_PEConnectFrameDiscarded
--- PASS: TestConnector_ReceiveLoop_PEConnectFrameDiscarded (0.01s)
=== RUN   TestConnector_ReceiveLoop_CtlFrameForwardedToCallback
--- PASS: TestConnector_ReceiveLoop_CtlFrameForwardedToCallback (0.01s)
=== RUN   TestConnector_BootstrapFrameTypePEConnect
--- PASS: TestConnector_BootstrapFrameTypePEConnect (0.01s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	0.280s
```

**Discharge status:** FULL.

---

## AC-004 — E-FWD-001 exhaustion discharge under load (peWriteFixture injection, no false duplicate suppression)

**Tape:** `AC-004-efwd001-exhaustion.tape`  
**BC anchors:** BC-2.02.008 (split-horizon drop + E-FWD-001 log), E-FWD-001 full discharge  
**Test files:** `cmd/switchboard/router_pe_receive_loop_test.go`

This story completes the E-FWD-001 discharge re-anchored from S-7.04-FU-PE-CONNECTOR
(declared divergence D-3 in that story). The `peWriteFixture` injection enables controlled
frame injection directly into the receive loop path. Under load (multiple frames all
exhausting available paths), E-FWD-001 is emitted for each qualifying frame. The
no-duplicate-suppression test confirms that frames with distinct outer headers each
independently produce E-FWD-001 (no false deduplication keyed on frame content).

**Evidence command:**

```
go test ./cmd/switchboard/ -run 'TestRunRouter_PE_EFWD001ExhaustionUnderLoad|TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader' -count=1 -v
```

**Captured output:**

```
=== RUN   TestRunRouter_PE_EFWD001ExhaustionUnderLoad
--- PASS: TestRunRouter_PE_EFWD001ExhaustionUnderLoad (0.05s)
=== RUN   TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader
--- PASS: TestPEReceiveLoop_NoDuplicateSuppression_DifferentOuterHeader (0.05s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.354s
```

**Discharge status:** FULL. Completes the E-FWD-001 postcondition-1 discharge
re-anchored from S-7.04-FU-PE-CONNECTOR AC-004 (F-P1-002 resolved).

---

## AC-005 — Receive goroutine lifecycle (exit on close/error, flap-cycle no-leak, per-reconnect join)

**Tape:** `AC-005-lifecycle-no-leak.tape`  
**BC anchors:** BC-2.09.001 (goroutine lifecycle), goroutine leak prevention  
**Test files:** `internal/upstreamdial/connector_test.go`

The receive goroutine started per-connection must exit cleanly under three termination
conditions: (1) connection closed (EOF/io.ErrClosedPipe), (2) read error (non-close
error), (3) version mismatch (protocol-level error). The flap-cycle test verifies that
across multiple connect/disconnect/reconnect cycles, the previous goroutine is joined
before a new one is spawned — no goroutine leak across reconnects.

**Evidence command:**

```
go test ./internal/upstreamdial/ -run 'TestConnector_ReceiveLoop_ExitsOnConnClose|TestConnector_ReceiveLoop_ExitsOnReadError|TestConnector_ReceiveLoop_ExitsOnVersionMismatch|TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak' -count=1 -v
```

**Captured output:**

```
=== RUN   TestConnector_ReceiveLoop_ExitsOnConnClose
--- PASS: TestConnector_ReceiveLoop_ExitsOnConnClose (0.00s)
=== RUN   TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak
--- PASS: TestConnector_ReceiveLoop_FlapCycleJoin_NoLeak (0.07s)
=== RUN   TestConnector_ReceiveLoop_ExitsOnReadError
--- PASS: TestConnector_ReceiveLoop_ExitsOnReadError (0.03s)
=== RUN   TestConnector_ReceiveLoop_ExitsOnVersionMismatch
--- PASS: TestConnector_ReceiveLoop_ExitsOnVersionMismatch (0.03s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	0.378s
```

**Discharge status:** FULL.

---

## Adversarial Convergence

This story completed 7 adversarial passes with 3/3 clean passes at the end
(passes 5, 6, and 7). No new findings were raised in the final three passes.
Convergence was declared after the 3-clean-pass threshold per story v1.25
adversarial protocol.
