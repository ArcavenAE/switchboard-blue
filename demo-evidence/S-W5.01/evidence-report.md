# Demo Evidence Report — S-W5.01: internal/mgmt Server, Config Additions, cmd/switchboard Wiring

## Header

| Field | Value |
|-------|-------|
| Story | S-W5.01 |
| Story spec version | 1.6 |
| HEAD SHA | 5be25ef |
| Branch | feature/S-W5.01-mgmt-server-config-wiring |
| Date | 2026-06-29 |
| `go build ./...` | PASS — no compilation errors |
| `go test -race` (internal/mgmt) | PASS — all tests pass |
| `go test -race` (internal/config) | PASS — all tests pass |
| `go test -race` (cmd/switchboard) | PASS — all tests pass |

Overall race results:
```
ok  	github.com/arcavenae/switchboard/internal/mgmt	4.776s
ok  	github.com/arcavenae/switchboard/internal/config	1.870s
ok  	github.com/arcavenae/switchboard/cmd/switchboard	1.995s
```

VHS/recording tool availability: VHS not available in this environment.
Format used: test-transcript + binary-demo format (same as S-6.03 and S-6.01 precedent).

---

## Demo File Index

| File | ACs Covered |
|------|-------------|
| [AC-001-003-004-challenge-handshake-auth.txt](AC-001-003-004-challenge-handshake-auth.txt) | AC-001, AC-003, AC-004 |
| [AC-002-005-008-009-010-013-remaining-acs.txt](AC-002-005-008-009-010-013-remaining-acs.txt) | AC-002, AC-005, AC-008, AC-009, AC-010, AC-013 |
| [AC-006-bounded-reads-vp066.txt](AC-006-bounded-reads-vp066.txt) | AC-006 (+ fuzz baseline) |
| [AC-007-daemon-version-erpc010-erpc011.txt](AC-007-daemon-version-erpc010-erpc011.txt) | AC-007 |
| [AC-011-012-config-validation-ecfg008-ecfg009.txt](AC-011-012-config-validation-ecfg008-ecfg009.txt) | AC-011, AC-012 |
| [AC-014-019-unix-socket-perms-console-tcp-pre-bind.txt](AC-014-019-unix-socket-perms-console-tcp-pre-bind.txt) | AC-014, AC-019 |
| [AC-015-016-ephemeral-key-nilkey-panic.txt](AC-015-016-ephemeral-key-nilkey-panic.txt) | AC-015, AC-016 |
| [AC-017-serve-shutdown-predicate-vp069.txt](AC-017-serve-shutdown-predicate-vp069.txt) | AC-017 (a–f) |
| [AC-018-020-write-deadlines-handler-timeout-vp072.txt](AC-018-020-write-deadlines-handler-timeout-vp072.txt) | AC-018, AC-020 |

---

## Per-AC Evidence Summary

### AC-001 — Challenge Issued First (BC-2.07.004 PC-1)

**Proving tests:** `TestMgmtServer_IssuesChallengeFirst_AC001`,
`TestMgmtServer_HandshakeTimeout_SilentStall_AC001`

**Transcript:** [AC-001-003-004-challenge-handshake-auth.txt](AC-001-003-004-challenge-handshake-auth.txt)

```
--- PASS: TestMgmtServer_IssuesChallengeFirst_AC001 (0.01s)
--- PASS: TestMgmtServer_HandshakeTimeout_SilentStall_AC001 (0.06s)
```

Server sends `{"type":"challenge","nonce":"<base64url 32B>","daemon_sig":"<base64url>"}` as the FIRST
message. Silent stall: connection closed after 50ms injected HandshakeTimeout — no AUTH_FAIL on timeout
(close-only per Ruling K / BC-2.07.004 EC-001 v1.4). mgmt.HandshakeTimeout==10s verified.

---

### AC-002 — Unauthenticated Connections Rejected (VP-064, BC-2.07.004 PC-2)

**Proving tests:** `TestMgmtServer_RejectsUnauthenticated_VP064`

**Transcript:** [AC-002-005-008-009-010-013-remaining-acs.txt](AC-002-005-008-009-010-013-remaining-acs.txt)

```
--- PASS: TestMgmtServer_RejectsUnauthenticated_VP064 (0.00s)
    --- PASS: TestMgmtServer_RejectsUnauthenticated_VP064/unrecognized_public_key (0.01s)
    --- PASS: TestMgmtServer_RejectsUnauthenticated_VP064/recognized_key_wrong_signature (0.01s)
```

AUTH_FAIL (E-ADM-010) returned for both unrecognized key and recognized key + wrong signature.
Identical response format — no oracle. Connection closed; no RPC dispatched.

---

### AC-003 — Post-Auth Structural Guard (VP-065, BC-2.07.004 PC-3)

**Proving tests:** `TestMgmtServer_PostAuthChallengeResponseRejected_VP065`

**Transcript:** [AC-001-003-004-challenge-handshake-auth.txt](AC-001-003-004-challenge-handshake-auth.txt)

```
--- PASS: TestMgmtServer_PostAuthChallengeResponseRejected_VP065 (0.02s)
```

Second `challenge_response` on an authenticated connection → AUTH_FAIL + E-ADM-010 + close.
Per-connection `authenticated` boolean (not nonce-set) causes rejection (Ruling 7).

**Note:** BC-2.07.004 PC-3/EC-004 security-event logging DEFERRED to S-HRD.02 (daemon logging
infrastructure / slog seam). This AC covers fail-closed connection control only.

---

### AC-004 — Auth Fail Closes Connection (BC-2.07.004 PC-4)

**Proving tests:** `TestMgmtServer_AuthFailClosesConnection_AC004`

**Transcript:** [AC-001-003-004-challenge-handshake-auth.txt](AC-001-003-004-challenge-handshake-auth.txt)

```
--- PASS: TestMgmtServer_AuthFailClosesConnection_AC004 (0.02s)
```

After AUTH_FAIL: next client read returns error (connection closed). rpcCallCount==0.

---

### AC-005 — RPC Without Auth Rejected (BC-2.07.004 PC-5)

**Proving tests:** `TestMgmtServer_RPCWithoutAuth_Rejected_AC005`

**Transcript:** [AC-002-005-008-009-010-013-remaining-acs.txt](AC-002-005-008-009-010-013-remaining-acs.txt)

```
--- PASS: TestMgmtServer_RPCWithoutAuth_Rejected_AC005 (0.00s)
```

Sending `type=request` before CHALLENGE_RESPONSE → AUTH_FAIL (E-ADM-010) + close. No RPC dispatched.

---

### AC-006 — Bounded Reads / VP-066 (BC-2.07.004 PC-6)

**Proving tests:** `TestMgmtServer_BoundedRead_VP066`, `FuzzMgmtServer_BoundedRead_VP066`

**Transcript:** [AC-006-bounded-reads-vp066.txt](AC-006-bounded-reads-vp066.txt)

```
--- PASS: TestMgmtServer_BoundedRead_VP066 (0.01s)
    --- PASS: TestMgmtServer_BoundedRead_VP066/under_limit_challenge_consumed (0.00s)
    --- PASS: TestMgmtServer_BoundedRead_VP066/oversized_message_closes_connection (2.00s)
--- PASS: FuzzMgmtServer_BoundedRead_VP066 (0.51s)
    --- PASS: FuzzMgmtServer_BoundedRead_VP066/seed#0 (0.10s)
    --- PASS: FuzzMgmtServer_BoundedRead_VP066/seed#1 (0.20s)
    --- PASS: FuzzMgmtServer_BoundedRead_VP066/seed#2 (0.10s)
    --- PASS: FuzzMgmtServer_BoundedRead_VP066/seed#3 (0.10s)
```

MaxMessageBytes+1 payload → connection closed, no OOM. io.LimitReader applied at every decode site.
Fuzz seed corpus (4 entries) — no panics, no deadlocks.

---

### AC-007 — daemon_version, E-RPC-010, E-RPC-011 (BC-2.07.004 PC-7, PC-11, PC-12)

**Proving tests:** `TestMgmtServer_DaemonVersion_Injected_AC007`,
`TestMgmtServer_AuthOK_DispatchesRPC_AC007`, `TestUnknownCommand_ReturnsERPC010_VP070`,
`TestHandlerError_ReturnsERPC011_VP071`

**Transcript:** [AC-007-daemon-version-erpc010-erpc011.txt](AC-007-daemon-version-erpc010-erpc011.txt)

```
--- PASS: TestMgmtServer_DaemonVersion_Injected_AC007 (0.00s)
    --- PASS: TestMgmtServer_DaemonVersion_Injected_AC007/auth_ok_carries_injected_version (0.02s)
    --- PASS: TestMgmtServer_DaemonVersion_Injected_AC007/empty_daemonVersion_panics (0.01s)
--- PASS: TestMgmtServer_AuthOK_DispatchesRPC_AC007 (0.02s)
--- PASS: TestUnknownCommand_ReturnsERPC010_VP070 (0.02s)
--- PASS: TestHandlerError_ReturnsERPC011_VP071 (0.02s)
```

AUTH_OK carries `daemon_version="0.1.0-test"` (not hardcoded). Empty string panics at construction.
Unknown command → E-RPC-010 in-band, connection stays open. Handler error → E-RPC-011 in-band.
E-RPC-001 and E-RPC-002 absent from internal/mgmt (Ruling C compliant).

---

### AC-008 — Constant-Time Comparison (BC-2.07.004 PC-8, Inv-5)

**Proving tests:** `TestOperatorKeySet_ConstantTimeCompare_AC008`

**Transcript:** [AC-002-005-008-009-010-013-remaining-acs.txt](AC-002-005-008-009-010-013-remaining-acs.txt)

```
--- PASS: TestOperatorKeySet_ConstantTimeCompare_AC008 (0.00s)
```

`IsAuthorized` uses `subtle.ConstantTimeCompare`. Correctness verified: authorized key → true,
one-byte mutation → false, unrecognized key → false. Timing-oracle property verified by code
inspection (same comparison code path regardless of key recognition state).

---

### AC-009 — Bootstrap Mode (BC-2.07.004 PC-9)

**Proving tests:** `TestMgmtServer_BootstrapMode_DaemonKeyAuthorized_AC009`

**Transcript:** [AC-002-005-008-009-010-013-remaining-acs.txt](AC-002-005-008-009-010-013-remaining-acs.txt)

```
--- PASS: TestMgmtServer_BootstrapMode_DaemonKeyAuthorized_AC009 (0.01s)
```

NewOperatorKeySet(nil).IsBootstrap()==true. Connection signed with daemon's own keypair → AUTH_OK.

---

### AC-010 — Graceful Shutdown (BC-2.07.004 PC-10)

**Proving tests:** `TestMgmtServer_GracefulShutdown_AC010`

**Transcript:** [AC-002-005-008-009-010-013-remaining-acs.txt](AC-002-005-008-009-010-013-remaining-acs.txt)

```
--- PASS: TestMgmtServer_GracefulShutdown_AC010 (0.03s)
    mgmt_test.go:887: Serve returned: <nil>
```

Shutdown() → Serve returns nil → goroutine count returns to baseline within 300ms. WaitGroup-tracked
per ARCH-01 §Goroutine WaitGroup Contract.

---

### AC-011 — management_socket Validation E-CFG-008 (BC-2.09.003 PC-10)

**Proving tests:** `TestConfig_Validate_ManagementSocket_E_CFG_008_AC011`

**Transcript:** [AC-011-012-config-validation-ecfg008-ecfg009.txt](AC-011-012-config-validation-ecfg008-ecfg009.txt)

```
--- PASS: TestConfig_Validate_ManagementSocket_E_CFG_008_AC011 (0.00s)
    --- PASS: .../empty_string_accepted_as_absent (0.00s)
    --- PASS: .../whitespace_only_rejected (0.00s)
    --- PASS: .../tab_only_rejected (0.00s)
    --- PASS: .../valid_unix_path_accepted (0.00s)
    --- PASS: .../valid_tcp_address_accepted (0.00s)
    --- PASS: .../exhaustive_reporting_with_other_errors (0.00s)
```

Empty string accepted as absent (Go zero-value = absent per AC-011 spec, Ruling F5).
Whitespace/tab-only → E-CFG-008. Collect-all-failures pattern verified.

---

### AC-012 — authorized_operator_keys PEM Validation E-CFG-009 (BC-2.09.003 PC-11)

**Proving tests:** `TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012`

**Transcript:** [AC-011-012-config-validation-ecfg008-ecfg009.txt](AC-011-012-config-validation-ecfg008-ecfg009.txt)

```
--- PASS: TestConfig_Validate_AuthorizedOperatorKeys_E_CFG_009_AC012 (0.00s)
    --- PASS: .../invalid_pem_at_index_0 (0.00s)
    --- PASS: .../valid_at_0_invalid_at_1 (0.01s)
    --- PASS: .../empty_list_accepted (0.00s)
    --- PASS: .../nil_list_accepted (0.00s)
    --- PASS: .../valid_ed25519_pem_accepted (0.01s)
    --- PASS: .../multiple_valid_pem_accepted (0.01s)
    --- PASS: .../rsa_pem_rejected_wrong_key_type (0.00s)
    --- PASS: .../wrong_pem_block_type_rejected (0.00s)
    --- PASS: .../exhaustive_reporting_both_bad_entries_collected (0.01s)
```

Invalid PEM, wrong block type, RSA key → E-CFG-009 with per-index error. Both bad entries collected
in one pass (Inv-4 exhaustive reporting). RSA-wrong-type rejection confirmed.

---

### AC-013 — Connection Cap / Bounded Accept Loop (BC-2.07.004 EC-012)

**Proving tests:** `TestMgmtServer_ConnectionCap_AC013`

**Transcript:** [AC-002-005-008-009-010-013-remaining-acs.txt](AC-002-005-008-009-010-013-remaining-acs.txt)

```
--- PASS: TestMgmtServer_ConnectionCap_AC013 (0.09s)
```

WithMaxConnections(3): 3 connections at cap → 4th does NOT receive CHALLENGE within 80ms (back-pressure).
Release one → 4th receives CHALLENGE within 500ms. MaxConcurrentConnections==128 verified.

---

### AC-014 — Unix Socket 0600 Permissions + Console TCP Loopback Rejection (BC-2.07.004 EC-013, VP-073)

**Proving tests:** `TestDaemonWiring_UnixSocketPermissions_AC014`,
`TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback_VP073`,
`TestBuildMgmtListener_ConsoleTCP_CanonicalECFG008Format_RulingL`

**Transcript:** [AC-014-019-unix-socket-perms-console-tcp-pre-bind.txt](AC-014-019-unix-socket-perms-console-tcp-pre-bind.txt)

```
--- PASS: TestDaemonWiring_UnixSocketPermissions_AC014 (0.00s)
--- PASS: TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback_VP073 (0.00s)
    --- PASS: .../0.0.0.0:9091 (0.00s)
    --- PASS: .../127.0.0.1:0 (0.00s)
    --- PASS: .../[::1]:0 (0.00s)
    --- PASS: .../localhost:0 (0.00s)
    ...
--- PASS: TestBuildMgmtListener_ConsoleTCP_CanonicalECFG008Format_RulingL (0.00s)
```

Socket file stat.Mode().Perm() == 0600. listenUnixMgmt uses syscall.Umask(0o177)+syscall.Bind —
no chmod-after-Listen TOCTOU. umaskMu serializes the critical section.
Non-loopback console TCP → E-CFG-008 prefix error (Ruling L canonical format).

---

### AC-015 — Ephemeral Keypair + Fatal-Start Abort (BC-2.07.004 Precondition 3, Ruling J)

**Proving tests:** `TestRunAccess_GeneratesEphemeralKey_AC015`,
`TestRunAccess_MgmtStartFailureAborts_RulingJ`

**Transcript:** [AC-015-016-ephemeral-key-nilkey-panic.txt](AC-015-016-ephemeral-key-nilkey-panic.txt)

```
--- PASS: TestRunAccess_GeneratesEphemeralKey_AC015 (0.02s)
    --- PASS: .../real_key_serves_wellformed_challenge (0.02s)
    --- PASS: .../startMgmtServer_with_nil_key_serves_no_challenge (0.00s)
--- PASS: TestRunAccess_MgmtStartFailureAborts_RulingJ (0.00s)
    --- PASS: .../explicit_socket_unbindable (0.00s)
    --- PASS: .../default_path_unbindable (0.00s)
```

runAccess generates ed25519.GenerateKey(rand.Reader) before startMgmtServer. Key is 64 bytes
(ed25519.PrivateKeySize). nil key → VP-068 guard fires → error returned → data plane NOT started
(Ruling J). startMgmtServer failure → runAccess returns error immediately.

---

### AC-016 — nil/Short-Key Construction Guard (BC-2.07.004 Invariant 8, VP-068)

**Proving tests:** `TestNewServer_PanicsOnNilKey_VP068`, `TestNewServer_PanicsOnShortKey_VP068`

**Transcript:** [AC-015-016-ephemeral-key-nilkey-panic.txt](AC-015-016-ephemeral-key-nilkey-panic.txt)

```
--- PASS: TestNewServer_PanicsOnNilKey_VP068 (0.00s)
--- PASS: TestNewServer_PanicsOnShortKey_VP068 (0.00s)
    --- PASS: .../len_32_public_key_size (0.00s)
    --- PASS: .../len_63_one_short (0.00s)
    --- PASS: .../len_1_single_byte (0.00s)
```

nil key (len==0) → panic at construction. 32-byte (public key size), 63-byte, 1-byte → all panic.
Fail-fast prevents remote-panic DoS via mid-connection nil dereference.

---

### AC-017 — Serve Shutdown Predicate, Graceful Drain (BC-2.07.004 PC-10, VP-069)

**Proving tests:** `TestServe_ReturnsNilOnShutdown_VP069`,
`TestServe_ReturnsNilOnCtxCancel_VP069`,
`TestServe_ReturnsErrOnUnexpectedListenerClose_VP069`,
`TestServe_ShutdownWindowNoAddAfterWaitPanic_RulingI`,
`TestServe_DrainCompletesWithinBudget_RulingI`,
`TestServe_FatalAcceptErrorDrainsQuickly`

**Transcript:** [AC-017-serve-shutdown-predicate-vp069.txt](AC-017-serve-shutdown-predicate-vp069.txt)

```
--- PASS: TestServe_ReturnsNilOnShutdown_VP069 (0.01s)
--- PASS: TestServe_ReturnsNilOnCtxCancel_VP069 (0.01s)
--- PASS: TestServe_ReturnsErrOnUnexpectedListenerClose_VP069 (0.02s)
--- PASS: TestServe_ShutdownWindowNoAddAfterWaitPanic_RulingI (0.19s)
--- PASS: TestServe_DrainCompletesWithinBudget_RulingI (0.01s)
--- PASS: TestServe_FatalAcceptErrorDrainsQuickly (0.03s)
```

Shutdown/ctx-cancel → nil. Unexpected listener close (ctx live, no shutdown) → non-nil error
(Ruling G). Shutdown window: 100 iterations under -race, no panics. Drain within budget: stalled
connection force-closed by closeAllConns() within 2s (not 10s HandshakeTimeout).
Fatal accept error: closeAllConns() before connWG.Wait() → Serve returns within 200ms (Ruling P).

---

### AC-018 — Write Deadlines / Slowloris Defense (BC-2.07.004 PC-1 amended, VP-072)

**Proving tests:** `TestWriteDeadline_SlowlorisDefense_VP072`,
`TestWriteDeadline_RPCResponse_VP072_Round5F4`

**Transcript:** [AC-018-020-write-deadlines-handler-timeout-vp072.txt](AC-018-020-write-deadlines-handler-timeout-vp072.txt)

```
--- PASS: TestWriteDeadline_SlowlorisDefense_VP072 (0.28s)
    VP-072: goroutines before=2 after-connect=5
    VP-072: goroutines at deadline point=4 (before=2)
--- PASS: TestWriteDeadline_RPCResponse_VP072_Round5F4 (0.40s)
    VP-072 RPC-response: goroutines before=2 after-rpc-send=5
    VP-072 RPC-response: goroutines at 350ms measurement=4 (before=2)
```

CHALLENGE write: goroutine count drops from 5 to 4 after 100ms HandshakeTimeout (connection goroutine
exited — write deadline fired on non-draining client). RPC-response write: same measurement with
50ms RPCIdleTimeout — confirms line 661 uses s.rpcIdleTimeout field (not 30s constant).

---

### AC-019 — Stale-Socket Pre-bind Cleanup (BC-2.07.004 EC-013, Ruling O)

**Proving tests:** `TestListenUnixMgmt_PreBindCleanup_AC019`

**Transcript:** [AC-014-019-unix-socket-perms-console-tcp-pre-bind.txt](AC-014-019-unix-socket-perms-console-tcp-pre-bind.txt)

```
--- PASS: TestListenUnixMgmt_PreBindCleanup_AC019 (0.00s)
    --- PASS: .../stale_socket_removed_before_bind (0.00s)
    --- PASS: .../regular_file_not_removed (0.00s)
```

Stale socket inode: Lstat detects ModeSocket → Remove → second bind succeeds (no EADDRINUSE).
Regular file: NOT removed; Bind fails with original error.

---

### AC-020 — Per-Handler Execution Timeout (BC-2.07.004 PC-6, Ruling R)

**Proving tests:** `TestMgmtServer_HandlerTimeout_AC020`

**Transcript:** [AC-018-020-write-deadlines-handler-timeout-vp072.txt](AC-018-020-write-deadlines-handler-timeout-vp072.txt)

```
--- PASS: TestMgmtServer_HandlerTimeout_AC020 (0.06s)
```

Blocking handler (select ctx.Done()) + WithRPCIdleTimeout(50ms) → handler cancelled after ~50ms.
Server responds E-RPC-011 ("context deadline exceeded") within ~200ms. Connection stays OPEN.

---

## Deferred ACs (Full Happy-Path Integration)

The following property cannot be demonstrated with this story alone — it requires both S-W5.01
(server side) and S-6.03 (sbctl client) plus a live multi-process integration harness:

| Deferred Property | Reason | Target Story |
|-------------------|--------|--------------|
| End-to-end sbctl ↔ daemon happy-path handshake + RPC | sbctl client (S-6.03) develops in parallel on feature/S-6.03-sbctl-client-auth; integration requires both to be merged | S-W5.02 (integration harness) |

All server-side behaviors (AC-001 through AC-020) are fully tested and demonstrated using
in-process `net.Pipe` fixtures and real TCP/Unix listeners. No fake implementations or stubs
were used for the behaviors being demonstrated.

---

## Architecture Compliance Spot-Checks

| Constraint | Evidence |
|------------|----------|
| `internal/mgmt` does NOT import data-plane packages | `go list -deps github.com/arcavenae/switchboard/internal/mgmt` contains no routing/multipath/arq/etc. |
| `internal/mgmt` does NOT import `internal/admission` | Same import check; admission not present. |
| stdlib `crypto/ed25519` only (no `golang.org/x/crypto`) | `go.mod` unchanged; internal/mgmt imports only stdlib. |
| All reads via `io.LimitReader(conn, MaxMessageBytes)` | Code: every `json.NewDecoder` in handleConnection wraps `io.LimitReader`. |
| `OperatorKeySet.IsAuthorized` uses `subtle.ConstantTimeCompare` | Code: mgmt.go line 116. |
| mgmt goroutine `wg.Add(1)` / `defer wg.Done()` per ARCH-01 | Code: startMgmtServer in mgmt_wire.go lines 295-299. |
| `NewServer` panics on `daemonVersion==""` and `len(daemonKey)!=64` | Tests: AC-007 + AC-016 verified above. |
