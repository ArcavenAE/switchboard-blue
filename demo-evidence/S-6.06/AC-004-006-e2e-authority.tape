# S-6.06 AC-004/AC-006: control-mode access gate + end-to-end authority enforcement
# BC-2.05.004 Precondition 1 / DI-001
# Tests: TestControlMode_AdminHandlersRegistered (AC-004: control mode wires handlers)
#        TestAccessMode_AdminHandlersNotRegistered (AC-004 negative: access mode has no admin handlers)
#        TestE2E_AdminRegister_HappyPath (AC-006: end-to-end register via mgmt stream)
#        TestE2E_AdminKeyRegister_RoleInsufficient (AC-006 negative: wrong role denied)

Output AC-004-006-e2e-authority.gif
Output AC-004-006-e2e-authority.webm

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

Type "# S-6.06 AC-004/006: control-mode gating + e2e authority enforcement"
Enter
Sleep 300ms

Type "# AC-004: control mode registers admin handlers; access mode does NOT"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestControlMode_AdminHandlersRegistered|TestAccessMode_AdminHandlersNotRegistered' ./cmd/switchboard/..."
Enter
Sleep 6s
Sleep 2s

Type "# AC-006: end-to-end register succeeds for correct role, fails for wrong role"
Enter
Sleep 300ms

Type "go test -count=1 -v -run 'TestE2E_AdminRegister_HappyPath|TestE2E_AdminKeyRegister_RoleInsufficient' ./cmd/switchboard/..."
Enter
Sleep 6s
Sleep 2s
