---
artifact_id: BC-2.04.003
document_type: behavioral-contract
level: L3
version: "1.3"
status: draft
producer: product-owner
timestamp: 2026-06-26T12:00:00
phase: 1a
bc_id: BC-2.04.003
subsystem: session-access
architecture_module: internal/session
capability: CAP-014
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-06-26
    version: "1.2"
    change: "adversary pass-3 F-H-6/F-PG-2: anchor KeystrokeSink abstraction in Inv-4; mark PC-4 (presence advertisement) and PC-5 (initial screen refresh) deferred to future story"
  - date: 2026-06-26
    version: "1.3"
    change: "adversary pass-4 F-H-2: strengthen Inv-4 — SendKeystroke MUST validate sessionName matches attached session; mismatch returns E-SES-006 / ErrSessionMismatch"
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
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.04.003: Console Attaches to Session by Name; Receives Downstream Stream and Sends Upstream Keystrokes

## Description

A console attaches to a remote session by specifying the session name (not a hostname or IP). The console establishes a channel with the access node hosting that session. On successful attach: the console receives the downstream output stream and sends upstream keystrokes. The session becomes interactive. Tier 2 authorization is verified by the access node before the channel is established.

## Preconditions

1. The console is admitted to the SVTN (Tier 1 admission complete).
2. The console's Tier 2 session authorization key is registered on the target access node for the named session.
3. The named session exists and is published on the SVTN.
4. The session is not already attached by another full-access console (if exclusive mode — implementation decision; shared attach is allowed per CAP-016).

## Postconditions

1. A bidirectional channel is established between console and access node.
2. The console receives the downstream half-channel (terminal output) from the access node.
3. The console's upstream half-channel (keystrokes) is accepted by the access node and forwarded to the tmux session via the `KeystrokeSink` abstraction (see Inv-4).
4. **[DEFERRED — presence-advertisement story]** The access node's session advertisement updates to attached=true. This postcondition is documented for completeness but its enforcement is deferred to a future advertisement-update story (tentatively S-3.03 or later). S-3.02 does not implement presence advertisement.
5. **[DEFERRED — S-3.03 or later]** The console displays the current terminal output state (implementation: may request a full screen refresh from tmux on attach). Initial screen refresh on attach is not implemented in S-3.02.

## Invariants

1. **DI-010**: Tier 2 authorization is enforced by the access node. The router does not decide whether a console may attach.
2. **DI-011**: Tier 2 session authorization is independent of Tier 1 admission. A console admitted to the SVTN cannot attach without Tier 2 authorization.
3. Session content flows SSH-encrypted end-to-end; the channel is not a raw TCP stream.
4. **Inv-4 (Keystroke forwarding boundary + session validation):** Upstream keystrokes from any attached console are accepted by `AccessNode.SendKeystroke(key, sessionName, payload)` and dispatched to a `KeystrokeSink` (the tmux command-mode pipe or PTY master) under a per-AccessNode serialization mutex. The `KeystrokeSink` interface is defined in `internal/session` and implemented by `internal/tmux.SessionConnector`. This boundary ensures that all keystroke routing is decoupled from the transport layer and testable via mock `KeystrokeSink` implementations. **Session validation obligation:** Before dispatching to the `KeystrokeSink`, `SendKeystroke` MUST verify that the `sessionName` argument matches the session_name recorded when this console attached. If the names do not match, `SendKeystroke` MUST return `E-SES-006 / session.ErrSessionMismatch` ("session: console <console_id> attached to session <attached_session_name>, not <requested_session_name>") without forwarding any keystroke. This catches misconfigured callers that route to the wrong access node or hold stale session state. It also anchors the future S-3.03 `SessionAuth` pattern, which will extend this boundary for authorization checks.

## Trigger

Console operator runs `sbctl sessions attach <session-name>` or equivalent API call.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-006) | Console has Tier 1 (SVTN admission) but not Tier 2 for this session | Access node rejects attach with E-ADM-006 "session authorization denied"; console receives explicit rejection. |
| EC-002 | Named session does not exist | Access node returns E-SES-001 "session not found: <session-name>". Console receives explicit error. |
| EC-003 | Session exists but access node is unreachable | Router returns E-NET-005 "access node unreachable". Session may appear in list (stale advertisement). |
| EC-004 (DEC-011) | Second console attempts to attach while first is attached | Both attach succeeds (per CAP-016); see BC-2.04.006. |
| EC-005 | SendKeystroke called with wrong session_name (sessionName does not match the session_name the console attached with) | `SendKeystroke` returns E-SES-006 / `session.ErrSessionMismatch`. No keystroke is forwarded. Caller must re-attach or correct routing. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl sessions attach agent-01`; Tier 2 authorized | Channel established; downstream stream starts; keystrokes forwarded; quality indicator shown | happy-path |
| `sbctl sessions attach agent-01`; Tier 2 not authorized | E-ADM-006 "session authorization denied for agent-01" | error |
| `sbctl sessions attach nonexistent` | E-SES-001 "session not found: nonexistent" | error |
| Two consoles attach to same session | Both channels established; both receive output (BC-2.04.006) | edge-case |
| Console attached to "agent-01" calls `SendKeystroke(key, "agent-02", payload)` | E-SES-006 ErrSessionMismatch; no keystroke forwarded | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-033 | Tier 2 authorization checked before channel established | integration |
| VP-033 | Downstream stream starts immediately on successful attach | e2e |
| VP-033 | Explicit error returned (not timeout) when session not found | unit |
| VP-012 | SessionAuth rejects unauthorized console key | proptest |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-014 ("Console session attach and detach") per capabilities.md §CAP-014 |
| L2 Domain Invariants | DI-010 (session authorization is access-node-enforced), DI-011 (role separation between Tier 1 and Tier 2 keys) |
| Architecture Module | internal/session |
| Stories | [filled by story-writer] |
| Capability Anchor Justification | CAP-014 ("Console session attach and detach") per capabilities.md §CAP-014 — this BC specifies the attach half of the CAP-014 operation including the "selects by name" requirement |

## Related BCs

- BC-2.05.003 — depends on: Tier 2 authorization enforcement
- BC-2.04.004 — composes with: detach is the inverse of this BC
- BC-2.04.006 — related to: multi-console attach allowed
