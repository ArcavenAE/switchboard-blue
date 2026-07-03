---
artifact_id: S-BL.ADMIN-RECOVER-WIRE
document_type: story
level: ops
story_id: S-BL.ADMIN-RECOVER-WIRE
version: "1.0"
title: "admin recover wire: sbctl admin recover dispatch + daemon handler registration"
status: draft
producer: product-owner
timestamp: 2026-07-03T00:00:00
modified: 2026-07-03T00:00:00
phase: 2
epic: E-7
wave: backlog
priority: P2
scope_phase: E
estimated_points: TBD
bc_traces:
  - BC-2.07.001
  - BC-2.05.004
depends_on: []
blocks: []
acceptance_criteria_count: 0
provenance:
  finding: F-P5P5-A-002
  spec_annotation: interface-definitions.md v1.18 PENDING-S-BL.ADMIN-RECOVER-WIRE
  adjudication: annotate-and-defer (consistent with five prior wire deferrals)
---

# S-BL.ADMIN-RECOVER-WIRE: Admin Recover Wire — sbctl Dispatch + Daemon Handler Registration

> **STATUS: DRAFT BACKLOG STUB.** Acceptance criteria, file structure, task list, and
> architecture mapping will be fleshed out when this story is scheduled.

## Context

`cmd/sbctl/admin.go` `runAdmin` switch covers `key | list-keys | svtn`; its default arm
returns `admin: unknown subcommand %q` (exit 2). `BuildAdminHandlers` in
`cmd/switchboard/admin_handlers.go:127-134` registers handlers for `admin.svtn.create`,
`admin.svtn.destroy`, `admin.key.register`, `admin.key.revoke`, `admin.key.expire`, and
`admin.list-keys` — but not `admin.recover`.

`interface-definitions.md` §119-125 (v1.18) specifies the emergency recovery subcommand
as first-class operator surface:

```
sbctl admin recover --svtn <id> --bootstrap-key <path> --confirm <svtn-short-id> | --yes
```

Exit codes: `0=ok`, `E-ADM-014` (bootstrap mismatch). Authority: bootstrap key only (set
at SVTN creation per BC-2.07.001 Inv-3). Confirm gate: per §128-133, the `--confirm` or
`--yes` flag is required on all destructive admin operations; `recover` is named among them.

Operators invoking `sbctl admin recover` on current builds receive exit 2 (unknown-subcommand)
with no explanation that the feature is not yet delivered. This is the gap adjudicated as
annotate-and-defer in Phase 5 Pass 5 (F-P5P5-A-002).

## BC Anchors

| BC | Why anchored |
|----|-------------|
| BC-2.07.001 | §125 authority cell: "Requires bootstrap key (set at SVTN creation per BC-2.07.001)"; Inv-3 establishes bootstrap-only authority for the class of operations that modify SVTN lifecycle state when all control keys are lost. `admin.recover` is the emergency path for that exact condition. |
| BC-2.05.004 | Confirm-gate semantics for destructive admin operations (§128-133): the five-path confirm gate (`--confirm=<svtn-short-id>` / interactive prompt / `--yes` / collision error E-CFG-012 / non-TTY guard E-CFG-013) applies to `recover` as a named destructive operation. BC-2.05.004 owns the confirm-gate contract for admin key operations; §128 cross-references it for all destructive admin surfaces. |

## Scope (at scheduling time)

1. Wire `runAdmin` switch arm for `recover` in `cmd/sbctl/admin.go`, forwarding to a new
   `runAdminRecover(ctx, args)` dispatcher.
2. Parse flags: `--svtn <id>` (required), `--bootstrap-key <path>` (required),
   `--confirm <svtn-short-id>` or `--yes` (confirm-gate per §128-133).
3. Register `admin.recover` handler in `BuildAdminHandlers`
   (`cmd/switchboard/admin_handlers.go:127-134`).
4. Implement daemon handler: load bootstrap key from path, verify against stored SVTN
   bootstrap fingerprint, recover admitted-key state; return `E-ADM-014` on mismatch.
5. Confirm-gate enforcement: validate `--confirm` token or `--yes`; return E-CFG-012 if
   both flags are combined; return E-CFG-013 (exit 2) if no TTY and neither flag supplied.
6. Unit and integration tests traced to BC-2.07.001 (bootstrap authority) and BC-2.05.004
   (confirm gate).

## Draft Acceptance Criteria

### AC-001 (traces to BC-2.07.001 Inv-3 — bootstrap-only authority)
`sbctl admin recover --svtn <id> --bootstrap-key <path> --confirm <svtn-short-id>` with the
correct bootstrap key succeeds (exit 0). A mismatched bootstrap key returns E-ADM-014 (exit 1).

### AC-002 (traces to BC-2.05.004 confirm-gate — §128-133)
`--confirm` and `--yes` are mutually exclusive: combining them returns E-CFG-012 (exit 2).
In a non-interactive session (no TTY) where neither flag is supplied, exit is E-CFG-013
(exit 2). When `--yes` is supplied, a warning is emitted to stderr
(`"WARNING: --yes bypasses confirmation; ensure correct --svtn target before scripting"`).

### AC-003 (traces to BC-2.07.001 Inv-3 — bootstrap-only; §125 exit-code column)
A non-bootstrap key invoking `admin.recover` (authenticated at the mgmt layer but not the
bootstrap key) returns E-ADM-009 (insufficient authority), exit 1.

### AC-004 (traces to BC-2.07.001 — daemon dispatch registration)
`BuildAdminHandlers` registers `admin.recover`; invoking `sbctl admin recover` on a running
daemon no longer returns `E-RPC-010: unknown command: admin.recover` (exit 2). The
`runAdmin` default arm no longer matches `recover`.

## Open Design Obligations (must be resolved before scheduling)

### 1. Recovery semantics — what does "recover" actually do?

§119-125 describes the surface but not the daemon-side semantics: when the bootstrap key is
verified, what state is restored? Candidate interpretations:
- Revoke all admitted keys and re-admit the bootstrap key as the sole control key (tabula
  rasa for the SVTN's admitted set).
- Return a new bootstrap token usable for re-registering keys.
- Other.

An architect ruling or BC-2.07.001 amendment is required before implementation.

### 2. `--svtn <id>` — ID or name?

§125 uses `--svtn <id>` while other admin subcommands use `--svtn <id>` (SVTN hex ID) vs
`--name=<svtn-name>` (name string). The flag name and type (ID vs name) must be confirmed
against the wire contract before the sbctl dispatcher and handler args struct are written.

## Provenance

- **Finding:** F-P5P5-A-002 (Phase 5 Pass 5 Adv-A, 2026-07-03) — `sbctl admin recover`
  fully specified in interface-definitions.md §119-125 but neither CLI nor daemon implement
  it; no PENDING/DRIFT annotation covered the gap. Adjudicated annotate-and-defer.
- **Spec annotation:** interface-definitions.md v1.18 §121, §126 PENDING-S-BL.ADMIN-RECOVER-WIRE
  annotation added as part of Burst 21 remediation.
- **Adjudication:** annotate-and-defer — same pattern as S-BL.PING-VERSION-WIRE (before
  wont-fix retirement), S-BL.SVTN-LIST-WIRE (before wont-fix retirement), S-BL.DISCOVERY-WIRE,
  S-BL.PATH-TRACKER-WIRING, and S-BL.PATH-FAILED-STATUS. Unlike those three, this surface
  is NOT being withdrawn from spec — emergency recovery is a required operator capability.
  The adjudication is defer (not wont-fix): the surface will be implemented when scheduled.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-03 | Draft backlog stub created per F-P5P5-A-002 adjudication (annotate-and-defer). Interface-definitions.md v1.18 §119-126 PENDING-S-BL.ADMIN-RECOVER-WIRE annotation is the spec-side closure; this stub is the backlog-side closure. BC anchors: BC-2.07.001 (bootstrap authority), BC-2.05.004 (confirm gate). Two open design obligations noted (recovery semantics, flag type). |
