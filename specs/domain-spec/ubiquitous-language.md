---
artifact_id: L2-ubiquitous-language
document_type: domain-spec-section
level: L2
section: ubiquitous-language
version: "1.0"
status: draft
producer: business-analyst
timestamp: 2026-06-23T00:00:00
phase: 1a
inputDocuments:
  - '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-07-001.md'
  - '_bmad-output/brainstorming/naming-node-type-parking-lot.md'
  - '_bmad-output/brainstorming/session-context-cache.md'
kos_anchors:
  - elem-asymmetric-half-channels
  - elem-dual-fastest-path-forwarding
  - elem-mvp-scope-single-lan
  - elem-node-router-architecture
  - elem-single-binary-three-modes
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Ubiquitous Language

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

All terms in this glossary are canonical. Use these exact names in code,
documentation, CLI output, and specifications. Where a term has an alias
or was rejected in favor of this term, the rejection is noted.

---

## Node Types

**Access node** — A Switchboard daemon that publishes tmux sessions over an
SVTN. Runs on the machine where the sessions live. Connects to one or more
routers. Never communicates directly with consoles. _Aliases considered and
rejected: anchor, exchange, CO (Central Office), switch — see naming-node-type-parking-lot.md._

**Console** — A Switchboard daemon that discovers and attaches to remote
sessions. The operator's entry point to the network. May hold full-access or
read-only keys. _Do not confuse with tmux "console" usage._

**Control node** — A Switchboard daemon that manages SVTN lifecycle and key
registration. A network participant, not an infrastructure component. Connects
to the SVTN as a peer node. _Do not call it "control plane" — that term
applies to the router control plane, which is distinct._

---

## Router Modes

**E router (Edge-local)** — A router instance with no upstream router
connections configured. Same binary as PE. The five-minute getting-started
experience. LAN-only — nodes must be co-located on the same LAN.

**PE router (Provider Edge)** — A router instance with both node-facing
(access node, console, control node connections) and router-facing (upstream
PE/P connections) interfaces. The production router for multi-site deployments.

**P router (Provider Core)** — A router instance with router-facing interfaces
only; no node protocol. Pure frame forwarding. Theoretical in current scope —
not built until justified by production data. _Note: P here is "Provider Core,"
borrowed from telecom MPLS terminology, not a file permission._

**Router graduation** — The act of upgrading an E router to PE by adding
upstream router connections to its config file. Same binary, same process,
no reinstall.

---

## Network Constructs

**SVTN (Switched Virtual Terminal Network)** — The canonical network type
name. A cryptographically isolated virtual network for terminal sessions.
"Switched" (forwarding through routers), "virtual" (overlay), "terminal"
(what it carries). _Note: the PRD uses both "SVTN" and "VSN" (Virtual Switched
Network). SVTN is preferred in the PRD's executive summary. Both appear in
source documents. For new artifacts, use SVTN. The domain spec uses SVTN
throughout._

**VSN** — An older term for what is now called SVTN. Appears in brainstorming
documents and some PRD sections. Treat as a synonym. When writing new
artifacts, use SVTN.

**Channel** — An active end-to-end connection between one console and one
access node, carrying a tmux session over the SVTN. One channel = two
half-channels.

**Half-channel** — One direction of a channel. Upstream (keystrokes) and
downstream (terminal output) are separate half-channels with independent
clocks, sequence spaces, and recovery strategies.

**Upstream half-channel** — Console-to-access-node traffic. Keystrokes. Tiny,
ordered, loss-intolerant. Uses idempotent replay (U-C sliding window).

**Downstream half-channel** — Access-node-to-console traffic. Terminal output.
Bursty, state-syncable. Uses reliable ordered stream with ARQ in MVP (D-A);
content-type-aware hybrid (D-CE) post-MVP.

---

## Protocol Terms

**Timeslice framing** — The framing model where each half-channel fires its
transmit clock on a fixed tick interval, regardless of whether there is data.
"The bus leaves on time, full or not." An empty frame is a liveness signal.

**Tick** — One firing of a half-channel's timeslice clock.

**Outer header** — The 44-byte router-visible header on every frame: version,
frame type, SVTN ID, destination address, source address, length, HMAC.

**Channel header** — The endpoint-visible inner header: channel ID, sequence
number, sender timestamp, FEC metadata, flags. Follows the outer header.
Opaque to routers.

**HMAC** — The 16-byte authentication tag in the outer header. Computed by
the sending node using its SVTN admission key. Verified by the first router
to reject frames from non-admitted sources.

**Node address** — An 8-byte value derived as `hash(SVTN-ID || public-key)`.
Self-assigned; no registration authority needed.

**TLPKTDROP (Too-Late Packet Drop)** — A signal sent when a frame cannot be
delivered within the perception deadline. Causes the console to advance its
degradation indicator. Borrowed from SRT protocol.

**Duplicate-and-race** — The multi-path forwarding strategy: send the same
frame on two paths simultaneously; the receiver delivers the first arrival and
discards the duplicate.

**Idempotent replay (U-C)** — The upstream loss recovery strategy: each frame
carries the last N keystrokes. The receiver deduplicates by sequence number.
A lost frame is self-healing.

**Split horizon** — The router forwarding rule that prevents frames from being
forwarded back toward the interface they arrived on. Prevents routing loops.

**Frame checksum** — A checksum computed at the intake E/PE router. Used for
duplicate suppression in multi-path topologies; retransmits with new content
produce different checksums.

**Drain** — A graceful router shutdown signal. The router notifies connected
nodes before disconnecting, allowing them to migrate to alternate routers.
The mechanism that enables rolling updates without dropping sessions.

---

## Admission and Security Terms

**Tier 1 admission** — SVTN-level access control. A node proves possession of
a registered private key via signed challenge. Grants access to the SVTN.
Enforced by routers.

**Tier 2 session authorization** — Session-level access control. A console
proves its public key is in the access node's authorization list for a specific
session. Enforced by the access node. Independent from Tier 1.

**Admission key** — An OpenSSH public key registered against an SVTN with a
role (control, console, access). Tier 1.

**Session authorization key** — An OpenSSH public key registered on an access
node per session with an access mode (full/read-only). Tier 2.

**Carrier-grade content separation** — The property that routers can route
intelligently (using identity, addressing, traffic patterns) but cannot read
or inject session content. Borrowed from telecom; the router provides
infrastructure, the customer holds the data keys.

---

## Observability Terms

**Quality indicator** — The per-session green/yellow/red status displayed by
the console. Derived from measured path latency and loss. Green = within
budget; yellow = degraded but functional; red = significantly degraded.

**Degradation signal** — A flag in a frame (TLPKTDROP or equivalent) that
tells the console the network could not deliver within the latency budget.

**Keep-alive** — A periodic frame sent on each path to measure RTT and detect
path liveness. An empty-tick frame serves double duty as a keep-alive.

---

## CLI and Tooling Terms

**sbctl** — The unified operator CLI for all Switchboard daemons. Not a daemon
itself. Authenticates via OpenSSH keys. The canonical management interface.

**`sbctl svtn`** — The canonical subcommand namespace for SVTN operations.
`sbctl net` is an accepted alias; documentation leads with `svtn`.

---

## Naming Notes (Open)

The abbreviation conflict (Console Node = SCN, Control Node = SCN) noted in
the naming parking lot has not been resolved at the domain level. Abbreviations
for daemon binary names are an implementation decision deferred to architecture
(PRD §"Binary Naming (Open)"). The full names — access node, console, control
node — are canonical and unambiguous.
