---
artifact_id: L2-invariants
document_type: domain-spec-section
level: L2
section: invariants
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
  - elem-node-router-architecture
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Domain Invariants

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

Domain invariants are business rules that must hold at all times in any
correct deployment. Violation of any invariant is a system defect, not a
degraded-mode behavior. Each invariant has exactly one interpretation.

---

## Content Separation Invariants

**DI-001 — Carrier-grade content separation**
A router, at any privilege level, cannot read, modify, or inject the payload
of any terminal session it forwards. The router sees only: outer header fields
(version, frame type, SVTN ID, destination address, source address, length,
HMAC). Session content is SSH-encrypted end-to-end between the originating
node and the destination node; no router holds or can derive the session keys.
_Grounded in: PRD §"Network Security Model" design invariant; Brief §"Carrier-grade content separation"; elem-ssh-end-to-end-encryption._

**DI-002 — Node private keys never transit the network**
A node's private SSH key is never serialized into any wire-format message,
frame, log entry, or diagnostic output. Public keys transit as required for
admission challenges and membership propagation. The invariant holds even
under diagnostic modes or error states.
_Grounded in: PRD FR39; Domain-Specific §"Cryptographic Standards."_

**DI-003 — Router compromise degrades availability, not confidentiality**
When a router is compromised, the attacker gains the ability to: drop frames
(availability), delay frames (quality), observe who communicates with whom
and when (traffic analysis), observe frame counts and sizes. The attacker
cannot gain: session content, session keystrokes, the ability to inject
content that a legitimate node will accept as authentic. This is a provable
property, not a policy claim.
_Grounded in: PRD §"Security Success"; Brief §"Carrier-grade content separation." Note: traffic analysis is within the operator's reach by design — the invariant covers content, not metadata._

---

## Network Architecture Invariants

**DI-004 — No direct node-to-node communication**
All traffic between nodes passes through at least one router. A node has no
mechanism to discover or contact another node's network address directly.
Admission control is enforced at the router; bypassing the router bypasses
admission control. This is the architectural invariant that makes SVTN
isolation meaningful.
_Grounded in: PRD §"Network Security Model" (no direct node-to-node); elem-node-router-architecture._

**DI-005 — SVTN cryptographic isolation**
A node admitted to SVTN-A cannot see traffic on SVTN-B, even when both SVTNs
are served by the same physical router infrastructure. Cross-SVTN visibility
requires possession of keys registered against both SVTNs. There is no
administrative override that bypasses this.
_Grounded in: PRD FR37 (content opaque), NFR §Security "SVTN cryptographic isolation."_

**DI-006 — HMAC frame authentication at first router**
Every frame carrying SVTN-scoped traffic is verified against the admitted key
set by the first router that receives it. Frames from non-admitted sources are
rejected before forwarding, not after. A frame that passes HMAC verification
was originated by a node holding a private key registered against that SVTN.
_Grounded in: PRD FR36; CAP-020._

---

## Protocol Invariants

**DI-007 — Outer header format stability within major version**
The 44-byte outer header layout is fixed within a major protocol version.
Routers parse the outer header at line rate; any change to field positions or
sizes requires a major version increment and is a breaking change. Extensions
to the channel header (TLV) do not require router upgrades.
_Grounded in: PRD §"Wire Protocol Constraints"; Morphological Parameter 10 (EXT-D)._

**DI-008 — Timeslice clock fires whether or not there is data**
The timeslice clock for each half-channel fires on every tick, even when no
application data is pending. An empty frame is a valid liveness signal. The
absence of a frame where one was expected is a degradation signal. Any
implementation that skips empty ticks breaks the liveness detection mechanism.
_Grounded in: elem-timeslice-framing; PRD §"Innovation & Novel Patterns" (validation section); PRD FR42._

**DI-009 — Receiver deduplication: first arrival wins**
When a frame is received on multiple paths simultaneously or in close
succession, the first-arriving copy is delivered and all subsequent copies of
the same frame (same checksum, same sequence number) are silently discarded.
Retransmits carry different checksums (new content with old sequence) and are
not suppressed.
_Grounded in: PRD FR10; CAP-010 duplicate suppression._

---

## Session and Key Invariants

**DI-010 — Session authorization is access-node-enforced**
The access node, not the router, enforces Tier 2 session authorization. A
console admitted to the SVTN (Tier 1) cannot attach to a session unless its
public key is in that session's authorization list. The router has no
knowledge of per-session authorization; it forwards based on addressing only.
_Grounded in: PRD FR26; CAP-018._

**DI-011 — Role separation between Tier 1 and Tier 2 keys**
A Tier 1 admission key grants access to the SVTN as a network participant. A
Tier 2 session authorization key grants access to a specific session on a
specific access node. The same keypair may serve both roles, but the
authorization scopes are independent. Revoking a Tier 1 key removes the node
from the network; revoking a Tier 2 key removes access to a specific session
without affecting SVTN membership.
_Grounded in: Morphological Parameter 5 (two-tier key model); PRD FR33–FR34._

**DI-012 — Control node is a network participant, not a router manager**
The control node manages SVTN lifecycle and key registration as a node on the
network. It does not have privileged access to router internal state,
forwarding tables, or router management APIs. The three planes (user/data,
router control, router management) are separate; the control node operates
only in the user/data plane.
_Grounded in: Morphological Parameter 12 §"Critical Correction"; PRD §"Control Node as Daemon."_

---

## Open Questions (not invariants — uncertainty requiring resolution)

These items were flagged as open in the brainstorming and PRD. They are
documented here to prevent assumptions from hardening into undocumented design
choices.

**OQ-001** — Console key registration: Can a console node register new Tier 1
keys, or only revoke/edit/expire existing ones? (Morphological Parameter 12
open question 1)

**OQ-002** — Access node key management: Do access nodes have any key
management capability, or is key management exclusive to control and console
nodes? (Morphological Parameter 12 open question 2)

**OQ-003** — Key permission hierarchy: Is there a permission hierarchy among
key roles? Can a console-role key revoke a control-role key? (Morphological
Parameter 12 open question 3)

**OQ-004** — Downstream switchover on path failover: When a node fails over
to an alternate router mid-session, how does the downstream half-channel
maintain state continuity? Upstream is covered by idempotent replay (U-C);
downstream behavior depends on the chosen downstream strategy (D-A vs D-CE).
(Morphological Parameter 9, IF-D)
