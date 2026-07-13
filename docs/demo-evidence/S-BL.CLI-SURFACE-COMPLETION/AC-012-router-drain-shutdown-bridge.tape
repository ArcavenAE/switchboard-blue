# AC-012 — `router.drain` bridges into the shipped shutdown sequence
#
# Demonstrates:
#   1. The router.drain handler sends on the new drainRequestCh (already-in-flight
#      drain -> no-op)
#   2. The select loop's third arm (case <-drainRequestCh: goto shutdown) reaches
#      the same shutdown: label as ctx.Done()/SIGTERM — same drain-broadcast,
#      per-node-flush, exit sequence
#   3. The RPC connection is expected to be severed as the daemon exits shortly
#      after — "connection reset" is an expected outcome, not a protocol error
#
# BC anchors: BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note)
#
# Test coverage:
#   TestRouterDrain_BridgesToShutdownSequence_ViaDrainRequestCh (cmd/switchboard/router_control_wire_test.go)
#   TestRouterDrain_ConnectionSeveredAfterAccepted_NotAnError    (cmd/switchboard/router_control_wire_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-012-router-drain-shutdown-bridge.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-012-router-drain-shutdown-bridge.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-012-router-drain-shutdown-bridge.webm

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

# ── AC-012: router.drain bridges into shutdown sequence via drainRequestCh ────

Type `echo "=== AC-012: router.drain bridges to shutdown sequence; conn-reset is expected, not an error ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterDrain_BridgesToShutdownSequence_ViaDrainRequestCh|TestRouterDrain_ConnectionSeveredAfterAccepted_NotAnError' -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-012 PASS ==="`
Enter
Sleep 2s
