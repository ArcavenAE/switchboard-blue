---
artifact_id: STORY-INDEX
document_type: story-index
level: ops
version: "1.0"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
cycle: v1.0.0-greenfield
inputDocuments:
  - '.factory/stories/dependency-graph.md'
  - '.factory/cycles/cycle-1/wave-schedule.md'
---

# Story Index: Switchboard Cycle 1

## Summary

| Metric | Value |
|--------|-------|
| Total stories | 21 |
| Complete | 2 (S-0.01, S-1.01) |
| Pending | 19 |
| E-phase | 17 |
| PE-phase | 4 |
| Total points | 102 |
| Waves | 7 (Wave 0–6) |

## Master Story Index

| Story ID | Title | Epic | Wave | BC Traces | Subsystems | Points | Priority | Scope | Status |
|---------|-------|------|------|-----------|-----------|--------|---------|-------|--------|
| S-0.01 | Port BMAD scaffolding as wave-0 baseline | E-0 | 0 | (none) | cmd/switchboard | 1 | P0 | E | complete |
| S-1.01 | Implement 44-byte outer header codec | E-1 | 1 | BC-2.01.004, BC-2.01.005, BC-2.01.006 | session-networking | 5 | P0 | E | completed |
| S-1.02 | Implement timeslice clock state machine | E-1 | 1 | BC-2.01.001, BC-2.01.002, BC-2.01.003 | session-networking | 8 | P0 | E | pending |
| S-1.03 | Session continuity via cryptographic re-authentication | E-1 | 2 | BC-2.01.007 | session-networking, admission-security | 5 | P0 | E | pending |
| S-2.01 | Implement HMAC-SHA256 frame authentication | E-2 | 2 | BC-2.05.005 | admission-security | 5 | P0 | E | pending |
| S-2.02 | Tier-1 admission and SVTN isolation | E-2 | 2 | BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007 | admission-security | 8 | P0 | E | pending |
| S-3.01 | Tmux control mode integration with PTY fallback | E-3 | 3 | BC-2.04.001, BC-2.04.002 | session-access | 8 | P0 | E | pending |
| S-3.02 | Console attach/detach and multi-console fan-out | E-3 | 3 | BC-2.04.003, BC-2.04.004, BC-2.04.006 | session-access | 8 | P0 | E | pending |
| S-3.03 | Tier-2 per-session authorization and read-only | E-3 | 3 | BC-2.04.005, BC-2.05.003 | session-access, admission-security | 5 | P0 | E | pending |
| S-4.01 | Per-path RTT/loss tracking and dup-and-race | E-4 | 4 | BC-2.02.001, BC-2.02.002, BC-2.02.003, BC-2.02.009 | multipath-forwarding | 8 | P0 | E | pending |
| S-4.02 | Upstream idempotent replay window | E-4 | 4 | BC-2.02.004 | multipath-forwarding | 5 | P0 | E | pending |
| S-4.03 | Downstream ARQ with ACK/SACK and TLPKTDROP | E-4 | 4 | BC-2.02.005, BC-2.02.006 | multipath-forwarding | 8 | P0 | E | pending |
| S-4.04 | Split-horizon loop prevention | E-4 | 4 | BC-2.02.008 | multipath-forwarding | 5 | P0 | E | pending |
| S-5.01 | Green/yellow/red quality indicator with hysteresis | E-5 | 5 | BC-2.06.001, BC-2.06.002 | quality-observability | 5 | P1 | E | pending |
| S-5.02 | sbctl router status metrics query | E-5 | 5 | BC-2.06.003 | quality-observability, network-management | 3 | P1 | E | pending |
| S-6.01 | Config parsing and validation | E-6 | 4 | BC-2.09.003 | deployment-operations | 3 | P0 | E | pending |
| S-6.02 | SVTN lifecycle and key management via sbctl admin | E-6 | 5 | BC-2.05.004, BC-2.07.001 | network-management, admission-security | 8 | P0 | E | pending |
| S-6.03 | sbctl unified CLI + connection error reporting | E-6 | 5 | BC-2.07.002, BC-2.07.003 | network-management | 5 | P0 | E | pending |
| S-7.01 | XOR parity FEC for single-loss recovery | E-7 | 6 | BC-2.02.007 | multipath-forwarding | 8 | P1 | PE | pending |
| S-7.02 | SVTN-scoped multicast session discovery | E-7 | 6 | BC-2.03.001, BC-2.03.002, BC-2.03.003 | session-discovery | 8 | P1 | PE | pending |
| S-7.03 | Console remote control via sbctl | E-7 | 6 | BC-2.08.001 | console-operations, network-management | 5 | P1 | PE | pending |
| S-7.04 | E-to-PE router graduation and graceful drain | E-7 | 6 | BC-2.09.001, BC-2.09.002 | deployment-operations | 8 | P2 | PE | pending |

## Wave Summary

| Wave | Stories | Points | Theme |
|------|---------|--------|-------|
| 0 | S-0.01 | 1 | BMAD scaffolding (complete) |
| 1 | S-1.01, S-1.02 | 13 | Frame codec + half-channel clock |
| 2 | S-1.03, S-2.01, S-2.02 | 18 | Security foundation + session continuity |
| 3 | S-3.01, S-3.02, S-3.03 | 21 | Session access MVP |
| 4 | S-4.01, S-4.02, S-4.03, S-4.04, S-6.01 | 29 | Reliability layer + config |
| 5 | S-5.01, S-5.02, S-6.02, S-6.03 | 21 | Observability + CLI |
| 6 | S-7.01, S-7.02, S-7.03, S-7.04 | 29 | PE-phase features |
| **Total** | **21** | **132** | |

> Note: Wave 2 includes S-1.03 (depends on S-1.01 + S-2.02). Total points including Wave 0: 133. Per-wave counts above use story points from individual story files.

## BC Coverage Check

All 42 BCs covered. See dependency-graph.md BC-to-Stories matrix for full traceability.

## Files

All story files are in `.factory/stories/S-N.MM-*.md`. Epic files are in `.factory/stories/epics/E-N-*.md`.
