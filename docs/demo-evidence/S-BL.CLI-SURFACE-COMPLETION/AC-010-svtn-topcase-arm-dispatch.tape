# AC-010 — `sbctl svtn` top-level case arm dispatch
#
# Demonstrates:
#   1. cmd/sbctl/main.go gains a new top-level `case "svtn":` (alongside sessions,
#      paths, router, console, admin) dispatching to a new runSvtn function
#   2. runSvtn routes status -> AC-005..AC-008 dispatch, destroy -> AC-009 shim
#   3. An unknown sub-verb under svtn returns a usage error, exit 2 (same shape
#      as the existing paths/router case arms' default arms)
#
# BC anchors: none (Scope item 1 — CLI dispatch structure)
#
# Test coverage:
#   TestSvtn_UnknownSubVerb_UsageErrorExit2 (cmd/sbctl/svtn_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-010-svtn-topcase-arm-dispatch.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-010-svtn-topcase-arm-dispatch.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-010-svtn-topcase-arm-dispatch.webm

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

# ── AC-010: sbctl svtn top-level case arm — unknown sub-verb exit 2 ───────────

Type `echo "=== AC-010: sbctl svtn top-level case arm — unknown sub-verb usage error, exit 2 ==="`
Enter
Sleep 300ms

Type `go test -race -run TestSvtn_UnknownSubVerb_UsageErrorExit2 -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-010 PASS ==="`
Enter
Sleep 2s
