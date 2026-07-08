# Demo Evidence Report — S-7.04-FU-SIGHUP-RELOAD

**Story:** SIGHUP config-reload path in router daemon signal loop — runtime
E-to-PE graduation without restart  
**Story version:** v1.7  
**Branch:** story/s-7.04-fu-sighup-reload  
**Evidence date:** 2026-07-07

---

## POL-004 Note

Per `docs/DEMO-EVIDENCE-POLICY.md` (ratified 2026-07-04), rendered binaries
(`.gif`, `.webm`, `.mp4`, `.png`) are **not committed**. `.tape` scripts and
this evidence report are the source of truth. To regenerate locally:

```bash
vhs docs/demo-evidence/S-7.04-FU-SIGHUP-RELOAD/AC-001-sighup-e-to-pe.tape
vhs docs/demo-evidence/S-7.04-FU-SIGHUP-RELOAD/AC-002-fail-closed-reload.tape
```

---

## Acceptance Criteria Evidence

| AC | Evidence type | What it shows | Test name(s) |
|----|--------------|---------------|--------------|
| AC-001 | Tape script + test citation | Router starts in E mode; config rewritten with `upstream_routers`; SIGHUP causes `mode=PE upstream_routers=[...]` emission without daemon restart (PID unchanged) | `TestRunRouter_SIGHUPReload_EtoPE` |
| AC-002 | Tape script + test citation | Valid start; config overwritten with invalid YAML (empty `listen_addr`); SIGHUP produces verbatim EC-004 line `config reload failed: E-CFG-001: ...; continuing with previous config`; no second `mode=` line; daemon still responds to a subsequent valid SIGHUP | `TestRunRouter_SIGHUPReload_BadConfig_FailClosed` |
| AC-003 | Test citation (no tape) | Active TCP connection to ingress listener survives a valid SIGHUP reload; `ingressCtx`/`ingressCancel`, `dataWG`, `drainCoord`, `mgmtSrv`/`mgmtWG`, and parent `ctx` are untouched by the reload path | `TestRunRouter_SIGHUPReload_SessionsNotInterrupted` |
| AC-004 | Test citation (no tape) | `testenv.RouterHandle.SendReloadSignal(t)` drives the in-process `sighupCh` seam; `mode=PE` line appears in the real `runRouter` writer output; goroutine has not returned (VP-038 activation without process restart) | `TestRunRouter_VP038_EtoPEViaConfigOnly` |

---

## AC-001 — SIGHUP capture and conditional mode re-emission

**Tape:** `AC-001-sighup-e-to-pe.tape`  
**BC anchors:** BC-2.09.001 PC-1, EC-002  
**Test:** `TestRunRouter_SIGHUPReload_EtoPE` (`cmd/switchboard/router_sighup_test.go`)

The tape demonstrates the operator flow end-to-end:

1. Build the `switchboard` binary (`go build -ldflags "-X main.version=demo"`).
2. Write an E-mode config (no `upstream_routers`), start router in background.
3. Startup emits `switchboard router: mode=E (no upstream_routers configured)`.
4. Overwrite config with a PE-capable version (adds `upstream_routers: [{addr: ...}]`).
5. Send `SIGHUP` to the running PID.
6. Router emits `switchboard router: mode=PE upstream_routers=[127.0.0.1:<port>]`.
7. `kill -0 $PID` confirms the process is still alive — no restart occurred.

**Key output lines verified (manual bash execution):**

```
switchboard router: mode=E (no upstream_routers configured)
```
→ (after SIGHUP with upstream_routers added to config)
```
switchboard router: mode=PE upstream_routers=[127.0.0.1:29802]
```

---

## AC-002 — Fail-closed reload on bad config (BC-2.09.003 EC-004)

**Tape:** `AC-002-fail-closed-reload.tape`  
**BC anchors:** BC-2.09.003 Inv-3, EC-004  
**Test:** `TestRunRouter_SIGHUPReload_BadConfig_FailClosed` (`cmd/switchboard/router_sighup_test.go`)

The tape demonstrates three beats:

1. Start router with a valid config (E mode startup, `mode=E` emitted).
2. Overwrite config with invalid YAML (omits `listen_addr` — triggers E-CFG-001);
   send SIGHUP. Daemon emits the EC-004 line verbatim and continues on the
   previous config (`mode=` line count remains 1).
3. Restore a valid PE config; send a second SIGHUP. Daemon recovers and emits
   `mode=PE` — proving it remained alive and continued executing the signal loop
   throughout the failed reload.

**Key output lines verified (manual bash execution):**

```
switchboard router: mode=E (no upstream_routers configured)
```
→ (after SIGHUP with invalid config)
```
config reload failed: E-CFG-001: config error: listen_addr: required field missing. Fix: add 'listen_addr: <ip>:<port>' to config, e.g. 'listen_addr: 0.0.0.0:9090'; continuing with previous config
```
→ (after second SIGHUP with valid PE config)
```
switchboard router: mode=PE upstream_routers=[127.0.0.1:29804]
```

---

## AC-003 — Active session non-interruption during valid reload

**Evidence type:** Test citation only (no tape)  
**Rationale:** This AC is an integration-level invariant over internal goroutine
state (`ingressCtx`, `dataWG`, `drainCoord`, `mgmtSrv`/`mgmtWG`, parent `ctx`).
A terminal demo would show only that a TCP connection stays open — the underlying
structural guarantee (none of the five constructs are touched by the reload path)
is visible in test assertions, not in a shell session. A tape would be contrived.

**Test:** `TestRunRouter_SIGHUPReload_SessionsNotInterrupted`  
**File:** `cmd/switchboard/router_sighup_test.go`  
**BC anchor:** BC-2.09.001 PC-4

**What the test asserts:**

1. A live TCP connection is dialed to the ingress listener before SIGHUP fires.
2. A valid PE config is sent via `sighupCh`; `mode=PE` line is confirmed in
   the writer output.
3. After reload: `ingressConn.Read` with a short deadline returns a timeout
   error (not EOF or connection-reset) — the connection was not closed by the
   daemon.
4. The management socket is still accepting: `dialMgmtAndReadChallenge` returns
   a well-formed `{"type":"challenge"}` response.
5. `runRouter` has not returned.

**Test result:** PASS (verified under `go test -race`).

---

## AC-004 — VP-038 activation: E→PE via SIGHUP without process restart

**Evidence type:** Test citation only (no tape)  
**Rationale:** AC-004 exercises the `testenv.RouterHandle` seam
(`SetSighupCh` + `SendReloadSignal`) — an in-process test rig that feeds the
`sighupCh` channel directly. A terminal demo cannot show the testenv-internal
wiring meaningfully; the integration test is the authoritative evidence for
VP-038 activation.

**Test:** `TestRunRouter_VP038_EtoPEViaConfigOnly`  
**File:** `cmd/switchboard/router_sighup_test.go`  
**VP anchor:** VP-038

**What the test asserts:**

1. `testenv.New(t, ctx)` + `env.StartRouter(t, ...)` constructs a `RouterHandle`.
2. `handle.SetSighupCh(rawSighupCh)` wires the real `runRouter` sighupCh into
   the handle (post-hoc setter, transitional seam — construction-time wiring
   deferred to PE-CONNECTOR era per AC-004 PC-2 deferral note).
3. `handle.SendReloadSignal(t)` sends `syscall.SIGHUP` on the in-process channel
   (no `syscall.Kill` / real OS signal — per AC-004 postcondition 3).
4. `mode=PE` line appears in the real `runRouter` writer output (the authoritative
   observable per v1.2 — `RouterHandle.Mode()` assertion deferred to
   testenv/PE-CONNECTOR era per adversary pass-1 F-002 ruling).
5. `runRouter` goroutine has not returned.

**Test result:** PASS (verified under `go test -race`).

**Note on `RouterHandle.Mode()` deferral:** The test does not assert
`handle.Mode() == testenv.ModePE` directly. The `handle.Restart(t, cfg)` path
unconditionally sets `r.mode = ModePE` regardless of the real goroutine state,
making that assertion tautological. The emission-based observable (`mode=PE` in
the writer output) is the non-tautological proof that the signal handler fired.
`RouterHandle.Mode()` state-tracking is deferred to the PE-CONNECTOR era when
the seam can track real router state.

---

## All Ten Tests in router_sighup_test.go

For reference, the full test inventory for the sighup test file:

| Test name | AC / source | Description |
|-----------|------------|-------------|
| `TestRunRouter_SIGHUPReload_EtoPE` | AC-001 | E→PE mode transition via SIGHUP |
| `TestRunRouter_SIGHUPReload_BadConfig_FailClosed` | AC-002 | Fail-closed on bad config (E-CFG-001) |
| `TestRunRouter_SIGHUPReload_SessionsNotInterrupted` | AC-003 | Session continuity during valid reload |
| `TestRunRouter_VP038_EtoPEViaConfigOnly` | AC-004 | VP-038 via testenv seam |
| `TestRunRouter_SIGHUPReload_LoadFileNotFound` | P1 F-005 | Fail-closed on missing file (E-CFG-004) |
| `TestRunRouter_SIGHUPReload_MalformedYAML` | P2 F-001 | Fail-closed on malformed YAML (E-CFG-005) |
| `TestRunRouter_SIGHUPReload_PEtoE` | P3 F-005a | PE→E downgrade via SIGHUP |
| `TestRunRouter_SIGHUPReload_PEtoPE` | P3 F-005a | PE→PE idempotent reload |
| `TestRunRouter_SIGHUPReload_IdempotentResend` | P3 F-005a | No extra emission when upstream list unchanged |
| `TestRunRouter_SIGHUPReload_InvalidUpstreamAddr_FailClosed` | P8 F-P8-002 | Fail-closed on invalid upstream addr |

All ten pass under `go test -race -count=1 ./cmd/switchboard/`.

---

## Supplemental: main_test.go real-OS-signal guard

**Test:** `TestRunRouterRun_RealSIGHUP_DoesNotExit`  
**File:** `cmd/switchboard/main_test.go`  
**Source:** Adversary pass-5 F-SIGHUP-P5-001 remediation

Verifies that a real OS-level SIGHUP sent through `run()` does not cause the
daemon to exit — confirming the dedicated `sighupCh` (via `signal.Notify`, not
`signal.NotifyContext`) is correctly isolated from the SIGTERM/SIGINT
cancellation path. This guards the architect's Q1 decision to use a dedicated
channel rather than a shared cancellable context.
