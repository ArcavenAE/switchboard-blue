# S-5.01 AC-001/AC-002: quality indicator threshold classification + hysteresis
# BC-2.06.001 postconditions 2-4, invariant 3
# Tests: TestQualityIndicator_ThresholdClassification (AC-001: green/yellow/red thresholds)
#        TestQualityIndicator_HysteresisUpgrade (AC-002: hysteresis on upgrade)
#        TestQualityIndicator_DegradationNeverSkipsLevel (AC-002: monotone degradation)

Output AC-001-002-quality-indicator-thresholds.gif
Output AC-001-002-quality-indicator-thresholds.webm

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

Type "# S-5.01 AC-001/002: quality indicator thresholds + hysteresis"
Enter
Sleep 300ms

Type "# AC-001: green/yellow/red threshold classification"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestQualityIndicator_ThresholdClassification ./internal/metrics/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-002: hysteresis — 6 consecutive good readings required for upgrade"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestQualityIndicator_(HysteresisUpgrade|DegradationNeverSkipsLevel|RecoveryNeverSkipsLevel)' ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s
