# S-6.06 AC-001/AC-003: BuildAdminHandlers registers key.register / key.revoke / key.expire / list-keys
# BC-2.05.004 postconditions 1-3
# Tests: TestBuildAdminHandlers_KeyRegister_HappyPath (AC-001: register RPC)
#        TestBuildAdminHandlers_KeyRevoke_HappyPath (AC-002: revoke RPC)
#        TestBuildAdminHandlers_KeyExpire_HappyPath (AC-003: expire RPC)
#        TestBuildAdminHandlers_SixHandlers (handler count assertion)

Output AC-001-003-handler-registration.gif
Output AC-001-003-handler-registration.webm

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

Type "# S-6.06 AC-001/003: BuildAdminHandlers wires register/revoke/expire handlers"
Enter
Sleep 300ms

Type "# registers 6 admin handlers (was nil/TODO before this story)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestBuildAdminHandlers_(KeyRegister|KeyRevoke|KeyExpire|SixHandlers)' ./cmd/switchboard/..."
Enter
Sleep 6s
Sleep 2s

Type "# error paths: bootstrap key forbidden for revoke/expire (AC-004/AC-005)"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestBuildAdminHandlers_KeyRevoke_BootstrapKeyForbidden|TestBuildAdminHandlers_KeyExpire_BootstrapKeyForbidden' ./cmd/switchboard/..."
Enter
Sleep 6s
Sleep 2s
