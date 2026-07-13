# AC-009 — `sbctl svtn destroy` top-level migration shim
#
# Demonstrates:
#   1. `sbctl svtn destroy` (any arguments) returns a usage error (exit 2) with the
#      exact redirect text: "svtn destroy: use 'sbctl admin svtn destroy
#      --name=<svtn-name> [--confirm=<svtn-short-id>|--yes]'"
#   2. No --id/--name flag parsing occurs — the shim never inspects either flag
#   3. No RPC is dispatched; admin.svtn.destroy is never called from this code path
#   4. runDestroyConfirmGate is never invoked from this shim — confirm gate stays
#      exclusively owned by `sbctl admin svtn destroy`
#
# BC anchors: none (Decision 3 — CLI-surface documentation only, not a BC anchor point)
#
# Test coverage:
#   TestSvtnDestroy_TopLevelShim_UsageErrorRedirect_Exit2 (cmd/sbctl/svtn_test.go)
#   TestSvtnDestroy_TopLevelShim_NoRPCDispatch            (cmd/sbctl/svtn_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-009-svtn-destroy-migration-shim.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-009-svtn-destroy-migration-shim.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-009-svtn-destroy-migration-shim.webm

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

# ── AC-009: sbctl svtn destroy — top-level migration shim, no RPC ─────────────

Type `echo "=== AC-009: sbctl svtn destroy top-level — usage-error redirect, no RPC dispatch ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestSvtnDestroy_TopLevelShim_UsageErrorRedirect_Exit2|TestSvtnDestroy_TopLevelShim_NoRPCDispatch' -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-009 PASS ==="`
Enter
Sleep 2s
