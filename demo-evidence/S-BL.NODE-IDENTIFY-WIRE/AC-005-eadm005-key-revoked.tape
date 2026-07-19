Output AC-005-eadm005-key-revoked.gif
Output AC-005-eadm005-key-revoked.webm

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

Type "# AC-005: ErrKeyRevoked → connection closed E-ADM-005 (revoked key)"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestNodeIdentifyHandshake_ErrKeyRevoked' -v -count=1 -timeout 30s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
