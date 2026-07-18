Output AC-007-router-startup-load.gif
Output AC-007-router-startup-load.webm

Set Shell "bash"
Set FontSize 14
Set FontFamily "Menlo"
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-BL.ADMISSION-SYNC-WIRE"
Enter
Type "clear"
Enter
Show

Type "# AC-007: router startup file-absent→empty+INFO; valid→load; corrupt→fail-closed E-KEY-002"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestRouterStartup_AdmissionStateFile' -v -count=1"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
