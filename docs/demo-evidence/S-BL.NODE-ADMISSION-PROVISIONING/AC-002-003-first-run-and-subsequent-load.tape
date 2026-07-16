# AC-002 / AC-003 — First-run keypair generation + subsequent load
#
# Demonstrates:
#   AC-002: When key file is absent, loadOrGenerateAdmissionKeypair generates an Ed25519
#           keypair, writes it atomically to <path>.tmp → <path> with mode 0600 (PKCS#8 PEM),
#           creates parent dir if needed, and logs INFO "admission identity: generated new keypair at <path>".
#   AC-003: On subsequent start (file present), the same public key is loaded — stable across
#           restarts. The public key is available for discovery.Config.LocalNodeAdmissionPubkey.
#
# BC anchors: BC-2.09.004 PC-3a–3f (first run); BC-2.09.004 PC-5 (subsequent start)
#
# Test coverage:
#   TestAdmissionKeypair_FirstRun_FileAbsent_KeypairGeneratedAtomically
#   TestAdmissionKeypair_FirstRun_ParentDirAbsent_MkdirAll
#   TestAdmissionKeypair_FirstRun_GeneratedPKCS8PEMParseable
#   TestAdmissionKeypair_FirstRun_Mode0600
#   TestAdmissionKeypair_FirstRun_OrphanTmpDoesNotWedgeStartup
#   TestAdmissionKeypair_SubsequentStart_LoadedKeyMatchesGenerated
#   TestAdmissionKeypair_SubsequentStart_PublicKeyStableAcrossRestarts
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-002-003-first-run-and-subsequent-load.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is checked in.

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-002-003-first-run-and-subsequent-load.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-002-003-first-run-and-subsequent-load.webm

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

# ── AC-002: first-run keypair generation ────────────────────────────────────────

Type `echo "=== AC-002: first-run — atomic write, mode 0600, PKCS#8 PEM, parent dir created, INFO log ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestAdmissionKeypair_FirstRun" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── AC-003: subsequent start — public key stable ─────────────────────────────────

Type `echo "=== AC-003: subsequent start — loaded key matches generated; pubkey stable across restarts ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestAdmissionKeypair_SubsequentStart" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-002/AC-003 PASS: first-run generation + subsequent load verified ==="`
Enter
Sleep 2s
