# AC-006: Exhaustive args validation — E-CFG-001 for control chars, U+2028/U+2029,
#         whitespace-only, >255 bytes; cross-SVTN denial.
#
# Demonstrates: all exhaustive name-validation cases return E-CFG-001;
# CLI rejects missing --name before connecting.
#
# Usage: vhs AC-006-args-validation.tape
Output docs/demo-evidence/S-6.07/AC-006-args-validation.gif
Output docs/demo-evidence/S-6.07/AC-006-args-validation.webm

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
Type "cd /Users/skippy/work/switchboard-blue/.worktrees/S-6.07 && clear"
Enter
Sleep 1s
Show

# Error path: exhaustive args-validation table (nil, empty, whitespace, control chars, >255 bytes)
Type "go test ./cmd/switchboard/... -run 'TestAdminSVTNCreate_ArgsValidation_E_CFG_001_Exhaustive' -v 2>&1 | grep -E 'RUN|PASS|FAIL|E-CFG-001|^ok|---'"
Enter
Sleep 6s

# Error path: base args-validation unit test
Type "go test ./cmd/switchboard/... -run 'TestAdminSVTNCreate_ArgsValidation_E_CFG_001$' -v 2>&1 | grep -E 'RUN|PASS|FAIL|^ok|---'"
Enter
Sleep 5s
