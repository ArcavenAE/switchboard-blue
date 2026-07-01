# AC-005/AC-005a — overallQuality precedence: green/yellow/red/pending, empty-paths→pending
# Tests: TestDaemonRouterStatus_QualityStatusIndependence (rows b,c,d) + TestRouterStatus_EmptyPaths_QualityIsPending
# EC-007 (SampleCount<10 → quality=pending), EC-008 (empty-paths → quality=pending)
# BC-2.06.003 v1.14 PC-3, EC-007, EC-008

Output docs/demo-evidence/S-W5.04/AC-005-overall-quality-precedence.gif
Output docs/demo-evidence/S-W5.04/AC-005-overall-quality-precedence.webm

Set FontFamily "Menlo"
Set FontSize 13
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set Padding 20
Set TypingSpeed 60ms

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-W5.04 && clear"
Enter
Sleep 300ms
Show

Type "# AC-005/005a: overallQuality — pending dominates, empty-paths → pending (EC-007/EC-008)"
Enter
Sleep 300ms

Type "go test ./internal/metrics/... -v -run 'TestDaemonRouterStatus_QualityStatusIndependence|TestRouterStatus_EmptyPaths_QualityIsPending|TestDaemonRouterStatus_RedBand' 2>&1"
Enter
Wait+Screen@30s /PASS/
Sleep 2000ms
