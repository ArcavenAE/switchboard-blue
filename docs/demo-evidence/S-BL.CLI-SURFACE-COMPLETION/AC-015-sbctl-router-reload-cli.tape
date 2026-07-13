# AC-015 — `sbctl router reload` CLI dispatch: happy path + sub-verb transition pin
#
# Demonstrates:
#   1. `sbctl router reload` dispatches router.reload via the existing
#      connectAndRun pattern (same dial+auth+dispatch shape `router metrics`
#      and `paths list` already use)
#   2. sbctl prints the {"accepted": true} response; exit code 0
#   3. Sub-verb transition pin — before this story, `reload` fell through to
#      the case arm's default usage-error arm; this AC proves the specific
#      transition (`sbctl router reload` now exits 0) while `sbctl router
#      bogus` (still-unknown) continues to exit 2 via the unchanged default arm
#
# BC anchors: BC-2.09.001 v1.2 PC-1 (RPC-trigger note) — same anchor as AC-011
#
# Test coverage:
#   TestRouterReload_CLIDispatch_HappyPath_AcceptedTrue                  (cmd/sbctl/router_control_test.go)
#   TestRouterReload_SubVerbTransition_KnownDispatchesUnknownStillExit2  (cmd/sbctl/router_control_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-015-sbctl-router-reload-cli.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-015-sbctl-router-reload-cli.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-015-sbctl-router-reload-cli.webm

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

# ── AC-015: sbctl router reload — happy path + sub-verb transition pin ────────

Type `echo "=== AC-015: sbctl router reload — accepted:true, reload dispatches while bogus still exits 2 ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestRouterReload_CLIDispatch_HappyPath_AcceptedTrue|TestRouterReload_SubVerbTransition_KnownDispatchesUnknownStillExit2' -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-015 PASS ==="`
Enter
Sleep 2s
