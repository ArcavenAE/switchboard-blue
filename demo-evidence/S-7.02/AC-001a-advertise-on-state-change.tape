# AC-001a: Advertise on state-change trigger
# BC-2.03.001 postcondition 3 — state-change trigger
# Test: TestDiscovery_Advertise_OnStateChange

Output AC-001a-advertise-on-state-change.gif
Output AC-001a-advertise-on-state-change.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-7.02"
Enter
Sleep 500ms
Show

Type "# AC-001a: state-change triggers presence advertisement within 1 tick"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Advertise_OnStateChange ./internal/discovery/..."
Enter
Sleep 8s

Type "# error path: detach event also triggers advertisement (EC-001)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Advertise_OnStateChange_DetachTriggersAdvert ./internal/discovery/..."
Enter
Sleep 8s
