# AC-007 — `admin.svtn.status` purity boundary and mode exclusion
#
# Demonstrates:
#   1. The response schema (svtn_id, name, created_at, key_counts) never carries
#      session or health-indicator fields — internal/session remains a forbidden
#      import for cmd/switchboard/admin_handlers.go
#   2. admin.svtn.status is registered in BuildAdminHandlers, control-mode-daemon-only
#      (needs *svtnmgmt.SVTNManager). Router, access, and console modes pass nil
#      admin handlers and correctly return E-RPC-010 (unknown command)
#
# BC anchors: BC-2.07.001 PC-4 (ARCH-09 purity note); ADR-004
#
# Test coverage:
#   TestAdminSVTNStatus_ResponseExcludesSessionHealthFields  (cmd/switchboard/admin_handlers_test.go)
#   TestAdminSVTNStatus_NonControlMode_NilAdminHandlers_ERPC010 (cmd/switchboard/admin_handlers_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-007-svtn-status-purity-mode-exclusion.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-007-svtn-status-purity-mode-exclusion.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-007-svtn-status-purity-mode-exclusion.webm

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

# ── AC-007: purity boundary — response schema + mode exclusion ────────────────

Type `echo "=== AC-007: admin.svtn.status purity boundary + non-control-mode E-RPC-010 ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestAdminSVTNStatus_ResponseExcludesSessionHealthFields|TestAdminSVTNStatus_NonControlMode_NilAdminHandlers_ERPC010' -count=1 -v ./cmd/switchboard/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-007 PASS ==="`
Enter
Sleep 2s
