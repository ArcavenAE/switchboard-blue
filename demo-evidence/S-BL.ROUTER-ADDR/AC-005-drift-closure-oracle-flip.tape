Output AC-005-drift-closure-oracle-flip.gif
Output AC-005-drift-closure-oracle-flip.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-BL.ROUTER-ADDR"
Enter
Type "clear"
Enter
Show

Type "# AC-005: VP-047 oracle flip — router_addr non-empty for NewPathTrackerWithAddr paths"
Enter
Sleep 500ms

Type "go test ./internal/metrics/... -run 'TestVP047_RouterAddrNonEmpty' -v"
Enter
Sleep 15s

Sleep 2s
