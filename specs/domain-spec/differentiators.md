---
artifact_id: L2-differentiators
document_type: domain-spec-section
level: L2
section: differentiators
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
  - elem-single-binary-three-modes
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Competitive Differentiators

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

Each differentiator from the product brief is traced to the capabilities
that realize it. A differentiator without a CAP-NNN trace is marketing, not
a domain property.

---

## D-1: New category — session network

**Claim:** Not a better tunnel, not a smarter relay, not an application hack.
A purpose-built network whose only job is terminal sessions. Multi-tenant,
multi-session, multi-node by design.

**Realizing capabilities:**
- CAP-001 (timeslice framing — framing model built for terminal sessions)
- CAP-002 (asymmetric half-channels — separate paths for keystrokes vs output)
- CAP-013 (tmux session publishing — native tmux integration)
- CAP-016 (multi-console fan-out — multi-session by design)
- CAP-017 / CAP-018 (two-tier key admission — multi-tenant by design)

**What competitors lack:**
SSH, Mosh, Eternal Terminal are point-to-point. tmate/Upterm are
relay-based with single-relay selection. Tailscale/Nebula are general-purpose
IP tunnels. None is a multi-tenant session network.

---

## D-2: Carrier-grade content separation

**Claim:** The network operator sees identity and traffic patterns; the
customer controls session content. Routers route intelligently but cannot
read or inject payload.

**Realizing capabilities:**
- CAP-003 (frame envelope — outer header router-visible, channel header opaque)
- CAP-020 (HMAC authentication — admission proof without content inspection)
- DI-001 (invariant: carrier-grade content separation is provable)
- DI-003 (invariant: router compromise → availability/quality, not content)

**What competitors lack:**
tmate/Upterm relays see session content. VibeTunnel/CloudeCode web relays
see content. Slack/Discord bots expose all commands/output to platform.
Tailscale DERP relays may see traffic depending on configuration. Only
WireGuard-based overlays (Tailscale, Nebula) provide comparable separation,
but they are general-purpose IP tunnels.

---

## D-3: Terminal-native optimization

**Claim:** Every design decision is tuned for keystroke-to-echo latency, not
general throughput. Asymmetric half-channels, content-type-aware loss recovery,
degradation signaling — built for how terminals actually work.

**Realizing capabilities:**
- CAP-001 (timeslice framing at terminal-native tick rates)
- CAP-002 (asymmetric half-channels — separate optimization per direction)
- CAP-007 (upstream idempotent replay — loss recovery tuned for keystrokes)
- CAP-008 (downstream ARQ with TLPKTDROP — tuned for terminal output)
- CAP-021 (quality indicator — degradation visibility)

**What competitors lack:**
Tailscale, Nebula, ZeroTier optimize for general IP traffic. SSH and Mosh are
stream-oriented without framing intelligence. No competitor implements
timeslice framing or content-type-aware recovery.

---

## D-4: The illusion of local

**Claim:** Remote sessions feel like local sessions. The network disappears
when working; it is honest when degrading.

**Realizing capabilities:**
- CAP-004 (session continuity across network transitions)
- CAP-005 (dual-path forwarding — resilience against single-path failure)
- CAP-021 (quality indicator — honest degradation, not mysterious freezes)
- CAP-027 (graceful drain — planned maintenance without session drops)

**What competitors lack:**
SSH freezes on network transition. Mosh improves reconnection but is
single-path. Eternal Terminal provides reconnection but single-path. No
competitor provides real-time quality indicators that distinguish network
problems from application problems.

---

## D-5: Progressive complexity

**Claim:** Two machines, one user, five minutes. Complexity is available but
never required. The E router MVP covers 85% of use cases.

**Realizing capabilities:**
- CAP-026 (E-to-PE graduation — same binary, different config)
- elem-single-binary-three-modes (one binary, three deployment modes)
- CAP-024 (unified CLI — single tool for all management operations)
- elem-mvp-scope-single-lan (MVP is deliberately narrow)

**What competitors lack:**
Tailscale requires a coordination server (vendor or self-hosted). Nebula
requires a lighthouse. ZeroTier requires a planet/moon infrastructure.
Switchboard's E router requires no external infrastructure: one binary, one
machine, five minutes.

---

## D-6: Open source, sovereign infrastructure

**Claim:** Run it on your infrastructure. Audit the code. No vendor lock-in,
no phone-home, no rug-pull. Your sessions, your network, your keys.

**Realizing capabilities:**
- DI-002 (private keys never transit the network)
- PRD §"What We Don't Measure" (no usage telemetry without consent)
- CAP-019 (key lifecycle locally controlled)
- Deployment model: single static binary, no proprietary service dependency

**What competitors lack:**
tmate, VibeTunnel, CloudeCode, HappyCoder, Dispatch/Remote Control require
vendor infrastructure. Tailscale requires Tailscale's coordination server
unless self-hosted (which is a recent option, not the default). Only
WireGuard-based self-hosted configurations and Nebula provide comparable
sovereignty — but neither is purpose-built for terminal sessions.

---

## Standards Heritage (Constraint, Not Differentiator)

The PRD's §"Standards and Protocol Heritage" is listed here as a constraint
that shapes capabilities rather than as a differentiator:

| Standard | Constraint |
|----------|-----------|
| OpenSSH key format | Credential system; interoperable with existing SSH infrastructure |
| Noise Protocol Framework | Router-to-router auth; no PKI overhead |
| QUIC (RFC 9000/9002) | Retransmit model for downstream ARQ |
| X.25 LAP-B | Sliding window + SREJ inspiration |
| SRT | TLPKTDROP for overdue frame handling |
| WebRTC (RFC 8854) | Four-tier recovery hierarchy shape |
| MPTCP (RFC 8684) | Lowest-RTT-first path selection |

These are constraints, not innovations. The brief is explicit:
"Multi-path forwarding, cryptographic admission, overlay routing with
latency-aware path selection, SSH as E2E trust layer — what's good execution,
not innovation." Differentiators D-1 and D-3 are the genuine novelty.
