---
artifact_id: ARCH-12-daemon-management-plane
document_type: architecture-section
level: L3
version: "1.6"
status: draft
producer: architect
timestamp: 2026-06-28T00:00:00
phase: wave-5
traces_to: ARCH-INDEX.md
inputDocuments:
  - '.factory/specs/prd.md'
  - '.factory/specs/prd-supplements/interface-definitions.md'
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md'
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.003.md'
  - '.factory/specs/architecture/ARCH-04-admission-security.md'
  - '.factory/specs/architecture/ARCH-05-cli-and-api.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'
kos_anchors:
  - elem-single-binary-three-modes
  - elem-ssh-end-to-end-encryption
modified:
  - 2026-06-28T00:00:00
  - 2026-06-28T00:00:00 # v1.1 — F-008: updated VP planning section to final VP-064–067 assignments; corrected totals (45 BCs, 67 VPs)
  - 2026-06-28T00:00:00 # v1.2 — adversarial review rulings: (1) concrete read deadlines added to ADR-012; (2) Authenticate() context-first signature ruled; (3) connection-cap semaphore added; (4) Unix socket permissions 0600 required; (5) E-NET-001 overload resolved with E-CFG-010/E-RPC-001; (6) daemonVersion param added to NewServer; (7) PC-3 replay-nonce resolved as structural post-auth guard
  - 2026-06-29T00:00:00 # v1.3 — Wave-5 convergence round-2 rulings A–F: (A) ephemeral daemon keypair for access daemon; NewServer nil-key guard; (B) Serve graceful-shutdown returns nil via shuttingDown atomic; (C) server-side RPC error codes E-RPC-010/E-RPC-011 distinct from client E-RPC-001; (D) console TCP loopback validation placement in buildMgmtListener; (E) write deadline on server sends; (F) client single-ctx-timeout model clarified vs. server per-RPC re-arm
  - 2026-06-29T00:00:00 # v1.4 — Wave-5 convergence round-3 rulings G–N: (G) Serve accept-error predicate: && ctx.Err() != nil conjunct missing; unexpected-close must return non-nil; new VP/test obligation; (H) dead case <-done: branches: bless as-built ctx-cancel design via done-channel ctx-watcher; amend Ruling B; (I) connWG.Add/Wait race + track-after-spawn drain gap; ordering fix + new VP/test obligation; (J) mgmt-server start failure policy: config-class errors fatal; (K) EC-001 silent-stall: amend BC to close-only for timeout path; (L) E-CFG-008 canonical message string: single placement ruling; (M) RPC wire-type mismatch: "request"/"response" canonical; new S-6.03 AC; (N) vp_traces forward-link gap in S-W5.01
  - 2026-06-30T00:00:00 # v1.6 — S-W5.02 Pass-1 fix-burst: stale §Story 3 prose replaced ("e2e VP deferred") with "e2e VP: VP-049 (pre-assigned via VP-INDEX v2.6)"; Story 3 summary table row updated to VP-049 and S-6.06 added to depends column; dep-graph v1.5 S-W5.02 row reflects S-6.06 dependency (F-P1L3-003)
  - 2026-06-29T00:00:00 # v1.5 — Wave-5 convergence round-4 rulings O–W: (O) stale Unix socket pre-bind unlink; fix-now in listenUnixMgmt; (P) fatal-accept-error drain gap: closeAllConns missing before connWG.Wait on unexpected-close path; fix-now; (Q) AC-001/EC-001 story propagation gap for Ruling K; story-writer fix only; (R) per-handler execution timeout: fix-now with context.WithTimeout wrapper; (S) version sentinel CI gate: follow-up story targeting CI epic; (T) TestServe_ShutdownWindowNoAddAfterWaitPanic non-discriminating: test-writer fix only; (U) dispatch() response type never validated: fix-now, add rejection + test; (V) dispatch() missing read deadline: fix-now, reconcile ARCH-12 residual-concern note, amend BC-2.07.003; (W-test) t.Fatalf from goroutines in client_test.go: test-writer fix only; (X) hardcoded id "1" + no resp.ID echo check: fix-now; (Y) Inv-2 vs Inv-4 spec conflict on auth-handshake-timeout: fix-now, amend BC-2.07.003 Inv-4
---

# ARCH-12: Daemon Management Plane

> **Context:** The management plane is the server-side counterpart that sbctl connects
> to. Before Wave 5, ADR-006 described the protocol conceptually but no server
> implementation existed. This document specifies the full system: ADR-012 wire protocol,
> `internal/mgmt` package design, wiring into each daemon mode, sbctl client auth, and
> recommended BC/VP/story decomposition.

## Problem Statement

`sbctl` (cmd/sbctl) is a client with nothing to talk to. No daemon-side management
server exists. ADR-006 describes JSON-over-Unix-socket with SSH signature auth but
leaves the auth wire protocol unspecified — the operator-CLI challenge-response
handshake has never been defined. This ARCH section closes that gap.

Four concrete gaps are resolved here:

1. **No management server.** The daemon modes start no listener for sbctl to connect to.
2. **Unspecified auth wire protocol.** ADR-006 says "same mechanism as SVTN admission"
   without specifying the management-channel message sequence.
3. **No operator key set.** Node admission uses SVTNs; operator auth needs its own key
   store with a defined source.
4. **No `internal/mgmt` package.** No shared RPC dispatch infrastructure exists.

---

## ADR-012: Management-Auth Wire Protocol

> **Decision:** A dedicated Ed25519 challenge-response handshake over the management
> socket, modeled on the Tier 1 node admission protocol (ARCH-04) but using an
> independent operator key set drawn from the daemon's config. Framing is
> newline-delimited JSON (NDJSON). Socket reads are bounded by io.LimitReader to
> prevent CWE-400 OOM on hostile connections.

### 1. Framing Selection: Newline-Delimited JSON (NDJSON)

Each message is a single JSON object followed by a `\n` byte.

**Rationale over alternatives:**
- **Length-prefixed binary framing:** more complex to implement and debug; no benefit
  on a Unix-socket or loopback-TCP management channel where framing errors signal a
  protocol mismatch, not data loss.
- **HTTP/1.1 with Content-Length:** heavyweight; pulls in an HTTP server dependency;
  inconsistent with the JSON-only sbctl output model.
- **NDJSON:** matches the existing JSON envelope from interface-definitions.md; readable
  with `nc`/`socat` for debugging; zero-dependency parsing with `json.NewDecoder`;
  fits the request/response (not streaming) RPC model perfectly.

The NDJSON decoder is always wrapped with `io.LimitReader(conn, maxMsgBytes)` where
`maxMsgBytes = 1 << 16` (64 KiB). This bounds memory used per connection regardless
of what a hostile peer sends (CWE-400; surfaced in S-6.03 adversarial review as a
read-loop risk on the client side — this policy extends it symmetrically to the server
side). The constant is defined in `internal/mgmt` as `MaxMessageBytes`.

### 2. Operator Key Set Source

The management plane authenticates against a **dedicated operator key set** that is
distinct from the SVTN admitted node key set. Concretely:

- The daemon config carries one or more authorized operator public keys as
  PEM-encoded Ed25519 public keys (new `authorized_operator_keys` config field,
  discussed in §Config Additions below).
- These keys are loaded at daemon startup and held in an
  `internal/mgmt.OperatorKeySet` (a simple `[]ed25519.PublicKey` with an
  `IsAuthorized(pubkey)` method).
- The separation from the SVTN AdmittedKeySet is intentional: an SVTN node key
  (RoleControl/RoleConsole/RoleAccess per ADR-004) grants network admission; an
  operator key grants management-plane access. These are orthogonal authorization
  domains. An access-node key must not implicitly grant `sbctl` management authority
  on any daemon.
- **Bootstrap case:** if no `authorized_operator_keys` are configured, the daemon
  accepts operator connections signed by the daemon's own keypair (i.e., `key_file`
  from config). This allows a single-operator deployment to work without an extra key
  file, while still requiring a signed handshake.

### 3. Management-Auth Message Sequence

```
(1) Client connects to management socket (Unix or TCP per ARCH-05 §Daemon Management Socket).

(2) Server → Client: CHALLENGE message
    {
      "type": "challenge",
      "nonce": "<base64url-encoded 32 random bytes>",
      "daemon_sig": "<base64url-encoded Ed25519 signature of nonce by daemon's own key>"
    }

    - nonce: crypto/rand.Read(32), re-encoded as base64url (no padding).
    - daemon_sig: Sign(daemon_private_key, nonce_bytes). Prevents nonce forgery
      by a MITM (same rationale as ARCH-04 §Tier 1 Admission Protocol step 2).
    - The daemon MUST send this message immediately on connection, before reading
      any client data. This prevents the server from being driven by client-first
      attack traffic (CWE-400 variant: client-driven state exhaustion).

(3) Client → Server: CHALLENGE_RESPONSE message
    {
      "type": "challenge_response",
      "nonce_sig": "<base64url-encoded Ed25519 signature of nonce_bytes by operator's private key>",
      "pubkey": "<base64url-encoded operator Ed25519 public key (32 bytes)>"
    }

    - nonce_sig: Sign(operator_private_key, nonce_bytes).
    - pubkey: the 32-byte Ed25519 public key (not a fingerprint; full key).
    - The private key NEVER leaves the client (DI-002 preserved).

(4) Server: verify nonce_sig against pubkey, and pubkey against OperatorKeySet.
    - ed25519.Verify(pubkey, nonce_bytes, nonce_sig) must return true.
    - OperatorKeySet.IsAuthorized(pubkey) must return true.
    - The nonce MUST be recorded in the server's per-connection nonce set and
      MUST NOT be accepted again within the connection lifetime (connection-scoped
      replay prevention — nonces are not shared across connections, but a single
      connection cannot replay the same challenge twice if renegotiation were added
      in future).

(5a) Server → Client on SUCCESS: AUTH_OK
    {
      "type": "auth_ok",
      "daemon_version": "<semver string>"
    }

(5b) Server → Client on FAILURE: AUTH_FAIL + close connection
    {
      "type": "auth_fail",
      "code": "E-ADM-010",
      "message": "authentication failed"
    }
    The connection is closed immediately after sending AUTH_FAIL (no retry on the
    same connection). Callers who present an unrecognized key receive the same
    E-ADM-010 message regardless of whether the key was recognized — no oracle.

(6) Authenticated RPC: Client sends request; server responds; both use the common
    JSON envelope from interface-definitions.md §JSON Output Schema.
    {
      "type": "request",
      "id": "<client-generated opaque string, echoed in response>",
      "command": "<subcommand name>",
      "args": { ... }
    }
    Server responds:
    {
      "type": "response",
      "id": "<echoed>",
      "ok": true|false,
      "error": null | { "code": "E-xxx", "message": "..." },
      "data": { ... }
    }

(7) Client closes connection after receiving the final response.
    Each sbctl invocation opens one connection, authenticates, executes one RPC, closes.
```

### 4. Reuse vs. Differences from Tier 1 Node Admission (ARCH-04)

| Aspect | Tier 1 Node Admission (ARCH-04) | Management Auth (ADR-012) |
|--------|--------------------------------|--------------------------|
| Purpose | A Switchboard node joins an SVTN | An operator authenticates sbctl to a daemon |
| Key set | AdmittedKeySet (per SVTN) | OperatorKeySet (per daemon, from config) |
| Nonce generation | `admission.GenerateChallenge()` (reused) | `mgmt.GenerateChallenge()` (new, same pattern) |
| Nonce format | 32 bytes, passed as struct field | 32 bytes, base64url-encoded in JSON |
| Daemon sig purpose | Router signs nonce to prevent MitM | Same: daemon signs nonce to prevent MitM |
| Signature verification | `admission.AdmitNode()` | `ed25519.Verify()` direct (no AdmitNode) |
| On success | Node enters admitted_nodes[svtnID] | Connection enters authenticated state |
| Replay scope | Global nonce set, 60s TTL | Per-connection; nonce discarded on close |
| Frame auth | HMAC key derived and sent back | Not applicable; JSON RPC follows |

The management plane REUSES the conceptual pattern but does NOT call
`admission.GenerateChallenge` or `admission.AdmitNode` — those functions operate
on an `AdmittedKeySet` with SVTN-scoped state. The management server performs its own
nonce generation (`crypto/rand.Read(32)`) and its own `ed25519.Verify` call, keeping
the two authorization domains independent.

### 5. Replay Safety

- The nonce is 32 bytes of `crypto/rand` output — probability of collision is
  negligible (2^-256 birthday space per connection).
- **Per-connection replay:** the nonce is single-use within the connection. If
  the client sends the same `nonce_sig` twice in a future protocol extension, the
  server rejects the second.
- **Cross-connection replay:** because the management auth is connection-scoped (one
  handshake per TCP/Unix connection) and the nonce is fresh per connection, a
  recorded nonce_sig from a prior connection is useless — the server will issue a
  different nonce on the new connection.
- This is simpler than the SVTN global nonce TTL (ARCH-04) because management
  connections are short-lived per-RPC, not long-lived data-plane sessions.

### 6. Bounded Read (CWE-400)

Every read from a management socket — on both the server and the client — MUST use
`io.LimitReader(conn, MaxMessageBytes)` where `MaxMessageBytes = 64 KiB`. A hostile
or non-Switchboard peer sending an unbounded stream MUST cause the read to terminate
with an error (and the connection to close) rather than OOM the process. The existing
S-6.03 adversarial review surfaced this risk on the client read loop; the server is
subject to the same constraint. This is not optional.

### 7. Per-Connection Read Deadlines (CWE-400 / Ruling 1)

Every blocking `Decode` on the management socket MUST be preceded by
`conn.SetReadDeadline`. Two concrete defaults are mandatory:

| Phase | Deadline constant | Default value | Justification |
|-------|-------------------|---------------|---------------|
| Handshake (steps 2–4: CHALLENGE sent → CHALLENGE_RESPONSE received) | `HandshakeTimeout` | **10 seconds** | A legitimate client authenticates within one round-trip on any LAN or local socket. 10 s is generous enough for slow machines; short enough that a stuck idle client does not permanently occupy a goroutine. |
| RPC idle (step 6: AUTH_OK sent → next RPC request received) | `RPCIdleTimeout` | **30 seconds** | sbctl is a one-shot tool (one RPC per connection). 30 s covers any reasonable operator interaction time while bounding hung connections. |

Both constants are defined in `internal/mgmt`:
```go
const (
    HandshakeTimeout = 10 * time.Second
    RPCIdleTimeout   = 30 * time.Second
)
```

**Implementation rule:** Before every `json.Decoder.Decode` call, the server MUST
call `conn.SetReadDeadline(time.Now().Add(<timeout>))`. After a successful decode
the deadline MUST be reset to `time.Time{}` (no deadline) so writes are not
inadvertently bounded. On deadline expiry `json.Decoder.Decode` returns an error
(`net.Error.Timeout() == true`); the server closes the connection and logs
E-ADM-010. This directly closes the EC-001 CWE-400 gap identified in BC-2.07.004.

EC-001 in BC-2.07.004 should be updated to cite these concrete timeout values.

### 8. Connection-Cap Semaphore (CWE-770 / Ruling 3)

The accept loop is bounded by a semaphore of `MaxConcurrentConnections` goroutines.
No unbounded fd/goroutine growth is permitted.

```go
const MaxConcurrentConnections = 128 // default; tunable via NewServer option
```

**Implementation rule:** Before spawning a per-connection goroutine, `Serve` MUST
acquire the semaphore (buffered channel of size `MaxConcurrentConnections`). If
the channel is full, the accept loop blocks until a slot is free (back-pressure
against the kernel accept queue; excess connections queue in the OS backlog and are
eventually refused with a TCP RST if the backlog fills). The semaphore slot is
released when the connection goroutine exits.

**Transient Accept errors:** A temporary `net.Error` (`.Temporary() == true`) from
`Accept` MUST NOT terminate `Serve`. The server backs off with an exponential sleep
starting at 5 ms, capped at 1 s, and retries. A non-temporary (fatal) error from
`Accept` causes `Serve` to return. This prevents a kernel-level `EMFILE` burst
(too many open files) from permanently killing the management server.

**Product-owner note:** Add EC-012 to BC-2.07.004 per §Product-Owner Handoff below.

---

## internal/mgmt Package Design

### Classification

`internal/mgmt` is an **effectful-boundary** package: it owns the Unix socket / TCP
listener (I/O), performs the auth handshake (uses crypto primitives), and dispatches
to command handlers. It has no globally pure-core logic worth isolating into a
sub-package at MVP scope. See ARCH-09 for boundary vs. effectful distinction.

### Exported API Surface

```go
// Package mgmt implements the daemon-side management server per ADR-012.
// It listens on a Unix socket or TCP address, performs the Ed25519 challenge-response
// handshake, and dispatches authenticated RPC commands to registered handlers.
//
// Purity classification (ARCH-09): boundary — owns listener I/O and socket state;
// pure-core logic (challenge generation, signature verify) lives in crypto/ed25519
// and crypto/rand directly (not re-wrapped here).
package mgmt

import (
    "context"
    "crypto/ed25519"
    "net"
)

// MaxMessageBytes is the maximum JSON message size accepted on the management
// socket (server or client side). io.LimitReader MUST be applied before any
// json.Decoder.Decode call. 64 KiB is generous for all management RPCs.
const MaxMessageBytes = 1 << 16 // 64 KiB

// HandshakeTimeout is the read deadline applied during the challenge-response
// handshake (from CHALLENGE sent to CHALLENGE_RESPONSE received). Default 10s.
// Closes EC-001 CWE-400 gap (ADR-012 §7).
const HandshakeTimeout = 10 * time.Second

// RPCIdleTimeout is the read deadline applied after AUTH_OK is sent, while
// waiting for the first RPC request. Default 30s. Closes EC-001 gap.
const RPCIdleTimeout = 30 * time.Second

// MaxConcurrentConnections is the default semaphore size for concurrent
// per-connection goroutines. Excess connections back-pressure into the OS
// accept backlog. Prevents CWE-770 fd/goroutine exhaustion.
const MaxConcurrentConnections = 128

// OperatorKeySet holds the set of authorized operator public keys for this daemon.
// IsAuthorized is safe for concurrent use.
type OperatorKeySet struct { /* unexported */ }

// NewOperatorKeySet creates an OperatorKeySet from a slice of authorized public keys.
// Keys are copied; the caller's slice is not retained.
func NewOperatorKeySet(keys []ed25519.PublicKey) *OperatorKeySet

// IsAuthorized reports whether pubkey appears in the authorized set.
// Uses constant-time comparison to prevent timing oracle on key enumeration.
func (o *OperatorKeySet) IsAuthorized(pubkey ed25519.PublicKey) bool

// Handler is a registered command handler. Command is the RPC command name
// (e.g. "svtn.list"). Fn receives the authenticated connection context and the
// raw args JSON, and returns a data value (marshaled into the response envelope)
// or an error.
type Handler struct {
    Command string
    Fn      func(ctx context.Context, args json.RawMessage) (any, error)
}

// Server is the management plane server. It is started once per daemon mode.
// Construct via NewServer; never copy after first use.
type Server struct { /* unexported */ }

// NewServer constructs a Server with the given listener, operator key set,
// and registered handlers. No init() functions — all dependencies injected.
// daemonKey is the daemon's own Ed25519 private key, used to sign challenges.
// daemonVersion is the semver string embedded in AUTH_OK messages (e.g.
// "1.2.3" from ldflags, or the sentinel "dev" for unreleased builds).
// The caller MUST pass the version injected by build ldflags from
// cmd/switchboard.version — never hardcode "dev" as the production value.
func NewServer(
    ln net.Listener,
    daemonKey ed25519.PrivateKey,
    ops *OperatorKeySet,
    handlers []Handler,
    daemonVersion string,
) *Server

// Serve accepts connections and handles them until ctx is cancelled or the
// listener is closed. Returns when all in-flight connections have terminated
// (WaitGroup-tracked). Safe to call from a wg-tracked goroutine in the daemon
// lifecycle per ARCH-01 §Goroutine WaitGroup Contract.
func (s *Server) Serve(ctx context.Context) error

// Shutdown drains in-flight connections and closes the listener.
// Called by the daemon on SIGTERM/context cancel. Blocks until drained or
// ctx expires (mirrors drain timeout semantics from internal/drain).
func (s *Server) Shutdown(ctx context.Context) error
```

### Command Handler Registry (No init())

Handlers are registered at construction time via the `handlers []Handler` parameter
to `NewServer`. Each daemon mode's `main` function (or runXxx function) constructs
the handler slice explicitly before calling `NewServer`. There are no package-level
registrations and no `init()` side effects — dependencies are explicit and testable.

Example (router mode):
```go
handlers := []mgmt.Handler{
    {Command: "router.status",  Fn: buildRouterStatusHandler(router)},
    {Command: "router.metrics", Fn: buildRouterMetricsHandler(router)},
    {Command: "router.reload",  Fn: buildRouterReloadHandler(cfgPath)},
    {Command: "svtn.list",      Fn: buildSvtnListHandler(admittedKeySet)},
    // ...
}
srv := mgmt.NewServer(ln, daemonPrivKey, operatorKeySet, handlers, version)
```

### JSON Envelope

All RPC responses from command handlers use the envelope defined in
interface-definitions.md §JSON Output Schema:
```json
{ "ok": true, "error": null, "data": { ... } }
```
or on error:
```json
{ "ok": false, "error": { "code": "E-xxx", "message": "..." }, "data": null }
```
The `mgmt.Server` wraps every handler return value (or error) in this envelope before
writing to the connection. Handlers return `(any, error)` — they do NOT format the
envelope themselves, preserving consistency across all commands.

### Package DAG Constraints

`internal/mgmt` MAY import:
- `crypto/ed25519`, `crypto/rand`, `encoding/json`, `net`, `context`, `io`,
  `sync` (stdlib only — no external mgmt deps)
- `internal/config` (to read management socket path and operator key material)

`internal/mgmt` MUST NOT import:
- `internal/routing`, `internal/multipath`, `internal/arq`, `internal/replay`,
  `internal/paths`, `internal/halfchannel`, `internal/session`, `internal/tmux`,
  `internal/discovery` — these are data-plane packages.
- `cmd/sbctl` — downward import is forbidden.
- `internal/svtnmgmt` — svtnmgmt is the SVTN lifecycle manager; management RPC
  handlers that need it receive it via dependency injection (the handler `Fn` closes
  over the dependency, not the mgmt package itself).

`internal/mgmt` SHOULD import `internal/admission` ONLY for the nonce-generation
helper if one is extracted to a shared package. If the nonce generation stays inline
(`crypto/rand.Read(32)` + `ed25519.Sign`), then `internal/admission` is not needed
as a dependency and the DAG stays cleaner.

**Nothing imports `internal/mgmt` except `cmd/switchboard`** (the daemon entry point).
`cmd/sbctl` does not import `internal/mgmt` — it speaks the wire protocol over a
connection, not through the Go API.

---

## Wiring into cmd/switchboard

### Daemon Mode Startup

Each daemon mode starts its management listener before its data-plane listener.
The pattern is the same across all four modes:

```go
// In runRouter / runAccess / runConsole / runControl:
func runRouter(ctx context.Context, cfg *config.Config) error {
    // 1. Load daemon keypair (already loaded for SVTN admission).
    // 2. Build operator key set from cfg.AuthorizedOperatorKeys.
    operatorKeys := mgmt.NewOperatorKeySet(cfg.AuthorizedOperatorKeys)
    // 3. Start management listener (see §Unix Socket Permissions below).
    mgmtLn, err := net.Listen("unix", cfg.ManagementSocket)  // or "tcp" for console
    // 4. Build handler slice (injects data-plane deps by closure).
    handlers := buildRouterHandlers(router, admittedKeySet, cfgPath)
    // 5. Construct and serve.
    //    daemonVersion is the ldflags-injected build version (or "dev" sentinel).
    mgmtSrv := mgmt.NewServer(mgmtLn, daemonPrivKey, operatorKeys, handlers, version)
    wg.Add(1)
    go func() { defer wg.Done(); _ = mgmtSrv.Serve(ctx) }()
    // On shutdown: mgmtSrv.Shutdown(shutdownCtx)
}
```

### Unix Socket Permissions (CWE-276 / Ruling 4)

Unix management sockets MUST be created with permissions `0600` (owner read/write
only). The default umask on Linux/macOS often produces `0666` or `0777`; relying
on the system umask is CWE-276. The required pattern:

```go
// Set a restrictive umask before net.Listen so the socket file is created
// with the right permissions atomically. Restore afterward.
old := syscall.Umask(0177) // 0777 &^ 0177 = 0600
mgmtLn, err := net.Listen("unix", cfg.ManagementSocket)
syscall.Umask(old)
```

**Rationale for 0600 vs 0660:** 0600 (owner-only) requires the daemon and sbctl
to run as the same OS user. 0660 + operator group would enable multi-user operator
access without key management overhead but introduces group membership as an
additional attack surface. For MVP, 0600 is simpler and safer. A future ADR may
relax to 0660 if multi-user access is required.

**TOCTOU window:** The umask-before-Listen approach creates the socket with the
correct permissions atomically — there is no window between creation and chmod
where an unprivileged process could connect. A chmod-after-Listen would have a
TOCTOU window and MUST NOT be used.

**Console TCP address (not a Unix socket):** The console daemon uses TCP at
`127.0.0.1:9091` (loopback-only). This is confirmed by the existing ARCH-05
§Daemon Management Socket table. Binding to `0.0.0.0` or `::` is FORBIDDEN for
the management TCP listener. The loopback-only binding already limits access to
local processes; no file permission applies, but the address binding check MUST
be enforced in config validation (see E-CFG-008 extension note below).

**Product-owner note:** BC-2.07.004 must gain EC-013 per §Product-Owner Handoff.

The management server goroutine is WaitGroup-tracked per ARCH-01
§Goroutine WaitGroup Contract and ARCH-01 v1.7 obligation.

### Socket Paths (from ARCH-05 §Daemon Management Socket)

| Daemon Mode | Socket Type | Default Address |
|-------------|-------------|-----------------|
| router | Unix socket | `/run/switchboard-router.sock` |
| access | Unix socket | `/run/switchboard-access.sock` |
| console | TCP | `127.0.0.1:9091` |
| control | Unix socket | `/run/switchboard-control.sock` |

These paths are already specified in ARCH-05. `cmd/switchboard` passes the appropriate
address for each mode; `internal/mgmt` is socket-type-agnostic (it accepts a
`net.Listener`).

---

## Config Additions (internal/config)

Two new fields are added to `internal/config.Config`:

```go
// ManagementSocket is the Unix socket path (or TCP address for console) that
// the daemon's management server listens on. Validated as a non-empty string.
// Defaults are applied by the daemon mode's startup code, not by Validate.
// Maps to the socket paths from ARCH-05 §Daemon Management Socket.
ManagementSocket string `yaml:"management_socket"`

// AuthorizedOperatorKeys is a list of PEM-encoded Ed25519 public keys that
// are authorized to authenticate to the management plane. If empty, the daemon
// falls back to accepting the daemon's own key (bootstrap mode).
// Each entry is a PEM block of type "PUBLIC KEY" (PKIX/SubjectPublicKeyInfo).
AuthorizedOperatorKeys []string `yaml:"authorized_operator_keys"`
```

**BC-2.09.003 config-schema impact (flag for product-owner):** These two fields add
to the config schema. BC-2.09.003 covers config validation correctness; a story
implementing these fields MUST add validation coverage for:
- `management_socket`: when set, must be a non-empty path (not whitespace).
- `authorized_operator_keys`: each entry must be a valid PEM block of type
  `"PUBLIC KEY"` containing a 32-byte Ed25519 key; reject at validation time with
  E-CFG-00x (new config error code to be assigned by product-owner).

The existing config validation in `Validate()` collects-all-failures pattern applies.

---

## cmd/sbctl Client Auth

### OpenSSH Key Loading

The client loads the operator's Ed25519 private key from the path given by `--key`
(default `~/.ssh/id_ed25519`). The loading sequence:

```go
import "golang.org/x/crypto/ssh"

keyBytes, err := os.ReadFile(keyPath)  // bounded: LimitReader(f, 1<<16)
signer, err := ssh.ParsePrivateKey(keyBytes)
// Extract raw ed25519.PrivateKey:
cryptoSigner, ok := signer.GetPublicKey().(ssh.CryptoPublicKey)
ed25519PrivKey := cryptoSigner.CryptoPublicKey().(ed25519.PrivateKey)
```

The private key is used locally for `ed25519.Sign(privKey, nonce_bytes)` only. It is
never serialized, logged, or transmitted over the socket (DI-002 preserved).

The `--key` path file is read with `io.LimitReader(f, 1<<16)` to prevent an
accidental `--key /dev/zero` from OOMing the client process (symmetric CWE-400
protection as specified in ADR-012 §6).

### New Dependency: golang.org/x/crypto

`golang.org/x/crypto/ssh` provides `ParsePrivateKey` and `ssh.CryptoPublicKey`. This
is the only new external dependency introduced by the management plane.

**Version pinning guidance:**
- Add to `go.mod`: `golang.org/x/crypto v0.38.0` (or latest stable at time of
  implementation — verify with `go get golang.org/x/crypto@latest` and pin
  the resolved version).
- The `x/crypto` family follows the same release cadence as Go itself; pin to a
  specific minor version in `go.mod` and update deliberately.
- In `.tool-versions` / mise context: the Go toolchain version (currently 1.25+)
  already includes the standard `crypto/ed25519` package. `x/crypto` is needed only
  for OpenSSH PEM parsing — the data-plane code does not gain a new dependency.

### Authenticate() FAIL-CLOSED Contract

**Ruling 2 (adversarial review):** The exported `Authenticate` function MUST carry
a `context.Context` as its first parameter per go.md rule 7. The function derives
the connection read deadline from the context deadline; if the context has no
deadline, the HandshakeTimeout constant (10 s) is applied. The caller (sbctl main)
is responsible for constructing a context with the operator-specified `--timeout`
(default 5 s for the dial; the handshake inherits the remaining deadline). This
self-bounds `Authenticate` regardless of caller negligence — it does NOT rely on
the caller to set a deadline on the connection.

**Canonical signature:**
```go
// Authenticate performs the ADR-012 Ed25519 challenge-response handshake.
// ctx governs the total handshake time: if ctx has a deadline,
// conn.SetReadDeadline is derived from it; otherwise HandshakeTimeout (10s)
// is used. Returns nil only on verified AUTH_OK. Any other outcome —
// connection error, malformed message, AUTH_FAIL, deadline exceeded,
// truncated stream, oversized response — returns a non-nil error.
func Authenticate(ctx context.Context, conn net.Conn, privKey ed25519.PrivateKey) error
```

The function MUST:

1. Compute the effective deadline: if `ctx.Deadline()` is set, use it; otherwise
   `time.Now().Add(HandshakeTimeout)`. Call `conn.SetReadDeadline(deadline)`.
2. Read the CHALLENGE message from the server using `json.NewDecoder(io.LimitReader(conn, MaxMessageBytes))`.
3. Decode `nonce_bytes` from the base64url nonce field.
4. Verify `daemon_sig` against the daemon's pubkey (if the daemon pubkey is known /
   pinned — optional in MVP; can be trusted on first use).
5. Sign: `nonce_sig = ed25519.Sign(privKey, nonce_bytes)`.
6. Send CHALLENGE_RESPONSE.
7. Read AUTH_OK or AUTH_FAIL (deadline still active).
8. Return `nil` ONLY if an AUTH_OK message was received and successfully decoded.
   Any other outcome — connection error, malformed message, AUTH_FAIL, deadline
   exceeded, oversized response — MUST return a non-nil error. There is no code
   path that returns `nil` without receiving a verified AUTH_OK. This is the
   fail-closed invariant.

The existing S-6.03 scaffold (`cmd/sbctl/client.go` run/dispatch/JSON-envelope) is
extended with `Authenticate()` as a pre-dispatch step. The connection attempt
(`net.Dial`) and the authentication are separate steps with separate error codes.

### Error Code Disambiguation (Ruling 5 — adversarial review)

Three distinct failure modes in sbctl MUST use distinct error codes:

| Stage | Error code | Message format | Exit |
|-------|-----------|----------------|------|
| Dial / connect unreachable | `E-NET-001` | "daemon unreachable: \<address\>: \<reason\>" | 1 |
| Key file load failure (missing / oversized / malformed / wrong type) | `E-CFG-010` | "key load failed: \<path\>: \<reason\>" | 1 |
| Post-auth RPC dispatch failure (response decode error, unknown command, handler error after AUTH_OK) | `E-RPC-001` | "rpc failed: \<command\>: \<reason\>" | 1 |

**Rationale:** All three were previously collapsed into `E-NET-001`, which tells
the operator "daemon unreachable" for failures that are actually a local key file
problem (E-CFG-010) or a post-auth server-side dispatch error (E-RPC-001). An
operator seeing E-NET-001 on a key-load failure would waste time checking network
connectivity. `E-NET-001` is reserved strictly for `net.Dial` / `net.DialContext`
failures — the daemon truly is not connectable.

`E-CFG-010` is chosen in the existing CFG family (free slot: E-CFG-008 and
E-CFG-009 are already taken for management-socket and PEM validation; E-CFG-010
is the next free code). `E-RPC-001` opens a new RPC family.

Authentication failure (`E-ADM-010`) is unchanged — it already has its own code.

---

## BC Recommendations for Product-Owner

The following NEW behavioral contracts are recommended. The architect does not write
BCs — these are recommendations for the product-owner to act on.

### New BCs Needed

**BC-2.07.004: Daemon management server authenticates all connections via Ed25519 challenge-response (server-side counterpart to BC-2.07.002)**
- Subsystem: network-management (SS-07)
- Module: internal/mgmt
- Key postconditions to specify:
  - PC-1: daemon issues a fresh CHALLENGE message immediately on connection
  - PC-2: daemon rejects connections without a valid AUTH_RESPONSE (closes with E-ADM-010)
  - PC-3: daemon rejects replay of a nonce within a connection
  - PC-4: daemon closes connection after AUTH_FAIL without processing any RPC
  - PC-5: all RPC commands require prior successful auth (no unauthenticated RPC path)
  - PC-6: daemon reads are bounded (MaxMessageBytes = 64 KiB) on every socket read
- Edge cases: unauthorized key, malformed JSON, truncated stream, oversized message (> 64 KiB), connection closed mid-handshake

**BC-2.09.003 amendment: config validation for management_socket and authorized_operator_keys fields**
- The product-owner should amend BC-2.09.003 to cover the two new config fields.
- New error codes: E-CFG-008 (invalid management_socket), E-CFG-009 (malformed operator pubkey PEM).

### BC-2.07.002 PC-2/PC-3 Anchor Update

BC-2.07.002 PC-2 ("sbctl authenticates the operator's OpenSSH key against the daemon's authorized key list") and PC-3 ("if authenticated: the requested operation is executed") previously had no server-side implementation to anchor to. With BC-2.07.004 in place, both PCs now have a real counterpart. The product-owner should add a cross-reference from BC-2.07.002 §Related BCs to BC-2.07.004.

### EC-002 Error Code Bug in S-6.03 (for product-owner / story-writer to fix)

**Story S-6.03, EC-002** currently specifies:
> "OpenSSH auth fails (wrong key) | E-ADM-001 returned from daemon; sbctl exits 1"

This is WRONG. `E-ADM-001` is the node admission failure code (ARCH-04 §Tier 1
Admission Protocol step 4: `failure → E-ADM-001, connection closed`). Operator-CLI
authentication failure is `E-ADM-010` per BC-2.07.002 PC-4.

The story-writer and product-owner should update EC-002 in S-6.03 to reference
`E-ADM-010`.

---

## VP Recommendations for Product-Owner

The following new Verification Properties are recommended. These extend the existing
VP catalog. The final assigned VP numbers (VP-064 through VP-067) were allocated after
initial drafting; an earlier draft of this document referenced VP-058–VP-062 — those
numbers are stale and were superseded during VP-INDEX reconciliation (VP-058 through
VP-063 were already assigned to other modules). The authoritative assignments below
must be used.

### New VPs (assigned VP-064 through VP-067)

**VP-064: Management server rejects unauthenticated connections (integration)**
- Property: a connection that sends a valid RPC request without completing the
  challenge-response handshake receives AUTH_FAIL + connection close, never RPC data
- Module: internal/mgmt
- Phase: P0 (MVP-blocking — all sbctl operations require auth)

**VP-065: Management server rejects replayed challenge nonce (integration)**
- Property: if client sends the same nonce_sig twice on one connection, the second
  auth attempt is rejected
- Module: internal/mgmt
- Phase: P1

**VP-066: Management server enforces bounded read (unit + fuzz)**
- Property: a connection sending a message > MaxMessageBytes causes an error and
  connection close, not OOM
- Module: internal/mgmt
- Phase: P0 (CWE-400, raised in S-6.03 security review)

**VP-067: Authenticate() is fail-closed (unit)**
- Property: `Authenticate()` returns non-nil error for every non-AUTH_OK outcome:
  connection error, malformed message, AUTH_FAIL, truncated stream
- Module: cmd/sbctl
- Phase: P0 (fail-closed is a hard security requirement)

> **Note:** The original draft included a fifth VP for E2E management plane across
> all four daemon types (previously draft-labeled VP-062). That property is deferred
> to a subsequent story's integration harness and is not assigned a VP number in this
> wave. See S-W5.02 scope below.

**VP-INDEX.md and ARCH-11 must be updated by the state-manager / product-owner when
these VPs are formally adopted.** The current project total is 67 VPs across 45 BCs.
The four Wave 5 management-plane VPs are VP-064, VP-065, VP-066, and VP-067.

---

## Story Decomposition Recommendations for Story-Writer

The management plane work decomposes into three stories with clear serialization
constraints.

### Story 1: S-6.03 (re-scope — existing story)

**Title:** sbctl CLI scaffold, connection-error reporting, client auth, and BC-2.07.003

**Scope:**
- `cmd/sbctl/client.go`: connection, `Authenticate()` (fail-closed), JSON dispatch
- `cmd/sbctl/main.go`: flag parsing (`--target`, `--key`, `--json`, `--timeout`),
  subcommand routing, error envelope output
- `--key` flag: `golang.org/x/crypto/ssh.ParsePrivateKey` key loading (bounded read)
- `cmd/sbctl/client.go`: all of BC-2.07.003 (connection error paths: E-NET-001)
- Wire Authenticate() to send CHALLENGE_RESPONSE and parse AUTH_OK/AUTH_FAIL
- Unit tests for VP-067 (fail-closed) and VP-030 (connection error)
- **Fix EC-002**: change E-ADM-001 → E-ADM-010 in the story's own edge-case table

**Dependencies:** Needs the wire protocol spec (ADR-012, this document) — unblocked
now. Does NOT need the server to exist for unit testing (mock the connection).

**Estimate:** 5 points

**S-6.02 ∥ S-5.02 conflict note (from existing spec):** S-6.03 touches
`cmd/sbctl/main.go`; S-6.02 also touches `cmd/sbctl`. These two stories MUST NOT
be in flight concurrently on the same branch. The existing serialization constraint
stands.

### Story 2: S-W5.01 (new — daemon management server)

**Title:** internal/mgmt server, config additions, cmd/switchboard wiring

**Scope:**
- `internal/mgmt/` package: full implementation of ADR-012 server side
  (`Server`, `NewServer`, `OperatorKeySet`, `Serve`, `Shutdown`, challenge generation,
  auth handshake, bounded read, handler dispatch, JSON envelope wrapping)
- `internal/config/config.go`: add `ManagementSocket` and `AuthorizedOperatorKeys`
  fields; extend `Validate()` with E-CFG-008 and E-CFG-009
- `cmd/switchboard/main.go` (or per-mode run functions): wire management listener
  start into all four daemon modes; WaitGroup-track the mgmt goroutine
- Unit tests for VP-064 (unauthenticated rejection), VP-065 (replay rejection),
  VP-066 (bounded read)

**Dependencies:** ADR-012 (this document); golang.org/x/crypto is NOT needed here
(server uses stdlib crypto/ed25519 only).

**Estimate:** 8 points

**Must ship AFTER S-6.03** so the client auth format is locked before the server
implements the matching handshake. In practice can be developed in parallel on
separate branches because the protocol is fully specified in ADR-012, but integration
tests require both.

### Story 3: S-W5.02 (new — e2e management plane)

**Title:** E2E integration test harness across all four daemon types

**Scope:**
- Integration test that starts all four daemon types in-process (or as subprocesses)
  and exercises the full sbctl connect → auth → RPC → disconnect cycle
- Covers e2e management plane validation across all four daemon types
- This is the holdout / convergence-gate story for the management plane
- e2e VP: VP-049 (pre-assigned via VP-INDEX v2.6)

**Dependencies:** S-6.03 AND S-W5.01 must both be merged before this story can begin.

**Estimate:** 5 points

### Summary

| Story | Scope | Est. | Depends On |
|-------|-------|------|-----------|
| S-6.03 (re-scoped) | sbctl client auth + connection error | 5pt | ADR-012 (unlocked) |
| S-W5.01 (new) | internal/mgmt server + config + wiring | 8pt | ADR-012 |
| S-W5.02 (new) | E2E integration harness (VP-049) | 5pt | S-6.03 + S-6.06 + S-W5.01 |

---

## PC-3 Replay-Nonce Ruling (Ruling 7 — adversarial review)

**Background:** BC-2.07.004 PC-3 specifies that the server records the challenge
nonce in a "per-connection nonce set" and rejects a second presentation of the same
nonce on the same connection. The ADR-012 protocol is single-handshake-per-connection:
exactly one CHALLENGE is issued and exactly one CHALLENGE_RESPONSE is accepted. Once
authentication succeeds, the connection enters the RPC phase and no further
CHALLENGE/CHALLENGE_RESPONSE exchange occurs. Therefore, a literal per-connection
nonce set that checks for nonce re-use is vacuous — the nonce can never be re-presented
without the connection first entering an already-authenticated state where
CHALLENGE_RESPONSE messages are meaningless.

**Ruling:** Option (a) — structural post-auth guard — is the correct resolution.

The server MUST maintain a boolean `authenticated` flag (or equivalent state variable)
per connection. After a successful auth handshake sets `authenticated = true`, any
subsequent message with `"type":"challenge_response"` or any attempt to re-enter the
handshake flow MUST cause the server to close the connection with E-ADM-010 and log
a security event. This is a genuine, enforceable invariant:

- It prevents any future protocol extension that adds renegotiation from
  inadvertently opening a replay path.
- It is implementable with a single boolean, not a set.
- It maps cleanly to EC-011 (existing): "Client skips handshake / unexpected
  type → AUTH_FAIL + close."

The per-connection nonce set mentioned in the original spec is REPLACED by this
structural guard. There is no need to store nonce bytes after authentication
completes — the state machine itself enforces the invariant.

**Product-owner PC-3 rewording (exact text for BC-2.07.004):**

> PC-3 — Post-auth protocol violation rejected: After a successful auth handshake
> on connection C, any further `"type":"challenge_response"` message on C is treated
> as a protocol violation. The server closes the connection with E-ADM-010 and logs
> a security event. The server maintains a per-connection `authenticated` boolean;
> no nonce set is stored after auth. Cross-connection replay is prevented by the
> fresh nonce issued on every new connection (`crypto/rand.Read(32)`).

This rewording replaces the current PC-3 text about "per-connection nonce set" in
BC-2.07.004. The VP-065 property must also be updated to reflect this structural
assertion rather than a nonce-set membership check (see §Product-Owner Handoff).

---

## Wave-5 Convergence Rulings (Round-2) — v1.3

These rulings were produced after a fresh 6-pass adversarial convergence round on
S-W5.01 and S-6.03. The convergence counter reset on BC-5.39.001. Each ruling is
concrete enough to implement against. Downstream agents (product-owner, story-writer,
test-writer, implementer) must apply the changes listed in §Product-Owner Handoff
(round-2) and §Story-Writer Handoff (round-2) at the end of this section.

---

### Ruling A — Daemon Key Provisioning for the Access Daemon (CRITICAL C-1)

**Finding grounded in code:**
`cmd/switchboard/access.go:135` calls
`startMgmtServer(ctx, &mgmtWG, cfg, "access", nil, nil)` passing a `nil`
`ed25519.PrivateKey`. `mgmt.NewServer` (mgmt.go:135–164) panics on empty
`daemonVersion` but has NO guard for a nil `daemonKey`. On the first connection,
`handleConnection` (mgmt.go:391) executes `s.daemonKey.Public().(ed25519.PublicKey)` —
a nil pointer dereference that panics in a per-connection goroutine with no `recover()`,
crashing the entire daemon process. The comment in access.go asserting "nil key means
no connection authenticates, which is acceptable" is factually wrong: the panic occurs
during challenge generation (PC-1), before any auth decision.

This is an unauthenticated remote DoS — any peer that connects to the access daemon's
management socket crashes it. Violates BC-2.07.004 Precondition 3 and Invariant 1
(fail-closed).

**Ruling A.1 — Ephemeral keypair at access daemon startup.**

The access daemon (`cmd/switchboard/access.go` — `runAccess` function) MUST generate
an ephemeral Ed25519 keypair via `crypto/rand` at startup and pass it to
`startMgmtServer`. The generation site is `runAccess` itself (not a shared helper),
immediately before calling `startMgmtServer`. Canonical code:

```go
// In runAccess, before startMgmtServer call:
_, daemonPrivKey, err := ed25519.GenerateKey(rand.Reader)
if err != nil {
    return fmt.Errorf("runAccess: generate daemon keypair: %w", err)
}
```

This keypair is ephemeral: it is generated fresh on every process start and is never
written to disk. It is the sole authority in bootstrap mode until the operator populates
`authorized_operator_keys` in config. In bootstrap mode the daemon's own public key is
the sole authorized key (BC-2.07.004 PC-9); because the keypair is ephemeral, the
operator must connect using the corresponding private key derived from the same ephemeral
key — this is not the normal operator flow. Bootstrap mode is for testing and initial
bring-up only.

**Security note (ephemeral key):** An ephemeral key means operators cannot pin the
daemon identity across restarts in this wave. The CHALLENGE `daemon_sig` field, which
the client uses to verify it is talking to the expected daemon, will change on every
restart. This is an acceptable MVP limitation: the daemon management plane is
functional and fail-closed immediately; key pinning (persistent `key_file` loading
with known public key for client-side verification) layers on in S-6.02 key
management. This note MUST be preserved in the S-6.02 story so the implementer
understands the trust gap.

**Ruling A.2 — nil-key guard in `mgmt.NewServer`.**

`mgmt.NewServer` MUST panic if `daemonKey` is nil or its length is not
`ed25519.PrivateKeySize` (64 bytes). The panic is at construction time, mirroring
the existing `daemonVersion` emptiness panic. Canonical guard:

```go
// In NewServer, immediately after the daemonVersion check:
if len(daemonKey) != ed25519.PrivateKeySize {
    panic("mgmt.NewServer: daemonKey must be a valid ed25519.PrivateKey (64 bytes)")
}
```

This ensures a nil key or a truncated key can never reach a request goroutine. The
check fires at daemon startup — not on first connection — so the failure is immediate
and unambiguous rather than a runtime panic under load.

**Ruling A.3 — Bootstrap mode and the OperatorKeySet interaction.**

When `runAccess` passes an ephemeral key to `startMgmtServer` with an empty
`AuthorizedOperatorKeys` config list, the `OperatorKeySet` is in bootstrap mode
(`IsBootstrap() == true`). `handleConnection` then compares the connecting client's
public key against `s.daemonKey.Public()`. Since the daemon key is ephemeral, no
external operator can successfully authenticate unless they hold the corresponding
private key — which in practice means only process-local testing. This is intentional.
When S-6.02 ships and the daemon loads a persistent key from `key_file`, the operator
can pre-configure `authorized_operator_keys` to enable normal authenticated access.

**Downstream changes required by Ruling A:**

- Story-writer: amend S-W5.01 to add an AC covering `runAccess` ephemeral keypair
  generation (new AC, e.g. AC-015: "runAccess generates an ephemeral ed25519.PrivateKey
  via crypto/rand before calling startMgmtServer; the returned error from GenerateKey
  aborts daemon startup"). Amend S-W5.01 to add AC-016: "mgmt.NewServer panics if
  len(daemonKey) != ed25519.PrivateKeySize; passing nil panics immediately."
- Product-owner: amend BC-2.07.004 Precondition 3 to state: "The daemon's own
  Ed25519 private key is available — either an ephemeral key generated at startup
  (this wave) or a persistent key loaded from key_file (S-6.02). A nil or invalid
  key causes NewServer to panic at construction time, preventing any connection from
  being accepted."
- Test-writer: add test `TestNewServer_PanicsOnNilKey` and
  `TestRunAccess_GeneratesEphemeralKey` (verify a non-nil key is passed to startMgmtServer).

---

### Ruling B — Serve Graceful-Shutdown Return Value (HIGH / PC-10)

**Finding grounded in code:**
`Serve` (mgmt.go:199–276) uses `defer close(done)` at the top of the function. The
`done` channel is closed only when `Serve` returns — it is NOT closed before the
for-loop runs. The ctx-watcher goroutine closes the listener on `ctx.Done()`, causing
`Accept` to return an error. When this error arrives at lines 229–263, the select
`case <-done:` (lines 234–238) is evaluated: but at this moment `done` is still open
(Serve has not returned yet), so the case is never selected and the check falls
through to the backoff/fatal-error path at line 242–263. `net.ErrClosed` is not a
`net.Error.Temporary()` error, so it falls through to `return err` at line 263.
Result: `Serve` returns `net.ErrClosed` on every normal shutdown. BC-2.07.004 PC-10
("Serve returns nil on normal Shutdown") is violated.

**Ruling:** Add an `atomic.Bool` named `shuttingDown` to the `Server` struct. `Shutdown`
sets it to `true` before calling `s.ln.Close()`. The ctx-watcher goroutine also sets it
to `true` before calling `s.ln.Close()`. In the `Accept` error path, the check becomes:

```go
if ne, ok := err.(net.Error); ok && ne.Temporary() { //nolint:staticcheck
    // ... existing backoff logic
}
// Intentional shutdown: listener was closed by Shutdown() or ctx cancel.
if s.shuttingDown.Load() || errors.Is(err, net.ErrClosed) && ctx.Err() != nil {
    s.connWG.Wait()
    return nil
}
// Fatal accept error — drain and return.
s.connWG.Wait()
return err
```

The authoritative predicate is: return nil if `s.shuttingDown.Load()` is true (explicit
`Shutdown()` called) OR `(errors.Is(err, net.ErrClosed) && ctx.Err() != nil)` (listener
closed because context was cancelled). In all other cases, return the error.

The dead `case <-done:` branches in the accept loop (lines 223–226 and 254–257) MUST be
removed. They are unreachable dead code because `done` is never closed while the loop is
running. Their presence is misleading and causes incorrect reasoning about the shutdown
path. The corrected loop structure is:

```go
// Before Accept: acquire semaphore.
select {
case s.sem <- struct{}{}:
case <-ctx.Done():
    s.connWG.Wait()
    return nil
}
```

The `case <-ctx.Done()` in the pre-Accept semaphore select IS correct (ctx cancel before
Accept) and MUST be retained.

**Ruling on the existing `case <-done:` in the backoff select (lines 252–256):** This
branch is also unreachable for the same reason. Remove it; replace with `case <-ctx.Done():`.

**Canonical Serve return contract (BC-2.07.004 PC-10):**
- Returns `nil`: `Shutdown(ctx)` was called, OR ctx was cancelled (either triggers
  `s.shuttingDown.Store(true)` followed by `s.ln.Close()`).
- Returns non-nil error: a fatal `Accept` error occurred that was NOT caused by
  intentional shutdown (e.g., the underlying fd was stolen by another process).

**Downstream changes required by Ruling B:**

- Story-writer: amend S-W5.01 to add AC-017: "Serve returns nil when Shutdown is
  called or ctx is cancelled. Test: `TestServe_ReturnsNilOnShutdown` — start server,
  call Shutdown, verify Serve return value is nil. Test: `TestServe_ReturnsNilOnCtxCancel`
  — start server with cancellable ctx, cancel ctx, verify Serve returns nil."
- Product-owner: amend BC-2.07.004 to add PC-10 explicitly:
  > PC-10: `Serve` returns nil when `Shutdown` is called or the context is cancelled
  > (normal daemon lifecycle). `Serve` returns a non-nil error only on an unexpected
  > listener failure unrelated to shutdown.
- Implementer: add `shuttingDown atomic.Bool` field to `Server`; set it in both
  `Shutdown` and the ctx-watcher goroutine before calling `s.ln.Close()`; update the
  Accept error path as specified above; remove the dead `case <-done:` branches.

---

### Ruling C — Server-Side RPC Error Taxonomy (HIGH)

**Finding grounded in code:**
`mgmt.go:555` emits `Code: "E-RPC-001"` for "unknown command". But `E-RPC-001` is the
CLIENT-SIDE sbctl error code ("rpc failed: \<command\>: \<reason\>") per ARCH-12
§Error Code Disambiguation (Ruling 5) and error-taxonomy.md. `mgmt.go:561` emits
`Code: "E-RPC-002"` for handler execution errors — `E-RPC-002` is UNDEFINED in
error-taxonomy.md and in any BC.

**Ruling:** Define two new SERVER-SIDE RPC error codes. These are emitted in the
server's JSON response envelope (`"ok": false, "error": {...}`) and are distinct from
the CLIENT-SIDE `E-RPC-001` that sbctl emits to its stderr:

| Code | Emitted by | Condition | Message format |
|------|-----------|-----------|----------------|
| `E-RPC-010` | `internal/mgmt` (server) | Authenticated request names a command that is not registered in the handler slice | `"unknown command: <command>"` |
| `E-RPC-011` | `internal/mgmt` (server) | Registered handler returns a non-nil error | `"<handler error message>"` (the handler's own error string, not wrapped further) |

**Rationale for numbering:** E-RPC-001 is the client code (exit 1, sbctl stderr).
E-RPC-010 and E-RPC-011 open a dedicated server sub-range within the RPC family,
making the boundary unambiguous in logs and tests. Neither number collides with
existing codes.

**Implementation:** In `handleConnection` (mgmt.go:553–568), change:
```go
// BEFORE (wrong):
resp.Error = &rpcError{Code: "E-RPC-001", Message: "unknown command"}
// ...
resp.Error = &rpcError{Code: "E-RPC-002", Message: err.Error()}

// AFTER (correct):
resp.Error = &rpcError{Code: "E-RPC-010", Message: fmt.Sprintf("unknown command: %s", req.Command)}
// ...
resp.Error = &rpcError{Code: "E-RPC-011", Message: err.Error()}
```

**Downstream changes required by Ruling C:**

- Product-owner: add E-RPC-010 and E-RPC-011 to error-taxonomy.md RPC section:
  > | E-RPC-010 | RPC | broken | — (in-band) | "unknown command: \<command\>" | BC-2.07.004; emitted by internal/mgmt server when an authenticated RPC names an unregistered command. Distinct from client-side E-RPC-001. |
  > | E-RPC-011 | RPC | broken | — (in-band) | "\<handler error message\>" | BC-2.07.004; emitted by internal/mgmt server when a registered handler returns a non-nil error. Distinct from E-RPC-001. |
- Product-owner: amend BC-2.07.004 to add postconditions:
  > PC-11: When an authenticated RPC names an unregistered command, the server
  > responds with `{"ok":false,"error":{"code":"E-RPC-010","message":"unknown command: <cmd>"},"data":null}`. The connection is NOT closed.
  > PC-12: When a registered handler returns a non-nil error, the server responds
  > with `{"ok":false,"error":{"code":"E-RPC-011","message":"<err>"},"data":null}`. The connection is NOT closed.
- Story-writer: amend S-W5.01 to update AC-007 (RPC dispatch): replace E-RPC-001/E-RPC-002
  references with E-RPC-010/E-RPC-011 throughout the story's test table. Add edge cases:
  EC-014 (unknown command → E-RPC-010 in-band, connection stays open) and EC-015
  (handler error → E-RPC-011 in-band, connection stays open).
- Implementer: change the two `rpcError` literals in mgmt.go as above.

---

### Ruling D — Console Management-Socket Loopback Enforcement (HIGH / AC-014 / BC-2.07.004 Inv-7 / EC-013)

**Finding grounded in code:**
`buildMgmtListener` (mgmt_wire.go:163–178) dispatches to `net.Listen(network, address)`
for TCP (console mode) with `address` taken directly from `resolveManagementSocket`
without any validation that the address is loopback-only. The config validation path
(`config.Validate()`) has no mode parameter and cannot distinguish console mode from
other modes, so a console `management_socket` of `"0.0.0.0:9091"` or `":9091"` passes
config validation unchecked and reaches `net.Listen`.

**Ruling:** The enforcement lives in `buildMgmtListener` in `cmd/switchboard/mgmt_wire.go`.
This is the correct placement because:
1. `buildMgmtListener` already branches on `network == "unix"` vs. TCP; the TCP branch
   is console-specific.
2. Adding mode-awareness to `config.Validate()` would require threading mode through
   the config package, violating its purity-boundary classification (pure-core parse+validate).
3. The wiring layer already owns all mode-specific socket decisions.

**Rejection predicate:** For TCP mode (console), the address MUST have a host that is
one of: `127.0.0.1`, `[::1]`, `localhost`. Any address whose host is `0.0.0.0`, `::`,
empty (bare port `:9091`), or any non-loopback IP is rejected with error code `E-CFG-008`
and message format: `"config error: management_socket: console mode requires a loopback
address (127.0.0.1, [::1], or localhost); got: <address>"`.

**Implementation in `buildMgmtListener`:**

```go
// TCP (console mode) — validate loopback-only BEFORE net.Listen.
if network == "tcp" {
    host, _, err := net.SplitHostPort(address)
    if err != nil {
        return nil, fmt.Errorf("E-CFG-008: management_socket: invalid address %q: %w", address, err)
    }
    switch host {
    case "127.0.0.1", "::1", "localhost":
        // Loopback — allowed.
    default:
        return nil, fmt.Errorf("E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: %s", address)
    }
}
ln, err := net.Listen(network, address)
```

Note: the error message embeds the code `E-CFG-008` directly in the format string so
it is grep-able in logs and test assertions. The daemon aborts startup with this error;
it is not treated as a non-fatal warning.

**Rationale for IPv6 loopback `[::1]`:** Including `[::1]` future-proofs for IPv6-only
hosts without requiring an ADR. Excluding it would cause unnecessary failures on
IPv6-only systems. Console-mode TCP management is already loopback-only by policy;
`[::1]` is loopback in IPv6.

**Rationale for `localhost`:** Allows operators to specify `localhost:9091` in config
rather than requiring the numeric IP. `net.Listen("tcp", "localhost:9091")` resolves
to loopback on all standard platforms.

**IPv6 `::` rejection:** The bare `::` host is the IPv6 unspecified address
(equivalent to `0.0.0.0`); it MUST be rejected.

**Empty host rejection:** A bare port like `:9091` produces an empty host from
`net.SplitHostPort`; this falls through the switch default and is rejected.

**Downstream changes required by Ruling D:**

- Story-writer: amend S-W5.01 AC-014 to specify the loopback-validation logic in
  `buildMgmtListener` for the TCP/console path, including the E-CFG-008 rejection for
  0.0.0.0/::/empty-host. Add a test:
  `TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback` — call `buildMgmtListener`
  with mode="console" and address "0.0.0.0:9091"; verify error contains "E-CFG-008".
  Add: `TestBuildMgmtListener_ConsoleTCP_AcceptsLoopback` — "127.0.0.1:9091", "[::1]:9091",
  "localhost:9091" all succeed.
- Product-owner: amend BC-2.07.004 EC-013 to add the explicit rejection predicate:
  "Any console-mode management_socket TCP address whose host is not 127.0.0.1, [::1],
  or localhost causes daemon startup to abort with E-CFG-008."
- Product-owner: amend error-taxonomy.md E-CFG-008 message format to include the
  console-mode loopback-rejection variant (see canonical message format above).

---

### Ruling E — Write Deadline on Server Sends (CWE-400 / Residual Slowloris)

**Finding grounded in code:**
`sendJSON` (mgmt.go:353–361) calls `conn.Write(data)` with no write deadline set.
`conn.SetReadDeadline` is set before every `Decode` call, but `conn.SetWriteDeadline`
is never set before any `conn.Write` call. A client that completes authentication and
then stops reading its TCP/Unix socket causes the server's `conn.Write` to block
indefinitely, pinning the `MaxConcurrentConnections` semaphore slot (CWE-400 /
slot-exhaustion DoS). This is symmetric to the read-deadline gap closed by Ruling 1
(v1.2).

**Ruling:** Add `conn.SetWriteDeadline` before every `sendJSON` call.

Two timeout values:

| Send context | Timeout to use | Rationale |
|-------------|----------------|-----------|
| CHALLENGE send (step 3 of handshake) | `HandshakeTimeout` (10 s) | The client is expected to read the CHALLENGE immediately before beginning its own crypto. 10 s matches the read deadline budget. |
| AUTH_OK / AUTH_FAIL sends (steps 5a/5b) | `HandshakeTimeout` (10 s) | Same handshake phase; symmetric. |
| RPC response sends (step 6) | `RPCIdleTimeout` (30 s) | The client should be actively waiting for the response it just requested. 30 s is generous. |

**Implementation:** A write-deadline helper is the cleanest pattern since `sendJSON`
does not currently take a deadline parameter. Two options:

Option 1 — set deadline at each call site before calling `sendJSON`:
```go
// Before CHALLENGE send:
_ = conn.SetWriteDeadline(time.Now().Add(s.handshakeTimeout))
if err := sendJSON(conn, challenge); err != nil { return }
_ = conn.SetWriteDeadline(time.Time{}) // clear

// Before AUTH_OK send:
_ = conn.SetWriteDeadline(time.Now().Add(s.handshakeTimeout))
if err := sendJSON(conn, authOKMsg{...}); err != nil { return }
_ = conn.SetWriteDeadline(time.Time{}) // clear

// Before RPC response send:
_ = conn.SetWriteDeadline(time.Now().Add(RPCIdleTimeout))
if err := sendJSON(conn, resp); err != nil { return }
_ = conn.SetWriteDeadline(time.Time{}) // clear
```

Option 2 — extend `sendJSON` to accept a deadline:
```go
func sendJSONWithDeadline(conn net.Conn, v any, d time.Duration) error {
    _ = conn.SetWriteDeadline(time.Now().Add(d))
    defer func() { _ = conn.SetWriteDeadline(time.Time{}) }()
    return sendJSON(conn, v)
}
```

**Ruling:** Use Option 1 (inline at call site). Option 2 changes the `sendJSON`
signature and would require updating `sendAuthFail` as well. Option 1 is more explicit
about intent and does not require a signature change to a shared helper. The three
call sites in `handleConnection` are well-localized.

**ARCH-12 §7 amendment:** The per-connection deadline mandate in §7 is extended to
cover both read and write:

> **Amended §7 rule:** Before every `json.Decoder.Decode` call, set `conn.SetReadDeadline`.
> Before every `conn.Write` call (via `sendJSON`), set `conn.SetWriteDeadline`. After
> each operation completes, clear the corresponding deadline to `time.Time{}`.
> Write deadlines follow the same phase-based timeout values as read deadlines:
> `HandshakeTimeout` during the handshake phase; `RPCIdleTimeout` for RPC responses.

**Downstream changes required by Ruling E:**

- Story-writer: amend S-W5.01 to add AC-018: "Before every sendJSON call in
  handleConnection, conn.SetWriteDeadline is set (HandshakeTimeout for CHALLENGE/
  AUTH_OK/AUTH_FAIL; RPCIdleTimeout for RPC responses) and cleared after. Test:
  `TestMgmtServer_WriteDeadlineSet_AC018` — use net.Pipe; after AUTH_OK, stop reading
  on the client side; attempt RPC; verify server sends response without hanging
  indefinitely (response fails with deadline exceeded, not blocked)."
- Product-owner: amend BC-2.07.004 PC-1 to add:
  > Before every `conn.Write` call (`sendJSON`) the server sets `conn.SetWriteDeadline`
  > using the same phase-based timeout as the corresponding read deadline
  > (HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC response sends).
  > Write deadlines are cleared after each send to `time.Time{}`.

---

### Ruling F — Client Single-Timeout-Budget Design (Cross-Lens Clarification)

**Background:**
A Lens-3 adversarial reviewer flagged the sbctl client (S-6.03) for not re-arming a
per-RPC `RPCIdleTimeout` after each response read, and for not setting a write deadline
before the CHALLENGE_RESPONSE send — citing ARCH-12 §7's deadline discipline. This
produced a false convergence failure that reset BC-5.39.001.

**Ruling: The sbctl single-ctx-timeout-budget model IS the sanctioned design.**

The sbctl client is a one-shot CLI:
1. Dial the management socket.
2. Complete one auth handshake.
3. Send one RPC request.
4. Read one RPC response.
5. Exit.

This entire sequence runs under a single `context.WithTimeout(--timeout)` budget
(default in the story; see `client.go:296–340`). `Authenticate` sets one absolute
deadline derived from the ctx deadline (or `HandshakeTimeout` as fallback per
Ruling 2 / v1.2). The RPC send and read inherit the remaining ctx deadline from the
same context passed to `net.DialContext`.

**ARCH-12 §7's per-RPC RPCIdleTimeout re-arm and write-deadline-reset discipline
applies to the SERVER (`internal/mgmt`) ONLY, not to the one-shot sbctl client.**

Rationale: RPCIdleTimeout exists to reclaim slots from idle connections on the SERVER,
where the same `net.Conn` can receive many RPC requests over its lifetime (currently
one, but the architecture allows multi-RPC connections in future). The client exits
immediately after its single RPC; there is no idle gap to guard against.

#### Client vs. Server Deadline Model (normative note — mandatory section heading)

This section MUST be read by any reviewer before flagging client deadline behavior:

| Aspect | sbctl client (`cmd/sbctl`) | mgmt server (`internal/mgmt`) |
|--------|--------------------------|-------------------------------|
| Connection lifetime | One dial → one auth → one RPC → exit | Long-lived: accepts many connections, each potentially multi-RPC |
| Timeout mechanism | Single `context.WithTimeout` budget for entire operation | Per-phase deadlines: `HandshakeTimeout` for handshake reads/writes, `RPCIdleTimeout` for RPC reads/writes |
| Re-arm on each RPC | Not applicable (one RPC total) | Required: server re-arms `RPCIdleTimeout` before each subsequent RPC read (mgmt.go:575) |
| Write deadline | Not explicitly set; covered by ctx-derived dial deadline | Set before every `sendJSON` call (Ruling E) |
| Spec authority | BC-2.07.002, BC-2.07.003 | BC-2.07.004, ADR-012 §7 |

**Residual concern (non-blocking, guidance only):** A future caller that passes
`context.Background()` (no deadline) into `connectAndRun` would receive the
`HandshakeTimeout` (10 s) fallback for auth but have no deadline on the RPC phase.
Story-writers and implementers SHOULD ensure that any `connectAndRun` call site
derives a context with a deadline from the `--timeout` flag. This is currently
correct in the S-6.03 implementation but should be validated in S-W5.02's E2E
harness to prevent regression.

**No code change required by Ruling F.** The finding is closed. Any future adversarial
pass that re-flags the client's single-ctx-timeout model MUST cite this section as
the architectural decision and treat the finding as resolved.

---

## Product-Owner Handoff (v1.3 — Wave-5 Convergence Round-2 Rulings A–F)

This section supplements the v1.2 Product-Owner Handoff. Apply each item exactly as
written. Precedence: the items below supersede any conflicting v1.2 instructions for
the same postcondition.

### BC-2.07.004 Changes (Round-2)

**Precondition 3 amendment (Ruling A):** Replace PC-3 text with:
> 3. The daemon's own Ed25519 private key is available — either an ephemeral key
>    generated at startup via `ed25519.GenerateKey(rand.Reader)` (this wave: access
>    daemon in S-W5.01) or a persistent key loaded from `key_file` (S-6.02). A nil
>    or invalid key (len != `ed25519.PrivateKeySize` = 64) causes `mgmt.NewServer`
>    to panic at construction time, preventing any connection from being accepted.

**New PC-10 (Ruling B):**
> PC-10: `mgmt.Server.Serve` returns nil when `Shutdown` is called or the context
> is cancelled (normal daemon lifecycle). `Serve` returns a non-nil error only on
> an unexpected listener failure unrelated to shutdown. The `shuttingDown atomic.Bool`
> field mediates the distinction.

**New PC-11 (Ruling C):**
> PC-11: When an authenticated RPC request names an unregistered command, the server
> sends `{"type":"response","id":"<id>","ok":false,"error":{"code":"E-RPC-010",
> "message":"unknown command: <cmd>"},"data":null}`. The connection is NOT closed.

**New PC-12 (Ruling C):**
> PC-12: When a registered handler returns a non-nil error, the server sends
> `{"type":"response","id":"<id>","ok":false,"error":{"code":"E-RPC-011",
> "message":"<err>"},"data":null}`. The connection is NOT closed.

**PC-1 amendment (Ruling E — adds write deadline):** Append to the existing PC-1 text:
> Before every `sendJSON` call in `handleConnection`, the server sets
> `conn.SetWriteDeadline(time.Now().Add(d))` where `d` is `HandshakeTimeout` for
> handshake-phase sends (CHALLENGE, AUTH_OK, AUTH_FAIL) and `RPCIdleTimeout` for
> RPC response sends. The write deadline is cleared to `time.Time{}` after each send.

**EC-013 amendment (Ruling D — adds rejection predicate):** Append to EC-013:
> For console-mode TCP, any `management_socket` value whose host is not `127.0.0.1`,
> `[::1]`, or `localhost` causes daemon startup to abort with E-CFG-008:
> `"config error: management_socket: console mode requires a loopback address
> (127.0.0.1, [::1], or localhost); got: <address>"`. This check is enforced in
> `buildMgmtListener` (cmd/switchboard/mgmt_wire.go), not in `config.Validate()`.

### error-taxonomy.md Changes (Round-2)

**New E-RPC-010** — add to the RPC table:
> | E-RPC-010 | RPC | broken | — (in-band response) | "unknown command: \<command\>" | BC-2.07.004 PC-11 (Wave-5 Ruling C); emitted by `internal/mgmt` server in the JSON response envelope when an authenticated RPC names an unregistered command. Distinct from client-side E-RPC-001. The connection is not closed after this error. |

**New E-RPC-011** — add to the RPC table:
> | E-RPC-011 | RPC | broken | — (in-band response) | "\<handler error message\>" | BC-2.07.004 PC-12 (Wave-5 Ruling C); emitted by `internal/mgmt` server in the JSON response envelope when a registered handler returns a non-nil error. The message is the handler's error string verbatim. The connection is not closed after this error. Distinct from E-RPC-001 (client), E-RPC-010 (server unknown command). |

**E-CFG-008 message format extension (Ruling D):** Append to the existing E-CFG-008 row:
> Additionally: for console-mode TCP, `"config error: management_socket: console mode
> requires a loopback address (127.0.0.1, [::1], or localhost); got: \<address\>"` is
> emitted when the host is not loopback. This variant is generated by
> `buildMgmtListener` (cmd/switchboard/mgmt_wire.go), not by `config.Validate()`.

---

## Story-Writer Handoff (v1.3 — Wave-5 Convergence Round-2 Rulings A–F)

### S-W5.01 Changes (Round-2)

**New AC-015 (Ruling A.1):**
> AC-015 (traces to BC-2.07.004 PC-3, Ruling A) `runAccess` generates an ephemeral
> Ed25519 private key via `ed25519.GenerateKey(rand.Reader)` immediately before calling
> `startMgmtServer`. If `GenerateKey` returns an error, `runAccess` returns it
> immediately (daemon startup aborts). Test: `TestRunAccess_GeneratesEphemeralKey` —
> mock `startMgmtServer`; verify it is called with a non-nil, 64-byte `daemonPrivKey`.

**New AC-016 (Ruling A.2):**
> AC-016 (traces to BC-2.07.004 PC-3, Ruling A) `mgmt.NewServer` panics immediately
> if `len(daemonKey) != ed25519.PrivateKeySize` (64 bytes). A nil `daemonKey` produces
> `len == 0`, triggering the panic. Test: `TestNewServer_PanicsOnNilKey` — use
> `defer func(){ recover() }()` to verify panic; pass nil daemonKey to `NewServer`.
> Test: `TestNewServer_PanicsOnShortKey` — pass a 32-byte key (ed25519.PublicKey size,
> common mistake); verify panic.

**New AC-017 (Ruling B):**
> AC-017 (traces to BC-2.07.004 PC-10, Ruling B) `mgmt.Server.Serve` returns nil
> when `Shutdown` is called. `Serve` returns nil when ctx is cancelled.
> Test: `TestServe_ReturnsNilOnShutdown` — start server via `net.Pipe` listener;
> call `Shutdown(ctx)`; verify `Serve` return value `== nil`.
> Test: `TestServe_ReturnsNilOnCtxCancel` — start server; cancel ctx; verify `Serve`
> return value `== nil`.

**AC-007 amendment (Ruling C):** In the existing AC-007 (RPC dispatch), replace all
references to `E-RPC-001` (server "unknown command") with `E-RPC-010`, and remove any
reference to `E-RPC-002`. Add to the test table:
> Row (unknown command): send authenticated RPC `{"command":"not.registered",...}`;
> expect response `{"ok":false,"error":{"code":"E-RPC-010","message":"unknown command: not.registered"}}`.
> Row (handler error): register a handler that returns `errors.New("boom")`;
> expect `{"ok":false,"error":{"code":"E-RPC-011","message":"boom"}}`.

**New AC-018 (Ruling E):**
> AC-018 (traces to BC-2.07.004 PC-1, Ruling E) Before every `sendJSON` call in
> `handleConnection`, `conn.SetWriteDeadline` is called with `HandshakeTimeout`
> (for CHALLENGE/AUTH_OK/AUTH_FAIL sends) or `RPCIdleTimeout` (for RPC response
> sends); the deadline is cleared after each send.
> Test: `TestMgmtServer_WriteDeadlineSet_AC018` — use `net.Pipe`; after auth, stop
> reading on the client pipe; send one RPC; verify the server's sendJSON fails within
> RPCIdleTimeout (not blocked indefinitely).

**AC-014 amendment (Ruling D):** Add to the existing AC-014:
> The TCP/console path in `buildMgmtListener` validates that the management socket
> address has a loopback host (127.0.0.1, [::1], localhost) before calling
> `net.Listen`. Addresses with host `0.0.0.0`, `::`, or any non-loopback IP are
> rejected with `E-CFG-008`.
> Test: `TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback` — call with "0.0.0.0:9091";
> verify error string contains "E-CFG-008".
> Test: `TestBuildMgmtListener_ConsoleTCP_AcceptsLoopback127` — "127.0.0.1:9091" succeeds
> (may fail to bind in CI; use a port that is free or mock net.Listen if needed).

---

## Risk Mitigations

| Risk | Mitigation |
|------|-----------|
| CWE-400 (unbounded socket read) | io.LimitReader(MaxMessageBytes = 64 KiB) on every socket read, client and server (ADR-012 §6) |
| CWE-400 (connection read blocking indefinitely) | Per-connection read deadlines: HandshakeTimeout=10s, RPCIdleTimeout=30s via conn.SetReadDeadline before every Decode (ADR-012 §7) |
| CWE-400 (write-blocking slowloris) | Per-send write deadlines: conn.SetWriteDeadline before every sendJSON call; HandshakeTimeout for handshake sends, RPCIdleTimeout for RPC responses (Ruling E / v1.3) |
| CWE-770 (fd/goroutine exhaustion from connection flood) | MaxConcurrentConnections=128 semaphore in accept loop; transient Accept errors backed off exponentially (ADR-012 §8) |
| CWE-276 (Unix socket world-accessible) | syscall.Umask(0177) before net.Listen creates socket at 0600 atomically; no TOCTOU window; console TCP bound to 127.0.0.1 only (Ruling D: buildMgmtListener validates loopback host before net.Listen) |
| E-ADM-010 oracle (key enumeration) | AUTH_FAIL returns same message for unrecognized and wrong-signature keys |
| DI-002 (private key transit) | Operator private key used only for local Sign(); never serialized or sent |
| Nil daemon key (remote panic DoS) | NewServer panics if len(daemonKey) != ed25519.PrivateKeySize (Ruling A.2); access daemon generates ephemeral key at startup (Ruling A.1); nil key can never reach a connection goroutine |
| Replay across connections | Fresh nonce per connection (crypto/rand.Read(32)); post-auth structural guard rejects any re-authentication attempt on an authenticated connection |
| E-NET-001 overload (key-load and RPC errors misattributed to network) | E-CFG-010 for key-load failures; E-RPC-001 for post-auth dispatch failures; E-NET-001 reserved for dial/connect unreachable |
| Server RPC error code collision (E-RPC-001 used on both sides) | Server uses E-RPC-010 (unknown command) and E-RPC-011 (handler error); E-RPC-001 is exclusively the client-side sbctl code (Ruling C / v1.3) |
| Serve returns non-nil on graceful shutdown | shuttingDown atomic.Bool set before ln.Close(); Serve returns nil when shuttingDown or ctx.Err() != nil (Ruling B / v1.3) |
| daemonVersion hardcoded "dev" | NewServer takes daemonVersion string injected from cmd/switchboard.version (ldflags); "dev" is the unreleased sentinel only |
| init() coupling | Handler registry is constructor-injected; no package-level init() |
| Goroutine leak | mgmt.Serve goroutine is WaitGroup-tracked per ARCH-01 §Goroutine WaitGroup Contract |
| Config schema drift | new fields added to Config struct with Validate() coverage; BC-2.09.003 amendment recommended |

---

## Product-Owner Handoff (v1.2 — adversarial review rulings)

This section is the authoritative instruction set for the product-owner. Apply each
item exactly as written. All items trace to the rulings above.

### BC-2.07.004 Changes

**PC-1 amendment (Ruling 1):** Add to the existing PC-1 text:
> The server calls `conn.SetReadDeadline(time.Now().Add(mgmt.HandshakeTimeout))`
> (default 10 s) immediately after sending the CHALLENGE and before any blocking
> read. On deadline expiry the connection is closed with E-ADM-010.

**PC-3 replacement (Ruling 7):** Replace the current PC-3 entirely with:
> PC-3 — Post-auth protocol violation rejected: After a successful auth handshake
> on connection C, any further `"type":"challenge_response"` message on C is treated
> as a protocol violation. The server closes the connection with E-ADM-010 and logs
> a security event. The server maintains a per-connection `authenticated` boolean;
> no nonce set is stored after auth. Cross-connection replay is prevented by the
> fresh nonce issued on every new connection (`crypto/rand.Read(32)`).

**PC-7 amendment (Ruling 6):** Update the AUTH_OK postcondition:
> `"daemon_version"` in the AUTH_OK message MUST equal the value of
> `cmd/switchboard.version` (injected by ldflags at build time). The sentinel
> value `"dev"` is used only for untagged/unreleased builds. Hardcoding `"dev"`
> in production is a defect.

**New EC-001 amendment (Ruling 1):** Update the existing EC-001 row:
> EC-001 | Client connects and sends nothing (no CHALLENGE_RESPONSE) | Server sends
> CHALLENGE, applies HandshakeTimeout read deadline (default 10 s) on the
> subsequent read. On expiry: sends E-ADM-010 + closes connection. The server does
> not hang indefinitely. HandshakeTimeout is `mgmt.HandshakeTimeout = 10 * time.Second`.

**New EC-012 (Ruling 3):** Add the following row to the Edge Cases table:
> EC-012 | Concurrent connections exceed MaxConcurrentConnections (default 128) |
> New Accept() calls block in the accept-loop semaphore until a slot frees.
> Connections queue in the OS accept backlog. No new goroutines are spawned beyond
> the limit. No fd exhaustion or goroutine leak. Transient Accept errors (EMFILE etc.)
> trigger exponential backoff (5 ms–1 s) rather than server shutdown.

**New EC-013 (Ruling 4):** Add the following row to the Edge Cases table:
> EC-013 | Unix socket created with world-readable permissions | The management
> socket MUST be created with permissions 0600 (owner read/write only) via
> `syscall.Umask(0177)` before `net.Listen`. A world-accessible socket allows any
> local user to connect and attempt authentication — the permission is the first
> defense before auth. Console TCP is bound to 127.0.0.1 only; `0.0.0.0` binding
> is forbidden.

**New Invariant (Ruling 4):** Add to the Invariants list:
> 7. The Unix management socket has permissions 0600 at creation time. The process
>    umask is set to 0177 immediately before `net.Listen` and restored afterward.
>    Console TCP is bound to 127.0.0.1; no management listener binds to `0.0.0.0`.

**VP-065 rewording (Ruling 7):** Update the VP-065 property description in the
Verification Properties table:
> VP-065 | Post-auth structural guard: after AUTH_OK, a subsequent
> `"type":"challenge_response"` message causes connection close with E-ADM-010 |
> integration

### BC-2.07.003 Changes (Ruling 5)

The current BC-2.07.003 PC-1 correctly scopes `E-NET-001` to "daemon unreachable."
No PC change is needed for BC-2.07.003 itself.

**New invariant to add to BC-2.07.003:**
> 4. `E-NET-001` is emitted ONLY on `net.Dial` / `net.DialContext` failure
>    (daemon unreachable). Key-load failures produce `E-CFG-010`; post-auth RPC
>    dispatch failures produce `E-RPC-001`. These three failure modes are
>    distinct and MUST NOT share an error code.

**New edge cases to add to BC-2.07.003:**
> EC-005 | Key file at `--key` path does not exist, is larger than 64 KiB, is
> malformed (not valid OpenSSH PEM), or contains a non-Ed25519 key type | sbctl
> prints `E-CFG-010 "key load failed: <path>: <reason>"` to stderr and exits 1.
> No connection attempt is made. No stdout output.
>
> EC-006 | Authentication succeeded (AUTH_OK received) but the subsequent RPC
> request fails (server returns `"ok":false`, or response decode fails, or
> connection drops mid-RPC) | sbctl prints `E-RPC-001 "rpc failed: <command>:
> <reason>"` to stderr and exits 1. No stdout output.

### error-taxonomy.md Changes (Ruling 5)

**New error code E-CFG-010** — add to the CFG table:
> | E-CFG-010 | CFG | broken | 1 | "key load failed: \<path\>: \<reason\>" |
> BC-2.07.003 EC-005 (Wave-5 Ruling 5); emitted by sbctl when the `--key` file
> is absent, oversized (> 64 KiB), not valid OpenSSH PEM, or contains a
> non-Ed25519 key. Distinct from E-CFG-008 (management_socket) and E-CFG-009
> (authorized_operator_keys PEM). Free slot in CFG family. |

**New error category RPC** — add a new section header and table:
> ### RPC — Remote Procedure Call
>
> | Error Code | Category | Severity | Exit Code | Message Format | FM/DEC Source |
> |-----------|----------|----------|-----------|----------------|---------------|
> | E-RPC-001 | RPC | broken | 1 | "rpc failed: \<command\>: \<reason\>" |
> BC-2.07.003 EC-006 (Wave-5 Ruling 5); emitted by sbctl when an authenticated
> RPC request fails: the server returns `"ok":false`, the response cannot be
> decoded, or the connection drops after AUTH_OK. Distinct from E-NET-001
> (unreachable before connection), E-ADM-010 (authentication failure), and
> E-CFG-010 (key load failure). Opens the RPC error family. |

**E-NET-001 note amendment** — add a precision note to the E-NET-001 row:
> Scope clarification (Wave-5 Ruling 5): E-NET-001 is emitted ONLY on
> `net.Dial`/`net.DialContext` failure — i.e., the daemon truly cannot be
> reached. Key-load failures (before dial) are E-CFG-010; post-auth dispatch
> failures (after AUTH_OK) are E-RPC-001.

---

## Story-Writer Handoff (v1.2 — adversarial review rulings)

Apply these changes to stories S-6.03 and S-W5.01.

### S-6.03 Changes

**AC-002 signature change (Ruling 2):** Replace the current `Authenticate` signature
in AC-002 with:
```go
func Authenticate(ctx context.Context, conn net.Conn, privKey ed25519.PrivateKey) error
```
Update the AC-002 step list to read:
> 1. Compute effective deadline: if `ctx.Deadline()` is set use it; else
>    `time.Now().Add(mgmt.HandshakeTimeout)`. Call `conn.SetReadDeadline(deadline)`.
> 2. Read CHALLENGE message via `json.NewDecoder(io.LimitReader(conn, 1<<16))`.
> 3. Decode `nonce_bytes` from the base64url `nonce` field; return error if absent
>    or not valid base64url-encoded 32 bytes.
> 4. Send `{"type":"challenge_response","nonce_sig":"<base64url sig>","pubkey":"<base64url pubkey>"}`.
> 5. Read AUTH_OK or AUTH_FAIL response (deadline still active).
> 6. Return `nil` **only** if `{"type":"auth_ok"}` was received and decoded.
> 7. Return non-nil error for: connection error, malformed CHALLENGE, AUTH_FAIL,
>    deadline exceeded, any other response type, truncated stream, oversized response.

Update the `TestAuthenticate_FailClosed_VP067` test to add sub-case for deadline
expiry (server sends nothing): verify non-nil error returned.

Update `cmd/sbctl/main.go` call site to pass a context derived from the
`--timeout` flag:
```go
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
// net.DialContext(ctx, ...) for dial; pass same ctx to Authenticate(ctx, conn, key)
```

**AC-003 amendment (Ruling 5):** The test `TestSbctl_AuthFailure_ExitsOneWithEADM010`
should remain focused on AUTH_FAIL → E-ADM-010. Add a separate test:
`TestSbctl_KeyLoadFailure_ExitsOneWithECFG010` — supply a key path that does not
exist; verify stderr contains "E-CFG-010" and exit code is 1.

**AC-004 amendment (Ruling 5):** No change to E-NET-001 test. Add a new test
`TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001` — mock server that completes
AUTH_OK then returns `{"ok":false,"error":{"code":"E-RPC-001",...}}`; verify
sbctl exits 1 with "E-RPC-001" on stderr.

**New EC-005 (Ruling 5):** Add to S-6.03 edge cases:
> EC-005 | Key file at `--key` path does not exist or is malformed | E-CFG-010
> "key load failed: \<path\>: \<reason\>"; exit 1; no connection attempt made.

**New EC-006 (Ruling 5):** Add to S-6.03 edge cases:
> EC-006 | AUTH_OK received but RPC dispatch returns error | E-RPC-001
> "rpc failed: \<command\>: \<reason\>"; exit 1.

### S-W5.01 Changes

**AC-001 amendment (Ruling 1):** Add to the test `TestMgmtServer_IssuesChallengeFirst_AC001`:
verify that the server closes the connection if no CHALLENGE_RESPONSE is received
within `mgmt.HandshakeTimeout` (10 s). Use a test that supplies a 50 ms timeout
to avoid slow tests.

**AC-002 amendment (Ruling 1):** Add the following sub-case to
`TestMgmtServer_RejectsUnauthenticated_VP064`:
> (d) client connects, receives CHALLENGE, then waits without responding — verify
>     server closes connection after HandshakeTimeout with no goroutine leak.

**AC-003 rewording (Ruling 7):** Replace the AC-003 text with:
> A connection that has completed a successful auth handshake (is in authenticated
> state) and then sends a second `{"type":"challenge_response",...}` message
> receives AUTH_FAIL (E-ADM-010) + connection close. The server tracks authentication
> state via a per-connection boolean `authenticated`; no nonce set is used.
> Test: `TestMgmtServer_PostAuthChallengeResponseRejected_VP065` — authenticate on
> C1; then send another CHALLENGE_RESPONSE on C1; verify AUTH_FAIL + close.

**AC-007 amendment (Ruling 6):** Add assertion to
`TestMgmtServer_AuthOK_DispatchesRPC_AC007`:
> Verify that the `daemon_version` field in the AUTH_OK response matches the
> `daemonVersion` string passed to `NewServer`. Test should pass `"0.1.0-test"`
> as daemonVersion and assert `auth_ok.daemon_version == "0.1.0-test"`. Assert
> that passing an empty `daemonVersion` is rejected by `NewServer` (panic or
> error — document the chosen enforcement).

**New AC-013 (Ruling 3):** Add to S-W5.01:
> AC-013 (traces to ADR-012 §8 — connection cap) `mgmt.Server` does not spawn
> more than `MaxConcurrentConnections` (128) simultaneous connection goroutines.
> When the limit is reached, additional `Accept` calls block rather than spawning
> new goroutines. Test: `TestMgmtServer_ConnectionCap_AC013` — use a semaphore-size
> of 3 (constructed via `NewServer` with a `WithMaxConnections(3)` option or
> by the semaphore channel size directly); open 3 connections that hold the server
> busy (do not send any data so they stay in handshake); attempt a 4th connection
> via `net.Pipe` and verify it does not immediately get a CHALLENGE (server is
> at capacity). Release one connection; verify 4th then gets CHALLENGE.

**New AC-014 (Ruling 4):** Add to S-W5.01:
> AC-014 (traces to ADR-012 §Unix Socket Permissions — CWE-276) The wiring code
> in `cmd/switchboard` that calls `net.Listen("unix", ...)` MUST precede the call
> with `syscall.Umask(0177)` and restore the old umask afterward. Test:
> `TestDaemonWiring_UnixSocketPermissions_AC014` — create a temp socket path;
> call the wiring helper; stat the resulting socket file; assert permissions are
> 0600. (Console TCP path does not need this test but must assert
> `ManagementSocket` begins with "127.0.0.1" for console mode.)

**daemonVersion in `NewServer` (Ruling 6):** Update the Task 7 item in S-W5.01:
> `func NewServer(ln net.Listener, daemonKey ed25519.PrivateKey, ops *OperatorKeySet, handlers []Handler, daemonVersion string) *Server`
> The `daemonVersion` parameter is embedded in AUTH_OK. Pass the build-injected
> `version` variable from `cmd/switchboard/main.go`. The value `"dev"` is
> acceptable for tests; production builds inject semver via ldflags.

---

## Wave-5 Convergence Rulings (Round-3) — v1.4

These rulings were produced after a fresh 6-pass (3 lenses × 2 stories) adversarial
convergence round on S-W5.01 and S-6.03. The convergence counter reset because the
fresh passes surfaced new Critical/High findings, none of which were flagged or settled
in rounds 1–2. Rulings A–F remain intact and authoritative. Each ruling below is citable
by stable heading (`### Ruling G` …) in all downstream artifacts.

---

### Ruling G — Serve Accept-Error Predicate: Missing `ctx.Err() != nil` Conjunct (HIGH)

**Finding grounded in code:**
`internal/mgmt/mgmt.go` line 253 (S-W5.01 worktree HEAD 76d39a8) reads:
```go
if s.shuttingDown.Load() || errors.Is(err, net.ErrClosed) {
```
The canonical predicate from Ruling B is:
```go
if s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil) {
```
The `&& ctx.Err() != nil` conjunct is absent.

**Why this matters (SOUL #4 — no silent failure):** `net.ErrClosed` is returned by
`Accept` whenever the underlying listener fd is closed — not only during intentional
shutdown. If an external actor closes the fd (OS-level fd theft, a bug in the wiring
layer calling `ln.Close()` on the wrong listener, or a platform-level socket error that
returns `net.ErrClosed`) while `ctx` is still alive and `Shutdown` has never been called,
`s.shuttingDown.Load()` is false and `errors.Is(err, net.ErrClosed)` is true. Under the
current predicate, `Serve` returns `nil` — the management plane dies silently. The daemon
continues running with no management plane and no error surfaced to the caller. This
violates PC-10's non-nil guarantee for unexpected listener failure and SOUL #4.

**Ruling:**

The canonical, authoritative Accept-error predicate for `Serve` is:

```go
// Intentional shutdown: listener closed by Shutdown() or by ctx-watcher goroutine.
// The shuttingDown flag is set before ln.Close() in both paths.
// The (errors.Is && ctx.Err() != nil) arm guards the edge case where the listener
// close and the flag-store race — in that window ctx cancellation has occurred even
// if the flag read misses it (conservative: both checks must be true).
if s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil) {
    s.connWG.Wait()
    return nil
}
// Fatal accept error: unexpected close, fd stolen, or other listener failure.
// Not caused by intentional shutdown. Return non-nil so the caller can log and
// restart the management plane.
s.connWG.Wait()
return err
```

This is a **one-line implementer fix**: add `&& ctx.Err() != nil` as a conjunct around
the `errors.Is` check. The `s.shuttingDown.Load()` arm is unchanged and correct.

**Test obligation — unexpected-close path (new, mandatory):**

The existing VP-069 test vehicles (`TestServe_ReturnsNilOnShutdown` /
`TestServe_ReturnsNilOnCtxCancel` from AC-017) drive close THROUGH `Shutdown()`
or context cancellation respectively; neither covers the unexpected-close path.
A new test is required:

> `TestServe_ReturnsErrOnUnexpectedListenerClose` (traces to VP-069 / BC-2.07.004 PC-10)
>
> Setup: create a server with a real `net.Listener` (e.g., `net.Listen("tcp",
> "127.0.0.1:0")`); start `Serve` with a non-cancelled context (use
> `context.Background()`); do NOT call `Shutdown` and do NOT cancel the context.
> Action: close the listener directly from the test goroutine —
> `ln.Close()` — with the context still live and `shuttingDown` still false.
> Assertion: `Serve` returns a **non-nil** error (any error, not nil).
>
> This test fails under the current code (returns nil) and passes after the fix.
> It directly exercises the BC-2.07.004 PC-10 clause: "returns a non-nil error
> only on an unexpected listener failure unrelated to shutdown."

**Downstream changes required by Ruling G:**

- **Implementer (S-W5.01):** Change line 253 in `internal/mgmt/mgmt.go` from
  `errors.Is(err, net.ErrClosed)` to `(errors.Is(err, net.ErrClosed) && ctx.Err() != nil)`.
  One-line change; no structural refactor needed.
- **Story-writer (S-W5.01):** Add to AC-017: "A third sub-case:
  `TestServe_ReturnsErrOnUnexpectedListenerClose` — listener closed directly (not
  via Shutdown, not via ctx cancel); assert Serve returns non-nil. This closes the
  unexpected-close gap in VP-069 coverage."
- **Test-writer (S-W5.01):** Implement `TestServe_ReturnsErrOnUnexpectedListenerClose`
  as specified above. This is a RED test before the implementer applies the fix.
- **Product-owner:** No BC change required. PC-10 already states "non-nil error only
  on an unexpected listener failure unrelated to shutdown." The code was failing to
  honour that clause; the ruling restores conformance.

---

### Ruling H — Dead `case <-done:` Branches: Design Adjudication (MEDIUM)

**Finding grounded in code:**
`mgmt.go` has the following `case <-done:` arms that Ruling B (v1.3) called "dead code"
and required removal:

1. Pre-Accept semaphore select (line ~240):
   ```go
   select {
   case s.sem <- struct{}{}:
   case <-done:       // Ruling B said REMOVE; implementer kept it
       s.connWG.Wait()
       return nil
   }
   ```

2. Post-Accept-error select (lines ~258-264):
   ```go
   select {
   case <-done:       // Ruling B said REMOVE; implementer kept it
       s.connWG.Wait()
       return nil
   default:
   }
   ```

Ruling B said remove these, replacing the pre-Accept arm with `case <-ctx.Done():`.
The implementer kept the `case <-done:` arms but also added the ctx-watcher goroutine
that stores `shuttingDown = true` and closes the listener on `ctx.Done()`. This creates
a spec-vs-code contradiction that causes fresh reviewers to re-flag the code on every
pass.

**Design analysis:**

The `done` channel is `defer close(done)` — it closes when `Serve` returns. The
`case <-done:` arms therefore can only fire AFTER `Serve` has already returned, which
is impossible. They are structurally dead:

- In the pre-Accept select: `case <-done:` fires only when `Serve` returns. But `Serve`
  is blocked on this select. Deadlock by definition — this case can never be selected
  while Serve is running.
- In the post-Accept-error select: same logic. The `default:` branch is always taken
  because `done` is open while Serve is executing.

The ctx-cancel path is already handled: the ctx-watcher goroutine closes the listener
→ `Accept` returns an error → the `s.shuttingDown.Load()` check (or after Ruling G,
the `ctx.Err() != nil` arm) returns nil. The `case <-done:` branches add no coverage
of the ctx-cancel path.

**Ruling: REAFFIRM Ruling B — remove `case <-done:` branches as written.**

The as-built design is NOT correct: the `case <-done:` arms are unreachable dead code
that mislead reviewers into thinking there is a second shutdown signal path when there
is not. Keeping them generates a spec-vs-code contradiction that will reset the
convergence counter on every future pass.

The correct structure, restating Ruling B with precision:

```go
// Pre-Accept select: acquire semaphore; bail on ctx cancel.
select {
case s.sem <- struct{}{}:
    // slot acquired — proceed to Accept
case <-ctx.Done():
    // Context cancelled before we even tried to Accept.
    // The ctx-watcher goroutine will also close the listener shortly,
    // but we exit the loop here rather than waiting for Accept to fail.
    s.connWG.Wait()
    return nil
}
```

The post-Accept-error non-default select is eliminated entirely — the
`s.shuttingDown.Load()` check at line 253 (corrected per Ruling G) subsumes it.

**Amendment to Ruling B:** Ruling B's statement that "The dead `case <-done:` branches
in the accept loop MUST be removed" was correct and remains in force. The as-built
implementation did not apply this instruction. The implementer must apply it now.
Specifically:

- Replace `case <-done:` in the pre-Accept semaphore select with `case <-ctx.Done():`.
- Remove the post-Accept-error `select { case <-done: ... default: }` block entirely.
  Its intent (detect shutdown) is already handled by the `s.shuttingDown.Load()` check
  immediately preceding it.

The backoff select (during transient-error exponential backoff) may retain a
`case <-done:` arm ONLY if it is guarded so it can actually fire — but in that context
the ctx-watcher goroutine will close the listener and the backoff select's
`case <-time.After(backoff):` will unblock the loop, then the next Accept will fail
and the shuttingDown check returns nil. Therefore the cleanest approach is to replace
all backoff-select `case <-done:` arms with `case <-ctx.Done():` as well, which fires
as soon as context is cancelled without waiting for the backoff timer. This is the
correct design.

**Goroutine leak guarantee:** The ctx-watcher goroutine exits via `case <-done:` when
Serve returns. With `defer close(done)` in place, this is correct and must be retained —
the ctx-watcher goroutine's own `case <-done:` is NOT dead code because it exits by
that signal when Serve finishes normally. Only the `case <-done:` arms in the accept
loop (which is INSIDE Serve) are dead.

**Downstream changes required by Ruling H:**

- **Implementer (S-W5.01):**
  1. In the pre-Accept semaphore select, replace `case <-done:` with `case <-ctx.Done():`.
  2. Delete the post-Accept-error `select { case <-done: s.connWG.Wait(); return nil; default: }` block.
  3. In the backoff `select`, replace `case <-done:` with `case <-ctx.Done():`.
  4. Do NOT change the ctx-watcher goroutine's own `case <-done:` — that one is correct.
- **Story-writer (S-W5.01):** Amend AC-017 text to read:
  > "The pre-Accept semaphore select uses `case <-ctx.Done():` (not `case <-done:`).
  > The dead post-Accept-error `select { case <-done: ...; default: }` block is absent.
  > Backoff selects use `case <-ctx.Done():`. The ctx-watcher goroutine's
  > `case <-done:` is retained — it exits the watcher when Serve returns."
- **Product-owner:** No BC change required. This is an implementation-correctness fix.

---

### Ruling I — `connWG.Add`/`Wait` Race + Track-After-Spawn Drain Gap (HIGH)

**Finding grounded in code:**
`mgmt.go` lines 292–298:
```go
s.connWG.Add(1)               // line 292: Add in accept loop
go func() {
    defer s.connWG.Done()
    defer func() { <-s.sem }()
    s.trackConn(conn)          // line 296: track INSIDE goroutine (after spawn)
    defer s.untrackConn(conn)
    s.handleConnection(ctx, conn)
}()
```

Two coupled defects:

**Defect I-1 — `connWG.Add` after `connWG.Wait` race (sync.WaitGroup misuse):**
`Shutdown` calls `connWG.Wait()` (line 316–323). A connection that is accepted
between `Shutdown` setting `shuttingDown = true` and the Accept loop detecting the
flag (i.e., a connection in the accept window) can cause `connWG.Add(1)` to be called
after `Wait` has observed a zero counter and returned — which is a sync.WaitGroup
panic. This window is:
1. `Shutdown` calls `s.shuttingDown.Store(true)`.
2. `Shutdown` calls `s.ln.Close()`.
3. Accept loop has already called `Accept()` and received a valid `conn` (before
   `ln.Close()` took effect).
4. Accept loop reaches `connWG.Add(1)`.
5. Meanwhile, `Shutdown` calls `closeAllConns()` then `connWG.Wait()`.
6. If the prior in-flight count was 1 and that goroutine just called `connWG.Done()`,
   `Wait` sees zero and returns.
7. Accept loop's `connWG.Add(1)` fires after `Wait` returned from zero → **panic**.

**Defect I-2 — track-after-spawn drain gap:**
`trackConn` is called INSIDE the goroutine. `closeAllConns` takes a snapshot of
`s.conns` under the mutex. If `closeAllConns` runs BEFORE the newly spawned goroutine
calls `trackConn`, the connection is not in the snapshot and is NOT force-closed. The
goroutine then blocks in `handleConnection` (waiting on a read from the non-responsive
connection), and `connWG.Wait()` in `Shutdown` blocks until `HandshakeTimeout` (10 s)
expires. This blows the `runAccess` 2-second shutdown budget (access.go:182).

**Ruling:**

The correct ordering is:

```go
// Check shuttingDown AFTER receiving conn but BEFORE Add/track.
// This is the safe window: we hold a valid conn but have not committed the
// goroutine to the WaitGroup yet. If we are shutting down, close the conn
// and release the semaphore slot rather than spawning.
if s.shuttingDown.Load() {
    _ = conn.Close()
    <-s.sem  // release pre-acquired semaphore slot
    continue // loop will hit the shuttingDown check at Accept-error and exit
}
// Register the connection in the track set BEFORE spawning so closeAllConns
// always sees it (Defect I-2 fix). The goroutine only calls untrackConn.
s.trackConn(conn)
s.connWG.Add(1)               // Add BEFORE spawn (Defect I-1 fix: no Add-after-Done-zero)
go func() {
    defer s.connWG.Done()
    defer s.untrackConn(conn)
    defer func() { <-s.sem }()
    s.handleConnection(ctx, conn)
}()
```

**Canonical PC-10 drain contract clarification:**

The drain contract for `Serve` on intentional shutdown is:
- `Shutdown` sets `shuttingDown = true`, closes the listener, calls `closeAllConns`
  (force-closing all tracked connections), then calls `connWG.Wait()`.
- `connWG.Wait()` completes when all goroutines spawned BEFORE the shutdown window
  call `connWG.Done()`. Any connection accepted in the shutdown window is rejected at
  the `shuttingDown.Load()` check BEFORE `connWG.Add(1)` — so it never enters the WG.
- The drain MUST complete within the caller's shutdown budget. The access daemon's
  budget is 2 s (access.go:182). With `closeAllConns` force-closing connections and
  `HandleConnection` exiting on closed conn, drain completes in sub-millisecond after
  `closeAllConns`.

**Test obligations (new, mandatory):**

> `TestServe_ShutdownWindowNoAddAfterWaitPanic` (traces to BC-2.07.004 PC-10 drain
> contract, Ruling I — Defect I-1)
>
> Runs under `go test -race`. Setup: create a server with `MaxConcurrentConnections=1`
> so the accept loop has minimal concurrency. Spawn a goroutine that dials the server
> in a tight loop, while simultaneously calling `Shutdown` from the test goroutine.
> Repeat 100 iterations. Assertion: no panic; race detector reports no WaitGroup
> misuse. (The test is primarily a race-detector test — if `connWG.Add` can race
> `connWG.Wait`, `-race` will catch it.)
>
> `TestServe_DrainCompletesWithinBudget` (traces to BC-2.07.004 PC-10 drain
> contract, Ruling I — Defect I-2)
>
> Setup: create a server; connect a client that completes auth and then stalls
> (stops reading). Call `Shutdown` with a 2-second context. Assertion: `Shutdown`
> returns within 2 s (not 10 s / HandshakeTimeout). The force-close via
> `closeAllConns` must unblock `handleConnection` quickly.

**BC-2.07.004 PC-10 clarification needed (product-owner):**
PC-10 currently says "`Serve` returns nil when `Shutdown` is called." It should also
specify the drain ordering guarantee: "Connections accepted after `Shutdown` is called
are closed immediately without entering the WaitGroup. Connections already in-flight
are force-closed by `closeAllConns` before `connWG.Wait`."

**Downstream changes required by Ruling I:**

- **Implementer (S-W5.01):** Apply the three-part fix in `Serve`:
  1. After a successful `Accept`, check `s.shuttingDown.Load()` and close+continue
     if true (drop the connection in the shutdown window).
  2. Call `s.trackConn(conn)` BEFORE `s.connWG.Add(1)` and BEFORE `go func()`.
  3. Remove `s.trackConn(conn)` from inside the goroutine body; retain only
     `defer s.untrackConn(conn)` inside the goroutine.
- **Story-writer (S-W5.01):** Amend AC-017 to add: "Connections accepted during the
  shutdown window (after `shuttingDown` is set but before the Accept loop exits) are
  closed immediately without entering `connWG`. `trackConn` is called before
  `connWG.Add` and before the goroutine spawn. No Add-after-Wait panic is possible."
  Add the two new test obligations above as required sub-cases of AC-017.
- **Test-writer (S-W5.01):** Implement `TestServe_ShutdownWindowNoAddAfterWaitPanic`
  and `TestServe_DrainCompletesWithinBudget` as specified. Both are RED before the
  implementer fix.
- **Product-owner:** Amend BC-2.07.004 PC-10 to add the drain ordering guarantee as
  stated above. No new PC number needed — this is a precision amendment to PC-10.

---

### Ruling J — Management Server Start Failure: Fatal vs. Degraded (HIGH)

**Finding grounded in code:**
`cmd/switchboard/access.go` lines 143–148:
```go
mgmtSrv, mgmtErr := startMgmtServer(ctx, &mgmtWG, cfg, "access", daemonPriv, nil)
if mgmtErr != nil {
    // Log but do not abort: management server failure is non-fatal for the
    // access data-plane in this wave ...
    fmt.Fprintf(stderr, "mgmt: failed to start management server: %v\n", mgmtErr)
}
```
The access daemon runs the data plane with NO management plane when `startMgmtServer`
fails. BC-2.07.004 Invariant 7 / EC-013 say that a config-class error (non-loopback
console address, bad socket perms) MUST abort startup.

**Ruling:**

Mgmt-server start failures are classified into two categories with different policies:

**Class 1 — Config/validation errors (FATAL — abort startup):**
These errors indicate a misconfigured or insecure deployment. The operator must fix
the config; running a data plane without management defeats operational safety.

| Error | Examples | Policy |
|-------|---------|--------|
| E-CFG-008 | Non-loopback console TCP address; empty management_socket | Fatal: return error → daemon exits non-zero |
| E-CFG-009 | Malformed operator public key PEM | Fatal: return error |
| NewServer construction panic | nil key, empty version (caught by recover) | Fatal: return error |
| Key generation failure | `ed25519.GenerateKey` error (already fatal in AC-015) | Fatal: already correct |

**Class 2 — Transient bind/OS errors (ALSO FATAL for access mode):**
After deliberation: for access mode in this wave, even a transient bind failure
(EADDRINUSE, permission denied on socket path, etc.) is treated as fatal. Rationale:
the management plane is not an optional feature — it is the operator's sole control
channel. Running an access daemon with no management plane is worse than not running
it; an operator cannot diagnose or recover from a broken daemon they cannot connect to.

**Policy:** `startMgmtServer` failure of ANY kind causes `runAccess` to return the
error immediately. The data plane is NOT started. The daemon exits non-zero. The
log-and-continue pattern at access.go:145–149 MUST be replaced.

**Canonical code:**
```go
mgmtSrv, err := startMgmtServer(ctx, &mgmtWG, cfg, "access", daemonPriv, nil)
if err != nil {
    return fmt.Errorf("access: start management server: %w", err)
}
```

This reconciles with EC-013: "config error (non-loopback console, bad socket perms)
ABORTS startup." There is no non-fatal variant for access mode in this wave.

**Future note:** If a future wave explicitly designs a "degraded-management" mode
(e.g., a planned maintenance window), a dedicated ADR must justify it. Until then, any
mgmt-server start failure is fatal. This ruling records that design decision.

**Downstream changes required by Ruling J:**

- **Implementer (S-W5.01):** Replace the log-and-continue block (access.go:143–149)
  with the fatal-return pattern above. Remove the `//nolint:errcheck` comment on
  `fmt.Fprintf` that was papering over the non-fatal path.
- **Story-writer (S-W5.01):** Amend AC-015 to add: "If `startMgmtServer` returns an
  error (any error — config-class or transient bind failure), `runAccess` returns that
  error immediately. The data plane is NOT started. Test: `TestRunAccess_MgmtStartFailureAborts`
  — inject a `startMgmtServer` stub that returns an error; verify `runAccess` returns
  non-nil without proceeding to `buildAccessComponents` or `runAccessWithConnector`."
- **Product-owner:** Amend BC-2.07.004 EC-013 to strengthen from "ABORTS startup" to
  "Any `startMgmtServer` failure aborts daemon startup — the data plane is never entered.
  This applies to config-class errors and transient bind failures equally in the access
  daemon (this wave)."

---

### Ruling K — EC-001 Silent-Stall: `send E-ADM-010 vs. close-only` (MEDIUM)

**Finding grounded in code:**
`mgmt.go` lines 458–468 (inside `handleConnection`):
```go
if err := dec.Decode(&cresp); err != nil {
    _ = conn.SetReadDeadline(time.Time{})
    if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
        sendAuthFail(conn, s.handshakeTimeout)
    }
    // Timeout case: falls through to return — no sendAuthFail
    return
}
```

On a `HandshakeTimeout` expiry (the silent-stall / EC-001 case), the implementation
deliberately skips `sendAuthFail`. The inline comment reads: "On timeout (HandshakeTimeout
expiry / silent stall) just close — no point sending AUTH_FAIL to a non-responsive client."

BC-2.07.004 EC-001 and AC-001 (v1.3) specify: "On expiry: sends E-ADM-010 + closes
connection." The implementation, test assertion, and BC text disagree.

**Ruling: AMEND BC-2.07.004 EC-001 and AC-001 to match the implementation.**

The implementation's reasoning is sound:
- A client that has timed out without sending a `CHALLENGE_RESPONSE` is not reading.
  Sending `AUTH_FAIL` to such a client would block on `conn.Write` (slowloris-on-write)
  until the write deadline fires, wasting `HandshakeTimeout` (10 s) on a dead connection.
- The `sendAuthFail` call already sets a write deadline, so the write would not hang
  indefinitely. However, the extra write provides no diagnostic value to a non-responsive
  peer and adds latency to slot reclamation.
- The security-relevant invariant — that a timed-out client does not get an RPC — is
  fully satisfied by the `return` alone.

The correct behaviour is **close-only** on `HandshakeTimeout` expiry.

**Canonical EC-001 text (product-owner must apply to BC-2.07.004):**
> EC-001 | Client connects and sends nothing (HandshakeTimeout silent stall) | Server
> sends CHALLENGE, applies `HandshakeTimeout` read deadline (default 10 s) on the
> subsequent read. On timeout expiry: the connection is closed immediately **without
> sending `AUTH_FAIL`** — the non-responsive client would not read it, and the extra
> write would delay slot reclamation. Non-timeout decode errors (malformed JSON,
> oversized message, EOF before timeout) DO send `AUTH_FAIL` before closing. The server
> does not hang indefinitely in either case.

**AC-001 test amendment (story-writer must apply to S-W5.01):**
The existing sub-case in `TestMgmtServer_IssuesChallengeFirst_AC001` that says
"server closes connection" is correct; the sub-case must NOT assert that an `AUTH_FAIL`
message was received on timeout. Update the assertion to: "verify the connection is
closed within ~100 ms of `HandshakeTimeout` expiry; verify NO `AUTH_FAIL` message was
received (the read on the client side returns EOF or closed-pipe error, not an auth_fail
JSON object)."

**No code change required.** The implementation is correct. Only BC-2.07.004 EC-001
and AC-001 need updating.

**Downstream changes required by Ruling K:**

- **Product-owner:** Amend BC-2.07.004 EC-001 with the canonical text above.
- **Story-writer (S-W5.01):** Amend AC-001 test assertion for the timeout sub-case:
  assert connection-close without AUTH_FAIL on timeout; AUTH_FAIL IS sent on
  non-timeout decode errors.
- **Implementer:** No change. The code is correct.

---

### Ruling L — E-CFG-008 Canonical Message String (MEDIUM)

**Finding:**
Three distinct placements of the `E-CFG-008` code have appeared across the codebase
artifacts:

1. `mgmt_wire.go` (Ruling D canonical implementation): error message with `E-CFG-008:`
   embedded as a prefix in the format string — `fmt.Errorf("E-CFG-008: management_socket:
   console mode requires a loopback address ...")`
2. `VP-073` property spec: references the canonical message WITHOUT the code prefix.
3. `AC-014` and `error-taxonomy.md`: inconsistent — some cite the code inline, some do not.

**Ruling:**

The canonical E-CFG-008 message string for the console-loopback-rejection variant is:

```
E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: <address>
```

The error code `E-CFG-008` IS embedded as a prefix in the format string so it is
grep-able in logs and test assertions. This placement was established by Ruling D
and is already in the `buildMgmtListener` implementation. It is the authoritative
placement.

The `config.Validate()` E-CFG-008 variant (empty/whitespace management_socket) uses a
DIFFERENT message format (it does not embed the code because `Validate()` returns a
structured error slice, and the calling code prefixes the code separately). This is
acceptable — the two E-CFG-008 variants come from different code sites and different
error-reporting paths. They share the code for taxonomy lookup but have different
message content.

**Authoritative message strings by variant:**

| Variant | Site | Message format |
|---------|------|----------------|
| Console TCP non-loopback | `buildMgmtListener` in `mgmt_wire.go` | `"E-CFG-008: management_socket: console mode requires a loopback address (127.0.0.1, [::1], or localhost); got: <address>"` (code embedded) |
| Empty/whitespace management_socket | `config.Validate()` | `"config error: management_socket: must not be empty. Fix: ..."` (code NOT embedded in string; Validate returns structured error with code field) |

**VP-073 update:** VP-073's property description must match the `buildMgmtListener`
variant above. If VP-073's assertion is `strings.Contains(err.Error(), "E-CFG-008")`,
that remains correct because the code is embedded. Any assertion checking the FULL
string literal must use the `buildMgmtListener` format above.

**AC-014 update:** The test `TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback`
asserts `error string contains "E-CFG-008"`. This is already correct. No change to the
assertion logic — only the comments/doc-strings need to cite the canonical format above.

**Downstream changes required by Ruling L:**

- **Product-owner:** Amend `error-taxonomy.md` E-CFG-008 row to document both variants
  explicitly (buildMgmtListener variant with embedded code; Validate() variant with
  structured code). Ensure the row clearly distinguishes the two message formats.
- **Story-writer (S-W5.01):** In AC-014, add a comment citing the canonical message
  format above so future reviewers do not re-flag the embedded-code pattern as
  inconsistent with other error messages.
- **Product-owner (VP-073):** Update VP-073 property description to cite the canonical
  `buildMgmtListener` message format. Ensure the test assertion is
  `strings.Contains(err.Error(), "E-CFG-008")` (not a full-string match that would
  break if the message wording is refined).
- **Implementer:** No code change required.

---

### Ruling M — RPC Envelope Wire-Type Mismatch: `"rpc_request"`/`"rpc_response"` vs. `"request"`/`"response"` (HIGH — Integration)

**Finding grounded in code:**
`cmd/sbctl/client.go` line 253 (S-6.03 worktree HEAD 93301fc):
```go
req := rpcRequestMsg{
    Type:    "rpc_request",   // WRONG — ADR-012 §3 step 6 specifies "request"
    ...
}
```
`cmd/sbctl/client.go` line 64 (the `rpcResponseMsg` struct tag comment context):
```go
type rpcResponseMsg struct {
    Type  string `json:"type"`   // expects "rpc_response" in dispatch()
    ...
}
```

`internal/mgmt/mgmt.go` (S-W5.01 worktree) reads `req.Type != "request"` (line 579) and
sets `resp.Type = "response"` (line 594). ADR-012 §3 step 6 unambiguously specifies:
```json
{ "type": "request", ... }
```
and:
```json
{ "type": "response", ... }
```

The client emits `"rpc_request"` but the server expects `"request"`. This is a live
client↔server protocol mismatch that will cause every RPC to fail with a silent
type-mismatch close (server sees unknown type after auth → closes connection; client
sees EOF → E-RPC-001 decode error). The mismatch is masked today because:
- The S-W5.01 unit tests mock the server-side handler, never exercising server's
  `req.Type != "request"` check.
- The S-6.03 unit tests mock the server, never exercising the actual wire type.
- No S-W5.02 E2E harness exists yet to catch the seam.

This is a [process-gap]: no current AC in either story constrains the RPC envelope
type string, allowing the mismatch to survive both story perimeters cleanly.

**Ruling:**

The canonical wire-type strings per ADR-012 §3 step 6 are:
- Client sends: `"type": "request"`
- Server responds: `"type": "response"`

These are the ONLY valid values. `"rpc_request"` and `"rpc_response"` are NOT valid
wire types and MUST NOT appear in any implementation.

**Required changes:**

1. **`cmd/sbctl/client.go` (implementer, S-6.03):** Change line 253:
   ```go
   // BEFORE (wrong):
   Type: "rpc_request",
   // AFTER (correct):
   Type: "request",
   ```
   The `rpcResponseMsg.Type` field in `dispatch()` does not need to change — `dispatch()`
   does not check `resp.Type`; it only checks `resp.OK`. However, the field should still
   carry `"response"` from the server, which it will once the server's `resp.Type =
   "response"` (already correct in the server) is received. No change needed to the
   response parsing logic.

2. **New AC in S-6.03 (story-writer):** Add `AC-010` (or next available number) to
   S-6.03:
   > AC-010 (traces to ADR-012 §3 step 6 / BC-2.07.002 PC-3 — RPC envelope wire type)
   > The `dispatch()` function in `cmd/sbctl/client.go` MUST set `rpcRequestMsg.Type`
   > to `"request"` (not `"rpc_request"` or any other string). The server expects
   > `"type":"request"` per ADR-012 §3 step 6 and closes the connection on any other
   > type after authentication.
   > Test: `TestDispatch_EmitsCorrectWireType` — use `net.Pipe` as transport;
   > read the raw JSON bytes sent by `dispatch()`; verify `"type":"request"` appears
   > in the serialized request. Also verify that a `"type":"response"` response from
   > the server is correctly decoded as a successful `dispatch()` call.

3. **BC-2.07.002 PC-3 annotation (product-owner):** Add a protocol-precision note
   to BC-2.07.002 PC-3: "The authenticated RPC request envelope MUST have
   `\"type\":\"request\"` (ADR-012 §3 step 6). The response envelope MUST have
   `\"type\":\"response\"`. Any other type string causes the server to close the
   connection."

**Downstream changes required by Ruling M:**

- **Implementer (S-6.03):** Change `Type: "rpc_request"` → `Type: "request"` in
  `dispatch()` in `cmd/sbctl/client.go`. One-line change.
- **Story-writer (S-6.03):** Add AC-010 as specified above.
- **Product-owner:** Add protocol-precision note to BC-2.07.002 PC-3 as specified.
  No existing PC or invariant is amended; this is an additive precision annotation.

---

### Ruling N — `vp_traces` Forward-Link Gap in S-W5.01 Frontmatter (LOW)

**Finding:**
S-W5.01 frontmatter `vp_traces: [VP-064, VP-065, VP-066]` omits VP-068, VP-069,
VP-070, VP-071, VP-072, VP-073 even though those VPs name `implementing_story:
S-W5.01` and their properties are exercised by AC-016 (VP-068), AC-017 (VP-069),
and AC-007/AC-014/AC-018 (VP-070–073).

**Ruling:** This is a story-writer fix only. The story frontmatter must enumerate
all VPs whose `implementing_story` points to S-W5.01.

**Downstream changes required by Ruling N:**

- **Story-writer (S-W5.01):** Update the `vp_traces:` frontmatter field from:
  ```yaml
  vp_traces: [VP-064, VP-065, VP-066]
  ```
  to:
  ```yaml
  vp_traces: [VP-064, VP-065, VP-066, VP-068, VP-069, VP-070, VP-071, VP-072, VP-073]
  ```
  (Add VP-068 through VP-073; retain VP-064/065/066 as they were.)

---

## Product-Owner Handoff (v1.4 — Wave-5 Convergence Round-3 Rulings G–N)

This section supplements the v1.3 Product-Owner Handoff. Apply each item exactly as
written. Items here supersede v1.3 instructions for the same clause where they conflict.

### BC-2.07.004 Changes (Round-3)

**EC-001 amendment (Ruling K):** Replace the existing EC-001 row with:
> EC-001 | Client connects and sends nothing (HandshakeTimeout silent stall) | Server
> sends CHALLENGE, applies `HandshakeTimeout` read deadline (default 10 s) on the
> subsequent read. On timeout expiry: connection is closed immediately WITHOUT sending
> `AUTH_FAIL` — the non-responsive client would not read it, and the extra write would
> delay slot reclamation. Non-timeout decode errors (malformed JSON, oversized message,
> EOF before timeout) DO send `AUTH_FAIL` before closing. The server does not hang
> indefinitely in either case.

**PC-10 drain-ordering clarification (Ruling I):** Append to the existing PC-10 text:
> Drain-ordering guarantee: connections accepted after `Shutdown` sets `shuttingDown`
> are closed immediately without entering `connWG` — no Add-after-Done-zero panic is
> possible. Connections already in-flight are force-closed by `closeAllConns` before
> `connWG.Wait`. `trackConn` is called before `connWG.Add` to ensure `closeAllConns`
> always captures every in-flight connection.

**EC-013 strengthening (Ruling J):** Append to the existing EC-013 row:
> Any `startMgmtServer` failure (config-class or transient bind) aborts the access
> daemon startup — the data plane is never entered. There is no non-fatal / degraded-
> management mode in the access daemon (this wave).

### BC-2.07.002 Changes (Round-3)

**PC-3 protocol-precision annotation (Ruling M):** Append to the existing PC-3 text:
> Protocol-precision note (Ruling M): the authenticated RPC request envelope MUST
> carry `"type":"request"` (ADR-012 §3 step 6). The server response carries
> `"type":"response"`. Any other type string after authentication causes the server
> to close the connection silently — `dispatch()` in `cmd/sbctl/client.go` MUST
> use `"type":"request"`.

### error-taxonomy.md Changes (Round-3)

**E-CFG-008 dual-variant documentation (Ruling L):** Amend the E-CFG-008 row to
distinguish the two message variants:
> E-CFG-008 has two distinct message formats depending on originating site:
> - `buildMgmtListener` variant (console TCP loopback rejection):
>   `"E-CFG-008: management_socket: console mode requires a loopback address
>   (127.0.0.1, [::1], or localhost); got: <address>"` — code is embedded in the
>   error string for grep-ability; emitted by `cmd/switchboard/mgmt_wire.go`.
> - `config.Validate()` variant (empty/whitespace management_socket):
>   `"config error: management_socket: must not be empty. Fix: ..."` — code is
>   returned as a structured field, not embedded in the message string; emitted by
>   `internal/config/config.go Validate()`.
> Both share the E-CFG-008 code for taxonomy lookup. Test assertions SHOULD use
> `strings.Contains(err.Error(), "E-CFG-008")` rather than a full-string match.

### VP-073 Update (Ruling L)

**VP-073 property description update:** Update the VP-073 property description to
cite the canonical `buildMgmtListener` message format:
> Property: for any TCP console-mode `management_socket` address whose host is not
> `127.0.0.1`, `[::1]`, or `localhost`, `buildMgmtListener` returns a non-nil error
> whose string representation contains `"E-CFG-008"`. The full canonical prefix is
> `"E-CFG-008: management_socket: console mode requires a loopback address"`.
> Test assertion MUST use `strings.Contains(err.Error(), "E-CFG-008")`.

---

## Story-Writer Handoff (v1.4 — Wave-5 Convergence Round-3 Rulings G–N)

### S-W5.01 Changes (Round-3)

**AC-017 amendment (Ruling G — unexpected-close test):** Add a third sub-case:
> Sub-case (c): `TestServe_ReturnsErrOnUnexpectedListenerClose` — create a server
> with a real listener (`net.Listen("tcp", "127.0.0.1:0")`); start `Serve` with
> `context.Background()` (live context, no cancel); close the listener directly
> from the test (do NOT call `Shutdown`, do NOT cancel ctx); assert `Serve` returns
> a **non-nil** error. This verifies BC-2.07.004 PC-10's "non-nil on unexpected
> failure" clause. The fix is the `&& ctx.Err() != nil` conjunct in the Accept-error
> predicate (Ruling G).

**AC-017 amendment (Ruling H — remove dead case <-done: branches):** Add to the
implementation contract in AC-017:
> The pre-Accept semaphore select uses `case <-ctx.Done():` (not `case <-done:`).
> The post-Accept-error non-default select block (`select { case <-done: ...; default: }`)
> is absent — the `s.shuttingDown.Load()` check subsumes it. Backoff selects use
> `case <-ctx.Done():`. The ctx-watcher goroutine's own `case <-done:` is retained.

**AC-017 amendment (Ruling I — shutdown-window ordering and drain):** Add to AC-017:
> Connection-acceptance ordering: after `s.shuttingDown.Load()` is true, connections
> accepted by the loop before the flag is detected are dropped (closed immediately,
> semaphore released) WITHOUT entering `connWG`. `trackConn` is called BEFORE
> `connWG.Add(1)` and BEFORE `go func()` so `closeAllConns` always captures every
> in-flight connection.
>
> Required additional tests:
> - `TestServe_ShutdownWindowNoAddAfterWaitPanic` (run with `-race`): dial in a
>   tight loop while calling `Shutdown`; 100 iterations; no panic, no race.
> - `TestServe_DrainCompletesWithinBudget`: stalled authenticated connection;
>   `Shutdown` with 2 s context; assert returns within 2 s (not 10 s).

**AC-015 amendment (Ruling J — mgmt start failure is fatal):** Add to AC-015:
> If `startMgmtServer` returns an error (any class), `runAccess` returns that error
> immediately. The data plane is NOT started.
> Test: `TestRunAccess_MgmtStartFailureAborts` — inject a `startMgmtServer` stub
> that returns a non-nil error; verify `runAccess` returns non-nil without entering
> `buildAccessComponents` or `runAccessWithConnector`.

**AC-001 amendment (Ruling K — timeout sub-case assertion):** In the
`TestMgmtServer_IssuesChallengeFirst_AC001` timeout sub-case, update the assertion:
> Assert that the connection is closed within ~100 ms of `HandshakeTimeout` expiry.
> Assert that NO `AUTH_FAIL` JSON message was received on the client pipe before the
> close — a timeout close is silent (close-only, no AUTH_FAIL).

**AC-014 amendment (Ruling L — canonical E-CFG-008 format comment):** In AC-014,
add a comment:
> The canonical E-CFG-008 message for `buildMgmtListener` embeds the code as a
> prefix: `"E-CFG-008: management_socket: console mode requires a loopback address
> (127.0.0.1, [::1], or localhost); got: <address>"`. Test assertions must use
> `strings.Contains(err.Error(), "E-CFG-008")` — not a full-string match.

**vp_traces frontmatter (Ruling N):** Update S-W5.01 frontmatter:
```yaml
vp_traces: [VP-064, VP-065, VP-066, VP-068, VP-069, VP-070, VP-071, VP-072, VP-073]
```

### S-6.03 Changes (Round-3)

**New AC-010 (Ruling M — RPC wire type):** Add to S-6.03:
> AC-010 (traces to ADR-012 §3 step 6 / BC-2.07.002 PC-3 — Ruling M)
> The `dispatch()` function MUST set `rpcRequestMsg.Type` to `"request"`. The
> current implementation uses `"rpc_request"` which the server rejects silently
> after authentication (server closes connection on unknown type). This is a
> one-line fix: `Type: "request"`.
>
> Test: `TestDispatch_EmitsCorrectWireType` — use `net.Pipe`; call `dispatch()`
> with a mock server side; read the raw JSON from the pipe; assert
> `strings.Contains(rawJSON, `"type":"request"`)`. Also verify a
> `{"type":"response","ok":true,...}` response is decoded as a successful call.
>
> Test: `TestDispatch_AcceptsResponseType` — mock server sends
> `{"type":"response","id":"1","ok":true,"data":{}}`;
> verify `dispatch()` returns non-nil data, nil error.

**vp_traces frontmatter (Ruling N — not applicable to S-6.03):** S-6.03's current
`vp_traces: [VP-067, VP-030]` is correct. VP-067 and VP-030 are the VPs implemented
by S-6.03. No change needed.

---

## Wave-5 Convergence Rulings (Round-4) — v1.5

These rulings were produced after a fresh-context adversarial pass covering both
S-W5.01 (`internal/mgmt`, `cmd/switchboard`) and S-6.03 (`cmd/sbctl`). Rulings A–N
remain intact and authoritative. Each ruling below is citable by stable heading
(`### Ruling O` …) in all downstream artifacts. The standing project directive is
**fix-now unless genuinely outside both stories' scope AND depends on code that
does not yet exist**; any deferral is a documented decision, never a silent drop.

---

### Ruling O — Stale Unix Socket: No Pre-Bind Unlink → EADDRINUSE Restart DoS (HIGH / S-W5.01)

**Finding grounded in code:**
`listenUnixMgmt` (`cmd/switchboard/mgmt_wire.go` lines 62–118) uses raw syscalls
(`syscall.Socket` → `syscall.Bind` → `syscall.Listen`) and sets
`SetUnlinkOnClose(false)` at line 116. There is no call to `os.Remove` or
`os.Lstat` before `syscall.Bind`. After a non-graceful exit (SIGKILL, crash, OOM
kill), the socket inode persists on the filesystem. On the next daemon start,
`syscall.Bind` returns `EADDRINUSE` and `listenUnixMgmt` returns a non-nil error;
`startMgmtServer` propagates it; Ruling J makes this unconditionally fatal for the
access daemon — so the daemon cannot restart at all until the operator manually
removes the socket file. This is an operator-hostile restart DoS.

**Ruling: FIX-NOW in S-W5.01 / `listenUnixMgmt`.**

Before calling `syscall.Bind`, `listenUnixMgmt` MUST check whether the path refers
to an existing socket-mode file and, if so, remove it. The canonical pre-bind cleanup
sequence is:

```go
// Pre-bind cleanup: remove a stale socket left by a prior non-graceful exit.
// Guard: only remove if the path is a socket (ModeSocket) — never clobber a
// regular file, directory, or device that happens to share the path.
// The small Lstat→Bind TOCTOU window (another process creating a socket at
// the same path between Lstat and Bind) is accepted: it repairs exactly the
// "restart after crash" case and is no worse than the current EADDRINUSE failure.
if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSocket != 0 {
    _ = os.Remove(path)
}
```

This block is inserted immediately BEFORE the `syscall.Bind` call (after the
`syscall.Socket` / `syscall.CloseOnExec` block). Placement notes:

- The guard `fi.Mode()&os.ModeSocket != 0` ensures we only remove socket inodes.
  A regular file or directory at the socket path is left untouched, causing
  `Bind` to fail with the original error (which is the correct behavior — a
  non-socket inode at the socket path is a misconfiguration, not a stale socket).
- `os.Remove` errors are silently ignored: if Remove fails (permission denied,
  file already removed by a concurrent daemon start), the subsequent Bind will
  either succeed (file gone) or fail with `EADDRINUSE` (file still present), both
  of which propagate correctly.
- `SetUnlinkOnClose(false)` is retained. The socket lifecycle is:
  startup → pre-bind unlink → bind → run → shutdown (`Shutdown` closes listener
  but does NOT unlink). The next startup performs pre-bind unlink again, so
  post-shutdown cleanup is not required.

**TOCTOU assessment:** The Lstat→Remove→Bind window allows another process to
create a socket at the path between Lstat and Bind. In that case Bind fails with
`EADDRINUSE` — the same behavior as today, and no worse. The window cannot cause
data loss or privilege escalation because the only action taken is `os.Remove` on
a path that is confirmed to be a socket.

**Downstream changes required by Ruling O:**

- **Implementer (S-W5.01):** Add the pre-bind cleanup block to `listenUnixMgmt`
  in `cmd/switchboard/mgmt_wire.go` immediately before `syscall.Bind`.
- **Story-writer (S-W5.01):** Add AC-019:
  > AC-019 (traces to BC-2.07.004 EC-013, Ruling O) `listenUnixMgmt` performs a
  > pre-bind cleanup: if `os.Lstat(path)` succeeds and the mode has `ModeSocket`
  > set, `os.Remove(path)` is called before `syscall.Bind`. Regular files and
  > directories at the path are NOT removed. This ensures the daemon can restart
  > cleanly after a non-graceful exit (SIGKILL, crash) without operator
  > intervention.
  > Test: `TestListenUnixMgmt_PreBindCleanup_AC019` — create a temp path via
  > `t.TempDir()`; pre-create a socket file at that path using a prior
  > `listenUnixMgmt` call (close the returned listener without unlinking, so the
  > inode persists); call `listenUnixMgmt` again on the same path; verify it
  > succeeds (no EADDRINUSE). Also test: if the path holds a regular file,
  > `listenUnixMgmt` returns a non-nil error (Bind fails; file NOT removed).
- **Product-owner:** Append to BC-2.07.004 EC-013:
  > Stale-socket restart resilience: `listenUnixMgmt` MUST check for and remove
  > a stale socket inode at the bind path before calling `Bind`. Only
  > `ModeSocket`-mode inodes are removed; other file types are left untouched.
  > This prevents an EADDRINUSE restart DoS after a non-graceful daemon exit.

---

### Ruling P — Fatal-Accept-Error Drain Gap: `closeAllConns` Missing Before `connWG.Wait` on Unexpected-Close Path (MEDIUM / S-W5.01)

**Finding grounded in code:**
`mgmt.go` `Serve` function has three paths that call `s.connWG.Wait()`:

1. **Intentional shutdown path** (line 267–269): checks
   `s.shuttingDown.Load() || (errors.Is(err, net.ErrClosed) && ctx.Err() != nil)`.
   This path is triggered by `Shutdown()` or ctx-cancel; `Shutdown()` calls
   `closeAllConns()` before closing the listener, and the ctx-watcher goroutine
   also calls `closeAllConns()` before `s.ln.Close()`. In both cases,
   `closeAllConns` runs BEFORE `Serve` reaches `connWG.Wait()`. ✓ Correct.

2. **Transient-backoff ctx-cancel exit** (line 285–289):
   ```go
   select {
   case <-time.After(backoff):
   case <-ctx.Done():
       s.connWG.Wait()
       return nil
   }
   ```
   The ctx-watcher goroutine has already called `closeAllConns()` (line 233) by
   the time `ctx.Done()` fires. ✓ Correct.

3. **Fatal accept error path** (line 292–294):
   ```go
   s.connWG.Wait()
   return err
   ```
   This path is taken when `Serve` receives a non-`net.ErrClosed` fatal error from
   `Accept` while `ctx` is still live and `Shutdown` was never called (e.g., the
   underlying fd is forcibly invalidated by the OS, or a kernel bug). `Shutdown`
   has NOT been called; the ctx-watcher goroutine has NOT fired (ctx is live).
   Therefore **`closeAllConns()` has NOT been called**. Any in-flight connection
   goroutines are still running; their `handleConnection` is blocked on a read
   deadline or network I/O. `connWG.Wait()` on this path will block until all
   those goroutines time out — up to `RPCIdleTimeout` (30 s) for an idle
   authenticated connection, or `HandshakeTimeout` (10 s) for a connection in
   the handshake phase. This blows any reasonable shutdown budget.

**Ruling: FIX-NOW — add `s.closeAllConns()` immediately before `s.connWG.Wait()`
on the fatal-accept-error path.**

The canonical corrected fatal-error path:

```go
// Fatal accept error — force-close in-flight connections so goroutines return
// quickly, then drain (Ruling P).
s.closeAllConns()
s.connWG.Wait()
return err
```

This is a one-line insertion before both occurrences of `s.connWG.Wait()` in the
fatal-error branch. `closeAllConns()` is safe to call multiple times:
it snapshots the connection set under the mutex and calls `Close()` on each;
`conn.Close()` twice is benign (second call returns `net.ErrClosed`, which is
ignored because the return value is discarded).

**Downstream changes required by Ruling P:**

- **Implementer (S-W5.01):** In `Serve`, add `s.closeAllConns()` immediately
  before `s.connWG.Wait()` on the fatal-accept-error path (the final
  `s.connWG.Wait(); return err` block at ~line 293). No other changes needed.
- **Story-writer (S-W5.01):** Amend AC-017 to add:
  > On the fatal-accept-error path (Accept returns a non-transient error while
  > ctx is live and Shutdown was never called), `closeAllConns()` is called
  > immediately before `connWG.Wait()` so in-flight connection goroutines are
  > force-closed and the drain completes quickly (Ruling P).
  > Test: `TestServe_FatalAcceptErrorDrainsQuickly` — open a server with a
  > stalled authenticated connection; simulate a fatal Accept error by closing
  > the listener from a goroutine while also controlling the Serve ctx to keep
  > it live (use `context.Background()` and do NOT call `Shutdown`); assert that
  > `Serve` returns within 200 ms (not 30 s / `RPCIdleTimeout`). This test
  > fails without the `closeAllConns()` insertion and passes after.
- **Product-owner:** No new BC text is needed — PC-10 already requires drain to
  complete within the caller's shutdown budget. This is an implementation fix to
  honour that clause on the unexpected-close path.

---

### Ruling Q — AC-001 / EC-001 Story Propagation Gap for Ruling K (MEDIUM / process-gap / S-W5.01)

**Finding grounded in code and spec:**
S-W5.01 v1.3 AC-001 prose at line 110 reads:
> "On deadline expiry the connection is closed with E-ADM-010 (traces to BC-2.07.004
> PC-1 and EC-001)."

This contradicts Ruling K (ARCH-12 v1.4) and BC-2.07.004 EC-001 v1.4, both of
which specify that HandshakeTimeout expiry is a **close-only** event with NO
`AUTH_FAIL` sent. The Ruling K story-writer handoff (v1.4) correctly directed the
story-writer to update the AC-001 test assertion, but did not address the stale
**prose** in the AC-001 body. The test assertion at S-W5.01 line 117–118 IS
correct ("NO AUTH_FAIL"), so the code and the test are both right — only the AC
prose is stale.

Also, S-W5.01 EC-001 (edge-case table line ~554) still says:
> "E-ADM-010 + close"

This is also a propagation gap from Ruling K — the EC-001 row should say
"close-only (no AUTH_FAIL)" consistent with BC-2.07.004 EC-001 v1.4.

**Implementation code is correct. No code change is required.**

**Ruling: STORY-WRITER FIX ONLY — update two stale locations in S-W5.01.**

**Downstream changes required by Ruling Q:**

- **Story-writer (S-W5.01):** Two edits:
  1. In AC-001 body (the prose immediately before the Test block), replace the
     sentence "On deadline expiry the connection is closed with E-ADM-010" with:
     "On deadline expiry (HandshakeTimeout silent stall) the connection is closed
     **without sending AUTH_FAIL** — the non-responsive client would not read it
     (BC-2.07.004 EC-001 v1.4 / Ruling K). Non-timeout decode errors (malformed
     JSON, EOF before timeout, oversized message) DO send AUTH_FAIL before close."
  2. In the EC-001 row of the Edge Cases table, replace "E-ADM-010 + close" with
     "close-only (no AUTH_FAIL on timeout); non-timeout decode errors do send
     AUTH_FAIL before close (BC-2.07.004 EC-001 v1.4 / Ruling K)".
- **No code change.** Implementation is correct per Ruling K.

---

### Ruling R — Per-Handler Execution Timeout: Blocking Handler Pins Goroutine + Semaphore (MEDIUM / S-W5.01)

**Finding:**
`handleConnection` in `mgmt.go` dispatches registered handlers at line 621:
```go
data, err := handlerFn(ctx, req.Args)
```
There is no timeout wrapping around `handlerFn`. A handler that blocks
indefinitely — on a slow database, a stalled external call, or a deadlock — pins
the connection goroutine and its semaphore slot until the parent `ctx` is
cancelled. With `MaxConcurrentConnections = 128`, 128 blocking handlers exhaust
the semaphore and prevent all new connections from being accepted (CWE-400).

**Note on current handler slate:** `startMgmtServer`'s `handlers` parameter is
always `nil` in this wave — no real handlers are registered. The risk is therefore
latent, not immediately exploitable. However:

1. The fix-now directive applies when a fix is cheap and future-proof.
2. The wrapper is approximately 3 lines and does not change any function signature.
3. The handler boundary IS defined in the current wave; deferring until "the first
   real handler lands" introduces a hazard window in the next wave when handler
   registration begins.

**Ruling: FIX-NOW — wrap each `handlerFn` call in a `context.WithTimeout` derived
from `RPCIdleTimeout`.**

The canonical wrapper in `handleConnection`, replacing the bare `handlerFn` call:

```go
// Bound handler execution: a handler that blocks beyond RPCIdleTimeout
// is cancelled. This prevents a single slow handler from permanently
// pinning a semaphore slot (CWE-400 / BC-2.07.004 PC-6 / Ruling R).
hCtx, hCancel := context.WithTimeout(ctx, RPCIdleTimeout)
data, err := handlerFn(hCtx, req.Args)
hCancel()
```

`RPCIdleTimeout` (30 s) is the RPC-phase deadline already used for the read
deadline before the handler call; applying it to handler execution as well
is symmetric and conservative.

**Note:** the existing write-deadline call after `handlerFn` (Ruling E)
already bounds the write side. This ruling bounds the execution side. Together
they close the full CWE-400 slot-exhaustion surface on the RPC dispatch path.

**BC text anchor:** BC-2.07.004 PC-6 (bounded reads / CWE-400) is the closest
existing anchor but addresses the read bound. The product-owner should add Inv-3
extension or a new sentence to PC-6 explicitly stating the handler-execution
bound. The existing Inv-3 says "All socket reads are bounded by MaxMessageBytes"
— this ruling addresses execution time, not message size.

**Downstream changes required by Ruling R:**

- **Implementer (S-W5.01):** In `handleConnection` in `mgmt.go`, replace
  `data, err := handlerFn(ctx, req.Args)` with the `context.WithTimeout` wrapper
  above. One-line change (plus the hCancel defer / explicit call).
- **Story-writer (S-W5.01):** Add AC-020:
  > AC-020 (traces to BC-2.07.004 PC-6 / Inv-3, Ruling R) Every registered
  > handler is invoked with a child context derived via
  > `context.WithTimeout(ctx, RPCIdleTimeout)`. A handler that blocks beyond
  > `RPCIdleTimeout` (30 s) is cancelled via the child context; the connection
  > goroutine returns E-RPC-011 (handler error) after cancellation.
  > Test: `TestMgmtServer_HandlerTimeout_AC020` — register a handler that blocks
  > until its context is cancelled; construct the server with a short
  > `RPCIdleTimeout` override (e.g., 50 ms); authenticate and send the RPC; verify
  > the server returns an E-RPC-011 error response within ~200 ms (not blocked
  > indefinitely). Verify the connection is still open after the timed-out handler
  > (handler timeout is not a connection-close event).
- **Product-owner:** Append to BC-2.07.004 PC-6:
  > Handler execution timeout: registered handler `Fn` functions are invoked with
  > a child context derived via `context.WithTimeout(ctx, RPCIdleTimeout)`. A
  > handler that does not return within `RPCIdleTimeout` is cancelled; the server
  > responds with E-RPC-011. This bounds handler execution to the same time budget
  > as the RPC idle read deadline, closing the CWE-400 goroutine-pin surface on
  > the handler dispatch path.

---

### Ruling S — `version="dev"` Sentinel: No CI Assertion That Release Artifacts Are Non-Dev (process-gap / MEDIUM)

**Finding:**
`cmd/switchboard/version.go` (or the equivalent `var version = "dev"` in `main.go`)
holds the build-time version sentinel. `startMgmtServer` passes it as `daemonVersion`
to `mgmt.NewServer`, which embeds it in AUTH_OK messages. BC-2.07.004 PC-7 says
"hardcoding `"dev"` in production is a defect." There is no CI assertion — in the
`release` or `release-verify` workflow — that the built binary's version string is
a semver value rather than `"dev"`. A broken ldflags injection silently produces a
release binary with `daemon_version: "dev"` in every AUTH_OK.

**Decision: FOLLOW-UP STORY — this is a CI/devops gap, not in either story's file scope.**

Rationale: the fix touches `.github/workflows/release-verify.yml` (or a
`Makefile`/`Justfile` CI target). Neither S-W5.01 nor S-6.03 owns CI workflow
files. Adding the assertion to one of these stories' file scope would be a scope
creep that risks blocking delivery of the management-plane code on a CI plumbing
problem.

**Follow-up story specification:**
- **Target epic:** E-9 (CI/devops — the existing release CI epic; if E-9 does not
  exist, open one titled "release verification and CI hardening").
- **Scope:** Add a `release-verify` CI step (or extend the existing
  `.github/workflows/release-verify.yml`) that:
  1. Builds the binary with release ldflags.
  2. Runs `switchboard --version` (or parses the binary's version via the management
     protocol once a test harness is available).
  3. Asserts the version string matches semver (`v\d+\.\d+\.\d+.*`) and is NOT
     `"dev"`.
  4. Fails the CI run with a clear error if the assertion does not hold.
- **Priority:** P1 (non-blocking for this wave; must ship before the first tagged
  release of the management plane).
- **Note for story-writer:** The story should also verify that the `just build`
  (dev build) correctly produces `"dev"` and that only the `just build-all` / CI
  release path injects the semver.

---

### Ruling T — `TestServe_ShutdownWindowNoAddAfterWaitPanic` Non-Discriminating Test (test-quality / S-W5.01)

**Finding:**
`TestServe_ShutdownWindowNoAddAfterWaitPanic` (introduced by Ruling I) is intended
to detect an `Add`-after-`Wait`-at-zero panic in the `sync.WaitGroup`. The stated
rationale is that `Shutdown` calls `connWG.Wait()` in a window where `connWG.Add`
could race. However:

- As adjudicated in Ruling H and implemented in `mgmt.go`, `Shutdown` does NOT
  call `connWG.Wait()`. `Serve` is the sole owner of `connWG.Wait()` (see
  `Shutdown()` implementation at lines 333–343 — only `closeAllConns()` and
  `s.ln.Close()`; no `Wait()`).
- The `Add`-after-`Wait` panic requires `Wait` to have returned from zero AND a
  concurrent `Add` to fire after that. In the current design, `Wait` is only
  called once — after the accept loop exits — by which point the `shuttingDown`
  flag prevents all new `Add(1)` calls. The panic scenario does not exist in the
  current implementation.
- The test's RED rationale (Add-after-Wait-at-zero panic) is structurally
  impossible in the current code. The test passes before and after the Ruling I
  fix and cannot distinguish between the fixed and unfixed states.

The DROP behavior IS genuinely tested by `TestServe_DrainCompletesWithinBudget`
(which verifies that a stalled connection drains within 2 s — which would fail
if `closeAllConns` were not called before `connWG.Wait`). The `connWG.Add` ordering
is tested by `go test -race` on the broader test suite.

**Ruling: TEST-WRITER FIX ONLY — re-document the test or rebuild it as a
deterministic harness.**

The test-writer should choose one of these options:

**Option A (re-document):** Update the test doc-comment to describe its real
property: "verifies that concurrent dial+Shutdown does not panic and passes the
race detector — a general concurrency-safety integration test, not a specific
Add-after-Wait guard." This is weaker but honest. The test still provides value
as a race-detector-driven smoke test.

**Option B (rebuild as discriminating test):** Replace the test with a harness
that specifically validates the `shuttingDown` pre-check ordering. Example:
override `connWG.Add` via a wrapper type that panics if `shuttingDown` is true
and `Add` is called; this directly exercises the Ruling I invariant. This is
stronger but requires more test scaffolding.

**No production code change required.**

---

### Ruling U — `dispatch()` Never Validates `resp.Type` Field (CRITICAL / S-6.03)

**Finding grounded in code:**
`cmd/sbctl/client.go` `dispatch()` (lines 263–276): the response is decoded into
`rpcResponseMsg` which has a `Type string` field (line 65). The decoded `resp.Type`
is never inspected. Ruling M established that the only valid server response type
is `"response"`. A server that sends a wire-type-mismatch response (e.g., a
future bug, a rogue proxy, or a test fixture mistake) is silently accepted if
`resp.OK == true` — dispatch returns success with whatever data the wrong-type
response carried.

More concretely: the finding states that `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001`
in S-6.03 uses a fixture that emits `"type":"rpc_response"`. If the type field
were never validated (as is the case before this ruling), a `"type":"rpc_response"`
response with `"ok":true` would be returned as a successful dispatch — contradicting
BC-2.07.002 PC-3 and ADR-012 §3 step 6.

**Verification of current code:** Looking at `dispatch()` at line 268: `if !resp.OK`.
The `resp.Type` is not inspected. The finding is confirmed.

**Ruling: FIX-NOW — `dispatch()` MUST reject any response where `resp.Type != "response"`.**

The canonical fix, replacing the current `if !resp.OK` guard:

```go
// Validate wire type first (ADR-012 §3 step 6 / BC-2.07.002 PC-3 / Ruling M+U).
// Any response other than "type":"response" is a protocol error.
if resp.Type != "response" {
    return nil, fmt.Errorf("rpc failed: %s: unexpected response type %q (want \"response\")", command, resp.Type)
}
if !resp.OK {
    reason := "server returned ok:false"
    if resp.Error != nil && resp.Error.Message != "" {
        reason = resp.Error.Message
    }
    return nil, fmt.Errorf("rpc failed: %s: %s", command, reason)
}
```

**Test fixture correction:** The test `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001`
fixture currently emits `"type":"rpc_response"`. This fixture MUST be corrected to
`"type":"response"` — the fixture should exercise the `"ok":false` path, not a
wire-type mismatch path. A SEPARATE negative test is required:

> `TestDispatch_RejectsWrongResponseType` — mock server sends
> `{"type":"rpc_response","id":"1","ok":true,"data":{}}`;
> verify `dispatch()` returns a non-nil error containing `"unexpected response type"`.

**Downstream changes required by Ruling U:**

- **Implementer (S-6.03):** Add `resp.Type != "response"` guard to `dispatch()`
  as shown above.
- **Test-writer (S-6.03):** 
  1. Fix the `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001` fixture: change
     the server response from `"type":"rpc_response"` to `"type":"response"` (the
     test is exercising the `"ok":false` path, which only works with the correct
     response type).
  2. Add `TestDispatch_RejectsWrongResponseType` as a negative test for the
     type-validation guard.
- **Story-writer (S-6.03):** Amend AC-010 to explicitly require that `dispatch()`
  rejects `resp.Type != "response"` (not just that `dispatch()` sends
  `"type":"request"`). The receiving side of the wire-type contract must be tested.
  Add the negative test as a required sub-case:
  > AC-010 additional requirement: `dispatch()` checks `resp.Type == "response"` after
  > decoding. Any response with a different `type` (e.g., `"rpc_response"`,
  > `"auth_fail"`, `""`) is rejected with a non-nil error containing the received
  > type. Test: `TestDispatch_RejectsWrongResponseType`.
- **Product-owner:** Append to BC-2.07.002 PC-3:
  > Receiving-side wire-type validation: after decoding the RPC response, `dispatch()`
  > MUST verify `resp.Type == "response"`. Any other value is treated as a protocol
  > error and returned as E-RPC-001. This closes the gap where a wrong-type response
  > with `"ok":true` would be silently accepted.

---

### Ruling V — `dispatch()` Missing Read Deadline; Reconcile ARCH-12 "Residual Concern" Note (HIGH / S-6.03)

**Finding grounded in code:**
`cmd/sbctl/client.go` `dispatch()` (line 264):
```go
if err := json.NewDecoder(io.LimitReader(conn, maxMessageBytes)).Decode(&resp); err != nil {
```
No `conn.SetReadDeadline` call precedes this `Decode`. `Authenticate()` sets a
deadline (line 189: `conn.SetReadDeadline(deadline)`) but the deadline is never
cleared after `Authenticate` returns — it remains set until it fires or is
explicitly reset. The context deadline from `--timeout` controls this implicitly
via `net.Conn`'s interaction with context cancellation on dial, but after the dial
completes, the deadline management is manual.

**ARCH-12 v1.4 "residual concern" note** (lines 1303–1307) explicitly flags this:
> "A future caller that passes `context.Background()` (no deadline) into
> `connectAndRun` would receive the `HandshakeTimeout` (10 s) fallback for auth
> but have no deadline on the RPC phase."

ADR-012 §7 states: "Every blocking `Decode` on the management socket MUST be
preceded by `conn.SetReadDeadline`." Ruling F (v1.3) clarified that the sbctl
single-ctx-timeout model is sanctioned, but the model relies on the ctx-derived
deadline being active through the RPC phase. The `Authenticate()` function sets a
deadline derived from ctx; after `Authenticate` returns nil, the deadline may have
partially elapsed but is still active. `dispatch()`'s decode therefore does inherit
the remaining deadline.

**However:** the `Authenticate()` deadline is derived from `ctx.Deadline()` if set,
or `time.Now().Add(handshakeTimeout)` if not. In the `context.Background()` caller
case flagged in the residual concern, the deadline after auth is `time.Now()` + 10 s
from the auth start — which gives the RPC phase whatever is left of that 10 s budget.
For a fast auth (100 ms), 9.9 s remains for the RPC. For a slow auth (9 s), only
1 s remains. This is fragile and not explicitly re-armed for the RPC phase.

**The fix-now directive applies.** The fix is a single `conn.SetReadDeadline` call
before `dispatch`'s decode, symmetric with the existing write deadline (Ruling E).

**Ruling: FIX-NOW — add an explicit read deadline in `dispatch()` before the
response decode, derived from the remaining ctx deadline.**

The canonical fix in `dispatch()`, preceding the `Decode` call:

```go
// Set a read deadline for the RPC response. The sbctl single-ctx-timeout model
// (Ruling F / ARCH-12 v1.4) governs: if ctx has a deadline, derive from it;
// otherwise fall back to RPCIdleTimeout (30 s) for defense in depth.
// This is symmetric with Authenticate()'s deadline and with the server's
// RPCIdleTimeout re-arm (Ruling E).
responseDeadline, ok := ctx.Deadline()
if !ok {
    responseDeadline = time.Now().UTC().Add(30 * time.Second) // RPCIdleTimeout symmetric
}
if err := conn.SetReadDeadline(responseDeadline); err != nil {
    return nil, fmt.Errorf("rpc failed: %s: set read deadline: %w", command, err)
}
defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
```

Note: `dispatch` needs `context.Context` as its first parameter to access the
deadline. The current signature `dispatch(conn net.Conn, command string, args any)`
does NOT have a context parameter, violating go.md rule 7. This ruling requires
the signature to be updated:

```go
// BEFORE:
func dispatch(conn net.Conn, command string, args any) (json.RawMessage, error)

// AFTER (go.md rule 7 — ctx first):
func dispatch(ctx context.Context, conn net.Conn, command string, args any) (json.RawMessage, error)
```

All call sites (`connectAndRun` in `client.go`) must pass `ctx`.

**ARCH-12 residual-concern note reconciliation:** The note at lines 1303–1307
("Residual concern (non-blocking)") is SUPERSEDED by this ruling. After the fix,
`dispatch()` explicitly re-arms the deadline from ctx for every caller. The
residual concern is closed. The note should be updated to reflect that the concern
is resolved.

**Note on `context.Context` in `dispatch`:** Adding ctx to dispatch's signature is
a small, self-contained change within S-6.03's file scope (`cmd/sbctl/client.go`).
This does NOT require changes to other packages or the daemon side.

**Downstream changes required by Ruling V:**

- **Implementer (S-6.03):** 
  1. Update `dispatch` signature to `dispatch(ctx context.Context, conn net.Conn, command string, args any)`.
  2. Add the read-deadline block above before the response `Decode` call.
  3. Update `connectAndRun`'s `dispatch` call to pass `ctx`.
- **Story-writer (S-6.03):** Add AC-011:
  > AC-011 (traces to BC-2.07.003 Invariant 2 + ADR-012 §7, Ruling V) `dispatch()`
  > accepts `context.Context` as its first parameter (go.md rule 7). Before decoding
  > the RPC response, it calls `conn.SetReadDeadline` derived from `ctx.Deadline()`
  > (or 30 s fallback if ctx has no deadline). The deadline is cleared via `defer`
  > after `dispatch` returns.
  > Test: `TestDispatch_RespReadDeadlineEnforced` — create a mock server that
  > completes AUTH_OK then hangs (stops writing the response); pass a context with a
  > 50 ms deadline to `dispatch`; verify `dispatch` returns a non-nil error within
  > ~200 ms (deadline fires, not blocked indefinitely).
- **Product-owner:** Amend BC-2.07.003 Invariant 2 to add:
  > The `dispatch()` function also sets a read deadline before decoding the RPC
  > response, derived from the context deadline (or `RPCIdleTimeout`-equivalent
  > as fallback). `sbctl` does not hang indefinitely on the RPC response phase.
- **ARCH-12 §Ruling F update:** Replace the "Residual concern (non-blocking)"
  note at lines 1303–1307 with:
  > ~~Residual concern (non-blocking):~~ **RESOLVED by Ruling V (v1.5):**
  > `dispatch()` now accepts `ctx` as its first parameter and explicitly sets
  > `conn.SetReadDeadline` before the response decode, derived from
  > `ctx.Deadline()` or the 30 s fallback. The concern about a `context.Background()`
  > caller leaving the RPC response phase without a deadline is closed.

---

### Ruling W — `t.Fatalf` Called from Non-Test Goroutines in `client_test.go` (CRITICAL / test-correctness / S-6.03)

**Finding:**
`cmd/sbctl/client_test.go` calls `t.Fatalf` (or functions containing `t.Fatalf`)
from within `go func()` goroutines spawned in test helpers. `t.Fatalf` calls
`runtime.Goexit()` on the calling goroutine. When called from a non-test goroutine,
`runtime.Goexit()` exits that goroutine — NOT the test goroutine — leaving the
test in an indeterminate state (it may continue running, appear to hang, or report
a different failure). This is a known Go testing hazard documented in `testing.T`'s
documentation.

Three confirmed sites per the finding:
1. `VP067_g` — `wellFormedChallenge(t)` called inside `go func()` at the mock
   server goroutine.
2. `VP067_h` — same pattern.
3. `TestAuthenticate_PrivKeyNeverTransmitted` — `freshNonce(t)` inside `go func()`.

**Ruling: TEST-WRITER FIX ONLY — no production code change, no BC change.**

The fix is to hoist all test-helper calls that could invoke `t.Fatalf` into the
test goroutine BEFORE spawning `go func()`, capturing the results by value:

```go
// BEFORE (wrong — t.Fatalf fires in server goroutine, not test goroutine):
go func() {
    challenge := wellFormedChallenge(t) // may call t.Fatalf
    _ = json.NewEncoder(conn).Encode(challenge)
}()

// AFTER (correct — nonce/challenge generated in test goroutine):
challenge := wellFormedChallenge(t) // t.Fatalf fires in test goroutine if this fails
go func(c challengeMsg) {
    _ = json.NewEncoder(conn).Encode(c)
}(challenge)
```

This is a [process-gap] for the test-writer agent's goroutine-safety discipline.
The `go test -race` detector may or may not catch this depending on timing; the
failure mode is non-deterministic test hangs or false passes, not a consistent
red test.

**Downstream changes required by Ruling W:**

- **Test-writer (S-6.03):** Audit all mock-server goroutines in `client_test.go`
  for `t.Fatalf`/`t.Fatal`/`t.Error`/`t.Log` calls (and any helper functions
  that call them). Hoist all such calls into the test goroutine before the
  `go func()` spawn. Capture the results by value, then pass them to the
  goroutine as parameters or via closure over already-computed values.
  The three confirmed sites are `VP067_g`, `VP067_h`, and
  `TestAuthenticate_PrivKeyNeverTransmitted`.

---

### Ruling X — `dispatch()` Hardcoded `id: "1"` and No `resp.ID` Echo Check (MEDIUM / S-6.03)

**Finding grounded in code:**
`dispatch()` at line 254: `ID: "1"` — always hardcoded. `resp.ID` is decoded (line
64, `rpcResponseMsg.ID`) but never compared against `req.ID`. ADR-012 §3 step 6
defines `id` as "a client-generated opaque string, echoed in response" — the intent
is correlation. For a single-RPC-per-connection client the practical impact is low,
but the spec-mandated echo check is absent and untested.

**Ruling: FIX-NOW — use a non-constant ID and assert the echo.**

Fix-now is warranted because:
1. The change is in the same decode block targeted by Ruling U (type validation)
   — minimal incremental diff cost.
2. A non-constant ID removes a future source of confusion when multi-RPC
   connections are added.
3. The spec contract is explicit and the check is trivially implementable.

**Canonical fix in `dispatch()`:**

```go
// Generate a per-call request ID (not constant) so response correlation is
// verifiable. The format is a short hex string — any non-empty opaque string
// satisfies ADR-012 §3 step 6.
reqID := fmt.Sprintf("%x", time.Now().UnixNano())

req := rpcRequestMsg{
    Type:    "request",
    ID:      reqID,
    Command: command,
    Args:    args,
}
```

After decoding the response, add:

```go
// Verify response ID echoes request ID (ADR-012 §3 step 6 correlation check).
if resp.ID != reqID {
    return nil, fmt.Errorf("rpc failed: %s: response id %q does not match request id %q", command, resp.ID, reqID)
}
```

**Note:** `time.Now().UnixNano()` produces a unique-enough ID for a one-shot CLI
where only one in-flight RPC exists per connection lifetime. A UUID or random hex
is also acceptable.

**Downstream changes required by Ruling X:**

- **Implementer (S-6.03):** Apply the two-part fix to `dispatch()`:
  1. Replace `ID: "1"` with a generated `reqID`.
  2. After decoding, add `resp.ID != reqID` guard returning a descriptive error.
- **Story-writer (S-6.03):** Add to AC-010 (or as a separate AC-012):
  > The `dispatch()` function generates a non-constant request `id` (e.g., a
  > time-based hex string). After decoding the response, it verifies
  > `resp.ID == req.ID`; a mismatch returns E-RPC-001.
  > Test: `TestDispatch_IDEchoEnforced` — mock server sends a response with
  > a different `id` than the request; verify `dispatch()` returns a non-nil
  > error.
- **Product-owner:** Append to BC-2.07.002 PC-3:
  > The `id` field in the RPC request envelope is a client-generated non-constant
  > value. The client verifies that the server echoes the same `id` in the
  > response. A mismatch is treated as a protocol error (E-RPC-001).

---

### Ruling Y — `Inv-2` vs `Inv-4` Spec Conflict: Auth-Handshake-Timeout → E-NET-001 (MEDIUM / spec-reconciliation / S-6.03)

**Finding:**
BC-2.07.003 has two invariants that conflict:

- **Inv-2:** "A timeout is treated as unreachable — sbctl does not hang indefinitely."
- **Inv-4:** "`E-NET-001` is emitted ONLY on `net.Dial`/`net.DialContext` failure
  (daemon unreachable). Key-load failures produce `E-CFG-010`; post-auth RPC
  dispatch failures produce `E-RPC-001`. These three failure modes are distinct
  and MUST NOT share an error code."

The implementation in `connectAndRun` (lines 318–329) handles a `net.Error.Timeout()`
from `Authenticate()` by emitting E-NET-001:

```go
var netErr net.Error
if errors.As(err, &netErr) && netErr.Timeout() {
    msg := fmt.Sprintf("daemon unreachable: %s: connection timed out", target)
    writeError(useJSON, "E-NET-001", msg)
    return fmt.Errorf("E-NET-001: %s", msg)
}
```

This is consistent with Inv-2 ("timeout = unreachable") but violates the literal
reading of Inv-4 ("E-NET-001 ONLY on dial failure"). The implementation and
`TestSbctl_ConnectionTimeout` resolve toward E-NET-001.

**Ruling: FIX-NOW in the spec — amend BC-2.07.003 Inv-4 to explicitly permit
auth-handshake-read-deadline timeout → E-NET-001, consistent with Inv-2.**

Option (a) is the correct resolution: the handshake read-deadline timeout means
the daemon accepted the TCP connection but then went silent before completing the
handshake. From the operator's perspective, the daemon is effectively unreachable
for management purposes — the same action is warranted: check the daemon. Reporting
E-NET-001 with message "connection timed out" is more actionable than a new code
that says "handshake timed out" — the operator checks connectivity either way.

**Canonical Inv-4 amendment (product-owner must apply to BC-2.07.003):**

Replace the current Inv-4 text with:

> **Inv-4:** `E-NET-001` is emitted for two cases: (a) `net.Dial`/`net.DialContext`
> failure — daemon connection refused or DNS failure; (b) handshake read-deadline
> timeout — the daemon accepted the TCP connection but did not complete the
> ADR-012 challenge-response handshake within the timeout budget (treated as
> unreachable per Inv-2). Key-load failures produce `E-CFG-010`; post-auth
> (post-AUTH_OK) RPC dispatch failures produce `E-RPC-001`. These failure modes
> MUST NOT share codes. The message format for case (b) is:
> `"daemon unreachable: <address>: connection timed out"` — the same address
> field used in case (a) to direct the operator's attention to the right target.

**No implementation change required.** The code is correct. Only BC-2.07.003
Inv-4 needs updating.

**Downstream changes required by Ruling Y:**

- **Product-owner:** Amend BC-2.07.003 Inv-4 with the canonical text above.
  No other BC changes needed.
- **Story-writer (S-6.03):** No AC changes needed. AC-007 (`TestSbctl_ConnectionTimeout`)
  already exercises the timeout path correctly. The AC's comment may be updated to
  cite Ruling Y for traceability.
- **Implementer:** No code change.

---

## Product-Owner Handoff (v1.5 — Wave-5 Convergence Round-4 Rulings O–Y)

This section supplements the v1.4 Product-Owner Handoff. Apply each item exactly
as written. Items here supersede v1.4 instructions for the same clause where they
conflict.

### BC-2.07.004 Changes (Round-4)

**EC-013 amendment (Ruling O — stale-socket pre-bind cleanup):** Append to the
existing EC-013 row:
> Stale-socket restart resilience: `listenUnixMgmt` checks for and removes a
> stale socket inode at the bind path before calling `Bind`. Only
> `ModeSocket`-mode inodes are removed; regular files and directories at the
> path cause `Bind` to fail with the original error (not silently clobbered).
> This prevents an EADDRINUSE restart DoS after a non-graceful exit (SIGKILL,
> crash, OOM kill). The Lstat→Remove→Bind TOCTOU window is accepted: it repairs
> exactly the restart-after-crash case and is no worse than the current failure.

**PC-6 amendment (Ruling R — handler execution timeout):** Append to PC-6:
> Handler execution timeout: registered handler `Fn` functions are invoked with
> a child context derived via `context.WithTimeout(ctx, RPCIdleTimeout)`. A
> handler that does not return within `RPCIdleTimeout` is cancelled via the child
> context; the server responds with E-RPC-011. This bounds handler execution to
> the same time budget as the RPC-phase read deadline, closing the CWE-400
> goroutine-pin surface on the handler dispatch path.

### BC-2.07.002 Changes (Round-4)

**PC-3 amendment (Ruling U — dispatch response-type validation):** Append to the
existing PC-3 text (which already has the Ruling M wire-type annotation):
> Receiving-side wire-type validation: after decoding the RPC response,
> `dispatch()` MUST verify `resp.Type == "response"`. Any other value (e.g.,
> `"rpc_response"`, `"auth_fail"`, `""`) is treated as a protocol error and
> returned as E-RPC-001. This closes the gap where a wrong-type response with
> `"ok":true` would be silently accepted as a successful RPC.

**PC-3 amendment (Ruling X — ID echo):** Append to BC-2.07.002 PC-3:
> The `id` field in the RPC request envelope is a client-generated non-constant
> value (not always `"1"`). The client verifies that the server echoes the same
> `id` in the response. A mismatch is treated as a protocol error (E-RPC-001).

### BC-2.07.003 Changes (Round-4)

**Inv-4 replacement (Ruling Y — auth-handshake-timeout → E-NET-001):** Replace
the current Inv-4 text with:
> **Inv-4:** `E-NET-001` is emitted for two cases: (a) `net.Dial`/`net.DialContext`
> failure — daemon connection refused or DNS failure; (b) handshake read-deadline
> timeout — the daemon accepted the TCP connection but did not complete the
> ADR-012 challenge-response handshake within the timeout budget (treated as
> unreachable per Inv-2). Key-load failures produce `E-CFG-010`; post-auth
> (post-AUTH_OK) RPC dispatch failures produce `E-RPC-001`. These failure modes
> MUST NOT share codes. The E-NET-001 message for case (b) is:
> `"daemon unreachable: <address>: connection timed out"`.

**Inv-2 amendment (Ruling V — dispatch read deadline):** Append to Inv-2:
> The `dispatch()` function also sets a read deadline before decoding the RPC
> response, derived from the context deadline (or `RPCIdleTimeout`-equivalent
> as fallback). `sbctl` does not hang indefinitely on the RPC response phase.

---

## Story-Writer Handoff (v1.5 — Wave-5 Convergence Round-4 Rulings O–Y)

### S-W5.01 Changes (Round-4)

**New AC-019 (Ruling O — stale-socket pre-bind cleanup):**
> AC-019 (traces to BC-2.07.004 EC-013, Ruling O) `listenUnixMgmt` performs a
> pre-bind cleanup: if `os.Lstat(path)` succeeds and the result has `ModeSocket`
> set in its mode bits, `os.Remove(path)` is called before `syscall.Bind`. Regular
> files and directories at the path are NOT removed (Bind fails with original error).
>
> Test: `TestListenUnixMgmt_PreBindCleanup_AC019` — (a) create a temp path; call
> `listenUnixMgmt` twice on the same path (first call succeeds; close without
> unlinking so inode persists; second call must also succeed — no EADDRINUSE);
> (b) create a regular file at the same path; call `listenUnixMgmt`; verify it
> returns a non-nil error and the regular file was NOT removed.

**New AC-020 (Ruling R — per-handler execution timeout):**
> AC-020 (traces to BC-2.07.004 PC-6 / Ruling R) Every registered handler `Fn` is
> invoked with `context.WithTimeout(ctx, RPCIdleTimeout)` (default 30 s). A handler
> that blocks past this budget is cancelled; the server returns E-RPC-011 to the
> client. The connection is NOT closed.
>
> Test: `TestMgmtServer_HandlerTimeout_AC020` — register a handler that blocks until
> its context is cancelled; construct server with short `RPCIdleTimeout` override
> (50 ms via `WithRPCIdleTimeout` option or equivalent); authenticate and send RPC;
> verify response is E-RPC-011 within ~200 ms; verify connection remains open.

**AC-017 amendment (Ruling P — fatal-accept-error drain):** Append to AC-017:
> On the fatal-accept-error path (Accept returns a non-transient error while ctx
> is live and Shutdown was never called), `closeAllConns()` is called immediately
> before `connWG.Wait()` so in-flight goroutines are force-closed and drain
> completes quickly.
>
> Test: `TestServe_FatalAcceptErrorDrainsQuickly` — stall an authenticated
> connection; close the listener directly (ctx live, no Shutdown); assert `Serve`
> returns within 200 ms (not blocked up to `RPCIdleTimeout`).

**AC-001 + EC-001 amendment (Ruling Q — propagation gap for Ruling K):**
> In AC-001 body, replace "On deadline expiry the connection is closed with E-ADM-010"
> with "On HandshakeTimeout expiry (silent stall) the connection is closed WITHOUT
> sending AUTH_FAIL — a non-responsive client would not read it (BC-2.07.004 EC-001
> v1.4 / Ruling K). Non-timeout decode errors DO send AUTH_FAIL before close."
>
> In EC-001 table row, replace "E-ADM-010 + close" with "close-only (no AUTH_FAIL
> on timeout); non-timeout decode errors send AUTH_FAIL before close".

### S-6.03 Changes (Round-4)

**New AC-011 (Ruling V — dispatch read deadline + ctx-first signature):**
> AC-011 (traces to BC-2.07.003 Inv-2, ADR-012 §7, Ruling V) `dispatch()` takes
> `ctx context.Context` as its first parameter (go.md rule 7). Before decoding the
> RPC response, it calls `conn.SetReadDeadline` derived from `ctx.Deadline()` (or
> 30 s fallback if ctx has no deadline). The deadline is cleared after decode.
>
> Test: `TestDispatch_RespReadDeadlineEnforced` — mock server completes AUTH_OK then
> hangs (never writes response); `dispatch()` with a 50 ms context deadline; verify
> non-nil error within ~200 ms.

**AC-010 amendment (Ruling U — response-type validation):** Add to AC-010:
> `dispatch()` validates `resp.Type == "response"` after decoding. Any other value
> is rejected with a non-nil error containing the received type string (E-RPC-001).
>
> Fix existing fixture: `TestSbctl_RPCDispatchFailure_ExitsOneWithERPC001` mock
> server response must use `"type":"response"` (not `"type":"rpc_response"`) — the
> test exercises the `"ok":false` path, which requires a valid type first.
>
> New test: `TestDispatch_RejectsWrongResponseType` — mock sends
> `{"type":"rpc_response","id":"1","ok":true,"data":{}}`;
> verify `dispatch()` returns non-nil error.

**AC-010 amendment (Ruling X — non-constant ID + echo check):** Add to AC-010:
> `dispatch()` generates a non-constant request `id`. After decode, it verifies
> `resp.ID == req.ID`; mismatch returns E-RPC-001.
>
> New test: `TestDispatch_IDEchoEnforced` — mock server sends response with
> different `id` than request; verify non-nil error returned.
