---
artifact_id: L2-risks
document_type: domain-spec-section
level: L2
section: risks
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
  - elem-ssh-end-to-end-encryption
  - elem-timeslice-framing
  - elem-node-router-architecture
input-hash: "[md5-pending]"
traces_to: L2-INDEX.md
---

# Risk Register

> **Sharded L2 section (DF-021).** Navigate via `L2-INDEX.md`.

Every R-NNN begins with `Status: open`. Security-focused risks are flagged.
NFR candidates are identified where Impact=HIGH and a quantifiable mitigation exists.

---

| ID | Risk | Likelihood | Impact | Category | Mitigation | Status | Traced To |
|----|------|-----------|--------|----------|-----------|--------|-----------|
| R-001 | Carrier-grade content separation is claimed but not provable. A cryptographic implementation error (wrong HMAC key scope, key leakage, channel header parsed by router) allows a router to access session content. | Low | HIGH | security | Integration tests capture traffic at router and verify payload is opaque under all conditions, including error paths. Any test showing decrypted content is a project failure. Code audit: no private key or channel header parser in router code path. Security focus: yes. NFR candidate: yes (NFR security §"Session content opacity"). | open | DI-001, CAP-020 |
| R-002 | Timeslice framing adds perceptible latency beyond raw SSH baseline, making sessions feel worse than what users already have. | Low | HIGH | performance | Build and measure before committing to the protocol. ASM-001 is the probe. If any tick interval (5–50ms) meets the p99 <100ms LAN target, the risk is mitigated. If none does, the framing model requires revision. NFR candidate: yes (NFR performance §"Timeslice framing overhead"). | open | CAP-001, ASM-001 |
| R-003 | GC pauses in the Go runtime cause perceptible latency jitter on developer laptops, breaking the "feels like local" experience. | Low | HIGH | performance | Go 1.24+ sub-ms GC pauses are well within tolerance for 50–100ms latency budgets (Tailscale precedent). Monitor via runtime/metrics. ASM-002 probe validates. Escape hatch: Rust shared library via CGo/FFI for hot path if profiling shows jitter. NFR candidate: yes (NFR performance §"Frame processing per-frame cost"). | open | CAP-001, CAP-003, ASM-002 |
| R-004 | tmux control mode is unreliable or incomplete for detached panes under sustained load, causing the access node to publish incomplete or stale session data. | Medium | Medium | reliability | PTY fallback (AN-E) provides degraded but functional operation. ASM-003 probe validates control mode reliability before shipping. If control mode is fragile, the access node defaults to PTY-only. | open | CAP-013, ASM-003 |
| R-005 | Protocol version incompatibility between nodes at different versions causes silent data corruption or session failure, rather than a clean rejection. | Low | HIGH | reliability | Version field in outer header from day one. Major version bump required for outer header changes. Interoperability tests run in CI against a version matrix. Channel header TLV extensions do not require router upgrades. NFR candidate: yes (NFR §"Protocol Compatibility"). | open | DI-007, CAP-003 |
| R-006 | "Good enough" inertia: SSH is familiar; dropped sessions are annoying but tolerated. The E router onboarding hurdle is nonzero even at three commands, and users never try the alternative. | Medium | Medium | business | Five-minute E router setup with demonstrable "session survives wifi blip" moment is the conversion event. Homebrew tap and direct binary download minimize installation friction. Progressive deployment means zero infrastructure commitment for trial. | open | ASM-004 |
| R-007 | tmux dependency: if tmux development stalls or a successor (Zellij, screen) displaces it, Switchboard's primary value proposition is coupled to a potentially declining technology. | Low | Medium | business | PTY fallback (AN-E) works with any terminal session, not just tmux. tmux is the optimized path, not the only path. Project decision "tmux-first, depth before breadth" is a scope choice, not an architectural lock-in. | open | CAP-013, DI constraints |
| R-008 | NAT traversal failures in restrictive enterprise network environments prevent PE router nodes from connecting, reducing deployment coverage for Priya and Marcus use cases. | Medium | Medium | reliability | UDP primary, TCP fallback, STUN/TURN hole punching (proven: WireGuard, Tailscale, WireGuard-go). E router (LAN-only) unaffected. NAT traversal is required before PE router ships. ASM-007. | open | CAP-004, CAP-026, ASM-007 |
| R-009 | A router with root access can perform traffic analysis: observe who communicates with whom, when, and how much. Customers with adversarial operator threat models (high-security environments) may require additional privacy guarantees beyond what carrier-grade separation provides. | Medium | Low (in-scope) / HIGH (if out-of-scope customers rely on it) | security | Traffic analysis is an explicit in-scope capability of the operator by design (DI-003). Documentation must be explicit: "The operator sees who communicates with whom." Customers requiring traffic pattern privacy need additional measures outside Switchboard's scope. Security focus: yes. | open | DI-003 |
| R-010 | A compromised router performs a denial-of-service by selectively dropping frames or flooding forged empty-tick frames, degrading session quality without affecting confidentiality or integrity. | Medium | Medium | security | Router compromise → availability/quality degradation is the expected threat model (DI-003). Degradation signaling makes the attack visible. Multi-homed nodes can migrate to alternate routers. The attack does not compromise session content. Security focus: yes. | open | DI-003, CAP-021, CAP-027 |
