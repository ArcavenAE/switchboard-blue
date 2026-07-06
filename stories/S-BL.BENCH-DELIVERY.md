---
artifact_id: S-BL.BENCH-DELIVERY
document_type: delivery-ledger
story_id: S-BL.BENCH
pr_number: 109
pr_url: https://github.com/ArcavenAE/switchboard-blue/pull/109
branch: feat/s-bl-bench
base: develop
commits:
  - sha: b0d3671
    msg: "test(bench): VP-041 gate — BenchmarkHalfChannelTickJitter b.Errorf ≤2ms (S-BL.BENCH AC-001)"
  - sha: cf8fca4
    msg: "test(bench): VP-042 harness + just bench recipe (S-BL.BENCH AC-002)"
timestamp: 2026-07-06
---

# S-BL.BENCH Delivery Ledger

## What Was Built

### VP-041 (AC-001): BenchmarkHalfChannelTickJitter — b.Errorf gate added

File: `internal/halfchannel/halfchannel_test.go`

The existing Phase-3 benchmark (from S-1.02 AC-005) had `b.ReportMetric` only.
This story adds the `b.Errorf` gate per VP-041 v1.2 Phase-6 enforcement:

```go
if b.N >= 100 && p99 > maxJitter {
    b.Errorf("tick p99 jitter %v exceeds NFR-009 limit %v (VP-041)", p99, maxJitter)
}
```

Gate activates only at ≥ 100 samples (statistical minimum for a valid p99).
Phase stratification preserved: Phase-3 path is unchanged; Phase-6 adds the gate.
DIAGNOSTIC per ADR-007 — not wired to required CI.

Also fixed a `gofumpt` formatting issue in the const block (golangci-lint clean).

### VP-042 (AC-002): BenchmarkKeystrokeEcho_P99 — new file

File: `internal/bench/keystroke_echo_bench_test.go` (new)

500-sample in-process loopback benchmark using `session.AccessNode` + `echoSink`.
Reports `p99_rtt_ms` and enforces VP-042 gate (≤ 100ms p99) via `b.Errorf`.

**Ruling Divergence:** VP-042.md proof skeleton calls `testenv.NewLoopback`
(S-BL.TESTENV, backlog, not delivered). This harness uses an equivalent
in-process path. The loopback is a lower bound (no arq, no multipath, no tick
scheduling). VP-042 on the full stack is gated on S-BL.TESTENV.

### `just bench` recipe (justfile)

New recipe runs both benchmarks sequentially with hardware info header.
DIAGNOSTIC only — not wired to required CI per ADR-007.

## Benchmark Results

**Hardware:** Apple M1, 8 CPUs (logical), macOS arm64

### VP-041 — BenchmarkHalfChannelTickJitter (1000 ticks, `-benchtime=1000x`)

| Run | jitter_p99_ms | Gate (≤ 2ms) |
|-----|--------------|--------------|
| 1   | 1.112        | PASS         |
| 2   | 1.066        | PASS         |
| 3   | 1.063        | PASS         |
| **Mean** | **1.080 ms** | **PASS** |

VP-041 SLO bound ≤ 2ms: **MET** on this hardware.
Note: developer hardware (M1) previously measured at 2.111ms p99 under load
(the original reason AC-005 deferred the gate to Phase-6). These runs were
conducted with no competing load and show the system well within budget.

### VP-042 — BenchmarkKeystrokeEcho_P99 (500 samples, `-benchtime=1x`)

| Run | p99_rtt_ms    | Gate (≤ 100ms) |
|-----|--------------|----------------|
| 1   | 0.000792     | PASS           |
| 2   | 0.002833     | PASS           |
| 3   | 0.000916     | PASS           |
| **Mean** | **~0.002 ms** | **PASS** |

VP-042 SLO bound ≤ 100ms: **MET** on this hardware.
Values are in µs scale because this is a lower-bound in-process loopback
(no network, no arq, no tick scheduling). The 100ms budget accommodates
the real stack's 10ms upstream + 50ms downstream tick cadence.
Full-stack VP-042 verification requires S-BL.TESTENV.

## Pre-PR Gate Results

| Gate | Result |
|------|--------|
| `go test -race ./...` | All packages PASS (14 pkgs, 0 failures) |
| `just smoke-quick` | 14/14 sentinel invariants PASS |
| `golangci-lint run ./internal/bench/... ./internal/halfchannel/...` | 0 issues |
| `go test -bench=. -benchtime=1x ./...` | Compile-proof PASS |

## VP Bounds Verdict

| VP | Bound | Local Result | Met? | Notes |
|----|-------|-------------|------|-------|
| VP-041 | ≤ 2ms p99 jitter (1000 ticks) | 1.080ms mean | YES | Developer M1 under no load |
| VP-042 | ≤ 100ms p99 RTT (500 samples) | ~0.002ms mean | YES | Lower-bound loopback only; full-stack requires S-BL.TESTENV |

## What the Prior Agent Built vs What Was Changed

The prior agent built all three files before the session died on API stalls:
- `internal/bench/keystroke_echo_bench_test.go` — complete, kept as-is
- `internal/halfchannel/halfchannel_test.go` — modified correctly; only fix
  applied was `gofumpt` const block alignment (linter requirement)
- `justfile` — complete, kept as-is

No rewrites were required. The prior agent's work was correct.

## AC Coverage

| AC | Status | Evidence |
|----|--------|---------|
| AC-001 (VP-041 gate) | IMPLEMENTED | `b.Errorf` at `b.N ≥ 100` gate in `halfchannel_test.go` |
| AC-002 (VP-042 harness) | IMPLEMENTED (lower-bound) | `BenchmarkKeystrokeEcho_P99` in `internal/bench/` |
| AC-003 (VP-041 lock) | DEFERRED to coordinator | VP locks set by coordinator, not story implementer |
| AC-004 (VP-042 lock) | DEFERRED to coordinator | Same; VP-042 lock also gated on S-BL.TESTENV ruling |

## PR

- Number: #109
- URL: https://github.com/ArcavenAE/switchboard-blue/pull/109
- Base: develop
- Head: feat/s-bl-bench
- Commits: b0d3671, cf8fca4
