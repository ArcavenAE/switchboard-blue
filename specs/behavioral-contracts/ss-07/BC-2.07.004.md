---
artifact_id: BC-2.07.004
document_type: behavioral-contract
level: L3
version: "1.6"
status: draft
producer: product-owner
timestamp: 2026-06-28T00:00:00
phase: 1a
bc_id: BC-2.07.004
subsystem: network-management
architecture_module: internal/mgmt
capability: CAP-024
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-06-28
    version: "1.1"
    change: >
      Wave-5 consistency audit F-004: subsystem field corrected from SS-07 (ID form)
      to network-management (canonical name) to match sibling BCs BC-2.07.002 and
      BC-2.07.003. No content changes. (Note: the v1.0.1 patch in the Changelog
      already noted this fix; this entry records it in the frontmatter modified array.)
  - date: 2026-06-28
    version: "1.2"
    change: >
      Wave-5 mgmt-plane adversarial review Rulings 1/3/4/6/7 (ARCH-12 v1.2): PC-1
      amended with HandshakeTimeout=10s/RPCIdleTimeout=30s deadlines (Ruling 1); PC-3
      replaced with structural post-auth guard (Ruling 7); PC-7 amended to require
      ldflags-injected daemonVersion (Ruling 6); EC-001 updated with concrete deadline
      values; EC-004 updated to match post-auth guard framing; EC-012 added
      (connection-flood cap MaxConcurrentConnections=128, Ruling 3); EC-013 added
      (Unix socket 0600 permissions, Ruling 4); Invariant 7 added (socket permissions
      and 127.0.0.1 binding, Ruling 4); VP-065 property updated (Ruling 7).
  - date: 2026-06-29
    version: "1.3"
    change: >
      ARCH-12 v1.3 Wave-5 Convergence Rulings A–E: (A) Precondition 3 rewritten —
      ephemeral keypair for access daemon, NewServer nil/short-key panic at construction
      (fail-fast, Invariant 8 added); MVP caveat on daemon-identity pinning noted.
      (B) PC-10 added/consolidated — Serve returns nil on Shutdown/ctx-cancel via
      shuttingDown atomic.Bool; merged prior Shutdown-method item into PC-10.
      (C) PC-11 (E-RPC-010 unknown command, in-band) and PC-12 (E-RPC-011 handler
      error, in-band) added; connection not closed on either; E-RPC-002 forbidden.
      (D) EC-013 extended — console TCP loopback rejection: host must be
      127.0.0.1/[::1]/localhost; others abort via E-CFG-008 in buildMgmtListener
      (not config.Validate()). (E) PC-1 amended — conn.SetWriteDeadline before
      every sendJSON (HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC
      responses), cleared after each send; closes CWE-400 write-side slowloris.
      Ruling F: no BC change (client single-ctx-budget model is sanctioned).
  - date: 2026-06-29
    version: "1.4"
    change: >
      ARCH-12 v1.4 Wave-5 Convergence Rulings G/I/J/K: (G) PC-10 extended — unexpected
      listener close (ctx live, Shutdown never called) returns non-nil; the accept-error
      predicate `shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil)`
      is now cited in the PC to make the unexpected-close path explicit; VP-069 updated.
      (I) PC-10 drain-ordering guarantee appended: connections accepted after shuttingDown
      is set are dropped without entering connWG; trackConn called before connWG.Add and
      before goroutine spawn; closeAllConns force-closes all tracked connections before
      connWG.Wait. (J) EC-013 strengthened: ANY startMgmtServer failure (config-class or
      transient bind) aborts access daemon startup — no degraded-management mode.
      (K) EC-001 amended: HandshakeTimeout expiry (silent stall) causes close-only
      WITHOUT sending AUTH_FAIL; non-timeout decode errors (malformed JSON, oversized,
      EOF) still send AUTH_FAIL before close.
  - date: 2026-06-29
    version: "1.5"
    change: >
      ARCH-12 v1.5 Wave-5 Convergence Rulings O/P/R: (O) EC-013 extended — stale-socket
      restart resilience: listenUnixMgmt MUST check for and remove a ModeSocket-mode
      inode at the bind path before Bind (Lstat→Remove→Bind; non-socket paths left
      untouched; prevents EADDRINUSE restart DoS after non-graceful exit).
      (P) PC-10 fatal-accept-error drain extended: closeAllConns() MUST be called
      immediately before connWG.Wait() on the fatal-accept-error path (Accept returns
      non-transient error while ctx is live and Shutdown never called) so in-flight
      goroutines are force-closed and drain completes quickly; VP-069 updated to add
      TestServe_FatalAcceptErrorDrainsQuickly test obligation.
      (R) PC-6 amended — handler execution timeout: handler Fn functions are invoked
      with context.WithTimeout(ctx, RPCIdleTimeout); a blocking handler is cancelled
      after RPCIdleTimeout and the server responds with E-RPC-011; closes CWE-400
      goroutine-pin surface on handler dispatch path.
  - date: 2026-06-29
    version: "1.6"
    change: >
      S-W5.01 mgmt-server convergence architect rulings (pass-3): (Amendment 2b) VP Anchors
      section replaced — stale v1.0 text removed; full current listing of VP-064–VP-073
      anchored to BC-2.07.004 added (VP-067 correctly excluded; it traces to BC-2.07.002).
      (Amendment 2a) Ruling P "ALL paths" wording precision: scoping note added to PC-10
      fatal-accept-error drain clause — Shutdown-initiated and ctx-cancel paths are excluded
      from the closeAllConns()-before-connWG.Wait() mandate (those paths already invoke
      closeAllConns() via Shutdown's own sequencing); pass-1/pass-2 adversary confirmation
      noted; VP-069.md receives a parallel scoping annotation. Spec-wording correction only,
      no code change. (Amendment 1) PC-3 and EC-004 security-event-log deferral: Implementation
      Deferral note added to both — fail-closed connection control is fully implemented and
      tested (VP-065); log side-effect deferred pending daemon logging infrastructure (slog
      seam on mgmt.Server); follow-up story S-HRD.02 (daemon logging infrastructure) created
      as owning stub. Known Scope Gaps section added listing this single deferral with
      S-HRD.02 reference.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/architecture/ARCH-12-daemon-management-plane.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
traces_to: [CAP-024]
kos_anchors:
  - elem-single-binary-three-modes
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.07.004: Daemon Management Server Authenticates All Connections via Ed25519 Challenge-Response (Fail-Closed)

## Description

The `internal/mgmt` server is the daemon-side counterpart to the sbctl client
(BC-2.07.002). Every daemon mode (router, access, console, control) starts an
`mgmt.Server` on its management socket before accepting any connections. The server
performs the ADR-012 Ed25519 challenge-response handshake immediately on each new
connection. Any connection that fails to complete a valid handshake is rejected with
E-ADM-010 and closed — no RPC command is ever processed on an unauthenticated
connection. All socket reads, on both the handshake and RPC paths, are bounded by
`io.LimitReader(conn, MaxMessageBytes)` (64 KiB) to prevent CWE-400 resource
exhaustion on hostile connections.

## Preconditions

1. The daemon is starting up with a valid config (passes BC-2.09.003 validation).
2. The config either contains one or more `authorized_operator_keys` (PEM-encoded
   Ed25519 public keys), or has none (bootstrap mode: the daemon's own keypair is
   the sole authorized key).
3. The daemon's own Ed25519 private key is available — either an ephemeral key
   generated at startup via `ed25519.GenerateKey(rand.Reader)` (this wave: access
   daemon in S-W5.01) or a persistent key loaded from `key_file` (S-6.02). A nil
   or invalid key (`len != ed25519.PrivateKeySize` = 64 bytes) causes
   `mgmt.NewServer` to panic at construction time, preventing any connection from
   being accepted. **MVP security note:** An ephemeral key means the daemon identity
   (the `daemon_sig` in CHALLENGE) changes on every process restart; client-side
   daemon-identity pinning is deferred to S-6.02. This is an acceptable MVP
   limitation: the management plane is functional and fail-closed immediately.
4. A `net.Listener` has been opened on the management socket address (Unix socket
   path or TCP address per ARCH-05 §Daemon Management Socket and ARCH-12 §Wiring).
5. `mgmt.NewServer(ln, daemonKey, operatorKeySet, handlers, daemonVersion)` has been
   called with the listener, daemon private key, operator key set, registered command
   handlers, and the build-injected `daemonVersion` string (from `cmd/switchboard.version`
   via ldflags; `"dev"` is the unreleased-build sentinel only).
6. `mgmt.Server.Serve(ctx)` is running and waiting for connections.

## Postconditions

1. **Challenge issued immediately:** On every new connection, the server sends a
   CHALLENGE message as the first action before reading any client data:
   ```json
   {"type":"challenge","nonce":"<base64url 32 bytes>","daemon_sig":"<base64url sig>"}
   ```
   The nonce is 32 bytes from `crypto/rand.Read`. The `daemon_sig` is
   `ed25519.Sign(daemonPrivateKey, nonceBytes)`. This message is always the first
   data sent — the server never reads from a new connection before issuing a
   challenge. The server calls `conn.SetReadDeadline(time.Now().Add(mgmt.HandshakeTimeout))`
   (default 10 s) immediately after sending the CHALLENGE and before any blocking
   read. On deadline expiry the connection is closed with E-ADM-010.
   After AUTH_OK is sent, `conn.SetReadDeadline(time.Now().Add(mgmt.RPCIdleTimeout))`
   (default 30 s) is applied before reading the first RPC request. After a successful
   decode the deadline is reset to `time.Time{}` (no deadline) so writes are not
   inadvertently bounded.
   **Write deadline obligation (Ruling E / ARCH-12 §7):** Before every `sendJSON`
   call in `handleConnection`, the server sets
   `conn.SetWriteDeadline(time.Now().Add(d))` where `d` is `HandshakeTimeout`
   (10 s) for handshake-phase sends (CHALLENGE, AUTH_OK, AUTH_FAIL) and
   `RPCIdleTimeout` (30 s) for RPC response sends. The write deadline is cleared
   to `time.Time{}` after each send completes. This closes the symmetric
   slowloris-on-write slot-exhaustion vector (CWE-400) identified in ARCH-12 §7.

2. **Unauthenticated connections rejected:** If the client sends a CHALLENGE_RESPONSE
   where either (a) `ed25519.Verify(pubkey, nonceBytes, nonceSig)` returns false, or
   (b) `OperatorKeySet.IsAuthorized(pubkey)` returns false, the server sends:
   ```json
   {"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}
   ```
   and closes the connection immediately. No RPC commands are processed on this
   connection.

3. **Post-auth protocol violation rejected:** After a successful auth handshake
   on connection C, any further `"type":"challenge_response"` message on C is treated
   as a protocol violation. The server closes the connection with E-ADM-010 and logs
   a security event. The server maintains a per-connection `authenticated` boolean;
   no nonce set is stored after auth. Cross-connection replay is prevented by the
   fresh nonce issued on every new connection (`crypto/rand.Read(32)`).

   ### Implementation Deferral
   The security-event log side effect mandated here is DEFERRED pending daemon logging
   infrastructure (an slog seam on mgmt.Server). The fail-closed connection control —
   AUTH_FAIL + E-ADM-010 + close — is fully implemented and tested (VP-065). Owning
   follow-up story: S-HRD.02 (daemon logging infrastructure).

4. **Auth failure closes connection without processing RPCs:** After sending
   AUTH_FAIL, the server closes the connection. The client receives no RPC response
   data. There is no retry opportunity on the same connection.

5. **All RPC commands require prior auth:** No RPC request (`"type":"request"`)
   is dispatched to any registered handler unless the connection has passed a
   successful auth handshake in this session. A client that skips the handshake and
   sends an RPC request directly receives AUTH_FAIL + close.

6. **Bounded reads and handler execution timeout (CWE-400):** Every `json.Decoder.Decode`
   call on the management socket — CHALLENGE_RESPONSE, RPC request, and any other
   message — is preceded by `io.LimitReader(conn, MaxMessageBytes)` where
   `MaxMessageBytes = 1 << 16` (64 KiB, defined as `internal/mgmt.MaxMessageBytes`).
   A connection that sends a message exceeding 64 KiB causes the read to terminate
   with an error and the connection to be closed. The process does not OOM.
   **Handler execution timeout (Ruling R / CWE-400):** Registered handler `Fn`
   functions are invoked with a child context derived via
   `context.WithTimeout(ctx, RPCIdleTimeout)`. A handler that does not return within
   `RPCIdleTimeout` (30 s) is cancelled via the child context; the server responds
   with E-RPC-011 (`"handler context deadline exceeded"` or the cancellation error
   message). The connection is NOT closed on handler timeout. This bounds handler
   execution to the same time budget as the RPC-phase read deadline, closing the
   CWE-400 goroutine-pin surface on the handler dispatch path — a blocking handler
   cannot permanently pin a connection goroutine and semaphore slot.

7. **Successful authentication path:** If both `ed25519.Verify` and `IsAuthorized`
   return true, the server sends AUTH_OK:
   ```json
   {"type":"auth_ok","daemon_version":"<semver>"}
   ```
   and the connection enters the authenticated state. The `"daemon_version"` field in
   the AUTH_OK message MUST equal the value of `cmd/switchboard.version` (injected by
   ldflags at build time). The sentinel value `"dev"` is used only for
   untagged/unreleased builds. Hardcoding `"dev"` in production is a defect.
   Subsequent RPC requests are dispatched to the registered handler for the command
   name. Responses are wrapped in the standard JSON envelope from
   interface-definitions.md §JSON Output Schema.

8. **Constant-time key comparison:** `OperatorKeySet.IsAuthorized` uses constant-time
   comparison (`subtle.ConstantTimeCompare` or equivalent) to prevent timing oracle
   attacks on key enumeration. Recognized and unrecognized keys receive the same
   E-ADM-010 response — no oracle differentiating "key known but wrong signature"
   from "key not in set."

9. **Bootstrap mode:** When `authorized_operator_keys` is empty in config, the daemon
   accepts connections signed by the daemon's own keypair (the `key_file` key from
   config). The handshake and rejection behavior are identical — only the authorized
   set changes.

10. **PC-10 — Serve returns nil on intentional shutdown; non-nil on unexpected failure (Rulings B/G/I):**
    `mgmt.Server.Serve` returns `nil` when `Shutdown` is called or the context
    is cancelled (normal daemon lifecycle). The observable postcondition a test
    asserts is: start the server, call `Shutdown(ctx)` or cancel the context,
    receive `nil` from `Serve`. `Serve` returns a non-nil error only on an
    unexpected listener failure unrelated to shutdown (e.g., the underlying fd
    was externally closed while `ctx` is still alive and `Shutdown` was never
    called). The distinction is mediated by a `shuttingDown atomic.Bool` field
    on the server: `Shutdown` and the ctx-watcher goroutine each set it to
    `true` before calling `s.ln.Close()`. The Accept error path applies the
    canonical predicate:
    ```go
    if s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil) {
        // intentional shutdown — return nil
    }
    // unexpected listener close — return err (non-nil)
    ```
    Both conjuncts of the `errors.Is` arm are required: `ctx.Err() != nil` guards
    the edge case where the fd is externally closed while the context is live
    (the `shuttingDown` flag is false and `ctx` is not cancelled — this MUST return
    non-nil). `mgmt.Server.Shutdown(ctx)` also drains in-flight connections and
    closes the listener within the context deadline; no new connections are accepted
    after shutdown is initiated; the goroutine is WaitGroup-tracked per ARCH-01
    §Goroutine WaitGroup Contract.
    **Drain-ordering guarantee (Ruling I):** Connections accepted after `Shutdown`
    sets `shuttingDown = true` are closed immediately without entering `connWG` —
    the accept loop checks `shuttingDown.Load()` after each successful `Accept()`
    and drops the connection before calling `connWG.Add`. No Add-after-Done-zero
    panic is possible. Connections already in-flight are registered in the `conns`
    set (via `trackConn`) BEFORE `connWG.Add(1)` and BEFORE the goroutine is spawned,
    so `closeAllConns` always captures every in-flight connection for force-close.
    `connWG.Wait` is called once (by `Shutdown`); no other caller calls `Wait`.
    Drain completes within the caller's shutdown budget because `closeAllConns`
    force-closes all tracked connections before `connWG.Wait` blocks.
    **Fatal-accept-error drain (Ruling P):** On the fatal-accept-error path (Accept
    returns a non-transient error while `ctx` is still live and `Shutdown` was never
    called — i.e., the predicate `shuttingDown.Load() || (errors.Is(err, net.ErrClosed)
    && ctx.Err() != nil)` is false), `s.closeAllConns()` is called UNCONDITIONALLY
    immediately before `s.connWG.Wait()` before returning the error. This ensures
    in-flight connection goroutines are force-closed even on the unexpected-close
    path and drain completes quickly (not blocked up to `RPCIdleTimeout` = 30 s).
    `closeAllConns()` is idempotent — calling it multiple times is safe (double-close
    on a `net.Conn` returns `net.ErrClosed`, which is ignored).
    **Scope of Ruling P:** `closeAllConns()` MUST precede `connWG.Wait()` on the
    fatal-accept-error return path (ctx live, Shutdown never called) — this is the gap
    Ruling P closes. The Shutdown-initiated drain path and the ctx-cancel path already
    invoke `closeAllConns()` via `Shutdown`'s own sequencing and are NOT subject to
    this requirement (pass-1 and pass-2 adversaries independently confirmed the
    ctx-cancel arms are a benign ordering condition, not a hang; `go test -race`
    passes). This is a spec-wording precision, no code change.

11. **PC-11 — Unknown RPC command returns E-RPC-010 in-band (Ruling C):**
    When an authenticated RPC request names a command that is not registered in
    the handler slice, the server responds with:
    ```json
    {"type":"response","id":"<id>","ok":false,"error":{"code":"E-RPC-010","message":"unknown command: <cmd>"},"data":null}
    ```
    The connection is NOT closed. The client may send further RPC requests on
    the same connection. This code is server-side only; it is distinct from the
    client-side `E-RPC-001` emitted by sbctl. The undefined `E-RPC-002` MUST NOT
    appear anywhere in `internal/mgmt` — any such reference is a defect.

12. **PC-12 — Handler execution error returns E-RPC-011 in-band (Ruling C):**
    When a registered handler's `Fn` returns a non-nil error, the server responds
    with:
    ```json
    {"type":"response","id":"<id>","ok":false,"error":{"code":"E-RPC-011","message":"<err>"},"data":null}
    ```
    where `<err>` is the handler's error string verbatim (not wrapped further by
    `internal/mgmt`). The connection is NOT closed. This code is server-side only;
    it is distinct from `E-RPC-001` (client) and `E-RPC-010` (server unknown
    command).

## Invariants

1. No unauthenticated connection ever receives an RPC response — the auth check is
   fail-closed: the default outcome is rejection.
2. The operator private key never transits the socket in any direction (DI-002). Only
   the public key (32 bytes) is sent by the client in the CHALLENGE_RESPONSE.
3. All socket reads are bounded by MaxMessageBytes (64 KiB). There is no
   unbounded-read code path in `internal/mgmt`.
4. The nonce is always fresh per connection (`crypto/rand.Read(32)`). The server
   never reuses a nonce across connections.
5. AUTH_FAIL responses are identical regardless of whether the key was recognized —
   no timing or content oracle.
6. The management auth domain is orthogonal to SVTN node admission (ARCH-04): an
   admitted node key does not imply management authority; an operator key does not
   imply SVTN admission. The two key sets are independently maintained.
7. The Unix management socket has permissions 0600 at creation time. The process
   umask is set to 0177 immediately before `net.Listen` and restored afterward.
   Console TCP is bound to 127.0.0.1; no management listener binds to `0.0.0.0`.
   There is no TOCTOU window between socket creation and permission assignment —
   the umask-before-Listen pattern is atomic.
8. **Daemon key size guard (Ruling A):** `mgmt.NewServer` MUST panic at
   construction time if `len(daemonKey) != ed25519.PrivateKeySize` (64 bytes).
   A nil key has `len == 0` and also triggers the panic. This fail-fast guard
   ensures a nil or truncated key can never reach a connection goroutine where it
   would cause a nil-pointer dereference (remote panic DoS, ARCH-12 Ruling A.2).
   The check mirrors the existing `daemonVersion` emptiness panic in `NewServer`.

## Trigger

A client connects to the daemon's management socket (Unix or TCP per ARCH-05
§Daemon Management Socket). The server handles each connection in a dedicated
goroutine tracked by the server's internal WaitGroup.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Client connects and sends nothing (HandshakeTimeout silent stall) | Server sends CHALLENGE, applies `HandshakeTimeout` read deadline (default 10 s) on the subsequent read. On timeout expiry: the connection is closed immediately **without sending `AUTH_FAIL`** — a non-responsive client would not read it, and the extra write delays slot reclamation and risks a slowloris-on-write. Non-timeout decode errors (malformed JSON, oversized message, EOF before timeout) **do** send `AUTH_FAIL` before closing. The server does not hang indefinitely in either case. `mgmt.HandshakeTimeout = 10 * time.Second`. |
| EC-002 | Client sends CHALLENGE_RESPONSE with an unrecognized public key | `OperatorKeySet.IsAuthorized` returns false. Server sends AUTH_FAIL (E-ADM-010, same message as wrong-signature). Connection closed. No oracle. |
| EC-003 | Client sends CHALLENGE_RESPONSE with recognized key but wrong signature | `ed25519.Verify` returns false. Server sends AUTH_FAIL (E-ADM-010). Connection closed. No oracle. |
| EC-004 | Client completes a successful auth handshake and then sends a second `"type":"challenge_response"` message on the same connection | Structural post-auth guard triggers: server closes the connection with E-ADM-010 and logs a security event. The `authenticated` boolean is already true; no nonce set is consulted. The connection enters no further authenticated state (it is closed). **Implementation Deferral:** The security-event log side effect mandated here is DEFERRED pending daemon logging infrastructure (an slog seam on mgmt.Server). The fail-closed connection control — AUTH_FAIL + E-ADM-010 + close — is fully implemented and tested (VP-065). Owning follow-up story: S-HRD.02 (daemon logging infrastructure). |
| EC-005 | Client sends a message > 64 KiB (oversized) | `io.LimitReader` causes `json.Decoder.Decode` to return an error. Server closes connection. No memory allocation beyond 64 KiB for this connection. Process does not OOM. |
| EC-006 | Client connects and sends malformed JSON (not valid JSON object) | `json.Decoder.Decode` returns error. Server closes connection with no response (or sends AUTH_FAIL depending on which decode fails). Process does not panic. |
| EC-007 | Client sends a JSON object of the right size but wrong `"type"` field | Server treats this as a protocol error; closes connection. No RPC dispatched. |
| EC-008 | Client closes connection mid-handshake (after CHALLENGE, before CHALLENGE_RESPONSE) | Server detects EOF/read error; cleans up connection state; no goroutine leak. |
| EC-009 | Non-Switchboard peer sends an arbitrary byte stream | `io.LimitReader` + `json.Decoder` returns error within first 64 KiB. Connection closed cleanly; no panic, no OOM. |
| EC-010 | `authorized_operator_keys` is empty (bootstrap mode) | Daemon's own keypair is the authorized key. Handshake proceeds normally. AUTH_OK on correct daemon-key signature. |
| EC-011 | Client skips handshake and sends an RPC request (`"type":"request"`) as first message | Server has not yet received CHALLENGE_RESPONSE; treats this as an unauthenticated request. Sends AUTH_FAIL + close. RPC handler is never called. |
| EC-012 | Concurrent connections exceed MaxConcurrentConnections (default 128) | New `Accept()` calls block in the accept-loop semaphore until a slot frees. Connections queue in the OS accept backlog. No new goroutines are spawned beyond the limit. No fd exhaustion or goroutine leak. Transient Accept errors (EMFILE etc.) trigger exponential backoff (5 ms–1 s) rather than server shutdown. `MaxConcurrentConnections = 128` (defined in `internal/mgmt`). |
| EC-013 | Unix socket created with world-readable permissions; or console-mode TCP bound to non-loopback address; or startMgmtServer fails for any reason; or stale socket inode exists at path from prior non-graceful exit | **Unix socket:** The management socket MUST be created with permissions 0600 (owner read/write only) via `syscall.Umask(0177)` before `net.Listen`. A world-accessible socket allows any local user to connect and attempt authentication — the permission is the first defense before auth. **Console TCP loopback rejection (Ruling D):** For console-mode TCP, any `management_socket` value whose host is not `127.0.0.1`, `[::1]`, or `localhost` causes daemon startup to abort with E-CFG-008: `"E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: <address>"`. Rejected hosts include: `0.0.0.0` (IPv4 unspecified), `::` (IPv6 unspecified), bare port (`:9091`, empty host), and any non-loopback IP. This check is enforced in `buildMgmtListener` (`cmd/switchboard/mgmt_wire.go`), NOT in `config.Validate()` (which has no mode parameter and cannot distinguish console mode from other modes). **Fatal startup policy (Ruling J):** Any `startMgmtServer` failure — whether config-class (E-CFG-008 non-loopback, E-CFG-009 malformed operator key, `NewServer` construction panic) OR transient bind failure (EADDRINUSE, permission denied on socket path) — ABORTS access daemon startup immediately. The data plane is never entered. There is no non-fatal / degraded-management mode in the access daemon (this wave). The daemon exits non-zero. Running an access daemon with no management plane is worse than not running it; the management plane is the operator's sole control channel. **Stale-socket restart resilience (Ruling O):** `listenUnixMgmt` MUST perform a pre-bind cleanup check: if `os.Lstat(path)` succeeds and the file's mode has `os.ModeSocket` set, `os.Remove(path)` is called before `syscall.Bind`. This prevents EADDRINUSE on daemon restart after a non-graceful exit (SIGKILL, crash, OOM kill) where the socket inode persists on the filesystem. Only `ModeSocket`-mode inodes are removed — regular files, directories, or device nodes at the socket path are left untouched and cause `Bind` to fail with the original error (not silently clobbered). `os.Remove` errors are silently ignored: if removal fails, the subsequent `Bind` either succeeds (file gone) or fails with `EADDRINUSE` (file still present), both propagating correctly. The Lstat→Remove→Bind TOCTOU window is accepted: it repairs exactly the restart-after-crash case and is no worse than the current EADDRINUSE failure. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Valid operator key signs challenge nonce correctly | AUTH_OK `{"type":"auth_ok","daemon_version":"..."}` | happy-path |
| Valid operator key, wrong signature (nonce bytes tampered) | AUTH_FAIL `{"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}`; connection closed | error |
| Unrecognized public key, valid signature format | AUTH_FAIL E-ADM-010; connection closed | error |
| Recognized key, correct signature, then RPC `{"type":"request","id":"r1","command":"router.status","args":{}}` | AUTH_OK then `{"type":"response","id":"r1","ok":true,"error":null,"data":{...}}` | happy-path |
| Message of 65537 bytes (> 64 KiB MaxMessageBytes) sent as CHALLENGE_RESPONSE | Connection closed; error returned from Decode; no OOM | error (CWE-400) |
| Malformed JSON: `{"type":"challenge_respon` (truncated) | Connection closed; no panic | error |
| Client disconnects after receiving CHALLENGE, sends nothing | Connection cleaned up; no goroutine leak | edge-case |
| Bootstrap mode: no authorized_operator_keys; daemon's own key signs nonce | AUTH_OK | happy-path (bootstrap) |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-064 | Server rejects unauthenticated connections (no CHALLENGE_RESPONSE or wrong key/sig) → AUTH_FAIL + close, no RPC | integration |
| VP-065 | Post-auth structural guard: after AUTH_OK, a subsequent `"type":"challenge_response"` message causes connection close with E-ADM-010 | integration |
| VP-066 | Server enforces bounded read: message > 64 KiB → error + close, no OOM (CWE-400) | unit + fuzz |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 |
| L2 Domain Invariants | DI-002 (private keys never transit — operator private key stays in sbctl, only the public key transits in CHALLENGE_RESPONSE) |
| Architecture Module | internal/mgmt |
| ADR | ADR-012 (Management-Auth Wire Protocol) per ARCH-12-daemon-management-plane.md §ADR-012 |
| Stories | S-W5.01 (implementing_story — confirm with story-writer) |
| Capability Anchor Justification | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 — this BC specifies the daemon-side authentication counterpart that makes the unified CLI contract (CAP-024) work end-to-end: sbctl cannot authenticate without a server that enforces the same Ed25519 challenge-response handshake |

## Related BCs

- BC-2.07.002 — composes with: client-side sbctl authentication (PC-2/PC-3) requires this server-side contract as its counterpart; both BCs together define the full ADR-012 handshake
- BC-2.07.003 — composes with: connection error handling applies before this BC's auth handshake begins
- BC-2.09.003 — depends on: management_socket and authorized_operator_keys config fields are validated per BC-2.09.003 before this BC's preconditions are met

## Architecture Anchors

- ARCH-12-daemon-management-plane.md §ADR-012 (Management-Auth Wire Protocol) — authoritative wire protocol definition
- ARCH-12-daemon-management-plane.md §internal/mgmt Package Design — exported API surface
- ARCH-12-daemon-management-plane.md §Wiring into cmd/switchboard — daemon startup sequence
- ARCH-05-cli-and-api.md §Daemon Management Socket — socket paths per daemon mode
- ARCH-04-admission-security.md §Tier 1 Admission Protocol — prior art; ADR-012 is explicitly NOT calling admission.AdmitNode (independent auth domain)

## Story Anchor

S-W5.01 — implementing story (confirm with story-writer; recommended by ARCH-12 §Story Decomposition)

## Known Scope Gaps

| Gap | Deferred Behavior | Status | Follow-up |
|-----|-------------------|--------|-----------|
| Security-event log on post-auth protocol violation (PC-3, EC-004) | The fail-closed connection control (AUTH_FAIL + E-ADM-010 + close) is fully implemented and tested (VP-065). The security-event log side effect is DEFERRED pending daemon logging infrastructure (an slog seam on mgmt.Server). | Deferred | S-HRD.02 (daemon logging infrastructure) |

## VP Anchors

All VPs currently anchored to BC-2.07.004 (source of truth: VP-INDEX.md):

| VP | Contract Anchor | Proof Method |
|----|----------------|--------------|
| VP-064 | PC-2/PC-5 — rejects unauthenticated connections (no CHALLENGE_RESPONSE, wrong key, bad signature) → AUTH_FAIL + close; no RPC dispatched | integration |
| VP-065 | PC-3 — post-auth structural guard: after AUTH_OK, a subsequent `"type":"challenge_response"` message closes connection with E-ADM-010 | integration |
| VP-066 | PC-6 — bounded read: message > 64 KiB → error + close, no OOM (CWE-400) | unit+fuzz |
| VP-068 | Invariant 8 — `mgmt.NewServer` panics at construction if `len(daemonKey) != ed25519.PrivateKeySize` | unit |
| VP-069 | PC-10 — `Serve` returns nil on Shutdown/ctx-cancel; non-nil on unexpected listener failure (fd externally closed, ctx live, Shutdown never called); fatal-accept-error drain ordering | integration |
| VP-070 | PC-11 — E-RPC-010 unknown command in-band, connection NOT closed | integration |
| VP-071 | PC-12 — E-RPC-011 handler error in-band (verbatim message), connection NOT closed | integration |
| VP-072 | PC-1 — write deadline set before every `sendJSON` (HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC responses); cleared after each send; closes CWE-400 write-side slowloris | integration |
| VP-073 | EC-013 — console-mode TCP non-loopback bind aborts startup with E-CFG-008 (`buildMgmtListener`) | integration |

Note: VP-067 (`sbctl.Authenticate()` fail-closed) is anchored to BC-2.07.002, not BC-2.07.004.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.6 | 2026-06-29 | S-W5.01 mgmt-server convergence architect rulings (pass-3): (2b) VP Anchors section replaced — stale v1.0 text removed; full current listing of VP-064–VP-073 anchored to BC-2.07.004 (VP-067 excluded; traces to BC-2.07.002). (2a) Ruling P "ALL paths" wording precision: scoping note added to PC-10 fatal-accept-error drain — Shutdown-initiated and ctx-cancel paths excluded (closeAllConns() already called via Shutdown's own sequencing); pass-1/pass-2 adversary confirmation cited; VP-069.md receives parallel scoping annotation; spec-wording correction, no code change. (1) PC-3 and EC-004 security-event-log deferral: Implementation Deferral note added to both — fail-closed connection control fully implemented and tested (VP-065); log side-effect deferred pending slog seam on mgmt.Server; Known Scope Gaps section added with S-HRD.02 follow-up reference. |
| 1.5 | 2026-06-29 | ARCH-12 v1.5 Wave-5 Convergence Rulings O/P/R: (O) EC-013 extended — stale-socket restart resilience: `listenUnixMgmt` MUST perform a pre-bind `os.Lstat` + `os.Remove` for `ModeSocket` inodes, preventing EADDRINUSE restart DoS after non-graceful exit; TOCTOU window accepted; non-socket inodes untouched. (P) PC-10 extended with fatal-accept-error drain clause: `closeAllConns()` MUST precede `connWG.Wait()` on ALL `Serve` return paths including the fatal-accept-error path (ctx live, Shutdown never called), so in-flight goroutines are force-closed and drain completes within budget; VP-069 updated. (R) PC-6 amended — handler execution timeout: all handler `Fn` invocations are wrapped in `context.WithTimeout(ctx, RPCIdleTimeout)`; blocking handlers cancelled at 30 s; server responds E-RPC-011; connection not closed; closes CWE-400 goroutine-pin surface on handler dispatch path. |
| 1.4 | 2026-06-29 | ARCH-12 v1.4 Wave-5 Convergence Rulings G/I/J/K: (G) PC-10 extended — unexpected listener close (ctx live, Shutdown never called) explicitly required to return non-nil; canonical accept-error predicate `shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil)` now quoted in PC-10 body; VP-069 updated to cover unexpected-close → non-nil path. (I) PC-10 drain-ordering guarantee appended — connections accepted after `shuttingDown` is set are dropped without entering `connWG`; `trackConn` called before `connWG.Add` and before goroutine spawn; `closeAllConns` force-closes all tracked connections before `connWG.Wait`; single `Wait` owner. (J) EC-013 strengthened — any `startMgmtServer` failure (config-class E-CFG-008/E-CFG-009, NewServer construction panic, transient bind) ABORTS access daemon startup; data plane never entered; no degraded-management mode. (K) EC-001 amended — HandshakeTimeout expiry (silent stall) is now close-only WITHOUT `AUTH_FAIL`; non-timeout decode errors (malformed JSON, oversized, EOF) still send `AUTH_FAIL` before close. |
| 1.3 | 2026-06-29 | ARCH-12 v1.3 Wave-5 Convergence Rulings A–E: (A) Precondition 3 rewritten — access daemon uses ephemeral `ed25519.GenerateKey(rand.Reader)` keypair; `mgmt.NewServer` panics if `len(daemonKey) != ed25519.PrivateKeySize`; MVP caveat on key pinning deferred to S-6.02; Invariant 8 added for key-size fail-fast guard. (B) PC-10 added/consolidated — `Serve` returns nil on `Shutdown`/ctx-cancel via `shuttingDown atomic.Bool`; non-nil only on unexpected fatal Accept error; merged prior item-10 Shutdown-method behavior into PC-10 (Ruling B). (C) PC-11 and PC-12 added — server-side E-RPC-010 (unknown command, in-band, connection not closed) and E-RPC-011 (handler error, in-band, connection not closed); E-RPC-002 forbidden (Ruling C). (D) EC-013 extended with console TCP loopback rejection predicate — host must be 127.0.0.1/[::1]/localhost; others abort with E-CFG-008 in `buildMgmtListener`, not `config.Validate()` (Ruling D). (E) PC-1 amended with write deadline obligation — `conn.SetWriteDeadline` before every `sendJSON` (HandshakeTimeout for CHALLENGE/AUTH_OK/AUTH_FAIL, RPCIdleTimeout for RPC responses), cleared after each send; closes CWE-400 symmetric write-side slot-exhaustion vector (Ruling E / ARCH-12 §7). |
| 1.2 | 2026-06-28 | Wave-5 mgmt-plane adversarial review Rulings 1/3/4/6/7 (ARCH-12 v1.2): PC-1 amended with concrete HandshakeTimeout (10 s) and RPCIdleTimeout (30 s) deadline requirements (Ruling 1); PC-3 replaced with structural post-auth guard — `authenticated` boolean, no nonce set (Ruling 7); PC-5 `NewServer` signature updated to include `daemonVersion` param; PC-7 amended to require ldflags-injected `cmd/switchboard.version`, `"dev"` sentinel only for unreleased builds (Ruling 6); EC-001 updated with concrete timeout values; EC-004 updated to post-auth `challenge_response` guard framing; EC-012 added (connection-flood → MaxConcurrentConnections=128, exponential backoff on EMFILE, Ruling 3); EC-013 added (Unix socket must be 0600 via umask(0177), console TCP 127.0.0.1-only, Ruling 4); Invariant 7 added (socket 0600, no 0.0.0.0 binding); VP-065 property reworded to structural assertion (Ruling 7). |
| 1.1 | 2026-06-28 | Wave-5 consistency audit F-004: subsystem frontmatter corrected from `SS-07` (ID form) to `network-management` (canonical name) to match sibling BCs BC-2.07.002 and BC-2.07.003. `modified:` array populated. No content changes. |
| 1.0 | 2026-06-28 | Initial draft — daemon-side management auth (ADR-012 server counterpart to BC-2.07.002). Wave-5 BC. |
| 1.0.1 | 2026-06-28 | Patch — subsystem back-filled to SS-07 (network-management); internal/mgmt was already listed under SS-07 in ARCH-INDEX Subsystem Registry. No content changes. |
