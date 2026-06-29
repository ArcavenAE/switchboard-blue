---
artifact_id: S-HRD.02-daemon-logging-infrastructure
document_type: story
level: ops
story_id: S-HRD.02
title: "daemon logging infrastructure + security-event emission"
status: draft
producer: product-owner
timestamp: 2026-06-29T00:00:00
phase: 2
epic: E-HRD
wave: unscheduled
priority: P1
scope_phase: E
estimated_points: TBD
version: "0.1-stub"
bc_traces:
  - BC-2.07.004
vp_traces: []
subsystems: [network-management]
architecture_modules: [internal/mgmt, cmd/switchboard]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-W5.01]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.004.md'
draft_origin:
  source: S-W5.01 v1.6 AC-003 deferral note (Architect Ruling 1)
  ruling: Architect Ruling 1 — S-W5.01 mgmt-server convergence
  deferred_from: S-W5.01 AC-003 (post-auth structural guard, VP-065)
  rationale: >
    BC-2.07.004 PC-3 / EC-004 requires the server to emit a security-event log when
    a post-auth challenge_response is received and rejected. S-W5.01 AC-003 covers
    only the fail-closed connection control (AUTH_FAIL + E-ADM-010 + close), which is
    fully implemented and tested via VP-065. The logging obligation is deferred here
    because the daemon currently has NO structured logging infrastructure — only a
    one-off stdlib log.New(stderr) router scaffolding in cmd/switchboard/access.go.
    Establishing a daemon-wide slog seam first avoids baking ad-hoc log.Print calls
    into mgmt.Server before the logging architecture is settled.
  severity: MEDIUM
  bc_ref: BC-2.07.004 PC-3 / EC-004
acceptance_criteria_count: 0
---

# S-HRD.02: Daemon Logging Infrastructure + Security-Event Emission

> **STATUS: STUB.** This story is a hardening follow-up placeholder created per
> Architect Ruling 1 (S-W5.01 mgmt-server convergence) and the AC-003 deferral note
> in S-W5.01 v1.6. Acceptance criteria are intentionally absent. When promoted to a
> hardening-pass wave, story-writer will flesh out full ACs, tasks, file structure,
> and architecture mapping.
>
> This is the **owning story** for the BC-2.07.004 PC-3 / EC-004 "logs a security
> event" deferral and the "Known Scope Gaps" entry in BC-2.07.004 v1.6.

## Deferral Rationale

S-W5.01 AC-003 asserts the post-auth structural guard: a connection in the
authenticated state that sends a second `challenge_response` receives AUTH_FAIL
(E-ADM-010) and is closed. This connection-control obligation is fully implemented
and tested via VP-065.

BC-2.07.004 PC-3 / EC-004 additionally requires the server to **emit a security-event
log** when this guard fires. This logging obligation is deferred because:

1. The daemon currently has **no structured logging infrastructure**. The only logging
   present is a one-off `log.New(os.Stderr, ...)` scaffolding in
   `cmd/switchboard/access.go` (router mode stub). No logger is injected into or
   available inside `internal/mgmt`.
2. Introducing ad-hoc `log.Print` calls directly into `mgmt.Server` before the
   logging architecture is settled would create technical debt and inconsistent
   log formatting across the daemon.
3. The practical security risk is bounded: the connection is fail-closed (AUTH_FAIL +
   close) regardless of whether the event is logged. The logging gap is observability
   debt, not a correctness or security-control gap.

## Scope

1. **Introduce a daemon-wide structured logging seam** using `log/slog` (stdlib,
   Go 1.21+):
   - Add a `logger *slog.Logger` field to `mgmt.Server` (or equivalent option type).
   - Add a `WithLogger(l *slog.Logger)` option to `mgmt.NewServer` so callers can
     inject a logger. Defaults to `slog.Default()` (no breaking change).
   - Wire the logger injection point into `runAccess` in `cmd/switchboard/access.go`
     so the access daemon passes a structured logger to `mgmt.NewServer`.

2. **Emit the deferred security-event log** on the post-auth guard path in
   `mgmt.go`:
   - When `handleConnection` detects an authenticated connection sending a second
     `challenge_response` (the AC-003 guard path), log a structured security event
     via the injected logger before sending AUTH_FAIL + close.
   - Log fields must include at minimum: event type (`security_event`), reason
     (`post_auth_challenge_response`), remote address, and timestamp.
   - This satisfies BC-2.07.004 PC-3 / EC-004 for the logging obligation deferred
     from S-W5.01.

3. **Cross-reference note:** The daemon currently has NO structured logging infra —
   only `stdlib log.New(stderr)` one-off router scaffolding in
   `cmd/switchboard/access.go`. This story establishes the daemon-wide slog seam.
   Other daemon paths (router, console, control) that need structured logging should
   follow the pattern introduced here.

**Scope out:** BC edits (beyond the BC-2.07.004 v1.6 "Known Scope Gaps" annotation
already in place), changes to `cmd/sbctl`, changes to `internal/config`, any other
security-event log paths not explicitly listed above.

## Behavioral Contract Anchors

| BC Reference | Requirement |
|-------------|-------------|
| BC-2.07.004 PC-3 | Post-auth challenge_response guard: fail-closed (AUTH_FAIL + E-ADM-010 + close) AND logs a security event. The connection-control half is satisfied by S-W5.01; this story satisfies the logging half. |
| BC-2.07.004 EC-004 | Security event emitted when post-auth structural guard fires. Deferred from S-W5.01 AC-003 per Architect Ruling 1. |

## Narrative

- **As a** security-operations engineer monitoring daemon logs
- **I want** the management server to emit a structured security-event log when the
  post-auth guard fires (an authenticated connection sends a second challenge_response)
- **So that** anomalous client behavior is observable in structured logs without
  requiring a pcap, even though the connection is already fail-closed

## Acceptance Criteria (sketch — expand when scheduled)

> Full ACs to be written when this story is promoted from stub to a scheduled wave.
> The sketch below captures the intent; it is NOT an implementation contract.

- AC-001: `mgmt.NewServer` accepts an optional `*slog.Logger` (via `WithLogger` option
  or constructor param); defaults to `slog.Default()`.
- AC-002: When the post-auth guard path fires in `handleConnection`, the server calls
  `logger.Warn(...)` (or `logger.Error(...)`) with structured fields: `event=security_event`,
  `reason=post_auth_challenge_response`, `remote_addr=<addr>`.
- AC-003: `runAccess` injects a `*slog.Logger` into `mgmt.NewServer`.
- AC-004: Tests verify the security-event log is emitted (inject a `slog.Handler` that
  captures records; assert the expected record is present after triggering the guard path).

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 0.1-stub | 2026-06-29 | product-owner | Initial stub — Architect Ruling 1 (S-W5.01 mgmt-server convergence); deferred BC-2.07.004 PC-3 / EC-004 security-event log from S-W5.01 AC-003; daemon slog seam + mgmt.Server logger injection + access.go wiring; MEDIUM severity. |
