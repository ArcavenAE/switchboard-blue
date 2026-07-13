---
artifact_id: BC-2.09.002
document_type: behavioral-contract
level: L3
version: "1.4"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "7aaa67b"
extracted_from: null
bc_id: BC-2.09.002
subsystem: deployment-operations
architecture_module: internal/drain
capability: CAP-027
priority: P2
criticality: medium
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - version: "1.4"
    date: 2026-07-12
    author: story-writer
    change: >
      Traceability Stories cell filled: S-7.04-FU-DRAIN-WIRE (PC-1..PC-4 SIGTERM/drain-broadcast
      path, historical backfill) + S-BL.CLI-SURFACE-COMPLETION (Trigger/PC-1 RPC-trigger
      governance addendum) — the distinct story-writer pass PO deferred. Governance-only; no
      PC/AC behavior change.
  - version: "1.3"
    date: 2026-07-12
    author: product-owner
    change: >
      S-BL.CLI-SURFACE-COMPLETION Ruling 4 (governance-only addendum, no PC/AC
      behavior change): Trigger gains a clarifying sentence — RPC-triggered
      drain via the `router.drain` wire verb causes the same shutdown sequence
      as SIGTERM (both reach the `shutdown:` label); the RPC connection is
      expected to be severed as the daemon exits, consistent with PC-3's
      best-effort-delivery framing. Resolves DRIFT-HS006-DRAIN-CLI-MISSING
      (drain half). [governance_leaf: true]
  - version: "1.2"
    date: 2026-07-11
    author: product-owner
    change: "PC-3 and PC-4 amended: acknowledgment is best-effort delivery (observer returns after dispatching DRAIN frame to node write path within drain window). No wire-level DRAIN-ACK opcode. Drain correctness proven by VP-037 observed-behavior property, not by protocol ACK. Refs: F-DW-SP1-006 adjudication."
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

When a router is about to shut down gracefully (SIGTERM received, `sbctl router drain` command), it sends a DRAIN frame to all connected nodes before disconnecting. Nodes receive the DRAIN frame, select alternate routers from their path set, and migrate their active sessions to the alternate paths. The drain coordinator waits (up to the drain window) for every observer to finish dispatching the DRAIN frame before the router disconnects; if the window expires, the router disconnects anyway (EC-003). Session migration correctness is verified by VP-037 observed-behavior, not by a wire-level ACK. This enables rolling updates without dropping active sessions.

## Preconditions

1. The router is running with at least one connected node.
2. Connected nodes have at least one alternate router path available (PE router phase — multi-homed).
3. The router receives SIGTERM or `sbctl router drain` command.

## Postconditions

1. Router sends DRAIN signal to all connected nodes.
2. Nodes receive DRAIN; select next-best router from their path ranking; migrate active sessions to the new path.
3. The router's drain coordinator dispatches the DRAIN frame to every connected node's write path (best-effort delivery). No wire-level acknowledgment opcode is required; acknowledgment is defined as the drain observer function returning after the DRAIN frame has been queued to the node's send channel, bounded by the drain window context. Session migration is verified by VP-037 observed-behavior (nodes reconnect to alternate router within 2 s) rather than by a protocol ACK byte.
4. After all observer functions have returned or the drain window has elapsed (EC-003): the drain coordinator signals completion; the router disconnects cleanly and exits with code 0.
5. Active sessions are maintained on the alternate paths; no session content is lost.

## Invariants

1. Drain signal is sent via the SVTN channel — it is authenticated and SVTN-scoped.
2. If a node has no alternate router, it cannot migrate. Those sessions are lost on router disconnect (unavoidable single-router dependency in E phase or if all other routers are also unavailable).
3. **DI-004**: Migration is node-to-router-to-node; nodes do not contact each other directly during migration.

## Trigger

Router receives SIGTERM or operator runs `sbctl router drain`. **RPC-trigger note (Ruling 4, S-BL.CLI-SURFACE-COMPLETION-rulings.md, 2026-07-12, governance-only):** RPC-triggered drain via the `router.drain` wire verb causes the same shutdown sequence as SIGTERM (both reach the `shutdown:` label); the RPC connection is expected to be severed as the daemon exits, consistent with PC-3's best-effort-delivery framing. See `S-BL.CLI-SURFACE-COMPLETION-rulings.md` Ruling 4.

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
| VP-037 | Sessions preserved on drain when alternate path available | integration/e2e |
| VP-037 | Drain timeout: router disconnects even without full acknowledgement | integration |
| VP-037 | New connections rejected during drain | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-027 ("Graceful router drain and session migration") per capabilities.md §CAP-027 |
| L2 Domain Invariants | DI-004 (all traffic through routers — migration routes through alternate routers) |
| Architecture Module | internal/drain |
| Stories | PC-1..PC-4 (SIGTERM/drain-broadcast path): S-7.04-FU-DRAIN-WIRE; Trigger/PC-1 (RPC-triggered drain via `router.drain`, governance addendum only): S-BL.CLI-SURFACE-COMPLETION |
| Capability Anchor Justification | CAP-027 ("Graceful router drain and session migration") per capabilities.md §CAP-027 — this BC is the direct behavioral specification of the "router signals impending shutdown; nodes migrate to alternate routers" mechanism CAP-027 defines |

## Related BCs

- BC-2.02.003 — depends on: alternate path must be ranked and available
- BC-2.09.001 — related to: PE graduation is required before drain migration makes sense (multi-path required)

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.4 | 2026-07-12 | story-writer | Traceability Stories cell filled: `S-7.04-FU-DRAIN-WIRE` (PC-1..PC-4 SIGTERM/drain-broadcast path, historical backfill) + `S-BL.CLI-SURFACE-COMPLETION` (Trigger/PC-1 RPC-trigger governance addendum) — the distinct story-writer pass PO deferred. Governance-only; no PC/AC behavior change. |
| 1.3 | 2026-07-12 | product-owner | S-BL.CLI-SURFACE-COMPLETION Ruling 4 (`S-BL.CLI-SURFACE-COMPLETION-rulings.md`): governance-only addendum — Trigger gains a clarifying sentence that RPC-triggered drain via the `router.drain` wire verb causes the same shutdown sequence as SIGTERM (both reach the `shutdown:` label); the RPC connection is expected to be severed as the daemon exits, consistent with PC-3's best-effort-delivery framing. No PC/AC behavior change. Resolves the drain half of `DRIFT-HS006-DRAIN-CLI-MISSING`. [governance_leaf: true — mirrors the POL-005/governance-leaf pattern, e.g. BC-2.07.001.md v1.13] |
| 1.2 | 2026-07-11 | product-owner | PC-3 and PC-4 amended: acknowledgment is best-effort delivery (observer returns after dispatching DRAIN frame to node write path within drain window). No wire-level DRAIN-ACK opcode. Drain correctness proven by VP-037 observed-behavior property, not by protocol ACK. Refs: F-DW-SP1-006 adjudication. |
| 1.1 | 2026-06-23 | product-owner | Initial draft — router sends drain signal before shutdown; nodes migrate to alternate routers. |
