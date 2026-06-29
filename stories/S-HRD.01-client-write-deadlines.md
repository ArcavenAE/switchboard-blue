---
artifact_id: S-HRD.01-client-write-deadlines
document_type: story
level: ops
story_id: S-HRD.01
title: "add conn.SetWriteDeadline to client write paths in cmd/sbctl/client.go (defense-in-depth CWE-400)"
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
  - BC-2.07.003
vp_traces: []
subsystems: [network-management]
architecture_modules: [cmd/sbctl]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-6.03]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.003.md'
draft_origin:
  source: ARCH-12 Wave-5 Convergence Round-5 Finding 2 (DEFER-WITH-FOLLOWUP)
  ruling: ARCH-12 v1.5 Round-5 architect ruling — DEFER-WITH-FOLLOWUP
  deferred_from: S-6.03 Wave-5 convergence Round-5
  rationale: >
    Rulings V and Y scoped client-side deadline enforcement to read+dial only.
    The server-side close already bounds the practical CWE-400 write-slowloris
    risk for short client writes, so the client write deadline is defense-in-depth.
    Deferred to avoid scope creep in the final Wave-5 convergence burst.
  severity: MEDIUM
  cwe: CWE-400
acceptance_criteria_count: 0
---

# S-HRD.01: Client Write Deadlines (Defense-in-Depth CWE-400)

> **STATUS: STUB.** This story is a hardening follow-up placeholder created per
> ARCH-12 Wave-5 Convergence Round-5 Finding 2 (DEFER-WITH-FOLLOWUP). Acceptance
> criteria are intentionally absent. When promoted to a hardening-pass wave,
> story-writer will flesh out full ACs, tasks, file structure, and architecture
> mapping.
>
> Anchored to: BC-2.07.003 Inv-2 (does-not-hang invariant).

## Deferral Rationale

ARCH-12 Wave-5 Convergence Rulings V and Y settled the `dispatch()` contract:
- Ruling V: `dispatch()` sets a **read** deadline before decoding the RPC response.
- Ruling Y: `E-NET-001` is emitted for dial failure and handshake read-deadline timeout.

Neither ruling required a **write** deadline on the two client write paths. The
server-side connection close already bounds the practical write-slowloris risk for
the small fixed-size payloads that `client.go` sends (CHALLENGE_RESPONSE and RPC
request frames). Adding a client write deadline is defense-in-depth — it closes the
theoretical CWE-400 vector but is not a correctness requirement for the current
production topology. It was deferred from S-6.03 Wave-5 convergence to avoid scope
creep in the final convergence burst.

## Scope

Add `conn.SetWriteDeadline` to two write paths in `cmd/sbctl/client.go`:

1. **`Authenticate()` CHALLENGE_RESPONSE send** — symmetry with the server-side
   S-W5.01 AC-018 handshake write deadline. The client already sets a read deadline
   (from context/handshake timeout) before awaiting the CHALLENGE; this story adds the
   corresponding write deadline before sending the signed CHALLENGE_RESPONSE.

2. **`dispatch()` RPC request send** — symmetry with the server AC-018 RPC-response
   write deadline. The client already sets a read deadline (Ruling V, BC-2.07.003
   Inv-2) before decoding the RPC response; this story adds the corresponding write
   deadline before sending the RPC request frame.

**Scope out:** `internal/mgmt`, `cmd/switchboard`, tests in other packages. No BC
edits are made at this stub stage (see BC annotation intent below).

## BC Annotation Intent (deferred to implementation)

When this story is implemented, **BC-2.07.003 Inv-2** should gain a clarifying
annotation along these lines:

> The does-not-hang invariant is satisfied by the server-side close for short client
> writes (CHALLENGE_RESPONSE and RPC request frames are fixed-size and small).
> `conn.SetWriteDeadline` on client write paths is defense-in-depth (CWE-400); it is
> implemented by S-HRD.01 but is NOT required for correctness of the Inv-2 guarantee.

**Do NOT edit BC-2.07.003 now.** This annotation is intentionally deferred to
implementation time so the BC reflects the settled invariant without premature churn.

## Behavioral Contract Anchors

| BC Reference | Requirement |
|-------------|-------------|
| BC-2.07.003 Inv-2 | `sbctl` does not hang indefinitely. The `dispatch()` function sets a read deadline before decoding the RPC response. This story extends the defense-in-depth posture by adding write deadlines symmetrically. |

## Narrative

- **As a** security-conscious operator
- **I want** `sbctl` to set write deadlines on all client TCP writes
- **So that** a pathological server or network cannot hold a `sbctl` process open
  indefinitely during the write phase — even though the server-side close already
  makes this risk theoretical in the current topology

## Acceptance Criteria (sketch — expand when scheduled)

### AC-001 (traces to BC-2.07.003 Inv-2 — defense-in-depth)
`Authenticate()` calls `conn.SetWriteDeadline(time.Now().Add(handshakeTimeout))`
immediately before writing the CHALLENGE_RESPONSE frame. The deadline is derived
from the same timeout budget as the read deadline already set by `Authenticate()`.

### AC-002 (traces to BC-2.07.003 Inv-2 — defense-in-depth)
`dispatch()` calls `conn.SetWriteDeadline(time.Now().Add(rpcTimeout))` immediately
before writing the RPC request frame. The deadline is derived from `ctx.Deadline()`
(or `RPCIdleTimeout`-equivalent 30 s fallback), consistent with the read deadline
already set by `dispatch()` per Ruling V.

### AC-003 (correctness guard)
Write deadline expiry (timeout on send) is treated as a connection error: `sbctl`
prints an appropriate error to stderr and exits non-zero. No partial-write state
is exposed to the caller.

## Architecture Notes (to be confirmed when scheduled)

- Both changes are confined to `cmd/sbctl/client.go`.
- The write-deadline value should mirror the read-deadline already present in each
  function — no new timeout configuration knobs are introduced.
- `defer conn.SetWriteDeadline(time.Time{})` to clear after the write is consistent
  with the pattern used for read deadlines; evaluate whether this is needed given the
  connection is short-lived.

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 0.1-stub | 2026-06-29 | product-owner | Initial stub — ARCH-12 Wave-5 Convergence Round-5 Finding 2 (DEFER-WITH-FOLLOWUP); deferred client write deadlines for cmd/sbctl/client.go (Authenticate + dispatch); MEDIUM severity, CWE-400; BC annotation intent recorded for implementation time. |
