---
artifact_id: BC-2.07.002
document_type: behavioral-contract
level: L3
version: "1.7"
status: draft
producer: product-owner
timestamp: 2026-06-28T00:00:00
phase: 1a
bc_id: BC-2.07.002
subsystem: network-management
architecture_module: cmd/sbctl
capability: CAP-024
priority: P2
criticality: high
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-06-28
    version: "1.2"
    change: >
      Wave-5 management plane cross-reference: added BC-2.07.004 to Related BCs
      as the server-side counterpart that completes the ADR-012 handshake. Updated
      Verification Properties table to add VP-067 (Authenticate() fail-closed, unit).
      PC-2 and PC-3 now have explicit server-side anchoring via BC-2.07.004;
      VP-049 e2e property anchors both client and server behaviors end-to-end.
      Story anchor updated: S-6.03 implements client-side (this BC + VP-067 + VP-030);
      S-W5.01 implements server-side (BC-2.07.004); S-W5.02 implements e2e (VP-049).
  - date: 2026-06-29
    version: "1.3"
    change: >
      ARCH-12 v1.4 Wave-5 Convergence Ruling M: PC-3 annotated with RPC wire-type
      precision — authenticated RPC request envelope MUST carry "type":"request"
      (ADR-012 §3 step 6); server response carries "type":"response"; any other type
      string causes the server to close the connection silently. Pins the client
      wire contract that was previously unconstrained.
  - date: 2026-06-29
    version: "1.4"
    change: >
      ARCH-12 v1.5 Wave-5 Convergence Rulings U and X: (U) PC-3 extended with
      receiving-side wire-type validation: dispatch() MUST verify resp.Type ==
      "response" after decoding; any other value (e.g., "rpc_response", "auth_fail",
      "") is treated as a protocol error and returned as E-RPC-001; a wrong-type
      response with "ok":true must NOT be silently accepted. (X) PC-3 extended with
      non-constant request ID and response echo check: dispatch() generates a
      per-call non-constant request id (not always "1"); after decoding the response,
      dispatch() verifies resp.ID echoes req.ID; mismatch treated as protocol error
      (E-RPC-001); per ADR-012 §3 step 6.
  - date: 2026-06-30
    version: "1.5"
    change: >
      Delete BC §VP table phantom rows 139-140 (both mis-labeled VP-049; per S-W5.02
      Pass-1 L3 F-P1L3-001 PO Q3 ruling). Follow-up minting VP-JSON-COVERAGE +
      VP-SBCTL-NOT-DAEMON tracked as Wave-6 backlog.
  - date: 2026-07-02
    version: "1.7"
    change: >
      Annotate EC-004 with PENDING-S-BL.PING-VERSION-WIRE for version wire handler gap;
      add EC-005 for ping wire handler gap (also PENDING-S-BL.PING-VERSION-WIRE). Closes
      Phase 5 Pass 2 F-P5P2-A-001, F-P5P2-A-002. Notes DRIFT-P5P2-A003: e2e_helpers_test.go:191
      uses stale wire name `admin.key.list` for a mock handler where the shipped surface
      is `admin.key.list-keys` — deferred to test-writer follow-up.
  - date: 2026-07-02
    version: "1.6"
    change: >
      Add PENDING-S-BL.SVTN-LIST-WIRE annotation to Canonical Test Vectors — happy-path
      row unreachable through shipped operator surface as of develop@7fe3e29e; wire
      handler tracked in backlog story S-BL.SVTN-LIST-WIRE. Closes
      DRIFT-P5P1-A001-SVTN-LIST-ORPHAN. Refs Phase 5 Pass 1 Adv-A F-P5P1-A-001.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-024]
kos_anchors:
  - elem-single-binary-three-modes
---

# Behavioral Contract BC-2.07.002: sbctl Unified CLI for All Four Daemon Types with OpenSSH Key Authentication

## Description

`sbctl` is the single operator CLI for all Switchboard daemon types: router (E and PE), access node, console, and control node. It authenticates the operator via their OpenSSH key (same key infrastructure used for SVTN admission). Subcommands are scoped by daemon type and operation category. No separate management tools are required.

## Preconditions

1. The operator has an OpenSSH key registered against the target daemon's SVTN or management scope.
2. The target daemon is running and listening on its management port.
3. sbctl can reach the daemon (network connectivity).

## Postconditions

1. sbctl connects to the target daemon.
2. sbctl authenticates the operator's OpenSSH key against the daemon's authorized key list.
3. If authenticated: the requested operation is executed; result returned in the configured output format (human-readable or JSON).
   **Protocol-precision note (Ruling M / ADR-012 §3 step 6):** The authenticated RPC
   request envelope sent by `dispatch()` in `cmd/sbctl/client.go` MUST carry
   `"type":"request"`. The server response envelope carries `"type":"response"`. Any
   other type string after authentication causes the server to close the connection
   silently (the server reads the type field and closes on mismatch; the client then
   receives EOF and returns E-RPC-001). `"rpc_request"` and `"rpc_response"` are NOT
   valid wire types and MUST NOT appear in any implementation.
   **Receiving-side wire-type validation (Ruling U):** After decoding the RPC response,
   `dispatch()` MUST verify `resp.Type == "response"`. Any other value (e.g.,
   `"rpc_response"`, `"auth_fail"`, `""`) is treated as a protocol error and returned
   as E-RPC-001 with a message containing the unexpected type. A wrong-type response
   with `"ok":true` MUST NOT be silently accepted as a successful RPC — the type check
   takes precedence over the ok-flag check.
   **Non-constant request ID and echo check (Ruling X / ADR-012 §3 step 6):** The `id`
   field in the RPC request envelope is a client-generated non-constant value (not
   always `"1"`; e.g., a hex encoding of `time.Now().UnixNano()`). After decoding the
   response, `dispatch()` verifies that `resp.ID` echoes `req.ID`. A mismatch is
   treated as a protocol error (E-RPC-001). For a single-RPC-per-connection client,
   the practical impact of ID mismatch is low, but the spec-mandated echo check is
   explicitly required per ADR-012 §3 step 6.
4. If not authenticated: E-ADM-010 "authentication failed"; exit code 1.
5. If daemon unreachable: E-NET-001 (per BC-2.07.003); exit code 1.

## Invariants

1. All sbctl subcommands are authenticated — there is no unauthenticated sbctl endpoint.
2. The sbctl binary is not a daemon; it exits after command completion.
3. Output format is consistent: `--json` for machine-readable output in all subcommands.

## Trigger

Operator runs any `sbctl <subcommand>` command.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | Operator's key not authorized for the requested operation | E-ADM-010 "authentication failed" OR E-ADM-009 "insufficient authority for operation" depending on whether the key is recognized at all. |
| EC-002 | Multiple daemon types running on the same machine | sbctl targets by address and port; `--target=<addr>` or config file specifies which daemon. |
| EC-003 | sbctl run without any subcommand | Prints help text and exits 0. |
| EC-004 | sbctl version mismatch with daemon | Daemon returns version info; sbctl prints warning if version differs; command may still succeed if protocol is compatible. **PENDING-S-BL.PING-VERSION-WIRE:** as of develop@7fe3e29e, wire command `version` has no daemon-side handler — `sbctl version` returns `E-RPC-010: unknown command: version`. Tracked in backlog story S-BL.PING-VERSION-WIRE. |
| EC-005 | sbctl ping smoke-test | Wire command `ping` has no daemon-side handler as of develop@7fe3e29e; returns `E-RPC-010: unknown command: ping`. **PENDING-S-BL.PING-VERSION-WIRE:** tracked in backlog story S-BL.PING-VERSION-WIRE. Product-owner decision at delivery: implement trivial handler returning `{"pong":true}`, or remove the sbctl case-arm. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl svtn list` with registered key | List of SVTNs returned | happy-path |
| `sbctl svtn list` with unregistered key | E-ADM-010 "authentication failed"; exit 1 | error |
| `sbctl --help` | Help text printed; exit 0 | happy-path |
| `sbctl router status --json` | JSON object with router status fields | happy-path |

> **PENDING-S-BL.SVTN-LIST-WIRE:** As of develop@7fe3e29e, the canonical `sbctl svtn list` happy-path test vector is not directly executable — the sbctl subcommand dispatches to wire command `svtn.list`, for which no daemon (control / router / access / console) currently registers a handler. Invocation returns `E-RPC-010: unknown command: svtn.list` regardless of authentication state, so both test vectors (happy and error) reach the wire boundary but the happy-path row's postcondition ("List of SVTNs returned") is unreachable through the shipped operator surface. The `admin.svtn.list` handler (see also Observation: no admin-scoped listing surface today — `cmd/switchboard/admin_handlers.go` registers `admin.svtn.create` and `admin.svtn.destroy` but not `admin.svtn.list`) is expected to land via backlog story `S-BL.SVTN-LIST-WIRE`. Until then, this test vector documents intended behavior, not shipped behavior.

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-049 | All subcommands require authentication — e2e across all four daemon types (router, access, console, control) | e2e |
| VP-067 | `Authenticate()` is fail-closed — returns nil only on verified AUTH_OK; all other outcomes (AUTH_FAIL, truncated stream, malformed message, connection error) return non-nil error | unit |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 |
| L2 Domain Invariants | DI-002 (private keys never transit — sbctl uses key-based auth without transmitting the private key) |
| Architecture Module | cmd/sbctl |
| Stories | S-6.03 (client auth, Authenticate() fail-closed, connection error); S-W5.02 (e2e VP-049 across all four daemon types) |
| Capability Anchor Justification | CAP-024 ("Unified CLI operator interface (sbctl)") per capabilities.md §CAP-024 — this BC specifies the unified CLI contract that CAP-024 defines as "single operator CLI for all four daemon types" with "OpenSSH key" authentication |

## Related BCs

- BC-2.07.003 — composes with: connection error handling is common to all sbctl operations
- BC-2.07.004 — composes with: server-side daemon auth counterpart (ADR-012); PC-2 (key auth) and PC-3 (execute if authenticated) of this BC require BC-2.07.004's server-side handshake enforcement to be meaningful end-to-end. VP-049 (e2e across all four daemon types) jointly verifies both BCs.

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.7 | 2026-07-02 | Annotate EC-004 with PENDING-S-BL.PING-VERSION-WIRE for version wire handler gap; add EC-005 for ping wire handler gap (also PENDING-S-BL.PING-VERSION-WIRE). Closes Phase 5 Pass 2 F-P5P2-A-001, F-P5P2-A-002. Also notes DRIFT-P5P2-A003: e2e_helpers_test.go:191 uses stale wire name `admin.key.list` for a mock handler where the shipped surface is `admin.key.list-keys` — deferred to test-writer follow-up. |
| 1.6 | 2026-07-02 | Add PENDING-S-BL.SVTN-LIST-WIRE annotation to Canonical Test Vectors — happy-path row unreachable through shipped operator surface as of develop@7fe3e29e; wire handler tracked in backlog story S-BL.SVTN-LIST-WIRE. Closes DRIFT-P5P1-A001-SVTN-LIST-ORPHAN. Refs Phase 5 Pass 1 Adv-A F-P5P1-A-001. |
| 1.5 | 2026-06-30 | Delete §VP table phantom rows 139-140 (both mis-labeled VP-049: `--json` fuzz row and `not-a-daemon` unit row). Per S-W5.02 Pass-1 L3 F-P1L3-001 PO Q3 ruling — these rows were never formally minted VP IDs; they are copy-paste leftovers predating the VP-062/VP-063 splits. Follow-up VPs VP-JSON-COVERAGE and VP-SBCTL-NOT-DAEMON tracked as Wave-6 backlog. |
| 1.4 | 2026-06-29 | ARCH-12 v1.5 Wave-5 Convergence Rulings U and X: (U) PC-3 extended — `dispatch()` MUST verify `resp.Type == "response"` after decoding; any other value is protocol error returned as E-RPC-001; wrong-type response with `"ok":true` must not be silently accepted. (X) PC-3 extended — `dispatch()` generates a non-constant per-call request `id`; after decoding, verifies `resp.ID == req.ID`; mismatch = E-RPC-001; per ADR-012 §3 step 6. |
| 1.3 | 2026-06-29 | ARCH-12 v1.4 Wave-5 Convergence Ruling M: PC-3 annotated with RPC wire-type precision note — `"type":"request"` is the only valid client RPC envelope type per ADR-012 §3 step 6; server responds with `"type":"response"`; any other type causes silent connection close. Frontmatter version/modified updated. |
| 1.2 | 2026-06-28 | Wave-5 management plane cross-reference: added BC-2.07.004 (server-side counterpart) to Related BCs. Verification Properties table extended with VP-067 (Authenticate() fail-closed, unit). Traceability Stories row updated: S-6.03 (client auth + VP-067 + VP-030), S-W5.02 (e2e VP-049 across all four daemon types). VP-049 description clarified to reflect e2e scope across all four daemon types (implementing story: S-W5.02). |
| 1.1 | 2026-06-23 | Initial published draft — sbctl unified CLI with OpenSSH key auth. |
