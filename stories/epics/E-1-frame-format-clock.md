---
artifact_id: E-1-frame-format-clock
document_type: epic
level: ops
epic_id: E-1
version: "1.0"
status: pending
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
scope_phase: E
priority: P0
bc_traces:
  - BC-2.01.001
  - BC-2.01.002
  - BC-2.01.003
  - BC-2.01.004
  - BC-2.01.005
  - BC-2.01.006
  - BC-2.01.007
subsystems: [session-networking]
architecture_modules: [internal/frame, internal/halfchannel, internal/admission]
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-02-protocol-stack.md'
---

# E-1: Frame Format + Timeslice Clock (Foundation)

## Goal

Deliver the wire-format codec (`internal/frame`) and the timeslice clock
state machine (`internal/halfchannel`) that are the foundation of every
session. Every other module depends on correct frame encoding and tick
regularity. Also includes cryptographic node address derivation and session
continuity via re-authentication (BC-2.01.006, BC-2.01.007).

## BCs

| BC | Title | Priority |
|----|-------|---------|
| BC-2.01.001 | Timeslice clock fires on every tick regardless of data availability | P0 |
| BC-2.01.002 | Empty-tick frame is a valid liveness signal | P0 |
| BC-2.01.003 | Upstream and downstream half-channels operate with independent clocks and sequence spaces | P0 |
| BC-2.01.004 | Frame outer-header encoding and decoding at 44-byte fixed layout | P0 |
| BC-2.01.005 | Channel header is opaque to routers — parseable only by endpoints | P0 |
| BC-2.01.006 | Session identity is cryptographic: node address derived from hash(SVTN-ID, public-key) | P0 |
| BC-2.01.007 | Session continuity survives IP address change via cryptographic re-authentication | P0 |

## Subsystems Touched

- SS-01 session-networking (primary)

## Estimated Stories

3 stories: S-1.01 (frame codec), S-1.02 (half-channel clock), S-1.03 (node identity + session continuity)
