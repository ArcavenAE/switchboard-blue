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

Wave 6 was implemented in three tranches. Tranches A and B are CLOSED AND CONVERGED.
Tranche C wave-level review is in progress.

#### Wave 6 — Tranche A (CLOSED)

| Story | Title | Points | PR | Merge SHA |
|-------|-------|--------|----|-----------|
| S-BL.LOOKUP | AdmittedKeySet.Lookup value-return migration | 1 | #40 | eac5d0a |
| S-W5.04 | daemon paths.list/router.metrics/router.status handlers | 5 | #41 | 851e164 |
| S-6.07 | admin.svtn.create handler + sbctl admin svtn create | 3 | #42 | 446efce |

**Tranche A wave-level:** CONVERGED (BC-5.39.001 3/3 clean passes) — see Tranche B wave-level gate below (combined pass covered Tranche A + B integration).

---

#### Wave 6 — Tranche B (CLOSED)

| Story | Title | Points | PR | Merge SHA |
|-------|-------|--------|----|-----------|
| S-7.01 | XOR parity FEC for single-loss recovery | 8 | #43 | 5c658e7 |
| S-7.02 | SVTN-scoped multicast session discovery | 8 | #55 | c54a8ad |
| S-BL.ROUTER-ADDR | populate PathSnapshot.RouterAddr (BC-2.06.003 PC-1) | 2 | #56 | 91d5675 |

**Tranche B wave-level:** CONVERGED (BC-5.39.001 3/3 clean passes) at 2026-07-01.

---

#### Wave 6 — Tranche C

**Composition rationale:** S-6.05 (SVTN destroy) and S-7.03 (Console remote
control) were paired into Tranche C because both stories serialize on
`cmd/sbctl/admin.go` and `cmd/sbctl/main.go` command-dispatch surface (per
RULING-W6TB-A serialization principle). Parallel implementation would produce
merge conflicts on the dispatch table; per-tranche serialization prevents that.

**Stories:**

| Story | Spec version | Primary BC | Primary VP | Merged PR | Merge SHA |
|-------|-------------|------------|-----------|-----------|-----------|
| S-7.03 Console remote control | v1.6 | BC-2.08.001 v1.4 | VP-050 v1.3 | #60 | 7142146 |
| S-6.05 SVTN destroy lifecycle | v1.11 | BC-2.07.001 v1.13 | VP-048 v1.9 | #61 | 7fe3e29 |

**Cross-story integration surface (W-6.C):**

Two surfaces verified in Tranche C:

1. **Shared `--json` envelope.** Both `admin svtn destroy` (S-6.05) and
   `console {attach,detach,switch}` (S-7.03) route through the same
   `writeSuccess`/`writeError` path in `cmd/sbctl/main.go`; envelope shape is
   provably identical. Verified by wave-adversary attempt-4 Adv-A Q2.

2. **Shared error-taxonomy namespaces.** S-6.05 emits `E-SVTN-*`,
   `E-ADM-011`, `E-CFG-006`; S-7.03 emits `E-SES-*`, `E-ADM-006`. No code
   collides between the two story surfaces. Verified by Adv-A Q4.

**Deferred cross-story behavior (out-of-scope for W-6.C).** Destroy-with-
active-console-attach cascade (SVTN destruction propagating a detach to
active console attaches on sessions inside the destroyed SVTN) is deferred
to `S-BL.SESSION-DRAIN` per S-6.05 v1.5 AC-002 out-of-scope note and the
in-code deferral marker at `internal/svtnmgmt/svtnmgmt.go:770-771`.
Wave-6 holdout HS-006 does not exercise this cascade. Manual-eval-only
for W-6.C; full boundary coverage will land with `S-BL.SESSION-DRAIN`.

**Convergence status:**
- S-7.03 per-story: 3/3 CONVERGED at 2213780 (factory-artifacts).
- S-6.05 per-story: 3/3 CONVERGED at 26ce8f0 (factory-artifacts).
- Wave-level: Pass 1 progression:
  - attempt 1: BLOCKING (dispatch-integrity; 3 CRIT / 3 HIGH; remediated at c9789b6)
  - attempt 2: STREAM_WATCHDOG_KILL (600s; no verdict; infrastructure failure)
  - attempt 3: STREAM_WATCHDOG_KILL (600s; no verdict; infrastructure failure)
  - attempt 4: BLOCKING (split-adversary; Adv-A CONVERGENT_L1, Adv-B BLOCKING_L2L3; 0/0/2/0)

**Closure target:** wave_6_tranche_c_wavelevel_converged_at TBD (requires 3/3
clean wave-level passes per BC-5.39.001).

**Wave 6 gate criteria (combined):**
- VP-043: XOR FEC: single loss in group recoverable
- VP-044: Presence advertisement includes required fields
- VP-050: Console remotely controllable via sbctl
- VP-037: Router drain: nodes migrate within 2s (S-7.04 deferred to Wave 7)
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
