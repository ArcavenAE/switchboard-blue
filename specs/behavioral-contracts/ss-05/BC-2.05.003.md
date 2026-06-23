---
artifact_id: BC-2.05.003
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.05.003
subsystem: admission-security
architecture_module: internal/session
capability: CAP-018
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
traces_to: [CAP-018]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.05.003: Per-Session Tier 2 Authorization Enforced by Access Node, Not Router

## Description

After a console is admitted to the SVTN (Tier 1), it must also be authorized for each specific session it wants to attach to (Tier 2). The access node maintains an authorized key list per session. Before forwarding a console's upstream traffic (keystrokes) to tmux, the access node verifies the console's public key against the session's authorization list. The router does not perform Tier 2 checks — it forwards all admitted-node traffic.

## Preconditions

1. The console is Tier 1 admitted to the SVTN.
2. The access node has a per-session authorization list for the target session.
3. The console is requesting to attach to the session (BC-2.04.003).

## Postconditions

1. If console's public key is in the session authorization list: attach proceeds; upstream forwarding enabled.
2. If console's public key is NOT in the session authorization list: access node rejects the attach request with E-ADM-006 "session authorization denied"; the channel is not established.
3. The router is not consulted for Tier 2 checks; the router has no knowledge of per-session authorization.
4. A console authorized for session A on access node X is not automatically authorized for session B on the same access node (authorization is per-session).

## Invariants

1. **DI-010**: Session authorization is access-node-enforced. This invariant is provable: the router has no data structure for per-session authorization lists.
2. **DI-011**: Tier 1 and Tier 2 keys may be the same keypair, but the authorization scopes are independent.
3. The authorization list is stored on the access node, not on the router or control node.

## Trigger

Console attach request arrives at the access node after SVTN frame routing.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-006) | Console admitted to SVTN but not in session auth list | Attach rejected; E-ADM-006. Console can still list sessions; cannot attach. |
| EC-002 | Session authorization list is empty (no consoles authorized) | All attach requests rejected. Operator must add authorized keys before any console can attach. |
| EC-003 | Console authorized for session "agent-01" requests to attach to "agent-02" | Check is per-session. Rejected unless "agent-02" also has the console's key authorized. |
| EC-004 | Access node authorization list changes mid-session (key added or revoked) | Existing sessions continue until next re-authorization event. New authorization applies to new attach requests immediately. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Console key in session auth list with mode=full | Attach succeeds; upstream enabled | happy-path |
| Console key in session auth list with mode=read-only | Attach succeeds; upstream rejected (BC-2.04.005) | happy-path |
| Console key not in session auth list | E-ADM-006 "session authorization denied"; channel not established | error |
| Empty session auth list | E-ADM-006 for all attach requests | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-012 | Router code has no per-session authorization data structure | code-audit |
| VP-012 | Tier 2 check is performed before upstream channel is opened | integration |
| VP-012 | Tier 2 authorization is per-session: different sessions require separate authorization | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-018 ("Per-session access authorization (Tier 2)") per capabilities.md §CAP-018 |
| L2 Domain Invariants | DI-010 (session authorization is access-node-enforced), DI-011 (role separation between Tier 1 and Tier 2 keys) |
| Architecture Module | internal/session |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-018 ("Per-session access authorization (Tier 2)") per capabilities.md §CAP-018 — this BC specifies the enforcement mechanism that CAP-018 defines as "authorized console key list per session" checked "before forwarding a console's upstream" |

## Related BCs

- BC-2.05.001 — related to: Tier 1 is prerequisite; Tier 2 is this BC
- BC-2.04.003 — depends on: attach flow triggers this check
- BC-2.04.005 — composes with: read-only mode is a Tier 2 scope designation
