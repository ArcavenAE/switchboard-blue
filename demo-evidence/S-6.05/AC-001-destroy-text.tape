# AC-001/AC-003 (happy path + not-found error path): sbctl admin svtn destroy — text mode
# BC-2.07.001 postcondition 3 (Destroy)
# Tests: TestSbctlAdmin_SVTNDestroy_HappyPath (happy path — prints "destroyed SVTN: <name>")
#        TestSbctlAdmin_SVTNDestroy_NotFound  (error path — E-SVTN-003 exit non-zero)

Output AC-001-destroy-text.gif
Output AC-001-destroy-text.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-6.05"
Enter
Sleep 500ms
Show

Type "# AC-001/AC-003: sbctl admin svtn destroy --name <name> --confirm <token>"
Enter
Sleep 300ms

Type "# Happy path: RPC dispatched, stdout contains 'destroyed SVTN: <name>'"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_HappyPath ./cmd/sbctl/..."
Enter
Sleep 8s

Type "# Error path: SVTN not found → E-SVTN-003, non-zero exit"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_SVTNDestroy_NotFound ./cmd/sbctl/..."
Enter
Sleep 8s
