---
stepsCompleted:
  - step-01-init
  - step-02-discovery
  - step-02b-vision
  - step-02c-executive-summary
  - step-03-success
  - step-04-journeys
  - step-05-domain
  - step-06-innovation
  - step-07-project-type
  - step-08-scoping
  - step-09-functional
  - step-10-nonfunctional
  - step-11-polish
  - step-12-complete
status: complete
classification:
  projectType: 'network-infrastructure'
  projectTypeDetail: 'Protocol implementation — CLI interfaces for users, service daemon for routing. TUI/GUI planned for control node and router management plane in future.'
  domain: 'networking-infrastructure'
  domainDetail: 'Terminal session networking — custom wire protocol, cryptographic admission, session-native framing'
  complexity: 'high'
  projectContext: 'greenfield'
inputDocuments:
  - '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
  - '_bmad-output/brainstorming/brainstorming-session-2026-03-07-001.md'
  - '_bmad-output/brainstorming/naming-node-type-parking-lot.md'
  - '_bmad-output/brainstorming/session-context-cache.md'
documentCounts:
  briefs: 1
  research: 0
  brainstorming: 3
  projectDocs: 0
workflowType: 'prd'
---

# Product Requirements Document - Switchboard

**Author:** maker
**Date:** 2026-04-04

## Executive Summary

Switchboard is a switched network purpose-built for terminal session networking. It provides multi-path, end-to-end encrypted access to remote tmux sessions over switched virtual terminal networks (SVTNs), optimized for interactive terminal latency.

Remote terminal sessions are load-bearing infrastructure. AI agent fleets run in tmux panes across machines worldwide. Platform engineers operate infrastructure from wherever they are. Existing tools address parts of this problem — transport (SSH, Mosh), overlay connectivity (Tailscale, Nebula), application-level access (tmate, browser proxies) — but Switchboard is purpose-built for terminal sessions.

Three node types (access nodes, consoles, control nodes) communicate through routers that provide latency-aware multi-path forwarding. SSH provides the end-to-end trust layer. Switchboard adds session-native framing, cryptographic network admission, degradation signaling, and carrier-grade content separation — routers see identity and traffic patterns but session content is opaque. When the network degrades, users see it clearly. When it can maintain quality, sessions feel local. Physics wins when physics wins, but the network is honest about it.

Four target users: Devon (solo operator, two machines), Kai (AI ops, 40+ agent sessions across 6 machines), Priya (platform engineer, multi-cloud, compliance-constrained), Marcus (network admin, skeptic who needs to see it work before believing it). Progressive deployment — same binary, different topology — scales from a single E router on a LAN to multi-hop provider networks without architecture changes.

Open source. Self-hosted. No phone-home, no vendor lock-in. Switchboard provides Switched Virtual Terminal Networks (SVTNs) — the network type name reflects what it is: switched (forwarding through routers), virtual (overlay), terminal (what it carries).

## What Makes This Special

Switchboard is a network designed specifically for terminal sessions.

- **Terminal-native framing.** Timeslice-driven, asymmetric half-channels (keystrokes up, output down), content-type-aware loss recovery. Built for how terminals actually work, not for general IP tunneling.
- **Carrier-grade content separation.** Routers authenticate nodes, enforce admission, and make forwarding decisions — but session content is encrypted end-to-end between nodes. The operator provides infrastructure; the customer holds the data keys.
- **Multi-path resilience.** Nodes maintain connections to multiple routers. Dual-path forwarding with a four-tier recovery cascade (duplication, FEC, selective reject, too-late-packet-drop). Failover in seconds, with clear signaling.
- **Progressive deployment.** E router on a LAN graduates to PE router with upstream connections. Same binary. No rearchitecture.
- **Degradation honesty.** Quality indicator visible to the user. Green, yellow, red. When the network can't deliver, it says so — no mysterious freezes.

## Project Classification

- **Type:** Network infrastructure / protocol implementation (CLI/server interfaces; TUI/GUI planned for control node and router management plane)
- **Domain:** Terminal session networking
- **Complexity:** High — custom wire protocol, cryptographic admission, real-time latency constraints, multi-path networking
- **Context:** Greenfield

## Success Criteria

### User Success

| Criterion | What Success Looks Like |
|-----------|------------------------|
| Session survivability | Sessions survive network transitions without user intervention |
| Keystroke-to-echo latency | Within human perception budget (see NFR Performance for specific targets) |
| Time to first session | Fast enough that a solo operator sets up and forgets in minutes |
| Path failover | Sessions resume on alternate path in seconds, not minutes |
| Session sharing | Read-only access via key; no credential sharing; no write capability |
| Degradation indicator accuracy | Indicator matches measured path quality |
| Session discovery | New sessions appear promptly after creation |

Specific measurable targets and test methods for each criterion are defined in Non-Functional Requirements.

### Operational Success

Testable assertions for the operator experience:

- **Attach to any session without specifying a hostname.** Console discovers sessions vian SVTN presence protocol. The operator selects a session, not a machine.
- **Survive network transitions without re-authenticating.** Move from wifi to LAN to mobile — sessions persist, no reconnection, no credential re-entry.
- **Discover all open sessions from a single console.** One command lists every session across every machine on the SVTN.
- **No manual router maintenance.** Routers run unattended on the internet. No periodic intervention required for normal operation.

### Security Success

Router compromise degrades availability or quality. It does not compromise confidentiality or integrity. This is the design invariant — the security model that makes shared infrastructure trustworthy.

Specific testable security requirements (content opacity, key isolation, HMAC authentication, SVTN isolation, console authorization) are defined in Non-Functional Requirements > Security. Design invariants (no direct node-to-node, cryptographic isolation model) are in Domain-Specific Requirements > Network Security Model.

### Death Conditions

The project has failed if:

1. UX breaks flow state — stutters, hangs, freezes that make the user notice the network
2. Session security is compromised by anything other than stolen SSH keys
3. Operational fragility — can't do rolling updates; a vulnerability requires taking the whole network down
4. Complexity barrier — a solo operator can't set up and forget in minutes

### Leading Indicators

| Death Condition | Indicator | Threshold |
|----------------|-----------|-----------|
| UX breaks flow state | p99 keystroke-to-echo latency | Creeping above 150ms |
| Complexity barrier | Onboarding step count | Getting-started guide exceeds 1 page or 3 commands |
| Complexity barrier | Binary dependency count | Requires anything beyond itself and tmux |
| Operational fragility | Single points of failure | Any component failure takes down all sessions |
| Operational fragility | Rolling update capability | Can't update a router without dropping all its sessions |
| Security compromise | Payload visibility at router | Any test showing router can access decrypted content |
| Security compromise | Outer header byte count | Growth beyond 44 bytes requires security justification |
| All | Dependency count | No proprietary service dependency, ever |

### What We Don't Measure

Switchboard is open-source software. Success is adoption and utility, not revenue.

- No revenue, MRR, conversion rates — this is free software
- No download counts — downloads don't mean usage
- No feature count — more features is not more success
- No usage telemetry without consent — no phone-home, ever

### Signals We Watch

Not success criteria — these are indicators of community health that we welcome but don't control.

| Signal | What It Means |
|--------|---------------|
| Issues filed by production users | People are using it for real work |
| PRs from non-maintainers | The project is approachable |
| Ecosystem packaging | Homebrew, distro repos, container images maintained by others |

## Product Scope

### E Router Release (single-hop, same LAN)

Proves the core: does a switched terminal network feel better than raw SSH? One person, two or more machines, one E router.

What ships: E router, access node, console, SVTN admission (Tier 1 key-based), session authorization (Tier 2 per-session access modes), edge protocol (timeslice framing, idempotent upstream replay, reliable downstream, piggybacked ACK+SACK), frame envelope (44-byte outer + channel header), degradation signaling, single-path operation, router status CLI (connected nodes, active SVTNs), admission status queries, diagnostic commands accessible via CLI.

**Design constraint:** E routers are LAN-only. Nodes must be routable to the E router. When a node leaves the local LAN, it cannot reach a local-only E router. Multi-path deployments require at least one internet-reachable PE router. E routers may become legacy once PE is widely deployed.

Delivers Devon's use case on the local network. Roaming across networks requires a PE router. Does not yet deliver maker's full use case (distributed agents across the internet require multi-hop).

### Multi-Path and Multi-Hop (PE routers, distributed topology)

Unlocked when: the second router connects. Same binary as E router, expanded topology.

What ships: dual-path forwarding, duplicate-and-race, XOR parity FEC, SREJ via alternate path, PE routers, link-state routing, distributed admission database, membership propagation, router-to-router Noise protocol, content-type-aware downstream (D-CE), console upgrades (CN-C local tmux mirroring, CN-A control mode consumer).

Delivers maker's use case, Kai's use case, Priya's use case, Marcus's multi-site use case.

### Global Topology (P routers, provider-scale)

Unlocked when: scale demands core-only forwarding nodes separate from PE functions.

What ships: P router binary (separate from E/PE; same codebase, different build target; router-facing interfaces only, pure forwarding, no node protocol). PE routers at cloud POPs. E/PE remains a single binary (E graduates to PE with upstream connections configured).

Delivers: any terminal professional, anywhere, connects to their sessions with the feeling of local.

## User Journeys

### Devon — Solo Operator

One person, two machines. A developer with a workstation at home and a build server in the closet or the cloud. Runs tmux on the remote machine, SSHes in from wherever.

**Opening scene:** SSH works until it doesn't. Home wifi hiccups, the session freezes, `~.` to kill the hung connection, reconnect, reattach tmux, find the right pane. Three times a day. The reconnection takes 30 seconds. Getting back to where you were in your head takes longer. Has tried Mosh — it helps with reconnection but the remote machine is behind NAT and Mosh's UDP doesn't punch through reliably. Has tried Eternal Terminal — same NAT problem, and it's another binary to maintain.

**Rising action:** Devon installs Switchboard on both machines. Runs an E router. Connects to the SVTN. Sees remote tmux sessions listed in the console. Attaches.

**Climax:** The wifi blips. The session stays connected. Devon doesn't notice. That's the point — the absence of disruption is the value.

**Resolution:** Devon stops thinking about SSH. Sessions are there when needed. The network is invisible.

**Success:** "I installed Switchboard on both machines. Took five minutes. My sessions don't drop when the wifi blips. I stopped thinking about SSH."

**Requirements revealed:** E router, access node, console, single-path edge protocol, session discovery via presence protocol, network transition survivability, ≤3 commands to first session.

---

### Kai — AI Operations Engineer

Manages a fleet of 40+ Claude Code agent sessions across 6 machines for a platform engineering team. Each agent runs in a tmux pane, supervised by coordinator agents that also run in tmux. Kai's day is spent monitoring agent work, intervening when agents get stuck, reviewing output, and restarting failed sessions.

**Opening scene:** SSH tunnels to each machine. When the home network hiccups, three sessions drop simultaneously and Kai loses 10 minutes reconnecting and re-finding the right panes. But the reconnection time isn't the real cost — it's the ambiguity. When a session feels sluggish, Kai can't tell if it's network degradation or an agent that's stuck. So Kai over-checks, manually polling sessions that are probably fine, because there's no signal saying otherwise. The anxiety tax is higher than the reconnection tax. Uses a patchwork of tmux, SSH, and a Discord bot to monitor.

**Rising action:** Kai connects a console to the SVTN. All 40+ sessions across 6 machines appear in the session list. Each session shows a quality indicator. Kai attaches to sessions by selecting them, not by SSHing to specific machines.

**Climax:** A yellow indicator appears on three sessions. Kai knows it's network — not agent — degradation. No need to check those sessions. A fourth session has a green indicator but no output. That's the one that needs intervention.

**Resolution:** Kai's monitoring is: look at indicators, intervene where green + silent. The anxiety is gone. The network tells Kai what it knows.

**Success:** "I see a yellow indicator — that means network, not agent. I don't need to check. When the indicator is green and the agent isn't producing output, that's when I intervene. Switchboard gave me trust in what I'm seeing."

**Requirements revealed:** Multi-machine session discovery, degradation signaling per session, quality indicators (green/yellow/red), multi-path resilience across multiple machines, session list from single console, read-only monitoring capability.

---

### Priya — Platform Engineer

Operates infrastructure across three cloud providers and a handful of bare-metal machines. Lives in tmux — 15-20 sessions open at any time. Runs ansible, kubectl, terraform from remote sessions. Needs to access the same sessions from her office, her home, and occasionally her phone.

**Opening scene:** SSH with Mosh for the flaky connections. Works most of the time but has no multi-path — when the primary route to the Singapore datacenter degrades, she waits or bounces through a jump host manually. No session quality visibility — "is it slow because of the network or because kubectl is thinking?" Cannot hand a session to a colleague without sharing SSH keys or setting up tmate. In her SOC2-audited environment, sharing credentials is a compliance violation — so she simply can't share sessions at all. Pair debugging means "I'll share my screen on Zoom."

**Rising action:** Priya sets up an SVTN for her infrastructure. Access nodes on each machine publish tmux sessions. She connects from office, home, phone — same sessions, no reconnection. When the Singapore route degrades, the session fails over to the backup path.

**Climax:** A colleague needs to see a specific kubectl session to debug a production issue. Priya issues a read-only key scoped to that one session. Colleague sees exactly what Priya sees. No credential sharing, no compliance violation, no Zoom screen share.

**Resolution:** Priya onboards new team members with keys, not credential sharing. Multi-path handles the flaky routes. Session sharing is a solved problem.

**Success:** "I hand a colleague a read-only key to one specific session. No credential sharing, no compliance violation, no Zoom screen share. They see exactly what I see. When I switch from office to home, the session doesn't even blink."

**Requirements revealed:** Multi-path failover, per-session read-only key issuance, session access modes (full, read-only, per-session scoped), network transition survivability across locations, SVTN admission with key-based access control, Tier 2 session authorization.

---

### Marcus — Network / Infrastructure Admin

Manages datacenter infrastructure, network gear, and monitoring systems. Needs persistent, reliable access to console sessions on remote equipment. Operates across sites connected by sometimes-unreliable WAN links. Deeply skeptical of adding another network layer on top of his carefully managed infrastructure.

**Opening scene:** SSH over VPN. When the VPN link degrades, sessions freeze. Uses Eternal Terminal for some hosts, raw SSH for others, a jump host topology he maintains by hand. No unified view. Has built his career on tmux and expects tools to respect that. His first reaction to Switchboard: "I don't need another overlay network."

**Rising action:** Marcus installs an E router between his laptop and the console server. No infrastructure changes. Same LAN, two machines. He watches. Sessions don't drop when the wifi blips. He adds a second site — a PE router with an upstream connection.

**Climax:** The WAN link to the second site fails over to the backup path automatically. Marcus finds out from the degradation indicator, not from a frozen screen. He inspects the router — it can't see what he's typing into those console sessions. Carrier-grade separation, verified.

**Resolution:** Marcus's E router graduated to a PE router connected to the wider network. Progressive deployment, no rearchitecture. He trusts it because he tested it, not because someone told him to.

**Success:** "I started with the E router between my laptop and the console server. No infrastructure changes. When I added a second site, the sessions failed over to the backup path automatically. I see link quality in real time. And the router infrastructure can't read what I'm typing into those console sessions."

**Requirements revealed:** Progressive deployment (E to PE), zero infrastructure disruption on install, WAN failover, degradation indicators with link quality, carrier-grade content separation verifiable by the operator, trust-building through incremental adoption.

---

### Team Lead — Read-Only Observer

Kai's team lead needs visibility into the agent fleet during an incident without interrupting Kai.

**Opening scene:** Three agent sessions have stalled simultaneously. Kai is diagnosing. The team lead needs to know: is this a network event or an agent problem? Is Kai handling it? What's the blast radius? In a distributed team, the only option today is asking Kai on Slack mid-incident.

**Rising action:** Kai previously issued a read-only key to the team lead, scoped to the fleet's SVTN. The team lead opens a console and sees Kai's 40+ sessions with quality indicators.

**Climax:** During an incident, the team lead sees three sessions yellow (network), Kai attached to a fourth green-but-silent session (agent stuck). The situation is visible without a single interrupting message.

**Resolution:** The team lead reports status to stakeholders from direct observation, not secondhand. Kai works uninterrupted.

**Requirements revealed:** Read-only access mode, per-SVTN key scoping, session quality indicators visible to read-only consoles, session list showing attachment status (who is attached where).

---

### The Operator as Admin — Setup and Key Management

The same person who uses Switchboard also administers it. In most deployments there is no separate admin role.

**Opening scene:** Devon (or Kai, or Priya, or Marcus) decides to set up Switchboard. They need to: install the binary, start a router, create an SVTN, register keys, and connect nodes.

**Rising action:** Install binary. Start E router. Create an SVTN — the first control key is bootstrapped locally (a file on the E router). Register node keys against the SVTN. Start access node on the remote machine. Start console on the local machine.

**Climax:** It works. Sessions appear. The setup took less than five minutes and fewer than three commands per machine.

**Key management:** Keys are managed in a local file on the E router for the initial release. Key registration, revocation, and expiry are CLI operations. When multi-router deployments arrive, the distributed admission database handles propagation. Console nodes can revoke/edit/expire keys for SVTNs they have keys for.

**Resolution:** The admin task is done. The operator doesn't think about it again until they need to add a machine or revoke a key.

**Requirements revealed:** CLI-based SVTN creation, local key file management, key registration/revocation/expiry via CLI, ≤3 commands per machine, ≤5 minutes total setup, key management operations available to console nodes (not just control nodes).

---

### The Troubleshooter — Diagnosing Problems

When something isn't working, the operator diagnoses it themselves. There is no separate support role.

**Opening scene:** A session feels sluggish, or a console can't see a remote session, or a node can't join the SVTN. The operator needs to figure out why.

**Rising action:** The operator checks the degradation indicator — is this a network problem or something else? If yellow/red: the network is degraded, and the indicator shows measured latency and loss. If green but slow: the problem is on the remote machine, not the network. If the console can't see a session: check whether the access node is connected to the router, whether the session is published, whether SVTN admission succeeded.

**Diagnostic tools:** All diagnostics are CLI commands on the router and node binaries. No log parsing, no additional tooling, no debugger attachment required.

- Router status (connected nodes, active SVTNs)
- Admission status (is this key admitted to this SVTN?)
- Session list (which sessions are visible, which machines they're on)
- Per-session quality indicator (green/yellow/red)
- Per-path latency and loss metrics (keep-alive measurements)

**Climax:** The operator identifies the problem from the signals Switchboard provides. The network is a fact, not a mystery — Value #3.

**Resolution:** The problem is either a network issue (Switchboard shows it) or not a network issue (Switchboard proves it isn't). Either way, the operator isn't guessing.

**Requirements revealed:** Router status CLI, admission status queries, per-session quality indicators, per-path latency/loss metrics, clear diagnostic output that distinguishes network problems from application problems. All accessible via CLI without additional tooling.

---

### Journey Requirements Summary

| Capability | Devon | Kai | Priya | Marcus | Team Lead | Admin | Troubleshooter |
|-----------|-------|-----|-------|--------|-----------|-------|----------------|
| E router, access node, console | x | x | x | x | | x | |
| Session discovery via presence | x | x | x | x | x | | x |
| Network transition survivability | x | x | x | x | | | |
| Degradation signaling (G/Y/R) | | x | x | x | x | | x |
| Multi-path failover | | x | x | x | | | |
| Read-only session access | | | x | | x | | |
| Per-session key scoping | | | x | | x | x | |
| SVTN admission (Tier 1) | x | x | x | x | x | x | x |
| Session auth (Tier 2) | | | x | | x | x | |
| CLI key management | | | x | | | x | |
| Progressive deployment (E→PE) | | | | x | | x | |
| Router status / diagnostics | | | | | | | x |
| Per-path latency/loss metrics | | | | | | | x |
| Admission status queries | | | | | | | x |
| ≤5 min setup, ≤3 commands | x | | | x | | x | |

## Domain-Specific Requirements

### Cryptographic Standards

- **SSH E2E encryption.** SSH is the trust layer between nodes. Switchboard does not double-encrypt. OpenSSH key format for all node credentials (admission keys, session authorization keys). No proprietary key formats.
- **Noise Protocol Framework.** Router-to-router authentication via Noise handshakes with static keypairs. Same crypto framework as node-to-router. No PKI overhead.
- **HMAC frame authentication.** Routers verify frames via HMAC in the outer envelope. Lightweight admission proof — non-members' frames rejected at first router, not forwarded.
- **Carrier-grade separation is a provable property.** Any test showing router access to decrypted session content is a project failure. The separation must be demonstrable, not claimed.

### Wire Protocol Constraints

- **Binary protocol correctness.** 44-byte outer header, channel header, frame envelope — byte-level layout is specified and must be conformant across all implementations. No undocumented fields, no version mismatches between node and router.
- **Protocol versioning from day one.** Version field in outer header (1 byte). Extensibility via versioned outer header (fixed format, fast parsing) and TLV extensions in channel header (endpoint flexibility).
- **Interoperability.** Nodes and routers at different versions must interoperate within the same major version. Outer header is router-parsed; changes require router upgrades. Channel header is endpoint-only; changes do not require router upgrades.

### Real-Time Latency Constraints

- **Human perception is the latency budget.** Keystroke-to-echo latency must stay within the threshold where remote sessions feel local. These are neuroscience-derived limits, not engineering targets. Specific targets are in NFR Performance.
- **Every protocol decision is latency-constrained.** Timeslice framing, recovery cascade, FEC group size, interleaving depth — all bounded by the perception budget. Mechanisms that add latency (interleaving depth × tick interval) must stay within budget or are not used.
- **Zero per-frame allocation.** Implementation must not allocate memory per frame. Buffer pooling, efficient IP types, GC tuning as needed.

### Network Security Model

- **Node private keys never transit the network.** Public keys transit as needed for admission and membership. Private keys remain on nodes.
- **No direct node-to-node communication.** Always through a router. This is the admission enforcement point — if nodes communicated directly, admission checks would be bypassable.
- **Router compromise = availability/quality degradation, not confidentiality/integrity compromise.** This is the design invariant.
- **SVTN cryptographic isolation.** Different SVTNs on the same router infrastructure are cryptographically separated. Cross-SVTN visibility is not possible without keys for both SVTNs.

### Session Substrate

- **tmux is the primary session substrate.** Access nodes use tmux control mode for session management, presence data, and content-type detection. Non-tmux sessions are supported via PTY fallback with reduced functionality (no control mode metadata, no content-type detection, byte-rate heuristic only).

### Standards and Protocol Heritage

| Standard/Protocol | How Switchboard Uses It |
|-------------------|------------------------|
| OpenSSH key format | Node credential system (admission, session auth) |
| tmux control mode (`-CC`) | Access node session management, presence data |
| Noise Protocol Framework | Router-to-router authentication |
| QUIC (RFC 9000/9002) | Retransmit frames not packets; per-path RTT tracking |
| X.25 LAP-B | Sliding window with SREJ; piggybacked ACKs |
| SRT | TLPKTDROP; immediate NAK on loss detection |
| WebRTC (RFC 8854) | Four-tier recovery hierarchy; adaptive FEC rate |
| MPTCP (RFC 8684) | Lowest-RTT-first path selection; dual-path scheduling |

### Risk Mitigations

| Risk | Mitigation |
|------|-----------|
| Cryptographic implementation error breaks carrier-grade separation | Integration tests that capture traffic at router and verify payload opacity |
| Protocol version incompatibility between nodes and routers | Version field in outer header from day one; interoperability tested in CI |
| GC pauses cause perceptible latency jitter | Go GC is sub-ms for this workload (Tailscale precedent). Monitor via `runtime/metrics`. Escape hatch: Rust shared library for hot path via FFI if profiling shows GC jitter |
| tmux control mode behavior changes across versions | Minimum tmux version requirement. PTY fallback (AN-E) for older tmux. Feature detection, not version sniffing |

## Innovation & Novel Patterns

### Protocol Design

Switchboard implements a session-layer protocol on top of standard transport (UDP, with TCP fallback for restrictive NAT environments). The L4 transport is deliberately conventional — NAT traversal requires wide existing support. The innovation is in the session layer above it:

- **Timeslice-driven framing.** "The bus leaves on time, full or not." Each direction has its own independent clock. Variable-size frames bounded by buffer contents at tick time. Borrowed from WAN/satellite acceleration (SKIPS protocol). The tick fires whether there's data or not — an empty frame is a liveness signal, a missing frame is a degradation signal. The bus leaving empty IS the heartbeat.
- **Asymmetric half-channels.** Upstream (keystrokes) and downstream (terminal output) are fundamentally different workloads treated as independent half-channels with independent clocks, sequence spaces, and recovery strategies. Upstream uses idempotent replay. Downstream uses content-type-aware recovery.
- **Content-type-aware loss recovery.** Five content types (interactive, streaming, bulk, TUI, graphics) with different recovery profiles within the same session. Interactive tolerates TLPKTDROP. Graphics never tolerates it. Streaming is reluctant. Most protocols treat all payload uniformly.
- **Telecom heritage.** Carrier-grade content separation (routers route but can't read), MPLS-inspired label switching (future), ATM-informed framing (learning from ATM's fixed-cell failure), X.25 sliding window with SREJ. Production-proven networking concepts applied to a new workload.

### What's Good Execution, Not Innovation

- Multi-path forwarding (MPTCP, QUIC precedent)
- Cryptographic admission (Nebula, ZeroTier precedent)
- Overlay routing with latency-aware path selection (RON, Tailscale precedent)
- SSH as E2E trust layer (standard practice)
- UDP transport with NAT traversal (WireGuard, Tailscale precedent)

### Validation

The E router release validates the core session-layer protocol. Timeslice framing is a functional requirement — the tick fires whether there's data or not. An empty frame is a liveness signal; a missing frame is a degradation signal. This is how the network remains observable.

Measurable validation:
- Under simulated packet loss, Switchboard maintains session continuity and signals degradation, while raw SSH stalls or disconnects.
- Keystroke-to-echo latency meets targets (p99 <100ms LAN, <200ms WAN) with timeslice framing overhead.
- Empty-tick frames provide accurate path liveness detection within one tick interval.
- Degradation detection latency is bounded by tick interval × missed-tick threshold.

### Innovation Risks

| Risk | Mitigation |
|------|-----------|
| Timeslice framing adds latency vs. send-immediately | Tunable tick interval. At 5-10ms ticks, added latency is below perception threshold. Measure and compare against raw SSH in integration tests. |
| Content-type detection is wrong, applies incorrect recovery | Conservative default (treat as interactive). Detection is additive — wrong detection degrades to uniform handling, not to failure. |
| NAT traversal failures in restrictive network environments | UDP primary with TCP fallback. STUN/TURN-style hole punching. Proven approach (WireGuard, Tailscale). |
| Asymmetric half-channel design adds protocol complexity | Complexity is in the edge protocol handler, not in the router. Router sees outer header only. Complexity is contained at the endpoints. |

## Network Infrastructure Requirements

### Architecture: Control CLI + Daemons

**`sbctl`** — operator CLI. Talks to running daemons via TCP, authenticates with OpenSSH keys. Not a daemon itself.

| Target | What `sbctl` Does |
|--------|-------------------|
| Router (E/PE/P) | Query status, manage SVTNs, manage keys, view path metrics, admission events |
| Access node | Query published sessions, manage session authorization, view connection state |
| Console | Attach/detach sessions, switch between sessions, navigation, view quality indicators |
| Control node | SVTN lifecycle (create/destroy), key registration/revocation/expiry, membership queries |

All four node types are daemons with remote control planes. `sbctl` is the unified client.

### Daemon Roles

| Role | What It Does |
|------|-------------|
| E/PE router | Forwards frames, enforces admission, serves node connections. E and PE are the same binary — E graduates to PE with upstream config. |
| P router | Pure forwarding, no node protocol. Separate build target, same codebase. |
| Access node | Publishes tmux sessions over the SVTN. Runs on the machine with tmux sessions. |
| Console | Discovers and attaches to remote sessions. Remotely controllable from the start. |
| Control node | Manages SVTN lifecycle, key registration. Daemon that connects to the SVTN as a network participant. |

### Console Control Plane

The console is remotely controllable from the start. Rationale:

- The operator viewing a session may not be the one controlling navigation (read-only viewers, supervisors, marvel orchestration).
- Inline tmux keybinding management (ctrl-b sequences) is awkward for switching sessions, attaching/detaching, and navigating across an SVTN.
- The control plane separation enables marvel to drive consoles programmatically without terminal interaction.

### Control Node as Daemon

The control node is a daemon that connects to the SVTN, not just CLI commands with elevated permissions. It manages SVTN lifecycle and key operations as a network participant. This preserves the three-node-type architecture: access node, console, control node — each a distinct daemon with a distinct role.

### Configuration

- **Format:** YAML config file.
- **Router config:** Listener addresses, upstream router connections (if PE), SVTN definitions, key file paths.
- **Node config:** Router connection addresses, SVTN membership, key file path.
- **Key management:** Local file on the E router for initial release. CLI operations via `sbctl` for registration, revocation, expiry.

### Platform Support

| Component | Platforms | Distribution |
|-----------|----------|-------------|
| Nodes (access, console, control) | macOS (arm64, amd64), Linux (amd64, arm64) | Homebrew tap, direct binary download |
| E/PE router | macOS, Linux | Homebrew tap, direct binary download, container image (scratch base) |
| P router | macOS, Linux (typically deployed in cloud/datacenter) | Container image (scratch base), direct binary download |

### Logging and Observability

- **OTEL telemetry:** Always available, never mandatory. When enabled, exports traces (tool executions, session lifecycle), metrics (latency, loss, session counts), and logs (structured events). Export to self-hosted collector or disabled.
- **Plain text logging:** stdout by default. Structured enough to be useful, not JSON-for-the-sake-of-JSON. Suitable for `journalctl`, Docker logs, terminal observation.
- **Router production output:** Connected nodes, active SVTNs, path quality metrics, admission events, state transitions. Observable without OTEL.

### Upgrade Model

- **Current scope:** Process-ending update. Admin stops the process, updates the binary (package manager, container image tag, direct download), restarts. No in-place hot reload. No self-orchestrated rolling restart.
- **Self-update command:** Optional `sbctl update` (same pattern as forestage). Convenience, not requirement. The admin controls when updates happen.
- **Future (out of scope):** Light container that balances two running processes for rolling restart and update-in-place. Not required to ship.

### Diagnostic Views

Two audiences, different views of the same network:

**Node-side views (operator perspective):**
- Session list with quality indicators (green/yellow/red)
- Per-session attachment status (who is attached where)
- Connection state to router(s)
- Admission status (is my key admitted to this SVTN?)

**Network operator views (router/infrastructure perspective):**
- Connected nodes per SVTN
- Per-path latency and loss metrics
- Admission events (grants, denials, revocations)
- SVTN membership (which nodes, which roles)
- Router status (forwarding table size, active connections)

Both accessible via `sbctl`. The view depends on what you're talking to.

### CLI Commands (`sbctl`)

`sbctl svtn` is canonical. `sbctl net` as alias. Documentation leads with `svtn`.

```
# Router operations
sbctl router status               # connected nodes, active SVTNs, path metrics

# SVTN management
sbctl svtn create <name>          # create SVTN, bootstrap first control key
sbctl svtn list                   # list SVTNs
sbctl svtn status <name>          # membership, admitted keys, active sessions

# Key management
sbctl key register <svtn> <pubkey> [--role control|console|access]
sbctl key revoke <svtn> <pubkey>
sbctl key list <svtn>

# Session operations
sbctl sessions                    # list all discoverable sessions across SVTNs
sbctl connect <session>           # attach console to a discovered session

# Console control
sbctl console status              # attached session, quality, connection state
sbctl console switch <session>    # switch to a different session
sbctl console detach              # detach from current session
sbctl console list                # list running consoles (local or remote)

# Access node
sbctl access status               # published sessions, connected consoles
sbctl access sessions             # list sessions on this access node

# Control node
sbctl control status              # SVTN membership, key state
sbctl control keys <svtn>         # detailed key listing with roles and expiry

# Diagnostics
sbctl status                      # overview: all reachable daemons, connections, SVTNs
sbctl diagnose <svtn>             # per-path latency/loss, admission state, membership
sbctl paths                       # all paths, per-path metrics
```

This is illustrative, not final. The command surface will evolve as implementation reveals what operators actually need.

### Binary Naming (Open)

The naming convention for daemon binaries is an implementation decision. The PRD specifies four daemon roles (router, access, console, control) and one CLI client (`sbctl`). Binary packaging is implementation.

### Implementation Considerations

- **Single static binary per daemon.** No runtime dependencies beyond tmux (for access nodes). `CGO_ENABLED=0` for pure Go static linking.
- **Zero per-frame allocation.** `sync.Pool` for frame buffers. `net/netip` not `net.IP`. GC tuning via `GOGC` and `GOMEMLIMIT`.
- **Container-ready.** Scratch or distroless base. Health endpoint for k8s liveness probes. Graceful shutdown on SIGTERM.
- **Cross-compilation.** `GOOS`/`GOARCH` for all target platforms from single build.

## Project Scoping & Phased Development

### Scope Phases

Defined in Product Scope (above). Three phases tied to architectural triggers: E Router Release, Multi-Path and Multi-Hop, Global Topology.

### Open Questions Requiring Probes

These are the riskiest technical bets in the project. Each is a question that requires probes to learn the answer. The project will adopt kos (see `../kos/`) for knowledge accumulation — these questions seed the frontier.

**Q1: Does timeslice framing work for terminal sessions?**

The entire protocol design rests on "the bus leaves on time, full or not." If timeslice framing introduces perceptible latency or jitter that makes sessions feel worse than raw SSH, the session-layer protocol needs fundamental revision. This is not a tuning question — it's a "does the concept work" question.

Requires probes: build the timeslice clock, send frames over UDP on a LAN, measure keystroke-to-echo latency against raw SSH baseline. Vary tick intervals (5ms, 10ms, 20ms, 50ms). The probe succeeds if there's a tick interval where the session feels indistinguishable from local.

**Q2: Can this run reliably on a typical laptop?**

The session-layer protocol requires sustained sub-100ms loops for frame processing, timeslice assembly, HMAC verification, and sequence tracking. The question is not "Go vs. Rust" — it's whether a user-space program on a typical laptop (macOS, Linux) can maintain reliable frame loops under normal operating conditions, and whether those loops produce a solid terminal experience. This includes: GC pauses, OS scheduling, power management, competing workloads, and the minimum reliable tick interval the platform can sustain.

Requires probes: build the frame processing loop. Measure loop timing stability on macOS and Linux laptops under typical developer workloads. Vary loop intervals. Determine the minimum reliable tick interval. If Go can't maintain it, probe with Rust. If user-space approaches hit OS scheduling limits, probe lower-level options. The probe answers the performance question; the language decision is a consequence.

**Q3: Is tmux control mode reliable enough for production session publishing?**

Access node architecture (AN-E) depends on tmux control mode (`-CC`) for session management, presence data, and content-type detection. Control mode is used by iTerm2 but is not widely exercised by other tools. If control mode has reliability issues under real workloads, the access node needs to fall back to PTY-only mode with reduced functionality.

Requires probes: run an access node prototype using control mode against real tmux sessions. Stress test with high-output sessions (Claude Code, log floods, TUI applications). Measure `%output` completeness against raw PTY capture. Test back-pressure behavior — what happens when the access node can't consume `%output` fast enough. Does tmux pause, drop, or hang? Test with detached terminals — tmux panes running without any client attached, which is the common case for agent sessions. The access node receives what tmux is displaying now, not a complete transcript. The probe must characterize what control mode delivers for detached panes under load. Test `%pause`/`%continue` flow control. The probe succeeds if control mode delivers reliable, characterizable output under sustained load.

### Build Order Within E Router Release

The E router phase has internal dependencies. Build order front-loads the riskiest technical bets:

| Order | What | Why This Order |
|-------|------|----------------|
| 1 | Frame processing and timeslice clock | The core protocol bet. If this is wrong, everything built on top of it is wrong. |
| 2 | Edge protocol (upstream idempotent replay, downstream reliable stream) | Proves the half-channel model and recovery mechanics work. |
| 3 | tmux control mode integration (AN-E) | Proves access node session publishing works. If control mode is fragile, architecture changes. |
| 4 | SVTN admission (Tier 1) and session authorization (Tier 2) | Admission is load-bearing for security model but not for protocol validation. |
| 5 | Console attach and session discovery (presence protocol) | Requires working access node and router. |
| 6 | Degradation signaling and diagnostic CLI | Requires working edge protocol to have something to measure. |
| 7 | Key management CLI and control node | Requires working admission. |

**`sbctl` and daemon APIs** are not a build step — they emerge in parallel with the features they operate. Building the CLI before or alongside the service is essential for testing and operation. Every feature ships with a way to use it.

### Risk Mitigation

| Risk | Mitigation | Probe |
|------|-----------|-------|
| Timeslice framing doesn't work for terminal sessions | Build and measure before building anything else | Q1 |
| User-space program can't maintain reliable frame loops on typical hardware | Profile early on macOS and Linux. Language and platform decisions follow from results. | Q2 |
| tmux control mode is unreliable or incomplete for detached panes under load | PTY fallback (AN-E) with reduced functionality | Q3 |
| E router LAN-only limits usefulness | Honest scope boundary — PE required for roaming. E router proves protocol, not deployment model. | Design constraint |
| NAT traversal for PE routers | UDP primary, TCP fallback, STUN/TURN. Proven approach. PE-scope, not E-scope. | Engineering, not research |

## Functional Requirements

### Session Networking

- **FR1:** Access node can publish local tmux sessions over an SVTN via tmux control mode, with PTY fallback for non-tmux or older tmux sessions.
- **FR2:** Console can attach to a remote tmux session through a router and receive terminal output with keystroke-to-echo latency within perception budget.
- **FR3:** Sending node sends each frame to its connected routers based on its own min/desired/max send configuration. The network handles further path replication.
- **FR4:** Nodes can maintain sessions across network transitions (wifi to LAN, IP address changes) without user intervention or re-authentication.
- **FR5:** Router can continue serving a node whose IP address changes, identified by cryptographic key rather than network address.
- **FR6:** Each half-channel (upstream and downstream) operates with independent timeslice clocks and sequence spaces.
- **FR7:** Upstream half-channel uses idempotent replay (sliding window of last N keystrokes) as application-layer loss recovery, independent of multi-path replication.
- **FR8:** Downstream half-channel uses reliable ordered delivery with piggybacked ACK and SACK bitmap for loss detection and recovery.
- **FR9:** Frames past the perception deadline are delivered via TLPKTDROP (too-late-packet-drop) with degradation signaled to the console.
- **FR10:** Receiving node deduplicates frames — first arrival wins, subsequent copies are discarded.

### Multi-Path Forwarding

- **FR11:** Each node configures its own min/desired/max send count to connected routers, independent of SVTN policy.
- **FR12:** Node tracks link quality to all connected routers and ranks them for path selection, sending to the topX routers with the best link characteristics.
- **FR13:** Node signals degradation when the number of usable router connections falls below its configured minimum.
- **FR14:** Router forwards frames toward a destination on multiple links simultaneously, selecting the best paths based on measured link characteristics, bounded by SVTN per-hop fanout policy.
- **FR15:** SVTN policy specifies per-hop router fanout separately for access-originated traffic and console-originated traffic, set by control node, changeable mid-session.
- **FR16:** Routers apply the class-appropriate fanout based on the originating node type of the frame.
- **FR17:** Router applies split horizon — frames are not forwarded back toward the direction they arrived from.
- **FR18:** Router applies duplicate suppression using a checksum computed at the intake E/PE router. Routers maintain a bounded drop cache of recently forwarded frame checksums to prevent loops. Retransmits and byte-shifted content produce different checksums and pass through.
- **FR19:** Router can signal impending shutdown to connected nodes, allowing graceful migration before disconnect. Per-link drain (removing one link from service without shutting down the router) is a future capability.

### Session Discovery & Presence

- **FR20:** Access node advertises available tmux sessions to all consoles on the same SVTN via multicast presence protocol.
- **FR21:** Console can discover all available sessions across all access nodes on the SVTN without specifying hostnames or IP addresses.
- **FR22:** Presence advertisements include session metadata (session name, attachment status, quality indicator).
- **FR23:** Presence updates are triggered by session state changes (created, closed, attached, detached, quality change), periodic heartbeat, and on-demand request.

### Session Access & Sharing

- **FR24:** Console can subscribe to a session's output stream independently of interactive attachment.
- **FR25:** Operator can attach a console to a discovered session by selecting the session, not by specifying a machine.
- **FR26:** Access node authorizes console access per session using Tier 2 session authorization (authorized console public keys per session).
- **FR27:** Session access modes include full access (read-write) and read-only (view output, no keystrokes).
- **FR28:** Read-only access can be scoped to a specific session, a specific access node, or an entire SVTN.
- **FR29:** Two or more consoles can view the same session simultaneously (one read-write, additional read-only).
- **FR30:** Access node delivers session output to its router(s) once per frame. Routers fan out to all routers with subscribed consoles for that session.
- **FR31:** Routers maintain per-session subscriber lists within an SVTN for efficient output forwarding.

### Network Admission & Security

- **FR32:** Control node can create and destroy SVTNs.
- **FR33:** Control node can register, revoke, and expire keys against an SVTN with role designation (control, console, access).
- **FR34:** Console nodes can revoke, edit, and expire keys for SVTNs they are admitted to.
- **FR35:** Node can join an SVTN by presenting a signed challenge proving possession of a private key whose public key is registered.
- **FR36:** Router verifies frame legitimacy via HMAC in the outer envelope and rejects frames from non-admitted sources.
- **FR37:** Session content is encrypted end-to-end (SSH) between nodes and opaque to routers at all times.
- **FR38:** Router cannot read, modify, or inject session content. Router sees outer header only (addressing, frame type, SVTN ID, HMAC).
- **FR39:** Node private keys never transit the network. Public keys transit as required for admission and membership.
- **FR40:** Console control plane access is authorized — not any sbctl client can drive any console. Authorization mechanism is in scope for architecture to design.

### Network Quality & Observability

- **FR41:** Console displays a per-session quality indicator (green/yellow/red) based on measured path latency and loss.
- **FR42:** Empty-tick frames (no payload) provide path liveness detection — a missing frame is a degradation signal.
- **FR43:** Operator can view per-path latency and loss metrics via sbctl.
- **FR44:** OTEL telemetry can be enabled for traces, metrics, and logs without being required for operation.
- **FR45:** Daemons emit plain text log output to stdout suitable for standard log aggregation and terminal observation.

### Network Management

- **FR46:** First SVTN control key can be bootstrapped locally on the E router (local file).
- **FR47:** Key management operations (register, revoke, expire, list) are available via sbctl CLI.
- **FR48:** Router reports status (connected nodes, active SVTNs, path metrics, admission events) via sbctl.
- **FR49:** Access node reports status (published sessions, connected consoles, connection state) via sbctl.
- **FR50:** Admission status (is this key admitted to this SVTN?) is queryable via sbctl.
- **FR51:** SVTN membership (which nodes, which roles) is queryable via sbctl.
- **FR52:** Router, access node, console, and control node are configurable via config file.
- **FR53:** Control node can update per-class forwarding parameters mid-session, and changes propagate to active routers.

### Console Operations

- **FR54:** Console can be remotely controlled via sbctl (attach, detach, switch session, navigate).
- **FR55:** Console control plane is separate from the session display — the operator viewing a session is not necessarily the one controlling the console.
- **FR56:** Operator can list all running consoles (local or remote) and their current state via sbctl.
- **FR57:** Console can switch between sessions on the same SVTN without disconnecting and reconnecting.

### Deployment & Operations

- **FR58:** E router graduates to PE router by adding upstream router connections in config — same binary, no reinstall.
- **FR59:** P router is a separate build target from the same codebase, producing a distinct binary for pure forwarding without node protocol.
- **FR60:** Daemons support graceful shutdown on SIGTERM.
- **FR61:** Daemons are updatable via process-ending admin-driven restart (stop, update binary, restart).
- **FR62:** All diagnostic and management capabilities are accessible via sbctl CLI without additional tooling, log parsing, or debugger attachment.
- **FR63:** Operator can set up an E router and establish a first session via CLI.

## Non-Functional Requirements

### Performance

| Requirement | Target | Measurement |
|-------------|--------|-------------|
| Keystroke-to-echo latency (LAN) | p50 <50ms, p99 <100ms | Integration test harness; OTEL when enabled |
| Keystroke-to-echo latency (WAN) | p50 <100ms, p99 <200ms | Integration test harness; OTEL when enabled |
| Session discovery latency | New session visible within presence propagation time + network RTT | Integration test: create session, measure appearance time |
| Degradation detection latency | Bounded by tick interval × missed-tick threshold | Integration test: stop sending, measure time to yellow indicator |
| Path failover time | <2 seconds from path loss to session resumption on alternate path | Integration test: kill path, measure recovery |
| Timeslice framing overhead | Tick interval adds <10ms latency vs. send-immediately at typical tick rates (5-10ms) | Benchmark against raw SSH baseline |
| Frame processing per-frame cost | Zero heap allocation per frame in steady state | Profiling; zero-allocation verified |
| E router setup time | <5 minutes from install to first working session; ≤3 commands per machine | Timed walkthrough |

### Security

| Requirement | Target | Measurement |
|-------------|--------|-------------|
| Session content opacity | Routers cannot access decrypted session content under any condition | Integration test: capture traffic at router, verify payload is opaque |
| Outer header information exposure | Routers see only: addressing, frame type, SVTN ID, HMAC, intake checksum. Base outer header is 44 bytes; intake checksum adds to this (size determined by protocol design). Growth beyond the established header size requires explicit security justification. | Protocol conformance test |
| Node key isolation | Node private keys never transit the network | Code audit; no private key in any wire-format message |
| HMAC frame authentication | Non-admitted frames rejected at first router | Integration test: send frame with invalid HMAC, verify rejection |
| SVTN cryptographic isolation | Cross-SVTN visibility impossible without keys for both SVTNs | Integration test: node on SVTN-A cannot see traffic on SVTN-B |
| Console control authorization | Only authorized sbctl clients can control a given console | Architecture to design mechanism; functional test once designed |

### Reliability

| Requirement | Target | Measurement |
|-------------|--------|-------------|
| Session survivability | Sessions survive network transitions (wifi switch, IP change, path failover) without user intervention | Integration test: simulate transitions, assert continuity |
| Router failure resilience | In multi-router topologies, sessions on multi-homed nodes survive the failure of one router | Integration test: kill one router, verify sessions continue via alternate |
| Graceful router drain | Router shutdown signal allows connected nodes to migrate before disconnect | Integration test: send drain, verify migration completes before shutdown |
| No single point of failure | In multi-router topologies, no single component failure takes down all sessions. The E router release is a developmental stage with a single router — single point of failure is inherent and accepted for that phase. | Architecture review; fault injection tests |
| Duplicate suppression | Router drop cache prevents frame loops without dropping legitimate retransmits | Integration test: inject duplicate frames, verify drop; inject retransmit, verify pass-through |

### Scalability

| Requirement | Context | Notes |
|-------------|---------|-------|
| E router node count | ≤50 nodes per E router | Design guide, not validated — requires probing under load |
| PE router: multi-hop topology | Multiple routers, distributed nodes | Performance targets must hold across router hops |
| Per-session fanout | Multiple consoles subscribed to one session | Adding read-only viewers must not measurably degrade primary console latency (within measurement noise of no-viewer baseline) |
| SVTN forwarding parameter changes | Control node updates fanout mid-session | Changes propagate without session interruption |

### Protocol Compatibility

| Requirement | Target | Measurement |
|-------------|--------|-------------|
| Version interoperability | Nodes and routers within the same major protocol version interoperate without coordination | Version matrix integration tests |
| Outer header stability | Outer header format is stable within a major version. Changes require major version bump. | Protocol conformance test |
| Channel header extensibility | Channel header extensions (TLV) do not require router upgrades — channel header is endpoint-only | Integration test: new TLV extension, verify older router forwards without issue |

### Operational

| Requirement | Target | Measurement |
|-------------|--------|-------------|
| Binary dependency | No dependencies beyond the binary itself and tmux (for access nodes) | Build verification |
| Service dependency | No proprietary service dependency, ever | Architecture review |
| Log output | Plain text to stdout, parseable by standard log aggregation tools (key=value or equivalent structured format) | Manual review |
| OTEL overhead | Enabling OTEL telemetry does not degrade session latency beyond measurement noise | Benchmark with/without OTEL |
| Container readiness | Scratch/distroless base, health endpoint, graceful SIGTERM shutdown | Container deployment test |
