---
artifact_id: E-7-pe-phase-features
document_type: epic
level: ops
epic_id: E-7
version: "1.0"
status: pending
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
scope_phase: PE
priority: P1
bc_traces:
  - BC-2.02.007
  - BC-2.03.001
  - BC-2.03.002
  - BC-2.03.003
  - BC-2.08.001
  - BC-2.09.001
  - BC-2.09.002
subsystems: [multipath-forwarding, session-discovery, console-operations, deployment-operations]
architecture_modules: [internal/arq, internal/discovery, internal/drain, internal/config, cmd/sbctl]
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
  - '.factory/specs/architecture/ARCH-06-deployment-and-ops.md'
---

# E-7: PE-Phase Features (Post-MVP)

## Goal

Deliver PE-phase (post-MVP) features: XOR parity FEC for single-loss recovery,
SVTN-scoped multicast session discovery with heartbeat, console remote control
via sbctl, E→PE router graduation via config change, and graceful router drain
with node migration.

## BCs

| BC | Title | Priority | Scope |
|----|-------|---------|-------|
| BC-2.02.007 | XOR parity FEC covers frame groups; single loss in group recoverable without retransmit | P1 | PE |
| BC-2.03.001 | Access node advertises session presence via SVTN-scoped multicast on state change and periodic heartbeat | P1 | PE |
| BC-2.03.002 | Console enumerates all SVTN sessions without specifying hostnames or IP addresses | P1 | PE |
| BC-2.03.003 | Presence advertisement includes session name, attachment status, and quality indicator | P1 | PE |
| BC-2.08.001 | Console remotely controllable via sbctl: attach, detach, switch session, navigate | P1 | PE |
| BC-2.09.001 | E router graduates to PE mode by adding upstream router connections in config | P2 | PE |
| BC-2.09.002 | Router sends drain signal before shutdown; nodes migrate to alternate routers | P2 | PE |

## Phase 1 Drift Addressed

- F-P8-008: BC-2.02.007 story uses canonical `frame_type=fec=0x05` (not `FRAME_TYPE=PARITY`)

## Subsystems Touched

- SS-02 multipath-forwarding (FEC)
- SS-03 session-discovery (discovery)
- SS-08 console-operations (remote control)
- SS-09 deployment-operations (graduation + drain)

## Estimated Stories

4 stories: S-7.01 (XOR FEC), S-7.02 (session discovery), S-7.03 (console remote control), S-7.04 (PE graduation + router drain)
