---
artifact_id: S-BL.CLI-SURFACE-COMPLETION
document_type: story
level: ops
story_id: S-BL.CLI-SURFACE-COMPLETION
version: "1.0"
title: "CLI surface completion: dispatch wire for paths ping, router reload, router drain, svtn destroy, svtn status"
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
  - BC-2.09.001
  - BC-2.09.002
  - BC-2.07.001
depends_on: []
blocks: []
acceptance_criteria_count: 0
provenance:
  finding: F-P5P6-A-005
  spec_annotation: interface-definitions.md v1.19 PENDING-S-BL.CLI-SURFACE-COMPLETION
  adjudication: annotate-and-defer
---

# S-BL.CLI-SURFACE-COMPLETION: CLI Surface Completion — Dispatch + Wire for Five Unimplemented Verbs

> **STATUS: DRAFT BACKLOG STUB.** Acceptance criteria, file structure, task list, and
> architecture mapping will be fleshed out when this story is scheduled.

## Context

Five `sbctl` verbs are specified in `interface-definitions.md` v1.19 but have no CLI
dispatch case arm — they return unknown-subcommand usage errors (exit 2) on current builds
(verified post-PR #65):

| Verb | Spec §§ | Current behavior (post-#65) |
|------|---------|-----------------------------|
| `sbctl paths ping --router=<addr>` | §77 | `paths` case dispatches `list` only; `ping` returns usage error, exit 2 |
| `sbctl router reload` | §82 | `router` case handles `metrics`/`status` only; `reload` returns unknown-subcommand usage error, exit 2 |
| `sbctl router drain` | §83 | `router` case handles `metrics`/`status` only; `drain` returns unknown-subcommand usage error, exit 2 |
| `sbctl svtn destroy --id=<svtn_id>` | §60 | No `svtn` top-level case arm in `main.go`; every `sbctl svtn ...` hits the default arm, exit 2 |
| `sbctl svtn status --id=<svtn_id>` | §62 | Same — no `svtn` top-level case arm, exit 2 |

These verbs were annotated collectively as `PENDING-S-BL.CLI-SURFACE-COMPLETION` in the
v1.19 spec-side remediation during Burst 23, per F-P5P6-A-005.

**Explicitly out of scope for this story:**

- `sbctl svtn list` — won't-fix (S-BL.SVTN-LIST-WIRE); surface removed from the canonical
  CLI, returns unknown-subcommand exit 2.
- `sbctl sessions attach/detach/status` — covered by S-BL.DISCOVERY-WIRE.
- `sbctl admin recover` — covered by S-BL.ADMIN-RECOVER-WIRE.

## BC Anchors

| BC | Verb(s) governed | Why anchored |
|----|-----------------|-------------|
| BC-2.09.001 | `router reload` | `sbctl router reload` (equivalently SIGHUP) is the operator trigger for the E-to-PE config reload described in BC-2.09.001. PC-1 specifies that the router reloads its config on receiving this signal; PC-2 requires session state is not lost during reload; PC-3 specifies the active-session-continuity guarantee. The CLI dispatch arm is the operator-facing surface for the BC's precondition. |
| BC-2.09.002 | `router drain` | `sbctl router drain` (equivalently SIGTERM) is the operator trigger for the graceful drain sequence described in BC-2.09.002. The BC's happy-path scenario explicitly names `sbctl router drain` as the operator trigger; the CLI dispatch arm is the surface through which operators invoke the drain protocol. |
| BC-2.07.001 | `svtn destroy` | BC-2.07.001 PC-3 (and the Destroy authority ruling RULING-W6TB-A) governs SVTN lifecycle destruction. `sbctl svtn destroy` is the non-admin form of this verb; the canonical form `sbctl admin svtn destroy` is already wired. The `svtn` top-level dispatch case arm is needed to route either `sbctl svtn destroy` (top-level form, §60) or return a clear "use sbctl admin svtn destroy" message. The confirm-gate contract from BC-2.05.004 may also apply depending on architect ruling during scheduling (open obligation below). |

### Verbs without a governing BC

**`sbctl paths ping --router=<addr>` (§77):** No BC currently governs a one-shot on-demand
RTT probe via CLI. BC-2.06.003 covers the per-path metrics surface (the continuous
keep-alive-derived RTT/loss system) but not a discrete operator-triggered latency probe.
BC-2.07.002 covers the unified CLI operator interface at a general level. No BC
postcondition specifies what `paths ping` must produce, what wire verb it calls, or what
error codes apply. This is an open design obligation — see below.

**`sbctl svtn status --id=<svtn_id>` (§62):** No BC governs a read-only SVTN status query.
BC-2.07.001 covers SVTN lifecycle create/destroy; BC-2.07.002 covers the unified sbctl
CLI interface generally. No BC specifies what status fields `svtn status` returns, what
wire verb it calls, or what the response schema is. This is an open design obligation —
see below.

## Scope (at scheduling time)

1. Add a `svtn` top-level case arm to `cmd/sbctl/main.go` dispatching sub-verbs:
   - `destroy` → `runSvtnDestroy(ctx, args)`
   - `status` → `runSvtnStatus(ctx, args)` (pending BC design obligation below)
   - unknown sub-verb → usage error, exit 2
2. Wire `paths ping` dispatch in `cmd/sbctl/main.go` `paths` case arm.
3. Add `router reload` and `router drain` case arms to the `router` switch
   in `cmd/sbctl/main.go`.
4. Register wire handlers for each new verb in the daemon (exact handler registration
   points TBD at scheduling; `BuildAdminHandlers` or new routing table entries
   depending on architect ruling).
5. Flag parsing and wire request construction per spec §60, §77, §82-83.
6. Unit and integration tests traced to the anchored BCs and any new BCs commissioned
   for the two ungoverned verbs.

## Open Design Obligations (must be resolved before scheduling)

### 1. `paths ping` — no governing BC

§77 specifies the CLI surface (`sbctl paths ping --router=<addr>`) and describes it as a
"one-shot RTT probe," but no BC defines:
- What wire verb the CLI calls (a new `paths.ping` RPC, or reuse of `paths.list`?).
- What the response schema is (a single RTT measurement? a series?).
- What happens if the router is unreachable vs. responds but latency is high.
- Which error codes apply.
- Whether BC-2.06.003 should be extended or a new BC-2.06.004 commissioned.

An architect ruling or new BC is required before implementation.

### 2. `svtn status` — no governing BC

§62 specifies the CLI surface (`sbctl svtn status --id=<svtn_id>`) but no BC defines:
- What fields the response includes (admitted key count? active session count? health
  indicators? creation timestamp?).
- What wire verb the CLI calls (a new `admin.svtn.status` RPC? or `svtn.query`?).
- Authority requirements (any admitted role, or control only?).
- The response schema.
- Which error codes apply (is `E-SVTN-003` reused for not-found?).

An architect ruling or new BC (extending BC-2.07.001 with PC-4, or commissioning
BC-2.07.005) is required before implementation.

### 3. `svtn destroy` — confirm-gate applicability

§60 uses `--id=<svtn_id>` while the canonical `sbctl admin svtn destroy` uses
`--name=<svtn-name>`. It is unclear whether the `svtn` top-level form should:
- Accept the same `--confirm` gate as the `admin svtn destroy` path (BC-2.05.004 confirm
  semantics), or
- Simply redirect to `sbctl admin svtn destroy` with a clear usage message, treating
  the top-level form as a migration shim.

An architect ruling is required to resolve this before the dispatch arm and tests are
written.

### 4. `router reload` / `router drain` — daemon-side handler wire verb names

Interface-definitions.md does not specify the wire verb names for `router.reload` and
`router.drain` in the Registered Verbs table (§379-395). The handler registration names
must be confirmed (candidates: `router.reload`, `router.drain`) against the daemon's
`cmd/switchboard/` routing layer before the CLI is wired.

## Provenance

- **Finding:** F-P5P6-A-005 (Phase 5 Pass 6 Adv-A, 2026-07-03) — seven `sbctl` verbs
  specified without PENDING annotations; five collective-annotated here (paths ping,
  router reload, router drain, svtn destroy, svtn status); two others resolved separately
  (svtn list → won't-fix S-BL.SVTN-LIST-WIRE; sessions attach/detach/status →
  S-BL.DISCOVERY-WIRE).
- **Spec annotation:** interface-definitions.md v1.19 §60, §62, §77, §82, §83
  PENDING-S-BL.CLI-SURFACE-COMPLETION annotation added as part of Burst 23 spec-side
  remediation.
- **Adjudication:** annotate-and-defer — consistent with S-BL.ADMIN-RECOVER-WIRE
  and prior wire-gap deferrals. Two of the five verbs (paths ping, svtn status) have
  no governing BC and require design work before implementation can begin.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-03 | Draft backlog stub created per F-P5P6-A-005 adjudication (annotate-and-defer). Interface-definitions.md v1.19 PENDING-S-BL.CLI-SURFACE-COMPLETION annotation is the spec-side closure; this stub is the backlog-side closure. BC anchors: BC-2.09.001 (router reload), BC-2.09.002 (router drain), BC-2.07.001 (svtn destroy). Two verbs (paths ping, svtn status) have no governing BC — open design obligations noted. Four open design obligations logged (paths ping BC, svtn status BC, svtn destroy confirm-gate, reload/drain wire verb names). |
