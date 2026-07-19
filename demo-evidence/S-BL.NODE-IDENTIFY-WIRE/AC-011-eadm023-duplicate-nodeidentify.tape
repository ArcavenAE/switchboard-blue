Output AC-011-eadm023-duplicate-nodeidentify.gif
Output AC-011-eadm023-duplicate-nodeidentify.webm

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

Type "# AC-011: Second NodeIdentify on established connection → hard error E-ADM-023 → connection closed"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestNodeIdentifyHandshake_DuplicateNodeIdentify_E_ADM_023' -v -count=1 -timeout 60s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
