# AC-004 — `paths.ping` RPC handler registration and authority
#
# Demonstrates:
#   1. mgmt.RegisterPingHandler is called from wireMetricsHandlers, making
#      paths.ping available on every daemon mode that already wires metrics
#      handlers: runRouter, runAccess, runConsole, runControl
#   2. paths.ping requires no additional Tier-2 authority beyond standard
#      Tier-1 operator-key authentication
#   3. The handler performs zero per-path metrics reads/writes — no
#      PathTracker interaction; request {} in, response {"pong": true} out
#      (VP-079)
#
# BC anchors: BC-2.06.004 Invariant 1, Trigger
#
# Test coverage:
#   TestPingHandler_EmptyArgsIn_PongOut_ZeroPathTrackerInteraction (internal/mgmt/register_ping_test.go)
#   TestWireMetricsHandlers_RegistersPingOnEveryMode                (cmd/switchboard/metrics_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-004-ping-handler-registration.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-004-ping-handler-registration.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-004-ping-handler-registration.webm

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

# ── Step 1: handler shape + zero PathTracker interaction ──────────────────────

Type `echo "=== AC-004: paths.ping handler — empty args in, pong out, zero PathTracker interaction ==="`
Enter
Sleep 300ms

Type `go test -race -run TestPingHandler_EmptyArgsIn_PongOut_ZeroPathTrackerInteraction -count=1 -v ./internal/mgmt/...`
Enter
Wait+Line /PASS/
Sleep 2s

# ── Step 2: registered on every daemon mode via wireMetricsHandlers ───────────

Type `echo "=== AC-004: RegisterPingHandler called from wireMetricsHandlers on every mode ==="`
Enter
Sleep 300ms

Type `go test -race -run TestWireMetricsHandlers_RegistersPingOnEveryMode -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-004 PASS ==="`
Enter
Sleep 2s
