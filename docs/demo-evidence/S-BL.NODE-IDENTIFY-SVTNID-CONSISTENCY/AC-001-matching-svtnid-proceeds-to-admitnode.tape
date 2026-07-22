Output AC-001-matching-svtnid-proceeds-to-admitnode.gif
Output AC-001-matching-svtnid-proceeds-to-admitnode.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.NODE-IDENTIFY-SVTNID-CONSISTENCY"
Enter
Type "clear"
Enter
Show

Type "# AC-001: Matching SVTNID in ChallengeResponse → handshake proceeds to AdmitNode → binding recorded"
Enter
Type "# BC-2.01.009 PC-9 — guard MUST NOT fire on the valid (matching) path"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/ -run 'TestNodeIdentifyHandshake_Success_BindingRecorded' -count=1 -v"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
