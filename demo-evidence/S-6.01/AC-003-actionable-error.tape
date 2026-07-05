# S-6.01 AC-003: actionable startup error — names field + suggests correction
# BC-2.05.001 postcondition 3 — VP-029 error message quality
# Tests: TestRouterStartup_ExitsWithActionableError (AC-003: actionable message)
#        TestVP029_ErrorMessageNamesFieldAndSuggestion (VP-029 verification)

Output AC-003-actionable-error.gif
Output AC-003-actionable-error.webm

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

Type "# S-6.01 AC-003: validation error names the field and suggests a fix"
Enter
Sleep 300ms

Type "# VP-029: error message quality — operator sees actionable text, not raw code"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestRouterStartup_ExitsWithActionableError ./internal/config/..."
Enter
Sleep 6s
Sleep 1s

Type "go test -count=1 -v -run TestVP029_ErrorMessageNamesFieldAndSuggestion ./internal/config/..."
Enter
Sleep 6s
Sleep 2s
