---
artifact_id: E-6-control-plane-cli
document_type: epic
level: ops
epic_id: E-6
version: "1.0"
status: pending
producer: story-writer
timestamp: 2026-06-24T00:00:00
phase: 2
scope_phase: E
priority: P0
bc_traces:
  - BC-2.05.004
  - BC-2.07.001
  - BC-2.07.002
  - BC-2.07.003
  - BC-2.09.003
subsystems: [network-management, admission-security]
architecture_modules: [internal/svtnmgmt, internal/config, cmd/sbctl]
inputDocuments:
  - '.factory/specs/behavioral-contracts/BC-INDEX.md'
  - '.factory/specs/architecture/ARCH-05-cli-and-api.md'
---

# E-6: Control Plane + CLI

## Goal

Deliver SVTN lifecycle management (create/destroy, control node bootstrap),
key lifecycle (register/revoke/expire via sbctl admin — note: fixes Phase 1
drift F-P8-001, F-P8-006 using canonical `sbctl admin` surface), unified
sbctl CLI with OpenSSH auth, connection error reporting, and config validation
with actionable startup errors.

## BCs

| BC | Title | Priority |
|----|-------|---------|
| BC-2.05.004 | Key lifecycle: register, revoke, and expire admission and session-authorization keys | P0 |
| BC-2.07.001 | Control node creates and destroys SVTNs; first control key bootstrapped locally | P2 |
| BC-2.07.002 | sbctl unified CLI for all four daemon types with OpenSSH key authentication | P2 |
| BC-2.07.003 | sbctl reports clear connection error when target daemon is unreachable | P0 |
| BC-2.09.003 | Router startup fails cleanly on malformed config with actionable error message | P0 |

## Phase 1 Drift Addressed

- F-P8-001: BC-2.05.004 story uses canonical `sbctl admin` surface (not removed `sbctl svtn keys`)
- F-P8-006: BC-2.05.007 test vector uses `sbctl svtn keys list` / `sbctl admin list-keys` canonical form

## Subsystems Touched

- SS-07 network-management (primary)
- SS-05 admission-security (key lifecycle)
- SS-09 deployment-operations (config validation)

## Estimated Stories

3 stories: S-6.01 (config validation + startup error), S-6.02 (SVTN lifecycle + key lifecycle), S-6.03 (sbctl CLI + connection error)
