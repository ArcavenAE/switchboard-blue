# Demo Evidence Report — S-6.06

**Story:** S-6.06 — Daemon-Side Admin RPC Handlers  
**Worktree HEAD:** d3f186c  
**Recorded:** 2026-06-30  
**Tool:** VHS 0.11.0 (Font: Menlo, Theme: Catppuccin Mocha)

## Coverage Summary

| AC | Description | Status | Evidence |
|----|-------------|--------|----------|
| AC-001 | BuildAdminHandlers unit — 4 handlers, happy paths, error mapping | PASS | GIF + WEBM |
| AC-002 | admin.key.revoke e2e — role-mismatch (E-ADM-019), no-confirm (E-ADM-018), success | PASS | GIF + WEBM |
| AC-003 | register + expire + list-keys e2e — register appears in list; expire sets TTL | PASS | GIF + WEBM |
| AC-004 | Per-daemon-mode registration — control dispatches; access returns E-RPC-010 | PASS | GIF + WEBM |
| AC-005 | Server-side TTL validation — valid accepted; <=0 and >100y rejected E-CFG-001 | PARTIAL | GIF + WEBM (unit only) |
| AC-006 | Caller-role authority gate — non-control E-ADM-009; list-keys any-role ok | PASS | GIF + WEBM |

5 of 6 ACs have full coverage. AC-005 has unit-level coverage only (see known issues).

---

## AC-001 — BuildAdminHandlers unit tests

**Traces:** BC-2.05.004 PC-1/PC-2/PC-3/PC-4

**Happy path:** `BuildAdminHandlers` returns 4 handlers; register/revoke/expire/list-keys
all return `{key_fingerprint, timestamp}` in the success payload (BC-2.05.004 PC-4).

**Error path:** `ErrSVTNNotFound` → `E-SVTN-003`; `ErrRoleMismatch` → `E-ADM-019`;
nil `SVTNManager` panics (EC-004).

| Artifact | Path |
|----------|------|
| GIF | `docs/demo-evidence/S-6.06/AC-001-build-admin-handlers-unit.gif` |
| WEBM | `docs/demo-evidence/S-6.06/AC-001-build-admin-handlers-unit.webm` |
| Tape | `docs/demo-evidence/S-6.06/AC-001-build-admin-handlers-unit.tape` |

---

## AC-002 — admin.key.revoke e2e (real mgmt.Server + Unix socket)

**Traces:** BC-2.05.004 PC-2; HOLD-001 hybrid; ADR-004

**Happy path:** control-to-control revoke with `confirm:true` succeeds; key no longer in
`AdmittedKeySet`.

**Error paths:**
- `console` role claims `control` key → `E-ADM-019` (role mismatch)
- control-to-control without `confirm:true` → `E-ADM-018`

| Artifact | Path |
|----------|------|
| GIF | `docs/demo-evidence/S-6.06/AC-002-revoke-handler-e2e.gif` |
| WEBM | `docs/demo-evidence/S-6.06/AC-002-revoke-handler-e2e.webm` |
| Tape | `docs/demo-evidence/S-6.06/AC-002-revoke-handler-e2e.tape` |

---

## AC-003 — register + expire + list-keys e2e

**Traces:** BC-2.05.004 PC-1/PC-3

**Happy paths:**
- `admin.key.register` → key appears in subsequent `admin.key.list-keys` response
- `admin.key.expire` → expiry timestamp set on key entry
- `admin.key.list-keys` with two registered keys → both returned

| Artifact | Path |
|----------|------|
| GIF | `docs/demo-evidence/S-6.06/AC-003-register-expire-list-e2e.gif` |
| WEBM | `docs/demo-evidence/S-6.06/AC-003-register-expire-list-e2e.webm` |
| Tape | `docs/demo-evidence/S-6.06/AC-003-register-expire-list-e2e.tape` |

---

## AC-004 — per-daemon-mode handler registration

**Traces:** BC-2.05.004 PC-1; ADR-004 role-exclusion

**Happy path:** control-mode daemon with `BuildAdminHandlers` registered; `admin.key.register`
dispatches (not `E-RPC-010`).

**Error path:** access-mode daemon (nil handlers) returns `E-RPC-010 "unknown command"`;
non-control-role callers (access, console) on any daemon are rejected.

| Artifact | Path |
|----------|------|
| GIF | `docs/demo-evidence/S-6.06/AC-004-per-mode-handler-registration.gif` |
| WEBM | `docs/demo-evidence/S-6.06/AC-004-per-mode-handler-registration.webm` |
| Tape | `docs/demo-evidence/S-6.06/AC-004-per-mode-handler-registration.tape` |

---

## AC-005 — server-side TTL validation (DI-003 defense-in-depth)

**Traces:** BC-2.05.004 PC-3; DI-003

**Happy path (unit):** expire handler accepts `24h` TTL; `TestBuildAdminHandlers_KeyExpire_HappyPath` PASS.

**Error path (unit):** negative TTL (`-1h`), zero TTL (`0s`), TTL exceeding 100 years (`876001h`),
and missing `after` field all return `E-CFG-001`; `TestBuildAdminHandlers_KeyExpire_NegativeTTL` and
`TestBuildAdminHandlers_KeyExpire_MissingAfterField` PASS.

**Known issue (DEMO-ISSUE-001):** `TestE2E_AdminExpire_ServerRejectsTTL*` tests currently fail
with `E-ADM-009` instead of `E-CFG-001`. Root cause: the test helper `newE2ESVTNManager` creates
a control key separate from `startE2EServer`'s daemon key. The `sendAdminRPC` authenticates as
the server daemon key, which is not registered in the test's SVTNManager — so
`resolveAndVerifyCallerRole` fires `E-ADM-009` before TTL validation. Fix: TTL rejection tests
must use `startE2EServerWithOps` passing the SVTNManager's control key as `daemonPriv`, or
register the server daemon key into the SVTNManager before the TTL test runs.

| Artifact | Path |
|----------|------|
| GIF | `docs/demo-evidence/S-6.06/AC-005-server-side-ttl-validation.gif` |
| WEBM | `docs/demo-evidence/S-6.06/AC-005-server-side-ttl-validation.webm` |
| Tape | `docs/demo-evidence/S-6.06/AC-005-server-side-ttl-validation.tape` |

---

## AC-006 — caller-role authority gate (VP-075)

**Traces:** BC-2.05.004 Precondition 1; DI-001; ADR-004; VP-075

**Error path:** console-role key calling `admin.key.register` → `E-ADM-009`;
access-role key calling `admin.key.revoke` → `E-ADM-009`.
Caller identity resolved server-side from authenticated handshake pubkey (never from payload).

**Happy path (F-L2-003):** console-role key calling `admin.key.list-keys` → `ok: true` (not
`E-ADM-009`); list-keys is read-only and admits any role.

| Artifact | Path |
|----------|------|
| GIF | `docs/demo-evidence/S-6.06/AC-006-caller-role-authority-gate.gif` |
| WEBM | `docs/demo-evidence/S-6.06/AC-006-caller-role-authority-gate.webm` |
| Tape | `docs/demo-evidence/S-6.06/AC-006-caller-role-authority-gate.tape` |

---

## Known Issues

### DEMO-ISSUE-001 — AC-005 e2e TTL rejection tests return E-ADM-009

**Severity:** medium (unit coverage demonstrates the behavior correctly)  
**Affected tests:** `TestE2E_AdminExpire_ServerRejectsTTLNegative`, `TestE2E_AdminExpire_ServerRejectsTTLZero`,
`TestE2E_AdminExpire_ServerRejectsTTLTooLong`

**Root cause:** These tests call `newE2ESVTNManager(t, "test-svtn", ...)` (which generates its own
control key) then `startE2EServer(t, handlers)` (which generates a separate daemon key). The
`sendAdminRPC` function authenticates using the server daemon key (from `testDaemonKeys`), but that
key is not registered in the test SVTNManager and is not the SVTNManager's bootstrap key. Result:
`resolveAndVerifyCallerRole` correctly denies the unregistered key with `E-ADM-009` before the
TTL bounds check fires.

**Fix required:** Use `startE2EServerWithOps` passing the `ctrlPriv` from `newE2ESVTNManager`
as the `daemonPriv`, so the server authenticates as the control key that IS registered in the
SVTNManager. This aligns the test setup with the same pattern used by the passing TTL tests
in `TestControlMode_AdminHandlersRegistered`.
