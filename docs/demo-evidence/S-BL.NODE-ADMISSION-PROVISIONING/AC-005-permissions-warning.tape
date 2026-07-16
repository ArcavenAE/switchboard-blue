# AC-005 — File permissions > 0600 → WARNING logged; daemon starts
#
# Demonstrates:
#   1. A key file with mode 0644 causes a permissions WARNING in stderr containing
#      the octal mode ("0644") and the word "permissions".
#   2. A key file with mode 0444 (world-readable) also triggers the WARNING —
#      the predicate is `perm&0o077 != 0` (any group or other bit), not `perm > 0o600`.
#      This was the F-1 fix from adversary pass-1: 0444 < 0600 numerically, so the
#      old `perm > 0o600` predicate silently skipped world-readable private keys.
#   3. A key file with exactly 0600 does NOT emit a permissions warning.
#   4. In all warning cases the daemon continues to start (advisory-not-fatal, OpenSSH posture).
#
# BC anchors: BC-2.09.004 PC-4; rulings §1.4 (adversary pass-1 F-1)
#
# Test coverage:
#   TestAdmissionKeypair_PermissionsWarning_BroaderThan0600
#     sub-tests: 0644_owner_rw_group_r_other_r_warns, 0444_world_readable_warns, 0440_group_readable_warns
#   TestAdmissionKeypair_NoPermissionsWarning_Exactly0600
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-005-permissions-warning.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is checked in.

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-005-permissions-warning.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-005-permissions-warning.webm

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

# ── AC-005: permissions warning for 0644, 0444, 0440; no warning for 0600 ───────

Type `echo "=== AC-005: key file perms > 0600 (any group/other bit) → WARNING logged; 0600 → no warning; daemon starts in both cases ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestAdmissionKeypair_PermissionsWarning_BroaderThan0600|TestAdmissionKeypair_NoPermissionsWarning_Exactly0600" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-005 PASS: permissions warning (0644/0444/0440 warns, 0600 silent); F-1 fix verified ==="`
Enter
Sleep 2s
