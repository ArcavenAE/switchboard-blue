# AC-001 — `paths ping` happy path: dial, authenticate, measure RTT
#
# Demonstrates:
#   1. `sbctl paths ping --router=<addr>` dials the target directly, overriding --target
#   2. Tier-1 authentication (no additional Tier-2 gate)
#   3. paths.ping issued with empty args {}; daemon returns {"pong": true}
#   4. sbctl reports {"router": "<addr>", "rtt_ms": <float64>}, exit 0 — both default
#      (bare object) and --json (envelope-wrapped) output shapes
#
# BC anchors: BC-2.06.004 PC-1, Invariant 1
#
# Test coverage:
#   TestPathsPing_HappyPath_ReportsRTT (cmd/sbctl/paths_ping_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-001-paths-ping-happy-path.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-001-paths-ping-happy-path.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-001-paths-ping-happy-path.webm

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

# ── AC-001: paths ping happy path — RTT report, default + --json ──────────────

Type `echo "=== AC-001: paths ping happy path — dial, authenticate, measure RTT ==="`
Enter
Sleep 300ms

Type `go test -race -run TestPathsPing_HappyPath_ReportsRTT -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-001 PASS ==="`
Enter
Sleep 2s
