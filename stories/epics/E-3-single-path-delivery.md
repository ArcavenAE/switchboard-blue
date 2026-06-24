---
artifact_id: E-3-single-path-delivery
document_type: epic
level: ops
epic_id: E-3
version: "1.0"
status: pending
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
scope_phase: E
priority: P0
bc_traces:
  - BC-2.04.001
  - BC-2.04.002
  - BC-2.04.003
  - BC-2.04.004
  - BC-2.04.005
  - BC-2.04.006
  - BC-2.05.003
subsystems: [session-access, admission-security]
architecture_modules: [internal/tmux, internal/session]
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-01-core-services.md'
---

# E-3: Single-Path Session Access (E-router MVP Core)

## Goal

Deliver end-to-end session access: access node attaches to tmux via control mode
(with PTY fallback), consoles attach/detach, read-only enforcement, multi-console
fan-out, and Tier 2 per-session authorization. This is the primary user value of
Switchboard MVP.

## BCs

| BC | Title | Priority |
|----|-------|---------|
| BC-2.04.001 | Access node connects to local tmux via control mode and publishes sessions over SVTN | P0 |
| BC-2.04.002 | Access node falls back to PTY proxy when tmux control mode unavailable | P0 |
| BC-2.04.003 | Console attaches to session by name; receives downstream stream and sends upstream keystrokes | P0 |
| BC-2.04.004 | Console detach releases session without closing it; session continues on access node | P0 |
| BC-2.04.005 | Read-only console receives downstream stream; upstream keystrokes are rejected by access node | P0 |
| BC-2.04.006 | Two or more consoles may subscribe to the same session output simultaneously | P0 |
| BC-2.05.003 | Per-session Tier 2 authorization enforced by access node, not router | P0 |

## Subsystems Touched

- SS-04 session-access (primary)
- SS-05 admission-security (Tier 2 auth)

## Estimated Stories

3 stories: S-3.01 (tmux control + PTY fallback), S-3.02 (session attach/detach + multi-console), S-3.03 (Tier 2 authorization + read-only enforcement)
