Output AC-006-eadm015-key-expired.gif
Output AC-006-eadm015-key-expired.webm

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

Type "# AC-006: ErrKeyExpired → connection closed E-ADM-015 (AdmitNode O-1 expiry check, BC-2.05.001 PC-6)"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestNodeIdentifyHandshake_ErrKeyExpired' -v -count=1 -timeout 30s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
