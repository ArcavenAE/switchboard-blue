# AC-002 — admin.key.revoke handler e2e tests
# Demonstrates: role-mismatch -> E-ADM-019; control-without-confirm -> E-ADM-018;
# control-with-confirm -> success (key removed from AdmittedKeySet).
# Uses real mgmt.Server + real Unix socket (NOT a stub).
# Traces: BC-2.05.004 PC-2; HOLD-001 hybrid; ADR-004; AC-002.
#
# Usage: vhs AC-002-revoke-handler-e2e.tape

Output docs/demo-evidence/S-6.06/AC-002-revoke-handler-e2e.gif
Output docs/demo-evidence/S-6.06/AC-002-revoke-handler-e2e.webm

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
Type "clear"
Enter
Sleep 1s
Show

# Success path: control-to-control revoke WITH confirm succeeds
Type "go test -tags integration ./cmd/switchboard/... -run 'TestE2E_AdminRevoke_ControlWithConfirm' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s

# Error paths: role mismatch (E-ADM-019) and missing confirm (E-ADM-018)
Type "go test -tags integration ./cmd/switchboard/... -run 'TestE2E_AdminRevoke_RoleMismatch|TestE2E_AdminRevoke_ControlWithoutConfirm' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s
