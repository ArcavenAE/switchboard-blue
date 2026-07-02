# AC-001: sbctl console attach — JSON RPC dispatch + error surfacing
# BC-2.08.001 postcondition 1 (PC-1)
# Tests: TestSbctlConsole_Attach/success_dispatches_console_attach_rpc (happy path)
#        TestSbctlConsole_Attach/unknown_session_surfaces_E_SES_001   (E-SES-001 error path)
#        TestSbctlConsole_Attach/auth_denied_surfaces_E_ADM_006       (E-ADM-006 error path)

Output AC-001-attach.gif
Output AC-001-attach.webm

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

Type "# AC-001: console attach dispatches console.attach RPC (BC-2.08.001 PC-1)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Attach/success_dispatches_console_attach_rpc ./cmd/sbctl/..."
Enter
Sleep 8s

Type "# error path: unknown session → E-SES-001"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Attach/unknown_session_surfaces_E_SES_001 ./cmd/sbctl/..."
Enter
Sleep 8s

Type "# error path: auth denied → E-ADM-006 (E-RPC-011 envelope)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlConsole_Attach/auth_denied_surfaces_E_ADM_006 ./cmd/sbctl/..."
Enter
Sleep 8s
