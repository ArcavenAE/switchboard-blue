# S-6.02 AC-005: control-role revocation requires --confirm flag (defense-in-depth)
# BC-2.05.004 precondition 1 + ADR-004
# Tests: TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI (AC-005)

Output AC-005-control-revocation-confirm.gif
Output AC-005-control-revocation-confirm.webm

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

Type "# S-6.02 AC-005: error path — control key revoke requires --confirm"
Enter
Sleep 300ms

Type "# revocation of control-role key without --confirm → rejected with guidance"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_ControlRevocation_RequiresConfirm_CLI ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
