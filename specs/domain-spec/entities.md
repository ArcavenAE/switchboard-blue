---
artifact_id: L2-entities
document_type: domain-spec-section
level: L2
section: entities
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
  - elem-node-router-architecture
  - elem-single-binary-three-modes
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Domain Entities

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

---

## Node Types

### Access Node
A daemon that runs on a machine hosting tmux sessions. Connects to one or
more routers. Publishes local tmux sessions over the SVTN using tmux control
mode (`-CC`) with PTY fallback. Maintains a Tier 2 authorization list (console
public keys per session). Never communicates directly with consoles — all
traffic passes through a router.

Key attributes: node identity keypair (OpenSSH), router connection set,
published session list, authorized console key list per session.

### Console
A daemon that discovers and attaches to remote sessions published by access
nodes. Receives downstream terminal output; sends upstream keystrokes (or
neither, in read-only mode). Remotely controllable via `sbctl`. May hold a
full-access or read-only key.

Key attributes: node identity keypair, router connection set, attached session
reference (0 or 1), access mode (full/read-only).

### Control Node
A daemon that manages SVTN lifecycle and key registration. Connects to the
SVTN as a network participant. Registers, revokes, and expires keys. Submits
key changes as signed network service requests; the router distributed
database handles propagation.

Key attributes: node identity keypair, SVTN(s) managed, key registry view.

---

## Router Types

### E Router (Edge-local)
A router instance with no upstream router connections configured. Serves
nodes on a single LAN. The simplest deployment mode — two machines, five
minutes. Graduates to PE by adding upstream router config.

### PE Router (Provider Edge)
A router instance with both node-facing and router-facing interfaces. Handles
SVTN admission, HMAC verification, keep-alive/latency probing, and inter-router
link-state exchange. The production router for multi-site topologies.

### P Router (Provider Core) — Theoretical
A router instance with router-facing interfaces only; no node protocol. Pure
frame forwarding based on labels. Not built until simulation or production
data proves need. Defined here because the binary supports the mode.

_Note: E and PE share the same binary. P shares the same codebase but may
be a distinct build target (PRD FR59). See elem-single-binary-three-modes._

---

## Network Constructs

### SVTN (Switched Virtual Terminal Network)
A cryptographically isolated virtual network for terminal sessions. Identified
by an SVTN ID (16-byte (128-bit) identifier). Admission is key-based. Multiple SVTNs
can coexist on the same router infrastructure without cross-SVTN visibility.

Key attributes: SVTN ID, control key set, admitted node keys (with roles),
per-class fanout policy (access-originated / console-originated).

### Channel
An active end-to-end tmux session connection between one console and one
access node, carried over the SVTN. Consists of two half-channels.

### Half-Channel (Upstream)
The keystroke path from console to access node. Independent timeslice clock,
sequence space, and recovery strategy (idempotent replay / U-C sliding
window). Tiny payloads. Ordered and loss-intolerant.

### Half-Channel (Downstream)
The terminal output path from access node to console. Independent timeslice
clock, sequence space, and recovery strategy (reliable ordered stream / ARQ
with TLPKTDROP). Bursty. Content-type-aware in post-MVP (D-CE).

---

## Frame and Key Entities

### Frame (Outer)
The network-visible unit of transmission. 44-byte outer header (version,
frame type, SVTN ID, destination address, source address, length, HMAC) plus
a channel header (endpoint-only). Router parses the outer header only.

### Frame (Channel Header)
The endpoint-visible inner header: channel ID, sequence number, flags
(FEC_present, ARQ_req, SACK_present). Followed by
SSH-encrypted payload. Opaque to routers.

### Node Address
An 8-byte SVTN-scoped hash of `hash(SVTN-ID || public-key)`. Self-derived from
the node's cryptographic identity. No assignment authority required.

### Admission Key (Tier 1)
An OpenSSH public key registered against an SVTN with a role designation
(control, console, access). Possession of the corresponding private key grants
SVTN admission via signed challenge.

### Session Authorization Key (Tier 2)
An OpenSSH public key registered on an access node per session, with an
access mode (full/read-only). Grants a console permission to attach to that
session (or observe read-only). Distinct from the Tier 1 admission key, though
the same keypair may serve both roles.

---

## Entity Relationships

```
SVTN  1──* AdmissionKey (role: control | console | access)
SVTN  1──* Node (admitted members)
Node  1──1 IdentityKeyPair

AccessNode  1──* TmuxSession (published)
TmuxSession 1──* SessionAuthKey (access mode: full | read-only)

Channel  ──  AccessNode (one end)
Channel  ──  Console    (other end)
Channel  1──2 HalfChannel (upstream + downstream)

Router   1──* NodeConnection
Router   1──* RouterPeerConnection (PE/P only)
```

Constraint: All channels are mediated by at least one router. No direct
node-to-node connections exist. This is enforced by the network architecture,
not by node policy (elem-node-router-architecture).
