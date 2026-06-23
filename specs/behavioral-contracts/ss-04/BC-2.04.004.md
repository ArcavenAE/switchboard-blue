---
artifact_id: BC-2.04.004
document_type: behavioral-contract
level: L3
version: "1.1"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
bc_id: BC-2.04.004
subsystem: SS-TBD
capability: CAP-014
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
traces_to: [CAP-014]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.04.004: Console Detach Releases Session Without Closing It; Session Continues on Access Node

## Description

When a console detaches from a session, the console's channel to the access node is closed. The tmux session on the access node continues running — it is not terminated by the detach. Other consoles that are subscribed to the session (read-only or full-access) are unaffected. The session becomes available for re-attachment. The access node updates its presence advertisement to attached=false (if this was the last full-access console).

## Preconditions

1. A console has an active attached channel to a session.
2. The console initiates a detach (explicitly or by process exit).

## Postconditions

1. The console's channel to the access node is closed cleanly (FIN exchange or equivalent).
2. The tmux session on the access node continues running unchanged.
3. No keystrokes are forwarded from the detached console after the detach.
4. If this was the last full-access console, the access node updates presence advertisement: attached=false.
5. Read-only observers (if any) continue receiving the downstream stream.
6. The session becomes available for a new full-access console to attach.

## Invariants

1. Detach is non-destructive: it never terminates the underlying tmux session.
2. The access node does not require a graceful detach — a console process crash also results in a clean detach (the access node detects the channel closure).
3. **DI-010**: The access node manages session state; the router is not involved in detach processing.

## Trigger

Console operator runs `sbctl sessions detach`; console process exits; channel keepalive timeout.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-012) | Full-access console detaches; read-only observers remain | Read-only observers continue receiving output. Session continues. Session shows attached=false in advertisements (no full-access console). |
| EC-002 | Console crashes without sending detach | Access node detects channel closure on next keepalive timeout. Session released. Same outcome as graceful detach. |
| EC-003 (DEC-014) | tmux session closes after console detach | Access node detects session closure; sends session-terminated presence update. Any subsequent attach attempt returns E-SES-001. |
| EC-004 | Console detaches and immediately re-attaches | Second attach proceeds normally (as per BC-2.04.003). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| Console detaches from "agent-01" | Channel closed; "agent-01" continues on access node; presence: attached=false | happy-path |
| Console detaches; 1 read-only observer remains | Observer continues receiving output; access node does not close session | edge-case |
| Console crashes | Access node detects crash on keepalive timeout; session released; presence: attached=false | edge-case |
| Console detaches from session that closes 100ms later | Detach completes normally; session closure is a separate event | happy-path |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-TBD | tmux session continues running after console detach | integration |
| VP-TBD | Read-only observers unaffected by full-access console detach | integration |
| VP-TBD | Channel closure detected by access node within keepalive timeout | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-014 ("Console session attach and detach") per capabilities.md §CAP-014 |
| L2 Domain Invariants | DI-010 (session authorization is access-node-enforced) |
| Architecture Module | [filled by architect] |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-014 ("Console session attach and detach") per capabilities.md §CAP-014 — this BC specifies the detach half: "Detach releases the session without closing it" as stated in CAP-014 |

## Related BCs

- BC-2.04.003 — composes with: detach is inverse of attach
- BC-2.04.006 — related to: multi-observer scenario depends on non-destructive detach
