# AC-002: Enumerate sessions without hostnames; >= 2 distinct advertisers
# BC-2.03.002 postcondition 1, postcondition 3, invariant 1
# Test: TestDiscovery_Enumerate_NoHostnameRequired

Output AC-002-enumerate-no-hostname.gif
Output AC-002-enumerate-no-hostname.webm

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

Type "# AC-002: enumerate without hostnames; oracle: len(distinctNodeAddrs) >= 2"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Enumerate_NoHostnameRequired ./internal/discovery/..."
Enter
Sleep 8s

Type "# error path: empty result when no advertisements received"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Enumerate_EmptyWithoutAdvertisements ./internal/discovery/..."
Enter
Sleep 8s
