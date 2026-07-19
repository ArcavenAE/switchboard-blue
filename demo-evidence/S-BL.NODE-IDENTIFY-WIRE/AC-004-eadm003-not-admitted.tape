Output AC-004-eadm003-not-admitted.gif
Output AC-004-eadm003-not-admitted.webm

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

Type "# AC-004: ErrNotAdmitted → connection closed E-ADM-003 (unregistered key)"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestNodeIdentifyHandshake_ErrNotAdmitted' -v -count=1 -timeout 30s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
