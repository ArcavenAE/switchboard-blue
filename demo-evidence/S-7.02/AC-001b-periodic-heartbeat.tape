# AC-001b: Periodic 30s heartbeat — independent of state-change advertisements
# BC-2.03.001 postcondition 4 — periodic heartbeat
# Tests: TestDiscovery_Advertise_PeriodicHeartbeat_ExactN (primary oracle, RULING-W6TB-G)
#        TestDiscovery_Advertise_PeriodicHeartbeat_IsIndependent
#        TestDiscovery_HeartbeatCount_MonotonicallyIncreases

Output AC-001b-periodic-heartbeat.gif
Output AC-001b-periodic-heartbeat.webm

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

Type "# AC-001b: exact-N tick oracle (RULING-W6TB-G — deterministic, no wall-clock)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Advertise_PeriodicHeartbeat_ExactN ./internal/discovery/..."
Enter
Sleep 8s

Type "# HeartbeatCount atomic counter — unconditional production observability"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_HeartbeatCount_MonotonicallyIncreases ./internal/discovery/..."
Enter
Sleep 8s
