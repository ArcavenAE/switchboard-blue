---
artifact_id: S-W5.02-mgmt-e2e-harness
document_type: story
level: ops
story_id: S-W5.02
title: "e2e management plane integration harness: sbctl authenticate + RPC across all four daemon types"
status: draft
producer: story-writer
timestamp: 2026-06-28T00:00:00
phase: 2
epic: E-6
wave: 5
priority: P0
scope_phase: E
estimated_points: 5
version: "1.1"
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

This story is the **convergence gate** for the Wave-5 management plane. Both
S-6.03 (sbctl client auth) and S-W5.01 (internal/mgmt server + config +
cmd/switchboard wiring) MUST be merged before this story begins.

The story delivers a single integration test that spins up all four daemon types
with management listeners, then exercises the full `sbctl connect → Authenticate()
→ RPC → disconnect` cycle against each daemon type. This verifies VP-049 (e2e
across all four daemon types) end-to-end.

No new production code is introduced by this story beyond the test harness
infrastructure in `internal/testenv` (or a new `internal/mgmt/testenv` helper).

## Behavioral Contracts

| BC | Title | PCs covered |
|----|-------|------------|
| BC-2.07.002 | sbctl Unified CLI for All Four Daemon Types with OpenSSH Key Authentication | PC-1 (connect), PC-2 (auth), PC-3 (execute if authenticated), Inv-1 (all subcommands authenticated), Inv-2 (sbctl not a daemon) |

## Narrative

- **As an** operator
- **I want** the full management plane (sbctl client + daemon server) to be
  verified end-to-end across all four daemon types in a single integration test
- **So that** I can be confident the authentication handshake and RPC dispatch
  work correctly against every daemon mode before shipping Wave 5

## Acceptance Criteria

### AC-001 (traces to BC-2.07.002 postcondition 1 — connect to all four daemon types)
The integration test starts all four daemon types with management listeners
(router, access, console, control) either as in-process goroutines using the
daemon `runXxx` functions or as subprocess binaries. Each daemon starts its
management listener before the test connects.
- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` — setup step: start
  all four daemons and wait for each management socket to be ready (poll with
  timeout).

### AC-002 (traces to BC-2.07.002 postcondition 2 — authenticate via OpenSSH key, VP-049)
Using a test-generated Ed25519 key pair (or an OpenSSH key written to a temp
file), `sbctl` (or the `Authenticate()` function directly) successfully
completes the ADR-012 challenge-response handshake against each of the four
running daemon management sockets.
- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` sub-test per daemon:
  call `client.Authenticate(conn, privKey)` and verify it returns nil for each
  of the four daemon sockets. If testing via subprocess `sbctl` binary, verify
  exit code 0.

### AC-003 (traces to BC-2.07.002 postcondition 3 — operation executed if authenticated)
After successful authentication, an RPC request (`{"type":"request","id":"t1","command":"status","args":{}}`)
is dispatched to each daemon's management server and a response with `"ok":true`
is received. The response contains a `"data"` field (even if empty/minimal for
MVP stub handlers).
- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` sub-test per daemon:
  send a `"status"` RPC request after auth; verify `"ok":true` in the response
  envelope.

### AC-004 (traces to BC-2.07.002 invariant 1 — all subcommands authenticated)
A connection that skips authentication and sends an RPC request directly to any
of the four daemon types receives AUTH_FAIL + connection close. This is verified
against at least one daemon type in the integration harness.
- **Test:** `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004` — for the router
  daemon (representative), send a `{"type":"request",...}` without completing
  the handshake; verify AUTH_FAIL response and connection close.

### AC-005 (traces to BC-2.07.002 invariant 2 — sbctl exits after command completion)
After the RPC response is received, the sbctl client (or `client.go` dispatch
function) closes the connection and does not remain running as a daemon.
- **Test:** `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` cleanup: after each
  per-daemon sub-test, verify the client-side connection is closed (read returns
  error) within 500ms.

## Architecture Mapping

| Component | Module | Pure/Effectful |
|-----------|--------|---------------|
| Test harness / testenv | internal/testenv (or new internal/mgmt/testenv_test.go) | effectful (subprocess/in-process daemon startup) |
| sbctl client under test | cmd/sbctl/client.go | effectful |
| mgmt.Server under test | internal/mgmt/mgmt.go | effectful |
| Daemon modes under test | cmd/switchboard | effectful |

## Test Infrastructure Notes

This story requires the full management stack to be available:
- `internal/mgmt.Server` (from S-W5.01) — the server under test
- `cmd/sbctl.Authenticate()` (from S-6.03) — the client under test
- `cmd/switchboard` daemon mode `runXxx` functions (from S-W5.01) wired with mgmt listeners

**In-process test approach (recommended):** Use `net.Pipe` pairs or local Unix
sockets within the test process, calling the `runXxx`-equivalent setup directly
to avoid subprocess management complexity. The `internal/testenv` pattern (from
VP-049.md proof harness `testenv.NewFull`) is the reference.

**Build tag:** Use `//go:build integration` on the e2e test file so it does not
run in the normal `just test` flow. Integration tests run explicitly:
`go test -tags=integration ./...`.

## Edge Cases

| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| EC-001 | Daemon management socket not ready at test start | Test polls with 1s timeout; if not ready, test fails with descriptive error (not panic) |
| EC-002 | Auth with correct key against wrong daemon type | AUTH_OK still received (all four daemon types use the same ADR-012 handshake) |
| EC-003 | e2e test run with no `--key` arg (default key absent) | Test generates a key in-process; does not depend on `~/.ssh/id_ed25519` existing |
| EC-004 | Daemon startup fails (port conflict) | Test cleanup correctly closes all listeners; no resource leak |

## Purity Classification

| Module | Classification | Justification |
|--------|---------------|---------------|
| internal/testenv (harness) | effectful | Daemon startup, socket management, process lifecycle |

## Token Budget Estimate (MANDATORY)

| Context Source | Estimated Tokens |
|---------------|-----------------|
| This story spec | ~1,500 |
| BC-2.07.002.md (v1.2) | ~800 |
| ARCH-12 §Wiring into cmd/switchboard | ~800 |
| ARCH-05 §Daemon Management Socket | ~400 |
| VP-049.md (proof harness skeleton) | ~600 |
| internal/mgmt/mgmt.go (from S-W5.01) | ~2,500 |
| cmd/sbctl/client.go (from S-6.03) | ~1,000 |
| cmd/switchboard/main.go (from S-W5.01 wiring) | ~2,000 |
| Test files (estimated) | ~2,000 |
| Tool outputs overhead | ~400 |
| **Total** | **~12,000** |
| Agent context window | 200K |
| **Budget usage** | **~6.0%** |

## Tasks (MANDATORY)

1. [ ] Confirm S-6.03 and S-W5.01 are both merged to develop before starting
2. [ ] Read BC-2.07.002 (v1.2), ARCH-12 §Wiring, ARCH-05 §Daemon Management Socket, VP-049.md
3. [ ] Read `internal/mgmt/mgmt.go` (S-W5.01 implementation) to understand `NewServer` API
4. [ ] Read `cmd/sbctl/client.go` (S-6.03 implementation) to understand `Authenticate()` and `dispatch()` API
5. [ ] Read `cmd/switchboard/main.go` to understand daemon mode entry points after S-W5.01 wiring
6. [ ] Write failing tests for AC-001 through AC-005 (`//go:build integration` tag)
7. [ ] Verify Red Gate — tests must fail before any implementation
8. [ ] Implement test harness infrastructure (in-process daemon startup helper, socket path generation, ready-poll with timeout)
9. [ ] Implement `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049` covering AC-001 through AC-003, AC-005
10. [ ] Implement `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004`
11. [ ] Verify all tests pass: `go test -tags=integration ./...`
12. [ ] `just fmt && just lint` pass

## Previous Story Intelligence (MANDATORY)

| Story | Key Decisions | Patterns Established | Gotchas Discovered |
|-------|--------------|---------------------|-------------------|
| S-6.03 | `Authenticate()` is fail-closed; `--key` loads OpenSSH Ed25519 PEM | `client.Authenticate(conn, privKey) error` API | Use `net.Pipe` for in-process tests; avoids OS socket complexity |
| S-W5.01 | `mgmt.NewServer(ln, daemonKey, ops, handlers)` API; WaitGroup-tracked goroutine | `Server.Serve(ctx)` + `Server.Shutdown(ctx)` | Must call `Shutdown` in test cleanup to drain connections and avoid goroutine leaks |
| S-W3.04 | Daemon assembly pattern; all subsystems wired with `wg.Add(1)` | In-process daemon setup for tests | Socket paths must be unique per test run (use `t.TempDir()` for Unix socket paths) |

## Architecture Compliance Rules (MANDATORY)

| Rule | Source | Enforcement |
|------|--------|-------------|
| Test uses `//go:build integration` tag | ARCH-12 §Story 3 scope; VP-049.md harness | Build tag in file header |
| Test generates Ed25519 key in-process (no dependency on `~/.ssh/id_ed25519`) | EC-003 | Key generation in test setup |
| `Server.Shutdown(ctx)` called in `t.Cleanup` for each daemon | ARCH-01 §Goroutine WaitGroup Contract; no goroutine leak | `t.Cleanup(func() { srv.Shutdown(ctx) })` |
| Unix socket paths unique per test run | OS socket uniqueness | `t.TempDir()` for socket directory |
| Test does NOT assert on handler implementation details — only auth + envelope | BC-2.07.002 scope | AC-003 asserts `"ok":true` only; handler stub is sufficient |

## Library & Framework Requirements (MANDATORY)

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.4 | Per go.mod (mise-pinned) |
| `crypto/ed25519` | stdlib | In-process test key generation |
| `crypto/rand` | stdlib | Key generation |
| `net` | stdlib | `net.Pipe`, `net.Listen` for in-process socket setup |
| `context` | stdlib | Test context with timeout |
| `testing` | stdlib | `t.Cleanup`, `t.Parallel`, `t.Run` |
| `encoding/json` | stdlib | RPC message construction and parsing in test harness |

## File Structure Requirements (MANDATORY)

| File | Action | Purpose |
|------|--------|---------|
| `internal/mgmt/e2e_test.go` (or `internal/testenv/mgmt_e2e_test.go`) | create | `//go:build integration` e2e test: `TestE2E_MgmtPlane_AllFourDaemonTypes_VP049`, `TestE2E_MgmtPlane_UnauthenticatedRejected_AC004` |
| `internal/testenv/testenv.go` | create or modify | In-process daemon startup helpers for mgmt tests (if not already present from prior stories) |

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.1 | 2026-06-29 | product-owner | Add S-6.06 to `depends_on`. Adversary Pass 1 on S-6.02 found CRITICAL gap F-001: daemon-side admin RPC handlers (admin.key.register / revoke / expire / list-keys) were never wired; `startMgmtServer(..., nil)` at every call site. S-6.06 (minted per CR-W5-SCOPE-SPLIT ruling) closes that gap. S-W5.02 cannot exercise admin RPC paths end-to-end until S-6.06 is merged. |
| 1.0 | 2026-06-28 | story-writer | Initial creation — Wave-5 net-new story per ARCH-12 product-owner handoff. E2E integration gate for management plane; implements VP-049 across all four daemon types. Depends on S-6.03 + S-W5.01. |
