# AC-004 — per-daemon-mode handler registration (ADR-004 role-exclusion)
# Demonstrates: control mode gets admin handlers (no E-RPC-010);
# access/non-control-role daemons return E-RPC-010 or E-ADM-009.
# Traces: BC-2.05.004 PC-1; ADR-004; AC-004.
#
# Happy path: control mode handles admin.key.register (ok or domain error, NOT E-RPC-010)
# Error path: access mode returns E-RPC-010 "unknown command"
#
# Usage: vhs AC-004-per-mode-handler-registration.tape

Output docs/demo-evidence/S-6.06/AC-004-per-mode-handler-registration.gif
Output docs/demo-evidence/S-6.06/AC-004-per-mode-handler-registration.webm

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
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-6.06 && clear"
Enter
Sleep 1s
Show

# Happy path: control mode registers admin handlers -> request dispatches (no E-RPC-010)
Type "go test -tags integration ./cmd/switchboard/... -run 'TestControlMode_AdminHandlersRegistered' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s

# Error path: access mode -> E-RPC-010 "unknown command"; non-control role also rejected
Type "go test -tags integration ./cmd/switchboard/... -run 'TestAccessMode_AdminHandlersNotRegistered|TestAC004_NonControlRoleRejected' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s
