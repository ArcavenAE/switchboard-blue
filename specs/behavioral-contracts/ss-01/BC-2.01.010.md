---
artifact_id: BC-2.01.010
document_type: behavioral-contract
level: L3
version: "1.0"
status: draft
producer: product-owner
timestamp: 2026-07-18T00:00:00Z
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
input-hash: "929cfd4"
extracted_from: null
bc_id: BC-2.01.010
subsystem: session-networking
architecture_module: internal/routing
capability: CAP-003
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - version: "1.0"
    date: 2026-07-18
    author: product-owner
    change: >
      Initial commission — BindInterface binding lifecycle.
      Authored per S-BL.NODE-IDENTIFY-WIRE-rulings.md §8, §12
      (S-BL.NODE-IDENTIFY-WIRE-rulings.md v1.1, 2026-07-18).
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
  - '_bmad-output/planning-artifacts/prd.md'
  - 'decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md'
  - 'decisions/identity-cluster-architecture.md'
traces_to: [CAP-003]
kos_anchors:
  - elem-node-router-architecture
---

# Behavioral Contract BC-2.01.010: BindInterface Binding Lifecycle — (SVTNID, NodeAddr) → IfaceID

## Description

After a successful NODE_IDENTIFY handshake (BC-2.01.009), the router records a `(SVTNID, NodeAddr) → IfaceID` binding in the `Router.identityIfaceMap`. This binding enables the DISCOVERY_RELAY fan-out path (BC-2.03.001, S-BL.DISCOVERY-WIRE Task 6) to resolve a node's cryptographic address to the `sendMap` key for its live connection. Bindings are created via `Router.BindInterface`, looked up via `Router.LookupInterface`, and removed via `Router.UnbindInterface`. All three methods are protected by the existing `r.mu` `sync.RWMutex`.

`BindInterface`, `LookupInterface`, and `UnbindInterface` are to-be-built by S-BL.NODE-IDENTIFY-WIRE — they do not yet exist in `internal/routing`. They require no new imports and no ARCH-08 DAG position changes.

## Preconditions

1. `Router.BindInterface(svtnID [16]byte, nodeAddr [8]byte, ifaceID InterfaceID)` is called from the `onAccept` closure in `runRouter`, immediately after `admission.AdmitNode` returns `nil`.
2. `ifaceID` is the `InterfaceID` of the accepted connection (from the `netingress.ConnHandle`).
3. `Router.identityIfaceMap` is initialized as `map[[16]byte]map[[8]byte]InterfaceID` (backed by `r.mu`, same mutex as `forwardingTable`).

## Postconditions

### BindInterface

1. **Binding created:** `Router.identityIfaceMap[svtnID][nodeAddr] = ifaceID`. A nested map for `svtnID` is allocated if absent.

2. **Last-writer-wins (LWW) on reconnect:** If a binding for `(svtnID, nodeAddr)` already exists (from a prior TCP connection that has not yet closed), the new `ifaceID` overwrites it. The prior TCP connection is NOT actively torn down. It self-removes when it eventually closes, via its cleanup func calling `UnbindInterface` (Postcondition 5). This LWW overwrite is consistent with ADR-003 and the existing `forwardingTable` mutation semantics.

3. **Security: rebind requires full re-handshake:** A LWW overwrite is only possible after `AdmitNode` returns `nil`, which requires the connecting node to prove possession of the registered private key. A different public key that does not match the registered key for this `(svtnID, nodeAddr)` will fail `AdmitNode` with `ErrNotAdmitted` before reaching `BindInterface`. Binding hijack by a different identity is cryptographically prevented.

4. **Write lock held:** `BindInterface` acquires `r.mu` write lock. No concurrent read or write to `identityIfaceMap` is possible while the binding is written.

### LookupInterface

4. **Lookup returns binding if present:** `Router.LookupInterface(svtnID, nodeAddr)` returns `(ifaceID, true)` if a binding exists for `(svtnID, nodeAddr)`. Returns `(0, false)` if no binding exists. Callers must test the `bool` flag before using the `InterfaceID`.

5. **Read lock held:** `LookupInterface` acquires `r.mu` read lock. Concurrent reads are permitted; no mutation occurs during lookup.

6. **Return type is value, not pointer:** Per `go.md` rule 12 (return value copies from locked accessors), `LookupInterface` returns `(InterfaceID, bool)` — a value type, not a pointer into internal state.

### UnbindInterface

7. **Binding removed on connection close:** The `onAccept` cleanup func (the `func()` returned by `onAccept` to `netingress.Serve`) MUST call `Router.UnbindInterface(svtnID, nodeAddr)` in addition to `sendMap.Delete(h.IfaceID)`. This is the only teardown required — no additional connection-lifecycle plumbing is needed.

8. **Stale cleanup guard:** If `UnbindInterface` is called for a `(svtnID, nodeAddr)` pair whose current binding maps to a DIFFERENT `ifaceID` (i.e., a LWW overwrite occurred and the prior connection's cleanup func fires after the new binding was installed), `UnbindInterface` MUST NOT remove the new binding. Implementation: check `identityIfaceMap[svtnID][nodeAddr] == myIfaceID` under write lock before deleting. Only delete if the stored `ifaceID` matches the caller's own `ifaceID`.

9. **Write lock held:** `UnbindInterface` acquires `r.mu` write lock.

## Invariants

1. **`r.mu` governs all three methods:** `BindInterface` and `UnbindInterface` hold the write lock; `LookupInterface` holds the read lock. This is the same discipline as `RegisterForwardingEntry` and the existing `forwardingTable` mutations. No additional synchronization primitive is introduced.

2. **`identityIfaceMap` lifetime is co-terminus with `Router`:** The map is initialized with the `Router` and cleared only on `Router` teardown. Individual entries are created by `BindInterface` and removed by `UnbindInterface`.

3. **A different pubkey cannot hijack an existing binding:** The LWW overwrite in PC-2 is only reachable after `AdmitNode` succeeds. `AdmitNode` verifies the signature against the registered public key for this `(svtnID, nodeAddr)`. A node presenting a different keypair cannot reach `BindInterface` for an address it does not own.

4. **Second `NodeIdentify` on the same connection never calls `BindInterface`:** The `onAccept` closure tracks whether the handshake has already completed. A second `NodeIdentify` frame on an established connection triggers E-ADM-023 and closes the connection (BC-2.01.009 Invariant 7). `BindInterface` is only callable once per connection.

5. **Prior connection's `sendMap` entry is NOT removed by a LWW overwrite:** `BindInterface` overwrites the `identityIfaceMap` entry but does NOT call `sendMap.Delete` for the prior `ifaceID`. The stale connection's `sendMap` entry remains live and self-removes when the prior connection's cleanup func fires. DISCOVERY_RELAY fan-out resolves to the new binding via `LookupInterface` (the overwritten entry); direct-`ifaceID` sends to the stale entry continue until TCP keepalive detects the dead connection.

## Trigger

- `BindInterface`: `onAccept` closure, after `admission.AdmitNode` returns `nil` (BC-2.01.009 Postcondition 6).
- `LookupInterface`: DISCOVERY_RELAY fan-out closure (S-BL.DISCOVERY-WIRE Task 6 / AC-017, AC-018) to resolve `NodeAddr → IfaceID` for the `sendMap` lookup.
- `UnbindInterface`: per-connection cleanup func (the `func()` returned by `onAccept`), when the connection closes.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 | LWW overwrite: node reconnects (new TCP) before prior connection closes | `BindInterface` overwrites with new `ifaceID`. Prior connection's cleanup func fires `UnbindInterface` with old `ifaceID`, which detects `identityIfaceMap[svtnID][nodeAddr] != oldIfaceID` (stale cleanup guard) and does NOT remove the new binding. |
| EC-002 | `UnbindInterface` called for a binding that was already overwritten by LWW | Stale cleanup guard fires (PC-8): stored `ifaceID` does not match caller's `ifaceID`; delete skipped. No error. |
| EC-003 | `LookupInterface` called for a `(svtnID, nodeAddr)` with no binding (node not yet admitted or already unbound) | Returns `(0, false)`. Caller MUST check the `bool` flag; a zero `InterfaceID` is not a valid send-map key. |
| EC-004 | SVTN removed (`RemoveSVTN` / `admin.svtn.destroy`) while bindings exist for that SVTN | Out of scope for this BC — `UnbindInterface` is the cleanup mechanism for per-connection teardown. SVTN-wide cleanup on destroy is a separate concern (tracked by S-BL.SVTN-DESTROY or equivalent). |
| EC-005 | `BindInterface` called for a node whose prior binding was removed by `UnbindInterface` (clean reconnect) | Normal case: nested map entry for `nodeAddr` was deleted by `UnbindInterface`; `BindInterface` re-inserts it. Behavior is identical to the first-bind case. |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `BindInterface(svtnID, nodeAddr, ifaceID=1)` on fresh `Router` | `LookupInterface(svtnID, nodeAddr)` returns `(1, true)` | happy-path |
| `BindInterface(svtnID, nodeAddr, ifaceID=2)` on already-bound entry (`ifaceID=1`) | `LookupInterface(svtnID, nodeAddr)` returns `(2, true)` (LWW overwrite) | rebind |
| `UnbindInterface(svtnID, nodeAddr)` with matching `ifaceID=2` | `LookupInterface(svtnID, nodeAddr)` returns `(0, false)` | cleanup |
| Stale `UnbindInterface(svtnID, nodeAddr)` with old `ifaceID=1` after LWW overwrite to `ifaceID=2` | `LookupInterface(svtnID, nodeAddr)` still returns `(2, true)` (stale cleanup guard fired) | stale-cleanup |
| `LookupInterface` for unbound `(svtnID, nodeAddr)` | Returns `(0, false)` | edge |
| Concurrent `BindInterface` and `LookupInterface` | No data race; reads and writes are mutex-protected | concurrency |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| test-as-evidence | `BindInterface` followed by `LookupInterface` returns the bound `ifaceID` | unit |
| test-as-evidence | LWW overwrite: second `BindInterface` replaces first; `LookupInterface` returns new `ifaceID` | unit |
| test-as-evidence | `UnbindInterface` removes binding; subsequent `LookupInterface` returns `(0, false)` | unit |
| test-as-evidence | Stale cleanup guard: `UnbindInterface` with old `ifaceID` after LWW overwrite does NOT remove new binding | unit |
| test-as-evidence | No data race under concurrent `BindInterface` + `LookupInterface` | `go test -race` |
| test-as-evidence | A different pubkey cannot reach `BindInterface` for an address it does not own — `AdmitNode` gate | integration (via BC-2.01.009 happy-path + wrong-key test vectors) |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation — binding map is a routing-plane construct), DI-002 (private keys never transit — `BindInterface` only records public identity derived from the admitted pubkey) |
| Architecture Module | internal/routing (`Router.identityIfaceMap`; `BindInterface`, `LookupInterface`, `UnbindInterface` methods) |
| Stories | S-BL.NODE-IDENTIFY-WIRE |
| Capability Anchor Justification | CAP-003 ("Frame envelope encoding and decoding") per capabilities.md §CAP-003 — the `(SVTNID, NodeAddr) → IfaceID` binding is the routing-plane data structure that maps the cryptographic node address (derived from the wire frame's outer header per BC-2.01.006) to the `sendMap` fan-out key |

## Related BCs

- BC-2.01.009 — invokes: `BindInterface` is called at BC-2.01.009 Postcondition 6 (on `AdmitNode` success); the cleanup func calls `UnbindInterface`
- BC-2.01.008 — context: `NODE_IDENTIFY = 0x04` opcode (the handshake that triggers this BC) is registered in BC-2.01.008 PC-2
- BC-2.01.006 — depends on: `NodeAddr` is derived from `(svtnID, pubkey)` via `frame.DeriveNodeAddress`; this BC's binding map uses `NodeAddr` as the inner key
- BC-2.03.001 — consumed by: DISCOVERY_RELAY fan-out (S-BL.DISCOVERY-WIRE Task 6) calls `LookupInterface` to resolve `NodeAddr → IfaceID`; this BC is the write side of that lookup

## Architecture Anchors

- decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md §8 (BindInterface/LookupInterface/UnbindInterface method signatures, `identityIfaceMap` field, concurrency contract)
- decisions/S-BL.NODE-IDENTIFY-WIRE-rulings.md §12 (Obligation 3 — LWW overwrite on reconnect; prior connection NOT torn down; stale cleanup semantics)
- decisions/identity-cluster-architecture.md Section 4 (motivation for the identity binding map — DISCOVERY_RELAY fan-out unblocking)

## Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-07-18 | Initial commission — `(SVTNID, NodeAddr) → IfaceID` binding lifecycle: `BindInterface` (LWW on reconnect), `LookupInterface` (read-lock value return), `UnbindInterface` (stale-cleanup guard). Sourced from S-BL.NODE-IDENTIFY-WIRE-rulings.md §8 and §12. |
