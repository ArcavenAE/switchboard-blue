Output AC-002-paths-list-passes-router-addr.gif
Output AC-002-paths-list-passes-router-addr.webm

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

Type "# AC-002: PathsList handler passes snap.RouterAddr to PathEntryFromSnapshot"
Enter
Sleep 500ms

Type "go test ./internal/metrics/... -run 'TestPathsList_PassesRouterAddr' -v"
Enter
Sleep 15s

Sleep 2s
