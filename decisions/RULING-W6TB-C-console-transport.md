---
artifact_id: RULING-W6TB-C-console-transport
document_type: decision
level: ops
version: "1.0"
status: final
producer: architect
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-7.03]
closes_findings: []
---

# Ruling W6TB-C — S-7.03 Console Remote-Control Transport

**Question:** does console remote control (attach/detach/switch) use the sbctl
mgmt Unix-socket transport (S-6.03 pattern) or the SVTN data-plane channel
(BC-2.08.001 Inv-3: "same SVTN channel as regular traffic — no separate
out-of-band channel")?

---

## Decision

**S-7.03 uses the Unix-socket mgmt-plane transport (S-6.03 pattern). BC-2.08.001
Inv-3 must be patched to v1.2: retract "same SVTN channel as regular traffic — no
separate out-of-band channel" and replace with the management-plane transport
constraint. AC-004 (console daemon session-list view) and AC-005 (MissCount()
accessor) are split into a follow-on observability story (S-BL.CONSOLE-OBS) and
removed from S-7.03 scope.**

---

## Rationale

### 1. Transport contradiction analysis

BC-2.08.001 Inv-3 states: "Remote control commands use the same SVTN channel as
regular traffic — no separate out-of-band channel."

Every merged sbctl command (S-6.02, S-6.06, S-6.07) uses the Unix-socket mgmt
plane (`internal/mgmt`, ADR-012, ADR-006 JSON envelope). The S-6.03 pattern is
the established canonical transport for all `sbctl` commands. These are
architecturally incompatible with routing traffic over the SVTN data plane:

- The SVTN data plane uses the ARQ/multipath/half-channel stack in
  `internal/routing`. `sbctl` has no ARQ stack; it opens a Unix socket and
  speaks the JSON-over-Unix-socket protocol (ADR-006 / ADR-012).
- `internal/mgmt.Server` (the daemon side) dispatches all sbctl RPCs. There is
  no pathway for sbctl to inject commands into the SVTN data plane.
- Console daemon architecture (`internal/session`, `internal/tmux`) is classified
  as a boundary/effectful module (S-7.03 Architecture Mapping). It receives
  mgmt-plane RPCs from `internal/mgmt`; it has no mechanism to receive control
  messages via SVTN routing frames.

Implementing Inv-3 as written would require building a second control-message
protocol inside the SVTN data plane — a major architectural undertaking with no
precedent in the codebase and no CAP/BC grounding for the required protocol design.

### 2. Inv-3 intent reconstruction

BC-2.08.001 Inv-3's "same SVTN channel" language most likely intended to prevent
a scenario where remote console control bypasses SVTN authorization by using a
privileged out-of-band channel that ignores the SVTN admission check. The
management-plane Unix socket already satisfies this intent: it requires sbctl
authentication (the operator's key is checked by `internal/mgmt.Server` per
BC-2.07.002 / ADR-012). The console daemon's Tier-2 authorization (BC-2.08.001
Inv-1, Inv-2) is enforced by `internal/session` regardless of which transport
delivers the command. The security invariant is preserved.

The "same SVTN channel" phrasing is an implementation detail that was written
before the Wave-5 management-plane transport was established as the canonical sbctl
transport. It is an architectural anachronism in BC-2.08.001.

### 3. Consistency with all other sbctl commands

Pattern consistency is a hard architectural constraint (ARCH-05; ADR-006; ADR-012).
All operator-facing commands in the `sbctl` surface use the JSON-over-Unix-socket
transport. Introducing a second transport protocol exclusively for console remote
control would:

- Split the sbctl codebase into two transport stacks;
- Require `cmd/sbctl/console.go` to import the SVTN data-plane stack (forbidden
  per ARCH-08 §6.6 — `cmd/sbctl` is an effectful CLI and must not import
  `internal/routing`, `internal/arq`, `internal/multipath`, etc.);
- Break ADR-012's "single authenticated channel for all management RPCs" principle.

The architecture does not support the SVTN-channel interpretation.

---

## BC-2.08.001 Patch Required (v1.2)

BC-2.08.001 must be patched from v1.1 to v1.2 before S-7.03 is delivered.

**Invariant 3 — current (v1.1):**
> "Remote control commands use the same SVTN channel as regular traffic — no
> separate out-of-band channel."

**Invariant 3 — patch (v1.2):**
> "Remote control commands use the management-plane Unix-socket transport
> (ADR-006 / ADR-012) — the same authenticated channel used by all sbctl
> commands. No separate data-plane or out-of-band channel is introduced. The
> operator's key is authenticated by `internal/mgmt.Server`; console Tier-2
> authorization (Inv-1) is enforced by the console daemon's session layer
> regardless of transport."

**Changelog entry:**
> "v1.2 | 2026-07-01 | architect | W6TB-C Ruling: Inv-3 retracted and replaced.
> 'Same SVTN channel' was architecturally incompatible with the established
> JSON-over-Unix-socket management-plane transport used by all sbctl commands
> (ADR-006/ADR-012). Inv-3 now correctly states the management-plane transport.
> Security intent preserved: operator key authentication via internal/mgmt.Server;
> Tier-2 authorization via internal/session. S-7.03 is the implementing story."

**Story-spec update (S-7.03 v1.2):** add to Architecture Compliance Rules:
> "BC-2.08.001 Inv-3 patched to v1.2 (W6TB-C): transport is mgmt-plane
> Unix socket (ADR-006/ADR-012), not SVTN data plane."

---

## AC-004 and AC-005 Disposition: Split to S-BL.CONSOLE-OBS

### AC-004 problem

AC-004 refers to a "console daemon session-list view" that does not exist. The
console daemon (`internal/session`) has no session-list RPC; session listing is
owned by `sbctl sessions list` (S-3.02 / BC-2.04.003 surface). The console daemon
attaches, detaches, and switches — it does not enumerate sessions independently.
AC-004 as written requires a new architectural surface (console-scoped session
list with quality indicator) that is out of scope for a story implementing
attach/detach/switch transport.

### AC-005 problem

AC-005 references `QualityIndicator.MissCount()` as an accessor on
`internal/metrics`. No such accessor exists on `QualityIndicator` or on any
exported type in `internal/metrics`. The `missCount` concept maps to
`PathTracker.consecutiveMisses` (unexported) or `PathSnapshot.SampleCount` (a
different metric). Implementing this AC would require either:

- Exporting `consecutiveMisses` from `PathTracker` (a new `internal/paths`
  surface not yet designed or reviewed), or
- Defining what "MissCount" means in the context of the quality indicator
  (which is currently a separate `internal/metrics` type — `QualityIndicator`
  is the green/yellow/red state machine, not a path-level tracker).

This is a spec gap that predates the AC authorship and cannot be resolved without
a design pass.

### Resolution

Both AC-004 and AC-005 trace to BC-2.06.001 PC-5 and BC-2.06.002 PC-3 (the
DRIFT-001b and DRIFT-002 deferred obligations). These are real obligations, but
they require:

1. A new "console session-list view" sub-surface to be designed (AC-004);
2. A `MissCount()` accessor to be specified and added to `internal/metrics` (AC-005).

These are independent deliverables that do not depend on the console remote-control
transport (AC-001/AC-002/AC-003). Coupling them to S-7.03 inflates a 5-point story
with undefined surface design.

**Ruling: remove AC-004 and AC-005 from S-7.03. Create follow-on backlog story
S-BL.CONSOLE-OBS** with the following scope:

- Design and implement the console daemon session-list view with quality indicator
  (BC-2.06.001 PC-5 console-half, DRIFT-001b);
- Specify and implement `QualityIndicator.MissCount()` or equivalent accessor
  (BC-2.06.002 PC-3, DRIFT-002);
- `sbctl sessions status` missCount surface.

S-BL.CONSOLE-OBS depends on S-7.03 (console daemon must exist before its
observability layer is added).

**S-7.03 post-ruling AC count: 3 (AC-001, AC-002, AC-003). `acceptance_criteria_count`
frontmatter must be updated 5 → 3. `bc_traces` frontmatter must be updated:
remove BC-2.06.001 and BC-2.06.002 (they move to S-BL.CONSOLE-OBS).**

### Deferred obligation tracking

The following DRIFT items remain open after this ruling and are assigned to
S-BL.CONSOLE-OBS:

| DRIFT | Description | Assigned to |
|-------|-------------|-------------|
| DRIFT-001b | BC-2.06.001 PC-5 console session-list quality surfacing | S-BL.CONSOLE-OBS |
| DRIFT-002 | BC-2.06.002 PC-3 missCount observability export | S-BL.CONSOLE-OBS |

---

## S-7.03 Spec Update Summary

| Field | Current (v1.1) | Post-ruling (v1.2) |
|-------|---------------|-------------------|
| `acceptance_criteria_count` | 5 | 3 |
| `bc_traces` | BC-2.08.001, BC-2.06.001, BC-2.06.002 | BC-2.08.001 |
| AC-004 | Present (DRIFT-001b) | Removed — moved to S-BL.CONSOLE-OBS |
| AC-005 | Present (DRIFT-002) | Removed — moved to S-BL.CONSOLE-OBS |
| Architecture Compliance Rule | (none for transport) | Add: "BC-2.08.001 Inv-3 patched v1.2: mgmt-plane transport (ADR-006/ADR-012)" |

**Estimated remaining points for S-7.03:** 3 (reduced from 5). The console
attach/detach/switch RPC handler (AC-001/AC-002/AC-003) follows the established
S-6.03 JSON-over-Unix-socket pattern. `cmd/sbctl/console.go` and
`internal/session/remote.go` remain the correct delivery files.

---

## New Backlog Stub Required

Create `.factory/stories/S-BL.CONSOLE-OBS.md` with:

```yaml
story_id: S-BL.CONSOLE-OBS
title: "Console daemon session-list observability: quality indicator + missCount"
status: backlog
wave: backlog
bc_traces: [BC-2.06.001, BC-2.06.002]
depends_on: [S-7.03]
```

Scope (TBD at scheduling time):
- Console session-list view with quality field (BC-2.06.001 PC-5 console-half);
- `QualityIndicator.MissCount()` or equivalent accessor on `internal/metrics`;
- `sbctl sessions status` missCount exposure (BC-2.06.002 PC-3).

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | architect | Initial ruling: mgmt-plane Unix-socket transport (S-6.03 pattern). BC-2.08.001 Inv-3 must be patched to v1.2. AC-004 and AC-005 removed from S-7.03 and assigned to new S-BL.CONSOLE-OBS backlog story. Rationale: SVTN-channel transport is architecturally incompatible with all established sbctl patterns and would require forbidden imports in cmd/sbctl. Inv-3 "same channel" language is an anachronism predating Wave-5 management-plane establishment. AC-004/AC-005 reference non-existent surfaces (console session-list view, QualityIndicator.MissCount()) requiring independent design. |

## Retrospective Note (2026-07-02)

F-P4L3-MED-002: original ruling wording and BC-2.08.001 v1.2 Inv-3 patch extrapolated "Unix-socket" from S-6.03 access-mode pattern without noting BC-2.07.004 EC-013 authorizes TCP loopback for console mode. Implementation (`cmd/switchboard/mgmt_wire.go:143,153-158`) unconditionally emits `"tcp"` + `"127.0.0.1:9091"` for console mode per BC-2.07.004 EC-013 / AC-014 Ruling D — this is correct behavior. BC-2.08.001 v1.4 rewording clarifies Inv-3 to defer per-mode transport type to BC-2.07.004 EC-013. Ruling intent (mgmt-plane authenticated channel, no data-plane bypass) is preserved by loopback TCP; transport-type wording only was the defect. This retrospective note is informational — the ruling itself is not revised (historical audit trail). The BC-2.08.001 v1.4 amendment is the operative correction.
