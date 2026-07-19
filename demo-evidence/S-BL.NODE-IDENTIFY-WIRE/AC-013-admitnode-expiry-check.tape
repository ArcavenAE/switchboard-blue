Output AC-013-admitnode-expiry-check.gif
Output AC-013-admitnode-expiry-check.webm

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

Type "# AC-013: AdmitNode expiry check (O-1 ruling §15): ErrKeyExpired for past-expiry key; future/no expiry succeed (BC-2.05.001 PC-6 / Invariant 5)"
Enter
Sleep 500ms

Type "go test ./internal/admission/... -run 'TestAdmitNode_ExpiredKey|TestAdmitNode_FutureExpiry|TestAdmitNode_NoExpiry' -v -count=1 -timeout 30s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
