# Demo Evidence Report — S-7.04-FU-DRAIN-WIRE

**Story:** DRAIN-over-SVTN wire propagation — per-node observer registration in drain coordinator
**Story version:** v1.11
**Branch:** feature/S-7.04-FU-DRAIN-WIRE
**Code HEAD:** e7614d7
**Adversarial convergence:** step 4.5 CONVERGED 3/3
**Evidence date:** 2026-07-12

---

## POL-004 Note

Per `docs/DEMO-EVIDENCE-POLICY.md` (ratified 2026-07-04), rendered binaries
(`.gif`, `.webm`, `.mp4`, `.png`) are **not committed**. `.tape` scripts and
this evidence report are the source of truth. To regenerate locally:

```bash
vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-001-drain-frame-assembly-send.tape
vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-002-onaccept-seam-registration-cleanup.tape
vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-003-startup-observer-ctl-guard.tape
vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-004-vp037-stage1-wire-roundtrip.tape
vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-005-panic-recovery-forced-exit.tape
```

This story is a daemon/library story. The primary evidence is targeted test
execution; tape scripts provide a reproducible terminal replay recipe. All
test commands below are run from the worktree root.

---

## Summary Table

| AC | Tape | Discharge status | Test(s) |
|----|------|-----------------|---------|
| AC-001 | `AC-001-drain-frame-assembly-send.tape` | FULL | `TestDrainObserver_AssemblesAndSendsDRAINFrame` |
| AC-002 | `AC-002-onaccept-seam-registration-cleanup.tape` | FULL | `TestNetingress_OnAccept_RegistersNodeHandle`, `TestRunRouter_NodeConnClose_CleansUpSendMap` |
| AC-003 | `AC-003-startup-observer-ctl-guard.tape` | FULL | `TestDrainObserver_RegisteredAtStartup_FiresOnSignal`, `TestRouter_CtlFrame_ShortPayload_NoConnClose`, `TestRouter_CtlFrame_UnknownControlType_SilentIgnore` |
| AC-004 | `AC-004-vp037-stage1-wire-roundtrip.tape` | FULL (VP-037 stage-1; `verification_lock` stays `false`) | `TestE2E_RouterDrain_WireRoundTrip` |
| AC-005 | `AC-005-panic-recovery-forced-exit.tape` | FULL | `TestDrain_ObserverPanicRecovery` (subprocess-isolated), `TestRunRouter_ForcedExitPastDrainTimeout` (pre-existing) |

---

## AC-001 — DRAIN frame assembled and sent to every connected PE node on drainCoord.Signal

**Tape:** `AC-001-drain-frame-assembly-send.tape`
**BC anchors:** BC-2.09.002 PC-1, BC-2.01.008 PC-2; placement note Q1, Q-SEAM, Q-SINGLE-OBS
**Test file:** `cmd/switchboard/router_drain_wire_test.go`

The single startup drain observer — registered once at `runRouter` setup — iterates the
live per-node send map at `Signal` time. For each registered node it assembles a
`FrameTypeCtl` (0x03) outer frame carrying the 4-byte DRAIN payload
(`control_type=0x01, version=0x01, reserved=0x0000`) via `frame.EncodeOuterHeader`, and
non-blocking-sends it to the node's send channel. The writer goroutine flushes it onto
the wire; a simulated node reads back `FrameTypeCtl` with `control_type=0x01` within 2s.
`drainCoord.Wait` returns `nil` when the observer completes.

**Evidence command:**

```
go test -race -run TestDrainObserver_AssemblesAndSendsDRAINFrame -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestDrainObserver_AssemblesAndSendsDRAINFrame
--- PASS: TestDrainObserver_AssemblesAndSendsDRAINFrame (0.02s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	1.578s
```

**Discharge status:** FULL.

---

## AC-002 — netingress OnAccept seam: per-node connection registers in send map; writer goroutine is sole writer; cleanup deregisters

**Tape:** `AC-002-onaccept-seam-registration-cleanup.tape`
**BC anchors:** BC-2.09.002 PC-1; placement note Q-SEAM (ownership split, F-DW-SP3-001)
**Test file:** `cmd/switchboard/router_drain_wire_test.go`

`netingress.Serve` owns DATA creation: it allocates `IfaceID` from the
`ServeConfig.IfaceIDSeed`-seeded counter, creates `Send`/`Done`, and populates a
fully-built `NodeHandle` before calling `OnAccept` — fired only for ADMITTED connections,
from the freshly spawned per-conn goroutine. `nodeConnHook` records a
`nodeConnRegistered` event for the accepted connection. On conn close, the
behavior-cleanup `func()` deletes the `sync.Map` entry, fires
`nodeConnHook(nodeConnRemoved, ...)`, and closes `done` via `doneOnce` — `send` is
never closed (single-closer/no-send-after-close invariant).

**Evidence command:**

```
go test -race -run 'TestNetingress_OnAccept_RegistersNodeHandle|TestRunRouter_NodeConnClose_CleansUpSendMap' -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestNetingress_OnAccept_RegistersNodeHandle
--- PASS: TestNetingress_OnAccept_RegistersNodeHandle (0.02s)
=== RUN   TestRunRouter_NodeConnClose_CleansUpSendMap
--- PASS: TestRunRouter_NodeConnClose_CleansUpSendMap (0.02s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	1.395s
```

**Discharge status:** FULL.

---

## AC-003 — Single startup observer registered; observer fires; FO-DRAIN-WIRE-001 forward-compat guard; ctl-short-payload + unknown-control_type pins

**Tape:** `AC-003-startup-observer-ctl-guard.tape`
**BC anchors:** BC-2.09.002 PC-1; BC-2.01.008 PC-4 + Invariant 2; placement note
Q-SINGLE-OBS, Q-AC003, Q-CTL-GUARD
**Test file:** `cmd/switchboard/router_drain_wire_test.go`

The single observer is registered at `drainCoord`-construction time — guaranteed to
precede `drainCoord.Signal` — and fires `drainObserverFiredHook` as the first statement
of its body on `Signal`, independent of any live node connection. Two ctl-guard pins:
a `FrameTypeCtl` frame with `payload_len<4` is silently discarded (E-PRT-002 log, no
connection close, per Q-CTL-GUARD/EC-002); a `FrameTypeCtl` frame with an unrecognized
`control_type` is silently ignored with NO logging and no connection close, pinning
BC-2.01.008 PC-4 / Invariant 2 / the FO-DRAIN-WIRE-001 forward-compat rule.

**Evidence command:**

```
go test -race -run 'TestDrainObserver_RegisteredAtStartup_FiresOnSignal|TestRouter_CtlFrame_ShortPayload_NoConnClose|TestRouter_CtlFrame_UnknownControlType_SilentIgnore' -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestDrainObserver_RegisteredAtStartup_FiresOnSignal
--- PASS: TestDrainObserver_RegisteredAtStartup_FiresOnSignal (0.02s)
=== RUN   TestRouter_CtlFrame_ShortPayload_NoConnClose
--- PASS: TestRouter_CtlFrame_ShortPayload_NoConnClose (0.23s)
=== RUN   TestRouter_CtlFrame_UnknownControlType_SilentIgnore
--- PASS: TestRouter_CtlFrame_UnknownControlType_SilentIgnore (0.23s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	1.876s
```

**Discharge status:** FULL.

---

## AC-004 — VP-037 stage-1 discharge: TestE2E_RouterDrain_WireRoundTrip, untagged, drainCoord.Wait nil within window; verification_lock stays false

**Tape:** `AC-004-vp037-stage1-wire-roundtrip.tape`
**BC anchors:** VP-037 stage-1; BC-2.09.002 PC-4; BC-2.01.008 PC-2; placement note Q4-AMENDED
**Test file:** `cmd/switchboard/router_drain_wire_test.go` (untagged — runs in standard `go test ./...`)

An untagged end-to-end test proves the full production path: a real `runRouter`
(`startRunRouterWithConfig`) is started; a simulated node dials `cfg.ListenAddr` and is
admitted via the `nodeConnHook` accept/register barrier; `cancel()` drives the
production shutdown block through `drainCoord.Signal`; the simulated node
deterministically reads `FrameTypeCtl` with `payload[0]=0x01` (DRAIN) within 2s, made
non-racy by the Shutdown ordering guarantee's flush-before-teardown sequencing.
`drainCoord.Wait` returns `nil` within the default drain window. This discharges VP-037
stage-1 (wire round-trip); `verification_lock` stays `false` — stage-2 (migration
end-to-end) is a named follow-on story.

**Evidence command:**

```
go test -race -run TestE2E_RouterDrain_WireRoundTrip -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestE2E_RouterDrain_WireRoundTrip
--- PASS: TestE2E_RouterDrain_WireRoundTrip (0.03s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	1.342s
```

**Discharge status:** FULL (VP-037 stage-1; `verification_lock` remains `false` per story
design — stage-2 discharge is out of scope for this story).

---

## AC-005 — EC-003 forced-exit via the existing TestRunRouter_ForcedExitPastDrainTimeout + drain.Signal panic recovery

**Tape:** `AC-005-panic-recovery-forced-exit.tape`
**BC anchors:** BC-2.09.002 EC-003; placement note Q5, Q-AC005
**Test files:** `internal/drain/drain_test.go`, `cmd/switchboard/router_drain_test.go`

`drain.Signal`'s fan-out goroutine gains a `defer recover()` wrapper: a panicking
observer no longer crashes the coordinator, and `Wait` still returns within the window
(either `nil` or `ErrTimeout`, never a hang). `TestDrain_ObserverPanicRecovery` proves
this via subprocess isolation — it re-execs the test binary with
`GO_WANT_DRAIN_PANIC_HELPER=1` to drive the actual panicking-observer scenario in a
child process, so a pre-fix panic crashes only the child (observed as a non-nil error
from `CombinedOutput`) rather than taking down the whole test binary. EC-003
(unresponsive registered observer forces exit past `cfg.DrainTimeout`) is evidenced by
the pre-existing `TestRunRouter_ForcedExitPastDrainTimeout`, left unchanged by this
story per Q-AC005/F-DW-SP3-007 — the existing test already discharges the postcondition
via `drainCoordHook` + a blocked observer + `cfg.DrainTimeout`, asserting the EC-003 log
marker and elapsed-time bounds.

**Evidence command (panic recovery):**

```
go test -race -run TestDrain_ObserverPanicRecovery -count=1 -v ./internal/drain/
```

**Captured output:**

```
=== RUN   TestDrain_ObserverPanicRecovery
=== PAUSE TestDrain_ObserverPanicRecovery
=== CONT  TestDrain_ObserverPanicRecovery
--- PASS: TestDrain_ObserverPanicRecovery (1.02s)
PASS
ok  	github.com/arcavenae/switchboard/internal/drain	2.360s
```

**Evidence command (EC-003 forced-exit, pre-existing):**

```
go test -race -run TestRunRouter_ForcedExitPastDrainTimeout -count=1 -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestRunRouter_ForcedExitPastDrainTimeout
--- PASS: TestRunRouter_ForcedExitPastDrainTimeout (0.53s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	1.838s
```

**Discharge status:** FULL.

---

## Adversarial Convergence

This story reached step 4.5 adversarial convergence at 3/3 clean passes as of code
HEAD `e7614d7` on `feature/S-7.04-FU-DRAIN-WIRE`. All five acceptance criteria above
are proven by tests passing under `go test -race`.
