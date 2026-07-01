Output AC-003-new-path-tracker-with-addr.gif
Output AC-003-new-path-tracker-with-addr.webm

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

Type "# AC-003: NewPathTrackerWithAddr constructor + NewPathTracker backward compat"
Enter
Sleep 500ms

Type "go test ./internal/paths/... -run 'TestBC_2_06_003_NewPathTrackerWithAddr_StoresAddr|TestBC_2_06_003_NewPathTracker_Unchanged|TestBC_2_06_003_RouterAddr_ImmutableAfterConstruction' -v"
Enter
Sleep 15s

Sleep 2s
