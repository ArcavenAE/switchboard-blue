---
artifact_id: S-BL.BENCH
document_type: story
level: ops
story_id: S-BL.BENCH
title: "Benchmark harness — tick jitter ≤ 2ms and keystroke-echo p99 ≤ 100ms"
status: backlog
producer: story-writer
timestamp: 2026-07-06T00:00:00Z
version: "0.1-backlog-stub"
phase: 2
epic: E-1
wave: backlog
priority: P2
scope_phase: E
estimated_points: 5
bc_traces:
  - BC-2.01.001   # NFR-009 tick jitter ≤ 2ms (VP-041); NFR-001 keystroke-echo p99 ≤ 100ms (VP-042)
  - BC-2.02.001   # duplicate-and-race — VP-042 round-trip includes multipath dispatch
vp_traces:
  - VP-041        # tick p99 jitter ≤ 2ms — Phase-3 b.Errorf gate deliberately omitted; gates here on stable CI
  - VP-042        # keystroke-echo p99 ≤ 100ms — unowned VP per Wave 4 audit L-2; RULING-002 removed from S-4.02
subsystems: [transport-layer]
architecture_modules:
  - internal/halfchannel      # VP-041 BenchmarkTick_Jitter target (proof harness skeleton exists in VP-041.md)
  - internal/testenv          # VP-042 loopback harness: NewLoopback + LoopbackConfig + CreateSession
  - internal/arq              # VP-042 round-trip path
  - internal/multipath        # VP-042 round-trip path (duplicate-and-race)
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on:
  - S-1.02    # halfchannel tick + BenchmarkTick_Jitter skeleton (AC-005 patch removed b.Errorf gate)
  - S-BL.TESTENV   # internal/testenv.NewLoopback required for VP-042 BenchmarkKeystrokeToEcho_P99
inputDocuments:
  - '.factory/specs/verification-properties/VP-041.md'
  - '.factory/specs/verification-properties/VP-042.md'
acceptance_criteria_count: 0
backlog_origin:
  source: VP-041/VP-042 Phase-6 audit + Wave-4-audit-L-2
  drift_items_consumed:
    - OBS-VP-BENCH   # STATE.md drift row: VP-041/VP-042 unverified pending S-BL.BENCH story
  notes: >
    VP-041 HISTORY: S-1.02 AC-005 (rev 1.2) deliberately removed the Phase-3 b.Errorf gate
    from BenchmarkTick_Jitter — developer hardware (M1 MacBook, 2.111ms p99) is not stable CI
    hardware. VP-041 text states: "the ≤ 2ms gate is enforced by VP-041 during Phase-6 formal
    verification on stable CI hardware." Phase-6 VP sweep (2026-07-06) audited VP-041 as
    UNPROVEN-BLOCKED: the benchmark implementation exists at internal/halfchannel/halfchannel_test.go
    BenchmarkTick_Jitter with observational b.ReportMetric; the b.Errorf gate is absent by design.
    Blocker: stable CI hardware + the b.Errorf enforcement gate that this story adds.

    VP-042 HISTORY: Wave-4 fresh-context audit L-2 (2026-06-28) found VP-042 unowned.
    RULING-002 correctly removed it from S-4.02: internal/replay is pure-core map-insert;
    benchmarking at per-call µs resolution cannot verify the 100ms round-trip SLO. VP-042
    requires a full in-process loopback stack (halfchannel + arq + multipath + echo detection)
    that no current story assembles. VP-042.md §Ownership Status names S-BL.BENCH as the
    intended future owner. Phase-6 VP sweep (2026-07-06) audited VP-042 as UNPROVEN-BLOCKED:
    no BenchmarkKeystrokeToEcho_P99 implementation found; depends on internal/testenv.NewLoopback
    which does not exist.

    RULING-002 NOTE: VP-042 was removed from S-4.02 per RULING-002 (Wave 4 audit). The
    unowned-VP history is documented in VP-042.md §Ownership Status and must be preserved
    in this stub's body — future story implementers must know the RULING-002 context to
    understand why VP-042 is not in any wave story's vp_traces.

    Both VPs share the loopback harness dependency (internal/testenv for VP-042; stable CI
    hardware + b.Errorf enforcement for VP-041). S-BL.TESTENV unblocks VP-042's harness;
    VP-041 can be gated on whatever CI baseline the formal-verifier establishes at scheduling.
---

# S-BL.BENCH: Benchmark Harness — VP-041 Tick Jitter and VP-042 Keystroke-Echo

> **STATUS: BACKLOG STUB.** This story owns VP-041 and VP-042, both UNPROVEN-BLOCKED
> per Phase-6 audit (2026-07-06). Acceptance criteria and task list will be fleshed out
> when the story is scheduled.

## Narrative

- **As an** operator and architect
- **I want** automated benchmarks that enforce VP-041 (tick jitter ≤ 2ms) and VP-042
  (keystroke-echo p99 ≤ 100ms) as gated CI checks on stable hardware
- **So that** performance regressions against NFR-009 and NFR-001 are caught before
  they reach production

## Context

### VP-041 — Tick Regularity

S-1.02 AC-005 (rev 1.2) added `BenchmarkTick_Jitter` with `b.ReportMetric(jitter_p99_ms)`
as an **observational** metric only. The `b.Errorf` gate was deliberately removed because
developer hardware (M1 MacBook at 2.111ms p99) is not stable CI. VP-041 states the gate
is deferred to Phase-6 on stable CI hardware. Phase-6 audit confirmed UNPROVEN-BLOCKED.

The proof harness skeleton is already in `internal/halfchannel/halfchannel_test.go:266`
(BenchmarkTick_Jitter). This story adds the `b.Errorf` enforcement gate on a CI baseline
where jitter is reliably ≤ 2ms.

### VP-042 — Keystroke-Echo Round Trip

VP-042 was removed from S-4.02 per RULING-002 (Wave 4 audit L-2): `internal/replay`
benchmarking cannot verify the 100ms round-trip SLO. VP-042 requires a full in-process
loopback stack (`internal/testenv.NewLoopback` + `LoopbackConfig` + `CreateSession` +
`SendKeystroke` + `WaitForEcho`) that does not exist. VP-042 has been unowned since
RULING-002. This story is the designated owner per VP-042.md §Ownership Status.

The proof harness skeleton is in `VP-042.md` (BenchmarkKeystrokeToEcho_P99). It depends
on `internal/testenv.NewLoopback` which is the S-BL.TESTENV deliverable.

### RULING-002 History Note

VP-042 is unowned because RULING-002 (Wave 4 audit L-2) correctly removed it from S-4.02.
No story's `vp_traces` currently carries VP-042. This stub and its scheduling closes
that ownership gap. The history must be preserved here so implementers understand
VP-042 is not missing from other stories by accident.

## Anchors Consumed

| Anchor | Verbatim ID | Source |
|--------|-------------|--------|
| OBS-VP-BENCH drift row | OBS-VP-BENCH | STATE.md; VP-041/VP-042 unverified pending S-BL.BENCH |
| VP-041 b.Errorf gate deferred from Phase-3 | VP-041 | S-1.02 rev 1.2 AC-005 patch; Phase-6 UNPROVEN-BLOCKED |
| VP-042 unowned since RULING-002 | VP-042 | Wave-4 audit L-2; RULING-002 |

## Sketched Acceptance Criteria

**AC-001 (VP-041 gate):** `BenchmarkTick_Jitter` in `internal/halfchannel` adds
`b.Errorf("tick p99 jitter %v exceeds NFR-009 limit 2ms", p99)` on stable CI hardware.
Gate fails CI if p99 exceeds 2ms over 1,000 ticks.

**AC-002 (VP-042 harness):** `BenchmarkKeystrokeToEcho_P99` in `internal/bench` (or
`integration_test`) wires `internal/testenv.NewLoopback` + `LoopbackConfig{upstreamInterval:
10ms, downstreamInterval: 50ms}` + `CreateSession` + `SendKeystroke` + `WaitForEcho` for
500 samples. `b.Errorf` gate on p99 > 100ms.

**AC-003 (VP-041 lock):** `VP-041.verification_lock` set to `true` after gate passes
on CI hardware baseline. Proof-completed-date recorded.

**AC-004 (VP-042 lock):** `VP-042.verification_lock` set to `true` after gate passes.
Proof-completed-date recorded.

## Dependencies

- `S-1.02` — MERGED. BenchmarkTick_Jitter skeleton exists; only the b.Errorf gate is missing.
- `S-BL.TESTENV` — BACKLOG. `internal/testenv.NewLoopback` required for VP-042 harness.
  VP-041 gate can land independently of S-BL.TESTENV.

## Non-Goals

- Does not implement testenv infrastructure. That is `S-BL.TESTENV`.
- Does not change halfchannel or multipath production code.
- VP-041 b.Errorf gate must NOT be added to the Phase-3 path — the AC-005 patch design
  (observational only in Phase 3) must be preserved. This story is the Phase-6 gate.

## When to Schedule

VP-041 AC can be scheduled independently of S-BL.TESTENV (harness skeleton exists).
VP-042 AC requires S-BL.TESTENV to deliver `internal/testenv.NewLoopback` first.

## Backlog Status

| Field | Value |
|-------|-------|
| Created | 2026-07-06 |
| Origin | VP-041/VP-042 Phase-6 audit (UNPROVEN-BLOCKED); OBS-VP-BENCH drift row; Wave-4 audit L-2 |
| VP traces | VP-041 (tick jitter), VP-042 (keystroke-echo; unowned since RULING-002) |
| BC traces | BC-2.01.001 (NFR-009/NFR-001), BC-2.02.001 (multipath for VP-042) |
| Status transitions | (none yet) |
