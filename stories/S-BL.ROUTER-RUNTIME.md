---
artifact_id: S-BL.ROUTER-RUNTIME
document_type: story
level: ops
story_id: S-BL.ROUTER-RUNTIME
version: "1.1"
title: "router daemon runtime — mgmt plane + data listener bind (retires runRouter stub)"
status: merged
producer: implementer
timestamp: 2026-07-05T00:00:00
modified: 2026-07-05T12:00:00
phase: 2
epic: E-6
wave: 7
priority: P1
scope_phase: E
estimated_points: 2
bc_traces:
  - BC-2.06.001
  - BC-2.06.002
  - BC-2.09.003
vp_traces: [VP-047]
subsystems: [network-management, transport-layer]
architecture_modules: [cmd/switchboard, internal/mgmt, internal/config]
tdd_mode: strict
cycle: v1.0.0-greenfield
depends_on: [S-6.06]
blocks: [S-BL.NI, S-BL.OA]
acceptance_criteria_count: 4
drift_origin: DRIFT-HS006-ROUTER-DAEMON-STUB
supersedes_issue: "task #144 (pre-Plan-B era)"
---

# S-BL.ROUTER-RUNTIME — Router daemon runtime skeleton

## Background

`cmd/switchboard/mgmt_wire.go:357-359` shipped with a stub:

```go
func runRouter(ctx context.Context, w io.Writer, cfg *config.Config) error {
    return errors.New("runRouter: not implemented")
}
```

Every other daemon subcommand (`access`, `console`, `control`) has a
working runtime that starts the management plane and (where applicable)
binds a data-plane listener. Router alone stayed a stub — surfaced in the
tier-3 tutorial smoke as the known `T3-2-router` expected-fail (exit 3),
and blocking real network-ingress work (S-BL.NI) and outer-assembler
wiring (S-BL.OA) from having a daemon to talk to.

This story delivers the **runtime skeleton only**: management server
comes up with the register-before-serve invariant honored, admin
handlers are NOT registered (router role-exclusion per ADR-004 / ARCH-04
/ S-6.06 AC-004), a data-plane TCP listener binds `cfg.ListenAddr`,
SIGTERM produces a clean drained exit.

Explicitly out of scope:

- Runtime-mode graduation (E vs PE) — deferred; the daemon accepts any
  `upstream_routers:` value but does not yet behave differently based on it.
- SIGHUP config reload — S-6.04 territory.
- Real frame accept + forward — S-BL.NI (network ingress) picks up the
  data plane; S-BL.OA composes outer + channel frames. This story does
  accept-and-immediately-close so the port is bindable and the daemon
  is reachable enough to prove startup.
- Router-to-router peer session establishment.
- DRAIN protocol to downstream nodes — S-7.04 Wave 7.

## Acceptance Criteria

### AC-001: Router starts management plane

`runRouter` constructs the management server via the canonical three-phase
pattern shared with `runControl` and `runConsole`:

1. Generate an ephemeral Ed25519 daemon keypair.
2. `newMgmtServer(cfg, "router", daemonPriv, adminHandlers=nil)` — pass
   `nil` for admin handlers per ADR-004 role-exclusion (router MUST NOT
   register `admin.key.*` handlers).
3. `wireMetricsHandlers(mgmtSrv)` — register non-admin management RPCs.
4. `serveMgmtServer(ctx, &wg, mgmtSrv)` — start the accept loop.

Register-before-serve invariant (F-P2L1-001, F-P2L1-002) is preserved:
metrics handlers are registered on the server before its accept
goroutine is spawned.

On success the socket file exists on disk with 0600 permissions
(AC-014 / CWE-276) at the path returned by `resolveManagementSocket(cfg, "router")`.

### AC-002: Data-plane listener binds cfg.ListenAddr

`runRouter` calls `net.Listen("tcp", cfg.ListenAddr)`. On failure the
error is wrapped and returned; management server is torn down (`Ruling J`:
any mgmt-start failure aborts daemon startup, no degraded mode).

On success an accept goroutine (`wg.Add(1)` + `defer wg.Done()` per
ARCH-01 §Goroutine WaitGroup Contract) accepts connections and closes
them immediately. Real frame handling is deferred to S-BL.NI / S-BL.OA;
this loop exists so the port is bindable and the daemon is reachable
enough for smoke to observe.

### AC-003: Observability lines on the writer

Two lines are written to the writer `main.go` passes:

```
switchboard router: data plane listening on <resolved-addr>
switchboard router: management socket at <sock-path>
```

The resolved data-plane address is `dataLn.Addr().String()` (not
`cfg.ListenAddr`), so operators using `"127.0.0.1:0"` see the
kernel-assigned port. The management socket path is
`resolveManagementSocket(cfg, "router")`.

**Note on stream direction:** `main.go:120` passes `os.Stderr` as the
writer for every daemon subcommand. `docs/getting-started.md:103` says
router logs go to stdout. This story preserves the code path
(`os.Stderr`) — the writer is a parameter, not a hard-coded stream —
and flags the docs discrepancy for a separate story to resolve. Not
resolved unilaterally here.

### AC-004: Graceful shutdown on ctx.Done()

`runRouter` blocks on `<-ctx.Done()`. On cancellation:

1. Close the data-plane listener (unblocks the accept loop).
2. `dataWG.Wait()` — accept goroutine exits.
3. `mgmtSrv.Shutdown(shutCtx)` with a 5-second deadline.
4. `mgmtWG.Wait()` — mgmt goroutines exit.
5. Return `nil` — clean exit 0.

`main.go`'s `signal.NotifyContext(ctx, SIGINT, SIGTERM)` provides the
cancellation. SIGTERM ⇒ ctx cancel ⇒ steps 1-5 ⇒ exit 0.

## Test coverage (all in cmd/switchboard/mgmt_wire_test.go)

| Test | AC | Approach |
|------|----|----|
| `TestRunRouter_StartsWithMgmt` | AC-001 | Pre-cancelled ctx; `ListenAddr: "127.0.0.1:0"`; assert `runRouter` returns nil and mgmt socket file exists via `os.Stat`. |
| `TestRunRouter_DataListenerBinds` | AC-002 | Probe-bind an ephemeral TCP port; run `runRouter` in a goroutine; poll `net.DialTimeout` until success (1s budget, 20ms poll); cancel; assert return within 2s. |
| `TestRunRouter_NoAdminHandlers` | AC-001 (role-exclusion) | Wait for socket; dial; read the CHALLENGE envelope via `json.NewDecoder`; assert `challenge["type"] == "challenge"`. Structural nil-handlers guarantee is covered by `TestAccessMode_AdminHandlersNotRegistered` via `startE2EServer(t, nil)`. |
| `TestRunRouter_SIGTERMLifecycle` | AC-004 | Live ctx; wait for socket; cancel; assert `runRouter` returns nil within 2s. |

## Tier-3 smoke transition

Before this story: `test/smoke/tier3-tutorial.sh` T3-2-router expected
`runRouter: not implemented` in the log and reported exit code 3
(re-emergent expected-fail).

After this story:

- T3-2-router runs the real binary against the tutorial-shape config
  (with `listen_addr` sed-substituted to a random `127.0.0.1:${PORT}`
  in [40000,49999) to avoid host-9090 collisions with Prometheus /
  OrbStack / etc., and `management_socket` sed-substituted to a
  tmpdir path).
- Exit code 0 or 143 (SIGTERM) is PASS; a `not implemented` return is
  a REGRESSION.
- Tier 3 exits 0 clean-pass. Exit 3 is now reserved for a future
  re-emergent expected-fail; currently unused.

Verified end-to-end: `bash test/smoke/tier3-tutorial.sh; echo "EXIT=$?"`
reports `Tier 3: 4 passed, 0 expected-failed, 0 unexpected-failed` +
`EXIT=0`.

## Out of Scope

- **E → PE runtime graduation.** The `upstream_routers:` field is
  accepted but does not yet select between modes.
- **SIGHUP config reload** (S-6.04).
- **Real frame accept + forward** (S-BL.NI network-ingress; S-BL.OA
  outer-assembler). The accept-and-close loop exists so the port is
  bindable; frame processing is a future story.
- **Router-to-router peer session establishment.**
- **DRAIN-signal-to-nodes protocol** (S-7.04 Wave 7).
- **Resolving the stdout/stderr docs discrepancy.** Flagged in the PR
  body but not fixed here — a separate story decides whether to
  change the tutorial or the code.

## Notes

Task #144 (pre-Plan-B era) originally tracked the router-daemon-stub
gap; the tier-3 harness carried a `#144` expected-fail annotation
throughout Plan-B rollout. This story is that task's successor: it
closes the underlying gap, retires the tier-3 annotation, and rolls
the annotation into an unused-slot in the exit-code contract for
symmetry with future re-emergent expected-fails.

## Traceability

| AC | BC / Spec anchor | Test |
|----|-----|------|
| AC-001 | S-6.06 AC-004 (role-exclusion); ADR-004; ARCH-04; F-P2L1-001 / F-P2L1-002 (register-before-serve) | `TestRunRouter_StartsWithMgmt`, `TestRunRouter_NoAdminHandlers` |
| AC-002 | BC-2.09.003 (`ListenAddr` config surface); ARCH-01 §Goroutine WaitGroup Contract | `TestRunRouter_DataListenerBinds` |
| AC-003 | operator-facing log contract (paired with `docs/getting-started.md` §2) | (observed in smoke via `grep -c 'switchboard router:'`) |
| AC-004 | Ruling J (mgmt-start failure aborts); main.go signal.NotifyContext contract | `TestRunRouter_SIGTERMLifecycle` |

## Related work

- **Depends on:** S-6.06 (daemon-admin-handlers — establishes the
  `newMgmtServer` / `wireMetricsHandlers` / `serveMgmtServer` shape
  and the role-exclusion rule router must honor).
- **Blocks:** S-BL.NI (network ingress can now target a bound port),
  S-BL.OA (outer-assembler can now target a running router).
- **Predecessor:** task #144 (retired).
