---
artifact_id: BC-2.07.003
document_type: behavioral-contract
level: L3
version: "1.2"
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

## Postconditions

1. sbctl reports: E-NET-001 "daemon unreachable: <address>: <reason>" on stderr.
2. sbctl exits with non-zero exit code (1).
3. sbctl does not produce any stdout output for the requested operation (partial success is not acceptable).
4. The error message is human-readable and operator-actionable (tells them what to check).

## Invariants

1. This contract applies to ALL sbctl operations — no subcommand bypasses connection error handling.
2. A timeout is treated as unreachable — sbctl does not hang indefinitely.
3. The connection timeout is configurable (implementation default: 5 seconds).

## Trigger

Network connection attempt to daemon fails or times out.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (FM-012) | Daemon running but not listening on configured port | Error: "connection refused: <addr>:<port>". |
| EC-002 | Daemon behind firewall | Error: "connection timed out after 5s: <addr>". |
| EC-003 | Wrong protocol (sbctl connecting to a non-Switchboard service) | Error: "unexpected response from daemon: not a Switchboard daemon". |
| EC-004 | Daemon address not specified and no default in config | Error: "no daemon address specified; use --target or set daemon.address in config". |

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
| Stories | S-6.03 (AC-004, AC-005, AC-007 — E-NET-001 connection error, no stdout on failure, connection timeout) |
| Capability Anchor Justification | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 — this BC specifies the error behavior for CLI unreachability, which is part of CAP-024's requirement that sbctl "exposes router status, SVTN management, key management, session operations" without misleading output |

## Related BCs

- BC-2.07.002 — depends on: this error handling is shared by all sbctl operations

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.2 | 2026-06-28 | Traceability refresh (Wave-5 consistency audit F-001/F-011): Stories field filled with S-6.03 (AC-004 E-NET-001, AC-005 no stdout, AC-007 timeout); `modified:` array populated (was erroneously empty at v1.1). E-CFG-002 collision flag added (F-003): taxonomy defines E-CFG-002 as "private key export not supported" (BC-2.05.007) but BC-2.09.003 v1.2 assigned E-CFG-002 to listen_addr validation — pre-existing inconsistency flagged for maintenance-pass resolution. |
| 1.1 | 2026-06-23 | Initial draft — sbctl reports clear connection error when daemon unreachable. |
