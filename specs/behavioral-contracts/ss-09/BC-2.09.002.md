---
artifact_id: BC-2.09.002
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.09.002
subsystem: deployment-operations
architecture_module: internal/drain
capability: CAP-027
priority: P2
criticality: supportive
scope_phase: PE
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
traces_to: [CAP-027]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.09.002: Router Sends Drain Signal Before Shutdown; Nodes Migrate to Alternate Routers

## Description

When a router is about to shut down gracefully (SIGTERM received, `sbctl router drain` command), it sends a drain signal to all connected nodes before disconnecting. Nodes receive the drain signal, select alternate routers from their path set, migrate their sessions to the alternate paths, and then acknowledge the drain. The router waits for acknowledgements (up to a timeout) before disconnecting. This enables rolling updates without dropping active sessions.

## Preconditions

1. The router is running with at least one connected node.
2. Connected nodes have at least one alternate router path available (PE router phase — multi-homed).
3. The router receives SIGTERM or `sbctl router drain` command.

## Postconditions

1. Router sends DRAIN signal to all connected nodes.
2. Nodes receive DRAIN; select next-best router from their path ranking; migrate active sessions to the new path.
3. Nodes acknowledge DRAIN signal to the router.
4. After all acknowledgements (or timeout): router disconnects cleanly; exits with code 0.
5. Active sessions are maintained on the alternate paths; no session content is lost.

## Invariants

1. Drain signal is sent via the SVTN channel — it is authenticated and SVTN-scoped.
2. If a node has no alternate router, it cannot migrate. Those sessions are lost on router disconnect (unavoidable single-router dependency in E phase or if all other routers are also unavailable).
3. **DI-004**: Migration is node-to-router-to-node; nodes do not contact each other directly during migration.

## Trigger

Router receives SIGTERM or operator runs `sbctl router drain`.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (FM-009) | Router crashes (SIGSEGV, OOM) without sending drain | No drain signal; nodes detect failure via missed keep-alives; multi-homed nodes failover automatically within keep-alive timeout. Single-homed nodes (E phase) lose sessions. |
| EC-002 | Node has no alternate router | Node cannot migrate. Router logs: "node <addr> cannot migrate: no alternate path". Session lost on disconnect. |
| EC-003 | Drain timeout exceeded (nodes not all acknowledged) | Router disconnects after timeout; logs remaining unacknowledged nodes. |
| EC-004 | New connection attempt to draining router | Router rejects new connections with E-NET-006 "router draining; connect to alternate router". |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Router drains; all nodes have alternate paths | All sessions migrate; router exits cleanly; no session loss | happy-path |
| Router drains; one node has no alternate | That node's sessions lost; others migrate; router exits | edge-case |
| Router crashes (no SIGTERM) | Nodes detect failure via keep-alive timeout; multi-homed nodes failover; E-phase nodes lose sessions | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Sessions preserved on drain when alternate path available | integration/e2e |
| VP-TBD | Drain timeout: router disconnects even without full acknowledgement | integration |
| VP-TBD | New connections rejected during drain | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-027 ("Graceful router drain and session migration") per capabilities.md §CAP-027 |
| L2 Domain Invariants | DI-004 (all traffic through routers — migration routes through alternate routers) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-027 ("Graceful router drain and session migration") per capabilities.md §CAP-027 — this BC is the direct behavioral specification of the "router signals impending shutdown; nodes migrate to alternate routers" mechanism CAP-027 defines |

## Related BCs

- BC-2.02.003 — depends on: alternate path must be ranked and available
- BC-2.09.001 — related to: PE graduation is required before drain migration makes sense (multi-path required)
