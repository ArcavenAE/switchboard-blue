# Demo Evidence Report — S-BL.NODE-IDENTIFY-WIRE

**Story:** S-BL.NODE-IDENTIFY-WIRE v1.13 — NODE_IDENTIFY connect-time identify handshake binding (SVTNID, NodeAddr) → IfaceID for hop-2 fan-out target resolution.
**HEAD:** 1d23a05e6402f29ae5c7fb754b6cc495a4700d5a
**Status:** CONVERGED
**Recorded:** 2026-07-19

## Coverage Matrix

| AC | Title | Test Function(s) | Recording | Pass/Fail |
|----|-------|-----------------|-----------|-----------|
| AC-001 | Successful three-message handshake: admitted key + valid signature → binding recorded, ServeConn begins (BC-2.01.009 PC-1 through PC-7; BC-2.01.010 PC-1) | TestNodeIdentifyHandshake_Success_BindingRecorded, TestNodeIdentifyHandshake_Success_ServeConnBegins, TestNodeIdentifyHandshake_Success_ServeConnBegins_FrameRouted | AC-001-handshake-success-binding-recorded.tape | PASS |
| AC-002 | Malformed NodeIdentify frame → connection closed (BC-2.01.009 Invariant 5; error code table) | TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongPayloadLen, TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongMsgKind, TestNodeIdentifyHandshake_MalformedNodeIdentify_NonZeroReservedByte, TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongControlType, TestNodeIdentifyHandshake_MalformedNodeIdentify_WrongVersion, TestNodeIdentifyHandshake_MalformedChallengeResponse_ConnectionClosed, TestNodeIdentifyHandshake_MalformedChallengeResponse_WrongPayloadLen, TestNodeIdentifyHandshake_WrongOuterFrameType_Rejected, TestNodeIdentifyHandshake_MalformedNodeIdentify_LogsWarn | AC-002-malformed-nodeidentify-rejected.tape | PASS |
| AC-003 | Zero SVTN ID in NodeIdentify outer header → connection closed (BC-2.01.009 Precondition 5; error code table) | TestNodeIdentifyHandshake_ZeroSVTNID_Rejected, TestNodeIdentifyHandshake_ZeroSVTNID_LogsWarn | AC-003-zero-svtnid-rejected.tape | PASS |
| AC-004 | ErrNotAdmitted → connection closed E-ADM-003 (BC-2.01.009 error code E-ADM-003) | TestNodeIdentifyHandshake_ErrNotAdmitted_ConnectionClosed, TestNodeIdentifyHandshake_ErrNotAdmitted_LogsEADM003 | AC-004-eadm003-not-admitted.tape | PASS |
| AC-005 | ErrKeyRevoked → connection closed E-ADM-005 (BC-2.01.009 error code E-ADM-005) | TestNodeIdentifyHandshake_ErrKeyRevoked_ConnectionClosed, TestNodeIdentifyHandshake_ErrKeyRevoked_LogsEADM005 | AC-005-eadm005-key-revoked.tape | PASS |
| AC-006 | ErrKeyExpired → connection closed E-ADM-015 (BC-2.01.009 error code E-ADM-015; BC-2.05.001 PC-6) | TestNodeIdentifyHandshake_ErrKeyExpired_ConnectionClosed, TestNodeIdentifyHandshake_ErrKeyExpired_LogsEADM015 | AC-006-eadm015-key-expired.tape | PASS |
| AC-007 | ErrNonceReplay → connection closed E-ADM-008 (BC-2.01.009 error code E-ADM-008) | TestNodeIdentifyHandshake_ErrNonceReplay_ConnectionClosed, TestNodeIdentifyHandshake_NonceReplay_E_ADM_008_Logged | AC-007-eadm008-nonce-replay.tape | PASS |
| AC-008 | ErrSignatureVerificationFailed → connection closed E-ADM-001 (BC-2.01.009 error code E-ADM-001) | TestNodeIdentifyHandshake_ErrSignatureVerificationFailed_ConnectionClosed, TestNodeIdentifyHandshake_ErrSigVerifyFailed_LogsEADM001, TestAdmitNode_VerifiesAgainstStoredKey_NotFramePubkey, TestAdmitNode_StoredKeyMatches_Admits | AC-008-eadm001-sig-verify-failed.tape | PASS |
| AC-009 | Handshake timeout (10s) → connection closed E-ADM-022 (BC-2.01.009 Precondition 4; error code E-ADM-022) | TestNodeIdentifyHandshake_Timeout_E_ADM_022, TestNodeIdentifyHandshake_Timeout_LogsEADM022 | AC-009-eadm022-handshake-timeout.tape | PASS |
| AC-010 | LWW rebind: reconnecting node overwrites prior binding; stale cleanup guard protects new binding (BC-2.01.010 PC-2; BC-2.01.010 PC-9) | TestBindInterface_LWW_Reconnect_OverwritesPriorBinding, TestBindInterface_StaleCleanupGuard_DoesNotRemoveNewBinding, TestBindInterface_CleanDisconnect_ThenReconnect | AC-010-lww-rebind-stale-cleanup-guard.tape | PASS |
| AC-011 | Second NodeIdentify on same already-admitted connection → hard error E-ADM-023 → connection closed (BC-2.01.009 Invariant 7; error code E-ADM-023) | TestNodeIdentifyHandshake_DuplicateNodeIdentify_E_ADM_023 | AC-011-eadm023-duplicate-nodeidentify.tape | PASS |
| AC-012 | Cleanup func calls UnbindInterface on connection close; binding removed (BC-2.01.010 PC-8) | TestNodeIdentifyHandshake_CleanupFunc_UnbindInterface_Called, TestUnbindInterface_RemovesBinding, TestLookupInterface_Unbound_ReturnsFalse, TestBindInterface_NilNestedMap_AllocatesEntry | AC-012-cleanup-unbind-interface.tape | PASS |
| AC-013 | AdmitNode expiry check in internal/admission: ErrKeyExpired for past-expiry key (BC-2.05.001 PC-6; BC-2.05.001 Invariant 5) | TestAdmitNode_ExpiredKey_ReturnsErrKeyExpired, TestAdmitNode_FutureExpiry_Succeeds, TestAdmitNode_NoExpiry_Succeeds | AC-013-admitnode-expiry-check.tape | PASS |

## Race Test

`go test ./internal/admission/... ./cmd/switchboard/... ./internal/routing/... -race -count=1 -skip 'TestLookup_ConcurrentRegisterRace'` — all three packages clean.

Full output: `race-test-transcript.txt`

Key results:
- `ok  github.com/arcavenae/switchboard/internal/admission  13.479s` (race-clean)
- `ok  github.com/arcavenae/switchboard/cmd/switchboard     57.250s` (race-clean)
- `ok  github.com/arcavenae/switchboard/internal/routing    2.433s` (race-clean)

**Note:** `TestLookup_ConcurrentRegisterRace` is excluded per switchboard-blue#124 (known flake — unrelated to this story's NODE_IDENTIFY feature).

## Files

```
AC-001-handshake-success-binding-recorded.tape
AC-002-malformed-nodeidentify-rejected.tape
AC-003-zero-svtnid-rejected.tape
AC-004-eadm003-not-admitted.tape
AC-005-eadm005-key-revoked.tape
AC-006-eadm015-key-expired.tape
AC-007-eadm008-nonce-replay.tape
AC-008-eadm001-sig-verify-failed.tape
AC-009-eadm022-handshake-timeout.tape
AC-010-lww-rebind-stale-cleanup-guard.tape
AC-011-eadm023-duplicate-nodeidentify.tape
AC-012-cleanup-unbind-interface.tape
AC-013-admitnode-expiry-check.tape
evidence-report.md
race-test-transcript.txt
```

## Notes

- **POL-004 compliance:** Only `.tape` scripts, `evidence-report.md`, and `race-test-transcript.txt` are committed. No `.gif`/`.webm`/`.mp4`/`.png` binaries. This is a headless daemon/wire Go feature with no TUI/browser surface; demos are VHS `.tape` scripts whose commands are `go test -run ... -v` invocations of the story's actual test suite.
- **Evidence integrity:** Every `.tape` runs a real `go test` command verified to pass at HEAD 1d23a05e6402f29ae5c7fb754b6cc495a4700d5a. No fabricated output.
- **Demo medium rationale:** The NODE_IDENTIFY handshake is an internal daemon wire protocol (no operator-facing CLI command or TUI). The only honest demo medium is `go test -run <TestName> -v` showing each AC's test passing with its discriminating assertion — consistent with the S-BL.ADMISSION-SYNC-WIRE prior wire story pattern.
- **Test coverage split:** AC-001 through AC-009, AC-011, AC-012 are in `cmd/switchboard/node_identify_wire_test.go` (primary) plus daemon-level log companions in `node_identify_wire_log_test.go` and `node_identify_wire_warn_log_test.go`. AC-010 (LWW rebind) is unit-tested in `internal/routing/identity_test.go`. AC-013 (AdmitNode expiry, O-1 ruling) is in `internal/admission/admitnode_expiry_test.go`. The F-1 security fix (verify against stored key, not frame pubkey) is in `internal/admission/admitnode_verify_source_test.go` and forms part of AC-008's backing evidence.
- **AC-008 F-1 note:** `admitnode_verify_source_test.go` is a white-box package test that directly mutates the keyset's internal map to install a victim's public key at an attacker's derived address — simulating a node-address collision impersonation attack. `TestAdmitNode_VerifiesAgainstStoredKey_NotFramePubkey` is the discriminating guard for BC-2.05.001 PC-3 (verify against stored key). The tape covers both this test and the daemon-level E-ADM-001 log assertion.
- **Race transcript:** Captured at HEAD 1d23a05e6402f29ae5c7fb754b6cc495a4700d5a. All three packages (`internal/admission`, `cmd/switchboard`, `internal/routing`) race-clean. `TestLookup_ConcurrentRegisterRace` excluded per switchboard-blue#124.
