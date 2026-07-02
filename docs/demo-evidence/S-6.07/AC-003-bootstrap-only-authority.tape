# AC-003: Bootstrap-only authority — non-bootstrap key → E-ADM-009;
#         genesis carve-out; cross-SVTN control-key denied;
#         mutation test verifies RoleControl check fires independently.
#
# Usage: vhs AC-003-bootstrap-only-authority.tape
Output docs/demo-evidence/S-6.07/AC-003-bootstrap-only-authority.gif
Output docs/demo-evidence/S-6.07/AC-003-bootstrap-only-authority.webm

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

# Error path: non-bootstrap control key + non-control-role caller → E-ADM-009
Type "go test ./cmd/switchboard/... -run 'TestAdminSVTNCreate_NonBootstrapControlKey_RejectsWithEADM009|TestAdminSVTNCreate_NonControlCallerDenied|TestAdminSVTNCreate_BootstrapOnly_CrossSVTNKeyDenied' -v 2>&1 | grep -E 'RUN|PASS|FAIL|E-ADM-009|^ok|---'"
Enter
Sleep 6s

# Mutation test: RoleControl gate must fire independently of bootstrap-key gate
Type "go test ./cmd/switchboard/... -run 'TestAdminSVTNCreate_MutationTest_RoleControlCheckMustFireIndependently' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s
