# AC-002: sbctl console detach — JSON RPC dispatch + error surfacing
# BC-2.08.001 postcondition 2 (PC-2)
# Tests: TestSbctlConsole_Detach/success_dispatches_console_detach_rpc (happy path)
#        TestSbctlConsole_Detach/not_attached_surfaces_E_SES_004       (E-SES-004 error path)

Output AC-002-detach.gif
Output AC-002-detach.webm

Require go

Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20
Set PlaybackSpeed 1

Hide
Type "cd /Users/skippy/work/aae-orc/run/switchboard-blue/.worktrees/S-7.03"
Enter
Sleep 500ms
Show

Type "# AC-002: console detach dispatches console.detach RPC (BC-2.08.001 PC-2)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Detach/success_dispatches_console_detach_rpc ./cmd/sbctl/..."
Enter
Sleep 8s

Type "# error path: detach when not attached → E-SES-004"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Detach/not_attached_surfaces_E_SES_004 ./cmd/sbctl/..."
Enter
Sleep 8s
