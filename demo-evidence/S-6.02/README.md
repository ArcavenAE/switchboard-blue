# Demo Evidence — S-6.02: SVTN Lifecycle and Key Management

**Story:** S-6.02 — implement SVTN lifecycle and key management via sbctl admin  
**Branch:** feat/S-6.02-svtn-lifecycle-sbctl-admin  
**Recorded:** 2026-06-30  
**VHS available:** no — captured command output used as fallback per demo-recorder protocol

---

## Coverage Summary

| AC | Description | BC Trace | Artifact | Result |
|----|-------------|----------|----------|--------|
| AC-001 | SVTNManager.Create bootstraps first control key, returns SVTN-ID | BC-2.07.001 PC-1,PC-2 | [AC-001-svtn-create-bootstrap-control-key.txt](AC-001-svtn-create-bootstrap-control-key.txt) | PASS |
| AC-002 | `sbctl admin key register` — key appears in admission checks | BC-2.05.004 PC-1 | [AC-002-key-register-cli.txt](AC-002-key-register-cli.txt) | PASS |
| AC-003 | `sbctl admin key revoke` — removes key; subsequent admission returns E-ADM-002 | BC-2.05.004 PC-2 | [AC-003-key-revoke-cli.txt](AC-003-key-revoke-cli.txt) | PASS |
| AC-004 | `sbctl admin key expire` — sets TTL; zero duration returns E-CFG-001 | BC-2.05.004 PC-3 | [AC-004-key-expire-cli.txt](AC-004-key-expire-cli.txt) | PASS |
| AC-005 | Control-to-control revocation requires `--confirm`; rejected without it | BC-2.05.004 PC-1/ADR-004 | [AC-005-control-revocation-requires-confirm.txt](AC-005-control-revocation-requires-confirm.txt) | PASS |

All 5 acceptance criteria demonstrated. Both success and error paths covered for each.

---

## Replay Commands

### AC-001 — SVTN Create + Bootstrap Control Key

```
go test -count=1 \
  -run "TestSVTNManager_Create_BootstrapsControlKey|TestSVTNManager_Create_BootstrapKeyAdmittedFalse_TrustAnchor|TestSVTNManager_VP048_CreateIdempotentFirstInvocation|TestSVTNManager_VP048_BootstrappedKeyIsControlRole|TestSVTNManager_CreateBootstrapAtomicity_RaceDetector" \
  -v -race ./internal/svtnmgmt/
```

**Tests covered:**
- `TestSVTNManager_Create_BootstrapsControlKey` — success path: Create returns SVTN-ID and bootstraps a control-role key
- `TestSVTNManager_Create_BootstrapKeyAdmittedFalse_TrustAnchor` — error/security path: bootstrap key has admitted=false (trust anchor, not a session key)
- `TestSVTNManager_VP048_CreateIdempotentFirstInvocation` — VP-048 property 1: first invocation semantics, no duplicate SVTN
- `TestSVTNManager_VP048_BootstrappedKeyIsControlRole` — VP-048 property 2: bootstrapped key role=control
- `TestSVTNManager_CreateBootstrapAtomicity_RaceDetector` — race detector: Create+bootstrap is atomic under concurrent access (HOLD-001)

---

### AC-002 — Key Register CLI

```
go test -count=1 \
  -run "TestSbctlAdmin_KeyRegister_CLI|TestAdminKeyRegisterArgs_JSONRoundTrip|TestSVTNManager_RegisterKey_AppearsInAdmissionChecks|TestSVTNManager_RegisterKey_DuplicateLastWriteWins|TestSVTNManager_RegisterKey_SVTNNotFound" \
  -v ./internal/svtnmgmt/ ./cmd/sbctl/
```

**Tests covered:**
- `TestSbctlAdmin_KeyRegister_CLI` — success path: `sbctl admin key register --key ... --svtn ...` dispatches wire envelope, receives JSON confirmation with fingerprint
- `TestAdminKeyRegisterArgs_JSONRoundTrip` — wire schema: AdminKeyRegisterArgs round-trips via JSON
- `TestSVTNManager_RegisterKey_AppearsInAdmissionChecks` — success path (3 roles: control, console, access): registered key is admitted
- `TestSVTNManager_RegisterKey_DuplicateLastWriteWins` — error/edge: duplicate register uses last-write-wins (ADR-003)
- `TestSVTNManager_RegisterKey_SVTNNotFound` — error path: register against unknown SVTN-ID returns error

---

### AC-003 — Key Revoke CLI

```
go test -count=1 \
  -run "TestSbctlAdmin_KeyRevoke_CLI|TestAdminKeyRevokeArgs_JSONRoundTrip|TestSVTNManager_RevokeKey_RemovesFromAdmissionSet|TestSVTNManager_RevokeKey_KeyNotFound|TestSVTNManager_RevokeKey_SVTNNotFound|TestSVTNManager_RevokeRaceVsRegister_HOLD001" \
  -v -race ./internal/svtnmgmt/ ./cmd/sbctl/
```

**Tests covered:**
- `TestSbctlAdmin_KeyRevoke_CLI` — success path: `sbctl admin key revoke --key ... --svtn ...` dispatches wire envelope, receives JSON confirmation
- `TestAdminKeyRevokeArgs_JSONRoundTrip` — wire schema: confirm=true and confirm=false both round-trip
- `TestSVTNManager_RevokeKey_RemovesFromAdmissionSet` — success path: revoked key is no longer admitted (would return E-ADM-002)
- `TestSVTNManager_RevokeKey_KeyNotFound` — error path: revoke unknown key returns E-ADM-013
- `TestSVTNManager_RevokeKey_SVTNNotFound` — error path: revoke on unknown SVTN-ID returns error
- `TestSVTNManager_RevokeRaceVsRegister_HOLD001` — race detector: HOLD-001 hybrid atomic RevokeKeyIfRoleMatches under concurrent register/revoke

---

### AC-004 — Key Expire CLI

```
go test -count=1 \
  -run "TestSbctlAdmin_KeyExpire_CLI|TestAdminKeyExpireArgs_JSONRoundTrip|TestSVTNManager_ExpireKey_SetsTTL|TestSVTNManager_ExpireKey_ZeroDurationReturnsError|TestSbctlAdmin_KeyExpire_ZeroDurationAfterFlag|TestSVTNManager_ExpireKey_SVTNNotFound|TestSVTNManager_ExpireKey_KeyNotFound" \
  -v ./internal/svtnmgmt/ ./cmd/sbctl/
```

**Tests covered:**
- `TestSbctlAdmin_KeyExpire_CLI` — success path: `sbctl admin key expire --key ... --svtn ... --after 24h` dispatches wire envelope
- `TestAdminKeyExpireArgs_JSONRoundTrip` — wire schema: AdminKeyExpireArgs round-trips via JSON
- `TestSVTNManager_ExpireKey_SetsTTL` — success path: TTL is set on the key entry
- `TestSVTNManager_ExpireKey_ZeroDurationReturnsError` — error path: zero/negative duration returns E-CFG-001 (3 sub-cases: zero, negative, negative-microsecond)
- `TestSbctlAdmin_KeyExpire_ZeroDurationAfterFlag` — error path: CLI rejects zero/invalid --after flag before dispatch
- `TestSVTNManager_ExpireKey_SVTNNotFound` — error path: expire on unknown SVTN returns error
- `TestSVTNManager_ExpireKey_KeyNotFound` — error path: expire on unregistered key returns error

---

### AC-005 — Control-to-Control Revocation Requires --confirm

```
go test -count=1 \
  -run "TestSbctlAdmin_ControlRevocation_RequiresConfirm|TestSVTNManager_ControlRevocation_RequiresConfirm|TestSVTNManager_RevokeKey_NonControlNoConfirmRequired|TestSVTNManager_RevokeKey_RoleMismatchReturnsError" \
  -v ./internal/svtnmgmt/ ./cmd/sbctl/
```

**Tests covered:**
- `TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI/without_confirm_confirm_false_in_wire` — error path: CLI without `--confirm` produces wire envelope with confirm=false; daemon returns `E-ADM-004: control-to-control revocation requires --confirm flag (ADR-004)`; printed as `E-RPC-001 rpc failed: admin.key.revoke: E-ADM-004: ...`
- `TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI/with_confirm_confirm_true_in_wire` — success path: `--confirm` flag sets confirm=true in wire envelope; daemon accepts and returns fingerprint
- `TestSVTNManager_ControlRevocation_RequiresConfirm/without_confirm_returns_error` — unit error path: SVTNManager.RevokeKey with control-role key and confirm=false returns ErrControlRevocationRequiresConfirm
- `TestSVTNManager_ControlRevocation_RequiresConfirm/with_confirm_succeeds` — unit success path: confirm=true allows revocation
- `TestSVTNManager_RevokeKey_NonControlNoConfirmRequired` — boundary: console and access role keys do not require --confirm
- `TestSVTNManager_RevokeKey_RoleMismatchReturnsError` — error path: role declared in revoke request does not match stored role, returns E-ADM-019

---

## Notable Behaviors Demonstrated

| Behavior | AC | Evidence |
|----------|-----|---------|
| HOLD-001 hybrid atomic RevokeKeyIfRoleMatches | AC-003 | `TestSVTNManager_RevokeRaceVsRegister_HOLD001` — race detector green under 50 goroutines |
| ErrControlRevocationRequiresConfirm gate | AC-005 | CLI emits `E-ADM-004` error string; unit test confirms sentinel error returned |
| Bootstrap key admitted=false trust anchor | AC-001 | `TestSVTNManager_Create_BootstrapKeyAdmittedFalse_TrustAnchor` — admitted field is false |
| Concurrent Create atomicity | AC-001 | `TestSVTNManager_CreateBootstrapAtomicity_RaceDetector` — no race under -race |

---

## Note on Recording Toolchain

VHS was not available on this machine (`which vhs` returned not found). Per demo-recorder
protocol, captured `go test -v -race` command output is used as fallback evidence.
The output files are direct `go test` stdout captures — each test name maps 1:1 to an
acceptance criterion and includes both success and error sub-cases.
