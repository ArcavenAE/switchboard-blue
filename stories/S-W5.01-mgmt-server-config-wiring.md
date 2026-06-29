---
artifact_id: S-W5.01-mgmt-server-config-wiring
document_type: story
level: ops
story_id: S-W5.01
title: "implement internal/mgmt server, config additions (E-CFG-008/009), and cmd/switchboard wiring for all four daemon modes"
status: draft
producer: story-writer
timestamp: 2026-06-28T00:00:00
phase: 2
epic: E-6
wave: 5
priority: P0
scope_phase: E
estimated_points: 8
version: "1.6"
bc_traces:
  - BC-2.07.004
  - BC-2.09.003
vp_traces: [VP-064, VP-065, VP-066, VP-068, VP-069, VP-070, VP-071, VP-072, VP-073]
subsystems: [network-management, deployment-operations]
architecture_modules: [internal/mgmt, internal/config, cmd/switchboard]
tdd_mode: strict
target_module: internal/mgmt
cycle: v1.0.0-greenfield
depends_on: [S-6.01]
blocks: [S-W5.02]
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.004.md'
  - '.factory/specs/behavioral-contracts/ss-09/BC-2.09.003.md'
  - '.factory/specs/architecture/ARCH-12-daemon-management-plane.md'
  - '.factory/specs/architecture/ARCH-01-system-overview.md'
  - '.factory/specs/architecture/ARCH-05-cli-and-api.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
  - '.factory/specs/verification-properties/VP-064.md'
  - '.factory/specs/verification-properties/VP-065.md'
  - '.factory/specs/verification-properties/VP-066.md'
  - '.factory/specs/verification-properties/VP-068.md'
  - '.factory/specs/verification-properties/VP-069.md'
  - '.factory/specs/verification-properties/VP-070.md'
  - '.factory/specs/verification-properties/VP-071.md'
  - '.factory/specs/verification-properties/VP-072.md'
  - '.factory/specs/verification-properties/VP-073.md'
acceptance_criteria_count: 20
# BC status: active — all BCs are final and assigned
---

# S-W5.01: internal/mgmt Server, Config Additions, and cmd/switchboard Wiring

> **Execute:** `/vsdd-factory:deliver-story S-W5.01`

## Scope Note

This story implements the **server side** of the ADR-012 management plane:

1. `internal/mgmt` package: `Server`, `NewServer`, `OperatorKeySet`, `Serve`,
   `Shutdown`, `MaxMessageBytes`, `HandshakeTimeout`, `RPCIdleTimeout`,
   `MaxConcurrentConnections`, challenge generation, ADR-012 auth handshake,
   bounded reads, per-connection read deadlines, connection-cap semaphore,
   handler dispatch, JSON envelope wrapping.
2. `internal/config` additions: `ManagementSocket` and `AuthorizedOperatorKeys`
   fields; `Validate()` extensions for E-CFG-008 and E-CFG-009.
3. `cmd/switchboard` wiring: start the management listener in the **access daemon**
   (the only currently non-stub mode); stub the mgmt goroutine wire-up for router,
   console, and control modes so those modes do NOT start a live mgmt listener or
   orphaned goroutine. WaitGroup-track the mgmt goroutine per ARCH-01.

**Scope clarification (ARCH-12 v1.3 Ruling A — resolves Round-1 contradiction H-2):**
The access daemon (`runAccess`) is the ONLY mode wired this wave. Router, console, and
control daemons are stubs: they MUST NOT start a `mgmt.Server` or leave an orphaned
listener/goroutine. Stub wiring (e.g., a `TODO` or no-op) is correct for those three
modes. The prior v1.1 Scope Note contained a self-contradiction — it simultaneously
said "all four modes MUST start mgmt.Server" AND "the access daemon is the only
currently non-stub mode." The architect's intent (ARCH-12 §Story-Writer Handoff v1.3)
is access-only scope this wave; the absolute "all four modes" clause is dropped.

**Server uses stdlib `crypto/ed25519` only — no `golang.org/x/crypto` dependency.
`golang.org/x/crypto` is used only by S-6.03 (`cmd/sbctl`) for OpenSSH PEM
parsing.**

The sbctl client (S-6.03) does NOT need to be merged before this story can begin
development — the wire protocol is fully specified in ADR-012. Both can develop
in parallel on separate branches; integration requires both (S-W5.02).

## Behavioral Contracts

| BC | Title | PCs covered |
|----|-------|------------|
| BC-2.07.004 | Daemon Management Server Authenticates All Connections via Ed25519 Challenge-Response (Fail-Closed) | PC-1 (challenge issued + write deadlines), PC-2 (unauth rejected), PC-3 (replay/post-auth guard + ephemeral keypair precondition), PC-4 (auth fail closes connection), PC-5 (all RPCs require auth), PC-6 (bounded reads CWE-400), PC-7 (AUTH_OK), PC-8 (constant-time comparison), PC-9 (bootstrap mode), PC-10 (Serve nil on shutdown), PC-11 (E-RPC-010 unknown command), PC-12 (E-RPC-011 handler error), Invariant 8 (nil-key construction guard) |
| BC-2.09.003 | Router Startup Fails Cleanly on Malformed Config (v1.6) | PC-10 (management_socket validation: E-CFG-008), PC-11 (authorized_operator_keys PEM validation: E-CFG-009) |

## Narrative

- **As a** Switchboard daemon (router, access, console, or control mode)
- **I want to** start an Ed25519-authenticated management server on my management
  socket before accepting data-plane connections
- **So that** `sbctl` operators can authenticate and issue management RPCs
  securely without any unauthenticated access path

## Acceptance Criteria

### AC-001 (traces to BC-2.07.004 postcondition 1 — challenge issued immediately + HandshakeTimeout deadline)
On every new connection, `mgmt.Server` sends a CHALLENGE message as the **first**
action before reading any client data:
`{"type":"challenge","nonce":"<base64url 32 bytes>","daemon_sig":"<base64url sig>"}`.
The nonce is 32 bytes from `crypto/rand.Read`. The `daemon_sig` is
`ed25519.Sign(daemonPrivKey, nonceBytes)`. After sending the CHALLENGE, the server
calls `conn.SetReadDeadline(time.Now().Add(mgmt.HandshakeTimeout))` (default 10 s)
before the subsequent blocking read. On HandshakeTimeout expiry (silent stall) the
connection is closed **without sending AUTH_FAIL** — a non-responsive client would
not read it (BC-2.07.004 EC-001 v1.4 / Ruling K). Non-timeout decode errors
(malformed JSON, EOF before timeout, oversized message) DO send AUTH_FAIL before
close. (Traces to BC-2.07.004 PC-1 and EC-001.)
- **Test:** `TestMgmtServer_IssuesChallengeFirst_AC001` — connect via `net.Pipe`;
  verify the first message received from the server has `"type":"challenge"`,
  a non-empty `"nonce"` (32 bytes when decoded), and a non-empty `"daemon_sig"`.
  Use a server constructed with a 50 ms `HandshakeTimeout` override; connect and
  send nothing; verify the server closes the connection within ~100 ms of
  `HandshakeTimeout` expiry. **Assert that NO `AUTH_FAIL` JSON message was
  received on the client pipe before the close — a timeout close is silent
  (close-only, no AUTH_FAIL per Ruling K / BC-2.07.004 EC-001 v1.4).** Non-timeout
  decode errors (malformed JSON, EOF before timeout) DO receive AUTH_FAIL before
  close. Verify no goroutine leak after close.

### AC-002 (traces to BC-2.07.004 postcondition 2 — unauthenticated connections rejected, VP-064)
A connection that sends a CHALLENGE_RESPONSE with either (a) an unrecognized
public key or (b) a signature that does not verify against the presented public
key receives `{"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}`
and the connection is immediately closed. No RPC handler is called.
- **Test:** `TestMgmtServer_RejectsUnauthenticated_VP064` — table-driven, four
  sub-cases: (a) no CHALLENGE_RESPONSE at all (VP-064 property "no CHALLENGE_RESPONSE
  at all" — server applies HandshakeTimeout; verify connection closed after timeout);
  (b) unrecognized public key → AUTH_FAIL + close; (c) recognized key with wrong
  signature → AUTH_FAIL + close; (d) client connects, receives CHALLENGE, then waits
  without responding — verify server closes connection after `HandshakeTimeout` with no
  goroutine leak (traces to BC-2.07.004 EC-001 / Ruling 1).
  Verify AUTH_FAIL response and no RPC dispatch for sub-cases (b) and (c).

### AC-003 (traces to BC-2.07.004 postcondition 3 — post-auth structural guard, VP-065)
A connection that has completed a successful auth handshake (is in the authenticated
state) and then sends a second `{"type":"challenge_response",...}` message receives
`{"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}` and the
connection is closed. The server tracks authentication state via a **per-connection
`authenticated` boolean**; no nonce set is stored after auth completes. Cross-connection
replay is prevented by the fresh nonce issued on every new connection
(`crypto/rand.Read(32)`) — there is no shared nonce state across connections.
(Ruling 7 — ARCH-12 v1.2: replaces the prior per-connection nonce-set framing.)
- **Test:** `TestMgmtServer_PostAuthChallengeResponseRejected_VP065` — authenticate
  successfully on connection C1 (verify AUTH_OK received); then send a second
  `{"type":"challenge_response",...}` message on the same C1; verify the server
  responds with AUTH_FAIL (E-ADM-010) and closes the connection. Assert that the
  `authenticated` boolean / state variable caused the rejection, not a nonce-set
  lookup. Verify no goroutine leak.
The BC-2.07.004 PC-3 / EC-004 requirement to emit a security-event log is explicitly DEFERRED to the daemon logging story S-HRD.02 (daemon logging infrastructure / slog seam on mgmt.Server); this AC covers only the fail-closed connection control (AUTH_FAIL + E-ADM-010 + close), which is fully implemented and tested via VP-065.

### AC-004 (traces to BC-2.07.004 postcondition 4 — auth fail closes connection without RPCs)
After sending AUTH_FAIL, the server closes the connection. No retry is possible
on the same connection. No RPC response is ever sent on an unauthenticated
connection.
- **Test:** `TestMgmtServer_AuthFailClosesConnection_AC004` — after receiving
  AUTH_FAIL, a subsequent read from the client-side `net.Pipe` returns an error
  (connection closed). Verify `rpcCallCount == 0`.

### AC-005 (traces to BC-2.07.004 postcondition 5 — all RPCs require auth)
A client that skips the handshake and sends a `{"type":"request",...}` directly
receives AUTH_FAIL + close. No RPC handler is invoked.
- **Test:** `TestMgmtServer_RPCWithoutAuth_Rejected_AC005` — send a `"type":"request"`
  message immediately after connecting (before any CHALLENGE_RESPONSE). Verify
  AUTH_FAIL returned and `rpcCallCount == 0`.

### AC-006 (traces to BC-2.07.004 postcondition 6 — bounded reads CWE-400, VP-066)
Every `json.Decoder.Decode` call on the management socket is preceded by
`io.LimitReader(conn, MaxMessageBytes)` (64 KiB). A message exceeding
`MaxMessageBytes` causes the decode to return an error and the connection to
close. The process does not OOM.
- **Test:** `TestMgmtServer_BoundedRead_VP066` — unit test and fuzz target per
  VP-066.md proof harness: (a) message of `MaxMessageBytes-1` bytes: accepted;
  (b) message of `MaxMessageBytes+1` bytes: connection closed, no OOM. Fuzz target:
  `FuzzMgmtServer_BoundedRead_VP066`.

### AC-007 (traces to BC-2.07.004 postcondition 7 — successful authentication + daemon_version, Ruling 6; updated Ruling C: E-RPC-010/E-RPC-011)
A client presenting a valid authorized key with a correct signature for the
challenge nonce receives `{"type":"auth_ok","daemon_version":"<semver>"}`.
The `daemon_version` field MUST equal the `daemonVersion` string passed to `NewServer`
(injected from `cmd/switchboard.version` via ldflags; `"dev"` is the unreleased-build
sentinel only — hardcoding `"dev"` in production is a defect). `NewServer` MUST accept
the `daemonVersion string` as its fifth parameter and MUST panic or return an error if
`daemonVersion` is an empty string.
Subsequent `{"type":"request","id":"<id>","command":"<cmd>","args":{}}` messages
are dispatched to the registered handler and receive a response wrapped in the
JSON envelope from `interface-definitions.md §JSON Output Schema`.
For unregistered commands, the server responds with
`{"ok":false,"error":{"code":"E-RPC-010","message":"unknown command: <cmd>"},"data":null}`
in-band (connection NOT closed). For handler errors, the server responds with
`{"ok":false,"error":{"code":"E-RPC-011","message":"<err>"},"data":null}` in-band
(connection NOT closed). The undefined `E-RPC-002` MUST NOT appear anywhere in
`internal/mgmt` — any such reference is a defect. `E-RPC-001` is the CLIENT-SIDE
sbctl error code and MUST NOT appear in server dispatch responses.
- **Test:** `TestMgmtServer_AuthOK_DispatchesRPC_AC007` — pass `"0.1.0-test"` as
  `daemonVersion` to `NewServer`; authenticate with an authorized key; verify AUTH_OK
  received and `auth_ok.daemon_version == "0.1.0-test"`. Send a test RPC; verify the
  response envelope has `"ok":true` and the handler's return value in `"data"`.
  Also verify that passing `daemonVersion = ""` to `NewServer` causes a panic or
  initialization error (document the chosen enforcement in the story tasks).
  Table rows for RPC error paths (Ruling C):
  - (unknown command) send authenticated RPC `{"command":"not.registered","id":"r2","args":{}}`;
    expect `{"ok":false,"error":{"code":"E-RPC-010","message":"unknown command: not.registered"},"data":null}`;
    verify connection remains open and accepts a subsequent RPC.
  - (handler error) register a handler that returns `errors.New("boom")`; send that RPC;
    expect `{"ok":false,"error":{"code":"E-RPC-011","message":"boom"},"data":null}`;
    verify connection remains open.
  (traces to BC-2.07.004 PC-11 and PC-12)

### AC-008 (traces to BC-2.07.004 postcondition 8 — constant-time comparison, no oracle)
`OperatorKeySet.IsAuthorized` uses `subtle.ConstantTimeCompare` (or equivalent)
so that AUTH_FAIL responses for unrecognized keys and recognized-key-wrong-signature
keys are behaviorally identical (same message, no timing side channel).
- **Test:** `TestOperatorKeySet_ConstantTimeCompare_AC008` — verify that
  `IsAuthorized` returns false for an unrecognized key and false for a key with
  a one-byte mutation; both use the same code path. (Timing-oracle property
  is verified by code inspection + review, not a Go unit test — document this
  in the test.)

### AC-009 (traces to BC-2.07.004 postcondition 9 — bootstrap mode)
When `authorized_operator_keys` is empty (nil or zero-length), `OperatorKeySet`
authorizes connections signed by the daemon's own keypair. The handshake
proceeds identically to the normal case; only the authorized set changes.
- **Test:** `TestMgmtServer_BootstrapMode_DaemonKeyAuthorized_AC009` — construct
  `NewOperatorKeySet(nil)`; sign the challenge nonce with the daemon's own private
  key; verify AUTH_OK.

### AC-010 (traces to BC-2.07.004 postcondition 10 — graceful shutdown)
`Server.Shutdown(ctx)` closes the listener so no new connections are accepted,
then waits for in-flight connections to terminate within the context deadline.
The mgmt goroutine is WaitGroup-tracked (ARCH-01 §Goroutine WaitGroup Contract).
- **Test:** `TestMgmtServer_GracefulShutdown_AC010` — start server, open a
  connection, call `Shutdown` with a short context; verify `Serve` returns and
  the listener is closed; verify no goroutine leak (use `goleak` or
  `t.Cleanup` + `runtime.NumGoroutine` comparison).

### AC-011 (traces to BC-2.09.003 postcondition 10 — management_socket validation, E-CFG-008)
When `management_socket` is present in config and is empty or whitespace-only,
`Validate()` returns `E-CFG-008`:
`"config error: management_socket: must not be empty. Fix: set to a valid Unix socket path..."`.
When absent, `Validate()` accepts without error. Exhaustive error collection:
if other config errors also exist, all are reported together.
- **Test:** `TestConfig_Validate_ManagementSocket_E_CFG_008_AC011` — four
  sub-cases: (a) `management_socket: ""` (empty string = Go zero-value = absent) → **no
  error** (test name: `empty_string_accepted_as_absent`). In Go a `string` yaml field
  cannot distinguish an explicit `""` from an absent field — both unmarshal to zero-value
  `""`; BC-2.09.003 PC-10 and EC-014 specify absent is accepted, so this sub-case
  validates that the empty-string path does NOT produce E-CFG-008. (b)
  `management_socket: "   "` (whitespace-only) → E-CFG-008 error; (c)
  `management_socket` field absent → no error (same code path as sub-case (a));
  (d) `management_socket: "/run/sb.sock"` → no error.

### AC-012 (traces to BC-2.09.003 postcondition 11 — authorized_operator_keys PEM validation, E-CFG-009)
When `authorized_operator_keys` contains entries that are not valid PEM blocks of
type `"PUBLIC KEY"` containing a 32-byte Ed25519 key, each invalid entry is
reported as `E-CFG-009`:
`"config error: authorized_operator_keys[<N>]: entry is not a valid Ed25519 PEM PUBLIC KEY block..."`.
Empty list or absent field is accepted (bootstrap mode). All errors collected
before exit (exhaustive reporting per BC-2.09.003 Inv-4).
- **Test:** `TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012` —
  sub-cases: (a) invalid PEM data at index 0 → E-CFG-009[0]; (b) valid PEM at
  index 0, invalid at index 1 → E-CFG-009[1] only; (c) empty list → no error;
  (d) valid Ed25519 PEM → no error; (e) valid PEM format but wrong key type
  (RSA) → E-CFG-009.

### AC-013 (traces to BC-2.07.004 EC-012 — bounded accept loop, CWE-770, Ruling 3)
`mgmt.Server` does not spawn more than `MaxConcurrentConnections` (default 128)
simultaneous connection goroutines. When the semaphore limit is reached, additional
`Accept()` calls block (back-pressure into the OS accept queue) rather than spawning
new goroutines — no fd exhaustion or goroutine leak. Transient `Accept` errors
(e.g., `EMFILE` — too many open files) trigger exponential backoff starting at 5 ms,
capped at 1 s, and do NOT terminate `Serve`. A non-temporary fatal `Accept` error
causes `Serve` to return.
- **Test:** `TestMgmtServer_ConnectionCap_AC013` — construct a `NewServer` with
  `MaxConcurrentConnections = 3` (via `WithMaxConnections(3)` option or equivalent
  constructor override); open 3 connections via `net.Pipe` that hold the server in
  the handshake phase (send nothing); attempt a 4th `net.Pipe` connection and verify
  it does not immediately receive a CHALLENGE (server is at capacity, the Accept-loop
  semaphore is full). Release one of the 3 held connections; verify the 4th then
  receives a CHALLENGE within a short timeout. Verify goroutine count does not exceed
  limit + 1 (accept loop goroutine).

### AC-014 (traces to BC-2.07.004 EC-013 + Invariant 7 — Unix socket 0600 permissions + console TCP loopback enforcement, CWE-276, Rulings 4 and D; Ruling L canonical message format)
The wiring code in `cmd/switchboard` that calls `net.Listen("unix", ...)` for the
management socket MUST set the process umask to `0177` immediately before the call
and restore it afterward:
```go
old := syscall.Umask(0177) // produces 0600 socket
mgmtLn, err := net.Listen("unix", cfg.ManagementSocket)
syscall.Umask(old)
```
This ensures the socket file is created with permissions `0600` (owner read/write only)
atomically — no TOCTOU window. A `chmod`-after-`Listen` approach MUST NOT be used.

**Console TCP loopback enforcement (Ruling D — ARCH-12 v1.3):** The TCP/console path
in `buildMgmtListener` (`cmd/switchboard/mgmt_wire.go`) MUST validate the address
before calling `net.Listen`. Enforcement is in `buildMgmtListener`, NOT in
`config.Validate()` (which has no mode parameter). For TCP mode (console), the host
component extracted via `net.SplitHostPort` MUST be one of: `127.0.0.1`, `[::1]`, or
`localhost`. Any address whose host is `0.0.0.0`, `::`, empty (bare port `:9091`), or
any non-loopback IP causes `buildMgmtListener` to return an error with code `E-CFG-008`
and message: `"config error: management_socket: console mode requires a loopback
address (127.0.0.1, [::1], or localhost); got: <address>"`. Daemon startup aborts —
this is not a non-fatal warning.
- **Test:** `TestDaemonWiring_UnixSocketPermissions_AC014` — create a temp socket
  path via `t.TempDir()`; call the wiring helper (or a testable extract of the
  `net.Listen("unix", ...)` setup); stat the resulting socket file with `os.Stat`;
  assert `stat.Mode().Perm() == 0600`. Verify the umask is restored to its prior
  value after the call (stat the test process umask or use a helper).
- **Test:** `TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback` — call
  `buildMgmtListener` with mode=console and address `"0.0.0.0:9091"`; verify the
  returned error string contains `"E-CFG-008"`. Repeat with `"::9091"` and `":9091"`
  (bare port); all must reject.
- **Test:** `TestBuildMgmtListener_ConsoleTCP_AcceptsLoopback127` — call with
  `"127.0.0.1:9091"`, `"[::1]:9091"`, `"localhost:9091"`; all must not return an
  E-CFG-008 error (may fail to bind in CI — use a free port or mock `net.Listen`).

**Canonical E-CFG-008 message format (Ruling L — ARCH-12 v1.4):** The
`buildMgmtListener` console-loopback-rejection variant embeds the code as a prefix
in the error string for grep-ability in logs and test assertions:
```
E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: <address>
```
Test assertions MUST use `strings.Contains(err.Error(), "E-CFG-008")` — not a
full-string match that would break if wording is refined. This format is distinct
from the `config.Validate()` E-CFG-008 variant (empty/whitespace management_socket),
which reports via the standard `ConfigError` umbrella (top-level code `"E-CFG-001"`)
with the field-specific E-CFG-008 problem text embedded in the message string — NOT
as a separate code field. Both variants share the E-CFG-008 taxonomy code for lookup
but the message embedding differs by call site.

### AC-015 (traces to BC-2.07.004 Precondition 3 — ephemeral daemon keypair for access daemon, Ruling A.1)
`runAccess` in `cmd/switchboard/access.go` MUST generate an ephemeral Ed25519 private
key via `ed25519.GenerateKey(rand.Reader)` immediately before calling
`startMgmtServer`. The key is generated every process start and never written to disk.
If `GenerateKey` returns an error, `runAccess` returns it immediately and daemon
startup aborts. The ephemeral key is the daemon identity in bootstrap mode (when
`authorized_operator_keys` is empty); since it changes on every restart, client-side
daemon-identity pinning is deferred to S-6.02. No other daemon mode generates a
keypair this wave (router/console/control are stubs that do not start a mgmt server).
Canonical code (in `runAccess`, before `startMgmtServer`):
```go
_, daemonPrivKey, err := ed25519.GenerateKey(rand.Reader)
if err != nil {
    return fmt.Errorf("runAccess: generate daemon keypair: %w", err)
}
```
**Fatal-policy for `startMgmtServer` failure (Ruling J — ARCH-12 v1.4 / BC-2.07.004
EC-013):** If `startMgmtServer` returns an error of ANY kind (config-class E-CFG-008,
E-CFG-009, NewServer construction panic caught by recover, OR transient bind failure
such as EADDRINUSE), `runAccess` MUST return that error immediately. The data plane is
NOT started. There is no log-and-continue / degraded-management mode in the access
daemon this wave. Canonical pattern:
```go
mgmtSrv, err := startMgmtServer(ctx, &mgmtWG, cfg, "access", daemonPriv, nil)
if err != nil {
    return fmt.Errorf("access: start management server: %w", err)
}
```
- **Test:** `TestRunAccess_GeneratesEphemeralKey` — mock or stub `startMgmtServer`;
  verify it is called with a non-nil, 64-byte `daemonPrivKey`
  (`len(daemonPrivKey) == ed25519.PrivateKeySize`). Verify that a `GenerateKey` error
  (inject via mock) causes `runAccess` to return a non-nil error without calling
  `startMgmtServer`.
- **Test:** `TestRunAccess_MgmtStartFailureAborts` — inject a `startMgmtServer` stub
  that returns a non-nil error; verify `runAccess` returns non-nil without entering
  `buildAccessComponents` or `runAccessWithConnector`. Verify that neither the data-plane
  setup nor the main access loop is reached (traces to BC-2.07.004 EC-013 + Ruling J).

### AC-016 (traces to BC-2.07.004 Invariant 8 — nil-key construction guard, Ruling A.2)
`mgmt.NewServer` MUST panic immediately at construction time if
`len(daemonKey) != ed25519.PrivateKeySize` (64 bytes). A nil `daemonKey` has `len == 0`
and also triggers the panic. This fail-fast guard prevents a nil or truncated key from
reaching a connection goroutine where it would cause a nil-pointer dereference (remote
panic DoS). The check mirrors the existing `daemonVersion` emptiness panic in
`NewServer`. Canonical guard (immediately after the `daemonVersion` check):
```go
if len(daemonKey) != ed25519.PrivateKeySize {
    panic("mgmt.NewServer: daemonKey must be a valid ed25519.PrivateKey (64 bytes)")
}
```
- **Test:** `TestNewServer_PanicsOnNilKey` — use `defer func(){ recover() }()` to verify
  panic; pass `nil` as `daemonKey` to `NewServer`.
- **Test:** `TestNewServer_PanicsOnShortKey` — pass a 32-byte slice (ed25519.PublicKey
  size, a common mistake) as `daemonKey`; verify panic.

### AC-017 (traces to BC-2.07.004 PC-10 — Serve return contract, Rulings B/G/H/I)
`mgmt.Server.Serve` MUST return `nil` when `Shutdown` is called or when the context is
cancelled (normal daemon lifecycle). `Serve` returns a non-nil error ONLY on an
unexpected listener failure unrelated to shutdown (e.g., the underlying fd was stolen
by another process, or a bug in the wiring layer closes the wrong listener while ctx is
still live and `Shutdown` was never called).

**Accept-error predicate (Ruling G — ARCH-12 v1.4 / BC-2.07.004 PC-10):**
The canonical, authoritative predicate in the Accept error path is:
```go
if s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil) {
    s.connWG.Wait()
    return nil
}
s.connWG.Wait()
return err
```
The `&& ctx.Err() != nil` conjunct is REQUIRED. Without it, an unexpected external
`ln.Close()` (fd stolen, wiring bug) while ctx is live returns nil — silently killing
the management plane (violates SOUL #4: no silent failure). The `s.shuttingDown.Load()`
arm is unchanged.

**Dead `case <-done:` branches must be removed (Ruling H — ARCH-12 v1.4, reaffirming
Ruling B):** The pre-Accept semaphore select MUST use `case <-ctx.Done():` (not
`case <-done:`). The post-Accept-error non-default select block
(`select { case <-done: s.connWG.Wait(); return nil; default: }`) MUST be absent — the
`s.shuttingDown.Load()` check subsumes it. Backoff selects MUST use `case <-ctx.Done():`.
The ctx-watcher goroutine's own `case <-done:` is retained — it is NOT dead code because
`done` is closed when `Serve` returns, correctly exiting the watcher.

**Shutdown-window ordering and drain guarantee (Ruling I — ARCH-12 v1.4 /
BC-2.07.004 PC-10 drain contract):** After a successful `Accept()`, the accept loop
MUST check `s.shuttingDown.Load()` before calling `connWG.Add`. A connection accepted in
the shutdown window (after `shuttingDown` is set but before the loop exits) is dropped:
the conn is closed, the semaphore slot released, and the loop `continue`s — the connection
NEVER enters `connWG`. `trackConn` is called BEFORE `connWG.Add(1)` and BEFORE the
goroutine spawn so `closeAllConns` always captures every in-flight connection. This
ordering prevents the `connWG.Add`-after-`Wait`-at-zero panic (sync.WaitGroup misuse)
and the track-after-spawn drain gap (stalled conn survives `closeAllConns` snapshot).
Canonical ordering:
```go
if s.shuttingDown.Load() {
    _ = conn.Close()
    <-s.sem
    continue
}
s.trackConn(conn)     // BEFORE connWG.Add and BEFORE go func
s.connWG.Add(1)       // BEFORE go func
go func() {
    defer s.connWG.Done()
    defer s.untrackConn(conn)
    defer func() { <-s.sem }()
    s.handleConnection(ctx, conn)
}()
```

- **Test (a):** `TestServe_ReturnsNilOnShutdown` — start server via `net.Listen("tcp",
  "127.0.0.1:0")`; run `Serve` in a goroutine; call `Shutdown(context.Background())`;
  verify the `Serve` goroutine returns `nil` (not `net.ErrClosed` or any other error).
- **Test (b):** `TestServe_ReturnsNilOnCtxCancel` — start server with a cancellable
  context; cancel the context; verify `Serve` returns `nil`.
- **Test (c):** `TestServe_ReturnsErrOnUnexpectedListenerClose` (Ruling G) — create a
  server with a real listener (`net.Listen("tcp", "127.0.0.1:0")`); start `Serve` with
  `context.Background()` (live context, no cancel); do NOT call `Shutdown` and do NOT
  cancel ctx; close the listener directly from the test goroutine (`ln.Close()`); assert
  `Serve` returns a **non-nil** error. This test fails under the pre-Ruling-G code
  (returns nil) and passes after the `&& ctx.Err() != nil` fix. Closes the unexpected-
  close gap in VP-069 coverage (BC-2.07.004 PC-10).
- **Test (d):** `TestServe_ShutdownWindowNoAddAfterWaitPanic` (Ruling I — run with
  `-race`): create a server with `MaxConcurrentConnections=1`; spawn a goroutine that
  dials in a tight loop; simultaneously call `Shutdown` from the test goroutine; repeat
  100 iterations; assert no panic and race detector reports no WaitGroup misuse.
- **Test (e):** `TestServe_DrainCompletesWithinBudget` (Ruling I) — create a server;
  connect a client that completes auth and then stalls (stops reading); call `Shutdown`
  with a 2-second context; assert `Shutdown` returns within 2 s (NOT 10 s /
  `HandshakeTimeout`). The force-close via `closeAllConns` must unblock `handleConnection`
  quickly, proving the track-before-spawn fix works.

**Fatal-accept-error drain (Ruling P — ARCH-12 v1.5 / BC-2.07.004 PC-10):**
On the fatal-accept-error path (Accept returns a non-transient error while ctx is live
and Shutdown was never called — i.e., `shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil)` evaluates to false), `s.closeAllConns()` MUST be called
**immediately before** `s.connWG.Wait()` before returning the error. Without this,
in-flight connection goroutines remain running (blocked on a read deadline or network
I/O) until their own timeouts fire — up to `RPCIdleTimeout` (30 s) for an idle
authenticated connection. With `closeAllConns()`, they are force-closed and drain
completes within milliseconds. `closeAllConns()` is idempotent — calling it multiple
times is safe (double-close returns `net.ErrClosed`, ignored). This obligation applies
to ALL `Serve` return paths: `closeAllConns` MUST precede `connWG.Wait` on the
intentional-shutdown path, the ctx-cancel backoff path, AND the fatal-error path.
- **Test (f):** `TestServe_FatalAcceptErrorDrainsQuickly` (Ruling P / VP-069 v1.2) —
  create a server; connect a client that completes authentication and then stalls (stops
  reading); close the listener directly from the test goroutine (ctx is `context.Background()`,
  Shutdown never called); assert `Serve` returns within 200 ms (not blocked up to 30 s /
  `RPCIdleTimeout`). This test fails without the `closeAllConns()` insertion and passes
  after. (Closes the VP-069 v1.2 fatal-accept-error drain obligation added in BC-2.07.004
  PC-10 / Ruling P.)

**Test-quality note (Ruling T — ARCH-12 v1.5):**
`TestServe_ShutdownWindowNoAddAfterWaitPanic` (Ruling I) is a general concurrency-safety
smoke test under the race detector. Its stated RED rationale (Add-after-Wait-at-zero
panic from `connWG`) is structurally impossible in the current design because `Shutdown`
does NOT call `connWG.Wait()` — `Serve` is the sole `Wait` owner. The test's real
property is: "concurrent dial + Shutdown does not panic and passes `go test -race`."
The DROP behavior (connections in the shutdown window are discarded) is discriminatingly
tested by `TestServe_DrainCompletesWithinBudget` (test (e) above). Test-writers MUST
re-document this test with its accurate property (Option A), or rebuild it as a
deterministic harness that directly validates the `shuttingDown` pre-check ordering
(Option B). No production code change is required.

### AC-018 (traces to BC-2.07.004 PC-1 amended — write deadlines before every sendJSON, Ruling E)
Before every `sendJSON` call in `handleConnection`, `conn.SetWriteDeadline` MUST be
set to `time.Now().Add(d)` where `d` is:
- `HandshakeTimeout` (10 s) for handshake-phase sends: CHALLENGE, AUTH_OK, AUTH_FAIL
- `RPCIdleTimeout` (30 s) for RPC response sends

The write deadline MUST be cleared to `time.Time{}` after each send completes. This
closes the symmetric slowloris-on-write slot-exhaustion vector (CWE-400): a client that
completes auth and then stops reading its socket would otherwise pin a
`MaxConcurrentConnections` semaphore slot indefinitely. Implementation uses inline
call-site deadline setting (Option 1 per ARCH-12 §Ruling E), not a modified `sendJSON`
signature.
- **Test:** `TestWriteDeadline_SlowlorisDefense_VP072` — use `net.Pipe`; complete
  authentication (AUTH_OK); then stop reading on the client pipe (do not drain); send
  one RPC request from the client before stopping the drain; verify the server's
  `sendJSON` for the response fails within `RPCIdleTimeout` (the server does NOT block
  indefinitely). Construct the server with a short `RPCIdleTimeout` override (e.g.,
  50 ms via `WithRPCIdleTimeout`) to keep the test fast. Note: `WithRPCIdleTimeout`
  governs ALL THREE RPC-phase timeout sites — the post-AUTH_OK RPC read deadline, the
  RPC-response write deadline, AND the per-handler execution timeout — not only the
  handler context. The AC-018 "construct with a short RPCIdleTimeout override (50ms)"
  test text is accurate and achievable once the `RPCIdleTimeout` field is correctly
  wired to all three sites in `mgmt.go` (lines 578/661/670).
  (VP assignment pending — BC-2.07.004 VP Anchors note: PC-10/PC-11/PC-12/Invariant 8
  are not yet covered by an assigned VP; cite BC clause until architect assigns VP numbers)

### AC-019 (traces to BC-2.07.004 EC-013, Ruling O — stale-socket pre-bind cleanup)
`listenUnixMgmt` in `cmd/switchboard/mgmt_wire.go` performs a pre-bind cleanup:
if `os.Lstat(path)` succeeds and the result has `os.ModeSocket` set in its mode
bits, `os.Remove(path)` is called before `syscall.Bind`. Regular files and
directories at the path are NOT removed — `Bind` fails with the original error
(not silently clobbered). `os.Remove` errors are silently ignored. This prevents
an EADDRINUSE restart DoS after a non-graceful daemon exit (SIGKILL, crash, OOM
kill) where the socket inode persists on the filesystem. The Lstat→Remove→Bind
TOCTOU window is accepted: it repairs exactly the restart-after-crash case and
is no worse than the current EADDRINUSE failure. (Cites BC-2.07.004 EC-013 /
Ruling O, ARCH-12 v1.5.)
- **Test:** `TestListenUnixMgmt_PreBindCleanup_AC019` —
  (a) create a temp path via `t.TempDir()`; call `listenUnixMgmt` once (succeeds);
  close the returned listener WITHOUT unlinking, so the socket inode persists; call
  `listenUnixMgmt` again on the same path; verify it succeeds (no EADDRINUSE).
  (b) create a regular file at the same temp path; call `listenUnixMgmt`; verify it
  returns a non-nil error and the regular file was NOT removed.

### AC-020 (traces to BC-2.07.004 PC-6 / Ruling R — per-handler execution timeout)
Every registered handler `Fn` is invoked with a child context derived via
`context.WithTimeout(ctx, RPCIdleTimeout)` (default 30 s). A handler that blocks
past this budget is cancelled via the child context; the server returns E-RPC-011
to the client (`"handler context deadline exceeded"` or the cancellation error
message). The connection is NOT closed on handler timeout. This bounds handler
execution to the same time budget as the RPC-phase read deadline, closing the
CWE-400 goroutine-pin surface on the handler dispatch path — a blocking handler
cannot permanently pin a connection goroutine and semaphore slot. (Cites
BC-2.07.004 PC-6 / Ruling R, ARCH-12 v1.5.)
- **Test:** `TestMgmtServer_HandlerTimeout_AC020` — register a handler that blocks
  until its context is cancelled (select on ctx.Done()); construct server with a
  short `RPCIdleTimeout` override (e.g., 50 ms via `WithRPCIdleTimeout` option or
  equivalent constructor parameter); authenticate and send the RPC; verify the server
  returns an E-RPC-011 error response within ~200 ms (not blocked indefinitely).
  Verify the connection remains open after the timed-out handler (handler timeout is
  NOT a connection-close event; client can send a subsequent RPC).

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| `mgmt.Server` | internal/mgmt | effectful (boundary) — owns listener I/O, connection state |
| `mgmt.OperatorKeySet` | internal/mgmt | pure-core (key comparison) |
| `mgmt.NewServer` / `Serve` / `Shutdown` | internal/mgmt | effectful |
| Config additions | internal/config | pure-core (validation) |
| Daemon wiring | cmd/switchboard | effectful |

See ARCH-12 §internal/mgmt Package Design and §Classification for the boundary
vs. effectful classification rationale.

## Package DAG Constraints (Forbidden Dependencies)

`internal/mgmt` MUST NOT import:
- `internal/routing`, `internal/multipath`, `internal/arq`, `internal/replay`,
  `internal/paths`, `internal/halfchannel`, `internal/session`, `internal/tmux`,
  `internal/discovery` — data-plane packages
- `cmd/sbctl` — downward import forbidden
- `internal/svtnmgmt` — management RPC handlers receive this via closure injection,
  not as a mgmt package dependency
- `internal/admission` — management auth is an independent domain; do NOT call
  `admission.AdmitNode` or `admission.GenerateChallenge`
- `golang.org/x/crypto` — server uses stdlib `crypto/ed25519` only

**Build-time enforcement:** If `internal/mgmt` ever gains a dependency on any of
the above packages, the build MUST fail (add a `go vet` check or CI `go list`
assertion).

Nothing imports `internal/mgmt` except `cmd/switchboard`. `cmd/sbctl` speaks the
wire protocol, not the Go API.

## Wiring Pattern (access daemon only this wave)

Per ARCH-12 §Wiring into cmd/switchboard and the Scope Note above (Ruling A.1
access-only scope), only `runAccess` wires the management server this wave. Router,
console, and control are stubs. The access-daemon pattern is:

```go
// In runAccess only (access-only scope this wave):
// 1. Generate ephemeral daemon keypair (AC-015, Ruling A.1)
_, daemonPrivKey, err := ed25519.GenerateKey(rand.Reader)
if err != nil {
    return fmt.Errorf("runAccess: generate daemon keypair: %w", err)
}
operatorKeys := mgmt.NewOperatorKeySet(cfg.AuthorizedOperatorKeys)
// Unix socket: set umask to 0177 before Listen for atomic 0600 permission (AC-014)
old := syscall.Umask(0177)
mgmtLn, err := net.Listen("unix", cfg.ManagementSocket)
syscall.Umask(old)
// ... handle err
handlers := buildAccessHandlers(...)
// daemonVersion is injected from cmd/switchboard.version via ldflags (Ruling 6)
// NewServer panics if len(daemonPrivKey) != ed25519.PrivateKeySize (AC-016, Ruling A.2)
mgmtSrv := mgmt.NewServer(mgmtLn, daemonPrivKey, operatorKeys, handlers, version)
wg.Add(1)
go func() { defer wg.Done(); _ = mgmtSrv.Serve(ctx) }()
// On shutdown: mgmtSrv.Shutdown(shutdownCtx)
```

Socket addresses (ARCH-05 §Daemon Management Socket):
- router: `cfg.ManagementSocket` (default `/run/switchboard-router.sock`)
- access: `cfg.ManagementSocket` (default `/run/switchboard-access.sock`)
- console: `cfg.ManagementSocket` (default `127.0.0.1:9091`, TCP)
- control: `cfg.ManagementSocket` (default `/run/switchboard-control.sock`)

Mode-specific defaults are applied by the daemon mode's startup code, not by
`Validate()`. If `ManagementSocket` is empty in config, the mode applies its
own default before calling `net.Listen`.

## Edge Cases

| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| EC-001 | Client connects and sends nothing (HandshakeTimeout silent stall) | Server sends CHALLENGE, applies `mgmt.HandshakeTimeout` read deadline (default 10 s) on the subsequent read. On timeout expiry: close-only (no AUTH_FAIL on timeout); non-timeout decode errors (malformed JSON, oversized message, EOF before timeout) DO send AUTH_FAIL before close (BC-2.07.004 EC-001 v1.4 / Ruling K); no goroutine leak |
| EC-002 | Unrecognized public key | `OperatorKeySet.IsAuthorized` returns false → AUTH_FAIL (E-ADM-010, same as wrong-signature). No oracle. |
| EC-003 | Recognized key, wrong signature | `ed25519.Verify` returns false → AUTH_FAIL (E-ADM-010). Connection closed. |
| EC-004 | Message > 64 KiB | `io.LimitReader` → decode error → connection closed. No OOM (CWE-400). |
| EC-005 | Malformed JSON (truncated) | `json.Decoder.Decode` returns error → connection closed; no panic |
| EC-006 | Client sends `"type":"request"` before handshake | AUTH_FAIL + close; no RPC dispatched |
| EC-007 | Client disconnects mid-handshake | EOF/read error detected; connection goroutine cleaned up; no leak |
| EC-008 | `authorized_operator_keys` empty (bootstrap mode) | Daemon's own keypair is the authorized key; normal handshake |
| EC-009 | `management_socket: "   "` (whitespace-only) | `Validate()` returns E-CFG-008; daemon exits 1 before listening |
| EC-010 | `authorized_operator_keys[1]` contains RSA PEM | `Validate()` returns E-CFG-009 for index 1; exhaustive error collection |
| EC-011 | Non-Switchboard peer (arbitrary byte stream) | `io.LimitReader` + `json.Decoder` returns error within 64 KiB; connection closed cleanly |
| EC-012 | Concurrent connections exceed `MaxConcurrentConnections` (default 128) | Accept-loop semaphore full; new `Accept()` calls block; no new goroutines spawned beyond limit; transient EMFILE errors trigger exponential backoff (5 ms–1 s); fatal errors cause `Serve` to return (BC-2.07.004 EC-012) |
| EC-013 | Unix socket created without restrictive umask; or console TCP bound to non-loopback address | Must use `syscall.Umask(0177)` before `net.Listen` for Unix socket; socket created with 0600 permissions atomically; no chmod-after-Listen TOCTOU window. Console TCP: `buildMgmtListener` validates host is 127.0.0.1/[::1]/localhost before `net.Listen`; any other host emits E-CFG-008 and daemon aborts (BC-2.07.004 EC-013 + Invariant 7 + Ruling D) |
| EC-014 | Authenticated RPC names an unregistered command | Server responds `{"ok":false,"error":{"code":"E-RPC-010","message":"unknown command: <cmd>"},"data":null}` in-band; connection NOT closed; client may send further RPCs (BC-2.07.004 PC-11) |
| EC-015 | Registered handler returns a non-nil error | Server responds `{"ok":false,"error":{"code":"E-RPC-011","message":"<err>"},"data":null}` in-band (handler error string verbatim); connection NOT closed (BC-2.07.004 PC-12) |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| internal/mgmt (Server, Serve, Shutdown) | effectful (boundary) | Owns listener I/O, connection state, goroutine lifecycle |
| internal/mgmt (OperatorKeySet.IsAuthorized) | pure-core | Deterministic key comparison; no I/O |
| internal/config additions | pure-core | Validation is deterministic; no I/O |
| cmd/switchboard wiring | effectful | Daemon startup; socket binding |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~4,000 |
| BC-2.07.004.md (v1.5) | ~2,500 |
| BC-2.09.003.md (v1.7, PC-10/PC-11 sections) | ~1,500 |
| ARCH-12 §internal/mgmt Package Design + §ADR-012 + §Wiring | ~2,500 |
| ARCH-01 §Goroutine WaitGroup Contract | ~300 |
| ARCH-05 §Daemon Management Socket | ~400 |
| interface-definitions.md §JSON Output Schema | ~400 |
| VP-064.md + VP-065.md + VP-066.md (proof harnesses) | ~2,000 |
| internal/config/config.go (existing) | ~1,500 |
| cmd/switchboard/main.go (existing) | ~2,000 |
| Test files (estimated) | ~2,500 |
| Tool outputs overhead | ~500 |
| **Total** | **~20,100** |
| Agent context window | 200K |
| **Budget usage** | **~10.1%** |

## Tasks (MANDATORY)

1. [ ] Read BC-2.07.004 (full), BC-2.09.003 (PC-10/PC-11 sections), ARCH-12 (full), VP-064.md, VP-065.md, VP-066.md
2. [ ] Read ARCH-01 §Goroutine WaitGroup Contract; ARCH-05 §Daemon Management Socket
3. [ ] Read `internal/config/config.go` and existing `Validate()` implementation
4. [ ] Read `cmd/switchboard/main.go` to understand existing daemon mode structure
5. [ ] Write failing tests for AC-001 through AC-014 (Red Gate)
6. [ ] Verify Red Gate — all tests must fail before implementation starts
7. [ ] Create `internal/mgmt/mgmt.go`:
   - `const MaxMessageBytes = 1 << 16` (64 KiB)
   - `const HandshakeTimeout = 10 * time.Second` (Ruling 1)
   - `const RPCIdleTimeout = 30 * time.Second` (Ruling 1)
   - `const MaxConcurrentConnections = 128` (Ruling 3)
   - `type OperatorKeySet` with `NewOperatorKeySet(keys []ed25519.PublicKey)` and
     `IsAuthorized(pubkey ed25519.PublicKey) bool` (constant-time comparison)
   - `type Handler struct { Command string; Fn func(ctx context.Context, args json.RawMessage) (any, error) }`
   - `type Server struct` (unexported fields: listener, daemonKey, ops, handlers, wg,
     semaphore chan struct{}, daemonVersion string)
   - `func NewServer(ln net.Listener, daemonKey ed25519.PrivateKey, ops *OperatorKeySet, handlers []Handler, daemonVersion string) *Server`
     — panics if `daemonVersion == ""` (document this enforcement choice in comments)
   - `func (s *Server) Serve(ctx context.Context) error` — accept loop with
     `MaxConcurrentConnections` semaphore; exponential backoff on transient Accept errors;
     per-connection goroutine, WaitGroup-tracked
   - `func (s *Server) Shutdown(ctx context.Context) error` — drain + close listener
   - `handleConnection(ctx, conn)` — ADR-012 auth handshake:
     (a) send CHALLENGE; (b) `conn.SetReadDeadline(time.Now().Add(HandshakeTimeout))`;
     (c) read CHALLENGE_RESPONSE via `io.LimitReader(conn, MaxMessageBytes)`;
     (d) verify; (e) send AUTH_OK (with `daemon_version`) or AUTH_FAIL + close;
     (f) if AUTH_OK: `conn.SetReadDeadline(time.Now().Add(RPCIdleTimeout))`; read RPC;
     (g) maintain per-connection `authenticated` boolean — post-auth CHALLENGE_RESPONSE
         triggers E-ADM-010 + close (Ruling 7)
   - All reads via `io.LimitReader(conn, MaxMessageBytes)`
8. [ ] Add `ManagementSocket string` and `AuthorizedOperatorKeys []string` to `internal/config/config.go`
9. [ ] Extend `Validate()` with E-CFG-008 and E-CFG-009 (exhaustive error collection per existing pattern)
10. [ ] Wire mgmt listener into `runAccess` ONLY (access-only scope this wave — Ruling A.1):
    generate ephemeral keypair via `ed25519.GenerateKey(rand.Reader)` before
    `startMgmtServer` (AC-015); use `syscall.Umask(0177)` before `net.Listen("unix", ...)`;
    pass `version` (ldflags-injected) as `daemonVersion` to `NewServer` (AC-007, AC-014);
    router/console/control remain stubs with no mgmt listener (no orphaned goroutine)
11. [ ] Write test `TestMgmtServer_ConnectionCap_AC013` for bounded accept loop (AC-013)
12. [ ] Write test `TestDaemonWiring_UnixSocketPermissions_AC014` for 0600 socket creation (AC-014)
13. [ ] Add nil-key guard in `NewServer` (`len(daemonKey) != ed25519.PrivateKeySize` → panic);
    write tests `TestNewServer_PanicsOnNilKey` and `TestNewServer_PanicsOnShortKey` (AC-016)
14. [ ] Add `shuttingDown atomic.Bool` to `Server`; set in `Shutdown` and ctx-watcher goroutine
    before `s.ln.Close()`; update Accept error path; remove dead `case <-done:` branches;
    write tests `TestServe_ReturnsNilOnShutdown` and `TestServe_ReturnsNilOnCtxCancel` (AC-017)
15. [ ] Replace `E-RPC-001`/`E-RPC-002` in `handleConnection` dispatch with `E-RPC-010` (unknown
    command) and `E-RPC-011` (handler error); verify no `E-RPC-002` survives in `internal/mgmt`;
    add test table rows to AC-007 test (AC-007 amendment, Ruling C)
16. [ ] Add loopback-validation block in `buildMgmtListener` for TCP/console path (reject non-
    loopback host with E-CFG-008); write tests `TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback`
    and `TestBuildMgmtListener_ConsoleTCP_AcceptsLoopback127` (AC-014 amendment, Ruling D)
17. [ ] Add `conn.SetWriteDeadline` calls before every `sendJSON` in `handleConnection`
    (HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC responses); clear deadline
    after each send; write test `TestWriteDeadline_SlowlorisDefense_VP072` (AC-018, Ruling E)
18. [ ] `just fmt && just lint` pass
19. [ ] Verify VP-064, VP-065, VP-066 test assertions pass

## Previous Story Intelligence (MANDATORY)

| Story | Key Decisions | Patterns Established | Gotchas Discovered |
|-------|--------------|---------------------|-------------------|
| S-6.01 | Config `Validate()` uses collect-all-failures pattern | Exhaustive error reporting before exit 1 | New config fields must follow same pattern |
| S-2.02 | Admission uses `admission.GenerateChallenge` + `admission.AdmitNode` | Challenge-response pattern | Management auth is INDEPENDENT — do NOT reuse AdmitNode; use `crypto/rand.Read(32)` + `ed25519.Sign` directly |
| S-W3.04 | Daemon assembly: all subsystems wired in `cmd/switchboard` with `wg.Add(1)` goroutine pattern | WaitGroup lifecycle contract per ARCH-01 | New management server goroutine follows same pattern |
| S-6.03 | Client sends `{"type":"challenge_response","nonce_sig":"...","pubkey":"..."}` | ADR-012 wire format is fixed | nonce_sig and pubkey are base64url-encoded (no padding) |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|-------------|
| Server uses `crypto/ed25519` (stdlib) — NOT `golang.org/x/crypto` | ARCH-12 §Dependencies | `go mod tidy`; `go list -m all` |
| `internal/mgmt` MUST NOT import data-plane packages (routing, multipath, arq, etc.) | ARCH-12 §Package DAG Constraints | `go vet`; CI `go list` assertion |
| `internal/mgmt` MUST NOT import `internal/admission` | ARCH-12 §Reuse vs. Differences | `go vet` |
| All socket reads via `io.LimitReader(conn, MaxMessageBytes)` — no unbounded reads | ADR-012 §6 Bounded Read (CWE-400); BC-2.07.004 PC-6, Inv-3 | `TestMgmtServer_BoundedRead_VP066`; `FuzzMgmtServer_BoundedRead_VP066` |
| `OperatorKeySet.IsAuthorized` uses constant-time comparison (`subtle.ConstantTimeCompare`) | BC-2.07.004 PC-8, Inv-5 | Code review + `TestOperatorKeySet_ConstantTimeCompare_AC008` |
| mgmt goroutine is `wg.Add(1)` / `defer wg.Done()` tracked per ARCH-01 | ARCH-01 §Goroutine WaitGroup Contract; ARCH-12 §Daemon Mode Startup | `TestMgmtServer_GracefulShutdown_AC010` |
| AUTH_FAIL responses are identical regardless of key recognition — no oracle | BC-2.07.004 PC-8, Inv-5 | `TestMgmtServer_RejectsUnauthenticated_VP064` checks both sub-cases return same message |
| Handlers registry is constructor-injected — no `init()` side effects | ARCH-12 §Command Handler Registry; go.md rule 10 | Code review |
| E-CFG-008 and E-CFG-009 follow existing `Validate()` collect-all-failures pattern | BC-2.09.003 Inv-4 | `TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012` (multiple entries) |
| `conn.SetReadDeadline(time.Now().Add(HandshakeTimeout))` before every CHALLENGE_RESPONSE read; `RPCIdleTimeout` after AUTH_OK | ADR-012 §7; BC-2.07.004 PC-1, EC-001 (Ruling 1) | `TestMgmtServer_IssuesChallengeFirst_AC001` (deadline sub-case) |
| Accept loop bounded by `MaxConcurrentConnections` semaphore (buffered channel); transient Accept errors backed off exponentially | ADR-012 §8; BC-2.07.004 EC-012 (Ruling 3) | `TestMgmtServer_ConnectionCap_AC013` |
| Unix socket created with `syscall.Umask(0177)` before `net.Listen` — NOT chmod-after-Listen; console TCP binds 127.0.0.1 only | ADR-012 §Unix Socket Permissions; BC-2.07.004 EC-013 + Invariant 7 (Ruling 4) | `TestDaemonWiring_UnixSocketPermissions_AC014` |
| `NewServer` `daemonVersion` param passed from `cmd/switchboard.version` (ldflags); panics on empty string | ADR-012 §Ruling 6; BC-2.07.004 PC-7 | `TestMgmtServer_AuthOK_DispatchesRPC_AC007` daemon_version assertion |
| Per-connection `authenticated` boolean — post-auth `challenge_response` triggers E-ADM-010 + close (structural guard, NOT nonce-set) | ADR-012 §PC-3 Replay-Nonce Ruling 7; BC-2.07.004 PC-3 | `TestMgmtServer_PostAuthChallengeResponseRejected_VP065` |
| `runAccess` generates ephemeral Ed25519 keypair via `ed25519.GenerateKey(rand.Reader)` before `startMgmtServer`; GenerateKey error aborts daemon | ARCH-12 Ruling A.1; BC-2.07.004 Precondition 3 | `TestRunAccess_GeneratesEphemeralKey` |
| `NewServer` panics if `len(daemonKey) != ed25519.PrivateKeySize`; nil key panics immediately | ARCH-12 Ruling A.2; BC-2.07.004 Invariant 8 | `TestNewServer_PanicsOnNilKey`; `TestNewServer_PanicsOnShortKey` |
| `Serve` returns nil on `Shutdown` or ctx-cancel; `shuttingDown atomic.Bool` mediates; dead `case <-done:` branches removed | BC-2.07.004 PC-10; ARCH-12 Ruling B | `TestServe_ReturnsNilOnShutdown`; `TestServe_ReturnsNilOnCtxCancel` |
| Server emits E-RPC-010 for unknown commands and E-RPC-011 for handler errors; connection NOT closed; E-RPC-002 MUST NOT appear in `internal/mgmt` | BC-2.07.004 PC-11/PC-12; ARCH-12 Ruling C | AC-007 test table rows (unknown command + handler error) |
| `buildMgmtListener` validates TCP/console host is 127.0.0.1/[::1]/localhost before `net.Listen`; non-loopback emits E-CFG-008 | BC-2.07.004 EC-013; ARCH-12 Ruling D | `TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback`; `TestBuildMgmtListener_ConsoleTCP_AcceptsLoopback127` |
| `conn.SetWriteDeadline` set before every `sendJSON` (HandshakeTimeout for handshake, RPCIdleTimeout for RPC); cleared after each send | BC-2.07.004 PC-1 (amended); ARCH-12 Ruling E | `TestWriteDeadline_SlowlorisDefense_VP072` |

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Per go.mod (mise-pinned) |
| `crypto/ed25519` | stdlib | `ed25519.Sign`, `ed25519.Verify`, `ed25519.PrivateKey`, `ed25519.PublicKey` |
| `crypto/rand` | stdlib | `rand.Read(32)` for nonce generation |
| `crypto/subtle` | stdlib | `subtle.ConstantTimeCompare` for key comparison (no oracle) |
| `encoding/json` | stdlib | NDJSON message framing (ADR-012) |
| `encoding/pem` | stdlib | PEM parsing for `authorized_operator_keys` validation (E-CFG-009) |
| `crypto/x509` | stdlib | `x509.ParsePKIXPublicKey` for Ed25519 PEM PUBLIC KEY validation |
| `net` | stdlib | `net.Listener`, `net.Conn`, `net.Listen` |
| `io` | stdlib | `io.LimitReader` (CWE-400 bounded reads) |
| `context` | stdlib | `context.Context` for `Serve`/`Shutdown` cancellation |
| `sync` | stdlib | `sync.WaitGroup` for connection goroutine tracking |
| `encoding/base64` | stdlib | base64url encoding/decoding for nonce and signatures |
| `syscall` | stdlib | `syscall.Umask(0177)` for atomic Unix socket 0600 permission (AC-014, CWE-276) |
| `time` | stdlib | `HandshakeTimeout`, `RPCIdleTimeout` constants; `conn.SetReadDeadline` |

**No external dependencies** added by this story on the server side.

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `internal/mgmt/mgmt.go` | create | Full `internal/mgmt` package: constants, types, `Server`, `OperatorKeySet`, `Handler`, `NewServer`, `Serve`, `Shutdown`, `handleConnection` |
| `internal/mgmt/mgmt_test.go` | create | Unit and integration tests: AC-001 through AC-010, AC-013, AC-015 (ephemeral key), AC-016 (nil-key panic), AC-017 (Serve returns nil + fatal-accept-error drain), AC-018 (write deadline), AC-020 (handler timeout); VP-064, VP-065, VP-066 test harnesses; fuzz target `FuzzMgmtServer_BoundedRead_VP066`; `TestMgmtServer_ConnectionCap_AC013`; `TestNewServer_PanicsOnNilKey`; `TestNewServer_PanicsOnShortKey`; `TestServe_ReturnsNilOnShutdown`; `TestServe_ReturnsNilOnCtxCancel`; `TestServe_FatalAcceptErrorDrainsQuickly`; `TestWriteDeadline_SlowlorisDefense_VP072`; `TestMgmtServer_HandlerTimeout_AC020` |
| `internal/config/config.go` | modify | Add `ManagementSocket string` and `AuthorizedOperatorKeys []string` fields; extend `Validate()` with E-CFG-008 and E-CFG-009 |
| `internal/config/config_test.go` | modify | Add `TestConfig_Validate_ManagementSocket_E_CFG_008_AC011` and `TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012` |
| `cmd/switchboard/access.go` (or equivalent run file) | modify | Wire mgmt listener into `runAccess` ONLY: generate ephemeral keypair, `syscall.Umask(0177)` before Unix socket Listen, pass `version` as `daemonVersion` to `NewServer`; router/console/control remain stubs |
| `cmd/switchboard/mgmt_wire.go` | create or modify | `buildMgmtListener`: TCP/console loopback-host validation (E-CFG-008 rejection) before `net.Listen`; Unix-socket umask wrapper |
| `cmd/switchboard/main_test.go` (or per-mode test file) | modify | Add `TestDaemonWiring_UnixSocketPermissions_AC014`; `TestRunAccess_GeneratesEphemeralKey`; `TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback`; `TestBuildMgmtListener_ConsoleTCP_AcceptsLoopback127`; `TestListenUnixMgmt_PreBindCleanup_AC019` |

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.6 | 2026-06-29 | story-writer | Architect Ruling 1 (S-W5.01 mgmt-server convergence): add AC-003 deferral note — BC-2.07.004 PC-3 / EC-004 "logs a security event" requirement explicitly deferred to S-HRD.02 (daemon logging infrastructure / slog seam); this AC covers only fail-closed connection control (AUTH_FAIL + E-ADM-010 + close), fully tested via VP-065. No AC count change. Opens S-HRD.02 as follow-up hardening stub. |
| 1.5 | 2026-06-29 | story-writer | ARCH-12 Round-5 Rulings: (F3) AC-011 sub-case (a) corrected — empty string = zero-value = absent = accepted, not E-CFG-008; test renamed empty_string_accepted_as_absent. (F4) WithRPCIdleTimeout doc clarified to govern all three RPC-phase deadline sites (read/write/handler); enables AC-018 50ms-override test. (F5) Ruling L prose corrected — config.Validate E-CFG-008 uses ConfigError umbrella E-CFG-001 with embedded message, not a code field. (F6) stale "all four daemon modes" comments to be scrubbed by implementer (access.go + mgmt_wire_test.go). No production logic change for F3/F5/F6; F4 is a 3-line const→field change. |
| 1.4 | 2026-06-29 | story-writer | BC-2.07.004 v1.5 / ARCH-12 v1.5 Wave-5 Convergence Rulings O–T: (O) AC-019 added — `listenUnixMgmt` pre-bind cleanup: `os.Lstat`+`os.Remove` for `ModeSocket` inodes before `syscall.Bind`; prevents EADDRINUSE restart DoS after non-graceful exit (BC-2.07.004 EC-013 / Ruling O); test `TestListenUnixMgmt_PreBindCleanup_AC019`. (P) AC-017 extended — fatal-accept-error drain: `closeAllConns()` MUST precede `connWG.Wait()` on fatal-accept-error path (ctx live, Shutdown never called); VP-069 v1.2 obligation; test `TestServe_FatalAcceptErrorDrainsQuickly` added (BC-2.07.004 PC-10 / Ruling P). (Q) AC-001 prose corrected — "On deadline expiry the connection is closed with E-ADM-010" → "close-only, no AUTH_FAIL on HandshakeTimeout expiry" (propagation gap from Ruling K); EC-001 table row corrected to "close-only (no AUTH_FAIL on timeout)". (R) AC-020 added — handler execution timeout: `context.WithTimeout(ctx, RPCIdleTimeout)` wraps every handler `Fn` call; blocking handlers cancelled; E-RPC-011 returned; connection not closed (BC-2.07.004 PC-6 / Ruling R); test `TestMgmtServer_HandlerTimeout_AC020`. (T) Ruling T test-quality note added to AC-017 area — `TestServe_ShutdownWindowNoAddAfterWaitPanic` is a race-detector smoke test, not an Add-after-Wait-at-zero discriminator; real DROP property covered by `TestServe_DrainCompletesWithinBudget`. BC version pins updated: BC-2.07.004 v1.5, BC-2.09.003 v1.7. Stale `TestMgmtServer_WriteDeadlineSet_AC018` traceability reference corrected to `TestWriteDeadline_SlowlorisDefense_VP072`. acceptance_criteria_count 18→20. |
| 1.3 | 2026-06-29 | story-writer | BC-2.07.004 v1.4 / ARCH-12 v1.4 Wave-5 Convergence Rulings G–N: (G) AC-017 sub-case (c) added — `TestServe_ReturnsErrOnUnexpectedListenerClose`: listener closed directly (not via Shutdown/ctx-cancel); assert Serve returns non-nil; closes unexpected-close gap in VP-069 coverage (BC-2.07.004 PC-10 canonical predicate `shuttingDown.Load() \|\| (errors.Is(err, net.ErrClosed) && ctx.Err() != nil)`). (H) AC-017 implementation-contract note added — dead `case <-done:` branches in the accept loop must be replaced/removed: pre-Accept semaphore select uses `case <-ctx.Done():`; post-Accept-error non-default select block absent; backoff selects use `case <-ctx.Done():`; ctx-watcher goroutine's own `case <-done:` retained. (I) AC-017 drain-ordering guarantee added — connections accepted in shutdown window are dropped without entering connWG; `trackConn` called before `connWG.Add` and before goroutine spawn; two new test obligations: `TestServe_ShutdownWindowNoAddAfterWaitPanic` (-race, 100 iterations) and `TestServe_DrainCompletesWithinBudget` (stalled conn, 2s budget). (J) AC-015 fatal-policy added — any `startMgmtServer` failure (config-class or transient bind) aborts `runAccess`; data plane NOT started; test `TestRunAccess_MgmtStartFailureAborts` added (BC-2.07.004 EC-013). (K) AC-001 timeout sub-case assertion amended — timeout close is silent (close-only, NO AUTH_FAIL); assertion updated to verify NO auth_fail received before EOF (BC-2.07.004 EC-001 v1.4). (L) AC-014 canonical E-CFG-008 format comment added — `buildMgmtListener` variant embeds code as prefix; `strings.Contains(err.Error(), "E-CFG-008")` required for test assertions. (N) `vp_traces` frontmatter updated from `[VP-064, VP-065, VP-066]` to `[VP-064, VP-065, VP-066, VP-068, VP-069, VP-070, VP-071, VP-072, VP-073]`; VP-068..073 added to inputDocuments. |
| 1.2 | 2026-06-29 | story-writer | BC-2.07.004 v1.3 / ARCH-12 v1.3 Wave-5 Convergence Rulings A–E: (A) AC-015 added — `runAccess` generates ephemeral Ed25519 keypair via `ed25519.GenerateKey(rand.Reader)` before `startMgmtServer`; GenerateKey error aborts startup (BC-2.07.004 Precondition 3). AC-016 added — `mgmt.NewServer` panics if `len(daemonKey) != ed25519.PrivateKeySize` (Invariant 8). Scope Note rewritten — access-only scope, contradicting "all four modes" clause dropped; router/console/control are stubs with no mgmt listener (resolves Round-1 H-2 self-contradiction). (B) AC-017 added — `Serve` returns nil on Shutdown/ctx-cancel via `shuttingDown atomic.Bool`; non-nil only on fatal Accept error unrelated to shutdown (BC-2.07.004 PC-10). (C) AC-007 updated — E-RPC-001/E-RPC-002 replaced with E-RPC-010 (unknown command, in-band) and E-RPC-011 (handler error, in-band); connection not closed; test table rows added (BC-2.07.004 PC-11/PC-12). (D) AC-014 updated — `buildMgmtListener` loopback-validation added for TCP/console path; E-CFG-008 on non-loopback host; tests added (BC-2.07.004 EC-013 + Ruling D). (E) AC-018 added — `conn.SetWriteDeadline` before every `sendJSON` (HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC responses), cleared after send; closes CWE-400 write-side slot-exhaustion (BC-2.07.004 PC-1 amended). BC table updated with new PCs/Invariant 8. EC-014, EC-015 added. Architecture Compliance Rules extended. Tasks 13–19 added. acceptance_criteria_count 14→18. |
| 1.1 | 2026-06-28 | story-writer | ARCH-12 v1.2 adversarial review rulings propagated: AC-001 amended with HandshakeTimeout=10s deadline + goroutine-leak test (Ruling 1); AC-002 sub-case (d) added (HandshakeTimeout silent-stall closes connection, Ruling 1); AC-003 REPLACED with structural post-auth guard (per-connection `authenticated` boolean, not nonce-set) + test `TestMgmtServer_PostAuthChallengeResponseRejected_VP065` (Ruling 7, BC-2.07.004 PC-3); AC-007 amended with `daemon_version` assertion and `daemonVersion string` param in `NewServer` — panics on empty string (Ruling 6, BC-2.07.004 PC-7); AC-013 ADDED (bounded accept loop semaphore MaxConcurrentConnections=128, EMFILE backoff, BC-2.07.004 EC-012, Ruling 3); AC-014 ADDED (Unix socket 0600 via syscall.Umask(0177), no chmod-after TOCTOU, console TCP 127.0.0.1-only, BC-2.07.004 EC-013 + Invariant 7, Ruling 4); Scope Note updated — ALL FOUR daemon modes required (access non-stub critical finding); Wiring Pattern updated with umask and daemonVersion; Library section adds syscall + time; Architecture Compliance Rules updated; acceptance_criteria_count 12→14. |
| 1.0 | 2026-06-28 | story-writer | Initial creation — Wave-5 net-new story per ARCH-12 product-owner handoff. internal/mgmt server + config E-CFG-008/009 + cmd/switchboard wiring. |
