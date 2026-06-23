---
artifact_id: BC-2.04.006
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.04.006
subsystem: session-access
architecture_module: internal/session
capability: CAP-016
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
traces_to: [CAP-016]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.04.006: Two or More Consoles May Subscribe to the Same Session Output Simultaneously

## Description

Multiple consoles can be subscribed to the same session's output stream at the same time. The access node delivers each downstream frame once to the router, which fans out delivery to all subscribed consoles. This supports the read-only observer use case (Priya observing Devon's session) and the fleet-view use case (Kai monitoring multiple sessions). There is no upper bound on simultaneous observers defined at the domain level.

## Preconditions

1. At least one console is already subscribed to the session's downstream stream.
2. A second console requests to subscribe (with valid Tier 1 and Tier 2 authorization).

## Postconditions

1. Both consoles receive the same downstream frames.
2. The access node delivers each frame once to the network; the router fans out via SVTN multicast-to-subscriber-set.
3. Keystrokes from any full-access console are forwarded to tmux; keystrokes from read-only consoles are rejected (per BC-2.04.005).
4. The detach of one console does not affect the other consoles' subscriptions.
5. There is no artificial limit on simultaneous subscribers at the protocol level (implementation may impose a practical limit; architecture decision).

## Invariants

1. **DI-001**: Each subscriber receives an identically encrypted downstream stream — there is no per-subscriber re-encryption at the router.
2. Fan-out is a router responsibility — the access node sends one copy per frame.
3. All full-access console keystrokes are serialized by the access node before forwarding to tmux (no keystroke race condition).

## Trigger

Second (or subsequent) console attaches to an already-subscribed session.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-011) | Full-access and read-only console both attached simultaneously | Both receive identical output; only full-access keystrokes reach tmux. |
| EC-002 (DEC-012) | Full-access console detaches; read-only still subscribed | Output continues to read-only console. Session not affected. |
| EC-003 | Two full-access consoles both send keystrokes simultaneously | Keystrokes from both are accepted; serialized by the access node before forwarding to tmux. tmux receives them in the order they arrive at the access node. |
| EC-004 | 50 consoles subscribed to one session | Router fans out to all 50; no artificial limit. Performance is an NFR (NFR-004). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Console A (full-access) and Console B (read-only) attach to "agent-01" | Both receive identical output stream | happy-path |
| Console A types 'ls'; Console B receives no permission to type | Access node forwards Console A's 'ls'; rejects Console B's keystroke attempt | happy-path |
| Console A detaches | Console B continues receiving; access node: attached=false (no more full-access) | edge-case |
| Two full-access consoles both type simultaneously | Both keystrokes reach tmux (serialized); no crash or data corruption | edge-case |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | All subscribed consoles receive the same bytes per downstream frame | integration |
| VP-TBD | Detach of one console does not disrupt other subscribers | integration |
| VP-TBD | Keystroke serialization: no tmux corruption under concurrent keystrokes | integration/fuzz |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-016 ("Simultaneous multi-console session viewing") per capabilities.md §CAP-016 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — router fans out without re-encrypting) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-016 ("Simultaneous multi-console session viewing") per capabilities.md §CAP-016 — this BC specifies the fan-out behavior that CAP-016 defines as "the router fans out to all subscribed consoles" |

## Related BCs

- BC-2.04.003 — depends on: each subscriber follows the attach flow
- BC-2.04.004 — composes with: non-destructive detach makes multi-subscriber safe
- BC-2.04.005 — composes with: read-only enforcement for observer consoles
