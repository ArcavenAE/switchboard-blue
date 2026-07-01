# Demo Evidence Report — S-W5.02

**Story:** S-W5.02 — E2E Management Plane Integration Harness Across All Four Daemon Types
**VP:** VP-049 (mgmt-plane e2e across all four daemon types)
**Impl SHA:** 07ce3db on branch `feat/S-W5.02-mgmt-e2e-harness`
**Evidence captured:** 2026-06-30
**Evidence format:** Integration test terminal capture (headless test harness — VHS not applicable per demo-recorder policy)

---

## Coverage Summary

| AC | Description | Test Function(s) | Result |
|----|-------------|-----------------|--------|
| AC-001 | Four mgmt.Server instances started, one per daemon type, each on a unique Unix socket | `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/{router,access,console,control}` | PASS |
| AC-002 (primary) | Distinct-operator-key auth: daemon key ≠ operator key; `Authenticate()` returns nil for all four sockets | `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/{router,access,console,control}` | PASS |
| AC-002 (bootstrap) | Bootstrap-mode auth: `NewOperatorKeySet(nil)`, authenticate with daemon key | `TestE2E_MgmtPlane_BootstrapAuth_VP049` | PASS |
| AC-003 | Post-auth RPC dispatch: non-constant request ID, `resp.Type=="response"`, `resp.ID==req.ID`, `resp.Ok==true`, `resp.Data` present | `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/{router,access,console,control}` | PASS |
| AC-004 | Unauthenticated RPC → AUTH_FAIL + connection close (router daemon, representative) | `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004` | PASS |
| AC-005 | Client-side FIN observed server-side within 500ms of RPC completion (`closingListenerWrapper`) | `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/{router,access,console,control}` | PASS |

---

## Artifact Index

| File | Description |
|------|-------------|
| `just-test-race-summary.txt` | `go test -race ./...` — all 17 packages PASS, 0 failures |
| `integration-test-output.txt` | `go test -race -tags integration ./cmd/sbctl/... -v -run TestE2E` — 3 top-level tests + 4 sub-tests PASS |
| `evidence-report.md` | This file |

---

## AC-001: Four daemon servers started, sockets ready

**Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` — setup step

**Mechanism:** Each sub-test creates a unique Unix socket via `t.TempDir()`, starts a `mgmt.NewServer` with a distinct per-mode handler table (`routerHandlers()`, `accessHandlers()`, `consoleHandlers()`, `controlHandlers()`), and polls with 1s timeout until the socket is ready. The four sub-tests run in parallel (`t.Parallel()`).

**PASS lines from integration-test-output.txt:**
```
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/router (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/access (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/console (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/control (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049 (0.01s)
```

---

## AC-002 (primary): Distinct-operator-key authentication

**Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` sub-tests per daemon

**Mechanism:** Generates an Ed25519 daemon key pair and a separate Ed25519 operator key pair in-process. Constructs `mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})` so the daemon only authorizes the operator key (not its own key). Calls `client.Authenticate(conn, operatorPriv)` and verifies it returns `nil` for each of the four daemon sockets. This is the primary VP-049 coverage path (Q5 ruling — Option B).

**PASS lines from integration-test-output.txt:**
```
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/router (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/access (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/console (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/control (0.01s)
```

---

## AC-002 (bootstrap): Bootstrap-mode authentication

**Test:** `TestE2E_MgmtPlane_BootstrapAuth_VP049`

**Mechanism:** Uses `mgmt.NewOperatorKeySet(nil)` (daemon key is sole authorized key) and authenticates with the daemon's own private key. Verifies `Authenticate()` returns `nil`. Covers the ADR-012 §bootstrap production path.

**PASS line from integration-test-output.txt:**
```
--- PASS: TestE2E_MgmtPlane_BootstrapAuth_VP049 (0.01s)
```

---

## AC-003: RPC dispatch after authentication (Rulings M/U/X)

**Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` sub-test per daemon

**Mechanism:** After successful authentication, sends a `"status"` RPC request with:
- `"type":"request"` (Ruling M)
- Non-constant per-call request ID (hex-encoded `time.Now().UnixNano()`) (Ruling X)

Asserts all four conditions:
1. `resp.Type == "response"` (Ruling U — wrong-type with `"ok":true` must not be silently accepted)
2. `resp.ID == req.ID` (Ruling X — ID echo; mismatch returns E-RPC-001)
3. `resp.Ok == true`
4. `resp.Data` field present (non-nil, even if empty/minimal)

**PASS lines from integration-test-output.txt:**
```
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/router (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/access (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/console (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/control (0.01s)
```

---

## AC-004: Unauthenticated RPC rejected

**Test:** `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004`

**Mechanism:** Against the router daemon (representative of all four), sends a `{"type":"request",...}` directly without completing the ADR-012 handshake. Verifies the server returns AUTH_FAIL and closes the connection.

**PASS line from integration-test-output.txt:**
```
--- PASS: TestE2E_MgmtPlane_UnauthenticatedRejected_AC004 (0.01s)
```

---

## AC-005: Client-side FIN observed server-side within 500ms

**Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` sub-test per daemon

**Mechanism:** The `net.Listener` passed to each `mgmt.NewServer` is wrapped in a `closingListenerWrapper` (implemented in `cmd/sbctl/e2e_helpers_test.go`). The wrapper tracks when each accepted `net.Conn` has its remote-side closed by counting `Read` returning `io.EOF` after dispatch returns. After each per-daemon sub-test completes the RPC cycle, the test asserts `wrapper.ClientClosedWithin(500ms)`. This instruments the actual production `defer conn.Close()` in `cmd/sbctl/client.go:connectAndRun`, not a tautological local-side close check (Q6 ruling — Option A).

**PASS lines from integration-test-output.txt:**
```
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/router (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/access (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/console (0.01s)
--- PASS: TestE2E_MgmtPlane_AllFourDaemonTypes_VP049/control (0.01s)
```

---

## Full Test Suite Health

`go test -race ./...` — 17 packages, all PASS, 0 failures.

```
ok  	github.com/arcavenae/switchboard/cmd/sbctl	2.827s
ok  	github.com/arcavenae/switchboard/cmd/switchboard	1.834s
ok  	github.com/arcavenae/switchboard/internal/admission	11.802s
ok  	github.com/arcavenae/switchboard/internal/arq	2.119s
ok  	github.com/arcavenae/switchboard/internal/config	2.712s
ok  	github.com/arcavenae/switchboard/internal/frame	2.640s
ok  	github.com/arcavenae/switchboard/internal/halfchannel	2.873s
ok  	github.com/arcavenae/switchboard/internal/hmac	3.101s
ok  	github.com/arcavenae/switchboard/internal/metrics	2.673s
ok  	github.com/arcavenae/switchboard/internal/mgmt	5.001s
ok  	github.com/arcavenae/switchboard/internal/multipath	1.273s
ok  	github.com/arcavenae/switchboard/internal/paths	1.649s
ok  	github.com/arcavenae/switchboard/internal/replay	1.680s
ok  	github.com/arcavenae/switchboard/internal/routing	1.981s
ok  	github.com/arcavenae/switchboard/internal/session	1.752s
ok  	github.com/arcavenae/switchboard/internal/svtnmgmt	1.470s
ok  	github.com/arcavenae/switchboard/internal/tmux	1.620s
```

Total: 607 passing tests (unit + integration), 0 failures, race detector clean.
