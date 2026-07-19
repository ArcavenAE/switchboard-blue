Output AC-001-handshake-success-binding-recorded.gif
Output AC-001-handshake-success-binding-recorded.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.NODE-IDENTIFY-WIRE"
Enter
Type "clear"
Enter
Show

Type "# AC-001: Successful three-message handshake → binding recorded, ServeConn begins"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestNodeIdentifyHandshake_Success_BindingRecorded|TestNodeIdentifyHandshake_Success_ServeConnBegins$|TestNodeIdentifyHandshake_Success_ServeConnBegins_FrameRouted' -v -count=1 -timeout 60s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
