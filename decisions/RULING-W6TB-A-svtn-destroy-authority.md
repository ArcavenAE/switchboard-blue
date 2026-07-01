---
artifact_id: RULING-W6TB-A-svtn-destroy-authority
document_type: decision
level: ops
version: "1.0"
status: final
producer: architect
timestamp: 2026-07-01T00:00:00
updated: 2026-07-01T00:00:00
cycle: v1.0.0-greenfield
stories_in_scope: [S-6.05]
closes_findings: []
---

# Ruling W6TB-A — S-6.05 SVTN Destroy Authority Model

**Question:** does `admin.svtn.destroy` use bootstrap-only gating (Ruling-2/8 parity
with `admin.svtn.create` per BC-2.07.001 v1.10 Inv-3) or the general control-role
handler gate (like `admin.key.revoke`)?

---

## Decision

**`admin.svtn.destroy` uses the general control-role handler gate (`resolveAndVerifyCallerRole`),
not the bootstrap-only gate used by `admin.svtn.create`.**

The `ErrDestroyUnauthorized` sentinel (E-ADM-011 Variant 2) is raised at the
`SVTNManager.Destroy` Go API layer, not by the handler pre-check. The handler
gates on control-role via `resolveAndVerifyCallerRole` (returning E-ADM-009 on
failure); only callers that pass that gate ever reach `SVTNManager.Destroy`, where
a defense-in-depth `ErrDestroyUnauthorized` check re-validates the role at the
Go-API boundary.

---

## Rationale

### 1. Bootstrap-only is scoped to `admin.svtn.create` by spec text

BC-2.07.001 v1.10 Inv-3 states the bootstrap-only restriction explicitly and
specifically:

> "Bootstrap-only restriction for `admin.svtn.create` (S-6.07 F-P1L1-005 closure):
> `admin.svtn.create` requires the daemon bootstrap key as the authorized caller."

The phrase "for `admin.svtn.create`" scopes the bootstrap-only rule. Inv-3's
opening clause — "Only control-role keys may create or destroy SVTNs" — names
both operations, but the bootstrap-only _restriction_ is stated only for create.
Destroy is not mentioned in the bootstrap-only clause. The canon is: create is
bootstrap-only; destroy is control-role.

The reason is operational: the bootstrap key seeds the very first SVTN. Preventing
any non-bootstrap key from ever creating SVTNs closes the "which key can bootstrap
a new network" ambiguity. Destroy has no such ambiguity — it decommissions an
existing SVTN, which is an operation any admitted control-role key should be
authorized to perform. Restricting destroy to the bootstrap key alone would make
destroy operationally inaccessible in any multi-admin setup where the bootstrap
key is held offline for security.

### 2. The S-6.07 handler pattern (`resolveAndVerifyCallerRole`) is the correct analog

`admin.key.revoke`, `admin.key.register`, and `admin.key.expire` all use
`resolveAndVerifyCallerRole` to gate on control-role at the handler layer and
return E-ADM-009 to non-control callers before the SVTNManager is ever invoked.
`admin.svtn.destroy` is a mutating operation on the same management plane with
the same authority requirement (control-role). Structural consistency demands the
same gate.

`admin.svtn.create` is the special case that skips `resolveAndVerifyCallerRole`
(see `makeAdminSVTNCreateHandler` comment: "resolveAndVerifyCallerRole is NOT
called here because the bootstrap-only constraint is stricter than the general
control-role check"). The inverse: destroy does NOT have the stricter constraint,
so the general handler gate applies.

### 3. E-ADM-011 at the Go-API layer, E-ADM-009 at the handler layer

error-taxonomy.md v3.1 (scope disambiguation for E-ADM-011) is clear:

> "E-ADM-011 is a Go API-layer code returned by `SVTNManager.RevokeKey` or
> `SVTNManager.Destroy` directly. It is NOT reachable via the `admin.key.*`
> mgmt RPC path when the handler-layer authority gate is correctly wired — the
> gate returns E-ADM-009 to non-control callers before `SVTNManager.RevokeKey`
> is ever invoked."

The same pattern applies to `admin.svtn.destroy`: the handler gate (E-ADM-009)
fires for non-control callers; `ErrDestroyUnauthorized` (E-ADM-011 Variant 2) is
a defense-in-depth check at the `SVTNManager.Destroy` Go API layer for callers
that somehow bypass the handler gate (unit-test path, direct API invocation, or
future refactor that removes the handler gate inadvertently).

The story v1.2 AC-004 text ("Only control-role keys may invoke destroy. A
non-control key attempting destroy receives `E-ADM-011`") correctly describes
the Go-API-layer outcome but should be read as: *the caller receives an error
that maps to E-ADM-011*. The observable mgmt-RPC response envelope carries
E-RPC-011 wrapping an E-ADM-009 message when the handler gate fires. E-ADM-011
is only surfaced directly in unit tests that call `SVTNManager.Destroy` without
the handler layer.

### 4. Genesis-recreate interaction (G8): destroying all SVTNs re-opens the genesis carve-out

**Ruling: permitted, and no spec change required.**

When `SVTNManager.Destroy` removes the last SVTN, `HasAnySVTN()` returns false
again. The genesis carve-out in BC-2.07.001 Inv-3 (Ruling-8) states: "On the
first-ever SVTN creation (when `HasAnySVTN() == false`), no keySet entry yet
exists for the bootstrap key. On that path the `IsBootstrapKey(caller)` check
alone suffices."

This means: after all SVTNs are destroyed, a subsequent `admin.svtn.create` call
with the bootstrap key will succeed via the genesis carve-out — the same as the
very first create. This is correct behavior. The bootstrap key is the trust anchor
that is always available for re-initialization. There is no security issue: the
daemon is in a state with zero SVTNs and zero admitted keys; a malicious actor
who could reach this state already has full daemon access. The genesis re-open is
not an escalation; it is recovery.

No change to BC-2.07.001 or to `SVTNManager` behavior is required for this
interaction. The implementation already handles it correctly because `HasAnySVTN()`
reflects live state.

---

## Implications for S-6.05 Acceptance Criteria

### AC-004 (requires control-role to destroy)

The handler for `admin.svtn.destroy` MUST call `resolveAndVerifyCallerRole` before
invoking `SVTNManager.Destroy`. The test `TestSbctlAdmin_SVTNDestroy_RequiresControlRole`
should exercise the handler path and assert:

- Non-control caller receives an error response with code E-RPC-011 and message
  containing `"E-ADM-009"` (handler gate fires).
- The SVTN is not destroyed.

The story v1.2 text "A non-control key attempting destroy receives `E-ADM-011`
(authorization error, Variant 2: destroy)" refers to the Go-API-layer sentinel.
For the RPC path the test MUST assert E-ADM-009 in the wire response (consistent
with how VP-075 tests the `admin.key.*` handler gate). Direct `SVTNManager.Destroy`
unit tests may assert `errors.Is(err, svtnmgmt.ErrDestroyUnauthorized)`.

### Error code table for S-6.05 implementer

| Caller path | Gate that fires | Observable error code |
|-------------|----------------|-----------------------|
| Non-control key via mgmt RPC | `resolveAndVerifyCallerRole` (handler layer) | E-RPC-011 wrapping E-ADM-009 |
| Non-control key via direct Go API | `SVTNManager.Destroy` defense-in-depth | `ErrDestroyUnauthorized` (E-ADM-011 Variant 2) |
| SVTN not found (any caller) | `SVTNManager.Destroy` | `ErrSVTNNotFound` (E-SVTN-003) |

### `--confirm` flag

`interface-definitions.md v1.1` line 117 specifies `--confirm` is required for
`admin.svtn.destroy`. This is correct and unchanged: destroy is a destructive
operation that follows the same ADR-004 `--confirm=<svtn-short-id>` gate as other
destructive admin commands. The `--confirm` gate is enforced at the CLI layer in
`cmd/sbctl`, not at the `SVTNManager` layer. The S-6.05 CLI implementation must
include it.

---

## BC-2.07.001 Patch Instructions (v1.11)

No invariant change is required. The existing Inv-3 text correctly states
"Only control-role keys may create or destroy SVTNs" with the bootstrap-only
restriction scoped to `admin.svtn.create` only. The destroy path is already
implicitly governed by the "control-role" clause without bootstrap restriction.

**Recommended v1.11 amendment (clarification only, no behavior change):**

Add a sentence to Inv-3 after the bootstrap-only block:

> "**Destroy authority (S-6.05 W6TB-A Ruling):** `admin.svtn.destroy` does NOT
> require the daemon bootstrap key. Any admitted control-role key may invoke
> destroy. The handler gate fires `resolveAndVerifyCallerRole` (E-ADM-009 to
> non-control callers at the RPC layer); `SVTNManager.Destroy` applies
> `ErrDestroyUnauthorized` (E-ADM-011 Variant 2) as a defense-in-depth Go-API
> check. Destroying the last SVTN re-opens the genesis carve-out for a subsequent
> `admin.svtn.create` — this is permitted (recovery semantics)."

Add a Canonical Test Vector:

| Input | Expected Output | Category |
|-------|----------------|----------|
| Non-bootstrap control-role key invokes `admin.svtn.destroy --name=mynet` | SVTN destroyed; all admitted keys removed; confirmation returned | happy-path |
| Non-control (console/readonly) key invokes `admin.svtn.destroy` via mgmt RPC | E-RPC-011 wrapping E-ADM-009 "insufficient authority for operation admin.svtn.destroy: key <fp> has role <role>"; SVTN unchanged | error |

---

## Decision Log

| Date | Actor | Entry |
|------|-------|-------|
| 2026-07-01 | architect | Initial ruling: destroy uses general control-role gate, not bootstrap-only. Rationale: Inv-3 bootstrap-only clause is explicitly scoped to create; destroy follows the `resolveAndVerifyCallerRole` pattern of `admin.key.*` handlers. E-ADM-011 is Go-API-layer defense-in-depth only. Genesis re-open after last-SVTN destroy is permitted (recovery, no escalation). BC-2.07.001 v1.11 amendment recommended for clarity. |
