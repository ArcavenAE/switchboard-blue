---
document_type: wave-schedule
level: ops
version: "1.0"
status: draft
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
cycle: cycle-1
inputs:
  - '.factory/stories/STORY-INDEX.md'
  - '.factory/stories/dependency-graph.md'
traces_to: '.factory/stories/STORY-INDEX.md'
---

# Wave Schedule: Switchboard Cycle 1

## Summary

| Metric | Value |
|--------|-------|
| Total stories | 21 |
| Total waves | 7 (Wave 0 – 6) |
| Max parallelism | Wave 4: 5 parallel stories |
| Total story points | 102 |
| E-phase stories | 17 |
| PE-phase stories | 4 |

## Wave Plan

### Wave 0 (scaffolding — COMPLETE)

| Story | Title | Points | Status |
|-------|-------|--------|--------|
| S-0.01 | Port BMAD scaffolding as wave-0 baseline | 1 | complete |

**Wave 0 gate:** CI green. BMAD pre-cycle delivered. No holdout required (meta-work).

---

### Wave 1 — Foundation (no product dependencies beyond Wave 0)

| Story | Title | Points | Deps | Priority |
|-------|-------|--------|------|---------|
| S-1.01 | Implement 44-byte outer header codec | 5 | S-0.01 | P0 |
| S-1.02 | Implement timeslice clock state machine | 8 | S-1.01 | P0 |

**Wave 1 total points:** 13
**Gate criteria:**
- All Wave 1 tests pass; Red Gate verified before implementation
- S-1.01: EncodeOuterHeader produces exactly 44 bytes (VP-003)
- S-1.02: HalfChannel.Tick emits exactly one frame per tick (VP-016)
- Holdout: see wave-scenarios/wave-1.md

---

### Wave 2 — Security Foundation

| Story | Title | Points | Deps | Priority |
|-------|-------|--------|------|---------|
| S-2.01 | Implement HMAC-SHA256 frame authentication | 5 | S-1.01 | P0 |
| S-2.02 | Implement tier-1 admission and SVTN isolation | 8 | S-1.01, S-2.01 | P0 |

**Wave 2 total points:** 13
**Gate criteria:**
- VP-004: ComputeHMAC / VerifyHMAC consistency
- VP-007: Node private key never in wire structs
- VP-010: SVTNRoute never delivers to wrong SVTN
- Holdout: see wave-scenarios/wave-2.md

---

### Wave 3 — Session Access MVP

| Story | Title | Points | Deps | Priority |
|-------|-------|--------|------|---------|
| S-3.01 | Tmux control mode integration with PTY fallback | 8 | S-1.02, S-2.02 | P0 |
| S-3.02 | Console attach/detach and multi-console fan-out | 8 | S-3.01 | P0 |
| S-3.03 | Tier-2 per-session authorization and read-only | 5 | S-3.02, S-2.02 | P0 |

**Wave 3 total points:** 21
**Gate criteria:**
- VP-032: PTY fallback activates on control mode failure
- VP-034: Multi-console fan-out: both consoles receive all frames
- VP-012: SessionAuth rejects unauthorized console key
- Holdout: see wave-scenarios/wave-3.md

---

### Wave 4 — Reliability Layer + Config

| Story | Title | Points | Deps | Priority |
|-------|-------|--------|------|---------|
| S-4.01 | Per-path RTT/loss tracking and duplicate-and-race | 8 | S-1.01, S-2.02 | P0 |
| S-4.02 | Upstream idempotent replay window | 5 | S-1.01, S-1.02 | P0 |
| S-4.03 | Downstream ARQ with ACK/SACK and TLPKTDROP | 8 | S-1.01, S-1.02 | P0 |
| S-4.04 | Split-horizon loop prevention | 5 | S-2.02, S-4.01 | P0 |
| S-6.01 | Config parsing and validation | 3 | S-1.01 | P0 |

**Wave 4 total points:** 29 (highest parallelism wave)
**Gate criteria:**
- VP-024: Multipath delivers first copy, discards duplicates
- VP-019: ARQ.OnAck never delivers a frame twice
- VP-022: Replay.OnUpstream never delivers same seq twice
- VP-011: Split-horizon: no forward toward arrival interface
- VP-028: Config.Validate rejects out-of-range tick_interval
- Holdout: see wave-scenarios/wave-4.md

---

### Wave 5 — Observability + CLI

| Story | Title | Points | Deps | Priority |
|-------|-------|--------|------|---------|
| S-5.01 | Green/yellow/red quality indicator | 5 | S-4.01, S-4.03 | P1 |
| S-6.03 | sbctl unified CLI + connection error | 5 | S-6.01 | P0 |
| S-6.02 | SVTN lifecycle and key management | 8 | S-2.02, S-6.01 | P0 |
| S-5.02 | sbctl router status metrics query | 3 | S-5.01, S-6.03 | P1 |

**Wave 5 total points:** 21
**Note:** S-6.03 can begin concurrently with S-5.01 and S-6.02 since it depends only on S-6.01 (Wave 4). S-5.02 depends on both S-5.01 and S-6.03 so runs last in Wave 5.
**Gate criteria:**
- VP-027: QualityIndicator transitions: degradation only goes down
- VP-030: sbctl exits 1 with E-NET-001 on connection refused
- VP-046: Key lifecycle: register/revoke/expire
- Holdout: see wave-scenarios/wave-5.md

---

### Wave 6 — PE-Phase Features

| Story | Title | Points | Deps | Priority |
|-------|-------|--------|------|---------|
| S-7.01 | XOR parity FEC for single-loss recovery | 8 | S-4.03 | P1 |
| S-7.02 | SVTN-scoped multicast session discovery | 8 | S-2.02, S-3.02 | P1 |
| S-7.03 | Console remote control via sbctl | 5 | S-3.02, S-6.03 | P1 |
| S-7.04 | E-to-PE graduation and graceful drain | 8 | S-6.01, S-4.04 | P2 |

**Wave 6 total points:** 29
**Gate criteria:**
- VP-043: XOR FEC: single loss in group recoverable
- VP-044: Presence advertisement includes required fields
- VP-050: Console remotely controllable via sbctl
- VP-037: Router drain: nodes migrate within 2s
- Holdout: see wave-scenarios/wave-6.md

---

## Critical Path

```
S-0.01 → S-1.01 → S-1.02 → S-3.01 → S-3.02 → S-3.03
                                                         [Wave 3 complete]
S-1.01 → S-2.01 → S-2.02 → S-4.01 → S-4.03 → S-5.01
                                                         [Wave 5 partial]
S-1.01 → S-6.01 → S-6.03 → S-6.02 → S-5.02
                                                         [Wave 5 complete]
```

Longest chain estimate: S-0.01(1) → S-1.01(5) → S-1.02(8) → S-3.01(8) → S-3.02(8) → S-3.03(5) = **35 points on critical path**

## Pipeline Overlap Plan

| Parallel Activity | When |
|------------------|------|
| Wave 2 stub generation | Start when Wave 1 types compile |
| Wave 2 tests | Start when Wave 1 Red Gate verified |
| S-4.01, S-4.02, S-4.03, S-4.04, S-6.01 | All parallel in Wave 4 after Wave 2+3 complete |
| S-5.01, S-6.03, S-6.02 | Parallel at Wave 5 start; S-5.02 last in Wave 5 |
| S-7.01, S-7.02, S-7.03, S-7.04 | All parallel in Wave 6 |
