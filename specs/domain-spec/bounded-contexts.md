---
artifact_id: L2-bounded-contexts
document_type: domain-spec-section
level: L2
section: bounded-contexts
version: "1.0"
status: draft
producer: business-analyst
timestamp: 2026-06-23T00:00:00
phase: 1a
inputDocuments:
  - '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-07-001.md'
  - '_bmad-output/brainstorming/session-context-cache.md'
kos_anchors:
  - elem-mvp-scope-single-lan
  - elem-node-router-architecture
  - elem-single-binary-three-modes
  - elem-ssh-end-to-end-encryption
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Bounded Contexts

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

This section maps the nine subsystem buckets (from L2-INDEX.md) to their
processing boundaries, key translations, and scope-phase membership. It
also captures the scope-phase topology constraint as a context boundary.

---

## Scope-Phase Topology Boundary

The three product scope phases (E Router Release, Multi-Path/Multi-Hop,
Global Topology) are not different products — they are deployment topology
configurations of the same binary. They do, however, represent distinct
context boundaries for design and implementation work:

| Phase | Topology | Context boundary |
|-------|----------|-----------------|
| E Router Release | Single LAN, single E router, single hop | No inter-router protocol. Local admission DB. All nodes co-LAN. |
| Multi-Path / Multi-Hop | PE routers, distributed topology, multi-hop | Inter-router link-state + Noise auth. Distributed admission DB. Membership propagation. |
| Global Topology | P routers at cloud POPs, provider-scale | P router build target (no node protocol). PE routers at POPs. E/PE single binary unchanged. |

Capabilities that belong only to Phase 2+ are marked P1 or P2 in capabilities.md.
The domain invariants (invariants.md) apply across all phases.

---

## Context Boundaries by Subsystem

### session-networking (sn)
**Boundary:** The timeslice clock, half-channel model, and frame encoding
are the core data plane. This context owns the on-wire representation.
**Upstream boundary:** Receives application bytes (keystrokes, terminal
output) from session-access and passes assembled frames to multipath-forwarding.
**Downstream boundary:** Receives frames from multipath-forwarding and delivers
decoded byte streams to session-access.
**Translation:** Raw byte stream ↔ framed, sequenced, HMAC-tagged network frame.

### multipath-forwarding (mf)
**Boundary:** Owns path selection, duplicate suppression, loop prevention, and
the node-side multi-homing logic. Does not parse channel headers (endpoint-only).
**Upstream boundary:** Receives assembled frames from session-networking and
dispatches to router connections.
**Downstream boundary:** Receives incoming frames from router connections and
delivers to session-networking after deduplication.
**Translation:** Single logical frame ↔ multiple physical path dispatches.

### session-discovery (sd)
**Boundary:** Owns the multicast presence protocol. Produces and consumes
presence advertisements. Separate from the session data path.
**Upstream boundary:** Receives session state change events from session-access.
**Downstream boundary:** Delivers session list to console-operations and the
console attach flow.
**Translation:** Session state (name, attached, quality) ↔ SVTN multicast
advertisement payload.

### session-access (sa)
**Boundary:** Owns the tmux control mode integration (access node side) and
the console stream subscription (console side). Mediates between the tmux
process and the network data path.
**Translation (access node):** tmux control mode `%output` events ↔ downstream
frames. Console keystrokes ↔ tmux input.
**Translation (console):** Upstream keystrokes from terminal emulator ↔ upstream
frames. Downstream frames ↔ bytes written to terminal emulator / PTY.

### admission-security (as)
**Boundary:** Owns the Tier 1 (SVTN admission) and Tier 2 (session auth) key
models. The router enforces Tier 1; the access node enforces Tier 2.
**Note on split enforcement:** The split between router-enforced (Tier 1) and
access-node-enforced (Tier 2) is a deliberate context boundary — not an
implementation shortcut. The router's context is identity and admission; the
access node's context is session permission. Conflating them would break
DI-010.
**Translation:** OpenSSH public key + signed challenge ↔ SVTN admission grant.
Authorized console key list ↔ per-session attach permission.

### quality-observability (qo)
**Boundary:** Owns latency measurement, loss detection, and the quality
indicator state machine. Consumes empty-tick heartbeats from session-networking
and per-path RTT/loss data from multipath-forwarding.
**Translation:** Raw RTT measurements and missed-frame counts ↔ green/yellow/red
quality indicator state ↔ TLPKTDROP degradation signal.

### network-management (nm)
**Boundary:** Owns the sbctl CLI surface, the control node daemon, and key
lifecycle. This context is the operator's interaction surface — it translates
operator intent into network configuration changes.
**Translation:** CLI commands ↔ daemon API calls ↔ SVTN state mutations (key
register/revoke/expire, SVTN create/destroy).

### console-operations (co)
**Boundary:** Owns the console control plane — the programmatic interface for
attach, detach, switch, and navigate. Separate from the console's session
display (which belongs to session-access).
**Translation:** sbctl console commands ↔ console daemon control messages ↔
session state transitions.

### deployment-operations (do)
**Boundary:** Owns the deployment lifecycle — binary distribution, config file
format, E-to-PE graduation, graceful shutdown, rolling update. Orthogonal to
the session data path.
**Translation:** Config file + startup flags → runtime router mode (E/PE/P).
SIGTERM → graceful drain → clean shutdown.

---

## Context Translation Points (Key Interfaces)

These are the points where data crosses context boundaries and semantic
translation occurs. Each is a future architecture decision point.

| From | To | Translation |
|------|----|-------------|
| session-access → session-networking | tmux `%output` byte stream → downstream frame batch |
| session-networking → multipath-forwarding | assembled frame → path dispatch set |
| multipath-forwarding → quality-observability | RTT measurements, missed frames → quality state |
| session-discovery → console-operations | SVTN presence advertisements → session list for attach |
| admission-security → network-management | Key registration events → admission DB update |
| network-management → admission-security | CLI key register/revoke → signed key change request |

---

## What Is Explicitly Not in Scope for Any Phase

These items are out of scope for the domain as currently defined:

- **Multi-host agent scheduling**: Identified in PRD as "marvel integration" —
  a separate system. Switchboard provides the session network; marvel provides
  the scheduling layer. The interface is undefined (frontier: question-marvel-integration).
- **MCP (Model Context Protocol) transport**: Explored in brainstorming session
  #3 and parked. Infrastructure alignment is real but an undisclosed context
  prevents full assessment. Not a current domain capability.
- **Non-terminal traffic**: Switchboard's framing and content-type models are
  purpose-built for terminal sessions (keystrokes, terminal output). General
  IP tunneling or file transfer are not domain capabilities.
