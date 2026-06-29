---
artifact_id: BC-2.07.004
document_type: behavioral-contract
level: L3
version: "1.1"
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
3. The daemon's own Ed25519 private key is loaded and available (used to sign
   CHALLENGE messages and, in bootstrap mode, as the sole authorized key).
4. A `net.Listener` has been opened on the management socket address (Unix socket
   path or TCP address per ARCH-05 §Daemon Management Socket and ARCH-12 §Wiring).
5. `mgmt.NewServer(ln, daemonKey, operatorKeySet, handlers)` has been called with
   the listener, daemon private key, operator key set, and registered command
   handlers.
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
   challenge.

2. **Unauthenticated connections rejected:** If the client sends a CHALLENGE_RESPONSE
   where either (a) `ed25519.Verify(pubkey, nonceBytes, nonceSig)` returns false, or
   (b) `OperatorKeySet.IsAuthorized(pubkey)` returns false, the server sends:
   ```json
   {"type":"auth_fail","code":"E-ADM-010","message":"authentication failed"}
   ```
   and closes the connection immediately. No RPC commands are processed on this
   connection.

3. **Replay rejection (per-connection):** The nonce issued per PC-1 is recorded in
   the server's per-connection nonce set. If the same `nonce_sig` (or any value that
   decodes to the same nonce bytes) is presented a second time on the same connection,
   the server rejects the second auth attempt with E-ADM-010 and closes. (In the
   standard one-handshake-per-connection protocol this property protects against
   future protocol extensions that might allow renegotiation.)

4. **Auth failure closes connection without processing RPCs:** After sending
   AUTH_FAIL, the server closes the connection. The client receives no RPC response
   data. There is no retry opportunity on the same connection.

5. **All RPC commands require prior auth:** No RPC request (`"type":"request"`)
   is dispatched to any registered handler unless the connection has passed a
   successful auth handshake in this session. A client that skips the handshake and
   sends an RPC request directly receives AUTH_FAIL + close.

6. **Bounded reads (CWE-400):** Every `json.Decoder.Decode` call on the management
   socket — CHALLENGE_RESPONSE, RPC request, and any other message — is preceded by
   `io.LimitReader(conn, MaxMessageBytes)` where `MaxMessageBytes = 1 << 16`
   (64 KiB, defined as `internal/mgmt.MaxMessageBytes`). A connection that sends a
   message exceeding 64 KiB causes the read to terminate with an error and the
   connection to be closed. The process does not OOM.

7. **Successful authentication path:** If both `ed25519.Verify` and `IsAuthorized`
   return true, the server sends AUTH_OK:
   ```json
   {"type":"auth_ok","daemon_version":"<semver>"}
   ```
   and the connection enters the authenticated state. Subsequent RPC requests are
   dispatched to the registered handler for the command name. Responses are wrapped
   in the standard JSON envelope from interface-definitions.md §JSON Output Schema.

8. **Constant-time key comparison:** `OperatorKeySet.IsAuthorized` uses constant-time
   comparison (`subtle.ConstantTimeCompare` or equivalent) to prevent timing oracle
   attacks on key enumeration. Recognized and unrecognized keys receive the same
   E-ADM-010 response — no oracle differentiating "key known but wrong signature"
   from "key not in set."

9. **Bootstrap mode:** When `authorized_operator_keys` is empty in config, the daemon
   accepts connections signed by the daemon's own keypair (the `key_file` key from
   config). The handshake and rejection behavior are identical — only the authorized
   set changes.

10. **Graceful shutdown:** `mgmt.Server.Shutdown(ctx)` drains in-flight connections
    and closes the listener within the context deadline. No new connections are
    accepted after shutdown is initiated. The goroutine is WaitGroup-tracked per
    ARCH-01 §Goroutine WaitGroup Contract.

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

## Trigger

A client connects to the daemon's management socket (Unix or TCP per ARCH-05
§Daemon Management Socket). The server handles each connection in a dedicated
goroutine tracked by the server's internal WaitGroup.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Client connects and sends nothing (no CHALLENGE_RESPONSE) | Server sends CHALLENGE, waits for response up to connection timeout; times out → E-ADM-010 + close. Server does not hang indefinitely. |
| EC-002 | Client sends CHALLENGE_RESPONSE with an unrecognized public key | `OperatorKeySet.IsAuthorized` returns false. Server sends AUTH_FAIL (E-ADM-010, same message as wrong-signature). Connection closed. No oracle. |
| EC-003 | Client sends CHALLENGE_RESPONSE with recognized key but wrong signature | `ed25519.Verify` returns false. Server sends AUTH_FAIL (E-ADM-010). Connection closed. No oracle. |
| EC-004 | Client sends CHALLENGE_RESPONSE and then replays the same nonce_sig again on the same connection | Second auth attempt → E-ADM-010 + close. Per-connection nonce set enforces single-use. |
| EC-005 | Client sends a message > 64 KiB (oversized) | `io.LimitReader` causes `json.Decoder.Decode` to return an error. Server closes connection. No memory allocation beyond 64 KiB for this connection. Process does not OOM. |
| EC-006 | Client connects and sends malformed JSON (not valid JSON object) | `json.Decoder.Decode` returns error. Server closes connection with no response (or sends AUTH_FAIL depending on which decode fails). Process does not panic. |
| EC-007 | Client sends a JSON object of the right size but wrong `"type"` field | Server treats this as a protocol error; closes connection. No RPC dispatched. |
| EC-008 | Client closes connection mid-handshake (after CHALLENGE, before CHALLENGE_RESPONSE) | Server detects EOF/read error; cleans up connection state; no goroutine leak. |
| EC-009 | Non-Switchboard peer sends an arbitrary byte stream | `io.LimitReader` + `json.Decoder` returns error within first 64 KiB. Connection closed cleanly; no panic, no OOM. |
| EC-010 | `authorized_operator_keys` is empty (bootstrap mode) | Daemon's own keypair is the authorized key. Handshake proceeds normally. AUTH_OK on correct daemon-key signature. |
| EC-011 | Client skips handshake and sends an RPC request (`"type":"request"`) as first message | Server has not yet received CHALLENGE_RESPONSE; treats this as an unauthenticated request. Sends AUTH_FAIL + close. RPC handler is never called. |

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
| VP-065 | Server rejects replayed nonce within a connection | integration |
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

## VP Anchors

VP-064, VP-065, VP-066 (minted in this pass; cover PC-2/PC-5, PC-3, and PC-6 respectively)

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.1 | 2026-06-28 | Wave-5 consistency audit F-004: subsystem frontmatter corrected from `SS-07` (ID form) to `network-management` (canonical name) to match sibling BCs BC-2.07.002 and BC-2.07.003. `modified:` array populated. No content changes. |
| 1.0 | 2026-06-28 | Initial draft — daemon-side management auth (ADR-012 server counterpart to BC-2.07.002). Wave-5 BC. |
| 1.0.1 | 2026-06-28 | Patch — subsystem back-filled to SS-07 (network-management); internal/mgmt was already listed under SS-07 in ARCH-INDEX Subsystem Registry. No content changes. |
