# AC-003: Presence advertisement payload has required fields
# BC-2.03.003 postcondition 1: session_name, attachment_status, quality_indicator
# Test: TestDiscovery_Advertisement_RequiredFields

Output AC-003-advertisement-required-fields.gif
Output AC-003-advertisement-required-fields.webm

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

Type "# AC-003: advertisement carries session_name, attachment_status, quality_indicator"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Advertisement_RequiredFields ./internal/discovery/..."
Enter
Sleep 8s

Type "# EC-002: quality_indicator may be absent at startup (startup state)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Advertisement_QualityUnknownOnStartup ./internal/discovery/..."
Enter
Sleep 8s
