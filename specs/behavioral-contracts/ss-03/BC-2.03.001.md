---
artifact_id: BC-2.03.001
document_type: behavioral-contract
level: L3
version: "1.4"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.03.001
subsystem: session-discovery
architecture_module: internal/discovery
capability: CAP-011
priority: P1
criticality: high
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
traces_to: [CAP-011]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.03.001: Access Node Advertises Session Presence via SVTN-Scoped Multicast on State Change and Periodic Heartbeat

## Description

An access node broadcasts its presence and the state of its published sessions to all nodes on the SVTN via an SVTN-scoped multicast address. Advertisements are triggered by: (1) session state changes (new session, session closed, attachment status change), (2) periodic heartbeat (configurable interval, default 30 seconds), and (3) on-demand when a console sends a presence request. This enables consoles to discover sessions without hostnames or IP addresses.

**Implementation scope note (Ruling W6TB-D):** BC-2.03.001 covers the advertisement trigger model and payload semantics. The wire transport (UDP multicast socket, SVTN-scoped multicast address allocation, admitted-node HMAC key derivation) is split to S-BL.DISCOVERY-WIRE. The implementing story S-7.02 delivers the in-process registry model; PC-1 (multicast to all admitted nodes), PC-3 (1-tick network delivery), and PC-4 (network heartbeat broadcast) are fully verified only after S-BL.DISCOVERY-WIRE ships.

## Preconditions

1. The access node is admitted to an SVTN (Tier 1 admission complete).
2. The access node has at least one published session.
3. A SVTN-scoped multicast address is allocated for the SVTN's discovery channel.

## Postconditions

1. The advertisement is multicast to all admitted nodes on the SVTN.
2. Each advertisement includes: access node address, list of session names, per-session attachment status, per-session quality indicator.
3. On state change (session added/removed/attached/detached): advertisement sent within 1 tick interval.
4. On periodic heartbeat: advertisement sent every 30 seconds regardless of state change. **Observability gate (Ruling W6TB-D):** in the registry model (S-7.02), the periodic heartbeat timer fires an observable side effect verifiable by injecting a tick and asserting a heartbeat counter increments. Network dispatch to wire is deferred to S-BL.DISCOVERY-WIRE.
5. Advertisement is authenticated (HMAC in outer header) so receivers can verify it is from an admitted node. **Key placeholder (Ruling W6TB-D / DRIFT-W6TBD-001):** in S-7.02, the HMAC key is `svtnID` (the SVTN identifier itself). This is a scoping placeholder — the SVTN ID is not admitted-node-scoped secret material. S-BL.DISCOVERY-WIRE must specify admitted-node HMAC key derivation before multicast deployment. The fail-closed rejection behavior (ErrInvalidHMACTag on wrong tag) is fully verified in S-7.02.

## Invariants

1. **DI-004**: Advertisements flow node-to-router-to-node via the SVTN; no direct node-to-node multicast.
2. **DI-005**: Advertisements are SVTN-scoped — nodes on SVTN-B do not receive advertisements from SVTN-A.
3. The access node does not advertise session content, only metadata (name, status, quality indicator).

## Trigger

Session state change; periodic heartbeat timer fires; console sends on-demand presence request.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (FM-005) | Heartbeat advertisement lost in transit | Next heartbeat (30s later) or next state change will refresh. Consoles may show stale data for one heartbeat interval — acceptable per FM-005. |
| EC-002 (DEC-014) | tmux session closes while advertisement is in flight | Access node sends a session-removed advertisement on the next state change event. |
| EC-003 | Access node loses SVTN connection briefly | Reconnects and sends a full-state advertisement on reconnection to resync all consoles. |
| EC-004 | Access node has 100 sessions | Single advertisement frame may need to be fragmented or multiple frames sent per heartbeat (implementation decision: max sessions per frame is architecture concern). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| New tmux session "agent-01" created | Advertisement multicast within 1 tick: {node_addr, sessions: [{name:"agent-01", attached:false, quality:green}]} | happy-path |
| Session "agent-01" attached by console | Advertisement multicast: {sessions: [{name:"agent-01", attached:true, quality:green}]} | happy-path |
| 30 seconds pass with no state change | Periodic heartbeat advertisement multicast | happy-path |
| Console sends presence request | Access node responds with current full-state advertisement | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-044 | Advertisement sent within 1 tick of state change | integration |
| VP-044 | Advertisement contains no session content — metadata only | code-audit |
| VP-044 | Advertisement is SVTN-scoped: received only by admitted nodes | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-011 ("Multicast presence advertisement") per capabilities.md §CAP-011 |
| L2 Domain Invariants | DI-004 (no direct node-to-node), DI-005 (SVTN cryptographic isolation) |
| Architecture Module | internal/discovery |
| Stories | S-7.02, S-BL.DISCOVERY-WIRE (deferred: PC-1/PC-3/PC-4 wire delivery) |
| Capability Anchor Justification | CAP-011 ("Multicast presence advertisement") per capabilities.md §CAP-011 — this BC specifies the advertisement trigger conditions and payload that CAP-011 defines as "state change, periodic heartbeat, and on-demand request" |

## Related BCs

- BC-2.03.002 — composes with: console discovery depends on these advertisements
- BC-2.03.003 — composes with: advertisement payload structure defined in BC-2.03.003

## Changelog

| Version | Date | Change |
|---------|------|--------|
| v1.4 | 2026-07-01 | Pass-2 L3 fix-burst (RULING-W6TB-D bidirectional-trace closure): Stories row updated to add S-BL.DISCOVERY-WIRE with deferred PC-1/PC-3/PC-4 wire delivery annotation. |
| v1.3 | 2026-07-01 | S-7.02 LENS-3 traceability backfill (RULING-W6TB-D): Traceability.Stories row filled with S-7.02. |
| v1.2 | 2026-07-01 | Ruling W6TB-D: scope split annotation added. PC-1 wire transport, PC-4 network dispatch, and admitted-node HMAC key vocabulary (DRIFT-W6TBD-001) deferred to S-BL.DISCOVERY-WIRE. Observability gate added to PC-4: heartbeat timer observable via injected counter. PC-5 key placeholder note added. |
| v1.1 | 2026-06-23 | Initial behavioral contract creation. |
