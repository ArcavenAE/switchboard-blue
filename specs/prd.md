---
artifact_id: PRD
document_type: prd
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-06-23T00:00:00
phase: 1a
inputs:
  - '.factory/specs/domain-spec/L2-INDEX.md'
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/edge-cases.md'
  - '.factory/specs/domain-spec/assumptions.md'
  - '.factory/specs/domain-spec/risks.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '.factory/specs/domain-spec/differentiators.md'
  - '_bmad-output/planning-artifacts/product-brief-switchboard-2026-03-31.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "[md5-pending]"
traces_to: '.factory/specs/domain-spec/L2-INDEX.md'
supplements:
  - 'prd-supplements/interface-definitions.md'
  - 'prd-supplements/error-taxonomy.md'
  - 'prd-supplements/nfr-catalog.md'
---

# Product Requirements Document: Switchboard

> **BC Index Model:** This PRD is an index document. Each Behavioral Contract (BC)
> lives in its own file under `behavioral-contracts/ss-NN/`. Tables in Section 2
> provide one-line summaries linking to individual BC files. Full contract details
> are NOT inlined here.

> **PRD Supplement Model:** Sections 3–5 reference separate supplement files under
> `prd-supplements/`. Each supplement targets a different consumer agent.

---

## 1. Product Overview

### 1.1 Problem Statement

Terminal sessions are increasingly mission-critical infrastructure. Platform engineers and AI ops professionals managing fleets of agent sessions (Claude Code, Codex, Devin) have no purpose-built network layer: they duct-tape SSH tunnels, suffer dropped reconnections on network transitions, and have zero visibility into session quality. Existing tools (SSH, Mosh, tmate, Tailscale) are either point-to-point, relay-based with visible session content, or general-purpose IP overlays not optimized for interactive terminal latency.

### 1.2 Solution Vision

Switchboard is a **session network** — a switched virtual terminal network (SVTN) whose only cargo is terminal sessions. It provides latency-optimized multi-path routing, cryptographic two-tier admission, multi-path failover, session quality monitoring, and carrier-grade content separation in a single static Go binary with three deployment modes (E/PE/P router). Operators need no external infrastructure for the E router MVP: one binary, one LAN, five minutes to a working session network. Complexity is available but never required.

Key architectural invariants: every router sees only outer-header metadata (no session content), all traffic flows node-to-router-to-node (no direct node connections), SSH provides end-to-end trust, and the timeslice clock fires on every tick whether or not data is pending.

### 1.3 Key Differentiators

| ID | Differentiator | Description |
|----|---------------|-------------|
| KD-001 | New category — session network | Purpose-built multi-tenant switched network for terminal sessions; not a tunnel, relay, or VPN |
| KD-002 | Carrier-grade content separation | Routers route by identity and addressing; session content is SSH-encrypted end-to-end, invisible to routers |
| KD-003 | Terminal-native optimization | Asymmetric half-channels, timeslice framing, content-type-aware loss recovery tuned for keystroke-to-echo latency |
| KD-004 | The illusion of local | Session continuity across IP changes, dual-path resilience, honest degradation indicators |
| KD-005 | Progressive complexity | E router MVP requires no external infrastructure; same binary graduates to PE/P by config change |
| KD-006 | Open source, sovereign infrastructure | No vendor dependency, no phone-home, keys and sessions stay on operator-controlled infrastructure |

### 1.4 Target Users

| Persona | Description | Volume | Pain Level |
|---------|-------------|--------|------------|
| Devon (AI ops) | Manages 40 Claude Code agent sessions from a single console; needs fleet visibility without hostnames | High (AI workload growth) | Critical — today's tooling is fragmented |
| Priya (platform engineer) | Observes agent sessions read-only for debugging without credential sharing | Medium | High — read-only SSH requires wrapper scripts |
| Marcus (network operator) | Maintains multi-site SVTN, needs per-path diagnostics, rolling updates without session drops | Low-Medium | High — no session-network tooling exists |
| Kai (team lead) | Views 40 sessions with quality indicators; escalates before sessions degrade | Medium | Medium — quality visibility not available today |
| Solo developer | Survives laptop wifi-to-LAN handoff without SSH freeze | High | Medium — tolerated but annoying |

### 1.5 Out of Scope

- P router (provider core, router-facing only) — architecture exists but not built until justified by production data
- Browser-based session viewing or web relay
- Non-tmux session substrates (Zellij, screen) — PTY fallback provides minimal support; tmux-first is scope
- General-purpose IP tunneling or VPN functionality
- Session content logging or recording by the router
- GUI or wizard-based onboarding (terminal professionals self-serve from binary + README)
- NAT traversal (STUN/TURN) — required before PE router ships, not in E router MVP
- Post-quantum cryptography — SSH key format assumed stable (ASM-006)
- Usage telemetry without operator consent

---

## 2. Behavioral Contracts Index

> BCs are grouped by L2 domain subsystem. Numbering: BC-S.SS.NNN where
> S = section (always 2 here), SS = subsystem (01-09), NNN = sequential.
> Full contracts live in `behavioral-contracts/ss-NN/`.
> Architecture subsystem IDs (SS-NN) are placeholders pending ARCH-INDEX (Phase 1b).

### 2.01 Session Networking (CAP-001–CAP-004)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.01.001 | Timeslice clock fires on every tick regardless of data availability | P0 | E |
| BC-2.01.002 | Empty-tick frame is a valid liveness signal | P0 | E |
| BC-2.01.003 | Upstream and downstream half-channels operate with independent clocks and sequence spaces | P0 | E |
| BC-2.01.004 | Frame outer-header encoding and decoding at 44-byte fixed layout | P0 | E |
| BC-2.01.005 | Channel header is opaque to routers — parseable only by endpoints | P0 | E |
| BC-2.01.006 | Session identity is cryptographic: node address derived from hash(SVTN-ID, public-key) | P0 | E |
| BC-2.01.007 | Session continuity survives IP address change via cryptographic re-authentication | P0 | E |

> Full contracts: `behavioral-contracts/ss-01/BC-2.01.001.md` – `BC-2.01.007.md`

### 2.02 Multipath Forwarding (CAP-005–CAP-010)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.02.001 | Duplicate-and-race: same frame sent on two fastest paths simultaneously | P0 | E |
| BC-2.02.002 | Receiver delivers first-arriving copy and silently discards subsequent duplicates | P0 | E |
| BC-2.02.003 | Per-path RTT and loss tracked via keep-alive probes; paths ranked by quality | P0 | E |
| BC-2.02.004 | Upstream idempotent replay window: each frame carries last N keystrokes | P0 | E |
| BC-2.02.005 | Downstream ARQ with piggybacked ACK and SACK bitmap | P0 | E |
| BC-2.02.006 | TLPKTDROP terminates overdue downstream frames and signals degradation | P0 | E |
| BC-2.02.007 | XOR parity FEC covers frame groups; single loss in group recoverable without retransmit | P1 | PE |
| BC-2.02.008 | Router split-horizon prevents frames being forwarded back toward arrival interface | P0 | E |
| BC-2.02.009 | Bounded drop cache suppresses looping duplicate frames by checksum | P0 | E |

> Full contracts: `behavioral-contracts/ss-02/BC-2.02.001.md` – `BC-2.02.009.md`

### 2.03 Session Discovery (CAP-011–CAP-012)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.03.001 | Access node advertises session presence via SVTN-scoped multicast on state change and periodic heartbeat | P1 | PE |
| BC-2.03.002 | Console enumerates all SVTN sessions without specifying hostnames or IP addresses | P1 | PE |
| BC-2.03.003 | Presence advertisement includes session name, attachment status, and quality indicator | P1 | PE |

> Full contracts: `behavioral-contracts/ss-03/BC-2.03.001.md` – `BC-2.03.003.md`

### 2.04 Session Access (CAP-013–CAP-016)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.04.001 | Access node connects to local tmux via control mode and publishes sessions over SVTN | P0 | E |
| BC-2.04.002 | Access node falls back to PTY proxy when tmux control mode unavailable | P0 | E |
| BC-2.04.003 | Console attaches to session by name; receives downstream stream and sends upstream keystrokes | P1 | E |
| BC-2.04.004 | Console detach releases session without closing it; session continues on access node | P1 | E |
| BC-2.04.005 | Read-only console receives downstream stream; upstream keystrokes are rejected by access node | P1 | E |
| BC-2.04.006 | Two or more consoles may subscribe to the same session output simultaneously | P0 | E |

> Full contracts: `behavioral-contracts/ss-04/BC-2.04.001.md` – `BC-2.04.006.md`

### 2.05 Admission Security (CAP-017–CAP-020)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.05.001 | Tier 1 SVTN admission via signed key challenge | P0 | E |
| BC-2.05.002 | Router rejects non-admitted nodes before forwarding — fail-closed | P0 | E |
| BC-2.05.003 | Per-session Tier 2 authorization enforced by access node, not router | P0 | E |
| BC-2.05.004 | Key lifecycle: register, revoke, and expire admission and session-authorization keys | P1 | E |
| BC-2.05.005 | HMAC frame authentication at first router boundary | P0 | E |
| BC-2.05.006 | SVTN cryptographic isolation: admitted node on SVTN-A cannot see SVTN-B traffic | P0 | E |
| BC-2.05.007 | Node private keys never transit the network under any condition | P0 | E |

> Full contracts: `behavioral-contracts/ss-05/BC-2.05.001.md` – `BC-2.05.007.md`

### 2.06 Quality Observability (CAP-021–CAP-022)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.06.001 | Quality indicator (green/yellow/red) derived from measured path latency and loss | P1 | E |
| BC-2.06.002 | Missing expected frame is a degradation signal triggering indicator downgrade | P1 | E |
| BC-2.06.003 | Per-path RTT and loss metrics queryable via sbctl | P1 | E |

> Full contracts: `behavioral-contracts/ss-06/BC-2.06.001.md` – `BC-2.06.003.md`

### 2.07 Network Management (CAP-023–CAP-024)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.07.001 | Control node creates and destroys SVTNs; first control key bootstrapped locally | P2 | E |
| BC-2.07.002 | sbctl unified CLI for all four daemon types with OpenSSH key authentication | P2 | E |
| BC-2.07.003 | sbctl reports clear connection error when target daemon is unreachable | P0 | E |

> Full contracts: `behavioral-contracts/ss-07/BC-2.07.001.md` – `BC-2.07.003.md`

### 2.08 Console Operations (CAP-025)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.08.001 | Console remotely controllable via sbctl: attach, detach, switch session, navigate | P1 | PE |

> Full contract: `behavioral-contracts/ss-08/BC-2.08.001.md`

### 2.09 Deployment Operations (CAP-026–CAP-027)

| BC ID | Title | Priority | Scope Phase |
|-------|-------|----------|-------------|
| BC-2.09.001 | E router graduates to PE mode by adding upstream router connections in config | P2 | PE |
| BC-2.09.002 | Router sends drain signal before shutdown; nodes migrate to alternate routers | P2 | PE |
| BC-2.09.003 | Router startup fails cleanly on malformed config with actionable error message | P0 | E |

> Full contracts: `behavioral-contracts/ss-09/BC-2.09.001.md` – `BC-2.09.003.md`

---

## 3. Interface Definition

> **Supplement:** Full interface definitions are in `prd-supplements/interface-definitions.md`.

Summary: `sbctl` is the single operator CLI targeting all four daemon types (router/E, router/PE, access, console, control). Daemons expose a local Unix socket or TCP port for sbctl connections. JSON output schema covers session list, path metrics, SVTN status, and key inventory. Config file is YAML for all daemons. See supplement for full CLI help text, exit code table, config schema, and flag interaction rules.

---

## 4. Non-Functional Requirements

> **Supplement:** Full NFR catalog is in `prd-supplements/nfr-catalog.md`.

Summary of quantitative targets:
- **NFR-001** Keystroke-to-echo p99 ≤ 100ms over single-hop LAN (ASM-001 probe)
- **NFR-002** Frame processing loop timing stability: p99 jitter ≤ 5ms on developer laptop (ASM-002 probe)
- **NFR-003** Multi-path failover time: < 2 seconds when one of N paths fails (DEC-003)
- **NFR-004** Router throughput: ≥ 1,000 concurrent sessions per E router instance
- **NFR-005** Session content opacity: 0 bytes of payload visible at router in any code path (R-001)
- **NFR-006** Protocol version compatibility: clean rejection (not corruption) on major version mismatch (R-005)

See `prd-supplements/nfr-catalog.md` for the complete catalog with validation methods.

---

## 5. Error Taxonomy

> **Supplement:** Full error taxonomy is in `prd-supplements/error-taxonomy.md`.

Summary: Error codes follow `E-<subsystem>-NNN` convention. 12 FM-NNN failure modes are mapped to E-codes. Severity: broken (non-zero exit), degraded (zero exit with warning), cosmetic. All user-facing messages use `<placeholder>` syntax. See supplement for complete catalog.

---

## 6. Competitive Differentiator Traceability

### 6.1 KD-001 — New Category: Session Network

| BC ID | Contribution |
|-------|-------------|
| BC-2.01.001 | Timeslice framing makes frames session-first primitives |
| BC-2.01.003 | Asymmetric half-channels encode the fundamental session structure |
| BC-2.04.001 | tmux session publishing is native — not a hack |
| BC-2.04.006 | Multi-console fan-out is structural, not bolted on |
| BC-2.05.001 | Two-tier key admission provides multi-tenant isolation |

### 6.2 KD-002 — Carrier-Grade Content Separation

| BC ID | Contribution |
|-------|-------------|
| BC-2.01.004 | Outer header defines exactly what router sees |
| BC-2.01.005 | Channel header is endpoint-only — router cannot parse |
| BC-2.05.005 | HMAC admission proof requires no content inspection |
| BC-2.05.006 | SVTN cryptographic isolation enforces per-network opacity |
| BC-2.05.007 | Private keys never transit — content keys stay on endpoints |

### 6.3 KD-003 — Terminal-Native Optimization

| BC ID | Contribution |
|-------|-------------|
| BC-2.01.001 | Timeslice clock tuned to terminal tick rates (5–50ms) |
| BC-2.01.003 | Independent upstream/downstream clocks optimize each direction |
| BC-2.02.004 | Upstream idempotent replay tuned for keystroke loss |
| BC-2.02.005 | Downstream ARQ tuned for terminal output delivery |
| BC-2.02.006 | TLPKTDROP limits terminal degradation to perception budget |

### 6.4 KD-004 — The Illusion of Local

| BC ID | Contribution |
|-------|-------------|
| BC-2.01.007 | Session survives IP address change — wifi blip transparent |
| BC-2.02.001 | Dual-path forwarding insures against single-path failure |
| BC-2.06.001 | Quality indicator is honest, not a mystery freeze |
| BC-2.09.002 | Graceful drain enables planned maintenance without drops |

### 6.5 KD-005 — Progressive Complexity

| BC ID | Contribution |
|-------|-------------|
| BC-2.09.001 | E→PE graduation is a config change, not a reinstall |
| BC-2.07.002 | sbctl unified CLI — one tool for all management |
| BC-2.09.003 | Clear startup error messages lower the debugging barrier |

### 6.6 KD-006 — Open Source, Sovereign Infrastructure

| BC ID | Contribution |
|-------|-------------|
| BC-2.05.007 | Private keys never transit — cryptographic sovereignty |
| BC-2.05.004 | Key lifecycle locally controlled by operator |
| BC-2.07.001 | SVTN creation requires no external coordination server |

---

## 7. Requirements Traceability Matrix

| BC ID | Source (L2 CAP) | Subsystem | Priority | Scope Phase | Test Type |
|-------|----------------|-----------|----------|-------------|-----------|
| BC-2.01.001 | CAP-001 | session-networking | P0 | E | unit/property |
| BC-2.01.002 | CAP-001 | session-networking | P0 | E | unit/property |
| BC-2.01.003 | CAP-002 | session-networking | P0 | E | unit/integration |
| BC-2.01.004 | CAP-003 | session-networking | P0 | E | unit/fuzz |
| BC-2.01.005 | CAP-003 | session-networking | P0 | E | unit |
| BC-2.01.006 | CAP-004 | session-networking | P0 | E | unit/property |
| BC-2.01.007 | CAP-004 | session-networking | P0 | E | integration/e2e |
| BC-2.02.001 | CAP-005 | multipath-forwarding | P0 | E | integration |
| BC-2.02.002 | CAP-005 | multipath-forwarding | P0 | E | unit/property |
| BC-2.02.003 | CAP-006 | multipath-forwarding | P0 | E | unit/integration |
| BC-2.02.004 | CAP-007 | multipath-forwarding | P0 | E | unit/property/fuzz |
| BC-2.02.005 | CAP-008 | multipath-forwarding | P0 | E | unit/integration |
| BC-2.02.006 | CAP-008 | multipath-forwarding | P0 | E | unit/integration |
| BC-2.02.007 | CAP-009 | multipath-forwarding | P1 | PE | unit/property |
| BC-2.02.008 | CAP-010 | multipath-forwarding | P0 | E | unit/property |
| BC-2.02.009 | CAP-010 | multipath-forwarding | P0 | E | unit/fuzz |
| BC-2.03.001 | CAP-011 | session-discovery | P1 | PE | integration/e2e |
| BC-2.03.002 | CAP-012 | session-discovery | P1 | PE | integration/e2e |
| BC-2.03.003 | CAP-011, CAP-012 | session-discovery | P1 | PE | integration |
| BC-2.04.001 | CAP-013 | session-access | P0 | E | integration/e2e |
| BC-2.04.002 | CAP-013 | session-access | P0 | E | integration |
| BC-2.04.003 | CAP-014 | session-access | P1 | E | integration/e2e |
| BC-2.04.004 | CAP-014 | session-access | P1 | E | integration |
| BC-2.04.005 | CAP-015 | session-access | P1 | E | integration |
| BC-2.04.006 | CAP-016 | session-access | P0 | E | integration/e2e |
| BC-2.05.001 | CAP-017 | admission-security | P0 | E | unit/integration |
| BC-2.05.002 | CAP-017 | admission-security | P0 | E | unit/integration/property |
| BC-2.05.003 | CAP-018 | admission-security | P0 | E | unit/integration |
| BC-2.05.004 | CAP-019 | admission-security | P1 | E | integration |
| BC-2.05.005 | CAP-020 | admission-security | P0 | E | unit/property/fuzz |
| BC-2.05.006 | CAP-020 | admission-security | P0 | E | integration/property |
| BC-2.05.007 | CAP-020 | admission-security | P0 | E | unit/property |
| BC-2.06.001 | CAP-021 | quality-observability | P1 | E | unit/integration |
| BC-2.06.002 | CAP-021 | quality-observability | P1 | E | unit/property |
| BC-2.06.003 | CAP-022 | quality-observability | P1 | E | integration |
| BC-2.07.001 | CAP-023 | network-management | P2 | E | integration/e2e |
| BC-2.07.002 | CAP-024 | network-management | P2 | E | integration/e2e |
| BC-2.07.003 | CAP-024 | network-management | P0 | E | unit/integration |
| BC-2.08.001 | CAP-025 | console-operations | P1 | PE | integration/e2e |
| BC-2.09.001 | CAP-026 | deployment-operations | P2 | PE | integration/e2e |
| BC-2.09.002 | CAP-027 | deployment-operations | P2 | PE | integration/e2e |
| BC-2.09.003 | CAP-023, CAP-024 | deployment-operations | P0 | E | unit/integration |
