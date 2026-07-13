# AC-016 — `sbctl router drain` CLI dispatch: happy path + sub-verb transition pin
#
# Demonstrates:
#   1. `sbctl router drain` dispatches router.drain via the existing
#      connectAndRun pattern (same dial+auth+dispatch shape `router metrics`
#      and `paths list` already use)
#   2. sbctl prints the {"accepted": true} response; exit code 0. Per
#      BC-2.09.002 PC-3's best-effort-delivery framing, a "connection reset"
#      observed instead of (or after) the response is an expected outcome,
#      not a protocol error — this AC's test tolerates either outcome
#   3. Sub-verb transition pin — `sbctl router drain` now dispatches (exit 0,
#      or tolerated conn-reset) while `sbctl router bogus` (still-unknown)
#      continues to exit 2 via the unchanged default arm
#
# BC anchors: BC-2.09.002 v1.3 Trigger/PC-1 (RPC-trigger note) — same anchor as AC-012
#
# Test coverage:
#   TestRouterDrain_CLIDispatch_HappyPath_AcceptedTrueOrConnReset       (cmd/sbctl/router_control_test.go)
#   TestRouterDrain_SubVerbTransition_KnownDispatchesUnknownStillExit2  (cmd/sbctl/router_control_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-016-sbctl-router-drain-cli.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-016-sbctl-router-drain-cli.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-016-sbctl-router-drain-cli.webm

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

# ── AC-016: sbctl router drain — happy path + sub-verb transition pin ─────────

Type `echo "=== AC-016: sbctl router drain — accepted:true (or tolerated conn-reset), drain dispatches while bogus still exits 2 ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterDrain_CLIDispatch_HappyPath_AcceptedTrueOrConnReset|TestRouterDrain_SubVerbTransition_KnownDispatchesUnknownStillExit2' -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-016 PASS ==="`
Enter
Sleep 2s
