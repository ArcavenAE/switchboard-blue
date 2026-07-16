# AC-007 — discovery.Config.LocalNodeAdmissionPubkey wired from loaded/generated keypair
#
# Demonstrates:
#   1. The 32-byte raw Ed25519 public key derived from the admission keypair is non-nil
#      and has exactly 32 bytes (ed25519.PublicKeySize).
#   2. disc.Run with a pre-cancelled context returns context.Canceled — not
#      ErrMissingNodeAdmissionPubkey — confirming LocalNodeAdmissionPubkey was non-empty
#      when discovery.New was called.
#   3. Through-runAccess wiring: the newDiscovery seam captures the discovery.Config
#      passed by runAccess and asserts LocalNodeAdmissionPubkey byte-equals the
#      known keypair's public key (adversary F-6 through-runAccess regression guard).
#
# BC anchors: BC-2.09.004 PC-3e; BC-2.04.008 Precondition 3; rulings Decision 5
#
# Test coverage:
#   TestAccessDaemon_LocalNodeAdmissionPubkey_PopulatedFrom_LoadedKeypair
#   TestAccessDaemon_LocalNodeAdmissionPubkey_NonNilLength32
#   TestRunAccess_WiresLocalNodeAdmissionPubkey_FromLoadedKeypair
#
# Usage: vhs docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-007-local-node-admission-pubkey-wired.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is checked in.

Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-007-local-node-admission-pubkey-wired.gif
Output docs/demo-evidence/S-BL.NODE-ADMISSION-PROVISIONING/AC-007-local-node-admission-pubkey-wired.webm

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

# ── AC-007: LocalNodeAdmissionPubkey wired and non-nil/32-byte ───────────────────

Type `echo "=== AC-007: discovery.Config.LocalNodeAdmissionPubkey populated from keypair — non-nil, 32 bytes, no ErrMissingNodeAdmissionPubkey ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestAccessDaemon_LocalNodeAdmissionPubkey" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

# ── AC-007 through-runAccess coverage (adversary F-6) ────────────────────────────

Type `echo "=== AC-007 adversary F-6: through-runAccess LocalNodeAdmissionPubkey wiring verified via newDiscovery seam ==="`
Enter
Sleep 300ms

Type `go test -race -run "TestRunAccess_WiresLocalNodeAdmissionPubkey_FromLoadedKeypair" -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-007 PASS: LocalNodeAdmissionPubkey wired correctly through runAccess ==="`
Enter
Sleep 2s
