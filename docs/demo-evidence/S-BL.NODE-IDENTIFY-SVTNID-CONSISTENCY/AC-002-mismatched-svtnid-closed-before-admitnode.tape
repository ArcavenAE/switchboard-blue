Output AC-002-mismatched-svtnid-closed-before-admitnode.gif
Output AC-002-mismatched-svtnid-closed-before-admitnode.webm

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

Type "# AC-002: Mismatched ChallengeResponse svtn_id (admitted keyset) → connection closed BEFORE AdmitNode"
Enter
Type "# BC-2.01.009 PC-9 / EC-008 — guard fires; LookupInterface (0,false) proves AdmitNode was NOT reached"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/ -run 'TestNodeIdentifyHandshake_CRSVTNIDMismatch_ConnectionClosed_BeforeAdmitNode' -count=1 -v"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
