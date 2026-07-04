# AC-005: Duplicate name → E-SVTN-001 stutter-free; non-duplicate failure → E-INT-001
#
# Demonstrates: duplicate SVTN name uses name from call args (not err.Error() wrap),
# producing "SVTN already exists: <name>" with no stutter;
# non-duplicate Create() failure maps to E-INT-001.
#
# Usage: vhs AC-005-duplicate-name-stutter-free.tape
Output docs/demo-evidence/S-6.07/AC-005-duplicate-name-stutter-free.gif
Output docs/demo-evidence/S-6.07/AC-005-duplicate-name-stutter-free.webm

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

# Error path: duplicate-name stutter-free + vacuity control
Type "go test ./cmd/switchboard/... -run 'TestAdminSVTNCreate_DuplicateName_E_SVTN_001|TestAdminSVTNCreate_NoStutterInDuplicateMessage|TestAdminSVTNCreate_DuplicateName_E_SVTN_001_VacuityControl' -v 2>&1 | grep -E 'RUN|PASS|FAIL|E-SVTN-001|stutter|^ok|---'"
Enter
Sleep 6s

# Error path: non-duplicate Create() failure → E-INT-001 (not E-RPC-011)
Type "go test ./cmd/switchboard/... -run 'TestAdminSVTNCreate_CryptoRandFailure_E_INT_001' -v 2>&1 | grep -E 'RUN|PASS|FAIL|E-INT-001|^ok|---'"
Enter
Sleep 5s
