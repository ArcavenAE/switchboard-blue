---
artifact_id: S-W5.02-mgmt-e2e-harness
document_type: story
level: ops
story_id: S-W5.02
title: "e2e management plane integration harness: sbctl authenticate + RPC across all four daemon types"
status: draft
producer: story-writer
timestamp: 2026-06-30T00:00:00
phase: 2
epic: E-6
wave: 5
priority: P0
scope_phase: E
estimated_points: 5
version: "1.2"
bc_traces:
  - BC-2.07.002
vp_traces: [VP-049]
subsystems: [network-management]
architecture_modules: [internal/mgmt, cmd/sbctl, cmd/switchboard]
tdd_mode: strict
target_module: internal/testenv
cycle: v1.0.0-greenfield
depends_on: [S-6.03, S-W5.01, S-6.06]
blocks: []
inputDocuments:
  - '.factory/specs/behavioral-contracts/ss-07/BC-2.07.002.md'
  - '.factory/specs/architecture/ARCH-12-daemon-management-plane.md'
  - '.factory/specs/architecture/ARCH-05-cli-and-api.md'
  - '.factory/specs/verification-properties/VP-049.md'
acceptance_criteria_count: 5
# BC status: active — all BCs are final and assigned
---

# S-W5.02: E2E Management Plane Integration Harness Across All Four Daemon Types

> **Execute:** `/vsdd-factory:deliver-story S-W5.02`

## Scope Note

This story is the **convergence gate** for the Wave-5 management plane. All of
S-6.03 (sbctl client auth), S-W5.01 (internal/mgmt server + config +
cmd/switchboard wiring), and S-6.06 (daemon-side admin RPC handlers) MUST be
merged before this story begins.

The story delivers a single integration test that spins up four `mgmt.Server`
instances — one per daemon type — with distinct per-mode handler tables, then
exercises the full `sbctl connect → Authenticate() → RPC → disconnect` cycle
against each. This verifies VP-049 (e2e across all four daemon types)
end-to-end.

**Daemon-mode axis (Q1 ruling — Option A):** `runRouter` and `runConsole` in
`cmd/switchboard/main.go` are currently stubs returning `errors.New("not
implemented")`. Per-daemon `runXxx` entrypoint wiring is therefore NOT the
test vehicle for this story. Instead, the harness instantiates four
`mgmt.NewServer` instances directly, each with a distinct handler table
reflecting the real per-mode handler set (see §Architecture Compliance Rules).
Per-daemon `runXxx` end-to-end wiring is deferred to Wave-6; a new VP will
be minted (VP-VW6.NN) at that time. VP-049 §Feasibility (pending spec-steward
burst) will be updated to document this boundary explicitly.

No new production code is introduced beyond the test harness infrastructure in
`internal/testenv` (or `internal/mgmt/e2e_test.go`).

## Behavioral Contracts

| BC | Version | Title | PCs covered |
|----|---------|-------|------------|
| BC-2.07.002 | v1.4 | sbctl Unified CLI for All Four Daemon Types with OpenSSH Key Authentication | PC-1 (connect), PC-2 (auth), PC-3 (execute if authenticated — incl. Rulings M/U/X), Inv-1 (all subcommands authenticated), Inv-2 (sbctl not a daemon) |

## Narrative

- **As an** operator
- **I want** the full management plane (sbctl client + daemon server) to be
  verified end-to-end across all four daemon types in a single integration test
- **So that** I can be confident the authentication handshake and RPC dispatch
  work correctly against every daemon mode before shipping Wave 5

## Acceptance Criteria

### AC-001 (traces to BC-2.07.002 postcondition 1 — connect to all four daemon types)
The integration test starts four `mgmt.NewServer` instances (one per daemon
type: router, access, console, control) with distinct per-mode handler tables,
each listening on a unique Unix socket. Each server starts before the test
connects.
- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` — setup step: start
  all four servers and poll with 1s timeout until each management socket is
  ready.

### AC-002 (traces to BC-2.07.002 postcondition 2 — authenticate via OpenSSH key, VP-049)

**Primary sub-test (distinct-operator-key, Q5 ruling — Option B):** Generate
a daemon Ed25519 key pair and a separate operator Ed25519 key pair in-process.
Construct `mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})` so the
daemon only authorizes the operator key (not its own key). Call
`client.Authenticate(conn, operatorPriv)` and verify it returns nil for each
of the four daemon sockets. This is the primary coverage path for VP-049.

**Bootstrap-mode sub-test (ADR-012 §bootstrap):** In a second variant, use
`mgmt.NewOperatorKeySet(nil)` (daemon key is sole authorized key) and
authenticate with the daemon's own key. Bootstrap mode is a real production
path (initial daemon setup before operator keys are registered); coverage by
unit tests in `internal/mgmt/mgmt_test.go` is supplemented here at the e2e
layer.

- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` sub-tests per daemon
  (distinct-operator variant); `TestE2E_MgmtPlane_BootstrapAuth_VP049` for the
  bootstrap variant.

### AC-003 (traces to BC-2.07.002 postcondition 3 — operation executed if authenticated, Rulings M/U/X)
After successful authentication, an RPC request is dispatched to each daemon's
management server and a valid response is received. The request envelope MUST
carry `"type":"request"` (Ruling M). The request ID MUST be a non-constant
per-call value (e.g., hex-encoded `time.Now().UnixNano()` or a UUID) — not the
constant `"t1"` (Ruling X). After receiving the response, assert:

1. `resp.Type == "response"` (Ruling U — wrong-type response with `"ok":true`
   MUST NOT be silently accepted)
2. `resp.ID == req.ID` (Ruling X — ID echo check; mismatch MUST return
   E-RPC-001)
3. `resp.Ok == true`
4. `resp` contains a `"data"` field (even if empty/minimal for MVP stub
   handlers)

- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` sub-test per daemon:
  send a `"status"` RPC after auth using a random request ID; assert all four
  conditions above.

### AC-004 (traces to BC-2.07.002 invariant 1 — all subcommands authenticated)
A connection that skips authentication and sends an RPC request directly to any
of the four daemon types receives AUTH_FAIL + connection close. Verified against
at least one daemon type in the integration harness.
- **Test:** `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004` — for the router
  daemon (representative), send `{"type":"request",...}` without completing the
  handshake; verify AUTH_FAIL response and connection close.

### AC-005 (traces to BC-2.07.002 invariant 2 — sbctl exits after command completion, Q6 ruling — Option A)
After RPC response is received, the sbctl client dispatch path (`connectAndRun`
with `defer conn.Close()`) closes the connection from the client side. This is
verified using a server-side listener wrapper that observes the client-side
FIN/close event within 500ms of RPC completion.

Implementation: wrap the `net.Listener` passed to `mgmt.NewServer` with a
`closingListener` that records when each accepted `net.Conn` has its
remote-side closed (by counting `Read` returning `io.EOF` or equivalent after
dispatch returns). After the per-daemon sub-test completes, assert the wrapper
observed close within 500ms.

This instruments the actual production close code (`defer conn.Close()` in
`cmd/sbctl/client.go:connectAndRun`), not a tautological local-side close
check.
- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` — per-daemon
  sub-test: after RPC, assert `listenerWrapper.ClientClosedWithin(500ms)`.

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| Test harness / server setup | internal/mgmt (e2e_test.go) | effectful (socket management, goroutines) |
| sbctl client under test | cmd/sbctl/client.go | effectful |
| mgmt.Server under test | internal/mgmt/mgmt.go | effectful |
| Per-mode handler tables | internal/mgmt (test-defined) | pure (handler registration) |

## Test Infrastructure Notes

This story requires the full management stack to be available:
- `internal/mgmt.Server` (from S-W5.01) — the server under test
- `cmd/sbctl.Authenticate()` (from S-6.03) — the client under test
- `internal/mgmt` handler registration API (from S-W5.01/S-6.06)

**In-process test approach:** Use local Unix sockets (via `t.TempDir()`)
within the test process, calling `mgmt.NewServer` directly with distinct
per-mode handler tables. The `testenv.NewFull` reference in prior spec
drafts was aspirational and does not exist in the repo; disregard it. The
concrete approach is:

```
// Per-daemon in-process setup (pseudocode)
ln, _ := net.Listen("unix", filepath.Join(t.TempDir(), "router.sock"))
wrapped := newClosingListenerWrapper(ln)  // for AC-005
srv := mgmt.NewServer(wrapped, daemonKey, ops, routerHandlers())
go srv.Serve(ctx)
t.Cleanup(func() { srv.Shutdown(ctx) })
```

**Per-mode handler tables (Q1 ruling):** Each of the four `mgmt.Server`
instances is created with a handler table that reflects its daemon mode's
actual registered subcommands. Suggested minimal differentiation:

| Daemon | Handler registered | Response stub |
|--------|--------------------|--------------|
| router | `"paths.list"`, `"status"` | `{"ok":true,"data":{}}` |
| access | `"session.list"`, `"status"` | `{"ok":true,"data":{}}` |
| console | `"console.status"`, `"status"` | `{"ok":true,"data":{}}` |
| control | `"admin.key.list"`, `"status"` | `{"ok":true,"data":{}}` |

This ensures per-mode handler-registration differences are exercised even
though `runXxx` entrypoints are not the test vehicle.

**Build tag:** Use `//go:build integration` on the e2e test file.
Integration tests run explicitly: `go test -tags=integration ./...`.

## Edge Cases

| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| EC-001 | Daemon management socket not ready at test start | Test polls with 1s timeout; if not ready, test fails with descriptive error (not panic) |
| EC-002 | Auth with correct key against wrong daemon type | AUTH_OK still received (all four daemon types use the same ADR-012 handshake) |
| EC-003 | e2e test run with no `--key` arg (default key absent) | Test generates all keys in-process; does not depend on `~/.ssh/id_ed25519` existing |
| EC-004 | Daemon startup fails (port conflict) | Test cleanup correctly closes all listeners; no resource leak |
| EC-005 | Response carries wrong `resp.Type` (e.g., `"rpc_response"`) | `dispatch()` returns E-RPC-001; test asserts this is not silently accepted |
| EC-006 | Response carries `resp.ID` != `req.ID` | `dispatch()` returns E-RPC-001; test asserts mismatch is detected |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| internal/mgmt e2e harness | effectful | Server goroutines, socket management, process lifecycle |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~1,800 |
| BC-2.07.002.md (v1.4) | ~1,000 |
| ARCH-12 §Wiring into cmd/switchboard | ~800 |
| ARCH-05 §Daemon Management Socket | ~400 |
| VP-049.md (proof harness skeleton) | ~600 |
| internal/mgmt/mgmt.go (from S-W5.01) | ~2,500 |
| cmd/sbctl/client.go (from S-6.03) | ~1,000 |
| cmd/switchboard/main.go (from S-W5.01 wiring) | ~2,000 |
| Test files (estimated) | ~2,500 |
| Tool outputs overhead | ~400 |
| **Total** | **~13,000** |
| Agent context window | 200K |
| **Budget usage** | **~6.5%** |

## Tasks (MANDATORY)

1. [ ] Confirm S-6.03, S-W5.01, and S-6.06 are all merged to develop before starting
2. [ ] Read BC-2.07.002 (v1.4), ARCH-12 §Wiring, ARCH-05 §Daemon Management Socket, VP-049.md
3. [ ] Read `internal/mgmt/mgmt.go` (S-W5.01 implementation) to understand `NewServer` API
4. [ ] Read `cmd/sbctl/client.go` (S-6.03 implementation) to understand `Authenticate()`, `dispatch()`, and `connectAndRun()` API (confirm `defer conn.Close()` in `connectAndRun`)
5. [ ] Read `cmd/switchboard/main.go` to confirm `runRouter` / `runConsole` are stubs (expected per Q1 ruling)
6. [ ] Write failing tests for AC-001 through AC-005 (`//go:build integration` tag)
7. [ ] Verify Red Gate — tests must fail before any implementation
8. [ ] Implement `closingListenerWrapper` helper (for AC-005 server-side FIN observation)
9. [ ] Implement per-mode handler table constructors: `routerHandlers()`, `accessHandlers()`, `consoleHandlers()`, `controlHandlers()`
10. [ ] Implement test harness infrastructure (distinct-operator-key setup, socket path generation, ready-poll with timeout)
11. [ ] Implement `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` covering AC-001, AC-002 (distinct-operator), AC-003 (Rulings M/U/X), AC-005
12. [ ] Implement `TestE2E_MgmtPlane_BootstrapAuth_VP049` covering AC-002 bootstrap variant
13. [ ] Implement `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004`
14. [ ] Verify all tests pass: `go test -tags=integration ./...`
15. [ ] `just fmt && just lint` pass

## Previous Story Intelligence (MANDATORY)

| Story | Key Decisions | Patterns Established | Gotchas Discovered |
|-------|--------------|---------------------|-------------------|
| S-6.03 | `Authenticate()` is fail-closed; `--key` loads OpenSSH Ed25519 PEM; `connectAndRun` uses `defer conn.Close()` | `client.Authenticate(conn, privKey) error` API; `dispatch()` generates non-constant request ID per Ruling X | Use `net.Pipe` or Unix sockets for in-process tests; avoids OS socket complexity |
| S-W5.01 | `mgmt.NewServer(ln, daemonKey, ops, handlers)` API; WaitGroup-tracked goroutine | `Server.Serve(ctx)` + `Server.Shutdown(ctx)` | Must call `Shutdown` in test cleanup to drain connections and avoid goroutine leaks |
| S-W5.01 | `runAccess` + `runControl` wire mgmt via `startMgmtServer`; `runRouter` + `runConsole` are stubs | Per-mode handler set is the differentiator, not `runXxx` | Socket paths must be unique per test run (use `t.TempDir()` for Unix socket paths) |
| S-W3.04 | Daemon assembly pattern; all subsystems wired with `wg.Add(1)` | In-process daemon setup for tests | Do not depend on global state; each test sub-tree gets its own server instance |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|-------------|
| Test uses `//go:build integration` tag | ARCH-12 §Story 3 scope; VP-049.md harness | Build tag in file header |
| Test generates all Ed25519 keys in-process (no dependency on `~/.ssh/id_ed25519`) | EC-003 | Key generation in test setup |
| `Server.Shutdown(ctx)` called in `t.Cleanup` for each daemon server instance | ARCH-01 §Goroutine WaitGroup Contract; no goroutine leak | `t.Cleanup(func() { srv.Shutdown(ctx) })` |
| Unix socket paths unique per test run | OS socket uniqueness | `t.TempDir()` for socket directory |
| AC-002 primary case uses DISTINCT operator key (not bootstrap) | Q5 ruling — Option B; VP-049 assertion strength | `mgmt.NewOperatorKeySet([]ed25519.PublicKey{operatorPub})` with separate `operatorPriv` |
| AC-003 request ID is non-constant per-call (not `"t1"`) | BC-2.07.002 v1.4 Ruling X | `id` = hex `time.Now().UnixNano()` or equivalent |
| AC-003 asserts `resp.Type == "response"` explicitly (not just `resp.Ok`) | BC-2.07.002 v1.4 Ruling U | Assert both fields; wrong-type with `"ok":true` must be caught |
| AC-005 uses server-side listener wrapper to observe FIN; NOT a tautological local Close | Q6 ruling — Option A | `closingListenerWrapper` tracks `io.EOF` on accepted conns after dispatch |
| Four `mgmt.Server` instances have distinct per-mode handler tables | Q1 ruling — Option A | Each server created with its own `handlerTable`; per §Test Infrastructure Notes |
| Test does NOT invoke `runRouter` or `runConsole` as the test vehicle | Q1 ruling — daemon-mode axis deferred to Wave-6 | Instantiate `mgmt.NewServer` directly |

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Per go.mod (mise-pinned) |
| `crypto/ed25519` | stdlib | In-process test key generation (daemon key + separate operator key) |
| `crypto/rand` | stdlib | Key generation |
| `net` | stdlib | `net.Listen` (Unix), `closingListenerWrapper` for AC-005 |
| `context` | stdlib | Test context with timeout |
| `testing` | stdlib | `t.Cleanup`, `t.Parallel`, `t.Run` |
| `encoding/json` | stdlib | RPC message construction and parsing in test harness |
| `encoding/hex` or `fmt` | stdlib | Non-constant request ID generation (`hex.EncodeToString(...)` or `fmt.Sprintf("%x", time.Now().UnixNano())`) |

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `internal/mgmt/e2e_test.go` | create | `//go:build integration` e2e test: `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049`, `TestE2E_MgmtPlane_BootstrapAuth_VP049`, `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004` |
| `internal/mgmt/testhelpers_test.go` | create | `closingListenerWrapper`, `routerHandlers()`, `accessHandlers()`, `consoleHandlers()`, `controlHandlers()` helper code used only by the e2e test |

## Adversary Pass-1 Rulings

> Source: Pass-1 adversarial review, findings L1 F-001/F-002, L2 F-001/F-003/F-004, L3 F-001/F-005/F-006/F-007.
> Ruling adjudicator: product-owner. Date: 2026-06-30.

| Q# | Finding(s) | Option Chosen | Rationale |
|----|-----------|---------------|-----------|
| Q1 | L1 F-002, L2 F-003, L3 F-007 — daemon-mode axis stub (`runRouter`/`runConsole` not implemented) | **A — Narrow VP-049** | Wave-5 convergence gate must not slip on undelivered per-daemon entrypoints. VP-049's assertion power is on the mgmt.Server contract (uniform across all four modes per ARCH-12), not on `runXxx` wiring. Per-daemon `runXxx` e2e deferred to Wave-6 (new VP-VW6.NN). Handler-table differentiation per mode preserves meaningful coverage of per-mode registration differences. Spec-steward burst will add §Feasibility paragraph to VP-049. |
| Q2 | L2 F-001, L3 F-006 — BC v1.2 pin in story does not reflect Rulings U/X from BC v1.4 | **A — Bump pin to v1.4, extend AC-003** | Rulings U and X are cheap to add and close the anchor gap cleanly. AC-003 now asserts `resp.Type`, `resp.ID` echo, and uses non-constant request ID. No rationale to defer. |
| Q3 | L3 F-001 — BC-2.07.002 §Verification Properties rows 139-140 labeled VP-049 are phantom rows | **A — Delete phantoms; mint new VP IDs if needed** | Rows 139-140 are pre-VP-063/VP-062 leftovers that create false traceability. Deleting them is correct. New VP-JSON-COVERAGE and VP-SBCTL-NOT-DAEMON can be minted as follow-up if coverage is desired. BC edit is deferred to spec-steward burst. |
| Q4 | L3 F-005 — VP-049.md has no `implementing_story`, no §Story Trace, references non-existent `testenv.NewFull` | **A — VP-049 v1.0→v1.1 in spec-steward burst; story drops `testenv.NewFull` reference** | Story §Test Infrastructure Notes updated to specify the in-process `mgmt.NewServer` approach concretely. VP-049 update is deferred to spec-steward burst (not this story). |
| Q5 | L2 F-004 — bootstrap-mode auth simulation (`NewOperatorKeySet(nil)`) may be insufficient coverage | **B — Add distinct-operator-key sub-test as primary; bootstrap stays as variant** | Trivial to add (generates a second key pair in-process). Strengthens VP-049 assertion power. Bootstrap-mode sub-test remains as `TestE2E_MgmtPlane_BootstrapAuth_VP049`; distinct-operator variant is the primary AC-002 case. |
| Q6 | L1 F-001 — AC-005 assertion is tautological (local Close → `net.ErrClosed` synchronously) | **A — Server-side listener wrapper observing client FIN** | Option A instruments the actual production `defer conn.Close()` in `connectAndRun`. Option B (subprocess) defers the invariant to a different story scope unnecessarily. `closingListenerWrapper` is ~10 lines of test helper code. |

## Downstream Fix-Burst Punch List

The following tasks are OUT OF SCOPE for the implementer/test-writer acting on
this story and MUST be dispatched to other agents in a subsequent burst:

1. **spec-steward:** Update VP-049 v1.0 → v1.1 — add `implementing_story: S-W5.02`, add §Story Trace section, rewrite proof harness skeleton to in-process pattern (Q4 ruling). Also delete phantom rows 139-140 from BC-2.07.002 §Verification Properties (Q3 ruling, per append_only_numbering policy — retire with strikethrough or delete as phantoms that were never formally minted VP IDs).
2. **product-owner (or spec-steward):** Add §Feasibility paragraph to VP-049 documenting that VP-049 covers mgmt.Server contract via direct instantiation; per-daemon `runXxx` wiring coverage deferred to Wave-6 VP-VW6.NN (Q1 ruling). Mint VP-VW6.NN in VP-INDEX as a placeholder.
3. **state-manager:** Update STORY-INDEX v3.16 → v3.17 with S-W5.02 version bump (v1.1 → v1.2) in the row and changelog.
4. **worktree / implementer:** After this story's spec lands on `develop`, update the worktree (`git rebase develop` or re-checkout) and proceed with the implementation under the revised AC set.

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.2 | 2026-06-30 | product-owner | Adversary Pass-1 rulings applied (Q1-Q6). Q1: narrow VP-049 to mgmt.Server contract with per-mode handler tables; per-daemon `runXxx` wiring deferred to Wave-6. Q2: BC pin bumped v1.2→v1.4; AC-003 extended with Rulings M/U/X (non-constant request ID, resp.Type assertion, resp.ID echo). Q3: phantom VP rows 139-140 flagged for spec-steward deletion. Q4: `testenv.NewFull` reference removed; in-process approach specified concretely. Q5: AC-002 now has distinct-operator-key as primary sub-test + bootstrap variant. Q6: AC-005 rewritten to use server-side `closingListenerWrapper` observing client FIN within 500ms. §Adversary Pass-1 Rulings section added. Task list updated (Tasks 8-15 replacing 8-12). File structure updated (testhelpers_test.go added). |
| 1.1 | 2026-06-29 | product-owner | Add S-6.06 to `depends_on`. Adversary Pass 1 on S-6.02 found CRITICAL gap F-001: daemon-side admin RPC handlers (admin.key.register / revoke / expire / list-keys) were never wired; `startMgmtServer(..., nil)` at every call site. S-6.06 (minted per CR-W5-SCOPE-SPLIT ruling) closes that gap. S-W5.02 cannot exercise admin RPC paths end-to-end until S-6.06 is merged. |
| 1.0 | 2026-06-28 | story-writer | Initial creation — Wave-5 net-new story per ARCH-12 product-owner handoff. E2E integration gate for management plane; implements VP-049 across all four daemon types. Depends on S-6.03 + S-W5.01. |
