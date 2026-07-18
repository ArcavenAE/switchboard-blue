Output AC-004-revoke-expire-destroy-push.gif
Output AC-004-revoke-expire-destroy-push.webm

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

Type "# AC-004: revoke/expire/destroy push corresponding commands; push advisory"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestAdmissionSync_RevokeKey_PushCalledAfterControlWrite|TestAdmissionSync_ExpireKey_PushCalledAfterControlWrite|TestAdmissionSync_RemoveSVTN_PushCalledAfterControlWrite|TestAdmissionSync_PushFailure_AllWritePaths_Advisory' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
