# AC-002 — `paths ping` error paths: unreachable and auth failure
#
# Demonstrates:
#   1. Target daemon unreachable before connection -> E-NET-001 "daemon unreachable: <address>"; exit 1
#   2. Connection succeeds but Tier-1 auth fails -> E-ADM-010; exit 1. No paths.ping RPC
#      is dispatched (auth failure occurs before command dispatch)
#
# BC anchors: BC-2.06.004 PC-2, PC-3, EC-001, EC-002
#
# Test coverage:
#   TestPathsPing_Unreachable_ENET001 (cmd/sbctl/paths_ping_test.go)
#   TestPathsPing_AuthFailure_EADM010 (cmd/sbctl/paths_ping_test.go)
#
# Usage: vhs docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-002-paths-ping-error-paths.tape
# Per POL-004: rendered artifacts (.gif/.webm) are gitignored; only this .tape script is
# checked in. All paths are repo-relative (no hardcoded worktree paths).

Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-002-paths-ping-error-paths.gif
Output docs/demo-evidence/S-BL.CLI-SURFACE-COMPLETION/AC-002-paths-ping-error-paths.webm

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

# ── AC-002: paths ping error paths — unreachable + auth failure ───────────────

Type `echo "=== AC-002: paths ping error paths — E-NET-001 unreachable, E-ADM-010 auth failure ==="`
Enter
Sleep 300ms

Type `go test -race -run 'TestPathsPing_Unreachable_ENET001|TestPathsPing_AuthFailure_EADM010' -count=1 -v ./cmd/sbctl/...`
Enter
Wait+Line /PASS/
Sleep 2s

Type `echo "=== AC-002 PASS ==="`
Enter
Sleep 2s
