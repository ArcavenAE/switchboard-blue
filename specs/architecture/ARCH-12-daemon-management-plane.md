---
artifact_id: ARCH-12-daemon-management-plane
document_type: architecture-section
level: L3
version: "1.1"
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
func NewServer(
    ln net.Listener,
    daemonKey ed25519.PrivateKey,
    ops *OperatorKeySet,
    handlers []Handler,
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
srv := mgmt.NewServer(ln, daemonPrivKey, operatorKeySet, handlers)
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
    // 3. Start management listener.
    mgmtLn, err := net.Listen("unix", cfg.ManagementSocket)  // or tcp for console
    // 4. Build handler slice (injects data-plane deps by closure).
    handlers := buildRouterHandlers(router, admittedKeySet, cfgPath)
    // 5. Construct and serve.
    mgmtSrv := mgmt.NewServer(mgmtLn, daemonPrivKey, operatorKeys, handlers)
    wg.Add(1)
    go func() { defer wg.Done(); _ = mgmtSrv.Serve(ctx) }()
    // On shutdown: mgmtSrv.Shutdown(shutdownCtx)
}
```

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

The client's `Authenticate(conn net.Conn, privKey ed25519.PrivateKey) error` function
in `cmd/sbctl/client.go` MUST:

1. Read the CHALLENGE message from the server using `json.NewDecoder(io.LimitReader(conn, MaxMessageBytes))`.
2. Decode `nonce_bytes` from the base64url nonce field.
3. Verify `daemon_sig` against the daemon's pubkey (if the daemon pubkey is known /
   pinned — optional in MVP; can be trusted on first use).
4. Sign: `nonce_sig = ed25519.Sign(privKey, nonce_bytes)`.
5. Send CHALLENGE_RESPONSE.
6. Read AUTH_OK or AUTH_FAIL.
7. Return `nil` ONLY if an AUTH_OK message was received and successfully decoded.
   Any other outcome — connection error, malformed message, AUTH_FAIL — MUST return
   a non-nil error. There is no code path that returns `nil` without receiving a
   verified AUTH_OK. This is the fail-closed invariant.

The existing S-6.03 scaffold (`cmd/sbctl/client.go` run/dispatch/JSON-envelope) is
extended with `Authenticate()` as a pre-dispatch step. The connection attempt
(`net.Dial`) and the authentication are separate steps with separate error codes:
- Connection failure: `E-NET-001` (BC-2.07.003)
- Authentication failure: `E-ADM-010` (BC-2.07.002 PC-4)

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
- Note: the e2e VP for this story is deferred to the integration harness and will
  receive a VP number during story decomposition (not pre-assigned in this ARCH wave)

**Dependencies:** S-6.03 AND S-W5.01 must both be merged before this story can begin.

**Estimate:** 5 points

### Summary

| Story | Scope | Est. | Depends On |
|-------|-------|------|-----------|
| S-6.03 (re-scoped) | sbctl client auth + connection error | 5pt | ADR-012 (unlocked) |
| S-W5.01 (new) | internal/mgmt server + config + wiring | 8pt | ADR-012 |
| S-W5.02 (new) | E2E integration harness (e2e VP deferred) | 5pt | S-6.03 + S-W5.01 |

---

## Risk Mitigations

| Risk | Mitigation |
|------|-----------|
| CWE-400 (unbounded socket read) | io.LimitReader(MaxMessageBytes = 64 KiB) on every socket read, client and server (ADR-012 §6) |
| E-ADM-010 oracle (key enumeration) | AUTH_FAIL returns same message for unrecognized and wrong-signature keys |
| DI-002 (private key transit) | Operator private key used only for local Sign(); never serialized or sent |
| Replay across connections | Fresh nonce per connection; per-connection replay prevention |
| init() coupling | Handler registry is constructor-injected; no package-level init() |
| Goroutine leak | mgmt.Serve goroutine is WaitGroup-tracked per ARCH-01 §Goroutine WaitGroup Contract |
| Config schema drift | new fields added to Config struct with Validate() coverage; BC-2.09.003 amendment recommended |
