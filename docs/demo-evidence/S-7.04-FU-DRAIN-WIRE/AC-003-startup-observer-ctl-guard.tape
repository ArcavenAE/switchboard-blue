# AC-003 — Single startup observer registered; observer fires; FO-DRAIN-WIRE-001
# forward-compat guard; ctl-short-payload + unknown-control_type pins
#
# Demonstrates:
#   1. The single observer is registered at drainCoord-construction time and fires
#      drainObserverFiredHook on Signal, independent of any live node connection.
#   2. A FrameTypeCtl frame with payload_len<4 is silently discarded (E-PRT-002 log,
#      no connection close) — Q-CTL-GUARD/EC-002.
#   3. A FrameTypeCtl frame with an unrecognized control_type is silently ignored with
#      NO logging and no connection close — BC-2.01.008 PC-4/Invariant 2/FO-DRAIN-WIRE-001.
#
# BC anchors: BC-2.09.002 PC-1; BC-2.01.008 PC-4 + Invariant 2; placement note
# Q-SINGLE-OBS, Q-AC003, Q-CTL-GUARD
#
# Test coverage:
#   TestDrainObserver_RegisteredAtStartup_FiresOnSignal     (cmd/switchboard/router_drain_wire_test.go)
#   TestRouter_CtlFrame_ShortPayload_NoConnClose            (cmd/switchboard/router_drain_wire_test.go)
#   TestRouter_CtlFrame_UnknownControlType_SilentIgnore     (cmd/switchboard/router_drain_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-003-startup-observer-ctl-guard.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-003-startup-observer-ctl-guard.gif
Output docs/demo-evidence/S-7.04-FU-DRAIN-WIRE/AC-003-startup-observer-ctl-guard.webm

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

# ── AC-003: startup observer + ctl guard pins ─────────────────────────────────

Type `echo "=== AC-003: startup observer fires; short-payload + unknown-control_type ctl guards ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestDrainObserver_RegisteredAtStartup_FiresOnSignal|TestRouter_CtlFrame_ShortPayload_NoConnClose|TestRouter_CtlFrame_UnknownControlType_SilentIgnore' -count=1 -v ./cmd/switchboard/`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-003 PASS ==="`
Enter
Sleep 2s
