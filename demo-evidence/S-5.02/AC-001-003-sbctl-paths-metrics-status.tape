# S-5.02 AC-001/AC-002/AC-003: sbctl paths list / router metrics / router status (alias)
# BC-2.06.003 postconditions 1-3
# Tests: TestSbctlPathsList_OutputsCanonicalFields (AC-001)
#        TestSbctlRouterMetrics_OutputsSVTNMetrics (AC-002)
#        TestSbctlRouterStatus_IsAliasForPathsList (AC-003: alias check)

Output AC-001-003-sbctl-paths-metrics-status.gif
Output AC-001-003-sbctl-paths-metrics-status.webm

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

Type "# S-5.02: sbctl paths list / router metrics / router status"
Enter
Sleep 300ms

Type "# AC-001: paths list outputs canonical fields (score, paths, latency)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlPathsList_OutputsCanonicalFields ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s

Type "# AC-002: router metrics outputs SVTN metrics object"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlRouterMetrics_OutputsSVTNMetrics ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-003: router status is alias for paths list"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlRouterStatus_IsAliasForPathsList ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
