---
artifact_id: L2-failure-modes
document_type: domain-spec-section
level: L2
section: failure-modes
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
  - elem-node-router-architecture
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Failure Modes

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

Failure modes are runtime situations where the system cannot meet its normal
operating contract. Each FM-NNN specifies the trigger, observable behavior,
recovery path, and whether the failure is within the expected threat model.

---

## Subsystem: session-networking / multipath-forwarding

**FM-001 — Single router failure (E router phase)**
Trigger: The E router process crashes, becomes unreachable, or the LAN link
fails.
Observable behavior: Active sessions freeze immediately. Quality indicator
goes red (if the indicator update can be delivered before the path dies) or
hangs without update. The console does not receive an explicit "disconnected"
message unless the router had time to send a drain signal.
Recovery: No automatic recovery in E router phase — single router is a
single point of failure by design (elem-mvp-scope-single-lan). User must
restart the router and reconnect.
Threat model: Expected. Not a death condition — the E router phase is a
proof-of-concept deployment. The death condition is not detecting this
failure (session freezes silently without indicator).

**FM-002 — All paths degrade below threshold (multi-path topology)**
Trigger: Measured RTT or loss on all connected router paths exceeds the
degradation threshold simultaneously.
Observable behavior: Quality indicator moves to red. Console receives
degradation signals (TLPKTDROP on overdue frames). Sessions remain connected
but interactive responsiveness degrades.
Recovery: Quality recovers automatically when path metrics improve. No user
intervention required.
Threat model: Expected. The correct response is transparency — red indicator,
not silent freeze.

**FM-003 — Frame duplication storm (drop cache overflow)**
Trigger: A routing loop or misconfigured multi-path topology floods the
network with duplicate frames faster than the drop cache can absorb.
Observable behavior: Increased CPU on routers processing and discarding
duplicates. Potential latency increase from drop-cache contention.
Recovery: Drop cache is bounded; excess checksums age out. Router operator
should inspect forwarding tables via `sbctl router status`. No user session
impact beyond latency increase.
Threat model: Operational misconfiguration. Should not occur in correct
deployments.

---

## Subsystem: session-access / session-discovery

**FM-004 — Access node loses connection to tmux control mode**
Trigger: The tmux control mode process exits, the socket becomes unavailable,
or tmux crashes while the access node is publishing sessions.
Observable behavior: The access node can no longer receive `%output`
notifications or query session state. Published sessions appear stale or
disappear from discovery.
Recovery: Access node detects the disconnection and attempts to re-establish
control mode. Falls back to PTY proxy if control mode cannot be restored.
Sends a "session unavailable" presence update to the SVTN.
Threat model: Expected operational event (tmux restart, upgrade). Must be
handled gracefully.

**FM-005 — Session discovery presence message lost or stale**
Trigger: A node's presence advertisement is lost in transit, or a node fails
to send its periodic heartbeat.
Observable behavior: A session that exists does not appear in the console's
session list, or a session that has ended still appears for one heartbeat
interval.
Recovery: Eventual consistency — the next heartbeat or state-change event
corrects the view. On-demand presence ping refreshes immediately.
Threat model: Expected in lossy network conditions. The absence of a session
in the list for one heartbeat interval is acceptable.

---

## Subsystem: admission-security

**FM-006 — HMAC verification failure (unexpected)**
Trigger: A frame arrives at the router with an HMAC that does not match the
expected value for any admitted node in the SVTN. Could indicate a non-member
sending frames, a key mismatch after key rotation, or a corrupted frame.
Observable behavior: Router rejects the frame, logs the event (SVTN ID,
source address, frame type). The sending node receives no delivery
confirmation and will retransmit. Repeated failures from the same source
trigger an admission alert.
Recovery: If due to key rotation lag, the node re-authenticates with the new
key and sessions resume. If due to a non-member, frames continue to be
rejected silently.
Threat model: Expected at low frequency (key rotation, network corruption).
High-frequency from a single source is an anomaly.

**FM-007 — Key revocation propagation delay**
Trigger: A key is revoked via the control node, but propagation to all routers
has not completed when the revoked node attempts to re-authenticate.
Observable behavior: The revoked node may successfully authenticate to a router
that has not yet received the revocation. Sessions on non-updated routers
continue until the next re-authentication challenge.
Recovery: Revocation propagates via the router distributed database. After
propagation completes, the next re-authentication challenge terminates the
session.
Threat model: Acknowledged. The propagation window is a known gap. For
immediate revocation, the router management plane must be consulted directly
(architecture decision, not domain invariant).

---

## Subsystem: quality-observability

**FM-008 — Quality indicator stuck at green during actual degradation**
Trigger: Keep-alive probes are succeeding (path alive) but data frames are
experiencing loss that the keep-alive traffic is not representative of. Or:
the empty-tick frame machinery is broken and not sending liveness probes.
Observable behavior: The console shows green but sessions feel laggy or
partial.
Recovery: If the empty-tick mechanism is functioning correctly, this cannot
happen — empty ticks are sent at every tick, so any loss on the data path
is also loss on the probe path. If the empty-tick mechanism is broken, the
quality indicator is unreliable. This is a bug in CAP-001/CAP-021.
Threat model: Implementation defect. DI-008 (empty ticks always fire) is
the invariant that prevents this.

---

## Subsystem: deployment-operations

**FM-009 — Router process exits without sending drain signal**
Trigger: Router crashes (OOM, SIGSEGV, OS kill) without completing graceful
shutdown.
Observable behavior: Connected nodes do not receive drain signal. Sessions
freeze rather than migrating. After the keep-alive timeout, nodes detect the
failure and migrate (if multi-homed) or report disconnection.
Recovery: Multi-homed nodes failover to alternate routers within the keep-alive
timeout window. Single-router (E phase) deployments require manual restart.
Threat model: Expected operational event. Drain signal (CAP-027) requires
SIGTERM; crash bypasses it. Acceptable tradeoff.

**FM-010 — Config file error on router startup**
Trigger: Malformed YAML, missing required fields, or invalid key file path
in the router config file.
Observable behavior: Router daemon refuses to start and exits with a clear
error message identifying the config problem. No sessions are affected (router
was not running).
Recovery: Operator corrects config and restarts.
Threat model: Operational error. Must surface a clear, actionable error
message — not a panic or silent failure.

**FM-011 — tmux not present on access node machine**
Trigger: An access node starts on a machine where the `tmux` binary is absent
or not in PATH.
Observable behavior: Access node cannot use control mode. Falls back to PTY
proxy mode and reports the fallback in its startup log. Sessions can still be
published via PTY proxy with reduced functionality.
Recovery: Install tmux for full functionality. PTY fallback continues
to work in the interim.
Threat model: Expected in minimal environments. PTY fallback is the mitigation.

**FM-012 — sbctl cannot connect to daemon**
Trigger: An operator runs an `sbctl` command targeting a daemon that is not
running, not listening on the configured port, or behind a firewall.
Observable behavior: `sbctl` reports a clear connection error (daemon not
reachable, connection refused, timeout) with the attempted address. It does
not silently succeed or produce a misleading output.
Recovery: Operator verifies the daemon is running and the address is correct.
Threat model: Operational. Must never produce a misleading "success" output
when the daemon is not reachable.
