# S-W5.04 AC-006: VP-047 end-to-end field presence integration test
# BC-2.06.003 postcondition 1
# Tests: TestVP047_SbctlPathsList_EndToEnd (AC-006: all required fields present)
#        TestVP047_FieldSwapOracle (AC-006: field swap oracle)

Output AC-006-vp047-e2e.gif
Output AC-006-vp047-e2e.webm

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

Type "# S-W5.04 AC-006: VP-047 end-to-end field presence"
Enter
Sleep 300ms

Type "# all required fields present in paths.list response"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestVP047_SbctlPathsList_EndToEnd ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s

Type "# error path: field swap oracle detects missing required fields"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestVP047_FieldSwapOracle ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s
