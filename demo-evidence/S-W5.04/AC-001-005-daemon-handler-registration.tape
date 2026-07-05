# S-W5.04 AC-001/AC-004/AC-005: daemon paths.list / router.metrics / router.status handlers
# BC-2.06.003 postconditions 1-3
# Tests: TestDaemonPathsList_HandlerRegistered (AC-001: paths.list registered)
#        TestDaemonRouterMetrics_HandlerRegistered (AC-004: router.metrics registered)
#        TestDaemonRouterStatus_HandlerRegistered (AC-005: router.status registered)
#        TestMetricsWire_PathsListRegistered (wire-level handler registration)

Output AC-001-005-daemon-handler-registration.gif
Output AC-001-005-daemon-handler-registration.webm

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

Type "# S-W5.04: daemon-side paths.list / router.metrics / router.status handler registration"
Enter
Sleep 300ms

Type "# AC-001/004/005: all three handlers registered and route correctly"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestDaemonPathsList_HandlerRegistered|TestDaemonRouterMetrics_HandlerRegistered|TestDaemonRouterStatus_HandlerRegistered' ./internal/metrics/..."
Enter
Sleep 6s
Sleep 2s

Type "# wire level: paths.list registered in daemon handler table"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestMetricsWire_PathsListRegistered ./cmd/switchboard/..."
Enter
Sleep 6s
Sleep 2s
