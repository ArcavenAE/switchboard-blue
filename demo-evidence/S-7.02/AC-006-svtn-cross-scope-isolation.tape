# AC-006: SVTN cross-scope isolation — SVTN-A Enumerate returns zero SVTN-B sessions
# BC-2.03.002 invariant 1
# Tests: TestDiscovery_Enumerate_SVTNIsolation (main oracle: len(sessionsFromSVTNB)==0)
#        TestDiscovery_Enumerate_SVTNIsolation_ErrSentinel (ErrSVTNMismatch sentinel)
#        TestDiscovery_VP045_SVTNIsolation_MultipleScopes (VP-045 e2e: multiple scopes)

Output AC-006-svtn-cross-scope-isolation.gif
Output AC-006-svtn-cross-scope-isolation.webm

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

Type "# AC-006: SVTN-A Enumerate returns 0 sessions from SVTN-B"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_Enumerate_SVTNIsolation ./internal/discovery/..."
Enter
Sleep 8s

Type "# VP-045 e2e: isolation holds across multiple independent SVTN scopes"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestDiscovery_VP045_SVTNIsolation_MultipleScopes ./internal/discovery/..."
Enter
Sleep 10s
