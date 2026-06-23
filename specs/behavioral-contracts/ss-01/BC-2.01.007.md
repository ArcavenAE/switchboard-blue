---
artifact_id: BC-2.01.007
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.01.007
subsystem: session-networking
architecture_module: internal/admission
capability: CAP-004
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
traces_to: [CAP-004]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.01.007: Session Continuity Survives IP Address Change via Cryptographic Re-Authentication

## Description

When a node's IP address changes (wifi-to-LAN handoff, DHCP renewal, mobile roaming), the active session is not terminated. The node detects its new IP, re-authenticates to the router using its cryptographic identity (same keypair, same derived node address), and resumes session traffic. The router recognizes the node by its cryptographic identity, not its source IP. Frames in-flight at the moment of IP change may be lost; the half-channel recovery mechanisms handle the gap.

## Preconditions

1. A node has an active session with a router.
2. The node's IP address changes (new source IP detected).
3. The node's SVTN admission keypair is unchanged.
4. The router is reachable at its configured address from the new IP.

## Postconditions

1. The node initiates a re-authentication challenge to the router from its new IP.
2. The router verifies the challenge signature against the admitted key set.
3. The router updates its routing entry for the node to reflect the new source IP.
4. Session traffic resumes on the existing channel; no new channel establishment required.
5. Frames lost during the transition are recovered by the half-channel recovery mechanisms (upstream: idempotent replay; downstream: ARQ).
6. Re-authentication completes within the configured timeout (implementation: ≤ 5 seconds).
7. The console's quality indicator may show yellow during the transition; returns to green on recovery.

## Invariants

1. **DI-004**: The node re-authenticates through the router — no direct node-to-node reconnection path.
2. **DI-002**: The re-authentication challenge uses a signed nonce; the private key never transits the network.
3. Session identity is the channel ID + cryptographic node address pair, not the IP 4-tuple.

## Trigger

Node detects IP address change (OS network interface event or failed keep-alive probe from old IP).

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-001) | IP changes while upstream frames are in-flight | Lost frames recovered by idempotent replay window. Receiver deduplicates if old+new frames both arrive. |
| EC-002 | IP changes and router is temporarily unreachable at new IP | Node retries re-authentication up to configured max attempts. Sessions frozen (not closed) during retry window. If router unreachable after timeout: E-NET-003 (router unreachable after IP change). |
| EC-003 | Node changes IP twice in rapid succession | Each re-authentication uses the latest IP. Previous re-auth in-progress is superseded. Router updates to final IP. |
| EC-004 (DEC-002) | E router phase: single router fails at moment of IP change | Session lost. No failover path available in E router phase. User must restart. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Node IP changes from 192.168.1.10 to 10.0.0.5; same keypair | Re-auth succeeds; session resumes; router routing entry updated | happy-path |
| Node IP changes; router offline for 2s then recovers | Session frozen during 2s; resumes on router recovery; gap covered by ARQ | edge-case |
| Node IP changes; re-auth challenge signature invalid (wrong key) | Router rejects with E-ADM-001; session terminated | error |
| Node IP changes; router has revoked the node's key | Router rejects with E-ADM-005; session terminated; node receives explicit rejection | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-036 | Session channel ID unchanged before and after IP change | integration |
| VP-036 | Re-authentication completes within 5s on LAN | e2e benchmark |
| VP-036 | Private key not present in re-authentication wire messages | property/audit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-004 ("Session continuity across network transitions") per capabilities.md §CAP-004 |
| L2 Domain Invariants | DI-004 (no direct node-to-node communication), DI-002 (private keys never transit) |
| Architecture Module | internal/admission |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-004 ("Session continuity across network transitions") per capabilities.md §CAP-004 — this BC is the direct behavioral realization of "nodes maintain sessions when the underlying IP address changes" |

## Related BCs

- BC-2.01.006 — depends on: cryptographic node address is the identity preserved across IP change
- BC-2.05.001 — depends on: re-authentication uses Tier 1 admission mechanism
- BC-2.06.001 — related to: quality indicator reflects transition degradation
