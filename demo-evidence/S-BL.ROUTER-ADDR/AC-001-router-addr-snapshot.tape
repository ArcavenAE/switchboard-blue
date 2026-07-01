Output AC-001-router-addr-snapshot.gif
Output AC-001-router-addr-snapshot.webm

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

Type "# AC-001: PathSnapshot.RouterAddr field propagation"
Enter
Sleep 500ms

Type "go test ./internal/paths/... -run 'TestBC_2_06_003_Snapshot_RouterAddr_Propagates|TestBC_2_06_003_Snapshot_RouterAddr_EmptyForAddrLess' -v"
Enter
Sleep 15s

Sleep 2s
