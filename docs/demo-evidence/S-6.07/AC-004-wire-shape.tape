# AC-004: adminSVTNCreateResult wire shape —
#         svtn_id (hex string), bootstrap_fingerprint (SHA256:<base64> verbatim).
#
# Demonstrates: JSON field names match BC-2.07.001 PC-2 contract;
# BootstrapFingerprint() value is carried verbatim (not re-encoded).
#
# Usage: vhs AC-004-wire-shape.tape
Output docs/demo-evidence/S-6.07/AC-004-wire-shape.gif
Output docs/demo-evidence/S-6.07/AC-004-wire-shape.webm

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

# Success path: wire-shape field names + format assertions
Type "go test ./cmd/switchboard/... -run 'TestAdminSVTNCreateResult_JSONFieldNames|TestAdminSVTNCreate_ControlCallerSucceeds' -v 2>&1 | grep -E 'RUN|PASS|FAIL|svtn_id|bootstrap_fingerprint|SHA256|^ok|---'"
Enter
Sleep 6s
