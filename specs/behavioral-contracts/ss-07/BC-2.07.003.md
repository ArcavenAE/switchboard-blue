---
artifact_id: BC-2.07.003
document_type: behavioral-contract
level: L3
version: "1.5"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.07.003
subsystem: network-management
architecture_module: cmd/sbctl
capability: CAP-024
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-06-28
    version: "1.2"
    change: >
      Traceability refresh (Wave-5 consistency audit F-001): Stories field filled
      with S-6.03 (the connection-error owner story that cites BC-2.07.003 in its
      bc_traces). Also added E-CFG-002 collision flag (Wave-5 audit F-003): the
      taxonomy defines E-CFG-002 as "private key export not supported" (BC-2.05.007),
      but BC-2.09.003 v1.2 assigned E-CFG-002 to listen_addr invalid host:port.
      This pre-existing inconsistency is now flagged for maintenance-pass resolution.
  - date: 2026-06-28
    version: "1.3"
    change: >
      Wave-5 mgmt-plane adversarial review Ruling 5 (ARCH-12 v1.2): Invariant 4 added
      (E-NET-001 strictly dial/connect-unreachable; E-CFG-010 for key-load failure;
      E-RPC-001 for post-auth dispatch failure; distinct codes must not share); EC-005
      added (key file absent/oversized/malformed/wrong-type → E-CFG-010, no connection
      attempt); EC-006 added (post-AUTH_OK RPC dispatch failure → E-RPC-001).
  - date: 2026-06-28
    version: "1.4"
    change: >
      Wave-5 mgmt-plane adversarial review: tilde/home expansion of sbctl --key path →
      EC-007 added (~ prefix expanded via os.UserHomeDir() before file open; expansion
      failure → E-CFG-010; successful expansion then file-read failure → E-CFG-010 with
      expanded path; ~user not required); Precondition 3 added (key path supports ~
      prefix); anchors S-6.03 AC-008.
  - date: 2026-06-29
    version: "1.5"
    change: >
      ARCH-12 v1.5 Wave-5 Convergence Rulings V and Y: (V) Invariant 2 extended —
      dispatch() also sets a read deadline before decoding the RPC response, derived
      from ctx.Deadline() (or 30 s RPCIdleTimeout-equivalent fallback if ctx has no
      deadline); sbctl does not hang indefinitely on the RPC response phase; dispatch()
      accepts context.Context as its first parameter (go.md rule 7). (Y) Invariant 4
      replaced — E-NET-001 is now explicitly permitted for two cases: (a) net.Dial/
      net.DialContext failure; (b) handshake read-deadline timeout (treated as
      unreachable per Inv-2); Inv-4 now explicitly reconciles with Inv-2 and captures
      the E-NET-001 message format for the timeout case.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-024]
kos_anchors:
  - elem-single-binary-three-modes
---

# Behavioral Contract BC-2.07.003: sbctl Reports Clear Connection Error When Target Daemon Is Unreachable

## Description

When `sbctl` cannot connect to the target daemon (daemon not running, wrong port, firewall, timeout), it reports a clear and actionable connection error. It never produces a misleading "success" output or a cryptic panic. The error message includes the attempted address and the failure reason. Exit code is non-zero.

## Preconditions

1. The operator runs an sbctl command targeting a specific daemon address.
2. The daemon is not reachable (any reason: not running, wrong address, firewall, timeout).
3. The `--key` flag value (and the default `~/.ssh/id_ed25519`) may begin with `~` or `~/`; sbctl supports tilde expansion for the current user (`~` and `~/`). Expansion of `~username` (other-user) is out of scope and treated as a literal path.

## Postconditions

1. sbctl reports: E-NET-001 "daemon unreachable: <address>: <reason>" on stderr.
2. sbctl exits with non-zero exit code (1).
3. sbctl does not produce any stdout output for the requested operation (partial success is not acceptable).
4. The error message is human-readable and operator-actionable (tells them what to check).

## Invariants

1. This contract applies to ALL sbctl operations — no subcommand bypasses connection error handling.
2. A timeout is treated as unreachable — sbctl does not hang indefinitely. The
   `dispatch()` function also sets a read deadline before decoding the RPC response,
   derived from `ctx.Deadline()` (or `RPCIdleTimeout`-equivalent 30 s as fallback if
   the context has no deadline). `sbctl` does not hang indefinitely on the RPC
   response phase. `dispatch()` accepts `context.Context` as its first parameter
   (go.md rule 7). The read deadline is cleared via `defer` after `dispatch` returns.
3. The connection timeout is configurable (implementation default: 5 seconds).
4. **`E-NET-001` is emitted for two cases (Ruling Y — reconciles with Inv-2):**
   (a) `net.Dial`/`net.DialContext` failure — daemon connection refused or DNS failure;
   (b) handshake read-deadline timeout — the daemon accepted the TCP connection but
   did not complete the ADR-012 challenge-response handshake within the timeout budget
   (treated as unreachable per Inv-2). A client that times out on the handshake is
   in the same operational position as one that cannot connect at all: the operator
   must check the daemon. The E-NET-001 message for case (b) is:
   `"daemon unreachable: <address>: connection timed out"` — the same address field
   as case (a) directs the operator's attention to the right target.
   Key-load failures (before dial) produce `E-CFG-010`; post-auth (post-AUTH_OK) RPC
   dispatch failures produce `E-RPC-001`. These failure modes MUST NOT share codes.

## Trigger

Network connection attempt to daemon fails or times out.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (FM-012) | Daemon running but not listening on configured port | Error: "connection refused: <addr>:<port>". |
| EC-002 | Daemon behind firewall | Error: "connection timed out after 5s: <addr>". |
| EC-003 | Wrong protocol (sbctl connecting to a non-Switchboard service) | Error: "unexpected response from daemon: not a Switchboard daemon". |
| EC-004 | Daemon address not specified and no default in config | Error: "no daemon address specified; use --target or set daemon.address in config". |
| EC-005 | Key file at `--key` path does not exist, is larger than 64 KiB, is malformed (not valid OpenSSH PEM), or contains a non-Ed25519 key type | sbctl prints `E-CFG-010 "key load failed: <path>: <reason>"` to stderr and exits 1. No connection attempt is made. No stdout output. |
| EC-006 | Authentication succeeded (AUTH_OK received) but the subsequent RPC request fails (server returns `"ok":false`, or response decode fails, or connection drops mid-RPC) | sbctl prints `E-RPC-001 "rpc failed: <command>: <reason>"` to stderr and exits 1. No stdout output. |
| EC-007 | `--key` value (or the default `~/.ssh/id_ed25519`) begins with `~` or `~/` | sbctl calls `os.UserHomeDir()` and substitutes the result for the leading `~` BEFORE opening the file. Expansion happens before any dial attempt. (a) If `os.UserHomeDir()` returns an error, sbctl prints `E-CFG-010 "key load failed: <original-path>: home directory unavailable: <reason>"` to stderr and exits 1; no connection attempt is made. (b) If expansion succeeds but the file cannot be read, sbctl prints `E-CFG-010 "key load failed: <expanded-path>: <reason>"` to stderr and exits 1; no connection attempt is made. The error message always uses the expanded path (not the raw `~`-prefixed value) so the operator can diagnose the actual filesystem path. `~username` (other-user) expansion is not performed; a path beginning with `~someuser/` is treated as a literal string and passes through to the OS. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl svtn list --target=localhost:9999` (nothing on 9999) | stderr: "daemon unreachable: localhost:9999: connection refused"; exit 1 | happy-path |
| `sbctl svtn list --target=10.0.0.1:9999` (firewall blocks) | stderr: "daemon unreachable: 10.0.0.1:9999: connection timed out after 5s"; exit 1 | edge-case |
| `sbctl svtn list` with no address configured | stderr: "no daemon address specified; use --target or set daemon.address in config"; exit 1 | error |
| `sbctl svtn list --target=someservice:9999` (wrong protocol) | stderr: "unexpected response from daemon: not a Switchboard daemon"; exit 1 | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-030 | sbctl always exits non-zero when daemon unreachable | unit |
| VP-030 | No stdout output on connection failure | unit |
| VP-030 | Error message includes attempted address | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 |
| L2 Domain Invariants | DI-002 (private keys never transit — error messages must not include key material) |
| Architecture Module | cmd/sbctl |
| Stories | S-6.03 (AC-004, AC-005, AC-007 — E-NET-001 connection error, no stdout on failure, connection timeout; AC-008 — tilde/home expansion of --key path, anchored to EC-007) |
| Capability Anchor Justification | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 — this BC specifies the error behavior for CLI unreachability, which is part of CAP-024's requirement that sbctl "exposes router status, SVTN management, key management, session operations" without misleading output |

## Related BCs

- BC-2.07.002 — depends on: this error handling is shared by all sbctl operations

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.5 | 2026-06-29 | ARCH-12 v1.5 Wave-5 Convergence Rulings V and Y: (V) Invariant 2 extended — `dispatch()` sets a read deadline before decoding the RPC response, derived from `ctx.Deadline()` (or 30 s fallback); `dispatch()` takes `context.Context` as first parameter (go.md rule 7); `sbctl` does not hang indefinitely on RPC response phase; ARCH-12 §Ruling F "residual concern" note is superseded. (Y) Invariant 4 replaced — `E-NET-001` is now explicitly emitted for two cases: (a) dial/connect failure; (b) handshake read-deadline timeout (treated as unreachable per Inv-2); the E-NET-001 case-(b) message format is `"daemon unreachable: <address>: connection timed out"`; reconciles prior Inv-2 vs Inv-4 conflict. |
| 1.4 | 2026-06-28 | Wave-5 mgmt-plane adversarial review: tilde/home expansion of sbctl --key path → EC-007 added; Precondition 3 added; Stories updated with S-6.03 AC-008. EC-007 specifies os.UserHomeDir() expansion before file open and before dial; both UserHomeDir error and subsequent file-read error map to E-CFG-010 with expanded path in message; ~username out-of-scope. |
| 1.3 | 2026-06-28 | Wave-5 mgmt-plane adversarial review Ruling 5 (ARCH-12 v1.2): Invariant 4 added (E-NET-001 strictly dial/connect-unreachable; E-CFG-010 for key-load failure; E-RPC-001 for post-auth dispatch failure; distinct codes must not share); EC-005 added (key file absent/oversized/malformed/wrong-type → E-CFG-010 "key load failed: \<path\>: \<reason\>", exit 1, no connection attempt); EC-006 added (post-AUTH_OK RPC dispatch failure → E-RPC-001 "rpc failed: \<command\>: \<reason\>", exit 1). |
| 1.2 | 2026-06-28 | Traceability refresh (Wave-5 consistency audit F-001/F-011): Stories field filled with S-6.03 (AC-004 E-NET-001, AC-005 no stdout, AC-007 timeout); `modified:` array populated (was erroneously empty at v1.1). E-CFG-002 collision flag added (F-003): taxonomy defines E-CFG-002 as "private key export not supported" (BC-2.05.007) but BC-2.09.003 v1.2 assigned E-CFG-002 to listen_addr validation — pre-existing inconsistency flagged for maintenance-pass resolution. |
| 1.1 | 2026-06-23 | Initial draft — sbctl reports clear connection error when daemon unreachable. |
