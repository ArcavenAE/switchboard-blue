---
artifact_id: BC-2.04.002
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.04.002
subsystem: session-access
architecture_module: internal/tmux
capability: CAP-013
priority: P0
criticality: critical
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
  - '.factory/specs/domain-spec/edge-cases.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '.factory/specs/domain-spec/assumptions.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-013]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.04.002: Access Node Falls Back to PTY Proxy When tmux Control Mode Unavailable

## Description

When tmux control mode is unavailable (tmux absent, version incompatible, or control mode fails), the access node falls back to PTY proxy mode. In PTY proxy mode, the access node opens a PTY and proxies its I/O as a single anonymous session. Session metadata is reduced: no structured session names (sessions named by PTY number), no content-type detection, quality signals derived from byte-rate heuristics only. The fallback is logged and the operator is notified.

## Preconditions

1. tmux control mode initialization failed (BC-2.04.001 could not establish control mode).
2. A PTY device is available on the host.

## Postconditions

1. Access node enters PTY proxy mode.
2. The access node publishes the PTY session to the SVTN with a synthetic name (implementation: "pty-<pid>" or similar).
3. A log entry is written: "tmux control mode unavailable; using PTY proxy mode. Functionality limited: no structured session metadata, no content-type detection."
4. The PTY session is accessible for attach by a console.
5. Session quality indicator is derived from byte rate heuristics (no tmux event data available).

## Invariants

1. **DI-001**: PTY proxy mode provides the same carrier-grade content separation as control mode — the access node still routes through the SVTN.
2. PTY proxy mode is a degraded-functionality state, not a failure. The session is still usable.
3. The fallback state is clearly communicated to the operator — never silent.

## Trigger

tmux control mode initialization failure detected at access node startup or mid-operation.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-013, ASM-003) | tmux binary exists but control mode flag not supported (old version) | PTY fallback; log: "tmux version does not support -CC flag". |
| EC-002 (FM-011) | tmux not found in PATH | PTY fallback immediately; no retry for tmux. Log: "tmux binary not found; using PTY proxy". |
| EC-003 | tmux control mode drops after successful start (mid-operation) | Access node attempts control mode reconnect; if reconnect fails after 3 attempts, switches to PTY proxy mode for existing sessions. Log: "tmux control mode lost; falling back to PTY proxy". |
| EC-004 | PTY device not available on host | Access node fails to start entirely; E-SYS-001 "PTY device unavailable; cannot start access node". No silent failure. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| tmux not found; PTY available | Access node starts in PTY proxy mode; log: "tmux not found; using PTY proxy" | happy-path |
| tmux -CC returns error | Access node starts in PTY proxy mode; log: "tmux control mode error: <error text>" | happy-path |
| PTY session "pty-12345" active | Session published to SVTN as "pty-12345"; console can attach | happy-path |
| PTY device not available | Startup error E-SYS-001; access node exits with non-zero | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Fallback to PTY mode never produces silent failure | unit |
| VP-TBD | PTY proxy session is accessible to console attach | integration |
| VP-TBD | Log entry is written on every fallback event | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — PTY proxy maintains the separation) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 — this BC specifies the "PTY fallback used when control mode is unavailable" path defined within CAP-013 |

## Related BCs

- BC-2.04.001 — depends on: control mode failure triggers this BC
