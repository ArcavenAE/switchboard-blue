---
artifact_id: STORY-INDEX
document_type: story-index
level: ops
version: "1.3"
status: draft
producer: story-writer
timestamp: 2026-06-25T00:00:00
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
| Total stories | 28 (26 wave stories + S-M.01 + S-M.02 maintenance) |
| Complete | 11 (S-0.01, S-1.01, S-1.02, S-2.01, S-2.02, S-1.03, S-3.04, S-3.01a, S-3.01b, S-3.02, S-3.03) |
| Pending | 15 (includes S-W3.04 draft, S-W3.05 ready) |
| Draft/unscheduled | 2 (S-M.01, S-M.02) |
| E-phase | 22 (waves 0–5 + Wave 3 fix-now additions) |
| PE-phase | 4 (wave 6) |
| Maintenance (draft/unscheduled) | 2 (S-M.01, S-M.02) |
| Total points (waves 0–6) | 156 |
| Total points (incl. S-M.01 + S-M.02) | 166 |
| Waves | 7 (Wave 0–6) + maintenance sweep (unscheduled) |
| Backlog | 1 (S-BL.OA) |

## Master Story Index

| Story ID | Title | Epic | Wave | BC Traces | Subsystems | Points | Priority | Scope | Status |
|---------|-------|------|------|-----------|-----------|--------|---------|-------|--------|
| S-0.01 | Port BMAD scaffolding as wave-0 baseline | E-0 | 0 | (none) | cmd/switchboard | 1 | P0 | E | complete |
| S-1.01 | Implement 44-byte outer header codec | E-1 | 1 | BC-2.01.004, BC-2.01.005, BC-2.01.006 | session-networking | 5 | P0 | E | completed |
| S-1.02 | Implement timeslice clock state machine | E-1 | 1 | BC-2.01.001, BC-2.01.002, BC-2.01.003 | session-networking | 8 | P0 | E | completed (PR #2, merge 9e9a98a) |
| S-1.03 | Session continuity via cryptographic re-authentication | E-1 | 2 | BC-2.01.007 | session-networking, admission-security | 5 | P0 | E | completed (PR #7, merge f35e836) |
| S-2.01 | Implement HMAC-SHA256 frame authentication | E-2 | 2 | BC-2.05.005 | admission-security | 5 | P0 | E | completed (PR #5, merge 3c4104e) |
| S-2.02 | Tier-1 admission and SVTN isolation | E-2 | 2 | BC-2.05.001, BC-2.05.002, BC-2.05.006, BC-2.05.007 | admission-security | 8 | P0 | E | completed (PR #6, merge a06b306) |
| S-3.01a | Tmux control mode integration | E-3 | 3 | BC-2.04.001 | session-access | 8 | P0 | E | completed (PR #11, merge 43208ab) |
| S-3.01b | PTY proxy fallback | E-3 | 3 | BC-2.04.002 | session-access | 5 | P0 | E | completed (PR #12, merge 56ec9c7) |
| S-3.02 | Console attach/detach and multi-console fan-out | E-3 | 3 | BC-2.04.003, BC-2.04.004, BC-2.04.006 | session-access | 8 | P0 | E | completed (PR #13, merge 1ff74f5) |
| S-3.03 | Tier-2 per-session authorization and read-only | E-3 | 3 | BC-2.04.005, BC-2.05.003 | session-access, admission-security | 8 | P0 | E | completed (PR #14, merge b68e498) |
| S-3.04 | Wire verifyFrameHMAC into RouteFrame (HMAC enforcement at router boundary) | E-2 | 3 | BC-2.05.008 | admission-security | 3 | P0 | E | completed (PR #9, merge d54bf1a) |
| S-W3.04 | Full daemon assembly — wire all Wave-3 subsystems in cmd/switchboard | E-3 | 3 | BC-2.04.001, BC-2.04.002, BC-2.04.003, BC-2.04.004, BC-2.04.005, BC-2.04.006, BC-2.05.008 | session-access, admission-security | 8 | P0 | E | draft (fix-now F-1; BC-2.04.007 PO decision pending) |
| S-W3.05 | Per-source HMAC failure counter and admission alert (BC-2.05.005 PC-3) | E-2 | 3 | BC-2.05.005, BC-2.05.008 | admission-security | 5 | P0 | E | ready (fix-now F-2) |
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
| 1 | S-1.01, S-1.02 + refactor PR #3 | 13 | Frame codec + half-channel clock — **CLOSED 2026-06-24 (pass-with-clean-drift; rollback resolved 2026-06-24)** |
| 2 | S-1.03, S-2.01, S-2.02 | 18 | Security foundation + session continuity — **COMPLETE 2026-06-25 (3/3 merged; integration gate next)** |
| 3 | S-3.01a, S-3.01b, S-3.02, S-3.03, S-3.04, **S-W3.04**, **S-W3.05** | 45 | Session access MVP + HMAC wire-up + Wave 3 fix-now blockers — original 5 closed; S-W3.04 (daemon assembly, 8pts, draft, F-1 fix-now), S-W3.05 (HMAC alert, 5pts, ready, F-2 fix-now) |
| 4 | S-4.01, S-4.02, S-4.03, S-4.04, S-6.01 | 29 | Reliability layer + config |
| 5 | S-5.01, S-5.02, S-6.02, S-6.03 | 21 | Observability + CLI |
| 6 | S-7.01, S-7.02, S-7.03, S-7.04 | 29 | PE-phase features |
| **Total** | **26** (wave stories) | **156** | (+ S-M.01 + S-M.02 maintenance, 10 pts, unscheduled — grand total 28 stories / 166 pts) |

> Note: Wave 2 includes S-1.03 (depends on S-1.01 + S-2.02). Wave 3 includes S-3.04 (HMAC wire-up into RouteFrame, E-2 epic, P0) and the split of original S-3.01 into S-3.01a (tmux control mode, 8pts) + S-3.01b (PTY fallback, 5pts); S-3.03 repointed 5→8pts. Wave 3 now also includes two FIX-NOW gate blockers: S-W3.04 (daemon assembly, 8pts, E-3, F-1) and S-W3.05 (HMAC failure counter, 5pts, E-2, F-2). Wave 3 total: 7 stories, 45 pts. Total points including Wave 0: 156. Per-wave counts above use story points from individual story files.

## BC Coverage Check

All 43 BCs covered (42 original + BC-2.05.008 minted Wave 3). S-3.04 covers BC-2.05.008 (VP-058). S-W3.04 adds coverage for BC-2.04.001–006 wiring obligations (SessionConnector.Frames() + daemon assembly) and BC-2.05.008 wiring. S-W3.05 adds coverage for BC-2.05.005 PC-3 (E-ADM-017 aggregate alert, VP-059). See dependency-graph.md BC-to-Stories matrix for full traceability.

**Open gap:** BC-2.04.007 (daemon lifecycle) does not yet exist. PO must decide whether to author it before S-W3.04 enters TDD (see S-W3.04 Open Question section). If deferred, a Gap Register entry will be added to dependency-graph.md.

## Backlog / Deferred Stories

Stories created as concrete drift-item targets BEFORE they're scheduled into a wave.
Backlog stubs have minimal frontmatter and no ACs yet. When a wave-N planning cycle
picks one up, story-writer fleshes it out into a normal wave-N story (move out of this
section, add full ACs/tasks/files/architecture).

Backlog convention introduced 2026-06-24 per drbothen/vsdd-factory#260 rollback —
addresses the "deferred to TBD story" anti-pattern.

| Story ID | Title | Status | Drift items consumed | Earliest wave |
|----------|-------|--------|----------------------|---------------|
| S-BL.OA | outer-assembler — compose ChannelFrame + OuterHeader into wire frames | backlog | wave-adv F-001 (spec closed) / F-003 / F-004 | Wave 3+ |

## Maintenance Stories

Stories for DX/tooling/infrastructure work that are NOT part of feature waves 1–7.
Maintenance IDs use the scheme `S-M.NN` (introduced 2026-06-27). No BC anchor required.
Execute in a post-Wave-7 maintenance sweep or standalone orchestrator dispatch.

| Story ID | Title | Epic | Wave | BC Traces | Points | Priority | Status |
|----------|-------|------|------|-----------|--------|----------|--------|
| S-M.01 | Migrate toolchain provisioning from Homebrew to mise | E-MAINT | unscheduled | (none — DX/tooling) | 5 | P2 | draft |
| S-M.02 | Formalize Apple code-signing and notarization of release binaries (toggle-gated) | E-MAINT | unscheduled | (none — release infra) | 5 | P2 | draft |

Epic E-MAINT covers maintenance/DX/self-improvement stories. No BC anchor applies to tooling stories.
Drift items MISE-DX-001 and MISE-DOC-002 are consumed by S-M.01.
Drift item SIGN-DX-001 is consumed by S-M.02. S-M.02 is milestone-gated — SIGNING_ENABLED stays OFF until functional-product milestone.

## Files

All story files are in `.factory/stories/S-N.MM-*.md`. Maintenance story files use `.factory/stories/S-M.NN-*.md`. Backlog stubs use `.factory/stories/S-BL.*-*.md`. Epic files are in `.factory/stories/epics/E-N-*.md`.
