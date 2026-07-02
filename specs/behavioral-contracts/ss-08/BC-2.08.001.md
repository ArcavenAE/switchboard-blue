---
artifact_id: BC-2.08.001
document_type: behavioral-contract
level: L3
version: "1.5"
status: draft
producer: product-owner
timestamp: 2026-07-02T00:00:00
phase: 1a
bc_id: BC-2.08.001
subsystem: console-operations
architecture_module: internal/session
capability: CAP-025
priority: P1
criticality: high
scope_phase: PE
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-07-02
    version: "1.4"
    actor: spec-steward
    change: >
      F-P4L3-MED-002 (spec-vs-impl mis-anchor): Inv-3 v1.2/v1.3 wording pinned Unix-socket
      unconditionally; console mode uses TCP loopback per BC-2.07.004 EC-013 / AC-014 Ruling D.
      Rewording defers per-mode transport type to BC-2.07.004 EC-013; ADR-012 remains canonical
      mgmt-plane ADR. Stories row bumped to S-7.03 v1.4.
  - date: 2026-07-02
    version: "1.3"
    actor: spec-steward
    change: >
      F-P3L3-MED-001: bump Stories row cell reference to S-7.03 v1.3 (POL-003 candidate
      sync — story v1.2→v1.3 landed 2026-07-02, this row was stale). No behavioral changes.
  - date: 2026-07-01
    version: "1.2"
    actor: spec-steward
    change: >
      RULING-W6TB-C (decisions/RULING-W6TB-C-console-transport.md): retract Inv-3
      "same SVTN channel as regular traffic — no separate out-of-band channel" language.
      Replace with management-plane Unix-socket transport requirement. Rationale: cmd/sbctl
      has no ARQ stack; forbidden from importing internal/routing per ARCH-08 §6.6;
      implementing SVTN-channel transport would require a second control-message protocol
      inside the SVTN data plane with no CAP/BC grounding. Security intent (no privilege
      bypass) is satisfied by mgmt-plane authentication (Ed25519 fail-closed per S-6.03 /
      BC-2.07.004; internal/mgmt.Server authenticates the operator key). Tier-2
      authorization (Inv-1) enforced by internal/session regardless of transport.
      S-7.03 v1.2 is the implementing story.
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
3. Remote control commands use the management-plane transport (ADR-012). Per-mode transport type is determined by BC-2.07.004 EC-013: Unix socket for access mode; TCP loopback (`127.0.0.1:9091`) for console mode. No separate data-plane or out-of-band channel is introduced. The operator's key is authenticated by `internal/mgmt.Server` (Ed25519 fail-closed per BC-2.07.004); console Tier-2 authorization (Inv-1) is enforced by the console daemon's session layer (`internal/session`) regardless of transport. Implementing the retracted "same SVTN channel" approach is architecturally forbidden: `cmd/sbctl` must not import `internal/routing`, `internal/arq`, or `internal/multipath` (ARCH-08 §6.6). (Originally patched v1.2, W6TB-C Ruling, 2026-07-01 — retracted: "same SVTN channel as regular traffic — no separate out-of-band channel". Revised v1.4, F-P4L3-MED-002, 2026-07-02 — unconditional "Unix-socket" phrasing replaced with per-mode deference to BC-2.07.004 EC-013; ADR-006 defers to BC-2.07.004 EC-013 for per-mode socket type.)

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
| VP-050 | Remote attach/detach has same effect as local attach/detach | integration |
| VP-050 | Tier 2 authorization enforced for remote attach | integration |
| VP-050 | Remote commands serialized correctly under concurrent invocation | integration/fuzz |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-025 ("Remote console control plane") per capabilities.md §CAP-025 |
| L2 Domain Invariants | DI-010 (session authorization is access-node-enforced — still applies via the console daemon's key) |
| Architecture Module | internal/session |
| Stories | S-7.03 v1.4 (console remote-control: attach/detach/switch via mgmt-plane transport per BC-2.07.004 EC-013; W6TB-C ruling + F-P4L3-MED-002) |
| Capability Anchor Justification | CAP-025 ("Remote console control plane") per capabilities.md §CAP-025 — this BC specifies the remote controllability that CAP-025 defines as "remotely controllable via sbctl: attach, detach, switch session, navigate" |

## Related BCs

- BC-2.04.003 — depends on: remote attach invokes the attach flow
- BC-2.04.004 — depends on: remote detach invokes the detach flow
- BC-2.07.002 — composes with: sbctl authentication is shared
- BC-2.07.004 — depends on: mgmt-plane Ed25519 authentication for operator key (Inv-3 v1.2)

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.5 | 2026-07-02 | spec-steward | F1 remediation from W-6 wave-gate Pass-3 Adv-B: retro-annotate v1.3 changelog row with `governance_leaf: true` per POL-003 Exception A audit-tool compatibility. Shape now matches BC-2.07.001 v1.13. No behavioral changes. [governance_leaf: true — annotation-shape correction, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A] |
| 1.4 | 2026-07-02 | spec-steward | F-P4L3-MED-002 (spec-vs-impl mis-anchor): Inv-3 v1.3 wording pinned Unix-socket unconditionally; console mode uses TCP loopback per BC-2.07.004 EC-013 / AC-014 Ruling D. Inv-3 rewording defers per-mode transport type to BC-2.07.004 EC-013; ADR-012 remains canonical mgmt-plane ADR; ADR-006 defers to BC-2.07.004 EC-013 for per-mode socket type. Stories row bumped S-7.03 v1.3 → v1.4. |
| 1.3 | 2026-07-02 | spec-steward | F-P3L3-MED-001: bump Stories row cell reference to S-7.03 v1.3 (POL-003 candidate sync — story v1.2→v1.3 landed 2026-07-02, this row was stale). No behavioral changes. [governance_leaf: true — Stories-row pin sync, downstream VP/story pins DO NOT need to re-sync per POL-003 Exception A] |
| 1.2 | 2026-07-01 | spec-steward | RULING-W6TB-C (decisions/RULING-W6TB-C-console-transport.md): Inv-3 retracted and replaced. "Same SVTN channel as regular traffic — no separate out-of-band channel" was architecturally incompatible with the established JSON-over-Unix-socket management-plane transport used by all sbctl commands (ADR-006/ADR-012). Inv-3 now correctly states the management-plane transport requirement. Security intent preserved: operator key authentication via internal/mgmt.Server (BC-2.07.004); Tier-2 authorization via internal/session. Forbidden-import constraint (ARCH-08 §6.6) documented explicitly. S-7.03 v1.2 is the implementing story. Update Traceability Stories row to cite S-7.03 v1.2 anchoring. Add BC-2.07.004 to Related BCs. |
| 1.1 | 2026-06-23 | product-owner | Initial draft — console remote control via sbctl: attach, detach, switch session, navigate. |
