# AC-004 — Fail-closed on corrupt PEM or non-Ed25519 key (E-KEY-001)
#
# Demonstrates:
#   1. A truncated/corrupt PEM file causes loadOrGenerateAdmissionKeypair to return
#      a non-nil error containing "PEM decode failed" and the file path.
#   2. A PKCS#8 PEM with an RSA key (not Ed25519) causes loadOrGenerateAdmissionKeypair
#      to return a non-nil error containing "not an Ed25519 key" and the file path.
#   3. In both cases the daemon refuses to start — no partial startup (AC-004 PC-3).
#
# BC anchors: BC-2.09.004 PC-6; E-KEY-001
#
# Test coverage:
#   TestAdmissionKeypair_FailClosed_CorruptPEM
#   TestAdmissionKeypair_FailClosed_NonEd25519Key
#   TestRunAccess_KeypairLoadFailure_MgmtGoroutineNotLeaked
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-004-fail-closed-corrupt-and-non-ed25519.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is checked in.

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-004-fail-closed-corrupt-and-non-ed25519.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-004-fail-closed-corrupt-and-non-ed25519.webm

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

# ── AC-004: fail-closed on corrupt PEM ──────────────────────────────────────────

Type `echo "=== AC-004: corrupt PEM → 'PEM decode failed'; RSA key → 'not an Ed25519 key'; daemon refuses to start ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestAdmissionKeypair_FailClosed" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── AC-004 PC-3: no partial startup — mgmt goroutine not leaked ──────────────────

Type `echo "=== AC-004 PC-3: on keypair-load failure, mgmt goroutine joined before runAccess returns ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestRunAccess_KeypairLoadFailure_MgmtGoroutineNotLeaked" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-004 PASS: fail-closed verified for corrupt PEM, RSA key, and no-partial-startup ==="`
Enter
Sleep 2s
