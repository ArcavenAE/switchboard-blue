---
artifact_id: L2-capabilities
document_type: domain-spec-section
level: L2
section: capabilities
version: "1.0"
status: draft
producer: business-analyst
timestamp: 2026-06-23T00:00:00
modified: ["2026-06-23"]
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

# Domain Capabilities

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

Capabilities are grouped by subsystem. Each CAP-NNN is grounded in the
product brief or PRD with an explicit anchor justification.

---

## Subsystem: session-networking (sn)

**CAP-001** — Timeslice-driven frame assembly and transmission (P0)
Each half-channel assembles frames on a fixed clock tick; the frame departs
whether full or empty. The upstream and downstream clocks are independent.
_Anchor: Brief §"Session-native primitives" + First Principle #3 (elem-timeslice-framing). CAP-001 covers the core framing clock because all latency guarantees depend on it._

**CAP-002** — Asymmetric half-channel operation (P0)
Upstream (keystrokes) and downstream (terminal output) operate as independent
half-channels with independent sequence spaces, clocks, and recovery strategies.
_Anchor: Brief §"Session-native primitives"; elem-asymmetric-half-channels. CAP-002 covers the structural asymmetry because upstream and downstream have fundamentally different loss semantics._

**CAP-003** — Frame envelope encoding and decoding (P0)
Each frame carries a 44-byte outer header (version, frame type, SVTN ID,
destination, source, length, HMAC) plus a channel header (channel ID,
sequence, flags). Router parses outer; endpoints
parse channel header.
_Anchor: PRD §"Wire Protocol Constraints" + Morphological Parameter 10. CAP-003 covers the wire format because carrier-grade separation requires a defined router-visible / endpoint-only boundary._

**CAP-004** — Session continuity across network transitions (P0)
Nodes maintain sessions when the underlying IP address changes (wifi handoff,
roaming). Identity is cryptographic, not network-address-based.
_Anchor: PRD FR4–FR5; Brief §"Multi-path resilience." CAP-004 covers survivability because it is the primary differentiator over raw SSH._

---

## Subsystem: multipath-forwarding (mf)

**CAP-005** — Dual-path frame forwarding with duplicate-and-race (P0)
A node sends the same frame on its two fastest connected router paths
simultaneously; the receiver keeps the first arrival and discards duplicates.
_Anchor: elem-dual-fastest-path-forwarding; PRD FR11–FR12. CAP-005 covers multi-path dispatch because it is the core resilience mechanism._

**CAP-006** — Latency-based path selection and ranking (P0)
Nodes track per-path RTT and loss via keep-alive probes and rank connected
routers accordingly. The top-2 paths by quality receive outbound frames.
_Anchor: PRD FR12–FR13; Brainstorming Parameter 9 (PS-D). CAP-006 covers path ranking because "fastest path" must be defined and measured._

**CAP-007** — Upstream idempotent replay (U-C sliding window) (P0)
Each upstream frame carries the last N keystrokes as a replay window.
The receiver deduplicates; loss in transit is self-healing.
_Anchor: PRD FR7; Morphological Parameter 6 (U-C). CAP-007 covers upstream loss recovery because keystrokes are ordered and loss-intolerant._

**CAP-008** — Downstream reliable ordered delivery with ARQ (P0)
The downstream half-channel delivers terminal output with piggybacked ACK
and SACK bitmap. ARQ retransmits on detected gaps using new frames carrying
old content (QUIC model). TLPKTDROP terminates overdue frames with
degradation signal.
_Anchor: PRD FR8–FR9; Morphological Parameter 6 (D-A MVP). CAP-008 covers downstream recovery because output correctness is required for readable terminal state._

**CAP-009** — XOR parity FEC for burst-loss recovery (P1)
A parity frame covers a group of data frames; a single loss within the group
is recoverable without retransmit. Activates in multi-path topologies.
_Anchor: PRD §"Multi-Path and Multi-Hop" (deferred from MVP). CAP-009 covers FEC because it is the post-MVP upgrade to CAP-008 for multi-hop topologies._

**CAP-010** — Router split-horizon and duplicate suppression (P0)
Routers do not forward frames back toward their arrival direction. A bounded
drop cache of frame checksums prevents loops. Retransmits produce different
checksums and pass through.
_Anchor: PRD FR17–FR18. CAP-010 covers loop prevention because multi-path forwarding requires it._

---

## Subsystem: session-discovery (sd)

**CAP-011** — Multicast presence advertisement (P1)
Access nodes advertise available sessions, attachment status, and quality
indicators to all nodes on the SVTN via an SVTN-scoped multicast address.
Consoles advertise their own presence. Triggered by state change, periodic
heartbeat, and on-demand request.
_Anchor: PRD FR20–FR23; Morphological Parameter 4. CAP-011 covers presence because session discovery without hostnames is a core differentiator._

**CAP-012** — Console session enumeration across SVTN (P1)
A console discovers all available sessions across all access nodes on its
SVTN without specifying IP addresses or hostnames. Metadata per session
includes name, attachment status, and quality indicator.
_Anchor: PRD FR21–FR22; User Journey §Kai. CAP-012 covers fleet-level discovery because it enables Kai's 40-session view._

---

## Subsystem: session-access (sa)

**CAP-013** — Access node tmux session publishing (P0)
An access node connects to local tmux via control mode (`-CC`) and publishes
available sessions over the SVTN. PTY fallback used when control mode is
unavailable.
_Anchor: PRD FR1; Domain-Specific §"Session Substrate." CAP-013 covers publishing because it is the source of all session traffic._

**CAP-014** — Console session attach and detach (P0)
A console attaches to a remote session by selecting it by name, not by
specifying a host. It receives the downstream output stream and sends
upstream keystrokes. Detach releases the session without closing it.
_Anchor: PRD FR24–FR25; User Journey §Devon. CAP-014 covers attach/detach because it is the primary operator interaction._

**CAP-015** — Read-only session access mode (P0)
A console holding a read-only key receives the downstream output stream but
its upstream channel is rejected at the access node. Scope may be per-session,
per-access-node, or per-SVTN.
_Anchor: PRD FR27–FR28; User Journey §Priya, §Team Lead. CAP-015 covers read-only access because it enables session sharing without credential sharing._

**CAP-016** — Simultaneous multi-console session viewing (P0)
Two or more consoles may subscribe to the same session output simultaneously.
The access node delivers output once per frame; the router fans out to all
subscribed consoles.
_Anchor: PRD FR29–FR31. CAP-016 covers fan-out because it is required for read-only observer use cases._

---

## Subsystem: admission-security (as)

**CAP-017** — SVTN admission via signed key challenge (Tier 1) (P0)
A node joins an SVTN by proving possession of a private key whose public key
is registered with the SVTN. The router verifies the signed challenge and
grants or denies admission.
_Anchor: PRD FR35; Brief §"Virtual switched networks." CAP-017 covers Tier 1 admission because it is the network entry gate._

**CAP-018** — Per-session access authorization (Tier 2) (P0)
The access node maintains an authorized console key list per session. Before
forwarding a console's upstream, it checks the console's public key against
the session's authorization list.
_Anchor: PRD FR26; Morphological Parameter 5. CAP-018 covers Tier 2 because session-level control is separate from network-level admission._

**CAP-019** — Key lifecycle management (register, revoke, expire) (P0)
Control nodes and admitted console nodes can register, revoke, and expire
public keys against an SVTN with role designation (control, console, access).
Key changes propagate via the router's distributed database.
_Anchor: PRD FR33–FR34; Morphological Parameter 12. CAP-019 covers key lifecycle because admission is only as strong as its revocation path._

**CAP-020** — HMAC frame authentication at router boundary (P0)
Every frame carries an HMAC in the outer envelope, computed by the sending
node. The first router verifies and rejects frames from non-admitted sources
before forwarding.
_Anchor: PRD FR36; Domain-Specific §"Cryptographic Standards." CAP-020 covers per-frame auth because it enforces the SVTN trust boundary at the wire level. Realized by BC-2.05.005._

**CAP-020a** — Private key non-transit (P0)
Node private keys never appear on the wire. Only their derived HMAC tags do.
Private material stays on the node that owns it. All network authentication
is accomplished by signature output and HMAC tags, never by transmitting the
key that produced them.
_Anchor: DI-002 (node private keys never transit the network), grounded in PRD FR39 and Brief §"Cryptographic Standards." CAP-020a covers private key non-transit because it is the key management invariant that underlies the HMAC trust model — without this guarantee, CAP-020's HMAC verification provides no security. Realized by BC-2.05.007._

**CAP-020b** — SVTN cryptographic isolation (P0)
Frames produced under SVTN-A's HMAC keying cannot validate as frames under
SVTN-B's keying. Cross-SVTN replay or substitution attacks fail at the router
boundary because HMAC keys are scoped per (node, SVTN) pair.
_Anchor: DI-005 (SVTN cryptographic isolation), grounded in PRD FR36 and Brief §"Virtual switched networks." CAP-020b covers SVTN cryptographic isolation because the multi-tenancy security guarantee depends on HMAC keys being non-transferable across SVTN boundaries — a node admitted to SVTN-A cannot forge valid frames for SVTN-B. Realized by BC-2.05.006._

---

## Subsystem: quality-observability (qo)

**CAP-021** — Per-session quality indicator (green/yellow/red) (P1)
The console displays a quality indicator per session, derived from measured
path latency and loss. Empty-tick frames serve as liveness probes; a missing
frame is a degradation signal.
_Anchor: PRD FR41–FR42; User Journey §Kai ("yellow indicator"). CAP-021 covers the quality signal because it is the primary operator feedback mechanism._

**CAP-022** — Per-path latency and loss metrics via CLI (P1)
Operators query per-path RTT and loss metrics via `sbctl`. Both node-side
(router connection quality) and network-operator-side (forwarding metrics)
views are available.
_Anchor: PRD FR43; User Journey §Marcus, §Troubleshooter. CAP-022 covers diagnostic metrics because operators must distinguish network problems from application problems._

---

## Subsystem: network-management (nm)

**CAP-023** — SVTN lifecycle management (create, destroy) (P2)
A control node creates and destroys SVTNs. The first control key is
bootstrapped locally on the E router; subsequent keys self-propagate.
_Anchor: PRD FR32, FR46. CAP-023 covers lifecycle because SVTN creation precedes all other operations._

**CAP-024** — Unified CLI operator interface (sbctl) (P0)
`sbctl` is the single operator CLI for all four daemon types (router, access,
console, control). It authenticates via OpenSSH keys and exposes router status,
SVTN management, key management, session operations, console control, and
diagnostics.
_Anchor: PRD §"CLI Commands (sbctl)." CAP-024 covers the CLI because all management surfaces are accessible without additional tooling._

---

## Subsystem: console-operations (co)

**CAP-025** — Remote console control plane (P1)
A console is remotely controllable via `sbctl`: attach, detach, switch session,
navigate. The viewing operator and the controlling operator may be different
principals.
_Anchor: PRD FR54–FR55; Domain-Specific §"Console Control Plane." CAP-025 covers the control plane because programmatic console driving is required for supervisor use cases._

---

## Subsystem: deployment-operations (do)

**CAP-026** — E-to-PE router graduation (P2)
An E router graduates to PE mode by adding upstream router connections in
its config file. Same binary, no reinstall, no rearchitecture.
_Anchor: PRD FR58; elem-single-binary-three-modes; User Journey §Marcus. CAP-026 covers graduation because it is the progressive-deployment promise._

**CAP-027** — Graceful router drain and session migration (P2)
A router signals impending shutdown to connected nodes. Nodes migrate to
alternate routers before the router disconnects. Enables rolling updates
without dropping active sessions.
_Anchor: PRD FR19; NFR §Reliability "Graceful router drain." CAP-027 covers drain because it prevents the "rolling update drops all sessions" death condition._

**CAP-028** — Daemon startup config validation (P0)
Daemons (router, access, console, control) validate their configuration at
startup. Malformed config produces a clear, actionable error message naming
the file, line, and field; the daemon exits non-zero before opening any
network sockets or accepting connections.
_Anchor: FM-010 (deployment misconfig). CAP-028 covers daemon startup config validation because a daemon that starts in a partially-configured state is a deployment correctness invariant — clean failure prevents silent misconfiguration from entering production. Subsystem: deployment-operations. Realized by: BC-2.09.003._
