# S-6.03 AC-001/AC-003/AC-004: Ed25519 key loading, auth failure, connection refused
# BC-2.07.002 PC-2, BC-2.07.003
# Tests: TestSbctl_KeyLoading_Ed25519 (AC-001: key load via ParseRawPrivateKey)
#        TestSbctl_AuthFailure_ExitsOneWithEADM010 (AC-003? key rejected → E-ADM-010)
#        TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030 (AC-004: dial fail → E-NET-001)

Output AC-001-004-key-loading-connection.gif
Output AC-001-004-key-loading-connection.webm

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

Type "# S-6.03: Ed25519 key loading + connection error codes"
Enter
Sleep 300ms

Type "# AC-001: --key loads Ed25519 via ssh.ParseRawPrivateKey"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctl_KeyLoading_Ed25519 ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-004 error: connection refused → E-NET-001, exit 1"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctl_ConnectionRefused_ExitsOneWithENET001_VP030 ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 1s

Type "# AC-003 error: auth failure → E-ADM-010, exit 1"
Enter
Sleep 300ms

Type "go test -count=1 -v -run TestSbctl_AuthFailure_ExitsOneWithEADM010 ./cmd/sbctl/..."
Enter
Sleep 6s
Sleep 2s
