# AC-008 — `sbctl svtn status` CLI dispatch: bare top-level, `--name` flag
#
# Demonstrates:
#   1. `sbctl svtn status --name=<svtn-name>` dispatches directly to admin.svtn.status —
#      NOT routed through `sbctl admin` framing (matches the `paths list`/`router status`
#      bare top-level read shape)
#   2. The flag is --name, not --id (SVTNManager is exclusively name-keyed)
#   3. Missing --name -> E-CFG-001 (client-side flag validation via usageErrf, exit 2)
#
# BC anchors: BC-2.07.001 PC-4 (CLI dispatch note)
#
# Test coverage:
#   TestSvtnStatus_CLIDispatch_BareTopLevel_NameFlag (cmd/sbctl/svtn_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-008-svtn-status-cli-dispatch.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-008-svtn-status-cli-dispatch.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-008-svtn-status-cli-dispatch.webm

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

# ── AC-008: sbctl svtn status — bare top-level dispatch, --name flag ──────────

Type `echo "=== AC-008: sbctl svtn status — bare top-level, --name flag, E-CFG-001 on missing flag ==="`
Enter
Sleep 300ms

Type `go test -race -run TestSvtnStatus_CLIDispatch_BareTopLevel_NameFlag -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-008 PASS ==="`
Enter
Sleep 2s
