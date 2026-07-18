Output AC-005-router-register-handler-snapshot.gif
Output AC-005-router-register-handler-snapshot.webm

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

Type "# AC-005: router register handler admitted=false; snapshot written atomically"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestRouterAdmissionHandler_Register|TestSnapshotWriteAtomic|TestRouterAdmission_SnapshotMutationOrderPreserved' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
