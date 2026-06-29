---
artifact_id: BC-2.07.002
document_type: behavioral-contract
level: L3
version: "1.2"
status: draft
producer: product-owner
timestamp: 2026-06-28T00:00:00
phase: 1a
bc_id: BC-2.07.002
subsystem: network-management
architecture_module: cmd/sbctl
capability: CAP-024
priority: P2
criticality: high
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-06-28
    version: "1.2"
    change: >
      Wave-5 management plane cross-reference: added BC-2.07.004 to Related BCs
      as the server-side counterpart that completes the ADR-012 handshake. Updated
      Verification Properties table to add VP-067 (Authenticate() fail-closed, unit).
      PC-2 and PC-3 now have explicit server-side anchoring via BC-2.07.004;
      VP-049 e2e property anchors both client and server behaviors end-to-end.
      Story anchor updated: S-6.03 implements client-side (this BC + VP-067 + VP-030);
      S-W5.01 implements server-side (BC-2.07.004); S-W5.02 implements e2e (VP-049).
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-024]
kos_anchors:
  - elem-single-binary-three-modes
---

# Behavioral Contract BC-2.07.002: sbctl Unified CLI for All Four Daemon Types with OpenSSH Key Authentication

## Description

`sbctl` is the single operator CLI for all Switchboard daemon types: router (E and PE), access node, console, and control node. It authenticates the operator via their OpenSSH key (same key infrastructure used for SVTN admission). Subcommands are scoped by daemon type and operation category. No separate management tools are required.

## Preconditions

1. The operator has an OpenSSH key registered against the target daemon's SVTN or management scope.
2. The target daemon is running and listening on its management port.
3. sbctl can reach the daemon (network connectivity).

## Postconditions

1. sbctl connects to the target daemon.
2. sbctl authenticates the operator's OpenSSH key against the daemon's authorized key list.
3. If authenticated: the requested operation is executed; result returned in the configured output format (human-readable or JSON).
4. If not authenticated: E-ADM-010 "authentication failed"; exit code 1.
5. If daemon unreachable: E-NET-001 (per BC-2.07.003); exit code 1.

## Invariants

1. All sbctl subcommands are authenticated — there is no unauthenticated sbctl endpoint.
2. The sbctl binary is not a daemon; it exits after command completion.
3. Output format is consistent: `--json` for machine-readable output in all subcommands.

## Trigger

Operator runs any `sbctl <subcommand>` command.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Operator's key not authorized for the requested operation | E-ADM-010 "authentication failed" OR E-ADM-009 "insufficient authority for operation" depending on whether the key is recognized at all. |
| EC-002 | Multiple daemon types running on the same machine | sbctl targets by address and port; `--target=<addr>` or config file specifies which daemon. |
| EC-003 | sbctl run without any subcommand | Prints help text and exits 0. |
| EC-004 | sbctl version mismatch with daemon | Daemon returns version info; sbctl prints warning if version differs; command may still succeed if protocol is compatible. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl svtn list` with registered key | List of SVTNs returned | happy-path |
| `sbctl svtn list` with unregistered key | E-ADM-010 "authentication failed"; exit 1 | error |
| `sbctl --help` | Help text printed; exit 0 | happy-path |
| `sbctl router status --json` | JSON object with router status fields | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-049 | All subcommands require authentication — e2e across all four daemon types (router, access, console, control) | e2e |
| VP-049 | `--json` flag produces valid JSON for all subcommands | fuzz |
| VP-049 | sbctl exits after command completion (not a daemon) | unit |
| VP-067 | `Authenticate()` is fail-closed — returns nil only on verified AUTH_OK; all other outcomes (AUTH_FAIL, truncated stream, malformed message, connection error) return non-nil error | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 |
| L2 Domain Invariants | DI-002 (private keys never transit — sbctl uses key-based auth without transmitting the private key) |
| Architecture Module | cmd/sbctl |
| Stories | S-6.03 (client auth, Authenticate() fail-closed, connection error); S-W5.02 (e2e VP-049 across all four daemon types) |
| Capability Anchor Justification | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 — this BC specifies the unified CLI contract that CAP-024 defines as "single operator CLI for all four daemon types" with "OpenSSH key" authentication |

## Related BCs

- BC-2.07.003 — composes with: connection error handling is common to all sbctl operations
- BC-2.07.004 — composes with: server-side daemon auth counterpart (ADR-012); PC-2 (key auth) and PC-3 (execute if authenticated) of this BC require BC-2.07.004's server-side handshake enforcement to be meaningful end-to-end. VP-049 (e2e across all four daemon types) jointly verifies both BCs.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.2 | 2026-06-28 | Wave-5 management plane cross-reference: added BC-2.07.004 (server-side counterpart) to Related BCs. Verification Properties table extended with VP-067 (Authenticate() fail-closed, unit). Traceability Stories row updated: S-6.03 (client auth + VP-067 + VP-030), S-W5.02 (e2e VP-049 across all four daemon types). VP-049 description clarified to reflect e2e scope across all four daemon types (implementing story: S-W5.02). |
| 1.1 | 2026-06-23 | Initial published draft — sbctl unified CLI with OpenSSH key auth. |
