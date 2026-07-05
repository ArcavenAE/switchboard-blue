# S-5.02 AC-004/AC-006/AC-008: p99 pending + JSON envelope + daemon unreachable + quality sentinel
# BC-2.06.003 EC-003, postcondition 4, v1.7 postcondition 3
# Tests: TestSbctlPathsList_P99Pending_LessThan10Samples (AC-004)
#        TestSbctlMetrics_JSONEnvelope (AC-006 JSON mode)
#        TestSbctlMetrics_DaemonUnreachable (AC-006 error path)
#        TestSbctlRouterStatus_QualityPending_WhenP99Pending (AC-008)

Output AC-004-006-008-error-paths.gif
Output AC-004-006-008-error-paths.webm

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

Type "# S-5.02 error and edge paths: p99 pending, JSON, daemon unreachable"
Enter
Sleep 300ms

Type "# AC-004: p99 shows 'pending' when fewer than 10 samples"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlPathsList_P99Pending_LessThan10Samples ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-006: JSON envelope + daemon unreachable exits non-zero"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestSbctlMetrics_(JSONEnvelope|DaemonUnreachable)' ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-008: quality 'pending' when p99 is pending sentinel"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlRouterStatus_QualityPending_WhenP99Pending ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
