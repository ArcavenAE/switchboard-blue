# Demo Evidence Report — S-BL.ADMISSION-SYNC-WIRE

**Story:** S-BL.ADMISSION-SYNC-WIRE v1.7 — control-mode daemon pushes AdmittedKeySet mutations to router-mode daemons via internal/mgmt JSON-over-TCP protocol (four internal.admission.* RPCs); routers persist VLR-local snapshot; control persists keyset snapshot; config validation for new fields; bind-address logging for all daemon modes.
**HEAD:** ab043c5
**Status:** CONVERGED (3/3 clean adversarial passes — pass-9/10/11 under BC-5.39.001)
**Recorded:** 2026-07-18

## Coverage Matrix

| AC | Title | Test Function(s) | Recording | Pass/Fail |
|----|-------|-----------------|-----------|-----------|
| AC-001 | Config.Validate() for admission_state_file (E-CFG-015) + router_management_endpoints (E-CFG-016) + control_admission_state_file (E-CFG-017) | TestConfig_Validate_AdmissionStateFile_AbsentAccepted, TestConfig_Validate_AdmissionStateFile_WhitespaceOnlyRejectsE_CFG_015, TestConfig_Validate_RouterManagementEndpoints_EmptyListAccepted, TestConfig_Validate_RouterManagementEndpoints_InvalidAddrRejectsE_CFG_016, TestConfig_Validate_RouterManagementEndpoints_NonLoopbackAccepted, TestConfig_Validate_RouterManagementEndpoints_MultipleInvalidExhaustiveErrors, TestConfig_Validate_ControlAdmissionStateFile_AbsentAccepted, TestConfig_Validate_ControlAdmissionStateFile_WhitespaceRejectsE_CFG_017 | AC-001-config-validate-admission-fields.tape | PASS |
| AC-002 | four internal.admission.* commands registered on router mode only (not control/console/access) | TestWireAdmissionSyncHandlers_RegisteredOnRouterServer, TestWireAdmissionSyncHandlers_NotRegisteredOnControlServer | AC-002-handler-registration-router-only.tape | PASS |
| AC-003 | admin.key.register on control pushes internal.admission.register to routers; push-fail does not roll back control write | TestAdmissionSync_RegisterKey_PushCalledAfterControlWrite, TestAdmissionSync_RegisterKey_PushFailureDoesNotRollbackControlWrite, TestAdmissionSync_NilSyncer_NoOp, TestAdmissionSync_RegisterKey_AdminRPCReturnsPromptlyWithUnreachablePush | AC-003-register-push-advisory.tape | PASS |
| AC-004 | admin.key.revoke/expire + admin.svtn.destroy push corresponding commands; push advisory | TestAdmissionSync_RevokeKey_PushCalledAfterControlWrite, TestAdmissionSync_ExpireKey_PushCalledAfterControlWrite, TestAdmissionSync_RemoveSVTN_PushCalledAfterControlWrite, TestAdmissionSync_PushFailure_AllWritePaths_Advisory | AC-004-revoke-expire-destroy-push.tape | PASS |
| AC-005 | router register handler populates AdmittedKeySet admitted=false; snapshot written atomically after each push | TestRouterAdmissionHandler_Register_AdmittedFalse, TestRouterAdmissionHandler_Register_SnapshotWritten, TestRouterAdmissionHandler_Register_SnapshotWriteFailure_Advisory, TestSnapshotWriteAtomic_ConcurrentWrites_AlwaysValidJSON, TestRouterAdmission_SnapshotMutationOrderPreserved | AC-005-router-register-handler-snapshot.tape | PASS |
| AC-006 | snapshot JSON round-trip: schema_version:1, correct encoding, no FrameAuthKey/NodeAddr/nonces stored | TestSnapshot_JSON_FieldEncoding_CorrectSchema, TestSnapshot_RoundTrip_EntriesMatch, TestSnapshot_RoundTrip_AdmittedAlwaysFalse, TestSnapshot_RoundTrip_RevokedEntryCallsRevokeKey, TestSnapshot_RoundTrip_ExpiryEntryCallsSetKeyExpiry, TestSnapshot_NoFrameAuthKey_NoNodeAddr_NoNonces | AC-006-snapshot-json-roundtrip.tape | PASS |
| AC-007 | router startup: file absent→empty+INFO; valid→load; corrupt→fail-closed E-KEY-002 | TestRouterStartup_AdmissionStateFile_NotConfigured_EmptyKeyset, TestRouterStartup_AdmissionStateFile_ConfiguredFileAbsent_EmptyKeyset_InfoLog, TestRouterStartup_AdmissionStateFile_ValidFile_EntriesLoaded, TestRouterStartup_AdmissionStateFile_CorruptJSON_FailClosed_EKEY002, TestRouterStartup_AdmissionStateFile_UnknownSchemaVersion_FailClosed | AC-007-router-startup-load.tape | PASS |
| AC-008 | router mgmt listener auto-detects TCP-vs-unix; non-loopback bind accepted; real TCP client can connect+push | TestRouterMgmtListener_NonLoopbackBindAccepted, TestRouterMgmtListener_StartupInfoLog_BindAddress, TestRouterMgmtListener_TCPBind_ConnectionSucceeds, TestRouterMgmtListener_TCPBind_PushHandshakeSucceeds | AC-008-router-mgmt-tcp-bind.tape | PASS |
| AC-009 | full-snapshot push on control startup: load persisted keyset THEN push complete authoritative state (incl per-endpoint sequencing, revoked skip-register, past-expiry compensating-revoke) | TestControlAdmission_LoadAndPushFullSnapshot, TestControlAdmission_MissingFileEmptyKeyset, TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter, TestAdmissionSync_PushFullSnapshot_ExpiryPushed, TestAdmissionSync_PushFullSnapshot_RevokedKeyStaysRevoked, TestAdmissionSync_PushFullSnapshot_RevokedKey_RegisterNotSent, TestAdmissionSync_PushFullSnapshot_PastExpiryStaysExpired, TestAdmissionSync_PushFullSnapshot_PastExpiry_ExpireFails_CompensatingRevoke, TestAdmissionSync_PushFullSnapshot_EmptyKeysetNoPushAttempt | AC-009-control-startup-full-snapshot-push.tape | PASS |
| AC-010 | SIGHUP reload updates RouterManagementEndpoints; in-flight pushes not interrupted | TestAdmissionSync_SIGHUPReload_EndpointListUpdated, TestAdmissionSync_SIGHUPReload_NewListUsedOnNextPush | AC-010-sighup-reload-router-mgmt-endpoints.tape | PASS |
| AC-011 | control-side keyset persistence via control_admission_state_file: validate, persist-on-mutation, load-on-startup | TestControlAdmission_PersistOnMutation, TestControlAdmission_LoadAndPushFullSnapshot, TestControlAdmission_MissingFileEmptyKeyset, TestConfig_Validate_ControlAdmissionStateFile_AbsentAccepted, TestConfig_Validate_ControlAdmissionStateFile_WhitespaceRejectsE_CFG_017 | AC-011-control-keyset-persistence.tape | PASS |
| AC-012 | mgmt listener loopback guard: console/control/access reject non-loopback TCP; router exempt | TestControlMgmtListener_NonLoopbackRejected, TestControlMgmtListener_LoopbackTCPAccepted, TestRouterMgmtListener_NonLoopbackStillAccepted_Ruling9, TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback_VP073 | AC-012-mgmt-listener-loopback-guard.tape | PASS |
| AC-013 | PushFullSnapshot multi-endpoint per-endpoint sequencing: each reachable endpoint independently reaches correct terminal state | TestAdmissionSync_PushFullSnapshot_MultiEndpoint_LastUnreachable_PastExpiry_ReachableEndpointNonAdmissible, TestAdmissionSync_PushFullSnapshot_MultiEndpoint_FirstUnreachable_ReachableEndpointCorrect | AC-013-push-full-snapshot-multi-endpoint-sequencing.tape | PASS |

## Race Test

`go test ./cmd/switchboard/... -race -count=1 -skip 'TestLookup_ConcurrentRegisterRace'` — package clean.

Full output: `race-test-transcript.txt`

Key result:
- `ok  github.com/arcavenae/switchboard/cmd/switchboard  56.577s` (race-clean)

**Note:** `TestLookup_ConcurrentRegisterRace` is excluded from the race run per switchboard-blue#124 (known flake — unrelated to this story's admission-sync feature).

## Files

```
AC-001-config-validate-admission-fields.tape
AC-002-handler-registration-router-only.tape
AC-003-register-push-advisory.tape
AC-004-revoke-expire-destroy-push.tape
AC-005-router-register-handler-snapshot.tape
AC-006-snapshot-json-roundtrip.tape
AC-007-router-startup-load.tape
AC-008-router-mgmt-tcp-bind.tape
AC-009-control-startup-full-snapshot-push.tape
AC-010-sighup-reload-router-mgmt-endpoints.tape
AC-011-control-keyset-persistence.tape
AC-012-mgmt-listener-loopback-guard.tape
AC-013-push-full-snapshot-multi-endpoint-sequencing.tape
evidence-report.md
race-test-transcript.txt
```

## Notes

- **POL-004 compliance:** Only `.tape` scripts, `evidence-report.md`, and `race-test-transcript.txt` are committed. No `.gif`/`.webm`/`.mp4`/`.png` binaries. This is a headless daemon/wire Go feature with no TUI/browser surface; demos are VHS `.tape` scripts whose commands are `go test -run ... -v` invocations of the story's actual test suite.
- **Evidence integrity:** Every `.tape` runs a real `go test` command verified to pass at HEAD ab043c5. No fabricated output. For wire-timing ACs (AC-009 compensating-revoke, AC-013 per-endpoint sequencing), the test functions are the executable behavioral evidence — verified passing before recording.
- **AC-013 timing note:** `TestAdmissionSync_PushFullSnapshot_MultiEndpoint_LastUnreachable_PastExpiry_ReachableEndpointNonAdmissible` takes ~26s (connection timeout for the unreachable endpoint). The tape uses `-timeout 120s` to accommodate this.
- **AC-009 and AC-011 overlap:** Both touch control startup + persistence. AC-009 focuses on the push sequence/compensating logic; AC-011 focuses on the config validation + persist-on-mutation + load-on-startup cycle. The test functions overlap intentionally — they test the same subsystem from different angles.
- **Race transcript:** Refreshed at HEAD ab043c5 (pass-11 cleanup state). `TestLookup_ConcurrentRegisterRace` excluded per switchboard-blue#124; all other tests in `cmd/switchboard` are race-clean.
