# AC-003 — FrameTypePEConnect constant, Valid(), bootstrap-flip, discrimination
#
# Demonstrates:
#   1. frame.FrameTypePEConnect (0x06) constant defined and Valid() returns true
#   2. Bootstrap frame type flipped from placeholder to FrameTypePEConnect
#   3. Receive loop discards FrameTypePEConnect frames (does not forward to callback)
#   4. Control frames (non-PEConnect) are forwarded to callback
#
# BC anchors: BC-2.09.001 (PEConnect discrimination), frame.FrameType validity
#
# Test coverage:
#   TestFrameType_Valid_PEConnect                            (internal/frame/frame_type_test.go)
#   TestConnector_BootstrapFrameTypePEConnect                (internal/upstreamdial/connector_test.go)
#   TestConnector_ReceiveLoop_PEConnectFrameDiscarded        (internal/upstreamdial/connector_test.go)
#   TestConnector_ReceiveLoop_CtlFrameForwardedToCallback    (internal/upstreamdial/connector_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-003-peconnect-discrimination.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-003-peconnect-discrimination.gif
Output docs/demo-evidence/S-BL.PE-RECEIVE-LOOP/AC-003-peconnect-discrimination.webm

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

# ── Step 1: FrameTypePEConnect constant and Valid() ───────────────────────────

Type `echo "=== AC-003: FrameTypePEConnect constant (0x06) + Valid() ==="`
Enter
Sleep 300ms

Type `go test ./internal/frame/ -run TestFrameType_Valid_PEConnect -count=1 -v`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 2: bootstrap-flip + discard + forward discrimination ─────────────────

Type `echo "=== AC-003: bootstrap-flip, PEConnect discard, ctl-frame forward ==="`
Enter
Sleep 300ms

Type `go test ./internal/upstreamdial/ -run 'TestConnector_BootstrapFrameTypePEConnect|TestConnector_ReceiveLoop_PEConnectFrameDiscarded|TestConnector_ReceiveLoop_CtlFrameForwardedToCallback' -count=1 -v`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-003 PASS ==="`
Enter
Sleep 2s
