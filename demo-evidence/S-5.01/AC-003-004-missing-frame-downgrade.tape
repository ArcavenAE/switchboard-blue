# S-5.01 AC-003/AC-004: missing-frame downgrade + property-based invariants
# BC-2.06.002 postcondition 2, invariant 3
# Tests: TestQualityIndicator_MissingFrameDowngradeGreenToYellow (AC-003: downgrade on missing frames)
#        TestProp_BC_2_06_001_NoSkipTransitionDuringDegradation (AC-004: property test)

Output AC-003-004-missing-frame-downgrade.gif
Output AC-003-004-missing-frame-downgrade.webm

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

Type "# S-5.01 AC-003/004: missing-frame triggers downgrade; property tests"
Enter
Sleep 300ms

Type "# AC-003: missing frames cause quality downgrade (green→yellow→red)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestQualityIndicator_MissingFrameDowngrade' ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s

Type "# AC-004: property tests — no level skipping during degradation or recovery"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestProp_BC_2_06_001' ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s
