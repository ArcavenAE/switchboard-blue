Output AC-010-lww-rebind-stale-cleanup-guard.gif
Output AC-010-lww-rebind-stale-cleanup-guard.webm

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

Type "# AC-010: LWW rebind on reconnect; stale cleanup guard protects new binding (BC-2.01.010 PC-2/PC-9)"
Enter
Sleep 500ms

Type "go test ./internal/routing/... -run 'TestBindInterface_LWW|TestBindInterface_StaleCleanupGuard|TestBindInterface_CleanDisconnect' -v -count=1 -timeout 30s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
