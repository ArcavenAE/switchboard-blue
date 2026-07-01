# AC-5 — Test coverage: go test ./internal/admission/... -run Lookup -race -v
# Demonstrates: all Lookup* tests pass under the race detector; full suite shown.
# Error path: go test with wrong -run filter returns "no tests to run" (empty filter).
# Traces: AC-5 (test coverage), AC-1, AC-2, AC-4.
#
# Usage: vhs AC-5-test-coverage-race.tape

Output docs/demo-evidence/S-BL.LOOKUP/AC-5-test-coverage-race.gif
Output docs/demo-evidence/S-BL.LOOKUP/AC-5-test-coverage-race.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 8
Set Padding 20
Set Framerate 30
Set TypingSpeed 30ms

Hide
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-BL.LOOKUP && clear"
Enter
Sleep 1s
Show

# Full Lookup test suite with race detector — the acceptance command from the story
Type "go test ./internal/admission/... -run Lookup -race -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|DATA RACE|---'"
Enter
Sleep 12s

# Error path: deliberate mismatch filter — confirms test framework works
Type "go test ./internal/admission/... -run LookupNonExistent -v 2>&1 | grep -E '^ok|FAIL|no test'"
Enter
Sleep 4s

Sleep 2s
