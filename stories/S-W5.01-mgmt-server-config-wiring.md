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
version: "1.0"
bc_traces:
  - BC-2.07.004
  - BC-2.09.003
vp_traces: [VP-064, VP-065, VP-066]
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
acceptance_criteria_count: 12
# BC status: active — all BCs are final and assigned
---

# S-W5.01: internal/mgmt Server, Config Additions, and cmd/switchboard Wiring

> **Execute:** `/vsdd-factory:deliver-story S-W5.01`

## Scope Note

This story implements the **server side** of the ADR-012 management plane:

1. `internal/mgmt` package: `Server`, `NewServer`, `OperatorKeySet`, `Serve`,
   `Shutdown`, `MaxMessageBytes`, challenge generation, ADR-012 auth handshake,
   bounded reads, handler dispatch, JSON envelope wrapping.
2. `internal/config` additions: `ManagementSocket` and `AuthorizedOperatorKeys`
   fields; `Validate()` extensions for E-CFG-008 and E-CFG-009.
3. `cmd/switchboard` wiring: start the management listener in all four daemon
   modes (router, access, console, control); WaitGroup-track the mgmt goroutine
   per ARCH-01.

**Server uses stdlib `crypto/ed25519` only — no `golang.org/x/crypto` dependency.
`golang.org/x/crypto` is used only by S-6.03 (`cmd/sbctl`) for OpenSSH PEM
parsing.**

The sbctl client (S-6.03) does NOT need to be merged before this story can begin
development — the wire protocol is fully specified in ADR-012. Both can develop
in parallel on separate branches; integration requires both (S-W5.02).

## Behavioral Contracts

| BC | Title | PCs covered |
|----|-------|------------|
| BC-2.07.004 | Daemon Management Server Authenticates All Connections via Ed25519 Challenge-Response (Fail-Closed) | PC-1 (challenge issued), PC-2 (unauth rejected), PC-3 (replay rejected), PC-4 (auth fail closes connection), PC-5 (all RPCs require auth), PC-6 (bounded reads CWE-400), PC-7 (AUTH_OK), PC-8 (constant-time comparison), PC-9 (bootstrap mode), PC-10 (graceful shutdown) |
| BC-2.09.003 | Router Startup Fails Cleanly on Malformed Config (v1.6) | PC-10 (management_socket validation: E-CFG-008), PC-11 (authorized_operator_keys PEM validation: E-CFG-009) |

## Narrative

- **As a** Switchboard daemon (router, access, console, or control mode)
- **I want to** start an Ed25519-authenticated management server on my management
  socket before accepting data-plane connections
- **So that** `sbctl` operators can authenticate and issue management RPCs
  securely without any unauthenticated access path

## Acceptance Criteria

### AC-001 (traces to BC-2.07.004 postcondition 1 — challenge issued immediately)
On every new connection, `mgmt.Server` sends a CHALLENGE message as the **first**
action before reading any client data:
`{"type":"challenge","nonce":"<base64url 32 bytes>","daemon_sig":"<base64url sig>"}`.
The nonce is 32 bytes from `crypto/rand.Read`. The `daemon_sig` is
`ed25519.Sign(daemonPrivKey, nonceBytes)`.
- **Test:** `TestMgmtServer_IssuesChallengeFirst_AC001` — connect via `net.Pipe`;
  verify the first message received from the server has `"type":"challenge"`,
  a non-empty `"nonce"` (32 bytes when decoded), and a non-empty `"daemon_sig"`.

### AC-002 (traces to BC-2.07.004 postcondition 2 — unauthenticated connections rejected, VP-064)
A connection that sends a CHALLENGE_RESPONSE with either (a) an unrecognized
public key or (b) a signature that does not verify against the presented public
key receives `{"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}`
and the connection is immediately closed. No RPC handler is called.
- **Test:** `TestMgmtServer_RejectsUnauthenticated_VP064` — table-driven, three
  sub-cases: (a) no CHALLENGE_RESPONSE at all (connection closed by server after
  timeout), (b) unrecognized public key, (c) recognized key with wrong signature.
  Verify AUTH_FAIL response and no RPC dispatch for sub-cases (b) and (c).

### AC-003 (traces to BC-2.07.004 postcondition 3 — replay rejection, VP-065)
A nonce recorded during a successful auth handshake on connection C1 cannot be
replayed on connection C2: the server issues a fresh nonce on C2 (cryptographic
guarantee), so `ed25519.Verify(pubkey, newNonce, oldSig)` returns false →
AUTH_FAIL. Per-connection nonce set is cleared on connection close.
- **Test:** `TestMgmtServer_RejectsReplayedNonce_VP065` — per VP-065.md proof
  harness: two `net.Pipe` connections; record nonce1 + sig1 from C1; present
  sig1 on C2 (which has a different nonce); verify AUTH_FAIL on C2.

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

### AC-007 (traces to BC-2.07.004 postcondition 7 — successful authentication)
A client presenting a valid authorized key with a correct signature for the
challenge nonce receives `{"type":"auth_ok","daemon_version":"<semver>"}`.
Subsequent `{"type":"request","id":"<id>","command":"<cmd>","args":{}}` messages
are dispatched to the registered handler and receive a response wrapped in the
JSON envelope from `interface-definitions.md §JSON Output Schema`.
- **Test:** `TestMgmtServer_AuthOK_DispatchesRPC_AC007` — use an authorized key,
  verify AUTH_OK received; send a test RPC; verify the response envelope has
  `"ok":true` and the handler's return value in `"data"`.

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
- **Test:** `TestConfig_Validate_ManagementSocket_E_CFG_008_AC011` — three
  sub-cases: (a) `management_socket: ""` → E-CFG-008 error; (b)
  `management_socket: "   "` → E-CFG-008 error; (c) `management_socket` field
  absent → no error; (d) `management_socket: "/run/sb.sock"` → no error.

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

## Wiring Pattern (all four daemon modes)

Per ARCH-12 §Wiring into cmd/switchboard, each of the four `runXxx` functions in
`cmd/switchboard` follows this pattern:

```go
operatorKeys := mgmt.NewOperatorKeySet(cfg.AuthorizedOperatorKeys)
mgmtLn, err := net.Listen("unix", cfg.ManagementSocket) // or "tcp" for console
// ... handle err
handlers := buildXxxHandlers(...)
mgmtSrv := mgmt.NewServer(mgmtLn, daemonPrivKey, operatorKeys, handlers)
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
| EC-001 | Client connects and sends nothing (no CHALLENGE_RESPONSE) | Server sends CHALLENGE, waits up to connection timeout; times out → E-ADM-010 + close; no goroutine leak |
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
| This story spec | ~2,500 |
| BC-2.07.004.md (v1.0) | ~1,800 |
| BC-2.09.003.md (v1.6, PC-10/PC-11 sections) | ~1,500 |
| ARCH-12 §internal/mgmt Package Design + §ADR-012 + §Wiring | ~2,500 |
| ARCH-01 §Goroutine WaitGroup Contract | ~300 |
| ARCH-05 §Daemon Management Socket | ~400 |
| interface-definitions.md §JSON Output Schema | ~400 |
| VP-064.md + VP-065.md + VP-066.md (proof harnesses) | ~2,000 |
| internal/config/config.go (existing) | ~1,500 |
| cmd/switchboard/main.go (existing) | ~2,000 |
| Test files (estimated) | ~2,500 |
| Tool outputs overhead | ~500 |
| **Total** | **~17,900** |
| Agent context window | 200K |
| **Budget usage** | **~9.0%** |

## Tasks (MANDATORY)

1. [ ] Read BC-2.07.004 (full), BC-2.09.003 (PC-10/PC-11 sections), ARCH-12 (full), VP-064.md, VP-065.md, VP-066.md
2. [ ] Read ARCH-01 §Goroutine WaitGroup Contract; ARCH-05 §Daemon Management Socket
3. [ ] Read `internal/config/config.go` and existing `Validate()` implementation
4. [ ] Read `cmd/switchboard/main.go` to understand existing daemon mode structure
5. [ ] Write failing tests for AC-001 through AC-012 (Red Gate)
6. [ ] Verify Red Gate — all tests must fail before implementation starts
7. [ ] Create `internal/mgmt/mgmt.go`:
   - `const MaxMessageBytes = 1 << 16`
   - `type OperatorKeySet` with `NewOperatorKeySet(keys []ed25519.PublicKey)` and
     `IsAuthorized(pubkey ed25519.PublicKey) bool` (constant-time comparison)
   - `type Handler struct { Command string; Fn func(ctx context.Context, args json.RawMessage) (any, error) }`
   - `type Server struct` (unexported fields: listener, daemonKey, ops, handlers, wg)
   - `func NewServer(ln net.Listener, daemonKey ed25519.PrivateKey, ops *OperatorKeySet, handlers []Handler) *Server`
   - `func (s *Server) Serve(ctx context.Context) error` — accept loop, per-connection goroutine, WaitGroup-tracked
   - `func (s *Server) Shutdown(ctx context.Context) error` — drain + close listener
   - `handleConnection(ctx, conn)` — ADR-012 auth handshake: send CHALLENGE, read CHALLENGE_RESPONSE, verify, send AUTH_OK or AUTH_FAIL, dispatch RPC
   - All reads via `io.LimitReader(conn, MaxMessageBytes)`
8. [ ] Add `ManagementSocket string` and `AuthorizedOperatorKeys []string` to `internal/config/config.go`
9. [ ] Extend `Validate()` with E-CFG-008 and E-CFG-009 (exhaustive error collection per existing pattern)
10. [ ] Wire mgmt listener into each of the four daemon mode `runXxx` functions in `cmd/switchboard`
11. [ ] `just fmt && just lint` pass
12. [ ] Verify VP-064, VP-065, VP-066 test assertions pass

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

**No external dependencies** added by this story on the server side.

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `internal/mgmt/mgmt.go` | create | Full `internal/mgmt` package: constants, types, `Server`, `OperatorKeySet`, `Handler`, `NewServer`, `Serve`, `Shutdown`, `handleConnection` |
| `internal/mgmt/mgmt_test.go` | create | Unit and integration tests: AC-001 through AC-010; VP-064, VP-065, VP-066 test harnesses; fuzz target `FuzzMgmtServer_BoundedRead_VP066` |
| `internal/config/config.go` | modify | Add `ManagementSocket string` and `AuthorizedOperatorKeys []string` fields; extend `Validate()` with E-CFG-008 and E-CFG-009 |
| `internal/config/config_test.go` | modify | Add `TestConfig_Validate_ManagementSocket_E_CFG_008_AC011` and `TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012` |
| `cmd/switchboard/main.go` (or per-mode run files) | modify | Wire mgmt listener start into `runRouter`, `runAccess`, `runConsole`, `runControl`; apply mode-specific socket defaults when `ManagementSocket` is empty |

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.0 | 2026-06-28 | story-writer | Initial creation — Wave-5 net-new story per ARCH-12 product-owner handoff. internal/mgmt server + config E-CFG-008/009 + cmd/switchboard wiring. |
