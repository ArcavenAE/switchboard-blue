---
artifact_id: BC-2.09.003
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.09.003
subsystem: deployment-operations
architecture_module: internal/config
capability: CAP-023
priority: P0
criticality: important
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified: []
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
traces_to: [CAP-023, CAP-024]
kos_anchors:
  - elem-single-binary-three-modes
---

# Behavioral Contract BC-2.09.003: Router Startup Fails Cleanly on Malformed Config with Actionable Error Message

## Description

When the router daemon starts with a malformed, incomplete, or invalid configuration file, it exits immediately with a non-zero exit code and prints a clear, actionable error message identifying the specific problem (field name, line number, value). The daemon does not start in a partially-configured state. No sessions are affected (the daemon was not running).

## Preconditions

1. The router daemon process is starting.
2. The configuration file exists but contains an error.

## Postconditions

1. The daemon exits with a non-zero exit code before accepting any connections.
2. stderr contains the error message: E-CFG-001 format: "config error: <field>: <problem>. Fix: <suggestion>".
3. stdout is empty.
4. No leftover state, lock files, or partial network bindings.

## Invariants

1. No daemon starts in a degraded-config state — it's all-or-nothing.
2. Error messages name the specific field and provide a fix suggestion.
3. This applies equally to initial startup and config reload (SIGHUP): a bad config reload leaves the daemon running on the previous config.

## Trigger

Daemon startup config parsing failure; config reload with invalid config.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Config file missing entirely | E-CFG-004 "config file not found: <path>"; exit 1. |
| EC-002 | Config file present but empty | E-CFG-001 "config error: required field 'listen_addr' missing"; exit 1. |
| EC-003 (FM-010) | Malformed YAML (syntax error) | E-CFG-005 "config parse error: invalid YAML at line N: <detail>"; exit 1. |
| EC-004 | Config reload (SIGHUP) with bad new config | Daemon logs: "config reload failed: <error>; continuing with previous config". Previous config remains active. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Missing required field `listen_addr` | E-CFG-001 "config error: listen_addr: required field missing. Fix: add 'listen_addr: <ip>:<port>' to config"; exit 1 | happy-path |
| Invalid YAML syntax | E-CFG-005 "config parse error: invalid YAML at line 5: unexpected token"; exit 1 | error |
| Config file not found | E-CFG-004 "config file not found: /etc/switchboard/router.yaml"; exit 1 | error |
| Config reload with bad config | Daemon logs "config reload failed"; continues on previous config; exits 0 (daemon still running) | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Startup with any config error always exits non-zero | unit |
| VP-TBD | Error message includes field name and fix suggestion | unit |
| VP-TBD | Config reload failure leaves daemon on previous config | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023; CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 |
| L2 Domain Invariants | DI-007 (outer header format stability — config errors at startup prevent mismatched protocol state) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-023 ("SVTN lifecycle management (create, destroy)") per capabilities.md §CAP-023 — router startup is the prerequisite for SVTN creation; also CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 because sbctl-launched daemon operations depend on clean startup behavior |

## Related BCs

- BC-2.09.001 — related to: config reload uses the same validation as startup
