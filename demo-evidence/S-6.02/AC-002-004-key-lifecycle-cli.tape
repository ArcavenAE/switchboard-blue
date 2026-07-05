# S-6.02 AC-002/AC-003/AC-004: sbctl admin key register / revoke / expire CLI
# BC-2.05.004 postconditions 1-3
# Tests: TestSbctlAdmin_KeyRegister_CLI (AC-002: register sends RPC)
#        TestSbctlAdmin_KeyRevoke_CLI (AC-003: revoke sends RPC)
#        TestSbctlAdmin_KeyExpire_CLI (AC-004: expire sends RPC with TTL)

Output AC-002-004-key-lifecycle-cli.gif
Output AC-002-004-key-lifecycle-cli.webm

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

Type "# S-6.02: sbctl admin key register / revoke / expire CLI dispatch"
Enter
Sleep 300ms

Type "# AC-002: sbctl admin key register --name --pub-key --role"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_KeyRegister_CLI ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-003: sbctl admin key revoke --name --confirm"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_KeyRevoke_CLI ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-004: sbctl admin key expire --name --after <duration>"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctlAdmin_KeyExpire_CLI ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
