---
artifact_id: L2-assumptions
document_type: domain-spec-section
level: L2
section: assumptions
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
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Assumptions

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

Every ASM-NNN begins with `Status: unvalidated`. Holdout candidates are
flagged where confidence is Low or Impact-if-Wrong is HIGH.

---

| ID | Assumption | Confidence | Impact if Wrong | Validation Method | Status | Traced To |
|----|-----------|-----------|----------------|-------------------|--------|-----------|
| ASM-001 | Timeslice framing at 5–10ms tick intervals delivers keystroke-to-echo latency indistinguishable from raw SSH on a LAN. If the tick adds perceptible latency, the core session-layer protocol needs fundamental revision. | Low | HIGH — entire framing model rethink | Build timeslice clock prototype. Measure p50/p99 keystroke-to-echo vs. raw SSH baseline at 5, 10, 20, 50ms tick rates. Target: at least one tick rate where session feels local. (PRD Q1 probe.) Holdout candidate: yes | unvalidated | elem-timeslice-framing, CAP-001 |
| ASM-002 | A Go user-space program on a typical developer laptop (macOS, Linux) can maintain reliable sub-100ms frame-processing loops under normal OS scheduling conditions. GC pauses do not produce perceptible latency jitter at terminal session timescales. | Medium | HIGH — may require Rust FFI or platform-specific workarounds | Build frame processing loop. Profile loop timing stability on macOS and Linux under typical developer workloads. Measure GC pause frequency and duration via runtime/metrics. (PRD Q2 probe.) Holdout candidate: yes | unvalidated | CAP-001, CAP-003 |
| ASM-003 | tmux control mode (`-CC`) is reliable enough for production session publishing under real workloads including high-output sessions, Claude Code output, TUI applications, and detached panes under load. | Medium | HIGH — access node architecture changes; PTY-only with reduced functionality | Run access node prototype with control mode against real tmux sessions. Stress test with high-output and detached panes. Measure %output completeness. Test %pause/%continue back-pressure. (PRD Q3 probe.) Holdout candidate: yes | unvalidated | CAP-013, elem-mvp-scope-single-lan |
| ASM-004 | Terminal professionals (the target audience) will self-serve from a binary + README without a GUI or onboarding wizard. The complexity ceiling for the E router case is three CLI commands. | High | Medium — onboarding wizard needed; increases scope | Timed onboarding walkthrough: install → E router running → first session attached. Target: ≤5 minutes, ≤3 commands per machine. | unvalidated | CAP-023, CAP-024 |
| ASM-005 | AI agent workloads (Claude Code, Codex, Devin running in tmux) constitute a significant and growing segment of terminal session traffic. If the industry moves to browser-based or API-only agent interfaces, the fleet-management use case weakens (though Devon, Priya, and Marcus use cases remain valid). | Medium | Low — core use cases survive without fleet-management; affects product priority, not architecture | Monitor AI coding agent deployment patterns. Track percentage of Claude Code deployments using terminal vs. browser-based interfaces. | unvalidated | CAP-011, CAP-012 |
| ASM-006 | OpenSSH keypairs remain the standard credential format for terminal infrastructure. The E2E trust model does not require rework if post-SSH encrypted terminal protocols emerge (architecture survives; trust layer adapts). | High | Medium — trust layer rework; SSH not replaced overnight | Monitor IETF/OpenSSH developments. No probe required; assumption is long-horizon. | unvalidated | DI-001, elem-ssh-end-to-end-encryption |
| ASM-007 | UDP transport with STUN/TURN-style hole punching is sufficient for NAT traversal in the majority of deployment environments. TCP fallback covers restrictive NAT cases. Exotic corporate NAT configurations may require additional traversal techniques. | Medium | Medium — some deployments unreachable; E router (LAN-only) not affected; PE router affected | Test NAT traversal in: home NAT, corporate symmetric NAT, cloud-to-cloud, mobile carrier NAT. Required before PE router ships. | unvalidated | CAP-004, CAP-026 |
| ASM-008 | The 44-byte outer header is sufficient for the E router phase. Future multi-hop topologies may require additional fields (TTL/hop-limit) via a major version bump, not an expansion of the current header. | High | Low for E router phase; Medium for multi-hop — protocol redesign if wrong | Protocol conformance testing in E router phase. Version bump assessment required before PE router ships. | unvalidated | DI-007, CAP-003 |
