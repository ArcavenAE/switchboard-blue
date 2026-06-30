---
artifact_id: BC-2.05.004
document_type: behavioral-contract
level: L3
version: "1.7"
status: draft
producer: product-owner
timestamp: 2026-06-30T00:00:00
phase: 1a
bc_id: BC-2.05.004
subsystem: admission-security
architecture_module: internal/svtnmgmt
capability: CAP-019
priority: P0
criticality: critical
scope_phase: E
origin: greenfield
lifecycle_status: active
introduced: v0.1.0
modified:
  - date: 2026-06-29
    version: "1.2"
    actor: product-owner
    change: >
      Task 3 reconverge (S-6.02 lens3 F-011/F-012): Trigger updated from
      `sbctl svtn keys register|revoke|expire` to `sbctl admin key {register,revoke,expire}`
      or `sbctl admin list-keys`. Canonical Test Vectors rewritten to use `sbctl admin key`
      form throughout.
  - date: 2026-06-30
    version: "1.3"
    actor: product-owner
    change: >
      S-6.06 lens-3 F-002 close: Traceability Stories row extended to include PC-4
      (confirmation response with key fingerprint and operation timestamp):
      S-6.06 (mgmt.Response success envelope). VP-075 minted for handler-layer
      caller-role enforcement (DI-001 / PC-1 admission-control authority).
  - date: 2026-06-30
    version: "1.4"
    actor: architect
    change: >
      Pass-2 lens-3 process-gap observation (F-T3-004): DI-001 (carrier-grade content
      separation) back-cited. DI-001 confirmed present in domain-spec/invariants.md.
      Added to Invariants section (item 5) and Traceability L2 Domain Invariants row.
      DI-001 applies: key lifecycle operations authenticate the nodes that enforce
      content separation; a revoked or misconfigured key breaks DI-001 guarantees.
  - date: 2026-06-30
    version: "1.5"
    actor: product-owner
    change: >
      S-6.06 Pass-4 rulings (F-L2-001, F-L2-002, F-L2-003, F-L2-007, F-P4L1-001,
      F-P4L1-003): PC-1 extended with bootstrap exception (operator-set membership
      grants register authority when no control key exists in target SVTN) and
      fail-closed revoked/expired key denial (E-ADM-009). PC-3 clarified with wire
      field name (`after` Go duration string) vs. CLI `--at <RFC3339>` translation.
      Canonical Test Vectors: "console attempts revoke control key" row updated to
      E-ADM-009 (not E-ADM-011) with disambiguation note. list-keys authority scope
      clarified: any admitted role may call. EC-005 (bootstrap first-register) and
      EC-006 (revoked key residual authority) added.
  - date: 2026-06-30
    version: "1.6"
    actor: product-owner
    change: >
      S-6.06 Pass-6 finding F-P6L2-002: Canonical Test Vectors list-keys row
      annotated with wire RPC name (admin.key.list-keys) to match the annotation
      pattern used on other rows in the table.
  - date: 2026-06-30
    version: "1.7"
    actor: product-owner
    change: >
      F-P7L3-001: VP-075 module corrected from internal/mgmt to cmd/switchboard.
      BuildAdminHandlers (and its handler closures) live in cmd/switchboard/admin_handlers.go;
      the authority-gate test must instantiate the handler builder from that package.
      VP Anchors table updated accordingly.
deprecated: null
deprecated_by: null
replacement: null
retired: null
removed: null
removal_reason: null
inputDocuments:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/edge-cases.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
traces_to: [CAP-019]
kos_anchors:
  - elem-ssh-end-to-end-encryption
---

# Behavioral Contract BC-2.05.004: Key Lifecycle — Register, Revoke, and Expire Admission and Session-Authorization Keys

## Description

Control nodes can manage the public key registry for an SVTN: registering new keys (with role designation), revoking existing keys, and setting expiry dates. Per ADR-004, key management is exclusive to the control node; console and access nodes have no key-management capability. Key changes propagate via the router's distributed key store to all routers serving the SVTN. Registration allows a new node to join; revocation removes a node's SVTN membership; expiry sets an automatic future revocation.

## Preconditions

1. The requesting node is admitted to the SVTN with a key that has management authority for the operation. Per **ADR-004**: key management (register, revoke, expire) is exclusive to the control node. Console nodes may call `list-keys` (read-only); they cannot register, revoke, or expire keys. Access nodes have no key-management capability. Console-to-control revocation is prohibited. Control-to-control revocation requires `sbctl admin` human authorization (split-brain mitigation).

   **Management authority definition (for register/revoke/expire):** The calling key MUST be (a) currently registered in the target SVTN's admitted set, (b) with `RoleControl`, (c) not revoked (`revoked=false`), and (d) not expired (`now < expiry` or no expiry set). A revoked or expired key is denied with E-ADM-009, even if it authenticated successfully at the mgmt connection layer.

   **Bootstrap exception (F-P4L1-001):** For `admin.key.register` only — if the calling key is a member of `mgmt.OperatorKeySet` AND no control key is yet registered in the target SVTN, the handler MUST allow the operation (operator-set membership grants register authority for the initial bootstrap). For all other operations (revoke, expire), operator-set membership alone is insufficient; the key must be registered as `RoleControl` in the SVTN.

   **list-keys authority (F-L2-003):** `admin.key.list-keys` is a read-only operation and may be called by any admitted role (control, console, access) or operator-set member. It is NOT subject to the control-only authority gate.
2. The key operation is well-formed: a valid OpenSSH public key in authorized_keys format.
3. The router's distributed key store is reachable.

## Postconditions

1. **Register**: The public key is added to the admitted key set with the specified role (control, console, access). The key becomes active for admission challenges. Propagation: key is pushed to all routers serving the SVTN.
2. **Revoke**: The public key is removed from the admitted key set. Existing sessions using this key continue until the next re-authentication challenge (propagation delay acknowledged, FM-007). Propagation: revocation is pushed to all routers.
3. **Expire**: An expiry timestamp is associated with the key. At expiry, the key is automatically revoked by routers that honor the expiry timestamp. **Wire field (F-L2-007):** the `admin.key.expire` RPC args field is `after` (a Go duration string, e.g., `"720h"`, `"30m"`). The CLI `--at <RFC3339-timestamp>` flag is translated client-side to a duration (`timestamp - time.Now()`) before sending on the wire. The daemon handler validates `after` as a positive Go duration string; zero, negative, and >100-year values are rejected with E-CFG-001.
4. Key changes are confirmed with a success response including the key fingerprint and operation timestamp.

## Invariants

1. **DI-011**: Revoking a Tier 1 key removes the node from the network; revoking a Tier 2 key removes access to a specific session. These are independent operations.
2. **DI-002**: Key registration and revocation operations use public keys only; private keys are never transmitted.
3. Key management operations are authenticated: the requesting node's signature is verified before any change is applied.
4. **DI-012**: The control node manages keys as a network participant; it does not have privileged router API access.
5. **DI-001**: Key lifecycle operations uphold carrier-grade content separation — the keys managed by this BC authenticate only the transport/admission layer; no key managed here grants any router the ability to read, modify, or inject session payload content.

## Trigger

Operator runs `sbctl admin key {register,revoke,expire}` or `sbctl admin list-keys` or equivalent management RPC.

## Edge Cases

| ID | Description | Expected Behavior |
|----|-------------|-------------------|
| EC-001 (DEC-005) | Key revoked; node has active session | Node's session continues until next re-authentication challenge. Revocation propagates asynchronously. |
| EC-002 (FM-007) | Key revocation propagation is slow (one router not updated) | The un-updated router may still admit the revoked key. Propagation completes within the eventual consistency window. |
| EC-003 (DEC-007) | Same public key registered twice with different roles | Per **ADR-003** (last-write-wins for duplicate key registration): the most recent registration supersedes earlier registrations for the same `(node_pubkey, svtn_id)` pair. The operation returns "updated" with the new role; no conflict; no manual reconciliation required. |
| EC-004 | Key expires while session is active | Same behavior as revocation: session continues until next re-authentication challenge. |
| EC-005 | Operator-key first-register into fresh SVTN (bootstrap path, F-P4L1-001) | No control key is registered in the target SVTN yet. The calling key is a member of `mgmt.OperatorKeySet`. `admin.key.register` MUST proceed (bootstrap grant). Subsequent register/revoke/expire operations require the caller to be registered as RoleControl in the SVTN admitted set. |
| EC-006 | Revoked or expired key attempts an admin.key.* operation after successful mgmt authentication (F-P4L1-003) | The key authenticated at the mgmt connection layer but its `revoked=true` or `now >= expiry` in the SVTN admitted set. Handler authority resolution treats the key as unregistered; returns E-ADM-009. This applies to register, revoke, and expire (not list-keys, which is open to all admitted roles). |

## Canonical Test Vectors

| Input | Expected Output | Category |
|-------|----------------|----------|
| `sbctl admin key register --svtn <id> --key <hex-pubkey> --role console` | Key registered; fingerprint returned; propagation initiated | happy-path |
| `sbctl admin key revoke --svtn <id> --key <hex-pubkey>` | Key revoked; active sessions continue until re-auth; propagation initiated | happy-path |
| `sbctl admin key expire --svtn <id> --key <hex-pubkey> --at <timestamp>` | CLI translates `--at <RFC3339>` → `after` duration string; expiry timestamp associated; auto-revocation scheduled | happy-path |
| `sbctl admin list-keys --svtn <id>` (CLI: `admin list-keys`; wire: `admin.key.list-keys`) | Returns all admitted keys with role, fingerprint, expiry | happy-path |
| `sbctl admin key register --svtn <id> --key <same-pubkey-already-registered> --role access` | Response: "updated" with new role (per ADR-003: last-write-wins) | edge-case |
| Key operation by node without management authority | E-ADM-009 "insufficient authority for operation admin.key.register: key <fp> has role <role>" | error |
| Console or readonly key attempts to revoke a control key via `admin.key.revoke` RPC | E-ADM-009 "insufficient authority for operation admin.key.revoke: key <fp> has role console" — handler gate fires first; `SVTNManager.RevokeKey` is never called (F-L2-002). Note: E-ADM-011 is the code returned by `SVTNManager.RevokeKey` directly (Go API level, unit-test path) when a lower-tier role attempts to revoke a higher-tier key; it is NOT reachable via the mgmt RPC path when the handler gate is wired. | error |

## Verification Properties

| VP-NNN | Property | Proof Method |
|--------|----------|-------------|
| VP-046 | Key registration makes key available for admission on all propagated routers | integration |
| VP-046 | Revocation propagates to all routers within eventual consistency window | integration |
| VP-046 | Private key never appears in key management wire messages | property |
| VP-075 | Handler-layer caller-role enforcement: admin.key.* RPCs reject callers without control-role authority (cmd/switchboard) | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation), DI-002 (private keys never transit), DI-011 (role separation between Tier 1 and Tier 2 keys), DI-012 (control node is a participant) |
| Architecture Module | internal/svtnmgmt |
| Stories | PC-1 (register): S-6.02 (CLI dispatch), S-6.06 (daemon handler); PC-2 (revoke): S-6.02 (CLI dispatch), S-6.06 (daemon handler); PC-3 (expire): S-6.02 (CLI dispatch), S-6.06 (daemon handler); PC-4 (confirmation response with key fingerprint and operation timestamp): S-6.06 (mgmt.Response success envelope) |
| Capability Anchor Justification | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 — this BC specifies the complete key lifecycle operations that CAP-019 defines as the revocation path |

## Related BCs

- BC-2.05.001 — depends on: registered keys are what the admission challenge verifies
- BC-2.05.002 — related to: revocation eventually removes key from admitted set
