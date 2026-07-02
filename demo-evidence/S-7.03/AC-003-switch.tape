# AC-003: sbctl console switch — atomic detach+attach + error surfacing
# BC-2.08.001 postcondition 3 (PC-3)
# Tests: TestSbctlConsole_Switch/success_dispatches_console_switch_rpc  (happy path)
#        TestSbctlConsole_Switch/unknown_session_surfaces_E_SES_001      (E-SES-001 attach leg)
#        TestSbctlConsole_Switch/not_attached_surfaces_E_SES_004         (E-SES-004 detach leg)
#        TestSbctlConsole_Switch/auth_denied_surfaces_E_ADM_006          (E-ADM-006 error path)

Output AC-003-switch.gif
Output AC-003-switch.webm

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

Type "# AC-003: console switch dispatches console.switch RPC (BC-2.08.001 PC-3)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Switch/success_dispatches_console_switch_rpc ./cmd/sbctl/..."
Enter
Sleep 8s

Type "# error path: unknown session → E-SES-001 (attach leg)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Switch/unknown_session_surfaces_E_SES_001 ./cmd/sbctl/..."
Enter
Sleep 8s

Type "# error path: not attached → E-SES-004 (detach leg)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Switch/not_attached_surfaces_E_SES_004 ./cmd/sbctl/..."
Enter
Sleep 8s
