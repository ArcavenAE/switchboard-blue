Output AC-010-sighup-reload-router-mgmt-endpoints.gif
Output AC-010-sighup-reload-router-mgmt-endpoints.webm

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

Type "# AC-010: SIGHUP reload updates RouterManagementEndpoints; in-flight pushes not interrupted"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestAdmissionSync_SIGHUPReload_EndpointListUpdated|TestAdmissionSync_SIGHUPReload_NewListUsedOnNextPush' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
