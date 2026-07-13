# AC-003 — `paths ping` slow round trip is not an error; no quality classification
#
# Demonstrates:
#   1. A connection that succeeds but measures high latency is NOT an error —
#      rtt_ms simply reports a larger number; exit 0
#   2. paths.ping's response and sbctl's synthesized output never carry a
#      quality/status field (no green/yellow/red) — router.status remains the
#      exclusive owner of quality classification
#
# BC anchors: BC-2.06.004 PC-4, EC-003, Invariant 2
#
# Test coverage:
#   TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField (cmd/sbctl/paths_ping_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-003-paths-ping-slow-rtt-not-error.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-003-paths-ping-slow-rtt-not-error.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-003-paths-ping-slow-rtt-not-error.webm

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

# ── AC-003: slow RTT is not an error; no quality classification ───────────────

Type `echo "=== AC-003: slow round trip is not an error; no quality field ==="`
Enter
Sleep 300ms

Type `go test -race -run TestPathsPing_SlowRoundTrip_NotAnError_NoQualityField -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-003 PASS ==="`
Enter
Sleep 2s
