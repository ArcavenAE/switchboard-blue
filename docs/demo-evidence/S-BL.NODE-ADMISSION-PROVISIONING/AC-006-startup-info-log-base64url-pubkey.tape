# AC-006 — Startup INFO log with base64url pubkey on every start
#
# Demonstrates:
#   1. On first run (file absent), after keypair generation, stderr contains the INFO log:
#      "access: admission identity pubkey (register with admin.key.register): <base64url>"
#      where <base64url> is the raw-URL-no-padding encoding of the 32-byte Ed25519 public key.
#   2. On subsequent start (file present), the same INFO log is emitted — unconditionally,
#      so the operator can recover the pubkey from logs after any restart.
#   3. The base64url value decodes to the exact 32-byte public key from the loaded keypair.
#
# BC anchors: BC-2.09.004 PC-7; rulings Decision 4 — log emitted unconditionally
#
# Test coverage:
#   TestAdmissionKeypair_StartupInfoLog_FirstRun_ContainsBase64UrlPubkey
#   TestAdmissionKeypair_StartupInfoLog_SubsequentStart_ContainsBase64UrlPubkey
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-006-startup-info-log-base64url-pubkey.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is checked in.

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-006-startup-info-log-base64url-pubkey.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-006-startup-info-log-base64url-pubkey.webm

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

# ── AC-006: base64url pubkey INFO log on every start ─────────────────────────────

Type `echo "=== AC-006: INFO log 'access: admission identity pubkey (register with admin.key.register): <base64url>' on first-run AND subsequent start ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestAdmissionKeypair_StartupInfoLog" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-006 PASS: base64url pubkey INFO log confirmed on first-run and subsequent start ==="`
Enter
Sleep 2s
