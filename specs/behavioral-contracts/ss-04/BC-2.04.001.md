---
artifact_id: BC-2.04.001
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.04.001
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
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-013]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.04.001: Access Node Connects to Local tmux via Control Mode and Publishes Sessions over SVTN

## Description

When the access node daemon starts, it connects to the local tmux server using control mode (`tmux -CC`) to enumerate and monitor sessions. Sessions discovered via control mode are published to the SVTN as available for attachment. The access node subscribes to tmux control mode events (`%output`, `%session-window-changed`, `%session-closed`, etc.) to maintain live session state.

## Preconditions

1. The access node daemon is running on a machine with tmux installed and in PATH.
2. tmux server is running (or startable by the access node).
3. The access node is admitted to an SVTN (or admission in progress).

## Postconditions

1. The access node has an active tmux control mode connection.
2. All current tmux sessions are enumerated and published to the SVTN.
3. New tmux sessions created after startup are automatically discovered and published.
4. Closed tmux sessions are automatically unpublished.
5. Session output events from control mode feed the downstream half-channel.

## Invariants

1. **DI-001**: The access node routes session content through the SVTN channels; it does not expose content to the router.
2. The access node connects to the local tmux server only — it never connects to remote tmux servers.
3. tmux session names are the canonical session identifiers used in advertisements and attach requests.

## Trigger

Access node daemon startup; tmux session lifecycle events (create, close).

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-013) | tmux control mode fails to initialize | Falls back to PTY proxy mode (BC-2.04.002); logs the fallback clearly. |
| EC-002 (FM-004) | tmux control mode connection drops mid-operation | Access node attempts to reconnect to control mode. If reconnect fails within timeout, falls back to PTY proxy mode. Sends "session unavailable" presence update. |
| EC-003 | tmux server has no sessions on startup | Access node starts successfully; session list is empty. No sessions published until sessions are created. |
| EC-004 (FM-011) | tmux not installed | Access node cannot start in control mode; falls back to PTY proxy mode with reduced functionality. Logs clearly: "tmux not found; using PTY fallback". |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Access node starts; tmux has 3 sessions: "agent-01", "agent-02", "build" | All 3 published to SVTN; presence advertisement sent | happy-path |
| New tmux session "agent-03" created after startup | "agent-03" published within 1 tick; presence advertisement updated | happy-path |
| tmux session "agent-02" closed | "agent-02" unpublished; presence advertisement updated (session-removed) | happy-path |
| tmux control mode connection drops | Reconnect attempt; presence update "session unavailable" | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-031 | All tmux sessions visible to control mode are published on startup | integration |
| VP-031 | Session create/close events propagate to SVTN within 1 tick | integration |
| VP-031 | Control mode disconnect triggers fallback or reconnect path | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation) |
| Architecture Module | internal/tmux |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 — this BC specifies the primary tmux control mode integration that CAP-013 defines as the source of all session traffic |

## Related BCs

- BC-2.04.002 — composes with: PTY fallback when this BC's control mode path fails
- BC-2.03.001 — composes with: session publish triggers advertisement
