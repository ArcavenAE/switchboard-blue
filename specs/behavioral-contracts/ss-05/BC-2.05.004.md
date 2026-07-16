---
artifact_id: BC-2.05.004
document_type: behavioral-contract
level: L3
version: "1.15"
status: draft
producer: product-owner
timestamp: 2026-06-30T00:00:01
last_modified: 2026-07-15
phase: 1a
inputs:
  - '.factory/specs/domain-spec/capabilities.md'
  - '.factory/specs/domain-spec/invariants.md'
  - '.factory/specs/domain-spec/edge-cases.md'
  - '.factory/specs/domain-spec/failure-modes.md'
  - '_bmad-output/planning-artifacts/prd.md'
input-hash: "b9ec04e"
extracted_from: null
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
  - date: 2026-06-30
    version: "1.8"
    actor: product-owner
    change: >
      F-P14L2-002 (HIGH): EC-007 added — bootstrap-key-revoke-forbidden scenario
      (operator attempts to revoke the last bootstrap key in a SVTN with no
      non-bootstrap control key registered). ErrBootstrapKeyRevokeForbidden maps to
      E-ADM-020. Closes dangling error-taxonomy.md anchor at line 84 which cited
      BC-2.05.004 EC-007 before EC-007 existed. Note: EC-004 gap (EC-003 → EC-005
      in v1.5) confirmed intentional — EC-004 was added during the same v1.5 pass
      covering key-expires-while-session-active; no orphan.
  - date: 2026-06-30
    version: "1.9"
    actor: product-owner
    change: >
      EC-007 narrative tightened: bootstrap key is unconditionally non-revocable
      (refs F-P15L1-002, lens-1 finding pass-15).
  - date: 2026-06-30
    version: "1.10"
    actor: product-owner
    change: >
      EC-007 extended to cover expire symmetrically; E-ADM-021 minted
      (refs F-P18L1-001 lens-1 pass-18). EC-007 Description broadened from
      "revoke" to "revoke OR expire". EC-007 Expected Behavior updated to cite
      both ErrBootstrapKeyRevokeForbidden (E-ADM-020) and ErrBootstrapKeyExpireForbidden
      (E-ADM-021). Protection invariant is now symmetric across revoke and expire.
  - date: 2026-06-30
    version: "1.11"
    actor: product-owner
    change: >
      Pass-19 sibling-fix propagation (F-P19L*-001, F-P19L3-002, F-P19L3-003):
      VP-076 added to Verification Properties table (bootstrap key non-revocable AND
      non-expirable invariant, symmetric revoke + expire forbidden sentinels
      E-ADM-020/E-ADM-021). Traceability Stories row updated to note S-6.06 covers
      EC-007 (bootstrap non-revocable AND non-expirable). modified: list reordered
      to strict monotonic chronological order (v1.7–v1.9 were out of sequence
      relative to v1.10 insertion).
  - date: 2026-06-30
    version: "1.12"
    actor: product-owner
    change: >
      Pass-20 lens-3 F-P20L3-001 (MEDIUM) ruling — Option B: EC-007 narrowed to
      remove over-broad "unconditionally" claim. The bootstrap guard fires for any
      well-formed revoke/expire request targeting the bootstrap key; malformed
      requests (invalid duration, missing fields) are rejected by the handler's
      input-validation layer with E-CFG-001 before the bootstrap sentinel is
      consulted. The mutation-prevention invariant is unaffected: SVTNManager is
      never called for either bootstrap+well-formed or any+malformed request, so
      the key store cannot be mutated in either case. EC-007 Expected Behavior and
      Tests citation updated accordingly. VP-076 property #3 narrowed in parallel.
  - date: 2026-07-03
    version: "1.14"
    actor: spec-steward
    change: >
      F-P5P14-B-003 traceability fix: EC-008 (three admission-failure modes for
      admin.key.list-keys) now references owning VP-077. Property text: list-keys
      admits iff IsAdmittedAnyRole OR OperatorKeySet OR BootstrapKey; else E-ADM-009.
      Complements VP-075 which scope-excludes list-keys. Verification Properties table
      extended with VP-077 row. No behavioral change.
  - date: 2026-07-03
    version: "1.13"
    actor: product-owner
    change: >
      Phase 5 Pass 13 spec-track sharpening (Burst 38, post-merge PR #69 03ce8e7).
      F-P5P13-A-001 [HIGH] admission gate distinction: Precondition 1 list-keys
      authority sentence (F-L2-003) sharpened — appended clarification that the
      ADMISSION gate still applies (callers must be admitted to the target SVTN in
      ANY active role, or be an operator-set member, or be the daemon bootstrap key);
      cross-SVTN callers denied with E-ADM-009 (CWE-862 defense). F-L2-003 removes
      the control-only AUTHORITY gate only, not the ADMISSION gate. EC-006 reference
      updated to note the list-keys admission-gate distinction. EC-008 added —
      enumerates the three reachable list-keys admission failure modes: (1) missing
      CallerPubkey / no ambient bootstrap identity; (2) CallerPubkey present but not
      registered on target SVTN AND not in operator-set AND not bootstrap key
      (cross-SVTN roster enumeration denial, CWE-862); (3) CallerPubkey present,
      registered on target SVTN, but revoked or expired (registered-any-state
      insufficient, active admission required). Refs: F-P5P13-A-001 [HIGH],
      interface-definitions v1.25. Errata pointer: interface-definitions.md v1.25
      changelog is the authoritative reconciliation record.
  - date: 2026-07-15
    version: "1.15"
    actor: product-owner
    change: >
      S-BL.ADMISSION-SYNC-WIRE BC groundwork item A5: added push-failure postcondition
      to PC-1 (register), PC-2 (revoke), and PC-3 (expire). Each write path now includes:
      "If RouterManagementEndpoints is non-empty and the push to a router endpoint fails,
      the write is not rolled back; WARN is logged." This makes push-failure behavior a
      first-class postcondition, not an implementation detail. Cross-reference: BC-2.05.009
      (admission-state-sync push RPC contract). No change to authority, admission, or
      key-store semantics.
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

   **list-keys authority (F-L2-003):** `admin.key.list-keys` is a read-only operation and may be called by any admitted role (control, console, access) or operator-set member. It is NOT subject to the control-only authority gate. The admission requirement itself still applies — callers must be admitted to the target SVTN in ANY active role, or be a member of the operator-set, or be the daemon bootstrap key. Callers whose only relationship to the daemon is a valid operator-key handshake against a DIFFERENT SVTN are denied with E-ADM-009 (CWE-862 defense against cross-SVTN roster enumeration). F-L2-003 removes the control-only AUTHORITY gate; it does NOT remove the ADMISSION gate.
2. The key operation is well-formed: a valid OpenSSH public key in authorized_keys format.
3. The router's distributed key store is reachable.

## Postconditions

1. **Register**: The public key is added to the admitted key set with the specified role (control, console, access). The key becomes active for admission challenges. Propagation: key is pushed to all routers serving the SVTN via `internal.admission.register` RPC (BC-2.05.009). **Push-failure postcondition (A5, S-BL.ADMISSION-SYNC-WIRE groundwork):** If `RouterManagementEndpoints` is non-empty and the push to a router endpoint fails, the control-side `RegisterKey` write is NOT rolled back; the failure is logged at WARN level. The `sbctl` response to the operator reflects the control-side write success. The router is temporarily stale until the next push event or control restart.
2. **Revoke**: The public key is removed from the admitted key set. Existing sessions using this key continue until the next re-authentication challenge (propagation delay acknowledged, FM-007). Propagation: revocation is pushed to all routers via `internal.admission.revoke` RPC (BC-2.05.009). **Push-failure postcondition (A5):** Same as PC-1 — push failure is logged at WARN; control write is not rolled back.
3. **Expire**: An expiry timestamp is associated with the key. At expiry, the key is automatically revoked by routers that honor the expiry timestamp. **Wire field (F-L2-007):** the `admin.key.expire` RPC args field is `after` (a Go duration string, e.g., `"720h"`, `"30m"`). The CLI `--at <RFC3339-timestamp>` flag is translated client-side to a duration (`timestamp - time.Now()`) before sending on the wire. The daemon handler validates `after` as a positive Go duration string; zero, negative, and >100-year values are rejected with E-CFG-001. Propagation: expiry is pushed to all routers via `internal.admission.expire` RPC (BC-2.05.009). **Push-failure postcondition (A5):** Same as PC-1 — push failure is logged at WARN; control write is not rolled back.
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
| EC-006 | Revoked or expired key attempts an admin.key.* operation after successful mgmt authentication (F-P4L1-003) | The key authenticated at the mgmt connection layer but its `revoked=true` or `now >= expiry` in the SVTN admitted set. Handler authority resolution treats the key as unregistered; returns E-ADM-009. This applies to register, revoke, and expire (not list-keys, which is open to all admitted roles per F-L2-003 authority rule — the admission gate still applies; see EC-008). |
| EC-008 | Caller attempts `admin.key.list-keys` but fails the ADMISSION gate | Three reachable failure modes: (1) Missing CallerPubkey in context AND no ambient bootstrap identity — handler cannot resolve caller identity, returns E-ADM-009. (2) CallerPubkey present but not registered on the target SVTN AND not in operator-set AND not the daemon bootstrap key — caller is not admitted to the SVTN in any role, returns E-ADM-009 (CWE-862 defense: cross-SVTN callers must not enumerate another SVTN's admitted roster). (3) CallerPubkey present and registered on the target SVTN but revoked (`revoked=true`) or expired (`now >= expiry`) — registered-any-state is insufficient; only an active admission qualifies, returns E-ADM-009. The AUTHORITY gate (F-L2-003) is bypassed for list-keys so any role suffices; the ADMISSION gate is not bypassed. **Verified by: VP-077.** |
| EC-007 | Operator attempts to revoke OR expire the bootstrap key (permanent trust anchor) with a well-formed request. | For any well-formed request (valid duration, all required fields present) targeting the bootstrap key: revoke returns `ErrBootstrapKeyRevokeForbidden` → E-ADM-020; expire returns `ErrBootstrapKeyExpireForbidden` → E-ADM-021. The bootstrap key cannot be revoked or expired at any time, regardless of whether other control keys have been registered. **Layering note (F-P20L3-001):** Handler input-validation (duration bounds check, required-field validation) fires BEFORE the bootstrap sentinel is consulted. A malformed request targeting the bootstrap key (e.g., `after:"-1h"`, `after:"0s"`, `after:">100y"`, missing `after` field) is rejected by the handler with E-CFG-001 before `SVTNManager.ExpireKey` is called. The mutation-prevention invariant is fully preserved in both paths: `SVTNManager` is never called for well-formed bootstrap-key requests (bootstrap guard fires) OR for any malformed-input requests (input-validation fires), so the key store cannot be mutated in either case. Tests: `TestMapAdminError_ErrorWrapping/ErrBootstrapKeyRevokeForbidden` (revoke sentinel); `TestMapAdminError_ErrorWrapping/ErrBootstrapKeyExpireForbidden` (expire sentinel). |

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
| VP-076 | Bootstrap key non-revocable AND non-expirable invariant: symmetric revoke + expire forbidden sentinels (E-ADM-020/E-ADM-021) | integration |
| VP-077 | Admin list-keys admission-gate — any-role OR operator-set OR bootstrap-key; else E-ADM-009 (EC-008 three failure modes) | integration |

## Traceability

| Field | Value |
|-------|-------|
| L2 Capability | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 |
| L2 Domain Invariants | DI-001 (carrier-grade content separation), DI-002 (private keys never transit), DI-011 (role separation between Tier 1 and Tier 2 keys), DI-012 (control node is a participant) |
| Architecture Module | internal/svtnmgmt |
| Stories | PC-1 (register): S-6.02 (CLI dispatch), S-6.06 (daemon handler); PC-2 (revoke): S-6.02 (CLI dispatch), S-6.06 (daemon handler); PC-3 (expire): S-6.02 (CLI dispatch), S-6.06 (daemon handler); PC-4 (confirmation response with key fingerprint and operation timestamp): S-6.06 (mgmt.Response success envelope); EC-007 (bootstrap non-revocable AND non-expirable): S-6.06 (bootstrap-protection guard for revoke + expire) |
| Capability Anchor Justification | CAP-019 ("Key lifecycle management (register, revoke, expire)") per capabilities.md §CAP-019 — this BC specifies the complete key lifecycle operations that CAP-019 defines as the revocation path |

## Related BCs

- BC-2.05.001 — depends on: registered keys are what the admission challenge verifies
- BC-2.05.002 — related to: revocation eventually removes key from admitted set
- BC-2.05.009 — extends: push-failure postconditions in PC-1/PC-2/PC-3 trace to the admission-state-sync push RPC contract

## Changelog

| Version | Date | Author | Change |
|---------|------|--------|--------|
| 1.15 | 2026-07-15 | product-owner | S-BL.ADMISSION-SYNC-WIRE BC groundwork item A5: added push-failure postcondition to PC-1 (register), PC-2 (revoke), and PC-3 (expire). Each write path now includes: if `RouterManagementEndpoints` is non-empty and the push to a router endpoint fails, the write is NOT rolled back; WARN is logged. `sbctl` response reflects control-side write success. Push RPC command references (BC-2.05.009) added to each propagation note. Added `inputs`, `input-hash`, `extracted_from` frontmatter fields (template conformance). |
| 1.14 | 2026-07-03 | spec-steward | F-P5P14-B-003 traceability fix: EC-008 (three admission-failure modes for admin.key.list-keys) now references owning VP-077. Property text: list-keys admits iff IsAdmittedAnyRole OR OperatorKeySet OR BootstrapKey; else E-ADM-009. Complements VP-075 which scope-excludes list-keys. Verification Properties table extended with VP-077 row. No behavioral change. |
| 1.13 | 2026-07-03 | product-owner | Phase 5 Pass 13 spec-track sharpening (Burst 38). F-P5P13-A-001 [HIGH]: Precondition 1 list-keys authority sentence sharpened — ADMISSION gate still applies; cross-SVTN callers denied E-ADM-009 (CWE-862). EC-006 updated; EC-008 added (three list-keys admission failure modes). |
