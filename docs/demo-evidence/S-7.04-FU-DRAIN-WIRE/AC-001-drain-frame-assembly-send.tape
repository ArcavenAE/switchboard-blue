# AC-001 — DRAIN frame assembled and sent to every connected PE node on drainCoord.Signal
#
# Demonstrates:
#   1. The single startup drain observer (registered once at runRouter setup) iterates
#      the live per-node send map at Signal time.
#   2. For each registered node it assembles a FrameTypeCtl (0x03) frame with the 4-byte
#      DRAIN payload (control_type=0x01, version=0x01) via frame.EncodeOuterHeader.
#   3. The frame is non-blocking sent to the node's send channel; the writer goroutine
#      flushes it onto the wire; the simulated node reads it back within 2s.
#   4. drainCoord.Wait returns nil when the observer completes.
#
# BC anchors: BC-2.09.002 PC-1, BC-2.01.008 PC-2; placement note Q1, Q-SEAM, Q-SINGLE-OBS
#
# Test coverage:
#   TestDrainObserver_AssemblesAndSendsDRAINFrame (cmd/switchboard/router_drain_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-001-drain-frame-assembly-send.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-001-drain-frame-assembly-send.gif
Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-001-drain-frame-assembly-send.webm

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

# ── AC-001: DRAIN frame assembly + per-node send ──────────────────────────────

Type `echo "=== AC-001: DRAIN frame assembled and sent to every connected PE node ==="`
Enter
Sleep 300ms

Type `go test -race -run TestDrainObserver_AssemblesAndSendsDRAINFrame -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-001 PASS ==="`
Enter
Sleep 2s
