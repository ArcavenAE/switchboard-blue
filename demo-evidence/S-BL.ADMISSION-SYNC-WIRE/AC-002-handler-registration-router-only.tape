Output AC-002-handler-registration-router-only.gif
Output AC-002-handler-registration-router-only.webm

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

Type "# AC-002: internal.admission.* handlers registered on router only"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestWireAdmissionSyncHandlers|TestRouterMode_AdminHandlersNotRegistered' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
