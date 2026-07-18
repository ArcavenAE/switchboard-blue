Output AC-008-router-mgmt-tcp-bind.gif
Output AC-008-router-mgmt-tcp-bind.webm

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

Type "# AC-008: router mgmt auto-detects TCP-vs-unix; non-loopback accepted; real TCP push works"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestRouterMgmtListener' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
