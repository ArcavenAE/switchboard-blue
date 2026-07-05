# S-5.03 AC-001/AC-002/AC-003: IsDegraded flag on PathTracker
# BC-2.02.003 postconditions 5-6
# Tests: TestBC_2_02_003_PathTracker_IsDegraded (AC-001: degraded on threshold cross)
#        TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation (AC-003: reactivation clears flag)

Output AC-001-003-degraded-path-flag.gif
Output AC-001-003-degraded-path-flag.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue"
Enter
Sleep 500ms
Show

Type "# S-5.03: IsDegraded flag on PathTracker"
Enter
Sleep 300ms

Type "# AC-001: path marked degraded when EWMA crosses threshold"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestBC_2_02_003_PathTracker_IsDegraded ./internal/paths/..."
Enter
Sleep 6s
Sleep 2s

Type "# AC-002 recovery: clear degraded on EWMA recovery"
Enter
Sleep 300ms

Type "# AC-003: reactivation also clears the degraded flag"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestBC_2_02_003_PathTracker_IsDegraded_OnReactivation ./internal/paths/..."
Enter
Sleep 6s
Sleep 2s
