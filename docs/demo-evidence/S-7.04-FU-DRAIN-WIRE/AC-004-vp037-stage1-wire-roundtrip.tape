# AC-004 — VP-037 stage-1 discharge: TestE2E_RouterDrain_WireRoundTrip, untagged,
# drainCoord.Wait nil within window; verification_lock stays false
#
# Demonstrates:
#   1. A real runRouter is started; a simulated node dials cfg.ListenAddr and is
#      admitted via the nodeConnHook accept/register barrier.
#   2. cancel() drives the production shutdown path -> drainCoord.Signal.
#   3. The simulated node deterministically reads FrameTypeCtl + payload[0]=0x01 (DRAIN)
#      within 2s (Shutdown ordering guarantee forces the flush before teardown).
#   4. drainCoord.Wait returns nil within the drain window. VP-037 stage-1 (wire
#      round-trip) is discharged; verification_lock stays false (stage-2 is a
#      named follow-on).
#
# BC anchors: VP-037 stage-1; BC-2.09.002 PC-4; BC-2.01.008 PC-2; placement note Q4-AMENDED
#
# Test coverage:
#   TestE2E_RouterDrain_WireRoundTrip (cmd/switchboard/router_drain_wire_test.go, untagged)
#
# Usage: vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-004-vp037-stage1-wire-roundtrip.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-004-vp037-stage1-wire-roundtrip.gif
Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-004-vp037-stage1-wire-roundtrip.webm

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

# ── AC-004: VP-037 stage-1 wire round-trip ────────────────────────────────────

Type `echo "=== AC-004: VP-037 stage-1 — DRAIN wire round-trip, untagged e2e ==="`
Enter
Sleep 300ms

Type `go test -race -run TestE2E_RouterDrain_WireRoundTrip -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-004 PASS ==="`
Enter
Sleep 2s
