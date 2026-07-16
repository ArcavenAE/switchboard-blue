# AC-001 — Config.Validate() admission_key_file E-CFG-014 validation
#
# Demonstrates:
#   1. A config with whitespace-only admission_key_file ("   ") is rejected with E-CFG-014.
#   2. A config with a valid (non-whitespace) path is accepted.
#   3. A config with absent/empty admission_key_file is accepted.
#   4. Config.Validate() performs no file I/O — a non-existent path is accepted.
#
# BC anchors: BC-2.09.003 v2.1 PC-12; BC-2.09.004 PC-1/PC-2; E-CFG-014
#
# Test coverage:
#   TestConfig_Validate_AdmissionKeyFile_AbsentAccepted
#   TestConfig_Validate_AdmissionKeyFile_ValidPathAccepted
#   TestConfig_Validate_AdmissionKeyFile_WhitespaceOnlyRejectsE_CFG_014
#   TestConfig_Validate_AdmissionKeyFile_NoIOPerformed
#   TestConfig_Validate_AdmissionKeyFile_ExhaustiveErrorCollection
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-001-config-validation-e-cfg-014.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-001-config-validation-e-cfg-014.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-001-config-validation-e-cfg-014.webm

Require go

Set Shell "zsh"
Set FontFamily "Menlo"
Set FontSize 14
Set Width 1200
Set Height 720
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

# ── AC-001: Config.Validate() admission_key_file whitespace-only → E-CFG-014 ────

Type `echo "=== AC-001: Config.Validate() admission_key_file — whitespace-only rejected (E-CFG-014), valid/absent accepted ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestConfig_Validate_AdmissionKeyFile" -count=1 -v ./internal/config/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-001 PASS: E-CFG-014 fires for whitespace-only; absent and valid paths accepted; no I/O; exhaustive collection ==="`
Enter
Sleep 2s
