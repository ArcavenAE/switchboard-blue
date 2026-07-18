Output AC-013-push-full-snapshot-multi-endpoint-sequencing.gif
Output AC-013-push-full-snapshot-multi-endpoint-sequencing.webm

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

Type "# AC-013: PushFullSnapshot multi-endpoint per-endpoint sequencing"
Enter
Type "#  — each reachable endpoint independently reaches correct terminal state"
Enter
Sleep 500ms

Type "go test ./cmd/switchboard/... -run 'TestAdmissionSync_PushFullSnapshot_MultiEndpoint' -v -count=1 -timeout 120s"
Enter
Wait+Line /^(ok|FAIL)/
Sleep 2s
