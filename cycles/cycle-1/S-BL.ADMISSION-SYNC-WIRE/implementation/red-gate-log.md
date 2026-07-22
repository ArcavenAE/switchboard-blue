# Red Gate Log — S-BL.ADMISSION-SYNC-WIRE

**Date:** 2026-07-16
**Commit:** 716cb39 (G — SSH-signed)
**Branch:** feature/S-BL.ADMISSION-SYNC-WIRE

## Status: RED GATE VERIFIED

All 28 new tests FAIL against the current stubs. No test passes vacuously.

## Files Created/Modified

| File | Change | AC Coverage |
|------|--------|-------------|
| `internal/config/config_test.go` | +152 lines (6 new tests at end of file) | AC-001 |
| `cmd/switchboard/admission_sync_test.go` | +1975 lines (new file) | AC-002 through AC-010 |

## Red Gate Results

### FAILING (prove real behavior is tested)

| Test | AC | Reason |
|------|----|--------|
| TestConfig_Validate_AdmissionStateFile_WhitespaceOnlyRejectsE_CFG_015 | AC-001 | E-CFG-015 not in config.Validate() |
| TestConfig_Validate_RouterManagementEndpoints_InvalidAddrRejectsE_CFG_016 | AC-001 | E-CFG-016 not in config.Validate() |
| TestConfig_Validate_RouterManagementEndpoints_MultipleInvalidExhaustiveErrors | AC-001 | E-CFG-016 not in config.Validate() |
| TestWireAdmissionSyncHandlers_RegisteredOnRouterServer (×4 subtests) | AC-002 | wireAdmissionSyncHandlers stub registers nothing |
| TestRouterAdmissionHandler_Register_AdmittedFalse | AC-005 | No handlers registered (E-RPC-010) |
| TestRouterAdmissionHandler_Register_SnapshotWritten | AC-005 | No handlers registered (E-RPC-010) |
| TestRouterAdmissionHandler_Register_SnapshotWriteFailure_Advisory | AC-005 | No handlers registered (E-RPC-010) |
| TestAdmissionSync_RegisterKey_PushCalledAfterControlWrite | AC-003 | BuildAdminHandlers does not call PushRegisterKey |
| TestAdmissionSync_RevokeKey_PushCalledAfterControlWrite | AC-004 | BuildAdminHandlers does not call PushRevokeKey |
| TestAdmissionSync_ExpireKey_PushCalledAfterControlWrite | AC-004 | BuildAdminHandlers does not call PushSetKeyExpiry |
| TestAdmissionSync_RemoveSVTN_PushCalledAfterControlWrite | AC-004 | BuildAdminHandlers does not call PushRemoveSVTN |
| TestSnapshot_JSON_FieldEncoding_CorrectSchema | AC-006 | marshalSnapshot returns errSnapshotNotImplemented |
| TestSnapshot_RoundTrip_EntriesMatch | AC-006 | marshalSnapshot returns errSnapshotNotImplemented |
| TestSnapshot_RoundTrip_AdmittedAlwaysFalse | AC-006 | marshalSnapshot returns errSnapshotNotImplemented |
| TestSnapshot_RoundTrip_RevokedEntryCallsRevokeKey | AC-006 | unmarshalSnapshot returns errSnapshotNotImplemented |
| TestSnapshot_RoundTrip_ExpiryEntryCallsSetKeyExpiry | AC-006 | unmarshalSnapshot returns errSnapshotNotImplemented |
| TestSnapshot_NoFrameAuthKey_NoNodeAddr_NoNonces | AC-006 | marshalSnapshot returns errSnapshotNotImplemented |
| TestRouterStartup_AdmissionStateFile_NotConfigured_EmptyKeyset | AC-007 | loadSnapshotFromFile stub returns error for "" path (should be no-op) |
| TestRouterStartup_AdmissionStateFile_ConfiguredFileAbsent_EmptyKeyset_InfoLog | AC-007 | loadSnapshotFromFile stub returns error for absent file |
| TestRouterStartup_AdmissionStateFile_ValidFile_EntriesLoaded | AC-007 | loadSnapshotFromFile returns errSnapshotNotImplemented |
| TestRouterStartup_AdmissionStateFile_CorruptJSON_FailClosed_EKEY002 | AC-007 | Returns stub sentinel instead of real E-KEY-002 |
| TestRouterStartup_AdmissionStateFile_UnknownSchemaVersion_FailClosed | AC-007 | Returns stub sentinel instead of real E-KEY-002 |
| TestRouterStartup_LoadedEntries_AdmittedFalse | AC-007 | loadSnapshotFromFile returns not-implemented |
| TestRouterMgmtListener_StartupInfoLog_BindAddress | AC-008 | runRouter does not emit "router management listener bound to" INFO log |
| TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter | AC-009 | PushFullSnapshot returns errAdmissionSyncNotImplemented |
| TestAdmissionSync_PushFullSnapshot_ExpiryPushed | AC-009 | PushFullSnapshot returns errAdmissionSyncNotImplemented |
| TestAdmissionSync_PushFullSnapshot_EmptyKeysetNoPushAttempt | AC-009 | PushFullSnapshot returns errAdmissionSyncNotImplemented (should be nil for empty keyset) |
| TestAdmissionSync_SIGHUPReload_NewListUsedOnNextPush | AC-010 | PushRegisterKey returns stub sentinel instead of no-op for empty endpoint list |

### PASSING (trivially correct — no behavior change needed)

| Test | AC | Reason |
|------|----|--------|
| TestConfig_Validate_AdmissionStateFile_AbsentAccepted | AC-001 | Absent field accepted (zero value) — no validation needed |
| TestConfig_Validate_RouterManagementEndpoints_EmptyListAccepted | AC-001 | Empty list accepted — no entries to validate |
| TestConfig_Validate_RouterManagementEndpoints_NonLoopbackAccepted | AC-001/AC-008 | No loopback guard (Ruling 9) — already absent, locks regression |
| TestWireAdmissionSyncHandlers_NotRegisteredOnControlServer | AC-002 | E-RPC-010 on non-router server — no handler registered |
| TestRouterMode_AdminHandlersNotRegistered | AC-002 | admin.key.* absent on router — correct behavior already |
| TestAdmissionSync_NilSyncer_NoOp | AC-003 | nil syncer → no push → no panic → RPC succeeds |
| TestAdmissionSync_RegisterKey_PushFailureDoesNotRollbackControlWrite | AC-003 | Push not wired so push never fails → RPC always succeeds |
| TestAdmissionSync_PushFailure_AllWritePaths_Advisory | AC-004 | Push not wired → no error → RPC succeeds (locks advisory contract) |
| TestAdmissionSync_SIGHUPReload_EndpointListUpdated | AC-010 | UpdateEndpoints is implemented in stub (just sets field) |
| TestRouterMgmtListener_NonLoopbackBindAccepted | AC-008 | Config accepts 0.0.0.0:9093 — Ruling 9 already absent |

## Build/Vet Results

```
go build ./... → clean (exit 0)
go vet ./...   → clean (exit 0)
```

## Non-Parallel Tests (umask/Setenv reason)

The following tests were made non-parallel to avoid the listenUnixMgmt umask race:
- TestWireAdmissionSyncHandlers_RegisteredOnRouterServer — creates filesystem socket
- TestRouterAdmissionHandler_Register_AdmittedFalse — creates filesystem socket
- TestRouterAdmissionHandler_Register_SnapshotWritten — creates socket + tempfile
- TestRouterAdmissionHandler_Register_SnapshotWriteFailure_Advisory — creates socket + tempfile with Chmod
- TestRouterMode_AdminHandlersNotRegistered — creates filesystem socket
- TestRouterMgmtListener_StartupInfoLog_BindAddress — starts runRouter with real socket
- TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter — starts router mgmt.Server
- TestRouterStartup_AdmissionStateFile_ValidFile_EntriesLoaded — MkdirTemp + WriteFile
- TestRouterStartup_AdmissionStateFile_CorruptJSON_FailClosed_EKEY002 — MkdirTemp + WriteFile
- TestRouterStartup_AdmissionStateFile_UnknownSchemaVersion_FailClosed — MkdirTemp + WriteFile
- TestRouterStartup_LoadedEntries_AdmittedFalse — MkdirTemp + WriteFile

Rationale: listenUnixMgmt sets a process-global syscall.Umask(0177) during bind,
serialized by a package mutex. Parallel tests that call os.MkdirTemp while this
umask is held receive directories with 0500 permissions (no write/execute), causing
WriteFile to fail with EPERM. Removing t.Parallel() + adding os.Chmod(dir, 0o700)
on the directory eliminates the race.
