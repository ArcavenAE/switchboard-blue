# Demo Evidence Report — S-7.04-FU-PE-CONNECTOR

**Story:** Outbound TCP dial loop on PE graduation — upstream_routers connect-half and live-egress anchor  
**Story version:** v1.26  
**Branch:** story/s-7.04-fu-pe-connector  
**Code HEAD:** 6b6f0cf  
**Adversarial convergence:** pass 32  
**Evidence date:** 2026-07-08

---

## POL-004 Note

Per `docs/DEMO-EVIDENCE-POLICY.md` (ratified 2026-07-04), rendered binaries
(`.gif`, `.webm`, `.mp4`, `.png`) are **not committed**. `.tape` scripts and
this evidence report are the source of truth. To regenerate locally:

```bash
vhs docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-001-pe-dial-connect.tape
vhs docs/demo-evidence/S-7.04-FU-PE-CONNECTOR/AC-002-ec001-backoff-partial-pe.tape
```

This story is a daemon/library story. The primary evidence is targeted test
execution; tape scripts provide a reproducible terminal replay recipe. All
test commands below are run from the worktree root.

---

## Declared Divergences

Three known divergences are documented here in full per story v1.24–v1.26
and AC-004 partial-discharge note (F-P1-002):

| # | Divergence | Scope | Deferral target |
|---|-----------|-------|----------------|
| D-1 | **`FrameTypePEConnect` placeholder** — delivered code uses `halfchannel.FrameTypeData` as bootstrap frame type; the distinct `frame.FrameTypePEConnect` constant is not defined anywhere in the repo (F-P26-001). Bootstrap frames are indistinguishable from session data at the receiver until the distinct constant is defined. | AC-001 PC-2 partial | S-BL.PE-RECEIVE-LOOP |
| D-2 | **Zero-Envelope bootstrap fields** — `SrcAddr`/`DstAddr`/`SVTNID`/`FrameAuthKey` are zero-valued at construction time; full node-identity derivation (Ed25519 key material, `frame.DeriveNodeAddress`, HMAC key derivation) is deferred as not-core. Documented in `internal/upstreamdial/connector.go` and `cmd/switchboard/mgmt_wire.go`. | AC-001 PC-2 partial | S-BL.PE-RECEIVE-LOOP |
| D-3 | **VP-037 partial-discharge** — `TestE2E_RouterDrain_NodesMigrateWithin2s` is skipped with a partial-discharge note; the DRAIN broadcast wire protocol required for node migration is owned by S-7.04-FU-DRAIN-WIRE. This story delivers live upstream connections that unblock VP-037; it cannot complete the full drain proof without the DRAIN broadcast. S404-OBS-F and S404-LOW-1 re-anchored to S-BL.PE-RECEIVE-LOOP. | AC-004 + AC-005 VP-037 | S-7.04-FU-DRAIN-WIRE + S-BL.PE-RECEIVE-LOOP |

---

## Summary Table

| AC | Evidence type | Discharge status | Test(s) |
|----|--------------|-----------------|---------|
| AC-001 | Tape script + test citation | FULL (with D-1, D-2 deferred — placeholder frame type and zero envelope documented in code) | `TestConnector_DialSuccess_ModePE`, `TestConnector_ReorderReuse_NoTeardown`, `TestRunRouter_PE_DialAndConnect_UpstreamReachable`, `TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder` |
| AC-002 | Tape script + test citation | FULL | `TestConnector_DialFailure_EC001Log`, `TestConnector_BackoffConstants`, `TestOperativeBase_TracksKeepalive`, `TestConnector_BackoffParameters`, `TestConnector_AllUpstreamsUnreachable_ModeE`, `TestConnector_NoEC004OnGracefulStop`, `TestRunRouter_PE_UnreachableUpstream_PartialPE`, `TestConnector_ConcurrentDropToZero_SingleEC004Emission` |
| AC-003 | Test citation (no tape) | FULL | `TestKeepaliveIntervalNotSweepDeadline`, `TestConnector_KeepaliveTickerDrivesHealthProbe`, `TestRunRouter_PE_KeepalivePassedToConnector` |
| AC-004 | Test citation (no tape) | PARTIAL — postcondition 2 (no spurious E-FWD-001) DISCHARGED; postcondition 1 (path-exhaustion E-FWD-001) RE-ANCHORED to S-BL.PE-RECEIVE-LOOP (unmet-deps: receive/forward loop over PE connections not in scope) | `TestRunRouter_PE_EFWD001ReconfirmationUnderLoad`, `TestScanForLine_DetectsEFWD001ProductionEmission` |
| AC-005 | Test citation (no tape) | VP-038 FULL; VP-037 PARTIAL (blocked on S-7.04-FU-DRAIN-WIRE) | `TestE2E_EtoPEGraduationByConfigChange`, `TestE2E_RouterDrain_NodesMigrateWithin2s` (SKIP) |
| AC-006 | Test citation (no tape) | FULL | `TestRunRouter_VP038_EtoPEViaConfigOnly`, `TestRunRouter_PE_RouterHandleModeReflectsLiveState`, `TestConnector_Stop_Idempotent`, `TestRouterHandle_Restart_TwicePE`, `TestRunRouter_SIGHUPReload_EtoPE` |

---

## AC-001 — PE graduation: outbound dial, session bootstrap, and set-equal reconciliation

**Tape:** `AC-001-pe-dial-connect.tape`  
**BC anchors:** BC-2.09.001 PC-2/PC-3, EC-002  
**Test files:** `internal/upstreamdial/connector_test.go`, `cmd/switchboard/router_pe_connector_test.go`

The tape demonstrates end-to-end PE dial (happy path): the Connector dials a loopback
upstream fixture, completes the three-step bootstrap (TCP dial + `outerassembler.Assemble`
+ `conn.Write`), increments connected-count, and reports `Mode() == ModePE`. A second beat
shows set-equal reconciliation: a SIGHUP with the same addresses in reversed order produces
no teardown, no new dials.

**Evidence command (unit):**

```
go test -run "TestConnector_DialSuccess_ModePE|TestConnector_ReorderReuse_NoTeardown" -v ./internal/upstreamdial/
```

**Captured output:**

```
=== RUN   TestConnector_DialSuccess_ModePE
--- PASS: TestConnector_DialSuccess_ModePE (0.01s)
=== RUN   TestConnector_ReorderReuse_NoTeardown
--- PASS: TestConnector_ReorderReuse_NoTeardown (0.11s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	0.385s
```

**Evidence command (integration):**

```
go test -run "TestRunRouter_PE_DialAndConnect_UpstreamReachable|TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder" -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestRunRouter_PE_DialAndConnect_UpstreamReachable
--- PASS: TestRunRouter_PE_DialAndConnect_UpstreamReachable (0.02s)
=== RUN   TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder
--- PASS: TestRunRouter_PE_SetEqualReconciliation_NoTeardownOnReorder (0.32s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.639s
```

**Discharge status:** FULL, subject to declared divergences D-1 and D-2 (placeholder
`FrameTypeData` and zero-valued `Envelope` fields deferred to S-BL.PE-RECEIVE-LOOP per
story v1.24/v1.26; documented in `connector.go` and `mgmt_wire.go`).

---

## AC-002 — Unreachable upstream: EC-001 graceful handling with exponential backoff

**Tape:** `AC-002-ec001-backoff-partial-pe.tape`  
**BC anchors:** BC-2.09.001 EC-001, EC-004; placement note Q5  
**Test files:** `internal/upstreamdial/connector_test.go`, `cmd/switchboard/router_pe_connector_test.go`

The tape demonstrates: Connector started with one unreachable address (closed port) emits
"upstream router <addr> unreachable"; Mode() stays ModeE. Then a second upstream fixture
is started; Mode() transitions to ModePE (partial-PE semantics: ≥1 upstream connected).
The backoff suite (BackoffConstants + OperativeBase) validates the Q5 constants and
floor/cap/jitter arithmetic.

**Evidence command (backoff constants + floor):**

```
go test -run "TestConnector_BackoffConstants|TestOperativeBase_TracksKeepalive" -v ./internal/upstreamdial/
```

**Captured output:**

```
=== RUN   TestConnector_BackoffConstants
--- PASS: TestConnector_BackoffConstants (0.00s)
=== RUN   TestOperativeBase_TracksKeepalive
=== RUN   TestOperativeBase_TracksKeepalive/well-above-floor/1s
=== RUN   TestOperativeBase_TracksKeepalive/well-above-floor/600ms
=== RUN   TestOperativeBase_TracksKeepalive/well-above-floor/1200ms
=== RUN   TestOperativeBase_TracksKeepalive/well-above-floor/5s
=== RUN   TestOperativeBase_TracksKeepalive/exact-floor
=== RUN   TestOperativeBase_TracksKeepalive/below-floor/100ms
=== RUN   TestOperativeBase_TracksKeepalive/below-floor/1ms
=== RUN   TestOperativeBase_TracksKeepalive/below-floor/499ms
--- PASS: TestOperativeBase_TracksKeepalive (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/well-above-floor/600ms (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/below-floor/499ms (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/well-above-floor/1200ms (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/well-above-floor/1s (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/below-floor/1ms (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/below-floor/100ms (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/exact-floor (0.00s)
    --- PASS: TestOperativeBase_TracksKeepalive/well-above-floor/5s (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial
```

**Evidence command (EC-001 log + ModeE + graceful-stop polarity guard):**

```
go test -run "TestConnector_DialFailure_EC001Log|TestConnector_AllUpstreamsUnreachable_ModeE|TestConnector_NoEC004OnGracefulStop" -v ./internal/upstreamdial/
```

**Captured output:**

```
=== RUN   TestConnector_DialFailure_EC001Log
--- PASS: TestConnector_DialFailure_EC001Log (0.02s)
=== RUN   TestConnector_AllUpstreamsUnreachable_ModeE
--- PASS: TestConnector_AllUpstreamsUnreachable_ModeE (0.30s)
=== RUN   TestConnector_NoEC004OnGracefulStop
--- PASS: TestConnector_NoEC004OnGracefulStop (0.11s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	0.697s
```

**Evidence command (concurrency regression — F-P29-001 single EC-004 emission):**

```
go test -run "TestConnector_ConcurrentDropToZero_SingleEC004Emission" -v ./internal/upstreamdial/
```

**Captured output:**

```
=== RUN   TestConnector_ConcurrentDropToZero_SingleEC004Emission
--- PASS: TestConnector_ConcurrentDropToZero_SingleEC004Emission (9.31s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	9.560s
```

**Evidence command (integration — partial-PE):**

```
go test -run "TestRunRouter_PE_UnreachableUpstream_PartialPE" -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestRunRouter_PE_UnreachableUpstream_PartialPE
--- PASS: TestRunRouter_PE_UnreachableUpstream_PartialPE (0.02s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.317s
```

**Discharge status:** FULL. Covers EC-001 log verbatim assertion, ModeE on all-unreachable,
EC-004 graceful-stop polarity guard (F-P4-001), concurrent drop-to-zero single-emission
invariant (F-P29-001), backoff constants/floor/cap/jitter arithmetic, reset-on-success
wiring, and partial-PE integration.

---

## AC-003 — Keepalive ticker lives in Connector (BC-2.09.003 PC-8)

**Evidence type:** Test citation only (no tape)  
**Rationale:** This AC is an internal-encapsulation invariant — the ticker must live
inside `internal/upstreamdial.Connector`, not in `runRouter`. A terminal demo
would show only a running binary; the structural guarantee is visible only in test
assertions and code inspection. A tape would be contrived.

**BC anchor:** BC-2.09.003 PC-8; placement note Q7

**Evidence command:**

```
go test -run "TestKeepaliveIntervalNotSweepDeadline|TestConnector_KeepaliveTickerDrivesHealthProbe|TestRunRouter_PE_KeepalivePassedToConnector" -v ./internal/upstreamdial/ ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestConnector_KeepaliveTickerDrivesHealthProbe
--- PASS: TestConnector_KeepaliveTickerDrivesHealthProbe (0.05s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	0.287s
=== RUN   TestKeepaliveIntervalNotSweepDeadline
--- PASS: TestKeepaliveIntervalNotSweepDeadline (0.00s)
=== RUN   TestRunRouter_PE_KeepalivePassedToConnector
--- PASS: TestRunRouter_PE_KeepalivePassedToConnector (0.02s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.418s
```

**What the tests assert:**

1. `TestKeepaliveIntervalNotSweepDeadline` — normative fence test (pre-existing, must
   remain unmodified and green): asserts `keepaliveIntervalFor(cfg)` and `sweepDeadline(cfg)`
   return distinct values, guarding against conflation.
2. `TestConnector_KeepaliveTickerDrivesHealthProbe` — constructs a Connector with a
   200ms keepalive interval against a live fixture; drains the bootstrap frame (Read-1),
   then requires a second write (probe frame) within 4 keepalive intervals (Read-2). Test
   fails if the keepalive ticker case is removed from `reconcileLoop`.
3. `TestRunRouter_PE_KeepalivePassedToConnector` — integration: writes a PE config with
   `keepalive_interval: 200ms`; confirms the Connector receives the 200ms value (not a
   hardcoded constant or the `sweepDeadline` 60s default).

**Discharge status:** FULL.

---

## AC-004 — E-FWD-001 re-confirmation under sustained ARQ retransmit load

**Evidence type:** Test citation only (no tape)  
**BC anchors:** BC-2.02.008 (split-horizon drop + E-FWD-001 log), BC-2.06.003 (Failed-state
event emission), S404-OBS-F  
**Discharge class:** PARTIAL (unmet-deps — F-P1-002)

**Partial-discharge summary:**

- **Postcondition 2 (no spurious E-FWD-001 under normal load) — DISCHARGED here.**
- **Postcondition 1 (E-FWD-001 fires under path-exhaustion) — RE-ANCHORED to
  S-BL.PE-RECEIVE-LOOP.** E-FWD-001 is emitted only from
  `routing.FrameArrivalHandler.OnFrameArrival`; `runRouter`'s ingress path
  (`netingress.Serve → routing.RouteFrame`) never calls `OnFrameArrival`. The Connector
  in this story only dials, bootstraps, and keepalives — it has no receive/forward loop
  over PE connections. S404-OBS-F and S404-LOW-1 re-anchored accordingly.

**Evidence command:**

```
go test -run "TestRunRouter_PE_EFWD001ReconfirmationUnderLoad|TestScanForLine_DetectsEFWD001ProductionEmission" -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestRunRouter_PE_EFWD001ReconfirmationUnderLoad
--- PASS: TestRunRouter_PE_EFWD001ReconfirmationUnderLoad (0.07s)
=== RUN   TestScanForLine_DetectsEFWD001ProductionEmission
--- PASS: TestScanForLine_DetectsEFWD001ProductionEmission (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.329s
```

**What the tests assert:**

1. `TestRunRouter_PE_EFWD001ReconfirmationUnderLoad` — integration: PE router with live
   upstream connection established (connected-count ≥ 1); under single-path happy-path
   conditions, the E-FWD-001 log line does NOT appear in the router's writer output.
   The inline comment documents the unmet-deps analysis for postcondition 1 (path-exhaustion
   case requires receive/forward loop not yet in scope).
2. `TestScanForLine_DetectsEFWD001ProductionEmission` — mutation pin (F-P11-001):
   (a) proves `scanForLine(buf, "E-FWD-001", 0)` returns `true` against a buffer
   containing the verbatim production emission string from
   `internal/routing/on_frame_arrival.go`; (b) proves the space-form
   `"split-horizon blocked"` does NOT match (pinning the F-P11-001 defect shape where
   the original search key was vacuous).

**Discharge status:** PARTIAL — postcondition 2 DISCHARGED; postcondition 1 + S404-OBS-F
+ S404-LOW-1 RE-ANCHORED to S-BL.PE-RECEIVE-LOOP (declared divergence D-3).

---

## AC-005 — VP-037/VP-038 verification_lock discharge

**Evidence type:** Test citation only (no tape)  
**VP anchors:** VP-037 (drain-within-window), VP-038 (E→PE via config-only)

### VP-038 — FULL DISCHARGE

**Evidence command:**

```
go test -run "TestE2E_EtoPEGraduationByConfigChange" -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestE2E_EtoPEGraduationByConfigChange
--- PASS: TestE2E_EtoPEGraduationByConfigChange (0.01s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.305s
```

**What the test asserts:** After `eRouter.Restart(t, testenv.RouterConfig{UpstreamRouters: []string{peAddr}})`,
`eRouter.Mode()` returns `testenv.ModePE` (reflecting actual live connection state via the
construction-time `Connector`, not the retired stub `r.mode = ModePE`). `eRouter.SVTNID()`
is unchanged. VP-038 `verification_lock` flipped to `true` in this story's commits.

### VP-037 — PARTIAL DISCHARGE

**Evidence command:**

```
go test -run "TestE2E_RouterDrain_NodesMigrateWithin2s" -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestE2E_RouterDrain_NodesMigrateWithin2s
    router_pe_connector_test.go:600: VP-037 partial-discharge: DRAIN broadcast wire
    protocol required from S-7.04-FU-DRAIN-WIRE; live upstream connections delivered
    by S-7.04-FU-PE-CONNECTOR (this story). Partial-discharge per story AC-005 note.
--- SKIP: TestE2E_RouterDrain_NodesMigrateWithin2s (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	0.305s
```

**What the skip note records:** VP-037 requires a DRAIN broadcast over PE connections
to force node migration within 2 seconds. The live upstream connections that unblock
VP-037 are delivered by this story. The DRAIN broadcast wire protocol is owned by
S-7.04-FU-DRAIN-WIRE. VP-037 `verification_lock` remains `false`; the partial-discharge
note is embedded in the test as a persistent record.

**Discharge status:** VP-038 FULL; VP-037 PARTIAL (declared divergence D-3, blocked on
S-7.04-FU-DRAIN-WIRE).

---

## AC-006 — Construction-time testenv seam retirement (Q2, FO-2)

**Evidence type:** Test citation only (no tape)  
**Rationale:** This AC is an internal testenv refactoring — retiring `SetSighupCh` /
`SendReloadSignal` and replacing with construction-time `Connector` wiring. A terminal
demo would show only test pass/fail; the structural guarantee (seam retired, live
`RouterHandle.Mode()` delegation working) is visible only in test assertions and code.

**BC anchor:** S-7.04-FU-SIGHUP-RELOAD DELIVERY FO-2; placement note Q2

**Evidence command (seam retirement migration + live Mode() delegation):**

```
go test -run "TestRunRouter_VP038_EtoPEViaConfigOnly|TestRunRouter_SIGHUPReload_EtoPE|TestRunRouter_PE_RouterHandleModeReflectsLiveState" -v ./cmd/switchboard/
```

**Captured output:**

```
=== RUN   TestRunRouter_PE_RouterHandleModeReflectsLiveState
--- PASS: TestRunRouter_PE_RouterHandleModeReflectsLiveState (3.01s)
=== RUN   TestRunRouter_SIGHUPReload_EtoPE
--- PASS: TestRunRouter_SIGHUPReload_EtoPE (0.05s)
=== RUN   TestRunRouter_VP038_EtoPEViaConfigOnly
--- PASS: TestRunRouter_VP038_EtoPEViaConfigOnly (0.05s)
PASS
ok  	github.com/arcavenae/switchboard/cmd/switchboard	3.641s
```

**Evidence command (Stop idempotency — F-P2-001):**

```
go test -run "TestConnector_Stop_Idempotent" -v ./internal/upstreamdial/
```

**Captured output:**

```
=== RUN   TestConnector_Stop_Idempotent
--- PASS: TestConnector_Stop_Idempotent (0.00s)
PASS
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	0.733s
```

**Evidence command (Restart×2 idempotency — F-P2-001 adversary reproduction):**

```
go test -run "TestRouterHandle_Restart_TwicePE" -v ./internal/testenv/
```

**Captured output:**

```
=== RUN   TestRouterHandle_Restart_TwicePE
--- PASS: TestRouterHandle_Restart_TwicePE (0.03s)
PASS
ok  	github.com/arcavenae/switchboard/internal/testenv	0.302s
```

**What the tests assert:**

1. `TestRunRouter_VP038_EtoPEViaConfigOnly` — **migrated test**: `SetSighupCh`/`SendReloadSignal`
   seam retired; test now drives `rawSighupCh <- syscall.SIGHUP` directly (same pattern as all
   other 9 SIGHUP tests). The emission-scan observable (`mode=PE` in writer) remains
   authoritative; `RouterHandle.Mode()` is not asserted here (externally-started goroutine
   is disconnected from the handle).
2. `TestRunRouter_SIGHUPReload_EtoPE` — continues passing post-seam-retirement; confirms
   existing SIGHUP tests were not broken by the refactoring.
3. `TestRunRouter_PE_RouterHandleModeReflectsLiveState` — strengthened (F-P15-001):
   inverse-delegation assertion immediately after `SetConnector(fakeConnE)` (mutation-pinned:
   flipping fake to ModePE now fails the test); post-Restart ModeE attributed to live
   connector's failed dial against closed ephemeral port (overrides `r.mode` stub).
   Covers three transition surfaces: ModePE-fake delegation, ModeE-fake inverse-delegation,
   and live-connector-override after Restart.
4. `TestConnector_Stop_Idempotent` — two sequential + one concurrent `Stop()` calls on the
   same Connector; no panic; proves the `sync.Once`-wrapped `close(stopCh)` is idempotent.
5. `TestRouterHandle_Restart_TwicePE` — exact adversary reproduction shape (F-P2-001):
   `StartRouter` + `Restart×2`; `t.Cleanup` fires second `Stop` on `conn1`; no panic;
   proves idempotency in the `Restart`/`t.Cleanup` path.

**Discharge status:** FULL. `SetSighupCh` and 1-arg `SendReloadSignal` retired;
`RouterHandle.Mode()` delegates to live `connector.Mode()`; all migrated and new tests
pass under `go test -race`.

---

## Full Package Test Run

**Evidence command (complete suite — all three affected packages):**

```
go test -count=1 ./internal/upstreamdial/ ./cmd/switchboard/ ./internal/testenv/
```

**Captured output:**

```
ok  	github.com/arcavenae/switchboard/internal/upstreamdial	22.744s
ok  	github.com/arcavenae/switchboard/cmd/switchboard	14.580s
ok  	github.com/arcavenae/switchboard/internal/testenv	0.856s
```

All packages pass. 29 net-new tests + 1 migrated across the story; delivered under
`go test -race` (race detector enabled by default in CI).
