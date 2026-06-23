---
artifact_id: BC-2.08.001
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.08.001
subsystem: console-operations
architecture_module: cmd/sbctl
capability: CAP-025
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
traces_to: [CAP-025]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.08.001: Console Remotely Controllable via sbctl — Attach, Detach, Switch Session, Navigate

## Description

A console daemon can be controlled remotely via `sbctl`. The controlling operator may attach the console to a session, detach it, switch to a different session, or navigate (scroll) — without being physically present at the console machine. The viewing operator (at the console terminal) and the controlling operator (via sbctl) may be different principals, both holding appropriate authorization.

## Preconditions

1. The console daemon is running and registered on the SVTN.
2. The controlling operator's sbctl key is authorized for the console management operations.
3. For attach/switch: the target session exists and the console's Tier 2 key is authorized for that session.

## Postconditions

1. **Attach**: console is attached to the specified session; terminal output appears at the console; equivalent to `sbctl sessions attach` run locally.
2. **Detach**: console detaches from the current session per BC-2.04.004.
3. **Switch session**: console detaches from current session and attaches to the new session atomically from the remote operator's perspective.
4. **Navigate**: scrollback navigation commands sent to the console's terminal display.
5. All operations return confirmation or error via sbctl.

## Invariants

1. Remote control does not bypass Tier 2 authorization: the console daemon's key must be authorized for the target session.
2. The viewing operator at the console sees the same operations as if they ran them locally.
3. Remote control commands use the same SVTN channel as regular traffic — no separate out-of-band channel.

## Trigger

Operator runs `sbctl console attach|detach|switch|navigate`.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Remote attach to session the console is not authorized for | E-ADM-006 "session authorization denied"; console remains in current state. |
| EC-002 | Remote switch: console is attached to A; switch to B requested; A closes before switch completes | Detach from A completes (session A closed); attach to B proceeds; console now on B. |
| EC-003 | Viewing operator and controlling operator both send conflicting commands simultaneously | Commands are serialized by the console daemon; both are executed in order of arrival. |
| EC-004 | Console daemon unreachable by sbctl | E-NET-001 "daemon unreachable: <console-address>" (per BC-2.07.003). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl console attach --session=agent-01 --console=<addr>` | Console attaches to agent-01; confirmation returned | happy-path |
| `sbctl console detach --console=<addr>` | Console detaches; confirmation returned | happy-path |
| `sbctl console switch --session=agent-02 --console=<addr>` | Console detaches from current; attaches to agent-02 | happy-path |
| `sbctl console attach --session=unauthorized --console=<addr>` | E-ADM-006 "session authorization denied" | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | Remote attach/detach has same effect as local attach/detach | integration |
| VP-TBD | Tier 2 authorization enforced for remote attach | integration |
| VP-TBD | Remote commands serialized correctly under concurrent invocation | integration/fuzz |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-025 ("Remote console control plane") per capabilities.md §CAP-025 |
| L2 Domain Invariants | DI-010 (session authorization is access-node-enforced — still applies via the console daemon's key) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-025 ("Remote console control plane") per capabilities.md §CAP-025 — this BC specifies the remote controllability that CAP-025 defines as "remotely controllable via sbctl: attach, detach, switch session, navigate" |

## Related BCs

- BC-2.04.003 — depends on: remote attach invokes the attach flow
- BC-2.04.004 — depends on: remote detach invokes the detach flow
- BC-2.07.002 — composes with: sbctl authentication is shared
