# AC-005 — `admin.svtn.status` happy path
#
# Demonstrates:
#   1. `sbctl svtn status --name=mynet` returns
#      {"svtn_id":"<hex>","name":"mynet","created_at":"<RFC3339>",
#       "key_counts":{"control":1,"console":0,"access":2}}; exit 0
#   2. key_counts are grouped by role, scoped exclusively to the target SVTN (VP-048 row 1)
#
# BC anchors: BC-2.07.001 PC-4 (happy-path Canonical Test Vector)
#
# Test coverage:
#   TestAdminSVTNStatus_HappyPath_KeyCounts (cmd/switchboard/admin_handlers_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-005-svtn-status-happy-path.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-005-svtn-status-happy-path.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-005-svtn-status-happy-path.webm

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

# ── AC-005: admin.svtn.status happy path — key_counts by role ─────────────────

Type `echo "=== AC-005: admin.svtn.status happy path — svtn_id, name, created_at, key_counts ==="`
Enter
Sleep 300ms

Type `go test -race -run TestAdminSVTNStatus_HappyPath_KeyCounts -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-005 PASS ==="`
Enter
Sleep 2s
