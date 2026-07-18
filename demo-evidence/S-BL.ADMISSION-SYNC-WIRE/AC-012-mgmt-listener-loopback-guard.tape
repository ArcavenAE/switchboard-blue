Output AC-012-mgmt-listener-loopback-guard.gif
Output AC-012-mgmt-listener-loopback-guard.webm

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

Type "# AC-012: mgmt listener loopback guard — console/control/access reject non-loopback; router exempt"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestControlMgmtListener_NonLoopbackRejected|TestControlMgmtListener_LoopbackTCPAccepted|TestRouterMgmtListener_NonLoopbackStillAccepted_Ruling9|TestBuildMgmtListener_ConsoleTCP_RejectsNonLoopback_VP073' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
