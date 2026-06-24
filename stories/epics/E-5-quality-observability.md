---
artifact_id: E-5-quality-observability
document_type: epic
level: ops
epic_id: E-5
version: "1.0"
status: pending
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
scope_phase: E
priority: P1
bc_traces:
  - BC-2.06.001
  - BC-2.06.002
  - BC-2.06.003
subsystems: [quality-observability]
architecture_modules: [internal/metrics, internal/paths]
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
---

# E-5: Quality Observability

## Goal

Deliver the green/yellow/red quality indicator derived from per-path RTT and loss
measurements, with hysteresis (3-measurement threshold), missing-frame degradation
signals, and per-path metrics queryable via sbctl. Operators need visibility into
connection quality; this is the observability layer.

## BCs

| BC | Title | Priority |
|----|-------|---------|
| BC-2.06.001 | Quality indicator (green/yellow/red) derived from measured path latency and loss | P1 |
| BC-2.06.002 | Missing expected frame is a degradation signal triggering indicator downgrade | P1 |
| BC-2.06.003 | Per-path RTT and loss metrics queryable via sbctl | P1 |

## Subsystems Touched

- SS-06 quality-observability (primary)

## Estimated Stories

2 stories: S-5.01 (quality indicator logic), S-5.02 (sbctl metrics query)
