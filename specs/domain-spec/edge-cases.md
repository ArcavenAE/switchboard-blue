---
artifact_id: L2-edge-cases
document_type: domain-spec-section
level: L2
section: edge-cases
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
  - elem-asymmetric-half-channels
  - elem-dual-fastest-path-forwarding
  - elem-node-router-architecture
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Domain Edge Cases

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

Edge cases are domain-level situations where normal assumptions break down
or where the system's behavior must be explicitly defined. Each DEC-NNN maps
to capabilities in capabilities.md and will drive test scenarios.

---

## Session Continuity Edge Cases

**DEC-001 — IP address change mid-session**
A node's IP changes (wifi-to-LAN handoff, DHCP lease renew, mobile roaming).
Expected behavior: the node re-authenticates to the same router with its new
source IP. The router recognizes the node's cryptographic identity and
continues the session. No user intervention required. Active frames in-flight
at the moment of change may be lost; the recovery mechanism (idempotent replay
upstream, ARQ downstream) handles the gap.
_Affects: CAP-004 (session continuity), CAP-006 (path selection)._

**DEC-002 — Router failure with single-homed node (E router phase)**
In the E router phase, a node has exactly one router. If the E router goes
down, all sessions on that node are lost. No failover is possible with a
single router. This is an accepted limitation of the E router phase
(elem-mvp-scope-single-lan). The node should detect the router as unavailable
and report session loss clearly rather than hanging.
_Affects: CAP-004, failure-modes.md FM-001._

**DEC-003 — Router failure with multi-homed node**
A multi-homed node loses one of two or more connected routers. Expected
behavior: the node detects the failure via missed keep-alives, removes the
failed router from its active path set, and continues session on the
remaining path(s) without user intervention. Failover time target: <2 seconds.
_Affects: CAP-005, CAP-006._

**DEC-004 — Both paths fail simultaneously**
A multi-homed node loses all connected routers simultaneously (e.g., datacenter
outage affecting both uplinks). Sessions are lost. Expected behavior: the
quality indicator goes red; the console displays a clear "disconnected" state.
The node does not hang or freeze. Session state is preserved locally where
possible; the node reconnects when a router becomes reachable.
_Affects: CAP-004, CAP-021._

---

## Admission and Key Edge Cases

**DEC-005 — Admitted node's key is revoked mid-session**
A node is in an active session when its Tier 1 admission key is revoked.
Expected behavior: the revocation propagates to the router; the router closes
the node's connection on the next re-authentication event (challenge or
keep-alive). The session terminates cleanly rather than silently degrading.
The exact timing of session teardown after revocation depends on the
re-authentication interval (implementation decision).
_Affects: CAP-019, CAP-017._

**DEC-006 — Console presents Tier 1 key but lacks Tier 2 session auth**
A console is admitted to the SVTN (Tier 1 passes) but attempts to attach to
a session whose authorization list does not include the console's key.
Expected behavior: the access node rejects the attach request. The console
receives an explicit "unauthorized" response, not a timeout or silent failure.
The console remains on the SVTN and can discover sessions; it just cannot
attach to unauthorized ones.
_Affects: CAP-018, CAP-014._

**DEC-007 — Duplicate public key registered for different roles**
An operator registers the same public key against an SVTN with two different
role designations in separate operations. Expected behavior: last-write-wins
(LWW) per ADR-003 (see `.factory/specs/architecture/ARCH-04-admission-security.md`).
The most recent authenticated registration supersedes earlier entries for the
same `(node_pubkey, svtn_id)` pair. No conflict; no manual reconciliation
required.
_Affects: CAP-019. Cross-reference: ADR-003._

---

## Protocol and Frame Edge Cases

**DEC-008 — Frame arrives with unknown major version in outer header**
A router receives a frame whose version field indicates a major version the
router does not support. Expected behavior: the router rejects the frame and
logs the event. It does not forward an uninterpretable frame. The node
receives no delivery confirmation; it will retransmit based on its ARQ logic.
_Affects: CAP-003, DI-007._

**DEC-009 — Frame loop detected via drop cache miss**
A multi-hop topology produces a routing loop for some pathological forwarding
table state. A router receives a frame whose checksum is in its drop cache.
Expected behavior: the frame is silently discarded. The drop cache has bounded
size; checksums age out. A legitimate retransmit with new content produces a
different checksum and is not suppressed.
_Affects: CAP-010, DI-009._

**DEC-010 — Empty-tick frame storm on degraded path**
A path is degraded but not fully failed. The sender continues to emit empty-tick
frames at the tick interval. The receiver accumulates these with no payload,
incrementing missed-data counters. Expected behavior: the quality indicator
correctly reflects path quality from the RTT and loss data, not from whether
frames carry payload. An all-empty-tick traffic pattern from a healthy path
shows green; from a lossy path shows yellow/red.
_Affects: CAP-001, CAP-021, DI-008._

---

## Multi-Consumer Edge Cases

**DEC-011 — Two consoles attach to the same session simultaneously**
One console holds full-access; a second console requests read-only access.
Expected behavior: both are served. The access node delivers output once per
downstream frame; the router fans out to both consoles' subscriptions. Upstream
keystrokes from the read-only console are rejected by the access node.
Upstream keystrokes from the full-access console are accepted. Both consoles
see the same output.
_Affects: CAP-016, CAP-015, FR29–FR30._

**DEC-012 — Primary console detaches; read-only console still observing**
A full-access console detaches. One or more read-only consoles remain
subscribed to the session's output stream. Expected behavior: the read-only
consoles continue to receive output. The session on the access node is not
affected by the primary console's detach. A new full-access console may
attach subsequently.
_Affects: CAP-016, CAP-014._

---

## tmux Integration Edge Cases

**DEC-013 — tmux control mode unavailable or fails**
An access node starts on a machine where tmux is absent, or where tmux control
mode (`-CC`) fails to initialize (version too old, permission issue, tmux
behavior change). Expected behavior: the access node falls back to PTY proxy
mode. Session publishing works with reduced functionality: no structured
session metadata, no content-type detection, byte-rate heuristic for quality
signals only. The access node logs the fallback clearly.
_Affects: CAP-013, ASM-003._

**DEC-014 — tmux session closes while console is attached**
The tmux session a console is attached to exits (e.g., the shell process
exits, the tmux window is killed). Expected behavior: the access node detects
the session closure via control mode events (or PTY EOF). It sends a session-
terminated notification to the subscribed consoles. Consoles receive the
notification, display a clear "session ended" message, and detach cleanly.
The console remains on the SVTN and can attach to another session.
_Affects: CAP-014, CAP-011 (presence update on session close)._
