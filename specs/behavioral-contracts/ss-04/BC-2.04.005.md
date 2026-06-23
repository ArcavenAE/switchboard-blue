---
artifact_id: BC-2.04.005
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.04.005
subsystem: SS-TBD
capability: CAP-015
priority: P1
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
traces_to: [CAP-015]
kos_anchors:
  - elem-ssh-end-to-end-encryption
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.04.005: Read-Only Console Receives Downstream Stream; Upstream Keystrokes Are Rejected by Access Node

## Description

A console holding a read-only session authorization key may subscribe to a session's downstream output stream but cannot send keystrokes. The downstream stream is delivered identically to a full-access console. Any upstream keystroke frames sent by the read-only console are rejected by the access node. The rejection is explicit: the access node returns an error to the console, not a silent drop.

## Preconditions

1. The console is admitted to the SVTN (Tier 1 admission complete).
2. The console's session authorization key for this session is registered as read-only on the access node.
3. The named session exists and is published.

## Postconditions

1. The console's subscription to the downstream stream is established.
2. The console receives all downstream frames for the session.
3. Any upstream keystroke frame from this console is rejected by the access node with E-ADM-007 "upstream rejected: read-only access".
4. The rejection does not terminate the console's downstream subscription.
5. The access node's presence advertisement does not change attached=true for a read-only subscriber (read-only subscribers are not "attached" in the full-access sense).

## Invariants

1. **DI-010**: Access node enforces read-only restriction. The router cannot distinguish read-only from full-access consoles — the restriction is enforced at the session layer.
2. **DI-011**: Read-only is a Tier 2 scope designation, not a Tier 1 admission property.
3. The downstream stream for a read-only console is identical to what a full-access console would receive.

## Trigger

Console attaches with a read-only session authorization key; console sends a keystroke while in read-only mode.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-011) | Full-access console and read-only console both attached | Full-access console's keystrokes are forwarded. Read-only console's keystrokes are rejected. Both receive the same downstream output. |
| EC-002 | Read-only console attempts to detach using a control keystroke sequence | The keystroke (as payload data) is rejected; the read-only console must use `sbctl sessions detach` to detach cleanly. |
| EC-003 | Scope is per-SVTN (all sessions read-only for this key) | Access node enforces read-only on all sessions this console attaches to on this SVTN. Not just one session. |
| EC-004 | Read-only console sends empty-tick frame (not a keystroke) | Empty-tick frames are part of the half-channel clock and are sent by both upstream and downstream. The access node accepts empty-tick frames from read-only consoles (they are liveness probes, not keystrokes). Only payload-bearing upstream frames are rejected. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Read-only console subscribes to "agent-01" | Downstream stream received; presence unchanged (attached=false per read-only) | happy-path |
| Read-only console sends keystroke 'a' | E-ADM-007 "upstream rejected: read-only access"; downstream continues | error |
| Full-access and read-only consoles attached | Both receive same output; only full-access keystrokes reach tmux | edge-case |
| Read-only console sends empty-tick frame | Accepted; liveness probe credited | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Read-only console keystrokes never reach tmux session | integration |
| VP-TBD | Downstream stream identical for read-only and full-access consoles | integration |
| VP-TBD | Explicit error returned on upstream reject (not silent drop) | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-015 ("Read-only session access mode") per capabilities.md §CAP-015 |
| L2 Domain Invariants | DI-010 (session authorization is access-node-enforced), DI-011 (role separation between Tier 1 and Tier 2 keys) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-015 ("Read-only session access mode") per capabilities.md §CAP-015 — this BC specifies the read-only enforcement behavior that CAP-015 defines as "upstream channel is rejected at the access node" |

## Related BCs

- BC-2.04.003 — related to: read-only attach follows the same admission path but with restricted role
- BC-2.04.006 — composes with: multi-console scenario includes read-only observers
- BC-2.05.003 — depends on: Tier 2 authorization determines read-only vs. full-access
