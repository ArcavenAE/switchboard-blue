---
artifact_id: S-BL.CLI-SURFACE-COMPLETION-rulings
document_type: decision
level: ops
version: "1.0"
status: final
producer: architect
timestamp: 2026-07-12T00:00:00Z
updated: 2026-07-12T00:00:00Z
cycle: cycle-1
stories_in_scope: [S-BL.CLI-SURFACE-COMPLETION]
bc_traces:
  - BC-2.06.003
  - BC-2.07.001
  - BC-2.07.002
  - BC-2.05.004
  - BC-2.09.001
  - BC-2.09.002
closes_findings: []
resolves: [DRIFT-HS006-DRAIN-CLI-MISSING]
---

# Ruling: S-BL.CLI-SURFACE-COMPLETION — Four Open Design Obligations

All factual claims below are grep-verified against the tree at commit
`4c276d935b089026fac4fa796612352374bb880f` (develop HEAD). File:line anchors
are cited per claim.

This ruling resolves the four Open Design Obligations blocking
`S-BL.CLI-SURFACE-COMPLETION`. It does not edit the story, the anchored BCs,
or `interface-definitions.md` — those edits belong to the product-owner /
story-writer, and are enumerated as explicit follow-on actions at the end of
each ruling and in the summary table.

---

## Verified Premises

| Premise | File:Line | Evidence |
|---|---|---|
| `paths` case arm dispatches only `list` | `cmd/sbctl/main.go:135-145` | `else if args[1] != "list" { err = usageErrf("paths: unknown sub-verb %q...") }` |
| `router` case arm dispatches only `metrics`/`status` | `cmd/sbctl/main.go:146-159` | `switch args[1] { case "metrics": ... case "status": ... default: usageErrf(...) }` |
| No `svtn` case arm exists in the top-level switch | `cmd/sbctl/main.go:132-166` | `switch subcommand { case "sessions": ... case "paths": ... case "router": ... case "console": ... case "admin": ... default: ... }` — no `case "svtn"` |
| `sbctl admin svtn destroy` is fully implemented, name-keyed, confirm-gated | `cmd/sbctl/admin.go:291-326` | `runAdminSvtnDestroy` — `--name`, `runDestroyConfirmGate("admin svtn destroy", ..., "--name", sio)` |
| `admin.key.list-keys` is read-only, gated by admission-any-role (not control-only) | `cmd/switchboard/admin_handlers.go:343-365,584-623` | `resolveCallerAdmissionAnyRole` — bootstrap key, OR operator-set member, OR any active admitted role |
| `SVTNManager` is exclusively name-keyed; no hex-ID reverse index exists | `internal/svtnmgmt/svtnmgmt.go:608` (`SVTNByName`), confirmed via `m.svtns map[string]SVTN` at `:122` | No `SVTNByID` or equivalent method exists anywhere in the package |
| `admin.svtn.create`/`admin.svtn.destroy` are registered in `BuildAdminHandlers`, control-mode-exclusive | `cmd/switchboard/admin_handlers.go:127-133`; `cmd/switchboard/mgmt_wire.go:1078` (`runControl` — only mode passing non-nil admin handlers) | `[]mgmt.Handler{ {Command: "admin.key.register",...}, ..., {Command: "admin.svtn.create",...}, {Command: "admin.svtn.destroy",...} }` |
| `wireMetricsHandlers` is the boundary-layer registration pattern for read-only router-scoped RPCs (`paths.list`, `router.metrics`, `router.status`) | `cmd/switchboard/metrics_wire.go:198-223` | Called from `runRouter` Phase (c), before `serveMgmtServer` (register-before-serve, F-P2L1-001) |
| Tier-1 mgmt authentication is gated solely by `OperatorKeySet` (or bootstrap = daemon's own ephemeral key); there is no SVTN-admission-based Tier-1 gate | `internal/mgmt/mgmt.go:579-598` | `if s.ops.IsBootstrap() { authorized = ... daemonPub ... } else { authorized = s.ops.IsAuthorized(pubkey) }` |
| `runRouter` already implements SIGHUP-triggered live config reload (BC-2.09.001, SHIPPED) | `cmd/switchboard/mgmt_wire.go:459,776-818` | `func runRouter(ctx, w, cfg, configPath string, sighupCh <-chan os.Signal) error` — select loop `case <-sighupCh:` re-parses + validates + diffs upstream routers |
| `runRouter` already implements the full DRAIN + graceful-shutdown sequence, triggered only by `ctx.Done()` (SIGTERM/SIGINT via `signal.NotifyContext`) | `cmd/switchboard/mgmt_wire.go:776-939`; `cmd/switchboard/main.go:118-125` | `case <-ctx.Done(): goto shutdown` → `drainCoord.Signal`/`Wait` → per-node flush → `ingressCancel()` → `mgmtSrv.Shutdown` |
| No wire verb `router.reload` or `router.drain` exists anywhere in the tree | grep across `cmd/`, `internal/` (excl. `_test.go`) | zero hits for `router\.reload`, `router\.drain`, `RouterReload`, `RouterDrain` |
| `sbctl router reload`/`sbctl router drain` RPC wiring was explicitly deferred as a "follow-on ops-UX story" — the exact gap this story closes | `.factory/decisions/S-7.04-FU-DRAIN-WIRE-placement-note.md:737-741`; `.factory/decisions/S-7.04-FU-SIGHUP-RELOAD-placement-note.md:319-322` | "`sbctl router drain` CLI subcommand — explicitly deferred (DRIFT-HS006-DRAIN-CLI-MISSING adjudicated at PR #103). This story does NOT implement a management-RPC drain verb... If a targeted-drain UX is needed, it is a follow-on ops-UX story." / "`sbctl router reload` RPC ... deferred per the existing `DRIFT-HS006-DRAIN-CLI-MISSING` adjudication" |

---

## Ruling 1 — `paths ping`: new RPC, not reuse of `paths.list`

**DECISION: Commission new RPC verb `paths.ping`. Commission new BC-2.06.004.**

### Rationale

`sbctl paths ping --router=<addr>` (§77) is architecturally distinct from
`sbctl paths list` (BC-2.06.003 PC-1). The `--router=<addr>` flag plays the
same role `--target <router>` plays for `sbctl router status` (BC-2.06.003
PC-3, `interface-definitions.md:80`) — it overrides the connection target for
this one dispatch (`cmd/sbctl/main.go` global `--target` override pattern),
not a wire payload field. Once dialed, `paths.list`/`router.status` report
**historical, keep-alive-derived, EWMA-smoothed** per-path metrics accumulated
by a `PathTracker` over time (`internal/paths`, `cmd/switchboard/metrics_wire.go:50-163`).
`paths ping` is a **one-shot, on-demand reachability + latency probe of a
specific target**, semantically closer to `ping(8)` than to a metrics query.
Reusing `paths.list` and discarding its body to derive a timing figure is a
category mismatch that would confuse the RPC-name-based audit trail (an
operator or auditor reading `paths.list` in an mgmt-RPC log has no way to
tell "real path enumeration" from "someone using it as a stopwatch").

Given Tier-1 mgmt authentication (`internal/mgmt/mgmt.go:579-598`) already
requires a full dial + Ed25519 challenge-response handshake before ANY RPC
dispatches, and the story asks for a "one-shot RTT probe," the minimal
correct primitive is a **bodyless ping RPC**: the daemon does no work beyond
authenticating and returning a trivial ack; `cmd/sbctl` measures the
round-trip wall-clock time itself (dial-start → response-decode-complete).
This avoids inventing server-side RTT computation and avoids any clock-sync
assumptions between client and daemon.

### Wire contract

- **Verb:** `paths.ping`
- **Request args:** `{}` (empty — the daemon being dialed via `--router=<addr>`
  / `--target` IS the probe target by construction; no `svtn_id` needed)
- **Response data:** `{"pong": true}`
- **CLI-synthesized output** (not on the wire — computed by `cmd/sbctl` around
  the `connectAndRun` call): `{"router": "<addr>", "rtt_ms": <float64>}`
- **Authority:** Tier-1 operator-key authentication only (same bar as
  `paths.list`/`router.metrics`/`router.status` — none of those three carry an
  additional Tier-2 role gate today; `paths.ping` should not invent one)

### Reachability vs. slow semantics

Unreachable-before-connection → **E-NET-001**, exit 1 (BC-2.07.003, shared
by every sbctl command). Auth failure after connection → **E-ADM-010**, exit
1 (BC-2.07.002 PC-4). A connection that succeeds but is slow is **not an
error** — `rtt_ms` simply reports a larger number, exactly like `ping(8)`.
`paths ping` performs **no quality classification** (no green/yellow/red);
that computation is `router.status`'s job (BC-2.06.003 PC-3) and pulling it
into `paths ping` would re-couple the two capabilities this ruling just
separated.

### BC action for PO

**Commission BC-2.06.004** ("On-Demand Single-Target Reachability Probe via
`sbctl paths ping`"), NOT an extension of BC-2.06.003 — the target scope
(arbitrary dialed router, not "the caller's own established paths") and the
underlying mechanism (raw connect+auth RTT, not accumulated EWMA history)
are different enough that folding this into BC-2.06.003 would blur its
"per-path metrics" contract. PC skeleton:

- **Precondition 1:** the target daemon at `--router=<addr>` is reachable by
  sbctl and Tier-1-authenticates the operator's key (shared preconditions
  with BC-2.07.002).
- **PC-1:** `sbctl paths ping --router=<addr>` dials `<addr>` directly
  (overriding `--target`), authenticates, and issues `paths.ping` with empty
  args. On success, reports round-trip time in milliseconds measured
  client-side from dial-start to response-decode-complete.
- **PC-2:** if the daemon is unreachable, sbctl returns E-NET-001 (BC-2.07.003);
  exit 1.
- **PC-3:** if authentication fails, sbctl returns E-ADM-010; exit 1.
- **PC-4:** `paths.ping` performs no per-path metrics computation and returns
  no quality classification; high latency is reported as a value, not an
  error.
- **Invariant:** `paths.ping` requires no additional Tier-2 authority beyond
  the daemon's standard Tier-1 operator-key authentication.
- **Trigger:** Operator runs `sbctl paths ping --router=<addr>`.
- **Registered Verbs table row (interface-definitions.md §397 area):**
  `paths.ping | BC-2.06.004 PC-1 | Tier-1 operator-key auth | {} | {"pong": true} | S-BL.CLI-SURFACE-COMPLETION`

### Implementation constraints

- Registration: new handler function (e.g. `mgmt.RegisterPingHandler` in
  `internal/mgmt/register_metrics.go` or a sibling file) called from
  `wireMetricsHandlers` (`cmd/switchboard/metrics_wire.go:212-223`) so it is
  available on **every** daemon mode that already wires metrics handlers
  (router, access, console, control) — `paths ping` targets an arbitrary
  daemon, so it should not be router-mode-exclusive the way Ruling 4's verbs
  are.
- CLI: new `runPathsPing(ctx, target, keyPath, useJSON, args, sio)` in
  `cmd/sbctl/paths_list.go` (or a new `paths_ping.go`), wired into the
  `paths` case arm in `cmd/sbctl/main.go:135-145` alongside `list`.

---

## Ruling 2 — `svtn status`: extend BC-2.07.001, wire as `admin.svtn.status`, any-admitted-role authority

**DECISION: Extend BC-2.07.001 with new Postcondition PC-4. Wire verb
`admin.svtn.status`. Authority: any admitted role (reuse
`resolveCallerAdmissionAnyRole`). Response schema excludes session/health
data (purity-boundary violation). E-SVTN-003 reused for not-found.**

### Rationale — extend, don't commission

Direct precedent: `admin.key.list-keys` (read-only) lives **inside**
BC-2.05.004 alongside the destructive key-lifecycle operations (register,
revoke, expire) as an added precondition/authority carve-out (F-L2-003,
`BC-2.05.004.md:185`), not as a separate BC. The read op and the destructive
ops share the same underlying manager (`SVTNManager`) and the same boundary
package (`cmd/switchboard/admin_handlers.go`). `svtn status` is the
symmetric case for `BC-2.07.001` (create/destroy): same manager, same
package, a new read accessor over existing state — not a new mechanism (unlike
Ruling 1's `paths ping`, which genuinely differs in mechanism and target
scope from BC-2.06.003). Extend, matching the list-keys precedent.

### Wire contract

- **Verb:** `admin.svtn.status` — keeps the `admin.svtn.*` naming family
  established by `admin.svtn.create`/`admin.svtn.destroy`
  (`cmd/switchboard/admin_handlers.go:127-133`).
- **Registration:** new handler in `BuildAdminHandlers`
  (`admin_handlers.go:127-133`), same as create/destroy — it needs
  `*svtnmgmt.SVTNManager`, which only exists on the control-mode daemon
  (`runControl`, `mgmt_wire.go:1078`). Router/access/console pass nil admin
  handlers (ADR-004) and correctly return E-RPC-010, exactly as they already
  do for `admin.svtn.create`/`destroy`.
- **Request args:** `{"name": "<svtn-name>"}`
- **Response data:**
  ```json
  {
    "svtn_id": "<hex>",
    "name": "<svtn-name>",
    "created_at": "<RFC3339>",
    "key_counts": {"control": <n>, "console": <n>, "access": <n>}
  }
  ```
- **Authority:** any admitted role (control, console, or access) in the
  target SVTN, OR operator-set member, OR bootstrap key — reuse
  `resolveCallerAdmissionAnyRole` (`admin_handlers.go:592-623`) verbatim, the
  same function `admin.key.list-keys` already uses. The admission gate still
  applies (CWE-862 defense against cross-SVTN roster/existence enumeration —
  same reasoning as BC-2.05.004 EC-008); it is only the control-only
  *authority* gate that is skipped, matching F-L2-003.
- **Error codes:** E-SVTN-003 (SVTN not found — reuse the existing
  `mapAdminError` `ErrSVTNNotFound` arm, `admin_handlers.go:413-414`),
  E-CFG-001 (missing `--name`), E-ADM-009 (admission failure, same three
  reachable modes as BC-2.05.004 EC-008).

### Why NOT session/health data

`admin_handlers.go`'s own package header states the purity boundary
explicitly: **"Forbidden imports: ... internal/session ..."**
(`admin_handlers.go:20-23`). Active-session data lives in `internal/session`,
populated on access/console nodes — not reachable from the control-mode
daemon's `SVTNManager` without crossing ARCH-09's boundary classification (a
genuinely new cross-daemon query design, disproportionate to this story).
The response schema above uses only fields `SVTNManager` already exposes:
`SVTN{ID, Name, CreatedAt}` (`internal/svtnmgmt/svtnmgmt.go:86-95`) via
`SVTNByName` (`:608`), and role-grouped counts derived from `ListKeys`
(`:719`, already used by `admin.key.list-keys`). No health indicator is
proposed for the same reason — there is no accessible signal to compute one
from at this boundary.

### `--id` vs `--name` (§62)

Same defect as Ruling 3 below: §62 specifies `--id=<svtn_id>`, but
`SVTNManager` is exclusively name-keyed (Verified Premises table). Rule:
CLI flag is `--name=<svtn-name>`, matching every other `admin svtn`/`admin
key` command family. §62 needs PO correction, same class as the existing
`--svtn` placeholder-semantics note (`interface-definitions.md:113`).

### CLI dispatch — real implementation, not a shim

Unlike Ruling 3, `svtn status` is **read-only and non-destructive**, so
none of the confirm-gate duplication risk that motivates Ruling 3's shim
applies here. `sbctl svtn status --name=<svtn-name>` (top-level `svtn` case
arm, Scope item 1) should be a genuine standalone dispatch directly to
`admin.svtn.status` — it does not need to route through `sbctl admin`
framing, exactly as `paths list`/`router status` are already bare top-level
(non-`admin`-prefixed) reads that touch daemon-internal state.

### BC action for PO

**Extend BC-2.07.001 → v1.14.** Add Postcondition **PC-4 (Status)**:

> **Status**: Returns the SVTN's `svtn_id` (hex), `name`, `created_at`, and
> admitted-key counts grouped by role. Authority: any admitted role in the
> target SVTN, OR operator-set member, OR bootstrap key
> (`resolveCallerAdmissionAnyRole`, mirroring BC-2.05.004 F-L2-003). Does
> **not** include active-session or health data — out of the control-mode
> daemon's accessible state (ARCH-09 purity boundary; `internal/session` is
> a forbidden import for `cmd/switchboard/admin_handlers.go`).

Add Canonical Test Vectors: happy-path (`sbctl svtn status --name=mynet` →
status fields), not-found (`E-SVTN-003`), admission-denied
(cross-SVTN caller → `E-ADM-009`). Add a VP (new VP-XXX or a sibling entry
under VP-048) for "status returns accurate key counts / correct
admission-gate enforcement." Update the Registered Verbs table with the
`admin.svtn.status` row.

---

## Ruling 3 — `svtn destroy` top-level form: migration shim, not a parallel alias

**DECISION: `sbctl svtn destroy` is a migration shim. It does not implement
`--id`, does not dispatch `admin.svtn.destroy`, and does not duplicate the
confirm-gate. It always returns a usage error (exit 2) redirecting to `sbctl
admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]`.**

### Rationale

1. **Direct precedent, same verb family.** `sbctl svtn create` — the sibling
   verb — was **removed entirely**, not aliased, for exactly this reason:
   `interface-definitions.md:59` — "`[REMOVED]` Alias removed as of Phase 5
   Pass 3 Path B remediation (PR #62)... Migration target: `sbctl admin svtn
   create`." The project's established convention for a top-level `svtn
   <verb>` that duplicates `admin svtn <verb>` is: don't maintain two
   parallel code paths for a destructive/administrative operation.

2. **`--id=<svtn_id>` cannot be honored literally.** `SVTNManager` is
   exclusively name-keyed — `m.svtns map[string]SVTN`
   (`internal/svtnmgmt/svtnmgmt.go:122`), looked up via `SVTNByName`
   (`:608`). No hex-ID reverse index exists anywhere in the package (grep
   confirmed — no `SVTNByID` or equivalent). Implementing `--id` as specified
   would require adding a new reverse-lookup index to `SVTNManager` — a real
   data-structure change, not "wire an existing accessor," and
   disproportionate to a CLI-surface-completion story. Silently
   reinterpreting `--id` to mean "name" would be worse: a misleading flag
   name on a **destructive** command is a footgun.

3. **Duplicating `runDestroyConfirmGate` doubles a security-sensitive
   surface for no operator benefit.** The confirm gate
   (`cmd/sbctl/admin.go:328-`) is the ADR-004 split-brain mitigation for
   destructive admin operations. `sbctl admin svtn destroy` already
   implements it correctly and is the documented canonical form
   (`docs/sbctl.md:516`). A second, top-level code path implementing the
   same gate is a second place that gate can drift or be gotten wrong; there
   is no operator-facing reason to have two.

4. **Cheapest option that still closes the Scope-item-1 obligation.** The
   `svtn` case arm still needs to exist (Scope item 1) and must not fall
   through to a generic "unknown subcommand" error for `destroy` — the shim
   satisfies that with near-zero new surface: recognize `destroy` as a
   known-but-redirected sub-verb, print the redirect, return `usageErrf`
   (exit 2). No RPC dispatch, no confirm-gate duplication, no `--id`/`--name`
   flag semantics needed at all.

### Implementation

In the new `runSvtn` dispatch function (`cmd/sbctl/main.go`'s `svtn` case
arm, Scope item 1), the `destroy` sub-verb is:

```go
case "destroy":
    return usageErrf("svtn destroy: use 'sbctl admin svtn destroy --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'")
```

This resolves the `--id` vs `--name` discrepancy (§60) by construction: the
shim never parses either flag, so the discrepancy is moot in the
implementation. §60 and `docs/sbctl.md`'s Unimplemented-verbs table
(`docs/sbctl.md:551`) both need PO/spec-steward correction — reclassify
`svtn destroy`'s disposition from "PENDING full implementation" to
"won't-fix / migration shim," same disposition class already used for `svtn
list` (`interface-definitions.md:61`, `docs/sbctl.md:553`) and the same
class `svtn create` was moved to at PR #62.

### BC action for PO

No BC change needed — BC-2.07.001 PC-3 already fully governs
`admin.svtn.destroy`; this ruling only concerns the top-level CLI alias
surface, which was never itself a BC anchor point (`interface-definitions.md`
§60 is CLI-surface documentation, not a BC citation). Correct §60's
annotation and `docs/sbctl.md`'s table entry as described above.

---

## Ruling 4 — `router reload` / `router drain`: new router-mode RPC verbs `router.reload` / `router.drain`, in scope (not descoped)

**DECISION: In scope. Wire verb names `router.reload` and `router.drain`,
registered on the router daemon only, via a new `wireRouterControlHandlers`
function called from `runRouter` alongside `wireMetricsHandlers`. Both
handlers bridge into the **already-shipped** SIGHUP-reload and
SIGTERM-drain code paths via new channels threaded the same way `sighupCh`
already is — no reload/drain logic is duplicated. This closes
`DRIFT-HS006-DRAIN-CLI-MISSING`.**

### This is not "confirm a name" — the RPC surface genuinely does not exist yet, and that is expected

Both mechanisms already exist and are shipped:

- **Reload** (BC-2.09.001 PC-1, `S-7.04-FU-SIGHUP-RELOAD`): `runRouter`
  selects on `sighupCh <-chan os.Signal`
  (`cmd/switchboard/mgmt_wire.go:459,776-818`); on receipt it re-parses,
  validates, and diffs the config, fail-closed on error.
- **Drain** (BC-2.09.002, `S-7.04-FU-DRAIN-WIRE`): `runRouter`'s
  `ctx.Done()` arm (`:778-782`) jumps to the `shutdown:` label
  (`:819-939`), which signals the drain coordinator, broadcasts a
  DRAIN-over-SVTN frame to every connected node, bounds the wait by
  `drain_timeout`, and cleanly tears the daemon down.

Both are triggerable **only by OS signal today** (SIGHUP, SIGTERM/SIGINT via
`signal.NotifyContext`, `cmd/switchboard/main.go:118-125`). No RPC path
reaches either. This was a **deliberate, documented, prior-architect
deferral**, not an oversight: both placement notes name the exact gap this
story closes.

> `.factory/decisions/S-7.04-FU-DRAIN-WIRE-placement-note.md:737-741`:
> "`sbctl router drain` CLI subcommand — explicitly deferred
> (DRIFT-HS006-DRAIN-CLI-MISSING adjudicated at PR #103). This story does
> NOT implement a management-RPC drain verb. Only the SIGTERM/shutdown path
> triggers the drain coordinator. If a targeted-drain UX is needed, it is a
> follow-on ops-UX story."

> `.factory/decisions/S-7.04-FU-SIGHUP-RELOAD-placement-note.md:319-322`:
> "`sbctl router reload` RPC — BC-2.09.001 Trigger names both SIGHUP and
> `sbctl router reload`. This story covers only the SIGHUP path. The sbctl
> surface is a management-RPC concern deferred per the existing
> `DRIFT-HS006-DRAIN-CLI-MISSING` adjudication in S-7.04-DELIVERY."

Given the underlying mechanisms are fully shipped and tested, what remains
is a **small, well-scoped RPC-to-channel bridge**, not new capability from
scratch. This is squarely CLI-surface-completion-shaped work. **Descoping is
not warranted** — I considered it (the story's brief explicitly allows it as
a legitimate ruling) and reject it: the missing piece is bounded, low-risk,
and directly named as this story's job by two prior architect notes.

### Wire verb names

- `router.reload`
- `router.drain`

These match the CLI sub-verb names already dispatched from the `router` case
arm (`cmd/sbctl/main.go:151-158`, alongside `metrics`/`status`), and match
the `<namespace>.<verb>` convention used throughout (`paths.list`,
`admin.svtn.destroy`, etc.).

### Registration point — new function, router-mode-exclusive

Register via a **new** `wireRouterControlHandlers` function
(`cmd/switchboard`, boundary layer — mirrors `metrics_wire.go`'s ARCH-09
classification, not `internal/config`/`internal/drain`, since this is wiring
code, not business logic):

```go
func wireRouterControlHandlers(srv *mgmt.Server, sighupCh chan os.Signal, drainRequestCh chan struct{}) error
```

Called from `runRouter` at Phase (c), alongside the existing
`wireMetricsHandlers` call (`mgmt_wire.go:496-498`), **before**
`serveMgmtServer` (Phase (e), `:510`) — satisfying the same
register-before-serve invariant (F-P2L1-001) `wireMetricsHandlers` already
follows.

**Router-mode-exclusive.** `runAccess`, `runConsole`, `runControl` never
call `wireRouterControlHandlers` — `router.reload`/`router.drain` are
meaningless on those daemon modes (they have no `sighupCh`/drain-coordinator
concept at all). This is a **new** mode-exclusion pattern, parallel to but
distinct from ADR-004's exclusion of `admin.*` handlers from non-control
modes. Recommend the PO/architect add a row for it wherever the ADR-004
disambiguation table (`.factory/specs/architecture/ARCH-04-admission-security.md:91`)
enumerates per-mode handler sets, so it doesn't silently drift.

### Bridging mechanism — reuse `sighupCh`, add `drainRequestCh` (symmetric with the already-shipped pattern)

**Reload:** no new channel. `router.reload`'s handler synthesizes the exact
same signal the SIGHUP path already consumes:

```go
select {
case sighupCh <- syscall.SIGHUP:
default: // a reload is already pending; drop, matching signal.Notify's own coalescing semantics
}
```

This requires widening `runRouter`'s `sighupCh` parameter from
`<-chan os.Signal` (receive-only, current signature at `:459`) to
`chan os.Signal` (bidirectional) — a one-line signature change. Every
existing call site (`main.go:120-125`, and every test per
`S-7.04-FU-SIGHUP-RELOAD-placement-note.md` Q6) already constructs a
bidirectional `make(chan os.Signal, 1)`; only `runRouter`'s own parameter
type needs to widen; no call site needs to change.

Recommended improvement over the raw signal-equivalence: unlike a bare
SIGHUP (which silently no-ops when `configPath == ""`, `:786-788`), the RPC
handler has a response channel and should surface that case synchronously —
`E-CFG-004: reload not applicable: daemon started without --config` (reusing
the existing `E-CFG-004` class already used for the "no config loaded"
guard at `mgmt_wire.go:465-467`) — rather than silently returning
`{"accepted": true}` for a request that will do nothing.

**Drain:** genuinely new. Add a third select-loop arm, symmetric with the
`sighupCh` arm added one story ago:

```go
for {
    select {
    case <-ctx.Done():
        goto shutdown
    case <-sighupCh:
        // existing reload logic, unchanged
    case <-drainRequestCh:
        goto shutdown
    }
}
```

`drainRequestCh chan struct{}` (buffered 1), threaded into `runRouter` the
same way `sighupCh` was threaded in by `S-7.04-FU-SIGHUP-RELOAD`
(constructed in `main.go`'s `"router"` case body, passed as a new
parameter). `router.drain`'s handler:

```go
select {
case drainRequestCh <- struct{}{}:
default: // a drain is already in flight; no-op
}
```

**Why a new channel and not threading `cancel func()` into the RPC layer:**
considered and rejected. `router.drain` triggering `cancel()` directly would
require handing the daemon's root-context cancel function to an RPC handler
closure — a capability normally owned solely by `main.go`. The channel
approach keeps `cancel()` ownership exactly where it is today, mirrors the
`sighupCh` precedent the prior architect note (`S-7.04-FU-SIGHUP-RELOAD-placement-note.md`
Q2) already established and justified ("callback / function pointer... adds
indirection with no testing benefit"), and is easier to test (send-to-channel,
exactly like the existing `sighupCh` test pattern in that note's Q6).

### Wire contract

- **Request args (both):** `{}`
- **Response data (both):** `{"accepted": true}` — fire-and-forget, matching
  the UX parity with sending a raw OS signal (a `kill -HUP`/`kill -TERM`
  sender gets no synchronous confirmation of completion either; the
  operator confirms via logs / `router status` afterward). A synchronous
  "wait for reload to actually complete and report success/failure" variant
  would require a response channel back from the select loop to the RPC
  handler goroutine — real added complexity, not required by either BC's
  Trigger text, and out of proportion to this story's P2/backlog priority.
  Flag as a future enhancement if operators want stronger confirmation
  semantics.
- **`router.drain` connection-teardown note (implementation constraint for
  test-writer):** because drain triggers the **full** shutdown sequence
  (same as SIGTERM per BC-2.09.002's Trigger equivalence), the RPC
  connection itself will likely be severed as the daemon exits shortly
  after — the client should treat "connection reset" following a
  `{"accepted": true}` (or even without one) as an expected outcome, not a
  protocol error. This mirrors BC-2.09.002 PC-3's existing "best-effort
  delivery... no wire-level DRAIN-ACK opcode" framing (v1.2 amendment,
  `BC-2.09.002.md:24`) — extend that same best-effort posture to the
  triggering RPC itself.
- **Authority:** Tier-1 operator-key authentication only — the same (and
  only) gate `paths.list`/`router.metrics`/`router.status` already use on
  this daemon. No stricter Tier-2 gate is available to reuse: router mode
  has no `SVTNManager`/`RoleControl` concept at all (it registers nil admin
  handlers; `RoleControl` is scoped to the control-mode daemon's key
  registry). Introducing a new "router-operator" role concept would be a
  substantial new capability neither BC requests (both frame the trigger as
  "operator runs..." with no role qualifier, treating RPC-triggered
  reload/drain as equivalent to already having OS-level access to signal
  the process) and disproportionate to this story. Tier-1 auth is already a
  real bar: in bootstrap mode (no `AuthorizedOperatorKeys` configured), only
  the caller who holds the daemon's own ephemeral private key can connect
  at all.
- **Error codes:** E-NET-001 (unreachable), E-ADM-010 (auth failure) — the
  standard shared connection-error codes (BC-2.07.002/2.07.003). Reload adds
  E-CFG-004 for the no-config-loaded case (see above). No new error codes
  needed for drain.

### BC action for PO

No new BC needed — both BC-2.09.001 and BC-2.09.002 already name `sbctl
router reload`/`sbctl router drain` explicitly in their Trigger sections as
equivalent alternatives to SIGHUP/SIGTERM; the CLI commands were always
sanctioned, only the wire mechanics were unstated. Recommend a **governance-only**
clarifying addendum to each (no PC/AC behavior change — mirrors the
`POL-005`/governance-leaf pattern already used elsewhere in this codebase,
e.g. `BC-2.07.001.md` v1.13 changelog):

- **BC-2.09.001**: add a sentence to PC-1 — "RPC-triggered reload via the
  `router.reload` wire verb is dispatched through the same `sighupCh`
  channel the SIGHUP OS-signal path consumes; the two triggers are
  code-path-identical from that point forward. See
  `S-BL.CLI-SURFACE-COMPLETION-rulings.md` Ruling 4."
- **BC-2.09.002**: add a sentence to the Trigger / PC-1 — "RPC-triggered
  drain via the `router.drain` wire verb causes the same shutdown sequence
  as SIGTERM (both reach the `shutdown:` label); the RPC connection is
  expected to be severed as the daemon exits, consistent with PC-3's
  best-effort-delivery framing. See `S-BL.CLI-SURFACE-COMPLETION-rulings.md`
  Ruling 4."

Recommend tagging the story/PR with `Resolves: DRIFT-HS006-DRAIN-CLI-MISSING`
per this codebase's convention of not using `closes`/`fixes` for
externally-reported items until confirmed — here the "reporter" is the prior
architect note itself, and the resolution IS the wire-verb registration
this ruling specifies, so `Resolves:` (not `Closes:`) is the correct verb
per this repo's existing `Refs:` convention for non-author-confirmed
closure.

### Implementation constraints (summary for implementer)

1. `runRouter` signature: `sighupCh <-chan os.Signal` → `chan os.Signal`;
   add `drainRequestCh chan struct{}` parameter.
2. `main.go`'s `"router"` case body: construct `drainRequestCh := make(chan
   struct{}, 1)` alongside the existing `sighupCh` construction
   (`main.go:120-125`); pass both into `runRouter`.
3. New file `cmd/switchboard/router_control_wire.go` (or fold into
   `mgmt_wire.go`): `wireRouterControlHandlers(srv, sighupCh,
   drainRequestCh) error`, called from `runRouter` Phase (c) alongside
   `wireMetricsHandlers` (`:496-498`).
4. Select loop (`:776-818`) gains the third `case <-drainRequestCh: goto
   shutdown` arm.
5. Test surface: mirror the `sighupCh` injection pattern from
   `S-7.04-FU-SIGHUP-RELOAD-placement-note.md` Q6 — tests construct their
   own `drainRequestCh` and send `struct{}{}` directly, no real OS signal or
   real RPC round-trip required for unit-level coverage; reserve one
   integration test per verb for the actual RPC → channel → shutdown path.

---

## Summary Table

| # | Verb(s) | Decision | Wire verb | BC action |
|---|---|---|---|---|
| 1 | `paths ping` | New RPC, not `paths.list` reuse | `paths.ping` (empty args → `{"pong": true}`) | **Commission BC-2.06.004** |
| 2 | `svtn status` | Extend BC-2.07.001; any-admitted-role authority | `admin.svtn.status` | **Extend BC-2.07.001 with PC-4** |
| 3 | `svtn destroy` (top-level) | Migration shim, not a parallel alias | none (no RPC dispatch; redirects to `sbctl admin svtn destroy`) | No BC change; correct §60 annotation |
| 4 | `router reload` / `router drain` | In scope; new router-mode RPCs bridging into shipped SIGHUP/SIGTERM code paths | `router.reload`, `router.drain` | Governance-only clarifying addenda to BC-2.09.001/BC-2.09.002; resolves `DRIFT-HS006-DRAIN-CLI-MISSING` |

**Nothing in this ruling descopes the story.** Ruling 4 is the largest piece
of new plumbing (two new channels, one new registration function,
router-mode-exclusive wiring) but is fully specified above and is squarely
what the story was scheduled to close.

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-12 | architect | Initial ruling on all four Open Design Obligations for `S-BL.CLI-SURFACE-COMPLETION`. `paths ping` → new BC-2.06.004 + new `paths.ping` RPC (category mismatch with `paths.list`'s historical-metrics contract). `svtn status` → extend BC-2.07.001 PC-4, wire `admin.svtn.status`, any-admitted-role authority (list-keys precedent), no session/health fields (ARCH-09 purity boundary). `svtn destroy` top-level → migration shim per the `svtn create` removal precedent; `--id` flag not implementable against a name-keyed `SVTNManager`. `router reload`/`router drain` → in scope, new router-mode-exclusive RPCs `router.reload`/`router.drain` bridging into the already-shipped SIGHUP-reload and SIGTERM-drain code paths via `sighupCh` reuse + new `drainRequestCh`; resolves the explicitly-deferred `DRIFT-HS006-DRAIN-CLI-MISSING` gap named in the `S-7.04-FU-SIGHUP-RELOAD` and `S-7.04-FU-DRAIN-WIRE` placement notes. |
