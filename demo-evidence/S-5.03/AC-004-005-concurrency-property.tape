# S-5.03 AC-004/AC-005: concurrency safety + property test (VP-063)
# BC-2.02.003 invariant 2, postcondition 5
# Tests: TestBC_2_02_003_PathTracker_IsDegraded_Race (AC-004: race detector)
#        TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded (AC-004: snapshot consistency)
#        TestProp_IsDegraded_TracksEWMAThreshold (AC-005: property test)

Output AC-004-005-concurrency-property.gif
Output AC-004-005-concurrency-property.webm

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

Type "# S-5.03 AC-004/005: concurrency + property tests"
Enter
Sleep 300ms

Type "# AC-004: race-detector clean under concurrent IsDegraded reads/writes"
Enter
Sleep 300ms

Type "go test -race -count=1 -v -run 'TestBC_2_02_003_PathTracker_IsDegraded_Race|TestBC_2_02_003_Snapshot_DegradedMirrorsIsDegraded' ./internal/paths/..."
Enter
Sleep 6s
Sleep 3s

Type "# AC-005: property test — degraded tracks EWMA threshold across random inputs"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestProp_IsDegraded_TracksEWMAThreshold ./internal/paths/..."
Enter
Sleep 6s
Sleep 2s
