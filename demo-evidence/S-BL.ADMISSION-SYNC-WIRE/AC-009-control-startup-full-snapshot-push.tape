Output AC-009-control-startup-full-snapshot-push.gif
Output AC-009-control-startup-full-snapshot-push.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.ADMISSION-SYNC-WIRE"
Enter
Type "clear"
Enter
Show

Type "# AC-009: control startup loads persisted keyset THEN pushes full authoritative snapshot"
Enter
Type "#  — revoked keys skip register; past-expiry gets compensating revoke"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestControlAdmission_LoadAndPushFullSnapshot|TestControlAdmission_MissingFileEmptyKeyset|TestAdmissionSync_PushFullSnapshot_AllEntriesPushedToRouter|TestAdmissionSync_PushFullSnapshot_ExpiryPushed|TestAdmissionSync_PushFullSnapshot_RevokedKeyStaysRevoked|TestAdmissionSync_PushFullSnapshot_RevokedKey_RegisterNotSent|TestAdmissionSync_PushFullSnapshot_PastExpiryStaysExpired|TestAdmissionSync_PushFullSnapshot_PastExpiry_ExpireFails_CompensatingRevoke|TestAdmissionSync_PushFullSnapshot_EmptyKeysetNoPushAttempt' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
