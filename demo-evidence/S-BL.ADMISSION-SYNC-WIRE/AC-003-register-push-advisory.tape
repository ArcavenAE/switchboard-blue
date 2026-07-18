Output AC-003-register-push-advisory.gif
Output AC-003-register-push-advisory.webm

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

Type "# AC-003: admin.key.register pushes to routers; push-fail does not roll back control write"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestAdmissionSync_RegisterKey|TestAdmissionSync_NilSyncer' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
