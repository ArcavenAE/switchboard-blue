---
artifact_id: BC-2.03.002
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.03.002
subsystem: session-discovery
architecture_module: internal/discovery
capability: CAP-012
priority: P1
criticality: important
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
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-012]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.03.002: Console Enumerates All SVTN Sessions Without Specifying Hostnames or IP Addresses

## Description

A console can list all available sessions across all access nodes on its SVTN by querying the discovery state — no hostnames, IP addresses, or manual configuration required. The console aggregates presence advertisements received from all access nodes and presents the session list. This is the core operator experience: `sbctl sessions list` returns all sessions across the fleet.

## Preconditions

1. The console is admitted to an SVTN (Tier 1 admission complete).
2. At least one access node on the SVTN has published at least one session.
3. The console has received presence advertisements from access nodes (via BC-2.03.001).

## Postconditions

1. `sbctl sessions list` (or equivalent API call) returns a list of all sessions currently known to the console from SVTN presence advertisements.
2. Each session entry includes: session name, access node address, attachment status, quality indicator.
3. Sessions are listed regardless of which access node they live on — the console has a unified view.
4. The list reflects the most recent state known (eventual consistency from heartbeat cycle).
5. Sessions no longer advertised (node gone, session closed) do not appear after the next heartbeat cycle.

## Invariants

1. **DI-005**: The session list contains only sessions from the console's SVTN — never sessions from other SVTNs.
2. Session names in the list are not necessarily unique across access nodes. The fully-qualified identifier is (access_node_addr, session_name).
3. The console does not contact access nodes directly to build the list — it uses only the broadcast advertisements.

## Trigger

Operator runs `sbctl sessions list` or console refreshes its session list view.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Two access nodes have sessions with the same name "agent-01" | Both appear in the list, differentiated by access node address. No collision error. |
| EC-002 | Console joins SVTN before any access nodes advertise | List is empty; no error. Console waits for next advertisement cycle. |
| EC-003 | Access node goes offline; its sessions disappear after next heartbeat window | Console's list shows stale sessions until next heartbeat (30s max staleness). Acceptable per FM-005. |
| EC-004 | Console requests on-demand refresh | Console sends presence request; all access nodes respond with current state; list updated immediately. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| SVTN has 2 access nodes: node-A (3 sessions), node-B (2 sessions) | `sbctl sessions list` returns 5 sessions with their node addresses | happy-path |
| Console has not received any advertisements yet | `sbctl sessions list` returns empty list with info message "no sessions discovered" | edge-case |
| Access node-B goes offline; 31 seconds pass | node-B's sessions no longer appear in list (heartbeat expired) | edge-case |
| Console requests refresh after node-B goes offline | Immediate empty response from node-B; node-B sessions removed | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Session list contains only sessions from current SVTN | integration |
| VP-TBD | After access node offline > 1 heartbeat interval, its sessions absent | integration |
| VP-TBD | Session list matches aggregate of all received advertisements | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-012 ("Console session enumeration across SVTN") per capabilities.md §CAP-012 |
| L2 Domain Invariants | DI-005 (SVTN cryptographic isolation) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-012 ("Console session enumeration across SVTN") per capabilities.md §CAP-012 — this BC specifies the console-side discovery that CAP-012 defines as "discovers all available sessions across all access nodes on its SVTN without specifying IP addresses" |

## Related BCs

- BC-2.03.001 — depends on: advertisements from access nodes are the data source
- BC-2.04.003 — composes with: session selection from this list feeds into attach flow
