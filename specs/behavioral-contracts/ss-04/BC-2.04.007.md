---
artifact_id: BC-2.04.007
document_type: behavioral-contract
level: L3
version: "1.0"
status: active
producer: product-owner
timestamp: 2026-06-27T00:00:00Z
phase: 1a
bc_id: BC-2.04.007
subsystem: session-access
architecture_module: cmd/switchboard
capability: CAP-013
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v1.0.0-greenfield
modified: []
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '.factory/specs/prd-supplements/error-taxonomy.md'
  - '.factory/specs/architecture/ARCH-01-core-services.md'
  - '.factory/specs/architecture/ARCH-08-dependency-graph.md'
  - '.factory/stories/S-W3.04-daemon-assembly.md'
traces_to: [CAP-013]
kos_anchors:
  - elem-node-router-architecture
  - elem-single-binary-three-modes
---

# BC-2.04.007: Access Node Daemon Startup Succeeds or Exits Non-Zero; SIGTERM/SIGINT Triggers Clean Shutdown

## Description

The access-node binary (`cmd/switchboard`) has two lifecycle obligations observable
by the OS process supervisor: (1) if `SessionConnector.Connect(ctx)` fails (both
control-mode and PTY fallback fail), the daemon logs the error and exits with a
non-zero exit code before entering its relay loop; (2) on receipt of SIGTERM or
SIGINT, the daemon cancels its root context, waits for all goroutines spawned after
connect to drain, and exits with code 0. These are observable contracts — the exit
code and signal handling behaviour are tested by the implementer and validated by
operators and process supervisors.

## Preconditions

**PC-1 (connect-failure path):**
1. The access-node binary has been launched.
2. `SessionConnector.Connect(ctx)` has been called; it returns a non-nil error
   (both tmux control mode and PTY proxy fallback have failed).

**PC-2 (clean-shutdown path):**
1. The access-node binary is running; `sc.Connect(ctx)` has returned nil.
2. The relay goroutine (`sc.Frames()` → `accessNode.DeliverFrame()`), the sweep
   ticker goroutine, and the frames-dropped ticker goroutine are all running.
3. SIGTERM or SIGINT is delivered to the process.

## Postconditions

**PC-1 postconditions (connect failure):**
1. The error is logged at ERROR level to stderr (E-SYS-002 format — see Error Codes).
2. A human-readable diagnostic message is written to stderr: `"fatal: cannot connect
   to session backend: <reason>"`.
3. The process exits with a non-zero exit code (exit 1).
4. The process does not panic.
5. No relay goroutines are started.

**PC-2 postconditions (clean shutdown):**
1. The root context is cancelled; `ctx.Done()` closes.
2. The relay goroutine exits when `sc.frames` is closed (via `sc.Close()` in the
   shutdown sequence).
3. The sweep ticker and frames-dropped ticker goroutines observe `ctx.Done()` and
   exit within one ticker period.
4. `sc.Close()` is called exactly once (enforced by `sync.Once` per ADR-011).
5. The process exits with code 0.
6. No goroutines are leaked (verified by test with `t.Cleanup` + short timeout).
7. No panic occurs.

## Invariants

1. **Fail-before-serve**: The daemon MUST NOT enter its relay loop if `Connect(ctx)`
   fails. No partial startup state is left behind.
2. **Exactly-once close**: `sc.Close()` is called at most once on shutdown, regardless
   of whether the shutdown is triggered by signal or by connect failure.
3. **Signal idempotency**: A second SIGTERM/SIGINT after the shutdown sequence has
   started does not cause a second `cancel()` call to panic or a second `sc.Close()`
   call — `sync.Once` and context cancellation are both idempotent.
4. **DI-001 compliance**: Even during shutdown, session content is not exposed to the
   router — `sc.Close()` closes the forwarding channel cleanly before any goroutines
   are torn down.

## Trigger

- `SessionConnector.Connect(ctx)` returns non-nil error (PC-1 path).
- OS signal SIGTERM or SIGINT delivered to the process (PC-2 path).

## Error Codes

| Code | Condition | Severity | Exit Code |
|------|-----------|----------|-----------|
| E-SYS-002 | `sc.Connect(ctx)` failed; both ctrl and PTY paths exhausted | broken | 1 |

> **Note on E-SYS-002:** This code is newly registered in this BC. It is distinct
> from E-SYS-001 (PTY device unavailable at OS level). E-SYS-002 covers the
> aggregate connect failure at the `SessionConnector` level, after both control-mode
> and PTY fallback have been tried and failed.
>
> Message format: `"fatal: cannot connect to session backend: <reason>"`
>
> See error-taxonomy.md §SYS for the full catalog. E-SYS-002 must be added there.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | tmux not installed; PTY also unavailable | E-SYS-001 (PTY unavailable) surfaces inside Connect(); BC-2.04.007 PC-1 fires: log + exit 1. |
| EC-002 | SIGTERM arrives before `sc.Connect(ctx)` returns | Context is cancelled; `Connect()` returns early with a context-cancelled error; PC-1 path fires (non-nil connect error); exit 1. |
| EC-003 | SIGTERM arrives during relay (normal case) | PC-2 path: context cancelled; relay goroutine exits on `sc.frames` close; sweep and dropped-log tickers exit on `ctx.Done()`; `sc.Close()` called; exit 0. |
| EC-004 | Second SIGINT arrives while shutdown is already in progress | First signal fires `cancel()`; second signal hits the same `sync.Once`-guarded cancel path — no double-close or panic. |
| EC-005 | One ticker goroutine is blocked when ctx is cancelled | Goroutine exits at the next `select` that includes `<-ctx.Done()` — within one tick period. Test enforces a deadline via `t.Cleanup` + `time.AfterFunc`. |
| EC-006 (FM-004) | tmux control-mode drops mid-relay (after successful Connect) | This is BC-2.04.002's domain: `SessionConnector` activates PTY fallback transparently. The relay channel remains open. BC-2.04.007 is not triggered. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sc.Connect(ctx)` returns `errors.New("no backend available")` | stderr: `"fatal: cannot connect to session backend: no backend available"`; exit code 1; no panic | happy-path (PC-1) |
| SIGTERM sent to running daemon | root ctx cancelled; all goroutines drain within 100ms; `sc.Close()` called once; exit code 0 | happy-path (PC-2) |
| SIGINT sent to running daemon | same as SIGTERM row | happy-path (PC-2) |
| `sc.Connect(ctx)` fails while ctx already cancelled (EC-002) | PC-1 path: log error, exit 1 — context cancellation is the underlying reason | edge-case |
| Two SIGTERM signals back-to-back | One shutdown, no double-close panic, exit 0 | edge-case |

## Verification Properties

| VP | Property | Proof Method |
|----|----------|-------------|
| VP-060 | Connect failure always exits non-zero before relay goroutines start | integration (subprocess) |
| VP-060 | SIGTERM/SIGINT triggers clean shutdown: all goroutines drain, exit 0, no leak | integration (subprocess + t.Cleanup timeout) |

> VP-060 is a new VP covering cmd/switchboard lifecycle. It must be registered in
> VP-INDEX and ARCH-11-verification-coverage-matrix.md by the architect.

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 |
| Capability Anchor Justification | CAP-013 ("Access node tmux session publishing") per capabilities.md §CAP-013 — the access node's connect-and-start and signal-driven-stop lifecycle directly frames *when* and *whether* CAP-013's publishing behaviour operates. A daemon that fails silently on connect, or leaks goroutines on shutdown, violates the operational contract of CAP-013 even though CAP-013 does not enumerate these lifecycle preconditions explicitly. No other existing CAP better captures binary-level startup/shutdown semantics; a new CAP-029 ("Daemon process lifecycle correctness") could be registered in a future domain-spec revision if lifecycle semantics expand, but for Wave 3 scope, CAP-013 is the best available anchor. |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — enforced even during shutdown teardown) |
| Architecture Module | cmd/switchboard |
| Architecture Doc | ARCH-01-core-services.md (daemon lifecycle §ADR-010, ADR-011) |
| Stories | S-W3.04 (AC-007, AC-008) |
| Capability Anchor Note | If domain-spec revisions introduce a dedicated daemon-lifecycle CAP (CAP-029), this BC should be re-anchored and the Traceability updated. Flag for next domain-spec maintenance sweep. |

## Related BCs

- BC-2.04.001 — depends on: connect succeeds before publishing begins
- BC-2.04.002 — composes with: PTY fallback is the last resort before PC-1 fires
- BC-2.09.003 — parallel: router startup fails cleanly on bad config (same class of lifecycle contract, different daemon, different subsystem)

## Architecture Anchors

- ARCH-01-core-services.md §ADR-010 (tmux/PTY backend selection), §ADR-011 (Frames() forwarding channel)
- ARCH-08-dependency-graph.md §6.5.1 (wiring obligations for cmd/switchboard)

## Story Anchor

S-W3.04 — AC-007 traces to PC-1 postconditions; AC-008 traces to PC-2 postconditions.

## VP Anchors

VP-060 (new — to be registered by architect per bc_array_changes_propagate_to_body_and_acs and vp_index_is_vp_catalog_source_of_truth policies).
