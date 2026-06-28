---
artifact_id: BC-2.09.003
document_type: behavioral-contract
level: L3
version: "1.2"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.09.003
subsystem: SS-09
architecture_module: internal/config
capability: CAP-028
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
      S-6.01 scope expansion: (a) deep field validation postconditions added
      (PC-5 through PC-8) — listen_addr host:port parse, upstream_routers
      host:port parse, DrainTimeout/KeepaliveInterval positive-value
      enforcement; new error codes E-CFG-002, E-CFG-003, E-CFG-006, E-CFG-007;
      (b) config-application postcondition added (PC-9) — daemon MUST use
      the validated config struct to configure subsystems, not discarded values
      or hardcoded constants. Edge cases EC-005 through EC-009 added.
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
traces_to: [CAP-028]
kos_anchors:
  - elem-single-binary-three-modes
---

# Behavioral Contract BC-2.09.003: Router Startup Fails Cleanly on Malformed Config with Actionable Error Message; Validated Config Is Applied to the Daemon

## Description

When the router daemon starts with a malformed, incomplete, or invalid configuration file, it exits immediately with a non-zero exit code and prints a clear, actionable error message identifying the specific problem (field name, line number, value). The daemon does not start in a partially-configured state. No sessions are affected (the daemon was not running). When startup succeeds, the daemon MUST use the validated config struct to configure all subsystems — it MUST NOT fall back to hardcoded defaults or discard the validated config.

## Preconditions

1. The router daemon process is starting.
2. A `--config <path>` flag (or equivalent) has been supplied.
3. The configuration file exists.

## Postconditions

### Failure path postconditions (any validation error)

1. The daemon exits with a non-zero exit code before accepting any connections.
2. stderr contains at least one error message in E-CFG-001 format: `"config error: <field>: <problem>. Fix: <suggestion>"`.
3. stdout is empty.
4. No leftover state, lock files, or partial network bindings.

### Deep field validation postconditions (v1.2 additions)

5. `listen_addr` is parsed as a valid `host:port` (net.ResolveTCPAddr or equivalent); if invalid, exits with E-CFG-002: `"config error: listen_addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '0.0.0.0:9090'"`.
6. Each entry in `upstream_routers[].addr` is parsed as a valid `host:port`; if any entry is invalid, exits with E-CFG-003: `"config error: upstream_routers[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '10.0.0.1:9090'"`.
7. `drain_timeout` (if present) must be a positive duration (> 0); if zero or negative, exits with E-CFG-006: `"config error: drain_timeout: must be > 0; got '<value>'. Fix: set to a positive duration, e.g. '10s'"`.
8. `keepalive_interval` (if present) must be a positive duration (> 0); if zero or negative, exits with E-CFG-007: `"config error: keepalive_interval: must be > 0; got '<value>'. Fix: set to a positive duration, e.g. '1s'"`.

### Config application postcondition (v1.2 addition)

9. When `--config` is supplied and validation passes, the daemon initializes all subsystems using the validated config struct. Specifically: it binds the listen socket on the address from `listen_addr` (not a hardcoded default); it sets the tick interval for half-channels from `tick_interval`; it configures `drain_timeout` for the drain subsystem; it configures `keepalive_interval` for keepalive probes. The daemon MUST NOT silently ignore a supplied config and fall back to hardcoded constants.

## Invariants

1. No daemon starts in a degraded-config state — it's all-or-nothing.
2. Error messages name the specific field (and index for array fields) and provide a fix suggestion.
3. This applies equally to initial startup and config reload (SIGHUP): a bad config reload leaves the daemon running on the previous config.
4. All validation errors are collected and reported together (exhaustive reporting), not just the first.
5. The validated config is the single source of truth for subsystem configuration; hardcoded fallback values for configurable fields are prohibited when a config file is supplied.

## Trigger

Daemon startup config parsing failure; config reload with invalid config.

## Error Codes

| Code | Condition | Severity | Exit Code | Message Template |
|------|-----------|----------|-----------|-----------------|
| E-CFG-001 | Required field missing or generic validation failure | broken | 1 | `"config error: <field>: <problem>. Fix: <suggestion>"` |
| E-CFG-002 | `listen_addr` is not a valid `host:port` | broken | 1 | `"config error: listen_addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '0.0.0.0:9090'"` |
| E-CFG-003 | `upstream_routers[N].addr` is not a valid `host:port` | broken | 1 | `"config error: upstream_routers[<N>].addr: '<value>' is not a valid host:port. Fix: use '<ip>:<port>' format, e.g. '10.0.0.1:9090'"` |
| E-CFG-004 | Config file not found at the supplied path | broken | 1 | `"config file not found: <path>"` |
| E-CFG-005 | Config file present but malformed YAML (syntax error) | broken | 1 | `"config parse error: invalid YAML at line <N>: <detail>"` |
| E-CFG-006 | `drain_timeout` is zero or negative | broken | 1 | `"config error: drain_timeout: must be > 0; got '<value>'. Fix: set to a positive duration, e.g. '10s'"` |
| E-CFG-007 | `keepalive_interval` is zero or negative | broken | 1 | `"config error: keepalive_interval: must be > 0; got '<value>'. Fix: set to a positive duration, e.g. '1s'"` |

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Config file missing entirely | E-CFG-004 "config file not found: <path>"; exit 1. |
| EC-002 | Config file present but empty | E-CFG-001 "config error: required field 'listen_addr' missing"; exit 1. |
| EC-003 (FM-010) | Malformed YAML (syntax error) | E-CFG-005 "config parse error: invalid YAML at line N: <detail>"; exit 1. |
| EC-004 | Config reload (SIGHUP) with bad new config | Daemon logs: "config reload failed: <error>; continuing with previous config". Previous config remains active. |
| EC-005 | `listen_addr` present but missing port (e.g. `"0.0.0.0"`) | E-CFG-002 with value `"0.0.0.0"`; exit 1. |
| EC-006 | `listen_addr` with non-numeric port (e.g. `"0.0.0.0:notaport"`) | E-CFG-002 with value `"0.0.0.0:notaport"`; exit 1. |
| EC-007 | `upstream_routers` has two entries; first is valid, second is invalid | E-CFG-003 naming index 1 (0-based); all errors collected before exit 1 (exhaustive reporting). |
| EC-008 | `drain_timeout: 0s` | E-CFG-006 with value `"0s"`; exit 1. |
| EC-009 | `keepalive_interval: -1s` | E-CFG-007 with value `"-1s"`; exit 1. |
| EC-010 | Config file supplied and valid; daemon starts | PC-9: listen socket bound on `listen_addr` from config (not `":9090"` or any hardcoded address); tick interval, drain_timeout, keepalive_interval all sourced from config struct. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Missing required field `listen_addr` | E-CFG-001 "config error: listen_addr: required field missing. Fix: add 'listen_addr: <ip>:<port>' to config"; exit 1 | happy-path |
| `listen_addr: "0.0.0.0"` (no port) | E-CFG-002 "config error: listen_addr: '0.0.0.0' is not a valid host:port..."; exit 1 | error |
| `upstream_routers: [{addr: "notvalid"}]` | E-CFG-003 "config error: upstream_routers[0].addr: 'notvalid' is not a valid host:port..."; exit 1 | error |
| `drain_timeout: 0s` | E-CFG-006 "config error: drain_timeout: must be > 0; got '0s'..."; exit 1 | error |
| `keepalive_interval: -1s` | E-CFG-007 "config error: keepalive_interval: must be > 0; got '-1s'..."; exit 1 | error |
| Invalid YAML syntax | E-CFG-005 "config parse error: invalid YAML at line 5: unexpected token"; exit 1 | error |
| Config file not found | E-CFG-004 "config file not found: /etc/switchboard/router.yaml"; exit 1 | error |
| Config reload with bad config | Daemon logs "config reload failed"; continues on previous config; exits 0 (daemon still running) | edge-case |
| Valid config supplied | Daemon binds `listen_addr` from config, not a hardcoded default; tick_interval, drain_timeout, keepalive_interval sourced from config | happy-path (PC-9) |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-028, VP-029 | Startup with any config error always exits non-zero | unit |
| VP-028, VP-029 | Error message includes field name and fix suggestion | unit |
| VP-028, VP-029 | listen_addr host:port parse enforced at validation | unit |
| VP-028, VP-029 | upstream_routers[N].addr host:port parse enforced | unit |
| VP-028, VP-029 | drain_timeout and keepalive_interval positive-value enforcement | unit |
| VP-028, VP-029 | Config reload failure leaves daemon on previous config | integration |
| VP-028, VP-029 | Validated config applied to daemon subsystems (not hardcoded defaults) | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-028 ("Daemon startup config validation") per capabilities.md §CAP-028 |
| L2 Domain Invariants | (none directly; anchored to FM-010 via capability CAP-028) |
| Architecture Module | internal/config |
| Stories | S-6.01 (AC-001, AC-002, AC-003, AC-004, AC-005, AC-006) |
| Capability Anchor Justification | CAP-028 ("Daemon startup config validation") per capabilities.md §CAP-028 — this BC directly realizes the guarantee that a daemon exits non-zero with an actionable error message before accepting any connections, which is exactly the scope of CAP-028. The config-application postcondition (PC-9) is a necessary corollary: validation is meaningless if the validated config is then discarded. Anchored to FM-010 (deployment misconfig). |

## Related BCs

- BC-2.09.001 — related to: config errors discovered on reload (including upstream_routers address validation) use the same E-CFG-* error mechanism
- BC-2.04.007 — parallel: access node daemon startup/shutdown lifecycle (same class of lifecycle contract, different daemon, different subsystem); BC-2.04.007 does not own config validation

## Architecture Anchors

- ARCH-06-deployment-and-ops.md §Config File Validation (BC-2.09.003, NFR-011) — binding sequence (loadConfigFile → Validate → bindListenSocket) is authoritative
- ARCH-INDEX.md §SS-09 (deployment-operations, internal/config)

## Story Anchor

S-6.01 — AC-001 through AC-006 trace to postconditions in this BC.

## VP Anchors

VP-028, VP-029 (existing; cover all postconditions including v1.2 additions).

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.2 | 2026-06-28 | S-6.01 scope expansion to cover (a) deep field validation and (b) config application. Added PC-5 through PC-9; added E-CFG-002, E-CFG-003, E-CFG-006, E-CFG-007 error codes; added EC-005 through EC-010; updated title H1 to reflect both behaviors; added Inv-4 and Inv-5. Fixed `subsystem:` frontmatter to use SS-09 (ARCH-INDEX Subsystem Registry). |
| 1.1 | 2026-06-23 | Initial draft — router startup fails cleanly on malformed config. |
