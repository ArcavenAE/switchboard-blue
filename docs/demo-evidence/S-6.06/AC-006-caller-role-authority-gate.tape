# AC-006 — caller-key-role enforcement at handler layer (VP-075; DI-001)
# Demonstrates: non-control-role caller -> E-ADM-009 for mutating ops;
# any-role caller can invoke admin.key.list-keys (F-L2-003).
# Caller identity resolved server-side from authenticated handshake pubkey.
# Traces: BC-2.05.004 Precondition 1; DI-001; ADR-004; VP-075; AC-006.
#
# Usage: vhs AC-006-caller-role-authority-gate.tape

Output docs/demo-evidence/S-6.06/AC-006-caller-role-authority-gate.gif
Output docs/demo-evidence/S-6.06/AC-006-caller-role-authority-gate.webm

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

# Error path: console-role / access-role caller on mutating ops -> E-ADM-009
Type "go test -tags integration ./cmd/switchboard/... -run 'TestE2E_AdminKeyRegister_RoleInsufficient|TestE2E_AdminKeyRevoke_RoleInsufficient' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s

# Happy path: any-role (console) caller on list-keys -> ok:true, NOT E-ADM-009 (F-L2-003)
Type "go test -tags integration ./cmd/switchboard/... -run 'TestE2E_AdminListKeys_AnyRole' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s
