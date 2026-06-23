---
artifact_id: BC-2.02.008
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.02.008
subsystem: SS-TBD
capability: CAP-010
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
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-010]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.02.008: Router Split-Horizon Prevents Frames Being Forwarded Back Toward Arrival Interface

## Description

A router applies split-horizon forwarding: a frame received on interface A is never forwarded back out on interface A. This prevents simple two-node routing loops in multi-hop topologies. Split-horizon is applied at the interface level (the physical or logical connection a frame arrived on), not at the node level.

## Preconditions

1. The router has received a frame on a specific interface.
2. The router's forwarding table indicates the destination is reachable via the arrival interface (creating a potential loop).

## Postconditions

1. The frame is not forwarded back on the arrival interface, regardless of what the forwarding table says.
2. The frame is forwarded on all other eligible interfaces that the destination is reachable through.
3. If the only eligible interface is the arrival interface, the frame is dropped and an E-FWD-001 event is logged.
4. Split-horizon does not interact with the drop cache (BC-2.02.009) — both mechanisms operate independently.

## Invariants

1. Split-horizon applies to all SVTN frame types (data, empty-tick, parity, control).
2. The router does not maintain session state across interfaces — split-horizon is stateless per-frame.

## Trigger

Router's forwarding engine processes a frame for output.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Multi-hop ring topology: A→B→C→A | Split-horizon at B prevents forwarding back to A; at C prevents forwarding back to B. Frame reaches intended destination without looping. |
| EC-002 | New router added to topology; forwarding tables temporarily inconsistent | Split-horizon limits loop damage during convergence. Drop cache (BC-2.02.009) handles checksum-based loop detection as a second line of defense. |
| EC-003 | Split-horizon drops the only available path | Frame dropped; E-FWD-001 logged. This indicates a topology misconfiguration; the operator should inspect the forwarding table via `sbctl router routes`. |
| EC-004 | Router has two interfaces on the same physical network | Split-horizon applies per logical interface — if frames arrive on interface "eth0", they are not forwarded back on "eth0", even if "eth1" is on the same physical LAN. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Frame arrives on interface A; forwarding table says: forward to B, C, A | Frame forwarded to B and C only; A excluded | happy-path |
| Frame arrives on interface A; forwarding table says: forward to A only | Frame dropped; E-FWD-001 logged | edge-case |
| Frame arrives; destination unreachable from any non-arrival interface | Frame dropped silently | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | For all frames: output interface set excludes arrival interface | proptest |
| VP-TBD | Split-horizon does not affect frames forwarded to non-arrival interfaces | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-010 ("Router split-horizon and duplicate suppression") per capabilities.md §CAP-010 |
| L2 Domain Invariants | DI-004 (all traffic flows through routers — no direct node-to-node) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-010 ("Router split-horizon and duplicate suppression") per capabilities.md §CAP-010 — this BC specifies the split-horizon rule that CAP-010 defines as the primary loop prevention mechanism |

## Related BCs

- BC-2.02.009 — composes with: drop cache is the second-line loop prevention; split-horizon is the first
