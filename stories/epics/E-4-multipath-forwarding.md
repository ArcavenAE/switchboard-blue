---
artifact_id: E-4-multipath-forwarding
document_type: epic
level: ops
epic_id: E-4
version: "1.0"
status: pending
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
scope_phase: E
priority: P0
bc_traces:
  - BC-2.02.001
  - BC-2.02.002
  - BC-2.02.003
  - BC-2.02.004
  - BC-2.02.005
  - BC-2.02.006
  - BC-2.02.008
  - BC-2.02.009
subsystems: [multipath-forwarding]
architecture_modules: [internal/multipath, internal/arq, internal/replay, internal/paths, internal/routing]
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-03-routing-engine.md'
---

# E-4: Multi-Path Forwarding (Reliability Layer)

## Goal

Deliver duplicate-and-race multipath forwarding, receiver deduplication, per-path
RTT/loss tracking, upstream idempotent replay, downstream ARQ with piggybacked ACK
and SACK bitmap, TLPKTDROP, split-horizon loop prevention, and bounded drop cache.
This is the reliability layer that makes Switchboard resilient to path failures.

## BCs

| BC | Title | Priority |
|----|-------|---------|
| BC-2.02.001 | Duplicate-and-race: same frame sent on two fastest paths simultaneously | P0 |
| BC-2.02.002 | Receiver delivers first-arriving copy and silently discards subsequent duplicates | P0 |
| BC-2.02.003 | Per-path RTT and loss tracked via keep-alive probes; paths ranked by quality | P0 |
| BC-2.02.004 | Upstream idempotent replay window: each frame carries last N keystrokes | P0 |
| BC-2.02.005 | Downstream ARQ with piggybacked ACK and SACK bitmap | P0 |
| BC-2.02.006 | TLPKTDROP terminates overdue downstream frames and signals degradation | P0 |
| BC-2.02.008 | Router split-horizon prevents frames being forwarded back toward arrival interface | P0 |
| BC-2.02.009 | Bounded drop cache suppresses looping duplicate frames by checksum | P0 |

## Subsystems Touched

- SS-02 multipath-forwarding (primary)

## Estimated Stories

4 stories: S-4.01 (paths + multipath dispatch), S-4.02 (upstream replay), S-4.03 (downstream ARQ + TLPKTDROP), S-4.04 (split-horizon + drop cache)
